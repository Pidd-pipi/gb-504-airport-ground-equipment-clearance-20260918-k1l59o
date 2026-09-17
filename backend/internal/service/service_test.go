package service

import (
	"errors"
	"testing"

	"groundclearance/internal/constants"
	"groundclearance/internal/util"
)

func TestClearanceTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{constants.ClearancePending, constants.ClearanceCleared, true},
		{constants.ClearancePending, constants.ClearanceRestricted, true},
		{constants.ClearancePending, constants.ClearanceRevoked, true},
		{constants.ClearanceCleared, constants.ClearanceRevoked, true},
		{constants.ClearanceRestricted, constants.ClearanceRevoked, true},
		{constants.ClearanceRevoked, constants.ClearanceCleared, false},
		{constants.ClearanceCleared, constants.ClearanceRestricted, false},
	}
	for _, item := range cases {
		if got := allowedClearanceTransition(item.from, item.to); got != item.want {
			t.Fatalf("transition %s -> %s: got %v want %v", item.from, item.to, got, item.want)
		}
	}
}

func TestTurnaroundTransitionRedLines(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{constants.TurnaroundOpen, constants.TurnaroundChecking, true},
		{constants.TurnaroundDecisioned, constants.TurnaroundCompleted, true},
		{constants.TurnaroundOpen, constants.TurnaroundDecisioned, false},
		{constants.TurnaroundChecking, constants.TurnaroundCompleted, false},
		{constants.TurnaroundCompleted, constants.TurnaroundOpen, false},
	}
	for _, item := range cases {
		if got := allowedTurnaroundTransition(item.from, item.to); got != item.want {
			t.Fatalf("turnaround transition %s -> %s: got %v want %v", item.from, item.to, got, item.want)
		}
	}
}

func TestGroundUnitTransitionRedLines(t *testing.T) {
	if allowedUnitTransition(constants.UnitRetired, constants.UnitAvailable) {
		t.Fatal("retired equipment must be terminal")
	}
	if allowedUnitTransition(constants.UnitAvailable, constants.UnitAvailable) {
		t.Fatal("same-state equipment transition must be rejected")
	}
	if !allowedUnitTransition(constants.UnitBlocked, constants.UnitAvailable) {
		t.Fatal("authorized recovery from blocked state must remain possible")
	}
}

func TestNormalizeEvidence(t *testing.T) {
	items, err := normalizeEvidence([]string{" inspection-1.jpg ", "inspection-1.jpg", "meter-2.json"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0] != "inspection-1.jpg" {
		t.Fatalf("unexpected normalized evidence: %#v", items)
	}
	if _, err := normalizeEvidence([]string{"   "}); err == nil {
		t.Fatal("blank evidence must be rejected")
	} else {
		var appErr *util.AppError
		if !errors.As(err, &appErr) || appErr.Code != constants.CodeValidationFailed {
			t.Fatalf("unexpected blank evidence error: %v", err)
		}
	}
}

func TestValidateBatchItems(t *testing.T) {
	results, err := validateBatchItems([]BatchReviewItem{
		{CheckID: 3, Result: constants.CheckFailed},
		{CheckID: 1, Result: constants.CheckPassed},
		{CheckID: 2, Result: constants.CheckPassed},
	})
	if err != nil {
		t.Fatalf("valid batch rejected: %v", err)
	}
	if len(results) != 3 || results[3] != constants.CheckFailed {
		t.Fatalf("unexpected batch results: %#v", results)
	}

	rejected := []struct {
		name  string
		items []BatchReviewItem
	}{
		{"empty", nil},
		{"zero id", []BatchReviewItem{{CheckID: 0, Result: constants.CheckPassed}}},
		{"pending result", []BatchReviewItem{{CheckID: 1, Result: constants.CheckPending}}},
		{"unknown result", []BatchReviewItem{{CheckID: 1, Result: "unknown"}}},
		{"duplicate check", []BatchReviewItem{
			{CheckID: 1, Result: constants.CheckPassed},
			{CheckID: 1, Result: constants.CheckFailed},
		}},
	}
	for _, tc := range rejected {
		if _, err := validateBatchItems(tc.items); err == nil {
			t.Fatalf("batch %s must be rejected", tc.name)
		} else {
			var appErr *util.AppError
			if !errors.As(err, &appErr) || appErr.Code != constants.CodeValidationFailed {
				t.Fatalf("batch %s returned unexpected error: %v", tc.name, err)
			}
		}
	}

	oversized := make([]BatchReviewItem, 101)
	for i := range oversized {
		oversized[i] = BatchReviewItem{CheckID: uint64(i + 1), Result: constants.CheckPassed}
	}
	if _, err := validateBatchItems(oversized); err == nil {
		t.Fatal("batch over 100 items must be rejected")
	}
}

func TestSharedEnums(t *testing.T) {
	if !constants.IsValidUnitState(constants.UnitInspection) || constants.IsValidUnitState("broken") {
		t.Fatal("unit state validation mismatch")
	}
	if !constants.IsValidRiskLevel(constants.RiskCritical) || constants.IsValidRiskLevel("urgent") {
		t.Fatal("risk validation mismatch")
	}
}
