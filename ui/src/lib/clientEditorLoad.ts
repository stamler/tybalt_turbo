import type { WorkflowRecord } from "$lib/clientEditors";
import type { ClientContactsResponse } from "$lib/pocketbase-types";
import { error } from "@sveltejs/kit";
import { get } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { globalStore } from "$lib/stores/global";
import { safeReturnTo, type WorkspaceClient } from "$lib/clientWorkspace";

async function loadClientContext(clientId: string): Promise<WorkspaceClient> {
  try {
    return await pb.send<WorkspaceClient>(`/api/clients/${clientId}`, {
      method: "GET",
      requestKey: null,
    });
  } catch (cause: any) {
    const status = Number(cause?.status);
    if (status >= 400 && status <= 599)
      error(status, status === 404 ? "Client not found." : "Unable to load client information.");
    throw cause;
  }
}

export async function loadClientEditor(
  clientId: string,
  url: URL,
  kind: "contact" | "invoicing",
  recordId?: string,
) {
  await globalStore.ensureClaims();
  if (!get(globalStore).claims.includes("job"))
    error(403, "The job permission is required to manage clients.");
  const client = await loadClientContext(clientId);
  const rows = kind === "contact" ? client.contacts : client.invoicing_profiles;
  const record: WorkflowRecord | undefined = recordId
    ? rows.find((row) => row.id === recordId)
    : {};
  if (!record)
    error(
      404,
      kind === "contact"
        ? "Contact not found for this client."
        : "Invoicing profile not found for this client.",
    );
  const candidate = url.searchParams.get("project_contact") || "";
  return {
    client,
    record,
    kind,
    returnTo: safeReturnTo(clientId, url.searchParams.get("return_to")),
    projectContact: client.contacts.some((contact) => contact.id === candidate) ? candidate : "",
  };
}

export async function loadClientMerge(clientId: string, contactId: string, url?: URL) {
  await globalStore.ensureClaims();
  if (!get(globalStore).claims.includes("absorb"))
    error(403, "The absorb permission is required to merge contacts.");
  const client = await loadClientContext(clientId);
  if (!client.contacts.some((contact) => contact.id === contactId))
    error(404, "Contact not found for this client.");
  const contacts = await pb.collection("client_contacts").getFullList<ClientContactsResponse>({
    filter: pb.filter("client = {:client}", { client: clientId }),
  });
  return {
    client,
    contacts,
    returnTo: url?.searchParams.has("return_to")
      ? safeReturnTo(clientId, url.searchParams.get("return_to"))
      : undefined,
  };
}
