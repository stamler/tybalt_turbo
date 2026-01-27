import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export interface RateSheetResponse {
  id: string;
  name: string;
  effective_date: string;
  revision: number;
  active: boolean;
}

const fetchAllRateSheets = async (): Promise<RateSheetResponse[]> =>
  pb.collection("rate_sheets").getFullList({ sort: "-effective_date" });

export const rateSheets = createCollectionStore<RateSheetResponse>(
  "rate_sheets",
  { requestKey: "rate_sheets" },
  {
    fields: ["name", "effective_date"],
    storeFields: ["id", "name", "effective_date", "revision", "active"],
  },
  // onCreate handler
  async (item) => {
    const record = await pb.collection("rate_sheets").getOne<RateSheetResponse>(item.id);
    rateSheets.update((s) => ({
      ...s,
      items: [...s.items, record],
      index: s.index?.add(record) || s.index,
    }));
  },
  // onUpdate handler
  async (item) => {
    const record = await pb.collection("rate_sheets").getOne<RateSheetResponse>(item.id);
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
