package semantic

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/internal/load"
)

func TestLowerSingleImplementationInterface(t *testing.T) {
	src := []byte(`package main

type reader interface { Value(delta int) int }
type item struct { value int }

func (value item) Value(delta int) int { return value.value + delta }
func read(value reader) int { return value.Value(1) }

func main() {
	var value reader = item{value: 42}
	asserted, ok := value.(item)
	if !ok || asserted.value != 42 || read(value) != 43 { return }
	switch value.(type) {
	case item:
		print("PASS\n")
	default:
		print("FAIL\n")
	}
}
`)
	module := load.Module{Root: "/repo/case", Path: "example.com/case", Ok: true}
	graph := load.LoadGraph(module, "/std", "/repo/case", "./cmd/app", []load.SourceFile{{Path: "/repo/case/cmd/app/main.go", Src: src}})
	if !graph.Ok {
		t.Fatal("input graph did not load")
	}

	LowerInterfaces(&graph)
	if !graph.Ok || len(graph.Packages) != 1 || len(graph.Packages[0].Files) != 1 {
		t.Fatal("lowered graph is invalid")
	}
	file := graph.Packages[0].Files[0]
	if !file.File.Ok {
		t.Fatal("lowered source did not parse")
	}
	got := string(file.Src)
	for _, want := range []string{
		"type reader = item",
		"func read(value item)",
		"var value item = item",
		"asserted, ok := value, true",
		"switch 1",
		"case 1:",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("lowered source does not contain %q:\n%s", want, got)
		}
	}
}
