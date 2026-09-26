package dex

import (
	"strings"
	"testing"
)

func counterSpec(bound int) Spec[int] {
	return Spec[int]{
		Init: []int{0},
		Actions: []Action[int]{
			Act("inc", func(s int) bool { return s < 10 }, func(s int) int { return s + 1 }),
		},
		Invariants: []Invariant[int]{
			Inv("bound", func(s int) bool { return s <= bound }),
		},
	}
}

func TestCheckPasses(t *testing.T) {
	res := Check(t, counterSpec(10))
	if res.States != 11 {
		t.Fatalf("states = %d, want 11", res.States)
	}
}

func TestRunFindsViolation(t *testing.T) {
	res := Run(counterSpec(4))
	if res.Violation == nil {
		t.Fatal("expected violation")
	}
	if got := len(res.Violation.Path); got != 6 {
		t.Fatalf("path length = %d, want 6", got)
	}
	out := res.Violation.String()
	if !strings.Contains(out, `invariant "bound" violated`) || !strings.Contains(out, "(init)") {
		t.Fatalf("unexpected rendering:\n%s", out)
	}
}

func TestNilGuardIsAlwaysEnabled(t *testing.T) {
	spec := Spec[int]{
		Init:       []int{0},
		Actions:    []Action[int]{Act("clamp", nil, func(s int) int { return 1 })},
		Invariants: []Invariant[int]{Inv("small", func(s int) bool { return s < 2 })},
	}
	Check(t, spec)
}

func TestRunReportsTruncation(t *testing.T) {
	spec := Spec[int]{
		Init:    []int{0},
		Actions: []Action[int]{Act("inc", nil, func(s int) int { return s + 1 })},
	}
	res := Run(spec, MaxStates(50))
	if !res.Truncated || res.States != 50 {
		t.Fatalf("got %+v", res)
	}
}
