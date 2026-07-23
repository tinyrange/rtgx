//go:build !renvo || windows

package linkedimage

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
