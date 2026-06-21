package main

type rtg0391Counter struct{ n int }

func appMain(args []string) int {
	c := rtg0391Counter{}
	for c.n < 3 {
		c.n = c.n + 1
	}
	if c.n != 3 {
		print("RTG-0391 struct loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
