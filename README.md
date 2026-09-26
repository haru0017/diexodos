# diexodos

An explicit-state model checker embedded in Go.

Describe a system as a state type, a set of guarded actions, and properties. The checker exhaustively walks every reachable state across every interleaving. When an invariant breaks it reports a shortest counterexample trace. When a liveness property breaks it reports a fair lasso: a path into a cycle that can repeat forever under the declared fairness. Specs are plain Go and run under `go test`.

The name is the Greek word διέξοδος, a way through. Its adverb διεξοδικά means exhaustively. The import name is `dex`.

```sh
go get github.com/haru0017/diexodos
```

## Quick start

Two processes race for one slot. Checking the slot and taking it are separate steps, so both can observe a free slot and both take it:

```go
import "github.com/haru0017/diexodos/dex"

type State struct {
	Saw   [3]bool
	Holds [3]bool
}

func Spec() dex.Spec[State] {
	free := func(s State) bool { return !s.Holds[1] && !s.Holds[2] }
	var actions []dex.Action[State]
	for i := 1; i <= 2; i++ {
		actions = append(actions,
			dex.Act(fmt.Sprintf("check(%d)", i),
				func(s State) bool { return !s.Saw[i] && free(s) },
				func(s State) State { s.Saw[i] = true; return s }),
			dex.Act(fmt.Sprintf("take(%d)", i),
				func(s State) bool { return s.Saw[i] },
				func(s State) State { s.Holds[i] = true; return s }))
	}
	return dex.Spec[State]{
		Init:    []State{{}},
		Actions: actions,
		Invariants: []dex.Invariant[State]{
			dex.Inv("mutual exclusion", func(s State) bool { return !(s.Holds[1] && s.Holds[2]) }),
		},
	}
}

func TestMutex(t *testing.T) {
	dex.Check(t, Spec()) // fails with a counterexample trace
}
```

The checker finds the shortest interleaving that breaks the invariant:

```
invariant "mutual exclusion" violated:
   0. (init) {Saw:[false false false] Holds:[false false false]}
   1. check(1): {Saw:[false true false] Holds:[false false false]}
   2. check(2): {Saw:[false true true] Holds:[false false false]}
   3. take(1): {Saw:[false true true] Holds:[false true false]}
   4. take(2): {Saw:[false true true] Holds:[false true true]}
```

`dex.Check` fails the test on any violation, a deadlock when `dex.DetectDeadlocks()` is set, or a truncated search. `dex.Run` returns the result instead, for tests where a violation is the expected outcome. `dex.MaxStates(n)` bounds the search and `dex.Progress(w)` reports progress periodically.

## Liveness and fairness

An invariant says a bad state is never reached. A liveness property says a good thing eventually happens, and it only means something together with fairness: without it, the scheduler that starves an action forever is always a counterexample.

```go
spec := dex.Spec[int]{
	Init: []int{0},
	Actions: []dex.Action[int]{
		dex.Act("spin", func(s int) bool { return s == 0 }, func(int) int { return 0 }),
		dex.Fair(dex.Act("finish", func(s int) bool { return s == 0 }, func(int) int { return 1 })),
	},
	Liveness: []dex.Liveness[int]{
		dex.EventuallyAlways("done", func(s int) bool { return s == 1 }),
	},
}
dex.Check(t, spec) // passes; without Fair the spin cycle would be a lasso
```

- `dex.Fair` declares weak fairness: an action that stays enabled forever must eventually run.
- `dex.StronglyFair` declares strong fairness: an action enabled infinitely often must run infinitely often, even if it keeps being disabled in between.
- `dex.EventuallyAlways(name, p)` states that p eventually holds forever. `dex.AlwaysEventually(name, p)` states that p holds infinitely often.

Terminal states stutter forever, so a system that halts in a bad state is caught too.

## States with slices or maps

State identity for comparable states is Go equality. States holding slices go through `dex.RunKeyed` with a canonical key, which `dexgen` writes for you:

```go
//go:generate go run github.com/haru0017/diexodos/cmd/dexgen -type State

type State struct {
	Round int
	Log   []Entry
}

res := dex.RunKeyed(spec, State.DexKey)
```

Fields that cannot be encoded canonically, such as pointers and maps, are rejected when the code is generated.

## Actor layer

The `actor` package builds message-passing systems on the same engine: named machines, typed handlers, mailboxes. Delivery order across machines is explored exhaustively, and the channel semantics is an explicit modeling decision:

- delivery is weakly fair by default; `actor.UnfairDelivery()` turns that off
- `actor.Loss()` lets any message vanish, and losing is never fair
- `actor.Reorder()` delivers any queued message instead of the head
- `actor.Defer` keeps a message queued while a predicate holds on the machine, like the defer keyword in P

```go
import "github.com/haru0017/diexodos/actor"

s := actor.NewSystem()
actor.Spawn(s, "slot", Slot{})
actor.On(s, "slot", func(m Slot, msg TryTake, ctx *actor.Ctx) Slot {
	if m.Holder == 0 {
		m.Holder = msg.ID
		ctx.Send(msg.From, Granted{})
	}
	return m
})
actor.Post(s, "slot", TryTake{From: "r1", ID: 1})
s.Inv("mutual exclusion", func(w actor.Snapshot) bool { ... })
s.EventuallyAlways("someone holds the slot", func(w actor.Snapshot) bool { ... })
actor.Check(t, s)
```

Machine states and messages must be plain values: booleans, numbers, strings, and arrays or structs of those. Anything else is rejected at registration time with a clear error. A delivered message with no matching handler panics instead of being dropped silently.

## Examples

- `examples/mutex`: the race above, and the atomic variant proven safe.
- `examples/diehard`: the water jug puzzle. The claim that the big jug never holds 4 gallons is disproven, and the counterexample is the solution.
- `examples/peterson`: Peterson's algorithm proven safe across all interleavings, and the classic bug of taking the turn for yourself caught.
- `examples/msgslot`: the same slot race through mailboxes, and a try-take variant proven safe with a liveness property under fair delivery.
