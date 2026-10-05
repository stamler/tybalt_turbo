export type ClientWorkspaceTab = "details" | "jobs" | "notes";

export type WorkspaceAddress = {
  address?: string;
  city?: string;
  province_state?: string;
  postal_code?: string;
  country?: string;
  phone?: string;
};
export type WorkspaceContact = WorkspaceAddress & {
  id: string;
  client: string;
  given_name: string;
  surname: string;
  email: string;
  job_count: number;
  profile_count: number;
};
export type WorkspaceInvoicing = {
  created?: string;
  creator?: string;
  creator_name?: string;
  id: string;
  client: string;
  name: string;
  contact: string;
  fax: string;
  invoicing_instructions: string;
  job_count: number;
};
export type WorkspaceClient = WorkspaceAddress & {
  id: string;
  name: string;
  business_development_lead: string;
  lead_given_name: string;
  lead_surname: string;
  outstanding_balance: number;
  outstanding_balance_date: string;
  referencing_jobs_count: number;
  contacts: WorkspaceContact[];
  invoicing_profiles: WorkspaceInvoicing[];
};

export function positivePage(value: string | null): number {
  const number = Number(value);
  return Number.isSafeInteger(number) && number > 0 ? number : 1;
}
export function listPageSize(value: string | null): number {
  const number = Number(value);
  return [10, 20, 50].includes(number) ? number : 20;
}

// Use the same filter and page bounds as DSList to locate a saved record.
export function savedListPosition<T extends { id: string }>(
  items: T[],
  id: string | undefined,
  query: string,
  searchText: (item: T) => string,
  page: number,
  perPage: number,
): { visible: boolean; page: number; clearSearch: boolean } | null {
  if (!id) return null;
  const index = items.findIndex((item) => item.id === id);
  if (index < 0) return null;
  const safePerPage = [10, 20, 50].includes(perPage) ? perPage : 20;
  const filtered = items.filter((item) =>
    searchText(item).toLowerCase().includes(query.toLowerCase()),
  );
  const filteredIndex = filtered.findIndex((item) => item.id === id);
  if (filteredIndex < 0)
    return { visible: false, page: Math.floor(index / safePerPage) + 1, clearSearch: true };
  const targetPage = Math.floor(filteredIndex / safePerPage) + 1;
  const totalPages = Math.max(1, Math.ceil(filtered.length / safePerPage));
  const activePage = Math.min(Math.max(1, Number.isSafeInteger(page) ? page : 1), totalPages);
  return { visible: activePage === targetPage, page: targetPage, clearSearch: false };
}

export function parseClientWorkspaceQuery(query: URLSearchParams) {
  const requestedView = query.get("view");
  return {
    view: requestedView === "proposals" || requestedView === "owner" ? requestedView : "projects",
    contactsQuery: query.get("contacts_q") ?? "",
    contactsPage: positivePage(query.get("contacts_page")),
    contactsPerPage: listPageSize(query.get("contacts_per_page")),
    invoicingQuery: query.get("invoicing_q") ?? "",
    invoicingPage: positivePage(query.get("invoicing_page")),
    invoicingPerPage: listPageSize(query.get("invoicing_per_page")),
    recipient: query.get("recipient") ?? "",
  } as const;
}
export function workspaceHref(
  clientId: string,
  query: URLSearchParams,
  changes: Record<string, string | number | null>,
  tab: ClientWorkspaceTab = "details",
): string {
  const next = new URLSearchParams(query);
  for (const [key, value] of Object.entries(changes)) {
    if (value === null || value === "") next.delete(key);
    else next.set(key, String(value));
  }
  const suffix = next.toString();
  return `/clients/${clientId}/${tab}${suffix ? `?${suffix}` : ""}`;
}
export function safeReturnTo(clientId: string, candidate: string | null | undefined): string {
  const fallback = `/clients/${clientId}/details`;
  if (
    !candidate ||
    (!candidate.startsWith(`${fallback}?`) &&
      candidate !== fallback &&
      !candidate.startsWith(`${fallback}#`))
  )
    return fallback;
  try {
    const parsed = new URL(candidate, "https://workspace.invalid");
    return parsed.origin === "https://workspace.invalid" && parsed.pathname === fallback
      ? `${parsed.pathname}${parsed.search}${parsed.hash}`
      : fallback;
  } catch {
    return fallback;
  }
}
export function contactName(
  contact: Partial<Pick<WorkspaceContact, "given_name" | "surname">> | undefined,
): string {
  return contact
    ? [contact.given_name, contact.surname].filter(Boolean).join(" ")
    : "Contact unavailable";
}
// "1 job", "2 jobs": a count with its noun in the right number.
export function countLabel(count: number, singular: string, plural = `${singular}s`): string {
  return `${count} ${count === 1 ? singular : plural}`;
}
export function contactSearchText(contact: WorkspaceContact): string {
  return [contact.given_name, contact.surname, contact.email, contact.phone]
    .filter(Boolean)
    .join(" ");
}
export function invoicingSearchText(
  profile: WorkspaceInvoicing,
  contacts: WorkspaceContact[],
): string {
  const contact = contacts.find((row) => row.id === profile.contact);
  return [profile.name, profile.invoicing_instructions, contact && contactSearchText(contact)]
    .filter(Boolean)
    .join(" ");
}
export function addressText(address: WorkspaceAddress): string {
  return [
    address.address,
    address.city,
    address.province_state,
    address.postal_code,
    address.country,
  ]
    .filter(Boolean)
    .join(", ");
}

// SvelteKit shallow replacement changes history and page.state, but not page.url.
// Keep the list URL in page.state so links and Back use the current filters.
export function currentWorkspaceUrl(
  clientId: string,
  url: URL,
  state: { clientWorkspaceUrl?: string },
): URL {
  if (url.pathname !== `/clients/${clientId}/details` || !state.clientWorkspaceUrl) return url;
  return new URL(safeReturnTo(clientId, state.clientWorkspaceUrl), url);
}

export function invoicingEmptyMessage(canEdit: boolean, hasContacts: boolean): string {
  const intro = "This client has no invoicing profiles.";
  if (!canEdit) return `${intro} Ask a client maintainer to create one.`;
  return hasContacts
    ? `${intro} Create one using an existing client contact.`
    : `${intro} Add a client contact, then create a profile.`;
}

export function profileCreatorName(
  profile: Pick<WorkspaceInvoicing, "creator" | "creator_name">,
): string {
  return profile.creator ? profile.creator_name?.trim() || "" : "";
}
