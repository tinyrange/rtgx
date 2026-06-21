package main

func rtg0811Add(
	a int,
	b int,
) int {
	return a + b
}

func appMain(args []string) int {
	if rtg0811Add(3, 4) != 7 {
		print("RTG-0811 multiline params failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
