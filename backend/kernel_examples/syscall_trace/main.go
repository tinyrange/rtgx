package main

// renvo:module-license GPL

const (
	traceCapacity   = 512 * 1024
	ringBytesPerCPU = 256 * 1024
	maxCPUs         = 64
	maxRecordText   = 320
	sysEnterPrefix  = uint64(0x65746e655f737973)
	sysExitPrefix   = uint64(0x746978655f737973)
)

var (
	traceBuffer      [traceCapacity]byte
	ringLockClassKey [64]byte
)

type syscallRecord struct {
	kind   uint64
	number uint64
	arg0   uint64
	arg1   uint64
	arg2   uint64
	arg3   uint64
	arg4   uint64
	arg5   uint64
	result int64
}

type cNamePrefix struct {
	value uint64
}

type kernelTracepoint struct {
	name *cNamePrefix
}

// ptRegs matches struct pt_regs from the x86-64 kernel BTF.
type ptRegs struct {
	r15, r14, r13, r12, bp, bx uint64
	r11, r10, r9, r8, ax, cx   uint64
	dx, si, di, origAx         uint64
	ip, cs, flags, sp, ss      uint64
}

// renvo:linkstatic kernel,for_each_kernel_tracepoint
func kernelForEachTracepoint(callback func(*kernelTracepoint, uintptr), data uintptr) {}

// renvo:linkstatic kernel,tracepoint_probe_register
func kernelTracepointRegister(tracepoint *kernelTracepoint, callback func(uintptr, *ptRegs, int64), data uintptr) int32 {
	return 0
}

// renvo:linkstatic kernel,tracepoint_probe_unregister
func kernelTracepointUnregister(tracepoint *kernelTracepoint, callback func(uintptr, *ptRegs, int64), data uintptr) int32 {
	return 0
}

// renvo:linkstatic kernel,synchronize_rcu
func kernelSynchronizeRCU() {}

// renvo:linkstatic kernel,filp_open
func kernelFileOpen(path *[24]byte, flags int32, mode uint16) uintptr { return 0 }

// renvo:linkstatic kernel,kernel_write
func kernelFileWrite(file uintptr, data *[traceCapacity]byte, count int, position *int64) int64 {
	return 0
}

// renvo:linkstatic kernel,filp_close
func kernelFileClose(file uintptr, owner uintptr) int32 { return 0 }

// renvo:linkstatic kernel,__ring_buffer_alloc
func kernelRingAlloc(size int, flags uint32, key *[64]byte) uintptr { return 0 }

// renvo:linkstatic kernel,ring_buffer_write
func kernelRingWrite(buffer uintptr, length int, record *syscallRecord) int32 { return 0 }

// renvo:linkstatic kernel,ring_buffer_consume
func kernelRingConsume(buffer uintptr, cpu int32, timestamp uintptr, lost uintptr) uintptr { return 0 }

// renvo:linkstatic kernel,ring_buffer_event_data
func kernelRingEventData(event uintptr) *syscallRecord { return nil }

// renvo:linkstatic kernel,ring_buffer_free
func kernelRingFree(buffer uintptr) {}

type traceWriter struct {
	buffer []byte
	length int
}

func (w *traceWriter) text(text string) {
	for i := 0; i < len(text) && w.length < len(w.buffer); i++ {
		w.buffer[w.length] = text[i]
		w.length++
	}
}

func (w *traceWriter) unsigned(value uint64) {
	var digits [20]byte
	count := 0
	if value == 0 {
		w.text("0")
		return
	}
	for value != 0 {
		digits[count] = byte(value%10) + '0'
		count++
		value /= 10
	}
	for count > 0 && w.length < len(w.buffer) {
		count--
		w.buffer[w.length] = digits[count]
		w.length++
	}
}

func (w *traceWriter) signed(value int64) {
	if value < 0 {
		w.text("-")
		w.unsigned(uint64(-(value + 1)) + 1)
		return
	}
	w.unsigned(uint64(value))
}

func (w *traceWriter) hex(value uint64) {
	w.text("0x")
	started := false
	for shift := 60; shift >= 0; shift -= 4 {
		digit := byte(value >> uint(shift) & 15)
		if digit == 0 && !started && shift != 0 {
			continue
		}
		started = true
		if digit < 10 {
			digit += '0'
		} else {
			digit += 'a' - 10
		}
		if w.length < len(w.buffer) {
			w.buffer[w.length] = digit
			w.length++
		}
	}
}

