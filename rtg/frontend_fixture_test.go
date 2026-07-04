package rtg_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/build"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/unit"
)

const crossArchTestsEnv = "RTG_CROSS_ARCH_TESTS"
const selfHostTestsEnv = "RTG_SELFHOST_TESTS"

type frontendSmokeTarget struct {
	name   string
	runner []string
}

func TestHelloFixtureGoldenUnits(t *testing.T) {
	assertFixtureGoldenUnits(t, "hello_module")
}

func TestHelloFixtureFrontendMatchesHostGo(t *testing.T) {
	fixture := filepath.Join("testdata", "hello_module")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStructFixtureGoldenUnits(t *testing.T) {
	assertFixtureGoldenUnits(t, "struct_module")
}

func TestStructFixtureFrontendMatchesHostGo(t *testing.T) {
	fixture := filepath.Join("testdata", "struct_module")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSelfHostedFrontendBuildsHelloFixture(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend test", selfHostTestsEnv)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("self-hosted frontend smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	tmp := t.TempDir()
	self := filepath.Join(tmp, "rtg-self")
	cmd := exec.Command("go", "run", "./cmd/rtg", "-t", "linux/amd64", "-o", self, "./cmd/rtg")
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self frontend build failed: %v\n%s", err, string(data))
	}
	out := filepath.Join(tmp, "hello")
	cmd = exec.Command(self, "-t", "linux/amd64", "-o", out, "./testdata/hello_module/cmd/app")
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted fixture build failed: %v\n%s", err, string(data))
	}
	cmd = exec.Command(out)
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted fixture failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("self-hosted fixture output = %q, want PASS", string(data))
	}
	unitDir := filepath.Join(tmp, "units")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("MkdirAll unit dir failed: %v", err)
	}
	cmd = exec.Command(self, "-emit-unit", "-o", unitDir, "./testdata/hello_module/cmd/app")
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted fixture emit-unit failed: %v\n%s", err, string(data))
	}
	linked := filepath.Join(tmp, "linked-hello")
	cmd = exec.Command(self, "-link", "-t", "linux/amd64", "-o", linked, unitDir)
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted unit directory link failed: %v\n%s", err, string(data))
	}
	cmd = exec.Command(linked)
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("self-hosted linked fixture failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("self-hosted linked fixture output = %q, want PASS", string(data))
	}
}

func TestSelfHostedFrontendBuildsStage3(t *testing.T) {
	if os.Getenv(selfHostTestsEnv) != "1" {
		t.Skipf("set %s=1 to run self-hosted frontend test", selfHostTestsEnv)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("self-hosted frontend smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	tmp := t.TempDir()
	root := ".."
	stage1 := filepath.Join(tmp, "stage1")
	cmd := exec.Command("go", "run", "./rtg/cmd/rtg", "-t", "linux/amd64", "-o", stage1, "./rtg/cmd/rtg")
	cmd.Dir = root
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 frontend build failed: %v\n%s", err, string(data))
	}

	unitDir := filepath.Join(tmp, "stage1-units")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatalf("MkdirAll unit dir failed: %v", err)
	}
	cmd = exec.Command(stage1, "-emit-unit", "-o", unitDir, "./rtg/cmd/rtg")
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 emit-unit failed: %v\n%s", err, string(data))
	}

	stage2 := filepath.Join(tmp, "stage2")
	cmd = exec.Command(stage1, "-link", "-t", "linux/amd64", "-o", stage2, unitDir)
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 link stage2 failed: %v\n%s", err, string(data))
	}

	stage3 := filepath.Join(tmp, "stage3")
	cmd = exec.Command(stage2, "-t", "linux/amd64", "-o", stage3, "./rtg/cmd/rtg")
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 build stage3 failed: %v\n%s", err, string(data))
	}

	hello := filepath.Join(tmp, "hello")
	cmd = exec.Command(stage3, "-t", "linux/amd64", "-o", hello, "./rtg/testdata/hello_module/cmd/app")
	cmd.Dir = root
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage3 fixture build failed: %v\n%s", err, string(data))
	}
	cmd = exec.Command(hello)
	data, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage3-built fixture failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("stage3-built fixture output = %q, want PASS", string(data))
	}
}

func TestNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nested\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/nested/pkg/dep"

func value() int {
	var total = dep.Join(dep.First(), dep.Second())
	if dep.Join(dep.First(), dep.Second()) == 12 {
		return total
	}
	return 0
}

