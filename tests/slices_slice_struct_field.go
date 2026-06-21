package main

type Rtg0568Bag struct {
	items []byte
}

func appMain(args []string) int {
	bag := Rtg0568Bag{}
	bag.items = []byte("az")
	if int(bag.items[1]) != 122 {
		print("RTG-0568 slice struct field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
