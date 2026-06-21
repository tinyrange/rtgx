package main

var rtg0522Seed int = 5

func rtg0522Fib(n int) int {
	if n < 2 {
		return n
	}
	return rtg0522Fib(n-1) + rtg0522Fib(n-2)
}

func appMain(args []string) int {
	if rtg0522Fib(rtg0522Seed) != 5 {
		print("RTG-0522 double recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
