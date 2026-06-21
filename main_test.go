package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func resetRuntime() {
	files = make(map[int]file)
	files[0] = os.Stdin
	files[1] = os.Stdout
	files[2] = os.Stderr
}

func getCompilerFiles() []string {
	return []string{"compiler_amd64_impl.go", "compiler_linux_amd64_impl.go", "rtg_main.go"}
}

func compile(inputFiles []string, outputFile string) error {
	resetRuntime()

	var input []int
	for _, path := range inputFiles {
		fd := open(path, O_RDONLY)
		if fd < 0 {
			return fmt.Errorf("failed to open input file: %s", path)
		}
		input = append(input, fd)
	}

	outputFd := open(outputFile, O_RDWR|O_CREATE|O_TRUNC)
	if outputFd < 0 {
		return fmt.Errorf("failed to open output file: %s", outputFile)
	}

	err := compileLinuxAmd64(input, outputFd)
	if err != 0 {
		return fmt.Errorf("compilation failed")
	}

	return nil
}

// test that the compiler can compile and run a simple "hello, world!" program.
func TestCompileAndRunHello(t *testing.T) {
	inputFiles := []string{"tests/hello.go"}

	outDir := t.TempDir()
	outputFile := filepath.Join(outDir, "hello")

	err := compile(inputFiles, outputFile)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// Run the compiled binary and check its output
	cmd := exec.Command(outputFile)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("execution failed: %v\nOutput: %s", err, string(output))
	}

	expectedOutput := "hello, world!\n"
	if string(output) != expectedOutput {
		t.Fatalf("unexpected output: got %q, want %q", string(output), expectedOutput)
	}
}

// Test the self-hosting of the compiler.
func TestCompilerCompiler(t *testing.T) {
	inputFiles := getCompilerFiles()

	// compile stage0
	outDir := t.TempDir()
	stage0 := filepath.Join(outDir, "stage0")

	err := compile(inputFiles, stage0)
	if err != nil {
		t.Fatalf("compilation failed: %v", err)
	}

	// use stage0 to compile stage1
	stage1 := filepath.Join(outDir, "stage1")
	cmd := exec.Command(stage0, append([]string{"-o", stage1}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage0 compilation failed: %v\nOutput: %s", err, string(output))
	}

	// use stage1 to compile stage2
	stage2 := filepath.Join(outDir, "stage2")
	cmd = exec.Command(stage1, append([]string{"-o", stage2}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage1 compilation failed: %v\nOutput: %s", err, string(output))
	}

	// use stage2 to compile stage3
	stage3 := filepath.Join(outDir, "stage3")
	cmd = exec.Command(stage2, append([]string{"-o", stage3}, inputFiles...)...)
	cmd.Env = os.Environ()
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("stage2 compilation failed: %v\nOutput: %s", err, string(output))
	}

	// make sure stage2 and stage3 are byte identical
	stage2Data, err := os.ReadFile(stage2)
	if err != nil {
		t.Fatalf("failed to read stage2: %v", err)
	}
	stage3Data, err := os.ReadFile(stage3)
	if err != nil {
		t.Fatalf("failed to read stage3: %v", err)
	}
	if !bytes.Equal(stage2Data, stage3Data) {
		t.Fatal("stage2 and stage3 are not identical")
	}
}

// Check the stage2 compiler compiles in under 25ms and produces a binary under 126KB which runs with under 1MB max RSS.
func TestCompilerPerformance(t *testing.T) {
	t.Fatal("not implemented")
}
