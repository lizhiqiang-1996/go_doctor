package scanner

import (
	"fmt"
	"time"

	"github.com/lizhiqiang-1996/go_doctor/pkg/analyzer"
	"github.com/lizhiqiang-1996/go_doctor/pkg/config"
	"github.com/lizhiqiang-1996/go_doctor/pkg/git"
	"github.com/lizhiqiang-1996/go_doctor/pkg/project"
	"github.com/lizhiqiang-1996/go_doctor/pkg/scorer"
	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

type Scanner struct {
	rootDir string
	options types.ScanOptions
	cfg     *types.Config
}

func New(rootDir string, options types.ScanOptions) *Scanner {
	cfg := config.Load(rootDir)
	cfg = config.MergeWithDefaults(cfg)

	if options.Lint {
		if cfg.Lint != nil {
			options.Lint = *cfg.Lint
		}
	}
	if options.DeadCode {
		if cfg.DeadCode != nil {
			options.DeadCode = *cfg.DeadCode
		}
	}
	if !options.Verbose {
		if cfg.Verbose != nil {
			options.Verbose = *cfg.Verbose
		}
	}

	return &Scanner{
		rootDir: rootDir,
		options: options,
		cfg:     cfg,
	}
}

func (s *Scanner) Scan() *types.ScanResult {
	start := time.Now()

	projectInfo := project.Discover(s.rootDir)

	ignoreRules := config.GetIgnoreRules(s.cfg)
	ignoreFiles := config.GetIgnoreFiles(s.cfg)

	a := analyzer.NewAnalyzer(ignoreRules, ignoreFiles)

	var allDiagnostics []types.Diagnostic
	var skippedChecks []string

	if s.options.DiffBase != "" {
		return s.scanDiff(start, a, projectInfo)
	}

	if s.options.Commit != "" {
		return s.scanCommit(start, a, projectInfo)
	}

	if s.options.Lint {
		lintDiagnostics := a.AnalyzeDirectory(s.rootDir)
		allDiagnostics = append(allDiagnostics, lintDiagnostics...)
	} else {
		skippedChecks = append(skippedChecks, "lint")
	}

	if s.options.DeadCode {
		deadCodeDiagnostics := a.FindDeadCode(s.rootDir)
		allDiagnostics = append(allDiagnostics, deadCodeDiagnostics...)
	} else {
		skippedChecks = append(skippedChecks, "dead-code")
	}

	var score *types.ScoreResult
	if !s.options.ScoreOnly || len(allDiagnostics) > 0 {
		score = scorer.CalculateWithFileCount(allDiagnostics, projectInfo.SourceFileCount)
	}

	elapsed := time.Since(start)

	return &types.ScanResult{
		Diagnostics:   allDiagnostics,
		Score:         score,
		SkippedChecks: skippedChecks,
		Project:       projectInfo,
		ElapsedMs:     elapsed.Milliseconds(),
	}
}

func (s *Scanner) scanDiff(start time.Time, a *analyzer.Analyzer, projectInfo types.ProjectInfo) *types.ScanResult {
	diffResult, err := git.GetDiffFiles(s.rootDir, s.options.DiffBase)
	if err != nil {
		return &types.ScanResult{
			Diagnostics: []types.Diagnostic{},
			Score:       &types.ScoreResult{Score: 0, Label: "Error"},
			Project:     projectInfo,
			ElapsedMs:   time.Since(start).Milliseconds(),
			DiffInfo: &types.DiffInfo{
				BaseBranch:    s.options.DiffBase,
				CurrentBranch: "unknown",
			},
		}
	}

	mergedBase, _ := git.GetMergeBase(s.rootDir, s.options.DiffBase)
	changedLines := git.GetChangedLines(s.rootDir, mergedBase)

	var allDiagnostics []types.Diagnostic

	if s.options.Lint && len(diffResult.ChangedFiles) > 0 {
		lintDiagnostics := a.AnalyzeFiles(s.rootDir, diffResult.ChangedFiles)
		allDiagnostics = append(allDiagnostics, lintDiagnostics...)
	}

	allDiagnostics = filterDiagnosticsByChangedLines(allDiagnostics, changedLines)

	diffFileCount := len(diffResult.ChangedFiles)
	var score *types.ScoreResult
	if len(allDiagnostics) > 0 {
		score = scorer.CalculateWithFileCount(allDiagnostics, diffFileCount)
	} else {
		score = &types.ScoreResult{Score: 100, Label: "Excellent"}
	}

	elapsed := time.Since(start)

	return &types.ScanResult{
		Diagnostics: allDiagnostics,
		Score:       score,
		Project:     projectInfo,
		ElapsedMs:   elapsed.Milliseconds(),
		DiffInfo: &types.DiffInfo{
			BaseBranch:    diffResult.BaseBranch,
			CurrentBranch: diffResult.CurrentBranch,
			ChangedFiles:  diffResult.ChangedFiles,
			AddedFiles:    diffResult.AddedFiles,
			ModifiedFiles: diffResult.ModifiedFiles,
			DeletedFiles:  diffResult.DeletedFiles,
			ChangedLines:  changedLines,
		},
	}
}

func (s *Scanner) scanCommit(start time.Time, a *analyzer.Analyzer, projectInfo types.ProjectInfo) *types.ScanResult {
	commitResult, err := git.GetCommitFiles(s.rootDir, s.options.Commit)
	if err != nil {
		return &types.ScanResult{
			Diagnostics: []types.Diagnostic{},
			Score:       &types.ScoreResult{Score: 0, Label: "Error"},
			Project:     projectInfo,
			ElapsedMs:   time.Since(start).Milliseconds(),
			CommitInfo: &types.CommitInfo{
				CommitHash: s.options.Commit,
			},
		}
	}

	changedLines := git.GetCommitChangedLines(s.rootDir, s.options.Commit)

	var allDiagnostics []types.Diagnostic

	if s.options.Lint && len(commitResult.ChangedFiles) > 0 {
		lintDiagnostics := a.AnalyzeFiles(s.rootDir, commitResult.ChangedFiles)
		allDiagnostics = append(allDiagnostics, lintDiagnostics...)
	}

	allDiagnostics = filterDiagnosticsByChangedLines(allDiagnostics, changedLines)

	commitFileCount := len(commitResult.ChangedFiles)
	var score *types.ScoreResult
	if len(allDiagnostics) > 0 {
		score = scorer.CalculateWithFileCount(allDiagnostics, commitFileCount)
	} else {
		score = &types.ScoreResult{Score: 100, Label: "Excellent"}
	}

	elapsed := time.Since(start)

	commitMsg := commitResult.Message
	if len(commitMsg) > 80 {
		commitMsg = commitMsg[:80] + "..."
	}

	return &types.ScanResult{
		Diagnostics: allDiagnostics,
		Score:       score,
		Project:     projectInfo,
		ElapsedMs:   elapsed.Milliseconds(),
		CommitInfo: &types.CommitInfo{
			CommitHash:   commitResult.CommitHash,
			Author:       commitResult.Author,
			Message:      commitMsg,
			ChangedFiles: commitResult.ChangedFiles,
			ChangedLines: changedLines,
		},
	}
}

func formatFileList(files []string, rootDir string) string {
	if len(files) == 0 {
		return "(none)"
	}
	result := ""
	for i, f := range files {
		if i >= 10 {
			result += fmt.Sprintf("  ... and %d more\n", len(files)-10)
			break
		}
		relPath := f
		if len(f) > len(rootDir) && f[:len(rootDir)] == rootDir {
			relPath = f[len(rootDir):]
			if len(relPath) > 0 && relPath[0] == '/' {
				relPath = relPath[1:]
			}
		}
		result += fmt.Sprintf("  %s\n", relPath)
	}
	return result
}

func filterDiagnosticsByChangedLines(diagnostics []types.Diagnostic, changedLines map[string][]types.LineRange) []types.Diagnostic {
	if len(changedLines) == 0 {
		return diagnostics
	}

	var filtered []types.Diagnostic
	for _, d := range diagnostics {
		ranges, ok := changedLines[d.FilePath]
		if !ok {
			continue
		}

		inRange := false
		for _, r := range ranges {
			if d.Line >= r.Start && d.Line <= r.End {
				inRange = true
				break
			}
		}

		if inRange {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
