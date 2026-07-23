//go:build renvo && (windows || (darwin && arm64))

package main

import "renvo.dev/internal/driver"

// renvo_runtime_CallJIT is lowered to an isolated-stack indirect call. The
// linked-image ABI is deliberately platform neutral even though the outer
// executable uses the host operating system's native ABI.
func renvo_runtime_CallJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return 0
}

func runNativeJIT(entry int, stackTop int, argsData int, argsLen int, envData int, envLen int) int {
	return renvo_runtime_CallJIT(entry, stackTop, argsData, argsLen, envData, envLen)
}

func configureRunPlatform() {
	driver.SetRunJITCall(runNativeJIT)
}
