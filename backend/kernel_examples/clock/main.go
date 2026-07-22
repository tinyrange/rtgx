package main

// renvo:module-license GPL

func main() {
	now := Kernel_KtimeGet()
	random := Kernel_GetRandomU32()
	milliseconds := Kernel_JiffiesToMsecs(250)
	if now > 0 && random == random && milliseconds > 0 {
		print("RENVO_KERNEL_BINDINGS_PASS\n")
	}
}

func moduleExit() {
	print("RENVO_KERNEL_BINDINGS_EXIT\n")
}
