package main

type Rtg0589Box struct {
	data []byte
}

func rtg0589Bytes() []byte {
	return []byte("box")
}

func appMain(args []string) int {
	b := Rtg0589Box{data: rtg0589Bytes()}
	if b.data[2] != 'x' {
		print("RTG-0589 return converted failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
