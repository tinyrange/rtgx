//go:build rtg

package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

func CheckGraphCore(graph load.Graph) Program {
	prog := Program{
		Graph:        graph,
		Ok:           true,
		Error:        CheckOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	if !graph.Ok {
		return checkFail(prog, CheckErrGraph, graph.ErrorPackage, -1, -1)
	}
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageHeader(graph, i)
		prog.Packages = append(prog.Packages, info)
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	for i := 0; i < len(graph.Packages); i++ {
		info, ok, err, file, tok := checkPackageBodyCore(graph, i, prog.Packages[i], prog.Packages)
		prog.Packages[i] = info
		if !ok {
			return checkFail(prog, err, i, file, tok)
		}
	}
	return prog
}

func checkPackageBodyCore(graph load.Graph, pkgIndex int, info PackageInfo, checked []PackageInfo) (PackageInfo, bool, int, int, int) {
	pkg := graph.Packages[pkgIndex]
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			decl := buildDeclInfo(file, fileIndex, info, checked, file.Decls[i])
			decl.Selectors = appendExprSelectors(decl.Selectors, file, fileIndex, info, checked, FuncScope{}, file.Decls[i].StartTok, file.Decls[i].EndTok)
			info.Decls = append(info.Decls, decl)
		}
	}
	sortDecls(info.Decls)
	info.DeclOrder = make([]int, len(info.Decls))
	for i := 0; i < len(info.DeclOrder); i++ {
		info.DeclOrder[i] = i
	}
	for i := 0; i < len(info.Decls); i++ {
		decl := info.Decls[i]
		if decl.Kind == SymbolType {
			file := pkg.Files[decl.File].File
			info.Types = append(info.Types, buildTypeInfo(file, decl, i))
		}
	}
	sortTypes(info.Types)
	info.TypeRefs = buildPackageTypeRefs(pkg, info, checked)
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Funcs); i++ {
			fn := file.Funcs[i]
			signature := buildFuncSignature(file, fn)
			body := syntax.ParseFuncBody(file, fn)
			if !body.Ok {
				return info, false, CheckErrBody, fileIndex, body.ErrorTok
			}
			scope, ok, scopeTok := buildFuncScope(file, fn, body)
			if !ok {
				return info, false, CheckErrScope, fileIndex, scopeTok
			}
			var out FuncBody
			out.Name = coreFuncName(file, fn)
			out.Kind = coreFuncKind(fn)
			out.File = fileIndex
			out.Func = i
			out.Signature = signature
			out.Body = body
			out.Scope = scope
			out.Refs = buildFuncRefs(file, fileIndex, info, body, scope)
			out.Selectors = buildFuncSelectors(file, fileIndex, info, checked, body, scope)
			out.Selectors = appendExprSelectors(out.Selectors, file, fileIndex, info, checked, scope, fn.StartTok, fn.EndTok)
			out.Calls = buildFuncCalls(file, fileIndex, info, checked, body, scope)
			out.Locals = buildFuncLocalDecls(file, fileIndex, info, checked, body, scope)
			out.TypeRefs = buildFuncTypeRefs(file, fileIndex, info, checked, signature, out.Locals, scope)
			info.Bodies = append(info.Bodies, out)
		}
	}
	return info, true, CheckOK, -1, -1
}

func coreFuncName(file syntax.File, fn syntax.FuncDecl) string {
	name := tokenString(file, fn.NameTok)
	if fn.ReceiverStart >= 0 {
		return receiverTypeName(file, fn) + "." + name
	}
	return name
}

func coreFuncKind(fn syntax.FuncDecl) int {
	if fn.ReceiverStart >= 0 {
		return SymbolMethod
	}
	return SymbolFunc
}
