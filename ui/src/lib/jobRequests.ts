// Shared request rules. The server checks these rules again on each action.
import type { WorkflowRecord } from "./clientEditors";

export interface RequestDetail {
  request: WorkflowRecord;
  allocations: WorkflowRecord[];
  attachment?: WorkflowRecord;
  related: Record<string, any>;
  review_token: string;
  permissions: Record<string, boolean>;
  enabled?: boolean;
}
export const requestChecklist = {
  fn_agreement_reviewed: "I reviewed the First Nations agreement answer.",
  no_known_conflict: "I am not aware of a company conflict of interest for this project.",
  no_executive_restriction:
    "There is no executive restriction or moratorium that prevents this work.",
  no_duplicate_work:
    "I checked existing jobs and this request does not duplicate work already represented by a job.",
  pdf_sufficient: "The attached PDF is sufficient to authorize this project and its stated value.",
  allocations_complete: "All necessary division allocations are present.",
};
export function emptyChecklist(): Record<string, boolean> {
  return Object.fromEntries(Object.keys(requestChecklist).map((key) => [key, false]));
}
// A contact's email address, or a note that the client has none.
export function contactEmail(contact?: WorkflowRecord): string {
  return contact?.email || (contact?.no_email ? "No email address" : "");
}
export function fullName(record?: WorkflowRecord): string {
  return record?.name || [record?.given_name, record?.surname].filter(Boolean).join(" ");
}
export function address(record?: WorkflowRecord): string {
  return [
    record?.address,
    record?.city,
    record?.province_state,
    record?.postal_code,
    record?.country,
  ]
    .filter(Boolean)
    .join(", ");
}
export function billingAddress(billing?: WorkflowRecord, client?: WorkflowRecord): string {
  return address(billing) || address(client);
}
export function wholeCAD(value: unknown): string {
  return typeof value === "number" && Number.isInteger(value) && value > 0
    ? new Intl.NumberFormat("en-CA", {
        style: "currency",
        currency: "CAD",
        maximumFractionDigits: 0,
      }).format(value)
    : "Not provided";
}
export function printMissing(detail: RequestDetail): string[] {
  const r = detail.request,
    d = detail.related;
  const missing: string[] = [];
  if (!d.client?.name?.trim()) missing.push("Client name");
  if (!r.description?.trim()) missing.push("Project / scope of work");
  if (
    !r.location?.trim() ||
    !/^[23456789CFGHJMPQRVWX]{8}\+[23456789CFGHJMPQRVWX]{2,3}$/i.test(r.location.trim())
  )
    missing.push("Valid Plus Code location");
  for (const [label, contact] of [
    ["Project contact", d.contact],
    ["Billing contact", d.billing_contact],
  ] as const) {
    if (!fullName(contact).trim()) missing.push(`${label} name`);
    if (!contact?.no_email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(contact?.email || ""))
      missing.push(`${label} email`);
  }
  if (!Number.isInteger(r.project_value) || r.project_value <= 0)
    missing.push("Positive whole-dollar project value");
  if (r.time_and_materials_set !== true || typeof r.time_and_materials !== "boolean")
    missing.push("Fee basis");
  if (!d.rate_sheet?.id || d.rate_sheet.active !== true) missing.push("Rate sheet");
  if (
    r.time_and_materials === true &&
    (!Array.isArray(d.rate_sheet_entries) ||
      !d.rate_sheet_entries.length ||
      d.rate_sheet_entries.some(
        (entry: WorkflowRecord) =>
          !Number.isFinite(entry.rate) || !Number.isFinite(entry.overtime_rate),
      ))
  )
    missing.push("Client billing rates");
  return missing;
}
// The server adds this count only for an active job holder while requests are enabled.
export function canViewJobCreateQueue(
  claims: string[],
  enabled: boolean,
  counts: Record<string, number>,
): boolean {
  return enabled && claims.includes("job") && Object.hasOwn(counts, "/job-requests?queue=creation");
}
