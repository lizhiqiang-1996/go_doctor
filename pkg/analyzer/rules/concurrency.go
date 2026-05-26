package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type DeferInLoopRule struct{}

func (r *DeferInLoopRule) ID() string { return "concurrency/defer-in-loop" }
func (r *DeferInLoopRule) Description() string {
	return "Detects defer calls inside loops which delay resource cleanup until the function returns"
}

func (r *DeferInLoopRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.ForStmt:
			if stmt.Body != nil {
				r.checkDeferInBlock(stmt.Body, fset, filePath, &diagnostics)
			}
		case *ast.RangeStmt:
			if stmt.Body != nil {
				r.checkDeferInBlock(stmt.Body, fset, filePath, &diagnostics)
			}
		}
		return true
	})

	return diagnostics
}

func (r *DeferInLoopRule) checkDeferInBlock(block *ast.BlockStmt, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	for _, stmt := range block.List {
		if deferStmt, ok := stmt.(*ast.DeferStmt); ok {
			pos := fset.Position(deferStmt.Pos())
			*diagnostics = append(*diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "defer inside a loop — resource will not be released until the function returns",
				Help:     "Move the deferred call into a separate function or use an anonymous function: `func() { defer cleanup(); ... }()`",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryConcurrency,
			})
		}
	}
}

type RangeVarCaptureRule struct{}

func (r *RangeVarCaptureRule) ID() string { return "concurrency/range-var-capture" }
func (r *RangeVarCaptureRule) Description() string {
	return "Detects range loop variables captured by reference in goroutines or closures"
}

func (r *RangeVarCaptureRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		rangeStmt, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}

		var loopVars []string
		if rangeStmt.Key != nil {
			if ident, ok := rangeStmt.Key.(*ast.Ident); ok && ident.Name != "_" {
				loopVars = append(loopVars, ident.Name)
			}
		}
		if rangeStmt.Value != nil {
			if ident, ok := rangeStmt.Value.(*ast.Ident); ok && ident.Name != "_" {
				loopVars = append(loopVars, ident.Name)
			}
		}

		if len(loopVars) == 0 {
			return true
		}

		ast.Inspect(rangeStmt.Body, func(inner ast.Node) bool {
			goStmt, ok := inner.(*ast.GoStmt)
			if !ok {
				return true
			}

			r.checkClosureCaptures(goStmt.Call.Fun, loopVars, fset, filePath, &diagnostics)
			return true
		})

		return true
	})

	return diagnostics
}

func (r *RangeVarCaptureRule) checkClosureCaptures(expr ast.Expr, loopVars []string, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	funLit, ok := expr.(*ast.FuncLit)
	if !ok {
		return
	}

	ast.Inspect(funLit.Body, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		for _, v := range loopVars {
			if ident.Name == v {
				pos := fset.Position(ident.Pos())
				*diagnostics = append(*diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityError,
					Message:  "Range variable '" + v + "' captured by reference in goroutine",
					Help:     "Create a local copy: `v := v` before the goroutine, or use a parameter: `go func(v Type) { ... }(v)`",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryConcurrency,
				})
			}
		}
		return true
	})
}

type MissingMutexUnlockRule struct{}

func (r *MissingMutexUnlockRule) ID() string { return "concurrency/missing-mutex-unlock" }
func (r *MissingMutexUnlockRule) Description() string {
	return "Detects mutex Lock calls without corresponding Unlock in the same function"
}

func (r *MissingMutexUnlockRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		r.checkMutexBalance(fn.Body, fset, filePath, &diagnostics)
	}

	return diagnostics
}

func (r *MissingMutexUnlockRule) checkMutexBalance(body *ast.BlockStmt, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	lockVars := make(map[string]bool)
	unlockVars := make(map[string]bool)
	deferUnlockVars := make(map[string]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		varName := r.getVarName(sel.X)
		if varName == "" {
			return true
		}

		switch sel.Sel.Name {
		case "Lock", "RLock":
			lockVars[varName] = true
		case "Unlock", "RUnlock":
			unlockVars[varName] = true
		}

		return true
	})

	ast.Inspect(body, func(n ast.Node) bool {
		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok {
			return true
		}

		call, ok := deferStmt.Call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		varName := r.getVarName(call.X)
		if varName == "" {
			return true
		}

		if call.Sel.Name == "Unlock" || call.Sel.Name == "RUnlock" {
			deferUnlockVars[varName] = true
		}

		return true
	})

	for v := range lockVars {
		if !unlockVars[v] && !deferUnlockVars[v] {
			*diagnostics = append(*diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityError,
				Message:  "Mutex '" + v + "' Lock() without matching Unlock() in the same function",
				Help:     "Add `defer " + v + ".Unlock()` immediately after the Lock() call",
				Line:     0,
				Column:   0,
				Category: types.CategoryConcurrency,
			})
		}
	}
}

func (r *MissingMutexUnlockRule) getVarName(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return r.getVarName(sel.X) + "." + sel.Sel.Name
	}
	return ""
}

type GoroutineLeakRule struct{}

func (r *GoroutineLeakRule) ID() string { return "concurrency/goroutine-leak" }
func (r *GoroutineLeakRule) Description() string {
	return "Detects goroutines launched without context cancellation or done channel"
}

func (r *GoroutineLeakRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		goStmt, ok := n.(*ast.GoStmt)
		if !ok {
			return true
		}

		funLit, ok := goStmt.Call.Fun.(*ast.FuncLit)
		if !ok {
			return true
		}

		hasContext := false
		hasDoneChannel := false
		hasReturn := false

		ast.Inspect(funLit, func(inner ast.Node) bool {
			switch node := inner.(type) {
			case *ast.Ident:
				if strings.Contains(node.Name, "ctx") || strings.Contains(node.Name, "context") {
					hasContext = true
				}
				if strings.Contains(node.Name, "done") || strings.Contains(node.Name, "stop") || strings.Contains(node.Name, "quit") {
					hasDoneChannel = true
				}
			case *ast.SelectorExpr:
				if node.Sel.Name == "Done" || node.Sel.Name == "Context" {
					hasContext = true
				}
			case *ast.ReturnStmt:
				hasReturn = true
			}
			return true
		})

		if !hasContext && !hasDoneChannel && hasReturn {
			pos := fset.Position(goStmt.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Goroutine may leak — no context cancellation or done channel detected",
				Help:     "Pass a context.Context and check ctx.Done() to allow graceful shutdown",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryConcurrency,
			})
		}

		return true
	})

	return diagnostics
}
