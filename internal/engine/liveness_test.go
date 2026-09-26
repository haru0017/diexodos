package engine

import "testing"

// A process that can spin forever or finish. Without fairness on finish the
// spin cycle is a valid counterexample. With it, the cycle is unfair and the
// property holds.
func spinnerRules(fairFinish bool) []Rule[int] {
	return []Rule[int]{
		{Name: "spin", Next: func(s int) []int {
			if s == 0 {
				return []int{0}
			}
			return nil
		}},
		{Name: "finish", Fair: fairFinish, Next: func(s int) []int {
			if s == 0 {
				return []int{1}
			}
			return nil
		}},
	}
}

func explore(t *testing.T, init []int, rules []Rule[int]) *Graph[int] {
	t.Helper()
	res, g := Explore(init, rules, nil, ident, Options{BuildGraph: true})
	if res.Violation != nil || g == nil {
		t.Fatalf("unexpected explore result: %+v", res)
	}
	return g
}

func TestUnfairSpinnerViolatesLiveness(t *testing.T) {
	g := explore(t, []int{0}, spinnerRules(false))
	done := Liveness[int]{Name: "done", Mode: ModeEventuallyAlways, Holds: func(s int) bool { return s == 1 }}
	l := CheckLiveness(g, []Liveness[int]{done})
	if l == nil {
		t.Fatal("expected a lasso")
	}
	if len(l.Cycle) != 1 || l.Cycle[0].Rule != "spin" {
		t.Fatalf("unexpected cycle: %+v", l.Cycle)
	}
}

func TestFairFinishSatisfiesLiveness(t *testing.T) {
	g := explore(t, []int{0}, spinnerRules(true))
	done := Liveness[int]{Name: "done", Mode: ModeEventuallyAlways, Holds: func(s int) bool { return s == 1 }}
	if l := CheckLiveness(g, []Liveness[int]{done}); l != nil {
		t.Fatalf("unexpected lasso: %+v", l)
	}
}

func TestTerminalStateStutters(t *testing.T) {
	// 0 -> 1, then 1 stutters forever with the predicate false.
	g := explore(t, []int{0}, []Rule[int]{incUpTo(1)})
	stuck := Liveness[int]{Name: "back to zero", Mode: ModeEventuallyAlways, Holds: func(s int) bool { return s == 0 }}
	l := CheckLiveness(g, []Liveness[int]{stuck})
	if l == nil {
		t.Fatal("expected a lasso")
	}
	if len(l.Cycle) != 1 || l.Cycle[0].Rule != "(stutter)" {
		t.Fatalf("unexpected cycle: %+v", l.Cycle)
	}
}

// 0 and 1 toggle forever. Escape to a terminal 2 stays enabled in both, so
// weak fairness of escape rules the toggle cycle out. If escape were enabled
// only in one of them, weak fairness would not: the cycle keeps disabling it.
func escapeRules(fairEscape bool) []Rule[int] {
	return []Rule[int]{
		{Name: "toggle", Next: func(s int) []int {
			if s <= 1 {
				return []int{1 - s}
			}
			return nil
		}},
		{Name: "escape", Fair: fairEscape, Next: func(s int) []int {
			if s <= 1 {
				return []int{2}
			}
			return nil
		}},
	}
}

func TestAlwaysEventuallyWithFairEscape(t *testing.T) {
	prop := Liveness[int]{Name: "reaches two", Mode: ModeAlwaysEventually, Holds: func(s int) bool { return s == 2 }}

	g := explore(t, []int{0}, escapeRules(false))
	l := CheckLiveness(g, []Liveness[int]{prop})
	if l == nil {
		t.Fatal("expected a lasso in the unfair variant")
	}
	if len(l.Cycle) != 2 {
		t.Fatalf("unexpected cycle: %+v", l.Cycle)
	}

	g = explore(t, []int{0}, escapeRules(true))
	if l := CheckLiveness(g, []Liveness[int]{prop}); l != nil {
		t.Fatalf("unexpected lasso: %+v", l)
	}
}

// Escape is enabled only at state 1, so the toggle cycle keeps disabling it.
// Weak fairness cannot rule the cycle out, strong fairness can: escape is
// enabled infinitely often on the cycle yet never taken.
func TestStrongFairnessBeatsIntermittentEnabling(t *testing.T) {
	rules := func(strongEscape bool) []Rule[int] {
		return []Rule[int]{
			{Name: "toggle", Next: func(s int) []int {
				if s <= 1 {
					return []int{1 - s}
				}
				return nil
			}},
			{Name: "escape", Fair: !strongEscape, StrongFair: strongEscape, Next: func(s int) []int {
				if s == 1 {
					return []int{2}
				}
				return nil
			}},
		}
	}
	prop := Liveness[int]{Name: "reaches two", Mode: ModeAlwaysEventually, Holds: func(s int) bool { return s == 2 }}

	g := explore(t, []int{0}, rules(false))
	if l := CheckLiveness(g, []Liveness[int]{prop}); l == nil {
		t.Fatal("weak fairness must not exclude the toggle cycle")
	}

	g = explore(t, []int{0}, rules(true))
	if l := CheckLiveness(g, []Liveness[int]{prop}); l != nil {
		t.Fatalf("strong fairness must exclude the toggle cycle: %+v", l)
	}
}

// Pruning the enabling state must still find a fair subcycle when one exists.
func TestStrongFairnessPruningKeepsSubcycle(t *testing.T) {
	// 0 <-> 1 and 1 <-> 2 form one component. Escape is enabled only at 2,
	// so pruning 2 leaves the 0 <-> 1 cycle, which is a valid counterexample.
	rules := []Rule[int]{
		{Name: "step", Next: func(s int) []int {
			switch s {
			case 0:
				return []int{1}
			case 1:
				return []int{0, 2}
			case 2:
				return []int{1}
			}
			return nil
		}},
		{Name: "escape", StrongFair: true, Next: func(s int) []int {
			if s == 2 {
				return []int{3}
			}
			return nil
		}},
	}
	g := explore(t, []int{0}, rules)
	prop := Liveness[int]{Name: "reaches three", Mode: ModeAlwaysEventually, Holds: func(s int) bool { return s == 3 }}
	l := CheckLiveness(g, []Liveness[int]{prop})
	if l == nil {
		t.Fatal("expected the pruned subcycle to remain a counterexample")
	}
	for _, s := range l.Cycle {
		if s.State == 2 {
			t.Fatalf("cycle must avoid the enabling state: %+v", l.Cycle)
		}
	}
}

func TestFairnessWitnessInsideCycle(t *testing.T) {
	// A fair rule taken inside the cycle keeps the cycle fair.
	rules := []Rule[int]{
		{Name: "fwd", Fair: true, Next: func(s int) []int {
			if s < 2 {
				return []int{s + 1}
			}
			return nil
		}},
		{Name: "back", Next: func(s int) []int {
			if s == 2 {
				return []int{0}
			}
			return nil
		}},
	}
	g := explore(t, []int{0}, rules)
	never := Liveness[int]{Name: "never three", Mode: ModeEventuallyAlways, Holds: func(s int) bool { return s == 3 }}
	l := CheckLiveness(g, []Liveness[int]{never})
	if l == nil {
		t.Fatal("expected a lasso: the fair rule fires inside the cycle")
	}
	if len(l.Cycle) != 3 {
		t.Fatalf("unexpected cycle: %+v", l.Cycle)
	}
}
