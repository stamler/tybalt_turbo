package routes

import (
	"strings"
	"testing"
	"tybalt/internal/testseed"
)

// Requests and value changes become job allocations, which store whole hours.
func TestRequestAllocationsRequireWholeHours(t *testing.T) {
	app := testseed.NewSeededTestApp(t)
	defer app.Cleanup()
	err := validateRequestAllocationInputs(app, []JobAllocationUpdate{{Division: "fy4i9poneukvq9u", Hours: 0.25}}, true)
	if err == nil || !strings.Contains(err.Error(), "whole numbers") {
		t.Fatalf("part-hour allocation accepted: %v", err)
	}
	if err = validateRequestAllocationInputs(app, []JobAllocationUpdate{{Division: "fy4i9poneukvq9u", Hours: 12}}, true); err != nil {
		t.Fatalf("whole-hour allocation rejected: %v", err)
	}
}
