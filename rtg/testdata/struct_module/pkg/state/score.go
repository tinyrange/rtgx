package state

func add(v int) int {
	current.value = current.value + v
	return current.value
}

func Score() int {
	current.value = base
	return add(2)
}
