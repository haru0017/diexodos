package actor

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/haru0017/diexodos/dex"
)

// Spec compiles the system into a dex spec over snapshots. Delivery of the
// pending messages of each machine is one action, weakly fair unless
// UnfairDelivery is set. Message loss, when enabled, is a separate action
// that is never fair.
func (s *System) Spec() dex.Spec[Snapshot] {
	init := Snapshot{
		names:    s.names,
		machines: make(map[string]any, len(s.initial)),
		queues:   make(map[string][]any),
	}
	for name, m := range s.initial {
		init.machines[name] = m
	}
	for _, p := range s.posts {
		init.queues[p.to] = appended(init.queues[p.to], p.msg)
	}

	var actions []dex.Action[Snapshot]
	for _, name := range s.names {
		name := name
		deliver := dex.ActN("deliver("+name+")", nil, func(w Snapshot) []Snapshot {
			return s.deliver(w, name)
		})
		if !s.unfair {
			deliver = dex.Fair(deliver)
		}
		actions = append(actions, deliver)
		if s.loss {
			actions = append(actions, dex.ActN("lose("+name+")", nil, func(w Snapshot) []Snapshot {
				return s.lose(w, name)
			}))
		}
	}

	var invs []dex.Invariant[Snapshot]
	for _, inv := range s.invs {
		invs = append(invs, dex.Inv(inv.name, inv.holds))
	}
	var live []dex.Liveness[Snapshot]
	for _, l := range s.liveness {
		if l.eventuallyAlways {
			live = append(live, dex.EventuallyAlways(l.name, l.holds))
		} else {
			live = append(live, dex.AlwaysEventually(l.name, l.holds))
		}
	}

	return dex.Spec[Snapshot]{
		Init:       []Snapshot{init},
		Actions:    actions,
		Invariants: invs,
		Liveness:   live,
	}
}

func (s *System) candidates(q []any) []int {
	if len(q) == 0 {
		return nil
	}
	if !s.reorder {
		return []int{0}
	}
	idx := make([]int, len(q))
	for i := range q {
		idx[i] = i
	}
	return idx
}

// deliverable returns the queue positions delivery may pick: the first
// non-deferred message under FIFO, every non-deferred message under Reorder.
func (s *System) deliverable(name string, m any, q []any) []int {
	var idx []int
	for i, msg := range q {
		if s.isDeferred(name, m, msg) {
			continue
		}
		idx = append(idx, i)
		if !s.reorder {
			break
		}
	}
	return idx
}

func (s *System) isDeferred(name string, m, msg any) bool {
	mt := reflect.TypeOf(msg)
	for _, d := range s.defers[name] {
		if d.msgType == mt && d.while(m) {
			return true
		}
	}
	return false
}

func (s *System) deliver(w Snapshot, name string) []Snapshot {
	var out []Snapshot
	for _, idx := range s.deliverable(name, w.machines[name], w.queues[name]) {
		msg := w.queues[name][idx]
		mt := reflect.TypeOf(msg)
		matched := false
		for _, h := range s.handlers[name] {
			if h.msgType != mt {
				continue
			}
			matched = true
			nw := w.clone()
			nw.queues[name] = without(nw.queues[name], idx)
			ctx := &Ctx{sys: s, self: name}
			nw.machines[name] = h.fn(w.machines[name], msg, ctx)
			for _, p := range ctx.sends {
				nw.queues[p.to] = appended(nw.queues[p.to], p.msg)
			}
			out = append(out, nw)
		}
		if !matched {
			panic(fmt.Sprintf("actor: machine %q has no handler for %T", name, msg))
		}
	}
	return out
}

func (s *System) lose(w Snapshot, name string) []Snapshot {
	var out []Snapshot
	for _, idx := range s.candidates(w.queues[name]) {
		nw := w.clone()
		nw.queues[name] = without(nw.queues[name], idx)
		out = append(out, nw)
	}
	return out
}

// Run explores the system without failing any test.
func Run(s *System, opts ...dex.Option) dex.Result[Snapshot] {
	return dex.RunKeyed(s.Spec(), Snapshot.key, opts...)
}

// Check explores the system and fails t on any violation.
func Check(t testing.TB, s *System, opts ...dex.Option) dex.Result[Snapshot] {
	t.Helper()
	return dex.CheckKeyed(t, s.Spec(), Snapshot.key, opts...)
}
