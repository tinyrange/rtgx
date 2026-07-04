package main

import (
	"flag"
	"fmt"
	"os"

	"j5.nz/rtg/rtgunit"
)

func main() {
	output := flag.String("o", "-", "output unit path")
	dump := flag.Bool("dump", false, "decode and print canonical source instead of encoding")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: rtgunitc [-o output.rtgu] input.go...\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *dump {
		if flag.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "rtgunitc: -dump expects exactly one unit input")
			os.Exit(1)
		}
		program, err := rtgunit.ReadFile(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "rtgunitc: failed to read unit: %v\n", err)
			os.Exit(1)
		}
		if _, err := os.Stdout.Write(rtgunit.Source(program)); err != nil {
			fmt.Fprintf(os.Stderr, "rtgunitc: failed to write source: %v\n", err)
			os.Exit(1)
		}
		return
	}

	program, err := rtgunit.ConvertFiles(flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "rtgunitc: conversion failed: %v\n", err)
		os.Exit(1)
	}
	if err := rtgunit.WriteFile(*output, program); err != nil {
		fmt.Fprintf(os.Stderr, "rtgunitc: failed to write unit: %v\n", err)
		os.Exit(1)
	}
}
