export type SummaryKind = "staff" | "divisions";
export type SummaryColumn = "name" | "hours" | "value" | "percent";
export type RateSheet = { id: string; name: string; revision: number };

// Keep the API rows intact so CSV exports retain all pricing details.
export type TimeSummaryRow = {
  number: string;
  given_name?: string;
  surname?: string;
  uid?: string;
  division_code?: string;
  division_name?: string;
  hours: number;
  value: number;
  total: number;
  percent: number | null;
  estimated_hours: number;
  total_estimated_hours: number;
  unpriced_hours: number;
  total_unpriced_hours: number;
  rate_sheet_name: string;
  rate_sheet_revision: number | null;
  value_status: string;
  total_status: string;
};

export type SummaryFilter = { column: SummaryColumn; value: string | number | null };

export function summaryName(row: TimeSummaryRow, kind: SummaryKind): string {
  return kind === "staff"
    ? [row.given_name, row.surname].filter(Boolean).join(" ") || "Unknown staff member"
    : [row.division_code, row.division_name].filter(Boolean).join(" — ") || "No division";
}

export function displayValue(row: Pick<TimeSummaryRow, "hours" | "unpriced_hours" | "value">) {
  return row.hours > 0 && row.unpriced_hours >= row.hours ? null : row.value;
}

export function summaryCell(row: TimeSummaryRow, column: SummaryColumn, kind: SummaryKind) {
  if (column === "name") return summaryName(row, kind);
  if (column === "value") return displayValue(row);
  return row[column];
}

export function visibleSummaryRows(
  rows: TimeSummaryRow[],
  kind: SummaryKind,
  filters: SummaryFilter[],
  sortColumn: SummaryColumn,
  sortDirection: 0 | 1 | -1,
) {
  const result = rows.filter((row) =>
    filters.every((filter) => summaryCell(row, filter.column, kind) === filter.value),
  );
  if (!sortDirection) return result;
  return result.sort((a, b) => {
    const left = summaryCell(a, sortColumn, kind);
    const right = summaryCell(b, sortColumn, kind);
    // Missing values stay last in both sort directions.
    if (left === null) return right === null ? 0 : 1;
    if (right === null) return -1;
    return (
      sortDirection *
      (typeof left === "number" && typeof right === "number"
        ? left - right
        : String(left).localeCompare(String(right)))
    );
  });
}

export function summaryTotals(rows: TimeSummaryRow[]) {
  const totals = rows.reduce(
    (sum, row) => ({
      hours: sum.hours + row.hours,
      value: sum.value + row.value,
      estimated_hours: sum.estimated_hours + row.estimated_hours,
      unpriced_hours: sum.unpriced_hours + row.unpriced_hours,
    }),
    { hours: 0, value: 0, estimated_hours: 0, unpriced_hours: 0 },
  );
  // Shares keep the API's denominator: the full job/date range, even when
  // the user filters the table. Do not add rounded row percentages.
  const first = rows[0];
  const percent =
    first && first.total > 0 && first.total_unpriced_hours === 0
      ? Math.round((totals.value * 1000) / first.total) / 10
      : null;
  return { ...totals, percent };
}

export function pricingAnomaly(
  estimatedHours: number,
  unpricedHours: number,
  noRateSheet: boolean,
) {
  if (unpricedHours > 0) return "Partial";
  // One notice above the table explains defaults when no sheet is assigned.
  if (estimatedHours > 0 && !noRateSheet) return "Estimated";
  return null;
}
