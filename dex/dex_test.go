package dex

import (
	"fmt"
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

func TestRunKeyedWithSliceState(t *testing.T) {
	// States hold a slice, so identity goes through a canonical key.
	type S struct{ Log []int }
	push := ActN("push", nil, func(s S) []S {
		if len(s.Log) >= 3 {
			return nil
		}
		next := append(append([]int(nil), s.Log...), len(s.Log))
		return []S{{Log: next}}
	})
	spec := Spec[S]{
		Init:       []S{{}},
		Actions:    []Action[S]{push},
		Invariants: []Invariant[S]{Inv("short", func(s S) bool { return len(s.Log) <= 3 })},
	}
	res := RunKeyed(spec, func(s S) string { return fmt.Sprint(s.Log) })
	if res.Violation != nil || res.States != 4 {
		t.Fatalf("got %+v", res)
	}
}

func spinnerSpec(fairFinish bool) Spec[int] {
	finish := Act("finish", func(s int) bool { return s == 0 }, func(int) int { return 1 })
	if fairFinish {
		finish = Fair(finish)
	}
	return Spec[int]{
		Init: []int{0},
		Actions: []Action[int]{
			Act("spin", func(s int) bool { return s == 0 }, func(int) int { return 0 }),
			finish,
		},
		Liveness: []Liveness[int]{EventuallyAlways("done", func(s int) bool { return s == 1 })},
	}
}

func TestLivenessNeedsFairness(t *testing.T) {
	res := Run(spinnerSpec(false))
	if res.Lasso == nil {
		t.Fatal("expected a lasso without fairness")
	}
	out := res.Lasso.String()
	if !strings.Contains(out, `liveness "done" violated`) || !strings.Contains(out, "cycle repeats forever") {
		t.Fatalf("unexpected rendering:\n%s", out)
	}

	Check(t, spinnerSpec(true))
}

func TestDetectDeadlocks(t *testing.T) {
	spec := counterSpec(10)
	res := Run(spec, DetectDeadlocks())
	if res.Violation == nil || res.Violation.Invariant != "deadlock" {
		t.Fatalf("got %+v", res)
	}
}
