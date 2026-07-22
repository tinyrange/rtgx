# Renvo kernel module examples

Generate complete bindings for the running kernel:

```sh
go run ./cmd/renvo-kernel-bindings -o kernel_generated.go
```

Generate a compact source file for selected functions:

```sh
go run ./cmd/renvo-kernel-bindings \
  -package main -types=false \
  -symbols=ktime_get,get_random_u32,jiffies_to_msecs \
  -o backend/kernel_examples/clock/kernel_bindings.go
```

Compile the clock example with the selected generated file in the same package:

```sh
renvo -mode=kernel-module -o clock.ko \
  backend/kernel_examples/clock/main.go \
  backend/kernel_examples/clock/kernel_bindings.go
```

Kernel modules default to the honest non-GPL license `Proprietary`. Declare a
different license in source when appropriate:

```go
// renvo:module-license GPL
```

Renvo rejects imports of `EXPORT_SYMBOL_GPL` symbols unless this directive
names a GPL-compatible kernel module license.

The `syscall_trace` example registers callbacks for the kernel's `sys_enter`
and `sys_exit` tracepoints and writes a bounded raw-argument trace to
`/tmp/renvo-syscalls.log`:

```sh
renvo -mode=kernel-module -o syscall_trace.ko \
  backend/kernel_examples/syscall_trace/main.go
```

This is deliberately a small observability demo. It records numeric pointer
values but never dereferences syscall arguments. Probe callbacks copy fixed
records into the kernel's lockless ring buffer; unload drains and formats up to
512 KiB before writing the file.

The generated full file contains exact-size raw storage for every BTF struct
and union, member type IDs and bit offsets, enums, every exported symbol and
CRC, and callable stubs for scalar/pointer prototypes of up to six ABI words.
Exported data symbols, aggregates, variadic functions, and wider prototypes are
retained as metadata but are not emitted as unsafe callable declarations.
