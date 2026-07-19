package main

var recovered bool
var calls int

func argument(value string, want int) string {
	if calls != want {
		return "FAIL"
	}
	calls++
	return value
}

func panicPath() {
	defer func() {
		recovered = recover() != nil
	}()
	defer print(argument("P", 2))
	panic("boom")
}

func builtinPath() (ok bool) {
	values := map[string]int{"value": 1}
	destination := []int{0}
	source := []int{7}
	defer func() {
		ok = recover() != nil && len(values) == 0 && destination[0] == 7
	}()
	defer copy(destination, source)
	defer delete(values, "value")
	defer panic("expected")
	return false
}

func main() {
	defer println()
	defer print(argument("S", 0))
	defer print(argument("S", 1))
	letter := "A"
	defer print(letter)
	letter = "X"
	panicPath()
	if !recovered || !builtinPath() || calls != 3 || letter != "X" {
		print("FAIL")
	}
}
