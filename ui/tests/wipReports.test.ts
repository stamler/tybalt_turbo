import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import { canViewWIPReports, rankWIP, type WIPReportData } from "../src/lib/reports/wipReports.ts";
const data: WIPReportData = JSON.parse(
  readFileSync(new URL("./fixtures/wipReports.json", import.meta.url), "utf8"),
);

test("rank complete values first, keep estimates and percentages over 100, and retain partial rows", () => {
  const ranked = rankWIP(data.items, true, true);
  assert.deepEqual(
    ranked.map((row) => row.data.id),
    ["over", "normal", "empty", "partial", "unknown-po", "unknown-expense"],
  );
  assert.equal(ranked[0].view.percent, 130);
  assert.equal(ranked[1].view.estimated, true);
  assert.equal(ranked[2].view.percent, 0);
  assert.equal(ranked[3].view.percent, null);
});

test("excluding incomplete factors restores ranking without changing source PO balances", () => {
  const original = structuredClone(data);
  for (const [expenses, pos] of [
    [false, true],
    [true, false],
    [false, false],
  ]) {
    const ranked = rankWIP(data.items, expenses, pos);
    const po = ranked.find((row) => row.data.id === "unknown-po")!;
    const expense = ranked.find((row) => row.data.id === "unknown-expense")!;
    assert.equal(po.view.partial, pos);
    assert.equal(expense.view.partial, expenses);
    assert.equal(
      po.view.rows[2].value,
      data.items.find((row) => row.id === "unknown-po")!.po_value,
    );
    assert.equal(ranked.at(-1)!.view.percent, null);
  }
  assert.deepEqual(data, original);
});

test("equal percentages sort by job number, then ID; empty results are valid", () => {
  const row = data.items[0];
  assert.deepEqual(
    rankWIP(
      [
        { ...row, id: "b", number: "2" },
        { ...row, id: "z", number: "1" },
        { ...row, id: "a", number: "2" },
      ],
      true,
      true,
    ).map((row) => row.data.id),
    ["z", "a", "b"],
  );
  assert.deepEqual(rankWIP([], true, true), []);
});

test("management WIP requires a branch manager, kpi, or admin", () => {
  assert.equal(canViewWIPReports([]), false);
  assert.equal(canViewWIPReports(["report", "job", "tapr"]), false);
  assert.equal(canViewWIPReports([], true), true);
  assert.equal(canViewWIPReports(["kpi"]), true);
  assert.equal(canViewWIPReports(["admin"]), true);
});
