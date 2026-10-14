// Keep a project form while its user maintains shared client information.
// Only an explicit return token can restore a draft, in the same browser tab.
type DraftEnvelope = {
  kind: string;
  user: string;
  path: string;
  created: number;
  payload: unknown;
};

const prefix = "client-workspace-draft:";
const lifetime = 4 * 60 * 60 * 1000;
// The in-memory copy also keeps a selected PDF while navigating in this tab.
// Session storage keeps only text, so a browser reload requires reselecting it.
const memory = new Map<string, DraftEnvelope>();

function storedDraft(token: string): DraftEnvelope | null {
  try {
    return memory.get(token) ?? JSON.parse(sessionStorage.getItem(prefix + token) || "null");
  } catch {
    return null;
  }
}

// A draft can be restored only for the user and form that stored it, before it expires.
function isUsable(draft: DraftEnvelope | null, user: string, path: string): draft is DraftEnvelope {
  const age = Date.now() - Number(draft?.created);
  return !!draft && draft.user === user && draft.path === path && age >= 0 && age <= lifetime;
}

export function safeWorkflowReturn(value: string | null | undefined): string | null {
  if (!value || !value.startsWith("/") || value.startsWith("//") || /[\\\r\n]/.test(value))
    return null;
  try {
    const url = new URL(value, "https://workspace.invalid");
    if (url.origin !== "https://workspace.invalid") return null;
    if (
      !/^\/job-requests\/(?:new|[a-zA-Z0-9_-]+)$/.test(url.pathname) &&
      !/^\/jobs\/(?:add(?:\/(?:from\/)?[a-zA-Z0-9_-]+)?|[a-zA-Z0-9_-]+\/edit)$/.test(url.pathname)
    )
      return null;
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
}

// Name the form a workflow return leads back to.
export function workflowFormName(workflowReturn: string): string {
  if (workflowReturn.startsWith("/job-requests/")) return "project request";
  return "job form";
}

export type ClientWorkflowTarget = "workspace" | "contact" | "invoicing";

export function keepClientWorkflowDraft(
  kind: string,
  user: string,
  payload: unknown,
  clientId: string,
  projectContact: string,
  target: ClientWorkflowTarget = "workspace",
  replaceSource?: (url: string) => void,
): string {
  const current = new URL(window.location.href);
  current.searchParams.delete("client_draft");
  const source = safeWorkflowReturn(`${current.pathname}${current.search}${current.hash}`);
  if (!source) throw new Error("This form cannot return from the client workspace.");
  const token = crypto.randomUUID();
  const draft: DraftEnvelope = {
    kind,
    user,
    path: current.pathname,
    created: Date.now(),
    payload,
  };
  // Save the draft for reloads and same-tab navigation.
  sessionStorage.setItem(prefix + token, JSON.stringify(draft));
  for (const [key, previous] of memory) {
    if (previous.kind === kind && previous.user === user && previous.path === draft.path)
      memory.delete(key);
  }
  memory.set(token, draft);
  // Remove unreadable and expired drafts, and earlier drafts for this form: old
  // workspace history entries must not bring a superseded draft back to life.
  for (const key of Object.keys(sessionStorage)) {
    if (!key.startsWith(prefix) || key === prefix + token) continue;
    const previous = storedDraft(key.slice(prefix.length));
    const superseded =
      previous?.kind === kind && previous.user === user && previous.path === draft.path;
    if (!previous || superseded || Date.now() - previous.created > lifetime) {
      sessionStorage.removeItem(key);
      memory.delete(key.slice(prefix.length));
    }
  }
  current.searchParams.set("client_draft", token);
  const formUrl = `${current.pathname}${current.search}${current.hash}`;
  replaceSource?.(formUrl);
  const query = new URLSearchParams({
    workflow_return: formUrl,
  });
  if (projectContact) query.set("project_contact", projectContact);
  const workspace = `/clients/${encodeURIComponent(clientId)}/details?${query}`;
  if (target === "workspace") return workspace;
  const section = target === "contact" ? "contacts" : "invoicing";
  const editorQuery = new URLSearchParams({ return_to: workspace });
  if (projectContact) editorQuery.set("project_contact", projectContact);
  return `/clients/${encodeURIComponent(clientId)}/${section}/add?${editorQuery}`;
}

export function readClientWorkflowDraft<T>(kind: string, user: string): T | null {
  if (typeof window === "undefined") return null;
  const url = new URL(window.location.href);
  const token = url.searchParams.get("client_draft");
  const stored = token ? storedDraft(token) : null;
  return isUsable(stored, user, url.pathname) && stored.kind === kind
    ? (stored.payload as T)
    : null;
}

// The draft kind and payload field that hold the profile for each originating form.
function draftTarget(pathname: string): { kind: string; field: string } {
  const [, section, id] = pathname.split("/");
  if (section === "job-requests") return { kind: `request:${id}`, field: "item" };
  if (id === "add") return { kind: "job:new", field: "item" };
  return { kind: `job:${id}`, field: "item" };
}

// Select a created, copied, or reused profile only in the originating form.
// Saving the profile does not save the job, request, or value-change draft.
export function selectClientWorkflowProfile(
  workflowReturn: string | null | undefined,
  user: string,
  clientId: string,
  profileId: string,
): string | null {
  const destination = safeWorkflowReturn(workflowReturn);
  if (!destination || !user || !clientId || !profileId || typeof window === "undefined")
    return null;
  const url = new URL(destination, "https://workspace.invalid");
  const token = url.searchParams.get("client_draft");
  if (!token) return null;
  const stored = storedDraft(token);
  const { kind, field } = draftTarget(url.pathname);
  if (!isUsable(stored, user, url.pathname) || stored.kind !== kind) return null;
  const payload = stored.payload as Record<string, unknown> | null;
  const item = payload?.[field] as Record<string, unknown> | undefined;
  if (!item || item.client !== clientId) return null;
  const updated: DraftEnvelope = {
    ...stored,
    payload: {
      ...payload,
      [field]: { ...item, invoicing_information: profileId },
      selectedProfile: profileId,
    },
  };
  // Change only the profile selection. A storage error leaves the draft intact.
  try {
    sessionStorage.setItem(prefix + token, JSON.stringify(updated));
  } catch {
    return null;
  }
  memory.set(token, updated);
  return destination;
}

export function clearClientWorkflowDraft(): void {
  if (typeof window === "undefined") return;
  const token = new URL(window.location.href).searchParams.get("client_draft");
  if (!token) return;
  memory.delete(token);
  try {
    sessionStorage.removeItem(prefix + token);
  } catch {
    // A blocked browser store must not prevent a successful form save.
  }
}
