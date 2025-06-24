import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { TimeSheetTallyQueryRow } from "$lib/utilities";

export const load: PageLoad = async () => {
  try {
    const tallies: TimeSheetTallyQueryRow[] = await pb.send(
      "/api/time_sheets/tallies/approved",
      { method: "GET" },
    );
    return { items: tallies };
  } catch (error) {
    console.error("Failed to load approved time sheets:", error);
    return { items: [] };
  }
}; 