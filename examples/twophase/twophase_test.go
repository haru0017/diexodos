package twophase

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
	"github.com/haru0017/diexodos/report"
)

func TestConsistencyHolds(t *testing.T) {
	for name, cfg := range map[string]Config{"healthy": {}, "crash": {Crash: true}} {
		t.Run(name, func(t *testing.T) {
			spec := Spec(cfg)
			spec.Invariants = []dex.Invariant[State]{Consistent()}
			dex.Check(t, spec)
		})
	}
}

func TestEveryManagerDecides(t *testing.T) {
	spec := Spec(Config{})
	spec.Invariants = []dex.Invariant[State]{Consistent()}
	spec.Liveness = []dex.Liveness[State]{EveryoneDecides()}
	dex.Check(t, spec)
}

// The blocking weakness: the TM stops before deciding and a prepared resource
// manager can neither commit nor abort, forever.
func TestAStoppedManagerBlocksThePrepared(t *testing.T) {
	spec := Spec(Config{Crash: true})
	spec.Liveness = []dex.Liveness[State]{EveryoneDecides()}
	res := dex.Run(spec)
	if res.Lasso == nil {
		t.Fatal("expected the prepared resource manager to block")
	}
	report.Save(t, res)
}
