package check

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

func invalidDefiniteAssignmentType(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl) int {
	file := pkg.Files[fileIndex].File
	for i := fn.BodyStart + 1; i+1 < fn.BodyEnd; i++ {
		if tokenTextIs(&file, i, "=") && file.Tokens[i-1].Kind == syntax.TokenIdent {
			valueKind := definiteLiteralKind(file, i+1)
			if valueKind != "" {
				name := tokenString(&file, i-1)
				for j := i - 2; j >= fn.BodyStart+1; j-- {
					if file.Tokens[j].Kind != syntax.TokenVar || j+2 >= i || file.Tokens[j+1].Kind != syntax.TokenIdent || tokenString(&file, j+1) != name {
						continue
					}
					declared := tokenString(&file, j+2)
					if definiteBuiltinType(declared) && declared != valueKind {
						return i + 1
					}
					break
				}
			}
		}
		if file.Tokens[i].Kind == syntax.TokenVar && !tokCharIs(&file, i+1, '(') {
			if invalid := invalidDefiniteInterfaceDecl(pkg, info, fileIndex, fn, i); invalid >= 0 {
				return invalid
			}
		}
	}
	return -1
}

func invalidDefiniteInterfaceDecl(pkg load.Package, info PackageInfo, fileIndex int, fn syntax.FuncDecl, at int) int {
	file := pkg.Files[fileIndex].File
	end := statementSpecEnd(file, at+1, fn.BodyEnd)
	names, namesEnd := localDeclNameTokens(file, at+1, end)
	assign := findDeclAssign(file, namesEnd, end)
	if len(names) != 1 || assign < 0 {
		return -1
	}
	typeStart, typeEnd := trimDeclSpan(file, namesEnd, assign)
	if typeEnd-typeStart != 1 || file.Tokens[typeStart].Kind != syntax.TokenIdent {
		return -1
	}
	interfaceType := LookupType(info, tokenString(&file, typeStart))
	if interfaceType < 0 || info.Types[interfaceType].Kind != TypeInterface {
		return -1
	}
	valueStart, valueEnd := trimExprSpan(file, assign+1, end)
	concreteName, pointer, ok := definiteCompositeType(file, valueStart, valueEnd)
	if !ok || LookupType(info, concreteName) < 0 || definiteTypeImplementsInterface(pkg, info, concreteName, pointer, interfaceType) {
		return -1
	}
	return valueStart
}

func definiteCompositeType(file syntax.File, start int, end int) (string, bool, bool) {
	pointer := false
	if start < end && tokCharIs(&file, start, '&') {
		pointer = true
		start++
	}
	if start+1 >= end || file.Tokens[start].Kind != syntax.TokenIdent || !tokCharIs(&file, start+1, '{') {
		return "", false, false
	}
	return tokenString(&file, start), pointer, true
}

func definiteTypeImplementsInterface(pkg load.Package, info PackageInfo, concreteName string, pointer bool, interfaceType int) bool {
	wanted := info.Types[interfaceType]
	for i := 0; i < len(wanted.InterfaceMethods); i++ {
		found := false
		for fileIndex := 0; fileIndex < len(pkg.Files) && !found; fileIndex++ {
			file := pkg.Files[fileIndex].File
			for funcIndex := 0; funcIndex < len(file.Funcs); funcIndex++ {
				fn := file.Funcs[funcIndex]
				if fn.ReceiverStart >= 0 && tokenString(&file, fn.NameTok) == wanted.InterfaceMethods[i].Name && definiteReceiverMatches(file, fn, concreteName, pointer) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func definiteReceiverMatches(file syntax.File, fn syntax.FuncDecl, concreteName string, pointer bool) bool {
	receiver := ""
	receiverPointer := false
	for i := fn.ReceiverStart; i < fn.ReceiverEnd; i++ {
		if tokCharIs(&file, i, '*') {
			receiverPointer = true
		} else if file.Tokens[i].Kind == syntax.TokenIdent {
			receiver = tokenString(&file, i)
		}
	}
	return receiver == concreteName && (pointer || !receiverPointer)
}

func excludedFileFeature(file syntax.File) (int, int) {
	for i := 0; i < len(file.Tokens); i++ {
		if file.Tokens[i].Kind == syntax.TokenSelect {
			return CheckErrSelect, i
		}
	}
	for i := 0; i < len(file.Tokens); i++ {
		kind := file.Tokens[i].Kind
		if kind == syntax.TokenGo {
			return CheckErrGoroutine, i
		}
		if kind == syntax.TokenChan || tokenTextIs(&file, i, "<-") {
			return CheckErrChannel, i
		}
	}
	return CheckOK, -1
}

func definiteLiteralKind(file syntax.File, tok int) string {
	if tok < 0 || tok >= len(file.Tokens) {
		return ""
	}
	if file.Tokens[tok].Kind == syntax.TokenString {
		return "string"
	}
	if file.Tokens[tok].Kind == syntax.TokenNumber {
		return "int"
	}
	if tokenTextIs(&file, tok, "true") || tokenTextIs(&file, tok, "false") {
		return "bool"
	}
	return ""
}

func definiteBuiltinType(name string) bool {
	return name == "int" || name == "string" || name == "bool"
}
