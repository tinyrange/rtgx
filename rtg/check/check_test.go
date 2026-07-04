package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
)

func writeCheckFile(t *testing.T, root string, name string, data string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%q) failed: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile(%q) failed: %v", path, err)
	}
}

func TestFileDiagnosesExcludedAndUnimplementedForms(t *testing.T) {
	src := []byte(`package main

type Box[T any] struct { value T }
type Runner interface { Run() }
type fixed *[4]int
type holder struct { values [2]int }
func takesArray(values [3]int) int { return 0 }
func useMap(m map[string]int) int { return 0 }
func makeChan(ch chan int) { go makeChan(ch); select {} }
func appMain() int {
	var runner Runner
	_ = runner
	print(func() int { return 1 })
	for _, v := range []int{1, 2} {
		_ = v
	}
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
		"arrays are not supported",
		"channels are not supported",
		"goroutines are not supported",
		"select statements are not supported",
		"function values and function types are not supported",
	} {
		if !strings.Contains(messages, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, messages)
		}
	}
}

func TestGraphAllowsImportedArrayStructFieldComparisons(t *testing.T) {
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

func appMain() int {
	same := dep.Global.Values == [2]int{5, 6}
	different := dep.Global.Values != dep.Global.Values
	if same && !different {
		return 0
	}
	return 1
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

type Box struct {
	Values [2]int
}

var Global Box
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported array struct field comparisons: %v", err)
	}
}

func TestGraphAllowsImportedTopLevelArrayValues(t *testing.T) {
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
			},
			{
				ImportPath: "example.com/app/dep",
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported top-level array values: %v", err)
	}
}

func TestFileAllowsTopLevelDeferCall(t *testing.T) {
	file, err := parse.FileSource("defer_ok.go", []byte(`package main

func done(text string) {
	print(text)
}

func appMain() int {
	text := "PASS\n"
	defer done(text)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected top-level defer: %v", diags)
	}
}

func TestFileAllowsNestedDeferCalls(t *testing.T) {
	file, err := parse.FileSource("defer_nested_ok.go", []byte(`package main

func done(text string, code int) {
	print(text)
}

func many(text string, values ...int) {}
func pass() {}

func appMain() int {
	text := "done"
	values := []int{1, 2}
	defer many("top", values...)
	defer many("slice", []int{1}...)
	if true {
		defer done(text, 7)
		defer many(text, values...)
	}
	for i := 0; i < 2; i++ {
		defer done(text, i)
		defer pass()
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected nested defer: %v", diags)
	}
}

func TestFileAllowsDeferredFunctionLiterals(t *testing.T) {
	file, err := parse.FileSource("defer_literal_ok.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected deferred function literal: %v", diags)
	}
}

func TestFileRejectsUnsupportedDeferForms(t *testing.T) {
	file, err := parse.FileSource("defer_bad.go", []byte(`package main

func done(xs ...int) {}

func appMain() int {
	defer
	if true {
		defer done(1)
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported defer forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"defer_bad.go:6:2: defer requires a direct function call",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsMapTypesWithoutArrayDiagnostic(t *testing.T) {
	file, err := parse.FileSource("maps.go", []byte(`package main

type table map[string]map[string]int
func use(values map[string]int) int {
	print("x")
	values["key"] = 1
	_ = values["key"]
	values["other"] = values["key"]
	return values["key"]
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted map types")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "maps are not supported") {
		t.Fatalf("missing map diagnostic in:\n%s", msg)
	}
	if strings.Contains(msg, "arrays are not supported") {
		t.Fatalf("map types produced array diagnostic:\n%s", msg)
	}
}

func TestFileAllowsInertNamedMapTypes(t *testing.T) {
	file, err := parse.FileSource("map_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named map types: %v", diags)
	}
}

func TestFileAllowsInertMapContainingTypes(t *testing.T) {
	file, err := parse.FileSource("map_containing_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert map-containing types: %v", diags)
	}
}

func TestFileAllowsDiscardedEmptyMapContainingTypeComposites(t *testing.T) {
	file, err := parse.FileSource("discarded_empty_map_containing_type_composites.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded empty map-containing composites: %v", diags)
	}
}

func TestFileRejectsNamedMapTypeValues(t *testing.T) {
	file, err := parse.FileSource("map_type_values.go", []byte(`package main

type Table map[string]int

func appMain() int {
	var table Table
	_ = table
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named map type value")
	}
	if !strings.Contains(diags.Error(), "map_type_values.go:3:12: maps are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsMapContainingTypeValues(t *testing.T) {
	file, err := parse.FileSource("map_containing_type_values.go", []byte(`package main

type Box struct { M map[string]int }
type Rows []map[string]int

func appMain() int {
	var box Box
	_ = box
	_ = Box{M: map[string]int{}}
	_ = []Rows{Rows{}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted map-containing type values")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "maps are not supported") {
		t.Fatalf("missing map diagnostic in:\n%s", msg)
	}
}

func TestFileRejectsMixedGroupedMapTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("mixed_map_types.go", []byte(`package main

type (
	Table map[string]int
	Count int
)

func appMain() int {
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted mixed grouped map declaration")
	}
	if !strings.Contains(diags.Error(), "mixed_map_types.go:4:8: maps are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsMapForms(t *testing.T) {
	file, err := parse.FileSource("map_forms.go", []byte(`package main

type Box struct {
	M map[string]int
	Nested []map[string]int
}

func appMain() int {
	var m map[string]int
	_ = map[string]int{"a": sideEffect()}
	m = make(map[string]int)
	_ = m["a"]
	m["b"] = 2
	_, _ = m["c"]
	delete(m, "d")
	_ = len(m)
	_ = []map[string]int{{"x": 1}}
	return 0
}

func sideEffect() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported map forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_forms.go:9:8: maps are not supported",
		"map_forms.go:11:11: maps are not supported",
		"map_forms.go:15:2: unsupported builtin: delete",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	for _, unexpected := range []string{
		"index must be integer",
		"assignment count mismatch",
		"argument count mismatch in call to make",
	} {
		if strings.Contains(msg, unexpected) {
			t.Fatalf("unsupported map forms produced semantic cascade %q:\n%s", unexpected, msg)
		}
	}
	if strings.Contains(msg, "arrays are not supported") {
		t.Fatalf("map forms produced array diagnostic:\n%s", msg)
	}
}

func TestFileAllowsDiscardedPureMapLiterals(t *testing.T) {
	file, err := parse.FileSource("map_discard.go", []byte(`package main

func appMain() int {
	_ = map[string]int{"a": 1, "b": -2}
	_ = map[int]string{1: "a", 2: "b"}
	_ = map[bool]*int{true: nil, false: nil}
	_ = (map[byte]string{'x': "ok"})
	_ = map[string]map[string]int{"outer": map[string]int{"inner": 7}}
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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded pure map literals: %v", diags)
	}
}

func TestFileAllowsDiscardedMapLiteralsWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_discard_calls.go", []byte(`package main

func appMain() int {
	_ = map[string]int{"a": first(), "b": second(2)}
	_, _ = map[int]string{1: text()}, map[string]map[string]int{"outer": map[string]int{"inner": third()}}
	_ = []map[string]int{{"a": first()}, {"b": second(2)}, map[string]int{"c": third()}}
	_ = []map[string]int{make(map[string]int, fourth())}
	return 0
}

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }
func text() string { return "x" }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded map literals with direct-call values: %v", diags)
	}
}

func TestFileAllowsMapLiteralDeleteWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_delete_calls.go", []byte(`package main

func appMain() int {
	delete(map[string]int{"a": first(), "b": second(2)}, "a")
	delete(map[string]map[string]int{"outer": map[string]int{"inner": third()}}, "outer")
	fn := func() {
		delete(map[string]int{"inner": fourth()}, "inner")
	}
	fn()
	return 0
}

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected map literal delete with direct-call values: %v", diags)
	}
}

func TestFileRejectsImpureMapLiteralDelete(t *testing.T) {
	file, err := parse.FileSource("map_delete_bad.go", []byte(`package main

func appMain() int {
	delete(map[string]int{"a": 1}, sideEffectString())
	delete(make(map[string]int, sideEffect() + 1), "a")
	delete(make(map[string]int), sideEffectString())
	delete(map[string]int{"a": first() + second()}, "a")
	return 0
}

func sideEffect() int { return 1 }
func sideEffectString() string { return "a" }
func first() int { return 1 }
func second() int { return 2 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted impure map literal delete")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_delete_bad.go:4:2: unsupported builtin: delete",
		"map_delete_bad.go:5:2: unsupported builtin: delete",
		"map_delete_bad.go:6:2: unsupported builtin: delete",
		"map_delete_bad.go:7:2: unsupported builtin: delete",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsImpureMapSliceLiteralDiscard(t *testing.T) {
	file, err := parse.FileSource("map_slice_bad.go", []byte(`package main

func appMain() int {
	_ = []map[string]int{{sideEffectString(): 1}}
	_ = []map[string]int{dynamic()}
	return 0
}

func sideEffectString() string { return "a" }
func dynamic() map[string]int { return map[string]int{"a": 1} }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted impure map slice literal discard")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "maps are not supported") {
		t.Fatalf("missing map diagnostic in:\n%s", msg)
	}
}

func TestFileRejectsImpureMapMakeDiscard(t *testing.T) {
	file, err := parse.FileSource("map_make_bad.go", []byte(`package main

func appMain() int {
	_ = make(map[string]int, sideEffect() + 1)
	_ = make(map[string]int, "bad")
	return 0
}

func sideEffect() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted impure map make discard")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_make_bad.go:4:11: maps are not supported",
		"map_make_bad.go:5:11: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsMapMakeWithDirectCallCapacity(t *testing.T) {
	file, err := parse.FileSource("map_make_calls.go", []byte(`package main

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

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }
func sixth(v int) int { return v }
func seventh(v int) int { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected make(map) with direct-call capacity: %v", diags)
	}
}

func TestFileAllowsLenOfPureMapLiterals(t *testing.T) {
	file, err := parse.FileSource("map_len.go", []byte(`package main

func appMain() int {
	total := len(map[string]int{"a": 1, "b": -2})
	total = total + len(map[string]map[string]int{"outer": map[string]int{"inner": 7}})
	total = total + len((map[byte]string{'x': "ok"}))
	total = total + len(make(map[string]int))
	total = total + len(make(map[string]int, 4))
	total = total + len((make(map[byte]string, 0x10)))
	return total - 4
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected len of pure map literals: %v", diags)
	}
}

func TestFileRejectsImpureMapMakeLen(t *testing.T) {
	file, err := parse.FileSource("map_len_make_bad.go", []byte(`package main

func appMain() int {
	_ = len(make(map[string]int, sideEffect() + 1))
	_ = len(make(map[string]int, "bad"))
	return 0
}

func sideEffect() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted impure len(make(map))")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_len_make_bad.go:4:15: maps are not supported",
		"map_len_make_bad.go:5:15: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsLenOfSideEffectingMapLiteral(t *testing.T) {
	file, err := parse.FileSource("map_len_side_effect.go", []byte(`package main

func appMain() int {
	total := len(map[string]int{"a": first(), "b": second(2)})
	total = total + len(map[string]map[string]int{"outer": map[string]int{"inner": third()}})
	total = total + len(map[string][2]int{"array": [2]int{fourth(), 5}})
	return total - 3
}

func first() int { return 1 }
func second(v int) int { return v }
func third() int { return 3 }
func fourth() int { return 4 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected len of side-effecting map literal: %v", diags)
	}
}

func TestFileRejectsLenOfUnsupportedMapLiteral(t *testing.T) {
	file, err := parse.FileSource("map_len_bad.go", []byte(`package main

func appMain() int {
	_ = len(map[string]int{key(): 1})
	_ = len(map[string]int{"a": first() + second()})
	return 0
}

func key() string { return "a" }
func first() int { return 1 }
func second() int { return 2 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported len(map literal)")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_len_bad.go:4:10: maps are not supported",
		"map_len_bad.go:5:10: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPureMapLiteralIndex(t *testing.T) {
	file, err := parse.FileSource("map_index.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected pure map literal index: %v", diags)
	}
}

func TestFileAllowsMapLiteralIndexWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_index_calls.go", []byte(`package main

func appMain() int {
	total := map[string]int{"miss": first(1), "hit": second(2), "tail": third(3)}["hit"]
	total = total + map[string]int{"miss": fourth(4)}["missing"]
	fn := func() int {
		return map[string]int{"inner": fifth(5)}["inner"]
	}
	return total + fn() - 7
}

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected map literal index with direct-call values: %v", diags)
	}
}

func TestFileAllowsMapLiteralCommaOkWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_comma_ok_calls.go", []byte(`package main

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

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
func fifth(v int) int { return v }
func sixth(v int) int { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected map literal comma-ok with direct-call values: %v", diags)
	}
}

func TestFileAllowsPureMapRange(t *testing.T) {
	file, err := parse.FileSource("map_range.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected pure map range: %v", diags)
	}
}

func TestFileAllowsMapRangeWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_range_calls.go", []byte(`package main

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

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
func fourth(v int) int { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected map range with direct-call values: %v", diags)
	}
}

func TestFileAllowsPureStaticMapAliases(t *testing.T) {
	file, err := parse.FileSource("map_alias.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected pure static map aliases: %v", diags)
	}
}

func TestFileAllowsNamedMapTypeStaticAliases(t *testing.T) {
	file, err := parse.FileSource("named_map_alias.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected named map type static aliases: %v", diags)
	}
}

func TestGraphAllowsImportedNamedMapTypeStaticAliases(t *testing.T) {
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
			},
			{
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported named map type static aliases: %v", err)
	}
}

func TestFileAllowsStaticMapAliasLimitationProbes(t *testing.T) {
	file, err := parse.FileSource("map_alias_limitations.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static map alias limitation probes: %v", diags)
	}
}

func TestFileAllowsStaticMapAliasAssignmentsWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_alias_calls.go", []byte(`package main

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

func strings() int {
	suffix := "!"
	words := map[string]string{"a": "A"}
	words["b"] = text("B") + text("C") + suffix
	words["c"] = "D" + suffix
	got, ok := words["b"]
	if ok && words["a"] + got + words["c"] == "ABC!D!" {
		return 0
	}
	return 1
}

func first(v int) int { return v }
func second(v int) int { return v }
func text(v string) string { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static map alias assignments with direct-call values: %v", diags)
	}
}

func TestFileAllowsStaticMapAliasCompoundAssignments(t *testing.T) {
	file, err := parse.FileSource("map_alias_compound.go", []byte(`package main

func appMain() int {
	m := map[string]int{"a": 1}
	m["a"] += 2
	m["b"] += first(3)
	m["a"] *= second(4)
	m["b"] -= 1
	m["a"]++
	m["b"]--
	words := map[string]string{"x": "A"}
	words["x"] += text("B")
	words["y"] += text("C")
	return 0
}

func first(v int) int { return v }
func second(v int) int { return v }
func text(v string) string { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static map alias compound assignments: %v", diags)
	}
}

func TestFileRejectsStaticMapAliasIncDecForNonNumericValues(t *testing.T) {
	file, err := parse.FileSource("map_alias_incdec_bad.go", []byte(`package main

func appMain() int {
	words := map[string]string{"x": "A"}
	flags := map[string]bool{"ok": true}
	words["x"]++
	flags["ok"]--
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted static map alias inc/dec for non-numeric values")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "map mutation is not supported"); count != 2 {
		t.Fatalf("got %d static map inc/dec diagnostics, want 2:\n%s", count, msg)
	}
}

func TestFileAllowsStaticMapAliasInitializersWithDirectCallValues(t *testing.T) {
	file, err := parse.FileSource("map_alias_init_calls.go", []byte(`package main

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

func first(v int) int { return v }
func second(v int) int { return v }
func third(v int) int { return v }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static map alias initializers with direct-call values: %v", diags)
	}
}

func TestFileRejectsUnsupportedStaticMapAliasUses(t *testing.T) {
	file, err := parse.FileSource("map_alias_bad.go", []byte(`package main

func appMain() int {
	m := map[string]int{"a": 1}
	_ = m
	m["b"] = "bad"
	m["c"] = first() == second()
	bad := map[string]int{"x": first() + second()}
	_ = len(bad)
	return 0
}

func first() int { return 1 }
func second() int { return 2 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported static map alias uses")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_alias_bad.go:6:9: map mutation is not supported",
		"map_alias_bad.go:7:9: map mutation is not supported",
		"map_alias_bad.go:8:9: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsImpureMapRange(t *testing.T) {
	file, err := parse.FileSource("map_range_bad.go", []byte(`package main

func appMain() int {
	for k := range map[string]int{sideEffectString(): 1} {
		_ = k
	}
	for k := range map[string]int{"a": sideEffect() + 1} {
		_ = k
	}
	for k := range make(map[string]int, sideEffect() + 1) {
		_ = k
	}
	return 0
}

func sideEffect() int { return 1 }
func sideEffectString() string { return "a" }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted impure map range")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_range_bad.go:4:17: maps are not supported",
		"map_range_bad.go:7:17: maps are not supported",
		"map_range_bad.go:10:22: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsUnsupportedMapLiteralIndex(t *testing.T) {
	file, err := parse.FileSource("map_index_bad.go", []byte(`package main

func appMain() int {
	_ = map[string]int{sideEffectString(): 1}["a"]
	_ = map[string]int{"a": first() + second()}["a"]
	_ = make(map[string]int, sideEffect() + 1)["a"]
	_ = make(map[string]int)[sideEffectString()]
	return 0
}

func sideEffect() int { return 1 }
func sideEffectString() string { return "a" }
func first() int { return 1 }
func second() int { return 2 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported map literal index")
	}
	msg := diags.Error()
	for _, want := range []string{
		"map_index_bad.go:4:6: maps are not supported",
		"map_index_bad.go:5:6: maps are not supported",
		"map_index_bad.go:6:11: maps are not supported",
		"map_index_bad.go:7:11: maps are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsGenericInstantiations(t *testing.T) {
	file, err := parse.FileSource("generics.go", []byte(`package main

type Box struct { value int }
func id(value int) int { return value }

func appMain() int {
	_ = Box[int]{value: 1}
	_ = id[int](1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted generic instantiations")
	}
	msg := err.Error()
	for _, want := range []string{
		"generics.go:7:9: generics are not supported",
		"generics.go:8:8: generics are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsIndexExpressionsInComparisons(t *testing.T) {
	file, err := parse.FileSource("index.go", []byte(`package main

func equal(a []byte, b []byte) bool {
	i := 0
	for i < len(a) {
		if a[i] != b[i] {
			return false
		}
		i = i + 1
	}
	return true
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected index comparisons: %v", diags)
	}
}

func TestFileAllowsDirectSliceLiteralIndexExpressions(t *testing.T) {
	file, err := parse.FileSource("slice_literal_index.go", []byte(`package main

func appMain() int {
	if []int{41, 42}[1] != 42 {
		return 1
	}
	if ([]byte{65, 66})[0] != byte('A') {
		return 2
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected direct slice literal indexes: %v", diags)
	}
}

func TestFileAllowsPointerSliceCompositeLiterals(t *testing.T) {
	file, err := parse.FileSource("pointer_slice_literal.go", []byte(`package main

type box struct { value int }

func appMain() int {
	items := []*box{&box{value: 40}, &box{value: 2}}
	_ = items
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected pointer slice composite literal: %v", diags)
	}
}

func TestFileAllowsDirectStructLiteralSelectors(t *testing.T) {
	file, err := parse.FileSource("struct_literal_selector.go", []byte(`package main

type inner struct { value int }
type box struct {
	value int
	inner inner
}

func want(x int) int { return x }

func appMain() int {
	if box{value: 41, inner: inner{value: 1}}.inner.value != 1 {
		return 1
	}
	if (box{value: 42}).value != 42 {
		return 2
	}
	if (&box{value: 43}).value != 43 {
		return 3
	}
	return want(box{value: 40}.value) - 40
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected direct struct literal selectors: %v", diags)
	}
}

func TestFileAllowsDirectEmbeddedStructLiteralSelectors(t *testing.T) {
	file, err := parse.FileSource("embedded_literal_selector.go", []byte(`package main

type Inner struct { X int }
type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func appMain() int {
	if Outer{Inner{1}}.X != 1 {
		return 1
	}
	if Nested{Mid{Inner{2}}}.X != 2 {
		return 2
	}
	if (PointerOuter{&Inner{3}}).X != 3 {
		return 3
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected direct embedded struct literal selectors: %v", diags)
	}
}

func TestFileRejectsCompositeLiteralSubexpressionTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("literal_subexpressions.go", []byte(`package main

type box struct { value int }

func (b box) Value() int { return b.value }
func (b *box) PtrValue() int { return b.value }

func appMain() int {
	if box{value: "bad"}.value == 1 {
		return 1
	}
	_ = box{missing: 1}.value
	_ = []int{1, "bad"}[0]
	_ = box{value: "bad"}.Value()
	_ = []*box{&box{value: "bad"}}[0].PtrValue()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted composite literal subexpression type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"literal_subexpressions.go:9:16: composite literal field type mismatch: box.value has int, got string",
		"literal_subexpressions.go:12:10: unknown field: box.missing",
		"literal_subexpressions.go:13:15: composite literal element type mismatch: want int, got string",
		"literal_subexpressions.go:14:17: composite literal field type mismatch: box.value has int, got string",
		"literal_subexpressions.go:15:25: composite literal field type mismatch: box.value has int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsIndexAssignmentAtBlockStart(t *testing.T) {
	file, err := parse.FileSource("index_assign.go", []byte(`package main

func move(values []int, i int, j int) {
	if i < j {
		values[j+1] = values[j]
	}
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected index assignment: %v", diags)
	}
}

func TestFileRejectsTypeAssertionsAndSwitches(t *testing.T) {
	file, err := parse.FileSource("assertions.go", []byte(`package main

func appMain() int {
var x interface{} = next()
_ = x.(int)
switch x.(type) {
}
return 0
}

func next() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted type assertion forms")
	}
	msg := err.Error()
	for _, want := range []string{
		"assertions.go:5:7: type assertions and type switches are not supported",
		"assertions.go:6:10: type assertions and type switches are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsStaticInterfaceAssertions(t *testing.T) {
	file, err := parse.FileSource("static_assertions.go", []byte(`package main

func appMain() int {
	var x interface{} = 7
	var text any = "ok"
	var flag interface{} = true
	var empty interface{}
	var nilAny any = nil
	var mismatch interface{} = 7
	value := x.(int)
	gotText, okText := text.(string)
	gotFlag, okFlag := flag.(bool)
	nilValue, okNilValue := empty.(int)
	nilText, okNilText := nilAny.(string)
	missingText, okMissingText := x.(string)
	missingInt, okMissingInt := text.(int)
	missingFlag, okMissingFlag := flag.(int)
	_ = mismatch.(string)
	missingValue := mismatch.(string)
	_ = missingValue
	missingValue = mismatch.(string)
	var missingVar string = mismatch.(string)
	_ = missingVar
	var missingInferred = mismatch.(bool)
	_ = missingInferred
	printString(mismatch.(string))
	if mismatch.(bool) {
		return 2
	}
	for mismatch.(bool) {
		return 3
	}
	switch mismatch.(string) {
	case "no":
		return 4
	}
	printTwo(mismatch.(string), "x")
	defer printTwo(mismatch.(string), "x")
	if okText && okFlag && gotText == "ok" && gotFlag && !okNilValue && nilValue == 0 && !okNilText && nilText == "" && !okMissingText && missingText == "" && !okMissingInt && missingInt == 0 && !okMissingFlag && missingFlag == 0 {
		return value - 7
	}
	return 1
}

func returnMismatch() string {
	var x interface{} = 7
	return x.(string)
}

func returnPairMismatch() (string, string) {
	var x interface{} = 7
	return x.(string), "x"
}

func deferMismatch() {
	var x interface{} = 7
	defer printString(x.(string))
}

func printString(value string) {}
func printTwo(a string, b string) {}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static interface assertions: %v", diags)
	}
}

func TestFileAllowsStaticInterfaceStructAssertions(t *testing.T) {
	file, err := parse.FileSource("static_struct_assertions.go", []byte(`package main

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
	gotAgain, ok := boxed.(Box)
	gotPtr := boxedPtr.(*Box)
	gotPtrAgain, okPtr := boxedPtr.(*Box)
	missing, okMissing := boxed.(Other)
	missingPtr, okMissingPtr := boxedPtr.(*Other)
	if ok && okPtr && !okMissing && !okMissingPtr && got.Value == 41 && gotAgain.Value == 41 && gotPtr.Value == 42 && gotPtrAgain.Value == 42 && missing.Value == 0 && missingPtr == nil {
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static interface struct assertions: %v", diags)
	}
}

func TestGraphAllowsImportedStaticInterfaceStructAssertionsAndSwitches(t *testing.T) {
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
	gotAgain, ok := boxed.(dep.Box)
	gotPtr := boxedPtr.(*dep.Box)
	gotPtrAgain, okPtr := boxedPtr.(*dep.Box)
	missing, okMissing := boxed.(dep.Other)
	missingPtr, okMissingPtr := boxedPtr.(*dep.Other)
	if !ok || !okPtr || okMissing || okMissingPtr || got.Value != 41 || gotAgain.Value != 41 || gotPtr.Value != 42 || gotPtrAgain.Value != 42 || missing.Value != 0 || missingPtr != nil {
		return 1
	}
	switch boxed.(type) {
	case dep.Other:
		return 2
	case dep.Box:
		fromSwitch := boxed.(dep.Box)
		if fromSwitch.Value != 41 {
			return 3
		}
	default:
		return 4
	}
	switch value := boxed.(type) {
	case dep.Box:
		if value.Value == 41 {
			return 0
		}
	default:
		return 5
	}
	switch boxedPtr.(type) {
	case *dep.Other:
		return 6
	case *dep.Box:
		fromPtrSwitch := boxedPtr.(*dep.Box)
		if fromPtrSwitch.Value != 42 {
			return 7
		}
	default:
		return 8
	}
	switch ptrValue := boxedPtr.(type) {
	case *dep.Box:
		if ptrValue.Value == 42 {
			return 0
		}
	default:
		return 9
	}
	return 6
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
	if err := Graph(&load.Graph{Packages: []load.Package{mainPkg, depPkg}}); err != nil {
		t.Fatalf("Graph rejected imported static interface struct assertions and switches: %v", err)
	}
}

func TestFileAllowsStaticInterfaceTypeSwitches(t *testing.T) {
	file, err := parse.FileSource("static_type_switches.go", []byte(`package main

type Box struct {
	Value int
}

type Other struct {
	Value int
}

func appMain() int {
	var x interface{} = 7
	var text any = "ok"
	var flag any = true
	var empty interface{}
	var nilAny any = nil
	var boxed any = Box{Value: 41}
	var boxedPtr any = &Box{Value: 42}
	switch x.(type) {
	case string:
		return 1
	case int:
	}
	switch text.(type) {
	case bool:
		return 2
	case string:
	}
	switch n := x.(type) {
	case int:
		if n != 7 {
			return 3
		}
	}
	switch unused := flag.(type) {
	case int:
		return 4
	default:
		_ = unused
	}
	switch never := x.(type) {
	case string:
		_ = never
		return 5
	}
	switch blank := empty.(type) {
	case nil:
		_ = blank
	case int:
		return 16
	default:
		return 17
	}
	switch fallback := nilAny.(type) {
	case int:
		return 18
	default:
		_ = fallback
	}
	switch s := text.(type) {
	case string:
		if s == "ok" {
			return 0
		}
	default:
		return 6
	}
	switch boxed.(type) {
	case Other:
		return 7
	case Box:
		got := boxed.(Box)
		if got.Value != 41 {
			return 8
		}
	default:
		return 9
	}
	switch got := boxed.(type) {
	case Box:
		if got.Value == 41 {
			return 0
		}
	default:
		return 10
	}
	switch never := boxed.(type) {
	case Other:
		_ = never
		return 11
	}
	switch boxedPtr.(type) {
	case *Other:
		return 12
	case *Box:
		got := boxedPtr.(*Box)
		if got.Value != 42 {
			return 13
		}
	default:
		return 14
	}
	switch got := boxedPtr.(type) {
	case *Box:
		if got.Value == 42 {
			return 0
		}
	default:
		return 15
	}
	return 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static interface type switches: %v", diags)
	}
}

func TestFileRejectsStaticNilTypeSwitchBindingUse(t *testing.T) {
	file, err := parse.FileSource("static_nil_type_switch_binding_bad.go", []byte(`package main

func appMain() int {
	var empty interface{}
	switch v := empty.(type) {
	case nil:
		if v == nil {
			return 1
		}
	default:
		_ = v
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted value-carrying static nil type switch binding")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "interfaces are not supported") && !strings.Contains(msg, "type assertions and type switches are not supported") {
		t.Fatalf("missing interface/type switch diagnostic in:\n%s", msg)
	}
}

func TestFileRejectsUnsupportedStaticInterfaceAssertions(t *testing.T) {
	file, err := parse.FileSource("static_assertions_bad.go", []byte(`package main

func appMain() int {
	var mismatch interface{} = 7
	printTwo("x", mismatch.(string))
	var dynamic any = next()
	_ = dynamic.(int)
	switch dynamic.(type) {
case int:
		return 2
	}
	var used interface{} = 1
	_ = used
	return 0
}

func next() int { return 1 }
func printTwo(a string, b string) {}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported static interface assertions")
	}
	msg := diags.Error()
	for _, want := range []string{
		"static_assertions_bad.go:4:15: interfaces are not supported",
		"static_assertions_bad.go:5:25: type assertions and type switches are not supported",
		"static_assertions_bad.go:6:14: interfaces are not supported",
		"static_assertions_bad.go:7:14: type assertions and type switches are not supported",
		"static_assertions_bad.go:8:17: type assertions and type switches are not supported",
		"static_assertions_bad.go:12:11: interfaces are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsStaticInterfaceAssertionCallArgumentTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("static_assertion_call_type_bad.go", []byte(`package main

func appMain() int {
	var mismatch interface{} = 7
	wantInt(mismatch.(string))
	defer wantInt(mismatch.(string))
	return 0
}

func wantInt(value int) {}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted static interface assertion call argument type mismatch")
	}
	msg := diags.Error()
	for _, want := range []string{
		"static_assertion_call_type_bad.go:5:10: argument type mismatch in call to wantInt: want int, got string",
		"static_assertion_call_type_bad.go:6:16: argument type mismatch in call to wantInt: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "type assertions and type switches are not supported") {
		t.Fatalf("static assertion call argument fell back to unsupported assertion diagnostic:\n%s", msg)
	}
}

func TestFileRejectsStaticInterfaceAssertionIfConditionTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("static_assertion_if_type_bad.go", []byte(`package main

func appMain() int {
	var mismatch interface{} = 7
	if mismatch.(int) {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted static interface assertion if condition type mismatch")
	}
	msg := diags.Error()
	want := "static_assertion_if_type_bad.go:5:5: condition must be bool, got int"
	if !strings.Contains(msg, want) {
		t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
	}
	if strings.Contains(msg, "type assertions and type switches are not supported") {
		t.Fatalf("static assertion if condition fell back to unsupported assertion diagnostic:\n%s", msg)
	}
}

func TestFileRejectsStaticInterfaceAssertionForConditionTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("static_assertion_for_type_bad.go", []byte(`package main

func appMain() int {
	var mismatch interface{} = 7
	for mismatch.(int) {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted static interface assertion for condition type mismatch")
	}
	msg := diags.Error()
	want := "static_assertion_for_type_bad.go:5:6: condition must be bool, got int"
	if !strings.Contains(msg, want) {
		t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
	}
	if strings.Contains(msg, "type assertions and type switches are not supported") {
		t.Fatalf("static assertion for condition fell back to unsupported assertion diagnostic:\n%s", msg)
	}
}

func TestFileRejectsStaticInterfaceAssertionSwitchCaseTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("static_assertion_switch_case_type_bad.go", []byte(`package main

func appMain() int {
	var mismatch interface{} = 7
	switch mismatch.(string) {
	case 1:
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted static interface assertion switch case type mismatch")
	}
	msg := diags.Error()
	want := "static_assertion_switch_case_type_bad.go:6:7: switch case type mismatch: switch has string, case has int"
	if !strings.Contains(msg, want) {
		t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
	}
	if strings.Contains(msg, "type assertions and type switches are not supported") {
		t.Fatalf("static assertion switch tag fell back to unsupported assertion diagnostic:\n%s", msg)
	}
}

func TestFileRejectsStaticInterfaceDefaultTypeSwitchBindingUse(t *testing.T) {
	file, err := parse.FileSource("static_type_switch_default_binding_bad.go", []byte(`package main

func appMain() int {
	var x interface{} = 7
	switch v := x.(type) {
	case string:
		return 1
	default:
		if v == nil {
			return 2
		}
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) == 0 {
		t.Fatalf("File accepted static interface default binding as a real interface value")
	}
}

func TestFileRejectsInterfaceForms(t *testing.T) {
	file, err := parse.FileSource("interfaces.go", []byte(`package main

type Empty interface{}
type Reader interface { Read([]byte) int }
type Embedded interface { Reader }
func use(value interface{}) interface{} { return value }
func appMain() int {
	var value interface{} = 1
	_ = value.(int)
	switch value.(type) {
	case int:
		return 0
	}
	return len([]interface{}{value})
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported interface forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"interfaces.go:6:16: interfaces are not supported",
		"interfaces.go:6:29: interfaces are not supported",
		"interfaces.go:8:12: interfaces are not supported",
		"interfaces.go:14:15: interfaces are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	for _, unexpected := range []string{
		"assignment count mismatch",
		"undefined identifier: any",
	} {
		if strings.Contains(msg, unexpected) {
			t.Fatalf("unsupported interface forms produced semantic cascade %q:\n%s", unexpected, msg)
		}
	}
}

func TestFileAllowsUnusedInterfaceParameters(t *testing.T) {
	file, err := parse.FileSource("unused_interface_params.go", []byte(`package main

func use(value interface{}) {}
func useAny(value any, keep int) int { return keep }
func grouped(left, right interface{}, keep int) int { return keep }

func appMain() int {
	use(1)
	useAny("x", 2)
	return grouped(1, 2, 3) - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameters: %v", diags)
	}
}

func TestFileRejectsUsedInterfaceParameters(t *testing.T) {
	file, err := parse.FileSource("used_interface_params.go", []byte(`package main

func use(value interface{}) int { return value }

func appMain() int {
	return use(1)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted used interface parameter")
	}
	if !strings.Contains(diags.Error(), "interfaces are not supported") {
		t.Fatalf("missing interface diagnostic in:\n%s", diags.Error())
	}
}

func TestFileAllowsUnusedInterfaceParameterCallStatementSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_side_effects.go", []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}) {}

func appMain() int {
	use(bump())
	use(wrap(bump() + 1))
	use(false && flag())
	use(true || flag())
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter call-statement side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterDeferSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_defer_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter defer side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterReturnSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_return_side_effects.go", []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	return use(wrap(bump() + 1), 0)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter return side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterAssignmentSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_assignment_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter assignment side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterVarInitializerSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_var_initializer_side_effects.go", []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func flag() bool { return true }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	var first = use(wrap(bump() + 1), 3)
	var second int = use(false && flag(), first)
	return use(true || flag(), second)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter var initializer side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterIfConditionSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_if_condition_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter if-condition side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterForConditionSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_for_condition_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter for-condition side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterEmptyClassicForConditionSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_classic_for_condition_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter empty classic-for-condition side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterClassicForConditionSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_classic_for_condition_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter classic-for-condition side effects: %v", diags)
	}
}

func TestFileAllowsUnusedInterfaceParameterSwitchTagSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_switch_tag_side_effects.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected unused interface parameter switch-tag side effects: %v", diags)
	}
}

func TestFileRejectsUnusedInterfaceParameterNestedExpressionSideEffects(t *testing.T) {
	file, err := parse.FileSource("unused_interface_param_nested_expression_side_effects.go", []byte(`package main

func bump() int { return 1 }
func use(value interface{}, keep int) int { return keep }

func appMain() int {
	value := use(bump(), 0) + 1
	return value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unused interface parameter nested expression side effects")
	}
	if !strings.Contains(diags.Error(), "interfaces are not supported") {
		t.Fatalf("missing interface diagnostic in:\n%s", diags.Error())
	}
}

func TestFileAllowsDiscardedInterfaceReturns(t *testing.T) {
	file, err := parse.FileSource("discarded_interface_returns.go", []byte(`package main

func value() interface{} { return 1 }
func valueAny() any { return "x" }

func appMain() int {
	_ = value()
	valueAny()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface returns: %v", diags)
	}
}

func TestFileRejectsUsedInterfaceReturns(t *testing.T) {
	file, err := parse.FileSource("used_interface_returns.go", []byte(`package main

func value() interface{} { return 1 }

func appMain() int {
	v := value()
	_ = v
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted used interface return")
	}
	if !strings.Contains(diags.Error(), "interfaces are not supported") {
		t.Fatalf("missing interface diagnostic in:\n%s", diags.Error())
	}
}

func TestFileAllowsDiscardedInterfaceReturnDirectCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("interface_return_side_effects.go", []byte(`package main

func bump() int { return 1 }
func value() interface{} { return bump() }

func appMain() int {
	_ = value()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface return direct-call side effects: %v", diags)
	}
}

func TestFileAllowsDiscardedInterfaceReturnNestedDirectCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("interface_return_nested_side_effects.go", []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func value() interface{} { return wrap(bump()) }

func appMain() int {
	_ = value()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface return nested direct-call side effects: %v", diags)
	}
}

func TestFileAllowsDiscardedInterfaceReturnComputedDirectCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("interface_return_computed_side_effects.go", []byte(`package main

func bump() int { return 1 }
func wrap(value int) int { return value }
func value() interface{} { return wrap(bump() + 1) }

func appMain() int {
	_ = value()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface return computed direct-call side effects: %v", diags)
	}
}

func TestFileAllowsDiscardedInterfaceReturnShortCircuitCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("interface_return_short_circuit_side_effects.go", []byte(`package main

func flag() bool { return true }
func wrap(value bool) bool { return value }
func valueAnd() interface{} { return wrap(false && flag()) }
func valueOr() interface{} { return wrap(true || flag()) }

func appMain() int {
	_ = valueAnd()
	valueOr()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface return short-circuit call side effects: %v", diags)
	}
}

func TestFileAllowsDiscardedInterfaceReturnBareComputedCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("interface_return_bare_computed_side_effects.go", []byte(`package main

func bump() int { return 1 }
func flag() bool { return true }
func valueSum() interface{} { return bump() + 1 }
func valueAnd() interface{} { return false && flag() }
func valueOr() interface{} { return true || flag() }

func appMain() int {
	_ = valueSum()
	valueAnd()
	_ = valueOr()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded interface return bare computed call side effects: %v", diags)
	}
}

func TestGraphAllowsCrossFileDiscardedInterfaceReturns(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "value.go",
						Source: []byte(`package main

func value() interface{} { return 1 }
`),
					},
					{
						Path: "main.go",
						Source: []byte(`package main

func appMain() int {
	_ = value()
	return 0
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected cross-file discarded interface return: %v", err)
	}
}

func TestGraphRejectsCrossFileUsedInterfaceReturns(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "value.go",
						Source: []byte(`package main

func value() interface{} { return 1 }
`),
					},
					{
						Path: "main.go",
						Source: []byte(`package main

func appMain() int {
	v := value()
	_ = v
	return 0
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted cross-file used interface return")
	}
	if !strings.Contains(err.Error(), "value.go:3:14: interfaces are not supported") {
		t.Fatalf("missing interface return diagnostic in:\n%s", err.Error())
	}
}

func TestFileAllowsInertNamedInterfaceTypes(t *testing.T) {
	file, err := parse.FileSource("interface_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named interface types: %v", diags)
	}
}

func TestFileAllowsInertInterfaceContainingTypes(t *testing.T) {
	file, err := parse.FileSource("interface_containing_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert interface-containing types: %v", diags)
	}
}

func TestFileAllowsBlankDiscardedInterfaceVars(t *testing.T) {
	file, err := parse.FileSource("blank_discarded_interface_vars.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected blank-discarded interface vars: %v", diags)
	}
}

func TestFileAllowsNilInterfaceVarComparisons(t *testing.T) {
	file, err := parse.FileSource("nil_interface_var_comparisons.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected nil interface var comparisons: %v", diags)
	}
}

func TestFileRejectsUsedInterfaceVars(t *testing.T) {
	file, err := parse.FileSource("used_interface_vars.go", []byte(`package main

func appMain() int {
	var value interface{}
	_ = value
	_ = value
	var other any
	_ = other
	_ = other
	var compared interface{}
	if compared == nil {
		_ = compared
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted used interface vars")
	}
	if !strings.Contains(diags.Error(), "interfaces are not supported") {
		t.Fatalf("missing interface diagnostic in:\n%s", diags.Error())
	}
}

func TestFileRejectsNamedInterfaceTypeValues(t *testing.T) {
	file, err := parse.FileSource("interface_values.go", []byte(`package main

type Empty interface{}

func appMain() int {
	var value Empty
	_ = value
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named interface type value")
	}
	if !strings.Contains(diags.Error(), "interface_values.go:3:12: interfaces are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsInterfaceContainingTypeValues(t *testing.T) {
	file, err := parse.FileSource("interface_containing_type_values.go", []byte(`package main

type Box struct { Value interface{} }
type AnyBox struct { Value any }

func appMain() int {
	_ = Box{}
	_ = AnyBox{}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted interface-containing type values")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "interfaces are not supported") {
		t.Fatalf("missing interface diagnostic in:\n%s", msg)
	}
}

func TestFileRejectsMixedGroupedInterfaceTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("mixed_interface_types.go", []byte(`package main

type (
	Empty interface{}
	Count int
)

func appMain() int {
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted mixed grouped interface type declaration")
	}
	if !strings.Contains(diags.Error(), "interfaces are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileAllowsFallthrough(t *testing.T) {
	file, err := parse.FileSource("switch.go", []byte(`package main

func appMain() int {
	total := 0
	switch 1 {
	case 1:
		total = total + 1
		fallthrough
	case 2:
		total = total + 2
	}
	return total - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected fallthrough: %v", diags)
	}
}

func TestFileRejectsFinalFallthrough(t *testing.T) {
	file, err := parse.FileSource("switch.go", []byte(`package main

func appMain() int {
	switch 1 {
	case 1:
		return 1
	case 2:
		fallthrough
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted final fallthrough")
	}
	if !strings.Contains(err.Error(), "switch.go:8:3: fallthrough is not inside a non-final switch case") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileAllowsExpressionlessSwitch(t *testing.T) {
	file, err := parse.FileSource("exprless_switch.go", []byte(`package main

func appMain() int {
	x := 1
	switch {
	case x == 1:
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsInvalidSwitchCaseTypes(t *testing.T) {
	file, err := parse.FileSource("switch_types.go", []byte(`package main

type Label string
type Count int

func appMain() int {
	x := 1
	switch x {
	case "bad":
		return 1
	}
	switch text := "ok"; text {
	case 1:
		return 2
	}
	switch {
	case 1:
		return 3
	}
	switch Label("ok") {
	case Count(1):
		return 4
	}
	switch {
	case Label("bad"):
		return 5
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid switch case types")
	}
	msg := diags.Error()
	for _, want := range []string{
		"switch case type mismatch: switch has int, case has string",
		"switch case type mismatch: switch has string, case has int",
		"switch case must be bool, got int",
		"switch case must be bool, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "switch case type mismatch: switch has string, case has int"); count != 2 {
		t.Fatalf("got %d string/int switch case diagnostics, want 2 in:\n%s", count, msg)
	}
}

func TestFileAllowsSwitchCaseTypes(t *testing.T) {
	file, err := parse.FileSource("switch_types_ok.go", []byte(`package main

type Flag bool

func ready() bool { return true }

func appMain() int {
	values := []int{1, 2}
	switch x := len(values); x {
	case 1, 2:
		values[0] = x
	}
	switch "ok" {
	case "bad", "ok":
		values[0] = values[0] + 1
	}
	switch ready() {
	case true:
		values[0] = values[0] + 1
	}
	switch {
	case ready(), len(values) == 2:
		values[0] = values[0] + 1
	case Flag(false):
		values[0] = values[0] + 1
	default:
		values[0] = values[0] + 1
	}
	return values[0] - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid switch case types: %v", diags)
	}
}

func TestFileAllowsSimpleRangeLoops(t *testing.T) {
	file, err := parse.FileSource("range.go", []byte(`package main

type scores []int

func appMain() int {
	values := []int{1, 2, 3}
	total := 0
	for i, v := range values {
		total = total + i + v
	}
	for i := range values {
		total = total + i
	}
	for _, v := range []int{4, 5} {
		total = total + v
	}
	for _, v := range (scores{6, 7}) {
		total = total + v
	}
	for i := range "ok" {
		total = total + i
	}
	return total - 12
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected simple range loops: %v", diags)
	}
}

func TestFileRejectsUnsupportedRangeOperands(t *testing.T) {
	file, err := parse.FileSource("range_bad.go", []byte(`package main

func appMain() int {
	x := 1
	for range x {
	}
	for range 2 {
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported range operands")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "cannot range over: int"); count != 2 {
		t.Fatalf("got %d range diagnostics, want 2:\n%s", count, msg)
	}
}

func TestFileAllowsKeyedSliceRangeOperands(t *testing.T) {
	file, err := parse.FileSource("range_keyed_slice.go", []byte(`package main

func appMain() int {
	total := 0
	for i, value := range []int{0: 1, 2: 3} {
		total = total + i + value
	}
	return total - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected keyed slice range operand: %v", diags)
	}
}

func TestFileAllowsNonASCIIStringLiteralRanges(t *testing.T) {
	file, err := parse.FileSource("range_unicode.go", []byte("package main\n\nfunc appMain() int {\n\tfor range \"é\" {\n\t}\n\tfor range `å` {\n\t}\n\treturn 0\n}\n"))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected non-ASCII string literal ranges: %v", diags)
	}
}

func TestFileAllowsDynamicStringRanges(t *testing.T) {
	file, err := parse.FileSource("range_dynamic_string.go", []byte(`package main

const label = "ok"

func value() string { return "ok" }

func appMain() int {
	text := "ok"
	for range text {
	}
	for range label {
	}
	for range value() {
	}
	for _, c := range text {
		var r int32 = c
		_ = r
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected dynamic string ranges: %v", diags)
	}
}

func TestFileRejectsStringRangeAssignmentTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("range_string_assign_bad.go", []byte(`package main

func value() string { return "é" }

func appMain() int {
	var b byte
	for _, b = range "é" {
	}
	var i int
	for _, i = range value() {
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted string range assignment type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: b has byte, got int32",
		"assignment type mismatch: i has int, got int32",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsRangeValueTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("range_value_types.go", []byte(`package main

type box struct {
	text string
	count int
}

func wantInt(x int) int { return x }

func appMain() int {
	words := []string{"bad"}
	for i, word := range words {
		var x int
		x = word
		_ = wantInt(word)
		_ = i + word
	}
	boxes := []box{{text: "bad", count: 1}}
	for _, b := range boxes {
		var n int
		n = b.text
		_ = wantInt(b.text)
		_ = b.missing
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted range value type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"invalid operands for +: int and string",
		"assignment type mismatch: n has int, got string",
		"unknown field: box.missing",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsRangeValueTypeInference(t *testing.T) {
	file, err := parse.FileSource("range_value_types_ok.go", []byte(`package main

type box struct {
	count int
}

func wantInt(x int) int { return x }

func appMain() int {
	total := 0
	values := []int{1, 2}
	for i, value := range values {
		total = total + wantInt(i) + wantInt(value)
	}
	boxes := []box{{count: 3}}
	for _, b := range boxes {
		total = total + wantInt(b.count)
	}
	return total - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected range value type inference: %v", diags)
	}
}

func TestFileRejectsRangeAssignmentTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("range_assignment_types.go", []byte(`package main

type box struct {
	text string
}

func appMain() int {
	words := []string{"bad"}
	var i string
	var word int
	for i, word = range words {
	}
	var only string
	for only = range words {
	}
	holder := box{}
	counts := []int{0}
	for holder.text, counts[0] = range words {
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted range assignment type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: i has string, got int",
		"assignment type mismatch: word has int, got string",
		"assignment type mismatch: only has string, got int",
		"assignment type mismatch: holder.text has string, got int",
		"assignment type mismatch: counts[0] has int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsRangeAssignmentTypeChecks(t *testing.T) {
	file, err := parse.FileSource("range_assignment_types_ok.go", []byte(`package main

type box struct {
	count int
}

func appMain() int {
	values := []int{1, 2}
	var i int
	var value int
	total := 0
	for i, value = range values {
		total = total + i + value
	}
	holder := box{}
	slots := []int{0}
	for holder.count, slots[0] = range values {
		total = total + holder.count + slots[0]
	}
	return total - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected range assignment type checks: %v", diags)
	}
}

func TestFileRejectsUnsupportedStructForms(t *testing.T) {
	file, err := parse.FileSource("struct_forms.go", []byte(`package main

type Inner struct { X int }
type Nested struct {
	Field struct { A int }
}
type Outer struct {
	Inner
	*Inner
}

type UnsupportedMapField struct {
	Field map[string]struct { A int }
}

func appMain() int {
	var one struct { A int }
	two := struct { A int }{A: 1}
	xs := []struct{ A int }{{1}}
	var unsupported UnsupportedMapField
	_ = unsupported
	return one.A + two.A + xs[0].A
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported struct forms")
	}
	msg := diags.Error()
	if strings.Contains(msg, "embedded struct fields are not supported") {
		t.Fatalf("simple embedded fields should be accepted:\n%s", msg)
	}
	if count := strings.Count(msg, "anonymous struct types are not supported"); count != 1 {
		t.Fatalf("got %d anonymous struct diagnostics, want 1:\n%s", count, msg)
	}
}

func TestFileAllowsAnonymousStructNamedSlices(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_named_slices.go", []byte(`package main

type Rows []struct{ A int }
type (
	MoreRows []struct{ A int }
)

var global Rows = Rows{{A: 1}}

func appMain() int {
	rows := Rows{{A: 2}, {3}}
	more := MoreRows{{A: 4}}
	return global[0].A + rows[0].A + rows[1].A + more[0].A - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected anonymous struct named slices: %v", diags)
	}
}

func TestFileAllowsAnonymousStructAliases(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_aliases.go", []byte(`package main

type Alias = struct{ A int }
type (
	MoreAlias = struct{ A int }
)

var global Alias = Alias{1}

func appMain() int {
	local := MoreAlias{A: 2}
	return global.A + local.A - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected anonymous struct aliases: %v", diags)
	}
}

func TestFileAllowsTopLevelAnonymousStructVariables(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_globals.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected top-level anonymous struct variables: %v", diags)
	}
}

func TestFileAllowsAnonymousStructFields(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_fields.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected anonymous struct fields: %v", diags)
	}
}

func TestFileAllowsAnonymousStructParametersAndReturns(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_signatures.go", []byte(`package main

func take(v struct{ A int }) int {
	return v.A
}

func give() struct{ A int } {
	return struct{ A int }{2}
}

func round(v struct{ A int }) struct{ A int } {
	return v
}

func named() (v struct{ A int }) {
	v = struct{ A int }{4}
	return
}

func appMain() int {
	one := take(struct{ A int }{1})
	two := give()
	three := round(struct{ A int }{3})
	four := named()
	return one + two.A + three.A + four.A - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected anonymous struct signatures: %v", diags)
	}
}

func TestFileAllowsLocalAnonymousStructVariables(t *testing.T) {
	file, err := parse.FileSource("anonymous_struct_locals.go", []byte(`package main

func appMain() int {
	var zero struct{ A int }
	one := struct{ A int }{A: 1}
	var two = struct{ A int }{2}
	var three struct{ A int } = struct{ A int }{3}
	xs := []struct{ A int }{{A: 3}}
	return zero.A + one.A + two.A + three.A + xs[0].A - 9
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected local anonymous struct variables: %v", diags)
	}
}

func TestFileAllowsEmbeddedStructFieldSelectors(t *testing.T) {
	file, err := parse.FileSource("embedded_selectors.go", []byte(`package main

type Inner struct { X int }
type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func appMain() int {
	outer := Outer{Inner{X: 2}}
	nested := Nested{Mid{Inner{X: 3}}}
	pointer := PointerOuter{Inner: &Inner{X: 4}}
	return outer.X + nested.X + pointer.X - 9
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected embedded struct field selectors: %v", diags)
	}
}

func TestFileAllowsPromotedEmbeddedMethodCalls(t *testing.T) {
	file, err := parse.FileSource("promoted_methods.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected promoted embedded method calls: %v", diags)
	}
}

func TestFileAllowsPromotedCompositeLiteralMethodCalls(t *testing.T) {
	file, err := parse.FileSource("promoted_composite_methods.go", []byte(`package main

type Inner struct { X int }
func (in Inner) Value() int { return in.X }
func (in *Inner) Add(v int) int { in.X = in.X + v; return in.X }

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func appMain() int {
	return Outer{Inner: Inner{X: 2}}.Value() + PointerOuter{Inner: &Inner{X: 4}}.Add(6) + (&Outer{Inner: Inner{X: 7}}).Add(8) - 27
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected promoted composite literal method calls: %v", diags)
	}
}

func TestFileRejectsUnaddressedPromotedCompositeLiteralPointerMethodCalls(t *testing.T) {
	file, err := parse.FileSource("promoted_composite_pointer_methods.go", []byte(`package main

type Inner struct { X int }
func (in *Inner) Add(v int) int { return in.X + v }

type Outer struct { Inner }

func appMain() int {
	return Outer{Inner: Inner{X: 2}}.Add(5)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unaddressed promoted composite literal pointer method call")
	}
	if !strings.Contains(diags.Error(), "cannot call pointer method Add on Outer") {
		t.Fatalf("missing promoted pointer receiver diagnostic in:\n%s", diags.Error())
	}
}

func TestFileRejectsPromotedEmbeddedMethodCallMismatches(t *testing.T) {
	file, err := parse.FileSource("promoted_method_mismatch.go", []byte(`package main

type Inner struct { X int }
func (in Inner) Add(v int, text string) int { return in.X + v + len(text) }

type Outer struct { Inner }

func appMain() int {
	outer := Outer{Inner: Inner{X: 2}}
	_ = outer.Add(1)
	_ = outer.Add("bad", 1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted promoted embedded method call mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"argument count mismatch in call to outer.Add",
		"argument type mismatch in call to outer.Add: want int, got string",
		"argument type mismatch in call to outer.Add: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsBreakContinueOutsideControl(t *testing.T) {
	file, err := parse.FileSource("control.go", []byte(`package main

func appMain() int {
	break
	if true {
		continue
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid break/continue")
	}
	msg := diags.Error()
	for _, want := range []string{
		"control.go:4:2: break is not inside a for or switch",
		"control.go:6:3: continue is not inside a for",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsBreakContinueInsideControl(t *testing.T) {
	file, err := parse.FileSource("control_ok.go", []byte(`package main

func appMain() int {
	for i := 0; i < 3; i++ {
		if i == 1 {
			continue
		}
		switch i {
		case 2:
			break
		}
	}
	for {
		switch 1 {
		case 1:
			continue
		}
		break
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid break/continue: %v", diags)
	}
}

func TestFileRejectsNonBoolConditions(t *testing.T) {
	file, err := parse.FileSource("conditions.go", []byte(`package main

type Label string

func appMain() int {
	if 1 {
		return 1
	}
	for "bad" {
		break
	}
	for i := 0; "bad"; i++ {
		break
	}
	if x := 1; x {
		return x
	}
	if Label("bad") {
		return 2
	}
	for Label("bad") {
		break
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted non-bool conditions")
	}
	msg := diags.Error()
	for _, want := range []string{
		"conditions.go:6:5: condition must be bool, got int",
		"conditions.go:9:6: condition must be bool, got string",
		"conditions.go:12:14: condition must be bool, got string",
		"conditions.go:15:13: condition must be bool, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "condition must be bool, got string"); count != 4 {
		t.Fatalf("got %d string condition diagnostics, want 4 in:\n%s", count, msg)
	}
}

func TestFileAllowsBoolConditions(t *testing.T) {
	file, err := parse.FileSource("conditions_ok.go", []byte(`package main

type Flag bool

func ready() bool { return true }

func appMain() int {
	values := []int{1}
	if ready() && len(values) > 0 {
		values[0] = values[0] + 1
	}
	if Flag(ready()) {
		values[0] = values[0] + 1
	}
	if x := len(values); x == 1 {
		values[0] = values[0] + x
	}
	for i := 0; i < len(values); i++ {
		values[0] = values[0] + i
	}
	for ready() {
		break
	}
	for Flag(true) {
		break
	}
	return values[0] - 4
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected bool conditions: %v", diags)
	}
}

func TestFileRejectsUnparenthesizedNamedCompositeLiteralsInControlHeaders(t *testing.T) {
	file, err := parse.FileSource("control_composite_literal_headers.go", []byte(`package main

type Box struct { Value int }

func appMain() int {
	if Box{Value: 1} == Box{Value: 1} {
		return 1
	}
	for Box{Value: 1} == Box{Value: 1} {
		break
	}
	if box := Box{Value: 1}; box.Value == 1 {
		return 2
	}
	switch Box{Value: 1} {
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unparenthesized named composite literals in control headers")
	}
	msg := diags.Error()
	want := "composite literal in control header must be parenthesized"
	if count := strings.Count(msg, want); count != 6 {
		t.Fatalf("got %d control-header composite literal diagnostics, want 6 in:\n%s", count, msg)
	}
}

func TestFileAllowsLabeledBreakContinue(t *testing.T) {
	file, err := parse.FileSource("labeled_control.go", []byte(`package main

func appMain() int {
outer:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if j == 1 {
				continue outer
			}
			switch i {
			case 2:
				break outer
			}
		}
	}
	goto done
done:
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected labeled break/continue: %v", diags)
	}
}

func TestFileRejectsInvalidLabeledBreakContinue(t *testing.T) {
	file, err := parse.FileSource("labeled_control_bad.go", []byte(`package main

func appMain() int {
plain:
	print("x")
	for i := 0; i < 3; i++ {
		break plain
	}
outer:
	switch 1 {
	case 1:
		continue outer
	}
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid labeled break/continue")
	}
	msg := diags.Error()
	for _, want := range []string{
		"labeled_control_bad.go:7:3: labeled break target is not an enclosing for or switch",
		"labeled_control_bad.go:12:3: labeled continue target is not an enclosing for",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPackageInitializerCalls(t *testing.T) {
	file, err := parse.FileSource("init_calls.go", []byte(`package main

func value() int { return 1 }
type box struct { value int }

var a = value()
var b = box{value: value()}

func appMain() int {
	return a + b.value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected package initializer calls: %v", diags)
	}
}

func TestFileRejectsPackageInitializerTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("package_types.go", []byte(`package main

var Count int = "bad"
const Label string = 1
var (
	A int = 1
	B int = "bad"
)

func appMain() int {
	return Count + A + B + len(Label)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted package initializer type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"initializer type mismatch: Count has int, got string",
		"initializer type mismatch: Label has string, got int",
		"initializer type mismatch: B has int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsInferredPackageInitializerOperandMismatches(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

var Bad = true + false
var AlsoBad = []int{1} < []int{2}
var _ = "a" - "b"

func appMain() int {
	return 0
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted invalid inferred package initializers")
	}
	msg := err.Error()
	for _, want := range []string{
		"invalid operands for +: bool and bool",
		"invalid operands for <: []int and []int",
		"invalid operands for -: string and string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPackageInitializerTypeMatches(t *testing.T) {
	file, err := parse.FileSource("package_types_ok.go", []byte(`package main

type box struct { value int }

var Count int = 1
const Label string = "ok"
var Values []int = []int{1, 2}
var Box box = box{value: 2}

func appMain() int {
	return Count + len(Label) + len(Values) + Box.value - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected package initializer type matches: %v", diags)
	}
}

func TestFileRejectsPackageStructSelectorTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("package_struct_selectors.go", []byte(`package main

type box struct {
	text string
	count int
}

var (
	Global box = box{text: "bad", count: 1}
	Ptr *box = &box{text: "ptr", count: 2}
)

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = Global.text
	_ = wantInt(Ptr.text)
	_ = Global.missing
	return Global.text
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted package struct selector type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: box.missing",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPackageStructSelectorTypeInference(t *testing.T) {
	file, err := parse.FileSource("package_struct_selectors_ok.go", []byte(`package main

type box struct {
	text string
	count int
}

var (
	Global = box{text: "ok", count: 1}
	Ptr *box = &box{text: "ptr", count: 2}
)

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	text := Global.text
	count := Ptr.count
	return wantString(text) + wantInt(count) - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected package struct selector type inference: %v", diags)
	}
}

func TestFileRejectsPackageValueTypeMismatchesInFunctions(t *testing.T) {
	file, err := parse.FileSource("package_value_types.go", []byte(`package main

const Text = "bad"
var Number = 1

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = Text
	_ = wantInt(Text)
	return Text
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted package value type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPackageValueTypeInferenceInFunctions(t *testing.T) {
	file, err := parse.FileSource("package_value_types_ok.go", []byte(`package main

const Text = "ok"
var Number = 1

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	local := Text
	Number := "shadow"
	return wantString(local) + wantString(Number) + wantInt(number()) - 8
}

func number() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected package value type inference: %v", diags)
	}
}

func TestGraphRejectsCrossFilePackageValueTypeMismatches(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "values.go",
						Source: []byte(`package main

type box struct {
	text string
	count int
}

const Text = "bad"
var Global = box{text: "bad", count: 1}
`),
					},
					{
						Path: "main.go",
						Source: []byte(`package main

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = Text
	_ = wantInt(Global.text)
	_ = Global.missing
	return Global.text
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted cross-file package value type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: box.missing",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsCrossFilePackageValueTypeInference(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "values.go",
						Source: []byte(`package main

type box struct {
	text string
	count int
}

const Text = "ok"
var Number = 1
var Global = box{text: "ok", count: 2}
`),
					},
					{
						Path: "main.go",
						Source: []byte(`package main

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	text := Global.text
	count := Global.count
	return wantString(Text) + wantString(text) + wantInt(Number) + wantInt(count) - 7
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected cross-file package value inference: %v", err)
	}
}

func TestFileAllowsLocalConstAndLocalTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("local_decls.go", []byte(`package main

func appMain() int {
	const answer = 42
	type local int
	type box struct { value local }
	var x local = 1
	var b box = box{value: x}
	return answer + int(b.value) - 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected local type declaration: %v", diags)
	}
}

func TestFileTypeChecksLocalTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("local_type_cascade.go", []byte(`package main

func appMain() int {
	type local int
	var x local = "bad"
	return int(x)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted local type mismatch")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "local_type_cascade.go:5:16: initializer type mismatch: x has local, got string") {
		t.Fatalf("missing local type mismatch diagnostic in:\n%s", msg)
	}
	for _, unexpected := range []string{
		"undefined identifier: local",
		"cannot convert",
	} {
		if strings.Contains(msg, unexpected) {
			t.Fatalf("local type declaration produced semantic cascade %q:\n%s", unexpected, msg)
		}
	}
}

func TestFileRejectsLocalInitializerTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("local_init_types.go", []byte(`package main

func text() string { return "bad" }
func number() int { return 1 }

func appMain() int {
	var x int = "bad"
	const label string = 1
	var (
		a int = number()
		b int = text()
	)
	return x + a + b + len(label)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted local initializer type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"initializer type mismatch: x has int, got string",
		"initializer type mismatch: label has string, got int",
		"initializer type mismatch: b has int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "cannot assign to non-addressable expression") {
		t.Fatalf("local declaration produced assignment diagnostic:\n%s", msg)
	}
}

func TestFileAllowsLocalInitializerTypeMatches(t *testing.T) {
	file, err := parse.FileSource("local_init_types_ok.go", []byte(`package main

func text() string { return "ok" }
func number() int { return 1 }

func appMain() int {
	var x int = number()
	const label string = "ok"
	var (
		a int = 1
		b string = text()
	)
	return x + a + len(label) + len(b) - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected local initializer type matches: %v", diags)
	}
}

func TestFileRejectsUnsupportedLiterals(t *testing.T) {
	file, err := parse.FileSource("literals.go", []byte("package main\n\nfunc appMain() int {\n\t_ = `raw`\n\t_ = 077\n\t_ = 0o77\n\tvalue := 1i\n\t_ = value\n\t_ = value\n\t_ = 0x10\n\t_ = 0b10\n\t_ = 0.5\n\treturn 0\n}\n"))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported literals")
	}
	msg := err.Error()
	for _, want := range []string{
		"literals.go:7:11: imaginary literals are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	for _, allowed := range []string{"`raw`", "077", "0o77", "0x10", "0b10", "0.5"} {
		if strings.Contains(msg, allowed) {
			t.Fatalf("supported literal %s was diagnosed:\n%s", allowed, msg)
		}
	}
}

func TestFileAllowsDiscardedPureComplexExpressions(t *testing.T) {
	file, err := parse.FileSource("complex_discard.go", []byte(`package main

func appMain() int {
	_ = 1i
	_ = 1 + 2i
	_ = (3 - 4i)
	_ = complex(1, 2)
	_, _ = 5i, complex(-6, +7)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded pure complex expressions: %v", diags)
	}
}

func TestFileAllowsDiscardedComplexCallSideEffects(t *testing.T) {
	file, err := parse.FileSource("complex_discard_calls.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded complex call side effects: %v", diags)
	}
}

func TestFileAllowsBlankDiscardedComplexVars(t *testing.T) {
	file, err := parse.FileSource("complex_var_discard.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected blank-discarded complex vars: %v", diags)
	}
}

func TestFileRejectsUsedComplexVars(t *testing.T) {
	file, err := parse.FileSource("complex_var_used.go", []byte(`package main

func appMain() int {
	var a complex64 = 1 + 2i
	_ = a
	_ = a
	b := 3 + 4i
	_ = b
	_ = b
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted used complex var")
	}
	msg := err.Error()
	if !strings.Contains(msg, "complex numbers are not supported") || !strings.Contains(msg, "imaginary literals are not supported") {
		t.Fatalf("missing complex diagnostics in:\n%s", msg)
	}
}

func TestFileRejectsDiscardedComplexCallNonFloatOperand(t *testing.T) {
	file, err := parse.FileSource("complex_discard_bad.go", []byte(`package main

func count() int { return 1 }
func value() float64 { return 1 }

func appMain() int {
	_ = complex(count(), 2)
	_ = value() + 2i
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted discarded complex call with non-float operand")
	}
	if !strings.Contains(err.Error(), "unsupported builtin: complex") {
		t.Fatalf("missing complex diagnostic in:\n%s", err)
	}
	if !strings.Contains(err.Error(), "imaginary literals are not supported") {
		t.Fatalf("missing imaginary literal diagnostic in:\n%s", err)
	}
}

func TestFileAllowsFrontendNormalizedLiterals(t *testing.T) {
	file, err := parse.FileSource("normalized_literals.go", []byte("package main\n\nfunc appMain() int {\n\tprint(`PASS\\n`)\n\t_ = 077\n\t_ = 0o77\n\treturn 0\n}\n"))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected frontend-normalized literals: %v", diags)
	}
}

func TestFileAllowsIotaConst(t *testing.T) {
	file, err := parse.FileSource("iota.go", []byte(`package main

const (
	A = iota
	B
)

func appMain() int { return A + B }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected iota: %v", diags)
	}
}

func TestFileRejectsAnyInterfaceValues(t *testing.T) {
	file, err := parse.FileSource("any.go", []byte(`package main

type Box struct { value any }
type Alias any
func use(value any) any { return value }
func appMain() int {
	var value any
	var pointer *any
	var values []any
	_ = value
	_ = pointer
	_ = values
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted any interface values")
	}
	msg := err.Error()
	for _, want := range []string{
		"any.go:5:16: interfaces are not supported",
		"any.go:5:21: interfaces are not supported",
		"any.go:7:12: interfaces are not supported",
		"any.go:8:15: interfaces are not supported",
		"any.go:9:15: interfaces are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsInertNamedAnyTypes(t *testing.T) {
	file, err := parse.FileSource("any_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named any types: %v", diags)
	}
}

func TestFileRejectsComplexTypes(t *testing.T) {
	file, err := parse.FileSource("complex.go", []byte(`package main

type Alias complex64
type Box struct { value complex128 }

func use(value complex64) complex128 {
	var local complex128
	var values []complex64
	_ = local
	_ = values
	return value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted complex types")
	}
	msg := err.Error()
	for _, want := range []string{
		"complex.go:6:16: complex numbers are not supported",
		"complex.go:6:27: complex numbers are not supported",
		"complex.go:7:12: complex numbers are not supported",
		"complex.go:8:15: complex numbers are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	for _, unexpected := range []string{
		"undefined identifier: complex64",
		"undefined identifier: complex128",
		"initializer type mismatch",
		"return type mismatch",
	} {
		if strings.Contains(msg, unexpected) {
			t.Fatalf("unsupported complex types produced semantic cascade %q:\n%s", unexpected, msg)
		}
	}
}

func TestFileAllowsInertNamedComplexTypes(t *testing.T) {
	file, err := parse.FileSource("complex_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named complex types: %v", diags)
	}
}

func TestFileAllowsInertComplexContainingTypes(t *testing.T) {
	file, err := parse.FileSource("complex_containing_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert complex-containing types: %v", diags)
	}
}

func TestFileRejectsComplexContainingTypeValues(t *testing.T) {
	file, err := parse.FileSource("complex_containing_type_values.go", []byte(`package main

type Box struct { Value complex64 }
type Values []complex128

func appMain() int {
	_ = Box{}
	_ = Values{}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted complex-containing type values")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "complex numbers are not supported") {
		t.Fatalf("missing complex diagnostic in:\n%s", msg)
	}
}

func TestFileAllowsShadowingComplexTypeNames(t *testing.T) {
	file, err := parse.FileSource("complex_shadow.go", []byte(`package main

func appMain() int {
	complex64 := 1
	complex128 := 2
	return complex64 + complex128 - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected complex type name shadows: %v", diags)
	}
}

func TestFileRejectsNestedArrayTypes(t *testing.T) {
	file, err := parse.FileSource("arrays.go", []byte(`package main

type Pointer *[3]int
type SliceOfArray [][2]int
type Box struct { values []*[4]int }
func use(values []*[5]int) int { return 0 }
func sum(values [9]int) int { return 0 }
func appMain() int {
	var values [][6]int
	_ = values
	_ = [][8]int{{1, 2}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted nested array types")
	}
	msg := err.Error()
	for _, want := range []string{
		"arrays.go:3:15: arrays are not supported",
		"arrays.go:4:21: arrays are not supported",
		"arrays.go:5:29: arrays are not supported",
		"arrays.go:6:20: arrays are not supported",
		"arrays.go:9:15: arrays are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsValueCarryingNestedFixedArrayTypes(t *testing.T) {
	file, err := parse.FileSource("nested_array_values.go", []byte(`package main

func matrix(values [2][2]int) int { return 0 }

func appMain() int {
	var values [2][2]int
	_ = values
	type Matrix [2][2]int
	var local Matrix
	_ = local
	return matrix(values)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted value-carrying nested fixed-array types")
	}
	if !strings.Contains(diags.Error(), "arrays are not supported") {
		t.Fatalf("missing nested array diagnostic in:\n%s", diags.Error())
	}
}

func TestFileAllowsInertNamedArrayTypes(t *testing.T) {
	file, err := parse.FileSource("array_types.go", []byte(`package main

type Values [3]int
type Bytes = [2]byte
type Alias [4]Values
type Matrix [2][3]int
type (
	Counts [2]int
	Rows = [3]Counts
	Grid [2][3]int
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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named array types: %v", diags)
	}
}

func TestFileAllowsNamedArrayTypeValues(t *testing.T) {
	file, err := parse.FileSource("array_type_values.go", []byte(`package main

type Values [3]int

func appMain() int {
	var values Values
	values[0] = 1
	copy := values
	copy[0] = 9
	if values == (Values{1, 0, 0}) && copy[0] == 9 && len(values) == 3 && cap(values) == 3 {
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected named array type value: %v", diags)
	}
}

func TestFileAllowsLocalNamedArrayTypeValues(t *testing.T) {
	file, err := parse.FileSource("local_array_type_values.go", []byte(`package main

func appMain() int {
	type Values [2]int
	var values Values = Values{1, 2}
	copy := values
	copy[0] = 9
	if values == (Values{1, 2}) && copy[0] == 9 && len(values) == 2 && cap(values) == 2 {
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected local named array type value: %v", diags)
	}
}

func TestFileAllowsMixedGroupedArrayTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("mixed_array_types.go", []byte(`package main

type (
	Values [3]int
	Count int
)

func appMain() int {
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected mixed grouped array declaration: %v", diags)
	}
}

func TestFileRejectsNestedSliceTypes(t *testing.T) {
	file, err := parse.FileSource("nested_slices.go", []byte(`package main

type Matrix [][]int
type Box struct { values [][]int }
func use(values [][]int) int { return 0 }
func appMain() int {
	var values [][]int
	_ = [][]int{{1, 2}}
	return len(values)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted nested slice types")
	}
	msg := err.Error()
	for _, want := range []string{
		"nested_slices.go:3:13: nested slice types are not supported",
		"nested_slices.go:4:26: nested slice types are not supported",
		"nested_slices.go:5:17: nested slice types are not supported",
		"nested_slices.go:7:13: nested slice types are not supported",
		"nested_slices.go:8:6: nested slice types are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsUnsupportedArrayLiterals(t *testing.T) {
	file, err := parse.FileSource("array_literals.go", []byte(`package main

type Box struct { value int }

func appMain() int {
	_ = [3]int{sideEffect() + 1, 2, 3}
	_ = [2][2]int{{sideEffect() + 1, 2}, {3, 4}}
	_ = Box{value: [2]int{1, 2}}
	return 0
}

func sideEffect() int { return 1 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted fixed array literals")
	}
	msg := err.Error()
	for _, want := range []string{
		"array_literals.go:6:6: arrays are not supported",
		"array_literals.go:7:6: arrays are not supported",
		"array_literals.go:8:17: composite literal field type mismatch: Box.value has int, got [2]int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "variadic syntax is not supported") {
		t.Fatalf("inferred array literal produced variadic cascade:\n%s", msg)
	}
}

func TestFileAllowsDiscardedArrayLiteralSideEffects(t *testing.T) {
	file, err := parse.FileSource("array_discard_side_effects.go", []byte(`package main

func appMain() int {
	_ = [3]int{first(), 2, second()}
	_, _ = [...]int{1: third()}, [2][1]int{{fourth()}, {5}}
	return 0
}

func first() int { return 1 }
func second() int { return 2 }
func third() int { return 3 }
func fourth() int { return 4 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded array literal side effects: %v", diags)
	}
}

func TestFileAllowsDiscardedArrayLiterals(t *testing.T) {
	file, err := parse.FileSource("array_discard.go", []byte(`package main

func appMain() int {
	_ = [3]int{1, -2, +3}
	_ = [...]int{1: 2, 3: 4}
	_ = [2]string{"a", "b"}
	_ = [2]bool{true, false}
	_ = [2]*int{nil, nil}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded array literals: %v", diags)
	}
}

func TestFileAllowsLowerableArrayForms(t *testing.T) {
	file, err := parse.FileSource("array_ok.go", []byte(`package main

func appMain() int {
	var zero [3]int
	values := [3]int{1, 2}
	keyed := [4]int{1: 7, 3: 9}
	inferred := [...]int{5, 6}
	var explicit [2]int = [2]int{8}
	total := len(zero) + cap(values) + values[2] + keyed[1] + keyed[3] + len(inferred) + explicit[1]
	for i, v := range values {
		total = total + i + v
	}
	return total - 36
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected lowerable array forms: %v", diags)
	}
}

func TestFileAllowsArrayParameters(t *testing.T) {
	file, err := parse.FileSource("array_params.go", []byte(`package main

func sum(values [3]int) int {
	total := len(values) + cap(values) + values[0] + values[1] + values[2]
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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected array parameters: %v", diags)
	}
}

func TestFileAllowsNamedArrayParametersAndResults(t *testing.T) {
	file, err := parse.FileSource("named_array_signatures.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected named array signatures: %v", diags)
	}
}

func TestFileAllowsArrayComparisons(t *testing.T) {
	file, err := parse.FileSource("array_comparisons.go", []byte(`package main

func same(left, right [2]int) bool { return left == right }

func appMain() int {
	left := [2]int{1, 2}
	right := [...]int{1, 3}
	if same(left, [2]int{1, 2}) && left == [2]int{1, 2} && left != right && [2]int{1, 2} == [...]int{1, 2} {
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected array comparisons: %v", diags)
	}
}

func TestFileAllowsArrayComparisonCalls(t *testing.T) {
	file, err := parse.FileSource("array_comparison_calls.go", []byte(`package main

func next() int { return 1 }
func left() [2]int { return [2]int{1, 2} }
func right() [2]int { return [2]int{1, next()} }

func appMain() int {
	_ = [2]int{next(), 2} == [2]int{1, 2}
	_ = left() == right()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected array comparison with call operands: %v", diags)
	}
}

func TestFileAllowsArrayStructFields(t *testing.T) {
	file, err := parse.FileSource("array_struct_fields.go", []byte(`package main

type Box struct {
	Values [3]int
	Bytes [2]byte
	Flags [2]bool
}

func appMain() int {
	b := Box{
		Values: [3]int{1, 2},
		Bytes: [2]byte{'A'},
		Flags: [2]bool{1: true},
	}
	total := len(b.Values) + cap(b.Bytes) + b.Values[0] + int(b.Bytes[0])
	if b.Flags[1] {
		total = total + 1
	}
	if b.Values == [3]int{1, 2} && b.Bytes != [2]byte{'B'} {
		total = total + 1
	}
	return total - 72
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected array struct fields: %v", diags)
	}
}

func TestFileRejectsArrayForms(t *testing.T) {
	file, err := parse.FileSource("array_forms.go", []byte(`package main

type Box struct {
	Values [2]int
	Matrix [2][2]int
	Slices [][2]int
}

func sum(values [2]int) int { return values[0] }

func appMain() int {
	var values [3]int
	_ = values
	_ = [3]int{1, 2, 3}
	_ = [...]int{1, 2, 3}
	_ = [2][2]int{{1, 2}, {3, 4}}
	_ = [][2]int{{1, 2}}
	_ = len([2]int{1, 2})
	_ = [2]int{1, 2} == [2]int{1, 2}
	return sum([2]int{1, 2})
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported array forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"array_forms.go:5:12: arrays are not supported",
		"array_forms.go:6:11: arrays are not supported",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsDiscardedNestedArrayLiterals(t *testing.T) {
	file, err := parse.FileSource("array_nested_discard.go", []byte(`package main

func appMain() int {
	_ = [2][2]int{{1, 2}, {3, 4}}
	_ = [][2]int{{1, 2}, {3, 4}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded nested array literals: %v", diags)
	}
}

func TestFileAcceptsSimpleSubsetProgram(t *testing.T) {
	file, err := parse.FileSource("ok.go", []byte(`package main

type box struct { value int }
func appMain() int {
	var b box
	b.value = 7
	any := 3
	values := []int{1, 2}
	if values[0] == 1 {
		b.value = b.value + values[1] + any
	}
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

func TestFileAcceptsStringIndexAndSliceExpressions(t *testing.T) {
	file, err := parse.FileSource("strings.go", []byte(`package main

func appMain() int {
	s := "prefix-body"
	if s[0] == 'p' && s[0:6] == "prefix" {
		return 0
	}
	if isByte(s[0]) && s[1:3] == "re" {
		return 0
	}
	return 1
}

func isByte(c byte) bool {
	return c > 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileAllowsArrayFunctionResults(t *testing.T) {
	file, err := parse.FileSource("array_results.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected array result types: %v", diags)
	}
}

func TestFileAcceptsOrdinaryMainEntrypoint(t *testing.T) {
	file, err := parse.FileSource("main.go", []byte(`package main

func main() {
	print("PASS\n")
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileRejectsInvalidOrdinaryMainSignatures(t *testing.T) {
	file, err := parse.FileSource("main.go", []byte(`package main

func main(args []string) {}
func mainResult() {}
func mainWithResult() int { return 0 }
func main() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid ordinary main signatures")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:3:6: main function must have no parameters or results",
		"main.go:6:6: main function must have no parameters or results",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "mainResult") || strings.Contains(msg, "mainWithResult") {
		t.Fatalf("non-entrypoint helper names were diagnosed:\n%s", msg)
	}
}

func TestFileAllowsInitFunctions(t *testing.T) {
	file, err := parse.FileSource("init.go", []byte(`package main

func init() {
	print("init")
}

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected init function: %v", diags)
	}
}

func TestFileRejectsInvalidInitSignatures(t *testing.T) {
	file, err := parse.FileSource("init.go", []byte(`package main

func init(x int) {}
func init() int { return 1 }

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid init signatures")
	}
	msg := err.Error()
	for _, want := range []string{
		"init.go:3:6: init function must have no parameters or results",
		"init.go:4:6: init function must have no parameters or results",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "init functions are not supported") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileAllowsNamedResultParameters(t *testing.T) {
	file, err := parse.FileSource("results.go", []byte(`package main

func value() (out int) {
	out = 1
	return
}

func appMain() int { return value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileAllowsFullSliceExpressions(t *testing.T) {
	file, err := parse.FileSource("slice.go", []byte(`package main

func appMain() int {
	values := []int{1, 2, 3}
	x := values[1:2:3]
	return x[0]
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsFullSliceString(t *testing.T) {
	file, err := parse.FileSource("slice.go", []byte(`package main

func appMain() int {
	values := "PASS"
	x := values[1:2:3]
	return len(x)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted string full slice expression")
	}
	if !strings.Contains(err.Error(), "slice.go:5:17: full slice expressions require slice, got string") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileAllowsSupportedVariadicSyntax(t *testing.T) {
	file, err := parse.FileSource("variadic.go", []byte(`package main

func sum(values ...int) int { return len(values) }

type box struct{}

func (b box) add(base int, values ...int) int { return base + len(values) }

func appMain() int {
	values := []int{1, 2}
	dst := []int{0}
	dst = append(dst, values...)
	b := box{}
	return sum(values...) + b.add(1, values...) + len(dst) - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected supported variadic syntax: %v", diags)
	}
}

func TestFileRejectsVariadicExpansionInNonVariadicCalls(t *testing.T) {
	file, err := parse.FileSource("variadic_bad.go", []byte(`package main

func one(values []int) int { return 0 }

func appMain() int {
	values := []int{1}
	_ = one(values...)
	_ = len(values...)
	_ = int(values...)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted variadic expansion in non-variadic calls")
	}
	msg := err.Error()
	for _, want := range []string{
		"variadic_bad.go:7:16: variadic expansion in non-variadic call",
		"variadic_bad.go:8:16: variadic expansion in non-variadic call",
		"variadic_bad.go:9:16: variadic expansion in non-variadic call",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsVariadicArgumentTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("variadic_types.go", []byte(`package main

func sum(values ...int) int { return len(values) }
func prefixed(prefix int, values ...int) int { return prefix + len(values) }

func appMain() int {
	text := []string{"x"}
	values := []int{1}
	_ = sum("bad")
	_ = sum(text...)
	_ = prefixed(values...)
	_ = prefixed(1, text...)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted variadic argument type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"variadic_types.go:9:10: argument type mismatch in call to sum: want int, got string",
		"variadic_types.go:10:10: argument type mismatch in call to sum: want []int, got []string",
		"variadic_types.go:11:15: argument type mismatch in call to prefixed: want int, got []int",
		"variadic_types.go:12:18: argument type mismatch in call to prefixed: want []int, got []string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsMethods(t *testing.T) {
	file, err := parse.FileSource("method.go", []byte(`package main

type box struct { value int }

func (b box) Value() int {
	return b.value
}

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected method declaration: %v", diags)
	}
}

func TestFileAllowsBlankImports(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import _ "example.com/side"

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected blank import: %v", diags)
	}
}

func TestGraphAllowsDotImports(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dot", "example.com/app/side", "example.com/app/alias"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import (
	_ "example.com/app/side"
	. "example.com/app/dot"
	alias "example.com/app/alias"
)

func appMain() int {
	var box Box
	box.Value = Value() + Number + alias.Value()
	return box.Value
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dot",
				Name:       "dot",
				Files: []load.File{
					{
						Path: "dot.go",
						Source: []byte(`package dot

const Number = 2
type Box struct { Value int }
func Value() int { return 39 }
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/alias",
				Name:       "alias",
				Files: []load.File{
					{
						Path: "alias.go",
						Source: []byte(`package alias

func Value() int { return 1 }
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/side",
				Name:       "side",
				Files: []load.File{
					{
						Path: "side.go",
						Source: []byte(`package side

func init() {}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected dot import: %v", err)
	}
}

func TestFileAllowsGoEmbedDirectiveShape(t *testing.T) {
	file, err := parse.FileSource("embed.go", []byte(`package main

var note = "//go:embed not-a-directive"

//go:embed data.txt
var data string

func appMain() int { return len(data) }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected go:embed directive shape: %v", diags)
	}
}

func TestFileRejectsDuplicateImportNames(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	"example.com/one/fmt"
	"fmt"
	lib "example.com/lib/a"
	lib "example.com/lib/b"
)

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted duplicate import names")
	}
	msg := err.Error()
	for _, want := range []string{
		"imports.go:5:2: duplicate import name: fmt",
		"imports.go:7:6: duplicate import name: lib",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAcceptsDistinctImportAliases(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	fmt1 "example.com/one/fmt"
	fmt2 "fmt"
)

func appMain() int { return fmt1.Value() + fmt2.Value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileRejectsUnusedImports(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import (
	"fmt"
	alias "example.com/alias"
)

func appMain() int { return 0 }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unused imports")
	}
	msg := err.Error()
	for _, want := range []string{
		"imports.go:4:2: unused import: fmt",
		"imports.go:5:8: unused import: alias",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileDoesNotCountLocalShadowSelectorAsImportUse(t *testing.T) {
	file, err := parse.FileSource("imports.go", []byte(`package main

import "example.com/dep"

type localDep struct { Value int }

func appMain() int {
	dep := localDep{Value: 1}
	return dep.Value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unused import shadowed by local selector")
	}
	if !strings.Contains(err.Error(), "imports.go:3:8: unused import: dep") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsUnsupportedBuiltins(t *testing.T) {
	file, err := parse.FileSource("builtins.go", []byte(`package main

func appMain() int {
	println("bad")
	close(nil)
	delete(nil, "key")
	_ = new([2]int)
	_ = real(1)
	_ = imag(1)
	_ = complex(len("x"), 2)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported builtins")
	}
	msg := err.Error()
	for _, want := range []string{
		"builtins.go:5:2: unsupported builtin: close",
		"builtins.go:6:2: unsupported builtin: delete",
		"builtins.go:7:10: arrays are not supported",
		"builtins.go:8:6: unsupported builtin: real",
		"builtins.go:9:6: unsupported builtin: imag",
		"builtins.go:10:6: unsupported builtin: complex",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "undefined identifier:") {
		t.Fatalf("unsupported builtins produced undefined-identifier cascades:\n%s", msg)
	}
}

func TestFileAllowsReducibleComplexComponents(t *testing.T) {
	file, err := parse.FileSource("complex_components.go", []byte(`package main

func first() float64 { return 8 }
func second() float64 { return 9 }

func appMain() int {
	var a float64 = 1
	b := 2.5
	r := real(complex(a, b))
	i := imag(complex(3, 4))
	side := real(complex(first(), second()))
	literalR := real(1+2i)
	literalI := imag(3-4i)
	pureI := imag(-5i)
	reverseR := real(6i-7)
	var total float64 = r + i + side + literalR + literalI + pureI + reverseR
	_ = total
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected reducible complex components: %v", diags)
	}
}

func TestFileAllowsStaticComplexAliasComponents(t *testing.T) {
	file, err := parse.FileSource("complex_alias_components.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static complex alias components: %v", diags)
	}
}

func TestFileDoesNotReduceShadowedComplexBuiltin(t *testing.T) {
	file, err := parse.FileSource("complex_shadow.go", []byte(`package main

func complex(a int, b int) int { return a + b }

func appMain() int {
	_ = real(complex(1, 2))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted shadowed complex builtin reduction")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "complex_shadow.go:6:6: unsupported builtin: real") {
		t.Fatalf("missing unsupported real diagnostic in:\n%s", msg)
	}
	if strings.Contains(msg, "unsupported builtin: complex") {
		t.Fatalf("shadowed complex was diagnosed as builtin:\n%s", msg)
	}
}

func TestFileRejectsInvalidReducibleComplexComponentTypes(t *testing.T) {
	file, err := parse.FileSource("complex_components_bad.go", []byte(`package main

func value() int { return 1 }

func appMain() int {
	text := "x"
	n := 1
	_ = real(complex(value(), 2))
	_ = imag(complex(1, text))
	_ = real(complex(n, 2))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid reducible complex component types")
	}
	msg := diags.Error()
	for _, want := range []string{
		"complex_components_bad.go:8:19: complex arguments must be floating-point values or numeric constants, got int",
		"complex_components_bad.go:9:22: complex arguments must be floating-point values or numeric constants, got string",
		"complex_components_bad.go:10:19: complex arguments must be floating-point values or numeric constants, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "unsupported builtin: real") || strings.Contains(msg, "unsupported builtin: imag") || strings.Contains(msg, "unsupported builtin: complex") {
		t.Fatalf("reducible complex diagnostics fell back to unsupported builtin:\n%s", msg)
	}
}

func TestFileAllowsStringPanicAndRecover(t *testing.T) {
	file, err := parse.FileSource("panic_recover.go", []byte(`package main

func cleanup() {
	value := recover()
	if value == "bad" {
		print("PASS\n")
	}
}

func appMain() int {
	defer cleanup()
	panic("bad")
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected string panic/recover: %v", diags)
	}
}

func TestFileRejectsUnsupportedPanicRecoverForms(t *testing.T) {
	file, err := parse.FileSource("panic_recover_bad.go", []byte(`package main

func appMain() int {
	panic()
	panic(1)
	_ = recover("x")
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported panic/recover forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"panic_recover_bad.go:4:2: panic expects one argument",
		"panic_recover_bad.go:5:8: panic currently supports string values only",
		"panic_recover_bad.go:6:6: recover expects no arguments",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsPrintlnBuiltin(t *testing.T) {
	file, err := parse.FileSource("println.go", []byte(`package main

func appMain() int {
	println("PASS")
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected println: %v", diags)
	}
}

func TestFileAllowsNewScalarBuiltin(t *testing.T) {
	file, err := parse.FileSource("new_scalar.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected new scalar builtin: %v", diags)
	}
}

func TestFileAllowsNewPointerAndSliceBuiltins(t *testing.T) {
	file, err := parse.FileSource("new_pointer_slice.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected new pointer/slice builtin: %v", diags)
	}
}

func TestFileAllowsNewNamedStructBuiltin(t *testing.T) {
	file, err := parse.FileSource("new_struct.go", []byte(`package main

type box struct { value int }

func appMain() int {
	p := new(box)
	p.value = 1
	return p.value - 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected new named struct: %v", diags)
	}
}

func TestGraphAllowsNewImportedNamedStructBuiltin(t *testing.T) {
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

func appMain() int {
	p := new(dep.Box)
	p.Value = 1
	return p.Value - 1
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
						Path: "box.go",
						Source: []byte(`package dep

type Box struct { Value int }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected new imported named struct: %v", err)
	}
}

func TestGraphAllowsNewNamedStructAliasBuiltin(t *testing.T) {
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
			},
			{
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected new named struct aliases: %v", err)
	}
}

func TestFileAllowsSupportedBuiltins(t *testing.T) {
	file, err := parse.FileSource("builtins.go", []byte(`package main

type Label string

func appMain() int {
	values := []int{1}
	dst := []int{0}
	_ = len(values)
	_ = cap(values)
	_ = append(values, 2)
	_ = copy(dst, values)
	_ = make([]int, 1)
	print("ok")
	print(Label("ok"))
	println()
	println("ok", Label("label"))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
}

func TestFileRejectsInvalidSupportedBuiltinCalls(t *testing.T) {
	file, err := parse.FileSource("builtin_args.go", []byte(`package main

type Label string

func appMain() int {
	values := []int{1}
	dst := []byte{'a'}
	_ = len()
	_ = len(1)
	_ = len(nil)
	_ = cap()
	_ = cap(1)
	_ = cap(nil)
	_ = cap("bad")
	_ = append(1, 2)
	_ = append(nil, 1)
	_ = append(values)
	_ = append(values, "bad")
	_ = copy(1, values)
	_ = copy(dst, 1)
	_ = make(int, 1)
	_ = make([]int)
	_ = make([]int, "bad")
	_ = make([]int, Label("bad"))
	_ = make([]int, nil)
	print()
	print("ok", "bad")
	print(1)
	print(nil)
	println(1)
	println(nil)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid supported builtin calls")
	}
	msg := diags.Error()
	for _, want := range []string{
		"argument count mismatch in call to len",
		"invalid argument to len: int",
		"invalid argument to len: nil",
		"argument count mismatch in call to cap",
		"invalid argument to cap: int",
		"invalid argument to cap: nil",
		"invalid argument to cap: string",
		"first argument to append must be slice, got int",
		"first argument to append must be slice, got nil",
		"argument count mismatch in call to append",
		"append element type mismatch: want int, got string",
		"first argument to copy must be slice, got int",
		"copy source type mismatch: want []byte, got int",
		"make requires slice type, got int",
		"argument count mismatch in call to make",
		"make size must be integer, got string",
		"make size must be integer, got nil",
		"argument count mismatch in call to print",
		"argument type mismatch in call to print: want string, got int",
		"argument type mismatch in call to print: want string, got nil",
		"argument type mismatch in call to println: want string, got int",
		"argument type mismatch in call to println: want string, got nil",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "make size must be integer, got string"); count != 2 {
		t.Fatalf("got %d make string-size diagnostics, want 2 in:\n%s", count, msg)
	}
}

func TestFileAllowsValidSupportedBuiltinCalls(t *testing.T) {
	file, err := parse.FileSource("builtin_args_ok.go", []byte(`package main

type Count int
type Counts []Count
type Bytes []byte

func appMain() int {
	values := []int{1}
	more := []int{2}
	bytes := []byte{'a'}
	counts := Counts{1}
	moreCounts := []Count{2}
	namedBytes := Bytes{'a'}
	_ = len(values)
	_ = len("ok")
	_ = cap(values)
	_ = cap(counts)
	_ = cap(make([]int, 1, 2))
	_ = append(values, 2)
	_ = append(counts, 2)
	_ = append(counts, Count(3))
	_ = append(counts, moreCounts...)
	_ = append(counts, Counts{4}...)
	_ = copy(values, more)
	_ = copy(counts, moreCounts)
	_ = copy(counts, Counts{5})
	_ = copy(bytes, "b")
	_ = copy(namedBytes, "b")
	_ = make([]int, 1)
	_ = make([]int, 1, 2)
	_ = make(Counts, 1, 2)
	_ = make([]int, Count(1), Count(2))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid supported builtin calls: %v", diags)
	}
}

func TestFileRejectsNamedSliceBuiltinElementTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("named_slice_builtin_types.go", []byte(`package main

type Count int
type Counts []Count
type Byte byte

func appMain() int {
	counts := Counts{1}
	ints := []int{1}
	i := 1
	counts = append(counts, i)
	counts = append(counts, ints...)
	_ = copy(counts, ints)
	var namedBytes []Byte
	_ = copy(namedBytes, "x")
	return len(counts)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named slice builtin element type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"append element type mismatch: want Count, got int",
		"append expansion type mismatch: want []Count, got []int",
		"copy source type mismatch: want Counts, got []int",
		"copy source type mismatch: want []Byte, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsInvalidConversionCalls(t *testing.T) {
	file, err := parse.FileSource("conversions.go", []byte(`package main

type intPtr *int
type box struct { value int }

func appMain() int {
	_ = int()
	_ = byte(1, 2)
	_ = int("bad")
	_ = bool(1)
	_ = []byte(1)
	_ = string(true)
	_ = int(nil)
	_ = intPtr(1)
	_ = box(1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid conversions")
	}
	msg := diags.Error()
	for _, want := range []string{
		"conversion requires one argument: int",
		"conversion requires one argument: byte",
		"cannot convert string to int",
		"cannot convert int to bool",
		"cannot convert int to []byte",
		"cannot convert bool to string",
		"cannot convert nil to int",
		"cannot convert int to *int",
		"cannot convert int to box",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsSupportedConversionCalls(t *testing.T) {
	file, err := parse.FileSource("conversions_ok.go", []byte(`package main

type namedString string
type intPtr *int

func appMain() int {
	n := int(byte(65))
	b := byte(n)
	w := int64(n)
	f := float64(w)
	data := []byte("PASS\n")
	text := string(data)
	var named namedString = namedString(text)
	_ = string(named)
	value := 1
	ptr := intPtr(&value)
	if b != byte('A') {
		return 1
	}
	if *ptr != 1 {
		return 2
	}
	return len(text) + int(f) - 70
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid conversions: %v", diags)
	}
}

func TestFileAllowsNamedScalarTypes(t *testing.T) {
	file, err := parse.FileSource("named_scalars.go", []byte(`package main

type Count int
type Label string

type item struct {
	count Count
	label Label
}

func add(x Count, y Count) Count {
	return x + y
}

func appMain() int {
	var one Count = 1
	two := Count(2)
	var text Label = Label("ok")
	if add(one, two) != Count(3) {
		return 1
	}
	if string(text) != "ok" {
		return 2
	}
	value := item{count: 4, label: Label("go")}
	if value.count != Count(4) || string(value.label) != "go" {
		return 3
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named scalar types: %v", diags)
	}
}

func TestFileRejectsDefinedTypeAssignmentFromTypedValues(t *testing.T) {
	file, err := parse.FileSource("defined_type_assignability.go", []byte(`package main

type Count int

func take(c Count) int { return int(c) }

func makeCount() Count {
	i := 1
	return i
}

func appMain() int {
	var c Count
	i := 1
	c = i
	var fromInit Count = i
	var plain int
	named := Count(1)
	plain = named
	return take(i) + take(2) + int(c) + int(fromInit) + plain
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid defined-type assignments")
	}
	msg := diags.Error()
	for _, want := range []string{
		"return type mismatch: want Count, got int",
		"assignment type mismatch: c has Count, got int",
		"initializer type mismatch: fromInit has Count, got int",
		"assignment type mismatch: plain has int, got Count",
		"argument type mismatch in call to take: want Count, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsDistinctDefinedTypeBinaryOperands(t *testing.T) {
	file, err := parse.FileSource("defined_type_binary_operands.go", []byte(`package main

type A int
type B int

func appMain() int {
	a := A(1)
	b := B(2)
	_ = a + b
	_ = a == b
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted distinct defined-type binary operands")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operands for +: A and B",
		"invalid operands for ==: A and B",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNamedAggregateAssignability(t *testing.T) {
	file, err := parse.FileSource("named_aggregate_assignability_ok.go", []byte(`package main

type Count int
type Counts []Count

func takeUnnamed(xs []Count) int { return len(xs) }
func takeNamed(xs Counts) int { return len(xs) }
func returnUnnamed() []Count { return Counts{1} }
func returnNamed() Counts { return []Count{1} }

func appMain() int {
	named := Counts{1}
	unnamed := []Count{2}
	var asUnnamed []Count = named
	var asNamed Counts = unnamed
	asUnnamed = named
	asNamed = unnamed
	return takeUnnamed(named) + takeNamed(unnamed) + len(returnUnnamed()) + len(returnNamed()) + len(asUnnamed) + len(asNamed) - 8
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named aggregate assignability: %v", diags)
	}
}

func TestFileRejectsDifferentDefinedElementAggregateAssignability(t *testing.T) {
	file, err := parse.FileSource("defined_element_aggregate_assignability.go", []byte(`package main

type Count int

func takeCounts(xs []Count) int { return len(xs) }
func takeInts(xs []int) int { return len(xs) }

func appMain() int {
	counts := []Count{1}
	ints := []int{1}
	var badInts []int = counts
	var badCounts []Count = ints
	return takeInts(counts) + takeCounts(ints) + len(badInts) + len(badCounts)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted different defined-element aggregate assignments")
	}
	msg := diags.Error()
	for _, want := range []string{
		"initializer type mismatch: badInts has []int, got []Count",
		"initializer type mismatch: badCounts has []Count, got []int",
		"argument type mismatch in call to takeInts: want []int, got []Count",
		"argument type mismatch in call to takeCounts: want []Count, got []int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNamedPointerAggregateAssignability(t *testing.T) {
	file, err := parse.FileSource("named_pointer_assignability_ok.go", []byte(`package main

type IntPtr *int

func takeUnnamed(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func takeNamed(p IntPtr) int {
	if p == nil {
		return 0
	}
	return *p
}

func appMain() int {
	x := 1
	var named IntPtr = &x
	var unnamed *int = named
	named = unnamed
	if named != unnamed {
		return 1
	}
	return takeUnnamed(named) + takeNamed(unnamed) - 2
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named pointer aggregate assignability: %v", diags)
	}
}

func TestFileRejectsDifferentDefinedElementPointerComparisons(t *testing.T) {
	file, err := parse.FileSource("defined_element_pointer_compare.go", []byte(`package main

type Count int
type CountPtr *Count

func appMain() int {
	var count Count
	var named CountPtr = &count
	var plain *int
	if named == plain {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted different defined-element pointer comparison")
	}
	if !strings.Contains(diags.Error(), "invalid operands for ==: CountPtr and *int") {
		t.Fatalf("missing pointer comparison diagnostic in:\n%s", diags.Error())
	}
}

func TestFileRejectsNamedScalarConversionExpressionTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("named_scalar_conversion_types.go", []byte(`package main

type Count int
type Label string

func wantInt(v int) int { return v }
func number() int { return Label("bad") }

func appMain() int {
	var x int = Label("bad")
	_ = x
	return wantInt(Label("bad")) + number() + int(Count(1))
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named scalar conversion expression type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"return type mismatch: want int, got Label",
		"initializer type mismatch: x has int, got Label",
		"argument type mismatch in call to wantInt: want int, got Label",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsNamedScalarConversionOperatorMismatches(t *testing.T) {
	file, err := parse.FileSource("named_scalar_conversion_ops.go", []byte(`package main

type Count int
type Label string

func appMain() int {
	_ = []int{Count(1), Label("bad")}
	return Count(1) + Label("bad")
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named scalar conversion operator mismatch")
	}
	if !strings.Contains(diags.Error(), "invalid operands for +: Count and Label") {
		t.Fatalf("missing named conversion operand diagnostic in:\n%s", diags.Error())
	}
	if !strings.Contains(diags.Error(), "composite literal element type mismatch: want int, got Label") {
		t.Fatalf("missing named conversion composite diagnostic in:\n%s", diags.Error())
	}
}

func TestFileRejectsNamedScalarBinaryExpressionTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("named_scalar_binary_types.go", []byte(`package main

type Count int
type Label string

func wantString(s string) int { return len(s) }

func number() string {
	return Count(1) + 2
}

func appMain() int {
	var text string
	text = Count(1) + 2
	_ = wantString(Count(1) + 2)
	var n int
	n = Label("a") + "b"
	return Count(1) + 2
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named scalar binary expression type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"return type mismatch: want string, got Count",
		"assignment type mismatch: text has string, got Count",
		"argument type mismatch in call to wantString: want string, got Count",
		"assignment type mismatch: n has int, got Label",
		"return type mismatch: want int, got Count",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNamedScalarBinaryExpressionTypeInference(t *testing.T) {
	file, err := parse.FileSource("named_scalar_binary_types_ok.go", []byte(`package main

type Count int
type Label string

func wantCount(x Count) int { return int(x) }
func wantLabel(s Label) int { return len(s) }

func appMain() int {
	n := Count(1) + 2
	text := Label("a") + "b"
	return wantCount(n) + wantLabel(text) - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named scalar binary expression inference: %v", diags)
	}
}

func TestFileRejectsNamedScalarUnaryExpressionTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("named_scalar_unary_types.go", []byte(`package main

type Count int
type Flag bool

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func number() string {
	return -Count(1)
}

func appMain() int {
	var text string
	text = -Count(1)
	_ = wantString(+Count(1))
	var n int
	n = !Flag(true)
	_ = wantInt(!Flag(false))
	return -Count(1)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named scalar unary expression type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"return type mismatch: want string, got Count",
		"assignment type mismatch: text has string, got Count",
		"argument type mismatch in call to wantString: want string, got Count",
		"assignment type mismatch: n has int, got bool",
		"argument type mismatch in call to wantInt: want int, got bool",
		"return type mismatch: want int, got Count",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNamedScalarUnaryExpressionTypeInference(t *testing.T) {
	file, err := parse.FileSource("named_scalar_unary_types_ok.go", []byte(`package main

type Count int
type Flag bool

func wantCount(x Count) int { return int(x) }
func wantBool(v bool) int {
	if v {
		return 1
	}
	return 0
}

func appMain() int {
	n := -Count(1)
	ok := !Flag(false)
	return wantCount(n) + wantBool(ok) - 2
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named scalar unary expression inference: %v", diags)
	}
}

func TestFileAllowsNamedStringErrorReturns(t *testing.T) {
	file, err := parse.FileSource("named_string_error.go", []byte(`package main

type parseError string

func (err parseError) Error() string {
	return string(err)
}

func makeError() error {
	return parseError("bad")
}

func appMain() int {
	if makeError() == nil {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named string error return: %v", diags)
	}
}

func TestFileAllowsErrorStringMethod(t *testing.T) {
	file, err := parse.FileSource("error_method.go", []byte(`package main

type parseError string

func (err parseError) Error() string {
	return string(err)
}

func makeError() error {
	return parseError("bad")
}

func appMain() int {
	err := makeError()
	return len(err.Error())
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected error Error method: %v", diags)
	}
}

func TestFileKeepsParameterTypesPerSegment(t *testing.T) {
	file, err := parse.FileSource("parameter_segments.go", []byte(`package main

type Options struct { Target string }

func use(entries []string, opts Options) int {
	entry := entries[0]
	if opts.Target == "" {
		return len(entry)
	}
	return len(entries)
}

func appMain() int {
	return use([]string{"ok"}, Options{})
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File confused parameter segment types: %v", diags)
	}
}

func TestFileTreatsHexIntegerWithEDigitAsInt(t *testing.T) {
	file, err := parse.FileSource("hex_integer_e.go", []byte(`package main

func wantInt(x int) int { return x }

func appMain() int {
	return wantInt(0x8e) - 142
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File treated hex integer as float: %v", diags)
	}
}

func TestFileAllowsStructLiteralAndBoolReturn(t *testing.T) {
	file, err := parse.FileSource("struct_bool_return.go", []byte(`package main

type box struct { value int }

func pair() (box, bool) {
	return box{value: 1}, true
}

func appMain() int {
	b, ok := pair()
	if ok {
		return b.value - 1
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected struct literal and bool return: %v", diags)
	}
}

func TestFileAllowsMultilineStructLiteralAndBoolReturn(t *testing.T) {
	file, err := parse.FileSource("multiline_struct_bool_return.go", []byte(`package main

type info struct {
	name string
	ok bool
}

func pair() (info, bool) {
	return info{
		name: "ok",
		ok: true,
	}, true
}

func appMain() int {
	v, ok := pair()
	if ok && v.ok {
		return len(v.name) - 2
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected multiline struct literal and bool return: %v", diags)
	}
}

func TestFileRejectsInvalidExpressionStatements(t *testing.T) {
	file, err := parse.FileSource("expression_statements.go", []byte(`package main

type box struct { value int }

func appMain() int {
	x := 1
	b := box{value: 2}
	x + 1
	"bad"
	b.value
	[]byte("bad")
	int(x)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid expression statements")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "expression statement must be a call"); count != 5 {
		t.Fatalf("got %d expression-statement diagnostics, want 5:\n%s", count, msg)
	}
}

func TestFileAllowsValidExpressionStatements(t *testing.T) {
	file, err := parse.FileSource("expression_statements_ok.go", []byte(`package main

type box struct { value int }

func touch() int { return 1 }
func (b box) touch() int { return b.value }

func appMain() int {
	dst := []byte("a")
	src := []byte("b")
	b := box{value: 2}
	touch()
	b.touch()
	copy(dst, src)
	print("ok")
	println("ok")
	if touch() == 1 {
		dst = []byte("c")
	}
	for i := 0; i < len(dst); i++ {
		b = box{value: b.value + i}
	}
	_ = b
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid expression statements: %v", diags)
	}
}

func TestFileAllowsMultilineAppendCompositeLiteralCallClose(t *testing.T) {
	file, err := parse.FileSource("multiline_append_composite_close.go", []byte(`package main

type item struct {
	value int
}

func appMain() int {
	var values []item
	values = append(values, item{
		value: 1,
	})
	return len(values) - 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected multiline append composite literal call close: %v", diags)
	}
}

func TestFileRejectsMultiResultCallsInSingleValueContexts(t *testing.T) {
	file, err := parse.FileSource("multi_result_contexts.go", []byte(`package main

func pair() (int, int) { return 1, 2 }
func one(x int) int { return x }

func appMain() int {
	var x = pair()
	var y int = pair()
	_ = pair() + 1
	_ = -pair()
	_ = int(pair())
	_ = one(pair())
	return x + y
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted multi-result calls in single-value contexts")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "initializer count mismatch"); count != 2 {
		t.Fatalf("got %d initializer count diagnostics, want 2:\n%s", count, msg)
	}
	if count := strings.Count(msg, "multiple-value call in single-value context"); count != 4 {
		t.Fatalf("got %d single-value diagnostics, want 4:\n%s", count, msg)
	}
}

func TestFileRejectsMultiResultCallsInNestedSingleValueContexts(t *testing.T) {
	file, err := parse.FileSource("multi_result_nested_contexts.go", []byte(`package main

type box struct {
	value int
}

func pair() (int, int) { return 1, 2 }

func appMain() int {
	if pair() {
		return 1
	}
	switch pair() {
	case pair():
		return 2
	}
	for range pair() {
		return 3
	}
	_ = []int{pair()}
	_ = box{value: pair()}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted nested multi-result calls in single-value contexts")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "multiple-value call in single-value context"); count != 6 {
		t.Fatalf("got %d single-value diagnostics, want 6:\n%s", count, msg)
	}
}

func TestFileAllowsDirectMultiResultCallsInMultiValueContexts(t *testing.T) {
	file, err := parse.FileSource("multi_result_contexts_ok.go", []byte(`package main

func pair() (int, int) { return 1, 2 }

func both() (int, int) {
	return pair()
}

func appMain() int {
	a, b := pair()
	var c, d = pair()
	var e, f int = pair()
	x, y := both()
	return a + b + c + d + e + f + x + y - 12
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected direct multi-result calls: %v", diags)
	}
}

func TestFileAllowsCommaSeparatedVarNamesAsKnownLocals(t *testing.T) {
	file, err := parse.FileSource("var_names.go", []byte(`package main

func appMain() int {
	var a, b = 1, 2
	var c, d int = 3, 4
	return a + b + c + d - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected comma-separated var names: %v", diags)
	}
}

func TestFileRejectsBasicSemanticErrorsBeforeLowering(t *testing.T) {
	file, err := parse.FileSource("semantic.go", []byte(`package main

func pair() (int, int) {
    return 1
}

func one(x int) int { return x }

func appMain() int {
    _ = missing
    x := 1
    x()
    one()
    a, b := 1
    1 = 2
    _, _ = pair()
    return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted semantic errors")
	}
	msg := diags.Error()
	for _, want := range []string{
		"semantic.go:4:5: return value count mismatch",
		"semantic.go:10:9: undefined identifier: missing",
		"semantic.go:12:5: call of non-function: x",
		"semantic.go:13:5: argument count mismatch in call to one",
		"semantic.go:14:10: assignment count mismatch",
		"semantic.go:15:5: cannot assign to non-addressable expression",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "pair") {
		t.Fatalf("multi-result assignment from pair was diagnosed unexpectedly:\n%s", msg)
	}
}

func TestFileRejectsEveryInvalidAssignmentTarget(t *testing.T) {
	file, err := parse.FileSource("assignment_targets.go", []byte(`package main

func number() int { return 1 }

func appMain() int {
	x := 1
	y := 2
	x, 1 = 3, 4
	1, y = 5, 6
	x + y = 7
	number() = 8
	int(x) = 9
	(x + y) = 10
	true = false
	false = true
	nil = nil
	return x + y
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid assignment targets")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "cannot assign to non-addressable expression"); count != 9 {
		t.Fatalf("got %d assignment target diagnostics, want 9:\n%s", count, msg)
	}
}

func TestFileAllowsValidAssignmentTargets(t *testing.T) {
	file, err := parse.FileSource("assignment_targets_ok.go", []byte(`package main

type box struct { value int }

func appMain() int {
	x := 1
	values := []int{1}
	b := box{value: 1}
	p := new(box)
	_ = x
	x = 2
	values[0] = x
	b.value = values[0]
	p.value = b.value
	*p = b
	(values)[0] = p.value
	return x + values[0] + b.value + p.value - 8
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid assignment targets: %v", diags)
	}
}

func TestFileRejectsInvalidAddressOfOperands(t *testing.T) {
	file, err := parse.FileSource("address_of.go", []byte(`package main

func value() int { return 1 }

func appMain() int {
	x := 1
	_ = &1
	_ = &(x + 1)
	_ = &value()
	_ = &true
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid address-of operands")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "cannot take address of non-addressable expression"); count != 4 {
		t.Fatalf("got %d address-of diagnostics, want 4:\n%s", count, msg)
	}
}

func TestFileAllowsValidAddressOfOperands(t *testing.T) {
	file, err := parse.FileSource("address_of_ok.go", []byte(`package main

type box struct { value int }

func appMain() int {
	x := 1
	values := []int{1}
	b := box{value: 2}
	p := &x
	_ = p
	_ = &x
	_ = &values[0]
	_ = &b.value
	_ = &*p
	_ = &box{value: 3}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid address-of operands: %v", diags)
	}
}

func TestFileRejectsStringIndexMutationAndAddress(t *testing.T) {
	file, err := parse.FileSource("string_index_mutation.go", []byte(`package main

func appMain() int {
	text := "abc"
	text[0] = byte(65)
	text[1]++
	text[2] += 1
	_ = &text[0]
	_ = &(text[1])
	text[0:1] = "x"
	_ = &text[0:1]
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted string index mutation and address")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "cannot assign to string index"); count != 3 {
		t.Fatalf("got %d string index assignment diagnostics, want 3:\n%s", count, msg)
	}
	if count := strings.Count(msg, "cannot take address of string index"); count != 2 {
		t.Fatalf("got %d string index address diagnostics, want 2:\n%s", count, msg)
	}
	if count := strings.Count(msg, "cannot assign to non-addressable expression"); count != 1 {
		t.Fatalf("got %d non-addressable assignment diagnostics, want 1:\n%s", count, msg)
	}
	if count := strings.Count(msg, "cannot take address of non-addressable expression"); count != 1 {
		t.Fatalf("got %d non-addressable address diagnostics, want 1:\n%s", count, msg)
	}
}

func TestFileRejectsConstAssignmentTargets(t *testing.T) {
	file, err := parse.FileSource("const_assignment_targets.go", []byte(`package main

const Global = 1
const (
	Grouped = 2
	Other = Grouped + 1
)

func appMain() int {
	const Local = 4
	const (
		Inner = 5
	)
	Global = 6
	Grouped += 1
	Local++
	Inner--
	return Global + Grouped + Other + Local + Inner
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted constant assignment targets")
	}
	msg := diags.Error()
	for _, want := range []string{
		"cannot assign to constant: Global",
		"cannot assign to constant: Grouped",
		"cannot assign to constant: Local",
		"cannot assign to constant: Inner",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "cannot assign to constant:"); count != 4 {
		t.Fatalf("got %d constant assignment diagnostics, want 4:\n%s", count, msg)
	}
}

func TestFileAllowsLocalShadowOfPackageConstAssignment(t *testing.T) {
	file, err := parse.FileSource("const_shadow_assignment.go", []byte(`package main

const Global = 1

func useParam(Global int) int {
	Global = 2
	return Global
}

func appMain() int {
	Global := 3
	Global += 1
	Global++
	return Global + useParam(1) - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected assignment to local shadows of package constants: %v", diags)
	}
}

func TestFileRejectsConstRangeAssignmentTargets(t *testing.T) {
	file, err := parse.FileSource("const_range_assignment.go", []byte(`package main

const Global = 0

func appMain() int {
	values := []int{1}
	const Local = 0
	for Global = range values {
	}
	for _, Local = range values {
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted constant range assignment targets")
	}
	msg := diags.Error()
	for _, want := range []string{
		"cannot assign to constant: Global",
		"cannot assign to constant: Local",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "cannot assign to constant:"); count != 2 {
		t.Fatalf("got %d constant assignment diagnostics, want 2:\n%s", count, msg)
	}
}

func TestFileRejectsInvalidCompoundAssignments(t *testing.T) {
	file, err := parse.FileSource("compound_assign.go", []byte(`package main

type Count int

func text() string { return "bad" }

func appMain() int {
	x := 1
	c := Count(1)
	x += "bad"
	x %= text()
	x += c
	c += x
	1 += x
	return x
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid compound assignments")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operands for +=: int and string",
		"invalid operands for %=: int and string",
		"invalid operands for +=: int and Count",
		"invalid operands for +=: Count and int",
		"cannot assign to non-addressable expression",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsStringCompoundAssignments(t *testing.T) {
	file, err := parse.FileSource("string_compound_assign.go", []byte(`package main

type box struct {
	text string
}

func suffix() string { return "!" }

func appMain() int {
	s := "a"
	s += "b"
	b := box{text: "c"}
	b.text += suffix()
	xs := []string{"d"}
	xs[0] += s
	return len(s) + len(b.text) + len(xs[0])
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected string compound assignments: %v", diags)
	}
}

func TestFileRejectsRangeStructCompoundAssignmentMismatches(t *testing.T) {
	file, err := parse.FileSource("range_struct_compound_assign.go", []byte(`package main

type box struct {
	text string
	count int
}

func appMain() int {
	boxes := []box{{text: "bad", count: 1}}
	for _, b := range boxes {
		b.text += 1
		b.count += "bad"
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted range struct compound assignment mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operands for +=: string and int",
		"invalid operands for +=: int and string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsValidCompoundAssignments(t *testing.T) {
	file, err := parse.FileSource("compound_assign_ok.go", []byte(`package main

type Count int

func number() int { return 2 }

func appMain() int {
	x := 10
	c := Count(1)
	x += number()
	x -= 1
	x *= 2
	x /= 2
	x %= 5
	c += 1
	c += Count(2)
	return x + int(c) - 4
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid compound assignments: %v", diags)
	}
}

func TestFileAllowsRangeStructCompoundAssignments(t *testing.T) {
	file, err := parse.FileSource("range_struct_compound_assign_ok.go", []byte(`package main

type box struct {
	count int
}

func appMain() int {
	total := 0
	boxes := []box{{count: 1}}
	for _, b := range boxes {
		b.count += 2
		total = total + b.count
	}
	return total - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected range struct compound assignments: %v", diags)
	}
}

func TestFileRejectsInvalidIncDecStatements(t *testing.T) {
	file, err := parse.FileSource("incdec.go", []byte(`package main

func appMain() int {
	text := "bad"
	flag := true
	text++
	flag--
	1++
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid inc/dec statements")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operand for ++: string",
		"invalid operand for --: bool",
		"cannot assign to non-addressable expression",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsRangeStructIncDecMismatches(t *testing.T) {
	file, err := parse.FileSource("range_struct_incdec.go", []byte(`package main

type box struct {
	text string
}

func appMain() int {
	boxes := []box{{text: "bad"}}
	for _, b := range boxes {
		b.text++
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted range struct inc/dec mismatches")
	}
	if !strings.Contains(diags.Error(), "invalid operand for ++: string") {
		t.Fatalf("missing inc/dec diagnostic in:\n%s", diags.Error())
	}
}

func TestFileAllowsValidIncDecStatements(t *testing.T) {
	file, err := parse.FileSource("incdec_ok.go", []byte(`package main

type counter struct { count int }

func appMain() int {
	x := 1
	x++
	x--
	values := []int{1}
	values[0]++
	c := counter{count: 1}
	c.count++
	return x + values[0] + c.count - 4
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid inc/dec statements: %v", diags)
	}
}

func TestFileAllowsRangeStructIncDecStatements(t *testing.T) {
	file, err := parse.FileSource("range_struct_incdec_ok.go", []byte(`package main

type counter struct { count int }

func appMain() int {
	boxes := []counter{{count: 1}}
	for _, c := range boxes {
		c.count++
		c.count--
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected range struct inc/dec statements: %v", diags)
	}
}

func TestFileRejectsShortDeclWithoutNewNames(t *testing.T) {
	file, err := parse.FileSource("short_decl.go", []byte(`package main

func appMain() int {
	x := 1
	x := 2
	x, _ := 3, 4
	_ := 5
	return x
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted short declarations without new names")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "no new variables on left side of :="); count != 3 {
		t.Fatalf("got %d short declaration diagnostics, want 3:\n%s", count, msg)
	}
}

func TestFileAllowsShortDeclWithNewNamesAndInnerShadowing(t *testing.T) {
	file, err := parse.FileSource("short_decl_ok.go", []byte(`package main

func appMain() int {
	x := 1
	x, y := 2, 3
	{
		x := 4
		y = y + x
	}
	if x := 5; x == 5 {
		y = y + x
	}
	return x + y - 15
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid short declarations: %v", diags)
	}
}

func TestFileAllowsConflictingShortDeclTypesInSeparateScopes(t *testing.T) {
	file, err := parse.FileSource("short_decl_scope_types.go", []byte(`package main

func text() string { return "ok" }

func appMain() int {
	if op := text(); op != "" {
		if len(op) != 2 {
			return 1
		}
	}
	op := 2
	return op - 2
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected scoped short declarations with different types: %v", diags)
	}
}

func TestFileAllowsRepeatedShortDeclNamesInSwitchCases(t *testing.T) {
	file, err := parse.FileSource("short_decl_switch_cases.go", []byte(`package main

func appMain() int {
	x := 0
	switch x {
	case 0:
		args := []int{1}
		x = args[0]
	default:
		args := []int{2}
		x = args[0]
	}
	return x - 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected repeated switch case short declarations: %v", diags)
	}
}

func TestFileRejectsSimpleAssignmentTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("assign_types.go", []byte(`package main

func appMain() int {
	var x int
	x = "bad"
	return x
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted assignment type mismatch")
	}
	if !strings.Contains(err.Error(), "assign_types.go:5:4: assignment type mismatch: x has int, got string") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsInvalidNilTypedContexts(t *testing.T) {
	file, err := parse.FileSource("nil_typed_contexts.go", []byte(`package main

func wantInt(x int) int { return x }

func returnsInt() int {
	return nil
}

func appMain() int {
	var x int = nil
	var text string
	text = nil
	_ = wantInt(nil)
	_ = []int{nil}
	return x
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid nil typed contexts")
	}
	msg := diags.Error()
	for _, want := range []string{
		"return type mismatch: want int, got nil",
		"initializer type mismatch: x has int, got nil",
		"assignment type mismatch: text has string, got nil",
		"argument type mismatch in call to wantInt: want int, got nil",
		"composite literal element type mismatch: want int, got nil",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNilInNilableTypedContexts(t *testing.T) {
	file, err := parse.FileSource("nil_typed_contexts_ok.go", []byte(`package main

var packageValues []int = nil

func returnsSlice() []int {
	return nil
}

func returnsError() error {
	return nil
}

func wantSlice(values []int) int { return len(values) }
func wantError(err error) int {
	if err == nil {
		return 1
	}
	return 0
}

func appMain() int {
	var values []int = nil
	var err error = nil
	values = nil
	err = returnsError()
	_ = []error{nil, err}
	return len(packageValues) + len(values) + len(returnsSlice()) + len([]error{err}) + wantSlice(nil) + wantError(nil) + wantError(returnsError()) - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected nilable nil typed contexts: %v", diags)
	}
}

func TestFileRejectsAssignmentTargetTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("assign_target_types.go", []byte(`package main

type box struct {
	text string
	count int
}

func appMain() int {
	b := box{text: "ok", count: 1}
	values := []int{1}
	words := []string{"ok"}
	b.text = 1
	b.count = "bad"
	values[0] = "bad"
	words[0] = 1
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted assignment target type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: b.text has string, got int",
		"assignment type mismatch: b.count has int, got string",
		"assignment type mismatch: values[0] has int, got string",
		"assignment type mismatch: words[0] has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsMultiAssignmentTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("multi_assign_types.go", []byte(`package main

func text() string { return "bad" }
func number() int { return 1 }

func appMain() int {
	var x int
	var s string
	x, s = "bad", 1
	x, s = text(), number()
	return x
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted multi-assignment type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"assignment type mismatch: s has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsMultiAssignmentTargetTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("multi_assign_target_types.go", []byte(`package main

type box struct {
	text string
	count int
}

func appMain() int {
	b := box{text: "ok", count: 1}
	values := []int{1}
	words := []string{"ok"}
	b.text, values[0] = 1, "bad"
	b.count, words[0] = "bad", 1
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted multi-assignment target type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: b.text has string, got int",
		"assignment type mismatch: values[0] has int, got string",
		"assignment type mismatch: b.count has int, got string",
		"assignment type mismatch: words[0] has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsMultiResultAssignmentTargetTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("multi_result_assign_target_types.go", []byte(`package main

type box struct {
	text string
	count int
}

func pair() (int, string) { return 1, "bad" }
func reverse() (string, int) { return "bad", 1 }

func appMain() int {
	b := box{text: "ok", count: 1}
	values := []int{1}
	words := []string{"ok"}
	b.text, values[0] = pair()
	b.count, words[0] = reverse()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted multi-result assignment target type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"assignment type mismatch: b.text has string, got int",
		"assignment type mismatch: values[0] has int, got string",
		"assignment type mismatch: b.count has int, got string",
		"assignment type mismatch: words[0] has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsMultiAssignmentTypeMatches(t *testing.T) {
	file, err := parse.FileSource("multi_assign_types_ok.go", []byte(`package main

func text() string { return "ok" }
func number() int { return 1 }

func appMain() int {
	var x int
	var s string
	x, s = 1, "ok"
	x, s = number(), text()
	return x + len(s) - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid multi-assignment types: %v", diags)
	}
}

func TestFileRejectsSimpleCallArgumentTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("call_types.go", []byte(`package main

func one(x int) int { return x }

func appMain() int {
	return one("bad")
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted call argument type mismatch")
	}
	if !strings.Contains(err.Error(), "call_types.go:6:13: argument type mismatch in call to one: want int, got string") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsSimpleReturnTypeMismatch(t *testing.T) {
	file, err := parse.FileSource("return_types.go", []byte(`package main

func value() int {
	return "bad"
}

func appMain() int { return value() }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted return type mismatch")
	}
	if !strings.Contains(err.Error(), "return_types.go:4:9: return type mismatch: want int, got string") {
		t.Fatalf("error = %q", err)
	}
}

func TestFileRejectsMultiResultTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("multi_result_types.go", []byte(`package main

func pair() (string, int) { return "bad", 1 }
func wantInt(x int) int { return x }

func values() (int, string) {
	return pair()
}

func appMain() int {
	a, _ := pair()
	_ = wantInt(a)
	var p, q int = pair()
	_, _ = p, q
	var x int
	var y string
	x, y = pair()
	return x + len(y)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted multi-result type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"return type mismatch: want int, got string",
		"return type mismatch: want string, got int",
		"argument type mismatch in call to wantInt: want int, got string",
		"initializer type mismatch: p has int, got string",
		"assignment type mismatch: x has int, got string",
		"assignment type mismatch: y has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsMultiResultTypeInference(t *testing.T) {
	file, err := parse.FileSource("multi_result_types_ok.go", []byte(`package main

func pair() (int, string) { return 1, "ok" }
func wantInt(x int) int { return x }
func wantString(s string) int { return len(s) }

func values() (int, string) {
	return pair()
}

func appMain() int {
	a, b := pair()
	var c, d = pair()
	var x int
	var s string
	x, s = pair()
	return wantInt(a) + wantString(b) + wantInt(c) + wantString(d) + wantInt(x) + wantString(s) - 9
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected multi-result type inference: %v", diags)
	}
}

func TestFileRejectsUnknownAndNonFunctionCalls(t *testing.T) {
	file, err := parse.FileSource("calls.go", []byte(`package main

var packageValue = 1

func appMain() int {
	local := 2
	packageValue()
	local()
	missing()
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid calls")
	}
	msg := err.Error()
	for _, want := range []string{
		"calls.go:7:2: call of non-function: packageValue",
		"calls.go:8:2: call of non-function: local",
		"calls.go:9:2: undefined identifier: missing",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsSyscallBuiltin(t *testing.T) {
	file, err := parse.FileSource("syscall.go", []byte(`package main

func appMain() int {
	return syscall(1, 2, 3)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected syscall builtin: %v", diags)
	}
}

func TestGraphAllowsBackendRuntimeIntrinsicsOnlyInBackendPackage(t *testing.T) {
	backend := load.Package{
		ImportPath: "j5.nz/rtg",
		Name:       "rtg",
		Files: []load.File{
			{
				Path: "backend.go",
				Source: []byte(`package rtg

func appMain() int {
	fd := open("out", 1)
	buf := []byte("ok")
	_ = write(fd, buf, -1)
	_ = read(fd, buf, int64(-1))
	_ = chmod(fd, 493)
	_ = close(fd)
	return fd
}
`),
			},
		},
	}
	if err := Graph(&load.Graph{Packages: []load.Package{backend}}); err != nil {
		t.Fatalf("Graph rejected backend runtime intrinsics: %v", err)
	}

	ordinary := backend
	ordinary.ImportPath = "example.com/app"
	err := Graph(&load.Graph{Packages: []load.Package{ordinary}})
	if err == nil {
		t.Fatalf("Graph accepted backend runtime intrinsics in ordinary package")
	}
	msg := err.Error()
	if !strings.Contains(msg, "undefined identifier: open") || !strings.Contains(msg, "undefined identifier: write") {
		t.Fatalf("ordinary package diagnostics did not reject runtime intrinsics:\n%s", msg)
	}
}

func TestFileAllowsSimpleTypedAssignmentsAndCalls(t *testing.T) {
	file, err := parse.FileSource("typed_ok.go", []byte(`package main

func one(x int, s string, b bool) int { return x }
func stringValue(s string) string { return s }

func appMain() int {
	var x int
	x = 1
	s := "ok"
	b := true
	return one(x, s, b) + len(stringValue(s))
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected simple typed locals: %v", diags)
	}
}

func TestFileRejectsCompositeExpressionTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("composite_types.go", []byte(`package main

func text() string { return "bad" }
func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = text() + "!"
	_ = wantInt(text() + "!")
	return text() + "!"
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted composite expression type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsCompositeExpressionTypeInference(t *testing.T) {
	file, err := parse.FileSource("composite_types_ok.go", []byte(`package main

func text() string { return "ok" }
func number() int { return 1 }
func flag() bool { return true }
func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }
func wantBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func appMain() int {
	s := text() + "!"
	n := number() + 2
	b := flag() && n == 3
	return wantString(s) + wantInt(n) + wantBool(b) - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected composite expression inference: %v", diags)
	}
}

func TestFileRejectsCompositeLiteralElementTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("composite_literal_values.go", []byte(`package main

type inner struct {
	text string
	count int
}

type box struct {
	name string
	count int
	inner inner
	values []int
}

func appMain() int {
	_ = box{name: 1, count: "bad", missing: 3, inner: inner{text: 1, count: "bad"}, values: []int{1, "bad"}}
	_ = []string{"ok", 2}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted composite literal element type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"composite literal field type mismatch: box.name has string, got int",
		"composite literal field type mismatch: box.count has int, got string",
		"unknown field: box.missing",
		"composite literal field type mismatch: inner.text has string, got int",
		"composite literal field type mismatch: inner.count has int, got string",
		"composite literal element type mismatch: want int, got string",
		"composite literal element type mismatch: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsNamedSliceCompositeLiteralElementTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("named_slice_composite_literal_values.go", []byte(`package main

type scores []int
type labels []string
type label string
type row struct { value int }
type rows []row

var packageScores = scores{1, "bad"}

func appMain() int {
	_ = scores{1, "bad"}
	_ = labels{"ok", 2}
	_ = rows{{value: label("bad")}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named slice composite literal element type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"composite literal element type mismatch: want int, got string",
		"composite literal element type mismatch: want string, got int",
		"composite literal field type mismatch: row.value has int, got label",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsSliceCompositeLiteralKeyMismatches(t *testing.T) {
	file, err := parse.FileSource("slice_literal_keys.go", []byte(`package main

type scores []int

type box struct {
	values []int
}

func index() int { return 1 }

func appMain() int {
	_ = []int{"bad": 1, true: 2, 0: 3, 00: 4}
	_ = scores{"bad": 1}
	_ = box{values: []int{false: 1}}
	_ = []int{index(): 5, -1: 6}
	_ = []int{1, 0: 2}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted slice composite literal key mismatches")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "slice literal index must be integer, got string"); count != 2 {
		t.Fatalf("got %d string key diagnostics, want 2:\n%s", count, msg)
	}
	if count := strings.Count(msg, "slice literal index must be integer, got bool"); count != 2 {
		t.Fatalf("got %d bool key diagnostics, want 2:\n%s", count, msg)
	}
	if count := strings.Count(msg, "duplicate index in slice literal: 0"); count != 2 {
		t.Fatalf("got %d duplicate index diagnostics, want 2:\n%s", count, msg)
	}
	for _, want := range []string{
		"slice literal index must be integer constant",
		"slice literal index must be non-negative",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsValidSliceCompositeLiteralKeys(t *testing.T) {
	file, err := parse.FileSource("slice_literal_keys_ok.go", []byte(`package main

type scores []int

type box struct {
	values []int
}

func appMain() int {
	_ = []int{0: 1, 2: 3}
	_ = scores{1: 4}
	_ = box{values: []int{2: 5}}
	_ = []int{1, 3: 4, 5}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid keyed slice literals: %v", diags)
	}
}

func TestFileRejectsImplicitCompositeLiteralElementTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("implicit_composite_literal_values.go", []byte(`package main

type inner struct {
	text string
	count int
}

type box struct {
	values []inner
	scalar int
}

func appMain() int {
	_ = []inner{{text: 1, count: "bad"}, {missing: 3}}
	_ = box{values: []inner{{count: "bad"}}, scalar: {1}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted implicit composite literal element type mismatches")
	}
	msg := diags.Error()
	for _, want := range []string{
		"composite literal field type mismatch: inner.text has string, got int",
		"composite literal field type mismatch: inner.count has int, got string",
		"unknown field: inner.missing",
		"composite literal field type mismatch: box.scalar has int, got composite literal",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsCompositeLiteralElementTypes(t *testing.T) {
	file, err := parse.FileSource("composite_literal_values_ok.go", []byte(`package main

type inner struct {
	text string
	count int
}

type box struct {
	name string
	count int
	inner inner
	values []int
}

func appMain() int {
	b := box{name: "ok", count: 1, inner: inner{text: "in", count: 2}, values: []int{1, 2}}
	words := []string{"a", "b"}
	return b.count + b.inner.count + len(b.values) + len(words) - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected composite literal element types: %v", diags)
	}
}

func TestFileAllowsNamedSliceCompositeLiteralElementTypes(t *testing.T) {
	file, err := parse.FileSource("named_slice_composite_literal_values_ok.go", []byte(`package main

type scores []int
type labels []string

var packageScores = scores{1, 2}

func appMain() int {
	values := scores{1, 2}
	names := labels{"a", "b"}
	return len(packageScores) + len(values) + len(names) - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named slice composite literal element types: %v", diags)
	}
}

func TestFileAllowsNamedSliceCompositeLiteralExpressionTypes(t *testing.T) {
	file, err := parse.FileSource("named_slice_composite_literal_expr_ok.go", []byte(`package main

type scores []int

func take(values scores) int { return len(values) }
func values() scores { return scores{1, 2} }

func appMain() int {
	var direct scores
	direct = scores{3, 4}
	return take(scores{5, 6}) + len(scores{7}) + len(direct) + len(values()) - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected named slice composite literal expression types: %v", diags)
	}
}

func TestFileAllowsImplicitCompositeLiteralElementTypes(t *testing.T) {
	file, err := parse.FileSource("implicit_composite_literal_values_ok.go", []byte(`package main

type inner struct {
	text string
	count int
}

type box struct {
	values []inner
}

func appMain() int {
	values := []inner{{text: "ok", count: 1}, {text: "in", count: 2}}
	b := box{values: []inner{{text: "v", count: 4}}}
	return len(values) + len(b.values) - 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected implicit composite literal element types: %v", diags)
	}
}

func TestFileRejectsInvalidStructLiteralShape(t *testing.T) {
	file, err := parse.FileSource("struct_literal_shape.go", []byte(`package main

type box struct {
	value int
	name string
}

func appMain() int {
	_ = box{1, "ok", 3}
	_ = box{value: 1, value: 2}
	_ = box{1, name: "bad"}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid struct literal shape")
	}
	msg := err.Error()
	for _, want := range []string{
		"too many values in composite literal: box",
		"duplicate field: box.value",
		"cannot mix keyed and positional composite literal values: box",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsInvalidImplicitStructLiteralShape(t *testing.T) {
	file, err := parse.FileSource("implicit_struct_literal_shape.go", []byte(`package main

type box struct {
	value int
	name string
}

func appMain() int {
	_ = []box{{1, "ok", 3}}
	_ = []box{{value: 1, value: 2}}
	_ = []box{{1, name: "bad"}}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid implicit struct literal shape")
	}
	msg := err.Error()
	for _, want := range []string{
		"too many values in composite literal: box",
		"duplicate field: box.value",
		"cannot mix keyed and positional composite literal values: box",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsInvalidBinaryOperands(t *testing.T) {
	file, err := parse.FileSource("binary_operands.go", []byte(`package main

func flag() bool { return true }

func appMain() int {
	_ = 1 + "bad"
	_ = 1 && flag()
	_ = "a" - "b"
	_ = true < false
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted invalid binary operands")
	}
	msg := err.Error()
	for _, want := range []string{
		"invalid operands for +: int and string",
		"invalid operands for &&: int and bool",
		"invalid operands for -: string and string",
		"invalid operands for <: bool and bool",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsSliceComparisons(t *testing.T) {
	file, err := parse.FileSource("slice_comparisons.go", []byte(`package main

func appMain() int {
	xs := []int{1}
	ys := []int{1}
	_ = xs == ys
	_ = xs != ys
	_ = []int{1} == 1
	_ = []int{1} < []int{2}
	if []int{1} < []int{2} {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted slice comparisons")
	}
	msg := err.Error()
	for _, want := range []string{
		"invalid operands for ==: []int and []int",
		"invalid operands for !=: []int and []int",
		"invalid operands for ==: []int and int",
		"invalid operands for <: []int and []int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsSliceCompositeLiteralControlConditionComparisons(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

func appMain() int {
	if []int{1} < []int{2} {
		return 1
	}
	return 0
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted slice composite literal comparison in condition")
	}
	if !strings.Contains(err.Error(), "invalid operands for <: []int and []int") {
		t.Fatalf("missing slice comparison diagnostic in:\n%s", err)
	}
}

func TestFileRejectsInvalidNilComparisons(t *testing.T) {
	file, err := parse.FileSource("nil_comparisons.go", []byte(`package main

func appMain() int {
	_ = 1 == nil
	_ = nil != "bad"
	_ = true == nil
	_ = nil == nil
	_ = nil < 1
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid nil comparisons")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operands for ==: int and nil",
		"invalid operands for !=: nil and string",
		"invalid operands for ==: bool and nil",
		"invalid operands for ==: nil and nil",
		"invalid operands for <: nil and int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNilComparisonsWithNilableTypes(t *testing.T) {
	file, err := parse.FileSource("nil_comparisons_ok.go", []byte(`package main

func makeError() error {
	return nil
}

func appMain() int {
	x := 1
	p := &x
	values := []int{1}
	err := makeError()
	if p != nil && nil != p && values != nil && nil == values && err == nil && nil != err {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid nil comparisons: %v", diags)
	}
}

func TestFileAllowsSimpleStructComparisons(t *testing.T) {
	file, err := parse.FileSource("struct_comparisons.go", []byte(`package main

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

type empty struct{}

type values [2]int

type namedArrayBox struct {
	values values
}

func same(left box, right box) bool {
	return left == right
}

func different(left box, right box) bool {
	return left != right
}

func sameEmpty(left empty, right empty) bool {
	return left == right
}

func sameNamedArray(left namedArrayBox, right namedArrayBox) bool {
	return left == right
}

func makeBox(value int) box {
	return box{value: value, name: "a", ok: true, ptr: nil, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
}

func appMain() int {
	x := 1
	left := box{value: 1, name: "a", ok: true, ptr: &x, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
	right := box{value: 1, name: "a", ok: true, ptr: &x, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
	emptyLeft := empty{}
	emptyRight := empty{}
	namedArrayLeft := namedArrayBox{values: values{1, 2}}
	namedArrayRight := namedArrayBox{values: values{1, 2}}
	_ = left == right
	_ = left != right
	_ = same(left, right)
	_ = different(left, right)
	_ = emptyLeft == emptyRight
	_ = emptyLeft != emptyRight
	_ = sameEmpty(emptyLeft, emptyRight)
	_ = namedArrayLeft == namedArrayRight
	_ = namedArrayLeft != namedArrayRight
	_ = sameNamedArray(namedArrayLeft, namedArrayRight)
	_ = box{value: 1, name: "a", ok: true, ptr: nil, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}} == box{value: 1, name: "a", ok: true, ptr: nil, values: [2]int{1, 2}, inner: innerBox{count: 3, tag: "b", flags: [2]bool{true, false}}}
	_ = makeBox(1) == makeBox(1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected simple struct comparisons: %v", diags)
	}
}

func TestGraphAllowsImportedStructComparisons(t *testing.T) {
	mainPkg := load.Package{
		ImportPath: "example.com/app/cmd/app",
		Name:       "main",
		Imports:    []string{"example.com/app/pkg/dep"},
		Files: []load.File{
			{
				Path: "main.go",
				Source: []byte(`package main

import "example.com/app/pkg/dep"

func main() {
	globalSame := dep.Left == dep.Right
	callSame := dep.Make(1) == dep.Make(1)
	literalSame := dep.Box{Value: 1, Name: "a"} == dep.Box{Value: 1, Name: "a"}
	if globalSame && callSame && literalSame && (dep.Box{Value: 2, Name: "b"}) == (dep.Box{Value: 2, Name: "b"}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
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
	if err := Graph(&load.Graph{Packages: []load.Package{mainPkg, depPkg}}); err != nil {
		t.Fatalf("Graph rejected imported struct comparisons: %v", err)
	}
}

func TestFileRejectsUnsupportedStructComparisons(t *testing.T) {
	file, err := parse.FileSource("struct_comparisons.go", []byte(`package main

type bad struct {
	values []int
}

type box struct {
	value int
}

func appMain() int {
	left := bad{values: []int{1}}
	right := bad{values: []int{1}}
	_ = left == right
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted unsupported struct comparisons")
	}
	msg := err.Error()
	if count := strings.Count(msg, "struct comparisons are not supported: bad"); count != 1 {
		t.Fatalf("got %d bad struct comparison diagnostics, want 1:\n%s", count, msg)
	}
}

func TestFileRejectsInvalidUnaryOperands(t *testing.T) {
	file, err := parse.FileSource("unary_operands.go", []byte(`package main

func appMain() int {
	_ = !1
	_ = -"bad"
	_ = +true
	if !"bad" {
		return 1
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid unary operands")
	}
	msg := diags.Error()
	for _, want := range []string{
		"invalid operand for !: int",
		"invalid operand for -: string",
		"invalid operand for +: bool",
		"invalid operand for !: string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsUnaryOperandTypes(t *testing.T) {
	file, err := parse.FileSource("unary_operands_ok.go", []byte(`package main

func number() int { return 1 }
func ready() bool { return true }

func appMain() int {
	x := 2
	if !ready() || !(number() == 1) {
		return 1
	}
	x = -number() + +x
	return x + 3
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid unary operands: %v", diags)
	}
}

func TestFileAllowsSliceExpressionsForTypedAssignmentsAndCalls(t *testing.T) {
	file, err := parse.FileSource("slice_types.go", []byte(`package main

type Count int

func takesBytes(values []byte) int { return len(values) }
func takesString(value string) int { return len(value) }

func appMain() int {
	values := []byte{1, 2, 3}
	text := "abcd"
	i := Count(1)
	values = values[1:len(values)]
	values = values[i:Count(len(values))]
	text = text[:2]
	return takesBytes(values[0:1]) + takesString(text[0:1])
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected slice expressions in typed contexts: %v", diags)
	}
}

func TestFileRejectsInvalidIndexAndSliceBoundTypes(t *testing.T) {
	file, err := parse.FileSource("index_bounds.go", []byte(`package main

func key() string { return "x" }

func appMain() int {
	text := "abc"
	values := []int{1, 2, 3}
	start := "x"
	_ = text["x"]
	_ = values["x"]
	_ = values[start:1]
	_ = values[0:start]
	_ = values[key()]
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid index and slice bound types")
	}
	msg := diags.Error()
	if count := strings.Count(msg, "index must be integer, got string"); count != 3 {
		t.Fatalf("got %d invalid index diagnostics, want 3:\n%s", count, msg)
	}
	if count := strings.Count(msg, "slice bound must be integer, got string"); count != 2 {
		t.Fatalf("got %d invalid slice bound diagnostics, want 2:\n%s", count, msg)
	}
}

func TestFileAllowsStaticFunctionAliases(t *testing.T) {
	file, err := parse.FileSource("function_aliases.go", []byte(`package main

func add(a int) int { return a + 1 }
func join(a int, b int) int { return a + b }

func appMain() int {
	f := add
	var g = join
	return f(1) + g(2, 3) - 7
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static function aliases: %v", diags)
	}
}

func TestFileAllowsStaticFunctionLiteralAliases(t *testing.T) {
	file, err := parse.FileSource("function_literal_aliases.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static function literal aliases: %v", diags)
	}
}

func TestFileAllowsAssignedFunctionLiteralCaptures(t *testing.T) {
	file, err := parse.FileSource("function_literal_assigned_capture.go", []byte(`package main

func appMain() int {
	x := 1
	f := func() int {
		x = x + 1
		x++
		x += 2
		return x
	}
	return f() + x - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected assigned function literal capture: %v", diags)
	}
}

func TestFileAllowsAssignedImmediatelyInvokedFunctionLiteralCaptures(t *testing.T) {
	file, err := parse.FileSource("function_literal_call_assigned_capture.go", []byte(`package main

func appMain() int {
	x := 1
	value := func() int {
		x = x + 1
		x++
		x += 2
		return x
	}()
	return value + x - 10
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected assigned immediately invoked function literal capture: %v", diags)
	}
}

func TestFileAllowsImmediatelyInvokedFunctionLiterals(t *testing.T) {
	file, err := parse.FileSource("function_literal_calls.go", []byte(`package main

func appMain() int {
	value := func(a int) int { return a + 1 }(1)
	if func(text string) bool { return len(text) == 4 }("PASS") {
		total := func(a int, b int) int {
			total := a + b
			return total
		}(value, 5)
		return total
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected immediately invoked function literals: %v", diags)
	}
}

func TestFileAllowsBlankDiscardImmediatelyInvokedFunctionLiterals(t *testing.T) {
	file, err := parse.FileSource("function_literal_call_blank_discard.go", []byte(`package main

func appMain() int {
	_ = func() int {
		return 1
	}()
	type Values [2]int
	values := Values{1, 2}
	_ = func(x Values) int {
		return x[0] + x[1]
	}(values)
	direct := [2]int{3, 4}
	_ = func(x [2]int) int {
		return x[0] + x[1]
	}(direct)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected blank-discarded immediately invoked function literals: %v", diags)
	}
}

func TestFileAllowsImmediatelyInvokedFunctionLiteralClosures(t *testing.T) {
	file, err := parse.FileSource("function_literal_call_closure.go", []byte(`package main

func appMain() int {
	x := 1
	text := "PASS"
	value := func(extra int) int {
		x = x + extra
		return x + len(text)
	}(2)
	shadow := func() int {
		x := x + 1
		return x
	}()
	return value + x + shadow - 14
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected immediately invoked closure: %v", diags)
	}
}

func TestFileRejectsFunctionAliasMismatchesAndValueUse(t *testing.T) {
	file, err := parse.FileSource("function_alias_mismatch.go", []byte(`package main

func add(a int) int { return a + 1 }
func join(a int, b int) int { return a + b }

func appMain() int {
	f := add
	g := join
	_ = f("bad")
	_ = g(1)
	_ = f
	value := f
	_ = value
	f = add
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid static function alias uses")
	}
	msg := diags.Error()
	for _, want := range []string{
		"function_alias_mismatch.go:9:8: argument type mismatch in call to f: want int, got string",
		"function_alias_mismatch.go:10:6: argument count mismatch in call to g",
		"function_alias_mismatch.go:12:11: function values are not supported: f",
		"function_alias_mismatch.go:14:2: function alias cannot be reassigned: f",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsDiscardedFunctionValues(t *testing.T) {
	file, err := parse.FileSource("discarded_function_values.go", []byte(`package main

func add(a int) int { return a + 1 }

func appMain() int {
	f := add
	_ = add
	_ = f
	_ = func() int { return 1 }
	x := 1
	_ = func() int { return x }
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected discarded function values: %v", diags)
	}
}

func TestFileRejectsTopLevelFunctionValues(t *testing.T) {
	file, err := parse.FileSource("function_values.go", []byte(`package main

func add(a int) int { return a + 1 }

var global = add

func appMain() int {
	print(add)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted function values")
	}
	msg := diags.Error()
	for _, want := range []string{
		"function_values.go:5:14: function values are not supported: add",
		"function_values.go:8:8: function values are not supported: add",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsCompositeFieldNamesMatchingFunctions(t *testing.T) {
	file, err := parse.FileSource("function_named_fields.go", []byte(`package main

type Unit struct {
	Package string
}

type Artifact struct {
	Source []byte
}

func Package() int { return 1 }
func Source() int { return 2 }

func appMain() int {
	_ = Unit{Package: "main"}
	_ = Artifact{
		Source: []byte{'x'},
	}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected composite fields matching function names: %v", diags)
	}
}

func TestFileAllowsInertNamedFunctionTypes(t *testing.T) {
	file, err := parse.FileSource("function_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert named function types: %v", diags)
	}
}

func TestFileAllowsInertFunctionContainingTypes(t *testing.T) {
	file, err := parse.FileSource("function_containing_types.go", []byte(`package main

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
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected inert function-containing types: %v", diags)
	}
}

func TestFileRejectsNamedFunctionTypeValues(t *testing.T) {
	file, err := parse.FileSource("function_types.go", []byte(`package main

type F func(int) int

func appMain() int {
	var f F
	_ = f
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named function type value")
	}
	if !strings.Contains(diags.Error(), "function_types.go:3:8: function values and function types are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsFunctionContainingTypeValues(t *testing.T) {
	file, err := parse.FileSource("function_containing_type_values.go", []byte(`package main

type Box struct { Fn func() int }
type Callbacks []func(int) int

func appMain() int {
	_ = Box{}
	_ = Callbacks{}
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted function-containing type values")
	}
	msg := diags.Error()
	if !strings.Contains(msg, "function values and function types are not supported") {
		t.Fatalf("missing function diagnostic in:\n%s", msg)
	}
}

func TestFileRejectsLocalNamedFunctionTypeValues(t *testing.T) {
	file, err := parse.FileSource("local_function_types.go", []byte(`package main

func appMain() int {
	type Local func() int
	var f Local
	_ = f
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted local named function type value")
	}
	if !strings.Contains(diags.Error(), "local_function_types.go:4:13: function values and function types are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsMixedGroupedFunctionTypeDeclarations(t *testing.T) {
	file, err := parse.FileSource("mixed_function_types.go", []byte(`package main

type (
	F func() int
	Count int
)

func appMain() int {
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted mixed grouped function type declaration")
	}
	if !strings.Contains(diags.Error(), "function values and function types are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileAllowsStaticCallbackForms(t *testing.T) {
	file, err := parse.FileSource("function_forms.go", []byte(`package main

func apply(f func(int, int) int, value int) int { return f(value, 1) }
func add(x int, y int) int { return x + y }

func appMain() int {
	_ = apply(add, 1)
	f := add
	_ = apply(f, 2)
	_ = apply(func(a int, b int) int { return a + b }, 3)
	offset := 4
	_ = apply(func(a int, b int) int { offset = offset + a; return offset + b }, 5)
	literal := func(a int, b int) int { return a * b }
	_ = apply(literal, 6)
	captured := func(a int, b int) int { offset = offset + a; return offset + b }
	_ = apply(captured, 7)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static callback forms: %v", diags)
	}
}

func TestFileAllowsNamedFunctionTypeStaticCallbacks(t *testing.T) {
	file, err := parse.FileSource("named_function_callback.go", []byte(`package main

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
	_ = apply(inc, 1)
	f := inc
	_ = apply(f, 2)
	_ = applyBinary(add, 3)
	_ = applyAlias(inc, 4)
	_ = applyBinaryAlias(add, 5)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected named function type static callbacks: %v", diags)
	}
}

func TestFunctionSignaturesRecognizeNamedFunctionTypeCallbacks(t *testing.T) {
	file, err := parse.FileSource("named_function_callback.go", []byte(`package main

type Unary func(int) int
type AlsoUnary Unary

func apply(f AlsoUnary, value int) int { return f(value) }
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	typeNames := fileNamedTypeUnderlyings(file)
	sigs := functionSignaturesWithTypes(file, nil, typeNames)
	index := funcSignatureIndex(sigs, "apply")
	if index < 0 {
		t.Fatalf("apply signature missing from %#v", sigs)
	}
	if len(sigs[index].callbackParams) != 1 {
		t.Fatalf("callback params = %#v, typeNames = %#v", sigs[index].callbackParams, typeNames)
	}
	callback := sigs[index].callbackParams[0]
	if callback.name != "f" || callback.sig.params != 1 || callback.sig.results != 1 || len(callback.sig.paramTypes) != 1 || callback.sig.paramTypes[0] != "int" || len(callback.sig.resultTypes) != 1 || callback.sig.resultTypes[0] != "int" {
		t.Fatalf("callback signature = %#v", callback)
	}
}

func TestFileRejectsNamedFunctionTypeAliasValues(t *testing.T) {
	file, err := parse.FileSource("named_function_alias_values.go", []byte(`package main

type Unary func(int) int
type AlsoUnary Unary

func appMain() int {
	var f AlsoUnary
	_ = f
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named function type alias value")
	}
	if !strings.Contains(diags.Error(), "function values and function types are not supported") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsNamedFunctionTypeCallbackMismatches(t *testing.T) {
	file, err := parse.FileSource("named_function_callback_mismatch.go", []byte(`package main

type Unary func(int) int

func apply(f Unary) int { return f(1) }
func bad(text string) int { return len(text) }

func appMain() int {
	_ = apply(bad)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted named function type callback mismatch")
	}
	if !strings.Contains(diags.Error(), "named_function_callback_mismatch.go:9:12: callback signature mismatch in call to apply") {
		t.Fatalf("error = %q", diags)
	}
}

func TestFileRejectsUnsupportedCallbackForms(t *testing.T) {
	file, err := parse.FileSource("function_forms.go", []byte(`package main

func apply(f func(int) int) int { return f(1) }
func inc(x int) int { return x + 1 }
func bad(text string) int { return len(text) }

func appMain() int {
	g := inc
	h := g
	_ = apply(h)
	_ = apply(bad)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported callback forms")
	}
	msg := diags.Error()
	for _, want := range []string{
		"function_forms.go:10:12: callback argument in call to apply must be a static function",
		"function_forms.go:11:12: callback signature mismatch in call to apply",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "call of non-function") {
		t.Fatalf("unsupported function values produced call cascade:\n%s", msg)
	}
	if strings.Contains(msg, "return value count mismatch") {
		t.Fatalf("unsupported function literal produced return-count cascade:\n%s", msg)
	}
}

func TestFileAllowsDirectMethodExpressionCalls(t *testing.T) {
	file, err := parse.FileSource("method_expression_calls.go", []byte(`package main

type box struct { value int }
func (b box) Value() int { return b.value }
func (b box) Add(x int) int { return b.value + x }

func appMain() int {
	return box.Value(box{value: 1}) + box.Add(box{value: 2}, 3) - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) > 0 {
		t.Fatalf("File rejected direct method expression calls: %v", diags)
	}
}

func TestFileRejectsMethodExpressionCallMismatches(t *testing.T) {
	file, err := parse.FileSource("method_expression_mismatch.go", []byte(`package main

type box struct { value int }
func (b box) Add(x int, text string) int { return b.value + x + len(text) }

func appMain() int {
	_ = box.Add(box{value: 1}, "bad", 2)
	_ = box.Add(box{value: 1}, 2)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid direct method expression calls")
	}
	msg := diags.Error()
	for _, want := range []string{
		"method_expression_mismatch.go:7:29: argument type mismatch in call to box.Add: want int, got string",
		"method_expression_mismatch.go:8:10: argument count mismatch in call to box.Add",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsStaticMethodExpressionAliases(t *testing.T) {
	file, err := parse.FileSource("method_expression_aliases.go", []byte(`package main

type box struct { value int }
func (b box) Value() int { return b.value }
func (b box) Add(x int) int { return b.value + x }

func appMain() int {
	f := box.Value
	var g = box.Add
	_ = box.Value
	_ = f
	return f(box{value: 1}) + g(box{value: 2}, 3) - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static method expression aliases: %v", diags)
	}
}

func TestFileAllowsStaticMethodExpressionCallbacks(t *testing.T) {
	file, err := parse.FileSource("method_expression_callbacks.go", []byte(`package main

type box struct { value int }
func (b box) Add(x int) int { return b.value + x }

func apply(f func(box, int) int, b box) int { return f(b, 1) }

func appMain() int {
	f := box.Add
	_ = apply(box.Add, box{value: 1})
	_ = apply(f, box{value: 2})
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static method expression callbacks: %v", diags)
	}
}

func TestFileRejectsMethodExpressionAliasMismatchesAndValueUse(t *testing.T) {
	file, err := parse.FileSource("method_expression_alias_mismatch.go", []byte(`package main

type box struct { value int }
func (b box) Add(x int) int { return b.value + x }

func appMain() int {
	f := box.Add
	_ = f(box{value: 1}, "bad")
	value := f
	_ = value
	f = box.Add
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid method expression alias uses")
	}
	msg := diags.Error()
	for _, want := range []string{
		"method_expression_alias_mismatch.go:8:23: argument type mismatch in call to f: want int, got string",
		"method_expression_alias_mismatch.go:9:11: function values are not supported: f",
		"method_expression_alias_mismatch.go:11:2: function alias cannot be reassigned: f",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "call of non-function: f") {
		t.Fatalf("method expression alias mismatch produced call cascade:\n%s", msg)
	}
}

func TestFileAllowsStaticMethodValueAliases(t *testing.T) {
	file, err := parse.FileSource("method_value_aliases.go", []byte(`package main

type box struct { value int }
func (b box) Value() int { return b.value }
func (b *box) Add(v int) int { b.value = b.value + v; return b.value }
func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func appMain() int {
	b := box{value: 1}
	f := b.Value
	g := b.Add
	_ = b.Value
	_ = b.Add
	_ = f
	_ = applyValue(f)
	_ = applyAdd(g, 40)
	_ = applyValue(b.Value)
	_ = applyAdd(b.Add, 1)
	return f() + g(40) - 42
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static method value aliases: %v", diags)
	}
}

func TestFileAllowsStaticPromotedMethodValueAliases(t *testing.T) {
	file, err := parse.FileSource("promoted_method_value_aliases.go", []byte(`package main

type Inner struct { value int }
func (in Inner) Value() int { return in.value }
func (in *Inner) Add(v int) int { in.value = in.value + v; return in.value }

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func appMain() int {
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
	_ = applyValue(outer.Value)
	_ = applyAdd(outer.Add, 1)
	_ = applyValue(pointer.Value)
	_ = applyAdd(pointer.Add, 1)
	return f() + g(2) + h() + k(5) - 29
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static promoted method value aliases: %v", diags)
	}
}

func TestFileAllowsStaticCompositeLiteralMethodValueAliases(t *testing.T) {
	file, err := parse.FileSource("composite_method_value_aliases.go", []byte(`package main

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

func appMain() int {
	f := Box{value: 1}.Value
	g := (&Box{value: 2}).Add
	h := Outer{Inner: Inner{value: 3}}.InnerValue
	k := PointerOuter{Inner: &Inner{value: 4}}.InnerAdd
	m := (&Outer{Inner: Inner{value: 5}}).InnerAdd
	_ = Box{value: 9}.Value
	_ = (&Box{value: 10}).Add
	_ = Outer{Inner: Inner{value: 11}}.InnerValue
	_ = f
	_ = applyValue(f)
	_ = applyAdd(g, 3)
	_ = applyValue(Box{value: 12}.Value)
	_ = applyAdd((&Box{value: 13}).Add, 4)
	_ = applyValue(Outer{Inner: Inner{value: 12}}.InnerValue)
	_ = applyAdd((&Outer{Inner: Inner{value: 13}}).InnerAdd, 4)
	_ = applyValue(PointerOuter{Inner: &Inner{value: 14}}.InnerValue)
	_ = applyAdd(PointerOuter{Inner: &Inner{value: 15}}.InnerAdd, 5)
	return f() + g(3) + h() + k(6) + m(7) - 31
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected static composite literal method value aliases: %v", diags)
	}
}

func TestFileRejectsMethodValueUse(t *testing.T) {
	file, err := parse.FileSource("method_values.go", []byte(`package main

type box struct { value int }
func (b box) Value() int { return b.value }

func appMain() int {
	b := box{value: 1}
	f := b.Value
	value := f
	_ = value
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted unsupported method value use")
	}
	if !strings.Contains(diags.Error(), "method_values.go:9:11: function values are not supported: f") {
		t.Fatalf("error = %q", diags)
	}
	if strings.Contains(diags.Error(), "call of non-function: f") {
		t.Fatalf("method value alias produced call cascade:\n%s", diags.Error())
	}
}

func TestFileRejectsPromotedMethodValueUse(t *testing.T) {
	file, err := parse.FileSource("promoted_method_values.go", []byte(`package main

type Inner struct { value int }
func (in Inner) Value() int { return in.value }
type Outer struct { Inner }

func appMain() int {
	outer := Outer{Inner: Inner{value: 1}}
	f := outer.Value
	value := f
	_ = value
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted promoted method value use")
	}
	if !strings.Contains(diags.Error(), "promoted_method_values.go:10:11: function values are not supported: f") {
		t.Fatalf("error = %q", diags)
	}
}

func TestGraphAllowsImportedStaticFunctionAliases(t *testing.T) {
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

func appMain() int {
	f := dep.Add
	return f(1) - 2
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

func Add(x int) int { return x + 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported static function alias: %v", err)
	}
}

func TestGraphRejectsImportedFunctionValues(t *testing.T) {
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

var global = dep.Add

func appMain() int {
	f := dep.Add
	_ = f
	_ = dep.Add
	value := f
	_ = value
	print(dep.Add)
	return dep.Add(1) - 2
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

func Add(x int) int { return x + 1 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported function values")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:5:18: function values are not supported: dep.Add",
		"main.go:11:11: function values are not supported: f",
		"main.go:13:12: function values are not supported: dep.Add",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "main.go:14") {
		t.Fatalf("imported function call was diagnosed unexpectedly:\n%s", msg)
	}
}

func TestGraphAllowsImportedStaticCallbackForms(t *testing.T) {
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

func apply(f func(int, int) int, value int) int { return f(value, 1) }

func appMain() int {
	_ = apply(dep.Add, 2)
	f := dep.Add
	_ = apply(f, 3)
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

func Add(x int, y int) int { return x + y }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported static callback forms: %v", err)
	}
}

func TestGraphAllowsImportedNamedFunctionTypeStaticCallbackWrappers(t *testing.T) {
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

func appMain() int {
	_ = dep.Apply(dep.Inc, 2)
	_ = dep.ApplyAlias(dep.Inc, 3)
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

type Unary func(int) int
type AlsoUnary Unary

func Apply(f Unary, value int) int { return f(Bias(value)) }
func ApplyAlias(f AlsoUnary, value int) int { return f(Bias(value)) }
func Bias(x int) int { return x + 1 }
func Inc(x int) int { return x + 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported named function type static callback wrappers: %v", err)
	}
}

func TestGraphAllowsImportedDirectMethodExpressionCalls(t *testing.T) {
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

func appMain() int {
	return dep.Buffer.Len(dep.Buffer{}) - 1
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported direct method expression call: %v", err)
	}
}

func TestGraphAllowsImportedStaticMethodExpressionAliases(t *testing.T) {
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

func appMain() int {
	f := dep.Buffer.Len
	_ = dep.Buffer.Len
	_ = f
	return f(dep.Buffer{}) - 1
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported static method expression alias: %v", err)
	}
}

func TestGraphAllowsImportedStaticMethodExpressionCallbacks(t *testing.T) {
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

func apply(f func(dep.Buffer, int) int, b dep.Buffer) int { return f(b, 1) }

func appMain() int {
	f := dep.Buffer.Add
	_ = apply(dep.Buffer.Add, dep.Buffer{Value: 1})
	_ = apply(f, dep.Buffer{Value: 2})
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct { Value int }

func (b Buffer) Add(x int) int { return b.Value + x }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported static method expression callbacks: %v", err)
	}
}

func TestGraphRejectsImportedMethodExpressionAliasValueUse(t *testing.T) {
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

func appMain() int {
	f := dep.Buffer.Len
	value := f
	_ = value
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int { return 1 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported method expression alias value use")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:7:11: function values are not supported: f",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsLocalMethodCallMismatches(t *testing.T) {
	file, err := parse.FileSource("method_calls.go", []byte(`package main

type box struct { value int }

func (b box) Add(x int, s string) int {
	return b.value + x + len(s)
}

func appMain() int {
	b := box{value: 1}
	_ = b.Add(1)
	_ = b.Add("bad", 1)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted method call mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"argument count mismatch in call to b.Add",
		"argument type mismatch in call to b.Add: want int, got string",
		"argument type mismatch in call to b.Add: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsLocalMethodCallTypeChecks(t *testing.T) {
	file, err := parse.FileSource("method_calls_ok.go", []byte(`package main

type box struct { value int }

func (b box) Add(x int, s string) int {
	return b.value + x + len(s)
}

func appMain() int {
	b := box{value: 1}
	return b.Add(2, "ok") - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid method call: %v", diags)
	}
}

func TestFileAllowsValueReceiverMethodCallsThroughNamedSlicePointers(t *testing.T) {
	file, err := parse.FileSource("named_slice_pointer_method.go", []byte(`package main

type Items []int

func (items Items) Add(v int) Items {
	return append(items, v)
}

func Fill(items *Items) {
	*items = items.Add(5)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected value-receiver method call through named slice pointer: %v", diags)
	}
}

func TestFileRejectsIndexedReceiverMethodCallMismatches(t *testing.T) {
	file, err := parse.FileSource("indexed_method_calls.go", []byte(`package main

type box struct { value int }

func (b box) Add(x int, s string) int {
	return b.value + x + len(s)
}

func wantString(s string) string { return s }

func appMain() int {
	items := []box{{value: 1}}
	_ = items[0].Add(1)
	_ = items[0].Add("bad", 1)
	_ = wantString(items[0].Add(1, "ok"))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted indexed receiver method call mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"argument count mismatch in call to items[0].Add",
		"argument type mismatch in call to items[0].Add: want int, got string",
		"argument type mismatch in call to items[0].Add: want string, got int",
		"argument type mismatch in call to wantString: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "expression statement must be a call") {
		t.Fatalf("indexed method call produced expression statement cascade:\n%s", msg)
	}
}

func TestFileAllowsIndexedReceiverMethodCallTypeChecks(t *testing.T) {
	file, err := parse.FileSource("indexed_method_calls_ok.go", []byte(`package main

type box struct { value int }
type boxes []box

func (b box) Add(x int, s string) int {
	return b.value + x + len(s)
}

func (b *box) Bump(x int) {
	b.value = b.value + x
}

func appMain() int {
	items := []box{{value: 1}}
	items[0].Bump(2)
	return items[0].Add(3, "ok") + []box{{value: 4}}[0].Add(5, "go") + (boxes{{value: 6}})[0].Add(7, "rtg") - 33
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid indexed receiver method calls: %v", diags)
	}
}

func TestFileRejectsPointerMethodCallsOnCompositeLiterals(t *testing.T) {
	file, err := parse.FileSource("composite_pointer_method_calls.go", []byte(`package main

type box struct { value int }

func (b *box) Add(x int) int {
	return b.value + x
}

func appMain() int {
	_ = (box{value: 1}).Add(2)
	box{value: 3}.Add(4)
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted pointer method calls on composite literals")
	}
	msg := err.Error()
	if strings.Count(msg, "cannot call pointer method Add on box") != 2 {
		t.Fatalf("missing pointer receiver composite literal diagnostics in:\n%s", msg)
	}
	if strings.Contains(msg, "expression statement must be a call") {
		t.Fatalf("composite method call produced expression statement cascade:\n%s", msg)
	}
}

func TestFileRejectsCompositeLiteralMethodResultTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("composite_method_results.go", []byte(`package main

type box struct{}

func (b box) Text() string {
	return "bad"
}

func (b box) Add(x int) int {
	return x + 1
}

func wantInt(x int) int { return x }
func wantString(s string) string { return s }

func appMain() int {
	_ = (box{}).Add("bad")
	_ = wantInt((box{}).Text())
	_ = wantString(box{}.Add(1))
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted composite literal method type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"argument type mismatch in call to box.Add: want int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"argument type mismatch in call to wantString: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsCompositeLiteralMethodValues(t *testing.T) {
	file, err := parse.FileSource("composite_method_values.go", []byte(`package main

type box struct{}

func (b box) Text() string {
	return "bad"
}

func appMain() int {
	f := (box{}).Text
	g := box{}.Text
	value := f
	other := g
	_, _ = value, other
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted composite literal method values")
	}
	msg := diags.Error()
	if strings.Count(msg, "function values are not supported") != 2 {
		t.Fatalf("missing composite literal method value storage diagnostics in:\n%s", msg)
	}
}

func TestFileRejectsLocalMethodResultTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("method_results.go", []byte(`package main

type box struct{}

func (b box) Text() string { return "bad" }
func wantInt(x int) int { return x }

func appMain() int {
	b := box{}
	var x int
	x = b.Text()
	_ = wantInt(b.Text())
	return b.Text()
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted local method result type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsLocalMethodResultTypeInference(t *testing.T) {
	file, err := parse.FileSource("method_results_ok.go", []byte(`package main

type box struct{}

func (b box) Text() string { return "ok" }
func wantString(s string) int { return len(s) }

func appMain() int {
	b := box{}
	var s string
	s = b.Text()
	inferred := b.Text()
	return wantString(s) + wantString(inferred)
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected local method result inference: %v", diags)
	}
}

func TestGraphAllowsImportedStaticMethodValueAliases(t *testing.T) {
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

func applyLen(f func() int) int { return f() }

func appMain() int {
	b := dep.Buffer{}
	f := b.Len
	_ = applyLen(f)
	_ = applyLen(b.Len)
	_ = applyLen(dep.Buffer{}.Len)
	return f() - 1
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int {
	return 1
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported static method value alias: %v", err)
	}
}

func TestGraphRejectsImportedMethodValueUse(t *testing.T) {
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

func appMain() int {
	b := dep.Buffer{}
	f := b.Len
	value := f
	_ = value
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int {
	return 0
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported method value use")
	}
	if !strings.Contains(err.Error(), "function values are not supported: f") {
		t.Fatalf("error = %q", err)
	}
}

func TestGraphAllowsImportedMethodCalls(t *testing.T) {
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

func appMain() int {
	b := dep.Buffer{}
	return b.Len()
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Len() int {
	return 0
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported method call: %v", err)
	}
}

func TestGraphRejectsImportedPointerMethodCallsOnCompositeLiterals(t *testing.T) {
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

func appMain() int {
	_ = (dep.Buffer{}).Len()
	dep.Buffer{}.Len()
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b *Buffer) Len() int {
	return 0
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported pointer method calls on composite literals")
	}
	msg := err.Error()
	if strings.Count(msg, "cannot call pointer method Len on dep.Buffer") != 2 {
		t.Fatalf("missing imported pointer receiver composite literal diagnostics in:\n%s", msg)
	}
	if strings.Contains(msg, "expression statement must be a call") {
		t.Fatalf("imported composite method call produced expression statement cascade:\n%s", msg)
	}
}

func TestGraphRejectsImportedMethodCallMismatches(t *testing.T) {
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

func appMain() int {
	b := dep.Buffer{}
	_ = b.Add(1)
	_ = b.Add("bad", 1)
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Add(x int, s string) int {
	return x + len(s)
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported method call mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"argument count mismatch in call to b.Add",
		"argument type mismatch in call to b.Add: want int, got string",
		"argument type mismatch in call to b.Add: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsImportedCompositeLiteralMethodResultTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }
func wantString(s string) string { return s }

func appMain() int {
	_ = (dep.Buffer{}).Add("bad")
	_ = wantInt((dep.Buffer{}).Text())
	_ = wantString(dep.Buffer{}.Add(1))
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Text() string {
	return "bad"
}

func (b Buffer) Add(x int) int {
	return x + 1
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported composite literal method type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"argument type mismatch in call to dep.Buffer.Add: want int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"argument type mismatch in call to wantString: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsImportedCompositeLiteralMethodValues(t *testing.T) {
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

func appMain() int {
	f := (dep.Buffer{}).Text
	g := dep.Buffer{}.Text
	value := f
	other := g
	_, _ = value, other
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Text() string {
	return "bad"
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported composite literal method values")
	}
	msg := err.Error()
	if strings.Count(msg, "function values are not supported") != 2 {
		t.Fatalf("missing imported composite literal method value storage diagnostics in:\n%s", msg)
	}
}

func TestGraphAllowsImportedMethodCallTypeChecks(t *testing.T) {
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

func appMain() int {
	b := dep.Buffer{}
	return b.Add(1, "ok")
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Add(x int, s string) int {
	return x + len(s)
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported method call: %v", err)
	}
}

func TestGraphRejectsImportedMethodResultTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	b := dep.Buffer{}
	var x int
	x = b.Text()
	_ = wantInt(b.Text())
	return b.Text()
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Text() string {
	return "bad"
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported method result type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedMethodResultTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }

func appMain() int {
	b := dep.Buffer{}
	var s string
	s = b.Text()
	inferred := b.Text()
	return wantString(s) + wantString(inferred)
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Text() string {
	return "ok"
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported method result inference: %v", err)
	}
}

func TestGraphRejectsImportedMethodMultiResultTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func values() (int, string) {
	b := dep.Buffer{}
	return b.Pair()
}

func appMain() int {
	b := dep.Buffer{}
	a, _ := b.Pair()
	_ = wantInt(a)
	var x int
	var y string
	x, y = b.Pair()
	return x + len(y)
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
						Path: "buffer.go",
						Source: []byte(`package dep

type Buffer struct{}

func (b Buffer) Pair() (string, int) {
	return "bad", 1
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported method multi-result mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"return type mismatch: want int, got string",
		"return type mismatch: want string, got int",
		"argument type mismatch in call to wantInt: want int, got string",
		"assignment type mismatch: x has int, got string",
		"assignment type mismatch: y has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsLocalShadowOfTopLevelFunctionName(t *testing.T) {
	file, err := parse.FileSource("function_shadow.go", []byte(`package main

func add(a int) int { return a + 1 }

func appMain() int {
	add := 3
	if add == 3 {
		return 0
	}
	return 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	if diags := File(file); len(diags) != 0 {
		t.Fatalf("File rejected local shadow: %v", diags)
	}
}

func TestGraphRejectsDuplicatePackageLevelNames(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package pkg

func Value() int { return 1 }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package pkg

func Value() int { return 2 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with duplicate declaration")
	}
	msg := err.Error()
	if !strings.Contains(msg, "a.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing first duplicate diagnostic in:\n%s", msg)
	}
	if !strings.Contains(msg, "b.go:3:6: duplicate package-level declaration: Value") {
		t.Fatalf("missing second duplicate diagnostic in:\n%s", msg)
	}
}

func TestGraphRejectsDuplicateGroupedPackageLevelNames(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "group.go",
						Source: []byte(`package pkg

const (
	Answer = 1
	Other = 2
)

var (
	Other = 3
)
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with duplicate grouped declaration")
	}
	msg := err.Error()
	for _, want := range []string{
		"group.go:5:2: duplicate package-level declaration: Other",
		"group.go:9:2: duplicate package-level declaration: Other",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsMethodNameMatchingTypeName(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app/pkg",
				Name:       "pkg",
				Files: []load.File{
					{
						Path: "error.go",
						Source: []byte(`package pkg

type Error string

func (e Error) Error() string {
	return string(e)
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected method with same name as type: %v", err)
	}
}

func TestGraphRejectsUnresolvedImportedSelectors(t *testing.T) {
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

func appMain() int {
	return dep.Missing() + dep.hidden()
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

func Value() int { return 1 }
func hidden() int { return 2 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph succeeded with unresolved imported selectors")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:6:13: unresolved imported selector: example.com/app/dep.Missing",
		"main.go:6:29: unresolved imported selector: example.com/app/dep.hidden",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "call of non-function") {
		t.Fatalf("unresolved imported selectors produced call cascade:\n%s", msg)
	}
}

func TestGraphRejectsUnavailableStdSelectors(t *testing.T) {
	root := t.TempDir()
	writeCheckFile(t, root, "go.mod", "module example.com/app\n")
	writeCheckFile(t, root, "main.go", `package main

import (
	"fmt"
	"os"
	"strconv"
)

func appMain() int {
	_ = fmt.Sprintf("x")
	_ = strconv.FormatBool(true)
	_, _ = os.Create("x")
	_, _ = os.OpenFile("x", os.O_RDONLY, 0)
	return 0
}
`)
	stdRoot, err := filepath.Abs(filepath.Join("..", "std"))
	if err != nil {
		t.Fatalf("Abs std root failed: %v", err)
	}
	graph, err := load.LoadEntries([]string{root}, load.Options{StdRoot: stdRoot})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	err = Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted unavailable std selectors")
	}
	msg := err.Error()
	for _, want := range []string{
		"unresolved imported selector: fmt.Sprintf",
		"unresolved imported selector: strconv.FormatBool",
		"unresolved imported selector: os.Create",
		"unresolved imported selector: os.OpenFile",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "assignment count mismatch") {
		t.Fatalf("unavailable std selectors produced assignment cascade:\n%s", msg)
	}
}

func TestGraphAllowsStdBytesConstructors(t *testing.T) {
	root := t.TempDir()
	writeCheckFile(t, root, "go.mod", "module example.com/app\n")
	writeCheckFile(t, root, "main.go", `package main

import "bytes"

func appMain() int {
	buffer := bytes.NewBufferString("PA")
	buffer.WriteByte('S')
	buffer.Write([]byte("S\n"))
	other := bytes.NewBuffer([]byte("OK"))
	return len(buffer.String()) + other.Len() - 7
}
`)
	stdRoot, err := filepath.Abs(filepath.Join("..", "std"))
	if err != nil {
		t.Fatalf("Abs std root failed: %v", err)
	}
	graph, err := load.LoadEntries([]string{root}, load.Options{StdRoot: stdRoot})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected std bytes constructors: %v", err)
	}
}

func TestGraphAllowsStdStringsReplaceAll(t *testing.T) {
	root := t.TempDir()
	writeCheckFile(t, root, "go.mod", "module example.com/app\n")
	writeCheckFile(t, root, "main.go", `package main

import "strings"

func appMain() int {
	_ = strings.ReplaceAll("aba", "a", "b")
	return 0
}
`)
	stdRoot, err := filepath.Abs(filepath.Join("..", "std"))
	if err != nil {
		t.Fatalf("Abs std root failed: %v", err)
	}
	graph, err := load.LoadEntries([]string{root}, load.Options{StdRoot: stdRoot})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected strings.ReplaceAll: %v", err)
	}
}

func TestGraphAllowsUnsafeSizeofFixedOperands(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"unsafe"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import u "unsafe"

type ByteAlias byte
type ByteAlias2 ByteAlias
type BytesAlias []byte
type PtrAlias *byte
type WordAlias int16

var g byte = 1
var ag ByteAlias = 1
var ax BytesAlias = BytesAlias{1, 2}
var ap PtrAlias

func appMain() int {
	var b byte = 1
	type LocalByte ByteAlias
	var lb LocalByte = 1
	xs := []byte{1, 2}
	s := "x"
	p := &b
	_ = u.Sizeof(g)
	_ = u.Sizeof(ag)
	_ = u.Sizeof(ax)
	_ = u.Sizeof(ap)
	_ = u.Sizeof(b)
	_ = u.Sizeof(lb)
	_ = u.Sizeof(xs)
	_ = u.Sizeof(s)
	_ = u.Sizeof(p)
	_ = u.Sizeof(byte(1))
	_ = u.Sizeof(ByteAlias(1))
	_ = u.Sizeof(ByteAlias2(1))
	_ = u.Sizeof(LocalByte(1))
	_ = u.Sizeof(WordAlias(1))
	_ = u.Sizeof(int16(1))
	_ = u.Sizeof(int32(1))
	_ = u.Sizeof(int64(1))
	_ = u.Sizeof(float64(1))
	_ = u.Sizeof(true)
	_ = u.Sizeof("x")
	_ = u.Sizeof(&b)
	return 0
}
`),
					},
				},
			},
			{
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected unsafe.Sizeof fixed operands: %v", err)
	}
}

func TestGraphAllowsUnsafeSizeofTargetWordOperands(t *testing.T) {
	graph := &load.Graph{
		Target: "linux/386",
		Packages: []load.Package{
			{
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
	_ = unsafe.Sizeof(1)
	var x MyInt = 1
	_ = unsafe.Sizeof(x)
	_ = unsafe.Sizeof(int(1))
	_ = unsafe.Sizeof(MyInt(1))
	_ = unsafe.Sizeof(&x)
	_ = unsafe.Sizeof("x")
	_ = unsafe.Sizeof([]byte{1, 2})
	_ = unsafe.Sizeof([3]int{1, 2, 3})
	_ = unsafe.Sizeof([2]string{"a", "b"})
	_ = unsafe.Sizeof([2][3]int{{1, 2, 3}, {4, 5, 6}})
	_ = unsafe.Sizeof(Matrix{})
	return 0
}
`),
					},
				},
			},
			{
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected target word unsafe.Sizeof operands: %v", err)
	}
}

func TestGraphAllowsUnsafeSizeofSimpleStructOperands(t *testing.T) {
	graph := &load.Graph{
		Target: "linux/386",
		Packages: []load.Package{
			{
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
	_ = unsafe.Sizeof(Inner{})
	_ = unsafe.Sizeof(Box{})
	_ = unsafe.Sizeof(Alias{})
	_ = unsafe.Sizeof(box)
	_ = unsafe.Sizeof([2]Box{})
	return 0
}
`),
					},
				},
			},
			{
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
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected simple struct unsafe.Sizeof operands: %v", err)
	}
}

func TestGraphRejectsUnsafeSizeofNonLowerableOperandForms(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"unsafe"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "unsafe"

func value() byte { return 1 }

func appMain() int {
	_ = unsafe.Sizeof(value())
	return 0
}
`),
					},
				},
			},
			{
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
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted non-lowerable unsafe.Sizeof operand")
	}
	if !strings.Contains(err.Error(), "unsafe.Sizeof operand form is not lowerable yet") {
		t.Fatalf("error = %q", err)
	}
}

func TestGraphRejectsImportedFunctionCallMismatches(t *testing.T) {
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

func appMain() int {
	_ = dep.Value(1)
	_ = dep.Value("bad", 1)
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

func Value(x int, s string) int { return x + len(s) }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported call mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:6:10: argument count mismatch in call to dep.Value",
		"main.go:7:16: argument type mismatch in call to dep.Value: want int, got string",
		"main.go:7:23: argument type mismatch in call to dep.Value: want string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedFunctionCalls(t *testing.T) {
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

func appMain() int {
	return dep.Value(1, "ok")
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

func Value(x int, s string) int { return x + len(s) }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported function call: %v", err)
	}
}

func TestGraphAllowsImportedFunctionStructTypes(t *testing.T) {
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

func appMain() int {
	graph := dep.Load(dep.Options{Count: 2}, []dep.Item{{Value: 3}})
	return dep.Score(graph) - 6
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

type Options struct { Count int }
type Item struct { Value int }
type Graph struct {
	First Item
	Items []Item
}

func Load(opts Options, items []Item) *Graph {
	return &Graph{First: items[0], Items: items}
}

func Score(graph *Graph) int {
	return graph.First.Value + graph.Items[0].Value + graph.Items[0].Value - graph.Items[0].Value
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported function struct types: %v", err)
	}
}

func TestGraphAllowsFormattedFmtErrorfLoweringShape(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"fmt"},
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

import "fmt"

func fail(name string, code int) error {
	return fmt.Errorf("bad %s: %d", name, code)
}

func appMain() int {
	if fail("input", 7) == nil {
		return 1
	}
	return 0
}
`),
					},
				},
			},
			{
				ImportPath: "fmt",
				Name:       "fmt",
				Files: []load.File{
					{
						Path: "fmt.go",
						Source: []byte(`package fmt

type Error string

func (err Error) Error() string { return string(err) }
func Errorf(format string) error { return Error(format) }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected fmt.Errorf lowering shape: %v", err)
	}
}

func TestGraphAllowsImportedFunctionNamedScalarTypesAcrossFiles(t *testing.T) {
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

func appMain() int {
	return int(dep.Add(dep.One, dep.Two) - 3)
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
						Path: "types.go",
						Source: []byte(`package dep

type Count int

const One Count = 1
const Two Count = 2
`),
					},
					{
						Path: "funcs.go",
						Source: []byte(`package dep

func Add(x Count, y Count) Count { return x + y }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported named scalar function types: %v", err)
	}
}

func TestGraphRejectsImportedDefinedTypeAssignabilityMismatches(t *testing.T) {
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

func appMain() int {
	i := 1
	var plain int
	count := dep.One
	plain = count
	plain = dep.Default.Count
	return dep.Add(i, dep.Two) + plain
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

type Count int

type Box struct {
	Count Count
}

const One Count = 1
const Two Count = 2
var Default = Box{Count: One}

func Add(x Count, y Count) Count { return x + y }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported defined-type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: plain has int, got dep.Count",
		"argument type mismatch in call to dep.Add: want dep.Count, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "assignment type mismatch: plain has int, got dep.Count"); count != 2 {
		t.Fatalf("got %d imported assignment diagnostics, want 2:\n%s", count, msg)
	}
}

func TestGraphRejectsImportedFunctionResultTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text()
	_ = wantInt(dep.Text())
	return dep.Text()
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

func Text() string { return "bad" }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported result type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedFunctionResultTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }

func appMain() int {
	var s string
	s = dep.Text()
	inferred := dep.Text()
	return wantString(s) + wantString(inferred)
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

func Text() string { return "ok" }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported result type inference: %v", err)
	}
}

func TestGraphRejectsImportedFunctionMultiResultTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func values() (int, string) {
	return dep.Pair()
}

func appMain() int {
	a, _ := dep.Pair()
	_ = wantInt(a)
	var x int
	var y string
	x, y = dep.Pair()
	return x + len(y)
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

func Pair() (string, int) { return "bad", 1 }
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported multi-result mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"return type mismatch: want int, got string",
		"return type mismatch: want string, got int",
		"argument type mismatch in call to wantInt: want int, got string",
		"assignment type mismatch: x has int, got string",
		"assignment type mismatch: y has string, got int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedFunctionMultiResultTypeInference(t *testing.T) {
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

func wantInt(x int) int { return x }
func wantString(s string) int { return len(s) }

func values() (int, string) {
	return dep.Pair()
}

func appMain() int {
	a, b := dep.Pair()
	var x int
	var s string
	x, s = dep.Pair()
	return wantInt(a) + wantString(b) + wantInt(x) + wantString(s) - 6
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

func Pair() (int, string) { return 1, "ok" }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported multi-result inference: %v", err)
	}
}

func TestGraphRejectsImportedValueTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text
	_ = wantInt(dep.Text)
	return dep.Text
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

const Text = "bad"
var Number int = 1
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported value type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedValueTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	s := dep.Text
	n := dep.Number
	return wantString(s) + wantInt(n)
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

const Text = "ok"
var Number = 1
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported value type inference: %v", err)
	}
}

func TestGraphAllowsImportedDerivedValueTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	text := dep.Text
	total := dep.Total
	return wantString(text) + wantInt(total) - 4
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
						Path: "base.go",
						Source: []byte(`package dep

const Prefix = "ok"
const One = 1
`),
					},
					{
						Path: "derived.go",
						Source: []byte(`package dep

const Text = Prefix + "!"
const Total = One + 2
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported derived value type inference: %v", err)
	}
}

func TestGraphRejectsImportedDerivedValueTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text
	_ = wantInt(dep.Text)
	return dep.Text
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
						Path: "base.go",
						Source: []byte(`package dep

const Prefix = "bad"
`),
					},
					{
						Path: "derived.go",
						Source: []byte(`package dep

const Text = Prefix + "!"
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported derived value mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsReexportedImportedValueTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text
	_ = wantInt(dep.Text)
	return dep.Text
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Imports:    []string{"example.com/app/base"},
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

import "example.com/app/base"

const Text = base.Text
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/base",
				Name:       "base",
				Files: []load.File{
					{
						Path: "base.go",
						Source: []byte(`package base

const Text = "bad"
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted re-exported imported value mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsReexportedImportedValueTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	return wantString(dep.Text) + wantInt(dep.Total) - 5
}
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/dep",
				Name:       "dep",
				Imports:    []string{"example.com/app/base"},
				Files: []load.File{
					{
						Path: "dep.go",
						Source: []byte(`package dep

import "example.com/app/base"

const Text = base.Text
const Total = base.Count + 2
`),
					},
				},
			},
			{
				ImportPath: "example.com/app/base",
				Name:       "base",
				Files: []load.File{
					{
						Path: "base.go",
						Source: []byte(`package base

const Text = "ok"
const Count = 3
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected re-exported imported value inference: %v", err)
	}
}

func TestGraphAllowsImportedStructFieldDerivedValueTypes(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	return wantString(dep.Text) + wantInt(dep.Count) - 5
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
						Path: "config.go",
						Source: []byte(`package dep

type config struct {
	Text string
	Count int
}

var settings = config{Text: "ok", Count: 3}
`),
					},
					{
						Path: "derived.go",
						Source: []byte(`package dep

const Text = settings.Text
const Count = settings.Count
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported struct-field derived value types: %v", err)
	}
}

func TestGraphRejectsImportedStructFieldDerivedValueMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text
	_ = wantInt(dep.Text)
	return dep.Text
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
						Path: "config.go",
						Source: []byte(`package dep

type config struct {
	Text string
}

var settings = config{Text: "bad"}
`),
					},
					{
						Path: "derived.go",
						Source: []byte(`package dep

const Text = settings.Text
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported struct-field derived value mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsImportedValueCompositeTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Text + "!"
	_ = wantInt(dep.Text + "!")
	_ = dep.Text - "!"
	return dep.Text + "!"
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

const Text = "bad"
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported value composite type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"invalid operands for -: string and string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphRejectsSelectorsOnImportedNonStructValues(t *testing.T) {
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

func appMain() int {
	_ = dep.Count.value
	_ = dep.Text.value
	_ = dep.Values.value
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

const Count = 1
const Text = "ok"
var Values = []int{1}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted selectors on imported non-struct values")
	}
	msg := err.Error()
	for _, want := range []string{
		"cannot select field on non-struct value: int",
		"cannot select field on non-struct value: string",
		"cannot select field on non-struct value: []int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedTopLevelStructValueSelectors(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	return wantString(dep.Default.Text) + wantInt(dep.Default.Count) - 3
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

type Box struct {
	Text string
	Count int
}

var Default = Box{Text: "ok", Count: 1}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported top-level struct value selectors: %v", err)
	}
}

func TestGraphAllowsImportedPromotedStructSelectors(t *testing.T) {
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

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Text: "ok"}}
	_ = o.Text
	_ = dep.Default.Text
	_ = dep.Outer{Inner: dep.Inner{Text: "literal"}}.Text
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

type Inner struct {
	Text string
}

type Outer struct {
	Inner
}

var Default = Outer{Inner: Inner{Text: "ok"}}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported promoted struct selectors: %v", err)
	}
}

func TestGraphAllowsImportedPromotedMethodCalls(t *testing.T) {
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

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Count: 2}}
	return o.Value() - 2
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

type Inner struct { Count int }

func (in Inner) Value() int {
	return in.Count
}

type Outer struct { Inner }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported promoted method calls: %v", err)
	}
}

func TestGraphRejectsImportedTopLevelStructValueSelectorTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	var x int
	x = dep.Default.Text
	_ = wantInt(dep.Default.Text)
	_ = dep.Default.Missing
	return dep.Default.Text
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

type Box struct {
	Text string
}

var Default = Box{Text: "bad"}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported top-level struct value selector type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: dep.Box.Missing",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsKnownLocalStructSelectors(t *testing.T) {
	file, err := parse.FileSource("selectors.go", []byte(`package main

type box struct { value int }

func read(b box, p *box) int {
	local := box{value: 2}
	return b.value + p.value + local.value
}

func appMain() int {
	return read(box{value: 1}, &box{value: 3}) - 6
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected known local struct selectors: %v", diags)
	}
}

func TestFileRejectsSelectorsOnNonStructValues(t *testing.T) {
	file, err := parse.FileSource("non_struct_selectors.go", []byte(`package main

func appMain() int {
	x := 1
	_ = x.value
	text := "hello"
	_ = text.value
	values := []int{1}
	_ = values.value
	p := &x
	_ = p.value
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted selectors on non-struct values")
	}
	msg := err.Error()
	for _, want := range []string{
		"cannot select field on non-struct value: int",
		"cannot select field on non-struct value: string",
		"cannot select field on non-struct value: []int",
		"cannot select field on non-struct value: *int",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileRejectsLocalStructFieldTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("selector_types.go", []byte(`package main

type box struct {
	text string
	count int
}

func wantInt(x int) int { return x }

func appMain() int {
	b := box{text: "bad", count: 1}
	var x int
	x = b.text
	_ = wantInt(b.text)
	return b.text
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted local struct field type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsLocalStructFieldTypeInference(t *testing.T) {
	file, err := parse.FileSource("selector_types_ok.go", []byte(`package main

type box struct {
	text string
	count int
	ok bool
}

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }
func wantBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func appMain() int {
	b := box{text: "ok", count: 1, ok: true}
	text := b.text
	count := b.count
	ok := b.ok
	return wantString(text) + wantInt(count) + wantBool(ok) - 4
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected local struct field type inference: %v", diags)
	}
}

func TestFileRejectsNestedLocalStructFieldTypeMismatches(t *testing.T) {
	file, err := parse.FileSource("nested_selector_types.go", []byte(`package main

type inner struct {
	text string
	count int
}

type outer struct {
	inner inner
	ptr *inner
}

func wantInt(x int) int { return x }

func appMain() int {
	o := outer{inner: inner{text: "bad", count: 1}, ptr: &inner{text: "ptr", count: 2}}
	nested := o.inner
	var x int
	x = o.inner.text
	_ = wantInt(o.ptr.text)
	_ = wantInt(nested.text)
	_ = o.inner.missing
	return o.inner.text
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	err = File(file)
	if err == nil {
		t.Fatalf("File accepted nested local struct field mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: inner.missing",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestFileAllowsNestedLocalStructFieldTypeInference(t *testing.T) {
	file, err := parse.FileSource("nested_selector_types_ok.go", []byte(`package main

type inner struct {
	text string
	count int
}

type outer struct {
	inner inner
	ptr *inner
}

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	o := outer{inner: inner{text: "ok", count: 1}, ptr: &inner{text: "ptr", count: 2}}
	nested := o.inner
	text := nested.text
	ptrText := o.ptr.text
	count := o.inner.count
	return wantString(text) + wantString(ptrText) + wantInt(count) - 5
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected nested local struct field inference: %v", diags)
	}
}

func TestGraphRejectsUnknownLocalStructSelectors(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "types.go",
						Source: []byte(`package main

type box struct { value int }
`),
					},
					{
						Path: "main.go",
						Source: []byte(`package main

func appMain() int {
	b := box{value: 1}
	return b.missing
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted unknown local struct selector")
	}
	if !strings.Contains(err.Error(), "main.go:5:11: unknown field: box.missing") {
		t.Fatalf("error = %q", err)
	}
}

func TestGraphRejectsImportedStructFieldTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	b := dep.Box{Text: "bad", Count: 1}
	var x int
	x = b.Text
	_ = wantInt(b.Text)
	_ = b.hidden
	return b.Text
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
						Path: "box.go",
						Source: []byte(`package dep

type Box struct {
	Text string
	Count int
	hidden int
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted imported struct field type mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: dep.Box.hidden",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsImportedStructFieldTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	b := dep.Box{Text: "ok", Count: 1}
	text := b.Text
	count := b.Count
	return wantString(text) + wantInt(count) - 3
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
						Path: "box.go",
						Source: []byte(`package dep

type Box struct {
	Text string
	Count int
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected imported struct field type inference: %v", err)
	}
}

func TestGraphRejectsNestedImportedStructFieldTypeMismatches(t *testing.T) {
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

func wantInt(x int) int { return x }

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Text: "bad", Count: 1}, Ptr: &dep.Inner{Text: "ptr", Count: 2}}
	nested := o.Inner
	var x int
	x = o.Inner.Text
	_ = wantInt(o.Ptr.Text)
	_ = wantInt(nested.Text)
	_ = o.Inner.hidden
	return o.Inner.Text
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
						Path: "types.go",
						Source: []byte(`package dep

type Inner struct {
	Text string
	Count int
	hidden int
}

type Outer struct {
	Inner Inner
	Ptr *Inner
}
`),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted nested imported struct field mismatches")
	}
	msg := err.Error()
	for _, want := range []string{
		"assignment type mismatch: x has int, got string",
		"argument type mismatch in call to wantInt: want int, got string",
		"unknown field: dep.Inner.hidden",
		"return type mismatch: want int, got string",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
}

func TestGraphAllowsNestedImportedStructFieldTypeInference(t *testing.T) {
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

func wantString(s string) int { return len(s) }
func wantInt(x int) int { return x }

func appMain() int {
	o := dep.Outer{Inner: dep.Inner{Text: "ok", Count: 1}, Ptr: &dep.Inner{Text: "ptr", Count: 2}}
	nested := o.Inner
	text := nested.Text
	ptrText := o.Ptr.Text
	count := o.Inner.Count
	return wantString(text) + wantString(ptrText) + wantInt(count) - 5
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
						Path: "types.go",
						Source: []byte(`package dep

type Inner struct {
	Text string
	Count int
}

type Outer struct {
	Inner Inner
	Ptr *Inner
}
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected nested imported struct field inference: %v", err)
	}
}

func TestFileRejectsInvalidLocalStructIndexAndDeref(t *testing.T) {
	file, err := parse.FileSource("struct_ops.go", []byte(`package main

type box struct { value int }

func use(b box, p *box) int {
	_ = b[0]
	_ = p[0]
	return *b
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid struct operations")
	}
	msg := diags.Error()
	for _, want := range []string{
		"struct_ops.go:6:7: cannot index struct value: box",
		"struct_ops.go:7:7: cannot index struct value: *box",
		"struct_ops.go:8:9: cannot dereference non-pointer: box",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "cannot dereference non-pointer: box"); count != 1 {
		t.Fatalf("got %d struct dereference diagnostics, want 1:\n%s", count, msg)
	}
}

func TestFileRejectsInvalidScalarIndexAndDeref(t *testing.T) {
	file, err := parse.FileSource("scalar_ops.go", []byte(`package main

func appMain() int {
	x := 1
	_ = x[0]
	_ = *x
	_ = *1
	_ = *(x + 1)
	_ = *true
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) == 0 {
		t.Fatalf("File accepted invalid scalar operations")
	}
	msg := diags.Error()
	for _, want := range []string{
		"cannot index non-indexable value: int",
		"cannot dereference non-pointer: int",
		"cannot dereference non-pointer: bool",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if count := strings.Count(msg, "cannot dereference non-pointer: int"); count != 3 {
		t.Fatalf("got %d int dereference diagnostics, want 3:\n%s", count, msg)
	}
}

func TestFileAllowsPointerDerefAndIndexableValues(t *testing.T) {
	file, err := parse.FileSource("scalar_ops_ok.go", []byte(`package main

func appMain() int {
	x := 1
	p := &x
	values := []int{2, 3}
	text := "ab"
	_ = *p
	_ = values[0]
	_ = values[0:1]
	_ = text[0]
	_ = text[0:1]
	return 0
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected valid pointer/index operations: %v", diags)
	}
}

func TestFileAllowsPointerFieldSelectorDeref(t *testing.T) {
	file, err := parse.FileSource("selector_deref_ok.go", []byte(`package main

type intPtr *int

type box struct {
	value *int
}

func appMain() int {
	x := 1
	b := box{value: intPtr(&x)}
	return *b.value - 1
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected pointer field selector dereference: %v", diags)
	}
}

func TestFileAllowsLocalStructPointerDeref(t *testing.T) {
	file, err := parse.FileSource("struct_deref.go", []byte(`package main

type box struct { value int }

func use(p *box) int {
	v := *p
	return v.value
}
`))
	if err != nil {
		t.Fatalf("FileSource failed: %v", err)
	}
	diags := File(file)
	if len(diags) != 0 {
		t.Fatalf("File rejected pointer dereference: %v", diags)
	}
}

func TestGraphIgnoresLocalShadowedImportSelector(t *testing.T) {
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

type localDep struct { hidden int }

func useImport() int {
	return dep.Value()
}

func appMain() int {
	dep := localDep{hidden: 1}
	return dep.hidden + useImport()
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

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected local shadowed import selector: %v", err)
	}
}

func TestGraphUsesFileScopedImportAliases(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package main

import first "example.com/app/dep"

func A() int { return first.Value() }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package main

import second "example.com/app/dep"

func appMain() int { return A() + second.Value() }
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

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected file-scoped import aliases: %v", err)
	}
}

func TestGraphDoesNotLeakImportAliasesAcrossFiles(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Imports:    []string{"example.com/app/dep"},
				Files: []load.File{
					{
						Path: "a.go",
						Source: []byte(`package main

import dep "example.com/app/dep"

func A() int { return dep.Value() }
`),
					},
					{
						Path: "b.go",
						Source: []byte(`package main

type localDep struct { hidden int }

func appMain() int {
	dep := localDep{hidden: 1}
	return dep.hidden + A()
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

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected local selector through leaked import alias: %v", err)
	}
}

func TestGraphAcceptsExportedImportedSelectors(t *testing.T) {
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

func appMain() int {
	return dep.Value()
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

func Value() int { return 1 }
`),
					},
				},
			},
		},
	}
	if err := Graph(graph); err != nil {
		t.Fatalf("Graph rejected exported selector: %v", err)
	}
}

func TestGraphPreservesParseDiagnosticPositions(t *testing.T) {
	graph := &load.Graph{
		Packages: []load.Package{
			{
				ImportPath: "example.com/app",
				Name:       "main",
				Files: []load.File{
					{
						Path: "main.go",
						Source: []byte(`package main

bad := 1
`),
					},
					{
						Path:   "broken.go",
						Source: []byte("package main\n\nfunc appMain() int { print(\"unterminated)\n"),
					},
				},
			},
		},
	}
	err := Graph(graph)
	if err == nil {
		t.Fatalf("Graph accepted parse errors")
	}
	msg := err.Error()
	for _, want := range []string{
		"main.go:3:1: expected top-level declaration",
		"broken.go:3:28: unterminated literal",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing diagnostic %q in:\n%s", want, msg)
		}
	}
	if strings.Contains(msg, "main.go:1:1:") || strings.Contains(msg, "broken.go:1:1:") {
		t.Fatalf("parse diagnostic was wrapped at 1:1:\n%s", msg)
	}
}

func messages(diags Diagnostics) []string {
	var out []string
	for _, diag := range diags {
		out = append(out, diag.Message)
	}
	return out
}
