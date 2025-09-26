import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
export type JobNote = {
  id: string;
  created: string;
  note: string;
  author: {
    id: string;
    email: string;
    given_name: string;
    surname: string;
  };
};

export const load: PageLoad = async ({ params }) => {
  try {
    const job = await pb.send(`/api/jobs/${params.jid}/details`, { method: "GET" });
    const notes = (await pb.send(`/api/jobs/${params.jid}/notes`, {
      method: "GET",
    })) as JobNote[];

    return { job, notes };
  } catch (err) {
    console.error(`loading job details: ${err}`);
    throw error(404, `Failed to load job details: ${err}`);
  }
};
