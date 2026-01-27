import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ url }) => {
  const fromId = url.searchParams.get("from");

  if (!fromId) {
    throw error(400, "Missing 'from' parameter - must specify source rate sheet");
  }

  try {
    // Fetch source rate sheet
    const sourceSheet = await pb.collection("rate_sheets").getOne(fromId);

    // Fetch all source entries with expanded role
    const sourceEntries = await pb.collection("rate_sheet_entries").getFullList({
      filter: `rate_sheet="${fromId}"`,
      expand: "role",
      sort: "expand.role.name",
    });

    return {
      sourceSheet,
      sourceEntries,
    };
  } catch (err) {
    console.error(`Failed to load source rate sheet for copy: ${err}`);
    throw error(404, `Failed to load source rate sheet: ${err}`);
  }
};
