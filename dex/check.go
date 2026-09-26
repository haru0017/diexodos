package dex

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/haru0017/diexodos/internal/engine"
)

// Step is one entry of a counterexample path. Action is empty for an initial
// state.
type Step[S comparable] struct {
	Action string
	State  S
}

type Violation[S comparable] struct {
	Invariant string
	Path      []Step[S]
}

func (v *Violation[S]) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "invariant %q violated:\n", v.Invariant)
	for i, s := range v.Path {
		if s.Action == "" {
			fmt.Fprintf(&b, "  %2d. (init) %+v\n", i, s.State)
		} else {
			fmt.Fprintf(&b, "  %2d. %s: %+v\n", i, s.Action, s.State)
		}
	}
	return b.String()
}

type Result[S comparable] struct {
	States    int
	Violation *Violation[S]
	Truncated bool
}

type Option func(*config)

type config struct {
	maxStates int
	progress  io.Writer
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

// Run explores the spec and returns the result without failing any test.
// Use it when a violation is the expected outcome.
func Run[S comparable](spec Spec[S], opts ...Option) Result[S] {
	var c config
	for _, o := range opts {
		o(&c)
	}

	rules := make([]engine.Rule[S], len(spec.Actions))
	for i, a := range spec.Actions {
		rules[i] = engine.Rule[S]{Name: a.Name, Next: nextFunc(a)}
	}
	invs := make([]engine.Invariant[S], len(spec.Invariants))
	for i, inv := range spec.Invariants {
		invs[i] = engine.Invariant[S]{Name: inv.Name, Holds: inv.Holds}
	}

	eopts := engine.Options{MaxStates: c.maxStates}
	if c.progress != nil {
		w := c.progress
		eopts.Progress = func(states int) {
			fmt.Fprintf(w, "explored %d states\n", states)
		}
	}

	er := engine.Explore(spec.Init, rules, invs, eopts)

	res := Result[S]{States: er.States, Truncated: er.Truncated}
	if er.Violation != nil {
		path := make([]Step[S], len(er.Violation.Path))
		for i, s := range er.Violation.Path {
			path[i] = Step[S]{Action: s.Rule, State: s.State}
		}
		res.Violation = &Violation[S]{Invariant: er.Violation.Invariant, Path: path}
	}
	return res
}

// Check explores the spec and fails t on a violation or a truncated search.
func Check[S comparable](t testing.TB, spec Spec[S], opts ...Option) Result[S] {
	t.Helper()
	res := Run(spec, opts...)
	if res.Violation != nil {
		t.Fatalf("%s", res.Violation)
	}
	if res.Truncated {
		t.Fatalf("search truncated at %d states; the result covers only part of the state space", res.States)
	}
	return res
}

func nextFunc[S comparable](a Action[S]) func(S) []S {
	return func(s S) []S {
		if a.Guard != nil && !a.Guard(s) {
			return nil
		}
		return []S{a.Update(s)}
	}
}
