import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params }) => {
  try {
    const expense = await pb.send(`/api/expenses/details/${params.eid}`, {
      method: "GET",
    });

    return { expense };
  } catch (err) {
    console.error(`loading expense details: ${err}`);
    throw error(404, `Failed to load expense details`);
  }
};
