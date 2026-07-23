//go:build renvo && !linux && !windows && !(darwin && arm64)

package driver

// LinkedImageSession is unavailable on hosts without the direct in-process
// Linux image loader. The REPL reports this explicitly instead of falling back
// to source replay or child processes.
type LinkedImageSession struct{}

func (s *LinkedImageSession) Prepare() {}

func (s *LinkedImageSession) Run(native []byte, script string, args []string, env []string) int {
	return -1
}

func (s *LinkedImageSession) Reset() {}
