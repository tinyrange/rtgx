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
	prog.Packages = make([]PackageInfo, 0, len(graph.Packages))
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
	info.Decls = make([]DeclInfo, 0, countPackageDeclsCore(pkg))
	info.Bodies = make([]FuncBody, 0, countPackageFuncsCore(pkg))
	for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
		file := pkg.Files[fileIndex].File
		for i := 0; i < len(file.Decls); i++ {
			info.Decls = append(info.Decls, buildDeclInfoCore(file, fileIndex, info, checked, file.Decls[i]))
		}
	}
	sortDecls(info.Decls)
	info.DeclOrder = make([]int, len(info.Decls))
	for i := 0; i < len(info.DeclOrder); i++ {
		info.DeclOrder[i] = i
	}
	info.Types = make([]TypeInfo, 0, countTypeDeclsCore(info.Decls))
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
			scope, ok, scopeTok := buildFuncScopeCore(file, fn)
			if !ok {
				return info, false, CheckErrScope, fileIndex, scopeTok
			}
			bodyStart := fn.BodyStart + 1
			bodyEnd := fn.BodyEnd - 1
			var out FuncBody
			out.Name = coreFuncName(file, fn)
			out.Kind = coreFuncKind(fn)
			out.File = fileIndex
			out.Func = i
			out.Signature = signature
			out.Scope = scope
			bodyTokens := bodyEnd - bodyStart
			out.Refs = make([]NameRef, 0, coreRefCapacity(bodyTokens))
			out.Selectors = make([]SelectorRef, 0, coreSelectorCapacity(bodyTokens))
			out.Refs = appendExprRefs(out.Refs, file, fileIndex, info, scope, bodyStart, bodyEnd)
			out.Selectors = appendExprSelectors(out.Selectors, file, fileIndex, info, checked, scope, bodyStart, bodyEnd)
			out.Locals = buildFuncLocalTypeSpansCore(file, fn)
			out.TypeRefs = buildFuncTypeRefs(file, fileIndex, info, checked, signature, out.Locals, scope)
			info.Bodies = append(info.Bodies, out)
		}
	}
	return info, true, CheckOK, -1, -1
}

func countPackageDeclsCore(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		count += len(pkg.Files[i].File.Decls)
	}
	return count
}

func countPackageFuncsCore(pkg load.Package) int {
	count := 0
	for i := 0; i < len(pkg.Files); i++ {
		count += len(pkg.Files[i].File.Funcs)
	}
	return count
}

func countTypeDeclsCore(decls []DeclInfo) int {
	count := 0
	for i := 0; i < len(decls); i++ {
		if decls[i].Kind == SymbolType {
			count++
		}
	}
	return count
}

