package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type UnusedGlobalVarRule struct{}

func (r *UnusedGlobalVarRule) ID() string { return "deadcode/unused-global-var" }
func (r *UnusedGlobalVarRule) Description() string {
	return "Detects global variables and constants that are declared but never used"
}

func (r *UnusedGlobalVarRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	declared := make(map[string]*usageInfo)
	used := make(map[string]bool)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok != token.VAR && genDecl.Tok != token.CONST {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == "_" || name.Name == "init" {
					continue
				}

				if !name.IsExported() {
					pos := fset.Position(name.Pos())
					declared[name.Name] = &usageInfo{
						name:     name.Name,
						filePath: filePath,
						line:     pos.Line,
						column:   pos.Column,
						kind:     "variable",
					}
				}
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		ident, ok := n.(*ast.Ident)
		if !ok {
			return true
		}

		if _, isDeclared := declared[ident.Name]; isDeclared {
			if isUsage(ident, file) {
				used[ident.Name] = true
			}
		}

		return true
	})

	for name, info := range declared {
		if !used[name] {
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: info.filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityWarning,
				Message:  "Global variable '" + name + "' is declared but never used",
				Help:     "Remove the unused variable or add a doc comment explaining why it's kept",
				Line:     info.line,
				Column:   info.column,
				Category: types.CategoryDeadCode,
			})
		}
	}

	return diagnostics
}

type usageInfo struct {
	name     string
	filePath string
	line     int
	column   int
	kind     string
}

func isUsage(ident *ast.Ident, file *ast.File) bool {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range valueSpec.Names {
				if name.Pos() == ident.Pos() {
					return false
				}
			}
		}
	}
	return true
}

type UnusedStructFieldRule struct{}

func (r *UnusedStructFieldRule) ID() string { return "deadcode/unused-struct-field" }
func (r *UnusedStructFieldRule) Description() string {
	return "Detects struct fields that are never accessed in the same package"
}

func (r *UnusedStructFieldRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	structFields := make(map[string]map[string]*fieldInfo)

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			structName := typeSpec.Name.Name
			if _, exists := structFields[structName]; !exists {
				structFields[structName] = make(map[string]*fieldInfo)
			}

			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue
				}
				for _, name := range field.Names {
					if name.IsExported() {
						continue
					}
					pos := fset.Position(name.Pos())
					structFields[structName][name.Name] = &fieldInfo{
						name:   name.Name,
						line:   pos.Line,
						column: pos.Column,
					}
				}
			}
		}
	}

	accessedFields := make(map[string]bool)

	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		accessedFields[sel.Sel.Name] = true
		return true
	})

	for structName, fields := range structFields {
		if strings.HasPrefix(structName, "mock") || strings.HasPrefix(structName, "Mock") {
			continue
		}

		for fieldName, info := range fields {
			if !accessedFields[fieldName] {
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityWarning,
					Message:  "Struct field '" + structName + "." + fieldName + "' is never accessed",
					Help:     "Remove the unused field or add a doc comment explaining why it's kept (e.g., JSON serialization)",
					Line:     info.line,
					Column:   info.column,
					Category: types.CategoryDeadCode,
				})
			}
		}
	}

	return diagnostics
}

type fieldInfo struct {
	name   string
	line   int
	column int
}
