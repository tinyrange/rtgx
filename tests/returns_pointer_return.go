package main

func rtg0535Ptr(p *int) *int {
	for {
		return p
	}
}

func appMain(args []string) int {
	value := 11
	p := rtg0535Ptr(&value)
	if *p != 11 {
		print("RTG-0535 pointer return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
