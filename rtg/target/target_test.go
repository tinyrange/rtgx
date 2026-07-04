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

func TestWordSize(t *testing.T) {
	cases := []struct {
		target string
		size   int
	}{
		{target: "linux/amd64", size: 8},
		{target: "linux/386", size: 4},
		{target: "linux/aarch64", size: 8},
		{target: "linux/arm", size: 4},
		{target: "windows/amd64", size: 8},
		{target: "windows/386", size: 4},
		{target: "wasi/wasm32", size: 4},
	}
	for _, tc := range cases {
		if got := WordSize(tc.target); got != tc.size {
			t.Fatalf("WordSize(%q) = %d, want %d", tc.target, got, tc.size)
		}
	}
}
