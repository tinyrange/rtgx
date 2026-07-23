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
