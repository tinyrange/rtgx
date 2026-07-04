package main

type rtgWasmReturnStrideRecord struct {
	value int
}

func rtgWasmReturnStrideBuild(seed int) []rtgWasmReturnStrideRecord {
	var out []rtgWasmReturnStrideRecord
	out = append(out, rtgWasmReturnStrideRecord{value: seed})
	out = append(out, rtgWasmReturnStrideRecord{value: seed + 1})
	return out
}

func appMain(args []string, env []string) int {
	values := rtgWasmReturnStrideBuild(10)
	if values[0].value == 10 && values[1].value == 11 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
