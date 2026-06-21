package main

func rtg0481Earlier() int { return 6 }
func appMain(args []string) int {
	if rtg0481Earlier() != 6 {
		print("RTG-0481 earlier helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
