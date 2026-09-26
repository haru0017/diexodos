package engine

import (
	"reflect"
	"testing"
)

func ident(s int) int { return s }

func incUpTo(limit int) Rule[int] {
	return Rule[int]{Name: "inc", Next: func(s int) []int {
		if s < limit {
			return []int{s + 1}
		}
		return nil
	}}
}

func TestLinearChain(t *testing.T) {
	res, _ := Explore([]int{0}, []Rule[int]{incUpTo(2)}, nil, ident, Options{})
	if res.States != 3 || res.Violation != nil || res.Truncated {
		t.Fatalf("got %+v", res)
	}
}

func TestDedup(t *testing.T) {
	// Two rules reaching the same successor must count it once.
	a := Rule[int]{Name: "a", Next: func(s int) []int { return []int{s + 1} }}
	b := Rule[int]{Name: "b", Next: func(s int) []int { return []int{s + 1} }}
	inv := Invariant[int]{Name: "bound", Holds: func(s int) bool { return s <= 2 }}
	res, _ := Explore([]int{0}, []Rule[int]{a, b}, []Invariant[int]{inv}, ident, Options{})
	if res.Violation == nil {
		t.Fatal("expected violation at 3")
	}
	if res.States != 4 {
		t.Fatalf("states = %d, want 4", res.States)
	}
}

func TestCycleTerminates(t *testing.T) {
	mod := Rule[int]{Name: "mod", Next: func(s int) []int { return []int{(s + 1) % 3} }}
	res, _ := Explore([]int{0}, []Rule[int]{mod}, nil, ident, Options{})
	if res.States != 3 || res.Violation != nil {
		t.Fatalf("got %+v", res)
	}
}

func TestShortestCounterexample(t *testing.T) {
	slow := Rule[int]{Name: "slow", Next: func(s int) []int {
		if s < 10 {
			return []int{s + 1}
		}
		return nil
	}}
	jump := Rule[int]{Name: "jump", Next: func(s int) []int {
		if s == 0 {
			return []int{10}
		}
		return nil
	}}
	inv := Invariant[int]{Name: "not ten", Holds: func(s int) bool { return s != 10 }}
	res, _ := Explore([]int{0}, []Rule[int]{slow, jump}, []Invariant[int]{inv}, ident, Options{})
	if res.Violation == nil {
		t.Fatal("expected violation")
	}
	want := []Step[int]{{Rule: "", State: 0}, {Rule: "jump", State: 10}}
	if !reflect.DeepEqual(res.Violation.Path, want) {
		t.Fatalf("path = %+v, want %+v", res.Violation.Path, want)
	}
}

func TestMaxStatesTruncates(t *testing.T) {
	inc := Rule[int]{Name: "inc", Next: func(s int) []int { return []int{s + 1} }}
	res, _ := Explore([]int{0}, []Rule[int]{inc}, nil, ident, Options{MaxStates: 100})
	if !res.Truncated || res.States != 100 {
		t.Fatalf("got %+v", res)
	}
}

func TestInitialStateViolation(t *testing.T) {
	inv := Invariant[int]{Name: "nonzero", Holds: func(s int) bool { return s != 0 }}
	res, _ := Explore([]int{0}, nil, []Invariant[int]{inv}, ident, Options{})
	if res.Violation == nil || len(res.Violation.Path) != 1 {
		t.Fatalf("got %+v", res)
	}
}

func TestDeterminism(t *testing.T) {
	rules := []Rule[int]{incUpTo(50), {Name: "double", Next: func(s int) []int {
		if s > 0 && s < 30 {
			return []int{s * 2}
		}
		return nil
	}}}
	inv := Invariant[int]{Name: "bound", Holds: func(s int) bool { return s < 55 }}
	r1, _ := Explore([]int{0}, rules, []Invariant[int]{inv}, ident, Options{})
	r2, _ := Explore([]int{0}, rules, []Invariant[int]{inv}, ident, Options{})
	if !reflect.DeepEqual(r1, r2) {
		t.Fatalf("nondeterministic results:\n%+v\n%+v", r1, r2)
	}
}

func TestDeadlockDetection(t *testing.T) {
	// 0 -> 1 -> 2, and 2 has no successors.
	res, _ := Explore([]int{0}, []Rule[int]{incUpTo(2)}, nil, ident, Options{DetectDeadlocks: true})
	if res.Violation == nil || res.Violation.Invariant != "deadlock" {
		t.Fatalf("got %+v", res)
	}
	if got := len(res.Violation.Path); got != 3 {
		t.Fatalf("path length = %d, want 3", got)
	}

	// A cycle never deadlocks even though it produces no new states.
	mod := Rule[int]{Name: "mod", Next: func(s int) []int { return []int{(s + 1) % 3} }}
	res, _ = Explore([]int{0}, []Rule[int]{mod}, nil, ident, Options{DetectDeadlocks: true})
	if res.Violation != nil {
		t.Fatalf("got %+v", res)
	}
}

func TestGraphIsBuilt(t *testing.T) {
	res, g := Explore([]int{0}, []Rule[int]{incUpTo(2)}, nil, ident, Options{BuildGraph: true})
	if res.Violation != nil || g == nil {
		t.Fatalf("got %+v, graph %v", res, g)
	}
	if len(g.States) != 3 || len(g.Edges[0]) != 1 || len(g.Edges[2]) != 0 {
		t.Fatalf("unexpected graph: %+v", g)
	}
}

func TestProgressCallback(t *testing.T) {
	inc := Rule[int]{Name: "inc", Next: func(s int) []int { return []int{s + 1} }}
	calls := 0
	res, _ := Explore([]int{0}, []Rule[int]{inc}, nil, ident, Options{
		MaxStates:     100,
		Progress:      func(int) { calls++ },
		ProgressEvery: 10,
	})
	if !res.Truncated {
		t.Fatalf("got %+v", res)
	}
	if calls != 10 {
		t.Fatalf("progress calls = %d, want 10", calls)
	}
}
