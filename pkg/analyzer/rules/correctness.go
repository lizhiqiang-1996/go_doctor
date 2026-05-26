package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type EmptyInterfaceRule struct{}

func (r *EmptyInterfaceRule) ID() string { return "correctness/empty-interface" }
func (r *EmptyInterfaceRule) Description() string {
	return "Detects usage of interface{} which defeats type safety — use any or a concrete interface"
}

func (r *EmptyInterfaceRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.InterfaceType:
			if len(expr.Methods.List) == 0 {
				pos := fset.Position(expr.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Empty interface (interface{}) used — provides no type safety",
					Help:     "Define a concrete interface with required methods, or use 'any' (Go 1.18+) for clarity",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryCorrectness,
				})
			}
		}
		return true
	})

	return diagnostics
}

type UnusedLabelRule struct{}

func (r *UnusedLabelRule) ID() string { return "correctness/unused-label" }
func (r *UnusedLabelRule) Description() string {
	return "Detects labels that are defined but never used by goto, break, or continue"
}

func (r *UnusedLabelRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		labels := make(map[string]bool)
		usedLabels := make(map[string]bool)

		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.LabeledStmt:
				labels[node.Label.Name] = true
			case *ast.BranchStmt:
				if node.Label != nil {
					usedLabels[node.Label.Name] = true
				}
			}
			return true
		})

		for label := range labels {
			if !usedLabels[label] {
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Label '" + label + "' is defined but never used",
					Help:     "Remove the unused label to improve code clarity",
					Line:     0,
					Column:   0,
					Category: types.CategoryCorrectness,
				})
			}
		}
	}

	return diagnostics
}

type RedundantReturnRule struct{}

func (r *RedundantReturnRule) ID() string { return "correctness/redundant-return" }
func (r *RedundantReturnRule) Description() string {
	return "Detects redundant return statements at the end of functions"
}

func (r *RedundantReturnRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		hasNamedReturns := false
		if fn.Type.Results != nil {
			for _, field := range fn.Type.Results.List {
				if len(field.Names) > 0 {
					hasNamedReturns = true
					break
				}
			}
		}

		if hasNamedReturns {
			continue
		}

		stmts := fn.Body.List
		if len(stmts) == 0 {
			continue
		}

		lastStmt := stmts[len(stmts)-1]
		retStmt, ok := lastStmt.(*ast.ReturnStmt)
		if !ok {
			continue
		}

		if len(retStmt.Results) == 0 {
			pos := fset.Position(retStmt.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Redundant return at end of function",
				Help:     "Remove the bare return statement — it's unnecessary at the end of a function",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryCorrectness,
			})
		}
	}

	return diagnostics
}

type ErrorCheckWithoutHandling struct{}

func (r *ErrorCheckWithoutHandling) ID() string { return "correctness/error-check-without-handling" }
func (r *ErrorCheckWithoutHandling) Description() string {
	return "Detects if err != nil checks that only log the error without returning or handling it"
}

func (r *ErrorCheckWithoutHandling) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		ifStmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}

		if !r.isErrorCheck(ifStmt.Cond) {
			return true
		}

		if ifStmt.Body == nil {
			return true
		}

		hasReturn := false
		hasPanic := false
		hasContinue := false
		hasBreak := false
		hasOsExit := false
		logOnly := true

		ast.Inspect(ifStmt.Body, func(inner ast.Node) bool {
			switch node := inner.(type) {
			case *ast.ReturnStmt:
				hasReturn = true
				logOnly = false
			case *ast.CallExpr:
				if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
					if sel.Sel.Name == "Exit" {
						hasOsExit = true
						logOnly = false
					}
					methodName := strings.ToLower(sel.Sel.Name)
					if !strings.Contains(methodName, "log") && !strings.Contains(methodName, "print") {
						logOnly = false
					}
				}
				if ident, ok := node.Fun.(*ast.Ident); ok {
					if ident.Name == "panic" {
						hasPanic = true
						logOnly = false
					}
				}
			case *ast.BranchStmt:
				if node.Tok == token.CONTINUE {
					hasContinue = true
					logOnly = false
				}
				if node.Tok == token.BREAK {
					hasBreak = true
					logOnly = false
				}
			}
			return true
		})

		if logOnly && !hasReturn && !hasPanic && !hasContinue && !hasBreak && !hasOsExit {
			pos := fset.Position(ifStmt.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Error is checked but only logged — error is not propagated or handled",
				Help:     "Return the error to the caller: `if err != nil { return err }` or `return fmt.Errorf(\"context: %w\", err)`",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryCorrectness,
			})
		}

		return true
	})

	return diagnostics
}

func (r *ErrorCheckWithoutHandling) isErrorCheck(cond ast.Expr) bool {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok {
		return false
	}

	if bin.Op != token.NEQ {
		return false
	}

	if ident, ok := bin.Y.(*ast.Ident); ok && ident.Name == "nil" {
		if x, ok := bin.X.(*ast.Ident); ok && x.Name == "err" {
			return true
		}
	}

	return false
}
