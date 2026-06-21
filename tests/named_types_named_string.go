package main

type Rtg0655Name string

func appMain(args []string) int {
	name := rtg0655Make("oak")
	if name != "oak" {
		print("RTG-0655 named string failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0655Make(s string) Rtg0655Name {
	return Rtg0655Name(s)
}
