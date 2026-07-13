//go:build rtg && linux && amd64

package driver

func rtgFrontendCanResetArena() bool {
	// Run the embedded backend from the compact persistent handoff assembled in
	// RunRTGCommand. Releasing the frontend's transient pages here prevents the
	// two compiler phases from accumulating in the same RSS peak.
	return true
}
