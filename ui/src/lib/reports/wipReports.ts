import { wipView, type JobWIP } from "../components/jobs/wip.ts";

export type WIPMode = "my" | "branch" | "division";
export type WIPReportRow = JobWIP & {
  id: string;
  number: string;
  description: string;
  client: string;
  manager: string;
  branch_id: string;
  branch: string;
};
export type WIPReportData = {
  items: WIPReportRow[];
  excluded_project_value: number;
  as_of: string;
};

export function canViewWIPReports(claims: string[], isBranchManager = false): boolean {
  return isBranchManager || claims.includes("kpi") || claims.includes("admin");
}

export function rankWIP(rows: WIPReportRow[], includeExpenses: boolean, includePOs: boolean) {
  return rows
    .map((data) => ({ data, view: wipView(data, includeExpenses, includePOs) }))
    .sort((a, b) => {
      if (a.view.percent === null && b.view.percent !== null) return 1;
      if (b.view.percent === null && a.view.percent !== null) return -1;
      return (
        (b.view.percent ?? 0) - (a.view.percent ?? 0) ||
        a.data.number.localeCompare(b.data.number) ||
        a.data.id.localeCompare(b.data.id)
      );
    });
}
