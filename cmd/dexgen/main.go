// Command dexgen generates DexKey methods: injective canonical encodings of
// struct types, usable as dex.RunKeyed keys for states that are not
// comparable. Run it from the package directory:
//
//	//go:generate go run github.com/haru0017/diexodos/cmd/dexgen -type State
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
	out := flag.String("o", "dexkey_gen.go", "output file name")
	flag.Parse()
	if *typeNames == "" {
		log.Fatal("dexgen: -type is required")
	}
	dir := "."
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}

	src, err := generate(dir, strings.Split(*typeNames, ","))
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, *out), src, 0o644); err != nil {
		log.Fatal(err)
	}
}
