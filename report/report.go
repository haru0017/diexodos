// Package report renders counterexamples as Markdown with a mermaid diagram,
// meant to be committed next to the test that pins them.
package report

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/haru0017/diexodos/dex"
)

// Markdown renders the counterexample in res, or "" when the check passed.
func Markdown[S any](res dex.Result[S]) string {
	switch {
	case res.Violation != nil:
		v := res.Violation
		return render(fmt.Sprintf("invariant %q violated", v.Invariant), res.States, v.Path, nil)
	case res.Lasso != nil:
		l := res.Lasso
		return render(fmt.Sprintf("liveness %q violated", l.Property), res.States, l.Prefix, l.Cycle)
	}
	return ""
}

// Save writes the counterexample beside the caller's test file as
// <TestName>.md. Green results write nothing. The output is deterministic,
// so an unchanged counterexample leaves the working tree clean.
func Save[S any](t testing.TB, res dex.Result[S]) {
	t.Helper()
	md := Markdown(res)
	if md == "" {
		return
	}
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("report: cannot locate the caller's test file")
	}
	name := filename(t.Name()) + ".md"
	path := filepath.Join(filepath.Dir(file), name)
	if err := os.WriteFile(path, []byte("# "+t.Name()+"\n\n"+md), 0o644); err != nil {
		t.Fatalf("report: %v", err)
	}
	t.Logf("counterexample written to %s", name)
}

// render draws one node per distinct state, so a lasso closes into a loop by
// itself: its last step revisits an earlier state. Cycle edges are thick, and
// an edge walked by both the prefix and the cycle is drawn once, thick.
func render[S any](title string, states int, prefix, cycle []dex.Step[S]) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\nExplored %d states.\n\n```mermaid\nflowchart TD\n", title, states)

	// Node identity is the raw state text; escaping is display only, so two
	// states that differ in an escaped character cannot merge into one node.
	ids := map[string]string{}
	node := func(s S) string {
		raw := fmt.Sprintf("%+v", s)
		if id, seen := ids[raw]; seen {
			return id
		}
		id := fmt.Sprintf("s%d", len(ids))
		ids[raw] = id
		fmt.Fprintf(&b, "    %s[\"%s\"]\n", id, stateLabel(s))
		return id
	}

	steps := append(append([]dex.Step[S]{}, prefix...), cycle...)
	type edge struct{ from, action, to string }
	order := []edge{}
	thick := map[edge]bool{}
	prev := ""
	for i, s := range steps {
		cur := node(s.State)
		if i > 0 {
			e := edge{prev, s.Action, cur}
			order = append(order, e)
			if i >= len(prefix) {
				thick[e] = true // the part that repeats forever
			}
		}
		prev = cur
	}
	seen := map[edge]bool{}
	for _, e := range order {
		if seen[e] {
			continue
		}
		seen[e] = true
		arrow := "-->"
		if thick[e] {
			arrow = "==>"
		}
		fmt.Fprintf(&b, "    %s %s|\"%s\"| %s\n", e.from, arrow, escape(e.action), e.to)
	}
	if len(cycle) == 0 && len(steps) > 0 {
		fmt.Fprintf(&b, "    style %s stroke:#d33,stroke-width:2px\n", prev)
	}
	b.WriteString("```\n\n")

	b.WriteString("| # | action | state |\n|--:|---|---|\n")
	for i, s := range steps {
		action := s.Action
		if action == "" {
			action = "(init)"
		}
		mark := ""
		if len(cycle) > 0 && i >= len(prefix) {
			mark = " ↺"
		}
		fmt.Fprintf(&b, "| %d%s | %s | `%s` |\n", i, mark, escape(action), stateLabel(s.State))
	}
	if len(cycle) > 0 {
		b.WriteString("\nThe ↺ steps repeat forever.\n")
	}
	return b.String()
}

func stateLabel[S any](s S) string {
	return escape(strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%+v", s), "{"), "}"))
}

// escape keeps a label from breaking out of its container: quotes end a
// mermaid label, pipes end a mermaid edge label and a Markdown table cell,
// and a newline ends both.
func escape(s string) string {
	return strings.NewReplacer("\"", "'", "|", "/", "\n", " ").Replace(s)
}

// filename maps a test name to a safe file name, one character class rather
// than a list of known offenders: subtest names can hold almost anything.
func filename(test string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-':
			return r
		default:
			return '_'
		}
	}, test)
}
