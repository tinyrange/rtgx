package main

type Rtg0549Item struct {
	value int
}

func rtg0549Field(x Rtg0549Item) int {
	return x.value
}

func appMain(args []string) int {
	item := Rtg0549Item{
		value: 19,
	}
	if rtg0549Field(item) != 19 {
		print("RTG-0549 field return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
