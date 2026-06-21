package main

type rtg0830Counter struct {
	value int
}

func rtg0830Add(c *rtg0830Counter, amount int) {
	c.value = c.value + amount
}

func appMain(args []string) int {
	c := rtg0830Counter{value: 1}
	rtg0830Add(&c, 6)
	if c.value != 7 {
		print("RTG-0830 function not method failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
