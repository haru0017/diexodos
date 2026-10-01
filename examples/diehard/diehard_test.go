package diehard

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
	"github.com/haru0017/diexodos/report"
)

func TestSolvesThePuzzle(t *testing.T) {
	res := dex.Run(Spec())
	if res.Violation == nil {
		t.Fatal("expected a solution")
	}
	last := res.Violation.Path[len(res.Violation.Path)-1].State
	if last.Big != 4 {
		t.Fatalf("unexpected final state: %+v", last)
	}
	// The known minimal solution takes 6 steps.
	if got := len(res.Violation.Path); got != 7 {
		t.Fatalf("path length = %d, want 7\n%s", got, res.Violation)
	}
	t.Logf("solution:\n%s", res.Violation)
	report.Save(t, res)
}
