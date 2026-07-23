package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateLinksBindingsAndPrintsExpressionsWithoutStatementReplay(t *testing.T) {
	var state State
	binding := state.Prepare("answer := 40")
	if binding.Kind != SubmissionStatement ||
		!strings.Contains(string(binding.First), "var renvo_repl_storage_0 = 40") ||
		!strings.Contains(string(binding.First), "var renvo_repl_value_0 = &renvo_repl_storage_0") {
		t.Fatalf("binding submission = %#v\n%s", binding, binding.First)
	}
	state.Accept(binding, 0)

	expression := state.Prepare("answer + 2")
	source := string(expression.First)
	if expression.Kind != SubmissionExpression ||
		strings.Contains(source, "answer := 40\n") ||
		!strings.Contains(source, "var renvo_repl_storage_0 = 40") ||
		!strings.Contains(source, "renvoreplfmt.Println((*renvo_repl_value_0) + 2)") ||
		!strings.Contains(source, `import renvoreplfmt "fmt"`) {
		t.Fatalf("expression submission = %#v\n%s", expression, source)
	}
}

func TestStateEmitsOnlyCurrentStatement(t *testing.T) {
	var state State
	binding := state.Prepare("counter := 0")
	state.Accept(binding, 0)
	increment := state.Prepare("counter++")
	if strings.Count(string(increment.First), "(*renvo_repl_value_0)++") != 1 {
		t.Fatalf("increment generation replays statements:\n%s", increment.First)
	}
	state.Accept(increment, 0)
	next := string(state.Prepare("counter").First)
	if strings.Contains(next, "++") {
		t.Fatalf("later generation replays increment:\n%s", next)
	}
}

func TestVarDeclarationsUsePersistentCells(t *testing.T) {
	var state State
	typed := state.Prepare("var count int = 7")
	if typed.Kind != SubmissionStatement ||
		!strings.Contains(string(typed.First), "var renvo_repl_storage_0 int = 7") {
		t.Fatalf("typed var submission = %#v\n%s", typed, typed.First)
	}
	state.Accept(typed, 0)
	zero := state.Prepare("var pending []int")
	if zero.Kind != SubmissionStatement ||
		!strings.Contains(string(zero.First), "var renvo_repl_storage_1 []int\n") {
		t.Fatalf("zero var submission = %#v\n%s", zero, zero.First)
	}
	state.Accept(zero, 0)
	source := string(state.Prepare("count + len(pending)").First)
	if !strings.Contains(source, "renvoreplfmt.Println((*renvo_repl_value_0) + len((*renvo_repl_value_1)))") {
		t.Fatalf("persistent var expression:\n%s", source)
	}
}

func TestFunctionParametersShadowSessionBindings(t *testing.T) {
	var state State
	binding := state.Prepare("value := 40")
	state.Accept(binding, 0)
	declaration := state.Prepare("func add(value int) int { return value + 2 }")
	source := string(declaration.First)
	if strings.Contains(source, "func add((*renvo_repl_value_0)") ||
		strings.Contains(source, "return (*renvo_repl_value_0)") ||
		!strings.Contains(source, "func add(value int) int { return value + 2 }") {
		t.Fatalf("parameter shadowing was rewritten:\n%s", source)
	}
	closure := string(state.Prepare("next := func() int { value++; return value }").First)
	if !strings.Contains(closure, "func() int { (*renvo_repl_value_0)++; return (*renvo_repl_value_0) }") {
		t.Fatalf("outer binding was not captured:\n%s", closure)
	}
}

func TestStateRetainsDeclarationsAtPackageScope(t *testing.T) {
	var state State
	declaration := state.Prepare("func twice(v int) int { return v * 2 }")
	if declaration.Kind != SubmissionDeclaration {
		t.Fatalf("declaration kind = %d", declaration.Kind)
	}
	state.Accept(declaration, 0)
	source := string(state.Prepare("twice(21)").First)
	declarationAt := strings.Index(source, "func twice")
	mainAt := strings.Index(source, "func main")
	if declarationAt < 0 || mainAt < 0 || declarationAt > mainAt ||
		!strings.Contains(source, "renvoreplfmt.Println(twice(21))") {
		t.Fatalf("declaration source:\n%s", source)
	}
}

func TestImportsActivateOnlyWhenReferenced(t *testing.T) {
	var state State
	imports := state.Prepare("import f \"fmt\"")
	if imports.Kind != SubmissionImport {
		t.Fatalf("import submission = %#v", imports)
	}
	if !strings.Contains(string(imports.First), `import _ "fmt"`) {
		t.Fatalf("import validation source:\n%s", imports.First)
	}
	state.Accept(imports, 0)
	plain := string(state.Prepare("1 + 1").First)
	if strings.Contains(plain, `import f "fmt"`) {
		t.Fatalf("unused import was emitted:\n%s", plain)
	}
	using := string(state.Prepare(`f.Sprintf("%d", 42)`).First)
	if !strings.Contains(using, `import f "fmt"`) {
		t.Fatalf("used import was omitted:\n%s", using)
	}
}

