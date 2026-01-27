import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ url }) => {
  const fromId = url.searchParams.get("from");
  const reviseId = url.searchParams.get("revise");
  const sourceId = fromId || reviseId;
  const isRevising = !!reviseId;

  if (!sourceId) {
    throw error(400, "Missing 'from' or 'revise' parameter - must specify source rate sheet");
  }

  try {
    // Fetch source rate sheet
    const sourceSheet = await pb.collection("rate_sheets").getOne(sourceId);

    // Fetch all source entries with expanded role
    const sourceEntries = await pb.collection("rate_sheet_entries").getFullList({
      filter: `rate_sheet="${sourceId}"`,
      expand: "role",
    });

    // For revise mode, find the next revision number
    let nextRevision = 0;
    if (isRevising) {
      const existingSheets = await pb.collection("rate_sheets").getFullList({
        filter: `name = "${sourceSheet.name}"`,
        sort: "-revision",
        fields: "revision",
      });
      nextRevision = existingSheets.length > 0 ? existingSheets[0].revision + 1 : 0;
    }

    return {
      sourceSheet,
      sourceEntries,
      isRevising,
      nextRevision,
    };
  } catch (err) {
    console.error(`Failed to load source rate sheet: ${err}`);
    throw error(404, `Failed to load source rate sheet: ${err}`);
  }
};
