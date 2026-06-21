# RTG Go Subset

This document describes the intended source language for RTG tests and the
self-hosting compiler. It is deliberately much smaller than Go. Anything not
listed here should be treated as unsupported until this file is updated.

## Program Shape

- A program is one or more files in package `main`.
- Tests define `func appMain(args []string) int`.
- The runtime provides `main`, calls `appMain(os.Args)`, and exits with its
  return value.
- Source files may contain top-level `const`, `var`, `type`, and `func`
  declarations.
- Imports are not part of the compiled subset. Runtime functions and constants
  are provided by `rtg_main.go`.

## Types

Required:

- `int`
- `int64`
- `byte`
- `bool`
- `string`
- floating point types needed by tests and compiler source
- slices of supported element types, especially `[]byte`, `[]int`, and
  `[]string`
- structs
- pointers to supported types
- named aliases or definitions of supported types

Unsupported:

- complex numbers
- interfaces
- maps
- arrays as distinct fixed-length values
- channels
- function values and closures
- methods
- generics

## Literals

Required:

- integer literals in decimal form
- integer literals in hexadecimal and binary form
- floating point literals
- character literals for byte-sized values, for example `'a'` and `'\n'`
- interpreted string literals with common escapes such as `\n`, `\"`, and `\\`
- boolean literals `true` and `false`
- composite literals for structs and supported aggregate values

Unsupported:

- raw string literals
- octal or imaginary literals

## Expressions

Required:

- identifiers
- integer, floating point, string, byte, and bool literals
- parenthesized expressions
- unary `+`, `-`, and `!`
- arithmetic `+`, `-`, `*`, `/`, `%`
- comparisons `==`, `!=`, `<`, `<=`, `>`, `>=`
- boolean `&&` and `||` with Go short-circuit behavior
- bitwise `&`, `|`, `^`, and `&^`
- shifts `<<` and `>>`
- address-of `&x`
- dereference `*p`
- struct field selection, for example `x.y` and `p.y`
- string indexing, for example `s[i]`
- slice indexing and assignment, for example `buf[i] = 65`
- slice length with `len(x)`
- function calls
- slice append with `append(slice, value)`
- conversions between supported integer-like types where needed, especially
  `byte(x)` and `int(x)`
- conversion from `string` to `[]byte`

String concatenation with `+` is optional; tests should avoid requiring it
unless the compiler source needs it.

Unsupported:

- slicing expressions `x[a:b]`
- `cap`
- variadic calls other than `append`
- type assertions and type switches

## Statements

Required:

- `var` declarations with explicit type, initializer, or both
- short variable declarations `:=`
- assignment `=`
- compound assignment for arithmetic operators: `+=`, `-=`, `*=`, `/=`, `%=`
- expression statements for function calls and append assignments
- `return` with zero or one value according to function result type
- `if`, `else if`, and `else`
- `for` loops in Go's three common forms:
  - `for condition { ... }`
  - `for init; condition; post { ... }`
- `for { ... }`
- `break` and `continue`
- labels and `goto`
- increment and decrement as expressions; `i++` and `i--` as statements are
  optional unless required by compiler source

Unsupported:

- `defer`
- `go`
- `select`
- `switch`
- `range`
- multiple assignment

## Functions

Required:

- named top-level functions
- zero or more parameters
- zero or one return value
- recursion
- calls before declarations

Unsupported:

- multiple return values
- named return values
- anonymous functions
- methods

## Runtime API

Compiled programs may call only these runtime-provided operations:

- `open(path string, flags int) int`
- `close(fd int) int`
- `read(fd int, buf []byte, off int64) int`
- `write(fd int, buf []byte, off int64) int`
- `chmod(fd int, mode int) int`
- `print(s string)`

Compiled programs may use these runtime constants:

- `O_RDONLY`
- `O_WRONLY`
- `O_RDWR`
- `O_CREATE`
- `O_TRUNC`

## Anti-Cheat Test Guidelines

- Tests should compare compiled behavior against host Go through the existing
  harness.
- Each test file must print exactly `PASS\n` on success.
- A failing test should print a distinct diagnostic and return a non-zero exit
  code.
- Tests should vary source spelling, ordering, whitespace, constants, control
  flow, helper functions, and data values so a compiler cannot pass by matching
  known test text.
- Tests should avoid relying on unsupported language features unless this file
  is first updated to include them.
