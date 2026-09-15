package main

import (
	"testing"
)

func TestStatusTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from        AppStatus
		to          AppStatus
		expectError bool
	}{
		{"DRAFT to SUBMITTED", StatusDraft, StatusSubmitted, false},
		{"SUBMITTED to APPROVED", StatusSubmitted, StatusApproved, false},
		{"SUBMITTED to REJECTED", StatusSubmitted, StatusRejected, false},
		{"DRAFT directly to APPROVED", StatusDraft, StatusApproved, true},
		{"DRAFT directly to REJECTED", StatusDraft, StatusRejected, true},
		{"APPROVED to DRAFT", StatusApproved, StatusDraft, true},
		{"APPROVED to SUBMITTED", StatusApproved, StatusSubmitted, true},
		{"APPROVED to REJECTED", StatusApproved, StatusRejected, true},
		{"REJECTED to APPROVED", StatusRejected, StatusApproved, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStatusTransition(tc.from, tc.to)
			if tc.expectError && err == nil {
				t.Fatalf("expected error from %s to %s, got nil", tc.from, tc.to)
			}
			if !tc.expectError && err != nil {
				t.Fatalf("did not expect error from %s to %s, got: %v", tc.from, tc.to, err)
			}
		})
	}
}
