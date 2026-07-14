package target

// CH32V003 returns the first constrained forcing profile. It describes board
// composition only; the RV32EC instruction emitter and ilp32e object ABI remain
// reusable by other devices.
func CH32V003() Board {
	return Board{
		Name:             "wch-ch32v003",
		ISA:              "rv32ec",
		ABI:              "ilp32e",
		ObjectFormat:     "elf32-littleriscv",
		VectorAddress:    0x08000000,
		StackTop:         0x20000800,
		DefaultStackSize: 512,
		StackGuardSize:   64,
		Regions: []Region{
			{Name: "flash", Kind: RegionFlash, Start: 0x08000000, Size: 16 * 1024},
			{Name: "sram", Kind: RegionRAM, Start: 0x20000000, Size: 2 * 1024},
		},
	}
}
