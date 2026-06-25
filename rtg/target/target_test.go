package target

import "testing"

func TestSupportedTargets(t *testing.T) {
	for _, name := range []string{"linux/amd64", "linux/386", "linux/aarch64", "linux/arm", "windows/amd64", "windows/386", "wasi/wasm32"} {
		if !Supported(name) {
			t.Fatalf("Supported(%q) = false", name)
		}
	}
	if Supported("linux/arm64") {
		t.Fatalf("linux/arm64 should require spelling linux/aarch64")
	}
}

func TestDefaultTargetIsSupported(t *testing.T) {
	got := Default()
	if got == "" {
		t.Fatalf("Default returned empty target")
	}
	if !Supported(got) {
		t.Fatalf("Default = %q, not supported", got)
	}
}
