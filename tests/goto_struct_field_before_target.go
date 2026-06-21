package main

type rtg0466Box struct{ n int }

func appMain(args []string) int {
	b := rtg0466Box{}
	b.n = 9
	goto check
check:
	if b.n != 9 {
		print("RTG-0466 struct goto failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
