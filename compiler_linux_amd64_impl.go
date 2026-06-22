package main

func compileLinuxAmd64(input []int, output int) int {
	rtgCompilerDiag = 0
	var src []byte
	for i := 0; i < len(input); i++ {
		src = rtgReadAll(input[i], src)
		src = append(src, '\n')
	}
	var prog rtgProgram
	prog = rtgParseProgram(src)
	if !prog.ok {
		rtgPrintCompilerDiagnostic(rtgCompilerDiag)
		return 1
	}
	var meta rtgMeta
	meta = rtgBuildMeta(&prog)
	if !meta.ok {
		rtgPrintCompilerDiagnostic(rtgCompilerDiag)
		return 1
	}
	var result rtgCompileResult
	result = rtgTryCompileScalarProgram(&prog, &meta)
	if result.ok {
		write(output, result.data, 0)
		return 0
	}
	rtgPrintCompilerDiagnostic(rtgCompilerDiag)
	return 1
}

func rtgTryCompileScalarProgram(p *rtgProgram, meta *rtgMeta) rtgCompileResult {
	appIndex := -1
	mainFound := false
	for i := 0; i < len(meta.funcs); i++ {
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "appMain") {
			appIndex = i
		}
		if rtgBytesEqualText(meta.prog.src, meta.funcs[i].nameStart, meta.funcs[i].nameEnd, "main") {
			mainFound = true
		}
	}
	if appIndex < 0 {
		if mainFound {
			rtgSetCompilerDiag(rtgDiagMainRequiresAppMain)
		} else {
			rtgSetCompilerDiag(rtgDiagAppMainRequired)
		}
		var result rtgCompileResult
		return result
	}
	var g rtgLinearGen
	g.prog = p
	g.meta = meta
	a := &g.asm
	rtgAsmInit(a)
	for i := 0; i < len(meta.funcs); i++ {
		g.funcLabels = append(g.funcLabels, rtgAsmNewLabel(a))
	}
	if !rtgLinearInitGlobals(&g) {
		rtgSetCompilerDiag(rtgDiagGlobalCodegen)
		var result rtgCompileResult
		return result
	}
	if !rtgEmitProgramEntryArgs(&g, appIndex) {
		rtgSetCompilerDiag(rtgDiagAppMainSignature)
		var result rtgCompileResult
		return result
	}
	rtgAsmCallLabel(a, g.funcLabels[appIndex])
	rtgAsmMovRdiRax(a)
	rtgAsmMovRaxImm(a, 60)
	rtgAsmSyscall(a)
	for i := 0; i < len(meta.funcs); i++ {
		if !rtgEmitScalarFunction(&g, i) {
			rtgSetCompilerDiag(rtgDiagFunctionCodegen)
			var result rtgCompileResult
			return result
		}
	}
	data := rtgAsmImage(a)
	var result rtgCompileResult
	result.data = data
	result.ok = true
	return result
}

func rtgEmitProgramEntryArgs(g *rtgLinearGen, appIndex int) bool {
	app := &g.meta.funcs[appIndex]
	if app.resultType != 0 && !rtgTypeIsInt(g.meta, app.resultType) {
		rtgSetCompilerDiag(rtgDiagAppMainSignature)
		return false
	}
	argsOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envDataOff := g.asm.bssSize
	g.asm.bssSize += 32768
	envLenOff := g.asm.bssSize
	g.asm.bssSize += 8
	rtgAsmBuildArgvEnvSlices(&g.asm, argsOff, envDataOff, envLenOff)
	if app.paramCount == 0 {
		return true
	}
	if app.paramCount > 2 {
		rtgSetCompilerDiag(rtgDiagAppMainSignature)
		return false
	}
	first := &g.meta.params[app.firstParam]
	if !rtgTypeIsStringSlice(g.meta, first.typ) {
		rtgSetCompilerDiag(rtgDiagAppMainSignature)
		return false
	}
	if app.paramCount == 1 {
		return true
	}
	second := &g.meta.params[app.firstParam+1]
	if !rtgTypeIsStringSlice(g.meta, second.typ) {
		rtgSetCompilerDiag(rtgDiagAppMainSignature)
		return false
	}
	return true
}

