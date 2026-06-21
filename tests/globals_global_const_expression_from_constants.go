package main

const rtg0682Base = 11
const rtg0682Value = rtg0682Base*3 + 1

func appMain(args []string) int {
	if rtg0682Value != 34 {
		print("RTG-0682 const expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
