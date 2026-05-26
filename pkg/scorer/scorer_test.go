package scorer

import (
	"testing"

	"github.com/go-doctor/go-doctor/pkg/types"
)

func TestPerfectScore(t *testing.T) {
	result := Calculate([]types.Diagnostic{})
	if result.Score != 100 {
		t.Errorf("Expected score 100, got %d", result.Score)
	}
	if result.Label != "Excellent" {
		t.Errorf("Expected label 'Excellent', got '%s'", result.Label)
	}
}

func TestGoodScore(t *testing.T) {
	diagnostics := []types.Diagnostic{
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
	}
	result := Calculate(diagnostics)
	if result.Score <= 0 {
		t.Errorf("Expected positive score for few warnings, got %d", result.Score)
	}
}

func TestCriticalScore(t *testing.T) {
	diagnostics := make([]types.Diagnostic, 200)
	for i := range diagnostics {
		diagnostics[i] = types.Diagnostic{Severity: types.SeverityError, Category: types.CategorySecurity}
	}
	result := Calculate(diagnostics)
	if result.Label != "Critical" {
		t.Errorf("Expected 'Critical' label for many errors, got '%s'", result.Label)
	}
}

func TestErrorPenalty(t *testing.T) {
	diagnostics := []types.Diagnostic{
		{Severity: types.SeverityError, Category: types.CategoryErrorHandling},
	}
	result := Calculate(diagnostics)
	if result.Score >= 100 {
		t.Errorf("Expected score < 100 for error, got %d", result.Score)
	}
}

func TestSecurityErrorsWeighMore(t *testing.T) {
	securityError := []types.Diagnostic{
		{Severity: types.SeverityError, Category: types.CategorySecurity},
	}
	styleWarning := []types.Diagnostic{
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
	}

	securityScore := Calculate(securityError).Score
	styleScore := Calculate(styleWarning).Score

	if securityScore >= styleScore {
		t.Errorf("Security error should penalize more than style warning: security=%d, style=%d", securityScore, styleScore)
	}
}

func TestScoreBreakdown(t *testing.T) {
	diagnostics := []types.Diagnostic{
		{Severity: types.SeverityError, Category: types.CategoryErrorHandling},
		{Severity: types.SeverityError, Category: types.CategorySecurity},
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
		{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle},
		{Severity: types.SeverityWarning, Category: types.CategoryDeadCode},
	}
	breakdown := CalculateBreakdown(diagnostics)
	if breakdown.ErrorCount != 2 {
		t.Errorf("Expected 2 errors, got %d", breakdown.ErrorCount)
	}
	if breakdown.WarningCount != 3 {
		t.Errorf("Expected 3 warnings, got %d", breakdown.WarningCount)
	}
	if breakdown.TotalDiagnostics != 5 {
		t.Errorf("Expected 5 total diagnostics, got %d", breakdown.TotalDiagnostics)
	}
	if breakdown.TotalPenalty <= 0 {
		t.Errorf("Expected positive total penalty, got %.1f", breakdown.TotalPenalty)
	}
}

func TestWithFileCount(t *testing.T) {
	diagnostics := make([]types.Diagnostic, 100)
	for i := range diagnostics {
		diagnostics[i] = types.Diagnostic{Severity: types.SeverityWarning, Category: types.CategoryCodeStyle}
	}

	smallProject := CalculateWithFileCount(diagnostics, 10)
	largeProject := CalculateWithFileCount(diagnostics, 1000)

	if smallProject.Score >= largeProject.Score {
		t.Errorf("Larger project with same issues should score higher (density-based): small=%d, large=%d", smallProject.Score, largeProject.Score)
	}
}
