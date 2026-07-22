package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const (
	btfInt       = 1
	btfPtr       = 2
	btfArray     = 3
	btfStruct    = 4
	btfUnion     = 5
	btfEnum      = 6
	btfFwd       = 7
	btfTypedef   = 8
	btfVolatile  = 9
	btfConst     = 10
	btfRestrict  = 11
	btfFunc      = 12
	btfFuncProto = 13
	btfVar       = 14
	btfDataSec   = 15
	btfFloat     = 16
	btfDeclTag   = 17
	btfTypeTag   = 18
	btfEnum64    = 19
)

type member struct {
	name         string
	typeID       uint32
	bitOffset    uint32
	bitfieldSize uint32
}

type enumValue struct {
	name  string
	value int64
}

type parameter struct {
	name   string
	typeID uint32
}

type btfType struct {
	id        uint32
	kind      uint32
	vlen      uint32
	kindFlag  bool
	name      string
	sizeType  uint32
	intData   uint32
	members   []member
	enums     []enumValue
	params    []parameter
	arrayElem uint32
	arrayLen  uint32
}

type btfFile struct {
	types []btfType
	byID  map[uint32]*btfType
	funcs map[string]*btfType
	vars  map[string]*btfType
}

type kernelSymbol struct {
	name      string
	crc       uint64
	module    string
	export    string
	namespace string
}

type generator struct {
	btf         btfFile
	symbols     []kernelSymbol
	packageName string
}

func main() {
	var btfPath string
	var symversPath string
	var outputPath string
	var packageName string
	var symbolList string
	var includeTypes bool
	flag.StringVar(&btfPath, "btf", "/sys/kernel/btf/vmlinux", "kernel BTF file")
	flag.StringVar(&symversPath, "symvers", "", "matching Module.symvers file")
	flag.StringVar(&outputPath, "o", "-", "output Go source file")
	flag.StringVar(&packageName, "package", "kernel", "generated package name")
	flag.StringVar(&symbolList, "symbols", "", "comma-separated exported symbols to include (default: all)")
	flag.BoolVar(&includeTypes, "types", true, "emit all BTF type and layout declarations")
	flag.Parse()
	if symversPath == "" {
		release, err := os.ReadFile("/proc/sys/kernel/osrelease")
		if err != nil {
			fatalf("read kernel release: %v", err)
		}
		symversPath = "/lib/modules/" + strings.TrimSpace(string(release)) + "/build/Module.symvers"
	}
	btfData, err := os.ReadFile(btfPath)
	if err != nil {
		fatalf("read BTF: %v", err)
	}
	symversData, err := os.ReadFile(symversPath)
	if err != nil {
		fatalf("read Module.symvers: %v", err)
	}
	b, err := parseBTF(btfData)
	if err != nil {
		fatalf("parse BTF: %v", err)
	}
	symbols, err := parseSymvers(symversData)
	if err != nil {
		fatalf("parse Module.symvers: %v", err)
	}
	if symbolList != "" {
		symbols, err = selectSymbols(symbols, symbolList)
		if err != nil {
			fatalf("select symbols: %v", err)
		}
	}
	source, stats, err := generateBindingsWithOptions(b, symbols, packageName, includeTypes)
	if err != nil {
		fatalf("generate bindings: %v", err)
	}
	if outputPath == "-" {
		_, err = os.Stdout.Write(source)
	} else {
		err = os.WriteFile(outputPath, source, 0644)
	}
	if err != nil {
		fatalf("write output: %v", err)
	}
	fmt.Fprintf(os.Stderr, "generated %d types, %d structs/unions, %d symbols, %d callable bindings\n", stats.types, stats.records, stats.symbols, stats.callable)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "renvo-kernel-bindings: "+format+"\n", args...)
	os.Exit(1)
}

func u32(data []byte, at int) (uint32, bool) {
	if at < 0 || at+4 > len(data) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(data[at : at+4]), true
}

