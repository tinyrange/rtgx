package main

type Rtg0639Box struct{ p *int }

func rtg0639Same(p *int) *int {
	return p
}

func appMain(args []string) int {
	value := 22
	box := Rtg0639Box{p: rtg0639Same(&value)}
	if *box.p != 22 {
		print("RTG-0639 pointer helper return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
