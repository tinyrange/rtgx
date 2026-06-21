package main

func rtg0489Read(p *int) int { return *p }
func appMain(args []string) int {
	x := 14
	if rtg0489Read(&x) != 14 {
		print("RTG-0489 pointer param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
