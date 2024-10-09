import type { ClientsRecord, ContactsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { ClientsPageData } from "$lib/svelte-types";

export const load: PageLoad<ClientsPageData> = async ({ params }) => {
  const defaultItem = {
    name: "",
  };
  const defaultContacts = [] as ContactsResponse[];
  let item: ClientsRecord;
  try {
    item = await pb.collection("clients").getOne(params.cid);
    const contacts = await pb.collection("contacts").getFullList({
      filter: `client="${params.cid}"`,
      sort: "given_name",
    });
    return { item, editing: true, id: params.cid, contacts };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return {
      item: { ...defaultItem } as ClientsRecord,
      editing: false,
      id: null,
      contacts: defaultContacts,
    };
  }
};
