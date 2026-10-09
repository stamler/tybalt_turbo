// Client, contact, and invoicing profile editors work with loosely typed records.
export type WorkflowRecord = Record<string, any>;

export const postalFields = {
  address: "Street",
  city: "City",
  province_state: "Province / state",
  postal_code: "Postal code",
  country: "Country",
};
export const contactFields = [
  "given_name",
  "surname",
  "email",
  "phone",
  ...Object.keys(postalFields),
];
export const invoicingFields = ["name", "contact", "billing_name", "fax", "invoicing_instructions"];
export const clientFields = [
  "name",
  "alias",
  "business_development_lead",
  "phone",
  ...Object.keys(postalFields),
];
export type FieldErrors = Record<string, { message: string }>;

export function editableFields(record: WorkflowRecord, fields: string[]): WorkflowRecord {
  return Object.fromEntries(fields.map((key) => [key, record[key] || ""]));
}
export function validateContact(record: WorkflowRecord): FieldErrors {
  const errors: FieldErrors = {};
  for (const key of ["given_name", "surname"]) {
    if (!String(record[key] || "").trim()) errors[key] = { message: "This field is required." };
  }
  if (record.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(record.email.trim())) {
    errors.email = { message: "Enter a valid email address." };
  }
  return errors;
}
export function contactInUse(record: WorkflowRecord): boolean {
  return [record.job_count, record.profile_count].some((value) => Number(value) > 0);
}
// Combine an API error message with its field messages for display beside a form.
export function workflowError(error: any): string {
  const body = error?.data ?? error?.response;
  const details = Object.entries(body?.data ?? {}).map(
    ([key, value]: [string, any]) => `${key.replaceAll("_", " ")}: ${value?.message ?? value}`,
  );
  return [body?.message ?? error?.message ?? "The action failed.", ...details].join(" ");
}
export function savedReturnTo(returnTo: string, kind: "contact" | "invoicing", id: string): string {
  const url = new URL(returnTo, "https://workspace.invalid");
  url.searchParams.delete("edit");
  url.searchParams.delete("saved_contact");
  url.searchParams.delete("saved_invoicing");
  url.searchParams.set(`saved_${kind}`, id);
  url.hash = kind === "contact" ? "contacts" : "invoicing";
  return url.pathname + url.search + url.hash;
}
