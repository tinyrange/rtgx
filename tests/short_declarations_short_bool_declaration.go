package main

func appMain(args []string) int {
	x := true
	if !(x) {
		print("RTG-0304 short_bool_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
