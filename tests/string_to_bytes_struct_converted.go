package main

type Rtg0591Blob struct {
	data []byte
}

func appMain(args []string) int {
	blob := Rtg0591Blob{data: []byte("hi")}
	if blob.data[1] != 'i' {
		print("RTG-0591 struct converted failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
