package main

// renvo:module-license GPL

// renvo:linkstatic kernel,ktime_get
func kernelKtimeGet() int64 { return 0 }

// This module intentionally has no moduleExit function. Linux accepts it as an
// init-only module and refuses rmmod because no cleanup callback exists.
func main() {
	if kernelKtimeGet() > 0 {
		print("RENVO_INIT_ONLY_PASS\n")
	}
}
