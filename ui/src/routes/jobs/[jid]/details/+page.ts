import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
import type { ClientNote } from "$lib/types/notes";

export const load: PageLoad = async ({ params }) => {
  try {
    const job = await pb.send(`/api/jobs/${params.jid}/details`, { method: "GET" });
    const notes = (await pb.send(`/api/jobs/${params.jid}/notes`, {
      method: "GET",
    })) as ClientNote[];

    return { job, notes };
  } catch (err) {
    console.error(`loading job details: ${err}`);
    throw error(404, `Failed to load job details: ${err}`);
  }
};
