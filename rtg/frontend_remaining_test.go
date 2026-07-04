package rtg_test

import (
	"os/exec"
	"strings"
	"testing"
)

type frontendRemainingFixture struct {
	name   string
	reason string
	files  map[string]string
}

func TestFrontendRemainingFeatureFixturesMatchHostGo(t *testing.T) {
	for _, tc := range frontendRemainingFixtures() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fixture := buildFrontendRemainingFixture(t, tc)
			cmd := exec.Command("go", "run", "./cmd/app")
			cmd.Dir = fixture
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("host fixture failed: %v\n%s", err, string(out))
			}
			if string(out) != "PASS\n" {
				t.Fatalf("host fixture output = %q, want PASS", string(out))
			}
		})
	}
}

func TestFrontendRemainingFeatureBacklog(t *testing.T) {
	for _, tc := range frontendRemainingFixtures() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Skipf("frontend TODO: %s", tc.reason)
			fixture := buildFrontendRemainingFixture(t, tc)
			runFrontendFixtureMatchesHostGo(t, fixture)
		})
	}
}

func buildFrontendRemainingFixture(t *testing.T, tc frontendRemainingFixture) string {
	t.Helper()
	root := t.TempDir()
	modulePath := frontendRemainingModulePath(tc.name)
	writeFixtureFile(t, root, "go.mod", "module "+modulePath+"\n\ngo 1.20\n")
	for path, data := range tc.files {
		writeFixtureFile(t, root, path, strings.ReplaceAll(data, "{{MODULE}}", modulePath))
	}
	return root
}

