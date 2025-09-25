import type { ClientsRecord, ClientContactsResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { ClientsPageData } from "$lib/svelte-types";

export const load: PageLoad<ClientsPageData> = async ({ params }) => {
  const defaultItem = {
    name: "",
    business_development_lead: "",
    outstanding_balance: 0,
    outstanding_balance_date: "",
  };
  const defaultContacts = [] as ClientContactsResponse[];
  let item: ClientsRecord;
  try {
    item = await pb.collection("clients").getOne(params.cid);
    const client_contacts = await pb.collection("client_contacts").getFullList({
      filter: `client="${params.cid}"`,
      sort: "given_name",
    });
    return { item, editing: true, id: params.cid, client_contacts };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return {
      item: { ...defaultItem } as ClientsRecord,
      editing: false,
      id: null,
      client_contacts: defaultContacts,
    };
  }
};
