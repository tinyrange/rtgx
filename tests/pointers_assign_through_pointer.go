package main

var rtg0628Global int

func appMain(args []string) int {
	p := &rtg0628Global
	*p = 16
	if rtg0628Global != 16 {
		print("RTG-0628 assign through pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
