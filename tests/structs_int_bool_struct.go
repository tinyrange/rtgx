package main

type Rtg0603Flag struct {
	value int
	ok    bool
}

var rtg0603Global = Rtg0603Flag{value: 5, ok: true}

func appMain(args []string) int {
	if rtg0603Global.value != 5 || !rtg0603Global.ok {
		print("RTG-0603 int bool struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
