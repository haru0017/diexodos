package report

import (
	"os"
	"strings"
	"testing"

	"github.com/haru0017/diexodos/dex"
)

// A minimal livelock: retrying forever without ever finishing.
type pump struct {
	Phase int
	Done  bool
}

func pumpSpec() dex.Spec[pump] {
	return dex.Spec[pump]{
		Init: []pump{{}},
		Actions: []dex.Action[pump]{
			dex.Act("dispatch",
				func(s pump) bool { return s.Phase == 0 },
				func(s pump) pump { s.Phase = 1; return s }),
			dex.Act("rejected",
				func(s pump) bool { return s.Phase == 1 },
				func(s pump) pump { s.Phase = 0; return s }),
			dex.Act("completed",
				func(s pump) bool { return s.Phase == 1 },
				func(s pump) pump { s.Done = true; s.Phase = 2; return s }),
		},
		Liveness: []dex.Liveness[pump]{
			dex.EventuallyAlways("the job finishes", func(s pump) bool { return s.Done }),
		},
	}
}

func TestLassoMarkdown(t *testing.T) {
	res := dex.Run(pumpSpec())
	if res.Lasso == nil {
		t.Fatal("expected a lasso")
	}
	md := Markdown(res)
	for _, want := range []string{
		"## liveness \"the job finishes\" violated",
		"```mermaid",
		"==>", // the repeating part is thick
		"↺",
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown misses %q:\n%s", want, md)
		}
	}
}

func TestViolationMarkdown(t *testing.T) {
	spec := pumpSpec()
	spec.Liveness = nil
	spec.Invariants = []dex.Invariant[pump]{
		dex.Inv("the job never finishes", func(s pump) bool { return !s.Done }),
	}
	res := dex.Run(spec)
	if res.Violation == nil {
		t.Fatal("expected a violation")
	}
	md := Markdown(res)
	for _, want := range []string{
		"## invariant \"the job never finishes\" violated",
		"style s", // the violating state is highlighted
	} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown misses %q:\n%s", want, md)
		}
	}
}

// Two states that differ only in a character escaping rewrites must stay two
// nodes: identity comes from the raw state, escaping is display only.
func TestEscapedStatesStayDistinct(t *testing.T) {
	type s struct{ V string }
	md := Markdown(dex.Result[s]{
		Violation: &dex.Violation[s]{
			Invariant: "quotes stay apart",
			Path: []dex.Step[s]{
				{State: s{V: `a"b`}},
				{Action: "swap", State: s{V: "a'b"}},
			},
		},
	})
	for _, want := range []string{"s0", "s1", `s0 -->|"swap"| s1`} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown misses %q:\n%s", want, md)
		}
	}
}

func TestGreenRendersNothing(t *testing.T) {
	spec := pumpSpec()
	spec.Liveness = nil
	if md := Markdown(dex.Run(spec)); md != "" {
		t.Fatalf("expected no markdown for a passing check, got:\n%s", md)
	}
	Save(t, dex.Run(spec))
	if _, err := os.Stat(t.Name() + ".md"); !os.IsNotExist(err) {
		t.Fatalf("expected no file for a passing check, stat: %v", err)
	}
}
