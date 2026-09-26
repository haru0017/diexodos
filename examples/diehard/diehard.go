// Package diehard solves the water jug puzzle from Die Hard 3 with the
// checker. Claiming that the big jug never holds exactly 4 gallons makes the
// checker disprove it, and the counterexample is the solution.
package diehard

import "github.com/haru0017/diexodos/dex"

// State holds the gallons in the 5 gallon and 3 gallon jugs.
type State struct {
	Big   int
	Small int
}

func Spec() dex.Spec[State] {
	return dex.Spec[State]{
		Init: []State{{}},
		Actions: []dex.Action[State]{
			dex.Act("fill big", nil, func(s State) State { s.Big = 5; return s }),
			dex.Act("fill small", nil, func(s State) State { s.Small = 3; return s }),
			dex.Act("empty big", nil, func(s State) State { s.Big = 0; return s }),
			dex.Act("empty small", nil, func(s State) State { s.Small = 0; return s }),
			dex.Act("pour big into small", nil, func(s State) State {
				n := min(s.Big, 3-s.Small)
				s.Big -= n
				s.Small += n
				return s
			}),
			dex.Act("pour small into big", nil, func(s State) State {
				n := min(s.Small, 5-s.Big)
				s.Small -= n
				s.Big += n
				return s
			}),
		},
		Invariants: []dex.Invariant[State]{
			dex.Inv("big jug is never at 4", func(s State) bool { return s.Big != 4 }),
		},
	}
}
