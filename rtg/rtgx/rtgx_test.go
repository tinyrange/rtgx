package rtgx

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/unit"
)

const crossArchTestsEnv = "RTG_CROSS_ARCH_TESTS"

func TestCompileSourceBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	out := filepath.Join(t.TempDir(), "app")
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	if err := CompileSource(src, Options{Target: "linux/amd64", Output: out, BackendRoot: root}); err != nil {
		t.Fatalf("CompileSource failed: %v", err)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatalf("Stat output failed: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Fatalf("output is not executable: %v", info.Mode())
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(data))
	}
}

func TestCompileSourceBytesBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	data, err := CompileSourceBytes(src, Options{Target: "linux/amd64", BackendRoot: root})
	if err != nil {
		t.Fatalf("CompileSourceBytes failed: %v", err)
	}
	out := filepath.Join(t.TempDir(), "app")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile output failed: %v", err)
	}
	cmd := exec.Command(out)
	outData, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(outData))
	}
	if string(outData) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(outData))
	}
}

func TestCompileSourceBytesProducesTargetBytes(t *testing.T) {
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	for _, target := range rtgxByteTargets(t) {
		target := target
		t.Run(target, func(t *testing.T) {
			data, err := CompileSourceBytes(src, Options{Target: target, BackendRoot: root})
			if err != nil {
				t.Fatalf("CompileSourceBytes failed: %v", err)
			}
			if len(data) == 0 {
				t.Fatalf("CompileSourceBytes returned empty output")
			}
			if !hasTargetMagic(target, data) {
				t.Fatalf("compiled bytes for %s have unexpected magic: % x", target, leadingBytes(data, 8))
			}
		})
	}
}

func TestCompileSourceWritesDashOutputToStdout(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 stdout smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}
	defer os.Chdir(cwd)
	src := []byte(`//go:build rtg

package main

func appMain() int {
	print("PASS\n")
	return 0
}
`)
	stdout := os.Stdout
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	outCh := make(chan []byte, 1)
	errCh := make(chan error, 1)
	go func() {
		data, err := io.ReadAll(read)
		outCh <- data
		errCh <- err
	}()
	os.Stdout = write
	err = CompileSource(src, Options{Target: "linux/amd64", Output: "-", BackendRoot: root})
	os.Stdout = stdout
	if closeErr := write.Close(); closeErr != nil {
		t.Fatalf("stdout pipe close failed: %v", closeErr)
	}
	data := <-outCh
	if readErr := <-errCh; readErr != nil {
		t.Fatalf("stdout pipe read failed: %v", readErr)
	}
	if err != nil {
		t.Fatalf("CompileSource failed: %v", err)
	}
	if !hasTargetMagic("linux/amd64", data) {
		t.Fatalf("stdout output has unexpected magic: % x", leadingBytes(data, 8))
	}
	if _, statErr := os.Stat("-"); !os.IsNotExist(statErr) {
		t.Fatalf("CompileSource created output file named '-': %v", statErr)
	}
}

func TestCompileSourceRequiresOutput(t *testing.T) {
	err := CompileSource([]byte("package main\n"), Options{Target: "linux/amd64"})
	if err == nil {
		t.Fatalf("CompileSource accepted missing output path")
	}
	if !strings.Contains(err.Error(), "missing output path") {
		t.Fatalf("error = %q", err)
	}
}

func rtgxByteTargets(t *testing.T) []string {
	t.Helper()
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		targets := []string{"linux/amd64"}
		if os.Getenv(crossArchTestsEnv) == "1" {
			targets = append(targets,
				"linux/386",
				"linux/aarch64",
				"linux/arm",
				"windows/amd64",
				"windows/386",
				"wasi/wasm32",
			)
		}
		return targets
	case "linux/arm64":
		return []string{"linux/aarch64"}
	default:
		t.Skipf("no rtgx byte targets supported on %s/%s", runtime.GOOS, runtime.GOARCH)
		return nil
	}
}

func hasTargetMagic(target string, data []byte) bool {
	if strings.HasPrefix(target, "linux/") {
		return len(data) >= 4 && data[0] == 0x7f && data[1] == 'E' && data[2] == 'L' && data[3] == 'F'
	}
	if strings.HasPrefix(target, "windows/") {
		return len(data) >= 2 && data[0] == 'M' && data[1] == 'Z'
	}
	if target == "wasi/wasm32" {
		return len(data) >= 4 && data[0] == 0x00 && data[1] == 'a' && data[2] == 's' && data[3] == 'm'
	}
	return false
}

func leadingBytes(data []byte, n int) []byte {
	if len(data) < n {
		return data
	}
	return data[:n]
}

