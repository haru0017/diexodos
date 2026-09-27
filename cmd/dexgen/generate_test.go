package main

import (
	"os"
	"strings"
	"testing"
)

func TestGeneratedFileIsUpToDate(t *testing.T) {
	src, err := generate("internal/fixture", []string{"State"})
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("internal/fixture/state_dexkey.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != string(committed) {
		t.Fatal("internal/fixture/state_dexkey.go is stale; rerun go generate")
	}
}

// Two invocations in one package produce self-contained files that coexist;
// the fixture package compiling with both is the real assertion here.
func TestSecondTypeFileIsUpToDate(t *testing.T) {
	src, err := generate("internal/fixture", []string{"Entry"})
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile("internal/fixture/entry_dexkey.go")
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != string(committed) {
		t.Fatal("internal/fixture/entry_dexkey.go is stale; rerun go generate")
	}
}

func TestUnknownTypeFails(t *testing.T) {
	_, err := generate("internal/fixture", []string{"Nope"})
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("got %v", err)
	}
}

func TestUnsupportedFieldFails(t *testing.T) {
	_, err := generate("internal/badfixture", []string{"Bad"})
	if err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("got %v", err)
	}
}
