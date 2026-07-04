package driver

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/pipeline"
)

const (
	BuildOK = iota
	BuildErrOptions
	BuildErrPipeline
)

type BuildResult struct {
	Options      Options
	Pipeline     pipeline.Result
	Unit         []byte
	Ok           bool
	Error        int
	ErrorArg     string
	ErrorAt      int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
}

func BuildUnit(args []string, workDir string, stdRoot string, files []load.SourceFile) BuildResult {
	result := BuildResult{
		Ok:           true,
		Error:        BuildOK,
		ErrorAt:      -1,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	options := ParseOptions(args)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, options.ErrorAt, -1, -1, -1)
	}
	built := pipeline.BuildUnit(workDir, stdRoot, options.Package, files)
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	return result
}

func buildFail(result BuildResult, err int, arg string, at int, pkg int, file int, tok int) BuildResult {
	result.Ok = false
	result.Error = err
	result.ErrorArg = arg
	result.ErrorAt = at
	result.ErrorPackage = pkg
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
