package main

func rtg0500A() int      { return 7 }
func rtg0500B(x int) int { return x + 18 }
func appMain(args []string) int {
	if rtg0500B(rtg0500A()) != 25 {
		print("RTG-0500 direct result failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
