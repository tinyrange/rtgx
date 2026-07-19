package link

import (
	"renvo.dev/internal/syntax"
	"renvo.dev/internal/unit"
)

func lowerOrdinaryBuiltins(program *unit.Program) bool {
	stringLess := ""
	generated := ""
	generatedCount := 0
	for {
		changed := false
		for i := len(program.Tokens) - 2; i >= 0; i-- {
			name := functionValueTokenText(program, i)
			if (name != "min" && name != "max" && name != "clear") || !functionValueTokenEquals(program, i+1, "(") || ordinaryBuiltinShadowed(program, i, name) {
				continue
			}
			close := functionValueFindMatchingParen(program, i+1)
			if close < 0 {
				return false
			}
			starts, ends := ordinaryBuiltinArguments(program, i+2, close)
			replacement := ""
			if name == "clear" {
				if len(starts) != 1 {
					return false
				}
				replacement = ordinaryClearText(program, i, starts[0], ends[0])
			} else {
				if len(starts) == 0 {
					return false
				}
				if constant := ordinaryConstantMinMax(program, name, starts, ends); constant != "" {
					replacement = constant
				} else {
					typ := ""
					for arg := 0; arg < len(starts); arg++ {
						candidate := ordinaryBuiltinExprType(program, i, starts[arg], ends[arg])
						if candidate != "" && !ordinaryUntypedExpression(program, starts[arg], ends[arg]) {
							typ = candidate
							break
						}
					}
					if typ == "" {
						typ = ordinaryBuiltinExprType(program, i, starts[0], ends[0])
					}
					if typ == "" {
						return false
					}
					underlying := ordinaryUnderlyingType(program, typ, 0)
					if underlying == "string" {
						if stringLess == "" {
							stringLess = ordinaryBuiltinGeneratedName(program, "__renvo_builtin_string_less")
						}
					}
					var declaration string
					replacement, declaration = ordinaryMinMaxText(program, name, typ, underlying, stringLess, generatedCount, starts, ends)
					generated += declaration
					generatedCount++
				}
			}
			if replacement == "" {
				return false
			}
			text, ok := applyFunctionValueEdits(program.Text, []functionValueEdit{functionValueTokenRangeEdit(program, i, close+1, replacement)})
			if !ok || !reparseFunctionValueProgram(program, text) {
				return false
			}
			changed = true
			break
		}
		if !changed {
			break
		}
	}
	if stringLess == "" && generated == "" {
		return true
	}
	text := program.Text
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	text = appendFunctionValueString(text, generated)
	if stringLess != "" {
		help := "func " + stringLess + "(left string, right string) bool { limit := len(left); if len(right) < limit { limit = len(right) }; for index := 0; index < limit; index++ { if left[index] < right[index] { return true }; if left[index] > right[index] { return false } }; return len(left) < len(right) }\n"
		text = appendFunctionValueString(text, help)
	}
	return reparseFunctionValueProgram(program, text)
}

func ordinaryBuiltinArguments(program *unit.Program, start int, close int) ([]int, []int) {
	var starts []int
	var ends []int
	part := start
	paren := 0
	bracket := 0
	brace := 0
	for i := start; i <= close; i++ {
		text := functionValueTokenText(program, i)
		if i < close {
			if text == "(" {
				paren++
			} else if text == ")" {
				paren--
			} else if text == "[" {
				bracket++
			} else if text == "]" {
				bracket--
			} else if text == "{" {
				brace++
			} else if text == "}" {
				brace--
			}
		}
		if i == close || text == "," && paren == 0 && bracket == 0 && brace == 0 {
			if part < i {
				starts = append(starts, part)
				ends = append(ends, i)
			}
			part = i + 1
		}
	}
	return starts, ends
}

