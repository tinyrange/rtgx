package main

const rtg0596Want byte = 'd'

func appMain(args []string) int {
	bs := []byte("code")
	i := 0
scan:
	if i >= len(bs) {
		print("RTG-0596 goto scanner missing\n")
		return 1
	}
	if bs[i] == rtg0596Want {
		print("PASS\n")
		return 0
	}
	i = i + 1
	goto scan
}
