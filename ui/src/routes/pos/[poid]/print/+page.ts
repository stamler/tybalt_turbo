import type { PageLoad } from "./$types";
import { error, isHttpError } from "@sveltejs/kit";
import { fetchVisiblePO } from "$lib/poVisibility";
import { readPOPrintRequest } from "$lib/poPrint";

export const load: PageLoad = async ({ params, url }) => {
  try {
    const po = await fetchVisiblePO(params.poid);

    if (po.status !== "Active") {
      throw error(404, "Printable purchase order not found");
    }

    try {
      return {
        po,
        printOptions: readPOPrintRequest(po, params.poid, url.searchParams, sessionStorage),
      };
    } catch (err) {
      throw error(400, err instanceof Error ? err.message : "Invalid print options.");
    }
  } catch (err: unknown) {
    if (isHttpError(err)) throw err;

    console.error(`loading printable purchase order: ${err}`);
    throw error(500, "Failed to load printable purchase order");
  }
};
