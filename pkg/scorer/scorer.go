package scorer

import (
	"math"

	"github.com/lizhiqiang-1996/go_doctor/pkg/types"
)

const (
	PerfectScore       = 100
	ScoreGoodThreshold = 75
	ScoreOKThreshold   = 50
)

type categoryWeight struct {
	errorPenalty   float64
	warningPenalty float64
}

var categoryWeights = map[types.Category]categoryWeight{
	types.CategoryErrorHandling: {errorPenalty: 5, warningPenalty: 2},
	types.CategorySecurity:      {errorPenalty: 5, warningPenalty: 2},
	types.CategoryConcurrency:   {errorPenalty: 4, warningPenalty: 1.5},
	types.CategoryCorrectness:   {errorPenalty: 3, warningPenalty: 1},
	types.CategoryPerformance:   {errorPenalty: 2, warningPenalty: 0.5},
	types.CategoryCodeStyle:     {errorPenalty: 1, warningPenalty: 0.2},
	types.CategoryDeadCode:      {errorPenalty: 1, warningPenalty: 0.1},
	types.CategoryAICode:        {errorPenalty: 3, warningPenalty: 1.5},
}

func Calculate(diagnostics []types.Diagnostic) *types.ScoreResult {
	return CalculateWithFileCount(diagnostics, 0)
}

func CalculateWithFileCount(diagnostics []types.Diagnostic, fileCount int) *types.ScoreResult {
	if len(diagnostics) == 0 {
		return &types.ScoreResult{
			Score: PerfectScore,
			Label: "Excellent",
		}
	}

	totalPenalty := 0.0

	for _, d := range diagnostics {
		weight, ok := categoryWeights[d.Category]
		if !ok {
			weight = categoryWeight{errorPenalty: 3, warningPenalty: 1}
		}

		switch d.Severity {
		case types.SeverityError:
			totalPenalty += weight.errorPenalty
		case types.SeverityWarning:
			totalPenalty += weight.warningPenalty
		}
	}

	if fileCount > 0 {
		perFilePenalty := totalPenalty / float64(fileCount)
		cappedFiles := math.Min(float64(fileCount), 100)
		totalPenalty = perFilePenalty * cappedFiles
		totalPenalty = math.Log1p(totalPenalty) * 13
	}

	adjustedPenalty := math.Min(totalPenalty, float64(PerfectScore))

	score := PerfectScore - int(adjustedPenalty)
	if score < 0 {
		score = 0
	}

	return &types.ScoreResult{
		Score: score,
		Label: scoreToLabel(score),
	}
}

func scoreToLabel(score int) string {
	if score >= ScoreGoodThreshold {
		return "Good"
	}
	if score >= ScoreOKThreshold {
		return "Needs Work"
	}
	return "Critical"
}

type ScoreBreakdown struct {
	TotalDiagnostics int                    `json:"totalDiagnostics"`
	ErrorCount       int                    `json:"errorCount"`
	WarningCount     int                    `json:"warningCount"`
	CategoryCounts   map[types.Category]int `json:"categoryCounts"`
	TotalPenalty     float64                `json:"totalPenalty"`
	FinalScore       int                    `json:"finalScore"`
}

func CalculateBreakdown(diagnostics []types.Diagnostic) ScoreBreakdown {
	errorCount := 0
	warningCount := 0
	categoryCounts := make(map[types.Category]int)
	totalPenalty := 0.0

	for _, d := range diagnostics {
		categoryCounts[d.Category]++

		switch d.Severity {
		case types.SeverityError:
			errorCount++
		case types.SeverityWarning:
			warningCount++
		}

		weight, ok := categoryWeights[d.Category]
		if !ok {
			weight = categoryWeight{errorPenalty: 3, warningPenalty: 1}
		}

		switch d.Severity {
		case types.SeverityError:
			totalPenalty += weight.errorPenalty
		case types.SeverityWarning:
			totalPenalty += weight.warningPenalty
		}
	}

	score := PerfectScore - int(math.Min(totalPenalty, float64(PerfectScore)))
	if score < 0 {
		score = 0
	}

	return ScoreBreakdown{
		TotalDiagnostics: len(diagnostics),
		ErrorCount:       errorCount,
		WarningCount:     warningCount,
		CategoryCounts:   categoryCounts,
		TotalPenalty:     totalPenalty,
		FinalScore:       score,
	}
}
