//go:build renvo && windows

package main

const (
	replWindowsStdInput  = -10
	replWindowsStdOutput = -11

	replWindowsEnableProcessedInput        = 0x0001
	replWindowsEnableLineInput             = 0x0002
	replWindowsEnableEchoInput             = 0x0004
	replWindowsEnableQuickEditMode         = 0x0040
	replWindowsEnableExtendedFlags         = 0x0080
	replWindowsEnableVirtualTerminalInput  = 0x0200
	replWindowsEnableVirtualTerminalOutput = 0x0004
)

var replWindowsInputHandle int
var replWindowsOutputHandle int
var replWindowsSavedInputMode int
var replWindowsSavedOutputMode int
var replWindowsTerminalActive bool

// renvo:linkstatic kernel32.dll,GetStdHandle
func replWindowsGetStdHandle(kind int) int { return 0 }

// renvo:linkstatic kernel32.dll,GetConsoleMode
func replWindowsGetConsoleMode(handle int, mode *int) int { return 0 }

// renvo:linkstatic kernel32.dll,SetConsoleMode
func replWindowsSetConsoleMode(handle int, mode int) int { return 0 }

func replTerminalEnable() bool {
	inputHandle := replWindowsGetStdHandle(replWindowsStdInput)
	outputHandle := replWindowsGetStdHandle(replWindowsStdOutput)
	inputMode := 0
	outputMode := 0
	if inputHandle == 0 || outputHandle == 0 ||
		replWindowsGetConsoleMode(inputHandle, &inputMode) == 0 ||
		replWindowsGetConsoleMode(outputHandle, &outputMode) == 0 {
		return false
	}
	rawInput := inputMode
	rawInput = replWindowsClearBits(rawInput,
		replWindowsEnableProcessedInput|
			replWindowsEnableLineInput|
			replWindowsEnableEchoInput|
			replWindowsEnableQuickEditMode)
	rawInput = rawInput | replWindowsEnableExtendedFlags | replWindowsEnableVirtualTerminalInput
	rawOutput := outputMode | replWindowsEnableVirtualTerminalOutput
	if replWindowsSetConsoleMode(outputHandle, rawOutput) == 0 {
		return false
	}
	if replWindowsSetConsoleMode(inputHandle, rawInput) == 0 {
		replWindowsSetConsoleMode(outputHandle, outputMode)
		return false
	}
	replWindowsInputHandle = inputHandle
	replWindowsOutputHandle = outputHandle
	replWindowsSavedInputMode = inputMode
	replWindowsSavedOutputMode = outputMode
	replWindowsTerminalActive = true
	return true
}

func replTerminalDisable() {
	if !replWindowsTerminalActive {
		return
	}
	replWindowsSetConsoleMode(replWindowsInputHandle, replWindowsSavedInputMode)
	replWindowsSetConsoleMode(replWindowsOutputHandle, replWindowsSavedOutputMode)
	replWindowsTerminalActive = false
}

func replWindowsClearBits(value int, bits int) int {
	return value ^ (value & bits)
}
