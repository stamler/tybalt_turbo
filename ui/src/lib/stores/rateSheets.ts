import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export interface RateSheetResponse {
  id: string;
  name: string;
  effective_date: string;
  revision: number;
  active: boolean;
  job_count: number;
}

// Fetch from the augmented view which includes job_count
const fetchAllRateSheets = async (): Promise<RateSheetResponse[]> =>
  pb.collection("rate_sheets_augmented").getFullList({ sort: "-effective_date" });

export const rateSheets = createCollectionStore<RateSheetResponse>(
  "rate_sheets", // Subscribe to the base collection for real-time updates
  { requestKey: "rate_sheets" },
  {
    fields: ["name", "effective_date"],
    storeFields: ["id", "name", "effective_date", "revision", "active", "job_count"],
  },
  // onCreate handler - refetch from augmented view
  async (item) => {
    const record = await pb.collection("rate_sheets_augmented").getOne<RateSheetResponse>(item.id);
    rateSheets.update((s) => ({
      ...s,
      items: [...s.items, record],
      index: s.index?.add(record) || s.index,
    }));
  },
  // onUpdate handler - refetch from augmented view
  async (item) => {
    const record = await pb.collection("rate_sheets_augmented").getOne<RateSheetResponse>(item.id);
    rateSheets.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? record : i)),
      index: state.index?.replace(record) || state.index,
    }));
  },
  "rate_sheets",
  fetchAllRateSheets,
  false, // no absorb subscription needed
);
