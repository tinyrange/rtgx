package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"j5.nz/rtg/rtgunit"
)

func TestBuildMetaHandlesMoreThanInitialFuncCapacity(t *testing.T) {
	src := []byte("package main\n")
	for i := 0; i < 1301; i++ {
		name := strconv.Itoa(i)
		src = append(src, []byte("func f"+name+"() int { return "+name+" }\n")...)
	}
	src = append(src, []byte("func appMain(args []string, env []string) int { return f1300() }\n")...)

	prog := rtgParseProgram(src)
	if !prog.ok {
		t.Fatalf("failed to parse generated source")
	}
	meta := rtgBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("failed to build metadata")
	}
	if len(meta.funcs) != 1302 {
		t.Fatalf("metadata function count = %d, want 1302", len(meta.funcs))
	}
}

func TestBuildMetaHandlesUnitGroupedConstSpecRows(t *testing.T) {
	program := unitProgramFromSource(t, []byte(`package main

const (
	BodyOK = iota
	BodyErrFunc
)

func appMain(args []string, env []string) int { return BodyErrFunc }
`))
	bodyOK := unitTokenByText(t, program, "BodyOK")
	iotaTok := unitTokenByText(t, program, "iota")
	program.Decls = []rtgunit.Decl{{
		Kind:      rtgTokConst,
		NameStart: unitTokenStart(program, bodyOK),
		NameEnd:   unitTokenEnd(program, bodyOK),
		StartTok:  bodyOK,
		EndTok:    iotaTok + 1,
	}}
	data, err := rtgunit.Marshal(program)
	if err != nil {
		t.Fatalf("unit marshal failed: %v", err)
	}
	prog, isUnit, ok := rtgDecodeUnitProgram(data)
	if !isUnit || !ok {
		t.Fatalf("unit decode failed: isUnit=%v ok=%v", isUnit, ok)
	}
	meta := rtgBuildMeta(&prog)
	if !meta.ok {
		t.Fatalf("metadata failed for grouped const spec-start unit row")
	}
	bodyErr := -1
	for i := 0; i < len(meta.globals); i++ {
		global := meta.globals[i]
		if rtgBytesEqualText(prog.src, global.nameStart, global.nameEnd, "BodyErrFunc") {
			bodyErr = i
			break
		}
	}
	if bodyErr < 0 || meta.globals[bodyErr].constValueOK == 0 || meta.globals[bodyErr].constValue != 1 {
		t.Fatalf("BodyErrFunc const = index %d globals %#v", bodyErr, meta.globals)
	}
}

func TestArbitrarySyscallLinuxAmd64Write(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skipf("linux/amd64 syscall execution test requires linux/amd64 host, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	src := []byte(`package main

func syscall(num int, fd int, msg string, n int) int { return 0 }

func appMain(args []string, env []string) int {
	syscall(1, 1, "PASS\n", 5)
	return 0
}
`)
	data, ok := RtgCompileSourceToBytes(src, "linux/amd64")
	if !ok {
		t.Fatalf("RtgCompileSourceToBytes failed")
	}
	out := filepath.Join(t.TempDir(), "syscall-write")
	if err := os.WriteFile(out, data, 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	cmd := exec.Command(out)
	got, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("compiled syscall test failed: %v\n%s", err, string(got))
	}
	if string(got) != "PASS\n" {
		t.Fatalf("compiled syscall output = %q, want PASS", string(got))
	}
}

func unitProgramFromSource(t *testing.T, src []byte) rtgunit.Program {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, src, 0644); err != nil {
		t.Fatalf("failed to write unit source: %v", err)
	}
	program, err := rtgunit.ConvertFiles([]string{path})
	if err != nil {
		t.Fatalf("unit conversion failed: %v", err)
	}
	return program
}

func unitTokenByText(t *testing.T, program rtgunit.Program, text string) int {
	t.Helper()
	count := len(program.Tokens) / 8
	for i := 0; i < count; i++ {
		if string(program.Text[unitTokenStart(program, i):unitTokenEnd(program, i)]) == text {
			return i
		}
	}
	t.Fatalf("token %q not found", text)
	return -1
}

func unitTokenStart(program rtgunit.Program, tok int) int {
	pos := tok * 8
	return int(program.Tokens[pos+1]) | int(program.Tokens[pos+2])<<8 | int(program.Tokens[pos+3])<<16
}

func unitTokenEnd(program rtgunit.Program, tok int) int {
	pos := tok * 8
	size := int(program.Tokens[pos+4])
	if int(program.Tokens[pos]) != rtgTokOp {
		size = size | int(program.Tokens[pos+5])<<8
	}
	return unitTokenStart(program, tok) + size
}

func TestLinkStaticAddsWindowsImport(t *testing.T) {
	src := []byte(`package main

// rtg:linkstatic user32.dll,MessageBeep
func messageBeep(kind int) int { return 0 }

func appMain(args []string, env []string) int {
	return messageBeep(0)
}
`)
	for _, target := range []string{"windows/amd64", "windows/386"} {
		target := target
		t.Run(target, func(t *testing.T) {
			data, ok := RtgCompileSourceToBytes(src, target)
			if !ok {
				t.Fatalf("RtgCompileSourceToBytes failed")
			}
			text := string(data)
			for _, want := range []string{"user32.dll", "MessageBeep"} {
				if !strings.Contains(text, want) {
					t.Fatalf("windows import table missing %q", want)
				}
			}
		})
	}
}
