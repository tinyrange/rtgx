package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"renvo.dev/backend/unit"
)

func TestLinkedUnitUntypedFloatConstantKeepsScaledValue(t *testing.T) {
	source := []byte(`package main

const value = (10 + 0.0)

func appMain() int {
	if value != 10.0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
`)
	program := unitProgramFromSource(t, source)
	for i := 0; i < len(program.Decls); i++ {
		if program.Decls[i].Kind == renvoTokConst {
			// Linked frontend units identify an individual declaration at its
			// name rather than at the package-level const keyword.
			program.Decls[i].StartTok++
		}
	}
	data, err := unit.Marshal(program)
	if err != nil {
		t.Fatal(err)
	}
	target := ""
	suffix := ""
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		target = "linux/amd64"
	case "linux/arm64":
		target = "linux/aarch64"
	case "darwin/arm64":
		target = "darwin/arm64"
	case "windows/amd64":
		target = "windows/amd64"
		suffix = ".exe"
	default:
		t.Skipf("no executable target for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	binary, ok := RenvoCompileUnitToBytesWithOptions(data, target, RenvoCompileOptions{StripSymbols: true})
	if !ok {
		t.Fatal("linked unit compilation failed")
	}
	path := filepath.Join(t.TempDir(), "mixed-float-constant"+suffix)
	if err = os.WriteFile(path, binary, 0o755); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(path).CombinedOutput()
	if err != nil || string(output) != "PASS\n" {
		t.Fatalf("linked unit result: err=%v output=%q", err, output)
	}
}
