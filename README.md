# diexodos

An explicit-state model checker embedded in Go.

Describe a system as a state type, a set of guarded actions, and properties. The checker exhaustively walks every reachable state across every interleaving. When an invariant breaks it reports a shortest counterexample trace. When a liveness property breaks it reports a fair lasso: a path into a cycle that can repeat forever under the declared fairness. Specs are plain Go and run under `go test`.

The name is the Greek word διέξοδος, a way through. Its adverb διεξοδικά means exhaustively. The import name is `dex`.

```sh
go get github.com/haru0017/diexodos
```

## Quick start

The classic first model: two wire transfers leave the same account, and checking the balance and withdrawing are separate steps, so both transfers can pass the check before either withdraws:

```go
import "github.com/haru0017/diexodos/dex"

type Stage int

const (
	Ready Stage = iota
	Checked
	Done
)

func (s Stage) String() string {
	return [...]string{"ready", "checked", "done"}[s]
}

type State struct {
	Balance int
	Stage   [2]Stage
}

func Spec() dex.Spec[State] {
	var actions []dex.Action[State]
	for i := range 2 {
		actions = append(actions,
			dex.Act(fmt.Sprintf("transfer %d checks the balance", i),
				func(s State) bool { return s.Stage[i] == Ready && s.Balance >= 6 },
				func(s State) State { s.Stage[i] = Checked; return s }),
			dex.Act(fmt.Sprintf("transfer %d withdraws", i),
				func(s State) bool { return s.Stage[i] == Checked },
				func(s State) State { s.Balance -= 6; s.Stage[i] = Done; return s }))
	}
	return dex.Spec[State]{
		Init:    []State{{Balance: 10}},
		Actions: actions,
		Invariants: []dex.Invariant[State]{
			dex.Inv("the balance never goes negative", func(s State) bool { return s.Balance >= 0 }),
		},
	}
}

func TestTransfers(t *testing.T) {
	dex.Check(t, Spec()) // fails with a counterexample trace
}
```

The checker finds the shortest interleaving that breaks the invariant:

```
invariant "the balance never goes negative" violated:
   0. (init) {Balance:10 Stage:[ready ready]}
   1. transfer 0 checks the balance: {Balance:10 Stage:[checked ready]}
   2. transfer 1 checks the balance: {Balance:10 Stage:[checked checked]}
   3. transfer 0 withdraws: {Balance:4 Stage:[done checked]}
   4. transfer 1 withdraws: {Balance:-2 Stage:[done done]}
```

`dex.Check` fails the test on any violation, a deadlock when `dex.DetectDeadlocks()` is set, or a truncated search. `dex.Run` returns the result instead, for tests where a violation is the expected outcome. `dex.MaxStates(n)` bounds the search and `dex.Progress(w)` reports progress periodically. A model with unbounded data is made finite by `Spec.Constraint`, a predicate that keeps exploration inside the states it accepts.

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

## Counterexample reports

The `report` package renders a counterexample as Markdown with a mermaid diagram, written next to the test that pins it:

```go
import "github.com/haru0017/diexodos/report"

func TestPoliteProtocolLivelocks(t *testing.T) {
	res := dex.Run(spec) // polite dining philosophers: everyone yields, nobody eats
	if res.Lasso == nil {
		t.Fatal("expected the livelock")
	}
	report.Save(t, res) // writes TestPoliteProtocolLivelocks.md beside this file
}
```

A green result writes nothing. The output is deterministic, so an unchanged counterexample leaves the working tree clean and a changed one shows up in the diff: the report behaves like a golden file without being asserted on. `report.Markdown(res)` returns the document instead of writing it.

The document carries the violated property, a mermaid diagram of the trace, and a step table. Each distinct state is drawn once, so a lasso closes into a visible loop and its repeating edges are thick; the state that breaks an invariant is highlighted. States render through `%+v`, so enum fields read best with a `String()` method.

GitHub renders the committed reports, so the examples double as a gallery: [a livelock closing into a loop](examples/philosophers/TestPoliteProtocolLivelocks.md), [a deadlock stuttering forever](examples/philosophers/TestNaiveProtocolDeadlocks.md), [two-phase commit blocking](examples/twophase/TestAStoppedManagerBlocksThePrepared.md), and [a straight path into a bad state](examples/wiretransfer/TestSeparateCheckOverdraws.md).

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

- `examples/wiretransfer`: the overdraft above, and the atomic variant proven safe.
- `examples/diehard`: the water jug puzzle. The claim that the big jug never holds 4 gallons is disproven, and the counterexample is the solution.
- `examples/peterson`: Peterson's algorithm proven safe across all interleavings, and the classic bug of taking the turn for yourself caught.
- `examples/msgslot`: two runners racing for one slot through mailboxes, and a try-take variant proven safe with a liveness property under fair delivery.
- `examples/philosophers`: dining philosophers. The naive protocol deadlocks, the polite one livelocks, and breaking the symmetry is proven to feed everyone. The committed reports show both loops.
- `examples/twophase`: transaction commit in the shape of the TLA+ tutorial's TCommit. Consistency is proven, and so is the famous weakness: a stopped transaction manager blocks a prepared participant forever.
