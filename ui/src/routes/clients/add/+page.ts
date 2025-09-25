import type { ClientContactsResponse, ClientsRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ClientsPageData } from "$lib/svelte-types";

export const load: PageLoad<ClientsPageData> = async () => {
  const defaultItem = {
    name: "",
    business_development_lead: "",
  };

  const defaultContacts = [] as ClientContactsResponse[];
  return {
    item: { ...defaultItem } as ClientsRecord,
    editing: false,
    id: null,
    client_contacts: defaultContacts,
  };
};
