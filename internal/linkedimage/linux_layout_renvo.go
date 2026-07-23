//go:build renvo && linux

package linkedimage

func imageAddLinuxSegment(native []byte, segment LinuxSegment, memorySize *int) bool {
	end := segment.Address + segment.MemorySize
	fileEnd := segment.FileOffset + segment.FileSize
	if segment.Address < 0 || segment.FileOffset < 0 || segment.FileSize < 0 ||
		segment.MemorySize < segment.FileSize || end < segment.Address ||
		fileEnd < segment.FileOffset || segment.FileSize > 0 && fileEnd > len(native) {
		return false
	}
	if end > *memorySize {
		*memorySize = end
	}
	return true
}

func imageFinishLinuxLayout(entry int, memorySize int, segments []LinuxSegment) (int, int, []LinuxSegment, bool) {
	if len(segments) == 0 || entry < 0 || entry >= memorySize {
		return 0, 0, nil, false
	}
	return entry, memorySize, segments, true
}
