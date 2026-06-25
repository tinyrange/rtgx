package fmt

func Print(s string) int {
	print(s)
	return 0
}

func Println(s string) int {
	print(s)
	print("\n")
	return 0
}

func PrintString(s string) int {
	print(s)
	return 0
}

func PrintInt(v int) int {
	print(v)
	return 0
}
