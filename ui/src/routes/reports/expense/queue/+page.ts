import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { ExpenseCommitQueueRow } from "$lib/svelte-types";

export const load: PageLoad = async () => {
  // Fetch all submitted, uncommitted expenses shown in the commit queue.
  const items = (await pb.send("/api/expenses/commit_queue", {
    method: "GET",
  })) as ExpenseCommitQueueRow[];
  return { items };
};
