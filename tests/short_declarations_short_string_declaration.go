package main

func appMain(args []string) int {
	x := "short"
	if !(x == "short") {
		print("RTG-0305 short_string_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
