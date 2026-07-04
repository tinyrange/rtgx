//go:build rtg

package main

import "j5.nz/rtg/rtg/internal/driver"

func appMain(args []string, env []string) int {
	return driver.RunRTGCommand(args, env)
}
