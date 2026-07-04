package driver

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

const (
	SourceOK = iota
	SourceErrMissingModule
	SourceErrModule
	SourceErrPackageArg
	SourceErrReadDir
	SourceErrReadFile
	SourceErrParse
	SourceErrImport
)

type DirEntry struct {
	Name  string
	IsDir bool
}

type SourceFS interface {
	ReadDir(path string) ([]DirEntry, bool)
	ReadFile(path string) ([]byte, bool)
}

type SourceResult struct {
	Files     []load.SourceFile
	Module    load.Module
	Root      load.PackageRef
	Ok        bool
	Error     int
	ErrorPath string
}

type sourceCollector struct {
	fs      SourceFS
	module  load.Module
	stdRoot string
	files   []load.SourceFile
	loaded  []string
	loading []string
	ok      bool
	err     int
	errPath string
}

func CollectSources(workDir string, stdRoot string, arg string, fs SourceFS) SourceResult {
	result := SourceResult{Ok: true, Error: SourceOK}
	workDir = load.CleanPath(workDir)
	stdRoot = load.CleanPath(stdRoot)
	moduleRoot, moduleSrc, modulePath, ok := findModuleSource(workDir, fs)
	if !ok {
		return sourceFail(result, SourceErrMissingModule, load.JoinPath(workDir, "go.mod"))
	}
	result.Files = append(result.Files, load.SourceFile{Path: modulePath, Src: moduleSrc})
	module := load.ParseModule(moduleRoot, moduleSrc)
	result.Module = module
	if !module.Ok {
		return sourceFail(result, SourceErrModule, modulePath)
	}
	root := load.ResolvePackageArg(module, workDir, arg)
	result.Root = root
	if !root.Ok {
		return sourceFail(result, SourceErrPackageArg, arg)
	}
	collector := sourceCollector{
		fs:      fs,
		module:  module,
		stdRoot: stdRoot,
		files:   result.Files,
		ok:      true,
		err:     SourceOK,
	}
	collector.collectPackage(root)
	result.Files = collector.files
	if !collector.ok {
		return sourceFail(result, collector.err, collector.errPath)
	}
	return result
}

func findModuleSource(workDir string, fs SourceFS) (string, []byte, string, bool) {
	dir := load.CleanPath(workDir)
	for {
		path := load.JoinPath(dir, "go.mod")
		src, ok := fs.ReadFile(path)
		if ok {
			return dir, src, path, true
		}
		next := load.DirPath(dir)
		if next == dir || dir == "." || dir == "/" {
			break
		}
		dir = next
	}
	return "", nil, "", false
}

func (c *sourceCollector) collectPackage(ref load.PackageRef) {
	if !c.ok {
		return
	}
	if ref.Kind != load.PackageInModule && ref.Kind != load.PackageStandard {
		c.fail(SourceErrImport, ref.ImportPath)
		return
	}
	if findString(c.loaded, ref.ImportPath) >= 0 || findString(c.loading, ref.ImportPath) >= 0 {
		return
	}
	c.loading = append(c.loading, ref.ImportPath)
	entries, ok := c.fs.ReadDir(ref.Dir)
	if !ok {
		c.fail(SourceErrReadDir, ref.Dir)
		return
	}
	sortDirEntries(entries)
	found := false
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if entry.IsDir || !isGoSourceName(entry.Name) {
			continue
		}
		path := load.JoinPath(ref.Dir, entry.Name)
		src, ok := c.fs.ReadFile(path)
		if !ok {
			c.fail(SourceErrReadFile, path)
			return
		}
		found = true
		c.files = append(c.files, load.SourceFile{Path: path, Src: src})
		parsed := syntax.ParseFile(src)
		if !parsed.Ok {
			c.fail(SourceErrParse, path)
			return
		}
		imports := load.FileImports(c.module, c.stdRoot, parsed)
		for j := 0; j < len(imports); j++ {
			if !imports[j].Ok {
				c.fail(SourceErrImport, imports[j].ImportPath)
				return
			}
			c.collectPackage(imports[j])
			if !c.ok {
				return
			}
		}
	}
	if !found {
		c.fail(SourceErrReadDir, ref.Dir)
		return
	}
	c.loading = c.loading[:len(c.loading)-1]
	c.loaded = append(c.loaded, ref.ImportPath)
}

func (c *sourceCollector) fail(err int, path string) {
	c.ok = false
	c.err = err
	c.errPath = path
}

func sortDirEntries(entries []DirEntry) {
	for i := 1; i < len(entries); i++ {
		item := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Name > item.Name {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = item
	}
}

func isGoSourceName(name string) bool {
	return stringHasSuffix(name, ".go") && !stringHasSuffix(name, "_test.go")
}

func stringHasSuffix(text string, suffix string) bool {
	if len(suffix) > len(text) {
		return false
	}
	off := len(text) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if text[off+i] != suffix[i] {
			return false
		}
	}
	return true
}

func findString(items []string, item string) int {
	for i := 0; i < len(items); i++ {
		if items[i] == item {
			return i
		}
	}
	return -1
}

func sourceFail(result SourceResult, err int, path string) SourceResult {
	result.Ok = false
	result.Error = err
	result.ErrorPath = path
	return result
}
