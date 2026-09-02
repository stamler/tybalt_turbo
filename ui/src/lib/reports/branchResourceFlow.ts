export type EmployeeBranchHoursRow = Record<string, string | number>;

export type BranchColumnPair = {
  branch: string;
  jobColumn: string;
  noJobColumn: string;
};

export type BranchFlowEmployee = {
  payrollId: string;
  surname: string;
  givenName: string;
  defaultBranch: string;
  hours: number;
};

export type BranchFlowSummary = {
  branch: string;
  localHours: number;
  staffJobHours: number;
  workHours: number;
  suppliedHours: number;
  receivedHours: number;
  noJobHours: number;
  localWorkShare: number;
  outsideStaffingShare: number;
  noJobShare: number;
  resourceBalance: number;
};

export type BranchResourceFlow = {
  workBranches: string[];
  homeBranches: string[];
  branchColumns: BranchColumnPair[];
  matrix: Record<string, Record<string, number>>;
  staffJobTotals: Record<string, number>;
  workTotals: Record<string, number>;
  noJobTotals: Record<string, number>;
  summaries: Record<string, BranchFlowSummary>;
};

export const noDefaultBranchLabel = "No default branch";

function numberValue(value: string | number | undefined): number {
  const number = typeof value === "number" ? value : Number(value ?? 0);
  return Number.isFinite(number) ? number : 0;
}

function textValue(value: string | number | undefined): string {
  return value === undefined ? "" : String(value);
}

function ratio(numerator: number, denominator: number): number {
  return denominator > 0 ? numerator / denominator : 0;
}

function homeBranch(row: EmployeeBranchHoursRow): string {
  return textValue(row.defaultBranch).trim() || noDefaultBranchLabel;
}

export function getBranchColumnPairs(columns: string[]): BranchColumnPair[] {
  const dynamicColumns = columns.slice(4, -1);
  if (dynamicColumns.length % 2 !== 0) {
    throw new Error("The employee branch-hours columns are not paired.");
  }

  const pairs: BranchColumnPair[] = [];
  for (let index = 0; index < dynamicColumns.length; index += 2) {
    const jobColumn = dynamicColumns[index];
    const noJobColumn = dynamicColumns[index + 1];
    if (noJobColumn !== `${jobColumn}NoJob`) {
      throw new Error(`The no-job column for ${jobColumn} is missing.`);
    }
    pairs.push({ branch: jobColumn, jobColumn, noJobColumn });
  }
  return pairs;
}

export function buildBranchResourceFlow(
  columns: string[],
  rows: EmployeeBranchHoursRow[],
): BranchResourceFlow {
  const branchColumns = getBranchColumnPairs(columns);
  const workBranches = branchColumns.map(({ branch }) => branch);
  const additionalHomeBranches = rows
    .map(homeBranch)
    .filter((branch) => !workBranches.includes(branch));
  const homeBranches = [...workBranches, ...new Set(additionalHomeBranches)].sort((a, b) =>
    a.localeCompare(b),
  );

  const matrix: Record<string, Record<string, number>> = {};
  const staffJobTotals: Record<string, number> = {};
  const workTotals: Record<string, number> = Object.fromEntries(
    workBranches.map((branch) => [branch, 0]),
  );
  const noJobTotals: Record<string, number> = {};

  for (const branch of homeBranches) {
    matrix[branch] = Object.fromEntries(workBranches.map((workBranch) => [workBranch, 0]));
    staffJobTotals[branch] = 0;
    noJobTotals[branch] = 0;
  }

  for (const row of rows) {
    const employeeHomeBranch = homeBranch(row);
    for (const pair of branchColumns) {
      const jobHours = numberValue(row[pair.jobColumn]);
      const noJobHours = numberValue(row[pair.noJobColumn]);
      matrix[employeeHomeBranch][pair.branch] += jobHours;
      staffJobTotals[employeeHomeBranch] += jobHours;
      workTotals[pair.branch] += jobHours;
      noJobTotals[employeeHomeBranch] += noJobHours;
    }
  }

  const summaries: Record<string, BranchFlowSummary> = {};
  for (const branch of homeBranches) {
    const localHours = matrix[branch]?.[branch] ?? 0;
    const staffJobHours = staffJobTotals[branch] ?? 0;
    const workHours = workTotals[branch] ?? 0;
    const suppliedHours = staffJobHours - localHours;
    const receivedHours = workHours - localHours;
    const noJobHours = noJobTotals[branch] ?? 0;
    summaries[branch] = {
      branch,
      localHours,
      staffJobHours,
      workHours,
      suppliedHours,
      receivedHours,
      noJobHours,
      localWorkShare: ratio(localHours, staffJobHours),
      outsideStaffingShare: ratio(receivedHours, workHours),
      noJobShare: ratio(noJobHours, staffJobHours + noJobHours),
      resourceBalance: receivedHours - suppliedHours,
    };
  }

  return {
    workBranches,
    homeBranches,
    branchColumns,
    matrix,
    staffJobTotals,
    workTotals,
    noJobTotals,
    summaries,
  };
}

export function getFlowEmployees(
  rows: EmployeeBranchHoursRow[],
  branchColumns: BranchColumnPair[],
  selectedHomeBranch: string,
  selectedWorkBranch: string | null,
): BranchFlowEmployee[] {
  return rows
    .filter((row) => homeBranch(row) === selectedHomeBranch)
    .map((row) => {
      const hours = selectedWorkBranch
        ? numberValue(row[selectedWorkBranch])
        : branchColumns.reduce((total, pair) => total + numberValue(row[pair.noJobColumn]), 0);
      return {
        payrollId: textValue(row.payrollId),
        surname: textValue(row.surname),
        givenName: textValue(row.givenName),
        defaultBranch: homeBranch(row),
        hours,
      };
    })
    .filter(({ hours }) => hours > 0)
    .sort(
      (left, right) =>
        right.hours - left.hours ||
        left.surname.localeCompare(right.surname) ||
        left.givenName.localeCompare(right.givenName) ||
        left.payrollId.localeCompare(right.payrollId),
    );
}
