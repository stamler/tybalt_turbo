import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { WorkRecordSearchRow } from "$lib/svelte-types";

export const load: PageLoad = async () => {
  try {
    const items = (await pb.send("/api/work_records", {
      method: "GET",
    })) as WorkRecordSearchRow[];

    return { items };
  } catch (error) {
    console.error(`loading work records list: ${error}`);
    return { items: [] as WorkRecordSearchRow[] };
  }
};