func ordinaryMinMaxText(program *unit.Program, name string, typ string, underlying string, stringLess string, call int, starts []int, ends []int) (string, string) {
	prefix := "__renvo_builtin_" + name + "_" + functionValueDecimal(call)
	typeName := prefix + "_arguments"
	functionName := prefix + "_call"
	declaration := "type " + typeName + " struct { "
	callText := functionName + "(" + typeName + "{"
	for i := 0; i < len(starts); i++ {
		field := "value" + functionValueDecimal(i)
		declaration += field + " " + typ + "; "
		if i > 0 {
			callText += ", "
		}
		callText += field + ": " + functionValueTokensText(program, starts[i], ends[i])
	}
	declaration += "}\nfunc " + functionName + "(values " + typeName + ") " + typ + " { result := values.value0; "
	for i := 1; i < len(starts); i++ {
		value := "values.value" + functionValueDecimal(i)
		less := value + " < result"
		if stringLess != "" {
			less = stringLess + "(" + value + ", result)"
		}
		if name == "max" {
			if stringLess != "" {
				less = stringLess + "(result, " + value + ")"
			} else {
				less = "result < " + value
			}
		}
		if underlying == "float32" || underlying == "float64" {
			declaration += "if " + value + " != " + value + " { return " + value + " }; "
		}
		declaration += "if " + less + " { result = " + value + " }; "
	}
	declaration += "return result }\n"
	return callText + "})", declaration
}

func ordinaryClearText(program *unit.Program, call int, start int, end int) string {
	typ := ordinaryBuiltinExprType(program, call, start, end)
	underlying := ordinaryUnderlyingType(program, typ, 0)
	prefix := "__renvo_builtin_" + functionValueDecimal(call)
	expr := functionValueTokensText(program, start, end)
	if functionValueHasPrefix(underlying, "map[") {
		return "{ " + prefix + " := " + expr + "; for " + prefix + "_key := range " + prefix + " { delete(" + prefix + ", " + prefix + "_key) } }"
	}
	if functionValueHasPrefix(underlying, "[]") {
		elem := underlying[2:]
		return "{ " + prefix + " := " + expr + "; var " + prefix + "_zero " + elem + "; for " + prefix + "_index := range " + prefix + " { " + prefix + "[" + prefix + "_index] = " + prefix + "_zero } }"
	}
	return ""
}

func ordinaryBuiltinExprType(program *unit.Program, before int, start int, end int) string {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if end <= start {
		return ""
	}
	if end-start == 1 {
		kind := program.Tokens[start].Kind
		if kind == unit.TokenString {
			return "string"
		}
		if kind == unit.TokenFloat {
			return "float64"
		}
		if kind == unit.TokenNumber || kind == unit.TokenChar {
			return "int"
		}
		name := functionValueTokenText(program, start)
		if name == "true" || name == "false" {
			return "bool"
		}
		if typ := functionValueEnclosingLocalType(program, before, name); typ != "" {
			return typ
		}
		return ordinaryGlobalType(program, name)
	}
	if (functionValueTokenEquals(program, start, "+") || functionValueTokenEquals(program, start, "-") || functionValueTokenEquals(program, start, "^")) && start+1 < end {
		return ordinaryBuiltinExprType(program, before, start+1, end)
	}
	if program.Tokens[start].Kind == unit.TokenIdent && functionValueTokenEquals(program, start+1, "(") && functionValueFindMatchingParen(program, start+1) == end-1 {
		name := functionValueTokenText(program, start)
		if name == "make" {
			starts, ends := ordinaryBuiltinArguments(program, start+2, end-1)
			if len(starts) > 0 {
				return functionValueTokensText(program, starts[0], ends[0])
			}
		}
		if name == "min" || name == "max" {
			starts, ends := ordinaryBuiltinArguments(program, start+2, end-1)
			for i := 0; i < len(starts); i++ {
				if typ := ordinaryBuiltinExprType(program, before, starts[i], ends[i]); typ != "" {
					return typ
				}
			}
		}
		if ordinaryBuiltinTypeName(name) {
			return name
		}
		return functionValueDeclaredFunctionResultType(program, name)
	}
	paren := 0
	bracket := 0
	brace := 0
	for i := start; i < end; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" {
			paren++
		} else if text == ")" {
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" {
			bracket--
		} else if text == "{" {
			brace++
		} else if text == "}" {
			brace--
		} else if paren == 0 && bracket == 0 && brace == 0 && (text == "+" || text == "-" || text == "*" || text == "/" || text == "%" || text == "&" || text == "|" || text == "^" || text == "<<" || text == ">>") {
			return ordinaryBuiltinExprType(program, before, start, i)
		}
	}
	return ""
}

