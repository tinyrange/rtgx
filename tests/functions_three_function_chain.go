package main

func rtg0483A() int { return rtg0483B() + 1 }
func rtg0483B() int { return rtg0483C() + 2 }
func rtg0483C() int { return 3 }
func appMain(args []string) int {
	if rtg0483A() != 6 {
		print("RTG-0483 chain failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
