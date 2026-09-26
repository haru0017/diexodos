// Package dex is an embedded explicit-state model checker. A model is a set
// of guarded actions over a comparable state type. Check explores every
// reachable state and reports a shortest counterexample when an invariant
// breaks.
package dex

// Spec is a checkable model.
type Spec[S comparable] struct {
	Init       []S
	Actions    []Action[S]
	Invariants []Invariant[S]
}

// Action is a guarded transition. A nil Guard is always enabled.
type Action[S comparable] struct {
	Name   string
	Guard  func(S) bool
	Update func(S) S
}

func Act[S comparable](name string, guard func(S) bool, update func(S) S) Action[S] {
	return Action[S]{Name: name, Guard: guard, Update: update}
}

// Invariant must hold in every reachable state.
type Invariant[S comparable] struct {
	Name  string
	Holds func(S) bool
}

func Inv[S comparable](name string, holds func(S) bool) Invariant[S] {
	return Invariant[S]{Name: name, Holds: holds}
}
