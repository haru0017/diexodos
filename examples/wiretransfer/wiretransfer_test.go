package wiretransfer

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
	"github.com/haru0017/diexodos/report"
)

// Both transfers pass the check before either withdraws.
func TestSeparateCheckOverdraws(t *testing.T) {
	res := dex.Run(Spec(Config{}))
	if res.Violation == nil {
		t.Fatal("expected the overdraft")
	}
	report.Save(t, res)
}

func TestAtomicTransferIsSafe(t *testing.T) {
	dex.Check(t, Spec(Config{Atomic: true}))
}
