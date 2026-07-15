//go:build !rtg

package unit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestCoreUnitModelStaysOutOfBuildTaggedForks(t *testing.T) {
	shared := declaredNames(t, "model_shared.go")
	for _, name := range []string{"Token", "Decl", "Func", "CallUnknown", "RefUnknown", "TypeRefUnknown", "appendNode", "appendUint16", "appendUint32", "appendVarint"} {
		if !shared[name] {
			t.Fatalf("shared unit model is missing %s", name)
		}
	}
	for _, path := range []string{"unit.go", "unit_full.go"} {
		fork := declaredNames(t, path)
		for name := range shared {
			if fork[name] {
				t.Fatalf("%s redeclares shared unit model name %s", path, name)
			}
		}
	}
}

func declaredNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[string]bool)
	for _, decl := range file.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			names[value.Name.Name] = true
		case *ast.GenDecl:
			for _, spec := range value.Specs {
				switch item := spec.(type) {
				case *ast.TypeSpec:
					names[item.Name.Name] = true
				case *ast.ValueSpec:
					for _, name := range item.Names {
						names[name.Name] = true
					}
				}
			}
		}
	}
	return names
}
