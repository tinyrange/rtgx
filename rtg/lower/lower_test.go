package lower

import (
	"strings"
	"testing"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/unit"
)

func TestTopLevelDeclKeywordCountHandlesTrailingLineWhitespace(t *testing.T) {
	for _, src := range [][]byte{
		[]byte("package main\n\t"),
		[]byte("package main\nfunc main() {}\n  "),
	} {
		func() {
			defer func() {
				if value := recover(); value != nil {
					t.Fatalf("topLevelDeclKeywordCount panicked for %q: %v", string(src), value)
				}
			}()
			_ = topLevelDeclKeywordCount(src)
		}()
	}
}

func TestPackageBuildsDeclarationUnit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path:     "/tmp/work/pkg/b.go",
				UnitPath: "pkg/b.go",
				Source: []byte(`package pkg

func B() int { return A() }
`),
			},
			{
				Path:     "/tmp/work/pkg/a.go",
				UnitPath: "pkg/a.go",
				Source: []byte(`package pkg

const answer = 7
func A() int { return answer }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if u.ImportPath != "example.com/app/pkg" || u.Package != "pkg" {
		t.Fatalf("unit identity = %#v", u)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[0].Name != "answer" || u.Decls[1].Name != "A" || u.Decls[2].Name != "B" {
		t.Fatalf("decl order = %#v", u.Decls)
	}
	if u.Decls[0].Path != "pkg/a.go" || u.Decls[2].Path != "pkg/b.go" {
		t.Fatalf("decl paths = %#v", u.Decls)
	}
	if len(u.Exports) != 2 || u.Exports[0].Name != "A" || u.Exports[1].Name != "B" {
		t.Fatalf("exports = %#v", u.Exports)
	}
	if u.Decls[0].UnitName != "rtg_example_com_app_pkg_answer" {
		t.Fatalf("const unit name = %q", u.Decls[0].UnitName)
	}
	if !strings.Contains(u.Decls[0].Body, "const rtg_example_com_app_pkg_answer = 7") {
		t.Fatalf("const body was not rewritten: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_A() int { return rtg_example_com_app_pkg_answer }") {
		t.Fatalf("A body was not rewritten: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "func rtg_example_com_app_pkg_B() int { return rtg_example_com_app_pkg_A() }") {
		t.Fatalf("B body was not rewritten: %q", u.Decls[2].Body)
	}
	if strings.Contains(u.Decls[0].Body, "package ") {
		t.Fatalf("decl retained package clause: %q", u.Decls[0].Body)
	}
}

func TestPackageLowersTopLevelNameLists(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "names.go",
				Source: []byte(`package pkg

const A, B = 1, 2
var C, D int
func Use() int { return A + B + C + D }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Exports) != 5 {
		t.Fatalf("exports = %#v, want A, B, C, D, Use", u.Exports)
	}
	if !strings.Contains(u.Decls[0].Body, "const rtg_example_com_app_pkg_A, rtg_example_com_app_pkg_B = 1, 2") {
		t.Fatalf("const list was not rewritten: %q", u.Decls[0].Body)
	}
	if u.Decls[0].Name != "A, B" || u.Decls[0].UnitName != "" {
		t.Fatalf("const list metadata = %#v", u.Decls[0])
	}
	if !strings.Contains(u.Decls[1].Body, "var rtg_example_com_app_pkg_C, rtg_example_com_app_pkg_D int") {
		t.Fatalf("var list was not rewritten: %q", u.Decls[1].Body)
	}
	if u.Decls[1].Name != "C, D" || u.Decls[1].UnitName != "" {
		t.Fatalf("var list metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[2].Body, "return rtg_example_com_app_pkg_A + rtg_example_com_app_pkg_B + rtg_example_com_app_pkg_C + rtg_example_com_app_pkg_D") {
		t.Fatalf("function body did not rewrite all names: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersRawStringLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path:   "main.go",
				Source: []byte("package main\n\nfunc appMain() int {\n\tx := 077\n\ty := 0o10\n\tprint(`PASS\n`)\n\t_ = x + y\n\treturn 0\n}\n"),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	if !strings.Contains(u.Decls[0].Body, "print(\"PASS\\n\")") {
		t.Fatalf("raw string literal was not normalized: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[0].Body, "x := 63") || !strings.Contains(u.Decls[0].Body, "y := 8") {
		t.Fatalf("octal literals were not normalized: %q", u.Decls[0].Body)
	}
	if strings.Contains(u.Decls[0].Body, "`") {
		t.Fatalf("raw string marker leaked into lowered body: %q", u.Decls[0].Body)
	}
}

func TestPackageStripsStructTags(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path:   "main.go",
				Source: []byte("package main\n\ntype Record struct {\n\tName string `json:\"name\"`\n\tCount int \"db:\\\"count\\\"\"\n}\n\nfunc appMain() int {\n\tr := Record{Name: `PASS\\n`, Count: 1}\n\tprint(r.Name)\n\treturn r.Count - 1\n}\n"),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want Record and appMain", u.Decls)
	}
	typeBody := u.Decls[0].Body
	if strings.Contains(typeBody, "json") || strings.Contains(typeBody, "db:") || strings.Contains(typeBody, "`") {
		t.Fatalf("struct tag leaked into lowered type: %q", typeBody)
	}
	if !strings.Contains(typeBody, "Name string") || !strings.Contains(typeBody, "Count int") {
		t.Fatalf("struct fields were not preserved: %q", typeBody)
	}
	if !strings.Contains(u.Decls[1].Body, "Name: \"PASS\\\\n\"") {
		t.Fatalf("raw string literal outside tag was not normalized: %q", u.Decls[1].Body)
	}
}

func TestPackageLowersMultiValueAppend(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	xs := []int{10, 20, 30}
	ys := xs[:1]
	ys = append(ys, 99, xs[1])
	if ys[1] == 99 && ys[2] == 20 && xs[1] == 99 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "append(ys, 99, xs[1])") {
		t.Fatalf("multi-value append was not lowered: %q", body)
	}
	if !strings.Contains(body, " := append(ys, 99)") || !strings.Contains(body, " := append(rtg_example_com_app_appMain_tmp_1, rtg_example_com_app_appMain_tmp_0)") {
		t.Fatalf("multi-value append did not become repeated single-value appends: %q", body)
	}
	if !strings.Contains(body, " := xs[1]") {
		t.Fatalf("indexed append argument was not pre-evaluated: %q", body)
	}
}

func TestPackageHoistsLocalTypeDeclarations(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	type (
		count int
		box struct {
			value count
			text string
		}
		scores []count
	)
	type arrayValues [2]count
	type single struct {
		value count
	}
	var x count = 41
	b := box{value: x + 1, text: "PASS\n"}
	s := single{value: 1}
	var items arrayValues = arrayValues{1, 2}
	itemsCopy := items
	itemsCopy[0] = 9
	values := scores{1, 2}
	if int(b.value) == 42 && int(s.value) == 1 && int(items[0]) == 1 && int(itemsCopy[0]) == 9 && len(values) == 2 {
		print(b.text)
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 6 {
		t.Fatalf("decls = %#v, want 5 generated types and appMain", u.Decls)
	}
	countName := "rtg_example_com_app_appMain_local_type_0_count"
	boxName := "rtg_example_com_app_appMain_local_type_1_box"
	scoresName := "rtg_example_com_app_appMain_local_type_2_scores"
	valuesName := "rtg_example_com_app_appMain_local_type_3_arrayValues"
	singleName := "rtg_example_com_app_appMain_local_type_4_single"
	if !strings.Contains(u.Decls[0].Body, "type "+countName+" int") {
		t.Fatalf("count type was not hoisted: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[1].Body, "type "+boxName+" struct") || !strings.Contains(u.Decls[1].Body, "value "+countName) {
		t.Fatalf("box type was not hoisted with rewritten field type: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "type "+scoresName+" []"+countName) {
		t.Fatalf("scores type was not hoisted with rewritten element type: %q", u.Decls[2].Body)
	}
	if !strings.Contains(u.Decls[3].Body, "type "+valuesName+" []"+countName) || strings.Contains(u.Decls[3].Body, "[2]") {
		t.Fatalf("local named array type was not hoisted to slice-backed type: %q", u.Decls[3].Body)
	}
	if !strings.Contains(u.Decls[4].Body, "type "+singleName+" struct") || !strings.Contains(u.Decls[4].Body, "value "+countName) {
		t.Fatalf("single local struct type was not hoisted with rewritten field type: %q", u.Decls[4].Body)
	}
	body := u.Decls[5].Body
	if strings.Contains(body, "\ttype (") || strings.Contains(body, "\ttype count") || strings.Contains(body, valuesName+"{") || strings.Contains(body, "[2]") {
		t.Fatalf("local type declaration leaked into function body: %q", body)
	}
	for _, want := range []string{countName, boxName, scoresName, singleName} {
		if !strings.Contains(body, want) {
			t.Fatalf("function body does not use generated type %s: %q", want, body)
		}
	}
	if strings.Contains(body, valuesName) {
		t.Fatalf("function body retained local named array type %s: %q", valuesName, body)
	}
	for _, want := range []string{"var items []" + countName + " = []" + countName + "{1, 2}", "append(rtg_example_com_app_appMain_tmp_", "itemsCopy[0] = 9"} {
		if !strings.Contains(body, want) {
			t.Fatalf("function body missing local named array lowering %q: %q", want, body)
		}
	}
}

func TestPackageErasesInertNamedFunctionTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type F func(int) int
type G = func() (int, int)
type (
	H func(string)
	I = func() bool
)

func appMain() int {
	type Local func() int
	type (
		LocalPair func(int) int
		LocalAlias = func() (int, int)
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, bad := range []string{"func(int)", "type F", "type Local", "LocalPair", "LocalAlias"} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("function type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
	for _, decl := range u.Decls {
		if decl.Name == "F" || decl.Name == "G" || decl.Name == "H" || decl.Name == "I" || strings.Contains(decl.Body, "func(int)") {
			t.Fatalf("function type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	if strings.Contains(u.Decls[0].Body, "type (") {
		t.Fatalf("function type declaration leaked into lowered body: %q", u.Decls[0].Body)
	}
}

func TestPackageErasesInertFunctionContainingTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct { Fn func(int) int }
type Handler struct { Fn func() (int, int) }
type Callbacks []func(string) bool
type Alias = struct { Fn func() }
type Rows []struct { Fn func() int }
type (
	Holder struct { Fn func() }
	List []func(int) int
)

func appMain() int {
	type LocalBox struct { Fn func(int) int }
	type LocalCallbacks []func(string) bool
	type LocalRows []struct { Fn func() int }
	type (
		LocalHolder struct { Fn func() }
		LocalList []func(int) int
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Box" || decl.Name == "Handler" || decl.Name == "Callbacks" || decl.Name == "Alias" || decl.Name == "Rows" || decl.Name == "Holder" || decl.Name == "List" || strings.Contains(decl.Body, "func(") || strings.Contains(decl.Body, "func()") {
			t.Fatalf("function-containing type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"func(", "func()", "type Local", "LocalBox", "LocalCallbacks", "LocalRows", "LocalHolder", "LocalList", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("function-containing local type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertNamedInterfaceTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Empty interface{}
type Reader = interface { Read([]byte) int }
type EmbeddedBase interface { Base() int }
type EmbeddedChild interface { EmbeddedBase }
type (
	Closer interface { Close() int }
	Seeker = interface { Seek(int64, int) int64 }
)

func appMain() int {
	type Local interface{}
	type LocalBase interface { Base() int }
	type LocalChild interface { LocalBase }
	type (
		LocalReader interface { Read([]byte) int }
		LocalCloser = interface { Close() int }
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Empty" || decl.Name == "Reader" || decl.Name == "EmbeddedBase" || decl.Name == "EmbeddedChild" || decl.Name == "Closer" || decl.Name == "Seeker" || strings.Contains(decl.Body, "interface") {
			t.Fatalf("interface type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"interface", "type Local", "LocalBase", "LocalChild", "LocalReader", "LocalCloser", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("interface type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertInterfaceContainingTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct { Value interface{} }
type AnyBox struct { Value any }
type Values []interface{}
type AnyValues []any
type Alias = struct {
	Reader interface { Read([]byte) int }
	Value any
}
type (
	Holder struct { Value interface{} }
	AnyHolder struct { Value any }
	List []interface{}
	AnyList []any
)

func appMain() int {
	type LocalBox struct { Value interface{} }
	type LocalAnyBox struct { Value any }
	type LocalValues []interface{}
	type LocalAnyValues []any
	type (
		LocalHolder struct { Value interface{} }
		LocalAnyHolder struct { Value any }
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Box" || decl.Name == "AnyBox" || decl.Name == "Values" || decl.Name == "AnyValues" || decl.Name == "Alias" || decl.Name == "Holder" || decl.Name == "AnyHolder" || decl.Name == "List" || decl.Name == "AnyList" || strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface-containing type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"interface", "any", "type Local", "LocalBox", "LocalAnyBox", "LocalValues", "LocalAnyValues", "LocalHolder", "LocalAnyHolder", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("interface-containing local type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesBlankDiscardedInterfaceVars(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var value interface{}
	_ = value
	var other any
	_ = other
	var nilValue interface{} = nil
	_ = nilValue
	var nilOther any = nil
	_ = nilOther
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, bad := range []string{"interface", "any", "_ = value", "_ = other", "_ = nilValue", "_ = nilOther", "var value", "var other", "var nilValue", "var nilOther"} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("blank-discarded interface var leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
	if !strings.Contains(u.Decls[0].Body, "return 0") {
		t.Fatalf("lowered body lost return statement: %q", u.Decls[0].Body)
	}
}

func TestPackageLowersNilInterfaceVarComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var value interface{}
	if value != nil {
		return 1
	}
	var other any = nil
	if nil != other {
		return 2
	}
	if value == nil && nil == other {
		return 0
	}
	return 3
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"interface", "any", "var value", "var other", "value", "other", "nil"} {
		if strings.Contains(body, bad) {
			t.Fatalf("nil interface comparison leaked into lowered body as %q: %q", bad, body)
		}
	}
	for _, want := range []string{"if false", "return 0"} {
		if !strings.Contains(body, want) {
			t.Fatalf("lowered body missing %q: %q", want, body)
		}
	}
	if strings.Count(body, "if true") < 2 {
		t.Fatalf("lowered body did not preserve true nil comparisons: %q", body)
	}
}

func TestPackageErasesUnusedInterfaceParameters(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func use(value interface{}) {}
func useAny(value any, keep int) int { return keep }
func grouped(left, right interface{}, keep int) int { return keep }

func appMain() int {
	use(bump())
	use(1)
	useAny("x", 2)
	return grouped(1, 2, 3) - 5
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want bump, use, useAny, grouped, appMain", u.Decls)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["use"], "func rtg_example_com_app_use()") {
		t.Fatalf("unused interface-only parameter was not erased: %q", bodies["use"])
	}
	if !strings.Contains(bodies["useAny"], "func rtg_example_com_app_useAny(keep int) int") {
		t.Fatalf("unused any parameter was not erased: %q", bodies["useAny"])
	}
	if !strings.Contains(bodies["grouped"], "func rtg_example_com_app_grouped(keep int) int") {
		t.Fatalf("grouped unused interface parameters were not erased: %q", bodies["grouped"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{"rtg_example_com_app_use(rtg_example_com_app_bump())", "rtg_example_com_app_use(1)", "\"x\"", "rtg_example_com_app_grouped(1, 2, 3)"} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{"rtg_example_com_app_bump()\n\trtg_example_com_app_use()", "rtg_example_com_app_use()", "rtg_example_com_app_useAny(2)", "rtg_example_com_app_grouped(3)"} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased call %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesUnusedInterfaceParameterComputedSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}) {}

func appMain() int {
	use(wrap(bump() + 1))
	use(false && flag())
	use(true || flag())
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"rtg_example_com_app_use(rtg_example_com_app_wrap",
		"rtg_example_com_app_use(false",
		"rtg_example_com_app_use(true",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		"rtg_example_com_app_use()",
		"return 0",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased computed side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "rtg_example_com_app_use()") != 3 {
		t.Fatalf("expected three erased use calls in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterDeferSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}) {}

func appMain() int {
	defer use(wrap(bump() + 1))
	defer use(false && flag())
	defer use(true || flag())
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"defer ",
		"rtg_example_com_app_use(rtg_example_com_app_wrap",
		"rtg_example_com_app_use(false",
		"rtg_example_com_app_use(true",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface defer argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_appMain_return_tmp_0 := 0",
		"return rtg_example_com_app_appMain_return_tmp_0",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased defer side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "rtg_example_com_app_use()") != 3 {
		t.Fatalf("expected three erased deferred use calls in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterReturnSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	return use(wrap(bump() + 1), 7)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["use"], "func rtg_example_com_app_use(keep int) int") {
		t.Fatalf("unused interface parameter was not erased from use: %q", bodies["use"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{"return rtg_example_com_app_use(rtg_example_com_app_wrap", "interface"} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface return argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"return rtg_example_com_app_use(7)",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased return side-effect fragment %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesUnusedInterfaceParameterAssignmentSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	value := use(wrap(bump() + 1), 3)
	value = use(false && flag(), value)
	value = use(true || flag(), value)
	return value
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"value := rtg_example_com_app_use(rtg_example_com_app_wrap",
		"value = rtg_example_com_app_use(false",
		"value = rtg_example_com_app_use(true",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface assignment argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"value := rtg_example_com_app_use(3)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"value = rtg_example_com_app_use(value)",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"return value",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased assignment side-effect fragment %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesUnusedInterfaceParameterVarInitializerSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	var first = use(wrap(bump() + 1), 3)
	var second int = use(false && flag(), first)
	return use(true || flag(), second)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"var first = rtg_example_com_app_use(rtg_example_com_app_wrap",
		"var second int = rtg_example_com_app_use(false",
		"return rtg_example_com_app_use(true",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface var initializer argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"first := rtg_example_com_app_use(3)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"var second int = rtg_example_com_app_use(first)",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"return rtg_example_com_app_use(second)",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased var initializer side-effect fragment %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesUnusedInterfaceParameterIfConditionSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func check(value interface{}, keep bool) bool { return keep }

func appMain() int {
	if check(wrap(bump() + 1), true) {
		return 1
	}
	if check(false && flag(), true) {
		return 2
	}
	if check(true || flag(), true) {
		return 3
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["check"], "func rtg_example_com_app_check(keep bool) bool") {
		t.Fatalf("unused interface parameter was not erased from check: %q", bodies["check"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"if rtg_example_com_app_check(rtg_example_com_app_wrap",
		"if rtg_example_com_app_check(false",
		"if rtg_example_com_app_check(true ||",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface if-condition argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"return 0",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased if-condition side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "if rtg_example_com_app_check(true)") != 3 {
		t.Fatalf("expected three erased check conditions in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterForConditionSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func check(value interface{}, keep bool) bool { return keep }

func appMain() int {
	count := 0
	for check(wrap(bump() + 1), count < 1) {
		count = count + 1
	}
	for check(false && flag(), count < 2) {
		count = count + 1
	}
	for check(true || flag(), count < 3) {
		count = count + 1
	}
	return count
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["check"], "func rtg_example_com_app_check(keep bool) bool") {
		t.Fatalf("unused interface parameter was not erased from check: %q", bodies["check"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"for rtg_example_com_app_check(",
		"rtg_example_com_app_check(rtg_example_com_app_wrap",
		"rtg_example_com_app_check(false",
		"rtg_example_com_app_check(true ||",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface for-condition argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"for {",
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"if !(rtg_example_com_app_check(count < 1)) {",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		"if !(rtg_example_com_app_check(count < 2)) {",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"if !(rtg_example_com_app_check(count < 3)) {",
		"break",
		"return count",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased for-condition side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "if !(rtg_example_com_app_check(") != 3 {
		t.Fatalf("expected three erased check guards in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterEmptyClassicForConditionSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func check(value interface{}, keep bool) bool { return keep }

func appMain() int {
	count := 0
	for ; check(wrap(bump() + 1), count < 1); {
		count = count + 1
	}
	for ; check(false && flag(), count < 2); {
		count = count + 1
	}
	for ; check(true || flag(), count < 3); {
		count = count + 1
	}
	return count
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"for ;",
		"rtg_example_com_app_check(rtg_example_com_app_wrap",
		"rtg_example_com_app_check(false",
		"rtg_example_com_app_check(true ||",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface empty classic-for-condition argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"for {",
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"if !(rtg_example_com_app_check(count < 1)) {",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		"if !(rtg_example_com_app_check(count < 2)) {",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"if !(rtg_example_com_app_check(count < 3)) {",
		"break",
		"return count",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased empty classic-for-condition side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "if !(rtg_example_com_app_check(") != 3 {
		t.Fatalf("expected three erased check guards in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterClassicForConditionSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func check(value interface{}, keep bool) bool { return keep }

func appMain() int {
	count := 0
	for i := 0; check(wrap(bump() + 1), i < 1); i = i + 1 {
		count = count + 1
	}
	for ; check(false && flag(), count < 2); count = count + 1 {
		count = count + 1
	}
	for ; check(true || flag(), count < 4); count = count + 1 {
		count = count + 1
	}
	return count
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"for i := 0; rtg_example_com_app_check(",
		"rtg_example_com_app_check(rtg_example_com_app_wrap",
		"rtg_example_com_app_check(false",
		"rtg_example_com_app_check(true ||",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface classic-for-condition argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"for i := 0; ; i = i + 1 {",
		"for ; ; count = count + 1 {",
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"if !(rtg_example_com_app_check(i < 1)) {",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		"if !(rtg_example_com_app_check(count < 2)) {",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"if !(rtg_example_com_app_check(count < 4)) {",
		"break",
		"return count",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased classic-for-condition side-effect fragment %q in:\n%s", want, app)
		}
	}
	if strings.Count(app, "if !(rtg_example_com_app_check(") != 3 {
		t.Fatalf("expected three erased check guards in:\n%s", app)
	}
}

func TestPackageErasesUnusedInterfaceParameterSwitchTagSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func choose(value interface{}, keep int) int { return keep }

func appMain() int {
	switch choose(wrap(bump() + 1), 2) {
	case 2:
		return 1
	}
	switch choose(false && flag(), 3) {
	case 3:
		return 2
	}
	switch choose(true || flag(), 4) {
	case 4:
		return 3
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface parameter leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["choose"], "func rtg_example_com_app_choose(keep int) int") {
		t.Fatalf("unused interface parameter was not erased from choose: %q", bodies["choose"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{
		"switch rtg_example_com_app_choose(rtg_example_com_app_wrap",
		"switch rtg_example_com_app_choose(false",
		"switch rtg_example_com_app_choose(true ||",
	} {
		if strings.Contains(app, bad) {
			t.Fatalf("erased interface switch-tag argument leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_iparam_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_appMain_iparam_tmp_1 := rtg_example_com_app_appMain_iparam_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_appMain_iparam_tmp_1)",
		"switch rtg_example_com_app_choose(2)",
		" := false",
		"if rtg_example_com_app_appMain_iparam_tmp_",
		"rtg_example_com_app_flag()",
		"switch rtg_example_com_app_choose(3)",
		" := true",
		"if !rtg_example_com_app_appMain_iparam_tmp_",
		"switch rtg_example_com_app_choose(4)",
		"return 0",
	} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-erased switch-tag side-effect fragment %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesDiscardedInterfaceReturns(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "value.go",
				Source: []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func value() interface{} { return wrap(bump() + 1) }
func valueAny() any { return "x" }
`),
			},
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = value()
	valueAny()
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want bump, wrap, value, valueAny, appMain", u.Decls)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface return leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	if !strings.Contains(bodies["value"], "func rtg_example_com_app_value()") || strings.Contains(bodies["value"], "return rtg_example_com_app_bump") {
		t.Fatalf("interface return value was not erased: %q", bodies["value"])
	}
	for _, want := range []string{
		"rtg_example_com_app_value_ireturn_tmp_0 := rtg_example_com_app_bump()",
		"rtg_example_com_app_value_ireturn_tmp_1 := rtg_example_com_app_value_ireturn_tmp_0 + 1",
		"rtg_example_com_app_wrap(rtg_example_com_app_value_ireturn_tmp_1)",
		"return",
	} {
		if !strings.Contains(bodies["value"], want) {
			t.Fatalf("interface return direct-call side effect fragment %q missing: %q", want, bodies["value"])
		}
	}
	if strings.Contains(bodies["value"], "return rtg_example_com_app_wrap") {
		t.Fatalf("interface return direct-call side effect was not preserved: %q", bodies["value"])
	}
	if !strings.Contains(bodies["valueAny"], "func rtg_example_com_app_valueAny()") || strings.Contains(bodies["valueAny"], `return "x"`) {
		t.Fatalf("any return value was not erased: %q", bodies["valueAny"])
	}
	app := bodies["appMain"]
	for _, bad := range []string{"_ =", "return rtg_example_com_app_value", "return rtg_example_com_app_valueAny"} {
		if strings.Contains(app, bad) {
			t.Fatalf("discarded interface return leaked into appMain as %q:\n%s", bad, app)
		}
	}
	for _, want := range []string{"rtg_example_com_app_value()", "rtg_example_com_app_valueAny()"} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-return call %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesDiscardedInterfaceReturnShortCircuitArgs(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "value.go",
				Source: []byte(`package main

func flag() bool { return true }
func wrap(value bool) bool { return value }
func valueAnd() interface{} { return wrap(false && flag()) }
func valueOr() interface{} { return wrap(true || flag()) }
`),
			},
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = valueAnd()
	valueOr()
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface return leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	andBody := bodies["valueAnd"]
	for _, want := range []string{
		" := false",
		"if rtg_example_com_app_valueAnd_ireturn_tmp_",
		"rtg_example_com_app_flag()",
		"rtg_example_com_app_wrap(",
		"return",
	} {
		if !strings.Contains(andBody, want) {
			t.Fatalf("interface return && side-effect fragment %q missing: %q", want, andBody)
		}
	}
	orBody := bodies["valueOr"]
	for _, want := range []string{
		" := true",
		"if !rtg_example_com_app_valueOr_ireturn_tmp_",
		"rtg_example_com_app_flag()",
		"rtg_example_com_app_wrap(",
		"return",
	} {
		if !strings.Contains(orBody, want) {
			t.Fatalf("interface return || side-effect fragment %q missing: %q", want, orBody)
		}
	}
	app := bodies["appMain"]
	for _, want := range []string{"rtg_example_com_app_valueAnd()", "rtg_example_com_app_valueOr()"} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-return call %q in:\n%s", want, app)
		}
	}
}

func TestPackageErasesDiscardedInterfaceReturnBareComputedExpressions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "value.go",
				Source: []byte(`package main

func bump() int { return 1 }
func flag() bool { return true }
func valueSum() interface{} { return bump() + 1 }
func valueAnd() interface{} { return false && flag() }
func valueOr() interface{} { return true || flag() }
`),
			},
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = valueSum()
	valueAnd()
	_ = valueOr()
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodies := map[string]string{}
	for _, decl := range u.Decls {
		bodies[decl.Name] = decl.Body
		if strings.Contains(decl.Body, "interface") || strings.Contains(decl.Body, "any") {
			t.Fatalf("interface return leaked into lowered decl %s: %q", decl.Name, decl.Body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_valueSum_ireturn_tmp_0 := rtg_example_com_app_bump()",
		"return",
	} {
		if !strings.Contains(bodies["valueSum"], want) {
			t.Fatalf("interface return bare computed side-effect fragment %q missing: %q", want, bodies["valueSum"])
		}
	}
	if strings.Contains(bodies["valueSum"], "+ 1") {
		t.Fatalf("interface return erased value expression was still evaluated: %q", bodies["valueSum"])
	}
	for _, want := range []string{
		" := false",
		"if rtg_example_com_app_valueAnd_ireturn_tmp_",
		"rtg_example_com_app_flag()",
		"return",
	} {
		if !strings.Contains(bodies["valueAnd"], want) {
			t.Fatalf("interface return bare && fragment %q missing: %q", want, bodies["valueAnd"])
		}
	}
	for _, want := range []string{
		" := true",
		"if !rtg_example_com_app_valueOr_ireturn_tmp_",
		"rtg_example_com_app_flag()",
		"return",
	} {
		if !strings.Contains(bodies["valueOr"], want) {
			t.Fatalf("interface return bare || fragment %q missing: %q", want, bodies["valueOr"])
		}
	}
	app := bodies["appMain"]
	for _, want := range []string{"rtg_example_com_app_valueSum()", "rtg_example_com_app_valueAnd()", "rtg_example_com_app_valueOr()"} {
		if !strings.Contains(app, want) {
			t.Fatalf("missing lowered interface-return call %q in:\n%s", want, app)
		}
	}
}

func TestPackageLowersStaticInterfaceAssertions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

	func appMain() int {
		var x interface{} = 7
		var text any = "ok"
		var flag interface{} = true
		value := x.(int)
		gotText, okText := text.(string)
		gotFlag, okFlag := flag.(bool)
		missingText, okMissingText := x.(string)
		missingInt, okMissingInt := text.(int)
		missingFlag, okMissingFlag := flag.(int)
		if okText && okFlag && gotText == "ok" && gotFlag && x.(int) == 7 && !okMissingText && missingText == "" && !okMissingInt && missingInt == 0 && !okMissingFlag && missingFlag == 0 {
			return value - 7
		}
		return 1
	}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"interface", "any", ".("} {
		if strings.Contains(body, bad) {
			t.Fatalf("static interface assertion leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"var x int = 7",
		`var text string = "ok"`,
		"var flag bool = true",
		"value := x",
		"gotText, okText := text, true",
		"gotFlag, okFlag := flag, true",
		`missingText, okMissingText := "", false`,
		"missingInt, okMissingInt := 0, false",
		"missingFlag, okMissingFlag := 0, false",
		"if okText",
		"if okFlag",
		"if gotText == \"ok\"",
		"if gotFlag",
		"if x == 7",
		"if !okMissingText",
		"if missingText == \"\"",
		"if !okMissingInt",
		"if missingInt == 0",
		"if !okMissingFlag",
		"if missingFlag == 0",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static interface assertion %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersNilStaticInterfaceAssertions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var empty interface{}
	value, okValue := empty.(int)
	var nilAny any = nil
	text, okText := nilAny.(string)
	flag, okFlag := empty.(bool)
	if value == 0 && !okValue && text == "" && !okText && !flag && !okFlag {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"interface", "any", ".(", "var empty", "var nilAny", "empty", "nilAny"} {
		if strings.Contains(body, bad) {
			t.Fatalf("nil static interface assertion leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"value, okValue := 0, false",
		`text, okText := "", false`,
		"flag, okFlag := false, false",
		"return 0",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered nil static interface assertion %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticInterfaceStructAssertions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Value int
}

type Other struct {
	Value int
}

func appMain() int {
	var boxed interface{} = Box{Value: 41}
	var boxedPtr interface{} = &Box{Value: 42}
	got := boxed.(Box)
	gotAgain, okBox := boxed.(Box)
	gotPtr := boxedPtr.(*Box)
	gotPtrAgain, okPtr := boxedPtr.(*Box)
	missing, okMissing := boxed.(Other)
	missingPtr, okMissingPtr := boxedPtr.(*Other)
	if okBox && okPtr && !okMissing && !okMissingPtr && got.Value == 41 && gotAgain.Value == 41 && gotPtr.Value == 42 && gotPtrAgain.Value == 42 && missing.Value == 0 && missingPtr == nil {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	for _, bad := range []string{"interface", "any", ".("} {
		if strings.Contains(body, bad) {
			t.Fatalf("static interface struct assertion leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	typeName := "rtg_example_com_app_Box"
	otherTypeName := "rtg_example_com_app_Other"
	for _, want := range []string{
		"var boxed " + typeName + " = " + typeName + "{Value: 41}",
		typeName + "{Value: 42}",
		"var boxedPtr *" + typeName + " = &",
		"got := boxed",
		"gotAgain, okBox := boxed, true",
		"gotPtr := boxedPtr",
		"gotPtrAgain, okPtr := boxedPtr, true",
		"missing, okMissing := " + otherTypeName + "{}, false",
		"missingPtr, okMissingPtr := nil, false",
		"if okBox",
		"if okPtr",
		"if !okMissing",
		"if !okMissingPtr",
		"if got.Value == 41",
		"if gotAgain.Value == 41",
		"if gotPtr.Value == 42",
		"if gotPtrAgain.Value == 42",
		"if missing.Value == 0",
		"if missingPtr == nil",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static interface struct assertion %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticInterfaceAssertionPanics(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func discardMismatch() {
	var x interface{} = 7
	_ = x.(string)
}

func shortMismatch() {
	var x interface{} = 7
	value := x.(string)
	_ = value
}

func assignMismatch() {
	var x interface{} = 7
	value := "keep"
	value = x.(string)
	_ = value
}

func varMismatch() {
	var x interface{} = 7
	var value string = x.(string)
	_ = value
}

func inferredVarMismatch() {
	var x interface{} = 7
	var value = x.(bool)
	_ = value
}

func callMismatch() {
	var x interface{} = 7
	takeString(x.(string))
}

func callFirstArgMismatch() {
	var x interface{} = 7
	takeTwo(x.(string), "bad")
}

func deferMismatch() {
	var x interface{} = 7
	defer takeString(x.(string))
}

func deferFirstArgMismatch() {
	var x interface{} = 7
	defer takeTwo(x.(string), "bad")
}

func ifMismatch() {
	var x interface{} = 7
	if x.(bool) {
		takeString("bad")
	}
}

func forMismatch() {
	var x interface{} = 7
	for x.(bool) {
		takeString("bad")
	}
}

func switchMismatch() {
	var x interface{} = 7
	switch x.(string) {
	case "ok":
		takeString("bad")
	}
}

func returnMismatch() string {
	var x interface{} = 7
	return x.(string)
}

func returnPairMismatch() (string, string) {
	var x interface{} = 7
	return x.(string), laterString()
}

func takeString(value string) {}
func takeTwo(a string, b string) {}
func laterString() string { return "bad" }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	expectedMessage := map[string]string{
		"discardMismatch":       `interface conversion: interface {} is int, not string`,
		"shortMismatch":         `interface conversion: interface {} is int, not string`,
		"assignMismatch":        `interface conversion: interface {} is int, not string`,
		"varMismatch":           `interface conversion: interface {} is int, not string`,
		"inferredVarMismatch":   `interface conversion: interface {} is int, not bool`,
		"callMismatch":          `interface conversion: interface {} is int, not string`,
		"callFirstArgMismatch":  `interface conversion: interface {} is int, not string`,
		"deferMismatch":         `interface conversion: interface {} is int, not string`,
		"deferFirstArgMismatch": `interface conversion: interface {} is int, not string`,
		"ifMismatch":            `interface conversion: interface {} is int, not bool`,
		"forMismatch":           `interface conversion: interface {} is int, not bool`,
		"switchMismatch":        `interface conversion: interface {} is int, not string`,
		"returnMismatch":        `interface conversion: interface {} is int, not string`,
		"returnPairMismatch":    `interface conversion: interface {} is int, not string`,
	}
	for _, name := range []string{"discardMismatch", "shortMismatch", "assignMismatch", "varMismatch", "inferredVarMismatch", "callMismatch", "callFirstArgMismatch", "deferMismatch", "deferFirstArgMismatch", "ifMismatch", "forMismatch", "switchMismatch", "returnMismatch", "returnPairMismatch"} {
		body := bodyByName[name]
		if body == "" {
			t.Fatalf("missing lowered body for %s in %#v", name, u.Decls)
		}
		for _, bad := range []string{"interface{}", ".(", `panic("interface conversion`} {
			if strings.Contains(body, bad) {
				t.Fatalf("static interface assertion panic source leaked into %s as %q:\n%s", name, bad, body)
			}
		}
		for _, want := range []string{
			"var x int = 7",
			SymbolName("example.com/app", "__rtg_panic_active") + " = true",
			expectedMessage[name],
			"return",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("missing lowered assertion panic fragment %q in %s:\n%s", want, name, body)
			}
		}
	}
	if !strings.Contains(bodyByName["shortMismatch"], `value := ""`) {
		t.Fatalf("short assertion panic did not preserve local declaration:\n%s", bodyByName["shortMismatch"])
	}
	if !strings.Contains(bodyByName["assignMismatch"], `value := "keep"`) || strings.Contains(bodyByName["assignMismatch"], "value = x") {
		t.Fatalf("assignment assertion panic did not preserve previous value without assigning:\n%s", bodyByName["assignMismatch"])
	}
	if !strings.Contains(bodyByName["varMismatch"], `var value string = ""`) {
		t.Fatalf("var assertion panic did not preserve explicit local declaration:\n%s", bodyByName["varMismatch"])
	}
	if !strings.Contains(bodyByName["inferredVarMismatch"], `value := false`) {
		t.Fatalf("var assertion panic did not give inferred declaration a concrete type:\n%s", bodyByName["inferredVarMismatch"])
	}
	if strings.Contains(bodyByName["callMismatch"], "takeString(") {
		t.Fatalf("call assertion panic did not remove the skipped call:\n%s", bodyByName["callMismatch"])
	}
	if strings.Contains(bodyByName["callFirstArgMismatch"], "takeTwo(") || strings.Contains(bodyByName["callFirstArgMismatch"], `"bad"`) {
		t.Fatalf("first-arg call assertion panic did not remove later arguments:\n%s", bodyByName["callFirstArgMismatch"])
	}
	if strings.Contains(bodyByName["deferMismatch"], "defer ") || strings.Contains(bodyByName["deferMismatch"], "takeString(") {
		t.Fatalf("defer assertion panic did not remove the failed deferred call:\n%s", bodyByName["deferMismatch"])
	}
	if strings.Contains(bodyByName["deferFirstArgMismatch"], "defer ") || strings.Contains(bodyByName["deferFirstArgMismatch"], "takeTwo(") || strings.Contains(bodyByName["deferFirstArgMismatch"], `"bad"`) {
		t.Fatalf("first-arg defer assertion panic did not remove later arguments:\n%s", bodyByName["deferFirstArgMismatch"])
	}
	if strings.Contains(bodyByName["ifMismatch"], "if x") || strings.Contains(bodyByName["ifMismatch"], `takeString("bad")`) {
		t.Fatalf("if assertion panic did not remove the skipped branch:\n%s", bodyByName["ifMismatch"])
	}
	if strings.Contains(bodyByName["forMismatch"], "for x") || strings.Contains(bodyByName["forMismatch"], `takeString("bad")`) {
		t.Fatalf("for assertion panic did not remove the skipped loop:\n%s", bodyByName["forMismatch"])
	}
	if strings.Contains(bodyByName["switchMismatch"], "switch x") || strings.Contains(bodyByName["switchMismatch"], `takeString("bad")`) {
		t.Fatalf("switch assertion panic did not remove the skipped switch:\n%s", bodyByName["switchMismatch"])
	}
	if !strings.Contains(bodyByName["returnMismatch"], `return ""`) || strings.Contains(bodyByName["returnMismatch"], "return x") {
		t.Fatalf("return assertion panic did not lower to zero return tail:\n%s", bodyByName["returnMismatch"])
	}
	if !strings.Contains(bodyByName["returnPairMismatch"], `return "", ""`) || strings.Contains(bodyByName["returnPairMismatch"], "return x") || strings.Contains(bodyByName["returnPairMismatch"], "laterString") {
		t.Fatalf("multi-return assertion panic did not lower to zero return tail before later operands:\n%s", bodyByName["returnPairMismatch"])
	}
	if !strings.Contains(bodyByName["__rtg_recover"], SymbolName("example.com/app", "__rtg_panic_value")) {
		t.Fatalf("package panic recover helper was not generated: %#v", u.Decls)
	}
}

func TestPackageLowersStaticInterfaceTypeSwitches(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var x interface{} = 7
	var text any = "ok"
	var flag any = true
	var empty interface{}
	var nilAny any = nil
	total := 0
	switch x.(type) {
	case string:
		return 1
	case int:
		x, ok := x.(int)
		if ok {
			total = total + x
		}
	default:
		return 2
	}
	switch word := text.(type) {
	case string:
		if word == "ok" {
			total = total + 5
		}
	default:
		return 3
	}
	switch unused := flag.(type) {
	case int:
		return 4
	default:
		_ = unused
		total = total + 9
	}
	switch never := x.(type) {
	case string:
		_ = never
		return 5
	}
	switch blank := empty.(type) {
	case int:
		return 6
	case nil:
		_ = blank
		total = total + 4
	default:
		return 7
	}
	switch fallback := nilAny.(type) {
	case int:
		return 8
	default:
		_ = fallback
		total = total + 6
	}
	return total - 31
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"interface", "any", ".(type)", ".(int)", "case string", "case nil", "default:", "unused", "never", "empty", "nilAny", "blank", "fallback", "var blank nil", "var fallback nil"} {
		if strings.Contains(body, bad) {
			t.Fatalf("static interface type switch leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"var x int = 7",
		`var text string = "ok"`,
		"var flag bool = true",
		"total := 0",
		"x, ok := x, true",
		"if ok",
		"total = total + x",
		"var word string = text",
		`if word == "ok"`,
		"total = total + 5",
		"total = total + 9",
		"total = total + 4",
		"total = total + 6",
		"return total - 31",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static interface type switch %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticInterfaceStructTypeSwitches(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Value int
}

type Other struct {
	Value int
}

func appMain() int {
	var boxed any = Box{Value: 41}
	var boxedPtr any = &Box{Value: 42}
	total := 0
	switch boxed.(type) {
	case Other:
		return 1
	case Box:
		got := boxed.(Box)
		total = total + got.Value
	default:
		return 2
	}
	switch got := boxed.(type) {
	case Box:
		total = total + got.Value
	default:
		return 3
	}
	switch never := boxed.(type) {
	case Other:
		_ = never
		return 4
	}
	switch boxedPtr.(type) {
	case *Other:
		return 5
	case *Box:
		got := boxedPtr.(*Box)
		total = total + got.Value
	default:
		return 6
	}
	switch got := boxedPtr.(type) {
	case *Box:
		total = total + got.Value
	default:
		return 7
	}
	return total - 166
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	for _, bad := range []string{"interface", "any", ".(type)", ".(Box)", "case ", "default:", "never", "Other"} {
		if strings.Contains(body, bad) {
			t.Fatalf("static interface struct type switch leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	typeName := "rtg_example_com_app_Box"
	for _, want := range []string{
		"var boxed " + typeName + " = " + typeName + "{Value: 41}",
		typeName + "{Value: 42}",
		"var boxedPtr *" + typeName + " = &",
		"total := 0",
		"got := boxed",
		"total = total + got.Value",
		"var got " + typeName + " = boxed",
		"total = total + got.Value",
		"got := boxedPtr",
		"var got *" + typeName + " = boxedPtr",
		"return total - 166",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static interface struct type switch %q in:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersImportedStaticInterfaceStructAssertionsAndSwitches(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	var boxed any = dep.Box{Value: 41}
	var boxedPtr any = &dep.Box{Value: 42}
	got := boxed.(dep.Box)
	gotAgain, okBox := boxed.(dep.Box)
	gotPtr := boxedPtr.(*dep.Box)
	gotPtrAgain, okPtr := boxedPtr.(*dep.Box)
	missing, okMissing := boxed.(dep.Other)
	missingPtr, okMissingPtr := boxedPtr.(*dep.Other)
	total := 0
	if okBox {
		total = total + got.Value + gotAgain.Value
	}
	if okPtr {
		total = total + gotPtr.Value + gotPtrAgain.Value
	}
	if !okMissing && missing.Value == 0 {
		total = total + 3
	}
	if !okMissingPtr && missingPtr == nil {
		total = total + 4
	}
	switch boxed.(type) {
	case dep.Other:
		return 1
	case dep.Box:
		fromSwitch := boxed.(dep.Box)
		total = total + fromSwitch.Value
	default:
		return 2
	}
	switch value := boxed.(type) {
	case dep.Box:
		total = total + value.Value
	default:
		return 3
	}
	switch boxedPtr.(type) {
	case *dep.Other:
		return 4
	case *dep.Box:
		fromPtrSwitch := boxedPtr.(*dep.Box)
		total = total + fromPtrSwitch.Value
	default:
		return 5
	}
	switch ptrValue := boxedPtr.(type) {
	case *dep.Box:
		total = total + ptrValue.Value
	default:
		return 6
	}
	return total - 339
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Box struct {
	Value int
}

type Other struct {
	Value int
}
`),
			},
		},
	}
	u, err := PackageWithGraph(mainPkg, &load.Graph{Packages: []load.Package{mainPkg, depPkg}})
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	for _, bad := range []string{"interface", "any", ".(", ".(type)", "case ", "default:", "dep."} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported static interface struct form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	typeName := "rtg_example_com_app_pkg_dep_Box"
	otherTypeName := "rtg_example_com_app_pkg_dep_Other"
	for _, want := range []string{
		"var boxed " + typeName + " = " + typeName + "{Value: 41}",
		typeName + "{Value: 42}",
		"var boxedPtr *" + typeName + " = &",
		"got := boxed",
		"gotAgain, okBox := boxed, true",
		"gotPtr := boxedPtr",
		"gotPtrAgain, okPtr := boxedPtr, true",
		"missing, okMissing := " + otherTypeName + "{}, false",
		"missingPtr, okMissingPtr := nil, false",
		"fromSwitch := boxed",
		"var value " + typeName + " = boxed",
		"fromPtrSwitch := boxedPtr",
		"var ptrValue *" + typeName + " = boxedPtr",
		"return total - 339",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported static interface struct fragment %q in:\n%s", want, body)
		}
	}
	refFound := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/pkg/dep" && ref.Name == "Box" && ref.UnitName == typeName {
			refFound = true
		}
	}
	if !refFound {
		t.Fatalf("imported Box reference missing from %#v", u.References)
	}
}

func TestPackageErasesInertNamedAnyTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Alias any
type Other = any
type (
	Boxed any
	Named = any
)

func appMain() int {
	type Local any
	type (
		LocalAlias any
		LocalOther = any
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Alias" || decl.Name == "Other" || decl.Name == "Boxed" || decl.Name == "Named" || strings.Contains(decl.Body, "any") {
			t.Fatalf("any type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"any", "type Local", "LocalAlias", "LocalOther", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("any type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertNamedComplexTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Small complex64
type Wide = complex128
type (
	LocalSmall complex64
	LocalWide = complex128
)

func appMain() int {
	type Inner complex64
	type (
		InnerSmall complex64
		InnerWide = complex128
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Small" || decl.Name == "Wide" || decl.Name == "LocalSmall" || decl.Name == "LocalWide" || strings.Contains(decl.Body, "complex") {
			t.Fatalf("complex type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"complex64", "complex128", "type Inner", "InnerSmall", "InnerWide", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("complex type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertComplexContainingTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct { Value complex64 }
type WideBox struct { Value complex128 }
type Values []complex64
type WideValues []complex128
type Alias = struct { Items []complex128 }
type Rows []struct { Value complex64 }
type (
	Holder struct { Value complex64 }
	WideHolder struct { Value complex128 }
	List []complex64
	WideList []complex128
)

func appMain() int {
	type LocalBox struct { Value complex64 }
	type LocalWideBox struct { Value complex128 }
	type LocalValues []complex64
	type LocalWideValues []complex128
	type LocalRows []struct { Value complex64 }
	type (
		LocalHolder struct { Value complex64 }
		LocalWideHolder struct { Value complex128 }
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Box" || decl.Name == "WideBox" || decl.Name == "Values" || decl.Name == "WideValues" || decl.Name == "Alias" || decl.Name == "Rows" || decl.Name == "Holder" || decl.Name == "WideHolder" || decl.Name == "List" || decl.Name == "WideList" || strings.Contains(decl.Body, "complex") {
			t.Fatalf("complex-containing type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"complex64", "complex128", "type Local", "LocalBox", "LocalWideBox", "LocalValues", "LocalWideValues", "LocalRows", "LocalHolder", "LocalWideHolder", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("complex-containing local type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertNamedMapTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Table map[string]int
type Nested = map[string]map[string]int
type Alias map[string]Table
type (
	Counts map[string]int
	Rows = map[int]Table
)

func appMain() int {
	type Local map[string]int
	type LocalNested map[string]Local
	type (
		LocalCounts map[int]int
		LocalRows = map[string]LocalCounts
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Table" || decl.Name == "Nested" || decl.Name == "Alias" || decl.Name == "Counts" || decl.Name == "Rows" || strings.Contains(decl.Body, "map[") {
			t.Fatalf("map type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"map[", "type Local", "LocalNested", "LocalCounts", "LocalRows", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("map type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesInertMapContainingTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct { M map[string]int }
type Rows []map[string]int
type Alias = struct { Rows []map[string]int }
type (
	Holder struct { Table map[string]int }
	Tables []map[string]string
)

func appMain() int {
	type LocalBox struct { M map[string]int }
	type LocalRows []map[string]int
	type (
		LocalHolder struct { Table map[string]int }
		LocalTables []map[string]string
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Box" || decl.Name == "Rows" || decl.Name == "Alias" || decl.Name == "Holder" || decl.Name == "Tables" || strings.Contains(decl.Body, "map[") {
			t.Fatalf("map-containing type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"type Local", "LocalBox", "LocalRows", "LocalHolder", "LocalTables", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("map-containing local type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageErasesDiscardedEmptyMapContainingTypeComposites(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct { M map[string]int }
type AnonymousBox struct { M map[string]struct { A int } }
type Rows []map[string]int
type (
	Holder struct { Table map[string]int }
	Tables []map[string]string
	AnonymousHolder struct { Table map[string]struct { A int } }
)

func appMain() int {
	_ = Box{}
	_, _, _ = (Holder{}), Tables{}, AnonymousBox{}
	_ = AnonymousHolder{}
	type LocalBox struct { M map[string]int }
	type LocalAnonymousBox struct { M map[string]struct { A int } }
	type LocalRows []map[string]int
	_ = LocalBox{}
	_ = LocalAnonymousBox{}
	_ = (LocalRows{})
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Box" || decl.Name == "AnonymousBox" || decl.Name == "Rows" || decl.Name == "Holder" || decl.Name == "Tables" || decl.Name == "AnonymousHolder" || strings.Contains(decl.Body, "map[") {
			t.Fatalf("map-containing empty composite declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"Box{}", "AnonymousBox{}", "Holder{}", "Tables{}", "AnonymousHolder{}", "LocalBox{}", "LocalAnonymousBox{}", "LocalRows{}", "type Local", "map[", "_ ="} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("discarded map-containing empty composite leaked into lowered body as %q:\n%s", bad, u.Decls[0].Body)
		}
	}
	if !strings.Contains(u.Decls[0].Body, "return 0") {
		t.Fatalf("missing return in lowered body:\n%s", u.Decls[0].Body)
	}
}

func TestPackageErasesInertNamedArrayTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Values [3]int
type Bytes = [2]byte
type Alias [4]Values
type (
	Counts [2]int
	Rows = [3]Counts
)

func appMain() int {
	type Local [2]int
	type LocalNested [3]Local
	type (
		LocalCounts [2]int
		LocalRows = [3]LocalCounts
	)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want only appMain", u.Decls)
	}
	if u.Decls[0].Name != "appMain" {
		t.Fatalf("decl = %#v, want appMain", u.Decls[0])
	}
	for _, decl := range u.Decls {
		if decl.Name == "Values" || decl.Name == "Bytes" || decl.Name == "Alias" || decl.Name == "Counts" || decl.Name == "Rows" || strings.Contains(decl.Body, "[3]int") || strings.Contains(decl.Body, "[2]byte") {
			t.Fatalf("array type declaration leaked into lowered decls: %#v", u.Decls)
		}
	}
	for _, bad := range []string{"[3]int", "[2]int", "type Local", "LocalNested", "LocalCounts", "LocalRows", "type ("} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("array type declaration leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
}

func TestPackageLowersNamedArrayTypeValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Values [3]int

func appMain() int {
	var values Values
	values[0] = 1
	copy := values
	copy[0] = 9
	if values == Values{1, 0, 0} && copy[0] == 9 && len(values) == 3 && cap(values) == 3 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	typeBody := bodyByName["Values"]
	if !strings.Contains(typeBody, "type rtg_example_com_app_Values []int") {
		t.Fatalf("named array type was not lowered to slice-backed type:\n%s", typeBody)
	}
	body := bodyByName["appMain"]
	for _, bad := range []string{"[3]int", "Values{", "rtg_example_com_app_Values{", "copy := values"} {
		if strings.Contains(body, bad) {
			t.Fatalf("named array value form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"values := []int{0, 0, 0}",
		"make([]int, 0, 3)",
		"copy := append(",
		", values...)",
		"if (values[0] == 1 && values[1] == 0 && values[2] == 0) {",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered named array fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLocalAnonymousStructTypes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var zero struct{ A int }
	one := struct{ A int }{A: 1}
	var two = struct{ A int }{2}
	var three struct{ A int } = struct{ A int }{3}
	xs := []struct{ A int }{{A: 3}}
	return zero.A + one.A + two.A + three.A + xs[0].A - 9
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want generated type and appMain", u.Decls)
	}
	anonName := "rtg_example_com_app_appMain_anon_struct_0"
	if !strings.Contains(u.Decls[0].Body, "type "+anonName+" struct") || !strings.Contains(u.Decls[0].Body, "A int") {
		t.Fatalf("anonymous struct type was not hoisted: %q", u.Decls[0].Body)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "struct{") || strings.Contains(body, "struct {") {
		t.Fatalf("anonymous struct type leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "[]"+anonName) || strings.Count(body, anonName) < 6 {
		t.Fatalf("lowered body does not use generated anonymous struct type %s: %q", anonName, body)
	}
}

func TestPackageLowersAnonymousStructFields(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Holder struct {
	One struct{ A int }
	Two, Three struct{ A int }
	Rows []struct{ A int }
	More, Extra []struct{ A int }
	Ptr *struct{ A int }
	Left, Right *struct{ A int }
	Fields []*struct{ A int }
	MoreFields, OtherFields []*struct{ A int }
	Double **struct{ A int }
	DoubleLeft, DoubleRight **struct{ A int }
	Triple ***struct{ A int }
	TripleLeft, TripleRight ***struct{ A int }
}

func appMain() int {
	first := &struct{ A int }{13}
	second := &struct{ A int }{14}
	third := &struct{ A int }{15}
	fourth := &struct{ A int }{16}
	fifth := &struct{ A int }{17}
	sixth := &struct{ A int }{18}
	fourthPtr := &fourth
	fifthPtr := &fifth
	sixthPtr := &sixth
	h := Holder{
		One: struct{ A int }{1},
		Two: struct{ A int }{A: 2},
		Three: struct{ A int }{3},
		Rows: []struct{ A int }{{4}},
		More: []struct{ A int }{{A: 5}},
		Extra: []struct{ A int }{{6}},
		Ptr: &struct{ A int }{7},
		Left: &struct{ A int }{7},
		Right: &struct{ A int }{A: 8},
		Fields: []*struct{ A int }{&struct{ A int }{9}, &struct{ A int }{A: 10}},
		MoreFields: []*struct{ A int }{&struct{ A int }{11}},
		OtherFields: []*struct{ A int }{&struct{ A int }{A: 12}},
		Double: &first,
		DoubleLeft: &second,
		DoubleRight: &third,
		Triple: &fourthPtr,
		TripleLeft: &fifthPtr,
		TripleRight: &sixthPtr,
	}
	field0 := h.Fields[0]
	field1 := h.Fields[1]
	more := h.MoreFields[0]
	other := h.OtherFields[0]
	double := *h.Double
	doubleLeft := *h.DoubleLeft
	doubleRight := *h.DoubleRight
	triple := **h.Triple
	tripleLeft := **h.TripleLeft
	tripleRight := **h.TripleRight
	return h.One.A + h.Two.A + h.Three.A + h.Rows[0].A + h.More[0].A + h.Extra[0].A + h.Ptr.A + h.Left.A + h.Right.A + field0.A + field1.A + more.A + other.A + double.A + doubleLeft.A + doubleRight.A + triple.A + tripleLeft.A + tripleRight.A - 178
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want generated anonymous type, Holder, and appMain", u.Decls)
	}
	anonName := "rtg_example_com_app___rtg_anon_struct_0"
	if !strings.Contains(u.Decls[0].Body, "type "+anonName+" struct") || !strings.Contains(u.Decls[0].Body, "A int") {
		t.Fatalf("anonymous field struct type was not generated: %q", u.Decls[0].Body)
	}
	holder := u.Decls[1].Body
	if strings.Contains(holder, "One struct") || strings.Contains(holder, "Three struct") {
		t.Fatalf("anonymous field struct leaked into Holder: %q", holder)
	}
	for _, want := range []string{"One " + anonName, "Two " + anonName, "Three " + anonName, "Rows []" + anonName, "More []" + anonName, "Extra []" + anonName, "Ptr *" + anonName, "Left *" + anonName, "Right *" + anonName, "Fields []*" + anonName, "MoreFields []*" + anonName, "OtherFields []*" + anonName, "Double **" + anonName, "DoubleLeft **" + anonName, "DoubleRight **" + anonName, "Triple ***" + anonName, "TripleLeft ***" + anonName, "TripleRight ***" + anonName} {
		if !strings.Contains(holder, want) {
			t.Fatalf("Holder does not use generated anonymous struct type %q: %q", want, holder)
		}
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "struct{") || strings.Contains(body, "struct {") {
		t.Fatalf("anonymous struct literal type leaked into appMain: %q", body)
	}
	if strings.Count(body, anonName+"{") < 18 {
		t.Fatalf("appMain does not rewrite anonymous struct literals to %s: %q", anonName, body)
	}
	if strings.Count(body, "[]"+anonName) != 3 {
		t.Fatalf("appMain does not rewrite anonymous struct slice literals to []%s: %q", anonName, body)
	}
	if strings.Count(body, "[]*"+anonName) != 3 {
		t.Fatalf("appMain does not rewrite anonymous struct pointer-slice literals to []*%s: %q", anonName, body)
	}
	if strings.Contains(body, "&"+anonName+"{") {
		t.Fatalf("address-of anonymous struct literal was not normalized through a temp: %q", body)
	}
	if strings.Count(body, "&rtg_example_com_app_appMain_tmp_") < 13 {
		t.Fatalf("anonymous struct pointer literals were not normalized through temps: %q", body)
	}
}

func TestPackageLowersAnonymousStructParametersAndReturns(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func take(v struct{ A int }) int {
	return v.A
}

func give() struct{ A int } {
	return struct{ A int }{2}
}

func named() (v struct{ A int }) {
	v = struct{ A int }{3}
	return
}

func appMain() int {
	one := take(struct{ A int }{1})
	two := give()
	three := named()
	return one + two.A + three.A - 6
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want generated anonymous type plus four funcs", u.Decls)
	}
	anonName := "rtg_example_com_app___rtg_anon_struct_0"
	if !strings.Contains(u.Decls[0].Body, "type "+anonName+" struct") || !strings.Contains(u.Decls[0].Body, "A int") {
		t.Fatalf("anonymous signature struct type was not generated: %q", u.Decls[0].Body)
	}
	bodies := strings.Join([]string{u.Decls[1].Body, u.Decls[2].Body, u.Decls[3].Body, u.Decls[4].Body}, "\n")
	if strings.Contains(bodies, "struct{") || strings.Contains(bodies, "struct {") {
		t.Fatalf("anonymous signature struct leaked into lowered bodies:\n%s", bodies)
	}
	for _, want := range []string{
		"func rtg_example_com_app_take(v " + anonName + ") int",
		"func rtg_example_com_app_give() " + anonName,
		"func rtg_example_com_app_named() " + anonName,
		"var v " + anonName,
		"rtg_example_com_app_take(" + anonName + "{1})",
		"return one + two.A + three.A - 6",
	} {
		if !strings.Contains(bodies, want) {
			t.Fatalf("missing anonymous signature lowering fragment %q in:\n%s", want, bodies)
		}
	}
}

func TestPackageLowersTopLevelAnonymousStructVariables(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

var zero struct{ A int }
var one struct{ A int } = struct{ A int }{1}
var two = struct{ A int }{A: 2}
var (
	three struct{ A int }
	four = struct{ A int }{4}
)

func appMain() int {
	three = struct{ A int }{3}
	return zero.A + one.A + two.A + three.A + four.A - 10
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	anonName := "rtg_example_com_app___rtg_anon_struct_0"
	if len(u.Decls) < 6 {
		t.Fatalf("decls = %#v, want generated anonymous type, vars, and appMain", u.Decls)
	}
	if !strings.Contains(u.Decls[0].Body, "type "+anonName+" struct") || !strings.Contains(u.Decls[0].Body, "A int") {
		t.Fatalf("anonymous top-level var struct type was not generated: %q", u.Decls[0].Body)
	}
	var bodies []string
	var appMain string
	for i := 1; i < len(u.Decls); i++ {
		bodies = append(bodies, u.Decls[i].Body)
		if u.Decls[i].Name == "appMain" {
			appMain = u.Decls[i].Body
		}
	}
	joined := strings.Join(bodies, "\n")
	if strings.Contains(joined, "struct{") || strings.Contains(joined, "struct {") {
		t.Fatalf("anonymous top-level struct leaked into lowered bodies:\n%s", joined)
	}
	for _, want := range []string{
		"zero " + anonName,
		"one " + anonName + " = " + anonName + "{1}",
		"two = " + anonName + "{A: 2}",
		"three " + anonName,
		"four = " + anonName + "{4}",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing top-level anonymous struct lowering fragment %q in:\n%s", want, joined)
		}
	}
	if appMain == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	if !strings.Contains(appMain, "three = "+anonName+"{3}") {
		t.Fatalf("appMain does not rewrite assignment literal to %s: %q", anonName, appMain)
	}
}

func TestPackageLowersAnonymousStructNamedSlices(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Rows []struct{ A int }
type MoreRows []struct{ A int }

var global Rows = Rows{{A: 1}}

func appMain() int {
	rows := Rows{{A: 2}, {3}}
	more := MoreRows{{A: 4}}
	return global[0].A + rows[0].A + rows[1].A + more[0].A - 10
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	anonName := "rtg_example_com_app___rtg_anon_struct_0"
	rowsName := "rtg_example_com_app_Rows"
	moreRowsName := "rtg_example_com_app_MoreRows"
	if len(u.Decls) < 5 {
		t.Fatalf("decls = %#v, want generated anonymous type, named slices, global, and appMain", u.Decls)
	}
	joined := ""
	sourceBodies := ""
	appMain := ""
	for i := 0; i < len(u.Decls); i++ {
		joined += u.Decls[i].Body
		joined += "\n"
		if u.Decls[i].Name != anonName {
			sourceBodies += u.Decls[i].Body
			sourceBodies += "\n"
		}
		if u.Decls[i].Name == "appMain" {
			appMain = u.Decls[i].Body
		}
	}
	if strings.Contains(sourceBodies, "struct{") || strings.Contains(sourceBodies, "struct {") {
		t.Fatalf("anonymous struct type leaked into lowered named-slice source bodies:\n%s", sourceBodies)
	}
	for _, want := range []string{
		"type " + anonName + " struct",
		"type " + rowsName + " []" + anonName,
		"type " + moreRowsName + " []" + anonName,
		rowsName + "{" + anonName + "{A: 1}}",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing anonymous named-slice lowering fragment %q in:\n%s", want, joined)
		}
	}
	if appMain == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	for _, want := range []string{
		rowsName + "{" + anonName + "{A: 2}, " + anonName + "{3}}",
		moreRowsName + "{" + anonName + "{A: 4}}",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing appMain anonymous named-slice fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersAnonymousStructAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Alias = struct{ A int }
type (
	MoreAlias = struct{ A int }
)

var global Alias = Alias{1}

func appMain() int {
	local := MoreAlias{A: 2}
	return global.A + local.A - 3
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	aliasName := "rtg_example_com_app_Alias"
	moreAliasName := "rtg_example_com_app_MoreAlias"
	joined := ""
	appMain := ""
	for i := 0; i < len(u.Decls); i++ {
		joined += u.Decls[i].Body
		joined += "\n"
		if u.Decls[i].Name == "appMain" {
			appMain = u.Decls[i].Body
		}
	}
	if strings.Contains(joined, "= struct") || strings.Contains(joined, "=struct") {
		t.Fatalf("anonymous struct alias marker leaked into lowered bodies:\n%s", joined)
	}
	for _, want := range []string{
		"type " + aliasName + " struct",
		moreAliasName + " struct",
		"rtg_example_com_app_global " + aliasName + " = " + aliasName + "{1}",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing anonymous struct alias lowering fragment %q in:\n%s", want, joined)
		}
	}
	if appMain == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	for _, want := range []string{
		moreAliasName + "{A: 2}",
		"rtg_example_com_app_global.A + local.A - 3",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing appMain anonymous struct alias fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersInitFunctions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

var value int

func init() {
	value = 42
}

func appMain() int {
	return value - 42
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want value, init, appMain, guard, package init", u.Decls)
	}
	initName := "rtg_example_com_app___rtg_init_func_0"
	pkgInitName := "rtg_example_com_app___rtg_init"
	if u.Decls[1].Name != "__rtg_init_func_0" || u.Decls[1].UnitName != initName {
		t.Fatalf("init decl metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[1].Body, "func "+initName+"()") || strings.Contains(u.Decls[1].Body, "func init()") {
		t.Fatalf("init function was not renamed: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, pkgInitName+"()") {
		t.Fatalf("appMain did not call package init: %q", u.Decls[2].Body)
	}
	if !strings.Contains(u.Decls[4].Body, initName+"()") {
		t.Fatalf("package init did not call source init: %q", u.Decls[4].Body)
	}
	if len(u.Exports) != 1 || u.Exports[0].Name != "__rtg_init" || u.Exports[0].UnitName != pkgInitName {
		t.Fatalf("init export = %#v", u.Exports)
	}
}

func TestPackageLowersPackageInitializerCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct { value int }

func value() int {
	return 41
}

var a = value()
var b = box{value: value() + 1}

func init() {
	a = a + 1
}

func appMain() int {
	return a + b.value - 84
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	aName := "rtg_example_com_app_a"
	bName := "rtg_example_com_app_b"
	valueName := "rtg_example_com_app_value"
	boxName := "rtg_example_com_app_box"
	initName := "rtg_example_com_app___rtg_init_func_0"
	pkgInitName := "rtg_example_com_app___rtg_init"
	var aBody string
	var bBody string
	var appMainBody string
	var pkgInitBody string
	for i := 0; i < len(u.Decls); i++ {
		switch u.Decls[i].Name {
		case "a":
			aBody = u.Decls[i].Body
		case "b":
			bBody = u.Decls[i].Body
		case "appMain":
			appMainBody = u.Decls[i].Body
		case "__rtg_init":
			pkgInitBody = u.Decls[i].Body
		}
	}
	if !strings.Contains(aBody, "var "+aName+" int") || strings.Contains(aBody, valueName+"()") {
		t.Fatalf("a initializer was not moved out of declaration: %q", aBody)
	}
	if !strings.Contains(bBody, "var "+bName+" "+boxName) || strings.Contains(bBody, valueName+"()") {
		t.Fatalf("b initializer was not moved out of declaration: %q", bBody)
	}
	if !strings.Contains(appMainBody, pkgInitName+"()") {
		t.Fatalf("appMain did not call package init: %q", appMainBody)
	}
	for _, want := range []string{
		aName + " = " + valueName + "()",
		boxName + "{value:",
		bName + " = " + pkgInitName + "_tmp_",
		initName + "()",
	} {
		if !strings.Contains(pkgInitBody, want) {
			t.Fatalf("package init body missing %q:\n%s", want, pkgInitBody)
		}
	}
	if strings.Index(pkgInitBody, aName+" = ") > strings.Index(pkgInitBody, initName+"()") {
		t.Fatalf("package var initializer ran after source init:\n%s", pkgInitBody)
	}
}

func TestPackageLowersAppendConversionExpansion(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	dst := []byte{}
	text := "ok"
	dst = append(dst, []byte(text)...)
	return len(dst) - 2
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "[]byte(text)...") {
		t.Fatalf("append conversion expansion leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, " := []byte(text)") || !strings.Contains(body, "append(dst, rtg_example_com_app_appMain_tmp_0...)") {
		t.Fatalf("append conversion expansion was not lowered through a temp: %q", body)
	}
}

func TestPackageLowersStringCompoundAssignments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct {
	text string
}

func suffix() string { return "!" }

func appMain() int {
	s := "A"
	s += "B"
	b := box{text: "C"}
	b.text += suffix()
	values := []string{"D"}
	values[0] += s
	return len(s) + len(b.text) + len(values[0])
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want box, suffix, appMain", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "+=") {
		t.Fatalf("string compound assignment leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"s = s + \"B\"",
		"b.text = b.text + rtg_example_com_app_suffix()",
		"values[0] = values[0] + s",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered string compound assignment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersExpressionlessSwitch(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	x := 1
	switch {
	case x == 1:
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "switch {") {
		t.Fatalf("expressionless switch was not lowered: %q", body)
	}
	if !strings.Contains(body, "switch true {") {
		t.Fatalf("expressionless switch did not become boolean switch: %q", body)
	}
}

func TestPackageLowersFallthrough(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	total := 0
	switch 1 {
	case 1:
		total = total + 1
		fallthrough
	case 2:
		total = total + 2
		fallthrough
	default:
		total = total + 4
	}
	return total - 7
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "\t\tfallthrough") {
		t.Fatalf("fallthrough leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"goto rtg_fallthrough_0",
		"rtg_fallthrough_0:",
		"goto rtg_fallthrough_1",
		"rtg_fallthrough_1:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered fallthrough fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLabeledBreakContinue(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	total := 0
outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer
			}
			if i == 2 {
				break outer
			}
			total = total + 1
		}
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	for _, bad := range []string{"continue outer", "break outer"} {
		if strings.Contains(body, bad) {
			t.Fatalf("labeled control leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"goto rtg_labeled_continue_0",
		"rtg_labeled_continue_0:",
		"goto rtg_labeled_break_1",
		"rtg_labeled_break_1:",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered labeled-control fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersTopLevelDefer(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func emit(text string) {
	print(text)
}

func value() int {
	return 3
}

func appMain() int {
	text := "PASS\n"
	defer emit(text)
	text = "FAIL\n"
	defer emit("!")
	return value()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "\tdefer ") {
		t.Fatalf("defer leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		`rtg_example_com_app_appMain_defer_tmp_0 := text`,
		`rtg_example_com_app_appMain_defer_tmp_1 := "!"`,
		`rtg_example_com_app_appMain_return_tmp_2 := rtg_example_com_app_value()`,
		`rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_1)`,
		`rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_0)`,
		`return rtg_example_com_app_appMain_return_tmp_2`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered defer fragment %q in:\n%s", want, body)
		}
	}
	first := strings.Index(body, `rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_1)`)
	second := strings.Index(body, `rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_0)`)
	if first < 0 || second < 0 || first > second {
		t.Fatalf("deferred calls are not in LIFO order:\n%s", body)
	}
}

func TestPackageLowersDeferredFunctionLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	text := "FAIL\n"
	defer func() {
		print(text)
	}()
	defer func(value string) {
		print(value)
	}("!")
	text = "PASS\n"
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appBody := bodyByName["appMain"]
	if appBody == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	if strings.Contains(appBody, "defer ") || strings.Contains(appBody, "func()") || strings.Contains(appBody, "func(value string)") {
		t.Fatalf("deferred function literal leaked into lowered appMain:\n%s", appBody)
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_defer_tmp_0 := &text",
		`rtg_example_com_app_appMain_defer_tmp_1 := "!"`,
		`text = "PASS\n"`,
		"rtg_example_com_app_appMain_func_literal_1(rtg_example_com_app_appMain_defer_tmp_1)",
		"rtg_example_com_app_appMain_func_literal_0(rtg_example_com_app_appMain_defer_tmp_0)",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing deferred function literal fragment %q in appMain:\n%s", want, appBody)
		}
	}
	literal0 := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if !strings.Contains(literal0, "func rtg_example_com_app_appMain_func_literal_0(rtg_capture_text *string)") || !strings.Contains(literal0, "print(*rtg_capture_text)") {
		t.Fatalf("deferred closure capture was not lowered through a pointer:\n%s", literal0)
	}
	literal1 := bodyByName["rtg_example_com_app_appMain_func_literal_1"]
	if !strings.Contains(literal1, "func rtg_example_com_app_appMain_func_literal_1(value string)") || !strings.Contains(literal1, "print(value)") {
		t.Fatalf("deferred function literal argument was not lowered:\n%s", literal1)
	}
	first := strings.Index(appBody, "rtg_example_com_app_appMain_func_literal_1(")
	second := strings.Index(appBody, "rtg_example_com_app_appMain_func_literal_0(")
	if first < 0 || second < 0 || first > second {
		t.Fatalf("deferred function literals are not in LIFO order:\n%s", appBody)
	}
}

func TestPackageLowersNestedNoArgDefer(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func pass() { print("PASS") }
func fail() { print("FAIL") }
func newline() { print("\n") }

func appMain() int {
	defer newline()
	if true {
		defer pass()
	}
	if false {
		defer fail()
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl: %#v", u.Decls)
	}
	if strings.Contains(body, "\tdefer ") {
		t.Fatalf("defer leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_defer_active_0 := false",
		"rtg_example_com_app_appMain_defer_active_1 := false",
		"rtg_example_com_app_appMain_defer_active_0 = true",
		"rtg_example_com_app_appMain_defer_active_1 = true",
		"if rtg_example_com_app_appMain_defer_active_1 {",
		"rtg_example_com_app_fail()",
		"if rtg_example_com_app_appMain_defer_active_0 {",
		"rtg_example_com_app_pass()",
		"rtg_example_com_app_newline()",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing nested defer fragment %q in:\n%s", want, body)
		}
	}
	failCall := strings.LastIndex(body, "rtg_example_com_app_fail()")
	passCall := strings.LastIndex(body, "rtg_example_com_app_pass()")
	newlineCall := strings.LastIndex(body, "rtg_example_com_app_newline()")
	if failCall < 0 || passCall < 0 || newlineCall < 0 || failCall > passCall || passCall > newlineCall {
		t.Fatalf("nested defers are not emitted in reverse lexical order:\n%s", body)
	}
}

func TestPackageLowersNestedDeferArgumentCaptures(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func emit(text string, code int) {
	print(text)
}

func newline() {
	print("\n")
}

func appMain() int {
	text := "FAIL"
	defer newline()
	if true {
		text = "PASS"
		defer emit(text, 7)
		text = "BAD"
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl: %#v", u.Decls)
	}
	if strings.Contains(body, "\tdefer ") {
		t.Fatalf("defer leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_defer_active_0 := false",
		"var rtg_example_com_app_appMain_defer_capture_tmp_0 string",
		"var rtg_example_com_app_appMain_defer_capture_tmp_1 int",
		"rtg_example_com_app_appMain_defer_capture_tmp_0 = text",
		"rtg_example_com_app_appMain_defer_capture_tmp_1 = 7",
		"rtg_example_com_app_appMain_defer_active_0 = true",
		"rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_capture_tmp_0, rtg_example_com_app_appMain_defer_capture_tmp_1)",
		"rtg_example_com_app_newline()",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing nested defer capture fragment %q in:\n%s", want, body)
		}
	}
	emitCall := strings.LastIndex(body, "rtg_example_com_app_emit(")
	newlineCall := strings.LastIndex(body, "rtg_example_com_app_newline()")
	if emitCall < 0 || newlineCall < 0 || emitCall > newlineCall {
		t.Fatalf("deferred calls are not in LIFO order:\n%s", body)
	}
}

func TestPackageLowersVariadicDeferExpansion(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func emit(label string, values ...int) {
	print(label)
}

func newline() {
	print("\n")
}

func appMain() int {
	values := []int{1, 2}
	defer newline()
	defer emit("OUTER", values...)
	values = []int{9}
	if true {
		more := []int{3, 4}
		defer emit("INNER", more...)
		more = []int{8}
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl: %#v", u.Decls)
	}
	if strings.Contains(body, "\tdefer ") {
		t.Fatalf("defer leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		`var rtg_example_com_app_appMain_defer_capture_tmp_0 string`,
		`var rtg_example_com_app_appMain_defer_capture_tmp_1 []int`,
		`rtg_example_com_app_appMain_defer_tmp_2 := "OUTER"`,
		`rtg_example_com_app_appMain_defer_tmp_3 := values`,
		`rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_2, rtg_example_com_app_appMain_defer_tmp_3...)`,
		`rtg_example_com_app_appMain_defer_capture_tmp_0 = "INNER"`,
		`rtg_example_com_app_appMain_defer_capture_tmp_1 = more`,
		`rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_capture_tmp_0, rtg_example_com_app_appMain_defer_capture_tmp_1...)`,
		`rtg_example_com_app_newline()`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing variadic defer fragment %q in:\n%s", want, body)
		}
	}
	innerCall := strings.LastIndex(body, "rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_capture_tmp_0")
	outerCall := strings.LastIndex(body, "rtg_example_com_app_emit(rtg_example_com_app_appMain_defer_tmp_2")
	newlineCall := strings.LastIndex(body, "rtg_example_com_app_newline()")
	if innerCall < 0 || outerCall < 0 || newlineCall < 0 || innerCall > outerCall || outerCall > newlineCall {
		t.Fatalf("variadic deferred calls are not in LIFO order:\n%s", body)
	}
}

func TestPackageLowersLoopDefersWithDynamicStack(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func emit(value int) {
	if value == 0 {
		print("0")
	}
	if value == 1 {
		print("1")
	}
	if value == 2 {
		print("2")
	}
}

func mark(text string) {
	print(text)
}

func appMain() int {
	defer mark("A")
	for i := 0; i < 3; i++ {
		defer emit(i)
	}
	defer mark("T")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl: %#v", u.Decls)
	}
	if strings.Contains(body, "\tdefer ") {
		t.Fatalf("defer leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		" := []int{}",
		" = append(",
		"for rtg_example_com_app_appMain_defer_index_tmp_",
		"len(rtg_example_com_app_appMain_defer_order_tmp_",
		"rtg_example_com_app_mark(",
		"rtg_example_com_app_emit(",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing loop defer dynamic fragment %q in:\n%s", want, body)
		}
	}
	if strings.Count(body, "rtg_example_com_app_appMain_defer_order_tmp_") < 4 {
		t.Fatalf("defer order stack was not used throughout:\n%s", body)
	}
}

func TestPackagePropagatesPanicWhileEvaluatingDeferArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func sink(v int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	defer sink(boom())
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_defer_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_sink(rtg_example_com_app_appMain_defer_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing defer-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	boomCall := strings.Index(appMain, "rtg_example_com_app_boom()")
	check := strings.Index(appMain, "if "+active+" {")
	failPrint := strings.Index(appMain, `print("FAIL\n")`)
	sinkCall := strings.Index(appMain, "rtg_example_com_app_sink(")
	if boomCall < 0 || check < 0 || failPrint < 0 || sinkCall < 0 || !(boomCall < check && check < failPrint && failPrint < sinkCall) {
		t.Fatalf("defer argument panic check is not between argument evaluation and later statements:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicWhileEvaluatingLoopDeferArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func sink(v int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	for i := 0; i < 1; i++ {
		defer sink(boom())
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	order := "rtg_example_com_app_appMain_defer_order_tmp_"
	for _, want := range []string{
		"rtg_example_com_app_appMain_defer_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_sink(",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing loop defer-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	boomCall := strings.Index(appMain, "rtg_example_com_app_boom()")
	check := strings.Index(appMain, "if "+active+" {")
	currentOrderAppend := strings.Index(appMain, " = append("+order)
	if currentOrderAppend >= 0 {
		next := strings.Index(appMain[currentOrderAppend+1:], " = append("+order)
		if next >= 0 {
			currentOrderAppend = currentOrderAppend + 1 + next
		}
	}
	if boomCall < 0 || check < 0 || currentOrderAppend < 0 || !(boomCall < check && check < currentOrderAppend) {
		t.Fatalf("loop defer argument panic check is not before current defer registration:\n%s", appMain)
	}
}

func TestPackageLowersRecoverableStringPanic(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	value := recover()
	if value == "boom" {
		print("PASS\n")
	}
}

func appMain() int {
	defer cleanup()
	panic("boom")
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	if !strings.Contains(bodyByName["__rtg_recover"], "return "+SymbolName("example.com/app", "__rtg_panic_value")) {
		t.Fatalf("recover helper missing panic value return: %q", bodyByName["__rtg_recover"])
	}
	cleanup := bodyByName["cleanup"]
	if strings.Contains(cleanup, "value := recover()") {
		t.Fatalf("recover builtin leaked into lowered cleanup: %q", cleanup)
	}
	if !strings.Contains(cleanup, SymbolName("example.com/app", "__rtg_recover")+"()") {
		t.Fatalf("cleanup does not call generated recover helper: %q", cleanup)
	}
	appMain := bodyByName["appMain"]
	for _, bad := range []string{"defer cleanup()", `panic("boom")`} {
		if strings.Contains(appMain, bad) {
			t.Fatalf("panic/defer source leaked into lowered appMain: %q", appMain)
		}
	}
	for _, want := range []string{
		SymbolName("example.com/app", "__rtg_panic_active") + " = true",
		SymbolName("example.com/app", "__rtg_panic_value") + " = rtg_example_com_app_appMain_panic_tmp_",
		"rtg_example_com_app_cleanup()",
		"return 0",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered panic fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageWithGraphBridgesImportedStringPanic(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "example.com/app/dep"

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func appMain() int {
	defer cleanup()
	dep.Boom()
	print("FAIL\n")
	return 0
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

func Boom() {
	panic("boom")
}
`),
					},
				},
			},
		},
	}
	u, err := PackageWithGraph(graph.Packages[0], graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	depActive := SymbolName("example.com/app/dep", "__rtg_panic_active")
	depValue := SymbolName("example.com/app/dep", "__rtg_panic_value")
	active := SymbolName("example.com/app", "__rtg_panic_active")
	value := SymbolName("example.com/app", "__rtg_panic_value")
	for _, want := range []string{
		"if " + depActive + " {",
		active + " = true",
		value + " = " + depValue,
		depActive + " = false",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing imported panic bridge fragment %q in:\n%s", want, appMain)
		}
	}
	for _, want := range []unit.Symbol{
		{ImportPath: "example.com/app/dep", Name: "__rtg_panic_active", UnitName: depActive},
		{ImportPath: "example.com/app/dep", Name: "__rtg_panic_value", UnitName: depValue},
	} {
		found := false
		for i := 0; i < len(u.References); i++ {
			if u.References[i] == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing bridge reference %#v in %#v", want, u.References)
		}
	}
	depUnit, err := PackageWithGraph(graph.Packages[1], graph)
	if err != nil {
		t.Fatalf("PackageWithGraph dep failed: %v", err)
	}
	for _, want := range []unit.Symbol{
		{ImportPath: "example.com/app/dep", Name: "__rtg_panic_active", UnitName: depActive},
		{ImportPath: "example.com/app/dep", Name: "__rtg_panic_value", UnitName: depValue},
	} {
		found := false
		for i := 0; i < len(depUnit.Exports); i++ {
			if depUnit.Exports[i] == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing dep panic export %#v in %#v", want, depUnit.Exports)
		}
	}
}

func TestPackageWithGraphKeepsAllImportedPanicBridgeReferences(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/depa", "example.com/app/depb"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import (
	depa "example.com/app/depa"
	depb "example.com/app/depb"
)

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func appMain() int {
	defer cleanup()
	depa.Boom()
	depb.Boom()
	print("FAIL\n")
	return 0
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/depa",
				Name:       "depa",
				Files: []load.File{
					{
						Path: "depa.go",
						Source: []byte(`package depa

func Boom() {
	panic("boom")
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/depb",
				Name:       "depb",
				Files: []load.File{
					{
						Path: "depb.go",
						Source: []byte(`package depb

func Boom() {
	panic("boom")
}
`),
					},
				},
			},
		},
	}
	u, err := PackageWithGraph(graph.Packages[0], graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	wantRefs := []unit.Symbol{
		{
			ImportPath: "example.com/app/depa",
			Name:       "__rtg_panic_active",
			UnitName:   SymbolName("example.com/app/depa", "__rtg_panic_active"),
		},
		{
			ImportPath: "example.com/app/depa",
			Name:       "__rtg_panic_value",
			UnitName:   SymbolName("example.com/app/depa", "__rtg_panic_value"),
		},
		{
			ImportPath: "example.com/app/depb",
			Name:       "__rtg_panic_active",
			UnitName:   SymbolName("example.com/app/depb", "__rtg_panic_active"),
		},
		{
			ImportPath: "example.com/app/depb",
			Name:       "__rtg_panic_value",
			UnitName:   SymbolName("example.com/app/depb", "__rtg_panic_value"),
		},
	}
	for _, want := range wantRefs {
		if !strings.Contains(appMain, want.UnitName) {
			t.Fatalf("missing imported panic bridge use %q in:\n%s", want.UnitName, appMain)
		}
		count := 0
		for i := 0; i < len(u.References); i++ {
			if u.References[i] == want {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("bridge reference %#v count = %d, want 1 in %#v", want, count, u.References)
		}
	}
}

func TestPackageLowersUnrecoveredAppMainStringPanicToAbort(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	panic("boom")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	abort := bodyByName["__rtg_panic_abort"]
	for _, want := range []string{
		`print("panic: ")`,
		"print(" + SymbolName("example.com/app", "__rtg_panic_value") + ")",
		"return 2",
	} {
		if !strings.Contains(abort, want) {
			t.Fatalf("missing panic abort fragment %q in:\n%s", want, abort)
		}
	}
	appMain := bodyByName["appMain"]
	for _, want := range []string{
		SymbolName("example.com/app", "__rtg_panic_active") + " = true",
		"if " + SymbolName("example.com/app", "__rtg_panic_active") + " {",
		"return " + SymbolName("example.com/app", "__rtg_panic_abort") + "()",
		"return 0",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing unrecovered appMain panic fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackagePropagatesPanicAcrossDirectCall(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() {
	panic("boom")
}

func appMain() int {
	defer cleanup()
	boom()
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"return 0",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing panic propagation fragment %q in:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	failPrint := strings.Index(appMain, `print("FAIL\n")`)
	if check < 0 || failPrint < 0 || check > failPrint {
		t.Fatalf("panic propagation check does not precede following statements:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicAcrossAssignmentCall(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func callBoom() int {
	value := boom()
	return value
}

func appMain() int {
	defer cleanup()
	value := callBoom()
	printInt(value)
	return 0
}

func printInt(v int) int { return v }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	active := SymbolName("example.com/app", "__rtg_panic_active")
	callBoom := bodyByName["callBoom"]
	for _, want := range []string{
		"rtg_example_com_app_callBoom_panic_assign_tmp_",
		" := rtg_example_com_app_boom()",
		"value := rtg_example_com_app_callBoom_panic_assign_tmp_",
		"if " + active + " {",
		"return 0",
	} {
		if !strings.Contains(callBoom, want) {
			t.Fatalf("missing assignment panic propagation fragment %q in callBoom:\n%s", want, callBoom)
		}
	}
	appMain := bodyByName["appMain"]
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_assign_tmp_",
		" := rtg_example_com_app_callBoom()",
		"value := rtg_example_com_app_appMain_panic_assign_tmp_",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"return 0",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing caller assignment panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	use := strings.Index(appMain, "rtg_example_com_app_printInt(value)")
	if check < 0 || use < 0 || check > use {
		t.Fatalf("assignment panic propagation check does not precede following statements:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicAcrossNestedAssignmentCallArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func wrap(v int) int {
	return v
}

func appMain() int {
	defer cleanup()
	value := wrap(boom())
	printInt(value)
	print("FAIL\n")
	return 0
}

func printInt(v int) int { return v }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_assign_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_appMain_panic_assign_tmp_",
		" := rtg_example_com_app_wrap(rtg_example_com_app_appMain_panic_assign_arg_tmp_",
		"value := rtg_example_com_app_appMain_panic_assign_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing nested assignment panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	outerCall := strings.Index(appMain, "rtg_example_com_app_wrap(")
	assign := strings.Index(appMain, "value := rtg_example_com_app_appMain_panic_assign_tmp_")
	if check < 0 || outerCall < 0 || assign < 0 || !(check < outerCall && outerCall < assign) {
		t.Fatalf("nested assignment panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBetweenAssignmentOperands(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func appMain() int {
	defer cleanup()
	a, b := boom(), fail()
	printInt(a + b)
	return 0
}

func printInt(v int) int { return v }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_assign_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		" := rtg_example_com_app_fail()",
		"a, b := rtg_example_com_app_appMain_panic_assign_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing assignment operand panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	failCall := strings.Index(appMain, "rtg_example_com_app_fail()")
	if check < 0 || failCall < 0 || check > failCall {
		t.Fatalf("assignment operand panic propagation check does not precede later operand:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicAfterMultiResultAssignmentCall(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boomPair() (int, int) {
	panic("boom")
	return 1, 2
}

func appMain() int {
	defer cleanup()
	a, b := boomPair()
	printInt(a + b)
	return 0
}

func printInt(v int) int { return v }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"a, b := rtg_example_com_app_boomPair()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"return 0",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing multi-result assignment panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	if strings.Contains(appMain, "panic_assign_tmp") {
		t.Fatalf("multi-result assignment call was incorrectly collapsed into one temp:\n%s", appMain)
	}
	assign := strings.Index(appMain, "a, b := rtg_example_com_app_boomPair()")
	check := strings.Index(appMain, "if "+active+" {")
	use := strings.Index(appMain, "rtg_example_com_app_printInt(a + b)")
	if assign < 0 || check < 0 || use < 0 || assign > check || check > use {
		t.Fatalf("multi-result assignment panic check is not between call and use:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeOuterCallStatement(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func fail(v int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	fail(boom())
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_fail(rtg_example_com_app_appMain_panic_arg_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing call-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	failCall := strings.Index(appMain, "rtg_example_com_app_fail(rtg_example_com_app_appMain_panic_arg_tmp_")
	if check < 0 || failCall < 0 || check > failCall {
		t.Fatalf("call-argument panic propagation check does not precede outer call:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeNestedCallStatementArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func wrap(v int) int {
	return v
}

func sink(v int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	sink(wrap(boom()))
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_arg_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_appMain_panic_arg_tmp_",
		" := rtg_example_com_app_wrap(rtg_example_com_app_appMain_panic_arg_arg_tmp_",
		"rtg_example_com_app_sink(rtg_example_com_app_appMain_panic_arg_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing nested call-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	outerArg := strings.Index(appMain, "rtg_example_com_app_wrap(")
	outerCall := strings.Index(appMain, "rtg_example_com_app_sink(")
	if check < 0 || outerArg < 0 || outerCall < 0 || !(check < outerArg && outerArg < outerCall) {
		t.Fatalf("nested call-argument panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeDeepNestedCallStatementArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 7
}

func mid(v int) int {
	return v
}

func top(v int) int {
	return v
}

func sink(v int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	sink(top(mid(boom())))
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_arg_arg_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_appMain_panic_arg_arg_tmp_",
		" := rtg_example_com_app_mid(rtg_example_com_app_appMain_panic_arg_arg_arg_tmp_",
		"rtg_example_com_app_appMain_panic_arg_tmp_",
		" := rtg_example_com_app_top(rtg_example_com_app_appMain_panic_arg_arg_tmp_",
		"rtg_example_com_app_sink(rtg_example_com_app_appMain_panic_arg_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing deep nested call-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	midCall := strings.Index(appMain, "rtg_example_com_app_mid(")
	topCall := strings.Index(appMain, "rtg_example_com_app_top(")
	sinkCall := strings.Index(appMain, "rtg_example_com_app_sink(")
	if check < 0 || midCall < 0 || topCall < 0 || sinkCall < 0 || !(check < midCall && midCall < topCall && topCall < sinkCall) {
		t.Fatalf("deep nested call-argument panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBetweenCallStatementArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func sink(a int, b int) {
	print("FAIL\n")
}

func appMain() int {
	defer cleanup()
	sink(boom(), fail())
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		" := rtg_example_com_app_fail()",
		"rtg_example_com_app_sink(rtg_example_com_app_appMain_panic_arg_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing call-argument panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	failCall := strings.Index(appMain, "rtg_example_com_app_fail()")
	if check < 0 || failCall < 0 || check > failCall {
		t.Fatalf("call-argument panic propagation check does not precede later argument:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeIfConditionBranch(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func shouldBoom() bool {
	panic("boom")
	return true
}

func appMain() int {
	defer cleanup()
	if shouldBoom() {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_shouldBoom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"if rtg_example_com_app_appMain_panic_cond_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing if-condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	branch := strings.Index(appMain, "if rtg_example_com_app_appMain_panic_cond_tmp_")
	if check < 0 || branch < 0 || check > branch {
		t.Fatalf("if-condition panic propagation check does not precede branch:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeNestedIfConditionArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func keep(v int) bool {
	return true
}

func appMain() int {
	defer cleanup()
	if keep(boom()) {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_cond_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_keep(rtg_example_com_app_appMain_panic_cond_arg_tmp_",
		"if rtg_example_com_app_appMain_panic_cond_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing nested if-condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	outerCall := strings.Index(appMain, "rtg_example_com_app_keep(")
	branch := strings.Index(appMain, "if rtg_example_com_app_appMain_panic_cond_tmp_")
	if check < 0 || outerCall < 0 || branch < 0 || !(check < outerCall && outerCall < branch) {
		t.Fatalf("nested if-condition panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeSwitchConditionBranch(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func shouldBoom() int {
	panic("boom")
	return 1
}

func appMain() int {
	defer cleanup()
	switch shouldBoom() {
	case 1:
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_shouldBoom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"switch rtg_example_com_app_appMain_panic_cond_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing switch-condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	branch := strings.Index(appMain, "switch rtg_example_com_app_appMain_panic_cond_tmp_")
	if check < 0 || branch < 0 || check > branch {
		t.Fatalf("switch-condition panic propagation check does not precede switch:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeForConditionBody(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func shouldBoom() bool {
	panic("boom")
	return true
}

func appMain() int {
	defer cleanup()
	for shouldBoom() {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"for {",
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_shouldBoom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"if !(rtg_example_com_app_appMain_panic_cond_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing for-condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	bodyPrint := strings.Index(appMain, `print("FAIL\n")`)
	if check < 0 || bodyPrint < 0 || check > bodyPrint {
		t.Fatalf("for-condition panic propagation check does not precede loop body:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeClassicForCondition(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func appMain() int {
	defer cleanup()
	for i := boom(); i < 1; i++ {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"{",
		"rtg_example_com_app_appMain_panic_for_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"i := rtg_example_com_app_appMain_panic_for_tmp_",
		"for ; i < 1; i++",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing classic-for init panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	condition := strings.Index(appMain, "for ; i < 1; i++")
	if check < 0 || condition < 0 || check > condition {
		t.Fatalf("classic-for init panic propagation check does not precede condition:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeClassicForEmptyConditionBody(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func shouldBoom() bool {
	panic("boom")
	return true
}

func appMain() int {
	defer cleanup()
	for ; shouldBoom(); {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"for {",
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_shouldBoom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"if !(rtg_example_com_app_appMain_panic_cond_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing classic-for condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	bodyPrint := strings.Index(appMain, `print("FAIL\n")`)
	if check < 0 || bodyPrint < 0 || check > bodyPrint {
		t.Fatalf("classic-for condition panic propagation check does not precede loop body:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicAfterClassicForBodyBeforePost(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func next() int {
	panic("boom")
	return 1
}

func appMain() int {
	defer cleanup()
	for i := 0; i < 1; i = next() {
		print("BODY\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"i := 0",
		"for {",
		"if !(i < 1) {",
		"print(\"BODY\\n\")",
		"rtg_example_com_app_appMain_panic_post_tmp_",
		" := rtg_example_com_app_next()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"i = rtg_example_com_app_appMain_panic_post_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing classic-for post panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	bodyPrint := strings.Index(appMain, `print("BODY\n")`)
	check := strings.Index(appMain, "if "+active+" {")
	fallthroughPrint := strings.LastIndex(appMain, `print("FAIL\n")`)
	if bodyPrint < 0 || check < 0 || bodyPrint > check || (fallthroughPrint >= 0 && check > fallthroughPrint) {
		t.Fatalf("classic-for post panic propagation check is not after body and before fallthrough:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBeforeCombinedClassicForCondition(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func shouldBoom() bool {
	panic("boom")
	return true
}

func appMain() int {
	defer cleanup()
	for i := 0; shouldBoom(); i++ {
		print("FAIL\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"i := 0",
		"for {",
		"rtg_example_com_app_appMain_panic_cond_tmp_",
		" := rtg_example_com_app_shouldBoom()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"if !(rtg_example_com_app_appMain_panic_cond_tmp_",
		"i++",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing combined classic-for condition panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	check := strings.Index(appMain, "if "+active+" {")
	bodyPrint := strings.Index(appMain, `print("FAIL\n")`)
	post := strings.Index(appMain, "i++")
	if check < 0 || bodyPrint < 0 || post < 0 || check > bodyPrint || bodyPrint > post {
		t.Fatalf("combined classic-for condition panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicThroughCombinedClassicForInitAndPost(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func start() int {
	return 0
}

func next() int {
	panic("boom")
	return 1
}

func appMain() int {
	defer cleanup()
	for i := start(); i < 1; i = next() {
		print("BODY\n")
	}
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	appMain := bodyByName["appMain"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_appMain_panic_for_tmp_",
		" := rtg_example_com_app_start()",
		"i := rtg_example_com_app_appMain_panic_for_tmp_",
		"if !(i < 1) {",
		`print("BODY\n")`,
		"rtg_example_com_app_appMain_panic_post_tmp_",
		" := rtg_example_com_app_next()",
		"if " + active + " {",
		"rtg_example_com_app_cleanup()",
		"i = rtg_example_com_app_appMain_panic_post_tmp_",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing combined classic-for init/post panic propagation fragment %q in appMain:\n%s", want, appMain)
		}
	}
	initCall := strings.Index(appMain, "rtg_example_com_app_start()")
	bodyPrint := strings.Index(appMain, `print("BODY\n")`)
	postCall := strings.Index(appMain, "rtg_example_com_app_next()")
	if initCall < 0 || bodyPrint < 0 || postCall < 0 || !(initCall < bodyPrint && bodyPrint < postCall) {
		t.Fatalf("combined classic-for init/post panic propagation order is wrong:\n%s", appMain)
	}
}

func TestPackagePropagatesPanicBetweenReturnOperands(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func pair() (int, int) {
	return boom(), fail()
}

func appMain() int {
	defer cleanup()
	_, _ = pair()
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	pair := bodyByName["pair"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_pair_panic_return_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"return 0, 0",
		"rtg_example_com_app_pair_panic_return_tmp_",
		" := rtg_example_com_app_fail()",
	} {
		if !strings.Contains(pair, want) {
			t.Fatalf("missing return-operand panic propagation fragment %q in pair:\n%s", want, pair)
		}
	}
	check := strings.Index(pair, "if "+active+" {")
	failCall := strings.Index(pair, "rtg_example_com_app_fail()")
	if check < 0 || failCall < 0 || check > failCall {
		t.Fatalf("return-operand panic propagation check does not precede later operand:\n%s", pair)
	}
}

func TestPackagePropagatesPanicFromSingleDirectReturn(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func value() int {
	return boom()
}

func appMain() int {
	defer cleanup()
	_ = value()
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	value := bodyByName["value"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_value_panic_return_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"return 0",
		"return rtg_example_com_app_value_panic_return_tmp_",
	} {
		if !strings.Contains(value, want) {
			t.Fatalf("missing single-return panic propagation fragment %q in value:\n%s", want, value)
		}
	}
}

func TestPackagePropagatesPanicFromNestedReturnCallArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
	}
}

func boom() int {
	panic("boom")
	return 1
}

func wrap(v int) int {
	return v
}

func value() int {
	return wrap(boom())
}

func appMain() int {
	defer cleanup()
	_ = value()
	print("FAIL\n")
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	value := bodyByName["value"]
	active := SymbolName("example.com/app", "__rtg_panic_active")
	for _, want := range []string{
		"rtg_example_com_app_value_panic_return_arg_tmp_",
		" := rtg_example_com_app_boom()",
		"if " + active + " {",
		"return 0",
		"rtg_example_com_app_value_panic_return_tmp_",
		" := rtg_example_com_app_wrap(rtg_example_com_app_value_panic_return_arg_tmp_",
		"return rtg_example_com_app_value_panic_return_tmp_",
	} {
		if !strings.Contains(value, want) {
			t.Fatalf("missing nested-return panic propagation fragment %q in value:\n%s", want, value)
		}
	}
	check := strings.Index(value, "if "+active+" {")
	outerCall := strings.Index(value, "rtg_example_com_app_wrap(")
	ret := strings.Index(value, "return rtg_example_com_app_value_panic_return_tmp_")
	if check < 0 || outerCall < 0 || ret < 0 || !(check < outerCall && outerCall < ret) {
		t.Fatalf("nested-return panic propagation order is wrong:\n%s", value)
	}
}

func TestPackageLowersDeferredMultiResultDirectReturn(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func emit(text string) {
	print(text)
}

func pair() (int, int) {
	return 1, 2
}

func wrapper() (int, int) {
	defer emit("!")
	return pair()
}

func appMain() int {
	a, b := wrapper()
	if a == 1 && b == 2 {
		print("PASS\n")
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	wrapper := bodyByName["wrapper"]
	for _, want := range []string{
		"rtg_example_com_app_wrapper_return_tmp_",
		" := rtg_example_com_app_pair()",
		"rtg_example_com_app_emit(rtg_example_com_app_wrapper_defer_tmp_0)",
		"return rtg_example_com_app_wrapper_return_tmp_",
		", rtg_example_com_app_wrapper_return_tmp_",
	} {
		if !strings.Contains(wrapper, want) {
			t.Fatalf("missing deferred multi-result return fragment %q in wrapper:\n%s", want, wrapper)
		}
	}
}

func TestPackageLowersSimpleRangeLoops(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Items []int
type Box struct {
	values []int
}

func makeBox() Box {
	return Box{values: []int{10, 11}}
}

func appMain() int {
	values := []int{1, 2, 3}
	total := 0
	assignedIndex := 0
	holder := Items{0}
	for i, v := range values {
		total = total + i + v
	}
	for i := range values {
		total = total + i
	}
	for _, v := range []int{4, 5} {
		total = total + v
	}
	for _, v := range (Items{6, 7}) {
		total = total + v
	}
	for _, v := range append([]int{}, 8, 9) {
		total = total + v
	}
	for _, v := range makeBox().values {
		total = total + v
	}
	for _, v := range Items([]int{12, 13}) {
		total = total + v
	}
	for assignedIndex, holder[0] = range values {
		total = total + assignedIndex + holder[0]
	}
	return total - 106
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "range") {
		t.Fatalf("range syntax leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "for i := 0; i < len(") || !strings.Contains(body, "v := rtg_example_com_app_appMain_tmp_0[i]") {
		t.Fatalf("two-value range was not lowered to index loop: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := values") || !strings.Contains(body, "for i := 0; i < len(rtg_example_com_app_appMain_tmp_1); i++") {
		t.Fatalf("index-only range was not lowered to index loop: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_2 := []int{4, 5}") {
		t.Fatalf("literal range operand was not evaluated once: %q", body)
	}
	if !strings.Contains(body, " := (rtg_example_com_app_Items{6, 7})") {
		t.Fatalf("named literal range operand was not evaluated once: %q", body)
	}
	if strings.Contains(body, "append([]int{}, 8, 9)") {
		t.Fatalf("multi-value append range operand was not normalized: %q", body)
	}
	if !strings.Contains(body, " := []int{}") || !strings.Contains(body, ", 8)") || !strings.Contains(body, ", 9)") {
		t.Fatalf("multi-value append range operand missing normalized append temps: %q", body)
	}
	if strings.Contains(body, "makeBox().values") {
		t.Fatalf("returned struct field range operand was not normalized: %q", body)
	}
	if !strings.Contains(body, " := rtg_example_com_app_makeBox()") || !strings.Contains(body, ".values") {
		t.Fatalf("returned struct field range operand missing selector temps: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_Items([]int{12, 13})") {
		t.Fatalf("named slice conversion range operand was not removed: %q", body)
	}
	if !strings.Contains(body, " := rtg_example_com_app_Items{12, 13}") {
		t.Fatalf("named slice conversion range operand missing slice temp: %q", body)
	}
	if !strings.Contains(body, "assignedIndex = rtg_example_com_app_appMain_tmp_") || !strings.Contains(body, "holder[0] = rtg_example_com_app_appMain_tmp_") {
		t.Fatalf("range assignment targets were not lowered: %q", body)
	}
}

func TestPackageLowersStringRangeLoops(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func text() string {
	return "aé日"
}

func appMain() int {
	total := 0
	for i, r := range text() {
		total = total + i + int(r)
	}
	for i := range "éx" {
		total = total + i
	}
	var assigned int32
	for _, assigned = range "é" {
		total = total + int(assigned)
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	declIndex := -1
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			declIndex = i
			break
		}
	}
	if declIndex < 0 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[declIndex].Body
	if strings.Contains(body, "range") {
		t.Fatalf("range syntax leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"; rtg_example_com_app_appMain_tmp_",
		" < len(",
		"); {",
		" := int(",
		" = int32(",
		" >= 240",
		" >= 224",
		" >= 192",
		" = 65533",
		"i = i + ",
		"assigned = rtg_example_com_app_appMain_tmp_",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing string range lowering fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersNamedSliceConversions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Items []int

func count(values Items) int {
	return len(values)
}

func appMain() int {
	values := Items([]int{1, 2})
	return len(values) + count(Items([]int{3, 4})) - 4
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	if strings.Contains(body, "rtg_example_com_app_Items([]int") {
		t.Fatalf("named slice conversion leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "values := rtg_example_com_app_Items{1, 2}") || !strings.Contains(body, " := rtg_example_com_app_Items{3, 4}") || !strings.Contains(body, "rtg_example_com_app_count(rtg_example_com_app_appMain_tmp_") {
		t.Fatalf("named slice conversions were not lowered to named literals: %q", body)
	}
}

func TestPackageLowersNamedPointerStructAndUnnamedSliceConversions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type A struct { X int }
type B A
type APtr *A
type Values []int

func wantB(v B) int {
	return v.X
}

func appMain() int {
	a := A{X: 3}
	p := APtr(&a)
	b := B(A{X: 5})
	xs := []int(Values{1, 2})
	return p.X + wantB(b) + len(xs) - 10
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{
		"rtg_example_com_app_APtr(&a)",
		"rtg_example_com_app_B(rtg_example_com_app_A{X: 5})",
		"[]int(rtg_example_com_app_Values{1, 2})",
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("conversion leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"p := &a",
		"b := rtg_example_com_app_B{X: 5}",
		"xs := rtg_example_com_app_Values{1, 2}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered conversion fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLocalConstDeclarations(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path:   "main.go",
				Source: []byte("package main\n\nfunc appMain() int {\n\tconst answer = 40 + 2\n\tconst (\n\t\tread = 1 << iota\n\t\twrite\n\t\traw = `PASS\\n`\n\t\tlegacy = 077\n\t)\n\tprint(raw)\n\treturn answer + read + write + legacy - 109\n}\n"),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[len(u.Decls)-1].Body
	if strings.Contains(body, "const") || strings.Contains(body, "iota") || strings.Contains(body, "`") || strings.Contains(body, "077") {
		t.Fatalf("local const syntax leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"answer := 40 + 2",
		"read := 1 << 0",
		"write := 1 << 1",
		"raw := \"PASS\\\\n\"",
		"legacy := 63",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered const %q in: %q", want, body)
		}
	}
}

func TestPackageLowersNamedResultReturns(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func value() (out int) {
	out = 7
	return
}

func pair() (left, right int) {
	left = 3
	right = 4
	return
}

func appMain() int {
	a, b := pair()
	return value() + a + b - 14
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want value, pair, appMain", u.Decls)
	}
	valueBody := u.Decls[0].Body
	if strings.Contains(valueBody, "(out int)") || !strings.Contains(valueBody, "func rtg_example_com_app_value() int") || !strings.Contains(valueBody, "var out int") || !strings.Contains(valueBody, "return out") {
		t.Fatalf("single named result was not lowered: %q", valueBody)
	}
	pairBody := u.Decls[1].Body
	if strings.Contains(pairBody, "left, right int") || !strings.Contains(pairBody, "func rtg_example_com_app_pair() (int, int)") || !strings.Contains(pairBody, "var left int") || !strings.Contains(pairBody, "var right int") || !strings.Contains(pairBody, "return left, right") {
		t.Fatalf("grouped named results were not lowered: %q", pairBody)
	}
}

func TestPackageLowersPrintlnBuiltin(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	msg := "PASS"
	println(msg)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[len(u.Decls)-1].Body
	if strings.Contains(body, "println") {
		t.Fatalf("println leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "print(msg)\n\tprint(\"\\n\")") {
		t.Fatalf("println was not lowered to print calls: %q", body)
	}
}

func TestPackageLowersReducibleComplexComponents(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() float64 { return 8 }
func second() float64 { return 9 }

func appMain() int {
	var a float64 = 7
	b := 3.5
	r := real(complex(a, b))
	i := imag(complex(4, 5))
	side := real(complex(first(), second()))
	literalR := real(1+2i)
	literalI := imag(3-4i)
	pureI := imag(-5i)
	reverseR := real(6i-7)
	_ = r
	_ = i
	_ = side
	_ = literalR
	_ = literalI
	_ = pureI
	_ = reverseR
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("decls = %#v, missing appMain", u.Decls)
	}
	for _, leaked := range []string{"real(", "imag(", "complex("} {
		if strings.Contains(body, leaked) {
			t.Fatalf("%s leaked into lowered body: %q", leaked, body)
		}
	}
	if !strings.Contains(body, "r := a") || !strings.Contains(body, "i := 5") {
		t.Fatalf("complex components were not reduced in lowered body: %q", body)
	}
	for _, want := range []string{
		"literalR := 1",
		"literalI := -4",
		"pureI := -5",
		"reverseR := -7",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing literal complex reduction %q in:\n%s", want, body)
		}
	}
	firstCall := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()")
	secondCall := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()")
	assign := strings.Index(body, "side := rtg_example_com_app_appMain_tmp_0")
	if firstCall < 0 || secondCall < 0 || assign < 0 || firstCall > secondCall || secondCall > assign {
		t.Fatalf("side-effecting complex components were not evaluated in order:\n%s", body)
	}
}

func TestPackageLowersStaticComplexAliasComponents(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() float64 { return 8 }
func second() float64 { return 9 }

func appMain() int {
	z := complex(first(), second())
	literal := 1.5 + 2.5i
	var typed complex128 = complex(first(), second())
	var typedLiteral complex64 = 3.5 + 4.5i
	r := real(z)
	i := imag(z)
	lr := real(literal)
	li := imag(literal)
	tr := real(typed)
	ti := imag(typed)
	tlr := real(typedLiteral)
	tli := imag(typedLiteral)
	var total float64 = r + i + lr + li + tr + ti + tlr + tli
	_ = total
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("decls = %#v, missing appMain", u.Decls)
	}
	for _, leaked := range []string{"real(", "imag(", "complex(", "complex64", "complex128", "1.5 + 2.5i", "3.5 + 4.5i", "z :=", "literal :=", "typed ", "typedLiteral "} {
		if strings.Contains(body, leaked) {
			t.Fatalf("%s leaked into lowered body:\n%s", leaked, body)
		}
	}
	for _, want := range []string{
		"rtg_complex_alias_",
		"_real := rtg_example_com_app_first()",
		"_imag := rtg_example_com_app_second()",
		"_real := 1.5",
		"_imag := 2.5",
		"_real := 3.5",
		"_imag := 4.5",
		"r := rtg_complex_alias_",
		"i := rtg_complex_alias_",
		"lr := rtg_complex_alias_",
		"li := rtg_complex_alias_",
		"tr := rtg_complex_alias_",
		"ti := rtg_complex_alias_",
		"tlr := rtg_complex_alias_",
		"tli := rtg_complex_alias_",
		"var total float64 = r + i + lr + li + tr + ti + tlr + tli",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing static complex alias lowering fragment %q in:\n%s", want, body)
		}
	}
	firstCall := strings.Index(body, "rtg_example_com_app_first()")
	secondCall := strings.Index(body, "rtg_example_com_app_second()")
	use := strings.Index(body, "r := rtg_complex_alias_")
	if firstCall < 0 || secondCall < 0 || use < 0 || firstCall > secondCall || secondCall > use {
		t.Fatalf("complex alias components were not evaluated before use:\n%s", body)
	}
}

func TestPackageErasesDiscardedPureComplexExpressions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = 1i
	_ = 1 + 2i
	_ = (3 - 4i)
	_ = complex(1, 2)
	_, _ = 5i, complex(-6, +7)
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, leaked := range []string{"1i", "2i", "4i", "5i", "complex(", "_ ="} {
		if strings.Contains(body, leaked) {
			t.Fatalf("discarded complex expression leaked as %q in lowered body:\n%s", leaked, body)
		}
	}
	if !strings.Contains(body, "return 0") {
		t.Fatalf("return statement missing after complex discard lowering:\n%s", body)
	}
}

func TestPackageErasesDiscardedComplexCallSideEffects(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(x int) float64 { return 1.5 }
func second() float64 { return 2.5 }
func third() float64 { return 3.5 }
func fourth() float64 { return 4.5 }

func appMain() int {
	_ = complex(first(1), 2.0)
	_, _ = 5i, complex(3.0, second())
	_ = complex(third(), second())
	_ = complex(first(1), 2.0) + 3i
	_ = 4i + complex(5.0, second())
	_ = complex(third(), 6.0) - complex(7.0, fourth())
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("decls = %#v, missing appMain", u.Decls)
	}
	for _, leaked := range []string{"complex(", "5i", "_ ="} {
		if strings.Contains(body, leaked) {
			t.Fatalf("discarded complex expression leaked as %q in lowered body:\n%s", leaked, body)
		}
	}
	searchStart := 0
	for _, want := range []string{
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second()",
		"rtg_example_com_app_third()",
		"rtg_example_com_app_second()",
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second()",
		"rtg_example_com_app_third()",
		"rtg_example_com_app_fourth()",
	} {
		idx := strings.Index(body[searchStart:], want)
		if idx < 0 {
			t.Fatalf("missing discarded complex side-effect call %q in order:\n%s", want, body)
		}
		searchStart += idx + len(want)
	}
	ret := strings.Index(body[searchStart:], "return 0")
	if ret < 0 {
		t.Fatalf("return statement did not follow discarded complex side effects:\n%s", body)
	}
}

func TestPackageErasesBlankDiscardedComplexVars(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() float64 { return 1.5 }
func second() float64 { return 2.5 }

func appMain() int {
	var a complex64 = 1 + 2i
	_ = a
	var b complex128 = complex(first(), second())
	_ = b
	var c = 3 - 4i
	_ = c
	d := complex(first(), second())
	_ = d
	e := 5 + 6i
	_ = e
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
			break
		}
	}
	if body == "" {
		t.Fatalf("decls = %#v, missing appMain", u.Decls)
	}
	for _, leaked := range []string{"complex64", "complex128", "1i", "2i", "4i", "6i", "complex(", "_ = a", "_ = b", "_ = c", "_ = d", "_ = e"} {
		if strings.Contains(body, leaked) {
			t.Fatalf("blank-discarded complex var leaked as %q in lowered body:\n%s", leaked, body)
		}
	}
	firstCall := strings.Index(body, "rtg_example_com_app_first()")
	secondCall := strings.Index(body, "rtg_example_com_app_second()")
	secondFirstCall := strings.Index(body[secondCall+1:], "rtg_example_com_app_first()")
	secondSecondCall := strings.Index(body[secondCall+1:], "rtg_example_com_app_second()")
	ret := strings.Index(body, "return 0")
	if firstCall < 0 || secondCall < 0 || secondFirstCall < 0 || secondSecondCall < 0 || ret < 0 {
		t.Fatalf("complex var side-effect calls were not preserved before return:\n%s", body)
	}
	secondFirstCall += secondCall + 1
	secondSecondCall += secondCall + 1
	if firstCall > secondCall || secondCall > secondFirstCall || secondFirstCall > secondSecondCall || secondSecondCall > ret {
		t.Fatalf("complex var side-effect calls were not preserved before return:\n%s", body)
	}
}

func TestPackageLowersNewNamedStructBuiltin(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct { value int }

func appMain() int {
	p := new(box)
	p.value = 1
	return p.value - 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want box and appMain", u.Decls)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "new(") {
		t.Fatalf("new leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "var rtg_example_com_app_appMain_tmp_0 []rtg_example_com_app_box = make([]rtg_example_com_app_box, 1)") || !strings.Contains(body, "p := &rtg_example_com_app_appMain_tmp_0[0]") {
		t.Fatalf("new named struct was not lowered to arena-backed element address: %q", body)
	}
}

func TestPackageLowersNewScalarBuiltins(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	p := new(int)
	*p = 7
	ok := new(bool)
	*ok = true
	text := new(string)
	*text = "ok"
	if *ok {
		return *p + len(*text) - 9
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "new(") {
		t.Fatalf("new scalar builtin leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"var rtg_example_com_app_appMain_tmp_0 []int = make([]int, 1)",
		"p := &rtg_example_com_app_appMain_tmp_0[0]",
		"var rtg_example_com_app_appMain_tmp_1 []bool = make([]bool, 1)",
		"ok := &rtg_example_com_app_appMain_tmp_1[0]",
		"var rtg_example_com_app_appMain_tmp_2 []string = make([]string, 1)",
		"text := &rtg_example_com_app_appMain_tmp_2[0]",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered scalar new fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersNewPointerAndSliceBuiltins(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	x := 3
	p := new(*int)
	*p = &x
	values := new([]int)
	*values = append(*values, **p)
	bytes := new([]byte)
	*bytes = append(*bytes, byte(2))
	return (*values)[0] + int((*bytes)[0]) - 5
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want two new slice helpers and appMain", u.Decls)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	body := bodyByName["appMain"]
	if strings.Contains(body, "new(") {
		t.Fatalf("new pointer/slice builtin leaked into lowered body: %q", body)
	}
	intHelper := newSliceHelperUnitName(SymbolName("example.com/app", "appMain"), "[]int")
	byteHelper := newSliceHelperUnitName(SymbolName("example.com/app", "appMain"), "[]byte")
	for _, helper := range []string{intHelper, byteHelper} {
		helperBody := bodyByName[helper]
		if !strings.Contains(helperBody, "func "+helper+"() *") || !strings.Contains(helperBody, "make([]int, 3)") || !strings.Contains(helperBody, "return &"+helper+"_words[0]") {
			t.Fatalf("missing generated new slice helper %s:\n%s", helper, helperBody)
		}
	}
	for _, want := range []string{
		"var rtg_example_com_app_appMain_tmp_0 []*int = make([]*int, 1)",
		"p := &rtg_example_com_app_appMain_tmp_0[0]",
		"values := " + intHelper + "()",
		"bytes := " + byteHelper + "()",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered pointer/slice new fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersNewImportedNamedStructBuiltin(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	p := new(dep.Box)
	p.Value = 1
	return p.Value - 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "box.go",
				Source: []byte(`package dep

type Box struct { Value int }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "new(") || strings.Contains(body, "dep.Box") {
		t.Fatalf("new imported struct leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "var rtg_example_com_app_main_appMain_tmp_0 []rtg_example_com_app_dep_Box = make([]rtg_example_com_app_dep_Box, 1)") || !strings.Contains(body, "p := &rtg_example_com_app_main_appMain_tmp_0[0]") {
		t.Fatalf("new imported named struct was not lowered to arena-backed element address: %q", body)
	}
	if len(u.References) != 1 || u.References[0].ImportPath != "example.com/app/dep" || u.References[0].Name != "Box" {
		t.Fatalf("references = %#v, want dep.Box", u.References)
	}
}

func TestPackageWithGraphLowersNewNamedStructAliasBuiltin(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

type localBox struct { value int }
type localAlias localBox

func appMain() int {
	p := new(localAlias)
	p.value = 1
	q := new(dep.Alias)
	q.Value = 2
	return p.value + q.Value - 3
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "box.go",
				Source: []byte(`package dep

type Box struct { Value int }
type Alias Box
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want localBox, localAlias, and appMain", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "new(") || strings.Contains(body, "dep.Alias") {
		t.Fatalf("new named struct alias leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "var rtg_example_com_app_main_appMain_tmp_0 []rtg_example_com_app_main_localAlias = make([]rtg_example_com_app_main_localAlias, 1)") || !strings.Contains(body, "p := &rtg_example_com_app_main_appMain_tmp_0[0]") {
		t.Fatalf("new local named struct alias was not lowered: %q", body)
	}
	if !strings.Contains(body, "var rtg_example_com_app_main_appMain_tmp_1 []rtg_example_com_app_dep_Alias = make([]rtg_example_com_app_dep_Alias, 1)") || !strings.Contains(body, "q := &rtg_example_com_app_main_appMain_tmp_1[0]") {
		t.Fatalf("new imported named struct alias was not lowered: %q", body)
	}
	if len(u.References) != 1 || u.References[0].ImportPath != "example.com/app/dep" || u.References[0].Name != "Alias" {
		t.Fatalf("references = %#v, want dep.Alias", u.References)
	}
}

func TestPackageLowersCopyStringSources(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Label string
type box struct { text string }

func (b box) Text() string {
	return b.text
}

func fill(text string) int {
	dst := []byte{0, 0}
	return copy(dst, text)
}

func makeText() string {
	return "hi"
}

func appMain() int {
	dst := []byte{0, 0}
	text := "hi"
	var alias = text
	var named Label = Label("hi")
	b := box{text: "hi"}
	fromCall := makeText()
	n := copy(dst, text)
	m := copy(dst, "ok")
	p := copy(dst, alias)
	q := copy(dst, makeText())
	r := copy(dst, fromCall)
	s := copy(dst, named)
	t := copy(dst, b.text)
	u := copy(dst, b.Text())
	return n + m + p + q + r + s + t + u + fill("x")
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	var body string
	for i := 0; i < len(u.Decls); i++ {
		body += u.Decls[i].Body + "\n"
	}
	for _, leaked := range []string{
		"copy(dst, text)",
		"copy(dst, \"ok\")",
		"copy(dst, alias)",
		"copy(dst, rtg_example_com_app_makeText())",
		"copy(dst, fromCall)",
		"copy(dst, named)",
		"copy(dst, b.text)",
		"copy(dst, rtg_example_com_app_box_Text(b))",
	} {
		if strings.Contains(body, leaked) {
			t.Fatalf("copy string source leaked as %q in lowered body:\n%s", leaked, body)
		}
	}
	for _, want := range []string{
		" := []byte(text)",
		" := []byte(\"ok\")",
		" := []byte(alias)",
		" := rtg_example_com_app_makeText()",
		" := rtg_example_com_app_box_Text(b)",
		" := []byte(rtg_example_com_app_appMain_tmp_",
		" := []byte(fromCall)",
		" := []byte(named)",
		" := []byte(b.text)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("lowered body missing %q:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersCopyImportedStringSources(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	dst := []byte{0, 0}
	text := dep.Text()
	n := copy(dst, dep.Text())
	m := copy(dst, text)
	return n + m
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

func Text() string {
	return "hi"
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	var body string
	for i := 0; i < len(u.Decls); i++ {
		body += u.Decls[i].Body + "\n"
	}
	for _, leaked := range []string{
		"copy(dst, rtg_example_com_app_dep_Text())",
		"copy(dst, text)",
	} {
		if strings.Contains(body, leaked) {
			t.Fatalf("copy imported string source leaked as %q in lowered body:\n%s", leaked, body)
		}
	}
	for _, want := range []string{
		" := rtg_example_com_app_dep_Text()",
		" := []byte(rtg_example_com_app_appMain_tmp_",
		" := []byte(text)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("lowered body missing %q:\n%s", want, body)
		}
	}
}

func TestPackageLowersInferredLocalVarDeclarations(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var values = append([]int{}, 1, 2)
	if len(values) == 2 && values[1] == 2 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "var values =") {
		t.Fatalf("inferred local var leaked into lowered body:\n%s", body)
	}
	if !strings.Contains(body, "values := rtg_example_com_app_appMain_tmp_2") {
		t.Fatalf("inferred local var was not lowered to short declaration after append temps:\n%s", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := append(rtg_example_com_app_appMain_tmp_0, 1)") || !strings.Contains(body, "rtg_example_com_app_appMain_tmp_2 := append(rtg_example_com_app_appMain_tmp_1, 2)") {
		t.Fatalf("multi-value append initializer was not normalized before inferred var:\n%s", body)
	}
}

func TestPackageLowersGroupedAndMultiLocalVarDeclarations(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func pair() (int, int) { return 7, 8 }

func appMain() int {
	var (
		a = 1
		b int = 2
		c int
	)
	var d, e = 3, 4
	var f, g int = 5, 6
	var h, i int = pair()
	return a + b + c + d + e + f + g + h + i
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[len(u.Decls)-1].Body
	if strings.Contains(body, "var (") || strings.Contains(body, "var d, e") || strings.Contains(body, "var f, g") {
		t.Fatalf("grouped or multi local var leaked into lowered body:\n%s", body)
	}
	for _, want := range []string{
		"a := 1",
		"var b int = 2",
		"var c int",
		"d, e := 3, 4",
		"var f int = 5",
		"var g int = 6",
		"var h int",
		"var i int",
		"h, i = rtg_example_com_app_pair()",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("lowered body missing %q:\n%s", want, body)
		}
	}
}

func TestPackageLowersAddressOfCompositeLiteral(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct { value int }

func appMain() int {
	p := &box{value: 7}
	return p.value - 7
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want box and appMain", u.Decls)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "&rtg_example_com_app_box{") {
		t.Fatalf("address-of composite literal leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_box{value: 7}") || !strings.Contains(body, "p := &rtg_example_com_app_appMain_tmp_0") {
		t.Fatalf("address-of composite literal was not lowered through a temp: %q", body)
	}
}

func TestPackageLowersMethodsToFunctions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Point struct { x int; y int }

func (p Point) Sum() int {
	return p.x + p.y
}

func Use() int {
	p := Point{x: 3, y: 4}
	return p.Sum()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[1].Name != "Point_Sum" || u.Decls[1].UnitName != "rtg_example_com_app_pkg_Point_Sum" {
		t.Fatalf("method metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_Point_Sum(p rtg_example_com_app_pkg_Point) int") {
		t.Fatalf("method declaration was not lowered: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "p := rtg_example_com_app_pkg_Point{x: 3, y: 4}") {
		t.Fatalf("method receiver local type was not rewritten: %q", u.Decls[2].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "return rtg_example_com_app_pkg_Point_Sum(p)") {
		t.Fatalf("method call was not lowered: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersDirectMethodExpressionCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method_expr.go",
				Source: []byte(`package pkg

type Point struct { x int; y int }

func (p Point) Sum() int {
	return p.x + p.y
}

func Use() int {
	return Point.Sum(Point{x: 3, y: 4})
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "Point.Sum") {
		t.Fatalf("method expression call leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_pkg_Point_Sum(rtg_example_com_app_pkg_Point{x: 3, y: 4})") {
		t.Fatalf("method expression call was not lowered: %q", body)
	}
}

func TestPackageLowersStaticMethodExpressionAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method_expr_alias.go",
				Source: []byte(`package pkg

type Point struct { x int; y int }

func (p Point) Sum() int {
	return p.x + p.y
}

func (p Point) Add(v int) int {
	return p.x + p.y + v
}

func Use() int {
	f := Point.Sum
	var g = Point.Add
	_ = Point.Sum
	_ = f
	return f(Point{x: 3, y: 4}) + g(Point{x: 5, y: 6}, 24) - 42
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[len(u.Decls)-1].Body
	for _, bad := range []string{"Point.Sum", "Point.Add", "f := ", "var g", "_ = f", "f(", "g("} {
		if strings.Contains(body, bad) {
			t.Fatalf("method expression alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	want := "return rtg_example_com_app_pkg_Point_Sum(rtg_example_com_app_pkg_Point{x: 3, y: 4}) + rtg_example_com_app_pkg_Point_Add(rtg_example_com_app_pkg_Point{x: 5, y: 6}, 24) - 42"
	if !strings.Contains(body, want) {
		t.Fatalf("method expression alias calls were not lowered:\n%s", body)
	}
}

func TestPackageLowersStaticMethodValueAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method_value.go",
				Source: []byte(`package pkg

type Box struct { value int }

func (b Box) Value() int {
	return b.value
}

func (b *Box) Add(v int) int {
	b.value = b.value + v
	return b.value
}

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func Use() int {
	b := Box{value: 1}
	f := b.Value
	g := b.Add
	_ = b.Value
	_ = b.Add
	_ = f
	other := Box{value: 2}
	return f() + g(40) + applyValue(other.Value) + applyAdd(other.Add, 1) - 47
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "Use" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("Use decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := b.Value", "g := b.Add", "_ = b.Value", "_ = b.Add", "_ = f", "f()", "g("} {
		if strings.Contains(body, bad) {
			t.Fatalf("method value alias leaked into lowered body: %q", body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0 := b",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1 := &b",
		"rtg_example_com_app_pkg_Box_Value(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0)",
		"rtg_example_com_app_pkg_Box_Add(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1, 40)",
		"rtg_example_com_app_pkg_applyValue_callback_0(other)",
		"rtg_example_com_app_pkg_applyAdd_callback_1(&other, 1)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered method value alias fragment %q in:\n%s", want, body)
		}
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	valueCallback := bodyByName["rtg_example_com_app_pkg_applyValue_callback_0"]
	if !strings.Contains(valueCallback, "func rtg_example_com_app_pkg_applyValue_callback_0(rtg_callback_capture_0_0 rtg_example_com_app_pkg_Box) int") || !strings.Contains(valueCallback, "return rtg_example_com_app_pkg_Box_Value(rtg_callback_capture_0_0)") {
		t.Fatalf("direct value method callback specialization was not rewritten correctly:\n%s", valueCallback)
	}
	addCallback := bodyByName["rtg_example_com_app_pkg_applyAdd_callback_1"]
	if !strings.Contains(addCallback, "func rtg_example_com_app_pkg_applyAdd_callback_1(rtg_callback_capture_0_0 *rtg_example_com_app_pkg_Box, v int) int") || !strings.Contains(addCallback, "return rtg_example_com_app_pkg_Box_Add(rtg_callback_capture_0_0, v)") {
		t.Fatalf("direct pointer method callback specialization was not rewritten correctly:\n%s", addCallback)
	}
}

func TestPackageLowersStaticPromotedMethodValueAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "promoted_method_value.go",
				Source: []byte(`package pkg

type Inner struct { value int }

func (in Inner) Value() int {
	return in.value
}

func (in *Inner) Add(v int) int {
	in.value = in.value + v
	return in.value
}

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func Use() int {
	outer := Outer{Inner: Inner{value: 1}}
	pointer := PointerOuter{Inner: &Inner{value: 10}}
	f := outer.Value
	g := outer.Add
	h := pointer.Value
	k := pointer.Add
	_ = outer.Value
	_ = outer.Add
	_ = pointer.Value
	_ = pointer.Add
	_ = f
	other := Outer{Inner: Inner{value: 20}}
	otherPtr := PointerOuter{Inner: &Inner{value: 30}}
	return f() + g(2) + h() + k(5) + applyValue(other.Value) + applyAdd(other.Add, 2) + applyValue(otherPtr.Value) + applyAdd(otherPtr.Add, 3) - 134
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "Use" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("Use decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := outer.Value", "g := outer.Add", "h := pointer.Value", "k := pointer.Add", "_ = outer.Value", "_ = outer.Add", "_ = pointer.Value", "_ = pointer.Add", "_ = f", "f()", "g(", "h()", "k("} {
		if strings.Contains(body, bad) {
			t.Fatalf("promoted method value alias leaked into lowered body: %q", body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0 := outer.Inner",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1 := &outer.Inner",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_2 := *pointer.Inner",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_3 := pointer.Inner",
		"rtg_example_com_app_pkg_Inner_Value(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0)",
		"rtg_example_com_app_pkg_Inner_Add(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1, 2)",
		"rtg_example_com_app_pkg_Inner_Value(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_2)",
		"rtg_example_com_app_pkg_Inner_Add(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_3, 5)",
		"rtg_example_com_app_pkg_applyValue_callback_0(other.Inner)",
		"rtg_example_com_app_pkg_applyAdd_callback_1(&other.Inner, 2)",
		"rtg_example_com_app_pkg_applyValue_callback_0(*otherPtr.Inner)",
		"rtg_example_com_app_pkg_applyAdd_callback_1(otherPtr.Inner, 3)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered promoted method value alias fragment %q in:\n%s", want, body)
		}
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	valueCallback := bodyByName["rtg_example_com_app_pkg_applyValue_callback_0"]
	if !strings.Contains(valueCallback, "func rtg_example_com_app_pkg_applyValue_callback_0(rtg_callback_capture_0_0 rtg_example_com_app_pkg_Inner) int") || !strings.Contains(valueCallback, "return rtg_example_com_app_pkg_Inner_Value(rtg_callback_capture_0_0)") {
		t.Fatalf("direct promoted value method callback specialization was not rewritten correctly:\n%s", valueCallback)
	}
	addCallback := bodyByName["rtg_example_com_app_pkg_applyAdd_callback_1"]
	if !strings.Contains(addCallback, "func rtg_example_com_app_pkg_applyAdd_callback_1(rtg_callback_capture_0_0 *rtg_example_com_app_pkg_Inner, v int) int") || !strings.Contains(addCallback, "return rtg_example_com_app_pkg_Inner_Add(rtg_callback_capture_0_0, v)") {
		t.Fatalf("direct promoted pointer method callback specialization was not rewritten correctly:\n%s", addCallback)
	}
}

func TestPackageLowersStaticCompositeLiteralMethodValueAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "composite_method_value.go",
				Source: []byte(`package pkg

type Box struct { value int }
func (b Box) Value() int { return b.value }
func (b *Box) Add(v int) int { b.value = b.value + v; return b.value }

type Inner struct { value int }
func (in Inner) InnerValue() int { return in.value }
func (in *Inner) InnerAdd(v int) int { in.value = in.value + v; return in.value }

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func Use() int {
	f := Box{value: 1}.Value
	g := (&Box{value: 2}).Add
	h := Outer{Inner: Inner{value: 3}}.InnerValue
	k := PointerOuter{Inner: &Inner{value: 4}}.InnerAdd
	m := (&Outer{Inner: Inner{value: 5}}).InnerAdd
	_ = Box{value: 9}.Value
	_ = (&Box{value: 10}).Add
	_ = Outer{Inner: Inner{value: 11}}.InnerValue
	_ = f
	return f() + g(3) + h() + k(6) + m(7) + applyValue(f) + applyAdd(g, 4) + applyValue(Box{value: 6}.Value) + applyAdd((&Box{value: 7}).Add, 2) + applyValue(Outer{Inner: Inner{value: 8}}.InnerValue) + applyAdd((&Outer{Inner: Inner{value: 9}}).InnerAdd, 2) + applyValue(PointerOuter{Inner: &Inner{value: 10}}.InnerValue) + applyAdd(PointerOuter{Inner: &Inner{value: 11}}.InnerAdd, 3) - 99
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	body := bodyByName["Use"]
	if body == "" {
		t.Fatalf("Use decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"Box{value: 1}.Value", "Box{value: 2}).Add", "Box{value: 9}.Value", "Box{value: 10}).Add", "}.InnerValue", "}.InnerAdd", "_ = f", "f()", "g(", "h()", "k(", "m(", "applyValue(f", "applyAdd(g"} {
		if strings.Contains(body, bad) {
			t.Fatalf("composite method value alias leaked as %q in:\n%s", bad, body)
		}
	}
	if strings.Contains(body, "*(rtg_example_com_app_pkg_PointerOuter{") {
		t.Fatalf("promoted pointer-field composite callback receiver was not normalized into a temp:\n%s", body)
	}
	for _, want := range []string{
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0 := rtg_example_com_app_pkg_Box{value: 1}",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1 := &",
		"rtg_example_com_app_pkg_Use_method_value_receiver_tmp_2 := ",
		".Inner",
		"rtg_example_com_app_pkg_Box_Value(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0)",
		"rtg_example_com_app_pkg_Box_Add(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1, 3)",
		"rtg_example_com_app_pkg_Inner_InnerValue(",
		"rtg_example_com_app_pkg_Inner_InnerAdd(",
		"rtg_example_com_app_pkg_applyValue_callback_0(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0)",
		"rtg_example_com_app_pkg_applyAdd_callback_1(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1, 4)",
		"rtg_example_com_app_pkg_applyValue_callback_0(rtg_example_com_app_pkg_Box{value: 6})",
		"rtg_example_com_app_pkg_Use_tmp_5 := rtg_example_com_app_pkg_Box{value: 7}",
		"rtg_example_com_app_pkg_applyAdd_callback_1(&rtg_example_com_app_pkg_Use_tmp_5, 2)",
		"rtg_example_com_app_pkg_applyValue_callback_2(",
		"rtg_example_com_app_pkg_applyAdd_callback_3(",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered composite method value alias fragment %q in:\n%s", want, body)
		}
	}
	valueCallback := bodyByName["rtg_example_com_app_pkg_applyValue_callback_0"]
	if !strings.Contains(valueCallback, "func rtg_example_com_app_pkg_applyValue_callback_0(rtg_callback_capture_0_0 rtg_example_com_app_pkg_Box) int") || !strings.Contains(valueCallback, "return rtg_example_com_app_pkg_Box_Value(rtg_callback_capture_0_0)") {
		t.Fatalf("composite value method callback specialization was not rewritten correctly:\n%s", valueCallback)
	}
	addCallback := bodyByName["rtg_example_com_app_pkg_applyAdd_callback_1"]
	if !strings.Contains(addCallback, "func rtg_example_com_app_pkg_applyAdd_callback_1(rtg_callback_capture_0_0 *rtg_example_com_app_pkg_Box, v int) int") || !strings.Contains(addCallback, "return rtg_example_com_app_pkg_Box_Add(rtg_callback_capture_0_0, v)") {
		t.Fatalf("composite pointer method callback specialization was not rewritten correctly:\n%s", addCallback)
	}
	promotedValueCallback := bodyByName["rtg_example_com_app_pkg_applyValue_callback_2"]
	if !strings.Contains(promotedValueCallback, "func rtg_example_com_app_pkg_applyValue_callback_2(rtg_callback_capture_0_0 rtg_example_com_app_pkg_Inner) int") || !strings.Contains(promotedValueCallback, "return rtg_example_com_app_pkg_Inner_InnerValue(rtg_callback_capture_0_0)") {
		t.Fatalf("composite promoted value method callback specialization was not rewritten correctly:\n%s", promotedValueCallback)
	}
	promotedAddCallback := bodyByName["rtg_example_com_app_pkg_applyAdd_callback_3"]
	if !strings.Contains(promotedAddCallback, "func rtg_example_com_app_pkg_applyAdd_callback_3(rtg_callback_capture_0_0 *rtg_example_com_app_pkg_Inner, v int) int") || !strings.Contains(promotedAddCallback, "return rtg_example_com_app_pkg_Inner_InnerAdd(rtg_callback_capture_0_0, v)") {
		t.Fatalf("composite promoted pointer method callback specialization was not rewritten correctly:\n%s", promotedAddCallback)
	}
}

func TestPackageLowersMethodCallsOnCompositeLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Point struct { x int; y int }

func (p Point) Sum() int {
	return p.x + p.y
}

func Use() int {
	if (Point{x: 3, y: 4}).Sum() != 7 {
		return 1
	}
	return Point{x: 5, y: 6}.Sum() - 11
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, ".Sum()") {
		t.Fatalf("composite literal method call was not lowered: %q", body)
	}
	for _, want := range []string{
		"rtg_example_com_app_pkg_Point_Sum(rtg_example_com_app_pkg_Point{x: 3, y: 4})",
		"return rtg_example_com_app_pkg_Point_Sum(rtg_example_com_app_pkg_Point{x: 5, y: 6}) - 11",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered method call %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersPointerReceiverMethodCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Counter struct { n int }

func (c *Counter) Add(v int) {
	c.n = c.n + v
}

func Use() int {
	var c Counter
	c.Add(5)
	return c.n
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if u.Decls[1].Name != "Counter_Add" || u.Decls[1].UnitName != "rtg_example_com_app_pkg_Counter_Add" {
		t.Fatalf("method metadata = %#v", u.Decls[1])
	}
	if !strings.Contains(u.Decls[1].Body, "func rtg_example_com_app_pkg_Counter_Add(c *rtg_example_com_app_pkg_Counter, v int)") {
		t.Fatalf("pointer receiver declaration was not lowered: %q", u.Decls[1].Body)
	}
	if !strings.Contains(u.Decls[2].Body, "rtg_example_com_app_pkg_Counter_Add(&c, 5)") {
		t.Fatalf("pointer receiver call was not lowered: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersPointerMethodCallsOnAddressedCompositeLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Counter struct { n int }

func (c *Counter) Value() int {
	return c.n
}

func Use() int {
	return (&Counter{n: 42}).Value()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, ".Value()") {
		t.Fatalf("addressed composite literal method call was not lowered: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_pkg_Use_tmp_0 := rtg_example_com_app_pkg_Counter{n: 42}") || !strings.Contains(body, "return rtg_example_com_app_pkg_Counter_Value(&rtg_example_com_app_pkg_Use_tmp_0)") {
		t.Fatalf("pointer receiver composite method call was not lowered: %q", body)
	}
}

func TestPackageLowersMethodCallsOnIndexedReceivers(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Box struct { value int }

func (b Box) Value() int {
	return b.value
}

func (b *Box) Add(v int) {
	b.value = b.value + v
}

func Use() int {
	items := []Box{{value: 40}, {value: 2}}
	items[0].Add(1)
	return items[0].Value() + []Box{{value: 1}}[0].Value()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want Box, Value, Add, Use", u.Decls)
	}
	body := u.Decls[3].Body
	if strings.Contains(body, ".Value()") || strings.Contains(body, ".Add(") {
		t.Fatalf("indexed receiver method call leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_pkg_Box_Add(&items[0], 1)") {
		t.Fatalf("pointer method on indexed receiver was not lowered: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_pkg_Use_tmp_0 := []rtg_example_com_app_pkg_Box{rtg_example_com_app_pkg_Box{value: 1}}") || !strings.Contains(body, "return rtg_example_com_app_pkg_Box_Value(items[0]) + rtg_example_com_app_pkg_Box_Value(rtg_example_com_app_pkg_Use_tmp_0[0])") {
		t.Fatalf("value method on indexed receivers was not lowered: %q", body)
	}
}

func TestPackageLowersMethodCallsOnIndexedPointerSliceLiteralReceivers(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Box struct { value int }

func (b Box) Value() int {
	return b.value
}

func Use() int {
	return []*Box{&Box{value: 42}}[0].Value()
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want Box, Value, Use", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, ".Value()") || strings.Contains(body, "&Box{") {
		t.Fatalf("indexed pointer-slice receiver was not fully lowered: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_pkg_Use_tmp_0 := rtg_example_com_app_pkg_Box{value: 42}") || !strings.Contains(body, "rtg_example_com_app_pkg_Use_tmp_1 := []*rtg_example_com_app_pkg_Box{&rtg_example_com_app_pkg_Use_tmp_0}") {
		t.Fatalf("addressed element literal was not normalized with lowered types: %q", body)
	}
}

func TestPackageLowersValueReceiverMethodCallsOnPointers(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method.go",
				Source: []byte(`package pkg

type Items []int

func (items Items) Add(v int) Items {
	return append(items, v)
}

func Fill(items *Items) {
	*items = items.Add(5)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	if !strings.Contains(u.Decls[2].Body, "*items = rtg_example_com_app_pkg_Items_Add(*items, 5)") {
		t.Fatalf("value receiver pointer call was not lowered with dereference: %q", u.Decls[2].Body)
	}
}

func TestPackageLowersSamePackageTypesInFunctionSignatures(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "types.go",
				Source: []byte(`package pkg

type Item struct { Name string }
type Items []Item

func Copy(out Items, values Items) Items {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want 3", u.Decls)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "out Items") || strings.Contains(body, "values Items") || strings.Contains(body, ") Items") {
		t.Fatalf("function signature retained source type names: %q", body)
	}
	if !strings.Contains(body, "out rtg_example_com_app_pkg_Items, values rtg_example_com_app_pkg_Items") {
		t.Fatalf("parameter types were not lowered: %q", body)
	}
	if !strings.Contains(body, ") rtg_example_com_app_pkg_Items") {
		t.Fatalf("result type was not lowered: %q", body)
	}
}

func TestPackageDoesNotRewriteCompositeFieldKeys(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "keys.go",
				Source: []byte(`package pkg

type Unit struct { Package string }
type Input struct { Name string }

func Package(input Input) Unit {
	return Unit{Package: input.Name}
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_pkg_Unit{Package: input.Name}") {
		t.Fatalf("composite field key was rewritten: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Package:") {
		t.Fatalf("composite field key contains unit name: %q", body)
	}
}

func TestPackageLowersImplicitCompositeElements(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "implicit.go",
				Source: []byte(`package pkg

type Pair struct {
	Left int
	Right int
}

func Sum() int {
	pairs := []Pair{{Left: 1, Right: 2}, {3, 4}}
	return pairs[0].Left + pairs[1].Right
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want Pair and Sum", u.Decls)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "[]rtg_example_com_app_pkg_Pair{{") {
		t.Fatalf("implicit composite elements leaked into lowered body: %q", body)
	}
	for _, want := range []string{
		"[]rtg_example_com_app_pkg_Pair{rtg_example_com_app_pkg_Pair{Left: 1, Right: 2}, rtg_example_com_app_pkg_Pair{3, 4}}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered composite %q in:\n%s", want, body)
		}
	}
}

func TestPackageNormalizesCompositeLiteralValueExpressions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct {
	values []int
}

func appMain() int {
	b := box{values: append([]int{}, 1, 2)}
	boxes := []box{{values: append([]int{}, 3, 4)}}
	return len(b.values) + len(boxes[0].values) - 4
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want box and appMain", u.Decls)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "append([]int{}, 1, 2)") || strings.Contains(body, "append([]int{}, 3, 4)") {
		t.Fatalf("multi-value append leaked inside composite literal: %q", body)
	}
	for _, want := range []string{
		"append(",
		"values:",
		"rtg_example_com_app_box{",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing expected lowered fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageNormalizesKeyedSliceLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type scores []int

var global = []int{2: 9}

func appMain() int {
	values := []int{0: 1, 2: 3}
	mixed := []int{1, 3: 4, 5}
	named := scores{1: 7}
	return len(values) + len(mixed) + len(named) - 10
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want scores, global, and appMain", u.Decls)
	}
	globalBody := u.Decls[1].Body
	if strings.Contains(globalBody, "2:") {
		t.Fatalf("package keyed slice literal leaked into lowered body: %q", globalBody)
	}
	if !strings.Contains(globalBody, "[]int{0, 0, 9}") {
		t.Fatalf("missing lowered package keyed slice literal in:\n%s", globalBody)
	}
	body := u.Decls[2].Body
	for _, bad := range []string{"0:", "1:", "2:", "3:"} {
		if strings.Contains(body, bad) {
			t.Fatalf("keyed slice literal leaked into lowered body: %q", body)
		}
	}
	for _, want := range []string{
		"[]int{1, 0, 3}",
		"[]int{1, 0, 0, 4, 5}",
		"rtg_example_com_app_scores{0, 7}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered keyed slice literal %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLocalArrayForms(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	var zero [3]int
	values := [3]int{1, 2}
	keyed := [4]int{1: 7, 3: 9}
	inferred := [...]int{5, 6}
	var explicit [2]int = [2]int{8}
	return len(zero) + values[2] + keyed[1] + keyed[3] + len(inferred) + explicit[1] - 21
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "[4]int", "[2]int", "[...]int", "1:", "3:"} {
		if strings.Contains(body, bad) {
			t.Fatalf("array form leaked into lowered body: %q", body)
		}
	}
	for _, want := range []string{
		"zero := []int{0, 0, 0}",
		"values := []int{1, 2, 0}",
		"keyed := []int{0, 7, 0, 9}",
		"inferred := []int{5, 6}",
		"var explicit []int = []int{8, 0}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered array fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLocalArrayAssignmentCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	original := [3]int{1, 2, 3}
	copy := original
	copy[0] = 9
	if original[0] == 1 && copy[0] == 9 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "copy := original"} {
		if strings.Contains(body, bad) {
			t.Fatalf("array copy form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"original := []int{1, 2, 3}",
		"rtg_example_com_app_appMain_tmp_0 := make([]int, 0, 3)",
		"copy := append(rtg_example_com_app_appMain_tmp_0, original...)",
		"copy[0] = 9",
		"if original[0] == 1",
		"if copy[0] == 9",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered array copy fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageErasesDiscardedArrayLiteralAssignments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = [3]int{1, -2, +3}
	_, _ = [...]int{1: 2, 3: 4}, [2]string{"a", "b"}
	_ = [2][2]int{{1, 2}, {3, 4}}
	_ = [][2]int{{1, 2}, {3, 4}}
	_ = [3]int{first(), -2, second()}
	_, _ = [2]int{1: third()}, [2][1]int{{fourth()}, {5}}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "[...]int", "[2]string", "[2][2]int", "[][2]int", `{"a", "b"}`, "1:"} {
		if strings.Contains(body, bad) {
			t.Fatalf("discarded array literal leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	if !strings.Contains(body, "return 0") {
		t.Fatalf("return statement missing after array discard lowering:\n%s", body)
	}
	for _, want := range []string{"first()", "second()", "third()", "fourth()"} {
		if !strings.Contains(body, want) {
			t.Fatalf("discarded array literal side effect %q missing from lowered body:\n%s", want, body)
		}
	}
}

func TestPackageErasesDiscardedMapLiteralAssignments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = map[string]int{"a": 1, "b": -2}
	_, _ = map[int]string{1: "a"}, map[string]map[string]int{"outer": map[string]int{"inner": 7}}
	_ = []map[string]int{{"a": 1}, {"b": -2}, nil}
	_ = [2]map[int]string{{1: "a"}, map[int]string{2: "b"}}
	_ = []map[string]int{make(map[string]int, 4)}
	_ = make(map[string]int)
	_ = make(map[string]int, 4)
	_ = (make(map[byte]string, 0x10))
	delete(map[string]int{"a": 1, "b": -2}, "a")
	delete((map[byte]string{'x': "ok"}), 'x')
	delete(make(map[string]int), "a")
	delete(make(map[string]int, 4), "a")
	delete((make(map[byte]string, 0x10)), 'x')
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"map[", "_ =", "make", "delete", `"outer"`, `"inner"`, `"a":`, "1:"} {
		if strings.Contains(body, bad) {
			t.Fatalf("discarded map literal leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	if !strings.Contains(body, "return 0") {
		t.Fatalf("return statement missing after map discard lowering:\n%s", body)
	}
}

func TestPackageErasesDiscardedMapLiteralsAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }
func text() string { return "x" }

func appMain() int {
	_ = map[string]int{"a": first(), "b": second(2)}
	_, _ = map[int]string{1: text()}, map[string]map[string]int{"outer": map[string]int{"inner": third()}}
	_ = []map[string]int{{"a": first()}, {"b": second(2)}, map[string]int{"c": third()}}
	_ = []map[string]int{make(map[string]int, fourth())}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `"outer"`, `"inner"`, `"a":`, "1:"} {
		if strings.Contains(body, bad) {
			t.Fatalf("discarded side-effect map literal leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first()",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_text()",
		"rtg_example_com_app_third()",
		"rtg_example_com_app_fourth()",
		"return 0",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing discarded map side effect %q in lowered body:\n%s", want, body)
		}
	}
}

func TestPackageLowersMapMakeAfterPreservingDirectCallCapacity(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }
func sixth(v int) int { return v }
func seventh(v int) int { return v }

func appMain() int {
	_ = make(map[string]int, first(1))
	delete(make(map[string]int, second(2)), "a")
	total := len(make(map[string]int, third(3)))
	total = total + make(map[string]int, fourth(4))["missing"]
	value, ok := make(map[string]int, fifth(5))["missing"]
	for k, v := range make(map[string]int, sixth(6)) {
		total = total + len(k) + v
	}
	fn := func() int {
		return len(make(map[string]int, seventh(7)))
	}
	if !ok {
		return total + value + fn()
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"make(", "map[", "delete"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("make(map) leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_third(3)",
		"rtg_example_com_app_fourth(4)",
		"rtg_example_com_app_fifth(5)",
		"rtg_example_com_app_sixth(6)",
		"total := 0",
		"total = total + 0",
		"value, ok := 0, false",
		"return total + value + rtg_example_com_app_appMain_func_literal_0()",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered make(map) fragment %q in app body:\n%s", want, appBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"make(", "map["} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("make(map) leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_seventh(7)",
		"return 0",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered make(map) fragment %q in function literal body:\n%s", want, literalBody)
		}
	}
}

func TestPackageErasesMapLiteralDeleteAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }

func appMain() int {
	delete(map[string]int{"a": first(), "b": second(2)}, "a")
	delete(map[string]map[string]int{"outer": map[string]int{"inner": third()}}, "outer")
	fn := func() {
		delete(map[string]int{"inner": fourth()}, "inner")
	}
	fn()
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"delete", "map[", `"outer"`, `"inner"`, `"a":`} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("map literal delete leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first()",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_third()",
		"rtg_example_com_app_appMain_func_literal_0()",
		"return 0",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered map delete fragment %q in app body:\n%s", want, appBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"delete", "map[", `"inner"`} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("map literal delete leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	if !strings.Contains(literalBody, "rtg_example_com_app_fourth()") {
		t.Fatalf("missing function literal delete side effect in:\n%s", literalBody)
	}
}

func TestPackageLowersLenOfMapLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	total := len(map[string]int{"a": 1, "b": -2})
	total = total + len(map[string]map[string]int{"outer": map[string]int{"inner": 7}})
	total = total + len((map[byte]string{'x': "ok"}))
	total = total + len(make(map[string]int))
	total = total + len(make(map[string]int, 4))
	total = total + len((make(map[byte]string, 0x10)))
	return total - 4
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"map[", `len(`, "make(", `"outer"`, `"inner"`} {
		if strings.Contains(body, bad) {
			t.Fatalf("map literal len leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"total := 2",
		"total = total + 1",
		"total = total + 0",
		"return total - 4",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered map len fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLenOfMapLiteralsAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }

func appMain() int {
	total := len(map[string]int{"a": first(), "b": second(2)})
	total = total + len(map[string]map[string]int{"outer": map[string]int{"inner": third()}})
	total = total + len(map[string][2]int{"array": [2]int{fourth(), 5}})
	return total - 3
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `len(`, `"outer"`, `"inner"`, `"array"`, `"a":`} {
		if strings.Contains(body, bad) {
			t.Fatalf("side-effecting map literal len leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first()",
		"rtg_example_com_app_second(2)",
		"total := 2",
		"rtg_example_com_app_third()",
		"total = total + 1",
		"rtg_example_com_app_fourth()",
		"return total - 3",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered side-effecting map len fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersLenOfMapLiteralInsideFunctionLiteral(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func step() int { return 1 }

func appMain() int {
	f := func() int {
		return len(map[string]int{"a": step()})
	}
	return f() - 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	found := false
	for _, decl := range u.Decls {
		if !strings.Contains(decl.UnitName, "func_literal") {
			continue
		}
		found = true
		for _, bad := range []string{"map[", `len(`, `"a":`} {
			if strings.Contains(decl.Body, bad) {
				t.Fatalf("function literal map len leaked into generated body as %q:\n%s", bad, decl.Body)
			}
		}
		for _, want := range []string{
			"rtg_example_com_app_step()",
			"return 1",
		} {
			if !strings.Contains(decl.Body, want) {
				t.Fatalf("missing generated function literal fragment %q in:\n%s", want, decl.Body)
			}
		}
	}
	if !found {
		t.Fatalf("generated function literal decl missing from %#v", u.Decls)
	}
}

func TestPackageLowersMapLiteralIndexExpressions(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	total := map[string]int{"a": 1, "b": -2}["a"]
	total = total + map[int]int{1: 3, 2: 4}[2]
	total = total + map[string]int{"present": 9}["missing"]
	if map[bool]bool{true: true, false: false}[true] {
		total = total + 5
	}
	if !map[string]bool{"yes": true}["missing"] {
		total = total + 6
	}
	text := (map[byte]string{'x': "ok"})['x']
	empty := map[string]string{"filled": "bad"}["missing"]
	var p *int = map[string]*int{"p": nil}["missing"]
	var xs []int = map[string][]int{"xs": nil}["missing"]
	if p == nil && xs == nil {
		total = total + 7
	}
	found, foundOK := map[string]int{"hit": 8}["hit"]
	missing, missingOK := map[string]int{"hit": 8}["missing"]
	missingSlice, missingSliceOK := map[string][]int{"xs": nil}["missing"]
	if foundOK && !missingOK && !missingSliceOK && found == 8 && missing == 0 && missingSlice == nil {
		total = total + 9
	}
	total = total + make(map[string]int)["missing"]
	if !make(map[string]bool)["missing"] {
		total = total + 2
	}
	emptyMake := make(map[string]string)["missing"]
	emptyMakeParen := (make(map[byte]string, 0x10))['x']
	var p2 *int = make(map[string]*int)["missing"]
	var xs2 []int = make(map[string][]int, 4)["missing"]
	made, madeOK := make(map[string]int)["missing"]
	if !madeOK && made == 0 && p2 == nil && xs2 == nil {
		total = total + 3
	}
	return total + len(text) + len(empty) + len(emptyMake) + len(emptyMakeParen) - 39
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"map[", "make(", `"a":`, `"b":`, `"yes":`, `"filled":`, "'x':"} {
		if strings.Contains(body, bad) {
			t.Fatalf("map literal index leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"total := 1",
		"total = total + 4",
		"total = total + 0",
		"if true {",
		"if !false {",
		`text := "ok"`,
		`empty := ""`,
		"var p *int = nil",
		"var xs []int = nil",
		"found, foundOK := 8, true",
		"missing, missingOK := 0, false",
		"missingSlice, missingSliceOK := nil, false",
		`emptyMake := ""`,
		`emptyMakeParen := ""`,
		"var p2 *int = nil",
		"var xs2 []int = nil",
		"made, madeOK := 0, false",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered map index fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersMapLiteralIndexAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }

func appMain() int {
	total := map[string]int{"miss": first(1), "hit": second(2), "tail": third(3)}["hit"]
	total = total + map[string]int{"miss": fourth(4)}["missing"]
	fn := func() int {
		return map[string]int{"inner": fifth(5)}["inner"]
	}
	return total + fn() - 7
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `"miss"`, `"hit"`, `"tail"`} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("side-effecting map index leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_third(3)",
		"rtg_example_com_app_fourth(4)",
		"total = total + 0",
		"return total + rtg_example_com_app_appMain_func_literal_0() - 7",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered map index fragment %q in app body:\n%s", want, appBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `"inner"`} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("side-effecting map index leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_fifth(5)",
		"return rtg_example_com_app_appMain_func_literal_0_tmp_0",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered map index fragment %q in function literal body:\n%s", want, literalBody)
		}
	}
}

func TestPackageLowersMapLiteralCommaOkAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }
func sixth(v int) int { return v }

func appMain() int {
	found, ok := map[string]int{"miss": first(1), "hit": second(2), "tail": third(3)}["hit"]
	missing, missingOK := map[string]int{"miss": fourth(4)}["missing"]
	if cond, condOK := map[string]int{"cond": fifth(5)}["cond"]; condOK {
		found = found + cond
	}
	fn := func() int {
		value, valueOK := map[string]int{"inner": sixth(6)}["inner"]
		if valueOK {
			return value
		}
		return 0
	}
	if ok && !missingOK {
		return found + missing + fn() - 13
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `"miss"`, `"hit"`, `"tail"`, `"cond"`} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("side-effecting map comma-ok leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_third(3)",
		"rtg_example_com_app_fourth(4)",
		"rtg_example_com_app_fifth(5)",
		"found, ok :=",
		", true",
		"missing, missingOK := 0, false",
		"cond, condOK :=",
		"if condOK",
		"return found + missing + rtg_example_com_app_appMain_func_literal_0() - 13",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered map comma-ok fragment %q in app body:\n%s", want, appBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", `"inner"`} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("side-effecting map comma-ok leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_sixth(6)",
		"value, valueOK :=",
		", true",
		"if valueOK",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered map comma-ok fragment %q in function literal body:\n%s", want, literalBody)
		}
	}
}

func TestPackageLowersPureMapRange(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	total := 0
	for k, v := range map[string]int{"a": 1, "bb": 2} {
		total = total + len(k) + v
	}
	for k := range map[byte]int{'x': 4} {
		total = total + int(k)
	}
	for _, v := range map[int]string{1: "a", 2: "bb"} {
		total = total + len(v)
	}
	for _, v := range make(map[string]int) {
		total = total + v
	}
	for k, v := range make(map[string]string, 4) {
		total = total + len(k) + len(v)
	}
	return total - 129
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"range", "map[", "make(", `"a":`, `"bb":`, "'x':"} {
		if strings.Contains(body, bad) {
			t.Fatalf("map range leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		`[]string{"a", "bb"}`,
		"[]byte{'x'}",
		`var rtg_example_com_app_appMain_tmp_`,
		" = 1",
		" = 2",
		"for rtg_example_com_app_appMain_tmp_",
		" < 0; ",
		"return total - 129",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered map range fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersMapRangeAfterPreservingDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }

func appMain() int {
	total := 0
	for k, v := range map[string]int{"a": first(1), "bb": second(2)} {
		total = total + len(k) + v
	}
	for k := range map[string]int{"ccc": third(3)} {
		total = total + len(k)
	}
	fn := func() int {
		for _, v := range map[string]int{"d": fourth(4)} {
			return v
		}
		return 0
	}
	return total + fn() - 14
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"range", "map["} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("side-effecting map range leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_second(2)",
		"rtg_example_com_app_third(3)",
		`[]string{"a", "bb"}`,
		`[]string{"ccc"}`,
		"return total + rtg_example_com_app_appMain_func_literal_0() - 14",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered map range fragment %q in app body:\n%s", want, appBody)
		}
	}
	firstCall := strings.Index(appBody, "rtg_example_com_app_first(1)")
	firstLoop := strings.Index(appBody, "for rtg_example_com_app_appMain_tmp_")
	if firstCall < 0 || firstLoop < 0 || firstCall > firstLoop {
		t.Fatalf("map range value calls were not emitted before the loop:\n%s", appBody)
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"range", "map["} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("side-effecting map range leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_fourth(4)",
		"var rtg_example_com_app_appMain_func_literal_0_tmp_",
		"return v",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered map range fragment %q in function literal body:\n%s", want, literalBody)
		}
	}
}

func TestPackageLowersPureStaticMapAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	m := map[string]int{"a": 1, "bb": 2}
	empty := make(map[string]string)
	_ = m
	_ = empty
	total := len(m) + m["a"] + len(empty)
	delete(m, "a")
	total = total + len(m) + m["a"]
	deleted, deletedOK := m["a"]
	if !deletedOK && deleted == 0 {
		total = total + 5
	}
	m["ccc"] = 4
	m["bb"] = 5
	empty["x"] = "yz"
	total = total + len(m) + m["ccc"]
	v, ok := m["bb"]
	if ok {
		total = total + v
	}
	missing, missingOK := empty["x"]
	if missingOK {
		total = total + len(missing)
	}
	for k, v := range m {
		total = total + len(k) + v
	}
	var made = make(map[string]int, 4)
	_ = (made)
	total = total + len(made) + made["z"]
	for _, v := range made {
		total = total + v
	}
	return total - 36
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"map[", "make(", "range", "delete(", "m :=", "empty :=", "var made", "_ =", "len(m)", `m["`} {
		if strings.Contains(body, bad) {
			t.Fatalf("static map alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"total := 2 + 1 + 0",
		"total = total + 1 + 0",
		"deleted, deletedOK := 0, false",
		"total = total + 2 + 4",
		"v, ok := 5, true",
		`missing, missingOK := "yz", true`,
		`[]string{"bb", "ccc"}`,
		"total = total + 0 + 0",
		"return total - 36",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static map alias fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersNamedMapTypeStaticAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Table map[string]int

func appMain() int {
	m := Table{"a": 1, "bb": 2}
	total := len(m) + m["a"]
	m["ccc"] = 4
	delete(m, "a")
	value, ok := m["ccc"]
	if ok {
		total = total + value
	}
	for k, v := range m {
		total = total + len(k) + v
	}
	direct := len(Table{"x": 5}) + Table{"x": 5}["x"]
	var empty Table = Table{}
	return total + direct + len(empty) - 24
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain only", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"type Table", "Table{", "map[", "m :=", "var empty", "delete(", "range", `m["`, "len(empty)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("named map static alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"total := 2 + 1",
		"value, ok := 4, true",
		`[]string{"bb", "ccc"}`,
		"direct := 1 + 5",
		"return total + direct + 0 - 24",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered named map alias fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersImportedNamedMapTypeStaticAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	m := dep.Table{"a": 1, "bb": 2}
	total := len(m) + m["a"]
	m["ccc"] = 4
	delete(m, "a")
	value, ok := m["ccc"]
	if ok {
		total = total + value
	}
	for k, v := range m {
		total = total + len(k) + v
	}
	direct := len(dep.Table{"x": 5}) + dep.Table{"x": 5}["x"]
	var empty dep.Table = dep.Table{}
	return total + direct + len(empty) - 24
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Table map[string]int
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain only", u.Decls)
	}
	if len(u.References) != 0 {
		t.Fatalf("references = %#v, want none for erased imported named map type", u.References)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"dep.Table", "rtg_example_com_app_dep_Table{", "map[", "m :=", "var empty", "delete(", "range", `m["`, "len(empty)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported named map static alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"total := 2 + 1",
		"value, ok := 4, true",
		`[]string{"bb", "ccc"}`,
		"direct := 1 + 5",
		"return total + direct + 0 - 24",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported named map alias fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticMapAliasLimitationProbes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	_ = make(map[string]int)
	indexed := map[string]int{"a": 1}
	if indexed["a"] != 1 {
		return 1
	}
	assigned := map[string]int{}
	assigned["a"] = 1
	if assigned["a"] != 1 {
		return 2
	}
	missing := map[string]int{}
	_, missingOK := missing["a"]
	if missingOK {
		return 3
	}
	deleted := map[string]int{}
	delete(deleted, "a")
	measured := map[string]int{}
	if len(measured) != 0 {
		return 4
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"map[", "make(", "delete(", `["a"]`, "len(measured)", "_ ="} {
		if strings.Contains(body, bad) {
			t.Fatalf("static map alias probe leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"if 1 != 1",
		"if 1 != 1",
		"_, missingOK := 0, false",
		"if missingOK",
		"if 0 != 0",
		"return 0",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered map alias probe fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticMapAliasAssignmentsWithDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func text(v string) string { return v }

func appMain() int {
	local := 3
	m := map[string]int{"a": 1}
	m["b"] = first(2)
	m["c"] = local
	m["d"] = first(5) * second(6) - local
	m["e"] = (first(1) << 3) | second(1)
	flags := map[string]bool{"base": true}
	flags["match"] = first(2) == second(2)
	flags["ready"] = first(1) != second(2)
	match, matchOK := flags["match"]
	ready, readyOK := flags["ready"]
	if !(match && ready && matchOK && readyOK) {
		return 2
	}
	total := m["a"] + m["b"] + m["c"] + m["d"] + m["e"]
	value, ok := m["b"]
	for _, item := range m {
		total = total + item
	}
	fn := func() int {
		inner := map[string]int{}
		inner["x"] = second(4)
		return inner["x"]
	}
	if ok {
		return total + value + fn() - 90
	}
	return 1
}

func strings() string {
	suffix := "!"
	words := map[string]string{"a": "A"}
	words["b"] = text("B") + text("C") + suffix
	words["c"] = "D" + suffix
	got, ok := words["b"]
	if ok {
		return words["a"] + got + words["c"]
	}
	return "bad"
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "range", "\tm :=", `m["`} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("static map alias direct-call assignment leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"local := 3",
		"rtg_static_map_alias_",
		"rtg_example_com_app_first(2)",
		"rtg_example_com_app_first(5) * rtg_example_com_app_second(6) - local",
		"(rtg_example_com_app_first(1) << 3) | rtg_example_com_app_second(1)",
		"rtg_example_com_app_first(2) == rtg_example_com_app_second(2)",
		"rtg_example_com_app_first(1) != rtg_example_com_app_second(2)",
		"match, matchOK := rtg_static_map_alias_",
		"ready, readyOK := rtg_static_map_alias_",
		"if !(match && ready && matchOK && readyOK)",
		"total := 1 + rtg_static_map_alias_",
		"value, ok := rtg_static_map_alias_",
		"return total + value + rtg_example_com_app_appMain_func_literal_0() - 90",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered static map alias fragment %q in app body:\n%s", want, appBody)
		}
	}
	stringBody := bodyByName["rtg_example_com_app_strings"]
	if stringBody == "" {
		t.Fatalf("strings decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "range", "\twords :=", `words["`} {
		if strings.Contains(stringBody, bad) {
			t.Fatalf("static string map alias assignment leaked into strings body as %q:\n%s", bad, stringBody)
		}
	}
	for _, want := range []string{
		"suffix := \"!\"",
		"rtg_static_map_alias_",
		"rtg_example_com_app_text(\"B\") + rtg_example_com_app_text(\"C\") + suffix",
		"\"D\" + suffix",
		"got, ok := rtg_static_map_alias_",
		"return \"A\" + got + rtg_static_map_alias_",
	} {
		if !strings.Contains(stringBody, want) {
			t.Fatalf("missing lowered string map alias fragment %q in strings body:\n%s", want, stringBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "inner :=", `inner["`} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("static map alias direct-call assignment leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_static_map_alias_",
		"rtg_example_com_app_second(4)",
		"return rtg_static_map_alias_",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered function literal static map alias fragment %q in:\n%s", want, literalBody)
		}
	}
}

func TestPackageLowersStaticMapAliasCompoundAssignments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func inc(v int) int { return v }
func text(v string) string { return v }

func appMain() int {
	m := map[string]int{"a": 1}
	m["a"] += 2
	m["b"] += inc(3)
	m["a"] *= 4
	m["a"]++
	m["b"]--
	words := map[string]string{"x": "A"}
	words["x"] += text("B")
	words["y"] += text("C")
	value, ok := m["b"]
	got, textOK := words["x"]
	missing, missingOK := words["y"]
	if ok && textOK && missingOK && got == "AB" && missing == "C" {
		return m["a"] + value - 15
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	body := bodyByName["rtg_example_com_app_appMain"]
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "\tm :=", "\twords :=", `m["`, `words["`, "+=", "*="} {
		if strings.Contains(body, bad) {
			t.Fatalf("static map alias compound assignment leaked into body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_inc(3)",
		"rtg_example_com_app_text(\"B\")",
		"rtg_example_com_app_text(\"C\")",
		":= (1) + (2)",
		":= (0) + (rtg_static_map_alias_",
		") * (4)",
		") + (1)",
		") - (1)",
		":= (\"A\") + (rtg_static_map_alias_",
		":= (\"\") + (rtg_static_map_alias_",
		"value, ok := rtg_static_map_alias_",
		"got, textOK := rtg_static_map_alias_",
		"missing, missingOK := rtg_static_map_alias_",
		"return rtg_static_map_alias_",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered static map alias compound fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStaticMapAliasInitializersWithDirectCallValues(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }

func appMain() int {
	local := 3
	m := map[string]int{"a": first(1), "b": local}
	empty := make(map[string]int, third(3))
	total := len(empty) + m["a"] + m["b"]
	value, ok := m["a"]
	for _, item := range m {
		total = total + item
	}
	fn := func() int {
		inner := map[string]int{"x": second(4)}
		return inner["x"]
	}
	if ok {
		return total + value + fn() - 13
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := make(map[string]string)
	for _, decl := range u.Decls {
		bodyByName[decl.UnitName] = decl.Body
	}
	appBody := bodyByName["rtg_example_com_app_appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "make(", "range", "\tm :=", "\tempty :=", `m["`} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("static map alias direct-call initializer leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"local := 3",
		"rtg_static_map_alias_",
		"rtg_example_com_app_first(1)",
		"rtg_example_com_app_third(3)",
		"total := 0 + rtg_static_map_alias_",
		"value, ok := rtg_static_map_alias_",
		"return total + value + rtg_example_com_app_appMain_func_literal_0() - 13",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing lowered static map alias initializer fragment %q in app body:\n%s", want, appBody)
		}
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if literalBody == "" {
		t.Fatalf("function literal decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"map[", "\tinner :=", `inner["`} {
		if strings.Contains(literalBody, bad) {
			t.Fatalf("static map alias initializer leaked into function literal body as %q:\n%s", bad, literalBody)
		}
	}
	for _, want := range []string{
		"rtg_static_map_alias_",
		"rtg_example_com_app_second(4)",
		"return rtg_static_map_alias_",
	} {
		if !strings.Contains(literalBody, want) {
			t.Fatalf("missing lowered function literal static map alias initializer fragment %q in:\n%s", want, literalBody)
		}
	}
}

func TestPackageLowersArrayStructFields(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Values [3]int
	Bytes [2]byte
}

func appMain() int {
	b := Box{Values: [3]int{1, 2}, Bytes: [2]byte{'A'}}
	if b.Values == [3]int{1, 2} && b.Bytes != [2]byte{'B'} {
		return len(b.Values) + cap(b.Bytes) + b.Values[2] + int(b.Bytes[0]) - 70
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want Box and appMain", u.Decls)
	}
	typeBody := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "[2]byte"} {
		if strings.Contains(typeBody, bad) {
			t.Fatalf("array field type leaked into lowered type body: %q", typeBody)
		}
	}
	for _, want := range []string{
		"Values []int",
		"Bytes []byte",
	} {
		if !strings.Contains(typeBody, want) {
			t.Fatalf("missing lowered array field type %q in:\n%s", want, typeBody)
		}
	}
	body := u.Decls[1].Body
	for _, bad := range []string{"[3]int", "[2]byte"} {
		if strings.Contains(body, bad) {
			t.Fatalf("array literal leaked into lowered function body: %q", body)
		}
	}
	for _, want := range []string{
		"Values: []int{1, 2, 0}",
		"Bytes: []byte{'A', 0}",
		"len(b.Values)",
		"cap(b.Bytes)",
		"((b.Values)[0] == 1 && (b.Values)[1] == 2 && (b.Values)[2] == 0)",
		"((b.Bytes)[0] != 'B' || (b.Bytes)[1] != 0)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered array field fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersArrayParameters(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func sum(values [3]int) int {
	total := len(values) + cap(values)
	for i, v := range values {
		total = total + i + v
	}
	return total
}

func pair(left, right [2]int) int {
	return left[0] + right[1] + len(left) + cap(right)
}

func appMain() int {
	values := [3]int{1, 2}
	return sum(values) + sum([3]int{4, 5}) + pair([2]int{3}, [2]int{1: 4}) - 41
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want sum, pair, and appMain", u.Decls)
	}
	for declIndex := 0; declIndex < len(u.Decls); declIndex++ {
		body := u.Decls[declIndex].Body
		for _, bad := range []string{"[3]int", "[2]int", "1:"} {
			if strings.Contains(body, bad) {
				t.Fatalf("array parameter form leaked into lowered body: %q", body)
			}
		}
	}
	for _, want := range []string{
		"values []int",
		"left []int, right []int",
		"values := []int{1, 2, 0}",
		"sum([]int{4, 5, 0})",
		"pair([]int{3, 0}, []int{0, 4})",
	} {
		found := false
		for declIndex := 0; declIndex < len(u.Decls); declIndex++ {
			if strings.Contains(u.Decls[declIndex].Body, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing lowered array parameter fragment %q in decls %#v", want, u.Decls)
		}
	}
}

func TestPackageLowersNamedArrayParametersAndResults(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Values [3]int

func mutate(values Values) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}

func makeValues() Values {
	return Values{1, 2, 3}
}

func echo(values Values) Values {
	values[1] = 8
	return values
}

func appMain() int {
	original := Values{1, 2, 3}
	changed := mutate(original)
	values := makeValues()
	echoed := echo(original)
	if original == (Values{1, 2, 3}) && values == (Values{1, 2, 3}) && echoed == (Values{1, 8, 3}) {
		return changed - 15
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	for _, bad := range []string{"Values{", "rtg_example_com_app_Values{", "[3]int", "mutate(original)", "echo(original)"} {
		for _, decl := range u.Decls {
			if strings.Contains(decl.Body, bad) {
				t.Fatalf("named array signature form leaked into lowered body as %q:\n%s", bad, decl.Body)
			}
		}
	}
	for _, want := range []string{
		"type rtg_example_com_app_Values []int",
		"func rtg_example_com_app_mutate(values []int) int",
		"func rtg_example_com_app_makeValues() []int",
		"return []int{1, 2, 3}",
		"func rtg_example_com_app_echo(values []int) []int",
	} {
		found := false
		for _, decl := range u.Decls {
			if strings.Contains(decl.Body, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing lowered named array signature fragment %q in decls %#v", want, u.Decls)
		}
	}
	appMain := bodyByName["appMain"]
	for _, want := range []string{
		"original := []int{1, 2, 3}",
		"changed := rtg_example_com_app_mutate(",
		"echoed := rtg_example_com_app_echo(",
		"append(",
		", original...)",
		"values := rtg_example_com_app_makeValues()",
		"if (original[0] == 1 && original[1] == 2 && original[2] == 3) {",
		"if (values[0] == 1 && values[1] == 2 && values[2] == 3) {",
		"if (echoed[0] == 1 && echoed[1] == 8 && echoed[2] == 3) {",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered named array signature fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersArrayStructFieldCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Values [3]int
}

func mutate(values [3]int) int {
	values[1] = 8
	return values[1]
}

func appMain() int {
	b := Box{Values: [3]int{1, 2, 3}}
	copy := b.Values
	copy[0] = 9
	other := Box{Values: [3]int{4, 5, 6}}
	b.Values = other.Values
	other.Values[0] = 99
	changed := mutate(b.Values)
	if b.Values[0] == 4 && b.Values[1] == 5 && other.Values[0] == 99 && copy[0] == 9 && changed == 8 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want Box, mutate, and appMain", u.Decls)
	}
	appMain := u.Decls[2].Body
	for _, bad := range []string{"[3]int", "copy := b.Values", "mutate(b.Values)"} {
		if strings.Contains(appMain, bad) {
			t.Fatalf("array field copy form leaked into lowered body as %q:\n%s", bad, appMain)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_tmp_0 := b.Values",
		"copy := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_0...)",
		"copy[0] = 9",
		"rtg_example_com_app_appMain_tmp_1 := other.Values",
		"b.Values = append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_1...)",
		"rtg_example_com_app_appMain_tmp_2 := b.Values",
		"rtg_example_com_app_appMain_tmp_3 := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_2...)",
		"changed := rtg_example_com_app_mutate(rtg_example_com_app_appMain_tmp_3)",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered array field copy fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersNestedArrayStructFieldCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Inner struct {
	Values [3]int
}

type Outer struct {
	Inner Inner
}

func mutate(values [3]int) int {
	values[2] = 7
	return values[2]
}

func appMain() int {
	outer := Outer{Inner: Inner{Values: [3]int{1, 2, 3}}}
	copy := outer.Inner.Values
	copy[0] = 9
	target := Outer{}
	target.Inner.Values = outer.Inner.Values
	changed := mutate(target.Inner.Values)
	if outer.Inner.Values[0] == 1 && copy[0] == 9 && changed == 7 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want Inner, Outer, mutate, and appMain", u.Decls)
	}
	appMain := u.Decls[3].Body
	for _, bad := range []string{"[3]int", "copy := outer.Inner.Values", "target.Inner.Values = outer.Inner.Values", "mutate(target.Inner.Values)"} {
		if strings.Contains(appMain, bad) {
			t.Fatalf("nested array field copy form leaked into lowered body as %q:\n%s", bad, appMain)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_tmp_0 := outer.Inner.Values",
		"copy := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_0...)",
		"rtg_example_com_app_appMain_tmp_1 := outer.Inner.Values",
		"target.Inner.Values = append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_1...)",
		"rtg_example_com_app_appMain_tmp_2 := target.Inner.Values",
		"rtg_example_com_app_appMain_tmp_3 := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_2...)",
		"changed := rtg_example_com_app_mutate(rtg_example_com_app_appMain_tmp_3)",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered nested array field copy fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersPromotedArrayStructFieldCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Inner struct {
	Values [3]int
}

type Outer struct {
	Inner
}

func mutate(values [3]int) int {
	values[2] = 7
	return values[2]
}

func appMain() int {
	outer := Outer{Inner: Inner{Values: [3]int{1, 2, 3}}}
	copy := outer.Values
	copy[0] = 9
	target := Outer{}
	target.Values = outer.Values
	changed := mutate(target.Values)
	if outer.Values[0] == 1 && copy[0] == 9 && changed == 7 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want Inner, Outer, mutate, and appMain", u.Decls)
	}
	appMain := u.Decls[3].Body
	for _, bad := range []string{"[3]int", "copy := outer.Values", "target.Values = outer.Values", "mutate(target.Values)"} {
		if strings.Contains(appMain, bad) {
			t.Fatalf("promoted array field copy form leaked into lowered body as %q:\n%s", bad, appMain)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_tmp_0 := outer.Inner.Values",
		"copy := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_0...)",
		"rtg_example_com_app_appMain_tmp_1 := outer.Inner.Values",
		"target.Inner.Values = append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_1...)",
		"rtg_example_com_app_appMain_tmp_2 := target.Inner.Values",
		"rtg_example_com_app_appMain_tmp_3 := append(make([]int, 0, 3), rtg_example_com_app_appMain_tmp_2...)",
		"changed := rtg_example_com_app_mutate(rtg_example_com_app_appMain_tmp_3)",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered promoted array field copy fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersArrayParameterCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func mutate(values [3]int) int {
	values[0] = 9
	return values[0]
}

func appMain() int {
	original := [3]int{1, 2, 3}
	changed := mutate(original)
	if original[0] == 1 && changed == 9 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want mutate and appMain", u.Decls)
	}
	for _, bad := range []string{"[3]int", "mutate(original)"} {
		for _, decl := range u.Decls {
			if strings.Contains(decl.Body, bad) {
				t.Fatalf("array parameter copy form leaked into lowered body as %q:\n%s", bad, decl.Body)
			}
		}
	}
	appMain := u.Decls[1].Body
	for _, want := range []string{
		"original := []int{1, 2, 3}",
		"rtg_example_com_app_appMain_tmp_0 := make([]int, 0, 3)",
		"rtg_example_com_app_appMain_tmp_1 := append(rtg_example_com_app_appMain_tmp_0, original...)",
		"changed := rtg_example_com_app_mutate(rtg_example_com_app_appMain_tmp_1)",
		"if original[0] == 1",
		"if changed == 9",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered array parameter copy fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageWithGraphLowersImportedArrayParameterCopies(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	original := [3]int{1, 2, 3}
	changed := dep.Mutate(original)
	if original[0] == 1 && changed == 9 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

func Mutate(values [3]int) int {
	values[0] = 9
	return values[0]
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "dep.Mutate", "rtg_example_com_app_pkg_dep_Mutate(original)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported array parameter copy form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"original := []int{1, 2, 3}",
		"rtg_example_com_app_cmd_app_appMain_tmp_0 := make([]int, 0, 3)",
		"rtg_example_com_app_cmd_app_appMain_tmp_1 := append(rtg_example_com_app_cmd_app_appMain_tmp_0, original...)",
		"changed := rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_cmd_app_appMain_tmp_1)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported array parameter copy fragment %q in:\n%s", want, body)
		}
	}
	refFound := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/pkg/dep" && ref.Name == "Mutate" && ref.UnitName == "rtg_example_com_app_pkg_dep_Mutate" {
			refFound = true
		}
	}
	if !refFound {
		t.Fatalf("imported array parameter reference missing from %#v", u.References)
	}
}

func TestPackageWithGraphLowersImportedNamedArrayParameterCopies(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	original := dep.Values{1, 2, 3}
	changed := dep.Mutate(original)
	if original == (dep.Values{1, 2, 3}) && changed == 15 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Values [3]int

func Mutate(values Values) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"dep.Values", "rtg_example_com_app_pkg_dep_Values{", "[3]int", "dep.Mutate", "rtg_example_com_app_pkg_dep_Mutate(original)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported named array parameter copy form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"original := []int{1, 2, 3}",
		"rtg_example_com_app_cmd_app_appMain_tmp_0 := make([]int, 0, 3)",
		"rtg_example_com_app_cmd_app_appMain_tmp_1 := append(rtg_example_com_app_cmd_app_appMain_tmp_0, original...)",
		"changed := rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_cmd_app_appMain_tmp_1)",
		"if (original[0] == 1 && original[1] == 2 && original[2] == 3) {",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported named array parameter copy fragment %q in:\n%s", want, body)
		}
	}
	refFound := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/pkg/dep" && ref.Name == "Mutate" && ref.UnitName == "rtg_example_com_app_pkg_dep_Mutate" {
			refFound = true
		}
	}
	if !refFound {
		t.Fatalf("imported named array parameter reference missing from %#v", u.References)
	}
}

func TestPackageWithGraphLowersImportedTopLevelArrayValues(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	copy := dep.Values
	copy[0] = 8
	same := dep.Values == [3]int{1, 2, 3}
	changed := dep.Mutate(dep.Values)
	if same && dep.Values[0] == 1 && copy[0] == 8 && changed == 15 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

var Values [3]int = [3]int{1, 2, 3}

func Mutate(values [3]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[3]int", "dep.Values", "copy := rtg_example_com_app_pkg_dep_Values", "rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_pkg_dep_Values)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported top-level array value form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_cmd_app_appMain_tmp_0 := make([]int, 0, 3)",
		"copy := append(rtg_example_com_app_cmd_app_appMain_tmp_0, rtg_example_com_app_pkg_dep_Values...)",
		"same := (rtg_example_com_app_pkg_dep_Values[0] == 1 && rtg_example_com_app_pkg_dep_Values[1] == 2 && rtg_example_com_app_pkg_dep_Values[2] == 3)",
		"rtg_example_com_app_cmd_app_appMain_tmp_1 := make([]int, 0, 3)",
		"rtg_example_com_app_cmd_app_appMain_tmp_2 := append(rtg_example_com_app_cmd_app_appMain_tmp_1, rtg_example_com_app_pkg_dep_Values...)",
		"changed := rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_cmd_app_appMain_tmp_2)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported top-level array value fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersImportedArrayStructFieldCopies(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	dep.Reset()
	copy := dep.Global.Values
	copy[0] = 8
	target := dep.Box{}
	target.Values = dep.Global.Values
	replacement := [2]int{5, 6}
	dep.Global.Values = replacement
	same := dep.Global.Values == [2]int{5, 6}
	replacement[0] = 99
	changed := dep.Mutate(dep.Global.Values)
	if same && dep.Global.Values[0] == 5 && copy[0] == 8 && target.Values[0] == 1 && replacement[0] == 99 && changed == 13 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Box struct {
	Values [2]int
}

var Global Box

func Reset() {
	values := [2]int{1, 2}
	Global.Values = values
}

func Mutate(values [2]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[2]int", "dep.Global", "copy := rtg_example_com_app_pkg_dep_Global.Values", "target.Values = rtg_example_com_app_pkg_dep_Global.Values", "rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_pkg_dep_Global.Values)"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported array field copy form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_pkg_dep_Reset()",
		"rtg_example_com_app_cmd_app_appMain_tmp_0 := rtg_example_com_app_pkg_dep_Global.Values",
		"copy := append(make([]int, 0, 2), rtg_example_com_app_cmd_app_appMain_tmp_0...)",
		"target := rtg_example_com_app_pkg_dep_Box{}",
		"rtg_example_com_app_cmd_app_appMain_tmp_1 := rtg_example_com_app_pkg_dep_Global.Values",
		"target.Values = append(make([]int, 0, 2), rtg_example_com_app_cmd_app_appMain_tmp_1...)",
		"replacement := []int{5, 6}",
		"rtg_example_com_app_cmd_app_appMain_tmp_2 := make([]int, 0, 2)",
		"rtg_example_com_app_pkg_dep_Global.Values = append(rtg_example_com_app_cmd_app_appMain_tmp_2, replacement...)",
		"same := ((rtg_example_com_app_pkg_dep_Global.Values)[0] == 5 && (rtg_example_com_app_pkg_dep_Global.Values)[1] == 6)",
		"replacement[0] = 99",
		"rtg_example_com_app_cmd_app_appMain_tmp_3 := rtg_example_com_app_pkg_dep_Global.Values",
		"rtg_example_com_app_cmd_app_appMain_tmp_4 := append(make([]int, 0, 2), rtg_example_com_app_cmd_app_appMain_tmp_3...)",
		"changed := rtg_example_com_app_pkg_dep_Mutate(rtg_example_com_app_cmd_app_appMain_tmp_4)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported array field copy fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersArrayComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func same(left, right [2]int) bool {
	return left == right
}

func next() int { return 2 }
func leftArray() [2]int { return [2]int{1, 2} }
func rightArray() [2]int { return [2]int{1, next()} }

func appMain() int {
	left := [2]int{1, 2}
	right := [...]int{1, 3}
	literalCalls := [2]int{1, next()} == [2]int{1, 2}
	callValues := leftArray() == rightArray()
	if same(left, [2]int{1, 2}) && left == [2]int{1, 2} && left != right && [2]int{1, 2} == [...]int{1, 2} && literalCalls && callValues {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want same, next, leftArray, rightArray, and appMain", u.Decls)
	}
	sameBody := u.Decls[0].Body
	if !strings.Contains(sameBody, "return (left[0] == right[0] && left[1] == right[1])") {
		t.Fatalf("same did not lower array parameter comparison:\n%s", sameBody)
	}
	body := u.Decls[4].Body
	for _, bad := range []string{"[2]int", "[...]int"} {
		if strings.Contains(body, bad) {
			t.Fatalf("array comparison form leaked into lowered body: %q", body)
		}
	}
	for _, want := range []string{
		"left := []int{1, 2}",
		"right := []int{1, 3}",
		"(left[0] == 1 && left[1] == 2)",
		"(left[0] != right[0] || left[1] != right[1])",
		"(1 == 1 && 2 == 2)",
		"rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_next()",
		"var rtg_example_com_app_appMain_array_cmp_tmp_0 []int = []int{1, rtg_example_com_app_appMain_tmp_0}",
		"literalCalls := (rtg_example_com_app_appMain_array_cmp_tmp_0[0] == 1 && rtg_example_com_app_appMain_array_cmp_tmp_0[1] == 2)",
		"var rtg_example_com_app_appMain_array_cmp_tmp_1 []int = rtg_example_com_app_leftArray()",
		"var rtg_example_com_app_appMain_array_cmp_tmp_2 []int = rtg_example_com_app_rightArray()",
		"callValues := (rtg_example_com_app_appMain_array_cmp_tmp_1[0] == rtg_example_com_app_appMain_array_cmp_tmp_2[0] && rtg_example_com_app_appMain_array_cmp_tmp_1[1] == rtg_example_com_app_appMain_array_cmp_tmp_2[1])",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered array comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersSimpleStructComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct {
	value int
	name string
	ok bool
	ptr *int
	values [2]int
	inner innerBox
}

type innerBox struct {
	count int
	tag string
	flags [2]bool
}

var globalLeft = box{value: 1, name: "a", ok: true, ptr: nil, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
var globalRight = box{value: 1, name: "a", ok: true, ptr: nil, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}

func same(left box, right box) bool {
	return left == right
}

func appMain() int {
	x := 1
	left := box{value: 1, name: "a", ok: true, ptr: &x, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
	right := box{value: 1, name: "a", ok: true, ptr: &x, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
	localSame := (left) == (right)
	localDifferent := left != right
	globalSame := globalLeft == globalRight
	if same(left, right) && localSame && !localDifferent && globalSame {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 6 {
		t.Fatalf("decls = %#v, want box, globals, same, and appMain", u.Decls)
	}
	sameBody := u.Decls[4].Body
	if !strings.Contains(sameBody, "left.inner.count == right.inner.count") || !strings.Contains(sameBody, "left.inner.tag == right.inner.tag") || !strings.Contains(sameBody, "((left.inner.flags)[0] == (right.inner.flags)[0] && (left.inner.flags)[1] == (right.inner.flags)[1])") {
		t.Fatalf("same did not lower struct parameter comparison:\n%s", sameBody)
	}
	body := u.Decls[5].Body
	for _, bad := range []string{"(left) == (right)", "left != right", "globalLeft == globalRight"} {
		if strings.Contains(body, bad) {
			t.Fatalf("struct comparison form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"localSame := (left.value == right.value",
		"left.ptr == right.ptr",
		"((left.values)[0] == (right.values)[0] && (left.values)[1] == (right.values)[1])",
		"localDifferent := (left.value != right.value",
		"left.ptr != right.ptr",
		"((left.values)[0] != (right.values)[0] || (left.values)[1] != (right.values)[1])",
		"left.inner.count == right.inner.count",
		"left.inner.tag == right.inner.tag",
		"((left.inner.flags)[0] == (right.inner.flags)[0] && (left.inner.flags)[1] == (right.inner.flags)[1])",
		"globalSame := (rtg_example_com_app_globalLeft.value == rtg_example_com_app_globalRight.value",
		"rtg_example_com_app_globalLeft.inner.count == rtg_example_com_app_globalRight.inner.count",
		"((rtg_example_com_app_globalLeft.inner.flags)[0] == (rtg_example_com_app_globalRight.inner.flags)[0] && (rtg_example_com_app_globalLeft.inner.flags)[1] == (rtg_example_com_app_globalRight.inner.flags)[1])",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered struct comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersNamedArrayStructFieldComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type values [2]int

type box struct {
	values values
}

func same(left box, right box) bool {
	return left == right
}

func appMain() int {
	left := box{values: values{1, 2}}
	right := box{values: values{1, 2}}
	different := box{values: values{1, 3}}
	localSame := left == right
	localDifferent := left != different
	if same(left, right) && localSame && localDifferent {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	sameBody := bodyByName["same"]
	if sameBody == "" {
		t.Fatalf("same declaration missing from lowered unit: %#v", u.Decls)
	}
	if strings.Contains(sameBody, "left == right") {
		t.Fatalf("named-array struct comparison leaked into same:\n%s", sameBody)
	}
	if !strings.Contains(sameBody, "((left.values)[0] == (right.values)[0] && (left.values)[1] == (right.values)[1])") {
		t.Fatalf("same did not lower named-array field comparison:\n%s", sameBody)
	}
	body := bodyByName["appMain"]
	if body == "" {
		t.Fatalf("appMain declaration missing from lowered unit: %#v", u.Decls)
	}
	for _, bad := range []string{"left == right", "left != different"} {
		if strings.Contains(body, bad) {
			t.Fatalf("named-array struct comparison leaked into appMain as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"localSame := ((left.values)[0] == (right.values)[0] && (left.values)[1] == (right.values)[1])",
		"localDifferent := ((left.values)[0] != (different.values)[0] || (left.values)[1] != (different.values)[1])",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered named-array struct comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersEmptyStructComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type empty struct{}

var globalLeft = empty{}
var globalRight = empty{}

func same(left empty, right empty) bool {
	return left == right
}

func appMain() int {
	left := empty{}
	right := empty{}
	localSame := left == right
	localDifferent := left != right
	globalSame := globalLeft == globalRight
	if same(left, right) && localSame && !localDifferent && globalSame {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want empty, globals, same, and appMain", u.Decls)
	}
	sameBody := u.Decls[3].Body
	if !strings.Contains(sameBody, "return true") {
		t.Fatalf("same did not lower empty struct parameter comparison:\n%s", sameBody)
	}
	body := u.Decls[4].Body
	for _, bad := range []string{"left == right", "left != right", "globalLeft == globalRight"} {
		if strings.Contains(body, bad) {
			t.Fatalf("empty struct comparison form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"localSame := true",
		"localDifferent := false",
		"globalSame := true",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered empty struct comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersStructComparisonOperandTemps(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct {
	value int
	name string
}

func leftBox() box {
	return box{value: 1, name: "a"}
}

func rightBox() box {
	return box{value: 1, name: "a"}
}

func appMain() int {
	literalSame := box{value: 1, name: "a"} == box{value: 1, name: "a"}
	callSame := leftBox() == rightBox()
	if (box{value: 2, name: "b"}) == (box{value: 2, name: "b"}) && leftBox() == rightBox() {
		if literalSame && callSame {
			return 0
		}
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want box, leftBox, rightBox, and appMain", u.Decls)
	}
	body := u.Decls[3].Body
	for _, bad := range []string{
		"} ==",
		"} !=",
		"rtg_example_com_app_leftBox() == rtg_example_com_app_rightBox()",
		"leftBox() == rightBox()",
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("struct comparison operand form leaked as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"var rtg_example_com_app_appMain_struct_cmp_tmp_0 rtg_example_com_app_box = rtg_example_com_app_box{value: 1, name: \"a\"}",
		"var rtg_example_com_app_appMain_struct_cmp_tmp_1 rtg_example_com_app_box = rtg_example_com_app_box{value: 1, name: \"a\"}",
		"literalSame := (rtg_example_com_app_appMain_struct_cmp_tmp_0.value == rtg_example_com_app_appMain_struct_cmp_tmp_1.value",
		"var rtg_example_com_app_appMain_struct_cmp_tmp_2 rtg_example_com_app_box = rtg_example_com_app_leftBox()",
		"var rtg_example_com_app_appMain_struct_cmp_tmp_3 rtg_example_com_app_box = rtg_example_com_app_rightBox()",
		"callSame := (rtg_example_com_app_appMain_struct_cmp_tmp_2.value == rtg_example_com_app_appMain_struct_cmp_tmp_3.value",
		"if (rtg_example_com_app_appMain_struct_cmp_tmp_4.value == rtg_example_com_app_appMain_struct_cmp_tmp_5.value",
		"if (rtg_example_com_app_appMain_struct_cmp_tmp_6.value == rtg_example_com_app_appMain_struct_cmp_tmp_7.value",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered struct operand fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageWithGraphLowersImportedStructComparisonOperands(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func appMain() int {
	globalSame := dep.Left == dep.Right
	callSame := dep.Make(1) == dep.Make(1)
	literalSame := dep.Box{Value: 1, Name: "a"} == dep.Box{Value: 1, Name: "a"}
	if globalSame && callSame && literalSame && (dep.Box{Value: 2, Name: "b"}) == (dep.Box{Value: 2, Name: "b"}) {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Box struct {
	Value int
	Name string
}

var Left = Box{Value: 1, Name: "a"}
var Right = Box{Value: 1, Name: "a"}

func Make(value int) Box {
	return Box{Value: value, Name: "a"}
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	body := bodyByName["appMain"]
	if body == "" {
		t.Fatalf("appMain declaration missing from lowered unit: %#v", u.Decls)
	}
	for _, bad := range []string{
		"dep.Left == dep.Right",
		"rtg_example_com_app_pkg_dep_Left == rtg_example_com_app_pkg_dep_Right",
		"rtg_example_com_app_pkg_dep_Make(1) == rtg_example_com_app_pkg_dep_Make(1)",
		"rtg_example_com_app_pkg_dep_Box{Value: 1, Name: \"a\"} ==",
		"rtg_example_com_app_pkg_dep_Box{Value: 2, Name: \"b\"}) ==",
	} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported struct comparison form leaked as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"globalSame := (rtg_example_com_app_pkg_dep_Left.Value == rtg_example_com_app_pkg_dep_Right.Value && rtg_example_com_app_pkg_dep_Left.Name == rtg_example_com_app_pkg_dep_Right.Name)",
		"var rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_0 rtg_example_com_app_pkg_dep_Box = rtg_example_com_app_pkg_dep_Make(1)",
		"var rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_1 rtg_example_com_app_pkg_dep_Box = rtg_example_com_app_pkg_dep_Make(1)",
		"callSame := (rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_0.Value == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_1.Value && rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_0.Name == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_1.Name)",
		"var rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_2 rtg_example_com_app_pkg_dep_Box = rtg_example_com_app_pkg_dep_Box{Value: 1, Name: \"a\"}",
		"var rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_3 rtg_example_com_app_pkg_dep_Box = rtg_example_com_app_pkg_dep_Box{Value: 1, Name: \"a\"}",
		"literalSame := (rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_2.Value == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_3.Value && rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_2.Name == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_3.Name)",
		"if (rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_4.Value == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_5.Value && rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_4.Name == rtg_example_com_app_cmd_app_appMain_struct_cmp_tmp_5.Name)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered imported struct comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersIndexedArrayStructFieldComparisons(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Values [2]int
}

func appMain() int {
	boxes := []Box{
		{Values: [2]int{1, 2}},
		{Values: [2]int{1, 2}},
		{Values: [2]int{3, 4}},
	}
	if boxes[0].Values == boxes[1].Values && boxes[0].Values != boxes[2].Values {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want Box and appMain", u.Decls)
	}
	body := u.Decls[1].Body
	for _, bad := range []string{"[2]int", "boxes[0].Values == boxes[1].Values", "boxes[0].Values != boxes[2].Values"} {
		if strings.Contains(body, bad) {
			t.Fatalf("indexed array field comparison form leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		"((boxes[0].Values)[0] == (boxes[1].Values)[0] && (boxes[0].Values)[1] == (boxes[1].Values)[1])",
		"((boxes[0].Values)[0] != (boxes[2].Values)[0] || (boxes[0].Values)[1] != (boxes[2].Values)[1])",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered indexed array field comparison fragment %q in:\n%s", want, body)
		}
	}
}

func TestPackageLowersIndexedArrayStructFieldCopies(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Box struct {
	Values [2]int
}

func mutate(values [2]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}

func appMain() int {
	first := Box{Values: [2]int{1, 2}}
	second := Box{Values: [2]int{3, 4}}
	boxes := []Box{first, second}
	copy := boxes[0].Values
	copy[0] = 8
	target := Box{}
	target.Values = boxes[1].Values
	replacement := [2]int{5, 6}
	boxes[0].Values = replacement
	replacement[0] = 99
	boxes[1].Values[0] = 6
	changed := mutate(boxes[1].Values)
	if boxes[0].Values[0] == 5 && boxes[1].Values[0] == 6 && copy[0] == 8 && target.Values[0] == 3 && replacement[0] == 99 && changed == 13 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 3 {
		t.Fatalf("decls = %#v, want Box, mutate, and appMain", u.Decls)
	}
	appMain := u.Decls[2].Body
	for _, bad := range []string{"[2]int", "copy := boxes[0].Values", "target.Values = boxes[1].Values", "mutate(boxes[1].Values)"} {
		if strings.Contains(appMain, bad) {
			t.Fatalf("indexed array field copy form leaked into lowered body as %q:\n%s", bad, appMain)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_appMain_tmp_0 := boxes[0].Values",
		"copy := append(make([]int, 0, 2), rtg_example_com_app_appMain_tmp_0...)",
		"rtg_example_com_app_appMain_tmp_1 := boxes[1].Values",
		"target.Values = append(make([]int, 0, 2), rtg_example_com_app_appMain_tmp_1...)",
		"replacement := []int{5, 6}",
		"rtg_example_com_app_appMain_tmp_2 := make([]int, 0, 2)",
		"boxes[0].Values = append(rtg_example_com_app_appMain_tmp_2, replacement...)",
		"replacement[0] = 99",
		"rtg_example_com_app_appMain_array_lvalue_tmp_0 := boxes[1].Values",
		"rtg_example_com_app_appMain_array_lvalue_tmp_0[0] = 6",
		"rtg_example_com_app_appMain_tmp_3 := boxes[1].Values",
		"rtg_example_com_app_appMain_tmp_4 := append(make([]int, 0, 2), rtg_example_com_app_appMain_tmp_3...)",
		"changed := rtg_example_com_app_mutate(rtg_example_com_app_appMain_tmp_4)",
	} {
		if !strings.Contains(appMain, want) {
			t.Fatalf("missing lowered indexed array field copy fragment %q in:\n%s", want, appMain)
		}
	}
}

func TestPackageLowersArrayResults(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func values() [3]int {
	return [3]int{1, 2}
}

func pair() ([2]int, int) {
	return [2]int{3}, 4
}

func zeroBytes() (out [2]byte) {
	return
}

func explicitBytes() (out [2]byte) {
	return [2]byte{'A'}
}

func appMain() int {
	vals := values()
	pairValues, n := pair()
	zeros := zeroBytes()
	bytes := explicitBytes()
	return len(vals) + cap(vals) + vals[1] + len(pairValues) + pairValues[0] + n + len(zeros) + len(bytes) + int(bytes[0]) - 85
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want values, pair, zeroBytes, explicitBytes, and appMain", u.Decls)
	}
	for declIndex := 0; declIndex < len(u.Decls); declIndex++ {
		body := u.Decls[declIndex].Body
		for _, bad := range []string{"[3]int", "[2]int"} {
			if strings.Contains(body, bad) {
				t.Fatalf("array result form leaked into lowered body: %q", body)
			}
		}
	}
	for _, want := range []string{
		"func rtg_example_com_app_values() []int",
		"return []int{1, 2, 0}",
		"func rtg_example_com_app_pair() ([]int, int)",
		"return []int{3, 0}, 4",
		"func rtg_example_com_app_zeroBytes() []byte",
		"out := []byte{0, 0}",
		"return out",
		"func rtg_example_com_app_explicitBytes() []byte",
		"return []byte{'A', 0}",
	} {
		found := false
		for declIndex := 0; declIndex < len(u.Decls); declIndex++ {
			if strings.Contains(u.Decls[declIndex].Body, want) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing lowered array result fragment %q in decls %#v", want, u.Decls)
		}
	}
}

func TestPackageLowersEmbeddedStructFieldsAndPromotedSelectors(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Inner struct { X int }
type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func appMain() int {
	outer := Outer{Inner: Inner{X: 2}}
	nested := Nested{Mid{Inner{X: 3}}}
	pointer := PointerOuter{Inner: &Inner{X: 4}}
	return outer.X + nested.X + pointer.X + Outer{Inner{5}}.X + Nested{Mid{Inner{6}}}.X + (PointerOuter{&Inner{7}}).X - 27
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	var outerBody string
	var nestedBody string
	var pointerBody string
	var appBody string
	for i := 0; i < len(u.Decls); i++ {
		switch u.Decls[i].Name {
		case "Outer":
			outerBody = u.Decls[i].Body
		case "Nested":
			nestedBody = u.Decls[i].Body
		case "PointerOuter":
			pointerBody = u.Decls[i].Body
		case "appMain":
			appBody = u.Decls[i].Body
		}
	}
	if !strings.Contains(outerBody, "Inner rtg_example_com_app_Inner") {
		t.Fatalf("embedded value field was not expanded: %q", outerBody)
	}
	if !strings.Contains(nestedBody, "Mid rtg_example_com_app_Mid") {
		t.Fatalf("nested embedded value field was not expanded: %q", nestedBody)
	}
	if !strings.Contains(pointerBody, "Inner *rtg_example_com_app_Inner") {
		t.Fatalf("embedded pointer field was not expanded: %q", pointerBody)
	}
	for _, want := range []string{
		"outer.Inner.X",
		"nested.Mid.Inner.X",
		"pointer.Inner.X",
		".Inner.X",
		".Mid.Inner.X",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing promoted selector rewrite %q in:\n%s", want, appBody)
		}
	}
	for _, bad := range []string{"outer.X", "nested.X", "pointer.X", "}.X"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("promoted selector leaked as %q in:\n%s", bad, appBody)
		}
	}
	if strings.Count(appBody, "_tmp_") < 3 {
		t.Fatalf("direct promoted literal selectors were not normalized through temps:\n%s", appBody)
	}
}

func TestPackageLowersPromotedEmbeddedMethodCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Inner struct { X int }
func (in Inner) Value() int { return in.X }
func (in *Inner) Add(v int) int { in.X = in.X + v; return in.X }

type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func appMain() int {
	outer := Outer{Inner: Inner{X: 2}}
	nested := Nested{Mid: Mid{Inner: Inner{X: 3}}}
	pointer := PointerOuter{Inner: &Inner{X: 4}}
	return outer.Value() + outer.Add(5) + nested.Value() + pointer.Value() + pointer.Add(6) - 24
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	var appBody string
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			appBody = u.Decls[i].Body
			break
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_Inner_Value(outer.Inner)",
		"rtg_example_com_app_Inner_Add(&outer.Inner, 5)",
		"rtg_example_com_app_Inner_Value(nested.Mid.Inner)",
		"rtg_example_com_app_Inner_Value(*pointer.Inner)",
		"rtg_example_com_app_Inner_Add(pointer.Inner, 6)",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing promoted method rewrite %q in:\n%s", want, appBody)
		}
	}
	for _, bad := range []string{"outer.Value()", "outer.Add(", "nested.Value()", "pointer.Value()", "pointer.Add("} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("promoted method call leaked as %q in:\n%s", bad, appBody)
		}
	}
}

func TestPackageLowersPromotedCompositeLiteralMethodCalls(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Inner struct { X int }
func (in Inner) Value() int { return in.X }
func (in *Inner) Add(v int) int { in.X = in.X + v; return in.X }

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func appMain() int {
	return Outer{Inner: Inner{X: 2}}.Value() + PointerOuter{Inner: &Inner{X: 4}}.Add(6) + (&Outer{Inner: Inner{X: 7}}).Add(8) - 27
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	var appBody string
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			appBody = u.Decls[i].Body
			break
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_Inner_Value(",
		"rtg_example_com_app_Inner_Add(",
		".Inner",
	} {
		if !strings.Contains(appBody, want) {
			t.Fatalf("missing promoted composite method rewrite fragment %q in:\n%s", want, appBody)
		}
	}
	for _, bad := range []string{"}.Value()", "}.Add(", ").Add("} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("promoted composite method call leaked as %q in:\n%s", bad, appBody)
		}
	}
}

func TestPackageNormalizesDirectSliceLiteralIndexes(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	if []int{41, 42}[1] != 42 {
		return 1
	}
	return ([]byte{65, 66})[0] - byte('A')
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"[]int{41, 42}[1]", "[]byte{65, 66})[0]"} {
		if strings.Contains(body, bad) {
			t.Fatalf("direct slice literal index leaked into lowered body: %q", body)
		}
	}
	if strings.Count(body, "_tmp_") < 2 {
		t.Fatalf("missing slice literal index temporaries in:\n%s", body)
	}
}

func TestPackageLowersNamedSliceLiteralIndexedReceiverMethods(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct { value int }
type boxes []box

func (b box) Value() int { return b.value }

func appMain() int {
	return (boxes{{value: 41}})[0].Value() - 41
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	if strings.Contains(body, "(boxes{{value: 41}})[0].Value()") {
		t.Fatalf("named slice indexed method call leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_box_Value(") {
		t.Fatalf("method call was not lowered:\n%s", body)
	}
	if !strings.Contains(body, "_tmp_") {
		t.Fatalf("named slice literal index was not normalized through a temp:\n%s", body)
	}
}

func TestPackageNormalizesDirectStructLiteralSelectors(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type box struct { value int }

func appMain() int {
	if (box{value: 42}).value != 42 {
		return 1
	}
	if (&box{value: 43}).value != 43 {
		return 2
	}
	return box{value: 40}.value - 40
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want box and appMain", u.Decls)
	}
	body := u.Decls[1].Body
	for _, bad := range []string{"(rtg_example_com_app_box{value: 42}).value", "(&rtg_example_com_app_box{value: 43}).value", "rtg_example_com_app_box{value: 40}.value"} {
		if strings.Contains(body, bad) {
			t.Fatalf("direct struct literal selector leaked into lowered body: %q", body)
		}
	}
	if strings.Count(body, "_tmp_") < 3 {
		t.Fatalf("missing struct literal selector temporaries in:\n%s", body)
	}
}

func TestPackageDoesNotRewriteStructFieldNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "fields.go",
				Source: []byte(`package pkg

type Artifact struct {
	Source []byte
}

func Source() []byte {
	return nil
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "Source []byte") {
		t.Fatalf("struct field name was rewritten: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Source []byte") {
		t.Fatalf("struct field contains unit name: %q", body)
	}
}

func TestPackageLowersStructFieldTypeNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "fields.go",
				Source: []byte(`package pkg

type Symbol struct {
	Name string
}

type Unit struct {
	Exports []Symbol
	Primary Symbol
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[1].Body
	if strings.Contains(body, "[]Symbol") || strings.Contains(body, "Primary Symbol") {
		t.Fatalf("struct field type retained source name: %q", body)
	}
	if !strings.Contains(body, "Exports []rtg_example_com_app_pkg_Symbol") {
		t.Fatalf("slice field type was not lowered: %q", body)
	}
	if !strings.Contains(body, "Primary rtg_example_com_app_pkg_Symbol") {
		t.Fatalf("plain field type was not lowered: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_Exports") || strings.Contains(body, "rtg_example_com_app_pkg_Primary") {
		t.Fatalf("struct field name was rewritten: %q", body)
	}
}

func TestPackageLowersImportedMethodCalls(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	var b dep.Buffer
	p := &b
	p.WriteString("PASS")
	return p.Len()
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct { buf []byte }

func (b *Buffer) WriteString(s string) int {
	return len(s)
}

func (b *Buffer) Len() int {
	return len(b.buf)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "var b rtg_example_com_app_dep_Buffer") {
		t.Fatalf("imported receiver type was not rewritten: %q", body)
	}
	if !strings.Contains(body, "p := &b") {
		t.Fatalf("address-of receiver local was not preserved: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_dep_Buffer_WriteString(p, \"PASS\")") {
		t.Fatalf("imported pointer method call was not lowered: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Buffer_Len(p)") {
		t.Fatalf("imported method call was not lowered: %q", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Buffer", "Buffer_Len", "Buffer_WriteString"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageWithGraphLowersImportedFunctionResultMethodCalls(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	buffer := dep.NewBufferString("PASS")
	alias := buffer
	buffer.WriteByte('\n')
	if alias.String() == "PASS\n" {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct { buf []byte }

func NewBufferString(s string) *Buffer {
	b := new(Buffer)
	b.buf = []byte(s)
	return b
}

func (b *Buffer) WriteByte(c byte) int {
	b.buf = append(b.buf, c)
	return 0
}

func (b *Buffer) String() string {
	return string(b.buf)
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"buffer.WriteByte", "alias.String", "dep.NewBufferString"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported function result method call leaked as %q in:\n%s", bad, body)
		}
	}
	for _, want := range []string{
		`buffer := rtg_example_com_app_dep_NewBufferString("PASS")`,
		`alias := buffer`,
		`rtg_example_com_app_dep_Buffer_WriteByte(buffer, '\n')`,
		`rtg_example_com_app_dep_Buffer_String(alias) == "PASS\n"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing lowered form %q in:\n%s", want, body)
		}
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Buffer_String", "Buffer_WriteByte", "NewBufferString"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageLowersImportedDirectMethodExpressionCalls(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	return dep.Buffer.Len(dep.Buffer{})
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int { return 1 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "dep.Buffer.Len") {
		t.Fatalf("imported method expression call leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Buffer_Len(rtg_example_com_app_dep_Buffer{})") {
		t.Fatalf("imported method expression call was not lowered: %q", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Buffer", "Buffer_Len"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageWithGraphLowersImportedStaticMethodExpressionAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	f := dep.Buffer.Len
	_ = dep.Buffer.Len
	_ = f
	return f(dep.Buffer{})
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int { return 1 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	for _, bad := range []string{"dep.Buffer.Len", "f := ", "_ = f", "f("} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported method expression alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Buffer_Len(rtg_example_com_app_dep_Buffer{})") {
		t.Fatalf("imported method expression alias call was not lowered: %q", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Buffer", "Buffer_Len"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageWithGraphLowersImportedStaticMethodValueAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func applyLen(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func appMain() int {
	b := dep.Buffer{Value: 42}
	f := b.Len
	_ = b.Len
	_ = f
	other := dep.Buffer{Value: 2}
	return f() + applyLen(f) + applyLen(b.Len) + applyLen(dep.Buffer{Value: 3}.Len) + applyAdd(other.Add, 4) - 135
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "buffer.go",
				Source: []byte(`package dep

type Buffer struct{ Value int }

func (b Buffer) Len() int { return b.Value }
func (b *Buffer) Add(v int) int { b.Value = b.Value + v; return b.Value }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	body := bodyByName["appMain"]
	if body == "" {
		t.Fatalf("missing appMain decl from %#v", u.Decls)
	}
	for _, bad := range []string{"f := b.Len", "_ = b.Len", "_ = f", "f()", "applyLen(f", "applyLen(b.Len", "applyLen(dep.Buffer", "applyAdd(other.Add"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported method value alias leaked as %q into lowered body: %q", bad, body)
		}
	}
	for _, want := range []string{
		"rtg_example_com_app_main_appMain_method_value_receiver_tmp_0 := b",
		"return rtg_example_com_app_dep_Buffer_Len(rtg_example_com_app_main_appMain_method_value_receiver_tmp_0) + rtg_example_com_app_main_applyLen_callback_0(rtg_example_com_app_main_appMain_method_value_receiver_tmp_0) + rtg_example_com_app_main_applyLen_callback_0(b) + rtg_example_com_app_main_applyLen_callback_0(rtg_example_com_app_dep_Buffer{Value: 3}) + rtg_example_com_app_main_applyAdd_callback_1(&other, 4) - 135",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing imported method value alias fragment %q in:\n%s", want, body)
		}
	}
	valueCallback := bodyByName["rtg_example_com_app_main_applyLen_callback_0"]
	if !strings.Contains(valueCallback, "func rtg_example_com_app_main_applyLen_callback_0(rtg_callback_capture_0_0 rtg_example_com_app_dep_Buffer) int") || !strings.Contains(valueCallback, "return rtg_example_com_app_dep_Buffer_Len(rtg_callback_capture_0_0)") {
		t.Fatalf("imported direct value method callback specialization was not rewritten correctly:\n%s", valueCallback)
	}
	addCallback := bodyByName["rtg_example_com_app_main_applyAdd_callback_1"]
	if !strings.Contains(addCallback, "func rtg_example_com_app_main_applyAdd_callback_1(rtg_callback_capture_0_0 *rtg_example_com_app_dep_Buffer, v int) int") || !strings.Contains(addCallback, "return rtg_example_com_app_dep_Buffer_Add(rtg_callback_capture_0_0, v)") {
		t.Fatalf("imported direct pointer method callback specialization was not rewritten correctly:\n%s", addCallback)
	}
	bufferRef := false
	lenRef := false
	addRef := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/dep" && ref.Name == "Buffer" && ref.UnitName == "rtg_example_com_app_dep_Buffer" {
			bufferRef = true
		}
		if ref.ImportPath == "example.com/app/dep" && ref.Name == "Buffer_Len" && ref.UnitName == "rtg_example_com_app_dep_Buffer_Len" {
			lenRef = true
		}
		if ref.ImportPath == "example.com/app/dep" && ref.Name == "Buffer_Add" && ref.UnitName == "rtg_example_com_app_dep_Buffer_Add" {
			addRef = true
		}
	}
	if !bufferRef || !lenRef || !addRef {
		t.Fatalf("references = %#v, want Buffer, Buffer_Len, and Buffer_Add", u.References)
	}
}

func TestPackageLowersImportedMethodCallsOnCompositeLiterals(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	if (dep.Point{X: 3, Y: 4}).Sum() != 7 {
		return 1
	}
	return (&dep.Counter{N: 42}).Value()
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Point struct { X int; Y int }

func (p Point) Sum() int {
	return p.X + p.Y
}

type Counter struct { N int }

func (c *Counter) Value() int {
	return c.N
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, ".Sum()") || strings.Contains(body, ".Value()") {
		t.Fatalf("imported composite literal method call was not lowered: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_dep_Point_Sum(rtg_example_com_app_dep_Point{X: 3, Y: 4})") {
		t.Fatalf("imported value receiver composite method call was not lowered: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_main_appMain_tmp_0 := rtg_example_com_app_dep_Counter{N: 42}") || !strings.Contains(body, "return rtg_example_com_app_dep_Counter_Value(&rtg_example_com_app_main_appMain_tmp_0)") {
		t.Fatalf("imported pointer receiver composite method call was not lowered: %q", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Counter", "Counter_Value", "Point", "Point_Sum"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageLowersImportedNamedSliceConversionRangeOperands(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	xs := dep.Values([]int{1, 2})
	total := len(xs)
	for _, v := range dep.Values([]int{1, 2}) {
		total = total + v
	}
	return total + dep.Sum(dep.Values([]int{3, 4})) - 12
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Values []int

func Sum(values Values) int {
	total := 0
	for i := 0; i < len(values); i++ {
		total = total + values[i]
	}
	return total
}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want 1", u.Decls)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "rtg_example_com_app_dep_Values([]int") {
		t.Fatalf("imported named slice conversion leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "xs := rtg_example_com_app_dep_Values{1, 2}") || !strings.Contains(body, " := rtg_example_com_app_dep_Values{1, 2}") || !strings.Contains(body, " := rtg_example_com_app_dep_Values{3, 4}") || !strings.Contains(body, "rtg_example_com_app_dep_Sum(rtg_example_com_app_main_appMain_tmp_") {
		t.Fatalf("imported named slice conversions did not lower to named literals: %q", body)
	}
}

func TestSymbolNameIsStableIdentifier(t *testing.T) {
	got := SymbolName("example.com/team/app-pkg", "Value")
	want := "rtg_example_com_team_app_pkg_Value"
	if got != want {
		t.Fatalf("SymbolName = %q, want %q", got, want)
	}
}

func TestPackageSynthesizesAppMainForOrdinaryMain(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func main() {
	print("PASS\n")
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want source main plus synthetic appMain", u.Decls)
	}
	if u.Decls[0].Name != "main" || u.Decls[0].UnitName != "rtg_example_com_app_cmd_app_main" {
		t.Fatalf("source main decl = %#v", u.Decls[0])
	}
	if u.Decls[1].Name != "appMain" || u.Decls[1].Path != "rtg-entrypoint" {
		t.Fatalf("synthetic entrypoint decl = %#v", u.Decls[1])
	}
	want := "func rtg_example_com_app_cmd_app_appMain(args []string, env []string) int {\n\trtg_example_com_app_cmd_app_main()\n\treturn 0\n}\n"
	if u.Decls[1].Body != want {
		t.Fatalf("synthetic entrypoint body = %q, want %q", u.Decls[1].Body, want)
	}
}

func TestPackageSynthesizesAppMainAbortCheckForOrdinaryMainPanic(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func main() {
	panic("boom")
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for i := 0; i < len(u.Decls); i++ {
		bodyByName[u.Decls[i].Name] = u.Decls[i].Body
	}
	entry := bodyByName["appMain"]
	for _, want := range []string{
		"rtg_example_com_app_cmd_app_main()",
		"if " + SymbolName("example.com/app/cmd/app", "__rtg_panic_active") + " {",
		"return " + SymbolName("example.com/app/cmd/app", "__rtg_panic_abort") + "()",
		"return 0",
	} {
		if !strings.Contains(entry, want) {
			t.Fatalf("missing synthetic appMain panic fragment %q in:\n%s", want, entry)
		}
	}
}

func TestPackageWithGraphRewritesImportedSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

func appMain() int { return answer.Value() }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
func hidden() int { return 9 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 {
		t.Fatalf("references = %#v, want one", u.References)
	}
	ref := u.References[0]
	if ref.ImportPath != "example.com/app/pkg/answer" || ref.Name != "Value" || ref.UnitName != "rtg_example_com_app_pkg_answer_Value" {
		t.Fatalf("reference = %#v", ref)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("imported selector was not rewritten: %q", u.Decls[0].Body)
	}
}

func TestPackageWithGraphLowersImportedPromotedSelectors(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Count: 2}}
	return o.Count + dep.Outer{Inner: dep.Inner{Count: 3}}.Count + dep.Default.Count - 9
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Inner struct {
	Count int
}

type Outer struct {
	Inner
}

var Default = Outer{Inner: Inner{Count: 4}}
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	for _, want := range []string{
		"o.Inner.Count",
		".Inner.Count",
		"rtg_example_com_app_dep_Default.Inner.Count",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing imported promoted selector rewrite %q in:\n%s", want, body)
		}
	}
	for _, bad := range []string{"o.Count", "}.Count", "rtg_example_com_app_dep_Default.Count"} {
		if strings.Contains(body, bad) {
			t.Fatalf("imported promoted selector leaked as %q in:\n%s", bad, body)
		}
	}
	if strings.Count(body, "_tmp_") == 0 {
		t.Fatalf("direct imported promoted literal selector was not normalized through a temp:\n%s", body)
	}
}

func TestPackageWithGraphLowersImportedPromotedMethodCalls(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/main",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Count: 2}}
	return o.Value() - 2
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

type Inner struct { Count int }

func (in Inner) Value() int {
	return in.Count
}

type Outer struct { Inner }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.Decls) != 1 {
		t.Fatalf("decls = %#v, want appMain", u.Decls)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "rtg_example_com_app_dep_Inner_Value(o.Inner)") {
		t.Fatalf("imported promoted method call was not lowered:\n%s", body)
	}
	if strings.Contains(body, "o.Value()") {
		t.Fatalf("imported promoted method call leaked into lowered body:\n%s", body)
	}
	var gotRefs []string
	for _, ref := range u.References {
		gotRefs = append(gotRefs, ref.Name)
	}
	wantRefs := []string{"Inner", "Inner_Value", "Outer"}
	if strings.Join(gotRefs, ",") != strings.Join(wantRefs, ",") {
		t.Fatalf("references = %#v, want %v", u.References, wantRefs)
	}
}

func TestPackageLowersStaticFunctionAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func add(a int) int { return a + 1 }
func join(a int, b int) int { return a + b }

func appMain() int {
	f := add
	var g = join
	_ = add
	_ = f
	_ = func() int { return 99 }
	return f(1) + g(2, 3) - 7
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			body = decl.Body
			break
		}
	}
	if body == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := add", "var g = join", "_ = add", "_ = f", "func() int", "f(", "g("} {
		if strings.Contains(body, bad) {
			t.Fatalf("function alias leaked into lowered body as %q: %q", bad, body)
		}
	}
	if strings.Contains(body, "99") {
		t.Fatalf("function alias leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_add(1) + rtg_example_com_app_join(2, 3) - 7") {
		t.Fatalf("function alias calls were not lowered: %q", body)
	}
}

func TestPackageLowersStaticCallbacks(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func apply(f func(int, int) int, value int) int { return f(value, 1) }
func add(a int, b int) int { return a + b }
type box struct { value int }
func (b box) Add(a int) int { return b.value + a }
func applyMethod(f func(box, int) int, b box) int { return f(b, 1) }

func appMain() int {
	f := add
	g := box.Add
	offset := 6
	return apply(f, 2) + applyMethod(box.Add, box{value: 3}) + applyMethod(g, box{value: 4}) + apply(func(a int, b int) int { return a * b }, 5) + apply(func(a int, b int) int { offset = offset + a; return offset + b }, 7) + offset - 44
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	if bodyByName["apply"] != "" || bodyByName["applyMethod"] != "" {
		t.Fatalf("unspecialized callback wrapper leaked into lowered decls: %#v", u.Decls)
	}
	appBody := bodyByName["appMain"]
	if !strings.Contains(appBody, "return rtg_example_com_app_apply_callback_0(2) + rtg_example_com_app_applyMethod_callback_1(") || !strings.Contains(appBody, ") + rtg_example_com_app_applyMethod_callback_1(") || !strings.Contains(appBody, ") + rtg_example_com_app_apply_callback_2(5) + rtg_example_com_app_apply_callback_3(&offset, 7) + offset - 44") {
		t.Fatalf("static callback call was not lowered:\n%s", appBody)
	}
	callbackBody := bodyByName["rtg_example_com_app_apply_callback_0"]
	if callbackBody == "" {
		t.Fatalf("static callback specialization missing from %#v", u.Decls)
	}
	for _, bad := range []string{"func(int", " f(", "f := add", "apply(f"} {
		if strings.Contains(callbackBody, bad) {
			t.Fatalf("callback specialization still contains %q:\n%s", bad, callbackBody)
		}
	}
	if !strings.Contains(callbackBody, "func rtg_example_com_app_apply_callback_0(value int) int") || !strings.Contains(callbackBody, "return rtg_example_com_app_add(value, 1)") {
		t.Fatalf("callback specialization was not rewritten correctly:\n%s", callbackBody)
	}
	methodCallbackBody := bodyByName["rtg_example_com_app_applyMethod_callback_1"]
	if methodCallbackBody == "" {
		t.Fatalf("static method-expression callback specialization missing from %#v", u.Decls)
	}
	if !strings.Contains(methodCallbackBody, "func rtg_example_com_app_applyMethod_callback_1(b rtg_example_com_app_box) int") || !strings.Contains(methodCallbackBody, "return rtg_example_com_app_box_Add(b, 1)") {
		t.Fatalf("method-expression callback specialization was not rewritten correctly:\n%s", methodCallbackBody)
	}
	literalCallbackBody := bodyByName["rtg_example_com_app_apply_callback_2"]
	if literalCallbackBody == "" {
		t.Fatalf("static function-literal callback specialization missing from %#v", u.Decls)
	}
	if !strings.Contains(literalCallbackBody, "func rtg_example_com_app_apply_callback_2(value int) int") || !strings.Contains(literalCallbackBody, "return rtg_example_com_app_appMain_func_literal_2(value, 1)") {
		t.Fatalf("function-literal callback specialization was not rewritten correctly:\n%s", literalCallbackBody)
	}
	literalBody := bodyByName["rtg_example_com_app_appMain_func_literal_2"]
	if literalBody == "" {
		t.Fatalf("function-literal helper missing from %#v", u.Decls)
	}
	if !strings.Contains(literalBody, "func rtg_example_com_app_appMain_func_literal_2(a int, b int) int") || !strings.Contains(literalBody, "return a * b") {
		t.Fatalf("function-literal helper was not lowered correctly:\n%s", literalBody)
	}
	capturingCallbackBody := bodyByName["rtg_example_com_app_apply_callback_3"]
	if capturingCallbackBody == "" {
		t.Fatalf("capturing function-literal callback specialization missing from %#v", u.Decls)
	}
	if !strings.Contains(capturingCallbackBody, "func rtg_example_com_app_apply_callback_3(rtg_callback_capture_0_0 *int, value int) int") || !strings.Contains(capturingCallbackBody, "return rtg_example_com_app_appMain_func_literal_3(rtg_callback_capture_0_0, value, 1)") {
		t.Fatalf("capturing function-literal callback specialization was not rewritten correctly:\n%s", capturingCallbackBody)
	}
	capturingLiteralBody := bodyByName["rtg_example_com_app_appMain_func_literal_3"]
	if capturingLiteralBody == "" {
		t.Fatalf("capturing function-literal helper missing from %#v", u.Decls)
	}
	if !strings.Contains(capturingLiteralBody, "func rtg_example_com_app_appMain_func_literal_3(rtg_capture_offset *int, a int, b int) int") || !strings.Contains(capturingLiteralBody, "*rtg_capture_offset = *rtg_capture_offset + a") || !strings.Contains(capturingLiteralBody, "return *rtg_capture_offset + b") {
		t.Fatalf("capturing function-literal helper was not lowered correctly:\n%s", capturingLiteralBody)
	}
}

func TestPackageLowersNamedFunctionTypeStaticCallbacks(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Unary func(int) int
type Binary = func(int, int) int
type AlsoUnary Unary
type AlsoBinary Binary

func apply(f Unary, value int) int { return f(value) }
func applyAlias(f AlsoUnary, value int) int { return f(value) }
func applyBinary(f Binary, value int) int { return f(value, 1) }
func applyBinaryAlias(f AlsoBinary, value int) int { return f(value, 1) }
func inc(x int) int { return x + 1 }
func add(x int, y int) int { return x + y }

func appMain() int {
	f := inc
	return apply(inc, 2) + apply(f, 3) + applyBinary(add, 4) + applyAlias(inc, 5) + applyBinaryAlias(add, 6) - 25
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	if bodyByName["apply"] != "" || bodyByName["applyAlias"] != "" || bodyByName["applyBinary"] != "" || bodyByName["applyBinaryAlias"] != "" {
		t.Fatalf("unspecialized named callback wrappers leaked into lowered decls: %#v", u.Decls)
	}
	for _, decl := range u.Decls {
		for _, bad := range []string{"type Unary", "type Binary", "type AlsoUnary", "type AlsoBinary", "func(int"} {
			if strings.Contains(decl.Body, bad) {
				t.Fatalf("named function type leaked into lowered decl %s as %q:\n%s", decl.Name, bad, decl.Body)
			}
		}
	}
	appBody := bodyByName["appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := inc", "apply(inc", "apply(f", "applyBinary(add", "applyAlias(inc", "applyBinaryAlias(add"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("named callback call leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	if !strings.Contains(appBody, "return rtg_example_com_app_apply_callback_0(2) + rtg_example_com_app_apply_callback_0(3) + rtg_example_com_app_applyBinary_callback_1(4) + rtg_example_com_app_applyAlias_callback_2(5) + rtg_example_com_app_applyBinaryAlias_callback_3(6) - 25") {
		t.Fatalf("named callback calls were not lowered:\n%s", appBody)
	}
	unaryCallback := bodyByName["rtg_example_com_app_apply_callback_0"]
	if !strings.Contains(unaryCallback, "func rtg_example_com_app_apply_callback_0(value int) int") || !strings.Contains(unaryCallback, "return rtg_example_com_app_inc(value)") {
		t.Fatalf("named unary callback specialization was not rewritten correctly:\n%s", unaryCallback)
	}
	binaryCallback := bodyByName["rtg_example_com_app_applyBinary_callback_1"]
	if !strings.Contains(binaryCallback, "func rtg_example_com_app_applyBinary_callback_1(value int) int") || !strings.Contains(binaryCallback, "return rtg_example_com_app_add(value, 1)") {
		t.Fatalf("named binary callback specialization was not rewritten correctly:\n%s", binaryCallback)
	}
	unaryAliasCallback := bodyByName["rtg_example_com_app_applyAlias_callback_2"]
	if !strings.Contains(unaryAliasCallback, "func rtg_example_com_app_applyAlias_callback_2(value int) int") || !strings.Contains(unaryAliasCallback, "return rtg_example_com_app_inc(value)") {
		t.Fatalf("named unary alias callback specialization was not rewritten correctly:\n%s", unaryAliasCallback)
	}
	binaryAliasCallback := bodyByName["rtg_example_com_app_applyBinaryAlias_callback_3"]
	if !strings.Contains(binaryAliasCallback, "func rtg_example_com_app_applyBinaryAlias_callback_3(value int) int") || !strings.Contains(binaryAliasCallback, "return rtg_example_com_app_add(value, 1)") {
		t.Fatalf("named binary alias callback specialization was not rewritten correctly:\n%s", binaryAliasCallback)
	}
}

func TestPackageLowersFunctionLiteralAliasStaticCallbacks(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func apply(f func(int, int) int, value int) int { return f(value, 1) }

func appMain() int {
	offset := 3
	mul := func(a int, b int) int { return a * b }
	bump := func(a int, b int) int { offset = offset + a; return offset + b }
	return apply(mul, 5) + apply(bump, 7) + offset - 26
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	appBody := bodyByName["appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"mul :=", "bump :=", "apply(mul", "apply(bump"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("function-literal callback alias leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	if !strings.Contains(appBody, "return rtg_example_com_app_apply_callback_0(5) + rtg_example_com_app_apply_callback_1(&offset, 7) + offset - 26") {
		t.Fatalf("function-literal alias callbacks were not lowered:\n%s", appBody)
	}
	mulBody := bodyByName["rtg_example_com_app_appMain_func_literal_0"]
	if !strings.Contains(mulBody, "func rtg_example_com_app_appMain_func_literal_0(a int, b int) int") || !strings.Contains(mulBody, "return a * b") {
		t.Fatalf("noncapturing function-literal alias helper was not lowered correctly:\n%s", mulBody)
	}
	mulCallbackBody := bodyByName["rtg_example_com_app_apply_callback_0"]
	if !strings.Contains(mulCallbackBody, "func rtg_example_com_app_apply_callback_0(value int) int") || !strings.Contains(mulCallbackBody, "return rtg_example_com_app_appMain_func_literal_0(value, 1)") {
		t.Fatalf("noncapturing function-literal alias callback was not specialized correctly:\n%s", mulCallbackBody)
	}
	bumpBody := bodyByName["rtg_example_com_app_appMain_func_literal_1"]
	if !strings.Contains(bumpBody, "func rtg_example_com_app_appMain_func_literal_1(rtg_capture_offset *int, a int, b int) int") || !strings.Contains(bumpBody, "*rtg_capture_offset = *rtg_capture_offset + a") || !strings.Contains(bumpBody, "return *rtg_capture_offset + b") {
		t.Fatalf("capturing function-literal alias helper was not lowered correctly:\n%s", bumpBody)
	}
	bumpCallbackBody := bodyByName["rtg_example_com_app_apply_callback_1"]
	if !strings.Contains(bumpCallbackBody, "func rtg_example_com_app_apply_callback_1(rtg_callback_capture_0_0 *int, value int) int") || !strings.Contains(bumpCallbackBody, "return rtg_example_com_app_appMain_func_literal_1(rtg_callback_capture_0_0, value, 1)") {
		t.Fatalf("capturing function-literal alias callback was not specialized correctly:\n%s", bumpCallbackBody)
	}
}

func TestPackageLowersMethodValueAliasStaticCallbacks(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app/pkg",
		Name:       "pkg",
		Files: []load.File{
			{
				Path: "method_value_callback.go",
				Source: []byte(`package pkg

type Box struct { value int }

func (b Box) Value() int {
	return b.value
}

func (b *Box) Add(v int) int {
	b.value = b.value + v
	return b.value
}

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func Use() int {
	b := Box{value: 1}
	f := b.Value
	g := b.Add
	return applyValue(f) + applyAdd(g, 40) - 42
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	if bodyByName["applyValue"] != "" || bodyByName["applyAdd"] != "" {
		t.Fatalf("unspecialized method-value callback wrappers leaked into lowered decls: %#v", u.Decls)
	}
	body := bodyByName["Use"]
	if body == "" {
		t.Fatalf("Use decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := b.Value", "g := b.Add", "applyValue(f", "applyAdd(g"} {
		if strings.Contains(body, bad) {
			t.Fatalf("method-value callback alias leaked into lowered body as %q:\n%s", bad, body)
		}
	}
	if !strings.Contains(body, "rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0 := b") || !strings.Contains(body, "rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1 := &b") || !strings.Contains(body, "return rtg_example_com_app_pkg_applyValue_callback_0(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_0) + rtg_example_com_app_pkg_applyAdd_callback_1(rtg_example_com_app_pkg_Use_method_value_receiver_tmp_1, 40) - 42") {
		t.Fatalf("method-value alias callbacks were not lowered:\n%s", body)
	}
	valueCallback := bodyByName["rtg_example_com_app_pkg_applyValue_callback_0"]
	if !strings.Contains(valueCallback, "func rtg_example_com_app_pkg_applyValue_callback_0(rtg_callback_capture_0_0 rtg_example_com_app_pkg_Box) int") || !strings.Contains(valueCallback, "return rtg_example_com_app_pkg_Box_Value(rtg_callback_capture_0_0)") {
		t.Fatalf("value method callback specialization was not rewritten correctly:\n%s", valueCallback)
	}
	addCallback := bodyByName["rtg_example_com_app_pkg_applyAdd_callback_1"]
	if !strings.Contains(addCallback, "func rtg_example_com_app_pkg_applyAdd_callback_1(rtg_callback_capture_0_0 *rtg_example_com_app_pkg_Box, v int) int") || !strings.Contains(addCallback, "return rtg_example_com_app_pkg_Box_Add(rtg_callback_capture_0_0, v)") {
		t.Fatalf("pointer method callback specialization was not rewritten correctly:\n%s", addCallback)
	}
}

func TestPackageLowersStaticFunctionLiteralAliases(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	f := func(a int) int { return a + 1 }
	var g = func(a int, b int) int {
		total := a + b
		return total
	}
	offset := 3
	label := "PASS"
	h := func(a int) int {
		offset = offset + a
		offset++
		offset += 2
		return offset + len(label)
	}
	shadow := func() int {
		offset := offset + 1
		return offset
	}
	return f(1) + g(2, 3) + h(2) + offset + shadow() - 36
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	appBody := bodyByName["appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := func", "var g = func", "h := func", "shadow := func", "f(", "g(", "h(", "shadow("} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("function literal alias leaked into lowered body as %q:\n%s", bad, appBody)
		}
	}
	for _, want := range []string{
		"return rtg_example_com_app_appMain_func_literal_0(1) + rtg_example_com_app_appMain_func_literal_1(2, 3) + rtg_example_com_app_appMain_func_literal_2(&offset, label, 2) + offset + rtg_example_com_app_appMain_func_literal_3(offset) - 36",
		"func rtg_example_com_app_appMain_func_literal_0(a int) int { return a + 1 }",
		"func rtg_example_com_app_appMain_func_literal_1(a int, b int) int",
		"total := a + b",
		"func rtg_example_com_app_appMain_func_literal_2(rtg_capture_offset *int, rtg_capture_label string, a int) int",
		"*rtg_capture_offset = *rtg_capture_offset + a",
		"*rtg_capture_offset++",
		"*rtg_capture_offset += 2",
		"return *rtg_capture_offset + len(rtg_capture_label)",
		"func rtg_example_com_app_appMain_func_literal_3(rtg_capture_offset int) int",
		"offset := rtg_capture_offset + 1",
		"return offset",
	} {
		found := strings.Contains(appBody, want)
		if !found {
			for _, decl := range u.Decls {
				if strings.Contains(decl.Body, want) {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatalf("missing function literal lowering fragment %q in decls: %#v", want, u.Decls)
		}
	}
}

func TestPackageLowersImmediatelyInvokedFunctionLiterals(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	value := func(a int) int { return a + 1 }(1)
	offset := 3
	label := "PASS"
	if func(text string) bool { return len(text) == 4 }("PASS") {
		return func(a int, b int) int {
			offset = offset + b
			total := a + b
			return total + offset + len(label)
		}(value, 5) - 7
	}
	shadow := func() int {
		offset := offset + 1
		return offset
	}()
	return shadow - 4
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 5 {
		t.Fatalf("decls = %#v, want four synthetic literals plus appMain", u.Decls)
	}
	appBody := ""
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			appBody = decl.Body
		}
	}
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	if strings.Contains(appBody, "func(") {
		t.Fatalf("function literal leaked into lowered appMain:\n%s", appBody)
	}
	for _, want := range []string{
		"value := rtg_example_com_app_appMain_func_literal_0(1)",
		"offset := 3",
		`label := "PASS"`,
		"if rtg_example_com_app_appMain_func_literal_1(\"PASS\")",
		"return rtg_example_com_app_appMain_func_literal_2(&offset, label, value, 5) - 7",
		"func rtg_example_com_app_appMain_func_literal_0(a int) int { return a + 1 }",
		"func rtg_example_com_app_appMain_func_literal_1(text string) bool { return len(text) == 4 }",
		"func rtg_example_com_app_appMain_func_literal_2(rtg_capture_offset *int, rtg_capture_label string, a int, b int) int",
		"*rtg_capture_offset = *rtg_capture_offset + b",
		"total := a + b",
		"return total + *rtg_capture_offset + len(rtg_capture_label)",
		"shadow := rtg_example_com_app_appMain_func_literal_3(offset)",
		"return shadow - 4",
		"func rtg_example_com_app_appMain_func_literal_3(rtg_capture_offset int) int",
		"offset := rtg_capture_offset + 1",
	} {
		found := strings.Contains(appBody, want)
		if !found {
			for _, decl := range u.Decls {
				if strings.Contains(decl.Body, want) {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatalf("missing IIFE lowering fragment %q in decls: %#v", want, u.Decls)
		}
	}
}

func TestPackageLowersFunctionLiteralArrayParameters(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func appMain() int {
	type Values [2]int
	values := Values{2, 3}
	_ = func(x Values) int {
		x[0] = 20
		return x[0] + x[1]
	}(values)
	direct := [2]int{1, 1}
	f := func(x [2]int) int {
		x[0] = 30
		return x[0] + x[1]
	}
	return f(direct) + values[0] + direct[0] - 34
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	var appBody string
	var generatedBodies []string
	for _, decl := range u.Decls {
		if decl.Name == "appMain" {
			appBody = decl.Body
			continue
		}
		if strings.Contains(decl.Name, "_func_literal_") {
			generatedBodies = append(generatedBodies, decl.Body)
		}
	}
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	if len(generatedBodies) != 2 {
		t.Fatalf("generated function literal decls = %#v, want two", generatedBodies)
	}
	for _, bad := range []string{"x Values", "x [2]int", "rtg_example_com_app_appMain_func_literal_0(direct)", "rtg_example_com_app_appMain_func_literal_1(values)"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("array function literal form leaked into appMain as %q:\n%s", bad, appBody)
		}
		for _, body := range generatedBodies {
			if strings.Contains(body, bad) {
				t.Fatalf("array function literal form leaked into generated body as %q:\n%s", bad, body)
			}
		}
	}
	for _, want := range []string{
		"func rtg_example_com_app_appMain_func_literal_0(x []int) int",
		"func rtg_example_com_app_appMain_func_literal_1(x []int) int",
		"x[0] = 20",
		"x[0] = 30",
		"values := []int{2, 3}",
		"direct := []int{1, 1}",
		"rtg_example_com_app_appMain_tmp_1 := append(rtg_example_com_app_appMain_tmp_0, values...)",
		"_ = rtg_example_com_app_appMain_func_literal_1(rtg_example_com_app_appMain_tmp_1)",
		"rtg_example_com_app_appMain_tmp_3 := append(rtg_example_com_app_appMain_tmp_2, direct...)",
		"return rtg_example_com_app_appMain_func_literal_0(rtg_example_com_app_appMain_tmp_3) + values[0] + direct[0] - 34",
	} {
		found := strings.Contains(appBody, want)
		if !found {
			for _, body := range generatedBodies {
				if strings.Contains(body, want) {
					found = true
					break
				}
			}
		}
		if !found {
			t.Fatalf("missing array function literal lowering fragment %q in appMain:\n%s\ngenerated: %#v", want, appBody, generatedBodies)
		}
	}
}

func TestPackageWithGraphLowersImportedStaticFunctionAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

func appMain() int {
	f := answer.Value
	_ = answer.Value
	_ = f
	return f()
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 {
		t.Fatalf("references = %#v, want one", u.References)
	}
	ref := u.References[0]
	if ref.ImportPath != "example.com/app/pkg/answer" || ref.Name != "Value" || ref.UnitName != "rtg_example_com_app_pkg_answer_Value" {
		t.Fatalf("reference = %#v", ref)
	}
	for _, bad := range []string{"f := answer.Value", "_ = answer.Value", "_ = f", "f()"} {
		if strings.Contains(u.Decls[0].Body, bad) {
			t.Fatalf("imported function alias leaked into lowered body as %q: %q", bad, u.Decls[0].Body)
		}
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("imported function alias call was not lowered: %q", u.Decls[0].Body)
	}
}

func TestPackageWithGraphLowersImportedStaticCallbacks(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

func apply(f func(int, int) int, value int) int { return f(value, 1) }
func applyBox(f func(answer.Box, int) int, b answer.Box) int { return f(b, 1) }

func appMain() int {
	f := answer.Add
	g := answer.Box.Add
	return apply(answer.Add, 2) + apply(f, 3) + applyBox(answer.Box.Add, answer.Box{Value: 4}) + applyBox(g, answer.Box{Value: 5}) - 18
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Add(a int, b int) int { return a + b }
type Box struct { Value int }
func (b Box) Add(a int) int { return b.Value + a }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	if bodyByName["apply"] != "" || bodyByName["applyBox"] != "" {
		t.Fatalf("unspecialized imported callback wrapper leaked into lowered decls: %#v", u.Decls)
	}
	appBody := bodyByName["appMain"]
	for _, bad := range []string{"f := answer.Add", "g := answer.Box.Add", "apply(answer.Add", "apply(f", "applyBox(answer.Box.Add", "applyBox(g"} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("imported callback leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	callbackName := "rtg_example_com_app_cmd_app_apply_callback_0"
	boxCallbackName := "rtg_example_com_app_cmd_app_applyBox_callback_1"
	callbackBody := bodyByName[callbackName]
	if callbackBody == "" {
		t.Fatalf("imported callback specialization missing from %#v", u.Decls)
	}
	if !strings.Contains(appBody, callbackName+"(2) + "+callbackName+"(3) + "+boxCallbackName+"(") || !strings.Contains(appBody, ") + "+boxCallbackName+"(") || !strings.Contains(appBody, ") - 18") {
		t.Fatalf("imported callback calls were not lowered:\n%s", appBody)
	}
	if !strings.Contains(callbackBody, "return rtg_example_com_app_pkg_answer_Add(value, 1)") {
		t.Fatalf("imported callback specialization was not rewritten correctly:\n%s", callbackBody)
	}
	boxCallbackBody := bodyByName[boxCallbackName]
	if boxCallbackBody == "" {
		t.Fatalf("imported method-expression callback specialization missing from %#v", u.Decls)
	}
	if !strings.Contains(boxCallbackBody, "return rtg_example_com_app_pkg_answer_Box_Add(b, 1)") {
		t.Fatalf("imported method-expression callback specialization was not rewritten correctly:\n%s", boxCallbackBody)
	}
	refFound := false
	methodRefFound := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/pkg/answer" && ref.Name == "Add" && ref.UnitName == "rtg_example_com_app_pkg_answer_Add" {
			refFound = true
		}
		if ref.ImportPath == "example.com/app/pkg/answer" && ref.Name == "Box_Add" && ref.UnitName == "rtg_example_com_app_pkg_answer_Box_Add" {
			methodRefFound = true
		}
	}
	if !refFound || !methodRefFound {
		t.Fatalf("imported callback reference missing from %#v", u.References)
	}
}

func TestPackageWithGraphLowersImportedNamedFunctionTypeStaticCallbackWrappers(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

func appMain() int {
	f := answer.Inc
	return answer.Apply(answer.Inc, 2) + answer.Apply(f, 3) + answer.ApplyAlias(answer.Inc, 4) - 15
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

type Unary func(int) int
type AlsoUnary Unary

func Apply(f Unary, value int) int { return f(Bias(value)) }
func ApplyAlias(f AlsoUnary, value int) int { return f(Bias(value)) }
func Bias(x int) int { return x + 1 }
func Inc(x int) int { return x + 1 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	bodyByName := map[string]string{}
	for _, decl := range u.Decls {
		bodyByName[decl.Name] = decl.Body
	}
	appBody := bodyByName["appMain"]
	if appBody == "" {
		t.Fatalf("appMain decl missing from %#v", u.Decls)
	}
	for _, bad := range []string{"f := answer.Inc", "answer.Apply(", "answer.ApplyAlias("} {
		if strings.Contains(appBody, bad) {
			t.Fatalf("imported named callback wrapper leaked into app body as %q:\n%s", bad, appBody)
		}
	}
	applyCallback := "rtg_example_com_app_pkg_answer_Apply_callback_0"
	applyAliasCallback := "rtg_example_com_app_pkg_answer_ApplyAlias_callback_1"
	if !strings.Contains(appBody, "return "+applyCallback+"(2) + "+applyCallback+"(3) + "+applyAliasCallback+"(4) - 15") {
		t.Fatalf("imported named callback calls were not lowered:\n%s", appBody)
	}
	if !strings.Contains(bodyByName[applyCallback], "func "+applyCallback+"(value int) int") || !strings.Contains(bodyByName[applyCallback], "return rtg_example_com_app_pkg_answer_Inc(rtg_example_com_app_pkg_answer_Bias(value))") {
		t.Fatalf("imported named callback specialization was not rewritten correctly:\n%s", bodyByName[applyCallback])
	}
	if !strings.Contains(bodyByName[applyAliasCallback], "func "+applyAliasCallback+"(value int) int") || !strings.Contains(bodyByName[applyAliasCallback], "return rtg_example_com_app_pkg_answer_Inc(rtg_example_com_app_pkg_answer_Bias(value))") {
		t.Fatalf("imported named alias callback specialization was not rewritten correctly:\n%s", bodyByName[applyAliasCallback])
	}
	refFound := false
	biasRefFound := false
	for _, ref := range u.References {
		if ref.ImportPath == "example.com/app/pkg/answer" && ref.Name == "Inc" && ref.UnitName == "rtg_example_com_app_pkg_answer_Inc" {
			refFound = true
		}
		if ref.ImportPath == "example.com/app/pkg/answer" && ref.Name == "Bias" && ref.UnitName == "rtg_example_com_app_pkg_answer_Bias" {
			biasRefFound = true
		}
		if ref.ImportPath == "example.com/app/pkg/answer" && (ref.Name == "Apply" || ref.Name == "ApplyAlias") {
			t.Fatalf("unspecialized imported callback wrapper reference leaked: %#v", u.References)
		}
	}
	if !refFound {
		t.Fatalf("imported callback target reference missing from %#v", u.References)
	}
	if !biasRefFound {
		t.Fatalf("imported callback helper reference missing from %#v", u.References)
	}
}

func TestPackageWithGraphRewritesDotImportIdentifiers(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import . "example.com/app/pkg/answer"

func appMain() int {
	var box Box
	box.Value = Value() + Number
	return box.Value + localShadow()
}

func localShadow() int {
	Value := 1
	return Value
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

const Number = 2
type Box struct { Value int }
func Value() int { return 39 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := u.Decls[0].Body
	for _, want := range []string{
		"var box rtg_example_com_app_pkg_answer_Box",
		"rtg_example_com_app_pkg_answer_Value()",
		"rtg_example_com_app_pkg_answer_Number",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing dot-import rewrite %q in: %q", want, body)
		}
	}
	if strings.Contains(u.Decls[1].Body, "rtg_example_com_app_pkg_answer_Value :=") {
		t.Fatalf("local dot-import shadow was rewritten: %q", u.Decls[1].Body)
	}
}

func TestPackageWithGraphUsesFileScopedImportAliases(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "a.go",
				Source: []byte(`package main

import first "example.com/app/pkg/answer"

func A() int { return first.Value() }
`),
			},
			{
				Path: "b.go",
				Source: []byte(`package main

import second "example.com/app/pkg/answer"

func appMain() int { return A() + second.Value() }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 || u.References[0].Name != "Value" {
		t.Fatalf("references = %#v, want one Value reference", u.References)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("first alias was not rewritten: %q", u.Decls[0].Body)
	}
	if !strings.Contains(u.Decls[1].Body, "return rtg_example_com_app_cmd_app_A() + rtg_example_com_app_pkg_answer_Value()") {
		t.Fatalf("second alias was not rewritten: %q", u.Decls[1].Body)
	}
	if strings.Contains(u.Decls[0].Body, "first.") || strings.Contains(u.Decls[1].Body, "second.") {
		t.Fatalf("alias selector leaked into lowered bodies: %#v", u.Decls)
	}
}

func TestPackageWithGraphDoesNotLeakImportAliasAcrossFiles(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "a.go",
				Source: []byte(`package main

import answer "example.com/app/pkg/answer"

func imported() int { return answer.Value() }
`),
			},
			{
				Path: "b.go",
				Source: []byte(`package main

type localAnswer struct { Value int }

func appMain() int {
	answer := localAnswer{Value: 3}
	return answer.Value + imported()
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 || u.References[0].Name != "Value" {
		t.Fatalf("references = %#v, want only imported file reference", u.References)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "return answer.Value + rtg_example_com_app_cmd_app_imported()") {
		t.Fatalf("local selector was rewritten by leaked import alias: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_answer_Value") {
		t.Fatalf("local selector contains imported symbol: %q", body)
	}
}

func TestPackageWithGraphPreservesLocalImportNameShadow(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/answer"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/answer"

type localAnswer struct { Value int }

func appMain() int {
	answer := localAnswer{Value: 3}
	return answer.Value
}
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/pkg/answer",
		Name:       "answer",
		Files: []load.File{
			{
				Path: "answer.go",
				Source: []byte(`package answer

func Value() int { return 7 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 0 {
		t.Fatalf("references = %#v, want none for local shadow", u.References)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "answer := rtg_example_com_app_cmd_app_localAnswer{Value: 3}") {
		t.Fatalf("local shadow declaration was not preserved: %q", body)
	}
	if !strings.Contains(body, "return answer.Value") {
		t.Fatalf("local selector was rewritten as import reference: %q", body)
	}
	if strings.Contains(body, "rtg_example_com_app_pkg_answer_Value") {
		t.Fatalf("local selector contains imported symbol: %q", body)
	}
}

func TestPackagePreservesLocalShadowNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7
const total = 9

func Use(answer int) int {
	return answer
}

func appMain() int {
	answer := 1
	var total int
	total = answer
	return Use(total)
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want 4", u.Decls)
	}
	use := u.Decls[2].Body
	if !strings.Contains(use, "func rtg_example_com_app_Use(answer int) int") {
		t.Fatalf("Use signature rewrote parameter name: %q", use)
	}
	if !strings.Contains(use, "return answer") {
		t.Fatalf("Use body rewrote parameter reference: %q", use)
	}
	appMain := u.Decls[3].Body
	if !strings.Contains(appMain, "answer := 1") {
		t.Fatalf("appMain rewrote local short declaration: %q", appMain)
	}
	if !strings.Contains(appMain, "var total int") || !strings.Contains(appMain, "total = answer") {
		t.Fatalf("appMain rewrote local var declaration: %q", appMain)
	}
	if !strings.Contains(appMain, "return rtg_example_com_app_Use(total)") {
		t.Fatalf("appMain did not preserve local argument while rewriting callee: %q", appMain)
	}
	if strings.Contains(appMain, "rtg_example_com_app_answer := 1") {
		t.Fatalf("appMain rewrote local shadow as package symbol: %q", appMain)
	}
	if strings.Contains(appMain, "var rtg_example_com_app_total int") {
		t.Fatalf("appMain rewrote var shadow as package symbol: %q", appMain)
	}
}

func TestPackageRewritesPackageNameBeforeLocalShadow(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7

func appMain() int {
	before := answer
	answer := 1
	return before + answer
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", u.Decls)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "before := rtg_example_com_app_answer") {
		t.Fatalf("package reference before local shadow was not rewritten: %q", body)
	}
	if !strings.Contains(body, "answer := 1") {
		t.Fatalf("local short declaration was rewritten: %q", body)
	}
	if !strings.Contains(body, "return before + answer") {
		t.Fatalf("local reference after shadow was rewritten: %q", body)
	}
}

func TestPackageRewritesPackageNameAfterInnerBlockShadow(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

const answer = 7

func appMain() int {
	if answer > 0 {
		answer := 1
		_ = answer
	}
	return answer
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 2 {
		t.Fatalf("decls = %#v, want 2", u.Decls)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "if rtg_example_com_app_answer > 0") {
		t.Fatalf("package reference before block shadow was not rewritten: %q", body)
	}
	if !strings.Contains(body, "answer := 1") || !strings.Contains(body, "_ = answer") {
		t.Fatalf("inner block local shadow was rewritten: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_answer") {
		t.Fatalf("package reference after inner block shadow was not rewritten: %q", body)
	}
}

func TestPackageNormalizesNestedReturnCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int { return join(first(), second()) }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	if len(u.Decls) != 4 {
		t.Fatalf("decls = %#v, want 4", u.Decls)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("return call did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesDeepNestedReturnCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func inner(v int) int { return v + 1 }
func outer(v int) int { return v + 1 }
func appMain() int { return outer(inner(first())) }
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	first := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()")
	inner := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_inner(rtg_example_com_app_appMain_tmp_0)")
	ret := strings.Index(body, "return rtg_example_com_app_outer(rtg_example_com_app_appMain_tmp_1)")
	if first < 0 || inner < 0 || ret < 0 {
		t.Fatalf("deep nested calls were not fully normalized: %q", body)
	}
	if !(first < inner && inner < ret) {
		t.Fatalf("deep nested call temps emitted in wrong order: %q", body)
	}
}

func TestPackageNormalizesNestedImportedReturnCallArguments(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int { return dep.Join(dep.First(), dep.Second()) }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 3 {
		t.Fatalf("references = %#v, want First, Join, Second", u.References)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_dep_First()") {
		t.Fatalf("first imported call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_dep_Second()") {
		t.Fatalf("second imported call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return rtg_example_com_app_dep_Join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("return call did not use imported lifted temps: %q", body)
	}
}

func TestPackageLowersFormattedErrorfToSubsetErrorf(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "fmt"

func fail(name string) error {
	return fmt.Errorf("bad %s", name)
}
`),
			},
		},
	}
	fmtPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func Errorf(format string) error { return nil }
type Error string
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, fmtPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, `return rtg_fmt_Error("bad %s")`) {
		t.Fatalf("fmt.Errorf was not lowered to subset call: %q", body)
	}
	if strings.Contains(body, ", name") {
		t.Fatalf("formatted fmt.Errorf retained non-subset operands: %q", body)
	}
}

func TestPackageLowersErrorMethodToStringConversion(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type parseError string

func (err parseError) Error() string {
	return string(err)
}

func fail() error {
	return parseError("bad")
}

func appMain() int {
	err := fail()
	return len(err.Error())
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
		}
	}
	if body == "" {
		t.Fatalf("missing lowered appMain declaration: %#v", u.Decls)
	}
	if !strings.Contains(body, "string(err)") || strings.Contains(body, ".Error()") {
		t.Fatalf("error Error method was not lowered: %q", body)
	}
}

func TestPackageNormalizesCallOperandInMultiReturn(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "fmt"

type config struct { output string }

func parse() (config, error) {
	cfg := config{}
	return cfg, fmt.Errorf("missing %s", cfg.output)
}
`),
			},
		},
	}
	fmtPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func Errorf(format string) error { return nil }
type Error string
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, fmtPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, `rtg_example_com_app_parse_tmp_0 := rtg_fmt_Error("missing %s")`) {
		t.Fatalf("multi-return call operand was not lifted: %q", body)
	}
	if !strings.Contains(body, "return cfg, rtg_example_com_app_parse_tmp_0") {
		t.Fatalf("multi-return did not use lifted call operand: %q", body)
	}
}

func TestPackageDoesNotLiftConversionOperandInMultiReturn(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func text(out []byte) (string, error) {
	return string(out), nil
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "_tmp_") {
		t.Fatalf("builtin conversion operand was lifted unexpectedly: %q", body)
	}
	if !strings.Contains(body, "return string(out), nil") {
		t.Fatalf("builtin conversion return changed unexpectedly: %q", body)
	}
}

func TestPackageLowersFmtFprintToBuiltinPrint(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt", "os"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import (
	"fmt"
	"os"
)

func appMain() int {
	fmt.Fprint(os.Stdout, "PASS\n")
	return 0
}
`),
			},
		},
	}
	fmtPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func Fprint(fd int, s string) int { return 0 }
`),
			},
		},
	}
	osPkg := load.Package{
		ImportPath: "os",
		Name:       "os",
		Files: []load.File{
			{
				Path: "os.go",
				Source: []byte(`package os

const Stdout = 1
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, fmtPkg, osPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, `print("PASS\n")`) {
		t.Fatalf("fmt.Fprint was not lowered to builtin print: %q", body)
	}
}

func TestPackageLowersFmtFprintByteStringToWrite(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt", "os"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import (
	"fmt"
	"os"
)

func appMain() int {
	data := []byte{80, 65, 83, 83, 10}
	fmt.Fprint(os.Stdout, bytesToString(data))
	return 0
}

func bytesToString(data []byte) string {
	return string(data)
}
`),
			},
		},
	}
	fmtPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func Fprint(fd int, s string) int { return 0 }
`),
			},
		},
	}
	osPkg := load.Package{
		ImportPath: "os",
		Name:       "os",
		Files: []load.File{
			{
				Path: "os.go",
				Source: []byte(`package os

const Stdout = 1
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, fmtPkg, osPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := u.Decls[0].Body
	if !strings.Contains(body, "write(1, data, 0)") {
		t.Fatalf("fmt.Fprint byte-string output was not lowered to write: %q", body)
	}
}

func TestPackageNormalizesNestedAssignmentCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := join(first(), second())
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first assignment call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second assignment call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("assignment did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesNestedVarInitializerCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	var total = join(first(), second())
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first var initializer call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second var initializer call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("var initializer did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesIndexBoundCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func index() int { return 1 }
func appMain() int {
	values := []int{10, 20}
	return values[index()]
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[1].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_index()") {
		t.Fatalf("index bound call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return values[rtg_example_com_app_appMain_tmp_0]") {
		t.Fatalf("index expression did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesSliceBoundCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func start() int { return 1 }
func end() int { return 5 }
func appMain() string {
	text := "xPASSx"
	return text[start():end()]
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_start()") {
		t.Fatalf("slice start call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_end()") {
		t.Fatalf("slice end call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "return text[rtg_example_com_app_appMain_tmp_0:rtg_example_com_app_appMain_tmp_1]") {
		t.Fatalf("slice expression did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesSliceBoundCallsInsideCallArgument(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func start() int { return 1 }
func end() int { return 5 }
func consume(text string) {}
func appMain() int {
	text := "xPASSx"
	consume(text[start():end()])
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	start := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_start()")
	end := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_end()")
	arg := strings.Index(body, "rtg_example_com_app_appMain_tmp_2 := text[rtg_example_com_app_appMain_tmp_0:rtg_example_com_app_appMain_tmp_1]")
	call := strings.Index(body, "rtg_example_com_app_consume(rtg_example_com_app_appMain_tmp_2)")
	if start < 0 || end < 0 || arg < 0 || call < 0 {
		t.Fatalf("slice bound calls inside call argument were not fully normalized: %q", body)
	}
	if !(start < end && end < arg && arg < call) {
		t.Fatalf("slice bound call argument temps emitted in wrong order: %q", body)
	}
}

func TestPackageNormalizesNestedIfConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	if join(first(), second()) == 12 {
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "if rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12 {") {
		t.Fatalf("condition did not use lifted temps: %q", body)
	}
}

func TestPackagePreservesShortCircuitConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

type Decl struct { Body string }
type Unit struct { Decls []Decl }
func declAt(u Unit, index int) Decl { return u.Decls[index] }
func bodyAt(u Unit, index int) string {
	decl := declAt(u, index)
	return decl.Body
}
func check(value string) bool { return value != "" }
func done(decl Decl) bool { return decl.Body == "" }
func appMain(args []string, env []string) int {
	var u Unit
	currentDecl := -1
	if currentDecl >= 0 && check(bodyAt(u, currentDecl)) && !done(declAt(u, currentDecl)) {
		return 1
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[len(u.Decls)-1].Body
	firstGuard := strings.Index(body, "if currentDecl >= 0 {")
	bodyTemp := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_bodyAt(u, currentDecl)")
	bodyGuard := strings.Index(body, "if rtg_example_com_app_check(rtg_example_com_app_appMain_tmp_0) {")
	declTemp := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_declAt(u, currentDecl)")
	declGuard := strings.Index(body, "if !rtg_example_com_app_done(rtg_example_com_app_appMain_tmp_1) {")
	if firstGuard < 0 || bodyTemp < 0 || bodyGuard < 0 || declTemp < 0 || declGuard < 0 {
		t.Fatalf("short-circuit condition was not lowered into nested guarded temps: %q", body)
	}
	if !(firstGuard < bodyTemp && bodyTemp < bodyGuard && bodyGuard < declTemp && declTemp < declGuard) {
		t.Fatalf("short-circuit condition temps were emitted before their guards: %q", body)
	}
}

func TestPackageNormalizesNestedIfShortStatementCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	if total := join(first(), second()); total == 12 {
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first short-statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second short-statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)\n\t\tif total == 12 {") {
		t.Fatalf("if short statement did not use lifted temps: %q", body)
	}
}

func TestPackageDoesNotHoistIfShortConditionCallsBeforeInit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func check(v int) bool { return v == 2 }
func appMain() int {
	if total := first(); check(next(total)) {
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	init := strings.Index(body, "total := rtg_example_com_app_first()")
	temp := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_next(total)")
	condition := strings.Index(body, "if rtg_example_com_app_check(rtg_example_com_app_appMain_tmp_0) {")
	if init < 0 || temp < 0 || condition < 0 {
		t.Fatalf("if short condition was not lowered into scoped temporaries: %q", body)
	}
	if !(init < temp && temp < condition) {
		t.Fatalf("if short condition temp was not after init and before condition: %q", body)
	}
}

func TestPackageNormalizesNestedSwitchTagCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	switch join(first(), second()) {
	case 12:
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first switch tag call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second switch tag call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "switch rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) {") {
		t.Fatalf("switch tag did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesNestedSwitchShortStatementCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	switch total := join(first(), second()); total {
	case 12:
		return 0
	}
	return 1
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first switch short statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second switch short statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)\n\t\tswitch total {") {
		t.Fatalf("switch short statement did not use lifted temps: %q", body)
	}
}

func TestPackageDoesNotHoistSwitchShortTagCallsBeforeInit(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	switch total := first(); next(total) {
	case 2:
		return total
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	init := strings.Index(body, "total := rtg_example_com_app_first()")
	tag := strings.Index(body, "switch rtg_example_com_app_next(total) {")
	if init < 0 || tag < 0 {
		t.Fatalf("switch short tag was not lowered into scoped form: %q", body)
	}
	if !(init < tag) {
		t.Fatalf("switch short tag was emitted before init: %q", body)
	}
}

func TestPackageNormalizesNestedCallStatementArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func consume(a int, b int) {}
func appMain() int {
	consume(first(), second())
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second statement call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_consume(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("call statement did not use lifted temps: %q", body)
	}
}

func TestPackageNormalizesForPostClauseCallArgumentsWithoutContinue(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	for i := 0; i < 3; i = next(first()) {
		_ = i
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "{\n\t\ti := 0\n\t\tfor {") {
		t.Fatalf("for post clause was not lowered into scoped loop: %q", body)
	}
	if !strings.Contains(body, "if !(i < 3) {\n\t\t\t\tbreak\n\t\t\t}") {
		t.Fatalf("for post condition guard was not emitted: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("for post nested call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_0)") {
		t.Fatalf("for post clause did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesClassicForConditionAndPostCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a + b }
func appMain() int {
	total := 0
	for i := 0; join(first(), second()) < 4; i = join(first(), second()) {
		total = total + i
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	condFirst := strings.Index(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()")
	condSecond := strings.Index(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()")
	guard := strings.Index(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) < 4)")
	postFirst := strings.Index(body, "rtg_example_com_app_appMain_tmp_2 := rtg_example_com_app_first()")
	postSecond := strings.Index(body, "rtg_example_com_app_appMain_tmp_3 := rtg_example_com_app_second()")
	postAssign := strings.Index(body, "i = rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_2, rtg_example_com_app_appMain_tmp_3)")
	if condFirst < 0 || condSecond < 0 || guard < 0 || postFirst < 0 || postSecond < 0 || postAssign < 0 {
		t.Fatalf("classic for condition/post calls were not all normalized: %q", body)
	}
	if !(condFirst < condSecond && condSecond < guard && guard < postFirst && postFirst < postSecond && postSecond < postAssign) {
		t.Fatalf("classic for condition/post temps emitted in wrong order: %q", body)
	}
}

func TestPackageDoesNotNormalizeForPostClauseWithContinue(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	for i := 0; i < 3; i = next(first()) {
		if i == 1 {
			continue
		}
		_ = i
	}
	return 0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "_tmp_") {
		t.Fatalf("for post clause with continue was normalized unsafely: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_first())") {
		t.Fatalf("for post clause with continue shape changed unexpectedly: %q", body)
	}
}

func TestPackageNormalizesNestedForConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := 0
	for join(first(), second()) == 12 {
		total = total + 1
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "for {\n") {
		t.Fatalf("for condition was not rewritten to loop body guard: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first for condition call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second for condition call was not lifted into loop body temp: %q", body)
	}
	if !strings.Contains(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12) {\n\t\t\tbreak\n\t\t}") {
		t.Fatalf("for condition guard did not use lifted temps: %q", body)
	}
	if !strings.Contains(body, "total = total + 1") {
		t.Fatalf("for body was not preserved: %q", body)
	}
}

func TestPackageNormalizesClassicForInitCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func next(v int) int { return v + 1 }
func appMain() int {
	total := 0
	for i := next(first()); i < 3; i = next(first()) {
		total = total + i
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("classic for init call was not lifted before loop: %q", body)
	}
	if !strings.Contains(body, "i := rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_0)\n\t\tfor {") {
		t.Fatalf("classic for init did not use lifted temp in scoped loop: %q", body)
	}
	if !strings.Contains(body, "if !(i < 3) {\n\t\t\t\tbreak\n\t\t\t}") {
		t.Fatalf("classic for condition guard was not emitted: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_first()") {
		t.Fatalf("classic for post call was not lifted into loop body: %q", body)
	}
	if !strings.Contains(body, "i = rtg_example_com_app_next(rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("classic for post clause did not use lifted temp: %q", body)
	}
}

func TestPackageNormalizesClassicForConditionCallArguments(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func second() int { return 2 }
func join(a int, b int) int { return a*10 + b }
func appMain() int {
	total := 0
	for i := 0; join(first(), second()) == 12; i = i + 1 {
		total = total + i
		break
	}
	return total
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[3].Body
	if !strings.Contains(body, "for i := 0; ; i = i + 1 {") {
		t.Fatalf("classic for condition was not moved into loop body guard: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("first classic for condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_second()") {
		t.Fatalf("second classic for condition call was not lifted into a temp: %q", body)
	}
	if !strings.Contains(body, "if !(rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_0, rtg_example_com_app_appMain_tmp_1) == 12) {\n\t\t\tbreak\n\t\t}") {
		t.Fatalf("classic for condition guard did not use lifted temps: %q", body)
	}
	if !strings.Contains(body, "total = total + i") {
		t.Fatalf("classic for body was not preserved: %q", body)
	}
}

func TestPackageNormalizesWithNonCollidingTempNames(t *testing.T) {
	pkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

func first() int { return 1 }
func join(a int) int { return a }
func appMain() int {
	rtg_example_com_app_appMain_tmp_0 := 99
	total := join(first())
	return total + rtg_example_com_app_appMain_tmp_0
}
`),
			},
		},
	}
	u, err := Package(pkg)
	if err != nil {
		t.Fatalf("Package failed: %v", err)
	}
	body := u.Decls[2].Body
	if strings.Contains(body, "rtg_example_com_app_appMain_tmp_0 := rtg_example_com_app_first()") {
		t.Fatalf("normalization reused an existing local name: %q", body)
	}
	if !strings.Contains(body, "rtg_example_com_app_appMain_tmp_1 := rtg_example_com_app_first()") {
		t.Fatalf("normalization did not skip colliding temp name: %q", body)
	}
	if !strings.Contains(body, "total := rtg_example_com_app_join(rtg_example_com_app_appMain_tmp_1)") {
		t.Fatalf("assignment did not use non-colliding temp: %q", body)
	}
}

func TestPackageWithGraphRewritesStdSelector(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"fmt"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "fmt"

func appMain() int { return fmt.PrintInt(7) }
`),
			},
		},
	}
	stdPkg := load.Package{
		ImportPath: "fmt",
		Name:       "fmt",
		Files: []load.File{
			{
				Path: "fmt.go",
				Source: []byte(`package fmt

func PrintInt(v int) int { return v }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, stdPkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 1 {
		t.Fatalf("references = %#v, want one", u.References)
	}
	ref := u.References[0]
	if ref.ImportPath != "fmt" || ref.Name != "PrintInt" || ref.UnitName != "rtg_fmt_PrintInt" {
		t.Fatalf("reference = %#v", ref)
	}
	if !strings.Contains(u.Decls[0].Body, "return rtg_fmt_PrintInt(7)") {
		t.Fatalf("std selector was not rewritten: %q", u.Decls[0].Body)
	}
}

func TestPackageWithGraphLowersUnsafeSizeof(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"unsafe"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import u "unsafe"

func appMain() int {
	var b byte = 1
	type LocalByte ByteAlias
	var lb LocalByte = 1
	xs := []byte{1, 2}
	s := "x"
	p := &b
	return u.Sizeof(g) + u.Sizeof(ag) + u.Sizeof(ax) + u.Sizeof(ap) + u.Sizeof(b) + u.Sizeof(lb) + u.Sizeof(xs) + u.Sizeof(s) + u.Sizeof(p) + u.Sizeof(byte(1)) + u.Sizeof(ByteAlias(1)) + u.Sizeof(ByteAlias2(1)) + u.Sizeof(LocalByte(1)) + u.Sizeof(WordAlias(1)) + u.Sizeof(int16(1)) + u.Sizeof(int32(1)) + u.Sizeof(int64(1)) + u.Sizeof(float64(1)) + u.Sizeof(true) + u.Sizeof("x") + u.Sizeof(&b)
}
`),
			},
			{
				Path: "types.go",
				Source: []byte(`package main

type ByteAlias byte
type ByteAlias2 ByteAlias
type BytesAlias []byte
type PtrAlias *byte
type WordAlias int16

var g byte = 1
var ag ByteAlias = 1
var ax BytesAlias = BytesAlias{1, 2}
var ap PtrAlias
`),
			},
		},
	}
	unsafePkg := load.Package{
		ImportPath: "unsafe",
		Name:       "unsafe",
		Files: []load.File{
			{
				Path: "unsafe.go",
				Source: []byte(`package unsafe

func Sizeof(value int) int { return 0 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, unsafePkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 0 {
		t.Fatalf("references = %#v, want none", u.References)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	if strings.Contains(body, "Sizeof") || strings.Contains(body, "rtg_unsafe") {
		t.Fatalf("unsafe.Sizeof leaked into lowered body: %q", body)
	}
	want := `return 1 + 1 + 24 + 8 + 1 + 1 + 24 + 16 + 8 + 1 + 1 + 1 + 1 + 2 + 2 + 4 + 8 + 8 + 1 + 16 + 8`
	if !strings.Contains(body, want) {
		t.Fatalf("unsafe.Sizeof constants missing %q in: %q", want, body)
	}
}

func TestPackageWithGraphLowersDotImportedUnsafeSizeof(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"unsafe"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import . "unsafe"

func appMain() int {
	return Sizeof(byte(1)) + Sizeof("x")
}
`),
			},
		},
	}
	unsafePkg := load.Package{
		ImportPath: "unsafe",
		Name:       "unsafe",
		Files: []load.File{
			{
				Path: "unsafe.go",
				Source: []byte(`package unsafe

func Sizeof(value int) int { return 0 }
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, unsafePkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	if len(u.References) != 0 {
		t.Fatalf("references = %#v, want none", u.References)
	}
	body := u.Decls[0].Body
	if strings.Contains(body, "Sizeof") || strings.Contains(body, "rtg_unsafe") {
		t.Fatalf("unsafe.Sizeof leaked into lowered body: %q", body)
	}
	if !strings.Contains(body, `return 1 + 16`) {
		t.Fatalf("dot-imported unsafe.Sizeof constants missing in: %q", body)
	}
}

func TestPackageWithGraphLowersUnsafeSizeofTargetWordOperands(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"unsafe"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "unsafe"

type MyInt int
type Matrix [2][3]int

func appMain() int {
	var x MyInt = 1
	xs := []byte{1, 2}
	return unsafe.Sizeof(1) + unsafe.Sizeof(x) + unsafe.Sizeof(MyInt(1)) + unsafe.Sizeof(&x) + unsafe.Sizeof("x") + unsafe.Sizeof(xs) + unsafe.Sizeof([3]int{1, 2, 3}) + unsafe.Sizeof([2]string{"a", "b"}) + unsafe.Sizeof([2][3]int{{1, 2, 3}, {4, 5, 6}}) + unsafe.Sizeof(Matrix{})
}
`),
			},
		},
	}
	unsafePkg := load.Package{
		ImportPath: "unsafe",
		Name:       "unsafe",
		Files: []load.File{
			{
				Path: "unsafe.go",
				Source: []byte(`package unsafe

func Sizeof(value int) int { return 0 }
`),
			},
		},
	}
	graph := &load.Graph{Target: "linux/386", Packages: []load.Package{mainPkg, unsafePkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	if strings.Contains(body, "Sizeof") || strings.Contains(body, "rtg_unsafe") {
		t.Fatalf("unsafe.Sizeof leaked into lowered body: %q", body)
	}
	want := `return 4 + 4 + 4 + 4 + 8 + 12 + 12 + 16 + 24 + 24`
	if !strings.Contains(body, want) {
		t.Fatalf("unsafe.Sizeof target constants missing %q in: %q", want, body)
	}
}

func TestPackageWithGraphLowersUnsafeSizeofSimpleStructOperands(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"unsafe"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "unsafe"

type Inner struct {
	A byte
	B int16
}

type Box struct {
	A byte
	B int
	C string
	D Inner
	E *byte
	F []byte
	G [2]byte
}

type Alias Box

func appMain() int {
	var box Box
	return unsafe.Sizeof(Inner{}) + unsafe.Sizeof(Box{}) + unsafe.Sizeof(Alias{}) + unsafe.Sizeof(box) + unsafe.Sizeof([2]Box{})
}
`),
			},
		},
	}
	unsafePkg := load.Package{
		ImportPath: "unsafe",
		Name:       "unsafe",
		Files: []load.File{
			{
				Path: "unsafe.go",
				Source: []byte(`package unsafe

func Sizeof(value int) int { return 0 }
`),
			},
		},
	}
	graph := &load.Graph{Target: "linux/386", Packages: []load.Package{mainPkg, unsafePkg}}
	u, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph failed: %v", err)
	}
	body := ""
	for i := 0; i < len(u.Decls); i++ {
		if u.Decls[i].Name == "appMain" {
			body = u.Decls[i].Body
		}
	}
	if body == "" {
		t.Fatalf("missing appMain decl in %#v", u.Decls)
	}
	if strings.Contains(body, "Sizeof") || strings.Contains(body, "rtg_unsafe") {
		t.Fatalf("unsafe.Sizeof leaked into lowered body: %q", body)
	}
	want := `return 4 + 40 + 40 + 40 + 80`
	if !strings.Contains(body, want) {
		t.Fatalf("unsafe.Sizeof struct constants missing %q in: %q", want, body)
	}
}

func TestPackageWithGraphExportsGroupedDeclNames(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app",
		Name:       "main",
		Imports:    []string{"example.com/app/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/dep"

func appMain() int { return dep.Answer + dep.Next }
`),
			},
		},
	}
	depPkg := load.Package{
		ImportPath: "example.com/app/dep",
		Name:       "dep",
		Files: []load.File{
			{
				Path: "dep.go",
				Source: []byte(`package dep

const (
	Answer = 41
	Next = Answer + 1
)
`),
			},
		},
	}
	graph := &load.Graph{Packages: []load.Package{mainPkg, depPkg}}
	depUnit, err := PackageWithGraph(depPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph dep failed: %v", err)
	}
	if len(depUnit.Exports) != 2 || depUnit.Exports[0].Name != "Answer" || depUnit.Exports[1].Name != "Next" {
		t.Fatalf("dep exports = %#v", depUnit.Exports)
	}
	if !strings.Contains(depUnit.Decls[0].Body, "rtg_example_com_app_dep_Answer = 41") {
		t.Fatalf("grouped const Answer was not rewritten: %q", depUnit.Decls[0].Body)
	}
	if !strings.Contains(depUnit.Decls[0].Body, "rtg_example_com_app_dep_Next = rtg_example_com_app_dep_Answer + 1") {
		t.Fatalf("grouped const Next was not rewritten: %q", depUnit.Decls[0].Body)
	}
	mainUnit, err := PackageWithGraph(mainPkg, graph)
	if err != nil {
		t.Fatalf("PackageWithGraph main failed: %v", err)
	}
	if len(mainUnit.References) != 2 || mainUnit.References[0].Name != "Answer" || mainUnit.References[1].Name != "Next" {
		t.Fatalf("main references = %#v", mainUnit.References)
	}
	if !strings.Contains(mainUnit.Decls[0].Body, "return rtg_example_com_app_dep_Answer + rtg_example_com_app_dep_Next") {
		t.Fatalf("main body did not rewrite grouped refs: %q", mainUnit.Decls[0].Body)
	}
}