func buildDeclInfoCore(file syntax.File, fileIndex int, info PackageInfo, checked []PackageInfo, decl syntax.TopDecl) DeclInfo {
	name := tokenString(file, decl.NameTok)
	out := DeclInfo{
		Name:       name,
		Kind:       declSymbolKind(decl.Kind),
		File:       fileIndex,
		Token:      decl.NameTok,
		Symbol:     LookupPackageSymbol(info, name),
		ValueIndex: declNameIndex(file, decl),
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: -1,
		ValueEnd:   -1,
	}
	if decl.Kind == syntax.TokenType {
		typeStart := decl.NameTok + 1
		if tokenTextIs(file, typeStart, "=") {
			out.Alias = true
			typeStart++
		}
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
		return out
	}
	typeStart := declNameListEnd(file, decl)
	valueStart := findDeclAssign(file, typeStart, decl.EndTok)
	if valueStart >= 0 {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, valueStart)
		out.ValueStart, out.ValueEnd = trimDeclSpan(file, valueStart+1, decl.EndTok)
		valueTokens := out.ValueEnd - out.ValueStart
		out.Refs = make([]NameRef, 0, coreRefCapacity(valueTokens))
		out.Selectors = make([]SelectorRef, 0, coreSelectorCapacity(valueTokens))
		out.Refs = appendExprRefs(out.Refs, file, fileIndex, info, FuncScope{}, out.ValueStart, out.ValueEnd)
		out.Selectors = appendExprSelectors(out.Selectors, file, fileIndex, info, checked, FuncScope{}, out.ValueStart, out.ValueEnd)
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

func buildFuncScopeCore(file syntax.File, fn syntax.FuncDecl) (FuncScope, bool, int) {
	var scope FuncScope
	scope.Names = make([]ScopeName, 0, coreScopeCapacity(fn.BodyEnd-fn.BodyStart))
	if fn.ReceiverStart >= 0 {
		tok := receiverNameToken(file, fn)
		if tok >= 0 {
			if !addScopeName(&scope, tokenString(file, tok), NameReceiver, tok, true, false) {
				return scope, false, tok
			}
		}
	}
	if fn.ParamsStart >= 0 && fn.ParamsEnd > fn.ParamsStart {
		ok, tok := collectFieldNames(file, fn.ParamsStart+1, fn.ParamsEnd-1, NameParam, &scope)
		if !ok {
			return scope, false, tok
		}
	}
	if fn.ResultStart >= 0 && fn.ResultEnd > fn.ResultStart && tokCharIs(file, fn.ResultStart, '(') {
		end := fn.ResultEnd - 1
		if tokCharIs(file, end, ')') {
			ok, tok := collectFieldNames(file, fn.ResultStart+1, end, NameResult, &scope)
			if !ok {
				return scope, false, tok
			}
		}
	}
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	for i := start; i < end; i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenConst || kind == syntax.TokenVar || kind == syntax.TokenType {
			i = collectCoreDeclScope(file, i, end, &scope)
			continue
		}
		if tokenTextIs(file, i, ":=") {
			collectCoreShortDeclScope(file, coreLHSStart(file, i, start), i, &scope)
			continue
		}
		if coreTokenLooksLikeLabel(file, i, start, end) {
			if !addScopeName(&scope, tokenString(file, i), NameLabel, i, true, true) {
				return scope, false, i
			}
		}
	}
	return scope, true, -1
}

func coreTokenLooksLikeLabel(file syntax.File, tok int, start int, end int) bool {
	if tok < start || tok+1 >= end || file.Tokens[tok].Kind != syntax.TokenIdent || !tokCharIs(file, tok+1, ':') || tokenTextIs(file, tok+1, ":=") {
		return false
	}
	if tok == start {
		return true
	}
	prev := tok - 1
	if tokCharIs(file, prev, '{') || tokCharIs(file, prev, ',') {
		return false
	}
	return file.Tokens[prev].Line != file.Tokens[tok].Line || tokCharIs(file, prev, ';') || tokCharIs(file, prev, '}')
}

func collectCoreDeclScope(file syntax.File, start int, end int, scope *FuncScope) int {
	specStart := start + 1
	if specStart < end && tokCharIs(file, specStart, '(') {
		closeTok := findTypeMatching(file, specStart, '(', ')')
		if closeTok <= specStart || closeTok > end {
			return start
		}
		i := specStart + 1
		for i < closeTok-1 {
			i = skipLocalSeparators(file, i, closeTok-1)
			if i >= closeTok-1 {
				break
			}
			specEnd := statementSpecEnd(file, i, closeTok-1)
			collectLeadingIdentList(file, i, specEnd, scope)
			if specEnd <= i {
				i++
			} else {
				i = specEnd
			}
		}
		return closeTok
	}
	specEnd := statementSpecEnd(file, specStart, end)
	collectLeadingIdentList(file, specStart, specEnd, scope)
	return specEnd
}

