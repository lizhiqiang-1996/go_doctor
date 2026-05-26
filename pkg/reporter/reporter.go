package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/go-doctor/go-doctor/pkg/scorer"
	"github.com/go-doctor/go-doctor/pkg/types"
)

const (
	barWidth = 30
)

type Reporter struct {
	verbose bool
	json    bool
	rootDir string
}

func New(verbose bool, jsonMode bool, rootDir string) *Reporter {
	return &Reporter{
		verbose: verbose,
		json:    jsonMode,
		rootDir: rootDir,
	}
}

func (r *Reporter) Print(result *types.ScanResult) {
	if r.json {
		r.printJSON(result)
		return
	}

	r.printScoreHeader(result.Score)
	r.printProjectInfo(result.Project)

	if result.DiffInfo != nil {
		r.printDiffInfo(result.DiffInfo)
	}
	if result.CommitInfo != nil {
		r.printCommitInfo(result.CommitInfo)
	}

	fmt.Println()

	if len(result.Diagnostics) == 0 {
		fmt.Println("  ✅ No issues found! Your Go code is healthy.")
		fmt.Println()
		return
	}

	r.printDiagnostics(result.Diagnostics)
	r.printSummary(result)
	r.printElapsedTime(result.ElapsedMs)
}

func (r *Reporter) printScoreHeader(score *types.ScoreResult) {
	if score == nil {
		fmt.Println("  Go Doctor — Go Code Quality Checker")
		fmt.Println()
		return
	}

	face := r.getDoctorFace(score.Score)
	colorize := r.colorizeByScore

	fmt.Println()
	fmt.Printf("  ┌─────┐\n")
	fmt.Printf("  │ %s │   %s %s %s\n", colorize(face[0], score.Score), colorize(fmt.Sprintf("%d", score.Score), score.Score), dim("/ 100"), colorize(score.Label, score.Score))
	fmt.Printf("  │ %s │   %s\n", colorize(face[1], score.Score), r.buildScoreBar(score.Score))
	fmt.Printf("  └─────┘   %s\n", dim("Go Doctor (github.com/go-doctor/go-doctor)"))
	fmt.Println()
}

func (r *Reporter) getDoctorFace(score int) [2]string {
	if score >= scorer.ScoreGoodThreshold {
		return [2]string{"◠ ◠", " ▽ "}
	}
	if score >= scorer.ScoreOKThreshold {
		return [2]string{"• •", " ─ "}
	}
	return [2]string{"x x", " ▽ "}
}

func (r *Reporter) buildScoreBar(score int) string {
	filled := score * barWidth / 100
	empty := barWidth - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return r.colorizeByScore(bar, score)
}

func (r *Reporter) printProjectInfo(info types.ProjectInfo) {
	fmt.Printf("  %s %s\n", bold("Project:"), info.ProjectName)
	fmt.Printf("  %s %s\n", bold("Go Version:"), info.GoVersion)

	frameworkName := string(info.Framework)
	if frameworkName == "" {
		frameworkName = "unknown"
	}
	fmt.Printf("  %s %s\n", bold("Framework:"), frameworkName)
	fmt.Printf("  %s %d\n", bold("Source Files:"), info.SourceFileCount)
}

func (r *Reporter) printDiagnostics(diagnostics []types.Diagnostic) {
	categories := r.groupByCategory(diagnostics)

	for _, category := range categories {
		fmt.Printf("  %s %s\n", bold(string(category.name)), dim(fmt.Sprintf("%d issues", category.count)))

		rules := r.groupByRule(category.diagnostics)
		for _, rule := range rules {
			first := rule.diagnostics[0]
			icon := r.severityIcon(first.Severity)
			ruleID := fmt.Sprintf("%s/%s", first.Plugin, first.Rule)

			if r.verbose {
				fmt.Printf("    %s %s ×%d\n", icon, r.colorizeBySeverity(ruleID, first.Severity), len(rule.diagnostics))
				fmt.Printf("        %s\n", dim(first.Message))
				if first.Help != "" {
					fmt.Printf("        %s%s\n", dim("→ "), dim(first.Help))
				}
				for _, d := range rule.diagnostics {
					relPath := r.relativePath(d.FilePath)
					fmt.Printf("        %s:%d\n", dim(relPath), d.Line)
				}
				fmt.Println()
			} else {
				siteCount := ""
				if len(rule.diagnostics) > 1 {
					siteCount = dim(fmt.Sprintf(" ×%d", len(rule.diagnostics)))
				}
				fmt.Printf("    %s %s%s\n", icon, r.colorizeBySeverity(ruleID, first.Severity), siteCount)
				fmt.Printf("        %s\n", dim(first.Message))
				if first.Help != "" {
					fmt.Printf("        %s\n", dim(first.Help))
				}
				relPath := r.relativePath(first.FilePath)
				if first.Line > 0 {
					fmt.Printf("        %s:%d\n", dim(relPath), first.Line)
				}
			}
		}
		fmt.Println()
	}
}

