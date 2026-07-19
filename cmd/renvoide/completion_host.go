//go:build !renvo

package main

import "renvo.dev/internal/driver"

func completionReadDir(path string) ([]driver.DirEntry, bool) {
	return (driver.OSFS{}).ReadDir(path)
}
func completionReadFile(path string) ([]byte, bool) { return (driver.OSFS{}).ReadFile(path) }
func completionStdRoot(env []string) string         { return driver.StdRootFromEnv(env) }
func completionModuleCache(env []string) string     { return driver.EnvValue(env, driver.ModuleCacheEnv) }
