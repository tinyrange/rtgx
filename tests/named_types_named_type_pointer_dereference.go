package main

type rtg0674Count int

func appMain(args []string) int {
	x := rtg0674Count(6)
	p := &x
	*p = *p + rtg0674Count(5)
	if int(x) != 11 {
		print("RTG-0674 named pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
