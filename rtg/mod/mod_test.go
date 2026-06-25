package mod

import "testing"

func TestParseModulePath(t *testing.T) {
	got, err := ParseModulePath("\n// comment\nmodule example.com/app // tail\n\ngo 1.25\n")
	if err != nil {
		t.Fatalf("ParseModulePath failed: %v", err)
	}
	if got != "example.com/app" {
		t.Fatalf("module path = %q, want example.com/app", got)
	}
}

func TestParseModulePathMissing(t *testing.T) {
	if _, err := ParseModulePath("go 1.25\n"); err == nil {
		t.Fatalf("ParseModulePath succeeded without module directive")
	}
}
