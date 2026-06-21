package main

func rtg0504Check(n int) int {
	return rtg0504Fib(n)
}

func rtg0504Fib(n int) int {
	if n < 2 {
		return n
	}
	return rtg0504Fib(n-1) + rtg0504Fib(n-2)
}

func appMain(args []string) int {
	if rtg0504Check(7) != 13 {
		print("RTG-0504 fibonacci failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
