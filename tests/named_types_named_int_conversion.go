package main

type Rtg0659Count int

func appMain(args []string) int {
	var n Rtg0659Count = 11
	total := 0
	for i := 0; i < int(n); i = i + 1 {
		total = total + 1
	}
	if total != 11 {
		print("RTG-0659 named int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
