package rules

import (
	"go/ast"
	"go/token"

	"github.com/go-doctor/go-doctor/pkg/types"
)

type Rule interface {
	ID() string
	Description() string
	Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic
}

type UsageInfo struct {
	Name     string
	Kind     string
	FilePath string
	Line     int
	Column   int
}

func AllRules() []Rule {
	return []Rule{
		&UncheckedErrorRule{},
		&SwallowedErrorRule{},
		&PanicInLibraryRule{},
		&DeferInLoopRule{},
		&RangeVarCaptureRule{},
		&MissingMutexUnlockRule{},
		&GoroutineLeakRule{},
		&StringConcatInLoopRule{},
		&UnnecessaryConversionRule{},
		&LargeStructCopyRule{},
		&SQLInjectionRule{},
		&CommandInjectionRule{},
		&WeakCryptoRule{},
		&HardcodedCredentialsRule{},
		&ExportedWithoutCommentRule{},
		&PackageNamingRule{},
		&FunctionComplexityRule{},
		&EmptyInterfaceRule{},
		&UnusedLabelRule{},
		&RedundantReturnRule{},
		&ErrorCheckWithoutHandling{},
		&FunctionDepthRule{},
		&FunctionLengthRule{},
		&FileLengthRule{},
		&LineLengthRule{},
		&VariableShadowRule{},
		&UnusedGlobalVarRule{},
		&UnusedStructFieldRule{},
		&PlaceholderCommentRule{},
		&SnakeCaseNamingRule{},
		&DebugPrintRule{},
		&EmptyFuncBodyRule{},
		&OverlyBroadInterfaceRule{},
	}
}

func CollectExports(file *ast.File, fset *token.FileSet, filePath string, rootDir string, exported map[string][]UsageInfo) {
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.IsExported() {
				pkg := file.Name.Name
				identifier := pkg + "." + d.Name.Name
				pos := fset.Position(d.Pos())
				exported[identifier] = append(exported[identifier], UsageInfo{
					Name:     d.Name.Name,
					Kind:     "function",
					FilePath: filePath,
					Line:     pos.Line,
					Column:   pos.Column,
				})
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						pkg := file.Name.Name
						identifier := pkg + "." + s.Name.Name
						pos := fset.Position(s.Pos())
						exported[identifier] = append(exported[identifier], UsageInfo{
							Name:     s.Name.Name,
							Kind:     "type",
							FilePath: filePath,
							Line:     pos.Line,
							Column:   pos.Column,
						})
					}
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							pkg := file.Name.Name
							identifier := pkg + "." + name.Name
							pos := fset.Position(name.Pos())
							exported[identifier] = append(exported[identifier], UsageInfo{
								Name:     name.Name,
								Kind:     "variable",
								FilePath: filePath,
								Line:     pos.Line,
								Column:   pos.Column,
							})
						}
					}
				}
			}
		}
	}
}

func CollectUsages(file *ast.File, fset *token.FileSet, filePath string, rootDir string, usages map[string][]string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.SelectorExpr:
			if ident, ok := node.X.(*ast.Ident); ok {
				identifier := ident.Name + "." + node.Sel.Name
				usages[identifier] = append(usages[identifier], filePath)
			}
		}
		return true
	})
}