func frontendRemainingModulePath(name string) string {
	var out strings.Builder
	out.WriteString("example.com/frontendremaining/")
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			out.WriteRune(r)
			continue
		}
		if r >= '0' && r <= '9' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func frontendRemainingFixtures() []frontendRemainingFixture {
	return []frontendRemainingFixture{
		{
			name:   "dynamic_map_storage",
			reason: "value-carrying maps need dynamic storage, mutation, lookup, delete, len, and range",
			files: map[string]string{
				"cmd/app/main.go": `package main

func main() {
	m := make(map[string]int)
	m["a"] = 1
	key := "b"
	m[key] = 2
	m["a"]++
	delete(m, "missing")
	total := len(m) + m["a"] + m["b"]
	if v, ok := m[key]; ok {
		total += v
	}
	for k, v := range m {
		total += len(k) + v
	}
	delete(m, "a")
	_, gone := m["a"]
	if total == 14 && len(m) == 1 && !gone {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "named_map_values",
			reason: "named map values need real storage across calls and returns",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Table map[string]int

func fill(t Table) {
	t["a"] = 1
	t["b"] = 2
}

func total(t Table) int {
	sum := 0
	for _, v := range t {
		sum += v
	}
	return sum
}

func makeTable() Table {
	return Table{"z": 3}
}

func main() {
	t := Table{}
	fill(t)
	u := makeTable()
	t["z"] = u["z"]
	if total(t) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "map_fields_and_slices",
			reason: "map-containing values need storage when maps live in structs and slices",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Box struct {
	Values map[string]int
	Rows   []map[string]int
}

func main() {
	b := Box{
		Values: map[string]int{"a": 1},
		Rows:   []map[string]int{{"x": 2}},
	}
	b.Values["b"] = 3
	b.Rows[0]["x"]++
	if b.Values["a"]+b.Values["b"]+b.Rows[0]["x"] == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "interface_dynamic_dispatch",
			reason: "value-carrying interfaces need method tables and dynamic dispatch",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Text interface {
	Text() string
}

type Label struct {
	value string
}

func (l Label) Text() string {
	return l.value
}

func read(v Text) string {
	return v.Text()
}

func main() {
	var v Text = Label{value: "PASS\n"}
	print(read(v))
}
`,
			},
		},
		{
			name:   "interface_params_returns",
			reason: "used interface parameters and returns need a real value representation",
			files: map[string]string{
				"cmd/app/main.go": `package main

func identity(v any) any {
	return v
}

func choose(ok bool) any {
	if ok {
		return "PASS\n"
	}
	return 0
}

func main() {
	v := identity(choose(true))
	if text, ok := v.(string); ok {
		print(text)
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "dynamic_type_switch",
			reason: "dynamic assertions and type switches need runtime type identity",
			files: map[string]string{
				"cmd/app/main.go": `package main

func score(v any) int {
	switch x := v.(type) {
	case int:
		return x + 1
	case string:
		return len(x)
	default:
		return 0
	}
}

func main() {
	if score(4)+score("abc") == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "nested_array_values",
			reason: "nested fixed arrays need value storage, copying, and comparisons",
			files: map[string]string{
				"cmd/app/main.go": `package main

func sum(a [2][2]int) int {
	a[0][0] = 100
	return a[0][0] + a[0][1] + a[1][0] + a[1][1]
}

func main() {
	a := [2][2]int{{1, 2}, {3, 4}}
	b := a
	b[1][1] = 5
	if sum(a) == 109 && a[0][0] == 1 && b != a {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "slices_of_arrays",
			reason: "value-carrying slices of fixed arrays need array element copy semantics",
			files: map[string]string{
				"cmd/app/main.go": `package main

func main() {
	rows := [][2]int{{1, 2}, {3, 4}}
	copy := rows[0]
	copy[0] = 9
	rows[1] = copy
	if rows[0][0] == 1 && rows[1][0] == 9 && rows[1][1] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "stored_function_values",
			reason: "function values need storage, reassignment, and indirect calls",
			files: map[string]string{
				"cmd/app/main.go": `package main

func add1(x int) int {
	return x + 1
}

func apply(f func(int) int, value int) int {
	return f(value)
}

func main() {
	funcs := []func(int) int{add1, func(x int) int { return x * 2 }}
	f := funcs[0]
	f = funcs[1]
	if apply(f, 3) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "returned_closures",
			reason: "closures returned from functions need heap-like environments",
			files: map[string]string{
				"cmd/app/main.go": `package main

func makeAdder(base int) func(int) int {
	total := base
	return func(next int) int {
		total += next
		return total
	}
}

func main() {
	add := makeAdder(10)
	if add(1)+add(2) == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "stored_method_values",
			reason: "method values need stored receiver environments and reassignment",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Counter struct {
	value int
}

func (c *Counter) Add(next int) int {
	c.value += next
	return c.value
}

func main() {
	c := &Counter{value: 1}
	var f func(int) int = c.Add
	g := f
	if f(2)+g(3) == 9 && c.value == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "non_string_panic_recover",
			reason: "panic/recover needs any-valued payloads, not only strings",
			files: map[string]string{
				"cmd/app/main.go": `package main

func main() {
	defer func() {
		if recover() == 7 {
			print("PASS\n")
			return
		}
		print("FAIL\n")
	}()
	panic(7)
}
`,
			},
		},
		{
			name:   "indirect_recover_semantics",
			reason: "recover must only work when called directly by a deferred function",
			files: map[string]string{
				"cmd/app/main.go": `package main

func helper() any {
	return recover()
}

func main() {
	defer func() {
		if helper() != nil {
			print("FAIL\n")
			return
		}
		if recover() == "boom" {
			print("PASS\n")
			return
		}
		print("FAIL\n")
	}()
	panic("boom")
}
`,
			},
		},
		{
			name:   "first_class_complex_values",
			reason: "complex numbers need a first-class value representation",
			files: map[string]string{
				"cmd/app/main.go": `package main

func add(a complex128, b complex128) complex128 {
	return a + b
}

type Box struct {
	value complex128
}

func main() {
	z := add(1+2i, complex(3, 4))
	box := Box{value: z}
	if real(box.value) == 4 && imag(box.value) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "named_complex_values",
			reason: "named complex values and complex-containing types need real storage",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Number complex128

func double(v Number) Number {
	return v + v
}

func main() {
	v := double(Number(2 + 3i))
	if real(complex128(v)) == 4 && imag(complex128(v)) == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "init_functions_and_initializer_calls",
			reason: "init ordering and package variable initializer calls need full frontend support",
			files: map[string]string{
				"pkg/dep/dep.go": `package dep

var Value = seed()

func seed() int {
	return 40
}

func init() {
	Value += 2
}
`,
				"cmd/app/main.go": `package main

import "{{MODULE}}/pkg/dep"

func main() {
	if dep.Value == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "value_carrying_composite_fields",
			reason: "struct fields containing slices, interfaces, and function values need full value support",
			files: map[string]string{
				"cmd/app/main.go": `package main

type Holder struct {
	Items []int
	Any   any
	Fn    func(int) int
}

func main() {
	h := Holder{
		Items: []int{1, 2},
		Any:   "ok",
		Fn:    func(v int) int { return v + 1 },
	}
	text := h.Any.(string)
	if len(h.Items) == 2 && h.Fn(h.Items[1]) == 3 && text == "ok" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "unsafe_recursive_layout",
			reason: "unsafe.Sizeof needs full layout for recursive and pointer-rich structs",
			files: map[string]string{
				"cmd/app/main.go": `package main

import "unsafe"

type Node struct {
	Next   *Node
	Values []int
}

func main() {
	if unsafe.Sizeof(Node{}) > 0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "embed_fs_tree",
			reason: "go:embed needs embed.FS and directory or multi-file pattern support",
			files: map[string]string{
				"cmd/app/data/msg.txt": "PASS\n",
				"cmd/app/main.go": `package main

import "embed"

//go:embed data/*.txt
var files embed.FS

func main() {
	data, err := files.ReadFile("data/msg.txt")
	if err == nil {
		print(string(data))
		return
	}
	print("FAIL\n")
}
`,
			},
		},
		{
			name:   "standard_library_compatibility",
			reason: "rtg/std needs the common package and selector surface used by ordinary Go programs",
			files: map[string]string{
				"cmd/app/main.go": `package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"time"
	"unicode/utf8"
)

func readSome(r io.Reader) int {
	buf := make([]byte, 4)
	n, _ := r.Read(buf)
	return n
}

func main() {
	values := []int{3, 1, 2}
	sort.Ints(values)
	ok := math.Abs(-1) == 1
	ok = ok && utf8.RuneCountInString("ok") == 2
	ok = ok && fmt.Sprintf("%d", values[0]) == "1"
	ok = ok && errors.New("x").Error() == "x"
	ok = ok && regexp.MustCompile("P.*S").MatchString("PASS")
	ok = ok && reflect.TypeOf(values[0]).Kind() == reflect.Int
	ok = ok && http.MethodGet == "GET"
	ok = ok && time.Unix(0, 0).Year() == 1970
	ok = ok && readSome(bytes.NewBufferString("abcd")) == 4
	file, err := os.Create("remaining.txt")
	if err != nil {
		print("FAIL\n")
		return
	}
	_, _ = file.WriteString("PASS\n")
	_ = file.Close()
	opened, err := os.OpenFile("remaining.txt", os.O_RDONLY, 0)
	if err != nil {
		print("FAIL\n")
		return
	}
	data := make([]byte, 5)
	n, _ := opened.Read(data)
	_ = opened.Close()
	_ = os.Remove("remaining.txt")
	if ok && string(data[:n]) == "PASS\n" {
		print(string(data[:n]))
		return
	}
	print("FAIL\n")
}
`,
			},
		},
	}
}
