package main

type Rtg0605Name struct{ name string }

func appMain(args []string) int {
	item := rtg0605Make("elm")
	if item.name != "elm" {
		print("RTG-0605 string field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func rtg0605Make(s string) Rtg0605Name {
	return Rtg0605Name{name: s}
}
