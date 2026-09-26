package dex

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/haru0017/diexodos/internal/engine"
)

// Step is one entry of a counterexample. Action is empty for an initial
// state.
type Step[S any] struct {
	Action string
	State  S
}

type Violation[S any] struct {
	Invariant string
	Path      []Step[S]
}

func (v *Violation[S]) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "invariant %q violated:\n", v.Invariant)
	writeSteps(&b, v.Path, 0)
	return b.String()
}

// Lasso is a liveness counterexample: after the prefix, the cycle can repeat
// forever without violating weak fairness.
type Lasso[S any] struct {
	Property string
	Prefix   []Step[S]
	Cycle    []Step[S]
}

func (l *Lasso[S]) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "liveness %q violated:\n", l.Property)
	writeSteps(&b, l.Prefix, 0)
	fmt.Fprintf(&b, "  then the cycle repeats forever:\n")
	writeSteps(&b, l.Cycle, len(l.Prefix))
	return b.String()
}

func writeSteps[S any](b *strings.Builder, steps []Step[S], base int) {
	for i, s := range steps {
		if s.Action == "" {
			fmt.Fprintf(b, "  %2d. (init) %+v\n", base+i, s.State)
		} else {
			fmt.Fprintf(b, "  %2d. %s: %+v\n", base+i, s.Action, s.State)
		}
	}
}

type Result[S any] struct {
	States    int
	Violation *Violation[S]
	Lasso     *Lasso[S]
	Truncated bool
}

type Option func(*config)

type config struct {
	maxStates int
	progress  io.Writer
	deadlocks bool
}

// MaxStates bounds exploration. Check treats hitting the bound as a failure
// because the result would not cover the full state space.
func MaxStates(n int) Option {
	return func(c *config) { c.maxStates = n }
}

// Progress reports the number of explored states to w periodically.
func Progress(w io.Writer) Option {
	return func(c *config) { c.progress = w }
}

// DetectDeadlocks reports a reachable state with no enabled action as a
// violation named "deadlock".
func DetectDeadlocks() Option {
	return func(c *config) { c.deadlocks = true }
}

// Run explores the spec using Go equality as state identity and returns the
// result without failing any test. Use it when a violation is the expected
// outcome.
func Run[S comparable](spec Spec[S], opts ...Option) Result[S] {
	return RunKeyed(spec, func(s S) S { return s }, opts...)
}

// RunKeyed explores a spec whose states are not comparable, using key to
// derive a canonical comparable identity. Two states with equal keys are
// treated as the same state, so key must be injective on the reachable
// states.
func RunKeyed[S any, K comparable](spec Spec[S], key func(S) K, opts ...Option) Result[S] {
	// An empty spec would report success without checking anything.
	if len(spec.Init) == 0 {
		panic("dex: spec has no initial states")
	}
	for _, a := range spec.Actions {
		if a.Update == nil && a.Multi == nil {
			panic(fmt.Sprintf("dex: action %q has neither Update nor Multi", a.Name))
		}
	}

	var c config
	for _, o := range opts {
		o(&c)
	}

	rules := make([]engine.Rule[S], len(spec.Actions))
	for i, a := range spec.Actions {
		rules[i] = engine.Rule[S]{Name: a.Name, Fair: a.Fair, StrongFair: a.StrongFair, Next: nextFunc(a, spec.Constraint)}
	}
	invs := make([]engine.Invariant[S], len(spec.Invariants))
	for i, inv := range spec.Invariants {
		invs[i] = engine.Invariant[S]{Name: inv.Name, Holds: inv.Holds}
	}

	eopts := engine.Options{
		MaxStates:       c.maxStates,
		DetectDeadlocks: c.deadlocks,
		BuildGraph:      len(spec.Liveness) > 0,
	}
	if c.progress != nil {
		w := c.progress
		eopts.Progress = func(states int) {
			fmt.Fprintf(w, "explored %d states\n", states)
		}
	}

	er, graph := engine.Explore(spec.Init, rules, invs, key, eopts)

	res := Result[S]{States: er.States, Truncated: er.Truncated}
	if er.Violation != nil {
		res.Violation = &Violation[S]{Invariant: er.Violation.Invariant, Path: steps(er.Violation.Path)}
		return res
	}
	if graph == nil || res.Truncated {
		return res
	}

	props := make([]engine.Liveness[S], len(spec.Liveness))
	for i, l := range spec.Liveness {
		props[i] = engine.Liveness[S]{Name: l.name, Mode: l.mode, Holds: l.holds}
	}
	if lasso := engine.CheckLiveness(graph, props); lasso != nil {
		res.Lasso = &Lasso[S]{Property: lasso.Property, Prefix: steps(lasso.Prefix), Cycle: steps(lasso.Cycle)}
	}
	return res
}

// Check explores the spec and fails t on any violation or a truncated search.
func Check[S comparable](t testing.TB, spec Spec[S], opts ...Option) Result[S] {
	t.Helper()
	return fail(t, Run(spec, opts...))
}

// CheckKeyed is Check for states that are not comparable. See RunKeyed.
func CheckKeyed[S any, K comparable](t testing.TB, spec Spec[S], key func(S) K, opts ...Option) Result[S] {
	t.Helper()
	return fail(t, RunKeyed(spec, key, opts...))
}

func fail[S any](t testing.TB, res Result[S]) Result[S] {
	t.Helper()
	if res.Violation != nil {
		t.Fatalf("%s", res.Violation)
	}
	if res.Lasso != nil {
		t.Fatalf("%s", res.Lasso)
	}
	if res.Truncated {
		t.Fatalf("search truncated at %d states; the result covers only part of the state space", res.States)
	}
	return res
}

func steps[S any](in []engine.Step[S]) []Step[S] {
	out := make([]Step[S], len(in))
	for i, s := range in {
		out[i] = Step[S]{Action: s.Rule, State: s.State}
	}
	return out
}

func nextFunc[S any](a Action[S], within func(S) bool) func(S) []S {
	return func(s S) []S {
		if a.Guard != nil && !a.Guard(s) {
			return nil
		}
		var next []S
		if a.Multi != nil {
			next = a.Multi(s)
		} else {
			next = []S{a.Update(s)}
		}
		if within == nil {
			return next
		}
		kept := make([]S, 0, len(next))
		for _, n := range next {
			if within(n) {
				kept = append(kept, n)
			}
		}
		return kept
	}
}
