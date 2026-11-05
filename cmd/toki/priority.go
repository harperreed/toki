// ABOUTME: Single definition of the todo priorities accepted by --priority
// ABOUTME: Shared by the add and list commands so validation cannot drift

package main

import (
	"errors"
	"slices"
	"strings"
)

// priorityLevels are the accepted todo priorities, in ascending order.
var priorityLevels = []string{"low", "medium", "high"}

// errInvalidPriority is returned for a priority outside priorityLevels.
var errInvalidPriority = errors.New("priority must be low, medium, or high")

// normalizePriority lowercases value and validates it against priorityLevels.
// An empty value means the flag was not supplied, which is always allowed and
// normalizes to the empty string so callers can treat it as unset.
func normalizePriority(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	normalized := strings.ToLower(value)
	if !slices.Contains(priorityLevels, normalized) {
		return "", errInvalidPriority
	}

	return normalized, nil
}
