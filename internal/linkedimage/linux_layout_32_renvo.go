//go:build renvo && linux && (386 || arm)

package linkedimage

func LinuxLayout(native []byte) (entry int, memorySize int, segments []LinuxSegment, ok bool) {
	if len(native) < 52 || native[0] != 0x7f || native[1] != 'E' || native[2] != 'L' ||
		native[3] != 'F' || native[4] != 1 || native[5] != 1 {
		return 0, 0, nil, false
	}
	entry = imageRead32(native, 24)
	phoff := imageRead32(native, 28)
	phentsize := imageRead16(native, 42)
	phnum := imageRead16(native, 44)
	if entry < 0 || phoff < 0 || phentsize < 32 || phnum < 1 {
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
		segment := LinuxSegment{
			FileOffset:  imageRead32(native, at+4),
			Address:     imageRead32(native, at+8),
			FileSize:    imageRead32(native, at+16),
			MemorySize:  imageRead32(native, at+20),
			Permissions: imageRead32(native, at+24),
		}
		if !imageAddLinuxSegment(native, segment, &memorySize) {
			return 0, 0, nil, false
		}
		segments = append(segments, segment)
	}
	return imageFinishLinuxLayout(entry, memorySize, segments)
}
