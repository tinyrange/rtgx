package main

type Rtg0653Letter byte

var rtg0653Global Rtg0653Letter = 'z'

func appMain(args []string) int {
	if rtg0653Global != 'z' {
		print("RTG-0653 named byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
