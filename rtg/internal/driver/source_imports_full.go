//go:build !rtg

package driver

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

func collectSourceImports(module load.Module, stdRoot string, src []byte) ([]load.PackageRef, bool) {
	parsed := syntax.ParseFile(src)
	if !parsed.Ok {
		return nil, false
	}
	return load.FileImports(module, stdRoot, parsed), true
}
