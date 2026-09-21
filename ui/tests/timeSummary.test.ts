import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  displayValue,
  pricingAnomaly,
  summaryName,
  summaryTotals,
  visibleSummaryRows,
  type TimeSummaryRow,
} from "../src/lib/components/jobs/timeSummary.ts";
const fixtures: Record<string, TimeSummaryRow[]> = JSON.parse(
  readFileSync(new URL("./fixtures/timeSummary.json", import.meta.url), "utf8"),
);

test("combines names and divisions into one readable column", () => {
  assert.equal(summaryName(fixtures.normal[0], "staff"), "Alex Morgan");
  assert.equal(summaryName(fixtures.divisions[0], "divisions"), "E — Environmental");
});

test("shows only anomalies and suppresses repeated fallback notices without a sheet", () => {
  assert.equal(pricingAnomaly(0, 0, false), null);
  assert.equal(pricingAnomaly(2, 0, false), "Estimated");
  assert.equal(pricingAnomaly(2, 0, true), null);
  assert.equal(pricingAnomaly(2, 1, false), "Partial");
  assert.equal(pricingAnomaly(2, 1, true), "Partial");
});

test("distinguishes an unavailable value from a real zero", () => {
  assert.equal(displayValue(fixtures.unpriced[0]), null);
  assert.equal(displayValue(fixtures.partial[0]), 600);
  assert.equal(displayValue({ hours: 0, unpriced_hours: 0, value: 0 }), 0);
});

test("totals rows once and calculates the share before rounding", () => {
  assert.deepEqual(summaryTotals(fixtures.normal), {
    hours: 24,
    value: 2700,
    estimated_hours: 0,
    unpriced_hours: 0,
    percent: 100,
  });
  assert.equal(summaryTotals(fixtures.mixed).estimated_hours, 2);
  assert.equal(summaryTotals(fixtures.partial).percent, null);
  assert.equal(summaryTotals([]).percent, null);
});

test("filters visible totals without changing the full report or share denominator", () => {
  const before = JSON.stringify(fixtures.normal);
  const rows = visibleSummaryRows(
    fixtures.normal,
    "staff",
    [{ column: "name", value: "Alex Morgan" }],
    "name",
    0,
  );
  assert.equal(rows.length, 1);
  assert.equal(summaryTotals(rows).value, 900);
  assert.equal(summaryTotals(rows).percent, 33.3);
  assert.equal(JSON.stringify(fixtures.normal), before);
  assert.equal(summaryTotals([fixtures.partial[1]]).percent, null);
});

test("sorts numeric values in both directions without mutating CSV order", () => {
  assert.deepEqual(
    visibleSummaryRows(fixtures.normal, "staff", [], "value", 1).map((r) => r.value),
    [600, 900, 1200],
  );
  assert.deepEqual(
    visibleSummaryRows(fixtures.normal, "staff", [], "value", -1).map((r) => r.value),
    [1200, 900, 600],
  );
  assert.deepEqual(
    fixtures.normal.map((r) => r.value),
    [900, 1200, 600],
  );
  assert.deepEqual(visibleSummaryRows(fixtures.normal, "staff", [], "value", 0), fixtures.normal);
});

test("keeps missing values last and supports an unavailable-value filter", () => {
  const rows = [fixtures.unpriced[0], fixtures.normal[1]];
  assert.equal(visibleSummaryRows(rows, "staff", [], "value", -1)[1], rows[0]);
  assert.equal(visibleSummaryRows(rows, "staff", [], "value", 1)[1], rows[0]);
  assert.deepEqual(
    visibleSummaryRows(rows, "staff", [{ column: "value", value: null }], "name", 0),
    [rows[0]],
  );
});
