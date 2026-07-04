package check

import (
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/target"
)

type Diagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

func (d Diagnostic) Error() string {
	return d.Path + ":" + strconv.Itoa(d.Line) + ":" + strconv.Itoa(d.Column) + ": " + d.Message
}

type Diagnostics []Diagnostic

type interfaceReturnLowerableSet struct {
	enabled bool
	names   []string
}

type diagnosticError string

func (err diagnosticError) Error() string {
	return string(err)
}

type exportedPackage struct {
	importPath string
	names      []string
}

type importEntry struct {
	name string
	path string
}

type importNameToken struct {
	name string
	tok  scan.Token
}

type localShadow struct {
	name  string
	start int
	end   int
}

type funcSignature struct {
	name            string
	params          int
	results         int
	paramTypes      []string
	resultTypes     []string
	callbackParams  []functionParamSignature
	erasedParams    []int
	receiverType    string
	pointerReceiver bool
	namedResults    []string
	variadic        bool
	staticAlias     bool
	staticCallback  bool
}

type functionParamSignature struct {
	index int
	name  string
	sig   funcSignature
}

type structFieldSet struct {
	name       string
	fields     []string
	fieldTypes []localValueType
}

type localStructType struct {
	name      string
	qualifier string
	typeName  string
	pointer   bool
}

type importedMethod struct {
	qualifier string
	typeName  string
	name      string
	sig       funcSignature
}

type importedFunction struct {
	qualifier  string
	importPath string
	sig        funcSignature
}

type functionAlias struct {
	name           string
	sig            funcSignature
	start          int
	end            int
	staticCallback bool
}

type functionLiteralInfo struct {
	start       int
	paramsOpen  int
	paramsClose int
	bodyOpen    int
	bodyClose   int
	end         int
	sig         funcSignature
}

type functionLiteralDirectCallInfo struct {
	literal   functionLiteralInfo
	callOpen  int
	callClose int
}

type importedValue struct {
	qualifier string
	name      string
	typ       string
	raw       string
}

type localValueType struct {
	name     string
	typ      string
	raw      string
	embedded bool
	start    int
	end      int
	scoped   bool
}

type expressionRange struct {
	start int
	end   int
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}
	var parts []string
	for i := 0; i < len(d); i++ {
		diag := d[i]
		parts = append(parts, diagnosticString(diag))
	}
	return strings.Join(parts, "\n")
}

func diagnosticString(diag Diagnostic) string {
	return diag.Error()
}

func Graph(g *load.Graph) error {
	var diags Diagnostics
	targetName := target.Default()
	if g.Target != "" {
		targetName = g.Target
	}
	wordSize := target.WordSize(targetName)
	parseDiags := parseGraphFiles(g)
	diags = appendDiagnostics(diags, parseDiags)
	if len(diags) > 0 {
		return diagnosticError(diags.Error())
	}
	exported, parseDiags := exportedDecls(g)
	diags = appendDiagnostics(diags, parseDiags)
	if len(diags) > 0 {
		return diagnosticError(diags.Error())
	}
	var packages []load.Package
	packages = g.Packages
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		var pkg load.Package
		pkg = packages[pkgIndex]
		var files []load.File
		files = pkg.Files
		pkgTopNames := packageTopLevelNames(files)
		pkgTopTypes := packageTopLevelNamesOfKind(files, "type")
		pkgTopConsts := packageTopLevelNamesOfKind(files, "const")
		pkgTypeNames := packageNamedTypeUnderlyings(files)
		pkgStructs := packageStructTypesWithTypes(files, pkgTypeNames)
		pkgSigs := packageFunctionSignaturesWithTypes(files, pkgStructs, pkgTypeNames)
		pkgSigs = appendBackendRuntimeIntrinsicSignatures(pkg.ImportPath, pkgSigs)
		pkgTopValues := packageTopLevelValueTypesWithImportsAndTypes(files, packages, pkgTypeNames)
		pkgTopStructValues := packageTopLevelStructValueTypes(files, pkgStructs)
		pkgInterfaceReturns := interfaceReturnLowerableSetForLoadFiles(files, pkgSigs)
		nameCap := packageLevelNameCapacity(files)
		names := make([]string, 0, nameCap)
		nameDiags := make([]Diagnostic, 0, nameCap)
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			var file load.File
			file = files[fileIndex]
			mark := arena.Mark()
			parsed, err := parsedLoadFile(file)
			if err != nil {
				diags = appendParseDiagnostic(diags, file.Path, err)
				continue
			}
			parsedPackageName := parsed.PackageName
			loadedPackageName := pkg.Name
			if !sameString(parsedPackageName, loadedPackageName) {
				diags = appendDiagnostic(diags, Diagnostic{Path: file.Path, Line: 1, Column: 1, Message: "package name changed during parsing"})
				continue
			}
			selectorDiags := importedSelectorDiagnostics(parsed, exported)
			diags = appendDiagnostics(diags, fileDiagnostics(parsed, pkgTopNames, pkgTopTypes, pkgTopConsts, pkgSigs, pkgStructs, pkgTopValues, pkgTopStructValues, packages, pkgTypeNames, pkgInterfaceReturns, len(selectorDiags) > 0, wordSize))
			diags = appendDiagnostics(diags, selectorDiags)
			var decls []parse.Decl
			decls = parsed.Decls
			for declIndex := 0; declIndex < len(decls); declIndex++ {
				var decl parse.Decl
				decl = decls[declIndex]
				namesForDecl := packageLevelDeclNames(decl)
				for i := 0; i < len(namesForDecl); i++ {
					name := namesForDecl[i]
					if name == "" || name == "_" {
						continue
					}
					current := declNameDiagnostic(parsed, decl, i, "duplicate package-level declaration: "+name)
					previousIndex := stringIndex(names, name)
					if previousIndex >= 0 {
						previous := nameDiags[previousIndex]
						diags = append(diags, previous)
						diags = append(diags, current)
						continue
					}
					names = append(names, arena.PersistString(name))
					nameDiags = append(nameDiags, current)
				}
			}
			if len(diags) == 0 {
				arena.Reset(mark)
			}
		}
	}
	if len(diags) > 0 {
		return diagnosticError(diags.Error())
	}
	return nil
}

func parseGraphFiles(g *load.Graph) Diagnostics {
	var diags Diagnostics
	for pkgIndex := 0; pkgIndex < len(g.Packages); pkgIndex++ {
		files := g.Packages[pkgIndex].Files
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			if files[fileIndex].Parsed.Path != "" {
				continue
			}
			parsed, err := parse.FileSource(files[fileIndex].Path, files[fileIndex].Source)
			if err != nil {
				diags = appendParseDiagnostic(diags, files[fileIndex].Path, err)
				continue
			}
			files[fileIndex].Parsed = parsed
		}
		g.Packages[pkgIndex].Files = files
	}
	return diags
}

func parseDiagnostic(path string, err error) Diagnostic {
	line, column, message, ok := splitPathPositionMessage(path, err.Error())
	if ok {
		return Diagnostic{
			Path:    path,
			Line:    line,
			Column:  column,
			Message: message,
		}
	}
	return Diagnostic{Path: path, Line: 1, Column: 1, Message: err.Error()}
}

func splitPathPositionMessage(path string, message string) (int, int, string, bool) {
	prefix := path + ":"
	if !strings.HasPrefix(message, prefix) {
		return 0, 0, "", false
	}
	rest := message[len(prefix):]
	first := strings.IndexByte(rest, ':')
	if first < 0 {
		return 0, 0, "", false
	}
	second := strings.IndexByte(rest[first+1:], ':')
	if second < 0 {
		return 0, 0, "", false
	}
	second = second + first + 1
	line, err := strconv.Atoi(rest[:first])
	if err != nil {
		return 0, 0, "", false
	}
	column, err := strconv.Atoi(rest[first+1 : second])
	if err != nil {
		return 0, 0, "", false
	}
	return line, column, strings.TrimSpace(rest[second+1:]), true
}

func declDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	var decls []parse.Decl
	decls = file.Decls
	for i := 0; i < len(decls); i++ {
		var decl parse.Decl
		decl = decls[i]
		if decl.Kind == "func" {
			if decl.Name == "init" {
				if !hasOrdinaryMainSignature(file, decl) {
					diags = appendDeclDiagnostic(diags, file, decl, "init function must have no parameters or results")
				}
			}
		}
		if decl.Kind == "type" {
			if tok, ok := declToken(file, decl, "func"); ok {
				if !inertTopLevelFunctionTypeDecl(file, decl) && !inertTopLevelFunctionContainingTypeDecl(file, decl) {
					diags = appendDiag(diags, file, tok, "function values and function types are not supported")
				}
			}
		}
		if file.PackageName == "main" && decl.Kind == "func" && decl.Name == "main" && !hasOrdinaryMainSignature(file, decl) {
			diags = appendDeclDiagnostic(diags, file, decl, "main function must have no parameters or results")
		}
	}
	return diags
}

func exportedDecls(g *load.Graph) ([]exportedPackage, Diagnostics) {
	var out []exportedPackage
	var diags Diagnostics
	var packages []load.Package
	packages = g.Packages
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		var pkg load.Package
		pkg = packages[pkgIndex]
		var files []load.File
		files = pkg.Files
		names := make([]string, 0, packageLevelNameCapacity(files))
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			var file load.File
			file = files[fileIndex]
			mark := arena.Mark()
			parsed, err := parsedLoadFile(file)
			if err != nil {
				diags = appendParseDiagnostic(diags, file.Path, err)
				continue
			}
			var decls []parse.Decl
			decls = parsed.Decls
			for declIndex := 0; declIndex < len(decls); declIndex++ {
				var decl parse.Decl
				decl = decls[declIndex]
				namesForDecl := packageLevelDeclNames(decl)
				for nameIndex := 0; nameIndex < len(namesForDecl); nameIndex++ {
					name := namesForDecl[nameIndex]
					if isExported(name) && !containsString(names, name) {
						names = append(names, arena.PersistString(name))
					}
				}
			}
			arena.Reset(mark)
		}
		var exported exportedPackage
		exported.importPath = pkg.ImportPath
		for i := 0; i < len(names); i++ {
			exported.names = append(exported.names, names[i])
		}
		out = append(out, exported)
	}
	return out, diags
}

func packageLevelNameCapacity(files []load.File) int {
	size := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		decls := files[fileIndex].Parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			size += len(packageLevelDeclNames(decls[declIndex]))
		}
	}
	if size < 8 {
		return 8
	}
	return size
}

func packageLevelDeclNames(decl parse.Decl) []string {
	if decl.Receiver {
		return nil
	}
	return declNames(decl)
}

func declNames(decl parse.Decl) []string {
	if len(decl.Names) > 1 {
		return decl.Names
	}
	if decl.Name == "" {
		if len(decl.Names) > 0 {
			return decl.Names
		}
		return nil
	}
	return []string{decl.Name}
}

func importedSelectorDiagnostics(file parse.File, exported []exportedPackage) Diagnostics {
	var localImports []importEntry
	var importNames []string
	var imports []parse.Import
	imports = file.Imports
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		localName := importLocalName(imp)
		if localName != "" {
			localImports = append(localImports, importEntry{name: localName, path: imp.Path})
			if !containsString(importNames, localName) {
				importNames = append(importNames, localName)
			}
		}
	}
	if len(localImports) == 0 {
		return nil
	}
	shadows := localImportShadows(file, importNames)
	var diags Diagnostics
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i+2 < len(tokens); i++ {
		var local scan.Token
		local = tokens[i]
		var dot scan.Token
		dot = tokens[i+1]
		var member scan.Token
		member = tokens[i+2]
		if local.Kind != scan.Ident || dot.Text != "." || member.Kind != scan.Ident {
			continue
		}
		importPath, ok := importEntryPath(localImports, local.Text)
		if !ok {
			continue
		}
		if isLocalShadowAt(shadows, local.Text, int(local.Start)) {
			continue
		}
		if exportedNameExists(exported, importPath, member.Text) {
			continue
		}
		diags = appendDiag(diags, file, member, "unresolved imported selector: "+importPath+"."+member.Text)
	}
	return diags
}

func File(file parse.File) Diagnostics {
	typeNames := fileNamedTypeUnderlyings(file)
	structs := fileStructTypesWithTypes(file, typeNames)
	return fileDiagnostics(file, fileTopLevelNames(file), fileTopLevelNamesOfKind(file, "type"), fileTopLevelNamesOfKind(file, "const"), functionSignaturesWithTypes(file, structs, typeNames), structs, fileTopLevelValueTypesWithTypeNames(file, typeNames), collectPackageTopLevelStructTypes(file, structs, fileImportNames(file)), nil, typeNames, interfaceReturnLowerableSet{}, false, target.WordSize(target.Default()))
}

func appendBackendRuntimeIntrinsicSignatures(importPath string, sigs []funcSignature) []funcSignature {
	if importPath != "j5.nz/rtg" {
		return sigs
	}
	sigs = append(sigs, funcSignature{name: "open", params: 2, results: 1, paramTypes: []string{"string", "int"}, resultTypes: []string{"int"}})
	sigs = append(sigs, funcSignature{name: "close", params: 1, results: 1, paramTypes: []string{"int"}, resultTypes: []string{"int"}})
	sigs = append(sigs, funcSignature{name: "read", params: 3, results: 1, paramTypes: []string{"int", "[]byte", "int64"}, resultTypes: []string{"int"}})
	sigs = append(sigs, funcSignature{name: "write", params: 3, results: 1, paramTypes: []string{"int", "[]byte", "int64"}, resultTypes: []string{"int"}})
	sigs = append(sigs, funcSignature{name: "chmod", params: 2, results: 1, paramTypes: []string{"int", "int"}, resultTypes: []string{"int"}})
	return sigs
}

func fileDiagnostics(file parse.File, topNames []string, topTypes []string, topConsts []string, sigs []funcSignature, structs []structFieldSet, topValues []localValueType, topStructValues []localStructType, packages []load.Package, typeNames []localValueType, interfaceReturns interfaceReturnLowerableSet, suppressSemantic bool, wordSize int) Diagnostics {
	var diags Diagnostics
	typeNames = appendLocalValueTypes(cloneLocalValueTypes(typeNames), importedNamedTypeUnderlyingsForFile(file, packages))
	previousNamedMaps := activeImportedNamedMapTypes
	previousStaticInterfaceStructs := activeImportedStaticInterfaceStructs
	activeImportedNamedMapTypes = importedNamedMapTypesForFile(file, packages)
	activeImportedStaticInterfaceStructs = importedStructTypesForFile(file, packages)
	diags = appendDiagnostics(diags, directiveDiagnostics(file))
	diags = appendDiagnostics(diags, importDiagnostics(file))
	diags = appendDiagnostics(diags, declDiagnostics(file))
	supportedNewStructs := appendStructFieldSets(nil, structs)
	supportedNewStructs = appendStructFieldSets(supportedNewStructs, importedStructTypesForFile(file, packages))
	var tokens []scan.Token
	tokens = file.Tokens
	unsupportedStructForms := false
	unsupportedSemanticForms := unsupportedImportSemanticForms(file)
	for i := 0; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind == scan.Number && isImaginaryLiteral(tok.Text) && !imaginaryLiteralInPredeclaredReducibleComplexLiteralComponent(tokens, i, sigs) && !discardedLowerableComplexStatementContainingToken(tokens, i, sigs) && !lowerableComplexVarBlankDiscardContainingToken(file, tokens, i, sigs) && !lowerableComplexAliasComponentContainingToken(file, tokens, i, sigs) {
			diags = appendDiag(diags, file, tok, "imaginary literals are not supported")
		}
		switch tok.Text {
		case "...":
			if !ellipsisAllowed(tokens, i) {
				diags = appendDiag(diags, file, tok, "variadic syntax is not supported")
			}
		case "go":
			diags = appendDiag(diags, file, tok, "goroutines are not supported")
		case "chan", "<-":
			diags = appendDiag(diags, file, tok, "channels are not supported")
		case "select":
			diags = appendDiag(diags, file, tok, "select statements are not supported")
		case "interface":
			if inertInterfaceTypeToken(file, i) || inertInterfaceContainingTypeToken(file, i) || lowerableInterfaceVarBlankDiscardContainingToken(file, tokens, i) || lowerableNilInterfaceVarComparisonContainingToken(file, tokens, i) || lowerableUnusedInterfaceParamTypeToken(file, i, sigs) || lowerableDiscardedInterfaceReturnTypeToken(file, i, interfaceReturns, sigs) || lowerableStaticInterfaceAssertionVarContainingToken(file, tokens, i) {
				break
			}
			diags = appendDiag(diags, file, tok, "interfaces are not supported")
			unsupportedSemanticForms = true
		case "map":
			if inertMapTypeToken(file, i) || inertMapContainingTypeToken(file, i) {
				break
			}
			if !discardedLowerableMapCompositeStatementContainingToken(tokens, i) && !discardedLowerableMapSliceCompositeStatementContainingToken(tokens, i) && !discardedLowerableMapMakeStatementContainingToken(tokens, i) && !lowerableMapMakeLenCallContainingToken(tokens, i) && !lowerableMapLiteralDeleteStatementContainingToken(tokens, i) && !lowerableMapLiteralLenCallContainingToken(tokens, i) && !lowerableMapLiteralIndexExpressionContainingToken(tokens, i) && !lowerableMapRangeStatementContainingToken(tokens, i) && !pureMapAliasStatementContainingToken(tokens, i) {
				diags = appendDiag(diags, file, tok, "maps are not supported")
				unsupportedSemanticForms = true
			}
		case "fallthrough":
			if fallthroughTargetCaseColon(tokens, i) < 0 {
				diags = appendDiag(diags, file, tok, "fallthrough is not inside a non-final switch case")
			}
		case "break":
			if hasSameLineLabelOperand(tokens, i) {
				if !labeledBreakAllowedAt(tokens, i) {
					diags = appendDiag(diags, file, tok, "labeled break target is not an enclosing for or switch")
				}
				break
			}
			if !breakAllowedAt(tokens, i) {
				diags = appendDiag(diags, file, tok, "break is not inside a for or switch")
			}
		case "continue":
			if hasSameLineLabelOperand(tokens, i) {
				if !labeledContinueAllowedAt(tokens, i) {
					diags = appendDiag(diags, file, tok, "labeled continue target is not an enclosing for")
				}
				break
			}
			if !continueAllowedAt(tokens, i) {
				diags = appendDiag(diags, file, tok, "continue is not inside a for")
			}
		case "func":
			if inertFunctionTypeToken(file, i) || inertFunctionContainingTypeToken(file, i) {
				break
			}
			if functionTypeTokenInFunctionParameter(file, i) {
				break
			}
			if !file.IsTopLevelFuncAt(i) && (functionLiteralAliasInitializerAt(tokens, i) || functionLiteralDirectCallAt(tokens, i) || functionLiteralStaticCallbackArgumentAt(tokens, i, sigs)) {
				break
			}
			if !file.IsTopLevelFuncAt(i) && functionValueBlankDiscardRHSAt(tokens, i) {
				break
			}
			if startsFunctionTypeInTypeDecl(tokens, i) || !file.IsTopLevelFuncAt(i) {
				diags = appendDiag(diags, file, tok, "function values and function types are not supported")
				unsupportedSemanticForms = true
			}
		}
		if startsArrayType(tokens, i) && !inertArrayTypeToken(file, i) && !frontendLowerableArrayType(file, i) && !discardedLowerableArrayCompositeStatementContainingToken(tokens, i) && !discardedLowerableMapSliceCompositeStatementContainingToken(tokens, i) {
			diags = appendDiag(diags, file, tok, "arrays are not supported")
		}
		if startsNestedSliceType(tokens, i) {
			diags = appendDiag(diags, file, tok, "nested slice types are not supported")
		}
		if startsAnonymousStructType(tokens, i) && !frontendLowerableAnonymousStructType(file, i) && !inertFunctionContainingTypeToken(file, i) && !inertInterfaceContainingTypeToken(file, i) && !inertMapContainingTypeToken(file, i) && !inertComplexContainingTypeToken(file, i) {
			diags = appendDiag(diags, file, tok, "anonymous struct types are not supported")
			unsupportedStructForms = true
		}
		if startsEmbeddedStructField(tokens, i) && !supportedEmbeddedStructField(tokens, i) {
			diags = appendDiag(diags, file, tok, "embedded struct fields are not supported")
			unsupportedStructForms = true
		}
		if startsAnyInterfaceType(tokens, i) && !inertAnyTypeToken(file, i) && !inertInterfaceContainingTypeToken(file, i) && !lowerableInterfaceVarBlankDiscardContainingToken(file, tokens, i) && !lowerableNilInterfaceVarComparisonContainingToken(file, tokens, i) && !lowerableUnusedInterfaceParamTypeToken(file, i, sigs) && !lowerableDiscardedInterfaceReturnTypeToken(file, i, interfaceReturns, sigs) && !lowerableStaticInterfaceAssertionVarContainingToken(file, tokens, i) {
			diags = appendDiag(diags, file, tok, "interfaces are not supported")
			unsupportedSemanticForms = true
		}
		if startsComplexType(tokens, i) && !inertComplexTypeToken(file, i) && !inertComplexContainingTypeToken(file, i) && !lowerableComplexVarBlankDiscardContainingToken(file, tokens, i, sigs) && !lowerableComplexAliasComponentContainingToken(file, tokens, i, sigs) {
			diags = appendDiag(diags, file, tok, "complex numbers are not supported")
			unsupportedSemanticForms = true
		}
		if startsGenericDecl(file, i) {
			var genericTok scan.Token
			genericTok = tokens[i+2]
			diags = appendDiag(diags, file, genericTok, "generics are not supported")
		}
		if startsGenericInstantiation(tokens, i) {
			var genericTok scan.Token
			genericTok = tokens[i+1]
			diags = appendDiag(diags, file, genericTok, "generics are not supported")
		}
		if startsTypeAssertion(tokens, i) && !lowerableStaticInterfaceAssertionAt(file, tokens, i) && !lowerableStaticInterfaceTypeSwitchAt(file, tokens, i) {
			var typeTok scan.Token
			typeTok = tokens[i+1]
			diags = appendDiag(diags, file, typeTok, "type assertions and type switches are not supported")
			unsupportedSemanticForms = true
		}
		if startsUnsupportedBuiltinCall(tokens, i) && !signedFunctionCallAt(tokens, i, sigs) && !isRuntimeOSIntrinsicCall(file, tokens, i) && !startsSupportedNewCall(tokens, i, supportedNewStructs, typeNames) && !startsPredeclaredReducibleComplexComponentCall(tokens, i, sigs) && !complexCallInsidePredeclaredReducibleComplexComponent(tokens, i, sigs) && !discardedLowerableComplexStatementContainingToken(tokens, i, sigs) && !lowerableComplexVarBlankDiscardContainingToken(file, tokens, i, sigs) && !lowerableComplexAliasComponentContainingToken(file, tokens, i, sigs) && !lowerableMapLiteralDeleteStatementContainingToken(tokens, i) && !pureMapAliasDeleteStatementContainingToken(tokens, i) {
			diags = appendDiag(diags, file, tok, "unsupported builtin: "+tok.Text)
		}
		if tok.Kind == scan.Ident && pureMapAliasUnsupportedUseAt(tokens, i) {
			diags = appendDiag(diags, file, tok, "maps are not supported")
			unsupportedSemanticForms = true
		}
		if pureMapAliasElementAssignmentAt(tokens, i) && !pureMapAliasElementAssignmentSupportedAt(tokens, i) {
			diags = appendDiag(diags, file, tok, "map mutation is not supported")
			unsupportedSemanticForms = true
		}
	}
	diags = appendDiagnostics(diags, packageFunctionValueDiagnostics(file, sigs))
	diags = appendDiagnostics(diags, packageValueTypeDiagnostics(file, structs, topValues, packages, typeNames))
	diags = appendDiagnostics(diags, importedFunctionValueDiagnostics(file, packages, sigs))
	if !unsupportedStructForms && !unsupportedSemanticForms && !suppressSemantic {
		diags = appendDiagnostics(diags, functionSemanticDiagnostics(file, topNames, topTypes, topConsts, sigs, structs, topValues, topStructValues, packages, typeNames, wordSize))
	}
	activeImportedNamedMapTypes = previousNamedMaps
	activeImportedStaticInterfaceStructs = previousStaticInterfaceStructs
	return diags
}

func directiveDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	source := file.Source
	line := 1
	column := 1
	for i := 0; i < len(source); {
		c := source[i]
		if c == '\n' {
			i++
			line++
			column = 1
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '/' {
			i += 2
			column += 2
			for i < len(source) && source[i] != '\n' {
				i++
				column++
			}
			continue
		}
		if c == '/' && i+1 < len(source) && source[i+1] == '*' {
			i += 2
			column += 2
			for i < len(source) {
				if source[i] == '\n' {
					i++
					line++
					column = 1
					continue
				}
				if source[i] == '*' && i+1 < len(source) && source[i+1] == '/' {
					i += 2
					column += 2
					break
				}
				i++
				column++
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote := c
			i++
			column++
			escaped := false
			for i < len(source) {
				ch := source[i]
				if ch == '\n' {
					i++
					line++
					column = 1
					escaped = false
					continue
				}
				i++
				column++
				if escaped {
					escaped = false
					continue
				}
				if ch == '\\' {
					escaped = true
					continue
				}
				if ch == quote {
					break
				}
			}
			continue
		}
		if c == '`' {
			i++
			column++
			for i < len(source) {
				ch := source[i]
				i++
				if ch == '\n' {
					line++
					column = 1
					continue
				}
				column++
				if ch == '`' {
					break
				}
			}
			continue
		}
		i++
		column++
	}
	return diags
}

func hasBytesAt(source []byte, pos int, text string) bool {
	if pos+len(text) > len(source) {
		return false
	}
	for i := 0; i < len(text); i++ {
		if source[pos+i] != text[i] {
			return false
		}
	}
	return true
}

func appendDiagnostics(out Diagnostics, values Diagnostics) Diagnostics {
	for i := 0; i < len(values); i++ {
		diag := values[i]
		out = append(out, diag)
	}
	return out
}

func parsedLoadFile(file load.File) (parse.File, error) {
	if file.Parsed.Path != "" {
		return file.Parsed, nil
	}
	return parse.FileSource(file.Path, file.Source)
}

func appendParseDiagnostic(diags Diagnostics, path string, err error) Diagnostics {
	d := parseDiagnostic(path, err)
	return appendDiagnostic(diags, d)
}

func appendDeclDiagnostic(diags Diagnostics, file parse.File, decl parse.Decl, message string) Diagnostics {
	d := declDiagnostic(file, decl, message)
	return appendDiagnostic(diags, d)
}

func appendDiag(diags Diagnostics, file parse.File, tok scan.Token, message string) Diagnostics {
	d := diag(file, tok, message)
	return appendDiagnostic(diags, d)
}

func appendDiagnostic(diags Diagnostics, d Diagnostic) Diagnostics {
	return append(diags, d)
}

func containsString(values []string, value string) bool {
	return stringIndex(values, value) >= 0
}

func sameString(a string, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringIndex(values []string, value string) int {
	for i := 0; i < len(values); i++ {
		if sameString(values[i], value) {
			return i
		}
	}
	return -1
}

func importEntryPath(values []importEntry, name string) (string, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].path, true
		}
	}
	return "", false
}

func exportedNameExists(values []exportedPackage, importPath string, name string) bool {
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.importPath == importPath {
			if len(value.names) == 0 {
				return true
			}
			return containsString(value.names, name)
		}
	}
	return true
}

func hasImportNameToken(values []importNameToken, name string) bool {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return true
		}
	}
	return false
}

func importDiagnostics(file parse.File) Diagnostics {
	var diags Diagnostics
	var names []importNameToken
	var importNames []string
	var imports []parse.Import
	imports = file.Imports
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		localName := importLocalName(imp)
		if localName != "" && localName != "." && localName != "_" {
			if !containsString(importNames, localName) {
				importNames = append(importNames, localName)
			}
		}
	}
	used := usedImportNames(file, importNames)
	for i := 0; i < len(imports); i++ {
		var imp parse.Import
		imp = imports[i]
		localName := importLocalName(imp)
		if localName == "" || localName == "." || localName == "_" {
			continue
		}
		if hasImportNameToken(names, localName) {
			diags = appendDiag(diags, file, imp.Tok, "duplicate import name: "+localName)
			continue
		}
		names = append(names, importNameToken{name: localName, tok: imp.Tok})
		if !containsString(used, localName) {
			diags = appendDiag(diags, file, imp.Tok, "unused import: "+localName)
		}
	}
	return diags
}

func unsupportedImportSemanticForms(file parse.File) bool {
	return false
}

func startsLocalDecl(file parse.File, pos int) bool {
	tokens := file.Tokens
	if topLevelDeclTokenAt(file, pos) {
		return false
	}
	if pos+1 >= len(tokens) || tokens[pos+1].Kind != scan.Ident {
		return false
	}
	if pos == 0 {
		return true
	}
	prev := tokens[pos-1].Text
	return prev == "{" || prev == "}" || prev == ";" || tokens[pos-1].Line != tokens[pos].Line
}

func functionContainsLocalTypeDecl(file parse.File, start int, end int) bool {
	tokens := file.Tokens
	for i := start; i < end; i++ {
		if tokens[i].Text == "type" && startsLocalDecl(file, i) {
			return true
		}
	}
	return false
}

func startsFunctionTypeInTypeDecl(tokens []scan.Token, pos int) bool {
	if pos <= 0 || pos >= len(tokens) || tokens[pos].Text != "func" {
		return false
	}
	namePos := pos - 1
	if tokens[namePos].Text == "=" && namePos > 0 {
		namePos--
	}
	if namePos < 0 || tokens[namePos].Kind != scan.Ident {
		return false
	}
	before := namePos - 1
	if before < 0 {
		return false
	}
	if tokens[before].Text == "type" || tokens[before].Text == "(" || tokens[before].Text == ";" {
		return true
	}
	return tokens[before].Line != tokens[namePos].Line
}

func functionTypeTokenInFunctionParameter(file parse.File, pos int) bool {
	tokens := file.Tokens
	if pos < 0 || pos >= len(tokens) || tokens[pos].Text != "func" {
		return false
	}
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "func" || pos < tokenIndexAt(tokens, decl.Start) {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 || int(tokens[pos].Start) >= decl.End {
			continue
		}
		body := findTokenText(tokens, start, decl.End, "{")
		if body < 0 {
			body = tokenIndexBefore(tokens, decl.End)
		}
		paramsOpen := -1
		for i := start; i < body; i++ {
			if tokens[i].Text == "(" {
				paramsOpen = i
				break
			}
		}
		if paramsOpen < 0 {
			continue
		}
		paramsClose := findClose(tokens, paramsOpen, "(", ")")
		if paramsClose < 0 || paramsClose > body {
			continue
		}
		return pos > paramsOpen && pos < paramsClose
	}
	return false
}

func inertFunctionTypeToken(file parse.File, pos int) bool {
	if inertTopLevelFunctionTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalFunctionTypeToken(file, pos)
}

func inertFunctionContainingTypeToken(file parse.File, pos int) bool {
	if inertTopLevelFunctionContainingTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalFunctionContainingTypeToken(file, pos)
}

func inertInterfaceTypeToken(file parse.File, pos int) bool {
	if inertTopLevelInterfaceTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalInterfaceTypeToken(file, pos)
}

func inertInterfaceContainingTypeToken(file parse.File, pos int) bool {
	if inertTopLevelInterfaceContainingTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalInterfaceContainingTypeToken(file, pos)
}

func inertMapTypeToken(file parse.File, pos int) bool {
	if inertTopLevelMapTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalMapTypeToken(file, pos)
}

func inertMapContainingTypeToken(file parse.File, pos int) bool {
	if inertTopLevelMapContainingTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalMapContainingTypeToken(file, pos)
}

func inertArrayTypeToken(file parse.File, pos int) bool {
	if inertTopLevelArrayTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalArrayTypeToken(file, pos)
}

func inertAnyTypeToken(file parse.File, pos int) bool {
	if inertTopLevelAnyTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalAnyTypeToken(file, pos)
}

func inertComplexTypeToken(file parse.File, pos int) bool {
	if inertTopLevelComplexTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalComplexTypeToken(file, pos)
}

func inertComplexContainingTypeToken(file parse.File, pos int) bool {
	if inertTopLevelComplexContainingTypeToken(file, pos) {
		return true
	}
	if tokenInsideTopLevelTypeDecl(file, pos) {
		return false
	}
	return inertLocalComplexContainingTypeToken(file, pos)
}

func inertTopLevelFunctionTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelFunctionTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelFunctionTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !functionTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingFunctionParameterTypes(file, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !functionTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingFunctionParameterTypes(file, name, decl.Start, decl.End)
}

func inertTopLevelFunctionContainingTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelFunctionContainingTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelFunctionContainingTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !functionContainingTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !functionContainingTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End)
}

func inertTopLevelInterfaceTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelInterfaceTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelInterfaceTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !interfaceTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !interfaceTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, decl.Start, decl.End)
}

func inertTopLevelInterfaceContainingTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelInterfaceContainingTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelInterfaceContainingTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !interfaceContainingTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !interfaceContainingTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, decl.Start, decl.End)
}

func inertTopLevelMapTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelMapTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelMapTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !mapTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !mapTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, decl.Start, decl.End)
}

func inertTopLevelMapContainingTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelMapContainingTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelMapContainingTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !mapContainingTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !mapContainingTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, decl.Start, decl.End)
}

func inertTopLevelArrayTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelArrayTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelArrayTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !arrayTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrUnsafeSizeof(file, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !arrayTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrUnsafeSizeof(file, name, decl.Start, decl.End)
}

func inertTopLevelAnyTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelAnyTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelAnyTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !anyTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !anyTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End)
}

func inertTopLevelComplexTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelComplexTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelComplexTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !complexTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !complexTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End)
}

func inertTopLevelComplexContainingTypeToken(file parse.File, pos int) bool {
	if pos < 0 || pos >= len(file.Tokens) {
		return false
	}
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		if int(file.Tokens[pos].Start) < decl.Start || int(file.Tokens[pos].Start) >= decl.End {
			continue
		}
		return inertTopLevelComplexContainingTypeDecl(file, decl)
	}
	return false
}

func inertTopLevelComplexContainingTypeDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close > tokenIndexBefore(toks, decl.End) {
			return false
		}
		ranges := localTypeSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !complexContainingTypeSpecRange(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecName(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	end := tokenIndexBefore(toks, decl.End) + 1
	if end <= start+2 || !complexContainingTypeSpecRange(toks, start+1, end) {
		return false
	}
	name := functionTypeSpecName(toks, start+1, end)
	return !identifierUsedOutsideSourceRange(toks, name, decl.Start, decl.End)
}

func inertLocalFunctionTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	namePos := functionTypeNameTokenBeforeFunc(toks, pos)
	if namePos < 0 {
		return false
	}
	if topLevelDeclTokenAt(file, namePos-1) {
		return false
	}
	specEnd := functionTypeSpecEndFromName(toks, namePos, len(toks))
	if !functionTypeSpecRange(toks, namePos, specEnd) {
		return false
	}
	name := functionTypeSpecName(toks, namePos, specEnd)
	return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[namePos].Start), int(toks[specEnd-1].End))
}

func inertLocalFunctionContainingTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !functionContainingTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !functionContainingTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalInterfaceTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	namePos := interfaceTypeNameTokenBeforeInterface(toks, pos)
	if namePos < 0 {
		return false
	}
	if !interfaceTypeSpecStartsInTypeDecl(toks, namePos) {
		return false
	}
	if topLevelDeclTokenAt(file, namePos-1) {
		return false
	}
	specEnd := functionTypeSpecEndFromName(toks, namePos, len(toks))
	if !interfaceTypeSpecRange(toks, namePos, specEnd) {
		return false
	}
	name := functionTypeSpecName(toks, namePos, specEnd)
	return name != "" && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, int(toks[namePos].Start), int(toks[specEnd-1].End))
}

func inertLocalInterfaceContainingTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !interfaceContainingTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !interfaceContainingTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalMapTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !mapTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !mapTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalMapContainingTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !mapContainingTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !mapContainingTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalArrayTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !arrayTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRangeAllowingArrayTypeSpecs(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !arrayTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRangeAllowingArrayTypeSpecs(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalAnyTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !anyTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !anyTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalComplexTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !complexTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !complexTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalComplexContainingTypeToken(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" || topLevelDeclTokenAt(file, i) || !startsLocalTypeDeclToken(toks, i, 0) {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if pos < spec.start || pos >= spec.end || !complexContainingTypeSpecRange(toks, spec.start, spec.end) {
					continue
				}
				name := functionTypeSpecName(toks, spec.start, spec.end)
				return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[spec.start].Start), int(toks[spec.end-1].End))
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, len(toks))
		if pos < i+1 || pos >= specEnd || !complexContainingTypeSpecRange(toks, i+1, specEnd) {
			continue
		}
		name := functionTypeSpecName(toks, i+1, specEnd)
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[i+1].Start), int(toks[specEnd-1].End))
	}
	return false
}

func inertLocalUnsupportedTypeDeclEnd(toks []scan.Token, pos int, body int, limit int) (int, bool) {
	if !startsLocalTypeDeclToken(toks, pos, body) {
		return 0, false
	}
	if pos+1 < limit && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close > limit {
			return 0, false
		}
		ranges := localTypeSpecRanges(toks, pos+2, close)
		if len(ranges) == 0 {
			return 0, false
		}
		for i := 0; i < len(ranges); i++ {
			if !inertLocalUnsupportedTypeSpec(toks, ranges[i].start, ranges[i].end) {
				return 0, false
			}
		}
		return close + 1, true
	}
	specEnd := localTypeSingleSpecEnd(toks, pos, limit)
	if !inertLocalUnsupportedTypeSpec(toks, pos+1, specEnd) {
		return 0, false
	}
	return specEnd, true
}

func localTypeDeclEnd(toks []scan.Token, pos int, body int, limit int) (int, bool) {
	if !startsLocalTypeDeclToken(toks, pos, body) {
		return 0, false
	}
	if pos+1 < limit && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close > limit {
			return 0, false
		}
		ranges := localTypeSpecRanges(toks, pos+2, close)
		if len(ranges) == 0 {
			return 0, false
		}
		for i := 0; i < len(ranges); i++ {
			if !localTypeSpecHasType(toks, ranges[i].start, ranges[i].end) {
				return 0, false
			}
		}
		return close + 1, true
	}
	specEnd := localTypeSingleSpecEnd(toks, pos, limit)
	if !localTypeSpecHasType(toks, pos+1, specEnd) {
		return 0, false
	}
	return specEnd, true
}

func localTypeSpecHasType(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return typeStart < end
}

func inertLocalUnsupportedTypeSpec(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if !functionTypeSpecRange(toks, start, end) && !functionContainingTypeSpecRange(toks, start, end) && !interfaceTypeSpecRange(toks, start, end) && !interfaceContainingTypeSpecRange(toks, start, end) && !mapTypeSpecRange(toks, start, end) && !mapContainingTypeSpecRange(toks, start, end) && !arrayTypeSpecRange(toks, start, end) && !anyTypeSpecRange(toks, start, end) && !complexTypeSpecRange(toks, start, end) && !complexContainingTypeSpecRange(toks, start, end) {
		return false
	}
	name := functionTypeSpecName(toks, start, end)
	if functionTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if functionContainingTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if interfaceTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if interfaceContainingTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if mapTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if mapContainingTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if arrayTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRangeAllowingArrayTypeSpecs(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if complexContainingTypeSpecRange(toks, start, end) {
		return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	return name != "" && !identifierUsedOutsideSourceRange(toks, name, int(toks[start].Start), int(toks[end-1].End))
}

func functionTypeNameTokenBeforeFunc(toks []scan.Token, pos int) int {
	if pos <= 0 || pos >= len(toks) || toks[pos].Text != "func" {
		return -1
	}
	namePos := pos - 1
	if toks[namePos].Text == "=" && namePos > 0 {
		namePos--
	}
	if namePos < 0 || toks[namePos].Kind != scan.Ident {
		return -1
	}
	return namePos
}

func interfaceTypeNameTokenBeforeInterface(toks []scan.Token, pos int) int {
	if pos <= 0 || pos >= len(toks) || toks[pos].Text != "interface" {
		return -1
	}
	namePos := pos - 1
	if toks[namePos].Text == "=" && namePos > 0 {
		namePos--
	}
	if namePos < 0 || toks[namePos].Kind != scan.Ident {
		return -1
	}
	return namePos
}

func functionTypeSpecEndFromName(toks []scan.Token, start int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < len(toks) && i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Kind == scan.EOF || toks[i].Text == ";" || toks[i].Text == "}" || toks[i].Text == ")" {
				return i
			}
			if i > start+1 && toks[i].Line > toks[i-1].Line {
				return i
			}
		}
		updateDepth(toks[i].Text, &paren, &brack, &brace)
	}
	if limit < len(toks) {
		return limit
	}
	return len(toks)
}

func functionTypeSpecName(toks []scan.Token, start int, end int) string {
	for start < end && toks[start].Text == ";" {
		start++
	}
	if start < end && toks[start].Kind == scan.Ident {
		return toks[start].Text
	}
	return ""
}

func functionTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || toks[typeStart].Text != "func" {
		return false
	}
	if typeStart+1 >= end || toks[typeStart+1].Text != "(" {
		return false
	}
	close := findClose(toks, typeStart+1, "(", ")")
	if close < 0 || close >= end {
		return false
	}
	if close == end-1 {
		return true
	}
	return functionTypeResultRange(toks, close+1, end)
}

func functionContainingTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalanced(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsFunctionType(toks, typeStart, end)
}

func typeRangeContainsFunctionType(toks []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if toks[i].Text == "func" && toks[i+1].Text == "(" {
			return true
		}
	}
	return false
}

func interfaceTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "interface" || toks[typeStart+1].Text != "{" {
		return false
	}
	close := findClose(toks, typeStart+1, "{", "}")
	return close == end-1
}

func interfaceContainingTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalanced(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsInterfaceType(toks, typeStart, end)
}

func typeRangeContainsInterfaceType(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "interface" || startsUnsupportedPredeclaredType(toks, i, "any") {
			return true
		}
	}
	return false
}

func mapTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+2 >= end || toks[typeStart].Text != "map" || toks[typeStart+1].Text != "[" {
		return false
	}
	close := findClose(toks, typeStart+1, "[", "]")
	return close > typeStart+1 && close < end-1 && typeRangeBalanced(toks, close+1, end)
}

func mapContainingTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalanced(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsMapType(toks, typeStart, end)
}

func typeRangeContainsMapType(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "map" {
			return true
		}
	}
	return false
}

func arrayTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return fixedArrayTypeSpecRange(toks, typeStart, end)
}

func anyTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return typeStart+1 == end && toks[typeStart].Text == "any"
}

func complexTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 != end {
		return false
	}
	return toks[typeStart].Text == "complex64" || toks[typeStart].Text == "complex128"
}

func complexContainingTypeSpecRange(toks []scan.Token, start int, end int) bool {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalanced(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsComplexType(toks, typeStart, end)
}

func typeRangeContainsComplexType(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if startsUnsupportedPredeclaredType(toks, i, "complex64") || startsUnsupportedPredeclaredType(toks, i, "complex128") {
			return true
		}
	}
	return false
}

func typeRangeBalanced(toks []scan.Token, start int, end int) bool {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		updateDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return paren == 0 && brack == 0 && brace == 0
}

func functionTypeResultRange(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		return close == end-1
	}
	return typeTextInRange(toks, start, end) != ""
}

func identifierUsedOutsideSourceRange(toks []scan.Token, name string, start int, end int) bool {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		return true
	}
	return false
}

func identifierUsedOutsideSourceRangeAllowingFunctionParameterTypes(file parse.File, name string, start int, end int) bool {
	toks := file.Tokens
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		if tokenIsNamedFunctionParameterType(file, i) {
			continue
		}
		if tokenIsNamedFunctionTypeAliasUse(file, i) {
			continue
		}
		return true
	}
	return false
}

func tokenIsNamedFunctionParameterType(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "func" || decl.Receiver {
			continue
		}
		name := tokenIndexAt(toks, int(decl.NameTok.Start))
		if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
			continue
		}
		paramsOpen := name + 1
		paramsClose := findClose(toks, paramsOpen, "(", ")")
		if paramsClose < 0 || int(toks[paramsClose].End) > decl.End {
			continue
		}
		if pos <= paramsOpen || pos >= paramsClose {
			continue
		}
		segments := expressionRanges(toks, paramsOpen+1, paramsClose)
		for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
			segment := segments[segmentIndex]
			_, typeStart, typeEnd, hasType := functionParameterSegment(toks, segment.start, segment.end)
			if !hasType {
				continue
			}
			typeStart, typeEnd = trimExpressionRange(toks, typeStart, typeEnd)
			if typeStart == pos && typeStart+1 == typeEnd {
				return true
			}
		}
	}
	return false
}

func tokenIsNamedFunctionTypeAliasUse(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "type" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
			continue
		}
		end := tokenIndexBefore(toks, decl.End) + 1
		if toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 || close >= end {
				continue
			}
			ranges := localTypeSpecRanges(toks, start+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				if tokenIsNamedFunctionTypeAliasUseInSpec(toks, pos, ranges[rangeIndex].start, ranges[rangeIndex].end) {
					return true
				}
			}
			continue
		}
		if tokenIsNamedFunctionTypeAliasUseInSpec(toks, pos, start+1, end) {
			return true
		}
	}
	return false
}

func tokenIsNamedFunctionTypeAliasUseInSpec(toks []scan.Token, pos int, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start+2 != end && start+3 != end {
		return false
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return typeStart == pos && typeStart+1 == end && toks[typeStart].Kind == scan.Ident
}

func identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecs(toks []scan.Token, name string, start int, end int) bool {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		if tokenInsideInterfaceTypeSpec(toks, i) || tokenInsideInterfaceContainingTypeSpec(toks, i) {
			continue
		}
		return true
	}
	return false
}

func identifierUsedOutsideSourceRangeAllowingMapTypeSpecs(toks []scan.Token, name string, start int, end int) bool {
	namedMaps := namedMapTypes(toks)
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		if tokenInsideMapTypeSpec(toks, i) || tokenInsideMapContainingTypeSpec(toks, i) {
			continue
		}
		if identifierInsideDiscardedEmptyCompositeUse(toks, i, name) {
			continue
		}
		if identifierInsideLowerableNamedMapUse(toks, i, name, namedMaps) {
			continue
		}
		return true
	}
	return false
}

func identifierInsideDiscardedEmptyCompositeUse(toks []scan.Token, pos int, name string) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos].Text != name {
		return false
	}
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	matched := false
	for i := 0; i < len(lhs); i++ {
		if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
			return false
		}
		if !discardedEmptyCompositeLiteralExpression(toks, rhs[i].start, rhs[i].end) {
			return false
		}
		if pos >= rhs[i].start && pos < rhs[i].end {
			matched = true
		}
	}
	return matched
}

func discardedEmptyCompositeLiteralExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedEmptyCompositeLiteralExpression(toks, start+1, close)
		}
	}
	if start+3 != end || toks[start].Kind != scan.Ident || toks[start+1].Text != "{" {
		return false
	}
	close := findClose(toks, start+1, "{", "}")
	return close == end-1 && start+2 == close
}

func identifierUsedOutsideSourceRangeAllowingArrayTypeSpecs(toks []scan.Token, name string, start int, end int) bool {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		if tokenInsideArrayTypeSpec(toks, i) {
			continue
		}
		return true
	}
	return false
}

func identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrUnsafeSizeof(file parse.File, name string, start int, end int) bool {
	toks := file.Tokens
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || tok.Text != name {
			continue
		}
		if int(tok.Start) >= start && int(tok.Start) < end {
			continue
		}
		if tokenInsideArrayTypeSpec(toks, i) {
			continue
		}
		if namedCompositeLiteralIsDirectUnsafeSizeofArg(file, toks, i) {
			continue
		}
		return true
	}
	return false
}

func namedCompositeLiteralIsDirectUnsafeSizeofArg(file parse.File, toks []scan.Token, pos int) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "{" {
		return false
	}
	close := findClose(toks, pos+1, "{", "}")
	if close < 0 {
		return false
	}
	return compositeLiteralIsDirectUnsafeSizeofArg(file, toks, pos, close)
}

func tokenInsideInterfaceTypeSpec(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDecl(toks, i) {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i, len(toks))
		if specEnd <= i || !interfaceTypeSpecRange(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideInterfaceContainingTypeSpec(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDecl(toks, i) {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i, len(toks))
		if specEnd <= i || !interfaceContainingTypeSpecRange(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func interfaceTypeSpecStartsInTypeDecl(toks []scan.Token, start int) bool {
	if start <= 0 || start >= len(toks) || toks[start].Kind != scan.Ident {
		return false
	}
	prev := toks[start-1]
	if prev.Text == "type" {
		return true
	}
	if prev.Text == "(" {
		return start > 1 && toks[start-2].Text == "type"
	}
	if prev.Text == ";" {
		return typeBlockContainsOpen(toks, start)
	}
	if prev.Line < toks[start].Line {
		return typeBlockContainsOpen(toks, start)
	}
	return false
}

func tokenInsideMapTypeSpec(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDecl(toks, i) {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i, len(toks))
		if specEnd <= i || !mapTypeSpecRange(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideMapContainingTypeSpec(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDecl(toks, i) {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i, len(toks))
		if specEnd <= i || !mapContainingTypeSpecRange(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideArrayTypeSpec(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDecl(toks, i) {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i, len(toks))
		if specEnd <= i || !arrayTypeSpecRange(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func topLevelDeclTokenAt(file parse.File, pos int) bool {
	tokens := file.Tokens
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	start := int(tokens[pos].Start)
	for i := 0; i < len(file.Decls); i++ {
		if file.Decls[i].Start == start {
			return true
		}
	}
	return false
}

func tokenInsideTopLevelTypeDecl(file parse.File, pos int) bool {
	tokens := file.Tokens
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	start := int(tokens[pos].Start)
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind == "type" && start >= decl.Start && start < decl.End {
			return true
		}
	}
	return false
}

func packageFunctionValueDiagnostics(file parse.File, sigs []funcSignature) Diagnostics {
	var diags Diagnostics
	tokens := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		inInitializer := false
		for i := start + 1; i < len(tokens); i++ {
			tok := tokens[i]
			if int(tok.Start) >= decl.End {
				break
			}
			if tok.Text == "=" {
				inInitializer = true
				continue
			}
			if tok.Text == ";" {
				inInitializer = false
				continue
			}
			if !inInitializer {
				continue
			}
			if isFunctionValueUse(tokens, i, sigs, nil) {
				diags = appendDiag(diags, file, tok, "function values are not supported: "+tok.Text)
			}
		}
	}
	return diags
}

func packageValueTypeDiagnostics(file parse.File, structs []structFieldSet, topValues []localValueType, packages []load.Package, typeNames []localValueType) Diagnostics {
	var diags Diagnostics
	tokens := file.Tokens
	importFuncs := importedFunctionsForFile(file, packages)
	importValues := importedValuesForFile(file, packages)
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" && decl.Kind != "const" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(tokens) && tokens[start+1].Text == "(" {
			close := findClose(tokens, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specStart := start + 2
			for i := specStart; i <= close; i++ {
				if i == close || tokens[i].Text == ";" {
					diags = appendPackageValueSpecTypeDiagnostics(diags, file, tokens, specStart, i, topValues, structs, typeNames, importFuncs, importValues)
					specStart = i + 1
					continue
				}
				if tokens[i].Line != tokens[specStart].Line {
					diags = appendPackageValueSpecTypeDiagnostics(diags, file, tokens, specStart, i, topValues, structs, typeNames, importFuncs, importValues)
					specStart = i
				}
			}
			continue
		}
		end := tokenIndexBefore(tokens, decl.End) + 1
		diags = appendPackageValueSpecTypeDiagnostics(diags, file, tokens, start+1, end, topValues, structs, typeNames, importFuncs, importValues)
	}
	return diags
}

func importedFunctionValueDiagnostics(file parse.File, packages []load.Package, sigs []funcSignature) Diagnostics {
	if len(packages) == 0 {
		return nil
	}
	importFuncs := importedFunctionsForFile(file, packages)
	importMethods := importedMethodsForFile(file, packages)
	if len(importFuncs) == 0 && len(importMethods) == 0 {
		return nil
	}
	importNames := fileImportNames(file)
	importShadows := localImportShadows(file, importNames)
	tokens := file.Tokens
	var diags Diagnostics
	for i := 0; i+2 < len(tokens); i++ {
		tok := tokens[i]
		if tok.Kind != scan.Ident || tokens[i+1].Text != "." || tokens[i+2].Kind != scan.Ident {
			continue
		}
		if !containsString(importNames, tok.Text) || isLocalShadowAt(importShadows, tok.Text, int(tok.Start)) {
			continue
		}
		if i+4 < len(tokens) && tokens[i+3].Text == "." && tokens[i+4].Kind == scan.Ident {
			if importedMethodIndex(importMethods, tok.Text, tokens[i+2].Text, tokens[i+4].Text) >= 0 {
				if i+5 < len(tokens) && tokens[i+5].Text == "(" && tokens[i+5].Line == tokens[i+4].Line {
					continue
				}
				if functionValueBlankDiscardExpressionAt(tokens, i, i+5) {
					continue
				}
				if body := enclosingFunctionBodyOpen(tokens, i); body >= 0 {
					close := findClose(tokens, body, "{", "}")
					if close < 0 {
						close = len(tokens)
					}
					if isMethodExpressionAliasInitializerUse(tokens, i, body, close, nil, importMethods) {
						continue
					}
					if isCallbackArgumentUse(tokens, i, body, close, sigs, importFuncs) {
						continue
					}
				}
				diags = appendDiag(diags, file, tok, "method expressions are not supported: "+tok.Text+"."+tokens[i+2].Text+"."+tokens[i+4].Text)
			}
			continue
		}
		if importedFunctionIndex(importFuncs, tok.Text, tokens[i+2].Text) < 0 {
			continue
		}
		if i+3 < len(tokens) && tokens[i+3].Text == "(" && tokens[i+3].Line == tokens[i+2].Line {
			continue
		}
		if functionValueBlankDiscardRHSAt(tokens, i) {
			continue
		}
		if body := enclosingFunctionBodyOpen(tokens, i); body >= 0 {
			close := findClose(tokens, body, "{", "}")
			if close < 0 {
				close = len(tokens)
			}
			if isFunctionAliasInitializerUse(tokens, i, body, close, nil, importFuncs) {
				continue
			}
			if isCallbackArgumentUse(tokens, i, body, close, sigs, importFuncs) {
				continue
			}
		}
		diags = appendDiag(diags, file, tokens[i+2], "function values are not supported: "+tok.Text+"."+tokens[i+2].Text)
	}
	return diags
}

func appendPackageValueSpecTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, topValues []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue) Diagnostics {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return diags
	}
	eq := findTopLevelToken(tokens, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	if typeStart := valueSpecFunctionTypeStart(tokens, start, lhsEnd, typeNames); typeStart >= 0 {
		diags = appendDiag(diags, file, tokens[typeStart], "function values and function types are not supported")
	}
	if eq < 0 {
		return diags
	}
	if staticInterfaceAssertionVarSpecLowerable(file, tokens, start, end) {
		return diags
	}
	var names []string
	typeStart := -1
	for i := start; i < eq; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendInitializerName(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return diags
	}
	rhs := expressionRanges(tokens, eq+1, end)
	for i := 0; i < len(rhs); i++ {
		value := rhs[i]
		diags = appendNamedSliceCompositeLiteralRangeDiagnostics(diags, file, tokens, value.start, value.end, topValues, structs, typeNames, importFuncs, importValues, nil, nil, nil)
	}
	if typeStart < 0 {
		diags = appendInitializerCountDiagnostics(diags, file, tokens, names, eq, end, nil, importFuncs, nil, nil, nil)
		for i := 0; i < len(rhs); i++ {
			value := rhs[i]
			diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, topValues, structs, typeNames, importFuncs, importValues, nil, nil, nil)
		}
		return diags
	}
	want := typeTextInRange(tokens, typeStart, eq)
	if want == "" {
		return diags
	}
	for i := 0; i < len(names) && i < len(rhs); i++ {
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, rhs[i].start, rhs[i].end, topValues, structs, typeNames, importFuncs, importValues, nil, nil, nil)
		got := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, topValues, structs, typeNames, importFuncs, importValues, nil, nil, nil)
		if got == "" || expressionAssignableToType(tokens, rhs[i].start, rhs[i].end, want, got, typeNames) {
			continue
		}
		diags = appendDiag(diags, file, tokens[rhs[i].start], "initializer type mismatch: "+names[i]+" has "+want+", got "+got)
	}
	return diags
}

func valueSpecFunctionTypeStart(tokens []scan.Token, start int, end int, typeNames []localValueType) int {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return -1
	}
	var names []string
	typeStart := -1
	for i := start; i < end; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendInitializerName(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 || typeStart < 0 {
		return -1
	}
	typ := normalizeNamedType(typeTextInRange(tokens, typeStart, end), typeNames)
	if strings.HasPrefix(typ, "func") {
		return typeStart
	}
	return -1
}

func appendLocalValueDeclTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if pos+1 < limit && tokens[pos+1].Text == "(" {
		close := findClose(tokens, pos+1, "(", ")")
		if close <= pos+1 || close > limit {
			return diags
		}
		specStart := pos + 2
		for i := specStart; i <= close; i++ {
			if i == close || tokens[i].Text == ";" {
				diags = appendValueSpecInitializerTypeDiagnostics(diags, file, tokens, specStart, i, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
				specStart = i + 1
				continue
			}
			if tokens[i].Line != tokens[specStart].Line {
				diags = appendValueSpecInitializerTypeDiagnostics(diags, file, tokens, specStart, i, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
				specStart = i
			}
		}
		return diags
	}
	end := simpleStatementEnd(tokens, pos+1, limit)
	return appendValueSpecInitializerTypeDiagnostics(diags, file, tokens, pos+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendValueSpecInitializerTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, values []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return diags
	}
	eq := findTopLevelToken(tokens, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	if typeStart := valueSpecFunctionTypeStart(tokens, start, lhsEnd, typeNames); typeStart >= 0 {
		diags = appendDiag(diags, file, tokens[typeStart], "function values and function types are not supported")
	}
	if eq < 0 {
		return diags
	}
	if staticInterfaceAssertionVarSpecLowerable(file, tokens, start, end) {
		return diags
	}
	if lowerableComplexAliasComponentVarSpecLowerable(file, tokens, start, sigs) {
		return diags
	}
	var names []string
	typeStart := -1
	for i := start; i < eq; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendInitializerName(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 || typeStart < 0 {
		if len(names) > 0 && eq >= 0 {
			if !(len(names) == 2 && staticInterfaceAssertionAssignmentResultCount(file, tokens, eq+1, end) == 2) {
				diags = appendInitializerCountDiagnostics(diags, file, tokens, names, eq, end, sigs, importFuncs, localStructs, structs, importMethods)
			}
			rhs := expressionRanges(tokens, eq+1, end)
			for i := 0; i < len(rhs); i++ {
				value := rhs[i]
				if staticInterfaceAssertionExpressionLowerable(file, tokens, value.start, value.end) {
					continue
				}
				if i < len(names) && staticAliasInitializerOperand(tokens, names[i], value.start, value.end, sigs, importFuncs, values, structs, typeNames, localStructs, importMethods) {
					continue
				}
				diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, values, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			}
		}
		return diags
	}
	diags = appendInitializerCountDiagnostics(diags, file, tokens, names, eq, end, sigs, importFuncs, localStructs, structs, importMethods)
	want := typeTextInRange(tokens, typeStart, eq)
	if want == "" {
		return diags
	}
	rhs := expressionRanges(tokens, eq+1, end)
	if len(rhs) == 1 {
		resultTypes := singleCallResultTypes(tokens, rhs[0].start, rhs[0].end, sigs, importFuncs, localStructs, structs, importMethods)
		if len(resultTypes) == len(names) {
			diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, rhs[0].start, rhs[0].end, values, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			for i := 0; i < len(names); i++ {
				got := resultTypes[i]
				if got == "" || typesAssignable(want, got, typeNames) {
					continue
				}
				diags = appendDiag(diags, file, tokens[rhs[0].start], "initializer type mismatch: "+names[i]+" has "+want+", got "+got)
			}
			return diags
		}
	}
	for i := 0; i < len(names) && i < len(rhs); i++ {
		if staticAliasInitializerOperand(tokens, names[i], rhs[i].start, rhs[i].end, sigs, importFuncs, values, structs, typeNames, localStructs, importMethods) {
			continue
		}
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, rhs[i].start, rhs[i].end, values, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		got := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, values, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" || expressionAssignableToType(tokens, rhs[i].start, rhs[i].end, want, got, typeNames) {
			continue
		}
		diags = appendDiag(diags, file, tokens[rhs[i].start], "initializer type mismatch: "+names[i]+" has "+want+", got "+got)
	}
	return diags
}

func appendInitializerCountDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, names []string, eq int, end int, sigs []funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod) Diagnostics {
	want := len(names)
	got := expressionListCount(tokens, eq+1, end)
	if got == 1 {
		callResults := singleCallResultCount(tokens, eq+1, end, sigs, importFuncs, localStructs, structs, importMethods)
		if callResults >= 0 {
			got = callResults
		}
		mapResults := lowerableMapLiteralCommaOkResultCount(tokens, eq+1, end)
		if want == 2 && mapResults >= 0 {
			got = mapResults
		}
	}
	if want != got {
		return appendDiag(diags, file, tokens[eq], "initializer count mismatch")
	}
	return diags
}

func functionSemanticDiagnostics(file parse.File, topNames []string, topTypes []string, topConsts []string, sigs []funcSignature, structs []structFieldSet, topValues []localValueType, topStructValues []localStructType, packages []load.Package, typeNames []localValueType, wordSize int) Diagnostics {
	var diags Diagnostics
	importNames := fileImportNames(file)
	importMethods := importedMethodsForFile(file, packages)
	importFuncs := importedFunctionsForFile(file, packages)
	importValues := importedValuesForFile(file, packages)
	allStructs := appendStructFieldSets(structs, importedStructTypesForFile(file, packages))
	importShadows := localImportShadows(file, importNames)
	tokens := file.Tokens
	decls := file.Decls
	for declIndex := 0; declIndex < len(decls); declIndex++ {
		decl := decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		body := findTokenText(tokens, start, decl.End, "{")
		if body < 0 {
			continue
		}
		close := findClose(tokens, body, "{", "}")
		if close < 0 {
			close = tokenIndexBefore(tokens, decl.End)
		}
		if close <= body {
			continue
		}
		functionTypeNames := appendLocalValueTypes(cloneLocalValueTypes(typeNames), functionLocalNamedTypeUnderlyings(tokens, body+1, close, typeNames))
		localTypeNames := functionLocalTypeDeclNames(tokens, body+1, close)
		functionTopTypes := appendStringSet(cloneStrings(topTypes), localTypeNames)
		functionTopNames := appendStringSet(cloneStrings(topNames), localTypeNames)
		functionStructs := appendStructFieldSets(allStructs, functionLocalStructTypes(tokens, body+1, close, functionTypeNames))
		locals := collectFunctionLocalNames(tokens, start, body, close)
		localStructs := cloneLocalStructTypes(topStructValues)
		localStructs = collectFunctionLocalStructTypes(tokens, start, body, close, functionStructs, importNames, localStructs)
		localTypes := cloneLocalValueTypes(topValues)
		localTypes = collectFunctionLocalValueTypes(tokens, start, body, close, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, sigs, importMethods, localTypes)
		functionAliases := collectFunctionAliases(tokens, body, close, sigs, importFuncs, localTypes, functionStructs, functionTypeNames, localStructs, importMethods)
		functionAliases = appendFunctionParamAliases(functionAliases, tokens, start, body, close, functionStructs, functionTypeNames)
		functionSigs := appendFunctionAliasSignatures(sigs, functionAliases)
		localTypes = cloneLocalValueTypes(topValues)
		localTypes = collectFunctionLocalValueTypes(tokens, start, body, close, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods, localTypes)
		sig := funcSignatureForDecl(tokens, decl, functionSigs)
		for i := body + 1; i < close; i++ {
			tok := tokens[i]
			if tok.Kind == scan.EOF {
				break
			}
			if tok.Text == "type" && startsLocalTypeDeclToken(tokens, i, body) {
				if end, ok := inertLocalUnsupportedTypeDeclEnd(tokens, i, body, close); ok {
					i = end - 1
					continue
				}
				if end, ok := localTypeDeclEnd(tokens, i, body, close); ok {
					i = end - 1
					continue
				}
			}
			if tok.Text == "var" || (tok.Kind == scan.Ident && i+1 < close && tokens[i+1].Text == ":=") {
				if _, _, discardEnd, ok := lowerableComplexVarBlankDiscardAt(file, tokens, i, functionSigs); ok {
					i = discardEnd - 1
					continue
				}
				if _, declEnd, _, ok := lowerableComplexAliasComponentAt(file, tokens, i, functionSigs); ok {
					i = declEnd - 1
					continue
				}
			}
			if tok.Text == "var" {
				if _, _, discardEnd, ok := lowerableInterfaceVarBlankDiscardAt(file, tokens, i); ok {
					i = discardEnd - 1
					continue
				}
				if info, ok := lowerableNilInterfaceVarComparisonsAt(file, tokens, i); ok {
					i = info.declEnd - 1
					continue
				}
			}
			if tok.Text == "func" && !file.IsTopLevelFuncAt(i) {
				if literal, ok := functionLiteralInfoAt(tokens, i, close, functionStructs, functionTypeNames); ok {
					if call, ok := functionLiteralDirectCallInfoAt(tokens, i, close, functionStructs, functionTypeNames); ok {
						outerLocals := sameScopeNamesBefore(tokens, start, body, i)
						diags = appendFunctionLiteralUnknownCaptureDiagnostics(diags, file, tokens, call.literal, outerLocals, localTypes)
						diags = appendFunctionLiteralDirectCallDiagnostics(diags, file, tokens, call, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
						i = call.callClose
						continue
					}
					if functionValueBlankDiscardRHSAt(tokens, i) {
						i = literal.bodyClose
						continue
					}
					if functionLiteralAliasInitializerAt(tokens, i) {
						outerLocals := sameScopeNamesBefore(tokens, start, body, i)
						diags = appendFunctionLiteralUnknownCaptureDiagnostics(diags, file, tokens, literal, outerLocals, localTypes)
						i = literal.bodyClose
						continue
					}
					if functionLiteralStaticCallbackArgumentAt(tokens, i, functionSigs) {
						outerLocals := sameScopeNamesBefore(tokens, start, body, i)
						diags = appendFunctionLiteralUnknownCaptureDiagnostics(diags, file, tokens, literal, outerLocals, localTypes)
						i = literal.bodyClose
						continue
					}
				}
			}
			if tok.Text == "{" {
				diags = appendNamedSliceCompositeLiteralOpenDiagnostics(diags, file, tokens, i, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			}
			if isSimpleStatementStart(tokens, body, i) {
				diags = appendExpressionStatementDiagnostics(diags, file, tokens, body, i, close, functionTopTypes, functionStructs)
				stmtEnd := simpleStatementEnd(tokens, i, close)
				if compositeEnd := compositeLiteralMethodCallEnd(tokens, i, close); compositeEnd > stmtEnd {
					stmtEnd = compositeEnd
				}
				diags = appendCompositeLiteralMethodCallDiagnostics(diags, file, tokens, i, stmtEnd, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			}
			if tok.Text == "return" {
				diags = appendReturnDiagnostics(diags, file, tokens, i, close, sig, functionSigs, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, importMethods)
				continue
			}
			if tok.Text == "defer" {
				diags = appendDeferDiagnostics(diags, file, tokens, body, i, close, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if tok.Text == "var" || tok.Text == "const" {
				diags = appendLocalValueDeclTypeDiagnostics(diags, file, tokens, i, close, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if tok.Text == "switch" {
				if lowerableStaticInterfaceTypeSwitchAtSwitch(file, tokens, i) {
					continue
				}
				diags = appendControlHeaderCompositeLiteralSyntaxDiagnostics(diags, file, tokens, i, close, functionStructs, functionTypeNames)
				diags = appendSwitchDiagnostics(diags, file, tokens, i, close, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if tok.Text == "if" || tok.Text == "for" {
				diags = appendControlHeaderCompositeLiteralSyntaxDiagnostics(diags, file, tokens, i, close, functionStructs, functionTypeNames)
				diags = appendConditionDiagnostics(diags, file, tokens, i, close, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if tok.Text == "range" {
				diags = appendRangeOperandDiagnostics(diags, file, tokens, i, close, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if isIncDecAt(tokens, i, close) {
				diags = appendIncDecDiagnostics(diags, file, tokens, start, body, i, close, localTypes, topConsts, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if isCompoundAssignmentAt(tokens, i, close) {
				diags = appendCompoundAssignmentDiagnostics(diags, file, tokens, start, body, i, close, localTypes, topConsts, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
				continue
			}
			if tok.Text == "=" || tok.Text == ":=" {
				if isCompoundAssignmentEquals(tokens, i) {
					continue
				}
				if tok.Text == "=" && anonymousStructAssignmentInLocalVarDecl(tokens, body, i) {
					continue
				}
				if tok.Text == "=" && staticInterfaceAssertionAssignmentInLocalVarDecl(file, tokens, body, i) {
					continue
				}
				if i+1 < close && tokens[i+1].Text == "range" {
					diags = appendRangeAssignmentDiagnostics(diags, file, tokens, start, body, i, close, localTypes, topConsts, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
					continue
				}
				if isInsideGroupedDeclaration(tokens, i) {
					continue
				}
				stmtStart := simpleStatementStart(tokens, body, i)
				if stmtStart < i && (tokens[stmtStart].Text == "var" || tokens[stmtStart].Text == "const") {
					continue
				}
				if tok.Text == ":=" {
					diags = appendShortDeclDiagnostics(diags, file, tokens, start, body, i)
				}
				diags = appendAssignmentDiagnostics(diags, file, tokens, start, body, i, close, functionSigs, localTypes, topConsts, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, importMethods)
				diags = appendFunctionAliasAssignmentDiagnostics(diags, file, tokens, body, i, functionAliases)
				continue
			}
			if tok.Text == "*" {
				diags = appendLocalStructDerefDiagnostics(diags, file, tokens, i, localStructs)
				diags = appendLocalValueDerefDiagnostics(diags, file, tokens, i, localTypes, localStructs)
				continue
			}
			if tok.Kind == scan.Ident || tok.Text == "[" {
				diags = appendIndexedReceiverMethodCallDiagnostics(diags, file, tokens, i, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
			}
			if tok.Kind != scan.Ident {
				continue
			}
			if i+2 < close && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident {
				if info, ok := localStructSelectorInfo(localStructs, localTypes, functionStructs, functionTypeNames, tok.Text, i); ok {
					if i+3 < close && tokens[i+3].Text == "(" {
						diags = appendMethodCallDiagnostics(diags, file, tokens, i, info, functionSigs, importMethods, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs)
						continue
					}
					if isLocalMethodValueUse(tokens, i, info, functionStructs, functionSigs, importMethods) {
						if isMethodAliasInitializerUse(tokens, i, body, close, localTypes, functionStructs, functionTypeNames, localStructs, functionSigs, importMethods) {
							continue
						}
						if isCallbackArgumentUse(tokens, i, body, close, functionSigs, importFuncs) {
							continue
						}
						if functionValueBlankDiscardRHSAt(tokens, i) {
							continue
						}
						diags = appendDiag(diags, file, tokens[i+2], "method values are not supported: "+tok.Text+"."+tokens[i+2].Text)
						continue
					}
					diags = appendLocalStructSelectorDiagnostics(diags, file, tokens, i, info, functionStructs)
					continue
				}
				if i+3 < close && tokens[i+3].Text == "(" && containsString(importNames, tok.Text) && !isLocalShadowAt(importShadows, tok.Text, int(tok.Start)) {
					diags = appendImportedCallDiagnostics(diags, file, tokens, i, importFuncs, importValues, localTypes, functionStructs, functionTypeNames, localStructs, functionSigs, importMethods, wordSize)
					continue
				}
				if i+3 < close && tokens[i+3].Text == "(" {
					if methodSig, ok := namedReceiverMethodSignatureAt(tokens, i, localTypes, functionSigs, importMethods); ok {
						methodClose := findClose(tokens, i+3, "(", ")")
						if methodClose <= i+3 {
							continue
						}
						argCount := expressionListCount(tokens, i+4, methodClose)
						display := tok.Text + "." + tokens[i+2].Text
						if (!methodSig.variadic && argCount != methodSig.params) || (methodSig.variadic && argCount < methodSig.params-1) {
							diags = appendDiag(diags, file, tokens[i+2], "argument count mismatch in call to "+display)
							continue
						}
						diags = appendCallArgumentTypeDiagnostics(diags, file, tokens, display, i+4, methodClose, methodSig, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
						continue
					}
				}
				var handled bool
				diags, handled = appendImportedValueSelectorDiagnostics(diags, file, tokens, i, importValues, functionStructs, functionTypeNames)
				if handled {
					continue
				}
				diags = appendLocalValueSelectorDiagnostics(diags, file, tokens, i, localTypes, functionStructs, functionTypeNames)
			}
			if i+1 < close && tokens[i+1].Text == "[" && localStructTypeName(localStructs, tok.Text) != "" {
				diags = appendLocalStructIndexDiagnostics(diags, file, tokens, i, localStructs)
				continue
			}
			if i+1 < close && tokens[i+1].Text == "[" {
				diags = appendLocalValueIndexDiagnostics(diags, file, tokens, i, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
			}
			if i+3 < close && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident && tokens[i+3].Text == "(" {
				diags = appendMethodExpressionCallDiagnostics(diags, file, tokens, i, functionTopTypes, importNames, importShadows, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
			} else if i+5 < close && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident && tokens[i+3].Text == "." && tokens[i+4].Kind == scan.Ident && tokens[i+5].Text == "(" {
				diags = appendMethodExpressionCallDiagnostics(diags, file, tokens, i, functionTopTypes, importNames, importShadows, localTypes, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, functionSigs, importMethods)
			}
			if isMethodExpressionUse(tokens, i, functionTopTypes, importNames) {
				if isMethodExpressionAliasInitializerUse(tokens, i, body, close, functionSigs, importMethods) {
					continue
				}
				if methodExpressionAssignedToFunctionAlias(tokens, i, body, close, functionAliases) {
					continue
				}
				if isCallbackArgumentUse(tokens, i, body, close, functionSigs, importFuncs) {
					continue
				}
				diags = appendDiag(diags, file, tok, "method expressions are not supported: "+tok.Text+"."+tokens[i+2].Text)
				continue
			}
			if i+1 < close && tokens[i+1].Text == "(" && tokens[i+1].Line == tok.Line && !isSelectorMember(tokens, i) && !isKeyword(tok.Text) {
				diags = appendCallDiagnostics(diags, file, tokens, i, functionSigs, functionTopTypes, locals, localTypes, functionTopNames, importNames, functionStructs, functionTypeNames, importFuncs, importValues, localStructs, importMethods, wordSize)
				continue
			}
			if isFunctionAliasValueUse(tokens, i, functionAliases) {
				if isCallbackArgumentUse(tokens, i, body, close, functionSigs, importFuncs) {
					continue
				}
				diags = appendDiag(diags, file, tok, "function values are not supported: "+tok.Text)
				continue
			}
			if isFunctionAliasInitializerUse(tokens, i, body, close, sigs, importFuncs) {
				continue
			}
			if lowerableStaticInterfaceAssertionVarContainingToken(file, tokens, i) {
				continue
			}
			if lowerableNilInterfaceVarComparisonContainingToken(file, tokens, i) {
				continue
			}
			if isFunctionValueUse(tokens, i, functionSigs, locals) {
				if isCallbackArgumentUse(tokens, i, body, close, functionSigs, importFuncs) {
					continue
				}
				if functionValueBlankDiscardRHSAt(tokens, i) {
					continue
				}
				diags = appendDiag(diags, file, tok, "function values are not supported: "+tok.Text)
				continue
			}
			if shouldCheckUndefinedIdent(tokens, i) && !knownIdentifier(tok.Text, functionTopNames, locals, importNames) && !dotImportedIdentifierKnown(tok.Text, importFuncs, importValues, functionTypeNames, functionStructs) {
				diags = appendDiag(diags, file, tok, "undefined identifier: "+tok.Text)
			}
		}
	}
	return diags
}

func appendExpressionStatementDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, body int, start int, limit int, topTypes []string, structs []structFieldSet) Diagnostics {
	if start >= limit || !canStartExpressionStatementCheck(tokens, body, start, topTypes, structs) {
		return diags
	}
	if tokenInsideLowerableAnonymousStructType(file, start) {
		return diags
	}
	end := simpleStatementEnd(tokens, start, limit)
	if compositeEnd := compositeLiteralMethodCallEnd(tokens, start, limit); compositeEnd > end {
		end = compositeEnd
	}
	if start >= end || hasTopLevelStatementSeparator(tokens, start, end) || hasAssignmentLikeOperator(tokens, start, end) {
		return diags
	}
	if isLabelStatement(tokens, start, end) || isCallExpressionStatement(tokens, start, end, topTypes) {
		return diags
	}
	return appendDiag(diags, file, tokens[start], "expression statement must be a call")
}

func tokenInsideLowerableAnonymousStructType(file parse.File, pos int) bool {
	toks := file.Tokens
	if pos <= 0 || pos >= len(toks) {
		return false
	}
	body := functionBodyOpenContainingToken(file, pos)
	if body < 0 {
		return false
	}
	for i := pos - 1; i > body; i-- {
		if toks[i].Text == "struct" && i+1 < len(toks) && toks[i+1].Text == "{" {
			close := findClose(toks, i+1, "{", "}")
			return close > pos && frontendLowerableAnonymousStructType(file, i)
		}
		if toks[i].Text == ";" || toks[i].Text == "}" || toks[i].Text == ")" {
			return false
		}
	}
	return false
}

func canStartExpressionStatementCheck(tokens []scan.Token, body int, start int, topTypes []string, structs []structFieldSet) bool {
	tok := tokens[start]
	if tok.Text == ";" || tok.Text == "}" || tok.Text == "{" || tok.Text == "," || tok.Text == ":" || tok.Text == "." || tok.Text == ")" || tok.Text == "]" {
		return false
	}
	if isStatementKeyword(tok.Text) {
		return false
	}
	if isSelectorMember(tokens, start) {
		return false
	}
	if isExpressionContinuationOnLine(tokens, start) || isInControlHeader(tokens, start) || isInsideGroupedDeclaration(tokens, start) || isInsideCompositeLiteral(tokens, body, start, topTypes, structs) {
		return false
	}
	return true
}

func isExpressionContinuationOnLine(tokens []scan.Token, pos int) bool {
	line := tokens[pos].Line
	for i := pos - 1; i >= 0 && tokens[i].Line == line; i-- {
		switch tokens[i].Text {
		case "return", "=", ":=", ",", "(", "[", ":", "&":
			return true
		case "{":
			continue
		}
	}
	return false
}

func isStatementKeyword(text string) bool {
	switch text {
	case "break", "case", "const", "continue", "default", "defer", "else", "fallthrough", "for", "go", "goto", "if", "range", "return", "select", "switch", "type", "var":
		return true
	}
	return false
}

func isLabelStatement(tokens []scan.Token, start int, end int) bool {
	return start+1 < end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == ":"
}

func isCallExpressionStatement(tokens []scan.Token, start int, end int, topTypes []string) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return true
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return isCallExpressionStatement(tokens, start+1, close, topTypes)
		}
	}
	if startsSliceConversionExpression(tokens, start, end, topTypes) {
		return false
	}
	if start+3 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "(" {
		close := findClose(tokens, start+1, "(", ")")
		if close != end-1 {
			return false
		}
		if _, ok := conversionTargetAt(tokens, start, topTypes); ok {
			return false
		}
		return !isValueOnlyBuiltinStatement(tokens[start].Text)
	}
	if start+5 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "(" {
		close := findClose(tokens, start+3, "(", ")")
		return close == end-1
	}
	if _, ok := compositeLiteralMethodCallInfoAt(tokens, start, end); ok {
		return true
	}
	if indexedReceiverMethodCallSyntaxAt(tokens, start, end) {
		return true
	}
	return false
}

func isValueOnlyBuiltinStatement(name string) bool {
	switch name {
	case "append", "len", "make":
		return true
	}
	return isBuiltinTypeName(name)
}

func startsSliceConversionExpression(tokens []scan.Token, start int, end int, topTypes []string) bool {
	if start+5 > end || tokens[start].Text != "[" || tokens[start+1].Text != "]" || tokens[start+2].Kind != scan.Ident || tokens[start+3].Text != "(" {
		return false
	}
	name := tokens[start+2].Text
	if !isBuiltinTypeName(name) && !containsString(topTypes, name) {
		return false
	}
	close := findClose(tokens, start+3, "(", ")")
	return close == end-1
}

func hasTopLevelStatementSeparator(tokens []scan.Token, start int, end int) bool {
	return findTopLevelToken(tokens, start, end, ",") >= 0
}

func hasAssignmentLikeOperator(tokens []scan.Token, start int, end int) bool {
	if findTopLevelToken(tokens, start, end, "=") >= 0 || findTopLevelToken(tokens, start, end, ":=") >= 0 {
		return true
	}
	for i := start; i < end; i++ {
		if isIncDecAt(tokens, i, end) || isCompoundAssignmentAt(tokens, i, end) {
			return true
		}
	}
	return false
}

func isInControlHeader(tokens []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		text := tokens[i].Text
		if text == "{" || text == "}" {
			return false
		}
		if text == "if" || text == "for" || text == "switch" {
			open := findTokenText(tokens, pos, maxSourcePosition(), "{")
			return open > pos
		}
	}
	return false
}

func isInsideGroupedDeclaration(tokens []scan.Token, pos int) bool {
	var opens []int
	for i := 0; i < pos; i++ {
		switch tokens[i].Text {
		case "(":
			opens = append(opens, i)
		case ")":
			if len(opens) > 0 {
				opens = opens[:len(opens)-1]
			}
		}
	}
	for i := len(opens) - 1; i >= 0; i-- {
		open := opens[i]
		if open == 0 {
			continue
		}
		prev := tokens[open-1].Text
		if prev == "var" || prev == "const" || prev == "type" || prev == "import" {
			return true
		}
	}
	return false
}

func isInsideCompositeLiteral(tokens []scan.Token, body int, pos int, topTypes []string, structs []structFieldSet) bool {
	var composites []bool
	for i := body; i < pos; i++ {
		switch tokens[i].Text {
		case "{":
			if i == body {
				composites = append(composites, false)
				continue
			}
			parentComposite := len(composites) > 0 && composites[len(composites)-1]
			composites = append(composites, parentComposite || braceStartsCompositeLiteral(tokens, i, topTypes, structs))
		case "}":
			if len(composites) > 0 {
				composites = composites[:len(composites)-1]
			}
		}
	}
	return len(composites) > 0 && composites[len(composites)-1]
}

func braceStartsCompositeLiteral(tokens []scan.Token, open int, topTypes []string, structs []structFieldSet) bool {
	if open <= 0 {
		return false
	}
	prev := tokens[open-1]
	if prev.Text == "]" {
		return true
	}
	if prev.Kind == scan.Ident {
		return isBuiltinTypeName(prev.Text) || containsString(topTypes, prev.Text) || structFieldSetIndex(structs, prev.Text) >= 0
	}
	return false
}

func appendReturnDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, limit int, sig funcSignature, sigs []funcSignature, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, importMethods []importedMethod) Diagnostics {
	start := pos + 1
	actual := 0
	end := start
	if start < limit && tokens[start].Line == tokens[pos].Line {
		end = simpleStatementEnd(tokens, start, limit)
		actual = expressionListCount(tokens, start, end)
	}
	if actual == 1 {
		callResults := singleCallResultCount(tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
		if callResults >= 0 {
			actual = callResults
		}
	}
	if actual == 0 && sig.results > 0 && len(sig.namedResults) == sig.results {
		return diags
	}
	if actual != sig.results {
		return appendDiag(diags, file, tokens[pos], "return value count mismatch")
	}
	if actual > 0 {
		diags = appendReturnTypeDiagnostics(diags, file, tokens, start, end, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func appendReturnTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, sig funcSignature, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(sig.resultTypes) == 0 {
		return diags
	}
	values := expressionRanges(tokens, start, end)
	if len(values) == 1 && len(sig.resultTypes) != 1 {
		value := values[0]
		if resultTypes := singleCallResultTypes(tokens, value.start, value.end, sigs, importFuncs, localStructs, structs, importMethods); len(resultTypes) == len(sig.resultTypes) {
			diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			for i := 0; i < len(sig.resultTypes); i++ {
				want := sig.resultTypes[i]
				got := resultTypes[i]
				if want == "" || got == "" || typesAssignable(want, got, typeNames) {
					continue
				}
				diags = appendDiag(diags, file, tokens[value.start], "return type mismatch: want "+want+", got "+got)
			}
			return diags
		}
	}
	if len(values) != len(sig.resultTypes) {
		return diags
	}
	for i := 0; i < len(values); i++ {
		want := sig.resultTypes[i]
		value := values[i]
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if want == "" || lowerableInterfaceParamRawType(want) {
			continue
		}
		got := expressionRawTypeWithCallsAndTypes(tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" || expressionAssignableToType(tokens, value.start, value.end, want, got, typeNames) {
			continue
		}
		diags = appendDiag(diags, file, tokens[value.start], "return type mismatch: want "+want+", got "+got)
	}
	return diags
}

func appendDeferDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, body int, pos int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	stmtEnd := simpleStatementEnd(tokens, pos, limit)
	open := -1
	literalCall := functionLiteralDirectCallInfo{}
	hasLiteralCall := false
	if pos+1 < stmtEnd && tokens[pos+1].Text == "func" {
		call, ok := deferFunctionLiteralDirectCallInfo(tokens, pos, stmtEnd, structs, typeNames)
		if !ok {
			return appendDiag(diags, file, tokens[pos], "defer requires a direct function call")
		}
		literalCall = call
		hasLiteralCall = true
		open = call.callOpen
	} else {
		open = deferDirectCallOpen(tokens, pos, stmtEnd)
		if open < 0 {
			return appendDiag(diags, file, tokens[pos], "defer requires a direct function call")
		}
	}
	close := findClose(tokens, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return appendDiag(diags, file, tokens[pos], "defer requires a direct function call")
	}
	for i := close + 1; i < stmtEnd; i++ {
		if tokens[i].Text != ";" {
			return appendDiag(diags, file, tokens[i], "unexpected token after deferred call")
		}
	}
	if !deferAtFunctionTopLevel(tokens, body, pos) {
		args := expressionRanges(tokens, open+1, close)
		for i := 0; i < len(args); i++ {
			arg := args[i]
			gotEnd := arg.end
			if hasVariadicExpansion(tokens, arg.start, arg.end) {
				gotEnd--
			}
			got := expressionRawTypeWithCallsAndTypes(tokens, arg.start, gotEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
			if got == "" {
				return appendDiag(diags, file, tokens[arg.start], "nested defer argument type is not supported")
			}
		}
	}
	if hasLiteralCall {
		diags = appendFunctionLiteralDirectCallDiagnostics(diags, file, tokens, literalCall, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		return diags
	}
	if pos+2 == open && staticInterfaceAssertionListContains(tokens, open+1, close) {
		index := funcSignatureIndex(sigs, tokens[pos+1].Text)
		if index >= 0 {
			diags = appendCallArgumentTypeDiagnostics(diags, file, tokens, tokens[pos+1].Text, open+1, close, sigs[index], localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
	}
	return diags
}

func deferFunctionLiteralDirectCallInfo(tokens []scan.Token, pos int, end int, structs []structFieldSet, typeNames []localValueType) (functionLiteralDirectCallInfo, bool) {
	if pos+1 >= end || tokens[pos+1].Text != "func" {
		return functionLiteralDirectCallInfo{}, false
	}
	call, ok := functionLiteralDirectCallInfoAt(tokens, pos+1, end, structs, typeNames)
	if !ok || call.literal.start != pos+1 {
		return functionLiteralDirectCallInfo{}, false
	}
	return call, true
}

func staticInterfaceAssertionListContains(tokens []scan.Token, start int, end int) bool {
	args := expressionRanges(tokens, start, end)
	for i := 0; i < len(args); i++ {
		argStart, argEnd := trimExpressionRange(tokens, args[i].start, args[i].end)
		if staticInterfaceAssertionExpressionType(tokens, argStart, argEnd) != "" {
			return true
		}
	}
	return false
}

func deferAtFunctionTopLevel(tokens []scan.Token, body int, pos int) bool {
	depth := 0
	for i := body + 1; i < pos && i < len(tokens); i++ {
		if tokens[i].Text == "{" {
			depth++
			continue
		}
		if tokens[i].Text == "}" && depth > 0 {
			depth--
		}
	}
	return depth == 0
}

func deferIsInsideLoop(tokens []scan.Token, body int, pos int) bool {
	for i := pos - 1; i > body; i-- {
		if tokens[i].Text != "{" {
			continue
		}
		owner := blockOwnerKeyword(tokens, i)
		if owner == "for" {
			return true
		}
	}
	return false
}

func deferDirectCallOpen(tokens []scan.Token, pos int, end int) int {
	i := pos + 1
	if i >= end || i >= len(tokens) || tokens[i].Kind != scan.Ident {
		return -1
	}
	i++
	for i+1 < end && tokens[i].Text == "." && tokens[i+1].Kind == scan.Ident {
		i += 2
	}
	if i < end && tokens[i].Text == "(" {
		return i
	}
	return -1
}

func appendRangeOperandDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, rangePos int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start := rangePos + 1
	end := rangeOperandEnd(tokens, start, limit)
	if start >= end {
		return diags
	}
	if pureMapRangeExpressionSupported(tokens, start, end) || pureMapAliasExpressionSupported(tokens, start, end) {
		return diags
	}
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ == "string" {
		return diags
	}
	if typ == "" || typeSupportsIndex(typ) {
		return diags
	}
	return appendDiag(diags, file, tokens[start], "cannot range over: "+typ)
}

func appendRangeAssignmentDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, assign int, limit int, localTypes []localValueType, topConsts []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if assign+1 >= limit || tokens[assign+1].Text != "range" {
		return diags
	}
	start := simpleStatementStart(tokens, body, assign)
	for start < assign && tokens[start].Text == "for" {
		start++
	}
	lhs := expressionRanges(tokens, start, assign)
	diags = appendAssignmentTargetDiagnostics(diags, file, tokens, funcStart, body, start, assign, localTypes, topConsts, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if len(lhs) == 0 || len(lhs) > 2 {
		return diags
	}
	operandStart := assign + 2
	operandEnd := rangeOperandEnd(tokens, operandStart, limit)
	if operandStart >= operandEnd {
		return diags
	}
	if keyType, valueType, ok := pureMapAliasOrDirectRangeExpressionTypes(tokens, operandStart, operandEnd); ok {
		if !blankIdentifierExpression(tokens, lhs[0].start, lhs[0].end) {
			diags = appendRangeAssignmentTargetDiagnostic(diags, file, tokens, lhs[0], keyType, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		if len(lhs) < 2 {
			return diags
		}
		if !blankIdentifierExpression(tokens, lhs[1].start, lhs[1].end) {
			return appendRangeAssignmentTargetDiagnostic(diags, file, tokens, lhs[1], valueType, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		return diags
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, operandStart, operandEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ == "" {
		return diags
	}
	diags = appendRangeAssignmentTargetDiagnostic(diags, file, tokens, lhs[0], "int", localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if len(lhs) < 2 {
		return diags
	}
	elem := rangeValueType(typ)
	if elem == "" {
		return diags
	}
	return appendRangeAssignmentTargetDiagnostic(diags, file, tokens, lhs[1], elem, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendRangeAssignmentTargetDiagnostic(diags Diagnostics, file parse.File, tokens []scan.Token, target expressionRange, got string, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	want := expressionRawTypeWithCallsAndTypes(tokens, target.start, target.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if rangeTypedValueAssignable(want, got, typeNames) {
		return diags
	}
	return appendDiag(diags, file, tokens[target.start], "assignment type mismatch: "+expressionDisplayText(tokens, target.start, target.end)+" has "+want+", got "+got)
}

func expressionDisplayText(tokens []scan.Token, start int, end int) string {
	out := ""
	for i := start; i < end; i++ {
		out = out + tokens[i].Text
	}
	return out
}

func rangeOperandIsStringLiteral(tokens []scan.Token, start int, end int) bool {
	_, ok := rangeOperandStringLiteral(tokens, start, end)
	return ok
}

func rangeOperandIsASCIIStringLiteral(tokens []scan.Token, start int, end int) bool {
	value, ok := rangeOperandStringLiteral(tokens, start, end)
	if !ok {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] >= 128 {
			return false
		}
	}
	return true
}

func rangeOperandStringLiteral(tokens []scan.Token, start int, end int) (string, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return "", false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return rangeOperandStringLiteral(tokens, start+1, close)
		}
	}
	if start+1 != end || tokens[start].Kind != scan.String {
		return "", false
	}
	value, err := scan.UnquoteString(tokens[start].Text)
	if err != nil {
		return "", false
	}
	return value, true
}

func appendConditionDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start, end, ok := conditionExpressionRange(tokens, pos, limit)
	if !ok || start >= end {
		return diags
	}
	before := len(diags)
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	if len(diags) > before {
		return diags
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ == "" || typ == "bool" {
		return diags
	}
	return appendDiag(diags, file, tokens[start], "condition must be bool, got "+typ)
}

func appendControlHeaderCompositeLiteralSyntaxDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, limit int, structs []structFieldSet, typeNames []localValueType) Diagnostics {
	if pos < 0 || pos >= limit {
		return diags
	}
	keyword := tokens[pos].Text
	if keyword != "if" && keyword != "for" && keyword != "switch" {
		return diags
	}
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "{" {
			typeStart := controlHeaderCompositeLiteralTypeStart(tokens, i)
			if controlHeaderCompositeLiteralTypeKnown(tokens, typeStart, i, structs, typeNames) {
				close := findClose(tokens, i, "{", "}")
				if close < 0 {
					return diags
				}
				if controlHeaderCompositeLiteralRequiresParens(tokens, close, limit) {
					diags = appendDiag(diags, file, tokens[typeStart], "composite literal in control header must be parenthesized")
				}
				i = close
				continue
			}
			return diags
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return diags
}

func controlHeaderCompositeLiteralTypeStart(tokens []scan.Token, open int) int {
	if open <= 0 || open >= len(tokens) || tokens[open].Text != "{" || tokens[open-1].Kind != scan.Ident {
		return -1
	}
	start := open - 1
	if start >= 2 && tokens[start-1].Text == "." && tokens[start-2].Kind == scan.Ident {
		start = start - 2
	}
	if start > 0 && tokens[start-1].Text == "]" {
		return -1
	}
	return start
}

func controlHeaderCompositeLiteralTypeKnown(tokens []scan.Token, start int, open int, structs []structFieldSet, typeNames []localValueType) bool {
	if start < 0 || open <= start {
		return false
	}
	raw := typeTextInRange(tokens, start, open)
	if raw == "" {
		return false
	}
	typ := normalizeNamedType(raw, typeNames)
	return localValueTypeName(typeNames, raw) != "" || structFieldSetIndex(structs, raw) >= 0 || structFieldSetIndex(structs, typ) >= 0
}

func controlHeaderCompositeLiteralRequiresParens(tokens []scan.Token, close int, limit int) bool {
	next := close + 1
	if next >= limit {
		return true
	}
	switch tokens[next].Text {
	case ".", "[", "(":
		return false
	}
	return true
}

func appendSwitchDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	tagStart, tagEnd, hasTag, ok := switchTagExpressionRange(tokens, pos, limit)
	if !ok {
		return diags
	}
	tagType := ""
	if hasTag {
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, tagStart, tagEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, tagStart, tagEnd, sigs, importFuncs, localStructs, structs, importMethods)
		tagType = expressionSimpleTypeWithCallsAndTypes(tokens, tagStart, tagEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if tagType != "" && !switchTagTypeSupported(tagType) {
			diags = appendDiag(diags, file, tokens[tagStart], "switch expression type is not supported: "+tagType)
		}
	}
	open := controlBodyOpen(tokens, pos+1, limit)
	if open < 0 {
		return diags
	}
	close := findClose(tokens, open, "{", "}")
	if close < 0 || close > limit {
		close = limit
	}
	paren := 0
	brack := 0
	brace := 0
	for i := open + 1; i < close; i++ {
		if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "case" {
			colon := caseClauseColon(tokens, i+1, close)
			if colon < 0 {
				continue
			}
			values := expressionRanges(tokens, i+1, colon)
			for valueIndex := 0; valueIndex < len(values); valueIndex++ {
				value := values[valueIndex]
				diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
				diags = appendSingleValueCallDiagnostics(diags, file, tokens, value.start, value.end, sigs, importFuncs, localStructs, structs, importMethods)
				caseType := expressionSimpleTypeWithCallsAndTypes(tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
				if caseType == "" {
					continue
				}
				if !hasTag {
					if caseType != "bool" {
						diags = appendDiag(diags, file, tokens[value.start], "switch case must be bool, got "+caseType)
					}
					continue
				}
				if tagType != "" && !comparableOperandTypes(tagType, caseType) {
					diags = appendDiag(diags, file, tokens[value.start], "switch case type mismatch: switch has "+tagType+", case has "+caseType)
				}
			}
			i = colon
			continue
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return diags
}

func switchTagExpressionRange(tokens []scan.Token, pos int, limit int) (int, int, bool, bool) {
	if pos < 0 || pos >= limit || tokens[pos].Text != "switch" {
		return 0, 0, false, false
	}
	open := controlBodyOpen(tokens, pos+1, limit)
	if open < 0 {
		return 0, 0, false, false
	}
	start := pos + 1
	end := open
	semi := findTopLevelToken(tokens, start, open, ";")
	if semi >= 0 {
		start = semi + 1
	}
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return 0, 0, false, true
	}
	return start, end, true, true
}

func caseClauseColon(tokens []scan.Token, start int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if tokens[i].Text == ":" {
				return i
			}
			if tokens[i].Text == "case" || tokens[i].Text == "default" || tokens[i].Text == "}" {
				return -1
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func switchTagTypeSupported(typ string) bool {
	return typ == "bool" || typ == "string" || isIntegerTypeName(typ)
}

func conditionExpressionRange(tokens []scan.Token, pos int, limit int) (int, int, bool) {
	if pos < 0 || pos >= limit {
		return 0, 0, false
	}
	keyword := tokens[pos].Text
	if keyword != "if" && keyword != "for" {
		return 0, 0, false
	}
	open := controlBodyOpen(tokens, pos+1, limit)
	if open < 0 {
		return 0, 0, false
	}
	if keyword == "for" && pos+1 == open {
		return 0, 0, false
	}
	if keyword == "for" && pos+2 < open && tokens[pos+2].Text == "range" {
		return 0, 0, false
	}
	if keyword == "for" && pos+3 < open && tokens[pos+3].Text == "range" {
		return 0, 0, false
	}
	start := pos + 1
	end := open
	firstSemi := findTopLevelToken(tokens, start, open, ";")
	if firstSemi >= 0 {
		secondSemi := findTopLevelToken(tokens, firstSemi+1, open, ";")
		if keyword == "for" {
			if secondSemi >= 0 {
				start = firstSemi + 1
				end = secondSemi
			} else {
				return 0, 0, false
			}
		} else {
			start = firstSemi + 1
			end = open
		}
	}
	start, end = trimExpressionRange(tokens, start, end)
	if keyword == "for" && start < end && tokens[start].Text == "range" {
		return 0, 0, false
	}
	return start, end, start < end
}

func controlBodyOpen(tokens []scan.Token, start int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "{" {
			if headerCompositeLiteralOpen(tokens, i) {
				close := findClose(tokens, i, "{", "}")
				if close < 0 {
					return i
				}
				i = close
				continue
			}
			return i
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func rangeOperandEnd(tokens []scan.Token, start int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if tokens[i].Text == "{" || tokens[i].Text == ";" {
				if tokens[i].Text == "{" && headerCompositeLiteralOpen(tokens, i) {
					close := findClose(tokens, i, "{", "}")
					if close < 0 {
						return i
					}
					i = close
					continue
				}
				return i
			}
			if i > start && tokens[i].Line != tokens[start].Line {
				return i
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return limit
}

func headerCompositeLiteralOpen(tokens []scan.Token, open int) bool {
	if open <= 0 || tokens[open].Text != "{" {
		return false
	}
	prev := open - 1
	if tokens[prev].Kind == scan.Ident && (prev == 0 || tokens[prev-1].Text != "]") {
		return compositeLiteralOpenContinuesHeader(tokens, open)
	}
	brackClose := prev
	if tokens[brackClose].Kind == scan.Ident && brackClose > 0 {
		brackClose--
	}
	if tokens[brackClose].Text != "]" {
		return false
	}
	brackOpen := findOpen(tokens, brackClose, "[", "]")
	if brackOpen < 0 {
		return false
	}
	if brackOpen > 0 && tokens[brackOpen-1].Text == "map" {
		return true
	}
	return brackOpen+1 == brackClose
}

func compositeLiteralOpenContinuesHeader(tokens []scan.Token, open int) bool {
	close := findClose(tokens, open, "{", "}")
	if close < 0 || close+1 >= len(tokens) {
		return false
	}
	next := tokens[close+1].Text
	if next == "{" || next == "." || next == "[" || next == "(" || next == ";" || next == "," {
		return true
	}
	return isBinaryOperatorText(next)
}

func compositeLiteralOpenFollowedByBlock(tokens []scan.Token, open int) bool {
	close := findClose(tokens, open, "{", "}")
	if close < 0 || close+1 >= len(tokens) {
		return false
	}
	return tokens[close+1].Text == "{"
}

func appendAssignmentDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, assign int, limit int, sigs []funcSignature, localTypes []localValueType, topConsts []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, importMethods []importedMethod) Diagnostics {
	start := simpleStatementStart(tokens, body, assign)
	for start < assign && (tokens[start].Text == "for" || tokens[start].Text == "if" || tokens[start].Text == "switch") {
		start++
	}
	end := simpleStatementEnd(tokens, assign+1, limit)
	lhsCount := expressionListCount(tokens, start, assign)
	rhsCount := expressionListCount(tokens, assign+1, end)
	if rhsCount == 1 {
		callResults := singleCallResultCount(tokens, assign+1, end, sigs, importFuncs, localStructs, structs, importMethods)
		if callResults >= 0 {
			rhsCount = callResults
		}
		mapResults := lowerableMapLiteralCommaOkResultCount(tokens, assign+1, end)
		if lhsCount == 2 && mapResults >= 0 {
			rhsCount = mapResults
		}
		assertionResults := staticInterfaceAssertionAssignmentResultCount(file, tokens, assign+1, end)
		if lhsCount == 2 && assertionResults >= 0 {
			rhsCount = assertionResults
		}
	}
	if lhsCount != rhsCount {
		diags = appendDiag(diags, file, tokens[assign], "assignment count mismatch")
	}
	diags = appendAssignmentTargetDiagnostics(diags, file, tokens, funcStart, body, start, assign, localTypes, topConsts, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendAssignmentRHSOperandDiagnostics(diags, file, tokens, start, assign, end, sigs, localTypes, structs, typeNames, importFuncs, importValues, localStructs, importMethods)
	if tokens[assign].Text == "=" && lhsCount == rhsCount && rhsCount > 1 {
		diags = appendAssignmentMultiResultTypeDiagnostics(diags, file, tokens, start, assign, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendAssignmentExpressionListTypeDiagnostics(diags, file, tokens, start, assign, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if tokens[assign].Text == "=" && lhsCount == 1 && rhsCount == 1 {
		diags = appendAssignmentTypeDiagnostics(diags, file, tokens, start, assign, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func appendAssignmentRHSOperandDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, lhsStart int, assign int, rhsEnd int, sigs []funcSignature, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, importMethods []importedMethod) Diagnostics {
	lhs := expressionRanges(tokens, lhsStart, assign)
	rhs := expressionRanges(tokens, assign+1, rhsEnd)
	for i := 0; i < len(rhs); i++ {
		if lowerableMapLiteralCommaOkResultCount(tokens, rhs[i].start, rhs[i].end) == 2 {
			continue
		}
		if staticInterfaceAssertionExpressionLowerable(file, tokens, rhs[i].start, rhs[i].end) {
			continue
		}
		if tokens[assign].Text == ":=" && i < len(lhs) {
			name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
			if staticAliasInitializerOperand(tokens, name, rhs[i].start, rhs[i].end, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods) {
				continue
			}
		}
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, rhs[i].start, rhs[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func staticAliasInitializerOperand(tokens []scan.Token, name string, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, importMethods []importedMethod) bool {
	if name == "" || name == "_" {
		return false
	}
	_, ok := staticAliasTargetSignature(tokens, start, end, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods)
	return ok
}

func appendShortDeclDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, assign int) Diagnostics {
	start := simpleStatementStart(tokens, body, assign)
	if start < assign && (tokens[start].Text == "if" || tokens[start].Text == "for" || tokens[start].Text == "switch") {
		return diags
	}
	lhs := expressionRanges(tokens, start, assign)
	if shortDeclIntroducesNewName(tokens, lhs, sameScopeNamesBefore(tokens, funcStart, body, assign)) {
		return diags
	}
	return appendDiag(diags, file, tokens[assign], "no new variables on left side of :=")
}

func shortDeclIntroducesNewName(tokens []scan.Token, lhs []expressionRange, names []string) bool {
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
		if name != "" && name != "_" && !containsString(names, name) {
			return true
		}
	}
	return false
}

func sameScopeNamesBefore(tokens []scan.Token, funcStart int, body int, pos int) []string {
	scope := innermostDeclarationScopeStart(tokens, body, pos)
	var names []string
	if scope == body {
		names = collectSignatureLocalNames(tokens, funcStart, body, names)
		names = collectNamedResultLocalNames(tokens, funcStart, body, names)
	}
	for i := body + 1; i < pos; i++ {
		if innermostDeclarationScopeStart(tokens, body, i) != scope {
			continue
		}
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, body, i)
			if stmtStart < i && (tokens[stmtStart].Text == "if" || tokens[stmtStart].Text == "for" || tokens[stmtStart].Text == "switch") {
				continue
			}
			if pos < simpleStatementEnd(tokens, i+1, len(tokens)) {
				continue
			}
			names = collectAssignmentLeftNames(tokens, stmtStart, i, names)
			continue
		}
		if tokens[i].Text == "var" || tokens[i].Text == "const" {
			names = collectVarStatementNames(tokens, i, pos, names)
		}
	}
	return names
}

func innermostDeclarationScopeStart(tokens []scan.Token, body int, pos int) int {
	block := innermostBlockOpen(tokens, body, pos)
	if clause := innermostCaseClauseStart(tokens, block, pos); clause >= 0 {
		return clause
	}
	return block
}

func innermostCaseClauseStart(tokens []scan.Token, block int, pos int) int {
	if block < 0 || block >= len(tokens) {
		return -1
	}
	owner := blockOwnerKeyword(tokens, block)
	if owner != "switch" && owner != "select" {
		return -1
	}
	depth := 0
	clause := -1
	for i := block + 1; i < pos && i < len(tokens); i++ {
		text := tokens[i].Text
		if depth == 0 && (text == "case" || text == "default") {
			clause = i
			continue
		}
		if text == "{" {
			depth++
			continue
		}
		if text == "}" {
			if depth == 0 {
				return clause
			}
			depth--
		}
	}
	return clause
}

func innermostBlockOpen(tokens []scan.Token, body int, pos int) int {
	var opens []int
	for i := body; i < pos && i < len(tokens); i++ {
		if tokens[i].Text == "{" {
			opens = append(opens, i)
			continue
		}
		if tokens[i].Text == "}" && len(opens) > 0 {
			opens = opens[:len(opens)-1]
		}
	}
	if len(opens) == 0 {
		return body
	}
	return opens[len(opens)-1]
}

func appendIncDecDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, pos int, limit int, localTypes []localValueType, topConsts []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start := simpleStatementStart(tokens, body, pos)
	targets := expressionRanges(tokens, start, pos)
	diags = appendAssignmentTargetDiagnostics(diags, file, tokens, funcStart, body, start, pos, localTypes, topConsts, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if len(targets) != 1 {
		return appendDiag(diags, file, tokens[pos], "increment/decrement target count mismatch")
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, targets[0].start, targets[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ == "" || isNumericTypeName(typ) {
		return diags
	}
	return appendDiag(diags, file, tokens[pos], "invalid operand for "+incDecTextAt(tokens, pos)+": "+typ)
}

func isIncDecAt(tokens []scan.Token, pos int, limit int) bool {
	if pos+1 >= limit {
		return false
	}
	if int(tokens[pos].End) != int(tokens[pos+1].Start) {
		return false
	}
	return (tokens[pos].Text == "+" && tokens[pos+1].Text == "+") || (tokens[pos].Text == "-" && tokens[pos+1].Text == "-")
}

func incDecTextAt(tokens []scan.Token, pos int) string {
	if !isIncDecAt(tokens, pos, len(tokens)) {
		return ""
	}
	return tokens[pos].Text + tokens[pos+1].Text
}

func appendCompoundAssignmentDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, assign int, limit int, localTypes []localValueType, topConsts []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start := simpleStatementStart(tokens, body, assign)
	rhsStart := assign + 2
	end := simpleStatementEnd(tokens, rhsStart, limit)
	lhs := expressionRanges(tokens, start, assign)
	rhs := expressionRanges(tokens, rhsStart, end)
	diags = appendAssignmentTargetDiagnostics(diags, file, tokens, funcStart, body, start, assign, localTypes, topConsts, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendExpressionListOperandDiagnosticsWithTypes(diags, file, tokens, rhsStart, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if len(lhs) != 1 || len(rhs) != 1 {
		return appendDiag(diags, file, tokens[assign], "assignment count mismatch")
	}
	compound := compoundAssignmentTextAt(tokens, assign)
	op := compoundAssignmentBinaryOperator(compound)
	left := expressionRawTypeWithCallsAndTypes(tokens, lhs[0].start, lhs[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	right := expressionRawTypeWithCallsAndTypes(tokens, rhs[0].start, rhs[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if compound == "+=" && normalizeNamedType(left, typeNames) == "string" && normalizeNamedType(right, typeNames) == "string" {
		if stringCompoundAssignmentCanLower(tokens, start, assign, lhs[0]) {
			return diags
		}
		return appendDiag(diags, file, tokens[assign], "string compound concatenation is not supported for computed targets")
	}
	if left == "" || right == "" || binaryOperandsAssignable(tokens, lhs[0].start, lhs[0].end, rhs[0].start, rhs[0].end, op, left, right, typeNames) {
		return diags
	}
	return appendDiag(diags, file, tokens[assign], "invalid operands for "+compound+": "+left+" and "+right)
}

func stringCompoundAssignmentCanLower(tokens []scan.Token, stmtStart int, assign int, lhs expressionRange) bool {
	if stmtStart < assign && (tokens[stmtStart].Text == "for" || tokens[stmtStart].Text == "if" || tokens[stmtStart].Text == "switch") {
		return false
	}
	start, end := trimExpressionRange(tokens, lhs.start, lhs.end)
	if start >= end {
		return false
	}
	if expressionContainsCallToken(tokens, start, end) {
		return false
	}
	return isAssignableExpression(tokens, start, end)
}

func expressionContainsCallToken(tokens []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if tokens[i].Kind == scan.Ident && tokens[i+1].Text == "(" {
			return true
		}
	}
	return false
}

func isCompoundAssignmentAt(tokens []scan.Token, pos int, limit int) bool {
	if pos+1 >= limit || tokens[pos+1].Text != "=" {
		return false
	}
	if int(tokens[pos].End) != int(tokens[pos+1].Start) {
		return false
	}
	switch tokens[pos].Text {
	case "+", "-", "*", "/", "%":
		return true
	}
	return false
}

func isCompoundAssignmentEquals(tokens []scan.Token, pos int) bool {
	if pos <= 0 || tokens[pos].Text != "=" {
		return false
	}
	return isCompoundAssignmentAt(tokens, pos-1, len(tokens))
}

func compoundAssignmentTextAt(tokens []scan.Token, pos int) string {
	if !isCompoundAssignmentAt(tokens, pos, len(tokens)) {
		return ""
	}
	return tokens[pos].Text + "="
}

func compoundAssignmentBinaryOperator(text string) string {
	switch text {
	case "+=":
		return "+"
	case "-=":
		return "-"
	case "*=":
		return "*"
	case "/=":
		return "/"
	case "%=":
		return "%"
	}
	return ""
}

func appendAssignmentTargetDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, funcStart int, body int, start int, assign int, localTypes []localValueType, topConsts []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	lhs := expressionRanges(tokens, start, assign)
	for i := 0; i < len(lhs); i++ {
		first := firstExpressionToken(tokens, lhs[i].start, lhs[i].end)
		if first >= 0 && !isAssignableExpression(tokens, lhs[i].start, lhs[i].end) {
			diags = appendDiag(diags, file, tokens[first], "cannot assign to non-addressable expression")
			continue
		}
		if first >= 0 && expressionIsStringIndex(tokens, lhs[i].start, lhs[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods) {
			diags = appendDiag(diags, file, tokens[first], "cannot assign to string index")
			continue
		}
		if first >= 0 {
			name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
			if tokens[assign].Text != ":=" && assignmentTargetIsConstant(tokens, funcStart, body, lhs[i].start, name, topConsts) {
				diags = appendDiag(diags, file, tokens[first], "cannot assign to constant: "+name)
			}
		}
	}
	return diags
}

func assignmentTargetIsConstant(tokens []scan.Token, funcStart int, body int, pos int, name string, topConsts []string) bool {
	if name == "" || name == "_" {
		return false
	}
	found, isConst := localDeclarationBefore(tokens, funcStart, body, pos, name)
	if found {
		return isConst
	}
	return containsString(topConsts, name)
}

func localDeclarationBefore(tokens []scan.Token, funcStart int, body int, pos int, name string) (bool, bool) {
	var signatureNames []string
	signatureNames = collectSignatureLocalNames(tokens, funcStart, body, signatureNames)
	signatureNames = collectNamedResultLocalNames(tokens, funcStart, body, signatureNames)
	if containsString(signatureNames, name) {
		return true, false
	}
	found := false
	isConst := false
	for i := body + 1; i < pos && i < len(tokens); i++ {
		if !declarationScopeContains(tokens, body, i, pos) {
			continue
		}
		if tokens[i].Text == "var" || tokens[i].Text == "const" {
			var names []string
			names = collectVarStatementNames(tokens, i, pos, names)
			if containsString(names, name) {
				found = true
				isConst = tokens[i].Text == "const"
			}
			continue
		}
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, body, i)
			var names []string
			names = collectAssignmentLeftNames(tokens, stmtStart, i, names)
			if containsString(names, name) {
				found = true
				isConst = false
			}
		}
	}
	return found, isConst
}

func declarationScopeContains(tokens []scan.Token, body int, decl int, pos int) bool {
	start := innermostDeclarationScopeStart(tokens, body, decl)
	if start < 0 {
		return false
	}
	end := declarationScopeEnd(tokens, body, decl, len(tokens))
	return pos >= start && (end < 0 || pos < end)
}

func declarationScopeEnd(tokens []scan.Token, body int, decl int, limit int) int {
	stmtStart := simpleStatementStart(tokens, body, decl)
	if stmtStart < decl && (tokens[stmtStart].Text == "if" || tokens[stmtStart].Text == "for" || tokens[stmtStart].Text == "switch") {
		if end := controlStatementScopeEnd(tokens, stmtStart, limit); end >= 0 {
			return end
		}
	}
	start := innermostDeclarationScopeStart(tokens, body, decl)
	if start < 0 {
		return limit
	}
	if start == body {
		return limit
	}
	if tokens[start].Text == "case" || tokens[start].Text == "default" {
		return caseClauseScopeEnd(tokens, start, limit)
	}
	close := findClose(tokens, start, "{", "}")
	if close >= 0 {
		return close
	}
	return limit
}

func controlStatementScopeEnd(tokens []scan.Token, start int, limit int) int {
	open := controlStatementBodyOpen(tokens, start, limit)
	if open < 0 {
		return limit
	}
	close := findClose(tokens, open, "{", "}")
	if close < 0 {
		return limit
	}
	return elseChainScopeEnd(tokens, close, limit)
}

func controlStatementBodyOpen(tokens []scan.Token, start int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < limit && i < len(tokens); i++ {
		if tokens[i].Text == "{" && paren == 0 && brack == 0 && brace == 0 {
			return i
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func elseChainScopeEnd(tokens []scan.Token, close int, limit int) int {
	end := close
	next := close + 1
	for next < limit && next < len(tokens) && tokens[next].Text == ";" {
		next++
	}
	if next >= limit || next >= len(tokens) || tokens[next].Text != "else" {
		return end
	}
	if next+1 >= limit || next+1 >= len(tokens) {
		return end
	}
	if tokens[next+1].Text == "if" {
		return controlStatementScopeEnd(tokens, next+1, limit)
	}
	if tokens[next+1].Text == "{" {
		elseClose := findClose(tokens, next+1, "{", "}")
		if elseClose >= 0 {
			return elseClose
		}
	}
	return end
}

func caseClauseScopeEnd(tokens []scan.Token, start int, limit int) int {
	block := innermostBlockOpen(tokens, 0, start)
	if block < 0 {
		return limit
	}
	depth := 0
	for i := start + 1; i < limit && i < len(tokens); i++ {
		text := tokens[i].Text
		if depth == 0 && (text == "case" || text == "default") {
			return i
		}
		if text == "{" {
			depth++
			continue
		}
		if text == "}" {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return limit
}

func isAssignableExpression(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return isAssignableExpression(tokens, start+1, close)
		}
	}
	if topLevelBinaryOperator(tokens, start, end) >= 0 {
		return false
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		text := tokens[start].Text
		return text != "true" && text != "false" && text != "nil"
	}
	if startsUnaryDeref(tokens, start) && start+1 < end {
		return true
	}
	if end > start && tokens[end-1].Text == "]" {
		open := findOpen(tokens, end-1, "[", "]")
		if open > start && isAssignableExpression(tokens, start, open) {
			if findTopLevelToken(tokens, open+1, end-1, ":") >= 0 {
				return false
			}
			return true
		}
	}
	if end >= start+3 && tokens[end-2].Text == "." && tokens[end-1].Kind == scan.Ident {
		return isAssignableExpression(tokens, start, end-2)
	}
	return false
}

func addressableExpression(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return addressableExpression(tokens, start+1, close)
		}
	}
	if compositeLiteralType(tokens, start, end) != "" {
		return true
	}
	if topLevelBinaryOperator(tokens, start, end) >= 0 {
		return false
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		text := tokens[start].Text
		return text != "true" && text != "false" && text != "nil"
	}
	if startsUnaryDeref(tokens, start) && start+1 < end {
		return true
	}
	if end > start && tokens[end-1].Text == "]" {
		open := findOpen(tokens, end-1, "[", "]")
		if open > start && addressableExpression(tokens, start, open) {
			if findTopLevelToken(tokens, open+1, end-1, ":") >= 0 {
				return false
			}
			return true
		}
	}
	if end >= start+3 && tokens[end-2].Text == "." && tokens[end-1].Kind == scan.Ident {
		return addressableExpression(tokens, start, end-2)
	}
	return false
}

func expressionIsStringIndex(tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionIsStringIndex(tokens, start+1, close, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
	}
	if end <= start || tokens[end-1].Text != "]" {
		return false
	}
	open := findOpen(tokens, end-1, "[", "]")
	if open <= start {
		return false
	}
	if findTopLevelToken(tokens, open+1, end-1, ":") >= 0 {
		return false
	}
	base := expressionSimpleTypeWithCallsAndTypes(tokens, start, open, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	return base == "string"
}

func appendAssignmentMultiResultTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, lhsStart int, assign int, rhsEnd int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	resultTypes := singleCallResultTypes(tokens, assign+1, rhsEnd, sigs, importFuncs, localStructs, structs, importMethods)
	if len(resultTypes) == 0 {
		return diags
	}
	lhs := expressionRanges(tokens, lhsStart, assign)
	if len(lhs) != len(resultTypes) {
		return diags
	}
	for i := 0; i < len(lhs); i++ {
		got := resultTypes[i]
		diags = appendAssignmentTargetTypeMismatchDiagnostic(diags, file, tokens, lhs[i], got, -1, -1, assign, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func appendAssignmentExpressionListTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, lhsStart int, assign int, rhsEnd int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	lhs := expressionRanges(tokens, lhsStart, assign)
	rhs := expressionRanges(tokens, assign+1, rhsEnd)
	if len(lhs) != len(rhs) || len(rhs) <= 1 {
		return diags
	}
	for i := 0; i < len(lhs); i++ {
		got := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendAssignmentTargetTypeMismatchDiagnostic(diags, file, tokens, lhs[i], got, rhs[i].start, rhs[i].end, rhs[i].start, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func appendAssignmentTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, lhsStart int, assign int, rhsEnd int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	lhs := expressionRanges(tokens, lhsStart, assign)
	if len(lhs) != 1 {
		return diags
	}
	got := expressionRawTypeWithCallsAndTypes(tokens, assign+1, rhsEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	return appendAssignmentTargetTypeMismatchDiagnostic(diags, file, tokens, lhs[0], got, assign+1, rhsEnd, assign, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendAssignmentTargetTypeMismatchDiagnostic(diags Diagnostics, file parse.File, tokens []scan.Token, target expressionRange, got string, gotStart int, gotEnd int, reportPos int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	want := expressionRawTypeWithCallsAndTypes(tokens, target.start, target.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if want == "" || got == "" {
		return diags
	}
	if gotStart >= 0 && expressionAssignableToType(tokens, gotStart, gotEnd, want, got, typeNames) {
		return diags
	}
	if gotStart < 0 && typesAssignable(want, got, typeNames) {
		return diags
	}
	return appendDiag(diags, file, tokens[reportPos], "assignment type mismatch: "+expressionDisplayText(tokens, target.start, target.end)+" has "+want+", got "+got)
}

func appendCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, sigs []funcSignature, topTypes []string, locals []string, localTypes []localValueType, topNames []string, importNames []string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, importMethods []importedMethod, wordSize int) Diagnostics {
	name := tokens[pos].Text
	if target, ok := conversionTargetAt(tokens, pos, topTypes); ok {
		return appendConversionDiagnostics(diags, file, tokens, pos, target, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if startsPredeclaredReducibleComplexComponentCall(tokens, pos, sigs) {
		return appendReducibleComplexComponentCallDiagnostics(diags, file, tokens, pos, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if isBuiltinCallable(name) {
		return appendBuiltinCallDiagnostics(diags, file, tokens, pos, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if isRuntimeOSIntrinsicCall(file, tokens, pos) || startsSupportedNewCall(tokens, pos, structs, typeNames) {
		return diags
	}
	if startsUnsupportedBuiltinCall(tokens, pos) && !signedFunctionCallAt(tokens, pos, sigs) {
		return diags
	}
	sigIndex := funcSignatureIndex(sigs, name)
	if sigIndex >= 0 {
		open := pos + 1
		close := findClose(tokens, open, "(", ")")
		if close > open {
			argCount := expressionListCount(tokens, open+1, close)
			sig := sigs[sigIndex]
			if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
				diags = appendDiag(diags, file, tokens[pos], "argument count mismatch in call to "+name)
				return diags
			}
			diags = appendCallArgumentTypeDiagnostics(diags, file, tokens, name, open+1, close, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		return diags
	}
	importSigIndex := importedFunctionIndex(importFuncs, ".", name)
	if importSigIndex >= 0 {
		open := pos + 1
		close := findClose(tokens, open, "(", ")")
		if close > open {
			if isUnsafeSizeofFunction(importFuncs[importSigIndex]) {
				return appendUnsafeSizeofCallDiagnostics(diags, file, tokens, tokens[pos], open+1, close, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, wordSize)
			}
			argCount := expressionListCount(tokens, open+1, close)
			sig := importFuncs[importSigIndex].sig
			if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
				diags = appendDiag(diags, file, tokens[pos], "argument count mismatch in call to "+name)
				return diags
			}
			diags = appendCallArgumentTypeDiagnostics(diags, file, tokens, name, open+1, close, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		return diags
	}
	if containsString(locals, name) {
		if localValueTypeNameAt(localTypes, name, pos) == "" {
			return diags
		}
		return appendDiag(diags, file, tokens[pos], "call of non-function: "+name)
	}
	if containsString(topNames, name) || containsString(importNames, name) {
		return appendDiag(diags, file, tokens[pos], "call of non-function: "+name)
	}
	diags = appendDiag(diags, file, tokens[pos], "undefined identifier: "+name)
	return diags
}

func appendFunctionAliasAssignmentDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, body int, assign int, aliases []functionAlias) Diagnostics {
	if len(aliases) == 0 {
		return diags
	}
	stmtStart := simpleStatementStart(tokens, body, assign)
	lhs := expressionRanges(tokens, stmtStart, assign)
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		if _, ok := functionAliasAt(aliases, name, int(tokens[assign].Start)); ok {
			diags = appendDiag(diags, file, tokens[lhs[i].start], "function alias cannot be reassigned: "+name)
		}
	}
	return diags
}

func isFunctionAliasValueUse(tokens []scan.Token, pos int, aliases []functionAlias) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Kind != scan.Ident {
		return false
	}
	if _, ok := functionAliasAt(aliases, tokens[pos].Text, int(tokens[pos].Start)); !ok {
		return false
	}
	if functionValueBlankDiscardRHSAt(tokens, pos) {
		return false
	}
	if isCompositeKey(tokens, pos) || isSelectorMember(tokens, pos) {
		return false
	}
	if pos+1 < len(tokens) {
		next := tokens[pos+1].Text
		if next == "(" || next == "=" || next == ":=" || next == "," {
			return false
		}
	}
	if pos > 0 {
		prev := tokens[pos-1].Text
		if prev == "func" || prev == "." || prev == "type" || prev == "goto" || prev == "var" {
			return false
		}
	}
	return true
}

func methodExpressionAssignedToFunctionAlias(tokens []scan.Token, pos int, body int, limit int, aliases []functionAlias) bool {
	if len(aliases) == 0 || pos < 0 || pos >= len(tokens) {
		return false
	}
	stmtStart := simpleStatementStart(tokens, body, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, limit)
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(tokens, assign) {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		if pos < rhs[i].start || pos >= rhs[i].end {
			continue
		}
		name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
		if name == "" {
			return false
		}
		_, ok := functionAliasAt(aliases, name, int(tokens[assign].Start))
		return ok
	}
	return false
}

func functionValueBlankDiscardRHSAt(tokens []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	stmtStart := sameLineStatementStart(tokens, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, len(tokens))
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(tokens, assign) {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		if pos < rhs[i].start || pos >= rhs[i].end {
			continue
		}
		return blankIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
	}
	return false
}

func functionValueBlankDiscardExpressionAt(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start < 0 || start >= end || end > len(tokens) {
		return false
	}
	stmtStart := sameLineStatementStart(tokens, start)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, len(tokens))
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(tokens, assign) {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		rhsStart, rhsEnd := trimExpressionRange(tokens, rhs[i].start, rhs[i].end)
		if start != rhsStart || end != rhsEnd {
			continue
		}
		return blankIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
	}
	return false
}

func blankIdentifierExpression(tokens []scan.Token, start int, end int) bool {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	return start+1 == end && tokens[start].Kind == scan.Ident && tokens[start].Text == "_"
}

func sameLineStatementStart(tokens []scan.Token, pos int) int {
	line := tokens[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if tokens[i].Line != line || tokens[i].Text == ";" || tokens[i].Text == "{" || tokens[i].Text == "}" {
			return i + 1
		}
	}
	return 0
}

func functionAliasAt(aliases []functionAlias, name string, pos int) (functionAlias, bool) {
	for i := len(aliases) - 1; i >= 0; i-- {
		alias := aliases[i]
		if alias.name != name {
			continue
		}
		if pos < alias.start {
			continue
		}
		if alias.end > 0 && pos >= alias.end {
			continue
		}
		return alias, true
	}
	return functionAlias{}, false
}

func appendBuiltinCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	name := tokens[pos].Text
	if name != "append" && name != "cap" && name != "copy" && name != "len" && name != "make" && name != "panic" && name != "print" && name != "println" && name != "recover" {
		return diags
	}
	if name == "make" && (discardedLowerableMapMakeStatementContainingToken(tokens, pos) || lowerableMapMakeLenCallContainingToken(tokens, pos) || lowerableMapLiteralDeleteStatementContainingToken(tokens, pos) || lowerableMapLiteralIndexExpressionContainingToken(tokens, pos) || lowerableMapRangeStatementContainingToken(tokens, pos) || pureMapAliasStatementContainingToken(tokens, pos)) {
		return diags
	}
	open := pos + 1
	close := findClose(tokens, open, "(", ")")
	if close <= open {
		return diags
	}
	args := expressionRanges(tokens, open+1, close)
	for i := 0; i < len(args); i++ {
		if name == "make" && i == 0 {
			continue
		}
		arg := args[i]
		if name != "append" && hasVariadicExpansion(tokens, arg.start, arg.end) {
			diags = appendDiag(diags, file, tokens[arg.end-1], "variadic expansion in non-variadic call")
			continue
		}
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, arg.start, arg.end, sigs, importFuncs, localStructs, structs, importMethods)
	}
	switch name {
	case "cap":
		return appendCapCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "len":
		return appendLenCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "append":
		return appendAppendCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "copy":
		return appendCopyCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "make":
		return appendMakeCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "panic":
		return appendPanicCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "print":
		return appendPrintCallDiagnostics(diags, file, tokens, pos, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "println":
		return appendPrintlnCallDiagnostics(diags, file, tokens, args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	case "recover":
		return appendRecoverCallDiagnostics(diags, file, tokens, pos, args)
	}
	return diags
}

func appendPanicCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 1 {
		return appendDiag(diags, file, tokens[pos], "panic expects one argument")
	}
	arg := args[0]
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ != "" && typ != "string" {
		return appendDiag(diags, file, tokens[arg.start], "panic currently supports string values only")
	}
	return diags
}

func appendRecoverCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange) Diagnostics {
	if len(args) != 0 {
		return appendDiag(diags, file, tokens[pos], "recover expects no arguments")
	}
	return diags
}

func appendCapCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 1 {
		return appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, args[0].start, args[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ != "" && !isSliceTypeName(typ) {
		return appendDiag(diags, file, tokens[args[0].start], "invalid argument to cap: "+typ)
	}
	return diags
}

func appendLenCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 1 {
		return appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	if discardableLowerableMapMakeExpression(tokens, args[0].start, args[0].end) || pureMapAliasExpressionSupported(tokens, args[0].start, args[0].end) {
		return diags
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, args[0].start, args[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ != "" && typ != "string" && !isSliceTypeName(typ) {
		return appendDiag(diags, file, tokens[args[0].start], "invalid argument to len: "+typ)
	}
	return diags
}

func appendAppendCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) < 2 {
		return appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	dst := expressionRawTypeWithCallsAndTypes(tokens, args[0].start, args[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if dst == "" {
		return diags
	}
	if !isSliceLikeTypeName(dst, typeNames) {
		return appendDiag(diags, file, tokens[args[0].start], "first argument to append must be slice, got "+dst)
	}
	elem := sliceLikeElementRawType(dst, typeNames)
	for i := 1; i < len(args); i++ {
		arg := args[i]
		gotEnd := trimVariadicExpansion(tokens, arg.start, arg.end)
		got := expressionRawTypeWithCallsAndTypes(tokens, arg.start, gotEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" {
			continue
		}
		if i == len(args)-1 && hasVariadicExpansion(tokens, arg.start, arg.end) {
			want := "[]" + elem
			if !sliceExpansionTypesCompatible(elem, got, typeNames) {
				diags = appendDiag(diags, file, tokens[arg.start], "append expansion type mismatch: want "+want+", got "+got)
			}
			continue
		}
		if hasVariadicExpansion(tokens, arg.start, arg.end) {
			diags = appendDiag(diags, file, tokens[arg.end-1], "variadic expansion must be final argument")
			continue
		}
		if !expressionAssignableToType(tokens, arg.start, gotEnd, elem, got, typeNames) {
			diags = appendDiag(diags, file, tokens[arg.start], "append element type mismatch: want "+elem+", got "+got)
		}
	}
	return diags
}

func appendCopyCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 2 {
		return appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	dst := expressionRawTypeWithCallsAndTypes(tokens, args[0].start, args[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	src := expressionRawTypeWithCallsAndTypes(tokens, args[1].start, args[1].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if dst != "" && !isSliceLikeTypeName(dst, typeNames) {
		return appendDiag(diags, file, tokens[args[0].start], "first argument to copy must be slice, got "+dst)
	}
	if dst == "" || src == "" {
		return diags
	}
	dstElem := sliceLikeElementRawType(dst, typeNames)
	srcElem := sliceLikeElementRawType(src, typeNames)
	if srcElem != "" && dstElem == srcElem {
		return diags
	}
	if dstElem == "byte" && normalizeNamedType(src, typeNames) == "string" {
		return diags
	}
	return appendDiag(diags, file, tokens[args[1].start], "copy source type mismatch: want "+dst+", got "+src)
}

func appendMakeCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 2 && len(args) != 3 {
		return appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	typ := typeTextInRange(tokens, args[0].start, args[0].end)
	if typ != "" && !isSliceLikeTypeName(typ, typeNames) {
		diags = appendDiag(diags, file, tokens[args[0].start], "make requires slice type, got "+typ)
	}
	for i := 1; i < len(args); i++ {
		got := expressionSimpleTypeWithCallsAndTypes(tokens, args[i].start, args[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got != "" && !isIntegerTypeName(got) {
			diags = appendDiag(diags, file, tokens[args[i].start], "make size must be integer, got "+got)
		}
	}
	return diags
}

func appendPrintCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(args) != 1 {
		diags = appendBuiltinArgCountDiagnostic(diags, file, tokens, pos)
	}
	return appendPrintArgDiagnostics(diags, file, tokens, "print", args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendPrintlnCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendPrintArgDiagnostics(diags, file, tokens, "println", args, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendPrintArgDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, callName string, args []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		got := expressionSimpleTypeWithCallsAndTypes(tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" || typesCompatible("string", got) {
			continue
		}
		diags = appendDiag(diags, file, tokens[arg.start], "argument type mismatch in call to "+callName+": want string, got "+got)
	}
	return diags
}

func appendBuiltinArgCountDiagnostic(diags Diagnostics, file parse.File, tokens []scan.Token, pos int) Diagnostics {
	return appendDiag(diags, file, tokens[pos], "argument count mismatch in call to "+tokens[pos].Text)
}

func isSliceTypeName(name string) bool {
	return strings.HasPrefix(name, "[]")
}

func sliceElementType(name string) string {
	if !isSliceTypeName(name) {
		return ""
	}
	return name[2:]
}

func isSliceLikeTypeName(name string, typeNames []localValueType) bool {
	return sliceLikeElementRawType(name, typeNames) != ""
}

func sliceLikeElementRawType(name string, typeNames []localValueType) string {
	if strings.HasPrefix(name, "[]") {
		return name[2:]
	}
	underlying := rawNamedUnderlyingType(name, typeNames)
	if strings.HasPrefix(underlying, "[]") {
		return underlying[2:]
	}
	return ""
}

func rawNamedUnderlyingType(name string, typeNames []localValueType) string {
	seen := 0
	for name != "" && seen < len(typeNames)+1 {
		raw := localValueRawTypeName(typeNames, name)
		if raw == "" || raw == name {
			return raw
		}
		if strings.HasPrefix(raw, "[]") || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "*") {
			return raw
		}
		name = raw
		seen++
	}
	return ""
}

func sliceExpansionTypesCompatible(wantElem string, got string, typeNames []localValueType) bool {
	gotElem := sliceLikeElementRawType(got, typeNames)
	return gotElem != "" && gotElem == wantElem
}

func hasVariadicExpansion(tokens []scan.Token, start int, end int) bool {
	return end > start && tokens[end-1].Text == "..."
}

func trimVariadicExpansion(tokens []scan.Token, start int, end int) int {
	if hasVariadicExpansion(tokens, start, end) {
		return end - 1
	}
	return end
}

func appendConversionDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, target string, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	target = normalizeNamedType(target, typeNames)
	open := pos + 1
	close := findClose(tokens, open, "(", ")")
	if close <= open {
		return diags
	}
	args := expressionRanges(tokens, open+1, close)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, arg.start, arg.end, sigs, importFuncs, localStructs, structs, importMethods)
		if hasVariadicExpansion(tokens, arg.start, arg.end) {
			diags = appendDiag(diags, file, tokens[arg.end-1], "variadic expansion in non-variadic call")
		}
	}
	if len(args) != 1 {
		return appendDiag(diags, file, tokens[pos], "conversion requires one argument: "+target)
	}
	got := expressionSimpleTypeWithCallsAndTypes(tokens, args[0].start, args[0].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if got == "" || conversionTypesCompatible(target, got) {
		return diags
	}
	return appendDiag(diags, file, tokens[pos], "cannot convert "+got+" to "+target)
}

func conversionTargetAt(tokens []scan.Token, pos int, topTypes []string) (string, bool) {
	if pos+1 >= len(tokens) || tokens[pos].Kind != scan.Ident || tokens[pos+1].Text != "(" {
		return "", false
	}
	name := tokens[pos].Text
	if pos >= 2 && tokens[pos-2].Text == "[" && tokens[pos-1].Text == "]" && (isBuiltinTypeName(name) || containsString(topTypes, name)) {
		return "[]" + name, true
	}
	if isBuiltinTypeName(name) || containsString(topTypes, name) {
		return name, true
	}
	return "", false
}

func conversionTypesCompatible(target string, got string) bool {
	if target == got {
		return true
	}
	if isNumericTypeName(target) {
		return isNumericTypeName(got)
	}
	if target == "bool" {
		return got == "bool"
	}
	if target == "string" {
		return isIntegerTypeName(got) || got == "[]byte"
	}
	if target == "[]byte" {
		return got == "string"
	}
	return false
}

func appendMethodCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, receiver localStructType, sigs []funcSignature, importMethods []importedMethod, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType) Diagnostics {
	if pos+3 >= len(tokens) || tokens[pos+1].Text != "." || tokens[pos+2].Kind != scan.Ident || tokens[pos+3].Text != "(" {
		return diags
	}
	methodName := tokens[pos+2].Text
	sig := funcSignature{}
	if receiverMethod, ok := methodSignatureForStructType(receiver, methodName, sigs, importMethods); ok {
		sig = receiverMethod
	} else {
		typeName := structFieldSetName(receiver)
		promoted, promotedOK := promotedMethodSignature(structs, typeName, methodName, sigs, importMethods)
		if !promotedOK {
			return diags
		}
		sig = promoted
	}
	open := pos + 3
	close := findClose(tokens, open, "(", ")")
	if close <= open {
		return diags
	}
	argCount := expressionListCount(tokens, open+1, close)
	display := tokens[pos].Text + "." + methodName
	if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
		return appendDiag(diags, file, tokens[pos+2], "argument count mismatch in call to "+display)
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, display, open+1, close, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func methodSignatureForStructType(receiver localStructType, methodName string, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if receiver.qualifier != "" {
		index := importedMethodIndex(importMethods, receiver.qualifier, receiver.typeName, methodName)
		if index < 0 {
			return funcSignature{}, false
		}
		return importMethods[index].sig, true
	}
	return methodSignatureForTypeName(receiver.typeName, methodName, sigs, importMethods)
}

func methodSignatureForTypeName(typeName string, methodName string, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if typeName == "" {
		return funcSignature{}, false
	}
	if dot := strings.IndexByte(typeName, '.'); dot >= 0 {
		index := importedMethodIndex(importMethods, typeName[:dot], typeName[dot+1:], methodName)
		if index < 0 {
			return funcSignature{}, false
		}
		return importMethods[index].sig, true
	}
	index := methodSignatureIndex(sigs, typeName, methodName)
	if index < 0 {
		return funcSignature{}, false
	}
	return sigs[index], true
}

func promotedMethodSignature(structs []structFieldSet, ownerType string, methodName string, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	sig, _, count := promotedMethodSignatureInfoIn(structs, ownerType, methodName, sigs, importMethods, nil)
	return sig, count == 1
}

func promotedMethodSignatureInfo(structs []structFieldSet, ownerType string, methodName string, sigs []funcSignature, importMethods []importedMethod) (funcSignature, localValueType, bool) {
	sig, receiver, count := promotedMethodSignatureInfoIn(structs, ownerType, methodName, sigs, importMethods, nil)
	return sig, receiver, count == 1
}

func promotedMethodSignatureInfoIn(structs []structFieldSet, ownerType string, methodName string, sigs []funcSignature, importMethods []importedMethod, seen []string) (funcSignature, localValueType, int) {
	if ownerType == "" || containsString(seen, ownerType) {
		return funcSignature{}, localValueType{}, 0
	}
	seen = append(seen, ownerType)
	index := structFieldSetIndex(structs, ownerType)
	if index < 0 {
		return funcSignature{}, localValueType{}, 0
	}
	var found funcSignature
	var foundReceiver localValueType
	count := 0
	fields := structs[index].fieldTypes
	for i := 0; i < len(fields); i++ {
		embedded := fields[i]
		if !embedded.embedded {
			continue
		}
		embeddedType := fieldStructTypeName(structs, ownerType, embedded.typ)
		if embeddedType == "" {
			continue
		}
		if sig, ok := methodSignatureForTypeName(embeddedType, methodName, sigs, importMethods); ok {
			found = sig
			foundReceiver = embedded
			count++
			continue
		}
		nested, nestedReceiver, nestedCount := promotedMethodSignatureInfoIn(structs, embeddedType, methodName, sigs, importMethods, seen)
		if nestedCount > 0 {
			found = nested
			foundReceiver = nestedReceiver
			count += nestedCount
		}
	}
	return found, foundReceiver, count
}

type methodExpressionCallInfo struct {
	qualifier    string
	typeName     string
	methodName   string
	methodNameAt int
	methodOpen   int
	callClose    int
}

func appendMethodExpressionCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, topTypes []string, importNames []string, importShadows []localShadow, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	call, ok := methodExpressionCallInfoAt(tokens, pos, len(tokens))
	if !ok {
		return diags
	}
	if call.qualifier == "" {
		if !containsString(topTypes, call.typeName) {
			return diags
		}
	} else {
		if !containsString(importNames, call.qualifier) || isLocalShadowAt(importShadows, call.qualifier, int(tokens[pos].Start)) {
			return diags
		}
	}
	sig, ok := methodExpressionCallSignature(call, sigs, importMethods)
	if !ok {
		return diags
	}
	display := methodExpressionCallDisplay(call)
	if sig.pointerReceiver {
		return appendDiag(diags, file, tokens[call.methodNameAt], "method expressions are not supported: "+display)
	}
	callSig := methodExpressionCallFunctionSignature(call, sig)
	argCount := expressionListCount(tokens, call.methodOpen+1, call.callClose)
	if (!callSig.variadic && argCount != callSig.params) || (callSig.variadic && argCount < callSig.params-1) {
		return appendDiag(diags, file, tokens[call.methodNameAt], "argument count mismatch in call to "+display)
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, display, call.methodOpen+1, call.callClose, callSig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func methodExpressionCallInfoAt(tokens []scan.Token, start int, end int) (methodExpressionCallInfo, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end || tokens[start].Kind != scan.Ident {
		return methodExpressionCallInfo{}, false
	}
	if start+5 < end && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "." && tokens[start+4].Kind == scan.Ident && tokens[start+5].Text == "(" {
		close := findClose(tokens, start+5, "(", ")")
		if close < 0 || close >= end {
			return methodExpressionCallInfo{}, false
		}
		return methodExpressionCallInfo{
			qualifier:    tokens[start].Text,
			typeName:     tokens[start+2].Text,
			methodName:   tokens[start+4].Text,
			methodNameAt: start + 4,
			methodOpen:   start + 5,
			callClose:    close,
		}, true
	}
	if start+3 < end && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "(" {
		close := findClose(tokens, start+3, "(", ")")
		if close < 0 || close >= end {
			return methodExpressionCallInfo{}, false
		}
		return methodExpressionCallInfo{
			typeName:     tokens[start].Text,
			methodName:   tokens[start+2].Text,
			methodNameAt: start + 2,
			methodOpen:   start + 3,
			callClose:    close,
		}, true
	}
	return methodExpressionCallInfo{}, false
}

func methodExpressionCallSignature(call methodExpressionCallInfo, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if call.qualifier == "" {
		index := methodSignatureIndex(sigs, call.typeName, call.methodName)
		if index < 0 {
			return funcSignature{}, false
		}
		return sigs[index], true
	}
	index := importedMethodIndex(importMethods, call.qualifier, call.typeName, call.methodName)
	if index < 0 {
		return funcSignature{}, false
	}
	return importMethods[index].sig, true
}

func methodExpressionCallFunctionSignature(call methodExpressionCallInfo, sig funcSignature) funcSignature {
	out := sig
	receiverType := call.typeName
	if call.qualifier != "" {
		receiverType = call.qualifier + "." + call.typeName
	}
	if sig.pointerReceiver {
		receiverType = "*" + receiverType
	}
	out.params = sig.params + 1
	out.paramTypes = append([]string{receiverType}, sig.paramTypes...)
	return out
}

func methodExpressionCallDisplay(call methodExpressionCallInfo) string {
	if call.qualifier != "" {
		return call.qualifier + "." + call.typeName + "." + call.methodName
	}
	return call.typeName + "." + call.methodName
}

type indexedMethodCallInfo struct {
	qualifier     string
	typeName      string
	methodName    string
	methodNameAt  int
	methodOpen    int
	callClose     int
	receiverStart int
	receiverEnd   int
	pointer       bool
}

func appendIndexedReceiverMethodCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	call, ok := indexedReceiverMethodCallInfoAt(tokens, pos, len(tokens), localTypes, structs, typeNames)
	if !ok {
		return diags
	}
	sig, ok := indexedReceiverMethodSignature(call, sigs, importMethods)
	if !ok {
		return diags
	}
	argCount := expressionListCount(tokens, call.methodOpen+1, call.callClose)
	display := tokenTextInRange(tokens, call.receiverStart, call.receiverEnd) + "." + call.methodName
	if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
		return appendDiag(diags, file, tokens[call.methodNameAt], "argument count mismatch in call to "+display)
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, display, call.methodOpen+1, call.callClose, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

type compositeMethodCallInfo struct {
	qualifier    string
	typeName     string
	methodName   string
	methodNameAt int
	methodOpen   int
	callClose    int
	addressed    bool
}

func appendCompositeLiteralMethodCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	call, ok := compositeLiteralMethodCallInfoAt(tokens, start, end)
	if !ok {
		return diags
	}
	sig, promotedReceiver, promoted, ok := compositeLiteralMethodSignatureInfo(call, structs, sigs, importMethods)
	if !ok {
		return diags
	}
	displayType := call.typeName
	if call.qualifier != "" {
		displayType = call.qualifier + "." + call.typeName
	}
	if sig.pointerReceiver && !call.addressed && (!promoted || !localValueTypeIsPointer(promotedReceiver)) {
		return appendDiag(diags, file, tokens[call.methodNameAt], "cannot call pointer method "+call.methodName+" on "+displayType)
	}
	argCount := expressionListCount(tokens, call.methodOpen+1, call.callClose)
	display := displayType + "." + call.methodName
	if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
		return appendDiag(diags, file, tokens[call.methodNameAt], "argument count mismatch in call to "+display)
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, display, call.methodOpen+1, call.callClose, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func compositeLiteralMethodSignature(call compositeMethodCallInfo, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	sig, _, _, ok := compositeLiteralMethodSignatureInfo(call, nil, sigs, importMethods)
	return sig, ok
}

func compositeLiteralMethodSignatureInfo(call compositeMethodCallInfo, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) (funcSignature, localValueType, bool, bool) {
	receiverType := compositeLiteralMethodReceiverType(call)
	if sig, ok := methodSignatureForTypeName(receiverType, call.methodName, sigs, importMethods); ok {
		return sig, localValueType{}, false, true
	}
	sig, receiver, ok := promotedMethodSignatureInfo(structs, receiverType, call.methodName, sigs, importMethods)
	if !ok {
		return funcSignature{}, localValueType{}, false, false
	}
	return sig, receiver, true, true
}

func compositeLiteralMethodReceiverType(call compositeMethodCallInfo) string {
	if call.qualifier != "" {
		return call.qualifier + "." + call.typeName
	}
	return call.typeName
}

func localValueTypeIsPointer(value localValueType) bool {
	return strings.HasPrefix(localValueRawType(value), "*") || strings.HasPrefix(value.typ, "*")
}

func compositeLiteralMethodCallSignatureInfo(tokens []scan.Token, start int, end int, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) (funcSignature, localValueType, bool, bool) {
	call, ok := compositeLiteralMethodCallInfoAt(tokens, start, end)
	if !ok {
		return funcSignature{}, localValueType{}, false, false
	}
	return compositeLiteralMethodSignatureInfo(call, structs, sigs, importMethods)
}

func compositeLiteralMethodCallSignature(tokens []scan.Token, start int, end int, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	sig, _, _, ok := compositeLiteralMethodCallSignatureInfo(tokens, start, end, structs, sigs, importMethods)
	return sig, ok
}

func compositeLiteralMethodCallSignatureDirect(tokens []scan.Token, start int, end int, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	call, ok := compositeLiteralMethodCallInfoAt(tokens, start, end)
	if !ok {
		return funcSignature{}, false
	}
	return compositeLiteralMethodSignature(call, sigs, importMethods)
}

func appendCompositeLiteralMethodValueDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	value, ok := compositeLiteralMethodValueInfoAt(tokens, start, end)
	if !ok {
		return diags
	}
	if _, _, _, ok := compositeLiteralMethodSignatureInfo(value, structs, sigs, importMethods); !ok {
		return diags
	}
	if functionValueBlankDiscardExpressionAt(tokens, start, end) {
		return diags
	}
	displayType := value.typeName
	if value.qualifier != "" {
		displayType = value.qualifier + "." + value.typeName
	}
	return appendDiag(diags, file, tokens[value.methodNameAt], "method values are not supported: "+displayType+"."+value.methodName)
}

func compositeLiteralMethodValueInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return compositeMethodCallInfo{}, false
	}
	if tokens[start].Text == "(" {
		return parenthesizedCompositeLiteralMethodValueInfoAt(tokens, start, end)
	}
	return bareCompositeLiteralMethodValueInfoAt(tokens, start, end)
}

func parenthesizedCompositeLiteralMethodValueInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	closeParen := findClose(tokens, start, "(", ")")
	if closeParen <= start || closeParen+2 >= end || closeParen+3 != end || tokens[closeParen+1].Text != "." || tokens[closeParen+2].Kind != scan.Ident {
		return compositeMethodCallInfo{}, false
	}
	baseStart := start + 1
	addressed := false
	if baseStart < closeParen && tokens[baseStart].Text == "&" {
		addressed = true
		baseStart++
	}
	open := compositeLiteralTypeOpenAt(tokens, baseStart, closeParen)
	if open < 0 {
		return compositeMethodCallInfo{}, false
	}
	typ := compositeMethodReceiverType(tokens, baseStart, open)
	if typ == "" {
		return compositeMethodCallInfo{}, false
	}
	qualifier, typeName := splitQualifiedTypeName(typ)
	if typeName == "" {
		return compositeMethodCallInfo{}, false
	}
	return compositeMethodCallInfo{
		qualifier:    qualifier,
		typeName:     typeName,
		methodName:   tokens[closeParen+2].Text,
		methodNameAt: closeParen + 2,
		addressed:    addressed,
	}, true
}

func bareCompositeLiteralMethodValueInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	open := compositeLiteralTypeOpenAt(tokens, start, end)
	if open < 0 {
		return compositeMethodCallInfo{}, false
	}
	closeBrace := findClose(tokens, open, "{", "}")
	if closeBrace < 0 || closeBrace+2 >= end || closeBrace+3 != end || tokens[closeBrace+1].Text != "." || tokens[closeBrace+2].Kind != scan.Ident {
		return compositeMethodCallInfo{}, false
	}
	typ := compositeMethodReceiverType(tokens, start, open)
	if typ == "" {
		return compositeMethodCallInfo{}, false
	}
	qualifier, typeName := splitQualifiedTypeName(typ)
	if typeName == "" {
		return compositeMethodCallInfo{}, false
	}
	return compositeMethodCallInfo{
		qualifier:    qualifier,
		typeName:     typeName,
		methodName:   tokens[closeBrace+2].Text,
		methodNameAt: closeBrace + 2,
	}, true
}

func compositeLiteralMethodCallInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return compositeMethodCallInfo{}, false
	}
	if tokens[start].Text == "(" {
		return parenthesizedCompositeLiteralMethodCallInfoAt(tokens, start, end)
	}
	return bareCompositeLiteralMethodCallInfoAt(tokens, start, end)
}

func compositeLiteralMethodCallEnd(tokens []scan.Token, start int, limit int) int {
	if start >= limit || start >= len(tokens) {
		return -1
	}
	if tokens[start].Text == "(" {
		closeParen := findClose(tokens, start, "(", ")")
		if closeParen <= start || closeParen+3 >= limit || tokens[closeParen+1].Text != "." || tokens[closeParen+2].Kind != scan.Ident || tokens[closeParen+3].Text != "(" {
			return -1
		}
		callClose := findClose(tokens, closeParen+3, "(", ")")
		if callClose < 0 || callClose >= limit {
			return -1
		}
		return callClose + 1
	}
	open := compositeLiteralTypeOpenAt(tokens, start, limit)
	if open < 0 {
		return -1
	}
	closeBrace := findClose(tokens, open, "{", "}")
	if closeBrace < 0 || closeBrace+3 >= limit || tokens[closeBrace+1].Text != "." || tokens[closeBrace+2].Kind != scan.Ident || tokens[closeBrace+3].Text != "(" {
		return -1
	}
	callClose := findClose(tokens, closeBrace+3, "(", ")")
	if callClose < 0 || callClose >= limit {
		return -1
	}
	return callClose + 1
}

func parenthesizedCompositeLiteralMethodCallInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	closeParen := findClose(tokens, start, "(", ")")
	if closeParen <= start || closeParen+3 >= end || tokens[closeParen+1].Text != "." || tokens[closeParen+2].Kind != scan.Ident || tokens[closeParen+3].Text != "(" {
		return compositeMethodCallInfo{}, false
	}
	callClose := findClose(tokens, closeParen+3, "(", ")")
	if callClose != end-1 {
		return compositeMethodCallInfo{}, false
	}
	baseStart := start + 1
	addressed := false
	if baseStart < closeParen && tokens[baseStart].Text == "&" {
		addressed = true
		baseStart++
	}
	open := compositeLiteralTypeOpenAt(tokens, baseStart, closeParen)
	if open < 0 {
		return compositeMethodCallInfo{}, false
	}
	typ := compositeMethodReceiverType(tokens, baseStart, open)
	if typ == "" {
		return compositeMethodCallInfo{}, false
	}
	qualifier, typeName := splitQualifiedTypeName(typ)
	if typeName == "" {
		return compositeMethodCallInfo{}, false
	}
	return compositeMethodCallInfo{
		qualifier:    qualifier,
		typeName:     typeName,
		methodName:   tokens[closeParen+2].Text,
		methodNameAt: closeParen + 2,
		methodOpen:   closeParen + 3,
		callClose:    callClose,
		addressed:    addressed,
	}, true
}

func bareCompositeLiteralMethodCallInfoAt(tokens []scan.Token, start int, end int) (compositeMethodCallInfo, bool) {
	open := compositeLiteralTypeOpenAt(tokens, start, end)
	if open < 0 {
		return compositeMethodCallInfo{}, false
	}
	closeBrace := findClose(tokens, open, "{", "}")
	if closeBrace < 0 || closeBrace+3 >= end || tokens[closeBrace+1].Text != "." || tokens[closeBrace+2].Kind != scan.Ident || tokens[closeBrace+3].Text != "(" {
		return compositeMethodCallInfo{}, false
	}
	callClose := findClose(tokens, closeBrace+3, "(", ")")
	if callClose != end-1 {
		return compositeMethodCallInfo{}, false
	}
	typ := compositeMethodReceiverType(tokens, start, open)
	if typ == "" {
		return compositeMethodCallInfo{}, false
	}
	qualifier, typeName := splitQualifiedTypeName(typ)
	if typeName == "" {
		return compositeMethodCallInfo{}, false
	}
	return compositeMethodCallInfo{
		qualifier:    qualifier,
		typeName:     typeName,
		methodName:   tokens[closeBrace+2].Text,
		methodNameAt: closeBrace + 2,
		methodOpen:   closeBrace + 3,
		callClose:    callClose,
	}, true
}

func compositeMethodReceiverType(tokens []scan.Token, start int, open int) string {
	if start+1 == open && tokens[start].Kind == scan.Ident {
		return tokens[start].Text
	}
	if start+3 == open && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		return tokens[start].Text + "." + tokens[start+2].Text
	}
	return typeTextInRange(tokens, start, open)
}

func indexedReceiverMethodCallSyntaxAt(tokens []scan.Token, start int, end int) bool {
	_, _, _, _, ok := indexedReceiverMethodCallTokens(tokens, start, end)
	return ok
}

func indexedReceiverMethodCallInfoAt(tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType) (indexedMethodCallInfo, bool) {
	receiverEnd, methodNameAt, methodOpen, callClose, ok := indexedReceiverMethodCallTokens(tokens, start, end)
	if !ok {
		return indexedMethodCallInfo{}, false
	}
	typ := expressionRawSimpleType(tokens, start, receiverEnd, localTypes, structs, typeNames)
	if typ == "" {
		typ = expressionSimpleType(tokens, start, receiverEnd, localTypes, structs)
	}
	if typ == "" {
		return indexedMethodCallInfo{}, false
	}
	pointer := strings.HasPrefix(typ, "*")
	base := selectorBaseType(typ)
	qualifier, typeName := splitQualifiedTypeName(base)
	if typeName == "" {
		return indexedMethodCallInfo{}, false
	}
	return indexedMethodCallInfo{
		qualifier:     qualifier,
		typeName:      typeName,
		methodName:    tokens[methodNameAt].Text,
		methodNameAt:  methodNameAt,
		methodOpen:    methodOpen,
		callClose:     callClose,
		receiverStart: start,
		receiverEnd:   receiverEnd,
		pointer:       pointer,
	}, true
}

func indexedReceiverMethodCallTokens(tokens []scan.Token, start int, end int) (int, int, int, int, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return 0, 0, 0, 0, false
	}
	if tokens[start].Kind == scan.Ident && start+1 < end && tokens[start+1].Text == "[" {
		closeIndex := findClose(tokens, start+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= end || tokens[closeIndex+1].Text != "." || tokens[closeIndex+2].Kind != scan.Ident || tokens[closeIndex+3].Text != "(" {
			return 0, 0, 0, 0, false
		}
		callClose := findClose(tokens, closeIndex+3, "(", ")")
		if callClose < 0 || callClose >= end {
			return 0, 0, 0, 0, false
		}
		return closeIndex + 1, closeIndex + 2, closeIndex + 3, callClose, true
	}
	if tokens[start].Text == "(" {
		closeParen := findClose(tokens, start, "(", ")")
		if closeParen < 0 || closeParen+1 >= end || tokens[closeParen+1].Text != "[" {
			return 0, 0, 0, 0, false
		}
		closeIndex := findClose(tokens, closeParen+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= end || tokens[closeIndex+1].Text != "." || tokens[closeIndex+2].Kind != scan.Ident || tokens[closeIndex+3].Text != "(" {
			return 0, 0, 0, 0, false
		}
		callClose := findClose(tokens, closeIndex+3, "(", ")")
		if callClose < 0 || callClose >= end {
			return 0, 0, 0, 0, false
		}
		return closeIndex + 1, closeIndex + 2, closeIndex + 3, callClose, true
	}
	if open := compositeLiteralTypeOpenAt(tokens, start, end); open >= 0 {
		closeBrace := findClose(tokens, open, "{", "}")
		if closeBrace < 0 || closeBrace+1 >= end || tokens[closeBrace+1].Text != "[" {
			return 0, 0, 0, 0, false
		}
		closeIndex := findClose(tokens, closeBrace+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= end || tokens[closeIndex+1].Text != "." || tokens[closeIndex+2].Kind != scan.Ident || tokens[closeIndex+3].Text != "(" {
			return 0, 0, 0, 0, false
		}
		callClose := findClose(tokens, closeIndex+3, "(", ")")
		if callClose < 0 || callClose >= end {
			return 0, 0, 0, 0, false
		}
		return closeIndex + 1, closeIndex + 2, closeIndex + 3, callClose, true
	}
	if tokens[start].Text != "[" || start+1 >= end || tokens[start+1].Text != "]" {
		return 0, 0, 0, 0, false
	}
	typeStart := start + 2
	if typeStart < end && tokens[typeStart].Text == "*" {
		typeStart++
	}
	open := compositeLiteralTypeOpenAt(tokens, typeStart, end)
	if open < 0 {
		return 0, 0, 0, 0, false
	}
	closeBrace := findClose(tokens, open, "{", "}")
	if closeBrace < 0 || closeBrace+1 >= end || tokens[closeBrace+1].Text != "[" {
		return 0, 0, 0, 0, false
	}
	closeIndex := findClose(tokens, closeBrace+1, "[", "]")
	if closeIndex < 0 || closeIndex+3 >= end || tokens[closeIndex+1].Text != "." || tokens[closeIndex+2].Kind != scan.Ident || tokens[closeIndex+3].Text != "(" {
		return 0, 0, 0, 0, false
	}
	callClose := findClose(tokens, closeIndex+3, "(", ")")
	if callClose < 0 || callClose >= end {
		return 0, 0, 0, 0, false
	}
	return closeIndex + 1, closeIndex + 2, closeIndex + 3, callClose, true
}

func indexedReceiverMethodSignature(call indexedMethodCallInfo, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if call.qualifier == "" {
		index := methodSignatureIndex(sigs, call.typeName, call.methodName)
		if index < 0 {
			return funcSignature{}, false
		}
		return sigs[index], true
	}
	index := importedMethodIndex(importMethods, call.qualifier, call.typeName, call.methodName)
	if index < 0 {
		return funcSignature{}, false
	}
	return importMethods[index].sig, true
}

func indexedReceiverMethodCallSignature(tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	call, ok := indexedReceiverMethodCallInfoAt(tokens, start, end, localTypes, structs, typeNames)
	if !ok || call.callClose != end-1 {
		return funcSignature{}, false
	}
	return indexedReceiverMethodSignature(call, sigs, importMethods)
}

func compositeLiteralTypeOpenAt(tokens []scan.Token, start int, end int) int {
	if start < 0 || start >= end || tokens[start].Kind != scan.Ident {
		return -1
	}
	if start+1 < end && tokens[start+1].Text == "{" {
		return start + 1
	}
	if start+3 < end && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "{" {
		return start + 3
	}
	return -1
}

func splitQualifiedTypeName(typ string) (string, string) {
	dot := strings.LastIndexByte(typ, '.')
	if dot < 0 {
		return "", typ
	}
	return typ[:dot], typ[dot+1:]
}

func methodSignatureIndex(sigs []funcSignature, receiverType string, methodName string) int {
	for i := 0; i < len(sigs); i++ {
		if sigs[i].receiverType == receiverType && sigs[i].name == methodName {
			return i
		}
	}
	return -1
}

func namedReceiverMethodSignatureAt(tokens []scan.Token, pos int, localTypes []localValueType, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if pos+3 >= len(tokens) || tokens[pos].Kind != scan.Ident || tokens[pos+1].Text != "." || tokens[pos+2].Kind != scan.Ident || tokens[pos+3].Text != "(" {
		return funcSignature{}, false
	}
	raw := localValueRawTypeNameAt(localTypes, tokens[pos].Text, pos)
	if raw == "" {
		return funcSignature{}, false
	}
	for strings.HasPrefix(raw, "*") {
		raw = raw[1:]
	}
	if raw == "" || strings.HasPrefix(raw, "[]") {
		return funcSignature{}, false
	}
	methodName := tokens[pos+2].Text
	if dot := strings.IndexByte(raw, '.'); dot >= 0 {
		index := importedMethodIndex(importMethods, raw[:dot], raw[dot+1:], methodName)
		if index < 0 {
			return funcSignature{}, false
		}
		return importMethods[index].sig, true
	}
	index := methodSignatureIndex(sigs, raw, methodName)
	if index < 0 {
		return funcSignature{}, false
	}
	return sigs[index], true
}

func appendImportedCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, funcs []importedFunction, importValues []importedValue, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, wordSize int) Diagnostics {
	if pos+3 >= len(tokens) || tokens[pos+1].Text != "." || tokens[pos+3].Text != "(" {
		return diags
	}
	qualifier := tokens[pos].Text
	name := tokens[pos+2].Text
	index := importedFunctionIndex(funcs, qualifier, name)
	if index < 0 {
		return diags
	}
	open := pos + 3
	close := findClose(tokens, open, "(", ")")
	if close <= open {
		return diags
	}
	fn := funcs[index]
	if isUnsafeSizeofFunction(fn) {
		return appendUnsafeSizeofCallDiagnostics(diags, file, tokens, tokens[pos+2], open+1, close, localTypes, structs, typeNames, funcs, importValues, localStructs, sigs, importMethods, wordSize)
	}
	if qualifier == "fmt" && name == "Errorf" {
		return appendFmtErrorfCallDiagnostics(diags, file, tokens, open+1, close, localTypes, structs, typeNames, funcs, importValues, localStructs, sigs, importMethods)
	}
	sig := fn.sig
	argCount := expressionListCount(tokens, open+1, close)
	display := qualifier + "." + name
	if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
		return appendDiag(diags, file, tokens[pos+2], "argument count mismatch in call to "+display)
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, display, open+1, close, sig, localTypes, structs, typeNames, funcs, importValues, localStructs, sigs, importMethods)
}

func appendFmtErrorfCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	args := expressionRanges(tokens, start, end)
	if len(args) == 0 {
		return appendDiag(diags, file, tokens[start-1], "argument count mismatch in call to fmt.Errorf")
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, arg.start, arg.end, sigs, importFuncs, localStructs, structs, importMethods)
	}
	first := args[0]
	got := expressionSimpleTypeWithCallsAndTypes(tokens, first.start, first.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if got != "" && normalizeNamedType(got, typeNames) != "string" {
		return appendDiag(diags, file, tokens[first.start], "argument type mismatch in call to fmt.Errorf: want string, got "+got)
	}
	return diags
}

func importedFunctionIndex(funcs []importedFunction, qualifier string, name string) int {
	for i := 0; i < len(funcs); i++ {
		if funcs[i].qualifier == qualifier && funcs[i].sig.name == name {
			return i
		}
	}
	return -1
}

func isUnsafeSizeofFunction(fn importedFunction) bool {
	return fn.importPath == "unsafe" && fn.sig.name == "Sizeof"
}

func appendUnsafeSizeofCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, report scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, wordSize int) Diagnostics {
	args := expressionRanges(tokens, start, end)
	if len(args) != 1 {
		return appendDiag(diags, file, report, "argument count mismatch in call to unsafe.Sizeof")
	}
	arg := args[0]
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, arg.start, arg.end, sigs, importFuncs, localStructs, structs, importMethods)
	typ := expressionRawTypeWithCallsAndTypes(tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if _, ok := unsafeSizeofFixedTypeSize(typ, typeNames, structs, wordSize); !ok {
		if typ == "" {
			typ = "unknown"
		}
		return appendDiag(diags, file, tokens[arg.start], "unsafe.Sizeof operand size is not lowerable yet, got "+typ)
	}
	if !unsafeSizeofOperandLowerable(tokens, arg.start, arg.end, typeNames, structs, wordSize) {
		return appendDiag(diags, file, tokens[arg.start], "unsafe.Sizeof operand form is not lowerable yet")
	}
	return diags
}

func unsafeSizeofFixedTypeSize(typ string, typeNames []localValueType, structs []structFieldSet, wordSize int) (int, bool) {
	size, _, ok := unsafeSizeofTypeLayout(typ, typeNames, structs, wordSize, nil)
	return size, ok
}

func unsafeSizeofTypeLayout(typ string, typeNames []localValueType, structs []structFieldSet, wordSize int, seen []string) (int, int, bool) {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return 0, 0, false
	}
	if containsString(seen, typ) {
		return 0, 0, false
	}
	if info, ok := fixedArrayTypeInfoFromType(typ, typeNames); ok {
		elemSize, elemAlign, elemOK := unsafeSizeofTypeLayout(info.elem, typeNames, structs, wordSize, seen)
		if !elemOK {
			return 0, 0, false
		}
		return int(info.length) * elemSize, elemAlign, true
	}
	if strings.HasPrefix(typ, "*") && len(typ) > 1 {
		return wordSize, wordSize, true
	}
	if strings.HasPrefix(typ, "[]") && len(typ) > 2 {
		return wordSize * 3, wordSize, true
	}
	switch typ {
	case "byte", "bool":
		return 1, 1, true
	case "int":
		return wordSize, wordSize, true
	case "int16":
		return 2, 2, true
	case "int32":
		return 4, 4, true
	case "int64", "float64":
		return 8, unsafeSizeofMinAlign(8, wordSize), true
	case "string":
		return wordSize * 2, wordSize, true
	}
	if size, align, ok := unsafeSizeofStructTypeLayout(typ, typeNames, structs, wordSize, seen); ok {
		return size, align, true
	}
	resolved := unsafeSizeofResolveNamedType(typ, typeNames)
	if resolved != typ {
		return unsafeSizeofTypeLayout(resolved, typeNames, structs, wordSize, append(cloneStrings(seen), typ))
	}
	return 0, 0, false
}

func unsafeSizeofStructTypeLayout(typ string, typeNames []localValueType, structs []structFieldSet, wordSize int, seen []string) (int, int, bool) {
	if typ == "" || containsString(seen, typ) {
		return 0, 0, false
	}
	index := structFieldSetIndex(structs, typ)
	if index < 0 {
		return 0, 0, false
	}
	offset := 0
	maxAlign := 1
	nextSeen := append(cloneStrings(seen), typ)
	fields := structs[index].fieldTypes
	for i := 0; i < len(fields); i++ {
		fieldType := localValueRawType(fields[i])
		fieldSize, fieldAlign, ok := unsafeSizeofTypeLayout(fieldType, typeNames, structs, wordSize, nextSeen)
		if !ok || fieldAlign <= 0 {
			return 0, 0, false
		}
		offset = unsafeSizeofAlignOffset(offset, fieldAlign)
		offset += fieldSize
		if fieldAlign > maxAlign {
			maxAlign = fieldAlign
		}
	}
	return unsafeSizeofAlignOffset(offset, maxAlign), maxAlign, true
}

func unsafeSizeofMinAlign(size int, wordSize int) int {
	if wordSize > 0 && wordSize < size {
		return wordSize
	}
	return size
}

func unsafeSizeofAlignOffset(offset int, align int) int {
	if align <= 1 {
		return offset
	}
	rem := offset % align
	if rem == 0 {
		return offset
	}
	return offset + align - rem
}

func unsafeSizeofResolveNamedType(typ string, typeNames []localValueType) string {
	if typ == "" || strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return typ
	}
	raw := localValueRawTypeName(typeNames, typ)
	if raw == "" || raw == typ {
		return typ
	}
	return raw
}

func unsafeSizeofOperandLowerable(tokens []scan.Token, start int, end int, typeNames []localValueType, structs []structFieldSet, wordSize int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return unsafeSizeofOperandLowerable(tokens, start+1, close, typeNames, structs, wordSize)
		}
	}
	if start+1 == end {
		tok := tokens[start]
		return tok.Kind == scan.String || tok.Kind == scan.Number || tok.Kind == scan.Char || tok.Kind == scan.Ident
	}
	if tokens[start].Text == "&" && start+1 < end {
		return true
	}
	if unsafeSizeofSliceLiteralOperand(tokens, start, end) {
		return true
	}
	if unsafeSizeofFixedArrayLiteralOperand(tokens, start, end) {
		return true
	}
	if unsafeSizeofStructLiteralOperand(tokens, start, end, typeNames, structs, wordSize) {
		return true
	}
	if unsafeSizeofConversionOperand(tokens, start, end, typeNames, structs, wordSize) {
		return true
	}
	return false
}

func unsafeSizeofFixedArrayLiteralOperand(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start+4 > end || tokens[start].Text != "[" {
		return false
	}
	open := compositeLiteralOpen(tokens, start, end)
	if open <= start || open >= end {
		return false
	}
	close := findClose(tokens, open, "{", "}")
	if close != end-1 {
		return false
	}
	brackClose := findClose(tokens, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= open {
		return false
	}
	if !fixedArrayLengthSupported(tokens, start+1, brackClose) {
		return false
	}
	return true
}

func unsafeSizeofSliceLiteralOperand(tokens []scan.Token, start int, end int) bool {
	if start+3 >= end || tokens[start].Text != "[" || tokens[start+1].Text != "]" {
		return false
	}
	open := compositeLiteralOpen(tokens, start, end)
	return open > start && open < end
}

func unsafeSizeofStructLiteralOperand(tokens []scan.Token, start int, end int, typeNames []localValueType, structs []structFieldSet, wordSize int) bool {
	typ := compositeLiteralType(tokens, start, end)
	if typ == "" {
		return false
	}
	_, ok := unsafeSizeofFixedTypeSize(typ, typeNames, structs, wordSize)
	return ok
}

func unsafeSizeofConversionOperand(tokens []scan.Token, start int, end int, typeNames []localValueType, structs []structFieldSet, wordSize int) bool {
	if start+3 > end {
		return false
	}
	if tokens[start].Kind == scan.Ident && tokens[start+1].Text == "(" {
		close := findClose(tokens, start+1, "(", ")")
		if close == end-1 {
			name := tokens[start].Text
			if isBuiltinTypeName(name) && localValueTypeName(typeNames, name) == "" {
				return true
			}
			_, ok := unsafeSizeofFixedTypeSize(name, typeNames, structs, wordSize)
			return ok
		}
	}
	if tokens[start].Text == "[" && start+1 < end && tokens[start+1].Text == "]" {
		paren := 0
		brack := 0
		brace := 0
		for i := start + 2; i < end; i++ {
			if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "(" {
				close := findClose(tokens, i, "(", ")")
				return close == end-1
			}
			updateDepth(tokens[i].Text, &paren, &brack, &brace)
		}
	}
	return false
}

func appendCallArgumentTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, callName string, start int, end int, sig funcSignature, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	args := expressionRanges(tokens, start, end)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if callbackArgIndex(sig.callbackParams, i) >= 0 {
			if _, ok := staticCallbackTargetSignature(tokens, arg.start, arg.end, sigs, importFuncs, importMethods, localTypes, structs, typeNames, localStructs); ok {
				continue
			}
		}
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if len(args) != 1 || !singleArgumentExpandsToParameters(tokens, arg.start, arg.end, sig, importFuncs, localStructs, structs, importMethods, sigs) {
			diags = appendSingleValueCallDiagnostics(diags, file, tokens, arg.start, arg.end, sigs, importFuncs, localStructs, structs, importMethods)
		}
		if hasVariadicExpansion(tokens, arg.start, arg.end) && !sig.variadic {
			diags = appendDiag(diags, file, tokens[arg.end-1], "variadic expansion in non-variadic call")
		}
	}
	diags = appendStaticCallbackArgumentDiagnostics(diags, file, tokens, callName, args, sig, sigs, importFuncs, importMethods, localTypes, structs, typeNames, localStructs)
	if sig.variadic {
		return appendVariadicCallArgumentTypeDiagnostics(diags, file, tokens, callName, args, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if len(sig.paramTypes) == 0 {
		return diags
	}
	typeArgs := callArgumentRangesForParamTypes(args, sig)
	for i := 0; i < len(typeArgs) && i < len(sig.paramTypes); i++ {
		if callbackArgIndex(sig.callbackParams, i) >= 0 {
			continue
		}
		want := sig.paramTypes[i]
		if want == "" || lowerableInterfaceParamRawType(want) {
			continue
		}
		arg := typeArgs[i]
		got := expressionRawTypeWithCallsAndTypes(tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" || expressionAssignableToType(tokens, arg.start, arg.end, want, got, typeNames) {
			continue
		}
		diags = appendDiag(diags, file, tokens[arg.start], "argument type mismatch in call to "+callName+": want "+want+", got "+got)
	}
	return diags
}

func callArgumentRangesForParamTypes(args []expressionRange, sig funcSignature) []expressionRange {
	if len(sig.erasedParams) == 0 {
		return args
	}
	out := make([]expressionRange, 0, len(args))
	for i := 0; i < len(args); i++ {
		if interfaceParamIndexInSet(sig.erasedParams, i) {
			continue
		}
		out = append(out, args[i])
	}
	return out
}

func lowerableInterfaceParamRawType(typ string) bool {
	typ = strings.ReplaceAll(typ, " ", "")
	return typ == "any" || typ == "interface{}"
}

func appendStaticCallbackArgumentDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, callName string, args []expressionRange, sig funcSignature, sigs []funcSignature, importFuncs []importedFunction, importMethods []importedMethod, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType) Diagnostics {
	for i := 0; i < len(sig.callbackParams); i++ {
		callback := sig.callbackParams[i]
		if callback.index < 0 || callback.index >= len(args) {
			continue
		}
		arg := args[callback.index]
		target, ok := staticCallbackTargetSignature(tokens, arg.start, arg.end, sigs, importFuncs, importMethods, localTypes, structs, typeNames, localStructs)
		if !ok {
			diags = appendDiag(diags, file, tokens[arg.start], "callback argument in call to "+callName+" must be a static function")
			continue
		}
		if !callbackSignaturesCompatible(callback.sig, target) {
			diags = appendDiag(diags, file, tokens[arg.start], "callback signature mismatch in call to "+callName)
		}
	}
	return diags
}

func callbackArgIndex(callbacks []functionParamSignature, index int) int {
	for i := 0; i < len(callbacks); i++ {
		if callbacks[i].index == index {
			return i
		}
	}
	return -1
}

func staticCallbackTargetSignature(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, importMethods []importedMethod, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return funcSignature{}, false
	}
	if sig, ok := functionLiteralAliasTargetSignature(tokens, start, end, structs, typeNames); ok {
		return sig, true
	}
	if sig, ok := methodValueAliasTargetSignature(tokens, start, end, localTypes, structs, typeNames, localStructs, sigs, importMethods); ok {
		return sig, true
	}
	if sig, ok := compositeLiteralMethodValueAliasTargetSignature(tokens, start, end, structs, sigs, importMethods); ok {
		return sig, true
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		index := funcSignatureIndex(sigs, tokens[start].Text)
		if index >= 0 && (!sigs[index].staticAlias || sigs[index].staticCallback) {
			return sigs[index], true
		}
		index = importedFunctionIndex(importFuncs, ".", tokens[start].Text)
		if index >= 0 {
			return importFuncs[index].sig, true
		}
		return funcSignature{}, false
	}
	if start+3 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		index := importedFunctionIndex(importFuncs, tokens[start].Text, tokens[start+2].Text)
		if index >= 0 {
			return importFuncs[index].sig, true
		}
	}
	if sig, ok := methodExpressionAliasTargetSignature(tokens, start, end, sigs, importMethods); ok {
		return sig, true
	}
	return funcSignature{}, false
}

func callbackSignaturesCompatible(want funcSignature, got funcSignature) bool {
	if want.params != got.params || want.results != got.results {
		return false
	}
	if len(want.paramTypes) == len(got.paramTypes) {
		for i := 0; i < len(want.paramTypes); i++ {
			if want.paramTypes[i] != "" && got.paramTypes[i] != "" && want.paramTypes[i] != got.paramTypes[i] {
				return false
			}
		}
	}
	if len(want.resultTypes) == len(got.resultTypes) {
		for i := 0; i < len(want.resultTypes); i++ {
			if want.resultTypes[i] != "" && got.resultTypes[i] != "" && want.resultTypes[i] != got.resultTypes[i] {
				return false
			}
		}
	}
	return true
}

func isCallbackArgumentUse(tokens []scan.Token, pos int, body int, close int, sigs []funcSignature, importFuncs []importedFunction) bool {
	for i := body + 1; i < close; i++ {
		if tokens[i].Kind != scan.Ident {
			continue
		}
		sig, open, ok := callbackCallSignatureAt(tokens, i, close, sigs, importFuncs)
		if !ok || len(sig.callbackParams) == 0 {
			continue
		}
		callClose := findClose(tokens, open, "(", ")")
		if callClose < 0 || callClose > close {
			continue
		}
		args := expressionRanges(tokens, open+1, callClose)
		for callbackIndex := 0; callbackIndex < len(sig.callbackParams); callbackIndex++ {
			callback := sig.callbackParams[callbackIndex]
			if callback.index < 0 || callback.index >= len(args) {
				continue
			}
			arg := args[callback.index]
			if pos < arg.start || pos >= arg.end {
				continue
			}
			return true
		}
	}
	return false
}

func callbackCallSignatureAt(tokens []scan.Token, pos int, close int, sigs []funcSignature, importFuncs []importedFunction) (funcSignature, int, bool) {
	if pos+1 < close && tokens[pos+1].Text == "(" && !isSelectorMember(tokens, pos) {
		sigIndex := funcSignatureIndex(sigs, tokens[pos].Text)
		if sigIndex >= 0 {
			return sigs[sigIndex], pos + 1, true
		}
		index := importedFunctionIndex(importFuncs, ".", tokens[pos].Text)
		if index >= 0 {
			return importFuncs[index].sig, pos + 1, true
		}
		return funcSignature{}, 0, false
	}
	if pos+3 < close && tokens[pos+1].Text == "." && tokens[pos+2].Kind == scan.Ident && tokens[pos+3].Text == "(" {
		index := importedFunctionIndex(importFuncs, tokens[pos].Text, tokens[pos+2].Text)
		if index >= 0 {
			return importFuncs[index].sig, pos + 3, true
		}
	}
	return funcSignature{}, 0, false
}

func appendVariadicCallArgumentTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, callName string, args []expressionRange, sig funcSignature, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if len(sig.paramTypes) == 0 {
		return diags
	}
	variadicIndex := len(sig.paramTypes) - 1
	variadicSlice := sig.paramTypes[variadicIndex]
	variadicElem := sliceElementType(variadicSlice)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		expands := hasVariadicExpansion(tokens, arg.start, arg.end)
		if expands && i != len(args)-1 {
			diags = appendDiag(diags, file, tokens[arg.end-1], "variadic expansion must be final argument")
			continue
		}
		want := variadicElem
		if i < variadicIndex {
			want = sig.paramTypes[i]
		} else {
			want = variadicElem
			if expands {
				want = variadicSlice
			}
		}
		if want == "" {
			continue
		}
		gotEnd := trimVariadicExpansion(tokens, arg.start, arg.end)
		got := expressionRawTypeWithCallsAndTypes(tokens, arg.start, gotEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got == "" || expressionAssignableToType(tokens, arg.start, gotEnd, want, got, typeNames) {
			continue
		}
		diags = appendDiag(diags, file, tokens[arg.start], "argument type mismatch in call to "+callName+": want "+want+", got "+got)
	}
	return diags
}

func appendExpressionListOperandDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendExpressionListOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, nil, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendExpressionListOperandDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	values := expressionRanges(tokens, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, value.start, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func appendExpressionOperandDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, nil, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendExpressionOperandDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return diags
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start+1, close, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
	}
	diags = appendCompositeLiteralMethodCallDiagnostics(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendCompositeLiteralMethodValueDiagnostics(diags, file, tokens, start, end, structs, sigs, importMethods)
	if tokens[start].Text == "*" && start+1 < end && topLevelBinaryOperator(tokens, start, end) < 0 {
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, start+1, end, sigs, importFuncs, localStructs, structs, importMethods)
		name := singleIdentifierExpression(tokens, start+1, end)
		if name != "" && name != "true" && name != "false" && name != "nil" {
			return diags
		}
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if typ == "" || strings.HasPrefix(typ, "*") {
			return diags
		}
		return appendDiag(diags, file, tokens[start], "cannot dereference non-pointer: "+typ)
	}
	if tokens[start].Text == "&" && start+1 < end && topLevelBinaryOperator(tokens, start, end) < 0 {
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, start+1, end, sigs, importFuncs, localStructs, structs, importMethods)
		if expressionIsStringIndex(tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods) {
			return appendDiag(diags, file, tokens[start], "cannot take address of string index")
		}
		if addressableExpression(tokens, start+1, end) {
			return diags
		}
		return appendDiag(diags, file, tokens[start], "cannot take address of non-addressable expression")
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, start+1, end, sigs, importFuncs, localStructs, structs, importMethods)
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, start+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if typ == "" || unaryOperandCompatible(op, typ) {
			return diags
		}
		return appendDiag(diags, file, tokens[start], "invalid operand for "+op+": "+typ)
	}
	op := topLevelBinaryOperator(tokens, start, end)
	if op < 0 {
		return appendCompositeLiteralSubexpressionDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, op, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, op, sigs, importFuncs, localStructs, structs, importMethods)
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, op+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, op+1, end, sigs, importFuncs, localStructs, structs, importMethods)
	left := expressionRawTypeWithCallsAndTypes(tokens, start, op, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	right := expressionRawTypeWithCallsAndTypes(tokens, op+1, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	leftNorm := normalizeNamedType(left, typeNames)
	rightNorm := normalizeNamedType(right, typeNames)
	leftNil := expressionIsNilLiteral(tokens, start, op)
	rightNil := expressionIsNilLiteral(tokens, op+1, end)
	if msg := nilComparisonDiagnostic(tokens[op].Text, leftNorm, rightNorm, leftNil, rightNil); msg != "" {
		return appendDiag(diags, file, tokens[op], msg)
	}
	if structComparisonUnsupported(tokens, start, op, op+1, end, tokens[op].Text, leftNorm, rightNorm, structs, typeNames, importFuncs, importValues) {
		return appendDiag(diags, file, tokens[op], "struct comparisons are not supported: "+leftNorm)
	}
	if leftNil || rightNil {
		return diags
	}
	if left == "" || right == "" || binaryOperandsAssignable(tokens, start, op, op+1, end, tokens[op].Text, left, right, typeNames) {
		return diags
	}
	return appendDiag(diags, file, tokens[op], "invalid operands for "+tokens[op].Text+": "+left+" and "+right)
}

func nilComparisonDiagnostic(op string, left string, right string, leftNil bool, rightNil bool) string {
	if !leftNil && !rightNil {
		return ""
	}
	leftText := operandDiagnosticType(left, leftNil)
	rightText := operandDiagnosticType(right, rightNil)
	if op != "==" && op != "!=" {
		if (leftNil && (rightNil || right != "")) || (rightNil && left != "") {
			return "invalid operands for " + op + ": " + leftText + " and " + rightText
		}
		return ""
	}
	if leftNil && rightNil {
		return "invalid operands for " + op + ": nil and nil"
	}
	if leftNil {
		if right != "" && !isNilableType(right) {
			return "invalid operands for " + op + ": nil and " + right
		}
		return ""
	}
	if rightNil && left != "" && !isNilableType(left) {
		return "invalid operands for " + op + ": " + left + " and nil"
	}
	return ""
}

func operandDiagnosticType(typ string, isNil bool) string {
	if isNil {
		return "nil"
	}
	return typ
}

func isNilableType(typ string) bool {
	if typ == "error" {
		return true
	}
	if lowerableInterfaceParamRawType(typ) {
		return true
	}
	return strings.HasPrefix(typ, "*") || isSliceTypeName(typ)
}

func expressionIsNilLiteral(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionIsNilLiteral(tokens, start+1, close)
		}
	}
	return start+1 == end && tokens[start].Text == "nil"
}

func structComparisonUnsupported(tokens []scan.Token, leftStart int, leftEnd int, rightStart int, rightEnd int, op string, left string, right string, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue) bool {
	if op != "==" && op != "!=" {
		return false
	}
	if left == "" || left != right {
		return false
	}
	index := structFieldSetIndex(structs, left)
	if index < 0 {
		return false
	}
	if !structComparisonOperandLowerable(tokens, leftStart, leftEnd, importFuncs, importValues) || !structComparisonOperandLowerable(tokens, rightStart, rightEnd, importFuncs, importValues) {
		return true
	}
	return !structFieldSetComparisonLowerable(structs[index], structs, typeNames, nil)
}

func structComparisonOperandLowerable(tokens []scan.Token, start int, end int, importFuncs []importedFunction, importValues []importedValue) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return structComparisonOperandLowerable(tokens, start+1, close, importFuncs, importValues)
		}
	}
	if singleIdentifierExpression(tokens, start, end) != "" {
		return true
	}
	if compositeLiteralType(tokens, start, end) != "" {
		return true
	}
	if structComparisonOperandIsDirectCall(tokens, start, end) {
		return true
	}
	if importedValueRawType(tokens, start, end, importValues) != "" {
		return true
	}
	return importedCallResultType(tokens, start, end, importFuncs) != ""
}

func structComparisonOperandIsDirectCall(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	open := -1
	if start+2 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "(" {
		open = start + 1
	} else if start+4 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "(" {
		open = start + 3
	}
	if open < 0 {
		return false
	}
	close := findClose(tokens, open, "(", ")")
	return close == end-1
}

func structFieldSetComparisonLowerable(info structFieldSet, structs []structFieldSet, typeNames []localValueType, seen []string) bool {
	if len(info.fieldTypes) == 0 {
		return true
	}
	if containsString(seen, info.name) {
		return false
	}
	seen = appendStringUniqueCheck(cloneStrings(seen), info.name)
	for i := 0; i < len(info.fieldTypes); i++ {
		if !structComparisonFieldTypeLowerable(info.fieldTypes[i], structs, typeNames, seen) {
			return false
		}
	}
	return true
}

func structComparisonFieldTypeLowerable(value localValueType, structs []structFieldSet, typeNames []localValueType, seen []string) bool {
	raw := localValueRawType(value)
	if strings.HasPrefix(raw, "*") {
		return true
	}
	if structComparisonFixedArrayFieldTypeLowerable(raw, typeNames) {
		return true
	}
	if index := structFieldSetIndex(structs, value.typ); index >= 0 {
		return structFieldSetComparisonLowerable(structs[index], structs, typeNames, seen)
	}
	if raw != value.typ {
		return false
	}
	switch value.typ {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32":
		return true
	}
	return false
}

func structComparisonFixedArrayFieldTypeLowerable(raw string, typeNames []localValueType) bool {
	info, ok := fixedArrayTypeInfoFromType(raw, typeNames)
	if !ok {
		return false
	}
	elem := normalizeNamedType(info.elem, typeNames)
	if strings.HasPrefix(elem, "*") {
		return true
	}
	switch elem {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32":
		return true
	}
	return false
}

func appendCompositeLiteralDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendCompositeLiteralDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, nil, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendCompositeLiteralSubexpressionDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	for i := start; i < end; i++ {
		if tokens[i].Text != "{" {
			continue
		}
		typeStart := compositeLiteralTypeStartBeforeOpen(tokens, i)
		if typeStart < start {
			continue
		}
		close := findClose(tokens, i, "{", "}")
		if close < 0 || close >= end {
			continue
		}
		diags = appendCompositeLiteralDiagnosticsWithTypes(diags, file, tokens, typeStart, close+1, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		i = close
	}
	return diags
}

func compositeLiteralTypeStartBeforeOpen(tokens []scan.Token, open int) int {
	if open <= 0 || open >= len(tokens) {
		return -1
	}
	start := open - 1
	if tokens[start].Kind != scan.Ident {
		return -1
	}
	if start >= 2 && tokens[start-1].Text == "." && tokens[start-2].Kind == scan.Ident {
		start = start - 2
	}
	if start >= 1 && tokens[start-1].Text == "*" {
		start--
	}
	for start >= 2 && tokens[start-2].Text == "[" && tokens[start-1].Text == "]" {
		start = start - 2
	}
	return start
}

func appendCompositeLiteralDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if compositeLiteralStartIsSliceElementTypeFragment(tokens, start) {
		return diags
	}
	if compositeLiteralStartIsMapElementTypeFragment(tokens, start) {
		return diags
	}
	open := compositeLiteralOpen(tokens, start, end)
	if open < 0 {
		return diags
	}
	close := findClose(tokens, open, "{", "}")
	if close != end-1 {
		return diags
	}
	typ := typeTextInRange(tokens, start, open)
	if typ == "" {
		return diags
	}
	values := expressionRanges(tokens, open+1, close)
	if isSliceTypeName(typ) {
		elem := sliceElementType(typ)
		diags = appendSliceCompositeLiteralKeyDiagnostics(diags, file, tokens, values, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		for i := 0; i < len(values); i++ {
			valueStart := compositeLiteralValueStart(tokens, values[i].start, values[i].end)
			diags = appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags, file, tokens, elem, "", "", valueStart, values[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		return diags
	}
	structIndex := structFieldSetIndex(structs, typ)
	if structIndex < 0 {
		return diags
	}
	fieldOrdinal := 0
	var seenFields []string
	hasKeyed := false
	hasPositional := false
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(tokens, value.start, value.end, ":")
		fieldName := ""
		want := ""
		valueStart := value.start
		if colon >= 0 {
			hasKeyed = true
			valueStart = colon + 1
			if value.start < colon && tokens[value.start].Kind == scan.Ident {
				fieldName = tokens[value.start].Text
				if containsString(seenFields, fieldName) {
					diags = appendDiag(diags, file, tokens[value.start], "duplicate field: "+typ+"."+fieldName)
				}
				seenFields = appendStringUniqueCheck(seenFields, fieldName)
				want = structFieldRawType(structs, typ, fieldName)
				if want == "" {
					diags = appendDiag(diags, file, tokens[value.start], "unknown field: "+typ+"."+fieldName)
				}
			}
		} else if fieldOrdinal < len(structs[structIndex].fieldTypes) {
			hasPositional = true
			field := structs[structIndex].fieldTypes[fieldOrdinal]
			fieldName = field.name
			want = localValueRawType(field)
			fieldOrdinal++
		} else {
			hasPositional = true
			diags = appendDiag(diags, file, tokens[value.start], "too many values in composite literal: "+typ)
		}
		diags = appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags, file, tokens, want, typ, fieldName, valueStart, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if hasKeyed && hasPositional {
		diags = appendDiag(diags, file, tokens[open], "cannot mix keyed and positional composite literal values: "+typ)
	}
	return diags
}

func compositeLiteralStartIsSliceElementTypeFragment(tokens []scan.Token, start int) bool {
	if start >= 2 && tokens[start-2].Text == "[" && tokens[start-1].Text == "]" {
		return true
	}
	if start >= 3 && tokens[start-3].Text == "[" && tokens[start-2].Text == "]" && tokens[start-1].Text == "*" {
		return true
	}
	return false
}

func compositeLiteralStartIsMapElementTypeFragment(tokens []scan.Token, start int) bool {
	if start <= 0 || start >= len(tokens) || tokens[start-1].Text != "]" {
		return false
	}
	keyOpen := findOpen(tokens, start-1, "[", "]")
	return keyOpen > 0 && tokens[keyOpen-1].Text == "map"
}

func appendCompositeLiteralExpectedTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, want string, owner string, fieldName string, start int, end int, localTypes []localValueType, structs []structFieldSet, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags, file, tokens, want, owner, fieldName, start, end, localTypes, structs, nil, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, want string, owner string, fieldName string, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return diags
	}
	wantRaw := want
	wantNorm := normalizeNamedType(want, typeNames)
	if expressionIsImplicitCompositeLiteral(tokens, start, end) {
		if compositeTypeSupported(wantNorm, structs) {
			return appendImplicitCompositeLiteralDiagnosticsWithTypes(diags, file, tokens, wantNorm, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		if strings.HasPrefix(wantNorm, "struct{") {
			return diags
		}
		if wantRaw != "" {
			return appendCompositeLiteralTypeMismatch(diags, file, tokens[start], wantRaw, "composite literal", owner, fieldName)
		}
	}
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	got := expressionRawTypeWithCallsAndTypes(tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if wantRaw != "" && got != "" && !expressionAssignableToType(tokens, start, end, wantRaw, got, typeNames) {
		return appendCompositeLiteralTypeMismatch(diags, file, tokens[start], wantRaw, got, owner, fieldName)
	}
	return diags
}

func appendImplicitCompositeLiteralDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, typ string, start int, end int, localTypes []localValueType, structs []structFieldSet, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	return appendImplicitCompositeLiteralDiagnosticsWithTypes(diags, file, tokens, typ, start, end, localTypes, structs, nil, importFuncs, importValues, localStructs, sigs, importMethods)
}

func appendImplicitCompositeLiteralDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, typ string, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	close := findClose(tokens, start, "{", "}")
	if close != end-1 {
		return diags
	}
	typ = normalizeNamedType(typ, typeNames)
	values := expressionRanges(tokens, start+1, close)
	if isSliceTypeName(typ) {
		elem := sliceElementType(typ)
		diags = appendSliceCompositeLiteralKeyDiagnostics(diags, file, tokens, values, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		for i := 0; i < len(values); i++ {
			valueStart := compositeLiteralValueStart(tokens, values[i].start, values[i].end)
			diags = appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags, file, tokens, elem, "", "", valueStart, values[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		return diags
	}
	structIndex := structFieldSetIndex(structs, typ)
	if structIndex < 0 {
		return diags
	}
	fieldOrdinal := 0
	var seenFields []string
	hasKeyed := false
	hasPositional := false
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(tokens, value.start, value.end, ":")
		fieldName := ""
		want := ""
		valueStart := value.start
		if colon >= 0 {
			hasKeyed = true
			valueStart = colon + 1
			if value.start < colon && tokens[value.start].Kind == scan.Ident {
				fieldName = tokens[value.start].Text
				if containsString(seenFields, fieldName) {
					diags = appendDiag(diags, file, tokens[value.start], "duplicate field: "+typ+"."+fieldName)
				}
				seenFields = appendStringUniqueCheck(seenFields, fieldName)
				want = structFieldRawType(structs, typ, fieldName)
				if want == "" {
					diags = appendDiag(diags, file, tokens[value.start], "unknown field: "+typ+"."+fieldName)
				}
			}
		} else if fieldOrdinal < len(structs[structIndex].fieldTypes) {
			hasPositional = true
			field := structs[structIndex].fieldTypes[fieldOrdinal]
			fieldName = field.name
			want = localValueRawType(field)
			fieldOrdinal++
		} else {
			hasPositional = true
			diags = appendDiag(diags, file, tokens[value.start], "too many values in composite literal: "+typ)
		}
		diags = appendCompositeLiteralExpectedTypeDiagnosticsWithTypes(diags, file, tokens, want, typ, fieldName, valueStart, value.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	if hasKeyed && hasPositional {
		diags = appendDiag(diags, file, tokens[start], "cannot mix keyed and positional composite literal values: "+typ)
	}
	return diags
}

func appendNamedSliceCompositeLiteralRangeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if tokens[i].Text == "{" && namedSliceCompositeLiteralTypeAtOpen(tokens, i, typeNames) != "" {
			diags = appendNamedSliceCompositeLiteralOpenDiagnostics(diags, file, tokens, i, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
		if paren < 0 || brack < 0 || brace < 0 {
			return diags
		}
	}
	return diags
}

func appendNamedSliceCompositeLiteralOpenDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, open int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	typ := namedSliceCompositeLiteralTypeAtOpen(tokens, open, typeNames)
	if typ == "" {
		return diags
	}
	close := findClose(tokens, open, "{", "}")
	if close < 0 {
		return diags
	}
	elem := sliceElementType(typ)
	values := expressionRanges(tokens, open+1, close)
	diags = appendSliceCompositeLiteralKeyDiagnostics(diags, file, tokens, values, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	for i := 0; i < len(values); i++ {
		valueStart := compositeLiteralValueStart(tokens, values[i].start, values[i].end)
		diags = appendNamedSliceCompositeLiteralValueDiagnostics(diags, file, tokens, elem, valueStart, values[i].end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	}
	return diags
}

func namedSliceCompositeLiteralTypeAtOpen(tokens []scan.Token, open int, typeNames []localValueType) string {
	if open <= 0 || open >= len(tokens) || tokens[open].Text != "{" {
		return ""
	}
	typeStart := compositeLiteralTypeStartBeforeOpen(tokens, open)
	if compositeLiteralStartIsMapElementTypeFragment(tokens, typeStart) {
		return ""
	}
	prev := tokens[open-1]
	if prev.Kind != scan.Ident {
		return ""
	}
	typ := normalizeNamedType(prev.Text, typeNames)
	if !isSliceTypeName(typ) {
		return ""
	}
	return typ
}

func appendNamedSliceCompositeLiteralValueDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, want string, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return diags
	}
	wantRaw := want
	wantNorm := normalizeNamedType(want, typeNames)
	if expressionIsImplicitCompositeLiteral(tokens, start, end) {
		if compositeTypeSupported(wantNorm, structs) {
			return appendImplicitCompositeLiteralDiagnosticsWithTypes(diags, file, tokens, wantNorm, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		if strings.HasPrefix(wantNorm, "struct{") {
			return diags
		}
		if wantRaw != "" {
			return appendCompositeLiteralTypeMismatch(diags, file, tokens[start], wantRaw, "composite literal", "", "")
		}
	}
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	got := expressionRawTypeWithCallsAndTypes(tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if wantRaw != "" && got != "" && !expressionAssignableToType(tokens, start, end, wantRaw, got, typeNames) {
		return appendCompositeLiteralTypeMismatch(diags, file, tokens[start], wantRaw, got, "", "")
	}
	return diags
}

func expressionIsImplicitCompositeLiteral(tokens []scan.Token, start int, end int) bool {
	if start >= end || tokens[start].Text != "{" {
		return false
	}
	close := findClose(tokens, start, "{", "}")
	return close == end-1
}

func compositeTypeSupported(typ string, structs []structFieldSet) bool {
	return isSliceTypeName(typ) || structFieldSetIndex(structs, typ) >= 0
}

func appendCompositeLiteralTypeMismatch(diags Diagnostics, file parse.File, token scan.Token, want string, got string, owner string, fieldName string) Diagnostics {
	if owner != "" && fieldName != "" {
		return appendDiag(diags, file, token, "composite literal field type mismatch: "+owner+"."+fieldName+" has "+want+", got "+got)
	}
	return appendDiag(diags, file, token, "composite literal element type mismatch: want "+want+", got "+got)
}

func compositeLiteralValueStart(tokens []scan.Token, start int, end int) int {
	colon := findTopLevelToken(tokens, start, end, ":")
	if colon >= 0 {
		return colon + 1
	}
	return start
}

func appendSliceCompositeLiteralKeyDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, values []expressionRange, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	var seen []string
	nextIndex := int64(0)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(tokens, value.start, value.end, ":")
		if colon < 0 {
			key := strconv.FormatInt(nextIndex, 10)
			if containsString(seen, key) {
				diags = appendDiag(diags, file, tokens[value.start], "duplicate index in slice literal: "+key)
				continue
			}
			seen = append(seen, key)
			nextIndex++
			continue
		}
		keyStart, keyEnd := trimExpressionRange(tokens, value.start, colon)
		if keyStart >= keyEnd {
			continue
		}
		diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, keyStart, keyEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendSingleValueCallDiagnostics(diags, file, tokens, keyStart, keyEnd, sigs, importFuncs, localStructs, structs, importMethods)
		got := expressionSimpleTypeWithCallsAndTypes(tokens, keyStart, keyEnd, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if got != "" && !isIntegerTypeName(got) {
			diags = appendDiag(diags, file, tokens[keyStart], "slice literal index must be integer, got "+got)
			continue
		}
		key, ok := simpleIntegerLiteralKey(tokens, keyStart, keyEnd)
		if !ok {
			diags = appendDiag(diags, file, tokens[keyStart], "slice literal index must be integer constant")
			continue
		}
		if strings.HasPrefix(key, "-") {
			diags = appendDiag(diags, file, tokens[keyStart], "slice literal index must be non-negative")
			continue
		}
		if containsString(seen, key) {
			diags = appendDiag(diags, file, tokens[keyStart], "duplicate index in slice literal: "+key)
			continue
		}
		seen = append(seen, key)
		keyValue, err := strconv.ParseInt(key, 10, 64)
		if err == nil {
			nextIndex = keyValue + 1
		}
	}
	return diags
}

func simpleIntegerLiteralKey(tokens []scan.Token, start int, end int) (string, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	sign := ""
	if start < end && (tokens[start].Text == "-" || tokens[start].Text == "+") {
		sign = tokens[start].Text
		start++
	}
	if start+1 != end || tokens[start].Kind != scan.Number {
		return "", false
	}
	text := tokens[start].Text
	if numberLiteralType(text) != "int" {
		return "", false
	}
	value, err := strconv.ParseInt(text, 0, 64)
	if err != nil {
		return "", false
	}
	if sign == "-" {
		value = -value
	}
	return strconv.FormatInt(value, 10), true
}

func appendSingleValueCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod) Diagnostics {
	if singleCallResultCount(tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods) <= 1 {
		return diags
	}
	return appendDiag(diags, file, tokens[start], "multiple-value call in single-value context")
}

func singleArgumentExpandsToParameters(tokens []scan.Token, start int, end int, sig funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod, sigs []funcSignature) bool {
	if sig.variadic {
		return false
	}
	count := singleCallResultCount(tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	return count > 1 && count == sig.params
}

func appendLocalStructSelectorDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, info localStructType, structs []structFieldSet) Diagnostics {
	if pos+2 >= len(tokens) || tokens[pos+1].Text != "." {
		return diags
	}
	if pos > 0 && tokens[pos-1].Text == "." {
		return diags
	}
	if pos+3 < len(tokens) && tokens[pos+3].Text == "(" {
		return diags
	}
	if info.typeName == "" {
		return diags
	}
	typeName := structFieldSetName(info)
	for memberPos := pos + 2; memberPos < len(tokens); memberPos += 2 {
		member := tokens[memberPos]
		if member.Kind != scan.Ident || memberPos-1 < 0 || tokens[memberPos-1].Text != "." {
			return diags
		}
		if structFieldSetIndex(structs, typeName) < 0 {
			return diags
		}
		fieldTyp := selectorFieldType(structs, typeName, member.Text)
		if fieldTyp == "" {
			return appendDiag(diags, file, member, "unknown field: "+typeName+"."+member.Text)
		}
		if memberPos+1 >= len(tokens) || tokens[memberPos+1].Text != "." {
			return diags
		}
		nextType := fieldStructTypeName(structs, typeName, fieldTyp)
		if nextType == "" {
			return diags
		}
		typeName = nextType
	}
	return diags
}

func appendLocalValueSelectorDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) Diagnostics {
	if pos+2 >= len(tokens) || tokens[pos+1].Text != "." {
		return diags
	}
	if pos > 0 && tokens[pos-1].Text == "." {
		return diags
	}
	typ := localValueTypeNameAt(locals, tokens[pos].Text, pos)
	if typ == "" {
		return diags
	}
	if typ == "struct" || strings.HasPrefix(typ, "struct{") {
		return diags
	}
	if normalizeNamedType(typ, typeNames) == "error" && tokens[pos+2].Text == "Error" {
		return diags
	}
	base := selectorBaseType(normalizeNamedType(typ, typeNames))
	if base == "struct" || strings.HasPrefix(base, "struct{") {
		return diags
	}
	if structFieldSetIndex(structs, base) >= 0 {
		return diags
	}
	return appendDiag(diags, file, tokens[pos+2], "cannot select field on non-struct value: "+typ)
}

func appendImportedValueSelectorDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, values []importedValue, structs []structFieldSet, typeNames []localValueType) (Diagnostics, bool) {
	if pos+4 >= len(tokens) || tokens[pos+1].Text != "." || tokens[pos+3].Text != "." {
		return diags, false
	}
	if pos > 0 && tokens[pos-1].Text == "." {
		return diags, false
	}
	typ := importedValueType(tokens, pos, pos+3, values)
	if typ == "" {
		return diags, false
	}
	typeName := importedValueSelectorStructTypeName(tokens[pos].Text, typ, structs, typeNames)
	if typeName == "" {
		return appendDiag(diags, file, tokens[pos+4], "cannot select field on non-struct value: "+typ), true
	}
	for memberPos := pos + 4; memberPos < len(tokens); memberPos += 2 {
		member := tokens[memberPos]
		if member.Kind != scan.Ident || memberPos-1 < 0 || tokens[memberPos-1].Text != "." {
			return diags, true
		}
		if structFieldSetIndex(structs, typeName) < 0 {
			return diags, true
		}
		fieldTyp := selectorFieldType(structs, typeName, member.Text)
		if fieldTyp == "" {
			return appendDiag(diags, file, member, "unknown field: "+typeName+"."+member.Text), true
		}
		if memberPos+1 >= len(tokens) || tokens[memberPos+1].Text != "." {
			return diags, true
		}
		nextType := fieldStructTypeName(structs, typeName, fieldTyp)
		if nextType == "" {
			return diags, true
		}
		typeName = nextType
	}
	return diags, true
}

func appendLocalStructIndexDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, locals []localStructType) Diagnostics {
	if pos+1 >= len(tokens) || tokens[pos+1].Text != "[" {
		return diags
	}
	if start, _, ok := pureMapAliasIndexExpressionRange(tokens, pos); ok && start == pos {
		return diags
	}
	info, ok := localStructTypeInfo(locals, tokens[pos].Text)
	if !ok {
		return diags
	}
	return appendDiag(diags, file, tokens[pos+1], "cannot index struct value: "+localStructDisplayName(info))
}

func appendLocalStructDerefDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, locals []localStructType) Diagnostics {
	if !startsUnaryDeref(tokens, pos) {
		return diags
	}
	info, ok := localStructTypeInfo(locals, tokens[pos+1].Text)
	if !ok || info.pointer {
		return diags
	}
	return appendDiag(diags, file, tokens[pos], "cannot dereference non-pointer: "+info.typeName)
}

func appendLocalValueIndexDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	if pos+1 >= len(tokens) || tokens[pos+1].Text != "[" {
		return diags
	}
	if start, _, ok := pureMapAliasIndexExpressionRange(tokens, pos); ok && start == pos {
		return diags
	}
	typ := localValueTypeNameAt(locals, tokens[pos].Text, pos)
	if typ == "" || typeSupportsIndex(typ) {
		open := pos + 1
		close := findClose(tokens, open, "[", "]")
		if close < 0 {
			return diags
		}
		colon := findTopLevelToken(tokens, open+1, close, ":")
		if colon < 0 {
			return appendIndexBoundTypeDiagnostics(diags, file, tokens, open+1, close, "index", locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		normalized := normalizeNamedType(typ, typeNames)
		secondColon := fullSliceSecondColon(tokens, open)
		highEnd := close
		if secondColon >= 0 {
			highEnd = secondColon
			if normalized == "string" {
				diags = appendDiag(diags, file, tokens[secondColon], "full slice expressions require slice, got string")
			}
			diags = appendIndexBoundTypeDiagnostics(diags, file, tokens, secondColon+1, close, "slice bound", locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		}
		diags = appendIndexBoundTypeDiagnostics(diags, file, tokens, open+1, colon, "slice bound", locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		diags = appendIndexBoundTypeDiagnostics(diags, file, tokens, colon+1, highEnd, "slice bound", locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		return diags
	}
	return appendDiag(diags, file, tokens[pos+1], "cannot index non-indexable value: "+typ)
}

func appendIndexBoundTypeDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, name string, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return diags
	}
	diags = appendExpressionOperandDiagnosticsWithTypes(diags, file, tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	diags = appendSingleValueCallDiagnostics(diags, file, tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	got := expressionSimpleTypeWithCallsAndTypes(tokens, start, end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if got == "" || isIntegerTypeName(got) {
		return diags
	}
	return appendDiag(diags, file, tokens[start], name+" must be integer, got "+got)
}

func appendLocalValueDerefDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, locals []localValueType, structs []localStructType) Diagnostics {
	if !startsUnaryDeref(tokens, pos) {
		return diags
	}
	if _, ok := localStructTypeInfo(structs, tokens[pos+1].Text); ok {
		return diags
	}
	typ := localValueTypeNameAt(locals, tokens[pos+1].Text, pos)
	if typ == "" || strings.HasPrefix(typ, "*") {
		return diags
	}
	return appendDiag(diags, file, tokens[pos], "cannot dereference non-pointer: "+typ)
}

func typeSupportsIndex(typ string) bool {
	return typ == "string" || strings.HasPrefix(typ, "[]")
}

func appendLocalStructExpressionDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, start int, end int, locals []localStructType) Diagnostics {
	for i := start; i < end; i++ {
		if tokens[i].Text == "*" {
			diags = appendLocalStructDerefDiagnostics(diags, file, tokens, i, locals)
			continue
		}
		if tokens[i].Kind == scan.Ident && i+1 < end && tokens[i+1].Text == "[" && localStructTypeName(locals, tokens[i].Text) != "" {
			diags = appendLocalStructIndexDiagnostics(diags, file, tokens, i, locals)
		}
	}
	return diags
}

func startsUnaryDeref(tokens []scan.Token, pos int) bool {
	if pos < 0 || pos+1 >= len(tokens) || tokens[pos].Text != "*" || tokens[pos+1].Kind != scan.Ident {
		return false
	}
	if pos+2 < len(tokens) && tokens[pos+2].Text == "." {
		return false
	}
	if pos == 0 {
		return true
	}
	prev := tokens[pos-1]
	if prev.Line != tokens[pos].Line {
		return true
	}
	if prev.Text == ")" || prev.Text == "]" || (prev.Kind == scan.Ident && !isKeyword(prev.Text)) || prev.Kind == scan.Number || prev.Kind == scan.String || prev.Kind == scan.Char {
		return false
	}
	return true
}

func localStructDisplayName(info localStructType) string {
	name := info.typeName
	if info.qualifier != "" {
		name = info.qualifier + "." + info.typeName
	}
	if info.pointer {
		return "*" + name
	}
	return name
}

func structFieldExists(structs []structFieldSet, typeName string, field string) bool {
	return structFieldType(structs, typeName, field) != ""
}

func structFieldType(structs []structFieldSet, typeName string, field string) string {
	index := structFieldSetIndex(structs, typeName)
	if index < 0 {
		return ""
	}
	for i := 0; i < len(structs[index].fieldTypes); i++ {
		value := structs[index].fieldTypes[i]
		if value.name == field {
			return value.typ
		}
	}
	if containsString(structs[index].fields, field) {
		return "?"
	}
	return ""
}

func structSelectorFieldType(structs []structFieldSet, typeName string, field string) string {
	typ := structFieldType(structs, typeName, field)
	if typ != "" {
		return typ
	}
	value, ok := promotedStructField(structs, typeName, field)
	if !ok {
		return ""
	}
	return value.typ
}

func selectorFieldType(structs []structFieldSet, typeName string, field string) string {
	return structSelectorFieldType(structs, typeName, field)
}

func structFieldRawType(structs []structFieldSet, typeName string, field string) string {
	index := structFieldSetIndex(structs, typeName)
	if index < 0 {
		return ""
	}
	for i := 0; i < len(structs[index].fieldTypes); i++ {
		value := structs[index].fieldTypes[i]
		if value.name == field {
			return localValueRawType(value)
		}
	}
	return ""
}

func structSelectorFieldRawType(structs []structFieldSet, typeName string, field string) string {
	raw := structFieldRawType(structs, typeName, field)
	if raw != "" {
		return raw
	}
	value, ok := promotedStructField(structs, typeName, field)
	if !ok {
		return ""
	}
	return localValueRawType(value)
}

func selectorFieldRawType(structs []structFieldSet, typeName string, field string) string {
	return structSelectorFieldRawType(structs, typeName, field)
}

func promotedStructField(structs []structFieldSet, typeName string, field string) (localValueType, bool) {
	value, count := promotedStructFieldIn(structs, typeName, field, nil)
	return value, count == 1
}

func promotedStructFieldIn(structs []structFieldSet, typeName string, field string, seen []string) (localValueType, int) {
	if containsString(seen, typeName) {
		return localValueType{}, 0
	}
	seen = append(seen, typeName)
	index := structFieldSetIndex(structs, typeName)
	if index < 0 {
		return localValueType{}, 0
	}
	var found localValueType
	count := 0
	fields := structs[index].fieldTypes
	for i := 0; i < len(fields); i++ {
		embedded := fields[i]
		if !embedded.embedded {
			continue
		}
		embeddedType := fieldStructTypeName(structs, typeName, embedded.typ)
		if embeddedType == "" {
			continue
		}
		if exact := structFieldValue(structs, embeddedType, field); exact.name != "" {
			found = exact
			count++
			continue
		}
		nested, nestedCount := promotedStructFieldIn(structs, embeddedType, field, seen)
		if nestedCount > 0 {
			found = nested
			count += nestedCount
		}
	}
	return found, count
}

func structFieldValue(structs []structFieldSet, typeName string, field string) localValueType {
	index := structFieldSetIndex(structs, typeName)
	if index < 0 {
		return localValueType{}
	}
	for i := 0; i < len(structs[index].fieldTypes); i++ {
		value := structs[index].fieldTypes[i]
		if value.name == field {
			return value
		}
	}
	return localValueType{}
}

func structFieldSetName(info localStructType) string {
	if info.qualifier != "" {
		return info.qualifier + "." + info.typeName
	}
	return info.typeName
}

func fieldStructTypeName(structs []structFieldSet, ownerType string, fieldTyp string) string {
	fieldTyp = selectorBaseType(fieldTyp)
	if fieldTyp == "" || strings.HasPrefix(fieldTyp, "[]") {
		return ""
	}
	if structFieldSetIndex(structs, fieldTyp) >= 0 {
		return fieldTyp
	}
	qualifier := ownerTypeQualifier(ownerType)
	if qualifier != "" {
		qualified := qualifier + "." + fieldTyp
		if structFieldSetIndex(structs, qualified) >= 0 {
			return qualified
		}
	}
	return ""
}

func selectorBaseType(typ string) string {
	for strings.HasPrefix(typ, "*") {
		typ = typ[1:]
	}
	return typ
}

func ownerTypeQualifier(typeName string) string {
	dot := strings.IndexByte(typeName, '.')
	if dot < 0 {
		return ""
	}
	return typeName[:dot]
}

func functionSignatures(file parse.File) []funcSignature {
	typeNames := fileNamedTypeUnderlyings(file)
	structs := fileStructTypesWithTypes(file, typeNames)
	return functionSignaturesWithTypes(file, structs, typeNames)
}

func functionSignaturesWithTypes(file parse.File, structs []structFieldSet, typeNames []localValueType) []funcSignature {
	var sigs []funcSignature
	tokens := file.Tokens
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "func" || decl.Name == "" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		name := tokenIndexAt(tokens, int(decl.NameTok.Start))
		if name < 0 || name+1 >= len(tokens) {
			continue
		}
		paramsOpen := name + 1
		if tokens[paramsOpen].Text != "(" {
			continue
		}
		paramsClose := findClose(tokens, paramsOpen, "(", ")")
		if paramsClose < 0 {
			continue
		}
		bodyOpen := functionBodyOpenAfterParams(tokens, paramsClose, decl.End)
		body := bodyOpen
		if body < 0 {
			body = tokenIndexBefore(tokens, decl.End)
		}
		var erasedParams []int
		if bodyOpen >= 0 {
			bodyClose := findClose(tokens, bodyOpen, "{", "}")
			if bodyClose < 0 {
				bodyClose = tokenIndexBefore(tokens, decl.End)
			}
			if bodyClose > bodyOpen {
				erasedParams = unusedInterfaceParamIndexesForSignature(tokens, paramsOpen+1, paramsClose, bodyOpen, bodyClose)
			}
		}
		sigs = append(sigs, funcSignature{
			name:            decl.Name,
			params:          parameterCount(tokens, paramsOpen+1, paramsClose),
			results:         resultCount(tokens, paramsClose+1, body),
			paramTypes:      parameterTypeNames(tokens, paramsOpen+1, paramsClose, structs, typeNames),
			resultTypes:     resultTypeNames(tokens, paramsClose+1, body, structs, typeNames),
			callbackParams:  functionParamSignatures(tokens, paramsOpen+1, paramsClose, structs, typeNames),
			erasedParams:    erasedParams,
			receiverType:    methodReceiverTypeName(tokens, decl),
			pointerReceiver: methodReceiverIsPointer(tokens, decl),
			namedResults:    namedResultNamesAfterParams(tokens, paramsClose, body),
			variadic:        parameterListIsVariadic(tokens, paramsOpen+1, paramsClose),
		})
	}
	return sigs
}

func methodReceiverTypeName(tokens []scan.Token, decl parse.Decl) string {
	if !decl.Receiver {
		return ""
	}
	start := tokenIndexAt(tokens, decl.Start)
	if start < 0 || start+1 >= len(tokens) || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close < 0 {
		return ""
	}
	name := ""
	for i := start + 2; i < close; i++ {
		if tokens[i].Kind == scan.Ident {
			name = tokens[i].Text
		}
	}
	return name
}

func methodReceiverIsPointer(tokens []scan.Token, decl parse.Decl) bool {
	if !decl.Receiver {
		return false
	}
	start := tokenIndexAt(tokens, decl.Start)
	if start < 0 || start+1 >= len(tokens) || tokens[start+1].Text != "(" {
		return false
	}
	close := findClose(tokens, start+1, "(", ")")
	if close < 0 {
		return false
	}
	for i := start + 2; i < close; i++ {
		if tokens[i].Text == "*" {
			return true
		}
	}
	return false
}

func isLocalMethodValueUse(tokens []scan.Token, pos int, receiver localStructType, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) bool {
	if pos < 0 || pos+2 >= len(tokens) || tokens[pos+1].Text != "." || tokens[pos+2].Kind != scan.Ident {
		return false
	}
	if pos+3 < len(tokens) && tokens[pos+3].Text == "(" {
		return false
	}
	if receiver.typeName == "" {
		return false
	}
	methodName := tokens[pos+2].Text
	if _, ok := methodSignatureForStructType(receiver, methodName, sigs, importMethods); ok {
		return true
	}
	_, ok := promotedMethodSignature(structs, structFieldSetName(receiver), methodName, sigs, importMethods)
	return ok
}

func methodExistsForType(sigs []funcSignature, typeName string, methodName string) bool {
	for i := 0; i < len(sigs); i++ {
		if sigs[i].receiverType == typeName && sigs[i].name == methodName {
			return true
		}
	}
	return false
}

func importedMethodExists(methods []importedMethod, qualifier string, typeName string, methodName string) bool {
	return importedMethodIndex(methods, qualifier, typeName, methodName) >= 0
}

func importedMethodIndex(methods []importedMethod, qualifier string, typeName string, methodName string) int {
	for i := 0; i < len(methods); i++ {
		method := methods[i]
		if method.qualifier == qualifier && method.typeName == typeName && method.name == methodName {
			return i
		}
	}
	return -1
}

func importedMethodsForFile(file parse.File, packages []load.Package) []importedMethod {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var methods []importedMethod
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		files := packages[pkgIndex].Files
		typeNames := packageNamedTypeUnderlyings(files)
		structs := packageStructTypesWithTypes(files, typeNames)
		sigs := packageFunctionSignaturesWithTypes(files, structs, typeNames)
		for sigIndex := 0; sigIndex < len(sigs); sigIndex++ {
			sig := sigs[sigIndex]
			if sig.receiverType == "" || !isExported(sig.name) {
				continue
			}
			sig = qualifyImportedSignatureTypes(sig, importedTypeQualifier(localName), typeNames, structs)
			methods = append(methods, importedMethod{qualifier: localName, typeName: sig.receiverType, name: sig.name, sig: sig})
		}
	}
	return methods
}

func importedFunctionsForFile(file parse.File, packages []load.Package) []importedFunction {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var funcs []importedFunction
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		files := packages[pkgIndex].Files
		typeNames := packageNamedTypeUnderlyings(files)
		structs := packageStructTypesWithTypes(files, typeNames)
		sigs := packageFunctionSignaturesWithTypes(files, structs, typeNames)
		for sigIndex := 0; sigIndex < len(sigs); sigIndex++ {
			sig := sigs[sigIndex]
			if sig.receiverType != "" || !isExported(sig.name) {
				continue
			}
			sig = qualifyImportedSignatureTypes(sig, importedTypeQualifier(localName), typeNames, structs)
			funcs = append(funcs, importedFunction{qualifier: localName, importPath: imp.Path, sig: sig})
		}
	}
	return funcs
}

func qualifyImportedSignatureTypes(sig funcSignature, qualifier string, typeNames []localValueType, structs []structFieldSet) funcSignature {
	for i := 0; i < len(sig.paramTypes); i++ {
		sig.paramTypes[i] = qualifyImportedSignatureType(qualifier, typeNames, structs, sig.paramTypes[i])
	}
	for i := 0; i < len(sig.resultTypes); i++ {
		sig.resultTypes[i] = qualifyImportedSignatureType(qualifier, typeNames, structs, sig.resultTypes[i])
	}
	return sig
}

func qualifyImportedSignatureType(qualifier string, typeNames []localValueType, structs []structFieldSet, typ string) string {
	typ = qualifyImportedDefinedType(qualifier, typeNames, typ)
	return qualifyImportedStructFieldType(qualifier, structs, typ)
}

func importedValuesForFile(file parse.File, packages []load.Package) []importedValue {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var values []importedValue
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		valueTypes := packageTopLevelValueTypesWithImports(packages[pkgIndex].Files, packages)
		for valueIndex := 0; valueIndex < len(valueTypes); valueIndex++ {
			value := valueTypes[valueIndex]
			if value.typ == "" || !isExported(value.name) {
				continue
			}
			typeQualifier := importedTypeQualifier(localName)
			values = append(values, importedValue{qualifier: localName, name: value.name, typ: value.typ, raw: qualifyImportedDefinedType(typeQualifier, packageNamedTypeUnderlyings(packages[pkgIndex].Files), localValueRawType(value))})
		}
	}
	return values
}

func importedNamedTypeUnderlyingsForFile(file parse.File, packages []load.Package) []localValueType {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var values []localValueType
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		typeNames := packageNamedTypeUnderlyings(packages[pkgIndex].Files)
		structs := packageStructTypesWithTypes(packages[pkgIndex].Files, typeNames)
		for i := 0; i < len(typeNames); i++ {
			name := typeNames[i].name
			if name == "" {
				continue
			}
			typeQualifier := importedTypeQualifier(localName)
			underlying := qualifyImportedDefinedType(typeQualifier, typeNames, typeNames[i].typ)
			underlying = qualifyImportedStructFieldType(typeQualifier, structs, underlying)
			raw := qualifyImportedDefinedType(typeQualifier, typeNames, localValueRawType(typeNames[i]))
			raw = qualifyImportedStructFieldType(typeQualifier, structs, raw)
			values = setLocalValueTypeWithRaw(values, importedQualifiedName(localName, name), underlying, raw)
		}
	}
	return values
}

func importedTypeQualifier(localName string) string {
	if localName == "." {
		return ""
	}
	return localName
}

func importedQualifiedName(localName string, name string) string {
	if localName == "." {
		return name
	}
	return localName + "." + name
}

func qualifyImportedDefinedType(qualifier string, typeNames []localValueType, typ string) string {
	if qualifier == "" || typ == "" {
		return typ
	}
	if strings.HasPrefix(typ, "*") {
		inner := qualifyImportedDefinedType(qualifier, typeNames, typ[1:])
		if inner == "" {
			return typ
		}
		return "*" + inner
	}
	if strings.HasPrefix(typ, "[]") {
		inner := qualifyImportedDefinedType(qualifier, typeNames, typ[2:])
		if inner == "" {
			return typ
		}
		return "[]" + inner
	}
	if strings.HasPrefix(typ, "[") {
		close := strings.IndexByte(typ, ']')
		if close > 0 && close+1 < len(typ) {
			inner := qualifyImportedDefinedType(qualifier, typeNames, typ[close+1:])
			if inner != "" {
				return typ[:close+1] + inner
			}
		}
	}
	if localValueTypeName(typeNames, typ) != "" {
		return qualifier + "." + typ
	}
	return typ
}

func importedStructTypesForFile(file parse.File, packages []load.Package) []structFieldSet {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var structs []structFieldSet
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		files := packages[pkgIndex].Files
		typeNames := packageNamedTypeUnderlyings(files)
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			parsed, err := parsedLoadFile(files[fileIndex])
			if err != nil {
				continue
			}
			fileStructs := fileStructTypesWithTypes(parsed, typeNames)
			for structIndex := 0; structIndex < len(fileStructs); structIndex++ {
				value := exportedStructFieldSet(importedTypeQualifier(localName), typeNames, fileStructs, fileStructs[structIndex])
				if value.name == "" {
					continue
				}
				structs = appendStructFieldSets(structs, []structFieldSet{value})
			}
		}
	}
	return structs
}

func exportedStructFieldSet(qualifier string, typeNames []localValueType, packageStructs []structFieldSet, value structFieldSet) structFieldSet {
	if value.name == "" || !isExported(value.name) {
		return structFieldSet{}
	}
	var out structFieldSet
	out.name = value.name
	if qualifier != "" {
		out.name = qualifier + "." + value.name
	}
	for i := 0; i < len(value.fieldTypes); i++ {
		field := value.fieldTypes[i]
		if !isExported(field.name) {
			continue
		}
		raw := localValueRawType(field)
		field.typ = qualifyImportedStructFieldType(qualifier, packageStructs, field.typ)
		field.raw = qualifyImportedStructFieldType(qualifier, packageStructs, qualifyImportedDefinedType(qualifier, typeNames, raw))
		out.fieldTypes = append(out.fieldTypes, field)
		out.fields = appendStringUniqueCheck(out.fields, field.name)
	}
	return out
}

func qualifyImportedStructFieldType(qualifier string, packageStructs []structFieldSet, typ string) string {
	if qualifier == "" || typ == "" {
		return typ
	}
	if strings.HasPrefix(typ, "*") {
		inner := qualifyImportedStructFieldType(qualifier, packageStructs, typ[1:])
		if inner == "" {
			return typ
		}
		return "*" + inner
	}
	if strings.HasPrefix(typ, "[]") {
		inner := qualifyImportedStructFieldType(qualifier, packageStructs, typ[2:])
		if inner == "" {
			return typ
		}
		return "[]" + inner
	}
	if strings.HasPrefix(typ, "[") {
		close := strings.IndexByte(typ, ']')
		if close > 0 && close+1 < len(typ) {
			inner := qualifyImportedStructFieldType(qualifier, packageStructs, typ[close+1:])
			if inner != "" {
				return typ[:close+1] + inner
			}
		}
	}
	if strings.Contains(typ, ".") {
		return typ
	}
	if isExported(typ) && structFieldSetIndex(packageStructs, typ) >= 0 {
		return qualifier + "." + typ
	}
	return typ
}

func fileTopLevelValueTypes(file parse.File) []localValueType {
	return fileTopLevelValueTypesWithTypeNames(file, fileNamedTypeUnderlyings(file))
}

func fileTopLevelValueTypesWithTypeNames(file parse.File, typeNames []localValueType) []localValueType {
	return fileTopLevelValueTypesWithKnown(file, nil, nil, nil, nil, typeNames)
}

func fileTopLevelValueTypesWithKnown(file parse.File, knownValues []localValueType, knownStructValues []localStructType, knownStructs []structFieldSet, importValues []importedValue, typeNames []localValueType) []localValueType {
	var out []localValueType
	tokens := file.Tokens
	structs := fileStructTypesWithTypes(file, typeNames)
	if len(knownStructs) > 0 {
		structs = knownStructs
	}
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" && decl.Kind != "const" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(tokens) && tokens[start+1].Text == "(" {
			close := findClose(tokens, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specStart := start + 2
			for i := specStart; i <= close; i++ {
				if i == close || tokens[i].Text == ";" {
					out = appendTopLevelValueSpecTypes(out, tokens, specStart, i, structs, knownValues, knownStructValues, importValues, typeNames)
					specStart = i + 1
					continue
				}
				if tokens[i].Line != tokens[specStart].Line {
					out = appendTopLevelValueSpecTypes(out, tokens, specStart, i, structs, knownValues, knownStructValues, importValues, typeNames)
					specStart = i
				}
			}
			continue
		}
		end := tokenIndexBefore(tokens, decl.End) + 1
		if end <= start+1 {
			continue
		}
		out = appendTopLevelValueSpecTypes(out, tokens, start+1, end, structs, knownValues, knownStructValues, importValues, typeNames)
	}
	return out
}

func appendTopLevelValueSpecTypes(out []localValueType, tokens []scan.Token, start int, end int, structs []structFieldSet, knownValues []localValueType, knownStructValues []localStructType, importValues []importedValue, typeNames []localValueType) []localValueType {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return out
	}
	eq := findTopLevelToken(tokens, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	var names []string
	typeStart := -1
	for i := start; i < lhsEnd; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendStringUniqueCheck(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return out
	}
	if typeStart >= 0 {
		raw := rawTypeTextInRange(tokens, typeStart, lhsEnd)
		typ := normalizeNamedType(raw, typeNames)
		if typ != "" {
			for i := 0; i < len(names); i++ {
				out = setLocalValueTypeWithRaw(out, names[i], typ, raw)
			}
			return out
		}
	}
	if eq < 0 {
		return out
	}
	rhs := expressionRanges(tokens, eq+1, end)
	known := appendLocalValueTypes(cloneLocalValueTypes(knownValues), out)
	for i := 0; i < len(names) && i < len(rhs); i++ {
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, known, structs, typeNames, nil, importValues, knownStructValues, nil, nil)
		raw := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, known, structs, typeNames, nil, importValues, knownStructValues, nil, nil)
		if typ != "" {
			out = setLocalValueTypeWithRaw(out, names[i], typ, raw)
			known = setLocalValueTypeWithRaw(known, names[i], typ, raw)
		}
	}
	return out
}

func packageIndexByImportPath(packages []load.Package, importPath string) int {
	for i := 0; i < len(packages); i++ {
		if packages[i].ImportPath == importPath {
			return i
		}
	}
	return -1
}

func fileTopLevelNames(file parse.File) []string {
	var names []string
	for i := 0; i < len(file.Decls); i++ {
		declNames := packageLevelDeclNames(file.Decls[i])
		for j := 0; j < len(declNames); j++ {
			name := declNames[j]
			if name != "" && name != "_" && !containsString(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

func packageTopLevelNames(files []load.File) []string {
	names := make([]string, 0, packageLevelNameCapacity(files))
	for i := 0; i < len(files); i++ {
		file := files[i].Parsed
		for j := 0; j < len(file.Decls); j++ {
			declNames := packageLevelDeclNames(file.Decls[j])
			for k := 0; k < len(declNames); k++ {
				name := declNames[k]
				if name != "" && name != "_" && !containsString(names, name) {
					names = append(names, name)
				}
			}
		}
	}
	return names
}

func fileTopLevelNamesOfKind(file parse.File, kind string) []string {
	var names []string
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != kind {
			continue
		}
		declNames := packageLevelDeclNames(decl)
		for j := 0; j < len(declNames); j++ {
			name := declNames[j]
			if name != "" && name != "_" && !containsString(names, name) {
				names = append(names, name)
			}
		}
	}
	return names
}

func packageTopLevelNamesOfKind(files []load.File, kind string) []string {
	names := make([]string, 0, packageLevelNameCapacity(files))
	for i := 0; i < len(files); i++ {
		file := files[i].Parsed
		for j := 0; j < len(file.Decls); j++ {
			decl := file.Decls[j]
			if decl.Kind != kind {
				continue
			}
			declNames := packageLevelDeclNames(decl)
			for k := 0; k < len(declNames); k++ {
				name := declNames[k]
				if name != "" && name != "_" && !containsString(names, name) {
					names = append(names, name)
				}
			}
		}
	}
	return names
}

func packageFunctionSignatures(files []load.File) []funcSignature {
	typeNames := packageNamedTypeUnderlyings(files)
	return packageFunctionSignaturesWithTypes(files, packageStructTypesWithTypes(files, typeNames), typeNames)
}

func packageFunctionSignaturesWithTypes(files []load.File, structs []structFieldSet, typeNames []localValueType) []funcSignature {
	size := 0
	for i := 0; i < len(files); i++ {
		size += len(files[i].Parsed.Decls)
	}
	sigs := make([]funcSignature, 0, size)
	for i := 0; i < len(files); i++ {
		sigs = append(sigs, functionSignaturesWithTypes(files[i].Parsed, structs, typeNames)...)
	}
	return sigs
}

func packageTopLevelValueTypes(files []load.File) []localValueType {
	return packageTopLevelValueTypesWithImports(files, nil)
}

func packageTopLevelValueTypesWithImports(files []load.File, packages []load.Package) []localValueType {
	return packageTopLevelValueTypesWithImportsAndTypes(files, packages, packageNamedTypeUnderlyings(files))
}

func packageTopLevelValueTypesWithImportsAndTypes(files []load.File, packages []load.Package, typeNames []localValueType) []localValueType {
	var values []localValueType
	structs := packageStructTypesWithTypes(files, typeNames)
	structValues := packageTopLevelStructValueTypes(files, structs)
	passes := len(files) + 2
	if passes < 2 {
		passes = 2
	}
	for pass := 0; pass < passes; pass++ {
		before := cloneLocalValueTypes(values)
		for i := 0; i < len(files); i++ {
			file, err := parsedLoadFile(files[i])
			if err != nil {
				continue
			}
			importValues := importedValuesForFile(file, packages)
			values = appendLocalValueTypes(values, fileTopLevelValueTypesWithKnown(file, values, structValues, structs, importValues, typeNames))
		}
		if localValueTypesEqual(before, values) {
			break
		}
	}
	return values
}

func localValueTypesEqual(a []localValueType, b []localValueType) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i].name != b[i].name || a[i].typ != b[i].typ || localValueRawType(a[i]) != localValueRawType(b[i]) {
			return false
		}
	}
	return true
}

func packageStructTypes(files []load.File) []structFieldSet {
	return packageStructTypesWithTypes(files, packageNamedTypeUnderlyings(files))
}

func packageStructTypesWithTypes(files []load.File, typeNames []localValueType) []structFieldSet {
	var structs []structFieldSet
	for i := 0; i < len(files); i++ {
		file, err := parsedLoadFile(files[i])
		if err != nil {
			continue
		}
		structs = appendStructFieldSets(structs, fileStructTypesWithTypes(file, typeNames))
	}
	return structs
}

func fileNamedTypeUnderlyings(file parse.File) []localValueType {
	return fileNamedTypeUnderlyingsWithKnown(file, nil)
}

func fileNamedTypeUnderlyingsWithKnown(file parse.File, known []localValueType) []localValueType {
	out := cloneLocalValueTypes(known)
	tokens := file.Tokens
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(tokens) && tokens[start+1].Text == "(" {
			close := findClose(tokens, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specStart := start + 2
			for j := specStart; j <= close; j++ {
				if j == close || tokens[j].Text == ";" || tokens[j].Line != tokens[specStart].Line {
					out = appendNamedTypeSpec(out, tokens, specStart, j)
					if j < close && tokens[j].Text == ";" {
						specStart = j + 1
					} else {
						specStart = j
					}
				}
			}
			continue
		}
		end := tokenIndexBefore(tokens, decl.End) + 1
		out = appendNamedTypeSpec(out, tokens, start+1, end)
	}
	return out
}

func functionLocalTypeDeclNames(tokens []scan.Token, start int, end int) []string {
	var names []string
	for i := start; i < end; i++ {
		if tokens[i].Text != "type" || !startsLocalTypeDeclToken(tokens, i, start) {
			continue
		}
		if i+1 < end && tokens[i+1].Text == "(" {
			close := findClose(tokens, i+1, "(", ")")
			if close < 0 || close > end {
				continue
			}
			ranges := localTypeSpecRanges(tokens, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				if spec.start < spec.end && tokens[spec.start].Kind == scan.Ident {
					names = appendStringUniqueCheck(names, tokens[spec.start].Text)
				}
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(tokens, i, end)
		if i+1 < specEnd && tokens[i+1].Kind == scan.Ident {
			names = appendStringUniqueCheck(names, tokens[i+1].Text)
		}
		if specEnd > i {
			i = specEnd - 1
		}
	}
	return names
}

func functionLocalNamedTypeUnderlyings(tokens []scan.Token, start int, end int, known []localValueType) []localValueType {
	out := cloneLocalValueTypes(known)
	before := len(out)
	for i := start; i < end; i++ {
		if tokens[i].Text != "type" || !startsLocalTypeDeclToken(tokens, i, start) {
			continue
		}
		if i+1 < end && tokens[i+1].Text == "(" {
			close := findClose(tokens, i+1, "(", ")")
			if close < 0 || close > end {
				continue
			}
			ranges := localTypeSpecRanges(tokens, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				out = appendNamedTypeSpec(out, tokens, spec.start, spec.end)
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(tokens, i, end)
		out = appendNamedTypeSpec(out, tokens, i+1, specEnd)
		if specEnd > i {
			i = specEnd - 1
		}
	}
	return out[before:]
}

func functionLocalStructTypes(tokens []scan.Token, start int, end int, typeNames []localValueType) []structFieldSet {
	var structs []structFieldSet
	for i := start; i < end; i++ {
		if tokens[i].Text != "type" || !startsLocalTypeDeclToken(tokens, i, start) {
			continue
		}
		if i+1 < end && tokens[i+1].Text == "(" {
			close := findClose(tokens, i+1, "(", ")")
			if close < 0 || close > end {
				continue
			}
			ranges := localTypeSpecRanges(tokens, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				structs = appendLocalStructTypeSpec(structs, tokens, ranges[rangeIndex].start, ranges[rangeIndex].end, typeNames)
			}
			i = close
			continue
		}
		specEnd := localTypeSingleSpecEnd(tokens, i, end)
		structs = appendLocalStructTypeSpec(structs, tokens, i+1, specEnd, typeNames)
		if specEnd > i {
			i = specEnd - 1
		}
	}
	return structs
}

func appendLocalStructTypeSpec(structs []structFieldSet, tokens []scan.Token, start int, end int, typeNames []localValueType) []structFieldSet {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start+3 >= end || tokens[start].Kind != scan.Ident {
		return structs
	}
	typeStart := start + 1
	if tokens[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || tokens[typeStart].Text != "struct" || tokens[typeStart+1].Text != "{" {
		return structs
	}
	close := findClose(tokens, typeStart+1, "{", "}")
	if close < 0 || close > end {
		return structs
	}
	fieldTypes := structFieldTypes(tokens, typeStart+1, close, typeNames)
	return appendStructFieldSets(structs, []structFieldSet{{name: tokens[start].Text, fields: structFieldNamesFromTypes(fieldTypes), fieldTypes: fieldTypes}})
}

func startsLocalTypeDeclToken(tokens []scan.Token, pos int, scopeStart int) bool {
	if pos < scopeStart || pos+1 >= len(tokens) {
		return false
	}
	if tokens[pos+1].Kind != scan.Ident && tokens[pos+1].Text != "(" {
		return false
	}
	if pos == 0 {
		return true
	}
	prev := tokens[pos-1]
	return prev.Text == "{" || prev.Text == "}" || prev.Text == ";" || prev.Line != tokens[pos].Line
}

func localTypeSingleSpecEnd(tokens []scan.Token, pos int, limit int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < len(tokens); i++ {
		if i >= limit || tokens[i].Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 {
			if tokens[i].Text == ";" || tokens[i].Text == "}" {
				return i
			}
			if i > pos+1 && tokens[i].Line > tokens[i-1].Line {
				return i
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return len(tokens)
}

func localTypeSpecRanges(tokens []scan.Token, start int, end int) []expressionRange {
	var ranges []expressionRange
	specStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if tokens[i].Text == ";" {
				if specStart < i {
					ranges = append(ranges, expressionRange{start: specStart, end: i})
				}
				specStart = i + 1
				continue
			}
			if i > specStart && tokens[i].Line > tokens[i-1].Line {
				ranges = append(ranges, expressionRange{start: specStart, end: i})
				specStart = i
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	if specStart < end {
		ranges = append(ranges, expressionRange{start: specStart, end: end})
	}
	return ranges
}

func packageNamedTypeUnderlyings(files []load.File) []localValueType {
	var values []localValueType
	passes := len(files) + 2
	if passes < 2 {
		passes = 2
	}
	for pass := 0; pass < passes; pass++ {
		before := cloneLocalValueTypes(values)
		for i := 0; i < len(files); i++ {
			file, err := parsedLoadFile(files[i])
			if err != nil {
				continue
			}
			values = appendLocalValueTypes(values, fileNamedTypeUnderlyingsWithKnown(file, values))
		}
		if localValueTypesEqual(before, values) {
			break
		}
	}
	return values
}

func appendNamedTypeSpec(out []localValueType, tokens []scan.Token, start int, end int) []localValueType {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end || tokens[start].Kind != scan.Ident {
		return out
	}
	name := tokens[start].Text
	typeStart := start + 1
	if typeStart < end && tokens[typeStart].Text == "=" {
		typeStart++
	}
	raw := rawTypeTextInRange(tokens, typeStart, end)
	typ := normalizeNamedType(raw, out)
	if typ == "" || typ == name {
		return out
	}
	if strings.HasPrefix(typ, "struct") || typ == "interface" || typ == "map" {
		return out
	}
	return setLocalValueTypeWithRaw(out, name, typ, raw)
}

func normalizeNamedType(typ string, typeNames []localValueType) string {
	if typ == "" {
		return typ
	}
	if strings.HasPrefix(typ, "*") {
		inner := normalizeNamedType(typ[1:], typeNames)
		if inner == "" {
			return typ
		}
		return "*" + inner
	}
	if strings.HasPrefix(typ, "[]") {
		inner := normalizeNamedType(typ[2:], typeNames)
		if inner == "" {
			return typ
		}
		return "[]" + inner
	}
	if strings.HasPrefix(typ, "[") {
		close := strings.IndexByte(typ, ']')
		if close > 1 && close+1 < len(typ) {
			inner := normalizeNamedType(typ[close+1:], typeNames)
			if inner != "" {
				return "[]" + inner
			}
		}
	}
	if len(typeNames) == 0 {
		return typ
	}
	underlying := localValueTypeName(typeNames, typ)
	if underlying == "" || underlying == typ {
		return typ
	}
	return normalizeNamedType(underlying, typeNames)
}

func packageTopLevelStructValueTypes(files []load.File, structs []structFieldSet) []localStructType {
	var values []localStructType
	for i := 0; i < len(files); i++ {
		file, err := parsedLoadFile(files[i])
		if err != nil {
			continue
		}
		values = appendLocalStructTypes(values, collectPackageTopLevelStructTypes(file, structs, fileImportNames(file)))
	}
	return values
}

func appendStructFieldSets(out []structFieldSet, values []structFieldSet) []structFieldSet {
	for i := 0; i < len(values); i++ {
		value := values[i]
		index := structFieldSetIndex(out, value.name)
		if index >= 0 {
			out[index] = value
			continue
		}
		out = append(out, value)
	}
	return out
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func appendLocalValueTypes(out []localValueType, values []localValueType) []localValueType {
	for i := 0; i < len(values); i++ {
		value := values[i]
		value.raw = localValueRawType(value)
		out = setLocalValueTypeEntry(out, value)
	}
	return out
}

func appendLocalStructTypes(out []localStructType, values []localStructType) []localStructType {
	for i := 0; i < len(values); i++ {
		out = setLocalStructTypeInfo(out, values[i].name, values[i])
	}
	return out
}

func cloneLocalValueTypes(values []localValueType) []localValueType {
	if len(values) == 0 {
		return nil
	}
	out := make([]localValueType, len(values))
	copy(out, values)
	return out
}

func cloneLocalStructTypes(values []localStructType) []localStructType {
	if len(values) == 0 {
		return nil
	}
	out := make([]localStructType, len(values))
	copy(out, values)
	return out
}

func fileStructTypes(file parse.File) []structFieldSet {
	return fileStructTypesWithTypes(file, fileNamedTypeUnderlyings(file))
}

func fileStructTypesWithTypes(file parse.File, typeNames []localValueType) []structFieldSet {
	var structs []structFieldSet
	tokens := file.Tokens
	for i := 0; i+2 < len(tokens); i++ {
		if tokens[i].Text != "type" {
			continue
		}
		if tokens[i+1].Text == "(" {
			close := findClose(tokens, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(tokens, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				structs = appendFileStructTypeSpec(structs, tokens, ranges[rangeIndex].start, ranges[rangeIndex].end, typeNames)
			}
			i = close
			continue
		}
		if tokens[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := localTypeSingleSpecEnd(tokens, i, len(tokens))
		structs = appendFileStructTypeSpec(structs, tokens, i+1, specEnd, typeNames)
		i = specEnd - 1
	}
	return structs
}

func appendFileStructTypeSpec(structs []structFieldSet, tokens []scan.Token, start int, end int, typeNames []localValueType) []structFieldSet {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start+2 >= end || tokens[start].Kind != scan.Ident {
		return structs
	}
	typeStart := start + 1
	if typeStart < end && tokens[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || tokens[typeStart].Text != "struct" || tokens[typeStart+1].Text != "{" {
		return structs
	}
	close := findClose(tokens, typeStart+1, "{", "}")
	if close < 0 || close > end {
		return structs
	}
	fieldTypes := structFieldTypes(tokens, typeStart+1, close, typeNames)
	return appendStructFieldSets(structs, []structFieldSet{{name: tokens[start].Text, fields: structFieldNamesFromTypes(fieldTypes), fieldTypes: fieldTypes}})
}

func structFieldNamesFromTypes(fields []localValueType) []string {
	var out []string
	for i := 0; i < len(fields); i++ {
		out = appendStringUniqueCheck(out, fields[i].name)
	}
	return out
}

func structFieldTypes(tokens []scan.Token, open int, close int, typeNames []localValueType) []localValueType {
	var fields []localValueType
	specStart := open + 1
	for i := open + 1; i <= close; i++ {
		if i == close || tokens[i].Text == ";" || tokens[i].Line != tokens[specStart].Line {
			fields = appendStructFieldTypes(fields, structFieldTypesInSpec(tokens, specStart, i, typeNames))
			specStart = i
			if i < close && tokens[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return fields
}

func appendStringSet(out []string, values []string) []string {
	for i := 0; i < len(values); i++ {
		out = appendStringUniqueCheck(out, values[i])
	}
	return out
}

func appendStructFieldTypes(out []localValueType, values []localValueType) []localValueType {
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.name == "" || value.typ == "" {
			continue
		}
		value.raw = localValueRawType(value)
		out = setLocalValueTypeEntry(out, value)
	}
	return out
}

func structFieldTypesInSpec(tokens []scan.Token, start int, end int, typeNames []localValueType) []localValueType {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	if field, ok := embeddedStructFieldType(tokens, start, end, typeNames); ok {
		return []localValueType{field}
	}
	if start >= end || tokens[start].Text == "*" {
		return nil
	}
	if tokens[start].Kind != scan.Ident {
		return nil
	}
	if start+1 >= end {
		return nil
	}
	var names []string
	typeStart := -1
	for i := start; i < end; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendStringUniqueCheck(names, tokens[i].Text)
			continue
		}
		if tokens[i].Text == "struct" || isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 || typeStart < 0 {
		return nil
	}
	typeEnd := end
	if typeEnd > typeStart && tokens[typeEnd-1].Kind == scan.String {
		typeEnd--
	}
	raw := rawTypeTextInRange(tokens, typeStart, typeEnd)
	typ := normalizeNamedType(raw, typeNames)
	if tokens[typeStart].Text == "struct" {
		raw = "struct"
		typ = "struct"
	}
	if typ == "" {
		return nil
	}
	var fields []localValueType
	for i := 0; i < len(names); i++ {
		fields = append(fields, localValueType{name: names[i], typ: typ, raw: raw})
	}
	return fields
}

func embeddedStructFieldType(tokens []scan.Token, start int, end int, typeNames []localValueType) (localValueType, bool) {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Kind == scan.String {
		end--
	}
	if start >= end {
		return localValueType{}, false
	}
	fieldName := ""
	typeStart := start
	if tokens[start].Text == "*" {
		typeStart = start + 1
	}
	if typeStart >= end || tokens[typeStart].Kind != scan.Ident {
		return localValueType{}, false
	}
	if typeStart+1 == end {
		fieldName = tokens[typeStart].Text
	} else if typeStart+3 == end && tokens[typeStart+1].Text == "." && tokens[typeStart+2].Kind == scan.Ident {
		fieldName = tokens[typeStart+2].Text
	} else {
		return localValueType{}, false
	}
	raw := rawTypeTextInRange(tokens, start, end)
	typ := normalizeNamedType(raw, typeNames)
	if raw == "" || typ == "" {
		return localValueType{}, false
	}
	return localValueType{name: fieldName, typ: typ, raw: raw, embedded: true}, true
}

func structFieldSetIndex(values []structFieldSet, name string) int {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return i
		}
	}
	return -1
}

func fileImportNames(file parse.File) []string {
	var names []string
	for i := 0; i < len(file.Imports); i++ {
		name := importLocalName(file.Imports[i])
		if name != "" && name != "." && name != "_" && !containsString(names, name) {
			names = append(names, name)
		}
	}
	return names
}

func collectFunctionLocalNames(tokens []scan.Token, start int, body int, close int) []string {
	var names []string
	names = collectSignatureLocalNames(tokens, start, body, names)
	names = collectNamedResultLocalNames(tokens, start, body, names)
	for i := body + 1; i < close; i++ {
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, body, i)
			names = collectAssignmentLeftNames(tokens, stmtStart, i, names)
			continue
		}
		if tokens[i].Text == "var" || tokens[i].Text == "const" {
			names = collectVarStatementNames(tokens, i, close, names)
		}
	}
	return names
}

func collectFunctionLocalStructTypes(tokens []scan.Token, start int, body int, close int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	locals = collectSignatureLocalStructTypes(tokens, start, body, structs, importNames, locals)
	for i := body + 1; i < close; i++ {
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, body, i)
			locals = collectShortDeclStructType(tokens, stmtStart, i, close, structs, importNames, locals)
			continue
		}
		if tokens[i].Text == "var" {
			locals = collectVarStatementStructType(tokens, i, close, structs, importNames, locals)
		}
	}
	return locals
}

func collectPackageTopLevelStructTypes(file parse.File, structs []structFieldSet, importNames []string) []localStructType {
	var locals []localStructType
	tokens := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(tokens) && tokens[start+1].Text == "(" {
			close := findClose(tokens, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specStart := start + 2
			for i := specStart; i <= close; i++ {
				if i == close || tokens[i].Text == ";" {
					locals = collectPackageValueSpecStructTypes(tokens, specStart, i, structs, importNames, locals)
					specStart = i + 1
					continue
				}
				if tokens[i].Line != tokens[specStart].Line {
					locals = collectPackageValueSpecStructTypes(tokens, specStart, i, structs, importNames, locals)
					specStart = i
				}
			}
			continue
		}
		end := tokenIndexBefore(tokens, decl.End) + 1
		locals = collectPackageValueSpecStructTypes(tokens, start+1, end, structs, importNames, locals)
	}
	return locals
}

func collectPackageValueSpecStructTypes(tokens []scan.Token, start int, end int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return locals
	}
	eq := findTopLevelToken(tokens, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	var names []string
	typeStart := -1
	for i := start; i < lhsEnd; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendStringUniqueCheck(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return locals
	}
	if typeStart >= 0 {
		info := structTypeInfoInRange(tokens, typeStart, lhsEnd, structs, importNames)
		if info.typeName != "" {
			for i := 0; i < len(names); i++ {
				locals = setLocalStructTypeInfo(locals, names[i], info)
			}
			return locals
		}
	}
	if eq < 0 {
		return locals
	}
	rhs := expressionRanges(tokens, eq+1, end)
	for i := 0; i < len(names) && i < len(rhs); i++ {
		info := compositeLiteralStructType(tokens, rhs[i].start, rhs[i].end, structs, importNames)
		if info.typeName == "" {
			continue
		}
		locals = setLocalStructTypeInfo(locals, names[i], info)
	}
	return locals
}

func collectFunctionLocalValueTypes(tokens []scan.Token, start int, body int, close int, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, locals []localValueType) []localValueType {
	locals = collectSignatureLocalValueTypes(tokens, start, body, structs, typeNames, locals)
	locals = collectNamedResultLocalValueTypes(tokens, start, body, structs, typeNames, locals)
	for i := body + 1; i < close; i++ {
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, body, i)
			locals = collectShortDeclValueTypes(tokens, body, stmtStart, i, close, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
			continue
		}
		if tokens[i].Text == "var" || tokens[i].Text == "const" {
			locals = collectValueDeclTypes(tokens, body, i, close, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
		}
	}
	return locals
}

func collectFunctionAliases(tokens []scan.Token, body int, close int, sigs []funcSignature, importFuncs []importedFunction, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, importMethods []importedMethod) []functionAlias {
	var aliases []functionAlias
	for i := body + 1; i < close; i++ {
		if tokens[i].Text == ":=" {
			aliases = collectShortDeclFunctionAliases(tokens, body, i, close, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods, aliases)
			continue
		}
		if tokens[i].Text == "var" {
			aliases = collectVarFunctionAliases(tokens, body, i, close, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods, aliases)
		}
	}
	return aliases
}

func appendFunctionParamAliases(aliases []functionAlias, tokens []scan.Token, start int, body int, close int, structs []structFieldSet, typeNames []localValueType) []functionAlias {
	paramsOpen := -1
	for i := start; i < body; i++ {
		if tokens[i].Text == "(" {
			paramsOpen = i
			break
		}
	}
	if paramsOpen < 0 {
		return aliases
	}
	paramsClose := findClose(tokens, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose > body {
		return aliases
	}
	callbacks := functionParamSignatures(tokens, paramsOpen+1, paramsClose, structs, typeNames)
	for i := 0; i < len(callbacks); i++ {
		callback := callbacks[i]
		if callback.name == "" {
			continue
		}
		sig := callback.sig
		sig.name = callback.name
		aliases = append(aliases, functionAlias{
			name:  callback.name,
			sig:   sig,
			start: int(tokens[body].Start),
			end:   int(tokens[close].End),
		})
	}
	return aliases
}

func collectShortDeclFunctionAliases(tokens []scan.Token, body int, assign int, limit int, sigs []funcSignature, importFuncs []importedFunction, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, importMethods []importedMethod, aliases []functionAlias) []functionAlias {
	stmtStart := simpleStatementStart(tokens, body, assign)
	if stmtStart < assign && (tokens[stmtStart].Text == "if" || tokens[stmtStart].Text == "for" || tokens[stmtStart].Text == "switch") {
		return aliases
	}
	stmtEnd := simpleStatementEnd(tokens, assign+1, limit)
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return aliases
	}
	scopeEnd := declarationScopeSourceEnd(tokens, body, assign, limit)
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		sig, ok := staticAliasTargetSignature(tokens, rhs[i].start, rhs[i].end, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods)
		if !ok {
			continue
		}
		staticCallback := staticCallbackAliasTarget(tokens, rhs[i].start, rhs[i].end, sigs, importFuncs, importMethods, localTypes, structs, typeNames, localStructs)
		sig.name = name
		sig.receiverType = ""
		aliases = append(aliases, functionAlias{name: name, sig: sig, start: int(tokens[assign].End), end: scopeEnd, staticCallback: staticCallback})
	}
	return aliases
}

func collectVarFunctionAliases(tokens []scan.Token, body int, pos int, limit int, sigs []funcSignature, importFuncs []importedFunction, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, importMethods []importedMethod, aliases []functionAlias) []functionAlias {
	if pos+1 >= limit || tokens[pos+1].Text == "(" || tokens[pos+1].Kind != scan.Ident {
		return aliases
	}
	stmtEnd := simpleStatementEnd(tokens, pos+1, limit)
	eq := findTopLevelToken(tokens, pos+1, stmtEnd, "=")
	if eq < 0 {
		return aliases
	}
	if eq != pos+2 {
		return aliases
	}
	sig, ok := staticAliasTargetSignature(tokens, eq+1, stmtEnd, sigs, importFuncs, localTypes, structs, typeNames, localStructs, importMethods)
	if !ok {
		return aliases
	}
	staticCallback := staticCallbackAliasTarget(tokens, eq+1, stmtEnd, sigs, importFuncs, importMethods, localTypes, structs, typeNames, localStructs)
	name := tokens[pos+1].Text
	sig.name = name
	sig.receiverType = ""
	scopeEnd := declarationScopeSourceEnd(tokens, body, pos, limit)
	return append(aliases, functionAlias{name: name, sig: sig, start: int(tokens[eq].End), end: scopeEnd, staticCallback: staticCallback})
}

func declarationScopeSourceEnd(tokens []scan.Token, body int, decl int, limit int) int {
	end := declarationScopeEnd(tokens, body, decl, limit)
	if end >= 0 && end < len(tokens) {
		return int(tokens[end].Start)
	}
	if limit >= 0 && limit < len(tokens) {
		return int(tokens[limit].Start)
	}
	return maxSourcePosition()
}

func appendFunctionAliasSignatures(sigs []funcSignature, aliases []functionAlias) []funcSignature {
	if len(aliases) == 0 {
		return sigs
	}
	out := make([]funcSignature, 0, len(sigs)+len(aliases))
	for i := len(aliases) - 1; i >= 0; i-- {
		sig := aliases[i].sig
		sig.staticAlias = true
		sig.staticCallback = aliases[i].staticCallback
		out = append(out, sig)
	}
	out = append(out, sigs...)
	return out
}

func staticCallbackAliasTarget(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, importMethods []importedMethod, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType) bool {
	if _, ok := functionLiteralAliasTargetSignature(tokens, start, end, structs, typeNames); ok {
		return true
	}
	if _, ok := functionAliasTargetSignature(tokens, start, end, sigs, importFuncs); ok {
		return true
	}
	if _, ok := methodValueAliasTargetSignature(tokens, start, end, localTypes, structs, typeNames, localStructs, sigs, importMethods); ok {
		return true
	}
	if _, ok := compositeLiteralMethodValueAliasTargetSignature(tokens, start, end, structs, sigs, importMethods); ok {
		return true
	}
	_, ok := methodExpressionAliasTargetSignature(tokens, start, end, sigs, importMethods)
	return ok
}

func functionAliasTargetSignature(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return funcSignature{}, false
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		index := funcSignatureIndex(sigs, tokens[start].Text)
		if index >= 0 {
			return sigs[index], true
		}
		index = importedFunctionIndex(importFuncs, ".", tokens[start].Text)
		if index >= 0 {
			return importFuncs[index].sig, true
		}
		return funcSignature{}, false
	}
	if start+3 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		index := importedFunctionIndex(importFuncs, tokens[start].Text, tokens[start+2].Text)
		if index >= 0 {
			return importFuncs[index].sig, true
		}
	}
	return funcSignature{}, false
}

func staticAliasTargetSignature(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, importMethods []importedMethod) (funcSignature, bool) {
	if sig, ok := functionLiteralAliasTargetSignature(tokens, start, end, structs, typeNames); ok {
		return sig, true
	}
	if sig, ok := functionAliasTargetSignature(tokens, start, end, sigs, importFuncs); ok {
		return sig, true
	}
	if sig, ok := methodValueAliasTargetSignature(tokens, start, end, localTypes, structs, typeNames, localStructs, sigs, importMethods); ok {
		return sig, true
	}
	if sig, ok := methodExpressionAliasTargetSignature(tokens, start, end, sigs, importMethods); ok {
		return sig, true
	}
	if sig, ok := compositeLiteralMethodValueAliasTargetSignature(tokens, start, end, structs, sigs, importMethods); ok {
		return sig, true
	}
	return funcSignature{}, false
}

func methodExpressionAliasTargetSignature(tokens []scan.Token, start int, end int, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start+3 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		call := methodExpressionCallInfo{typeName: tokens[start].Text, methodName: tokens[start+2].Text, methodNameAt: start + 2}
		sig, ok := methodExpressionCallSignature(call, sigs, importMethods)
		if !ok || sig.pointerReceiver {
			return funcSignature{}, false
		}
		return methodExpressionCallFunctionSignature(call, sig), true
	}
	if start+5 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "." && tokens[start+4].Kind == scan.Ident {
		call := methodExpressionCallInfo{qualifier: tokens[start].Text, typeName: tokens[start+2].Text, methodName: tokens[start+4].Text, methodNameAt: start + 4}
		sig, ok := methodExpressionCallSignature(call, sigs, importMethods)
		if !ok || sig.pointerReceiver {
			return funcSignature{}, false
		}
		return methodExpressionCallFunctionSignature(call, sig), true
	}
	return funcSignature{}, false
}

func functionLiteralAliasTargetSignature(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	literal, ok := functionLiteralInfoAt(tokens, start, end, structs, typeNames)
	if !ok || literal.start != start || literal.end != end {
		return funcSignature{}, false
	}
	return literal.sig, true
}

func functionLiteralInfoAt(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (functionLiteralInfo, bool) {
	if start < 0 || start+2 >= len(tokens) || start >= end || tokens[start].Text != "func" || tokens[start+1].Text != "(" {
		return functionLiteralInfo{}, false
	}
	paramsOpen := start + 1
	paramsClose := findClose(tokens, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= end {
		return functionLiteralInfo{}, false
	}
	bodyOpen := -1
	for i := paramsClose + 1; i < end; i++ {
		if tokens[i].Text == "{" {
			bodyOpen = i
			break
		}
		if tokens[i].Text == ";" {
			return functionLiteralInfo{}, false
		}
	}
	if bodyOpen < 0 {
		return functionLiteralInfo{}, false
	}
	bodyClose := findClose(tokens, bodyOpen, "{", "}")
	if bodyClose < 0 || bodyClose >= end {
		return functionLiteralInfo{}, false
	}
	sig := funcSignature{
		params:         parameterCount(tokens, paramsOpen+1, paramsClose),
		results:        resultCount(tokens, paramsClose+1, bodyOpen),
		paramTypes:     parameterTypeNames(tokens, paramsOpen+1, paramsClose, structs, typeNames),
		resultTypes:    resultTypeNames(tokens, paramsClose+1, bodyOpen, structs, typeNames),
		callbackParams: functionParamSignatures(tokens, paramsOpen+1, paramsClose, structs, typeNames),
		namedResults:   namedResultNamesAfterParams(tokens, paramsClose, bodyOpen),
		variadic:       parameterListIsVariadic(tokens, paramsOpen+1, paramsClose),
	}
	return functionLiteralInfo{
		start:       start,
		paramsOpen:  paramsOpen,
		paramsClose: paramsClose,
		bodyOpen:    bodyOpen,
		bodyClose:   bodyClose,
		end:         bodyClose + 1,
		sig:         sig,
	}, true
}

func functionLiteralAliasInitializerAt(tokens []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Text != "func" {
		return false
	}
	literal, ok := functionLiteralInfoAt(tokens, pos, len(tokens), nil, nil)
	if !ok {
		return false
	}
	body := enclosingFunctionBodyOpen(tokens, pos)
	if body < 0 {
		return false
	}
	close := findClose(tokens, body, "{", "}")
	if close < 0 {
		close = len(tokens)
	}
	stmtStart := simpleStatementStart(tokens, body, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, close)
	if literal.end > stmtEnd {
		return false
	}
	eq := findTopLevelToken(tokens, stmtStart, stmtEnd, ":=")
	if eq >= 0 {
		rhs := expressionRanges(tokens, eq+1, stmtEnd)
		for i := 0; i < len(rhs); i++ {
			start, end := trimExpressionRange(tokens, rhs[i].start, rhs[i].end)
			if start == pos && end == literal.end {
				return true
			}
		}
		return false
	}
	if tokens[stmtStart].Text != "var" {
		return false
	}
	eq = findTopLevelToken(tokens, stmtStart+1, stmtEnd, "=")
	if eq < 0 {
		return false
	}
	rhs := expressionRanges(tokens, eq+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		start, end := trimExpressionRange(tokens, rhs[i].start, rhs[i].end)
		if start == pos && end == literal.end {
			return true
		}
	}
	return false
}

func functionLiteralDirectCallAt(tokens []scan.Token, pos int) bool {
	_, ok := functionLiteralDirectCallInfoAt(tokens, pos, len(tokens), nil, nil)
	return ok
}

func functionLiteralStaticCallbackArgumentAt(tokens []scan.Token, pos int, sigs []funcSignature) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Text != "func" {
		return false
	}
	literal, ok := functionLiteralInfoAt(tokens, pos, len(tokens), nil, nil)
	if !ok {
		return false
	}
	body := enclosingFunctionBodyOpen(tokens, pos)
	if body < 0 {
		return false
	}
	close := findClose(tokens, body, "{", "}")
	if close < 0 {
		close = len(tokens)
	}
	for i := body + 1; i < close; i++ {
		if tokens[i].Kind != scan.Ident || i+1 >= close || tokens[i+1].Text != "(" || isSelectorMember(tokens, i) {
			continue
		}
		sigIndex := funcSignatureIndex(sigs, tokens[i].Text)
		if sigIndex < 0 || len(sigs[sigIndex].callbackParams) == 0 {
			continue
		}
		callClose := findClose(tokens, i+1, "(", ")")
		if callClose < 0 || callClose > close {
			continue
		}
		args := expressionRanges(tokens, i+2, callClose)
		for callbackIndex := 0; callbackIndex < len(sigs[sigIndex].callbackParams); callbackIndex++ {
			callback := sigs[sigIndex].callbackParams[callbackIndex]
			if callback.index < 0 || callback.index >= len(args) {
				continue
			}
			start, end := trimExpressionRange(tokens, args[callback.index].start, args[callback.index].end)
			if start == literal.start && end == literal.end {
				return true
			}
		}
	}
	return false
}

func functionLiteralDirectCallInfoAt(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (functionLiteralDirectCallInfo, bool) {
	literal, ok := functionLiteralInfoAt(tokens, start, end, structs, typeNames)
	if !ok {
		return functionLiteralDirectCallInfo{}, false
	}
	callOpen := literal.end
	if callOpen >= end || callOpen >= len(tokens) || tokens[callOpen].Text != "(" {
		return functionLiteralDirectCallInfo{}, false
	}
	callClose := findClose(tokens, callOpen, "(", ")")
	if callClose < 0 || callClose >= end {
		return functionLiteralDirectCallInfo{}, false
	}
	return functionLiteralDirectCallInfo{literal: literal, callOpen: callOpen, callClose: callClose}, true
}

func functionLiteralDirectCallSignature(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	call, ok := functionLiteralDirectCallInfoAt(tokens, start, end, structs, typeNames)
	if !ok || call.literal.start != start || call.callClose != end-1 {
		return funcSignature{}, false
	}
	return call.literal.sig, true
}

func appendFunctionLiteralDirectCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, call functionLiteralDirectCallInfo, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	argCount := expressionListCount(tokens, call.callOpen+1, call.callClose)
	sig := call.literal.sig
	if (!sig.variadic && argCount != sig.params) || (sig.variadic && argCount < sig.params-1) {
		return appendDiag(diags, file, tokens[call.literal.start], "argument count mismatch in call to function literal")
	}
	return appendCallArgumentTypeDiagnostics(diags, file, tokens, "function literal", call.callOpen+1, call.callClose, sig, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
}

func functionLiteralDirectCallResultType(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) string {
	sig, ok := functionLiteralDirectCallSignature(tokens, start, end, structs, typeNames)
	if !ok || sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func appendFunctionLiteralCaptureDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, literal functionLiteralInfo, outerLocals []string) Diagnostics {
	return appendFunctionLiteralCaptureDiagnosticsWithTypes(diags, file, tokens, literal, outerLocals, nil, false)
}

func appendFunctionLiteralUnknownCaptureDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, literal functionLiteralInfo, outerLocals []string, localTypes []localValueType) Diagnostics {
	return appendFunctionLiteralCaptureDiagnosticsWithTypes(diags, file, tokens, literal, outerLocals, localTypes, true)
}

func appendFunctionLiteralCaptureDiagnosticsWithTypes(diags Diagnostics, file parse.File, tokens []scan.Token, literal functionLiteralInfo, outerLocals []string, localTypes []localValueType, allowTyped bool) Diagnostics {
	if len(outerLocals) == 0 {
		return diags
	}
	reported := make([]string, 0, 2)
	captures := functionLiteralCaptureNames(tokens, literal, outerLocals)
	for captureIndex := 0; captureIndex < len(captures); captureIndex++ {
		tokIndex := captures[captureIndex]
		tok := tokens[tokIndex]
		if containsString(reported, tok.Text) {
			continue
		}
		if allowTyped && localValueRawTypeName(localTypes, tok.Text) != "" {
			continue
		}
		reported = append(reported, tok.Text)
		diags = appendDiag(diags, file, tok, "closures are not supported: "+tok.Text)
	}
	return diags
}

func functionLiteralTypedCaptureNames(tokens []scan.Token, literal functionLiteralInfo, outerLocals []string, localTypes []localValueType) []string {
	var out []string
	captures := functionLiteralCaptureNames(tokens, literal, outerLocals)
	for captureIndex := 0; captureIndex < len(captures); captureIndex++ {
		name := tokens[captures[captureIndex]].Text
		if localValueRawTypeName(localTypes, name) == "" || containsString(out, name) {
			continue
		}
		out = append(out, name)
	}
	return out
}

func functionLiteralCaptureNames(tokens []scan.Token, literal functionLiteralInfo, outerLocals []string) []int {
	if len(outerLocals) == 0 {
		return nil
	}
	var out []int
	for i := literal.bodyOpen + 1; i < literal.bodyClose; i++ {
		tok := tokens[i]
		if tok.Kind != scan.Ident {
			continue
		}
		if !containsString(outerLocals, tok.Text) || containsString(sameScopeNamesBefore(tokens, literal.start, literal.bodyOpen, i), tok.Text) {
			continue
		}
		if isKeyword(tok.Text) || isCompositeKey(tokens, i) || isSelectorMember(tokens, i) || functionLiteralShortDeclTargetAt(tokens, literal, i) {
			continue
		}
		out = append(out, i)
	}
	return out
}

func functionLiteralShortDeclTargetAt(tokens []scan.Token, literal functionLiteralInfo, pos int) bool {
	for i := pos + 1; i < literal.bodyClose && tokens[i].Line == tokens[pos].Line; i++ {
		if tokens[i].Text == ":=" {
			stmtStart := simpleStatementStart(tokens, literal.bodyOpen, i)
			lhs := expressionRanges(tokens, stmtStart, i)
			for lhsIndex := 0; lhsIndex < len(lhs); lhsIndex++ {
				if pos >= lhs[lhsIndex].start && pos < lhs[lhsIndex].end {
					return true
				}
			}
			return false
		}
		if tokens[i].Text == "=" || tokens[i].Text == ";" || tokens[i].Text == "{" || tokens[i].Text == "}" {
			return false
		}
	}
	return false
}

func captureAssignmentTargetToken(tokens []scan.Token, lhs []expressionRange, name string) int {
	for i := 0; i < len(lhs); i++ {
		for j := lhs[i].start; j < lhs[i].end; j++ {
			if tokens[j].Kind != scan.Ident || tokens[j].Text != name {
				continue
			}
			if isSelectorMember(tokens, j) || isCompositeKey(tokens, j) {
				continue
			}
			return j
		}
	}
	return -1
}

func compositeLiteralMethodValueAliasTargetSignature(tokens []scan.Token, start int, end int, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	value, ok := compositeLiteralMethodValueInfoAt(tokens, start, end)
	if !ok {
		return funcSignature{}, false
	}
	sig, promotedReceiver, promoted, ok := compositeLiteralMethodSignatureInfo(value, structs, sigs, importMethods)
	if !ok {
		return funcSignature{}, false
	}
	if sig.pointerReceiver && !value.addressed && (!promoted || !localValueTypeIsPointer(promotedReceiver)) {
		return funcSignature{}, false
	}
	return sig, true
}

func methodValueAliasTargetSignature(tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start+3 != end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return funcSignature{}, false
	}
	info, ok := localStructSelectorInfo(localStructs, localTypes, structs, typeNames, tokens[start].Text, start)
	if !ok {
		return funcSignature{}, false
	}
	methodName := tokens[start+2].Text
	if sig, ok := methodSignatureForStructType(info, methodName, sigs, importMethods); ok {
		return sig, true
	}
	if sig, ok := promotedMethodSignature(structs, structFieldSetName(info), methodName, sigs, importMethods); ok {
		return sig, true
	}
	return funcSignature{}, false
}

func collectSignatureLocalValueTypes(tokens []scan.Token, start int, body int, structs []structFieldSet, typeNames []localValueType, locals []localValueType) []localValueType {
	for i := start; i < body; i++ {
		if tokens[i].Text != "(" {
			continue
		}
		close := findClose(tokens, i, "(", ")")
		if close < 0 || close > body {
			continue
		}
		locals = collectParameterListValueTypes(tokens, i+1, close, structs, typeNames, locals)
		i = close
	}
	return locals
}

func collectNamedResultLocalValueTypes(tokens []scan.Token, start int, body int, structs []structFieldSet, typeNames []localValueType, locals []localValueType) []localValueType {
	paramsOpen := -1
	for i := start; i < body; i++ {
		if tokens[i].Text == "(" {
			paramsOpen = i
			break
		}
	}
	if paramsOpen < 0 {
		return locals
	}
	paramsClose := findClose(tokens, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose+1 >= body || tokens[paramsClose+1].Text != "(" {
		return locals
	}
	resultOpen := paramsClose + 1
	resultClose := findClose(tokens, resultOpen, "(", ")")
	if resultClose < 0 || resultClose > body {
		return locals
	}
	return collectParameterListValueTypes(tokens, resultOpen+1, resultClose, structs, typeNames, locals)
}

func collectParameterListValueTypes(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType, locals []localValueType) []localValueType {
	var pending []string
	segments := expressionRanges(tokens, start, end)
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		name, typ, raw := parameterValueSegment(tokens, segment.start, segment.end, structs, typeNames)
		if typ != "" {
			if preserved := parameterValueSegmentRawType(tokens, segment.start, segment.end); preserved != "" {
				raw = preserved
			}
			if name != "" {
				pending = appendStringUniqueCheck(pending, name)
			}
			for j := 0; j < len(pending); j++ {
				locals = setLocalValueTypeWithRaw(locals, pending[j], typ, raw)
			}
			if len(pending) == 0 && name != "" {
				locals = setLocalValueTypeWithRaw(locals, name, typ, raw)
			}
			pending = nil
		} else if name != "" {
			pending = appendStringUniqueCheck(pending, name)
		}
	}
	return locals
}

func functionParamSignatures(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) []functionParamSignature {
	var out []functionParamSignature
	var pending []string
	paramIndex := 0
	segments := expressionRanges(tokens, start, end)
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		name, typeStart, typeEnd, hasType := functionParameterSegment(tokens, segment.start, segment.end)
		if hasType {
			names := pending
			if name != "" {
				names = append(names, name)
			}
			if len(names) == 0 {
				names = append(names, "")
			}
			if sig, ok := functionTypeSignatureAt(tokens, typeStart, typeEnd, structs, typeNames); ok {
				for nameIndex := 0; nameIndex < len(names); nameIndex++ {
					out = append(out, functionParamSignature{
						index: paramIndex + nameIndex,
						name:  names[nameIndex],
						sig:   sig,
					})
				}
			}
			paramIndex += len(names)
			pending = nil
		} else if name != "" {
			pending = append(pending, name)
		}
	}
	return out
}

func functionParameterSegment(tokens []scan.Token, start int, end int) (string, int, int, bool) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start >= end {
		return "", 0, 0, false
	}
	if tokens[start].Kind == scan.Ident {
		if start+1 < end && isFunctionParameterTypeStart(tokens[start+1]) {
			return tokens[start].Text, start + 1, end, true
		}
		if start+1 == end {
			return tokens[start].Text, 0, 0, false
		}
	}
	if isFunctionParameterTypeStart(tokens[start]) {
		return "", start, end, true
	}
	return "", start, end, true
}

func isFunctionParameterTypeStart(tok scan.Token) bool {
	return isTypeStart(tok) || tok.Text == "func"
}

func functionTypeSignatureAt(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start+1 == end && tokens[start].Kind == scan.Ident {
		return namedFunctionTypeSignature(tokens[start].Text, structs, typeNames)
	}
	if start >= end || tokens[start].Text != "func" || start+1 >= end || tokens[start+1].Text != "(" {
		return funcSignature{}, false
	}
	paramsOpen := start + 1
	paramsClose := findClose(tokens, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= end {
		return funcSignature{}, false
	}
	return funcSignature{
		params:         parameterCount(tokens, paramsOpen+1, paramsClose),
		results:        resultCount(tokens, paramsClose+1, end),
		paramTypes:     parameterTypeNames(tokens, paramsOpen+1, paramsClose, structs, typeNames),
		resultTypes:    resultTypeNames(tokens, paramsClose+1, end, structs, typeNames),
		callbackParams: functionParamSignatures(tokens, paramsOpen+1, paramsClose, structs, typeNames),
		variadic:       parameterListIsVariadic(tokens, paramsOpen+1, paramsClose),
	}, true
}

func namedFunctionTypeSignature(name string, structs []structFieldSet, typeNames []localValueType) (funcSignature, bool) {
	underlying := localValueTypeName(typeNames, name)
	if underlying == "" || underlying == name || !strings.HasPrefix(underlying, "func") {
		return funcSignature{}, false
	}
	toks, err := scan.Tokens([]byte(underlying))
	if err != nil {
		return funcSignature{}, false
	}
	end := len(toks)
	if end > 0 && toks[end-1].Kind == scan.EOF {
		end--
	}
	if end == 0 {
		return funcSignature{}, false
	}
	return functionTypeSignatureAt(toks, 0, end, structs, typeNames)
}

func parameterTypeNames(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) []string {
	var out []string
	pending := 0
	segments := expressionRanges(tokens, start, end)
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		name, typ, raw := parameterValueSegment(tokens, segment.start, segment.end, structs, typeNames)
		if typ != "" {
			if preserved := parameterValueSegmentRawType(tokens, segment.start, segment.end); preserved != "" {
				raw = preserved
			}
			if name != "" {
				pending++
			}
			if pending == 0 {
				out = append(out, raw)
			} else {
				for j := 0; j < pending; j++ {
					out = append(out, raw)
				}
			}
			pending = 0
		} else if name != "" {
			pending++
		}
	}
	return out
}

func resultTypeNames(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) []string {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	if start >= end {
		return nil
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close > start && close <= end {
			return parameterTypeNames(tokens, start+1, close, structs, typeNames)
		}
	}
	raw := rawTypeTextInRange(tokens, start, end)
	if raw == "" {
		return nil
	}
	return []string{raw}
}

func parameterValueSegment(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (string, string, string) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start >= end {
		return "", "", ""
	}
	if tokens[start].Kind == scan.Ident {
		if start+1 < end && isFunctionParameterTypeStart(tokens[start+1]) {
			raw := typeTextInRange(tokens, start+1, end)
			return tokens[start].Text, normalizeNamedType(raw, typeNames), raw
		}
		if start+1 == end {
			if isSingleIdentifierTypeName(tokens[start].Text, structs, typeNames) {
				return "", normalizeNamedType(tokens[start].Text, typeNames), tokens[start].Text
			}
			return tokens[start].Text, "", ""
		}
	}
	raw := typeTextInRange(tokens, start, end)
	return "", normalizeNamedType(raw, typeNames), raw
}

func parameterValueSegmentRawType(tokens []scan.Token, start int, end int) string {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start >= end {
		return ""
	}
	if tokens[start].Kind == scan.Ident {
		if start+1 < end && isFunctionParameterTypeStart(tokens[start+1]) {
			return rawTypeTextInRange(tokens, start+1, end)
		}
		if start+1 == end {
			return ""
		}
	}
	return rawTypeTextInRange(tokens, start, end)
}

func isSingleIdentifierTypeName(name string, structs []structFieldSet, typeNames []localValueType) bool {
	return isBuiltinTypeName(name) || structFieldSetIndex(structs, name) >= 0 || localValueTypeName(typeNames, name) != ""
}

func collectShortDeclValueTypes(tokens []scan.Token, body int, start int, assign int, limit int, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, locals []localValueType) []localValueType {
	stmtStart := start
	for start < assign && (tokens[start].Text == "for" || tokens[start].Text == "if" || tokens[start].Text == "switch") {
		start++
	}
	if stmtStart < assign && tokens[stmtStart].Text == "switch" {
		name, typ, scopeEnd, ok := staticInterfaceTypeSwitchBindingType(tokens, body, stmtStart, structs)
		if ok {
			return setScopedLocalValueTypeWithRaw(locals, name, typ, typ, assign+1, scopeEnd)
		}
	}
	if assign+1 < limit && tokens[assign+1].Text == "range" {
		return collectRangeShortDeclValueTypes(tokens, body, start, assign, limit, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
	}
	scopeEnd := declarationScopeEnd(tokens, body, assign, limit)
	end := simpleStatementEnd(tokens, assign+1, limit)
	lhs := expressionRanges(tokens, start, assign)
	rhs := expressionRanges(tokens, assign+1, end)
	if len(rhs) == 1 && len(lhs) > 1 {
		if resultTypes := singleCallResultTypes(tokens, rhs[0].start, rhs[0].end, sigs, importFuncs, localStructs, structs, importMethods); len(resultTypes) == len(lhs) {
			for i := 0; i < len(lhs); i++ {
				name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
				if name == "" || resultTypes[i] == "" {
					continue
				}
				locals = setScopedLocalValueTypeWithRaw(locals, name, normalizeNamedType(resultTypes[i], typeNames), resultTypes[i], assign+1, scopeEnd)
			}
			return locals
		}
	}
	if len(lhs) != len(rhs) {
		return locals
	}
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpression(tokens, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		raw := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if typ == "" {
			continue
		}
		locals = setScopedLocalValueTypeWithRaw(locals, name, typ, raw, assign+1, scopeEnd)
	}
	return locals
}

func collectRangeShortDeclValueTypes(tokens []scan.Token, body int, start int, assign int, limit int, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, locals []localValueType) []localValueType {
	lhs := expressionRanges(tokens, start, assign)
	if len(lhs) == 0 || len(lhs) > 2 {
		return locals
	}
	operandStart := assign + 2
	operandEnd := rangeOperandEnd(tokens, operandStart, limit)
	if operandStart >= operandEnd {
		return locals
	}
	scopeEnd := declarationScopeEnd(tokens, body, assign, limit)
	if keyType, valueType, ok := pureMapAliasOrDirectRangeExpressionTypes(tokens, operandStart, operandEnd); ok {
		name := singleIdentifierExpression(tokens, lhs[0].start, lhs[0].end)
		if name != "" {
			locals = setScopedLocalValueType(locals, name, keyType, assign+1, scopeEnd)
		}
		if len(lhs) < 2 {
			return locals
		}
		name = singleIdentifierExpression(tokens, lhs[1].start, lhs[1].end)
		if name != "" {
			locals = setScopedLocalValueType(locals, name, valueType, assign+1, scopeEnd)
		}
		return locals
	}
	typ := expressionSimpleTypeWithCallsAndTypes(tokens, operandStart, operandEnd, locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
	if typ == "" {
		return locals
	}
	name := singleIdentifierExpression(tokens, lhs[0].start, lhs[0].end)
	if name != "" {
		locals = setScopedLocalValueType(locals, name, "int", assign+1, scopeEnd)
	}
	if len(lhs) < 2 {
		return locals
	}
	elem := rangeValueType(typ)
	if elem == "" {
		return locals
	}
	name = singleIdentifierExpression(tokens, lhs[1].start, lhs[1].end)
	if name != "" {
		locals = setScopedLocalValueType(locals, name, elem, assign+1, scopeEnd)
	}
	return locals
}

func rangeValueType(typ string) string {
	if typ == "string" {
		return "int32"
	}
	if isSliceTypeName(typ) {
		return sliceElementType(typ)
	}
	return ""
}

func collectValueDeclTypes(tokens []scan.Token, body int, pos int, limit int, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, locals []localValueType) []localValueType {
	if pos+1 < limit && tokens[pos+1].Text == "(" {
		close := findClose(tokens, pos+1, "(", ")")
		if close < 0 || close >= limit {
			return locals
		}
		specStart := pos + 2
		for i := specStart; i <= close; i++ {
			if i == close || tokens[i].Text == ";" {
				locals = collectValueSpecTypes(tokens, body, limit, specStart, i, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
				specStart = i + 1
				continue
			}
			if tokens[i].Line != tokens[specStart].Line {
				locals = collectValueSpecTypes(tokens, body, limit, specStart, i, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
				specStart = i
			}
		}
		return locals
	}
	end := simpleStatementEnd(tokens, pos+1, limit)
	return collectValueSpecTypes(tokens, body, limit, pos+1, end, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods, locals)
}

func collectValueSpecTypes(tokens []scan.Token, body int, limit int, start int, end int, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod, locals []localValueType) []localValueType {
	for start < end && tokens[start].Text == ";" {
		start++
	}
	for end > start && tokens[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return locals
	}
	eq := findTopLevelToken(tokens, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	var names []string
	typeStart := -1
	for i := start; i < lhsEnd; i++ {
		if tokens[i].Text == "," {
			continue
		}
		if tokens[i].Kind == scan.Ident && (i == start || tokens[i-1].Text == ",") {
			names = appendStringUniqueCheck(names, tokens[i].Text)
			continue
		}
		if isTypeStart(tokens[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return locals
	}
	scopeEnd := declarationScopeEnd(tokens, body, start, limit)
	if eq >= 0 && len(names) == 1 {
		rhs := expressionRanges(tokens, eq+1, end)
		if len(rhs) == 1 {
			if _, _, ok := lowerableMapRangeExpressionTypes(tokens, rhs[0].start, rhs[0].end); ok {
				return locals
			}
		}
	}
	if typeStart >= 0 {
		raw := rawTypeTextInRange(tokens, typeStart, lhsEnd)
		typText := typeTextInRange(tokens, typeStart, lhsEnd)
		typ := normalizeNamedType(typText, typeNames)
		if typ != "" {
			for i := 0; i < len(names); i++ {
				locals = setScopedLocalValueTypeWithRaw(locals, names[i], typ, raw, end, scopeEnd)
			}
			return locals
		}
	}
	if eq < 0 {
		return locals
	}
	rhs := expressionRanges(tokens, eq+1, end)
	if len(rhs) == 1 && len(names) > 1 {
		if resultTypes := singleCallResultTypes(tokens, rhs[0].start, rhs[0].end, sigs, importFuncs, localStructs, structs, importMethods); len(resultTypes) == len(names) {
			for i := 0; i < len(names); i++ {
				if resultTypes[i] == "" {
					continue
				}
				locals = setScopedLocalValueTypeWithRaw(locals, names[i], normalizeNamedType(resultTypes[i], typeNames), resultTypes[i], end, scopeEnd)
			}
			return locals
		}
	}
	for i := 0; i < len(names) && i < len(rhs); i++ {
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		raw := expressionRawTypeWithCallsAndTypes(tokens, rhs[i].start, rhs[i].end, locals, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if typ != "" {
			locals = setScopedLocalValueTypeWithRaw(locals, names[i], typ, raw, end, scopeEnd)
		}
	}
	return locals
}

func setLocalValueType(locals []localValueType, name string, typ string) []localValueType {
	return setLocalValueTypeWithRaw(locals, name, typ, typ)
}

func setScopedLocalValueType(locals []localValueType, name string, typ string, start int, end int) []localValueType {
	return setScopedLocalValueTypeWithRaw(locals, name, typ, typ, start, end)
}

func setLocalValueTypeWithRaw(locals []localValueType, name string, typ string, raw string) []localValueType {
	return setLocalValueTypeEntry(locals, localValueType{name: name, typ: typ, raw: raw})
}

func setLocalValueTypeEntry(locals []localValueType, value localValueType) []localValueType {
	name := value.name
	typ := value.typ
	raw := value.raw
	if name == "" || name == "_" || typ == "" {
		return locals
	}
	if raw == "" {
		raw = typ
	}
	value.raw = raw
	for i := 0; i < len(locals); i++ {
		if locals[i].name == name {
			locals[i] = value
			return locals
		}
	}
	return append(locals, value)
}

func setScopedLocalValueTypeWithRaw(locals []localValueType, name string, typ string, raw string, start int, end int) []localValueType {
	if name == "" || name == "_" || typ == "" {
		return locals
	}
	if raw == "" {
		raw = typ
	}
	return append(locals, localValueType{name: name, typ: typ, raw: raw, start: start, end: end, scoped: true})
}

func localValueTypeName(locals []localValueType, name string) string {
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].name == name {
			return locals[i].typ
		}
	}
	return ""
}

func localValueTypeNameAt(locals []localValueType, name string, pos int) string {
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].name == name && localValueTypeVisibleAt(locals[i], pos) {
			return locals[i].typ
		}
	}
	return ""
}

func localValueRawTypeName(locals []localValueType, name string) string {
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].name == name {
			return localValueRawType(locals[i])
		}
	}
	return ""
}

func localValueRawTypeNameAt(locals []localValueType, name string, pos int) string {
	for i := len(locals) - 1; i >= 0; i-- {
		if locals[i].name == name && localValueTypeVisibleAt(locals[i], pos) {
			return localValueRawType(locals[i])
		}
	}
	return ""
}

func localValueTypeVisibleAt(value localValueType, pos int) bool {
	if !value.scoped {
		return true
	}
	if pos < value.start {
		return false
	}
	return value.end <= 0 || pos < value.end
}

func localValueRawType(value localValueType) string {
	if value.raw != "" {
		return value.raw
	}
	return value.typ
}

func singleIdentifierExpression(tokens []scan.Token, start int, end int) string {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start+1 != end || tokens[start].Kind != scan.Ident || tokens[start].Text == "_" {
		return ""
	}
	return tokens[start].Text
}

func expressionRanges(tokens []scan.Token, start int, end int) []expressionRange {
	var out []expressionRange
	exprStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "," {
			if exprStart < i {
				out = append(out, expressionRange{start: exprStart, end: i})
			}
			exprStart = i + 1
			continue
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	if exprStart < end {
		out = append(out, expressionRange{start: exprStart, end: end})
	}
	return out
}

func expressionSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionSimpleType(tokens, start+1, close, locals, structs)
		}
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		typ := expressionSimpleType(tokens, start+1, end, locals, structs)
		return unaryExpressionResultType(op, typ)
	}
	if start+1 == end {
		tok := tokens[start]
		if tok.Kind == scan.String {
			return "string"
		}
		if tok.Kind == scan.Char {
			return "int"
		}
		if tok.Kind == scan.Number {
			return numberLiteralType(tok.Text)
		}
		if tok.Kind == scan.Ident {
			if tok.Text == "true" || tok.Text == "false" {
				return "bool"
			}
			if tok.Text == "nil" {
				return "nil"
			}
			return localValueTypeNameAt(locals, tok.Text, start)
		}
	}
	if tokens[start].Text == "&" {
		target := expressionSimpleType(tokens, start+1, end, locals, structs)
		if target != "" {
			return "*" + target
		}
	}
	if tokens[start].Text == "*" && start+1 < end {
		target := expressionSimpleType(tokens, start+1, end, locals, structs)
		if strings.HasPrefix(target, "*") {
			return target[1:]
		}
	}
	if typ := lowerableMapLiteralIndexValueType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := indexedExpressionType(tokens, start, end, locals, structs); typ != "" {
		return typ
	}
	if typ := compositeLiteralSelectorSimpleType(tokens, start, end, structs, nil); typ != "" {
		return typ
	}
	if typ := compositeLiteralType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := callExpressionSimpleType(tokens, start, end, locals, structs); typ != "" {
		return typ
	}
	return ""
}

func expressionSimpleTypeWithImports(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, funcs []importedFunction) string {
	start, end = trimExpressionRange(tokens, start, end)
	if typ := importedCallResultType(tokens, start, end, funcs); typ != "" {
		return typ
	}
	return expressionSimpleType(tokens, start, end, locals, structs)
}

func expressionSimpleTypeWithCalls(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start < end && tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionSimpleTypeWithCalls(tokens, start+1, close, locals, structs, funcs, values, localStructs, sigs, importMethods)
		}
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		typ := expressionSimpleTypeWithCalls(tokens, start+1, end, locals, structs, funcs, values, localStructs, sigs, importMethods)
		return unaryExpressionResultType(op, typ)
	}
	if typ := binaryExpressionSimpleTypeWithCalls(tokens, start, end, locals, structs, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return typ
	}
	if tokens[start].Text == "*" && start+1 < end {
		target := expressionSimpleTypeWithCalls(tokens, start+1, end, locals, structs, funcs, values, localStructs, sigs, importMethods)
		if strings.HasPrefix(target, "*") {
			return target[1:]
		}
		return ""
	}
	if typ := functionLiteralDirectCallResultType(tokens, start, end, structs, nil); typ != "" {
		return typ
	}
	if typ := localCallResultType(tokens, start, end, sigs); typ != "" {
		return typ
	}
	if typ := compositeLiteralMethodCallResultType(tokens, start, end, structs, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := methodExpressionCallResultType(tokens, start, end, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := indexedReceiverMethodCallResultType(tokens, start, end, locals, structs, nil, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := methodCallResultType(tokens, start, end, localStructs, structs, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := importedCallResultType(tokens, start, end, funcs); typ != "" {
		return typ
	}
	if typ := importedValueType(tokens, start, end, values); typ != "" {
		return typ
	}
	if typ := compositeLiteralSelectorSimpleType(tokens, start, end, structs, nil); typ != "" {
		return typ
	}
	if typ := structSelectorSimpleType(tokens, start, end, localStructs, structs); typ != "" {
		return typ
	}
	return expressionSimpleType(tokens, start, end, locals, structs)
}

func expressionSimpleTypeWithCallsAndTypes(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if typ := staticInterfaceAssertionExpressionType(tokens, start, end); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := reducibleComplexComponentSimpleType(tokens, start, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := namedConversionExpressionType(tokens, start, end, typeNames); typ != "" {
		return typ
	}
	if typ := localValueStructSelectorSimpleType(tokens, start, end, locals, structs, typeNames); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := importedValueSelectorSimpleType(tokens, start, end, values, structs, typeNames); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := compositeLiteralSelectorSimpleType(tokens, start, end, structs, typeNames); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := unaryExpressionSimpleTypeWithCallsAndTypes(tokens, start, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := binaryExpressionSimpleTypeWithCallsAndTypes(tokens, start, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := indexedReceiverMethodCallResultType(tokens, start, end, locals, structs, typeNames, sigs, importMethods); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	if typ := functionLiteralDirectCallResultType(tokens, start, end, structs, typeNames); typ != "" {
		return normalizeNamedType(typ, typeNames)
	}
	return normalizeNamedType(expressionSimpleTypeWithCalls(tokens, start, end, locals, structs, funcs, values, localStructs, sigs, importMethods), typeNames)
}

func expressionRawTypeWithCallsAndTypes(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if typ := staticInterfaceAssertionExpressionType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := reducibleComplexComponentRawType(tokens, start, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return typ
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionRawTypeWithCallsAndTypes(tokens, start+1, close, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
		}
	}
	if typ := conversionExpressionRawType(tokens, start, end, typeNames); typ != "" {
		return typ
	}
	if typ := localValueStructSelectorRawType(tokens, start, end, locals, structs, typeNames); typ != "" {
		return typ
	}
	if typ := compositeLiteralSelectorRawType(tokens, start, end, structs, typeNames); typ != "" {
		return typ
	}
	if typ := importedValueSelectorRawType(tokens, start, end, values, structs, typeNames); typ != "" {
		return typ
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		typ := expressionRawTypeWithCallsAndTypes(tokens, start+1, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
		if !unaryOperandCompatible(op, normalizeNamedType(typ, typeNames)) {
			return ""
		}
		if op == "!" {
			return "bool"
		}
		return typ
	}
	if typ := binaryExpressionRawTypeWithCallsAndTypes(tokens, start, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods); typ != "" {
		return typ
	}
	if tokens[start].Text == "*" && start+1 < end {
		target := expressionRawTypeWithCallsAndTypes(tokens, start+1, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
		if strings.HasPrefix(target, "*") {
			return target[1:]
		}
		return ""
	}
	if typ := functionLiteralDirectCallResultType(tokens, start, end, structs, typeNames); typ != "" {
		return typ
	}
	if typ := localCallResultType(tokens, start, end, sigs); typ != "" {
		return typ
	}
	if typ := compositeLiteralMethodCallResultType(tokens, start, end, structs, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := methodExpressionCallResultType(tokens, start, end, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := indexedReceiverMethodCallResultType(tokens, start, end, locals, structs, typeNames, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := methodCallResultType(tokens, start, end, localStructs, structs, sigs, importMethods); typ != "" {
		return typ
	}
	if typ := importedCallResultType(tokens, start, end, funcs); typ != "" {
		return typ
	}
	if typ := importedValueRawType(tokens, start, end, values); typ != "" {
		return typ
	}
	if typ := structSelectorRawType(tokens, start, end, localStructs, structs); typ != "" {
		return typ
	}
	return expressionRawSimpleType(tokens, start, end, locals, structs, typeNames)
}

func staticInterfaceAssertionExpressionType(tokens []scan.Token, start int, end int) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+4 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." {
		return ""
	}
	close := findClose(tokens, start+2, "(", ")")
	if close != end-1 {
		return ""
	}
	return staticInterfaceAssertionTypeName(tokens, start+1)
}

func expressionRawSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionRawSimpleType(tokens, start+1, close, locals, structs, typeNames)
		}
	}
	if start+1 == end {
		tok := tokens[start]
		if tok.Kind == scan.String {
			return "string"
		}
		if tok.Kind == scan.Char {
			return "int"
		}
		if tok.Kind == scan.Number {
			return numberLiteralType(tok.Text)
		}
		if tok.Kind == scan.Ident {
			if tok.Text == "true" || tok.Text == "false" {
				return "bool"
			}
			if tok.Text == "nil" {
				return "nil"
			}
			return localValueRawTypeNameAt(locals, tok.Text, start)
		}
	}
	if tokens[start].Text == "&" {
		target := expressionRawSimpleType(tokens, start+1, end, locals, structs, typeNames)
		if target != "" {
			return "*" + target
		}
	}
	if typ := lowerableMapLiteralIndexValueType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := indexedExpressionRawType(tokens, start, end, locals, structs, typeNames); typ != "" {
		return typ
	}
	if typ := compositeLiteralSelectorRawType(tokens, start, end, structs, typeNames); typ != "" {
		return typ
	}
	if typ := arrayCompositeLiteralRawType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := compositeLiteralType(tokens, start, end); typ != "" {
		return typ
	}
	if typ := callExpressionRawSimpleType(tokens, start, end, locals, structs, typeNames); typ != "" {
		return typ
	}
	return ""
}

func conversionExpressionRawType(tokens []scan.Token, start int, end int, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close != end-1 {
		return ""
	}
	name := tokens[start].Text
	if isBuiltinTypeName(name) || localValueTypeName(typeNames, name) != "" {
		return name
	}
	return ""
}

func indexedExpressionRawType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) string {
	if start+3 > end || tokens[end-1].Text != "]" {
		return ""
	}
	open := findOpen(tokens, end-1, "[", "]")
	if open <= start {
		return ""
	}
	base := expressionRawSimpleType(tokens, start, open, locals, structs, typeNames)
	if topLevelTokenExists(tokens, open+1, end-1, ":") {
		return base
	}
	normalized := normalizeNamedType(base, typeNames)
	if normalized == "string" {
		return "byte"
	}
	if strings.HasPrefix(base, "[]") {
		return base[2:]
	}
	if strings.HasPrefix(normalized, "[]") {
		return normalized[2:]
	}
	return ""
}

func callExpressionRawSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) string {
	if start+2 >= end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close != end-1 {
		return ""
	}
	name := tokens[start].Text
	if isBuiltinTypeName(name) || localValueTypeName(typeNames, name) != "" {
		return name
	}
	switch name {
	case "cap", "len", "copy":
		return "int"
	case "syscall":
		return "int"
	case "recover":
		return "string"
	case "append":
		args := expressionRanges(tokens, start+2, close)
		if len(args) == 0 {
			return ""
		}
		return expressionRawSimpleType(tokens, args[0].start, args[0].end, locals, structs, typeNames)
	case "make":
		args := expressionRanges(tokens, start+2, close)
		if len(args) == 0 {
			return ""
		}
		return typeTextInRange(tokens, args[0].start, args[0].end)
	case "new":
		typ := typeTextInRange(tokens, start+2, close)
		if typ != "" {
			return "*" + typ
		}
	}
	return ""
}

func binaryExpressionRawTypeWithCallsAndTypes(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	op := topLevelBinaryOperator(tokens, start, end)
	if op < 0 {
		return ""
	}
	left := expressionRawTypeWithCallsAndTypes(tokens, start, op, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
	right := expressionRawTypeWithCallsAndTypes(tokens, op+1, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
	if !binaryOperandsAssignable(tokens, start, op, op+1, end, tokens[op].Text, left, right, typeNames) {
		return ""
	}
	return binaryRawResultType(tokens[op].Text, left, right, typeNames)
}

func binaryRawResultType(op string, left string, right string, typeNames []localValueType) string {
	switch op {
	case "||", "&&", "==", "!=", "<", "<=", ">", ">=":
		return "bool"
	}
	leftNorm := normalizeNamedType(left, typeNames)
	rightNorm := normalizeNamedType(right, typeNames)
	if left == right {
		return left
	}
	if isDefinedTypeName(left, typeNames) && leftNorm == rightNorm {
		return left
	}
	if isDefinedTypeName(right, typeNames) && leftNorm == rightNorm {
		return right
	}
	return binaryExpressionResultType(op, leftNorm, rightNorm)
}

func unaryExpressionSimpleTypeWithCallsAndTypes(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return unaryExpressionSimpleTypeWithCallsAndTypes(tokens, start+1, close, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
		}
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, start+1, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
		return unaryExpressionResultType(op, typ)
	}
	return ""
}

func namedConversionExpressionType(tokens []scan.Token, start int, end int, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return namedConversionExpressionType(tokens, start+1, close, typeNames)
		}
	}
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close != end-1 {
		return ""
	}
	if localValueTypeName(typeNames, tokens[start].Text) == "" {
		return ""
	}
	return normalizeNamedType(tokens[start].Text, typeNames)
}

func binaryExpressionSimpleTypeWithCalls(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	op := topLevelBinaryOperator(tokens, start, end)
	if op < 0 {
		return ""
	}
	left := expressionSimpleTypeWithCalls(tokens, start, op, locals, structs, funcs, values, localStructs, sigs, importMethods)
	right := expressionSimpleTypeWithCalls(tokens, op+1, end, locals, structs, funcs, values, localStructs, sigs, importMethods)
	return binaryExpressionResultType(tokens[op].Text, left, right)
}

func binaryExpressionSimpleTypeWithCallsAndTypes(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	op := topLevelBinaryOperator(tokens, start, end)
	if op < 0 {
		return ""
	}
	left := expressionSimpleTypeWithCallsAndTypes(tokens, start, op, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
	right := expressionSimpleTypeWithCallsAndTypes(tokens, op+1, end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
	return binaryExpressionResultType(tokens[op].Text, left, right)
}

func topLevelBinaryOperator(tokens []scan.Token, start int, end int) int {
	best := -1
	bestPrec := 100
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end {
			prec := binaryOperatorPrecedence(tokens[i].Text)
			if prec > 0 && prec <= bestPrec {
				best = i
				bestPrec = prec
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return best
}

func binaryOperatorPrecedence(op string) int {
	switch op {
	case "||":
		return 1
	case "&&":
		return 2
	case "==", "!=", "<", "<=", ">", ">=":
		return 3
	case "+", "-", "|", "^":
		return 4
	case "*", "/", "%", "<<", ">>", "&", "&^":
		return 5
	}
	return 0
}

func isBinaryOperatorText(text string) bool {
	return binaryOperatorPrecedence(text) > 0
}

func unaryOperatorText(text string) string {
	switch text {
	case "+", "-", "!", "^":
		return text
	}
	return ""
}

func unaryExpressionResultType(op string, typ string) string {
	if !unaryOperandCompatible(op, typ) {
		return ""
	}
	if op == "!" {
		return "bool"
	}
	return typ
}

func unaryOperandCompatible(op string, typ string) bool {
	switch op {
	case "+", "-":
		return isNumericTypeName(typ)
	case "!":
		return typ == "bool"
	case "^":
		return isIntegerTypeName(typ)
	}
	return false
}

func binaryExpressionResultType(op string, left string, right string) string {
	switch op {
	case "||", "&&":
		if binaryOperandsCompatible(op, left, right) {
			return "bool"
		}
		return ""
	case "==", "!=", "<", "<=", ">", ">=":
		if binaryOperandsCompatible(op, left, right) {
			return "bool"
		}
		return ""
	case "+":
		if !binaryOperandsCompatible(op, left, right) {
			return ""
		}
		if left == "string" {
			return "string"
		}
		return numericBinaryResultType(left, right)
	case "-", "*", "/", "%", "|", "^", "&", "&^":
		if !binaryOperandsCompatible(op, left, right) {
			return ""
		}
		return numericBinaryResultType(left, right)
	case "<<", ">>":
		if !binaryOperandsCompatible(op, left, right) {
			return ""
		}
		return left
	}
	return ""
}

func binaryOperandsCompatible(op string, left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	switch op {
	case "||", "&&":
		return left == "bool" && right == "bool"
	case "==", "!=":
		return comparableOperandTypes(left, right)
	case "<", "<=", ">", ">=":
		return orderedOperandTypes(left, right)
	case "+":
		return (left == "string" && right == "string") || numericOperandTypes(left, right)
	case "-", "*", "/":
		return numericOperandTypes(left, right)
	case "%", "|", "^", "&", "&^", "<<", ">>":
		return integerOperandTypes(left, right)
	}
	return false
}

func comparableOperandTypes(left string, right string) bool {
	if left == "nil" || right == "nil" {
		if left == "nil" && right == "nil" {
			return false
		}
		if left == "nil" {
			return isNilableType(right)
		}
		return isNilableType(left)
	}
	if isSliceTypeName(left) || isSliceTypeName(right) {
		return false
	}
	if left == right {
		return true
	}
	return numericOperandTypes(left, right) || integerOperandTypes(left, right)
}

func orderedOperandTypes(left string, right string) bool {
	if left == "string" && right == "string" {
		return true
	}
	return numericOperandTypes(left, right)
}

func numericOperandTypes(left string, right string) bool {
	return isNumericTypeName(left) && isNumericTypeName(right)
}

func integerOperandTypes(left string, right string) bool {
	return isIntegerTypeName(left) && isIntegerTypeName(right)
}

func numericBinaryResultType(left string, right string) string {
	if !isNumericTypeName(left) || !isNumericTypeName(right) {
		return ""
	}
	if isFloatTypeName(left) || isFloatTypeName(right) {
		return "float64"
	}
	return "int"
}

func isNumericTypeName(name string) bool {
	return isIntegerTypeName(name) || isFloatTypeName(name)
}

func isFloatTypeName(name string) bool {
	return name == "float32" || name == "float64"
}

func localCallResultType(tokens []scan.Token, start int, end int, sigs []funcSignature) string {
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close != end-1 {
		return ""
	}
	index := funcSignatureIndex(sigs, tokens[start].Text)
	if index < 0 {
		return ""
	}
	sig := sigs[index]
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func indexedReceiverMethodCallResultType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, sigs []funcSignature, importMethods []importedMethod) string {
	sig, ok := indexedReceiverMethodCallSignature(tokens, start, end, locals, structs, typeNames, sigs, importMethods)
	if !ok {
		return ""
	}
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func importedValueType(tokens []scan.Token, start int, end int, values []importedValue) string {
	if len(values) == 0 {
		return ""
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		for i := 0; i < len(values); i++ {
			value := values[i]
			if value.qualifier == "." && value.name == tokens[start].Text {
				return value.typ
			}
		}
		return ""
	}
	if start+3 != end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.qualifier == tokens[start].Text && value.name == tokens[start+2].Text {
			return value.typ
		}
	}
	return ""
}

func importedValueRawType(tokens []scan.Token, start int, end int, values []importedValue) string {
	if len(values) == 0 {
		return ""
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		for i := 0; i < len(values); i++ {
			value := values[i]
			if value.qualifier == "." && value.name == tokens[start].Text {
				if value.raw != "" {
					return value.raw
				}
				return value.typ
			}
		}
		return ""
	}
	if start+3 != end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.qualifier == tokens[start].Text && value.name == tokens[start+2].Text {
			if value.raw != "" {
				return value.raw
			}
			return value.typ
		}
	}
	return ""
}

func localValueStructSelectorSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	typ := normalizeNamedType(localValueTypeNameAt(locals, tokens[start].Text, start), typeNames)
	info := localStructInfoFromType(structs, typ)
	if info.typeName == "" {
		return ""
	}
	typeName := structFieldSetName(info)
	for memberPos := start + 2; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" {
			return ""
		}
		if memberPos+1 >= end {
			return typ
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func localValueStructSelectorRawType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	typ := normalizeNamedType(localValueTypeNameAt(locals, tokens[start].Text, start), typeNames)
	info := localStructInfoFromType(structs, typ)
	if info.typeName == "" {
		return ""
	}
	typeName := structFieldSetName(info)
	for memberPos := start + 2; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		raw := selectorFieldRawType(structs, typeName, tokens[memberPos].Text)
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" || raw == "" {
			return ""
		}
		if memberPos+1 >= end {
			return raw
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func compositeLiteralSelectorSimpleType(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) string {
	typeName, memberPos, ok := compositeLiteralSelectorStart(tokens, start, end, structs, typeNames)
	if !ok {
		return ""
	}
	typ := ""
	for ; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" {
			return ""
		}
		if memberPos+1 >= end {
			return typ
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func compositeLiteralSelectorRawType(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) string {
	typeName, memberPos, ok := compositeLiteralSelectorStart(tokens, start, end, structs, typeNames)
	if !ok {
		return ""
	}
	typ := ""
	for ; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		raw := selectorFieldRawType(structs, typeName, tokens[memberPos].Text)
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" || raw == "" {
			return ""
		}
		if memberPos+1 >= end {
			return raw
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func compositeLiteralSelectorStart(tokens []scan.Token, start int, end int, structs []structFieldSet, typeNames []localValueType) (string, int, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start+4 > end {
		return "", 0, false
	}
	baseStart := start
	baseEnd := 0
	memberPos := 0
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close < 0 || close+2 >= end || tokens[close+1].Text != "." || tokens[close+2].Kind != scan.Ident {
			return "", 0, false
		}
		baseStart = start + 1
		baseEnd = close
		if baseStart < baseEnd && tokens[baseStart].Text == "&" {
			baseStart++
		}
		memberPos = close + 2
	} else {
		open := findTopLevelToken(tokens, start, end, "{")
		if open < 0 {
			return "", 0, false
		}
		close := findClose(tokens, open, "{", "}")
		if close < 0 || close+2 >= end || tokens[close+1].Text != "." || tokens[close+2].Kind != scan.Ident {
			return "", 0, false
		}
		baseEnd = close + 1
		memberPos = close + 2
	}
	typ := compositeLiteralType(tokens, baseStart, baseEnd)
	if typ == "" {
		return "", 0, false
	}
	typ = normalizeNamedType(typ, typeNames)
	typeName := selectorBaseType(typ)
	if typeName == "" || structFieldSetIndex(structs, typeName) < 0 {
		return "", 0, false
	}
	return typeName, memberPos, true
}

func importedValueSelectorSimpleType(tokens []scan.Token, start int, end int, values []importedValue, structs []structFieldSet, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+5 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident || tokens[start+3].Text != "." || tokens[start+4].Kind != scan.Ident {
		return ""
	}
	typ := importedValueType(tokens, start, start+3, values)
	if typ == "" {
		return ""
	}
	typeName := importedValueSelectorStructTypeName(tokens[start].Text, typ, structs, typeNames)
	if typeName == "" {
		return ""
	}
	for memberPos := start + 4; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" {
			return ""
		}
		if memberPos+1 >= end {
			return typ
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func importedValueSelectorRawType(tokens []scan.Token, start int, end int, values []importedValue, structs []structFieldSet, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start+5 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident || tokens[start+3].Text != "." || tokens[start+4].Kind != scan.Ident {
		return ""
	}
	typ := importedValueType(tokens, start, start+3, values)
	if typ == "" {
		return ""
	}
	typeName := importedValueSelectorStructTypeName(tokens[start].Text, typ, structs, typeNames)
	if typeName == "" {
		return ""
	}
	for memberPos := start + 4; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		raw := selectorFieldRawType(structs, typeName, tokens[memberPos].Text)
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" || raw == "" {
			return ""
		}
		if memberPos+1 >= end {
			return raw
		}
		nextType := fieldStructTypeName(structs, typeName, typ)
		if nextType == "" {
			return ""
		}
		typeName = nextType
	}
	return typ
}

func importedValueSelectorStructTypeName(qualifier string, typ string, structs []structFieldSet, typeNames []localValueType) string {
	base := selectorBaseType(normalizeNamedType(typ, typeNames))
	if base == "" || strings.HasPrefix(base, "[]") {
		return ""
	}
	if structFieldSetIndex(structs, base) >= 0 {
		return base
	}
	qualified := qualifier + "." + base
	if structFieldSetIndex(structs, qualified) >= 0 {
		return qualified
	}
	return ""
}

func structSelectorSimpleType(tokens []scan.Token, start int, end int, localStructs []localStructType, structs []structFieldSet) string {
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	info, ok := localStructTypeInfo(localStructs, tokens[start].Text)
	if !ok || info.typeName == "" {
		return ""
	}
	typeName := structFieldSetName(info)
	typ := ""
	for memberPos := start + 2; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" || typ == "?" {
			return ""
		}
		if memberPos+1 >= end {
			break
		}
		if tokens[memberPos+1].Text != "." {
			return ""
		}
		typeName = fieldStructTypeName(structs, typeName, typ)
		if typeName == "" {
			return ""
		}
	}
	if typ == "" {
		return ""
	}
	return typ
}

func structSelectorRawType(tokens []scan.Token, start int, end int, localStructs []localStructType, structs []structFieldSet) string {
	if start+3 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident {
		return ""
	}
	info, ok := localStructTypeInfo(localStructs, tokens[start].Text)
	if !ok || info.typeName == "" {
		return ""
	}
	typeName := structFieldSetName(info)
	typ := ""
	for memberPos := start + 2; memberPos < end; memberPos += 2 {
		if memberPos-1 < start || tokens[memberPos-1].Text != "." || tokens[memberPos].Kind != scan.Ident {
			return ""
		}
		raw := selectorFieldRawType(structs, typeName, tokens[memberPos].Text)
		typ = selectorFieldType(structs, typeName, tokens[memberPos].Text)
		if typ == "" || raw == "" || typ == "?" {
			return ""
		}
		if memberPos+1 >= end {
			return raw
		}
		if tokens[memberPos+1].Text != "." {
			return ""
		}
		typeName = fieldStructTypeName(structs, typeName, typ)
		if typeName == "" {
			return ""
		}
	}
	return typ
}

func methodCallResultType(tokens []scan.Token, start int, end int, localStructs []localStructType, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) string {
	if start+5 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident || tokens[start+3].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+3, "(", ")")
	if close != end-1 {
		return ""
	}
	receiver, ok := localStructTypeInfo(localStructs, tokens[start].Text)
	if !ok {
		return ""
	}
	sig := funcSignature{}
	methodName := tokens[start+2].Text
	if direct, directOK := methodSignatureForStructType(receiver, methodName, sigs, importMethods); directOK {
		sig = direct
	} else {
		promoted, promotedOK := promotedMethodSignature(structs, structFieldSetName(receiver), methodName, sigs, importMethods)
		if !promotedOK {
			return ""
		}
		sig = promoted
	}
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func methodExpressionCallResultType(tokens []scan.Token, start int, end int, sigs []funcSignature, importMethods []importedMethod) string {
	sig, ok := methodExpressionCallSignatureAt(tokens, start, end, sigs, importMethods)
	if !ok || sig.pointerReceiver {
		return ""
	}
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func methodExpressionCallSignatureAt(tokens []scan.Token, start int, end int, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	call, ok := methodExpressionCallInfoAt(tokens, start, end)
	if !ok || call.callClose != end-1 {
		return funcSignature{}, false
	}
	sig, ok := methodExpressionCallSignature(call, sigs, importMethods)
	if !ok || sig.pointerReceiver {
		return funcSignature{}, false
	}
	return sig, true
}

func compositeLiteralMethodCallResultType(tokens []scan.Token, start int, end int, structs []structFieldSet, sigs []funcSignature, importMethods []importedMethod) string {
	sig, ok := compositeLiteralMethodCallSignature(tokens, start, end, structs, sigs, importMethods)
	if !ok {
		return ""
	}
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func importedCallResultType(tokens []scan.Token, start int, end int, funcs []importedFunction) string {
	if len(funcs) == 0 {
		return ""
	}
	if start+3 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "(" {
		close := findClose(tokens, start+1, "(", ")")
		if close != end-1 {
			return ""
		}
		index := importedFunctionIndex(funcs, ".", tokens[start].Text)
		if index < 0 {
			return ""
		}
		sig := funcs[index].sig
		if sig.results != 1 || len(sig.resultTypes) != 1 {
			return ""
		}
		return sig.resultTypes[0]
	}
	if start+5 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "." || tokens[start+2].Kind != scan.Ident || tokens[start+3].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+3, "(", ")")
	if close != end-1 {
		return ""
	}
	index := importedFunctionIndex(funcs, tokens[start].Text, tokens[start+2].Text)
	if index < 0 {
		return ""
	}
	sig := funcs[index].sig
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func trimExpressionRange(tokens []scan.Token, start int, end int) (int, int) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	return start, end
}

func numberLiteralType(text string) string {
	if strings.Contains(text, ".") || strings.ContainsAny(text, "pP") {
		return "float64"
	}
	if !strings.HasPrefix(text, "0x") && !strings.HasPrefix(text, "0X") && strings.ContainsAny(text, "eE") {
		return "float64"
	}
	return "int"
}

func compositeLiteralType(tokens []scan.Token, start int, end int) string {
	open := compositeLiteralOpen(tokens, start, end)
	if open < 0 {
		return ""
	}
	typ := typeTextInRange(tokens, start, open)
	if typ == "" {
		return ""
	}
	return typ
}

func arrayCompositeLiteralRawType(tokens []scan.Token, start int, end int) string {
	start, end = trimExpressionRange(tokens, start, end)
	open := compositeLiteralOpen(tokens, start, end)
	if open < 0 || start >= open || tokens[start].Text != "[" {
		return ""
	}
	brackClose := findClose(tokens, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= open {
		return ""
	}
	elem := rawTypeTextInRange(tokens, brackClose+1, open)
	if elem == "" || strings.HasPrefix(elem, "[") {
		return ""
	}
	lengthText := tokenTextInRange(tokens, start+1, brackClose)
	if lengthText == "..." {
		close := findClose(tokens, open, "{", "}")
		if close != end-1 {
			return ""
		}
		length, ok := inferredArrayCompositeLiteralLength(tokens, open, close)
		if !ok {
			return ""
		}
		lengthText = strconv.FormatInt(length, 10)
	}
	length, err := strconv.ParseInt(lengthText, 0, 64)
	if err != nil || length < 0 {
		return ""
	}
	return "[" + strconv.FormatInt(length, 10) + "]" + elem
}

func inferredArrayCompositeLiteralLength(tokens []scan.Token, open int, close int) (int64, bool) {
	values := expressionRanges(tokens, open+1, close)
	nextIndex := int64(0)
	maxIndex := int64(-1)
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.start >= value.end {
			continue
		}
		index := nextIndex
		colon := findTopLevelToken(tokens, value.start, value.end, ":")
		if colon >= 0 {
			key, ok := simpleIntegerLiteralKey(tokens, value.start, colon)
			if !ok {
				return 0, false
			}
			var err error
			index, err = strconv.ParseInt(key, 10, 64)
			if err != nil || index < 0 {
				return 0, false
			}
		}
		if index > maxIndex {
			maxIndex = index
		}
		nextIndex = index + 1
	}
	return maxIndex + 1, true
}

func compositeLiteralOpen(tokens []scan.Token, start int, end int) int {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return -1
	}
	if tokens[start].Text == "struct" && start+1 < end && tokens[start+1].Text == "{" {
		typeClose := findClose(tokens, start+1, "{", "}")
		if typeClose+1 < end && tokens[typeClose+1].Text == "{" {
			literalClose := findClose(tokens, typeClose+1, "{", "}")
			if literalClose == end-1 {
				return typeClose + 1
			}
		}
	}
	if start+3 < end && tokens[start].Text == "[" && tokens[start+1].Text == "]" && tokens[start+2].Text == "struct" && tokens[start+3].Text == "{" {
		typeClose := findClose(tokens, start+3, "{", "}")
		if typeClose+1 < end && tokens[typeClose+1].Text == "{" {
			literalClose := findClose(tokens, typeClose+1, "{", "}")
			if literalClose == end-1 {
				return typeClose + 1
			}
		}
	}
	if start+4 < end && tokens[start].Text == "[" && tokens[start+1].Text == "]" && tokens[start+2].Text == "*" && tokens[start+3].Text == "struct" && tokens[start+4].Text == "{" {
		typeClose := findClose(tokens, start+4, "{", "}")
		if typeClose+1 < end && tokens[typeClose+1].Text == "{" {
			literalClose := findClose(tokens, typeClose+1, "{", "}")
			if literalClose == end-1 {
				return typeClose + 1
			}
		}
	}
	return findTopLevelToken(tokens, start, end, "{")
}

func indexedExpressionType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet) string {
	if start+3 > end || tokens[end-1].Text != "]" {
		return ""
	}
	open := findOpen(tokens, end-1, "[", "]")
	if open <= start {
		return ""
	}
	base := expressionSimpleType(tokens, start, open, locals, structs)
	if topLevelTokenExists(tokens, open+1, end-1, ":") {
		return base
	}
	if base == "string" {
		return "byte"
	}
	if strings.HasPrefix(base, "[]") {
		return base[2:]
	}
	return ""
}

func topLevelTokenExists(tokens []scan.Token, start int, end int, text string) bool {
	return findTopLevelToken(tokens, start, end, text) >= 0
}

func callExpressionSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet) string {
	if start+2 >= end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return ""
	}
	close := findClose(tokens, start+1, "(", ")")
	if close != end-1 {
		return ""
	}
	name := tokens[start].Text
	if isBuiltinTypeName(name) {
		return name
	}
	switch name {
	case "cap", "len", "copy":
		return "int"
	case "syscall":
		return "int"
	case "recover":
		return "string"
	case "append":
		args := expressionRanges(tokens, start+2, close)
		if len(args) == 0 {
			return ""
		}
		return expressionSimpleType(tokens, args[0].start, args[0].end, locals, structs)
	case "make":
		args := expressionRanges(tokens, start+2, close)
		if len(args) == 0 {
			return ""
		}
		return typeTextInRange(tokens, args[0].start, args[0].end)
	case "new":
		typ := typeTextInRange(tokens, start+2, close)
		if typ != "" {
			return "*" + typ
		}
	}
	return ""
}

func typeTextInRange(tokens []scan.Token, start int, end int) string {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start >= end {
		return ""
	}
	if tokens[start].Text == "struct" && start+1 < end && tokens[start+1].Text == "{" {
		close := findClose(tokens, start+1, "{", "}")
		if close == end-1 {
			return anonymousStructTypeKey(tokens, start, end)
		}
	}
	if tokens[start].Text == "func" {
		return functionTypeTextInRange(tokens, start, end, false)
	}
	if tokens[start].Text == "..." {
		inner := typeTextInRange(tokens, start+1, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if tokens[start].Text == "*" {
		inner := typeTextInRange(tokens, start+1, end)
		if inner == "" {
			return ""
		}
		return "*" + inner
	}
	if start+2 < end && tokens[start].Text == "[" && tokens[start+1].Text != "]" {
		close := findClose(tokens, start, "[", "]")
		if close <= start+1 || close+1 >= end {
			return ""
		}
		inner := typeTextInRange(tokens, close+1, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if start+1 < end && tokens[start].Text == "[" && tokens[start+1].Text == "]" {
		inner := typeTextInRange(tokens, start+2, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if start+3 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		return tokens[start].Text + "." + tokens[start+2].Text
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		return tokens[start].Text
	}
	return ""
}

func rawTypeTextInRange(tokens []scan.Token, start int, end int) string {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	if start >= end {
		return ""
	}
	if tokens[start].Text == "struct" && start+1 < end && tokens[start+1].Text == "{" {
		close := findClose(tokens, start+1, "{", "}")
		if close == end-1 {
			return anonymousStructTypeKey(tokens, start, end)
		}
	}
	if tokens[start].Text == "func" {
		return functionTypeTextInRange(tokens, start, end, true)
	}
	if tokens[start].Text == "..." {
		inner := rawTypeTextInRange(tokens, start+1, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if tokens[start].Text == "*" {
		inner := rawTypeTextInRange(tokens, start+1, end)
		if inner == "" {
			return ""
		}
		return "*" + inner
	}
	if start+2 < end && tokens[start].Text == "[" && tokens[start+1].Text != "]" {
		close := findClose(tokens, start, "[", "]")
		if close <= start+1 || close+1 >= end {
			return ""
		}
		inner := rawTypeTextInRange(tokens, close+1, end)
		if inner == "" {
			return ""
		}
		return "[" + tokenTextInRange(tokens, start+1, close) + "]" + inner
	}
	if start+1 < end && tokens[start].Text == "[" && tokens[start+1].Text == "]" {
		inner := rawTypeTextInRange(tokens, start+2, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if start+3 == end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident {
		return tokens[start].Text + "." + tokens[start+2].Text
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		return tokens[start].Text
	}
	return ""
}

func functionTypeTextInRange(tokens []scan.Token, start int, end int, raw bool) string {
	if start+1 >= end || tokens[start].Text != "func" || tokens[start+1].Text != "(" {
		return ""
	}
	paramsClose := findClose(tokens, start+1, "(", ")")
	if paramsClose < 0 || paramsClose >= end {
		return ""
	}
	if paramsClose+1 == end {
		return tokenTextInRange(tokens, start, end)
	}
	resultStart := paramsClose + 1
	if tokens[resultStart].Text == "(" {
		resultClose := findClose(tokens, resultStart, "(", ")")
		if resultClose != end-1 {
			return ""
		}
		return tokenTextInRange(tokens, start, end)
	}
	resultType := ""
	if raw {
		resultType = rawTypeTextInRange(tokens, resultStart, end)
	} else {
		resultType = typeTextInRange(tokens, resultStart, end)
	}
	if resultType == "" {
		return ""
	}
	return tokenTextInRange(tokens, start, end)
}

func tokenTextInRange(tokens []scan.Token, start int, end int) string {
	var out []byte
	for i := start; i < end; i++ {
		out = append(out, tokens[i].Text...)
	}
	return string(out)
}

func typesCompatible(want string, got string) bool {
	if want == "" || got == "" {
		return true
	}
	if got == "nil" {
		return isNilableType(want)
	}
	if want == got {
		return true
	}
	if want == "error" && got == "string" {
		return true
	}
	if isIntegerTypeName(want) && isIntegerTypeName(got) {
		return true
	}
	if want == "float64" && isIntegerTypeName(got) {
		return true
	}
	return false
}

func expressionAssignableToType(tokens []scan.Token, start int, end int, want string, got string, typeNames []localValueType) bool {
	if typesAssignable(want, got, typeNames) {
		return true
	}
	if !expressionIsUntypedConstant(tokens, start, end) {
		return false
	}
	return typesCompatible(normalizeNamedType(want, typeNames), normalizeNamedType(got, typeNames))
}

func typesAssignable(want string, got string, typeNames []localValueType) bool {
	if want == "" || got == "" {
		return true
	}
	if got == "nil" {
		return isNilableType(normalizeNamedType(want, typeNames))
	}
	if want == got {
		return true
	}
	if strings.HasPrefix(want, "struct{") && got == "struct" {
		return true
	}
	if want == "struct" && strings.HasPrefix(got, "struct{") {
		return true
	}
	wantNorm := normalizeNamedType(want, typeNames)
	gotNorm := normalizeNamedType(got, typeNames)
	if want == "error" && gotNorm == "string" {
		return true
	}
	if assignabilityUnderlyingType(want, typeNames) == assignabilityUnderlyingType(got, typeNames) && (!isNamedAssignableType(want, typeNames) || !isNamedAssignableType(got, typeNames)) {
		return true
	}
	if isDefinedTypeName(want, typeNames) || isDefinedTypeName(got, typeNames) {
		return false
	}
	return typesCompatible(wantNorm, gotNorm)
}

func rangeTypedValueAssignable(want string, got string, typeNames []localValueType) bool {
	if want == "" || got == "" {
		return true
	}
	if got == "nil" {
		return isNilableType(normalizeNamedType(want, typeNames))
	}
	if want == got {
		return true
	}
	if want == "error" && normalizeNamedType(got, typeNames) == "string" {
		return true
	}
	if assignabilityUnderlyingType(want, typeNames) == assignabilityUnderlyingType(got, typeNames) && (!isNamedAssignableType(want, typeNames) || !isNamedAssignableType(got, typeNames)) {
		return true
	}
	return false
}

func assignabilityUnderlyingType(typ string, typeNames []localValueType) string {
	raw := rawNamedUnderlyingType(typ, typeNames)
	if raw != "" {
		return raw
	}
	return typ
}

func isNamedAssignableType(typ string, typeNames []localValueType) bool {
	if strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return false
	}
	return isBuiltinTypeName(typ) || localValueTypeName(typeNames, typ) != ""
}

func binaryOperandsAssignable(tokens []scan.Token, leftStart int, leftEnd int, rightStart int, rightEnd int, op string, left string, right string, typeNames []localValueType) bool {
	if left == "" || right == "" {
		return false
	}
	if (op == "==" || op == "!=") && fixedArrayOperandTypesComparable(left, right, typeNames) {
		return arrayComparisonOperandLowerable(tokens, leftStart, leftEnd, typeNames) && arrayComparisonOperandLowerable(tokens, rightStart, rightEnd, typeNames)
	}
	leftNorm := normalizeNamedType(left, typeNames)
	rightNorm := normalizeNamedType(right, typeNames)
	if !binaryOperandsCompatible(op, leftNorm, rightNorm) {
		return false
	}
	if typesAssignable(left, right, typeNames) || typesAssignable(right, left, typeNames) {
		return true
	}
	leftUntyped := expressionIsUntypedConstant(tokens, leftStart, leftEnd)
	rightUntyped := expressionIsUntypedConstant(tokens, rightStart, rightEnd)
	if isDefinedTypeName(left, typeNames) || isDefinedTypeName(right, typeNames) {
		if left == right {
			return true
		}
		if leftUntyped || rightUntyped {
			return true
		}
		return false
	}
	return true
}

type fixedArrayTypeInfo struct {
	elem   string
	length int64
}

func fixedArrayOperandTypesComparable(left string, right string, typeNames []localValueType) bool {
	leftInfo, leftOK := fixedArrayTypeInfoFromType(left, typeNames)
	rightInfo, rightOK := fixedArrayTypeInfoFromType(right, typeNames)
	if !leftOK || !rightOK {
		return false
	}
	if leftInfo.length != rightInfo.length {
		return false
	}
	leftElem := normalizeNamedType(leftInfo.elem, typeNames)
	rightElem := normalizeNamedType(rightInfo.elem, typeNames)
	return leftElem == rightElem && comparableOperandTypes(leftElem, rightElem)
}

func fixedArrayTypeInfoFromType(typ string, typeNames []localValueType) (fixedArrayTypeInfo, bool) {
	raw := rawNamedUnderlyingType(typ, typeNames)
	if raw != "" {
		typ = raw
	}
	if len(typ) < 4 || typ[0] != '[' {
		return fixedArrayTypeInfo{}, false
	}
	close := strings.IndexByte(typ, ']')
	if close <= 1 || close+1 >= len(typ) {
		return fixedArrayTypeInfo{}, false
	}
	length, err := strconv.ParseInt(typ[1:close], 0, 64)
	if err != nil || length < 0 {
		return fixedArrayTypeInfo{}, false
	}
	elem := typ[close+1:]
	if elem == "" {
		return fixedArrayTypeInfo{}, false
	}
	return fixedArrayTypeInfo{elem: elem, length: length}, true
}

func arrayComparisonOperandLowerable(tokens []scan.Token, start int, end int, typeNames []localValueType) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return arrayComparisonOperandLowerable(tokens, start+1, close, typeNames)
		}
	}
	if start+1 == end && tokens[start].Kind == scan.Ident {
		return true
	}
	if arrayComparisonSelectorOperandLowerable(tokens, start, end) {
		return true
	}
	if arrayCompositeLiteralRawType(tokens, start, end) != "" {
		return true
	}
	if namedArrayCompositeLiteralRawType(tokens, start, end, typeNames) != "" {
		return true
	}
	return directCallExpression(tokens, start, end)
}

func namedArrayCompositeLiteralRawType(tokens []scan.Token, start int, end int, typeNames []localValueType) string {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return ""
	}
	open := compositeLiteralOpen(tokens, start, end)
	if open < 0 || open <= start {
		return ""
	}
	close := findClose(tokens, open, "{", "}")
	if close != end-1 {
		return ""
	}
	typ := typeTextInRange(tokens, start, open)
	if typ == "" || strings.HasPrefix(typ, "[") {
		return ""
	}
	info, ok := fixedArrayTypeInfoFromType(typ, typeNames)
	if !ok {
		return ""
	}
	return "[" + strconv.FormatInt(info.length, 10) + "]" + info.elem
}

func arrayComparisonSelectorOperandLowerable(tokens []scan.Token, start int, end int) bool {
	if start >= end || tokens[start].Kind != scan.Ident || !containsTokenText(tokens, start, end, ".") {
		return false
	}
	for i := start + 1; i < end; {
		if tokens[i].Text == "[" {
			close := findClose(tokens, i, "[", "]")
			if close < 0 || close >= end {
				return false
			}
			i = close + 1
			continue
		}
		if i+1 >= end || tokens[i].Text != "." || tokens[i+1].Kind != scan.Ident {
			return false
		}
		i += 2
	}
	return true
}

func containsTokenText(tokens []scan.Token, start int, end int, text string) bool {
	if start < 0 {
		start = 0
	}
	if end > len(tokens) {
		end = len(tokens)
	}
	for i := start; i < end; i++ {
		if tokens[i].Text == text {
			return true
		}
	}
	return false
}

func directCallExpression(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return directCallExpression(tokens, start+1, close)
		}
	}
	if start+2 > end || tokens[start].Kind != scan.Ident || tokens[start+1].Text != "(" {
		return false
	}
	close := findClose(tokens, start+1, "(", ")")
	return close == end-1
}

func expressionIsUntypedConstant(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return expressionIsUntypedConstant(tokens, start+1, close)
		}
	}
	if op := unaryOperatorText(tokens[start].Text); op != "" && start+1 < end {
		return expressionIsUntypedConstant(tokens, start+1, end)
	}
	if op := topLevelBinaryOperator(tokens, start, end); op >= 0 {
		return expressionIsUntypedConstant(tokens, start, op) && expressionIsUntypedConstant(tokens, op+1, end)
	}
	if start+1 != end {
		return false
	}
	tok := tokens[start]
	if tok.Kind == scan.Number || tok.Kind == scan.String || tok.Kind == scan.Char {
		return true
	}
	return tok.Text == "true" || tok.Text == "false"
}

func isDefinedTypeName(typ string, typeNames []localValueType) bool {
	for strings.HasPrefix(typ, "*") {
		typ = typ[1:]
	}
	if strings.HasPrefix(typ, "[]") {
		typ = typ[2:]
	}
	return localValueTypeName(typeNames, typ) != ""
}

func isIntegerTypeName(name string) bool {
	switch name {
	case "int", "int16", "int32", "int64", "byte":
		return true
	}
	return false
}

func collectSignatureLocalStructTypes(tokens []scan.Token, start int, body int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	for i := start; i < body; i++ {
		if tokens[i].Text != "(" {
			continue
		}
		close := findClose(tokens, i, "(", ")")
		if close < 0 || close > body {
			continue
		}
		locals = collectParameterListStructTypes(tokens, i+1, close, structs, importNames, locals)
		i = close
	}
	return locals
}

func collectParameterListStructTypes(tokens []scan.Token, start int, end int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || tokens[i].Text == "," {
			name, info := parameterStructSegment(tokens, segmentStart, i, structs, importNames)
			if name != "" && info.typeName == "" {
				if parameterSegmentHasExplicitType(tokens, segmentStart, i) {
					pending = nil
				} else {
					pending = appendStringUniqueCheck(pending, name)
				}
			}
			if info.typeName != "" {
				if name != "" {
					pending = appendStringUniqueCheck(pending, name)
				}
				for j := 0; j < len(pending); j++ {
					locals = setLocalStructTypeInfo(locals, pending[j], info)
				}
				pending = nil
			}
			segmentStart = i + 1
		}
	}
	return locals
}

func parameterSegmentHasExplicitType(tokens []scan.Token, start int, end int) bool {
	for start < end && tokens[start].Text == "," {
		start++
	}
	for end > start && tokens[end-1].Text == "," {
		end--
	}
	return start+1 < end && isTypeStart(tokens[start+1])
}

func parameterStructSegment(tokens []scan.Token, start int, end int, structs []structFieldSet, importNames []string) (string, localStructType) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	if start >= end || tokens[start].Kind != scan.Ident {
		return "", localStructType{}
	}
	if start+1 >= end {
		if structFieldSetIndex(structs, tokens[start].Text) >= 0 {
			return "", localStructType{}
		}
		return tokens[start].Text, localStructType{}
	}
	info := structTypeInfoInRange(tokens, start+1, end, structs, importNames)
	if info.typeName != "" {
		return tokens[start].Text, info
	}
	return tokens[start].Text, localStructType{}
}

func collectShortDeclStructType(tokens []scan.Token, start int, assign int, limit int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	if start+1 != assign || tokens[start].Kind != scan.Ident {
		return locals
	}
	end := simpleStatementEnd(tokens, assign+1, limit)
	info := compositeLiteralStructType(tokens, assign+1, end, structs, importNames)
	if info.typeName == "" {
		info = selectorStructTypeInfo(tokens, assign+1, end, structs, locals)
	}
	if info.typeName == "" {
		return locals
	}
	return setLocalStructTypeInfo(locals, tokens[start].Text, info)
}

func selectorStructTypeInfo(tokens []scan.Token, start int, end int, structs []structFieldSet, locals []localStructType) localStructType {
	typ := structSelectorSimpleType(tokens, start, end, locals, structs)
	return localStructInfoFromType(structs, typ)
}

func localStructInfoFromType(structs []structFieldSet, typ string) localStructType {
	pointer := false
	for strings.HasPrefix(typ, "*") {
		pointer = true
		typ = typ[1:]
	}
	if typ == "" || strings.HasPrefix(typ, "[]") || structFieldSetIndex(structs, typ) < 0 {
		return localStructType{}
	}
	if dot := strings.IndexByte(typ, '.'); dot >= 0 {
		return localStructType{qualifier: typ[:dot], typeName: typ[dot+1:], pointer: pointer}
	}
	return localStructType{typeName: typ, pointer: pointer}
}

func localStructSelectorInfo(structs []localStructType, values []localValueType, allStructs []structFieldSet, typeNames []localValueType, name string, pos int) (localStructType, bool) {
	if info, ok := localStructTypeInfo(structs, name); ok {
		return info, true
	}
	typ := normalizeNamedType(localValueTypeNameAt(values, name, pos), typeNames)
	info := localStructInfoFromType(allStructs, typ)
	if info.typeName == "" {
		return localStructType{}, false
	}
	return info, true
}

func collectVarStatementStructType(tokens []scan.Token, pos int, limit int, structs []structFieldSet, importNames []string, locals []localStructType) []localStructType {
	end := simpleStatementEnd(tokens, pos+1, limit)
	if pos+1 >= end || tokens[pos+1].Kind != scan.Ident {
		return locals
	}
	name := tokens[pos+1].Text
	info := localStructType{}
	if pos+2 < end && tokens[pos+2].Text != "=" {
		typeEnd := end
		for i := pos + 2; i < end; i++ {
			if tokens[i].Text == "=" {
				typeEnd = i
				break
			}
		}
		info = structTypeInfoInRange(tokens, pos+2, typeEnd, structs, importNames)
	}
	if info.typeName == "" {
		eq := findTopLevelToken(tokens, pos+2, end, "=")
		if eq >= 0 {
			info = compositeLiteralStructType(tokens, eq+1, end, structs, importNames)
		}
	}
	if info.typeName == "" {
		return locals
	}
	return setLocalStructTypeInfo(locals, name, info)
}

func compositeLiteralStructType(tokens []scan.Token, start int, end int, structs []structFieldSet, importNames []string) localStructType {
	pointer := false
	for start < end && tokens[start].Text == "&" {
		pointer = true
		start++
	}
	if start >= end || tokens[start].Kind != scan.Ident {
		return localStructType{}
	}
	if start+1 < end && tokens[start+1].Text == "{" && structFieldSetIndex(structs, tokens[start].Text) >= 0 {
		return localStructType{typeName: tokens[start].Text, pointer: pointer}
	}
	if start+3 < end && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "{" && containsString(importNames, tokens[start].Text) {
		return localStructType{qualifier: tokens[start].Text, typeName: tokens[start+2].Text, pointer: pointer}
	}
	return localStructType{}
}

func structTypeInfoInRange(tokens []scan.Token, start int, end int, structs []structFieldSet, importNames []string) localStructType {
	pointer := false
	for start < end && tokens[start].Text == "*" {
		pointer = true
		start++
	}
	if start < end && tokens[start].Kind == scan.Ident && structFieldSetIndex(structs, tokens[start].Text) >= 0 {
		return localStructType{typeName: tokens[start].Text, pointer: pointer}
	}
	if start+2 < end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && containsString(importNames, tokens[start].Text) {
		return localStructType{qualifier: tokens[start].Text, typeName: tokens[start+2].Text, pointer: pointer}
	}
	return localStructType{}
}

func setLocalStructType(locals []localStructType, name string, typeName string, pointer bool) []localStructType {
	return setLocalStructTypeInfo(locals, name, localStructType{typeName: typeName, pointer: pointer})
}

func setLocalStructTypeInfo(locals []localStructType, name string, info localStructType) []localStructType {
	if name == "" || name == "_" || info.typeName == "" {
		return locals
	}
	for i := 0; i < len(locals); i++ {
		if locals[i].name == name {
			locals[i].qualifier = info.qualifier
			locals[i].typeName = info.typeName
			locals[i].pointer = info.pointer
			return locals
		}
	}
	info.name = name
	return append(locals, info)
}

func localStructTypeName(locals []localStructType, name string) string {
	info, ok := localStructTypeInfo(locals, name)
	if ok {
		return info.typeName
	}
	return ""
}

func localStructTypeInfo(locals []localStructType, name string) (localStructType, bool) {
	for i := 0; i < len(locals); i++ {
		if locals[i].name == name {
			return locals[i], true
		}
	}
	return localStructType{}, false
}

func collectSignatureLocalNames(tokens []scan.Token, start int, body int, names []string) []string {
	for i := start; i < body; i++ {
		if tokens[i].Text != "(" {
			continue
		}
		close := findClose(tokens, i, "(", ")")
		if close < 0 || close > body {
			continue
		}
		names = collectParameterNames(tokens, i+1, close, names)
		i = close
	}
	return names
}

func collectNamedResultLocalNames(tokens []scan.Token, start int, body int, names []string) []string {
	paramsOpen := -1
	for i := start; i < body; i++ {
		if tokens[i].Text == "(" {
			paramsOpen = i
			break
		}
	}
	if paramsOpen < 0 {
		return names
	}
	paramsClose := findClose(tokens, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= body {
		return names
	}
	resultNames := namedResultNamesAfterParams(tokens, paramsClose, body)
	for i := 0; i < len(resultNames); i++ {
		names = appendStringUniqueCheck(names, resultNames[i])
	}
	return names
}

func namedResultNamesAfterParams(tokens []scan.Token, paramsClose int, body int) []string {
	resultOpen := paramsClose + 1
	if resultOpen >= body || resultOpen >= len(tokens) || tokens[resultOpen].Text != "(" {
		return nil
	}
	resultClose := findClose(tokens, resultOpen, "(", ")")
	if resultClose < 0 || resultClose > body {
		return nil
	}
	return namedResultNamesInList(tokens, resultOpen+1, resultClose)
}

func namedResultNamesInList(tokens []scan.Token, start int, end int) []string {
	var names []string
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || tokens[i].Text == "," {
			name, hasType := namedResultSegment(tokens, segmentStart, i)
			if name != "" {
				if hasType {
					pending = appendStringUniqueCheck(pending, name)
					for j := 0; j < len(pending); j++ {
						names = appendStringUniqueCheck(names, pending[j])
					}
					pending = nil
				} else {
					pending = appendStringUniqueCheck(pending, name)
				}
			}
			segmentStart = i + 1
		}
	}
	return names
}

func namedResultSegment(tokens []scan.Token, start int, end int) (string, bool) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	if start >= end || tokens[start].Kind != scan.Ident {
		return "", false
	}
	return tokens[start].Text, start+1 < end && isTypeStart(tokens[start+1])
}

func collectParameterNames(tokens []scan.Token, start int, end int, names []string) []string {
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || tokens[i].Text == "," {
			name, hasType := parameterNameSegment(tokens, segmentStart, i)
			if name != "" {
				if hasType {
					pending = appendStringUniqueCheck(pending, name)
					for j := 0; j < len(pending); j++ {
						names = appendStringUniqueCheck(names, pending[j])
					}
					pending = nil
				} else if !isBuiltinTypeName(name) {
					pending = appendStringUniqueCheck(pending, name)
				}
			}
			segmentStart = i + 1
		}
	}
	return names
}

func parameterNameSegment(tokens []scan.Token, start int, end int) (string, bool) {
	for start < end && tokens[start].Text == "," {
		start++
	}
	if start >= end || tokens[start].Kind != scan.Ident {
		return "", false
	}
	return tokens[start].Text, start+1 < end && isTypeStart(tokens[start+1])
}

func collectAssignmentLeftNames(tokens []scan.Token, start int, end int, names []string) []string {
	for i := start; i < end; i++ {
		if tokens[i].Kind == scan.Ident && tokens[i].Text != "_" && !isSelectorMember(tokens, i) {
			names = appendStringUniqueCheck(names, tokens[i].Text)
		}
	}
	return names
}

func collectVarStatementNames(tokens []scan.Token, pos int, limit int, names []string) []string {
	end := simpleStatementEnd(tokens, pos+1, limit)
	if pos+1 < limit && tokens[pos+1].Text == "(" {
		close := findClose(tokens, pos+1, "(", ")")
		if close > pos+1 && close < limit {
			end = close
			for i := pos + 2; i < end; i++ {
				if tokens[i].Kind == scan.Ident && (i == pos+2 || tokens[i-1].Text == "," || tokens[i-1].Line != tokens[i].Line) {
					names = appendStringUniqueCheck(names, tokens[i].Text)
				}
			}
			return names
		}
	}
	if pos+1 < end && tokens[pos+1].Kind == scan.Ident {
		names = appendStringUniqueCheck(names, tokens[pos+1].Text)
		for i := pos + 2; i < end; i++ {
			if tokens[i].Text == "=" {
				break
			}
			if tokens[i].Kind == scan.Ident && tokens[i-1].Text == "," {
				names = appendStringUniqueCheck(names, tokens[i].Text)
				continue
			}
			if isTypeStart(tokens[i]) {
				break
			}
		}
	}
	return names
}

func appendStringUniqueCheck(names []string, name string) []string {
	if name == "" || name == "_" || containsString(names, name) {
		return names
	}
	return append(names, name)
}

func appendInitializerName(names []string, name string) []string {
	if name == "" {
		return names
	}
	if name == "_" {
		return append(names, name)
	}
	return appendStringUniqueCheck(names, name)
}

func parameterCount(tokens []scan.Token, start int, end int) int {
	if start >= end {
		return 0
	}
	count := 0
	segments := expressionRanges(tokens, start, end)
	for i := 0; i < len(segments); i++ {
		count += parameterSegmentCount(tokens, segments[i].start, segments[i].end)
	}
	return count
}

func parameterSegmentCount(tokens []scan.Token, start int, end int) int {
	for start < end && tokens[start].Text == "," {
		start++
	}
	if start >= end {
		return 0
	}
	if tokens[start].Kind != scan.Ident {
		return 1
	}
	if start+1 < end && tokens[start+1].Text == "," {
		count := 0
		for i := start; i < end; i++ {
			if tokens[i].Kind == scan.Ident {
				count++
			}
			if i+1 < end && tokens[i+1].Text != "," {
				break
			}
		}
		if count > 0 {
			return count
		}
	}
	return 1
}

func resultCount(tokens []scan.Token, start int, end int) int {
	for start < end && tokens[start].Text == "\n" {
		start++
	}
	if start >= end {
		return 0
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close > start && close <= end {
			return parameterCount(tokens, start+1, close)
		}
	}
	if isTypeStart(tokens[start]) {
		return 1
	}
	return 0
}

func parameterListIsVariadic(tokens []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if tokens[i].Text == "..." {
			return true
		}
	}
	return false
}

func funcSignatureByName(sigs []funcSignature, name string) funcSignature {
	index := funcSignatureIndex(sigs, name)
	if index >= 0 {
		return sigs[index]
	}
	return funcSignature{name: name}
}

func funcSignatureForDecl(tokens []scan.Token, decl parse.Decl, sigs []funcSignature) funcSignature {
	receiverType := methodReceiverTypeName(tokens, decl)
	for i := 0; i < len(sigs); i++ {
		if sigs[i].name == decl.Name && sigs[i].receiverType == receiverType {
			return sigs[i]
		}
	}
	return funcSignature{name: decl.Name, receiverType: receiverType}
}

func funcSignatureIndex(sigs []funcSignature, name string) int {
	for i := 0; i < len(sigs); i++ {
		if sigs[i].receiverType == "" && sigs[i].name == name {
			return i
		}
	}
	return -1
}

func simpleStatementStart(tokens []scan.Token, body int, pos int) int {
	line := tokens[pos].Line
	for i := pos - 1; i > body; i-- {
		if tokens[i].Text == "{" || tokens[i].Text == "}" || tokens[i].Text == ";" {
			return i + 1
		}
		if tokens[i].Line != line {
			return i + 1
		}
	}
	return body + 1
}

func isSimpleStatementStart(tokens []scan.Token, body int, pos int) bool {
	return simpleStatementStart(tokens, body, pos) == pos
}

func simpleStatementEnd(tokens []scan.Token, start int, limit int) int {
	if start >= limit {
		return start
	}
	line := tokens[start].Line
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < limit; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if tokens[i].Text == ";" || tokens[i].Text == "}" {
				return i
			}
			if i > start && tokens[i].Line != line {
				if tokens[i].Text == "," || tokens[i-1].Text == "," || continuesFunctionLiteralCall(tokens, i) {
					line = tokens[i].Line
				} else {
					return i
				}
			}
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return limit
}

func continuesFunctionLiteralCall(tokens []scan.Token, pos int) bool {
	return pos > 0 && pos < len(tokens) && tokens[pos].Text == "(" && tokens[pos-1].Text == "}" && tokens[pos].Line == tokens[pos-1].Line
}

func expressionListCount(tokens []scan.Token, start int, end int) int {
	for start < end && tokens[start].Text == "," {
		start++
	}
	if start >= end {
		return 0
	}
	count := 1
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && tokens[i].Text == "," {
			count++
			continue
		}
		updateDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return count
}

func singleCallResultCount(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod) int {
	resultTypes := singleCallResultTypes(tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods)
	if resultTypes == nil {
		return -1
	}
	return len(resultTypes)
}

func singleCallResultTypes(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod) []string {
	if sig, ok := singleCallSignature(tokens, start, end, sigs, importFuncs, localStructs, structs, importMethods); ok {
		if len(sig.resultTypes) == sig.results {
			return sig.resultTypes
		}
		return make([]string, sig.results)
	}
	return nil
}

func singleCallSignature(tokens []scan.Token, start int, end int, sigs []funcSignature, importFuncs []importedFunction, localStructs []localStructType, structs []structFieldSet, importMethods []importedMethod) (funcSignature, bool) {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return funcSignature{}, false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return singleCallSignature(tokens, start+1, close, sigs, importFuncs, localStructs, structs, importMethods)
		}
	}
	if sig, ok := functionLiteralDirectCallSignature(tokens, start, end, structs, nil); ok {
		return sig, true
	}
	if start+3 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "(" {
		close := findClose(tokens, start+1, "(", ")")
		if close == end-1 {
			index := funcSignatureIndex(sigs, tokens[start].Text)
			if index >= 0 {
				return sigs[index], true
			}
			index = importedFunctionIndex(importFuncs, ".", tokens[start].Text)
			if index >= 0 {
				return importFuncs[index].sig, true
			}
		}
	}
	if sig, ok := compositeLiteralMethodCallSignature(tokens, start, end, structs, sigs, importMethods); ok {
		return sig, true
	}
	if sig, ok := methodExpressionCallSignatureAt(tokens, start, end, sigs, importMethods); ok {
		return sig, true
	}
	if start+5 <= end && tokens[start].Kind == scan.Ident && tokens[start+1].Text == "." && tokens[start+2].Kind == scan.Ident && tokens[start+3].Text == "(" {
		close := findClose(tokens, start+3, "(", ")")
		if close != end-1 {
			return funcSignature{}, false
		}
		if receiver, ok := localStructTypeInfo(localStructs, tokens[start].Text); ok {
			if receiver.qualifier == "" {
				index := methodSignatureIndex(sigs, receiver.typeName, tokens[start+2].Text)
				if index >= 0 {
					return sigs[index], true
				}
			} else {
				index := importedMethodIndex(importMethods, receiver.qualifier, receiver.typeName, tokens[start+2].Text)
				if index >= 0 {
					return importMethods[index].sig, true
				}
			}
		}
		index := importedFunctionIndex(importFuncs, tokens[start].Text, tokens[start+2].Text)
		if index >= 0 {
			return importFuncs[index].sig, true
		}
	}
	return funcSignature{}, false
}

func isFunctionValueUse(tokens []scan.Token, pos int, sigs []funcSignature, locals []string) bool {
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	tok := tokens[pos]
	if tok.Kind != scan.Ident {
		return false
	}
	if funcSignatureIndex(sigs, tok.Text) < 0 {
		return false
	}
	if containsString(locals, tok.Text) {
		return false
	}
	if isCompositeKey(tokens, pos) {
		return false
	}
	if pos+1 < len(tokens) && tokens[pos+1].Text == "(" {
		return false
	}
	if pos > 0 {
		prev := tokens[pos-1].Text
		if prev == "func" || prev == "." || prev == "type" || prev == "goto" {
			return false
		}
	}
	return true
}

func isFunctionAliasInitializerUse(tokens []scan.Token, pos int, body int, limit int, sigs []funcSignature, importFuncs []importedFunction) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Kind != scan.Ident {
		return false
	}
	stmtStart := simpleStatementStart(tokens, body, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, limit)
	if stmtStart >= stmtEnd {
		return false
	}
	if aliasInitializerTargetAt(tokens, stmtStart, stmtEnd, pos, sigs, importFuncs) {
		return true
	}
	if stmtStart > body && tokens[stmtStart].Text != "var" {
		for i := stmtStart; i < stmtEnd; i++ {
			if tokens[i].Text == ":=" {
				return aliasInitializerTargetAt(tokens, stmtStart, stmtEnd, pos, sigs, importFuncs)
			}
		}
	}
	return false
}

func isMethodAliasInitializerUse(tokens []scan.Token, pos int, body int, limit int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Kind != scan.Ident {
		return false
	}
	stmtStart := simpleStatementStart(tokens, body, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, limit)
	return methodAliasInitializerTargetAt(tokens, stmtStart, stmtEnd, pos, localTypes, structs, typeNames, localStructs, sigs, importMethods)
}

func isMethodExpressionAliasInitializerUse(tokens []scan.Token, pos int, body int, limit int, sigs []funcSignature, importMethods []importedMethod) bool {
	if pos < 0 || pos >= len(tokens) || tokens[pos].Kind != scan.Ident {
		return false
	}
	stmtStart := simpleStatementStart(tokens, body, pos)
	stmtEnd := simpleStatementEnd(tokens, stmtStart, limit)
	return methodExpressionAliasInitializerTargetAt(tokens, stmtStart, stmtEnd, pos, sigs, importMethods)
}

func methodExpressionAliasInitializerTargetAt(tokens []scan.Token, stmtStart int, stmtEnd int, pos int, sigs []funcSignature, importMethods []importedMethod) bool {
	if stmtStart < stmtEnd && tokens[stmtStart].Text == "var" {
		eq := findTopLevelToken(tokens, stmtStart+1, stmtEnd, "=")
		if eq < 0 || eq != stmtStart+2 {
			return false
		}
		_, ok := methodExpressionAliasTargetSignature(tokens, eq+1, stmtEnd, sigs, importMethods)
		return ok && pos >= eq+1 && pos < stmtEnd
	}
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		if pos < rhs[i].start || pos >= rhs[i].end {
			continue
		}
		_, ok := methodExpressionAliasTargetSignature(tokens, rhs[i].start, rhs[i].end, sigs, importMethods)
		return ok
	}
	return false
}

func methodAliasInitializerTargetAt(tokens []scan.Token, stmtStart int, stmtEnd int, pos int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) bool {
	if stmtStart < stmtEnd && tokens[stmtStart].Text == "var" {
		eq := findTopLevelToken(tokens, stmtStart+1, stmtEnd, "=")
		if eq < 0 || eq != stmtStart+2 {
			return false
		}
		_, ok := methodAliasTargetSignature(tokens, eq+1, stmtEnd, localTypes, structs, typeNames, localStructs, sigs, importMethods)
		return ok && pos >= eq+1 && pos < stmtEnd
	}
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		if pos < rhs[i].start || pos >= rhs[i].end {
			continue
		}
		_, ok := methodAliasTargetSignature(tokens, rhs[i].start, rhs[i].end, localTypes, structs, typeNames, localStructs, sigs, importMethods)
		return ok
	}
	return false
}

func methodAliasTargetSignature(tokens []scan.Token, start int, end int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) (funcSignature, bool) {
	if sig, ok := methodValueAliasTargetSignature(tokens, start, end, localTypes, structs, typeNames, localStructs, sigs, importMethods); ok {
		return sig, true
	}
	return compositeLiteralMethodValueAliasTargetSignature(tokens, start, end, structs, sigs, importMethods)
}

func aliasInitializerTargetAt(tokens []scan.Token, stmtStart int, stmtEnd int, pos int, sigs []funcSignature, importFuncs []importedFunction) bool {
	if stmtStart < stmtEnd && tokens[stmtStart].Text == "var" {
		eq := findTopLevelToken(tokens, stmtStart+1, stmtEnd, "=")
		if eq < 0 || eq != stmtStart+2 {
			return false
		}
		_, ok := functionAliasTargetSignature(tokens, eq+1, stmtEnd, sigs, importFuncs)
		return ok && pos >= eq+1 && pos < stmtEnd
	}
	assign := findTopLevelToken(tokens, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		return false
	}
	lhs := expressionRanges(tokens, stmtStart, assign)
	rhs := expressionRanges(tokens, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return false
	}
	for i := 0; i < len(rhs); i++ {
		if pos < rhs[i].start || pos >= rhs[i].end {
			continue
		}
		_, ok := functionAliasTargetSignature(tokens, rhs[i].start, rhs[i].end, sigs, importFuncs)
		return ok
	}
	return false
}

func enclosingFunctionBodyOpen(tokens []scan.Token, pos int) int {
	for i := pos - 1; i >= 0; i-- {
		if tokens[i].Text != "{" {
			continue
		}
		close := findClose(tokens, i, "{", "}")
		if close <= pos {
			continue
		}
		if blockOwnerKeyword(tokens, i) == "func" {
			return i
		}
	}
	return -1
}

func isMethodExpressionUse(tokens []scan.Token, pos int, topTypes []string, importNames []string) bool {
	if pos < 0 || pos+2 >= len(tokens) {
		return false
	}
	if tokens[pos].Kind != scan.Ident || tokens[pos+1].Text != "." || tokens[pos+2].Kind != scan.Ident {
		return false
	}
	if isSelectorMember(tokens, pos) {
		return false
	}
	if pos+3 < len(tokens) && tokens[pos+3].Text == "(" {
		return false
	}
	if functionValueBlankDiscardExpressionAt(tokens, pos, pos+3) {
		return false
	}
	name := tokens[pos].Text
	if containsString(importNames, name) {
		return false
	}
	return containsString(topTypes, name)
}

func firstExpressionToken(tokens []scan.Token, start int, end int) int {
	for i := start; i < end; i++ {
		if tokens[i].Text == "," {
			continue
		}
		return i
	}
	return -1
}

func isAssignableStart(tokens []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	tok := tokens[pos]
	return tok.Kind == scan.Ident || tok.Text == "*" || tok.Text == "("
}

func isSelectorMember(tokens []scan.Token, pos int) bool {
	return pos > 0 && tokens[pos-1].Text == "."
}

func knownIdentifier(name string, topNames []string, locals []string, imports []string) bool {
	if name == "_" || name == "true" || name == "false" || name == "nil" || name == "iota" {
		return true
	}
	if isBuiltinCallable(name) || isBuiltinTypeName(name) || isRuntimeConstantName(name) {
		return true
	}
	return containsString(topNames, name) || containsString(locals, name) || containsString(imports, name)
}

func dotImportedIdentifierKnown(name string, funcs []importedFunction, values []importedValue, typeNames []localValueType, structs []structFieldSet) bool {
	if importedFunctionIndex(funcs, ".", name) >= 0 {
		return true
	}
	for i := 0; i < len(values); i++ {
		if values[i].qualifier == "." && values[i].name == name {
			return true
		}
	}
	if localValueTypeName(typeNames, name) != "" {
		return true
	}
	return structFieldSetIndex(structs, name) >= 0
}

func isBuiltinCallable(name string) bool {
	switch name {
	case "append", "cap", "copy", "len", "make", "panic", "print", "println", "recover", "syscall", "string", "int", "int64", "byte", "bool", "float64", "int16", "int32":
		return true
	}
	return false
}

func isBuiltinTypeName(name string) bool {
	switch name {
	case "int", "int64", "byte", "bool", "string", "error", "float64", "int16", "int32":
		return true
	}
	return false
}

func isRuntimeConstantName(name string) bool {
	switch name {
	case "O_RDONLY", "O_WRONLY", "O_RDWR", "O_CREATE", "O_TRUNC":
		return true
	}
	return false
}

func shouldCheckUndefinedIdent(tokens []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(tokens) {
		return false
	}
	tok := tokens[pos]
	if tok.Kind != scan.Ident || isKeyword(tok.Text) {
		return false
	}
	if isSelectorMember(tokens, pos) {
		return false
	}
	if pos+1 < len(tokens) && (tokens[pos+1].Text == "." || tokens[pos+1].Text == ":" || tokens[pos+1].Text == ":=") {
		return false
	}
	if pos > 0 && (tokens[pos-1].Text == "goto" || tokens[pos-1].Text == "break" || tokens[pos-1].Text == "continue" || tokens[pos-1].Text == "func" || tokens[pos-1].Text == "type") {
		return false
	}
	if isCompositeKey(tokens, pos) {
		return false
	}
	if nameInFunctionSignature(tokens, pos) || nameInStructFieldList(tokens, pos) {
		return false
	}
	return true
}

func usedImportNames(file parse.File, importNames []string) []string {
	var used []string
	shadows := localImportShadows(file, importNames)
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i+1 < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		var next scan.Token
		next = tokens[i+1]
		if tok.Kind == scan.Ident && next.Text == "." {
			if isLocalShadowAt(shadows, tok.Text, int(tok.Start)) {
				continue
			}
			name := tok.Text
			if !containsString(used, name) {
				used = append(used, name)
			}
		}
	}
	return used
}

func importLocalName(imp parse.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return load.PackageNameFromImportPath(imp.Path)
}

func ellipsisAllowed(tokens []scan.Token, pos int) bool {
	return ellipsisInFunctionSignature(tokens, pos) || ellipsisInFinalCallArgument(tokens, pos) || ellipsisInInferredArrayLength(tokens, pos)
}

func ellipsisInInferredArrayLength(tokens []scan.Token, pos int) bool {
	return pos > 0 && pos+1 < len(tokens) && tokens[pos-1].Text == "[" && tokens[pos+1].Text == "]"
}

func ellipsisInFunctionSignature(tokens []scan.Token, pos int) bool {
	open := containingOpen(tokens, pos, "(", ")")
	if open < 0 {
		return false
	}
	if open > 0 && tokens[open-1].Text == "func" {
		return true
	}
	if open > 1 && tokens[open-2].Text == "func" && tokens[open-1].Kind == scan.Ident {
		return true
	}
	if open > 2 && tokens[open-2].Text == ")" && tokens[open-1].Kind == scan.Ident {
		receiverOpen := findOpen(tokens, open-2, "(", ")")
		return receiverOpen > 0 && tokens[receiverOpen-1].Text == "func"
	}
	return false
}

func ellipsisInFinalCallArgument(tokens []scan.Token, pos int) bool {
	if pos+1 >= len(tokens) || tokens[pos+1].Text != ")" {
		return false
	}
	open := containingOpen(tokens, pos, "(", ")")
	if open <= 0 {
		return false
	}
	prev := tokens[open-1]
	return prev.Kind == scan.Ident || prev.Text == ")"
}

func startsGenericDecl(file parse.File, i int) bool {
	toks := file.Tokens
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text == "type" && toks[i+1].Kind == scan.Ident && toks[i+2].Text == "[" {
		close := findClose(toks, i+2, "[", "]")
		return close > i+4
	}
	if toks[i].Text == "func" && file.IsTopLevelFuncAt(i) {
		namePos := i + 1
		if toks[namePos].Text == "(" {
			close := findClose(toks, namePos, "(", ")")
			if close < 0 || close+2 >= len(toks) {
				return false
			}
			namePos = close + 1
		}
		if toks[namePos].Kind != scan.Ident || toks[namePos+1].Text != "[" {
			return false
		}
		close := findClose(toks, namePos+1, "[", "]")
		return close > namePos+3
	}
	return false
}

func startsGenericInstantiation(toks []scan.Token, i int) bool {
	if i+3 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "[" {
		return false
	}
	close := findClose(toks, i+1, "[", "]")
	if close < 0 {
		return false
	}
	if close+1 >= len(toks) {
		return false
	}
	if toks[close+1].Text == "{" && isControlBlockOpen(toks, close+1) {
		return false
	}
	return toks[close+1].Text == "{" || toks[close+1].Text == "("
}

func isControlBlockOpen(toks []scan.Token, open int) bool {
	parenDepth := 0
	brackDepth := 0
	for i := open - 1; i >= 0; i-- {
		text := toks[i].Text
		if text == ")" {
			parenDepth++
			continue
		}
		if text == "(" && parenDepth > 0 {
			parenDepth--
			continue
		}
		if text == "]" {
			brackDepth++
			continue
		}
		if text == "[" && brackDepth > 0 {
			brackDepth--
			continue
		}
		if parenDepth != 0 || brackDepth != 0 {
			continue
		}
		if text == "if" || text == "for" || text == "switch" || text == "select" {
			return true
		}
		if text == "{" || text == "}" || text == "func" {
			return false
		}
	}
	return false
}

func isImaginaryLiteral(text string) bool {
	return strings.HasSuffix(text, "i")
}

func startsArrayType(toks []scan.Token, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Text != "[" {
		return false
	}
	if toks[i+1].Text == "]" {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	if prev.Text == "map" {
		return false
	}
	if nameInStructFieldList(toks, i) {
		return true
	}
	if prev.Text == "*" {
		return precededByTypeContext(toks, i-1)
	}
	if prev.Text == "]" {
		open := findOpen(toks, i-1, "[", "]")
		if open < 0 || open+1 != i-1 {
			return false
		}
		if precededByTypeContext(toks, open) || expressionTypeContext(toks, open) {
			return true
		}
		return open > 0 && toks[open-1].Kind == scan.Ident && nameInStructFieldList(toks, open-1)
	}
	if prev.Text == ")" {
		return closesFunctionSignature(toks, i-1)
	}
	if prev.Kind != scan.Ident {
		return expressionTypeContext(toks, i)
	}
	if nameInStructFieldList(toks, i-1) {
		return true
	}
	if nameInFunctionSignature(toks, i-1) {
		return true
	}
	return precededByTypeContext(toks, i-1)
}

func frontendLowerableArrayType(file parse.File, pos int) bool {
	toks := file.Tokens
	if arrayTypeInNamedFixedArrayTypeDecl(file, pos) {
		return true
	}
	if arrayTypeInLocalNamedFixedArrayTypeDecl(file, pos) {
		return true
	}
	if arrayTypeInFunctionResult(toks, pos) {
		return true
	}
	if arrayTypeInFunctionParameter(toks, pos) {
		return true
	}
	if arrayTypeInStructField(toks, pos) {
		return true
	}
	close := arrayCompositeLiteralClose(toks, pos)
	if close >= 0 && arrayLiteralIsDirectUnsafeSizeofArg(file, toks, pos, close) {
		return true
	}
	if nestedArrayTypeInDirectUnsafeSizeofArg(file, toks, pos) {
		return true
	}
	if arrayTypeInPackageVarDecl(file, pos) {
		return true
	}
	if close >= 0 && arrayCompositeLiteralInPackageVarInitializer(file, pos, close) {
		return true
	}
	body := functionBodyOpenContainingToken(file, pos)
	if body < 0 {
		return false
	}
	if arrayTypeInLocalVarDecl(toks, body, pos) {
		return true
	}
	if close < 0 {
		return false
	}
	return arrayCompositeLiteralLowerableContext(toks, body, pos, close)
}

func arrayTypeInLocalNamedFixedArrayTypeDecl(file parse.File, pos int) bool {
	toks := file.Tokens
	body := functionBodyOpenContainingToken(file, pos)
	if body < 0 {
		return false
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		return false
	}
	for i := body + 1; i < close; i++ {
		if toks[i].Text != "type" || !startsLocalTypeDeclToken(toks, i, body) {
			continue
		}
		if i+1 < close && toks[i+1].Text == "(" {
			groupClose := findClose(toks, i+1, "(", ")")
			if groupClose < 0 || groupClose > close {
				continue
			}
			specs := localTypeSpecRanges(toks, i+2, groupClose)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				if namedFixedArrayTypeSpecRange(toks, specs[specIndex].start, specs[specIndex].end, pos) {
					return true
				}
			}
			i = groupClose
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, close)
		if namedFixedArrayTypeSpecRange(toks, i+1, specEnd, pos) {
			return true
		}
		if specEnd > i {
			i = specEnd - 1
		}
	}
	return false
}

func arrayTypeInNamedFixedArrayTypeDecl(file parse.File, pos int) bool {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "type" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 || pos <= start || int(toks[pos].Start) >= decl.End {
			continue
		}
		end := tokenIndexBefore(toks, decl.End) + 1
		if end <= start+2 {
			continue
		}
		if start+1 < end && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 || close >= end {
				continue
			}
			specs := localTypeSpecRanges(toks, start+2, close)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				if namedFixedArrayTypeSpecRange(toks, specs[specIndex].start, specs[specIndex].end, pos) {
					return true
				}
			}
			continue
		}
		if namedFixedArrayTypeSpecRange(toks, start+1, end, pos) {
			return true
		}
	}
	return false
}

func namedFixedArrayTypeSpecRange(toks []scan.Token, start int, end int, pos int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if pos != typeStart || typeStart+3 > end || toks[typeStart].Text != "[" {
		return false
	}
	close := findClose(toks, typeStart, "[", "]")
	if close != typeStart+2 || close+1 >= end {
		return false
	}
	if toks[typeStart+1].Kind != scan.Number {
		return false
	}
	elem := typeTextInRange(toks, close+1, end)
	return elem != "" && !strings.HasPrefix(elem, "[]")
}

func arrayTypeInPackageVarDecl(file parse.File, pos int) bool {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 || pos <= start || int(toks[pos].Start) >= decl.End {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 {
				continue
			}
			specs := localTypeSpecRanges(toks, start+2, close)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				spec := specs[specIndex]
				if pos >= spec.start && pos < spec.end && localVarSpecArrayTypeStart(toks, spec.start, spec.end, pos) {
					return true
				}
			}
			continue
		}
		end := tokenIndexBefore(toks, decl.End) + 1
		if end <= start+1 {
			continue
		}
		if localVarSpecArrayTypeStart(toks, start+1, end, pos) {
			return true
		}
	}
	return false
}

func arrayCompositeLiteralInPackageVarInitializer(file parse.File, pos int, close int) bool {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 || pos <= start || close <= pos || int(toks[close].End) > decl.End {
			continue
		}
		declEnd := tokenIndexBefore(toks, decl.End) + 1
		if declEnd <= start+1 {
			continue
		}
		eq := findTopLevelToken(toks, start+1, declEnd, "=")
		return eq >= 0 && pos > eq
	}
	return false
}

func functionBodyOpenContainingToken(file parse.File, pos int) int {
	toks := file.Tokens
	if pos < 0 || pos >= len(toks) {
		return -1
	}
	tokenStart := int(toks[pos].Start)
	decls := file.Decls
	for i := 0; i < len(decls); i++ {
		decl := decls[i]
		if decl.Kind != "func" || tokenStart <= decl.Start || tokenStart >= decl.End {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		body := findTokenText(toks, start, decl.End, "{")
		if body >= 0 && pos > body {
			return body
		}
	}
	return -1
}

func arrayTypeInLocalVarDecl(toks []scan.Token, body int, pos int) bool {
	stmtStart := simpleStatementStart(toks, body, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if stmtStart < stmtEnd && toks[stmtStart].Text == "var" {
		return localVarSpecArrayTypeStart(toks, stmtStart+1, stmtEnd, pos)
	}
	if !insideVarGroupSpec(toks, stmtStart) {
		return false
	}
	return localVarSpecArrayTypeStart(toks, stmtStart, stmtEnd, pos)
}

func insideVarGroupSpec(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		switch toks[i].Text {
		case "(":
			return i > 0 && toks[i-1].Text == "var"
		case ")", "{", "}":
			return false
		}
	}
	return false
}

func localVarSpecArrayTypeStart(toks []scan.Token, start int, end int, pos int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	for i := start; i < end; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			continue
		}
		return i == pos
	}
	return false
}

func arrayCompositeLiteralClose(toks []scan.Token, start int) int {
	open := arrayCompositeLiteralOpen(toks, start)
	if open < 0 {
		return -1
	}
	return findClose(toks, open, "{", "}")
}

func arrayCompositeLiteralOpen(toks []scan.Token, start int) int {
	if start < 0 || start+1 >= len(toks) || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || !arrayLengthSupported(toks, start+1, brackClose) {
		return -1
	}
	elemStart := brackClose + 1
	if elemStart >= len(toks) {
		return -1
	}
	paren := 0
	brack := 0
	for i := elemStart; i < len(toks); i++ {
		if paren == 0 && brack == 0 {
			switch toks[i].Text {
			case "{":
				return i
			case ";", ")", ",", "=", ":=", "==", "!=", "<", "<=", ">", ">=":
				return -1
			}
		}
		switch toks[i].Text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			brack++
		case "]":
			brack--
		}
	}
	return -1
}

func arrayLengthSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if start+1 == end && toks[start].Kind == scan.Number {
		return true
	}
	return start+1 == end && toks[start].Text == "..."
}

func arrayCompositeLiteralLowerableContext(toks []scan.Token, body int, start int, close int) bool {
	if arrayLiteralInShortDecl(toks, body, start, close) {
		return true
	}
	if arrayLiteralInArrayVarDecl(toks, body, start, close) {
		return true
	}
	if arrayLiteralIsDirectLenCapArg(toks, start, close) {
		return true
	}
	if arrayLiteralIsDirectIndexBase(toks, start, close) {
		return true
	}
	if arrayLiteralIsRangeOperand(toks, start, close) {
		return true
	}
	if arrayLiteralIsCallArgument(toks, start, close) {
		return true
	}
	if arrayLiteralIsReturnValue(toks, start, close) {
		return true
	}
	if arrayLiteralIsComparisonOperand(toks, start, close) {
		return true
	}
	if arrayLiteralIsBlankAssignmentValue(toks, body, start, close) {
		return true
	}
	return arrayLiteralIsCompositeLiteralValue(toks, start)
}

func arrayTypeInFunctionResult(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return false
	}
	if arrayTypeStartsDirectFunctionResult(toks, pos) {
		end := functionParameterArrayTypeEnd(toks, pos)
		return simpleFixedArrayType(toks, pos, end)
	}
	open := containingOpen(toks, pos, "(", ")")
	if !functionResultListOpen(toks, open) {
		return false
	}
	if !arrayTypeStartsFunctionParameterType(toks, open, pos) {
		return false
	}
	end := functionParameterArrayTypeEnd(toks, pos)
	return simpleFixedArrayType(toks, pos, end)
}

func arrayTypeStartsDirectFunctionResult(toks []scan.Token, pos int) bool {
	if pos <= 0 || toks[pos-1].Text != ")" {
		return false
	}
	paramsOpen := findOpen(toks, pos-1, "(", ")")
	return functionParameterListOpen(toks, paramsOpen)
}

func functionResultListOpen(toks []scan.Token, open int) bool {
	if open <= 0 || toks[open-1].Text != ")" {
		return false
	}
	paramsOpen := findOpen(toks, open-1, "(", ")")
	return functionParameterListOpen(toks, paramsOpen)
}

func arrayTypeInFunctionParameter(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return false
	}
	open := containingOpen(toks, pos, "(", ")")
	if !functionParameterListOpen(toks, open) {
		return false
	}
	if !arrayTypeStartsFunctionParameterType(toks, open, pos) {
		return false
	}
	end := functionParameterArrayTypeEnd(toks, pos)
	return simpleFixedArrayType(toks, pos, end)
}

func functionParameterListOpen(toks []scan.Token, open int) bool {
	if open < 0 {
		return false
	}
	if open > 0 && toks[open-1].Text == "func" {
		return true
	}
	if open > 1 && toks[open-2].Text == "func" && toks[open-1].Kind == scan.Ident {
		return true
	}
	if open > 1 && toks[open-1].Kind == scan.Ident && toks[open-2].Text == ")" {
		receiverOpen := findOpen(toks, open-2, "(", ")")
		return receiverOpen > 0 && toks[receiverOpen-1].Text == "func"
	}
	return false
}

func arrayTypeStartsFunctionParameterType(toks []scan.Token, open int, pos int) bool {
	if pos <= open {
		return false
	}
	prev := toks[pos-1]
	if prev.Text == "(" || prev.Text == "," {
		return true
	}
	if prev.Kind != scan.Ident || pos-2 < open {
		return false
	}
	beforeName := toks[pos-2].Text
	return beforeName == "(" || beforeName == ","
}

func functionParameterArrayTypeEnd(toks []scan.Token, pos int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := pos; i < len(toks); i++ {
		if i > pos && paren == 0 && brack == 0 && brace == 0 {
			switch toks[i].Text {
			case ",", ")", "{", "=", ";":
				return i
			}
		}
		updateDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func arrayTypeInStructField(toks []scan.Token, pos int) bool {
	if pos <= 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return false
	}
	if toks[pos-1].Kind != scan.Ident || !nameInStructFieldList(toks, pos-1) {
		return false
	}
	end := structFieldSpecEnd(toks, pos)
	return simpleFixedArrayType(toks, pos, end)
}

func fixedArrayTypeSpecRange(toks []scan.Token, start int, end int) bool {
	if start < 0 || start+1 >= end || toks[start].Text != "[" {
		return false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return false
	}
	if !fixedArrayLengthSupported(toks, start+1, brackClose) {
		return false
	}
	elemStart := brackClose + 1
	if elemStart >= end {
		return false
	}
	if end > elemStart && toks[end-1].Kind == scan.String {
		end--
	}
	return typeTextInRange(toks, elemStart, end) != ""
}

func simpleFixedArrayType(toks []scan.Token, start int, end int) bool {
	if start < 0 || start+1 >= end || toks[start].Text != "[" {
		return false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return false
	}
	if !fixedArrayLengthSupported(toks, start+1, brackClose) {
		return false
	}
	elemStart := brackClose + 1
	if elemStart >= end || toks[elemStart].Text == "[" {
		return false
	}
	if end > elemStart && toks[end-1].Kind == scan.String {
		end--
	}
	return typeTextInRange(toks, elemStart, end) != ""
}

func fixedArrayLengthSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	return start+1 == end && toks[start].Kind == scan.Number
}

func arrayLiteralIsCompositeLiteralValue(toks []scan.Token, start int) bool {
	open := containingOpen(toks, start, "{", "}")
	if open <= 0 {
		return false
	}
	prev := toks[open-1]
	if prev.Kind == scan.Ident {
		if open >= 2 {
			before := toks[open-2].Text
			if before == ")" || before == "func" || before == "if" || before == "for" || before == "switch" {
				return false
			}
		}
		return true
	}
	return prev.Text == "]" || prev.Text == "}"
}

func arrayLiteralInShortDecl(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := simpleStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 || start <= assign {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			return true
		}
	}
	return false
}

func arrayLiteralInArrayVarDecl(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := simpleStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || start <= assign {
		return false
	}
	typeStart := -1
	if stmtStart < stmtEnd && toks[stmtStart].Text == "var" {
		_, typeStart = localVarSpecNamesAndTypeForArrayContext(toks, stmtStart+1, assign)
	} else if insideVarGroupSpec(toks, stmtStart) {
		_, typeStart = localVarSpecNamesAndTypeForArrayContext(toks, stmtStart, assign)
	}
	if typeStart < 0 || !startsArrayType(toks, typeStart) {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			return true
		}
	}
	return false
}

func localVarSpecNamesAndTypeForArrayContext(toks []scan.Token, start int, end int) ([]string, int) {
	var names []string
	typeStart := -1
	for i := start; i < end; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			names = append(names, toks[i].Text)
			continue
		}
		typeStart = i
		break
	}
	return names, typeStart
}

func expressionMatchesTokenRange(toks []scan.Token, expr expressionRange, start int, end int) bool {
	expr.start, expr.end = trimExpressionRange(toks, expr.start, expr.end)
	return expr.start == start && expr.end == end
}

func arrayLiteralIsDirectLenCapArg(toks []scan.Token, start int, close int) bool {
	if start < 2 || close+1 >= len(toks) || toks[start-1].Text != "(" || toks[close+1].Text != ")" {
		return false
	}
	return toks[start-2].Text == "len" || toks[start-2].Text == "cap"
}

func arrayLiteralIsDirectUnsafeSizeofArg(file parse.File, toks []scan.Token, start int, close int) bool {
	if !compositeLiteralIsDirectUnsafeSizeofArg(file, toks, start, close) {
		return false
	}
	return unsafeSizeofFixedArrayLiteralOperand(toks, start, close+1)
}

func compositeLiteralIsDirectUnsafeSizeofArg(file parse.File, toks []scan.Token, start int, close int) bool {
	if start < 2 || close+1 >= len(toks) || toks[start-1].Text != "(" || toks[close+1].Text != ")" {
		return false
	}
	if toks[start-2].Text == "Sizeof" && unsafeDotImported(file) {
		return true
	}
	if start < 4 || toks[start-2].Text != "Sizeof" || toks[start-3].Text != "." || toks[start-4].Kind != scan.Ident {
		return false
	}
	return importNamePath(file, toks[start-4].Text) == "unsafe"
}

func nestedArrayTypeInDirectUnsafeSizeofArg(file parse.File, toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return false
	}
	for start := pos - 1; start >= 0; start-- {
		if toks[start].Text == ";" || toks[start].Text == "{" || toks[start].Text == "}" {
			break
		}
		if toks[start].Text != "[" {
			continue
		}
		close := arrayCompositeLiteralClose(toks, start)
		if close < 0 || pos <= start || pos >= close {
			continue
		}
		if arrayLiteralIsDirectUnsafeSizeofArg(file, toks, start, close) {
			return true
		}
	}
	return false
}

func unsafeDotImported(file parse.File) bool {
	imports := file.Imports
	for i := 0; i < len(imports); i++ {
		if imports[i].Alias == "." && imports[i].Path == "unsafe" {
			return true
		}
	}
	return false
}

func importNamePath(file parse.File, name string) string {
	imports := file.Imports
	for i := 0; i < len(imports); i++ {
		if importLocalName(imports[i]) == name {
			return imports[i].Path
		}
	}
	return ""
}

func arrayLiteralIsDirectIndexBase(toks []scan.Token, start int, close int) bool {
	next := close + 1
	if next < len(toks) && toks[next].Text == "[" {
		return true
	}
	if start > 0 && toks[start-1].Text == "(" && next+1 < len(toks) && toks[next].Text == ")" && toks[next+1].Text == "[" {
		return true
	}
	return false
}

func arrayLiteralIsRangeOperand(toks []scan.Token, start int, close int) bool {
	if start > 0 && toks[start-1].Text == "range" {
		return true
	}
	if start <= 0 {
		return false
	}
	for i := start - 1; i >= 0; i-- {
		if toks[i].Text == "range" {
			end := rangeOperandEnd(toks, i+1, len(toks))
			return end == close+1
		}
		if toks[i].Text == "{" || toks[i].Text == "}" || toks[i].Text == ";" {
			return false
		}
	}
	return false
}

func arrayLiteralIsCallArgument(toks []scan.Token, start int, close int) bool {
	open := containingOpen(toks, start, "(", ")")
	if open < 0 || open+1 > start {
		return false
	}
	if open == 0 || toks[open-1].Text == "func" || toks[open-1].Text == "if" || toks[open-1].Text == "for" || toks[open-1].Text == "switch" {
		return false
	}
	parenClose := findClose(toks, open, "(", ")")
	if parenClose < close {
		return false
	}
	args := expressionRanges(toks, open+1, parenClose)
	for i := 0; i < len(args); i++ {
		if expressionMatchesTokenRange(toks, args[i], start, close+1) {
			return true
		}
	}
	return false
}

func arrayLiteralIsReturnValue(toks []scan.Token, start int, close int) bool {
	for i := start - 1; i >= 0; i-- {
		if toks[i].Text == "return" {
			stmtEnd := simpleStatementEnd(toks, i+1, len(toks))
			values := expressionRanges(toks, i+1, stmtEnd)
			for valueIndex := 0; valueIndex < len(values); valueIndex++ {
				if expressionMatchesTokenRange(toks, values[valueIndex], start, close+1) {
					return true
				}
			}
			return false
		}
		if toks[i].Text == "{" || toks[i].Text == "}" || toks[i].Text == ";" {
			return false
		}
	}
	return false
}

func arrayLiteralIsComparisonOperand(toks []scan.Token, start int, close int) bool {
	prev := start - 1
	if prev >= 0 && toks[prev].Text == "(" {
		prev--
	}
	if prev >= 0 && (toks[prev].Text == "==" || toks[prev].Text == "!=") {
		return true
	}
	next := close + 1
	if next < len(toks) && toks[next].Text == ")" {
		next++
	}
	return next < len(toks) && (toks[next].Text == "==" || toks[next].Text == "!=")
}

func arrayLiteralIsBlankAssignmentValue(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := simpleStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || start <= assign {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	matched := false
	for i := 0; i < len(lhs); i++ {
		if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
			return false
		}
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			matched = true
		}
		if !discardableArrayLiteralExpression(toks, rhs[i].start, rhs[i].end) {
			return false
		}
	}
	return matched
}

func discardableArrayLiteralExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return false
	}
	close := arrayCompositeLiteralClose(toks, start)
	if close != end-1 {
		return false
	}
	open := arrayCompositeLiteralOpen(toks, start)
	if open < 0 {
		return false
	}
	values := expressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardableArrayLiteralElement(toks, value.start, colon) {
				return false
			}
			value.start = colon + 1
		}
		if !discardableArrayLiteralElement(toks, value.start, value.end) {
			return false
		}
	}
	return true
}

func discardedPureArrayCompositeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedArrayCompositeStatementContainingToken(toks, pos, discardablePureArrayCompositeExpression)
}

func discardedLowerableArrayCompositeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedArrayCompositeStatementContainingToken(toks, pos, discardableLowerableArrayCompositeExpression)
}

func discardedArrayCompositeStatementContainingToken(toks []scan.Token, pos int, supported func([]scan.Token, int, int) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	matched := false
	for i := 0; i < len(lhs); i++ {
		if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
			return false
		}
		if pos >= rhs[i].start && pos < rhs[i].end {
			matched = true
		}
		if !supported(toks, rhs[i].start, rhs[i].end) {
			return false
		}
	}
	return matched
}

func sameLineAssignmentStatementStart(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" {
			return i + 1
		}
	}
	return 0
}

func discardablePureArrayCompositeExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardablePureArrayCompositeExpression(toks, start+1, close)
		}
	}
	open := pureArrayCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardablePureCompositeElements(toks, open+1, close)
}

func discardableLowerableArrayCompositeExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableArrayCompositeExpression(toks, start+1, close)
		}
	}
	open := pureArrayCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardableLowerableCompositeElements(toks, open+1, close)
}

func pureArrayCompositeLiteralOpen(toks []scan.Token, start int, end int) int {
	if start < 0 || start+1 >= end || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose < 0 || brackClose+1 >= end {
		return -1
	}
	if brackClose > start+1 && !arrayLengthSupported(toks, start+1, brackClose) {
		return -1
	}
	paren := 0
	brack := 0
	for i := brackClose + 1; i < end; i++ {
		if paren == 0 && brack == 0 {
			switch toks[i].Text {
			case "{":
				return i
			case ";", ")", ",", "=", ":=", "==", "!=", "<", "<=", ">", ">=":
				return -1
			}
		}
		switch toks[i].Text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			brack++
		case "]":
			brack--
		}
	}
	return -1
}

func discardablePureCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardablePureCompositeKey(toks, value.start, colon) {
				return false
			}
			value.start = colon + 1
		}
		if !discardablePureCompositeValue(toks, value.start, value.end) {
			return false
		}
	}
	return true
}

func discardableLowerableCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardablePureCompositeKey(toks, value.start, colon) {
				return false
			}
			value.start = colon + 1
		}
		if !discardableLowerableCompositeValue(toks, value.start, value.end) {
			return false
		}
	}
	return true
}

func discardablePureCompositeKey(toks []scan.Token, start int, end int) bool {
	_, imaginary, ok := signedNumberLiteralText(toks, start, end)
	return ok && !imaginary
}

func discardablePureCompositeValue(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardablePureCompositeElements(toks, start+1, close)
	}
	if toks[start].Text == "[" {
		return discardablePureArrayCompositeExpression(toks, start, end)
	}
	return discardableArrayLiteralElement(toks, start, end)
}

func discardableLowerableCompositeValue(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardableLowerableCompositeElements(toks, start+1, close)
	}
	if toks[start].Text == "[" {
		return discardableLowerableArrayCompositeExpression(toks, start, end)
	}
	if discardableArrayLiteralElement(toks, start, end) {
		return true
	}
	return directCallExpressionWithoutCallArgs(toks, start, end)
}

func directCallExpressionWithoutCallArgs(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directCallExpressionWithoutCallArgs(toks, start+1, close)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	for i := start + 2; i < close; i++ {
		if i+1 < close && toks[i].Kind == scan.Ident && toks[i+1].Text == "(" {
			return false
		}
	}
	return true
}

func discardableArrayLiteralElement(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if start+1 == end {
		tok := toks[start]
		return tok.Kind == scan.Number || tok.Kind == scan.String || tok.Kind == scan.Char || tok.Text == "true" || tok.Text == "false" || tok.Text == "nil"
	}
	if start+2 == end && (toks[start].Text == "-" || toks[start].Text == "+") {
		return toks[start+1].Kind == scan.Number
	}
	return false
}

func startsAnonymousStructSliceType(toks []scan.Token, i int) bool {
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text != "[" || toks[i+1].Text != "]" || toks[i+2].Text != "struct" {
		return false
	}
	return precededByTypeContext(toks, i) || expressionTypeContext(toks, i)
}

func frontendLowerableAnonymousStructType(file parse.File, pos int) bool {
	toks := file.Tokens
	structOpen := pos + 1
	if toks[structOpen].Text != "{" {
		return false
	}
	structClose := findClose(toks, structOpen, "{", "}")
	if structClose < 0 {
		return false
	}
	if anonymousStructTypeInNamedStructField(toks, pos, structClose) {
		return true
	}
	if anonymousStructTypeInFunctionSignature(file, pos, structClose) {
		return true
	}
	if anonymousStructTypeInTopLevelNamedSliceTypeSpec(file, pos, structClose) {
		return true
	}
	if anonymousStructTypeInTopLevelVarDecl(file, pos, structClose) {
		return true
	}
	body := functionBodyOpenContainingToken(file, pos)
	if body < 0 {
		if structClose+1 < len(toks) && toks[structClose+1].Text == "{" && anonymousStructLiteralInTopLevelVarDecl(file, pos, structClose) {
			return true
		}
		return false
	}
	if anonymousStructTypeInLocalVarDecl(toks, body, pos, structClose) {
		return true
	}
	if structClose+1 >= len(toks) || toks[structClose+1].Text != "{" {
		return false
	}
	literalClose := findClose(toks, structClose+1, "{", "}")
	if literalClose < 0 {
		return false
	}
	if anonymousStructLiteralMatchesDeclaredAnonymousType(file, pos, structClose) {
		return true
	}
	if pos >= 2 && toks[pos-2].Text == "[" && toks[pos-1].Text == "]" && anonymousStructSliceLiteralInShortDecl(toks, body, pos-2, literalClose) {
		return true
	}
	return anonymousStructLiteralInShortDecl(toks, body, pos, literalClose) || anonymousStructLiteralInVarDecl(toks, body, pos, literalClose)
}

func anonymousStructLiteralMatchesDeclaredAnonymousType(file parse.File, pos int, structClose int) bool {
	toks := file.Tokens
	key := anonymousStructTypeKey(toks, pos, structClose+1)
	if key == "" {
		return false
	}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "struct" || toks[i+1].Text != "{" {
			continue
		}
		close := findClose(toks, i+1, "{", "}")
		if close < 0 {
			continue
		}
		if anonymousStructTypeKey(toks, i, close+1) != key {
			continue
		}
		if anonymousStructTypeInNamedStructField(toks, i, close) || anonymousStructTypeInFunctionSignature(file, i, close) || anonymousStructTypeInTopLevelNamedSliceTypeSpec(file, i, close) || anonymousStructTypeInTopLevelVarDecl(file, i, close) || anonymousStructLiteralInTopLevelVarDecl(file, i, close) {
			return true
		}
	}
	return false
}

func anonymousStructTypeKey(toks []scan.Token, start int, end int) string {
	if start < 0 || end > len(toks) || start >= end {
		return ""
	}
	var out []byte
	for i := start; i < end; i++ {
		out = append(out, toks[i].Text...)
	}
	return string(out)
}

func anonymousStructTypeInNamedStructField(toks []scan.Token, pos int, structClose int) bool {
	outerOpen := anonymousStructContainingStructOpen(toks, pos)
	if outerOpen < 0 || outerOpen+1 >= len(toks) {
		return false
	}
	if outerOpen == pos+1 || toks[outerOpen-1].Text != "struct" || !startsNamedStructType(toks, outerOpen-1) {
		return false
	}
	outerClose := findClose(toks, outerOpen, "{", "}")
	if outerClose < 0 || pos >= outerClose {
		return false
	}
	ranges := anonymousStructFieldSpecRanges(toks, outerOpen, outerClose)
	for i := 0; i < len(ranges); i++ {
		typeStart, typeEnd, ok := anonymousStructFieldTypeRange(toks, ranges[i].start, ranges[i].end)
		if ok && typeStart == pos && typeEnd == structClose+1 {
			return true
		}
	}
	return false
}

func anonymousStructTypeInFunctionSignature(file parse.File, pos int, structClose int) bool {
	toks := file.Tokens
	tokenStart := int(toks[pos].Start)
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "func" || tokenStart <= decl.Start || tokenStart >= decl.End {
			continue
		}
		name := tokenIndexAt(toks, int(decl.NameTok.Start))
		if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
			continue
		}
		paramsOpen := name + 1
		paramsClose := findClose(toks, paramsOpen, "(", ")")
		if paramsClose < 0 {
			continue
		}
		body := functionBodyOpenAfterParams(toks, paramsClose, decl.End)
		if body < 0 {
			body = tokenIndexBefore(toks, decl.End)
		}
		if anonymousStructTypeInSignatureRange(toks, pos, structClose, paramsOpen+1, paramsClose) {
			return true
		}
		if anonymousStructTypeInSignatureRange(toks, pos, structClose, paramsClose+1, body) {
			return true
		}
	}
	return false
}

func anonymousStructTypeInTopLevelNamedSliceTypeSpec(file parse.File, pos int, structClose int) bool {
	toks := file.Tokens
	specs := topLevelTypeSpecRangesContainingToken(file, pos)
	for i := 0; i < len(specs); i++ {
		start := specs[i].start
		end := specs[i].end
		for start < end && toks[start].Text == ";" {
			start++
		}
		if start+3 >= end || toks[start].Kind != scan.Ident {
			continue
		}
		typeStart := start + 1
		if typeStart < end && toks[typeStart].Text == "=" {
			typeStart++
		}
		if typeStart+3 <= end && toks[typeStart].Text == "[" && toks[typeStart+1].Text == "]" && typeStart+2 == pos && toks[pos].Text == "struct" && toks[pos+1].Text == "{" && findClose(toks, pos+1, "{", "}") == structClose {
			return true
		}
	}
	return false
}

func topLevelTypeSpecRangesContainingToken(file parse.File, pos int) []expressionRange {
	var out []expressionRange
	toks := file.Tokens
	tokenStart := int(toks[pos].Start)
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "type" || tokenStart <= decl.Start || tokenStart >= decl.End {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close > start+1 {
				return localTypeSpecRanges(toks, start+2, close)
			}
			return nil
		}
		end := tokenIndexBefore(toks, decl.End) + 1
		if end > start+1 {
			out = append(out, expressionRange{start: start + 1, end: end})
		}
	}
	return out
}

func anonymousStructTypeInTopLevelVarDecl(file parse.File, pos int, structClose int) bool {
	toks := file.Tokens
	specs := topLevelVarSpecRangesContainingToken(file, pos)
	for i := 0; i < len(specs); i++ {
		typeStart, typeEnd, ok := anonymousStructTypeRangeInValueSpec(toks, specs[i].start, specs[i].end)
		if ok && typeStart == pos && typeEnd == structClose+1 {
			return true
		}
	}
	return false
}

func anonymousStructLiteralInTopLevelVarDecl(file parse.File, pos int, structClose int) bool {
	toks := file.Tokens
	specs := topLevelVarSpecRangesContainingToken(file, pos)
	for i := 0; i < len(specs); i++ {
		eq := findTopLevelToken(toks, specs[i].start, specs[i].end, "=")
		if eq < 0 || pos <= eq {
			continue
		}
		if pos+1 < specs[i].end && toks[pos].Text == "struct" && toks[pos+1].Text == "{" && findClose(toks, pos+1, "{", "}") == structClose {
			return true
		}
	}
	return false
}

func topLevelVarSpecRangesContainingToken(file parse.File, pos int) []expressionRange {
	var out []expressionRange
	toks := file.Tokens
	tokenStart := int(toks[pos].Start)
	for i := 0; i < len(file.Decls); i++ {
		decl := file.Decls[i]
		if decl.Kind != "var" || tokenStart <= decl.Start || tokenStart >= decl.End {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close > start+1 {
				return localTypeSpecRanges(toks, start+2, close)
			}
			return nil
		}
		end := tokenIndexBefore(toks, decl.End) + 1
		if end > start+1 {
			out = append(out, expressionRange{start: start + 1, end: end})
		}
	}
	return out
}

func anonymousStructTypeRangeInValueSpec(toks []scan.Token, start int, end int) (int, int, bool) {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	eq := findTopLevelToken(toks, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	for i := start; i+1 < lhsEnd; i++ {
		if toks[i].Text != "struct" || toks[i+1].Text != "{" {
			continue
		}
		close := findClose(toks, i+1, "{", "}")
		if close > i+1 && close < lhsEnd {
			return i, close + 1, true
		}
	}
	return 0, 0, false
}

func anonymousStructTypeInSignatureRange(toks []scan.Token, pos int, structClose int, start int, end int) bool {
	if start < 0 || end > len(toks) || pos < start || structClose >= end {
		return false
	}
	return pos >= 0 && pos+1 < len(toks) && toks[pos].Text == "struct" && toks[pos+1].Text == "{" && findClose(toks, pos+1, "{", "}") == structClose
}

func functionBodyOpenAfterParams(toks []scan.Token, paramsClose int, declEnd int) int {
	start := functionBodySearchStartAfterParams(toks, paramsClose, declEnd)
	if start < 0 {
		return -1
	}
	return findTokenText(toks, start, declEnd, "{")
}

func functionBodySearchStartAfterParams(toks []scan.Token, paramsClose int, declEnd int) int {
	start := paramsClose + 1
	for start < len(toks) && int(toks[start].Start) < declEnd && toks[start].Text == ";" {
		start++
	}
	if start >= len(toks) || int(toks[start].Start) >= declEnd {
		return -1
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close < 0 || int(toks[close].Start) >= declEnd {
			return -1
		}
		return close + 1
	}
	if toks[start].Text == "struct" && start+1 < len(toks) && toks[start+1].Text == "{" {
		close := findClose(toks, start+1, "{", "}")
		if close < 0 || int(toks[close].Start) >= declEnd {
			return -1
		}
		return close + 1
	}
	return start
}

func anonymousStructContainingStructOpen(toks []scan.Token, pos int) int {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "}" {
			depth++
			continue
		}
		if toks[i].Text == "{" {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func anonymousStructFieldSpecRanges(toks []scan.Token, open int, close int) []expressionRange {
	var ranges []expressionRange
	specStart := open + 1
	paren := 0
	brack := 0
	brace := 0
	for i := open + 1; i <= close; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if i == close || toks[i].Text == ";" {
				if specStart < i {
					ranges = append(ranges, expressionRange{start: specStart, end: i})
				}
				specStart = i + 1
				continue
			}
			if i > specStart && toks[i].Line > toks[i-1].Line {
				ranges = append(ranges, expressionRange{start: specStart, end: i})
				specStart = i
			}
		}
		updateDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return ranges
}

func anonymousStructFieldTypeRange(toks []scan.Token, start int, end int) (int, int, bool) {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Kind == scan.String {
		end--
	}
	if start >= end || toks[start].Text == "*" || toks[start].Kind != scan.Ident {
		return 0, 0, false
	}
	for i := start + 1; i < end; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && toks[i-1].Text == "," {
			continue
		}
		typeStart := anonymousStructFieldStructToken(toks, i, end)
		if typeStart >= 0 {
			close := findClose(toks, typeStart+1, "{", "}")
			if close > typeStart+1 && close < end {
				return typeStart, close + 1, true
			}
		}
		return 0, 0, false
	}
	return 0, 0, false
}

func anonymousStructFieldStructToken(toks []scan.Token, pos int, end int) int {
	if pos >= end {
		return -1
	}
	if toks[pos].Text == "[" {
		if pos+1 >= end || toks[pos+1].Text != "]" {
			return -1
		}
		pos += 2
	}
	for pos < end && toks[pos].Text == "*" {
		pos++
	}
	if pos+1 < end && toks[pos].Text == "struct" && toks[pos+1].Text == "{" {
		return pos
	}
	return -1
}

func anonymousStructSliceLiteralInShortDecl(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 || start <= assign {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			return true
		}
	}
	return false
}

func anonymousStructLiteralInShortDecl(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 || start <= assign {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			return true
		}
	}
	return false
}

func anonymousStructLiteralInVarDecl(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStart(toks, body, start)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || start <= assign || !anonymousStructLocalVarStatement(toks, stmtStart) {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		if expressionMatchesTokenRange(toks, rhs[i], start, close+1) {
			return true
		}
	}
	return false
}

func anonymousStructTypeInLocalVarDecl(toks []scan.Token, body int, pos int, structClose int) bool {
	stmtStart := anonymousStructStatementStart(toks, body, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	prefixEnd := stmtEnd
	if assign >= 0 {
		prefixEnd = assign
	}
	if stmtStart < prefixEnd && toks[stmtStart].Text == "var" {
		_, typeStart := localVarSpecNamesAndTypeForArrayContext(toks, stmtStart+1, prefixEnd)
		return anonymousStructTypeStartMatches(toks, typeStart, pos, structClose, prefixEnd)
	}
	if !insideVarGroupSpec(toks, stmtStart) {
		return false
	}
	_, typeStart := localVarSpecNamesAndTypeForArrayContext(toks, stmtStart, prefixEnd)
	return anonymousStructTypeStartMatches(toks, typeStart, pos, structClose, prefixEnd)
}

func anonymousStructTypeStartMatches(toks []scan.Token, typeStart int, pos int, structClose int, end int) bool {
	if typeStart == pos {
		return true
	}
	return typeStart == pos-2 && pos >= 2 && toks[pos-2].Text == "[" && toks[pos-1].Text == "]" && structClose < end
}

func anonymousStructLocalVarStatement(toks []scan.Token, stmtStart int) bool {
	if stmtStart >= 0 && stmtStart < len(toks) && toks[stmtStart].Text == "var" {
		return true
	}
	return insideVarGroupSpec(toks, stmtStart)
}

func anonymousStructAssignmentInLocalVarDecl(toks []scan.Token, body int, assign int) bool {
	if assign <= body || assign >= len(toks) {
		return false
	}
	stmtStart := anonymousStructStatementStart(toks, body, assign)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if assign >= stmtEnd || !anonymousStructLocalVarStatement(toks, stmtStart) {
		return false
	}
	for i := stmtStart; i < assign; i++ {
		if toks[i].Text == "struct" && i+1 < assign && toks[i+1].Text == "{" {
			close := findClose(toks, i+1, "{", "}")
			if close > i+1 && close < assign {
				return true
			}
		}
	}
	return false
}

func anonymousStructStatementStart(toks []scan.Token, body int, pos int) int {
	if pos <= body || pos >= len(toks) {
		return simpleStatementStart(toks, body, pos)
	}
	line := toks[pos].Line
	brace := 0
	for i := pos - 1; i > body; i-- {
		if brace == 0 && toks[i].Line != line {
			return i + 1
		}
		if toks[i].Text == "}" {
			brace++
			continue
		}
		if toks[i].Text == "{" && brace > 0 {
			brace--
			continue
		}
		if brace == 0 && toks[i].Text == ";" {
			return i + 1
		}
	}
	return body + 1
}

func startsAnonymousStructType(toks []scan.Token, i int) bool {
	if i < 0 || i >= len(toks) || toks[i].Text != "struct" {
		return false
	}
	return !startsNamedStructType(toks, i)
}

func startsNamedStructType(toks []scan.Token, i int) bool {
	if i <= 0 || toks[i].Text != "struct" {
		return false
	}
	namePos := i - 1
	if toks[namePos].Text == "=" && namePos > 0 {
		namePos--
	}
	if toks[namePos].Kind != scan.Ident {
		return false
	}
	before := namePos - 1
	if before < 0 {
		return false
	}
	if toks[before].Text == "type" {
		return true
	}
	if toks[before].Text == "(" {
		return before > 0 && toks[before-1].Text == "type"
	}
	if toks[before].Text == ";" {
		return typeBlockContainsOpen(toks, before)
	}
	if toks[before].Line < toks[namePos].Line {
		return typeBlockContainsOpen(toks, namePos)
	}
	return false
}

func typeBlockContainsOpen(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "(" {
			return i > 0 && toks[i-1].Text == "type"
		}
		if toks[i].Text == ")" || toks[i].Text == "func" || toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
	}
	return false
}

func startsNestedSliceType(toks []scan.Token, i int) bool {
	if i+3 >= len(toks) {
		return false
	}
	if toks[i].Text != "[" || toks[i+1].Text != "]" || toks[i+2].Text != "[" || toks[i+3].Text != "]" {
		return false
	}
	return precededByTypeContext(toks, i) || expressionTypeContext(toks, i)
}

func startsEmbeddedStructField(toks []scan.Token, i int) bool {
	if i < 0 || i >= len(toks) {
		return false
	}
	tok := toks[i]
	if tok.Text != "*" && tok.Kind != scan.Ident {
		return false
	}
	if !nameInStructFieldList(toks, i) || !startsStructFieldLine(toks, i) {
		return false
	}
	if tok.Text == "*" {
		return i+1 < len(toks) && toks[i+1].Kind == scan.Ident
	}
	if i+1 >= len(toks) {
		return false
	}
	next := toks[i+1]
	if next.Text == "." {
		return true
	}
	if next.Text == "}" || next.Text == ";" || next.Line != tok.Line || next.Kind == scan.String {
		return true
	}
	return false
}

func supportedEmbeddedStructField(toks []scan.Token, i int) bool {
	end := structFieldSpecEnd(toks, i)
	_, ok := embeddedStructFieldType(toks, i, end, nil)
	return ok
}

func structFieldSpecEnd(toks []scan.Token, start int) int {
	if start < 0 || start >= len(toks) {
		return start
	}
	line := toks[start].Line
	for i := start + 1; i < len(toks); i++ {
		if toks[i].Text == "}" || toks[i].Text == ";" || toks[i].Line != line {
			return i
		}
	}
	return len(toks)
}

func startsStructFieldLine(toks []scan.Token, i int) bool {
	if i <= 0 {
		return false
	}
	prev := toks[i-1]
	return prev.Text == "{" || prev.Text == ";" || prev.Line != toks[i].Line
}

func expressionTypeContext(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return true
	}
	prev := toks[pos-1].Text
	return prev == ":=" || prev == "=" || prev == ":" || prev == "return" || prev == "(" || prev == "," || prev == "range"
}

func closesFunctionSignature(toks []scan.Token, close int) bool {
	open := findOpen(toks, close, "(", ")")
	if open < 0 {
		return false
	}
	if open > 0 && toks[open-1].Text == "func" {
		return true
	}
	if open > 1 && toks[open-2].Text == "func" && toks[open-1].Kind == scan.Ident {
		return true
	}
	return false
}

func startsAnyInterfaceType(toks []scan.Token, i int) bool {
	return startsUnsupportedPredeclaredType(toks, i, "any")
}

func startsComplexType(toks []scan.Token, i int) bool {
	return startsUnsupportedPredeclaredType(toks, i, "complex64") || startsUnsupportedPredeclaredType(toks, i, "complex128")
}

func startsUnsupportedPredeclaredType(toks []scan.Token, i int, name string) bool {
	if toks[i].Text != name {
		return false
	}
	if i == 0 {
		return false
	}
	prev := toks[i-1]
	if prev.Text == "*" {
		return true
	}
	if prev.Text == "]" && i >= 2 && toks[i-2].Text == "[" {
		return true
	}
	if prev.Text == ")" {
		return isFunctionSignatureResult(toks, i)
	}
	if prev.Kind != scan.Ident {
		return false
	}
	if i < 2 {
		return false
	}
	beforeName := toks[i-2].Text
	return beforeName == "var" || beforeName == "type" || beforeName == "(" || beforeName == "{" || beforeName == ","
}

func isFunctionSignatureResult(toks []scan.Token, pos int) bool {
	for i := pos - 2; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "func" {
			return true
		}
		if toks[i].Text == "{" || toks[i].Text == ";" {
			return false
		}
	}
	return false
}

func startsTypeAssertion(toks []scan.Token, i int) bool {
	if i+2 >= len(toks) {
		return false
	}
	if toks[i].Text != "." {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	close := findClose(toks, i+1, "(", ")")
	return close > i+2
}

func fullSliceSecondColon(toks []scan.Token, i int) int {
	if i >= len(toks) {
		return -1
	}
	if toks[i].Text != "[" {
		return -1
	}
	close := findClose(toks, i, "[", "]")
	if close < 0 {
		return -1
	}
	colons := 0
	paren := 0
	brack := 0
	brace := 0
	for j := i + 1; j < close; j++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[j].Text == ":" {
			colons++
			if colons == 2 {
				return j
			}
			continue
		}
		updateDepth(toks[j].Text, &paren, &brack, &brace)
	}
	return -1
}

func startsUnsupportedBuiltinCall(toks []scan.Token, i int) bool {
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	if i > 0 && toks[i-1].Text == "." {
		return false
	}
	switch toks[i].Text {
	case "close", "complex", "delete", "imag", "new", "real":
		return true
	}
	return false
}

type reducibleComplexComponentCallInfo struct {
	outerClose   int
	complexPos   int
	complexClose int
	args         []expressionRange
	selectedArg  int
}

func startsReducibleComplexComponentCall(toks []scan.Token, i int) bool {
	_, ok := reducibleComplexComponentCall(toks, i)
	return ok
}

func startsPredeclaredReducibleComplexComponentCall(toks []scan.Token, i int, sigs []funcSignature) bool {
	info, ok := reducibleComplexComponentCall(toks, i)
	if ok {
		if signedFunctionCallAt(toks, i, sigs) {
			return false
		}
		if signedFunctionCallAt(toks, info.complexPos, sigs) {
			return false
		}
		return true
	}
	_, ok = reducibleComplexLiteralComponentCall(toks, i)
	return ok && !signedFunctionCallAt(toks, i, sigs)
}

func complexCallInsidePredeclaredReducibleComplexComponent(toks []scan.Token, i int, sigs []funcSignature) bool {
	if i < 2 || toks[i].Text != "complex" || toks[i-1].Text != "(" {
		return false
	}
	info, ok := reducibleComplexComponentCall(toks, i-2)
	if !ok || info.complexPos != i {
		return false
	}
	return !signedFunctionCallAt(toks, i-2, sigs) && !signedFunctionCallAt(toks, i, sigs)
}

func reducibleComplexComponentCall(toks []scan.Token, i int) (reducibleComplexComponentCallInfo, bool) {
	if i+4 >= len(toks) || toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
		return reducibleComplexComponentCallInfo{}, false
	}
	selected := -1
	if toks[i].Text == "real" {
		selected = 0
	} else if toks[i].Text == "imag" {
		selected = 1
	} else {
		return reducibleComplexComponentCallInfo{}, false
	}
	if i > 0 && toks[i-1].Text == "." {
		return reducibleComplexComponentCallInfo{}, false
	}
	outerClose := findClose(toks, i+1, "(", ")")
	if outerClose < 0 {
		return reducibleComplexComponentCallInfo{}, false
	}
	outerArgs := expressionRanges(toks, i+2, outerClose)
	if len(outerArgs) != 1 {
		return reducibleComplexComponentCallInfo{}, false
	}
	innerStart, innerEnd := trimExpressionRange(toks, outerArgs[0].start, outerArgs[0].end)
	if innerStart+3 > innerEnd || toks[innerStart].Text != "complex" || toks[innerStart+1].Text != "(" {
		return reducibleComplexComponentCallInfo{}, false
	}
	if innerStart > 0 && toks[innerStart-1].Text == "." {
		return reducibleComplexComponentCallInfo{}, false
	}
	innerClose := findClose(toks, innerStart+1, "(", ")")
	if innerClose != innerEnd-1 {
		return reducibleComplexComponentCallInfo{}, false
	}
	args := expressionRanges(toks, innerStart+2, innerClose)
	if len(args) != 2 {
		return reducibleComplexComponentCallInfo{}, false
	}
	return reducibleComplexComponentCallInfo{
		outerClose:   outerClose,
		complexPos:   innerStart,
		complexClose: innerClose,
		args:         args,
		selectedArg:  selected,
	}, true
}

type reducibleComplexLiteralComponentCallInfo struct {
	outerClose  int
	realText    string
	imagText    string
	selectedArg int
}

func reducibleComplexLiteralComponentCall(toks []scan.Token, i int) (reducibleComplexLiteralComponentCallInfo, bool) {
	if i+3 >= len(toks) || toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	selected := -1
	if toks[i].Text == "real" {
		selected = 0
	} else if toks[i].Text == "imag" {
		selected = 1
	} else {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	if i > 0 && toks[i-1].Text == "." {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	outerClose := findClose(toks, i+1, "(", ")")
	if outerClose < 0 {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	outerArgs := expressionRanges(toks, i+2, outerClose)
	if len(outerArgs) != 1 {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	realText, imagText, ok := reducibleComplexLiteralParts(toks, outerArgs[0].start, outerArgs[0].end)
	if !ok {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	return reducibleComplexLiteralComponentCallInfo{
		outerClose:  outerClose,
		realText:    realText,
		imagText:    imagText,
		selectedArg: selected,
	}, true
}

func imaginaryLiteralInPredeclaredReducibleComplexLiteralComponent(toks []scan.Token, pos int, sigs []funcSignature) bool {
	for i := 0; i < len(toks); i++ {
		info, ok := reducibleComplexLiteralComponentCall(toks, i)
		if !ok {
			continue
		}
		if signedFunctionCallAt(toks, i, sigs) {
			continue
		}
		if pos > i && pos < info.outerClose {
			return true
		}
	}
	return false
}

func discardedPureMapCompositeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedMapCompositeStatementContainingToken(toks, pos, discardablePureMapCompositeExpression)
}

func discardedLowerableMapCompositeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedMapCompositeStatementContainingToken(toks, pos, discardableLowerableMapCompositeExpression)
}

func discardedLowerableMapSliceCompositeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedMapCompositeStatementContainingToken(toks, pos, discardableLowerableMapSliceCompositeExpression)
}

func discardedMapCompositeStatementContainingToken(toks []scan.Token, pos int, supported func([]scan.Token, int, int) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for stmtStart := 0; stmtStart < len(toks); stmtStart++ {
		if toks[stmtStart].Text != "_" {
			continue
		}
		stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
		if pos < stmtStart || pos >= stmtEnd {
			continue
		}
		assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
		if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
			continue
		}
		lhs := expressionRanges(toks, stmtStart, assign)
		rhs := expressionRanges(toks, assign+1, stmtEnd)
		if len(lhs) == 0 || len(lhs) != len(rhs) {
			continue
		}
		matched := false
		ok := true
		for i := 0; i < len(lhs); i++ {
			if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
				ok = false
				break
			}
			if pos >= rhs[i].start && pos < rhs[i].end {
				matched = true
			}
			if !supported(toks, rhs[i].start, rhs[i].end) {
				ok = false
				break
			}
		}
		if ok && matched {
			return true
		}
	}
	return false
}

func discardedPureMapMakeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedMapMakeStatementContainingToken(toks, pos, discardablePureMapMakeExpression)
}

func discardedLowerableMapMakeStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedMapMakeStatementContainingToken(toks, pos, discardableLowerableMapMakeExpression)
}

func discardedMapMakeStatementContainingToken(toks []scan.Token, pos int, accept func([]scan.Token, int, int) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for stmtStart := 0; stmtStart < len(toks); stmtStart++ {
		if toks[stmtStart].Text != "_" {
			continue
		}
		stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
		if pos < stmtStart || pos >= stmtEnd {
			continue
		}
		assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
		if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
			continue
		}
		lhs := expressionRanges(toks, stmtStart, assign)
		rhs := expressionRanges(toks, assign+1, stmtEnd)
		if len(lhs) == 0 || len(lhs) != len(rhs) {
			continue
		}
		matched := false
		ok := true
		for i := 0; i < len(lhs); i++ {
			if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
				ok = false
				break
			}
			if pos >= rhs[i].start && pos < rhs[i].end {
				matched = true
			}
			if !accept(toks, rhs[i].start, rhs[i].end) {
				ok = false
				break
			}
		}
		if ok && matched {
			return true
		}
	}
	return false
}

type namedMapTypeInfo struct {
	qualifier string
	name      string
	keyType   string
	valueType string
}

var activeImportedNamedMapTypes []namedMapTypeInfo

func namedMapTypes(toks []scan.Token) []namedMapTypeInfo {
	out := localNamedMapTypes(toks)
	for i := 0; i < len(activeImportedNamedMapTypes); i++ {
		out = setNamedMapTypeInfo(out, activeImportedNamedMapTypes[i])
	}
	return out
}

func localNamedMapTypes(toks []scan.Token) []namedMapTypeInfo {
	var out []namedMapTypeInfo
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localTypeSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				out = appendNamedMapTypeInfo(out, toks, ranges[rangeIndex].start, ranges[rangeIndex].end)
			}
			i = close
			continue
		}
		if i+1 >= len(toks) || toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := functionTypeSpecEndFromName(toks, i+1, len(toks))
		out = appendNamedMapTypeInfo(out, toks, i+1, specEnd)
		i = specEnd - 1
	}
	return out
}

func importedNamedMapTypesForFile(file parse.File, packages []load.Package) []namedMapTypeInfo {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var out []namedMapTypeInfo
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		pkgIndex := packageIndexByImportPath(packages, imp.Path)
		if pkgIndex < 0 {
			continue
		}
		files := packages[pkgIndex].Files
		typeNames := packageNamedTypeUnderlyings(files)
		structs := packageStructTypesWithTypes(files, typeNames)
		pkgMaps := packageNamedMapTypes(files)
		for mapIndex := 0; mapIndex < len(pkgMaps); mapIndex++ {
			info := pkgMaps[mapIndex]
			if !isExported(info.name) {
				continue
			}
			info.qualifier = importedTypeQualifier(localName)
			info.keyType = qualifyImportedDefinedType(info.qualifier, typeNames, info.keyType)
			info.keyType = qualifyImportedStructFieldType(info.qualifier, structs, info.keyType)
			info.valueType = qualifyImportedDefinedType(info.qualifier, typeNames, info.valueType)
			info.valueType = qualifyImportedStructFieldType(info.qualifier, structs, info.valueType)
			out = setNamedMapTypeInfo(out, info)
		}
	}
	return out
}

func packageNamedMapTypes(files []load.File) []namedMapTypeInfo {
	var out []namedMapTypeInfo
	for i := 0; i < len(files); i++ {
		file, err := parsedLoadFile(files[i])
		if err != nil {
			continue
		}
		values := localNamedMapTypes(file.Tokens)
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			out = setNamedMapTypeInfo(out, values[valueIndex])
		}
	}
	return out
}

func appendNamedMapTypeInfo(out []namedMapTypeInfo, toks []scan.Token, start int, end int) []namedMapTypeInfo {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return out
	}
	name := toks[start].Text
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	keyType, valueType := mapTypeKeyValueText(toks, typeStart, end)
	if name == "" || name == "_" || keyType == "" || valueType == "" {
		return out
	}
	return setNamedMapTypeInfo(out, namedMapTypeInfo{name: name, keyType: keyType, valueType: valueType})
}

func setNamedMapTypeInfo(out []namedMapTypeInfo, info namedMapTypeInfo) []namedMapTypeInfo {
	for i := 0; i < len(out); i++ {
		if out[i].qualifier == info.qualifier && out[i].name == info.name {
			out[i].keyType = info.keyType
			out[i].valueType = info.valueType
			return out
		}
	}
	return append(out, info)
}

func namedMapTypeByName(namedMaps []namedMapTypeInfo, name string) (namedMapTypeInfo, bool) {
	for i := 0; i < len(namedMaps); i++ {
		if namedMaps[i].qualifier == "" && namedMaps[i].name == name {
			return namedMaps[i], true
		}
	}
	return namedMapTypeInfo{}, false
}

func namedMapTypeBySelector(namedMaps []namedMapTypeInfo, qualifier string, name string) (namedMapTypeInfo, bool) {
	for i := 0; i < len(namedMaps); i++ {
		if namedMaps[i].qualifier == qualifier && namedMaps[i].name == name {
			return namedMaps[i], true
		}
	}
	return namedMapTypeInfo{}, false
}

func namedMapCompositeLiteralOpen(toks []scan.Token, start int, end int, namedMaps []namedMapTypeInfo) int {
	if start < 0 || start+1 >= end || toks[start].Kind != scan.Ident {
		return -1
	}
	if toks[start+1].Text == "{" {
		if _, ok := namedMapTypeByName(namedMaps, toks[start].Text); !ok {
			return -1
		}
		return start + 1
	}
	if start+3 >= end || toks[start+1].Text != "." || toks[start+2].Kind != scan.Ident || toks[start+3].Text != "{" {
		return -1
	}
	if _, ok := namedMapTypeBySelector(namedMaps, toks[start].Text, toks[start+2].Text); !ok {
		return -1
	}
	return start + 3
}

func identifierInsideLowerableNamedMapUse(toks []scan.Token, pos int, name string, namedMaps []namedMapTypeInfo) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos].Text != name {
		return false
	}
	if pureMapAliasStatementContainingToken(toks, pos) {
		return true
	}
	if namedMapCompositeLiteralOpen(toks, pos, len(toks), namedMaps) < 0 {
		return false
	}
	if discardedLowerableMapCompositeStatementContainingToken(toks, pos) {
		return true
	}
	if lowerableMapLiteralDeleteStatementContainingToken(toks, pos) {
		return true
	}
	if lowerableMapLiteralLenCallContainingToken(toks, pos) {
		return true
	}
	if lowerableMapLiteralIndexExpressionContainingToken(toks, pos) {
		return true
	}
	if lowerableMapRangeStatementContainingToken(toks, pos) {
		return true
	}
	if pureMapAliasStatementContainingToken(toks, pos) {
		return true
	}
	return false
}

func discardablePureMapMakeExpression(toks []scan.Token, start int, end int) bool {
	return discardableMapMakeExpression(toks, start, end, false)
}

func discardableLowerableMapMakeExpression(toks []scan.Token, start int, end int) bool {
	return discardableMapMakeExpression(toks, start, end, true)
}

func discardableMapMakeExpression(toks []scan.Token, start int, end int, allowDirectCallCapacity bool) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableMapMakeExpression(toks, start+1, close, allowDirectCallCapacity)
		}
	}
	if start+3 > end || toks[start].Text != "make" || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	args := expressionRanges(toks, start+2, close)
	if len(args) != 1 && len(args) != 2 {
		return false
	}
	if !discardableMapType(toks, args[0].start, args[0].end) {
		return false
	}
	if len(args) == 2 {
		if _, ok := simpleIntegerLiteralKey(toks, args[1].start, args[1].end); ok {
			return true
		}
		return allowDirectCallCapacity && directCallExpressionWithoutCallArgs(toks, args[1].start, args[1].end)
	}
	return true
}

func discardableMapType(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start+4 > end || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return false
	}
	keyClose := findClose(toks, start+1, "[", "]")
	return keyClose > start+1 && keyClose+1 < end
}

func pureMapMakeLenCallContainingToken(toks []scan.Token, pos int) bool {
	return mapMakeLenCallContainingToken(toks, pos, discardablePureMapMakeExpression)
}

func lowerableMapMakeLenCallContainingToken(toks []scan.Token, pos int) bool {
	return mapMakeLenCallContainingToken(toks, pos, discardableLowerableMapMakeExpression)
}

func mapMakeLenCallContainingToken(toks []scan.Token, pos int, accept func([]scan.Token, int, int) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "len" || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && toks[i-1].Text == "." {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			continue
		}
		args := expressionRanges(toks, i+2, close)
		if len(args) != 1 {
			continue
		}
		if pos < i || pos > close {
			continue
		}
		if accept(toks, args[0].start, args[0].end) {
			return true
		}
	}
	return false
}

func pureMapRangeStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "range" {
			continue
		}
		start := i + 1
		end := rangeOperandEnd(toks, start, len(toks))
		if pos < start || pos >= end {
			continue
		}
		if pureMapRangeExpressionSupported(toks, start, end) {
			return true
		}
	}
	return false
}

func lowerableMapRangeStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	if pureMapRangeStatementContainingToken(toks, pos) {
		return true
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "range" {
			continue
		}
		start := i + 1
		end := rangeOperandEnd(toks, start, len(toks))
		if pos < start || pos >= end {
			continue
		}
		if lowerableMapRangeExpressionSupported(toks, start, end) {
			return true
		}
	}
	return false
}

func pureMapRangeExpressionSupported(toks []scan.Token, start int, end int) bool {
	_, _, ok := pureMapRangeExpressionTypes(toks, start, end)
	return ok
}

func lowerableMapRangeExpressionSupported(toks []scan.Token, start int, end int) bool {
	_, _, ok := lowerableMapRangeExpressionTypes(toks, start, end)
	return ok
}

func pureMapRangeExpressionTypes(toks []scan.Token, start int, end int) (string, string, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapRangeExpressionTypes(toks, start+1, close)
		}
	}
	if discardablePureMapMakeExpression(toks, start, end) {
		keyType, valueType := mapExpressionKeyValueTypeText(toks, start, end)
		return keyType, valueType, mapRangeKeyTypeSupported(keyType) && mapRangeValueTypeSupported(valueType)
	}
	if !discardablePureMapCompositeExpression(toks, start, end) {
		return "", "", false
	}
	keyType, valueType := mapExpressionKeyValueTypeText(toks, start, end)
	return keyType, valueType, mapRangeKeyTypeSupported(keyType) && mapRangeValueTypeSupported(valueType)
}

func lowerableMapRangeExpressionTypes(toks []scan.Token, start int, end int) (string, string, bool) {
	if keyType, valueType, ok := pureMapRangeExpressionTypes(toks, start, end); ok {
		return keyType, valueType, true
	}
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableMapRangeExpressionTypes(toks, start+1, close)
		}
	}
	if discardableLowerableMapMakeExpression(toks, start, end) {
		keyType, valueType := mapExpressionKeyValueTypeText(toks, start, end)
		return keyType, valueType, mapRangeKeyTypeSupported(keyType) && mapRangeValueTypeSupported(valueType)
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks))
	}
	if open < 0 {
		return "", "", false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return "", "", false
	}
	if !lowerableMapRangeCompositeElements(toks, open+1, close) {
		return "", "", false
	}
	keyType, valueType := mapExpressionKeyValueTypeText(toks, start, end)
	return keyType, valueType, mapRangeKeyTypeSupported(keyType) && mapRangeValueTypeSupported(valueType)
}

func lowerableMapRangeCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		if !discardablePureMapCompositeKey(toks, value.start, colon) {
			return false
		}
		if !lowerableMapRangeCompositeValue(toks, colon+1, value.end) {
			return false
		}
	}
	return true
}

func lowerableMapRangeCompositeValue(toks []scan.Token, start int, end int) bool {
	if discardableArrayLiteralElement(toks, start, end) {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" {
		return true
	}
	return directCallExpressionWithoutCallArgs(toks, start, end)
}

func pureMapAliasStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		alias, ok := pureMapAliasStatementAt(toks, i, len(toks))
		if !ok {
			continue
		}
		if pos >= alias.stmtStart && pos < alias.stmtEnd {
			return true
		}
		i = alias.stmtEnd - 1
	}
	return false
}

type pureMapAliasInfo struct {
	name      string
	stmtStart int
	stmtEnd   int
	exprStart int
	exprEnd   int
	keyType   string
	valueType string
}

func pureMapAliasStatementAt(toks []scan.Token, pos int, limit int) (pureMapAliasInfo, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) {
		return pureMapAliasInfo{}, false
	}
	stmtStart := pos
	stmtEnd := simpleStatementEnd(toks, stmtStart, limit)
	if toks[pos].Text == "var" {
		return pureMapAliasVarStatement(toks, stmtStart, stmtEnd)
	}
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		return pureMapAliasInfo{}, false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return pureMapAliasInfo{}, false
	}
	name := singleIdentifierExpression(toks, lhs[0].start, lhs[0].end)
	if name == "" {
		return pureMapAliasInfo{}, false
	}
	keyType, valueType, ok := lowerableMapRangeExpressionTypes(toks, rhs[0].start, rhs[0].end)
	if !ok {
		return pureMapAliasInfo{}, false
	}
	return pureMapAliasInfo{name: name, stmtStart: stmtStart, stmtEnd: stmtEnd, exprStart: rhs[0].start, exprEnd: rhs[0].end, keyType: keyType, valueType: valueType}, true
}

func pureMapAliasVarStatement(toks []scan.Token, stmtStart int, stmtEnd int) (pureMapAliasInfo, bool) {
	if stmtStart+1 >= stmtEnd || toks[stmtStart].Text != "var" {
		return pureMapAliasInfo{}, false
	}
	eq := findTopLevelToken(toks, stmtStart+1, stmtEnd, "=")
	if eq < 0 {
		return pureMapAliasInfo{}, false
	}
	lhs := expressionRanges(toks, stmtStart+1, eq)
	rhs := expressionRanges(toks, eq+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return pureMapAliasInfo{}, false
	}
	nameStart, nameEnd := trimExpressionRange(toks, lhs[0].start, lhs[0].end)
	if nameStart >= nameEnd || toks[nameStart].Kind != scan.Ident || toks[nameStart].Text == "_" {
		return pureMapAliasInfo{}, false
	}
	name := toks[nameStart].Text
	keyType, valueType, ok := lowerableMapRangeExpressionTypes(toks, rhs[0].start, rhs[0].end)
	if !ok {
		return pureMapAliasInfo{}, false
	}
	return pureMapAliasInfo{name: name, stmtStart: stmtStart, stmtEnd: stmtEnd, exprStart: rhs[0].start, exprEnd: rhs[0].end, keyType: keyType, valueType: valueType}, true
}

func pureMapAliasForIdentifierAt(toks []scan.Token, name string, pos int) (pureMapAliasInfo, bool) {
	if name == "" {
		return pureMapAliasInfo{}, false
	}
	var found pureMapAliasInfo
	ok := false
	for i := 0; i < len(toks) && i < pos; i++ {
		alias, aliasOK := pureMapAliasStatementAt(toks, i, len(toks))
		if aliasOK {
			if alias.name == name && alias.stmtEnd <= pos {
				found = alias
				ok = true
			}
			i = alias.stmtEnd - 1
		}
	}
	return found, ok
}

func pureMapAliasExpressionSupported(toks []scan.Token, start int, end int) bool {
	_, _, ok := pureMapAliasOrDirectRangeExpressionTypes(toks, start, end)
	return ok
}

func pureMapAliasOrDirectRangeExpressionTypes(toks []scan.Token, start int, end int) (string, string, bool) {
	if keyType, valueType, ok := pureMapRangeExpressionTypes(toks, start, end); ok {
		return keyType, valueType, true
	}
	start, end = trimExpressionRange(toks, start, end)
	if start+1 != end || toks[start].Kind != scan.Ident {
		return "", "", false
	}
	alias, ok := pureMapAliasForIdentifierAt(toks, toks[start].Text, start)
	if !ok {
		return "", "", false
	}
	return alias.keyType, alias.valueType, true
}

func pureMapAliasIndexExpressionRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos+3 > len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "[" {
		return 0, 0, false
	}
	alias, ok := pureMapAliasForIdentifierAt(toks, toks[pos].Text, pos)
	if !ok || !mapRangeValueTypeSupported(alias.valueType) {
		return 0, 0, false
	}
	close := findClose(toks, pos+1, "[", "]")
	if close < 0 {
		return 0, 0, false
	}
	if _, ok := mapLiteralComparableKey(toks, pos+2, close); !ok {
		return 0, 0, false
	}
	return pos, close + 1, true
}

func pureMapAliasIndexValueType(toks []scan.Token, start int, end int) string {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return ""
	}
	exprStart, exprEnd, ok := pureMapAliasIndexExpressionRange(toks, start)
	if !ok || exprStart != start || exprEnd != end {
		return ""
	}
	alias, ok := pureMapAliasForIdentifierAt(toks, toks[start].Text, start)
	if !ok {
		return ""
	}
	return alias.valueType
}

func lowerableMapLiteralIndexValueType(toks []scan.Token, start int, end int) string {
	if typ := pureMapAliasIndexValueType(toks, start, end); typ != "" {
		return typ
	}
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return ""
	}
	exprStart, exprEnd, ok := lowerableMapLiteralIndexExpressionRange(toks, start)
	if !ok || exprStart != start || exprEnd != end {
		return ""
	}
	mapStart, mapEnd, ok := mapLiteralIndexMapExpressionRange(toks, start)
	if !ok {
		return ""
	}
	return mapElementTypeText(toks, mapStart, mapEnd)
}

func mapLiteralIndexMapExpressionRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos >= len(toks) {
		return 0, 0, false
	}
	if toks[pos].Text == "(" {
		closeParen := findClose(toks, pos, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || toks[closeParen+1].Text != "[" {
			return 0, 0, false
		}
		return pos + 1, closeParen, true
	}
	if toks[pos].Text == "make" && pos+1 < len(toks) && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
			return 0, 0, false
		}
		return pos, close + 1, true
	}
	open := pureMapCompositeLiteralOpen(toks, pos, len(toks))
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, pos, len(toks), namedMapTypes(toks))
	}
	if open < 0 {
		return 0, 0, false
	}
	close := findClose(toks, open, "{", "}")
	if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
		return 0, 0, false
	}
	return pos, close + 1, true
}

func pureMapAliasUnsupportedUseAt(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	if _, ok := pureMapAliasForIdentifierAt(toks, toks[pos].Text, pos); !ok {
		return false
	}
	if pureMapAliasStatementContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasLenCallContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasIndexExpressionContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasRangeStatementContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasDeleteStatementContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasElementAssignmentStatementContainingToken(toks, pos) {
		return false
	}
	if pureMapAliasBlankDiscardStatementContainingToken(toks, pos) {
		return false
	}
	return true
}

func pureMapAliasLenCallContainingToken(toks []scan.Token, pos int) bool {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "len" || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && toks[i-1].Text == "." {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			continue
		}
		args := expressionRanges(toks, i+2, close)
		if len(args) != 1 || pos < args[0].start || pos >= args[0].end {
			continue
		}
		if pureMapAliasExpressionSupported(toks, args[0].start, args[0].end) {
			return true
		}
	}
	return false
}

func pureMapAliasIndexExpressionContainingToken(toks []scan.Token, pos int) bool {
	for i := 0; i < len(toks); i++ {
		start, end, ok := pureMapAliasIndexExpressionRange(toks, i)
		if ok && pos >= start && pos < end {
			return true
		}
	}
	return false
}

func pureMapAliasRangeStatementContainingToken(toks []scan.Token, pos int) bool {
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "range" {
			continue
		}
		start := i + 1
		end := rangeOperandEnd(toks, start, len(toks))
		if pos < start || pos >= end {
			continue
		}
		if pureMapAliasExpressionSupported(toks, start, end) {
			return true
		}
	}
	return false
}

func pureMapAliasDeleteStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "delete" {
			continue
		}
		stmtStart := simpleStatementStart(toks, -1, i)
		if stmtStart != i {
			continue
		}
		stmtEnd := simpleStatementEnd(toks, i, len(toks))
		close := findClose(toks, i+1, "(", ")")
		if close < 0 || close != stmtEnd-1 {
			continue
		}
		if pos < i || pos > close {
			continue
		}
		if pureMapAliasDeleteCall(toks, i, close) {
			return true
		}
	}
	return false
}

func pureMapAliasDeleteCall(toks []scan.Token, pos int, close int) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "delete" || toks[pos+1].Text != "(" {
		return false
	}
	args := expressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return false
	}
	nameStart, nameEnd := trimExpressionRange(toks, args[0].start, args[0].end)
	if nameStart+1 != nameEnd || toks[nameStart].Kind != scan.Ident {
		return false
	}
	if _, ok := pureMapAliasForIdentifierAt(toks, toks[nameStart].Text, pos); !ok {
		return false
	}
	_, ok := mapLiteralComparableKey(toks, args[1].start, args[1].end)
	return ok
}

func pureMapAliasElementAssignmentAt(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	if isIncDecAt(toks, pos, len(toks)) {
		stmtStart := simpleStatementStart(toks, -1, pos)
		if stmtStart < pos && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch") {
			stmtStart++
		}
		lhs := expressionRanges(toks, stmtStart, pos)
		for i := 0; i < len(lhs); i++ {
			start, end, ok := pureMapAliasIndexExpressionRange(toks, lhs[i].start)
			if ok && start == lhs[i].start && end == lhs[i].end {
				return true
			}
		}
		return false
	}
	text := toks[pos].Text
	lhsEnd := pos
	if isCompoundAssignmentAt(toks, pos, len(toks)) {
		// The scanner stores += and friends as adjacent operator tokens.
	} else if text == "=" {
		if isCompoundAssignmentEquals(toks, pos) {
			return false
		}
	} else if text != "+=" && text != "-=" && text != "*=" && text != "/=" && text != "%=" {
		return false
	}
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart < pos && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch") {
		stmtStart++
	}
	lhs := expressionRanges(toks, stmtStart, lhsEnd)
	for i := 0; i < len(lhs); i++ {
		start, end, ok := pureMapAliasIndexExpressionRange(toks, lhs[i].start)
		if ok && start == lhs[i].start && end == lhs[i].end {
			return true
		}
	}
	return false
}

func pureMapAliasElementAssignmentStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		stmtStart, stmtEnd, ok := pureMapAliasElementAssignmentSupportedRange(toks, i)
		if ok && pos >= stmtStart && pos < stmtEnd {
			return true
		}
	}
	return false
}

func pureMapAliasBlankDiscardStatementContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		stmtStart, stmtEnd, ok := pureMapAliasBlankDiscardStatementRange(toks, i)
		if ok && pos >= stmtStart && pos < stmtEnd {
			return true
		}
	}
	return false
}

func pureMapAliasBlankDiscardStatementRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "=" || isCompoundAssignmentEquals(toks, pos) {
		return 0, 0, false
	}
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart < pos && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch") {
		return 0, 0, false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	lhs := expressionRanges(toks, stmtStart, pos)
	rhs := expressionRanges(toks, pos+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return 0, 0, false
	}
	if !blankIdentifierExpression(toks, lhs[0].start, lhs[0].end) {
		return 0, 0, false
	}
	aliasName := pureMapAliasIdentifierExpression(toks, rhs[0].start, rhs[0].end)
	if aliasName == "" {
		return 0, 0, false
	}
	if _, ok := pureMapAliasForIdentifierAt(toks, aliasName, pos); !ok {
		return 0, 0, false
	}
	return stmtStart, stmtEnd, true
}

func pureMapAliasIdentifierExpression(toks []scan.Token, start int, end int) string {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasIdentifierExpression(toks, start+1, close)
		}
	}
	return singleIdentifierExpression(toks, start, end)
}

func pureMapAliasElementAssignmentSupportedAt(toks []scan.Token, pos int) bool {
	_, _, ok := pureMapAliasElementAssignmentSupportedRange(toks, pos)
	return ok
}

func pureMapAliasElementAssignmentSupportedRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos >= len(toks) {
		return 0, 0, false
	}
	assign := pos
	lhsEnd := pos
	rhsStart := pos + 1
	compoundOp := ""
	incDecOp := ""
	if isCompoundAssignmentAt(toks, pos, len(toks)) {
		assign = pos + 1
		lhsEnd = pos
		rhsStart = assign + 1
		compoundOp = toks[pos].Text
	} else if isIncDecAt(toks, pos, len(toks)) {
		assign = pos
		lhsEnd = pos
		rhsStart = pos + 2
		incDecOp = toks[pos].Text
	} else if toks[pos].Text == "=" && !isCompoundAssignmentEquals(toks, pos) {
		assign = pos
		lhsEnd = pos
		rhsStart = pos + 1
	} else {
		return 0, 0, false
	}
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart < pos && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch") {
		return 0, 0, false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if assign >= stmtEnd || rhsStart > stmtEnd {
		return 0, 0, false
	}
	lhs := expressionRanges(toks, stmtStart, lhsEnd)
	if incDecOp != "" {
		if rhsStart != stmtEnd || len(lhs) != 1 {
			return 0, 0, false
		}
		start, end, ok := pureMapAliasIndexExpressionRange(toks, lhs[0].start)
		if !ok || start != lhs[0].start || end != lhs[0].end {
			return 0, 0, false
		}
		alias, ok := pureMapAliasForIdentifierAt(toks, toks[start].Text, start)
		if !ok || !pureMapAliasIncDecSupported(alias.valueType) {
			return 0, 0, false
		}
		return stmtStart, stmtEnd, true
	}
	rhs := expressionRanges(toks, rhsStart, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return 0, 0, false
	}
	start, end, ok := pureMapAliasIndexExpressionRange(toks, lhs[0].start)
	if !ok || start != lhs[0].start || end != lhs[0].end {
		return 0, 0, false
	}
	alias, ok := pureMapAliasForIdentifierAt(toks, toks[start].Text, start)
	if !ok {
		return 0, 0, false
	}
	if compoundOp != "" && !pureMapAliasCompoundAssignmentOperatorSupported(compoundOp, alias.valueType) {
		return 0, 0, false
	}
	if !pureMapAliasAssignmentValueSupported(toks, rhs[0].start, rhs[0].end, alias.valueType) {
		return 0, 0, false
	}
	return stmtStart, stmtEnd, true
}

func pureMapAliasIncDecSupported(valueType string) bool {
	switch valueType {
	case "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return false
}

func pureMapAliasCompoundAssignmentOperatorSupported(op string, valueType string) bool {
	switch valueType {
	case "int", "int16", "int32", "int64", "byte":
		return op == "+" || op == "-" || op == "*" || op == "/" || op == "%"
	case "float64":
		return op == "+" || op == "-" || op == "*" || op == "/"
	case "string":
		return op == "+"
	}
	return false
}

func pureMapAliasAssignmentValueSupported(toks []scan.Token, start int, end int, valueType string) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasAssignmentValueSupported(toks, start+1, close, valueType)
		}
	}
	if valueType == "string" {
		if start+1 == end && toks[start].Kind == scan.String {
			return true
		}
	}
	if valueType == "bool" {
		if start+1 == end && (toks[start].Text == "true" || toks[start].Text == "false") {
			return true
		}
	}
	if strings.HasPrefix(valueType, "*") || strings.HasPrefix(valueType, "[]") {
		if start+1 == end && toks[start].Text == "nil" {
			return true
		}
	}
	if valueType == "byte" {
		if start+1 == end && toks[start].Kind == scan.Char {
			return true
		}
		if signedNumberLiteral(toks, start, end) {
			return true
		}
	}
	if valueType == "int" || valueType == "int16" || valueType == "int32" || valueType == "int64" || valueType == "float64" {
		if signedNumberLiteral(toks, start, end) {
			return true
		}
	}
	if valueType == "int" && pureMapAliasAssignmentIntBinaryValueSupported(toks, start, end) {
		return true
	}
	if valueType == "string" && pureMapAliasAssignmentStringBinaryValueSupported(toks, start, end) {
		return true
	}
	if valueType == "bool" && pureMapAliasAssignmentBoolBinaryValueSupported(toks, start, end) {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" {
		return true
	}
	return directCallExpressionWithoutCallArgs(toks, start, end)
}

func pureMapAliasAssignmentIntBinaryValueSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasAssignmentIntBinaryValueSupported(toks, start+1, close)
		}
	}
	if signedNumberLiteral(toks, start, end) || singleIdentifierExpression(toks, start, end) != "" || directCallExpressionWithoutCallArgs(toks, start, end) {
		return true
	}
	op := topLevelBinaryOperator(toks, start, end)
	if op < 0 || !pureMapAliasAssignmentIntBinaryOperator(toks[op].Text) {
		return false
	}
	return pureMapAliasAssignmentIntBinaryValueSupported(toks, start, op) && pureMapAliasAssignmentIntBinaryValueSupported(toks, op+1, end)
}

func pureMapAliasAssignmentIntBinaryOperator(op string) bool {
	switch op {
	case "+", "-", "*", "/", "%", "&", "|", "^", "&^", "<<", ">>":
		return true
	}
	return false
}

func pureMapAliasAssignmentStringBinaryValueSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasAssignmentStringBinaryValueSupported(toks, start+1, close)
		}
	}
	if start+1 == end && toks[start].Kind == scan.String {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" || directCallExpressionWithoutCallArgs(toks, start, end) {
		return true
	}
	op := topLevelBinaryOperator(toks, start, end)
	if op < 0 || toks[op].Text != "+" {
		return false
	}
	return pureMapAliasAssignmentStringBinaryValueSupported(toks, start, op) && pureMapAliasAssignmentStringBinaryValueSupported(toks, op+1, end)
}

func pureMapAliasAssignmentBoolBinaryValueSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasAssignmentBoolBinaryValueSupported(toks, start+1, close)
		}
	}
	if start+1 == end && (toks[start].Text == "true" || toks[start].Text == "false") {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" || directCallExpressionWithoutCallArgs(toks, start, end) {
		return true
	}
	op := topLevelBinaryOperator(toks, start, end)
	if op < 0 {
		return false
	}
	switch toks[op].Text {
	case "==", "!=", "<", "<=", ">", ">=":
		return pureMapAliasAssignmentComparableValueSupported(toks, start, op) && pureMapAliasAssignmentComparableValueSupported(toks, op+1, end)
	}
	return false
}

func pureMapAliasAssignmentComparableValueSupported(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapAliasAssignmentComparableValueSupported(toks, start+1, close)
		}
	}
	if singleIdentifierExpression(toks, start, end) != "" || directCallExpressionWithoutCallArgs(toks, start, end) {
		return true
	}
	if start+1 == end {
		return toks[start].Kind == scan.Number || toks[start].Kind == scan.String || toks[start].Kind == scan.Char || toks[start].Text == "true" || toks[start].Text == "false"
	}
	return signedNumberLiteral(toks, start, end)
}

func signedNumberLiteral(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start < end && (toks[start].Text == "-" || toks[start].Text == "+") {
		start++
	}
	return start+1 == end && toks[start].Kind == scan.Number
}

func pureMapLiteralDeleteStatementContainingToken(toks []scan.Token, pos int) bool {
	return mapLiteralDeleteStatementContainingToken(toks, pos, pureMapLiteralDeleteCall)
}

func lowerableMapLiteralDeleteStatementContainingToken(toks []scan.Token, pos int) bool {
	return mapLiteralDeleteStatementContainingToken(toks, pos, lowerableMapLiteralDeleteCall)
}

func mapLiteralDeleteStatementContainingToken(toks []scan.Token, pos int, supported func([]scan.Token, int, int) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "delete" {
			continue
		}
		stmtStart := simpleStatementStart(toks, -1, i)
		if stmtStart != i {
			continue
		}
		stmtEnd := simpleStatementEnd(toks, i, len(toks))
		close := findClose(toks, i+1, "(", ")")
		if close < 0 || close != stmtEnd-1 {
			continue
		}
		if pos < i || pos > close {
			continue
		}
		if supported(toks, i, close) {
			return true
		}
	}
	return false
}

func pureMapLiteralDeleteCall(toks []scan.Token, pos int, close int) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "delete" || toks[pos+1].Text != "(" {
		return false
	}
	args := expressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return false
	}
	return discardablePureMapDeleteTargetExpression(toks, args[0].start, args[0].end) && discardableArrayLiteralElement(toks, args[1].start, args[1].end)
}

func lowerableMapLiteralDeleteCall(toks []scan.Token, pos int, close int) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "delete" || toks[pos+1].Text != "(" {
		return false
	}
	args := expressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return false
	}
	return discardableLowerableMapDeleteTargetExpression(toks, args[0].start, args[0].end) && discardableArrayLiteralElement(toks, args[1].start, args[1].end)
}

func discardablePureMapDeleteTargetExpression(toks []scan.Token, start int, end int) bool {
	return discardablePureMapCompositeExpression(toks, start, end) || discardablePureMapMakeExpression(toks, start, end)
}

func discardableLowerableMapDeleteTargetExpression(toks []scan.Token, start int, end int) bool {
	return discardableLowerableMapCompositeExpression(toks, start, end) || discardableLowerableMapMakeExpression(toks, start, end)
}

func discardablePureMapCompositeExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardablePureMapCompositeExpression(toks, start+1, close)
		}
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks))
	}
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardablePureMapCompositeElements(toks, open+1, close)
}

func discardableLowerableMapCompositeExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableMapCompositeExpression(toks, start+1, close)
		}
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks))
	}
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardableLowerableMapCompositeElements(toks, open+1, close)
}

func discardableLowerableMapSliceCompositeExpression(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableMapSliceCompositeExpression(toks, start+1, close)
		}
	}
	open := mapSliceCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardableLowerableMapSliceCompositeElements(toks, open+1, close)
}

func mapSliceCompositeLiteralOpen(toks []scan.Token, start int, end int) int {
	if start < 0 || start+4 >= end || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose < 0 || brackClose+2 >= end {
		return -1
	}
	if brackClose > start+1 && !arrayLengthSupported(toks, start+1, brackClose) {
		return -1
	}
	if toks[brackClose+1].Text != "map" || toks[brackClose+2].Text != "[" {
		return -1
	}
	keyClose := findClose(toks, brackClose+2, "[", "]")
	if keyClose < 0 || keyClose+1 >= end {
		return -1
	}
	paren := 0
	brack := 0
	for i := keyClose + 1; i < end; i++ {
		if paren == 0 && brack == 0 {
			switch toks[i].Text {
			case "{":
				return i
			case ";", ")", ",", "=", ":=", "==", "!=", "<", "<=", ">", ">=":
				return -1
			}
		}
		switch toks[i].Text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			brack++
		case "]":
			brack--
		}
	}
	return -1
}

func pureMapCompositeLiteralOpen(toks []scan.Token, start int, end int) int {
	if start < 0 || start+3 >= end || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return -1
	}
	keyClose := findClose(toks, start+1, "[", "]")
	if keyClose < 0 || keyClose+1 >= end {
		return -1
	}
	paren := 0
	brack := 0
	for i := keyClose + 1; i < end; i++ {
		if paren == 0 && brack == 0 {
			switch toks[i].Text {
			case "{":
				return i
			case ";", ")", ",", "=", ":=", "==", "!=", "<", "<=", ">", ">=":
				return -1
			}
		}
		switch toks[i].Text {
		case "(":
			paren++
		case ")":
			paren--
		case "[":
			brack++
		case "]":
			brack--
		}
	}
	return -1
}

func discardablePureMapCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		if !discardablePureMapCompositeKey(toks, value.start, colon) {
			return false
		}
		if !discardablePureMapCompositeValue(toks, colon+1, value.end) {
			return false
		}
	}
	return true
}

func discardableLowerableMapCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		if !discardablePureMapCompositeKey(toks, value.start, colon) {
			return false
		}
		if !discardableLowerableMapCompositeValue(toks, colon+1, value.end) {
			return false
		}
	}
	return true
}

func discardableLowerableMapSliceCompositeElements(toks []scan.Token, start int, end int) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardablePureCompositeKey(toks, value.start, colon) {
				return false
			}
			value.start = colon + 1
		}
		if !discardableLowerableMapSliceCompositeValue(toks, value.start, value.end) {
			return false
		}
	}
	return true
}

func discardablePureMapCompositeKey(toks []scan.Token, start int, end int) bool {
	return discardableArrayLiteralElement(toks, start, end)
}

func discardablePureMapCompositeValue(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardablePureMapCompositeElements(toks, start+1, close)
	}
	if toks[start].Text == "map" {
		return discardablePureMapCompositeExpression(toks, start, end)
	}
	if namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks)) >= 0 {
		return discardablePureMapCompositeExpression(toks, start, end)
	}
	if toks[start].Text == "[" {
		return discardableArrayLiteralExpression(toks, start, end)
	}
	return discardableArrayLiteralElement(toks, start, end)
}

func discardableLowerableMapCompositeValue(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardableLowerableMapCompositeElements(toks, start+1, close)
	}
	if toks[start].Text == "map" {
		return discardableLowerableMapCompositeExpression(toks, start, end)
	}
	if namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks)) >= 0 {
		return discardableLowerableMapCompositeExpression(toks, start, end)
	}
	if toks[start].Text == "[" {
		return discardableLowerableArrayCompositeExpression(toks, start, end)
	}
	if discardableArrayLiteralElement(toks, start, end) {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" {
		return true
	}
	return directCallExpressionWithoutCallArgs(toks, start, end)
}

func discardableLowerableMapSliceCompositeValue(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableMapSliceCompositeValue(toks, start+1, close)
		}
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardableLowerableMapCompositeElements(toks, start+1, close)
	}
	if toks[start].Text == "map" {
		return discardableLowerableMapCompositeExpression(toks, start, end)
	}
	if namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks)) >= 0 {
		return discardableLowerableMapCompositeExpression(toks, start, end)
	}
	if toks[start].Text == "make" {
		return discardableLowerableMapMakeExpression(toks, start, end)
	}
	return start+1 == end && toks[start].Text == "nil"
}

func lowerableMapLiteralLenCallContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "len" || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && toks[i-1].Text == "." {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			continue
		}
		args := expressionRanges(toks, i+2, close)
		if len(args) != 1 {
			continue
		}
		if pos < args[0].start || pos >= args[0].end {
			continue
		}
		if discardableLowerableMapCompositeExpression(toks, args[0].start, args[0].end) {
			return true
		}
	}
	return false
}

func pureMapLiteralIndexExpressionContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		start, end, ok := pureMapLiteralIndexExpressionRange(toks, i)
		if !ok || pos < start || pos >= end {
			continue
		}
		return true
	}
	return false
}

func lowerableMapLiteralIndexExpressionContainingToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	if pureMapLiteralIndexExpressionContainingToken(toks, pos) {
		return true
	}
	for i := 0; i < len(toks); i++ {
		start, end, ok := lowerableMapLiteralIndexExpressionRange(toks, i)
		if !ok || pos < start || pos >= end {
			continue
		}
		return true
	}
	return false
}

func lowerableMapLiteralIndexExpressionRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos >= len(toks) {
		return 0, 0, false
	}
	start := pos
	mapStart := pos
	mapEnd := -1
	indexOpen := -1
	if toks[pos].Text == "(" {
		closeParen := findClose(toks, pos, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || toks[closeParen+1].Text != "[" {
			return 0, 0, false
		}
		mapStart = pos + 1
		mapEnd = closeParen
		indexOpen = closeParen + 1
	} else if toks[pos].Text == "make" && pos+1 < len(toks) && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
			return 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	} else {
		open := pureMapCompositeLiteralOpen(toks, mapStart, len(toks))
		if open < 0 {
			open = namedMapCompositeLiteralOpen(toks, mapStart, len(toks), namedMapTypes(toks))
		}
		if open < 0 {
			return 0, 0, false
		}
		close := findClose(toks, open, "{", "}")
		if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
			return 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	}
	indexClose := findClose(toks, indexOpen, "[", "]")
	if indexClose < 0 {
		return 0, 0, false
	}
	key, ok := mapLiteralComparableKey(toks, indexOpen+1, indexClose)
	if !ok {
		return 0, 0, false
	}
	if !lowerableMapIndexValueSupported(toks, mapStart, mapEnd, key) {
		return 0, 0, false
	}
	return start, indexClose + 1, true
}

func pureMapLiteralIndexExpressionRange(toks []scan.Token, pos int) (int, int, bool) {
	if pos < 0 || pos >= len(toks) {
		return 0, 0, false
	}
	if start, end, ok := pureMapAliasIndexExpressionRange(toks, pos); ok {
		return start, end, true
	}
	start := pos
	mapStart := pos
	mapEnd := -1
	indexOpen := -1
	if toks[pos].Text == "(" {
		closeParen := findClose(toks, pos, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || toks[closeParen+1].Text != "[" {
			return 0, 0, false
		}
		mapStart = pos + 1
		mapEnd = closeParen
		indexOpen = closeParen + 1
	} else if toks[pos].Text == "make" && pos+1 < len(toks) && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
			return 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	} else {
		open := pureMapCompositeLiteralOpen(toks, mapStart, len(toks))
		if open < 0 {
			open = namedMapCompositeLiteralOpen(toks, mapStart, len(toks), namedMapTypes(toks))
		}
		if open < 0 {
			return 0, 0, false
		}
		close := findClose(toks, open, "{", "}")
		if close < 0 || close+1 >= len(toks) || toks[close+1].Text != "[" {
			return 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	}
	indexClose := findClose(toks, indexOpen, "[", "]")
	if indexClose < 0 {
		return 0, 0, false
	}
	key, ok := mapLiteralComparableKey(toks, indexOpen+1, indexClose)
	if !ok {
		return 0, 0, false
	}
	if !pureMapIndexValueSupported(toks, mapStart, mapEnd, key) {
		return 0, 0, false
	}
	return start, indexClose + 1, true
}

func pureMapLiteralCommaOkResultCount(toks []scan.Token, start int, end int) int {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return -1
	}
	literalStart, literalEnd, ok := pureMapLiteralIndexExpressionRange(toks, start)
	if !ok || literalStart != start || literalEnd != end {
		return -1
	}
	return 2
}

func lowerableMapLiteralCommaOkResultCount(toks []scan.Token, start int, end int) int {
	if pureMapLiteralCommaOkResultCount(toks, start, end) == 2 {
		return 2
	}
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return -1
	}
	literalStart, literalEnd, ok := lowerableMapLiteralIndexExpressionRange(toks, start)
	if !ok || literalStart != start || literalEnd != end {
		return -1
	}
	return 2
}

func pureMapIndexValueSupported(toks []scan.Token, start int, end int, key string) bool {
	start, end = trimExpressionRange(toks, start, end)
	if discardablePureMapMakeExpression(toks, start, end) {
		return mapSimpleZeroValueSupported(toks, start, end)
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks))
	}
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	if !discardablePureMapCompositeElements(toks, open+1, close) {
		return false
	}
	values := expressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		valueKey, ok := mapLiteralComparableKey(toks, value.start, colon)
		if !ok || valueKey != key {
			continue
		}
		return discardableArrayLiteralElement(toks, colon+1, value.end)
	}
	return mapSimpleZeroValueSupported(toks, start, end)
}

func lowerableMapIndexValueSupported(toks []scan.Token, start int, end int, key string) bool {
	start, end = trimExpressionRange(toks, start, end)
	if discardableLowerableMapMakeExpression(toks, start, end) {
		return mapSimpleZeroValueSupported(toks, start, end)
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 {
		open = namedMapCompositeLiteralOpen(toks, start, end, namedMapTypes(toks))
	}
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	if !mapSimpleZeroValueSupported(toks, start, end) {
		return false
	}
	values := expressionRanges(toks, open+1, close)
	found := false
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		valueKey, ok := mapLiteralComparableKey(toks, value.start, colon)
		if !ok {
			return false
		}
		if valueKey == key {
			found = true
			if !lowerableMapSelectedIndexValueSupported(toks, colon+1, value.end) {
				return false
			}
			continue
		}
		if !discardableLowerableMapCompositeValue(toks, colon+1, value.end) {
			return false
		}
	}
	return found || mapSimpleZeroValueSupported(toks, start, end)
}

func lowerableMapSelectedIndexValueSupported(toks []scan.Token, start int, end int) bool {
	if discardableArrayLiteralElement(toks, start, end) {
		return true
	}
	if singleIdentifierExpression(toks, start, end) != "" {
		return true
	}
	return directCallExpressionWithoutCallArgs(toks, start, end)
}

func mapSimpleZeroValueSupported(toks []scan.Token, start int, end int) bool {
	typ := mapElementTypeText(toks, start, end)
	switch typ {
	case "bool", "string", "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]")
}

func mapRangeKeyTypeSupported(typ string) bool {
	switch typ {
	case "bool", "string", "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return false
}

func mapRangeValueTypeSupported(typ string) bool {
	switch typ {
	case "bool", "string", "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]")
}

func mapExpressionKeyValueTypeText(toks []scan.Token, start int, end int) (string, string) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return "", ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapExpressionKeyValueTypeText(toks, start+1, close)
		}
	}
	if start+1 < end && toks[start].Text == "make" && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close != end-1 {
			return "", ""
		}
		args := expressionRanges(toks, start+2, close)
		if len(args) != 1 && len(args) != 2 {
			return "", ""
		}
		return mapTypeKeyValueText(toks, args[0].start, args[0].end)
	}
	namedMaps := namedMapTypes(toks)
	if info, ok := namedMapExpressionType(toks, start, end, namedMaps); ok {
		open := namedMapCompositeLiteralOpen(toks, start, end, namedMaps)
		if open >= 0 {
			close := findClose(toks, open, "{", "}")
			if close == end-1 {
				return info.keyType, info.valueType
			}
		}
	}
	open := pureMapCompositeLiteralOpen(toks, start, end)
	if open < 0 || start+1 >= open {
		return "", ""
	}
	return mapTypeKeyValueText(toks, start, open)
}

func namedMapExpressionType(toks []scan.Token, start int, end int, namedMaps []namedMapTypeInfo) (namedMapTypeInfo, bool) {
	if start < 0 || start >= end || toks[start].Kind != scan.Ident {
		return namedMapTypeInfo{}, false
	}
	if info, ok := namedMapTypeByName(namedMaps, toks[start].Text); ok {
		return info, true
	}
	if start+2 < end && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		return namedMapTypeBySelector(namedMaps, toks[start].Text, toks[start+2].Text)
	}
	return namedMapTypeInfo{}, false
}

func mapElementTypeText(toks []scan.Token, start int, end int) string {
	_, valueType := mapExpressionKeyValueTypeText(toks, start, end)
	return valueType
}

func mapTypeElementText(toks []scan.Token, start int, end int) string {
	_, valueType := mapTypeKeyValueText(toks, start, end)
	return valueType
}

func mapTypeKeyValueText(toks []scan.Token, start int, end int) (string, string) {
	start, end = trimExpressionRange(toks, start, end)
	if start+3 > end || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return "", ""
	}
	keyClose := findClose(toks, start+1, "[", "]")
	if keyClose < 0 || keyClose+1 >= end {
		return "", ""
	}
	keyType := typeTextInRange(toks, start+2, keyClose)
	valueType := typeTextInRange(toks, keyClose+1, end)
	return keyType, valueType
}

func mapLiteralComparableKey(toks []scan.Token, start int, end int) (string, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapLiteralComparableKey(toks, start+1, close)
		}
	}
	if start+1 == end {
		tok := toks[start]
		if tok.Kind == scan.String || tok.Kind == scan.Char || tok.Text == "true" || tok.Text == "false" {
			return tok.Text, true
		}
		if tok.Kind == scan.Number {
			return tok.Text, true
		}
	}
	if start+2 == end && (toks[start].Text == "-" || toks[start].Text == "+") && toks[start+1].Kind == scan.Number {
		return toks[start].Text + toks[start+1].Text, true
	}
	return "", false
}

func discardedPureComplexStatementContainingToken(toks []scan.Token, pos int) bool {
	return discardedComplexStatementContainingToken(toks, pos, nil, discardablePureComplexExpression)
}

func discardedLowerableComplexStatementContainingToken(toks []scan.Token, pos int, sigs []funcSignature) bool {
	return discardedComplexStatementContainingToken(toks, pos, sigs, discardableLowerableComplexExpression)
}

func lowerableComplexVarBlankDiscardContainingToken(file parse.File, toks []scan.Token, pos int, sigs []funcSignature) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" && (toks[i].Kind != scan.Ident || i+1 >= len(toks) || toks[i+1].Text != ":=") {
			continue
		}
		varEnd, discardStart, discardEnd, ok := lowerableComplexVarBlankDiscardAt(file, toks, i, sigs)
		if !ok {
			continue
		}
		if (pos >= i && pos < varEnd) || (pos >= discardStart && pos < discardEnd) {
			return true
		}
	}
	return false
}

func lowerableComplexVarBlankDiscardAt(file parse.File, toks []scan.Token, start int, sigs []funcSignature) (int, int, int, bool) {
	if start < 0 || start+3 >= len(toks) {
		return 0, 0, 0, false
	}
	body := functionBodyOpenContainingToken(file, start)
	if body < 0 {
		return 0, 0, 0, false
	}
	bodyClose := findClose(toks, body, "{", "}")
	if bodyClose < 0 {
		return 0, 0, 0, false
	}
	if simpleStatementStart(toks, body, start) != start {
		return 0, 0, 0, false
	}
	stmtEnd := simpleStatementEnd(toks, start, bodyClose)
	name := ""
	eq := -1
	if toks[start].Text == "var" {
		if start+4 >= stmtEnd || toks[start+1].Kind != scan.Ident {
			return 0, 0, 0, false
		}
		name = toks[start+1].Text
		eq = findTopLevelToken(toks, start+2, stmtEnd, "=")
		if eq < 0 || isCompoundAssignmentEquals(toks, eq) || eq+1 >= stmtEnd {
			return 0, 0, 0, false
		}
		if eq > start+2 {
			if eq != start+3 || (toks[start+2].Text != "complex64" && toks[start+2].Text != "complex128") {
				return 0, 0, 0, false
			}
		}
	} else {
		if toks[start].Kind != scan.Ident || start+2 >= stmtEnd || toks[start+1].Text != ":=" {
			return 0, 0, 0, false
		}
		name = toks[start].Text
		eq = start + 1
	}
	if !discardableLowerableComplexExpression(toks, eq+1, stmtEnd, sigs) {
		return 0, 0, 0, false
	}
	discardStart := nextSimpleStatementStart(toks, stmtEnd, bodyClose)
	if discardStart < 0 {
		return 0, 0, 0, false
	}
	discardEnd := simpleStatementEnd(toks, discardStart, bodyClose)
	if !blankDiscardOfIdentifierStatement(toks, discardStart, discardEnd, name) {
		return 0, 0, 0, false
	}
	if identifierUsedOutsideRanges(toks, name, body+1, bodyClose, start, stmtEnd, discardStart, discardEnd) {
		return 0, 0, 0, false
	}
	return stmtEnd, discardStart, discardEnd, true
}

func lowerableComplexAliasComponentContainingToken(file parse.File, toks []scan.Token, pos int, sigs []funcSignature) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" && (toks[i].Kind != scan.Ident || i+1 >= len(toks) || toks[i+1].Text != ":=") {
			continue
		}
		declStart, declEnd, uses, ok := lowerableComplexAliasComponentAt(file, toks, i, sigs)
		if !ok {
			continue
		}
		if pos >= declStart && pos < declEnd {
			return true
		}
		if tokenInExpressionRanges(pos, uses) {
			return true
		}
		i = declEnd - 1
	}
	return false
}

func lowerableComplexAliasComponentAt(file parse.File, toks []scan.Token, start int, sigs []funcSignature) (int, int, []expressionRange, bool) {
	if start < 0 || start+3 >= len(toks) {
		return 0, 0, nil, false
	}
	body := functionBodyOpenContainingToken(file, start)
	if body < 0 {
		return 0, 0, nil, false
	}
	bodyClose := findClose(toks, body, "{", "}")
	if bodyClose < 0 {
		return 0, 0, nil, false
	}
	if simpleStatementStart(toks, body, start) != start {
		return 0, 0, nil, false
	}
	stmtEnd := simpleStatementEnd(toks, start, bodyClose)
	name := ""
	eq := -1
	if toks[start].Text == "var" {
		if start+3 >= stmtEnd || toks[start+1].Kind != scan.Ident || toks[start+1].Text == "_" {
			return 0, 0, nil, false
		}
		name = toks[start+1].Text
		eq = findTopLevelToken(toks, start+2, stmtEnd, "=")
		if eq < 0 || isCompoundAssignmentEquals(toks, eq) || eq+1 >= stmtEnd {
			return 0, 0, nil, false
		}
		if eq > start+2 {
			if eq != start+3 || (toks[start+2].Text != "complex64" && toks[start+2].Text != "complex128") {
				return 0, 0, nil, false
			}
		}
	} else {
		if toks[start].Kind != scan.Ident || toks[start].Text == "_" || start+2 >= stmtEnd || toks[start+1].Text != ":=" {
			return 0, 0, nil, false
		}
		name = toks[start].Text
		eq = start + 1
	}
	if !lowerableComplexAliasExpression(toks, eq+1, stmtEnd, sigs) {
		return 0, 0, nil, false
	}
	uses, ok := lowerableComplexAliasComponentUseRanges(toks, name, stmtEnd, bodyClose, sigs)
	if !ok || len(uses) == 0 {
		return 0, 0, nil, false
	}
	if identifierUsedOutsideRangeList(toks, name, body+1, bodyClose, start, stmtEnd, uses) {
		return 0, 0, nil, false
	}
	return start, stmtEnd, uses, true
}

func lowerableComplexAliasComponentVarSpecLowerable(file parse.File, toks []scan.Token, start int, sigs []funcSignature) bool {
	if start <= 0 || start >= len(toks) {
		return false
	}
	varStart := start - 1
	if toks[varStart].Text != "var" {
		return false
	}
	declStart, _, _, ok := lowerableComplexAliasComponentAt(file, toks, varStart, sigs)
	return ok && declStart == varStart
}

func lowerableComplexAliasExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableComplexAliasExpression(toks, start+1, close, sigs)
		}
	}
	if _, _, ok := reducibleComplexLiteralParts(toks, start, end); ok {
		return true
	}
	if start+3 > end || toks[start].Text != "complex" || toks[start+1].Text != "(" {
		return false
	}
	if signedFunctionCallAt(toks, start, sigs) || (start > 0 && toks[start-1].Text == ".") {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	args := expressionRanges(toks, start+2, close)
	if len(args) != 2 {
		return false
	}
	return lowerableComplexAliasComponentExpression(toks, args[0].start, args[0].end, sigs) &&
		lowerableComplexAliasComponentExpression(toks, args[1].start, args[1].end, sigs)
}

func lowerableComplexAliasComponentExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableComplexAliasComponentExpression(toks, start+1, close, sigs)
		}
	}
	if discardablePureRealNumberExpression(toks, start, end) {
		return true
	}
	return lowerableDirectCallResultType(toks, start, end, sigs) == "float64"
}

func lowerableComplexAliasComponentUseRanges(toks []scan.Token, name string, start int, end int, sigs []funcSignature) ([]expressionRange, bool) {
	var uses []expressionRange
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		useStart, useEnd, ok := lowerableComplexAliasComponentUseRange(toks, i, sigs)
		if !ok {
			return nil, false
		}
		uses = append(uses, expressionRange{start: useStart, end: useEnd})
		i = useEnd - 1
	}
	return uses, true
}

func lowerableComplexAliasComponentUseRange(toks []scan.Token, namePos int, sigs []funcSignature) (int, int, bool) {
	if namePos < 2 || namePos+1 >= len(toks) || toks[namePos-1].Text != "(" {
		return 0, 0, false
	}
	callStart := namePos - 2
	if toks[callStart].Text != "real" && toks[callStart].Text != "imag" {
		return 0, 0, false
	}
	if signedFunctionCallAt(toks, callStart, sigs) || (callStart > 0 && toks[callStart-1].Text == ".") {
		return 0, 0, false
	}
	close := findClose(toks, namePos-1, "(", ")")
	if close != namePos+1 {
		return 0, 0, false
	}
	return callStart, close + 1, true
}

func lowerableInterfaceVarBlankDiscardContainingToken(file parse.File, toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" {
			continue
		}
		varEnd, discardStart, discardEnd, ok := lowerableInterfaceVarBlankDiscardAt(file, toks, i)
		if !ok {
			continue
		}
		if (pos >= i && pos < varEnd) || (pos >= discardStart && pos < discardEnd) {
			return true
		}
	}
	return false
}

func lowerableUnusedInterfaceParamTypeToken(file parse.File, pos int, sigs []funcSignature) bool {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "func" || decl.Receiver || decl.Name == "" || isExported(decl.Name) {
			continue
		}
		if lowerableUnusedInterfaceParamDeclContainsToken(toks, decl, pos, sigs) {
			return true
		}
	}
	return false
}

func lowerableUnusedInterfaceParamDeclContainsToken(toks []scan.Token, decl parse.Decl, pos int, sigs []funcSignature) bool {
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return false
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || pos <= paramsOpen || pos >= paramsClose {
		return false
	}
	bodyOpen := functionBodyOpenForDeclAfterParams(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBefore(toks, decl.End)
	}
	infos := unusedInterfaceParamInfos(toks, paramsOpen+1, paramsClose, bodyOpen, bodyClose)
	if len(infos) == 0 {
		return false
	}
	matched := false
	var indexes []int
	for i := 0; i < len(infos); i++ {
		info := infos[i]
		indexes = append(indexes, info.index)
		if pos >= info.typeStart && pos < info.typeEnd {
			matched = true
		}
	}
	return matched && unusedInterfaceParamCallSitesLowerable(toks, decl.Name, indexes, sigs)
}

func functionBodyOpenForDeclAfterParams(toks []scan.Token, paramsClose int, declEnd int) int {
	bodyClose := tokenIndexBefore(toks, declEnd)
	if bodyClose < 0 || toks[bodyClose].Text != "}" {
		return -1
	}
	for i := paramsClose + 1; i < bodyClose; i++ {
		if toks[i].Text != "{" {
			continue
		}
		if findClose(toks, i, "{", "}") == bodyClose {
			return i
		}
	}
	return -1
}

type unusedInterfaceParamInfo struct {
	index     int
	typeStart int
	typeEnd   int
}

func unusedInterfaceParamInfos(toks []scan.Token, start int, end int, bodyOpen int, bodyClose int) []unusedInterfaceParamInfo {
	var out []unusedInterfaceParamInfo
	paramIndex := 0
	var pendingNames []string
	segments := expressionRanges(toks, start, end)
	for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
		segment := segments[segmentIndex]
		if name, ok := bareParameterNameSegment(toks, segment.start, segment.end); ok {
			pendingNames = append(pendingNames, name)
			continue
		}
		names, typeStart, typeEnd, ok := interfaceParamSegment(toks, segment.start, segment.end)
		count := parameterSegmentCount(toks, segment.start, segment.end)
		if ok && len(names) == count {
			names = append(append([]string{}, pendingNames...), names...)
			count = len(names)
			used := false
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := names[nameIndex]
				if name != "_" && identifierUsedInTokenRange(toks, name, bodyOpen+1, bodyClose) {
					used = true
					break
				}
			}
			if !used {
				for nameIndex := 0; nameIndex < len(names); nameIndex++ {
					out = append(out, unusedInterfaceParamInfo{index: paramIndex + nameIndex, typeStart: typeStart, typeEnd: typeEnd})
				}
			}
		} else {
			count += len(pendingNames)
		}
		paramIndex += count
		pendingNames = nil
	}
	return out
}

func unusedInterfaceParamIndexesForSignature(toks []scan.Token, start int, end int, bodyOpen int, bodyClose int) []int {
	infos := unusedInterfaceParamInfos(toks, start, end, bodyOpen, bodyClose)
	var out []int
	for i := 0; i < len(infos); i++ {
		out = append(out, infos[i].index)
	}
	return out
}

func bareParameterNameSegment(toks []scan.Token, start int, end int) (string, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text, true
	}
	return "", false
}

func interfaceParamSegment(toks []scan.Token, start int, end int) ([]string, int, int, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return nil, 0, 0, false
	}
	typeStart := -1
	typeEnd := end
	for i := start; i < end; i++ {
		if toks[i].Text == "interface" && i+1 < end && toks[i+1].Text == "{" {
			close := findClose(toks, i+1, "{", "}")
			if close >= i+1 && close < end {
				typeStart = i
				typeEnd = close + 1
				break
			}
		}
		if toks[i].Kind == scan.Ident && toks[i].Text == "any" {
			typeStart = i
			typeEnd = i + 1
			break
		}
	}
	if typeStart <= start {
		return nil, 0, 0, false
	}
	var names []string
	for i := start; i < typeStart; i++ {
		if toks[i].Kind == scan.Ident {
			names = append(names, toks[i].Text)
		}
	}
	if len(names) == 0 {
		return nil, 0, 0, false
	}
	return names, typeStart, typeEnd, true
}

func identifierUsedInTokenRange(toks []scan.Token, name string, start int, end int) bool {
	for i := start; i < len(toks) && i < end; i++ {
		if toks[i].Kind == scan.Ident && toks[i].Text == name {
			return true
		}
	}
	return false
}

func unusedInterfaceParamCallSitesLowerable(toks []scan.Token, name string, indexes []int, sigs []funcSignature) bool {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && (toks[i-1].Text == "func" || toks[i-1].Text == ".") {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			return false
		}
		args := expressionRanges(toks, i+2, close)
		if !unusedInterfaceParamCallSiteLowerable(toks, i, close, args, indexes, sigs) {
			return false
		}
		i = close
	}
	return true
}

func unusedInterfaceParamCallSiteLowerable(toks []scan.Token, pos int, close int, args []expressionRange, indexes []int, sigs []funcSignature) bool {
	lastSideEffectErasedArg := -1
	for indexIndex := 0; indexIndex < len(indexes); indexIndex++ {
		index := indexes[indexIndex]
		if index < 0 || index >= len(args) {
			return false
		}
		arg := args[index]
		if !expressionContainsCallToken(toks, arg.start, arg.end) {
			continue
		}
		if !unusedInterfaceParamDirectCallStatementSite(toks, pos, close) && !unusedInterfaceParamDeferCallStatementSite(toks, pos, close) && !unusedInterfaceParamReturnExpressionSite(toks, pos, close) && !unusedInterfaceParamAssignmentExpressionSite(toks, pos, close) && !unusedInterfaceParamVarInitializerSite(toks, pos, close) && !unusedInterfaceParamIfConditionSite(toks, pos, close) && !unusedInterfaceParamForConditionSite(toks, pos, close) && !unusedInterfaceParamClassicForConditionSite(toks, pos, close) && !unusedInterfaceParamSwitchTagSite(toks, pos, close) {
			return false
		}
		if !interfaceReturnSideEffectExpressionLowerable(toks, arg.start, arg.end, sigs) {
			return false
		}
		if index > lastSideEffectErasedArg {
			lastSideEffectErasedArg = index
		}
	}
	if lastSideEffectErasedArg < 0 {
		return true
	}
	for argIndex := 0; argIndex < lastSideEffectErasedArg; argIndex++ {
		if interfaceParamIndexInSet(indexes, argIndex) {
			continue
		}
		arg := args[argIndex]
		if expressionContainsCallToken(toks, arg.start, arg.end) {
			return false
		}
	}
	return true
}

func unusedInterfaceParamDirectCallStatementSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart != pos {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	return stmtEnd == close+1
}

func unusedInterfaceParamDeferCallStatementSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart+1 != pos || toks[stmtStart].Text != "defer" {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	return stmtEnd == close+1
}

func unusedInterfaceParamReturnExpressionSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStart(toks, -1, pos)
	if stmtStart < 0 || toks[stmtStart].Text != "return" {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	values := expressionRanges(toks, stmtStart+1, stmtEnd)
	if len(values) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, values[0].start, values[0].end)
	return start == pos && end == close+1
}

func unusedInterfaceParamAssignmentExpressionSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(rhs) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func unusedInterfaceParamVarInitializerSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	if stmtStart < 0 || toks[stmtStart].Text != "var" {
		return false
	}
	if stmtStart+1 < len(toks) && toks[stmtStart+1].Text == "(" {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(rhs) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func unusedInterfaceParamIfConditionSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "if" {
		return false
	}
	start, end, ok := staticInterfaceAssertionIfConditionRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func unusedInterfaceParamForConditionSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "for" {
		return false
	}
	start, end, ok := staticInterfaceAssertionForConditionRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func unusedInterfaceParamClassicForConditionSite(toks []scan.Token, pos int, close int) bool {
	if pos < 2 || close+1 >= len(toks) || toks[pos-1].Text != ";" || toks[close+1].Text != ";" {
		return false
	}
	forPos := pos - 2
	for forPos >= 0 && toks[forPos].Text != "for" {
		if toks[forPos].Text == ";" || toks[forPos].Text == "{" || toks[forPos].Text == "}" {
			return false
		}
		forPos--
	}
	if forPos < 0 {
		return false
	}
	open := controlBodyOpen(toks, forPos+1, len(toks))
	if open < 0 {
		return false
	}
	secondSemi := findTopLevelToken(toks, pos, open, ";")
	if secondSemi != close+1 {
		return false
	}
	start, end := trimExpressionRange(toks, pos, secondSemi)
	return start == pos && end == close+1
}

func unusedInterfaceParamSwitchTagSite(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "switch" {
		return false
	}
	start, end, ok := staticInterfaceAssertionSwitchTagRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func directKnownFunctionCallExpressionWithoutCallArgs(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directKnownFunctionCallExpressionWithoutCallArgs(toks, start+1, close, sigs)
		}
	}
	if !directCallExpressionWithoutCallArgs(toks, start, end) {
		return false
	}
	return funcSignatureIndex(sigs, toks[start].Text) >= 0
}

func interfaceParamIndexInSet(indexes []int, index int) bool {
	for i := 0; i < len(indexes); i++ {
		if indexes[i] == index {
			return true
		}
	}
	return false
}

func lowerableDiscardedInterfaceReturnTypeToken(file parse.File, pos int, interfaceReturns interfaceReturnLowerableSet, sigs []funcSignature) bool {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "func" || decl.Receiver || decl.Name == "" || isExported(decl.Name) {
			continue
		}
		if lowerableDiscardedInterfaceReturnDeclContainsToken(toks, decl, pos, interfaceReturns, sigs) {
			return true
		}
	}
	return false
}

func lowerableDiscardedInterfaceReturnDeclContainsToken(toks []scan.Token, decl parse.Decl, pos int, interfaceReturns interfaceReturnLowerableSet, sigs []funcSignature) bool {
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return false
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return false
	}
	if interfaceReturnParamsContainInterface(toks, paramsOpen+1, paramsClose) {
		return false
	}
	bodyOpen := functionBodyOpenForDeclAfterParams(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return false
	}
	resultStart, resultEnd, ok := interfaceReturnResultRange(toks, paramsClose+1, bodyOpen)
	if !ok || pos < resultStart || pos >= resultEnd {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBefore(toks, decl.End)
	}
	if !interfaceReturnExpressionsLowerable(toks, bodyOpen, bodyClose, sigs) {
		return false
	}
	if interfaceReturns.enabled {
		return interfaceReturnLowerableSetContains(interfaceReturns, decl.Name)
	}
	return interfaceReturnCallSitesLowerable(toks, decl.Name)
}

func interfaceReturnLowerableSetContains(set interfaceReturnLowerableSet, name string) bool {
	for i := 0; i < len(set.names); i++ {
		if set.names[i] == name {
			return true
		}
	}
	return false
}

func interfaceReturnLowerableSetForLoadFiles(files []load.File, sigs []funcSignature) interfaceReturnLowerableSet {
	var set interfaceReturnLowerableSet
	set.enabled = true
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if !interfaceReturnDeclLowerableForPackage(parsed, decl, sigs) {
				continue
			}
			if !interfaceReturnCallSitesLowerableForLoadFiles(files, decl.Name) {
				continue
			}
			set.names = append(set.names, decl.Name)
		}
	}
	return set
}

func interfaceReturnDeclLowerableForPackage(file parse.File, decl parse.Decl, sigs []funcSignature) bool {
	if decl.Kind != "func" || decl.Receiver || decl.Name == "" || isExported(decl.Name) {
		return false
	}
	toks := file.Tokens
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return false
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return false
	}
	if interfaceReturnParamsContainInterface(toks, paramsOpen+1, paramsClose) {
		return false
	}
	bodyOpen := functionBodyOpenForDeclAfterParams(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return false
	}
	if _, _, ok := interfaceReturnResultRange(toks, paramsClose+1, bodyOpen); !ok {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBefore(toks, decl.End)
	}
	return interfaceReturnExpressionsLowerable(toks, bodyOpen, bodyClose, sigs)
}

func interfaceReturnCallSitesLowerableForLoadFiles(files []load.File, name string) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			return false
		}
		if !interfaceReturnCallSitesLowerable(parsed.Tokens, name) {
			return false
		}
	}
	return true
}

func interfaceReturnParamsContainInterface(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "interface" || startsAnyInterfaceType(toks, i) {
			return true
		}
	}
	return false
}

func interfaceReturnResultRange(toks []scan.Token, start int, end int) (int, int, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return 0, 0, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return interfaceReturnResultRange(toks, start+1, close)
		}
	}
	if start+1 == end && toks[start].Kind == scan.Ident && toks[start].Text == "any" {
		return start, end, true
	}
	if start+3 == end && toks[start].Text == "interface" && toks[start+1].Text == "{" && toks[start+2].Text == "}" {
		return start, end, true
	}
	return 0, 0, false
}

func interfaceReturnExpressionsLowerable(toks []scan.Token, bodyOpen int, bodyClose int, sigs []funcSignature) bool {
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text == "func" {
			return false
		}
		if toks[i].Text != "return" {
			continue
		}
		stmtEnd := simpleStatementEnd(toks, i, bodyClose)
		values := expressionRanges(toks, i+1, stmtEnd)
		if len(values) != 1 {
			return false
		}
		value := values[0]
		if expressionContainsCallToken(toks, value.start, value.end) {
			return interfaceReturnSideEffectExpressionLowerable(toks, value.start, value.end, sigs)
		}
		i = stmtEnd - 1
	}
	return true
}

func directKnownFunctionCallExpressionWithDirectCallArgs(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directKnownFunctionCallExpressionWithDirectCallArgs(toks, start+1, close, sigs)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	if funcSignatureIndex(sigs, toks[start].Text) < 0 {
		return false
	}
	args := expressionRanges(toks, start+2, close)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if expressionContainsCallToken(toks, arg.start, arg.end) && !interfaceReturnSideEffectExpressionLowerable(toks, arg.start, arg.end, sigs) {
			return false
		}
	}
	return true
}

func interfaceReturnSideEffectExpressionLowerable(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if !expressionContainsCallToken(toks, start, end) {
		return true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return interfaceReturnSideEffectExpressionLowerable(toks, start+1, close, sigs)
		}
	}
	if op := topLevelBinaryOperator(toks, start, end); op >= 0 {
		return interfaceReturnSideEffectExpressionLowerable(toks, start, op, sigs) && interfaceReturnSideEffectExpressionLowerable(toks, op+1, end, sigs)
	}
	if op := unaryOperatorText(toks[start].Text); op != "" && start+1 < end {
		return interfaceReturnSideEffectExpressionLowerable(toks, start+1, end, sigs)
	}
	return directKnownFunctionCallExpressionWithDirectCallArgs(toks, start, end, sigs)
}

func interfaceReturnCallSitesLowerable(toks []scan.Token, name string) bool {
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && (toks[i-1].Text == "func" || toks[i-1].Text == ".") {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			return false
		}
		if !interfaceReturnCallSiteLowerable(toks, i, close) {
			return false
		}
		i = close
	}
	return true
}

func interfaceReturnCallSiteLowerable(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if stmtStart == pos && stmtEnd == close+1 {
		return true
	}
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return false
	}
	if !blankIdentifierExpression(toks, lhs[0].start, lhs[0].end) {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func lowerableInterfaceVarBlankDiscardAt(file parse.File, toks []scan.Token, start int) (int, int, int, bool) {
	if start < 0 || start+3 >= len(toks) || toks[start].Text != "var" {
		return 0, 0, 0, false
	}
	body := functionBodyOpenContainingToken(file, start)
	if body < 0 {
		return 0, 0, 0, false
	}
	bodyClose := findClose(toks, body, "{", "}")
	if bodyClose < 0 {
		return 0, 0, 0, false
	}
	if simpleStatementStart(toks, body, start) != start {
		return 0, 0, 0, false
	}
	stmtEnd := simpleStatementEnd(toks, start, bodyClose)
	if start+3 > stmtEnd || toks[start+1].Kind != scan.Ident {
		return 0, 0, 0, false
	}
	name := toks[start+1].Text
	typeEnd := stmtEnd
	if eq := findTopLevelToken(toks, start+2, stmtEnd, "="); eq >= 0 {
		if !nilInterfaceInitializer(toks, eq+1, stmtEnd) {
			return 0, 0, 0, false
		}
		typeEnd = eq
	}
	if !lowerableInterfaceVarTypeRange(toks, start+2, typeEnd) {
		return 0, 0, 0, false
	}
	discardStart := nextSimpleStatementStart(toks, stmtEnd, bodyClose)
	if discardStart < 0 {
		return 0, 0, 0, false
	}
	discardEnd := simpleStatementEnd(toks, discardStart, bodyClose)
	if !blankDiscardOfIdentifierStatement(toks, discardStart, discardEnd, name) {
		return 0, 0, 0, false
	}
	if identifierUsedOutsideRanges(toks, name, body+1, bodyClose, start, stmtEnd, discardStart, discardEnd) {
		return 0, 0, 0, false
	}
	return stmtEnd, discardStart, discardEnd, true
}

type nilInterfaceVarComparisonInfo struct {
	declStart   int
	declEnd     int
	comparisons []expressionRange
}

func lowerableNilInterfaceVarComparisonContainingToken(file parse.File, toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := lowerableNilInterfaceVarComparisonsAt(file, toks, i)
		if !ok {
			continue
		}
		if pos >= info.declStart && pos < info.declEnd {
			return true
		}
	}
	return false
}

func lowerableNilInterfaceVarComparisonsAt(file parse.File, toks []scan.Token, start int) (nilInterfaceVarComparisonInfo, bool) {
	if start < 0 || start+3 >= len(toks) || toks[start].Text != "var" {
		return nilInterfaceVarComparisonInfo{}, false
	}
	body := functionBodyOpenContainingToken(file, start)
	if body < 0 {
		return nilInterfaceVarComparisonInfo{}, false
	}
	bodyClose := findClose(toks, body, "{", "}")
	if bodyClose < 0 {
		return nilInterfaceVarComparisonInfo{}, false
	}
	if simpleStatementStart(toks, body, start) != start {
		return nilInterfaceVarComparisonInfo{}, false
	}
	stmtEnd := simpleStatementEnd(toks, start, bodyClose)
	if start+3 > stmtEnd || toks[start+1].Kind != scan.Ident {
		return nilInterfaceVarComparisonInfo{}, false
	}
	name := toks[start+1].Text
	typeEnd := stmtEnd
	if eq := findTopLevelToken(toks, start+2, stmtEnd, "="); eq >= 0 {
		if !nilInterfaceInitializer(toks, eq+1, stmtEnd) {
			return nilInterfaceVarComparisonInfo{}, false
		}
		typeEnd = eq
	}
	if !lowerableInterfaceVarTypeRange(toks, start+2, typeEnd) {
		return nilInterfaceVarComparisonInfo{}, false
	}
	comparisons := nilInterfaceComparisonRanges(toks, name, stmtEnd, bodyClose)
	if len(comparisons) == 0 {
		return nilInterfaceVarComparisonInfo{}, false
	}
	if identifierUsedOutsideRangeList(toks, name, body+1, bodyClose, start, stmtEnd, comparisons) {
		return nilInterfaceVarComparisonInfo{}, false
	}
	return nilInterfaceVarComparisonInfo{declStart: start, declEnd: stmtEnd, comparisons: comparisons}, true
}

func nilInterfaceComparisonRanges(toks []scan.Token, name string, start int, end int) []expressionRange {
	var out []expressionRange
	for i := start + 1; i+1 < end; i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left := nilInterfaceComparisonOperand(toks, i-1, start, end)
		right := nilInterfaceComparisonOperand(toks, i+1, start, end)
		if !nilInterfaceComparisonNamesMatch(left, right, name) {
			continue
		}
		if len(out) > 0 && i-1 < out[len(out)-1].end {
			continue
		}
		out = append(out, expressionRange{start: i - 1, end: i + 2})
	}
	return out
}

func nilInterfaceComparisonOperand(toks []scan.Token, pos int, start int, end int) string {
	if pos < start || pos >= end || toks[pos].Kind != scan.Ident {
		return ""
	}
	if pos > start {
		prev := toks[pos-1]
		if prev.Text == "." || prev.Text == "!" || prev.Text == "*" || prev.Text == "&" || prev.Text == "<-" {
			return ""
		}
	}
	if pos+1 < end {
		next := toks[pos+1]
		if next.Text == "." || next.Text == "(" || next.Text == "[" {
			return ""
		}
	}
	return toks[pos].Text
}

func nilInterfaceComparisonNamesMatch(left string, right string, name string) bool {
	return (left == name && right == "nil") || (left == "nil" && right == name)
}

func identifierUsedOutsideRangeList(toks []scan.Token, name string, scopeStart int, scopeEnd int, firstStart int, firstEnd int, ranges []expressionRange) bool {
	for i := scopeStart; i < len(toks) && i < scopeEnd; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		if i >= firstStart && i < firstEnd {
			continue
		}
		if tokenInExpressionRanges(i, ranges) {
			continue
		}
		return true
	}
	return false
}

func tokenInExpressionRanges(pos int, ranges []expressionRange) bool {
	for i := 0; i < len(ranges); i++ {
		if pos >= ranges[i].start && pos < ranges[i].end {
			return true
		}
	}
	return false
}

func nilInterfaceInitializer(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	return start+1 == end && toks[start].Kind == scan.Ident && toks[start].Text == "nil"
}

func lowerableInterfaceVarTypeRange(toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if start+1 == end && toks[start].Text == "any" {
		return true
	}
	if toks[start].Text != "interface" || start+2 >= end || toks[start+1].Text != "{" {
		return false
	}
	close := findClose(toks, start+1, "{", "}")
	return close == end-1 && start+2 == close
}

type staticInterfaceAssertionVar struct {
	name         string
	concreteType string
	nilInterface bool
	declStart    int
	declEnd      int
	bodyOpen     int
	bodyClose    int
}

type staticInterfaceTypeSwitchGuard struct {
	sourceName  string
	bindingName string
	sourcePos   int
	dot         int
	open        int
}

var activeImportedStaticInterfaceStructs []structFieldSet

func staticInterfaceStructsForFile(file parse.File) []structFieldSet {
	structs := fileStructTypesWithTypes(file, fileNamedTypeUnderlyings(file))
	return appendStructFieldSets(structs, activeImportedStaticInterfaceStructs)
}

func lowerableStaticInterfaceAssertionVarContainingToken(file parse.File, toks []scan.Token, pos int) bool {
	bodyOpen := functionBodyOpenContainingToken(file, pos)
	if bodyOpen < 0 {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		return false
	}
	structs := staticInterfaceStructsForFile(file)
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAt(toks, i, bodyOpen, bodyClose, structs)
		if !ok {
			continue
		}
		if pos >= info.declStart && pos < info.declEnd {
			return true
		}
		i = info.declEnd - 1
	}
	return false
}

func lowerableStaticInterfaceAssertionAt(file parse.File, toks []scan.Token, dot int) bool {
	if dot <= 0 || dot+1 >= len(toks) || toks[dot].Text != "." || toks[dot-1].Kind != scan.Ident {
		return false
	}
	bodyOpen := functionBodyOpenContainingToken(file, dot)
	if bodyOpen < 0 {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		return false
	}
	structs := staticInterfaceStructsForFile(file)
	for i := bodyOpen + 1; i < dot; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAt(toks, i, bodyOpen, bodyClose, structs)
		if !ok {
			continue
		}
		if info.name != toks[dot-1].Text {
			i = info.declEnd - 1
			continue
		}
		asserted := staticInterfaceAssertionTypeName(toks, dot)
		close := findClose(toks, dot+1, "(", ")")
		if asserted == info.concreteType || staticInterfaceAssertionMismatchLowerable(toks, dot-1, close, asserted, structs) {
			return true
		}
		i = info.declEnd - 1
	}
	return false
}

func lowerableStaticInterfaceTypeSwitchAt(file parse.File, toks []scan.Token, dot int) bool {
	if dot <= 1 || dot+3 >= len(toks) || toks[dot].Text != "." {
		return false
	}
	for _, switchPos := range []int{dot - 2, dot - 4} {
		guard, ok := staticInterfaceTypeSwitchGuardAt(toks, switchPos)
		if ok && guard.dot == dot {
			return lowerableStaticInterfaceTypeSwitchAtSwitch(file, toks, switchPos)
		}
	}
	return false
}

func lowerableStaticInterfaceTypeSwitchAtSwitch(file parse.File, toks []scan.Token, switchPos int) bool {
	guard, ok := staticInterfaceTypeSwitchGuardAt(toks, switchPos)
	if !ok {
		return false
	}
	bodyOpen := functionBodyOpenContainingToken(file, switchPos)
	if bodyOpen < 0 {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		return false
	}
	structs := staticInterfaceStructsForFile(file)
	for i := bodyOpen + 1; i < switchPos; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAt(toks, i, bodyOpen, bodyClose, structs)
		if !ok {
			continue
		}
		if info.name == guard.sourceName && staticInterfaceTypeSwitchBodyLowerable(toks, switchPos, info, structs) {
			return true
		}
		i = info.declEnd - 1
	}
	return false
}

func staticInterfaceTypeSwitchGuardAt(toks []scan.Token, switchPos int) (staticInterfaceTypeSwitchGuard, bool) {
	if switchPos < 0 || switchPos >= len(toks) || toks[switchPos].Text != "switch" {
		return staticInterfaceTypeSwitchGuard{}, false
	}
	if switchPos+6 < len(toks) && toks[switchPos+1].Kind == scan.Ident && toks[switchPos+2].Text == "." {
		dot := switchPos + 2
		assertClose := findClose(toks, dot+1, "(", ")")
		if assertClose == dot+3 && toks[dot+2].Text == "type" && assertClose+1 < len(toks) && toks[assertClose+1].Text == "{" {
			return staticInterfaceTypeSwitchGuard{sourceName: toks[switchPos+1].Text, sourcePos: switchPos + 1, dot: dot, open: assertClose + 1}, true
		}
	}
	if switchPos+8 < len(toks) && toks[switchPos+1].Kind == scan.Ident && toks[switchPos+2].Text == ":=" && toks[switchPos+3].Kind == scan.Ident && toks[switchPos+4].Text == "." {
		dot := switchPos + 4
		assertClose := findClose(toks, dot+1, "(", ")")
		if assertClose == dot+3 && toks[dot+2].Text == "type" && assertClose+1 < len(toks) && toks[assertClose+1].Text == "{" {
			return staticInterfaceTypeSwitchGuard{sourceName: toks[switchPos+3].Text, bindingName: toks[switchPos+1].Text, sourcePos: switchPos + 3, dot: dot, open: assertClose + 1}, true
		}
	}
	return staticInterfaceTypeSwitchGuard{}, false
}

func staticInterfaceAssertionVarSpecLowerable(file parse.File, toks []scan.Token, start int, end int) bool {
	if start <= 0 || start >= len(toks) {
		return false
	}
	bodyOpen := functionBodyOpenContainingToken(file, start)
	if bodyOpen < 0 {
		return false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		return false
	}
	varStart := start - 1
	if toks[varStart].Text != "var" {
		return false
	}
	info, ok := staticInterfaceAssertionVarAt(toks, varStart, bodyOpen, bodyClose, staticInterfaceStructsForFile(file))
	return ok && info.declStart == varStart && end <= info.declEnd
}

func staticInterfaceAssertionAssignmentInLocalVarDecl(file parse.File, toks []scan.Token, body int, assign int) bool {
	if assign <= body || assign >= len(toks) || toks[assign].Text != "=" {
		return false
	}
	for start := assign - 1; start > body && toks[start].Line == toks[assign].Line; start-- {
		if toks[start].Text == ";" || toks[start].Text == "=" {
			return false
		}
		if toks[start].Text != "var" {
			continue
		}
		end := simpleStatementEnd(toks, start+1, len(toks))
		return staticInterfaceAssertionVarSpecLowerable(file, toks, start+1, end)
	}
	return false
}

func staticInterfaceAssertionVarAt(toks []scan.Token, start int, bodyOpen int, bodyClose int, structs []structFieldSet) (staticInterfaceAssertionVar, bool) {
	stmtEnd := simpleStatementEnd(toks, start, bodyClose)
	if start < 0 || start+3 > stmtEnd || toks[start].Text != "var" {
		return staticInterfaceAssertionVar{}, false
	}
	if toks[start+1].Kind != scan.Ident || toks[start+1].Text == "_" {
		return staticInterfaceAssertionVar{}, false
	}
	eq := findTopLevelToken(toks, start, stmtEnd, "=")
	typeEnd := stmtEnd
	if eq >= 0 {
		typeEnd = eq
	}
	if start+2 >= typeEnd || !lowerableInterfaceVarTypeRange(toks, start+2, typeEnd) {
		return staticInterfaceAssertionVar{}, false
	}
	concrete := "nil"
	nilInterface := true
	if eq >= 0 {
		if eq+1 >= stmtEnd {
			return staticInterfaceAssertionVar{}, false
		}
		values := expressionRanges(toks, eq+1, stmtEnd)
		if len(values) != 1 {
			return staticInterfaceAssertionVar{}, false
		}
		value := values[0]
		if expressionContainsCallToken(toks, value.start, value.end) {
			return staticInterfaceAssertionVar{}, false
		}
		if nilInterfaceInitializer(toks, value.start, value.end) {
			concrete = "nil"
			nilInterface = true
		} else {
			concrete = staticInterfaceConcreteType(toks, value.start, value.end, structs)
			nilInterface = false
		}
		if concrete == "" {
			return staticInterfaceAssertionVar{}, false
		}
	}
	info := staticInterfaceAssertionVar{name: toks[start+1].Text, concreteType: concrete, nilInterface: nilInterface, declStart: start, declEnd: stmtEnd, bodyOpen: bodyOpen, bodyClose: bodyClose}
	if !staticInterfaceAssertionUsesLowerable(toks, info, structs) {
		return staticInterfaceAssertionVar{}, false
	}
	return info, true
}

func staticInterfaceAssertionUsesLowerable(toks []scan.Token, info staticInterfaceAssertionVar, structs []structFieldSet) bool {
	seenAssertion := false
	for i := info.declEnd; i < info.bodyClose; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != info.name {
			continue
		}
		if staticInterfaceTypeSwitchUseLowerable(toks, i, info, structs) {
			seenAssertion = true
			switchBody := findClose(toks, i+2, "(", ")") + 1
			close := findClose(toks, switchBody, "{", "}")
			if close < 0 {
				return false
			}
			i = close
			continue
		}
		if i+1 < info.bodyClose && startsTypeAssertion(toks, i+1) {
			close := findClose(toks, i+2, "(", ")")
			asserted := staticInterfaceAssertionTypeName(toks, i+1)
			if close < 0 || (asserted != info.concreteType && !staticInterfaceAssertionMismatchLowerable(toks, i, close, asserted, structs)) {
				return false
			}
			seenAssertion = true
			i = close
			continue
		}
		return false
	}
	return seenAssertion
}

func staticInterfaceTypeSwitchUseLowerable(toks []scan.Token, namePos int, info staticInterfaceAssertionVar, structs []structFieldSet) bool {
	if namePos < 1 || namePos+5 >= info.bodyClose || toks[namePos].Kind != scan.Ident || toks[namePos].Text != info.name {
		return false
	}
	for _, switchPos := range []int{namePos - 1, namePos - 3} {
		guard, ok := staticInterfaceTypeSwitchGuardAt(toks, switchPos)
		if ok && guard.sourcePos == namePos {
			return staticInterfaceTypeSwitchBodyLowerable(toks, switchPos, info, structs)
		}
	}
	return false
}

func staticInterfaceTypeSwitchBodyLowerable(toks []scan.Token, switchPos int, info staticInterfaceAssertionVar, structs []structFieldSet) bool {
	guard, ok := staticInterfaceTypeSwitchGuardAt(toks, switchPos)
	if !ok || guard.sourceName != info.name {
		return false
	}
	if guard.bindingName != "" && guard.bindingName == info.name {
		return false
	}
	open := guard.open
	close := findClose(toks, open, "{", "}")
	if close < 0 || close > info.bodyClose {
		return false
	}
	seenClause := false
	seenMatchingCase := false
	matchingStart := -1
	matchingEnd := -1
	defaultStart := -1
	defaultEnd := -1
	paren := 0
	brack := 0
	brace := 0
	for i := open + 1; i < close; i++ {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 {
			if text == "case" {
				colon := caseClauseColon(toks, i+1, close)
				if colon < 0 || !staticInterfaceTypeSwitchCaseListLowerable(toks, i+1, colon, structs) {
					return false
				}
				if staticInterfaceTypeSwitchCaseListContainsType(toks, i+1, colon, info.concreteType) {
					seenMatchingCase = true
					matchingStart = colon + 1
					matchingEnd = staticInterfaceTypeSwitchNextClause(toks, colon+1, close)
				}
				seenClause = true
				i = colon
				continue
			}
			if text == "default" {
				colon := caseClauseColon(toks, i+1, close)
				if colon < 0 {
					return false
				}
				defaultStart = colon + 1
				defaultEnd = staticInterfaceTypeSwitchNextClause(toks, colon+1, close)
				seenClause = true
				i = colon
				continue
			}
			if text == "break" || text == "fallthrough" {
				return false
			}
		}
		updateDepth(text, &paren, &brack, &brace)
	}
	if !seenClause {
		return false
	}
	if guard.bindingName == "" {
		return true
	}
	if seenMatchingCase {
		if info.nilInterface {
			return staticInterfaceTypeSwitchDefaultBindingLowerable(toks, open, matchingStart, matchingEnd, guard.bindingName)
		}
		return true
	}
	if defaultStart >= 0 {
		return staticInterfaceTypeSwitchDefaultBindingLowerable(toks, open, defaultStart, defaultEnd, guard.bindingName)
	}
	return true
}

func staticInterfaceTypeSwitchNextClause(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 && (text == "case" || text == "default") {
			return i
		}
		updateDepth(text, &paren, &brack, &brace)
	}
	return end
}

func staticInterfaceTypeSwitchDefaultBindingLowerable(toks []scan.Token, scopeOpen int, start int, end int, binding string) bool {
	if binding == "" {
		return true
	}
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != binding {
			continue
		}
		stmtStart := simpleStatementStart(toks, scopeOpen, i)
		stmtEnd := simpleStatementEnd(toks, stmtStart, end)
		if !blankDiscardOfIdentifierStatement(toks, stmtStart, stmtEnd, binding) {
			return false
		}
		i = stmtEnd - 1
	}
	return true
}

func staticInterfaceTypeSwitchCaseListLowerable(toks []scan.Token, start int, end int, structs []structFieldSet) bool {
	values := expressionRanges(toks, start, end)
	if len(values) == 0 {
		return false
	}
	for i := 0; i < len(values); i++ {
		value := values[i]
		value.start, value.end = trimExpressionRange(toks, value.start, value.end)
		typ := typeTextInRange(toks, value.start, value.end)
		if typ == "" {
			return false
		}
		if typ == "nil" {
			continue
		}
		if !staticInterfaceAssertionSupportedType(typ) && !staticInterfaceStructTypeSupported(typ, structs) {
			return false
		}
	}
	return true
}

func staticInterfaceStructTypeSupported(typ string, structs []structFieldSet) bool {
	for strings.HasPrefix(typ, "*") {
		typ = typ[1:]
	}
	return typ != "" && !strings.HasPrefix(typ, "[]") && structFieldSetIndex(structs, typ) >= 0
}

func staticInterfaceTypeSwitchCaseListContainsType(toks []scan.Token, start int, end int, concrete string) bool {
	values := expressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		value.start, value.end = trimExpressionRange(toks, value.start, value.end)
		if typeTextInRange(toks, value.start, value.end) == concrete {
			return true
		}
	}
	return false
}

func staticInterfaceTypeSwitchBindingType(toks []scan.Token, body int, switchPos int, structs []structFieldSet) (string, string, int, bool) {
	guard, ok := staticInterfaceTypeSwitchGuardAt(toks, switchPos)
	if !ok || guard.bindingName == "" || guard.bindingName == guard.sourceName {
		return "", "", 0, false
	}
	bodyClose := findClose(toks, body, "{", "}")
	if bodyClose < 0 {
		return "", "", 0, false
	}
	for i := body + 1; i < switchPos; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAt(toks, i, body, bodyClose, structs)
		if !ok {
			continue
		}
		if info.name == guard.sourceName && staticInterfaceTypeSwitchBodyLowerable(toks, switchPos, info, structs) {
			close := findClose(toks, guard.open, "{", "}")
			if close < 0 {
				return "", "", 0, false
			}
			return guard.bindingName, info.concreteType, close, true
		}
		i = info.declEnd - 1
	}
	return "", "", 0, false
}

func staticInterfaceAssertionTypeMatches(toks []scan.Token, dot int, concrete string) bool {
	return staticInterfaceAssertionTypeName(toks, dot) == concrete
}

func staticInterfaceAssertionTypeName(toks []scan.Token, dot int) string {
	if !startsTypeAssertion(toks, dot) {
		return ""
	}
	close := findClose(toks, dot+1, "(", ")")
	if close < 0 {
		return ""
	}
	start, end := trimExpressionRange(toks, dot+2, close)
	if start+1 == end && toks[start].Text == "type" {
		return ""
	}
	return typeTextInRange(toks, start, end)
}

func staticInterfaceAssertionSupportedType(typ string) bool {
	switch typ {
	case "int", "string", "bool":
		return true
	}
	return false
}

func staticInterfaceAssertionMismatchLowerable(toks []scan.Token, pos int, close int, asserted string, structs []structFieldSet) bool {
	if staticInterfaceAssertionCommaOKContext(toks, pos, close) {
		return staticInterfaceAssertionSupportedType(asserted) || staticInterfaceStructTypeSupported(asserted, structs)
	}
	if staticInterfaceAssertionPanicContext(toks, pos, close) {
		return staticInterfaceAssertionSupportedType(asserted)
	}
	return false
}

func staticInterfaceAssertionCommaOKContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 2 || len(rhs) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionPanicContext(toks []scan.Token, pos int, close int) bool {
	if staticInterfaceAssertionIfPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionForPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionSwitchPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionDeferPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionCallPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionReturnPanicContext(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionVarDeclPanicContext(toks, pos, close) {
		return true
	}
	if close < 0 {
		return false
	}
	stmtStart := sameLineAssignmentStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	shortDecl := assign >= 0
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	if start != pos || end != close+1 {
		return false
	}
	name := singleIdentifierExpression(toks, lhs[0].start, lhs[0].end)
	if name != "" {
		return true
	}
	lhsStart, lhsEnd := trimExpressionRange(toks, lhs[0].start, lhs[0].end)
	return !shortDecl && lhsStart+1 == lhsEnd && toks[lhsStart].Kind == scan.Ident && toks[lhsStart].Text == "_"
}

func staticInterfaceAssertionIfPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 || staticInterfaceAssertionTypeName(toks, pos+1) != "bool" {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "if" {
		return false
	}
	start, end, ok := staticInterfaceAssertionIfConditionRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionIfConditionRange(toks []scan.Token, ifPos int, limit int) (int, int, bool) {
	if ifPos < 0 || ifPos >= limit || toks[ifPos].Text != "if" {
		return 0, 0, false
	}
	open := controlBodyOpen(toks, ifPos+1, limit)
	if open < 0 {
		return 0, 0, false
	}
	if findTopLevelToken(toks, ifPos+1, open, ";") >= 0 {
		return 0, 0, false
	}
	start, end := trimExpressionRange(toks, ifPos+1, open)
	return start, end, start < end
}

func staticInterfaceAssertionForPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 || staticInterfaceAssertionTypeName(toks, pos+1) != "bool" {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "for" {
		return false
	}
	start, end, ok := staticInterfaceAssertionForConditionRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionForConditionRange(toks []scan.Token, forPos int, limit int) (int, int, bool) {
	if forPos < 0 || forPos >= limit || toks[forPos].Text != "for" {
		return 0, 0, false
	}
	open := controlBodyOpen(toks, forPos+1, limit)
	if open < 0 {
		return 0, 0, false
	}
	if findTopLevelToken(toks, forPos+1, open, ";") >= 0 {
		return 0, 0, false
	}
	start, end := trimExpressionRange(toks, forPos+1, open)
	if start < end && toks[start].Text == "range" {
		return 0, 0, false
	}
	return start, end, start < end
}

func staticInterfaceAssertionSwitchPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "switch" {
		return false
	}
	start, end, ok := staticInterfaceAssertionSwitchTagRange(toks, stmtStart, len(toks))
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionSwitchTagRange(toks []scan.Token, switchPos int, limit int) (int, int, bool) {
	if switchPos < 0 || switchPos >= limit || toks[switchPos].Text != "switch" {
		return 0, 0, false
	}
	open := controlBodyOpen(toks, switchPos+1, limit)
	if open < 0 {
		return 0, 0, false
	}
	if findTopLevelToken(toks, switchPos+1, open, ";") >= 0 {
		return 0, 0, false
	}
	start, end := trimExpressionRange(toks, switchPos+1, open)
	return start, end, start < end
}

func staticInterfaceAssertionDeferPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart+2 >= len(toks) || toks[stmtStart].Text != "defer" || toks[stmtStart+1].Kind != scan.Ident || toks[stmtStart+2].Text != "(" {
		return false
	}
	callClose := findClose(toks, stmtStart+2, "(", ")")
	if callClose < 0 {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if callClose != stmtEnd-1 {
		return false
	}
	args := expressionRanges(toks, stmtStart+3, callClose)
	if len(args) < 1 {
		return false
	}
	start, end := trimExpressionRange(toks, args[0].start, args[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionCallPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart+1 >= len(toks) || toks[stmtStart].Kind != scan.Ident || toks[stmtStart+1].Text != "(" {
		return false
	}
	callClose := findClose(toks, stmtStart+1, "(", ")")
	if callClose < 0 {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	if callClose != stmtEnd-1 {
		return false
	}
	args := expressionRanges(toks, stmtStart+2, callClose)
	if len(args) < 1 {
		return false
	}
	start, end := trimExpressionRange(toks, args[0].start, args[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionVarDeclPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart+1 >= len(toks) || toks[stmtStart].Text != "var" || toks[stmtStart+1].Kind != scan.Ident || toks[stmtStart+1].Text == "_" {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	eq := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if eq < 0 || eq <= stmtStart+1 {
		return false
	}
	if findTopLevelToken(toks, stmtStart+1, eq, ",") >= 0 {
		return false
	}
	rhs := expressionRanges(toks, eq+1, stmtEnd)
	if len(rhs) != 1 {
		return false
	}
	start, end := trimExpressionRange(toks, rhs[0].start, rhs[0].end)
	if start != pos || end != close+1 {
		return false
	}
	if stmtStart+2 >= eq {
		return true
	}
	declared := typeTextInRange(toks, stmtStart+2, eq)
	return declared == staticInterfaceAssertionTypeName(toks, pos+1)
}

func staticInterfaceAssertionReturnPanicContext(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStart(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "return" {
		return false
	}
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	values := expressionRanges(toks, stmtStart+1, stmtEnd)
	if len(values) < 1 {
		return false
	}
	start, end := trimExpressionRange(toks, values[0].start, values[0].end)
	return start == pos && end == close+1
}

func sameLineSimpleStatementStart(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}" {
			return i + 1
		}
	}
	return 0
}

func staticInterfaceAssertionAssignmentResultCount(file parse.File, toks []scan.Token, start int, end int) int {
	if staticInterfaceAssertionExpressionLowerable(file, toks, start, end) {
		return 2
	}
	return -1
}

func staticInterfaceAssertionExpressionLowerable(file parse.File, toks []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start+4 > end {
		return false
	}
	if toks[start].Kind != scan.Ident || toks[start+1].Text != "." {
		return false
	}
	close := findClose(toks, start+2, "(", ")")
	if close != end-1 {
		return false
	}
	return lowerableStaticInterfaceAssertionAt(file, toks, start+1)
}

func staticInterfaceConcreteType(toks []scan.Token, start int, end int, structs []structFieldSet) string {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticInterfaceConcreteType(toks, start+1, close, structs)
		}
	}
	if (toks[start].Text == "+" || toks[start].Text == "-") && start+1 < end {
		typ := staticInterfaceConcreteType(toks, start+1, end, structs)
		if typ == "int" {
			return typ
		}
		return ""
	}
	if typ := staticInterfaceCompositeConcreteType(toks, start, end, structs); typ != "" {
		return typ
	}
	if start+1 != end {
		return ""
	}
	tok := toks[start]
	if tok.Kind == scan.String {
		return "string"
	}
	if tok.Kind == scan.Char || tok.Kind == scan.Number {
		if numberLiteralType(tok.Text) == "int" {
			return "int"
		}
		return ""
	}
	if tok.Kind == scan.Ident && (tok.Text == "true" || tok.Text == "false") {
		return "bool"
	}
	return ""
}

func staticInterfaceCompositeConcreteType(toks []scan.Token, start int, end int, structs []structFieldSet) string {
	if start < end && toks[start].Text == "&" {
		typ := staticInterfaceCompositeConcreteType(toks, start+1, end, structs)
		if typ == "" {
			return ""
		}
		return "*" + typ
	}
	open := compositeLiteralOpen(toks, start, end)
	if open < 0 {
		return ""
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return ""
	}
	typ := typeTextInRange(toks, start, open)
	if typ == "" || structFieldSetIndex(structs, typ) < 0 {
		return ""
	}
	return typ
}

func nextSimpleStatementStart(toks []scan.Token, pos int, limit int) int {
	for pos < limit && toks[pos].Text == ";" {
		pos++
	}
	if pos >= limit || toks[pos].Text == "}" {
		return -1
	}
	return pos
}

func blankDiscardOfIdentifierStatement(toks []scan.Token, start int, end int, name string) bool {
	assign := findTopLevelToken(toks, start, end, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, start, assign)
	rhs := expressionRanges(toks, assign+1, end)
	return len(lhs) == 1 && len(rhs) == 1 && blankIdentifierExpression(toks, lhs[0].start, lhs[0].end) && singleIdentifierExpression(toks, rhs[0].start, rhs[0].end) == name
}

func identifierUsedOutsideRanges(toks []scan.Token, name string, scopeStart int, scopeEnd int, firstStart int, firstEnd int, secondStart int, secondEnd int) bool {
	for i := scopeStart; i < len(toks) && i < scopeEnd; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		if i >= firstStart && i < firstEnd {
			continue
		}
		if i >= secondStart && i < secondEnd {
			continue
		}
		return true
	}
	return false
}

func discardedComplexStatementContainingToken(toks []scan.Token, pos int, sigs []funcSignature, supported func([]scan.Token, int, int, []funcSignature) bool) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	stmtStart := sameLineStatementStart(toks, pos)
	stmtEnd := simpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || isCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := expressionRanges(toks, stmtStart, assign)
	rhs := expressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	matched := false
	for i := 0; i < len(lhs); i++ {
		if !blankIdentifierExpression(toks, lhs[i].start, lhs[i].end) {
			return false
		}
		if pos >= rhs[i].start && pos < rhs[i].end {
			matched = true
		}
		if !supported(toks, rhs[i].start, rhs[i].end, sigs) {
			return false
		}
	}
	return matched
}

func discardablePureComplexExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardablePureComplexExpression(toks, start+1, close, sigs)
		}
	}
	if _, _, ok := reducibleComplexLiteralParts(toks, start, end); ok {
		return true
	}
	if start+3 > end || toks[start].Text != "complex" || toks[start+1].Text != "(" {
		return false
	}
	if start > 0 && toks[start-1].Text == "." {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	args := expressionRanges(toks, start+2, close)
	if len(args) != 2 {
		return false
	}
	for i := 0; i < len(args); i++ {
		if !discardablePureRealNumberExpression(toks, args[i].start, args[i].end) {
			return false
		}
	}
	return true
}

func discardableLowerableComplexExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableComplexExpression(toks, start+1, close, sigs)
		}
	}
	if _, _, ok := reducibleComplexLiteralParts(toks, start, end); ok {
		return true
	}
	if discardableLowerableComplexBinaryExpression(toks, start, end, sigs) {
		return true
	}
	if start+3 > end || toks[start].Text != "complex" || toks[start+1].Text != "(" {
		return false
	}
	if start > 0 && toks[start-1].Text == "." {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	args := expressionRanges(toks, start+2, close)
	if len(args) != 2 {
		return false
	}
	for i := 0; i < len(args); i++ {
		if !discardableLowerableComplexComponentExpression(toks, args[i].start, args[i].end, sigs) {
			return false
		}
	}
	return true
}

func discardableLowerableComplexBinaryExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	op := topLevelBinaryOperator(toks, start, end)
	if op < 0 || (toks[op].Text != "+" && toks[op].Text != "-") {
		return false
	}
	return discardableLowerableComplexExpression(toks, start, op, sigs) && discardableLowerableComplexExpression(toks, op+1, end, sigs)
}

func discardableLowerableComplexComponentExpression(toks []scan.Token, start int, end int, sigs []funcSignature) bool {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardableLowerableComplexComponentExpression(toks, start+1, close, sigs)
		}
	}
	if discardablePureRealNumberExpression(toks, start, end) {
		return true
	}
	typ := lowerableDirectCallResultType(toks, start, end, sigs)
	return typ == "float64"
}

func lowerableDirectCallResultType(toks []scan.Token, start int, end int, sigs []funcSignature) string {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableDirectCallResultType(toks, start+1, close, sigs)
		}
	}
	if !directCallExpressionWithoutCallArgs(toks, start, end) {
		return ""
	}
	index := funcSignatureIndex(sigs, toks[start].Text)
	if index < 0 {
		return ""
	}
	sig := sigs[index]
	if sig.results != 1 || len(sig.resultTypes) != 1 {
		return ""
	}
	return sig.resultTypes[0]
}

func discardablePureRealNumberExpression(toks []scan.Token, start int, end int) bool {
	_, imaginary, ok := signedNumberLiteralText(toks, start, end)
	return ok && !imaginary
}

func appendReducibleComplexComponentCallDiagnostics(diags Diagnostics, file parse.File, tokens []scan.Token, pos int, localTypes []localValueType, structs []structFieldSet, typeNames []localValueType, importFuncs []importedFunction, importValues []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) Diagnostics {
	info, ok := reducibleComplexComponentCall(tokens, pos)
	if !ok {
		return diags
	}
	for i := 0; i < len(info.args); i++ {
		arg := info.args[i]
		typ := expressionSimpleTypeWithCallsAndTypes(tokens, arg.start, arg.end, localTypes, structs, typeNames, importFuncs, importValues, localStructs, sigs, importMethods)
		if typ != "" && !complexComponentOperandTypeAllowed(tokens, arg.start, arg.end, typ) {
			diags = appendDiag(diags, file, tokens[arg.start], "complex arguments must be floating-point values or numeric constants, got "+typ)
		}
	}
	return diags
}

func reducibleComplexComponentSimpleType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if typ := reducibleComplexLiteralComponentType(tokens, start, end, sigs); typ != "" {
		return typ
	}
	info, ok := reducibleComplexComponentCall(tokens, start)
	if !ok || info.outerClose != end-1 {
		return ""
	}
	if signedFunctionCallAt(tokens, start, sigs) || signedFunctionCallAt(tokens, info.complexPos, sigs) {
		return ""
	}
	arg := info.args[info.selectedArg]
	return expressionSimpleTypeWithCallsAndTypes(tokens, arg.start, arg.end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
}

func reducibleComplexComponentRawType(tokens []scan.Token, start int, end int, locals []localValueType, structs []structFieldSet, typeNames []localValueType, funcs []importedFunction, values []importedValue, localStructs []localStructType, sigs []funcSignature, importMethods []importedMethod) string {
	start, end = trimExpressionRange(tokens, start, end)
	if typ := reducibleComplexLiteralComponentType(tokens, start, end, sigs); typ != "" {
		return typ
	}
	info, ok := reducibleComplexComponentCall(tokens, start)
	if !ok || info.outerClose != end-1 {
		return ""
	}
	if signedFunctionCallAt(tokens, start, sigs) || signedFunctionCallAt(tokens, info.complexPos, sigs) {
		return ""
	}
	arg := info.args[info.selectedArg]
	return expressionRawTypeWithCallsAndTypes(tokens, arg.start, arg.end, locals, structs, typeNames, funcs, values, localStructs, sigs, importMethods)
}

func reducibleComplexLiteralComponentType(tokens []scan.Token, start int, end int, sigs []funcSignature) string {
	info, ok := reducibleComplexLiteralComponentCall(tokens, start)
	if !ok || info.outerClose != end-1 || signedFunctionCallAt(tokens, start, sigs) {
		return ""
	}
	if info.selectedArg == 0 {
		return numericComponentLiteralType(info.realText)
	}
	return numericComponentLiteralType(info.imagText)
}

func numericComponentLiteralType(text string) string {
	if strings.HasPrefix(text, "+") || strings.HasPrefix(text, "-") {
		text = text[1:]
	}
	return numberLiteralType(text)
}

func reducibleComplexLiteralParts(toks []scan.Token, start int, end int) (string, string, bool) {
	start, end = trimExpressionRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return reducibleComplexLiteralParts(toks, start+1, close)
		}
	}
	text, imaginary, ok := signedNumberLiteralText(toks, start, end)
	if ok {
		if !imaginary {
			return "", "", false
		}
		return "0", stripImaginaryLiteralSuffix(text), true
	}
	op := topLevelBinaryOperator(toks, start, end)
	if op < 0 || (toks[op].Text != "+" && toks[op].Text != "-") {
		return "", "", false
	}
	leftText, leftImaginary, leftOK := signedNumberLiteralText(toks, start, op)
	rightText, rightImaginary, rightOK := signedNumberLiteralText(toks, op+1, end)
	if !leftOK || !rightOK || leftImaginary == rightImaginary {
		return "", "", false
	}
	if toks[op].Text == "-" {
		rightText = negateSignedNumberLiteralText(rightText)
	}
	if leftImaginary {
		return rightText, stripImaginaryLiteralSuffix(leftText), true
	}
	return leftText, stripImaginaryLiteralSuffix(rightText), true
}

func signedNumberLiteralText(toks []scan.Token, start int, end int) (string, bool, bool) {
	start, end = trimExpressionRange(toks, start, end)
	sign := ""
	if start < end && (toks[start].Text == "+" || toks[start].Text == "-") {
		sign = toks[start].Text
		start++
	}
	if start+1 != end || toks[start].Kind != scan.Number {
		return "", false, false
	}
	text := toks[start].Text
	if sign == "-" {
		text = "-" + text
	}
	return text, isImaginaryLiteral(toks[start].Text), true
}

func stripImaginaryLiteralSuffix(text string) string {
	if strings.HasSuffix(text, "i") {
		return text[:len(text)-1]
	}
	return text
}

func negateSignedNumberLiteralText(text string) string {
	if strings.HasPrefix(text, "-") {
		return text[1:]
	}
	return "-" + text
}

func complexComponentOperandTypeAllowed(tokens []scan.Token, start int, end int, typ string) bool {
	if typ == "float32" || typ == "float64" {
		return true
	}
	if complexComponentOperandIsNumericConstant(tokens, start, end) {
		return isNumericTypeName(typ)
	}
	return false
}

func complexComponentOperandIsNumericConstant(tokens []scan.Token, start int, end int) bool {
	start, end = trimExpressionRange(tokens, start, end)
	if start >= end {
		return false
	}
	if tokens[start].Text == "(" {
		close := findClose(tokens, start, "(", ")")
		if close == end-1 {
			return complexComponentOperandIsNumericConstant(tokens, start+1, close)
		}
	}
	if start+1 == end {
		return tokens[start].Kind == scan.Number || tokens[start].Kind == scan.Char
	}
	if start+2 == end && (tokens[start].Text == "+" || tokens[start].Text == "-") {
		return tokens[start+1].Kind == scan.Number || tokens[start+1].Kind == scan.Char
	}
	return false
}

func signedFunctionCallAt(toks []scan.Token, i int, sigs []funcSignature) bool {
	if i < 0 || i+1 >= len(toks) || toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
		return false
	}
	return funcSignatureIndex(sigs, toks[i].Text) >= 0
}

func startsSupportedNewCall(toks []scan.Token, i int, structs []structFieldSet, typeNames []localValueType) bool {
	if i+1 >= len(toks) || toks[i].Text != "new" || toks[i+1].Text != "(" {
		return false
	}
	close := findClose(toks, i+1, "(", ")")
	if close < 0 {
		return false
	}
	target := typeTextInRange(toks, i+2, close)
	return supportedNewTypeText(target, structs, typeNames)
}

func supportedNewTypeText(target string, structs []structFieldSet, typeNames []localValueType) bool {
	if target == "" {
		return false
	}
	if isSupportedNewBuiltinTypeName(target) {
		return true
	}
	target = normalizeNamedType(target, typeNames)
	if isSupportedNewBuiltinTypeName(target) {
		return true
	}
	if structFieldSetIndex(structs, target) >= 0 {
		return true
	}
	if strings.HasPrefix(target, "*") {
		return supportedNewTypeText(target[1:], structs, typeNames)
	}
	if strings.HasPrefix(target, "[]") {
		return supportedNewTypeText(target[2:], structs, typeNames)
	}
	return false
}

func isSupportedNewBuiltinTypeName(name string) bool {
	switch name {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32":
		return true
	}
	return false
}

func isRuntimeOSIntrinsicCall(file parse.File, toks []scan.Token, i int) bool {
	if file.PackageName != "os" {
		return false
	}
	if file.Path != "rtg/std/os/os_rtg.go" && file.Path != "rtg\\std\\os\\os_rtg.go" && !strings.HasSuffix(file.Path, "/rtg/std/os/os_rtg.go") && !strings.HasSuffix(file.Path, "\\rtg\\std\\os\\os_rtg.go") {
		return false
	}
	if i+1 >= len(toks) {
		return false
	}
	if toks[i].Kind != scan.Ident {
		return false
	}
	if toks[i+1].Text != "(" {
		return false
	}
	switch toks[i].Text {
	case "open", "close", "read", "write", "chmod":
		return true
	}
	return false
}

func breakAllowedAt(toks []scan.Token, pos int) bool {
	return hasControlOwner(toks, pos, true)
}

func labeledBreakAllowedAt(toks []scan.Token, pos int) bool {
	_, owner, ok := labeledControlTarget(toks, pos)
	return ok && (owner == "for" || owner == "switch")
}

func hasSameLineLabelOperand(toks []scan.Token, pos int) bool {
	if pos+1 >= len(toks) {
		return false
	}
	if toks[pos+1].Kind != scan.Ident {
		return false
	}
	return toks[pos+1].Line == toks[pos].Line
}

func continueAllowedAt(toks []scan.Token, pos int) bool {
	return hasControlOwner(toks, pos, false)
}

func labeledContinueAllowedAt(toks []scan.Token, pos int) bool {
	_, owner, ok := labeledControlTarget(toks, pos)
	return ok && owner == "for"
}

func hasControlOwner(toks []scan.Token, pos int, allowSwitch bool) bool {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		text := toks[i].Text
		if text == "}" {
			depth++
			continue
		}
		if text != "{" {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		owner := blockOwnerKeyword(toks, i)
		if owner == "for" {
			return true
		}
		if allowSwitch && owner == "switch" {
			return true
		}
	}
	return false
}

func labeledControlTarget(toks []scan.Token, pos int) (int, string, bool) {
	if !hasSameLineLabelOperand(toks, pos) {
		return -1, "", false
	}
	label := toks[pos+1].Text
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != label || toks[i+1].Text != ":" {
			continue
		}
		target := nextNonSemicolonToken(toks, i+2)
		if target < 0 {
			continue
		}
		owner := toks[target].Text
		if owner != "for" && owner != "switch" {
			continue
		}
		open, close := labeledControlTargetBody(toks, target, pos)
		if open >= 0 && close > pos {
			return close, owner, true
		}
	}
	return -1, "", false
}

func nextNonSemicolonToken(toks []scan.Token, start int) int {
	for i := start; i < len(toks); i++ {
		if toks[i].Text == ";" {
			continue
		}
		return i
	}
	return -1
}

func labeledControlTargetBody(toks []scan.Token, target int, pos int) (int, int) {
	owner := toks[target].Text
	for i := target + 1; i < len(toks) && i <= pos; i++ {
		if toks[i].Text != "{" {
			continue
		}
		close := findClose(toks, i, "{", "}")
		if close <= pos {
			continue
		}
		if blockOwnerKeyword(toks, i) == owner {
			return i, close
		}
	}
	return -1, -1
}

func blockOwnerKeyword(toks []scan.Token, open int) string {
	paren := 0
	brack := 0
	for i := open - 1; i >= 0; i-- {
		text := toks[i].Text
		if text == ")" {
			paren++
			continue
		}
		if text == "]" {
			brack++
			continue
		}
		if text == "(" {
			if paren > 0 {
				paren--
				continue
			}
		}
		if text == "[" {
			if brack > 0 {
				brack--
				continue
			}
		}
		if paren != 0 || brack != 0 {
			continue
		}
		if text == "for" || text == "switch" || text == "if" || text == "else" || text == "func" || text == "select" {
			return text
		}
		if text == "{" || text == "}" {
			return ""
		}
	}
	return ""
}

func fallthroughTargetCaseColon(toks []scan.Token, pos int) int {
	open := enclosingSwitchBodyOpen(toks, pos)
	if open < 0 {
		return -1
	}
	close := findClose(toks, open, "{", "}")
	if close < 0 {
		return -1
	}
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < close; i++ {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 {
			if text == "case" || text == "default" {
				return caseClauseColon(toks, i+1, close)
			}
			if text != ";" {
				return -1
			}
		}
		updateDepth(text, &paren, &brack, &brace)
	}
	return -1
}

func enclosingSwitchBodyOpen(toks []scan.Token, pos int) int {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		text := toks[i].Text
		if text == "}" {
			depth++
			continue
		}
		if text != "{" {
			continue
		}
		if depth > 0 {
			depth--
			continue
		}
		if blockOwnerKeyword(toks, i) == "switch" {
			return i
		}
	}
	return -1
}

func findClose(toks []scan.Token, pos int, open string, close string) int {
	depth := 0
	for pos < len(toks) {
		if toks[pos].Text == open {
			depth++
		} else if toks[pos].Text == close {
			depth--
			if depth == 0 {
				return pos
			}
		}
		pos++
	}
	return -1
}

func findOpen(toks []scan.Token, pos int, open string, close string) int {
	depth := 0
	for pos >= 0 {
		if toks[pos].Text == close {
			depth++
		} else if toks[pos].Text == open {
			depth--
			if depth == 0 {
				return pos
			}
		}
		pos--
	}
	return -1
}

func findTopLevelToken(toks []scan.Token, start int, end int, text string) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == text {
			return i
		}
		updateDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func updateDepth(text string, paren *int, brack *int, brace *int) {
	switch text {
	case "(":
		*paren = *paren + 1
	case ")":
		*paren = *paren - 1
	case "[":
		*brack = *brack + 1
	case "]":
		*brack = *brack - 1
	case "{":
		*brace = *brace + 1
	case "}":
		*brace = *brace - 1
	}
}

func precededByTypeContext(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return false
	}
	prev := toks[pos-1]
	switch prev.Text {
	case "var", "type", "*":
		return true
	case "]":
		return toks[pos].Text == "*"
	}
	if prev.Kind == scan.Ident && pos >= 2 {
		if isKeyword(prev.Text) {
			return false
		}
		beforeName := toks[pos-2].Text
		if beforeName == "var" || beforeName == "type" {
			return true
		}
		if beforeName == "{" {
			return nameInStructFieldList(toks, pos-1)
		}
		if beforeName == ";" {
			return nameInStructFieldList(toks, pos-1)
		}
		if beforeName == "(" || beforeName == "," {
			namePos := pos - 1
			return nameInFunctionSignature(toks, namePos) || nameInStructFieldList(toks, namePos)
		}
	}
	return false
}

func nameInFunctionSignature(toks []scan.Token, namePos int) bool {
	open := containingOpen(toks, namePos, "(", ")")
	if open < 0 {
		return false
	}
	if open > 0 && toks[open-1].Text == "func" {
		return true
	}
	return open > 1 && toks[open-2].Text == "func" && toks[open-1].Kind == scan.Ident
}

func nameInStructFieldList(toks []scan.Token, namePos int) bool {
	open := containingOpen(toks, namePos, "{", "}")
	return open > 0 && toks[open-1].Text == "struct"
}

func containingOpen(toks []scan.Token, pos int, openText string, closeText string) int {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == closeText {
			depth++
			continue
		}
		if toks[i].Text == openText {
			if depth == 0 {
				return i
			}
			depth--
		}
	}
	return -1
}

func isKeyword(text string) bool {
	switch text {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return true
	}
	return false
}

func localImportShadows(file parse.File, importNames []string) []localShadow {
	var shadows []localShadow
	if len(importNames) == 0 {
		return shadows
	}
	var decls []parse.Decl
	decls = file.Decls
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i < len(decls); i++ {
		var decl parse.Decl
		decl = decls[i]
		if decl.Kind != "func" {
			continue
		}
		start := tokenIndexAt(tokens, decl.Start)
		if start < 0 {
			continue
		}
		body := findTokenText(tokens, start, decl.End, "{")
		if body < 0 {
			continue
		}
		shadows = collectFuncSignatureImportShadows(tokens, start, body, importNames, shadows)
		for i := body + 1; i < len(tokens); i++ {
			var tok scan.Token
			tok = tokens[i]
			if int(tok.Start) >= decl.End {
				break
			}
			if tok.Text == ":=" {
				shadows = collectShortDeclImportShadows(tokens, body, i, decl.End, importNames, shadows)
			}
			if tok.Text == "var" {
				shadows = collectVarImportShadows(tokens, body, i, decl.End, importNames, shadows)
			}
		}
	}
	return shadows
}

func collectFuncSignatureImportShadows(toks []scan.Token, start int, end int, names []string, shadows []localShadow) []localShadow {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		shadows = collectParameterImportShadows(toks, i+1, close, names, shadows)
		i = close
	}
	return shadows
}

func collectParameterImportShadows(toks []scan.Token, start int, end int, names []string, shadows []localShadow) []localShadow {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || !containsString(names, toks[i].Text) {
			continue
		}
		if i > start && toks[i-1].Text != "," {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			shadows = addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			shadows = addLocalShadow(shadows, toks[i].Text, 0, maxSourcePosition())
		}
	}
	return shadows
}

func collectShortDeclImportShadows(toks []scan.Token, body int, assign int, declEnd int, names []string, shadows []localShadow) []localShadow {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line || isStatementBoundary(toks[i].Text) {
			return shadows
		}
		if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (i == 0 || toks[i-1].Text != ".") {
			shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
	return shadows
}

func collectVarImportShadows(toks []scan.Token, body int, pos int, end int, names []string, shadows []localShadow) []localShadow {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && int(toks[i].Start) < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return shadows
			}
			if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line) {
				shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
			}
		}
		return shadows
	}
	line := toks[pos].Line
	for i := pos + 1; i < len(toks) && int(toks[i].Start) < end && toks[i].Line == line; i++ {
		if toks[i].Text == ")" || toks[i].Text == "}" || toks[i].Text == ":=" || toks[i].Text == "=" {
			return shadows
		}
		if toks[i].Kind == scan.Ident && containsString(names, toks[i].Text) && (i == pos+1 || toks[i-1].Text == ",") {
			shadows = addLocalShadow(shadows, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
	return shadows
}

func localScopeEnd(toks []scan.Token, body int, pos int, fallback int) int {
	var opens []int
	for i := body; i <= pos && i < len(toks); i++ {
		if toks[i].Text == "{" {
			opens = append(opens, i)
		} else if toks[i].Text == "}" && len(opens) > 0 {
			opens = opens[:len(opens)-1]
		}
	}
	if len(opens) == 0 {
		return fallback
	}
	close := findClose(toks, opens[len(opens)-1], "{", "}")
	if close < 0 {
		return fallback
	}
	return int(toks[close].Start)
}

func addLocalShadow(shadows []localShadow, name string, start int, end int) []localShadow {
	return append(shadows, localShadow{name: name, start: start, end: end})
}

func isLocalShadowAt(shadows []localShadow, name string, pos int) bool {
	for i := 0; i < len(shadows); i++ {
		shadow := shadows[i]
		if shadow.name == name && pos >= shadow.start && pos < shadow.end {
			return true
		}
	}
	return false
}

func findTokenText(toks []scan.Token, start int, end int, text string) int {
	for i := start; i < len(toks) && int(toks[i].Start) < end; i++ {
		if toks[i].Text == text {
			return i
		}
	}
	return -1
}

func isTypeStart(tok scan.Token) bool {
	return tok.Kind == scan.Ident || tok.Text == "*" || tok.Text == "[" || tok.Text == "..." || tok.Text == "struct"
}

func isTypeStartAfterName(toks []scan.Token, pos int, end int) bool {
	if pos+1 >= end {
		return false
	}
	if toks[pos+1].Text == "," {
		return isTypeStartAfterName(toks, pos+2, end)
	}
	return isTypeStart(toks[pos+1])
}

func isStatementBoundary(text string) bool {
	return text == "{" || text == "}" || text == ";" || text == "if" || text == "for" || text == "switch"
}

func maxSourcePosition() int {
	return 2147483647
}

func namedResultToken(file parse.File, decl parse.Decl) (scan.Token, bool) {
	var tokens []scan.Token
	tokens = file.Tokens
	name := tokenIndexAt(tokens, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(tokens) {
		return scan.Token{}, false
	}
	var openTok scan.Token
	openTok = tokens[name+1]
	if openTok.Text != "(" {
		return scan.Token{}, false
	}
	paramsClose := findClose(tokens, name+1, "(", ")")
	if paramsClose < 0 || paramsClose+1 >= len(tokens) {
		return scan.Token{}, false
	}
	var resultOpenTok scan.Token
	resultOpenTok = tokens[paramsClose+1]
	if resultOpenTok.Text != "(" {
		return scan.Token{}, false
	}
	resultsOpen := paramsClose + 1
	resultsClose := findClose(tokens, resultsOpen, "(", ")")
	if resultsClose < 0 {
		return scan.Token{}, false
	}
	var closeTok scan.Token
	closeTok = tokens[resultsClose]
	if int(closeTok.Start) >= decl.End {
		return scan.Token{}, false
	}
	for i := resultsOpen + 1; i < resultsClose; i++ {
		var tok scan.Token
		tok = tokens[i]
		if tok.Kind == scan.Ident && isTypeStartAfterName(tokens, i, resultsClose) {
			return tok, true
		}
	}
	return scan.Token{}, false
}

func hasOrdinaryMainSignature(file parse.File, decl parse.Decl) bool {
	var tokens []scan.Token
	tokens = file.Tokens
	name := tokenIndexAt(tokens, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(tokens) {
		return false
	}
	var openTok scan.Token
	openTok = tokens[name+1]
	if openTok.Text != "(" {
		return false
	}
	open := name + 1
	close := findClose(tokens, open, "(", ")")
	if close != open+1 {
		return false
	}
	for i := close + 1; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if int(tok.Start) >= decl.End {
			break
		}
		return tok.Text == "{"
	}
	return false
}

func tokenIndexAt(toks []scan.Token, start int) int {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if int(tok.Start) == start {
			return i
		}
	}
	return -1
}

func tokenIndexBefore(toks []scan.Token, end int) int {
	for i := 0; i < len(toks); i++ {
		if int(toks[i].Start) >= end {
			return i - 1
		}
	}
	return len(toks) - 1
}

func isCompositeKey(toks []scan.Token, pos int) bool {
	return pos+1 < len(toks) && toks[pos+1].Text == ":"
}

func declToken(file parse.File, decl parse.Decl, text string) (scan.Token, bool) {
	var tokens []scan.Token
	tokens = file.Tokens
	for i := 0; i < len(tokens); i++ {
		var tok scan.Token
		tok = tokens[i]
		if int(tok.Start) < decl.Start {
			continue
		}
		if int(tok.Start) >= decl.End {
			break
		}
		if tok.Text == text {
			return tok, true
		}
	}
	return scan.Token{}, false
}

func diag(file parse.File, tok scan.Token, message string) Diagnostic {
	return Diagnostic{Path: file.Path, Line: int(tok.Line), Column: int(tok.Column), Message: message}
}

func declDiagnostic(file parse.File, decl parse.Decl, message string) Diagnostic {
	tok := decl.NameTok
	if tok.Text == "" {
		tok = decl.Tok
	}
	return diag(file, tok, message)
}

func declNameDiagnostic(file parse.File, decl parse.Decl, index int, message string) Diagnostic {
	var nameToks []scan.Token
	nameToks = decl.NameToks
	if index >= 0 && index < len(nameToks) {
		var tok scan.Token
		tok = nameToks[index]
		return diag(file, tok, message)
	}
	return declDiagnostic(file, decl, message)
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