func TestCompileSourceRejectsUnsupportedTarget(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileSource([]byte("package main\n"), Options{Target: "linux/arm64", Output: out})
	if err == nil {
		t.Fatalf("CompileSource accepted unsupported target")
	}
	msg := err.Error()
	for _, want := range []string{"rtg: unsupported target: linux/arm64", "linux/amd64", "linux/aarch64", "wasi/wasm32"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestCompileUnitsBuildsRunnableExecutable(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 executable smoke requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	out := filepath.Join(t.TempDir(), "app")
	units := []unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Imports:    []string{"example.com/app/dep"},
			References: []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_dep_Print() }\n"},
			},
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
			Exports:    []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "Print", UnitName: "rtg_example_com_app_dep_Print", Body: "func rtg_example_com_app_dep_Print() int { print(\"PASS\\n\"); return 0 }\n"},
			},
		},
	}
	if err := CompileUnits(units, Options{Target: "linux/amd64", Output: out, BackendRoot: root}); err != nil {
		t.Fatalf("CompileUnits failed: %v", err)
	}
	cmd := exec.Command(out)
	data, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled app failed: %v\n%s", err, string(data))
	}
	if string(data) != "PASS\n" {
		t.Fatalf("compiled app output = %q", string(data))
	}
}

func TestCompileUnitsArtifactIncludesLinkedMetadataAndTargetBytes(t *testing.T) {
	root, ok := findBackendRootUpward(".")
	if !ok {
		t.Skip("backend root not found")
	}
	units := []unit.Unit{
		{
			ImportPath: "example.com/app/main",
			Package:    "main",
			Imports:    []string{"example.com/app/dep"},
			References: []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return rtg_example_com_app_dep_Print() }\n"},
			},
		},
		{
			ImportPath: "example.com/app/dep",
			Package:    "dep",
			Exports:    []unit.Symbol{{ImportPath: "example.com/app/dep", Name: "Print", UnitName: "rtg_example_com_app_dep_Print"}},
			Decls: []unit.Decl{
				{Kind: "func", Name: "Print", UnitName: "rtg_example_com_app_dep_Print", Body: "func rtg_example_com_app_dep_Print() int { print(\"PASS\\n\"); return 0 }\n"},
			},
		},
	}
	artifact, err := CompileUnitsArtifact(units, Options{Target: "linux/amd64", BackendRoot: root})
	if err != nil {
		t.Fatalf("CompileUnitsArtifact failed: %v", err)
	}
	if artifact.Target != "linux/amd64" {
		t.Fatalf("artifact target = %q", artifact.Target)
	}
	if !hasTargetMagic("linux/amd64", artifact.Output) {
		t.Fatalf("artifact output has unexpected magic: % x", leadingBytes(artifact.Output, 8))
	}
	if len(artifact.LinkedUnits) != 2 || artifact.LinkedUnits[0] != "example.com/app/dep" || artifact.LinkedUnits[1] != "example.com/app/main" {
		t.Fatalf("linked units = %#v", artifact.LinkedUnits)
	}
	if artifact.Entrypoint != (unit.Symbol{ImportPath: "example.com/app/main", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain"}) {
		t.Fatalf("entrypoint = %#v", artifact.Entrypoint)
	}
	if !strings.Contains(string(artifact.LinkedSource), "// rtg:linked-unit example.com/app/main\n") {
		t.Fatalf("linked source missing linked-unit metadata:\n%s", string(artifact.LinkedSource))
	}
	if len(artifact.ReachableFunctions) != 2 {
		t.Fatalf("reachable functions = %#v", artifact.ReachableFunctions)
	}
}

func TestCompileUnitsValidatesUnitGraph(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnits([]unit.Unit{{
		ImportPath: "example.com/app/main",
		Package:    "main",
		Imports:    []string{"example.com/app/missing"},
		Decls: []unit.Decl{
			{Kind: "func", Name: "appMain", UnitName: "rtg_example_com_app_main_appMain", Body: "func rtg_example_com_app_main_appMain() int { return 0 }\n"},
		},
	}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnits accepted missing imported unit")
	}
	if !strings.Contains(err.Error(), "missing imported unit example.com/app/missing") {
		t.Fatalf("error = %q", err)
	}
}

func TestCompileUnitSourcesReportsParseErrors(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnitSources([]unit.SourceFile{{Path: "broken.rtg.go", Source: []byte("//go:build rtg\n\npackage main\n")}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnitSources accepted malformed unit source")
	}
	if !strings.Contains(err.Error(), "broken.rtg.go: missing rtg unit metadata") {
		t.Fatalf("error = %q", err)
	}
}

func TestCompileUnitSourcesValidatesUnitGraph(t *testing.T) {
	out := filepath.Join(t.TempDir(), "app")
	err := CompileUnitSources([]unit.SourceFile{{
		Path: "main.rtg.go",
		Source: []byte(`//go:build rtg

// rtg:unit example.com/app/main
package main

// rtg:import "example.com/app/missing"
// rtg:decl func appMain => rtg_example_com_app_main_appMain main.go
func rtg_example_com_app_main_appMain() int { return 0 }
`),
	}}, Options{Target: "linux/amd64", Output: out})
	if err == nil {
		t.Fatalf("CompileUnitSources accepted missing imported unit")
	}
	if !strings.Contains(err.Error(), "missing imported unit example.com/app/missing") {
		t.Fatalf("error = %q", err)
	}
}
