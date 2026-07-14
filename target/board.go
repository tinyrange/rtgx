// Package target defines host-side composition and resource checks for
// freestanding RTG artifacts. Instruction selection and board composition are
// intentionally separate contracts.
package target

import "fmt"

type RegionKind int

const (
	RegionFlash RegionKind = iota + 1
	RegionRAM
	RegionReserved
)

type Region struct {
	Name  string
	Kind  RegionKind
	Start uint64
	Size  uint64
}

func (r Region) End() uint64 {
	return safeEnd(r.Start, r.Size)
}

type SectionFlags uint32

const (
	SectionAlloc SectionFlags = 1 << iota
	SectionWrite
	SectionExec
)

type Section struct {
	Name        string
	Address     uint64
	Size        uint64
	LoadAddress uint64
	LoadSize    uint64
	Flags       SectionFlags
}

type Board struct {
	Name             string
	ISA              string
	ABI              string
	ObjectFormat     string
	Regions          []Region
	VectorAddress    uint64
	StackTop         uint64
	DefaultStackSize uint64
	StackGuardSize   uint64
	AllowedImports   []string
}

type Artifact struct {
	Entry         uint64
	VectorAddress uint64
	Sections      []Section
	Imports       []string
	HeapSize      uint64
	StackSize     uint64
}

type Usage struct {
	FlashUsed     uint64
	FlashCapacity uint64
	FlashFree     uint64
	RAMStatic     uint64
	HeapReserved  uint64
	StackReserved uint64
	GuardReserved uint64
	RAMCapacity   uint64
	RAMFree       uint64
}

type ViolationCode string

const (
	ViolationBoard            ViolationCode = "invalid-board"
	ViolationSectionRegion    ViolationCode = "section-outside-region"
	ViolationSectionOverlap   ViolationCode = "section-overlap"
	ViolationReservedOverlap  ViolationCode = "reserved-overlap"
	ViolationLoadRegion       ViolationCode = "load-outside-flash"
	ViolationFlashBudget      ViolationCode = "flash-budget"
	ViolationRAMBudget        ViolationCode = "ram-budget"
	ViolationStackOverlap     ViolationCode = "stack-overlap"
	ViolationEntry            ViolationCode = "bad-entry"
	ViolationVector           ViolationCode = "bad-vector"
	ViolationUnresolvedImport ViolationCode = "forbidden-import"
)

type Violation struct {
	Code    ViolationCode
	Subject string
	Detail  string
}

func (v Violation) Error() string {
	if v.Subject == "" {
		return fmt.Sprintf("%s: %s", v.Code, v.Detail)
	}
	return fmt.Sprintf("%s %s: %s", v.Code, v.Subject, v.Detail)
}

type Validation struct {
	Usage      Usage
	Violations []Violation
}

func (v Validation) OK() bool {
	return len(v.Violations) == 0
}

