# AGENTS.md

Guidelines for AI agents and contributors working on pssg.

## Hard Rules

1. **Zero external dependencies.** pssg uses only the Go standard library. Do not add any `require` directives to `go.mod`. This is enforced by a unit test (`TestZeroDependencies`). YAML parsing and markdown rendering are implemented internally.

2. **Test-first.** Write failing tests before implementing features. All new code must have unit tests.

3. **No generated code.** No code generators, no protobuf, no build-time codegen.

## Design Goals

- Single binary, zero config defaults, fast builds
- Every feature earns its complexity — if stdlib can do it, use stdlib
- Backward-compatible changes only; existing sites must not break

## Architecture

The build pipeline flows: `loader -> entity -> taxonomy -> render -> output`. Each package has a single responsibility. The `build` package orchestrates the pipeline.

Templates use Go's `html/template`. Config uses YAML (parsed by `internal/yaml`). Content uses markdown with YAML frontmatter.

## Implementation Order

When adding features:
1. Add config types to `internal/config/types.go`
2. Write tests for the new behavior
3. Implement in the appropriate package
4. Wire into `internal/build/build.go`
5. Run `go test ./...` — all tests must pass
