# Incremental object-backend bring-up

The `bringup` package makes the boundary between a trusted C/assembly shell and
a candidate RTG backend explicit. `StandardStages` assigns stable milestones
and exported omnibus roots from a single C-ABI constant-return function through
a complete backend-owned image.

Each target records its C compiler, linker, object format, ABI, trusted shell,
and optional runner. `ValidateELFObject` rejects a candidate before linking when
its ELF class, byte order, machine, relocatable-object type, section alignment,
exports, undefined symbols, relocation-section links, or relocation kinds do
not satisfy the current milestone contract. Trusted shell imports and supported
relocations must be allowlisted; hosted symbols are never accepted implicitly.

This establishes the host-side object contract and malformed-artifact tests for
#22. Building both objects from the canonical omnibus unit, invoking the target
linker/simulator, and comparing result blocks are the next pipeline layers once
the C and candidate emitters from #19 are available.