func (w *traceWriter) syscall(record *syscallRecord) {
	if record.kind == 0 {
		w.text("enter")
	} else {
		w.text("exit")
	}
	w.text(" nr=")
	if record.number > 1024 {
		w.text("?")
	} else {
		w.unsigned(record.number)
	}
	if record.kind != 0 {
		w.text(" ret=")
		w.signed(record.result)
	}
	w.text(" args=[")
	arguments := [6]uint64{record.arg0, record.arg1, record.arg2, record.arg3, record.arg4, record.arg5}
	for i, argument := range arguments {
		if i != 0 {
			w.text(",")
		}
		w.hex(argument)
	}
	w.text("]\n")
}

type syscallTracer struct {
	file       uintptr
	position   int64
	ring       uintptr
	enterpoint *kernelTracepoint
	exitpoint  *kernelTracepoint
	running    bool
}

var tracer syscallTracer

func findSyscallTracepoints(point *kernelTracepoint, data uintptr) {
	if point == nil || point.name == nil {
		return
	}
	switch point.name.value {
	case sysEnterPrefix:
		tracer.enterpoint = point
	case sysExitPrefix:
		tracer.exitpoint = point
	}
}

func syscallEnter(data uintptr, registers *ptRegs, number int64) {
	if !tracer.running || registers == nil {
		return
	}
	record := syscallRecord{number: uint64(number), arg0: registers.di, arg1: registers.si, arg2: registers.dx, arg3: registers.r10, arg4: registers.r8, arg5: registers.r9}
	kernelRingWrite(tracer.ring, 72, &record)
}

func syscallExit(data uintptr, registers *ptRegs, result int64) {
	if !tracer.running || registers == nil {
		return
	}
	record := syscallRecord{kind: 1, number: registers.origAx, arg0: registers.di, arg1: registers.si, arg2: registers.dx, arg3: registers.r10, arg4: registers.r8, arg5: registers.r9, result: result}
	kernelRingWrite(tracer.ring, 72, &record)
}

func (t *syscallTracer) release() {
	if t.file != 0 {
		kernelFileClose(t.file, 0)
		t.file = 0
	}
	if t.ring != 0 {
		kernelRingFree(t.ring)
		t.ring = 0
	}
}

func (t *syscallTracer) start() {
	t.ring = kernelRingAlloc(ringBytesPerCPU, 1, &ringLockClassKey)
	if t.ring == 0 {
		return
	}

	path := [24]byte{'/', 't', 'm', 'p', '/', 'r', 'e', 'n', 'v', 'o', '-', 's', 'y', 's', 'c', 'a', 'l', 'l', 's', '.', 'l', 'o', 'g', 0}
	t.file = kernelFileOpen(&path, 577, 384)
	if t.file == 0 {
		t.release()
		print("renvo syscall trace: could not open output file\n")
		return
	}

	kernelForEachTracepoint(findSyscallTracepoints, 0)
	if t.enterpoint == nil || t.exitpoint == nil {
		t.release()
		print("renvo syscall trace: tracepoints were not found\n")
		return
	}
	if kernelTracepointRegister(t.enterpoint, syscallEnter, 0) != 0 {
		t.release()
		return
	}
	if kernelTracepointRegister(t.exitpoint, syscallExit, 0) != 0 {
		kernelTracepointUnregister(t.enterpoint, syscallEnter, 0)
		kernelSynchronizeRCU()
		t.release()
		return
	}

	t.running = true
	print("renvo syscall trace: recording /tmp/renvo-syscalls.log\n")
}

func (t *syscallTracer) stop() {
	t.running = false
	if t.enterpoint != nil {
		kernelTracepointUnregister(t.enterpoint, syscallEnter, 0)
	}
	if t.exitpoint != nil {
		kernelTracepointUnregister(t.exitpoint, syscallExit, 0)
	}
	kernelSynchronizeRCU()

	writer := traceWriter{buffer: traceBuffer[:]}
	for cpu := 0; cpu < maxCPUs && writer.length <= traceCapacity-maxRecordText; cpu++ {
		for writer.length <= traceCapacity-maxRecordText {
			event := kernelRingConsume(t.ring, int32(cpu), 0, 0)
			if event == 0 {
				break
			}
			record := kernelRingEventData(event)
			if record != nil {
				writer.syscall(record)
			}
		}
	}
	if t.file != 0 && writer.length != 0 {
		kernelFileWrite(t.file, &traceBuffer, writer.length, &t.position)
	}
	t.release()
	print("renvo syscall trace: stopped\n")
}

func appMain() {
	tracer.start()
}

func moduleExit() {
	tracer.stop()
}
