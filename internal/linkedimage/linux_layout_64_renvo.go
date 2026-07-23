//go:build renvo && linux && (amd64 || aarch64)

package linkedimage

func LinuxLayout(native []byte) (entry int, memorySize int, segments []LinuxSegment, ok bool) {
	if len(native) < 64 || native[0] != 0x7f || native[1] != 'E' || native[2] != 'L' ||
		native[3] != 'F' || native[4] != 2 || native[5] != 1 {
		return 0, 0, nil, false
	}
	entry, ok = imageRead64Offset(native, 24)
	if !ok {
		return 0, 0, nil, false
	}
	phoff, ok := imageRead64Offset(native, 32)
	phentsize := imageRead16(native, 54)
	phnum := imageRead16(native, 56)
	if !ok || phentsize < 56 || phnum < 1 {
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
		segment := LinuxSegment{Permissions: imageRead32(native, at+4)}
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
		if !ok || !imageAddLinuxSegment(native, segment, &memorySize) {
			return 0, 0, nil, false
		}
		segments = append(segments, segment)
	}
	return imageFinishLinuxLayout(entry, memorySize, segments)
}

func imageRead64Offset(src []byte, at int) (int, bool) {
	if at < 0 || at+8 > len(src) || imageRead32(src, at+4) != 0 {
		return 0, false
	}
	value := imageRead32(src, at)
	return value, value >= 0
}
