//go:build !renvo

package linkedimage

func LinuxLayout(native []byte) (entry int, memorySize int, segments []LinuxSegment, ok bool) {
	if len(native) < 52 || native[0] != 0x7f || native[1] != 'E' || native[2] != 'L' ||
		native[3] != 'F' || native[5] != 1 {
		return 0, 0, nil, false
	}
	class := int(native[4])
	phoff, phentsize, phnum := 0, 0, 0
	if class == 2 {
		if len(native) < 64 {
			return 0, 0, nil, false
		}
		entry, ok = imageRead64Offset(native, 24)
		if ok {
			phoff, ok = imageRead64Offset(native, 32)
		}
		phentsize, phnum = imageRead16(native, 54), imageRead16(native, 56)
	} else if class == 1 {
		entry, phoff = imageRead32(native, 24), imageRead32(native, 28)
		phentsize, phnum = imageRead16(native, 42), imageRead16(native, 44)
		ok = true
	}
	if !ok || entry < 0 || phoff < 0 || phentsize < 32 || phnum < 1 {
		return 0, 0, nil, false
	}
	segments = make([]LinuxSegment, 0, phnum)
	for i := 0; i < phnum; i++ {
		at := phoff + i*phentsize
		if at < phoff || at < 0 || at+phentsize > len(native) {
			return 0, 0, nil, false
		}
		if imageRead32(native, at) != 1 {
			continue
		}
		var segment LinuxSegment
		if class == 2 {
			if phentsize < 56 {
				return 0, 0, nil, false
			}
			segment.Permissions = imageRead32(native, at+4)
			segment.FileOffset, ok = imageRead64Offset(native, at+8)
			if ok {
				segment.Address, ok = imageRead64Offset(native, at+16)
			}
			if ok {
				segment.FileSize, ok = imageRead64Offset(native, at+32)
			}
			if ok {
				segment.MemorySize, ok = imageRead64Offset(native, at+40)
			}
			if !ok {
				return 0, 0, nil, false
			}
		} else {
			segment.FileOffset = imageRead32(native, at+4)
			segment.Address = imageRead32(native, at+8)
			segment.FileSize = imageRead32(native, at+16)
			segment.MemorySize = imageRead32(native, at+20)
			segment.Permissions = imageRead32(native, at+24)
		}
		end, fileEnd := segment.Address+segment.MemorySize, segment.FileOffset+segment.FileSize
		if segment.Address < 0 || segment.FileOffset < 0 || segment.FileSize < 0 ||
			segment.MemorySize < segment.FileSize || end < segment.Address ||
			fileEnd < segment.FileOffset || segment.FileSize > 0 && fileEnd > len(native) {
			return 0, 0, nil, false
		}
		if end > memorySize {
			memorySize = end
		}
		segments = append(segments, segment)
	}
	if len(segments) == 0 || entry >= memorySize {
		return 0, 0, nil, false
	}
	return entry, memorySize, segments, true
}

func imageRead64Offset(src []byte, at int) (int, bool) {
	if at < 0 || at+8 > len(src) || imageRead32(src, at+4) != 0 {
		return 0, false
	}
	value := imageRead32(src, at)
	return value, value >= 0
}