func collectCoreShortDeclScope(file syntax.File, start int, end int, scope *FuncScope) {
	for i := start; i < end; i++ {
		if file.Tokens[i].Kind == syntax.TokenIdent && tokenString(file, i) != "_" {
			if LookupScopeName(*scope, tokenString(file, i)) < 0 {
				addScopeName(scope, tokenString(file, i), NameLocal, i, false, false)
			}
		}
	}
}

func coreLHSStart(file syntax.File, assign int, limit int) int {
	start := assign - 1
	for start > limit {
		if tokCharIs(file, start, ';') || tokCharIs(file, start, '{') || tokCharIs(file, start, '}') {
			return start + 1
		}
		if file.Tokens[start].Line != file.Tokens[assign].Line {
			return start + 1
		}
		start--
	}
	return start
}

func buildFuncLocalTypeSpansCore(file syntax.File, fn syntax.FuncDecl) []LocalDeclInfo {
	decls := make([]LocalDeclInfo, 0, coreLocalTypeCapacity(fn.BodyEnd-fn.BodyStart))
	start := fn.BodyStart + 1
	end := fn.BodyEnd - 1
	for i := start; i < end; i++ {
		kind := file.Tokens[i].Kind
		if kind != syntax.TokenConst && kind != syntax.TokenVar && kind != syntax.TokenType {
			continue
		}
		specStart := i + 1
		if specStart < end && tokCharIs(file, specStart, '(') {
			closeTok := findTypeMatching(file, specStart, '(', ')')
			if closeTok <= specStart || closeTok > end {
				continue
			}
			j := specStart + 1
			for j < closeTok-1 {
				j = skipLocalSeparators(file, j, closeTok-1)
				if j >= closeTok-1 {
					break
				}
				specEnd := statementSpecEnd(file, j, closeTok-1)
				decls = appendLocalTypeSpanCore(decls, file, declSymbolKind(kind), j, specEnd)
				if specEnd <= j {
					j++
				} else {
					j = specEnd
				}
			}
			i = closeTok
			continue
		}
		specEnd := statementSpecEnd(file, specStart, end)
		decls = appendLocalTypeSpanCore(decls, file, declSymbolKind(kind), specStart, specEnd)
		i = specEnd
	}
	return decls
}

func coreRefCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 4
	if capacity < 4 {
		return 4
	}
	return capacity
}

func coreSelectorCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 16
	if capacity < 2 {
		return 2
	}
	return capacity
}

func coreScopeCapacity(tokens int) int {
	if tokens <= 0 {
		return 4
	}
	capacity := tokens / 32
	if capacity < 4 {
		return 4
	}
	return capacity
}

func coreLocalTypeCapacity(tokens int) int {
	if tokens <= 0 {
		return 0
	}
	capacity := tokens / 64
	if capacity < 2 {
		return 2
	}
	return capacity
}

func appendLocalTypeSpanCore(decls []LocalDeclInfo, file syntax.File, kind int, start int, end int) []LocalDeclInfo {
	start, end = trimDeclSpan(file, start, end)
	if start < 0 || end <= start || start >= len(file.Tokens) || file.Tokens[start].Kind != syntax.TokenIdent {
		return decls
	}
	typeStart := -1
	typeEnd := -1
	if kind == SymbolType {
		typeStart = start + 1
		if tokenTextIs(file, typeStart, "=") {
			typeStart++
		}
		typeStart, typeEnd = trimDeclSpan(file, typeStart, end)
	} else {
		_, namesEnd := localDeclNameTokens(file, start, end)
		valueStart := findDeclAssign(file, namesEnd, end)
		if valueStart >= 0 {
			typeStart, typeEnd = trimDeclSpan(file, namesEnd, valueStart)
		} else {
			typeStart, typeEnd = trimDeclSpan(file, namesEnd, end)
		}
	}
	if typeStart < 0 || typeEnd <= typeStart {
		return decls
	}
	return append(decls, LocalDeclInfo{TypeStart: typeStart, TypeEnd: typeEnd})
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
