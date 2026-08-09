package evaluation

import (
	"strings"
	"testing"
)

func TestDefaultEvaluationDataset(t *testing.T) {
	cases, err := LoadDefaultCases()
	if err != nil {
		t.Fatalf("LoadDefaultCases: %v", err)
	}
	if len(cases) < 20 {
		t.Fatalf("case count = %d, want at least 20", len(cases))
	}

	report := Run(cases, 1)
	if report.Total != len(cases) || len(report.Results) != len(cases) {
		t.Fatalf("unexpected report size: %#v", report)
	}
	if report.ExpectedAccepts == 0 || report.ExpectedRejects == 0 {
		t.Fatal("dataset must contain both acceptance and rejection cases")
	}
	for _, path := range []string{"direct", "extracted", "local_repair"} {
		if report.PathCounts[path] == 0 {
			t.Fatalf("dataset does not exercise path %q", path)
		}
	}
	if report.RecoveryEligible == 0 || report.RecoveredCorrect == 0 {
		t.Fatal("dataset must quantify deterministic recovery behavior")
	}
	if report.Accuracy <= 0 || report.Accuracy > 1 {
		t.Fatalf("accuracy = %f, want value in (0, 1]", report.Accuracy)
	}
}

func TestLoadCasesRejectsInvalidRecord(t *testing.T) {
	_, err := LoadCases(strings.NewReader(`{"id":"missing-fields"}`))
	if err == nil {
		t.Fatal("expected invalid evaluation case to fail")
	}
}

func TestRunNormalizesInvalidIterations(t *testing.T) {
	cases := []Case{
		{
			ID:             "direct",
			Category:       "direct",
			Input:          `1`,
			Schema:         `{"type":"integer"}`,
			ExpectedAccept: true,
			ExpectedJSON:   `1`,
			ExpectedPaths:  []string{"direct"},
		},
	}
	report := Run(cases, 0)
	if report.Iterations != 1 || report.Passed != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
