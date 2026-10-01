// Package wiretransfer models the classic introduction to model checking: two
// transfers from the same account, each checking the balance and then
// withdrawing in separate steps. Both can pass the check before either
// withdraws, and the account that must never go negative does. Making the
// check and the withdrawal one atomic step removes every bad interleaving.
package wiretransfer

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

type Stage int

const (
	Ready Stage = iota
	Checked
	Done
)

func (s Stage) String() string {
	return [...]string{"ready", "checked", "done"}[s]
}

const (
	balance = 10
	amount  = 6 // two of these overdraw the account
)

type State struct {
	Balance int
	Stage   [2]Stage
}

type Config struct {
	// Atomic folds the check and the withdrawal into one step.
	Atomic bool
}

func Spec(cfg Config) dex.Spec[State] {
	var actions []dex.Action[State]
	for i := range 2 {
		if cfg.Atomic {
			actions = append(actions, dex.Act(fmt.Sprintf("transfer %d checks and withdraws", i),
				func(s State) bool { return s.Stage[i] == Ready && s.Balance >= amount },
				func(s State) State { s.Balance -= amount; s.Stage[i] = Done; return s }))
			continue
		}
		actions = append(actions,
			dex.Act(fmt.Sprintf("transfer %d checks the balance", i),
				func(s State) bool { return s.Stage[i] == Ready && s.Balance >= amount },
				func(s State) State { s.Stage[i] = Checked; return s }),
			dex.Act(fmt.Sprintf("transfer %d withdraws", i),
				func(s State) bool { return s.Stage[i] == Checked },
				func(s State) State { s.Balance -= amount; s.Stage[i] = Done; return s }),
		)
	}
	return dex.Spec[State]{
		Init:    []State{{Balance: balance}},
		Actions: actions,
		Invariants: []dex.Invariant[State]{
			dex.Inv("the balance never goes negative", func(s State) bool {
				return s.Balance >= 0
			}),
		},
	}
}
