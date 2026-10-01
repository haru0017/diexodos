// Package twophase models transaction commit in the shape of the TLA+
// tutorial's TCommit: resource managers prepare, the transaction manager
// commits only when every one of them is prepared, decisions are irreversible,
// and a working resource manager may still abort on its own. Consistency, no
// mixed outcome, holds in every interleaving. So does the protocol's famous
// weakness: two-phase commit blocks, a prepared resource manager waits on the
// transaction manager and a stopped manager leaves it undecided forever.
package twophase

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

type RM int

const (
	Working RM = iota
	Prepared
	Committed
	Aborted
)

func (r RM) String() string {
	return [...]string{"working", "prepared", "committed", "aborted"}[r]
}

type TM int

const (
	Deciding TM = iota
	CommitDecided
	AbortDecided
	Stopped
)

func (m TM) String() string {
	return [...]string{"deciding", "committed", "aborted", "stopped"}[m]
}

type State struct {
	TM TM
	RM [2]RM
}

type Config struct {
	// Crash lets the transaction manager stop before deciding.
	Crash bool
}

func Spec(cfg Config) dex.Spec[State] {
	prepared := func(s State) bool { return s.RM[0] == Prepared && s.RM[1] == Prepared }

	actions := []dex.Action[State]{
		dex.Act("the TM decides to commit",
			func(s State) bool { return s.TM == Deciding && prepared(s) },
			func(s State) State { s.TM = CommitDecided; return s }),
		dex.Act("the TM decides to abort",
			func(s State) bool { return s.TM == Deciding },
			func(s State) State { s.TM = AbortDecided; return s }),
	}
	for i := range 2 {
		actions = append(actions,
			dex.Act(fmt.Sprintf("RM%d prepares", i),
				func(s State) bool { return s.RM[i] == Working },
				func(s State) State { s.RM[i] = Prepared; return s }),
			dex.Act(fmt.Sprintf("RM%d aborts on its own", i),
				func(s State) bool { return s.RM[i] == Working },
				func(s State) State { s.RM[i] = Aborted; return s }),
			dex.Act(fmt.Sprintf("RM%d learns the commit", i),
				func(s State) bool { return s.TM == CommitDecided && s.RM[i] == Prepared },
				func(s State) State { s.RM[i] = Committed; return s }),
			dex.Act(fmt.Sprintf("RM%d learns the abort", i),
				func(s State) bool {
					return s.TM == AbortDecided && (s.RM[i] == Working || s.RM[i] == Prepared)
				},
				func(s State) State { s.RM[i] = Aborted; return s }),
		)
	}
	if cfg.Crash {
		actions = append(actions, dex.Act("the TM stops",
			func(s State) bool { return s.TM == Deciding },
			func(s State) State { s.TM = Stopped; return s }))
	}
	return dex.Spec[State]{Init: []State{{}}, Actions: actions}
}

// Consistent is transaction commit's defining invariant: no resource manager
// sees a commit while another sees an abort.
func Consistent() dex.Invariant[State] {
	return dex.Inv("no mixed outcome", func(s State) bool {
		committed := s.RM[0] == Committed || s.RM[1] == Committed
		aborted := s.RM[0] == Aborted || s.RM[1] == Aborted
		return !(committed && aborted)
	})
}

func EveryoneDecides() dex.Liveness[State] {
	return dex.EventuallyAlways("every resource manager decides", func(s State) bool {
		for _, r := range s.RM {
			if r != Committed && r != Aborted {
				return false
			}
		}
		return true
	})
}
