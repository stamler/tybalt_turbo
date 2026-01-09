// Custom type that matches the payload returned by /api/jobs
export interface JobApiResponse {
  id: string;
  number: string;
  description: string;
  location: string;
  client_id: string;
  client: string; // client name
  branch: string; // branch code
  outstanding_balance: number;
  outstanding_balance_date: string;
}

import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

// Helper that fetches all jobs via the custom endpoint
const fetchAllJobs = async (): Promise<JobApiResponse[]> => {
  return pb.send("/api/jobs", { method: "GET" });
};

// Note: we deliberately pass <any> here because the JobApiResponse we get from the
// custom endpoint does not include PocketBase system fields (collectionId, etc.),
// and we want to avoid spreading the PB-specific types into the rest of the app.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const jobs = createCollectionStore<any>(
  "jobs",
  {},
  {
    fields: ["number", "description", "location", "client"],
    storeFields: [
      "id",
      "number",
      "description",
      "location",
      "client",
      "branch",
      "outstanding_balance",
      "outstanding_balance_date",
    ],
    extractField: (document, fieldName) =>
      (document as Record<string, unknown>)[fieldName] as string,
  },
  // onCreate – fetch the full job via the API and add it
  async (item) => {
    const fullRecord: JobApiResponse = await pb.send(`/api/jobs/${item.id}`, { method: "GET" });
    jobs.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  // onUpdate – re-fetch, then replace existing entry in the store
  async (item) => {
    const fullRecord: JobApiResponse = await pb.send(`/api/jobs/${item.id}`, { method: "GET" });
    jobs.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
  // proxyCollectionName – listen to realtime events on the underlying jobs collection
  "jobs",
  // Custom fetchAll implementation (avoids PocketBase N+1 queries)
  fetchAllJobs,
);
