package routes

import _ "embed"

//go:embed job_priced_time_entries.sql
var jobPricedTimeEntriesQuery string

// JobTimeValuation identifies employee-rate estimates and unpriced hours in
// the API and CSV exports. Percent is null for incomplete or zero totals.
type JobTimeValuation struct {
	RateSheetName       string   `db:"rate_sheet_name" json:"rate_sheet_name"`
	RateSheetRevision   *int     `db:"rate_sheet_revision" json:"rate_sheet_revision"`
	Value               float64  `db:"value" json:"value"`
	Total               float64  `db:"total" json:"total"`
	Percent             *float64 `db:"percent" json:"percent"`
	EstimatedHours      float64  `db:"estimated_hours" json:"estimated_hours"`
	TotalEstimatedHours float64  `db:"total_estimated_hours" json:"total_estimated_hours"`
	UnpricedHours       float64  `db:"unpriced_hours" json:"unpriced_hours"`
	TotalUnpricedHours  float64  `db:"total_unpriced_hours" json:"total_unpriced_hours"`
	ValueStatus         string   `db:"value_status" json:"value_status"`
	TotalStatus         string   `db:"total_status" json:"total_status"`
}
