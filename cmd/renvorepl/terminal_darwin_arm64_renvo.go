//go:build renvo && darwin && arm64

package main

const (
	replDarwinTermiosSize = 128
	replDarwinTCSANOW     = 0
	replDarwinOPOST       = 1
	replDarwinOFlagOffset = 8
)

var replDarwinSavedTermios []byte

// renvo:linkstatic /usr/lib/libSystem.B.dylib,tcgetattr
func replDarwinTCGetAttr(fd int, state *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,tcsetattr
func replDarwinTCSetAttr(fd int, action int, state *byte) int { return -1 }

// renvo:linkstatic /usr/lib/libSystem.B.dylib,cfmakeraw
func replDarwinMakeRaw(state *byte) {}

func replTerminalEnable() bool {
	saved := make([]byte, replDarwinTermiosSize)
	if replDarwinTCGetAttr(0, &saved[0]) != 0 {
		return false
	}
	raw := make([]byte, len(saved))
	copy(raw, saved)
	replDarwinMakeRaw(&raw[0])
	// Preserve normal newline processing for output produced by submissions.
	raw[replDarwinOFlagOffset] = raw[replDarwinOFlagOffset] | replDarwinOPOST
	if replDarwinTCSetAttr(0, replDarwinTCSANOW, &raw[0]) != 0 {
		return false
	}
	replDarwinSavedTermios = saved
	return true
}

func replTerminalDisable() {
	if len(replDarwinSavedTermios) == 0 {
		return
	}
	replDarwinTCSetAttr(0, replDarwinTCSANOW, &replDarwinSavedTermios[0])
	replDarwinSavedTermios = nil
}
