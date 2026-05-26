package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/reporter"
	"github.com/lizhiqiang-1996/go_doctor/pkg/scanner"
	"github.com/lizhiqiang-1996/go_doctor/pkg/scorer"
	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

const version = "0.2.0"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-v", "--version":
			fmt.Printf("go-doctor %s\n", version)
			os.Exit(0)
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		}
	}

	rootDir := "."
	verbose := false
	jsonMode := false
	scoreOnly := false
	noLint := false
	noDeadCode := false
	diffBase := ""
	commitHash := ""

	args := os.Args[1:]
	i := 0
	for i < len(args) {
		arg := args[i]
		switch arg {
		case "--verbose":
			verbose = true
		case "--json":
			jsonMode = true
		case "--score":
			scoreOnly = true
		case "--no-lint":
			noLint = true
		case "--no-dead-code":
			noDeadCode = true
		case "--diff":
			i++
			if i < len(args) && !strings.HasPrefix(args[i], "-") {
				diffBase = args[i]
			} else {
				diffBase = "main"
				i--
			}
		case "--commit":
			i++
			if i < len(args) && !strings.HasPrefix(args[i], "-") {
				commitHash = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "Error: --commit requires a commit hash\n")
				os.Exit(1)
			}
		case "-v", "--version":
			fmt.Printf("go-doctor %s\n", version)
			os.Exit(0)
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		default:
			if !strings.HasPrefix(arg, "-") {
				rootDir = arg
			}
		}
		i++
	}

	if diffBase != "" && commitHash != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot use --diff and --commit together\n")
		os.Exit(1)
	}

	absRootDir, err := filepath.Abs(rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving directory: %v\n", err)
		os.Exit(1)
	}

	options := types.ScanOptions{
		Lint:      !noLint,
		DeadCode:  !noDeadCode,
		Verbose:   verbose,
		ScoreOnly: scoreOnly,
		JSON:      jsonMode,
		DiffBase:  diffBase,
		Commit:    commitHash,
	}

	s := scanner.New(absRootDir, options)
	result := s.Scan()

	r := reporter.New(verbose, jsonMode, absRootDir)
	r.Print(result)

	if result.Score != nil && result.Score.Score < scorer.ScoreOKThreshold {
		os.Exit(1)
	}

	errorCount := 0
	for _, d := range result.Diagnostics {
		if d.Severity == types.SeverityError {
			errorCount++
		}
	}
	if errorCount > 0 {
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Printf(`go-doctor - Diagnose Go codebase health

Usage:
  go-doctor [directory] [options]

Arguments:
  directory         Project directory to scan (default: current directory)

Options:
  --verbose         Show file details per rule
  --score           Output only the score
  --json            Output structured JSON report
  --no-lint         Skip lint checks
  --no-dead-code    Skip dead code detection
  --diff [branch]   Scan only files changed vs base branch (default: main)
  --commit <hash>   Scan only files changed in a specific commit
  -v, --version     Display version number
  -h, --help        Display help information

Diff Mode (--diff):
  Scan only files that differ from a base branch. Useful for MR/PR code review.
  If no branch is specified, defaults to 'main'.

  Examples:
    go-doctor . --diff              Compare against main branch
    go-doctor . --diff main         Compare against main branch
    go-doctor . --diff master       Compare against master branch
    go-doctor . --diff origin/main  Compare against remote main

Commit Mode (--commit):
  Scan only files changed in a specific commit. Useful for reviewing individual changes.

  Examples:
    go-doctor . --commit abc1234    Scan files in commit abc1234
    go-doctor . --commit HEAD       Scan files in the latest commit
    go-doctor . --commit HEAD~1     Scan files in the previous commit

Configuration:
  Create a go-doctor.config.json in your project root:

  {
    "ignore": {
      "rules": ["style/exported-without-comment"],
      "files": ["generated/**"]
    },
    "lint": true,
    "deadCode": true,
    "verbose": false
  }

Examples:
  go-doctor .                    Scan current directory
  go-doctor ./myproject          Scan specific project
  go-doctor . --verbose          Show detailed diagnostics
  go-doctor . --json             Output JSON report
  go-doctor . --score            Output only the score
  go-doctor . --diff main        MR review: scan changed files vs main
  go-doctor . --commit HEAD      Commit review: scan latest commit
`)
}
