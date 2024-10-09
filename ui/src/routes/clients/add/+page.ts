import type { ClientsRecord, ContactsResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ClientsPageData } from "$lib/svelte-types";

export const load: PageLoad<ClientsPageData> = async () => {
  const defaultItem = {
    name: "",
  };

  const defaultContacts = [] as ContactsResponse[];
  return {
    item: { ...defaultItem } as ClientsRecord,
    editing: false,
    id: null,
    contacts: defaultContacts,
  };
};
