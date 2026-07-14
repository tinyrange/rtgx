package target

import "testing"

func TestCH32V003ValidOmnibusLayout(t *testing.T) {
	artifact := validCH32Artifact()
	got := Validate(CH32V003(), artifact)
	if !got.OK() {
		t.Fatalf("valid layout rejected: %v", got.Violations)
	}
	if got.Usage.FlashCapacity != 16*1024 || got.Usage.RAMCapacity != 2*1024 {
		t.Fatalf("capacities = %+v", got.Usage)
	}
	if got.Usage.RAMStatic != 768 || got.Usage.HeapReserved != 256 || got.Usage.StackReserved != 512 || got.Usage.GuardReserved != 64 || got.Usage.RAMFree != 448 {
		t.Fatalf("RAM accounting = %+v", got.Usage)
	}
	if got.Usage.FlashUsed != 3520 || got.Usage.FlashFree != 12864 {
		t.Fatalf("flash accounting = %+v", got.Usage)
	}
}

func TestFreestandingLayoutFailuresAreLocalized(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Board, *Artifact)
		code ViolationCode
	}{
		{name: "flash range", edit: func(_ *Board, a *Artifact) { a.Sections[1].Address = 0x08004000 }, code: ViolationSectionRegion},
		{name: "overflowing section range", edit: func(_ *Board, a *Artifact) {
			a.Sections[1].Address = ^uint64(0) - 3
			a.Sections[1].Size = 8
		}, code: ViolationSectionRegion},
		{name: "load range", edit: func(_ *Board, a *Artifact) { a.Sections[3].LoadAddress = 0x08004000 }, code: ViolationLoadRegion},
		{name: "overflowing load range", edit: func(_ *Board, a *Artifact) {
			a.Sections[3].LoadAddress = ^uint64(0) - 3
			a.Sections[3].LoadSize = 8
		}, code: ViolationLoadRegion},
		{name: "section overlap", edit: func(_ *Board, a *Artifact) { a.Sections[2].Address = a.Sections[1].Address }, code: ViolationSectionOverlap},
		{name: "stack overlap", edit: func(_ *Board, a *Artifact) { a.Sections[4].Address = 0x20000600 }, code: ViolationStackOverlap},
		{name: "RAM budget", edit: func(_ *Board, a *Artifact) { a.HeapSize = 2048 }, code: ViolationRAMBudget},
		{name: "overflowing RAM budget", edit: func(_ *Board, a *Artifact) { a.HeapSize = ^uint64(0) }, code: ViolationRAMBudget},
		{name: "entry", edit: func(_ *Board, a *Artifact) { a.Entry = 0x20000000 }, code: ViolationEntry},
		{name: "vector", edit: func(_ *Board, a *Artifact) { a.VectorAddress++ }, code: ViolationVector},
		{name: "hosted import", edit: func(_ *Board, a *Artifact) { a.Imports = []string{"printf"} }, code: ViolationUnresolvedImport},
		{name: "reserved", edit: func(b *Board, _ *Artifact) {
			b.Regions = append(b.Regions, Region{Name: "debug", Kind: RegionReserved, Start: 0x20000000, Size: 16})
		}, code: ViolationReservedOverlap},
		{name: "overflowing board region", edit: func(b *Board, _ *Artifact) {
			b.Regions[0].Start = ^uint64(0) - 3
			b.Regions[0].Size = 8
		}, code: ViolationBoard},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			board := CH32V003()
			artifact := validCH32Artifact()
			test.edit(&board, &artifact)
			got := Validate(board, artifact)
			if !hasViolation(got, test.code) {
				t.Fatalf("violations = %v; want %s", got.Violations, test.code)
			}
		})
	}
}

func TestAllowedFreestandingImportIsExplicit(t *testing.T) {
	board := CH32V003()
	board.AllowedImports = []string{"rtg_board_result_commit"}
	artifact := validCH32Artifact()
	artifact.Imports = []string{"rtg_board_result_commit"}
	if got := Validate(board, artifact); !got.OK() {
		t.Fatalf("explicit board import rejected: %v", got.Violations)
	}
}

func validCH32Artifact() Artifact {
	return Artifact{
		Entry:         0x08000040,
		VectorAddress: 0x08000000,
		HeapSize:      256,
		Sections: []Section{
			{Name: ".vectors", Address: 0x08000000, Size: 64, LoadAddress: 0x08000000, LoadSize: 64, Flags: SectionAlloc | SectionExec},
			{Name: ".text", Address: 0x08000040, Size: 3000, LoadAddress: 0x08000040, LoadSize: 3000, Flags: SectionAlloc | SectionExec},
			{Name: ".rodata", Address: 0x08000c00, Size: 200, LoadAddress: 0x08000c00, LoadSize: 200, Flags: SectionAlloc},
			{Name: ".data", Address: 0x20000000, Size: 256, LoadAddress: 0x08000d00, LoadSize: 256, Flags: SectionAlloc | SectionWrite},
			{Name: ".bss", Address: 0x20000100, Size: 512, Flags: SectionAlloc | SectionWrite},
		},
	}
}

func hasViolation(result Validation, code ViolationCode) bool {
	for _, violation := range result.Violations {
		if violation.Code == code {
			return true
		}
	}
	return false
}
