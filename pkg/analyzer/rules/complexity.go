package rules

import (
	"go/ast"
	"go/token"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type FunctionDepthRule struct{}

func (r *FunctionDepthRule) ID() string { return "complexity/function-depth" }
func (r *FunctionDepthRule) Description() string {
	return "Detects functions with excessive nesting depth — deep nesting makes code hard to read and test"
}

const maxDepth = 5

func (r *FunctionDepthRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		depth := r.calculateMaxDepth(fn.Body)
		if depth > maxDepth {
			pos := fset.Position(fn.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Function has deep nesting (depth " + intToStr(depth) + " > " + intToStr(maxDepth) + ")",
				Help:     "Extract nested logic into separate functions with descriptive names to reduce nesting",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryCodeStyle,
			})
		}
	}

	return diagnostics
}

func (r *FunctionDepthRule) calculateMaxDepth(block *ast.BlockStmt) int {
	maxD := 0
	var walk func(ast.Node, int)
	walk = func(node ast.Node, depth int) {
		if depth > maxD {
			maxD = depth
		}

		switch n := node.(type) {
		case *ast.BlockStmt:
			for _, stmt := range n.List {
				walk(stmt, depth)
			}
		case *ast.IfStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
			if n.Else != nil {
				switch els := n.Else.(type) {
				case *ast.IfStmt:
					walk(els, depth)
				case *ast.BlockStmt:
					walk(els, depth+1)
				}
			}
		case *ast.ForStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
		case *ast.RangeStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
		case *ast.SwitchStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
		case *ast.TypeSwitchStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
		case *ast.SelectStmt:
			if n.Body != nil {
				walk(n.Body, depth+1)
			}
		case *ast.CaseClause:
			for _, stmt := range n.Body {
				walk(stmt, depth)
			}
		case *ast.CommClause:
			for _, stmt := range n.Body {
				walk(stmt, depth)
			}
		}
	}

	for _, stmt := range block.List {
		walk(stmt, 1)
	}

	return maxD
}
