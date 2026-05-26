package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type PlaceholderCommentRule struct{}

func (r *PlaceholderCommentRule) ID() string { return "completeness/placeholder-comment" }
func (r *PlaceholderCommentRule) Description() string {
	return "Evaluates placeholder comments (TODO/FIXME/HACK/XXX) — indicates incomplete implementation that lowers AI-generated code quality"
}

func (r *PlaceholderCommentRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	placeholders := []string{"TODO", "FIXME", "HACK", "XXX", "PLACEHOLDER", "STUB"}

	ast.Inspect(file, func(n ast.Node) bool {
		group, ok := n.(*ast.CommentGroup)
		if !ok {
			return true
		}
		for _, c := range group.List {
			text := strings.ToUpper(c.Text)
			for _, p := range placeholders {
				if strings.Contains(text, p) {
					pos := fset.Position(c.Pos())
					diagnostics = append(diagnostics, types.Diagnostic{
						FilePath: filePath,
						Plugin:   "go-doctor",
						Rule:     r.ID(),
						Severity: types.SeverityWarning,
						Message:  "Placeholder comment detected: " + strings.TrimSpace(c.Text),
						Help:     "Replace placeholder with actual implementation — incomplete code lowers quality",
						Line:     pos.Line,
						Column:   pos.Column,
						Category: types.CategoryAICode,
					})
					break
				}
			}
		}
		return true
	})

	return diagnostics
}

type SnakeCaseNamingRule struct{}

func (r *SnakeCaseNamingRule) ID() string { return "idiomatic/snake-case-naming" }
func (r *SnakeCaseNamingRule) Description() string {
	return "Evaluates snake_case variable names — non-idiomatic Go naming lowers AI-generated code quality"
}

func (r *SnakeCaseNamingRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	seen := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.File:
			return true
		case *ast.ImportSpec:
			return false
		case *ast.Ident:
			name := node.Name
			if name == "_" || name == "nil" || seen[name] {
				return true
			}

			if name == file.Name.Name {
				return true
			}

			if isSnakeCase(name) {
				seen[name] = true
				pos := fset.Position(node.Pos())
				goName := toCamelCase(name)
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Variable '" + name + "' uses snake_case — not idiomatic Go",
					Help:     "Use camelCase in Go: rename '" + name + "' to '" + goName + "'",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryAICode,
				})
			}
		}
		return true
	})

	return diagnostics
}

func isSnakeCase(name string) bool {
	if len(name) < 2 {
		return false
	}
	if !isLower(name[0]) {
		return false
	}
	hasUnderscore := false
	for i, ch := range name {
		if ch == '_' {
			if i > 0 && i < len(name)-1 {
				hasUnderscore = true
			}
		}
	}
	return hasUnderscore
}

func isLower(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func toCamelCase(snake string) string {
	parts := strings.Split(snake, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	result := strings.Join(parts, "")
	if len(result) > 0 {
		result = strings.ToLower(result[:1]) + result[1:]
	}
	return result
}

type DebugPrintRule struct{}

func (r *DebugPrintRule) ID() string { return "cleanliness/debug-print" }
func (r *DebugPrintRule) Description() string {
	return "Evaluates fmt.Println/Printf debug calls — leftover debug code lowers AI-generated code quality"
}

func (r *DebugPrintRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	debugFuncs := map[string]bool{
		"Println": true,
		"Printf":  true,
		"Print":   true,
		"Sprintf": false,
	}

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "fmt" {
			if isDebug, exists := debugFuncs[sel.Sel.Name]; exists && isDebug {
				if !isTestFile(file.Name.Name) {
					pos := fset.Position(call.Pos())
					diagnostics = append(diagnostics, types.Diagnostic{
						FilePath: filePath,
						Plugin:   "go-doctor",
						Rule:     r.ID(),
						Severity: types.SeverityWarning,
						Message:  "Debug print statement: fmt." + sel.Sel.Name + "()",
						Help:     "Remove debug print or replace with proper logging: `log.Printf(...)` or `log.Println(...)`",
						Line:     pos.Line,
						Column:   pos.Column,
						Category: types.CategoryAICode,
					})
				}
			}
		}

		if sel.Sel.Name == "Println" || sel.Sel.Name == "Printf" || sel.Sel.Name == "Print" {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "log" {
				return true
			}
		}

		return true
	})

	return diagnostics
}

