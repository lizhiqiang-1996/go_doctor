---
name: "go-doctor"
description: "Evaluates AI-generated Go code quality. Invoke after writing/reviewing Go code, generating Go code with AI, checking MR quality, or when user asks for code quality check."
---

# Go Doctor - AI Code Quality Checker

Go Doctor evaluates the quality of AI-generated Go code across 5 dimensions (completeness, idiomatic, cleanliness, implementation, type-safety) and detects bugs, security issues, and performance problems.

## When to Invoke

- After generating Go code with AI
- Before committing Go code changes
- When reviewing a Go project's quality
- When user asks for code quality check, score, or review
- When user asks "check my code", "is this code good", "review this Go code", "check MR quality"

## Commands

### Diff Mode (most common — MR/PR quality check)
```bash
go-doctor <directory> --diff master --verbose
```
Scans only changed lines vs master branch. Use for every MR/PR review. This is the primary use case.

### Quality Scan
```bash
go-doctor <directory> --verbose
```
Scans all Go files. Reports issues across all categories including 5 AI quality dimensions.

### Quick Score
```bash
go-doctor <directory> --score
```
Outputs only the quality score (0-100). Use for quick checks.

### JSON Report
```bash
go-doctor <directory> --json
```
Structured JSON output for programmatic use.

### Commit Mode
```bash
go-doctor <directory> --commit <hash> --verbose
```
Scans only changed lines in a specific commit.

## Score Interpretation

| Score | Label | Action |
|-------|-------|--------|
| 75-100 | Good | Code quality is acceptable |
| 50-74 | Needs Work | Fix reported issues before merging |
| 0-49 | Critical | Must fix before committing |

## 5 AI Quality Dimensions

These dimensions are always included in every scan:

| Dimension | Rule ID | What It Checks | Example Bad → Good |
|-----------|---------|---------------|-------------------|
| **Completeness** | `completeness/placeholder-comment` | TODO/FIXME/HACK/XXX/PLACEHOLDER/STUB comments | `// TODO: implement` → write real code |
| **Idiomatic** | `idiomatic/snake-case-naming` | Non-idiomatic snake_case naming | `user_id` → `userID` |
| **Cleanliness** | `cleanliness/debug-print` | fmt.Println/Printf/Print in production code | `fmt.Println(x)` → `log.Printf("%v", x)` |
| **Implementation** | `implementation/empty-func-body` | Empty bodies, `return nil`, `panic("not implemented")` | `func Foo() { return nil }` → real implementation |
| **Type Safety** | `type-safety/overly-broad-interface` | interface{}/any as parameter or return type | `func Process(data any) any` → `func Process(data *Request) *Response` |

## Other Rule Categories

| Category | Rules | Key Issues |
|----------|-------|-----------|
| Error Handling | 3 | Unchecked errors, swallowed errors, panic in library |
| Security | 4 | SQL injection, command injection, weak crypto, hardcoded credentials |
| Concurrency | 4 | Defer in loop, range var capture, missing mutex unlock, goroutine leak |
| Performance | 3 | String concat in loop, unnecessary conversion, large struct copy |
| Code Style | 3 | Missing comments, package naming, function complexity |
| Correctness | 4 | Empty interface, unused label, redundant return, error check without handling |
| Length/Complexity | 4 | Function depth, function length, file length, line length |
| Dead Code | 3 | Variable shadow, unused global var, unused struct field |

## Workflow

### After generating Go code
1. Run `go-doctor . --verbose` to evaluate code quality
2. Fix all reported issues, prioritizing:
   - **P0**: Security (SQL/command injection), unchecked errors
   - **P1**: AI quality dimensions (completeness, idiomatic, cleanliness, implementation, type safety)
   - **P2**: Code style, performance, dead code
3. Re-run `go-doctor . --score` to verify score >= 75

### Before merging MR/PR
1. Run `go-doctor . --diff master --verbose` to check only changed lines
2. Fix all P0 and P1 issues
3. Ensure score >= 50 before merging

## Installation

```bash
go install github.com/lizhiqiang-1996/go_doctor/cmd/go-doctor@latest
```

Or clone and build:

```bash
git clone git@github.com:lizhiqiang-1996/go_doctor.git
cd go_doctor
go build -o go-doctor ./cmd/go-doctor/
```

## Repository

https://github.com/lizhiqiang-1996/go_doctor
