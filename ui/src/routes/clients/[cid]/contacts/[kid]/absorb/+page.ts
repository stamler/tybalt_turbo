import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type { ClientContactsResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    // Get the target contact
    const contact = await pb
      .collection("client_contacts")
      .getOne<ClientContactsResponse>(params.kid);

    // Get all contacts for this client
    const contacts = await pb
      .collection("client_contacts")
      .getFullList<ClientContactsResponse>({ filter: `client = "${params.cid}"` });

    return {
      contact,
      contacts,
      params,
    };
  } catch (err) {
    throw error(404, `Contact not found: ${err}`);
  }
};