func (r *Reporter) printSummary(result *types.ScanResult) {
	errorCount := 0
	warningCount := 0
	affectedFiles := make(map[string]bool)

	for _, d := range result.Diagnostics {
		switch d.Severity {
		case types.SeverityError:
			errorCount++
		case types.SeverityWarning:
			warningCount++
		}
		affectedFiles[d.FilePath] = true
	}

	breakdown := scorer.CalculateBreakdown(result.Diagnostics)

	fmt.Printf("  %s %d errors, %d warnings across %d files\n",
		bold("Summary:"),
		errorCount,
		warningCount,
		len(affectedFiles))

	if r.verbose {
		fmt.Printf("  %s Weighted penalty: %.1f (errors: %d, warnings: %d) → Score: %d\n",
			dim("Formula:"),
			breakdown.TotalPenalty,
			breakdown.ErrorCount,
			breakdown.WarningCount,
			breakdown.FinalScore)
		if len(breakdown.CategoryCounts) > 0 {
			fmt.Printf("  %s ", dim("By category:"))
			first := true
			for cat, count := range breakdown.CategoryCounts {
				if !first {
					fmt.Print(dim(", "))
				}
				fmt.Printf("%s: %d", string(cat), count)
				first = false
			}
			fmt.Println()
		}
	}

	fmt.Println()
}

func (r *Reporter) printElapsedTime(ms int64) {
	if ms < 1000 {
		fmt.Printf("  %s %dms\n", dim("Completed in"), ms)
	} else {
		fmt.Printf("  %s %.1fs\n", dim("Completed in"), float64(ms)/1000.0)
	}
	fmt.Println()
}

type categoryGroup struct {
	name        types.Category
	diagnostics []types.Diagnostic
	count       int
}

