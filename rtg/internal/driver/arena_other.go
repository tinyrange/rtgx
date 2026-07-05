//go:build rtg && !(linux && amd64)

package driver

func rtgFrontendCanResetArena() bool {
	return false
}