func Validate(board Board, artifact Artifact) Validation {
	var result Validation
	flash, ram, boardViolations := validateBoard(board)
	result.Violations = append(result.Violations, boardViolations...)
	var flashCapacityOverflow bool
	var ramCapacityOverflow bool
	result.Usage.FlashCapacity, flashCapacityOverflow = regionCapacity(flash)
	result.Usage.RAMCapacity, ramCapacityOverflow = regionCapacity(ram)
	if flashCapacityOverflow || ramCapacityOverflow {
		result.Violations = append(result.Violations, Violation{Code: ViolationBoard, Detail: "aggregate memory-region capacity overflows the address space"})
	}

	stackSize := artifact.StackSize
	if stackSize == 0 {
		stackSize = board.DefaultStackSize
	}
	result.Usage.HeapReserved = artifact.HeapSize
	result.Usage.StackReserved = stackSize
	result.Usage.GuardReserved = board.StackGuardSize

	stackReservation, stackOverflow := saturatingAdd(stackSize, board.StackGuardSize)
	stackStart := uint64(0)
	stackValid := !stackOverflow && stackReservation <= board.StackTop
	if stackValid {
		stackStart = board.StackTop - stackReservation
	}
	flashUsageOverflow := false
	ramUsageOverflow := false

	for i := range artifact.Sections {
		section := artifact.Sections[i]
		if section.Flags&SectionAlloc == 0 || section.Size == 0 {
			continue
		}
		want := RegionFlash
		if section.Flags&SectionWrite != 0 {
			want = RegionRAM
		}
		if !containedByKind(board.Regions, want, section.Address, section.Size) {
			result.Violations = append(result.Violations, Violation{Code: ViolationSectionRegion, Subject: section.Name, Detail: fmt.Sprintf("%#x..%#x is not in %s", section.Address, safeEnd(section.Address, section.Size), regionKindName(want))})
		}
		if want == RegionFlash {
			result.Usage.FlashUsed, flashUsageOverflow = addWithOverflow(result.Usage.FlashUsed, section.Size, flashUsageOverflow)
		} else {
			result.Usage.RAMStatic, ramUsageOverflow = addWithOverflow(result.Usage.RAMStatic, section.Size, ramUsageOverflow)
			if stackValid && rangesOverlap(section.Address, section.Size, stackStart, stackReservation) {
				result.Violations = append(result.Violations, Violation{Code: ViolationStackOverlap, Subject: section.Name, Detail: "section overlaps configured stack or guard"})
			}
		}
		for _, reserved := range board.Regions {
			if reserved.Kind == RegionReserved && rangesOverlap(section.Address, section.Size, reserved.Start, reserved.Size) {
				result.Violations = append(result.Violations, Violation{Code: ViolationReservedOverlap, Subject: section.Name, Detail: "overlaps " + reserved.Name})
			}
		}
		if section.LoadSize > 0 {
			if !containedByRegions(flash, section.LoadAddress, section.LoadSize) {
				result.Violations = append(result.Violations, Violation{Code: ViolationLoadRegion, Subject: section.Name, Detail: "load image is not in flash"})
			}
			if !(want == RegionFlash && section.LoadAddress == section.Address && section.LoadSize == section.Size) {
				result.Usage.FlashUsed, flashUsageOverflow = addWithOverflow(result.Usage.FlashUsed, section.LoadSize, flashUsageOverflow)
			}
		}
		for j := 0; j < i; j++ {
			other := artifact.Sections[j]
			if other.Flags&SectionAlloc != 0 && rangesOverlap(section.Address, section.Size, other.Address, other.Size) {
				result.Violations = append(result.Violations, Violation{Code: ViolationSectionOverlap, Subject: section.Name, Detail: "overlaps " + other.Name})
			}
		}
	}

	if flashUsageOverflow || result.Usage.FlashUsed > result.Usage.FlashCapacity {
		result.Violations = append(result.Violations, Violation{Code: ViolationFlashBudget, Detail: fmt.Sprintf("uses %d bytes; capacity is %d", result.Usage.FlashUsed, result.Usage.FlashCapacity)})
	} else {
		result.Usage.FlashFree = result.Usage.FlashCapacity - result.Usage.FlashUsed
	}
	ramNeed, ramNeedOverflow := saturatingAdd(result.Usage.RAMStatic, result.Usage.HeapReserved)
	ramNeed, ramNeedOverflow = addWithOverflow(ramNeed, result.Usage.StackReserved, ramNeedOverflow)
	ramNeed, ramNeedOverflow = addWithOverflow(ramNeed, result.Usage.GuardReserved, ramNeedOverflow)
	if ramUsageOverflow || ramNeedOverflow || ramNeed > result.Usage.RAMCapacity {
		result.Violations = append(result.Violations, Violation{Code: ViolationRAMBudget, Detail: fmt.Sprintf("needs %d bytes; capacity is %d", ramNeed, result.Usage.RAMCapacity)})
	} else {
		result.Usage.RAMFree = result.Usage.RAMCapacity - ramNeed
	}

	if artifact.VectorAddress != board.VectorAddress || !addressInAllocatedSection(artifact.Sections, artifact.VectorAddress, SectionExec) {
		result.Violations = append(result.Violations, Violation{Code: ViolationVector, Detail: fmt.Sprintf("vector address %#x; want %#x in an executable section", artifact.VectorAddress, board.VectorAddress)})
	}
	if !addressInAllocatedSection(artifact.Sections, artifact.Entry, SectionExec) {
		result.Violations = append(result.Violations, Violation{Code: ViolationEntry, Detail: fmt.Sprintf("entry %#x is not in an executable section", artifact.Entry)})
	}
	for _, name := range artifact.Imports {
		if !stringInList(board.AllowedImports, name) {
			result.Violations = append(result.Violations, Violation{Code: ViolationUnresolvedImport, Subject: name, Detail: "not provided by the freestanding runtime contract"})
		}
	}
	return result
}

