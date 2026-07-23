//go:build renvo && darwin && arm64

package driver

// RunNativeLinkedImage enters a callable Darwin image in-process. A temporary
// executable cannot be used here: linked images expose the Renvo JIT entry ABI,
// not dyld's process-entry ABI.
func RunNativeLinkedImage(native []byte, script string, args []string, env []string) int {
	var session LinkedImageSession
	session.Prepare()
	exitCode := session.Run(native, script, args, env)
	session.Reset()
	return exitCode
}
