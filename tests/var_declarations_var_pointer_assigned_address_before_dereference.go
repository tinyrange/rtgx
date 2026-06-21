package main

func appMain(args []string) int {
	x := 8
	var p *int = &x
	if !(*p == 8) {
		print("RTG-0287 var_pointer_assigned_address_before_dereference failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
