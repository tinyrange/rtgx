package main

func rtgCompilerError(msg string) {
	write(2, []byte("rtgx: error: "), -1)
	write(2, []byte(msg), -1)
	write(2, []byte("\n"), -1)
}

func rtgCompilerErrorValue(msg string, value string) {
	write(2, []byte("rtgx: error: "), -1)
	write(2, []byte(msg), -1)
	write(2, []byte(value), -1)
	write(2, []byte("\n"), -1)
}

func rtgPrintCompilerDiagnostic(diag int) {
	if diag == rtgDiagParseMissingPackage {
		rtgCompilerError("expected a package declaration at the start of the file")
		return
	}
	if diag == rtgDiagParseMissingPackageName {
		rtgCompilerError("expected a package name after package")
		return
	}
	if diag == rtgDiagParsePackageName {
		rtgCompilerError("expected a package name after package")
		return
	}
	if diag == rtgDiagParseGroupedDecl {
		rtgCompilerError("expected ')' to close the grouped declaration")
		return
	}
	if diag == rtgDiagParseTopDecl {
		rtgCompilerError("could not read the top-level declaration")
		return
	}
	if diag == rtgDiagParseFuncDecl {
		rtgCompilerError("could not read the function declaration")
		return
	}
	if diag == rtgDiagParseStatement {
		rtgCompilerError("could not read one of the statements")
		return
	}
	if diag == rtgDiagParseExpression {
		rtgCompilerError("could not read one of the expressions")
		return
	}
	if diag == rtgDiagParseComposite {
		rtgCompilerError("could not read the composite literal")
		return
	}
	if diag == rtgDiagParseCall {
		rtgCompilerError("could not read the function call")
		return
	}
	if diag == rtgDiagParseIndex {
		rtgCompilerError("could not read the index or slice expression")
		return
	}
	if diag == rtgDiagParseParen {
		rtgCompilerError("expected ')' to close the expression")
		return
	}
	if diag == rtgDiagMetaConstDecl {
		rtgCompilerError("could not read the const declaration")
		return
	}
	if diag == rtgDiagMetaTopDecl {
		rtgCompilerError("could not read the top-level declaration type")
		return
	}
	if diag == rtgDiagMetaFuncDecl {
		rtgCompilerError("could not read the function signature")
		return
	}
	if diag == rtgDiagMetaResultType {
		rtgCompilerError("could not read the function result type")
		return
	}
	if diag == rtgDiagMetaParamList {
		rtgCompilerError("could not read the function parameter list")
		return
	}
	if diag == rtgDiagAppMainRequired {
		rtgCompilerError("no appMain entrypoint was found")
		return
	}
	if diag == rtgDiagMainRequiresAppMain {
		rtgCompilerError("found main, but rtgx programs use appMain as the entrypoint")
		return
	}
	if diag == rtgDiagAppMainSignature {
		rtgCompilerError("supported appMain forms are appMain(), appMain(args []string), and appMain(args []string, env []string), with an optional int result")
		return
	}
	if diag == rtgDiagGlobalCodegen {
		rtgCompilerError("could not compile one of the global initializers")
		return
	}
	if diag == rtgDiagFunctionCodegen {
		rtgCompilerError("could not compile one of the function bodies")
		return
	}
	if diag == rtgDiagCompileFailed {
		rtgCompilerError("could not compile this program")
		return
	}
	if diag == rtgDiagFunctionParams {
		rtgCompilerError("could not prepare the function parameters")
		return
	}
	if diag == rtgDiagStatementCodegen {
		rtgCompilerError("could not compile one of the statements")
		return
	}
	if diag == rtgDiagAssignmentCodegen {
		rtgCompilerError("could not compile the assignment")
		return
	}
	if diag == rtgDiagReturnCodegen {
		rtgCompilerError("could not compile the return value")
		return
	}
	if diag == rtgDiagConditionCodegen {
		rtgCompilerError("could not compile the condition expression")
		return
	}
	if diag == rtgDiagSwitchCodegen {
		rtgCompilerError("could not compile the switch statement")
		return
	}
	if diag == rtgDiagCallCodegen {
		rtgCompilerError("could not compile the function call")
		return
	}
	if diag == rtgDiagBreakOutsideLoop {
		rtgCompilerError("break is only supported inside for or switch")
		return
	}
	if diag == rtgDiagContinueOutsideLoop {
		rtgCompilerError("continue is only supported inside for")
		return
	}
	if diag == rtgDiagUnsupportedStatement {
		rtgCompilerError("this statement form is not supported yet")
		return
	}
	rtgCompilerError("could not compile this program")
}

func rtgOpenInputFallback(path string, env []string) int {
	if len(path) == 0 || path[0] == '/' {
		return -1
	}
	for e := 0; e < len(env); e++ {
		pwd := env[e]
		if len(pwd) > 4 {
			if pwd[0] == 'P' && pwd[1] == 'W' && pwd[2] == 'D' && pwd[3] == '=' {
				var full []byte
				for i := 4; i < len(pwd); i++ {
					full = append(full, pwd[i])
				}
				full = append(full, '/')
				for i := 0; i < len(path); i++ {
					full = append(full, path[i])
				}
				full = append(full, 0)
				return open(string(full), O_RDONLY)
			}
		}
	}
	return -1
}

func appMain(args []string, env []string) int {
	var input []int
	if len(args) < 4 {
		rtgCompilerError("usage: rtgx -o <output> <input.go> [input.go...]")
		return 1
	}
	if args[1] != "-o" {
		rtgCompilerError("usage: rtgx -o <output> <input.go> [input.go...]")
		return 1
	}
	var output int = open(args[2], O_RDWR|O_CREATE|O_TRUNC)
	if output < 0 {
		rtgCompilerErrorValue("failed to open output file: ", args[2])
		return 1
	}
	for i := 3; i < len(args); i++ {
		fd := open(args[i], O_RDONLY)
		if fd < 0 {
			fd = rtgOpenInputFallback(args[i], env)
		}
		if fd < 0 {
			rtgCompilerErrorValue("failed to open input file: ", args[i])
			return 1
		}
		input = append(input, fd)
	}
	if compileLinuxAmd64(input, output) != 0 {
		return 1
	}

	return 0
}
