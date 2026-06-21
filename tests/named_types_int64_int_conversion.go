package main

type Rtg0664Box struct{ value int }

func appMain(args []string) int {
	wide := int64(14)
	box := Rtg0664Box{value: int(wide)}
	if box.value != 14 {
		print("RTG-0664 int64 int conversion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
