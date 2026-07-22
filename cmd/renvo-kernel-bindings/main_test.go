package main

import (
	"encoding/binary"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func append32(out []byte, value uint32) []byte {
	var word [4]byte
	binary.LittleEndian.PutUint32(word[:], value)
	return append(out, word[:]...)
}

func testBTFData() []byte {
	stringsData := []byte("\x00u64\x00sample\x00value\x00ktime_get_ns\x00")
	var types []byte
	// 1: unsigned 64-bit integer.
	types = append32(types, 1)
	types = append32(types, btfInt<<24)
	types = append32(types, 8)
	types = append32(types, 64)
	// 2: struct sample { value u64 }.
	types = append32(types, 5)
	types = append32(types, btfStruct<<24|1)
	types = append32(types, 8)
	types = append32(types, 12)
	types = append32(types, 1)
	types = append32(types, 0)
	// 3: func proto () u64.
	types = append32(types, 0)
	types = append32(types, btfFuncProto<<24)
	types = append32(types, 1)
	// 4: func ktime_get_ns, type 3.
	types = append32(types, 18)
	types = append32(types, btfFunc<<24)
	types = append32(types, 3)
	var out []byte
	out = append(out, 0x9f, 0xeb, 1, 0)
	out = append32(out, 24)
	out = append32(out, 0)
	out = append32(out, uint32(len(types)))
	out = append32(out, uint32(len(types)))
	out = append32(out, uint32(len(stringsData)))
	out = append(out, types...)
	return append(out, stringsData...)
}

func TestGenerateBindings(t *testing.T) {
	b, err := parseBTF(testBTFData())
	if err != nil {
		t.Fatal(err)
	}
	symbols, err := parseSymvers([]byte("0x11223344\tktime_get_ns\tvmlinux\tEXPORT_SYMBOL_GPL\n0xaabbccdd\tunknown_symbol\tvmlinux\tEXPORT_SYMBOL\n"))
	if err != nil {
		t.Fatal(err)
	}
	source, stats, err := generateBindings(b, symbols, "kernel")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, want := range []string{"type BTFStruct_Sample_2 struct { Raw [8]byte }", "BTFStruct_Sample_2_Value_0_BitOffset = 0", "// renvo:linkstatic kernel,ktime_get_ns", "func Kernel_KtimeGetNs() uint64", "Kernel_UnknownSymbol_CRC"} {
		if !strings.Contains(text, want) {
			t.Errorf("generated source missing %q", want)
		}
	}
	if stats.types != 4 || stats.records != 1 || stats.symbols != 2 || stats.callable != 1 {
		t.Fatalf("stats = %#v", stats)
	}
	if _, err := parser.ParseFile(token.NewFileSet(), "kernel_generated.go", source, 0); err != nil {
		t.Fatalf("generated source does not parse: %v", err)
	}
}

func TestParseBTFRejectsTruncation(t *testing.T) {
	data := testBTFData()
	if _, err := parseBTF(data[:len(data)-3]); err == nil {
		t.Fatal("truncated BTF was accepted")
	}
}

func TestSelectSymbols(t *testing.T) {
	symbols, err := parseSymvers([]byte("0x1\talpha\tvmlinux\tEXPORT_SYMBOL\n0x2\tbeta\tvmlinux\tEXPORT_SYMBOL\n"))
	if err != nil {
		t.Fatal(err)
	}
	selected, err := selectSymbols(symbols, "beta")
	if err != nil || len(selected) != 1 || selected[0].name != "beta" {
		t.Fatalf("selection = %#v, %v", selected, err)
	}
	if _, err := selectSymbols(symbols, "missing"); err == nil {
		t.Fatal("missing symbol was accepted")
	}
}
