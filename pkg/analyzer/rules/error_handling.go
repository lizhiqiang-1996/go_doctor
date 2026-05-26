package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type UncheckedErrorRule struct{}

func (r *UncheckedErrorRule) ID() string { return "error-handling/unchecked-error" }
func (r *UncheckedErrorRule) Description() string {
	return "Detects function calls that return errors but the error is not checked"
}

func (r *UncheckedErrorRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		stmt, ok := n.(*ast.ExprStmt)
		if !ok {
			return true
		}

		call, ok := stmt.X.(*ast.CallExpr)
		if !ok {
			return true
		}

		if r.isTestFile(file.Name.Name) {
			return true
		}

		if r.callReturnsError(call, file) {
			pos := fset.Position(call.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityError,
				Message:  "Function call returns an error but the error is not checked",
				Help:     "Assign the error return value and check it: `result, err := func(); if err != nil { ... }`",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryErrorHandling,
			})
		}

		return true
	})

	return diagnostics
}

func (r *UncheckedErrorRule) callReturnsError(call *ast.CallExpr, file *ast.File) bool {
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		methodName := fun.Sel.Name
		errorReturningMethods := map[string]bool{
			"Write":       true,
			"WriteString": true,
			"Read":        true,
			"ReadFile":    true,
			"WriteFile":   true,
			"Open":        true,
			"Create":      true,
			"Mkdir":       true,
			"MkdirAll":    true,
			"Remove":      true,
			"RemoveAll":   true,
			"Rename":      true,
			"Copy":        true,
			"Move":        true,
			"Seek":        true,
			"Close":       false,
			"Exec":        true,
			"Parse":       true,
			"Marshal":     true,
			"Unmarshal":   true,
			"Decode":      true,
			"Encode":      true,
			"Scan":        true,
			"Query":       true,
			"QueryRow":    true,
			"Prepare":     true,
			"Begin":       true,
			"Send":        true,
			"Recv":        true,
			"Dial":        true,
			"Listen":      true,
			"Accept":      true,
			"Connect":     true,
			"Get":         true,
			"Post":        true,
			"Put":         true,
			"Delete":      true,
			"Do":          true,
		}
		if errorReturningMethods[methodName] {
			return true
		}
	}

	if fun, ok := call.Fun.(*ast.Ident); ok {
		errorReturningFuncs := []string{
			"ParseInt", "ParseFloat", "ParseBool", "ParseUint",
			"Atoi", "Itoa",
			"ReadFile", "WriteFile", "ReadAll",
			"Open", "Create", "MkdirAll", "Mkdir",
			"Marshal", "Unmarshal",
			"NewDecoder", "NewEncoder",
			"Printf", "Sprintf", "Fprintf",
			"Errorf",
			"Join", "Append",
		}
		for _, fn := range errorReturningFuncs {
			if fun.Name == fn {
				return true
			}
		}
	}

	return false
}

func (r *UncheckedErrorRule) isTestFile(pkgName string) bool {
	return strings.HasSuffix(pkgName, "_test")
}

type SwallowedErrorRule struct{}

func (r *SwallowedErrorRule) ID() string { return "error-handling/swallowed-error" }
func (r *SwallowedErrorRule) Description() string {
	return "Detects error values that are assigned but never used"
}

func (r *SwallowedErrorRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok || ident.Name != "_" {
				continue
			}

			if i < len(assign.Rhs) {
				if call, ok := assign.Rhs[i].(*ast.CallExpr); ok {
					if r.isLikelyErrorReturningCall(call) {
						pos := fset.Position(ident.Pos())
						diagnostics = append(diagnostics, types.Diagnostic{
							FilePath: filePath,
							Plugin:   "go-doctor",
							Rule:     r.ID(),
							Severity: types.SeverityWarning,
							Message:  "Error return value is discarded with blank identifier",
							Help:     "Handle the error explicitly: `if err != nil { return err }` or `log.Printf(\"...\", err)`",
							Line:     pos.Line,
							Column:   pos.Column,
							Category: types.CategoryErrorHandling,
						})
					}
				}
			}
		}

		return true
	})

	return diagnostics
}

func (r *SwallowedErrorRule) isLikelyErrorReturningCall(call *ast.CallExpr) bool {
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		name := fun.Sel.Name
		suspectMethods := map[string]bool{
			"Write": true, "Read": true, "Open": true, "Close": true,
			"Parse": true, "Marshal": true, "Unmarshal": true,
			"Exec": true, "Query": true, "Dial": true,
			"Listen": true, "Accept": true, "Connect": true,
			"Send": true, "Recv": true, "Scan": true,
		}
		return suspectMethods[name]
	}
	return false
}

type PanicInLibraryRule struct{}

func (r *PanicInLibraryRule) ID() string { return "error-handling/panic-in-library" }
func (r *PanicInLibraryRule) Description() string {
	return "Detects panic calls in library code that should return errors instead"
}

func (r *PanicInLibraryRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	isMain := file.Name.Name == "main"

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		fun, ok := call.Fun.(*ast.Ident)
		if !ok || fun.Name != "panic" {
			return true
		}

		if isMain {
			return true
		}

		pos := fset.Position(call.Pos())
		diagnostics = append(diagnostics, types.Diagnostic{
			FilePath: filePath,
			Plugin:   "go-doctor",
			Rule:     r.ID(),
			Severity: types.SeverityWarning,
			Message:  "panic() used in library code — prefer returning errors",
			Help:     "Return an error instead: `func Foo() error { return fmt.Errorf(\"...\") }`",
			Line:     pos.Line,
			Column:   pos.Column,
			Category: types.CategoryErrorHandling,
		})

		return true
	})

	return diagnostics
}
