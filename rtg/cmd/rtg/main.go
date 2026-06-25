package main

import (
	"fmt"
	"os"

	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/link"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/lower"
	"j5.nz/rtg/rtg/unit"
)

type config struct {
	target   string
	output   string
	emitUnit bool
	check    bool
	link     bool
	inputs   []string
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(1)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	if cfg.link {
		return runLink(cfg)
	}
	graph, err := load.LoadEntries(cfg.inputs, load.Options{})
	if err != nil {
		return err
	}
	if cfg.check {
		return check.Graph(graph)
	}
	if cfg.emitUnit {
		if cfg.output == "" {
			return fmt.Errorf("rtg: -emit-unit requires -o")
		}
		if len(graph.Packages) == 0 {
			return fmt.Errorf("rtg: no packages loaded")
		}
		if err := check.Graph(graph); err != nil {
			return err
		}
		u, err := lower.PackageWithGraph(graph.Packages[0], graph)
		if err != nil {
			return err
		}
		data := emit.Source(u)
		return os.WriteFile(cfg.output, data, 0644)
	}
	return fmt.Errorf("rtg: build pipeline after package loading is not implemented yet")
}

func runLink(cfg config) error {
	if cfg.output == "" {
		return fmt.Errorf("rtg: -link requires -o")
	}
	if len(cfg.inputs) == 0 {
		return fmt.Errorf("rtg: -link requires input units")
	}
	var units []unit.Unit
	for _, input := range cfg.inputs {
		data, err := os.ReadFile(input)
		if err != nil {
			return err
		}
		u, err := unit.ParseSource(input, data)
		if err != nil {
			return err
		}
		units = append(units, u)
	}
	plan, err := link.Build(units)
	if err != nil {
		return err
	}
	return os.WriteFile(cfg.output, link.Source(plan), 0644)
}

func parseArgs(args []string) (config, error) {
	cfg := config{target: "linux/amd64"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-t" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -t")
			}
			cfg.target = args[i]
			continue
		}
		if arg == "-o" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -o")
			}
			cfg.output = args[i]
			continue
		}
		if arg == "-emit-unit" {
			cfg.emitUnit = true
			continue
		}
		if arg == "-check" {
			cfg.check = true
			continue
		}
		if arg == "-link" {
			cfg.link = true
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return cfg, fmt.Errorf("rtg: unknown option: %s", arg)
		}
		cfg.inputs = append(cfg.inputs, arg)
	}
	if len(cfg.inputs) == 0 {
		cfg.inputs = append(cfg.inputs, ".")
	}
	return cfg, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: rtg [-t target] [-check] [-emit-unit -o output.rtg.go] [package-or-files...]")
}
