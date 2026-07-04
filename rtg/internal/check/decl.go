package check

import "j5.nz/rtg/rtg/internal/syntax"

func LookupDecl(info PackageInfo, name string) int {
	for i := 0; i < len(info.Decls); i++ {
		if info.Decls[i].Name == name {
			return i
		}
	}
	return -1
}

func buildDeclInfo(file syntax.File, fileIndex int, info PackageInfo, decl syntax.TopDecl) DeclInfo {
	name := tokenString(file, decl.NameTok)
	out := DeclInfo{
		Name:       name,
		Kind:       declSymbolKind(decl.Kind),
		File:       fileIndex,
		Token:      decl.NameTok,
		Symbol:     LookupPackageSymbol(info, name),
		TypeStart:  -1,
		TypeEnd:    -1,
		ValueStart: -1,
		ValueEnd:   -1,
	}
	if decl.Kind == syntax.TokenType {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, decl.NameTok+1, decl.EndTok)
		return out
	}
	typeStart := declNameListEnd(file, decl)
	valueStart := findDeclAssign(file, typeStart, decl.EndTok)
	if valueStart >= 0 {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, valueStart)
		out.ValueStart, out.ValueEnd = trimDeclSpan(file, valueStart+1, decl.EndTok)
	} else {
		out.TypeStart, out.TypeEnd = trimDeclSpan(file, typeStart, decl.EndTok)
	}
	return out
}

func declNameListEnd(file syntax.File, decl syntax.TopDecl) int {
	i := decl.StartTok + 1
	for i < decl.EndTok {
		if !tokCharIs(file, i, ',') {
			return i
		}
		i++
		if i >= decl.EndTok || file.Tokens[i].Kind != syntax.TokenIdent {
			return i
		}
		i++
	}
	return i
}

func findDeclAssign(file syntax.File, start int, end int) int {
	parenDepth := 0
	bracketDepth := 0
	braceDepth := 0
	for i := start; i < end; i++ {
		if tokCharIs(file, i, '(') {
			parenDepth++
		} else if tokCharIs(file, i, ')') {
			if parenDepth > 0 {
				parenDepth--
			}
		} else if tokCharIs(file, i, '[') {
			bracketDepth++
		} else if tokCharIs(file, i, ']') {
			if bracketDepth > 0 {
				bracketDepth--
			}
		} else if tokCharIs(file, i, '{') {
			braceDepth++
		} else if tokCharIs(file, i, '}') {
			if braceDepth > 0 {
				braceDepth--
			}
		} else if parenDepth == 0 && bracketDepth == 0 && braceDepth == 0 && tokenTextIs(file, i, "=") {
			return i
		}
	}
	return -1
}

func trimDeclSpan(file syntax.File, start int, end int) (int, int) {
	for start < end && isDeclSpanSeparator(file, start) {
		start++
	}
	for end > start && isDeclSpanSeparator(file, end-1) {
		end--
	}
	if start >= end {
		return -1, -1
	}
	return start, end
}

func isDeclSpanSeparator(file syntax.File, tok int) bool {
	return tokCharIs(file, tok, ';')
}

func sortDecls(decls []DeclInfo) {
	for i := 1; i < len(decls); i++ {
		item := decls[i]
		j := i - 1
		for j >= 0 && declAfter(decls[j], item) {
			decls[j+1] = decls[j]
			j--
		}
		decls[j+1] = item
	}
}

func declAfter(left DeclInfo, right DeclInfo) bool {
	if left.Name != right.Name {
		return left.Name > right.Name
	}
	if left.File != right.File {
		return left.File > right.File
	}
	return left.Token > right.Token
}
