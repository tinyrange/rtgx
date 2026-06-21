package main

type Rtg0613Flag struct{ ok bool }

func rtg0613All(n int) bool {
	if n == 0 {
		return true
	}
	return rtg0613All(n - 1)
}

func appMain(args []string) int {
	f := Rtg0613Flag{ok: rtg0613All(2)}
	if !f.ok {
		print("RTG-0613 field bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
