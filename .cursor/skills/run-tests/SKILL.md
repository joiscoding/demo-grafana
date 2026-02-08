---
name: run-tests
description: Run frontend and backend unit tests for the Grafana application. Use when the user asks to run tests, check tests, verify changes with tests, run unit tests, run Jest tests, run Go tests, or wants to validate code changes.
---

# Run Unit Tests

## Quick reference

| What | Frontend | Backend |
|------|----------|---------|
| Framework | Jest + Testing Library | Go testing + testify |
| Run all | `yarn test` | `make test-go-unit` |
| Run specific file | `yarn jest <path>` | `go test -v -short ./<pkg>/...` |
| Test file pattern | `*.test.ts` / `*.test.tsx` | `*_test.go` |
| Config | `jest.config.js` | Standard Go conventions |

## Deciding which tests to run

1. **Identify changed files** using `git diff --name-only` or context from the conversation.
2. **Frontend changes** (files under `public/`, `packages/`, or `.ts`/`.tsx` files) — run frontend tests.
3. **Backend changes** (files under `pkg/`, or `.go` files) — run backend tests.
4. If unsure, ask the user or run both.

## Frontend tests

### Run all frontend tests

```sh
yarn test
```

This starts Jest in watch mode. For a single non-interactive run, use:

```sh
yarn test --watchAll=false
```

### Run tests for a specific file or directory

```sh
yarn jest public/app/path/to/file.test.tsx
```

Or match by pattern:

```sh
yarn jest --testPathPattern="ComponentName"
```

### Run tests with coverage

```sh
yarn test:coverage
```

### Environment variables

| Variable | Purpose |
|----------|---------|
| `TEST_MAX_WORKERS` | Number of Jest workers (default: all CPUs) |
| `TEST_SHARD` / `TEST_SHARD_TOTAL` | Sharding for CI |

### Frontend test conventions

- Test files are co-located with source: `Component.tsx` → `Component.test.tsx`
- Uses `@testing-library/react` for component rendering
- Uses `userEvent` (async) for simulating interactions
- Test regex: `(\\.|/)(test)\\.(jsx?|tsx?)$`

## Backend tests

### Run all backend unit tests

```sh
make test-go-unit
```

Or directly with Go:

```sh
go test -v -short ./pkg/...
```

The `-short` flag skips integration tests and runs only unit tests.

### Run tests for a specific package

```sh
go test -v -short ./pkg/services/ngalert/state/...
```

### Run a specific test function

```sh
go test -v -short -run "TestFunctionName" ./pkg/path/to/package/...
```

### Run with race detection

```sh
GO_RACE=true make test-go-unit
```

Or create a `.go-race-enabled-locally` file in the repo root.

### Environment variables

| Variable | Purpose |
|----------|---------|
| `GO_RACE` | Enable race detector |
| `GO_TEST_FLAGS` | Extra flags passed to `go test` |
| `GO_BUILD_TAGS` | Build tags (e.g., `enterprise,sqlite`) |
| `CGO_ENABLED` | Enable/disable CGO (`0` or `1`) |

### Backend test conventions

- Test files live in the same package: `strings.go` → `strings_test.go`
- Unit test functions: `TestXxx(t *testing.T)`
- Integration test functions: `TestIntegrationXxx(t *testing.T)`
- Use `testify/assert` and `testify/require` for assertions
- Table-driven tests are preferred

## Workflow

When asked to run tests, follow these steps:

1. **Determine scope** — identify which files changed and whether they are frontend or backend.
2. **Run targeted tests first** — run only the tests related to changed code for faster feedback.
3. **Report results** — summarize pass/fail counts and highlight any failures.
4. **On failure** — read the failing test output, identify the root cause, and suggest or apply a fix.
5. **Re-run after fix** — confirm the fix by running the failing tests again.

## Run everything

To run both frontend and backend tests:

```sh
make test
```

This calls `make test-go` and `make test-js` together.
