package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type rtgAdvertisedTargetContract struct {
	name  string
	id    int
	magic string
}

var rtgAdvertisedTargetContracts = []rtgAdvertisedTargetContract{
	{"linux/amd64", rtgTargetLinuxAmd64, "\x7fELF"},
	{"linux/386", rtgTargetLinux386, "\x7fELF"},
	{"linux/aarch64", rtgTargetLinuxAarch64, "\x7fELF"},
	{"linux/arm", rtgTargetLinuxArm, "\x7fELF"},
	{"windows/amd64", rtgTargetWindowsAmd64, "MZ"},
	{"windows/386", rtgTargetWindows386, "MZ"},
	{"windows/arm64", rtgTargetWindowsArm64, "MZ"},
	{"wasi/wasm32", rtgTargetWasiWasm32, "\x00asm"},
	{"darwin/arm64", rtgTargetDarwinArm64, "\xcf\xfa\xed\xfe"},
}

func TestAdvertisedTargetsHaveProfilesAndRecognizableImages(t *testing.T) {
	source := []byte("package main\nfunc appMain() int { print(\"PASS\\n\"); return 0 }\n")
	for _, contract := range rtgAdvertisedTargetContracts {
		contract := contract
		t.Run(strings.ReplaceAll(contract.name, "/", "-"), func(t *testing.T) {
			if got := rtgParseTargetArg(contract.name); got != contract.id {
				t.Fatalf("target parser returned %d, want %d", got, contract.id)
			}
			profile, ok := rtgProfileForTarget(contract.id)
			if !ok || !rtgProfileIsValid(profile) {
				t.Fatalf("target profile invalid: %#v", profile)
			}
			image, ok := RtgCompileSourceToBytesStrip(source, contract.name, true)
			if !ok {
				t.Fatal("target image compilation failed")
			}
			if len(image) < len(contract.magic) || string(image[:len(contract.magic)]) != contract.magic {
				t.Fatalf("image prefix = %x, want %x", image[:min(len(image), len(contract.magic))], []byte(contract.magic))
			}
		})
	}
}

func TestCompilerSourceManifestCoversBackendImplementationFiles(t *testing.T) {
	data, err := os.ReadFile("compiler_sources.txt")
	if err != nil {
		t.Fatalf("read compiler source manifest: %v", err)
	}
	var manifest []string
	for _, line := range strings.Split(string(data), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			manifest = append(manifest, path)
		}
	}
	implementationFiles, err := filepath.Glob("compiler_*_impl.go")
	if err != nil {
		t.Fatalf("glob backend implementation files: %v", err)
	}
	implementationFiles = append(implementationFiles, "compiler_main.go")
	sort.Strings(manifest)
	sort.Strings(implementationFiles)
	if strings.Join(manifest, "\n") != strings.Join(implementationFiles, "\n") {
		t.Fatalf("compiler source manifest drift\nmanifest: %v\nfiles: %v", manifest, implementationFiles)
	}
}