func (r *Reporter) groupByCategory(diagnostics []types.Diagnostic) []categoryGroup {
	groups := make(map[types.Category][]types.Diagnostic)
	order := []types.Category{}

	for _, d := range diagnostics {
		if _, exists := groups[d.Category]; !exists {
			order = append(order, d.Category)
		}
		groups[d.Category] = append(groups[d.Category], d)
	}

	var result []categoryGroup
	for _, cat := range order {
		result = append(result, categoryGroup{
			name:        cat,
			diagnostics: groups[cat],
			count:       len(groups[cat]),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count
	})

	return result
}

type ruleGroup struct {
	ruleID      string
	diagnostics []types.Diagnostic
}

func (r *Reporter) groupByRule(diagnostics []types.Diagnostic) []ruleGroup {
	groups := make(map[string][]types.Diagnostic)
	order := []string{}

	for _, d := range diagnostics {
		ruleID := fmt.Sprintf("%s/%s", d.Plugin, d.Rule)
		if _, exists := groups[ruleID]; !exists {
			order = append(order, ruleID)
		}
		groups[ruleID] = append(groups[ruleID], d)
	}

	var result []ruleGroup
	for _, id := range order {
		result = append(result, ruleGroup{
			ruleID:      id,
			diagnostics: groups[id],
		})
	}

	sort.Slice(result, func(i, j int) bool {
		iSev := r.severityOrder(result[i].diagnostics[0].Severity)
		jSev := r.severityOrder(result[j].diagnostics[0].Severity)
		if iSev != jSev {
			return iSev < jSev
		}
		return len(result[i].diagnostics) > len(result[j].diagnostics)
	})

	return result
}

func (r *Reporter) severityOrder(s types.Severity) int {
	switch s {
	case types.SeverityError:
		return 0
	case types.SeverityWarning:
		return 1
	}
	return 2
}

func (r *Reporter) printJSON(result *types.ScanResult) {
	type jsonOutput struct {
		SchemaVersion int                    `json:"schemaVersion"`
		Version       string                 `json:"version"`
		OK            bool                   `json:"ok"`
		Directory     string                 `json:"directory"`
		Project       types.ProjectInfo      `json:"project"`
		Diagnostics   []types.Diagnostic     `json:"diagnostics"`
		Score         *types.ScoreResult     `json:"score"`
		Summary       map[string]interface{} `json:"summary"`
		DiffInfo      *types.DiffInfo        `json:"diffInfo,omitempty"`
		CommitInfo    *types.CommitInfo      `json:"commitInfo,omitempty"`
		ElapsedMs     int64                  `json:"elapsedMilliseconds"`
	}

	errorCount := 0
	warningCount := 0
	for _, d := range result.Diagnostics {
		switch d.Severity {
		case types.SeverityError:
			errorCount++
		case types.SeverityWarning:
			warningCount++
		}
	}

	output := jsonOutput{
		SchemaVersion: 1,
		Version:       "0.1.0",
		OK:            len(result.Diagnostics) == 0,
		Directory:     result.Project.RootDirectory,
		Project:       result.Project,
		Diagnostics:   result.Diagnostics,
		Score:         result.Score,
		Summary: map[string]interface{}{
			"errorCount":   errorCount,
			"warningCount": warningCount,
			"totalIssues":  len(result.Diagnostics),
		},
		ElapsedMs: result.ElapsedMs,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	encoder.Encode(output)
}

func (r *Reporter) severityIcon(s types.Severity) string {
	switch s {
	case types.SeverityError:
		return red("✗")
	case types.SeverityWarning:
		return yellow("⚠")
	}
	return "?"
}

func (r *Reporter) colorizeByScore(text string, score int) string {
	if score >= scorer.ScoreGoodThreshold {
		return green(text)
	}
	if score >= scorer.ScoreOKThreshold {
		return yellow(text)
	}
	return red(text)
}

func (r *Reporter) colorizeBySeverity(text string, s types.Severity) string {
	switch s {
	case types.SeverityError:
		return red(text)
	case types.SeverityWarning:
		return yellow(text)
	}
	return text
}

func (r *Reporter) relativePath(filePath string) string {
	if r.rootDir != "" && strings.HasPrefix(filePath, r.rootDir) {
		rel := strings.TrimPrefix(filePath, r.rootDir)
		if strings.HasPrefix(rel, "/") {
			return rel[1:]
		}
		return rel
	}
	return filePath
}

func bold(text string) string {
	return fmt.Sprintf("\033[1m%s\033[0m", text)
}

func dim(text string) string {
	return fmt.Sprintf("\033[2m%s\033[0m", text)
}

func red(text string) string {
	return fmt.Sprintf("\033[31m%s\033[0m", text)
}

func green(text string) string {
	return fmt.Sprintf("\033[32m%s\033[0m", text)
}

func yellow(text string) string {
	return fmt.Sprintf("\033[33m%s\033[0m", text)
}

func FormatDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func (r *Reporter) printDiffInfo(info *types.DiffInfo) {
	fmt.Println()
	fmt.Printf("  %s Diff Mode: %s...%s\n", bold("Branch:"), dim(info.BaseBranch), info.CurrentBranch)

	if len(info.AddedFiles) > 0 {
		fmt.Printf("  %s %d files added\n", green("+"), len(info.AddedFiles))
	}
	if len(info.ModifiedFiles) > 0 {
		fmt.Printf("  %s %d files modified\n", yellow("~"), len(info.ModifiedFiles))
	}
	if len(info.DeletedFiles) > 0 {
		fmt.Printf("  %s %d files deleted\n", red("-"), len(info.DeletedFiles))
	}

	goFileCount := len(info.ChangedFiles)
	fmt.Printf("  %s %d Go files to scan\n", bold("Scanning:"), goFileCount)

	if r.verbose && goFileCount > 0 {
		fmt.Println()
		for _, f := range info.ChangedFiles {
			relPath := r.relativePath(f)
			fmt.Printf("    %s\n", dim(relPath))
		}
	}
}

func (r *Reporter) printCommitInfo(info *types.CommitInfo) {
	fmt.Println()
	fmt.Printf("  %s %s\n", bold("Commit:"), info.CommitHash[:minLen(len(info.CommitHash), 8)])
	if info.Author != "" {
		fmt.Printf("  %s %s\n", bold("Author:"), info.Author)
	}
	if info.Message != "" {
		fmt.Printf("  %s %s\n", bold("Message:"), dim(info.Message))
	}

	goFileCount := len(info.ChangedFiles)
	fmt.Printf("  %s %d Go files changed\n", bold("Scanning:"), goFileCount)

	if r.verbose && goFileCount > 0 {
		fmt.Println()
		for _, f := range info.ChangedFiles {
			relPath := r.relativePath(f)
			fmt.Printf("    %s\n", dim(relPath))
		}
	}
}

func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}
