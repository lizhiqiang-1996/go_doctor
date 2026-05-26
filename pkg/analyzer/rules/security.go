package rules

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type SQLInjectionRule struct{}

func (r *SQLInjectionRule) ID() string { return "security/sql-injection" }
func (r *SQLInjectionRule) Description() string {
	return "Detects potential SQL injection by string concatenation in query construction"
}

func (r *SQLInjectionRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		methodName := sel.Sel.Name
		if methodName != "Query" && methodName != "QueryRow" && methodName != "Exec" {
			return true
		}

		if len(call.Args) == 0 {
			return true
		}

		firstArg := call.Args[0]
		if r.isStringConcat(firstArg) || r.isSprintf(firstArg) {
			pos := fset.Position(firstArg.Pos())
			diagnostics = append(diagnostics, types.Diagnostic{
				FilePath: filePath,
				Plugin:   "go-doctor",
				Rule:     r.ID(),
				Severity: types.SeverityError,
				Message:  "Potential SQL injection — query built with string concatenation",
				Help:     "Use parameterized queries: `db.Query(\"SELECT * FROM users WHERE id = $1\", userID)`",
				Line:     pos.Line,
				Column:   pos.Column,
				Category: types.CategorySecurity,
			})
		}

		return true
	})

	return diagnostics
}

func (r *SQLInjectionRule) isStringConcat(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	if bin.Op == token.ADD {
		return true
	}
	return false
}

func (r *SQLInjectionRule) isSprintf(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		return fun.Sel.Name == "Sprintf"
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name == "Sprintf"
	}
	return false
}

type CommandInjectionRule struct{}

func (r *CommandInjectionRule) ID() string { return "security/command-injection" }
func (r *CommandInjectionRule) Description() string {
	return "Detects potential command injection via exec.Command with string concatenation"
}

func (r *CommandInjectionRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "Command" {
			return true
		}

		if ident, ok := sel.X.(*ast.Ident); ok && ident.Name != "exec" {
			return true
		}

		for _, arg := range call.Args {
			if r.isStringConcat(arg) || r.isSprintfCall(arg) {
				pos := fset.Position(arg.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityError,
					Message:  "Potential command injection — exec.Command argument built with string concatenation",
					Help:     "Pass arguments separately: `exec.Command(\"cmd\", arg1, arg2)` instead of concatenating",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategorySecurity,
				})
			}
		}

		return true
	})

	return diagnostics
}

func (r *CommandInjectionRule) isSprintfCall(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	if fun, ok := call.Fun.(*ast.SelectorExpr); ok {
		return fun.Sel.Name == "Sprintf"
	}
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name == "Sprintf"
	}
	return false
}

func (r *CommandInjectionRule) isStringConcat(expr ast.Expr) bool {
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return bin.Op == token.ADD
}

type WeakCryptoRule struct{}

func (r *WeakCryptoRule) ID() string { return "security/weak-crypto" }
func (r *WeakCryptoRule) Description() string {
	return "Detects use of weak cryptographic algorithms"
}

func (r *WeakCryptoRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	weakAlgos := map[string]string{
		"md4":      "MD4 is cryptographically broken",
		"md5":      "MD5 is cryptographically broken — use SHA-256 or stronger",
		"sha1":     "SHA-1 is cryptographically broken — use SHA-256 or stronger",
		"des":      "DES is insecure — use AES-256",
		"rc4":      "RC4 is insecure — use AES-256",
		"blowfish": "Blowfish is outdated — use AES-256",
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

		methodName := strings.ToLower(sel.Sel.Name)
		for algo, msg := range weakAlgos {
			if strings.Contains(methodName, algo) {
				pos := fset.Position(call.Pos())
				diagnostics = append(diagnostics, types.Diagnostic{
					FilePath: filePath,
					Plugin:   "go-doctor",
					Rule:     r.ID(),
					Severity: types.SeverityError,
					Message:  msg,
					Help:     "Use crypto/sha256 for hashing or crypto/aes for encryption",
					Line:     pos.Line,
					Column:   pos.Column,
					Category: types.CategorySecurity,
				})
				break
			}
		}

		return true
	})

	return diagnostics
}

type HardcodedCredentialsRule struct{}

func (r *HardcodedCredentialsRule) ID() string { return "security/hardcoded-credentials" }
func (r *HardcodedCredentialsRule) Description() string {
	return "Detects hardcoded passwords, API keys, or secrets in source code"
}

func (r *HardcodedCredentialsRule) Check(file *ast.File, fset *token.FileSet, filePath string, rootDir string) []types.Diagnostic {
	var diagnostics []types.Diagnostic

	credentialPatterns := []string{
		"password", "passwd", "secret", "apikey", "api_key", "accesskey", "access_key",
		"token", "auth_token", "bearer", "credential",
	}

	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}

		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if !ok {
				continue
			}

			nameLower := strings.ToLower(ident.Name)
			isCredential := false
			for _, pattern := range credentialPatterns {
				if strings.Contains(nameLower, pattern) {
					isCredential = true
					break
				}
			}

			if !isCredential {
				continue
			}

			if i < len(assign.Rhs) {
				if _, ok := assign.Rhs[i].(*ast.BasicLit); ok {
					pos := fset.Position(ident.Pos())
					diagnostics = append(diagnostics, types.Diagnostic{
						FilePath: filePath,
						Plugin:   "go-doctor",
						Rule:     r.ID(),
						Severity: types.SeverityError,
						Message:  "Potential hardcoded credential in variable '" + ident.Name + "'",
						Help:     "Use environment variables or a secrets manager: `os.Getenv(\"API_KEY\")`",
						Line:     pos.Line,
						Column:   pos.Column,
						Category: types.CategorySecurity,
					})
				}
			}
		}

		return true
	})

	return diagnostics
}
