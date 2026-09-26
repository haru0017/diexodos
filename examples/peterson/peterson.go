// Package peterson models Peterson's mutual exclusion algorithm for two
// processes. The correct version yields the turn to the other process. The
// buggy variant takes the turn for itself, a classic mistake that lets both
// processes enter the critical section.
package peterson

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

// Program counters per process.
const (
	idle = iota
	flagged
	waiting
	critical
)

type State struct {
	Flag [2]bool
	Turn int
	PC   [2]int
}

func Spec(yieldTurn bool) dex.Spec[State] {
	var actions []dex.Action[State]
	for i := 0; i <= 1; i++ {
		j := 1 - i
		turn := j
		if !yieldTurn {
			turn = i
		}
		actions = append(actions,
			dex.Act(fmt.Sprintf("flag(%d)", i),
				func(s State) bool { return s.PC[i] == idle },
				func(s State) State { s.Flag[i] = true; s.PC[i] = flagged; return s }),
			dex.Act(fmt.Sprintf("turn(%d)", i),
				func(s State) bool { return s.PC[i] == flagged },
				func(s State) State { s.Turn = turn; s.PC[i] = waiting; return s }),
			dex.Act(fmt.Sprintf("enter(%d)", i),
				func(s State) bool { return s.PC[i] == waiting && (!s.Flag[j] || s.Turn == i) },
				func(s State) State { s.PC[i] = critical; return s }),
			dex.Act(fmt.Sprintf("exit(%d)", i),
				func(s State) bool { return s.PC[i] == critical },
				func(s State) State { s.Flag[i] = false; s.PC[i] = idle; return s }))
	}

	return dex.Spec[State]{
		Init:    []State{{}},
		Actions: actions,
		Invariants: []dex.Invariant[State]{
			dex.Inv("mutual exclusion", func(s State) bool {
				return !(s.PC[0] == critical && s.PC[1] == critical)
			}),
		},
	}
}
