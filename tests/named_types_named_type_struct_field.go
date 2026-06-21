package main

type rtg0672Amount int
type rtg0672Box struct {
	value rtg0672Amount
}

func appMain(args []string) int {
	b := rtg0672Box{value: rtg0672Amount(9)}
	if int(b.value) != 9 {
		print("RTG-0672 named field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
