package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLinuxLinkStaticDirectiveUsesPortableBodyFallback(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 linkstatic fallback test requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	source, err := os.ReadFile("tests/linkstatic_portable_fallback.renvo")
	if err != nil {
		t.Fatal(err)
	}
	image, ok := RenvoCompileSourceToBytes(source, "linux/amd64")
	if !ok {
		t.Fatal("failed to compile linkstatic fallback program")
	}
	output := filepath.Join(t.TempDir(), "linkstatic-fallback")
	if err := os.WriteFile(output, image, 0755); err != nil {
		t.Fatal(err)
	}
	result, err := runCommand(t, output)
	if err != nil {
		t.Fatalf("linkstatic fallback execution failed: %v", err)
	}
	if result.exitCode != 0 || result.stdout != "PASS\n" || result.stderr != "" {
		t.Fatalf("linkstatic fallback mismatch: exit=%d stdout=%q stderr=%q", result.exitCode, result.stdout, result.stderr)
	}
}
