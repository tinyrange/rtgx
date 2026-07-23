package repl

func replEnvValue(env []string, key string) string {
	for i := 0; i < len(env); i++ {
		item := env[i]
		if len(item) <= len(key) || item[len(key)] != '=' {
			continue
		}
		matched := true
		for j := 0; j < len(key); j++ {
			if item[j] != key[j] {
				matched = false
				break
			}
		}
		if matched {
			return item[len(key)+1:]
		}
	}
	return ""
}