func ordinaryBuiltinTypeName(name string) bool {
	return name == "string" || name == "float32" || name == "float64" || name == "int" || name == "int8" || name == "int16" || name == "int32" || name == "int64" || name == "uint" || name == "uint8" || name == "uint16" || name == "uint32" || name == "uint64" || name == "uintptr" || name == "byte" || name == "rune"
}

func ordinaryGlobalType(program *unit.Program, name string) string {
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		if functionValueTokenText(program, functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)) != name {
			continue
		}
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		start := nameTok + 1
		end := functionValueTypeEnd(program, start)
		if end > start && !functionValueTokenEquals(program, start, "=") {
			return functionValueTokensText(program, start, end)
		}
		if functionValueTokenEquals(program, start, "=") {
			return ordinaryBuiltinExprType(program, nameTok, start+1, decl.EndTok)
		}
	}
	return ""
}

func ordinaryUnderlyingType(program *unit.Program, typ string, depth int) string {
	if depth > len(program.Decls)+1 || ordinaryBuiltinTypeName(typ) || functionValueHasPrefix(typ, "[]") || functionValueHasPrefix(typ, "map[") {
		return typ
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if functionValueTokenText(program, nameTok) != typ {
			continue
		}
		start := nameTok + 1
		if functionValueTokenEquals(program, start, "=") {
			start++
		}
		return ordinaryUnderlyingType(program, functionValueTokensText(program, start, decl.EndTok), depth+1)
	}
	return typ
}

func ordinaryUntypedExpression(program *unit.Program, start int, end int) bool {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if end-start == 1 {
		kind := program.Tokens[start].Kind
		return kind == unit.TokenNumber || kind == unit.TokenFloat || kind == unit.TokenString || kind == unit.TokenChar
	}
	return false
}

func ordinaryConstantMinMax(program *unit.Program, name string, starts []int, ends []int) string {
	best := 0
	kind, intValue, stringValue, ok := ordinaryConstantValue(program, starts[0], ends[0])
	if !ok {
		return ""
	}
	for i := 1; i < len(starts); i++ {
		nextKind, nextInt, nextString, nextOK := ordinaryConstantValue(program, starts[i], ends[i])
		if !nextOK || nextKind != kind {
			return ""
		}
		selectNext := nextInt < intValue
		if kind == 2 {
			selectNext = ordinaryStringLess(nextString, stringValue)
		}
		if name == "max" {
			selectNext = !selectNext && (nextInt != intValue || kind == 2 && nextString != stringValue)
		}
		if selectNext {
			best = i
			intValue = nextInt
			stringValue = nextString
		}
	}
	return functionValueTokensText(program, starts[best], ends[best])
}

func ordinaryStringLess(left string, right string) bool {
	limit := len(left)
	if len(right) < limit {
		limit = len(right)
	}
	for i := 0; i < limit; i++ {
		if left[i] < right[i] {
			return true
		}
		if left[i] > right[i] {
			return false
		}
	}
	return len(left) < len(right)
}

func ordinaryConstantValue(program *unit.Program, start int, end int) (int, int, string, bool) {
	for end-start >= 2 && functionValueTokenEquals(program, start, "(") && functionValueFindMatchingParen(program, start) == end-1 {
		start++
		end--
	}
	if program.Tokens[start].Kind == unit.TokenIdent && start+2 < end && functionValueTokenEquals(program, start+1, "(") && functionValueFindMatchingParen(program, start+1) == end-1 && ordinaryBuiltinTypeName(functionValueTokenText(program, start)) {
		return ordinaryConstantValue(program, start+2, end-1)
	}
	if end-start == 1 && program.Tokens[start].Kind == unit.TokenString {
		tok := program.Tokens[start]
		value, ok := syntax.StringLiteralValue(program.Text, syntax.Token{Kind: syntax.TokenString, Start: tok.Start, End: tok.Start + tok.Size, Line: tok.Line})
		return 2, 0, value, ok
	}
	sign := 1
	if end-start == 2 && functionValueTokenEquals(program, start, "-") {
		sign = -1
		start++
	}
	if end-start != 1 || program.Tokens[start].Kind != unit.TokenNumber {
		return 0, 0, "", false
	}
	value, ok := ordinaryParseInt(functionValueTokenText(program, start))
	return 1, sign * value, "", ok
}

