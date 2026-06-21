package main

type Rtg0614Outer struct{ inner int }

func appMain(args []string) int {
	out := Rtg0614Outer{}
	if len(args) >= 0 {
		out.inner = 21
	}
	if out.inner != 21 {
		print("RTG-0614 if field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
