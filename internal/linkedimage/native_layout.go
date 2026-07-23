package linkedimage

// NativeSegment describes one range which an in-process loader must copy and
// protect. Permissions use the ELF convention: read=4, write=2, execute=1.
type NativeSegment struct {
	FileOffset  int
	Address     int
	FileSize    int
	MemorySize  int
	Permissions int
}

// NativeImport describes a pointer slot which must be bound before entering an
// image. Library is the native library named by the executable container.
type NativeImport struct {
	Library string
	Name    string
	Address int
}

type imagePESection struct {
	address    int
	memorySize int
	fileOffset int
	fileSize   int
}

// WindowsLayout decodes the loadable and imported parts of a PE32 or PE32+
// image. It intentionally uses only the Renvo language core so the bundled
// compiler can use the same parser as host-side tests.
func WindowsLayout(native []byte) (entry int, memorySize int, imageBase int, wordSize int, segments []NativeSegment, imports []NativeImport, ok bool) {
	if len(native) < 64 || native[0] != 'M' || native[1] != 'Z' {
		return 0, 0, 0, 0, nil, nil, false
	}
	pe := imageRead32(native, 60)
	if pe < 0 || pe+24 > len(native) ||
		native[pe] != 'P' || native[pe+1] != 'E' ||
		native[pe+2] != 0 || native[pe+3] != 0 {
		return 0, 0, 0, 0, nil, nil, false
	}
	sectionCount := imageRead16(native, pe+6)
	optionalSize := imageRead16(native, pe+20)
	optional := pe + 24
	sectionTable := optional + optionalSize
	if sectionCount < 1 || optionalSize < 104 || sectionTable < optional ||
		sectionTable+sectionCount*40 > len(native) {
		return 0, 0, 0, 0, nil, nil, false
	}
	magic := imageRead16(native, optional)
	directory := 0
	if magic == 0x20b {
		wordSize = 8
		if optionalSize < 120 {
			return 0, 0, 0, 0, nil, nil, false
		}
		low := imageRead32(native, optional+24)
		high := imageRead32(native, optional+28)
		factor := 65536
		factor = factor * 65536
		if high != 0 && factor == 0 {
			return 0, 0, 0, 0, nil, nil, false
		}
		imageBase = low + high*factor
		directory = optional + 112
	} else if magic == 0x10b {
		wordSize = 4
		imageBase = imageRead32(native, optional+28)
		directory = optional + 96
	} else {
		return 0, 0, 0, 0, nil, nil, false
	}
	entry = imageRead32(native, optional+16)
	memorySize = imageRead32(native, optional+56)
	headerSize := imageRead32(native, optional+60)
	if entry < 0 || memorySize <= 0 || entry >= memorySize ||
		headerSize <= 0 || headerSize > len(native) || directory+16 > sectionTable {
		return 0, 0, 0, 0, nil, nil, false
	}
	segments = make([]NativeSegment, 0, sectionCount+1)
	segments = append(segments, NativeSegment{
		FileOffset: 0, Address: 0, FileSize: headerSize,
		MemorySize: headerSize, Permissions: 4,
	})
	sections := make([]imagePESection, 0, sectionCount)
	for i := 0; i < sectionCount; i++ {
		at := sectionTable + i*40
		virtualSize := imageRead32(native, at+8)
		address := imageRead32(native, at+12)
		rawSize := imageRead32(native, at+16)
		rawOffset := imageRead32(native, at+20)
		segmentSize := virtualSize
		if rawSize > segmentSize {
			segmentSize = rawSize
		}
		fileEnd := rawOffset + rawSize
		memoryEnd := address + segmentSize
		if virtualSize < 0 || address < 0 || rawSize < 0 || rawOffset < 0 ||
			fileEnd < rawOffset || memoryEnd < address || fileEnd > len(native) ||
			memoryEnd > memorySize {
			return 0, 0, 0, 0, nil, nil, false
		}
		permissions := 0
		flagsTop := native[at+39]
		if flagsTop&0x40 != 0 {
			permissions |= 4
		}
		if flagsTop&0x80 != 0 {
			permissions |= 2
		}
		if flagsTop&0x20 != 0 {
			permissions |= 1
		}
		segments = append(segments, NativeSegment{
			FileOffset: rawOffset, Address: address, FileSize: rawSize,
			MemorySize: segmentSize, Permissions: permissions,
		})
		sections = append(sections, imagePESection{
			address: address, memorySize: segmentSize,
			fileOffset: rawOffset, fileSize: rawSize,
		})
	}
	importRVA := imageRead32(native, directory+8)
	importSize := imageRead32(native, directory+12)
	if importRVA == 0 && importSize == 0 {
		return entry, memorySize, imageBase, wordSize, segments, nil, true
	}
	if importRVA <= 0 || importSize < 20 {
		return 0, 0, 0, 0, nil, nil, false
	}
	importAt, found := imagePERVAToFile(sections, importRVA, 20)
	if !found {
		return 0, 0, 0, 0, nil, nil, false
	}
	for descriptor := 0; descriptor+20 <= importSize; descriptor += 20 {
		at := importAt + descriptor
		if at < importAt || at+20 > len(native) {
			return 0, 0, 0, 0, nil, nil, false
		}
		lookupRVA := imageRead32(native, at)
		nameRVA := imageRead32(native, at+12)
		iatRVA := imageRead32(native, at+16)
		if lookupRVA == 0 && nameRVA == 0 && iatRVA == 0 {
			return entry, memorySize, imageBase, wordSize, segments, imports, true
		}
		if lookupRVA == 0 {
			lookupRVA = iatRVA
		}
		nameAt, nameOK := imagePERVAToFile(sections, nameRVA, 1)
		lookupAt, lookupOK := imagePERVAToFile(sections, lookupRVA, wordSize)
		if !nameOK || !lookupOK || iatRVA <= 0 {
			return 0, 0, 0, 0, nil, nil, false
		}
		library, nameOK := imageStringZ(native, nameAt)
		if !nameOK {
			return 0, 0, 0, 0, nil, nil, false
		}
		for slot := 0; ; slot++ {
			thunkAt := lookupAt + slot*wordSize
			if thunkAt < lookupAt || thunkAt+wordSize > len(native) {
				return 0, 0, 0, 0, nil, nil, false
			}
			thunk := imageRead32(native, thunkAt)
			if wordSize == 8 && imageRead32(native, thunkAt+4) != 0 {
				// Renvo currently emits name imports only. Reject ordinals and
				// addresses which cannot be represented by the host int.
				return 0, 0, 0, 0, nil, nil, false
			}
			if thunk == 0 {
				break
			}
			symbolAt, symbolOK := imagePERVAToFile(sections, thunk, 3)
			if !symbolOK {
				return 0, 0, 0, 0, nil, nil, false
			}
			symbol, symbolOK := imageStringZ(native, symbolAt+2)
			if !symbolOK || iatRVA+slot*wordSize+wordSize > memorySize {
				return 0, 0, 0, 0, nil, nil, false
			}
			imports = append(imports, NativeImport{
				Library: library, Name: symbol,
				Address: iatRVA + slot*wordSize,
			})
		}
	}
	return 0, 0, 0, 0, nil, nil, false
}

