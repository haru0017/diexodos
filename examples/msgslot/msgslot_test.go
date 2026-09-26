package msgslot

import (
	"testing"

	"github.com/haru0017/diexodos/actor"
)

func TestSeparateCheckAndTakeRace(t *testing.T) {
	res := actor.Run(System(false))
	if res.Violation == nil {
		t.Fatal("expected the race to be found")
	}
}

func TestTryTakeIsSafe(t *testing.T) {
	s := System(true)
	s.EventuallyAlways("someone holds the slot", func(w actor.Snapshot) bool {
		return actor.Machine[Runner](w, "r1").Holding || actor.Machine[Runner](w, "r2").Holding
	})
	actor.Check(t, s)
}
