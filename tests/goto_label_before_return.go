package main

func appMain(args []string) int {
	goto exit
	print("RTG-0458 skipped\n")
	return 1
exit:
	print("PASS\n")
	return 0
}
