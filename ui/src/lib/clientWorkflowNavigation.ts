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

function storedDraft(token: string): DraftEnvelope | null {
  try {
    return JSON.parse(sessionStorage.getItem(prefix + token) || "null");
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
      !/^\/jobs\/(?:add(?:\/(?:from\/)?[a-zA-Z0-9_-]+)?|[a-zA-Z0-9_-]+\/edit)$/.test(url.pathname)
    )
      return null;
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
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
  // Remove unreadable and expired drafts, and earlier drafts for this form: old
  // workspace history entries must not bring a superseded draft back to life.
  for (const key of Object.keys(sessionStorage)) {
    if (!key.startsWith(prefix) || key === prefix + token) continue;
    const previous = storedDraft(key.slice(prefix.length));
    const superseded =
      previous?.kind === kind && previous.user === user && previous.path === draft.path;
    if (!previous || superseded || Date.now() - previous.created > lifetime)
      sessionStorage.removeItem(key);
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

// Select a created, copied, or reused profile only in the originating form.
// Saving the profile does not save the job draft.
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
  const stored = token ? storedDraft(token) : null;
  const formId = url.pathname.split("/")[2];
  const kind = formId === "add" ? "job:new" : `job:${formId}`;
  if (!isUsable(stored, user, url.pathname) || stored.kind !== kind) return null;
  const payload = stored.payload as { item?: Record<string, unknown> } | null;
  if (!payload?.item || payload.item.client !== clientId) return null;
  const updated: DraftEnvelope = {
    ...stored,
    payload: {
      ...payload,
      item: { ...payload.item, invoicing_information: profileId },
      selectedProfile: profileId,
    },
  };
  // Change only the profile selection. A storage error leaves the draft intact.
  try {
    sessionStorage.setItem(prefix + token, JSON.stringify(updated));
  } catch {
    return null;
  }
  return destination;
}

export function clearClientWorkflowDraft(): void {
  if (typeof window === "undefined") return;
  const token = new URL(window.location.href).searchParams.get("client_draft");
  if (!token) return;
  try {
    sessionStorage.removeItem(prefix + token);
  } catch {
    // A blocked browser store must not prevent a successful form save.
  }
}
