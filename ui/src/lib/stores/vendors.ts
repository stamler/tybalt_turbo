import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export interface VendorApiResponse {
  id: string;
  name: string;
  alias: string;
  expenses_count: number;
  purchase_orders_count: number;
}

const fetchAllVendors = async (): Promise<VendorApiResponse[]> =>
  pb.send("/api/vendors", { method: "GET" });

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const vendors = createCollectionStore<any>(
  "vendors",
  {
    filter: "status = 'Active'",
    requestKey: "vendors",
  },
  {
    fields: ["name", "alias"],
    storeFields: ["id", "name", "alias", "expenses_count", "purchase_orders_count"],
  },

  async (item) => {
    const record: VendorApiResponse = await pb.send(`/api/vendors/${item.id}`, { method: "GET" });
    vendors.update((s) => ({
      ...s,
      items: [...s.items, record],
      index: s.index?.add(record) || s.index,
    }));
  },

  async (item) => {
    const fullRecord: VendorApiResponse = await pb.send(`/api/vendors/${item.id}`, {
      method: "GET",
    });
    vendors.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
  "vendors",
  fetchAllVendors,
  true,
);
