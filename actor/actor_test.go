package actor

import (
	"strings"
	"testing"

	"github.com/haru0017/diexodos/dex"
)

type counter struct{ N int }
type ping struct{ From string }
type pong struct{}

func pingPong(rounds int) *System {
	s := NewSystem()
	Spawn(s, "a", counter{})
	Spawn(s, "b", counter{})
	On(s, "b", func(m counter, msg ping, ctx *Ctx) counter {
		ctx.Send(msg.From, pong{})
		m.N++
		return m
	})
	On(s, "a", func(m counter, msg pong, ctx *Ctx) counter {
		m.N++
		if m.N < rounds {
			ctx.Send("b", ping{From: "a"})
		}
		return m
	})
	Post(s, "b", ping{From: "a"})
	return s
}

func TestPingPong(t *testing.T) {
	s := pingPong(3)
	s.Inv("bounded", func(w Snapshot) bool { return Machine[counter](w, "a").N <= 3 })
	s.EventuallyAlways("settles", func(w Snapshot) bool {
		return w.Quiet() && Machine[counter](w, "a").N == 3
	})
	Check(t, s)
}

func TestLossBreaksDelivery(t *testing.T) {
	s := pingPong(3)
	s.EventuallyAlways("settles", func(w Snapshot) bool {
		return w.Quiet() && Machine[counter](w, "a").N == 3
	})
	res := Run(s)
	if res.Lasso != nil || res.Violation != nil {
		t.Fatalf("reliable channel should settle: %+v", res)
	}

	lossy := NewSystem(Loss())
	Spawn(lossy, "a", counter{})
	Spawn(lossy, "b", counter{})
	On(lossy, "b", func(m counter, msg ping, ctx *Ctx) counter {
		ctx.Send(msg.From, pong{})
		m.N++
		return m
	})
	On(lossy, "a", func(m counter, msg pong, ctx *Ctx) counter { m.N++; return m })
	Post(lossy, "b", ping{From: "a"})
	lossy.EventuallyAlways("settles", func(w Snapshot) bool {
		return w.Quiet() && Machine[counter](w, "a").N == 1
	})
	res = Run(lossy)
	if res.Lasso == nil {
		t.Fatal("expected losing the message to break the liveness property")
	}
}

type inbox struct {
	GotA    bool
	Ordered bool
}
type msgA struct{}
type msgB struct{}

func orderSystem(opts ...SystemOption) *System {
	s := NewSystem(opts...)
	Spawn(s, "m", inbox{Ordered: true})
	On(s, "m", func(m inbox, _ msgA, _ *Ctx) inbox { m.GotA = true; return m })
	On(s, "m", func(m inbox, _ msgB, _ *Ctx) inbox {
		if !m.GotA {
			m.Ordered = false
		}
		return m
	})
	Post(s, "m", msgA{})
	Post(s, "m", msgB{})
	s.Inv("in order", func(w Snapshot) bool { return Machine[inbox](w, "m").Ordered })
	return s
}

func TestReorderExploresOutOfOrderDelivery(t *testing.T) {
	Check(t, orderSystem())

	res := Run(orderSystem(Reorder()))
	if res.Violation == nil {
		t.Fatal("expected reordering to break the ordering invariant")
	}
}

func TestUnfairDeliveryAllowsStarvation(t *testing.T) {
	// The b mailbox can be starved forever only when delivery is unfair.
	build := func(opts ...SystemOption) *System {
		s := NewSystem(opts...)
		Spawn(s, "spinner", counter{})
		Spawn(s, "b", counter{})
		On(s, "spinner", func(m counter, msg ping, ctx *Ctx) counter {
			ctx.Send("spinner", ping{From: "spinner"})
			return m
		})
		On(s, "b", func(m counter, _ pong, _ *Ctx) counter { m.N++; return m })
		Post(s, "spinner", ping{From: "spinner"})
		Post(s, "b", pong{})
		s.EventuallyAlways("b ran", func(w Snapshot) bool { return Machine[counter](w, "b").N == 1 })
		return s
	}

	if res := Run(build()); res.Lasso != nil {
		t.Fatalf("fair delivery must run b eventually: %+v", res.Lasso)
	}
	if res := Run(build(UnfairDelivery())); res.Lasso == nil {
		t.Fatal("expected a starvation lasso under unfair delivery")
	}
}

func TestUnhandledMessagePanics(t *testing.T) {
	s := NewSystem()
	Spawn(s, "m", counter{})
	Post(s, "m", ping{From: "m"})
	defer func() {
		r := recover()
		if r == nil || !strings.Contains(r.(string), "no handler") {
			t.Fatalf("got %v", r)
		}
	}()
	Run(s)
}

func TestRejectsPointerState(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a state containing a slice")
		}
	}()
	type bad struct{ Q []int }
	s := NewSystem()
	Spawn(s, "m", bad{})
}

func TestMaxStatesOptionPassesThrough(t *testing.T) {
	s := pingPong(1000000)
	res := Run(s, dex.MaxStates(50))
	if !res.Truncated {
		t.Fatalf("got %+v", res)
	}
}
