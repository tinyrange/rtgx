package main

var rtg0553Items []int

func appMain(args []string) int {
	rtg0553Items = append(rtg0553Items, 8)
	if len(rtg0553Items) != 1 || rtg0553Items[0] != 8 {
		print("RTG-0553 append one int failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
