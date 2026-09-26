// Package actor is a message-passing layer on top of dex. A system is a set
// of named machines with typed message handlers. The layer compiles machines,
// mailboxes, and delivery into guarded actions and explores every delivery
// interleaving.
//
// Machine states and messages must be plain values: booleans, numbers,
// strings, and arrays or structs of those. Pointers, slices, and maps are
// rejected at registration time so that state identity stays exact.
package actor

import (
	"fmt"
	"reflect"
)

type handler struct {
	msgType reflect.Type
	fn      func(m, msg any, ctx *Ctx) any
}

type post struct {
	to  string
	msg any
}

// System is a buildable actor model. Configure it with Spawn, On, and Post,
// then explore it with Run or Check.
type System struct {
	names    []string
	initial  map[string]any
	handlers map[string][]handler
	posts    []post
	loss     bool
	reorder  bool
	unfair   bool

	invs     []namedPred
	liveness []livenessProp
}

type namedPred struct {
	name  string
	holds func(Snapshot) bool
}

type livenessProp struct {
	namedPred
	eventuallyAlways bool
}

type SystemOption func(*System)

// Loss lets any queued message vanish, modeling an unreliable channel.
// Losing a message is never a fair action.
func Loss() SystemOption {
	return func(s *System) { s.loss = true }
}

// Reorder delivers any queued message instead of the head, modeling channels
// without ordering.
func Reorder() SystemOption {
	return func(s *System) { s.reorder = true }
}

// UnfairDelivery drops the weak fairness of delivery actions. Without it a
// behavior that starves a nonempty mailbox forever is not a valid liveness
// counterexample.
func UnfairDelivery() SystemOption {
	return func(s *System) { s.unfair = true }
}

func NewSystem(opts ...SystemOption) *System {
	s := &System{
		initial:  make(map[string]any),
		handlers: make(map[string][]handler),
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Spawn adds a machine with its initial state.
func Spawn[M any](s *System, name string, initial M) {
	if _, ok := s.initial[name]; ok {
		panic(fmt.Sprintf("actor: machine %q already spawned", name))
	}
	mustBePlain(reflect.TypeOf(initial))
	s.names = append(s.names, name)
	s.initial[name] = initial
}

// On registers a handler for messages of type Msg delivered to the machine.
// Several handlers for the same message type fork the exploration, one
// successor per handler.
func On[M, Msg any](s *System, name string, fn func(M, Msg, *Ctx) M) {
	if _, ok := s.initial[name]; !ok {
		panic(fmt.Sprintf("actor: machine %q not spawned", name))
	}
	var msg Msg
	mt := reflect.TypeOf(msg)
	mustBePlain(mt)
	s.handlers[name] = append(s.handlers[name], handler{
		msgType: mt,
		fn: func(m, msg any, ctx *Ctx) any {
			return fn(m.(M), msg.(Msg), ctx)
		},
	})
}

// Post queues a message before exploration starts.
func Post[Msg any](s *System, to string, msg Msg) {
	if _, ok := s.initial[to]; !ok {
		panic(fmt.Sprintf("actor: machine %q not spawned", to))
	}
	mustBePlain(reflect.TypeOf(msg))
	s.posts = append(s.posts, post{to: to, msg: msg})
}

// Inv adds an invariant over snapshots of the whole system.
func (s *System) Inv(name string, holds func(Snapshot) bool) {
	s.invs = append(s.invs, namedPred{name, holds})
}

// EventuallyAlways adds a liveness property: on every fair behavior the
// predicate eventually holds forever.
func (s *System) EventuallyAlways(name string, holds func(Snapshot) bool) {
	s.liveness = append(s.liveness, livenessProp{namedPred{name, holds}, true})
}

// AlwaysEventually adds a liveness property: on every fair behavior the
// predicate holds infinitely often.
func (s *System) AlwaysEventually(name string, holds func(Snapshot) bool) {
	s.liveness = append(s.liveness, livenessProp{namedPred{name, holds}, false})
}

// Ctx carries the effects of one handler execution.
type Ctx struct {
	sys   *System
	self  string
	sends []post
}

// Self is the name of the machine handling the message.
func (c *Ctx) Self() string { return c.self }

// Send queues a message to another machine. Delivery is a separate later
// step, interleaved with everything else.
func (c *Ctx) Send(to string, msg any) {
	if _, ok := c.sys.initial[to]; !ok {
		panic(fmt.Sprintf("actor: machine %q not spawned", to))
	}
	mustBePlain(reflect.TypeOf(msg))
	c.sends = append(c.sends, post{to: to, msg: msg})
}
