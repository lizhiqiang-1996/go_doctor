package rules

import (
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type ExportedWithoutCommentRule struct{}

func (r *ExportedWithoutCommentRule) ID() string { return "style/exported-without-comment" }
func (r *ExportedWithoutCommentRule) Description() string {
	return "Detects exported functions, types, and variables without doc comments"
}

func (r *ExportedWithoutCommentRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() && d.Doc == nil {
				pos := fset.Position(d.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Exported function '" + d.Name.Name + "' has no doc comment",
					Help:     "Add a doc comment starting with the function name: `// " + d.Name.Name + " does ...`",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryCodeStyle,
				})
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() && d.Doc == nil {
						pos := fset.Position(s.Pos())
						diagnostics = append(diagnostics, types.Diagnostic{
							FilePath: filePath,
							Plugin:   "go-doctor",
							Rule:     r.ID(),
							Severity: types.SeverityWarning,
							Message:  "Exported type '" + s.Name.Name + "' has no doc comment",
							Help:     "Add a doc comment starting with the type name: `// " + s.Name.Name + " represents ...`",
							Line:     pos.Line,
							Column:   pos.Column,
							Category: types.CategoryCodeStyle,
						})
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() && d.Doc == nil {
							pos := fset.Position(name.Pos())
							diagnostics = append(diagnostics, types.Diagnostic{
								FilePath: filePath,
								Plugin:   "go-doctor",
								Rule:     r.ID(),
								Severity: types.SeverityWarning,
								Message:  "Exported variable '" + name.Name + "' has no doc comment",
								Help:     "Add a doc comment: `// " + name.Name + " is ...`",
								Line:     pos.Line,
								Column:   pos.Column,
								Category: types.CategoryCodeStyle,
							})
						}
					}
				}
			}
		}
	}

	return diagnostics
}

type PackageNamingRule struct{}

func (r *PackageNamingRule) ID() string { return "style/package-naming" }
func (r *PackageNamingRule) Description() string {
	return "Detects package names that don't follow Go conventions (lowercase, no underscores)"
}

func (r *PackageNamingRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	name := file.Name.Name
	if name == "" {
		return diagnostics
	}

	hasUpper := false
	hasUnderscore := false
	hasHyphen := false

	for _, ch := range name {
		if unicode.IsUpper(ch) {
			hasUpper = true
		}
		if ch == '_' {
			hasUnderscore = true
		}
		if ch == '-' {
			hasHyphen = true
		}
	}

	if hasUpper || hasUnderscore || hasHyphen {
		pos := fset.Position(file.Pos())
		suggested := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(name, "_", ""), "-", ""))
		diagnostics = append(diagnostics, types.Diagnostic{
			FilePath: filePath,
			Plugin:   "go-doctor",
			Rule:     r.ID(),
			Severity: types.SeverityWarning,
			Message:  "Package name '" + name + "' doesn't follow Go naming conventions",
			Help:     "Use lowercase without underscores or hyphens, e.g., '" + suggested + "'",
			Line:     pos.Line,
			Column:   pos.Column,
			Category: types.CategoryCodeStyle,
		})
	}

	return diagnostics
}

type FunctionComplexityRule struct{}

func (r *FunctionComplexityRule) ID() string { return "style/function-complexity" }
func (r *FunctionComplexityRule) Description() string {
	return "Detects functions with high cyclomatic complexity"
}

const maxComplexity = 15

func (r *FunctionComplexityRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		complexity := r.calculateComplexity(fn.Body)
		if complexity > maxComplexity {
			pos := fset.Position(fn.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Function has high cyclomatic complexity (" + intToStr(complexity) + " > " + intToStr(maxComplexity) + ")",
				Help:     "Break the function into smaller, focused functions with clear responsibilities",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryCodeStyle,
			})
		}
	}

	return diagnostics
}

func (r *FunctionComplexityRule) calculateComplexity(body *ast.BlockStmt) int {
	complexity := 1

	ast.Inspect(body, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		case *ast.CommClause:
			complexity++
		case *ast.BinaryExpr:
			complexity++
		}
		return true
	})

	return complexity
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
