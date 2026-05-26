# 🏥 Go Doctor

AI-generated Go code quality evaluator. Assesses AI-generated code across 5 dimensions — Completeness, Idiomatic, Cleanliness, Implementation, Type Safety — plus 33 rules covering error handling, security, concurrency, and more.

Inspired by [react-doctor](https://github.com/millionco/react-doctor) and [goreporter](https://github.com/qax-os/goreporter).

## Features

- **5 AI Quality Dimensions** — Completeness / Idiomatic / Cleanliness / Implementation / Type Safety
- **33 Detection Rules** — Error handling, security, concurrency, performance, style, correctness, dead code
- **0-100 Scoring** — Category weighting + density normalization + logarithmic decay
- **MR/PR Review** — `--diff` mode scans only changed lines
- **Commit Review** — `--commit` mode scans only lines changed in a commit
- **JSON Output** — Structured reports for CI/CD integration

## Install

```bash
go install github.com/lizhiqiang-1996/go_doctor/cmd/go-doctor@latest
```

## Quick Start

```bash
# Evaluate code quality
go-doctor . --verbose

# Quick score
go-doctor . --score

# JSON report
go-doctor . --json

# MR/PR review (most common)
go-doctor . --diff master --verbose
```

## AI Quality Evaluation

Go Doctor evaluates AI-generated code quality across 5 dimensions:

| Dimension | Rule | What It Checks | Example |
|-----------|------|---------------|---------|
| **Completeness** | `completeness/placeholder-comment` | TODO/FIXME/HACK placeholders — incomplete implementation | `// TODO: implement this` |
| **Idiomatic** | `idiomatic/snake-case-naming` | snake_case naming — not idiomatic Go | `user_id` → `userID` |
| **Cleanliness** | `cleanliness/debug-print` | fmt.Println debug statements — leftover debug code | `fmt.Println("debug")` → `log.Printf` |
| **Implementation** | `implementation/empty-func-body` | Empty bodies / stubs — not truly implemented | `func Foo() { return nil }` |
| **Type Safety** | `type-safety/overly-broad-interface` | interface{}/any — insufficient type safety | `func Process(data any) any` |

These 5 rules are included in every scan by default.

## Usage

### Full Scan

```bash
go-doctor /path/to/project --verbose
```

Scans all Go files and reports all quality issues (including 5 AI quality dimensions).

### MR/PR Review

```bash
# Compare with main branch
go-doctor . --diff main --verbose

# Compare with remote branch
go-doctor . --diff origin/main --verbose
```

Scans only changed lines vs the target branch. Ideal for MR/PR review.

### Commit Review

```bash
# Latest commit
go-doctor . --commit HEAD --verbose

# Specific commit
go-doctor . --commit abc1234 --verbose
```

Scans only lines changed in the specified commit.

## CLI Options

```
go-doctor [directory] [options]

Options:
  --verbose         Show detailed issues for each rule
  --score           Output only the score
  --json            Output JSON format report
  --no-lint         Skip lint checks
  --no-dead-code    Skip dead code detection
  --diff [branch]   Scan only changed lines vs target branch (default: main)
  --commit <hash>   Scan only lines changed in the specified commit
  -v, --version     Show version
  -h, --help        Show help
```

## Scoring

### Score Levels

| Score | Level | Meaning |
|-------|-------|---------|
| 75-100 | Good | Code quality is acceptable |
| 50-74 | Needs Work | Issues should be fixed before merging |
| 0-49 | Critical | Must fix before committing |

### Algorithm

1. Each issue is weighted by category and severity
2. Penalty is normalized by file count (density normalization)
3. Logarithmic decay `math.Log1p(penalty) * 13` smooths the score

### Category Weights

| Category | Error Penalty | Warning Penalty |
|----------|-------------|----------------|
| Security | 5 | 2 |
| Error Handling | 5 | 2 |
| Concurrency | 4 | 1.5 |
| AI Quality | 3 | 1.5 |
| Correctness | 3 | 1 |
| Performance | 2 | 0.5 |
| Code Style | 1 | 0.2 |
| Dead Code | 1 | 0.1 |

## Detection Rules

### AI Quality (5 rules)

| Dimension | Rule | Severity | What It Checks |
|-----------|------|----------|---------------|
| Completeness | `completeness/placeholder-comment` | warning | TODO/FIXME/HACK placeholders |
| Idiomatic | `idiomatic/snake-case-naming` | warning | Non-idiomatic snake_case naming |
| Cleanliness | `cleanliness/debug-print` | warning | fmt.Println debug statements |
| Implementation | `implementation/empty-func-body` | warning | Empty function bodies / stubs |
| Type Safety | `type-safety/overly-broad-interface` | warning | Overly broad interface{}/any |

### Error Handling (3 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `error-handling/unchecked-error` | error | Function returns error but not checked |
| `error-handling/swallowed-error` | warning | Error assigned but not handled |
| `error-handling/panic-in-library` | warning | panic used in library code |

### Security (4 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `security/sql-injection` | error | SQL concatenation injection risk |
| `security/command-injection` | error | Command injection risk |
| `security/weak-crypto` | warning | Weak crypto algorithms (MD5/SHA1) |
| `security/hardcoded-credentials` | error | Hardcoded passwords/keys |

### Concurrency (4 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `concurrency/defer-in-loop` | warning | defer used inside a loop |
| `concurrency/range-var-capture` | warning | Range variable capture issue |
| `concurrency/missing-mutex-unlock` | error | Mutex Unlock not paired |
| `concurrency/goroutine-leak` | warning | Goroutine leak risk |

### Performance (3 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `performance/string-concat-in-loop` | warning | String concatenation in loop |
| `performance/unnecessary-conversion` | warning | Unnecessary type conversion |
| `performance/large-struct-copy` | warning | Large struct passed by value |

### Code Style (3 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `style/exported-without-comment` | warning | Exported identifier missing doc comment |
| `style/package-naming` | warning | Package name not following conventions |
| `style/function-complexity` | warning | Function cyclomatic complexity too high |

### Correctness (4 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `correctness/empty-interface` | warning | Empty interface{} lacks type safety |
| `correctness/unused-label` | warning | Unused label |
| `correctness/redundant-return` | warning | Redundant return statement |
| `correctness/error-check-without-handling` | warning | Error checked but not handled |

### Length & Complexity (4 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `complexity/function-depth` | warning | Function nesting too deep |
| `length/function-length` | warning | Function too long (>80 lines) |
| `length/file-length` | warning | File too long (>500 lines) |
| `length/line-length` | warning | Line too long (>120 chars) |

### Dead Code (3 rules)

| Rule | Severity | What It Checks |
|------|----------|---------------|
| `shadow/variable-shadow` | warning | Variable shadowing |
| `deadcode/unused-global-var` | warning | Unused global variable |
| `deadcode/unused-struct-field` | warning | Unused struct field |

## Configuration

Create `go-doctor.config.json` in the project root:

```json
{
  "ignore": {
    "rules": ["style/exported-without-comment"],
    "files": ["generated/**", "vendor/**"]
  },
  "lint": true,
  "deadCode": true,
  "verbose": false
}
```

## AI Tool Integration

### Skill

Copy `.trae/skills/go-doctor/` to the project root. AI coding assistants will automatically detect and use it.

### .cursorrules / CLAUDE.md

Copy the included `.cursorrules` or `CLAUDE.md` to the target project root. AI coding tools will automatically run quality checks after generating code.

## CI/CD Integration

### GitHub Actions

```yaml
name: Code Quality
on: [pull_request]
jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go install github.com/lizhiqiang-1996/go_doctor/cmd/go-doctor@latest
      - run: go-doctor . --verbose
      - run: |
          SCORE=$(go-doctor . --score | grep -oE '[0-9]+' | head -1)
          [ "$SCORE" -lt 50 ] && exit 1
```

### Pre-commit Hook

```bash
cp hooks/pre-commit .git/hooks/pre-commit
chmod +x .git/hooks/pre-commit
```

## Example Output

```
  ┌─────┐
  │ • • │   55 / 100 Needs Work
  │  ─  │   ████████████████░░░░░░░░░░░░░░
  └─────┘   Go Doctor

  Project: myproject
  Go Version: 1.21.6
  Framework: gin
  Source Files: 42

  AI Quality 15 issues
    ⚠ idiomatic/snake-case-naming ×6
        Variable 'user_id' uses snake_case — not idiomatic Go
        → Use camelCase in Go: rename 'user_id' to 'userID'
        main.go:10  main.go:22

    ⚠ type-safety/overly-broad-interface ×4
        Function parameter uses overly broad type (interface{}/any) — lacks type safety
        → Define a specific interface with required methods
        handler.go:15  handler.go:23

    ⚠ completeness/placeholder-comment ×3
        Placeholder comment detected: // TODO: implement this
        → Replace placeholder with actual implementation
        service.go:8  service.go:45

    ⚠ implementation/empty-func-body ×1
        Function 'Validate' has a minimal body — likely a stub
        → Implement the function with actual logic
        validator.go:12

    ⚠ cleanliness/debug-print ×1
        Debug print statement: fmt.Println()
        → Remove or replace with proper logging: log.Printf(...)
        main.go:42

  Error Handling 3 issues
    ✗ error-handling/unchecked-error ×3
        Function call returns an error but the error is not checked
        → Assign the error return value and check it
        main.go:42  main.go:67  handler.go:15

  Summary: 3 errors, 15 warnings across 8 files
```

## Project Structure

```
go-doctor/
├── cmd/
│   └── go-doctor/          # CLI entry point
├── pkg/
│   ├── analyzer/
│   │   ├── analyzer.go     # AST analysis engine
│   │   └── rules/          # Detection rules
│   │       ├── ai_code.go  # AI quality (5 dimensions)
│   │       ├── complexity.go
│   │       ├── concurrency.go
│   │       ├── correctness.go
│   │       ├── deadcode.go
│   │       ├── error_handling.go
│   │       ├── length.go
│   │       ├── performance.go
│   │       ├── registry.go
│   │       ├── security.go
│   │       ├── shadow.go
│   │       └── style.go
│   ├── config/             # Configuration loading
│   ├── git/                # Git diff/commit integration
│   ├── project/            # Project discovery (framework/version)
│   ├── reporter/           # Report output (terminal + JSON)
│   ├── scanner/            # Scan orchestrator
│   ├── scorer/             # Weighted scoring algorithm
│   └── types/              # Core type definitions
├── .trae/skills/           # Trae Skill
├── .github/workflows/      # CI/CD template
├── hooks/                  # Pre-commit hook
├── .cursorrules            # Cursor rules
└── CLAUDE.md               # Claude Code rules
```

## License

MIT
