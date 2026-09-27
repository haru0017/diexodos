// Command dexgen generates DexKey methods: injective canonical encodings of
// struct types, usable as dex.RunKeyed keys for states that are not
// comparable. Run it from the package directory:
//
//	//go:generate go run github.com/haru0017/diexodos/cmd/dexgen -type State
//
// The output file is named after the first type, state_dexkey.go here, in the
// manner of stringer. Each generated file is self contained, so separate
// invocations for different types in one package coexist.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.SetFlags(0)
	typeNames := flag.String("type", "", "comma separated struct type names")
	out := flag.String("o", "", "output file name; defaults to <type>_dexkey.go")
	flag.Parse()
	if *typeNames == "" {
		log.Fatal("dexgen: -type is required")
	}
	types := strings.Split(*typeNames, ",")
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	name := *out
	if name == "" {
		name = strings.ToLower(types[0]) + "_dexkey.go"
	}

	src, err := generate(dir, types)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), src, 0o644); err != nil {
		log.Fatal(err)
	}
}
