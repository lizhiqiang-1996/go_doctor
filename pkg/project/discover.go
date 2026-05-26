package project

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type GoModFile struct {
	Module  string `json:"Module"`
	Go      string `json:"Go"`
	Require []struct {
		Path    string `json:"Path"`
		Version string `json:"Version"`
	} `json:"Require"`
}

func Discover(rootDir string) types.ProjectInfo {
	info := types.ProjectInfo{
		RootDirectory: rootDir,
		Framework:     types.FrameworkUnknown,
	}

	info.ProjectName = filepath.Base(rootDir)

	goModPath := filepath.Join(rootDir, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		info.HasGoMod = true
		parseGoMod(goModPath, &info)
	}

	goSumPath := filepath.Join(rootDir, "go.sum")
	if _, err := os.Stat(goSumPath); err == nil {
		info.HasGoSum = true
	}

	detectGoVersion(rootDir, &info)
	detectFramework(rootDir, &info)
	countSourceFiles(rootDir, &info)

	return info
}

func parseGoMod(path string, info *types.ProjectInfo) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			info.ModulePath = strings.TrimPrefix(line, "module ")
			info.ModulePath = strings.TrimSpace(info.ModulePath)
		}
		if strings.HasPrefix(line, "go ") {
			version := strings.TrimPrefix(line, "go ")
			version = strings.TrimSpace(version)
			if info.GoVersion == "" || info.GoVersion == "unknown" {
				info.GoVersion = version
				parseVersionNumbers(version, info)
			}
		}
	}
}

func detectGoVersion(rootDir string, info *types.ProjectInfo) {
	cmd := exec.Command("go", "version")
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err == nil {
		parts := strings.Fields(string(output))
		for _, part := range parts {
			if strings.HasPrefix(part, "go") && len(part) > 2 {
				version := strings.TrimPrefix(part, "go")
				if version != "" && version[0] >= '0' && version[0] <= '9' {
					info.GoVersion = version
					parseVersionNumbers(version, info)
					return
				}
			}
		}
	}

	if info.GoVersion == "" {
		goModPath := filepath.Join(rootDir, "go.mod")
		if data, err := os.ReadFile(goModPath); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "go ") {
					version := strings.TrimSpace(strings.TrimPrefix(line, "go "))
					info.GoVersion = version
					parseVersionNumbers(version, info)
					return
				}
			}
		}
	}

	if info.GoVersion == "" {
		info.GoVersion = "unknown"
	}
}

func parseVersionNumbers(version string, info *types.ProjectInfo) {
	parts := strings.SplitN(version, ".", 3)
	if len(parts) >= 1 {
		fmt.Sscanf(parts[0], "%d", &info.GoMajorVersion)
	}
	if len(parts) >= 2 {
		fmt.Sscanf(parts[1], "%d", &info.GoMinorVersion)
	}
}

func detectFramework(rootDir string, info *types.ProjectInfo) {
	goModPath := filepath.Join(rootDir, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return
	}

	content := string(data)

	frameworkDeps := map[string]types.Framework{
		"github.com/gin-gonic/gin":   types.FrameworkGin,
		"github.com/labstack/echo":   types.FrameworkEcho,
		"github.com/gofiber/fiber":   types.FrameworkFiber,
		"github.com/go-chi/chi":      types.FrameworkChi,
		"google.golang.org/grpc":     types.FrameworkGRPC,
		"github.com/cloudwego/kitex": types.FrameworkKitex,
		"github.com/cloudwego/hertz": types.FrameworkHertz,
	}

	for dep, fw := range frameworkDeps {
		if strings.Contains(content, dep) {
			info.Framework = fw
			return
		}
	}

	info.Framework = types.FrameworkStdLib
}

func countSourceFiles(rootDir string, info *types.ProjectInfo) {
	count := 0
	fset := token.NewFileSet()

	filepath.Walk(rootDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			name := fi.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		_, parseErr := parser.ParseFile(fset, path, nil, parser.PackageClauseOnly)
		if parseErr == nil {
			count++
		}
		return nil
	})

	info.SourceFileCount = count
}

func DiscoverFromGoList(rootDir string) []string {
	cmd := exec.Command("go", "list", "-json", "./...")
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var packages []struct {
		Dir         string   `json:"Dir"`
		GoFiles     []string `json:"GoFiles"`
		TestGoFiles []string `json:"TestGoFiles"`
	}

	decoder := json.NewDecoder(strings.NewReader(string(output)))
	for decoder.More() {
		var pkg struct {
			Dir         string   `json:"Dir"`
			GoFiles     []string `json:"GoFiles"`
			TestGoFiles []string `json:"TestGoFiles"`
		}
		if err := decoder.Decode(&pkg); err != nil {
			break
		}
		packages = append(packages, pkg)
	}

	var files []string
	for _, pkg := range packages {
		for _, f := range pkg.GoFiles {
			files = append(files, filepath.Join(pkg.Dir, f))
		}
	}

	_ = packages
	return files
}
