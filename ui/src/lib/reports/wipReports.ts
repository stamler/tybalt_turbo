import { wipView, type JobWIP } from "../components/jobs/wip.ts";

export type WIPMode = "my" | "branch" | "division";
export type WIPSort = "percent" | "remaining";
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

export function rankWIP(
  rows: WIPReportRow[],
  includeExpenses: boolean,
  includePOs: boolean,
  sort: WIPSort = "percent",
) {
  return rows
    .map((data) => ({ data, view: wipView(data, includeExpenses, includePOs) }))
    .sort((a, b) => {
      if (a.view.percent === null && b.view.percent !== null) return 1;
      if (b.view.percent === null && a.view.percent !== null) return -1;
      const difference =
        sort === "remaining"
          ? (a.view.balance ?? 0) - (b.view.balance ?? 0)
          : (b.view.percent ?? 0) - (a.view.percent ?? 0);
      return (
        difference ||
        a.data.number.localeCompare(b.data.number) ||
        a.data.id.localeCompare(b.data.id)
      );
    });
}

// Match every search word across the visible identity and project details.
export function filterWIP(rows: WIPReportRow[], search: string): WIPReportRow[] {
  const words = search.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean);
  return rows.filter((row) => {
    const text = [row.number, row.client, row.description, row.manager, row.branch]
      .join(" ")
      .toLocaleLowerCase();
    return words.every((word) => text.includes(word));
  });
}