func imagePERVAToFile(sections []imagePESection, rva int, size int) (int, bool) {
	if rva < 0 || size < 0 {
		return 0, false
	}
	for i := 0; i < len(sections); i++ {
		section := sections[i]
		if rva < section.address || rva-section.address > section.fileSize {
			continue
		}
		relative := rva - section.address
		if size > section.fileSize-relative {
			return 0, false
		}
		return section.fileOffset + relative, true
	}
	return 0, false
}

type imageMachSegment struct {
	address     int
	memorySize  int
	fileOffset  int
	fileSize    int
	permissions int
}

// DarwinLayout decodes the subset of Mach-O load and dyld bind commands
// emitted by Renvo's arm64 backend.
func DarwinLayout(native []byte) (entry int, memorySize int, segments []NativeSegment, imports []NativeImport, libraries []string, ok bool) {
	if len(native) < 32 || native[0] != 0xcf || native[1] != 0xfa ||
		native[2] != 0xed || native[3] != 0xfe {
		return 0, 0, nil, nil, nil, false
	}
	commandCount := imageRead32(native, 16)
	commandBytes := imageRead32(native, 20)
	if commandCount < 1 || commandBytes < 8 || 32+commandBytes > len(native) {
		return 0, 0, nil, nil, nil, false
	}
	allSegments := make([]imageMachSegment, 0, 4)
	base := -1
	entryFileOffset := -1
	bindOffset := 0
	bindSize := 0
	at := 32
	for i := 0; i < commandCount; i++ {
		if at+8 > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		command := imageRead32(native, at)
		size := imageRead32(native, at+4)
		if size < 8 || at+size < at || at+size > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		if command == 0x19 {
			if size < 72 {
				return 0, 0, nil, nil, nil, false
			}
			address, addressOK := imageRead64Int(native, at+24)
			segmentMemory, memoryOK := imageRead64Int(native, at+32)
			fileOffset, fileOK := imageRead64Int(native, at+40)
			fileSize, sizeOK := imageRead64Int(native, at+48)
			if !addressOK || !memoryOK || !fileOK || !sizeOK ||
				address < 0 || segmentMemory < 0 || fileOffset < 0 ||
				fileSize < 0 || fileSize > segmentMemory ||
				fileOffset+fileSize < fileOffset || fileOffset+fileSize > len(native) {
				return 0, 0, nil, nil, nil, false
			}
			allSegments = append(allSegments, imageMachSegment{
				address: address, memorySize: segmentMemory,
				fileOffset: fileOffset, fileSize: fileSize,
				permissions: imageRead32(native, at+60),
			})
			if address != 0 || fileSize != 0 {
				if base < 0 || address < base {
					base = address
				}
			}
		} else if command&0x7fffffff == 0x28 && native[at+3]&0x80 != 0 {
			var entryOK bool
			entryFileOffset, entryOK = imageRead64Int(native, at+8)
			if size < 24 || !entryOK {
				return 0, 0, nil, nil, nil, false
			}
		} else if command&0x7fffffff == 0x22 && native[at+3]&0x80 != 0 {
			if size < 48 {
				return 0, 0, nil, nil, nil, false
			}
			bindOffset = imageRead32(native, at+16)
			bindSize = imageRead32(native, at+20)
		} else if command == 0x0c || command == 0x18 || command == 0x1f {
			if size < 24 {
				return 0, 0, nil, nil, nil, false
			}
			pathOffset := imageRead32(native, at+8)
			if pathOffset < 0 || pathOffset >= size {
				return 0, 0, nil, nil, nil, false
			}
			library, libraryOK := imageStringZ(native, at+pathOffset)
			if !libraryOK || at+pathOffset+len(library) >= at+size {
				return 0, 0, nil, nil, nil, false
			}
			libraries = append(libraries, library)
		}
		at += size
	}
	if base < 0 || entryFileOffset < 0 || len(allSegments) == 0 {
		return 0, 0, nil, nil, nil, false
	}
	entryAddress := -1
	segments = make([]NativeSegment, 0, len(allSegments))
	for i := 0; i < len(allSegments); i++ {
		segment := allSegments[i]
		if segment.address == 0 && segment.fileSize == 0 {
			continue
		}
		address := segment.address - base
		end := address + segment.memorySize
		if address < 0 || end < address {
			return 0, 0, nil, nil, nil, false
		}
		if end > memorySize {
			memorySize = end
		}
		if entryFileOffset >= segment.fileOffset &&
			entryFileOffset-segment.fileOffset < segment.fileSize {
			entryAddress = address + entryFileOffset - segment.fileOffset
		}
		segments = append(segments, NativeSegment{
			FileOffset: segment.fileOffset, Address: address,
			FileSize: segment.fileSize, MemorySize: segment.memorySize,
			Permissions: segment.permissions,
		})
	}
	if entryAddress < 0 || entryAddress >= memorySize {
		return 0, 0, nil, nil, nil, false
	}
	if bindSize > 0 {
		if bindOffset < 0 || bindSize < 0 || bindOffset+bindSize < bindOffset ||
			bindOffset+bindSize > len(native) {
			return 0, 0, nil, nil, nil, false
		}
		var bindOK bool
		imports, bindOK = imageDarwinImports(native[bindOffset:bindOffset+bindSize], allSegments, base, libraries)
		if !bindOK {
			return 0, 0, nil, nil, nil, false
		}
		for i := 0; i < len(imports); i++ {
			if imports[i].Address < 0 || imports[i].Address+8 < imports[i].Address ||
				imports[i].Address+8 > memorySize {
				return 0, 0, nil, nil, nil, false
			}
		}
	}
	return entryAddress, memorySize, segments, imports, libraries, true
}

