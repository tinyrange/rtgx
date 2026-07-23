//go:build !renvo

package repl

import "renvo.dev/internal/driver"

func replCompletionFS() driver.SourceFS { return driver.OSFS{} }

func replCompletionTarget() string { return driver.DefaultTarget }

func replCompletionStdRoot(env []string) string { return driver.StdRootFromEnv(env) }

func replCompletionModuleCache(env []string) string {
	return driver.EnvValue(env, driver.ModuleCacheEnv)
}
