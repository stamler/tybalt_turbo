import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params }) => {
  const rateSheetId = params.id;

  try {
    // Fetch rate sheet record
    const rateSheet = await pb.collection("rate_sheets").getOne(rateSheetId);

    // Fetch all rate_sheet_entries for this sheet (expanded with role relation)
    const entries = await pb.collection("rate_sheet_entries").getFullList({
      filter: `rate_sheet="${rateSheetId}"`,
      expand: "role",
    });

    // Fetch all rate_roles (to compute missing roles)
    const allRoles = await pb.collection("rate_roles").getFullList({ sort: "name" });

    return {
      rateSheet,
      entries,
      allRoles,
    };
  } catch (err) {
    console.error(`loading rate sheet details: ${err}`);
    throw error(404, `Failed to load rate sheet details: ${err}`);
  }
};
