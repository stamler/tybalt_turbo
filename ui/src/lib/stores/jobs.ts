import type { JobsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

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
  async (item) => {
    // Fetch the new record with expand options and add to store
    const fullRecord = await pb.collection("jobs").getOne<JobsResponse>(item.id, {
      expand: "categories_via_job,client",
    });
    jobs.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    // Fetch the updated record with expand options and add to store
    const fullRecord = await pb.collection("jobs").getOne<JobsResponse>(item.id, {
      expand: "categories_via_job,client",
    });
    jobs.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
);
