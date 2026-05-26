package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type StringConcatInLoopRule struct{}

func (r *StringConcatInLoopRule) ID() string { return "performance/string-concat-in-loop" }
func (r *StringConcatInLoopRule) Description() string {
	return "Detects string concatenation using += inside loops — should use strings.Builder"
}

func (r *StringConcatInLoopRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		var loopBody *ast.BlockStmt

		switch stmt := n.(type) {
		case *ast.ForStmt:
			loopBody = stmt.Body
		case *ast.RangeStmt:
			loopBody = stmt.Body
		default:
			return true
		}

		if loopBody == nil {
			return true
		}

		r.checkStringConcatInBlock(loopBody, fset, filePath, &diagnostics)
		return true
	})

	return diagnostics
}

func (r *StringConcatInLoopRule) checkStringConcatInBlock(block *ast.BlockStmt, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	for _, stmt := range block.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			continue
		}

		if assign.Tok != token.ADD_ASSIGN {
			continue
		}

		for _, rhs := range assign.Rhs {
			if r.isStringType(rhs) {
				pos := fset.Position(assign.Pos())
				*diagnostics = append(*diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "String concatenation with += inside a loop is O(n²) — use strings.Builder",
					Help:     "Replace with: `var b strings.Builder; b.WriteString(...); result = b.String()`",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryPerformance,
				})
			}
		}
	}
}

func (r *StringConcatInLoopRule) isStringType(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit:
		return e.Kind == token.STRING
	case *ast.BinaryExpr:
		if e.Op == token.ADD {
			return r.isStringType(e.X) || r.isStringType(e.Y)
		}
	case *ast.CallExpr:
		if fun, ok := e.Fun.(*ast.SelectorExpr); ok {
			return fun.Sel.Name == "Sprintf" || fun.Sel.Name == "FormatInt" || fun.Sel.Name == "FormatFloat"
		}
		if ident, ok := e.Fun.(*ast.Ident); ok {
			return ident.Name == "string" || ident.Name == "Sprintf"
		}
	}
	return true
}

type UnnecessaryConversionRule struct{}

func (r *UnnecessaryConversionRule) ID() string { return "performance/unnecessary-conversion" }
func (r *UnnecessaryConversionRule) Description() string {
	return "Detects unnecessary type conversions like string([]byte(s))"
}

func (r *UnnecessaryConversionRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}

		fun, ok := call.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		arg, ok := call.Args[0].(*ast.CallExpr)
		if !ok {
			return true
		}

		argFun, ok := arg.Fun.(*ast.Ident)
		if !ok {
			return true
		}

		if fun.Name == "string" && argFun.Name == "[]byte" {
			pos := fset.Position(call.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Unnecessary conversion: string([]byte(s)) — the string is already a string",
				Help:     "Remove the redundant conversion and use the string directly",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryPerformance,
			})
		}

		if fun.Name == "[]byte" && argFun.Name == "string" {
			pos := fset.Position(call.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Unnecessary conversion: []byte(string(b)) — the bytes are already bytes",
				Help:     "Remove the redundant conversion and use the byte slice directly",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryPerformance,
			})
		}

		return true
	})

	return diagnostics
}

type LargeStructCopyRule struct{}

func (r *LargeStructCopyRule) ID() string { return "performance/large-struct-copy" }
func (r *LargeStructCopyRule) Description() string {
	return "Detects function parameters that pass large structs by value instead of by pointer"
}

func (r *LargeStructCopyRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			continue
		}

		for _, field := range fn.Type.Params.List {
			if r.isStructType(field.Type) && !r.isPointerType(field.Type) {
				typeName := r.getTypeName(field.Type)
				if r.isLikelyLargeStruct(typeName) {
					pos := fset.Position(field.Pos())
					diagnostics = append(diagnostics, types.Diagnostic{
						FilePath: filePath,
						Plugin:   "go-doctor",
						Rule:     r.ID(),
						Severity: types.SeverityWarning,
						Message:  "Large struct '" + typeName + "' passed by value — consider using a pointer",
						Help:     "Change the parameter type to *" + typeName + " to avoid copying the entire struct",
						Line:     pos.Line,
						Column:   pos.Column,
						Category: types.CategoryPerformance,
					})
				}
			}
		}
	}

	return diagnostics
}

func (r *LargeStructCopyRule) isStructType(expr ast.Expr) bool {
	switch expr.(type) {
	case *ast.Ident:
		return true
	case *ast.SelectorExpr:
		return true
	case *ast.StarExpr:
		return false
	}
	return false
}

func (r *LargeStructCopyRule) isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

func (r *LargeStructCopyRule) getTypeName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	}
	return ""
}

func (r *LargeStructCopyRule) isLikelyLargeStruct(name string) bool {
	largeStructPatterns := []string{
		"Config", "Request", "Response", "Options", "Settings",
		"Context", "Message", "Payload", "Data", "Info",
		"Record", "Document", "Entity", "Model",
	}

	for _, pattern := range largeStructPatterns {
		if strings.Contains(name, pattern) {
			return true
		}
	}
	return false
}
