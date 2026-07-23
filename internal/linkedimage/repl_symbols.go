package linkedimage

// PersistentSymbol is a state slot exported by a Renvo linked image. Address
// and RestoreAddress are offsets from the mapped image base.
type PersistentSymbol struct {
	ID             int
	Address        int
	Size           int
	RestoreAddress int
}

// PersistentSymbols reads the compact link table appended after an ELF, PE, or
// Mach-O image's loadable bytes. Native loaders ignore this trailer; the
// REPL's in-process linker uses it to migrate typed storage between generations.
func PersistentSymbols(native []byte, memorySize int) ([]PersistentSymbol, bool) {
	if len(native) < 8 ||
		native[len(native)-4] != 'R' || native[len(native)-3] != 'P' ||
		native[len(native)-2] != 'L' || native[len(native)-1] != '1' {
		return nil, true
	}
	count := imageRead32(native, len(native)-8)
	if count < 0 || count > (len(native)-8)/16 {
		return nil, false
	}
	start := len(native) - 8 - count*16
	symbols := make([]PersistentSymbol, 0, count)
	for i := 0; i < count; i++ {
		at := start + i*16
		symbol := PersistentSymbol{
			ID:             imageRead32(native, at),
			Address:        imageRead32(native, at+4),
			Size:           imageRead32(native, at+8),
			RestoreAddress: imageRead32(native, at+12),
		}
		valueEnd := symbol.Address + symbol.Size
		restoreEnd := symbol.RestoreAddress + 8
		if symbol.ID < 0 || symbol.Address < 0 || symbol.Size < 0 ||
			symbol.RestoreAddress < 0 || valueEnd < symbol.Address ||
			restoreEnd < symbol.RestoreAddress || valueEnd > memorySize ||
			restoreEnd > memorySize {
			return nil, false
		}
		symbols = append(symbols, symbol)
	}
	return symbols, true
}
