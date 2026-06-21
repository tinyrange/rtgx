package main

type Rtg0606Wide struct{ n int64 }

func appMain(args []string) int {
	w := Rtg0606Wide{n: int64(44)}
	if w.n == 44 {
		print("PASS\n")
		return 0
	}
	print("RTG-0606 int64 field failed\n")
	return 1
}
