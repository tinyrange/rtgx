//go:build rtg

package driver

func RunRTGCommand(args []string, env []string) int {
	options := ParseOptions(dropProgramArg(args))
	if !options.Ok {
		printOptionError(options)
		return 1
	}
	print("rtg: RTG command backend bridge is not available yet\n")
	return 1
}

func dropProgramArg(args []string) []string {
	if len(args) == 0 {
		return args
	}
	return args[1:]
}

func printOptionError(options Options) {
	if options.Error == ParseErrMissingOutput {
		print("rtg: missing output after -o\n")
		return
	}
	if options.Error == ParseErrMissingTarget {
		print("rtg: missing target after -t\n")
		return
	}
	if options.Error == ParseErrUnsupportedTarget {
		print("rtg: unsupported target: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	if options.Error == ParseErrUnknownOption {
		print("rtg: unknown option: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	if options.Error == ParseErrMissingPackage {
		print("rtg: missing package path\n")
		return
	}
	if options.Error == ParseErrExtraPackage {
		print("rtg: extra package path: ")
		print(options.ErrorArg)
		print("\n")
		return
	}
	print("rtg: option parse failed\n")
}
