package main

var rtg0686Text string = "go"

func appMain(args []string) int {
	if rtg0686Text[1] != 'o' {
		print("RTG-0686 string global init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
