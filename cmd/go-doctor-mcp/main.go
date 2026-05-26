package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lizhiqiang-1996/go_doctor/pkg/scanner"
	"github.com/lizhiqiang-1996/go_doctor/pkg/types"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer(
		"go-doctor",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(mcp.NewTool("scan_code_quality",
		mcp.WithDescription("Scan Go project code quality. Returns a structured report with score, issues by category including AI quality dimensions (completeness, idiomatic, cleanliness, implementation, type-safety)."),
		mcp.WithString("directory",
			mcp.Required(),
			mcp.Description("Absolute path to the Go project directory to scan"),
		),
		mcp.WithString("mode",
			mcp.Description("Scan mode: 'full' (all checks), 'diff' (changed files vs branch), 'commit' (specific commit)"),
			mcp.Enum("full", "diff", "commit"),
		),
		mcp.WithString("base_branch",
			mcp.Description("Base branch for diff mode (default: main)"),
		),
		mcp.WithString("commit_hash",
			mcp.Description("Commit hash for commit mode"),
		),
	), scanCodeQuality)

	s.AddTool(mcp.NewTool("get_code_score",
		mcp.WithDescription("Get only the code quality score (0-100) for a Go project. Fast and concise."),
		mcp.WithString("directory",
			mcp.Required(),
			mcp.Description("Absolute path to the Go project directory"),
		),
	), getCodeScore)

	s.AddTool(mcp.NewTool("evaluate_ai_code_quality",
		mcp.WithDescription("Evaluate the quality of AI-generated Go code across 5 dimensions: completeness, idiomatic naming, cleanliness, implementation depth, type safety."),
		mcp.WithString("directory",
			mcp.Required(),
			mcp.Description("Absolute path to the Go project directory"),
		),
	), evaluateAICodeQuality)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "MCP server error: %v\n", err)
		os.Exit(1)
	}
}

func getStringArg(args map[string]any, key string) string {
	val, _ := args[key]
	if val == nil {
		return ""
	}
	s, _ := val.(string)
	return s
}

