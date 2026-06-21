package main

type Rtg0618Inner struct{ value int }
type Rtg0618Outer struct{ inner Rtg0618Inner }

func appMain(args []string) int {
	outer := Rtg0618Outer{}
	outer.inner.value = int(byte(12))
	if outer.inner.value != 12 {
		print("RTG-0618 nested field assign failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
