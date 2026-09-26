// Package mutex models two processes racing for one slot. When checking the
// slot and taking it are separate steps, both processes can observe a free
// slot and both take it. A single atomic test-and-set step restores mutual
// exclusion.
package mutex

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

type State struct {
	Saw   [3]bool
	Holds [3]bool
}

func Spec(atomicTake bool) dex.Spec[State] {
	free := func(s State) bool { return !s.Holds[1] && !s.Holds[2] }

	var actions []dex.Action[State]
	for i := 1; i <= 2; i++ {
		if atomicTake {
			actions = append(actions, dex.Act(fmt.Sprintf("take(%d)", i),
				free,
				func(s State) State { s.Holds[i] = true; return s }))
			continue
		}
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