func ordinaryParseInt(text string) (int, bool) {
	base := 10
	start := 0
	if len(text) > 2 && text[0] == '0' {
		if text[1] == 'x' || text[1] == 'X' {
			base, start = 16, 2
		} else if text[1] == 'b' || text[1] == 'B' {
			base, start = 2, 2
		} else if text[1] == 'o' || text[1] == 'O' {
			base, start = 8, 2
		}
	}
	value := 0
	for i := start; i < len(text); i++ {
		c := text[i]
		if c == '_' {
			continue
		}
		digit := -1
		if c >= '0' && c <= '9' {
			digit = int(c - '0')
		} else if c >= 'a' && c <= 'f' {
			digit = int(c-'a') + 10
		} else if c >= 'A' && c <= 'F' {
			digit = int(c-'A') + 10
		}
		if digit < 0 || digit >= base {
			return 0, false
		}
		value = value*base + digit
	}
	return value, true
}

func ordinaryBuiltinGeneratedName(program *unit.Program, base string) string {
	name := base
	for ordinaryBuiltinTopLevelObject(program, name) {
		name += "_"
	}
	return name
}

func ordinaryBuiltinShadowed(program *unit.Program, at int, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].NameTok == at {
			return true
		}
	}
	if at > 0 && functionValueTokenEquals(program, at-1, ".") {
		return true
	}
	if ordinaryBuiltinTopLevelObject(program, name) || functionValueEnclosingLocalType(program, at, name) != "" {
		return true
	}
	fnIndex := -1
	for i := 0; i < len(program.Funcs); i++ {
		if program.Funcs[i].BodyStart < at && at < program.Funcs[i].BodyEnd {
			fnIndex = i
			break
		}
	}
	if fnIndex < 0 {
		return false
	}
	// Track the active lexical blocks so declarations in an earlier sibling
	// block do not hide a predeclared builtin at the call site.
	active := []bool{false}
	for i := program.Funcs[fnIndex].BodyStart + 1; i < at; i++ {
		if functionValueTokenEquals(program, i, "{") {
			active = append(active, false)
			continue
		}
		if functionValueTokenEquals(program, i, "}") {
			if len(active) > 1 {
				active = active[:len(active)-1]
			}
			continue
		}
		if functionValueTokenEquals(program, i, name) && ordinaryBuiltinLocalDeclaration(program, i, at) {
			active[len(active)-1] = true
		}
	}
	for i := 0; i < len(active); i++ {
		if active[i] {
			return true
		}
	}
	return false
}

func ordinaryBuiltinTopLevelObject(program *unit.Program, name string) bool {
	for i := 0; i < len(program.Funcs); i++ {
		fn := program.Funcs[i]
		if fn.ReceiverStart >= fn.ReceiverEnd && functionValueTokenText(program, fn.NameTok) == name {
			return true
		}
	}
	for i := 0; i < len(program.Decls); i++ {
		decl := program.Decls[i]
		nameTok := functionValueTokenAtSpan(program, decl.NameStart, decl.NameEnd)
		if nameTok >= 0 && functionValueTokenText(program, nameTok) == name {
			return true
		}
	}
	return false
}

func ordinaryBuiltinLocalDeclaration(program *unit.Program, nameTok int, limit int) bool {
	if nameTok > 0 && (functionValueTokenEquals(program, nameTok-1, "var") || functionValueTokenEquals(program, nameTok-1, "const") || functionValueTokenEquals(program, nameTok-1, "type")) {
		return true
	}
	paren, bracket := 0, 0
	for i := nameTok + 1; i < limit; i++ {
		text := functionValueTokenText(program, i)
		if text == "(" {
			paren++
		} else if text == ")" {
			if paren == 0 {
				return false
			}
			paren--
		} else if text == "[" {
			bracket++
		} else if text == "]" {
			bracket--
		} else if paren == 0 && bracket == 0 {
			if text == ":=" {
				return true
			}
			if text == ";" || text == "{" || text == "}" || text == "=" {
				return false
			}
		}
	}
	return false
}
