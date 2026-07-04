package main

type rtgGlobalStructSliceItem struct {
	value int
}

type rtgGlobalStructSliceItems []rtgGlobalStructSliceItem

var rtgGlobalStructSliceValues rtgGlobalStructSliceItems = rtgGlobalStructSliceItems{rtgGlobalStructSliceItem{value: 4}, rtgGlobalStructSliceItem{6}}

func appMain(args []string) int {
	if len(rtgGlobalStructSliceValues) != 2 {
		print("global struct slice literal length failed\n")
		return 1
	}
	if rtgGlobalStructSliceValues[0].value+rtgGlobalStructSliceValues[1].value != 10 {
		print("global struct slice literal value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
