//go:build renvo

package main

import "renvo.dev/internal/driver"

func appMain(args []string, env []string) int {
	configureRunPlatform()
	return driver.RunRenvoCommand(args, env)
}
