//go:build renvo && windows

package driver

// RunNativeLinkedImage enters a callable PE image in-process. Spawning the
// bytes as an executable would invoke the Windows process-entry ABI instead of
// the Renvo JIT entry ABI.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	var session LinkedImageSession
	session.Prepare()
	exitCode := session.Run(native, script, args, env)
	session.Reset()
	return exitCode
}