func TestGroupedImportsAreSplitForIndependentUse(t *testing.T) {
	var state State
	imports := state.Prepare("import (\n\"fmt\"\n\"strings\"\n)")
	if imports.Kind != SubmissionImport {
		t.Fatalf("grouped import submission = %#v", imports)
	}
	state.Accept(imports, 0)
	source := string(state.Prepare(`strings.Count("a-a", "-")`).First)
	if strings.Contains(source, `import "fmt"`) ||
		!strings.Contains(source, `import "strings"`) {
		t.Fatalf("filtered grouped imports:\n%s", source)
	}
}

func TestKnownOutputCallsCompileAsStatements(t *testing.T) {
	var state State
	for _, input := range []string{`print("hello")`, `println("hello")`, `fmt.Println("hello")`} {
		prepared := state.Prepare(input)
		if prepared.Kind != SubmissionStatement || len(prepared.Second) != 0 {
			t.Fatalf("Prepare(%q) = %#v", input, prepared)
		}
	}
}

func TestInputCompleteHandlesMultilineConstructs(t *testing.T) {
	for _, input := range []string{
		"func value() int {",
		"items := []int{1, 2,",
		"value := `line one",
		"/* comment",
	} {
		if InputComplete(input) {
			t.Fatalf("InputComplete(%q) = true", input)
		}
	}
	for _, input := range []string{
		"func value() int {\nreturn 42\n}",
		"items := []int{1, 2,\n3}",
		"value := `line one\nline two`",
		"/* comment */ 42",
		`"{"`,
	} {
		if !InputComplete(input) {
			t.Fatalf("InputComplete(%q) = false", input)
		}
	}
}

func TestResetClearsLinkedSourceAndHistory(t *testing.T) {
	var state State
	binding := state.Prepare("value := 1")
	state.Accept(binding, 0)
	if len(state.History()) != 1 {
		t.Fatal("accepted submission was not recorded")
	}
	state.Reset()
	if len(state.History()) != 0 || strings.Contains(state.Source(), "renvo_repl_value_") {
		t.Fatalf("reset state:\n%s", state.Source())
	}
}

func TestSemanticCompletionUsesLiveStateAndImportedPackages(t *testing.T) {
	root := replTestRoot(t)
	env := []string{"PWD=" + t.TempDir(), "RENVO_STDROOT=" + filepath.Join(root, "std")}
	var state State

	binding := state.Prepare("answer := 40")
	state.Accept(binding, 0)
	if !hasCompletion(state.Complete("ans", 3, env), "answer") {
		source, query := state.completionSource("ans", 3)
		t.Fatalf("session binding completion = %#v query=%d\n%s", state.Complete("ans", 3, env), query, source)
	}

	imp := state.Prepare(`import "strings"`)
	state.Accept(imp, 0)
	packageItems := state.Complete("strings.Co", len("strings.Co"), env)
	if !hasCompletion(packageItems, "Count") || completionNamed(packageItems, "Count").Signature == "" {
		t.Fatalf("package completion = %#v", packageItems)
	}

	functionInput := "func choose(value int) int {\nreturn val"
	if !hasCompletion(state.Complete(functionInput, len(functionInput), env), "value") {
		t.Fatalf("function-local completion = %#v", state.Complete(functionInput, len(functionInput), env))
	}

	importItems := state.Complete(`import "`, len(`import "`), env)
	fmtItem := completionNamed(importItems, "fmt")
	if fmtItem.Name == "" || fmtItem.Insert != `fmt"` || fmtItem.ReplaceStart != len(`import "`) {
		t.Fatalf("import completion = %#v", importItems)
	}

	call := `strings.Count("a-a", `
	help := state.Signature(call, len(call), env)
	if !help.Ok || help.ActiveParameter != 1 || !strings.Contains(help.Label, "Count") ||
		len(help.Parameters) != 2 || help.Parameters[1].Name == "" {
		t.Fatalf("signature help = %#v", help)
	}
}

func hasCompletion(items []Completion, name string) bool {
	for i := 0; i < len(items); i++ {
		if items[i].Name == name {
			return true
		}
	}
	return false
}

func completionNamed(items []Completion, name string) Completion {
	for i := 0; i < len(items); i++ {
		if items[i].Name == name {
			return items[i]
		}
	}
	return Completion{}
}

func replTestRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}
