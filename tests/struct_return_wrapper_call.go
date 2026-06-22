package main

type rtgStructReturnWrapperCallBox struct {
	value int
	ok    bool
}

func rtgStructReturnWrapperCallInner() rtgStructReturnWrapperCallBox {
	return rtgStructReturnWrapperCallBox{value: 11, ok: true}
}

func rtgStructReturnWrapperCallOuter() rtgStructReturnWrapperCallBox {
	return rtgStructReturnWrapperCallInner()
}

func appMain(args []string) int {
	box := rtgStructReturnWrapperCallOuter()
	if !box.ok || box.value != 11 {
		print("struct return wrapper call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
