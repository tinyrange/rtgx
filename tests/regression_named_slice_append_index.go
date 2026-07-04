package main

type rtgNamedSliceAppendItem struct {
	name string
}

type rtgNamedSliceAppendItems []rtgNamedSliceAppendItem

func rtgNamedSliceAppendCopy(out rtgNamedSliceAppendItems, values rtgNamedSliceAppendItems) rtgNamedSliceAppendItems {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func appMain() int {
	values := rtgNamedSliceAppendItems{rtgNamedSliceAppendItem{name: "PASS\n"}}
	out := rtgNamedSliceAppendCopy(rtgNamedSliceAppendItems{}, values)
	if len(out) == 1 {
		if out[0].name == "PASS\n" {
			print(out[0].name)
			return 0
		}
	}
	print("FAIL\n")
	return 1
}
