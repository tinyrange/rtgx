//go:build renvo && windows

package driver

import renvoos "renvo.dev/std/os"

// renvo:linkstatic kernel32.dll,GetTempPathA
func renvoRunWindowsGetTempPath(size int, path *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,GetTempFileNameA
func renvoRunWindowsGetTempFileName(path *byte, prefix *byte, unique int, output *byte) int {
	return 0
}

// renvo:linkstatic kernel32.dll,CreateProcessA
func renvoRunWindowsCreateProcess(application int, command int, processAttributes int, threadAttributes int, inheritHandles int, flags int, environment int, directory int, startup int, processInfo int) int {
	return 0
}

// renvo:linkstatic kernel32.dll,WaitForSingleObject
func renvoRunWindowsWait(handle int, milliseconds int) int { return -1 }

// renvo:linkstatic kernel32.dll,GetExitCodeProcess
func renvoRunWindowsGetExitCode(handle int, code *byte) int { return 0 }

// renvo:linkstatic kernel32.dll,CloseHandle
func renvoRunWindowsCloseHandle(handle int) int { return 0 }

// renvo:linkstatic kernel32.dll,DeleteFileA
func renvoRunWindowsDeleteFile(path *byte) int { return 0 }

// RunNativeLinkedImage executes a Windows image through the native process
// bridge.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	pathBuffer := make([]byte, 260)
	pathLength := renvoRunWindowsGetTempPath(len(pathBuffer), &pathBuffer[0])
	if pathLength <= 0 || pathLength >= len(pathBuffer) {
		return -1
	}
	prefix := []byte{'R', 'N', 'V', 0}
	fileBuffer := make([]byte, 260)
	if renvoRunWindowsGetTempFileName(&pathBuffer[0], &prefix[0], 0, &fileBuffer[0]) == 0 {
		return -1
	}
	fileEnd := 0
	for fileEnd < len(fileBuffer) && fileBuffer[fileEnd] != 0 {
		fileEnd++
	}
	path := string(fileBuffer[:fileEnd])
	if renvoos.WriteFile(path, native, 0755) != nil {
		renvoRunWindowsDeleteFile(&fileBuffer[0])
		return -1
	}
	command := renvoRunWindowsCommandLine(script, args)
	environment := renvoRunWindowsEnvironment(env)
	startup := make([]byte, renvoRunWindowsStartupSize())
	renvoRunWindowsPut32(startup, 0, len(startup))
	processInfo := make([]byte, renvoRunWindowsProcessInfoSize())
	created := renvoRunWindowsCreateProcess(
		renvoRunBytePointer(fileBuffer),
		renvoRunBytePointer(command),
		0,
		0,
		0,
		0,
		renvoRunBytePointer(environment),
		0,
		renvoRunBytePointer(startup),
		renvoRunBytePointer(processInfo),
	)
	if created == 0 {
		renvoRunWindowsDeleteFile(&fileBuffer[0])
		return -1
	}
	pointerSize := renvoRunWindowsPointerSize()
	processHandle := renvoRunWindowsReadWord(processInfo, 0, pointerSize)
	threadHandle := renvoRunWindowsReadWord(processInfo, pointerSize, pointerSize)
	renvoRunWindowsCloseHandle(threadHandle)
	waited := renvoRunWindowsWait(processHandle, -1)
	codeBytes := make([]byte, 4)
	gotCode := renvoRunWindowsGetExitCode(processHandle, &codeBytes[0])
	renvoRunWindowsCloseHandle(processHandle)
	renvoRunWindowsDeleteFile(&fileBuffer[0])
	if waited != 0 || gotCode == 0 {
		return -1
	}
	return renvoRunWindowsReadWord(codeBytes, 0, 4)
}

func renvoRunWindowsCommandLine(path string, args []string) []byte {
	out := renvoRunWindowsAppendArgument(nil, path)
	for i := 0; i < len(args); i++ {
		out = append(out, ' ')
		out = renvoRunWindowsAppendArgument(out, args[i])
	}
	return append(out, 0)
}

func renvoRunWindowsAppendArgument(out []byte, value string) []byte {
	quote := len(value) == 0
	for i := 0; i < len(value); i++ {
		if value[i] == ' ' || value[i] == '\t' || value[i] == '"' {
			quote = true
		}
	}
	if !quote {
		return append(out, []byte(value)...)
	}
	out = append(out, '"')
	slashes := 0
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '\\' {
			slashes++
			continue
		}
		if ch == '"' {
			for j := 0; j < slashes*2+1; j++ {
				out = append(out, '\\')
			}
			out = append(out, '"')
			slashes = 0
			continue
		}
		for j := 0; j < slashes; j++ {
			out = append(out, '\\')
		}
		slashes = 0
		out = append(out, ch)
	}
	for j := 0; j < slashes*2; j++ {
		out = append(out, '\\')
	}
	return append(out, '"')
}

func renvoRunWindowsEnvironment(env []string) []byte {
	if len(env) == 0 {
		return nil
	}
	var out []byte
	for i := 0; i < len(env); i++ {
		out = append(out, []byte(env[i])...)
		out = append(out, 0)
	}
	return append(out, 0)
}

func renvoRunWindowsPut32(out []byte, at int, value int) {
	out[at] = byte(value)
	out[at+1] = byte(value >> 8)
	out[at+2] = byte(value >> 16)
	out[at+3] = byte(value >> 24)
}

func renvoRunWindowsReadWord(data []byte, at int, size int) int {
	value := 0
	for i := 0; i < size; i++ {
		value = value | int(data[at+i])<<(i*8)
	}
	return value
}
