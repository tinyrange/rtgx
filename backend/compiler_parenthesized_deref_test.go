package main

import "testing"

func TestParenthesizedPointerToSliceOperations(t *testing.T) {
	resetRuntime()
	source := []byte("package main\nvar storage = []int{1}\nvar value = &storage\nfunc appMain() int {\n(*value) = append((*value), 7)\nreturn (*value)[0] + (*value)[1] - len((*value))\n}\n")
	if _, ok := RenvoCompileSourceToBytesStrip(source, "linux/amd64", true); !ok {
		t.Fatal("parenthesized pointer-to-slice operations did not compile")
	}
}

func TestParenthesizedPointerCompoundAssignment(t *testing.T) {
	resetRuntime()
	source := []byte("package main\nvar storage = 40\nvar value = &storage\nfunc appMain() int {\n(*value) += 2\nreturn (*value) - 42\n}\n")
	if _, ok := RenvoCompileSourceToBytesStrip(source, "linux/amd64", true); !ok {
		t.Fatal("parenthesized pointer compound assignment did not compile")
	}
}

func TestParenthesizedPointerToFunctionCall(t *testing.T) {
	resetRuntime()
	source := []byte("package main\nvar storage = func() int { return 42 }\nvar value = &storage\nfunc appMain() int {\nreturn (*value)() - 42\n}\n")
	if _, ok := RenvoCompileSourceToBytesStrip(source, "linux/amd64", true); !ok {
		t.Fatal("parenthesized pointer-to-function call did not compile")
	}
}

func TestReplFunctionValueTagsSurviveFunctionTableRenumbering(t *testing.T) {
	first := replLiteralTag(t, "package main\nvar renvo_repl_storage_0 = func() int { return 42 }\nvar renvo_repl_value_0 = &renvo_repl_storage_0\nfunc appMain() int { return 0 }\n")
	second := replLiteralTag(t, "package main\nfunc unrelated() int { return 7 }\nvar renvo_repl_storage_0 = func() int { return 42 }\nvar renvo_repl_value_0 = &renvo_repl_storage_0\nfunc appMain() int { return 0 }\n")
	if first != second {
		t.Fatalf("persistent closure tag changed from %d to %d after function-table renumbering", first, second)
	}
}

func TestReplNamedFunctionTagsSurviveFunctionTableRenumbering(t *testing.T) {
	first := replNamedTag(t, "package main\nfunc stable() int { return 42 }\nvar renvo_repl_storage_0 = stable\nvar renvo_repl_value_0 = &renvo_repl_storage_0\nfunc appMain() int { return 0 }\n")
	second := replNamedTag(t, "package main\nfunc unrelated() int { return 7 }\nfunc stable() int { return 42 }\nvar renvo_repl_storage_0 = stable\nvar renvo_repl_value_0 = &renvo_repl_storage_0\nfunc appMain() int { return 0 }\n")
	if first != second {
		t.Fatalf("persistent named-function tag changed from %d to %d after function-table renumbering", first, second)
	}
}

func replLiteralTag(t *testing.T, sourceText string) int {
	t.Helper()
	source := []byte(sourceText)
	program := renvoParseProgram(source)
	if !program.ok {
		t.Fatal("source did not parse")
	}
	path := []byte("renvo.dev/repl")
	pathA, pathB := renvoObjectHashRange(1879, 3761, path, 0, len(path))
	program.packageTable = &renvoPackageTable{items: []renvoPackageInfo{{
		pathKeyA: pathA, pathKeyB: pathB, textEnd: len(source),
		funcEnd: len(program.funcs), declEnd: len(program.decls), tokenEnd: renvoTokCount(&program),
	}}}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	gen := renvoLinearGen{prog: &program, meta: &meta, replRestoreOffsets: []int{0}}
	for i := 0; i < len(meta.funcs); i++ {
		if meta.funcs[i].literalTok > 0 {
			return renvoReplFunctionValueTag(&gen, i)
		}
	}
	t.Fatal("function literal was not discovered")
	return 0
}

func replNamedTag(t *testing.T, sourceText string) int {
	t.Helper()
	source := []byte(sourceText)
	program := renvoParseProgram(source)
	if !program.ok {
		t.Fatal("source did not parse")
	}
	path := []byte("renvo.dev/repl")
	pathA, pathB := renvoObjectHashRange(1879, 3761, path, 0, len(path))
	program.packageTable = &renvoPackageTable{items: []renvoPackageInfo{{
		pathKeyA: pathA, pathKeyB: pathB, textEnd: len(source),
		funcEnd: len(program.funcs), declEnd: len(program.decls), tokenEnd: renvoTokCount(&program),
	}}}
	var meta renvoMeta
	renvoBuildMetaInto(&program, &meta)
	gen := renvoLinearGen{prog: &program, meta: &meta, replRestoreOffsets: []int{0}}
	for i := 0; i < len(meta.funcs); i++ {
		if renvoBytesEqualText(source, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "stable") {
			return renvoReplFunctionValueTag(&gen, i)
		}
	}
	t.Fatal("named function was not discovered")
	return 0
}
