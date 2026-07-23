//go:build renvo && darwin && arm64

package driver

// renvo:linkstatic /usr/lib/libSystem.B.dylib,mkstemp
func renvoRunDarwinMkstemp(template *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,posix_spawn
func renvoRunDarwinPosixSpawn(pid *int, path *byte, actions int, attributes int, argv *int, env *int) int {
	return -1
}

// renvo:linkstatic /usr/lib/libSystem.B.dylib,waitpid
func renvoRunDarwinWaitpid(pid int, status *int, options int) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,unlink
func renvoRunDarwinUnlink(path *byte) int { return -1 }

// RunNativeLinkedImage executes a Darwin image through the native process
// bridge.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	template := []byte("/tmp/renvo-run-XXXXXX\x00")
	fd := renvoRunDarwinMkstemp(&template[0])
	if fd < 0 {
		return -1
	}
	written := 0
	for written < len(native) {
		n := write(fd, native[written:], -1)
		if n <= 0 {
			close(fd)
			renvoRunDarwinUnlink(&template[0])
			return -1
		}
		written += n
	}
	if chmod(fd, 493) != 0 {
		close(fd)
		renvoRunDarwinUnlink(&template[0])
		return -1
	}
	close(fd)
	argvStorage, argv := renvoRunCStringVector(script, args)
	envStorage, envp := renvoRunCStringVector("", env)
	_ = argvStorage
	_ = envStorage
	pid := 0
	spawned := renvoRunDarwinPosixSpawn(&pid, &template[0], 0, 0, &argv[0], &envp[0])
	if spawned != 0 {
		renvoRunDarwinUnlink(&template[0])
		return -1
	}
	status := 0
	waited := renvoRunDarwinWaitpid(pid, &status, 0)
	renvoRunDarwinUnlink(&template[0])
	if waited != pid {
		return -1
	}
	signal := status & 127
	if signal != 0 {
		return 128 + signal
	}
	return status >> 8 & 255
}
