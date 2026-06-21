package main

type rtg0339Box struct{ value int }

func appMain(args []string) int {
	b := rtg0339Box{value: 1}
	b.value = 9
	if b.value != 9 {
		print("RTG-0339 struct field assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
