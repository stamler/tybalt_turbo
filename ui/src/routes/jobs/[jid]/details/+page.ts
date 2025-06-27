import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params }) => {
  try {
    const job = await pb.send(`/api/jobs/${params.jid}/details`, { method: "GET" });

    return { job };
  } catch (err) {
    console.error(`loading job details: ${err}`);
    throw error(404, `Failed to load job details: ${err}`);
  }
};
