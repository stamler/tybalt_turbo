import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  filterWIP,
  canViewWIPReports,
  rankWIP,
  type WIPReportData,
} from "../src/lib/reports/wipReports.ts";
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

test("remaining sorts signed dollar balances rather than percentages or magnitudes", () => {
  const base = { ...data.items[0], expense_value: 0, po_value: 0 };
  const rows = [
    { ...base, id: "small-over", project_value: 100, time_value: 200 },
    { ...base, id: "large-over", project_value: 10000, time_value: 15000 },
    { ...base, id: "zero", project_value: 100, time_value: 100 },
    { ...base, id: "under", project_value: 100, time_value: 50 },
    data.items.find((row) => row.id === "partial")!,
  ];
  assert.equal(rankWIP(rows, true, true)[0].data.id, "small-over");
  const ranked = rankWIP(rows, true, true, "remaining");
  assert.deepEqual(
    ranked.map((row) => row.data.id),
    ["large-over", "small-over", "zero", "under", "partial"],
  );
  assert.deepEqual(
    ranked.map((row) => row.view.balance),
    [-5000, -100, 0, 50, null],
  );
});

test("remaining respects selected factors, restores complete rows, and keeps stable ties", () => {
  for (const expenses of [false, true]) {
    for (const pos of [false, true]) {
      const ranked = rankWIP(data.items, expenses, pos, "remaining");
      assert.equal(
        ranked.find((row) => row.data.id === "normal")!.view.balance,
        60000 - (expenses ? 10000 : 0) - (pos ? 15000 : 0),
      );
      assert.equal(ranked.find((row) => row.data.id === "unknown-po")!.view.balance === null, pos);
      assert.equal(
        ranked.find((row) => row.data.id === "unknown-expense")!.view.balance === null,
        expenses,
      );
    }
  }
  const row = data.items[0];
  assert.deepEqual(
    rankWIP(
      [
        { ...row, number: "2", id: "b" },
        { ...row, number: "1", id: "z" },
        { ...row, number: "2", id: "a" },
      ],
      true,
      true,
      "remaining",
    ).map((row) => row.data.id),
    ["z", "a", "b"],
  );
  assert.deepEqual(rankWIP([], true, true, "remaining"), []);
});

test("search matches identity and hidden details, with all words required", () => {
  const row = {
    ...data.items[0],
    client: "Acme Rail",
    manager: "Jane Smith",
    description: "Bridge repair",
    branch: "Thunder Bay",
  };
  for (const term of [
    "26-001",
    "ACME",
    "jane smith",
    "bridge",
    "thunder",
    "  Acme  bridge Jane ",
  ]) {
    assert.deepEqual(filterWIP([row], term), [row]);
  }
  assert.deepEqual(filterWIP([row], "acme missing"), []);
  assert.deepEqual(filterWIP([row], "   "), [row]);
  assert.deepEqual(filterWIP([], "acme"), []);
});
