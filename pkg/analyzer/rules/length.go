package rules

import (
	"bufio"
	"go/ast"
	"go/token"
	"os"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type FunctionLengthRule struct{}

func (r *FunctionLengthRule) ID() string { return "style/function-length" }
func (r *FunctionLengthRule) Description() string {
	return "Detects functions that are too long — long functions are hard to understand and test"
}

const maxFunctionLength = 80
const maxFileLength = 500
const maxLineLength = 120

func (r *FunctionLengthRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		startPos := fset.Position(fn.Body.Lbrace)
		endPos := fset.Position(fn.Body.Rbrace)
		length := endPos.Line - startPos.Line - 1
		if length < 0 {
			length = 1
		}

		if length > maxFunctionLength {
			pos := fset.Position(fn.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Function '" + fn.Name.Name + "' is too long (" + intToStr(length) + " lines > " + intToStr(maxFunctionLength) + ")",
				Help:     "Break the function into smaller, focused functions — each should do one thing well",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryCodeStyle,
			})
		}
	}

	return diagnostics
}

type FileLengthRule struct{}

func (r *FileLengthRule) ID() string { return "style/file-length" }
func (r *FileLengthRule) Description() string {
	return "Detects files that are too long — large files are hard to navigate and maintain"
}

func (r *FileLengthRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	lineCount, err := countFileLines(filePath)
	if err != nil {
		return diagnostics
	}

	if lineCount > maxFileLength {
		pos := fset.Position(file.Pos())
		diagnostics = append(diagnostics, types.Diagnostic{
			FilePath: filePath,
			Plugin:   "go-doctor",
			Rule:     r.ID(),
			Severity: types.SeverityWarning,
			Message:  "File is too long (" + intToStr(lineCount) + " lines > " + intToStr(maxFileLength) + ")",
			Help:     "Split the file into smaller, cohesive files by responsibility",
			Line:     pos.Line,
			Column:   pos.Column,
			Category: types.CategoryCodeStyle,
		})
	}

	return diagnostics
}

type LineLengthRule struct{}

func (r *LineLengthRule) ID() string { return "style/line-length" }
func (r *LineLengthRule) Description() string {
	return "Detects lines that exceed the recommended length limit"
}

func (r *LineLengthRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	longLines, err := findLongLines(filePath, maxLineLength)
	if err != nil {
		return diagnostics
	}

	for lineNum, lineLen := range longLines {
		if len(diagnostics) >= 10 {
			break
		}
		diagnostics = append(diagnostics, types.Diagnostic{
			FilePath: filePath,
			Plugin:   "go-doctor",
			Rule:     r.ID(),
			Severity: types.SeverityWarning,
			Message:  "Line exceeds recommended length (" + intToStr(lineLen) + " chars > " + intToStr(maxLineLength) + ")",
			Help:     "Break long lines or use multi-line formatting for readability",
			Line:     lineNum,
			Column:   1,
			Category: types.CategoryCodeStyle,
		})
	}

	return diagnostics
}

func countFileLines(filePath string) (int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func findLongLines(filePath string, maxLen int) (map[int]int, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[int]int)
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) > maxLen && !strings.HasPrefix(strings.TrimSpace(line), "//") && !strings.HasPrefix(strings.TrimSpace(line), "/*") {
			result[lineNum] = len(line)
		}
	}
	return result, scanner.Err()
}
