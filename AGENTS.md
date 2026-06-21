# Retargetable Go (RTG)

rtg is a minimal Golang compiler that compiles a small subset of the Golang language.

## Restrictions

- The only files that can be modified are `./compiler_*_impl.go` and `./main_rtg.go`.
- The only syscalls are `open`, `close`, `read`, `write`, and `print`.
- Performance requirements are strictly defined in `./main_test.go` and cannot be violated.
- Do not hardcode test cases, emit prebuilt/self-copying binaries, or patch the test harness/runtime instead of implementing the compiler.
