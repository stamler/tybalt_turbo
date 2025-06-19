import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";
import type { ClientContactsResponse } from "$lib/pocketbase-types";

export const load: PageLoad = async ({ params }) => {
  try {
    // Fetch all contacts that belong to the current client. The page component
    // handles everything else.
    const contacts = await pb
      .collection("client_contacts")
      .getFullList<ClientContactsResponse>({ filter: `client = "${params.cid}"` });

    return {
      contacts,
    };
  } catch (err) {
    throw error(404, `Failed to load contacts: ${err}`);
  }
};
