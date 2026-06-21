package main

const rtg0700A = 10

var rtg0700B int = 20
var rtg0700C byte = 3

func appMain(args []string) int {
	if rtg0700A+rtg0700B+int(rtg0700C) != 33 {
		print("RTG-0700 global checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
