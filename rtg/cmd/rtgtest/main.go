//go:build !rtg

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"j5.nz/rtg/rtg/internal/testfront"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	var output string
	flags := flag.NewFlagSet("rtgtest", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	flags.StringVar(&output, "o", "", "generated package output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if output == "" {
		fmt.Fprintln(os.Stderr, "rtgtest: missing -o output directory")
		return 2
	}
	if flags.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "rtgtest: expected exactly one package directory")
		return 2
	}
	result, err := testfront.GeneratePackage(flags.Arg(0))
	if err != nil {
		printGenerateError(err)
		return 1
	}
	if err := testfront.WritePackage(output, result); err != nil {
		fmt.Fprintf(os.Stderr, "rtgtest: failed to write generated package: %v\n", err)
		return 1
	}
	return 0
}

func printGenerateError(err error) {
	if errors.Is(err, testfront.ErrNoTests) {
		fmt.Fprintln(os.Stderr, "rtgtest: no tests found")
		return
	}
	if errors.Is(err, testfront.ErrExternalTests) {
		fmt.Fprintln(os.Stderr, "rtgtest: external test packages are not supported yet")
		return
	}
	fmt.Fprintf(os.Stderr, "rtgtest: failed to generate test runner: %v\n", err)
}