func parseBTF(data []byte) (btfFile, error) {
	var out btfFile
	out.byID = make(map[uint32]*btfType)
	out.funcs = make(map[string]*btfType)
	out.vars = make(map[string]*btfType)
	if len(data) < 24 || data[0] != 0x9f || data[1] != 0xeb || data[2] != 1 {
		return out, fmt.Errorf("unsupported BTF header")
	}
	hdrLen, _ := u32(data, 4)
	typeOff, _ := u32(data, 8)
	typeLen, _ := u32(data, 12)
	strOff, _ := u32(data, 16)
	strLen, _ := u32(data, 20)
	typeStart := int(hdrLen + typeOff)
	typeEnd := typeStart + int(typeLen)
	strStart := int(hdrLen + strOff)
	strEnd := strStart + int(strLen)
	if hdrLen < 24 || typeStart < int(hdrLen) || typeEnd > len(data) || strStart < int(hdrLen) || strEnd > len(data) {
		return out, fmt.Errorf("invalid BTF section bounds")
	}
	stringAt := func(off uint32) (string, error) {
		at := strStart + int(off)
		if at < strStart || at >= strEnd {
			return "", fmt.Errorf("string offset %d is out of range", off)
		}
		end := at
		for end < strEnd && data[end] != 0 {
			end++
		}
		if end == strEnd {
			return "", fmt.Errorf("unterminated BTF string")
		}
		return string(data[at:end]), nil
	}
	pos := typeStart
	id := uint32(1)
	for pos < typeEnd {
		if pos+12 > typeEnd {
			return out, fmt.Errorf("truncated type %d", id)
		}
		nameOff := binary.LittleEndian.Uint32(data[pos : pos+4])
		info := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		sizeType := binary.LittleEndian.Uint32(data[pos+8 : pos+12])
		kind := info >> 24 & 31
		vlen := info & 0xffff
		name, err := stringAt(nameOff)
		if err != nil {
			return out, fmt.Errorf("type %d: %w", id, err)
		}
		t := btfType{id: id, kind: kind, vlen: vlen, kindFlag: info>>31 != 0, name: name, sizeType: sizeType}
		pos += 12
		need := func(n int) error {
			if n < 0 || pos+n > typeEnd {
				return fmt.Errorf("truncated type %d payload", id)
			}
			return nil
		}
		switch kind {
		case btfInt:
			if err := need(4); err != nil {
				return out, err
			}
			t.intData = binary.LittleEndian.Uint32(data[pos : pos+4])
			pos += 4
		case btfArray:
			if err := need(12); err != nil {
				return out, err
			}
			t.arrayElem = binary.LittleEndian.Uint32(data[pos : pos+4])
			t.arrayLen = binary.LittleEndian.Uint32(data[pos+8 : pos+12])
			pos += 12
		case btfStruct, btfUnion:
			if err := need(int(vlen) * 12); err != nil {
				return out, err
			}
			for i := uint32(0); i < vlen; i++ {
				memberName, err := stringAt(binary.LittleEndian.Uint32(data[pos : pos+4]))
				if err != nil {
					return out, err
				}
				offset := binary.LittleEndian.Uint32(data[pos+8 : pos+12])
				m := member{name: memberName, typeID: binary.LittleEndian.Uint32(data[pos+4 : pos+8]), bitOffset: offset}
				if t.kindFlag {
					m.bitfieldSize = offset >> 24
					m.bitOffset = offset & 0x00ffffff
				}
				t.members = append(t.members, m)
				pos += 12
			}
		case btfEnum:
			if err := need(int(vlen) * 8); err != nil {
				return out, err
			}
			for i := uint32(0); i < vlen; i++ {
				enumName, err := stringAt(binary.LittleEndian.Uint32(data[pos : pos+4]))
				if err != nil {
					return out, err
				}
				value := int64(int32(binary.LittleEndian.Uint32(data[pos+4 : pos+8])))
				t.enums = append(t.enums, enumValue{name: enumName, value: value})
				pos += 8
			}
		case btfFuncProto:
			if err := need(int(vlen) * 8); err != nil {
				return out, err
			}
			for i := uint32(0); i < vlen; i++ {
				paramName, err := stringAt(binary.LittleEndian.Uint32(data[pos : pos+4]))
				if err != nil {
					return out, err
				}
				t.params = append(t.params, parameter{name: paramName, typeID: binary.LittleEndian.Uint32(data[pos+4 : pos+8])})
				pos += 8
			}
		case btfVar, btfDeclTag:
			if err := need(4); err != nil {
				return out, err
			}
			pos += 4
		case btfDataSec, btfEnum64:
			if err := need(int(vlen) * 12); err != nil {
				return out, err
			}
			if kind == btfEnum64 {
				for i := uint32(0); i < vlen; i++ {
					enumName, err := stringAt(binary.LittleEndian.Uint32(data[pos : pos+4]))
					if err != nil {
						return out, err
					}
					lo := uint64(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
					hi := uint64(binary.LittleEndian.Uint32(data[pos+8 : pos+12]))
					t.enums = append(t.enums, enumValue{name: enumName, value: int64(hi<<32 | lo)})
					pos += 12
				}
			} else {
				pos += int(vlen) * 12
			}
		case btfPtr, btfFwd, btfTypedef, btfVolatile, btfConst, btfRestrict, btfFunc, btfFloat, btfTypeTag:
		default:
			return out, fmt.Errorf("unsupported BTF kind %d at type %d", kind, id)
		}
		out.types = append(out.types, t)
		id++
	}
	for i := range out.types {
		t := &out.types[i]
		out.byID[t.id] = t
		if t.kind == btfFunc && t.name != "" {
			out.funcs[t.name] = t
		} else if t.kind == btfVar && t.name != "" {
			out.vars[t.name] = t
		}
	}
	return out, nil
}

func parseSymvers(data []byte) ([]kernelSymbol, error) {
	var out []kernelSymbol
	for lineNumber, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if len(fields) < 4 {
			return nil, fmt.Errorf("Module.symvers line %d has %d fields", lineNumber+1, len(fields))
		}
		crc, err := strconv.ParseUint(strings.TrimPrefix(fields[0], "0x"), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("Module.symvers line %d CRC: %w", lineNumber+1, err)
		}
		s := kernelSymbol{crc: crc, name: fields[1], module: fields[2], export: fields[3]}
		if len(fields) > 4 {
			s.namespace = fields[4]
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

func selectSymbols(symbols []kernelSymbol, list string) ([]kernelSymbol, error) {
	want := make(map[string]bool)
	for _, name := range strings.Split(list, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("empty symbol name")
		}
		want[name] = false
	}
	var out []kernelSymbol
	for _, symbol := range symbols {
		if _, ok := want[symbol.name]; ok {
			out = append(out, symbol)
			want[symbol.name] = true
		}
	}
	for name, found := range want {
		if !found {
			return nil, fmt.Errorf("symbol %q is not exported by this kernel", name)
		}
	}
	return out, nil
}

type generationStats struct {
	types    int
	records  int
	symbols  int
	callable int
}

func generateBindings(b btfFile, symbols []kernelSymbol, packageName string) ([]byte, generationStats, error) {
	return generateBindingsWithOptions(b, symbols, packageName, true)
}

func generateBindingsWithOptions(b btfFile, symbols []kernelSymbol, packageName string, includeTypes bool) ([]byte, generationStats, error) {
	var stats generationStats
	if !validIdentifier(packageName) {
		return nil, stats, fmt.Errorf("invalid package name %q", packageName)
	}
	g := generator{btf: b, symbols: symbols, packageName: packageName}
	var out bytes.Buffer
	fmt.Fprintln(&out, "// Code generated by renvo-kernel-bindings; DO NOT EDIT.")
	fmt.Fprintln(&out, "// Structs and unions use exact-size raw storage. Member constants are BTF bit offsets.")
	fmt.Fprintln(&out, "// Callable symbol stubs are emitted only for ABI-safe prototypes with at most six words.")
	fmt.Fprintf(&out, "package %s\n\n", packageName)
	fmt.Fprintln(&out, "const BTFKindInt = 1")
	fmt.Fprintln(&out, "const BTFKindPointer = 2")
	fmt.Fprintln(&out, "const BTFKindStruct = 4")
	fmt.Fprintln(&out, "const BTFKindUnion = 5")
	fmt.Fprintln(&out)
	if includeTypes {
		for i := range b.types {
			t := &b.types[i]
			stats.types++
			base := typeIdentifier(t)
			fmt.Fprintf(&out, "const BTFType_%d_Kind = %d\n", t.id, t.kind)
			fmt.Fprintf(&out, "const BTFType_%d_Name = %q\n", t.id, t.name)
			fmt.Fprintf(&out, "const BTFType_%d_SizeOrType = %d\n", t.id, t.sizeType)
			if t.kind == btfStruct || t.kind == btfUnion {
				stats.records++
				fmt.Fprintf(&out, "type %s struct { Raw [%d]byte }\n", base, t.sizeType)
				for j, m := range t.members {
					memberName := identifier(m.name)
					if memberName == "" {
						memberName = "Anonymous"
					}
					prefix := fmt.Sprintf("%s_%s_%d", base, memberName, j)
					fmt.Fprintf(&out, "const %s_TypeID = %d\n", prefix, m.typeID)
					fmt.Fprintf(&out, "const %s_BitOffset = %d\n", prefix, m.bitOffset)
					fmt.Fprintf(&out, "const %s_BitfieldSize = %d\n", prefix, m.bitfieldSize)
				}
			} else if t.kind == btfEnum || t.kind == btfEnum64 {
				underlying := "int32"
				if t.kind == btfEnum64 || t.sizeType == 8 {
					underlying = "int64"
				}
				fmt.Fprintf(&out, "type %s %s\n", base, underlying)
				for j, value := range t.enums {
					name := identifier(value.name)
					if name == "" {
						name = "Value"
					}
					fmt.Fprintf(&out, "const %s_%s_%d %s = %d\n", base, name, j, base, value.value)
				}
			}
			fmt.Fprintln(&out)
		}
	}
	usedNames := make(map[string]int)
	for _, symbol := range symbols {
		stats.symbols++
		name := uniqueIdentifier("Kernel_"+identifier(symbol.name), usedNames)
		fmt.Fprintf(&out, "const %s_Name = %q\n", name, symbol.name)
		fmt.Fprintf(&out, "const %s_CRC uint64 = 0x%08x\n", name, symbol.crc)
		fmt.Fprintf(&out, "const %s_Export = %q\n", name, symbol.export)
		if symbol.namespace != "" {
			fmt.Fprintf(&out, "const %s_Namespace = %q\n", name, symbol.namespace)
		}
		fn := b.funcs[symbol.name]
		if fn == nil {
			if v := b.vars[symbol.name]; v != nil {
				fmt.Fprintf(&out, "const %s_BTFTypeID = %d\n\n", name, v.id)
			} else {
				fmt.Fprintf(&out, "const %s_BTFTypeID = 0\n\n", name)
			}
			continue
		}
		fmt.Fprintf(&out, "const %s_BTFTypeID = %d\n", name, fn.id)
		proto := b.byID[fn.sizeType]
		params, result, ok := g.callablePrototype(proto)
		if !ok {
			fmt.Fprintf(&out, "const %s_Callable = false\n\n", name)
			continue
		}
		stats.callable++
		fmt.Fprintf(&out, "const %s_Callable = true\n", name)
		fmt.Fprintf(&out, "// renvo:linkstatic kernel,%s\n", symbol.name)
		fmt.Fprintf(&out, "func %s(", name)
		for i, typ := range params {
			if i != 0 {
				fmt.Fprint(&out, ", ")
			}
			fmt.Fprintf(&out, "arg%d %s", i, typ)
		}
		fmt.Fprint(&out, ")")
		if result != "" {
			fmt.Fprintf(&out, " %s", result)
		}
		if result == "" {
			fmt.Fprintln(&out, " {}")
		} else {
			fmt.Fprintln(&out, " { return 0 }")
		}
		fmt.Fprintln(&out)
	}
	return out.Bytes(), stats, nil
}

func (g generator) callablePrototype(proto *btfType) ([]string, string, bool) {
	if proto == nil || proto.kind != btfFuncProto || len(proto.params) > 6 {
		return nil, "", false
	}
	params := make([]string, 0, len(proto.params))
	for _, param := range proto.params {
		typ, ok := g.abiType(param.typeID, false)
		if !ok || typ == "" {
			return nil, "", false
		}
		params = append(params, typ)
	}
	result, ok := g.abiType(proto.sizeType, true)
	if !ok {
		return nil, "", false
	}
	return params, result, true
}

func (g generator) abiType(typeID uint32, result bool) (string, bool) {
	if typeID == 0 {
		return "", result
	}
	seen := 0
	for seen < 32 {
		t := g.btf.byID[typeID]
		if t == nil {
			return "", false
		}
		switch t.kind {
		case btfTypedef, btfVolatile, btfConst, btfRestrict, btfTypeTag:
			typeID = t.sizeType
			seen++
			continue
		case btfPtr:
			return "uintptr", true
		case btfInt:
			signed := t.intData>>24&1 != 0
			switch t.sizeType {
			case 1:
				if signed {
					return "int8", true
				}
				return "uint8", true
			case 2:
				if signed {
					return "int16", true
				}
				return "uint16", true
			case 4:
				if signed {
					return "int32", true
				}
				return "uint32", true
			case 8:
				if signed {
					return "int64", true
				}
				return "uint64", true
			}
		case btfEnum:
			return "int32", true
		case btfEnum64:
			return "int64", true
		}
		return "", false
	}
	return "", false
}

func typeIdentifier(t *btfType) string {
	prefix := "Type"
	if t.kind == btfStruct {
		prefix = "Struct"
	}
	if t.kind == btfUnion {
		prefix = "Union"
	}
	if t.kind == btfEnum || t.kind == btfEnum64 {
		prefix = "Enum"
	}
	name := identifier(t.name)
	if name == "" {
		name = "Anonymous"
	}
	return fmt.Sprintf("BTF%s_%s_%d", prefix, name, t.id)
}

func identifier(value string) string {
	var out strings.Builder
	upper := true
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if out.Len() == 0 && unicode.IsDigit(r) {
				out.WriteByte('_')
			}
			if upper {
				r = unicode.ToUpper(r)
			}
			out.WriteRune(r)
			upper = false
		} else {
			upper = true
		}
	}
	return out.String()
}

func uniqueIdentifier(base string, used map[string]int) string {
	if base == "Kernel_" {
		base = "Kernel_Anonymous"
	}
	count := used[base]
	used[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s_%d", base, count+1)
}

func validIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		if i == 0 && !(unicode.IsLetter(r) || r == '_') {
			return false
		}
		if i > 0 && !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
			return false
		}
	}
	return true
}
