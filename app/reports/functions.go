package reports

import (
	"encoding/csv"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
)

// convertToCSV takes a slice of dbx.NullStringMap and converts it into a CSV formatted string.
func convertToCSV(report []dbx.NullStringMap, headers []string) (string, error) {
	if len(report) == 0 {
		return "", nil // Return empty string for empty report
	}

	if headers == nil {
		// Collect all unique headers
		headerMap := make(map[string]struct{})
		for _, row := range report {
			for key := range row {
				headerMap[key] = struct{}{}
			}
		}
		headers = make([]string, 0, len(headerMap))
		for key := range headerMap {
			headers = append(headers, key)
		}
		sort.Strings(headers) // Sort headers for consistent column order
	}

	var builder strings.Builder
	writer := csv.NewWriter(&builder)

	// Write header row
	if err := writer.Write(headers); err != nil {
		return "", err // Error writing header
	}

	// Write data rows
	record := make([]string, len(headers)) // Preallocate slice for performance
	for _, rowMap := range report {
		for i, header := range headers {
			if val, ok := rowMap[header]; ok && val.Valid {
				record[i] = val.String
			} else {
				record[i] = "" // Use empty string for NULL or missing values
			}
		}
		if err := writer.Write(record); err != nil {
			// It's possible rows have different numbers of columns if not careful,
			// but csv.Writer handles this by default. Still, good to check.
			return "", err // Error writing record
		}
	}

	writer.Flush() // Ensure all data is written to the builder

	// Check for errors encountered during flushing
	if err := writer.Error(); err != nil {
		return "", err
	}

	return builder.String(), nil
}
