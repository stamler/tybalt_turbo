import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export interface POSearchApiResponse {
  id: string;
  po_number: string;
  status: "Active" | "Closed" | "Cancelled";
  uid: string;
  uid_name: string;
  legacy_manual_entry: boolean;
  type: "One-Time" | "Cumulative" | "Recurring";
  date: string;
  end_date: string;
  frequency: string;
  description: string;
  total: number;
  approval_total_home: number;
  currency: string;
  currency_code: string;
  currency_symbol: string;
  currency_icon: string;
  currency_rate: number;
  currency_rate_date: string;
  attachment: string;
  committed_expenses_count: number;
  vendor_name: string;
  vendor_alias: string;
  job_number: string;
  client_name: string;
  job_description: string;
  category_name: string;
}

async function fetchAllSearchPOs(): Promise<POSearchApiResponse[]> {
  return pb.send("/api/purchase_orders/search", { method: "GET" });
}

async function fetchSearchEligiblePO(id: string): Promise<POSearchApiResponse | null> {
  try {
    const po = (await pb.send(`/api/purchase_orders/visible/${id}`, {
      method: "GET",
    })) as POSearchApiResponse & { status: string };

    if (po.status !== "Active" && po.status !== "Closed" && po.status !== "Cancelled") {
      return null;
    }

    return po;
  } catch (error: any) {
    if (error?.status === 404 || error?.response?.code === "po_not_found_or_not_visible") {
      return null;
    }
    throw error;
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export const poSearch = createCollectionStore<any>(
  "purchase_orders_search",
  {},
  {
    fields: [
      "po_number",
      "vendor_name",
      "vendor_alias",
      "description",
      "job_number",
      "client_name",
      "job_description",
      "category_name",
      "uid_name",
    ],
    storeFields: [
      "id",
      "po_number",
      "status",
      "uid",
      "uid_name",
      "legacy_manual_entry",
      "type",
      "date",
      "end_date",
      "frequency",
      "description",
      "total",
      "approval_total_home",
      "currency",
      "currency_code",
      "currency_symbol",
      "currency_icon",
      "currency_rate",
      "currency_rate_date",
      "attachment",
      "committed_expenses_count",
      "vendor_name",
      "vendor_alias",
      "job_number",
      "client_name",
      "job_description",
      "category_name",
    ],
    extractField: (document, fieldName) =>
      (document as Record<string, unknown>)[fieldName] as string,
    searchOptions: {
      boost: { po_number: 3, vendor_name: 2, job_number: 2 },
    },
  },
  async (item) => {
    const fullRecord = await fetchSearchEligiblePO(item.id);
    if (!fullRecord) return;

    poSearch.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    const fullRecord = await fetchSearchEligiblePO(item.id);

    poSearch.update((state) => {
      const exists = state.items.some((i) => i.id === item.id);

      if (!fullRecord) {
        return {
          ...state,
          items: state.items.filter((i) => i.id !== item.id),
          index: state.index?.discard(item.id) || state.index,
        };
      }

      if (!exists) {
        return {
          ...state,
          items: [...state.items, fullRecord],
          index: state.index?.add(fullRecord) || state.index,
        };
      }

      return {
        ...state,
        items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
        index: state.index?.replace(fullRecord) || state.index,
      };
    });
  },
  "purchase_orders",
  fetchAllSearchPOs,
);
