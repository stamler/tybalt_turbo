import type { JobsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";

export const jobs = createCollectionStore<JobsResponse>(
  "jobs",
  {
    expand: "categories_via_job,client",
    sort: "-number",
    requestKey: "job",
  },
  {
    fields: ["id", "number", "description", "client"],
    storeFields: ["id", "number", "description", "client"],
    extractField: (document, fieldName) => {
      if (fieldName === "client") {
        return document.expand?.client?.name ?? "";
      }
      return document[fieldName as keyof typeof document] as string;
    },
  },
);
