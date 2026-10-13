package routes

import (
	"strings"
	"testing"
)

func TestNoticeTitleUsesFirstLineOfScope(t *testing.T) {
	for _, tc := range []struct{ description, want string }{
		{"Bridge inspection\nPrepare the final report.", "Bridge inspection"},
		{"\n  \n  Survey   of Lot 12  \nMore", "Survey of Lot 12"},
		{strings.Repeat("a", 61), strings.Repeat("a", 59) + "…"},
		{strings.Repeat("a", 60), strings.Repeat("a", 60)},
		{"", ""},
	} {
		if got := noticeTitle(tc.description); got != tc.want {
			t.Errorf("noticeTitle(%q) = %q; want %q", tc.description, got, tc.want)
		}
	}
}

func TestNoticeWholeCADUsesThousandsSeparators(t *testing.T) {
	for value, want := range map[float64]string{0: "$0", 999: "$999", 1000: "$1,000", 100000: "$100,000", 1234567: "$1,234,567"} {
		if got := noticeWholeCAD(value); got != want {
			t.Errorf("noticeWholeCAD(%v) = %q; want %q", value, got, want)
		}
	}
}
