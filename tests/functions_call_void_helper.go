package main

var rtg0476Seen int

func rtg0476Mark() { rtg0476Seen = 1 }
func appMain(args []string) int {
	rtg0476Mark()
	if rtg0476Seen != 1 {
		print("RTG-0476 void helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
