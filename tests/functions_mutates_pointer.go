package main

func rtg0492Set(p *int) { *p = 17 }
func appMain(args []string) int {
	x := 0
	rtg0492Set(&x)
	if x != 17 {
		print("RTG-0492 pointer mutate failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
