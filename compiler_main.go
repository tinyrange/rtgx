package main

var rtgCompilerDefaultTarget int = rtgTargetLinuxAmd64
var rtgCompilerFixedTarget int
var rtgCompilerStripSymbols bool

func rtgOpenArg(path string, env []string) int {
	directFd := open(path, O_RDONLY)
	if directFd >= 0 {
		return directFd
	}
	for e := 0; e < len(env); e++ {
		pwd := env[e]
		if pwd[0] == 'P' && pwd[1] == 'W' && pwd[2] == 'D' && pwd[3] == '=' {
			var full []byte
			for i := 4; i < len(pwd); i++ {
				full = append(full, pwd[i])
			}
			full = append(full, '/')
			for i := 0; i < len(path); i++ {
				full = append(full, path[i])
			}
			fd := open(string(full), O_RDONLY)
			if fd >= 0 {
				return fd
			}
			full = append(full, 0)
			return open(string(full), O_RDONLY)
		}
	}
	return -1
}

func rtgParseTargetArg(target string) int {
	if len(target) == 11 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'm' && target[8] == 'd' && target[9] == '6' && target[10] == '4' {
		return rtgTargetLinuxAmd64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == '3' && target[7] == '8' && target[8] == '6' {
		return rtgTargetLinux386
	}
	if len(target) == 13 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'a' && target[8] == 'r' && target[9] == 'c' && target[10] == 'h' && target[11] == '6' && target[12] == '4' {
		return rtgTargetLinuxAarch64
	}
	if len(target) == 9 && target[0] == 'l' && target[1] == 'i' && target[2] == 'n' && target[3] == 'u' && target[4] == 'x' && target[5] == '/' && target[6] == 'a' && target[7] == 'r' && target[8] == 'm' {
		return rtgTargetLinuxArm
	}
	if len(target) == 13 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == 'a' && target[9] == 'm' && target[10] == 'd' && target[11] == '6' && target[12] == '4' {
		return rtgTargetWindowsAmd64
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'i' && target[2] == 'n' && target[3] == 'd' && target[4] == 'o' && target[5] == 'w' && target[6] == 's' && target[7] == '/' && target[8] == '3' && target[9] == '8' && target[10] == '6' {
		return rtgTargetWindows386
	}
	if len(target) == 11 && target[0] == 'w' && target[1] == 'a' && target[2] == 's' && target[3] == 'i' && target[4] == '/' && target[5] == 'w' && target[6] == 'a' && target[7] == 's' && target[8] == 'm' && target[9] == '3' && target[10] == '2' {
		return rtgTargetWasiWasm32
	}
	return 0
}

func rtgPrintErr(s string) {
	write(2, []byte(s), -1)
}

func rtgPrintIntErr(v int) {
	if v == 0 {
		rtgPrintErr("0")
		return
	}
	if v < 0 {
		rtgPrintErr("-")
		v = -v
	}
	var digits []byte
	for v > 0 {
		digits = append(digits, byte('0'+v%10))
		v = v / 10
	}
	for i := len(digits) - 1; i >= 0; i-- {
		write(2, digits[i:i+1], -1)
	}
}

func rtgPrintUsage() {
	rtgPrintErr("usage: rtg [-s] [-t linux/amd64|linux/386|linux/aarch64|linux/arm|windows/amd64|windows/386|wasi/wasm32] -o <output|-> <input.go|->...\n")
}

func rtgPrintUnsupportedTarget(target string) {
	rtgPrintErr("rtg: unsupported target: ")
	rtgPrintErr(target)
	rtgPrintErr("\n")
	rtgPrintErr("rtg: supported targets: linux/amd64, linux/386, linux/aarch64, linux/arm, windows/amd64, windows/386, wasi/wasm32\n")
}

func appMain(args []string, env []string) int {
	input := make([]int, 256)
	inputCount := 0
	var outputPath string
	target := rtgCompilerDefaultTarget
	if len(args) == 0 {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	i := 1
	for i != len(args) {
		arg := args[i]
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 's' {
			rtgCompilerStripSymbols = true
			i++
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 'o' {
			i++
			if i == len(args) {
				rtgPrintErr("rtg: missing argument for -o\n")
				rtgPrintUsage()
				return 1
			}
			outputArg := args[i]
			outputPath = outputArg
			i++
			continue
		}
		if len(arg) == 2 && arg[0] == '-' && arg[1] == 't' {
			i++
			if i == len(args) {
				rtgPrintErr("rtg: missing argument for -t\n")
				rtgPrintUsage()
				return 1
			}
			targetArg := args[i]
			target = rtgParseTargetArg(targetArg)
			if target == 0 {
				rtgPrintUnsupportedTarget(targetArg)
				return 1
			}
			i++
			continue
		}
		if len(arg) == 1 && arg[0] == '-' {
			if inputCount == len(input) {
				rtgPrintErr("rtg: too many input files\n")
				return 1
			}
			input[inputCount] = 0
			inputCount++
			i++
			continue
		}
		if len(arg) > 0 {
			if arg[0] == '-' {
				rtgPrintErr("rtg: unknown option: ")
				rtgPrintErr(arg)
				rtgPrintErr("\n")
				rtgPrintUsage()
				return 1
			}
		}
		fd := rtgOpenArg(arg, env)
		if fd < 0 {
			rtgPrintErr("rtg: failed to open input: ")
			rtgPrintErr(arg)
			rtgPrintErr("\n")
			return 1
		}
		if inputCount == len(input) {
			rtgPrintErr("rtg: too many input files\n")
			return 1
		}
		input[inputCount] = fd
		inputCount++
		i++
	}
	if outputPath == "" {
		rtgPrintErr("rtg: missing output path (-o)\n")
		rtgPrintUsage()
		return 1
	}
	if inputCount == 0 {
		rtgPrintErr("rtg: no input files\n")
		rtgPrintUsage()
		return 1
	}
	output := 1
	if outputPath != "-" {
		output = open(outputPath, O_RDWR|O_CREATE|O_TRUNC)
		if output < 0 {
			rtgPrintErr("rtg: failed to open output: ")
			rtgPrintErr(outputPath)
			rtgPrintErr("\n")
			return 1
		}
	}
	return compileTarget(input[:inputCount], output, target)
}
