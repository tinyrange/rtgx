package main

func appMain(args []string) int {
	if !((3 < 4 && 8 >= 8) || false) {
		print("RTG-0200 comparison_nested_inside_boolean_expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
