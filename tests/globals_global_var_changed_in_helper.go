package main

var rtg0694Value int = 2

func rtg0694Add() {
	rtg0694Value = rtg0694Value + 5
}

func appMain(args []string) int {
	rtg0694Add()
	if rtg0694Value != 7 {
		print("RTG-0694 helper global mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
