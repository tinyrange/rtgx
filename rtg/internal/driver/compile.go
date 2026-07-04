package driver

import "j5.nz/rtg/rtg/internal/load"

const (
	CompileOK = iota
	CompileErrBuild
	CompileErrBackend
)

type Backend interface {
	CompileUnit(unit []byte, target string, strip bool) ([]byte, bool)
}

type CompileResult struct {
	Build  BuildResult
	Binary []byte
	Ok     bool
	Error  int
}

func CompileUnit(args []string, workDir string, stdRoot string, files []load.SourceFile, backend Backend) CompileResult {
	result := CompileResult{Ok: true, Error: CompileOK}
	built := BuildUnit(args, workDir, stdRoot, files)
	result.Build = built
	if !built.Ok {
		return compileFail(result, CompileErrBuild)
	}
	return compileBuiltUnit(result, built, backend)
}

func CompileFromFS(args []string, workDir string, stdRoot string, fs SourceFS, backend Backend) CompileResult {
	result := CompileResult{Ok: true, Error: CompileOK}
	built := BuildFromFS(args, workDir, stdRoot, fs)
	result.Build = built
	if !built.Ok {
		return compileFail(result, CompileErrBuild)
	}
	return compileBuiltUnit(result, built, backend)
}

func compileBuiltUnit(result CompileResult, built BuildResult, backend Backend) CompileResult {
	if backend == nil {
		return compileFail(result, CompileErrBackend)
	}
	binary, ok := backend.CompileUnit(built.Unit, built.Options.Target, built.Options.Strip)
	if !ok || len(binary) == 0 {
		return compileFail(result, CompileErrBackend)
	}
	result.Binary = binary
	return result
}

func compileFail(result CompileResult, err int) CompileResult {
	result.Ok = false
	result.Error = err
	return result
}
