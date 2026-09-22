import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { wipView, type JobWIP } from "../src/lib/components/jobs/wip.ts";
const fixtures: Record<string, JobWIP> = JSON.parse(
  readFileSync(new URL("./fixtures/wip.json", import.meta.url), "utf8"),
);

test("checkboxes select independent factors without changing PO balances", () => {
  for (const [expenses, pos, total] of [
    [true, true, 65000],
    [false, true, 55000],
    [true, false, 50000],
    [false, false, 40000],
  ] as const) {
    const view = wipView(fixtures.normal, expenses, pos);
    assert.equal(view.total, total);
    assert.equal(view.percent, total / 1000);
    assert.equal(view.rows[2].value, 15000);
    assert.equal(view.rows[1].percent, expenses ? 10 : null);
    assert.equal(view.rows[2].percent, pos ? 15 : null);
    assert.equal(view.balance, 100000 - total);
  }
});

test("budget percentages and doughnut shares use different denominators", () => {
  const view = wipView(fixtures.normal, true, true);
  assert.equal(view.rows[0].percent, 40);
  assert.equal(view.rows[0].share, (40000 / 65000) * 100);
  assert.ok(Math.abs(view.rows.reduce((sum, row) => sum + row.share, 0) - 100) < 1e-10);
  assert.equal(view.rows[1].offset, view.rows[0].share);
  assert.equal(view.budgetMarker, 100);
});

test("over-budget totals remain above 100 and the budget marker moves within the bar", () => {
  const view = wipView(fixtures.over, true, true);
  assert.equal(view.percent, 130);
  assert.equal(view.balance, -15000);
  assert.equal(view.budgetMarker, (50000 * 100) / 65000);
  assert.ok(Math.abs(view.rows.reduce((sum, row) => sum + row.width, 0) - 100) < 1e-10);
});

test("incomplete selected factors hide charts and total percentage", () => {
  const time = wipView(fixtures.partial, true, true);
  assert.equal(time.partial, true);
  assert.equal(time.estimated, true);
  assert.equal(time.percent, null);
  assert.equal(time.balance, null);
  assert.equal(time.chartable, false);
  assert.equal(time.rows[0].percent, null);
  assert.equal(time.rows[1].percent, 10);
  assert.equal(wipView(fixtures["unknown-po"], true, true).partial, true);
  assert.equal(wipView(fixtures["unknown-po"], true, false).percent, 50);
  assert.equal(wipView(fixtures["unknown-expense"], false, true).percent, 55);
});

test("zero project values and empty reports never divide by zero", () => {
  assert.equal(wipView(fixtures["zero-project"], true, true).percent, null);
  const empty = wipView(fixtures.empty, true, true);
  assert.equal(empty.percent, 0);
  assert.ok(empty.rows.every((row) => row.share === 0 && row.width === 0));
});

test("negative corrections remain in totals but cannot produce chart segments", () => {
  const view = wipView(fixtures.negative, true, true);
  assert.equal(view.total, 54900);
  assert.equal(view.chartable, false);
  assert.ok(view.rows.every((row) => row.share === 0 && row.width === 0));
});
