# diexodos

An explicit-state model checker embedded in Go.

You describe a system as a state type, a set of guarded actions, and properties. The checker exhaustively walks every reachable state across every interleaving. When an invariant breaks it reports a shortest counterexample trace, and when a liveness property breaks it reports a fair lasso: a path into a cycle that can repeat forever. Specs are plain Go, run under `go test`, and need no external toolchain.

The name is the Greek word διέξοδος, a way through. Its adverb διεξοδικά means exhaustively. The import name is `dex`.

## Status

Early and experimental. Safety checking, deadlock detection, liveness with weak fairness, and a message-passing actor layer work. See the roadmap below.

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

## Liveness and fairness

Safety says a bad state is never reached. Liveness says a good thing eventually happens, and it only makes sense together with fairness: without it, the scheduler that starves an action forever is always a counterexample. Actions marked `dex.Fair` are weakly fair: a behavior that keeps the action enabled forever must eventually take it.

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

`dex.AlwaysEventually` states that a predicate holds infinitely often. Terminal states stutter forever, so a system that halts in a bad state is caught too.

## Actor layer

The `actor` package builds message-passing systems on top of the same engine: named machines, typed handlers, mailboxes. Delivery order across machines is explored exhaustively. Delivery is weakly fair by default, and message loss and reordering are opt-in system options, so the semantics of the channel is an explicit modeling decision.

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
s.Inv("mutual exclusion", func(w actor.Snapshot) bool { ... })
s.EventuallyAlways("someone holds the slot", func(w actor.Snapshot) bool { ... })
actor.Check(t, s)
```

Machine states and messages must be plain values: booleans, numbers, strings, and arrays or structs of those. Pointers, slices, and maps are rejected at registration time with a clear error instead of being silently mishandled.

## Examples

- `examples/mutex`: the race above, plus the atomic variant proven safe.
- `examples/diehard`: the water jug puzzle. The claim that the big jug never holds 4 gallons is disproven, and the counterexample is the solution.
- `examples/peterson`: Peterson's algorithm proven safe across all interleavings, and the classic bug of taking the turn for yourself caught.
- `examples/msgslot`: the same slot race through mailboxes, and a try-take variant proven safe with a liveness property under fair delivery.

## API

- `dex.Spec[S]` holds initial states, actions, invariants, and liveness properties.
- `dex.Act(name, guard, update)` declares a guarded transition. A nil guard is always enabled. `dex.ActN` produces several successors at once. Nondeterminism is expressed by declaring several enabled actions.
- `dex.Fair(action)` marks an action weakly fair for liveness checking.
- `dex.Inv(name, predicate)` declares an invariant. `dex.EventuallyAlways` and `dex.AlwaysEventually` declare liveness properties.
- `dex.Check(t, spec, opts...)` explores and fails the test on any violation or a truncated search. `dex.Run` returns the result instead, for tests where a violation is the expected outcome.
- `dex.RunKeyed` and `dex.CheckKeyed` handle states that are not comparable, using a caller-supplied canonical key.
- Options: `dex.MaxStates(n)` bounds the search, `dex.Progress(w)` reports progress periodically, `dex.DetectDeadlocks()` reports states with no enabled action.

## Design notes

State identity is exact. Comparable states are their own map key, and keyed states go through an injective canonical key, so two distinct states are never conflated by a hash collision.

Exploration is breadth first, so the first counterexample found is a shortest one.

Liveness checking totalizes the graph with stuttering self-loops, finds strongly connected components, and accepts a violating component only if a closed walk covering it satisfies weak fairness: every fair action is either taken inside the component or disabled somewhere in it. The reported lasso visits every witness that argument needs.

The engine lives in an internal package behind the small `dex` surface, keeping room to change the exploration strategy without breaking the API.

## Roadmap

- Code generation for canonical keys and clones of user-defined rich states.
- Strong fairness.
- Deferred messages and crash modeling helpers in the actor layer.
- Symmetry reduction and parallel exploration.