func rtgAsmBuildArgvEnvSlices(a *rtgAsm, bssOff int, envOff int, envLenOff int) {
	loopLabel := rtgAsmNewLabel(a)
	strlenLabel := rtgAsmNewLabel(a)
	afterLenLabel := rtgAsmNewLabel(a)
	doneLabel := rtgAsmNewLabel(a)
	envScanLabel := rtgAsmNewLabel(a)
	envStartLabel := rtgAsmNewLabel(a)
	envLoopLabel := rtgAsmNewLabel(a)
	envStrlenLabel := rtgAsmNewLabel(a)
	envAfterLenLabel := rtgAsmNewLabel(a)
	envDoneLabel := rtgAsmNewLabel(a)
	rtgAsmEmit32(a, 604277576)
	rtgAsmEmit24(a, 12618057)
	rtgAsmEmit32(a, 608996684)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMovR10BssAddr(a, bssOff)
	rtgAsmEmit32(a, 1305774413)
	rtgAsmEmit16(a, 56113)
	rtgAsmMarkLabel(a, loopLabel)
	rtgAsmEmit24(a, 12794189)
	rtgAsmEmit16(a, 36111)
	at := len(a.code)
	rtgAsmEmit32(a, 0)
	rtgAsmAddReloc(a, at, doneLabel, rtgRel32)
	rtgAsmEmit32(a, -650343605)
	rtgAsmEmit32(a, 1211795785)
	rtgAsmEmit16(a, 49201)
	rtgAsmMarkLabel(a, strlenLabel)
	rtgAsmEmit32(a, 474240)
	rtgAsmJzLabel(a, afterLenLabel)
	rtgAsmEmit24(a, 12648264)
	rtgAsmJmpLabel(a, strlenLabel)
	rtgAsmMarkLabel(a, afterLenLabel)
	rtgAsmEmit32(a, 138578249)
	rtgAsmEmit32(a, 281183049)
	rtgAsmEmit24(a, 12844873)
	rtgAsmJmpLabel(a, loopLabel)
	rtgAsmMarkLabel(a, doneLabel)

	rtgAsmEmit32(a, 608996684)
	rtgAsmEmit8(a, 0x08)
	rtgAsmMarkLabel(a, envScanLabel)
	rtgAsmEmit32(a, 3769161)
	rtgAsmJzLabel(a, envStartLabel)
	rtgAsmEmit32(a, 146899785)
	rtgAsmJmpLabel(a, envScanLabel)
	rtgAsmMarkLabel(a, envStartLabel)
	rtgAsmEmit32(a, 146899785)
	rtgAsmMovR10BssAddr(a, envOff)
	rtgAsmEmit24(a, 14365005)
	rtgAsmMarkLabel(a, envLoopLabel)
	rtgAsmEmit32(a, -650343605)
	rtgAsmEmit32(a, 16745288)
	rtgAsmJzLabel(a, envDoneLabel)
	rtgAsmEmit32(a, 1211795785)
	rtgAsmEmit16(a, 49201)
	rtgAsmMarkLabel(a, envStrlenLabel)
	rtgAsmEmit32(a, 474240)
	rtgAsmJzLabel(a, envAfterLenLabel)
	rtgAsmEmit24(a, 12648264)
	rtgAsmJmpLabel(a, envStrlenLabel)
	rtgAsmMarkLabel(a, envAfterLenLabel)
	rtgAsmEmit32(a, 138578249)
	rtgAsmEmit32(a, 281183049)
	rtgAsmEmit24(a, 12844873)
	rtgAsmJmpLabel(a, envLoopLabel)
	rtgAsmMarkLabel(a, envDoneLabel)
	rtgAsmEmit24(a, 14190924)
	rtgAsmStoreRaxBss(a, envLenOff)

	rtgAsmEmit32(a, 1290242380)
	rtgAsmEmit32(a, -1991457143)
	rtgAsmEmit8(a, 0xc2)
	rtgAsmMovRaxBssAddr(a, envOff)
	rtgAsmMovRcxRax(a)
	rtgAsmLoadRaxBss(a, envLenOff)
	rtgAsmMovR8Rax(a)
	rtgAsmMovR9Rax(a)
}

func rtgAsmImage(a *rtgAsm) []byte {
	rtgAsmPatch(a)
	fileSize := a.codeOffset + len(a.code) + len(a.data)
	memSize := fileSize + a.bssSize
	var out []byte
	out = rtgAppendElfHeader(out, a.codeOffset, fileSize, memSize)
	for i := 0; i < len(a.code); i++ {
		out = append(out, a.code[i])
	}
	for i := 0; i < len(a.data); i++ {
		out = append(out, a.data[i])
	}
	return out
}

func rtgAppendElfHeader(out []byte, entryOff int, fileSize int, memSize int) []byte {
	base := 0x400000

	out = append(out, 0x7f)
	out = append(out, 'E')
	out = append(out, 'L')
	out = append(out, 'F')
	out = append(out, 2)
	out = append(out, 1)
	out = append(out, 1)
	out = append(out, 0)
	for i := 0; i < 8; i++ {
		out = append(out, 0)
	}
	out = rtgAppend16(out, 2)
	out = rtgAppend16(out, 0x3e)
	out = rtgAppend32(out, 1)
	out = rtgAppend64(out, base+entryOff)
	out = rtgAppend64(out, 64)
	out = rtgAppend64(out, 0)
	out = rtgAppend32(out, 0)
	out = rtgAppend16(out, 64)
	out = rtgAppend16(out, 56)
	out = rtgAppend16(out, 1)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)
	out = rtgAppend16(out, 0)

	out = rtgAppend32(out, 1)
	out = rtgAppend32(out, 7)
	out = rtgAppend64(out, 0)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, base)
	out = rtgAppend64(out, fileSize)
	out = rtgAppend64(out, memSize)
	out = rtgAppend64(out, 0x1000)
	return out
}
