package main

type Rtg0509Box struct {
	count int
}

func rtg0509Fill(b *Rtg0509Box, n int) {
	if n == 0 {
		return
	}
	b.count = b.count + n
	rtg0509Fill(b, n-1)
}

func appMain(args []string) int {
	box := Rtg0509Box{count: 0}
	for i := 0; i < 1; i = i + 1 {
		rtg0509Fill(&box, 4)
	}
	if box.count != 10 {
		print("RTG-0509 struct recursion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
