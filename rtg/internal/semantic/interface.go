package semantic

import (
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/syntax"
)

type interfaceMethod struct {
	name      string
	signature string
}

type interfaceType struct {
	pkg       int
	file      int
	name      string
	start     int
	end       int
	methods   []interfaceMethod
	candidate string
}

type concreteMethod struct {
	pkg       int
	receiver  string
	pointer   bool
	name      string
	signature string
}

type concreteType struct {
	pkg  int
	name string
}

type sourceEdit struct {
	start int
	end   int
	text  string
}

// LowerInterfaces performs closed-world devirtualization only when an
// interface has exactly one concrete implementation in its package. The
// result is ordinary Go source, so normal checking still validates it.
func LowerInterfaces(graph *load.Graph) {
	if !graph.Ok {
		return
	}
	interfaces, methods, concrete := collectInterfaceTypes(graph)
	for i := 0; i < len(interfaces); i++ {
		candidate := ""
		for j := 0; j < len(concrete); j++ {
			if concrete[j].pkg != interfaces[i].pkg || !implementsInterface(methods, &concrete[j], &interfaces[i]) {
				continue
			}
			if candidate != "" {
				candidate = ""
				break
			}
			candidate = concrete[j].name
		}
		interfaces[i].candidate = candidate
	}
	for pkgIndex := 0; pkgIndex < len(graph.Packages); pkgIndex++ {
		for fileIndex := 0; fileIndex < len(graph.Packages[pkgIndex].Files); fileIndex++ {
			file := &graph.Packages[pkgIndex].Files[fileIndex]
			edits := interfaceFileEdits(&file.File, interfaces, pkgIndex, fileIndex)
			if len(edits) == 0 {
				continue
			}
			file.Src = applySourceEdits(file.Src, edits)
			file.File = syntax.ParseFile(file.Src)
		}
	}
}

func collectInterfaceTypes(graph *load.Graph) ([]interfaceType, []concreteMethod, []concreteType) {
	var interfaces []interfaceType
	var methods []concreteMethod
	var concrete []concreteType
	for pkgIndex := 0; pkgIndex < len(graph.Packages); pkgIndex++ {
		pkg := graph.Packages[pkgIndex]
		for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
			file := pkg.Files[fileIndex].File
			for i := 0; i < len(file.Decls); i++ {
				decl := file.Decls[i]
				if decl.Kind != syntax.TokenType {
					continue
				}
				name := semanticToken(&file, decl.NameTok)
				typeStart := decl.NameTok + 1
				if semanticToken(&file, typeStart) == "=" {
					typeStart++
				}
				if typeStart < decl.EndTok && file.Tokens[typeStart].Kind == syntax.TokenInterface && semanticToken(&file, typeStart+1) == "{" {
					closeTok := semanticMatching(&file, typeStart+1, "{", "}")
					if closeTok > typeStart+1 {
						interfaces = append(interfaces, interfaceType{pkg: pkgIndex, file: fileIndex, name: name, start: file.Tokens[typeStart].Start, end: file.Tokens[closeTok].End, methods: semanticInterfaceMethods(&file, typeStart+2, closeTok)})
					}
				} else if name != "" {
					concrete = append(concrete, concreteType{pkg: pkgIndex, name: name})
				}
			}
			for i := 0; i < len(file.Funcs); i++ {
				fn := file.Funcs[i]
				if fn.ReceiverStart < 0 {
					continue
				}
				receiver := ""
				pointer := false
				for tok := fn.ReceiverStart; tok < fn.ReceiverEnd; tok++ {
					if semanticToken(&file, tok) == "*" {
						pointer = true
					}
					if file.Tokens[tok].Kind == syntax.TokenIdent {
						receiver = semanticToken(&file, tok)
					}
				}
				openTok := fn.NameTok + 1
				closeTok := semanticMatching(&file, openTok, "(", ")")
				if receiver != "" && closeTok > openTok {
					methods = append(methods, concreteMethod{pkg: pkgIndex, receiver: receiver, pointer: pointer, name: semanticToken(&file, fn.NameTok), signature: semanticTokenSpan(&file, openTok, fn.BodyStart)})
				}
			}
		}
	}
	return interfaces, methods, concrete
}

func semanticInterfaceMethods(file *syntax.File, start int, end int) []interfaceMethod {
	var out []interfaceMethod
	for i := start; i+1 < end; i++ {
		if file.Tokens[i].Kind != syntax.TokenIdent || semanticToken(file, i+1) != "(" {
			continue
		}
		closeTok := semanticMatching(file, i+1, "(", ")")
		if closeTok < i+1 {
			return nil
		}
		resultEnd := closeTok + 1
		for resultEnd < end && semanticToken(file, resultEnd) != ";" && file.Tokens[resultEnd].Line == file.Tokens[closeTok].Line {
			resultEnd++
		}
		out = append(out, interfaceMethod{name: semanticToken(file, i), signature: semanticTokenSpan(file, i+1, resultEnd)})
		i = resultEnd
	}
	return out
}

