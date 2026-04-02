import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";
import type { WorkRecordEntryRow } from "$lib/svelte-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const items = (await pb.send(`/api/work_records/${encodeURIComponent(params.workRecord)}`, {
      method: "GET",
    })) as WorkRecordEntryRow[];

    return {
      items,
      workRecord: params.workRecord,
    };
  } catch (err) {
    console.error(`loading work record details: ${err}`);
    throw error(404, `Failed to load work record details: ${err}`);
  }
};