func imageDarwinImports(bind []byte, segments []imageMachSegment, base int, libraries []string) ([]NativeImport, bool) {
	var imports []NativeImport
	ordinal := 0
	symbol := ""
	address := -1
	for at := 0; at < len(bind); {
		opcode := int(bind[at])
		at++
		operation := opcode & 0xf0
		immediate := opcode & 15
		if operation == 0 {
			return imports, true
		}
		if operation == 0x10 {
			ordinal = immediate
			continue
		}
		if operation == 0x40 {
			var symbolOK bool
			symbol, symbolOK = imageStringZ(bind, at)
			if !symbolOK {
				return nil, false
			}
			at += len(symbol) + 1
			continue
		}
		if operation == 0x50 {
			continue
		}
		if operation == 0x70 {
			offset, next, offsetOK := imageReadULEB(bind, at)
			if !offsetOK || immediate >= len(segments) {
				return nil, false
			}
			at = next
			address = segments[immediate].address - base + offset
			continue
		}
		if operation == 0x80 {
			offset, next, offsetOK := imageReadULEB(bind, at)
			if !offsetOK || address < 0 {
				return nil, false
			}
			at = next
			address += offset
			continue
		}
		if operation == 0x90 {
			if address < 0 || symbol == "" || ordinal <= 0 || ordinal > len(libraries) {
				return nil, false
			}
			imports = append(imports, NativeImport{
				Library: libraries[ordinal-1], Name: symbol, Address: address,
			})
			address += 8
			continue
		}
		return nil, false
	}
	return imports, true
}

