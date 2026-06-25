package lower

import (
	"fmt"
	"sort"

	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/unit"
)

func Package(pkg load.Package) (unit.Unit, error) {
	return PackageWithGraph(pkg, nil)
}

func PackageWithGraph(pkg load.Package, graph *load.Graph) (unit.Unit, error) {
	u := unit.Unit{ImportPath: pkg.ImportPath, Package: pkg.Name}
	u.Imports = append(u.Imports, pkg.Imports...)
	files := append([]load.File(nil), pkg.Files...)
	sort.Slice(files, func(i int, j int) bool {
		return files[i].Path < files[j].Path
	})
	parsedFiles := make([]parse.File, 0, len(files))
	topNames := map[string]string{}
	for _, file := range files {
		parsed, err := parse.FileSource(file.Path, file.Source)
		if err != nil {
			return unit.Unit{}, err
		}
		if parsed.PackageName != pkg.Name {
			return unit.Unit{}, fmt.Errorf("%s: package name %s does not match loaded package %s", file.Path, parsed.PackageName, pkg.Name)
		}
		parsedFiles = append(parsedFiles, parsed)
		for _, decl := range parsed.Decls {
			if decl.Name != "" && decl.Name != "_" {
				topNames[decl.Name] = SymbolName(pkg.ImportPath, decl.Name)
			}
		}
	}
	for name, unitName := range topNames {
		if isExported(name) {
			u.Exports = append(u.Exports, unit.Symbol{ImportPath: pkg.ImportPath, Name: name, UnitName: unitName})
		}
	}
	sort.Slice(u.Exports, func(i int, j int) bool {
		return u.Exports[i].Name < u.Exports[j].Name
	})
	importRefs := importReferenceMap(pkg, graph)
	seenRefs := map[string]bool{}
	for _, parsed := range parsedFiles {
		for _, decl := range parsed.Decls {
			var refs []unit.Symbol
			body := rewriteDecl(parsed, decl, topNames, importRefs, &refs)
			for _, ref := range refs {
				key := ref.ImportPath + "\x00" + ref.Name
				if !seenRefs[key] {
					seenRefs[key] = true
					u.References = append(u.References, ref)
				}
			}
			u.Decls = append(u.Decls, unit.Decl{
				Path:     unitPathForDecl(files, parsed.Path),
				Kind:     decl.Kind,
				Name:     decl.Name,
				UnitName: topNames[decl.Name],
				Body:     body,
			})
		}
	}
	sort.Slice(u.References, func(i int, j int) bool {
		if u.References[i].ImportPath == u.References[j].ImportPath {
			return u.References[i].Name < u.References[j].Name
		}
		return u.References[i].ImportPath < u.References[j].ImportPath
	})
	return u, nil
}

func unitPathForDecl(files []load.File, path string) string {
	for _, file := range files {
		if file.Path == path {
			if file.UnitPath != "" {
				return file.UnitPath
			}
			return file.Path
		}
	}
	return path
}

func SymbolName(importPath string, name string) string {
	out := []byte("rtg_")
	for i := 0; i < len(importPath); i++ {
		c := importPath[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	out = append(out, '_')
	out = append(out, name...)
	return string(out)
}

func rewriteDecl(file parse.File, decl parse.Decl, topNames map[string]string, importRefs map[string]map[string]unit.Symbol, refs *[]unit.Symbol) string {
	start := decl.Start
	end := decl.End
	if start < 0 {
		start = 0
	}
	if end > len(file.Source) {
		end = len(file.Source)
	}
	for end > start && (file.Source[end-1] == ' ' || file.Source[end-1] == '\t' || file.Source[end-1] == '\r' || file.Source[end-1] == '\n') {
		end--
	}
	var out []byte
	cursor := start
	prevText := ""
	for i := 0; i < len(file.Tokens); i++ {
		tok := file.Tokens[i]
		if tok.End <= start {
			prevText = tok.Text
			continue
		}
		if tok.Start >= end {
			break
		}
		if tok.Start > cursor {
			out = append(out, file.Source[cursor:tok.Start]...)
		}
		replacement := ""
		if tok.Kind == scan.Ident && i+2 < len(file.Tokens) && file.Tokens[i+1].Text == "." && file.Tokens[i+2].Kind == scan.Ident {
			if symbols, ok := importRefs[tok.Text]; ok {
				member := file.Tokens[i+2]
				if sym, ok := symbols[member.Text]; ok {
					replacement = sym.UnitName
					*refs = append(*refs, sym)
					out = append(out, replacement...)
					cursor = member.End
					prevText = member.Text
					i += 2
					continue
				}
			}
		}
		if tok.Kind == scan.Ident && prevText != "." {
			replacement = topNames[tok.Text]
		}
		if replacement != "" {
			out = append(out, replacement...)
		} else {
			out = append(out, file.Source[tok.Start:tok.End]...)
		}
		cursor = tok.End
		prevText = tok.Text
	}
	if cursor < end {
		out = append(out, file.Source[cursor:end]...)
	}
	return string(out)
}

func importReferenceMap(pkg load.Package, graph *load.Graph) map[string]map[string]unit.Symbol {
	refs := map[string]map[string]unit.Symbol{}
	if graph == nil {
		return refs
	}
	packages := map[string]load.Package{}
	for _, dep := range graph.Packages {
		packages[dep.ImportPath] = dep
	}
	for importPath, localName := range pkg.ImportNames {
		dep, ok := packages[importPath]
		if !ok || localName == "" {
			continue
		}
		symbols := map[string]unit.Symbol{}
		for _, file := range dep.Files {
			parsed, err := parse.FileSource(file.Path, file.Source)
			if err != nil {
				continue
			}
			for _, decl := range parsed.Decls {
				if isExported(decl.Name) {
					symbols[decl.Name] = unit.Symbol{ImportPath: importPath, Name: decl.Name, UnitName: SymbolName(importPath, decl.Name)}
				}
			}
		}
		refs[localName] = symbols
	}
	return refs
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
