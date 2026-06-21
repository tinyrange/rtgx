package main

func rtg0485Word() string { return "ok" }
func appMain(args []string) int {
	if rtg0485Word() != "ok" {
		print("RTG-0485 string return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
