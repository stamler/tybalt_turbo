import assert from "node:assert/strict";
import test from "node:test";
import {
  buildBranchResourceFlow,
  getBranchColumnPairs,
  getFlowEmployees,
  noDefaultBranchLabel,
  type EmployeeBranchHoursRow,
} from "../src/lib/reports/branchResourceFlow.ts";

const columns = [
  "payrollId",
  "surname",
  "givenName",
  "defaultBranch",
  "Alpha",
  "AlphaNoJob",
  "Beta",
  "BetaNoJob",
  "total",
];

const rows: EmployeeBranchHoursRow[] = [
  {
    payrollId: "1",
    surname: "Able",
    givenName: "Avery",
    defaultBranch: "Alpha",
    Alpha: 10,
    AlphaNoJob: 2,
    Beta: 5,
    BetaNoJob: 1,
    total: 18,
  },
  {
    payrollId: "2",
    surname: "Baker",
    givenName: "Bailey",
    defaultBranch: "Alpha",
    Alpha: 0,
    AlphaNoJob: 1,
    Beta: 5,
    BetaNoJob: 0,
    total: 6,
  },
  {
    payrollId: "3",
    surname: "Clark",
    givenName: "Casey",
    defaultBranch: "Beta",
    Alpha: 4,
    AlphaNoJob: 0,
    Beta: 6,
    BetaNoJob: 2,
    total: 12,
  },
];

test("builds branch flows, totals, shares, and resource balances", () => {
  const flow = buildBranchResourceFlow(columns, rows);

  assert.deepEqual(flow.workBranches, ["Alpha", "Beta"]);
  assert.deepEqual(flow.homeBranches, ["Alpha", "Beta"]);
  assert.deepEqual(flow.matrix, {
    Alpha: { Alpha: 10, Beta: 10 },
    Beta: { Alpha: 4, Beta: 6 },
  });
  assert.deepEqual(flow.staffJobTotals, { Alpha: 20, Beta: 10 });
  assert.deepEqual(flow.workTotals, { Alpha: 14, Beta: 16 });
  assert.deepEqual(flow.noJobTotals, { Alpha: 4, Beta: 2 });

  assert.deepEqual(flow.summaries.Alpha, {
    branch: "Alpha",
    localHours: 10,
    staffJobHours: 20,
    workHours: 14,
    suppliedHours: 10,
    receivedHours: 4,
    noJobHours: 4,
    localWorkShare: 0.5,
    outsideStaffingShare: 4 / 14,
    noJobShare: 4 / 24,
    resourceBalance: -6,
  });
  assert.deepEqual(flow.summaries.Beta, {
    branch: "Beta",
    localHours: 6,
    staffJobHours: 10,
    workHours: 16,
    suppliedHours: 4,
    receivedHours: 10,
    noJobHours: 2,
    localWorkShare: 0.6,
    outsideStaffingShare: 10 / 16,
    noJobShare: 2 / 12,
    resourceBalance: 6,
  });
});

test("returns contributing employees for a flow and for no-job hours", () => {
  const pairs = getBranchColumnPairs(columns);

  assert.deepEqual(getFlowEmployees(rows, pairs, "Alpha", "Beta"), [
    {
      payrollId: "1",
      surname: "Able",
      givenName: "Avery",
      defaultBranch: "Alpha",
      hours: 5,
    },
    {
      payrollId: "2",
      surname: "Baker",
      givenName: "Bailey",
      defaultBranch: "Alpha",
      hours: 5,
    },
  ]);
  assert.deepEqual(getFlowEmployees(rows, pairs, "Alpha", null), [
    {
      payrollId: "1",
      surname: "Able",
      givenName: "Avery",
      defaultBranch: "Alpha",
      hours: 3,
    },
    {
      payrollId: "2",
      surname: "Baker",
      givenName: "Bailey",
      defaultBranch: "Alpha",
      hours: 1,
    },
  ]);
});

test("keeps employees without a default branch visible", () => {
  const flow = buildBranchResourceFlow(columns, [
    {
      ...rows[0],
      defaultBranch: "",
    },
  ]);

  assert.ok(flow.homeBranches.includes(noDefaultBranchLabel));
  assert.equal(flow.matrix[noDefaultBranchLabel].Alpha, 10);
  assert.equal(flow.noJobTotals[noDefaultBranchLabel], 3);
});

test("rejects an invalid dynamic column contract", () => {
  assert.throws(
    () =>
      getBranchColumnPairs([
        "payrollId",
        "surname",
        "givenName",
        "defaultBranch",
        "Alpha",
        "WrongNoJob",
        "total",
      ]),
    /no-job column for Alpha is missing/,
  );
});
