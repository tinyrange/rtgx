//go:build renvo && !linux && !windows && !(darwin && arm64)

package main

func replTerminalEnable() bool { return false }

func replTerminalDisable() {}