func imageReadULEB(src []byte, at int) (int, int, bool) {
	value := 0
	shift := 0
	for at < len(src) && shift < 28 {
		ch := int(src[at])
		at++
		value |= (ch & 127) << shift
		if ch&128 == 0 {
			return value, at, value >= 0
		}
		shift += 7
	}
	return 0, 0, false
}

func imageRead64Int(src []byte, at int) (int, bool) {
	if at < 0 || at+8 > len(src) {
		return 0, false
	}
	low := imageRead32(src, at)
	high := imageRead32(src, at+4)
	factor := 65536
	factor = factor * 65536
	if high != 0 && factor == 0 {
		return 0, false
	}
	value := low + high*factor
	return value, low >= 0 && high >= 0 && value >= 0
}

func imageStringZ(src []byte, at int) (string, bool) {
	if at < 0 || at >= len(src) {
		return "", false
	}
	end := at
	for end < len(src) && src[end] != 0 {
		end++
	}
	if end >= len(src) {
		return "", false
	}
	return string(src[at:end]), true
}

// BaseRelocations reads the compact relocation trailer emitted for callable
// Windows/386 images. Each address names one image-base-dependent 32-bit word.
func BaseRelocations(native []byte, memorySize int) ([]int, bool) {
	end, ok := imageReplTrailerStart(native)
	if !ok {
		return nil, false
	}
	if end < 8 ||
		native[end-4] != 'R' || native[end-3] != 'B' ||
		native[end-2] != 'R' || native[end-1] != '1' {
		return nil, true
	}
	count := imageRead32(native, end-8)
	if count < 0 || count > (end-8)/4 {
		return nil, false
	}
	start := end - 8 - count*4
	relocations := make([]int, 0, count)
	for i := 0; i < count; i++ {
		address := imageRead32(native, start+i*4)
		if address < 0 || address+4 < address || address+4 > memorySize {
			return nil, false
		}
		relocations = append(relocations, address)
	}
	return relocations, true
}

func imageReplTrailerStart(native []byte) (int, bool) {
	if len(native) < 8 ||
		native[len(native)-4] != 'R' || native[len(native)-3] != 'P' ||
		native[len(native)-2] != 'L' || native[len(native)-1] != '1' {
		return len(native), true
	}
	count := imageRead32(native, len(native)-8)
	if count < 0 || count > (len(native)-8)/16 {
		return 0, false
	}
	return len(native) - 8 - count*16, true
}
