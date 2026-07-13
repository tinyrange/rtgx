package pipeline

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/link"
	"j5.nz/rtg/rtg/internal/load"
)

const (
	PipelineOK = iota
	PipelineErrLoad
	PipelineErrBuild
	PipelineErrLink
)

type Result struct {
	Workspace    load.Workspace
	Build        build.Result
	Link         link.Result
	Ok           bool
	Error        int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
}

func BuildUnit(workDir string, stdRoot string, arg string, files []load.SourceFile) Result {
	return buildUnit(workDir, stdRoot, arg, files, 0, 0)
}

// BuildUnitWithTransientFiles allows the command driver to release source
// collection storage once lowering has copied every package into link units.
func BuildUnitWithTransientFiles(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int) Result {
	return buildUnit(workDir, stdRoot, arg, files, filesStart, filesEnd)
}

func buildUnit(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int) Result {
	result := Result{
		Ok:           true,
		Error:        PipelineOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	loadStart := arena.Mark()
	workspace := load.LoadWorkspace(workDir, stdRoot, arg, files)
	loadEnd := arena.Mark()
	result.Workspace = workspace
	if !workspace.Ok {
		return pipelineFail(result, PipelineErrLoad, -1, workspace.ErrorFile, -1)
	}
	built := build.BuildPrograms(workspace.Graph)
	result.Build = built
	if !built.Ok {
		return pipelineFail(result, PipelineErrBuild, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	// Lowered units own their text and symbol names, so the parsed workspace is
	// no longer part of the successful link handoff.
	arena.Discard(loadStart, loadEnd)
	arena.Discard(filesStart, filesEnd)
	linked := link.LinkBuildCore(built)
	result.Link = linked
	if !linked.Ok {
		return pipelineFail(result, PipelineErrLink, linked.ErrorPackage, -1, -1)
	}
	return result
}

func pipelineFail(result Result, err int, pkg int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorPackage = pkg
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
