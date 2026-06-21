package main

func appMain(args []string) int {
	bs := []byte("first")
	if len(bs) == 0 {
		print("RTG-0582 empty unexpected\n")
		return 1
	} else {
		if bs[0] != 'f' {
			print("RTG-0582 first byte failed\n")
			return 2
		}
	}
	print("PASS\n")
	return 0
}