func validateBoard(board Board) ([]Region, []Region, []Violation) {
	var flash []Region
	var ram []Region
	var violations []Violation
	if board.Name == "" || board.ISA == "" || board.ABI == "" || board.ObjectFormat == "" {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: "name, ISA, ABI, and object format are required"})
	}
	for i, region := range board.Regions {
		if _, valid := checkedEnd(region.Start, region.Size); region.Name == "" || !valid {
			violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "region has an invalid name or range"})
		}
		if region.Kind == RegionFlash {
			flash = append(flash, region)
		} else if region.Kind == RegionRAM {
			ram = append(ram, region)
		} else if region.Kind != RegionReserved {
			violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "unknown region kind"})
		}
		for j := 0; j < i; j++ {
			other := board.Regions[j]
			if region.Kind != RegionReserved && other.Kind != RegionReserved && rangesOverlap(region.Start, region.Size, other.Start, other.Size) {
				violations = append(violations, Violation{Code: ViolationBoard, Subject: region.Name, Detail: "overlaps board region " + other.Name})
			}
		}
	}
	if len(flash) == 0 || len(ram) == 0 {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: "at least one flash and one RAM region are required"})
	}
	if board.StackTop == 0 || !containedByRegions(ram, board.StackTop-1, 1) {
		violations = append(violations, Violation{Code: ViolationBoard, Detail: "stack top is outside RAM"})
	}
	return flash, ram, violations
}

func regionCapacity(regions []Region) (uint64, bool) {
	var total uint64
	overflow := false
	for _, region := range regions {
		total, overflow = addWithOverflow(total, region.Size, overflow)
	}
	return total, overflow
}

func containedByKind(regions []Region, kind RegionKind, start uint64, size uint64) bool {
	for _, region := range regions {
		if region.Kind == kind && rangeContains(region.Start, region.Size, start, size) {
			return true
		}
	}
	return false
}

func containedByRegions(regions []Region, start uint64, size uint64) bool {
	for _, region := range regions {
		if rangeContains(region.Start, region.Size, start, size) {
			return true
		}
	}
	return false
}

func rangeContains(outerStart uint64, outerSize uint64, innerStart uint64, innerSize uint64) bool {
	outerEnd, outerValid := checkedEnd(outerStart, outerSize)
	innerEnd, innerValid := checkedEnd(innerStart, innerSize)
	return outerValid && innerValid && innerStart >= outerStart && innerEnd <= outerEnd
}

func rangesOverlap(leftStart uint64, leftSize uint64, rightStart uint64, rightSize uint64) bool {
	leftEnd, leftValid := checkedEnd(leftStart, leftSize)
	rightEnd, rightValid := checkedEnd(rightStart, rightSize)
	return leftValid && rightValid && leftStart < rightEnd && rightStart < leftEnd
}

func safeEnd(start uint64, size uint64) uint64 {
	if end, valid := checkedEnd(start, size); valid {
		return end
	}
	return ^uint64(0)
}

func checkedEnd(start uint64, size uint64) (uint64, bool) {
	if size == 0 || start > ^uint64(0)-size {
		return 0, false
	}
	return start + size, true
}

func saturatingAdd(left uint64, right uint64) (uint64, bool) {
	if left > ^uint64(0)-right {
		return ^uint64(0), true
	}
	return left + right, false
}

func addWithOverflow(left uint64, right uint64, alreadyOverflowed bool) (uint64, bool) {
	if alreadyOverflowed {
		return ^uint64(0), true
	}
	return saturatingAdd(left, right)
}

func addressInAllocatedSection(sections []Section, address uint64, flags SectionFlags) bool {
	for _, section := range sections {
		if section.Flags&SectionAlloc != 0 && section.Flags&flags == flags && rangeContains(section.Address, section.Size, address, 1) {
			return true
		}
	}
	return false
}

func stringInList(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func regionKindName(kind RegionKind) string {
	if kind == RegionFlash {
		return "flash"
	}
	if kind == RegionRAM {
		return "RAM"
	}
	return "reserved memory"
}
