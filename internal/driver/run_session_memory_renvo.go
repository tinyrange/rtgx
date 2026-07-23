//go:build renvo && (linux || windows || darwin)

package driver

import "unsafe"

func renvoRunStringWords(values []string) []int {
	stride := renvoRunStringWordStride()
	words := make([]int, len(values)*stride)
	for i := 0; i < len(values); i++ {
		value := []byte(values[i])
		words[i*stride] = renvoRunBytePointer(value)
		words[i*stride+stride/2] = len(value)
	}
	return words
}

func renvoRunStoreByte(address int, value byte) {
	pointer := (*byte)(unsafe.Pointer(address))
	*pointer = value
}

func renvoRunLoadByte(address int) byte {
	pointer := (*byte)(unsafe.Pointer(address))
	return *pointer
}
