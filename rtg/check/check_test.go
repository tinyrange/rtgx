package check

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/parse"
)

func TestFileRejectsExcludedFeatures(t *testing.T) {
	src := []byte(`package main

type Box[T any] struct { value T }
type Runner interface { Run() }
func useMap(m map[string]int) int { return 0 }
func makeChan(ch chan int) { go makeChan(ch); select {} }
func appMain() int {
	defer print("done")
	fn := func() int { return 1 }
	_ = fn
	return 0
}
`)
	file, err := parse.FileSource("bad.go", src)
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	messages := strings.Join(messages(diags), "\n")
	for _, want := range []string{
		"generics are not supported",
		"interfaces are not supported",
		"maps are not supported",
		"channels are not supported",
		"goroutines are not supported",
		"select statements are not supported",
		"defer is not supported",
		"function values and function types are not supported",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, messages)
		}
	}
}

func TestFileAcceptsSimpleSubsetProgram(t *testing.T) {
	file, err := parse.FileSource("ok.go", []byte(`package main

type box struct { value int }
func appMain() int {
	var b box
	b.value = 7
	return b.value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func messages(diags Diagnostics) []string {
	var out []string
	for _, diag := range diags {
		out = append(out, diag.Message)
	}
	return out
}
