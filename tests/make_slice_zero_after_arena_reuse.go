package main

func rtg_runtime_ArenaMark() int { return 0 }

func rtg_runtime_ArenaReset(mark int) {}

func dynamicMakeLength() int { return 4 }

func appMain(args []string) int {
	mark := rtg_runtime_ArenaMark()
	dirty := make([]int, dynamicMakeLength())
	for i := 0; i < len(dirty); i++ {
		dirty[i] = i + 1
	}
	rtg_runtime_ArenaReset(mark)

	clean := make([]int, dynamicMakeLength())
	for i := 0; i < len(clean); i++ {
		if clean[i] != 0 {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
