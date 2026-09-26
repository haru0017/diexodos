# diexodos

An explicit-state model checker embedded in Go.

You describe a system as a state type, a set of guarded actions, and invariants. The checker exhaustively walks every reachable state across every interleaving and reports a shortest counterexample trace when an invariant breaks. Specs are plain Go, run under `go test`, and need no external toolchain.

The name is the Greek word διέξοδος, a way through. Its adverb διεξοδικά means exhaustively. The import name is `dex`.

## Status

Early and experimental. Safety checking works. See the roadmap below.

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

Counterexamples are also useful on their own. `examples/diehard` states that the big jug never holds 4 gallons, and the counterexample is the solution to the puzzle.

## API

- `dex.Spec[S]` holds initial states, actions, and invariants. `S` is any comparable type.
- `dex.Act(name, guard, update)` declares a guarded transition. A nil guard is always enabled. Nondeterminism is expressed by declaring several enabled actions.
- `dex.Inv(name, predicate)` declares an invariant checked in every reachable state.
- `dex.Check(t, spec, opts...)` explores and fails the test on a violation or a truncated search.
- `dex.Run(spec, opts...)` explores and returns the result, for tests where a violation is the expected outcome.
- Options: `dex.MaxStates(n)` bounds the search, `dex.Progress(w)` reports progress periodically.

## Design notes

States must be comparable. State identity is Go equality on a map key, so two distinct states can never be conflated by a hash collision. This also means states are plain values: copying is assignment and no reflection runs during the search.

Exploration is breadth first, so the first counterexample found is a shortest one.

The engine lives in an internal package behind the small `dex` surface, keeping room to change the exploration strategy without breaking the API.

## Roadmap

- Keyed states through code generation, for states that hold slices or maps such as message queues.
- An actor layer on top of guarded actions: machines, mailboxes, message delivery with configurable loss and reordering.
- Deadlock detection.
- Liveness properties with fairness.
