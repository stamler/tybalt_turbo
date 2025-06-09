import type { VendorsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";

export const vendors = createCollectionStore<VendorsResponse>(
  "vendors",
  {
    filter: "status = 'Active'",
    requestKey: "vendors",
  },
  {
    fields: ["id", "name", "alias"],
    storeFields: ["id", "name", "alias"],
  },
);
