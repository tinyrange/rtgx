package rtgunit

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestMarshalRoundTrip(t *testing.T) {
	program := testProgram(t, []byte(`package main

const answer = 42

func appMain() int { return answer }
`))
	data, err := Marshal(program)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	decoded, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded.Package != program.Package ||
		!bytes.Equal(decoded.Text, program.Text) ||
		!bytes.Equal(decoded.Tokens, program.Tokens) ||
		len(decoded.Decls) != len(program.Decls) ||
		len(decoded.Funcs) != len(program.Funcs) {
		t.Fatalf("decoded program = %#v, want %#v", decoded, program)
	}
	if len(decoded.Decls) != 1 || decoded.Decls[0].Kind != rtgTokConst {
		t.Fatalf("decls = %#v, want one const decl", decoded.Decls)
	}
	if len(decoded.Funcs) != 1 {
		t.Fatalf("funcs = %#v, want one function", decoded.Funcs)
	}
	if !bytes.Contains(Source(decoded), []byte("func appMain")) {
		t.Fatalf("Source() did not include function text: %q", string(Source(decoded)))
	}
}

func TestConvertFiles(t *testing.T) {
	src := []byte(`package main

const answer = 42

// rtg:linkstatic libc,puts
func puts(s string) int { return 0 }

func appMain() int { return answer }
`)
	program := testProgram(t, src)
	if program.Package != "main" {
		t.Fatalf("package = %q, want main", program.Package)
	}
	if len(program.Decls) != 1 {
		t.Fatalf("decl count = %d, want 1", len(program.Decls))
	}
	if len(program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(program.Funcs))
	}
	if len(program.Tokens)%tokenStride != 0 {
		t.Fatalf("token table size = %d, want multiple of %d", len(program.Tokens), tokenStride)
	}
	if !bytes.Contains(program.Text, []byte("// rtg:linkstatic libc,puts")) {
		t.Fatalf("linkstatic directive was not preserved: %q", string(program.Text))
	}
	data, err := Marshal(program)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if !bytes.HasPrefix(data, []byte(Magic)) {
		t.Fatalf("unit missing magic: %q", data[:4])
	}
}

func testProgram(t *testing.T, src []byte) Program {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}
	program, err := ConvertFiles([]string{path})
	if err != nil {
		t.Fatalf("ConvertFiles failed: %v", err)
	}
	return program
}
