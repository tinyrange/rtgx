//go:build renvo && linux

package main

import "renvo.dev/internal/driver"

// syscall is the frontend's generic Linux syscall intrinsic. Keeping its call
// in the root package prevents linked package symbol qualification from
// changing the compiler-recognized intrinsic name.
func syscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return 0
}

func runLinuxSyscall(number int, first int, second int, third int, fourth int, fifth int, sixth int) int {
	return syscall(number, first, second, third, fourth, fifth, sixth)
}

// renvo_runtime_CallJIT is lowered by the backend to an isolated-stack
// indirect call. Its body is only a source-level declaration for the frontend.
func renvo_runtime_CallJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return 0
}

func runLinuxJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return renvo_runtime_CallJIT(entry, stackTop, argsData, argsLen, envData, envLen)
}

func configureRunPlatform() {
	driver.SetRunLinuxSyscall(runLinuxSyscall)
	driver.SetRunJITCall(runLinuxJIT)
}
