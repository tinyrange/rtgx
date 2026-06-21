package main

func appMain(args []string) int {
	var b byte = 'r'
	p := rtg0630Ptr(&b)
	if *p != 'r' {
		print("RTG-0630 byte address failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0630Ptr(p *byte) *byte {
	return p
}
