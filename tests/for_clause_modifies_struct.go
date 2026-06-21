package main

type rtg0418Box struct{ n int }

func appMain(args []string) int {
	b := rtg0418Box{}
	for i := 0; i < 4; i = i + 1 {
		b.n = b.n + i
	}
	if b.n != 6 {
		print("RTG-0418 struct for failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
