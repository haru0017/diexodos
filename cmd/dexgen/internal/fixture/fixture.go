// Package fixture exercises dexgen with nested and non-comparable state.
package fixture

//go:generate go run github.com/haru0017/diexodos/cmd/dexgen -type State
//go:generate go run github.com/haru0017/diexodos/cmd/dexgen -type Entry

type Kind string

type Entry struct {
	Kind Kind
	N    int
}

type State struct {
	Round int
	Done  bool
	Names []string
	Log   []Entry
	Flags [2]bool
}
