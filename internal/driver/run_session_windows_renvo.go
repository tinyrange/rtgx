//go:build renvo && windows

package driver

import "renvo.dev/internal/linkedimage"

const renvoRunJITStackSize = 8 * 1024 * 1024

const (
	renvoRunWindowsMemCommit       = 0x1000
	renvoRunWindowsMemReserve      = 0x2000
	renvoRunWindowsMemRelease      = 0x8000
	renvoRunWindowsPageRead        = 0x02
	renvoRunWindowsPageReadWrite   = 0x04
	renvoRunWindowsPageExecuteRead = 0x20
)

// renvo:linkstatic kernel32.dll,VirtualAlloc
func renvoRunWindowsVirtualAlloc(address int, size int, allocationType int, protection int) int {
	return 0
}

// renvo:linkstatic kernel32.dll,VirtualProtect
func renvoRunWindowsVirtualProtect(address int, size int, protection int, oldProtection *int) int {
	return 0
}

// renvo:linkstatic kernel32.dll,VirtualFree
func renvoRunWindowsVirtualFree(address int, size int, freeType int) int { return 0 }

// renvo:linkstatic kernel32.dll,LoadLibraryA
func renvoRunWindowsLoadLibrary(name *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,FreeLibrary
func renvoRunWindowsFreeLibrary(module int) int { return 0 }

// renvo:linkstatic kernel32.dll,GetProcAddress
func renvoRunWindowsGetProcAddress(module int, name *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,GetCurrentProcess
func renvoRunWindowsGetCurrentProcess() int { return 0 }

// renvo:linkstatic kernel32.dll,FlushInstructionCache
func renvoRunWindowsFlushInstructionCache(process int, address int, size int) int {
	return 0
}

type renvoRunMapping struct {
	base int
	size int
}

type renvoRunPersistentSymbol struct {
	id      int
	address int
	size    int
}

// LinkedImageSession retains every successful PE generation. Older mappings
// remain live because closures and pointer-valued globals can still refer to
// their code and storage after a relink.
type LinkedImageSession struct {
	mappings       []renvoRunMapping
	symbols        []renvoRunPersistentSymbol
	libraryHandles []int
	stack          int
}

func (s *LinkedImageSession) Prepare() {
	if cap(s.mappings) == 0 {
		s.mappings = make([]renvoRunMapping, 0, 2048)
	}
	if cap(s.symbols) == 0 {
		s.symbols = make([]renvoRunPersistentSymbol, 0, 8192)
	}
	if cap(s.libraryHandles) == 0 {
		s.libraryHandles = make([]int, 0, 8192)
	}
}

func (s *LinkedImageSession) Run(native []byte, script string, args []string, env []string) int {
	if renvoRunJITCall == nil {
		return -1
	}
	entry, memorySize, preferredBase, wordSize, segments, imports, ok := linkedimage.WindowsLayout(native)
	if !ok || wordSize != renvoRunWindowsPointerSize() {
		return -1
	}
	linkSymbols, ok := linkedimage.PersistentSymbols(native, memorySize)
	if !ok {
		return -1
	}
	relocations, ok := linkedimage.BaseRelocations(native, memorySize)
	if !ok {
		return -1
	}
	newSymbols := 0
	for i := 0; i < len(linkSymbols); i++ {
		if s.findSymbol(linkSymbols[i].ID) < 0 {
			newSymbols++
		}
	}
	if len(s.mappings) >= cap(s.mappings) || len(s.symbols)+newSymbols > cap(s.symbols) {
		return -1
	}
	if len(s.libraryHandles)+len(imports) > cap(s.libraryHandles) {
		return -1
	}
	memorySize = renvoRunWindowsPageAlign(memorySize)
	base := renvoRunWindowsVirtualAlloc(
		0, memorySize,
		renvoRunWindowsMemCommit|renvoRunWindowsMemReserve,
		renvoRunWindowsPageReadWrite,
	)
	if base == 0 {
		return -1
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		for at := 0; at < segment.FileSize; at++ {
			renvoRunStoreByte(base+segment.Address+at, native[segment.FileOffset+at])
		}
	}
	delta := base - preferredBase
	for i := 0; i < len(relocations); i++ {
		address := base + relocations[i]
		value := renvoRunLoadNativeWord(address, 4)
		renvoRunStoreNativeWord(address, value+delta, 4)
	}
	libraryNames := make([]string, 0, len(imports))
	libraryHandles := make([]int, 0, len(imports))
	for i := 0; i < len(imports); i++ {
		item := imports[i]
		handle := 0
		for j := 0; j < len(libraryNames); j++ {
			if libraryNames[j] == item.Library {
				handle = libraryHandles[j]
				break
			}
		}
		if handle == 0 {
			name := renvoRunCString(item.Library)
			handle = renvoRunWindowsLoadLibrary(&name[0])
			if handle != 0 {
				libraryNames = append(libraryNames, item.Library)
				libraryHandles = append(libraryHandles, handle)
				s.libraryHandles = append(s.libraryHandles, handle)
			}
		}
		if handle == 0 {
			renvoRunWindowsRelease(base)
			return -1
		}
		name := renvoRunCString(item.Name)
		address := renvoRunWindowsGetProcAddress(handle, &name[0])
		if address == 0 {
			renvoRunWindowsRelease(base)
			return -1
		}
		renvoRunStoreNativeWord(base+item.Address, address, wordSize)
	}
	for i := 0; i < len(linkSymbols); i++ {
		symbol := linkSymbols[i]
		previous := s.findSymbol(symbol.ID)
		if previous < 0 || s.symbols[previous].size != symbol.Size {
			continue
		}
		oldAddress := s.symbols[previous].address
		newAddress := base + symbol.Address
		for at := 0; at < symbol.Size; at++ {
			renvoRunStoreByte(newAddress+at, renvoRunLoadByte(oldAddress+at))
		}
		renvoRunStoreByte(base+symbol.RestoreAddress, 1)
	}
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		if segment.MemorySize == 0 {
			continue
		}
		protection := renvoRunWindowsPageRead
		if segment.Permissions&2 != 0 {
			protection = renvoRunWindowsPageReadWrite
		} else if segment.Permissions&1 != 0 {
			protection = renvoRunWindowsPageExecuteRead
		}
		start := renvoRunWindowsPageFloor(segment.Address)
		end := renvoRunWindowsPageAlign(segment.Address + segment.MemorySize)
		oldProtection := 0
		if renvoRunWindowsVirtualProtect(base+start, end-start, protection, &oldProtection) == 0 {
			renvoRunWindowsRelease(base)
			return -1
		}
	}
	if renvoRunWindowsFlushInstructionCache(
		renvoRunWindowsGetCurrentProcess(), base, memorySize,
	) == 0 {
		renvoRunWindowsRelease(base)
		return -1
	}
	if s.stack == 0 {
		s.stack = renvoRunWindowsVirtualAlloc(
			0, renvoRunJITStackSize,
			renvoRunWindowsMemCommit|renvoRunWindowsMemReserve,
			renvoRunWindowsPageReadWrite,
		)
	}
	if s.stack == 0 {
		renvoRunWindowsRelease(base)
		return -1
	}
	programArgs := make([]string, 1, len(args)+1)
	programArgs[0] = script
	programArgs = append(programArgs, args...)
	argWords := renvoRunStringWords(programArgs)
	envWords := renvoRunStringWords(env)
	exitCode := renvoRunJITCall(
		base+entry,
		s.stack+renvoRunJITStackSize,
		renvoRunIntPointer(argWords),
		len(programArgs),
		renvoRunIntPointer(envWords),
		len(env),
	)
	if exitCode != 0 {
		renvoRunWindowsRelease(base)
		return exitCode
	}
	s.mappings = append(s.mappings, renvoRunMapping{base: base, size: memorySize})
	for i := 0; i < len(linkSymbols); i++ {
		symbol := linkSymbols[i]
		current := renvoRunPersistentSymbol{
			id: symbol.ID, address: base + symbol.Address, size: symbol.Size,
		}
		previous := s.findSymbol(symbol.ID)
		if previous >= 0 {
			s.symbols[previous] = current
		} else {
			s.symbols = append(s.symbols, current)
		}
	}
	return 0
}

func (s *LinkedImageSession) findSymbol(id int) int {
	for i := 0; i < len(s.symbols); i++ {
		if s.symbols[i].id == id {
			return i
		}
	}
	return -1
}

func (s *LinkedImageSession) Reset() {
	for i := 0; i < len(s.mappings); i++ {
		renvoRunWindowsRelease(s.mappings[i].base)
	}
	if s.stack != 0 {
		renvoRunWindowsRelease(s.stack)
	}
	for i := len(s.libraryHandles) - 1; i >= 0; i-- {
		renvoRunWindowsFreeLibrary(s.libraryHandles[i])
	}
	s.mappings = nil
	s.symbols = nil
	s.libraryHandles = nil
	s.stack = 0
}

func renvoRunWindowsRelease(address int) {
	if address != 0 {
		renvoRunWindowsVirtualFree(address, 0, renvoRunWindowsMemRelease)
	}
}

func renvoRunCString(value string) []byte {
	out := make([]byte, len(value)+1)
	for i := 0; i < len(value); i++ {
		out[i] = value[i]
	}
	return out
}

func renvoRunLoadNativeWord(address int, size int) int {
	value := 0
	for i := 0; i < size; i++ {
		value |= int(renvoRunLoadByte(address+i)) << (i * 8)
	}
	return value
}

func renvoRunStoreNativeWord(address int, value int, size int) {
	for i := 0; i < size; i++ {
		renvoRunStoreByte(address+i, byte(value>>(i*8)))
	}
}

func renvoRunWindowsPageAlign(value int) int {
	return (value + 4095) & -4096
}

func renvoRunWindowsPageFloor(value int) int {
	return value & -4096
}
