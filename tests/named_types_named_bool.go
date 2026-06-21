package main

type Rtg0654Flag bool

func rtg0654Make(v bool) Rtg0654Flag {
	return Rtg0654Flag(v)
}

func appMain(args []string) int {
	if !rtg0654Make(true) {
		print("RTG-0654 named bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
