package link

import (
	"bytes"
	"testing"

	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/unit"
	"j5.nz/rtg/rtgunit"
)

func TestLinkBuildCombinesPackageUnits(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte(`package lib

const answer = 42

func Value() int { return answer }
`)},
	})
	linked := LinkBuild(result)
	if !linked.Ok {
		t.Fatalf("LinkBuild failed: err=%d pkg=%d", linked.Error, linked.ErrorPackage)
	}
	if linked.Program.Package != "main" {
		t.Fatalf("linked package = %q, want main", linked.Program.Package)
	}
	if !bytes.Contains(linked.Program.Text, []byte("package lib")) || !bytes.Contains(linked.Program.Text, []byte("package main")) {
		t.Fatalf("linked text missing package sources: %q", string(linked.Program.Text))
	}
	decoded, err := rtgunit.Unmarshal(linked.Data)
	if err != nil {
		t.Fatalf("linked unit did not decode: %v", err)
	}
	if decoded.Package != "main" {
		t.Fatalf("decoded package = %q, want main", decoded.Package)
	}
	if len(decoded.Decls) != 1 {
		t.Fatalf("decoded decl count = %d, want 1", len(decoded.Decls))
	}
	if len(decoded.Funcs) != 2 {
		t.Fatalf("decoded func count = %d, want 2", len(decoded.Funcs))
	}
	if string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]) != "answer" {
		t.Fatalf("decl name = %q", string(decoded.Text[decoded.Decls[0].NameStart:decoded.Decls[0].NameEnd]))
	}
	if functionName(decoded, decoded.Funcs[0]) != "Value" || functionName(decoded, decoded.Funcs[1]) != "appMain" {
		t.Fatalf("function names = %q %q", functionName(decoded, decoded.Funcs[0]), functionName(decoded, decoded.Funcs[1]))
	}
}

func TestLinkUnitsAdjustsTokenOffsets(t *testing.T) {
	result := buildFromFiles(t, []load.SourceFile{
		{Path: "/repo/case/go.mod", Src: []byte("module example.com/case\n")},
		{Path: "/repo/case/cmd/app/main.go", Src: []byte(`package main

import "example.com/case/pkg/lib"

func appMain() int { return lib.Value() }
`)},
		{Path: "/repo/case/pkg/lib/lib.go", Src: []byte("package lib\n\nfunc Value() int { return 1 }\n")},
	})
	program, ok := LinkUnits(result.Units, result.Root)
	if !ok {
		t.Fatal("LinkUnits failed")
	}
	if len(program.Funcs) != 2 {
		t.Fatalf("func count = %d, want 2", len(program.Funcs))
	}
	first := program.Funcs[0]
	second := program.Funcs[1]
	if string(program.Text[first.NameStart:first.NameEnd]) != "Value" {
		t.Fatalf("first func name = %q", string(program.Text[first.NameStart:first.NameEnd]))
	}
	if string(program.Text[second.NameStart:second.NameEnd]) != "appMain" {
		t.Fatalf("second func name = %q", string(program.Text[second.NameStart:second.NameEnd]))
	}
	if program.Tokens[first.NameTok].Start != first.NameStart {
		t.Fatalf("first func name token start = %d, want %d", program.Tokens[first.NameTok].Start, first.NameStart)
	}
	if program.Tokens[second.NameTok].Start != second.NameStart {
		t.Fatalf("second func name token start = %d, want %d", program.Tokens[second.NameTok].Start, second.NameStart)
	}
	if program.Tokens[second.NameTok].Line <= program.Tokens[first.NameTok].Line {
		t.Fatalf("line offsets were not adjusted: first=%d second=%d", program.Tokens[first.NameTok].Line, program.Tokens[second.NameTok].Line)
	}
	if program.Tokens[len(program.Tokens)-1].Kind != unit.TokenEOF {
		t.Fatalf("last token kind = %d, want EOF", program.Tokens[len(program.Tokens)-1].Kind)
	}
}

func TestLinkBuildRejectsInvalidInput(t *testing.T) {
	badBuild := build.Result{Ok: false, ErrorPackage: 7}
	linked := LinkBuild(badBuild)
	if linked.Ok || linked.Error != LinkErrBuild || linked.ErrorPackage != 7 {
		t.Fatalf("bad build link result = %#v", linked)
	}

	linked = LinkBuild(build.Result{Ok: true, Root: -1})
	if linked.Ok || linked.Error != LinkErrRoot {
		t.Fatalf("bad root link result = %#v", linked)
	}

	if _, ok := LinkUnits(nil, 0); ok {
		t.Fatal("LinkUnits accepted empty unit list")
	}
}

func buildFromFiles(t *testing.T, files []load.SourceFile) build.Result {
	t.Helper()
	workspace := load.LoadWorkspace("/repo/case", "/std", "./cmd/app", files)
	if !workspace.Ok {
		t.Fatalf("LoadWorkspace failed: err=%d file=%d", workspace.Error, workspace.ErrorFile)
	}
	result := build.BuildUnits(workspace.Graph)
	if !result.Ok {
		t.Fatalf("BuildUnits failed: err=%d pkg=%d file=%d tok=%d", result.Error, result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	}
	return result
}

func functionName(program rtgunit.Program, fn rtgunit.Func) string {
	if fn.NameStart < 0 || fn.NameEnd < fn.NameStart || fn.NameEnd > len(program.Text) {
		return ""
	}
	return string(program.Text[fn.NameStart:fn.NameEnd])
}
