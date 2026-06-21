package main

func rtg0439Done(x int) bool { return x >= 3 }
func appMain(args []string) int {
	i := 0
	for {
		if rtg0439Done(i) {
			break
		}
		i = i + 1
	}
	if i != 3 {
		print("RTG-0439 helper break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
