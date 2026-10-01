package philosophers

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
	"github.com/haru0017/diexodos/report"
)

func withLiveness(cfg Config) dex.Spec[State] {
	spec := Spec(cfg)
	spec.Liveness = []dex.Liveness[State]{EveryoneEats()}
	return spec
}

// Everyone takes a left fork and waits for the right one forever: the
// deadlock shows up as the jammed state stuttering.
func TestNaiveProtocolDeadlocks(t *testing.T) {
	res := dex.Run(withLiveness(Config{}))
	if res.Lasso == nil {
		t.Fatal("expected the deadlock")
	}
	report.Save(t, res)
}

// Yielding the fork removes the deadlock and replaces it with a livelock:
// everyone can pick up and put back in step, and dinner never happens.
func TestPoliteProtocolLivelocks(t *testing.T) {
	res := dex.Run(withLiveness(Config{Polite: true}))
	if res.Lasso == nil {
		t.Fatal("expected the livelock")
	}
	report.Save(t, res)
}

// One philosopher reaching right first breaks the symmetry: every
// interleaving ends with everyone fed.
func TestAsymmetricProtocolIsSafe(t *testing.T) {
	res := dex.Run(withLiveness(Config{Asymmetric: true}))
	if res.Violation != nil || res.Lasso != nil {
		t.Fatalf("expected every interleaving to feed everyone:\n%v%v", res.Violation, res.Lasso)
	}
}
