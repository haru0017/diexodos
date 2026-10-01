// Package philosophers models the dining philosophers. Three philosophers
// share three forks and each needs both adjacent forks to eat. The naive
// protocol deadlocks once everyone holds a left fork. The polite variant that
// yields the fork instead livelocks: everyone can pick up and put back
// forever. Breaking the symmetry, one philosopher reaching for the right fork
// first, is the classic fix.
package philosophers

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

type Phase int

const (
	Thinking Phase = iota
	OneFork
	Done
)

func (p Phase) String() string {
	return [...]string{"thinking", "one fork", "done"}[p]
}

type State struct {
	Taken [3]bool // fork i sits between philosopher i and i+1
	Phase [3]Phase
}

type Config struct {
	// Polite philosophers put the first fork back when the second is taken.
	Polite bool
	// Asymmetric makes philosopher 2 reach for the right fork first.
	Asymmetric bool
}

func Spec(cfg Config) dex.Spec[State] {
	// forks returns the order philosopher i picks its two forks in.
	forks := func(i int) (first, second int) {
		first, second = i, (i+1)%3
		if cfg.Asymmetric && i == 2 {
			first, second = second, first
		}
		return first, second
	}

	var actions []dex.Action[State]
	for i := range 3 {
		first, second := forks(i)
		actions = append(actions,
			dex.Act(fmt.Sprintf("P%d takes a first fork", i),
				func(s State) bool { return s.Phase[i] == Thinking && !s.Taken[first] },
				func(s State) State { s.Taken[first] = true; s.Phase[i] = OneFork; return s }),
			dex.Act(fmt.Sprintf("P%d takes the second fork and eats", i),
				func(s State) bool { return s.Phase[i] == OneFork && !s.Taken[second] },
				func(s State) State { s.Taken[first] = false; s.Phase[i] = Done; return s }),
		)
		if cfg.Polite {
			actions = append(actions, dex.Act(fmt.Sprintf("P%d puts the fork back", i),
				func(s State) bool { return s.Phase[i] == OneFork && s.Taken[second] },
				func(s State) State { s.Taken[first] = false; s.Phase[i] = Thinking; return s }))
		}
	}
	return dex.Spec[State]{Init: []State{{}}, Actions: actions}
}

func EveryoneEats() dex.Liveness[State] {
	return dex.EventuallyAlways("everyone has eaten", func(s State) bool {
		return s.Phase[0] == Done && s.Phase[1] == Done && s.Phase[2] == Done
	})
}
