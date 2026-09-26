// Package dex is an embedded explicit-state model checker. A model is a set
// of guarded actions over a state type. Check explores every reachable state
// and reports a shortest counterexample when an invariant breaks, and a fair
// lasso when a liveness property breaks.
package dex

import "github.com/haru0017/diexodos/internal/engine"

// Spec is a checkable model.
type Spec[S any] struct {
	Init       []S
	Actions    []Action[S]
	Invariants []Invariant[S]
	Liveness   []Liveness[S]
}

// Action is a guarded transition. A nil Guard is always enabled. Multi, when
// set, replaces Update and may produce several successors at once.
type Action[S any] struct {
	Name       string
	Fair       bool
	StrongFair bool
	Guard      func(S) bool
	Update     func(S) S
	Multi      func(S) []S
}

func Act[S any](name string, guard func(S) bool, update func(S) S) Action[S] {
	return Action[S]{Name: name, Guard: guard, Update: update}
}

// ActN declares a transition that may produce several successors at once.
func ActN[S any](name string, guard func(S) bool, multi func(S) []S) Action[S] {
	return Action[S]{Name: name, Guard: guard, Multi: multi}
}

// Fair marks the action as weakly fair: a behavior that keeps the action
// enabled forever must eventually take it. Only liveness checking uses this.
func Fair[S any](a Action[S]) Action[S] {
	a.Fair = true
	return a
}

// StronglyFair marks the action as strongly fair: a behavior that enables the
// action infinitely often must take it infinitely often, even if the action
// keeps being disabled in between. Only liveness checking uses this.
func StronglyFair[S any](a Action[S]) Action[S] {
	a.StrongFair = true
	return a
}

// Invariant must hold in every reachable state.
type Invariant[S any] struct {
	Name  string
	Holds func(S) bool
}

func Inv[S any](name string, holds func(S) bool) Invariant[S] {
	return Invariant[S]{Name: name, Holds: holds}
}

// Liveness is a temporal property checked over infinite behaviors under weak
// fairness.
type Liveness[S any] struct {
	name  string
	mode  engine.LivenessMode
	holds func(S) bool
}

// EventuallyAlways states that on every fair behavior the predicate
// eventually holds forever.
func EventuallyAlways[S any](name string, holds func(S) bool) Liveness[S] {
	return Liveness[S]{name: name, mode: engine.ModeEventuallyAlways, holds: holds}
}

// AlwaysEventually states that on every fair behavior the predicate holds
// infinitely often.
func AlwaysEventually[S any](name string, holds func(S) bool) Liveness[S] {
	return Liveness[S]{name: name, mode: engine.ModeAlwaysEventually, holds: holds}
}
