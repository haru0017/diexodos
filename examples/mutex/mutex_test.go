package mutex

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
)

func TestCheckThenTakeViolatesMutex(t *testing.T) {
	res := dex.Run(Spec(false))
	if res.Violation == nil {
		t.Fatal("expected the race to be found")
	}
	last := res.Violation.Path[len(res.Violation.Path)-1].State
	if !(last.Holds[1] && last.Holds[2]) {
		t.Fatalf("unexpected final state: %+v", last)
	}
	// Shortest interleaving: check(1), check(2), take(1), take(2).
	if got := len(res.Violation.Path); got != 5 {
		t.Fatalf("path length = %d, want 5\n%s", got, res.Violation)
	}
}

func TestAtomicTakeIsSafe(t *testing.T) {
	res := dex.Check(t, Spec(true))
	if res.States != 3 {
		t.Fatalf("states = %d, want 3", res.States)
	}
}
