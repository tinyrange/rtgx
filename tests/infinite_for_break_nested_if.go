package main

func appMain(args []string) int {
	i := 0
	for {
		i = i + 1
		if i > 2 {
			if true {
				break
			}
		}
	}
	if i != 3 {
		print("RTG-0445 nested break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
