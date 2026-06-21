package main

const rtg0679Text = "ok"

func appMain(args []string) int {
	if len(rtg0679Text) != 2 {
		print("RTG-0679 string const failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
