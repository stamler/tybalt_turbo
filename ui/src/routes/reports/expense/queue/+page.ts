import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async () => {
  // Fetch all uncommitted submitted/approved expenses
  const items = await pb.send("/api/expenses/tracking", { method: "GET" });
  return { items } as any;
};
