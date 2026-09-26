package peterson

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
)

func TestPetersonIsSafe(t *testing.T) {
	dex.Check(t, Spec(true))
}

func TestTakingTheTurnForYourselfBreaksIt(t *testing.T) {
	res := dex.Run(Spec(false))
	if res.Violation == nil {
		t.Fatal("expected the bug to be found")
	}
	last := res.Violation.Path[len(res.Violation.Path)-1].State
	if !(last.PC[0] == critical && last.PC[1] == critical) {
		t.Fatalf("unexpected final state: %+v", last)
	}
}
