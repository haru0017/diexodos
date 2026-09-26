package fixture

import (
	"testing"

	"github.com/haru0017/diexodos/dex"
)

// Length-prefixed strings keep adjacent fields from bleeding into each other.
func TestKeyIsInjectiveAcrossBoundaries(t *testing.T) {
	a := State{Names: []string{"ab", "c"}}
	b := State{Names: []string{"a", "bc"}}
	if a.DexKey() == b.DexKey() {
		t.Fatal("distinct states share a key")
	}
	c := State{Log: []Entry{{Kind: "x", N: 1}, {Kind: "y", N: 2}}}
	d := State{Log: []Entry{{Kind: "y", N: 2}, {Kind: "x", N: 1}}}
	if c.DexKey() == d.DexKey() {
		t.Fatal("order must matter")
	}
	if a.DexKey() != (State{Names: []string{"ab", "c"}}).DexKey() {
		t.Fatal("equal states must share a key")
	}
}

func TestGeneratedKeyDrivesRunKeyed(t *testing.T) {
	grow := dex.ActN("grow", nil, func(s State) []State {
		if len(s.Names) >= 2 {
			return nil
		}
		names := append(append([]string(nil), s.Names...), "n")
		return []State{{Names: names}}
	})
	spec := dex.Spec[State]{
		Init:       []State{{}},
		Actions:    []dex.Action[State]{grow},
		Invariants: []dex.Invariant[State]{dex.Inv("short", func(s State) bool { return len(s.Names) <= 2 })},
	}
	res := dex.RunKeyed(spec, State.DexKey)
	if res.Violation != nil || res.States != 3 {
		t.Fatalf("got %+v", res)
	}
}
