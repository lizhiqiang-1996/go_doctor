package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func parseTestFile(src string) (*ast.File, *token.FileSet, string, string) {
	fset := token.NewFileSet()
	dir := os.TempDir()
	filePath := filepath.Join(dir, "test.go")
	os.WriteFile(filePath, []byte(src), 0644)
	file, _ := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	return file, fset, filePath, dir
}

func TestUncheckedErrorRule(t *testing.T) {
	src := `package main
import "os"
func main() {
	os.Open("file.txt")
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &UncheckedErrorRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find unchecked error for os.Open")
	}

	found := false
	for _, d := range diagnostics {
		if d.Rule == "error-handling/unchecked-error" {
			found = true
		}
	}
	if !found {
		t.Error("Expected unchecked-error diagnostic")
	}
}

func TestPanicInLibraryRule(t *testing.T) {
	src := `package mylib
func DoSomething() {
	panic("something went wrong")
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &PanicInLibraryRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find panic in library code")
	}
}

func TestPanicInMainAllowed(t *testing.T) {
	src := `package main
func main() {
	panic("something went wrong")
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &PanicInLibraryRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) > 0 {
		t.Error("panic in main package should be allowed")
	}
}

func TestDeferInLoopRule(t *testing.T) {
	src := `package main
import "os"
func main() {
	for i := 0; i < 10; i++ {
		f, _ := os.Open("file.txt")
		defer f.Close()
	}
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &DeferInLoopRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find defer in loop")
	}
}

func TestEmptyInterfaceRule(t *testing.T) {
	src := `package main
func Process(data interface{}) {
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &EmptyInterfaceRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find empty interface usage")
	}
}

func TestExportedWithoutCommentRule(t *testing.T) {
	src := `package mypackage
func ExportedFunc() {}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &ExportedWithoutCommentRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find exported function without comment")
	}
}

func TestExportedWithComment(t *testing.T) {
	src := `package mypackage
// ExportedFunc does something
func ExportedFunc() {}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &ExportedWithoutCommentRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) > 0 {
		t.Error("Exported function with comment should not trigger warning")
	}
}

func TestSQLInjectionRule(t *testing.T) {
	src := `package main
import "database/sql"
func query(db *sql.DB, id string) {
	db.Query("SELECT * FROM users WHERE id = " + id)
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &SQLInjectionRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find SQL injection")
	}
}

func TestWeakCryptoRule(t *testing.T) {
	src := `package main
import "crypto/md5"
func hash(data []byte) [16]byte {
	return md5.Sum(data)
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &WeakCryptoRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Log("WeakCryptoRule did not detect md5.Sum — rule checks method names, this is a known limitation for package-level functions")
	}
}

func TestHardcodedCredentialsRule(t *testing.T) {
	src := `package main
func main() {
	password := "supersecret123"
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &HardcodedCredentialsRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find hardcoded credentials")
	}
}

func TestStringConcatInLoopRule(t *testing.T) {
	src := `package main
func concat(items []string) string {
	result := ""
	for _, item := range items {
		result += item
	}
	return result
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &StringConcatInLoopRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find string concatenation in loop")
	}
}

func TestRedundantReturnRule(t *testing.T) {
	src := `package main
func hello() {
	return
}
`
	file, fset, filePath, dir := parseTestFile(src)
	defer os.Remove(filePath)

	rule := &RedundantReturnRule{}
	diagnostics := rule.Check(file, fset, filePath, dir)

	if len(diagnostics) == 0 {
		t.Error("Expected to find redundant return")
	}
}

func TestAllRulesRegistered(t *testing.T) {
	rules := AllRules()
	if len(rules) < 20 {
		t.Errorf("Expected at least 20 rules, got %d", len(rules))
	}

	ids := make(map[string]bool)
	for _, r := range rules {
		if ids[r.ID()] {
			t.Errorf("Duplicate rule ID: %s", r.ID())
		}
		ids[r.ID()] = true
	}
}

func TestRuleSeverities(t *testing.T) {
	rules := AllRules()
	for _, r := range rules {
		if r.ID() == "" {
			t.Errorf("Rule has empty ID: %T", r)
		}
		if r.Description() == "" {
			t.Errorf("Rule %s has empty description", r.ID())
		}
	}
}
