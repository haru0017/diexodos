package dex_test

import (
	"fmt"

	"github.com/haru0017/diexodos/dex"
)

func ExampleRun() {
	spec := dex.Spec[int]{
		Init: []int{0},
		Actions: []dex.Action[int]{
			dex.Act("inc", func(s int) bool { return s < 5 }, func(s int) int { return s + 1 }),
		},
		Invariants: []dex.Invariant[int]{
			dex.Inv("small", func(s int) bool { return s < 4 }),
		},
	}
	res := dex.Run(spec)
	fmt.Println(res.Violation.Invariant, len(res.Violation.Path))
	// Output: small 5
}
