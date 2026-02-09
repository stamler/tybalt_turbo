package routes

import "testing"

func TestAllocationsDiffer(t *testing.T) {
	t.Parallel()

	base := []existingJobAllocation{
		{Division: "div-a", Hours: 8},
		{Division: "div-b", Hours: 4.5},
	}

	tests := []struct {
		name     string
		incoming []JobAllocationUpdate
		wantDiff bool
	}{
		{
			name: "same allocations different order",
			incoming: []JobAllocationUpdate{
				{Division: "div-b", Hours: 4.5},
				{Division: "div-a", Hours: 8},
			},
			wantDiff: false,
		},
		{
			name: "hours changed",
			incoming: []JobAllocationUpdate{
				{Division: "div-a", Hours: 7.5},
				{Division: "div-b", Hours: 4.5},
			},
			wantDiff: true,
		},
		{
			name: "division removed",
			incoming: []JobAllocationUpdate{
				{Division: "div-a", Hours: 8},
			},
			wantDiff: true,
		},
		{
			name: "division added",
			incoming: []JobAllocationUpdate{
				{Division: "div-a", Hours: 8},
				{Division: "div-b", Hours: 4.5},
				{Division: "div-c", Hours: 2},
			},
			wantDiff: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := allocationsDiffer(base, tc.incoming)
			if got != tc.wantDiff {
				t.Fatalf("allocationsDiffer() = %v, want %v", got, tc.wantDiff)
			}
		})
	}
}
