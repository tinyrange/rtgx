package lower

import (
	"fmt"
	"strconv"
	"strings"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/scan"
	"j5.nz/rtg/rtg/target"
	"j5.nz/rtg/rtg/unit"
)

type localTypeInfo struct {
	qualifier string
	name      string
	pointer   bool
	embedded  bool
}

type methodInfo struct {
	name            string
	receiverType    string
	pointerReceiver bool
	unitName        string
	importPath      string
}

type functionAliasInfo struct {
	name         string
	unitName     string
	symbol       unit.Symbol
	receiver     string
	receiverType string
	capture      string
	captures     []functionLiteralCapture
	start        int
	end          int
	declStart    int
	declEnd      int
	decl         unit.Decl
	hasDecl      bool
	arrayParams  []arrayTypeLowerInfo
}

type functionLiteralForLower struct {
	start       int
	paramsOpen  int
	paramsClose int
	bodyOpen    int
	bodyClose   int
	end         int
	callOpen    int
	callClose   int
}

type functionLiteralCapture struct {
	name    string
	param   string
	typ     string
	pointer bool
}

type staticCallbackParamForLower struct {
	index int
	name  string
}

type staticCallbackTargetForLower struct {
	param    staticCallbackParamForLower
	unitName string
	symbol   unit.Symbol
	captures []functionLiteralCapture
	decl     unit.Decl
	hasDecl  bool
}

type staticCallbackTemplateForLower struct {
	name           string
	unitName       string
	importPath     string
	packageSymbols symbolNameTable
	params         []staticCallbackParamForLower
	body           string
	generatedDecls []unit.Decl
	refs           []unit.Symbol
	helpersEmitted bool
}

type staticCallbackSpecForLower struct {
	key      string
	unitName string
}

type staticCallbackSetForLower struct {
	templates []staticCallbackTemplateForLower
	specs     []staticCallbackSpecForLower
}

type symbolName struct {
	name     string
	unitName string
}

type methodEntry struct {
	lookup string
	info   methodInfo
}

type importSymbolGroup struct {
	localName       string
	importPath      string
	symbols         []unit.Symbol
	functionSymbols []unit.Symbol
}

type localNameRange struct {
	name  string
	start int
	end   int
}

type sizeofLocalType struct {
	name  string
	typ   string
	start int
	end   int
}

type sizeofNamedType struct {
	name  string
	typ   string
	start int
	end   int
}

type localTypeEntry struct {
	name string
	info localTypeInfo
}

type structFieldTypeEntry struct {
	owner string
	field string
	info  localTypeInfo
}

type structOwnerTable []string

type arrayStructFieldLowerInfo struct {
	owner string
	field string
	info  arrayTypeLowerInfo
}

type arrayStructFieldLowerInfoTable []arrayStructFieldLowerInfo

type namedSliceInfo struct {
	name     string
	elem     string
	elemInfo localTypeInfo
}

type namedArrayInfo struct {
	name string
	info arrayTypeLowerInfo
}

type namedMapInfo struct {
	name      string
	keyType   string
	valueType string
}

type localTypeDeclInfo struct {
	name       string
	unitName   string
	path       string
	declStart  int
	declEnd    int
	nameStart  int
	typeStart  int
	typeEnd    int
	scopeStart int
	scopeEnd   int
	anonymous  bool
	typeKey    string
	skipEmit   bool
	body       string
}

type interfaceParamEraseInfo struct {
	name     string
	unitName string
	indexes  []int
}

type interfaceParamEraseTable []interfaceParamEraseInfo

type interfaceReturnEraseInfo struct {
	name     string
	unitName string
}

type interfaceReturnEraseTable []interfaceReturnEraseInfo

type initFunctionInfo struct {
	path     string
	start    int
	name     string
	unitName string
}

type symbolNameTable []symbolName

type methodTable []methodEntry

type importSymbolTable []importSymbolGroup

type localNameTable []localNameRange

type sizeofLocalTypeTable []sizeofLocalType

type sizeofNamedTypeTable []sizeofNamedType

type localTypeTable []localTypeEntry

type structFieldTypeTable []structFieldTypeEntry

func symbolNameTableUnitName(table symbolNameTable, name string) string {
	index, ok := symbolNameTableSearch(table, name)
	if ok {
		return table[index].unitName
	}
	return ""
}

func symbolNameTableSet(table symbolNameTable, name string, unitName string) symbolNameTable {
	index, ok := symbolNameTableSearch(table, name)
	if ok {
		table[index].unitName = unitName
		return table
	}
	table = append(table, symbolName{})
	for i := len(table) - 1; i > index; i-- {
		table[i] = table[i-1]
	}
	table[index] = symbolName{name: name, unitName: unitName}
	return table
}

func symbolNameTableSearch(table symbolNameTable, name string) (int, bool) {
	low := 0
	high := len(table)
	for low < high {
		mid := (low + high) / 2
		entryName := table[mid].name
		if entryName == name {
			return mid, true
		}
		if stringGreater(entryName, name) {
			high = mid
		} else {
			low = mid + 1
		}
	}
	return low, false
}

func methodTableLookup(table methodTable, name string) methodInfo {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.lookup == name {
			info := entry.info
			return info
		}
	}
	return methodInfo{}
}

func methodTableSet(table methodTable, name string, info methodInfo) methodTable {
	for i := 0; i < len(table); i++ {
		if table[i].lookup == name {
			table[i].info = info
			return table
		}
	}
	return append(table, methodEntry{lookup: name, info: info})
}

func importSymbolTableGroup(table importSymbolTable, localName string) (importSymbolGroup, bool) {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.localName == localName {
			return entry, true
		}
	}
	return importSymbolGroup{}, false
}

func symbolByName(symbols []unit.Symbol, name string) (unit.Symbol, bool) {
	for i := 0; i < len(symbols); i++ {
		sym := symbols[i]
		if sym.Name == name {
			return sym, true
		}
	}
	return unit.Symbol{}, false
}

func importSymbolByName(group importSymbolGroup, name string) (unit.Symbol, bool) {
	sym, ok := symbolByName(group.symbols, name)
	if ok {
		return sym, true
	}
	if group.importPath != "" && isExported(name) {
		importPath := arena.PersistString(group.importPath)
		symbolName := arena.PersistString(name)
		unitName := SymbolName(importPath, name)
		unitName = arena.PersistString(unitName)
		return unit.Symbol{ImportPath: importPath, Name: symbolName, UnitName: unitName}, true
	}
	return unit.Symbol{}, false
}

func importFunctionSymbolByName(group importSymbolGroup, name string) (unit.Symbol, bool) {
	return symbolByName(group.functionSymbols, name)
}

func dotImportSymbol(table importSymbolTable, name string) (unit.Symbol, bool) {
	group, ok := importSymbolTableGroup(table, ".")
	if !ok {
		return unit.Symbol{}, false
	}
	return symbolByName(group.symbols, name)
}

func setSymbol(symbols []unit.Symbol, sym unit.Symbol) []unit.Symbol {
	for i := 0; i < len(symbols); i++ {
		if symbols[i].Name == sym.Name {
			symbols[i] = sym
			return symbols
		}
	}
	return append(symbols, sym)
}

func localTypeTableLookup(table localTypeTable, name string) localTypeInfo {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.name == name {
			info := entry.info
			return info
		}
	}
	return localTypeInfo{}
}

func localTypeTableSet(table localTypeTable, name string, info localTypeInfo) localTypeTable {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			table[i].info = info
			return table
		}
	}
	return append(table, localTypeEntry{name: name, info: info})
}

func Package(pkg load.Package) (unit.Unit, error) {
	return PackageWithGraph(pkg, nil)
}

func packageDeclCapacity(files []load.File) int {
	size := len(files) * 4
	for i := 0; i < len(files); i++ {
		size += topLevelDeclKeywordCount(files[i].Source)
	}
	if size < 16 {
		size = 16
	}
	return size
}

func packageSymbolCapacity(files []load.File, importCount int) int {
	size := importCount*8 + 16
	for i := 0; i < len(files); i++ {
		size += len(files[i].Source)/8 + 1
	}
	if size < 32 {
		size = 32
	}
	return size
}

func packageReferenceCapacity(files []load.File, importCount int) int {
	size := importCount*8 + 16
	for i := 0; i < len(files); i++ {
		size += sourceSelectorReferenceCapacity(files[i].Source)
	}
	if size < 32 {
		size = 32
	}
	return size
}

func topLevelDeclKeywordCount(src []byte) int {
	count := 0
	i := 0
	lineStart := true
	for i < len(src) {
		if lineStart {
			for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
				i++
			}
			if i >= len(src) {
				break
			}
			if hasTopLevelDeclKeywordAt(src, i, "func") || hasTopLevelDeclKeywordAt(src, i, "const") || hasTopLevelDeclKeywordAt(src, i, "var") || hasTopLevelDeclKeywordAt(src, i, "type") {
				count++
			}
			lineStart = false
		}
		if src[i] == '\n' {
			lineStart = true
		}
		i++
	}
	return count
}

func hasTopLevelDeclKeywordAt(src []byte, start int, word string) bool {
	if start+len(word) > len(src) {
		return false
	}
	for i := 0; i < len(word); i++ {
		if src[start+i] != word[i] {
			return false
		}
	}
	after := start + len(word)
	if after >= len(src) {
		return true
	}
	return src[after] == ' ' || src[after] == '\t' || src[after] == '('
}

func sourceSelectorReferenceCapacity(src []byte) int {
	count := 0
	for i := 1; i+1 < len(src); i++ {
		if src[i] == '.' && isIdentByte(src[i-1]) && isIdentByte(src[i+1]) {
			count += 2
		}
	}
	return count
}

func isIdentByte(ch byte) bool {
	return ch == '_' || (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func PackageWithGraph(pkg load.Package, graph *load.Graph) (unit.Unit, error) {
	u := unit.Unit{ImportPath: pkg.ImportPath, Package: pkg.Name, Entry: pkg.Entry}
	u.Imports = appendStrings(u.Imports, pkg.Imports)
	files := snapshotLoadFiles(pkg.Files)
	sortFilesByPath(files)
	targetName := target.Default()
	var depPackages []load.Package
	if graph != nil {
		if graph.Target != "" {
			targetName = graph.Target
		}
		depPackages = make([]load.Package, 0, len(graph.Packages))
		for depIndex := 0; depIndex < len(graph.Packages); depIndex++ {
			dep := graph.Packages[depIndex]
			dep.Files = snapshotLoadFiles(dep.Files)
			sortFilesByPath(dep.Files)
			depPackages = append(depPackages, dep)
		}
	}
	declCap := packageDeclCapacity(files)
	symbolCap := packageSymbolCapacity(files, len(pkg.Imports))
	refCap := packageReferenceCapacity(files, len(pkg.Imports))
	u.Exports = make([]unit.Symbol, 0, symbolCap)
	u.References = make([]unit.Symbol, 0, refCap)
	u.Decls = make([]unit.Decl, 0, declCap)
	topNames := make(symbolNameTable, 0, symbolCap)
	topFunctionNames := make(symbolNameTable, 0, symbolCap)
	topNameOrder := make([]string, 0, symbolCap)
	methods := make(methodTable, 0, declCap)
	methodOrder := make([]string, 0, declCap)
	hasOrdinaryMainDecl := false
	initFuncs := packageInitFunctionInfos(files, pkg.ImportPath)
	hasPackageInitializerCalls := packageHasCallableVarInitializers(files)
	needsPackageInit := packageNeedsInit(pkg, depPackages, initFuncs, hasPackageInitializerCalls)
	needsPanicRecover := packageNeedsPanicRecover(pkg, depPackages, nil)
	panicRecover := panicRecoverNames{}
	if needsPanicRecover {
		panicRecover = panicRecoverNamesForPackage(pkg.ImportPath)
		panicRecover.bridges = panicRecoverBridgesForPackage(pkg, depPackages)
	}
	functionTypeNames := functionTypeNamesForLoadFiles(files)
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		file := files[fileIndex]
		parsed, err := parsedLoadFile(file)
		if err != nil {
			arena.Reset(mark)
			return unit.Unit{}, err
		}
		if parsed.PackageName != pkg.Name {
			arena.Reset(mark)
			return unit.Unit{}, fmt.Errorf("%s: package name %s does not match loaded package %s", file.Path, parsed.PackageName, pkg.Name)
		}
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := decls[declIndex]
			if topLevelFunctionTypeDeclForLower(&parsed, decl, functionTypeNames) || topLevelFunctionContainingTypeDeclForLower(&parsed, decl) || topLevelInterfaceTypeDeclForLower(&parsed, decl) || topLevelInterfaceContainingTypeDeclForLower(&parsed, decl) || topLevelMapTypeDeclForLower(&parsed, decl) || topLevelMapContainingTypeDeclForLower(&parsed, decl) || topLevelArrayTypeDeclForLower(&parsed, decl) || topLevelAnyTypeDeclForLower(&parsed, decl) || topLevelComplexTypeDeclForLower(&parsed, decl) || topLevelComplexContainingTypeDeclForLower(&parsed, decl) {
				continue
			}
			if isOrdinaryMainDecl(parsed, decl) {
				hasOrdinaryMainDecl = true
			}
			names := declTopNames(&parsed, &decl)
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := arena.PersistString(names[nameIndex])
				if name != "" && name != "_" {
					unitName := arena.PersistString(SymbolName(pkg.ImportPath, name))
					if symbolNameTableUnitName(topNames, name) == "" {
						topNameOrder = append(topNameOrder, name)
					}
					topNames = symbolNameTableSet(topNames, name, unitName)
					if decl.Kind == "func" && !decl.Receiver {
						topFunctionNames = symbolNameTableSet(topFunctionNames, name, unitName)
					}
				}
			}
		}
		arena.Reset(mark)
	}
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			return unit.Unit{}, err
		}
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := decls[declIndex]
			if decl.Kind != "func" || !decl.Receiver {
				continue
			}
			info := methodDeclInfo(&parsed, &decl)
			if info.name != "" {
				unitName := symbolNameTableUnitName(topNames, info.name)
				info.name = arena.PersistString(info.name)
				info.receiverType = arena.PersistString(info.receiverType)
				info.unitName = unitName
				existingMethod := methodTableLookup(methods, info.name)
				if existingMethod.unitName == "" {
					methodOrder = append(methodOrder, info.name)
				}
				methods = append(methods, methodEntry{lookup: info.name, info: info})
			}
		}
		arena.Reset(mark)
	}
	syntheticEntrypoint := false
	needsSyntheticEntrypoint := false
	if pkg.Name == "main" {
		if symbolNameTableUnitName(topNames, "appMain") == "" {
			if symbolNameTableUnitName(topNames, "main") != "" {
				needsSyntheticEntrypoint = hasOrdinaryMainDecl
			}
		}
	}
	if needsSyntheticEntrypoint {
		topNameOrder = append(topNameOrder, "appMain")
		unitName := arena.PersistString(SymbolName(pkg.ImportPath, "appMain"))
		topNames = symbolNameTableSet(topNames, "appMain", unitName)
		topFunctionNames = symbolNameTableSet(topFunctionNames, "appMain", unitName)
		syntheticEntrypoint = true
	}
	staticCallbackDeclNames := packageStaticCallbackDeclNames(files, functionTypeNames)
	for i := 0; i < len(topNameOrder); i++ {
		name := topNameOrder[i]
		unitName := symbolNameTableUnitName(topNames, name)
		if isExported(name) && !containsString(staticCallbackDeclNames, name) {
			u.Exports = append(u.Exports, unit.Symbol{ImportPath: pkg.ImportPath, Name: name, UnitName: unitName})
		}
	}
	if needsPackageInit {
		u.Exports = append(u.Exports, packageInitSymbol(pkg.ImportPath))
	}
	if needsPanicRecover {
		panicSymbols := packagePanicStateSymbols(pkg.ImportPath)
		for i := 0; i < len(panicSymbols); i++ {
			u.Exports = append(u.Exports, panicSymbols[i])
		}
	}
	sortSymbolsByName(u.Exports)
	depMeta := dependencyMetadataForPackages(depPackages)
	packageAnonymousTypes := packageAnonymousStructTypeDeclsForLoadFiles(files, pkg.ImportPath)
	namedTypes := namedTypeUnderlyingsForLoadFiles(files)
	namedTypes = appendGeneratedTypeUnderlyingsFromBodies(namedTypes, packageAnonymousTypes)
	namedSlices := namedSliceTypesForLoadFiles(files, topNames)
	namedSlices = rewriteNamedSliceAnonymousStructElems(namedSlices, packageAnonymousTypes)
	namedSlices = appendDependencyNamedSliceTypes(namedSlices, depMeta.namedSlices)
	namedArrays := namedArrayTypesForLoadFiles(files, topNames)
	namedArrays = appendDependencyNamedArrayTypes(namedArrays, depMeta.namedArrays)
	namedMaps := namedMapTypesForLoadFiles(files, topNames)
	namedMaps = appendDependencyNamedMapTypes(namedMaps, depMeta.namedMaps)
	sizeofValueTypes := sizeofLocalTypesForLoadFiles(files)
	sizeofNamedTypes := sizeofNamedTypesForLoadFiles(files)
	namedConversions := namedConversionTypesForLoadFiles(files, topNames)
	namedConversions = appendDependencyNamedConversionTypes(namedConversions, depMeta.namedConversions)
	fieldTypes := packageStructFieldTypesForLoadFiles(files, pkg.ImportPath, namedTypes)
	structOwners := packageStructOwnersForLoadFiles(files)
	packageValueTypes := packageTopLevelValueTypesForLoadFiles(files, topNames)
	arrayFieldTypes := packageArrayStructFieldLowerInfosForLoadFiles(files, pkg.ImportPath, namedArrays)
	functionResults := functionResultTypesForLoadFiles(files, topNames)
	functionResults = appendLocalTypeEntries(functionResults, depMeta.functionResults)
	arrayFunctionResults := arrayFunctionResultTypesForLoadFiles(files, topNames, namedArrays)
	arrayFunctionResults = appendDependencyArrayFunctionResultTypes(arrayFunctionResults, depPackages)
	arrayFunctionParams := arrayFunctionParamTypesForLoadFiles(files, topNames, namedArrays)
	arrayFunctionParams = appendDependencyArrayFunctionParamTypes(arrayFunctionParams, depPackages)
	interfaceParamErasures := interfaceParamErasuresForLoadFiles(files, topNames)
	interfaceReturnErasures := interfaceReturnErasuresForLoadFiles(files, topNames)
	wordSize := target.WordSize(targetName)
	staticCallbacks := staticCallbackTemplatesForLoadFiles(files, pkg, depPackages, topNames, topFunctionNames, methods, methodOrder, packageAnonymousTypes, namedSlices, namedArrays, namedMaps, sizeofValueTypes, sizeofNamedTypes, fieldTypes, structOwners, functionResults, arrayFunctionResults, arrayFieldTypes, panicRecover, wordSize, functionTypeNames)
	staticCallbacks = appendDependencyStaticCallbackTemplates(staticCallbacks, depPackages, pkg.ImportPath, targetName)
	seenRefs := make([]string, 0, refCap)
	packageInitStatements := make([]string, 0, 8)
	packageInitTempIndex := 0
	for anonIndex := 0; anonIndex < len(packageAnonymousTypes); anonIndex++ {
		info := packageAnonymousTypes[anonIndex]
		if info.skipEmit {
			continue
		}
		var anonDecl unit.Decl
		anonDecl.Path = arena.PersistString(info.path)
		anonDecl.Kind = "type"
		anonDecl.Name = arena.PersistString(info.unitName)
		anonDecl.UnitName = arena.PersistString(info.unitName)
		anonDecl.Body = arena.PersistString(packageAnonymousStructDeclBody(info))
		u.Decls = append(u.Decls, anonDecl)
	}
	if needsPanicRecover {
		decls := packagePanicRecoverDecls(pkg.ImportPath, panicRecover)
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			u.Decls = append(u.Decls, decls[declIndex])
		}
	}
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsedValue, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			return unit.Unit{}, err
		}
		parsed := &parsedValue
		importRefs, _ := importReferenceMap(parsed, depPackages)
		fileFieldTypes := appendStructFieldTypeTables(cloneStructFieldTypeTable(fieldTypes), importedStructFieldTypesForFile(parsed, depPackages))
		fileStructOwners := cloneStructOwnerTable(structOwners)
		fileArrayFieldTypes := appendArrayStructFieldLowerInfoTables(cloneArrayStructFieldLowerInfoTable(arrayFieldTypes), importedArrayStructFieldLowerInfosForFile(parsed, depPackages))
		importedValueTypes := importedTopLevelValueTypesForFile(parsed, depPackages)
		importMethods, importMethodOrder := importMethodMap(parsed, depPackages, importedTypeLocalNames(parsed))
		allMethods := mergedMethodMap(methods, methodOrder, importMethods, importMethodOrder)
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := decls[declIndex]
			if staticCallbackTemplateForDecl(staticCallbacks, parsed, decl) >= 0 {
				continue
			}
			if topLevelFunctionTypeDeclForLower(parsed, decl, functionTypeNames) || topLevelFunctionContainingTypeDeclForLower(parsed, decl) || topLevelInterfaceTypeDeclForLower(parsed, decl) || topLevelInterfaceContainingTypeDeclForLower(parsed, decl) || topLevelMapTypeDeclForLower(parsed, decl) || topLevelMapContainingTypeDeclForLower(parsed, decl) || topLevelArrayTypeDeclForLower(parsed, decl) || topLevelAnyTypeDeclForLower(parsed, decl) || topLevelComplexTypeDeclForLower(parsed, decl) || topLevelComplexContainingTypeDeclForLower(parsed, decl) {
				continue
			}
			refs := make([]unit.Symbol, 0, declReferenceCapacity(parsed, &decl))
			initInfo, isInitDecl := initFunctionInfoForDecl(initFuncs, parsed.Path, decl.Start)
			localTypeDecls := localTypeDeclsForDecl(parsed, &decl, pkg.ImportPath, topNames)
			if len(localTypeDecls) > 0 {
				namedTypes = appendGeneratedLocalTypeUnderlyings(namedTypes, parsed, localTypeDecls)
				namedSlices = appendGeneratedLocalNamedSliceTypes(namedSlices, parsed, localTypeDecls)
				namedArrays = appendGeneratedLocalNamedArrayTypes(namedArrays, parsed, localTypeDecls)
				namedMaps = appendGeneratedLocalNamedMapTypes(namedMaps, parsed, localTypeDecls)
				namedConversions = appendGeneratedLocalNamedConversionTypes(namedConversions, parsed, localTypeDecls)
				fieldTypes = appendGeneratedLocalStructFieldTypes(fieldTypes, parsed, localTypeDecls, namedTypes)
				fileFieldTypes = appendGeneratedLocalStructFieldTypes(fileFieldTypes, parsed, localTypeDecls, namedTypes)
				structOwners = appendGeneratedLocalStructOwners(structOwners, parsed, localTypeDecls)
				fileStructOwners = appendGeneratedLocalStructOwners(fileStructOwners, parsed, localTypeDecls)
				arrayFieldTypes = appendGeneratedLocalArrayStructFieldLowerInfos(arrayFieldTypes, parsed, localTypeDecls, namedArrays)
				fileArrayFieldTypes = appendGeneratedLocalArrayStructFieldLowerInfos(fileArrayFieldTypes, parsed, localTypeDecls, namedArrays)
				for localIndex := 0; localIndex < len(localTypeDecls); localIndex++ {
					info := localTypeDecls[localIndex]
					if info.skipEmit {
						continue
					}
					var localDecl unit.Decl
					localDecl.Path = arena.PersistString(unitPathForDecl(files, parsed.Path))
					localDecl.Kind = "type"
					localDecl.Name = arena.PersistString(info.unitName)
					localDecl.UnitName = arena.PersistString(info.unitName)
					localBody := localTypeDeclBody(parsed, localTypeDecls, info)
					localBody = lowerNamedArrayTypeDeclarations(localBody, namedArrays)
					localDecl.Body = arena.PersistString(localBody)
					u.Decls = append(u.Decls, localDecl)
				}
			}
			declNamedTypes := namedTypes
			declFieldTypes := fileFieldTypes
			declStructOwners := fileStructOwners
			declArrayFieldTypes := fileArrayFieldTypes
			if len(localTypeDecls) > 0 {
				declNamedTypes = appendOriginalLocalTypeUnderlyings(cloneLocalTypeTable(namedTypes), parsed, localTypeDecls)
				declFieldTypes = appendOriginalLocalStructFieldTypes(cloneStructFieldTypeTable(fileFieldTypes), parsed, localTypeDecls, declNamedTypes)
				declStructOwners = appendOriginalLocalStructOwners(cloneStructOwnerTable(fileStructOwners), parsed, localTypeDecls)
				declArrayFieldTypes = appendOriginalLocalArrayStructFieldLowerInfos(cloneArrayStructFieldLowerInfoTable(fileArrayFieldTypes), parsed, localTypeDecls, namedArrays)
			}
			localTypes := localTypesForDecl(parsed, &decl)
			importLocalNames := importLocalNameTableForDecl(parsed, &decl, importRefs)
			localTypes = collectImportedFunctionResultLocalTypesForDecl(parsed, &decl, localTypes, importRefs, importLocalNames, functionResults)
			nameOverride := ""
			if isInitDecl {
				nameOverride = initInfo.unitName
			}
			declUnitName := unitDeclSymbol(&decl, parsed, topNames)
			rewriteTypeDecls := appendLocalTypeDeclInfos(packageAnonymousTypes, localTypeDecls)
			var generatedDecls []unit.Decl
			body := rewriteDecl(parsed, &decl, topNames, topFunctionNames, importRefs, allMethods, namedSlices, namedArrays, namedMaps, sizeofValueTypes, sizeofNamedTypes, rewriteTypeDecls, declFieldTypes, declStructOwners, packageValueTypes, importedValueTypes, functionResults, arrayFunctionResults, arrayFunctionParams, declArrayFieldTypes, interfaceParamErasures, interfaceReturnErasures, nameOverride, declUnitName, wordSize, &refs, &generatedDecls, &staticCallbacks)
			body = normalizeKeyedSliceLiterals(body, unitDeclSymbol(&decl, parsed, topNames), namedSlices, namedConversions)
			if decl.Kind == "var" {
				var initStatements []string
				body, initStatements = lowerPackageVarInitializerCalls(body, functionResults, packageInitUnitName(pkg.ImportPath), &packageInitTempIndex)
				for initIndex := 0; initIndex < len(initStatements); initIndex++ {
					packageInitStatements = append(packageInitStatements, arena.PersistString(initStatements[initIndex]))
				}
			}
			if decl.Kind == "func" {
				body = lowerStaticInterfaceAssertions(body, topNames, importRefs, declFieldTypes)
				body = lowerDeferStatements(body, declUnitName, panicRecover, packageMainEntrypointNeedsInitCall(pkg, decl), normalizationContext{
					localTypes:      localTypes,
					functionResults: functionResults,
					namedTypes:      declNamedTypes,
					fieldTypes:      declFieldTypes,
					generatedDecls:  &generatedDecls,
				})
				body = lowerCopyStringSources(body, localTypes, functionResults, declNamedTypes, declFieldTypes)
				body = normalizeFunctionExpressionsWithContext(body, declUnitName, namedSlices, namedConversions, normalizationContext{
					localTypes:      localTypes,
					functionResults: functionResults,
					namedTypes:      declNamedTypes,
					fieldTypes:      declFieldTypes,
					generatedDecls:  &generatedDecls,
				})
				body = lowerErrorMethodCalls(body, localTypes, functionResults)
				if needsPackageInit && packageMainEntrypointNeedsInitCall(pkg, decl) && !isInitDecl {
					body = prependPackageInitCallToFunction(body, packageInitUnitName(pkg.ImportPath))
				}
			}
			for generatedIndex := 0; generatedIndex < len(generatedDecls); generatedIndex++ {
				generated := generatedDecls[generatedIndex]
				generated.Path = arena.PersistString(generated.Path)
				generated.Kind = arena.PersistString(generated.Kind)
				generated.Name = arena.PersistString(generated.Name)
				generated.UnitName = arena.PersistString(generated.UnitName)
				generated.Body = arena.PersistString(generated.Body)
				u.Decls = append(u.Decls, generated)
			}
			for refIndex := 0; refIndex < len(refs); refIndex++ {
				ref := refs[refIndex]
				key := ref.ImportPath + "\x00" + ref.Name
				if !containsString(seenRefs, key) {
					seenRefs = append(seenRefs, arena.PersistString(key))
					persistedRef := persistUnitSymbol(ref)
					u.References = append(u.References, persistedRef)
				}
			}
			var outDecl unit.Decl
			outDecl.Path = arena.PersistString(unitPathForDecl(files, parsed.Path))
			outDecl.Kind = arena.PersistString(decl.Kind)
			if isInitDecl {
				outDecl.Name = arena.PersistString(initInfo.name)
				outDecl.UnitName = arena.PersistString(initInfo.unitName)
			} else {
				outDecl.Name = arena.PersistString(unitDeclName(parsed, &decl))
				outDecl.UnitName = arena.PersistString(unitDeclSymbol(&decl, parsed, topNames))
			}
			outDecl.Body = arena.PersistString(body)
			u.Decls = append(u.Decls, outDecl)
		}
		arena.Reset(mark)
	}
	if needsPackageInit {
		u.Decls = append(u.Decls, packageInitDoneDecl(pkg.ImportPath))
		u.Decls = append(u.Decls, packageInitDecl(pkg, depPackages, packageInitStatements, initFuncs, namedSlices, namedConversions))
		for importIndex := 0; importIndex < len(pkg.Imports); importIndex++ {
			importPath := pkg.Imports[importIndex]
			if dependencyPackageNeedsInit(importPath, depPackages, nil) {
				appendUnitSymbolRef(&u.References, packageInitSymbol(importPath))
			}
		}
	}
	if needsPanicRecover {
		bridgeSymbols := panicRecoverBridgeSymbols(panicRecover.bridges)
		for bridgeIndex := 0; bridgeIndex < len(bridgeSymbols); bridgeIndex++ {
			appendUnitSymbolRef(&u.References, bridgeSymbols[bridgeIndex])
		}
	}
	if syntheticEntrypoint {
		hasOS := containsString(pkg.Imports, "os")
		seenOSArgs := containsString(seenRefs, "os\x00Args")
		if hasOS && !seenOSArgs {
			osArgsUnitName := SymbolName("os", "Args")
			u.References = append(u.References, unit.Symbol{ImportPath: "os", Name: "Args", UnitName: osArgsUnitName})
		}
		appMainUnitName := symbolNameTableUnitName(topNames, "appMain")
		mainUnitName := symbolNameTableUnitName(topNames, "main")
		initUnitName := ""
		if needsPackageInit {
			initUnitName = packageInitUnitName(pkg.ImportPath)
		}
		entryDecl := syntheticAppMainDecl(appMainUnitName, mainUnitName, hasOS, initUnitName, panicRecover)
		u.Decls = append(u.Decls, entryDecl)
	}
	sortSymbolsByImportPathName(u.References)
	return u, nil
}

func declReferenceCapacity(file *parse.File, decl *parse.Decl) int {
	start := decl.Start
	end := decl.End
	if start < 0 {
		start = 0
	}
	if end > len(file.Source) {
		end = len(file.Source)
	}
	count := selectorReferenceCapacity(file.Tokens, start, end)
	if count < 4 {
		return 4
	}
	return count + 4
}

func selectorReferenceCapacity(toks []scan.Token, start int, end int) int {
	count := 0
	for i := 1; i+1 < len(toks); i++ {
		tok := toks[i]
		if int(tok.End) <= start {
			continue
		}
		if int(tok.Start) >= end {
			break
		}
		if tok.Text == "." && toks[i-1].Kind == scan.Ident && toks[i+1].Kind == scan.Ident {
			count += 2
		}
	}
	return count
}

func parsedLoadFile(file load.File) (parse.File, error) {
	if file.Parsed.Path != "" {
		return file.Parsed, nil
	}
	return parse.FileSource(file.Path, file.Source)
}

func parsedLoadFiles(files []load.File) ([]parse.File, error) {
	out := make([]parse.File, 0, len(files))
	for i := 0; i < len(files); i++ {
		parsed, err := parsedLoadFile(files[i])
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

type dependencyMetadata struct {
	namedSlices      []namedSliceInfo
	namedArrays      []namedArrayInfo
	namedMaps        []namedMapInfo
	namedConversions []string
	functionResults  localTypeTable
}

func dependencyMetadataForPackages(packages []load.Package) dependencyMetadata {
	sliceCap := 0
	arrayCap := 0
	mapCap := 0
	conversionCap := 0
	resultCap := 0
	for i := 0; i < len(packages); i++ {
		imports := packages[i].Imports
		files := packages[i].Files
		topNames := packageSymbolNamesForLoadFiles(packages[i].ImportPath, files, packageSymbolCapacity(files, len(imports)))
		s, a, m, c, r := dependencyMetadataCapacityForPackage(files, packages[i].ImportPath, topNames)
		sliceCap += s
		arrayCap += a
		mapCap += m
		conversionCap += c
		resultCap += r
	}
	meta := dependencyMetadata{
		namedSlices:      make([]namedSliceInfo, 0, sliceCap),
		namedArrays:      make([]namedArrayInfo, 0, arrayCap),
		namedMaps:        make([]namedMapInfo, 0, mapCap),
		namedConversions: make([]string, 0, conversionCap),
		functionResults:  make(localTypeTable, 0, resultCap),
	}
	for i := 0; i < len(packages); i++ {
		importPath := packages[i].ImportPath
		imports := packages[i].Imports
		files := packages[i].Files
		topNames := packageSymbolNamesForLoadFiles(importPath, files, packageSymbolCapacity(files, len(imports)))
		anonymousTypes := packageAnonymousStructTypeDeclsForLoadFiles(files, importPath)
		for parseIndex := 0; parseIndex < len(files); parseIndex++ {
			mark := arena.Mark()
			sliceStart := len(meta.namedSlices)
			arrayStart := len(meta.namedArrays)
			mapStart := len(meta.namedMaps)
			conversionStart := len(meta.namedConversions)
			resultStart := len(meta.functionResults)
			parsedFile, parseErr := parsedLoadFile(files[parseIndex])
			if parseErr != nil {
				arena.Reset(mark)
				break
			}
			toks := parsedFile.Tokens
			for tokIndex := 0; tokIndex+3 < len(toks); tokIndex++ {
				if toks[tokIndex].Text != "type" {
					continue
				}
				if toks[tokIndex+1].Text == "(" {
					close := findClose(toks, tokIndex+1, "(", ")")
					if close >= 0 {
						meta.namedSlices = appendGroupedNamedSliceTypes(meta.namedSlices, toks, tokIndex+2, close, topNames)
						meta.namedArrays = appendGroupedNamedArrayTypes(meta.namedArrays, toks, tokIndex+2, close, topNames)
						meta.namedMaps = appendGroupedNamedMapTypes(meta.namedMaps, toks, tokIndex+2, close, topNames)
						meta.namedConversions = appendGroupedNamedConversionTypes(meta.namedConversions, toks, tokIndex+2, close, topNames)
						tokIndex = close
					}
					continue
				}
				if toks[tokIndex+1].Kind != scan.Ident {
					continue
				}
				specEnd := packageTypeSpecEndForLower(toks, tokIndex+1)
				meta.namedSlices = appendNamedSliceType(meta.namedSlices, toks, tokIndex+1, tokIndex+2, specEnd, topNames)
				meta.namedArrays = appendNamedArrayType(meta.namedArrays, toks, tokIndex+1, tokIndex+2, specEnd, topNames)
				meta.namedMaps = appendNamedMapType(meta.namedMaps, toks, tokIndex+1, tokIndex+2, specEnd, topNames)
				if namedConversionNeedsNormalization(toks, tokIndex+2, specEnd) {
					name := toks[tokIndex+1].Text
					if name != "" && name != "_" {
						meta.namedConversions = append(meta.namedConversions, name)
						unitName := symbolNameTableUnitName(topNames, name)
						if unitName != "" {
							meta.namedConversions = append(meta.namedConversions, unitName)
						}
					}
				}
				tokIndex = specEnd - 1
			}
			decls := parsedFile.Decls
			for declIndex := 0; declIndex < len(decls); declIndex++ {
				decl := decls[declIndex]
				if decl.Kind != "func" {
					continue
				}
				typ := singleFunctionResultType(&parsedFile, &decl)
				if typ.name == "" {
					continue
				}
				name := decl.Name
				if decl.Receiver {
					name = methodDeclName(&parsedFile, &decl)
				}
				typ = qualifyDependencyFunctionResultTypeForLower(typ, topNames)
				meta.functionResults = localTypeTableSet(meta.functionResults, SymbolName(importPath, name), typ)
			}
			for entryIndex := sliceStart; entryIndex < len(meta.namedSlices); entryIndex++ {
				meta.namedSlices = rewriteNamedSliceAnonymousStructElems(meta.namedSlices, anonymousTypes)
				meta.namedSlices[entryIndex].name = arena.PersistString(meta.namedSlices[entryIndex].name)
				meta.namedSlices[entryIndex].elem = arena.PersistString(meta.namedSlices[entryIndex].elem)
				meta.namedSlices[entryIndex].elemInfo = persistLocalTypeInfo(meta.namedSlices[entryIndex].elemInfo)
			}
			for entryIndex := arrayStart; entryIndex < len(meta.namedArrays); entryIndex++ {
				meta.namedArrays[entryIndex].name = arena.PersistString(meta.namedArrays[entryIndex].name)
				meta.namedArrays[entryIndex].info.elem = arena.PersistString(meta.namedArrays[entryIndex].info.elem)
			}
			for entryIndex := mapStart; entryIndex < len(meta.namedMaps); entryIndex++ {
				meta.namedMaps[entryIndex].name = arena.PersistString(meta.namedMaps[entryIndex].name)
				meta.namedMaps[entryIndex].keyType = arena.PersistString(meta.namedMaps[entryIndex].keyType)
				meta.namedMaps[entryIndex].valueType = arena.PersistString(meta.namedMaps[entryIndex].valueType)
			}
			for entryIndex := conversionStart; entryIndex < len(meta.namedConversions); entryIndex++ {
				meta.namedConversions[entryIndex] = arena.PersistString(meta.namedConversions[entryIndex])
			}
			for entryIndex := resultStart; entryIndex < len(meta.functionResults); entryIndex++ {
				meta.functionResults[entryIndex].name = arena.PersistString(meta.functionResults[entryIndex].name)
				meta.functionResults[entryIndex].info = persistLocalTypeInfo(meta.functionResults[entryIndex].info)
			}
			arena.Reset(mark)
		}
	}
	return meta
}

func qualifyDependencyFunctionResultTypeForLower(info localTypeInfo, topNames symbolNameTable) localTypeInfo {
	if info.name == "" || info.qualifier != "" {
		return info
	}
	if unitName := symbolNameTableUnitName(topNames, info.name); unitName != "" {
		info.name = unitName
	}
	return info
}

func dependencyMetadataCapacityForPackage(files []load.File, importPath string, topNames symbolNameTable) (int, int, int, int, int) {
	sliceCount := 0
	arrayCount := 0
	mapCount := 0
	conversionCount := 0
	resultCount := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		sliceCount += len(appendNamedSliceTypesForFile(&parsed, topNames, nil))
		arrayCount += len(appendNamedArrayTypesForFile(&parsed, topNames, nil))
		mapCount += len(appendNamedMapTypesForFile(&parsed, topNames, nil))
		conversionCount += len(appendNamedConversionTypesForFile(&parsed, topNames, nil))
		resultCount += len(appendFunctionResultTypesForFile(&parsed, importPath, topNames, nil))
		arena.Reset(mark)
	}
	return sliceCount, arrayCount, mapCount, conversionCount, resultCount
}

func sortFilesByPath(files []load.File) {
	for i := 1; i < len(files); i++ {
		value := files[i]
		j := i - 1
		for j >= 0 && stringGreater(files[j].Path, value.Path) {
			files[j+1] = files[j]
			j = j - 1
		}
		files[j+1] = value
	}
}

func sortSymbolsByName(symbols []unit.Symbol) {
	for i := 1; i < len(symbols); i++ {
		value := symbols[i]
		j := i - 1
		for j >= 0 && stringGreater(symbols[j].Name, value.Name) {
			symbols[j+1] = symbols[j]
			j = j - 1
		}
		symbols[j+1] = value
	}
}

func sortSymbolsByImportPathName(symbols []unit.Symbol) {
	for i := 1; i < len(symbols); i++ {
		value := symbols[i]
		j := i - 1
		for j >= 0 && symbolAfterByImportPathName(symbols[j], value) {
			symbols[j+1] = symbols[j]
			j = j - 1
		}
		symbols[j+1] = value
	}
}

func symbolAfterByImportPathName(a unit.Symbol, b unit.Symbol) bool {
	if a.ImportPath == b.ImportPath {
		return stringGreater(a.Name, b.Name)
	}
	return stringGreater(a.ImportPath, b.ImportPath)
}

func stringGreater(a string, b string) bool {
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
		i = i + 1
	}
	return len(a) > len(b)
}

func unitDeclName(file *parse.File, decl *parse.Decl) string {
	if decl.Kind == "func" && decl.Receiver {
		return methodDeclName(file, decl)
	}
	names := declNames(decl)
	if len(names) == 1 {
		return names[0]
	}
	if decl.Kind == "func" {
		return decl.Name
	}
	return strings.Join(names, ", ")
}

func unitDeclSymbol(decl *parse.Decl, file *parse.File, topNames symbolNameTable) string {
	if decl.Kind == "func" && decl.Receiver {
		return symbolNameTableUnitName(topNames, methodDeclName(file, decl))
	}
	names := declNames(decl)
	if len(names) != 1 {
		return ""
	}
	return symbolNameTableUnitName(topNames, names[0])
}

func topLevelFunctionTypeDeclForLower(file *parse.File, decl parse.Decl, functionTypeNames []string) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !functionTypeSpecRangeForLower(toks, spec.start, spec.end) && !functionTypeAliasSpecRangeForLower(toks, spec.start, spec.end, functionTypeNames) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingFunctionParameterTypesForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		containsString(functionTypeNames, name) &&
		!identifierUsedOutsideSourceRangeAllowingFunctionParameterTypesForLower(toks, name, decl.Start, decl.End)
}

func topLevelFunctionContainingTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !functionContainingTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		functionContainingTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End)
}

func topLevelInterfaceTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !interfaceTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		interfaceTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, decl.Start, decl.End)
}

func topLevelInterfaceContainingTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !interfaceContainingTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		interfaceContainingTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, decl.Start, decl.End)
}

func topLevelMapTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !mapTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		mapTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, decl.Start, decl.End)
}

func topLevelMapContainingTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !mapContainingTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		mapContainingTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, decl.Start, decl.End)
}

func topLevelArrayTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !arrayTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrSizeofForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		arrayTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrSizeofForLower(toks, name, decl.Start, decl.End)
}

func topLevelAnyTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !anyTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		anyTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End)
}

func topLevelComplexTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !complexTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		complexTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End)
}

func topLevelComplexContainingTypeDeclForLower(file *parse.File, decl parse.Decl) bool {
	if decl.Kind != "type" {
		return false
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
		return false
	}
	end := tokenIndexBeforeForLower(toks, decl.End) + 1
	if end <= start+2 {
		return false
	}
	if toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close < 0 || close >= end {
			return false
		}
		ranges := localConstSpecRanges(toks, start+2, close)
		if len(ranges) == 0 {
			return false
		}
		for i := 0; i < len(ranges); i++ {
			spec := ranges[i]
			if !complexContainingTypeSpecRangeForLower(toks, spec.start, spec.end) {
				return false
			}
			name := functionTypeSpecNameForLower(toks, spec.start, spec.end)
			if name == "" || identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End) {
				return false
			}
		}
		return true
	}
	if toks[start+1].Kind != scan.Ident {
		return false
	}
	name := functionTypeSpecNameForLower(toks, start+1, end)
	return name != "" &&
		complexContainingTypeSpecRangeForLower(toks, start+1, end) &&
		!identifierUsedOutsideSourceRangeForLower(toks, name, decl.Start, decl.End)
}

func functionTypeSpecNameForLower(toks []scan.Token, start int, end int) string {
	start, _ = trimTokenRange(toks, start, end)
	if start < end && toks[start].Kind == scan.Ident {
		return toks[start].Text
	}
	return ""
}

func functionTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "func" || toks[typeStart+1].Text != "(" {
		return false
	}
	close := findClose(toks, typeStart+1, "(", ")")
	return close > typeStart+1 && close < end
}

func functionTypeNamesForLoadFiles(files []load.File) []string {
	var names []string
	passes := len(files) + 4
	if passes < 4 {
		passes = 4
	}
	for pass := 0; pass < passes; pass++ {
		before := cloneStrings(names)
		for fileIndex := 0; fileIndex < len(files); fileIndex++ {
			mark := arena.Mark()
			parsed, err := parsedLoadFile(files[fileIndex])
			if err != nil {
				arena.Reset(mark)
				continue
			}
			names = appendFunctionTypeNamesForFile(names, &parsed)
			arena.Reset(mark)
		}
		if stringSlicesEqualForLower(before, names) {
			break
		}
	}
	return names
}

func appendFunctionTypeNamesForFile(names []string, file *parse.File) []string {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "type" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 || start+2 >= len(toks) || toks[start].Text != "type" {
			continue
		}
		end := tokenIndexBeforeForLower(toks, decl.End) + 1
		if toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 || close >= end {
				continue
			}
			ranges := localConstSpecRanges(toks, start+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				names = appendFunctionTypeNameForSpec(names, toks, ranges[rangeIndex].start, ranges[rangeIndex].end)
			}
			continue
		}
		names = appendFunctionTypeNameForSpec(names, toks, start+1, end)
	}
	return names
}

func appendFunctionTypeNameForSpec(names []string, toks []scan.Token, start int, end int) []string {
	if !functionTypeSpecRangeForLower(toks, start, end) && !functionTypeAliasSpecRangeForLower(toks, start, end, names) {
		return names
	}
	name := functionTypeSpecNameForLower(toks, start, end)
	if name == "" {
		return names
	}
	return appendStringUnique(names, name)
}

func functionTypeAliasSpecRangeForLower(toks []scan.Token, start int, end int, functionTypeNames []string) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return typeStart+1 == end && toks[typeStart].Kind == scan.Ident && containsString(functionTypeNames, toks[typeStart].Text)
}

func stringSlicesEqualForLower(a []string, b []string) bool {
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

func functionContainingTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalancedForLower(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsFunctionTypeForLower(toks, typeStart, end)
}

func typeRangeContainsFunctionTypeForLower(toks []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if toks[i].Text == "func" && toks[i+1].Text == "(" {
			return true
		}
	}
	return false
}

func interfaceTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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

func interfaceContainingTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalancedForLower(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsInterfaceTypeForLower(toks, typeStart, end)
}

func typeRangeContainsInterfaceTypeForLower(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "interface" || startsUnsupportedPredeclaredTypeForLower(toks, i, "any") {
			return true
		}
	}
	return false
}

func startsUnsupportedPredeclaredTypeForLower(toks []scan.Token, i int, name string) bool {
	if toks[i].Text != name || i == 0 {
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
		return isFunctionSignatureResultForLower(toks, i)
	}
	if prev.Kind != scan.Ident || i < 2 {
		return false
	}
	beforeName := toks[i-2].Text
	return beforeName == "var" || beforeName == "type" || beforeName == "(" || beforeName == "{" || beforeName == ","
}

func isFunctionSignatureResultForLower(toks []scan.Token, pos int) bool {
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

func mapTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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
	return close > typeStart+1 && close < end-1 && typeRangeBalancedForLower(toks, close+1, end)
}

func mapContainingTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalancedForLower(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsMapTypeForLower(toks, typeStart, end)
}

func typeRangeContainsMapTypeForLower(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "map" {
			return true
		}
	}
	return false
}

func arrayTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	_, ok := fixedArrayTypeLowerInfoForTokens(toks, typeStart, end)
	return ok
}

func anyTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	return typeStart+1 == end && toks[typeStart].Text == "any"
}

func complexTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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

func complexContainingTypeSpecRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= end || !typeRangeBalancedForLower(toks, typeStart, end) {
		return false
	}
	return typeRangeContainsComplexTypeForLower(toks, typeStart, end)
}

func typeRangeContainsComplexTypeForLower(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if startsUnsupportedPredeclaredTypeForLower(toks, i, "complex64") || startsUnsupportedPredeclaredTypeForLower(toks, i, "complex128") {
			return true
		}
	}
	return false
}

func typeRangeBalancedForLower(toks []scan.Token, start int, end int) bool {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		updateDepthForLower(toks[i].Text, &paren, &brack, &brace)
	}
	return paren == 0 && brack == 0 && brace == 0
}

func identifierUsedOutsideSourceRangeForLower(toks []scan.Token, name string, start int, end int) bool {
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

func identifierUsedOutsideSourceRangeAllowingFunctionParameterTypesForLower(toks []scan.Token, name string, start int, end int) bool {
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
		if tokenIsNamedFunctionParameterTypeForLower(toks, i) {
			continue
		}
		if tokenIsNamedFunctionTypeAliasUseForLower(toks, i) {
			continue
		}
		return true
	}
	return false
}

func tokenIsNamedFunctionParameterTypeForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "func" || toks[i+1].Kind != scan.Ident || toks[i+2].Text != "(" {
			continue
		}
		paramsOpen := i + 2
		paramsClose := findClose(toks, paramsOpen, "(", ")")
		if paramsClose < 0 {
			continue
		}
		if pos <= paramsOpen || pos >= paramsClose {
			continue
		}
		segments := topLevelExpressionRanges(toks, paramsOpen+1, paramsClose)
		for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
			segment := segments[segmentIndex]
			_, typeStart, typeEnd, hasType := staticCallbackParameterSegmentForLower(toks, segment.start, segment.end)
			if !hasType {
				continue
			}
			typeStart, typeEnd = trimTokenRange(toks, typeStart, typeEnd)
			if typeStart == pos && typeStart+1 == typeEnd {
				return true
			}
		}
	}
	return false
}

func tokenIsNamedFunctionTypeAliasUseForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			ranges := localConstSpecRanges(toks, i+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				if tokenIsNamedFunctionTypeAliasUseInSpecForLower(toks, pos, ranges[rangeIndex].start, ranges[rangeIndex].end) {
					return true
				}
			}
			i = close
			continue
		}
		specEnd := typeSpecEnd(toks, i+2)
		if tokenIsNamedFunctionTypeAliasUseInSpecForLower(toks, pos, i+1, specEnd) {
			return true
		}
		i = specEnd - 1
	}
	return false
}

func tokenIsNamedFunctionTypeAliasUseInSpecForLower(toks []scan.Token, pos int, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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

func identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks []scan.Token, name string, start int, end int) bool {
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
		if tokenInsideInterfaceTypeSpecForLower(toks, i) || tokenInsideInterfaceContainingTypeSpecForLower(toks, i) {
			continue
		}
		return true
	}
	return false
}

func identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks []scan.Token, name string, start int, end int) bool {
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
		if tokenInsideMapTypeSpecForLower(toks, i) || tokenInsideMapContainingTypeSpecForLower(toks, i) {
			continue
		}
		if identifierInsideDiscardedEmptyCompositeUseForLower(toks, i, name) {
			continue
		}
		if identifierInsideNamedMapCompositeUseForLower(toks, i, name) {
			continue
		}
		return true
	}
	return false
}

func identifierInsideNamedMapCompositeUseForLower(toks []scan.Token, pos int, name string) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos].Text != name {
		return false
	}
	if toks[pos+1].Text == "{" {
		return true
	}
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		for j := rhs[i].start; j+1 < rhs[i].end; j++ {
			if toks[j].Kind == scan.Ident && toks[j].Text == name && toks[j+1].Text == "{" {
				return true
			}
		}
	}
	return false
}

func identifierInsideDiscardedEmptyCompositeUseForLower(toks []scan.Token, pos int, name string) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos].Text != name {
		return false
	}
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return false
	}
	matched := false
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return false
		}
		if !emptyCompositeLiteralExpressionForLower(toks, rhs[i].start, rhs[i].end) {
			return false
		}
		if pos >= rhs[i].start && pos < rhs[i].end {
			matched = true
		}
	}
	return matched
}

func sameLineAssignmentStatementStartForLower(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" {
			return i + 1
		}
	}
	return 0
}

func emptyCompositeLiteralExpressionForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return emptyCompositeLiteralExpressionForLower(toks, start+1, close)
		}
	}
	if start+3 != end || toks[start].Kind != scan.Ident || toks[start+1].Text != "{" {
		return false
	}
	close := findClose(toks, start+1, "{", "}")
	return close == end-1 && start+2 == close
}

func identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsForLower(toks []scan.Token, name string, start int, end int) bool {
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
		if tokenInsideArrayTypeSpecForLower(toks, i) {
			continue
		}
		return true
	}
	return false
}

func identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsOrSizeofForLower(toks []scan.Token, name string, start int, end int) bool {
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
		if tokenInsideArrayTypeSpecForLower(toks, i) {
			continue
		}
		if namedCompositeLiteralIsDirectUnsafeSizeofArgForLower(toks, i) {
			continue
		}
		return true
	}
	return false
}

func namedCompositeLiteralIsDirectUnsafeSizeofArgForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return false
	}
	open := compositeLiteralOpenForTypeStart(toks, pos)
	if open <= pos {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close < 0 {
		return false
	}
	return compositeLiteralIsDirectUnsafeSizeofArgForLower(toks, pos, close)
}

func compositeLiteralIsDirectUnsafeSizeofArgForLower(toks []scan.Token, start int, close int) bool {
	if start < 2 || close+1 >= len(toks) || toks[start-1].Text != "(" || toks[close+1].Text != ")" {
		return false
	}
	if toks[start-2].Text == "Sizeof" && (start < 4 || toks[start-3].Text != ".") {
		return true
	}
	return start >= 4 && toks[start-2].Text == "Sizeof" && toks[start-3].Text == "." && toks[start-4].Kind == scan.Ident
}

func tokenInsideInterfaceTypeSpecForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDeclForLower(toks, i) {
			continue
		}
		specEnd := interfaceTypeSpecEndFromNameForLower(toks, i, len(toks))
		if specEnd <= i || !interfaceTypeSpecRangeForLower(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideInterfaceContainingTypeSpecForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDeclForLower(toks, i) {
			continue
		}
		specEnd := interfaceTypeSpecEndFromNameForLower(toks, i, len(toks))
		if specEnd <= i || !interfaceContainingTypeSpecRangeForLower(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideMapTypeSpecForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDeclForLower(toks, i) {
			continue
		}
		specEnd := interfaceTypeSpecEndFromNameForLower(toks, i, len(toks))
		if specEnd <= i || !mapTypeSpecRangeForLower(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideMapContainingTypeSpecForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDeclForLower(toks, i) {
			continue
		}
		specEnd := interfaceTypeSpecEndFromNameForLower(toks, i, len(toks))
		if specEnd <= i || !mapContainingTypeSpecRangeForLower(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func tokenInsideArrayTypeSpecForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	for i := 0; i < len(toks); i++ {
		if !interfaceTypeSpecStartsInTypeDeclForLower(toks, i) {
			continue
		}
		specEnd := interfaceTypeSpecEndFromNameForLower(toks, i, len(toks))
		if specEnd <= i || !arrayTypeSpecRangeForLower(toks, i, specEnd) {
			continue
		}
		if pos >= i && pos < specEnd {
			return true
		}
	}
	return false
}

func interfaceTypeSpecStartsInTypeDeclForLower(toks []scan.Token, start int) bool {
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
		return typeBlockContainsOpenForLower(toks, start)
	}
	if prev.Line < toks[start].Line {
		return typeBlockContainsOpenForLower(toks, start)
	}
	return false
}

func typeBlockContainsOpenForLower(toks []scan.Token, pos int) bool {
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

func interfaceTypeSpecEndFromNameForLower(toks []scan.Token, start int, limit int) int {
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
		updateDepthForLower(toks[i].Text, &paren, &brack, &brace)
	}
	if limit < len(toks) {
		return limit
	}
	return len(toks)
}

func updateDepthForLower(text string, paren *int, brack *int, brace *int) {
	if text == "(" {
		*paren = *paren + 1
	}
	if text == ")" && *paren > 0 {
		*paren = *paren - 1
	}
	if text == "[" {
		*brack = *brack + 1
	}
	if text == "]" && *brack > 0 {
		*brack = *brack - 1
	}
	if text == "{" {
		*brace = *brace + 1
	}
	if text == "}" && *brace > 0 {
		*brace = *brace - 1
	}
}

func declTopNames(file *parse.File, decl *parse.Decl) []string {
	if decl.Kind == "func" && decl.Name == "init" && !decl.Receiver {
		return nil
	}
	if decl.Kind == "func" && decl.Receiver {
		name := methodDeclName(file, decl)
		if name == "" {
			return nil
		}
		return []string{name}
	}
	return declNames(decl)
}

func methodDeclName(file *parse.File, decl *parse.Decl) string {
	info := methodDeclInfoFromTokens(file.Tokens, decl)
	if info.receiverType == "" || decl.Name == "" {
		return decl.Name
	}
	return info.name
}

func methodDeclInfo(file *parse.File, decl *parse.Decl) methodInfo {
	return methodDeclInfoFromTokens(file.Tokens, decl)
}

func methodDeclNameFromTokens(toks []scan.Token, decl *parse.Decl) string {
	info := methodDeclInfoFromTokens(toks, decl)
	if info.receiverType == "" || decl.Name == "" {
		return decl.Name
	}
	return info.name
}

func methodDeclInfoFromTokens(toks []scan.Token, decl *parse.Decl) methodInfo {
	receiver := methodReceiverTypeNameFromTokens(toks, decl)
	name := decl.Name
	if receiver != "" && decl.Name != "" {
		name = receiver + "_" + decl.Name
	}
	return methodInfo{
		name:            name,
		receiverType:    receiver,
		pointerReceiver: methodReceiverIsPointerFromTokens(toks, decl),
	}
}

func methodReceiverTypeName(file *parse.File, decl *parse.Decl) string {
	return methodReceiverTypeNameFromTokens(file.Tokens, decl)
}

func methodReceiverTypeNameFromTokens(toks []scan.Token, decl *parse.Decl) string {
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return ""
	}
	close := findClose(toks, start+1, "(", ")")
	if close < 0 {
		return ""
	}
	name := ""
	for i := start + 2; i < close; i++ {
		if toks[i].Kind == scan.Ident {
			name = toks[i].Text
		}
	}
	return name
}

func methodReceiverIsPointer(file *parse.File, decl *parse.Decl) bool {
	return methodReceiverIsPointerFromTokens(file.Tokens, decl)
}

func methodReceiverIsPointerFromTokens(toks []scan.Token, decl *parse.Decl) bool {
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close < 0 {
		return false
	}
	return containsTokenText(toks, start+2, close, "*")
}

func hasOrdinaryMain(files []parse.File) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		file := files[fileIndex]
		for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
			decl := file.Decls[declIndex]
			if isOrdinaryMainDecl(file, decl) {
				return true
			}
		}
	}
	return false
}

func isOrdinaryMainDecl(file parse.File, decl parse.Decl) bool {
	if decl.Kind != "func" || decl.Name != "main" || decl.Receiver {
		return false
	}
	name := tokenIndexAt(file.Tokens, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(file.Tokens) || file.Tokens[name+1].Text != "(" {
		return false
	}
	open := name + 1
	close := findClose(file.Tokens, open, "(", ")")
	if close != open+1 {
		return false
	}
	for i := close + 1; i < len(file.Tokens) && int(file.Tokens[i].Start) < decl.End; i++ {
		if file.Tokens[i].Text == "{" {
			return true
		}
		return false
	}
	return false
}

func syntheticAppMainDecl(appMainUnitName string, mainUnitName string, setOSArgs bool, initUnitName string, panicRecover panicRecoverNames) unit.Decl {
	body := "func " + appMainUnitName + "(args []string, env []string) int {\n"
	if setOSArgs {
		body = body + "\t" + SymbolName("os", "Args") + " = args\n"
	}
	if initUnitName != "" {
		body = body + "\t" + initUnitName + "()\n"
	}
	body = body + "\t" + mainUnitName + "()\n"
	if panicRecover.active != "" && panicRecover.abort != "" {
		body = body + "\tif " + panicRecover.active + " {\n"
		body = body + "\t\treturn " + panicRecover.abort + "()\n"
		body = body + "\t}\n"
	}
	body = body + "\treturn 0\n}\n"
	return unit.Decl{
		Path:     "rtg-entrypoint",
		Kind:     "func",
		Name:     "appMain",
		UnitName: appMainUnitName,
		Body:     body,
	}
}

func packageInitMetadataName() string {
	return "__rtg_init"
}

func packageInitDoneMetadataName() string {
	return "__rtg_init_done"
}

func packageInitUnitName(importPath string) string {
	return SymbolName(importPath, packageInitMetadataName())
}

func packageInitDoneUnitName(importPath string) string {
	return SymbolName(importPath, packageInitDoneMetadataName())
}

func packageInitSymbol(importPath string) unit.Symbol {
	return unit.Symbol{ImportPath: importPath, Name: packageInitMetadataName(), UnitName: packageInitUnitName(importPath)}
}

func packageInitFunctionInfos(files []load.File, importPath string) []initFunctionInfo {
	var infos []initFunctionInfo
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if decl.Kind != "func" || decl.Name != "init" || decl.Receiver {
				continue
			}
			name := packageInitMetadataName() + "_func_" + strconv.Itoa(len(infos))
			infos = append(infos, initFunctionInfo{
				path:     arena.PersistString(parsed.Path),
				start:    decl.Start,
				name:     arena.PersistString(name),
				unitName: arena.PersistString(SymbolName(importPath, name)),
			})
		}
		arena.Reset(mark)
	}
	return infos
}

func packageHasCallableVarInitializers(files []load.File) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if decl.Kind == "var" && packageVarDeclHasCallableInitializer(&parsed, &decl) {
				arena.Reset(mark)
				return true
			}
		}
		arena.Reset(mark)
	}
	return false
}

func packageVarDeclHasCallableInitializer(file *parse.File, decl *parse.Decl) bool {
	start := decl.Start
	end := decl.End
	if start < 0 {
		start = 0
	}
	if end > len(file.Source) {
		end = len(file.Source)
	}
	if start >= end {
		return false
	}
	return packageVarBodyHasCallableInitializer(string(file.Source[start:end]))
}

func packageVarBodyHasCallableInitializer(body string) bool {
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return false
	}
	if len(toks) < 3 || toks[0].Text != "var" {
		return false
	}
	if toks[1].Text == "(" {
		close := findClose(toks, 1, "(", ")")
		if close < 0 {
			return false
		}
		specs := localConstSpecRanges(toks, 2, close)
		for i := 0; i < len(specs); i++ {
			if packageVarSpecHasCallableInitializer(toks, specs[i].start, specs[i].end) {
				return true
			}
		}
		return false
	}
	end := len(toks)
	if end > 0 && toks[end-1].Kind == scan.EOF {
		end--
	}
	return packageVarSpecHasCallableInitializer(toks, 1, end)
}

func packageVarSpecHasCallableInitializer(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	eq := findTopLevelToken(toks, start, end, "=")
	if eq < 0 || eq+1 >= end {
		return false
	}
	values := topLevelExpressionRanges(toks, eq+1, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		if expressionContainsPackageInitializerCall(toks, value.start, value.end) {
			return true
		}
	}
	return false
}

func expressionContainsPackageInitializerCall(toks []scan.Token, start int, end int) bool {
	return expressionContainsNonConversionCall(toks, start, end)
}

func packageNeedsInit(pkg load.Package, depPackages []load.Package, initFuncs []initFunctionInfo, hasPackageInitializerCalls bool) bool {
	if len(initFuncs) > 0 {
		return true
	}
	if hasPackageInitializerCalls {
		return true
	}
	for i := 0; i < len(pkg.Imports); i++ {
		if dependencyPackageNeedsInit(pkg.Imports[i], depPackages, nil) {
			return true
		}
	}
	return false
}

func dependencyPackageNeedsInit(importPath string, depPackages []load.Package, visiting []string) bool {
	if containsString(visiting, importPath) {
		return false
	}
	for i := 0; i < len(depPackages); i++ {
		pkg := depPackages[i]
		if pkg.ImportPath != importPath {
			continue
		}
		if len(packageInitFunctionInfos(pkg.Files, pkg.ImportPath)) > 0 {
			return true
		}
		if packageHasCallableVarInitializers(pkg.Files) {
			return true
		}
		nextVisiting := append(visiting, importPath)
		for importIndex := 0; importIndex < len(pkg.Imports); importIndex++ {
			if dependencyPackageNeedsInit(pkg.Imports[importIndex], depPackages, nextVisiting) {
				return true
			}
		}
		return false
	}
	return false
}

func initFunctionInfoForDecl(infos []initFunctionInfo, path string, start int) (initFunctionInfo, bool) {
	for i := 0; i < len(infos); i++ {
		info := infos[i]
		if info.path == path && info.start == start {
			return info, true
		}
	}
	return initFunctionInfo{}, false
}

func packageInitDoneDecl(importPath string) unit.Decl {
	name := packageInitDoneMetadataName()
	unitName := packageInitDoneUnitName(importPath)
	return unit.Decl{
		Path:     "rtg-init",
		Kind:     "var",
		Name:     name,
		UnitName: unitName,
		Body:     "var " + unitName + " bool\n",
	}
}

func packageInitDecl(pkg load.Package, depPackages []load.Package, initStatements []string, initFuncs []initFunctionInfo, namedSlices []namedSliceInfo, namedConversions []string) unit.Decl {
	unitName := packageInitUnitName(pkg.ImportPath)
	doneName := packageInitDoneUnitName(pkg.ImportPath)
	body := "func " + unitName + "() {\n"
	body = body + "\tif " + doneName + " {\n\t\treturn\n\t}\n"
	body = body + "\t" + doneName + " = true\n"
	imports := copyStrings(pkg.Imports)
	sortLowerStrings(imports)
	for i := 0; i < len(imports); i++ {
		if !dependencyPackageNeedsInit(imports[i], depPackages, nil) {
			continue
		}
		body = body + "\t" + packageInitUnitName(imports[i]) + "()\n"
	}
	for i := 0; i < len(initStatements); i++ {
		body = body + "\t" + initStatements[i] + "\n"
	}
	for i := 0; i < len(initFuncs); i++ {
		body = body + "\t" + initFuncs[i].unitName + "()\n"
	}
	body = body + "}\n"
	body = normalizeFunctionExpressions(body, unitName, namedSlices, namedConversions)
	return unit.Decl{
		Path:     "rtg-init",
		Kind:     "func",
		Name:     packageInitMetadataName(),
		UnitName: unitName,
		Body:     body,
	}
}

func sortLowerStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && stringGreater(values[j], value) {
			values[j+1] = values[j]
			j--
		}
		values[j+1] = value
	}
}

func packageMainEntrypointNeedsInitCall(pkg load.Package, decl parse.Decl) bool {
	return pkg.Name == "main" && decl.Kind == "func" && decl.Name == "appMain" && !decl.Receiver
}

func prependPackageInitCallToFunction(body string, initUnitName string) string {
	if initUnitName == "" {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == "{" {
			insert := int(toks[i].End)
			return body[:insert] + "\n\t" + initUnitName + "()" + body[insert:]
		}
	}
	return body
}

type panicRecoverNames struct {
	active  string
	value   string
	recover string
	abort   string
	bridges []panicRecoverBridge
}

type panicRecoverBridge struct {
	importPath string
	active     string
	value      string
}

type loweredDeferCall struct {
	call     string
	active   string
	captures []deferCapture
}

type deferCapture struct {
	name string
	typ  string
}

type nestedDeferActivation struct {
	pos      int
	active   string
	captures []deferCapture
}

type dynamicDeferPlan struct {
	enabled bool
	order   string
	index   string
	kind    string
	sites   []dynamicDeferSite
}

type dynamicDeferSite struct {
	pos      int
	id       int
	callee   string
	captures []dynamicDeferCapture
}

type dynamicDeferCapture struct {
	name     string
	stack    string
	typ      string
	expanded bool
}

type pendingPanicBlockClose struct {
	token       int
	indent      string
	beforeClose []string
}

func panicRecoverNamesForPackage(importPath string) panicRecoverNames {
	return panicRecoverNames{
		active:  SymbolName(importPath, "__rtg_panic_active"),
		value:   SymbolName(importPath, "__rtg_panic_value"),
		recover: SymbolName(importPath, "__rtg_recover"),
		abort:   SymbolName(importPath, "__rtg_panic_abort"),
	}
}

func packageNeedsPanicRecover(pkg load.Package, depPackages []load.Package, visiting []string) bool {
	if containsString(visiting, pkg.ImportPath) {
		return false
	}
	if packageUsesPanicRecover(pkg.Files) {
		return true
	}
	nextVisiting := append(visiting, pkg.ImportPath)
	for i := 0; i < len(pkg.Imports); i++ {
		dep, ok := packageByImportPath(depPackages, pkg.Imports[i])
		if !ok {
			continue
		}
		if packageNeedsPanicRecover(dep, depPackages, nextVisiting) {
			return true
		}
	}
	return false
}

func panicRecoverBridgesForPackage(pkg load.Package, depPackages []load.Package) []panicRecoverBridge {
	var out []panicRecoverBridge
	for i := 0; i < len(pkg.Imports); i++ {
		importPath := pkg.Imports[i]
		dep, ok := packageByImportPath(depPackages, importPath)
		if !ok {
			continue
		}
		if !packageNeedsPanicRecover(dep, depPackages, []string{pkg.ImportPath}) {
			continue
		}
		names := panicRecoverNamesForPackage(importPath)
		out = append(out, panicRecoverBridge{importPath: importPath, active: names.active, value: names.value})
	}
	return out
}

func packagePanicStateSymbols(importPath string) []unit.Symbol {
	return []unit.Symbol{
		packagePanicActiveSymbol(importPath),
		packagePanicValueSymbol(importPath),
	}
}

func packagePanicActiveSymbol(importPath string) unit.Symbol {
	names := panicRecoverNamesForPackage(importPath)
	return unit.Symbol{ImportPath: importPath, Name: "__rtg_panic_active", UnitName: names.active}
}

func packagePanicValueSymbol(importPath string) unit.Symbol {
	names := panicRecoverNamesForPackage(importPath)
	return unit.Symbol{ImportPath: importPath, Name: "__rtg_panic_value", UnitName: names.value}
}

func panicRecoverBridgeSymbols(bridges []panicRecoverBridge) []unit.Symbol {
	var out []unit.Symbol
	for i := 0; i < len(bridges); i++ {
		bridge := bridges[i]
		out = appendSymbolByImportPathName(out, packagePanicActiveSymbol(bridge.importPath))
		out = appendSymbolByImportPathName(out, packagePanicValueSymbol(bridge.importPath))
	}
	return out
}

func appendSymbolByImportPathName(symbols []unit.Symbol, sym unit.Symbol) []unit.Symbol {
	for i := 0; i < len(symbols); i++ {
		if symbols[i].ImportPath == sym.ImportPath && symbols[i].Name == sym.Name {
			symbols[i] = sym
			return symbols
		}
	}
	return append(symbols, sym)
}

func packageUsesPanicRecover(files []load.File) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		toks, err := scan.Tokens(files[fileIndex].Source)
		if err != nil {
			continue
		}
		for i := 0; i+1 < len(toks); i++ {
			if toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
				continue
			}
			if toks[i].Text == "panic" || toks[i].Text == "recover" {
				return true
			}
		}
		if packageUsesStaticInterfaceAssertionPanic(files[fileIndex]) {
			return true
		}
	}
	return false
}

func packageUsesStaticInterfaceAssertionPanic(file load.File) bool {
	parsed, err := parsedLoadFile(file)
	if err != nil {
		return false
	}
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" || decl.Start < 0 || decl.End > len(parsed.Source) || decl.Start >= decl.End {
			continue
		}
		if functionUsesStaticInterfaceAssertionPanicForLower(string(parsed.Source[decl.Start:decl.End])) {
			return true
		}
	}
	return false
}

func functionUsesStaticInterfaceAssertionPanicForLower(body string) bool {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return false
	}
	bodyOpen, bodyClose, ok := functionBodyRange(toks)
	if !ok {
		return false
	}
	var vars []staticInterfaceAssertionVarForLower
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAtForLower(toks, i, bodyOpen, bodyClose, symbolNameTable{}, nil, nil)
		if !ok {
			continue
		}
		vars = append(vars, info)
		i = info.declEnd - 1
	}
	for i := bodyOpen + 1; i+2 < bodyClose; i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "." {
			continue
		}
		info, ok := staticInterfaceAssertionVarByNameForLower(vars, toks[i].Text)
		if !ok {
			continue
		}
		assertClose := findClose(toks, i+2, "(", ")")
		asserted := staticInterfaceAssertionTypeNameForLower(toks, i+1)
		if asserted != info.concreteType && staticInterfaceAssertionSupportedTypeForLower(asserted) && staticInterfaceAssertionPanicContextForLower(toks, i, assertClose) {
			return true
		}
	}
	return false
}

func packagePanicRecoverDecls(importPath string, names panicRecoverNames) []unit.Decl {
	return []unit.Decl{
		{
			Path:     "rtg-panic",
			Kind:     "var",
			Name:     "__rtg_panic_active",
			UnitName: names.active,
			Body:     arena.PersistString("var " + names.active + " bool\n"),
		},
		{
			Path:     "rtg-panic",
			Kind:     "var",
			Name:     "__rtg_panic_value",
			UnitName: names.value,
			Body:     arena.PersistString("var " + names.value + " string\n"),
		},
		{
			Path:     "rtg-panic",
			Kind:     "func",
			Name:     "__rtg_recover",
			UnitName: names.recover,
			Body:     arena.PersistString(packageRecoverFunctionBody(names)),
		},
		{
			Path:     "rtg-panic",
			Kind:     "func",
			Name:     "__rtg_panic_abort",
			UnitName: names.abort,
			Body:     arena.PersistString(packagePanicAbortFunctionBody(names)),
		},
	}
}

func packageRecoverFunctionBody(names panicRecoverNames) string {
	body := "func " + names.recover + "() string {\n"
	body = body + "\tif " + names.active + " {\n"
	body = body + "\t\t" + names.active + " = false\n"
	body = body + "\t\treturn " + names.value + "\n"
	body = body + "\t}\n"
	body = body + "\treturn \"\"\n"
	body = body + "}\n"
	return body
}

func packagePanicAbortFunctionBody(names panicRecoverNames) string {
	body := "func " + names.abort + "() int {\n"
	body = body + "\tprint(\"panic: \")\n"
	body = body + "\tprint(" + names.value + ")\n"
	body = body + "\tprint(\"\\n\")\n"
	body = body + "\t" + names.active + " = false\n"
	body = body + "\treturn 2\n"
	body = body + "}\n"
	return body
}

func lowerDeferStatements(body string, unitName string, panicRecover panicRecoverNames, abortUnrecovered bool, ctx normalizationContext) string {
	if panicRecover.active == "" && !strings.Contains(body, "defer") && !strings.Contains(body, "panic") && !strings.Contains(body, "recover") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	open, close, ok := functionBodyRange(toks)
	if !ok {
		return body
	}
	hasResults := functionHasResults(toks, open)
	var deferred []loweredDeferCall
	tempIndex := 0
	dynamicPlan := dynamicDeferPlanForFunction(body, toks, open, close, unitName, &tempIndex, ctx)
	var nestedActivations []nestedDeferActivation
	if !dynamicPlan.enabled {
		nestedActivations = nestedDeferActivations(body, toks, open, close, unitName, &tempIndex, ctx)
	}
	panicReturnLabel := ""
	var pendingCloses []pendingPanicBlockClose
	out := make([]byte, 0, len(body)+128)
	cursor := 0
	if dynamicPlan.enabled {
		cursor = int(toks[open].End)
		out = appendBytes(out, []byte(body[:cursor]))
		lines := dynamicDeferPreludeLines(dynamicPlan)
		for i := 0; i < len(lines); i++ {
			out = append(out, '\n')
			out = append(out, '\t')
			out = appendString(out, lines[i])
		}
	} else if len(nestedActivations) > 0 {
		cursor = int(toks[open].End)
		out = appendBytes(out, []byte(body[:cursor]))
		for i := 0; i < len(nestedActivations); i++ {
			out = append(out, '\n')
			out = append(out, '\t')
			out = appendString(out, nestedActivations[i].active)
			out = appendString(out, " := false")
			for captureIndex := 0; captureIndex < len(nestedActivations[i].captures); captureIndex++ {
				capture := nestedActivations[i].captures[captureIndex]
				out = append(out, '\n')
				out = append(out, '\t')
				out = appendString(out, "var ")
				out = appendString(out, capture.name)
				out = append(out, ' ')
				out = appendString(out, capture.typ)
			}
		}
	}
	for i := open + 1; i < close; i++ {
		tok := toks[i]
		if tok.Text == "}" {
			closeIndex := pendingPanicBlockCloseIndex(pendingCloses, i)
			if closeIndex >= 0 {
				closeInfo := pendingCloses[closeIndex]
				if len(closeInfo.beforeClose) > 0 {
					closeLineStart := int(tok.Start)
					for closeLineStart > cursor && body[closeLineStart-1] != '\n' {
						closeLineStart--
					}
					out = appendBytes(out, []byte(body[cursor:closeLineStart]))
					if len(out) > 0 && out[len(out)-1] != '\n' {
						out = append(out, '\n')
					}
					beforeIndent := closeInfo.indent + "\t"
					out = appendString(out, beforeIndent)
					out = appendString(out, strings.Join(closeInfo.beforeClose, "\n"+beforeIndent))
					out = append(out, '\n')
					out = appendBytes(out, []byte(body[closeLineStart:int(tok.End)]))
				} else {
					out = appendBytes(out, []byte(body[cursor:int(tok.End)]))
				}
				out = append(out, '\n')
				out = appendString(out, closeInfo.indent)
				out = append(out, '}')
				cursor = int(tok.End)
				pendingCloses = append(pendingCloses[:closeIndex], pendingCloses[closeIndex+1:]...)
				continue
			}
		}
		if tok.Text == "defer" {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			if dynamicPlan.enabled {
				lines, ok := lowerDynamicDeferSite(body, toks, i, stmtEnd, unitName, dynamicPlan, panicRecover, abortUnrecovered, &tempIndex)
				if !ok {
					continue
				}
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(lines, "\n"+indent))
				cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
				i = stmtEnd - 1
				continue
			}
			activation, nested := nestedDeferActivationForPos(nestedActivations, i)
			lines, call, ok := lowerDeferCall(body, toks, i, stmtEnd, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex, activation, nested)
			if !ok {
				continue
			}
			out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
			indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
			if len(lines) > 0 {
				out = appendString(out, strings.Join(lines, "\n"+indent))
			}
			if nested {
				if len(lines) > 0 {
					out = append(out, '\n')
					out = appendString(out, indent)
				}
				out = appendString(out, activation.active)
				out = appendString(out, " = true")
			}
			deferred = append(deferred, loweredDeferCall{call: call, active: activation.active, captures: activation.captures})
			cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
			i = stmtEnd - 1
			continue
		}
		if tok.Text == "panic" && startsDirectCallAt(toks, i, close) {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			if panicReturnLabel == "" {
				panicReturnLabel = unitName + "_panic_return"
			}
			var replacement []string
			var ok bool
			if dynamicPlan.enabled {
				replacement, ok = lowerPanicWithDynamicDefers(body, toks, i, stmtEnd, unitName, panicRecover, panicReturnLabel, dynamicPlan, &tempIndex)
			} else {
				replacement, ok = lowerPanicWithDefers(body, toks, i, stmtEnd, unitName, panicRecover, panicReturnLabel, deferred, &tempIndex)
			}
			if !ok {
				continue
			}
			out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
			indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
			out = appendString(out, strings.Join(replacement, "\n"+indent))
			cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
			i = stmtEnd - 1
			continue
		}
		if tok.Text == "return" && (panicRecover.active != "" || len(deferred) > 0 || dynamicPlan.enabled) {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			var replacement []string
			var ok bool
			if panicRecover.active != "" {
				replacement, ok = lowerReturnWithPanicChecks(body, toks, i, stmtEnd, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			}
			if !ok && dynamicPlan.enabled {
				replacement = lowerReturnWithDynamicDefers(body, toks, i, stmtEnd, unitName, dynamicPlan, &tempIndex)
				ok = true
			} else if !ok && len(deferred) > 0 {
				replacement = lowerReturnWithDefers(body, toks, i, stmtEnd, unitName, deferred, &tempIndex)
				ok = true
			}
			if !ok {
				continue
			}
			out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
			indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
			out = appendString(out, strings.Join(replacement, "\n"+indent))
			cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
			i = stmtEnd - 1
			continue
		}
		if panicRecover.active != "" && (tok.Text == "if" || tok.Text == "switch") {
			replacement, openBrace, ok := lowerBranchConditionDirectCallWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				cursor = int(toks[openBrace].Start)
				i = openBrace - 1
				continue
			}
		}
		if panicRecover.active != "" && tok.Text == "for" {
			replacement, beforeClose, openBrace, closeBrace, ok := lowerClassicForCombinedHeaderDirectCallsWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				pendingCloses = append(pendingCloses, pendingPanicBlockClose{token: closeBrace, indent: indent, beforeClose: beforeClose})
				cursor = int(toks[openBrace].End)
				i = openBrace
				continue
			}
			replacement, beforeClose, openBrace, closeBrace, ok = lowerClassicForPostDirectCallWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				pendingCloses = append(pendingCloses, pendingPanicBlockClose{token: closeBrace, indent: indent, beforeClose: beforeClose})
				cursor = int(toks[openBrace].End)
				i = openBrace
				continue
			}
			replacement, openBrace, closeBrace, ok = lowerClassicForInitDirectCallWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				pendingCloses = append(pendingCloses, pendingPanicBlockClose{token: closeBrace, indent: indent})
				cursor = int(toks[openBrace].Start)
				i = openBrace - 1
				continue
			}
			replacement, openBrace, ok = lowerClassicForConditionDirectCallWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				cursor = int(toks[openBrace].End)
				i = openBrace
				continue
			}
			replacement, openBrace, ok = lowerForConditionDirectCallWithPanic(body, toks, i, close, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				cursor = int(toks[openBrace].End)
				i = openBrace
				continue
			}
		}
		if panicRecover.active != "" {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			replacement, ok := lowerAssignmentStatementWithPanicOperands(body, toks, i, stmtEnd, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
				i = stmtEnd - 1
				continue
			}
		}
		if panicRecover.active != "" && startsCallStatement(toks, i) {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			replacement, ok := lowerDirectCallStatementWithPanicArgument(body, toks, i, stmtEnd, unitName, panicRecover, dynamicPlan, deferred, abortUnrecovered, &tempIndex)
			if ok {
				out = appendBytes(out, []byte(body[cursor:int(tok.Start)]))
				indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
				out = appendString(out, strings.Join(replacement, "\n"+indent))
				cursor = lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
				i = stmtEnd - 1
				continue
			}
		}
		if panicRecover.active != "" && startsCallStatement(toks, i) {
			stmtEnd := lowerSimpleStatementEnd(toks, i, close)
			lines := lowerPanicPropagationAfterCallLines(toks, panicRecover, dynamicPlan, deferred, abortUnrecovered)
			if len(lines) == 0 {
				continue
			}
			stmtSourceEnd := lowerStatementSourceEnd(toks, stmtEnd, int(tok.End))
			out = appendBytes(out, []byte(body[cursor:stmtSourceEnd]))
			if stmtSourceEnd > 0 && body[stmtSourceEnd-1] != '\n' {
				out = append(out, '\n')
			}
			indent := fallthroughLabelIndent([]byte(body), int(tok.Start))
			out = appendString(out, indent)
			out = appendString(out, strings.Join(lines, "\n"+indent))
			cursor = stmtSourceEnd
			i = stmtEnd - 1
			continue
		}
	}
	out = appendBytes(out, []byte(body[cursor:int(toks[close].Start)]))
	if dynamicPlan.enabled && !hasResults {
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		lines := appendDynamicDeferredCalls(nil, dynamicPlan)
		for i := 0; i < len(lines); i++ {
			out = append(out, '\t')
			out = appendString(out, lines[i])
			out = append(out, '\n')
		}
	} else if len(deferred) > 0 && !hasResults {
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		lines := appendDeferredCalls(nil, deferred)
		for i := 0; i < len(lines); i++ {
			out = append(out, '\t')
			out = appendString(out, lines[i])
			out = append(out, '\n')
		}
	}
	if panicReturnLabel != "" {
		if len(out) > 0 && out[len(out)-1] != '\n' {
			out = append(out, '\n')
		}
		out = append(out, '\t')
		out = appendString(out, panicReturnLabel)
		out = appendString(out, ":\n")
		tailLines := panicReturnTailLines(toks, panicRecover, abortUnrecovered)
		for i := 0; i < len(tailLines); i++ {
			out = append(out, '\t')
			out = appendString(out, tailLines[i])
			out = append(out, '\n')
		}
	}
	out = appendBytes(out, []byte(body[int(toks[close].Start):]))
	return rewriteRecoverCalls(string(out), panicRecover)
}

func functionHasResults(toks []scan.Token, bodyOpen int) bool {
	paramsOpen := -1
	for i := 0; i < bodyOpen; i++ {
		if toks[i].Text == "func" {
			paramsOpen = findTokenText(toks, i+1, int(toks[bodyOpen].Start), "(")
			break
		}
	}
	if paramsOpen < 0 {
		return false
	}
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= bodyOpen {
		return false
	}
	for i := paramsClose + 1; i < bodyOpen; i++ {
		if toks[i].Text != ";" {
			return true
		}
	}
	return false
}

func functionBodyRange(toks []scan.Token) (int, int, bool) {
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "{" {
			continue
		}
		close := findClose(toks, i, "{", "}")
		if close > i {
			return i, close, true
		}
		return 0, 0, false
	}
	return 0, 0, false
}

func dynamicDeferPlanForFunction(body string, toks []scan.Token, bodyOpen int, bodyClose int, unitName string, tempIndex *int, ctx normalizationContext) dynamicDeferPlan {
	if !functionHasLoopDefer(toks, bodyOpen, bodyClose) {
		return dynamicDeferPlan{}
	}
	plan := dynamicDeferPlan{
		enabled: true,
		order:   nextExpressionTempName(body, unitName+"_defer_order", tempIndex),
	}
	(*tempIndex)++
	plan.index = nextExpressionTempName(body, unitName+"_defer_index", tempIndex)
	(*tempIndex)++
	plan.kind = nextExpressionTempName(body, unitName+"_defer_kind", tempIndex)
	(*tempIndex)++
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "defer" {
			continue
		}
		stmtEnd := lowerSimpleStatementEnd(toks, i, bodyClose)
		site, ok := dynamicDeferSiteForStatement(body, toks, i, stmtEnd, len(plan.sites), unitName, tempIndex, ctx)
		if ok {
			plan.sites = append(plan.sites, site)
		}
	}
	return plan
}

func functionHasLoopDefer(toks []scan.Token, bodyOpen int, bodyClose int) bool {
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text == "defer" && deferIsInsideLoopInLower(toks, bodyOpen, i) {
			return true
		}
	}
	return false
}

func dynamicDeferSiteForStatement(body string, toks []scan.Token, pos int, stmtEnd int, id int, unitName string, tempIndex *int, ctx normalizationContext) (dynamicDeferSite, bool) {
	open := lowerDeferDirectCallOpen(toks, pos, stmtEnd)
	if open < 0 {
		return dynamicDeferSite{}, false
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return dynamicDeferSite{}, false
	}
	site := dynamicDeferSite{
		pos:    pos,
		id:     id,
		callee: strings.TrimSpace(body[int(toks[pos+1].Start):int(toks[open].Start)]),
	}
	ranges := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(ranges); i++ {
		start, end := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if start >= end {
			continue
		}
		expanded := hasVariadicExpansionInTokens(toks, start, end)
		if expanded {
			end--
		}
		typ := deferCaptureTypeText(body, toks, start, end, expanded, ctx)
		if typ == "" {
			return dynamicDeferSite{}, false
		}
		name := nextExpressionTempName(body, unitName+"_defer_value", tempIndex)
		(*tempIndex)++
		stack := nextExpressionTempName(body, unitName+"_defer_stack", tempIndex)
		(*tempIndex)++
		site.captures = append(site.captures, dynamicDeferCapture{name: name, stack: stack, typ: typ, expanded: expanded})
	}
	return site, true
}

func dynamicDeferPreludeLines(plan dynamicDeferPlan) []string {
	var lines []string
	lines = append(lines, plan.order+" := []int{}")
	for siteIndex := 0; siteIndex < len(plan.sites); siteIndex++ {
		site := plan.sites[siteIndex]
		for captureIndex := 0; captureIndex < len(site.captures); captureIndex++ {
			capture := site.captures[captureIndex]
			lines = append(lines, "var "+capture.stack+" []"+capture.typ+" = nil")
		}
	}
	return lines
}

func lowerDynamicDeferSite(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, plan dynamicDeferPlan, names panicRecoverNames, abortUnrecovered bool, tempIndex *int) ([]string, bool) {
	site, ok := dynamicDeferSiteForPos(plan, pos)
	if !ok {
		return nil, false
	}
	open := lowerDeferDirectCallOpen(toks, pos, stmtEnd)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return nil, false
	}
	ranges := topLevelExpressionRanges(toks, open+1, close)
	if len(ranges) != len(site.captures) {
		return nil, false
	}
	var lines []string
	for i := 0; i < len(ranges); i++ {
		start, end := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if start >= end {
			continue
		}
		if hasVariadicExpansionInTokens(toks, start, end) {
			end--
			start, end = trimTokenRange(toks, start, end)
			if start >= end {
				return nil, false
			}
		}
		expr := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
		capture := site.captures[i]
		if names.active != "" {
			directCall := expressionIsDirectCallWithoutCallArgs(toks, start, end)
			if !directCall && expressionContainsCall(toks, start, end) {
				exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, start, end, unitName+"_defer", names, plan, nil, abortUnrecovered, tempIndex)
				if !ok {
					return nil, false
				}
				lines = append(lines, exprLines...)
				lines = append(lines, capture.stack+" = append("+capture.stack+", "+temp+")")
				continue
			}
			if directCall {
				temp := nextExpressionTempName(body, unitName+"_defer", tempIndex)
				(*tempIndex)++
				lines = append(lines, temp+" := "+expr)
				lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, nil, abortUnrecovered)...)
				lines = append(lines, capture.stack+" = append("+capture.stack+", "+temp+")")
				continue
			}
		}
		lines = append(lines, capture.stack+" = append("+capture.stack+", "+expr+")")
	}
	lines = append(lines, plan.order+" = append("+plan.order+", "+strconv.Itoa(site.id)+")")
	return lines, true
}

func dynamicDeferSiteForPos(plan dynamicDeferPlan, pos int) (dynamicDeferSite, bool) {
	for i := 0; i < len(plan.sites); i++ {
		if plan.sites[i].pos == pos {
			return plan.sites[i], true
		}
	}
	return dynamicDeferSite{}, false
}

func nestedDeferActivations(body string, toks []scan.Token, bodyOpen int, bodyClose int, unitName string, tempIndex *int, ctx normalizationContext) []nestedDeferActivation {
	var out []nestedDeferActivation
	index := 0
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "defer" {
			continue
		}
		if deferIsNestedInFunctionBody(toks, bodyOpen, i) {
			stmtEnd := lowerSimpleStatementEnd(toks, i, bodyClose)
			captures := nestedDeferCaptures(body, toks, i, stmtEnd, unitName, tempIndex, ctx)
			out = append(out, nestedDeferActivation{
				pos:      i,
				active:   unitName + "_defer_active_" + strconv.Itoa(index),
				captures: captures,
			})
			index++
		}
	}
	return out
}

func nestedDeferActivationForPos(activations []nestedDeferActivation, pos int) (nestedDeferActivation, bool) {
	for i := 0; i < len(activations); i++ {
		if activations[i].pos == pos {
			return activations[i], true
		}
	}
	return nestedDeferActivation{}, false
}

func nestedDeferCaptures(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, tempIndex *int, ctx normalizationContext) []deferCapture {
	open := lowerDeferDirectCallOpen(toks, pos, stmtEnd)
	if open < 0 {
		return nil
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return nil
	}
	ranges := topLevelExpressionRanges(toks, open+1, close)
	captures := make([]deferCapture, 0, len(ranges))
	for i := 0; i < len(ranges); i++ {
		start, end := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if start >= end {
			continue
		}
		expanded := hasVariadicExpansionInTokens(toks, start, end)
		if expanded {
			end--
		}
		typ := deferCaptureTypeText(body, toks, start, end, expanded, ctx)
		name := nextExpressionTempName(body, unitName+"_defer_capture", tempIndex)
		(*tempIndex)++
		captures = append(captures, deferCapture{name: name, typ: typ})
	}
	return captures
}

func deferCaptureTypeText(body string, toks []scan.Token, start int, end int, expanded bool, ctx normalizationContext) string {
	info := localInitializerTypeWithFunctions(toks, start, end, ctx.localTypes, ctx.functionResults)
	if info.name != "" {
		text := localTypeInfoText(info)
		if expanded && !strings.HasPrefix(text, "[]") {
			return "[]" + text
		}
		return text
	}
	text := packageInitializerValueTypeText(body, toks, expressionRange{start: start, end: end}, ctx.functionResults)
	if expanded && text != "" && !strings.HasPrefix(text, "[]") {
		return "[]" + text
	}
	return text
}

func deferIsNestedInFunctionBody(toks []scan.Token, bodyOpen int, pos int) bool {
	depth := 0
	for i := bodyOpen + 1; i < pos && i < len(toks); i++ {
		if toks[i].Text == "{" {
			depth++
			continue
		}
		if toks[i].Text == "}" && depth > 0 {
			depth--
		}
	}
	return depth > 0
}

func deferIsInsideLoopInLower(toks []scan.Token, bodyOpen int, pos int) bool {
	for i := pos - 1; i > bodyOpen; i-- {
		if toks[i].Text != "{" {
			continue
		}
		close := findClose(toks, i, "{", "}")
		if close <= pos {
			continue
		}
		if blockOwnerKeywordInLower(toks, i) == "for" {
			return true
		}
	}
	return false
}

func blockOwnerKeywordInLower(toks []scan.Token, open int) string {
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

func lowerDeferCall(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int, activation nestedDeferActivation, nested bool) ([]string, string, bool) {
	open := lowerDeferDirectCallOpen(toks, pos, stmtEnd)
	if open < 0 {
		return nil, "", false
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return nil, "", false
	}
	callee := strings.TrimSpace(body[int(toks[pos+1].Start):int(toks[open].Start)])
	ranges := topLevelExpressionRanges(toks, open+1, close)
	if nested && len(ranges) != len(activation.captures) {
		return nil, "", false
	}
	var lines []string
	var args []string
	for i := 0; i < len(ranges); i++ {
		start, end := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if start >= end {
			continue
		}
		expanded := hasVariadicExpansionInTokens(toks, start, end)
		if expanded {
			end--
			start, end = trimTokenRange(toks, start, end)
			if start >= end {
				return nil, "", false
			}
		}
		expr := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
		if names.active != "" {
			directCall := expressionIsDirectCallWithoutCallArgs(toks, start, end)
			if !directCall && expressionContainsCall(toks, start, end) {
				exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, start, end, unitName+"_defer", names, plan, deferred, abortUnrecovered, tempIndex)
				if !ok {
					return nil, "", false
				}
				lines = append(lines, exprLines...)
				arg := temp
				if nested {
					capture := activation.captures[i]
					if capture.name == "" || capture.typ == "" {
						return nil, "", false
					}
					lines = append(lines, capture.name+" = "+temp)
					arg = capture.name
				}
				if expanded {
					arg = arg + "..."
				}
				args = append(args, arg)
				continue
			}
			if directCall {
				temp := nextExpressionTempName(body, unitName+"_defer", tempIndex)
				(*tempIndex)++
				lines = append(lines, temp+" := "+expr)
				lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
				arg := temp
				if nested {
					capture := activation.captures[i]
					if capture.name == "" || capture.typ == "" {
						return nil, "", false
					}
					lines = append(lines, capture.name+" = "+temp)
					arg = capture.name
				}
				if expanded {
					arg = arg + "..."
				}
				args = append(args, arg)
				continue
			}
		}
		if nested {
			capture := activation.captures[i]
			if capture.name == "" || capture.typ == "" {
				return nil, "", false
			}
			lines = append(lines, capture.name+" = "+expr)
			arg := capture.name
			if expanded {
				arg = arg + "..."
			}
			args = append(args, arg)
			continue
		}
		temp := nextExpressionTempName(body, unitName+"_defer", tempIndex)
		(*tempIndex)++
		lines = append(lines, temp+" := "+expr)
		arg := temp
		if expanded {
			arg = arg + "..."
		}
		args = append(args, arg)
	}
	return lines, callee + "(" + strings.Join(args, ", ") + ")", true
}

func hasVariadicExpansionInTokens(toks []scan.Token, start int, end int) bool {
	return end > start && toks[end-1].Text == "..."
}

func lowerPanicWithDefers(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, returnLabel string, deferred []loweredDeferCall, tempIndex *int) ([]string, bool) {
	if names.active == "" || names.value == "" {
		return nil, false
	}
	open := pos + 1
	if open >= stmtEnd || toks[open].Text != "(" {
		return nil, false
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return nil, false
	}
	args := topLevelExpressionRanges(toks, open+1, close)
	if len(args) != 1 {
		return nil, false
	}
	argStart, argEnd := trimTokenRange(toks, args[0].start, args[0].end)
	if argStart >= argEnd {
		return nil, false
	}
	temp := nextExpressionTempName(body, unitName+"_panic", tempIndex)
	(*tempIndex)++
	expr := strings.TrimSpace(body[int(toks[argStart].Start):int(toks[argEnd-1].End)])
	var lines []string
	lines = append(lines, temp+" := "+expr)
	lines = append(lines, names.active+" = true")
	lines = append(lines, names.value+" = "+temp)
	lines = appendDeferredCalls(lines, deferred)
	lines = append(lines, "goto "+returnLabel)
	return lines, true
}

func lowerPanicWithDynamicDefers(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, returnLabel string, plan dynamicDeferPlan, tempIndex *int) ([]string, bool) {
	if names.active == "" || names.value == "" {
		return nil, false
	}
	open := pos + 1
	if open >= stmtEnd || toks[open].Text != "(" {
		return nil, false
	}
	close := findClose(toks, open, "(", ")")
	if close < 0 || close > stmtEnd {
		return nil, false
	}
	args := topLevelExpressionRanges(toks, open+1, close)
	if len(args) != 1 {
		return nil, false
	}
	argStart, argEnd := trimTokenRange(toks, args[0].start, args[0].end)
	if argStart >= argEnd {
		return nil, false
	}
	temp := nextExpressionTempName(body, unitName+"_panic", tempIndex)
	(*tempIndex)++
	expr := strings.TrimSpace(body[int(toks[argStart].Start):int(toks[argEnd-1].End)])
	var lines []string
	lines = append(lines, temp+" := "+expr)
	lines = append(lines, names.active+" = true")
	lines = append(lines, names.value+" = "+temp)
	lines = appendDynamicDeferredCalls(lines, plan)
	lines = append(lines, "goto "+returnLabel)
	return lines, true
}

func panicReturnTail(toks []scan.Token) string {
	open, _, ok := functionBodyRange(toks)
	if !ok {
		return ""
	}
	results := functionResultTypeTexts(toks, open)
	if len(results) == 0 {
		return ""
	}
	values := make([]string, 0, len(results))
	for i := 0; i < len(results); i++ {
		values = append(values, zeroValueForReturnType(results[i]))
	}
	return "return " + strings.Join(values, ", ")
}

func panicReturnTailLines(toks []scan.Token, names panicRecoverNames, abortUnrecovered bool) []string {
	tail := panicReturnTail(toks)
	if abortUnrecovered && names.active != "" && names.abort != "" && panicAbortTailSupported(toks) {
		lines := []string{"if " + names.active + " {", "\treturn " + names.abort + "()", "}"}
		if tail != "" {
			lines = append(lines, tail)
		}
		return lines
	}
	if tail != "" {
		return []string{tail}
	}
	return nil
}

func panicAbortTailSupported(toks []scan.Token) bool {
	open, _, ok := functionBodyRange(toks)
	if !ok {
		return false
	}
	results := functionResultTypeTexts(toks, open)
	return len(results) == 1 && results[0] == "int"
}

func functionResultTypeTexts(toks []scan.Token, bodyOpen int) []string {
	paramsOpen := -1
	for i := 0; i < bodyOpen; i++ {
		if toks[i].Text == "func" {
			paramsOpen = findTokenText(toks, i+1, int(toks[bodyOpen].Start), "(")
			break
		}
	}
	if paramsOpen < 0 {
		return nil
	}
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= bodyOpen {
		return nil
	}
	start := paramsClose + 1
	for start < bodyOpen && toks[start].Text == ";" {
		start++
	}
	if start >= bodyOpen {
		return nil
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close < 0 || close >= bodyOpen {
			return nil
		}
		ranges := topLevelExpressionRanges(toks, start+1, close)
		out := make([]string, 0, len(ranges))
		for i := 0; i < len(ranges); i++ {
			typ := resultFieldTypeText(toks, ranges[i].start, ranges[i].end)
			if typ != "" {
				out = append(out, typ)
			}
		}
		return out
	}
	typ := tokenTextWithSpaces(toks, start, bodyOpen)
	if typ == "" {
		return nil
	}
	return []string{typ}
}

func resultFieldTypeText(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if end-start >= 2 && toks[start].Kind == scan.Ident && !isBuiltinScalarTypeName(toks[start].Text) && toks[start+1].Kind == scan.Ident {
		start++
	}
	return tokenTextWithSpaces(toks, start, end)
}

func tokenTextWithSpaces(toks []scan.Token, start int, end int) string {
	out := ""
	for i := start; i < end; i++ {
		if out != "" && tokenTextNeedsSpace(toks[i-1].Text, toks[i].Text) {
			out += " "
		}
		out += toks[i].Text
	}
	return out
}

func tokenTextNeedsSpace(left string, right string) bool {
	if left == "*" || left == "[]" || right == "." || left == "." || right == ")" || right == "]" || right == "}" {
		return false
	}
	if right == "," || left == "(" || left == "[" || left == "{" {
		return false
	}
	return true
}

func zeroValueForReturnType(typ string) string {
	typ = strings.TrimSpace(typ)
	switch typ {
	case "bool":
		return "false"
	case "string", "error":
		return "\"\""
	case "int", "int16", "int32", "int64", "byte", "float64":
		return "0"
	}
	if strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return "nil"
	}
	if typ == "" {
		return "0"
	}
	return typ + "{}"
}

func rewriteRecoverCalls(body string, names panicRecoverNames) string {
	if names.recover == "" || !strings.Contains(body, "recover") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	out := make([]byte, 0, len(body))
	cursor := 0
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "recover" || toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && toks[i-1].Text == "." {
			continue
		}
		out = appendStringRange(out, body, cursor, int(toks[i].Start))
		out = appendString(out, names.recover)
		cursor = int(toks[i].End)
	}
	out = appendStringRange(out, body, cursor, len(body))
	return string(out)
}

func startsDirectCallAt(toks []scan.Token, pos int, limit int) bool {
	if pos+1 >= limit || pos+1 >= len(toks) {
		return false
	}
	if toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return false
	}
	close := findClose(toks, pos+1, "(", ")")
	return close >= pos+1 && close < limit
}

func startsSimpleAssignmentCallStatement(toks []scan.Token, pos int, limit int) bool {
	if pos >= limit || statementStartToken(toks, pos) != pos {
		return false
	}
	switch toks[pos].Text {
	case "return", "if", "for", "switch", "defer", "panic", "go", "select":
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := simpleStatementAssignmentOperator(toks, pos, stmtEnd)
	if assign < 0 {
		return false
	}
	start, end := trimTokenRange(toks, assign+1, stmtEnd)
	return expressionIsDirectCallWithoutCallArgs(toks, start, end)
}

func lowerAssignmentStatementWithPanicOperands(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool) {
	if pos >= stmtEnd || statementStartToken(toks, pos) != pos {
		return nil, false
	}
	if isInsideClassicForHeader(toks, pos) || isInsideConditionalShortHeader(toks, pos) {
		return nil, false
	}
	switch toks[pos].Text {
	case "return", "if", "for", "switch", "defer", "panic", "go", "select":
		return nil, false
	}
	assign := simpleStatementAssignmentOperator(toks, pos, stmtEnd)
	if assign < 0 {
		return nil, false
	}
	ranges := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(ranges) == 0 {
		return nil, false
	}
	lhsRanges := topLevelExpressionRanges(toks, pos, assign)
	if len(ranges) == 1 && len(lhsRanges) > 1 {
		exprStart, exprEnd := trimTokenRange(toks, ranges[0].start, ranges[0].end)
		if exprStart < exprEnd && expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd) {
			prefix := strings.TrimRight(body[int(toks[pos].Start):int(toks[assign].End)], " \t")
			expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
			lines := []string{prefix + " " + expr}
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			return lines, true
		}
		if exprStart < exprEnd && expressionContainsCall(toks, exprStart, exprEnd) {
			lines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, exprStart, exprEnd, unitName+"_panic_assign", names, plan, deferred, abortUnrecovered, tempIndex)
			if ok {
				prefix := strings.TrimRight(body[int(toks[pos].Start):int(toks[assign].End)], " \t")
				lines = append(lines, prefix+" "+temp)
				return lines, true
			}
		}
	}
	var lines []string
	var values []string
	hasPanicOperand := false
	for i := 0; i < len(ranges); i++ {
		exprStart, exprEnd := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if exprStart >= exprEnd {
			continue
		}
		directCall := expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd)
		if !directCall && expressionContainsCall(toks, exprStart, exprEnd) {
			exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, exprStart, exprEnd, unitName+"_panic_assign", names, plan, deferred, abortUnrecovered, tempIndex)
			if !ok {
				return nil, false
			}
			hasPanicOperand = true
			lines = append(lines, exprLines...)
			values = append(values, temp)
			continue
		}
		if directCall {
			hasPanicOperand = true
			temp := nextExpressionTempName(body, unitName+"_panic_assign", tempIndex)
			(*tempIndex)++
			expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
			lines = append(lines, temp+" := "+expr)
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			values = append(values, temp)
			continue
		}
		values = append(values, strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)]))
	}
	if !hasPanicOperand {
		return nil, false
	}
	prefix := strings.TrimRight(body[int(toks[pos].Start):int(toks[assign].End)], " \t")
	lines = append(lines, prefix+" "+strings.Join(values, ", "))
	return lines, true
}

func simpleStatementAssignmentOperator(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && isAssignmentOperator(toks[i].Text) {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func expressionIsDirectCallWithoutCallArgs(toks []scan.Token, start int, end int) bool {
	start, end = trimOuterParens(toks, start, end)
	if start+2 > end || toks[end-1].Text != ")" {
		return false
	}
	open := findOpen(toks, end-1, "(", ")")
	if open <= start || open >= end-1 {
		return false
	}
	if !callCalleeTokensSupportedForPanicPropagation(toks, start, open) {
		return false
	}
	return !expressionContainsCall(toks, open+1, end-1)
}

func callCalleeTokensSupportedForPanicPropagation(toks []scan.Token, start int, end int) bool {
	if start >= end || toks[start].Kind != scan.Ident || isBuiltinConversionName(toks[start].Text) {
		return false
	}
	i := start + 1
	for i < end {
		if i+1 >= end || toks[i].Text != "." || toks[i+1].Kind != scan.Ident {
			return false
		}
		i += 2
	}
	return true
}

func pendingPanicBlockCloseIndex(values []pendingPanicBlockClose, token int) int {
	for i := 0; i < len(values); i++ {
		if values[i].token == token {
			return i
		}
	}
	return -1
}

func lowerDirectCallStatementWithPanicArgument(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool) {
	if !startsCallStatement(toks, pos) || pos+1 >= stmtEnd || toks[pos+1].Text != "(" {
		return nil, false
	}
	close := findClose(toks, pos+1, "(", ")")
	if close <= pos+1 || close >= stmtEnd {
		return nil, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) == 0 {
		return nil, false
	}
	lines := []string{}
	var call []byte
	cursor := int(toks[pos].Start)
	hasPanicArg := false
	for i := 0; i < len(args); i++ {
		argStart, argEnd := trimTokenRange(toks, args[i].start, args[i].end)
		if argStart >= argEnd {
			continue
		}
		directCall := expressionIsDirectCallWithoutCallArgs(toks, argStart, argEnd)
		if !directCall && expressionContainsCall(toks, argStart, argEnd) {
			exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, argStart, argEnd, unitName+"_panic_arg", names, plan, deferred, abortUnrecovered, tempIndex)
			if !ok {
				return nil, false
			}
			hasPanicArg = true
			call = appendStringRange(call, body, cursor, int(toks[argStart].Start))
			lines = append(lines, exprLines...)
			call = appendString(call, temp)
			cursor = int(toks[argEnd-1].End)
			continue
		}
		call = appendStringRange(call, body, cursor, int(toks[argStart].Start))
		if directCall {
			hasPanicArg = true
			temp := nextExpressionTempName(body, unitName+"_panic_arg", tempIndex)
			(*tempIndex)++
			arg := strings.TrimSpace(body[int(toks[argStart].Start):int(toks[argEnd-1].End)])
			lines = append(lines, temp+" := "+arg)
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			call = appendString(call, temp)
		} else {
			call = appendStringRange(call, body, int(toks[argStart].Start), int(toks[argEnd-1].End))
		}
		cursor = int(toks[argEnd-1].End)
	}
	if !hasPanicArg {
		return nil, false
	}
	call = appendStringRange(call, body, cursor, int(toks[close].End))
	lines = append(lines, strings.TrimSpace(string(call)))
	lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
	return lines, true
}

func lowerDirectCallExpressionWithPanicArguments(body string, toks []scan.Token, start int, end int, tempPrefix string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, string, bool) {
	start, end = trimOuterParens(toks, start, end)
	if start+2 > end || toks[end-1].Text != ")" {
		return nil, "", false
	}
	open := findOpen(toks, end-1, "(", ")")
	if open <= start || open >= end-1 {
		return nil, "", false
	}
	if !callCalleeTokensSupportedForPanicPropagation(toks, start, open) {
		return nil, "", false
	}
	args := topLevelExpressionRanges(toks, open+1, end-1)
	if len(args) == 0 {
		return nil, "", false
	}
	lines := []string{}
	var call []byte
	cursor := int(toks[start].Start)
	hasPanicArg := false
	for i := 0; i < len(args); i++ {
		argStart, argEnd := trimTokenRange(toks, args[i].start, args[i].end)
		if argStart >= argEnd {
			continue
		}
		directCall := expressionIsDirectCallWithoutCallArgs(toks, argStart, argEnd)
		if !directCall && expressionContainsCall(toks, argStart, argEnd) {
			exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, argStart, argEnd, tempPrefix+"_arg", names, plan, deferred, abortUnrecovered, tempIndex)
			if !ok {
				return nil, "", false
			}
			hasPanicArg = true
			call = appendStringRange(call, body, cursor, int(toks[argStart].Start))
			lines = append(lines, exprLines...)
			call = appendString(call, temp)
			cursor = int(toks[argEnd-1].End)
			continue
		}
		call = appendStringRange(call, body, cursor, int(toks[argStart].Start))
		if directCall {
			hasPanicArg = true
			temp := nextExpressionTempName(body, tempPrefix+"_arg", tempIndex)
			(*tempIndex)++
			arg := strings.TrimSpace(body[int(toks[argStart].Start):int(toks[argEnd-1].End)])
			lines = append(lines, temp+" := "+arg)
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			call = appendString(call, temp)
		} else {
			call = appendStringRange(call, body, int(toks[argStart].Start), int(toks[argEnd-1].End))
		}
		cursor = int(toks[argEnd-1].End)
	}
	if !hasPanicArg {
		return nil, "", false
	}
	call = appendStringRange(call, body, cursor, int(toks[end-1].End))
	temp := nextExpressionTempName(body, tempPrefix, tempIndex)
	(*tempIndex)++
	lines = append(lines, temp+" := "+strings.TrimSpace(string(call)))
	lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
	return lines, temp, true
}

func lowerBranchConditionDirectCallWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, int, bool) {
	if pos >= limit || statementStartToken(toks, pos) != pos {
		return nil, 0, false
	}
	keyword := toks[pos].Text
	if keyword != "if" && keyword != "switch" {
		return nil, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, 0, false
	}
	start, end := trimTokenRange(toks, pos+1, openBrace)
	if !expressionIsDirectCallWithoutCallArgs(toks, start, end) {
		lines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, start, end, unitName+"_panic_cond", names, plan, deferred, abortUnrecovered, tempIndex)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, keyword+" "+temp)
		return lines, openBrace, true
	}
	temp := nextExpressionTempName(body, unitName+"_panic_cond", tempIndex)
	(*tempIndex)++
	condition := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	lines := []string{temp + " := " + condition}
	lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
	lines = append(lines, keyword+" "+temp)
	return lines, openBrace, true
}

func lowerForConditionDirectCallWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, int, bool) {
	if pos >= limit || toks[pos].Text != "for" || statementStartToken(toks, pos) != pos {
		return nil, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, 0, false
	}
	if expressionContainsTopLevelSemicolon(toks, pos+1, openBrace) {
		return nil, 0, false
	}
	start, end := trimTokenRange(toks, pos+1, openBrace)
	if !expressionIsDirectCallWithoutCallArgs(toks, start, end) {
		conditionLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, start, end, unitName+"_panic_cond", names, plan, deferred, abortUnrecovered, tempIndex)
		if !ok {
			return nil, 0, false
		}
		lines := []string{"for {"}
		for i := 0; i < len(conditionLines); i++ {
			lines = append(lines, "\t"+conditionLines[i])
		}
		lines = append(lines, "\tif !("+temp+") {")
		lines = append(lines, "\t\tbreak")
		lines = append(lines, "\t}")
		return lines, openBrace, true
	}
	temp := nextExpressionTempName(body, unitName+"_panic_cond", tempIndex)
	(*tempIndex)++
	condition := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	lines := []string{"for {"}
	lines = append(lines, "\t"+temp+" := "+condition)
	panicLines := lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)
	for i := 0; i < len(panicLines); i++ {
		lines = append(lines, "\t"+panicLines[i])
	}
	lines = append(lines, "\tif !("+temp+") {")
	lines = append(lines, "\t\tbreak")
	lines = append(lines, "\t}")
	return lines, openBrace, true
}

func lowerClassicForInitDirectCallWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, int, int, bool) {
	if pos >= limit || toks[pos].Text != "for" || statementStartToken(toks, pos) != pos {
		return nil, 0, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, 0, 0, false
	}
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace <= openBrace || closeBrace >= limit {
		return nil, 0, 0, false
	}
	firstSemi := topLevelSemicolon(toks, pos+1, openBrace)
	if firstSemi <= pos+1 {
		return nil, 0, 0, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, openBrace)
	if secondSemi < 0 {
		return nil, 0, 0, false
	}
	condStart, condEnd := trimTokenRange(toks, firstSemi+1, secondSemi)
	postStart, postEnd := trimTokenRange(toks, secondSemi+1, openBrace)
	if expressionContainsCall(toks, condStart, condEnd) || expressionContainsCall(toks, postStart, postEnd) {
		return nil, 0, 0, false
	}
	initLines, ok := lowerAssignmentRangeWithPanicOperands(body, toks, pos+1, firstSemi, unitName+"_panic_for", names, plan, deferred, abortUnrecovered, tempIndex)
	if !ok {
		return nil, 0, 0, false
	}
	lines := []string{"{"}
	for i := 0; i < len(initLines); i++ {
		lines = append(lines, "\t"+initLines[i])
	}
	header := "for " + strings.TrimLeft(body[int(toks[firstSemi].Start):int(toks[openBrace].Start)], " \t")
	lines = append(lines, "\t"+header)
	return lines, openBrace, closeBrace, true
}

func lowerClassicForPostDirectCallWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, []string, int, int, bool) {
	if pos >= limit || toks[pos].Text != "for" || statementStartToken(toks, pos) != pos {
		return nil, nil, 0, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, nil, 0, 0, false
	}
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace <= openBrace || closeBrace >= limit {
		return nil, nil, 0, 0, false
	}
	firstSemi := topLevelSemicolon(toks, pos+1, openBrace)
	if firstSemi < 0 {
		return nil, nil, 0, 0, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, openBrace)
	if secondSemi < 0 || secondSemi+1 >= openBrace {
		return nil, nil, 0, 0, false
	}
	initStart, initEnd := trimTokenRange(toks, pos+1, firstSemi)
	condStart, condEnd := trimTokenRange(toks, firstSemi+1, secondSemi)
	if expressionContainsCall(toks, initStart, initEnd) || expressionContainsCall(toks, condStart, condEnd) {
		return nil, nil, 0, 0, false
	}
	postLines, ok := lowerAssignmentRangeWithPanicOperands(body, toks, secondSemi+1, openBrace, unitName+"_panic_post", names, plan, deferred, abortUnrecovered, tempIndex)
	if !ok {
		return nil, nil, 0, 0, false
	}
	var lines []string
	lines = append(lines, "{")
	if initStart < initEnd {
		init := strings.TrimSpace(body[int(toks[initStart].Start):int(toks[initEnd-1].End)])
		lines = append(lines, "\t"+init)
	}
	lines = append(lines, "\tfor {")
	if condStart < condEnd {
		condition := strings.TrimSpace(body[int(toks[condStart].Start):int(toks[condEnd-1].End)])
		lines = append(lines, "\t\tif !("+condition+") {")
		lines = append(lines, "\t\t\tbreak")
		lines = append(lines, "\t\t}")
	}
	return lines, postLines, openBrace, closeBrace, true
}

func lowerClassicForConditionDirectCallWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, int, bool) {
	if pos >= limit || toks[pos].Text != "for" || statementStartToken(toks, pos) != pos {
		return nil, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, 0, false
	}
	firstSemi := topLevelSemicolon(toks, pos+1, openBrace)
	if firstSemi != pos+1 {
		return nil, 0, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, openBrace)
	if secondSemi <= firstSemi+1 || secondSemi+1 != openBrace {
		return nil, 0, false
	}
	start, end := trimTokenRange(toks, firstSemi+1, secondSemi)
	if !expressionIsDirectCallWithoutCallArgs(toks, start, end) {
		return nil, 0, false
	}
	temp := nextExpressionTempName(body, unitName+"_panic_cond", tempIndex)
	(*tempIndex)++
	condition := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	lines := []string{"for {"}
	lines = append(lines, "\t"+temp+" := "+condition)
	panicLines := lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)
	for i := 0; i < len(panicLines); i++ {
		lines = append(lines, "\t"+panicLines[i])
	}
	lines = append(lines, "\tif !("+temp+") {")
	lines = append(lines, "\t\tbreak")
	lines = append(lines, "\t}")
	return lines, openBrace, true
}

func lowerClassicForCombinedHeaderDirectCallsWithPanic(body string, toks []scan.Token, pos int, limit int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, []string, int, int, bool) {
	if pos >= limit || toks[pos].Text != "for" || statementStartToken(toks, pos) != pos {
		return nil, nil, 0, 0, false
	}
	openBrace := conditionExpressionEnd(toks, pos)
	if openBrace <= pos+1 || openBrace >= limit || toks[openBrace].Text != "{" {
		return nil, nil, 0, 0, false
	}
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace <= openBrace || closeBrace >= limit {
		return nil, nil, 0, 0, false
	}
	firstSemi := topLevelSemicolon(toks, pos+1, openBrace)
	if firstSemi < 0 {
		return nil, nil, 0, 0, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, openBrace)
	if secondSemi < 0 {
		return nil, nil, 0, 0, false
	}
	initStart, initEnd := trimTokenRange(toks, pos+1, firstSemi)
	condStart, condEnd := trimTokenRange(toks, firstSemi+1, secondSemi)
	postStart, postEnd := trimTokenRange(toks, secondSemi+1, openBrace)
	if postStart < postEnd && containsTokenText(toks, openBrace+1, closeBrace, "continue") {
		return nil, nil, 0, 0, false
	}
	localTempIndex := *tempIndex
	initLines, initHasCheck, ok := lowerClassicForSimpleStatementSectionWithPanic(body, toks, initStart, initEnd, unitName+"_panic_for", names, plan, deferred, abortUnrecovered, &localTempIndex)
	if !ok {
		return nil, nil, 0, 0, false
	}
	conditionLines, conditionHasCheck, ok := lowerClassicForConditionSectionWithPanic(body, toks, condStart, condEnd, unitName+"_panic_cond", names, plan, deferred, abortUnrecovered, &localTempIndex)
	if !ok {
		return nil, nil, 0, 0, false
	}
	postLines, postHasCheck, ok := lowerClassicForSimpleStatementSectionWithPanic(body, toks, postStart, postEnd, unitName+"_panic_post", names, plan, deferred, abortUnrecovered, &localTempIndex)
	if !ok {
		return nil, nil, 0, 0, false
	}
	checkSections := 0
	if initHasCheck {
		checkSections++
	}
	if conditionHasCheck {
		checkSections++
	}
	if postHasCheck {
		checkSections++
	}
	if checkSections == 0 {
		return nil, nil, 0, 0, false
	}
	if checkSections == 1 && !conditionHasCheck {
		return nil, nil, 0, 0, false
	}
	if conditionHasCheck && checkSections == 1 && initStart >= initEnd && postStart >= postEnd {
		return nil, nil, 0, 0, false
	}
	*tempIndex = localTempIndex
	lines := []string{"{"}
	for i := 0; i < len(initLines); i++ {
		lines = append(lines, "\t"+initLines[i])
	}
	lines = append(lines, "\tfor {")
	for i := 0; i < len(conditionLines); i++ {
		lines = append(lines, "\t\t"+conditionLines[i])
	}
	return lines, postLines, openBrace, closeBrace, true
}

func lowerClassicForSimpleStatementSectionWithPanic(body string, toks []scan.Token, start int, end int, tempPrefix string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool, bool) {
	if start >= end {
		return nil, false, true
	}
	if expressionContainsCall(toks, start, end) {
		lines, ok := lowerAssignmentRangeWithPanicOperands(body, toks, start, end, tempPrefix, names, plan, deferred, abortUnrecovered, tempIndex)
		return lines, ok, ok
	}
	text := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	if text == "" {
		return nil, false, true
	}
	return []string{text}, false, true
}

func lowerClassicForConditionSectionWithPanic(body string, toks []scan.Token, start int, end int, tempPrefix string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool, bool) {
	if start >= end {
		return nil, false, true
	}
	if expressionIsDirectCallWithoutCallArgs(toks, start, end) {
		temp := nextExpressionTempName(body, tempPrefix, tempIndex)
		(*tempIndex)++
		condition := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
		lines := []string{temp + " := " + condition}
		lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
		lines = append(lines, "if !("+temp+") {")
		lines = append(lines, "\tbreak")
		lines = append(lines, "}")
		return lines, true, true
	}
	if lines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, start, end, tempPrefix, names, plan, deferred, abortUnrecovered, tempIndex); ok {
		lines = append(lines, "if !("+temp+") {")
		lines = append(lines, "\tbreak")
		lines = append(lines, "}")
		return lines, true, true
	}
	if expressionContainsCall(toks, start, end) {
		return nil, false, false
	}
	condition := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	if condition == "" {
		return nil, false, true
	}
	return []string{"if !(" + condition + ") {", "\tbreak", "}"}, false, true
}

func lowerAssignmentRangeWithPanicOperands(body string, toks []scan.Token, start int, end int, tempPrefix string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool) {
	assign := simpleStatementAssignmentOperator(toks, start, end)
	if assign < 0 {
		return nil, false
	}
	ranges := topLevelExpressionRanges(toks, assign+1, end)
	if len(ranges) == 0 {
		return nil, false
	}
	lhsRanges := topLevelExpressionRanges(toks, start, assign)
	if len(ranges) == 1 && len(lhsRanges) > 1 {
		exprStart, exprEnd := trimTokenRange(toks, ranges[0].start, ranges[0].end)
		if exprStart < exprEnd && expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd) {
			prefix := strings.TrimRight(body[int(toks[start].Start):int(toks[assign].End)], " \t")
			expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
			lines := []string{prefix + " " + expr}
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			return lines, true
		}
	}
	var lines []string
	var values []string
	hasPanicOperand := false
	for i := 0; i < len(ranges); i++ {
		exprStart, exprEnd := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if exprStart >= exprEnd {
			continue
		}
		directCall := expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd)
		if !directCall && expressionContainsCall(toks, exprStart, exprEnd) {
			return nil, false
		}
		if directCall {
			hasPanicOperand = true
			temp := nextExpressionTempName(body, tempPrefix, tempIndex)
			(*tempIndex)++
			expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
			lines = append(lines, temp+" := "+expr)
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			values = append(values, temp)
			continue
		}
		values = append(values, strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)]))
	}
	if !hasPanicOperand {
		return nil, false
	}
	prefix := strings.TrimRight(body[int(toks[start].Start):int(toks[assign].End)], " \t")
	lines = append(lines, prefix+" "+strings.Join(values, ", "))
	return lines, true
}

func lowerReturnWithDefers(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, deferred []loweredDeferCall, tempIndex *int) []string {
	start, end := trimTokenRange(toks, pos+1, stmtEnd)
	var lines []string
	if start >= end || toks[start].Line != toks[pos].Line {
		lines = appendDeferredCalls(lines, deferred)
		return append(lines, "return")
	}
	ranges := topLevelExpressionRanges(toks, start, end)
	resultCount := returnResultCountForLower(toks)
	if len(ranges) == 1 && resultCount > 1 {
		exprStart, exprEnd := trimTokenRange(toks, ranges[0].start, ranges[0].end)
		line, names, ok := lowerSingleDirectMultiResultReturn(body, toks, exprStart, exprEnd, resultCount, unitName+"_return", tempIndex)
		if ok {
			lines = append(lines, line)
			lines = appendDeferredCalls(lines, deferred)
			return append(lines, "return "+strings.Join(names, ", "))
		}
	}
	var values []string
	for i := 0; i < len(ranges); i++ {
		exprStart, exprEnd := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if exprStart >= exprEnd {
			continue
		}
		temp := nextExpressionTempName(body, unitName+"_return", tempIndex)
		(*tempIndex)++
		expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
		lines = append(lines, temp+" := "+expr)
		values = append(values, temp)
	}
	lines = appendDeferredCalls(lines, deferred)
	if len(values) == 0 {
		return append(lines, "return")
	}
	return append(lines, "return "+strings.Join(values, ", "))
}

func lowerReturnWithPanicChecks(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool, tempIndex *int) ([]string, bool) {
	start, end := trimTokenRange(toks, pos+1, stmtEnd)
	if start >= end || toks[start].Line != toks[pos].Line {
		return nil, false
	}
	ranges := topLevelExpressionRanges(toks, start, end)
	if len(ranges) == 0 {
		return nil, false
	}
	resultCount := returnResultCountForLower(toks)
	if len(ranges) == 1 && resultCount == 0 {
		return nil, false
	}
	var lines []string
	var values []string
	hasPanicCheck := false
	for i := 0; i < len(ranges); i++ {
		exprStart, exprEnd := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if exprStart >= exprEnd {
			continue
		}
		directCall := expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd)
		if !directCall && expressionContainsCall(toks, exprStart, exprEnd) {
			if len(ranges) == 1 && resultCount > 1 {
				return nil, false
			}
			exprLines, temp, ok := lowerDirectCallExpressionWithPanicArguments(body, toks, exprStart, exprEnd, unitName+"_panic_return", names, plan, deferred, abortUnrecovered, tempIndex)
			if !ok {
				return nil, false
			}
			hasPanicCheck = true
			lines = append(lines, exprLines...)
			values = append(values, temp)
			continue
		}
		if len(ranges) == 1 && directCall && resultCount > 1 {
			line, tempNames, ok := lowerSingleDirectMultiResultReturn(body, toks, exprStart, exprEnd, resultCount, unitName+"_panic_return", tempIndex)
			if !ok {
				return nil, false
			}
			lines = append(lines, line)
			values = append(values, tempNames...)
			hasPanicCheck = true
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
			continue
		}
		temp := nextExpressionTempName(body, unitName+"_panic_return", tempIndex)
		(*tempIndex)++
		expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
		lines = append(lines, temp+" := "+expr)
		values = append(values, temp)
		if directCall {
			hasPanicCheck = true
			lines = append(lines, lowerPanicPropagationAfterCallLines(toks, names, plan, deferred, abortUnrecovered)...)
		}
	}
	if !hasPanicCheck {
		return nil, false
	}
	if plan.enabled {
		lines = appendDynamicDeferredCalls(lines, plan)
	} else {
		lines = appendDeferredCalls(lines, deferred)
	}
	if len(values) == 0 {
		return append(lines, "return"), true
	}
	return append(lines, "return "+strings.Join(values, ", ")), true
}

func returnResultCountForLower(toks []scan.Token) int {
	open, _, ok := functionBodyRange(toks)
	if !ok {
		return 0
	}
	return len(functionResultTypeTexts(toks, open))
}

func lowerSingleDirectMultiResultReturn(body string, toks []scan.Token, exprStart int, exprEnd int, resultCount int, tempPrefix string, tempIndex *int) (string, []string, bool) {
	if resultCount <= 1 || exprStart >= exprEnd {
		return "", nil, false
	}
	if !expressionIsDirectCallWithoutCallArgs(toks, exprStart, exprEnd) {
		return "", nil, false
	}
	names := make([]string, 0, resultCount)
	for i := 0; i < resultCount; i++ {
		temp := nextExpressionTempName(body, tempPrefix, tempIndex)
		(*tempIndex)++
		names = append(names, temp)
	}
	expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
	return strings.Join(names, ", ") + " := " + expr, names, true
}

func appendDeferredCalls(lines []string, deferred []loweredDeferCall) []string {
	for i := len(deferred) - 1; i >= 0; i-- {
		call := deferred[i]
		if call.active == "" {
			lines = append(lines, call.call)
			continue
		}
		lines = append(lines, "if "+call.active+" {")
		lines = append(lines, "\t"+call.call)
		lines = append(lines, "\t"+call.active+" = false")
		lines = append(lines, "}")
	}
	return lines
}

func lowerReturnWithDynamicDefers(body string, toks []scan.Token, pos int, stmtEnd int, unitName string, plan dynamicDeferPlan, tempIndex *int) []string {
	start, end := trimTokenRange(toks, pos+1, stmtEnd)
	var lines []string
	if start >= end || toks[start].Line != toks[pos].Line {
		lines = appendDynamicDeferredCalls(lines, plan)
		return append(lines, "return")
	}
	ranges := topLevelExpressionRanges(toks, start, end)
	resultCount := returnResultCountForLower(toks)
	if len(ranges) == 1 && resultCount > 1 {
		exprStart, exprEnd := trimTokenRange(toks, ranges[0].start, ranges[0].end)
		line, names, ok := lowerSingleDirectMultiResultReturn(body, toks, exprStart, exprEnd, resultCount, unitName+"_return", tempIndex)
		if ok {
			lines = append(lines, line)
			lines = appendDynamicDeferredCalls(lines, plan)
			return append(lines, "return "+strings.Join(names, ", "))
		}
	}
	var values []string
	for i := 0; i < len(ranges); i++ {
		exprStart, exprEnd := trimTokenRange(toks, ranges[i].start, ranges[i].end)
		if exprStart >= exprEnd {
			continue
		}
		temp := nextExpressionTempName(body, unitName+"_return", tempIndex)
		(*tempIndex)++
		expr := strings.TrimSpace(body[int(toks[exprStart].Start):int(toks[exprEnd-1].End)])
		lines = append(lines, temp+" := "+expr)
		values = append(values, temp)
	}
	lines = appendDynamicDeferredCalls(lines, plan)
	if len(values) == 0 {
		return append(lines, "return")
	}
	return append(lines, "return "+strings.Join(values, ", "))
}

func appendDynamicDeferredCalls(lines []string, plan dynamicDeferPlan) []string {
	lines = append(lines, "for "+plan.index+" := len("+plan.order+") - 1; "+plan.index+" >= 0; "+plan.index+"-- {")
	lines = append(lines, "\t"+plan.kind+" := "+plan.order+"["+plan.index+"]")
	for siteIndex := 0; siteIndex < len(plan.sites); siteIndex++ {
		site := plan.sites[siteIndex]
		lines = append(lines, "\tif "+plan.kind+" == "+strconv.Itoa(site.id)+" {")
		siteLines := dynamicDeferSiteReplayLines(site)
		for lineIndex := 0; lineIndex < len(siteLines); lineIndex++ {
			lines = append(lines, "\t\t"+siteLines[lineIndex])
		}
		lines = append(lines, "\t}")
	}
	lines = append(lines, "}")
	return lines
}

func lowerPanicPropagationAfterCallLines(toks []scan.Token, names panicRecoverNames, plan dynamicDeferPlan, deferred []loweredDeferCall, abortUnrecovered bool) []string {
	if names.active == "" {
		return nil
	}
	var lines []string
	lines = append(lines, panicRecoverBridgeLines(names)...)
	lines = append(lines, "if "+names.active+" {")
	var deferLines []string
	if plan.enabled {
		deferLines = appendDynamicDeferredCalls(deferLines, plan)
	} else {
		deferLines = appendDeferredCalls(deferLines, deferred)
	}
	for i := 0; i < len(deferLines); i++ {
		lines = append(lines, "\t"+deferLines[i])
	}
	tailLines := panicPropagationReturnTailLines(toks, names, abortUnrecovered)
	for i := 0; i < len(tailLines); i++ {
		lines = append(lines, "\t"+tailLines[i])
	}
	lines = append(lines, "}")
	return lines
}

func panicRecoverBridgeLines(names panicRecoverNames) []string {
	var lines []string
	for i := 0; i < len(names.bridges); i++ {
		bridge := names.bridges[i]
		if bridge.active == "" || bridge.value == "" {
			continue
		}
		lines = append(lines, "if "+bridge.active+" {")
		lines = append(lines, "\t"+names.active+" = true")
		lines = append(lines, "\t"+names.value+" = "+bridge.value)
		lines = append(lines, "\t"+bridge.active+" = false")
		lines = append(lines, "}")
	}
	return lines
}

func panicPropagationReturnTailLines(toks []scan.Token, names panicRecoverNames, abortUnrecovered bool) []string {
	lines := panicReturnTailLines(toks, names, abortUnrecovered)
	if len(lines) > 0 {
		return lines
	}
	return []string{"return"}
}

func dynamicDeferSiteReplayLines(site dynamicDeferSite) []string {
	var lines []string
	indexName := ""
	if len(site.captures) > 0 {
		indexName = site.captures[0].name + "_index"
		lines = append(lines, indexName+" := len("+site.captures[0].stack+") - 1")
	}
	args := make([]string, 0, len(site.captures))
	for i := 0; i < len(site.captures); i++ {
		capture := site.captures[i]
		arg := capture.stack + "[" + indexName + "]"
		if capture.expanded {
			arg = arg + "..."
		}
		args = append(args, arg)
	}
	lines = append(lines, site.callee+"("+strings.Join(args, ", ")+")")
	for i := 0; i < len(site.captures); i++ {
		capture := site.captures[i]
		lines = append(lines, capture.stack+" = "+capture.stack+"[:"+indexName+"]")
	}
	return lines
}

func lowerDeferDirectCallOpen(toks []scan.Token, pos int, end int) int {
	i := pos + 1
	if i >= end || i >= len(toks) || toks[i].Kind != scan.Ident {
		return -1
	}
	i++
	for i+1 < end && toks[i].Text == "." && toks[i+1].Kind == scan.Ident {
		i += 2
	}
	if i < end && toks[i].Text == "(" {
		return i
	}
	return -1
}

func lowerSimpleStatementEnd(tokens []scan.Token, start int, limit int) int {
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
				if tokens[i].Text == "," || tokens[i-1].Text == "," || continuesFunctionLiteralCallForLower(tokens, i) {
					line = tokens[i].Line
				} else {
					return i
				}
			}
		}
		updateExpressionDepth(tokens[i].Text, &paren, &brack, &brace)
	}
	return limit
}

func continuesFunctionLiteralCallForLower(tokens []scan.Token, pos int) bool {
	return pos > 0 && pos < len(tokens) && tokens[pos].Text == "(" && tokens[pos-1].Text == "}" && tokens[pos].Line == tokens[pos-1].Line
}

func lowerStatementSourceEnd(toks []scan.Token, stmtEnd int, fallback int) int {
	if stmtEnd >= 0 && stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
		return int(toks[stmtEnd].End)
	}
	if stmtEnd > 0 && stmtEnd-1 < len(toks) {
		return int(toks[stmtEnd-1].End)
	}
	return fallback
}

func lowerPackageVarInitializerCalls(body string, functionResults localTypeTable, initUnitName string, tempIndex *int) (string, []string) {
	if !strings.Contains(body, "=") {
		return body, nil
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body, nil
	}
	if len(toks) < 3 || toks[0].Text != "var" {
		return body, nil
	}
	if toks[1].Text == "(" {
		close := findClose(toks, 1, "(", ")")
		if close < 0 {
			return body, nil
		}
		specs := localConstSpecRanges(toks, 2, close)
		var lines []string
		var initStatements []string
		changed := false
		for i := 0; i < len(specs); i++ {
			specLines, specInitStatements, specChanged, ok := lowerPackageVarSpec(body, toks, specs[i], functionResults, initUnitName, tempIndex)
			if !ok {
				return body, nil
			}
			changed = changed || specChanged
			for lineIndex := 0; lineIndex < len(specLines); lineIndex++ {
				lines = append(lines, specLines[lineIndex])
			}
			for stmtIndex := 0; stmtIndex < len(specInitStatements); stmtIndex++ {
				initStatements = append(initStatements, specInitStatements[stmtIndex])
			}
		}
		if !changed {
			return body, nil
		}
		return packageVarGroupBody(lines), initStatements
	}
	end := len(toks)
	if end > 0 && toks[end-1].Kind == scan.EOF {
		end--
	}
	specLines, initStatements, changed, ok := lowerPackageVarSpec(body, toks, expressionRange{start: 1, end: end}, functionResults, initUnitName, tempIndex)
	if !ok || !changed {
		return body, nil
	}
	if len(specLines) == 1 {
		return "var " + specLines[0] + "\n", initStatements
	}
	return packageVarGroupBody(specLines), initStatements
}

func lowerPackageVarSpec(body string, toks []scan.Token, spec expressionRange, functionResults localTypeTable, initUnitName string, tempIndex *int) ([]string, []string, bool, bool) {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return nil, nil, false, true
	}
	if !packageVarSpecHasCallableInitializer(toks, start, end) {
		return []string{strings.TrimSpace(tokenRangeText(body, toks, start, end))}, nil, false, true
	}
	eq := findTopLevelToken(toks, start, end, "=")
	if eq < 0 || eq+1 >= end {
		return []string{strings.TrimSpace(tokenRangeText(body, toks, start, end))}, nil, false, true
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, eq)
	if len(names) == 0 {
		return nil, nil, false, false
	}
	values := topLevelExpressionRanges(toks, eq+1, end)
	if len(values) == 0 {
		return nil, nil, false, false
	}
	if typeStart >= 0 {
		typ := strings.TrimSpace(tokenRangeText(body, toks, typeStart, eq))
		if typ == "" {
			return nil, nil, false, false
		}
		lines := make([]string, 0, len(names))
		for i := 0; i < len(names); i++ {
			lines = append(lines, names[i]+" "+typ)
		}
		if len(values) == len(names) {
			initStatements := make([]string, 0, len(names))
			for i := 0; i < len(names); i++ {
				initStatements = appendPackageInitializerAssignment(initStatements, body, toks, names[i], values[i], initUnitName, tempIndex)
			}
			return lines, initStatements, true, true
		}
		if len(values) == 1 {
			initStatement := joinNames(names) + " = " + strings.TrimSpace(tokenRangeText(body, toks, values[0].start, values[0].end))
			return lines, []string{initStatement}, true, true
		}
		return nil, nil, false, false
	}
	if len(values) != len(names) {
		return nil, nil, false, false
	}
	lines := make([]string, 0, len(names))
	initStatements := make([]string, 0, len(names))
	for i := 0; i < len(names); i++ {
		typ := packageInitializerValueTypeText(body, toks, values[i], functionResults)
		if typ == "" {
			return nil, nil, false, false
		}
		lines = append(lines, names[i]+" "+typ)
		initStatements = appendPackageInitializerAssignment(initStatements, body, toks, names[i], values[i], initUnitName, tempIndex)
	}
	return lines, initStatements, true, true
}

func appendPackageInitializerAssignment(statements []string, body string, toks []scan.Token, name string, value expressionRange, initUnitName string, tempIndex *int) []string {
	expr := strings.TrimSpace(tokenRangeText(body, toks, value.start, value.end))
	if packageInitializerAssignmentNeedsTemp(toks, value) {
		tempName := nextExpressionTempName(body, initUnitName, tempIndex)
		(*tempIndex)++
		statements = append(statements, tempName+" := "+expr)
		expr = tempName
	}
	statements = append(statements, name+" = "+expr)
	return statements
}

func packageInitializerAssignmentNeedsTemp(toks []scan.Token, value expressionRange) bool {
	start, end := trimTokenRange(toks, value.start, value.end)
	start, end = trimOuterParens(toks, start, end)
	if start >= end {
		return false
	}
	return tokensAreCompositeLiteral(toks, start, end) || tokensAreAddressOfCompositeLiteral(toks, start, end)
}

func packageInitializerValueTypeText(body string, toks []scan.Token, expr expressionRange, functionResults localTypeTable) string {
	start, end := trimTokenRange(toks, expr.start, expr.end)
	start, end = trimOuterParens(toks, start, end)
	if start >= end {
		return ""
	}
	if tokensAreAddressOfCompositeLiteral(toks, start, end) {
		text := packageCompositeLiteralTypeText(body, toks, start+1, end)
		if text == "" {
			return ""
		}
		return "*" + text
	}
	if tokensAreCompositeLiteral(toks, start, end) {
		return packageCompositeLiteralTypeText(body, toks, start, end)
	}
	if toks[start].Text == "[" && start+1 < end && toks[start+1].Text == "]" {
		open := findTopLevelToken(toks, start+2, end, "{")
		if open > start+2 {
			return strings.TrimSpace(tokenRangeText(body, toks, start, open))
		}
	}
	if toks[start].Kind == scan.Ident && start+1 < end && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			typ := localTypeTableLookup(functionResults, toks[start].Text)
			return localTypeInfoText(typ)
		}
	}
	if start+1 == end {
		tok := toks[start]
		if tok.Kind == scan.String {
			return "string"
		}
		if tok.Kind == scan.Number || tok.Kind == scan.Char {
			return "int"
		}
		if tok.Text == "true" || tok.Text == "false" {
			return "bool"
		}
	}
	return ""
}

func packageCompositeLiteralTypeText(body string, toks []scan.Token, start int, end int) string {
	if !tokensAreCompositeLiteral(toks, start, end) {
		return ""
	}
	open := findOpen(toks, end-1, "{", "}")
	if open < start {
		return ""
	}
	typeStart := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
	if typeStart != start {
		return ""
	}
	return strings.TrimSpace(tokenRangeText(body, toks, typeStart, open))
}

func localTypeInfoText(info localTypeInfo) string {
	if info.name == "" {
		return ""
	}
	var out []byte
	if info.pointer {
		out = append(out, '*')
	}
	if info.qualifier != "" {
		out = appendString(out, info.qualifier)
		out = append(out, '.')
	}
	out = appendString(out, info.name)
	return string(out)
}

func packageVarGroupBody(lines []string) string {
	var out []byte
	out = appendString(out, "var (\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		out = append(out, '\t')
		out = appendString(out, line)
		out = append(out, '\n')
	}
	out = append(out, ')')
	out = append(out, '\n')
	return string(out)
}

func tokenRangeText(body string, toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	return body[int(toks[start].Start):int(toks[end-1].End)]
}

func declNames(decl *parse.Decl) []string {
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

func unitPathForDecl(files []load.File, path string) string {
	for i := 0; i < len(files); i++ {
		file := files[i]
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
	if importPath == "j5.nz/rtg" {
		out = append(out, 'm')
		out = append(out, '_')
		out = appendString(out, name)
		return string(out)
	}
	if strings.HasPrefix(importPath, "j5.nz/rtg/") {
		out = append(out, 'm')
		importPath = importPath[len("j5.nz/rtg/"):]
	}
	for i := 0; i < len(importPath); i++ {
		c := importPath[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	out = append(out, '_')
	out = appendString(out, name)
	return string(out)
}

func appendString(out []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		out = append(out, s[i])
	}
	return out
}

func appendInterpretedStringLiteral(out []byte, raw string) []byte {
	out = append(out, '"')
	for i := 1; i+1 < len(raw); i++ {
		c := raw[i]
		if c == '\n' {
			out = append(out, '\\')
			out = append(out, 'n')
		} else if c == '\t' {
			out = append(out, '\\')
			out = append(out, 't')
		} else if c == '\r' {
			out = append(out, '\\')
			out = append(out, 'r')
		} else if c == '"' || c == '\\' {
			out = append(out, '\\')
			out = append(out, c)
		} else {
			out = append(out, c)
		}
	}
	out = append(out, '"')
	return out
}

func isOctalNumberText(text string) bool {
	if len(text) < 2 || text[0] != '0' {
		return false
	}
	next := text[1]
	if next == 'x' || next == 'X' || next == 'b' || next == 'B' || next == '.' {
		return false
	}
	if next == 'o' || next == 'O' {
		return true
	}
	return next >= '0' && next <= '9'
}

func parseOctalNumberText(text string) int {
	start := 1
	if len(text) > 2 && text[0] == '0' && (text[1] == 'o' || text[1] == 'O') {
		start = 2
	}
	value := 0
	for i := start; i < len(text); i++ {
		c := text[i]
		if c >= '0' && c <= '7' {
			value = value*8 + int(c-'0')
		}
	}
	return value
}

func appendStringRange(out []byte, s string, start int, end int) []byte {
	for i := start; i < end; i++ {
		out = append(out, s[i])
	}
	return out
}

func appendBytes(out []byte, values []byte) []byte {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func rewriteBufferCapacity(size int) int {
	capacity := size + size/4 + 256
	if capacity < 512 {
		return 512
	}
	return capacity
}

func appendStrings(out []string, values []string) []string {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func copyStrings(values []string) []string {
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func copyLoadFiles(values []load.File) []load.File {
	out := make([]load.File, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func snapshotLoadFiles(values []load.File) []load.File {
	out := make([]load.File, len(values))
	for i := 0; i < len(values); i++ {
		file := values[i]
		file.Path = copyLowerString(file.Path)
		file.UnitPath = copyLowerString(file.UnitPath)
		file.Source = copyLowerBytes(file.Source)
		file.Parsed = parse.File{}
		out[i] = file
	}
	return out
}

func copyLowerString(value string) string {
	var out []byte
	for i := 0; i < len(value); i++ {
		out = append(out, value[i])
	}
	return string(out)
}

func copyLowerBytes(values []byte) []byte {
	out := make([]byte, len(values))
	for i := 0; i < len(values); i++ {
		out[i] = values[i]
	}
	return out
}

func appendExpressionTemps(out []expressionTemp, values []expressionTemp) []expressionTemp {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}

func appendExpressionReplacements(out []expressionReplacement, values []expressionReplacement) []expressionReplacement {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = append(out, value)
	}
	return out
}

func cloneLocalTypeTable(values localTypeTable) localTypeTable {
	if len(values) == 0 {
		return nil
	}
	out := make(localTypeTable, len(values))
	copy(out, values)
	return out
}

func cloneStructFieldTypeTable(values structFieldTypeTable) structFieldTypeTable {
	if len(values) == 0 {
		return nil
	}
	out := make(structFieldTypeTable, len(values))
	copy(out, values)
	return out
}

func cloneStructOwnerTable(values structOwnerTable) structOwnerTable {
	if len(values) == 0 {
		return nil
	}
	out := make(structOwnerTable, len(values))
	copy(out, values)
	return out
}

func cloneArrayStructFieldLowerInfoTable(values arrayStructFieldLowerInfoTable) arrayStructFieldLowerInfoTable {
	if len(values) == 0 {
		return nil
	}
	out := make(arrayStructFieldLowerInfoTable, len(values))
	copy(out, values)
	return out
}

func appendStructFieldTypeTables(out structFieldTypeTable, values structFieldTypeTable) structFieldTypeTable {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = structFieldTypeTableSet(out, value.owner, value.field, value.info)
	}
	return out
}

func appendStructOwnerTables(out structOwnerTable, values structOwnerTable) structOwnerTable {
	for i := 0; i < len(values); i++ {
		out = structOwnerTableSet(out, values[i])
	}
	return out
}

func appendArrayStructFieldLowerInfoTables(out arrayStructFieldLowerInfoTable, values arrayStructFieldLowerInfoTable) arrayStructFieldLowerInfoTable {
	for i := 0; i < len(values); i++ {
		value := values[i]
		out = arrayStructFieldLowerInfoTableSet(out, value.owner, value.field, value.info)
	}
	return out
}

func localTypeDeclsForDecl(file *parse.File, decl *parse.Decl, importPath string, topNames symbolNameTable) []localTypeDeclInfo {
	if decl.Kind != "func" {
		return nil
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return nil
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return nil
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		return nil
	}
	functionUnitName := unitDeclSymbol(decl, file, topNames)
	if functionUnitName == "" {
		functionUnitName = SymbolName(importPath, unitDeclName(file, decl))
	}
	var out []localTypeDeclInfo
	index := 0
	for i := body + 1; i < close; i++ {
		if toks[i].Text != "type" || !startsLocalTypeDeclToken(toks, i, body) {
			continue
		}
		if i+1 < close && toks[i+1].Text == "(" {
			groupClose := findClose(toks, i+1, "(", ")")
			if groupClose < 0 || groupClose > close {
				continue
			}
			ranges := localConstSpecRanges(toks, i+2, groupClose)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				out = appendLocalTypeSpec(out, toks, body, i, int(toks[i].Start), int(toks[groupClose].End), ranges[rangeIndex].start, ranges[rangeIndex].end, functionUnitName, &index)
			}
			i = groupClose
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, decl.End)
		out = appendLocalTypeSpec(out, toks, body, i, int(toks[i].Start), localTypeSingleDeclSourceEnd(toks, i, decl.End), i+1, specEnd, functionUnitName, &index)
		if specEnd > i {
			i = specEnd - 1
		}
	}
	out = appendAnonymousStructTypeDecls(out, toks, body, close, functionUnitName, &index)
	return out
}

func appendLocalTypeSpec(out []localTypeDeclInfo, toks []scan.Token, body int, declTok int, declStart int, declEnd int, start int, end int, functionUnitName string, index *int) []localTypeDeclInfo {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return out
	}
	typeStart := start + 1
	if typeStart >= end {
		return out
	}
	name := toks[start].Text
	unitName := localTypeGeneratedName(functionUnitName, name, *index)
	(*index)++
	skipEmit := functionTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	if !skipEmit {
		skipEmit = functionContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = interfaceTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = interfaceContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = mapTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = mapContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = arrayTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingArrayTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = anyTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = complexTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	if !skipEmit {
		skipEmit = complexContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End))
	}
	out = append(out, localTypeDeclInfo{
		name:       name,
		unitName:   unitName,
		declStart:  declStart,
		declEnd:    declEnd,
		nameStart:  int(toks[start].Start),
		typeStart:  typeStart,
		typeEnd:    end,
		scopeStart: int(toks[start].Start),
		scopeEnd:   localScopeEnd(toks, body, declTok, maxSourcePosition()),
		skipEmit:   skipEmit,
	})
	return out
}

func startsLocalTypeDeclToken(toks []scan.Token, pos int, body int) bool {
	if pos <= body || pos+1 >= len(toks) {
		return false
	}
	if toks[pos+1].Kind != scan.Ident && toks[pos+1].Text != "(" {
		return false
	}
	prev := toks[pos-1]
	return prev.Text == "{" || prev.Text == "}" || prev.Text == ";" || prev.Line != toks[pos].Line
}

func localTypeSingleSpecEnd(toks []scan.Token, pos int, declEnd int) int {
	return localConstSingleEnd(toks, pos, declEnd)
}

func localTypeSingleDeclSourceEnd(toks []scan.Token, pos int, declEnd int) int {
	end := localTypeSingleSpecEnd(toks, pos, declEnd)
	if end < len(toks) && toks[end].Text == ";" && int(toks[end].Start) < declEnd {
		return int(toks[end].End)
	}
	if end > pos {
		return int(toks[end-1].End)
	}
	return int(toks[pos].End)
}

func localTypeGeneratedName(functionUnitName string, name string, index int) string {
	return functionUnitName + "_local_type_" + strconv.Itoa(index) + "_" + name
}

func localAnonymousStructGeneratedName(functionUnitName string, index int) string {
	return functionUnitName + "_anon_struct_" + strconv.Itoa(index)
}

func packageAnonymousStructGeneratedName(importPath string, index int) string {
	return SymbolName(importPath, "__rtg_anon_struct_"+strconv.Itoa(index))
}

func appendLocalTypeDeclInfos(out []localTypeDeclInfo, values []localTypeDeclInfo) []localTypeDeclInfo {
	if len(values) == 0 {
		return out
	}
	return append(out, values...)
}

func packageAnonymousStructDeclBody(info localTypeDeclInfo) string {
	if info.body != "" {
		return info.body
	}
	return "type " + info.unitName + " struct{}\n"
}

func packageAnonymousStructTypeDeclsForLoadFiles(files []load.File, importPath string) []localTypeDeclInfo {
	var out []localTypeDeclInfo
	index := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		out = appendPackageAnonymousStructTypeDecls(out, &parsed, importPath, &index)
		arena.Reset(mark)
	}
	return out
}

func appendPackageAnonymousStructTypeDecls(out []localTypeDeclInfo, file *parse.File, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			groupClose := findClose(toks, i+1, "(", ")")
			if groupClose < 0 {
				continue
			}
			ranges := localConstSpecRanges(toks, i+2, groupClose)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				out = appendPackageAnonymousStructTypeDeclsInTypeSpec(out, file, ranges[rangeIndex].start, ranges[rangeIndex].end, importPath, index)
			}
			i = groupClose
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := packageTypeSpecEndForLower(toks, i+1)
		out = appendPackageAnonymousStructTypeDeclsInTypeSpec(out, file, i+1, specEnd, importPath, index)
		i = specEnd - 1
	}
	out = appendPackageAnonymousStructTypeDeclsInFunctionSignatures(out, file, importPath, index)
	out = appendPackageAnonymousStructTypeDeclsInTopLevelVars(out, file, importPath, index)
	return out
}

func appendPackageAnonymousStructTypeDeclsInFunctionSignatures(out []localTypeDeclInfo, file *parse.File, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "func" {
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
		body := functionBodyOpenAfterParamsForLower(toks, paramsClose, decl.End)
		if body < 0 {
			body = tokenIndexBeforeForLower(toks, decl.End)
		}
		out = appendPackageAnonymousStructTypeDeclsInRange(out, file, paramsOpen+1, paramsClose, importPath, index)
		out = appendPackageAnonymousStructTypeDeclsInRange(out, file, paramsClose+1, body, importPath, index)
	}
	return out
}

func appendPackageAnonymousStructTypeDeclsInRange(out []localTypeDeclInfo, file *parse.File, start int, end int, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for i := start; i+1 < end; i++ {
		if toks[i].Text != "struct" || toks[i+1].Text != "{" {
			continue
		}
		close := findClose(toks, i+1, "{", "}")
		if close < 0 || close >= end {
			continue
		}
		out = appendPackageAnonymousStructTypeDecl(out, file, i, close+1, importPath, index)
		i = close
	}
	return out
}

func appendPackageAnonymousStructTypeDeclsInTopLevelVars(out []localTypeDeclInfo, file *parse.File, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close > start+1 {
				ranges := localConstSpecRanges(toks, start+2, close)
				for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
					out = appendPackageAnonymousStructTypeDeclsInValueSpec(out, file, ranges[rangeIndex].start, ranges[rangeIndex].end, importPath, index)
				}
			}
			continue
		}
		end := tokenIndexBeforeForLower(toks, decl.End) + 1
		if end > start+1 {
			out = appendPackageAnonymousStructTypeDeclsInValueSpec(out, file, start+1, end, importPath, index)
		}
	}
	return out
}

func appendPackageAnonymousStructTypeDeclsInValueSpec(out []localTypeDeclInfo, file *parse.File, start int, end int, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end {
		return out
	}
	eq := findTopLevelToken(toks, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	out = appendPackageAnonymousStructTypeDeclsInRange(out, file, start, lhsEnd, importPath, index)
	if eq >= 0 {
		out = appendPackageAnonymousStructTypeDeclsInRange(out, file, eq+1, end, importPath, index)
	}
	return out
}

func packageTypeSpecEndForLower(toks []scan.Token, namePos int) int {
	if namePos+1 >= len(toks) {
		return namePos + 1
	}
	typeStart := namePos + 1
	if toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 < len(toks) && toks[typeStart].Text == "struct" && toks[typeStart+1].Text == "{" {
		close := findClose(toks, typeStart+1, "{", "}")
		if close > typeStart+1 {
			return close + 1
		}
	}
	if typeStart+3 < len(toks) && toks[typeStart].Text == "[" && toks[typeStart+1].Text == "]" && toks[typeStart+2].Text == "struct" && toks[typeStart+3].Text == "{" {
		close := findClose(toks, typeStart+3, "{", "}")
		if close > typeStart+3 {
			return close + 1
		}
	}
	return typeSpecEnd(toks, namePos+1)
}

func appendPackageAnonymousStructTypeDeclsInTypeSpec(out []localTypeDeclInfo, file *parse.File, start int, end int, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	for start < end && toks[start].Text == ";" {
		start++
	}
	if start+2 >= end || toks[start].Kind != scan.Ident {
		return out
	}
	name := functionTypeSpecNameForLower(toks, start, end)
	if name != "" {
		if functionContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End)) {
			return out
		}
		if interfaceContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingInterfaceTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End)) {
			return out
		}
		if mapContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeAllowingMapTypeSpecsForLower(toks, name, int(toks[start].Start), int(toks[end-1].End)) {
			return out
		}
		if complexContainingTypeSpecRangeForLower(toks, start, end) && !identifierUsedOutsideSourceRangeForLower(toks, name, int(toks[start].Start), int(toks[end-1].End)) {
			return out
		}
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+3 <= end && toks[typeStart].Text == "[" && toks[typeStart+1].Text == "]" && toks[typeStart+2].Text == "struct" && toks[typeStart+3].Text == "{" {
		close := findClose(toks, typeStart+3, "{", "}")
		if close > typeStart+3 && close < end {
			out = appendPackageAnonymousStructTypeDecl(out, file, typeStart+2, close+1, importPath, index)
		}
		return out
	}
	if typeStart+1 >= end || toks[typeStart].Text != "struct" || toks[typeStart+1].Text != "{" {
		return out
	}
	close := findClose(toks, typeStart+1, "{", "}")
	if close < 0 || close >= end {
		return out
	}
	ranges := anonymousStructFieldSpecRangesForLower(toks, typeStart+1, close)
	for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
		fieldTypeStart, typeEnd, ok := anonymousStructFieldTypeRangeForLower(toks, ranges[rangeIndex].start, ranges[rangeIndex].end)
		if !ok {
			continue
		}
		out = appendPackageAnonymousStructTypeDecl(out, file, fieldTypeStart, typeEnd, importPath, index)
	}
	return out
}

func anonymousStructFieldSpecRangesForLower(toks []scan.Token, open int, close int) []expressionRange {
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
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return ranges
}

func anonymousStructFieldTypeRangeForLower(toks []scan.Token, start int, end int) (int, int, bool) {
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
		typeStart := anonymousStructFieldStructTokenForLower(toks, i, end)
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

func anonymousStructFieldStructTokenForLower(toks []scan.Token, pos int, end int) int {
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

func appendPackageAnonymousStructTypeDecl(out []localTypeDeclInfo, file *parse.File, typeStart int, typeEnd int, importPath string, index *int) []localTypeDeclInfo {
	toks := file.Tokens
	key := anonymousStructTypeKey(toks, typeStart, typeEnd)
	unitName := ""
	skipEmit := false
	for i := 0; i < len(out); i++ {
		if out[i].anonymous && out[i].typeKey == key && out[i].unitName != "" {
			unitName = out[i].unitName
			skipEmit = true
			break
		}
	}
	if unitName == "" {
		unitName = packageAnonymousStructGeneratedName(importPath, *index)
		(*index)++
	}
	body := ""
	if !skipEmit {
		body = "type " + unitName + " " + localTypeText(file.Source, toks, typeStart, typeEnd, out, true) + "\n"
	}
	return append(out, localTypeDeclInfo{
		name:       unitName,
		unitName:   unitName,
		path:       file.Path,
		declStart:  -1,
		declEnd:    -1,
		nameStart:  int(toks[typeStart].Start),
		typeStart:  typeStart,
		typeEnd:    typeEnd,
		scopeStart: 0,
		scopeEnd:   maxSourcePosition(),
		anonymous:  true,
		typeKey:    key,
		skipEmit:   skipEmit,
		body:       body,
	})
}

func appendAnonymousStructTypeDecls(out []localTypeDeclInfo, toks []scan.Token, body int, close int, functionUnitName string, index *int) []localTypeDeclInfo {
	for i := body + 1; i < close; i++ {
		if toks[i].Text == "[" && i+3 < close && toks[i+1].Text == "]" && toks[i+2].Text == "struct" && toks[i+3].Text == "{" {
			structClose := findClose(toks, i+3, "{", "}")
			if structClose < 0 {
				continue
			}
			literalClose := -1
			if structClose+1 < close && toks[structClose+1].Text == "{" {
				literalClose = findClose(toks, structClose+1, "{", "}")
			}
			if !anonymousStructTypeInLocalVarDeclForLower(toks, body, i+2, structClose) && (literalClose < 0 || !anonymousStructSliceLiteralInShortDeclForLower(toks, body, i, literalClose)) {
				continue
			}
			out = appendAnonymousStructTypeDecl(out, toks, i+2, structClose+1, int(toks[i].Start), localScopeEnd(toks, body, i, maxSourcePosition()), functionUnitName, index)
			if literalClose >= 0 {
				i = literalClose
			} else {
				i = structClose
			}
			continue
		}
		if toks[i].Text != "struct" || i+1 >= close || toks[i+1].Text != "{" {
			continue
		}
		structClose := findClose(toks, i+1, "{", "}")
		if structClose < 0 {
			continue
		}
		literalClose := -1
		if structClose+1 < close && toks[structClose+1].Text == "{" {
			literalClose = findClose(toks, structClose+1, "{", "}")
		}
		if !anonymousStructTypeInLocalVarDeclForLower(toks, body, i, structClose) && (literalClose < 0 || (!anonymousStructLiteralInShortDeclForLower(toks, body, i, literalClose) && !anonymousStructLiteralInVarDeclForLower(toks, body, i, literalClose))) {
			continue
		}
		out = appendAnonymousStructTypeDecl(out, toks, i, structClose+1, int(toks[i].Start), localScopeEnd(toks, body, i, maxSourcePosition()), functionUnitName, index)
		if literalClose >= 0 {
			i = literalClose
		} else {
			i = structClose
		}
	}
	return out
}

func appendAnonymousStructTypeDecl(out []localTypeDeclInfo, toks []scan.Token, typeStart int, typeEnd int, scopeStart int, scopeEnd int, functionUnitName string, index *int) []localTypeDeclInfo {
	key := anonymousStructTypeKey(toks, typeStart, typeEnd)
	unitName := ""
	skipEmit := false
	for i := 0; i < len(out); i++ {
		if out[i].anonymous && out[i].typeKey == key && out[i].unitName != "" {
			unitName = out[i].unitName
			skipEmit = true
			break
		}
	}
	if unitName == "" {
		unitName = localAnonymousStructGeneratedName(functionUnitName, *index)
		(*index)++
	}
	return append(out, localTypeDeclInfo{
		name:       unitName,
		unitName:   unitName,
		declStart:  -1,
		declEnd:    -1,
		nameStart:  int(toks[typeStart].Start),
		typeStart:  typeStart,
		typeEnd:    typeEnd,
		scopeStart: scopeStart,
		scopeEnd:   scopeEnd,
		anonymous:  true,
		typeKey:    key,
		skipEmit:   skipEmit,
	})
}

func anonymousStructTypeKey(toks []scan.Token, start int, end int) string {
	var out []byte
	for i := start; i < end; i++ {
		out = appendString(out, toks[i].Text)
	}
	return string(out)
}

func anonymousStructSliceLiteralInShortDeclForLower(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStartForLower(toks, body, start)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 || start <= assign {
		return false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		rhsStart, rhsEnd := trimTokenRange(toks, rhs[i].start, rhs[i].end)
		if rhsStart == start && rhsEnd == close+1 {
			return true
		}
	}
	return false
}

func anonymousStructLiteralInShortDeclForLower(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStartForLower(toks, body, start)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 || start <= assign {
		return false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		rhsStart, rhsEnd := trimTokenRange(toks, rhs[i].start, rhs[i].end)
		if rhsStart == start && rhsEnd == close+1 {
			return true
		}
	}
	return false
}

func anonymousStructLiteralInVarDeclForLower(toks []scan.Token, body int, start int, close int) bool {
	stmtStart := anonymousStructStatementStartForLower(toks, body, start)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || start <= assign || !anonymousStructLocalVarStatementForLower(toks, stmtStart) {
		return false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	for i := 0; i < len(rhs); i++ {
		rhsStart, rhsEnd := trimTokenRange(toks, rhs[i].start, rhs[i].end)
		if rhsStart == start && rhsEnd == close+1 {
			return true
		}
	}
	return false
}

func anonymousStructTypeInLocalVarDeclForLower(toks []scan.Token, body int, pos int, structClose int) bool {
	stmtStart := anonymousStructStatementStartForLower(toks, body, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	prefixEnd := stmtEnd
	if assign >= 0 {
		prefixEnd = assign
	}
	if stmtStart < prefixEnd && toks[stmtStart].Text == "var" {
		_, typeStart := localVarSpecNamesAndType(toks, stmtStart+1, prefixEnd)
		return typeStart == pos || (typeStart == pos-2 && pos >= 2 && toks[pos-2].Text == "[" && toks[pos-1].Text == "]" && structClose < prefixEnd)
	}
	if !insideVarGroupSpecForLower(toks, stmtStart) {
		return false
	}
	_, typeStart := localVarSpecNamesAndType(toks, stmtStart, prefixEnd)
	return typeStart == pos || (typeStart == pos-2 && pos >= 2 && toks[pos-2].Text == "[" && toks[pos-1].Text == "]" && structClose < prefixEnd)
}

func anonymousStructLocalVarStatementForLower(toks []scan.Token, stmtStart int) bool {
	if stmtStart >= 0 && stmtStart < len(toks) && toks[stmtStart].Text == "var" {
		return true
	}
	return insideVarGroupSpecForLower(toks, stmtStart)
}

func insideVarGroupSpecForLower(toks []scan.Token, pos int) bool {
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

func anonymousStructStatementStartForLower(toks []scan.Token, body int, pos int) int {
	if pos <= body || pos >= len(toks) {
		return simpleStatementStartForLower(toks, body, pos)
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

func localTypeDeclBody(file *parse.File, decls []localTypeDeclInfo, info localTypeDeclInfo) string {
	if info.body != "" {
		return info.body
	}
	body := "type " + info.unitName + " "
	body = body + localTypeText(file.Source, file.Tokens, info.typeStart, info.typeEnd, decls, true)
	body = body + "\n"
	return body
}

func localTypeOriginalDeclBody(file *parse.File, decls []localTypeDeclInfo, info localTypeDeclInfo) string {
	body := "type " + info.name + " "
	body = body + localTypeText(file.Source, file.Tokens, info.typeStart, info.typeEnd, decls, false)
	body = body + "\n"
	return body
}

func localTypeText(source []byte, toks []scan.Token, start int, end int, decls []localTypeDeclInfo, generated bool) string {
	if start >= end {
		return ""
	}
	textStart := int(toks[start].Start)
	textEnd := int(toks[end-1].End)
	out := make([]byte, 0, textEnd-textStart)
	cursor := textStart
	for i := start; i < end; i++ {
		tok := toks[i]
		if int(tok.Start) > cursor {
			out = appendBytes(out, source[cursor:int(tok.Start)])
		}
		if tok.Kind == scan.String && isStructTagToken(toks, i) {
			cursor = int(tok.End)
			continue
		}
		if generated && tok.Kind == scan.Ident {
			replacement := localTypeReplacementAt(decls, tok.Text, int(tok.Start))
			if replacement != "" {
				out = appendString(out, replacement)
				cursor = int(tok.End)
				continue
			}
		}
		out = appendBytes(out, source[int(tok.Start):int(tok.End)])
		cursor = int(tok.End)
	}
	if cursor < textEnd {
		out = appendBytes(out, source[cursor:textEnd])
	}
	return strings.TrimSpace(string(out))
}

func localTypeDeclAt(decls []localTypeDeclInfo, start int) bool {
	for i := 0; i < len(decls); i++ {
		if decls[i].declStart == start {
			return true
		}
	}
	return false
}

func localTypeDeclEndAt(decls []localTypeDeclInfo, start int) int {
	end := start
	for i := 0; i < len(decls); i++ {
		if decls[i].declStart == start && decls[i].declEnd > end {
			end = decls[i].declEnd
		}
	}
	return end
}

func anonymousStructSliceTypeReplacement(toks []scan.Token, path string, pos int, decls []localTypeDeclInfo) (string, int, int, bool) {
	if pos < 0 || pos+2 >= len(toks) || toks[pos].Text != "[" || toks[pos+1].Text != "]" || toks[pos+2].Text != "struct" {
		return "", 0, 0, false
	}
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if !info.anonymous || info.typeStart != pos+2 || info.typeEnd <= info.typeStart || info.unitName == "" {
			continue
		}
		if info.path != "" && info.path != path {
			continue
		}
		return "[]" + info.unitName, int(toks[info.typeEnd-1].End), info.typeEnd - 1, true
	}
	return "", 0, 0, false
}

func anonymousStructAliasEqualsTokenForLower(toks []scan.Token, pos int, limit int) bool {
	if pos <= 0 || pos+2 >= len(toks) || int(toks[pos].Start) >= limit || toks[pos].Text != "=" {
		return false
	}
	if toks[pos-1].Kind != scan.Ident || toks[pos+1].Text != "struct" || toks[pos+2].Text != "{" {
		return false
	}
	close := findClose(toks, pos+2, "{", "}")
	return close > pos+2 && int(toks[close].End) <= limit
}

func anonymousStructTypeReplacement(toks []scan.Token, path string, pos int, decls []localTypeDeclInfo) (string, int, int, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "struct" {
		return "", 0, 0, false
	}
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if !info.anonymous || info.typeStart != pos || info.typeEnd <= info.typeStart || info.unitName == "" {
			continue
		}
		if info.path != "" && info.path != path {
			continue
		}
		return info.unitName, int(toks[info.typeEnd-1].End), info.typeEnd - 1, true
	}
	return "", 0, 0, false
}

func anonymousStructGeneratedNameForRange(toks []scan.Token, path string, start int, end int, decls []localTypeDeclInfo) string {
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if !info.anonymous || info.unitName == "" || info.typeStart != start || info.typeEnd != end {
			continue
		}
		if info.path != "" && info.path != path {
			continue
		}
		return info.unitName
	}
	key := anonymousStructTypeKey(toks, start, end)
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if !info.anonymous || info.unitName == "" || info.typeKey != key {
			continue
		}
		if info.scopeStart == 0 && info.scopeEnd == maxSourcePosition() {
			return info.unitName
		}
	}
	return ""
}

func anonymousStructLiteralTypeReplacement(toks []scan.Token, pos int, decls []localTypeDeclInfo) (string, int, int, bool) {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "struct" || toks[pos+1].Text != "{" {
		return "", 0, 0, false
	}
	structClose := findClose(toks, pos+1, "{", "}")
	if structClose < 0 || structClose+1 >= len(toks) || toks[structClose+1].Text != "{" {
		return "", 0, 0, false
	}
	name := anonymousStructGeneratedNameForRange(toks, "", pos, structClose+1, decls)
	if name == "" {
		return "", 0, 0, false
	}
	return name, int(toks[structClose].End), structClose, true
}

func localTypeReplacementAt(decls []localTypeDeclInfo, name string, pos int) string {
	for i := len(decls) - 1; i >= 0; i-- {
		info := decls[i]
		if info.anonymous {
			continue
		}
		if info.name == name && pos >= info.scopeStart && pos < info.scopeEnd {
			return info.unitName
		}
	}
	return ""
}

func appendGeneratedLocalTypeUnderlyings(types localTypeTable, file *parse.File, decls []localTypeDeclInfo) localTypeTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			types = appendNamedTypeUnderlying(types, toks, 1, 2, len(toks)-1)
		}
	}
	return types
}

func appendGeneratedTypeUnderlyingsFromBodies(types localTypeTable, decls []localTypeDeclInfo) localTypeTable {
	for i := 0; i < len(decls); i++ {
		body := packageAnonymousStructDeclBody(decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			types = appendNamedTypeUnderlying(types, toks, 1, 2, len(toks)-1)
		}
	}
	return types
}

func appendOriginalLocalTypeUnderlyings(types localTypeTable, file *parse.File, decls []localTypeDeclInfo) localTypeTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeOriginalDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			types = appendNamedTypeUnderlying(types, toks, 1, 2, len(toks)-1)
		}
	}
	return types
}

func appendGeneratedLocalNamedSliceTypes(out []namedSliceInfo, file *parse.File, decls []localTypeDeclInfo) []namedSliceInfo {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			out = appendNamedSliceType(out, toks, 1, 2, len(toks)-1, nil)
		}
	}
	return out
}

func appendGeneratedLocalNamedArrayTypes(out []namedArrayInfo, file *parse.File, decls []localTypeDeclInfo) []namedArrayInfo {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			out = appendNamedArrayType(out, toks, 1, 2, len(toks)-1, nil)
		}
	}
	return out
}

func appendOriginalLocalNamedArrayTypes(out []namedArrayInfo, file *parse.File, decls []localTypeDeclInfo) []namedArrayInfo {
	for i := 0; i < len(decls); i++ {
		body := localTypeOriginalDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			out = appendNamedArrayType(out, toks, 1, 2, len(toks)-1, nil)
		}
	}
	return out
}

func appendGeneratedLocalNamedMapTypes(out []namedMapInfo, file *parse.File, decls []localTypeDeclInfo) []namedMapInfo {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			out = appendNamedMapType(out, toks, 1, 2, len(toks)-1, nil)
		}
	}
	return out
}

func appendGeneratedLocalNamedConversionTypes(out []string, file *parse.File, decls []localTypeDeclInfo) []string {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		toks, err := scan.Tokens([]byte(body))
		if err == nil && len(toks) > 2 {
			out = appendNamedConversionType(out, toks, 1, 2, len(toks)-1, nil)
		}
	}
	return out
}

func appendGeneratedLocalStructFieldTypes(fields structFieldTypeTable, file *parse.File, decls []localTypeDeclInfo, namedTypes localTypeTable) structFieldTypeTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		fields = appendLocalStructFieldTypesFromBody(fields, body, decls[i].unitName, namedTypes)
	}
	return fields
}

func appendGeneratedLocalStructOwners(owners structOwnerTable, file *parse.File, decls []localTypeDeclInfo) structOwnerTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		owners = appendLocalStructOwnerFromBody(owners, body, decls[i].unitName)
	}
	return owners
}

func appendGeneratedLocalArrayStructFieldLowerInfos(fields arrayStructFieldLowerInfoTable, file *parse.File, decls []localTypeDeclInfo, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeDeclBody(file, decls, decls[i])
		fields = appendArrayStructFieldLowerInfosFromBody(fields, body, decls[i].unitName, namedArrays)
	}
	return fields
}

func appendGeneratedStructFieldTypesFromBodies(fields structFieldTypeTable, decls []localTypeDeclInfo, namedTypes localTypeTable) structFieldTypeTable {
	for i := 0; i < len(decls); i++ {
		body := packageAnonymousStructDeclBody(decls[i])
		fields = appendLocalStructFieldTypesFromBody(fields, body, decls[i].unitName, namedTypes)
	}
	return fields
}

func appendGeneratedArrayStructFieldLowerInfosFromBodies(fields arrayStructFieldLowerInfoTable, decls []localTypeDeclInfo, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	for i := 0; i < len(decls); i++ {
		body := packageAnonymousStructDeclBody(decls[i])
		fields = appendArrayStructFieldLowerInfosFromBody(fields, body, decls[i].unitName, namedArrays)
	}
	return fields
}

func appendOriginalLocalStructFieldTypes(fields structFieldTypeTable, file *parse.File, decls []localTypeDeclInfo, namedTypes localTypeTable) structFieldTypeTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeOriginalDeclBody(file, decls, decls[i])
		fields = appendLocalStructFieldTypesFromBody(fields, body, decls[i].name, namedTypes)
	}
	return fields
}

func appendOriginalLocalStructOwners(owners structOwnerTable, file *parse.File, decls []localTypeDeclInfo) structOwnerTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeOriginalDeclBody(file, decls, decls[i])
		owners = appendLocalStructOwnerFromBody(owners, body, decls[i].name)
	}
	return owners
}

func appendOriginalLocalArrayStructFieldLowerInfos(fields arrayStructFieldLowerInfoTable, file *parse.File, decls []localTypeDeclInfo, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	for i := 0; i < len(decls); i++ {
		body := localTypeOriginalDeclBody(file, decls, decls[i])
		fields = appendArrayStructFieldLowerInfosFromBody(fields, body, decls[i].name, namedArrays)
	}
	return fields
}

func appendLocalStructFieldTypesFromBody(fields structFieldTypeTable, body string, owner string, namedTypes localTypeTable) structFieldTypeTable {
	toks, err := scan.Tokens([]byte(body))
	if err != nil || len(toks) < 5 {
		return fields
	}
	if toks[0].Text != "type" || toks[1].Kind != scan.Ident || toks[2].Text != "struct" || toks[3].Text != "{" {
		return fields
	}
	close := findClose(toks, 3, "{", "}")
	if close < 0 {
		return fields
	}
	return appendStructFieldTypeEntries(fields, owner, toks, 3, close, namedTypes)
}

func appendLocalStructOwnerFromBody(owners structOwnerTable, body string, owner string) structOwnerTable {
	toks, err := scan.Tokens([]byte(body))
	if err != nil || len(toks) < 5 {
		return owners
	}
	if toks[0].Text != "type" || toks[1].Kind != scan.Ident || toks[2].Text != "struct" || toks[3].Text != "{" {
		return owners
	}
	close := findClose(toks, 3, "{", "}")
	if close < 0 {
		return owners
	}
	return structOwnerTableSet(owners, owner)
}

func appendArrayStructFieldLowerInfosFromBody(fields arrayStructFieldLowerInfoTable, body string, owner string, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	toks, err := scan.Tokens([]byte(body))
	if err != nil || len(toks) < 5 {
		return fields
	}
	if toks[0].Text != "type" || toks[1].Kind != scan.Ident || toks[2].Text != "struct" || toks[3].Text != "{" {
		return fields
	}
	close := findClose(toks, 3, "{", "}")
	if close < 0 {
		return fields
	}
	return appendArrayStructFieldLowerInfoEntries(fields, owner, toks, 3, close, namedArrays)
}

func rewriteDecl(file *parse.File, decl *parse.Decl, topNames symbolNameTable, topFunctionNames symbolNameTable, importRefs importSymbolTable, methods methodTable, namedSlices []namedSliceInfo, namedArrays []namedArrayInfo, namedMaps []namedMapInfo, packageSizeofValueTypes sizeofLocalTypeTable, packageSizeofNamedTypes sizeofNamedTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, structOwners structOwnerTable, packageValueTypes localTypeTable, importedValueTypes localTypeTable, functionResults localTypeTable, arrayFunctionResults []arrayFunctionResultLowerInfo, arrayFunctionParams []arrayFunctionParamLowerInfo, arrayFieldTypes arrayStructFieldLowerInfoTable, interfaceParamErasures interfaceParamEraseTable, interfaceReturnErasures interfaceReturnEraseTable, nameOverride string, unitName string, wordSize int, refs *[]unit.Symbol, generatedDecls *[]unit.Decl, staticCallbacks *staticCallbackSetForLower) string {
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
	out := make([]byte, 0, rewriteBufferCapacity(end-start))
	mark := arena.Mark()
	source := file.Source
	sourceText := string(source)
	tokens := file.Tokens
	localNames := localNamesForDecl(file, decl, topNames)
	var importNames symbolNameTable
	for i := 0; i < len(importRefs); i++ {
		name := importRefs[i].localName
		if name == "." {
			symbols := importRefs[i].symbols
			for symbolIndex := 0; symbolIndex < len(symbols); symbolIndex++ {
				importNames = symbolNameTableSet(importNames, symbols[symbolIndex].Name, symbols[symbolIndex].Name)
			}
			continue
		}
		importNames = symbolNameTableSet(importNames, name, name)
	}
	importLocalNames := localNamesForDecl(file, decl, importNames)
	var localTypes localTypeTable
	localTypes = localTypesForDecl(file, decl)
	localTypes = collectImportedFunctionResultLocalTypesForDecl(file, decl, localTypes, importRefs, importLocalNames, functionResults)
	sizeofTypes := sizeofLocalTypesForDecl(file, decl, packageSizeofValueTypes)
	sizeofNamedTypes := sizeofNamedTypesForDecl(file, decl, packageSizeofNamedTypes)
	functionLiteralNamedArrays := appendOriginalLocalNamedArrayTypes(namedArrays, file, localTypeDecls)
	functionAliases := functionAliasesForDecl(file, decl, topFunctionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, functionLiteralNamedArrays, unitName, refs)
	functionArrayParams := appendFunctionAliasArrayParamInfos(arrayFunctionParams, functionAliases)
	if generatedDecls != nil {
		for aliasIndex := 0; aliasIndex < len(functionAliases); aliasIndex++ {
			alias := functionAliases[aliasIndex]
			if alias.hasDecl {
				*generatedDecls = append(*generatedDecls, alias.decl)
			}
		}
	}
	functionLiteralIndex := len(functionAliases)
	interfaceParamTempIndex := 0
	interfaceReturnTempIndex := 0
	cursor := start
	checkStructFieldNames := decl.Kind != "func"
	if decl.Kind == "func" && decl.Receiver {
		cursor = appendMethodDeclPrefix(file, decl, topNames, &out)
	}
	currentInterfaceParamErase, currentInterfaceParamEraseOK := interfaceParamEraseByName(interfaceParamErasures, decl.Name)
	currentParamsOpen := -1
	currentParamsClose := -1
	if currentInterfaceParamEraseOK && decl.Kind == "func" && !decl.Receiver {
		namePos := tokenIndexAt(tokens, int(decl.NameTok.Start))
		if namePos >= 0 && namePos+1 < len(tokens) && tokens[namePos+1].Text == "(" {
			currentParamsOpen = namePos + 1
			currentParamsClose = findClose(tokens, currentParamsOpen, "(", ")")
		}
	}
	_, currentInterfaceReturnEraseOK := interfaceReturnEraseByName(interfaceReturnErasures, decl.Name)
	currentResultStart := -1
	currentResultEnd := -1
	if currentInterfaceReturnEraseOK && decl.Kind == "func" && !decl.Receiver {
		namePos := tokenIndexAt(tokens, int(decl.NameTok.Start))
		if namePos >= 0 && namePos+1 < len(tokens) && tokens[namePos+1].Text == "(" {
			paramsOpen := namePos + 1
			paramsClose := findClose(tokens, paramsOpen, "(", ")")
			if paramsClose > paramsOpen {
				bodyOpen := functionBodyOpenForDeclAfterParamsForLower(tokens, paramsClose, decl.End)
				if bodyOpen > paramsClose {
					var ok bool
					currentResultStart, currentResultEnd, ok = interfaceReturnResultRangeForLower(tokens, paramsClose+1, bodyOpen)
					if !ok {
						currentInterfaceReturnEraseOK = false
					}
				}
			}
		}
	}
	namedResults := namedResultRewriteForDecl(file, decl, localTypeDecls)
	nameOverrideToken := -1
	if nameOverride != "" && decl.NameTok.Text != "" {
		nameOverrideToken = tokenIndexAt(tokens, int(decl.NameTok.Start))
	}
	prevText := ""
	fallthroughs := fallthroughRewrites(tokens, start, end)
	labeledControls := labeledControlRewrites(tokens, start, end)
	nilInterfaceComparisons := nilInterfaceVarComparisonLoweringsForDecl(tokens, decl)
	nilInterfaceComparisonReplacements := nilInterfaceComparisonExpressionReplacements(nilInterfaceComparisons)
	complexAliasComponents := complexAliasComponentLoweringsForDecl(sourceText, tokens, decl, localNames, topNames, topFunctionNames)
	complexAliasComponentReplacements := complexAliasComponentExpressionReplacements(complexAliasComponents)
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if int(tok.End) <= cursor {
			prevText = tok.Text
			continue
		}
		if int(tok.Start) >= end {
			break
		}
		if int(tok.Start) > cursor {
			part := source[cursor:int(tok.Start)]
			out = appendBytes(out, part)
		}
		if repl, ok := expressionReplacementAtStart(nilInterfaceComparisonReplacements, int(tok.Start)); ok {
			out = appendString(out, repl.text)
			cursor = repl.end
			prevText = repl.text
			continue
		}
		if repl, ok := expressionReplacementAtStart(complexAliasComponentReplacements, int(tok.Start)); ok {
			out = appendString(out, repl.text)
			cursor = repl.end
			prevText = repl.text
			continue
		}
		if currentInterfaceParamEraseOK && i == currentParamsOpen && currentParamsClose > currentParamsOpen {
			out = appendString(out, interfaceParamErasedParameterListForLower(source, tokens, currentParamsOpen, currentParamsClose, currentInterfaceParamErase))
			cursor = int(tokens[currentParamsClose].End)
			prevText = ")"
			i = currentParamsClose
			continue
		}
		if currentInterfaceReturnEraseOK && i == currentResultStart && currentResultEnd > currentResultStart {
			cursor = int(tokens[currentResultEnd-1].End)
			prevText = tokens[currentResultEnd-1].Text
			i = currentResultEnd - 1
			continue
		}
		if decl.Kind == "func" {
			if alias, ok := functionAliasDeclAt(functionAliases, i); ok {
				if alias.capture != "" {
					out = appendString(out, alias.receiver)
					out = appendString(out, " := ")
					out = appendString(out, alias.capture)
				}
				cursor = lowerStatementSourceEnd(tokens, alias.declEnd, int(tok.End))
				prevText = tok.Text
				i = alias.declEnd - 1
				continue
			}
			if stmtEnd, ok := discardedFunctionValueStatementForLowerAt(source, tokens, i, end, functionAliases, topFunctionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, functionLiteralNamedArrays, unitName); ok {
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if stmtEnd, ok := discardedEmptyCompositeLiteralStatementForLowerAt(tokens, i, end, topNames, localTypeDecls); ok {
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedMapSliceLiteralStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedArrayLiteralStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedMapLiteralStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedMapMakeStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedMapLiteralDeleteStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := complexAliasComponentDeclAt(complexAliasComponents, i); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := complexVarBlankDiscardStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if stmtEnd, ok := interfaceVarBlankDiscardStatementForLowerAt(tokens, i, end); ok {
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if stmtEnd, ok := nilInterfaceVarComparisonDeclEndAt(nilInterfaceComparisons, i); ok {
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if replacement, stmtEnd, ok := interfaceReturnDiscardStatementForLowerAt(source, tokens, i, end, interfaceReturnErasures); ok {
				out = appendString(out, replacement)
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if lines, stmtEnd, ok := discardedComplexStatementForLowerAt(sourceText, tokens, i, end, topNames); ok {
				if len(lines) > 0 {
					out = appendString(out, joinIndentedLines(lines, statementIndent(sourceText, int(tok.Start))))
				}
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if tok.Text == "return" {
				replacement, stmtEnd, ok := interfaceParamErasedReturnStatementForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
					prevText = tok.Text
					i = stmtEnd - 1
					continue
				}
			}
			if replacement, stmtEnd, ok := interfaceParamErasedAssignmentStatementForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex); ok {
				out = appendString(out, replacement)
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if replacement, stmtEnd, ok := interfaceParamErasedVarStatementForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex); ok {
				out = appendString(out, replacement)
				cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
				prevText = tok.Text
				i = stmtEnd - 1
				continue
			}
			if tok.Text == "defer" {
				replacement, stmtEnd, ok := interfaceParamErasedDeferStatementForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
					prevText = tok.Text
					i = stmtEnd - 1
					continue
				}
			}
			if tok.Text == "if" {
				replacement, replacementCursor, replacementToken, ok := interfaceParamErasedIfConditionForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tok.Text
					i = replacementToken
					continue
				}
			}
			if tok.Text == "for" {
				replacement, replacementCursor, replacementToken, ok := interfaceParamErasedForConditionForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tok.Text
					i = replacementToken
					continue
				}
				replacement, replacementCursor, replacementToken, ok = interfaceParamErasedClassicForConditionForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tok.Text
					i = replacementToken
					continue
				}
			}
			if tok.Text == "switch" {
				replacement, replacementCursor, replacementToken, ok := interfaceParamErasedSwitchTagForLowerAt(source, tokens, i, end, interfaceParamErasures, topNames, unitName+"_iparam", &interfaceParamTempIndex)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tok.Text
					i = replacementToken
					continue
				}
			}
			if currentInterfaceReturnEraseOK && tok.Text == "return" {
				stmtEnd := lowerSimpleStatementEnd(tokens, i, end)
				values := topLevelExpressionRanges(tokens, i+1, stmtEnd)
				if len(values) == 1 && !expressionContainsCall(tokens, values[0].start, values[0].end) {
					out = appendString(out, "return")
					cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
					prevText = tok.Text
					i = stmtEnd - 1
					continue
				}
				if len(values) == 1 && expressionContainsCall(tokens, values[0].start, values[0].end) {
					lines, ok := discardedInterfaceReturnSideEffectLinesForLower(sourceText, tokens, values[0].start, values[0].end, topNames, unitName+"_ireturn", &interfaceReturnTempIndex)
					if ok {
						indent := statementIndent(sourceText, int(tok.Start))
						lines = append(lines, "return")
						out = appendString(out, joinIndentedLines(lines, indent))
						cursor = lowerStatementSourceEnd(tokens, stmtEnd, int(tok.End))
						prevText = tok.Text
						i = stmtEnd - 1
						continue
					}
				}
			}
		}
		if i == nameOverrideToken {
			out = appendString(out, nameOverride)
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if namedResults.ok && i == namedResults.resultOpen {
			out = appendString(out, namedResults.signature)
			cursor = int(tokens[namedResults.resultClose].End)
			prevText = tok.Text
			continue
		}
		if namedResults.ok && i == namedResults.bodyOpen {
			out = appendBytes(out, source[int(tok.Start):int(tok.End)])
			out = appendString(out, namedResults.declarations)
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if namedResults.ok && tok.Text == "return" && bareReturnAt(tokens, i, end) {
			out = appendString(out, "return ")
			out = appendString(out, namedResults.returnExpr)
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if decl.Kind == "func" && tok.Text == "fallthrough" {
			label, ok := fallthroughLabelForToken(fallthroughs, i)
			if ok {
				out = appendString(out, "goto ")
				out = appendString(out, label)
				cursor = int(tok.End)
				prevText = label
				continue
			}
		}
		if decl.Kind == "func" && (tok.Text == "break" || tok.Text == "continue") {
			rewrite, ok := labeledControlForToken(labeledControls, i)
			if ok {
				out = appendString(out, "goto ")
				out = appendString(out, rewrite.label)
				cursor = int(tokens[rewrite.operand].End)
				prevText = rewrite.label
				i = rewrite.operand
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "const" {
			var ok bool
			out, cursor, ok = appendLoweredLocalConstDecl(out, source, tokens, i, end)
			if ok {
				prevText = tok.Text
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "var" {
			var ok bool
			out, cursor, ok = appendLoweredLocalInferredVarDecl(out, source, tokens, i, end)
			if ok {
				prevText = tok.Text
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "type" && localTypeDeclAt(localTypeDecls, int(tok.Start)) {
			cursor = localTypeDeclEndAt(localTypeDecls, int(tok.Start))
			prevText = tok.Text
			continue
		}
		if decl.Kind == "func" && tok.Text == "[" {
			replacement, replacementCursor, replacementToken, ok := anonymousStructSliceTypeReplacement(tokens, file.Path, i, localTypeDecls)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = tokens[replacementToken].Text
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "type" && anonymousStructAliasEqualsTokenForLower(tokens, i, end) {
			nextStart := int(tokens[i+1].Start)
			cursor = int(tok.End)
			for cursor < nextStart && (source[cursor] == ' ' || source[cursor] == '\t') {
				cursor++
			}
			if len(out) > 0 && out[len(out)-1] != ' ' && out[len(out)-1] != '\t' && out[len(out)-1] != '\n' {
				out = append(out, ' ')
			}
			prevText = tok.Text
			continue
		}
		if tok.Text == "struct" {
			replacement, replacementCursor, replacementToken, ok := anonymousStructTypeReplacement(tokens, file.Path, i, localTypeDecls)
			if !ok && (decl.Kind == "func" || decl.Kind == "var") {
				replacement, replacementCursor, replacementToken, ok = anonymousStructLiteralTypeReplacement(tokens, i, localTypeDecls)
			}
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = tokens[replacementToken].Text
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "println" {
			var ok bool
			out, cursor, ok = appendLoweredPrintlnCall(out, source, tokens, i, end)
			if ok {
				prevText = ")"
				continue
			}
		}
		if decl.Kind == "func" && tok.Kind == scan.Ident && (tok.Text == "real" || tok.Text == "imag") {
			replacement, replacementCursor, replacementToken, ok := reducibleComplexComponentReplacement(string(source), tokens, i, localNames, topNames, topFunctionNames)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = tokens[replacementToken].Text
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "type" {
			replacement, replacementCursor, replacementToken, ok := embeddedStructFieldDeclReplacement(tokens, i, topNames, importRefs, refs)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = tokens[replacementToken].Text
				i = replacementToken
				continue
			}
		}
		if tok.Kind == scan.String && isStructTagToken(tokens, i) {
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if tok.Kind == scan.String && strings.HasPrefix(tok.Text, "`") {
			out = appendInterpretedStringLiteral(out, tok.Text)
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if decl.Kind == "func" && tok.Text == "func" {
			literal, ok := functionLiteralDirectCallForLowerAt(tokens, i, end)
			if ok {
				captures := functionLiteralCapturesForLower(tokens, literal, localTypes, localTypeDecls)
				if functionLiteralDirectCallIsDeferredForLower(tokens, i) {
					captures = pointerFunctionLiteralCaptures(captures)
				}
				literalUnitName := unitName + "_func_literal_" + strconv.Itoa(functionLiteralIndex)
				functionLiteralIndex++
				if params := functionLiteralGeneratedArrayParamTypesForLower(source, tokens, literal, captures, functionLiteralNamedArrays); arrayTypeLowerInfosContainArray(params) {
					functionArrayParams = append(functionArrayParams, arrayFunctionParamLowerInfo{name: literalUnitName, params: params})
				}
				if generatedDecls != nil {
					*generatedDecls = append(*generatedDecls, functionLiteralDecl(source, tokens, literal, literalUnitName, captures, topNames, functionLiteralNamedArrays))
				}
				out = appendString(out, literalUnitName)
				out = append(out, '(')
				for captureIndex := 0; captureIndex < len(captures); captureIndex++ {
					if captureIndex > 0 {
						out = appendString(out, ", ")
					}
					out = appendFunctionLiteralCaptureArgument(out, captures[captureIndex])
				}
				if len(captures) > 0 && int(tokens[literal.callOpen].End) < int(tokens[literal.callClose].Start) {
					out = appendString(out, ", ")
				}
				cursor = int(tokens[literal.callOpen].End)
				prevText = "("
				i = literal.callOpen
				continue
			}
		}
		if tok.Text == "switch" && i+1 < len(tokens) && tokens[i+1].Text == "{" {
			out = appendString(out, "switch true")
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if tok.Kind == scan.Number && isOctalNumberText(tok.Text) {
			out = appendString(out, strconv.Itoa(parseOctalNumberText(tok.Text)))
			cursor = int(tok.End)
			prevText = tok.Text
			continue
		}
		if decl.Kind == "func" && tok.Text == ":" {
			labels := fallthroughLabelsForCaseColon(fallthroughs, i)
			if len(labels) > 0 {
				out = appendBytes(out, source[int(tok.Start):int(tok.End)])
				for labelIndex := 0; labelIndex < len(labels); labelIndex++ {
					out = append(out, '\n')
					out = appendString(out, fallthroughLabelIndent(source, int(tok.Start)))
					out = appendString(out, labels[labelIndex])
					out = append(out, ':')
				}
				cursor = int(tok.End)
				prevText = tok.Text
				continue
			}
		}
		if decl.Kind == "func" {
			replacement, replacementCursor, replacementToken, ok := compositeLiteralMethodCallReplacement(source, tokens, i, topNames, importRefs, methods, fieldTypes, refs)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = "("
				i = replacementToken
				continue
			}
			replacement, replacementCursor, replacementToken, ok = indexedReceiverMethodCallReplacement(source, tokens, i, localTypes, topNames, importRefs, methods, namedSlices, refs)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = "("
				i = replacementToken
				continue
			}
			replacement, replacementCursor, replacementToken, ok = methodExpressionCallReplacement(source, tokens, i, localTypes, importLocalNames, methods, refs)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = "("
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "." {
			replacement, replacementCursor, replacementToken, ok := promotedCompositeLiteralSelectorReplacement(tokens, i, fieldTypes)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = tokens[replacementToken].Text
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "func" && tok.Kind == scan.Ident && i+1 < len(tokens) && tokens[i+1].Text == "(" {
			if int(tok.Start) != int(decl.NameTok.Start) && prevText != "." && !isCompositeKey(tokens, i) && !isLocalNameAt(localNames, tok.Text, int(tok.Start)) {
				if info, ok := interfaceParamEraseByName(interfaceParamErasures, tok.Text); ok {
					replacement, replacementCursor, replacementToken, replaceOK := interfaceParamErasedCallReplacement(source, tokens, i, info, topNames, unitName+"_iparam", &interfaceParamTempIndex)
					if replaceOK {
						out = appendString(out, replacement)
						cursor = replacementCursor
						prevText = ")"
						i = replacementToken
						continue
					}
				}
			}
			replacement, replacementCursor, replacementToken, ok := staticCallbackCallReplacement(source, tokens, i, end, topNames, topFunctionNames, importRefs, importLocalNames, functionAliases, methods, localTypes, localTypeDecls, fieldTypes, functionLiteralNamedArrays, unitName, &functionLiteralIndex, staticCallbacks, refs, generatedDecls)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = ")"
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "func" && tok.Kind == scan.Ident && i+3 < len(tokens) && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident && tokens[i+3].Text == "(" {
			replacement, replacementCursor, replacementToken, ok := staticCallbackCallReplacement(source, tokens, i, end, topNames, topFunctionNames, importRefs, importLocalNames, functionAliases, methods, localTypes, localTypeDecls, fieldTypes, functionLiteralNamedArrays, unitName, &functionLiteralIndex, staticCallbacks, refs, generatedDecls)
			if ok {
				out = appendString(out, replacement)
				cursor = replacementCursor
				prevText = ")"
				i = replacementToken
				continue
			}
		}
		if decl.Kind == "func" && tok.Kind == scan.Ident && i+1 < len(tokens) && tokens[i+1].Text == "(" {
			if alias, ok := functionAliasAt(functionAliases, tok.Text, int(tok.Start)); ok {
				if alias.symbol.ImportPath != "" {
					appendUnitSymbolRef(refs, alias.symbol)
				}
				if alias.receiver != "" {
					open := i + 1
					close := findClose(tokens, open, "(", ")")
					out = appendString(out, alias.unitName)
					out = append(out, '(')
					out = appendString(out, alias.receiver)
					if close < 0 || int(tokens[open].End) < int(tokens[close].Start) {
						out = appendString(out, ", ")
					}
					cursor = int(tokens[open].End)
				} else if len(alias.captures) > 0 {
					open := i + 1
					close := findClose(tokens, open, "(", ")")
					out = appendString(out, alias.unitName)
					out = append(out, '(')
					for captureIndex := 0; captureIndex < len(alias.captures); captureIndex++ {
						if captureIndex > 0 {
							out = appendString(out, ", ")
						}
						out = appendFunctionLiteralCaptureArgument(out, alias.captures[captureIndex])
					}
					if close < 0 || int(tokens[open].End) < int(tokens[close].Start) {
						out = appendString(out, ", ")
					}
					cursor = int(tokens[open].End)
				} else {
					out = appendString(out, alias.unitName)
					cursor = int(tok.End)
				}
				prevText = tok.Text
				continue
			}
		}
		if tok.Kind == scan.Ident && i+2 < len(tokens) && tokens[i+1].Text == "." && tokens[i+2].Kind == scan.Ident {
			if decl.Kind == "func" {
				replacement, replacementCursor, replacementToken, ok := promotedSelectorReplacement(tokens, i, localTypes, fieldTypes)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tokens[replacementToken].Text
					i = replacementToken
					continue
				}
			}
			if i+3 < len(tokens) && tokens[i+3].Text == "(" {
				receiverType := localTypeTableLookup(localTypes, tok.Text)
				if receiverType.name != "" {
					memberTok := tokens[i+2]
					if receiverType.name == "error" && memberTok.Text == "Error" {
						close := findClose(tokens, i+3, "(", ")")
						if close == i+4 {
							out = appendString(out, "string(")
							out = appendString(out, tok.Text)
							out = append(out, ')')
							cursor = int(tokens[close].End)
							prevText = ")"
							i = close
							continue
						}
					}
					methodName := methodLookupName(receiverType, memberTok.Text)
					method := methodTableLookup(methods, methodName)
					var receiverArg string
					if method.unitName != "" {
						receiverArg = tok.Text
						if method.pointerReceiver && !receiverType.pointer {
							receiverArg = "&" + receiverArg
						} else if !method.pointerReceiver && receiverType.pointer {
							receiverArg = "*" + receiverArg
						}
					} else {
						path, promotedReceiver, promotedMethod, ok := promotedStructMethodPath(fieldTypes, localTypeInfoOwnerName(receiverType), memberTok.Text, methods)
						if !ok {
							method = methodInfo{}
						} else {
							method = promotedMethod
							receiverArg = promotedMethodReceiverArg(tok.Text, path, promotedReceiver, method)
						}
					}
					if method.unitName != "" {
						open := tokens[i+3]
						close := findClose(tokens, i+3, "(", ")")
						if method.importPath != "" {
							appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
						}
						replacement := method.unitName + "(" + receiverArg
						if close < 0 || int(open.End) < int(tokens[close].Start) {
							replacement = replacement + ", "
						}
						out = appendString(out, replacement)
						cursor = int(open.End)
						prevText = "("
						i += 3
						continue
					}
				}
			}
			symbolGroup, symbolsOK := importSymbolTableGroup(importRefs, tok.Text)
			if symbolsOK && !isLocalNameAt(importLocalNames, tok.Text, int(tok.Start)) {
				member := tokens[i+2]
				sym, symOK := importSymbolByName(symbolGroup, member.Text)
				if symOK {
					if sym.ImportPath == "unsafe" && sym.Name == "Sizeof" {
						replacement, replacementCursor, replacementToken, ok := unsafeSizeofSelectorReplacement(tokens, i, sizeofTypes, sizeofNamedTypes, fieldTypes, arrayFieldTypes, wordSize)
						if ok {
							out = appendString(out, replacement)
							cursor = replacementCursor
							prevText = ")"
							i = replacementToken
							continue
						}
					}
					if sym.ImportPath != "" {
						appendUnitSymbolRef(refs, sym)
					}
					replacement, replacementCursor, replacementToken, ok := importedPromotedSelectorReplacement(tokens, i, sym, importedValueTypes, fieldTypes)
					if ok {
						out = appendString(out, replacement)
						cursor = replacementCursor
						prevText = tokens[replacementToken].Text
						i = replacementToken
						continue
					}
					if sym.ImportPath == "fmt" && sym.Name == "Errorf" && i+3 < len(tokens) && tokens[i+3].Text == "(" {
						close := findClose(tokens, i+3, "(", ")")
						if close > i+4 {
							argStart, argEnd, argOK := callArgumentRange(tokens, i+4, close, 0)
							if !argOK {
								argStart = i + 4
								argEnd = firstCallArgumentEnd(tokens, i+4, close)
							}
							errorSym, errorSymOK := importSymbolByName(symbolGroup, "Error")
							errorUnitName := SymbolName("fmt", "Error")
							if errorSymOK {
								errorUnitName = errorSym.UnitName
								if errorSym.ImportPath != "" {
									appendUnitSymbolRef(refs, errorSym)
								}
							}
							out = appendString(out, errorUnitName)
							out = append(out, '(')
							firstArg := source[int(tokens[argStart].Start):int(tokens[argEnd-1].End)]
							out = appendBytes(out, firstArg)
							out = append(out, ')')
							cursor = int(tokens[close].End)
							prevText = ")"
							i = close
							continue
						}
					}
					if sym.ImportPath == "fmt" && isFmtPrintName(sym.Name) && i+3 < len(tokens) && tokens[i+3].Text == "(" {
						close := findClose(tokens, i+3, "(", ")")
						if close > i+4 {
							fdStart, fdEnd, fdOK := callArgumentRange(tokens, i+4, close, 0)
							argStart, argEnd, argOK := callArgumentRange(tokens, i+4, close, 1)
							if argOK {
								innerStart, innerEnd, bytesOK := bytesToStringCallArgumentRange(tokens, argStart, argEnd)
								if fdOK && bytesOK {
									out = appendString(out, "write(")
									if callArgumentIsStdout(tokens, fdStart, fdEnd) {
										out = append(out, '1')
									} else {
										fdArg := source[int(tokens[fdStart].Start):int(tokens[fdEnd-1].End)]
										out = appendBytes(out, fdArg)
									}
									out = appendString(out, ", ")
									innerArg := source[int(tokens[innerStart].Start):int(tokens[innerEnd-1].End)]
									out = appendBytes(out, innerArg)
									out = appendString(out, ", 0)")
									cursor = int(tokens[close].End)
									prevText = ")"
									i = close
									continue
								}
								out = appendString(out, "print(")
								arg := source[int(tokens[argStart].Start):int(tokens[argEnd-1].End)]
								out = appendBytes(out, arg)
								out = append(out, ')')
								cursor = int(tokens[close].End)
								prevText = ")"
								i = close
								continue
							}
						}
					}
					out = appendString(out, sym.UnitName)
					cursor = int(member.End)
					prevText = member.Text
					i += 2
					continue
				}
			}
		}
		if decl.Kind == "func" && tok.Kind == scan.Ident && prevText != "." && !isCompositeKey(tokens, i) {
			replacement := localTypeReplacementAt(localTypeDecls, tok.Text, int(tok.Start))
			if replacement != "" {
				out = appendString(out, replacement)
				cursor = int(tok.End)
				prevText = tok.Text
				continue
			}
		}
		if tok.Kind == scan.Ident && prevText != "." && !isCompositeKey(tokens, i) && !(checkStructFieldNames && isStructFieldName(tokens, i)) && !isLocalNameAt(localNames, tok.Text, int(tok.Start)) {
			unitName := symbolNameTableUnitName(topNames, tok.Text)
			if unitName != "" {
				out = appendString(out, unitName)
				cursor = int(tok.End)
				prevText = tok.Text
				continue
			}
			sym, dotImportOK := dotImportSymbol(importRefs, tok.Text)
			if dotImportOK && !isLocalNameAt(importLocalNames, tok.Text, int(tok.Start)) {
				if sym.ImportPath == "unsafe" && sym.Name == "Sizeof" {
					replacement, replacementCursor, replacementToken, ok := unsafeSizeofDotReplacement(tokens, i, sizeofTypes, sizeofNamedTypes, fieldTypes, arrayFieldTypes, wordSize)
					if ok {
						out = appendString(out, replacement)
						cursor = replacementCursor
						prevText = ")"
						i = replacementToken
						continue
					}
				}
				if sym.ImportPath != "" {
					appendUnitSymbolRef(refs, sym)
				}
				replacement, replacementCursor, replacementToken, ok := importedPromotedDotSelectorReplacement(tokens, i, sym, importedValueTypes, fieldTypes)
				if ok {
					out = appendString(out, replacement)
					cursor = replacementCursor
					prevText = tokens[replacementToken].Text
					i = replacementToken
					continue
				}
				out = appendString(out, sym.UnitName)
				cursor = int(tok.End)
				prevText = tok.Text
				continue
			}
		}
		if decl.Kind == "func" && tok.Text == "}" {
			continueLabels := labeledControlLabelsForClose(labeledControls, i, "continue")
			breakLabels := labeledControlLabelsForClose(labeledControls, i, "break")
			if len(continueLabels) > 0 || len(breakLabels) > 0 {
				closeIndent := statementIndent(sourceText, int(tok.Start))
				if len(continueLabels) > 0 {
					out = appendLabelLinePrefix(out, closeIndent)
				}
				for labelIndex := 0; labelIndex < len(continueLabels); labelIndex++ {
					if labelIndex > 0 {
						out = append(out, '\n')
						out = appendString(out, closeIndent)
					}
					out = appendString(out, continueLabels[labelIndex])
					out = append(out, ':')
				}
				if len(continueLabels) > 0 {
					out = append(out, '\n')
					out = appendString(out, closeIndent)
				}
				out = appendBytes(out, source[int(tok.Start):int(tok.End)])
				for labelIndex := 0; labelIndex < len(breakLabels); labelIndex++ {
					out = append(out, '\n')
					out = appendString(out, closeIndent)
					out = appendString(out, breakLabels[labelIndex])
					out = append(out, ':')
				}
				cursor = int(tok.End)
				prevText = tok.Text
				continue
			}
		}
		part := source[int(tok.Start):int(tok.End)]
		out = appendBytes(out, part)
		cursor = int(tok.End)
		prevText = tok.Text
	}
	if cursor < end {
		part := source[cursor:end]
		out = appendBytes(out, part)
	}
	body := string(out)
	body = lowerNamedMapTypeUses(body, namedMaps)
	body = lowerStaticMapAliases(body)
	body = lowerMapLiteralCommaOkAssignments(body)
	body = lowerMapLiteralIndexExpressions(body)
	body = lowerMapLiteralLenCalls(body)
	removeUnusedNamedMapRefs(refs, namedMaps, body)
	if decl.Kind == "func" {
		body = lowerNamedArrayTypeUses(body, namedArrays)
		structComparisonTypes := appendLocalTypeTable(cloneLocalTypeTable(packageValueTypes), localTypes)
		structComparisonTypes = appendLocalTypeEntries(structComparisonTypes, importedValueTypes)
		body = normalizeStructComparisonOperands(body, unitName, functionResults, structComparisonTypes, fieldTypes, structOwners, arrayFieldTypes, topNames)
		structComparisonTypes = collectBodyStructComparisonLocalTypes(body, structComparisonTypes, topNames)
		body = lowerStructComparisons(body, structComparisonTypes, fieldTypes, structOwners, arrayFieldTypes)
		arrayLocalTypes := appendLocalTypeTable(cloneLocalTypeTable(packageValueTypes), localTypes)
		arrayLocalTypes = appendLocalTypeEntries(arrayLocalTypes, importedValueTypes)
		arrayLocalTypes = collectBodyStructComparisonLocalTypes(body, arrayLocalTypes, topNames)
		body = normalizeArrayComparisonOperands(body, unitName, arrayFunctionResults, arrayLocalTypes, fieldTypes, arrayFieldTypes)
		body = lowerArrayComparisons(body, arrayFunctionResults, arrayLocalTypes, fieldTypes, arrayFieldTypes)
		body = lowerArrayFunctionParameterTypes(body)
		body = lowerArrayFunctionResultTypes(body)
		body = lowerGroupedFunctionParameterTypes(body)
		body = normalizeArrayArgumentCopies(body, functionArrayParams, arrayLocalTypes, fieldTypes, arrayFieldTypes)
		body = normalizeArrayValueCopies(body, arrayLocalTypes, fieldTypes, arrayFieldTypes)
		body = normalizeIndexedArraySelectorElementAssignments(body, unitName, arrayLocalTypes, fieldTypes, arrayFieldTypes)
		body = lowerLocalVarDeclarations(body)
		body = lowerArrayCompositeLiterals(body)
	}
	if decl.Kind == "var" {
		body = lowerNamedArrayTypeUses(body, namedArrays)
		body = lowerPackageArrayVarDeclarations(body)
		body = lowerArrayCompositeLiterals(body)
	}
	if decl.Kind == "type" {
		body = lowerNamedArrayTypeDeclarations(body, namedArrays)
		body = lowerArrayStructFieldTypes(body)
		body = splitGroupedAnonymousStructFieldLines(body, localTypeDecls)
	}
	body = lowerImplicitCompositeElements(body, namedSlices)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func lowerNamedMapTypeUses(body string, namedMaps []namedMapInfo) string {
	if len(namedMaps) == 0 || !strings.Contains(body, "{") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "{" {
			continue
		}
		info, ok := namedMapInfoByName(namedMaps, toks[i].Text)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
			start: int(toks[i].Start),
			end:   int(toks[i].End),
			text:  "map[" + info.keyType + "]" + info.valueType,
		}})
	}
	if len(replacements) == 0 {
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func lowerNamedArrayTypeDeclarations(body string, namedArrays []namedArrayInfo) string {
	if len(namedArrays) == 0 || !strings.Contains(body, "type") || !strings.Contains(body, "[") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			specs := localConstSpecRanges(toks, i+2, close)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				replacements = appendNamedArrayTypeDeclReplacement(replacements, toks, specs[specIndex].start, specs[specIndex].end, namedArrays)
			}
			i = close
			continue
		}
		specEnd := typeSpecEnd(toks, i+2)
		replacements = appendNamedArrayTypeDeclReplacement(replacements, toks, i+1, specEnd, namedArrays)
		i = specEnd - 1
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func appendNamedArrayTypeDeclReplacement(replacements []expressionReplacement, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) []expressionReplacement {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return replacements
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	info, ok := namedArrayInfoByName(namedArrays, toks[start].Text)
	if !ok {
		return replacements
	}
	if _, directOK := namedArrayTypeInfoForRange(toks, typeStart, end, nil); !directOK {
		return replacements
	}
	return append(replacements, expressionReplacement{
		start: int(toks[typeStart].Start),
		end:   int(toks[end-1].End),
		text:  "[]" + info.elem,
	})
}

func splitGroupedAnonymousStructFieldLines(body string, decls []localTypeDeclInfo) string {
	names := anonymousStructGeneratedTypeNames(decls)
	if len(names) == 0 || !strings.Contains(body, ",") {
		return body
	}
	lines := strings.Split(body, "\n")
	changed := false
	var out []string
	for lineIndex := 0; lineIndex < len(lines); lineIndex++ {
		line := lines[lineIndex]
		replacement, ok := splitGroupedAnonymousStructFieldLine(line, names)
		if ok {
			out = append(out, replacement...)
			changed = true
			continue
		}
		out = append(out, line)
	}
	if !changed {
		return body
	}
	return strings.Join(out, "\n")
}

func anonymousStructGeneratedTypeNames(decls []localTypeDeclInfo) []string {
	var out []string
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if !info.anonymous || info.unitName == "" {
			continue
		}
		if !containsString(out, info.unitName) {
			out = append(out, info.unitName)
		}
	}
	return out
}

func splitGroupedAnonymousStructFieldLine(line string, typeNames []string) ([]string, bool) {
	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	for i := 0; i < len(typeNames); i++ {
		typeName := typeNames[i]
		index := strings.Index(trimmed, typeName)
		if index <= 0 {
			continue
		}
		prefix := strings.TrimSpace(trimmed[:index])
		prefix, fieldType := splitGroupedAnonymousStructFieldType(prefix, typeName)
		if !strings.Contains(prefix, ",") {
			continue
		}
		suffix := trimmed[index+len(typeName):]
		names := strings.Split(prefix, ",")
		var lines []string
		for nameIndex := 0; nameIndex < len(names); nameIndex++ {
			name := strings.TrimSpace(names[nameIndex])
			if name == "" || !isIdentifierString(name) {
				return nil, false
			}
			lines = append(lines, indent+name+" "+fieldType+suffix)
		}
		return lines, true
	}
	return nil, false
}

func splitGroupedAnonymousStructFieldType(prefix string, typeName string) (string, string) {
	prefix = strings.TrimSpace(prefix)
	stars := 0
	for strings.HasSuffix(prefix, "*") {
		stars++
		prefix = strings.TrimSpace(prefix[:len(prefix)-1])
	}
	typePrefix := ""
	if strings.HasSuffix(prefix, "[]") {
		typePrefix = "[]"
		prefix = strings.TrimSpace(prefix[:len(prefix)-2])
	}
	for i := 0; i < stars; i++ {
		typePrefix += "*"
	}
	return prefix, typePrefix + typeName
}

func isIdentifierString(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (i > 0 && ch >= '0' && ch <= '9') {
			continue
		}
		return false
	}
	return true
}

func embeddedStructFieldDeclReplacement(toks []scan.Token, pos int, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, int, int, bool) {
	if !embeddedStructFieldStartForLower(toks, pos) {
		return "", 0, 0, false
	}
	end := structFieldSpecEndForLower(toks, pos)
	field, _, ok := embeddedStructFieldTypeInfo(toks, pos, end)
	if !ok {
		return "", 0, 0, false
	}
	typ, ok := loweredEmbeddedFieldTypeText(toks, pos, end, topNames, importRefs, refs)
	if !ok {
		return "", 0, 0, false
	}
	return field + " " + typ, int(toks[end-1].End), end - 1, true
}

func loweredEmbeddedFieldTypeText(toks []scan.Token, start int, end int, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	for end > start && toks[end-1].Kind == scan.String {
		end--
	}
	prefix := ""
	if start < end && toks[start].Text == "*" {
		prefix = "*"
		start++
	}
	typ, ok := loweredCompositeReceiverType(toks, start, end, topNames, importRefs, refs)
	if !ok {
		return "", false
	}
	return prefix + typ, true
}

func embeddedStructFieldStartForLower(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) {
		return false
	}
	if toks[pos].Text != "*" && toks[pos].Kind != scan.Ident {
		return false
	}
	if !nameInStructFieldListForLower(toks, pos) || !startsStructFieldLineForLower(toks, pos) {
		return false
	}
	end := structFieldSpecEndForLower(toks, pos)
	_, _, ok := embeddedStructFieldTypeInfo(toks, pos, end)
	return ok
}

func structFieldSpecEndForLower(toks []scan.Token, start int) int {
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

func nameInStructFieldListForLower(toks []scan.Token, pos int) bool {
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "}" {
			depth++
			continue
		}
		if toks[i].Text == "{" {
			if depth == 0 {
				return i > 0 && toks[i-1].Text == "struct"
			}
			depth--
		}
	}
	return false
}

func startsStructFieldLineForLower(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return false
	}
	prev := toks[pos-1]
	return prev.Text == "{" || prev.Text == ";" || prev.Line != toks[pos].Line
}

func promotedSelectorReplacement(toks []scan.Token, pos int, localTypes localTypeTable, fieldTypes structFieldTypeTable) (string, int, int, bool) {
	if pos <= 0 || pos+2 >= len(toks) {
		return "", 0, 0, false
	}
	if toks[pos-1].Text == "." || toks[pos+1].Text != "." || toks[pos].Kind != scan.Ident || toks[pos+2].Kind != scan.Ident {
		return "", 0, 0, false
	}
	if pos+3 < len(toks) && toks[pos+3].Text == "(" {
		return "", 0, 0, false
	}
	receiver := localTypeTableLookup(localTypes, toks[pos].Text)
	if receiver.name == "" {
		return "", 0, 0, false
	}
	owner := localTypeInfoOwnerName(receiver)
	path, ok := promotedStructFieldPath(fieldTypes, owner, toks[pos+2].Text)
	if !ok || len(path) == 0 {
		return "", 0, 0, false
	}
	out := toks[pos].Text
	for i := 0; i < len(path); i++ {
		out += "." + path[i]
	}
	out += "." + toks[pos+2].Text
	return out, int(toks[pos+2].End), pos + 2, true
}

func promotedCompositeLiteralSelectorReplacement(toks []scan.Token, dot int, fieldTypes structFieldTypeTable) (string, int, int, bool) {
	if dot <= 0 || dot+1 >= len(toks) || toks[dot].Text != "." || toks[dot+1].Kind != scan.Ident {
		return "", 0, 0, false
	}
	_, _, exprStart, exprEnd, ok := compositeLiteralSelectorBaseRanges(toks, dot)
	if !ok {
		return "", 0, 0, false
	}
	owner := compositeLiteralOwnerName(toks, exprStart, exprEnd)
	if owner == "" {
		return "", 0, 0, false
	}
	path, ok := promotedStructFieldPath(fieldTypes, owner, toks[dot+1].Text)
	if !ok || len(path) == 0 {
		return "", 0, 0, false
	}
	out := ""
	for i := 0; i < len(path); i++ {
		out += "." + path[i]
	}
	out += "." + toks[dot+1].Text
	return out, int(toks[dot+1].End), dot + 1, true
}

func compositeLiteralOwnerName(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "&" {
		start++
	}
	open := compositeLiteralOpenForTypeStart(toks, start)
	if open < 0 || open > end {
		return ""
	}
	return localTypeInfoOwnerName(typeInfoInRange(toks, start, open))
}

func importedPromotedSelectorReplacement(toks []scan.Token, pos int, sym unit.Symbol, importedValueTypes localTypeTable, fieldTypes structFieldTypeTable) (string, int, int, bool) {
	if pos+4 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "." || toks[pos+2].Kind != scan.Ident || toks[pos+3].Text != "." || toks[pos+4].Kind != scan.Ident {
		return "", 0, 0, false
	}
	valueName := toks[pos].Text + "." + toks[pos+2].Text
	return importedPromotedValueSelectorReplacement(toks, pos+4, sym.UnitName, valueName, importedValueTypes, fieldTypes)
}

func importedPromotedDotSelectorReplacement(toks []scan.Token, pos int, sym unit.Symbol, importedValueTypes localTypeTable, fieldTypes structFieldTypeTable) (string, int, int, bool) {
	if pos+2 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "." || toks[pos+2].Kind != scan.Ident {
		return "", 0, 0, false
	}
	return importedPromotedValueSelectorReplacement(toks, pos+2, sym.UnitName, toks[pos].Text, importedValueTypes, fieldTypes)
}

func importedPromotedValueSelectorReplacement(toks []scan.Token, fieldPos int, unitName string, valueName string, importedValueTypes localTypeTable, fieldTypes structFieldTypeTable) (string, int, int, bool) {
	if unitName == "" || fieldPos < 0 || fieldPos >= len(toks) || toks[fieldPos].Kind != scan.Ident {
		return "", 0, 0, false
	}
	valueType := localTypeTableLookup(importedValueTypes, valueName)
	owner := localTypeInfoOwnerName(valueType)
	path, ok := promotedStructFieldPath(fieldTypes, owner, toks[fieldPos].Text)
	if !ok || len(path) == 0 {
		return "", 0, 0, false
	}
	out := unitName
	for i := 0; i < len(path); i++ {
		out += "." + path[i]
	}
	out += "." + toks[fieldPos].Text
	return out, int(toks[fieldPos].End), fieldPos, true
}

func promotedStructFieldPath(table structFieldTypeTable, owner string, field string) ([]string, bool) {
	if owner == "" || field == "" {
		return nil, false
	}
	if structFieldTypeTableLookup(table, owner, field).name != "" {
		return nil, false
	}
	path, count := promotedStructFieldPathIn(table, owner, field, nil)
	return path, count == 1
}

func promotedStructFieldPathIn(table structFieldTypeTable, owner string, field string, seen []string) ([]string, int) {
	if containsString(seen, owner) {
		return nil, 0
	}
	seen = append(seen, owner)
	var found []string
	count := 0
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.owner != owner || !entry.info.embedded {
			continue
		}
		embeddedOwner := localTypeInfoOwnerName(entry.info)
		if embeddedOwner == "" {
			continue
		}
		if structFieldTypeTableLookup(table, embeddedOwner, field).name != "" {
			found = []string{entry.field}
			count++
			continue
		}
		nested, nestedCount := promotedStructFieldPathIn(table, embeddedOwner, field, seen)
		if nestedCount > 0 {
			path := make([]string, 0, len(nested)+1)
			path = append(path, entry.field)
			path = append(path, nested...)
			found = path
			count += nestedCount
		}
	}
	return found, count
}

func promotedStructMethodPath(table structFieldTypeTable, owner string, methodName string, methods methodTable) ([]string, localTypeInfo, methodInfo, bool) {
	path, receiver, method, count := promotedStructMethodPathIn(table, owner, methodName, methods, nil)
	return path, receiver, method, count == 1
}

func promotedStructMethodPathIn(table structFieldTypeTable, owner string, methodName string, methods methodTable, seen []string) ([]string, localTypeInfo, methodInfo, int) {
	if owner == "" || containsString(seen, owner) {
		return nil, localTypeInfo{}, methodInfo{}, 0
	}
	seen = append(seen, owner)
	var foundPath []string
	var foundReceiver localTypeInfo
	var foundMethod methodInfo
	count := 0
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.owner != owner || !entry.info.embedded {
			continue
		}
		embeddedOwner := localTypeInfoOwnerName(entry.info)
		if embeddedOwner == "" {
			continue
		}
		method := methodTableLookup(methods, methodLookupName(entry.info, methodName))
		if method.unitName != "" {
			foundPath = []string{entry.field}
			foundReceiver = entry.info
			foundMethod = method
			count++
			continue
		}
		nestedPath, nestedReceiver, nestedMethod, nestedCount := promotedStructMethodPathIn(table, embeddedOwner, methodName, methods, seen)
		if nestedCount > 0 {
			path := make([]string, 0, len(nestedPath)+1)
			path = append(path, entry.field)
			path = append(path, nestedPath...)
			foundPath = path
			foundReceiver = nestedReceiver
			foundMethod = nestedMethod
			count += nestedCount
		}
	}
	return foundPath, foundReceiver, foundMethod, count
}

func promotedMethodReceiverArg(receiver string, path []string, promotedReceiver localTypeInfo, method methodInfo) string {
	out := receiver
	for i := 0; i < len(path); i++ {
		out += "." + path[i]
	}
	if method.pointerReceiver && !promotedReceiver.pointer {
		return "&" + out
	}
	if !method.pointerReceiver && promotedReceiver.pointer {
		return "*" + out
	}
	return out
}

func localTypeInfoOwnerName(info localTypeInfo) string {
	if info.name == "" {
		return ""
	}
	if info.qualifier != "" {
		return info.qualifier + "." + info.name
	}
	return info.name
}

func lowerImplicitCompositeElements(body string, namedSlices []namedSliceInfo) string {
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var inserts []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "{" {
			continue
		}
		typ := implicitCompositeElementTypeWithNamedSlices(toks, i, namedSlices)
		if typ == "" {
			continue
		}
		pos := int(toks[i].Start)
		inserts = append(inserts, expressionReplacement{start: pos, end: pos, text: typ})
	}
	if len(inserts) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), inserts)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func lowerCopyStringSources(body string, localTypes localTypeTable, functionResults localTypeTable, namedTypes localTypeTable, fieldTypes structFieldTypeTable) string {
	if !strings.Contains(body, "copy") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	localTypes = collectBodyStringLocalTypes(toks, localTypes, functionResults)
	var replacements []expressionReplacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != "copy" || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && toks[i-1].Text == "." {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close <= i+1 {
			continue
		}
		args := topLevelExpressionRanges(toks, i+2, close)
		if len(args) != 2 || !copySourceIsStringExpression(toks, args[1], localTypes, functionResults, namedTypes, fieldTypes) {
			i = close
			continue
		}
		start := int(toks[args[1].start].Start)
		end := int(toks[args[1].end-1].End)
		src := body[start:end]
		replacements = append(replacements, expressionReplacement{start: start, end: end, text: "[]byte(" + src + ")"})
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func lowerErrorMethodCalls(body string, localTypes localTypeTable, functionResults localTypeTable) string {
	if !strings.Contains(body, ".Error()") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	localTypes = collectBodyStringLocalTypes(toks, localTypes, functionResults)
	var replacements []expressionReplacement
	for i := 0; i+4 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "." || toks[i+2].Text != "Error" || toks[i+3].Text != "(" || toks[i+4].Text != ")" {
			continue
		}
		receiver := localTypeTableLookup(localTypes, toks[i].Text)
		if receiver.name != "error" {
			continue
		}
		start := int(toks[i].Start)
		end := int(toks[i+4].End)
		replacements = append(replacements, expressionReplacement{start: start, end: end, text: "string(" + toks[i].Text + ")"})
		i += 4
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func collectBodyStringLocalTypes(toks []scan.Token, localTypes localTypeTable, functionResults localTypeTable) localTypeTable {
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == ":=" {
			if i-1 < 0 || i+1 >= len(toks) || toks[i-1].Kind != scan.Ident {
				continue
			}
			initEnd := shortDeclInitializerEnd(toks, i+1)
			typ := localInitializerTypeWithFunctions(toks, i+1, initEnd, localTypes, functionResults)
			if typ.name != "" {
				localTypes = localTypeTableSet(localTypes, toks[i-1].Text, typ)
			}
			continue
		}
		if toks[i].Text == "var" {
			if i+3 >= len(toks) || toks[i+1].Kind != scan.Ident || toks[i+2].Text != "=" {
				continue
			}
			initEnd := varInitializerEnd(toks, i+3, len(toks))
			typ := localInitializerTypeWithFunctions(toks, i+3, initEnd, localTypes, functionResults)
			if typ.name != "" {
				localTypes = localTypeTableSet(localTypes, toks[i+1].Text, typ)
			}
		}
	}
	return localTypes
}

func copySourceIsStringExpression(toks []scan.Token, expr expressionRange, localTypes localTypeTable, functionResults localTypeTable, namedTypes localTypeTable, fieldTypes structFieldTypeTable) bool {
	start, end := trimTokenRange(toks, expr.start, expr.end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return copySourceIsStringExpression(toks, expressionRange{start: start + 1, end: close}, localTypes, functionResults, namedTypes, fieldTypes)
		}
	}
	if start+1 == end {
		tok := toks[start]
		if tok.Kind == scan.String {
			return true
		}
		if tok.Kind == scan.Ident {
			typ := localTypeTableLookup(localTypes, tok.Text)
			return localTypeIsString(typ, namedTypes)
		}
		return false
	}
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		receiver := localTypeTableLookup(localTypes, toks[start].Text)
		field := structFieldTypeTableLookup(fieldTypes, receiver.name, toks[start+2].Text)
		return localTypeIsString(field, namedTypes)
	}
	if toks[start].Kind == scan.Ident && toks[start].Text == "string" && start+1 < end && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		return close == end-1
	}
	if toks[start].Kind == scan.Ident && start+1 < end && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			typ := localTypeTableLookup(functionResults, toks[start].Text)
			return localTypeIsString(typ, namedTypes)
		}
	}
	return false
}

func localTypeIsString(typ localTypeInfo, namedTypes localTypeTable) bool {
	for i := 0; i < 8; i++ {
		if typ.pointer || typ.qualifier != "" {
			return false
		}
		if typ.name == "string" {
			return true
		}
		underlying := localTypeTableLookup(namedTypes, typ.name)
		if underlying.name == "" || underlying.name == typ.name {
			return false
		}
		typ = underlying
	}
	return false
}

func functionResultTypes(files []parse.File, topNames symbolNameTable) localTypeTable {
	var results localTypeTable
	results = appendFunctionResultTypesForFiles(results, files, "", topNames)
	return results
}

func appendLocalTypeTable(out localTypeTable, values localTypeTable) localTypeTable {
	for i := 0; i < len(values); i++ {
		out = localTypeTableSet(out, values[i].name, values[i].info)
	}
	return out
}

func appendLocalTypeEntries(out localTypeTable, values localTypeTable) localTypeTable {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func appendFunctionResultTypesForFiles(results localTypeTable, files []parse.File, importPath string, topNames symbolNameTable) localTypeTable {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		results = appendFunctionResultTypesForFile(&parsed, importPath, topNames, results)
	}
	return results
}

func appendFunctionResultTypesForFile(parsed *parse.File, importPath string, topNames symbolNameTable, results localTypeTable) localTypeTable {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		typ := singleFunctionResultType(parsed, &decl)
		if typ.name == "" {
			continue
		}
		name := decl.Name
		if decl.Receiver {
			name = methodDeclName(parsed, &decl)
		}
		if importPath == "" {
			results = localTypeTableSet(results, name, typ)
			unitName := symbolNameTableUnitName(topNames, name)
			if unitName != "" {
				results = localTypeTableSet(results, unitName, typ)
			}
		} else {
			results = localTypeTableSet(results, SymbolName(importPath, name), typ)
		}
	}
	return results
}

func functionResultTypesForLoadFiles(files []load.File, topNames symbolNameTable) localTypeTable {
	out := make(localTypeTable, 0, functionResultTypeCapacityForLoadFiles(files, "", topNames))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendFunctionResultTypesForFile(&parsed, "", topNames, nil)
		tmp = persistLocalTypeTable(tmp)
		out = appendLocalTypeEntries(out, tmp)
		arena.Reset(mark)
	}
	return out
}

func functionResultTypeCapacityForLoadFiles(files []load.File, importPath string, topNames symbolNameTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(appendFunctionResultTypesForFile(&parsed, importPath, topNames, nil))
		}
		arena.Reset(mark)
	}
	return count
}

func singleFunctionResultType(file *parse.File, decl *parse.Decl) localTypeInfo {
	toks := file.Tokens
	name := tokenIndexAt(toks, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
		return localTypeInfo{}
	}
	paramsClose := findClose(toks, name+1, "(", ")")
	if paramsClose < 0 {
		return localTypeInfo{}
	}
	body := functionBodyOpenAfterParamsForLower(toks, paramsClose, decl.End)
	if body < 0 || paramsClose+1 >= body {
		return localTypeInfo{}
	}
	resultStart := paramsClose + 1
	if toks[resultStart].Text == "(" {
		resultClose := findClose(toks, resultStart, "(", ")")
		if resultClose < 0 || resultClose > body {
			return localTypeInfo{}
		}
		results := topLevelExpressionRanges(toks, resultStart+1, resultClose)
		if len(results) != 1 {
			return localTypeInfo{}
		}
		typ := typeInfoInRange(toks, results[0].start, results[0].end)
		typ.pointer = typeRangeIsPointer(toks, results[0].start, results[0].end)
		return typ
	}
	typ := typeInfoInRange(toks, resultStart, body)
	typ.pointer = typeRangeIsPointer(toks, resultStart, body)
	return typ
}

type arrayFunctionResultLowerInfo struct {
	name string
	info arrayTypeLowerInfo
}

type arrayFunctionParamLowerInfo struct {
	name   string
	params []arrayTypeLowerInfo
}

func arrayFunctionResultTypesForLoadFiles(files []load.File, topNames symbolNameTable, namedArrays []namedArrayInfo) []arrayFunctionResultLowerInfo {
	var out []arrayFunctionResultLowerInfo
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendArrayFunctionResultTypesForFile(nil, &parsed, topNames, namedArrays)
		for i := 0; i < len(tmp); i++ {
			tmp[i].name = arena.PersistString(tmp[i].name)
			tmp[i].info.elem = arena.PersistString(tmp[i].info.elem)
			out = append(out, tmp[i])
		}
		arena.Reset(mark)
	}
	return out
}

func arrayFunctionParamTypesForLoadFiles(files []load.File, topNames symbolNameTable, namedArrays []namedArrayInfo) []arrayFunctionParamLowerInfo {
	var out []arrayFunctionParamLowerInfo
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendArrayFunctionParamTypesForFile(nil, &parsed, topNames, namedArrays)
		for i := 0; i < len(tmp); i++ {
			tmp[i].name = arena.PersistString(tmp[i].name)
			for paramIndex := 0; paramIndex < len(tmp[i].params); paramIndex++ {
				tmp[i].params[paramIndex].elem = arena.PersistString(tmp[i].params[paramIndex].elem)
			}
			out = append(out, tmp[i])
		}
		arena.Reset(mark)
	}
	return out
}

func appendDependencyArrayFunctionResultTypes(out []arrayFunctionResultLowerInfo, packages []load.Package) []arrayFunctionResultLowerInfo {
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		pkg := packages[pkgIndex]
		topNames := packageSymbolNamesForLoadFiles(pkg.ImportPath, pkg.Files, packageSymbolCapacity(pkg.Files, len(pkg.Imports)))
		namedArrays := namedArrayTypesForLoadFiles(pkg.Files, topNames)
		for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
			mark := arena.Mark()
			parsed, err := parsedLoadFile(pkg.Files[fileIndex])
			if err != nil {
				arena.Reset(mark)
				continue
			}
			tmp := appendArrayFunctionResultTypesForImportPath(nil, &parsed, pkg.ImportPath, namedArrays)
			for i := 0; i < len(tmp); i++ {
				tmp[i].name = arena.PersistString(tmp[i].name)
				tmp[i].info.elem = arena.PersistString(tmp[i].info.elem)
				out = append(out, tmp[i])
			}
			arena.Reset(mark)
		}
	}
	return out
}

func appendDependencyArrayFunctionParamTypes(out []arrayFunctionParamLowerInfo, packages []load.Package) []arrayFunctionParamLowerInfo {
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		pkg := packages[pkgIndex]
		topNames := packageSymbolNamesForLoadFiles(pkg.ImportPath, pkg.Files, packageSymbolCapacity(pkg.Files, len(pkg.Imports)))
		namedArrays := namedArrayTypesForLoadFiles(pkg.Files, topNames)
		for fileIndex := 0; fileIndex < len(pkg.Files); fileIndex++ {
			mark := arena.Mark()
			parsed, err := parsedLoadFile(pkg.Files[fileIndex])
			if err != nil {
				arena.Reset(mark)
				continue
			}
			tmp := appendArrayFunctionParamTypesForImportPath(nil, &parsed, pkg.ImportPath, namedArrays)
			for i := 0; i < len(tmp); i++ {
				tmp[i].name = arena.PersistString(tmp[i].name)
				for paramIndex := 0; paramIndex < len(tmp[i].params); paramIndex++ {
					tmp[i].params[paramIndex].elem = arena.PersistString(tmp[i].params[paramIndex].elem)
				}
				out = append(out, tmp[i])
			}
			arena.Reset(mark)
		}
	}
	return out
}

func appendArrayFunctionResultTypesForFile(out []arrayFunctionResultLowerInfo, parsed *parse.File, topNames symbolNameTable, namedArrays []namedArrayInfo) []arrayFunctionResultLowerInfo {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		info, ok := singleFunctionResultArrayType(parsed, &decl, namedArrays)
		if !ok {
			continue
		}
		name := decl.Name
		if decl.Receiver {
			name = methodDeclName(parsed, &decl)
		}
		out = append(out, arrayFunctionResultLowerInfo{name: name, info: info})
		unitName := symbolNameTableUnitName(topNames, name)
		if unitName != "" && unitName != name {
			out = append(out, arrayFunctionResultLowerInfo{name: unitName, info: info})
		}
	}
	return out
}

func appendArrayFunctionParamTypesForFile(out []arrayFunctionParamLowerInfo, parsed *parse.File, topNames symbolNameTable, namedArrays []namedArrayInfo) []arrayFunctionParamLowerInfo {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		params, ok := singleFunctionParamArrayTypes(parsed, &decl, namedArrays)
		if !ok {
			continue
		}
		name := decl.Name
		if decl.Receiver {
			name = methodDeclName(parsed, &decl)
		}
		out = append(out, arrayFunctionParamLowerInfo{name: name, params: params})
		unitName := symbolNameTableUnitName(topNames, name)
		if unitName != "" && unitName != name {
			out = append(out, arrayFunctionParamLowerInfo{name: unitName, params: params})
		}
	}
	return out
}

func appendArrayFunctionResultTypesForImportPath(out []arrayFunctionResultLowerInfo, parsed *parse.File, importPath string, namedArrays []namedArrayInfo) []arrayFunctionResultLowerInfo {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		info, ok := singleFunctionResultArrayType(parsed, &decl, namedArrays)
		if !ok {
			continue
		}
		name := decl.Name
		if decl.Receiver {
			name = methodDeclName(parsed, &decl)
		}
		out = append(out, arrayFunctionResultLowerInfo{name: SymbolName(importPath, name), info: info})
	}
	return out
}

func appendArrayFunctionParamTypesForImportPath(out []arrayFunctionParamLowerInfo, parsed *parse.File, importPath string, namedArrays []namedArrayInfo) []arrayFunctionParamLowerInfo {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		if decl.Kind != "func" {
			continue
		}
		params, ok := singleFunctionParamArrayTypes(parsed, &decl, namedArrays)
		if !ok {
			continue
		}
		name := decl.Name
		if decl.Receiver {
			name = methodDeclName(parsed, &decl)
		}
		out = append(out, arrayFunctionParamLowerInfo{name: SymbolName(importPath, name), params: params})
	}
	return out
}

func singleFunctionResultArrayType(file *parse.File, decl *parse.Decl, namedArrays []namedArrayInfo) (arrayTypeLowerInfo, bool) {
	toks := file.Tokens
	name := tokenIndexAt(toks, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
		return arrayTypeLowerInfo{}, false
	}
	paramsClose := findClose(toks, name+1, "(", ")")
	if paramsClose < 0 {
		return arrayTypeLowerInfo{}, false
	}
	body := functionBodyOpenAfterParamsForLower(toks, paramsClose, decl.End)
	if body < 0 || paramsClose+1 >= body {
		return arrayTypeLowerInfo{}, false
	}
	resultStart := paramsClose + 1
	if toks[resultStart].Text == "(" {
		resultClose := findClose(toks, resultStart, "(", ")")
		if resultClose < 0 || resultClose > body {
			return arrayTypeLowerInfo{}, false
		}
		results := topLevelExpressionRanges(toks, resultStart+1, resultClose)
		if len(results) != 1 {
			return arrayTypeLowerInfo{}, false
		}
		return functionResultArrayTypeInRange(file, toks, results[0].start, results[0].end, namedArrays)
	}
	return functionResultArrayTypeInRange(file, toks, resultStart, body, namedArrays)
}

func singleFunctionParamArrayTypes(file *parse.File, decl *parse.Decl, namedArrays []namedArrayInfo) ([]arrayTypeLowerInfo, bool) {
	toks := file.Tokens
	name := tokenIndexAt(toks, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
		return nil, false
	}
	paramsClose := findClose(toks, name+1, "(", ")")
	if paramsClose < 0 {
		return nil, false
	}
	params := functionParamArrayTypesInRange(file, toks, name+2, paramsClose, namedArrays)
	hasArray := false
	for i := 0; i < len(params); i++ {
		if params[i].elem != "" {
			hasArray = true
			break
		}
	}
	return params, hasArray
}

func functionParamArrayTypesInRange(file *parse.File, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) []arrayTypeLowerInfo {
	var params []arrayTypeLowerInfo
	pending := 0
	segments := topLevelExpressionRanges(toks, start, end)
	for i := 0; i < len(segments); i++ {
		segStart, segEnd := trimTokenRange(toks, segments[i].start, segments[i].end)
		if segStart >= segEnd {
			continue
		}
		info, hasType, hasName := functionParamArraySegment(string(file.Source), toks, segStart, segEnd, namedArrays)
		if hasType {
			if hasName {
				pending++
			}
			count := pending
			if count == 0 {
				count = 1
			}
			for j := 0; j < count; j++ {
				params = append(params, info)
			}
			pending = 0
			continue
		}
		if hasName {
			pending++
			continue
		}
		params = append(params, arrayTypeLowerInfo{})
	}
	for pending > 0 {
		params = append(params, arrayTypeLowerInfo{})
		pending--
	}
	return params
}

func functionLiteralArrayParamTypesForLower(source []byte, toks []scan.Token, literal functionLiteralForLower, namedArrays []namedArrayInfo) []arrayTypeLowerInfo {
	file := parse.File{Source: source}
	return functionParamArrayTypesInRange(&file, toks, literal.paramsOpen+1, literal.paramsClose, namedArrays)
}

func functionLiteralGeneratedArrayParamTypesForLower(source []byte, toks []scan.Token, literal functionLiteralForLower, captures []functionLiteralCapture, namedArrays []namedArrayInfo) []arrayTypeLowerInfo {
	params := make([]arrayTypeLowerInfo, 0, len(captures)+4)
	for i := 0; i < len(captures); i++ {
		params = append(params, arrayTypeLowerInfo{})
	}
	params = append(params, functionLiteralArrayParamTypesForLower(source, toks, literal, namedArrays)...)
	return params
}

func arrayTypeLowerInfosContainArray(params []arrayTypeLowerInfo) bool {
	for i := 0; i < len(params); i++ {
		if params[i].elem != "" {
			return true
		}
	}
	return false
}

func appendFunctionAliasArrayParamInfos(out []arrayFunctionParamLowerInfo, aliases []functionAliasInfo) []arrayFunctionParamLowerInfo {
	for i := 0; i < len(aliases); i++ {
		alias := aliases[i]
		if alias.unitName == "" || !arrayTypeLowerInfosContainArray(alias.arrayParams) {
			continue
		}
		out = append(out, arrayFunctionParamLowerInfo{name: alias.unitName, params: alias.arrayParams})
	}
	return out
}

func functionParamArraySegment(body string, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) (arrayTypeLowerInfo, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return arrayTypeLowerInfo{}, false, false
	}
	typeStart := start
	hasName := false
	if toks[start].Kind == scan.Ident {
		if start+1 < end && isTypeStart(toks[start+1]) {
			typeStart = start + 1
			hasName = true
		} else if start+1 == end {
			return arrayTypeLowerInfo{}, false, true
		}
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, typeStart, end)
	if ok && !info.inferred {
		return info, true, hasName
	}
	if typeStart+1 == end && toks[typeStart].Kind == scan.Ident {
		info, ok := namedArrayInfoByName(namedArrays, toks[typeStart].Text)
		if ok {
			return info, true, hasName
		}
	}
	if typeStart != start || isTypeStart(toks[start]) {
		return arrayTypeLowerInfo{}, true, hasName
	}
	return arrayTypeLowerInfo{}, false, hasName
}

func functionResultArrayTypeInRange(file *parse.File, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return arrayTypeLowerInfo{}, false
	}
	if toks[start].Kind == scan.Ident && start+1 < end && isTypeStart(toks[start+1]) {
		start++
	}
	info, _, _, ok := arrayTypeInfoForRange(string(file.Source), toks, start, end)
	if ok && !info.inferred {
		return info, true
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		return namedArrayInfoByName(namedArrays, toks[start].Text)
	}
	return arrayTypeLowerInfo{}, false
}

func namedTypeUnderlyings(files []parse.File) localTypeTable {
	var types localTypeTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		toks := parsed.Tokens
		for i := 0; i+2 < len(toks); i++ {
			if toks[i].Text != "type" {
				continue
			}
			if toks[i+1].Text == "(" {
				close := findClose(toks, i+1, "(", ")")
				if close < 0 {
					continue
				}
				types = appendGroupedNamedTypeUnderlyings(types, toks, i+2, close)
				i = close
				continue
			}
			if toks[i+1].Kind != scan.Ident {
				continue
			}
			specEnd := typeSpecEnd(toks, i+2)
			types = appendNamedTypeUnderlying(types, toks, i+1, i+2, specEnd)
			i = specEnd - 1
		}
	}
	return types
}

func namedTypeUnderlyingsForLoadFiles(files []load.File) localTypeTable {
	out := make(localTypeTable, 0, namedTypeUnderlyingCapacityForLoadFiles(files))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := namedTypeUnderlyings([]parse.File{parsed})
		tmp = persistLocalTypeTable(tmp)
		out = appendLocalTypeEntries(out, tmp)
		arena.Reset(mark)
	}
	return out
}

func namedTypeUnderlyingCapacityForLoadFiles(files []load.File) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(namedTypeUnderlyings([]parse.File{parsed}))
		}
		arena.Reset(mark)
	}
	return count
}

func namedSliceTypes(files []parse.File, topNames symbolNameTable) []namedSliceInfo {
	var out []namedSliceInfo
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		out = appendNamedSliceTypesForFile(&parsed, topNames, out)
	}
	return out
}

func namedSliceTypesForLoadFiles(files []load.File, topNames symbolNameTable) []namedSliceInfo {
	out := make([]namedSliceInfo, 0, namedSliceTypeCapacityForLoadFiles(files, topNames))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendNamedSliceTypesForFile(&parsed, topNames, nil)
		tmp = persistNamedSliceInfos(tmp)
		for i := 0; i < len(tmp); i++ {
			out = appendNamedSliceInfo(out, tmp[i].name, tmp[i].elem, tmp[i].elemInfo)
		}
		arena.Reset(mark)
	}
	return out
}

func namedSliceTypeCapacityForLoadFiles(files []load.File, topNames symbolNameTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(appendNamedSliceTypesForFile(&parsed, topNames, nil))
		}
		arena.Reset(mark)
	}
	return count
}

func namedArrayTypesForLoadFiles(files []load.File, topNames symbolNameTable) []namedArrayInfo {
	out := make([]namedArrayInfo, 0, namedArrayTypeCapacityForLoadFiles(files, topNames))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendNamedArrayTypesForFile(&parsed, topNames, nil)
		for i := 0; i < len(tmp); i++ {
			tmp[i].name = arena.PersistString(tmp[i].name)
			tmp[i].info.elem = arena.PersistString(tmp[i].info.elem)
			out = appendNamedArrayInfo(out, tmp[i].name, tmp[i].info)
		}
		arena.Reset(mark)
	}
	return out
}

func namedArrayTypeCapacityForLoadFiles(files []load.File, topNames symbolNameTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(appendNamedArrayTypesForFile(&parsed, topNames, nil))
		}
		arena.Reset(mark)
	}
	return count
}

func namedMapTypesForLoadFiles(files []load.File, topNames symbolNameTable) []namedMapInfo {
	out := make([]namedMapInfo, 0, namedMapTypeCapacityForLoadFiles(files, topNames))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendNamedMapTypesForFile(&parsed, topNames, nil)
		for i := 0; i < len(tmp); i++ {
			tmp[i].name = arena.PersistString(tmp[i].name)
			tmp[i].keyType = arena.PersistString(tmp[i].keyType)
			tmp[i].valueType = arena.PersistString(tmp[i].valueType)
			out = appendNamedMapInfo(out, tmp[i].name, tmp[i].keyType, tmp[i].valueType)
		}
		arena.Reset(mark)
	}
	return out
}

func namedMapTypeCapacityForLoadFiles(files []load.File, topNames symbolNameTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(appendNamedMapTypesForFile(&parsed, topNames, nil))
		}
		arena.Reset(mark)
	}
	return count
}

func appendNamedSliceTypesForFile(parsed *parse.File, topNames symbolNameTable, out []namedSliceInfo) []namedSliceInfo {
	toks := parsed.Tokens
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			out = appendGroupedNamedSliceTypes(out, toks, i+2, close, topNames)
			i = close
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := packageTypeSpecEndForLower(toks, i+1)
		out = appendNamedSliceType(out, toks, i+1, i+2, specEnd, topNames)
		i = specEnd - 1
	}
	return out
}

func appendNamedArrayTypesForFile(parsed *parse.File, topNames symbolNameTable, out []namedArrayInfo) []namedArrayInfo {
	toks := parsed.Tokens
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			out = appendGroupedNamedArrayTypes(out, toks, i+2, close, topNames)
			i = close
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := packageTypeSpecEndForLower(toks, i+1)
		out = appendNamedArrayType(out, toks, i+1, i+2, specEnd, topNames)
		i = specEnd - 1
	}
	return out
}

func appendNamedMapTypesForFile(parsed *parse.File, topNames symbolNameTable, out []namedMapInfo) []namedMapInfo {
	toks := parsed.Tokens
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			out = appendGroupedNamedMapTypes(out, toks, i+2, close, topNames)
			i = close
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := packageTypeSpecEndForLower(toks, i+1)
		out = appendNamedMapType(out, toks, i+1, i+2, specEnd, topNames)
		i = specEnd - 1
	}
	return out
}

func appendDependencyNamedSliceTypes(out []namedSliceInfo, depSlices []namedSliceInfo) []namedSliceInfo {
	for i := 0; i < len(depSlices); i++ {
		info := depSlices[i]
		out = appendNamedSliceInfo(out, info.name, info.elem, info.elemInfo)
	}
	return out
}

func appendDependencyNamedArrayTypes(out []namedArrayInfo, depArrays []namedArrayInfo) []namedArrayInfo {
	for i := 0; i < len(depArrays); i++ {
		info := depArrays[i]
		out = appendNamedArrayInfo(out, info.name, info.info)
	}
	return out
}

func appendDependencyNamedMapTypes(out []namedMapInfo, depMaps []namedMapInfo) []namedMapInfo {
	for i := 0; i < len(depMaps); i++ {
		info := depMaps[i]
		out = appendNamedMapInfo(out, info.name, info.keyType, info.valueType)
	}
	return out
}

func namedConversionTypes(files []parse.File, topNames symbolNameTable) []string {
	var out []string
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		out = appendNamedConversionTypesForFile(&parsed, topNames, out)
	}
	return out
}

func namedConversionTypesForLoadFiles(files []load.File, topNames symbolNameTable) []string {
	out := make([]string, 0, namedConversionTypeCapacityForLoadFiles(files, topNames))
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := appendNamedConversionTypesForFile(&parsed, topNames, nil)
		for i := 0; i < len(tmp); i++ {
			out = append(out, arena.PersistString(tmp[i]))
		}
		arena.Reset(mark)
	}
	return out
}

func namedConversionTypeCapacityForLoadFiles(files []load.File, topNames symbolNameTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(appendNamedConversionTypesForFile(&parsed, topNames, nil))
		}
		arena.Reset(mark)
	}
	return count
}

func appendNamedConversionTypesForFile(parsed *parse.File, topNames symbolNameTable, out []string) []string {
	toks := parsed.Tokens
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			out = appendGroupedNamedConversionTypes(out, toks, i+2, close, topNames)
			i = close
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := typeSpecEnd(toks, i+2)
		if namedConversionNeedsNormalization(toks, i+2, specEnd) {
			name := toks[i+1].Text
			if name != "" && name != "_" {
				out = append(out, name)
				unitName := symbolNameTableUnitName(topNames, name)
				if unitName != "" {
					out = append(out, unitName)
				}
			}
		}
		i = specEnd - 1
	}
	return out
}

func appendGroupedNamedConversionTypes(out []string, toks []scan.Token, start int, end int, topNames symbolNameTable) []string {
	specStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			if specStart < i && toks[specStart].Kind == scan.Ident {
				if namedConversionNeedsNormalization(toks, specStart+1, i) {
					name := toks[specStart].Text
					if name != "" && name != "_" {
						out = append(out, name)
						unitName := symbolNameTableUnitName(topNames, name)
						if unitName != "" {
							out = append(out, unitName)
						}
					}
				}
			}
			specStart = i
			if i < end && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return out
}

func appendNamedConversionType(out []string, toks []scan.Token, namePos int, typeStart int, typeEnd int, topNames symbolNameTable) []string {
	if !namedConversionNeedsNormalization(toks, typeStart, typeEnd) {
		return out
	}
	name := toks[namePos].Text
	if name == "" || name == "_" {
		return out
	}
	out = appendUniqueString(out, name)
	unitName := symbolNameTableUnitName(topNames, name)
	if unitName != "" {
		out = appendUniqueString(out, unitName)
	}
	return out
}

func namedConversionNeedsNormalization(toks []scan.Token, typeStart int, typeEnd int) bool {
	if typeStart < typeEnd && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= typeEnd {
		return false
	}
	if toks[typeStart].Text == "struct" || toks[typeStart].Text == "*" {
		return true
	}
	if typeStart+1 < typeEnd && toks[typeStart].Text == "[" && toks[typeStart+1].Text == "]" {
		return true
	}
	if toks[typeStart].Kind == scan.Ident {
		return !isBuiltinScalarTypeName(toks[typeStart].Text)
	}
	return false
}

func isBuiltinScalarTypeName(name string) bool {
	return name == "int" || name == "int64" || name == "int32" || name == "int16" || name == "byte" || name == "bool" || name == "string" || name == "float64" || name == "error"
}

func appendDependencyNamedConversionTypes(out []string, depConversions []string) []string {
	for i := 0; i < len(depConversions); i++ {
		out = append(out, depConversions[i])
	}
	return out
}

func appendUniqueString(out []string, value string) []string {
	if value == "" {
		return out
	}
	for i := 0; i < len(out); i++ {
		if out[i] == value {
			return out
		}
	}
	return append(out, value)
}

func packageSymbolNames(importPath string, files []parse.File, capacity int) symbolNameTable {
	names := make(symbolNameTable, 0, capacity)
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		names = appendPackageSymbolNamesForFile(names, importPath, &parsed)
	}
	return names
}

func packageSymbolNamesForLoadFiles(importPath string, files []load.File, capacity int) symbolNameTable {
	names := make(symbolNameTable, 0, capacity)
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		names = appendPackageSymbolNamesForFile(names, importPath, &parsed)
		for i := 0; i < len(names); i++ {
			names[i].name = arena.PersistString(names[i].name)
			names[i].unitName = arena.PersistString(names[i].unitName)
		}
		arena.Reset(mark)
	}
	return names
}

func appendPackageSymbolNamesForFile(names symbolNameTable, importPath string, parsed *parse.File) symbolNameTable {
	for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
		decl := parsed.Decls[declIndex]
		declNames := declTopNames(parsed, &decl)
		for nameIndex := 0; nameIndex < len(declNames); nameIndex++ {
			name := declNames[nameIndex]
			if name == "" || name == "_" {
				continue
			}
			names = symbolNameTableSet(names, name, SymbolName(importPath, name))
		}
	}
	return names
}

func appendGroupedNamedSliceTypes(out []namedSliceInfo, toks []scan.Token, start int, end int, topNames symbolNameTable) []namedSliceInfo {
	specStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			if specStart < i && toks[specStart].Kind == scan.Ident {
				out = appendNamedSliceType(out, toks, specStart, specStart+1, i, topNames)
			}
			specStart = i
			if i < end && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return out
}

func appendNamedSliceType(out []namedSliceInfo, toks []scan.Token, namePos int, typeStart int, typeEnd int, topNames symbolNameTable) []namedSliceInfo {
	if typeStart < typeEnd && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+2 >= typeEnd || toks[typeStart].Text != "[" || toks[typeStart+1].Text != "]" {
		return out
	}
	name := toks[namePos].Text
	elem := loweredTypeText(toks, typeStart+2, typeEnd, topNames)
	if elem == "" {
		return out
	}
	elemInfo := typeInfoInRange(toks, typeStart+2, typeEnd)
	elemInfo.pointer = typeRangeIsPointer(toks, typeStart+2, typeEnd)
	out = appendNamedSliceInfo(out, name, elem, elemInfo)
	unitName := symbolNameTableUnitName(topNames, name)
	if unitName != "" && unitName != name {
		out = appendNamedSliceInfo(out, unitName, elem, elemInfo)
	}
	return out
}

func appendNamedSliceInfo(out []namedSliceInfo, name string, elem string, elemInfo localTypeInfo) []namedSliceInfo {
	for i := 0; i < len(out); i++ {
		if out[i].name == name {
			out[i].elem = elem
			out[i].elemInfo = elemInfo
			return out
		}
	}
	return append(out, namedSliceInfo{name: name, elem: elem, elemInfo: elemInfo})
}

func appendGroupedNamedArrayTypes(out []namedArrayInfo, toks []scan.Token, start int, end int, topNames symbolNameTable) []namedArrayInfo {
	specStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			if specStart < i && toks[specStart].Kind == scan.Ident {
				out = appendNamedArrayType(out, toks, specStart, specStart+1, i, topNames)
			}
			specStart = i
			if i < end && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return out
}

func appendNamedArrayType(out []namedArrayInfo, toks []scan.Token, namePos int, typeStart int, typeEnd int, topNames symbolNameTable) []namedArrayInfo {
	if typeStart < typeEnd && toks[typeStart].Text == "=" {
		typeStart++
	}
	info, ok := namedArrayTypeInfoForRange(toks, typeStart, typeEnd, topNames)
	if !ok {
		return out
	}
	name := toks[namePos].Text
	if name == "" || name == "_" {
		return out
	}
	out = appendNamedArrayInfo(out, name, info)
	unitName := symbolNameTableUnitName(topNames, name)
	if unitName != "" && unitName != name {
		out = appendNamedArrayInfo(out, unitName, info)
	}
	return out
}

func namedArrayTypeInfoForRange(toks []scan.Token, start int, end int, topNames symbolNameTable) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return arrayTypeLowerInfo{}, false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return arrayTypeLowerInfo{}, false
	}
	length, inferred, ok := arrayLengthForTokens(toks, start+1, brackClose)
	if !ok || inferred || length < 0 {
		return arrayTypeLowerInfo{}, false
	}
	elemStart := brackClose + 1
	if elemStart >= end || toks[elemStart].Text == "[" {
		return arrayTypeLowerInfo{}, false
	}
	elem := loweredTypeText(toks, elemStart, end, topNames)
	if elem == "" {
		return arrayTypeLowerInfo{}, false
	}
	return arrayTypeLowerInfo{elem: elem, length: length}, true
}

func appendNamedArrayInfo(out []namedArrayInfo, name string, info arrayTypeLowerInfo) []namedArrayInfo {
	if name == "" || info.elem == "" {
		return out
	}
	for i := 0; i < len(out); i++ {
		if out[i].name == name {
			out[i].info = info
			return out
		}
	}
	return append(out, namedArrayInfo{name: name, info: info})
}

func namedArrayInfoByName(values []namedArrayInfo, name string) (arrayTypeLowerInfo, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i].info, true
		}
	}
	return arrayTypeLowerInfo{}, false
}

func appendGroupedNamedMapTypes(out []namedMapInfo, toks []scan.Token, start int, end int, topNames symbolNameTable) []namedMapInfo {
	specStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			if specStart < i && toks[specStart].Kind == scan.Ident {
				out = appendNamedMapType(out, toks, specStart, specStart+1, i, topNames)
			}
			specStart = i
			if i < end && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return out
}

func appendNamedMapType(out []namedMapInfo, toks []scan.Token, namePos int, typeStart int, typeEnd int, topNames symbolNameTable) []namedMapInfo {
	if typeStart < typeEnd && toks[typeStart].Text == "=" {
		typeStart++
	}
	keyType, valueType := mapTypeKeyValueTextForLower(toks, typeStart, typeEnd)
	if keyType == "" || valueType == "" {
		return out
	}
	if topNames != nil {
		keyClose := findClose(toks, typeStart+1, "[", "]")
		if keyClose >= 0 {
			keyType = loweredTypeText(toks, typeStart+2, keyClose, topNames)
			valueType = loweredTypeText(toks, keyClose+1, typeEnd, topNames)
		}
	}
	if keyType == "" || valueType == "" {
		return out
	}
	name := toks[namePos].Text
	if name == "" || name == "_" {
		return out
	}
	out = appendNamedMapInfo(out, name, keyType, valueType)
	unitName := symbolNameTableUnitName(topNames, name)
	if unitName != "" && unitName != name {
		out = appendNamedMapInfo(out, unitName, keyType, valueType)
	}
	return out
}

func appendNamedMapInfo(out []namedMapInfo, name string, keyType string, valueType string) []namedMapInfo {
	if name == "" || keyType == "" || valueType == "" {
		return out
	}
	for i := 0; i < len(out); i++ {
		if out[i].name == name {
			out[i].keyType = keyType
			out[i].valueType = valueType
			return out
		}
	}
	return append(out, namedMapInfo{name: name, keyType: keyType, valueType: valueType})
}

func namedMapInfoByName(values []namedMapInfo, name string) (namedMapInfo, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i], true
		}
	}
	return namedMapInfo{}, false
}

func rewriteNamedSliceAnonymousStructElems(values []namedSliceInfo, decls []localTypeDeclInfo) []namedSliceInfo {
	if len(values) == 0 || len(decls) == 0 {
		return values
	}
	for i := 0; i < len(values); i++ {
		unitName := anonymousStructGeneratedNameForKey(decls, values[i].elem)
		if unitName == "" {
			continue
		}
		values[i].elem = unitName
		values[i].elemInfo = localTypeInfo{name: unitName}
	}
	return values
}

func anonymousStructGeneratedNameForKey(decls []localTypeDeclInfo, key string) string {
	if key == "" {
		return ""
	}
	for i := 0; i < len(decls); i++ {
		info := decls[i]
		if info.anonymous && info.typeKey == key && info.unitName != "" {
			return info.unitName
		}
	}
	return ""
}

func persistNamedSliceInfos(values []namedSliceInfo) []namedSliceInfo {
	for i := 0; i < len(values); i++ {
		values[i].name = arena.PersistString(values[i].name)
		values[i].elem = arena.PersistString(values[i].elem)
		values[i].elemInfo = persistLocalTypeInfo(values[i].elemInfo)
	}
	return values
}

func persistLocalTypeTable(values localTypeTable) localTypeTable {
	for i := 0; i < len(values); i++ {
		values[i].name = arena.PersistString(values[i].name)
		values[i].info = persistLocalTypeInfo(values[i].info)
	}
	return values
}

func persistLocalTypeInfo(info localTypeInfo) localTypeInfo {
	info.qualifier = arena.PersistString(info.qualifier)
	info.name = arena.PersistString(info.name)
	return info
}

func persistStructFieldTypeTable(values structFieldTypeTable) structFieldTypeTable {
	for i := 0; i < len(values); i++ {
		values[i].owner = arena.PersistString(values[i].owner)
		values[i].field = arena.PersistString(values[i].field)
		values[i].info = persistLocalTypeInfo(values[i].info)
	}
	return values
}

func persistStructOwnerTable(values structOwnerTable) structOwnerTable {
	for i := 0; i < len(values); i++ {
		values[i] = arena.PersistString(values[i])
	}
	return values
}

func loweredTypeText(toks []scan.Token, start int, end int, topNames symbolNameTable) string {
	var out []byte
	for i := start; i < end; i++ {
		text := toks[i].Text
		if toks[i].Kind == scan.Ident {
			unitName := symbolNameTableUnitName(topNames, text)
			if unitName != "" {
				text = unitName
			}
		}
		out = appendString(out, text)
	}
	return string(out)
}

func appendGroupedNamedTypeUnderlyings(types localTypeTable, toks []scan.Token, start int, end int) localTypeTable {
	specStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			if specStart < i && toks[specStart].Kind == scan.Ident {
				types = appendNamedTypeUnderlying(types, toks, specStart, specStart+1, i)
			}
			specStart = i
			if i < end && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return types
}

func appendNamedTypeUnderlying(types localTypeTable, toks []scan.Token, namePos int, typeStart int, typeEnd int) localTypeTable {
	if typeStart < typeEnd && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart >= typeEnd || toks[typeStart].Text == "struct" {
		return types
	}
	typ := typeInfoInRange(toks, typeStart, typeEnd)
	if typ.name == "" {
		return types
	}
	typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
	return localTypeTableSet(types, toks[namePos].Text, typ)
}

func typeSpecEnd(toks []scan.Token, start int) int {
	if start >= len(toks) {
		return start
	}
	line := toks[start].Line
	for i := start; i < len(toks); i++ {
		if toks[i].Kind == scan.EOF || toks[i].Text == ")" || toks[i].Text == "}" {
			return i
		}
		if toks[i].Text == ";" {
			return i
		}
		if toks[i].Line != line {
			return i
		}
	}
	return len(toks)
}

func packageStructFieldTypes(files []parse.File, namedTypes localTypeTable) structFieldTypeTable {
	var fields structFieldTypeTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		toks := parsed.Tokens
		for i := 0; i+2 < len(toks); i++ {
			if toks[i].Text != "type" {
				continue
			}
			if toks[i+1].Text == "(" {
				close := findClose(toks, i+1, "(", ")")
				if close < 0 {
					continue
				}
				ranges := localConstSpecRanges(toks, i+2, close)
				for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
					fields = appendPackageStructFieldTypeSpec(fields, toks, ranges[rangeIndex].start, ranges[rangeIndex].end, namedTypes)
				}
				i = close
				continue
			}
			if toks[i+1].Kind != scan.Ident {
				continue
			}
			specEnd := packageTypeSpecEndForLower(toks, i+1)
			fields = appendPackageStructFieldTypeSpec(fields, toks, i+1, specEnd, namedTypes)
			i = specEnd - 1
		}
	}
	return fields
}

func packageStructOwners(files []parse.File) structOwnerTable {
	var owners structOwnerTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		toks := parsed.Tokens
		for i := 0; i+2 < len(toks); i++ {
			if toks[i].Text != "type" {
				continue
			}
			if toks[i+1].Text == "(" {
				close := findClose(toks, i+1, "(", ")")
				if close < 0 {
					continue
				}
				ranges := localConstSpecRanges(toks, i+2, close)
				for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
					owners = appendPackageStructOwnerSpec(owners, toks, ranges[rangeIndex].start, ranges[rangeIndex].end)
				}
				i = close
				continue
			}
			if toks[i+1].Kind != scan.Ident {
				continue
			}
			specEnd := packageTypeSpecEndForLower(toks, i+1)
			owners = appendPackageStructOwnerSpec(owners, toks, i+1, specEnd)
			i = specEnd - 1
		}
	}
	return owners
}

func appendPackageStructFieldTypeSpec(fields structFieldTypeTable, toks []scan.Token, start int, end int, namedTypes localTypeTable) structFieldTypeTable {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start+2 >= end || toks[start].Kind != scan.Ident {
		return fields
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "struct" || toks[typeStart+1].Text != "{" {
		return fields
	}
	close := findClose(toks, typeStart+1, "{", "}")
	if close < 0 || close > end {
		return fields
	}
	return appendStructFieldTypeEntries(fields, toks[start].Text, toks, typeStart+1, close, namedTypes)
}

func appendPackageStructOwnerSpec(owners structOwnerTable, toks []scan.Token, start int, end int) structOwnerTable {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start+2 >= end || toks[start].Kind != scan.Ident {
		return owners
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "struct" || toks[typeStart+1].Text != "{" {
		return owners
	}
	close := findClose(toks, typeStart+1, "{", "}")
	if close < 0 || close > end {
		return owners
	}
	return structOwnerTableSet(owners, toks[start].Text)
}

func packageStructFieldTypesForLoadFiles(files []load.File, importPath string, namedTypes localTypeTable) structFieldTypeTable {
	out := make(structFieldTypeTable, 0, structFieldTypeCapacityForLoadFiles(files, namedTypes))
	anonymousTypes := packageAnonymousStructTypeDeclsForLoadFiles(files, importPath)
	if len(anonymousTypes) > 0 {
		namedTypes = appendGeneratedTypeUnderlyingsFromBodies(namedTypes, anonymousTypes)
	}
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := packageStructFieldTypes([]parse.File{parsed}, namedTypes)
		tmp = persistStructFieldTypeTable(tmp)
		for i := 0; i < len(tmp); i++ {
			out = structFieldTypeTableSet(out, tmp[i].owner, tmp[i].field, tmp[i].info)
		}
		arena.Reset(mark)
	}
	if len(anonymousTypes) > 0 {
		out = appendAnonymousStructOwnerFieldTypesForLoadFiles(out, files, anonymousTypes)
		out = appendGeneratedStructFieldTypesFromBodies(out, anonymousTypes, namedTypes)
	}
	return out
}

func packageStructOwnersForLoadFiles(files []load.File) structOwnerTable {
	var out structOwnerTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := packageStructOwners([]parse.File{parsed})
		tmp = persistStructOwnerTable(tmp)
		out = appendStructOwnerTables(out, tmp)
		arena.Reset(mark)
	}
	return out
}

func packageArrayStructFieldLowerInfos(files []parse.File, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	var fields arrayStructFieldLowerInfoTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed := files[fileIndex]
		toks := parsed.Tokens
		for i := 0; i+2 < len(toks); i++ {
			if toks[i].Text != "type" {
				continue
			}
			if toks[i+1].Text == "(" {
				close := findClose(toks, i+1, "(", ")")
				if close < 0 {
					continue
				}
				ranges := localConstSpecRanges(toks, i+2, close)
				for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
					fields = appendPackageArrayStructFieldLowerInfoSpec(fields, toks, ranges[rangeIndex].start, ranges[rangeIndex].end, namedArrays)
				}
				i = close
				continue
			}
			if toks[i+1].Kind != scan.Ident {
				continue
			}
			specEnd := packageTypeSpecEndForLower(toks, i+1)
			fields = appendPackageArrayStructFieldLowerInfoSpec(fields, toks, i+1, specEnd, namedArrays)
			i = specEnd - 1
		}
	}
	return fields
}

func appendPackageArrayStructFieldLowerInfoSpec(fields arrayStructFieldLowerInfoTable, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start+2 >= end || toks[start].Kind != scan.Ident {
		return fields
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "struct" || toks[typeStart+1].Text != "{" {
		return fields
	}
	close := findClose(toks, typeStart+1, "{", "}")
	if close < 0 || close > end {
		return fields
	}
	return appendArrayStructFieldLowerInfoEntries(fields, toks[start].Text, toks, typeStart+1, close, namedArrays)
}

func packageArrayStructFieldLowerInfosForLoadFiles(files []load.File, importPath string, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	var out arrayStructFieldLowerInfoTable
	anonymousTypes := packageAnonymousStructTypeDeclsForLoadFiles(files, importPath)
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		tmp := packageArrayStructFieldLowerInfos([]parse.File{parsed}, namedArrays)
		for i := 0; i < len(tmp); i++ {
			tmp[i].owner = arena.PersistString(tmp[i].owner)
			tmp[i].field = arena.PersistString(tmp[i].field)
			tmp[i].info.elem = arena.PersistString(tmp[i].info.elem)
			out = arrayStructFieldLowerInfoTableSet(out, tmp[i].owner, tmp[i].field, tmp[i].info)
		}
		arena.Reset(mark)
	}
	if len(anonymousTypes) > 0 {
		out = appendGeneratedArrayStructFieldLowerInfosFromBodies(out, anonymousTypes, namedArrays)
	}
	return out
}

func appendAnonymousStructOwnerFieldTypesForLoadFiles(out structFieldTypeTable, files []load.File, decls []localTypeDeclInfo) structFieldTypeTable {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		out = appendAnonymousStructOwnerFieldTypes(out, &parsed, decls)
		arena.Reset(mark)
	}
	return out
}

func appendAnonymousStructOwnerFieldTypes(out structFieldTypeTable, file *parse.File, decls []localTypeDeclInfo) structFieldTypeTable {
	toks := file.Tokens
	for i := 0; i+3 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Text == "(" {
			groupClose := findClose(toks, i+1, "(", ")")
			if groupClose < 0 {
				continue
			}
			ranges := localConstSpecRanges(toks, i+2, groupClose)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				out = appendAnonymousStructOwnerFieldTypesInTypeSpec(out, file, ranges[rangeIndex].start, ranges[rangeIndex].end, decls)
			}
			i = groupClose
			continue
		}
		if toks[i+1].Kind != scan.Ident {
			continue
		}
		specEnd := packageTypeSpecEndForLower(toks, i+1)
		out = appendAnonymousStructOwnerFieldTypesInTypeSpec(out, file, i+1, specEnd, decls)
		i = specEnd - 1
	}
	return out
}

func appendAnonymousStructOwnerFieldTypesInTypeSpec(out structFieldTypeTable, file *parse.File, start int, end int, decls []localTypeDeclInfo) structFieldTypeTable {
	toks := file.Tokens
	for start < end && toks[start].Text == ";" {
		start++
	}
	if start+2 >= end || toks[start].Kind != scan.Ident {
		return out
	}
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	if typeStart+1 >= end || toks[typeStart].Text != "struct" || toks[typeStart+1].Text != "{" {
		return out
	}
	owner := toks[start].Text
	close := findClose(toks, typeStart+1, "{", "}")
	if close < 0 || close >= end {
		return out
	}
	ranges := anonymousStructFieldSpecRangesForLower(toks, typeStart+1, close)
	for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
		specStart := ranges[rangeIndex].start
		specEnd := ranges[rangeIndex].end
		typeStart, typeEnd, ok := anonymousStructFieldTypeRangeForLower(toks, specStart, specEnd)
		if !ok {
			continue
		}
		unitName := anonymousStructGeneratedNameForRange(toks, file.Path, typeStart, typeEnd, decls)
		if unitName == "" {
			continue
		}
		prefix := anonymousStructFieldTypePrefixForLower(toks, specStart, typeStart)
		fieldType := localTypeInfo{name: prefix + unitName}
		if prefix == "*" {
			fieldType = localTypeInfo{name: unitName, pointer: true}
		}
		fields := fieldNamesBeforeTypeForLower(toks, specStart, typeStart)
		for fieldIndex := 0; fieldIndex < len(fields); fieldIndex++ {
			out = structFieldTypeTableSet(out, owner, fields[fieldIndex], fieldType)
		}
	}
	return out
}

func anonymousStructFieldTypePrefixForLower(toks []scan.Token, start int, typeStart int) string {
	var out []byte
	for i := start; i < typeStart; i++ {
		if toks[i].Text == "[" && i+1 < typeStart && toks[i+1].Text == "]" {
			out = appendString(out, "[]")
			i++
			continue
		}
		if toks[i].Text == "*" {
			out = append(out, '*')
		}
	}
	return string(out)
}

func fieldNamesBeforeTypeForLower(toks []scan.Token, start int, typeStart int) []string {
	var out []string
	for i := start; i < typeStart; i++ {
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			out = append(out, toks[i].Text)
		}
	}
	return out
}

func importedStructFieldTypesForFile(file *parse.File, packages []load.Package) structFieldTypeTable {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var out structFieldTypeTable
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		dep, ok := packageByImportPath(packages, imp.Path)
		if !ok {
			continue
		}
		namedTypes := namedTypeUnderlyingsForLoadFiles(dep.Files)
		depFields := packageStructFieldTypesForLoadFiles(dep.Files, dep.ImportPath, namedTypes)
		for fieldIndex := 0; fieldIndex < len(depFields); fieldIndex++ {
			entry := depFields[fieldIndex]
			if !isExported(entry.owner) || !isExported(entry.field) {
				continue
			}
			owner := importedQualifiedLowerName(localName, entry.owner)
			info := qualifyImportedFieldTypeInfo(localName, depFields, namedTypes, entry.info)
			out = structFieldTypeTableSet(out, owner, entry.field, info)
			unitOwner := SymbolName(dep.ImportPath, entry.owner)
			out = structFieldTypeTableSet(out, unitOwner, entry.field, info)
		}
	}
	return out
}

func importedArrayStructFieldLowerInfosForFile(file *parse.File, packages []load.Package) arrayStructFieldLowerInfoTable {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var out arrayStructFieldLowerInfoTable
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		dep, ok := packageByImportPath(packages, imp.Path)
		if !ok {
			continue
		}
		depTopNames := packageSymbolNamesForLoadFiles(dep.ImportPath, dep.Files, packageSymbolCapacity(dep.Files, len(dep.Imports)))
		depNamedArrays := namedArrayTypesForLoadFiles(dep.Files, depTopNames)
		depArrayFields := packageArrayStructFieldLowerInfosForLoadFiles(dep.Files, dep.ImportPath, depNamedArrays)
		for fieldIndex := 0; fieldIndex < len(depArrayFields); fieldIndex++ {
			entry := depArrayFields[fieldIndex]
			if !isExported(entry.owner) || !isExported(entry.field) {
				continue
			}
			owner := importedQualifiedLowerName(localName, entry.owner)
			out = arrayStructFieldLowerInfoTableSet(out, owner, entry.field, entry.info)
			unitOwner := SymbolName(dep.ImportPath, entry.owner)
			out = arrayStructFieldLowerInfoTableSet(out, unitOwner, entry.field, entry.info)
		}
	}
	return out
}

func importedTopLevelValueTypesForFile(file *parse.File, packages []load.Package) localTypeTable {
	if len(packages) == 0 || len(file.Imports) == 0 {
		return nil
	}
	var out localTypeTable
	for importIndex := 0; importIndex < len(file.Imports); importIndex++ {
		imp := file.Imports[importIndex]
		localName := importLocalName(imp)
		if localName == "" || localName == "_" {
			continue
		}
		dep, ok := packageByImportPath(packages, imp.Path)
		if !ok {
			continue
		}
		namedTypes := namedTypeUnderlyingsForLoadFiles(dep.Files)
		depFields := packageStructFieldTypesForLoadFiles(dep.Files, dep.ImportPath, namedTypes)
		for fileIndex := 0; fileIndex < len(dep.Files); fileIndex++ {
			parsed, err := parsedLoadFile(dep.Files[fileIndex])
			if err != nil {
				continue
			}
			out = appendImportedTopLevelValueTypesForParsed(out, &parsed, localName, dep.ImportPath, depFields, namedTypes)
		}
	}
	return out
}

func packageTopLevelValueTypesForLoadFiles(files []load.File, topNames symbolNameTable) localTypeTable {
	var out localTypeTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			continue
		}
		out = appendPackageTopLevelValueTypesForParsed(out, &parsed, topNames)
	}
	return out
}

func appendPackageTopLevelValueTypesForParsed(out localTypeTable, file *parse.File, topNames symbolNameTable) localTypeTable {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specs := localConstSpecRanges(toks, start+2, close)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				out = appendPackageTopLevelValueSpecTypes(out, toks, specs[specIndex], topNames)
			}
			continue
		}
		end := localConstSingleEnd(toks, start, decl.End)
		if end <= start+1 {
			continue
		}
		out = appendPackageTopLevelValueSpecTypes(out, toks, expressionRange{start: start + 1, end: end}, topNames)
	}
	return out
}

func appendPackageTopLevelValueSpecTypes(out localTypeTable, toks []scan.Token, spec expressionRange, topNames symbolNameTable) localTypeTable {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return out
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if len(names) == 0 {
		return out
	}
	var types []localTypeInfo
	if typeStart >= 0 {
		info := fixedArrayLocalTypeInfoForRange(toks, typeStart, prefixEnd)
		if info.name == "" {
			info = typeInfoInRange(toks, typeStart, prefixEnd)
		}
		if info.name != "" {
			info.pointer = typeRangeIsPointer(toks, typeStart, prefixEnd)
			types = append(types, info)
		}
	} else if eq >= 0 {
		values := topLevelExpressionRanges(toks, eq+1, end)
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			info := importedValueInitializerTypeInfo(toks, values[valueIndex])
			if info.name != "" {
				types = append(types, info)
			}
		}
	}
	if len(types) == 0 {
		return out
	}
	for i := 0; i < len(names); i++ {
		typeIndex := i
		if typeIndex >= len(types) {
			typeIndex = len(types) - 1
		}
		info := types[typeIndex]
		if info.name == "" {
			continue
		}
		out = localTypeTableSet(out, names[i], info)
		if unitName := symbolNameTableUnitName(topNames, names[i]); unitName != "" {
			out = localTypeTableSet(out, unitName, info)
		}
	}
	return out
}

func appendImportedTopLevelValueTypesForParsed(out localTypeTable, file *parse.File, localName string, importPath string, depFields structFieldTypeTable, namedTypes localTypeTable) localTypeTable {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close <= start+1 {
				continue
			}
			specs := localConstSpecRanges(toks, start+2, close)
			for specIndex := 0; specIndex < len(specs); specIndex++ {
				out = appendImportedTopLevelValueSpecTypes(out, toks, specs[specIndex], localName, importPath, depFields, namedTypes)
			}
			continue
		}
		end := localConstSingleEnd(toks, start, decl.End)
		if end <= start+1 {
			continue
		}
		out = appendImportedTopLevelValueSpecTypes(out, toks, expressionRange{start: start + 1, end: end}, localName, importPath, depFields, namedTypes)
	}
	return out
}

func appendImportedTopLevelValueSpecTypes(out localTypeTable, toks []scan.Token, spec expressionRange, localName string, importPath string, depFields structFieldTypeTable, namedTypes localTypeTable) localTypeTable {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return out
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if len(names) == 0 {
		return out
	}
	var types []localTypeInfo
	if typeStart >= 0 {
		info := fixedArrayLocalTypeInfoForRange(toks, typeStart, prefixEnd)
		if info.name == "" {
			info = typeInfoInRange(toks, typeStart, prefixEnd)
		}
		if info.name != "" {
			info.pointer = typeRangeIsPointer(toks, typeStart, prefixEnd)
			types = append(types, info)
		}
	} else if eq >= 0 {
		values := topLevelExpressionRanges(toks, eq+1, end)
		for valueIndex := 0; valueIndex < len(values); valueIndex++ {
			info := importedValueInitializerTypeInfo(toks, values[valueIndex])
			if info.name != "" {
				types = append(types, info)
			}
		}
	}
	if len(types) == 0 {
		return out
	}
	for i := 0; i < len(names); i++ {
		if !isExported(names[i]) {
			continue
		}
		typeIndex := i
		if typeIndex >= len(types) {
			typeIndex = len(types) - 1
		}
		info := qualifyImportedFieldTypeInfo(localName, depFields, namedTypes, types[typeIndex])
		if info.name == "" {
			continue
		}
		out = localTypeTableSet(out, importedQualifiedLowerName(localName, names[i]), info)
		if importPath != "" {
			out = localTypeTableSet(out, SymbolName(importPath, names[i]), info)
		}
	}
	return out
}

func importedValueInitializerTypeInfo(toks []scan.Token, value expressionRange) localTypeInfo {
	start, end := trimTokenRange(toks, value.start, value.end)
	start, end = trimOuterParens(toks, start, end)
	if start >= end {
		return localTypeInfo{}
	}
	if toks[start].Text == "&" {
		start++
	}
	open := compositeLiteralOpenForTypeStart(toks, start)
	if open < 0 || open >= end {
		return localTypeInfo{}
	}
	info := fixedArrayLocalTypeInfoForRange(toks, start, open)
	if info.name == "" {
		info = typeInfoInRange(toks, start, open)
	}
	info.pointer = value.start < value.end && toks[value.start].Text == "&"
	return info
}

func fixedArrayLocalTypeInfoForRange(toks []scan.Token, start int, end int) localTypeInfo {
	info, ok := fixedArrayTypeLowerInfoForTokens(toks, start, end)
	if !ok {
		return localTypeInfo{}
	}
	return localTypeInfo{name: arrayTypeLowerInfoFixedName(info)}
}

func qualifyImportedFieldTypeInfo(localName string, depFields structFieldTypeTable, namedTypes localTypeTable, info localTypeInfo) localTypeInfo {
	qualified := qualifyImportedFieldTypeName(localName, depFields, namedTypes, info.qualifier, info.name)
	dot := strings.IndexByte(qualified, '.')
	if dot >= 0 {
		info.qualifier = qualified[:dot]
		info.name = qualified[dot+1:]
		return info
	}
	info.qualifier = ""
	info.name = qualified
	return info
}

func qualifyImportedFieldTypeName(localName string, depFields structFieldTypeTable, namedTypes localTypeTable, qualifier string, name string) string {
	if name == "" || localName == "" || localName == "_" {
		return name
	}
	if strings.Contains(name, ".") {
		return name
	}
	if qualifier != "" {
		return qualifier + "." + name
	}
	if localTypeTableLookup(namedTypes, name).name != "" || structFieldTypeTableOwnerExists(depFields, name) {
		return importedQualifiedLowerName(localName, name)
	}
	return name
}

func importedQualifiedLowerName(localName string, name string) string {
	if localName == "." {
		return name
	}
	return localName + "." + name
}

func structFieldTypeTableOwnerExists(table structFieldTypeTable, owner string) bool {
	for i := 0; i < len(table); i++ {
		if table[i].owner == owner {
			return true
		}
	}
	return false
}

func structOwnerTableContains(table structOwnerTable, owner string) bool {
	for i := 0; i < len(table); i++ {
		if table[i] == owner {
			return true
		}
	}
	return false
}

func structOwnerTableSet(table structOwnerTable, owner string) structOwnerTable {
	if owner == "" || structOwnerTableContains(table, owner) {
		return table
	}
	return append(table, owner)
}

func structComparisonOwnerExists(fieldTypes structFieldTypeTable, owners structOwnerTable, owner string) bool {
	return structOwnerTableContains(owners, owner) || structFieldTypeTableOwnerExists(fieldTypes, owner)
}

func structFieldTypeCapacityForLoadFiles(files []load.File, namedTypes localTypeTable) int {
	count := 0
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err == nil {
			count += len(packageStructFieldTypes([]parse.File{parsed}, namedTypes))
		}
		arena.Reset(mark)
	}
	return count
}

func appendStructFieldTypeEntries(fields structFieldTypeTable, owner string, toks []scan.Token, open int, close int, namedTypes localTypeTable) structFieldTypeTable {
	specStart := open + 1
	for i := open + 1; i <= close; i++ {
		if i == close || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			fields = appendStructFieldTypesInSpec(fields, owner, toks, specStart, i, namedTypes)
			specStart = i
			if i < close && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return fields
}

func appendArrayStructFieldLowerInfoEntries(fields arrayStructFieldLowerInfoTable, owner string, toks []scan.Token, open int, close int, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	specStart := open + 1
	for i := open + 1; i <= close; i++ {
		if i == close || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
			fields = appendArrayStructFieldLowerInfosInSpec(fields, owner, toks, specStart, i, namedArrays)
			specStart = i
			if i < close && toks[i].Text == ";" {
				specStart = i + 1
			}
		}
	}
	return fields
}

func appendArrayStructFieldLowerInfosInSpec(fields arrayStructFieldLowerInfoTable, owner string, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) arrayStructFieldLowerInfoTable {
	for start < end && toks[start].Text == ";" {
		start++
	}
	if start >= end || toks[start].Text == "*" || toks[start].Kind != scan.Ident {
		return fields
	}
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
		if isTypeStart(toks[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 || typeStart < 0 {
		return fields
	}
	typeEnd := end
	if typeEnd > typeStart && toks[typeEnd-1].Kind == scan.String {
		typeEnd--
	}
	info, ok := fixedArrayTypeLowerInfoForTokens(toks, typeStart, typeEnd)
	if !ok {
		info, ok = namedArrayStructFieldLowerInfoForTokens(toks, typeStart, typeEnd, namedArrays)
	}
	if !ok {
		return fields
	}
	for i := 0; i < len(names); i++ {
		fields = arrayStructFieldLowerInfoTableSet(fields, owner, names[i], info)
	}
	return fields
}

func namedArrayStructFieldLowerInfoForTokens(toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+1 != end || toks[start].Kind != scan.Ident {
		return arrayTypeLowerInfo{}, false
	}
	return namedArrayInfoByName(namedArrays, toks[start].Text)
}

func fixedArrayTypeLowerInfoForTokens(toks []scan.Token, start int, end int) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return arrayTypeLowerInfo{}, false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return arrayTypeLowerInfo{}, false
	}
	length, inferred, ok := arrayLengthForTokens(toks, start+1, brackClose)
	if !ok || inferred || length < 0 {
		return arrayTypeLowerInfo{}, false
	}
	elemStart := brackClose + 1
	if elemStart >= end {
		return arrayTypeLowerInfo{}, false
	}
	elem := tokenTextWithSpaces(toks, elemStart, end)
	if elem == "" {
		return arrayTypeLowerInfo{}, false
	}
	return arrayTypeLowerInfo{elem: elem, length: length}, true
}

func appendStructFieldTypesInSpec(fields structFieldTypeTable, owner string, toks []scan.Token, start int, end int, namedTypes localTypeTable) structFieldTypeTable {
	for start < end && toks[start].Text == ";" {
		start++
	}
	if field, info, ok := embeddedStructFieldTypeInfo(toks, start, end); ok {
		return structFieldTypeTableSet(fields, owner, field, info)
	}
	if start >= end || toks[start].Text == "*" || toks[start].Kind != scan.Ident {
		return fields
	}
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
		if isTypeStart(toks[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 || typeStart < 0 {
		return fields
	}
	typeEnd := end
	if typeEnd > typeStart && toks[typeEnd-1].Kind == scan.String {
		typeEnd--
	}
	typ, arrayOK := arrayStructFieldTypeInfoForLower(toks, typeStart, typeEnd)
	sliceOK := false
	if !arrayOK {
		typ, sliceOK = sliceStructFieldTypeInfoForLower(toks, typeStart, typeEnd)
	}
	if !arrayOK && !sliceOK {
		typ = typeInfoInRange(toks, typeStart, typeEnd)
	}
	if typ.name == "" {
		return fields
	}
	if !arrayOK && !sliceOK {
		typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
	}
	for i := 0; i < len(names); i++ {
		fields = structFieldTypeTableSet(fields, owner, names[i], typ)
	}
	return fields
}

func arrayStructFieldTypeInfoForLower(toks []scan.Token, start int, end int) (localTypeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return localTypeInfo{}, false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return localTypeInfo{}, false
	}
	length, inferred, ok := arrayLengthForTokens(toks, start+1, brackClose)
	if !ok || inferred || length < 0 {
		return localTypeInfo{}, false
	}
	elemStart := brackClose + 1
	if elemStart >= end || toks[elemStart].Text == "[" {
		return localTypeInfo{}, false
	}
	elem := tokenTextWithSpaces(toks, elemStart, end)
	if elem == "" {
		return localTypeInfo{}, false
	}
	return localTypeInfo{name: "[]" + elem}, true
}

func sliceStructFieldTypeInfoForLower(toks []scan.Token, start int, end int) (localTypeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+3 > end || toks[start].Text != "[" || toks[start+1].Text != "]" {
		return localTypeInfo{}, false
	}
	elem := tokenTextWithSpaces(toks, start+2, end)
	if elem == "" || strings.HasPrefix(elem, "[") {
		return localTypeInfo{}, false
	}
	return localTypeInfo{name: "[]" + elem}, true
}

func embeddedStructFieldTypeInfo(toks []scan.Token, start int, end int) (string, localTypeInfo, bool) {
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Kind == scan.String {
		end--
	}
	if start >= end {
		return "", localTypeInfo{}, false
	}
	typeStart := start
	if toks[start].Text == "*" {
		typeStart = start + 1
	}
	if typeStart >= end || toks[typeStart].Kind != scan.Ident {
		return "", localTypeInfo{}, false
	}
	field := ""
	if typeStart+1 == end {
		field = toks[typeStart].Text
	} else if typeStart+3 == end && toks[typeStart+1].Text == "." && toks[typeStart+2].Kind == scan.Ident {
		field = toks[typeStart+2].Text
	} else {
		return "", localTypeInfo{}, false
	}
	info := typeInfoInRange(toks, start, end)
	if info.name == "" {
		return "", localTypeInfo{}, false
	}
	info.pointer = typeRangeIsPointer(toks, start, end)
	info.embedded = true
	return field, info, true
}

func structFieldTypeTableSet(table structFieldTypeTable, owner string, field string, info localTypeInfo) structFieldTypeTable {
	for i := 0; i < len(table); i++ {
		if table[i].owner == owner && table[i].field == field {
			table[i].info = info
			return table
		}
	}
	return append(table, structFieldTypeEntry{owner: owner, field: field, info: info})
}

func structFieldTypeTableLookup(table structFieldTypeTable, owner string, field string) localTypeInfo {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.owner == owner && entry.field == field {
			return entry.info
		}
	}
	return localTypeInfo{}
}

func arrayStructFieldLowerInfoTableSet(table arrayStructFieldLowerInfoTable, owner string, field string, info arrayTypeLowerInfo) arrayStructFieldLowerInfoTable {
	for i := 0; i < len(table); i++ {
		if table[i].owner == owner && table[i].field == field {
			table[i].info = info
			return table
		}
	}
	return append(table, arrayStructFieldLowerInfo{owner: owner, field: field, info: info})
}

func arrayStructFieldLowerInfoTableLookup(table arrayStructFieldLowerInfoTable, owner string, field string) (arrayTypeLowerInfo, bool) {
	for i := 0; i < len(table); i++ {
		entry := table[i]
		if entry.owner == owner && entry.field == field {
			return entry.info, true
		}
	}
	return arrayTypeLowerInfo{}, false
}

func implicitCompositeElementType(toks []scan.Token, open int) string {
	return implicitCompositeElementTypeWithNamedSlices(toks, open, nil)
}

func implicitCompositeElementTypeWithNamedSlices(toks []scan.Token, open int, namedSlices []namedSliceInfo) string {
	parent := containingBraceOpen(toks, open)
	if parent < 0 || !implicitCompositeElementStart(toks, parent, open) {
		return ""
	}
	parentType := compositeLiteralTypeForOpen(toks, parent)
	if !strings.HasPrefix(parentType, "[]") {
		for i := 0; i < len(namedSlices); i++ {
			if namedSlices[i].name == parentType {
				return namedSlices[i].elem
			}
		}
		return ""
	}
	return parentType[2:]
}

func containingBraceOpen(toks []scan.Token, pos int) int {
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

func implicitCompositeElementStart(toks []scan.Token, parent int, open int) bool {
	if open == parent+1 {
		return true
	}
	return open > parent+1 && toks[open-1].Text == ","
}

func compositeLiteralTypeForOpen(toks []scan.Token, open int) string {
	typ := explicitCompositeLiteralTypeBeforeOpen(toks, open)
	if typ != "" {
		return typ
	}
	return implicitCompositeElementType(toks, open)
}

func explicitCompositeLiteralTypeBeforeOpen(toks []scan.Token, open int) string {
	if open <= 0 {
		return ""
	}
	end := open
	start := open - 1
	if toks[start].Kind != scan.Ident {
		return ""
	}
	if start >= 2 && toks[start-1].Text == "." && toks[start-2].Kind == scan.Ident {
		start -= 2
	}
	if start >= 1 && toks[start-1].Text == "*" {
		start--
	}
	for start >= 2 && toks[start-2].Text == "[" && toks[start-1].Text == "]" {
		start -= 2
	}
	var out []byte
	for i := start; i < end; i++ {
		out = appendString(out, toks[i].Text)
	}
	return string(out)
}

func firstCallArgumentEnd(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "," {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return end
}

func callArgumentRange(toks []scan.Token, start int, end int, want int) (int, int, bool) {
	argStart := start
	argIndex := 0
	paren := 0
	brack := 0
	brace := 0
	for i := start; i <= end; i++ {
		if i == end || (paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ",") {
			if argIndex == want && argStart < i {
				return argStart, i, true
			}
			argIndex++
			argStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return 0, 0, false
}

func isFmtPrintName(name string) bool {
	return name == "Fprint" || name == "Fprintln" || name == "Fprintf"
}

func bytesToStringCallArgumentRange(toks []scan.Token, start int, end int) (int, int, bool) {
	if start+3 >= end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return 0, 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return 0, 0, false
	}
	return callArgumentRange(toks, start+2, close, 0)
}

func callArgumentIsStdout(toks []scan.Token, start int, end int) bool {
	return start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Text == "Stdout"
}

func appendUnitSymbolRef(refs *[]unit.Symbol, sym unit.Symbol) {
	values := *refs
	values = append(values, sym)
	*refs = values
}

func removeUnusedNamedMapRefs(refs *[]unit.Symbol, namedMaps []namedMapInfo, body string) {
	if refs == nil || len(*refs) == 0 || len(namedMaps) == 0 {
		return
	}
	values := *refs
	out := values[:0]
	for i := 0; i < len(values); i++ {
		ref := values[i]
		if _, ok := namedMapInfoByName(namedMaps, ref.UnitName); ok && !strings.Contains(body, ref.UnitName) {
			continue
		}
		out = append(out, ref)
	}
	*refs = out
}

func persistUnitSymbol(sym unit.Symbol) unit.Symbol {
	return unit.Symbol{
		ImportPath: arena.PersistString(sym.ImportPath),
		Name:       arena.PersistString(sym.Name),
		UnitName:   arena.PersistString(sym.UnitName),
	}
}

type namedResultValue struct {
	name string
	typ  string
}

type namedResultRewrite struct {
	ok           bool
	resultOpen   int
	resultClose  int
	bodyOpen     int
	signature    string
	declarations string
	returnExpr   string
}

func namedResultRewriteForDecl(file *parse.File, decl *parse.Decl, localTypeDecls []localTypeDeclInfo) namedResultRewrite {
	if decl.Kind != "func" {
		return namedResultRewrite{}
	}
	toks := file.Tokens
	name := tokenIndexAt(toks, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
		return namedResultRewrite{}
	}
	paramsOpen := name + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose+1 >= len(toks) || toks[paramsClose+1].Text != "(" {
		return namedResultRewrite{}
	}
	resultOpen := paramsClose + 1
	resultClose := findClose(toks, resultOpen, "(", ")")
	if resultClose < 0 || int(toks[resultClose].Start) >= decl.End {
		return namedResultRewrite{}
	}
	bodyOpen := findTokenText(toks, resultClose+1, decl.End, "{")
	if bodyOpen < 0 {
		return namedResultRewrite{}
	}
	results := namedResultValuesInList(file.Source, toks, file.Path, resultOpen+1, resultClose, localTypeDecls)
	if len(results) == 0 {
		return namedResultRewrite{}
	}
	indent := lineLeadingIndent(file.Source, int(toks[bodyOpen].Start)) + "\t"
	return namedResultRewrite{
		ok:           true,
		resultOpen:   resultOpen,
		resultClose:  resultClose,
		bodyOpen:     bodyOpen,
		signature:    namedResultSignature(results),
		declarations: namedResultDeclarations(results, indent),
		returnExpr:   namedResultReturnExpr(results),
	}
}

func namedResultValuesInList(source []byte, toks []scan.Token, path string, start int, end int, localTypeDecls []localTypeDeclInfo) []namedResultValue {
	var results []namedResultValue
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == "," {
			name, typeStart, typeEnd, hasType := namedResultValueSegment(toks, segmentStart, i)
			if name != "" {
				if hasType {
					pending = append(pending, name)
					typ := strings.TrimSpace(string(source[int(toks[typeStart].Start):int(toks[typeEnd-1].End)]))
					if replacement := anonymousStructGeneratedNameForRange(toks, path, typeStart, typeEnd, localTypeDecls); replacement != "" {
						typ = replacement
					}
					for j := 0; j < len(pending); j++ {
						results = append(results, namedResultValue{name: pending[j], typ: typ})
					}
					pending = nil
				} else {
					pending = append(pending, name)
				}
			}
			segmentStart = i + 1
		}
	}
	return results
}

func namedResultValueSegment(toks []scan.Token, start int, end int) (string, int, int, bool) {
	for start < end && toks[start].Text == "," {
		start++
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return "", 0, 0, false
	}
	if start+1 < end && isTypeStart(toks[start+1]) {
		return toks[start].Text, start + 1, end, true
	}
	return toks[start].Text, 0, 0, false
}

func namedResultSignature(results []namedResultValue) string {
	if len(results) == 1 {
		return results[0].typ
	}
	var out []byte
	out = append(out, '(')
	for i := 0; i < len(results); i++ {
		if i > 0 {
			out = appendString(out, ", ")
		}
		out = appendString(out, results[i].typ)
	}
	out = append(out, ')')
	return string(out)
}

func namedResultDeclarations(results []namedResultValue, indent string) string {
	var out []byte
	for i := 0; i < len(results); i++ {
		out = append(out, '\n')
		out = appendString(out, indent)
		out = appendString(out, "var ")
		out = appendString(out, results[i].name)
		out = append(out, ' ')
		out = appendString(out, results[i].typ)
	}
	return string(out)
}

func namedResultReturnExpr(results []namedResultValue) string {
	var out []byte
	for i := 0; i < len(results); i++ {
		if i > 0 {
			out = appendString(out, ", ")
		}
		out = appendString(out, results[i].name)
	}
	return string(out)
}

func bareReturnAt(toks []scan.Token, pos int, declEnd int) bool {
	if pos+1 >= len(toks) || int(toks[pos+1].Start) >= declEnd {
		return true
	}
	next := toks[pos+1]
	return next.Kind == scan.EOF || next.Text == ";" || next.Text == "}" || next.Line != toks[pos].Line
}

func appendLoweredPrintlnCall(out []byte, source []byte, toks []scan.Token, pos int, declEnd int) ([]byte, int, bool) {
	if pos+1 >= len(toks) || toks[pos].Text != "println" || toks[pos+1].Text != "(" {
		return out, 0, false
	}
	close := findClose(toks, pos+1, "(", ")")
	if close < 0 || int(toks[close].Start) >= declEnd {
		return out, 0, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	indent := lineLeadingIndent(source, int(toks[pos].Start))
	wrote := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg.start >= arg.end {
			continue
		}
		if wrote {
			out = append(out, '\n')
			out = appendString(out, indent)
		}
		out = appendString(out, "print(")
		out = appendString(out, localConstExpressionText(source, toks, arg.start, arg.end, 0))
		out = append(out, ')')
		wrote = true
		if i+1 < len(args) {
			out = append(out, '\n')
			out = appendString(out, indent)
			out = appendString(out, "print(\" \")")
		}
	}
	if wrote {
		out = append(out, '\n')
		out = appendString(out, indent)
	}
	out = appendString(out, "print(\"\\n\")")
	return out, int(toks[close].End), true
}

func appendLoweredLocalConstDecl(out []byte, source []byte, toks []scan.Token, pos int, declEnd int) ([]byte, int, bool) {
	if pos+1 >= len(toks) || toks[pos].Text != "const" {
		return out, 0, false
	}
	if toks[pos+1].Text == "(" {
		return appendLoweredLocalConstGroup(out, source, toks, pos, declEnd)
	}
	end := localConstSingleEnd(toks, pos, declEnd)
	if end <= pos+1 {
		return out, 0, false
	}
	eq := findTopLevelToken(toks, pos+2, end, "=")
	if eq < 0 || eq+1 >= end {
		return out, 0, false
	}
	lhs := localConstLHSNames(toks, pos+1, eq)
	if lhs == "" {
		return out, 0, false
	}
	rhs := localConstExpressionText(source, toks, eq+1, end, 0)
	if rhs == "" {
		return out, 0, false
	}
	out = appendString(out, lhs)
	out = appendString(out, " := ")
	out = appendString(out, rhs)
	return out, int(toks[end-1].End), true
}

func appendLoweredLocalInferredVarDecl(out []byte, source []byte, toks []scan.Token, pos int, declEnd int) ([]byte, int, bool) {
	if pos+3 >= len(toks) || toks[pos].Text != "var" || toks[pos+1].Kind != scan.Ident || toks[pos+2].Text != "=" {
		return out, 0, false
	}
	initStart := pos + 3
	out = appendString(out, toks[pos+1].Text)
	out = appendString(out, " := ")
	return out, int(toks[initStart].Start), true
}

func lowerLocalVarDeclarations(body string) string {
	if !strings.Contains(body, "var") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			lines := lowerLocalVarGroupLines(body, toks, i+2, close)
			if len(lines) == 0 {
				continue
			}
			text := joinIndentedLines(lines, statementIndent(body, int(toks[i].Start)))
			replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[close].End), text: text})
			i = close
			continue
		}
		end := localConstSingleEnd(toks, i, len(body))
		if end <= i+1 {
			continue
		}
		lines, changed := lowerLocalVarSpecLines(body, toks, expressionRange{start: i + 1, end: end})
		if !changed || len(lines) == 0 {
			continue
		}
		text := joinIndentedLines(lines, statementIndent(body, int(toks[i].Start)))
		replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[end-1].End), text: text})
		i = end - 1
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func lowerNamedArrayTypeUses(body string, namedArrays []namedArrayInfo) string {
	if len(namedArrays) == 0 || !strings.Contains(body, "{") && !strings.Contains(body, "var") && !strings.Contains(body, "func") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text == "var" {
			replacements = appendNamedArrayVarDeclarationReplacements(replacements, toks, i, len(toks), namedArrays)
			continue
		}
		if toks[i].Kind != scan.Ident {
			continue
		}
		info, ok := namedArrayInfoByName(namedArrays, toks[i].Text)
		if !ok {
			continue
		}
		if repl, ok := namedArrayFunctionSignatureTypeReplacement(toks, i, info); ok {
			replacements = append(replacements, repl)
			continue
		}
		if i+1 >= len(toks) || toks[i+1].Text != "{" {
			continue
		}
		replacements = append(replacements, expressionReplacement{
			start: int(toks[i].Start),
			end:   int(toks[i].End),
			text:  arrayTypeLowerInfoFixedName(info),
		})
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func namedArrayFunctionSignatureTypeReplacement(toks []scan.Token, pos int, info arrayTypeLowerInfo) (expressionReplacement, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident || info.elem == "" {
		return expressionReplacement{}, false
	}
	open := containingOpenForLower(toks, pos, "(", ")")
	if functionParameterListOpenForLower(toks, open) && arrayTypeStartsFunctionParameterTypeForLower(toks, open, pos) {
		return namedArrayTypeReplacementForToken(toks, pos, info), true
	}
	if namedArrayTypeStartsDirectFunctionResultForLower(toks, pos) {
		return namedArrayTypeReplacementForToken(toks, pos, info), true
	}
	if functionResultListOpenForLower(toks, open) && arrayTypeStartsFunctionParameterTypeForLower(toks, open, pos) {
		return namedArrayTypeReplacementForToken(toks, pos, info), true
	}
	return expressionReplacement{}, false
}

func namedArrayTypeStartsDirectFunctionResultForLower(toks []scan.Token, pos int) bool {
	if pos <= 0 || toks[pos-1].Text != ")" {
		return false
	}
	paramsOpen := findOpen(toks, pos-1, "(", ")")
	return functionParameterListOpenForLower(toks, paramsOpen)
}

func namedArrayTypeReplacementForToken(toks []scan.Token, pos int, info arrayTypeLowerInfo) expressionReplacement {
	return expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[pos].End),
		text:  arrayTypeLowerInfoFixedName(info),
	}
}

func appendNamedArrayVarDeclarationReplacements(replacements []expressionReplacement, toks []scan.Token, pos int, limit int, namedArrays []namedArrayInfo) []expressionReplacement {
	if pos+1 < limit && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close >= limit {
			return replacements
		}
		specs := localConstSpecRanges(toks, pos+2, close)
		for i := 0; i < len(specs); i++ {
			replacements = appendNamedArrayVarSpecTypeReplacement(replacements, toks, specs[i].start, specs[i].end, namedArrays)
		}
		return replacements
	}
	end := lowerSimpleStatementEnd(toks, pos+1, limit)
	if end <= pos+1 {
		return replacements
	}
	return appendNamedArrayVarSpecTypeReplacement(replacements, toks, pos+1, end, namedArrays)
}

func appendNamedArrayVarSpecTypeReplacement(replacements []expressionReplacement, toks []scan.Token, start int, end int, namedArrays []namedArrayInfo) []expressionReplacement {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return replacements
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	_, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if typeStart < 0 {
		return replacements
	}
	typeStart, prefixEnd = trimTokenRange(toks, typeStart, prefixEnd)
	if typeStart+1 != prefixEnd || toks[typeStart].Kind != scan.Ident {
		return replacements
	}
	info, ok := namedArrayInfoByName(namedArrays, toks[typeStart].Text)
	if !ok {
		return replacements
	}
	return append(replacements, expressionReplacement{
		start: int(toks[typeStart].Start),
		end:   int(toks[typeStart].End),
		text:  arrayTypeLowerInfoFixedName(info),
	})
}

func lowerPackageArrayVarDeclarations(body string) string {
	if !strings.Contains(body, "var") || !strings.Contains(body, "[") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "var" {
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close < 0 {
				continue
			}
			lines := lowerPackageArrayVarGroupLines(body, toks, i+2, close)
			if len(lines) == 0 {
				continue
			}
			replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[close].End), text: strings.Join(lines, "\n")})
			i = close
			continue
		}
		end := lowerSimpleStatementEnd(toks, i+1, len(toks))
		if end <= i+1 {
			continue
		}
		lines, changed := lowerPackageArrayVarSpecLines(body, toks, expressionRange{start: i + 1, end: end})
		if !changed || len(lines) == 0 {
			continue
		}
		replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[end-1].End), text: strings.Join(lines, "\n")})
		i = end - 1
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func lowerPackageArrayVarGroupLines(body string, toks []scan.Token, start int, end int) []string {
	specs := localConstSpecRanges(toks, start, end)
	var lines []string
	for i := 0; i < len(specs); i++ {
		specLines, changed := lowerPackageArrayVarSpecLines(body, toks, specs[i])
		if !changed {
			continue
		}
		for j := 0; j < len(specLines); j++ {
			lines = append(lines, specLines[j])
		}
	}
	return lines
}

func lowerPackageArrayVarSpecLines(body string, toks []scan.Token, spec expressionRange) ([]string, bool) {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return nil, false
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if len(names) == 0 || typeStart < 0 {
		return nil, false
	}
	arrayInfo, ok := localArrayVarTypeInfo(body, toks, typeStart, prefixEnd)
	if !ok {
		return nil, false
	}
	typ := "[]" + arrayInfo.elem
	if eq < 0 {
		zero := arrayZeroLiteral(arrayInfo.elem, arrayInfo.length)
		lines := make([]string, 0, len(names))
		for i := 0; i < len(names); i++ {
			lines = append(lines, "var "+names[i]+" "+typ+" = "+zero)
		}
		return lines, true
	}
	values := topLevelExpressionRanges(toks, eq+1, end)
	if len(values) != len(names) {
		return nil, false
	}
	lines := make([]string, 0, len(names))
	for i := 0; i < len(names); i++ {
		value := values[i]
		rhs := strings.TrimSpace(body[int(toks[value.start].Start):int(toks[value.end-1].End)])
		lines = append(lines, "var "+names[i]+" "+typ+" = "+rhs)
	}
	return lines, true
}

func lowerLocalVarGroupLines(body string, toks []scan.Token, start int, end int) []string {
	specs := localConstSpecRanges(toks, start, end)
	var lines []string
	for i := 0; i < len(specs); i++ {
		specLines, _ := lowerLocalVarSpecLines(body, toks, specs[i])
		for j := 0; j < len(specLines); j++ {
			lines = append(lines, specLines[j])
		}
	}
	return lines
}

func lowerLocalVarSpecLines(body string, toks []scan.Token, spec expressionRange) ([]string, bool) {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return nil, false
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if len(names) == 0 {
		return nil, false
	}
	if typeStart < 0 {
		if eq < 0 {
			return []string{"var " + strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])}, false
		}
		lhs := joinNames(names)
		rhs := strings.TrimSpace(body[int(toks[eq+1].Start):int(toks[end-1].End)])
		return []string{lhs + " := " + rhs}, true
	}
	var lines []string
	if arrayInfo, ok := localArrayVarTypeInfo(body, toks, typeStart, prefixEnd); ok {
		typ := "[]" + arrayInfo.elem
		if eq < 0 {
			zero := arrayZeroLiteral(arrayInfo.elem, arrayInfo.length)
			for i := 0; i < len(names); i++ {
				lines = append(lines, names[i]+" := "+zero)
			}
			return lines, true
		}
		values := topLevelExpressionRanges(toks, eq+1, end)
		if len(values) != len(names) {
			return []string{"var " + strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])}, false
		}
		for i := 0; i < len(names); i++ {
			value := values[i]
			rhs := strings.TrimSpace(body[int(toks[value.start].Start):int(toks[value.end-1].End)])
			lines = append(lines, "var "+names[i]+" "+typ+" = "+rhs)
		}
		return lines, true
	}
	if len(names) == 1 {
		return []string{"var " + strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])}, false
	}
	typ := strings.TrimSpace(body[int(toks[typeStart].Start):int(toks[prefixEnd-1].End)])
	if eq < 0 {
		for i := 0; i < len(names); i++ {
			lines = append(lines, "var "+names[i]+" "+typ)
		}
		return lines, true
	}
	values := topLevelExpressionRanges(toks, eq+1, end)
	if len(values) != len(names) {
		if len(values) == 1 && expressionContainsCall(toks, values[0].start, values[0].end) {
			rhs := strings.TrimSpace(body[int(toks[values[0].start].Start):int(toks[values[0].end-1].End)])
			for i := 0; i < len(names); i++ {
				lines = append(lines, "var "+names[i]+" "+typ)
			}
			lines = append(lines, joinNames(names)+" = "+rhs)
			return lines, true
		}
		return []string{"var " + strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])}, false
	}
	for i := 0; i < len(names); i++ {
		value := values[i]
		rhs := strings.TrimSpace(body[int(toks[value.start].Start):int(toks[value.end-1].End)])
		lines = append(lines, "var "+names[i]+" "+typ+" = "+rhs)
	}
	return lines, true
}

func localVarSpecNamesAndType(toks []scan.Token, start int, end int) ([]string, int) {
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
		if isTypeStart(toks[i]) {
			typeStart = i
			break
		}
	}
	return names, typeStart
}

func joinNames(names []string) string {
	var out []byte
	for i := 0; i < len(names); i++ {
		if i > 0 {
			out = appendString(out, ", ")
		}
		out = appendString(out, names[i])
	}
	return string(out)
}

func joinIndentedLines(lines []string, indent string) string {
	var out []byte
	for i := 0; i < len(lines); i++ {
		if i > 0 {
			out = append(out, '\n')
			out = appendString(out, indent)
		}
		out = appendString(out, lines[i])
	}
	return string(out)
}

func appendLoweredLocalConstGroup(out []byte, source []byte, toks []scan.Token, pos int, declEnd int) ([]byte, int, bool) {
	open := pos + 1
	close := findClose(toks, open, "(", ")")
	if close < 0 || int(toks[close].Start) >= declEnd {
		return out, 0, false
	}
	specs := localConstSpecRanges(toks, open+1, close)
	if len(specs) == 0 {
		return out, 0, false
	}
	indent := localConstLineIndent(source, int(toks[pos].Start))
	buf := make([]byte, 0, int(toks[close].End-toks[pos].Start))
	prevRhsStart := -1
	prevRhsEnd := -1
	wrote := false
	for i := 0; i < len(specs); i++ {
		spec := specs[i]
		if spec.start >= spec.end {
			continue
		}
		eq := findTopLevelToken(toks, spec.start, spec.end, "=")
		lhsEnd := spec.end
		rhsStart := prevRhsStart
		rhsEnd := prevRhsEnd
		if eq >= 0 {
			lhsEnd = eq
			rhsStart = eq + 1
			rhsEnd = spec.end
			prevRhsStart = rhsStart
			prevRhsEnd = rhsEnd
		}
		if rhsStart < 0 || rhsEnd <= rhsStart {
			return out, 0, false
		}
		lhs := localConstLHSNames(toks, spec.start, lhsEnd)
		rhs := localConstExpressionText(source, toks, rhsStart, rhsEnd, i)
		if lhs == "" || rhs == "" {
			return out, 0, false
		}
		if wrote {
			buf = append(buf, '\n')
			buf = appendString(buf, indent)
		}
		buf = appendString(buf, lhs)
		buf = appendString(buf, " := ")
		buf = appendString(buf, rhs)
		wrote = true
	}
	if !wrote {
		return out, 0, false
	}
	out = appendBytes(out, buf)
	return out, int(toks[close].End), true
}

func localConstSingleEnd(toks []scan.Token, pos int, declEnd int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < len(toks); i++ {
		if int(toks[i].Start) >= declEnd || toks[i].Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == ";" {
				return i
			}
			if i > pos+1 && toks[i].Line > toks[i-1].Line {
				return i
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func localConstSpecRanges(toks []scan.Token, start int, end int) []expressionRange {
	var ranges []expressionRange
	specStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == ";" {
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
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	if specStart < end {
		ranges = append(ranges, expressionRange{start: specStart, end: end})
	}
	return ranges
}

func localConstLHSNames(toks []scan.Token, start int, end int) string {
	var out []byte
	expectName := true
	wrote := false
	for i := start; i < end; i++ {
		tok := toks[i]
		if expectName {
			if tok.Kind != scan.Ident {
				break
			}
			if wrote {
				out = appendString(out, ", ")
			}
			out = appendString(out, tok.Text)
			wrote = true
			expectName = false
			continue
		}
		if tok.Text == "," {
			expectName = true
			continue
		}
		break
	}
	return string(out)
}

func localConstExpressionText(source []byte, toks []scan.Token, start int, end int, iotaValue int) string {
	if start >= end {
		return ""
	}
	exprStart := int(toks[start].Start)
	exprEnd := int(toks[end-1].End)
	buf := make([]byte, 0, exprEnd-exprStart)
	cursor := exprStart
	for i := start; i < end; i++ {
		tok := toks[i]
		if tok.Text != "iota" && !(tok.Kind == scan.String && strings.HasPrefix(tok.Text, "`")) && !(tok.Kind == scan.Number && isOctalNumberText(tok.Text)) {
			continue
		}
		buf = appendBytes(buf, source[cursor:int(tok.Start)])
		if tok.Text == "iota" {
			buf = appendString(buf, strconv.Itoa(iotaValue))
		} else if tok.Kind == scan.String && strings.HasPrefix(tok.Text, "`") {
			buf = appendInterpretedStringLiteral(buf, tok.Text)
		} else {
			buf = appendString(buf, strconv.Itoa(parseOctalNumberText(tok.Text)))
		}
		cursor = int(tok.End)
	}
	buf = appendBytes(buf, source[cursor:exprEnd])
	return strings.TrimSpace(string(buf))
}

func localConstLineIndent(source []byte, pos int) string {
	lineStart := pos
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	return string(source[lineStart:pos])
}

func lineLeadingIndent(source []byte, pos int) string {
	lineStart := pos
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := lineStart
	for lineEnd < pos && (source[lineEnd] == ' ' || source[lineEnd] == '\t') {
		lineEnd++
	}
	return string(source[lineStart:lineEnd])
}

func findTopLevelToken(toks []scan.Token, start int, end int, text string) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == text {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func isCompositeKey(toks []scan.Token, pos int) bool {
	return pos+1 < len(toks) && toks[pos+1].Text == ":"
}

func isStructFieldName(toks []scan.Token, pos int) bool {
	if !startsStructFieldName(toks, pos) {
		return false
	}
	if pos+1 >= len(toks) || !isTypeStartAfterName(toks, pos, len(toks)) {
		return false
	}
	if pos > 0 && toks[pos-1].Text == "type" {
		return false
	}
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "}" {
			depth++
			continue
		}
		if toks[i].Text == "{" {
			if depth == 0 {
				return i > 0 && toks[i-1].Text == "struct"
			}
			depth--
		}
	}
	return false
}

func isStructTagToken(toks []scan.Token, pos int) bool {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.String {
		return false
	}
	depth := 0
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "}" {
			depth++
			continue
		}
		if toks[i].Text == "{" {
			if depth == 0 {
				return i > 0 && toks[i-1].Text == "struct"
			}
			depth--
		}
	}
	return false
}

func startsStructFieldName(toks []scan.Token, pos int) bool {
	if pos <= 0 {
		return false
	}
	prev := toks[pos-1]
	if prev.Text == "{" || prev.Text == ";" || prev.Text == "," {
		return true
	}
	return prev.Line != toks[pos].Line
}

func methodLookupName(receiver localTypeInfo, methodName string) string {
	name := receiver.name + "_" + methodName
	if receiver.qualifier != "" {
		return receiver.qualifier + "." + name
	}
	return name
}

func methodExpressionCallReplacement(source []byte, toks []scan.Token, pos int, localTypes localTypeTable, importLocalNames localNameTable, methods methodTable, refs *[]unit.Symbol) (string, int, int, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return "", 0, 0, false
	}
	if pos+5 < len(toks) && toks[pos+1].Text == "." && toks[pos+2].Kind == scan.Ident && toks[pos+3].Text == "." && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "(" {
		if isLocalNameAt(importLocalNames, toks[pos].Text, int(toks[pos].Start)) {
			return "", 0, 0, false
		}
		methodName := toks[pos].Text + "." + toks[pos+2].Text + "_" + toks[pos+4].Text
		method := methodTableLookup(methods, methodName)
		if method.unitName == "" || method.pointerReceiver {
			return "", 0, 0, false
		}
		if method.importPath != "" {
			appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
		}
		return method.unitName + "(", int(toks[pos+5].End), pos + 5, true
	}
	if pos+3 < len(toks) && toks[pos+1].Text == "." && toks[pos+2].Kind == scan.Ident && toks[pos+3].Text == "(" {
		if localTypeTableLookup(localTypes, toks[pos].Text).name != "" {
			return "", 0, 0, false
		}
		methodName := toks[pos].Text + "_" + toks[pos+2].Text
		method := methodTableLookup(methods, methodName)
		if method.unitName == "" || method.pointerReceiver {
			return "", 0, 0, false
		}
		return method.unitName + "(", int(toks[pos+3].End), pos + 3, true
	}
	_ = source
	return "", 0, 0, false
}

type indexedMethodCall struct {
	receiverStart     int
	receiverEnd       int
	typeStart         int
	literalOpen       int
	literalClose      int
	indexOpen         int
	methodName        int
	methodOpen        int
	callClose         int
	receiverType      localTypeInfo
	namedSliceLiteral bool
}

func indexedReceiverMethodCallReplacement(source []byte, toks []scan.Token, pos int, localTypes localTypeTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, namedSlices []namedSliceInfo, refs *[]unit.Symbol) (string, int, int, bool) {
	call, ok := indexedReceiverMethodCallAt(source, toks, pos, localTypes, topNames, importRefs, namedSlices, refs)
	if !ok {
		return "", 0, 0, false
	}
	methodName := methodLookupName(call.receiverType, toks[call.methodName].Text)
	method := methodTableLookup(methods, methodName)
	if method.unitName == "" {
		return "", 0, 0, false
	}
	if method.importPath != "" {
		appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
	}
	receiverArg := string(source[int(toks[call.receiverStart].Start):int(toks[call.receiverEnd-1].End)])
	if toks[call.receiverStart].Text == "[" {
		arg, argOK := loweredIndexedSliceLiteralReceiverArg(source, toks, call, topNames, importRefs, refs)
		if !argOK {
			return "", 0, 0, false
		}
		receiverArg = arg
	} else if call.namedSliceLiteral {
		arg, argOK := loweredIndexedNamedSliceLiteralReceiverArg(source, toks, call, topNames, importRefs, refs)
		if !argOK {
			return "", 0, 0, false
		}
		receiverArg = arg
	}
	if method.pointerReceiver && !call.receiverType.pointer {
		receiverArg = "&" + receiverArg
	} else if !method.pointerReceiver && call.receiverType.pointer {
		receiverArg = "*" + receiverArg
	}
	replacement := method.unitName + "(" + receiverArg
	if call.callClose < 0 || int(toks[call.methodOpen].End) < int(toks[call.callClose].Start) {
		replacement = replacement + ", "
	}
	return replacement, int(toks[call.methodOpen].End), call.methodOpen, true
}

func indexedReceiverMethodCallAt(source []byte, toks []scan.Token, pos int, localTypes localTypeTable, topNames symbolNameTable, importRefs importSymbolTable, namedSlices []namedSliceInfo, refs *[]unit.Symbol) (indexedMethodCall, bool) {
	if pos < 0 || pos >= len(toks) {
		return indexedMethodCall{}, false
	}
	if toks[pos].Kind == scan.Ident && pos+1 < len(toks) && toks[pos+1].Text == "[" {
		closeIndex := findClose(toks, pos+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= len(toks) || toks[closeIndex+1].Text != "." || toks[closeIndex+2].Kind != scan.Ident || toks[closeIndex+3].Text != "(" {
			return indexedMethodCall{}, false
		}
		receiverType := localTypeTableLookup(localTypes, toks[pos].Text)
		if receiverType.name == "" {
			return indexedMethodCall{}, false
		}
		return indexedMethodCall{
			receiverStart: pos,
			receiverEnd:   closeIndex + 1,
			methodName:    closeIndex + 2,
			methodOpen:    closeIndex + 3,
			callClose:     findClose(toks, closeIndex+3, "(", ")"),
			receiverType:  receiverType,
		}, true
	}
	if toks[pos].Text == "(" {
		closeParen := findClose(toks, pos, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || toks[closeParen+1].Text != "[" {
			return indexedMethodCall{}, false
		}
		closeIndex := findClose(toks, closeParen+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= len(toks) || toks[closeIndex+1].Text != "." || toks[closeIndex+2].Kind != scan.Ident || toks[closeIndex+3].Text != "(" {
			return indexedMethodCall{}, false
		}
		innerStart := pos + 1
		open := compositeLiteralOpenForTypeStart(toks, innerStart)
		if open < 0 {
			return indexedMethodCall{}, false
		}
		closeBrace := findClose(toks, open, "{", "}")
		if closeBrace < 0 || closeBrace+1 != closeParen {
			return indexedMethodCall{}, false
		}
		receiverType, receiverOK := namedSliceLiteralElementTypeInfo(toks, innerStart, open, namedSlices)
		if !receiverOK {
			return indexedMethodCall{}, false
		}
		return indexedMethodCall{
			receiverStart:     pos,
			receiverEnd:       closeIndex + 1,
			typeStart:         innerStart,
			literalOpen:       open,
			literalClose:      closeBrace,
			indexOpen:         closeParen + 1,
			methodName:        closeIndex + 2,
			methodOpen:        closeIndex + 3,
			callClose:         findClose(toks, closeIndex+3, "(", ")"),
			receiverType:      receiverType,
			namedSliceLiteral: true,
		}, true
	}
	if open := compositeLiteralOpenForTypeStart(toks, pos); open >= 0 {
		closeBrace := findClose(toks, open, "{", "}")
		if closeBrace < 0 || closeBrace+1 >= len(toks) || toks[closeBrace+1].Text != "[" {
			return indexedMethodCall{}, false
		}
		closeIndex := findClose(toks, closeBrace+1, "[", "]")
		if closeIndex < 0 || closeIndex+3 >= len(toks) || toks[closeIndex+1].Text != "." || toks[closeIndex+2].Kind != scan.Ident || toks[closeIndex+3].Text != "(" {
			return indexedMethodCall{}, false
		}
		receiverType, receiverOK := namedSliceLiteralElementTypeInfo(toks, pos, open, namedSlices)
		if !receiverOK {
			return indexedMethodCall{}, false
		}
		return indexedMethodCall{
			receiverStart:     pos,
			receiverEnd:       closeIndex + 1,
			typeStart:         pos,
			literalOpen:       open,
			literalClose:      closeBrace,
			indexOpen:         closeBrace + 1,
			methodName:        closeIndex + 2,
			methodOpen:        closeIndex + 3,
			callClose:         findClose(toks, closeIndex+3, "(", ")"),
			receiverType:      receiverType,
			namedSliceLiteral: true,
		}, true
	}
	if toks[pos].Text != "[" || pos+1 >= len(toks) || toks[pos+1].Text != "]" {
		return indexedMethodCall{}, false
	}
	typeStart, open, pointer, ok := sliceLiteralElementTypeRange(toks, pos)
	if !ok {
		return indexedMethodCall{}, false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace+1 >= len(toks) || toks[closeBrace+1].Text != "[" {
		return indexedMethodCall{}, false
	}
	closeIndex := findClose(toks, closeBrace+1, "[", "]")
	if closeIndex < 0 || closeIndex+3 >= len(toks) || toks[closeIndex+1].Text != "." || toks[closeIndex+2].Kind != scan.Ident || toks[closeIndex+3].Text != "(" {
		return indexedMethodCall{}, false
	}
	receiverType := typeInfoInRange(toks, typeStart, open)
	receiverType.pointer = pointer || typeRangeIsPointer(toks, typeStart, open)
	if receiverType.name == "" {
		return indexedMethodCall{}, false
	}
	return indexedMethodCall{
		receiverStart: pos,
		receiverEnd:   closeIndex + 1,
		methodName:    closeIndex + 2,
		methodOpen:    closeIndex + 3,
		callClose:     findClose(toks, closeIndex+3, "(", ")"),
		receiverType:  receiverType,
	}, true
}

func loweredIndexedSliceLiteralReceiverArg(source []byte, toks []scan.Token, call indexedMethodCall, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	if toks[call.receiverStart].Text != "[" || call.receiverStart+1 >= len(toks) || toks[call.receiverStart+1].Text != "]" {
		return "", false
	}
	typeStart, open, pointer, ok := sliceLiteralElementTypeRange(toks, call.receiverStart)
	if !ok {
		return "", false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace+1 >= call.receiverEnd {
		return "", false
	}
	indexOpen := closeBrace + 1
	if toks[indexOpen].Text != "[" {
		return "", false
	}
	indexClose := findClose(toks, indexOpen, "[", "]")
	if indexClose < 0 || indexClose+1 != call.receiverEnd {
		return "", false
	}
	elemType, ok := loweredCompositeReceiverType(toks, typeStart, open, topNames, importRefs, refs)
	if !ok {
		return "", false
	}
	literal := string(source[int(toks[open].Start):int(toks[closeBrace].End)])
	if pointer {
		literal = loweredPointerSliceLiteralBody(source, toks, open, closeBrace, typeStart, open, elemType)
	}
	index := string(source[int(toks[indexOpen].Start):int(toks[indexClose].End)])
	prefix := "[]"
	if pointer {
		prefix = "[]*"
	}
	return prefix + elemType + literal + index, true
}

func loweredIndexedNamedSliceLiteralReceiverArg(source []byte, toks []scan.Token, call indexedMethodCall, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	if !call.namedSliceLiteral || call.literalOpen <= call.typeStart || call.literalClose < call.literalOpen {
		return "", false
	}
	typeText, ok := loweredCompositeReceiverType(toks, call.typeStart, call.literalOpen, topNames, importRefs, refs)
	if !ok {
		return "", false
	}
	indexOpen := call.indexOpen
	if indexOpen <= 0 {
		indexOpen = call.literalClose + 1
	}
	if indexOpen >= call.receiverEnd || toks[indexOpen].Text != "[" {
		return "", false
	}
	indexClose := findClose(toks, indexOpen, "[", "]")
	if indexClose < 0 || indexClose+1 != call.receiverEnd {
		return "", false
	}
	literal := string(source[int(toks[call.literalOpen].Start):int(toks[call.literalClose].End)])
	index := string(source[int(toks[indexOpen].Start):int(toks[indexClose].End)])
	return typeText + literal + index, true
}

func namedSliceLiteralElementTypeInfo(toks []scan.Token, typeStart int, open int, namedSlices []namedSliceInfo) (localTypeInfo, bool) {
	typeText := explicitCompositeLiteralTypeBeforeOpen(toks, open)
	if typeText == "" {
		return localTypeInfo{}, false
	}
	for i := 0; i < len(namedSlices); i++ {
		if namedSlices[i].name == typeText && namedSlices[i].elemInfo.name != "" {
			return namedSlices[i].elemInfo, true
		}
	}
	return localTypeInfo{}, false
}

func loweredPointerSliceLiteralBody(source []byte, toks []scan.Token, open int, close int, typeStart int, typeEnd int, elemType string) string {
	typeCount := typeEnd - typeStart
	src := string(source)
	if typeCount <= 0 {
		return src[int(toks[open].Start):int(toks[close].End)]
	}
	var out []byte
	cursor := int(toks[open].Start)
	for i := open; i <= close; i++ {
		if toks[i].Text != "&" {
			continue
		}
		literalTypeStart := i + 1
		literalTypeEnd := literalTypeStart + typeCount
		if literalTypeEnd >= len(toks) || literalTypeEnd > close || toks[literalTypeEnd].Text != "{" {
			continue
		}
		matched := true
		for j := 0; j < typeCount; j++ {
			if toks[literalTypeStart+j].Text != toks[typeStart+j].Text {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		out = appendStringRange(out, src, cursor, int(toks[literalTypeStart].Start))
		out = appendString(out, elemType)
		cursor = int(toks[literalTypeEnd-1].End)
		i = literalTypeEnd - 1
	}
	out = appendStringRange(out, src, cursor, int(toks[close].End))
	return string(out)
}

type compositeMethodCall struct {
	receiverStart int
	receiverEnd   int
	typeStart     int
	typeEnd       int
	literalOpen   int
	literalClose  int
	methodName    int
	methodOpen    int
	callClose     int
	pointer       bool
}

func compositeLiteralMethodCallReplacement(source []byte, toks []scan.Token, pos int, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, fieldTypes structFieldTypeTable, refs *[]unit.Symbol) (string, int, int, bool) {
	call, ok := compositeLiteralMethodCallAt(toks, pos)
	if !ok {
		return "", 0, 0, false
	}
	receiverType := typeInfoInRange(toks, call.typeStart, call.typeEnd)
	if receiverType.name == "" {
		return "", 0, 0, false
	}
	receiverType.pointer = call.pointer
	methodName := methodLookupName(receiverType, toks[call.methodName].Text)
	method := methodTableLookup(methods, methodName)
	if method.unitName == "" {
		path, promotedReceiver, promotedMethod, promotedOK := promotedStructMethodPath(fieldTypes, localTypeInfoOwnerName(receiverType), toks[call.methodName].Text, methods)
		if !promotedOK {
			return "", 0, 0, false
		}
		if promotedMethod.pointerReceiver && !promotedReceiver.pointer && !call.pointer {
			return "", 0, 0, false
		}
		method = promotedMethod
		if method.importPath != "" {
			appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
		}
		receiverArg, ok := compositePromotedMethodReceiverArg(source, toks, call, path, promotedReceiver, method, topNames, importRefs, refs)
		if !ok {
			return "", 0, 0, false
		}
		replacement := method.unitName + "(" + receiverArg
		if call.callClose < 0 || int(toks[call.methodOpen].End) < int(toks[call.callClose].Start) {
			replacement = replacement + ", "
		}
		return replacement, int(toks[call.methodOpen].End), call.methodOpen, true
	}
	if method.pointerReceiver && !receiverType.pointer {
		return "", 0, 0, false
	}
	if method.importPath != "" {
		appendUnitSymbolRef(refs, unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName})
	}
	receiverArg, ok := compositeMethodReceiverArg(source, toks, call, method, topNames, importRefs, refs)
	if !ok {
		return "", 0, 0, false
	}
	replacement := method.unitName + "(" + receiverArg
	if call.callClose < 0 || int(toks[call.methodOpen].End) < int(toks[call.callClose].Start) {
		replacement = replacement + ", "
	}
	return replacement, int(toks[call.methodOpen].End), call.methodOpen, true
}

func compositeLiteralMethodCallAt(toks []scan.Token, pos int) (compositeMethodCall, bool) {
	if pos < 0 || pos >= len(toks) {
		return compositeMethodCall{}, false
	}
	if toks[pos].Text == "(" {
		return parenthesizedCompositeLiteralMethodCallAt(toks, pos)
	}
	return bareCompositeLiteralMethodCallAt(toks, pos)
}

func parenthesizedCompositeLiteralMethodCallAt(toks []scan.Token, pos int) (compositeMethodCall, bool) {
	closeParen := findClose(toks, pos, "(", ")")
	if closeParen <= pos || closeParen+3 >= len(toks) || toks[closeParen+1].Text != "." || toks[closeParen+2].Kind != scan.Ident || toks[closeParen+3].Text != "(" {
		return compositeMethodCall{}, false
	}
	innerStart := pos + 1
	pointer := false
	if innerStart < closeParen && toks[innerStart].Text == "&" {
		pointer = true
		innerStart++
	}
	if !tokensAreCompositeLiteral(toks, innerStart, closeParen) {
		return compositeMethodCall{}, false
	}
	open := compositeLiteralOpenForTypeStart(toks, innerStart)
	if open < 0 {
		return compositeMethodCall{}, false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace != closeParen-1 {
		return compositeMethodCall{}, false
	}
	callClose := findClose(toks, closeParen+3, "(", ")")
	return compositeMethodCall{
		receiverStart: pos,
		receiverEnd:   closeParen + 1,
		typeStart:     innerStart,
		typeEnd:       open,
		literalOpen:   open,
		literalClose:  closeBrace,
		methodName:    closeParen + 2,
		methodOpen:    closeParen + 3,
		callClose:     callClose,
		pointer:       pointer,
	}, true
}

func bareCompositeLiteralMethodCallAt(toks []scan.Token, pos int) (compositeMethodCall, bool) {
	open := compositeLiteralOpenForTypeStart(toks, pos)
	if open < 0 {
		return compositeMethodCall{}, false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace+3 >= len(toks) || toks[closeBrace+1].Text != "." || toks[closeBrace+2].Kind != scan.Ident || toks[closeBrace+3].Text != "(" {
		return compositeMethodCall{}, false
	}
	if !tokensAreCompositeLiteral(toks, pos, closeBrace+1) {
		return compositeMethodCall{}, false
	}
	callClose := findClose(toks, closeBrace+3, "(", ")")
	return compositeMethodCall{
		receiverStart: pos,
		receiverEnd:   closeBrace + 1,
		typeStart:     pos,
		typeEnd:       open,
		literalOpen:   open,
		literalClose:  closeBrace,
		methodName:    closeBrace + 2,
		methodOpen:    closeBrace + 3,
		callClose:     callClose,
	}, true
}

func compositeLiteralMethodValueAt(toks []scan.Token, start int, end int) (compositeMethodCall, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return compositeMethodCall{}, false
	}
	if toks[start].Text == "(" {
		return parenthesizedCompositeLiteralMethodValueAt(toks, start, end)
	}
	return bareCompositeLiteralMethodValueAt(toks, start, end)
}

func parenthesizedCompositeLiteralMethodValueAt(toks []scan.Token, start int, end int) (compositeMethodCall, bool) {
	closeParen := findClose(toks, start, "(", ")")
	if closeParen <= start || closeParen+2 >= end || closeParen+3 != end || toks[closeParen+1].Text != "." || toks[closeParen+2].Kind != scan.Ident {
		return compositeMethodCall{}, false
	}
	innerStart := start + 1
	pointer := false
	if innerStart < closeParen && toks[innerStart].Text == "&" {
		pointer = true
		innerStart++
	}
	if !tokensAreCompositeLiteral(toks, innerStart, closeParen) {
		return compositeMethodCall{}, false
	}
	open := compositeLiteralOpenForTypeStart(toks, innerStart)
	if open < 0 {
		return compositeMethodCall{}, false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace != closeParen-1 {
		return compositeMethodCall{}, false
	}
	return compositeMethodCall{
		receiverStart: start,
		receiverEnd:   closeParen + 1,
		typeStart:     innerStart,
		typeEnd:       open,
		literalOpen:   open,
		literalClose:  closeBrace,
		methodName:    closeParen + 2,
		pointer:       pointer,
	}, true
}

func bareCompositeLiteralMethodValueAt(toks []scan.Token, start int, end int) (compositeMethodCall, bool) {
	open := compositeLiteralOpenForTypeStart(toks, start)
	if open < 0 {
		return compositeMethodCall{}, false
	}
	closeBrace := findClose(toks, open, "{", "}")
	if closeBrace < 0 || closeBrace+2 >= end || closeBrace+3 != end || toks[closeBrace+1].Text != "." || toks[closeBrace+2].Kind != scan.Ident {
		return compositeMethodCall{}, false
	}
	if !tokensAreCompositeLiteral(toks, start, closeBrace+1) {
		return compositeMethodCall{}, false
	}
	return compositeMethodCall{
		receiverStart: start,
		receiverEnd:   closeBrace + 1,
		typeStart:     start,
		typeEnd:       open,
		literalOpen:   open,
		literalClose:  closeBrace,
		methodName:    closeBrace + 2,
	}, true
}

func compositeLiteralOpenForTypeStart(toks []scan.Token, start int) int {
	if start < 0 || start >= len(toks) || toks[start].Kind != scan.Ident {
		return -1
	}
	if start+1 < len(toks) && toks[start+1].Text == "{" {
		return start + 1
	}
	if start+3 < len(toks) && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident && toks[start+3].Text == "{" {
		return start + 3
	}
	return -1
}

func compositeMethodReceiverArg(source []byte, toks []scan.Token, call compositeMethodCall, method methodInfo, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	typeText, ok := loweredCompositeReceiverType(toks, call.typeStart, call.typeEnd, topNames, importRefs, refs)
	if !ok {
		return "", false
	}
	literalStart := int(toks[call.literalOpen].Start)
	literalEnd := int(toks[call.literalClose].End)
	receiver := typeText + lowerGeneratedCompositeLiteralTypeNames(string(source[literalStart:literalEnd]), topNames, importRefs, refs)
	if call.pointer && method.pointerReceiver {
		receiver = "&" + receiver
	}
	return receiver, true
}

func compositePromotedMethodReceiverArg(source []byte, toks []scan.Token, call compositeMethodCall, path []string, promotedReceiver localTypeInfo, method methodInfo, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	typeText, ok := loweredCompositeReceiverType(toks, call.typeStart, call.typeEnd, topNames, importRefs, refs)
	if !ok {
		return "", false
	}
	literalStart := int(toks[call.literalOpen].Start)
	literalEnd := int(toks[call.literalClose].End)
	receiver := typeText + lowerGeneratedCompositeLiteralTypeNames(string(source[literalStart:literalEnd]), topNames, importRefs, refs)
	for i := 0; i < len(path); i++ {
		receiver += "." + path[i]
	}
	if method.pointerReceiver && !promotedReceiver.pointer {
		return "&(" + receiver + ")", true
	}
	if !method.pointerReceiver && promotedReceiver.pointer {
		return "*(" + receiver + ")", true
	}
	return receiver, true
}

func lowerGeneratedCompositeLiteralTypeNames(body string, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) string {
	if body == "" || !strings.Contains(body, "{") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var out []byte
	cursor := 0
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			break
		}
		if tok.Kind != scan.Ident || isCompositeKey(toks, i) {
			continue
		}
		if i+3 < len(toks) && toks[i+1].Text == "." && toks[i+2].Kind == scan.Ident && toks[i+3].Text == "{" {
			group, ok := importSymbolTableGroup(importRefs, tok.Text)
			if !ok {
				continue
			}
			sym, symOK := importSymbolByName(group, toks[i+2].Text)
			if !symOK {
				continue
			}
			if sym.ImportPath != "" {
				appendUnitSymbolRef(refs, sym)
			}
			out = appendStringRange(out, body, cursor, int(tok.Start))
			out = appendString(out, sym.UnitName)
			cursor = int(toks[i+2].End)
			i += 2
			continue
		}
		if i+1 < len(toks) && toks[i+1].Text == "{" {
			unitName := symbolNameTableUnitName(topNames, tok.Text)
			if unitName == "" {
				continue
			}
			out = appendStringRange(out, body, cursor, int(tok.Start))
			out = appendString(out, unitName)
			cursor = int(tok.End)
		}
	}
	if len(out) == 0 {
		return body
	}
	out = appendStringRange(out, body, cursor, len(body))
	return string(out)
}

func loweredCompositeReceiverType(toks []scan.Token, start int, end int, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) (string, bool) {
	if start+1 == end && toks[start].Kind == scan.Ident {
		unitName := symbolNameTableUnitName(topNames, toks[start].Text)
		if unitName != "" {
			return unitName, true
		}
		return toks[start].Text, true
	}
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		group, ok := importSymbolTableGroup(importRefs, toks[start].Text)
		if ok {
			sym, symOK := importSymbolByName(group, toks[start+2].Text)
			if symOK {
				if sym.ImportPath != "" {
					appendUnitSymbolRef(refs, sym)
				}
				return sym.UnitName, true
			}
		}
		return toks[start].Text + "." + toks[start+2].Text, true
	}
	return "", false
}

func appendMethodDeclPrefix(file *parse.File, decl *parse.Decl, topNames symbolNameTable, out *[]byte) int {
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 || start+1 >= len(toks) || toks[start+1].Text != "(" {
		return decl.Start
	}
	receiverOpen := start + 1
	receiverClose := findClose(toks, receiverOpen, "(", ")")
	if receiverClose < 0 || receiverClose+2 >= len(toks) {
		return decl.Start
	}
	nameTok := receiverClose + 1
	paramsOpen := nameTok + 1
	if toks[nameTok].Kind != scan.Ident || toks[paramsOpen].Text != "(" {
		return decl.Start
	}
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return decl.Start
	}
	unitName := symbolNameTableUnitName(topNames, methodDeclNameFromTokens(toks, decl))
	if unitName == "" {
		return decl.Start
	}
	appendStringRef(out, "func ")
	appendStringRef(out, unitName)
	appendByteRef(out, '(')
	appendStringRef(out, rewriteReceiverSegment(file, receiverOpen+1, receiverClose, topNames))
	if paramsOpen+1 < paramsClose {
		appendStringRef(out, ", ")
		return int(toks[paramsOpen].End)
	}
	return int(toks[paramsClose].Start)
}

func appendStringRef(out *[]byte, s string) {
	values := *out
	values = appendString(values, s)
	*out = values
}

func appendByteRef(out *[]byte, c byte) {
	values := *out
	values = append(values, c)
	*out = values
}

func rewriteReceiverSegment(file *parse.File, start int, end int, topNames symbolNameTable) string {
	toks := file.Tokens
	source := file.Source
	capacity := 512
	if end > start {
		capacity = rewriteBufferCapacity(int(toks[end-1].End) - int(toks[start].Start))
	}
	out := make([]byte, 0, capacity)
	cursor := int(toks[start].Start)
	for i := start; i < end; i++ {
		tok := toks[i]
		if int(tok.Start) > cursor {
			part := source[cursor:int(tok.Start)]
			out = appendBytes(out, part)
		}
		if tok.Kind == scan.Ident && (i > start || !receiverSegmentHasName(toks, start, end)) {
			unitName := symbolNameTableUnitName(topNames, tok.Text)
			if unitName != "" {
				out = appendString(out, unitName)
				cursor = int(tok.End)
				continue
			}
		}
		part := source[int(tok.Start):int(tok.End)]
		out = appendBytes(out, part)
		cursor = int(tok.End)
	}
	if end > start && cursor < int(toks[end-1].End) {
		part := source[cursor:int(toks[end-1].End)]
		out = appendBytes(out, part)
	}
	return string(out)
}

func receiverSegmentHasName(toks []scan.Token, start int, end int) bool {
	idents := 0
	for i := start; i < end; i++ {
		if toks[i].Kind == scan.Ident {
			idents++
		}
	}
	return idents > 1
}

func normalizeKeyedSliceLiterals(body string, unitName string, namedSlices []namedSliceInfo, namedConversions []string) string {
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	tempIndex := 0
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "{" {
			continue
		}
		close := findClose(toks, i, "{", "}")
		if close <= i {
			continue
		}
		temps, repl, ok := normalizeKeyedSliceCompositeLiteral(body, toks, i, close, unitName, &tempIndex, namedSlices, namedConversions, normalizationContext{})
		if ok && len(temps) == 0 {
			replacements = append(replacements, repl)
			i = close
		}
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	normalized := applyExpressionReplacements(body, 0, len(body), replacements)
	normalized = arena.PersistString(normalized)
	arena.Reset(mark)
	return normalized
}

type expressionTemp struct {
	name string
	typ  string
	expr string
}

func appendExpressionTempDecl(out []byte, indent string, temp expressionTemp) []byte {
	out = appendString(out, indent)
	if temp.typ != "" {
		out = appendString(out, "var ")
		out = appendString(out, temp.name)
		out = append(out, ' ')
		out = appendString(out, temp.typ)
		if temp.expr != "" {
			out = appendString(out, " = ")
			out = appendString(out, temp.expr)
		}
		out = append(out, '\n')
		return out
	}
	out = appendString(out, temp.name)
	out = appendString(out, " := ")
	out = appendString(out, temp.expr)
	out = append(out, '\n')
	return out
}

type expressionReplacement struct {
	start int
	end   int
	text  string
}

type fallthroughRewrite struct {
	token int
	colon int
	label string
}

type labeledControlRewrite struct {
	token   int
	operand int
	close   int
	kind    string
	label   string
}

func fallthroughRewrites(toks []scan.Token, start int, end int) []fallthroughRewrite {
	var out []fallthroughRewrite
	for i := 0; i < len(toks); i++ {
		if int(toks[i].End) <= start {
			continue
		}
		if int(toks[i].Start) >= end {
			break
		}
		if toks[i].Text != "fallthrough" {
			continue
		}
		colon := fallthroughTargetCaseColon(toks, i)
		if colon < 0 {
			continue
		}
		label := "rtg_fallthrough_" + strconv.Itoa(len(out))
		out = append(out, fallthroughRewrite{token: i, colon: colon, label: label})
	}
	return out
}

func labeledControlRewrites(toks []scan.Token, start int, end int) []labeledControlRewrite {
	var out []labeledControlRewrite
	for i := 0; i < len(toks); i++ {
		if int(toks[i].End) <= start {
			continue
		}
		if int(toks[i].Start) >= end {
			break
		}
		kind := toks[i].Text
		if kind != "break" && kind != "continue" {
			continue
		}
		if i+1 >= len(toks) || toks[i+1].Kind != scan.Ident || toks[i+1].Line != toks[i].Line {
			continue
		}
		close, owner, ok := labeledControlTargetForLower(toks, i)
		if !ok {
			continue
		}
		if kind == "continue" && owner != "for" {
			continue
		}
		if kind == "break" && owner != "for" && owner != "switch" {
			continue
		}
		label := "rtg_labeled_" + kind + "_" + strconv.Itoa(len(out))
		out = append(out, labeledControlRewrite{token: i, operand: i + 1, close: close, kind: kind, label: label})
	}
	return out
}

func labeledControlForToken(rewrites []labeledControlRewrite, token int) (labeledControlRewrite, bool) {
	for i := 0; i < len(rewrites); i++ {
		if rewrites[i].token == token {
			return rewrites[i], true
		}
	}
	return labeledControlRewrite{}, false
}

func labeledControlLabelsForClose(rewrites []labeledControlRewrite, close int, kind string) []string {
	var labels []string
	for i := 0; i < len(rewrites); i++ {
		if rewrites[i].close == close && rewrites[i].kind == kind {
			labels = append(labels, rewrites[i].label)
		}
	}
	return labels
}

func labeledControlTargetForLower(toks []scan.Token, pos int) (int, string, bool) {
	label := toks[pos+1].Text
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != label || toks[i+1].Text != ":" {
			continue
		}
		target := nextNonSemicolonTokenForLower(toks, i+2)
		if target < 0 {
			continue
		}
		owner := toks[target].Text
		if owner != "for" && owner != "switch" {
			continue
		}
		open, close := labeledControlTargetBodyForLower(toks, target, pos)
		if open >= 0 && close > pos {
			return close, owner, true
		}
	}
	return -1, "", false
}

func nextNonSemicolonTokenForLower(toks []scan.Token, start int) int {
	for i := start; i < len(toks); i++ {
		if toks[i].Text == ";" {
			continue
		}
		return i
	}
	return -1
}

func labeledControlTargetBodyForLower(toks []scan.Token, target int, pos int) (int, int) {
	owner := toks[target].Text
	for i := target + 1; i < len(toks) && i <= pos; i++ {
		if toks[i].Text != "{" {
			continue
		}
		close := findClose(toks, i, "{", "}")
		if close <= pos {
			continue
		}
		if lowerBlockOwnerKeyword(toks, i) == owner {
			return i, close
		}
	}
	return -1, -1
}

func fallthroughLabelForToken(rewrites []fallthroughRewrite, token int) (string, bool) {
	for i := 0; i < len(rewrites); i++ {
		if rewrites[i].token == token {
			return rewrites[i].label, true
		}
	}
	return "", false
}

func fallthroughLabelsForCaseColon(rewrites []fallthroughRewrite, colon int) []string {
	var labels []string
	for i := 0; i < len(rewrites); i++ {
		if rewrites[i].colon == colon {
			labels = append(labels, rewrites[i].label)
		}
	}
	return labels
}

func fallthroughLabelIndent(source []byte, pos int) string {
	lineStart := pos
	for lineStart > 0 && source[lineStart-1] != '\n' {
		lineStart--
	}
	lineEnd := lineStart
	for lineEnd < len(source) && (source[lineEnd] == ' ' || source[lineEnd] == '\t') {
		lineEnd++
	}
	return string(source[lineStart:lineEnd]) + "\t"
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
				return caseClauseColon(toks, i, close)
			}
			if text != ";" {
				return -1
			}
		}
		updateExpressionDepth(text, &paren, &brack, &brace)
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
		if lowerBlockOwnerKeyword(toks, i) == "switch" {
			return i
		}
	}
	return -1
}

func lowerBlockOwnerKeyword(toks []scan.Token, open int) string {
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

func caseClauseColon(toks []scan.Token, start int, limit int) int {
	if toks[start].Text == "default" {
		if start+1 < limit && toks[start+1].Text == ":" {
			return start + 1
		}
		return -1
	}
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < limit; i++ {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 && text == ":" {
			return i
		}
		updateExpressionDepth(text, &paren, &brack, &brace)
	}
	return -1
}

type expressionStatement struct {
	token           int
	exprStart       int
	exprEnd         int
	kind            string
	openBrace       int
	returnStatement bool
	forCondition    bool
}

type compoundAssignmentStatement struct {
	token    int
	assign   int
	lhsStart int
	lhsEnd   int
	rhsStart int
	rhsEnd   int
}

type rangeStatement struct {
	token      int
	lhsStart   int
	lhsEnd     int
	assign     int
	rangeToken int
	exprStart  int
	exprEnd    int
	openBrace  int
	end        int
	define     bool
	names      []string
	targets    []expressionRange
}

type normalizationContext struct {
	localTypes      localTypeTable
	functionResults localTypeTable
	namedTypes      localTypeTable
	fieldTypes      structFieldTypeTable
	generatedDecls  *[]unit.Decl
}

type conditionalShortStatement struct {
	token     int
	initStart int
	semi      int
	condStart int
	condEnd   int
	openBrace int
	end       int
	kind      string
}

type classicForConditionStatement struct {
	token     int
	condStart int
	condEnd   int
	openBrace int
}

type classicForPostStatement struct {
	token     int
	initStart int
	initEnd   int
	condStart int
	condEnd   int
	postStart int
	postEnd   int
	openBrace int
	end       int
}

type shortCircuitIfStatement struct {
	token     int
	condStart int
	condEnd   int
	openBrace int
	end       int
	operands  []expressionRange
}

func normalizeFunctionExpressions(body string, unitName string, namedSlices []namedSliceInfo, namedConversions []string) string {
	tempIndex := 0
	return normalizeFunctionExpressionsWithTemp(body, unitName, &tempIndex, namedSlices, namedConversions, normalizationContext{})
}

func normalizeFunctionExpressionsWithContext(body string, unitName string, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) string {
	tempIndex := 0
	return normalizeFunctionExpressionsWithTemp(body, unitName, &tempIndex, namedSlices, namedConversions, ctx)
}

func normalizeFunctionExpressionsWithTemp(body string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) string {
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	out := make([]byte, 0, rewriteBufferCapacity(len(body)))
	cursor := 0
	for i := 0; i < len(toks); i++ {
		rangeStmt, ok := normalizationRangeStatement(toks, i)
		if ok {
			insertStart := statementInsertStart(body, int(toks[rangeStmt.token].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[rangeStmt.token].Start))
			appendRangeStatementLowering(&out, body, toks, rangeStmt, indent, unitName, tempIndex, namedSlices, namedConversions, ctx)
			cursor = int(toks[rangeStmt.end].End)
			i = rangeStmt.end
			continue
		}
		short, ok := normalizationConditionalShortStatement(toks, i)
		if ok {
			initTemps, initReplacements, initMapOK := normalizeMapLiteralCommaOkSimpleStatement(body, toks, short.initStart, short.semi, unitName, tempIndex)
			if !initMapOK {
				initTemps, initReplacements = normalizeExpressionWithContext(body, toks, short.initStart, short.semi, unitName, tempIndex, namedSlices, namedConversions, ctx)
			}
			condTemps, condReplacements := normalizeExpressionWithContext(body, toks, short.condStart, short.condEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
			insertStart := statementInsertStart(body, int(toks[short.token].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[short.token].Start))
			innerIndent := indent + "\t"
			init := strings.TrimSpace(applyExpressionReplacements(body, int(toks[short.initStart].Start), int(toks[short.semi-1].End), initReplacements))
			condition := strings.TrimSpace(applyExpressionReplacements(body, int(toks[short.condStart].Start), int(toks[short.condEnd-1].End), condReplacements))
			out = appendString(out, indent)
			out = appendString(out, "{\n")
			for j := 0; j < len(initTemps); j++ {
				temp := initTemps[j]
				out = appendExpressionTempDecl(out, innerIndent, temp)
			}
			out = appendString(out, innerIndent)
			out = appendString(out, init)
			out = append(out, '\n')
			for j := 0; j < len(condTemps); j++ {
				temp := condTemps[j]
				out = appendExpressionTempDecl(out, innerIndent, temp)
			}
			out = appendString(out, innerIndent)
			out = appendString(out, short.kind)
			out = append(out, ' ')
			out = appendString(out, condition)
			out = append(out, ' ')
			out = appendStringRange(out, body, int(toks[short.openBrace].Start), int(toks[short.end].End))
			out = append(out, '\n')
			out = appendString(out, indent)
			out = append(out, '}')
			cursor = int(toks[short.end].End)
			i = short.end
			continue
		}
		shortCircuit, ok := normalizationShortCircuitIfStatement(toks, i)
		if ok {
			insertStart := statementInsertStart(body, int(toks[shortCircuit.token].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[shortCircuit.token].Start))
			appendShortCircuitIf(&out, body, toks, shortCircuit, indent, unitName, tempIndex, namedSlices, namedConversions, ctx)
			cursor = int(toks[shortCircuit.end].End)
			i = shortCircuit.end
			continue
		}
		post, ok := normalizationClassicForPostStatement(toks, i)
		if ok {
			if expressionContainsCall(toks, post.postStart, post.postEnd) {
				initTemps, initReplacements, initMapOK := normalizeMapLiteralCommaOkSimpleStatement(body, toks, post.initStart, post.initEnd, unitName, tempIndex)
				if !initMapOK {
					initTemps, initReplacements = normalizeExpressionWithContext(body, toks, post.initStart, post.initEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
				}
				condTemps, condReplacements := normalizeExpressionWithContext(body, toks, post.condStart, post.condEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
				postTemps, postReplacements := normalizeExpressionWithContext(body, toks, post.postStart, post.postEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
				insertStart := statementInsertStart(body, int(toks[post.token].Start))
				out = appendStringRange(out, body, cursor, insertStart)
				indent := statementIndent(body, int(toks[post.token].Start))
				innerIndent := indent + "\t"
				loopIndent := innerIndent + "\t"
				out = appendString(out, indent)
				out = appendString(out, "{\n")
				for j := 0; j < len(initTemps); j++ {
					temp := initTemps[j]
					out = appendExpressionTempDecl(out, innerIndent, temp)
				}
				if post.initStart < post.initEnd {
					init := strings.TrimSpace(applyExpressionReplacements(body, int(toks[post.initStart].Start), int(toks[post.initEnd-1].End), initReplacements))
					out = appendString(out, innerIndent)
					out = appendString(out, init)
					out = append(out, '\n')
				}
				out = appendString(out, innerIndent)
				out = appendString(out, "for {\n")
				if post.condStart < post.condEnd {
					condition := strings.TrimSpace(applyExpressionReplacements(body, int(toks[post.condStart].Start), int(toks[post.condEnd-1].End), condReplacements))
					for j := 0; j < len(condTemps); j++ {
						temp := condTemps[j]
						out = appendExpressionTempDecl(out, loopIndent, temp)
					}
					out = appendString(out, loopIndent)
					out = appendString(out, "if !(")
					out = appendString(out, condition)
					out = appendString(out, ") {\n")
					out = appendString(out, loopIndent)
					out = appendString(out, "\tbreak\n")
					out = appendString(out, loopIndent)
					out = appendString(out, "}\n")
				}
				out = appendStringRange(out, body, int(toks[post.openBrace].End), int(toks[post.end].Start))
				if len(out) == 0 || out[len(out)-1] != '\n' {
					out = append(out, '\n')
				}
				for j := 0; j < len(postTemps); j++ {
					temp := postTemps[j]
					out = appendExpressionTempDecl(out, loopIndent, temp)
				}
				postExpr := strings.TrimSpace(applyExpressionReplacements(body, int(toks[post.postStart].Start), int(toks[post.postEnd-1].End), postReplacements))
				out = appendString(out, loopIndent)
				out = appendString(out, postExpr)
				out = append(out, '\n')
				out = appendString(out, innerIndent)
				out = appendString(out, "}\n")
				out = appendString(out, indent)
				out = append(out, '}')
				cursor = int(toks[post.end].End)
				i = post.end
				continue
			}
		}
		classic, ok := normalizationClassicForConditionStatement(toks, i)
		if ok {
			temps, replacements := normalizeExpressionWithContext(body, toks, classic.condStart, classic.condEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if len(temps) > 0 {
				condition := applyExpressionReplacements(body, int(toks[classic.condStart].Start), int(toks[classic.condEnd-1].End), replacements)
				out = appendStringRange(out, body, cursor, int(toks[classic.condStart].Start))
				out = appendStringRange(out, body, int(toks[classic.condEnd].Start), int(toks[classic.openBrace].End))
				indent := statementIndent(body, int(toks[classic.token].Start))
				innerIndent := indent + "\t"
				out = append(out, '\n')
				for j := 0; j < len(temps); j++ {
					temp := temps[j]
					out = appendExpressionTempDecl(out, innerIndent, temp)
				}
				out = appendString(out, innerIndent)
				out = appendString(out, "if !(")
				out = appendString(out, condition)
				out = appendString(out, ") {\n")
				out = appendString(out, innerIndent)
				out = appendString(out, "\tbreak\n")
				out = appendString(out, innerIndent)
				out = appendString(out, "}\n")
				cursor = int(toks[classic.openBrace].End)
				i = classic.openBrace
				continue
			}
		}
		compound, ok := normalizationCompoundAssignmentStatement(toks, i)
		if ok {
			temps, replacements := normalizeExpressionWithContext(body, toks, compound.rhsStart, compound.rhsEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
			insertStart := statementInsertStart(body, int(toks[compound.token].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[compound.token].Start))
			if insertStart == int(toks[compound.token].Start) {
				out = append(out, '\n')
			}
			for j := 0; j < len(temps); j++ {
				temp := temps[j]
				out = appendExpressionTempDecl(out, indent, temp)
			}
			lhs := strings.TrimSpace(applyExpressionReplacements(body, int(toks[compound.lhsStart].Start), int(toks[compound.lhsEnd-1].End), nil))
			rhs := strings.TrimSpace(applyExpressionReplacements(body, int(toks[compound.rhsStart].Start), int(toks[compound.rhsEnd-1].End), replacements))
			out = appendStringRange(out, body, insertStart, int(toks[compound.lhsStart].Start))
			out = appendString(out, lhs)
			out = appendString(out, " = ")
			out = appendString(out, lhs)
			out = appendString(out, " + ")
			out = appendString(out, rhs)
			cursor = int(toks[compound.rhsEnd-1].End)
			i = compound.rhsEnd - 1
			continue
		}
		commaOKTemps, commaOKReplacement, commaOKStart, commaOKEnd, commaOK := normalizeMapLiteralCommaOkAssignment(body, toks, i, unitName, tempIndex)
		if commaOK {
			insertStart := statementInsertStart(body, int(toks[commaOKStart].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[commaOKStart].Start))
			if insertStart == int(toks[commaOKStart].Start) {
				out = append(out, '\n')
			}
			for j := 0; j < len(commaOKTemps); j++ {
				temp := commaOKTemps[j]
				out = appendExpressionTempDecl(out, indent, temp)
			}
			out = appendStringRange(out, body, insertStart, commaOKReplacement.start)
			out = appendString(out, commaOKReplacement.text)
			cursor = commaOKReplacement.end
			i = commaOKEnd - 1
			continue
		}
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		if stmt.returnStatement {
			temps, replacements := normalizeReturnValues(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if len(temps) == 0 && len(replacements) == 0 {
				continue
			}
			insertStart := statementInsertStart(body, int(toks[stmt.token].Start))
			out = appendStringRange(out, body, cursor, insertStart)
			indent := statementIndent(body, int(toks[stmt.token].Start))
			if insertStart == int(toks[stmt.token].Start) {
				out = append(out, '\n')
			}
			for j := 0; j < len(temps); j++ {
				temp := temps[j]
				out = appendExpressionTempDecl(out, indent, temp)
			}
			out = appendStringRange(out, body, insertStart, int(toks[stmt.exprStart].Start))
			out = appendString(out, applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements))
			cursor = int(toks[stmt.exprEnd-1].End)
			i = stmt.exprEnd - 1
			continue
		}
		temps, replacements := normalizeExpressionWithContext(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
		if len(temps) == 0 && len(replacements) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, int(toks[stmt.token].Start))
		out = appendStringRange(out, body, cursor, insertStart)
		indent := statementIndent(body, int(toks[stmt.token].Start))
		if stmt.forCondition {
			innerIndent := indent + "\t"
			condition := applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements)
			out = appendStringRange(out, body, insertStart, int(toks[stmt.token].Start))
			out = appendString(out, "for {\n")
			for j := 0; j < len(temps); j++ {
				temp := temps[j]
				out = appendExpressionTempDecl(out, innerIndent, temp)
			}
			out = appendString(out, innerIndent)
			out = appendString(out, "if !(")
			out = appendString(out, condition)
			out = appendString(out, ") {\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "\tbreak\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "}\n")
			cursor = int(toks[stmt.openBrace].End)
			i = stmt.openBrace
			continue
		}
		if insertStart == int(toks[stmt.token].Start) {
			out = append(out, '\n')
		}
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			out = appendExpressionTempDecl(out, indent, temp)
		}
		out = appendStringRange(out, body, insertStart, int(toks[stmt.exprStart].Start))
		out = appendString(out, applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements))
		cursor = int(toks[stmt.exprEnd-1].End)
		i = stmt.exprEnd - 1
	}
	if len(out) == 0 {
		arena.Reset(mark)
		return body
	}
	out = appendStringRange(out, body, cursor, len(body))
	if strings.HasPrefix(strings.TrimSpace(body), "func ") && !trimmedBytesHavePrefix(out, "func ") {
		arena.Reset(mark)
		return body
	}
	normalized := string(out)
	normalized = arena.PersistString(normalized)
	arena.Reset(mark)
	return normalized
}

func appendShortCircuitIf(out *[]byte, body string, toks []scan.Token, stmt shortCircuitIfStatement, indent string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) {
	currentIndent := indent
	for i := 0; i < len(stmt.operands); i++ {
		operand := stmt.operands[i]
		temps, replacements := normalizeExpressionWithContext(body, toks, operand.start, operand.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			*out = appendExpressionTempDecl(*out, currentIndent, temp)
		}
		condition := strings.TrimSpace(applyExpressionReplacements(body, int(toks[operand.start].Start), int(toks[operand.end-1].End), replacements))
		appendStringRef(out, currentIndent)
		appendStringRef(out, "if ")
		appendStringRef(out, condition)
		appendStringRef(out, " {\n")
		currentIndent = currentIndent + "\t"
	}
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBodySource := body[innerBodyStart:innerBodyEnd]
	innerBody := normalizeFunctionExpressionsWithTemp(innerBodySource, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, innerBody)
	if len(*out) == 0 || (*out)[len(*out)-1] != '\n' {
		appendByteRef(out, '\n')
	}
	for i := len(stmt.operands) - 1; i >= 0; i-- {
		currentIndent = currentIndent[:len(currentIndent)-1]
		appendStringRef(out, currentIndent)
		appendByteRef(out, '}')
		if i > 0 {
			appendByteRef(out, '\n')
		}
	}
}

func appendRangeStatementLowering(out *[]byte, body string, toks []scan.Token, stmt rangeStatement, indent string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) {
	if info, ok := lowerableMapRangeInfoForLower(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex); ok {
		appendMapRangeStatementLowering(out, body, toks, stmt, info, indent, unitName, tempIndex, namedSlices, namedConversions, ctx)
		return
	}
	if rangeStatementOperandIsString(body, toks, stmt, ctx) {
		appendStringRangeStatementLowering(out, body, toks, stmt, indent, unitName, tempIndex, namedSlices, namedConversions, ctx)
		return
	}
	innerIndent := indent + "\t"
	loopIndent := innerIndent + "\t"
	seqName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	indexName := rangeLoopIndexName(body, unitName, tempIndex, stmt)
	temps, expr := normalizeRangeOperand(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, indent)
	appendStringRef(out, "{\n")
	for i := 0; i < len(temps); i++ {
		temp := temps[i]
		*out = appendExpressionTempDecl(*out, innerIndent, temp)
	}
	appendStringRef(out, innerIndent)
	appendStringRef(out, seqName)
	appendStringRef(out, " := ")
	appendStringRef(out, expr)
	appendByteRef(out, '\n')
	appendStringRef(out, innerIndent)
	appendStringRef(out, "for ")
	appendStringRef(out, indexName)
	appendStringRef(out, " := 0; ")
	appendStringRef(out, indexName)
	appendStringRef(out, " < len(")
	appendStringRef(out, seqName)
	appendStringRef(out, "); ")
	appendStringRef(out, indexName)
	appendStringRef(out, "++ {")
	appendRangeBindings(out, body, toks, stmt, seqName, indexName, seqName+"["+indexName+"]", loopIndent)
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBodySource := body[innerBodyStart:innerBodyEnd]
	innerBody := normalizeFunctionExpressionsWithTemp(innerBodySource, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, innerBody)
	if len(*out) == 0 || (*out)[len(*out)-1] != '\n' {
		appendByteRef(out, '\n')
	}
	appendStringRef(out, innerIndent)
	appendStringRef(out, "}\n")
	appendStringRef(out, indent)
	appendByteRef(out, '}')
}

func appendStringRangeStatementLowering(out *[]byte, body string, toks []scan.Token, stmt rangeStatement, indent string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) {
	innerIndent := indent + "\t"
	loopIndent := innerIndent + "\t"
	seqName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	indexName := rangeLoopIndexName(body, unitName, tempIndex, stmt)
	widthName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	byteName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	runeName := ""
	if rangeStatementNeedsValue(toks, stmt) {
		runeName = nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
	}
	temps, expr := normalizeRangeOperand(body, toks, stmt.exprStart, stmt.exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, indent)
	appendStringRef(out, "{\n")
	for i := 0; i < len(temps); i++ {
		temp := temps[i]
		*out = appendExpressionTempDecl(*out, innerIndent, temp)
	}
	appendStringRef(out, innerIndent)
	appendStringRef(out, seqName)
	appendStringRef(out, " := ")
	appendStringRef(out, expr)
	appendByteRef(out, '\n')
	appendStringRef(out, innerIndent)
	appendStringRef(out, "for ")
	appendStringRef(out, indexName)
	appendStringRef(out, " := 0; ")
	appendStringRef(out, indexName)
	appendStringRef(out, " < len(")
	appendStringRef(out, seqName)
	appendStringRef(out, "); {\n")
	appendStringRef(out, loopIndent)
	appendStringRef(out, widthName)
	appendStringRef(out, " := 1\n")
	appendStringRef(out, loopIndent)
	appendStringRef(out, byteName)
	appendStringRef(out, " := int(")
	appendStringRef(out, seqName)
	appendByteRef(out, '[')
	appendStringRef(out, indexName)
	appendStringRef(out, "])\n")
	if runeName != "" {
		appendStringRef(out, loopIndent)
		appendStringRef(out, runeName)
		appendStringRef(out, " := int32(")
		appendStringRef(out, byteName)
		appendStringRef(out, ")\n")
	}
	appendStringRangeDecode(out, seqName, indexName, widthName, byteName, runeName, loopIndent)
	valueExpr := runeName
	if valueExpr == "" {
		valueExpr = "int32(" + byteName + ")"
	}
	appendRangeBindings(out, body, toks, stmt, seqName, indexName, valueExpr, loopIndent)
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBodySource := body[innerBodyStart:innerBodyEnd]
	innerBody := normalizeFunctionExpressionsWithTemp(innerBodySource, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, innerBody)
	if len(*out) == 0 || (*out)[len(*out)-1] != '\n' {
		appendByteRef(out, '\n')
	}
	appendStringRef(out, loopIndent)
	appendStringRef(out, indexName)
	appendStringRef(out, " = ")
	appendStringRef(out, indexName)
	appendStringRef(out, " + ")
	appendStringRef(out, widthName)
	appendByteRef(out, '\n')
	appendStringRef(out, innerIndent)
	appendStringRef(out, "}\n")
	appendStringRef(out, indent)
	appendByteRef(out, '}')
}

type mapRangeInfo struct {
	keyType   string
	valueType string
	keys      []string
	keyIDs    []string
	values    []string
	temps     []expressionTemp
}

func appendMapRangeStatementLowering(out *[]byte, body string, toks []scan.Token, stmt rangeStatement, info mapRangeInfo, indent string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) {
	innerIndent := indent + "\t"
	loopIndent := innerIndent + "\t"
	indexName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	keyNeeded := len(stmt.targets) > 0 && !rangeTargetIsBlank(toks, stmt.targets[0])
	valueNeeded := len(stmt.targets) > 1 && !rangeTargetIsBlank(toks, stmt.targets[1])
	keySeqName := ""
	appendStringRef(out, indent)
	appendStringRef(out, "{\n")
	for i := 0; i < len(info.temps); i++ {
		*out = appendExpressionTempDecl(*out, innerIndent, info.temps[i])
	}
	if keyNeeded {
		keySeqName = nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		appendStringRef(out, innerIndent)
		appendStringRef(out, keySeqName)
		appendStringRef(out, " := []")
		appendStringRef(out, info.keyType)
		appendByteRef(out, '{')
		for i := 0; i < len(info.keys); i++ {
			if i > 0 {
				appendStringRef(out, ", ")
			}
			appendStringRef(out, info.keys[i])
		}
		appendStringRef(out, "}\n")
	}
	appendStringRef(out, innerIndent)
	appendStringRef(out, "for ")
	appendStringRef(out, indexName)
	appendStringRef(out, " := 0; ")
	appendStringRef(out, indexName)
	appendStringRef(out, " < ")
	if keyNeeded {
		appendStringRef(out, "len(")
		appendStringRef(out, keySeqName)
		appendByteRef(out, ')')
	} else {
		appendStringRef(out, strconv.Itoa(len(info.values)))
	}
	appendStringRef(out, "; ")
	appendStringRef(out, indexName)
	appendStringRef(out, "++ {")
	keyExpr := ""
	if keyNeeded {
		keyExpr = keySeqName + "[" + indexName + "]"
	}
	valueExpr := ""
	if valueNeeded {
		valueName := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		valueExpr = valueName
		appendByteRef(out, '\n')
		appendStringRef(out, loopIndent)
		appendStringRef(out, "var ")
		appendStringRef(out, valueName)
		appendByteRef(out, ' ')
		appendStringRef(out, info.valueType)
		appendStringRef(out, " = ")
		appendStringRef(out, mapRangeZeroValueTextForLower(info.valueType))
		for i := 0; i < len(info.values); i++ {
			appendByteRef(out, '\n')
			appendStringRef(out, loopIndent)
			appendStringRef(out, "if ")
			appendStringRef(out, indexName)
			appendStringRef(out, " == ")
			appendStringRef(out, strconv.Itoa(i))
			appendStringRef(out, " {\n")
			appendStringRef(out, loopIndent)
			appendByteRef(out, '\t')
			appendStringRef(out, valueName)
			appendStringRef(out, " = ")
			appendStringRef(out, info.values[i])
			appendByteRef(out, '\n')
			appendStringRef(out, loopIndent)
			appendByteRef(out, '}')
		}
	}
	appendMapRangeBindings(out, body, toks, stmt, keyExpr, valueExpr, loopIndent)
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBodySource := body[innerBodyStart:innerBodyEnd]
	innerBody := normalizeFunctionExpressionsWithTemp(innerBodySource, unitName, tempIndex, namedSlices, namedConversions, ctx)
	appendStringRef(out, innerBody)
	if len(*out) == 0 || (*out)[len(*out)-1] != '\n' {
		appendByteRef(out, '\n')
	}
	appendStringRef(out, innerIndent)
	appendStringRef(out, "}\n")
	appendStringRef(out, indent)
	appendByteRef(out, '}')
}

func appendMapRangeBindings(out *[]byte, body string, toks []scan.Token, stmt rangeStatement, keyExpr string, valueExpr string, loopIndent string) {
	if len(stmt.targets) == 0 {
		appendByteRef(out, '\n')
		return
	}
	if !rangeTargetIsBlank(toks, stmt.targets[0]) {
		appendByteRef(out, '\n')
		appendStringRef(out, loopIndent)
		if stmt.define {
			appendStringRef(out, stmt.names[0])
			appendStringRef(out, " := ")
		} else {
			appendRangeTargetSource(out, body, toks, stmt.targets[0])
			appendStringRef(out, " = ")
		}
		appendStringRef(out, keyExpr)
	}
	if len(stmt.targets) > 1 && !rangeTargetIsBlank(toks, stmt.targets[1]) {
		appendByteRef(out, '\n')
		appendStringRef(out, loopIndent)
		if stmt.define {
			appendStringRef(out, stmt.names[1])
			appendStringRef(out, " := ")
		} else {
			appendRangeTargetSource(out, body, toks, stmt.targets[1])
			appendStringRef(out, " = ")
		}
		appendStringRef(out, valueExpr)
	}
	appendByteRef(out, '\n')
}

func lowerableMapRangeInfoForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (mapRangeInfo, bool) {
	if info, ok := mapRangeInfoForLower(body, toks, start, end); ok {
		return info, true
	}
	if info, ok := lowerableMapMakeRangeInfoForLower(body, toks, start, end, unitName, tempIndex); ok {
		return info, true
	}
	return lowerableDirectMapRangeInfoForLower(body, toks, start, end, unitName, tempIndex)
}

func lowerableMapMakeRangeInfoForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (mapRangeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return mapRangeInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableMapMakeRangeInfoForLower(body, toks, start+1, close, unitName, tempIndex)
		}
	}
	if !lowerableMapMakeExpressionForLower(toks, start, end) {
		return mapRangeInfo{}, false
	}
	keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
	if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
		return mapRangeInfo{}, false
	}
	temps, ok := lowerableMapMakeSideEffectTempsForLower(body, toks, start, end, unitName, tempIndex)
	if !ok {
		return mapRangeInfo{}, false
	}
	return mapRangeInfo{keyType: keyType, valueType: valueType, temps: temps}, true
}

func lowerableDirectMapRangeInfoForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (mapRangeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return mapRangeInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableDirectMapRangeInfoForLower(body, toks, start+1, close, unitName, tempIndex)
		}
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return mapRangeInfo{}, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return mapRangeInfo{}, false
	}
	keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
	if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
		return mapRangeInfo{}, false
	}
	info := mapRangeInfo{keyType: keyType, valueType: valueType}
	values := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return mapRangeInfo{}, false
		}
		keyStart, keyEnd := trimTokenRange(toks, value.start, colon)
		if keyStart >= keyEnd {
			return mapRangeInfo{}, false
		}
		keyID, ok := mapLiteralComparableKeyForLower(toks, value.start, colon)
		if !ok {
			return mapRangeInfo{}, false
		}
		valueText, valueTemps, valueOK := lowerableMapSelectedIndexValueForNormalize(body, toks, colon+1, value.end, unitName, tempIndex)
		if !valueOK {
			return mapRangeInfo{}, false
		}
		info.keys = append(info.keys, strings.TrimSpace(tokenRangeText(body, toks, keyStart, keyEnd)))
		info.keyIDs = append(info.keyIDs, keyID)
		info.values = append(info.values, valueText)
		info.temps = appendExpressionTemps(info.temps, valueTemps)
	}
	return info, true
}

func mapRangeInfoForLower(body string, toks []scan.Token, start int, end int) (mapRangeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return mapRangeInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapRangeInfoForLower(body, toks, start+1, close)
		}
	}
	if discardedMapMakeExpressionForLower(toks, start, end) {
		keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
		if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
			return mapRangeInfo{}, false
		}
		return mapRangeInfo{keyType: keyType, valueType: valueType}, true
	}
	if !discardedMapLiteralExpressionForLower(toks, start, end) {
		return mapRangeInfo{}, false
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return mapRangeInfo{}, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return mapRangeInfo{}, false
	}
	keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
	if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
		return mapRangeInfo{}, false
	}
	info := mapRangeInfo{keyType: keyType, valueType: valueType}
	values := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 || !discardedArrayLiteralElementForLower(toks, colon+1, value.end) {
			return mapRangeInfo{}, false
		}
		keyStart, keyEnd := trimTokenRange(toks, value.start, colon)
		valueStart, valueEnd := trimTokenRange(toks, colon+1, value.end)
		if keyStart >= keyEnd || valueStart >= valueEnd {
			return mapRangeInfo{}, false
		}
		keyID, ok := mapLiteralComparableKeyForLower(toks, value.start, colon)
		if !ok {
			return mapRangeInfo{}, false
		}
		info.keys = append(info.keys, strings.TrimSpace(tokenRangeText(body, toks, keyStart, keyEnd)))
		info.keyIDs = append(info.keyIDs, keyID)
		info.values = append(info.values, strings.TrimSpace(tokenRangeText(body, toks, valueStart, valueEnd)))
	}
	return info, true
}

func mapRangeKeyTypeSupportedForLower(typ string) bool {
	switch typ {
	case "bool", "string", "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return false
}

func mapRangeValueTypeSupportedForLower(typ string) bool {
	switch typ {
	case "bool", "string", "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]")
}

func mapRangeZeroValueTextForLower(typ string) string {
	switch typ {
	case "bool":
		return "false"
	case "string":
		return "\"\""
	case "int", "int16", "int32", "int64", "byte", "float64":
		return "0"
	}
	if strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return "nil"
	}
	return "0"
}

func appendStringRangeDecode(out *[]byte, seqName string, indexName string, widthName string, byteName string, runeName string, indent string) {
	appendStringRef(out, indent)
	appendStringRef(out, "if ")
	appendStringRef(out, byteName)
	appendStringRef(out, " >= 240 && ")
	appendStringRef(out, indexName)
	appendStringRef(out, " + 3 < len(")
	appendStringRef(out, seqName)
	appendStringRef(out, ") {\n")
	appendStringRangeDecodeBody(out, seqName, indexName, widthName, byteName, runeName, indent+"\t", "4", "7", "18", "12", "6")
	appendStringRef(out, indent)
	appendStringRef(out, "} else if ")
	appendStringRef(out, byteName)
	appendStringRef(out, " >= 224 && ")
	appendStringRef(out, indexName)
	appendStringRef(out, " + 2 < len(")
	appendStringRef(out, seqName)
	appendStringRef(out, ") {\n")
	appendStringRangeDecodeBody(out, seqName, indexName, widthName, byteName, runeName, indent+"\t", "3", "15", "12", "6", "0")
	appendStringRef(out, indent)
	appendStringRef(out, "} else if ")
	appendStringRef(out, byteName)
	appendStringRef(out, " >= 192 && ")
	appendStringRef(out, indexName)
	appendStringRef(out, " + 1 < len(")
	appendStringRef(out, seqName)
	appendStringRef(out, ") {\n")
	appendStringRangeDecodeBody(out, seqName, indexName, widthName, byteName, runeName, indent+"\t", "2", "31", "6", "0", "")
	appendStringRef(out, indent)
	appendStringRef(out, "} else if ")
	appendStringRef(out, byteName)
	appendStringRef(out, " >= 128 {\n")
	if runeName != "" {
		appendStringRef(out, indent)
		appendStringRef(out, "\t")
		appendStringRef(out, runeName)
		appendStringRef(out, " = 65533\n")
	}
	appendStringRef(out, indent)
	appendStringRef(out, "}\n")
}

func appendStringRangeDecodeBody(out *[]byte, seqName string, indexName string, widthName string, byteName string, runeName string, indent string, width string, firstMask string, firstShift string, secondShift string, thirdShift string) {
	appendStringRef(out, indent)
	appendStringRef(out, widthName)
	appendStringRef(out, " = ")
	appendStringRef(out, width)
	appendByteRef(out, '\n')
	if runeName == "" {
		return
	}
	appendStringRef(out, indent)
	appendStringRef(out, runeName)
	appendStringRef(out, " = int32(((")
	appendStringRef(out, byteName)
	appendStringRef(out, " & ")
	appendStringRef(out, firstMask)
	appendStringRef(out, ") << ")
	appendStringRef(out, firstShift)
	appendStringRef(out, ")")
	appendStringRangeContinuation(out, seqName, indexName, "1", secondShift)
	if thirdShift != "" {
		appendStringRangeContinuation(out, seqName, indexName, "2", thirdShift)
	}
	if width == "4" {
		appendStringRangeContinuation(out, seqName, indexName, "3", "0")
	}
	appendStringRef(out, ")\n")
}

func appendStringRangeContinuation(out *[]byte, seqName string, indexName string, offset string, shift string) {
	appendStringRef(out, " | ((int(")
	appendStringRef(out, seqName)
	appendByteRef(out, '[')
	appendStringRef(out, indexName)
	appendStringRef(out, " + ")
	appendStringRef(out, offset)
	appendStringRef(out, "]) & 63)")
	if shift != "0" && shift != "" {
		appendStringRef(out, " << ")
		appendStringRef(out, shift)
	}
	appendByteRef(out, ')')
}

func rangeStatementOperandIsString(body string, toks []scan.Token, stmt rangeStatement, ctx normalizationContext) bool {
	return copySourceIsStringExpression(toks, expressionRange{start: stmt.exprStart, end: stmt.exprEnd}, ctx.localTypes, ctx.functionResults, ctx.namedTypes, ctx.fieldTypes)
}

func rangeStatementNeedsValue(toks []scan.Token, stmt rangeStatement) bool {
	return len(stmt.targets) > 1 && !rangeTargetIsBlank(toks, stmt.targets[1])
}

func normalizeRangeOperand(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, string) {
	if temps, expr, ok := normalizeNamedSliceConversionRangeOperand(body, toks, start, end, unitName, tempIndex, namedSlices, namedConversions, ctx); ok {
		return temps, expr
	}
	if temps, expr, ok := normalizeSelectorRangeOperand(body, toks, start, end, unitName, tempIndex, namedSlices, namedConversions, ctx); ok {
		return temps, expr
	}
	exprStart := int(toks[start].Start)
	exprEnd := int(toks[end-1].End)
	temps, replacements := normalizeExpressionWithContext(body, toks, start, end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	expr := strings.TrimSpace(applyExpressionReplacements(body, exprStart, exprEnd, replacements))
	return temps, expr
}

func normalizeNamedSliceConversionRangeOperand(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, string, bool) {
	start, end = trimOuterParens(toks, start, end)
	if start+3 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" || toks[end-1].Text != ")" {
		return nil, "", false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return nil, "", false
	}
	return normalizeNamedConversionExpression(body, toks, start, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
}

func normalizeNamedSliceConversionCall(body string, toks []scan.Token, pos int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	temps, expr, ok := normalizeNamedConversionExpression(body, toks, pos, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
	if !ok {
		return nil, expressionReplacement{}, false
	}
	repl := expressionReplacement{start: int(toks[pos].Start), end: int(toks[close].End), text: expr}
	return temps, repl, true
}

func normalizeNamedConversionExpression(body string, toks []scan.Token, pos int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, string, bool) {
	if pos+2 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" || close >= len(toks) || toks[close].Text != ")" {
		return nil, "", false
	}
	if !isNamedSliceTypeName(toks[pos].Text, namedSlices) && !containsString(namedConversions, toks[pos].Text) {
		return nil, "", false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 1 {
		return nil, "", false
	}
	arg := args[0]
	if arg.start >= arg.end {
		return nil, "", false
	}
	if containsString(namedConversions, toks[pos].Text) {
		temps, expr, ok := normalizeCompositeLiteralConversion(body, toks, arg.start, arg.end, toks[pos].Text, unitName, tempIndex, namedSlices, namedConversions, ctx)
		if ok {
			return temps, expr, true
		}
	}
	argStart := int(toks[arg.start].Start)
	argEnd := int(toks[arg.end-1].End)
	temps, replacements := normalizeExpressionWithContext(body, toks, arg.start, arg.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	expr := strings.TrimSpace(applyExpressionReplacements(body, argStart, argEnd, replacements))
	return temps, expr, true
}

func normalizeCompositeLiteralConversion(body string, toks []scan.Token, start int, end int, conversionName string, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, string, bool) {
	start, end = trimOuterParens(toks, start, end)
	if !tokensAreCompositeLiteral(toks, start, end) {
		return nil, "", false
	}
	open := findTopLevelToken(toks, start, end, "{")
	if open < 0 {
		return nil, "", false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, "", false
	}
	valueTemps, valueReplacements := normalizeCompositeLiteralValues(body, toks, open, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
	literalBody := applyExpressionReplacements(body, int(toks[open].Start), int(toks[close].End), valueReplacements)
	return valueTemps, conversionName + strings.TrimSpace(literalBody), true
}

func normalizeSelectorRangeOperand(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, string, bool) {
	start, end = trimOuterParens(toks, start, end)
	dot := lastTopLevelSelectorDot(toks, start, end)
	if dot < 0 || dot+2 != end || toks[dot+1].Kind != scan.Ident {
		return nil, "", false
	}
	if simpleSelectorBase(toks, start, dot) {
		return nil, "", false
	}
	baseTemps, baseExpr := normalizeRangeOperand(body, toks, start, dot, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps := appendExpressionTemps(nil, baseTemps)
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(baseExpr)})
	return temps, name + "." + toks[dot+1].Text, true
}

func isNamedSliceTypeName(name string, namedSlices []namedSliceInfo) bool {
	for i := 0; i < len(namedSlices); i++ {
		if namedSlices[i].name == name {
			return true
		}
	}
	return false
}

func lastTopLevelSelectorDot(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := end - 1; i >= start; i-- {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 && text == "." {
			return i
		}
		switch text {
		case ")":
			paren++
		case "(":
			if paren > 0 {
				paren--
			}
		case "]":
			brack++
		case "[":
			if brack > 0 {
				brack--
			}
		case "}":
			brace++
		case "{":
			if brace > 0 {
				brace--
			}
		}
	}
	return -1
}

func simpleSelectorBase(toks []scan.Token, start int, end int) bool {
	start, end = trimOuterParens(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return false
	}
	expectIdent := true
	for i := start; i < end; i++ {
		if expectIdent {
			if toks[i].Kind != scan.Ident {
				return false
			}
			expectIdent = false
			continue
		}
		if toks[i].Text != "." {
			return false
		}
		expectIdent = true
	}
	return !expectIdent
}

func trimOuterParens(toks []scan.Token, start int, end int) (int, int) {
	for start < end && toks[start].Text == "(" && toks[end-1].Text == ")" {
		close := findClose(toks, start, "(", ")")
		if close != end-1 {
			break
		}
		start++
		end--
	}
	return start, end
}

func rangeLoopIndexName(body string, unitName string, tempIndex *int, stmt rangeStatement) string {
	if stmt.define && len(stmt.names) > 0 && stmt.names[0] != "_" {
		return stmt.names[0]
	}
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	return name
}

func appendRangeBindings(out *[]byte, body string, toks []scan.Token, stmt rangeStatement, seqName string, indexName string, valueExpr string, loopIndent string) {
	if len(stmt.targets) == 0 {
		appendByteRef(out, '\n')
		return
	}
	if !stmt.define && !rangeTargetIsBlank(toks, stmt.targets[0]) {
		appendByteRef(out, '\n')
		appendStringRef(out, loopIndent)
		appendRangeTargetSource(out, body, toks, stmt.targets[0])
		appendStringRef(out, " = ")
		appendStringRef(out, indexName)
	}
	if len(stmt.targets) > 1 && !rangeTargetIsBlank(toks, stmt.targets[1]) {
		appendByteRef(out, '\n')
		appendStringRef(out, loopIndent)
		if stmt.define {
			appendStringRef(out, stmt.names[1])
		} else {
			appendRangeTargetSource(out, body, toks, stmt.targets[1])
		}
		if stmt.define {
			appendStringRef(out, " := ")
		} else {
			appendStringRef(out, " = ")
		}
		appendStringRef(out, valueExpr)
	}
	appendByteRef(out, '\n')
}

func rangeTargetIsBlank(toks []scan.Token, expr expressionRange) bool {
	return expr.start+1 == expr.end && toks[expr.start].Text == "_"
}

func appendRangeTargetSource(out *[]byte, body string, toks []scan.Token, expr expressionRange) {
	if expr.start >= expr.end {
		return
	}
	*out = appendStringRange(*out, body, int(toks[expr.start].Start), int(toks[expr.end-1].End))
}

func normalizationRangeStatement(toks []scan.Token, pos int) (rangeStatement, bool) {
	if toks[pos].Text != "for" {
		return rangeStatement{}, false
	}
	openBrace := topLevelTokenBeforeStatementBody(toks, pos+1, "{")
	if openBrace < 0 {
		return rangeStatement{}, false
	}
	rangeTok := topLevelTokenBefore(toks, pos+1, openBrace, "range")
	if rangeTok < 0 {
		return rangeStatement{}, false
	}
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace < 0 {
		return rangeStatement{}, false
	}
	stmt := rangeStatement{
		token:      pos,
		rangeToken: rangeTok,
		exprStart:  rangeTok + 1,
		exprEnd:    openBrace,
		openBrace:  openBrace,
		end:        closeBrace,
	}
	if stmt.exprStart >= stmt.exprEnd {
		return rangeStatement{}, false
	}
	if rangeTok == pos+1 {
		return stmt, true
	}
	assign := rangeTok - 1
	if assign <= pos || (toks[assign].Text != ":=" && toks[assign].Text != "=") {
		return rangeStatement{}, false
	}
	stmt.assign = assign
	stmt.define = toks[assign].Text == ":="
	stmt.lhsStart = pos + 1
	stmt.lhsEnd = assign
	targets, names, ok := rangeStatementTargets(toks, stmt.lhsStart, stmt.lhsEnd, stmt.define)
	if !ok {
		return rangeStatement{}, false
	}
	stmt.targets = targets
	stmt.names = names
	return stmt, true
}

func rangeStatementTargets(toks []scan.Token, start int, end int, define bool) ([]expressionRange, []string, bool) {
	ranges := topLevelExpressionRanges(toks, start, end)
	if len(ranges) > 2 {
		return nil, nil, false
	}
	names := make([]string, 0, len(ranges))
	for i := 0; i < len(ranges); i++ {
		expr := ranges[i]
		name := ""
		if expr.start+1 == expr.end && toks[expr.start].Kind == scan.Ident {
			name = toks[expr.start].Text
		}
		if define && name == "" {
			return nil, nil, false
		}
		names = append(names, name)
	}
	return ranges, names, true
}

func topLevelTokenBeforeStatementBody(toks []scan.Token, start int, text string) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < len(toks); i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == text {
				if text == "{" && rangeHeaderCompositeLiteralOpen(toks, i) {
					close := findClose(toks, i, "{", "}")
					if close < 0 {
						return -1
					}
					i = close
					continue
				}
				return i
			}
			if toks[i].Text == ";" {
				return -1
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func rangeHeaderCompositeLiteralOpen(toks []scan.Token, open int) bool {
	if open <= 0 || toks[open].Text != "{" {
		return false
	}
	prev := open - 1
	if toks[prev].Kind == scan.Ident && (prev == 0 || toks[prev-1].Text != "]") {
		return headerCompositeLiteralOpenFollowedByBlock(toks, open)
	}
	brackClose := prev
	if toks[brackClose].Kind == scan.Ident && brackClose > 0 {
		brackClose--
	}
	if toks[brackClose].Text != "]" {
		return false
	}
	brackOpen := findOpen(toks, brackClose, "[", "]")
	if brackOpen < 0 {
		return false
	}
	if brackOpen > 0 && toks[brackOpen-1].Text == "map" {
		return true
	}
	return brackOpen+1 == brackClose
}

func headerCompositeLiteralOpenFollowedByBlock(toks []scan.Token, open int) bool {
	close := findClose(toks, open, "{", "}")
	if close < 0 || close+1 >= len(toks) {
		return false
	}
	return toks[close+1].Text == "{"
}

func topLevelTokenBefore(toks []scan.Token, start int, end int, text string) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == text {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func trimmedBytesHavePrefix(body []byte, prefix string) bool {
	start := 0
	for start < len(body) && (body[start] == ' ' || body[start] == '\t' || body[start] == '\r' || body[start] == '\n') {
		start++
	}
	if start+len(prefix) > len(body) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if body[start+i] != prefix[i] {
			return false
		}
	}
	return true
}

func tokenSpansMatchSource(body string, toks []scan.Token) bool {
	for i := 0; i < len(toks); i++ {
		tok := toks[i]
		if int(tok.Start) < 0 || int(tok.End) < int(tok.Start) || int(tok.End) > len(body) {
			return false
		}
		part := body[int(tok.Start):int(tok.End)]
		if part != tok.Text {
			return false
		}
	}
	return true
}

func normalizationConditionalShortStatement(toks []scan.Token, pos int) (conditionalShortStatement, bool) {
	if toks[pos].Text != "if" && toks[pos].Text != "switch" {
		return conditionalShortStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return normalizationConditionalShortStatementWithCompositeInit(toks, pos)
	}
	semi := topLevelSemicolon(toks, exprStart, exprEnd)
	if semi < 0 || semi <= exprStart || semi+1 >= exprEnd {
		return normalizationConditionalShortStatementWithCompositeInit(toks, pos)
	}
	end := conditionalStatementEnd(toks, pos, exprEnd)
	if end <= exprEnd {
		return conditionalShortStatement{}, false
	}
	posTok := toks[pos]
	var stmt conditionalShortStatement
	stmt.token = pos
	stmt.initStart = exprStart
	stmt.semi = semi
	stmt.condStart = semi + 1
	stmt.condEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = end
	stmt.kind = posTok.Text
	return stmt, true
}

func normalizationConditionalShortStatementWithCompositeInit(toks []scan.Token, pos int) (conditionalShortStatement, bool) {
	if toks[pos].Text != "if" && toks[pos].Text != "switch" {
		return conditionalShortStatement{}, false
	}
	exprStart := pos + 1
	semi, openBrace, ok := conditionalShortHeaderWithCompositeInit(toks, exprStart)
	if !ok || semi <= exprStart || semi+1 >= openBrace {
		return conditionalShortStatement{}, false
	}
	end := conditionalStatementEnd(toks, pos, openBrace)
	if end <= openBrace {
		return conditionalShortStatement{}, false
	}
	return conditionalShortStatement{
		token:     pos,
		initStart: exprStart,
		semi:      semi,
		condStart: semi + 1,
		condEnd:   openBrace,
		openBrace: openBrace,
		end:       end,
		kind:      toks[pos].Text,
	}, true
}

func conditionalShortHeaderWithCompositeInit(toks []scan.Token, start int) (int, int, bool) {
	semi := -1
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < len(toks); i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == ";" {
				semi = i
				continue
			}
			if toks[i].Text == "{" && conditionalHeaderCompositeLiteralOpen(toks, i) {
				close := findClose(toks, i, "{", "}")
				if close < 0 {
					return -1, -1, false
				}
				i = close
				continue
			}
			if toks[i].Text == "{" && semi < 0 {
				return -1, -1, false
			}
			if toks[i].Text == "{" {
				return semi, i, true
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1, -1, false
}

func conditionalHeaderCompositeLiteralOpen(toks []scan.Token, open int) bool {
	if headerCompositeLiteralOpen(toks, open) || rangeHeaderCompositeLiteralOpen(toks, open) {
		return true
	}
	close := findClose(toks, open, "{", "}")
	if close < 0 || close+1 >= len(toks) {
		return false
	}
	switch toks[close+1].Text {
	case "[", ".", "(", ";", ",", ")", "==", "!=", "<", "<=", ">", ">=", "+", "-", "*", "/", "%", "&&", "||":
		return true
	}
	return false
}

func normalizationShortCircuitIfStatement(toks []scan.Token, pos int) (shortCircuitIfStatement, bool) {
	if toks[pos].Text != "if" {
		return shortCircuitIfStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return shortCircuitIfStatement{}, false
	}
	if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
		return shortCircuitIfStatement{}, false
	}
	closeBrace := findClose(toks, exprEnd, "{", "}")
	if closeBrace <= exprEnd {
		return shortCircuitIfStatement{}, false
	}
	if closeBrace+1 < len(toks) && toks[closeBrace+1].Text == "else" {
		return shortCircuitIfStatement{}, false
	}
	operands, ok := topLevelAndOperands(toks, exprStart, exprEnd)
	if !ok || len(operands) < 2 {
		return shortCircuitIfStatement{}, false
	}
	var stmt shortCircuitIfStatement
	stmt.token = pos
	stmt.condStart = exprStart
	stmt.condEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = closeBrace
	stmt.operands = operands
	return stmt, true
}

func normalizationClassicForPostStatement(toks []scan.Token, pos int) (classicForPostStatement, bool) {
	if toks[pos].Text != "for" {
		return classicForPostStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return classicForPostStatement{}, false
	}
	firstSemi := topLevelSemicolon(toks, exprStart, exprEnd)
	if firstSemi < 0 {
		return classicForPostStatement{}, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, exprEnd)
	if secondSemi < 0 || secondSemi+1 >= exprEnd {
		return classicForPostStatement{}, false
	}
	end := findClose(toks, exprEnd, "{", "}")
	if end <= exprEnd || containsTokenText(toks, exprEnd+1, end, "continue") {
		return classicForPostStatement{}, false
	}
	var stmt classicForPostStatement
	stmt.token = pos
	stmt.initStart = exprStart
	stmt.initEnd = firstSemi
	stmt.condStart = firstSemi + 1
	stmt.condEnd = secondSemi
	stmt.postStart = secondSemi + 1
	stmt.postEnd = exprEnd
	stmt.openBrace = exprEnd
	stmt.end = end
	return stmt, true
}

func normalizationClassicForConditionStatement(toks []scan.Token, pos int) (classicForConditionStatement, bool) {
	if toks[pos].Text != "for" {
		return classicForConditionStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := conditionExpressionEnd(toks, pos)
	if exprEnd <= exprStart || exprEnd >= len(toks) || toks[exprEnd].Text != "{" {
		return classicForConditionStatement{}, false
	}
	firstSemi := topLevelSemicolon(toks, exprStart, exprEnd)
	if firstSemi < 0 {
		return classicForConditionStatement{}, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, exprEnd)
	if secondSemi < 0 || secondSemi <= firstSemi+1 {
		return classicForConditionStatement{}, false
	}
	var stmt classicForConditionStatement
	stmt.token = pos
	stmt.condStart = firstSemi + 1
	stmt.condEnd = secondSemi
	stmt.openBrace = exprEnd
	return stmt, true
}

func normalizationCompoundAssignmentStatement(toks []scan.Token, pos int) (compoundAssignmentStatement, bool) {
	if !isPlusAssignmentAt(toks, pos) {
		return compoundAssignmentStatement{}, false
	}
	if isInsideClassicForHeader(toks, pos) || isInsideConditionalShortHeader(toks, pos) {
		return compoundAssignmentStatement{}, false
	}
	rhsStart := pos + 2
	rhsEnd := lineExpressionEnd(toks, pos+1)
	if rhsEnd <= rhsStart {
		return compoundAssignmentStatement{}, false
	}
	stmtStart := statementStartToken(toks, pos)
	if stmtStart >= pos || (stmtStart < len(toks) && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch")) {
		return compoundAssignmentStatement{}, false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, pos)
	if len(lhs) != 1 {
		return compoundAssignmentStatement{}, false
	}
	lhsStart, lhsEnd := trimTokenRange(toks, lhs[0].start, lhs[0].end)
	if !lowerableCompoundAssignmentTarget(toks, lhsStart, lhsEnd) {
		return compoundAssignmentStatement{}, false
	}
	return compoundAssignmentStatement{
		token:    stmtStart,
		assign:   pos,
		lhsStart: lhsStart,
		lhsEnd:   lhsEnd,
		rhsStart: rhsStart,
		rhsEnd:   rhsEnd,
	}, true
}

func isPlusAssignmentAt(toks []scan.Token, pos int) bool {
	return pos+1 < len(toks) && toks[pos].Text == "+" && toks[pos+1].Text == "=" && int(toks[pos].End) == int(toks[pos+1].Start)
}

func lowerableCompoundAssignmentTarget(toks []scan.Token, start int, end int) bool {
	start, end = trimOuterParens(toks, start, end)
	if start >= end || expressionContainsCall(toks, start, end) {
		return false
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		text := toks[start].Text
		return text != "true" && text != "false" && text != "nil"
	}
	if toks[start].Text == "*" && start+1 < end {
		return lowerableCompoundAssignmentTarget(toks, start+1, end)
	}
	if end > start && toks[end-1].Text == "]" {
		open := findOpen(toks, end-1, "[", "]")
		if open > start && lowerableCompoundAssignmentTarget(toks, start, open) {
			return findTopLevelToken(toks, open+1, end-1, ":") < 0
		}
	}
	if end >= start+3 && toks[end-2].Text == "." && toks[end-1].Kind == scan.Ident {
		return lowerableCompoundAssignmentTarget(toks, start, end-2)
	}
	return false
}

func normalizationStatement(toks []scan.Token, pos int) (expressionStatement, bool) {
	if toks[pos].Text == "return" {
		exprStart := pos + 1
		exprEnd := lineExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd, returnStatement: true}, true
	}
	if toks[pos].Text == "if" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := shortHeaderInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "switch" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := shortHeaderInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "for" {
		exprStart := pos + 1
		exprEnd := conditionExpressionEnd(toks, pos)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		if expressionContainsTopLevelSemicolon(toks, exprStart, exprEnd) {
			assign, semi := classicForInitAssignment(toks, exprStart, exprEnd)
			if assign < 0 || semi <= assign+1 {
				return expressionStatement{}, false
			}
			return expressionStatement{token: pos, exprStart: assign + 1, exprEnd: semi}, true
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd, openBrace: exprEnd, forCondition: true}, true
	}
	if isInsideClassicForHeader(toks, pos) {
		return expressionStatement{}, false
	}
	if isInsideConditionalShortHeader(toks, pos) {
		return expressionStatement{}, false
	}
	if startsCallStatement(toks, pos) {
		exprEnd := lineExpressionEnd(toks, pos)
		if exprEnd <= pos {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: pos, exprEnd: exprEnd}, true
	}
	if toks[pos].Text == "var" {
		exprStart := varInitializerStart(toks, pos)
		if exprStart < 0 {
			return expressionStatement{}, false
		}
		exprEnd := lineExpressionEnd(toks, exprStart-1)
		if exprEnd <= exprStart {
			return expressionStatement{}, false
		}
		return expressionStatement{token: pos, exprStart: exprStart, exprEnd: exprEnd}, true
	}
	if !isAssignmentOperator(toks[pos].Text) {
		return expressionStatement{}, false
	}
	if isClassicForHeaderAssignment(toks, pos) {
		return expressionStatement{}, false
	}
	exprStart := pos + 1
	exprEnd := lineExpressionEnd(toks, pos)
	if exprEnd <= exprStart {
		return expressionStatement{}, false
	}
	stmtStart := statementStartToken(toks, pos)
	return expressionStatement{token: stmtStart, exprStart: exprStart, exprEnd: exprEnd}, true
}

func conditionExpressionEnd(toks []scan.Token, start int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "{" {
			if headerCompositeLiteralOpen(toks, i) {
				close := findClose(toks, i, "{", "}")
				if close < 0 {
					return i
				}
				i = close
				continue
			}
			return i
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func headerCompositeLiteralOpen(toks []scan.Token, open int) bool {
	if open <= 0 || toks[open].Text != "{" {
		return false
	}
	typeStart := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
	if typeStart < 0 {
		return false
	}
	if toks[typeStart].Kind == scan.Ident {
		return compositeLiteralOpenFollowedByBlock(toks, open)
	}
	return true
}

func compositeLiteralOpenFollowedByBlock(toks []scan.Token, open int) bool {
	close := findClose(toks, open, "{", "}")
	if close < 0 || close+1 >= len(toks) {
		return false
	}
	return toks[close+1].Text == "{"
}

func lineExpressionEnd(toks []scan.Token, start int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start + 1; i < len(toks); i++ {
		tok := toks[i]
		if tok.Kind == scan.EOF {
			return i
		}
		if paren == 0 && brack == 0 && brace == 0 {
			if tok.Text == "}" || tok.Text == ";" {
				return i
			}
			if tok.Line != toks[start].Line {
				return i
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func statementStartToken(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}" {
			return i + 1
		}
	}
	return 0
}

func isAssignmentOperator(text string) bool {
	return text == "=" || text == ":="
}

func isClassicForHeaderAssignment(toks []scan.Token, pos int) bool {
	start := statementStartToken(toks, pos)
	if start < len(toks) && toks[start].Text == "for" {
		exprEnd := conditionExpressionEnd(toks, start)
		return expressionContainsTopLevelSemicolon(toks, start+1, exprEnd)
	}
	return isForPostClauseAssignment(toks, pos)
}

func varInitializerStart(toks []scan.Token, pos int) int {
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		return -1
	}
	paren := 0
	brack := 0
	brace := 0
	for i := pos + 1; i < len(toks) && toks[i].Line == toks[pos].Line; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "=" {
			return i + 1
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func isForPostClauseAssignment(toks []scan.Token, pos int) bool {
	semi := -1
	for i := pos - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text == ";" {
			semi = i
			break
		}
	}
	if semi < 0 {
		return false
	}
	for i := semi - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text == "for" {
			return true
		}
	}
	return false
}

func isInsideClassicForHeader(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0 && toks[i].Line == toks[pos].Line; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text != "for" {
			continue
		}
		exprEnd := conditionExpressionEnd(toks, i)
		return pos < exprEnd && expressionContainsTopLevelSemicolon(toks, i+1, exprEnd)
	}
	return false
}

func classicForInitAssignment(toks []scan.Token, start int, end int) (int, int) {
	return shortHeaderInitAssignment(toks, start, end)
}

func shortHeaderInitAssignment(toks []scan.Token, start int, end int) (int, int) {
	paren := 0
	brack := 0
	brace := 0
	assign := -1
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 {
			if toks[i].Text == ";" {
				return assign, i
			}
			if assign < 0 && isAssignmentOperator(toks[i].Text) {
				assign = i
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1, -1
}

func isInsideConditionalShortHeader(toks []scan.Token, pos int) bool {
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
		if toks[i].Text != "if" && toks[i].Text != "switch" {
			continue
		}
		exprEnd := conditionExpressionEnd(toks, i)
		return pos < exprEnd && expressionContainsTopLevelSemicolon(toks, i+1, exprEnd)
	}
	return false
}

func startsCallStatement(toks []scan.Token, pos int) bool {
	if pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return false
	}
	return statementStartToken(toks, pos) == pos
}

func normalizeExpression(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string) ([]expressionTemp, []expressionReplacement) {
	return normalizeExpressionWithContext(body, toks, start, end, unitName, tempIndex, namedSlices, namedConversions, normalizationContext{})
}

func normalizeExpressionWithContext(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	paren := 0
	brack := 0
	brace := 0
	for i := start; i+1 < end; i++ {
		tok := toks[i]
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "(" && i > start && toks[i-1].Text == "*" {
			close := findClose(toks, i, "(", ")")
			if close > i && close < end {
				innerTemps, innerReplacements := normalizeExpressionWithContext(body, toks, i+1, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
				temps = appendExpressionTemps(temps, innerTemps)
				replacements = appendExpressionReplacements(replacements, innerReplacements)
				i = close
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "&" {
			addrTemps, addrReplacement, addrClose, addrOK := normalizeAddressOfCompositeLiteralSelector(body, toks, i, end, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if addrOK {
				temps = appendExpressionTemps(temps, addrTemps)
				replacements = append(replacements, addrReplacement)
				i = addrClose
				continue
			}
			addrTemps, addrReplacement, addrClose, addrOK = normalizeAddressOfCompositeLiteral(body, toks, i, end, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if addrOK {
				temps = appendExpressionTemps(temps, addrTemps)
				replacements = append(replacements, addrReplacement)
				i = addrClose
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && (tok.Text == "map" || tok.Text == "make" || tok.Text == "(") {
			mapTemps, mapReplacement, mapClose, mapOK := normalizeMapLiteralIndexExpression(body, toks, i, end, unitName, tempIndex)
			if mapOK {
				temps = appendExpressionTemps(temps, mapTemps)
				replacements = append(replacements, mapReplacement)
				i = mapClose
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Kind == scan.Ident && toks[i+1].Text == "(" {
			close := findClose(toks, i+1, "(", ")")
			if close > i+1 && close < end {
				conversionTemps, conversionReplacement, conversionOK := normalizeNamedSliceConversionCall(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
				if conversionOK {
					temps = appendExpressionTemps(temps, conversionTemps)
					replacements = append(replacements, conversionReplacement)
					i = close
					continue
				}
				if tok.Text == "append" {
					appendTemps, appendReplacement, appendOK := normalizeAppendExpansionCall(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
					if appendOK {
						temps = appendExpressionTemps(temps, appendTemps)
						replacements = append(replacements, appendReplacement)
						i = close
						continue
					}
					appendTemps, appendReplacement, appendOK = normalizeAppendMultiValueCall(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
					if appendOK {
						temps = appendExpressionTemps(temps, appendTemps)
						replacements = append(replacements, appendReplacement)
						i = close
						continue
					}
				}
				if tok.Text == "new" {
					newTemp, newReplacement, newOK := normalizeNewValueCall(body, toks, i, close, unitName, tempIndex, ctx)
					if newOK {
						if newTemp.name != "" {
							temps = append(temps, newTemp)
						}
						replacements = append(replacements, newReplacement)
						i = close
						continue
					}
				}
				if tok.Text == "real" || tok.Text == "imag" {
					complexTemps, complexReplacement, complexOK := normalizeReducibleComplexComponentCall(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
					if complexOK {
						temps = appendExpressionTemps(temps, complexTemps)
						replacements = append(replacements, complexReplacement)
						i = close
						continue
					}
				}
				if tok.Text == "len" {
					mapTemps, mapReplacement, mapOK := normalizeMapLiteralLenCall(body, toks, i, close, unitName, tempIndex)
					if mapOK {
						temps = appendExpressionTemps(temps, mapTemps)
						replacements = append(replacements, mapReplacement)
						i = close
						continue
					}
				}
				callTemps, callReplacements := normalizeOneCallArguments(body, toks, i+2, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
				temps = appendExpressionTemps(temps, callTemps)
				replacements = appendExpressionReplacements(replacements, callReplacements)
				i = close
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "[" {
			sliceTemps, sliceReplacement, sliceClose, sliceOK := normalizeUnnamedSliceConversionCall(body, toks, i, end, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if sliceOK {
				temps = appendExpressionTemps(temps, sliceTemps)
				replacements = append(replacements, sliceReplacement)
				i = sliceClose
				continue
			}
			close := findClose(toks, i, "[", "]")
			if close > i && close < end {
				baseTemps, baseReplacement, baseOK := normalizeCompositeLiteralIndexBase(body, toks, i, unitName, tempIndex, namedSlices, namedConversions, ctx)
				if baseOK {
					temps = appendExpressionTemps(temps, baseTemps)
					replacements = append(replacements, baseReplacement)
				}
				indexTemps, indexReplacements := normalizeIndexBounds(body, toks, i+1, close, unitName, tempIndex)
				temps = appendExpressionTemps(temps, indexTemps)
				replacements = appendExpressionReplacements(replacements, indexReplacements)
				i = close
				continue
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "." {
			baseTemps, baseReplacement, baseOK := normalizeCompositeLiteralSelectorBase(body, toks, i, unitName, tempIndex, namedSlices, namedConversions, ctx)
			if baseOK {
				temps = appendExpressionTemps(temps, baseTemps)
				replacements = append(replacements, baseReplacement)
			}
		}
		if paren == 0 && brack == 0 && brace == 0 && tok.Text == "{" {
			close := findClose(toks, i, "{", "}")
			if close > i && close < end {
				if compositeLiteralOpenHasDirectIndex(toks, i, close, end) || compositeLiteralOpenHasDirectSelector(toks, i, close, end) {
					i = close
					continue
				}
				keyedTemps, keyedReplacement, keyedOK := normalizeKeyedSliceCompositeLiteral(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
				if keyedOK {
					temps = appendExpressionTemps(temps, keyedTemps)
					replacements = append(replacements, keyedReplacement)
					i = close
					continue
				}
				valueTemps, valueReplacements := normalizeCompositeLiteralValues(body, toks, i, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
				temps = appendExpressionTemps(temps, valueTemps)
				replacements = appendExpressionReplacements(replacements, valueReplacements)
				i = close
				continue
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	return temps, replacements
}

func normalizeUnnamedSliceConversionCall(body string, toks []scan.Token, pos int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, int, bool) {
	if pos+4 >= end || toks[pos].Text != "[" || toks[pos+1].Text != "]" {
		return nil, expressionReplacement{}, -1, false
	}
	typeEnd := sliceTypeEnd(toks, pos+2, end)
	if typeEnd <= pos+2 || typeEnd >= end || toks[typeEnd].Text != "(" {
		return nil, expressionReplacement{}, -1, false
	}
	close := findClose(toks, typeEnd, "(", ")")
	if close <= typeEnd || close >= end {
		return nil, expressionReplacement{}, -1, false
	}
	args := topLevelExpressionRanges(toks, typeEnd+1, close)
	if len(args) != 1 || !expressionStartsKnownNamedSlice(toks, args[0], namedSlices) {
		return nil, expressionReplacement{}, -1, false
	}
	arg := args[0]
	argStart := int(toks[arg.start].Start)
	argEnd := int(toks[arg.end-1].End)
	temps, replacements := normalizeExpressionWithContext(body, toks, arg.start, arg.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	expr := strings.TrimSpace(applyExpressionReplacements(body, argStart, argEnd, replacements))
	repl := expressionReplacement{start: int(toks[pos].Start), end: int(toks[close].End), text: expr}
	return temps, repl, close, true
}

func sliceTypeEnd(toks []scan.Token, start int, end int) int {
	if start >= end {
		return start
	}
	if toks[start].Text == "*" {
		start++
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return start
	}
	if start+2 < end && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		return start + 3
	}
	return start + 1
}

func expressionStartsKnownNamedSlice(toks []scan.Token, expr expressionRange, namedSlices []namedSliceInfo) bool {
	start, end := trimTokenRange(toks, expr.start, expr.end)
	start, end = trimOuterParens(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident || !isNamedSliceTypeName(toks[start].Text, namedSlices) {
		return false
	}
	if start+1 >= end {
		return true
	}
	return toks[start+1].Text == "{" || toks[start+1].Text == "("
}

func normalizeCompositeLiteralSelectorBase(body string, toks []scan.Token, dot int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	replaceStart, replaceEnd, exprStart, exprEnd, ok := compositeLiteralSelectorBaseRanges(toks, dot)
	if !ok {
		return nil, expressionReplacement{}, false
	}
	var temps []expressionTemp
	valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, exprStart, exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, valueTemps)
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	expr := applyExpressionReplacements(body, int(toks[exprStart].Start), int(toks[exprEnd-1].End), valueReplacements)
	temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(expr)})
	repl := expressionReplacement{
		start: int(toks[replaceStart].Start),
		end:   int(toks[replaceEnd-1].End),
		text:  name,
	}
	return temps, repl, true
}

func compositeLiteralSelectorBaseRanges(toks []scan.Token, dot int) (int, int, int, int, bool) {
	if dot <= 0 || dot+1 >= len(toks) || toks[dot].Text != "." || toks[dot+1].Kind != scan.Ident {
		return 0, 0, 0, 0, false
	}
	before := dot - 1
	if toks[before].Text == ")" {
		parenOpen := findOpen(toks, before, "(", ")")
		if parenOpen < 0 {
			return 0, 0, 0, 0, false
		}
		innerStart := parenOpen + 1
		innerEnd := before
		if tokensAreAddressOfCompositeLiteral(toks, innerStart, innerEnd) {
			return parenOpen, dot, innerStart + 1, innerEnd, true
		}
		if !tokensAreCompositeLiteral(toks, innerStart, innerEnd) {
			return 0, 0, 0, 0, false
		}
		return parenOpen, dot, innerStart, innerEnd, true
	}
	if toks[before].Text != "}" {
		return 0, 0, 0, 0, false
	}
	open := findOpen(toks, before, "{", "}")
	if open < 0 {
		return 0, 0, 0, 0, false
	}
	start := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
	if start < 0 {
		return 0, 0, 0, 0, false
	}
	return start, dot, start, dot, true
}

func normalizeCompositeLiteralIndexBase(body string, toks []scan.Token, indexOpen int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	replaceStart, replaceEnd, exprStart, exprEnd, ok := compositeLiteralIndexBaseRanges(toks, indexOpen)
	if !ok {
		return nil, expressionReplacement{}, false
	}
	var temps []expressionTemp
	valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, exprStart, exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, valueTemps)
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	expr := applyExpressionReplacements(body, int(toks[exprStart].Start), int(toks[exprEnd-1].End), valueReplacements)
	temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(expr)})
	repl := expressionReplacement{
		start: int(toks[replaceStart].Start),
		end:   int(toks[replaceEnd-1].End),
		text:  name,
	}
	return temps, repl, true
}

func compositeLiteralIndexBaseRanges(toks []scan.Token, indexOpen int) (int, int, int, int, bool) {
	if indexOpen <= 0 || toks[indexOpen].Text != "[" {
		return 0, 0, 0, 0, false
	}
	before := indexOpen - 1
	if toks[before].Text == ")" {
		parenOpen := findOpen(toks, before, "(", ")")
		if parenOpen < 0 {
			return 0, 0, 0, 0, false
		}
		innerStart := parenOpen + 1
		innerEnd := before
		if !tokensAreCompositeLiteral(toks, innerStart, innerEnd) {
			return 0, 0, 0, 0, false
		}
		return parenOpen, indexOpen, innerStart, innerEnd, true
	}
	if toks[before].Text != "}" {
		return 0, 0, 0, 0, false
	}
	open := findOpen(toks, before, "{", "}")
	if open < 0 {
		return 0, 0, 0, 0, false
	}
	start := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
	if start < 0 {
		return 0, 0, 0, 0, false
	}
	return start, indexOpen, start, indexOpen, true
}

func tokensAreCompositeLiteral(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[end-1].Text != "}" {
		return false
	}
	open := findOpen(toks, end-1, "{", "}")
	if open < start {
		return false
	}
	typeStart := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
	return typeStart == start
}

func tokensAreAddressOfCompositeLiteral(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "&" {
		return false
	}
	return tokensAreCompositeLiteral(toks, start+1, end)
}

func explicitCompositeLiteralTypeStartBeforeOpen(toks []scan.Token, open int) int {
	if open <= 0 {
		return -1
	}
	start := open - 1
	if toks[start].Kind != scan.Ident {
		return -1
	}
	if start >= 2 && toks[start-1].Text == "." && toks[start-2].Kind == scan.Ident {
		start -= 2
	}
	if start >= 1 && toks[start-1].Text == "*" {
		start--
	}
	for start >= 2 && toks[start-2].Text == "[" && toks[start-1].Text == "]" {
		start -= 2
	}
	return start
}

func compositeLiteralOpenHasDirectIndex(toks []scan.Token, open int, close int, limit int) bool {
	if close+1 < limit && toks[close+1].Text == "[" {
		indexClose := findClose(toks, close+1, "[", "]")
		return indexClose > close+1 && indexClose < limit
	}
	if close+2 < limit && toks[close+1].Text == ")" && toks[close+2].Text == "[" {
		indexClose := findClose(toks, close+2, "[", "]")
		return indexClose > close+2 && indexClose < limit
	}
	return false
}

func compositeLiteralOpenHasDirectSelector(toks []scan.Token, open int, close int, limit int) bool {
	if close+2 < limit && toks[close+1].Text == "." && toks[close+2].Kind == scan.Ident {
		return true
	}
	if close+3 < limit && toks[close+1].Text == ")" && toks[close+2].Text == "." && toks[close+3].Kind == scan.Ident {
		return true
	}
	return false
}

func normalizeCompositeLiteralValues(body string, toks []scan.Token, open int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	values := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.start >= value.end {
			continue
		}
		valueStart := value.start
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			valueStart = colon + 1
		}
		valueStart, valueEnd := trimTokenRange(toks, valueStart, value.end)
		if valueStart >= valueEnd {
			continue
		}
		valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, valueStart, valueEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
		temps = appendExpressionTemps(temps, valueTemps)
		if expressionContainsNonConversionCall(toks, valueStart, valueEnd) {
			valueSource := applyExpressionReplacements(body, int(toks[valueStart].Start), int(toks[valueEnd-1].End), valueReplacements)
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(valueSource)})
			replacements = append(replacements, expressionReplacement{start: int(toks[valueStart].Start), end: int(toks[valueEnd-1].End), text: name})
			continue
		}
		replacements = appendExpressionReplacements(replacements, valueReplacements)
	}
	return temps, replacements
}

func normalizeKeyedSliceCompositeLiteral(body string, toks []scan.Token, open int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	elem, ok := compositeLiteralSliceElementType(toks, open, namedSlices)
	if !ok {
		return nil, expressionReplacement{}, false
	}
	values := topLevelExpressionRanges(toks, open+1, close)
	hasKey := false
	for i := 0; i < len(values); i++ {
		if findTopLevelToken(toks, values[i].start, values[i].end, ":") >= 0 {
			hasKey = true
			break
		}
	}
	if !hasKey {
		return nil, expressionReplacement{}, false
	}
	typeStart := compositeLiteralTypeStartBeforeOpen(toks, open)
	if typeStart < 0 {
		return nil, expressionReplacement{}, false
	}
	nextIndex := int64(0)
	maxIndex := int64(-1)
	var temps []expressionTemp
	var indexed []keyedSliceLiteralValue
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.start >= value.end {
			continue
		}
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		index := nextIndex
		valueStart := value.start
		if colon >= 0 {
			var keyOK bool
			index, keyOK = simpleIntegerLiteralIndex(toks, value.start, colon)
			if !keyOK || index < 0 {
				return nil, expressionReplacement{}, false
			}
			valueStart = colon + 1
		}
		for j := 0; j < len(indexed); j++ {
			if indexed[j].index == index {
				return nil, expressionReplacement{}, false
			}
		}
		valueStart, valueEnd := trimTokenRange(toks, valueStart, value.end)
		if valueStart >= valueEnd {
			return nil, expressionReplacement{}, false
		}
		valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, valueStart, valueEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
		temps = appendExpressionTemps(temps, valueTemps)
		exprStart := int(toks[valueStart].Start)
		exprEnd := int(toks[valueEnd-1].End)
		expr := strings.TrimSpace(applyExpressionReplacements(body, exprStart, exprEnd, valueReplacements))
		indexed = append(indexed, keyedSliceLiteralValue{index: index, expr: expr})
		if index > maxIndex {
			maxIndex = index
		}
		nextIndex = index + 1
	}
	if maxIndex < 0 {
		return nil, expressionReplacement{}, false
	}
	elems := make([]string, int(maxIndex)+1)
	zero := zeroValueForSliceElement(elem)
	for i := 0; i < len(elems); i++ {
		elems[i] = zero
	}
	for i := 0; i < len(indexed); i++ {
		value := indexed[i]
		elems[int(value.index)] = value.expr
	}
	var text []byte
	text = appendStringRange(text, body, int(toks[typeStart].Start), int(toks[open].Start))
	text = append(text, '{')
	for i := 0; i < len(elems); i++ {
		if i > 0 {
			text = appendString(text, ", ")
		}
		text = appendString(text, elems[i])
	}
	text = append(text, '}')
	repl := expressionReplacement{
		start: int(toks[typeStart].Start),
		end:   int(toks[close].End),
		text:  strings.TrimSpace(string(text)),
	}
	return temps, repl, true
}

type keyedSliceLiteralValue struct {
	index int64
	expr  string
}

func compositeLiteralSliceElementType(toks []scan.Token, open int, namedSlices []namedSliceInfo) (string, bool) {
	typeText := explicitCompositeLiteralTypeBeforeOpen(toks, open)
	if strings.HasPrefix(typeText, "[]") {
		return typeText[2:], true
	}
	for i := 0; i < len(namedSlices); i++ {
		if namedSlices[i].name == typeText {
			return namedSlices[i].elem, true
		}
	}
	return "", false
}

func compositeLiteralTypeStartBeforeOpen(toks []scan.Token, open int) int {
	if open <= 0 {
		return -1
	}
	start := open - 1
	if toks[start].Kind != scan.Ident {
		return -1
	}
	if start >= 2 && toks[start-1].Text == "." && toks[start-2].Kind == scan.Ident {
		start -= 2
	}
	if start >= 1 && toks[start-1].Text == "*" {
		start--
	}
	for start >= 2 && toks[start-2].Text == "[" && toks[start-1].Text == "]" {
		start -= 2
	}
	return start
}

func simpleIntegerLiteralIndex(toks []scan.Token, start int, end int) (int64, bool) {
	start, end = trimTokenRange(toks, start, end)
	sign := int64(1)
	if start < end && (toks[start].Text == "-" || toks[start].Text == "+") {
		if toks[start].Text == "-" {
			sign = -1
		}
		start++
	}
	if start+1 != end || toks[start].Kind != scan.Number {
		return 0, false
	}
	value, err := strconv.ParseInt(toks[start].Text, 0, 64)
	if err != nil {
		return 0, false
	}
	return value * sign, true
}

func zeroValueForSliceElement(elem string) string {
	if elem == "string" {
		return "\"\""
	}
	if elem == "bool" {
		return "false"
	}
	if elem == "int" || elem == "int64" || elem == "byte" || elem == "float64" || elem == "int16" || elem == "int32" {
		return "0"
	}
	if strings.HasPrefix(elem, "*") || strings.HasPrefix(elem, "[]") {
		return "nil"
	}
	return elem + "{}"
}

type arrayTypeLowerInfo struct {
	elem     string
	length   int64
	inferred bool
}

type localArrayLowerInfo struct {
	name  string
	info  arrayTypeLowerInfo
	start int
	end   int
}

type arrayComparisonOperand struct {
	start int
	end   int
	info  arrayTypeLowerInfo
}

func normalizeArrayComparisonOperands(body string, unitName string, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	tempIndex := 0
	return normalizeArrayComparisonOperandsWithTemp(body, unitName, arrayResults, localTypes, fieldTypes, arrayFieldTypes, &tempIndex)
}

func normalizeArrayComparisonOperandsWithTemp(body string, unitName string, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, tempIndex *int) string {
	if !strings.Contains(body, "==") && !strings.Contains(body, "!=") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	locals := collectLocalArrayLowerInfos(body, toks, localTypes, fieldTypes, arrayFieldTypes, arrayResults)
	out := make([]byte, 0, rewriteBufferCapacity(len(body)))
	cursor := 0
	for i := 0; i < len(toks); i++ {
		shortCircuit, ok := normalizationShortCircuitIfStatement(toks, i)
		if ok {
			replacement, changed := arrayComparisonShortCircuitIfReplacement(body, toks, shortCircuit, unitName, arrayResults, locals, localTypes, fieldTypes, arrayFieldTypes, tempIndex)
			if changed {
				insertStart := statementInsertStart(body, int(toks[shortCircuit.token].Start))
				out = appendStringRange(out, body, cursor, insertStart)
				out = appendString(out, replacement)
				cursor = int(toks[shortCircuit.end].End)
				i = shortCircuit.end
				continue
			}
		}
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		temps, replacements := normalizeArrayComparisonExpression(body, toks, stmt.exprStart, stmt.exprEnd, unitName, arrayResults, locals, localTypes, fieldTypes, arrayFieldTypes, tempIndex)
		if len(temps) == 0 && len(replacements) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, int(toks[stmt.token].Start))
		out = appendStringRange(out, body, cursor, insertStart)
		indent := statementIndent(body, int(toks[stmt.token].Start))
		if stmt.forCondition {
			innerIndent := indent + "\t"
			condition := applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements)
			out = appendStringRange(out, body, insertStart, int(toks[stmt.token].Start))
			out = appendString(out, "for {\n")
			for j := 0; j < len(temps); j++ {
				temp := temps[j]
				out = appendExpressionTempDecl(out, innerIndent, temp)
			}
			out = appendString(out, innerIndent)
			out = appendString(out, "if !(")
			out = appendString(out, condition)
			out = appendString(out, ") {\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "\tbreak\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "}\n")
			cursor = int(toks[stmt.openBrace].End)
			i = stmt.openBrace
			continue
		}
		if insertStart == int(toks[stmt.token].Start) {
			out = append(out, '\n')
		}
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			out = appendExpressionTempDecl(out, indent, temp)
		}
		out = appendStringRange(out, body, insertStart, int(toks[stmt.exprStart].Start))
		out = appendString(out, applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements))
		cursor = int(toks[stmt.exprEnd-1].End)
		i = stmt.exprEnd - 1
	}
	if len(out) == 0 {
		arena.Reset(mark)
		return body
	}
	out = appendStringRange(out, body, cursor, len(body))
	normalized := arena.PersistString(string(out))
	arena.Reset(mark)
	return normalized
}

func arrayComparisonShortCircuitIfReplacement(body string, toks []scan.Token, stmt shortCircuitIfStatement, unitName string, arrayResults []arrayFunctionResultLowerInfo, locals []localArrayLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, tempIndex *int) (string, bool) {
	indent := statementIndent(body, int(toks[stmt.token].Start))
	currentIndent := indent
	var out []byte
	changed := false
	for i := 0; i < len(stmt.operands); i++ {
		operand := stmt.operands[i]
		temps, replacements := normalizeArrayComparisonExpression(body, toks, operand.start, operand.end, unitName, arrayResults, locals, localTypes, fieldTypes, arrayFieldTypes, tempIndex)
		if len(temps) > 0 || len(replacements) > 0 {
			changed = true
		}
		for j := 0; j < len(temps); j++ {
			temp := temps[j]
			out = appendExpressionTempDecl(out, currentIndent, temp)
		}
		condition := strings.TrimSpace(applyExpressionReplacements(body, int(toks[operand.start].Start), int(toks[operand.end-1].End), replacements))
		out = appendString(out, currentIndent)
		out = appendString(out, "if ")
		out = appendString(out, condition)
		out = appendString(out, " {\n")
		currentIndent += "\t"
	}
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBody := normalizeArrayComparisonOperandsWithTemp(body[innerBodyStart:innerBodyEnd], unitName, arrayResults, localTypes, fieldTypes, arrayFieldTypes, tempIndex)
	if innerBody != body[innerBodyStart:innerBodyEnd] {
		changed = true
	}
	out = appendString(out, innerBody)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	for i := len(stmt.operands) - 1; i >= 0; i-- {
		currentIndent = currentIndent[:len(currentIndent)-1]
		out = appendString(out, currentIndent)
		out = appendString(out, "}\n")
	}
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return string(out), changed
}

func normalizeArrayComparisonExpression(body string, toks []scan.Token, start int, end int, unitName string, arrayResults []arrayFunctionResultLowerInfo, locals []localArrayLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	for i := start; i < end; i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left, leftOK := arrayComparisonOperandBefore(body, toks, i, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes)
		if !leftOK {
			continue
		}
		right, rightOK := arrayComparisonOperandAfter(body, toks, i, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes)
		if !rightOK || !arrayTypeLowerInfosComparable(left.info, right.info) {
			continue
		}
		if arrayComparisonOperandNeedsTemp(toks, left) {
			name := nextExpressionTempName(body, unitName+"_array_cmp", tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, typ: arrayTypeLowerInfoText(left.info), expr: arrayComparisonOperandSource(body, toks, left)})
			replacements = append(replacements, expressionReplacement{start: int(toks[left.start].Start), end: int(toks[left.end-1].End), text: name})
		}
		if arrayComparisonOperandNeedsTemp(toks, right) {
			name := nextExpressionTempName(body, unitName+"_array_cmp", tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, typ: arrayTypeLowerInfoText(right.info), expr: arrayComparisonOperandSource(body, toks, right)})
			replacements = append(replacements, expressionReplacement{start: int(toks[right.start].Start), end: int(toks[right.end-1].End), text: name})
		}
		i = right.end - 1
	}
	return temps, replacements
}

func arrayComparisonOperandNeedsTemp(toks []scan.Token, operand arrayComparisonOperand) bool {
	if operand.start+1 == operand.end && toks[operand.start].Kind == scan.Ident {
		return false
	}
	return expressionContainsCall(toks, operand.start, operand.end)
}

func arrayComparisonOperandSource(body string, toks []scan.Token, operand arrayComparisonOperand) string {
	return strings.TrimSpace(body[int(toks[operand.start].Start):int(toks[operand.end-1].End)])
}

func arrayTypeLowerInfoText(info arrayTypeLowerInfo) string {
	return "[" + strconv.FormatInt(info.length, 10) + "]" + info.elem
}

func lowerArrayComparisons(body string, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if !strings.Contains(body, "==") && !strings.Contains(body, "!=") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	locals := collectLocalArrayLowerInfos(body, toks, localTypes, fieldTypes, arrayFieldTypes, arrayResults)
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left, leftOK := arrayComparisonOperandBefore(body, toks, i, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes)
		if !leftOK {
			continue
		}
		right, rightOK := arrayComparisonOperandAfter(body, toks, i, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes)
		if !rightOK {
			continue
		}
		if !arrayTypeLowerInfosComparable(left.info, right.info) {
			continue
		}
		text := loweredArrayComparisonExpression(body, toks, left, right, toks[i].Text)
		if text == "" {
			continue
		}
		replacements = append(replacements, expressionReplacement{
			start: int(toks[left.start].Start),
			end:   int(toks[right.end-1].End),
			text:  text,
		})
		i = right.end - 1
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

type structComparisonOperand struct {
	start int
	end   int
	owner string
	typ   string
}

type structComparisonFieldPath struct {
	fields []string
}

func normalizeStructComparisonOperands(body string, unitName string, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, topNames symbolNameTable) string {
	tempIndex := 0
	return normalizeStructComparisonOperandsWithTemp(body, unitName, functionResults, localTypes, fieldTypes, structOwners, arrayFieldTypes, topNames, &tempIndex)
}

func normalizeStructComparisonOperandsWithTemp(body string, unitName string, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, topNames symbolNameTable, tempIndex *int) string {
	if !strings.Contains(body, "==") && !strings.Contains(body, "!=") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	out := make([]byte, 0, rewriteBufferCapacity(len(body)))
	cursor := 0
	for i := 0; i < len(toks); i++ {
		shortCircuit, ok := normalizationShortCircuitIfStatement(toks, i)
		if ok {
			replacement, changed := structComparisonShortCircuitIfReplacement(body, toks, shortCircuit, unitName, functionResults, localTypes, fieldTypes, structOwners, arrayFieldTypes, topNames, tempIndex)
			if changed {
				insertStart := statementInsertStart(body, int(toks[shortCircuit.token].Start))
				out = appendStringRange(out, body, cursor, insertStart)
				out = appendString(out, replacement)
				cursor = int(toks[shortCircuit.end].End)
				i = shortCircuit.end
				continue
			}
		}
		stmt, ok := normalizationStatement(toks, i)
		if !ok {
			continue
		}
		temps, replacements := normalizeStructComparisonExpression(body, toks, stmt.exprStart, stmt.exprEnd, unitName, functionResults, localTypes, fieldTypes, structOwners, arrayFieldTypes, topNames, tempIndex)
		if len(temps) == 0 && len(replacements) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, int(toks[stmt.token].Start))
		out = appendStringRange(out, body, cursor, insertStart)
		indent := statementIndent(body, int(toks[stmt.token].Start))
		if stmt.forCondition {
			innerIndent := indent + "\t"
			condition := applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements)
			out = appendStringRange(out, body, insertStart, int(toks[stmt.token].Start))
			out = appendString(out, "for {\n")
			for j := 0; j < len(temps); j++ {
				out = appendExpressionTempDecl(out, innerIndent, temps[j])
			}
			out = appendString(out, innerIndent)
			out = appendString(out, "if !(")
			out = appendString(out, condition)
			out = appendString(out, ") {\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "\tbreak\n")
			out = appendString(out, innerIndent)
			out = appendString(out, "}\n")
			cursor = int(toks[stmt.openBrace].End)
			i = stmt.openBrace
			continue
		}
		if insertStart == int(toks[stmt.token].Start) {
			out = append(out, '\n')
		}
		for j := 0; j < len(temps); j++ {
			out = appendExpressionTempDecl(out, indent, temps[j])
		}
		out = appendStringRange(out, body, insertStart, int(toks[stmt.exprStart].Start))
		out = appendString(out, applyExpressionReplacements(body, int(toks[stmt.exprStart].Start), int(toks[stmt.exprEnd-1].End), replacements))
		cursor = int(toks[stmt.exprEnd-1].End)
		i = stmt.exprEnd - 1
	}
	if len(out) == 0 {
		arena.Reset(mark)
		return body
	}
	out = appendStringRange(out, body, cursor, len(body))
	normalized := arena.PersistString(string(out))
	arena.Reset(mark)
	return normalized
}

func structComparisonShortCircuitIfReplacement(body string, toks []scan.Token, stmt shortCircuitIfStatement, unitName string, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, topNames symbolNameTable, tempIndex *int) (string, bool) {
	indent := statementIndent(body, int(toks[stmt.token].Start))
	currentIndent := indent
	var out []byte
	changed := false
	for i := 0; i < len(stmt.operands); i++ {
		operand := stmt.operands[i]
		temps, replacements := normalizeStructComparisonExpression(body, toks, operand.start, operand.end, unitName, functionResults, localTypes, fieldTypes, structOwners, arrayFieldTypes, topNames, tempIndex)
		if len(temps) > 0 || len(replacements) > 0 {
			changed = true
		}
		for j := 0; j < len(temps); j++ {
			out = appendExpressionTempDecl(out, currentIndent, temps[j])
		}
		condition := strings.TrimSpace(applyExpressionReplacements(body, int(toks[operand.start].Start), int(toks[operand.end-1].End), replacements))
		out = appendString(out, currentIndent)
		out = appendString(out, "if ")
		out = appendString(out, condition)
		out = appendString(out, " {\n")
		currentIndent += "\t"
	}
	innerBodyStart := int(toks[stmt.openBrace].End)
	innerBodyEnd := int(toks[stmt.end].Start)
	innerBody := normalizeStructComparisonOperandsWithTemp(body[innerBodyStart:innerBodyEnd], unitName, functionResults, localTypes, fieldTypes, structOwners, arrayFieldTypes, topNames, tempIndex)
	if innerBody != body[innerBodyStart:innerBodyEnd] {
		changed = true
	}
	out = appendString(out, innerBody)
	if len(out) == 0 || out[len(out)-1] != '\n' {
		out = append(out, '\n')
	}
	for i := len(stmt.operands) - 1; i >= 0; i-- {
		currentIndent = currentIndent[:len(currentIndent)-1]
		out = appendString(out, currentIndent)
		out = appendString(out, "}\n")
	}
	if len(out) > 0 && out[len(out)-1] == '\n' {
		out = out[:len(out)-1]
	}
	return string(out), changed
}

func normalizeStructComparisonExpression(body string, toks []scan.Token, start int, end int, unitName string, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, topNames symbolNameTable, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	for i := start; i < end; i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left, leftOK := structComparisonOperandBeforeForNormalize(body, toks, i, functionResults, localTypes, fieldTypes, structOwners, topNames)
		if !leftOK {
			continue
		}
		right, rightOK := structComparisonOperandAfterForNormalize(body, toks, i, functionResults, localTypes, fieldTypes, structOwners, topNames)
		if !rightOK || left.owner != right.owner {
			continue
		}
		_, fieldsOK := structComparisonFieldsForLower(fieldTypes, structOwners, arrayFieldTypes, left.owner)
		if !fieldsOK {
			continue
		}
		if structComparisonOperandNeedsTemp(toks, left) {
			name := nextExpressionTempName(body, unitName+"_struct_cmp", tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, typ: left.typ, expr: structComparisonOperandSource(body, toks, left)})
			replacements = append(replacements, expressionReplacement{start: int(toks[left.start].Start), end: int(toks[left.end-1].End), text: name})
		}
		if structComparisonOperandNeedsTemp(toks, right) {
			name := nextExpressionTempName(body, unitName+"_struct_cmp", tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, typ: right.typ, expr: structComparisonOperandSource(body, toks, right)})
			replacements = append(replacements, expressionReplacement{start: int(toks[right.start].Start), end: int(toks[right.end-1].End), text: name})
		}
		i = right.end - 1
	}
	return temps, replacements
}

func structComparisonOperandNeedsTemp(toks []scan.Token, operand structComparisonOperand) bool {
	return !structComparisonOperandIsSingleIdentifier(toks, operand.start, operand.end)
}

func structComparisonOperandIsSingleIdentifier(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	for start < end && toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close != end-1 {
			break
		}
		start++
		end = close
	}
	return start+1 == end && toks[start].Kind == scan.Ident
}

func structComparisonOperandSource(body string, toks []scan.Token, operand structComparisonOperand) string {
	return strings.TrimSpace(body[int(toks[operand.start].Start):int(toks[operand.end-1].End)])
}

func structComparisonOperandBeforeForNormalize(body string, toks []scan.Token, op int, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, topNames symbolNameTable) (structComparisonOperand, bool) {
	end := op
	start, innerEnd := trimOuterParensBefore(toks, end)
	if start >= 0 {
		operand, ok := structComparisonOperandInRangeForNormalize(body, toks, start+1, innerEnd, functionResults, localTypes, fieldTypes, structOwners, topNames)
		if ok {
			return structComparisonOperand{start: start, end: end, owner: operand.owner, typ: operand.typ}, true
		}
	}
	if end > 0 && toks[end-1].Text == "}" {
		open := findOpen(toks, end-1, "{", "}")
		if open >= 0 {
			start := explicitCompositeLiteralTypeStartBeforeOpen(toks, open)
			if start >= 0 {
				return structComparisonOperandInRangeForNormalize(body, toks, start, end, functionResults, localTypes, fieldTypes, structOwners, topNames)
			}
		}
	}
	if end > 0 && toks[end-1].Text == ")" {
		open := findOpen(toks, end-1, "(", ")")
		if open > 0 && toks[open-1].Kind == scan.Ident {
			return structComparisonOperandInRangeForNormalize(body, toks, open-1, end, functionResults, localTypes, fieldTypes, structOwners, topNames)
		}
	}
	return structComparisonOperandInRangeForNormalize(body, toks, op-1, op, functionResults, localTypes, fieldTypes, structOwners, topNames)
}

func structComparisonOperandAfterForNormalize(body string, toks []scan.Token, op int, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, topNames symbolNameTable) (structComparisonOperand, bool) {
	start := op + 1
	if start >= len(toks) {
		return structComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close > start {
			operand, ok := structComparisonOperandInRangeForNormalize(body, toks, start+1, close, functionResults, localTypes, fieldTypes, structOwners, topNames)
			if ok {
				return structComparisonOperand{start: start, end: close + 1, owner: operand.owner, typ: operand.typ}, true
			}
		}
	}
	if toks[start].Kind == scan.Ident {
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close > start+1 {
				return structComparisonOperandInRangeForNormalize(body, toks, start, close+1, functionResults, localTypes, fieldTypes, structOwners, topNames)
			}
		}
		if open := compositeLiteralOpenForTypeStart(toks, start); open > start {
			close := findClose(toks, open, "{", "}")
			if close > open {
				return structComparisonOperandInRangeForNormalize(body, toks, start, close+1, functionResults, localTypes, fieldTypes, structOwners, topNames)
			}
		}
		return structComparisonOperandInRangeForNormalize(body, toks, start, start+1, functionResults, localTypes, fieldTypes, structOwners, topNames)
	}
	return structComparisonOperand{}, false
}

func structComparisonOperandInRangeForNormalize(body string, toks []scan.Token, start int, end int, functionResults localTypeTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, topNames symbolNameTable) (structComparisonOperand, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return structComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			operand, ok := structComparisonOperandInRangeForNormalize(body, toks, start+1, close, functionResults, localTypes, fieldTypes, structOwners, topNames)
			if ok {
				return structComparisonOperand{start: start, end: end, owner: operand.owner, typ: operand.typ}, true
			}
		}
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		info := localTypeTableLookup(localTypes, toks[start].Text)
		return structComparisonOperandForTypeInfo(toks, start, end, info, fieldTypes, structOwners, topNames, false)
	}
	if open := compositeLiteralOpenForTypeStart(toks, start); open > start {
		close := findClose(toks, open, "{", "}")
		if close == end-1 {
			info := typeInfoInRange(toks, start, open)
			info.pointer = typeRangeIsPointer(toks, start, open)
			return structComparisonOperandForTypeInfo(toks, start, end, info, fieldTypes, structOwners, topNames, true)
		}
	}
	if start+2 <= end && toks[start].Kind == scan.Ident && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			info := localTypeTableLookup(functionResults, toks[start].Text)
			return structComparisonOperandForTypeInfo(toks, start, end, info, fieldTypes, structOwners, topNames, true)
		}
	}
	return structComparisonOperand{}, false
}

func structComparisonOperandForTypeInfo(toks []scan.Token, start int, end int, info localTypeInfo, fieldTypes structFieldTypeTable, structOwners structOwnerTable, topNames symbolNameTable, needsTypeText bool) (structComparisonOperand, bool) {
	if info.name == "" || info.pointer {
		return structComparisonOperand{}, false
	}
	owner := structComparisonResolvedOwner(localTypeInfoOwnerName(info), fieldTypes, structOwners, topNames)
	if owner == "" {
		return structComparisonOperand{}, false
	}
	typ := ""
	if needsTypeText {
		typ = structComparisonTempTypeText(info, topNames)
		if typ == "" {
			return structComparisonOperand{}, false
		}
	}
	return structComparisonOperand{start: start, end: end, owner: owner, typ: typ}, true
}

func structComparisonResolvedOwner(owner string, fieldTypes structFieldTypeTable, structOwners structOwnerTable, topNames symbolNameTable) string {
	if structComparisonOwnerExists(fieldTypes, structOwners, owner) {
		return owner
	}
	name := symbolNameTableSourceNameForUnit(topNames, owner)
	if name != "" && structComparisonOwnerExists(fieldTypes, structOwners, name) {
		return name
	}
	return ""
}

func structComparisonTempTypeText(info localTypeInfo, topNames symbolNameTable) string {
	if info.name == "" || info.pointer {
		return ""
	}
	if info.qualifier == "" {
		if unitName := symbolNameTableUnitName(topNames, info.name); unitName != "" {
			return unitName
		}
	}
	return localTypeInfoText(info)
}

func symbolNameTableSourceNameForUnit(table symbolNameTable, unitName string) string {
	for i := 0; i < len(table); i++ {
		if table[i].unitName == unitName {
			return table[i].name
		}
	}
	return ""
}

func collectBodyStructComparisonLocalTypes(body string, localTypes localTypeTable, topNames symbolNameTable) localTypeTable {
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return localTypes
	}
	for i := 0; i+2 < len(toks); i++ {
		if toks[i].Text != "var" || toks[i+1].Kind != scan.Ident {
			continue
		}
		typeStart := i + 2
		typeEnd := varTypeEnd(toks, typeStart, len(toks))
		if typeEnd <= typeStart {
			continue
		}
		info := typeInfoInRange(toks, typeStart, typeEnd)
		if info.name == "" {
			continue
		}
		info.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
		if info.qualifier == "" {
			if name := symbolNameTableSourceNameForUnit(topNames, info.name); name != "" {
				info.name = name
			}
		}
		localTypes = localTypeTableSet(localTypes, toks[i+1].Text, info)
	}
	return localTypes
}

func lowerStructComparisons(body string, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if !strings.Contains(body, "==") && !strings.Contains(body, "!=") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left, leftOK := structComparisonOperandBefore(toks, i, localTypes, fieldTypes, structOwners)
		if !leftOK {
			continue
		}
		right, rightOK := structComparisonOperandAfter(toks, i, localTypes, fieldTypes, structOwners)
		if !rightOK || left.owner != right.owner {
			continue
		}
		fields, fieldsOK := structComparisonFieldsForLower(fieldTypes, structOwners, arrayFieldTypes, left.owner)
		if !fieldsOK {
			continue
		}
		text := loweredStructComparisonExpression(body, toks, left, right, fields, toks[i].Text)
		if text == "" {
			continue
		}
		replacements = append(replacements, expressionReplacement{
			start: int(toks[left.start].Start),
			end:   int(toks[right.end-1].End),
			text:  text,
		})
		i = right.end - 1
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func structComparisonOperandBefore(toks []scan.Token, op int, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable) (structComparisonOperand, bool) {
	end := op
	start, innerEnd := trimOuterParensBefore(toks, end)
	if start >= 0 {
		operand, ok := structComparisonOperandInRange(toks, start+1, innerEnd, localTypes, fieldTypes, structOwners)
		if ok {
			return structComparisonOperand{start: start, end: end, owner: operand.owner}, true
		}
	}
	return structComparisonOperandInRange(toks, op-1, op, localTypes, fieldTypes, structOwners)
}

func structComparisonOperandAfter(toks []scan.Token, op int, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable) (structComparisonOperand, bool) {
	start := op + 1
	if start >= len(toks) {
		return structComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close > start {
			operand, ok := structComparisonOperandInRange(toks, start+1, close, localTypes, fieldTypes, structOwners)
			if ok {
				return structComparisonOperand{start: start, end: close + 1, owner: operand.owner}, true
			}
		}
	}
	return structComparisonOperandInRange(toks, start, start+1, localTypes, fieldTypes, structOwners)
}

func structComparisonOperandInRange(toks []scan.Token, start int, end int, localTypes localTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable) (structComparisonOperand, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return structComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			operand, ok := structComparisonOperandInRange(toks, start+1, close, localTypes, fieldTypes, structOwners)
			if ok {
				return structComparisonOperand{start: start, end: end, owner: operand.owner}, true
			}
		}
	}
	name := singleIdentifierExpressionInLower(toks, start, end)
	if name == "" {
		return structComparisonOperand{}, false
	}
	info := localTypeTableLookup(localTypes, name)
	if info.name == "" || info.pointer {
		return structComparisonOperand{}, false
	}
	owner := localTypeInfoOwnerName(info)
	if owner == "" || !structComparisonOwnerExists(fieldTypes, structOwners, owner) {
		return structComparisonOperand{}, false
	}
	return structComparisonOperand{start: start, end: end, owner: owner}, true
}

func structComparisonFieldsForLower(fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, owner string) ([]structComparisonFieldPath, bool) {
	return structComparisonFieldsForLowerWithPrefix(fieldTypes, structOwners, arrayFieldTypes, owner, nil, nil)
}

func structComparisonFieldsForLowerWithPrefix(fieldTypes structFieldTypeTable, structOwners structOwnerTable, arrayFieldTypes arrayStructFieldLowerInfoTable, owner string, prefix []string, seen []string) ([]structComparisonFieldPath, bool) {
	if owner == "" || containsString(seen, owner) || !structComparisonOwnerExists(fieldTypes, structOwners, owner) {
		return nil, false
	}
	seen = appendStringUnique(cloneStrings(seen), owner)
	var fields []structComparisonFieldPath
	ownerFound := false
	for i := 0; i < len(fieldTypes); i++ {
		entry := fieldTypes[i]
		if entry.owner != owner {
			continue
		}
		ownerFound = true
		path := cloneStrings(prefix)
		path = append(path, entry.field)
		if structComparisonFieldTypeLowerableForLower(entry, arrayFieldTypes) {
			fields = append(fields, structComparisonFieldPath{fields: path})
			continue
		}
		nestedOwner := localTypeInfoOwnerName(entry.info)
		if entry.info.pointer || nestedOwner == "" || !structComparisonOwnerExists(fieldTypes, structOwners, nestedOwner) {
			return nil, false
		}
		nestedFields, ok := structComparisonFieldsForLowerWithPrefix(fieldTypes, structOwners, arrayFieldTypes, nestedOwner, path, seen)
		if !ok {
			return nil, false
		}
		fields = append(fields, nestedFields...)
	}
	return fields, ownerFound || structOwnerTableContains(structOwners, owner)
}

func structComparisonFieldTypeLowerableForLower(entry structFieldTypeEntry, arrayFieldTypes arrayStructFieldLowerInfoTable) bool {
	info := entry.info
	if info.pointer {
		return true
	}
	if arrayInfo, ok := arrayStructFieldLowerInfoTableLookup(arrayFieldTypes, entry.owner, entry.field); ok {
		return structComparisonArrayFieldTypeLowerableForLower(arrayInfo)
	}
	if info.qualifier != "" {
		return false
	}
	switch info.name {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32":
		return true
	}
	return false
}

func structComparisonArrayFieldTypeLowerableForLower(info arrayTypeLowerInfo) bool {
	elem := info.elem
	if strings.HasPrefix(elem, "*") {
		return true
	}
	switch elem {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32":
		return true
	}
	return false
}

func loweredStructComparisonExpression(body string, toks []scan.Token, left structComparisonOperand, right structComparisonOperand, fields []structComparisonFieldPath, op string) string {
	if len(fields) == 0 {
		if op == "!=" {
			return "false"
		}
		return "true"
	}
	join := " && "
	fieldOp := "=="
	if op == "!=" {
		join = " || "
		fieldOp = "!="
	}
	var out []byte
	if len(fields) > 1 {
		out = append(out, '(')
	}
	for i := 0; i < len(fields); i++ {
		if i > 0 {
			out = appendString(out, join)
		}
		out = appendString(out, structComparisonFieldExpression(body, toks, left, fields[i].fields))
		out = append(out, ' ')
		out = appendString(out, fieldOp)
		out = append(out, ' ')
		out = appendString(out, structComparisonFieldExpression(body, toks, right, fields[i].fields))
	}
	if len(fields) > 1 {
		out = append(out, ')')
	}
	return string(out)
}

func structComparisonFieldExpression(body string, toks []scan.Token, operand structComparisonOperand, fields []string) string {
	start, end := trimTokenRange(toks, operand.start, operand.end)
	for start < end && toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close != end-1 {
			break
		}
		start++
		end = close
	}
	suffix := structComparisonFieldSelectorSuffix(fields)
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text + suffix
	}
	expr := strings.TrimSpace(body[int(toks[operand.start].Start):int(toks[operand.end-1].End)])
	return "(" + expr + ")" + suffix
}

func structComparisonFieldSelectorSuffix(fields []string) string {
	var out []byte
	for i := 0; i < len(fields); i++ {
		out = append(out, '.')
		out = appendString(out, fields[i])
	}
	return string(out)
}

func collectLocalArrayLowerInfos(body string, toks []scan.Token, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, arrayResults []arrayFunctionResultLowerInfo) []localArrayLowerInfo {
	bodyOpen := firstFunctionBodyOpenForLower(toks)
	if bodyOpen < 0 {
		return nil
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = len(toks) - 1
	}
	var infos []localArrayLowerInfo
	for i := 0; i < bodyOpen; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > bodyOpen {
			continue
		}
		if functionParameterListOpenForLower(toks, i) || functionResultListOpenForLower(toks, i) {
			infos = collectParameterArrayLowerInfos(body, toks, i+1, close, 0, maxSourcePosition(), infos)
		}
		i = close
	}
	for i := bodyOpen + 1; i < bodyClose; i++ {
		switch toks[i].Text {
		case "var":
			infos = collectVarArrayLowerInfos(body, toks, bodyOpen, i, bodyClose, infos)
		case ":=":
			infos = collectShortDeclArrayLowerInfos(body, toks, bodyOpen, i, bodyClose, infos, localTypes, fieldTypes, arrayFieldTypes, arrayResults)
		}
	}
	return infos
}

func firstFunctionBodyOpenForLower(toks []scan.Token) int {
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "func" {
			continue
		}
		for j := i + 1; j < len(toks); j++ {
			if toks[j].Text != "(" {
				continue
			}
			close := findClose(toks, j, "(", ")")
			if close < 0 {
				return -1
			}
			return functionBodyOpenAfterParamsForLower(toks, close, maxSourcePosition())
		}
	}
	return -1
}

func collectParameterArrayLowerInfos(body string, toks []scan.Token, start int, end int, scopeStart int, scopeEnd int, infos []localArrayLowerInfo) []localArrayLowerInfo {
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == "," {
			name, info, hasType, hasName := parameterArrayLowerSegment(body, toks, segmentStart, i)
			if hasType {
				if hasName {
					pending = appendStringUnique(pending, name)
				}
				for j := 0; j < len(pending); j++ {
					infos = setLocalArrayLowerInfo(infos, pending[j], info, scopeStart, scopeEnd)
				}
				if len(pending) == 0 && hasName {
					infos = setLocalArrayLowerInfo(infos, name, info, scopeStart, scopeEnd)
				}
				pending = nil
			} else if hasName {
				pending = appendStringUnique(pending, name)
			}
			segmentStart = i + 1
		}
	}
	return infos
}

func parameterArrayLowerSegment(body string, toks []scan.Token, start int, end int) (string, arrayTypeLowerInfo, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", arrayTypeLowerInfo{}, false, false
	}
	if toks[start].Kind == scan.Ident {
		if start+1 < end && isTypeStart(toks[start+1]) {
			info, _, _, ok := arrayTypeInfoForRange(body, toks, start+1, end)
			if ok && !info.inferred {
				return toks[start].Text, info, true, true
			}
			return toks[start].Text, arrayTypeLowerInfo{}, false, true
		}
		if start+1 == end {
			return toks[start].Text, arrayTypeLowerInfo{}, false, true
		}
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, start, end)
	return "", info, ok && !info.inferred, false
}

func collectVarArrayLowerInfos(body string, toks []scan.Token, bodyOpen int, pos int, limit int, infos []localArrayLowerInfo) []localArrayLowerInfo {
	if pos+1 < limit && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close > limit {
			return infos
		}
		specs := localConstSpecRanges(toks, pos+2, close)
		for i := 0; i < len(specs); i++ {
			infos = collectVarSpecArrayLowerInfos(body, toks, bodyOpen, specs[i], close, infos)
		}
		return infos
	}
	end := lowerSimpleStatementEnd(toks, pos+1, limit)
	return collectVarSpecArrayLowerInfos(body, toks, bodyOpen, expressionRange{start: pos + 1, end: end}, limit, infos)
}

func collectVarSpecArrayLowerInfos(body string, toks []scan.Token, bodyOpen int, spec expressionRange, limit int, infos []localArrayLowerInfo) []localArrayLowerInfo {
	start, end := trimTokenRange(toks, spec.start, spec.end)
	if start >= end {
		return infos
	}
	eq := findTopLevelToken(toks, start, end, "=")
	prefixEnd := end
	if eq >= 0 {
		prefixEnd = eq
	}
	names, typeStart := localVarSpecNamesAndType(toks, start, prefixEnd)
	if len(names) == 0 || typeStart < 0 {
		return infos
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, typeStart, prefixEnd)
	if !ok || info.inferred {
		return infos
	}
	scopeStart := int(toks[end-1].End)
	scopeEnd := localScopeEnd(toks, bodyOpen, start, int(toks[limit].End))
	for i := 0; i < len(names); i++ {
		infos = setLocalArrayLowerInfo(infos, names[i], info, scopeStart, scopeEnd)
	}
	return infos
}

func collectShortDeclArrayLowerInfos(body string, toks []scan.Token, bodyOpen int, assign int, limit int, infos []localArrayLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, arrayResults []arrayFunctionResultLowerInfo) []localArrayLowerInfo {
	stmtStart := simpleStatementStartForLower(toks, bodyOpen, assign)
	for stmtStart < assign && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
		stmtStart++
	}
	stmtEnd := lowerSimpleStatementEnd(toks, assign+1, limit)
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return infos
	}
	scopeStart := int(toks[assign].End)
	scopeEnd := localScopeEnd(toks, bodyOpen, assign, int(toks[limit].End))
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		info, ok := arrayCompositeLiteralInfoForRange(body, toks, rhs[i].start, rhs[i].end)
		if !ok {
			rhsStart, rhsEnd := trimTokenRange(toks, rhs[i].start, rhs[i].end)
			_, rhsInfo, rhsOK := arrayCopySourceExpression(body, toks, rhsStart, rhsEnd, infos, localTypes, fieldTypes, arrayFieldTypes, int(toks[assign].Start))
			if !rhsOK {
				rhsInfo, rhsOK = arrayFunctionCallResultLowerInfo(toks, rhsStart, rhsEnd, arrayResults)
				if !rhsOK {
					continue
				}
			}
			info = rhsInfo
		}
		infos = setLocalArrayLowerInfo(infos, name, info, scopeStart, scopeEnd)
	}
	return infos
}

func setLocalArrayLowerInfo(infos []localArrayLowerInfo, name string, info arrayTypeLowerInfo, start int, end int) []localArrayLowerInfo {
	if name == "" || name == "_" || info.elem == "" || info.inferred {
		return infos
	}
	return append(infos, localArrayLowerInfo{name: name, info: info, start: start, end: end})
}

func localArrayLowerInfoAt(infos []localArrayLowerInfo, name string, pos int) (arrayTypeLowerInfo, bool) {
	for i := len(infos) - 1; i >= 0; i-- {
		info := infos[i]
		if info.name == name && pos >= info.start && (info.end <= 0 || pos < info.end) {
			return info.info, true
		}
	}
	return arrayTypeLowerInfo{}, false
}

func arrayComparisonOperandBefore(body string, toks []scan.Token, op int, locals []localArrayLowerInfo, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) (arrayComparisonOperand, bool) {
	end := op
	start, innerEnd := trimOuterParensBefore(toks, end)
	if start >= 0 {
		operand, ok := arrayComparisonOperandInRange(body, toks, start, innerEnd, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
		if ok {
			return arrayComparisonOperand{start: start, end: end, info: operand.info}, true
		}
	}
	if end > 0 && toks[end-1].Text == "}" {
		open := findOpen(toks, end-1, "{", "}")
		if open >= 0 {
			start := arrayCompositeLiteralTypeStartBeforeOpenForLower(toks, open)
			if start >= 0 {
				return arrayComparisonOperandInRange(body, toks, start, end, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
			}
		}
	}
	if end > 0 && toks[end-1].Text == ")" {
		open := findOpen(toks, end-1, "(", ")")
		if open > 0 && toks[open-1].Kind == scan.Ident {
			return arrayComparisonOperandInRange(body, toks, open-1, end, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
		}
	}
	if start := arraySelectorOperandStartBefore(toks, end); start >= 0 {
		return arrayComparisonOperandInRange(body, toks, start, end, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
	}
	if end >= 3 && toks[end-1].Kind == scan.Ident && toks[end-2].Text == "." && toks[end-3].Kind == scan.Ident {
		return arrayComparisonOperandInRange(body, toks, end-3, end, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
	}
	return arrayComparisonOperandInRange(body, toks, op-1, op, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
}

func arrayComparisonOperandAfter(body string, toks []scan.Token, op int, locals []localArrayLowerInfo, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) (arrayComparisonOperand, bool) {
	start := op + 1
	if start >= len(toks) {
		return arrayComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close > start {
			operand, ok := arrayComparisonOperandInRange(body, toks, start+1, close, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
			if ok {
				return arrayComparisonOperand{start: start, end: close + 1, info: operand.info}, true
			}
		}
	}
	if toks[start].Kind == scan.Ident {
		if start+1 < len(toks) && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close > start+1 {
				return arrayComparisonOperandInRange(body, toks, start, close+1, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
			}
		}
		if end := arraySelectorOperandEndAfter(toks, start); end > start+1 {
			operand, ok := arrayComparisonOperandInRange(body, toks, start, end, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
			if ok {
				return operand, true
			}
		}
		if start+2 < len(toks) && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
			return arrayComparisonOperandInRange(body, toks, start, start+3, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
		}
		return arrayComparisonOperandInRange(body, toks, start, start+1, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
	}
	if toks[start].Text == "[" {
		open := arrayCompositeLiteralOpenForLower(toks, start)
		if open < 0 {
			return arrayComparisonOperand{}, false
		}
		close := findClose(toks, open, "{", "}")
		if close < 0 {
			return arrayComparisonOperand{}, false
		}
		return arrayComparisonOperandInRange(body, toks, start, close+1, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, op)
	}
	return arrayComparisonOperand{}, false
}

func arraySelectorOperandStartBefore(toks []scan.Token, end int) int {
	start := postfixExpressionStartBefore(toks, end)
	if start < 0 || start+2 >= end || !containsTokenText(toks, start, end, ".") {
		return -1
	}
	return start
}

func postfixExpressionStartBefore(toks []scan.Token, end int) int {
	if end <= 0 || end > len(toks) {
		return -1
	}
	last := end - 1
	if toks[last].Text == "]" {
		open := findOpen(toks, last, "[", "]")
		if open <= 0 {
			return -1
		}
		return postfixExpressionStartBefore(toks, open)
	}
	if toks[last].Text == ")" {
		open := findOpen(toks, last, "(", ")")
		if open < 0 {
			return -1
		}
		return open
	}
	if toks[last].Kind != scan.Ident {
		return -1
	}
	start := last
	for start > 1 && toks[start-1].Text == "." {
		receiver := postfixExpressionStartBefore(toks, start-1)
		if receiver < 0 {
			break
		}
		start = receiver
	}
	return start
}

func arraySelectorOperandEndAfter(toks []scan.Token, start int) int {
	end := postfixExpressionEndAfter(toks, start)
	if end <= start+1 || !containsTokenText(toks, start, end, ".") {
		return -1
	}
	return end
}

func postfixExpressionEndAfter(toks []scan.Token, start int) int {
	if start >= len(toks) {
		return -1
	}
	end := start
	if toks[end].Kind == scan.Ident {
		end++
	} else if toks[end].Text == "(" {
		close := findClose(toks, end, "(", ")")
		if close < 0 {
			return -1
		}
		end = close + 1
	} else {
		return -1
	}
	for end < len(toks) {
		if toks[end].Text == "[" {
			close := findClose(toks, end, "[", "]")
			if close < 0 {
				return end
			}
			end = close + 1
			continue
		}
		if end+1 < len(toks) && toks[end].Text == "." && toks[end+1].Kind == scan.Ident {
			end += 2
			continue
		}
		break
	}
	return end
}

func trimOuterParensBefore(toks []scan.Token, end int) (int, int) {
	if end <= 0 || toks[end-1].Text != ")" {
		return -1, -1
	}
	open := findOpen(toks, end-1, "(", ")")
	if open < 0 {
		return -1, -1
	}
	return open, end - 1
}

func arrayComparisonOperandInRange(body string, toks []scan.Token, start int, end int, locals []localArrayLowerInfo, arrayResults []arrayFunctionResultLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int) (arrayComparisonOperand, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return arrayComparisonOperand{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			operand, ok := arrayComparisonOperandInRange(body, toks, start+1, close, locals, arrayResults, localTypes, fieldTypes, arrayFieldTypes, pos)
			if ok {
				return arrayComparisonOperand{start: start, end: end, info: operand.info}, true
			}
		}
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		info, ok := localArrayLowerInfoAt(locals, toks[start].Text, int(toks[pos].Start))
		if ok {
			return arrayComparisonOperand{start: start, end: end, info: info}, true
		}
		info, ok = directArrayValueLowerInfo(toks[start].Text, localTypes)
		if ok {
			return arrayComparisonOperand{start: start, end: end, info: info}, true
		}
	}
	if info, ok := arrayFunctionCallResultLowerInfo(toks, start, end, arrayResults); ok {
		return arrayComparisonOperand{start: start, end: end, info: info}, true
	}
	if info, ok := arraySelectorLowerInfo(toks, start, end, localTypes, fieldTypes, arrayFieldTypes); ok {
		return arrayComparisonOperand{start: start, end: end, info: info}, true
	}
	info, ok := arrayCompositeLiteralInfoForRange(body, toks, start, end)
	if !ok {
		return arrayComparisonOperand{}, false
	}
	return arrayComparisonOperand{start: start, end: end, info: info}, true
}

func arraySelectorLowerInfo(toks []scan.Token, start int, end int, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return arrayTypeLowerInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return arraySelectorLowerInfo(toks, start+1, close, localTypes, fieldTypes, arrayFieldTypes)
		}
	}
	if toks[start].Kind != scan.Ident || start+2 >= end {
		return arrayTypeLowerInfo{}, false
	}
	receiver := localTypeTableLookup(localTypes, toks[start].Text)
	owner := localTypeInfoOwnerName(receiver)
	if owner == "" {
		return arrayTypeLowerInfo{}, false
	}
	for i := start + 1; i < end; {
		if toks[i].Text == "[" {
			close := findClose(toks, i, "[", "]")
			if close < 0 || close >= end {
				return arrayTypeLowerInfo{}, false
			}
			owner = indexedSelectorOwnerName(owner)
			if owner == "" {
				return arrayTypeLowerInfo{}, false
			}
			i = close + 1
			continue
		}
		if i+1 >= end {
			return arrayTypeLowerInfo{}, false
		}
		if toks[i].Text != "." || toks[i+1].Kind != scan.Ident {
			return arrayTypeLowerInfo{}, false
		}
		field := toks[i+1].Text
		if i+2 == end {
			return arrayStructFieldLowerInfoTableLookup(arrayFieldTypes, owner, field)
		}
		next := structFieldTypeTableLookup(fieldTypes, owner, field)
		owner = localTypeInfoOwnerName(next)
		if owner == "" || next.pointer {
			return arrayTypeLowerInfo{}, false
		}
		i += 2
	}
	return arrayTypeLowerInfo{}, false
}

func indexedSelectorOwnerName(owner string) string {
	if strings.HasPrefix(owner, "[]") {
		return strings.TrimPrefix(owner[2:], "*")
	}
	if strings.HasPrefix(owner, "[") {
		close := strings.Index(owner, "]")
		if close > 0 && close+1 < len(owner) {
			return strings.TrimPrefix(owner[close+1:], "*")
		}
	}
	return owner
}

func arrayFunctionCallResultLowerInfo(toks []scan.Token, start int, end int, arrayResults []arrayFunctionResultLowerInfo) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return arrayTypeLowerInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return arrayFunctionCallResultLowerInfo(toks, start+1, close, arrayResults)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return arrayTypeLowerInfo{}, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return arrayTypeLowerInfo{}, false
	}
	for i := 0; i < len(arrayResults); i++ {
		entry := arrayResults[i]
		if entry.name == toks[start].Text {
			return entry.info, true
		}
	}
	return arrayTypeLowerInfo{}, false
}

func arrayCompositeLiteralInfoForRange(body string, toks []scan.Token, start int, end int) (arrayTypeLowerInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return arrayTypeLowerInfo{}, false
	}
	open := arrayCompositeLiteralOpenForLower(toks, start)
	if open < 0 {
		return arrayTypeLowerInfo{}, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return arrayTypeLowerInfo{}, false
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, start, open)
	if !ok {
		return arrayTypeLowerInfo{}, false
	}
	if info.inferred {
		elems, elemsOK := loweredArrayLiteralElements(body, toks, open, close, info)
		if !elemsOK {
			return arrayTypeLowerInfo{}, false
		}
		info.length = int64(len(elems))
		info.inferred = false
	}
	return info, true
}

func arrayCompositeLiteralTypeStartBeforeOpenForLower(toks []scan.Token, open int) int {
	if open <= 0 {
		return -1
	}
	start := open - 1
	if toks[start].Kind != scan.Ident {
		return -1
	}
	if start >= 2 && toks[start-1].Text == "." && toks[start-2].Kind == scan.Ident {
		start -= 2
	}
	if start >= 1 && toks[start-1].Text == "*" {
		start--
	}
	for start > 0 && toks[start-1].Text == "]" {
		brackOpen := findOpen(toks, start-1, "[", "]")
		if brackOpen < 0 {
			return -1
		}
		start = brackOpen
	}
	if toks[start].Text != "[" || start+1 >= open || toks[start+1].Text == "]" {
		return -1
	}
	return start
}

func arrayTypeLowerInfosComparable(left arrayTypeLowerInfo, right arrayTypeLowerInfo) bool {
	return !left.inferred && !right.inferred && left.length == right.length && left.elem == right.elem
}

func loweredArrayComparisonExpression(body string, toks []scan.Token, left arrayComparisonOperand, right arrayComparisonOperand, op string) string {
	if left.info.length == 0 {
		if op == "==" {
			return "true"
		}
		return "false"
	}
	join := " && "
	elementOp := "=="
	if op == "!=" {
		join = " || "
		elementOp = "!="
	}
	var out []byte
	if left.info.length > 1 {
		out = append(out, '(')
	}
	for i := int64(0); i < left.info.length; i++ {
		if i > 0 {
			out = appendString(out, join)
		}
		out = appendString(out, arrayComparisonElementExpression(body, toks, left, i))
		out = append(out, ' ')
		out = appendString(out, elementOp)
		out = append(out, ' ')
		out = appendString(out, arrayComparisonElementExpression(body, toks, right, i))
	}
	if left.info.length > 1 {
		out = append(out, ')')
	}
	return string(out)
}

func arrayComparisonElementExpression(body string, toks []scan.Token, operand arrayComparisonOperand, index int64) string {
	if expr, ok := arrayComparisonLiteralElementExpression(body, toks, operand, index); ok {
		return expr
	}
	expr := strings.TrimSpace(body[int(toks[operand.start].Start):int(toks[operand.end-1].End)])
	if operand.start+1 == operand.end && toks[operand.start].Kind == scan.Ident {
		return expr + "[" + strconv.FormatInt(index, 10) + "]"
	}
	return "(" + expr + ")[" + strconv.FormatInt(index, 10) + "]"
}

func arrayComparisonLiteralElementExpression(body string, toks []scan.Token, operand arrayComparisonOperand, index int64) (string, bool) {
	if operand.start < operand.end && toks[operand.start].Text == "(" {
		close := findClose(toks, operand.start, "(", ")")
		if close == operand.end-1 {
			return arrayComparisonLiteralElementExpression(body, toks, arrayComparisonOperand{start: operand.start + 1, end: close, info: operand.info}, index)
		}
	}
	if operand.start >= operand.end || toks[operand.start].Text != "[" {
		return "", false
	}
	open := arrayCompositeLiteralOpenForLower(toks, operand.start)
	if open < 0 {
		return "", false
	}
	close := findClose(toks, open, "{", "}")
	if close != operand.end-1 {
		return "", false
	}
	elems, ok := loweredArrayLiteralElements(body, toks, open, close, operand.info)
	if !ok || index < 0 || index >= int64(len(elems)) {
		return "", false
	}
	return elems[int(index)], true
}

func localArrayVarTypeInfo(body string, toks []scan.Token, start int, end int) (arrayTypeLowerInfo, bool) {
	info, _, _, ok := arrayTypeInfoForRange(body, toks, start, end)
	if !ok || info.inferred {
		return arrayTypeLowerInfo{}, false
	}
	return info, true
}

func lowerArrayCompositeLiterals(body string) string {
	if !strings.Contains(body, "[") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, close, ok := arrayCompositeLiteralReplacement(body, toks, i)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func normalizeArrayValueCopies(body string, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if !strings.Contains(body, "=") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	locals := collectLocalArrayLowerInfos(body, toks, localTypes, fieldTypes, arrayFieldTypes, nil)
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if !isAssignmentOperator(toks[i].Text) || isClassicForHeaderAssignment(toks, i) {
			continue
		}
		stmtEnd := lowerSimpleStatementEnd(toks, i+1, len(toks))
		if stmtEnd <= i+1 {
			continue
		}
		rhs := topLevelExpressionRanges(toks, i+1, stmtEnd)
		for rhsIndex := 0; rhsIndex < len(rhs); rhsIndex++ {
			start, end := trimTokenRange(toks, rhs[rhsIndex].start, rhs[rhsIndex].end)
			source, info, ok := arrayCopySourceExpression(body, toks, start, end, locals, localTypes, fieldTypes, arrayFieldTypes, int(toks[i].Start))
			if !ok {
				continue
			}
			replacements = append(replacements, expressionReplacement{
				start: int(toks[start].Start),
				end:   int(toks[end-1].End),
				text:  arrayCloneExpression(info, source),
			})
		}
		i = stmtEnd - 1
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func arrayCopySourceLocal(body string, toks []scan.Token, start int, end int, locals []localArrayLowerInfo, pos int) (string, arrayTypeLowerInfo, bool) {
	start, end = trimOuterParens(toks, start, end)
	if start+1 != end || toks[start].Kind != scan.Ident {
		return "", arrayTypeLowerInfo{}, false
	}
	info, ok := localArrayLowerInfoAt(locals, toks[start].Text, pos)
	if !ok {
		return "", arrayTypeLowerInfo{}, false
	}
	return toks[start].Text, info, true
}

func arrayCopySourceExpression(body string, toks []scan.Token, start int, end int, locals []localArrayLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int) (string, arrayTypeLowerInfo, bool) {
	source, info, ok := arrayCopySourceLocal(body, toks, start, end, locals, pos)
	if ok {
		return source, info, true
	}
	start, end = trimOuterParens(toks, start, end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		info, ok = directArrayValueLowerInfo(toks[start].Text, localTypes)
		if ok {
			return toks[start].Text, info, true
		}
	}
	info, ok = arraySelectorLowerInfo(toks, start, end, localTypes, fieldTypes, arrayFieldTypes)
	if !ok {
		return "", arrayTypeLowerInfo{}, false
	}
	source = strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	if source == "" {
		return "", arrayTypeLowerInfo{}, false
	}
	return source, info, true
}

func arrayCloneExpression(info arrayTypeLowerInfo, source string) string {
	return "append(make([]" + info.elem + ", 0, " + strconv.FormatInt(info.length, 10) + "), " + source + "...)"
}

func directArrayValueLowerInfo(name string, localTypes localTypeTable) (arrayTypeLowerInfo, bool) {
	info := localTypeTableLookup(localTypes, name)
	return arrayTypeLowerInfoFromLocalTypeInfo(info)
}

func arrayTypeLowerInfoFromLocalTypeInfo(info localTypeInfo) (arrayTypeLowerInfo, bool) {
	if info.pointer || info.qualifier != "" || info.name == "" {
		return arrayTypeLowerInfo{}, false
	}
	if !strings.HasPrefix(info.name, "[") {
		return arrayTypeLowerInfo{}, false
	}
	close := strings.Index(info.name, "]")
	if close <= 1 || close+1 >= len(info.name) {
		return arrayTypeLowerInfo{}, false
	}
	length, err := strconv.ParseInt(info.name[1:close], 0, 64)
	if err != nil || length < 0 {
		return arrayTypeLowerInfo{}, false
	}
	elem := strings.TrimSpace(info.name[close+1:])
	if elem == "" || strings.HasPrefix(elem, "[") {
		return arrayTypeLowerInfo{}, false
	}
	return arrayTypeLowerInfo{elem: elem, length: length}, true
}

func arrayTypeLowerInfoFixedName(info arrayTypeLowerInfo) string {
	return "[" + strconv.FormatInt(info.length, 10) + "]" + info.elem
}

func normalizeIndexedArraySelectorElementAssignments(body string, unitName string, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if !strings.Contains(body, "=") || !strings.Contains(body, "]") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	bodyOpen := firstFunctionBodyOpenForLower(toks)
	if bodyOpen < 0 {
		arena.Reset(mark)
		return body
	}
	tempIndex := 0
	out := make([]byte, 0, rewriteBufferCapacity(len(body)))
	cursor := 0
	for i := bodyOpen + 1; i < len(toks); i++ {
		if toks[i].Text != "=" || isClassicForHeaderAssignment(toks, i) {
			continue
		}
		stmtStart := simpleStatementStartForLower(toks, bodyOpen, i)
		for stmtStart < i && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
			stmtStart++
		}
		lhs := topLevelExpressionRanges(toks, stmtStart, i)
		var temps []expressionTemp
		var replacements []expressionReplacement
		for lhsIndex := 0; lhsIndex < len(lhs); lhsIndex++ {
			replacement, temp, ok := indexedArraySelectorElementAssignmentReplacement(body, toks, lhs[lhsIndex], unitName, localTypes, fieldTypes, arrayFieldTypes, &tempIndex)
			if !ok {
				continue
			}
			replacements = append(replacements, replacement)
			temps = append(temps, temp)
			tempIndex++
		}
		if len(replacements) == 0 {
			continue
		}
		insertStart := statementInsertStart(body, int(toks[stmtStart].Start))
		out = appendStringRange(out, body, cursor, insertStart)
		indent := statementIndent(body, int(toks[stmtStart].Start))
		if insertStart == int(toks[stmtStart].Start) {
			out = append(out, '\n')
		}
		for tempIndex := 0; tempIndex < len(temps); tempIndex++ {
			temp := temps[tempIndex]
			out = appendString(out, indent)
			out = appendString(out, temp.name)
			out = appendString(out, " := ")
			out = appendString(out, temp.expr)
			out = append(out, '\n')
		}
		stmtEnd := lowerSimpleStatementEnd(toks, i+1, len(toks))
		out = appendStringRange(out, body, insertStart, int(toks[stmtStart].Start))
		out = appendString(out, applyExpressionReplacements(body, int(toks[stmtStart].Start), int(toks[stmtEnd-1].End), replacements))
		cursor = int(toks[stmtEnd-1].End)
		i = stmtEnd - 1
	}
	if len(out) == 0 {
		arena.Reset(mark)
		return body
	}
	out = appendStringRange(out, body, cursor, len(body))
	body = arena.PersistString(string(out))
	arena.Reset(mark)
	return body
}

func indexedArraySelectorElementAssignmentReplacement(body string, toks []scan.Token, expr expressionRange, unitName string, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, tempIndex *int) (expressionReplacement, expressionTemp, bool) {
	start, end := trimTokenRange(toks, expr.start, expr.end)
	if start >= end || toks[end-1].Text != "]" {
		return expressionReplacement{}, expressionTemp{}, false
	}
	indexOpen := findOpen(toks, end-1, "[", "]")
	if indexOpen <= start {
		return expressionReplacement{}, expressionTemp{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == indexOpen-1 {
			return expressionReplacement{}, expressionTemp{}, false
		}
	}
	if _, ok := arraySelectorLowerInfo(toks, start, indexOpen, localTypes, fieldTypes, arrayFieldTypes); !ok {
		return expressionReplacement{}, expressionTemp{}, false
	}
	source := strings.TrimSpace(body[int(toks[start].Start):int(toks[indexOpen-1].End)])
	if source == "" {
		return expressionReplacement{}, expressionTemp{}, false
	}
	name := nextExpressionTempName(body, unitName+"_array_lvalue", tempIndex)
	return expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[indexOpen-1].End),
		text:  name,
	}, expressionTemp{name: name, expr: source}, true
}

func normalizeArrayArgumentCopies(body string, params []arrayFunctionParamLowerInfo, localTypes localTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if len(params) == 0 || !strings.Contains(body, "(") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	locals := collectLocalArrayLowerInfos(body, toks, localTypes, fieldTypes, arrayFieldTypes, nil)
	var replacements []expressionReplacement
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "(" {
			continue
		}
		if i > 0 && (toks[i-1].Text == "func" || toks[i-1].Text == ".") {
			continue
		}
		info, ok := arrayFunctionParamInfoByName(params, toks[i].Text)
		if !ok {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close <= i+1 {
			continue
		}
		args := topLevelExpressionRanges(toks, i+2, close)
		limit := len(args)
		if len(info.params) < limit {
			limit = len(info.params)
		}
		for argIndex := 0; argIndex < limit; argIndex++ {
			param := info.params[argIndex]
			if param.elem == "" {
				continue
			}
			start, end := trimTokenRange(toks, args[argIndex].start, args[argIndex].end)
			if start >= end || hasVariadicExpansionInTokens(toks, start, end) {
				continue
			}
			source, _, ok := arrayCopySourceExpression(body, toks, start, end, locals, localTypes, fieldTypes, arrayFieldTypes, int(toks[i].Start))
			if !ok {
				continue
			}
			replacements = append(replacements, expressionReplacement{
				start: int(toks[start].Start),
				end:   int(toks[end-1].End),
				text:  arrayCloneExpression(param, source),
			})
		}
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func arrayFunctionParamInfoByName(values []arrayFunctionParamLowerInfo, name string) (arrayFunctionParamLowerInfo, bool) {
	for i := 0; i < len(values); i++ {
		if values[i].name == name {
			return values[i], true
		}
	}
	return arrayFunctionParamLowerInfo{}, false
}

func arrayCompositeLiteralReplacement(body string, toks []scan.Token, start int) (expressionReplacement, int, bool) {
	open := arrayCompositeLiteralOpenForLower(toks, start)
	if open < 0 {
		return expressionReplacement{}, 0, false
	}
	close := findClose(toks, open, "{", "}")
	if close < 0 {
		return expressionReplacement{}, 0, false
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, start, open)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	elems, ok := loweredArrayLiteralElements(body, toks, open, close, info)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	var text []byte
	text = appendString(text, "[]")
	text = appendString(text, info.elem)
	text = append(text, '{')
	for i := 0; i < len(elems); i++ {
		if i > 0 {
			text = appendString(text, ", ")
		}
		text = appendString(text, elems[i])
	}
	text = append(text, '}')
	return expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[close].End),
		text:  string(text),
	}, close, true
}

func arrayCompositeLiteralOpenForLower(toks []scan.Token, start int) int {
	if start < 0 || start+1 >= len(toks) || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 {
		return -1
	}
	if _, _, ok := arrayLengthForTokens(toks, start+1, brackClose); !ok {
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

func arrayTypeInfoForRange(body string, toks []scan.Token, start int, end int) (arrayTypeLowerInfo, int, int, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Text != "[" {
		return arrayTypeLowerInfo{}, 0, 0, false
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose <= start+1 || brackClose+1 >= end {
		return arrayTypeLowerInfo{}, 0, 0, false
	}
	length, inferred, ok := arrayLengthForTokens(toks, start+1, brackClose)
	if !ok {
		return arrayTypeLowerInfo{}, 0, 0, false
	}
	elemStart := brackClose + 1
	elemEnd := end
	if elemStart >= elemEnd || toks[elemStart].Text == "[" {
		return arrayTypeLowerInfo{}, 0, 0, false
	}
	elem := strings.TrimSpace(body[int(toks[elemStart].Start):int(toks[elemEnd-1].End)])
	if elem == "" {
		return arrayTypeLowerInfo{}, 0, 0, false
	}
	return arrayTypeLowerInfo{elem: elem, length: length, inferred: inferred}, elemStart, elemEnd, true
}

func arrayLengthForTokens(toks []scan.Token, start int, end int) (int64, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+1 != end {
		return 0, false, false
	}
	if toks[start].Text == "..." {
		return 0, true, true
	}
	value, ok := simpleIntegerLiteralIndex(toks, start, end)
	if !ok || value < 0 {
		return 0, false, false
	}
	return value, false, true
}

func loweredArrayLiteralElements(body string, toks []scan.Token, open int, close int, info arrayTypeLowerInfo) ([]string, bool) {
	values := topLevelExpressionRanges(toks, open+1, close)
	nextIndex := int64(0)
	maxIndex := int64(-1)
	var indexed []keyedSliceLiteralValue
	for i := 0; i < len(values); i++ {
		value := values[i]
		if value.start >= value.end {
			continue
		}
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		index := nextIndex
		valueStart := value.start
		if colon >= 0 {
			var keyOK bool
			index, keyOK = simpleIntegerLiteralIndex(toks, value.start, colon)
			if !keyOK || index < 0 {
				return nil, false
			}
			valueStart = colon + 1
		}
		if !info.inferred && index >= info.length {
			return nil, false
		}
		for j := 0; j < len(indexed); j++ {
			if indexed[j].index == index {
				return nil, false
			}
		}
		valueStart, valueEnd := trimTokenRange(toks, valueStart, value.end)
		if valueStart >= valueEnd {
			return nil, false
		}
		expr := strings.TrimSpace(body[int(toks[valueStart].Start):int(toks[valueEnd-1].End)])
		indexed = append(indexed, keyedSliceLiteralValue{index: index, expr: expr})
		if index > maxIndex {
			maxIndex = index
		}
		nextIndex = index + 1
	}
	length := info.length
	if info.inferred {
		length = maxIndex + 1
		if length < 0 {
			length = 0
		}
	}
	elems := make([]string, int(length))
	zero := zeroValueForSliceElement(info.elem)
	for i := 0; i < len(elems); i++ {
		elems[i] = zero
	}
	for i := 0; i < len(indexed); i++ {
		value := indexed[i]
		if value.index < 0 || value.index >= int64(len(elems)) {
			return nil, false
		}
		elems[int(value.index)] = value.expr
	}
	return elems, true
}

func arrayZeroLiteral(elem string, length int64) string {
	if length <= 0 {
		return "[]" + elem + "{}"
	}
	zero := zeroValueForSliceElement(elem)
	var out []byte
	out = appendString(out, "[]")
	out = appendString(out, elem)
	out = append(out, '{')
	for i := int64(0); i < length; i++ {
		if i > 0 {
			out = appendString(out, ", ")
		}
		out = appendString(out, zero)
	}
	out = append(out, '}')
	return string(out)
}

func normalizeAddressOfCompositeLiteral(body string, toks []scan.Token, addrPos int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, int, bool) {
	open, close, ok := addressOfCompositeLiteralRange(toks, addrPos, end)
	if !ok {
		return nil, expressionReplacement{}, -1, false
	}
	var temps []expressionTemp
	valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, open+1, close, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, valueTemps)
	literalStart := int(toks[addrPos+1].Start)
	literalEnd := int(toks[close].End)
	expr := applyExpressionReplacements(body, literalStart, literalEnd, valueReplacements)
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(expr)})
	repl := expressionReplacement{
		start: int(toks[addrPos].Start),
		end:   int(toks[close].End),
		text:  "&" + name,
	}
	return temps, repl, close, true
}

func normalizeAddressOfCompositeLiteralSelector(body string, toks []scan.Token, addrPos int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, int, bool) {
	if addrPos+3 >= end || toks[addrPos].Text != "&" || toks[addrPos+1].Text != "(" {
		return nil, expressionReplacement{}, -1, false
	}
	closeParen := findClose(toks, addrPos+1, "(", ")")
	if closeParen <= addrPos+2 || closeParen >= end {
		return nil, expressionReplacement{}, -1, false
	}
	innerStart := addrPos + 2
	innerEnd := closeParen
	dot := firstTopLevelTokenInRange(toks, innerStart, innerEnd, ".")
	if dot < 0 {
		return nil, expressionReplacement{}, -1, false
	}
	replaceStart, replaceEnd, exprStart, exprEnd, ok := compositeLiteralSelectorBaseRanges(toks, dot)
	if !ok || replaceStart != innerStart || replaceEnd != dot {
		return nil, expressionReplacement{}, -1, false
	}
	var temps []expressionTemp
	valueTemps, valueReplacements := normalizeExpressionWithContext(body, toks, exprStart, exprEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, valueTemps)
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	expr := applyExpressionReplacements(body, int(toks[exprStart].Start), int(toks[exprEnd-1].End), valueReplacements)
	temps = append(temps, expressionTemp{name: name, expr: strings.TrimSpace(expr)})
	suffix := tokenRangeText(body, toks, dot, innerEnd)
	repl := expressionReplacement{
		start: int(toks[addrPos].Start),
		end:   int(toks[closeParen].End),
		text:  "&" + name + suffix,
	}
	return temps, repl, closeParen, true
}

func firstTopLevelTokenInRange(toks []scan.Token, start int, end int, text string) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == text {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func addressOfCompositeLiteralRange(toks []scan.Token, addrPos int, end int) (int, int, bool) {
	if addrPos+2 >= end || toks[addrPos].Text != "&" {
		return -1, -1, false
	}
	paren := 0
	brack := 0
	for i := addrPos + 1; i < end; i++ {
		if paren == 0 && brack == 0 && toks[i].Text == "{" {
			if i == addrPos+1 {
				return -1, -1, false
			}
			close := findClose(toks, i, "{", "}")
			return i, close, close >= i && close < end
		}
		switch toks[i].Text {
		case "(":
			paren++
		case ")":
			if paren > 0 {
				paren--
			}
		case "[":
			brack++
		case "]":
			if brack > 0 {
				brack--
			}
		}
	}
	return -1, -1, false
}

func normalizeAppendMultiValueCall(body string, toks []scan.Token, appendPos int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	if appendPos+1 >= len(toks) || toks[appendPos].Text != "append" || toks[appendPos+1].Text != "(" {
		return nil, expressionReplacement{}, false
	}
	ranges := topLevelExpressionRanges(toks, appendPos+2, close)
	if len(ranges) <= 2 {
		return nil, expressionReplacement{}, false
	}
	for i := 0; i < len(ranges); i++ {
		if expressionRangeContainsToken(toks, ranges[i], "...") {
			return nil, expressionReplacement{}, false
		}
	}
	var temps []expressionTemp
	args := make([]string, 0, len(ranges))
	for i := 0; i < len(ranges); i++ {
		arg := ranges[i]
		if arg.start >= arg.end {
			return nil, expressionReplacement{}, false
		}
		argStart := int(toks[arg.start].Start)
		argEnd := int(toks[arg.end-1].End)
		argTemps, argReplacements := normalizeExpressionWithContext(body, toks, arg.start, arg.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
		temps = appendExpressionTemps(temps, argTemps)
		expr := applyExpressionReplacements(body, argStart, argEnd, argReplacements)
		if appendArgumentNeedsTemp(toks, arg) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, expr: expr})
			args = append(args, name)
			continue
		}
		args = append(args, strings.TrimSpace(expr))
	}
	current := args[0]
	for i := 1; i < len(args); i++ {
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		temps = append(temps, expressionTemp{name: name, expr: "append(" + current + ", " + args[i] + ")"})
		current = name
	}
	repl := expressionReplacement{
		start: int(toks[appendPos].Start),
		end:   int(toks[close].End),
		text:  current,
	}
	return temps, repl, true
}

func normalizeAppendExpansionCall(body string, toks []scan.Token, appendPos int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	if appendPos+1 >= len(toks) || toks[appendPos].Text != "append" || toks[appendPos+1].Text != "(" {
		return nil, expressionReplacement{}, false
	}
	ranges := topLevelExpressionRanges(toks, appendPos+2, close)
	if len(ranges) != 2 {
		return nil, expressionReplacement{}, false
	}
	dst := ranges[0]
	src := ranges[1]
	if src.start >= src.end || toks[src.end-1].Text != "..." {
		return nil, expressionReplacement{}, false
	}
	src.end--
	if !appendExpansionSourceNeedsTemp(toks, src) {
		return nil, expressionReplacement{}, false
	}
	var temps []expressionTemp
	dstStart := int(toks[dst.start].Start)
	dstEnd := int(toks[dst.end-1].End)
	dstTemps, dstReplacements := normalizeExpressionWithContext(body, toks, dst.start, dst.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, dstTemps)
	dstExpr := strings.TrimSpace(applyExpressionReplacements(body, dstStart, dstEnd, dstReplacements))

	srcStart := int(toks[src.start].Start)
	srcEnd := int(toks[src.end-1].End)
	srcTemps, srcReplacements := normalizeExpressionWithContext(body, toks, src.start, src.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	temps = appendExpressionTemps(temps, srcTemps)
	srcExpr := strings.TrimSpace(applyExpressionReplacements(body, srcStart, srcEnd, srcReplacements))
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	temps = append(temps, expressionTemp{name: name, expr: srcExpr})
	repl := expressionReplacement{
		start: int(toks[appendPos].Start),
		end:   int(toks[close].End),
		text:  "append(" + dstExpr + ", " + name + "...)",
	}
	return temps, repl, true
}

func appendExpansionSourceNeedsTemp(toks []scan.Token, expr expressionRange) bool {
	start, end := trimTokenRange(toks, expr.start, expr.end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		return false
	}
	return true
}

func normalizeNewValueCall(body string, toks []scan.Token, newPos int, close int, unitName string, tempIndex *int, ctx normalizationContext) (expressionTemp, expressionReplacement, bool) {
	if newPos+2 >= close || toks[newPos].Text != "new" || toks[newPos+1].Text != "(" {
		return expressionTemp{}, expressionReplacement{}, false
	}
	typeName := newTypeTextInRange(toks, newPos+2, close)
	if typeName == "" {
		return expressionTemp{}, expressionReplacement{}, false
	}
	if strings.HasPrefix(typeName, "[]") && ctx.generatedDecls != nil {
		helper := ensureNewSliceHelperDecl(ctx.generatedDecls, unitName, typeName)
		repl := expressionReplacement{
			start: int(toks[newPos].Start),
			end:   int(toks[close].End),
			text:  helper + "()",
		}
		return expressionTemp{}, repl, true
	}
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	if strings.HasPrefix(typeName, "[]") {
		temp := expressionTemp{name: name, typ: typeName, expr: "nil"}
		repl := expressionReplacement{
			start: int(toks[newPos].Start),
			end:   int(toks[close].End),
			text:  "&" + name,
		}
		return temp, repl, true
	}
	sliceType := "[]" + typeName
	temp := expressionTemp{name: name, typ: sliceType, expr: "make(" + sliceType + ", 1)"}
	repl := expressionReplacement{
		start: int(toks[newPos].Start),
		end:   int(toks[close].End),
		text:  "&" + name + "[0]",
	}
	return temp, repl, true
}

func ensureNewSliceHelperDecl(decls *[]unit.Decl, unitName string, typeName string) string {
	helper := newSliceHelperUnitName(unitName, typeName)
	for i := 0; i < len(*decls); i++ {
		if (*decls)[i].UnitName == helper {
			return helper
		}
	}
	words := helper + "_words"
	body := "func " + helper + "() *" + typeName + " {\n"
	body = body + "\t" + words + " := make([]int, 3)\n"
	body = body + "\treturn &" + words + "[0]\n"
	body = body + "}\n"
	*decls = append(*decls, unit.Decl{
		Path:     "rtg-new-slice",
		Kind:     "func",
		Name:     helper,
		UnitName: helper,
		Body:     body,
	})
	return helper
}

func newSliceHelperUnitName(unitName string, typeName string) string {
	return unitName + "_new_" + lowerIdentifierSuffix(typeName)
}

func lowerIdentifierSuffix(text string) string {
	out := make([]byte, 0, len(text)+12)
	var sum uint32
	for i := 0; i < len(text); i++ {
		c := text[i]
		sum = sum*33 + uint32(c)
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			out = append(out, c)
		} else {
			out = append(out, '_')
		}
	}
	out = append(out, '_')
	out = appendString(out, strconv.Itoa(int(sum)))
	return string(out)
}

func newTypeTextInRange(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "*" {
		inner := newTypeTextInRange(toks, start+1, end)
		if inner == "" {
			return ""
		}
		return "*" + inner
	}
	if start+1 < end && toks[start].Text == "[" && toks[start+1].Text == "]" {
		inner := newTypeTextInRange(toks, start+2, end)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	}
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		return toks[start].Text + "." + toks[start+2].Text
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text
	}
	return ""
}

func appendArgumentNeedsTemp(toks []scan.Token, expr expressionRange) bool {
	if expr.start >= expr.end {
		return false
	}
	if expressionIsSimpleAppendArgument(toks, expr.start, expr.end) {
		return false
	}
	return expressionContainsCall(toks, expr.start, expr.end) || expressionContainsTokenText(toks, expr.start, expr.end, "[") || expressionContainsTokenText(toks, expr.start, expr.end, ".") || expressionStartsCompositeLiteral(toks, expr.start, expr.end)
}

func expressionIsSimpleAppendArgument(toks []scan.Token, start int, end int) bool {
	if start+1 != end {
		return false
	}
	tok := toks[start]
	if tok.Kind == scan.Ident {
		return true
	}
	if tok.Kind == scan.Number || tok.Kind == scan.String || tok.Kind == scan.Char {
		return true
	}
	return tok.Text == "true" || tok.Text == "false" || tok.Text == "nil"
}

func expressionRangeContainsToken(toks []scan.Token, expr expressionRange, text string) bool {
	return expressionContainsTokenText(toks, expr.start, expr.end, text)
}

func expressionContainsTokenText(toks []scan.Token, start int, end int, text string) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == text {
			return true
		}
	}
	return false
}

func normalizeReturnValues(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, []expressionReplacement) {
	ranges := topLevelExpressionRanges(toks, start, end)
	if len(ranges) <= 1 {
		return normalizeExpressionWithContext(body, toks, start, end, unitName, tempIndex, namedSlices, namedConversions, ctx)
	}
	var temps []expressionTemp
	var replacements []expressionReplacement
	for i := 0; i < len(ranges); i++ {
		exprRange := ranges[i]
		exprTemps, exprReplacements := normalizeExpressionWithContext(body, toks, exprRange.start, exprRange.end, unitName, tempIndex, namedSlices, namedConversions, ctx)
		temps = appendExpressionTemps(temps, exprTemps)
		exprStart := int(toks[exprRange.start].Start)
		exprEnd := int(toks[exprRange.end-1].End)
		expr := applyExpressionReplacements(body, exprStart, exprEnd, exprReplacements)
		if expressionContainsNonConversionCall(toks, exprRange.start, exprRange.end) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, expr: expr})
			replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
			continue
		}
		replacements = appendExpressionReplacements(replacements, exprReplacements)
	}
	return temps, replacements
}

func topLevelExpressionRanges(toks []scan.Token, start int, end int) []expressionRange {
	var out []expressionRange
	exprStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "," {
			if exprStart < i {
				out = append(out, expressionRange{start: exprStart, end: i})
			}
			exprStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	if exprStart < end {
		out = append(out, expressionRange{start: exprStart, end: end})
	}
	return out
}

type reducibleComplexComponentCallInfo struct {
	outerClose  int
	complexPos  int
	args        []expressionRange
	selectedArg int
}

func reducibleComplexComponentReplacement(body string, toks []scan.Token, pos int, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) (string, int, int, bool) {
	literalInfo, literalOK := reducibleComplexLiteralComponentCallForLower(toks, pos)
	if literalOK {
		if complexComponentBuiltinShadowed(toks, pos, localNames, topNames, topFunctionNames) {
			return "", 0, 0, false
		}
		replacement := literalInfo.realText
		if literalInfo.selectedArg == 1 {
			replacement = literalInfo.imagText
		}
		return replacement, int(toks[literalInfo.outerClose].End), literalInfo.outerClose, true
	}
	info, ok := reducibleComplexComponentCallForLower(toks, pos)
	if !ok {
		return "", 0, 0, false
	}
	if complexComponentBuiltinShadowed(toks, pos, localNames, topNames, topFunctionNames) {
		return "", 0, 0, false
	}
	if complexComponentBuiltinShadowed(toks, info.complexPos, localNames, topNames, topFunctionNames) {
		return "", 0, 0, false
	}
	for i := 0; i < len(info.args); i++ {
		if !complexComponentOperandLowerableForLower(toks, info.args[i].start, info.args[i].end) {
			return "", 0, 0, false
		}
	}
	arg := info.args[info.selectedArg]
	start, end := trimTokenRange(toks, arg.start, arg.end)
	if start >= end {
		return "", 0, 0, false
	}
	return tokenRangeText(body, toks, start, end), int(toks[info.outerClose].End), info.outerClose, true
}

type reducibleComplexLiteralComponentCallInfo struct {
	outerClose  int
	realText    string
	imagText    string
	selectedArg int
}

func reducibleComplexLiteralComponentCallForLower(toks []scan.Token, pos int) (reducibleComplexLiteralComponentCallInfo, bool) {
	if pos+3 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	selected := -1
	if toks[pos].Text == "real" {
		selected = 0
	} else if toks[pos].Text == "imag" {
		selected = 1
	} else {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	if pos > 0 && toks[pos-1].Text == "." {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	outerClose := findClose(toks, pos+1, "(", ")")
	if outerClose < 0 {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	outerArgs := topLevelExpressionRanges(toks, pos+2, outerClose)
	if len(outerArgs) != 1 {
		return reducibleComplexLiteralComponentCallInfo{}, false
	}
	realText, imagText, ok := reducibleComplexLiteralPartsForLower(toks, outerArgs[0].start, outerArgs[0].end)
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

func reducibleComplexLiteralPartsForLower(toks []scan.Token, start int, end int) (string, string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return reducibleComplexLiteralPartsForLower(toks, start+1, close)
		}
	}
	text, imaginary, ok := signedNumberLiteralTextForLower(toks, start, end)
	if ok {
		if !imaginary {
			return "", "", false
		}
		return "0", stripImaginaryLiteralSuffixForLower(text), true
	}
	op := topLevelPlusMinusOperatorForLower(toks, start, end)
	if op < 0 {
		return "", "", false
	}
	leftText, leftImaginary, leftOK := signedNumberLiteralTextForLower(toks, start, op)
	rightText, rightImaginary, rightOK := signedNumberLiteralTextForLower(toks, op+1, end)
	if !leftOK || !rightOK || leftImaginary == rightImaginary {
		return "", "", false
	}
	if toks[op].Text == "-" {
		rightText = negateSignedNumberLiteralTextForLower(rightText)
	}
	if leftImaginary {
		return rightText, stripImaginaryLiteralSuffixForLower(leftText), true
	}
	return leftText, stripImaginaryLiteralSuffixForLower(rightText), true
}

func topLevelPlusMinusOperatorForLower(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := end - 1; i >= start; i-- {
		text := toks[i].Text
		updateExpressionDepthReverse(text, &paren, &brack, &brace)
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end && (text == "+" || text == "-") {
			return i
		}
	}
	return -1
}

func updateExpressionDepthReverse(text string, paren *int, brack *int, brace *int) {
	switch text {
	case ")":
		(*paren)++
	case "(":
		if *paren > 0 {
			(*paren)--
		}
	case "]":
		(*brack)++
	case "[":
		if *brack > 0 {
			(*brack)--
		}
	case "}":
		(*brace)++
	case "{":
		if *brace > 0 {
			(*brace)--
		}
	}
}

func signedNumberLiteralTextForLower(toks []scan.Token, start int, end int) (string, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
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
	return text, strings.HasSuffix(toks[start].Text, "i"), true
}

func stripImaginaryLiteralSuffixForLower(text string) string {
	if strings.HasSuffix(text, "i") {
		return text[:len(text)-1]
	}
	return text
}

func negateSignedNumberLiteralTextForLower(text string) string {
	if strings.HasPrefix(text, "-") {
		return text[1:]
	}
	return "-" + text
}

func normalizeReducibleComplexComponentCall(body string, toks []scan.Token, pos int, close int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, expressionReplacement, bool) {
	info, ok := reducibleComplexComponentCallForLower(toks, pos)
	if !ok || info.outerClose != close {
		return nil, expressionReplacement{}, false
	}
	var temps []expressionTemp
	selectedExpr := ""
	for argIndex := 0; argIndex < len(info.args); argIndex++ {
		arg := info.args[argIndex]
		argStart, argEnd := trimTokenRange(toks, arg.start, arg.end)
		if argStart >= argEnd {
			return nil, expressionReplacement{}, false
		}
		exprStart := int(toks[argStart].Start)
		exprEnd := int(toks[argEnd-1].End)
		argTemps, argReplacements := normalizeExpressionWithContext(body, toks, argStart, argEnd, unitName, tempIndex, namedSlices, namedConversions, ctx)
		temps = appendExpressionTemps(temps, argTemps)
		expr := strings.TrimSpace(applyExpressionReplacements(body, exprStart, exprEnd, argReplacements))
		if expressionContainsNonConversionCall(toks, argStart, argEnd) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			temps = append(temps, expressionTemp{name: name, expr: expr})
			expr = name
		}
		if argIndex == info.selectedArg {
			selectedExpr = expr
		}
	}
	if selectedExpr == "" {
		return nil, expressionReplacement{}, false
	}
	return temps, expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[info.outerClose].End),
		text:  selectedExpr,
	}, true
}

func complexComponentBuiltinShadowed(toks []scan.Token, pos int, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) bool {
	if pos < 0 || pos >= len(toks) {
		return true
	}
	name := toks[pos].Text
	sourcePos := int(toks[pos].Start)
	if isLocalNameAt(localNames, name, sourcePos) {
		return true
	}
	if symbolNameTableUnitName(topNames, name) != "" {
		return true
	}
	if symbolNameTableUnitName(topFunctionNames, name) != "" {
		return true
	}
	return false
}

func reducibleComplexComponentCallForLower(toks []scan.Token, pos int) (reducibleComplexComponentCallInfo, bool) {
	if pos+4 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return reducibleComplexComponentCallInfo{}, false
	}
	selected := -1
	if toks[pos].Text == "real" {
		selected = 0
	} else if toks[pos].Text == "imag" {
		selected = 1
	} else {
		return reducibleComplexComponentCallInfo{}, false
	}
	if pos > 0 && toks[pos-1].Text == "." {
		return reducibleComplexComponentCallInfo{}, false
	}
	outerClose := findClose(toks, pos+1, "(", ")")
	if outerClose < 0 {
		return reducibleComplexComponentCallInfo{}, false
	}
	outerArgs := topLevelExpressionRanges(toks, pos+2, outerClose)
	if len(outerArgs) != 1 {
		return reducibleComplexComponentCallInfo{}, false
	}
	innerStart, innerEnd := trimTokenRange(toks, outerArgs[0].start, outerArgs[0].end)
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
	args := topLevelExpressionRanges(toks, innerStart+2, innerClose)
	if len(args) != 2 {
		return reducibleComplexComponentCallInfo{}, false
	}
	return reducibleComplexComponentCallInfo{
		outerClose:  outerClose,
		complexPos:  innerStart,
		args:        args,
		selectedArg: selected,
	}, true
}

func complexComponentOperandLowerableForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return complexComponentOperandLowerableForLower(toks, start+1, close)
		}
	}
	if start+1 == end {
		return complexComponentOperandAtomForLower(toks[start])
	}
	if start+2 == end && (toks[start].Text == "+" || toks[start].Text == "-") {
		return complexComponentOperandAtomForLower(toks[start+1])
	}
	return false
}

func complexComponentOperandAtomForLower(tok scan.Token) bool {
	return tok.Kind == scan.Number || tok.Kind == scan.Char || tok.Kind == scan.String || tok.Kind == scan.Ident
}

func trimTokenRange(toks []scan.Token, start int, end int) (int, int) {
	for start < end && toks[start].Text == "," {
		start++
	}
	for end > start && toks[end-1].Text == "," {
		end--
	}
	return start, end
}

func normalizeIndexBounds(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	bounds := indexBoundRanges(toks, start, end)
	for i := 0; i < len(bounds); i++ {
		bound := bounds[i]
		if bound.start >= bound.end || !expressionContainsCall(toks, bound.start, bound.end) {
			continue
		}
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		exprStart := int(toks[bound.start].Start)
		exprEnd := int(toks[bound.end-1].End)
		expr := body[exprStart:exprEnd]
		temps = append(temps, expressionTemp{name: name, expr: expr})
		replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
	}
	return temps, replacements
}

type expressionRange struct {
	start int
	end   int
}

func topLevelAndOperands(toks []scan.Token, start int, end int) ([]expressionRange, bool) {
	var out []expressionRange
	operandStart := start
	paren := 0
	brack := 0
	brace := 0
	sawAnd := false
	for i := start; i < end; i++ {
		tok := toks[i]
		if paren == 0 && brack == 0 && brace == 0 {
			if tok.Text == "||" {
				return nil, false
			}
			if tok.Text == "&&" {
				if operandStart >= i {
					return nil, false
				}
				out = append(out, expressionRange{start: operandStart, end: i})
				operandStart = i + 1
				sawAnd = true
				continue
			}
		}
		updateExpressionDepth(tok.Text, &paren, &brack, &brace)
	}
	if !sawAnd || operandStart >= end {
		return nil, false
	}
	out = append(out, expressionRange{start: operandStart, end: end})
	return out, true
}

func lowerArrayStructFieldTypes(body string) string {
	if !strings.Contains(body, "[") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, ok := arrayStructFieldTypeReplacement(body, toks, i)
		if ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		}
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func arrayStructFieldTypeReplacement(body string, toks []scan.Token, pos int) (expressionReplacement, bool) {
	if pos <= 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return expressionReplacement{}, false
	}
	if !nameInStructFieldListForLower(toks, pos-1) {
		return expressionReplacement{}, false
	}
	end := structFieldSpecEndForLower(toks, pos)
	typeEnd := end
	if typeEnd > pos && toks[typeEnd-1].Kind == scan.String {
		typeEnd--
	}
	info, _, _, ok := arrayTypeInfoForRange(body, toks, pos, typeEnd)
	if !ok || info.inferred {
		return expressionReplacement{}, false
	}
	return expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[typeEnd-1].End),
		text:  "[]" + info.elem,
	}, true
}

func lowerArrayFunctionParameterTypes(body string) string {
	if !strings.Contains(body, "[") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, ok := arrayFunctionParameterTypeReplacement(body, toks, i)
		if ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		}
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func lowerArrayFunctionResultTypes(body string) string {
	if !strings.Contains(body, "[") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, ok := arrayFunctionResultTypeReplacement(body, toks, i)
		if ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		}
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func lowerGroupedFunctionParameterTypes(body string) string {
	if !strings.Contains(body, ",") {
		return body
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil || !tokenSpansMatchSource(body, toks) {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "(" || !functionParameterListOpenForLower(toks, i) {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 {
			continue
		}
		repl, ok := groupedFunctionParameterListReplacement(body, toks, i, close)
		if ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		}
		i = close
	}
	if len(replacements) == 0 {
		return body
	}
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

type pendingParameterForLower struct {
	name string
	text string
}

func groupedFunctionParameterListReplacement(body string, toks []scan.Token, open int, close int) (expressionReplacement, bool) {
	var parts []string
	var pending []pendingParameterForLower
	changed := false
	segmentStart := open + 1
	paren := 0
	brack := 0
	brace := 0
	for i := open + 1; i <= close; i++ {
		if i == close || (paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ",") {
			part, nextPending, flush, ok := groupedFunctionParameterSegment(body, toks, segmentStart, i)
			if ok && flush {
				pending = append(pending, nextPending)
				for pendingIndex := 0; pendingIndex < len(pending); pendingIndex++ {
					parts = append(parts, pending[pendingIndex].name+" "+part)
				}
				if len(pending) > 1 {
					changed = true
				}
				pending = nil
			} else if ok {
				pending = append(pending, nextPending)
			} else {
				parts = appendPendingParameterTexts(parts, pending)
				pending = nil
				if strings.TrimSpace(part) != "" {
					parts = append(parts, part)
				}
			}
			segmentStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	parts = appendPendingParameterTexts(parts, pending)
	if !changed {
		return expressionReplacement{}, false
	}
	return expressionReplacement{
		start: int(toks[open].End),
		end:   int(toks[close].Start),
		text:  strings.Join(parts, ", "),
	}, true
}

func groupedFunctionParameterSegment(body string, toks []scan.Token, start int, end int) (string, pendingParameterForLower, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", pendingParameterForLower{}, false, false
	}
	text := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	if toks[start].Kind != scan.Ident {
		return text, pendingParameterForLower{}, false, false
	}
	if start+1 < end && isTypeStart(toks[start+1]) {
		typ := strings.TrimSpace(body[int(toks[start+1].Start):int(toks[end-1].End)])
		return typ, pendingParameterForLower{name: toks[start].Text, text: text}, true, true
	}
	if start+1 == end && !isBuiltinTypeNameForLower(toks[start].Text) {
		return "", pendingParameterForLower{name: toks[start].Text, text: text}, false, true
	}
	return text, pendingParameterForLower{}, false, false
}

func appendPendingParameterTexts(parts []string, pending []pendingParameterForLower) []string {
	for i := 0; i < len(pending); i++ {
		if pending[i].text != "" {
			parts = append(parts, pending[i].text)
		}
	}
	return parts
}

func isBuiltinTypeNameForLower(name string) bool {
	switch name {
	case "int", "int64", "byte", "bool", "string", "float64", "int16", "int32", "error":
		return true
	}
	return false
}

func arrayFunctionResultTypeReplacement(body string, toks []scan.Token, pos int) (expressionReplacement, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return expressionReplacement{}, false
	}
	if arrayTypeStartsDirectFunctionResultForLower(toks, pos) {
		end := functionParameterArrayTypeEndForLower(toks, pos)
		return arrayFunctionSignatureTypeReplacement(body, toks, pos, end)
	}
	open := containingOpenForLower(toks, pos, "(", ")")
	if !functionResultListOpenForLower(toks, open) {
		return expressionReplacement{}, false
	}
	if !arrayTypeStartsFunctionParameterTypeForLower(toks, open, pos) {
		return expressionReplacement{}, false
	}
	end := functionParameterArrayTypeEndForLower(toks, pos)
	return arrayFunctionSignatureTypeReplacement(body, toks, pos, end)
}

func arrayTypeStartsDirectFunctionResultForLower(toks []scan.Token, pos int) bool {
	if pos <= 0 || toks[pos-1].Text != ")" {
		return false
	}
	paramsOpen := findOpen(toks, pos-1, "(", ")")
	return functionParameterListOpenForLower(toks, paramsOpen)
}

func functionResultListOpenForLower(toks []scan.Token, open int) bool {
	if open <= 0 || toks[open-1].Text != ")" {
		return false
	}
	paramsOpen := findOpen(toks, open-1, "(", ")")
	return functionParameterListOpenForLower(toks, paramsOpen)
}

func arrayFunctionParameterTypeReplacement(body string, toks []scan.Token, pos int) (expressionReplacement, bool) {
	if pos < 0 || pos >= len(toks) || toks[pos].Text != "[" {
		return expressionReplacement{}, false
	}
	open := containingOpenForLower(toks, pos, "(", ")")
	if !functionParameterListOpenForLower(toks, open) {
		return expressionReplacement{}, false
	}
	if !arrayTypeStartsFunctionParameterTypeForLower(toks, open, pos) {
		return expressionReplacement{}, false
	}
	end := functionParameterArrayTypeEndForLower(toks, pos)
	return arrayFunctionSignatureTypeReplacement(body, toks, pos, end)
}

func arrayFunctionSignatureTypeReplacement(body string, toks []scan.Token, pos int, end int) (expressionReplacement, bool) {
	info, _, _, ok := arrayTypeInfoForRange(body, toks, pos, end)
	if !ok || info.inferred {
		return expressionReplacement{}, false
	}
	return expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[end-1].End),
		text:  "[]" + info.elem,
	}, true
}

func functionParameterListOpenForLower(toks []scan.Token, open int) bool {
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

func arrayTypeStartsFunctionParameterTypeForLower(toks []scan.Token, open int, pos int) bool {
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

func functionParameterArrayTypeEndForLower(toks []scan.Token, pos int) int {
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
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return len(toks)
}

func containingOpenForLower(toks []scan.Token, pos int, openText string, closeText string) int {
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

func indexBoundRanges(toks []scan.Token, start int, end int) []expressionRange {
	if start >= end {
		return nil
	}
	var out []expressionRange
	boundStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ":" {
			out = append(out, expressionRange{start: boundStart, end: i})
			boundStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	out = append(out, expressionRange{start: boundStart, end: end})
	return out
}

func normalizeOneCallArguments(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int, namedSlices []namedSliceInfo, namedConversions []string, ctx normalizationContext) ([]expressionTemp, []expressionReplacement) {
	var temps []expressionTemp
	var replacements []expressionReplacement
	argStart := start
	paren := 0
	brack := 0
	brace := 0
	for i := start; i <= end; i++ {
		if i == end || (paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ",") {
			if argStart < i {
				if expressionStartsCompositeLiteral(toks, argStart, i) && tokensAreCompositeLiteral(toks, argStart, i) {
					argStart = i + 1
					continue
				}
				argTemps, argReplacements := normalizeExpressionWithContext(body, toks, argStart, i, unitName, tempIndex, namedSlices, namedConversions, ctx)
				temps = appendExpressionTemps(temps, argTemps)
				exprStart := int(toks[argStart].Start)
				exprEnd := int(toks[i-1].End)
				expr := applyExpressionReplacements(body, exprStart, exprEnd, argReplacements)
				if !expressionContainsCall(toks, argStart, i) {
					replacements = appendExpressionReplacements(replacements, argReplacements)
					argStart = i + 1
					continue
				}
				name := nextExpressionTempName(body, unitName, tempIndex)
				(*tempIndex)++
				temps = append(temps, expressionTemp{name: name, expr: expr})
				replacements = append(replacements, expressionReplacement{start: exprStart, end: exprEnd, text: name})
			}
			argStart = i + 1
			continue
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return temps, replacements
}

func expressionStartsCompositeLiteral(toks []scan.Token, start int, end int) bool {
	if start+1 >= end || toks[start].Kind != scan.Ident {
		return false
	}
	if toks[start+1].Text == "{" {
		return true
	}
	if start+3 < end && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident && toks[start+3].Text == "{" {
		return true
	}
	return false
}

func nextExpressionTempName(body string, unitName string, tempIndex *int) string {
	for {
		name := unitName + "_tmp_" + strconv.Itoa(*tempIndex)
		if !strings.Contains(body, name) {
			return name
		}
		(*tempIndex)++
	}
}

func updateExpressionDepth(text string, paren *int, brack *int, brace *int) {
	switch text {
	case "(":
		(*paren)++
	case ")":
		if *paren > 0 {
			(*paren)--
		}
	case "[":
		(*brack)++
	case "]":
		if *brack > 0 {
			(*brack)--
		}
	case "{":
		(*brace)++
	case "}":
		if *brace > 0 {
			(*brace)--
		}
	}
}

func expressionContainsCall(toks []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if toks[i].Kind == scan.Ident && toks[i+1].Text == "(" {
			return true
		}
	}
	return false
}

func expressionContainsNonConversionCall(toks []scan.Token, start int, end int) bool {
	for i := start; i+1 < end; i++ {
		if toks[i].Kind == scan.Ident && toks[i+1].Text == "(" && !isBuiltinConversionName(toks[i].Text) {
			return true
		}
	}
	return false
}

func isBuiltinConversionName(name string) bool {
	if name == "int" {
		return true
	}
	if name == "int16" {
		return true
	}
	if name == "int32" {
		return true
	}
	if name == "int64" {
		return true
	}
	if name == "byte" {
		return true
	}
	if name == "bool" {
		return true
	}
	if name == "string" {
		return true
	}
	if name == "error" {
		return true
	}
	if name == "float64" {
		return true
	}
	return false
}

func expressionContainsTopLevelSemicolon(toks []scan.Token, start int, end int) bool {
	return topLevelSemicolon(toks, start, end) >= 0
}

func topLevelSemicolon(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == ";" {
			return i
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return -1
}

func conditionalStatementEnd(toks []scan.Token, pos int, openBrace int) int {
	closeBrace := findClose(toks, openBrace, "{", "}")
	if closeBrace < 0 {
		return -1
	}
	if toks[pos].Text != "if" {
		return closeBrace
	}
	if closeBrace+1 >= len(toks) || toks[closeBrace+1].Text != "else" {
		return closeBrace
	}
	next := closeBrace + 2
	if next >= len(toks) {
		return closeBrace
	}
	if toks[next].Text == "if" {
		nextOpen := conditionExpressionEnd(toks, next)
		if nextOpen >= len(toks) || toks[nextOpen].Text != "{" {
			return closeBrace
		}
		end := conditionalStatementEnd(toks, next, nextOpen)
		if end >= 0 {
			return end
		}
		return closeBrace
	}
	if toks[next].Text == "{" {
		end := findClose(toks, next, "{", "}")
		if end >= 0 {
			return end
		}
	}
	return closeBrace
}

func applyExpressionReplacements(body string, start int, end int, replacements []expressionReplacement) string {
	out := make([]byte, 0, rewriteBufferCapacity(end-start))
	cursor := start
	for i := 0; i < len(replacements); i++ {
		repl := replacements[i]
		if repl.start < cursor || repl.end > end {
			continue
		}
		out = appendStringRange(out, body, cursor, repl.start)
		replText := repl.text
		out = appendString(out, replText)
		cursor = repl.end
	}
	out = appendStringRange(out, body, cursor, end)
	return string(out)
}

func statementIndent(body string, pos int) string {
	lineStart := pos
	for lineStart > 0 && body[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < pos; i++ {
		if body[i] != ' ' && body[i] != '\t' {
			return "\t"
		}
	}
	indent := body[lineStart:pos]
	if indent == "" {
		return "\t"
	}
	return indent
}

func appendLabelLinePrefix(out []byte, indent string) []byte {
	for i := len(out) - 1; i >= 0; i-- {
		if out[i] == '\n' {
			return out
		}
		if out[i] != ' ' && out[i] != '\t' {
			out = append(out, '\n')
			return appendString(out, indent)
		}
	}
	return out
}

func statementInsertStart(body string, pos int) int {
	lineStart := pos
	for lineStart > 0 && body[lineStart-1] != '\n' {
		lineStart--
	}
	for i := lineStart; i < pos; i++ {
		if body[i] != ' ' && body[i] != '\t' {
			return pos
		}
	}
	return lineStart
}

func localNamesForDecl(file *parse.File, decl *parse.Decl, namesOfInterest symbolNameTable) localNameTable {
	var names localNameTable
	if decl.Kind != "func" {
		return names
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return names
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return names
	}
	collectFuncSignatureLocals(toks, start, body, namesOfInterest, &names)
	for i := body + 1; i < len(toks) && int(toks[i].Start) < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocals(toks, body, i, decl.End, namesOfInterest, &names)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocals(toks, body, i, decl.End, namesOfInterest, &names)
		}
	}
	return names
}

func importLocalNameTableForDecl(file *parse.File, decl *parse.Decl, importRefs importSymbolTable) localNameTable {
	var importNames symbolNameTable
	for i := 0; i < len(importRefs); i++ {
		name := importRefs[i].localName
		if name == "." {
			symbols := importRefs[i].symbols
			for symbolIndex := 0; symbolIndex < len(symbols); symbolIndex++ {
				importNames = symbolNameTableSet(importNames, symbols[symbolIndex].Name, symbols[symbolIndex].Name)
			}
			continue
		}
		importNames = symbolNameTableSet(importNames, name, name)
	}
	return localNamesForDecl(file, decl, importNames)
}

func localTypesForDecl(file *parse.File, decl *parse.Decl) localTypeTable {
	var types localTypeTable
	if decl.Kind != "func" {
		return types
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return types
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return types
	}
	collectFuncSignatureLocalTypes(toks, start, body, &types)
	for i := body + 1; i < len(toks) && int(toks[i].Start) < decl.End; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclLocalTypes(toks, i, &types)
			continue
		}
		if toks[i].Text == "var" {
			collectVarLocalTypes(toks, i, decl.End, &types)
		}
	}
	return types
}

func collectImportedFunctionResultLocalTypesForDecl(file *parse.File, decl *parse.Decl, types localTypeTable, importRefs importSymbolTable, importLocalNames localNameTable, functionResults localTypeTable) localTypeTable {
	if decl.Kind != "func" {
		return types
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return types
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return types
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		close = tokenIndexBeforeForLower(toks, decl.End)
	}
	for i := body + 1; i < len(toks) && i < close; i++ {
		if toks[i].Text == ":=" {
			if i-1 < 0 || i+1 >= len(toks) || toks[i-1].Kind != scan.Ident {
				continue
			}
			initEnd := shortDeclInitializerEnd(toks, i+1)
			typ := localInitializerTypeWithImportedFunctions(toks, i+1, initEnd, types, functionResults, importRefs, importLocalNames)
			if typ.name != "" {
				types = localTypeTableSet(types, toks[i-1].Text, typ)
			}
			continue
		}
		if toks[i].Text == "var" {
			if i+3 >= len(toks) || toks[i+1].Kind != scan.Ident || toks[i+2].Text != "=" {
				continue
			}
			initEnd := varInitializerEnd(toks, i+3, decl.End)
			typ := localInitializerTypeWithImportedFunctions(toks, i+3, initEnd, types, functionResults, importRefs, importLocalNames)
			if typ.name != "" {
				types = localTypeTableSet(types, toks[i+1].Text, typ)
			}
		}
	}
	return types
}

func sizeofLocalTypesForLoadFiles(files []load.File) sizeofLocalTypeTable {
	var types sizeofLocalTypeTable
	for i := 0; i < len(files); i++ {
		file, err := parsedLoadFile(files[i])
		if err != nil {
			continue
		}
		collectPackageSizeofTypes(&file, &types)
	}
	return types
}

func sizeofLocalTypesForDecl(file *parse.File, decl *parse.Decl, packageTypes sizeofLocalTypeTable) sizeofLocalTypeTable {
	types := cloneSizeofLocalTypes(packageTypes)
	if decl.Kind != "func" {
		return types
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return types
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return types
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		close = tokenIndexBeforeForLower(toks, decl.End)
	}
	collectFuncSignatureSizeofTypes(toks, start, body, &types)
	for i := body + 1; i < len(toks) && i < close; i++ {
		if toks[i].Text == ":=" {
			collectShortDeclSizeofTypes(toks, body, i, close, &types)
			continue
		}
		if toks[i].Text == "var" {
			collectVarSizeofTypes(toks, body, i, close, &types)
		}
	}
	return types
}

func sizeofNamedTypesForLoadFiles(files []load.File) sizeofNamedTypeTable {
	var types sizeofNamedTypeTable
	for i := 0; i < len(files); i++ {
		file, err := parsedLoadFile(files[i])
		if err != nil {
			continue
		}
		collectPackageSizeofNamedTypes(&file, &types)
	}
	return types
}

func sizeofNamedTypesForDecl(file *parse.File, decl *parse.Decl, packageTypes sizeofNamedTypeTable) sizeofNamedTypeTable {
	types := cloneSizeofNamedTypes(packageTypes)
	if decl.Kind != "func" {
		return types
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return types
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return types
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		close = tokenIndexBeforeForLower(toks, decl.End)
	}
	collectLocalSizeofNamedTypes(toks, body, close, &types)
	return types
}

func collectPackageSizeofNamedTypes(file *parse.File, types *sizeofNamedTypeTable) {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "type" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		end := tokenIndexBeforeForLower(toks, decl.End) + 1
		if end <= start {
			continue
		}
		if start+1 < end && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 || close >= end {
				continue
			}
			ranges := localConstSpecRanges(toks, start+2, close)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				*types = appendSizeofNamedTypeSpec(*types, toks, spec.start, spec.end, 0, maxSourcePosition())
			}
			continue
		}
		*types = appendSizeofNamedTypeSpec(*types, toks, start+1, end, 0, maxSourcePosition())
	}
}

func collectLocalSizeofNamedTypes(toks []scan.Token, body int, close int, types *sizeofNamedTypeTable) {
	for i := body + 1; i < len(toks) && i < close; i++ {
		if toks[i].Text != "type" || !startsLocalTypeDeclToken(toks, i, body) {
			continue
		}
		scopeEnd := localScopeEnd(toks, body, i, int(toks[close].End))
		if i+1 < close && toks[i+1].Text == "(" {
			groupClose := findClose(toks, i+1, "(", ")")
			if groupClose < 0 || groupClose > close {
				continue
			}
			ranges := localConstSpecRanges(toks, i+2, groupClose)
			for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
				spec := ranges[rangeIndex]
				*types = appendSizeofNamedTypeSpec(*types, toks, spec.start, spec.end, int(toks[i].Start), scopeEnd)
			}
			i = groupClose
			continue
		}
		specEnd := localTypeSingleSpecEnd(toks, i, int(toks[close].End))
		*types = appendSizeofNamedTypeSpec(*types, toks, i+1, specEnd, int(toks[i].Start), scopeEnd)
		if specEnd > i {
			i = specEnd - 1
		}
	}
}

func collectPackageSizeofTypes(file *parse.File, types *sizeofLocalTypeTable) {
	toks := file.Tokens
	for declIndex := 0; declIndex < len(file.Decls); declIndex++ {
		decl := file.Decls[declIndex]
		if decl.Kind != "var" {
			continue
		}
		start := tokenIndexAt(toks, decl.Start)
		if start < 0 {
			continue
		}
		end := tokenIndexBeforeForLower(toks, decl.End) + 1
		if end <= start {
			continue
		}
		if start+1 < end && toks[start+1].Text == "(" {
			close := findClose(toks, start+1, "(", ")")
			if close < 0 || close >= end {
				continue
			}
			specStart := start + 2
			for i := specStart; i <= close; i++ {
				if i == close || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
					collectPackageVarSizeofSpecTypes(toks, specStart, i, types)
					if i < close && toks[i].Text == ";" {
						specStart = i + 1
					} else {
						specStart = i
					}
				}
			}
			continue
		}
		collectPackageVarSizeofSpecTypes(toks, start+1, end, types)
	}
}

func collectPackageVarSizeofSpecTypes(toks []scan.Token, start int, end int, types *sizeofLocalTypeTable) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return
	}
	eq := findTopLevelToken(toks, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	var names []string
	typeStart := -1
	for i := start; i < lhsEnd; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			names = appendStringUnique(names, toks[i].Text)
			continue
		}
		if isTypeStart(toks[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return
	}
	if typeStart >= 0 {
		typ := sizeofTypeTextInRange(toks, typeStart, lhsEnd)
		for i := 0; i < len(names); i++ {
			*types = appendSizeofLocalType(*types, names[i], typ, 0, maxSourcePosition())
		}
		return
	}
	if eq < 0 {
		return
	}
	rhs := topLevelExpressionRanges(toks, eq+1, end)
	if len(rhs) != len(names) {
		return
	}
	for i := 0; i < len(names); i++ {
		typ := inferSizeofTypeText(toks, rhs[i].start, rhs[i].end, *types)
		if typ == "" {
			continue
		}
		*types = appendSizeofLocalType(*types, names[i], typ, 0, maxSourcePosition())
	}
}

func collectFuncSignatureSizeofTypes(toks []scan.Token, start int, end int, types *sizeofLocalTypeTable) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterListSizeofTypes(toks, i+1, close, types)
		i = close
	}
}

func collectParameterListSizeofTypes(toks []scan.Token, start int, end int, types *sizeofLocalTypeTable) {
	var pending []string
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == "," {
			name, typ := parameterSizeofSegment(toks, segmentStart, i)
			if typ != "" {
				if name != "" {
					pending = appendStringUnique(pending, name)
				}
				for j := 0; j < len(pending); j++ {
					*types = appendSizeofLocalType(*types, pending[j], typ, 0, maxSourcePosition())
				}
				if len(pending) == 0 && name != "" {
					*types = appendSizeofLocalType(*types, name, typ, 0, maxSourcePosition())
				}
				pending = nil
			} else if name != "" {
				pending = appendStringUnique(pending, name)
			}
			segmentStart = i + 1
		}
	}
}

func parameterSizeofSegment(toks []scan.Token, start int, end int) (string, string) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end || toks[start].Kind != scan.Ident {
		return "", ""
	}
	if start+1 < end && isTypeStart(toks[start+1]) {
		return toks[start].Text, sizeofTypeTextInRange(toks, start+1, end)
	}
	if start+1 == end {
		return toks[start].Text, ""
	}
	return "", ""
}

func collectShortDeclSizeofTypes(toks []scan.Token, body int, assign int, limit int, types *sizeofLocalTypeTable) {
	stmtStart := simpleStatementStartForLower(toks, body, assign)
	for stmtStart < assign && (toks[stmtStart].Text == "for" || toks[stmtStart].Text == "if" || toks[stmtStart].Text == "switch") {
		stmtStart++
	}
	stmtEnd := lowerSimpleStatementEnd(toks, assign+1, limit)
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return
	}
	scopeStart := int(toks[assign].End)
	scopeEnd := localScopeEnd(toks, body, assign, int(toks[limit].End))
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		typ := inferSizeofTypeText(toks, rhs[i].start, rhs[i].end, *types)
		if typ == "" {
			continue
		}
		*types = appendSizeofLocalType(*types, name, typ, scopeStart, scopeEnd)
	}
}

func collectVarSizeofTypes(toks []scan.Token, body int, pos int, limit int, types *sizeofLocalTypeTable) {
	if pos+1 >= limit {
		return
	}
	if toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close >= limit {
			return
		}
		specStart := pos + 2
		for i := specStart; i <= close; i++ {
			if i == close || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
				collectVarSizeofSpecTypes(toks, body, specStart, i, limit, types)
				if i < close && toks[i].Text == ";" {
					specStart = i + 1
				} else {
					specStart = i
				}
			}
		}
		return
	}
	end := lowerSimpleStatementEnd(toks, pos+1, limit)
	collectVarSizeofSpecTypes(toks, body, pos+1, end, limit, types)
}

func collectVarSizeofSpecTypes(toks []scan.Token, body int, start int, end int, limit int, types *sizeofLocalTypeTable) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return
	}
	eq := findTopLevelToken(toks, start, end, "=")
	lhsEnd := end
	if eq >= 0 {
		lhsEnd = eq
	}
	var names []string
	typeStart := -1
	for i := start; i < lhsEnd; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			names = appendStringUnique(names, toks[i].Text)
			continue
		}
		if isTypeStart(toks[i]) {
			typeStart = i
			break
		}
	}
	if len(names) == 0 {
		return
	}
	scopeStart := int(toks[end-1].End)
	scopeEnd := localScopeEnd(toks, body, start, int(toks[limit].End))
	if typeStart >= 0 {
		typ := sizeofTypeTextInRange(toks, typeStart, lhsEnd)
		for i := 0; i < len(names); i++ {
			*types = appendSizeofLocalType(*types, names[i], typ, scopeStart, scopeEnd)
		}
		return
	}
	if eq < 0 {
		return
	}
	rhs := topLevelExpressionRanges(toks, eq+1, end)
	if len(rhs) != len(names) {
		return
	}
	scopeStart = int(toks[eq].End)
	for i := 0; i < len(names); i++ {
		typ := inferSizeofTypeText(toks, rhs[i].start, rhs[i].end, *types)
		if typ == "" {
			continue
		}
		*types = appendSizeofLocalType(*types, names[i], typ, scopeStart, scopeEnd)
	}
}

func appendSizeofLocalType(types sizeofLocalTypeTable, name string, typ string, start int, end int) sizeofLocalTypeTable {
	if name == "" || name == "_" || typ == "" {
		return types
	}
	return append(types, sizeofLocalType{name: name, typ: typ, start: start, end: end})
}

func sizeofLocalTypeLookup(types sizeofLocalTypeTable, name string, pos int) string {
	for i := len(types) - 1; i >= 0; i-- {
		entry := types[i]
		if entry.name != name {
			continue
		}
		if pos < entry.start {
			continue
		}
		if entry.end > 0 && pos >= entry.end {
			continue
		}
		return entry.typ
	}
	return ""
}

func cloneSizeofLocalTypes(types sizeofLocalTypeTable) sizeofLocalTypeTable {
	if len(types) == 0 {
		return nil
	}
	out := make(sizeofLocalTypeTable, len(types))
	copy(out, types)
	return out
}

func appendSizeofNamedTypeSpec(types sizeofNamedTypeTable, toks []scan.Token, start int, end int, scopeStart int, scopeEnd int) sizeofNamedTypeTable {
	start, end = trimTokenRange(toks, start, end)
	for start < end && toks[start].Text == ";" {
		start++
	}
	for end > start && toks[end-1].Text == ";" {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return types
	}
	name := toks[start].Text
	typeStart := start + 1
	if typeStart < end && toks[typeStart].Text == "=" {
		typeStart++
	}
	typ := sizeofTypeTextInRange(toks, typeStart, end)
	if name == "" || name == "_" || typ == "" || typ == name {
		return types
	}
	return append(types, sizeofNamedType{name: name, typ: typ, start: scopeStart, end: scopeEnd})
}

func sizeofNamedTypeLookup(types sizeofNamedTypeTable, name string, pos int) string {
	for i := len(types) - 1; i >= 0; i-- {
		entry := types[i]
		if entry.name != name {
			continue
		}
		if pos < entry.start {
			continue
		}
		if entry.end > 0 && pos >= entry.end {
			continue
		}
		return entry.typ
	}
	return ""
}

func cloneSizeofNamedTypes(types sizeofNamedTypeTable) sizeofNamedTypeTable {
	if len(types) == 0 {
		return nil
	}
	out := make(sizeofNamedTypeTable, len(types))
	copy(out, types)
	return out
}

func sizeofTypeTextInRange(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	var out []byte
	for i := start; i < end; i++ {
		out = appendString(out, toks[i].Text)
	}
	return string(out)
}

func inferSizeofTypeText(toks []scan.Token, start int, end int, types sizeofLocalTypeTable) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return inferSizeofTypeText(toks, start+1, close, types)
		}
	}
	if start+1 == end {
		tok := toks[start]
		if tok.Kind == scan.String {
			return "string"
		}
		if tok.Text == "true" || tok.Text == "false" {
			return "bool"
		}
		if tok.Kind == scan.Number || tok.Kind == scan.Char {
			return "int"
		}
		if tok.Kind == scan.Ident {
			return sizeofLocalTypeLookup(types, tok.Text, int(tok.Start))
		}
	}
	if toks[start].Text == "&" && start+1 < end {
		return "*byte"
	}
	if typ := unsafeSizeofSliceLiteralTypeText(toks, start, end); typ != "" {
		return typ
	}
	if typ := unsafeSizeofArrayLiteralTypeText(toks, start, end); typ != "" {
		return typ
	}
	if typ := unsafeSizeofCompositeLiteralTypeText(toks, start, end); typ != "" {
		return typ
	}
	if typ := unsafeSizeofConversionTypeText(toks, start, end); typ != "" {
		return typ
	}
	return ""
}

func unsafeSizeofArrayLiteralTypeText(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start+4 > end || toks[start].Text != "[" {
		return ""
	}
	open := arrayCompositeLiteralOpenForLower(toks, start)
	if open <= start || open >= end {
		return ""
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return ""
	}
	info, ok := fixedArrayTypeLowerInfoForTokens(toks, start, open)
	if !ok {
		return ""
	}
	return arrayTypeLowerInfoText(info)
}

func unsafeSizeofSliceLiteralTypeText(toks []scan.Token, start int, end int) string {
	if start+3 >= end || toks[start].Text != "[" || toks[start+1].Text != "]" {
		return ""
	}
	paren := 0
	brack := 0
	brace := 0
	for i := start + 2; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "{" {
			close := findClose(toks, i, "{", "}")
			if close == end-1 {
				return sizeofTypeTextInRange(toks, start, i)
			}
			return ""
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return ""
}

func unsafeSizeofConversionTypeText(toks []scan.Token, start int, end int) string {
	if start+3 > end {
		return ""
	}
	if toks[start].Kind == scan.Ident && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			return toks[start].Text
		}
	}
	if toks[start].Text == "[" && start+1 < end && toks[start+1].Text == "]" {
		paren := 0
		brack := 0
		brace := 0
		for i := start + 2; i < end; i++ {
			if paren == 0 && brack == 0 && brace == 0 && toks[i].Text == "(" {
				close := findClose(toks, i, "(", ")")
				if close == end-1 {
					return sizeofTypeTextInRange(toks, start, i)
				}
				return ""
			}
			updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
		}
	}
	return ""
}

func unsafeSizeofCompositeLiteralTypeText(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	open := compositeLiteralOpenForTypeStart(toks, start)
	if open <= start || open >= end {
		return ""
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return ""
	}
	return sizeofTypeTextInRange(toks, start, open)
}

func unsafeSizeofSelectorReplacement(toks []scan.Token, pos int, types sizeofLocalTypeTable, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, wordSize int) (string, int, int, bool) {
	if pos+3 >= len(toks) || toks[pos+3].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, pos+3, "(", ")")
	if close <= pos+3 {
		return "", 0, 0, false
	}
	args := topLevelExpressionRanges(toks, pos+4, close)
	if len(args) != 1 {
		return "", 0, 0, false
	}
	size, ok := unsafeSizeofOperandSizeForLower(toks, args[0].start, args[0].end, types, namedTypes, fieldTypes, arrayFieldTypes, wordSize)
	if !ok {
		return "", 0, 0, false
	}
	return strconv.Itoa(size), int(toks[close].End), close, true
}

func unsafeSizeofDotReplacement(toks []scan.Token, pos int, types sizeofLocalTypeTable, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, wordSize int) (string, int, int, bool) {
	if pos+1 >= len(toks) || toks[pos+1].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, pos+1, "(", ")")
	if close <= pos+1 {
		return "", 0, 0, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 1 {
		return "", 0, 0, false
	}
	size, ok := unsafeSizeofOperandSizeForLower(toks, args[0].start, args[0].end, types, namedTypes, fieldTypes, arrayFieldTypes, wordSize)
	if !ok {
		return "", 0, 0, false
	}
	return strconv.Itoa(size), int(toks[close].End), close, true
}

func unsafeSizeofOperandSizeForLower(toks []scan.Token, start int, end int, types sizeofLocalTypeTable, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, wordSize int) (int, bool) {
	typ := inferSizeofTypeText(toks, start, end, types)
	pos := 0
	if start >= 0 && start < len(toks) {
		pos = int(toks[start].Start)
	}
	return unsafeSizeofFixedTypeSizeForLower(typ, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize)
}

func unsafeSizeofFixedTypeSizeForLower(typ string, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int, wordSize int) (int, bool) {
	size, _, ok := unsafeSizeofTypeLayoutForLower(typ, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, nil)
	return size, ok
}

func unsafeSizeofTypeLayoutForLower(typ string, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int, wordSize int, seen []string) (int, int, bool) {
	typ = strings.TrimSpace(typ)
	if typ == "" {
		return 0, 0, false
	}
	if containsString(seen, typ) {
		return 0, 0, false
	}
	if size, align, ok := unsafeSizeofFixedArrayTypeLayoutForLower(typ, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, seen); ok {
		return size, align, true
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
		return 8, unsafeSizeofMinAlignForLower(8, wordSize), true
	case "string":
		return wordSize * 2, wordSize, true
	}
	if size, align, ok := unsafeSizeofStructTypeLayoutForLower(typ, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, seen); ok {
		return size, align, true
	}
	resolved := unsafeSizeofResolveNamedTypeForLower(typ, namedTypes, pos)
	if resolved != typ {
		return unsafeSizeofTypeLayoutForLower(resolved, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, append(cloneStrings(seen), typ))
	}
	return 0, 0, false
}

func unsafeSizeofFixedArrayTypeLayoutForLower(typ string, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int, wordSize int, seen []string) (int, int, bool) {
	if len(typ) < 4 || typ[0] != '[' {
		return 0, 0, false
	}
	close := strings.IndexByte(typ, ']')
	if close <= 1 || close+1 >= len(typ) {
		return 0, 0, false
	}
	length, err := strconv.ParseInt(typ[1:close], 0, 64)
	if err != nil || length < 0 {
		return 0, 0, false
	}
	elem := typ[close+1:]
	if elem == "" {
		return 0, 0, false
	}
	elemSize, elemAlign, ok := unsafeSizeofTypeLayoutForLower(elem, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, seen)
	if !ok {
		return 0, 0, false
	}
	return int(length) * elemSize, elemAlign, true
}

func unsafeSizeofStructTypeLayoutForLower(typ string, namedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, arrayFieldTypes arrayStructFieldLowerInfoTable, pos int, wordSize int, seen []string) (int, int, bool) {
	if typ == "" || containsString(seen, typ) || !structFieldTypeTableOwnerExists(fieldTypes, typ) {
		return 0, 0, false
	}
	offset := 0
	maxAlign := 1
	nextSeen := append(cloneStrings(seen), typ)
	for i := 0; i < len(fieldTypes); i++ {
		entry := fieldTypes[i]
		if entry.owner != typ {
			continue
		}
		fieldType := unsafeSizeofStructFieldTypeTextForLower(entry, arrayFieldTypes)
		fieldSize, fieldAlign, ok := unsafeSizeofTypeLayoutForLower(fieldType, namedTypes, fieldTypes, arrayFieldTypes, pos, wordSize, nextSeen)
		if !ok || fieldAlign <= 0 {
			return 0, 0, false
		}
		offset = unsafeSizeofAlignOffsetForLower(offset, fieldAlign)
		offset += fieldSize
		if fieldAlign > maxAlign {
			maxAlign = fieldAlign
		}
	}
	return unsafeSizeofAlignOffsetForLower(offset, maxAlign), maxAlign, true
}

func unsafeSizeofStructFieldTypeTextForLower(entry structFieldTypeEntry, arrayFieldTypes arrayStructFieldLowerInfoTable) string {
	if info, ok := arrayStructFieldLowerInfoTableLookup(arrayFieldTypes, entry.owner, entry.field); ok {
		return "[" + strconv.FormatInt(info.length, 10) + "]" + info.elem
	}
	return localTypeInfoText(entry.info)
}

func unsafeSizeofMinAlignForLower(size int, wordSize int) int {
	if wordSize > 0 && wordSize < size {
		return wordSize
	}
	return size
}

func unsafeSizeofAlignOffsetForLower(offset int, align int) int {
	if align <= 1 {
		return offset
	}
	rem := offset % align
	if rem == 0 {
		return offset
	}
	return offset + align - rem
}

func unsafeSizeofResolveNamedTypeForLower(typ string, namedTypes sizeofNamedTypeTable, pos int) string {
	if typ == "" || strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return typ
	}
	raw := sizeofNamedTypeLookup(namedTypes, typ, pos)
	if raw == "" || raw == typ {
		return typ
	}
	return raw
}

func staticCallbackTemplatesForLoadFiles(files []load.File, pkg load.Package, depPackages []load.Package, topNames symbolNameTable, topFunctionNames symbolNameTable, methods methodTable, methodOrder []string, packageAnonymousTypes []localTypeDeclInfo, namedSlices []namedSliceInfo, namedArrays []namedArrayInfo, namedMaps []namedMapInfo, sizeofValueTypes sizeofLocalTypeTable, sizeofNamedTypes sizeofNamedTypeTable, fieldTypes structFieldTypeTable, structOwners structOwnerTable, functionResults localTypeTable, arrayFunctionResults []arrayFunctionResultLowerInfo, arrayFieldTypes arrayStructFieldLowerInfoTable, panicRecover panicRecoverNames, wordSize int, functionTypeNames []string) staticCallbackSetForLower {
	var set staticCallbackSetForLower
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsedValue, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		parsed := &parsedValue
		importRefs, _ := importReferenceMap(parsed, depPackages)
		fileFieldTypes := appendStructFieldTypeTables(cloneStructFieldTypeTable(fieldTypes), importedStructFieldTypesForFile(parsed, depPackages))
		fileStructOwners := cloneStructOwnerTable(structOwners)
		fileArrayFieldTypes := appendArrayStructFieldLowerInfoTables(cloneArrayStructFieldLowerInfoTable(arrayFieldTypes), importedArrayStructFieldLowerInfosForFile(parsed, depPackages))
		importedValueTypes := importedTopLevelValueTypesForFile(parsed, depPackages)
		importMethods, importMethodOrder := importMethodMap(parsed, depPackages, importedTypeLocalNames(parsed))
		allMethods := mergedMethodMap(methods, methodOrder, importMethods, importMethodOrder)
		decls := parsed.Decls
		for declIndex := 0; declIndex < len(decls); declIndex++ {
			decl := decls[declIndex]
			params := staticCallbackParamsForDecl(parsed, &decl, functionTypeNames)
			if len(params) == 0 {
				continue
			}
			refs := make([]unit.Symbol, 0, declReferenceCapacity(parsed, &decl))
			localTypeDecls := localTypeDeclsForDecl(parsed, &decl, pkg.ImportPath, topNames)
			rewriteTypeDecls := appendLocalTypeDeclInfos(packageAnonymousTypes, localTypeDecls)
			rewriteNamedArrays := namedArrays
			rewriteNamedMaps := namedMaps
			rewriteStructOwners := fileStructOwners
			if len(localTypeDecls) > 0 {
				rewriteNamedArrays = appendGeneratedLocalNamedArrayTypes(rewriteNamedArrays, parsed, localTypeDecls)
				rewriteNamedMaps = appendGeneratedLocalNamedMapTypes(rewriteNamedMaps, parsed, localTypeDecls)
				fileStructOwners = appendGeneratedLocalStructOwners(fileStructOwners, parsed, localTypeDecls)
				rewriteStructOwners = appendOriginalLocalStructOwners(cloneStructOwnerTable(fileStructOwners), parsed, localTypeDecls)
			}
			declUnitName := unitDeclSymbol(&decl, parsed, topNames)
			var generatedDecls []unit.Decl
			body := rewriteDecl(parsed, &decl, topNames, topFunctionNames, importRefs, allMethods, namedSlices, rewriteNamedArrays, rewriteNamedMaps, sizeofValueTypes, sizeofNamedTypes, rewriteTypeDecls, fileFieldTypes, rewriteStructOwners, nil, importedValueTypes, functionResults, arrayFunctionResults, nil, fileArrayFieldTypes, nil, nil, "", declUnitName, wordSize, &refs, &generatedDecls, nil)
			localTypes := localTypesForDecl(parsed, &decl)
			body = lowerDeferStatements(body, declUnitName, panicRecover, false, normalizationContext{
				localTypes:      localTypes,
				functionResults: functionResults,
				fieldTypes:      fileFieldTypes,
				generatedDecls:  &generatedDecls,
			})
			set.templates = append(set.templates, staticCallbackTemplateForLower{
				name:           decl.Name,
				unitName:       declUnitName,
				params:         params,
				body:           body,
				generatedDecls: generatedDecls,
				refs:           refs,
			})
		}
		arena.Reset(mark)
	}
	return set
}

func appendDependencyStaticCallbackTemplates(set staticCallbackSetForLower, packages []load.Package, currentImportPath string, targetName string) staticCallbackSetForLower {
	for pkgIndex := 0; pkgIndex < len(packages); pkgIndex++ {
		pkg := packages[pkgIndex]
		if pkg.ImportPath == currentImportPath {
			continue
		}
		templates := staticCallbackTemplatesForPackage(pkg, packages, targetName)
		set.templates = append(set.templates, templates.templates...)
	}
	return set
}

func staticCallbackTemplatesForPackage(pkg load.Package, packages []load.Package, targetName string) staticCallbackSetForLower {
	files := snapshotLoadFiles(pkg.Files)
	sortFilesByPath(files)
	symbolCap := packageSymbolCapacity(files, len(pkg.Imports))
	topNames := packageSymbolNamesForLoadFiles(pkg.ImportPath, files, symbolCap)
	topFunctionNames := packageFunctionSymbolNamesForLoadFiles(pkg.ImportPath, files)
	methods, methodOrder := packageMethodTableForLoadFiles(pkg.ImportPath, files, topNames)
	packageAnonymousTypes := packageAnonymousStructTypeDeclsForLoadFiles(files, pkg.ImportPath)
	namedTypes := namedTypeUnderlyingsForLoadFiles(files)
	namedTypes = appendGeneratedTypeUnderlyingsFromBodies(namedTypes, packageAnonymousTypes)
	namedSlices := namedSliceTypesForLoadFiles(files, topNames)
	namedSlices = rewriteNamedSliceAnonymousStructElems(namedSlices, packageAnonymousTypes)
	namedArrays := namedArrayTypesForLoadFiles(files, topNames)
	namedMaps := namedMapTypesForLoadFiles(files, topNames)
	sizeofValueTypes := sizeofLocalTypesForLoadFiles(files)
	sizeofNamedTypes := sizeofNamedTypesForLoadFiles(files)
	fieldTypes := packageStructFieldTypesForLoadFiles(files, pkg.ImportPath, namedTypes)
	structOwners := packageStructOwnersForLoadFiles(files)
	functionResults := functionResultTypesForLoadFiles(files, topNames)
	arrayFunctionResults := arrayFunctionResultTypesForLoadFiles(files, topNames, namedArrays)
	arrayFieldTypes := packageArrayStructFieldLowerInfosForLoadFiles(files, pkg.ImportPath, namedArrays)
	panicRecover := panicRecoverNames{}
	if packageUsesPanicRecover(files) {
		panicRecover = panicRecoverNamesForPackage(pkg.ImportPath)
	}
	wordSize := target.WordSize(targetName)
	functionTypeNames := functionTypeNamesForLoadFiles(files)
	set := staticCallbackTemplatesForLoadFiles(files, pkg, packages, topNames, topFunctionNames, methods, methodOrder, packageAnonymousTypes, namedSlices, namedArrays, namedMaps, sizeofValueTypes, sizeofNamedTypes, fieldTypes, structOwners, functionResults, arrayFunctionResults, arrayFieldTypes, panicRecover, wordSize, functionTypeNames)
	return attachStaticCallbackTemplatePackageSymbols(set, pkg.ImportPath, topNames)
}

func attachStaticCallbackTemplatePackageSymbols(set staticCallbackSetForLower, importPath string, topNames symbolNameTable) staticCallbackSetForLower {
	for templateIndex := 0; templateIndex < len(set.templates); templateIndex++ {
		set.templates[templateIndex].importPath = importPath
		set.templates[templateIndex].packageSymbols = topNames
	}
	return set
}

func packageFunctionSymbolNamesForLoadFiles(importPath string, files []load.File) symbolNameTable {
	var names symbolNameTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if decl.Kind != "func" || decl.Receiver || decl.Name == "" {
				continue
			}
			names = symbolNameTableSet(names, decl.Name, SymbolName(importPath, decl.Name))
		}
		arena.Reset(mark)
	}
	return names
}

func packageMethodTableForLoadFiles(importPath string, files []load.File, topNames symbolNameTable) (methodTable, []string) {
	var methods methodTable
	var order []string
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if decl.Kind != "func" || !decl.Receiver {
				continue
			}
			info := methodDeclInfo(&parsed, &decl)
			if info.name == "" {
				continue
			}
			info.unitName = symbolNameTableUnitName(topNames, info.name)
			info.importPath = importPath
			if methodTableLookup(methods, info.name).unitName == "" {
				order = append(order, info.name)
			}
			methods = append(methods, methodEntry{lookup: info.name, info: info})
		}
		arena.Reset(mark)
	}
	return methods, order
}

func staticCallbackTemplateForDecl(set staticCallbackSetForLower, file *parse.File, decl parse.Decl) int {
	if decl.Kind != "func" || decl.Receiver {
		return -1
	}
	name := unitDeclName(file, &decl)
	for i := 0; i < len(set.templates); i++ {
		if set.templates[i].name == name {
			return i
		}
	}
	return -1
}

func staticCallbackParamsForDecl(file *parse.File, decl *parse.Decl, functionTypeNames []string) []staticCallbackParamForLower {
	if decl.Kind != "func" || decl.Receiver {
		return nil
	}
	toks := file.Tokens
	name := tokenIndexAt(toks, int(decl.NameTok.Start))
	if name < 0 || name+1 >= len(toks) || toks[name+1].Text != "(" {
		return nil
	}
	paramsOpen := name + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || int(toks[paramsClose].End) > decl.End {
		return nil
	}
	return staticCallbackParamsInRange(toks, paramsOpen+1, paramsClose, functionTypeNames)
}

func packageStaticCallbackDeclNames(files []load.File, functionTypeNames []string) []string {
	var names []string
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		mark := arena.Mark()
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			arena.Reset(mark)
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			decl := parsed.Decls[declIndex]
			if len(staticCallbackParamsForDecl(&parsed, &decl, functionTypeNames)) == 0 {
				continue
			}
			names = appendStringUnique(names, decl.Name)
		}
		arena.Reset(mark)
	}
	return names
}

func staticCallbackParamsInRange(toks []scan.Token, start int, end int, functionTypeNames []string) []staticCallbackParamForLower {
	var out []staticCallbackParamForLower
	var pending []string
	paramIndex := 0
	segments := topLevelExpressionRanges(toks, start, end)
	for i := 0; i < len(segments); i++ {
		segment := segments[i]
		name, typeStart, typeEnd, hasType := staticCallbackParameterSegmentForLower(toks, segment.start, segment.end)
		if hasType {
			names := pending
			if name != "" {
				names = append(names, name)
			}
			if len(names) == 0 {
				names = append(names, "")
			}
			if staticCallbackTypeIsFunction(toks, typeStart, typeEnd, functionTypeNames) {
				for nameIndex := 0; nameIndex < len(names); nameIndex++ {
					out = append(out, staticCallbackParamForLower{index: paramIndex + nameIndex, name: names[nameIndex]})
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

func staticCallbackParameterSegmentForLower(toks []scan.Token, start int, end int) (string, int, int, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", 0, 0, false
	}
	if toks[start].Kind == scan.Ident {
		if start+1 < end && staticCallbackParameterTypeStart(toks[start+1]) {
			return toks[start].Text, start + 1, end, true
		}
		if start+1 == end {
			return toks[start].Text, 0, 0, false
		}
	}
	if staticCallbackParameterTypeStart(toks[start]) {
		return "", start, end, true
	}
	return "", start, end, true
}

func staticCallbackParameterTypeStart(tok scan.Token) bool {
	return isTypeStart(tok) || tok.Text == "func"
}

func staticCallbackTypeIsFunction(toks []scan.Token, start int, end int, functionTypeNames []string) bool {
	start, end = trimTokenRange(toks, start, end)
	if start < end && toks[start].Text == "func" {
		return true
	}
	return start+1 == end && toks[start].Kind == scan.Ident && containsString(functionTypeNames, toks[start].Text)
}

func staticCallbackCallReplacement(source []byte, toks []scan.Token, pos int, limit int, topNames symbolNameTable, functionNames symbolNameTable, importRefs importSymbolTable, importLocalNames localNameTable, aliases []functionAliasInfo, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, functionLiteralIndex *int, callbacks *staticCallbackSetForLower, refs *[]unit.Symbol, generatedDecls *[]unit.Decl) (string, int, int, bool) {
	if callbacks == nil || pos < 0 || pos >= limit || toks[pos].Kind != scan.Ident {
		return "", 0, 0, false
	}
	templateIndex, open, ok := staticCallbackTemplateIndexForCall(toks, pos, limit, *callbacks, importRefs, importLocalNames)
	if !ok {
		return "", 0, 0, false
	}
	callClose := findClose(toks, open, "(", ")")
	if callClose < 0 || callClose >= limit {
		return "", 0, 0, false
	}
	args := topLevelExpressionRanges(toks, open+1, callClose)
	template := callbacks.templates[templateIndex]
	var targets []staticCallbackTargetForLower
	localFunctionLiteralIndex := 0
	if functionLiteralIndex != nil {
		localFunctionLiteralIndex = *functionLiteralIndex
	}
	for paramIndex := 0; paramIndex < len(template.params); paramIndex++ {
		param := template.params[paramIndex]
		if param.index < 0 || param.index >= len(args) {
			return "", 0, 0, false
		}
		arg := args[param.index]
		target, ok := staticCallbackTargetForLowerArg(source, toks, arg.start, arg.end, topNames, functionNames, importRefs, aliases, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, &localFunctionLiteralIndex, refs)
		if !ok {
			return "", 0, 0, false
		}
		target.param = param
		target = staticCallbackTargetWithWrapperCaptures(target, len(targets))
		targets = append(targets, target)
	}
	if functionLiteralIndex != nil {
		*functionLiteralIndex = localFunctionLiteralIndex
	}
	if generatedDecls != nil {
		for targetIndex := 0; targetIndex < len(targets); targetIndex++ {
			if targets[targetIndex].hasDecl {
				*generatedDecls = append(*generatedDecls, targets[targetIndex].decl)
			}
		}
	}
	specializedUnitName := staticCallbackSpecializationUnitName(callbacks, templateIndex, targets, refs, generatedDecls)
	var callArgs []string
	for targetIndex := 0; targetIndex < len(targets); targetIndex++ {
		target := targets[targetIndex]
		for captureIndex := 0; captureIndex < len(target.captures); captureIndex++ {
			callArgs = append(callArgs, string(appendFunctionLiteralCaptureArgument(nil, target.captures[captureIndex])))
		}
	}
	sourceText := string(source)
	for argIndex := 0; argIndex < len(args); argIndex++ {
		if staticCallbackParamIndex(template.params, argIndex) >= 0 {
			continue
		}
		callArgs = append(callArgs, staticCallbackArgumentText(sourceText, toks, args[argIndex].start, args[argIndex].end, topNames, importRefs, refs))
	}
	replacement := specializedUnitName + "(" + strings.Join(callArgs, ", ") + ")"
	return replacement, int(toks[callClose].End), callClose, true
}

func staticCallbackTemplateIndexForCall(toks []scan.Token, pos int, limit int, callbacks staticCallbackSetForLower, importRefs importSymbolTable, importLocalNames localNameTable) (int, int, bool) {
	if pos+1 < limit && toks[pos+1].Text == "(" {
		templateIndex := staticCallbackTemplateIndexByName(callbacks, toks[pos].Text)
		if templateIndex >= 0 {
			return templateIndex, pos + 1, true
		}
	}
	if pos+3 < limit && toks[pos+1].Text == "." && toks[pos+2].Kind == scan.Ident && toks[pos+3].Text == "(" {
		if isLocalNameAt(importLocalNames, toks[pos].Text, int(toks[pos].Start)) {
			return -1, 0, false
		}
		group, ok := importSymbolTableGroup(importRefs, toks[pos].Text)
		if !ok {
			return -1, 0, false
		}
		sym, ok := importSymbolByName(group, toks[pos+2].Text)
		if !ok || sym.UnitName == "" {
			return -1, 0, false
		}
		templateIndex := staticCallbackTemplateIndexByUnitName(callbacks, sym.UnitName)
		if templateIndex >= 0 {
			return templateIndex, pos + 3, true
		}
	}
	return -1, 0, false
}

func staticCallbackTargetWithWrapperCaptures(target staticCallbackTargetForLower, targetIndex int) staticCallbackTargetForLower {
	if len(target.captures) == 0 {
		return target
	}
	captures := make([]functionLiteralCapture, len(target.captures))
	copy(captures, target.captures)
	for captureIndex := 0; captureIndex < len(captures); captureIndex++ {
		captures[captureIndex].param = staticCallbackWrapperCaptureName(targetIndex, captureIndex)
	}
	target.captures = captures
	return target
}

func staticCallbackWrapperCaptureName(targetIndex int, captureIndex int) string {
	return "rtg_callback_capture_" + strconv.Itoa(targetIndex) + "_" + strconv.Itoa(captureIndex)
}

func staticCallbackArgumentText(source string, toks []scan.Token, start int, end int, topNames symbolNameTable, importRefs importSymbolTable, refs *[]unit.Symbol) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	var out []byte
	cursor := int(toks[start].Start)
	for i := start; i < end; i++ {
		tok := toks[i]
		if int(tok.Start) < cursor {
			continue
		}
		if tok.Kind == scan.Ident {
			if i+1 < end && toks[i+1].Text == "{" {
				unitName := symbolNameTableUnitName(topNames, tok.Text)
				if unitName != "" {
					out = appendStringRange(out, source, cursor, int(tok.Start))
					out = appendString(out, unitName)
					cursor = int(tok.End)
					continue
				}
			}
			if i+3 < end && toks[i+1].Text == "." && toks[i+2].Kind == scan.Ident && toks[i+3].Text == "{" {
				group, groupOK := importSymbolTableGroup(importRefs, tok.Text)
				if groupOK {
					sym, symOK := importSymbolByName(group, toks[i+2].Text)
					if symOK && sym.UnitName != "" {
						if sym.ImportPath != "" {
							appendUnitSymbolRef(refs, sym)
						}
						out = appendStringRange(out, source, cursor, int(tok.Start))
						out = appendString(out, sym.UnitName)
						cursor = int(toks[i+2].End)
						i += 2
						continue
					}
				}
			}
		}
	}
	out = appendStringRange(out, source, cursor, int(toks[end-1].End))
	return strings.TrimSpace(string(out))
}

func staticCallbackTargetForLowerArg(source []byte, toks []scan.Token, start int, end int, topNames symbolNameTable, functionNames symbolNameTable, importRefs importSymbolTable, aliases []functionAliasInfo, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, functionLiteralIndex *int, refs *[]unit.Symbol) (staticCallbackTargetForLower, bool) {
	symbol, ok := functionAliasTargetSymbol(toks, start, end, functionNames, importRefs)
	if ok {
		return staticCallbackTargetForLower{unitName: symbol.UnitName, symbol: symbol}, true
	}
	if info, methodOK := methodExpressionAliasTargetInfo(toks, start, end, methods, localTypes); methodOK {
		return staticCallbackTargetForLower{unitName: info.unitName, symbol: info.symbol}, true
	}
	if info, methodValueOK := methodValueAliasTargetInfo(toks, start, end, topNames, methods, localTypes, fieldTypes, unitName, 0); methodValueOK {
		return staticCallbackTargetFromMethodValueAlias(info), true
	}
	if info, compositeMethodValueOK := compositeLiteralMethodValueAliasTargetInfo(source, toks, start, end, topNames, importRefs, methods, fieldTypes, unitName, 0, refs); compositeMethodValueOK {
		return staticCallbackTargetFromMethodValueAlias(info), true
	}
	if functionLiteralIndex != nil {
		info, literalOK := functionLiteralAliasTargetInfo(source, toks, start, end, topNames, localTypes, localTypeDecls, namedArrays, unitName, *functionLiteralIndex)
		if literalOK {
			*functionLiteralIndex = *functionLiteralIndex + 1
			return staticCallbackTargetForLower{unitName: info.unitName, symbol: info.symbol, captures: info.captures, decl: info.decl, hasDecl: info.hasDecl}, true
		}
	}
	start, end = trimTokenRange(toks, start, end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		alias, aliasOK := functionAliasAt(aliases, toks[start].Text, int(toks[start].Start))
		if aliasOK && alias.unitName != "" {
			if alias.receiver != "" {
				capture := functionLiteralCapture{name: alias.receiver, typ: alias.receiverType}
				return staticCallbackTargetForLower{unitName: alias.unitName, symbol: alias.symbol, captures: []functionLiteralCapture{capture}}, true
			}
			return staticCallbackTargetForLower{unitName: alias.unitName, symbol: alias.symbol, captures: alias.captures}, true
		}
	}
	return staticCallbackTargetForLower{}, false
}

func staticCallbackTargetFromMethodValueAlias(info functionAliasInfo) staticCallbackTargetForLower {
	capture := functionLiteralCapture{name: info.capture, typ: info.receiverType}
	return staticCallbackTargetForLower{unitName: info.unitName, symbol: info.symbol, captures: []functionLiteralCapture{capture}}
}

func staticCallbackTemplateIndexByName(callbacks staticCallbackSetForLower, name string) int {
	for i := 0; i < len(callbacks.templates); i++ {
		if callbacks.templates[i].name == name {
			return i
		}
	}
	return -1
}

func staticCallbackTemplateIndexByUnitName(callbacks staticCallbackSetForLower, unitName string) int {
	for i := 0; i < len(callbacks.templates); i++ {
		if callbacks.templates[i].unitName == unitName {
			return i
		}
	}
	return -1
}

func staticCallbackParamIndex(params []staticCallbackParamForLower, index int) int {
	for i := 0; i < len(params); i++ {
		if params[i].index == index {
			return i
		}
	}
	return -1
}

func staticCallbackSpecializationUnitName(callbacks *staticCallbackSetForLower, templateIndex int, targets []staticCallbackTargetForLower, refs *[]unit.Symbol, generatedDecls *[]unit.Decl) string {
	template := callbacks.templates[templateIndex]
	key := staticCallbackSpecializationKey(template, targets)
	for i := 0; i < len(callbacks.specs); i++ {
		if callbacks.specs[i].key == key {
			return callbacks.specs[i].unitName
		}
	}
	unitName := template.unitName + "_callback_" + strconv.Itoa(len(callbacks.specs))
	callbacks.specs = append(callbacks.specs, staticCallbackSpecForLower{key: key, unitName: unitName})
	for refIndex := 0; refIndex < len(template.refs); refIndex++ {
		appendUnitSymbolRef(refs, template.refs[refIndex])
	}
	for targetIndex := 0; targetIndex < len(targets); targetIndex++ {
		if targets[targetIndex].symbol.ImportPath != "" {
			appendUnitSymbolRef(refs, targets[targetIndex].symbol)
		}
	}
	if !callbacks.templates[templateIndex].helpersEmitted {
		for declIndex := 0; declIndex < len(template.generatedDecls); declIndex++ {
			*generatedDecls = append(*generatedDecls, template.generatedDecls[declIndex])
		}
		callbacks.templates[templateIndex].helpersEmitted = true
	}
	specializedBody := staticCallbackSpecializedBody(template.body, unitName, template.params, targets)
	appendStaticCallbackTemplatePackageRefs(refs, specializedBody, template.importPath, template.packageSymbols, template.unitName)
	*generatedDecls = append(*generatedDecls, unit.Decl{
		Path:     "rtg-static-callback",
		Kind:     "func",
		Name:     unitName,
		UnitName: unitName,
		Body:     specializedBody,
	})
	return unitName
}

func appendStaticCallbackTemplatePackageRefs(refs *[]unit.Symbol, body string, importPath string, topNames symbolNameTable, templateUnitName string) {
	if importPath == "" || len(topNames) == 0 {
		return
	}
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return
	}
	for tokenIndex := 0; tokenIndex < len(toks); tokenIndex++ {
		if toks[tokenIndex].Kind != scan.Ident {
			continue
		}
		for nameIndex := 0; nameIndex < len(topNames); nameIndex++ {
			sym := topNames[nameIndex]
			if sym.unitName == "" || sym.unitName == templateUnitName || toks[tokenIndex].Text != sym.unitName {
				continue
			}
			appendUnitSymbolRef(refs, unit.Symbol{
				ImportPath: importPath,
				Name:       sym.name,
				UnitName:   sym.unitName,
			})
		}
	}
}

func staticCallbackSpecializationKey(template staticCallbackTemplateForLower, targets []staticCallbackTargetForLower) string {
	key := template.unitName
	for i := 0; i < len(targets); i++ {
		key += "\x00" + targets[i].param.name + "\x00" + targets[i].unitName
	}
	return key
}

func staticCallbackSpecializedBody(body string, unitName string, params []staticCallbackParamForLower, targets []staticCallbackTargetForLower) string {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return body
	}
	if len(toks) < 4 || toks[0].Text != "func" || toks[1].Kind != scan.Ident || toks[2].Text != "(" {
		return body
	}
	paramsOpen := 2
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return body
	}
	var replacements []expressionReplacement
	replacements = append(replacements, expressionReplacement{start: int(toks[1].Start), end: int(toks[1].End), text: unitName})
	replacements = append(replacements, expressionReplacement{start: int(toks[paramsOpen].Start), end: int(toks[paramsClose].End), text: staticCallbackSpecializedParameterList(body, toks, paramsOpen, paramsClose, params, targets)})
	for i := paramsClose + 1; i+1 < len(toks); i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "(" || lowerSelectorMember(toks, i) {
			continue
		}
		target := staticCallbackTargetByParamName(targets, toks[i].Text)
		if target.unitName == "" {
			continue
		}
		replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[i].End), text: target.unitName})
		if len(target.captures) > 0 {
			callClose := findClose(toks, i+1, "(", ")")
			if callClose > i+1 {
				replacements = append(replacements, expressionReplacement{start: int(toks[i+1].End), end: int(toks[i+1].End), text: staticCallbackSpecializedCaptureArguments(target.captures, int(toks[i+1].End) < int(toks[callClose].Start))})
			}
		}
	}
	sortExpressionReplacementsByStart(replacements)
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func staticCallbackSpecializedParameterList(body string, toks []scan.Token, open int, close int, params []staticCallbackParamForLower, targets []staticCallbackTargetForLower) string {
	segments := topLevelExpressionRanges(toks, open+1, close)
	var kept []string
	for targetIndex := 0; targetIndex < len(targets); targetIndex++ {
		target := targets[targetIndex]
		for captureIndex := 0; captureIndex < len(target.captures); captureIndex++ {
			kept = append(kept, staticCallbackCaptureParameterText(target.captures[captureIndex]))
		}
	}
	for i := 0; i < len(segments); i++ {
		if staticCallbackParamIndex(params, i) >= 0 {
			continue
		}
		kept = append(kept, strings.TrimSpace(tokenRangeText(body, toks, segments[i].start, segments[i].end)))
	}
	return "(" + strings.Join(kept, ", ") + ")"
}

func staticCallbackCaptureParameterText(capture functionLiteralCapture) string {
	text := capture.param + " "
	if capture.pointer {
		text += "*"
	}
	return text + capture.typ
}

func staticCallbackSpecializedCaptureArguments(captures []functionLiteralCapture, hasOrdinaryArgs bool) string {
	var args []string
	for captureIndex := 0; captureIndex < len(captures); captureIndex++ {
		args = append(args, captures[captureIndex].param)
	}
	text := strings.Join(args, ", ")
	if hasOrdinaryArgs {
		text += ", "
	}
	return text
}

func staticCallbackTargetByParamName(targets []staticCallbackTargetForLower, name string) staticCallbackTargetForLower {
	for i := 0; i < len(targets); i++ {
		if targets[i].param.name == name {
			return targets[i]
		}
	}
	return staticCallbackTargetForLower{}
}

func functionAliasesForDecl(file *parse.File, decl *parse.Decl, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, refs *[]unit.Symbol) []functionAliasInfo {
	var aliases []functionAliasInfo
	if decl.Kind != "func" {
		return aliases
	}
	toks := file.Tokens
	start := tokenIndexAt(toks, decl.Start)
	if start < 0 {
		return aliases
	}
	body := findTokenText(toks, start, decl.End, "{")
	if body < 0 {
		return aliases
	}
	close := findClose(toks, body, "{", "}")
	if close < 0 {
		close = len(toks) - 1
	}
	for i := body + 1; i < close; i++ {
		if toks[i].Text == ":=" {
			aliases = collectShortDeclFunctionAliasInfos(file.Source, toks, body, i, close, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, refs, aliases)
			continue
		}
		if toks[i].Text == "var" {
			aliases = collectVarFunctionAliasInfos(file.Source, toks, body, i, close, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, refs, aliases)
		}
	}
	return aliases
}

func collectShortDeclFunctionAliasInfos(source []byte, toks []scan.Token, body int, assign int, limit int, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, refs *[]unit.Symbol, aliases []functionAliasInfo) []functionAliasInfo {
	stmtStart := simpleStatementStartForLower(toks, body, assign)
	if stmtStart < assign && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
		return aliases
	}
	stmtEnd := lowerSimpleStatementEnd(toks, assign+1, limit)
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != len(rhs) {
		return aliases
	}
	scopeEnd := localScopeEnd(toks, body, assign, int(toks[limit].End))
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end)
		if name == "" {
			continue
		}
		info, ok := staticAliasTargetInfo(source, toks, rhs[i].start, rhs[i].end, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, len(aliases), refs)
		if !ok {
			continue
		}
		info.name = name
		info.start = int(toks[assign].End)
		info.end = scopeEnd
		info.declStart = stmtStart
		info.declEnd = stmtEnd
		aliases = append(aliases, info)
	}
	return aliases
}

func collectVarFunctionAliasInfos(source []byte, toks []scan.Token, body int, pos int, limit int, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, refs *[]unit.Symbol, aliases []functionAliasInfo) []functionAliasInfo {
	if pos+1 >= limit || toks[pos+1].Text == "(" || toks[pos+1].Kind != scan.Ident {
		return aliases
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos+1, limit)
	eq := findTopLevelToken(toks, pos+1, stmtEnd, "=")
	if eq < 0 || eq != pos+2 {
		return aliases
	}
	info, ok := staticAliasTargetInfo(source, toks, eq+1, stmtEnd, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, len(aliases), refs)
	if !ok {
		return aliases
	}
	scopeEnd := localScopeEnd(toks, body, pos, int(toks[limit].End))
	info.name = toks[pos+1].Text
	info.start = int(toks[eq].End)
	info.end = scopeEnd
	info.declStart = pos
	info.declEnd = stmtEnd
	return append(aliases, info)
}

func staticAliasTargetInfo(source []byte, toks []scan.Token, start int, end int, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string, index int, refs *[]unit.Symbol) (functionAliasInfo, bool) {
	if info, ok := functionLiteralAliasTargetInfo(source, toks, start, end, topNames, localTypes, localTypeDecls, namedArrays, unitName, index); ok {
		return info, true
	}
	if symbol, ok := functionAliasTargetSymbol(toks, start, end, functionNames, importRefs); ok {
		return functionAliasInfo{unitName: symbol.UnitName, symbol: symbol}, true
	}
	if info, ok := methodValueAliasTargetInfo(toks, start, end, topNames, methods, localTypes, fieldTypes, unitName, index); ok {
		return info, true
	}
	if info, ok := methodExpressionAliasTargetInfo(toks, start, end, methods, localTypes); ok {
		return info, true
	}
	if info, ok := compositeLiteralMethodValueAliasTargetInfo(source, toks, start, end, topNames, importRefs, methods, fieldTypes, unitName, index, refs); ok {
		return info, true
	}
	return functionAliasInfo{}, false
}

func functionLiteralAliasTargetInfo(source []byte, toks []scan.Token, start int, end int, topNames symbolNameTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, namedArrays []namedArrayInfo, unitName string, index int) (functionAliasInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	literal, ok := functionLiteralForLowerAt(toks, start, end)
	if !ok || literal.start != start || literal.end != end {
		return functionAliasInfo{}, false
	}
	literalUnitName := unitName + "_func_literal_" + strconv.Itoa(index)
	captures := functionLiteralCapturesForLower(toks, literal, localTypes, localTypeDecls)
	params := functionLiteralGeneratedArrayParamTypesForLower(source, toks, literal, captures, namedArrays)
	return functionAliasInfo{unitName: literalUnitName, captures: captures, decl: functionLiteralDecl(source, toks, literal, literalUnitName, captures, topNames, namedArrays), hasDecl: true, arrayParams: params}, true
}

func functionLiteralDecl(source []byte, toks []scan.Token, literal functionLiteralForLower, literalUnitName string, captures []functionLiteralCapture, topNames symbolNameTable, namedArrays []namedArrayInfo) unit.Decl {
	params := functionLiteralParameterText(source, toks, literal, captures)
	tail := functionLiteralTailText(source, toks, literal, captures)
	body := "func " + literalUnitName + params + tail
	body = lowerNamedArrayTypeUses(body, namedArrays)
	body = lowerArrayComparisons(body, nil, nil, nil, nil)
	body = lowerArrayFunctionParameterTypes(body)
	body = lowerArrayFunctionResultTypes(body)
	body = lowerGroupedFunctionParameterTypes(body)
	body = qualifyFunctionLiteralMapMakeDirectCalls(body, topNames)
	body = qualifyFunctionLiteralStaticMapAliasInitializerDirectCalls(body, topNames)
	body = qualifyFunctionLiteralStaticMapAliasAssignmentDirectCalls(body, topNames)
	body = lowerStaticMapAliases(body)
	body = lowerMapLiteralCommaOkAssignments(body)
	body = lowerMapLiteralIndexExpressions(body)
	body = lowerMapLiteralLenCalls(body)
	body = lowerDiscardedMapMakeStatements(body, topNames)
	body = lowerDiscardedMapLiteralDeleteStatements(body, topNames)
	body = lowerLocalVarDeclarations(body)
	body = lowerArrayCompositeLiterals(body)
	body = lowerImplicitCompositeElements(body, nil)
	body = qualifyFunctionLiteralMapMakeDirectCalls(body, topNames)
	body = qualifyFunctionLiteralMapLenDirectCalls(body, topNames)
	body = qualifyFunctionLiteralMapIndexDirectCalls(body, topNames)
	body = qualifyFunctionLiteralMapRangeDirectCalls(body, topNames)
	if functionLiteralHasLowerableMapMakeExpression(body) || functionLiteralHasLowerableMapLiteralLenCall(body) || functionLiteralHasLowerableMapLiteralIndexExpression(body) || functionLiteralHasLowerableMapRangeStatement(body) {
		body = normalizeFunctionExpressions(body, literalUnitName, nil, nil)
	}
	return unit.Decl{
		Path:     "rtg-function-literal",
		Kind:     "func",
		Name:     literalUnitName,
		UnitName: literalUnitName,
		Body:     body,
	}
}

func functionLiteralHasLowerableMapLiteralIndexExpression(body string) bool {
	if !strings.Contains(body, "map") || !strings.Contains(body, "[") {
		return false
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return false
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "map" && toks[i].Text != "(" {
			continue
		}
		_, _, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, i, len(toks))
		if !ok {
			continue
		}
		key, keyOK := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
		if !keyOK {
			continue
		}
		tempIndex := 0
		if _, _, valueOK := lowerableMapLiteralIndexValueForNormalize(body, toks, mapStart, mapEnd, key, "rtg_probe", &tempIndex); valueOK {
			arena.Reset(mark)
			return true
		}
	}
	arena.Reset(mark)
	return false
}

func functionLiteralHasLowerableMapLiteralLenCall(body string) bool {
	if !strings.Contains(body, "map") || !strings.Contains(body, "len") {
		return false
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return false
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
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
		args := topLevelExpressionRanges(toks, i+2, close)
		if len(args) != 1 {
			continue
		}
		if _, ok := mapLiteralLenDirectCallNameReplacements(toks, args[0].start, args[0].end, nil); ok {
			arena.Reset(mark)
			return true
		}
		i = close
	}
	arena.Reset(mark)
	return false
}

func functionLiteralHasLowerableMapMakeExpression(body string) bool {
	if !strings.Contains(body, "make") || !strings.Contains(body, "map") {
		return false
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return false
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "make" {
			continue
		}
		close := -1
		if i+1 < len(toks) && toks[i+1].Text == "(" {
			close = findClose(toks, i+1, "(", ")")
		}
		if close < 0 {
			continue
		}
		if lowerableMapMakeExpressionForLower(toks, i, close+1) {
			arena.Reset(mark)
			return true
		}
		i = close
	}
	arena.Reset(mark)
	return false
}

func functionLiteralHasLowerableMapRangeStatement(body string) bool {
	if !strings.Contains(body, "map") || !strings.Contains(body, "range") {
		return false
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return false
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return false
	}
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "for" {
			continue
		}
		stmt, ok := normalizationRangeStatement(toks, i)
		if !ok {
			continue
		}
		tempIndex := 0
		if _, ok := lowerableMapRangeInfoForLower(body, toks, stmt.exprStart, stmt.exprEnd, "rtg_probe", &tempIndex); ok {
			arena.Reset(mark)
			return true
		}
		i = stmt.end
	}
	arena.Reset(mark)
	return false
}

func qualifyFunctionLiteralStaticMapAliasInitializerDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "map") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		alias, ok := staticMapAliasStatementForLowerAt(body, toks, i, len(toks))
		if !ok {
			continue
		}
		callReplacements, ok := mapLiteralLenDirectCallNameReplacements(toks, alias.exprStart, alias.exprEnd, topNames)
		if ok {
			replacements = appendExpressionReplacements(replacements, callReplacements)
		}
		i = alias.stmtEnd - 1
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func qualifyFunctionLiteralStaticMapAliasAssignmentDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "[") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	aliases := staticMapAliasesForLower(body, toks)
	if len(aliases) == 0 {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Kind != scan.Ident {
			continue
		}
		stmtStart := simpleStatementStartForLower(toks, -1, i)
		if stmtStart != i {
			continue
		}
		stmtEnd := lowerSimpleStatementEnd(toks, i, len(toks))
		assign := findTopLevelToken(toks, i, stmtEnd, "=")
		if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
			continue
		}
		lhs := topLevelExpressionRanges(toks, i, assign)
		rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
		if len(lhs) != 1 || len(rhs) != 1 {
			continue
		}
		lhsStart, lhsEnd := trimTokenRange(toks, lhs[0].start, lhs[0].end)
		if lhsStart+3 > lhsEnd || toks[lhsStart].Kind != scan.Ident || toks[lhsStart+1].Text != "[" {
			continue
		}
		indexClose := findClose(toks, lhsStart+1, "[", "]")
		if indexClose != lhsEnd-1 {
			continue
		}
		if _, ok := staticMapAliasForLowerAt(aliases, toks[lhsStart].Text, lhsStart); !ok {
			continue
		}
		callReplacements, ok := directCallNameReplacement(toks, rhs[0].start, rhs[0].end, topNames)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, callReplacements)
		i = stmtEnd - 1
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func qualifyFunctionLiteralMapMakeDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "make") || !strings.Contains(body, "map") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "make" || i+1 >= len(toks) || toks[i+1].Text != "(" {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 || !lowerableMapMakeExpressionForLower(toks, i, close+1) {
			continue
		}
		args := topLevelExpressionRanges(toks, i+2, close)
		if len(args) == 2 && !simpleIntegerLiteralKeyForLower(toks, args[1].start, args[1].end) {
			callReplacements, ok := directCallNameReplacement(toks, args[1].start, args[1].end, topNames)
			if ok {
				replacements = appendExpressionReplacements(replacements, callReplacements)
			}
		}
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func qualifyFunctionLiteralMapIndexDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "[") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "map" && toks[i].Text != "(" {
			continue
		}
		_, end, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, i, len(toks))
		if !ok {
			continue
		}
		key, keyOK := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
		if !keyOK {
			continue
		}
		tempIndex := 0
		if _, _, valueOK := lowerableMapLiteralIndexValueForNormalize(body, toks, mapStart, mapEnd, key, "rtg_probe", &tempIndex); !valueOK {
			continue
		}
		callReplacements, callsOK := mapLiteralLenDirectCallNameReplacements(toks, mapStart, mapEnd, topNames)
		if callsOK {
			replacements = appendExpressionReplacements(replacements, callReplacements)
			i = end - 1
		}
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func qualifyFunctionLiteralMapRangeDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "range") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		if toks[i].Text != "for" {
			continue
		}
		stmt, ok := normalizationRangeStatement(toks, i)
		if !ok {
			continue
		}
		tempIndex := 0
		if _, ok := lowerableMapRangeInfoForLower(body, toks, stmt.exprStart, stmt.exprEnd, "rtg_probe", &tempIndex); !ok {
			continue
		}
		callReplacements, ok := mapLiteralLenDirectCallNameReplacements(toks, stmt.exprStart, stmt.exprEnd, topNames)
		if ok {
			replacements = appendExpressionReplacements(replacements, callReplacements)
			i = stmt.end
		}
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func qualifyFunctionLiteralMapLenDirectCalls(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "len") || len(topNames) == 0 {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
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
		args := topLevelExpressionRanges(toks, i+2, close)
		if len(args) != 1 {
			continue
		}
		callReplacements, ok := mapLiteralLenDirectCallNameReplacements(toks, args[0].start, args[0].end, topNames)
		if ok {
			replacements = appendExpressionReplacements(replacements, callReplacements)
			i = close
		}
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func mapLiteralLenDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapLiteralLenDirectCallNameReplacements(toks, start+1, close, topNames)
		}
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return mapCompositeDirectCallNameReplacements(toks, open+1, close, topNames)
}

func mapCompositeDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var replacements []expressionReplacement
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return nil, false
		}
		if !discardedPureMapCompositeKeyForLower(toks, value.start, colon) {
			return nil, false
		}
		valueReplacements, ok := mapCompositeValueDirectCallNameReplacements(toks, colon+1, value.end, topNames)
		if !ok {
			return nil, false
		}
		replacements = appendExpressionReplacements(replacements, valueReplacements)
	}
	return replacements, true
}

func mapCompositeValueDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return mapCompositeDirectCallNameReplacements(toks, start+1, close, topNames)
	}
	if toks[start].Text == "map" {
		return mapLiteralLenDirectCallNameReplacements(toks, start, end, topNames)
	}
	if toks[start].Text == "[" {
		return arrayCompositeDirectCallNameReplacements(toks, start, end, topNames)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" {
		return nil, true
	}
	return directCallNameReplacement(toks, start, end, topNames)
}

func arrayCompositeDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return arrayCompositeDirectCallNameReplacements(toks, start+1, close, topNames)
		}
	}
	open := pureArrayCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return compositeDirectCallNameReplacements(toks, open+1, close, topNames)
}

func compositeDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var replacements []expressionReplacement
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardedPureCompositeKeyForLower(toks, value.start, colon) {
				return nil, false
			}
			value.start = colon + 1
		}
		valueReplacements, ok := compositeValueDirectCallNameReplacements(toks, value.start, value.end, topNames)
		if !ok {
			return nil, false
		}
		replacements = appendExpressionReplacements(replacements, valueReplacements)
	}
	return replacements, true
}

func compositeValueDirectCallNameReplacements(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return compositeDirectCallNameReplacements(toks, start+1, close, topNames)
	}
	if toks[start].Text == "[" {
		return arrayCompositeDirectCallNameReplacements(toks, start, end, topNames)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	return directCallNameReplacement(toks, start, end, topNames)
}

func directCallNameReplacement(toks []scan.Token, start int, end int, topNames symbolNameTable) ([]expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directCallNameReplacement(toks, start+1, close, topNames)
		}
	}
	if !directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return nil, false
	}
	unitName := symbolNameTableUnitName(topNames, toks[start].Text)
	if unitName == "" {
		return nil, true
	}
	return []expressionReplacement{{
		start: int(toks[start].Start),
		end:   int(toks[start].End),
		text:  unitName,
	}}, true
}

func functionLiteralParameterText(source []byte, toks []scan.Token, literal functionLiteralForLower, captures []functionLiteralCapture) string {
	if len(captures) == 0 {
		return string(source[int(toks[literal.paramsOpen].Start):int(toks[literal.paramsClose].End)])
	}
	var out []byte
	out = append(out, '(')
	for i := 0; i < len(captures); i++ {
		if i > 0 {
			out = appendString(out, ", ")
		}
		out = appendString(out, captures[i].param)
		out = append(out, ' ')
		if captures[i].pointer {
			out = append(out, '*')
		}
		out = appendString(out, captures[i].typ)
	}
	if int(toks[literal.paramsOpen].End) < int(toks[literal.paramsClose].Start) {
		out = appendString(out, ", ")
		out = appendBytes(out, source[int(toks[literal.paramsOpen].End):int(toks[literal.paramsClose].Start)])
	}
	out = append(out, ')')
	return string(out)
}

func functionLiteralTailText(source []byte, toks []scan.Token, literal functionLiteralForLower, captures []functionLiteralCapture) string {
	if len(captures) == 0 {
		return string(source[int(toks[literal.paramsClose].End):int(toks[literal.bodyClose].End)])
	}
	var out []byte
	out = appendBytes(out, source[int(toks[literal.paramsClose].End):int(toks[literal.bodyOpen].End)])
	cursor := int(toks[literal.bodyOpen].End)
	for i := literal.bodyOpen + 1; i < literal.bodyClose; i++ {
		tok := toks[i]
		capture := functionLiteralCaptureByName(captures, tok.Text)
		if tok.Kind != scan.Ident || capture.name == "" || !functionLiteralIdentifierIsCaptureReferenceForLower(toks, literal, i, capture) {
			continue
		}
		if int(tok.Start) < cursor {
			continue
		}
		out = appendBytes(out, source[cursor:int(tok.Start)])
		out = appendString(out, functionLiteralCaptureReference(toks, i, capture))
		cursor = int(tok.End)
	}
	out = appendBytes(out, source[cursor:int(toks[literal.bodyClose].End)])
	return string(out)
}

func functionLiteralCaptureByName(captures []functionLiteralCapture, name string) functionLiteralCapture {
	for i := 0; i < len(captures); i++ {
		if captures[i].name == name {
			return captures[i]
		}
	}
	return functionLiteralCapture{}
}

func functionLiteralIdentifierIsCaptureReferenceForLower(toks []scan.Token, literal functionLiteralForLower, pos int, capture functionLiteralCapture) bool {
	if lowerKeyword(toks[pos].Text) || isCompositeKey(toks, pos) || lowerSelectorMember(toks, pos) || functionLiteralShortDeclTargetForLowerAt(toks, literal, pos) {
		return false
	}
	return !containsString(functionLiteralSameScopeNamesBeforeForLower(toks, literal, pos), capture.name)
}

func functionLiteralCaptureReference(toks []scan.Token, pos int, capture functionLiteralCapture) string {
	if !capture.pointer {
		return capture.param
	}
	if pos+1 < len(toks) && (toks[pos+1].Text == "." || toks[pos+1].Text == "[") {
		return "(*" + capture.param + ")"
	}
	return "*" + capture.param
}

func appendFunctionLiteralCaptureArgument(out []byte, capture functionLiteralCapture) []byte {
	if capture.pointer {
		out = append(out, '&')
	}
	return appendString(out, capture.name)
}

func functionLiteralCapturesForLower(toks []scan.Token, literal functionLiteralForLower, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo) []functionLiteralCapture {
	if len(localTypes) == 0 {
		return nil
	}
	literalLocals := functionLiteralLocalNamesForLower(toks, literal)
	var captures []functionLiteralCapture
	for i := literal.bodyOpen + 1; i < literal.bodyClose; i++ {
		tok := toks[i]
		if tok.Kind != scan.Ident || lowerKeyword(tok.Text) || lowerSelectorMember(toks, i) || isCompositeKey(toks, i) || functionLiteralShortDeclTargetForLowerAt(toks, literal, i) {
			continue
		}
		if containsString(functionLiteralSameScopeNamesBeforeForLower(toks, literal, i), tok.Text) {
			continue
		}
		if functionLiteralCaptureIndex(captures, tok.Text) >= 0 {
			continue
		}
		info := localTypeTableLookup(localTypes, tok.Text)
		typ := localTypeInfoTextForCapture(info, localTypeDecls, int(tok.Start))
		if typ == "" {
			continue
		}
		captures = append(captures, functionLiteralCapture{
			name:    tok.Text,
			param:   functionLiteralCaptureParamName(tok.Text, literalLocals, captures),
			typ:     typ,
			pointer: functionLiteralCaptureAssignedForLower(toks, literal, tok.Text),
		})
	}
	return captures
}

func functionLiteralCaptureParamName(name string, literalLocals []string, captures []functionLiteralCapture) string {
	base := "rtg_capture_" + name
	candidate := base
	index := 0
	for containsString(literalLocals, candidate) || functionLiteralCaptureParamExists(captures, candidate) {
		index++
		candidate = base + "_" + strconv.Itoa(index)
	}
	return candidate
}

func functionLiteralCaptureParamExists(captures []functionLiteralCapture, name string) bool {
	for i := 0; i < len(captures); i++ {
		if captures[i].param == name {
			return true
		}
	}
	return false
}

func functionLiteralCaptureAssignedForLower(toks []scan.Token, literal functionLiteralForLower, name string) bool {
	for i := literal.bodyOpen + 1; i < literal.bodyClose; i++ {
		if toks[i].Text == "func" {
			if nested, ok := functionLiteralForLowerAt(toks, i, literal.bodyClose); ok {
				i = nested.bodyClose
				continue
			}
		}
		if lowerIncDecAt(toks, i, literal.bodyClose) {
			stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
			lhs := topLevelExpressionRanges(toks, stmtStart, i)
			if functionLiteralCaptureAssignmentTargetForLower(toks, lhs, name) {
				return true
			}
			continue
		}
		if lowerCompoundAssignmentAt(toks, i, literal.bodyClose) {
			stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
			lhs := topLevelExpressionRanges(toks, stmtStart, i)
			if functionLiteralCaptureAssignmentTargetForLower(toks, lhs, name) {
				return true
			}
			continue
		}
		if toks[i].Text != "=" || lowerCompoundAssignmentEquals(toks, i) {
			continue
		}
		stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
		lhs := topLevelExpressionRanges(toks, stmtStart, i)
		if functionLiteralCaptureAssignmentTargetForLower(toks, lhs, name) {
			return true
		}
	}
	return false
}

func functionLiteralSameScopeNamesBeforeForLower(toks []scan.Token, literal functionLiteralForLower, pos int) []string {
	scope := lowerInnermostDeclarationScopeStart(toks, literal.bodyOpen, pos)
	var names []string
	if scope == literal.bodyOpen {
		names = appendFunctionLiteralParameterNamesForLower(names, toks, literal.paramsOpen+1, literal.paramsClose)
	}
	for i := literal.bodyOpen + 1; i < pos; i++ {
		if lowerInnermostDeclarationScopeStart(toks, literal.bodyOpen, i) != scope {
			continue
		}
		if toks[i].Text == ":=" {
			stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
			if stmtStart < i && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
				continue
			}
			if pos < lowerSimpleStatementEnd(toks, i+1, literal.bodyClose) {
				continue
			}
			names = appendLowerAssignmentLeftNames(names, toks, stmtStart, i)
			continue
		}
		if toks[i].Text == "var" || toks[i].Text == "const" {
			names = appendLowerVarStatementNames(names, toks, i, pos)
		}
	}
	return names
}

func appendLowerAssignmentLeftNames(names []string, toks []scan.Token, start int, end int) []string {
	lhs := topLevelExpressionRanges(toks, start, end)
	for i := 0; i < len(lhs); i++ {
		name := singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end)
		if name != "" && name != "_" {
			names = appendStringUnique(names, name)
		}
	}
	return names
}

func appendLowerVarStatementNames(names []string, toks []scan.Token, pos int, limit int) []string {
	end := lowerSimpleStatementEnd(toks, pos+1, limit)
	if pos+1 < limit && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close > pos+1 && close < limit {
			for i := pos + 2; i < close; i++ {
				if toks[i].Kind == scan.Ident && (i == pos+2 || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line) {
					names = appendStringUnique(names, toks[i].Text)
				}
			}
			return names
		}
	}
	if pos+1 < end && toks[pos+1].Kind == scan.Ident {
		names = appendStringUnique(names, toks[pos+1].Text)
		for i := pos + 2; i < end; i++ {
			if toks[i].Text == "=" {
				break
			}
			if toks[i].Kind == scan.Ident && toks[i-1].Text == "," {
				names = appendStringUnique(names, toks[i].Text)
			}
		}
	}
	return names
}

func lowerInnermostDeclarationScopeStart(toks []scan.Token, body int, pos int) int {
	block := lowerInnermostBlockOpen(toks, body, pos)
	if clause := lowerInnermostCaseClauseStart(toks, block, pos); clause >= 0 {
		return clause
	}
	return block
}

func lowerInnermostCaseClauseStart(toks []scan.Token, block int, pos int) int {
	if block < 0 || block >= len(toks) {
		return -1
	}
	owner := blockOwnerKeywordInLower(toks, block)
	if owner != "switch" && owner != "select" {
		return -1
	}
	depth := 0
	clause := -1
	for i := block + 1; i < pos && i < len(toks); i++ {
		text := toks[i].Text
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

func lowerInnermostBlockOpen(toks []scan.Token, body int, pos int) int {
	var opens []int
	for i := body; i < pos && i < len(toks); i++ {
		if toks[i].Text == "{" {
			opens = append(opens, i)
			continue
		}
		if toks[i].Text == "}" && len(opens) > 0 {
			opens = opens[:len(opens)-1]
		}
	}
	if len(opens) == 0 {
		return body
	}
	return opens[len(opens)-1]
}

func functionLiteralShortDeclTargetForLowerAt(toks []scan.Token, literal functionLiteralForLower, pos int) bool {
	for i := pos + 1; i < literal.bodyClose && toks[i].Line == toks[pos].Line; i++ {
		if toks[i].Text == ":=" {
			stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
			lhs := topLevelExpressionRanges(toks, stmtStart, i)
			for lhsIndex := 0; lhsIndex < len(lhs); lhsIndex++ {
				if pos >= lhs[lhsIndex].start && pos < lhs[lhsIndex].end {
					return true
				}
			}
			return false
		}
		if toks[i].Text == "=" || toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}" {
			return false
		}
	}
	return false
}

func functionLiteralCaptureAssignmentTargetForLower(toks []scan.Token, lhs []expressionRange, name string) bool {
	for i := 0; i < len(lhs); i++ {
		for j := lhs[i].start; j < lhs[i].end; j++ {
			if toks[j].Kind != scan.Ident || toks[j].Text != name {
				continue
			}
			if lowerSelectorMember(toks, j) || isCompositeKey(toks, j) {
				continue
			}
			return true
		}
	}
	return false
}

func lowerIncDecAt(toks []scan.Token, pos int, limit int) bool {
	if pos+1 >= limit {
		return false
	}
	if int(toks[pos].End) != int(toks[pos+1].Start) {
		return false
	}
	return (toks[pos].Text == "+" && toks[pos+1].Text == "+") || (toks[pos].Text == "-" && toks[pos+1].Text == "-")
}

func lowerCompoundAssignmentAt(toks []scan.Token, pos int, limit int) bool {
	if pos+1 >= limit || toks[pos+1].Text != "=" {
		return false
	}
	if int(toks[pos].End) != int(toks[pos+1].Start) {
		return false
	}
	switch toks[pos].Text {
	case "+", "-", "*", "/", "%":
		return true
	}
	return false
}

func lowerCompoundAssignmentEquals(toks []scan.Token, pos int) bool {
	if pos <= 0 || toks[pos].Text != "=" {
		return false
	}
	return lowerCompoundAssignmentAt(toks, pos-1, len(toks))
}

func functionLiteralCaptureIndex(captures []functionLiteralCapture, name string) int {
	for i := 0; i < len(captures); i++ {
		if captures[i].name == name {
			return i
		}
	}
	return -1
}

func localTypeInfoTextForCapture(info localTypeInfo, localTypeDecls []localTypeDeclInfo, pos int) string {
	if info.name == "" {
		return ""
	}
	name := info.name
	if info.qualifier == "" {
		if replacement := localTypeReplacementAt(localTypeDecls, info.name, pos); replacement != "" {
			name = replacement
		}
	}
	out := ""
	if info.pointer {
		out = "*"
	}
	if info.qualifier != "" {
		out += info.qualifier + "."
	}
	return out + name
}

func functionLiteralLocalNamesForLower(toks []scan.Token, literal functionLiteralForLower) []string {
	var names []string
	names = appendFunctionLiteralParameterNamesForLower(names, toks, literal.paramsOpen+1, literal.paramsClose)
	for i := literal.bodyOpen + 1; i < literal.bodyClose; i++ {
		if toks[i].Text == ":=" {
			stmtStart := simpleStatementStartForLower(toks, literal.bodyOpen, i)
			lhs := topLevelExpressionRanges(toks, stmtStart, i)
			for lhsIndex := 0; lhsIndex < len(lhs); lhsIndex++ {
				name := singleIdentifierExpressionInLower(toks, lhs[lhsIndex].start, lhs[lhsIndex].end)
				if name != "" && name != "_" {
					names = appendStringUnique(names, name)
				}
			}
			continue
		}
		if toks[i].Text == "var" {
			stmtEnd := lowerSimpleStatementEnd(toks, i+1, literal.bodyClose)
			names = appendFunctionLiteralVarNamesForLower(names, toks, i, stmtEnd)
			i = stmtEnd - 1
		}
	}
	return names
}

func appendFunctionLiteralParameterNamesForLower(names []string, toks []scan.Token, start int, end int) []string {
	segmentStart := start
	for i := start; i <= end; i++ {
		if i == end || toks[i].Text == "," {
			names = appendFunctionLiteralParameterSegmentNamesForLower(names, toks, segmentStart, i)
			segmentStart = i + 1
		}
	}
	return names
}

func appendFunctionLiteralParameterSegmentNamesForLower(names []string, toks []scan.Token, start int, end int) []string {
	for start < end && toks[start].Text == "," {
		start++
	}
	for end > start && toks[end-1].Text == "," {
		end--
	}
	if start >= end || toks[start].Kind != scan.Ident {
		return names
	}
	if start+1 < end && isTypeStart(toks[start+1]) {
		return appendStringUnique(names, toks[start].Text)
	}
	if start+1 < end && toks[start+1].Text == "," {
		for i := start; i < end; i++ {
			if toks[i].Kind == scan.Ident {
				names = appendStringUnique(names, toks[i].Text)
			}
			if i+1 < end && toks[i+1].Text != "," {
				break
			}
		}
	}
	return names
}

func appendFunctionLiteralVarNamesForLower(names []string, toks []scan.Token, pos int, end int) []string {
	if pos+1 >= end {
		return names
	}
	if toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close > end {
			return names
		}
		specStart := pos + 2
		for i := specStart; i <= close; i++ {
			if i == close || toks[i].Text == ";" || toks[i].Line != toks[specStart].Line {
				names = appendFunctionLiteralVarSpecNamesForLower(names, toks, specStart, i)
				specStart = i
				if i < close && toks[i].Text == ";" {
					specStart = i + 1
				}
			}
		}
		return names
	}
	return appendFunctionLiteralVarSpecNamesForLower(names, toks, pos+1, end)
}

func appendFunctionLiteralVarSpecNamesForLower(names []string, toks []scan.Token, start int, end int) []string {
	for start < end && toks[start].Text == ";" {
		start++
	}
	lhsEnd := end
	if eq := findTopLevelToken(toks, start, end, "="); eq >= 0 {
		lhsEnd = eq
	}
	for i := start; i < lhsEnd; i++ {
		if toks[i].Text == "," {
			continue
		}
		if toks[i].Kind == scan.Ident && (i == start || toks[i-1].Text == ",") {
			names = appendStringUnique(names, toks[i].Text)
		}
		if i > start && isTypeStart(toks[i]) {
			break
		}
	}
	return names
}

func functionLiteralDirectCallIsDeferredForLower(toks []scan.Token, pos int) bool {
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	return stmtStart >= 0 && stmtStart+1 == pos && toks[stmtStart].Text == "defer"
}

func pointerFunctionLiteralCaptures(captures []functionLiteralCapture) []functionLiteralCapture {
	if len(captures) == 0 {
		return captures
	}
	out := make([]functionLiteralCapture, len(captures))
	for i := 0; i < len(captures); i++ {
		out[i] = captures[i]
		out[i].pointer = true
	}
	return out
}

func lowerSelectorMember(toks []scan.Token, pos int) bool {
	return pos > 0 && toks[pos-1].Text == "."
}

func lowerKeyword(text string) bool {
	switch text {
	case "break", "case", "chan", "const", "continue", "default", "defer", "else", "fallthrough", "for", "func", "go", "goto", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var":
		return true
	}
	return false
}

func functionLiteralDirectCallForLowerAt(toks []scan.Token, start int, end int) (functionLiteralForLower, bool) {
	literal, ok := functionLiteralForLowerAt(toks, start, end)
	if !ok {
		return functionLiteralForLower{}, false
	}
	callOpen := literal.end
	if callOpen >= end || callOpen >= len(toks) || toks[callOpen].Text != "(" {
		return functionLiteralForLower{}, false
	}
	callClose := findClose(toks, callOpen, "(", ")")
	if callClose < 0 || callClose >= end {
		return functionLiteralForLower{}, false
	}
	literal.callOpen = callOpen
	literal.callClose = callClose
	return literal, true
}

func functionLiteralForLowerAt(toks []scan.Token, start int, end int) (functionLiteralForLower, bool) {
	if start < 0 || start+2 >= len(toks) || start >= end || toks[start].Text != "func" || toks[start+1].Text != "(" {
		return functionLiteralForLower{}, false
	}
	paramsOpen := start + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 || paramsClose >= end {
		return functionLiteralForLower{}, false
	}
	bodyOpen := -1
	for i := paramsClose + 1; i < end; i++ {
		if toks[i].Text == "{" {
			bodyOpen = i
			break
		}
		if toks[i].Text == ";" {
			return functionLiteralForLower{}, false
		}
	}
	if bodyOpen < 0 {
		return functionLiteralForLower{}, false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 || bodyClose >= end {
		return functionLiteralForLower{}, false
	}
	return functionLiteralForLower{
		start:       start,
		paramsOpen:  paramsOpen,
		paramsClose: paramsClose,
		bodyOpen:    bodyOpen,
		bodyClose:   bodyClose,
		end:         bodyClose + 1,
	}, true
}

func functionAliasTargetSymbol(toks []scan.Token, start int, end int, topNames symbolNameTable, importRefs importSymbolTable) (unit.Symbol, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return unit.Symbol{}, false
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		unitName := symbolNameTableUnitName(topNames, toks[start].Text)
		if unitName == "" {
			return unit.Symbol{}, false
		}
		return unit.Symbol{Name: toks[start].Text, UnitName: unitName}, true
	}
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		group, ok := importSymbolTableGroup(importRefs, toks[start].Text)
		if !ok {
			return unit.Symbol{}, false
		}
		sym, ok := importFunctionSymbolByName(group, toks[start+2].Text)
		if !ok || sym.UnitName == "" {
			return unit.Symbol{}, false
		}
		return sym, true
	}
	return unit.Symbol{}, false
}

func methodValueAliasTargetInfo(toks []scan.Token, start int, end int, topNames symbolNameTable, methods methodTable, localTypes localTypeTable, fieldTypes structFieldTypeTable, unitName string, index int) (functionAliasInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+3 != end || toks[start].Kind != scan.Ident || toks[start+1].Text != "." || toks[start+2].Kind != scan.Ident {
		return functionAliasInfo{}, false
	}
	receiverType := localTypeTableLookup(localTypes, toks[start].Text)
	if receiverType.name == "" {
		return functionAliasInfo{}, false
	}
	methodName := methodLookupName(receiverType, toks[start+2].Text)
	method := methodTableLookup(methods, methodName)
	if method.unitName == "" {
		path, promotedReceiver, promotedMethod, ok := promotedStructMethodPath(fieldTypes, localTypeInfoOwnerName(receiverType), toks[start+2].Text, methods)
		if !ok {
			return functionAliasInfo{}, false
		}
		method = promotedMethod
		capture := promotedMethodReceiverArg(toks[start].Text, path, promotedReceiver, method)
		return methodValueAliasInfo(method, capture, unitName, index, topNames), true
	}
	capture := toks[start].Text
	if method.pointerReceiver && !receiverType.pointer {
		capture = "&" + capture
	} else if !method.pointerReceiver && receiverType.pointer {
		capture = "*" + capture
	}
	return methodValueAliasInfo(method, capture, unitName, index, topNames), true
}

func methodExpressionAliasTargetInfo(toks []scan.Token, start int, end int, methods methodTable, localTypes localTypeTable) (functionAliasInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		if localTypeTableLookup(localTypes, toks[start].Text).name != "" {
			return functionAliasInfo{}, false
		}
		methodName := toks[start].Text + "_" + toks[start+2].Text
		method := methodTableLookup(methods, methodName)
		if method.unitName == "" || method.pointerReceiver {
			return functionAliasInfo{}, false
		}
		return functionAliasInfo{unitName: method.unitName, symbol: methodUnitSymbol(method)}, true
	}
	if start+5 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident && toks[start+3].Text == "." && toks[start+4].Kind == scan.Ident {
		methodName := toks[start].Text + "." + toks[start+2].Text + "_" + toks[start+4].Text
		method := methodTableLookup(methods, methodName)
		if method.unitName == "" || method.pointerReceiver {
			return functionAliasInfo{}, false
		}
		return functionAliasInfo{unitName: method.unitName, symbol: methodUnitSymbol(method)}, true
	}
	return functionAliasInfo{}, false
}

func methodUnitSymbol(method methodInfo) unit.Symbol {
	if method.importPath == "" {
		return unit.Symbol{}
	}
	return unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName}
}

func compositeLiteralMethodValueAliasTargetInfo(source []byte, toks []scan.Token, start int, end int, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, fieldTypes structFieldTypeTable, unitName string, index int, refs *[]unit.Symbol) (functionAliasInfo, bool) {
	call, ok := compositeLiteralMethodValueAt(toks, start, end)
	if !ok {
		return functionAliasInfo{}, false
	}
	receiverType := typeInfoInRange(toks, call.typeStart, call.typeEnd)
	if receiverType.name == "" {
		return functionAliasInfo{}, false
	}
	receiverType.pointer = call.pointer
	methodName := methodLookupName(receiverType, toks[call.methodName].Text)
	method := methodTableLookup(methods, methodName)
	if method.unitName == "" {
		path, promotedReceiver, promotedMethod, promotedOK := promotedStructMethodPath(fieldTypes, localTypeInfoOwnerName(receiverType), toks[call.methodName].Text, methods)
		if !promotedOK {
			return functionAliasInfo{}, false
		}
		if promotedMethod.pointerReceiver && !promotedReceiver.pointer && !call.pointer {
			return functionAliasInfo{}, false
		}
		method = promotedMethod
		capture, ok := compositePromotedMethodReceiverArg(source, toks, call, path, promotedReceiver, method, topNames, importRefs, refs)
		if !ok {
			return functionAliasInfo{}, false
		}
		return methodValueAliasInfo(method, capture, unitName, index, topNames), true
	}
	if method.pointerReceiver && !receiverType.pointer {
		return functionAliasInfo{}, false
	}
	capture, ok := compositeMethodReceiverArg(source, toks, call, method, topNames, importRefs, refs)
	if !ok {
		return functionAliasInfo{}, false
	}
	return methodValueAliasInfo(method, capture, unitName, index, topNames), true
}

func methodValueAliasInfo(method methodInfo, capture string, unitName string, index int, topNames symbolNameTable) functionAliasInfo {
	receiver := unitName + "_method_value_receiver_tmp_" + strconv.Itoa(index)
	symbol := unit.Symbol{}
	if method.importPath != "" {
		symbol = unit.Symbol{ImportPath: method.importPath, Name: method.name, UnitName: method.unitName}
	}
	receiverType := methodValueReceiverTypeText(method, topNames)
	return functionAliasInfo{unitName: method.unitName, symbol: symbol, receiver: receiver, receiverType: receiverType, capture: capture}
}

func methodValueReceiverTypeText(method methodInfo, topNames symbolNameTable) string {
	typ := method.receiverType
	if method.importPath != "" {
		typ = SymbolName(method.importPath, method.receiverType)
	} else if lowered := symbolNameTableUnitName(topNames, method.receiverType); lowered != "" {
		typ = lowered
	}
	if method.pointerReceiver {
		return "*" + typ
	}
	return typ
}

func functionAliasDeclAt(aliases []functionAliasInfo, pos int) (functionAliasInfo, bool) {
	for i := 0; i < len(aliases); i++ {
		if aliases[i].declStart == pos {
			return aliases[i], true
		}
	}
	return functionAliasInfo{}, false
}

func functionAliasAt(aliases []functionAliasInfo, name string, pos int) (functionAliasInfo, bool) {
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
	return functionAliasInfo{}, false
}

func discardedFunctionValueStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, aliases []functionAliasInfo, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string) (int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return 0, false
	}
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return 0, false
		}
		if !discardedFunctionValueExpressionForLower(source, toks, rhs[i].start, rhs[i].end, aliases, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName) {
			return 0, false
		}
	}
	return stmtEnd, true
}

func discardedFunctionValueExpressionForLower(source []byte, toks []scan.Token, start int, end int, aliases []functionAliasInfo, functionNames symbolNameTable, topNames symbolNameTable, importRefs importSymbolTable, methods methodTable, localTypes localTypeTable, localTypeDecls []localTypeDeclInfo, fieldTypes structFieldTypeTable, namedArrays []namedArrayInfo, unitName string) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		if _, ok := functionAliasAt(aliases, toks[start].Text, int(toks[start].Start)); ok {
			return true
		}
	}
	var discardRefs []unit.Symbol
	if _, ok := staticAliasTargetInfo(source, toks, start, end, functionNames, topNames, importRefs, methods, localTypes, localTypeDecls, fieldTypes, namedArrays, unitName, 0, &discardRefs); ok {
		return true
	}
	return false
}

func discardedEmptyCompositeLiteralStatementForLowerAt(toks []scan.Token, pos int, limit int, topNames symbolNameTable, localTypeDecls []localTypeDeclInfo) (int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return 0, false
	}
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return 0, false
		}
		if !discardedKnownEmptyCompositeLiteralExpressionForLower(toks, rhs[i].start, rhs[i].end, topNames, localTypeDecls) {
			return 0, false
		}
	}
	return stmtEnd, true
}

func discardedKnownEmptyCompositeLiteralExpressionForLower(toks []scan.Token, start int, end int, topNames symbolNameTable, localTypeDecls []localTypeDeclInfo) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedKnownEmptyCompositeLiteralExpressionForLower(toks, start+1, close, topNames, localTypeDecls)
		}
	}
	if !emptyCompositeLiteralExpressionForLower(toks, start, end) {
		return false
	}
	name := toks[start].Text
	pos := int(toks[start].Start)
	return declaredTypeNameForEmptyCompositeForLower(toks, name, pos, topNames, localTypeDecls)
}

func declaredTypeNameForEmptyCompositeForLower(toks []scan.Token, name string, pos int, topNames symbolNameTable, localTypeDecls []localTypeDeclInfo) bool {
	for i := 0; i < len(localTypeDecls); i++ {
		decl := localTypeDecls[i]
		if decl.name == name && pos >= decl.scopeStart && pos < decl.scopeEnd {
			return true
		}
	}
	if topNames != nil && symbolNameTableUnitName(topNames, name) != "" {
		return true
	}
	for i := 0; i+1 < len(toks); i++ {
		if toks[i].Text != "type" {
			continue
		}
		if toks[i+1].Kind == scan.Ident && toks[i+1].Text == name {
			return true
		}
		if toks[i+1].Text != "(" {
			continue
		}
		close := findClose(toks, i+1, "(", ")")
		if close < 0 {
			continue
		}
		ranges := localConstSpecRanges(toks, i+2, close)
		for rangeIndex := 0; rangeIndex < len(ranges); rangeIndex++ {
			spec := ranges[rangeIndex]
			if spec.start < spec.end && toks[spec.start].Kind == scan.Ident && toks[spec.start].Text == name {
				return true
			}
		}
		i = close
	}
	return false
}

func discardedArrayLiteralStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return nil, 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return nil, 0, false
	}
	var lines []string
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return nil, 0, false
		}
		exprLines, ok := discardedArrayLiteralSideEffectStatementsForLower(body, toks, rhs[i].start, rhs[i].end, topNames)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, exprLines...)
	}
	return lines, stmtEnd, true
}

func discardedArrayLiteralExpressionForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedArrayLiteralExpressionForLower(toks, start+1, close)
		}
	}
	open := pureArrayCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardedPureCompositeElementsForLower(toks, open+1, close)
}

func discardedArrayLiteralSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedArrayLiteralSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	open := pureArrayCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return discardedCompositeSideEffectStatementsForLower(body, toks, open+1, close, topNames)
}

func pureArrayCompositeLiteralOpenForLower(toks []scan.Token, start int, end int) int {
	if start < 0 || start+1 >= end || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose < 0 || brackClose+1 >= end {
		return -1
	}
	if brackClose > start+1 {
		_, _, ok := arrayLengthForTokens(toks, start+1, brackClose)
		if !ok {
			return -1
		}
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

func discardedPureCompositeElementsForLower(toks []scan.Token, start int, end int) bool {
	values := topLevelExpressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardedPureCompositeKeyForLower(toks, value.start, colon) {
				return false
			}
			value.start = colon + 1
		}
		if !discardedPureCompositeValueForLower(toks, value.start, value.end) {
			return false
		}
	}
	return true
}

func discardedCompositeSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var lines []string
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardedPureCompositeKeyForLower(toks, value.start, colon) {
				return nil, false
			}
			value.start = colon + 1
		}
		valueLines, ok := discardedCompositeValueSideEffectStatementsForLower(body, toks, value.start, value.end, topNames)
		if !ok {
			return nil, false
		}
		lines = append(lines, valueLines...)
	}
	return lines, true
}

func discardedPureCompositeKeyForLower(toks []scan.Token, start int, end int) bool {
	_, imaginary, ok := signedNumberLiteralTextForLower(toks, start, end)
	return ok && !imaginary
}

func discardedPureCompositeValueForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardedPureCompositeElementsForLower(toks, start+1, close)
	}
	if toks[start].Text == "[" {
		return discardedArrayLiteralExpressionForLower(toks, start, end)
	}
	return discardedArrayLiteralElementForLower(toks, start, end)
}

func discardedCompositeValueSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return discardedCompositeSideEffectStatementsForLower(body, toks, start+1, close, topNames)
	}
	if toks[start].Text == "[" {
		return discardedArrayLiteralSideEffectStatementsForLower(body, toks, start, end, topNames)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	if call, ok := discardedDirectCallExpressionTextForLower(body, toks, start, end, topNames); ok {
		name := "rtg_discard_array_" + strconv.Itoa(int(toks[start].Start))
		return []string{name + " := " + call}, true
	}
	return nil, false
}

func discardedDirectCallExpressionTextForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) (string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedDirectCallExpressionTextForLower(body, toks, start+1, close, topNames)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 || expressionContainsCall(toks, start+2, close) {
		return "", false
	}
	unitName := symbolNameTableUnitName(topNames, toks[start].Text)
	if unitName == "" {
		return "", false
	}
	tail := body[int(toks[start].End):int(toks[close].End)]
	return unitName + tail, true
}

func discardedArrayLiteralElementForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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

func discardedMapLiteralStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return nil, 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return nil, 0, false
	}
	var lines []string
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return nil, 0, false
		}
		exprLines, ok := discardedMapLiteralSideEffectStatementsForLower(body, toks, rhs[i].start, rhs[i].end, topNames)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, exprLines...)
	}
	return lines, stmtEnd, true
}

func discardedMapMakeStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return nil, 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return nil, 0, false
	}
	var lines []string
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return nil, 0, false
		}
		exprLines, ok := discardedMapMakeSideEffectStatementsForLower(body, toks, rhs[i].start, rhs[i].end, topNames)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, exprLines...)
	}
	return lines, stmtEnd, true
}

func discardedMapSliceLiteralStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return nil, 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return nil, 0, false
	}
	var lines []string
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return nil, 0, false
		}
		exprLines, ok := discardedMapSliceLiteralSideEffectStatementsForLower(body, toks, rhs[i].start, rhs[i].end, topNames)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, exprLines...)
	}
	return lines, stmtEnd, true
}

func discardedMapMakeExpressionForLower(toks []scan.Token, start int, end int) bool {
	return mapMakeExpressionForLower(toks, start, end, false)
}

func lowerableMapMakeExpressionForLower(toks []scan.Token, start int, end int) bool {
	return mapMakeExpressionForLower(toks, start, end, true)
}

func mapMakeExpressionForLower(toks []scan.Token, start int, end int, allowDirectCallCapacity bool) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapMakeExpressionForLower(toks, start+1, close, allowDirectCallCapacity)
		}
	}
	if start+3 > end || toks[start].Text != "make" || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	args := topLevelExpressionRanges(toks, start+2, close)
	if len(args) != 1 && len(args) != 2 {
		return false
	}
	if !discardedMapTypeForLower(toks, args[0].start, args[0].end) {
		return false
	}
	if len(args) == 2 {
		if simpleIntegerLiteralKeyForLower(toks, args[1].start, args[1].end) {
			return true
		}
		return allowDirectCallCapacity && directCallExpressionWithoutNestedCallsForLower(toks, args[1].start, args[1].end)
	}
	return true
}

func discardedMapMakeSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedMapMakeSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	if !lowerableMapMakeExpressionForLower(toks, start, end) {
		return nil, false
	}
	open := start + 1
	close := findClose(toks, open, "(", ")")
	args := topLevelExpressionRanges(toks, open+1, close)
	if len(args) != 2 || simpleIntegerLiteralKeyForLower(toks, args[1].start, args[1].end) {
		return nil, true
	}
	call, ok := discardedDirectCallExpressionTextForLower(body, toks, args[1].start, args[1].end, topNames)
	if !ok {
		return nil, false
	}
	name := "rtg_discard_map_make_" + strconv.Itoa(int(toks[args[1].start].Start))
	return []string{name + " := " + call}, true
}

func lowerDiscardedMapMakeStatements(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "make") || !strings.Contains(body, "map") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		lines, stmtEnd, ok := discardedMapMakeStatementForLowerAt(body, toks, i, len(toks), topNames)
		if !ok {
			continue
		}
		text := ""
		if len(lines) > 0 {
			text = joinIndentedLines(lines, statementIndent(body, int(toks[i].Start)))
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
			start: int(toks[i].Start),
			end:   lowerStatementSourceEnd(toks, stmtEnd, int(toks[i].End)),
			text:  text,
		}})
		i = stmtEnd - 1
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func discardedMapTypeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start+4 > end || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return false
	}
	keyClose := findClose(toks, start+1, "[", "]")
	return keyClose > start+1 && keyClose+1 < end
}

func simpleIntegerLiteralKeyForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start < end && (toks[start].Text == "-" || toks[start].Text == "+") {
		start++
	}
	if start+1 != end || toks[start].Kind != scan.Number {
		return false
	}
	text := toks[start].Text
	if strings.Contains(text, ".") || strings.ContainsAny(text, "pPi") {
		return false
	}
	if !strings.HasPrefix(text, "0x") && !strings.HasPrefix(text, "0X") && strings.ContainsAny(text, "eE") {
		return false
	}
	_, err := strconv.ParseInt(text, 0, 64)
	return err == nil
}

func lowerDiscardedMapLiteralDeleteStatements(body string, topNames symbolNameTable) string {
	if !strings.Contains(body, "delete") || !strings.Contains(body, "map") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		lines, stmtEnd, ok := discardedMapLiteralDeleteStatementForLowerAt(body, toks, i, len(toks), topNames)
		if !ok {
			continue
		}
		text := ""
		if len(lines) > 0 {
			text = joinIndentedLines(lines, statementIndent(body, int(toks[i].Start)))
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
			start: int(toks[i].Start),
			end:   lowerStatementSourceEnd(toks, stmtEnd, int(toks[i].End)),
			text:  text,
		}})
		i = stmtEnd - 1
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func discardedMapLiteralDeleteStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "delete" {
		return nil, 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	close := findClose(toks, pos+1, "(", ")")
	if close < 0 || close != stmtEnd-1 {
		return nil, 0, false
	}
	lines, ok := discardedMapLiteralDeleteCallSideEffectStatementsForLower(body, toks, pos, close, topNames)
	if !ok {
		return nil, 0, false
	}
	return lines, stmtEnd, true
}

func discardedMapLiteralDeleteCallForLower(toks []scan.Token, pos int, close int) bool {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "delete" || toks[pos+1].Text != "(" {
		return false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return false
	}
	return discardedMapDeleteTargetExpressionForLower(toks, args[0].start, args[0].end) && discardedArrayLiteralElementForLower(toks, args[1].start, args[1].end)
}

func discardedMapLiteralDeleteCallSideEffectStatementsForLower(body string, toks []scan.Token, pos int, close int, topNames symbolNameTable) ([]string, bool) {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "delete" || toks[pos+1].Text != "(" {
		return nil, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return nil, false
	}
	if !discardedArrayLiteralElementForLower(toks, args[1].start, args[1].end) {
		return nil, false
	}
	if lowerableMapMakeExpressionForLower(toks, args[0].start, args[0].end) {
		return discardedMapMakeSideEffectStatementsForLower(body, toks, args[0].start, args[0].end, topNames)
	}
	return discardedMapLiteralSideEffectStatementsForLower(body, toks, args[0].start, args[0].end, topNames)
}

func discardedMapDeleteTargetExpressionForLower(toks []scan.Token, start int, end int) bool {
	return discardedMapLiteralExpressionForLower(toks, start, end) || lowerableMapMakeExpressionForLower(toks, start, end)
}

func discardedMapLiteralExpressionForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedMapLiteralExpressionForLower(toks, start+1, close)
		}
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return false
	}
	return discardedPureMapCompositeElementsForLower(toks, open+1, close)
}

func discardedMapLiteralSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedMapLiteralSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return discardedMapCompositeSideEffectStatementsForLower(body, toks, open+1, close, topNames)
}

func discardedMapSliceLiteralSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedMapSliceLiteralSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	open := mapSliceCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return discardedMapSliceCompositeSideEffectStatementsForLower(body, toks, open+1, close, topNames)
}

func mapSliceCompositeLiteralOpenForLower(toks []scan.Token, start int, end int) int {
	if start < 0 || start+4 >= end || toks[start].Text != "[" {
		return -1
	}
	brackClose := findClose(toks, start, "[", "]")
	if brackClose < 0 || brackClose+2 >= end {
		return -1
	}
	if brackClose > start+1 {
		_, _, ok := arrayLengthForTokens(toks, start+1, brackClose)
		if !ok {
			return -1
		}
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

func pureMapCompositeLiteralOpenForLower(toks []scan.Token, start int, end int) int {
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

func discardedPureMapCompositeElementsForLower(toks []scan.Token, start int, end int) bool {
	values := topLevelExpressionRanges(toks, start, end)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return false
		}
		if !discardedPureMapCompositeKeyForLower(toks, value.start, colon) {
			return false
		}
		if !discardedPureMapCompositeValueForLower(toks, colon+1, value.end) {
			return false
		}
	}
	return true
}

func discardedMapCompositeSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var lines []string
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return nil, false
		}
		if !discardedPureMapCompositeKeyForLower(toks, value.start, colon) {
			return nil, false
		}
		valueLines, ok := discardedMapCompositeValueSideEffectStatementsForLower(body, toks, colon+1, value.end, topNames)
		if !ok {
			return nil, false
		}
		lines = append(lines, valueLines...)
	}
	return lines, true
}

func discardedMapSliceCompositeSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var lines []string
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardedPureCompositeKeyForLower(toks, value.start, colon) {
				return nil, false
			}
			value.start = colon + 1
		}
		valueLines, ok := discardedMapSliceCompositeValueSideEffectStatementsForLower(body, toks, value.start, value.end, topNames)
		if !ok {
			return nil, false
		}
		lines = append(lines, valueLines...)
	}
	return lines, true
}

func discardedPureMapCompositeKeyForLower(toks []scan.Token, start int, end int) bool {
	return discardedArrayLiteralElementForLower(toks, start, end)
}

func discardedPureMapCompositeValueForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		return close == end-1 && discardedPureMapCompositeElementsForLower(toks, start+1, close)
	}
	if toks[start].Text == "map" {
		return discardedMapLiteralExpressionForLower(toks, start, end)
	}
	if toks[start].Text == "[" {
		return discardedArrayLiteralExpressionForLower(toks, start, end)
	}
	return discardedArrayLiteralElementForLower(toks, start, end)
}

func discardedMapCompositeValueSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return discardedMapCompositeSideEffectStatementsForLower(body, toks, start+1, close, topNames)
	}
	if toks[start].Text == "map" {
		return discardedMapLiteralSideEffectStatementsForLower(body, toks, start, end, topNames)
	}
	if toks[start].Text == "[" {
		return discardedArrayLiteralSideEffectStatementsForLower(body, toks, start, end, topNames)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" {
		return nil, true
	}
	if call, ok := discardedDirectCallExpressionTextForLower(body, toks, start, end, topNames); ok {
		name := "rtg_discard_map_" + strconv.Itoa(int(toks[start].Start))
		return []string{name + " := " + call}, true
	}
	return nil, false
}

func discardedMapSliceCompositeValueSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedMapSliceCompositeValueSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return discardedMapCompositeSideEffectStatementsForLower(body, toks, start+1, close, topNames)
	}
	if toks[start].Text == "map" {
		return discardedMapLiteralSideEffectStatementsForLower(body, toks, start, end, topNames)
	}
	if toks[start].Text == "make" {
		return discardedMapMakeSideEffectStatementsForLower(body, toks, start, end, topNames)
	}
	if start+1 == end && toks[start].Text == "nil" {
		return nil, true
	}
	return nil, false
}

type staticMapAliasForLower struct {
	name      string
	stmtStart int
	stmtEnd   int
	exprStart int
	exprEnd   int
	exprText  string
	keyType   string
	valueType string
	keys      []string
	keyIDs    []string
	values    []string
	temps     []expressionTemp
}

func lowerStaticMapAliases(body string) string {
	if !strings.Contains(body, "map") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	var aliases []staticMapAliasForLower
	for i := 0; i < len(toks); i++ {
		if alias, ok := staticMapAliasStatementForLowerAt(body, toks, i, len(toks)); ok {
			aliases = append(aliases, alias)
			text := ""
			if len(alias.temps) > 0 {
				var out []byte
				indent := statementIndent(body, int(toks[alias.stmtStart].Start))
				for tempIndex := 0; tempIndex < len(alias.temps); tempIndex++ {
					out = appendExpressionTempDecl(out, indent, alias.temps[tempIndex])
				}
				text = string(out)
			}
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
				start: int(toks[alias.stmtStart].Start),
				end:   int(toks[alias.stmtEnd-1].End),
				text:  text,
			}})
			i = alias.stmtEnd - 1
			continue
		}
		if stmtEnd, ok := staticMapAliasDeleteStatementForLowerAt(toks, i, len(toks), aliases); ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
				start: int(toks[i].Start),
				end:   int(toks[stmtEnd-1].End),
				text:  "",
			}})
			i = stmtEnd - 1
			continue
		}
		if text, stmtEnd, ok := staticMapAliasAssignmentStatementForLowerAt(body, toks, i, len(toks), aliases); ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
				start: int(toks[i].Start),
				end:   int(toks[stmtEnd-1].End),
				text:  text,
			}})
			i = stmtEnd - 1
			continue
		}
		if text, stmtEnd, ok := staticMapAliasIncDecStatementForLowerAt(body, toks, i, len(toks), aliases); ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
				start: int(toks[i].Start),
				end:   int(toks[stmtEnd-1].End),
				text:  text,
			}})
			i = stmtEnd - 1
			continue
		}
		if stmtEnd, ok := staticMapAliasBlankDiscardStatementForLowerAt(toks, i, len(toks), aliases); ok {
			replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
				start: int(toks[i].Start),
				end:   int(toks[stmtEnd-1].End),
				text:  "",
			}})
			i = stmtEnd - 1
			continue
		}
		if toks[i].Kind != scan.Ident {
			continue
		}
		alias, ok := staticMapAliasForLowerAt(aliases, toks[i].Text, i)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{{
			start: int(toks[i].Start),
			end:   int(toks[i].End),
			text:  "(" + staticMapAliasExpressionText(alias) + ")",
		}})
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func sortExpressionReplacementsByStart(replacements []expressionReplacement) {
	for i := 1; i < len(replacements); i++ {
		value := replacements[i]
		j := i - 1
		for j >= 0 && replacements[j].start > value.start {
			replacements[j+1] = replacements[j]
			j--
		}
		replacements[j+1] = value
	}
}

func staticMapAliasesForLower(body string, toks []scan.Token) []staticMapAliasForLower {
	var aliases []staticMapAliasForLower
	for i := 0; i < len(toks); i++ {
		alias, ok := staticMapAliasStatementForLowerAt(body, toks, i, len(toks))
		if !ok {
			continue
		}
		aliases = append(aliases, alias)
		i = alias.stmtEnd - 1
	}
	return aliases
}

func staticMapAliasForLowerAt(aliases []staticMapAliasForLower, name string, pos int) (staticMapAliasForLower, bool) {
	var found staticMapAliasForLower
	ok := false
	for i := 0; i < len(aliases); i++ {
		alias := aliases[i]
		if alias.name != name || alias.stmtEnd > pos {
			continue
		}
		if pos >= alias.stmtStart && pos < alias.stmtEnd {
			continue
		}
		found = alias
		ok = true
	}
	return found, ok
}

func staticMapAliasExpressionInfoForLower(body string, toks []scan.Token, start int, end int) (mapRangeInfo, bool) {
	if info, ok := mapRangeInfoForLower(body, toks, start, end); ok {
		return info, true
	}
	return staticMapAliasLowerableExpressionInfoForLower(body, toks, start, end)
}

func staticMapAliasLowerableExpressionInfoForLower(body string, toks []scan.Token, start int, end int) (mapRangeInfo, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return mapRangeInfo{}, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasLowerableExpressionInfoForLower(body, toks, start+1, close)
		}
	}
	if lowerableMapMakeExpressionForLower(toks, start, end) {
		keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
		if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
			return mapRangeInfo{}, false
		}
		temps, ok := staticMapAliasMakeSideEffectTempsForLower(body, toks, start, end)
		if !ok {
			return mapRangeInfo{}, false
		}
		return mapRangeInfo{keyType: keyType, valueType: valueType, temps: temps}, true
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return mapRangeInfo{}, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return mapRangeInfo{}, false
	}
	keyType, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
	if !mapRangeKeyTypeSupportedForLower(keyType) || !mapRangeValueTypeSupportedForLower(valueType) {
		return mapRangeInfo{}, false
	}
	info := mapRangeInfo{keyType: keyType, valueType: valueType}
	values := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return mapRangeInfo{}, false
		}
		keyStart, keyEnd := trimTokenRange(toks, value.start, colon)
		if keyStart >= keyEnd {
			return mapRangeInfo{}, false
		}
		keyID, ok := mapLiteralComparableKeyForLower(toks, value.start, colon)
		if !ok {
			return mapRangeInfo{}, false
		}
		valueText, valueTemps, ok := staticMapAliasInitializerValueForLower(body, toks, colon+1, value.end)
		if !ok {
			return mapRangeInfo{}, false
		}
		info.keys = append(info.keys, strings.TrimSpace(tokenRangeText(body, toks, keyStart, keyEnd)))
		info.keyIDs = append(info.keyIDs, keyID)
		info.values = append(info.values, valueText)
		info.temps = appendExpressionTemps(info.temps, valueTemps)
	}
	return info, true
}

func staticMapAliasInitializerValueForLower(body string, toks []scan.Token, start int, end int) (string, []expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasInitializerValueForLower(body, toks, start+1, close)
		}
	}
	if discardedArrayLiteralElementForLower(toks, start, end) || singleIdentifierExpressionInLower(toks, start, end) != "" {
		return strings.TrimSpace(tokenRangeText(body, toks, start, end)), nil, true
	}
	if !directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return "", nil, false
	}
	name := "rtg_static_map_alias_" + strconv.Itoa(int(toks[start].Start))
	expr := strings.TrimSpace(tokenRangeText(body, toks, start, end))
	return name, []expressionTemp{{name: name, expr: expr}}, true
}

func staticMapAliasMakeSideEffectTempsForLower(body string, toks []scan.Token, start int, end int) ([]expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasMakeSideEffectTempsForLower(body, toks, start+1, close)
		}
	}
	if !lowerableMapMakeExpressionForLower(toks, start, end) {
		return nil, false
	}
	open := start + 1
	close := findClose(toks, open, "(", ")")
	args := topLevelExpressionRanges(toks, open+1, close)
	if len(args) != 2 || simpleIntegerLiteralKeyForLower(toks, args[1].start, args[1].end) {
		return nil, true
	}
	name := "rtg_static_map_alias_" + strconv.Itoa(int(toks[args[1].start].Start))
	expr := strings.TrimSpace(tokenRangeText(body, toks, args[1].start, args[1].end))
	return []expressionTemp{{name: name, expr: expr}}, true
}

func staticMapAliasStatementForLowerAt(body string, toks []scan.Token, pos int, limit int) (staticMapAliasForLower, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) {
		return staticMapAliasForLower{}, false
	}
	stmtStart := pos
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, limit)
	if toks[pos].Text == "var" {
		return staticMapAliasVarStatementForLower(body, toks, stmtStart, stmtEnd)
	}
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		return staticMapAliasForLower{}, false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return staticMapAliasForLower{}, false
	}
	name := singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end)
	if name == "" || name == "_" {
		return staticMapAliasForLower{}, false
	}
	info, ok := staticMapAliasExpressionInfoForLower(body, toks, rhs[0].start, rhs[0].end)
	if !ok {
		return staticMapAliasForLower{}, false
	}
	exprStart, exprEnd := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return staticMapAliasForLower{name: name, stmtStart: stmtStart, stmtEnd: stmtEnd, exprStart: exprStart, exprEnd: exprEnd, exprText: strings.TrimSpace(tokenRangeText(body, toks, exprStart, exprEnd)), keyType: info.keyType, valueType: info.valueType, keys: info.keys, keyIDs: info.keyIDs, values: info.values, temps: info.temps}, true
}

func staticMapAliasVarStatementForLower(body string, toks []scan.Token, stmtStart int, stmtEnd int) (staticMapAliasForLower, bool) {
	if stmtStart+1 >= stmtEnd || toks[stmtStart].Text != "var" {
		return staticMapAliasForLower{}, false
	}
	assign := findTopLevelToken(toks, stmtStart+1, stmtEnd, "=")
	if assign < 0 {
		return staticMapAliasForLower{}, false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart+1, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return staticMapAliasForLower{}, false
	}
	nameStart, nameEnd := trimTokenRange(toks, lhs[0].start, lhs[0].end)
	if nameStart >= nameEnd || toks[nameStart].Kind != scan.Ident || toks[nameStart].Text == "_" {
		return staticMapAliasForLower{}, false
	}
	info, ok := staticMapAliasExpressionInfoForLower(body, toks, rhs[0].start, rhs[0].end)
	if !ok {
		return staticMapAliasForLower{}, false
	}
	exprStart, exprEnd := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return staticMapAliasForLower{name: toks[nameStart].Text, stmtStart: stmtStart, stmtEnd: stmtEnd, exprStart: exprStart, exprEnd: exprEnd, exprText: strings.TrimSpace(tokenRangeText(body, toks, exprStart, exprEnd)), keyType: info.keyType, valueType: info.valueType, keys: info.keys, keyIDs: info.keyIDs, values: info.values, temps: info.temps}, true
}

func staticMapAliasDeleteStatementForLowerAt(toks []scan.Token, pos int, limit int, aliases []staticMapAliasForLower) (int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "delete" {
		return 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	close := findClose(toks, pos+1, "(", ")")
	if close < 0 || close != stmtEnd-1 {
		return 0, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 2 {
		return 0, false
	}
	name := singleIdentifierExpressionInLower(toks, args[0].start, args[0].end)
	if name == "" {
		return 0, false
	}
	keyID, ok := mapLiteralComparableKeyForLower(toks, args[1].start, args[1].end)
	if !ok {
		return 0, false
	}
	for i := len(aliases) - 1; i >= 0; i-- {
		if aliases[i].name != name || aliases[i].stmtEnd > pos {
			continue
		}
		staticMapAliasDeleteKey(&aliases[i], keyID)
		return stmtEnd, true
	}
	return 0, false
}

func staticMapAliasDeleteKey(alias *staticMapAliasForLower, keyID string) {
	for i := 0; i < len(alias.keyIDs); i++ {
		if alias.keyIDs[i] != keyID {
			continue
		}
		alias.keys = append(alias.keys[:i], alias.keys[i+1:]...)
		alias.keyIDs = append(alias.keyIDs[:i], alias.keyIDs[i+1:]...)
		alias.values = append(alias.values[:i], alias.values[i+1:]...)
		i--
	}
}

func staticMapAliasAssignmentStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, aliases []staticMapAliasForLower) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return "", 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 {
		return "", 0, false
	}
	lhsEnd := assign
	compoundOp := ""
	if lowerCompoundAssignmentEquals(toks, assign) {
		lhsEnd = assign - 1
		compoundOp = toks[assign-1].Text
	}
	lhs := topLevelExpressionRanges(toks, pos, lhsEnd)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return "", 0, false
	}
	lhsStart, lhsEnd := trimTokenRange(toks, lhs[0].start, lhs[0].end)
	if lhsStart+3 > lhsEnd || toks[lhsStart].Kind != scan.Ident || toks[lhsStart+1].Text != "[" {
		return "", 0, false
	}
	indexClose := findClose(toks, lhsStart+1, "[", "]")
	if indexClose != lhsEnd-1 {
		return "", 0, false
	}
	keyID, ok := mapLiteralComparableKeyForLower(toks, lhsStart+2, indexClose)
	if !ok {
		return "", 0, false
	}
	valueStart, valueEnd := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if valueStart >= valueEnd {
		return "", 0, false
	}
	name := toks[lhsStart].Text
	for i := len(aliases) - 1; i >= 0; i-- {
		if aliases[i].name != name || aliases[i].stmtEnd > pos {
			continue
		}
		if compoundOp != "" && !staticMapAliasCompoundAssignmentOperatorSupportedForLower(compoundOp, aliases[i].valueType) {
			return "", 0, false
		}
		valueText, replacementText, valueOK := staticMapAliasAssignmentValueForLower(body, toks, rhs[0].start, rhs[0].end, aliases[i].valueType)
		if !valueOK {
			return "", 0, false
		}
		if compoundOp != "" {
			oldValue, ok := staticMapAliasValueForKey(aliases[i], keyID)
			if !ok {
				oldValue = mapRangeZeroValueTextForLower(aliases[i].valueType)
			}
			resultName := "rtg_static_map_alias_" + strconv.Itoa(int(toks[assign-1].Start))
			resultText := resultName + " := (" + oldValue + ") " + compoundOp + " (" + valueText + ")"
			if replacementText != "" {
				indent := statementIndent(body, int(toks[pos].Start))
				replacementText = replacementText + "\n" + indent + resultText
			} else {
				replacementText = resultText
			}
			valueText = resultName
		}
		keyText := strings.TrimSpace(tokenRangeText(body, toks, lhsStart+2, indexClose))
		staticMapAliasSetKey(&aliases[i], keyText, keyID, valueText)
		return replacementText, stmtEnd, true
	}
	return "", 0, false
}

func staticMapAliasIncDecStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, aliases []staticMapAliasForLower) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Kind != scan.Ident {
		return "", 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	if pos+3 >= stmtEnd || toks[pos+1].Text != "[" {
		return "", 0, false
	}
	indexClose := findClose(toks, pos+1, "[", "]")
	incDec := indexClose + 1
	if indexClose < 0 || incDec+2 != stmtEnd || !lowerIncDecAt(toks, incDec, stmtEnd) {
		return "", 0, false
	}
	keyID, ok := mapLiteralComparableKeyForLower(toks, pos+2, indexClose)
	if !ok {
		return "", 0, false
	}
	name := toks[pos].Text
	for i := len(aliases) - 1; i >= 0; i-- {
		if aliases[i].name != name || aliases[i].stmtEnd > pos {
			continue
		}
		if !staticMapAliasIncDecSupportedForLower(aliases[i].valueType) {
			return "", 0, false
		}
		oldValue, ok := staticMapAliasValueForKey(aliases[i], keyID)
		if !ok {
			oldValue = mapRangeZeroValueTextForLower(aliases[i].valueType)
		}
		op := toks[incDec].Text
		resultName := "rtg_static_map_alias_" + strconv.Itoa(int(toks[incDec].Start))
		resultText := resultName + " := (" + oldValue + ") " + op + " (1)"
		keyText := strings.TrimSpace(tokenRangeText(body, toks, pos+2, indexClose))
		staticMapAliasSetKey(&aliases[i], keyText, keyID, resultName)
		return resultText, stmtEnd, true
	}
	return "", 0, false
}

func staticMapAliasBlankDiscardStatementForLowerAt(toks []scan.Token, pos int, limit int, aliases []staticMapAliasForLower) (int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return 0, false
	}
	name := singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end)
	if name != "_" {
		return 0, false
	}
	aliasName := staticMapAliasIdentifierExpressionForLower(toks, rhs[0].start, rhs[0].end)
	if aliasName == "" {
		return 0, false
	}
	for i := len(aliases) - 1; i >= 0; i-- {
		if aliases[i].name == aliasName && aliases[i].stmtEnd <= pos {
			return stmtEnd, true
		}
	}
	return 0, false
}

func staticMapAliasIdentifierExpressionForLower(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasIdentifierExpressionForLower(toks, start+1, close)
		}
	}
	return singleIdentifierExpressionInLower(toks, start, end)
}

func staticMapAliasSetKey(alias *staticMapAliasForLower, key string, keyID string, value string) {
	for i := 0; i < len(alias.keyIDs); i++ {
		if alias.keyIDs[i] != keyID {
			continue
		}
		alias.keys[i] = key
		alias.values[i] = value
		return
	}
	alias.keys = append(alias.keys, key)
	alias.keyIDs = append(alias.keyIDs, keyID)
	alias.values = append(alias.values, value)
}

func staticMapAliasValueForKey(alias staticMapAliasForLower, keyID string) (string, bool) {
	for i := 0; i < len(alias.keyIDs); i++ {
		if alias.keyIDs[i] == keyID {
			return alias.values[i], true
		}
	}
	return "", false
}

func staticMapAliasIncDecSupportedForLower(valueType string) bool {
	switch valueType {
	case "int", "int16", "int32", "int64", "byte", "float64":
		return true
	}
	return false
}

func staticMapAliasCompoundAssignmentOperatorSupportedForLower(op string, valueType string) bool {
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

func staticMapAliasAssignmentValueForLower(body string, toks []scan.Token, start int, end int, valueType string) (string, string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentValueForLower(body, toks, start+1, close, valueType)
		}
	}
	if staticMapAliasAssignmentLiteralValueSupportedForLower(toks, start, end, valueType) || singleIdentifierExpressionInLower(toks, start, end) != "" {
		return strings.TrimSpace(tokenRangeText(body, toks, start, end)), "", true
	}
	if !directCallExpressionWithoutNestedCallsForLower(toks, start, end) && !(valueType == "int" && staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks, start, end)) && !(valueType == "string" && staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks, start, end)) && !(valueType == "bool" && staticMapAliasAssignmentBoolBinaryValueSupportedForLower(toks, start, end)) {
		return "", "", false
	}
	name := "rtg_static_map_alias_" + strconv.Itoa(int(toks[start].Start))
	expr := strings.TrimSpace(tokenRangeText(body, toks, start, end))
	return name, name + " := " + expr, true
}

func staticMapAliasAssignmentValueSupportedForLower(toks []scan.Token, start int, end int, valueType string) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentValueSupportedForLower(toks, start+1, close, valueType)
		}
	}
	return staticMapAliasAssignmentLiteralValueSupportedForLower(toks, start, end, valueType) || singleIdentifierExpressionInLower(toks, start, end) != "" || directCallExpressionWithoutNestedCallsForLower(toks, start, end) || (valueType == "int" && staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks, start, end)) || (valueType == "string" && staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks, start, end)) || (valueType == "bool" && staticMapAliasAssignmentBoolBinaryValueSupportedForLower(toks, start, end))
}

func staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks, start+1, close)
		}
	}
	if signedNumberLiteralForLower(toks, start, end) || singleIdentifierExpressionInLower(toks, start, end) != "" || directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return true
	}
	op := staticMapAliasAssignmentIntBinaryOperatorForLower(toks, start, end)
	if op < 0 {
		return false
	}
	return staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks, start, op) && staticMapAliasAssignmentIntBinaryValueSupportedForLower(toks, op+1, end)
}

func staticMapAliasAssignmentIntBinaryOperatorForLower(toks []scan.Token, start int, end int) int {
	best := -1
	bestPrec := 100
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end {
			prec := staticMapAliasAssignmentIntBinaryOperatorPrecedenceForLower(toks[i].Text)
			if prec > 0 && prec <= bestPrec {
				best = i
				bestPrec = prec
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return best
}

func staticMapAliasAssignmentIntBinaryOperatorPrecedenceForLower(op string) int {
	switch op {
	case "+", "-", "|", "^":
		return 4
	case "*", "/", "%", "<<", ">>", "&", "&^":
		return 5
	}
	return 0
}

func staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks, start+1, close)
		}
	}
	if start+1 == end && toks[start].Kind == scan.String {
		return true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" || directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return true
	}
	op := staticMapAliasAssignmentStringBinaryOperatorForLower(toks, start, end)
	if op < 0 {
		return false
	}
	return staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks, start, op) && staticMapAliasAssignmentStringBinaryValueSupportedForLower(toks, op+1, end)
}

func staticMapAliasAssignmentStringBinaryOperatorForLower(toks []scan.Token, start int, end int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := end - 1; i >= start; i-- {
		text := toks[i].Text
		if text == ")" {
			paren++
			continue
		}
		if text == "]" {
			brack++
			continue
		}
		if text == "}" {
			brace++
			continue
		}
		if text == "(" {
			paren--
			continue
		}
		if text == "[" {
			brack--
			continue
		}
		if text == "{" {
			brace--
			continue
		}
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end && text == "+" {
			return i
		}
	}
	return -1
}

func staticMapAliasAssignmentBoolBinaryValueSupportedForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentBoolBinaryValueSupportedForLower(toks, start+1, close)
		}
	}
	if start+1 == end && (toks[start].Text == "true" || toks[start].Text == "false") {
		return true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" || directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return true
	}
	op := staticMapAliasAssignmentBoolBinaryOperatorForLower(toks, start, end)
	if op < 0 {
		return false
	}
	switch toks[op].Text {
	case "==", "!=", "<", "<=", ">", ">=":
		return staticMapAliasAssignmentComparableValueSupportedForLower(toks, start, op) && staticMapAliasAssignmentComparableValueSupportedForLower(toks, op+1, end)
	}
	return false
}

func staticMapAliasAssignmentBoolBinaryOperatorForLower(toks []scan.Token, start int, end int) int {
	best := -1
	bestPrec := 100
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end {
			prec := staticMapAliasAssignmentBoolBinaryOperatorPrecedenceForLower(toks[i].Text)
			if prec > 0 && prec <= bestPrec {
				best = i
				bestPrec = prec
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return best
}

func staticMapAliasAssignmentBoolBinaryOperatorPrecedenceForLower(op string) int {
	switch op {
	case "==", "!=", "<", "<=", ">", ">=":
		return 3
	}
	return 0
}

func staticMapAliasAssignmentComparableValueSupportedForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentComparableValueSupportedForLower(toks, start+1, close)
		}
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" || directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return true
	}
	if start+1 == end {
		return toks[start].Kind == scan.Number || toks[start].Kind == scan.String || toks[start].Kind == scan.Char || toks[start].Text == "true" || toks[start].Text == "false"
	}
	return signedNumberLiteralForLower(toks, start, end)
}

func staticMapAliasAssignmentLiteralValueSupportedForLower(toks []scan.Token, start int, end int, valueType string) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticMapAliasAssignmentLiteralValueSupportedForLower(toks, start+1, close, valueType)
		}
	}
	if valueType == "string" {
		return start+1 == end && toks[start].Kind == scan.String
	}
	if valueType == "bool" {
		return start+1 == end && (toks[start].Text == "true" || toks[start].Text == "false")
	}
	if strings.HasPrefix(valueType, "*") || strings.HasPrefix(valueType, "[]") {
		return start+1 == end && toks[start].Text == "nil"
	}
	if valueType == "byte" {
		if start+1 == end && toks[start].Kind == scan.Char {
			return true
		}
		return signedNumberLiteralForLower(toks, start, end)
	}
	if valueType == "int" || valueType == "int16" || valueType == "int32" || valueType == "int64" || valueType == "float64" {
		return signedNumberLiteralForLower(toks, start, end)
	}
	return false
}

func signedNumberLiteralForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start < end && (toks[start].Text == "-" || toks[start].Text == "+") {
		start++
	}
	return start+1 == end && toks[start].Kind == scan.Number
}

func staticMapAliasExpressionText(alias staticMapAliasForLower) string {
	out := "map[" + alias.keyType + "]" + alias.valueType + "{"
	for i := 0; i < len(alias.keys) && i < len(alias.values); i++ {
		if i > 0 {
			out += ", "
		}
		out += alias.keys[i] + ": " + alias.values[i]
	}
	out += "}"
	return out
}

func lowerMapLiteralCommaOkAssignments(body string) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "[") || !strings.Contains(body, ",") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, close, ok := mapLiteralCommaOkAssignmentReplacement(body, toks, i)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func mapLiteralCommaOkAssignmentReplacement(body string, toks []scan.Token, assign int) (expressionReplacement, int, bool) {
	if assign < 0 || assign >= len(toks) || (toks[assign].Text != ":=" && toks[assign].Text != "=") {
		return expressionReplacement{}, 0, false
	}
	if lowerCompoundAssignmentEquals(toks, assign) {
		return expressionReplacement{}, 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, assign)
	for stmtStart < assign && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
		stmtStart++
	}
	stmtEnd := lowerSimpleStatementEnd(toks, assign+1, len(toks))
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 2 || len(rhs) != 1 {
		return expressionReplacement{}, 0, false
	}
	valueText, found, ok := mapLiteralIndexValueOKTextForLower(body, toks, rhs[0].start, rhs[0].end)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	okText := "false"
	if found {
		okText = "true"
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start >= end {
		return expressionReplacement{}, 0, false
	}
	return expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[end-1].End),
		text:  valueText + ", " + okText,
	}, stmtEnd, true
}

func normalizeMapLiteralCommaOkAssignment(body string, toks []scan.Token, assign int, unitName string, tempIndex *int) ([]expressionTemp, expressionReplacement, int, int, bool) {
	if assign < 0 || assign >= len(toks) || (toks[assign].Text != ":=" && toks[assign].Text != "=") {
		return nil, expressionReplacement{}, 0, 0, false
	}
	if lowerCompoundAssignmentEquals(toks, assign) {
		return nil, expressionReplacement{}, 0, 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, assign)
	for stmtStart < assign && (toks[stmtStart].Text == "if" || toks[stmtStart].Text == "for" || toks[stmtStart].Text == "switch") {
		stmtStart++
	}
	stmtEnd := lowerSimpleStatementEnd(toks, assign+1, len(toks))
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 2 || len(rhs) != 1 {
		return nil, expressionReplacement{}, 0, 0, false
	}
	valueText, found, temps, ok := lowerableMapLiteralIndexValueOKForNormalize(body, toks, rhs[0].start, rhs[0].end, unitName, tempIndex)
	if !ok {
		return nil, expressionReplacement{}, 0, 0, false
	}
	okText := "false"
	if found {
		okText = "true"
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start >= end {
		return nil, expressionReplacement{}, 0, 0, false
	}
	return temps, expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[end-1].End),
		text:  valueText + ", " + okText,
	}, stmtStart, stmtEnd, true
}

func normalizeMapLiteralCommaOkSimpleStatement(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, []expressionReplacement, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, nil, false
	}
	assign := findTopLevelToken(toks, start, end, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, start, end, "=")
	}
	if assign < 0 {
		return nil, nil, false
	}
	temps, repl, stmtStart, stmtEnd, ok := normalizeMapLiteralCommaOkAssignment(body, toks, assign, unitName, tempIndex)
	if !ok || stmtStart != start || stmtEnd != end {
		return nil, nil, false
	}
	return temps, []expressionReplacement{repl}, true
}

func lowerMapLiteralIndexExpressions(body string) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "[") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(toks); i++ {
		repl, close, ok := mapLiteralIndexReplacement(body, toks, i)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func mapLiteralIndexReplacement(body string, toks []scan.Token, pos int) (expressionReplacement, int, bool) {
	start, end, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, pos, len(toks))
	if !ok {
		return expressionReplacement{}, 0, false
	}
	key, ok := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	text, ok := pureMapLiteralIndexValueTextForLower(body, toks, mapStart, mapEnd, key)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	return expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[indexClose].End),
		text:  text,
	}, end - 1, true
}

func pureMapLiteralIndexValueTextForLower(body string, toks []scan.Token, start int, end int, key string) (string, bool) {
	text, _, ok := pureMapLiteralIndexValueOKTextForLower(body, toks, start, end, key)
	return text, ok
}

func pureMapLiteralIndexValueOKTextForLower(body string, toks []scan.Token, start int, end int, key string) (string, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	if discardedMapMakeExpressionForLower(toks, start, end) {
		text, ok := mapLiteralSimpleZeroValueTextForLower(toks, start, end)
		return text, false, ok
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return "", false, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return "", false, false
	}
	if !discardedPureMapCompositeElementsForLower(toks, open+1, close) {
		return "", false, false
	}
	values := topLevelExpressionRanges(toks, open+1, close)
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return "", false, false
		}
		valueKey, ok := mapLiteralComparableKeyForLower(toks, value.start, colon)
		if !ok || valueKey != key {
			continue
		}
		if !discardedArrayLiteralElementForLower(toks, colon+1, value.end) {
			return "", false, false
		}
		valueStart, valueEnd := trimTokenRange(toks, colon+1, value.end)
		if valueStart >= valueEnd {
			return "", false, false
		}
		return strings.TrimSpace(tokenRangeText(body, toks, valueStart, valueEnd)), true, true
	}
	text, ok := mapLiteralSimpleZeroValueTextForLower(toks, start, end)
	return text, false, ok
}

func mapLiteralIndexValueOKTextForLower(body string, toks []scan.Token, start int, end int) (string, bool, bool) {
	start, end = trimTokenRange(toks, start, end)
	exprStart, exprEnd, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, start, end)
	if !ok || exprStart != start || exprEnd != end {
		return "", false, false
	}
	key, ok := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
	if !ok {
		return "", false, false
	}
	return pureMapLiteralIndexValueOKTextForLower(body, toks, mapStart, mapEnd, key)
}

func mapLiteralIndexExpressionPartsForLower(toks []scan.Token, pos int, limit int) (int, int, int, int, int, int, bool) {
	start := pos
	mapStart := pos
	mapEnd := -1
	indexOpen := -1
	if pos < 0 || pos >= len(toks) || pos >= limit {
		return 0, 0, 0, 0, 0, 0, false
	}
	if toks[pos].Text == "(" {
		closeParen := findClose(toks, pos, "(", ")")
		if closeParen < 0 || closeParen+1 >= len(toks) || closeParen+1 >= limit || toks[closeParen+1].Text != "[" {
			return 0, 0, 0, 0, 0, 0, false
		}
		mapStart = pos + 1
		mapEnd = closeParen
		indexOpen = closeParen + 1
	} else if toks[pos].Text == "make" && pos+1 < len(toks) && toks[pos+1].Text == "(" {
		close := findClose(toks, pos+1, "(", ")")
		if close < 0 || close+1 >= len(toks) || close+1 >= limit || toks[close+1].Text != "[" {
			return 0, 0, 0, 0, 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	} else {
		open := pureMapCompositeLiteralOpenForLower(toks, mapStart, limit)
		if open < 0 {
			return 0, 0, 0, 0, 0, 0, false
		}
		close := findClose(toks, open, "{", "}")
		if close < 0 || close+1 >= len(toks) || close+1 >= limit || toks[close+1].Text != "[" {
			return 0, 0, 0, 0, 0, 0, false
		}
		mapEnd = close + 1
		indexOpen = close + 1
	}
	indexClose := findClose(toks, indexOpen, "[", "]")
	if indexClose < 0 || indexClose >= limit {
		return 0, 0, 0, 0, 0, 0, false
	}
	return start, indexClose + 1, mapStart, mapEnd, indexOpen, indexClose, true
}

func mapLiteralSimpleZeroValueTextForLower(toks []scan.Token, start int, end int) (string, bool) {
	typ := mapLiteralElementTypeTextForLower(toks, start, end)
	switch typ {
	case "bool":
		return "false", true
	case "string":
		return "\"\"", true
	case "int", "int16", "int32", "int64", "byte", "float64":
		return "0", true
	}
	if strings.HasPrefix(typ, "*") || strings.HasPrefix(typ, "[]") {
		return "nil", true
	}
	return "", false
}

func mapLiteralElementTypeTextForLower(toks []scan.Token, start int, end int) string {
	_, valueType := mapExpressionKeyValueTypeTextForLower(toks, start, end)
	return valueType
}

func mapExpressionKeyValueTypeTextForLower(toks []scan.Token, start int, end int) (string, string) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapExpressionKeyValueTypeTextForLower(toks, start+1, close)
		}
	}
	if start+1 < end && toks[start].Text == "make" && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close != end-1 {
			return "", ""
		}
		args := topLevelExpressionRanges(toks, start+2, close)
		if len(args) != 1 && len(args) != 2 {
			return "", ""
		}
		return mapTypeKeyValueTextForLower(toks, args[0].start, args[0].end)
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 || start+1 >= open || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return "", ""
	}
	return mapTypeKeyValueTextForLower(toks, start, open)
}

func mapTypeElementTextForLower(toks []scan.Token, start int, end int) string {
	_, valueType := mapTypeKeyValueTextForLower(toks, start, end)
	return valueType
}

func mapTypeKeyValueTextForLower(toks []scan.Token, start int, end int) (string, string) {
	start, end = trimTokenRange(toks, start, end)
	if start+3 > end || toks[start].Text != "map" || toks[start+1].Text != "[" {
		return "", ""
	}
	keyClose := findClose(toks, start+1, "[", "]")
	if keyClose < 0 || keyClose+1 >= end {
		return "", ""
	}
	keyType := mapLiteralTypeTextForLower(toks, start+2, keyClose)
	valueType := mapLiteralTypeTextForLower(toks, keyClose+1, end)
	return keyType, valueType
}

func mapLiteralTypeTextForLower(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	var out []byte
	for i := start; i < end; i++ {
		out = append(out, toks[i].Text...)
	}
	return string(out)
}

func mapLiteralComparableKeyForLower(toks []scan.Token, start int, end int) (string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return mapLiteralComparableKeyForLower(toks, start+1, close)
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

func lowerMapLiteralLenCalls(body string) string {
	if !strings.Contains(body, "map") || !strings.Contains(body, "len") {
		return body
	}
	mark := arena.Mark()
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		arena.Reset(mark)
		return body
	}
	if !tokenSpansMatchSource(body, toks) {
		arena.Reset(mark)
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i+1 < len(toks); i++ {
		repl, close, ok := mapLiteralLenReplacement(toks, i)
		if !ok {
			continue
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		i = close
	}
	if len(replacements) == 0 {
		arena.Reset(mark)
		return body
	}
	body = applyExpressionReplacements(body, 0, len(body), replacements)
	body = arena.PersistString(body)
	arena.Reset(mark)
	return body
}

func mapLiteralLenReplacement(toks []scan.Token, pos int) (expressionReplacement, int, bool) {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "len" || toks[pos+1].Text != "(" {
		return expressionReplacement{}, 0, false
	}
	if pos > 0 && toks[pos-1].Text == "." {
		return expressionReplacement{}, 0, false
	}
	close := findClose(toks, pos+1, "(", ")")
	if close < 0 {
		return expressionReplacement{}, 0, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 1 {
		return expressionReplacement{}, 0, false
	}
	count, ok := pureMapLiteralElementCountForLower(toks, args[0].start, args[0].end)
	text := ""
	if ok {
		text = strconv.Itoa(count)
	} else if discardedMapMakeExpressionForLower(toks, args[0].start, args[0].end) {
		text = "0"
	} else {
		return expressionReplacement{}, 0, false
	}
	return expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[close].End),
		text:  text,
	}, close, true
}

func pureMapLiteralElementCountForLower(toks []scan.Token, start int, end int) (int, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return 0, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return pureMapLiteralElementCountForLower(toks, start+1, close)
		}
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return 0, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return 0, false
	}
	if !discardedPureMapCompositeElementsForLower(toks, open+1, close) {
		return 0, false
	}
	values := topLevelExpressionRanges(toks, open+1, close)
	return len(values), true
}

func normalizeMapLiteralIndexExpression(body string, toks []scan.Token, pos int, limit int, unitName string, tempIndex *int) ([]expressionTemp, expressionReplacement, int, bool) {
	start, end, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, pos, limit)
	if !ok {
		return nil, expressionReplacement{}, 0, false
	}
	key, ok := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
	if !ok {
		return nil, expressionReplacement{}, 0, false
	}
	text, temps, ok := lowerableMapLiteralIndexValueForNormalize(body, toks, mapStart, mapEnd, key, unitName, tempIndex)
	if !ok {
		return nil, expressionReplacement{}, 0, false
	}
	repl := expressionReplacement{
		start: int(toks[start].Start),
		end:   int(toks[indexClose].End),
		text:  text,
	}
	return temps, repl, end - 1, true
}

func lowerableMapLiteralIndexValueForNormalize(body string, toks []scan.Token, start int, end int, key string, unitName string, tempIndex *int) (string, []expressionTemp, bool) {
	text, _, temps, ok := lowerableMapLiteralIndexValueOKForNormalizeFromParts(body, toks, start, end, key, unitName, tempIndex)
	return text, temps, ok
}

func lowerableMapLiteralIndexValueOKForNormalize(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (string, bool, []expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	exprStart, exprEnd, mapStart, mapEnd, indexOpen, indexClose, ok := mapLiteralIndexExpressionPartsForLower(toks, start, end)
	if !ok || exprStart != start || exprEnd != end {
		return "", false, nil, false
	}
	key, ok := mapLiteralComparableKeyForLower(toks, indexOpen+1, indexClose)
	if !ok {
		return "", false, nil, false
	}
	return lowerableMapLiteralIndexValueOKForNormalizeFromParts(body, toks, mapStart, mapEnd, key, unitName, tempIndex)
}

func lowerableMapLiteralIndexValueOKForNormalizeFromParts(body string, toks []scan.Token, start int, end int, key string, unitName string, tempIndex *int) (string, bool, []expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if lowerableMapMakeExpressionForLower(toks, start, end) {
		text, ok := mapLiteralSimpleZeroValueTextForLower(toks, start, end)
		if !ok {
			return "", false, nil, false
		}
		temps, ok := lowerableMapMakeSideEffectTempsForLower(body, toks, start, end, unitName, tempIndex)
		return text, false, temps, ok
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return "", false, nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return "", false, nil, false
	}
	zeroText, zeroOK := mapLiteralSimpleZeroValueTextForLower(toks, start, end)
	if !zeroOK {
		return "", false, nil, false
	}
	values := topLevelExpressionRanges(toks, open+1, close)
	var temps []expressionTemp
	resultText := zeroText
	found := false
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return "", false, nil, false
		}
		valueKey, keyOK := mapLiteralComparableKeyForLower(toks, value.start, colon)
		if !keyOK {
			return "", false, nil, false
		}
		if valueKey == key {
			found = true
			valueText, valueTemps, valueOK := lowerableMapSelectedIndexValueForNormalize(body, toks, colon+1, value.end, unitName, tempIndex)
			if !valueOK {
				return "", false, nil, false
			}
			temps = appendExpressionTemps(temps, valueTemps)
			resultText = valueText
			continue
		}
		valueTemps, valueOK := lowerableMapCompositeValueSideEffectTempsForLower(body, toks, colon+1, value.end, unitName, tempIndex)
		if !valueOK {
			return "", false, nil, false
		}
		temps = appendExpressionTemps(temps, valueTemps)
	}
	return resultText, found, temps, true
}

func lowerableMapSelectedIndexValueForNormalize(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (string, []expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", nil, false
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return strings.TrimSpace(tokenRangeText(body, toks, start, end)), nil, true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" {
		return strings.TrimSpace(tokenRangeText(body, toks, start, end)), nil, true
	}
	if !directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return "", nil, false
	}
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	expr := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
	return name, []expressionTemp{{name: name, expr: expr}}, true
}

func lowerableMapMakeSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableMapMakeSideEffectTempsForLower(body, toks, start+1, close, unitName, tempIndex)
		}
	}
	if !lowerableMapMakeExpressionForLower(toks, start, end) {
		return nil, false
	}
	open := start + 1
	close := findClose(toks, open, "(", ")")
	args := topLevelExpressionRanges(toks, open+1, close)
	if len(args) != 2 || simpleIntegerLiteralKeyForLower(toks, args[1].start, args[1].end) {
		return nil, true
	}
	name := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	expr := strings.TrimSpace(body[int(toks[args[1].start].Start):int(toks[args[1].end-1].End)])
	return []expressionTemp{{name: name, expr: expr}}, true
}

func normalizeMapLiteralLenCall(body string, toks []scan.Token, pos int, close int, unitName string, tempIndex *int) ([]expressionTemp, expressionReplacement, bool) {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Text != "len" || toks[pos+1].Text != "(" {
		return nil, expressionReplacement{}, false
	}
	if pos > 0 && toks[pos-1].Text == "." {
		return nil, expressionReplacement{}, false
	}
	args := topLevelExpressionRanges(toks, pos+2, close)
	if len(args) != 1 {
		return nil, expressionReplacement{}, false
	}
	count, temps, ok := lowerableMapLiteralElementCountForLower(body, toks, args[0].start, args[0].end, unitName, tempIndex)
	if !ok {
		return nil, expressionReplacement{}, false
	}
	repl := expressionReplacement{
		start: int(toks[pos].Start),
		end:   int(toks[close].End),
		text:  strconv.Itoa(count),
	}
	return temps, repl, true
}

func lowerableMapLiteralElementCountForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) (int, []expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return 0, nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableMapLiteralElementCountForLower(body, toks, start+1, close, unitName, tempIndex)
		}
	}
	if lowerableMapMakeExpressionForLower(toks, start, end) {
		temps, ok := lowerableMapMakeSideEffectTempsForLower(body, toks, start, end, unitName, tempIndex)
		return 0, temps, ok
	}
	open := pureMapCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return 0, nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return 0, nil, false
	}
	temps, ok := lowerableMapCompositeSideEffectTempsForLower(body, toks, open+1, close, unitName, tempIndex)
	if !ok {
		return 0, nil, false
	}
	values := topLevelExpressionRanges(toks, open+1, close)
	return len(values), temps, true
}

func lowerableMapCompositeSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var temps []expressionTemp
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon < 0 {
			return nil, false
		}
		if !discardedPureMapCompositeKeyForLower(toks, value.start, colon) {
			return nil, false
		}
		valueTemps, ok := lowerableMapCompositeValueSideEffectTempsForLower(body, toks, colon+1, value.end, unitName, tempIndex)
		if !ok {
			return nil, false
		}
		temps = appendExpressionTemps(temps, valueTemps)
	}
	return temps, true
}

func lowerableMapCompositeValueSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return lowerableMapCompositeSideEffectTempsForLower(body, toks, start+1, close, unitName, tempIndex)
	}
	if toks[start].Text == "map" {
		_, temps, ok := lowerableMapLiteralElementCountForLower(body, toks, start, end, unitName, tempIndex)
		return temps, ok
	}
	if toks[start].Text == "[" {
		return lowerableArrayCompositeSideEffectTempsForLower(body, toks, start, end, unitName, tempIndex)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	if singleIdentifierExpressionInLower(toks, start, end) != "" {
		return nil, true
	}
	if directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		expr := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
		return []expressionTemp{{name: name, expr: expr}}, true
	}
	return nil, false
}

func lowerableArrayCompositeSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return lowerableArrayCompositeSideEffectTempsForLower(body, toks, start+1, close, unitName, tempIndex)
		}
	}
	open := pureArrayCompositeLiteralOpenForLower(toks, start, end)
	if open < 0 {
		return nil, false
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return nil, false
	}
	return lowerableCompositeSideEffectTempsForLower(body, toks, open+1, close, unitName, tempIndex)
}

func lowerableCompositeSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	var temps []expressionTemp
	for i := 0; i < len(values); i++ {
		value := values[i]
		colon := findTopLevelToken(toks, value.start, value.end, ":")
		if colon >= 0 {
			if !discardedPureCompositeKeyForLower(toks, value.start, colon) {
				return nil, false
			}
			value.start = colon + 1
		}
		valueTemps, ok := lowerableCompositeValueSideEffectTempsForLower(body, toks, value.start, value.end, unitName, tempIndex)
		if !ok {
			return nil, false
		}
		temps = appendExpressionTemps(temps, valueTemps)
	}
	return temps, true
}

func lowerableCompositeValueSideEffectTempsForLower(body string, toks []scan.Token, start int, end int, unitName string, tempIndex *int) ([]expressionTemp, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, true
	}
	if toks[start].Text == "{" {
		close := findClose(toks, start, "{", "}")
		if close != end-1 {
			return nil, false
		}
		return lowerableCompositeSideEffectTempsForLower(body, toks, start+1, close, unitName, tempIndex)
	}
	if toks[start].Text == "[" {
		return lowerableArrayCompositeSideEffectTempsForLower(body, toks, start, end, unitName, tempIndex)
	}
	if discardedArrayLiteralElementForLower(toks, start, end) {
		return nil, true
	}
	if directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		expr := strings.TrimSpace(body[int(toks[start].Start):int(toks[end-1].End)])
		return []expressionTemp{{name: name, expr: expr}}, true
	}
	return nil, false
}

func directCallExpressionWithoutNestedCallsForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directCallExpressionWithoutNestedCallsForLower(toks, start+1, close)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	return !expressionContainsCall(toks, start+2, close)
}

func discardedComplexStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return nil, 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(lhs) != len(rhs) {
		return nil, 0, false
	}
	var lines []string
	for i := 0; i < len(lhs); i++ {
		if singleIdentifierExpressionInLower(toks, lhs[i].start, lhs[i].end) != "_" {
			return nil, 0, false
		}
		exprLines, ok := discardedComplexSideEffectStatementsForLower(body, toks, rhs[i].start, rhs[i].end, topNames)
		if !ok {
			return nil, 0, false
		}
		lines = append(lines, exprLines...)
	}
	return lines, stmtEnd, true
}

func complexVarBlankDiscardStatementForLowerAt(body string, toks []scan.Token, pos int, limit int, topNames symbolNameTable) ([]string, int, bool) {
	if pos < 0 || pos+3 >= len(toks) || pos >= limit {
		return nil, 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	name := ""
	eq := -1
	if toks[pos].Text == "var" {
		if pos+4 >= stmtEnd || toks[pos+1].Kind != scan.Ident {
			return nil, 0, false
		}
		name = toks[pos+1].Text
		eq = findTopLevelToken(toks, pos+2, stmtEnd, "=")
		if eq < 0 || lowerCompoundAssignmentEquals(toks, eq) || eq+1 >= stmtEnd {
			return nil, 0, false
		}
		if eq > pos+2 {
			if eq != pos+3 || (toks[pos+2].Text != "complex64" && toks[pos+2].Text != "complex128") {
				return nil, 0, false
			}
		}
	} else {
		if toks[pos].Kind != scan.Ident || pos+2 >= stmtEnd || toks[pos+1].Text != ":=" {
			return nil, 0, false
		}
		name = toks[pos].Text
		eq = pos + 1
	}
	lines, ok := discardedComplexSideEffectStatementsForLower(body, toks, eq+1, stmtEnd, topNames)
	if !ok {
		return nil, 0, false
	}
	discardStart := nextSimpleStatementStartForLower(toks, stmtEnd, limit)
	if discardStart < 0 {
		return nil, 0, false
	}
	discardEnd := lowerSimpleStatementEnd(toks, discardStart, limit)
	if !blankDiscardOfIdentifierStatementForLower(toks, discardStart, discardEnd, name) {
		return nil, 0, false
	}
	if identifierUsedOutsideRangesForLower(toks, name, stmtEnd, limit, discardStart, discardEnd) {
		return nil, 0, false
	}
	return lines, discardEnd, true
}

type complexAliasComponentLowering struct {
	declStart    int
	declEnd      int
	lines        []string
	replacements []expressionReplacement
}

func complexAliasComponentLoweringsForDecl(body string, toks []scan.Token, decl *parse.Decl, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) []complexAliasComponentLowering {
	if decl.Kind != "func" {
		return nil
	}
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return nil
	}
	paramsClose := findClose(toks, namePos+1, "(", ")")
	if paramsClose < 0 {
		return nil
	}
	bodyOpen := functionBodyOpenForDeclAfterParamsForLower(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return nil
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBeforeForLower(toks, decl.End)
	}
	var out []complexAliasComponentLowering
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "var" && (toks[i].Kind != scan.Ident || i+1 >= bodyClose || toks[i+1].Text != ":=") {
			continue
		}
		lowering, ok := complexAliasComponentLoweringAt(body, toks, i, bodyOpen, bodyClose, localNames, topNames, topFunctionNames)
		if !ok {
			continue
		}
		out = append(out, lowering)
		i = lowering.declEnd - 1
	}
	return out
}

func complexAliasComponentLoweringAt(body string, toks []scan.Token, pos int, bodyOpen int, bodyClose int, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) (complexAliasComponentLowering, bool) {
	if pos < 0 || pos+3 >= bodyClose || simpleStatementStartForLower(toks, bodyOpen, pos) != pos {
		return complexAliasComponentLowering{}, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, bodyClose)
	name := ""
	eq := -1
	if toks[pos].Text == "var" {
		if pos+3 >= stmtEnd || toks[pos+1].Kind != scan.Ident || toks[pos+1].Text == "_" {
			return complexAliasComponentLowering{}, false
		}
		name = toks[pos+1].Text
		eq = findTopLevelToken(toks, pos+2, stmtEnd, "=")
		if eq < 0 || lowerCompoundAssignmentEquals(toks, eq) || eq+1 >= stmtEnd {
			return complexAliasComponentLowering{}, false
		}
		if eq > pos+2 {
			if eq != pos+3 || (toks[pos+2].Text != "complex64" && toks[pos+2].Text != "complex128") {
				return complexAliasComponentLowering{}, false
			}
		}
	} else {
		if toks[pos].Kind != scan.Ident || toks[pos].Text == "_" || pos+2 >= stmtEnd || toks[pos+1].Text != ":=" {
			return complexAliasComponentLowering{}, false
		}
		name = toks[pos].Text
		eq = pos + 1
	}
	realExpr, imagExpr, ok := complexAliasExpressionComponentsForLower(body, toks, eq+1, stmtEnd, localNames, topNames, topFunctionNames)
	if !ok {
		return complexAliasComponentLowering{}, false
	}
	realName := "rtg_complex_alias_" + strconv.Itoa(int(toks[pos].Start)) + "_real"
	imagName := "rtg_complex_alias_" + strconv.Itoa(int(toks[pos].Start)) + "_imag"
	replacements, ok := complexAliasComponentUseReplacementsForLower(toks, name, stmtEnd, bodyClose, realName, imagName, localNames, topNames, topFunctionNames)
	if !ok || len(replacements) == 0 {
		return complexAliasComponentLowering{}, false
	}
	if identifierUsedOutsideReplacementRangesForLower(toks, name, bodyOpen+1, bodyClose, pos, stmtEnd, replacements) {
		return complexAliasComponentLowering{}, false
	}
	return complexAliasComponentLowering{
		declStart:    pos,
		declEnd:      stmtEnd,
		lines:        []string{realName + " := " + realExpr, imagName + " := " + imagExpr},
		replacements: replacements,
	}, true
}

func complexAliasExpressionComponentsForLower(body string, toks []scan.Token, start int, end int, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) (string, string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return complexAliasExpressionComponentsForLower(body, toks, start+1, close, localNames, topNames, topFunctionNames)
		}
	}
	realText, imagText, ok := reducibleComplexLiteralPartsForLower(toks, start, end)
	if ok {
		return realText, imagText, true
	}
	if start+3 > end || toks[start].Text != "complex" || toks[start+1].Text != "(" {
		return "", "", false
	}
	if complexComponentBuiltinShadowed(toks, start, localNames, topNames, topFunctionNames) || (start > 0 && toks[start-1].Text == ".") {
		return "", "", false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", "", false
	}
	args := topLevelExpressionRanges(toks, start+2, close)
	if len(args) != 2 {
		return "", "", false
	}
	realText, ok = complexAliasComponentExpressionTextForLower(body, toks, args[0].start, args[0].end, topNames)
	if !ok {
		return "", "", false
	}
	imagText, ok = complexAliasComponentExpressionTextForLower(body, toks, args[1].start, args[1].end, topNames)
	if !ok {
		return "", "", false
	}
	return realText, imagText, true
}

func complexAliasComponentExpressionTextForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) (string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return complexAliasComponentExpressionTextForLower(body, toks, start+1, close, topNames)
		}
	}
	if discardedPureRealNumberExpressionForLower(toks, start, end) {
		return strings.TrimSpace(tokenRangeText(body, toks, start, end)), true
	}
	if directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		expr := strings.TrimSpace(tokenRangeText(body, toks, start, end))
		if unitName := symbolNameTableUnitName(topNames, toks[start].Text); unitName != "" {
			expr = unitName + expr[len(toks[start].Text):]
		}
		return expr, true
	}
	return "", false
}

func complexAliasComponentUseReplacementsForLower(toks []scan.Token, name string, start int, end int, realName string, imagName string, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) ([]expressionReplacement, bool) {
	var replacements []expressionReplacement
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		repl, replEnd, ok := complexAliasComponentUseReplacementForLower(toks, i, realName, imagName, localNames, topNames, topFunctionNames)
		if !ok {
			return nil, false
		}
		replacements = appendExpressionReplacements(replacements, []expressionReplacement{repl})
		i = replEnd - 1
	}
	sortExpressionReplacementsByStart(replacements)
	return replacements, true
}

func complexAliasComponentUseReplacementForLower(toks []scan.Token, namePos int, realName string, imagName string, localNames localNameTable, topNames symbolNameTable, topFunctionNames symbolNameTable) (expressionReplacement, int, bool) {
	if namePos < 2 || namePos+1 >= len(toks) || toks[namePos-1].Text != "(" {
		return expressionReplacement{}, 0, false
	}
	callStart := namePos - 2
	replText := ""
	if toks[callStart].Text == "real" {
		replText = realName
	} else if toks[callStart].Text == "imag" {
		replText = imagName
	} else {
		return expressionReplacement{}, 0, false
	}
	if complexComponentBuiltinShadowed(toks, callStart, localNames, topNames, topFunctionNames) || (callStart > 0 && toks[callStart-1].Text == ".") {
		return expressionReplacement{}, 0, false
	}
	close := findClose(toks, namePos-1, "(", ")")
	if close != namePos+1 {
		return expressionReplacement{}, 0, false
	}
	return expressionReplacement{
		start: int(toks[callStart].Start),
		end:   int(toks[close].End),
		text:  replText,
	}, close + 1, true
}

func complexAliasComponentExpressionReplacements(lowerings []complexAliasComponentLowering) []expressionReplacement {
	var out []expressionReplacement
	for i := 0; i < len(lowerings); i++ {
		out = appendExpressionReplacements(out, lowerings[i].replacements)
	}
	sortExpressionReplacementsByStart(out)
	return out
}

func complexAliasComponentDeclAt(lowerings []complexAliasComponentLowering, pos int) ([]string, int, bool) {
	for i := 0; i < len(lowerings); i++ {
		if lowerings[i].declStart == pos {
			return lowerings[i].lines, lowerings[i].declEnd, true
		}
	}
	return nil, 0, false
}

func interfaceVarBlankDiscardStatementForLowerAt(toks []scan.Token, pos int, limit int) (int, bool) {
	if pos < 0 || pos+3 >= len(toks) || pos >= limit || toks[pos].Text != "var" {
		return 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	if pos+3 > stmtEnd || toks[pos+1].Kind != scan.Ident {
		return 0, false
	}
	name := toks[pos+1].Text
	typeEnd := stmtEnd
	if eq := findTopLevelToken(toks, pos+2, stmtEnd, "="); eq >= 0 {
		if !nilInterfaceInitializerForLower(toks, eq+1, stmtEnd) {
			return 0, false
		}
		typeEnd = eq
	}
	if !interfaceVarTypeRangeForLower(toks, pos+2, typeEnd) {
		return 0, false
	}
	discardStart := nextSimpleStatementStartForLower(toks, stmtEnd, limit)
	if discardStart < 0 {
		return 0, false
	}
	discardEnd := lowerSimpleStatementEnd(toks, discardStart, limit)
	if !blankDiscardOfIdentifierStatementForLower(toks, discardStart, discardEnd, name) {
		return 0, false
	}
	if identifierUsedOutsideRangesForLower(toks, name, stmtEnd, limit, discardStart, discardEnd) {
		return 0, false
	}
	return discardEnd, true
}

type nilInterfaceVarComparisonLowering struct {
	declStart    int
	declEnd      int
	replacements []expressionReplacement
}

func nilInterfaceVarComparisonLoweringsForDecl(toks []scan.Token, decl *parse.Decl) []nilInterfaceVarComparisonLowering {
	if decl.Kind != "func" {
		return nil
	}
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return nil
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return nil
	}
	bodyOpen := functionBodyOpenForDeclAfterParamsForLower(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return nil
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBeforeForLower(toks, decl.End)
	}
	var out []nilInterfaceVarComparisonLowering
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "var" {
			continue
		}
		lowering, ok := nilInterfaceVarComparisonForLowerAt(toks, i, bodyOpen, bodyClose)
		if !ok {
			continue
		}
		out = append(out, lowering)
		i = lowering.declEnd - 1
	}
	return out
}

func nilInterfaceVarComparisonForLowerAt(toks []scan.Token, pos int, bodyOpen int, bodyClose int) (nilInterfaceVarComparisonLowering, bool) {
	if pos < 0 || pos+3 >= len(toks) || pos >= bodyClose || toks[pos].Text != "var" {
		return nilInterfaceVarComparisonLowering{}, false
	}
	if simpleStatementStartForLower(toks, bodyOpen, pos) != pos {
		return nilInterfaceVarComparisonLowering{}, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, bodyClose)
	if pos+3 > stmtEnd || toks[pos+1].Kind != scan.Ident {
		return nilInterfaceVarComparisonLowering{}, false
	}
	name := toks[pos+1].Text
	typeEnd := stmtEnd
	if eq := findTopLevelToken(toks, pos+2, stmtEnd, "="); eq >= 0 {
		if !nilInterfaceInitializerForLower(toks, eq+1, stmtEnd) {
			return nilInterfaceVarComparisonLowering{}, false
		}
		typeEnd = eq
	}
	if !interfaceVarTypeRangeForLower(toks, pos+2, typeEnd) {
		return nilInterfaceVarComparisonLowering{}, false
	}
	comparisons := nilInterfaceComparisonReplacementsForLower(toks, name, stmtEnd, bodyClose)
	if len(comparisons) == 0 {
		return nilInterfaceVarComparisonLowering{}, false
	}
	if identifierUsedOutsideReplacementRangesForLower(toks, name, bodyOpen+1, bodyClose, pos, stmtEnd, comparisons) {
		return nilInterfaceVarComparisonLowering{}, false
	}
	return nilInterfaceVarComparisonLowering{declStart: pos, declEnd: stmtEnd, replacements: comparisons}, true
}

func nilInterfaceComparisonReplacementsForLower(toks []scan.Token, name string, start int, end int) []expressionReplacement {
	var out []expressionReplacement
	for i := start + 1; i+1 < end; i++ {
		if toks[i].Text != "==" && toks[i].Text != "!=" {
			continue
		}
		left := nilInterfaceComparisonOperandForLower(toks, i-1, start, end)
		right := nilInterfaceComparisonOperandForLower(toks, i+1, start, end)
		if !nilInterfaceComparisonNamesMatchForLower(left, right, name) {
			continue
		}
		if len(out) > 0 && int(toks[i-1].Start) < out[len(out)-1].end {
			continue
		}
		text := "true"
		if toks[i].Text == "!=" {
			text = "false"
		}
		out = append(out, expressionReplacement{start: int(toks[i-1].Start), end: int(toks[i+1].End), text: text})
	}
	return out
}

func nilInterfaceComparisonOperandForLower(toks []scan.Token, pos int, start int, end int) string {
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

func nilInterfaceComparisonNamesMatchForLower(left string, right string, name string) bool {
	return (left == name && right == "nil") || (left == "nil" && right == name)
}

func identifierUsedOutsideReplacementRangesForLower(toks []scan.Token, name string, scopeStart int, scopeEnd int, firstStart int, firstEnd int, replacements []expressionReplacement) bool {
	for i := scopeStart; i < len(toks) && i < scopeEnd; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		if i >= firstStart && i < firstEnd {
			continue
		}
		pos := int(toks[i].Start)
		if sourcePositionInExpressionReplacements(pos, replacements) {
			continue
		}
		return true
	}
	return false
}

func sourcePositionInExpressionReplacements(pos int, replacements []expressionReplacement) bool {
	for i := 0; i < len(replacements); i++ {
		if pos >= replacements[i].start && pos < replacements[i].end {
			return true
		}
	}
	return false
}

func nilInterfaceComparisonExpressionReplacements(lowerings []nilInterfaceVarComparisonLowering) []expressionReplacement {
	var out []expressionReplacement
	for i := 0; i < len(lowerings); i++ {
		out = appendExpressionReplacements(out, lowerings[i].replacements)
	}
	sortExpressionReplacementsByStart(out)
	return out
}

func nilInterfaceVarComparisonDeclEndAt(lowerings []nilInterfaceVarComparisonLowering, pos int) (int, bool) {
	for i := 0; i < len(lowerings); i++ {
		if lowerings[i].declStart == pos {
			return lowerings[i].declEnd, true
		}
	}
	return 0, false
}

func expressionReplacementAtStart(replacements []expressionReplacement, start int) (expressionReplacement, bool) {
	for i := 0; i < len(replacements); i++ {
		if replacements[i].start == start {
			return replacements[i], true
		}
	}
	return expressionReplacement{}, false
}

func nilInterfaceInitializerForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
	return start+1 == end && toks[start].Kind == scan.Ident && toks[start].Text == "nil"
}

func interfaceParamErasuresForLoadFiles(files []load.File, topNames symbolNameTable) interfaceParamEraseTable {
	var out interfaceParamEraseTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			info, ok := interfaceParamEraseForDecl(&parsed, parsed.Decls[declIndex], topNames)
			if !ok {
				continue
			}
			if !interfaceParamEraseCallSitesLowerableForLoadFiles(files, info.name, info.indexes, topNames) {
				continue
			}
			out = append(out, info)
		}
	}
	return out
}

func interfaceParamEraseForDecl(file *parse.File, decl parse.Decl, topNames symbolNameTable) (interfaceParamEraseInfo, bool) {
	if decl.Kind != "func" || decl.Receiver || decl.Name == "" || isExported(decl.Name) {
		return interfaceParamEraseInfo{}, false
	}
	unitName := symbolNameTableUnitName(topNames, decl.Name)
	if unitName == "" {
		return interfaceParamEraseInfo{}, false
	}
	toks := file.Tokens
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return interfaceParamEraseInfo{}, false
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return interfaceParamEraseInfo{}, false
	}
	bodyOpen := functionBodyOpenForDeclAfterParamsForLower(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return interfaceParamEraseInfo{}, false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBeforeForLower(toks, decl.End)
	}
	indexes := unusedInterfaceParamIndexesForLower(toks, paramsOpen+1, paramsClose, bodyOpen, bodyClose)
	if len(indexes) == 0 {
		return interfaceParamEraseInfo{}, false
	}
	return interfaceParamEraseInfo{name: decl.Name, unitName: unitName, indexes: indexes}, true
}

func functionBodyOpenForDeclAfterParamsForLower(toks []scan.Token, paramsClose int, declEnd int) int {
	bodyClose := tokenIndexBeforeForLower(toks, declEnd)
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

func unusedInterfaceParamIndexesForLower(toks []scan.Token, start int, end int, bodyOpen int, bodyClose int) []int {
	var out []int
	paramIndex := 0
	var pendingNames []string
	segments := topLevelExpressionRanges(toks, start, end)
	for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
		segment := segments[segmentIndex]
		if name, ok := bareParameterNameSegmentForLower(toks, segment.start, segment.end); ok {
			pendingNames = append(pendingNames, name)
			continue
		}
		names, _, _, ok := interfaceParamSegmentForLower(toks, segment.start, segment.end)
		count := parameterSegmentCountForLower(toks, segment.start, segment.end)
		if ok && len(names) == count {
			names = append(append([]string{}, pendingNames...), names...)
			count = len(names)
			used := false
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := names[nameIndex]
				if name != "_" && identifierUsedInTokenRangeForLower(toks, name, bodyOpen+1, bodyClose) {
					used = true
					break
				}
			}
			if !used {
				for nameIndex := 0; nameIndex < len(names); nameIndex++ {
					out = append(out, paramIndex+nameIndex)
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

func bareParameterNameSegmentForLower(toks []scan.Token, start int, end int) (string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text, true
	}
	return "", false
}

func interfaceParamSegmentForLower(toks []scan.Token, start int, end int) ([]string, int, int, bool) {
	start, end = trimTokenRange(toks, start, end)
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

func parameterSegmentCountForLower(toks []scan.Token, start int, end int) int {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return 0
	}
	if toks[start].Kind != scan.Ident {
		return 1
	}
	if start+1 < end && toks[start+1].Text == "," {
		count := 0
		for i := start; i < end; i++ {
			if toks[i].Kind == scan.Ident {
				count++
			}
			if i+1 < end && toks[i+1].Text != "," {
				break
			}
		}
		if count > 0 {
			return count
		}
	}
	return 1
}

func identifierUsedInTokenRangeForLower(toks []scan.Token, name string, start int, end int) bool {
	for i := start; i < len(toks) && i < end; i++ {
		if toks[i].Kind == scan.Ident && toks[i].Text == name {
			return true
		}
	}
	return false
}

func interfaceParamEraseCallSitesLowerableForLoadFiles(files []load.File, name string, indexes []int, topNames symbolNameTable) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			return false
		}
		if !interfaceParamEraseCallSitesLowerableForLower(parsed.Tokens, name, indexes, topNames) {
			return false
		}
	}
	return true
}

func interfaceParamEraseCallSitesLowerableForLower(toks []scan.Token, name string, indexes []int, topNames symbolNameTable) bool {
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
		args := topLevelExpressionRanges(toks, i+2, close)
		if !interfaceParamEraseCallSiteLowerableForLower(toks, i, close, args, indexes, topNames) {
			return false
		}
		i = close
	}
	return true
}

func interfaceParamEraseCallSiteLowerableForLower(toks []scan.Token, pos int, close int, args []expressionRange, indexes []int, topNames symbolNameTable) bool {
	lastSideEffectErasedArg := -1
	for indexIndex := 0; indexIndex < len(indexes); indexIndex++ {
		index := indexes[indexIndex]
		if index < 0 || index >= len(args) {
			return false
		}
		arg := args[index]
		if !expressionContainsCall(toks, arg.start, arg.end) {
			continue
		}
		if !interfaceParamEraseDirectCallStatementSiteForLower(toks, pos, close) && !interfaceParamEraseDeferCallStatementSiteForLower(toks, pos, close) && !interfaceParamEraseReturnExpressionSiteForLower(toks, pos, close) && !interfaceParamEraseAssignmentExpressionSiteForLower(toks, pos, close) && !interfaceParamEraseVarInitializerSiteForLower(toks, pos, close) && !interfaceParamEraseIfConditionSiteForLower(toks, pos, close) && !interfaceParamEraseForConditionSiteForLower(toks, pos, close) && !interfaceParamEraseClassicForConditionSiteForLower(toks, pos, close) && !interfaceParamEraseSwitchTagSiteForLower(toks, pos, close) {
			return false
		}
		if !interfaceReturnSideEffectExpressionLowerableForLower(toks, arg.start, arg.end, topNames) {
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
		if interfaceParamEraseHasRawIndex(indexes, argIndex) {
			continue
		}
		arg := args[argIndex]
		if expressionContainsCall(toks, arg.start, arg.end) {
			return false
		}
	}
	return true
}

func interfaceParamEraseDirectCallStatementSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	return stmtEnd == close+1
}

func interfaceParamEraseDeferCallStatementSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart+1 != pos || toks[stmtStart].Text != "defer" {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	return stmtEnd == close+1
}

func interfaceParamEraseReturnExpressionSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart < 0 || toks[stmtStart].Text != "return" {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	values := topLevelExpressionRanges(toks, stmtStart+1, stmtEnd)
	if len(values) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, values[0].start, values[0].end)
	return start == pos && end == close+1
}

func interfaceParamEraseAssignmentExpressionSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) == 0 || len(rhs) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func interfaceParamEraseVarInitializerSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	if stmtStart < 0 || toks[stmtStart].Text != "var" {
		return false
	}
	if stmtStart+1 < len(toks) && toks[stmtStart+1].Text == "(" {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(rhs) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func interfaceParamEraseIfConditionSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "if" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionIfConditionRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func interfaceParamEraseForConditionSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "for" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionForConditionRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func interfaceParamEraseClassicForConditionSiteForLower(toks []scan.Token, pos int, close int) bool {
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
	open := conditionExpressionEnd(toks, forPos)
	if open >= len(toks) || toks[open].Text != "{" {
		return false
	}
	secondSemi := topLevelSemicolon(toks, pos, open)
	if secondSemi != close+1 {
		return false
	}
	start, end := trimTokenRange(toks, pos, secondSemi)
	return start == pos && end == close+1
}

func interfaceParamEraseSwitchTagSiteForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "switch" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionSwitchTagRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func directKnownFunctionCallExpressionWithoutNestedCallsForLower(toks []scan.Token, start int, end int, topNames symbolNameTable) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directKnownFunctionCallExpressionWithoutNestedCallsForLower(toks, start+1, close, topNames)
		}
	}
	if !directCallExpressionWithoutNestedCallsForLower(toks, start, end) {
		return false
	}
	return symbolNameTableUnitName(topNames, toks[start].Text) != ""
}

func interfaceParamEraseHasRawIndex(indexes []int, index int) bool {
	for i := 0; i < len(indexes); i++ {
		if indexes[i] == index {
			return true
		}
	}
	return false
}

func interfaceParamEraseByName(table interfaceParamEraseTable, name string) (interfaceParamEraseInfo, bool) {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			return table[i], true
		}
	}
	return interfaceParamEraseInfo{}, false
}

func interfaceParamEraseHasIndex(info interfaceParamEraseInfo, index int) bool {
	for i := 0; i < len(info.indexes); i++ {
		if info.indexes[i] == index {
			return true
		}
	}
	return false
}

func interfaceParamErasedParameterListForLower(source []byte, toks []scan.Token, paramsOpen int, paramsClose int, info interfaceParamEraseInfo) string {
	segments := topLevelExpressionRanges(toks, paramsOpen+1, paramsClose)
	var kept []string
	paramIndex := 0
	var pending []string
	pendingCount := 0
	for segmentIndex := 0; segmentIndex < len(segments); segmentIndex++ {
		segment := segments[segmentIndex]
		segmentText := string(source[int(toks[segment.start].Start):int(toks[segment.end-1].End)])
		if _, ok := bareParameterNameSegmentForLower(toks, segment.start, segment.end); ok {
			pending = append(pending, segmentText)
			pendingCount++
			continue
		}
		count := parameterSegmentCountForLower(toks, segment.start, segment.end)
		groupCount := pendingCount + count
		erase := false
		for i := 0; i < groupCount; i++ {
			if interfaceParamEraseHasIndex(info, paramIndex+i) {
				erase = true
				break
			}
		}
		if !erase {
			if len(pending) > 0 {
				pending = append(pending, segmentText)
				kept = append(kept, strings.Join(pending, ", "))
			} else {
				kept = append(kept, segmentText)
			}
		}
		paramIndex += groupCount
		pending = nil
		pendingCount = 0
	}
	if len(pending) > 0 {
		kept = append(kept, strings.Join(pending, ", "))
	}
	return "(" + strings.Join(kept, ", ") + ")"
}

func interfaceParamErasedCallReplacement(source []byte, toks []scan.Token, pos int, info interfaceParamEraseInfo, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, int, bool) {
	sideEffects, call, cursorEnd, close, ok := interfaceParamErasedCallPartsForLower(source, toks, pos, info, topNames, tempPrefix, tempIndex)
	if !ok {
		return "", 0, 0, false
	}
	if len(sideEffects) == 0 {
		return call, cursorEnd, close, true
	}
	if !interfaceParamEraseDirectCallStatementSiteForLower(toks, pos, close) {
		return "", 0, 0, false
	}
	sideEffects = append(sideEffects, call)
	indent := statementIndent(string(source), int(toks[pos].Start))
	return strings.Join(sideEffects, "\n"+indent), cursorEnd, close, true
}

func interfaceParamErasedCallPartsForLower(source []byte, toks []scan.Token, pos int, info interfaceParamEraseInfo, topNames symbolNameTable, tempPrefix string, tempIndex *int) ([]string, string, int, int, bool) {
	if pos < 0 || pos+1 >= len(toks) || toks[pos].Kind != scan.Ident || toks[pos+1].Text != "(" {
		return nil, "", 0, 0, false
	}
	open := pos + 1
	close := findClose(toks, open, "(", ")")
	if close < 0 {
		return nil, "", 0, 0, false
	}
	args := topLevelExpressionRanges(toks, open+1, close)
	var kept []string
	var sideEffects []string
	for argIndex := 0; argIndex < len(args); argIndex++ {
		arg := args[argIndex]
		if interfaceParamEraseHasIndex(info, argIndex) {
			if expressionContainsCall(toks, arg.start, arg.end) {
				lines, ok := discardedInterfaceReturnSideEffectLinesForLower(string(source), toks, arg.start, arg.end, topNames, tempPrefix, tempIndex)
				if !ok {
					return nil, "", 0, 0, false
				}
				sideEffects = append(sideEffects, lines...)
			}
			continue
		}
		kept = append(kept, string(source[int(toks[arg.start].Start):int(toks[arg.end-1].End)]))
	}
	call := info.unitName + "(" + strings.Join(kept, ", ") + ")"
	return sideEffects, call, int(toks[close].End), close, true
}

func interfaceParamErasedReturnStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "return" {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	values := topLevelExpressionRanges(toks, pos+1, stmtEnd)
	if len(values) != 1 {
		return "", 0, false
	}
	start, end := trimTokenRange(toks, values[0].start, values[0].end)
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, false
	}
	lines := append([]string{}, sideEffects...)
	lines = append(lines, "return "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), stmtEnd, true
}

func interfaceParamErasedAssignmentStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) {
		return "", 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, pos, stmtEnd, "=")
	}
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return "", 0, false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(rhs) != 1 {
		return "", 0, false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, false
	}
	prefix := strings.TrimSpace(string(source[int(toks[pos].Start):int(toks[assign].End)]))
	lines := append([]string{}, sideEffects...)
	lines = append(lines, prefix+" "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), stmtEnd, true
}

func interfaceParamErasedVarStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "var" {
		return "", 0, false
	}
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		return "", 0, false
	}
	stmtStart := simpleStatementStartForLower(toks, -1, pos)
	if stmtStart != pos {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return "", 0, false
	}
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(rhs) != 1 {
		return "", 0, false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, false
	}
	prefix := strings.TrimSpace(string(source[int(toks[pos].Start):int(toks[assign].End)]))
	lines := append([]string{}, sideEffects...)
	lines = append(lines, prefix+" "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), stmtEnd, true
}

func interfaceParamErasedDeferStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, bool) {
	if pos < 0 || pos >= limit || pos+2 >= len(toks) || toks[pos].Text != "defer" {
		return "", 0, false
	}
	start := pos + 1
	if toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close < 0 {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	if stmtEnd != close+1 {
		return "", 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, false
	}
	lines := append([]string{}, sideEffects...)
	lines = append(lines, "defer "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), stmtEnd, true
}

func interfaceParamErasedIfConditionForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "if" {
		return "", 0, 0, false
	}
	start, end, open, ok := staticInterfaceAssertionIfConditionRangeForLower(toks, pos)
	if !ok || open >= limit {
		return "", 0, 0, false
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, 0, false
	}
	lines := append([]string{}, sideEffects...)
	lines = append(lines, "if "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), int(toks[open].Start), open - 1, true
}

func interfaceParamErasedForConditionForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "for" {
		return "", 0, 0, false
	}
	start, end, open, ok := staticInterfaceAssertionForConditionRangeForLower(toks, pos)
	if !ok || open >= limit {
		return "", 0, 0, false
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, 0, false
	}
	lines := []string{"for {"}
	for i := 0; i < len(sideEffects); i++ {
		lines = append(lines, "\t"+sideEffects[i])
	}
	lines = append(lines, "\tif !("+call+") {")
	lines = append(lines, "\t\tbreak")
	lines = append(lines, "\t}")
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), int(toks[open].End), open, true
}

func interfaceParamErasedClassicForConditionForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "for" {
		return "", 0, 0, false
	}
	open := conditionExpressionEnd(toks, pos)
	if open >= limit || toks[open].Text != "{" {
		return "", 0, 0, false
	}
	firstSemi := topLevelSemicolon(toks, pos+1, open)
	if firstSemi < 0 {
		return "", 0, 0, false
	}
	secondSemi := topLevelSemicolon(toks, firstSemi+1, open)
	if secondSemi < 0 {
		return "", 0, 0, false
	}
	start, end := trimTokenRange(toks, firstSemi+1, secondSemi)
	if toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, 0, false
	}
	initStart, initEnd := trimTokenRange(toks, pos+1, firstSemi)
	postStart, postEnd := trimTokenRange(toks, secondSemi+1, open)
	header := string(source[int(toks[pos].Start):int(toks[start].Start)]) + string(source[int(toks[secondSemi].Start):int(toks[open].End)])
	if initStart == initEnd && postStart == postEnd {
		header = "for {"
	}
	lines := []string{header}
	for i := 0; i < len(sideEffects); i++ {
		lines = append(lines, "\t"+sideEffects[i])
	}
	lines = append(lines, "\tif !("+call+") {")
	lines = append(lines, "\t\tbreak")
	lines = append(lines, "\t}")
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), int(toks[open].End), open, true
}

func interfaceParamErasedSwitchTagForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceParamEraseTable, topNames symbolNameTable, tempPrefix string, tempIndex *int) (string, int, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "switch" {
		return "", 0, 0, false
	}
	start, end, open, ok := staticInterfaceAssertionSwitchTagRangeForLower(toks, pos)
	if !ok || open >= limit {
		return "", 0, 0, false
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", 0, 0, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", 0, 0, false
	}
	info, ok := interfaceParamEraseByName(table, toks[start].Text)
	if !ok {
		return "", 0, 0, false
	}
	sideEffects, call, _, _, ok := interfaceParamErasedCallPartsForLower(source, toks, start, info, topNames, tempPrefix, tempIndex)
	if !ok || len(sideEffects) == 0 {
		return "", 0, 0, false
	}
	lines := append([]string{}, sideEffects...)
	lines = append(lines, "switch "+call)
	return joinIndentedLines(lines, statementIndent(string(source), int(toks[pos].Start))), int(toks[open].Start), open - 1, true
}

func interfaceReturnErasuresForLoadFiles(files []load.File, topNames symbolNameTable) interfaceReturnEraseTable {
	var out interfaceReturnEraseTable
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			continue
		}
		for declIndex := 0; declIndex < len(parsed.Decls); declIndex++ {
			info, ok := interfaceReturnEraseForDecl(&parsed, parsed.Decls[declIndex], topNames)
			if !ok {
				continue
			}
			if !interfaceReturnCallSitesLowerableForLoadFiles(files, info.name) {
				continue
			}
			out = append(out, info)
		}
	}
	return out
}

func interfaceReturnEraseForDecl(file *parse.File, decl parse.Decl, topNames symbolNameTable) (interfaceReturnEraseInfo, bool) {
	if decl.Kind != "func" || decl.Receiver || decl.Name == "" || isExported(decl.Name) {
		return interfaceReturnEraseInfo{}, false
	}
	unitName := symbolNameTableUnitName(topNames, decl.Name)
	if unitName == "" {
		return interfaceReturnEraseInfo{}, false
	}
	toks := file.Tokens
	namePos := tokenIndexAt(toks, int(decl.NameTok.Start))
	if namePos < 0 || namePos+1 >= len(toks) || toks[namePos+1].Text != "(" {
		return interfaceReturnEraseInfo{}, false
	}
	paramsOpen := namePos + 1
	paramsClose := findClose(toks, paramsOpen, "(", ")")
	if paramsClose < 0 {
		return interfaceReturnEraseInfo{}, false
	}
	if interfaceReturnParamsContainInterfaceForLower(toks, paramsOpen+1, paramsClose) {
		return interfaceReturnEraseInfo{}, false
	}
	bodyOpen := functionBodyOpenForDeclAfterParamsForLower(toks, paramsClose, decl.End)
	if bodyOpen < 0 {
		return interfaceReturnEraseInfo{}, false
	}
	if _, _, ok := interfaceReturnResultRangeForLower(toks, paramsClose+1, bodyOpen); !ok {
		return interfaceReturnEraseInfo{}, false
	}
	bodyClose := findClose(toks, bodyOpen, "{", "}")
	if bodyClose < 0 {
		bodyClose = tokenIndexBeforeForLower(toks, decl.End)
	}
	if !interfaceReturnExpressionsLowerableForLower(toks, bodyOpen, bodyClose, topNames) {
		return interfaceReturnEraseInfo{}, false
	}
	return interfaceReturnEraseInfo{name: decl.Name, unitName: unitName}, true
}

func interfaceReturnParamsContainInterfaceForLower(toks []scan.Token, start int, end int) bool {
	for i := start; i < end; i++ {
		if toks[i].Text == "interface" || (toks[i].Kind == scan.Ident && toks[i].Text == "any") {
			return true
		}
	}
	return false
}

func interfaceReturnResultRangeForLower(toks []scan.Token, start int, end int) (int, int, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return 0, 0, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return interfaceReturnResultRangeForLower(toks, start+1, close)
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

func interfaceReturnExpressionsLowerableForLower(toks []scan.Token, bodyOpen int, bodyClose int, topNames symbolNameTable) bool {
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text == "func" {
			return false
		}
		if toks[i].Text != "return" {
			continue
		}
		stmtEnd := lowerSimpleStatementEnd(toks, i, bodyClose)
		values := topLevelExpressionRanges(toks, i+1, stmtEnd)
		if len(values) != 1 {
			return false
		}
		value := values[0]
		if expressionContainsCall(toks, value.start, value.end) {
			return interfaceReturnSideEffectExpressionLowerableForLower(toks, value.start, value.end, topNames)
		}
		i = stmtEnd - 1
	}
	return true
}

func directKnownFunctionCallExpressionWithDirectCallArgsForLower(toks []scan.Token, start int, end int, topNames symbolNameTable) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return directKnownFunctionCallExpressionWithDirectCallArgsForLower(toks, start+1, close, topNames)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return false
	}
	if symbolNameTableUnitName(topNames, toks[start].Text) == "" {
		return false
	}
	args := topLevelExpressionRanges(toks, start+2, close)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if expressionContainsCall(toks, arg.start, arg.end) && !interfaceReturnSideEffectExpressionLowerableForLower(toks, arg.start, arg.end, topNames) {
			return false
		}
	}
	return true
}

func interfaceReturnSideEffectExpressionLowerableForLower(toks []scan.Token, start int, end int, topNames symbolNameTable) bool {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return false
	}
	if !expressionContainsCall(toks, start, end) {
		return true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return interfaceReturnSideEffectExpressionLowerableForLower(toks, start+1, close, topNames)
		}
	}
	if op := topLevelBinaryOperatorForLower(toks, start, end); op >= 0 {
		return interfaceReturnSideEffectExpressionLowerableForLower(toks, start, op, topNames) && interfaceReturnSideEffectExpressionLowerableForLower(toks, op+1, end, topNames)
	}
	if op := unaryOperatorTextForLower(toks[start].Text); op != "" && start+1 < end {
		return interfaceReturnSideEffectExpressionLowerableForLower(toks, start+1, end, topNames)
	}
	return directKnownFunctionCallExpressionWithDirectCallArgsForLower(toks, start, end, topNames)
}

func topLevelBinaryOperatorForLower(toks []scan.Token, start int, end int) int {
	best := -1
	bestPrec := 100
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < end; i++ {
		if paren == 0 && brack == 0 && brace == 0 && i > start && i+1 < end {
			prec := binaryOperatorPrecedenceForLower(toks[i].Text)
			if prec > 0 && prec <= bestPrec {
				best = i
				bestPrec = prec
			}
		}
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return best
}

func binaryOperatorPrecedenceForLower(op string) int {
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

func unaryOperatorTextForLower(text string) string {
	switch text {
	case "+", "-", "!", "^":
		return text
	}
	return ""
}

func discardedInterfaceReturnSideEffectLinesForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable, unitName string, tempIndex *int) ([]string, bool) {
	if directKnownFunctionCallExpressionWithDirectCallArgsForLower(toks, start, end, topNames) {
		lines, call, ok := interfaceReturnSideEffectCallExpressionForLower(body, toks, start, end, topNames, unitName, tempIndex)
		if !ok {
			return nil, false
		}
		lines = append(lines, call)
		return lines, true
	}
	lines, _, ok := interfaceReturnSideEffectExpressionForLower(body, toks, start, end, topNames, unitName, tempIndex)
	if !ok {
		return nil, false
	}
	return lines, true
}

func interfaceReturnSideEffectCallExpressionForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable, unitName string, tempIndex *int) ([]string, string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, "", false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return interfaceReturnSideEffectCallExpressionForLower(body, toks, start+1, close, topNames, unitName, tempIndex)
		}
	}
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return nil, "", false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return nil, "", false
	}
	callName := symbolNameTableUnitName(topNames, toks[start].Text)
	if callName == "" {
		return nil, "", false
	}
	args := topLevelExpressionRanges(toks, start+2, close)
	var lines []string
	var argText []string
	for argIndex := 0; argIndex < len(args); argIndex++ {
		arg := args[argIndex]
		if expressionContainsCall(toks, arg.start, arg.end) {
			argLines, argExpr, ok := interfaceReturnSideEffectExpressionForLower(body, toks, arg.start, arg.end, topNames, unitName, tempIndex)
			if !ok {
				return nil, "", false
			}
			lines = append(lines, argLines...)
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			lines = append(lines, name+" := "+argExpr)
			argText = append(argText, name)
			continue
		}
		argText = append(argText, string(body[int(toks[arg.start].Start):int(toks[arg.end-1].End)]))
	}
	return lines, callName + "(" + strings.Join(argText, ", ") + ")", true
}

func interfaceReturnSideEffectExpressionForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable, unitName string, tempIndex *int) ([]string, string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, "", false
	}
	if !expressionContainsCall(toks, start, end) {
		return nil, string(body[int(toks[start].Start):int(toks[end-1].End)]), true
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			lines, expr, ok := interfaceReturnSideEffectExpressionForLower(body, toks, start+1, close, topNames, unitName, tempIndex)
			if !ok {
				return nil, "", false
			}
			return lines, "(" + expr + ")", true
		}
	}
	if op := topLevelBinaryOperatorForLower(toks, start, end); op >= 0 {
		if toks[op].Text == "&&" || toks[op].Text == "||" {
			return interfaceReturnShortCircuitExpressionForLower(body, toks, start, op, end, topNames, unitName, tempIndex)
		}
		leftLines, leftExpr, leftOK := interfaceReturnSideEffectExpressionForLower(body, toks, start, op, topNames, unitName, tempIndex)
		if !leftOK {
			return nil, "", false
		}
		lines := append([]string{}, leftLines...)
		if expressionContainsCall(toks, start, op) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			lines = append(lines, name+" := "+leftExpr)
			leftExpr = name
		}
		rightLines, rightExpr, rightOK := interfaceReturnSideEffectExpressionForLower(body, toks, op+1, end, topNames, unitName, tempIndex)
		if !rightOK {
			return nil, "", false
		}
		lines = append(lines, rightLines...)
		if expressionContainsCall(toks, op+1, end) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			lines = append(lines, name+" := "+rightExpr)
			rightExpr = name
		}
		return lines, leftExpr + " " + toks[op].Text + " " + rightExpr, true
	}
	if op := unaryOperatorTextForLower(toks[start].Text); op != "" && start+1 < end {
		lines, expr, ok := interfaceReturnSideEffectExpressionForLower(body, toks, start+1, end, topNames, unitName, tempIndex)
		if !ok {
			return nil, "", false
		}
		if expressionContainsCall(toks, start+1, end) {
			name := nextExpressionTempName(body, unitName, tempIndex)
			(*tempIndex)++
			lines = append(lines, name+" := "+expr)
			expr = name
		}
		return lines, op + expr, true
	}
	return interfaceReturnSideEffectCallExpressionForLower(body, toks, start, end, topNames, unitName, tempIndex)
}

func interfaceReturnShortCircuitExpressionForLower(body string, toks []scan.Token, start int, op int, end int, topNames symbolNameTable, unitName string, tempIndex *int) ([]string, string, bool) {
	leftLines, leftExpr, leftOK := interfaceReturnSideEffectExpressionForLower(body, toks, start, op, topNames, unitName, tempIndex)
	if !leftOK {
		return nil, "", false
	}
	lines := append([]string{}, leftLines...)
	if expressionContainsCall(toks, start, op) {
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		lines = append(lines, name+" := "+leftExpr)
		leftExpr = name
	}
	resultName := nextExpressionTempName(body, unitName, tempIndex)
	(*tempIndex)++
	lines = append(lines, resultName+" := "+leftExpr)
	condition := resultName
	if toks[op].Text == "||" {
		condition = "!" + resultName
	}
	lines = append(lines, "if "+condition+" {")
	rightLines, rightExpr, rightOK := interfaceReturnSideEffectExpressionForLower(body, toks, op+1, end, topNames, unitName, tempIndex)
	if !rightOK {
		return nil, "", false
	}
	for i := 0; i < len(rightLines); i++ {
		lines = append(lines, "\t"+rightLines[i])
	}
	if expressionContainsCall(toks, op+1, end) {
		name := nextExpressionTempName(body, unitName, tempIndex)
		(*tempIndex)++
		lines = append(lines, "\t"+name+" := "+rightExpr)
		rightExpr = name
	}
	lines = append(lines, "\t"+resultName+" = "+rightExpr)
	lines = append(lines, "}")
	return lines, resultName, true
}

func interfaceReturnCallSitesLowerableForLoadFiles(files []load.File, name string) bool {
	for fileIndex := 0; fileIndex < len(files); fileIndex++ {
		parsed, err := parsedLoadFile(files[fileIndex])
		if err != nil {
			return false
		}
		if !interfaceReturnCallSitesLowerableForLower(parsed.Tokens, name) {
			return false
		}
	}
	return true
}

func interfaceReturnCallSitesLowerableForLower(toks []scan.Token, name string) bool {
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
		if !interfaceReturnCallSiteLowerableForLower(toks, i, close) {
			return false
		}
		i = close
	}
	return true
}

func interfaceReturnCallSiteLowerableForLower(toks []scan.Token, pos int, close int) bool {
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	if stmtStart == pos && stmtEnd == close+1 {
		return true
	}
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return false
	}
	if singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end) != "_" {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func interfaceReturnEraseByName(table interfaceReturnEraseTable, name string) (interfaceReturnEraseInfo, bool) {
	for i := 0; i < len(table); i++ {
		if table[i].name == name {
			return table[i], true
		}
	}
	return interfaceReturnEraseInfo{}, false
}

func interfaceReturnDiscardStatementForLowerAt(source []byte, toks []scan.Token, pos int, limit int, table interfaceReturnEraseTable) (string, int, bool) {
	if pos < 0 || pos >= limit || pos >= len(toks) || toks[pos].Text != "_" {
		return "", 0, false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, pos, limit)
	assign := findTopLevelToken(toks, pos, stmtEnd, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return "", 0, false
	}
	lhs := topLevelExpressionRanges(toks, pos, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return "", 0, false
	}
	if singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end) != "_" {
		return "", 0, false
	}
	replacement, ok := interfaceReturnCallReplacementForLower(source, toks, rhs[0].start, rhs[0].end, table)
	if !ok {
		return "", 0, false
	}
	return replacement, stmtEnd, true
}

func interfaceReturnCallReplacementForLower(source []byte, toks []scan.Token, start int, end int, table interfaceReturnEraseTable) (string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start+2 > end || toks[start].Kind != scan.Ident || toks[start+1].Text != "(" {
		return "", false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return "", false
	}
	info, ok := interfaceReturnEraseByName(table, toks[start].Text)
	if !ok {
		return "", false
	}
	args := ""
	if start+2 < close {
		args = string(source[int(toks[start+2].Start):int(toks[close-1].End)])
	}
	return info.unitName + "(" + args + ")", true
}

type staticInterfaceAssertionVarForLower struct {
	name         string
	concreteType string
	nilInterface bool
	declStart    int
	declEnd      int
	initStart    int
	initEnd      int
	bodyOpen     int
	bodyClose    int
}

type staticInterfaceTypeSwitchGuardForLower struct {
	sourceName  string
	bindingName string
	sourcePos   int
	dot         int
	open        int
}

func lowerStaticInterfaceAssertions(body string, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) string {
	toks, err := scan.Tokens([]byte(body))
	if err != nil {
		return body
	}
	bodyOpen, bodyClose, ok := functionBodyRange(toks)
	if !ok {
		return body
	}
	var vars []staticInterfaceAssertionVarForLower
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "var" {
			continue
		}
		info, ok := staticInterfaceAssertionVarAtForLower(toks, i, bodyOpen, bodyClose, topNames, importRefs, fieldTypes)
		if !ok {
			continue
		}
		vars = append(vars, info)
		i = info.declEnd - 1
	}
	if len(vars) == 0 {
		return body
	}
	var replacements []expressionReplacement
	for i := 0; i < len(vars); i++ {
		info := vars[i]
		text := ""
		if !info.nilInterface {
			initText := strings.TrimSpace(tokenRangeText(body, toks, info.initStart, info.initEnd))
			text = "var " + info.name + " " + info.concreteType + " = " + initText
		}
		replacements = append(replacements, expressionReplacement{
			start: int(toks[info.declStart].Start),
			end:   int(toks[info.declEnd-1].End),
			text:  text,
		})
	}
	for i := bodyOpen + 1; i < bodyClose; i++ {
		if toks[i].Text != "switch" {
			continue
		}
		repl, close, ok := staticInterfaceTypeSwitchReplacementForLower(body, toks, i, vars, topNames, importRefs, fieldTypes)
		if !ok {
			continue
		}
		replacements = append(replacements, repl)
		i = close
	}
	for i := bodyOpen + 1; i+2 < bodyClose; i++ {
		if toks[i].Kind != scan.Ident || toks[i+1].Text != "." {
			continue
		}
		info, ok := staticInterfaceAssertionVarByNameForLower(vars, toks[i].Text)
		if !ok {
			continue
		}
		assertClose := findClose(toks, i+2, "(", ")")
		if assertClose < 0 {
			continue
		}
		asserted := staticInterfaceAssertionTypeNameForLower(toks, i+1)
		commaOK := staticInterfaceAssertionCommaOKContextForLower(toks, i, assertClose)
		text := info.name
		if asserted != info.concreteType {
			if !commaOK {
				if !staticInterfaceAssertionSupportedTypeForLower(asserted) {
					continue
				}
				repl, ok := staticInterfaceAssertionPanicReplacementForLower(body, toks, i, assertClose, info.concreteType, asserted)
				if ok {
					replacements = append(replacements, repl)
					i = assertClose
				}
				continue
			}
			zero, ok := staticInterfaceAssertionZeroValueTextForLower(asserted)
			if !ok || !staticInterfaceAssertionCommaOKSupportedTypeForLower(asserted, topNames, importRefs, fieldTypes) {
				continue
			}
			text = zero + ", false"
		} else if commaOK {
			text = info.name + ", true"
		}
		replacements = append(replacements, expressionReplacement{
			start: int(toks[i].Start),
			end:   int(toks[assertClose].End),
			text:  text,
		})
		i = assertClose
	}
	if len(replacements) == 0 {
		return body
	}
	sortExpressionReplacementsByStart(replacements)
	return applyExpressionReplacements(body, 0, len(body), replacements)
}

func staticInterfaceAssertionVarAtForLower(toks []scan.Token, start int, bodyOpen int, bodyClose int, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) (staticInterfaceAssertionVarForLower, bool) {
	stmtEnd := lowerSimpleStatementEnd(toks, start, bodyClose)
	if start < 0 || start+3 > stmtEnd || toks[start].Text != "var" {
		return staticInterfaceAssertionVarForLower{}, false
	}
	if toks[start+1].Kind != scan.Ident || toks[start+1].Text == "_" {
		return staticInterfaceAssertionVarForLower{}, false
	}
	eq := findTopLevelToken(toks, start, stmtEnd, "=")
	typeEnd := stmtEnd
	if eq >= 0 {
		typeEnd = eq
	}
	if start+2 >= typeEnd || !interfaceVarTypeRangeForLower(toks, start+2, typeEnd) {
		return staticInterfaceAssertionVarForLower{}, false
	}
	concrete := "nil"
	nilInterface := true
	initStart := typeEnd
	initEnd := typeEnd
	if eq >= 0 {
		if eq+1 >= stmtEnd {
			return staticInterfaceAssertionVarForLower{}, false
		}
		values := topLevelExpressionRanges(toks, eq+1, stmtEnd)
		if len(values) != 1 {
			return staticInterfaceAssertionVarForLower{}, false
		}
		value := values[0]
		if expressionContainsCall(toks, value.start, value.end) {
			return staticInterfaceAssertionVarForLower{}, false
		}
		initStart = value.start
		initEnd = value.end
		if nilInterfaceInitializerForLower(toks, value.start, value.end) {
			concrete = "nil"
			nilInterface = true
		} else {
			concrete = staticInterfaceConcreteTypeForLower(toks, value.start, value.end)
			nilInterface = false
		}
		if concrete == "" {
			return staticInterfaceAssertionVarForLower{}, false
		}
	}
	info := staticInterfaceAssertionVarForLower{name: toks[start+1].Text, concreteType: concrete, nilInterface: nilInterface, declStart: start, declEnd: stmtEnd, initStart: initStart, initEnd: initEnd, bodyOpen: bodyOpen, bodyClose: bodyClose}
	if !staticInterfaceAssertionUsesLowerableForLower(toks, info, topNames, importRefs, fieldTypes) {
		return staticInterfaceAssertionVarForLower{}, false
	}
	return info, true
}

func staticInterfaceAssertionUsesLowerableForLower(toks []scan.Token, info staticInterfaceAssertionVarForLower, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) bool {
	seenAssertion := false
	for i := info.declEnd; i < info.bodyClose; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != info.name {
			continue
		}
		if staticInterfaceTypeSwitchUseLowerableForLower(toks, i, info, topNames) {
			seenAssertion = true
			open := i + 5
			close := findClose(toks, open, "{", "}")
			if close < 0 {
				return false
			}
			i = close
			continue
		}
		if i+1 < info.bodyClose && toks[i+1].Text == "." {
			close := findClose(toks, i+2, "(", ")")
			asserted := staticInterfaceAssertionTypeNameForLower(toks, i+1)
			if close < 0 || (asserted != info.concreteType && !staticInterfaceAssertionMismatchLowerableForLower(toks, i, close, asserted, topNames, importRefs, fieldTypes)) {
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

func staticInterfaceTypeSwitchUseLowerableForLower(toks []scan.Token, namePos int, info staticInterfaceAssertionVarForLower, topNames symbolNameTable) bool {
	if namePos < 1 || namePos+5 >= info.bodyClose || toks[namePos].Kind != scan.Ident || toks[namePos].Text != info.name {
		return false
	}
	for _, switchPos := range []int{namePos - 1, namePos - 3} {
		guard, ok := staticInterfaceTypeSwitchGuardAtForLower(toks, switchPos)
		if ok && guard.sourcePos == namePos {
			_, _, _, ok := staticInterfaceTypeSwitchCaseBodyForLower(toks, switchPos, info, topNames)
			return ok
		}
	}
	return false
}

func staticInterfaceTypeSwitchReplacementForLower(body string, toks []scan.Token, switchPos int, vars []staticInterfaceAssertionVarForLower, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) (expressionReplacement, int, bool) {
	guard, ok := staticInterfaceTypeSwitchGuardAtForLower(toks, switchPos)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	info, ok := staticInterfaceAssertionVarByNameForLower(vars, guard.sourceName)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	bodyStart, bodyEnd, matchedCase, ok := staticInterfaceTypeSwitchCaseBodyForLower(toks, switchPos, info, topNames)
	if !ok {
		return expressionReplacement{}, 0, false
	}
	open := guard.open
	close := findClose(toks, open, "{", "}")
	if close < 0 {
		return expressionReplacement{}, 0, false
	}
	indent := statementIndent(body, int(toks[switchPos].Start))
	replacement := "{}"
	if bodyStart < bodyEnd {
		eraseBinding := !matchedCase || info.nilInterface
		bodyText := staticInterfaceTypeSwitchSelectedBodyForLower(body, toks, bodyStart, bodyEnd, info, guard.bindingName, eraseBinding, topNames, importRefs, fieldTypes)
		if guard.bindingName != "" && matchedCase && !info.nilInterface {
			bindIndent := indent + "\t"
			if bodyStart < bodyEnd {
				bindIndent = statementIndent(body, int(toks[bodyStart].Start))
			}
			bodyText = bindIndent + "var " + guard.bindingName + " " + info.concreteType + " = " + info.name + "\n" + bodyText
		}
		if strings.TrimSpace(bodyText) != "" {
			replacement = "{\n" + bodyText
			if len(bodyText) == 0 || bodyText[len(bodyText)-1] != '\n' {
				replacement += "\n"
			}
			replacement += indent + "}"
		}
	}
	return expressionReplacement{start: int(toks[switchPos].Start), end: int(toks[close].End), text: replacement}, close, true
}

func staticInterfaceTypeSwitchCaseBodyForLower(toks []scan.Token, switchPos int, info staticInterfaceAssertionVarForLower, topNames symbolNameTable) (int, int, bool, bool) {
	guard, ok := staticInterfaceTypeSwitchGuardAtForLower(toks, switchPos)
	if !ok || guard.sourceName != info.name {
		return 0, 0, false, false
	}
	if guard.bindingName != "" && guard.bindingName == info.name {
		return 0, 0, false, false
	}
	open := guard.open
	close := findClose(toks, open, "{", "}")
	if close < 0 || close > info.bodyClose {
		return 0, 0, false, false
	}
	selectedStart := -1
	selectedEnd := -1
	defaultStart := -1
	defaultEnd := -1
	seenClause := false
	for i := open + 1; i < close; i++ {
		if !topLevelInRange(toks, open+1, i) {
			continue
		}
		if toks[i].Text != "case" && toks[i].Text != "default" {
			if toks[i].Text == "break" || toks[i].Text == "fallthrough" {
				return 0, 0, false, false
			}
			continue
		}
		colon := caseClauseColon(toks, i, close)
		if colon < 0 {
			return 0, 0, false, false
		}
		next := nextCaseClauseStartForLower(toks, colon+1, close)
		seenClause = true
		if toks[i].Text == "default" {
			defaultStart = colon + 1
			defaultEnd = next
			i = next - 1
			continue
		}
		match, lowerable := staticInterfaceTypeSwitchCaseListMatchForLower(toks, i+1, colon, info.concreteType, topNames)
		if !lowerable {
			return 0, 0, false, false
		}
		if match && selectedStart < 0 {
			selectedStart = colon + 1
			selectedEnd = next
		}
		i = next - 1
	}
	if !seenClause {
		return 0, 0, false, false
	}
	if selectedStart >= 0 {
		return selectedStart, selectedEnd, true, true
	}
	if defaultStart >= 0 {
		if guard.bindingName != "" && !staticInterfaceTypeSwitchDefaultBindingLowerableForLower(toks, open, defaultStart, defaultEnd, guard.bindingName) {
			return 0, 0, false, false
		}
		return defaultStart, defaultEnd, false, true
	}
	return close, close, false, true
}

func staticInterfaceTypeSwitchGuardAtForLower(toks []scan.Token, switchPos int) (staticInterfaceTypeSwitchGuardForLower, bool) {
	if switchPos < 0 || switchPos >= len(toks) || toks[switchPos].Text != "switch" {
		return staticInterfaceTypeSwitchGuardForLower{}, false
	}
	if switchPos+6 < len(toks) && toks[switchPos+1].Kind == scan.Ident && toks[switchPos+2].Text == "." {
		dot := switchPos + 2
		assertClose := findClose(toks, dot+1, "(", ")")
		if assertClose == dot+3 && toks[dot+2].Text == "type" && assertClose+1 < len(toks) && toks[assertClose+1].Text == "{" {
			return staticInterfaceTypeSwitchGuardForLower{sourceName: toks[switchPos+1].Text, sourcePos: switchPos + 1, dot: dot, open: assertClose + 1}, true
		}
	}
	if switchPos+8 < len(toks) && toks[switchPos+1].Kind == scan.Ident && toks[switchPos+2].Text == ":=" && toks[switchPos+3].Kind == scan.Ident && toks[switchPos+4].Text == "." {
		dot := switchPos + 4
		assertClose := findClose(toks, dot+1, "(", ")")
		if assertClose == dot+3 && toks[dot+2].Text == "type" && assertClose+1 < len(toks) && toks[assertClose+1].Text == "{" {
			return staticInterfaceTypeSwitchGuardForLower{sourceName: toks[switchPos+3].Text, bindingName: toks[switchPos+1].Text, sourcePos: switchPos + 3, dot: dot, open: assertClose + 1}, true
		}
	}
	return staticInterfaceTypeSwitchGuardForLower{}, false
}

func staticInterfaceTypeSwitchCaseListMatchForLower(toks []scan.Token, start int, end int, concrete string, topNames symbolNameTable) (bool, bool) {
	values := topLevelExpressionRanges(toks, start, end)
	if len(values) == 0 {
		return false, false
	}
	matched := false
	for i := 0; i < len(values); i++ {
		value := values[i]
		value.start, value.end = trimTokenRange(toks, value.start, value.end)
		typ := staticInterfaceTypeTextForLower(toks, value.start, value.end)
		if typ == "" {
			return false, false
		}
		if typ == "nil" {
			if concrete == "nil" {
				matched = true
			}
			continue
		}
		typ = staticInterfaceNormalizeTypeForLower(typ, topNames)
		if typ == concrete {
			matched = true
		}
	}
	return matched, true
}

func staticInterfaceNormalizeTypeForLower(typ string, topNames symbolNameTable) string {
	prefix := ""
	for strings.HasPrefix(typ, "*") {
		prefix += "*"
		typ = typ[1:]
	}
	if strings.Contains(typ, ".") {
		return prefix + typ
	}
	if unitName := symbolNameTableUnitName(topNames, typ); unitName != "" {
		return prefix + unitName
	}
	return prefix + typ
}

func nextCaseClauseStartForLower(toks []scan.Token, start int, close int) int {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < close; i++ {
		text := toks[i].Text
		if paren == 0 && brack == 0 && brace == 0 && (text == "case" || text == "default") {
			return i
		}
		updateExpressionDepth(text, &paren, &brack, &brace)
	}
	return close
}

func staticInterfaceTypeSwitchSelectedBodyForLower(body string, toks []scan.Token, start int, end int, info staticInterfaceAssertionVarForLower, bindingName string, eraseBinding bool, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) string {
	startByte := int(toks[start].Start)
	endByte := startByte
	if end > start {
		endByte = int(toks[end-1].End)
	}
	var replacements []expressionReplacement
	for i := start; i+2 < end; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != info.name || toks[i+1].Text != "." {
			continue
		}
		assertClose := findClose(toks, i+2, "(", ")")
		if assertClose < 0 || assertClose >= end {
			continue
		}
		asserted := staticInterfaceAssertionTypeNameForLower(toks, i+1)
		commaOK := staticInterfaceAssertionCommaOKContextForLower(toks, i, assertClose)
		text := info.name
		if asserted != info.concreteType {
			if !commaOK {
				if !staticInterfaceAssertionSupportedTypeForLower(asserted) {
					continue
				}
				repl, ok := staticInterfaceAssertionPanicReplacementForLower(body, toks, i, assertClose, info.concreteType, asserted)
				if ok {
					replacements = append(replacements, repl)
					i = assertClose
				}
				continue
			}
			zero, ok := staticInterfaceAssertionZeroValueTextForLower(asserted)
			if !ok || !staticInterfaceAssertionCommaOKSupportedTypeForLower(asserted, topNames, importRefs, fieldTypes) {
				continue
			}
			text = zero + ", false"
		} else if commaOK {
			text = info.name + ", true"
		}
		replacements = append(replacements, expressionReplacement{start: int(toks[i].Start), end: int(toks[assertClose].End), text: text})
		i = assertClose
	}
	if eraseBinding && bindingName != "" {
		for i := start; i < end; i++ {
			if toks[i].Kind != scan.Ident || toks[i].Text != bindingName {
				continue
			}
			stmtStart := simpleStatementStartForLower(toks, info.bodyOpen, i)
			stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, end)
			if !blankDiscardOfIdentifierStatementForLower(toks, stmtStart, stmtEnd, bindingName) {
				continue
			}
			replEnd := int(toks[stmtEnd-1].End)
			if stmtEnd < end && toks[stmtEnd].Text == ";" {
				replEnd = int(toks[stmtEnd].End)
			}
			replacements = append(replacements, expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: ""})
			i = stmtEnd - 1
		}
	}
	if len(replacements) == 0 {
		return body[startByte:endByte]
	}
	sortExpressionReplacementsByStart(replacements)
	return applyExpressionReplacements(body, startByte, endByte, replacements)
}

func topLevelInRange(toks []scan.Token, start int, pos int) bool {
	paren := 0
	brack := 0
	brace := 0
	for i := start; i < pos; i++ {
		updateExpressionDepth(toks[i].Text, &paren, &brack, &brace)
	}
	return paren == 0 && brack == 0 && brace == 0
}

func staticInterfaceTypeSwitchDefaultBindingLowerableForLower(toks []scan.Token, scopeOpen int, start int, end int, binding string) bool {
	if binding == "" {
		return true
	}
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != binding {
			continue
		}
		stmtStart := simpleStatementStartForLower(toks, scopeOpen, i)
		stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, end)
		if !blankDiscardOfIdentifierStatementForLower(toks, stmtStart, stmtEnd, binding) {
			return false
		}
		i = stmtEnd - 1
	}
	return true
}

func staticInterfaceAssertionVarByNameForLower(vars []staticInterfaceAssertionVarForLower, name string) (staticInterfaceAssertionVarForLower, bool) {
	for i := 0; i < len(vars); i++ {
		if vars[i].name == name {
			return vars[i], true
		}
	}
	return staticInterfaceAssertionVarForLower{}, false
}

func staticInterfaceAssertionTypeMatchesForLower(toks []scan.Token, dot int, concrete string) bool {
	return staticInterfaceAssertionTypeNameForLower(toks, dot) == concrete
}

func staticInterfaceAssertionTypeNameForLower(toks []scan.Token, dot int) string {
	if dot <= 0 || dot+2 >= len(toks) || toks[dot].Text != "." || toks[dot+1].Text != "(" {
		return ""
	}
	close := findClose(toks, dot+1, "(", ")")
	if close < 0 {
		return ""
	}
	start, end := trimTokenRange(toks, dot+2, close)
	if start+1 == end && toks[start].Text == "type" {
		return ""
	}
	return staticInterfaceTypeTextForLower(toks, start, end)
}

func staticInterfaceAssertionSupportedTypeForLower(typ string) bool {
	switch typ {
	case "int", "string", "bool":
		return true
	}
	return false
}

func staticInterfaceAssertionMismatchLowerableForLower(toks []scan.Token, pos int, close int, asserted string, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) bool {
	if staticInterfaceAssertionCommaOKContextForLower(toks, pos, close) {
		return staticInterfaceAssertionCommaOKSupportedTypeForLower(asserted, topNames, importRefs, fieldTypes)
	}
	if staticInterfaceAssertionPanicContextForLower(toks, pos, close) {
		return staticInterfaceAssertionSupportedTypeForLower(asserted)
	}
	return false
}

func staticInterfaceAssertionCommaOKSupportedTypeForLower(typ string, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) bool {
	return staticInterfaceAssertionSupportedTypeForLower(typ) || staticInterfaceStructTypeSupportedForLower(typ, topNames, importRefs, fieldTypes)
}

func staticInterfaceAssertionZeroValueForLower(typ string) string {
	value, ok := staticInterfaceAssertionZeroValueTextForLower(typ)
	if ok {
		return value
	}
	return "0"
}

func staticInterfaceAssertionZeroValueTextForLower(typ string) (string, bool) {
	switch typ {
	case "int":
		return "0", true
	case "string":
		return `""`, true
	case "bool":
		return "false", true
	}
	if strings.HasPrefix(typ, "*") {
		return "nil", true
	}
	if typ != "" && !strings.HasPrefix(typ, "[]") {
		return typ + "{}", true
	}
	return "", false
}

func staticInterfaceStructTypeSupportedForLower(typ string, topNames symbolNameTable, importRefs importSymbolTable, fieldTypes structFieldTypeTable) bool {
	for strings.HasPrefix(typ, "*") {
		typ = typ[1:]
	}
	if typ == "" || strings.HasPrefix(typ, "[]") {
		return false
	}
	if structFieldTypeTableOwnerExists(fieldTypes, typ) {
		return true
	}
	for i := 0; i < len(topNames); i++ {
		if topNames[i].unitName == typ && structFieldTypeTableOwnerExists(fieldTypes, topNames[i].name) {
			return true
		}
	}
	for groupIndex := 0; groupIndex < len(importRefs); groupIndex++ {
		group := importRefs[groupIndex]
		for symbolIndex := 0; symbolIndex < len(group.symbols); symbolIndex++ {
			sym := group.symbols[symbolIndex]
			owner := importedQualifiedLowerName(group.localName, sym.Name)
			if (typ == owner || typ == sym.UnitName) && structFieldTypeTableOwnerExists(fieldTypes, owner) {
				return true
			}
		}
	}
	return false
}

func staticInterfaceAssertionCommaOKContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 2 || len(rhs) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if staticInterfaceAssertionIfPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionForPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionSwitchPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionDeferPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionCallPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionReturnPanicContextForLower(toks, pos, close) {
		return true
	}
	if staticInterfaceAssertionVarDeclPanicContextForLower(toks, pos, close) {
		return true
	}
	if close < 0 {
		return false
	}
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	shortDecl := assign >= 0
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, stmtEnd)
	if len(lhs) != 1 || len(rhs) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start != pos || end != close+1 {
		return false
	}
	name := singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end)
	if name != "" && name != "_" {
		return true
	}
	return !shortDecl && name == "_"
}

func staticInterfaceAssertionIfPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 || staticInterfaceAssertionTypeNameForLower(toks, pos+1) != "bool" {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "if" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionIfConditionRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionIfConditionRangeForLower(toks []scan.Token, ifPos int) (int, int, int, bool) {
	if ifPos < 0 || ifPos >= len(toks) || toks[ifPos].Text != "if" {
		return 0, 0, 0, false
	}
	open := conditionExpressionEnd(toks, ifPos)
	if open >= len(toks) || toks[open].Text != "{" {
		return 0, 0, 0, false
	}
	if topLevelSemicolon(toks, ifPos+1, open) >= 0 {
		return 0, 0, 0, false
	}
	start, end := trimTokenRange(toks, ifPos+1, open)
	return start, end, open, start < end
}

func staticInterfaceAssertionForPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 || staticInterfaceAssertionTypeNameForLower(toks, pos+1) != "bool" {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "for" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionForConditionRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionForConditionRangeForLower(toks []scan.Token, forPos int) (int, int, int, bool) {
	if forPos < 0 || forPos >= len(toks) || toks[forPos].Text != "for" {
		return 0, 0, 0, false
	}
	open := conditionExpressionEnd(toks, forPos)
	if open >= len(toks) || toks[open].Text != "{" {
		return 0, 0, 0, false
	}
	if topLevelSemicolon(toks, forPos+1, open) >= 0 {
		return 0, 0, 0, false
	}
	start, end := trimTokenRange(toks, forPos+1, open)
	if start < end && toks[start].Text == "range" {
		return 0, 0, 0, false
	}
	return start, end, open, start < end
}

func staticInterfaceAssertionSwitchPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "switch" {
		return false
	}
	start, end, _, ok := staticInterfaceAssertionSwitchTagRangeForLower(toks, stmtStart)
	if !ok {
		return false
	}
	return start == pos && end == close+1
}

func staticInterfaceAssertionSwitchTagRangeForLower(toks []scan.Token, switchPos int) (int, int, int, bool) {
	if switchPos < 0 || switchPos >= len(toks) || toks[switchPos].Text != "switch" {
		return 0, 0, 0, false
	}
	open := conditionExpressionEnd(toks, switchPos)
	if open >= len(toks) || toks[open].Text != "{" {
		return 0, 0, 0, false
	}
	if topLevelSemicolon(toks, switchPos+1, open) >= 0 {
		return 0, 0, 0, false
	}
	start, end := trimTokenRange(toks, switchPos+1, open)
	return start, end, open, start < end
}

func staticInterfaceAssertionDeferPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart+2 >= len(toks) || toks[stmtStart].Text != "defer" || toks[stmtStart+1].Kind != scan.Ident || toks[stmtStart+2].Text != "(" {
		return false
	}
	callClose := findClose(toks, stmtStart+2, "(", ")")
	if callClose < 0 {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	if callClose != stmtEnd-1 {
		return false
	}
	args := topLevelExpressionRanges(toks, stmtStart+3, callClose)
	if len(args) < 1 {
		return false
	}
	start, end := trimTokenRange(toks, args[0].start, args[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionCallPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart+1 >= len(toks) || toks[stmtStart].Kind != scan.Ident || toks[stmtStart+1].Text != "(" {
		return false
	}
	callClose := findClose(toks, stmtStart+1, "(", ")")
	if callClose < 0 {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	if callClose != stmtEnd-1 {
		return false
	}
	args := topLevelExpressionRanges(toks, stmtStart+2, callClose)
	if len(args) < 1 {
		return false
	}
	start, end := trimTokenRange(toks, args[0].start, args[0].end)
	return start == pos && end == close+1
}

func staticInterfaceAssertionVarDeclPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart+1 >= len(toks) || toks[stmtStart].Text != "var" || toks[stmtStart+1].Kind != scan.Ident || toks[stmtStart+1].Text == "_" {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	eq := findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	if eq < 0 || eq <= stmtStart+1 {
		return false
	}
	if findTopLevelToken(toks, stmtStart+1, eq, ",") >= 0 {
		return false
	}
	rhs := topLevelExpressionRanges(toks, eq+1, stmtEnd)
	if len(rhs) != 1 {
		return false
	}
	start, end := trimTokenRange(toks, rhs[0].start, rhs[0].end)
	if start != pos || end != close+1 {
		return false
	}
	if stmtStart+2 >= eq {
		return true
	}
	typeStart, typeEnd := trimTokenRange(toks, stmtStart+2, eq)
	return typeStart+1 == typeEnd && toks[typeStart].Text == staticInterfaceAssertionTypeNameForLower(toks, pos+1)
}

func staticInterfaceAssertionReturnPanicContextForLower(toks []scan.Token, pos int, close int) bool {
	if close < 0 {
		return false
	}
	stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
	if stmtStart < 0 || stmtStart >= len(toks) || toks[stmtStart].Text != "return" {
		return false
	}
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	values := topLevelExpressionRanges(toks, stmtStart+1, stmtEnd)
	if len(values) < 1 {
		return false
	}
	start, end := trimTokenRange(toks, values[0].start, values[0].end)
	return start == pos && end == close+1
}

func sameLineSimpleStatementStartForLower(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i >= 0; i-- {
		if toks[i].Line != line || toks[i].Text == ";" || toks[i].Text == "{" || toks[i].Text == "}" {
			return i + 1
		}
	}
	return 0
}

func staticInterfaceAssertionPanicReplacementForLower(body string, toks []scan.Token, pos int, close int, concrete string, asserted string) (expressionReplacement, bool) {
	if !staticInterfaceAssertionPanicContextForLower(toks, pos, close) {
		return expressionReplacement{}, false
	}
	message := staticInterfaceAssertionPanicMessageForLower(concrete, asserted)
	if staticInterfaceAssertionIfPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		_, _, open, ok := staticInterfaceAssertionIfConditionRangeForLower(toks, stmtStart)
		if !ok {
			return expressionReplacement{}, false
		}
		stmtEnd := conditionalStatementEnd(toks, stmtStart, open)
		if stmtEnd < 0 {
			return expressionReplacement{}, false
		}
		replEnd := int(toks[stmtEnd].End)
		if stmtEnd+1 < len(toks) && toks[stmtEnd+1].Text == ";" {
			replEnd = int(toks[stmtEnd+1].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionForPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		_, _, open, ok := staticInterfaceAssertionForConditionRangeForLower(toks, stmtStart)
		if !ok {
			return expressionReplacement{}, false
		}
		stmtEnd := conditionalStatementEnd(toks, stmtStart, open)
		if stmtEnd < 0 {
			return expressionReplacement{}, false
		}
		replEnd := int(toks[stmtEnd].End)
		if stmtEnd+1 < len(toks) && toks[stmtEnd+1].Text == ";" {
			replEnd = int(toks[stmtEnd+1].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionSwitchPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		_, _, open, ok := staticInterfaceAssertionSwitchTagRangeForLower(toks, stmtStart)
		if !ok {
			return expressionReplacement{}, false
		}
		stmtEnd := conditionalStatementEnd(toks, stmtStart, open)
		if stmtEnd < 0 {
			return expressionReplacement{}, false
		}
		replEnd := int(toks[stmtEnd].End)
		if stmtEnd+1 < len(toks) && toks[stmtEnd+1].Text == ";" {
			replEnd = int(toks[stmtEnd+1].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionDeferPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
		replEnd := int(toks[stmtEnd-1].End)
		if stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
			replEnd = int(toks[stmtEnd].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionCallPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
		replEnd := int(toks[stmtEnd-1].End)
		if stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
			replEnd = int(toks[stmtEnd].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionReturnPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
		replEnd := int(toks[stmtEnd-1].End)
		if stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
			replEnd = int(toks[stmtEnd].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: `panic("` + message + `")`}, true
	}
	if staticInterfaceAssertionVarDeclPanicContextForLower(toks, pos, close) {
		stmtStart := sameLineSimpleStatementStartForLower(toks, pos)
		stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
		name := toks[stmtStart+1].Text
		indent := statementIndent(body, int(toks[stmtStart].Start))
		text := "var " + name + " " + asserted + " = " + staticInterfaceAssertionZeroValueForLower(asserted) + "\n" + indent + `panic("` + message + `")`
		replEnd := int(toks[stmtEnd-1].End)
		if stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
			replEnd = int(toks[stmtEnd].End)
		}
		return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: text}, true
	}
	stmtStart := sameLineAssignmentStatementStartForLower(toks, pos)
	stmtEnd := lowerSimpleStatementEnd(toks, stmtStart, len(toks))
	assign := findTopLevelToken(toks, stmtStart, stmtEnd, ":=")
	shortDecl := assign >= 0
	if assign < 0 {
		assign = findTopLevelToken(toks, stmtStart, stmtEnd, "=")
	}
	lhs := topLevelExpressionRanges(toks, stmtStart, assign)
	if len(lhs) != 1 {
		return expressionReplacement{}, false
	}
	lhsText := strings.TrimSpace(tokenRangeText(body, toks, lhs[0].start, lhs[0].end))
	text := `panic("` + message + `")`
	if shortDecl {
		indent := statementIndent(body, int(toks[stmtStart].Start))
		text = lhsText + " := " + staticInterfaceAssertionZeroValueForLower(asserted) + "\n" + indent + text
	}
	replEnd := int(toks[stmtEnd-1].End)
	if stmtEnd < len(toks) && toks[stmtEnd].Text == ";" {
		replEnd = int(toks[stmtEnd].End)
	}
	return expressionReplacement{start: int(toks[stmtStart].Start), end: replEnd, text: text}, true
}

func staticInterfaceAssertionPanicMessageForLower(concrete string, asserted string) string {
	if concrete == "nil" {
		return "interface conversion: interface {} is nil, not " + asserted
	}
	return "interface conversion: interface {} is " + concrete + ", not " + asserted
}

func staticInterfaceConcreteTypeForLower(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return ""
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return staticInterfaceConcreteTypeForLower(toks, start+1, close)
		}
	}
	if (toks[start].Text == "+" || toks[start].Text == "-") && start+1 < end {
		typ := staticInterfaceConcreteTypeForLower(toks, start+1, end)
		if typ == "int" {
			return typ
		}
		return ""
	}
	if typ := staticInterfaceCompositeConcreteTypeForLower(toks, start, end); typ != "" {
		return typ
	}
	if start+1 != end {
		return ""
	}
	tok := toks[start]
	if tok.Kind == scan.String {
		return "string"
	}
	if tok.Kind == scan.Char {
		return "int"
	}
	if tok.Kind == scan.Number {
		if strings.ContainsAny(tok.Text, ".eEpP") {
			return ""
		}
		return "int"
	}
	if tok.Kind == scan.Ident && (tok.Text == "true" || tok.Text == "false") {
		return "bool"
	}
	return ""
}

func staticInterfaceCompositeConcreteTypeForLower(toks []scan.Token, start int, end int) string {
	if start < end && toks[start].Text == "&" {
		typ := staticInterfaceCompositeConcreteTypeForLower(toks, start+1, end)
		if typ == "" {
			return ""
		}
		return "*" + typ
	}
	open := compositeLiteralOpenForTypeStart(toks, start)
	if open < 0 {
		return ""
	}
	close := findClose(toks, open, "{", "}")
	if close != end-1 {
		return ""
	}
	return staticInterfaceTypeTextForLower(toks, start, open)
}

func staticInterfaceTypeTextForLower(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start < end && toks[start].Text == "*" {
		inner := staticInterfaceTypeTextForLower(toks, start+1, end)
		if inner == "" {
			return ""
		}
		return "*" + inner
	}
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text
	}
	if start+3 == end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident {
		return toks[start].Text + "." + toks[start+2].Text
	}
	return ""
}

func interfaceVarTypeRangeForLower(toks []scan.Token, start int, end int) bool {
	start, end = trimTokenRange(toks, start, end)
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

func nextSimpleStatementStartForLower(toks []scan.Token, pos int, limit int) int {
	for pos < limit && toks[pos].Text == ";" {
		pos++
	}
	if pos >= limit || toks[pos].Text == "}" {
		return -1
	}
	return pos
}

func blankDiscardOfIdentifierStatementForLower(toks []scan.Token, start int, end int, name string) bool {
	assign := findTopLevelToken(toks, start, end, "=")
	if assign < 0 || lowerCompoundAssignmentEquals(toks, assign) {
		return false
	}
	lhs := topLevelExpressionRanges(toks, start, assign)
	rhs := topLevelExpressionRanges(toks, assign+1, end)
	return len(lhs) == 1 && len(rhs) == 1 && singleIdentifierExpressionInLower(toks, lhs[0].start, lhs[0].end) == "_" && singleIdentifierExpressionInLower(toks, rhs[0].start, rhs[0].end) == name
}

func identifierUsedOutsideRangesForLower(toks []scan.Token, name string, scopeStart int, scopeEnd int, skipStart int, skipEnd int) bool {
	for i := scopeStart; i < len(toks) && i < scopeEnd; i++ {
		if toks[i].Kind != scan.Ident || toks[i].Text != name {
			continue
		}
		if i >= skipStart && i < skipEnd {
			continue
		}
		return true
	}
	return false
}

func discardedComplexSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedComplexSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	if _, _, ok := reducibleComplexLiteralPartsForLower(toks, start, end); ok {
		return nil, true
	}
	if lines, ok := discardedComplexBinarySideEffectStatementsForLower(body, toks, start, end, topNames); ok {
		return lines, true
	}
	if start+3 > end || toks[start].Text != "complex" || toks[start+1].Text != "(" {
		return nil, false
	}
	if start > 0 && toks[start-1].Text == "." {
		return nil, false
	}
	close := findClose(toks, start+1, "(", ")")
	if close != end-1 {
		return nil, false
	}
	args := topLevelExpressionRanges(toks, start+2, close)
	if len(args) != 2 {
		return nil, false
	}
	var lines []string
	for i := 0; i < len(args); i++ {
		argLines, ok := discardedComplexComponentSideEffectStatementsForLower(body, toks, args[i].start, args[i].end, topNames)
		if !ok {
			return nil, false
		}
		lines = append(lines, argLines...)
	}
	return lines, true
}

func discardedComplexBinarySideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	op := topLevelPlusMinusOperatorForLower(toks, start, end)
	if op < 0 {
		return nil, false
	}
	leftLines, leftOK := discardedComplexSideEffectStatementsForLower(body, toks, start, op, topNames)
	if !leftOK {
		return nil, false
	}
	rightLines, rightOK := discardedComplexSideEffectStatementsForLower(body, toks, op+1, end, topNames)
	if !rightOK {
		return nil, false
	}
	lines := append(leftLines, rightLines...)
	return lines, true
}

func discardedComplexComponentSideEffectStatementsForLower(body string, toks []scan.Token, start int, end int, topNames symbolNameTable) ([]string, bool) {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return nil, false
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return discardedComplexComponentSideEffectStatementsForLower(body, toks, start+1, close, topNames)
		}
	}
	if discardedPureRealNumberExpressionForLower(toks, start, end) {
		return nil, true
	}
	if call, ok := discardedDirectCallExpressionTextForLower(body, toks, start, end, topNames); ok {
		name := "rtg_discard_complex_" + strconv.Itoa(int(toks[start].Start))
		return []string{name + " := " + call}, true
	}
	return nil, false
}

func discardedPureRealNumberExpressionForLower(toks []scan.Token, start int, end int) bool {
	_, imaginary, ok := signedNumberLiteralTextForLower(toks, start, end)
	return ok && !imaginary
}

func simpleStatementStartForLower(toks []scan.Token, body int, pos int) int {
	line := toks[pos].Line
	for i := pos - 1; i > body; i-- {
		if toks[i].Text == "{" || toks[i].Text == "}" || toks[i].Text == ";" {
			return i + 1
		}
		if toks[i].Line != line {
			return i + 1
		}
	}
	return body + 1
}

func singleIdentifierExpressionInLower(toks []scan.Token, start int, end int) string {
	start, end = trimTokenRange(toks, start, end)
	if start+1 == end && toks[start].Kind == scan.Ident {
		return toks[start].Text
	}
	return ""
}

func collectFuncSignatureLocalTypes(toks []scan.Token, start int, end int, types *localTypeTable) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterListLocalTypes(toks, i+1, close, types)
		i = close
	}
}

func collectParameterListLocalTypes(toks []scan.Token, start int, end int, types *localTypeTable) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			typeStart := i + 1
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeInfoInRange(toks, typeStart, typeEnd)
			if typ.name != "" {
				typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
				*types = localTypeTableSet(*types, toks[i].Text, typ)
			}
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			typeStart := i + 3
			for typeStart < end && toks[typeStart].Text == "," {
				typeStart++
			}
			typeEnd := parameterTypeEnd(toks, typeStart, end)
			typ := typeInfoInRange(toks, typeStart, typeEnd)
			if typ.name != "" {
				info := typ
				info.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
				*types = localTypeTableSet(*types, toks[i].Text, info)
				*types = localTypeTableSet(*types, toks[i+2].Text, info)
			}
		}
	}
}

func collectShortDeclLocalTypes(toks []scan.Token, assign int, types *localTypeTable) {
	if assign-1 < 0 || assign+2 >= len(toks) {
		return
	}
	if toks[assign-1].Kind != scan.Ident {
		return
	}
	if toks[assign+1].Kind == scan.Ident && toks[assign+2].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{name: toks[assign+1].Text})
		return
	}
	if info, ok := sliceLiteralElementTypeInfo(toks, assign+1); ok {
		*types = localTypeTableSet(*types, toks[assign-1].Text, info)
		return
	}
	if assign+4 < len(toks) && toks[assign+1].Kind == scan.Ident && toks[assign+2].Text == "." && toks[assign+3].Kind == scan.Ident && toks[assign+4].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{qualifier: toks[assign+1].Text, name: toks[assign+3].Text})
		return
	}
	if assign+3 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident && toks[assign+3].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{name: toks[assign+2].Text, pointer: true})
		return
	}
	if assign+5 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident && toks[assign+3].Text == "." && toks[assign+4].Kind == scan.Ident && toks[assign+5].Text == "{" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, localTypeInfo{qualifier: toks[assign+2].Text, name: toks[assign+4].Text, pointer: true})
		return
	}
	if assign+2 < len(toks) && toks[assign+1].Text == "&" && toks[assign+2].Kind == scan.Ident {
		pointed := localTypeTableLookup(*types, toks[assign+2].Text)
		if pointed.name != "" {
			pointed.pointer = true
			*types = localTypeTableSet(*types, toks[assign-1].Text, pointed)
		}
	}
	initEnd := shortDeclInitializerEnd(toks, assign+1)
	typ := localInitializerType(toks, assign+1, initEnd, *types)
	if typ.name != "" {
		*types = localTypeTableSet(*types, toks[assign-1].Text, typ)
	}
}

func collectVarLocalTypes(toks []scan.Token, pos int, end int, types *localTypeTable) {
	if pos+2 >= len(toks) || toks[pos+1].Kind != scan.Ident {
		return
	}
	if toks[pos+2].Text == "=" {
		if pos+4 < len(toks) && toks[pos+3].Kind == scan.Ident && toks[pos+4].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{name: toks[pos+3].Text})
		}
		if info, ok := sliceLiteralElementTypeInfo(toks, pos+3); ok {
			*types = localTypeTableSet(*types, toks[pos+1].Text, info)
		}
		if pos+6 < len(toks) && toks[pos+3].Kind == scan.Ident && toks[pos+4].Text == "." && toks[pos+5].Kind == scan.Ident && toks[pos+6].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{qualifier: toks[pos+3].Text, name: toks[pos+5].Text})
		}
		if pos+5 < len(toks) && toks[pos+3].Text == "&" && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{name: toks[pos+4].Text, pointer: true})
		}
		if pos+7 < len(toks) && toks[pos+3].Text == "&" && toks[pos+4].Kind == scan.Ident && toks[pos+5].Text == "." && toks[pos+6].Kind == scan.Ident && toks[pos+7].Text == "{" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, localTypeInfo{qualifier: toks[pos+4].Text, name: toks[pos+6].Text, pointer: true})
		}
		initEnd := varInitializerEnd(toks, pos+3, end)
		typ := localInitializerType(toks, pos+3, initEnd, *types)
		if typ.name != "" {
			*types = localTypeTableSet(*types, toks[pos+1].Text, typ)
		}
		return
	}
	typeStart := pos + 2
	typeEnd := varTypeEnd(toks, typeStart, end)
	typ := typeInfoInRange(toks, typeStart, typeEnd)
	if typ.name != "" {
		typ.pointer = typeRangeIsPointer(toks, typeStart, typeEnd)
		*types = localTypeTableSet(*types, toks[pos+1].Text, typ)
	}
}

func localInitializerType(toks []scan.Token, start int, end int, types localTypeTable) localTypeInfo {
	return localInitializerTypeWithFunctions(toks, start, end, types, nil)
}

func localInitializerTypeWithFunctions(toks []scan.Token, start int, end int, types localTypeTable, functionResults localTypeTable) localTypeInfo {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return localTypeInfo{}
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return localInitializerTypeWithFunctions(toks, start+1, close, types, functionResults)
		}
	}
	if start+1 == end {
		tok := toks[start]
		if tok.Kind == scan.String {
			return localTypeInfo{name: "string"}
		}
		if tok.Kind == scan.Number || tok.Kind == scan.Char {
			return localTypeInfo{name: "int"}
		}
		if tok.Text == "true" || tok.Text == "false" {
			return localTypeInfo{name: "bool"}
		}
		if tok.Kind == scan.Ident {
			return localTypeTableLookup(types, tok.Text)
		}
		return localTypeInfo{}
	}
	if toks[start].Kind == scan.Ident && toks[start].Text == "string" && start+1 < end && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			return localTypeInfo{name: "string"}
		}
	}
	if toks[start].Kind == scan.Ident && start+1 < end && toks[start+1].Text == "(" {
		close := findClose(toks, start+1, "(", ")")
		if close == end-1 {
			return localTypeTableLookup(functionResults, toks[start].Text)
		}
	}
	return localTypeInfo{}
}

func localInitializerTypeWithImportedFunctions(toks []scan.Token, start int, end int, types localTypeTable, functionResults localTypeTable, importRefs importSymbolTable, importLocalNames localNameTable) localTypeInfo {
	start, end = trimTokenRange(toks, start, end)
	if start >= end {
		return localTypeInfo{}
	}
	if toks[start].Text == "(" {
		close := findClose(toks, start, "(", ")")
		if close == end-1 {
			return localInitializerTypeWithImportedFunctions(toks, start+1, close, types, functionResults, importRefs, importLocalNames)
		}
	}
	if start+4 <= end && toks[start].Kind == scan.Ident && toks[start+1].Text == "." && toks[start+2].Kind == scan.Ident && toks[start+3].Text == "(" && !isLocalNameAt(importLocalNames, toks[start].Text, int(toks[start].Start)) {
		close := findClose(toks, start+3, "(", ")")
		if close == end-1 {
			group, ok := importSymbolTableGroup(importRefs, toks[start].Text)
			if ok {
				sym, ok := importFunctionSymbolByName(group, toks[start+2].Text)
				if ok {
					info := localTypeTableLookup(functionResults, sym.UnitName)
					if info.name != "" {
						return importedFunctionResultLocalTypeInfo(info, group)
					}
				}
			}
		}
	}
	return localInitializerTypeWithFunctions(toks, start, end, types, functionResults)
}

func importedFunctionResultLocalTypeInfo(info localTypeInfo, group importSymbolGroup) localTypeInfo {
	if info.name == "" || info.qualifier != "" || group.localName == "" || group.localName == "." {
		return info
	}
	for i := 0; i < len(group.symbols); i++ {
		sym := group.symbols[i]
		if sym.UnitName == info.name && sym.Name != "" {
			info.qualifier = group.localName
			info.name = sym.Name
			return info
		}
	}
	return info
}

func shortDeclInitializerEnd(toks []scan.Token, start int) int {
	line := toks[start].Line
	for i := start; i < len(toks); i++ {
		if toks[i].Line != line || isStatementBoundary(toks[i].Text) || toks[i].Text == "," {
			return i
		}
	}
	return len(toks)
}

func varInitializerEnd(toks []scan.Token, start int, end int) int {
	line := toks[start].Line
	for i := start; i < len(toks) && int(toks[i].Start) < end; i++ {
		if toks[i].Line != line || isStatementBoundary(toks[i].Text) || toks[i].Text == "," {
			return i
		}
	}
	return end
}

func parameterTypeEnd(toks []scan.Token, start int, end int) int {
	for i := start; i < end; i++ {
		if toks[i].Text == "," || toks[i].Text == ")" || toks[i].Text == "{" || toks[i].Text == "=" || toks[i].Text == ";" {
			return i
		}
	}
	return end
}

func varTypeEnd(toks []scan.Token, start int, end int) int {
	line := toks[start].Line
	for i := start; i < end; i++ {
		if toks[i].Line != line {
			return i
		}
		if toks[i].Text == "," || toks[i].Text == ")" || toks[i].Text == "{" || toks[i].Text == "=" || toks[i].Text == ";" {
			return i
		}
	}
	return end
}

func typeNameInRange(toks []scan.Token, start int, end int) string {
	info := typeInfoInRange(toks, start, end)
	return info.name
}

func typeInfoInRange(toks []scan.Token, start int, end int) localTypeInfo {
	name := ""
	qualifier := ""
	for i := start; i < end; i++ {
		if i+2 < end && toks[i].Kind == scan.Ident && toks[i+1].Text == "." && toks[i+2].Kind == scan.Ident {
			qualifier = toks[i].Text
			name = toks[i+2].Text
			i += 2
			continue
		}
		if toks[i].Kind == scan.Ident {
			qualifier = ""
			name = toks[i].Text
		}
	}
	return localTypeInfo{qualifier: qualifier, name: name}
}

func sliceLiteralElementTypeInfo(toks []scan.Token, start int) (localTypeInfo, bool) {
	typeStart, open, pointer, ok := sliceLiteralElementTypeRange(toks, start)
	if !ok {
		return localTypeInfo{}, false
	}
	info := typeInfoInRange(toks, typeStart, open)
	if info.name == "" {
		return localTypeInfo{}, false
	}
	info.pointer = pointer || typeRangeIsPointer(toks, typeStart, open)
	return info, true
}

func sliceLiteralElementTypeRange(toks []scan.Token, start int) (int, int, bool, bool) {
	if start+3 >= len(toks) || toks[start].Text != "[" || toks[start+1].Text != "]" {
		return 0, 0, false, false
	}
	typeStart := start + 2
	pointer := false
	if typeStart < len(toks) && toks[typeStart].Text == "*" {
		pointer = true
		typeStart++
	}
	open := compositeLiteralOpenForTypeStart(toks, typeStart)
	if open < 0 {
		return 0, 0, false, false
	}
	return typeStart, open, pointer, true
}

func typeRangeIsPointer(toks []scan.Token, start int, end int) bool {
	return containsTokenText(toks, start, end, "*")
}

func isLocalNameAt(names localNameTable, name string, pos int) bool {
	for i := 0; i < len(names); i++ {
		scope := names[i]
		if scope.name == name && pos >= scope.start && pos < scope.end {
			return true
		}
	}
	return false
}

func collectFuncSignatureLocals(toks []scan.Token, start int, end int, topNames symbolNameTable, names *localNameTable) {
	for i := start; i < end; i++ {
		if toks[i].Text != "(" {
			continue
		}
		close := findClose(toks, i, "(", ")")
		if close < 0 || close > end {
			continue
		}
		collectParameterListLocals(toks, i+1, close, topNames, names)
		i = close
	}
}

func collectParameterListLocals(toks []scan.Token, start int, end int, topNames symbolNameTable, names *localNameTable) {
	for i := start; i < end; i++ {
		if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
			continue
		}
		if i > start && toks[i-1].Text != "," {
			continue
		}
		if i+1 < end && isTypeStart(toks[i+1]) {
			addLocalName(names, toks[i].Text, 0, maxSourcePosition())
			continue
		}
		if i+2 < end && toks[i+1].Text == "," && toks[i+2].Kind == scan.Ident && isTypeStartAfterName(toks, i+2, end) {
			addLocalName(names, toks[i].Text, 0, maxSourcePosition())
		}
	}
}

func collectShortDeclLocals(toks []scan.Token, body int, assign int, declEnd int, topNames symbolNameTable, names *localNameTable) {
	line := toks[assign].Line
	scopeEnd := localScopeEnd(toks, body, assign, declEnd)
	for i := assign - 1; i >= 0; i-- {
		if toks[i].Line != line {
			return
		}
		if isStatementBoundary(toks[i].Text) {
			return
		}
		if toks[i].Kind == scan.Ident && symbolNameTableUnitName(topNames, toks[i].Text) != "" && (i == 0 || toks[i-1].Text != ".") {
			addLocalName(names, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
}

func collectVarLocals(toks []scan.Token, body int, pos int, end int, topNames symbolNameTable, names *localNameTable) {
	scopeEnd := localScopeEnd(toks, body, pos, end)
	if pos+1 < len(toks) && toks[pos+1].Text == "(" {
		for i := pos + 2; i < len(toks) && int(toks[i].Start) < end; i++ {
			if toks[i].Text == ")" || toks[i].Text == "}" {
				return
			}
			if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
				continue
			}
			if toks[i-1].Text == "(" || toks[i-1].Text == "," || toks[i-1].Line != toks[i].Line {
				addLocalName(names, toks[i].Text, int(toks[i].Start), scopeEnd)
			}
		}
		return
	}
	line := toks[pos].Line
	for i := pos + 1; i < len(toks) && int(toks[i].Start) < end && toks[i].Line == line; i++ {
		if toks[i].Text == ")" || toks[i].Text == "}" || toks[i].Text == ":=" {
			return
		}
		if toks[i].Text == "=" {
			return
		}
		if toks[i].Kind != scan.Ident || symbolNameTableUnitName(topNames, toks[i].Text) == "" {
			continue
		}
		if i == pos+1 || toks[i-1].Text == "," {
			addLocalName(names, toks[i].Text, int(toks[i].Start), scopeEnd)
		}
	}
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

func addLocalName(names *localNameTable, name string, start int, end int) {
	values := *names
	values = append(values, localNameRange{name: name, start: start, end: end})
	*names = values
}

func maxSourcePosition() int {
	return 2147483647
}

func tokenIndexAt(toks []scan.Token, start int) int {
	low := 0
	high := len(toks)
	for low < high {
		mid := (low + high) / 2
		tokStart := int(toks[mid].Start)
		if tokStart == start {
			return mid
		}
		if tokStart > start {
			high = mid
		} else {
			low = mid + 1
		}
	}
	return -1
}

func tokenIndexBeforeForLower(toks []scan.Token, end int) int {
	for i := 0; i < len(toks); i++ {
		if int(toks[i].Start) >= end {
			return i - 1
		}
	}
	return len(toks) - 1
}

func functionBodyOpenAfterParamsForLower(toks []scan.Token, paramsClose int, declEnd int) int {
	start := functionBodySearchStartAfterParamsForLower(toks, paramsClose, declEnd)
	if start < 0 {
		return -1
	}
	return findTokenText(toks, start, declEnd, "{")
}

func functionBodySearchStartAfterParamsForLower(toks []scan.Token, paramsClose int, declEnd int) int {
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

func findTokenText(toks []scan.Token, start int, end int, text string) int {
	for i := start; i < len(toks) && int(toks[i].Start) < end; i++ {
		if toks[i].Text == text {
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

func containsTokenText(toks []scan.Token, start int, end int, text string) bool {
	for i := start; i < end && i < len(toks); i++ {
		if toks[i].Text == text {
			return true
		}
	}
	return false
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

func containsString(values []string, value string) bool {
	for i := 0; i < len(values); i++ {
		if values[i] == value {
			return true
		}
	}
	return false
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func dependencyPackages(graph *load.Graph) []load.Package {
	var packages []load.Package
	if graph == nil {
		return packages
	}
	for i := 0; i < len(graph.Packages); i++ {
		dep := graph.Packages[i]
		dep.Files = snapshotLoadFiles(dep.Files)
		sortFilesByPath(dep.Files)
		packages = append(packages, dep)
	}
	return packages
}

func packageByImportPath(packages []load.Package, importPath string) (load.Package, bool) {
	for i := 0; i < len(packages); i++ {
		pkg := packages[i]
		if pkg.ImportPath == importPath {
			return pkg, true
		}
	}
	return load.Package{}, false
}

func mergedMethodMap(local methodTable, localOrder []string, imported methodTable, importedOrder []string) methodTable {
	if len(imported) == 0 {
		return local
	}
	var methods methodTable
	for i := 0; i < len(localOrder); i++ {
		name := localOrder[i]
		method := methodTableLookup(local, name)
		methods = methodTableSet(methods, name, method)
	}
	for i := 0; i < len(importedOrder); i++ {
		name := importedOrder[i]
		method := methodTableLookup(imported, name)
		existing := methodTableLookup(methods, name)
		if existing.unitName == "" {
			methods = methodTableSet(methods, name, method)
		}
	}
	return methods
}

func importReferenceMap(file *parse.File, packages []load.Package) (importSymbolTable, []string) {
	var refs importSymbolTable
	var refNames []string
	for impIndex := 0; impIndex < len(file.Imports); impIndex++ {
		imp := file.Imports[impIndex]
		localName := importLocalName(imp)
		importPath := imp.Path
		dep, ok := packageByImportPath(packages, importPath)
		if !ok || localName == "" || localName == "_" {
			continue
		}
		var symbols []unit.Symbol
		var functionSymbols []unit.Symbol
		for fileIndex := 0; fileIndex < len(dep.Files); fileIndex++ {
			names := dependencyExportedNames(dep.Files[fileIndex].Source)
			for nameIndex := 0; nameIndex < len(names); nameIndex++ {
				name := names[nameIndex]
				symbols = setSymbol(symbols, unit.Symbol{ImportPath: dep.ImportPath, Name: name, UnitName: SymbolName(dep.ImportPath, name)})
			}
			functionNames := dependencyExportedFunctionNames(dep.Files[fileIndex].Source)
			for nameIndex := 0; nameIndex < len(functionNames); nameIndex++ {
				name := functionNames[nameIndex]
				functionSymbols = setSymbol(functionSymbols, unit.Symbol{ImportPath: dep.ImportPath, Name: name, UnitName: SymbolName(dep.ImportPath, name)})
			}
		}
		intrinsicNames := intrinsicImportSymbolNames(importPath)
		for i := 0; i < len(intrinsicNames); i++ {
			name := intrinsicNames[i]
			intrinsic := intrinsicImportSymbol(importPath, name)
			symbols = setSymbol(symbols, unit.Symbol{Name: name, UnitName: intrinsic})
		}
		intrinsicFunctionNames := intrinsicImportFunctionSymbolNames(importPath)
		for i := 0; i < len(intrinsicFunctionNames); i++ {
			name := intrinsicFunctionNames[i]
			intrinsic := intrinsicImportSymbol(importPath, name)
			functionSymbols = setSymbol(functionSymbols, unit.Symbol{Name: name, UnitName: intrinsic})
		}
		refNames = append(refNames, localName)
		refs = append(refs, importSymbolGroup{localName: localName, importPath: dep.ImportPath, symbols: symbols, functionSymbols: functionSymbols})
	}
	return refs, refNames
}

func importMethodMap(file *parse.File, packages []load.Package, typeLocalNames []string) (methodTable, []string) {
	var methods methodTable
	var methodNames []string
	for impIndex := 0; impIndex < len(file.Imports); impIndex++ {
		imp := file.Imports[impIndex]
		localName := importLocalName(imp)
		importPath := imp.Path
		dep, ok := packageByImportPath(packages, importPath)
		if !ok || localName == "" || localName == "_" {
			continue
		}
		if !containsString(typeLocalNames, localName) {
			continue
		}
		for fileIndex := 0; fileIndex < len(dep.Files); fileIndex++ {
			depFile := dep.Files[fileIndex]
			methodsInFile := dependencyExportedMethods(importPath, depFile.Source)
			for methodIndex := 0; methodIndex < len(methodsInFile); methodIndex++ {
				info := methodsInFile[methodIndex]
				if info.name == "" || info.receiverType == "" {
					continue
				}
				methodName := localName + "." + info.name
				existing := methodTableLookup(methods, methodName)
				if existing.unitName == "" {
					methodNames = append(methodNames, methodName)
				}
				methods = methodTableSet(methods, methodName, info)
			}
		}
	}
	return methods, methodNames
}

func importedTypeLocalNames(file *parse.File) []string {
	var names []string
	decls := file.Decls
	for declIndex := 0; declIndex < len(decls); declIndex++ {
		decl := decls[declIndex]
		types := localTypesForDecl(file, &decl)
		for typeIndex := 0; typeIndex < len(types); typeIndex++ {
			qualifier := types[typeIndex].info.qualifier
			if qualifier != "" && !containsString(names, qualifier) {
				names = append(names, qualifier)
			}
		}
	}
	toks := file.Tokens
	for i := 0; i < len(toks); i++ {
		if i+3 < len(toks) && toks[i].Kind == scan.Ident && toks[i+1].Text == "." && toks[i+2].Kind == scan.Ident && toks[i+3].Text == "(" {
			names = appendStringUnique(names, toks[i].Text)
			continue
		}
		open := compositeLiteralOpenForTypeStart(toks, i)
		if open < 0 {
			continue
		}
		typ := typeInfoInRange(toks, i, open)
		if typ.qualifier != "" {
			names = appendStringUnique(names, typ.qualifier)
		}
	}
	return names
}

func dependencyExportedNames(src []byte) []string {
	toks, err := scan.Tokens(src)
	if err != nil {
		return nil
	}
	var names []string
	pos := dependencyTopLevelStart(toks)
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		tok := toks[pos]
		if tok.Text == "func" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				if close > next && close+1 < len(toks) {
					nameTok := close + 1
					if toks[nameTok].Kind == scan.Ident && isExported(toks[nameTok].Text) {
						names = appendStringUnique(names, toks[nameTok].Text)
					}
				}
			} else if next < len(toks) && toks[next].Kind == scan.Ident && isExported(toks[next].Text) {
				names = appendStringUnique(names, toks[next].Text)
			}
			pos = dependencySkipFunc(toks, pos+1)
			continue
		}
		if tok.Text == "type" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				names = appendExportedGroupedNames(names, toks, next+1, close)
				if close > next {
					pos = close + 1
					continue
				}
			} else if next < len(toks) && toks[next].Kind == scan.Ident && isExported(toks[next].Text) {
				names = appendStringUnique(names, toks[next].Text)
			}
			pos = dependencySkipLine(toks, pos)
			continue
		}
		if tok.Text == "const" || tok.Text == "var" {
			next := pos + 1
			if next < len(toks) && toks[next].Text == "(" {
				close := findClose(toks, next, "(", ")")
				names = appendExportedGroupedNames(names, toks, next+1, close)
				if close > next {
					pos = close + 1
					continue
				}
			} else {
				lineEnd := dependencyLineEnd(toks, pos)
				names = appendExportedSingleValueNames(names, toks, next, lineEnd)
			}
			pos = dependencySkipLine(toks, pos)
			continue
		}
		pos++
	}
	return names
}

func dependencyExportedFunctionNames(src []byte) []string {
	toks, err := scan.Tokens(src)
	if err != nil {
		return nil
	}
	var names []string
	pos := dependencyTopLevelStart(toks)
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		tok := toks[pos]
		if tok.Text != "func" {
			pos++
			continue
		}
		next := pos + 1
		if next < len(toks) && toks[next].Text == "(" {
			pos = dependencySkipFunc(toks, next)
			continue
		}
		if next < len(toks) && toks[next].Kind == scan.Ident && isExported(toks[next].Text) {
			names = appendStringUnique(names, toks[next].Text)
		}
		pos = dependencySkipFunc(toks, next)
	}
	return names
}

func dependencyExportedMethods(importPath string, src []byte) []methodInfo {
	toks, err := scan.Tokens(src)
	if err != nil {
		return nil
	}
	var methods []methodInfo
	pos := dependencyTopLevelStart(toks)
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		if toks[pos].Text != "func" || pos+1 >= len(toks) || toks[pos+1].Text != "(" {
			pos++
			continue
		}
		receiverOpen := pos + 1
		receiverClose := findClose(toks, receiverOpen, "(", ")")
		if receiverClose > receiverOpen && receiverClose+1 < len(toks) && toks[receiverClose+1].Kind == scan.Ident {
			nameTok := receiverClose + 1
			if isExported(toks[nameTok].Text) {
				var decl parse.Decl
				decl.Kind = "func"
				decl.Name = toks[nameTok].Text
				decl.Receiver = true
				decl.Start = int(toks[pos].Start)
				info := methodDeclInfoFromTokens(toks, &decl)
				if info.name != "" {
					info.unitName = SymbolName(importPath, info.name)
					info.importPath = importPath
					methods = append(methods, info)
				}
			}
		}
		pos = dependencySkipFunc(toks, pos+1)
	}
	return methods
}

func dependencyTopLevelStart(toks []scan.Token) int {
	pos := 0
	if len(toks) >= 2 && toks[0].Text == "package" {
		pos = 2
	}
	for pos < len(toks) && toks[pos].Text == "import" {
		pos = dependencySkipImport(toks, pos)
	}
	return pos
}

func dependencySkipImport(toks []scan.Token, pos int) int {
	next := pos + 1
	if next < len(toks) && toks[next].Text == "(" {
		close := findClose(toks, next, "(", ")")
		if close > next {
			return close + 1
		}
	}
	return dependencySkipLine(toks, pos)
}

func dependencySkipFunc(toks []scan.Token, pos int) int {
	for pos < len(toks) && toks[pos].Kind != scan.EOF {
		if toks[pos].Text == "{" {
			close := findClose(toks, pos, "{", "}")
			if close > pos {
				return close + 1
			}
		}
		pos++
	}
	return pos
}

func dependencySkipLine(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for pos < len(toks) && toks[pos].Line == line && toks[pos].Kind != scan.EOF {
		pos++
	}
	return pos
}

func dependencyLineEnd(toks []scan.Token, pos int) int {
	line := toks[pos].Line
	for pos < len(toks) && toks[pos].Line == line && toks[pos].Kind != scan.EOF {
		pos++
	}
	return pos
}

func appendExportedGroupedNames(names []string, toks []scan.Token, start int, end int) []string {
	if end <= start {
		return names
	}
	lineStart := true
	lastLine := -1
	for i := start; i < end; i++ {
		tok := toks[i]
		if lastLine != int(tok.Line) {
			lineStart = true
			lastLine = int(tok.Line)
		}
		if lineStart && tok.Kind == scan.Ident {
			if isExported(tok.Text) {
				names = appendStringUnique(names, tok.Text)
			}
			lineStart = false
			continue
		}
		if tok.Text == ";" {
			lineStart = true
		}
	}
	return names
}

func appendExportedSingleValueNames(names []string, toks []scan.Token, start int, end int) []string {
	expectName := true
	for i := start; i < end; i++ {
		tok := toks[i]
		if tok.Text == "=" {
			return names
		}
		if tok.Kind == scan.Ident && expectName {
			if isExported(tok.Text) {
				names = appendStringUnique(names, tok.Text)
			}
			expectName = false
			continue
		}
		if tok.Text == "," {
			expectName = true
			continue
		}
		if len(names) > 0 {
			return names
		}
	}
	return names
}

func appendStringUnique(values []string, value string) []string {
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func intrinsicImportSymbol(importPath string, name string) string {
	if importPath != "os" {
		return ""
	}
	switch name {
	case "Open":
		return "open"
	case "Close":
		return "close"
	case "Read":
		return "read"
	case "Write":
		return "write"
	case "Chmod":
		return "chmod"
	case "O_RDONLY":
		return "O_RDONLY"
	case "O_WRONLY":
		return "O_WRONLY"
	case "O_RDWR":
		return "O_RDWR"
	case "O_CREATE":
		return "O_CREATE"
	case "O_TRUNC":
		return "O_TRUNC"
	case "Stdin":
		return "0"
	case "Stdout":
		return "1"
	case "Stderr":
		return "2"
	}
	return ""
}

func intrinsicImportSymbolNames(importPath string) []string {
	if importPath != "os" {
		return nil
	}
	return []string{"Open", "Close", "Read", "Write", "Chmod", "O_RDONLY"}
}

func intrinsicImportFunctionSymbolNames(importPath string) []string {
	if importPath != "os" {
		return nil
	}
	return []string{"Open", "Close", "Read", "Write", "Chmod"}
}

func importLocalName(imp parse.Import) string {
	if imp.Alias != "" {
		return imp.Alias
	}
	return load.PackageNameFromImportPath(imp.Path)
}

func isExported(name string) bool {
	if name == "" {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}
