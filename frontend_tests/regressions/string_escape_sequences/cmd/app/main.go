package main

func main() {
	value := "\u0214\U0001F642\141\a\b\f\v"
	if len(value) != 11 || value[0] != 0xc8 || value[1] != 0x94 || value[2] != 0xf0 || value[3] != 0x9f || value[4] != 0x99 || value[5] != 0x82 || value[6] != 'a' || value[7] != '\a' || value[8] != '\b' || value[9] != '\f' || value[10] != '\v' {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
