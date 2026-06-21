package main

func rtg0650After(p *int, n int) {
	if n == 0 {
		*p = *p + 10
		return
	}
	rtg0650After(p, n-1)
	*p = *p + 1
}

func appMain(args []string) int {
	value := 0
	rtg0650After(&value, 3)
	if value != 13 {
		print("RTG-0650 recursive visible mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
