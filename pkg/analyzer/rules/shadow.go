package rules

import (
	"go/ast"
	"go/token"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type VariableShadowRule struct{}

func (r *VariableShadowRule) ID() string { return "correctness/variable-shadow" }
func (r *VariableShadowRule) Description() string {
	return "Detects variable shadowing where an inner scope declares a variable with the same name as an outer scope"
}

func (r *VariableShadowRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		r.checkShadowInFunc(fn, fset, filePath, &diagnostics)
	}

	return diagnostics
}

func (r *VariableShadowRule) checkShadowInFunc(fn *ast.FuncDecl, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	params := make(map[string]bool)
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			for _, name := range field.Names {
				params[name.Name] = true
			}
		}
	}

	if fn.Type.Results != nil {
		for _, field := range fn.Type.Results.List {
			for _, name := range field.Names {
				if name.Name != "" && name.Name != "_" {
					params[name.Name] = true
				}
			}
		}
	}

	outerVars := make(map[string]bool)
	for k := range params {
		outerVars[k] = true
	}

	r.collectAssignments(fn.Body, outerVars, fset, filePath, diagnostics)
}

func (r *VariableShadowRule) collectAssignments(block *ast.BlockStmt, outerVars map[string]bool, fset *token.FileSet, filePath string, diagnostics *[]types.Diagnostic) {
	if block == nil {
		return
	}

	localDecls := make(map[string]bool)

	for _, stmt := range block.List {
		switch node := stmt.(type) {
		case *ast.DeclStmt:
			if genDecl, ok := node.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for _, name := range valueSpec.Names {
							if name.Name != "_" && outerVars[name.Name] {
								pos := fset.Position(name.Pos())
								*diagnostics = append(*diagnostics, types.Diagnostic{
									FilePath: filePath,
									Plugin:   "go-doctor",
									Rule:     r.ID(),
									Severity: types.SeverityWarning,
									Message:  "Variable '" + name.Name + "' shadows an outer declaration",
									Help:     "Rename the variable to avoid confusion with the outer scope variable",
									Line:     pos.Line,
									Column:   pos.Column,
									Category: types.CategoryCorrectness,
								})
							}
							localDecls[name.Name] = true
						}
					}
				}
			}

		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
						if outerVars[ident.Name] && !localDecls[ident.Name] {
							pos := fset.Position(ident.Pos())
							*diagnostics = append(*diagnostics, types.Diagnostic{
								FilePath: filePath,
								Plugin:   "go-doctor",
								Rule:     r.ID(),
								Severity: types.SeverityWarning,
								Message:  "Variable '" + ident.Name + "' shadows an outer declaration (:=)",
								Help:     "Use = instead of := to assign to the outer variable, or rename to avoid shadowing",
								Line:     pos.Line,
								Column:   pos.Column,
								Category: types.CategoryCorrectness,
							})
						}
						localDecls[ident.Name] = true
					}
				}
			}

		case *ast.IfStmt:
			if node.Init != nil {
				if assign, ok := node.Init.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
					for _, lhs := range assign.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
							if outerVars[ident.Name] {
								pos := fset.Position(ident.Pos())
								*diagnostics = append(*diagnostics, types.Diagnostic{
									FilePath: filePath,
									Plugin:   "go-doctor",
									Rule:     r.ID(),
									Severity: types.SeverityWarning,
									Message:  "Variable '" + ident.Name + "' shadows an outer declaration in if-init",
									Help:     "Rename the variable to avoid confusion with the outer scope variable",
									Line:     pos.Line,
									Column:   pos.Column,
									Category: types.CategoryCorrectness,
								})
							}
						}
					}
				}
			}

		case *ast.ForStmt:
			if node.Init != nil {
				if assign, ok := node.Init.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
					for _, lhs := range assign.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
							if outerVars[ident.Name] {
								pos := fset.Position(ident.Pos())
								*diagnostics = append(*diagnostics, types.Diagnostic{
									FilePath: filePath,
									Plugin:   "go-doctor",
									Rule:     r.ID(),
									Severity: types.SeverityWarning,
									Message:  "Variable '" + ident.Name + "' shadows an outer declaration in for-init",
									Help:     "Rename the variable to avoid confusion with the outer scope variable",
									Line:     pos.Line,
									Column:   pos.Column,
									Category: types.CategoryCorrectness,
								})
							}
						}
					}
				}
			}

		case *ast.RangeStmt:
			if node.Key != nil {
				if ident, ok := node.Key.(*ast.Ident); ok && ident.Name != "_" {
					if outerVars[ident.Name] {
						pos := fset.Position(ident.Pos())
						*diagnostics = append(*diagnostics, types.Diagnostic{
							FilePath: filePath,
							Plugin:   "go-doctor",
							Rule:     r.ID(),
							Severity: types.SeverityWarning,
							Message:  "Range variable '" + ident.Name + "' shadows an outer declaration",
							Help:     "Rename the range variable to avoid confusion with the outer scope variable",
							Line:     pos.Line,
							Column:   pos.Column,
							Category: types.CategoryCorrectness,
						})
					}
				}
			}
			if node.Value != nil {
				if ident, ok := node.Value.(*ast.Ident); ok && ident.Name != "_" {
					if outerVars[ident.Name] {
						pos := fset.Position(ident.Pos())
						*diagnostics = append(*diagnostics, types.Diagnostic{
							FilePath: filePath,
							Plugin:   "go-doctor",
							Rule:     r.ID(),
							Severity: types.SeverityWarning,
							Message:  "Range variable '" + ident.Name + "' shadows an outer declaration",
							Help:     "Rename the range variable to avoid confusion with the outer scope variable",
							Line:     pos.Line,
							Column:   pos.Column,
							Category: types.CategoryCorrectness,
						})
					}
				}
			}

		case *ast.SwitchStmt:
			if node.Init != nil {
				if assign, ok := node.Init.(*ast.AssignStmt); ok && assign.Tok == token.DEFINE {
					for _, lhs := range assign.Lhs {
						if ident, ok := lhs.(*ast.Ident); ok && ident.Name != "_" {
							if outerVars[ident.Name] {
								pos := fset.Position(ident.Pos())
								*diagnostics = append(*diagnostics, types.Diagnostic{
									FilePath: filePath,
									Plugin:   "go-doctor",
									Rule:     r.ID(),
									Severity: types.SeverityWarning,
									Message:  "Variable '" + ident.Name + "' shadows an outer declaration in switch-init",
									Help:     "Rename the variable to avoid confusion with the outer scope variable",
									Line:     pos.Line,
									Column:   pos.Column,
									Category: types.CategoryCorrectness,
								})
							}
						}
					}
				}
			}
		}
	}
}
