package main

func rtg0629Add(p *int, n int) {
	*p += n
}

func appMain(args []string) int {
	value := 3
	rtg0629Add(&value, 9)
	if value != 12 {
		print("RTG-0629 compound pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