func main() {
	if value() == 12 {
		dep.Emit(dep.First(), dep.Second())
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
func Emit(a int, b int) int {
	if Join(a, b) == 12 {
		print("PASS\n")
	}
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticFunctionAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/functionalias\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func add(a int) int { return a + 1 }
func join(a int, b int) int { return a + b }

func main() {
	f := add
	var g = join
	_ = add
	_ = f
	_ = func() int { return 99 }
	if f(1) + g(2, 3) == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedStaticFunctionAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedfunctionalias\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedfunctionalias/pkg/dep"

func main() {
	f := dep.Add
	_ = dep.Add
	_ = f
	if f(41) == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Add(x int) int {
	return x + 1
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSwitchNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/switchcall\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/switchcall/pkg/dep"

func main() {
	switch dep.Join(dep.First(), dep.Second()) {
	case 12:
		dep.Emit()
	default:
		print("FAIL\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Second() int { return 2 }
func Join(a int, b int) int { return a*10 + b }
func Emit() int {
	print("PASS\n")
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestFallthroughFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/fallthrough\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	if total == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLabeledBreakContinueFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/labeledcontrol\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	total := 0
outer:
	for i := 0; i < 4; i++ {
		total = total + 10
		for j := 0; j < 4; j++ {
			if i == 1 && j == 1 {
				continue outer
			}
			if i == 3 && j == 0 {
				break outer
			}
			total = total + 1
		}
		total = total + 100
	}
	if total == 249 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestTopLevelDeferFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferbasic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func emit(text string) {
	print(text)
}

func finish() int {
	print("R")
	return 0
}

func run() int {
	text := "PASS"
	defer emit("\n")
	defer emit(text)
	text = "FAIL"
	return finish()
}

func main() {
	_ = run()
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedNoArgDeferFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nesteddefer\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func pass() { print("PASS") }
func fail() { print("FAIL") }
func newline() { print("\n") }

func main() {
	defer newline()
	if true {
		defer pass()
	}
	if false {
		defer fail()
	}
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedDeferArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nesteddeferargs\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func emit(text string, code int) {
	if code == 7 {
		print(text)
		return
	}
	print("FAIL")
}

func newline() {
	print("\n")
}

func main() {
	text := "FAIL"
	defer newline()
	if true {
		text = "PASS"
		defer emit(text, 7)
		text = "BAD"
	}
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestVariadicDeferExpansionFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferexpand\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func emit(label string, values ...int) {
	if label == "INNER" && len(values) == 2 && values[0] == 3 && values[1] == 4 {
		print("INNER")
		return
	}
	if label == "OUTER" && len(values) == 2 && values[0] == 1 && values[1] == 2 {
		print("OUTER")
		return
	}
	print("FAIL")
}

func newline() {
	print("\n")
}

func main() {
	values := []int{1, 2}
	defer newline()
	defer emit("OUTER", values...)
	values = []int{9}
	if true {
		more := []int{3, 4}
		defer emit("INNER", more...)
		more = []int{8}
	}
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDeferredFunctionLiteralFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferliteral\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	text := "FAIL\n"
	defer func() {
		print(text)
	}()
	defer func(value string) {
		if value == "!" {
			print(value)
		}
	}("!")
	text = "PASS\n"
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLoopDeferFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/loopdefer\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	defer mark("A")
	for i := 0; i < 3; i++ {
		defer emit(i)
	}
	defer mark("T")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDeferArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func sink(v int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	defer sink(boom())
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLoopDeferArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/loopdeferargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func sink(v int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	for i := 0; i < 1; i++ {
		defer sink(boom())
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDeferredFunctionLiteralArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferredliteralargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func main() {
	defer cleanup()
	defer func(v int) {
		print("FAIL\n")
	}(boom())
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/recoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func main() {
	defer cleanup()
	panic("boom")
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestUnrecoveredStringPanicFrontendFails(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/unrecoveredpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	panic("boom")
	print("FAIL\n")
}
`)
	runFrontendFixtureFailureContains(t, fixture, []byte("panic: boom\n"))
}

func TestUnrecoveredAppMainStringPanicFrontendFails(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/unrecoveredappmainpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func appMain(args []string, env []string) int {
	panic("boom")
	return 0
}
`)
	runFrontendFixtureFailureContains(t, fixture, []byte("panic: boom\n"))
}

func TestCallerRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/callerrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() {
	panic("boom")
	print("FAIL\n")
}

func main() {
	defer cleanup()
	boom()
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedrecoverpanic/pkg/dep"

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func main() {
	defer cleanup()
	dep.Boom()
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Boom() {
	panic("boom")
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAssignmentCallRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/assignmentrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func callBoom() int {
	value := boom()
	print("FAIL\n")
	return value
}

func main() {
	defer cleanup()
	value := callBoom()
	printInt(value)
	print("FAIL\n")
}

func printInt(v int) int { return v }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedAssignmentCallRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedassignmentrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedassignmentrecoverpanic/pkg/dep"

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func main() {
	defer cleanup()
	value := dep.Boom()
	printInt(value)
	print("FAIL\n")
}

func printInt(v int) int { return v }
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMethodCallRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methodrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct{}

func (b box) boom() {
	panic("boom")
	print("FAIL\n")
}

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func main() {
	defer cleanup()
	value := box{}
	value.boom()
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedAssignmentCallArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nestedassignmentrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func wrap(v int) int {
	print("FAIL\n")
	return v
}

func main() {
	defer cleanup()
	value := wrap(boom())
	printInt(value)
	print("FAIL\n")
}

func printInt(v int) int { return v }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAssignmentOperandsRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/assignmentoperandsrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func main() {
	defer cleanup()
	a, b := boom(), fail()
	printInt(a + b)
	print("FAIL\n")
}

func printInt(v int) int { return v }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCallArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/callargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func fail(v int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	fail(boom())
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedCallArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nestedcallargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func wrap(v int) int {
	print("FAIL\n")
	return v
}

func sink(v int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	sink(wrap(boom()))
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDeepNestedCallArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deepnestedcallargrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 7
}

func mid(v int) int {
	print("FAIL\n")
	return v
}

func top(v int) int {
	print("FAIL\n")
	return v
}

func sink(v int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	sink(top(mid(boom())))
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCallArgumentsRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/callargsrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func sink(a int, b int) {
	print("FAIL\n")
}

func main() {
	defer cleanup()
	sink(boom(), fail())
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestIfConditionRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ifconditionrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func shouldBoom() bool {
	panic("boom")
	print("FAIL\n")
	return true
}

func main() {
	defer cleanup()
	if shouldBoom() {
		print("FAIL\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedConditionArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nestedconditionrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup(text string) {
	if recover() == "boom" {
		print(text)
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func keep(v int) bool {
	print("FAIL\n")
	return true
}

func runIf() {
	defer cleanup("IF\n")
	if keep(boom()) {
		print("FAIL\n")
	}
	print("FAIL\n")
}

func runFor() {
	defer cleanup("FOR\n")
	for keep(boom()) {
		print("FAIL\n")
	}
	print("FAIL\n")
}

func main() {
	runIf()
	runFor()
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSwitchConditionRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/switchconditionrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func shouldBoom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func main() {
	defer cleanup()
	switch shouldBoom() {
	case 1:
		print("FAIL\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestForConditionRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forconditionrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func shouldBoom() bool {
	panic("boom")
	print("FAIL\n")
	return true
}

func main() {
	defer cleanup()
	for shouldBoom() {
		print("FAIL\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForInitRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/classicforinitrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func main() {
	defer cleanup()
	for i := boom(); i < 1; i++ {
		print("FAIL\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForConditionRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/classicforconditionrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func shouldBoom() bool {
	panic("boom")
	print("FAIL\n")
	return true
}

func main() {
	defer cleanup()
	for ; shouldBoom(); {
		print("FAIL\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForPostRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/classicforpostrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func next() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func main() {
	defer cleanup()
	for i := 0; i < 1; i = next() {
		print("BODY\n")
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCombinedClassicForRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/combinedclassicforrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup(text string) {
	if recover() == "boom" {
		print(text)
		return
	}
	print("FAIL\n")
}

func shouldBoom() bool {
	panic("boom")
	print("FAIL\n")
	return true
}

func start() int {
	return 0
}

func next() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func runCondition() {
	defer cleanup("COND\n")
	for i := 0; shouldBoom(); i++ {
		print("FAIL\n")
	}
	print("FAIL\n")
}

func runPost() {
	defer cleanup("POST\n")
	for i := start(); i < 1; i = next() {
		print("BODY\n")
	}
	print("FAIL\n")
}

func main() {
	runCondition()
	runPost()
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestReturnOperandsRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/returnoperandsrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func fail() int {
	print("FAIL\n")
	return 2
}

func pair() (int, int) {
	return boom(), fail()
}

func main() {
	defer cleanup()
	_, _ = pair()
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNestedReturnCallArgumentRecoverableStringPanicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nestedreturnrecoverpanic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func cleanup() {
	if recover() == "boom" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}

func boom() int {
	panic("boom")
	print("FAIL\n")
	return 1
}

func wrap(v int) int {
	print("FAIL\n")
	return v
}

func value() int {
	return wrap(boom())
}

func main() {
	defer cleanup()
	_ = value()
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDeferredMultiResultDirectReturnFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/deferredmultireturn\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	a, b := wrapper()
	if a == 1 && b == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestFullSliceFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/fullslice\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	xs := []int{1, 2, 3, 4}
	ys := xs[1:2:2]
	ys = append(ys, 9)
	if len(ys) == 2 && ys[0] == 2 && ys[1] == 9 && xs[2] == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCapBuiltinFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/capbuiltin\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Counts []int

type box struct {
	items []int
}

func fromCall() []int {
	return make([]int, 1, 4)
}

func main() {
	xs := make([]int, 2, 5)
	ys := xs[:1:3]
	lit := []int{1, 2, 3}
	named := Counts{4, 5}
	b := box{items: make([]int, 0, 6)}
	called := fromCall()
	if cap(xs) == 5 && cap(ys) == 3 && cap(lit) == 3 && cap(named) == 2 && cap(b.items) == 6 && cap(called) == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestReducibleComplexComponentsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/complexcomponents\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var firstCount int
var secondCount int

func first() float64 {
	firstCount = firstCount + 1
	return 6
}

func second() float64 {
	secondCount = secondCount + 1
	return 9.5
}

func main() {
	var a float64 = 6
	b := 9.5
	r := real(complex(a, b))
	i := imag(complex(2, 5))
	side := real(complex(first(), second()))
	literalR := real(1+2i)
	literalI := imag(3-4i)
	pureI := imag(-5i)
	reverseR := real(6i-7)
	if r == 6 && i == 5 && side == 6 && literalR == 1 && literalI == -4 && pureI == -5 && reverseR == -7 && firstCount == 1 && secondCount == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticComplexAliasComponentsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/complexaliascomponents\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var order int

func first() float64 {
	order = order*10 + 1
	return 6
}

func second() float64 {
	order = order*10 + 2
	return 9.5
}

func main() {
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
	again := real(z)
	if r == 6 && i == 9.5 && lr == 1.5 && li == 2.5 && tr == 6 && ti == 9.5 && tlr == 3.5 && tli == 4.5 && again == 6 && order == 1212 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedComplexCallSideEffectsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/complexdiscard\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var order int

func first(x int) float64 {
	order = order*10 + x
	return 1.5
}

func second() float64 {
	order = order*10 + 2
	return 2.5
}

func third() float64 {
	order = order*10 + 3
	return 3.5
}

func fourth() float64 {
	order = order*10 + 4
	return 4.5
}

func main() {
	_ = complex(first(1), 2.0)
	_, _ = 5i, complex(3.0, second())
	_ = complex(third(), second())
	_ = complex(first(1), 2.0) + 3i
	_ = 4i + complex(5.0, second())
	_ = complex(third(), 6.0) - complex(7.0, fourth())
	if order == 12321234 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestBlankDiscardedComplexVarsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/complexvardiscard\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var order int

func first() float64 {
	order = order*10 + 1
	return 1.5
}

func second() float64 {
	order = order*10 + 2
	return 2.5
}

func main() {
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
	if order == 1212 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestForConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forcall\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forcall/pkg/dep"

func main() {
	count := 0
	for dep.KeepGoing(dep.Step()) {
		count = count + 1
		if count > 3 {
			print("FAIL\n")
			return
		}
	}
	if count == 2 && dep.Count() == 3 {
		dep.Emit()
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func KeepGoing(v int) bool { return v < 3 }
func Count() int { return count }
func Emit() int {
	print("PASS\n")
	return 0
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForInitNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forinit\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forinit/pkg/dep"

func main() {
	total := 0
	for i := dep.Next(dep.First()); i < 4; i = i + 1 {
		total = total + i
	}
	if total == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forcond\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forcond/pkg/dep"

func main() {
	total := 0
	for i := 0; dep.KeepGoing(dep.Step()); i = i + 1 {
		total = total + i
		if i > 4 {
			print("FAIL\n")
			return
		}
	}
	if total == 3 && dep.Count() == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func KeepGoing(v int) bool { return v < 4 }
func Count() int { return count }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestClassicForPostNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/forpost\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/forpost/pkg/dep"

func main() {
	total := 0
	for i := 0; i < 3; i = dep.Next(dep.Step()) {
		total = total + i
	}
	if total == 2 && dep.Count() == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var count int
func Step() int {
	count = count + 1
	return count
}
func Next(v int) int { return v + 1 }
func Count() int { return count }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSliceBoundCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/slicebounds\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/slicebounds/pkg/dep"

func main() {
	text := "xPASS\nx"
	print(text[dep.Start():dep.End()])
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Start() int { return 1 }
func End() int { return 6 }
`)
	runFrontendFixtureOutput(t, fixture, []byte("PASS\n"))
}

func TestIfShortConditionNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ifshortcond\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/ifshortcond/pkg/dep"

func main() {
	if total := dep.First(); dep.Check(dep.Next(total)) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
func Check(v int) bool { return v == 2 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestShortCircuitNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/shortcircuit\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/shortcircuit/pkg/dep"

func main() {
	if false && dep.Check(dep.Fail()) {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Fail() int {
	print("FAIL\n")
	return 1
}

func Check(v int) bool {
	return v == 1
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSwitchShortTagNestedCallArgumentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/switchshorttag\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/switchshorttag/pkg/dep"

func main() {
	switch total := dep.First(); dep.Next(total) {
	case 2:
		print("PASS\n")
		return
	default:
		print("FAIL\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func First() int { return 1 }
func Next(v int) int { return v + 1 }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdFmtPrintlnFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdprint\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "fmt"

func main() {
	fmt.Print("PA")
	fmt.Print("SS")
	fmt.Print("\n")
	fmt.Println("PASS")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestUnsafeSizeofFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/unsafesizeof\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "unsafe"

type ByteAlias byte
type ByteAlias2 ByteAlias
type BytesAlias []byte
type PtrAlias *byte
type WordAlias int16
type Matrix [2][3]int

var ag ByteAlias = 1
var ax BytesAlias = BytesAlias{1, 2}
var ap PtrAlias

func main() {
	var b byte = 1
	xs := []byte{1, 2}
	s := "x"
	p := &b
	local := ByteAlias2(1)
	total := unsafe.Sizeof(ag) + unsafe.Sizeof(ax) + unsafe.Sizeof(ap) + unsafe.Sizeof(local) + unsafe.Sizeof(WordAlias(1)) + unsafe.Sizeof(b) + unsafe.Sizeof(xs) + unsafe.Sizeof(s) + unsafe.Sizeof(p) + unsafe.Sizeof(byte(1)) + unsafe.Sizeof(int16(1)) + unsafe.Sizeof(int32(1)) + unsafe.Sizeof(int64(1)) + unsafe.Sizeof(float64(1)) + unsafe.Sizeof(true) + unsafe.Sizeof("x") + unsafe.Sizeof(&b) + unsafe.Sizeof([2][3]int{{1, 2, 3}, {4, 5, 6}}) + unsafe.Sizeof(Matrix{})
	if total == 229 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdOSFileIntrinsicsFrontendRuns(t *testing.T) {
	fixture := t.TempDir()
	dataPath := filepath.ToSlash(filepath.Join(fixture, "rtg_std_os.tmp"))
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdos\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "os"

func fail(msg string) {
	os.Write(os.Stdout, []byte(msg), -1)
}

func main() {
	fd := os.Open(`+strconv.Quote(dataPath)+`, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
	if fd < 0 {
		fail("FAIL open\n")
		return
	}
	var data []byte
	data = append(data, 'O')
	data = append(data, 'K')
	if os.Write(fd, data, int64(0)) != 2 {
		fail("FAIL write\n")
		os.Close(fd)
		return
	}
	var readBuf []byte
	readBuf = append(readBuf, 0)
	readBuf = append(readBuf, 0)
	if os.Read(fd, readBuf, int64(0)) != 2 {
		fail("FAIL read\n")
		os.Close(fd)
		return
	}
	if os.Chmod(fd, 420) != 0 {
		fail("FAIL chmod\n")
		os.Close(fd)
		return
	}
	os.Close(fd)
	if readBuf[0] == 'O' && readBuf[1] == 'K' {
		os.Write(os.Stdout, []byte("PASS\n"), -1)
		return
	}
	fail("FAIL data\n")
}
`)
	runFrontendFixtureOutput(t, fixture, []byte("PASS\n"))
}

func TestStdStringsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdstrings\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "strings"

func main() {
	value := "prefix-body-suffix"
	ok := strings.HasPrefix(value, "prefix")
	ok = ok && strings.HasSuffix(value, "suffix")
	ok = ok && strings.Contains(value, "body")
	ok = ok && !strings.Contains(value, "missing")
	ok = ok && strings.Contains(value, "")
	ok = ok && strings.ContainsAny(value, "xyz-")
	ok = ok && !strings.ContainsAny(value, "XYZ")
	ok = ok && strings.IndexByte(value, '-') == 6
	ok = ok && strings.IndexByte(value, 'z') == -1
	ok = ok && strings.LastIndexByte(value, '-') == 11
	ok = ok && strings.LastIndexByte(value, 'z') == -1
	ok = ok && strings.Index(value, "body") == 7
	ok = ok && strings.Index(value, "missing") == -1
	ok = ok && strings.LastIndex(value, "-") == 11
	ok = ok && strings.LastIndex(value, "missing") == -1
	ok = ok && strings.Count(value, "-") == 2
	ok = ok && strings.Count(value, "") == len(value)+1
	ok = ok && strings.Repeat("PA", 2) == "PAPA"
	ok = ok && strings.Repeat("x", 0) == ""
	ok = ok && strings.ReplaceAll("banana", "na", "NA") == "baNANA"
	ok = ok && strings.ReplaceAll("ab", "", ".") == ".a.b."
	ok = ok && strings.ReplaceAll("miss", "z", "x") == "miss"
	ok = ok && strings.TrimPrefix(value, "prefix-") == "body-suffix"
	ok = ok && strings.TrimSuffix(value, "-suffix") == "prefix-body"
	ok = ok && strings.TrimSpace(" \tPASS\n") == "PASS"
	parts := strings.Split("red,green,blue", ",")
	ok = ok && len(parts) == 3 && parts[0] == "red" && parts[2] == "blue"
	fields := strings.Fields("  alpha\tbeta\n gamma ")
	ok = ok && len(fields) == 3 && fields[1] == "beta"
	ok = ok && strings.Join(fields, "-") == "alpha-beta-gamma"
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdStringsBuilderFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdstringsbuilder\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "strings"

func main() {
	var b strings.Builder
	p := &b
	p.Write([]byte("P"))
	p.WriteString("A")
	b.WriteByte('S')
	b.WriteByte('S')
	if b.Len() != 4 {
		print("FAIL len\n")
		return
	}
	if b.String() != "PASS" {
		print("FAIL string\n")
		return
	}
	b.Reset()
	p.Write([]byte("PASS"))
	p.WriteByte('\n')
	print(p.String())
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdStrconvFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdstrconv\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "strconv"

func main() {
	ok := strconv.Itoa(0) == "0"
	ok = ok && strconv.Itoa(12345) == "12345"
	ok = ok && strconv.Itoa(-42) == "-42"
	formatted := strconv.FormatInt(-255, 16)
	ok = ok && formatted == "-ff"
	parsed, err := strconv.ParseInt("0x1e", 0, 64)
	ok = ok && err == nil && parsed == 30
	parsed, err = strconv.ParseInt("-101", 2, 64)
	ok = ok && err == nil && parsed == -5
	ok = ok && strconv.Quote("PASS\n") == "\"PASS\\n\""
	ok = ok && strconv.Quote("a\\b") == "\"a\\\\b\""
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdPathFilepathFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdfilepath\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "path/filepath"

func main() {
	ok := filepath.Separator == '/'
	ok = ok && filepath.ToSlash("a/b") == "a/b"
	ok = ok && filepath.FromSlash("a/b") == "a/b"
	ok = ok && filepath.IsAbs("/tmp")
	ok = ok && !filepath.IsAbs("tmp")
	ok = ok && filepath.Clean("/a//b/./c/..") == "/a/b"
	ok = ok && filepath.Clean("a/../b") == "b"
	ok = ok && filepath.Join("/tmp/", "app") == "/tmp/app"
	ok = ok && filepath.Base("/tmp/app/main.go") == "main.go"
	ok = ok && filepath.Dir("/tmp/app/main.go") == "/tmp/app"
	ok = ok && filepath.Ext("/tmp/app/main.go") == ".go"
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdBytesBufferFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdbytes\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "bytes"

func main() {
	var b bytes.Buffer
	p := &b
	p.Write([]byte("PA"))
	p.WriteByte('S')
	p.WriteByte('S')
	if b.Len() != 4 {
		print("FAIL len\n")
		return
	}
	if string(p.Bytes()) != "PASS" {
		print("FAIL bytes\n")
		return
	}
	p.WriteString("\n")
	if p.String() == "PASS\n" {
		fromString := bytes.NewBufferString("PASS")
		fromString.WriteByte('\n')
		fromBytes := bytes.NewBuffer([]byte("OK"))
		fromBytes.WriteString("!")
		if fromString.String() == "PASS\n" && fromBytes.String() == "OK!" {
			print(p.String())
			return
		}
	}
	print("FAIL string\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdBytesHelpersFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdbyteshelpers\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "bytes"

func main() {
	value := []byte("prefix-body-suffix")
	ok := bytes.Equal([]byte("PASS"), []byte("PASS"))
	ok = ok && !bytes.Equal([]byte("PASS"), []byte("FAIL"))
	ok = ok && bytes.HasPrefix(value, []byte("prefix"))
	ok = ok && !bytes.HasPrefix(value, []byte("body"))
	ok = ok && bytes.HasSuffix(value, []byte("suffix"))
	ok = ok && !bytes.HasSuffix(value, []byte("prefix"))
	ok = ok && bytes.Contains(value, []byte("body"))
	ok = ok && !bytes.Contains(value, []byte("missing"))
	ok = ok && bytes.Contains(value, []byte(""))
	ok = ok && bytes.Index(value, []byte("body")) == 7
	ok = ok && bytes.Index(value, []byte("missing")) == -1
	ok = ok && bytes.IndexByte(value, '-') == 6
	ok = ok && bytes.IndexByte(value, 'z') == -1
	if ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdRuntimeFrontendMatchesHostGo(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("native runtime fixture expects linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdruntime\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "runtime"

func main() {
	if runtime.GOOS == "linux" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "386" || runtime.GOARCH == "arm64" || runtime.GOARCH == "arm") {
		print("PASS\n")
		return
	}
	if runtime.GOOS == "wasi" && runtime.GOARCH == "wasm32" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStdRuntimeFrontendTargetConstants(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stdruntimetarget\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "runtime"

func main() {
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	units := loadFixtureUnitsForTarget(t, fixture, "linux/aarch64")
	if !unitsContainBody(units, `const rtg_runtime_GOOS = "linux"`) {
		t.Fatalf("linux/aarch64 units did not include linux GOOS: %#v", units)
	}
	if !unitsContainBody(units, `const rtg_runtime_GOARCH = "arm64"`) {
		t.Fatalf("linux/aarch64 units did not include arm64 GOARCH: %#v", units)
	}
	if unitsContainBody(units, `const rtg_runtime_GOARCH = "amd64"`) {
		t.Fatalf("linux/aarch64 units included amd64 GOARCH: %#v", units)
	}
}

func TestMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type point struct {
	x int
	y int
}

func (p point) Sum() int {
	return p.x + p.y
}

func main() {
	p := point{x: 3, y: 4}
	if p.Sum() == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDirectMethodExpressionFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methodexpressions\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type point struct {
	x int
	y int
}

func (p point) Sum() int {
	return p.x + p.y
}

func (p point) Add(v int) int {
	return p.x + p.y + v
}

func main() {
	if point.Sum(point{x: 3, y: 4}) + point.Add(point{x: 5, y: 6}, 24) == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMethodExpressionAliasesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methodexpressionaliases\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type point struct {
	x int
	y int
}

func (p point) Sum() int {
	return p.x + p.y
}

func (p point) Add(v int) int {
	return p.x + p.y + v
}

func main() {
	f := point.Sum
	var g = point.Add
	_ = point.Sum
	_ = f
	if f(point{x: 3, y: 4}) + g(point{x: 5, y: 6}, 24) == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMethodValueFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/methodvalues\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
}

func (b box) Value() int {
	return b.value
}

func (b *box) Add(v int) int {
	b.value = b.value + v
	return b.value
}

func main() {
	b := box{value: 1}
	f := b.Value
	g := b.Add
	_ = b.Value
	_ = b.Add
	_ = f
	if f() + g(40) == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticFunctionLiteralFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/functionliterals\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	if f(1) + g(2, 3) + h(2) + offset + shadow() == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticCallbackFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/staticcallbacks\n")
	writeFixtureFile(t, fixture, "dep/dep.go", `package dep

func Add(a int, b int) int {
	return a + b
}

type Box struct {
	Value int
}

func (b Box) Add(a int) int {
	return b.Value + a
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/staticcallbacks/dep"

func apply(f func(int, int) int, value int) int {
	return f(value, 1)
}

func applyInt(f func(int) int, value int) int {
	return f(value)
}

func add(a int, b int) int {
	return a + b
}

type box struct {
	value int
}

func (b box) add(a int) int {
	return b.value + a
}

func applyBox(f func(box, int) int, b box) int {
	return f(b, 1)
}

func applyDepBox(f func(dep.Box, int) int, b dep.Box) int {
	return f(b, 1)
}

func main() {
	f := add
	g := dep.Add
	h := box.add
	k := dep.Box.Add
	offset := 12
	captured := apply(func(a int, b int) int { offset = offset + a; return offset + b }, 2) + offset
	aliasOffset := 3
	aliasCallback := func(a int, b int) int { aliasOffset = aliasOffset + a; return aliasOffset + b }
	aliasCaptured := apply(aliasCallback, 4) + aliasOffset
	methodBox := box{value: 15}
	methodValue := methodBox.add
	methodCaptured := applyInt(methodValue, 2)
	directMethodCaptured := applyInt(methodBox.add, 3)
	if apply(add, 2)+apply(f, 3)+apply(dep.Add, 4)+apply(g, 5)+applyBox(box.add, box{value: 6})+applyBox(h, box{value: 7})+applyDepBox(dep.Box.Add, dep.Box{Value: 8})+applyDepBox(k, dep.Box{Value: 9})+apply(func(a int, b int) int { return a * b }, 11)+captured+aliasCaptured+methodCaptured+directMethodCaptured == 142 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedFunctionTypeStaticCallbackFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedfunctioncallback\n")
	writeFixtureFile(t, fixture, "dep/dep.go", `package dep

type Unary func(int) int
type AlsoUnary Unary

func Apply(f Unary, value int) int {
	return f(Bias(value))
}

func ApplyAlias(f AlsoUnary, value int) int {
	return f(Bias(value))
}

func Inc(x int) int {
	return x + 1
}

func Bias(x int) int {
	return x + 1
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/namedfunctioncallback/dep"

type Unary func(int) int
type Binary = func(int, int) int
type AlsoUnary Unary
type AlsoBinary Binary

func apply(f Unary, value int) int {
	return f(value)
}

func applyAlias(f AlsoUnary, value int) int {
	return f(value)
}

func applyBinary(f Binary, value int) int {
	return f(value, 1)
}

func applyBinaryAlias(f AlsoBinary, value int) int {
	return f(value, 1)
}

func inc(x int) int {
	return x + 1
}

func add(x int, y int) int {
	return x + y
}

func main() {
	f := inc
	g := dep.Inc
	if apply(inc, 2)+apply(f, 3)+applyBinary(add, 4)+applyAlias(inc, 5)+applyBinaryAlias(add, 6)+dep.Apply(dep.Inc, 7)+dep.Apply(g, 8)+dep.ApplyAlias(dep.Inc, 9) == 55 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImmediatelyInvokedFunctionLiteralFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/iifefunctionliterals\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	value := func(a int) int { return a + 1 }(1)
	offset := 3
	label := "PASS"
	type Values [2]int
	values := Values{2, 3}
	_ = func(x Values) int {
		x[0] = 20
		offset = offset + x[0] + x[1]
		return offset
	}(values)
	offset = offset + values[0]
	direct := [2]int{1, 1}
	_ = func(x [2]int) int {
		x[0] = 30
		offset = offset + x[0] + x[1]
		return offset
	}(direct)
	offset = offset + direct[0]
	if func(text string) bool { return len(text) == 4 }("PASS") {
		total := func(a int, b int) int {
			offset = offset + b
			sum := a + b
			return sum + offset + len(label)
		}(value, 5)
		shadow := func() int {
			offset := offset + 1
			return offset
		}()
		if total + offset + shadow == 207 {
			print("PASS\n")
			return
		}
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticPromotedMethodValueFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/promotedmethodvalues\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct {
	value int
}

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

func main() {
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
	if f() + g(2) + h() + k(5) + applyValue(other.Value) + applyAdd(other.Add, 2) + applyValue(otherPtr.Value) + applyAdd(otherPtr.Add, 3) == 134 && outer.value == 3 && pointer.value == 15 && other.value == 22 && otherPtr.value == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticCompositeLiteralMethodValueFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/compositemethodvalues\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	value int
}

func (b Box) Value() int {
	return b.value
}

func (b *Box) Add(v int) int {
	b.value = b.value + v
	return b.value
}

type Inner struct {
	value int
}

func (in Inner) InnerValue() int {
	return in.value
}

func (in *Inner) InnerAdd(v int) int {
	in.value = in.value + v
	return in.value
}

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func applyValue(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func main() {
	f := Box{value: 1}.Value
	g := (&Box{value: 2}).Add
	h := Outer{Inner: Inner{value: 3}}.InnerValue
	k := PointerOuter{Inner: &Inner{value: 4}}.InnerAdd
	m := (&Outer{Inner: Inner{value: 5}}).InnerAdd
	_ = Box{value: 9}.Value
	_ = (&Box{value: 10}).Add
	_ = Outer{Inner: Inner{value: 11}}.InnerValue
	_ = f
	if f() + g(3) + h() + k(6) + m(7) + applyValue(f) + applyAdd(g, 4) + applyValue(Box{value: 6}.Value) + applyAdd((&Box{value: 7}).Add, 2) + applyValue(Outer{Inner: Inner{value: 8}}.InnerValue) + applyAdd((&Outer{Inner: Inner{value: 9}}).InnerAdd, 2) + applyValue(PointerOuter{Inner: &Inner{value: 10}}.InnerValue) + applyAdd(PointerOuter{Inner: &Inner{value: 11}}.InnerAdd, 3) == 99 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCompositeLiteralMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/litmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type point struct {
	x int
	y int
}

func (p point) Sum() int {
	return p.x + p.y
}

func main() {
	if (point{x: 3, y: 4}).Sum() != 7 {
		print("FAIL\n")
		return
	}
	if (point{x: 5, y: 6}).Sum() != 11 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPointerReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ptrmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type counter struct {
	n int
}

func (c *counter) Add(v int) {
	c.n = c.n + v
}

func main() {
	var c counter
	c.Add(7)
	c.Add(5)
	if c.n == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestValueReceiverPointerMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/valueptrmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Items []int

func (items Items) Add(v int) Items {
	return append(items, v)
}

func Fill(items *Items) {
	*items = items.Add(5)
}

func main() {
	values := Items([]int{37})
	Fill(&values)
	if len(values) == 2 && values[0]+values[1] == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAddressedCompositeLiteralMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/litptrmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type counter struct {
	n int
}

func (c *counter) Value() int {
	return c.n
}

func main() {
	if (&counter{n: 42}).Value() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestIndexedReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/indexmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
}

func (b box) Value() int {
	return b.value
}

func (b *box) Add(v int) {
	b.value = b.value + v
}

func main() {
	items := []box{{value: 40}, {value: 2}}
	items[0].Add(1)
	if items[0].Value() + []box{{value: 1}}[0].Value() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedSliceIndexedReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedindexmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
}

type boxes []box

func (b box) Value() int {
	return b.value
}

func main() {
	if (boxes{{value: 41}})[0].Value() == 41 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedmethods/pkg/dep"

func main() {
	var b dep.Buffer
	p := &b
	p.WriteString("PASS")
	if p.Len() == 4 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Buffer struct {
	buf []byte
}

func (b *Buffer) WriteString(s string) int {
	b.buf = append(b.buf, []byte(s)...)
	return len(s)
}

func (b *Buffer) Len() int {
	return len(b.buf)
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedDirectMethodExpressionFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedmethodexpressions\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedmethodexpressions/pkg/dep"

func main() {
	if dep.Buffer.Len(dep.Buffer{Value: 42}) == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Buffer struct {
	Value int
}

func (b Buffer) Len() int {
	return b.Value
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedStaticMethodValueFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedmethodvalues\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedmethodvalues/pkg/dep"

func applyLen(f func() int) int { return f() }
func applyAdd(f func(int) int, v int) int { return f(v) }

func main() {
	b := dep.Buffer{Value: 42}
	f := b.Len
	_ = b.Len
	_ = f
	other := dep.Buffer{Value: 2}
	if f() + applyLen(f) + applyLen(b.Len) + applyLen(dep.Buffer{Value: 3}.Len) + applyAdd(other.Add, 4) == 135 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Buffer struct {
	Value int
}

func (b Buffer) Len() int {
	return b.Value
}

func (b *Buffer) Add(v int) int {
	b.Value = b.Value + v
	return b.Value
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedIndexedReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedindexmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedindexmethods/pkg/dep"

func main() {
	items := []dep.Box{{Value: 40}, {Value: 2}}
	items[0].Add(1)
	if items[0].Read() + []dep.Box{{Value: 1}}[0].Read() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Box struct {
	Value int
}

func (b Box) Read() int {
	return b.Value
}

func (b *Box) Add(v int) {
	b.Value = b.Value + v
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPointerSliceIndexedReceiverMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/ptrindexmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
}

func (b box) Read() int {
	return b.value
}

func (b *box) Add(v int) {
	b.value = b.value + v
}

func main() {
	items := []*box{&box{value: 40}, &box{value: 2}}
	items[0].Add(1)
	if items[0].Read() + []*box{&box{value: 1}}[0].Read() == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedCompositeLiteralMethodFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedlitmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedlitmethods/pkg/dep"

func main() {
	if (dep.Point{X: 3, Y: 4}).Sum() != 7 {
		print("FAIL\n")
		return
	}
	if (&dep.Counter{N: 42}).Value() != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Point struct {
	X int
	Y int
}

func (p Point) Sum() int {
	return p.X + p.Y
}

type Counter struct {
	N int
}

func (c *Counter) Value() int {
	return c.N
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestTopLevelNameListsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namelists\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/namelists/pkg/dep"

func main() {
	dep.C = 3
	dep.D = 4
	if dep.Sum() == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

const A, B = 1, 2
var C, D int

func Sum() int {
	return A + B + C + D
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestIotaFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/iotafrontend\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/iotafrontend/pkg/flags"

func main() {
	if flags.Mask() == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/flags/flags.go", `package flags

const (
	Read = 1 << iota
	Write
	Exec
)

const (
	Offset = 10 + iota
	Next
)

func Mask() int {
	return Read + Exec + Next - Offset
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestHexFloatFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/hexfloat\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	a := 0x1.8p+1
	b := 0x1.4p+2
	if a+b == 8.0 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestRawStringFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/rawstrings\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", "package main\n\nimport `example.com/rawstrings/pkg/msg`\n\nfunc main() {\n\tif msg.Value() == 71 {\n\t\tprint(msg.Text())\n\t\treturn\n\t}\n\tprint(\"FAIL\\n\")\n}\n")
	writeFixtureFile(t, fixture, "pkg/msg/msg.go", "package msg\n\nfunc Text() string {\n\treturn `PASS\n`\n}\n\nfunc Value() int {\n\treturn 077 + 0o10\n}\n")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStructTagsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/structtags\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", "package main\n\ntype Record struct {\n\tName string `json:\"name\"`\n\tCount int \"db:\\\"count\\\"\"\n}\n\nfunc main() {\n\tr := Record{Name: \"PASS\\n\", Count: 1}\n\tif r.Count == 1 {\n\t\tprint(r.Name)\n\t\treturn\n\t}\n\tprint(\"FAIL\\n\")\n}\n")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalAnonymousStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructlocal\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	var zero struct{ A int }
	one := struct{ A int }{A: 1}
	var two = struct{ A int }{2}
	var three struct{ A int } = struct{ A int }{3}
	rows := []struct{ A int }{{A: 3}}
	if zero.A + one.A + two.A + three.A + rows[0].A == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAnonymousStructFieldsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructfields\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
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
	if h.One.A + h.Two.A + h.Three.A + h.Rows[0].A + h.More[0].A + h.Extra[0].A + h.Ptr.A + h.Left.A + h.Right.A + field0.A + field1.A + more.A + other.A + double.A + doubleLeft.A + doubleRight.A + triple.A + tripleLeft.A + tripleRight.A == 178 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAnonymousStructSignaturesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructsignatures\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	one := take(struct{ A int }{1})
	two := give()
	three := round(struct{ A int }{3})
	four := named()
	if one + two.A + three.A + four.A == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestTopLevelAnonymousStructVariablesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructglobals\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var zero struct{ A int }
var one struct{ A int } = struct{ A int }{1}
var two = struct{ A int }{A: 2}
var (
	three struct{ A int }
	four = struct{ A int }{4}
)

func main() {
	three = struct{ A int }{3}
	if zero.A + one.A + two.A + three.A + four.A == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAnonymousStructNamedSlicesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructslices\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Rows []struct{ A int }
type MoreRows []struct{ A int }

var global Rows = Rows{{A: 1}}

func main() {
	rows := Rows{{A: 2}, {3}}
	more := MoreRows{{A: 4}}
	if global[0].A + rows[0].A + rows[1].A + more[0].A == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAnonymousStructAliasesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/anonstructaliases\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Alias = struct{ A int }
type MoreAlias = struct{ A int }

var global Alias = Alias{1}

func main() {
	local := MoreAlias{A: 2}
	if global.A + local.A == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImplicitCompositeLiteralsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/implicitcomposite\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type pair struct {
	left int
	right int
}

type box struct {
	first pair
	pairs []pair
}

func main() {
	pairs := []pair{{left: 1, right: 2}, {3, 4}}
	boxed := box{first: pair{left: 8, right: 9}, pairs: []pair{{10, 11}}}
	total := pairs[0].left + pairs[1].right
	total += boxed.first.left + boxed.first.right + boxed.pairs[0].left + boxed.pairs[0].right
	if total == 43 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestKeyedSliceLiteralsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/keyedslices\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type scores []int

var global = []int{2: 9}

func sum(values []int) int {
	total := 0
	for i, value := range values {
		total = total + i + value
	}
	return total
}

func main() {
	values := []int{0: 1, 2: 3}
	mixed := []int{1, 3: 4, 5}
	named := scores{1: 7}
	total := sum(global) + sum(values) + sum(mixed) + sum(named)
	if total == 43 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDirectSliceLiteralIndexFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/slicelitindex\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	if []int{41, 42}[1] != 42 {
		print("FAIL\n")
		return
	}
	if ([]byte{65, 66})[0] != byte('A') {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDirectStructLiteralSelectorFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/structlitselector\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
}

func want(x int) int { return x }

func main() {
	if (box{value: 42}).value != 42 {
		print("FAIL\n")
		return
	}
	if (&box{value: 43}).value != 43 {
		print("FAIL\n")
		return
	}
	if want(box{value: 40}.value) != 40 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMultiValueAppendFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/multiappend\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	xs := []int{10, 20, 30}
	ys := xs[:1]
	ys = append(ys, 99, xs[1])
	if len(ys) == 3 && ys[1] == 99 && ys[2] == 20 && xs[1] == 99 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestVariadicFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/variadic\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func sum(base int, values ...int) int {
	total := base
	for i := 0; i < len(values); i++ {
		total += values[i]
	}
	return total
}

type box struct {
	total int
}

func (b *box) add(base int, values ...int) {
	b.total += base
	for i := 0; i < len(values); i++ {
		b.total += values[i]
	}
}

func main() {
	values := []int{2, 3}
	b := &box{}
	b.add(1, values...)
	got := sum(4, values...) + sum(5, 6, 7) + b.total
	if got == 33 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestExpressionlessSwitchFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/exprswitch\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	x := 2
	switch {
	case x == 1:
		print("FAIL\n")
	case x == 2:
		print("PASS\n")
	default:
		print("FAIL\n")
	}
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSimpleRangeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/rangeloops\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type items []int
type box struct {
	values []int
}

func countItems(values items) int {
	return len(values)
}

func makeBox() box {
	return box{values: []int{8, 9}}
}

func main() {
	values := []int{1, 2, 3}
	total := 0
	for i, v := range values {
		total = total + i + v
	}
	for i := range "abc" {
		total = total + i
	}
	for _, v := range []int{4, 5} {
		total = total + v
	}
	for _, v := range (items{6, 7}) {
		total = total + v
	}
	for _, v := range makeBox().values {
		total = total + v
	}
	for _, v := range items([]int{10, 11}) {
		total = total + v
	}
	converted := items([]int{12, 13})
	total = total + len(converted) + countItems(items([]int{14, 15}))
	if total == 76 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalArrayFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/localarrays\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	var zero [3]int
	values := [3]int{1, 2}
	keyed := [4]int{1: 7, 3: 9}
	inferred := [...]int{5, 6}
	var explicit [2]int = [2]int{8}

	total := len(zero) + cap(values) + values[2] + keyed[1] + keyed[2] + keyed[3]
	total = total + len(inferred) + inferred[1] + explicit[1]
	total = total + len([2]int{8}) + cap([2]int{8})
	for i, v := range values {
		total = total + i + v
	}
	if total == 40 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalArrayAssignmentCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/localarraycopy\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	original := [3]int{1, 2, 3}
	copy := original
	copy[0] = 9
	if original[0] == 1 && copy[0] == 9 && len(copy) == 3 && cap(copy) == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayParameterCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arrayparamcopy\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func mutate(values [3]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}

func main() {
	original := [3]int{1, 2, 3}
	changed := mutate(original)
	if original[0] == 1 && changed == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedArrayParameterCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedarrayparamcopy\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedarrayparamcopy/pkg/dep"

func main() {
	original := [3]int{1, 2, 3}
	changed := dep.Mutate(original)
	if original[0] == 1 && changed == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

func Mutate(values [3]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedTopLevelArrayValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedtoparray\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedtoparray/pkg/dep"

func main() {
	copy := dep.Values
	copy[0] = 8
	same := dep.Values == [3]int{1, 2, 3}
	changed := dep.Mutate(dep.Values)
	if same && dep.Values[0] == 1 && copy[0] == 8 && changed == 15 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var Values [3]int = [3]int{1, 2, 3}

func Mutate(values [3]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedArrayStructFieldCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedarrayfieldcopies\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedarrayfieldcopies/pkg/dep"

func main() {
	dep.Reset()
	copy := dep.Global.Values
	copy[0] = 8
	target := dep.Box{}
	target.Values = dep.Global.Values
	target.Values[0] = 7
	replacement := [2]int{5, 6}
	dep.Global.Values = replacement
	same := dep.Global.Values == [2]int{5, 6}
	replacement[0] = 99
	changed := dep.Mutate(dep.Global.Values)
	if same && dep.Global.Values[0] == 5 && copy[0] == 8 && target.Values[0] == 7 && replacement[0] == 99 && changed == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

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
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedArrayLiteralSideEffectsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/discardedarrays\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var total int

func step(v int) int {
	total = total*10 + v
	return v
}

func main() {
	_ = [3]int{step(1), 2, step(3)}
	_, _ = [...]int{1: step(4)}, [2][1]int{{step(5)}, {6}}
	if total == 1345 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedMapLiteralFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/discardedmaps\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	total := len(map[string]int{"a": 1, "b": -2})
	total = total + len(map[string]map[string]int{"outer": map[string]int{"inner": 7}})
	total = total + len((map[byte]string{'x': "ok"}))
	total = total + len(make(map[string]int))
	total = total + len(make(map[string]int, 4))
	total = total + len((make(map[byte]string, 0x10)))
	total = total + map[string]int{"a": 1, "b": -2}["a"]
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
	if total == 41 && text == "ok" && empty == "" && emptyMake == "" && emptyMakeParen == "" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedMapLiteralDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/discardedmapcalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var text string

func emit(part string) int {
	text = text + part
	return len(part)
}

func main() {
	_ = map[string]int{"a": emit("P"), "b": emit("A")}
	_, _ = map[int]int{1: emit("S")}, map[string]map[string]int{"outer": map[string]int{"inner": emit("S")}}
	_ = []map[string]int{{"x": emit("!"), "y": emit("?")}}
	_ = []map[string]int{make(map[string]int, emit("~"))}
	if text == "PASS!?~" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapLiteralDeleteDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapdeletecalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var text string

func emit(part string) int {
	text = text + part
	return len(part)
}

func main() {
	delete(map[string]int{"a": emit("P"), "b": emit("A")}, "a")
	delete(map[string]map[string]int{"outer": map[string]int{"inner": emit("S")}}, "outer")
	fn := func() {
		delete(map[string]int{"inner": emit("S")}, "inner")
	}
	fn()
	if text == "PASS" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapLiteralLenDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/maplencalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var total int

func step(v int) int {
	total = total*10 + v
	return v
}

func main() {
	length := len(map[string]int{"a": step(1), "b": step(2)})
	length = length + len(map[string]map[string]int{"outer": map[string]int{"inner": step(3)}})
	length = length + len(map[string][2]int{"array": [2]int{step(4), 5}})
	fn := func() int {
		return len(map[string]int{"literal": step(5)})
	}
	length = length + fn()
	if total == 12345 && length == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapLiteralIndexDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapindexcalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var total int

func step(v int) int {
	total = total*10 + v
	return v
}

func main() {
	value := map[string]int{"miss": step(1), "hit": step(2), "tail": step(3)}["hit"]
	missing := map[string]int{"missing": step(4)}["absent"]
	fn := func() int {
		return map[string]int{"inner": step(5)}["inner"]
	}
	value = value + missing + fn()
	if total == 12345 && value == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapLiteralCommaOkDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapcommaokcalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var total int

func step(v int) int {
	total = total*10 + v
	return v
}

func main() {
	found, ok := map[string]int{"miss": step(1), "hit": step(2), "tail": step(3)}["hit"]
	missing, missingOK := map[string]int{"missing": step(4)}["absent"]
	if cond, condOK := map[string]int{"cond": step(5)}["cond"]; condOK {
		found = found + cond
	}
	fn := func() int {
		value, valueOK := map[string]int{"inner": step(6)}["inner"]
		if valueOK {
			return value
		}
		return 0
	}
	found = found + missing + fn()
	if total == 123456 && ok && !missingOK && found == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPureMapRangeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/maprange\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	if total == 129 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
	`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapRangeDirectCallValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/maprangecalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var trace string

func emit(part string, value int) int {
	trace = trace + part
	return value
}

func main() {
	total := 0
	for k, v := range map[string]int{"a": emit("A", 1), "bb": emit("B", 2)} {
		total = total + len(k) + v
	}
	for k := range map[string]int{"ccc": emit("C", 3)} {
		total = total + len(k)
	}
	fn := func() int {
		for _, v := range map[string]int{"d": emit("D", 4)} {
			return v
		}
		return 0
	}
	total = total + fn()
	if trace == "ABCD" && total == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestMapMakeDirectCallCapacityFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapmakecalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var trace string

func emit(part string, value int) int {
	trace = trace + part
	return value
}

func main() {
	_ = make(map[string]int, emit("A", 1))
	delete(make(map[string]int, emit("B", 2)), "missing")
	total := len(make(map[string]int, emit("C", 3)))
	total = total + make(map[string]int, emit("D", 4))["missing"]
	value, ok := make(map[string]int, emit("E", 5))["missing"]
	for k, v := range make(map[string]int, emit("F", 6)) {
		total = total + len(k) + v
	}
	fn := func() int {
		return len(make(map[string]int, emit("G", 7)))
	}
	total = total + value + fn()
	if trace == "ABCDEFG" && total == 0 && !ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPureStaticMapAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapalias\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	if total == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedMapTypeStaticAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedmapalias\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Table map[string]int

func main() {
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
	total = total + direct + len(empty)
	if total == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedNamedMapTypeStaticAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importednamedmapalias\n")
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Table map[string]int
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importednamedmapalias/pkg/dep"

func main() {
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
	total = total + direct + len(empty)
	if total == 24 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMapAliasLimitationProbesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapaliasprobes\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	_ = make(map[string]int)
	indexed := map[string]int{"a": 1}
	if indexed["a"] != 1 {
		print("FAIL\n")
		return
	}
	assigned := map[string]int{}
	assigned["a"] = 1
	if assigned["a"] != 1 {
		print("FAIL\n")
		return
	}
	missing := map[string]int{}
	_, missingOK := missing["a"]
	if missingOK {
		print("FAIL\n")
		return
	}
	deleted := map[string]int{}
	delete(deleted, "a")
	measured := map[string]int{}
	if len(measured) != 0 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedMapAnonymousStructFieldCompositeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapanonstruct\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Items map[string]struct{ A int }
}

type (
	Holder struct {
		Rows []map[int]struct{ B string }
	}
)

func main() {
	_, _ = Box{}, (Holder{})
	type Local struct {
		Table map[string]struct{ Value int }
	}
	_ = Local{}
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMapAliasDirectCallAssignmentsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapaliascalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var trace string

func emit(part string, value int) int {
	trace = trace + part
	return value
}

func word(part string, value string) string {
	trace = trace + part
	return value
}

func main() {
	local := 3
	m := map[string]int{"a": 1}
	m["b"] = emit("A", 2)
	m["c"] = local
	m["d"] = emit("B", 5) * emit("C", 6) - local
	m["e"] = (emit("D", 1) << 3) | emit("E", 1)
	total := m["a"] + m["b"] + m["c"] + m["d"] + m["e"]
	value, ok := m["b"]
	for _, item := range m {
		total = total + item
	}
	fn := func() int {
		inner := map[string]int{}
		inner["x"] = emit("F", 4)
		return inner["x"]
	}
	total = total + value + fn()
	suffix := "!"
	words := map[string]string{"a": "A"}
	words["b"] = word("G", "B") + word("H", "C") + suffix
	words["c"] = "D" + suffix
	got, okText := words["b"]
	text := words["a"] + got + words["c"]
	flags := map[string]bool{"base": true}
	flags["match"] = emit("I", 2) == emit("J", 2)
	flags["ready"] = emit("K", 1) != emit("L", 2)
	match, matchOK := flags["match"]
	ready, readyOK := flags["ready"]
	if trace == "ABCDEFGHIJKL" && total == 90 && ok && okText && text == "ABC!D!" && match && ready && matchOK && readyOK {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMapAliasCompoundAssignmentsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapaliascompound\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var trace string

func emit(part string, value int) int {
	trace += part
	return value
}

func word(part string, value string) string {
	trace += part
	return value
}

func main() {
	m := map[string]int{"a": 1}
	m["a"] += 2
	m["b"] += emit("A", 3)
	m["a"] *= 4
	m["b"] -= 1
	m["a"]++
	m["b"]--
	words := map[string]string{"x": "A"}
	words["x"] += word("B", "B")
	words["y"] += word("C", "C")
	value, ok := m["b"]
	got, okText := words["x"]
	missing, missingOK := words["y"]
	if trace == "ABC" && m["a"]+value == 14 && ok && okText && missingOK && got == "AB" && missing == "C" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticMapAliasDirectCallInitializersFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/mapaliasinitcalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var trace string

func emit(part string, value int) int {
	trace = trace + part
	return value
}

func main() {
	local := 3
	m := map[string]int{"a": emit("A", 1), "b": local}
	empty := make(map[string]int, emit("B", 2))
	total := len(empty) + m["a"] + m["b"]
	value, ok := m["a"]
	for _, item := range m {
		total = total + item
	}
	fn := func() int {
		inner := map[string]int{"x": emit("C", 4)}
		return inner["x"]
	}
	total = total + value + fn()
	if trace == "ABC" && total == 13 && ok {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayStructFieldFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arrayfields\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Values [3]int
	Bytes [2]byte
	Flags [2]bool
}

func main() {
	b := Box{
		Values: [3]int{1, 2},
		Bytes: [2]byte{'A'},
		Flags: [2]bool{1: true},
	}
	total := len(b.Values) + cap(b.Bytes) + b.Values[0] + b.Values[2] + int(b.Bytes[0])
	for i, v := range b.Values {
		total = total + i + v
	}
	if b.Flags[1] {
		total = total + 1
	}
	if total == 77 && b.Values == [3]int{1, 2} && b.Bytes != [2]byte{'B'} {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayStructFieldCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arrayfieldcopies\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Values [3]int
}

func mutate(values [3]int) int {
	values[1] = 8
	return values[1] + len(values) + cap(values)
}

func main() {
	b := Box{Values: [3]int{1, 2, 3}}
	copy := b.Values
	copy[0] = 9
	assigned := [3]int{4, 5, 6}
	b.Values = assigned
	assigned[1] = 99
	other := Box{Values: [3]int{7, 8, 9}}
	target := Box{}
	target.Values = other.Values
	other.Values[0] = 88
	changed := mutate(target.Values)
	if copy[0] == 9 && copy[1] == 2 && b.Values[0] == 4 && b.Values[1] == 5 && assigned[1] == 99 && target.Values[0] == 7 && other.Values[0] == 88 && changed == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayNestedStructFieldCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arraynestedfieldcopies\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct {
	Values [3]int
}

type Outer struct {
	Inner Inner
}

func mutate(values [3]int) int {
	values[2] = 7
	return values[2] + len(values) + cap(values)
}

func main() {
	outer := Outer{Inner: Inner{Values: [3]int{1, 2, 3}}}
	copy := outer.Inner.Values
	copy[0] = 9
	target := Outer{}
	target.Inner.Values = outer.Inner.Values
	outer.Inner.Values[1] = 88
	changed := mutate(target.Inner.Values)
	if copy[0] == 9 && copy[1] == 2 && target.Inner.Values[0] == 1 && target.Inner.Values[1] == 2 && outer.Inner.Values[1] == 88 && changed == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayPromotedStructFieldCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arraypromotedfieldcopies\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct {
	Values [3]int
}

type Outer struct {
	Inner
}

func mutate(values [3]int) int {
	values[2] = 7
	return values[2] + len(values) + cap(values)
}

func main() {
	outer := Outer{Inner: Inner{Values: [3]int{1, 2, 3}}}
	copy := outer.Values
	copy[0] = 9
	target := Outer{}
	target.Values = outer.Values
	outer.Values[1] = 88
	changed := mutate(target.Values)
	if copy[0] == 9 && copy[1] == 2 && target.Values[0] == 1 && target.Values[1] == 2 && outer.Values[1] == 88 && changed == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayParameterFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arrayparams\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func first(values [2]byte) int {
	return int(values[0]) + len(values)
}

func main() {
	values := [3]int{1, 2}
	total := sum(values) + sum([3]int{4, 5}) + pair([2]int{3}, [2]int{1: 4}) + first([2]byte{'A'})
	if total == 108 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedArraySignatureFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedarraysignatures\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	original := Values{1, 2, 3}
	changed := mutate(original)
	values := makeValues()
	echoed := echo(original)
	if original == (Values{1, 2, 3}) && changed == 15 && values == (Values{1, 2, 3}) && echoed == (Values{1, 8, 3}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedNamedArraySignatureFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importednamedarraysignatures\n")
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Values [3]int

func Mutate(values Values) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}

func MakeValues() Values {
	return Values{1, 2, 3}
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importednamedarraysignatures/pkg/dep"

func main() {
	original := dep.Values{1, 2, 3}
	changed := dep.Mutate(original)
	values := dep.MakeValues()
	if original == (dep.Values{1, 2, 3}) && changed == 15 && values == (dep.Values{1, 2, 3}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arraycomparisons\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func same(left, right [2]int) bool {
	return left == right
}

var seq int

func next(want int, value int) int {
	seq = seq + 1
	if seq != want {
		return -1000
	}
	return value
}

func leftArray() [2]int {
	return [2]int{next(1, 1), next(2, 2)}
}

func rightArray() [2]int {
	return [2]int{next(3, 1), next(4, 2)}
}

func main() {
	left := [2]int{1, 2}
	right := [...]int{1, 3}
	bytes := [2]byte{'A', 'B'}
	callValues := leftArray() == rightArray()
	literalValues := [2]int{next(5, 1), next(6, 2)} == [2]int{next(7, 1), next(8, 2)}
	if same(left, [2]int{1, 2}) && left == [2]int{1, 2} && left != right && [2]int{1, 2} == [...]int{1, 2} && bytes == [2]byte{'A', 'B'} && callValues && literalValues && seq == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSimpleStructComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/structcomparisons\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Value int
	Name string
	OK bool
	Ptr *int
	Values [2]int
	Inner Inner
}

type Inner struct {
	Count int
	Tag string
	Flags [2]bool
}

var globalLeft = Box{Value: 1, Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
var globalRight = Box{Value: 1, Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}

var seq int

func next(want int, value int) int {
	seq = seq + 1
	if seq != want {
		return -1000
	}
	return value
}

func same(left Box, right Box) bool {
	return left == right
}

func makeBox(want int, value int) Box {
	return Box{Value: next(want, value), Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
}

func main() {
	x := 1
	y := 1
	left := Box{Value: 1, Name: "a", OK: true, Ptr: &x, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
	sameLocal := Box{Value: 1, Name: "a", OK: true, Ptr: &x, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
	differentPtr := Box{Value: 1, Name: "a", OK: true, Ptr: &y, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
	differentValues := Box{Value: 1, Name: "a", OK: true, Ptr: &x, Values: [2]int{1, 3}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
	differentInner := Box{Value: 1, Name: "a", OK: true, Ptr: &x, Values: [2]int{1, 2}, Inner: Inner{Count: 4, Tag: "b", Flags: [2]bool{true, false}}}
	callValues := makeBox(1, 1) == makeBox(2, 1)
	literalValues := Box{Value: next(3, 1), Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}} == Box{Value: next(4, 1), Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}
	if same(left, sameLocal) && left == sameLocal && !(left != sameLocal) && left != differentPtr && left != differentValues && left != differentInner && globalLeft == globalRight && callValues && literalValues && (Box{Value: next(5, 1), Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}) == (Box{Value: next(6, 1), Name: "a", OK: true, Ptr: nil, Values: [2]int{1, 2}, Inner: Inner{Count: 3, Tag: "b", Flags: [2]bool{true, false}}}) && makeBox(7, 1) == makeBox(8, 1) && seq == 8 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedStructComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedstructcomparisons\n")
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Box struct {
	Value int
	Name string
}

var Left = Box{Value: 1, Name: "a"}
var Right = Box{Value: 1, Name: "a"}

var Seq int

func Make(want int, value int) Box {
	Seq = Seq + 1
	if Seq != want {
		return Box{Value: -1000, Name: "bad"}
	}
	return Box{Value: value, Name: "a"}
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedstructcomparisons/pkg/dep"

func main() {
	globalSame := dep.Left == dep.Right
	callSame := dep.Make(1, 1) == dep.Make(2, 1)
	literalSame := dep.Box{Value: 1, Name: "a"} == dep.Box{Value: 1, Name: "a"}
	if globalSame && callSame && literalSame && (dep.Box{Value: 2, Name: "b"}) == (dep.Box{Value: 2, Name: "b"}) && dep.Seq == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedArrayStructFieldComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedarraystructcomparisons\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Values [2]int

type Box struct {
	Values Values
}

var globalLeft = Box{Values: Values{1, 2}}
var globalRight = Box{Values: Values{1, 2}}

func same(left Box, right Box) bool {
	return left == right
}

func main() {
	left := Box{Values: Values{1, 2}}
	right := Box{Values: Values{1, 2}}
	different := Box{Values: Values{1, 3}}
	if same(left, right) && left == right && !(left != right) && left != different && globalLeft == globalRight {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestEmptyStructComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/emptystructcomparisons\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Empty struct{}

var globalLeft = Empty{}
var globalRight = Empty{}

func same(left Empty, right Empty) bool {
	return left == right
}

func main() {
	left := Empty{}
	right := Empty{}
	if same(left, right) && left == right && !(left != right) && globalLeft == globalRight {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestIndexedArrayStructFieldComparisonFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/indexedarrayfieldcomparisons\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Values [2]int
}

func main() {
	first := Box{Values: [2]int{1, 2}}
	second := Box{Values: [2]int{1, 2}}
	third := Box{Values: [2]int{3, 4}}
	boxes := []Box{first, second, third}
	same := boxes[0].Values == boxes[1].Values
	different := boxes[0].Values != boxes[2].Values
	if same && different {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestIndexedArrayStructFieldCopiesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/indexedarrayfieldcopies\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Values [2]int
}

func mutate(values [2]int) int {
	values[0] = 9
	return values[0] + len(values) + cap(values)
}

func main() {
	first := Box{Values: [2]int{1, 2}}
	second := Box{Values: [2]int{3, 4}}
	boxes := []Box{first, second}
	copy := boxes[0].Values
	copy[0] = 8
	target := Box{}
	target.Values = boxes[1].Values
	target.Values[0] = 7
	replacement := [2]int{5, 6}
	boxes[0].Values = replacement
	replacement[0] = 99
	boxes[1].Values[0] = 6
	changed := mutate(boxes[1].Values)
	if boxes[0].Values[0] == 5 && boxes[1].Values[0] == 6 && copy[0] == 8 && target.Values[0] == 7 && replacement[0] == 99 && changed == 13 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestArrayResultFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/arrayresults\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func values() [3]int {
	return [3]int{1, 2}
}

func sum(values [3]int) int {
	return len(values) + cap(values) + values[0] + values[1] + values[2]
}

func zeroBytes() (out [2]byte) {
	return
}

func explicitBytes() (out [2]byte) {
	return [2]byte{'A'}
}

func main() {
	vals := values()
	zeros := zeroBytes()
	bytes := explicitBytes()
	total := sum(vals) + len(zeros) + cap(zeros) + int(zeros[0]) + len(bytes) + cap(bytes) + int(bytes[0])
	if total == 82 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestEmbeddedStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/embeddedstructs\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct { X int }
type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func main() {
	outer := Outer{Inner: Inner{X: 2}}
	nested := Nested{Mid{Inner{X: 3}}}
	pointer := PointerOuter{Inner: &Inner{X: 4}}
	outer.X = outer.X + 1
	pointer.X = pointer.X + outer.X
	total := outer.X + nested.X + pointer.X
	total = total + Outer{Inner{X: 5}}.X + Nested{Mid{Inner{X: 6}}}.X + (PointerOuter{&Inner{X: 7}}).X
	if total == 35 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPromotedEmbeddedMethodsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/promotedmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct { X int }

func (in Inner) Value() int { return in.X }

func (in *Inner) Add(v int) int {
	in.X = in.X + v
	return in.X
}

type Mid struct { Inner }
type Outer struct { Inner }
type Nested struct { Mid }
type PointerOuter struct { *Inner }

func main() {
	outer := Outer{Inner: Inner{X: 2}}
	nested := Nested{Mid: Mid{Inner: Inner{X: 3}}}
	pointer := PointerOuter{Inner: &Inner{X: 4}}
	total := outer.Value() + outer.Add(5) + nested.Value() + pointer.Value() + pointer.Add(6)
	if total == 26 && outer.X == 7 && pointer.X == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPromotedCompositeLiteralMethodsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/promotedlitmethods\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Inner struct { X int }

func (in Inner) Value() int { return in.X }

func (in *Inner) Add(v int) int {
	in.X = in.X + v
	return in.X
}

type Outer struct { Inner }
type PointerOuter struct { *Inner }

func main() {
	total := Outer{Inner: Inner{X: 2}}.Value()
	total = total + PointerOuter{Inner: &Inner{X: 4}}.Add(6)
	total = total + (&Outer{Inner: Inner{X: 7}}).Add(8)
	if total == 27 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedEmbeddedStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedembedded\n")
	writeFixtureFile(t, fixture, "dep/dep.go", `package dep

type Inner struct {
	Count int
}

type Outer struct {
	Inner
}

var Default = Outer{Inner: Inner{Count: 4}}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedembedded/dep"

func main() {
	o := dep.Outer{Inner: dep.Inner{Count: 2}}
	total := o.Count + dep.Outer{Inner: dep.Inner{Count: 3}}.Count + dep.Default.Count
	if total == 9 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedPromotedEmbeddedMethodsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedpromotedmethods\n")
	writeFixtureFile(t, fixture, "dep/dep.go", `package dep

type Inner struct {
	Count int
}

func (in Inner) Value() int {
	return in.Count
}

type Outer struct {
	Inner
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedpromotedmethods/dep"

func main() {
	o := dep.Outer{Inner: dep.Inner{Count: 7}}
	if o.Value() == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedFunctionTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertfunctype\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type F func(int) int
type G = func() (int, int)
type (
	H func(string)
	I = func() bool
)

func main() {
	type Local func() int
	type (
		LocalPair func(int) int
		LocalAlias = func() (int, int)
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertFunctionContainingTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertfunccontaining\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct { Fn func(int) int }
type Handler struct { Fn func() (int, int) }
type Callbacks []func(string) bool
type Alias = struct { Fn func() }
type Rows []struct { Fn func() int }
type (
	Holder struct { Fn func() }
	List []func(int) int
)

func main() {
	type LocalBox struct { Fn func(int) int }
	type LocalCallbacks []func(string) bool
	type LocalRows []struct { Fn func() int }
	type (
		LocalHolder struct { Fn func() }
		LocalList []func(int) int
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedInterfaceTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertinterface\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Empty interface{}
type Reader = interface { Read([]byte) int }
type EmbeddedBase interface { Base() int }
type EmbeddedChild interface { EmbeddedBase }
type (
	Closer interface { Close() int }
	Seeker = interface { Seek(int64, int) int64 }
)

func main() {
	type Local interface{}
	type LocalBase interface { Base() int }
	type LocalChild interface { LocalBase }
	type (
		LocalReader interface { Read([]byte) int }
		LocalCloser = interface { Close() int }
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertInterfaceContainingTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertinterfacecontaining\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	type LocalBox struct { Value interface{} }
	type LocalAnyBox struct { Value any }
	type LocalValues []interface{}
	type LocalAnyValues []any
	type (
		LocalHolder struct { Value interface{} }
		LocalAnyHolder struct { Value any }
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestBlankDiscardedInterfaceVarsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/blankdiscardedinterface\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	var value interface{}
	_ = value
	var other any
	_ = other
	var nilValue interface{} = nil
	_ = nilValue
	var nilOther any = nil
	_ = nilOther
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNilInterfaceVarComparisonsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nilinterfacecomparison\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	var value interface{}
	if value != nil {
		print("FAIL value\n")
		return
	}
	var other any = nil
	if nil != other {
		print("FAIL other\n")
		return
	}
	if value == nil && nil == other {
		print("PASS\n")
		return
	}
	print("FAIL compare\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestUnusedInterfaceParametersFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/unusedinterfaceparams\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var total int

func bump() int {
	total = total + 5
	return total
}

func wrap(value int) int {
	total = total + 7
	return value
}

func flag() bool {
	total = total + 100
	return true
}

func use(value interface{}) {}
func useAny(value any, keep int) int { return keep }
func grouped(left, right interface{}, keep int) int { return keep }
func check(value interface{}, keep bool) bool { return keep }
func choose(value interface{}, keep int) int { return keep }

func returned() int {
	return useAny(wrap(bump() + 1), 4)
}

func main() {
	use(wrap(bump() + 1))
	use(false && flag())
	use(true || flag())
	use(true && flag())
	use(1)
	useAny("x", 2)
	assigned := useAny(wrap(bump() + 1), 6)
	assigned = useAny(false && flag(), assigned)
	assigned = useAny(true || flag(), assigned)
	var declared = useAny(wrap(bump() + 1), 8)
	var declaredTyped int = useAny(false && flag(), declared)
	if check(wrap(bump() + 1), true) {
		total = total + 11
	}
	if check(false && flag(), true) {
		total = total + 13
	}
	if check(true || flag(), true) {
		total = total + 17
	}
	switch choose(wrap(bump() + 1), 2) {
	case 2:
		total = total + 19
	}
	switch choose(false && flag(), 3) {
	case 3:
		total = total + 23
	}
	switch choose(true || flag(), 4) {
	case 4:
		total = total + 29
	}
	loops := 0
	for check(wrap(bump() + 1), loops < 1) {
		total = total + 31
		loops = loops + 1
	}
	for check(false && flag(), loops < 2) {
		total = total + 37
		loops = loops + 1
	}
	for check(true || flag(), loops < 3) {
		total = total + 41
		loops = loops + 1
	}
	classic := 0
	for ; check(wrap(bump() + 1), classic < 1); {
		total = total + 43
		classic = classic + 1
	}
	for ; check(false && flag(), classic < 2); {
		total = total + 47
		classic = classic + 1
	}
	for ; check(true || flag(), classic < 3); {
		total = total + 53
		classic = classic + 1
	}
	mixed := 0
	for i := 0; check(wrap(bump() + 1), i < 1); i = i + 1 {
		total = total + 59
		mixed = mixed + 1
	}
	for ; check(false && flag(), mixed < 2); mixed = mixed + 1 {
		total = total + 61
	}
	for ; check(true || flag(), mixed < 3); mixed = mixed + 1 {
		total = total + 67
	}
	defer use(wrap(bump() + 1))
	defer use(false && flag())
	defer use(true || flag())
	defer use(true && flag())
	if total == 895 && assigned == 6 && declaredTyped == 8 && loops == 3 && classic == 3 && mixed == 3 && grouped(1, 2, 3) == 3 && returned() == 4 && total == 907 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
	`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedInterfaceReturnsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/discardedinterfacereturns\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	_ = value()
	_ = valueAnd()
	valueOr()
	_ = valueSum()
	valueBareAnd()
	_ = valueBareOr()
	valueAny()
	if total == 54 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "cmd/app/value.go", `package main

var total int

func bump() int {
	total = total + 1
	return 7
}

func wrap(value int) int {
	total = total + 10
	return value
}

func flag() bool {
	total = total + 100
	return true
}

func wrapBool(value bool) bool {
	total = total + 20
	return value
}

func value() interface{} {
	return wrap(bump() + 1)
}

func valueAnd() interface{} {
	return wrapBool(false && flag())
}

func valueOr() interface{} {
	return wrapBool(true || flag())
}

func valueSum() interface{} {
	return bump() + 1
}

func valueBareAnd() interface{} {
	return false && flag()
}

func valueBareOr() interface{} {
	return true || flag()
}

func valueAny() any {
	total = total + 2
	return "x"
}
	`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticInterfaceAssertionsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/staticinterfaceassertions\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Value int
}

type Other struct {
	Value int
}

func main() {
	var x interface{} = 7
	var text any = "ok"
	var flag interface{} = true
	var boxed interface{} = Box{Value: 41}
	var boxedPtr interface{} = &Box{Value: 42}
	value := x.(int)
	gotText, okText := text.(string)
	gotFlag, okFlag := flag.(bool)
	gotBox := boxed.(Box)
	gotBoxAgain, okBox := boxed.(Box)
	gotBoxPtr := boxedPtr.(*Box)
	gotBoxPtrAgain, okBoxPtr := boxedPtr.(*Box)
	missingText, okMissingText := x.(string)
	missingInt, okMissingInt := text.(int)
	missingFlag, okMissingFlag := flag.(int)
	missingBox, okMissingBox := boxed.(Other)
	missingBoxPtr, okMissingBoxPtr := boxedPtr.(*Other)
	if okText && okFlag && okBox && okBoxPtr && gotText == "ok" && gotFlag && gotBox.Value == 41 && gotBoxAgain.Value == 41 && gotBoxPtr.Value == 42 && gotBoxPtrAgain.Value == 42 && value == 7 && x.(int) == 7 && !okMissingText && missingText == "" && !okMissingInt && missingInt == 0 && !okMissingFlag && missingFlag == 0 && !okMissingBox && missingBox.Value == 0 && !okMissingBoxPtr && missingBoxPtr == nil {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNilStaticInterfaceAssertionsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/nilstaticinterfaceassertions\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	var empty interface{}
	value, okValue := empty.(int)
	var nilAny any = nil
	text, okText := nilAny.(string)
	flag, okFlag := empty.(bool)
	if value == 0 && !okValue && text == "" && !okText && !flag && !okFlag {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticInterfaceAssertionPanicsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/staticinterfaceassertionpanics\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

var score int
var assigned string

func cleanupString() {
	recover()
	score = score + 1
}

func cleanupBool() {
	recover()
	score = score + 2
}

func cleanupAssign() {
	recover()
	if assigned == "keep" {
		score = score + 4096
		return
	}
	print("FAIL\n")
}

func cleanupVar() {
	recover()
	score = score + 8
}

func cleanupCall() {
	recover()
	score = score + 16
}

func cleanupCallFirstArg() {
	recover()
	score = score + 1024
}

func cleanupDefer() {
	recover()
	score = score + 64
}

func cleanupDeferFirstArg() {
	recover()
	score = score + 2048
}

func cleanupIf() {
	recover()
	score = score + 128
}

func cleanupFor() {
	recover()
	score = score + 256
}

func cleanupSwitch() {
	recover()
	score = score + 512
}

func cleanupReturn() {
	recover()
	score = score + 4
}

func cleanupReturnPair() {
	recover()
	score = score + 8192
}

func discardMismatch() {
	defer cleanupString()
	var x interface{} = 7
	_ = x.(string)
	print("FAIL\n")
}

func shortMismatch() {
	defer cleanupBool()
	var x interface{} = 7
	value := x.(bool)
	_ = value
	print("FAIL\n")
}

func assignMismatch() {
	defer cleanupAssign()
	var x interface{} = 7
	assigned = "keep"
	assigned = x.(string)
	print("FAIL\n")
}

func varMismatch() {
	defer cleanupVar()
	var x interface{} = 7
	var value string = x.(string)
	_ = value
	print("FAIL\n")
}

func callMismatch() {
	defer cleanupCall()
	var x interface{} = 7
	takeString(x.(string))
}

func callFirstArgMismatch() {
	defer cleanupCallFirstArg()
	var x interface{} = 7
	takeTwo(x.(string), laterArg())
}

func takeString(value string) {
	print("FAIL\n")
}

func takeTwo(a string, b string) {
	print("FAIL\n")
}

func laterArg() string {
	print("FAIL\n")
	return "bad"
}

func deferMismatch() {
	defer cleanupDefer()
	var x interface{} = 7
	defer takeString(x.(string))
}

func deferFirstArgMismatch() {
	defer cleanupDeferFirstArg()
	var x interface{} = 7
	defer takeTwo(x.(string), laterArg())
}

func ifMismatch() {
	defer cleanupIf()
	var x interface{} = 7
	if x.(bool) {
		print("FAIL\n")
	}
	print("FAIL\n")
}

func forMismatch() {
	defer cleanupFor()
	var x interface{} = 7
	for x.(bool) {
		print("FAIL\n")
	}
	print("FAIL\n")
}

func switchMismatch() {
	defer cleanupSwitch()
	var x interface{} = 7
	switch x.(string) {
	case "ok":
		print("FAIL\n")
	}
	print("FAIL\n")
}

func returnMismatch() string {
	defer cleanupReturn()
	var x interface{} = 7
	return x.(string)
}

func returnPairMismatch() (string, string) {
	defer cleanupReturnPair()
	var x interface{} = 7
	return x.(string), laterArg()
}

func main() {
	discardMismatch()
	shortMismatch()
	assignMismatch()
	varMismatch()
	callMismatch()
	callFirstArgMismatch()
	deferMismatch()
	deferFirstArgMismatch()
	ifMismatch()
	forMismatch()
	switchMismatch()
	if returnMismatch() != "" {
		print("FAIL\n")
		return
	}
	a, b := returnPairMismatch()
	if len(a) != 0 {
		print("FAIL\n")
		return
	}
	if len(b) != 0 {
		print("FAIL\n")
		return
	}
	if score == 16351 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStaticInterfaceTypeSwitchesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/staticinterfacetypeswitches\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct {
	Value int
}

type Other struct {
	Value int
}

func main() {
	var x interface{} = 7
	var text any = "ok"
	var flag any = true
	var empty interface{}
	var nilAny any = nil
	var boxed any = Box{Value: 41}
	var boxedPtr any = &Box{Value: 42}
	score := 0
	switch x.(type) {
	case string:
		score = score + 100
	case int:
		value, ok := x.(int)
		if ok {
			score = score + value
		}
	default:
		score = score + 200
	}
	switch text.(type) {
	case bool:
		score = score + 300
	case string:
		score = score + 5
	default:
		score = score + 400
	}
	switch word := text.(type) {
	case string:
		if word == "ok" {
			score = score + 11
		}
	default:
		score = score + 500
	}
	switch unused := flag.(type) {
	case int:
		score = score + 600
	default:
		_ = unused
		score = score + 13
	}
	switch never := x.(type) {
	case string:
		_ = never
		score = score + 700
	}
	switch blank := empty.(type) {
	case int:
		score = score + 1500
	case nil:
		_ = blank
		score = score + 17
	default:
		score = score + 1600
	}
	switch fallback := nilAny.(type) {
	case int:
		score = score + 1700
	default:
		_ = fallback
		score = score + 19
	}
	switch boxed.(type) {
	case Other:
		score = score + 800
	case Box:
		got := boxed.(Box)
		score = score + got.Value
	default:
		score = score + 900
	}
	switch got := boxed.(type) {
	case Box:
		score = score + got.Value
	default:
		score = score + 1000
	}
	switch neverBox := boxed.(type) {
	case Other:
		_ = neverBox
		score = score + 1100
	}
	switch boxedPtr.(type) {
	case *Other:
		score = score + 1200
	case *Box:
		got := boxedPtr.(*Box)
		score = score + got.Value
	default:
		score = score + 1300
	}
	switch got := boxedPtr.(type) {
	case *Box:
		score = score + got.Value
	default:
		score = score + 1400
	}
	if score == 238 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedStaticInterfaceStructsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedstaticinterfacestructs\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Box struct {
	Value int
}

type Other struct {
	Value int
}
`)
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedstaticinterfacestructs/pkg/dep"

func main() {
	var boxed any = dep.Box{Value: 41}
	var boxedPtr any = &dep.Box{Value: 42}
	got := boxed.(dep.Box)
	gotAgain, okBox := boxed.(dep.Box)
	gotPtr := boxedPtr.(*dep.Box)
	gotPtrAgain, okPtr := boxedPtr.(*dep.Box)
	missing, okMissing := boxed.(dep.Other)
	missingPtr, okMissingPtr := boxedPtr.(*dep.Other)
	score := 0
	if okBox {
		score = score + got.Value + gotAgain.Value
	}
	if okPtr {
		score = score + gotPtr.Value + gotPtrAgain.Value
	}
	if !okMissing && missing.Value == 0 {
		score = score + 3
	}
	if !okMissingPtr && missingPtr == nil {
		score = score + 4
	}
	switch boxed.(type) {
	case dep.Other:
		score = score + 1000
	case dep.Box:
		fromSwitch := boxed.(dep.Box)
		score = score + fromSwitch.Value
	default:
		score = score + 2000
	}
	switch value := boxed.(type) {
	case dep.Box:
		score = score + value.Value
	default:
		score = score + 3000
	}
	switch boxedPtr.(type) {
	case *dep.Other:
		score = score + 4000
	case *dep.Box:
		fromPtrSwitch := boxedPtr.(*dep.Box)
		score = score + fromPtrSwitch.Value
	default:
		score = score + 5000
	}
	switch ptrValue := boxedPtr.(type) {
	case *dep.Box:
		score = score + ptrValue.Value
	default:
		score = score + 6000
	}
	if score == 339 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedAnyTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertany\n\ngo 1.18\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Alias any
type Other = any
type (
	Boxed any
	Named = any
)

func main() {
	type Local any
	type (
		LocalAlias any
		LocalOther = any
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedComplexTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertcomplex\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Small complex64
type Wide = complex128
type (
	LocalSmall complex64
	LocalWide = complex128
)

func main() {
	type Inner complex64
	type (
		InnerSmall complex64
		InnerWide = complex128
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertComplexContainingTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertcomplexcontaining\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

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

func main() {
	type LocalBox struct { Value complex64 }
	type LocalWideBox struct { Value complex128 }
	type LocalValues []complex64
	type LocalWideValues []complex128
	type LocalRows []struct { Value complex64 }
	type (
		LocalHolder struct { Value complex64 }
		LocalWideHolder struct { Value complex128 }
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedMapTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertmap\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Table map[string]int
type Nested = map[string]map[string]int
type Alias map[string]Table
type (
	Counts map[string]int
	Rows = map[int]Table
)

func main() {
	type Local map[string]int
	type LocalNested map[string]Local
	type (
		LocalCounts map[int]int
		LocalRows = map[string]LocalCounts
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertMapContainingTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertmapcontaining\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct { M map[string]int }
type Rows []map[string]int
type Alias = struct { Rows []map[string]int }
type (
	Holder struct { Table map[string]int }
	Tables []map[string]string
)

func main() {
	type LocalBox struct { M map[string]int }
	type LocalRows []map[string]int
	type (
		LocalHolder struct { Table map[string]int }
		LocalTables []map[string]string
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDiscardedEmptyMapContainingTypeCompositesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/discardedmapcontainingcomposites\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Box struct { M map[string]int }
type Rows []map[string]int
type (
	Holder struct { Table map[string]int }
	Tables []map[string]string
)

func main() {
	_ = Box{}
	_, _ = (Holder{}), Tables{}
	type LocalBox struct { M map[string]int }
	type LocalRows []map[string]int
	_ = LocalBox{}
	_ = (LocalRows{})
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInertNamedArrayTypesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/inertarray\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Values [3]int
type Bytes = [2]byte
type Alias [4]Values
type (
	Counts [2]int
	Rows = [3]Counts
)

func main() {
	type Local [2]int
	type LocalNested [3]Local
	type (
		LocalCounts [2]int
		LocalRows = [3]LocalCounts
	)
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedArrayTypeValuesFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedarrayvalues\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Values [3]int

func main() {
	var values Values
	values[0] = 1
	copy := values
	copy[0] = 9
	if values == (Values{1, 0, 0}) && copy[0] == 9 && len(values) == 3 && cap(values) == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStringRangeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stringrange\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func score(s string) int {
	total := 0
	for i, r := range s {
		total = total + i + int(r)
	}
	return total
}

func main() {
	text := "aé日"
	total := score(text)
	for i := range "éx" {
		total = total + i
	}
	var lastIndex int
	var lastRune int32
	for lastIndex, lastRune = range text {
	}
	if total == 26421 && lastIndex == 3 && lastRune == 26085 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestImportedNamedSliceRangeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/importedrangetype\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/importedrangetype/pkg/dep"

func main() {
	xs := dep.Values([]int{1, 2})
	total := len(xs)
	for _, v := range dep.Values([]int{1, 2}) {
		total = total + v
	}
	total = total + dep.Sum(dep.Values([]int{3, 4}))
	if total == 12 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Values []int

func Sum(values Values) int {
	total := 0
	for i := 0; i < len(values); i++ {
		total = total + values[i]
	}
	return total
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedConversionsFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedconversions\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type A struct {
	X int
}

type B A
type APtr *A
type Values []int

func wantB(v B) int {
	return v.X
}

func main() {
	a := A{X: 3}
	p := APtr(&a)
	b := B(A{X: 5})
	xs := []int(Values{1, 2})
	total := p.X + wantB(b) + len(xs)
	if total == 10 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestSamePackageTypeRewriteFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/typefields\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type Symbol struct {
	Name string
}

type Symbols []Symbol

type Unit struct {
	Exports Symbols
	Primary Symbol
	Package string
}

type Input struct {
	Name string
}

func Copy(out Symbols, values Symbols) Symbols {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func Make(input Input) Unit {
	values := Copy(Symbols{}, Symbols{Symbol{Name: input.Name}})
	return Unit{Package: input.Name, Primary: values[0], Exports: values}
}

func main() {
	unit := Make(Input{Name: "PASS\n"})
	if len(unit.Exports) == 1 && unit.Primary.Name == "PASS\n" && unit.Package == "PASS\n" {
		print(unit.Package)
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestAddressOfCompositeLiteralFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/addrofcomposite\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	value int
	next *box
}

func main() {
	first := &box{value: 20}
	second := &box{value: 22}
	first.next = second
	if first.value+first.next.value == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestCopyStringSourceFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/copystringsource\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type label string
type box struct {
	text string
}

func (b box) Text() string {
	return b.text
}

func main() {
	dst := []byte{'x', 'x', 'x', 'x', 'x'}
	var named label = label("S")
	boxed := box{text: "S\n"}
	n := copy(dst, "PA")
	m := copy(dst[2:], named)
	p := copy(dst[3:], boxed.Text())
	if n == 2 && m == 1 && p == 2 && string(dst) == "PASS\n" {
		print(string(dst))
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalConstFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/localconst\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", "package main\n\nfunc main() {\n\tconst answer = 40 + 2\n\tconst (\n\t\tread = 1 << iota\n\t\twrite\n\t\traw = `PASS\\n`\n\t\tlegacy = 077\n\t)\n\tif answer + read + write + legacy == 108 {\n\t\tprint(raw)\n\t\treturn\n\t}\n\tprint(\"FAIL\\n\")\n}\n")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalTypeFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/localtypes\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
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
	var x count = 20
	b := box{value: x + 22, text: "PASS\n"}
	s := single{value: 1}
	var items arrayValues = arrayValues{1, 2}
	itemsCopy := items
	itemsCopy[0] = 9
	values := scores{1, 2}
	if int(b.value) == 42 && int(s.value) == 1 && int(items[0]) == 1 && int(itemsCopy[0]) == 9 && len(values) == 2 {
		print(b.text)
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestInitFunctionFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/initfuncs\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/initfuncs/pkg/dep"

var total int

func init() {
	total = total + dep.Value()
}

func main() {
	if total == 42 && dep.Ready() == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var ready int

func init() {
	ready = 7
}

func Value() int {
	return ready * 6
}

func Ready() int {
	return ready
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestBlankImportInitFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/blankimport\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import _ "example.com/blankimport/pkg/side"

func main() {}
`)
	writeFixtureFile(t, fixture, "pkg/side/side.go", `package side

func init() {
	print("PASS\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestDotImportFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/dotimport\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import . "example.com/dotimport/pkg/dep"

func main() {
	var box Box
	box.Value = Value() + Number
	if box.Value == 42 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

const Number = 2

type Box struct {
	Value int
}

func Value() int {
	return 40
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestGoEmbedFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/goembed\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import _ "embed"

//go:embed text.txt
var text string

//go:embed bytes.bin
var blob []byte

//go:embed "text file.txt"
var spacedText string

//go:embed "bytes file.bin"
var spacedBlob []byte

//go:embed duplicate.txt duplicate.txt
var duplicateText string

//go:embed glob*.txt
var globText string

//go:embed multi.bin
//go:embed multi.bin
var multiBlob []byte

func main() {
	if text == "PASS\n" && spacedText == "OK" && duplicateText == "DUP" && globText == "GLOB" && len(blob) == 4 && blob[0] == 'D' && blob[1] == 'A' && blob[2] == 'T' && blob[3] == 'A' && len(spacedBlob) == 2 && spacedBlob[0] == 'X' && spacedBlob[1] == 'Y' && len(multiBlob) == 2 && multiBlob[0] == 'M' && multiBlob[1] == 'B' {
		print(text)
		return
	}
	print("FAIL\n")
}
	`)
	writeFixtureFile(t, fixture, "cmd/app/text.txt", "PASS\n")
	writeFixtureFile(t, fixture, "cmd/app/bytes.bin", "DATA")
	writeFixtureFile(t, fixture, "cmd/app/text file.txt", "OK")
	writeFixtureFile(t, fixture, "cmd/app/bytes file.bin", "XY")
	writeFixtureFile(t, fixture, "cmd/app/duplicate.txt", "DUP")
	writeFixtureFile(t, fixture, "cmd/app/glob-one.txt", "GLOB")
	writeFixtureFile(t, fixture, "cmd/app/multi.bin", "MB")
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPackageInitializerCallFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/pkginitcalls\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/pkginitcalls/pkg/dep"

type box struct { value int }

var calls int

func next() int {
	calls = calls + 1
	return calls
}

var a = next()
var b = box{value: next()}
var c = dep.Ready()

func init() {
	calls = calls + 10
}

func main() {
	if a == 1 && b.value == 2 && c == 2 && calls == 12 && dep.Counter() == 1 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

var counter int

func next() int {
	counter = counter + 1
	return counter
}

var ReadyValue = next()

func Ready() int {
	return ReadyValue + counter
}

func Counter() int {
	return counter
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestLocalVarFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/localvars\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func pair() (int, int) {
	return 7, 8
}

func main() {
	var (
		a = 1
		b int = 2
		c int
	)
	var d, e = 3, 4
	var f, g int = 5, 6
	var h, i int = pair()
	values := []int{a, b, c, d, e, f, g, h, i}
	total := 0
	for _, value := range values {
		total += value
	}
	if total == 36 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNamedResultFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/namedresults\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func value() (out int) {
	out = 7
	return
}

func pair() (left, right int) {
	left = 3
	right = 4
	return
}

func main() {
	a, b := pair()
	if value()+a+b == 14 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestPrintlnFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/println\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	msg := "PASS"
	println(msg)
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestStringCompoundAssignmentFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/stringcompound\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct {
	text string
}

func main() {
	values := []string{"A", "S"}
	values[0] += values[1]
	s := "P"
	s += values[0]
	s += "S"
	b := box{text: ""}
	b.text += "\n"
	if s == "PASS" && b.text == "\n" {
		print(s)
		print(b.text)
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewNamedStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newstruct\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct { value int }

func main() {
	p := new(box)
	p.value = 7
	if p.value == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewEscapingStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newescape\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

type box struct { value int }

func makeBox(v int) *box {
	p := new(box)
	p.value = v
	return p
}

func main() {
	left := makeBox(40)
	right := makeBox(2)
	left.value = left.value + right.value
	if left.value == 42 && right.value == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewScalarFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newscalar\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	p := new(int)
	*p = 7
	ok := new(bool)
	*ok = true
	text := new(string)
	*text = "ok"
	if *ok && *p == 7 && *text == "ok" {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewPointerAndSliceFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newptrslice\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func main() {
	x := 3
	p := new(*int)
	*p = &x
	values := new([]int)
	*values = append(*values, **p)
	bytes := new([]byte)
	*bytes = append(*bytes, byte(2))
	if (*values)[0] == 3 && int((*bytes)[0]) == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewEscapingSliceFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newsliceescape\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

func makeValues(v int) *[]int {
	p := new([]int)
	*p = append(*p, v)
	return p
}

func main() {
	left := makeValues(40)
	right := makeValues(2)
	*left = append(*left, (*right)[0])
	if len(*left) == 2 && (*left)[0] == 40 && (*left)[1] == 2 && len(*right) == 1 && (*right)[0] == 2 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewImportedNamedStructFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newimportstruct\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/newimportstruct/pkg/dep"

func main() {
	p := new(dep.Box)
	p.Value = 7
	if p.Value == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Box struct { Value int }
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestNewNamedStructAliasFrontendMatchesHostGo(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/newaliasstruct\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/newaliasstruct/pkg/dep"

type localBox struct { value int }
type localAlias localBox

func main() {
	p := new(localAlias)
	p.value = 5
	q := new(dep.Alias)
	q.Value = 7
	if p.value == 5 && q.Value == 7 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/dep.go", `package dep

type Box struct { Value int }
type Alias Box
`)
	runFrontendFixtureMatchesHostGo(t, fixture)
}

func TestFrontendUnitsUseRequestedTargetFiles(t *testing.T) {
	fixture := t.TempDir()
	writeFixtureFile(t, fixture, "go.mod", "module example.com/targetfiles\n")
	writeFixtureFile(t, fixture, "cmd/app/main.go", `package main

import "example.com/targetfiles/pkg/dep"

func main() {
	if dep.Value() == 1 {
		print("PASS\n")
	}
}
`)
	writeFixtureFile(t, fixture, "pkg/dep/value_amd64.go", `package dep

func Value() int { return 1 }
`)
	writeFixtureFile(t, fixture, "pkg/dep/value_arm64.go", `package dep

func Value() int { return 2 }
`)
	amd64Units := loadFixtureUnitsForTarget(t, fixture, "linux/amd64")
	aarch64Units := loadFixtureUnitsForTarget(t, fixture, "linux/aarch64")
	if !unitsContainBody(amd64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 1 }") {
		t.Fatalf("linux/amd64 units did not include amd64 body: %#v", amd64Units)
	}
	if unitsContainBody(amd64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 2 }") {
		t.Fatalf("linux/amd64 units included arm64 body: %#v", amd64Units)
	}
	if !unitsContainBody(aarch64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 2 }") {
		t.Fatalf("linux/aarch64 units did not include arm64 body: %#v", aarch64Units)
	}
	if unitsContainBody(aarch64Units, "func rtg_example_com_targetfiles_pkg_dep_Value() int { return 1 }") {
		t.Fatalf("linux/aarch64 units included amd64 body: %#v", aarch64Units)
	}
}

func runFrontendFixtureMatchesHostGo(t *testing.T, fixture string) {
	t.Helper()
	host := exec.Command("go", "run", "./cmd/app")
	host.Dir = fixture
	hostOut, err := host.CombinedOutput()
	if err != nil {
		t.Fatalf("host fixture failed: %v\n%s", err, string(hostOut))
	}
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			units := loadFixtureUnitsForTarget(t, fixture, target.name)
			out := filepath.Join(t.TempDir(), "hello")
			if err := rtgx.CompileUnits(units, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileUnits failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err != nil {
				t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
			}
			if !bytes.Equal(frontOut, hostOut) {
				t.Fatalf("frontend output = %q, host output = %q", string(frontOut), string(hostOut))
			}
		})
	}
}

func runFrontendFixtureOutput(t *testing.T, fixture string, want []byte) {
	t.Helper()
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			units := loadFixtureUnitsForTarget(t, fixture, target.name)
			out := filepath.Join(t.TempDir(), "app")
			if err := rtgx.CompileUnits(units, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileUnits failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err != nil {
				t.Fatalf("frontend fixture failed: %v\n%s", err, string(frontOut))
			}
			if !bytes.Equal(frontOut, want) {
				t.Fatalf("frontend output = %q, want %q", string(frontOut), string(want))
			}
		})
	}
}

func runFrontendFixtureFailureContains(t *testing.T, fixture string, want []byte) {
	t.Helper()
	for _, target := range frontendSmokeTargets(t) {
		target := target
		t.Run(target.name, func(t *testing.T) {
			if len(target.runner) > 0 {
				if _, err := exec.LookPath(target.runner[0]); err != nil {
					t.Skipf("runner %s is not installed", target.runner[0])
				}
			}
			units := loadFixtureUnitsForTarget(t, fixture, target.name)
			out := filepath.Join(t.TempDir(), "app")
			if err := rtgx.CompileUnits(units, rtgx.Options{Target: target.name, Output: out}); err != nil {
				t.Fatalf("CompileUnits failed: %v", err)
			}
			frontOut, err := runFrontendTarget(target, out)
			if err == nil {
				t.Fatalf("frontend fixture succeeded unexpectedly with output %q", string(frontOut))
			}
			if !bytes.Contains(frontOut, want) {
				t.Fatalf("frontend failure output = %q, want to contain %q", string(frontOut), string(want))
			}
		})
	}
}

func unitsContainBody(units []unit.Unit, body string) bool {
	for _, u := range units {
		for _, decl := range u.Decls {
			if strings.Contains(decl.Body, body) {
				return true
			}
		}
	}
	return false
}

func assertFixtureGoldenUnits(t *testing.T, name string) {
	t.Helper()
	fixture := filepath.Join("testdata", name)
	units := loadFixtureUnits(t, fixture)
	for _, u := range units {
		unitName := emit.FileName(u.ImportPath)
		got := string(emit.Source(u))
		wantPath := filepath.Join("testdata", "golden", name, unitName)
		want, err := os.ReadFile(wantPath)
		if err != nil {
			t.Fatalf("ReadFile golden %s failed: %v", wantPath, err)
		}
		if got != string(want) {
			t.Fatalf("golden mismatch for %s\n%s", unitName, diffText(string(want), got))
		}
	}
}

func writeFixtureFile(t *testing.T, root string, path string, data string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll %s failed: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(data), 0644); err != nil {
		t.Fatalf("WriteFile %s failed: %v", full, err)
	}
}

func frontendSmokeTargets(t *testing.T) []frontendSmokeTarget {
	t.Helper()
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		targets := []frontendSmokeTarget{{name: "linux/amd64"}}
		if os.Getenv(crossArchTestsEnv) == "1" {
			targets = append(targets,
				frontendSmokeTarget{name: "linux/386"},
				frontendSmokeTarget{name: "linux/aarch64", runner: []string{"qemu-aarch64"}},
				frontendSmokeTarget{name: "linux/arm", runner: []string{"qemu-arm"}},
				frontendSmokeTarget{name: "wasi/wasm32", runner: []string{"wasmtime", "run", "--dir=.", "--dir=/", "--env", "PWD", "--env", "PATH"}},
			)
		}
		return targets
	case "linux/arm64":
		return []frontendSmokeTarget{{name: "linux/aarch64"}}
	default:
		t.Skipf("no frontend smoke targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func runFrontendTarget(target frontendSmokeTarget, path string) ([]byte, error) {
	if len(target.runner) == 0 {
		cmd := exec.Command(path)
		return cmd.CombinedOutput()
	}
	args := append([]string{}, target.runner[1:]...)
	args = append(args, path)
	cmd := exec.Command(target.runner[0], args...)
	return cmd.CombinedOutput()
}

func loadFixtureUnits(t *testing.T, fixture string) []unit.Unit {
	t.Helper()
	return loadFixtureUnitsForTarget(t, fixture, "")
}

func loadFixtureUnitsForTarget(t *testing.T, fixture string, target string) []unit.Unit {
	t.Helper()
	graph, err := load.LoadEntries([]string{filepath.Join(fixture, "cmd", "app")}, load.Options{Target: target})
	if err != nil {
		t.Fatalf("LoadEntries failed: %v", err)
	}
	units, err := build.Units(graph)
	if err != nil {
		t.Fatalf("build.Units failed: %v", err)
	}
	sort.Slice(units, func(i int, j int) bool {
		return units[i].ImportPath < units[j].ImportPath
	})
	return units
}

func diffText(want string, got string) string {
	if want == got {
		return ""
	}
	return "want:\n" + want + "\ngot:\n" + got
}