func isTestFile(pkgName string) bool {
	return strings.HasSuffix(pkgName, "_test")
}

type EmptyFuncBodyRule struct{}

func (r *EmptyFuncBodyRule) ID() string { return "implementation/empty-func-body" }
func (r *EmptyFuncBodyRule) Description() string {
	return "Evaluates functions with empty or minimal bodies — stub implementations lower AI-generated code quality"
}

func (r *EmptyFuncBodyRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		if fn.Name != nil && fn.Name.IsExported() && len(fn.Body.List) == 0 {
			pos := fset.Position(fn.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Function '" + fn.Name.Name + "' has an empty body — likely a stub",
				Help:     "Implement the function or add a doc comment explaining why it's intentionally empty",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategoryAICode,
			})
		}

		if fn.Name != nil && fn.Name.IsExported() && len(fn.Body.List) == 1 {
			if isOnlyReturn(fn.Body.List[0]) || isOnlyPanic(fn.Body.List[0]) {
				pos := fset.Position(fn.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Function '" + fn.Name.Name + "' has a minimal body — likely a stub",
					Help:     "Implement the function with actual logic or add a doc comment explaining the placeholder",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryAICode,
				})
			}
		}
	}

	return diagnostics
}

func isOnlyReturn(stmt ast.Stmt) bool {
	ret, ok := stmt.(*ast.ReturnStmt)
	if !ok {
		return false
	}
	if len(ret.Results) == 0 {
		return true
	}
	if len(ret.Results) == 1 {
		if ident, ok := ret.Results[0].(*ast.Ident); ok && ident.Name == "nil" {
			return true
		}
	}
	return false
}

func isOnlyPanic(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == "panic" {
		return true
	}
	return false
}

type OverlyBroadInterfaceRule struct{}

func (r *OverlyBroadInterfaceRule) ID() string { return "type-safety/overly-broad-interface" }
func (r *OverlyBroadInterfaceRule) Description() string {
	return "Evaluates functions using interface{}/any — overly broad types lower AI-generated code quality"
}

func (r *OverlyBroadInterfaceRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Type.Params == nil {
			continue
		}

		for _, field := range fn.Type.Params.List {
			if r.isBroadInterface(field.Type) {
				pos := fset.Position(field.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Function parameter uses overly broad type (interface{}/any) — lacks type safety",
					Help:     "Define a specific interface with required methods instead of accepting any type",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategoryAICode,
				})
			}
		}

		if fn.Type.Results != nil {
			for _, field := range fn.Type.Results.List {
				if r.isBroadInterface(field.Type) {
					pos := fset.Position(field.Pos())
					diagnostics = append(diagnostics, types.Diagnostic{
						FilePath: filePath,
						Plugin:   "go-doctor",
						Rule:     r.ID(),
						Severity: types.SeverityWarning,
						Message:  "Function return type uses overly broad type (interface{}/any) — lacks type safety",
						Help:     "Return a concrete type or define a specific interface",
						Line:     pos.Line,
						Column:   pos.Column,
						Category: types.CategoryAICode,
					})
				}
			}
		}
	}

	return diagnostics
}

func (r *OverlyBroadInterfaceRule) isBroadInterface(expr ast.Expr) bool {
	if iface, ok := expr.(*ast.InterfaceType); ok && len(iface.Methods.List) == 0 {
		return true
	}
	if ident, ok := expr.(*ast.Ident); ok && (ident.Name == "any" || ident.Name == "interface{}") {
		return true
	}
	return false
}