func scanCodeQuality(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	directory := getStringArg(args, "directory")
	mode := getStringArg(args, "mode")
	baseBranch := getStringArg(args, "base_branch")
	commitHash := getStringArg(args, "commit_hash")

	if directory == "" {
		return mcp.NewToolResultError("directory is required"), nil
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid directory: %v", err)), nil
	}

	options := types.ScanOptions{
		Lint:     true,
		DeadCode: true,
		Verbose:  true,
	}

	switch mode {
	case "diff":
		if baseBranch == "" {
			baseBranch = "main"
		}
		options.DiffBase = baseBranch
	case "commit":
		if commitHash == "" {
			return mcp.NewToolResultError("commit_hash is required for commit mode"), nil
		}
		options.Commit = commitHash
	}

	sc := scanner.New(absDir, options)
	result := sc.Scan()

	report := buildReport(result, absDir)
	jsonBytes, _ := json.MarshalIndent(report, "", "  ")

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func getCodeScore(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	directory := getStringArg(args, "directory")

	if directory == "" {
		return mcp.NewToolResultError("directory is required"), nil
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid directory: %v", err)), nil
	}

	options := types.ScanOptions{
		Lint:      true,
		DeadCode:  true,
		ScoreOnly: true,
	}

	sc := scanner.New(absDir, options)
	result := sc.Scan()

	scoreInfo := map[string]interface{}{
		"project": result.Project.ProjectName,
		"score":   0,
		"label":   "Unknown",
		"files":   result.Project.SourceFileCount,
		"issues":  len(result.Diagnostics),
	}

	if result.Score != nil {
		scoreInfo["score"] = result.Score.Score
		scoreInfo["label"] = result.Score.Label
	}

	categoryCounts := make(map[string]int)
	for _, d := range result.Diagnostics {
		categoryCounts[string(d.Category)]++
	}
	if len(categoryCounts) > 0 {
		scoreInfo["breakdown"] = categoryCounts
	}

	jsonBytes, _ := json.MarshalIndent(scoreInfo, "", "  ")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func evaluateAICodeQuality(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	directory := getStringArg(args, "directory")

	if directory == "" {
		return mcp.NewToolResultError("directory is required"), nil
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid directory: %v", err)), nil
	}

	options := types.ScanOptions{
		Lint:     true,
		DeadCode: false,
	}

	sc := scanner.New(absDir, options)
	result := sc.Scan()

	aiIssues := make(map[string][]map[string]interface{})
	totalAI := 0
	for _, d := range result.Diagnostics {
		if d.Category == types.CategoryAICode {
			totalAI++
			issue := map[string]interface{}{
				"file":    d.FilePath,
				"line":    d.Line,
				"message": d.Message,
				"help":    d.Help,
			}
			aiIssues[d.Rule] = append(aiIssues[d.Rule], issue)
		}
	}

	aiScore := 100
	if result.Score != nil {
		aiScore = result.Score.Score
	}

	qualityLevel := "Excellent"
	if aiScore < 50 {
		qualityLevel = "Critical — significant quality issues found"
	} else if aiScore < 75 {
		qualityLevel = "Needs Work — quality issues should be addressed"
	}

	report := map[string]interface{}{
		"project":          result.Project.ProjectName,
		"ai_quality_score": aiScore,
		"quality_level":    qualityLevel,
		"issue_count":      totalAI,
		"issues_by_rule":   aiIssues,
		"quality_dimensions": map[string]string{
			"completeness":   "Are all functions fully implemented (no stubs/placeholders)?",
			"idiomatic":      "Does code follow Go naming conventions (camelCase)?",
			"cleanliness":    "Is production code free of debug print statements?",
			"type_safety":    "Are specific types used instead of interface{}/any?",
			"implementation": "Do all exported functions have real implementations?",
		},
	}

	jsonBytes, _ := json.MarshalIndent(report, "", "  ")
	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func buildReport(result *types.ScanResult, rootDir string) map[string]interface{} {
	report := map[string]interface{}{
		"project": map[string]interface{}{
			"name":        result.Project.ProjectName,
			"goVersion":   result.Project.GoVersion,
			"framework":   string(result.Project.Framework),
			"sourceFiles": result.Project.SourceFileCount,
			"modulePath":  result.Project.ModulePath,
		},
		"score": map[string]interface{}{
			"value": 0,
			"label": "Unknown",
		},
		"totalIssues": len(result.Diagnostics),
		"elapsedMs":   result.ElapsedMs,
	}

	if result.Score != nil {
		report["score"] = map[string]interface{}{
			"value": result.Score.Score,
			"label": result.Score.Label,
		}
	}

	categoryMap := make(map[string]interface{})
	for _, d := range result.Diagnostics {
		cat := string(d.Category)
		if _, ok := categoryMap[cat]; !ok {
			categoryMap[cat] = map[string]interface{}{
				"count":  0,
				"issues": []map[string]interface{}{},
			}
		}
		catData := categoryMap[cat].(map[string]interface{})
		catData["count"] = catData["count"].(int) + 1

		relPath := d.FilePath
		if len(d.FilePath) > len(rootDir) && d.FilePath[:len(rootDir)] == rootDir {
			relPath = d.FilePath[len(rootDir):]
			if len(relPath) > 0 && relPath[0] == '/' {
				relPath = relPath[1:]
			}
		}

		issue := map[string]interface{}{
			"rule":     d.Rule,
			"severity": string(d.Severity),
			"message":  d.Message,
			"help":     d.Help,
			"file":     relPath,
			"line":     d.Line,
		}
		catData["issues"] = append(catData["issues"].([]map[string]interface{}), issue)
	}

	if len(categoryMap) > 0 {
		report["categories"] = categoryMap
	}

	if result.DiffInfo != nil {
		report["diffInfo"] = result.DiffInfo
	}
	if result.CommitInfo != nil {
		report["commitInfo"] = result.CommitInfo
	}

	return report
}
