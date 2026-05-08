package routes

import "testing"

func TestLegacyPurchaseOrderNumberPattern(t *testing.T) {
	tests := []struct {
		name     string
		poNumber string
		want     bool
	}{
		{
			name:     "allows 2024 legacy manual PO number",
			poNumber: "2401-5000",
			want:     true,
		},
		{
			name:     "allows 2025 legacy manual PO number",
			poNumber: "2501-5000",
			want:     true,
		},
		{
			name:     "allows 2026 legacy manual PO number",
			poNumber: "2612-5999",
			want:     true,
		},
		{
			name:     "rejects pre-2024 legacy manual PO number",
			poNumber: "2301-5000",
			want:     false,
		},
		{
			name:     "rejects non-5XXX manual PO number",
			poNumber: "2401-4000",
			want:     false,
		},
		{
			name:     "rejects invalid month",
			poNumber: "2413-5000",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := legacyPurchaseOrderNumberPattern.MatchString(tt.poNumber)
			if got != tt.want {
				t.Fatalf("legacyPurchaseOrderNumberPattern.MatchString(%q) = %v, want %v", tt.poNumber, got, tt.want)
			}
		})
	}
}
