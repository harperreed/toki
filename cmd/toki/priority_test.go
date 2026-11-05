// ABOUTME: Tests for the shared --priority normalization used by add and list
// ABOUTME: Ensures both commands accept the same values in any case

package main

import (
	"errors"
	"testing"
)

func TestNormalizePriority(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"low", "low", "low", false},
		{"medium", "medium", "medium", false},
		{"high", "high", "high", false},
		{"mixed case low", "LoW", "low", false},
		{"upper case medium", "MEDIUM", "medium", false},
		{"mixed case high", "High", "high", false},
		{"unset stays empty", "", "", false},
		{"unknown value", "urgent", "", true},
		{"padded value is not accepted", " high", "", true},
		{"empty-ish value", " ", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizePriority(tt.value)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizePriority(%q) expected error, got %q", tt.value, got)
				}
				if !errors.Is(err, errInvalidPriority) {
					t.Errorf("expected errInvalidPriority, got %v", err)
				}
				if err.Error() != "priority must be low, medium, or high" {
					t.Errorf("unexpected error message: %q", err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("normalizePriority(%q) unexpected error: %v", tt.value, err)
			}
			if got != tt.want {
				t.Errorf("normalizePriority(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}

func TestAddAndListAcceptSamePriorities(t *testing.T) {
	// Every level is accepted by both commands once normalization is applied,
	// so the two commands cannot drift apart.
	if addCmd.Flags().Lookup("priority") == nil {
		t.Fatal("add command is missing the priority flag")
	}
	if listCmd.Flags().Lookup("priority") == nil {
		t.Fatal("list command is missing the priority flag")
	}

	for _, level := range priorityLevels {
		if _, err := normalizePriority(level); err != nil {
			t.Errorf("priority %q should be accepted, got error: %v", level, err)
		}
	}
}
