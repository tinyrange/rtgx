//go:build renvo && !browser

package driver

import "unsafe"

func renvoRunCStringVector(first string, values []string) ([][]byte, []int) {
	count := len(values)
	if first != "" {
		count++
	}
	storage := make([][]byte, 0, count)
	pointers := make([]int, 0, count+1)
	if first != "" {
		value := append([]byte(first), 0)
		storage = append(storage, value)
		pointers = append(pointers, renvoRunBytePointer(value))
	}
	for i := 0; i < len(values); i++ {
		value := append([]byte(values[i]), 0)
		storage = append(storage, value)
		pointers = append(pointers, renvoRunBytePointer(value))
	}
	pointers = append(pointers, 0)
	return storage, pointers
}

func renvoRunBytePointer(value []byte) int {
	if len(value) == 0 {
		return 0
	}
	return int(unsafe.Pointer(&value[0]))
}

func renvoRunIntPointer(value []int) int {
	if len(value) == 0 {
		return 0
	}
	return int(unsafe.Pointer(&value[0]))
}
