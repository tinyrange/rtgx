package main

type Rtg0648Number int

func appMain(args []string) int {
	var n Rtg0648Number = 35
	p := &n
	if *p != 35 {
		print("RTG-0648 named pointer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
