//go:build renvo

package repl

import "renvo.dev/internal/driver"

func replCompletionFS() driver.SourceFS { return driver.RenvoFS{} }

func replCompletionTarget() string { return replTarget() }

func replCompletionStdRoot(env []string) string {
	if root := replEnvValue(env, "RENVO_STDROOT"); root != "" {
		return root
	}
	return "/std"
}

func replCompletionModuleCache(env []string) string {
	return replEnvValue(env, "RENVO_MODCACHE")
}
