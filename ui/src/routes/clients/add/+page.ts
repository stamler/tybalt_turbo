import type { ClientContactsResponse, ClientsRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ClientsPageData } from "$lib/svelte-types";
import { error } from "@sveltejs/kit";
import { get } from "svelte/store";
import { globalStore } from "$lib/stores/global";
export const load: PageLoad<ClientsPageData> = async () => {
  await globalStore.ensureClaims();
  if (!get(globalStore).claims.includes("job"))
    error(403, "The job permission is required to create clients.");
  return {
    item: { name: "", business_development_lead: "" } as ClientsRecord,
    editing: false,
    id: null,
    client_contacts: [] as ClientContactsResponse[],
  };
};
