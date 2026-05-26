package analyzer

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-doctor/go-doctor/pkg/analyzer/rules"
	"github.com/go-doctor/go-doctor/pkg/types"
)

type Analyzer struct {
	fset        *token.FileSet
	rules       []rules.Rule
	ignoreRules map[string]bool
	ignoreFiles []string
}

func NewAnalyzer(ignoreRules []string, ignoreFiles []string) *Analyzer {
	ignoreMap := make(map[string]bool)
	for _, r := range ignoreRules {
		ignoreMap[r] = true
	}

	a := &Analyzer{
		fset:        token.NewFileSet(),
		rules:       rules.AllRules(),
		ignoreRules: ignoreMap,
		ignoreFiles: ignoreFiles,
	}

	return a
}

func (a *Analyzer) AnalyzeDirectory(rootDir string) []types.Diagnostic {
	var allDiagnostics []types.Diagnostic

	filePaths := a.collectGoFiles(rootDir)

	for _, filePath := range filePaths {
		if a.shouldIgnoreFile(filePath, rootDir) {
			continue
		}

		diagnostics := a.analyzeFile(filePath, rootDir)
		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	return a.filterDiagnostics(allDiagnostics)
}

func (a *Analyzer) AnalyzeFiles(rootDir string, filePaths []string) []types.Diagnostic {
	var allDiagnostics []types.Diagnostic

	for _, filePath := range filePaths {
		absPath := filePath
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(rootDir, filePath)
		}

		if a.shouldIgnoreFile(absPath, rootDir) {
			continue
		}

		diagnostics := a.analyzeFile(absPath, rootDir)
		allDiagnostics = append(allDiagnostics, diagnostics...)
	}

	return a.filterDiagnostics(allDiagnostics)
}

func (a *Analyzer) collectGoFiles(rootDir string) []string {
	var files []string

	filepath.Walk(rootDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			name := fi.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})

	return files
}

func (a *Analyzer) shouldIgnoreFile(filePath string, rootDir string) bool {
	relPath, err := filepath.Rel(rootDir, filePath)
	if err != nil {
		return false
	}

	for _, pattern := range a.ignoreFiles {
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}
		if strings.Contains(relPath, strings.ReplaceAll(pattern, "**", "")) {
			if strings.HasSuffix(pattern, "/**") {
				prefix := strings.TrimSuffix(pattern, "**")
				if strings.HasPrefix(relPath, prefix) {
					return true
				}
			}
		}
	}

	return false
}

func (a *Analyzer) analyzeFile(filePath string, rootDir string) []types.Diagnostic {
	file, err := parser.ParseFile(a.fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil
	}

	var diagnostics []types.Diagnostic

	for _, rule := range a.rules {
		if a.ignoreRules[rule.ID()] {
			continue
		}

		results := rule.Check(file, a.fset, filePath, rootDir)
		diagnostics = append(diagnostics, results...)
	}

	return diagnostics
}

func (a *Analyzer) filterDiagnostics(diagnostics []types.Diagnostic) []types.Diagnostic {
	var filtered []types.Diagnostic
	for _, d := range diagnostics {
		if !a.ignoreRules[d.Rule] {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (a *Analyzer) FindDeadCode(rootDir string) []types.Diagnostic {
	var allDiagnostics []types.Diagnostic

	filePaths := a.collectGoFiles(rootDir)

	exported := make(map[string][]rules.UsageInfo)
	allUsages := make(map[string][]string)

	for _, filePath := range filePaths {
		if a.shouldIgnoreFile(filePath, rootDir) {
			continue
		}

		file, err := parser.ParseFile(a.fset, filePath, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		rules.CollectExports(file, a.fset, filePath, rootDir, exported)
		rules.CollectUsages(file, a.fset, filePath, rootDir, allUsages)
	}

	for identifier, usageList := range exported {
		if _, used := allUsages[identifier]; !used {
			for _, info := range usageList {
				allDiagnostics = append(allDiagnostics, types.Diagnostic{
					FilePath: info.FilePath,
					Plugin:   "go-doctor",
					Rule:     "deadcode/unused-export",
					Severity: types.SeverityWarning,
					Message:  fmt.Sprintf("Exported %s '%s' is not used anywhere in the project", info.Kind, info.Name),
					Help:     "Consider removing or using this exported identifier, or add a doc comment explaining why it's exported",
					Line:     info.Line,
					Column:   info.Column,
					Category: types.CategoryDeadCode,
				})
			}
		}
	}

	return a.filterDiagnostics(allDiagnostics)
}