func implementsInterface(methods []concreteMethod, concrete *concreteType, iface *interfaceType) bool {
	if len(iface.methods) == 0 {
		return false
	}
	for i := 0; i < len(iface.methods); i++ {
		found := false
		for j := 0; j < len(methods); j++ {
			method := methods[j]
			if method.pkg == concrete.pkg && method.receiver == concrete.name && !method.pointer && method.name == iface.methods[i].name && method.signature == iface.methods[i].signature {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func interfaceFileEdits(file *syntax.File, interfaces []interfaceType, pkgIndex int, fileIndex int) []sourceEdit {
	var edits []sourceEdit
	var names []string
	var candidates []string
	for i := 0; i < len(interfaces); i++ {
		iface := interfaces[i]
		if iface.pkg != pkgIndex || iface.candidate == "" {
			continue
		}
		names = append(names, iface.name)
		candidates = append(candidates, iface.candidate)
		if iface.file == fileIndex {
			edits = append(edits, sourceEdit{start: iface.start, end: iface.end, text: "= " + iface.candidate})
		}
	}
	if len(names) == 0 {
		return edits
	}
	var values []string
	for i := 0; i+1 < len(file.Tokens); i++ {
		nameIndex := semanticStringIndex(names, semanticToken(file, i+1))
		if file.Tokens[i].Kind == syntax.TokenIdent && nameIndex >= 0 {
			values = semanticAppendUnique(values, semanticToken(file, i))
			edits = append(edits, sourceEdit{start: file.Tokens[i+1].Start, end: file.Tokens[i+1].End, text: candidates[nameIndex]})
		}
	}
	for i := 1; i+4 < len(file.Tokens); i++ {
		base := semanticToken(file, i-1)
		if semanticStringIndex(values, base) < 0 || semanticToken(file, i) != "." || semanticToken(file, i+1) != "(" || semanticToken(file, i+3) != ")" {
			continue
		}
		if semanticToken(file, i+2) == "type" {
			if i >= 2 && file.Tokens[i-2].Kind == syntax.TokenSwitch {
				closeBrace := semanticMatching(file, i+4, "{", "}")
				if closeBrace > i+4 {
					edits = append(edits, sourceEdit{start: file.Tokens[i-1].Start, end: file.Tokens[i+3].End, text: "1"})
					edits = append(edits, semanticTypeSwitchCaseEdits(file, i+5, closeBrace, candidates)...)
				}
			}
			continue
		}
		if semanticStringIndex(candidates, semanticToken(file, i+2)) < 0 {
			continue
		}
		replacement := ""
		if semanticCommaOK(file, i-1) {
			replacement = ", true"
		}
		edits = append(edits, sourceEdit{start: file.Tokens[i].Start, end: file.Tokens[i+3].End, text: replacement})
	}
	return edits
}

func semanticTypeSwitchCaseEdits(file *syntax.File, start int, end int, candidates []string) []sourceEdit {
	var edits []sourceEdit
	for i := start; i+1 < end; i++ {
		if file.Tokens[i].Kind != syntax.TokenCase || file.Tokens[i+1].Kind != syntax.TokenIdent {
			continue
		}
		value := "0"
		if semanticStringIndex(candidates, semanticToken(file, i+1)) >= 0 {
			value = "1"
		}
		edits = append(edits, sourceEdit{start: file.Tokens[i+1].Start, end: file.Tokens[i+1].End, text: value})
	}
	return edits
}

func semanticCommaOK(file *syntax.File, rhs int) bool {
	assignment := false
	for i := rhs - 1; i >= 0 && file.Tokens[i].Line == file.Tokens[rhs].Line; i-- {
		text := semanticToken(file, i)
		if text == ":=" || text == "=" {
			assignment = true
			continue
		}
		if assignment && text == "," {
			return true
		}
	}
	return false
}

func applySourceEdits(src []byte, edits []sourceEdit) []byte {
	for i := 1; i < len(edits); i++ {
		item := edits[i]
		j := i - 1
		for j >= 0 && edits[j].start > item.start {
			edits[j+1] = edits[j]
			j--
		}
		edits[j+1] = item
	}
	out := make([]byte, 0, len(src))
	at := 0
	for i := 0; i < len(edits); i++ {
		edit := edits[i]
		if edit.start < at || edit.end < edit.start || edit.end > len(src) {
			continue
		}
		out = append(out, src[at:edit.start]...)
		out = append(out, []byte(edit.text)...)
		at = edit.end
	}
	return append(out, src[at:]...)
}

func semanticMatching(file *syntax.File, open int, left string, right string) int {
	if semanticToken(file, open) != left {
		return -1
	}
	depth := 0
	for i := open; i < len(file.Tokens); i++ {
		if semanticToken(file, i) == left {
			depth++
		} else if semanticToken(file, i) == right {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func semanticTokenSpan(file *syntax.File, start int, end int) string {
	out := ""
	for i := start; i < end; i++ {
		out += semanticToken(file, i)
	}
	return out
}

func semanticToken(file *syntax.File, index int) string {
	if index < 0 || index >= len(file.Tokens) {
		return ""
	}
	return string(syntax.TokenText(file.Src, file.Tokens[index]))
}

func semanticStringIndex(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return i
		}
	}
	return -1
}

func semanticAppendUnique(values []string, value string) []string {
	if value != "" && semanticStringIndex(values, value) < 0 {
		return append(values, value)
	}
	return values
}
