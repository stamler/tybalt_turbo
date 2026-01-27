import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async ({ url }) => {
  const reviseId = url.searchParams.get("revise");

  if (reviseId) {
    try {
      const sourceSheet = await pb.collection("rate_sheets").getOne(reviseId);

      // Find the highest revision number for this rate sheet name
      const existingSheets = await pb.collection("rate_sheets").getFullList({
        filter: `name = "${sourceSheet.name}"`,
        sort: "-revision",
        fields: "revision",
      });

      const maxRevision = existingSheets.length > 0 ? existingSheets[0].revision : 0;

      return {
        revising: true,
        sourceSheet,
        nextRevision: maxRevision + 1,
      };
    } catch (err) {
      console.error(`Failed to load source rate sheet for revision: ${err}`);
      // Fall through to default (new sheet)
    }
  }

  return {
    revising: false,
    sourceSheet: null,
    nextRevision: 0,
  };
};
