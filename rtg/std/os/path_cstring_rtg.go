//go:build rtg && !wasi && !wasip1

package os

// Native open APIs consume a NUL-terminated path. Go strings forwarded from a
// caller are not necessarily followed by a zero byte, so always own the
// terminator at the runtime boundary.
func rtgPathCString(path string) string {
	return path + "\x00"
}
