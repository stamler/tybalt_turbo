import type { CurrenciesResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const currencies = createCollectionStore<CurrenciesResponse>(
  "currencies",
  {
    sort: "ui_sort,code",
    requestKey: "currencies",
  },
  {
    fields: ["code", "symbol", "rate_date"],
    storeFields: ["id", "code", "symbol", "icon", "rate", "rate_date", "ui_sort"],
  },
  async (item) => {
    const fullRecord = await pb.collection("currencies").getOne<CurrenciesResponse>(item.id);
    currencies.update((state) => ({
      ...state,
      items: [...state.items, fullRecord].sort(
        (a, b) => (a.ui_sort ?? 0) - (b.ui_sort ?? 0) || a.code.localeCompare(b.code),
      ),
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    const fullRecord = await pb.collection("currencies").getOne<CurrenciesResponse>(item.id);
    currencies.update((state) => ({
      ...state,
      items: state.items
        .map((i) => (i.id === item.id ? fullRecord : i))
        .sort((a, b) => (a.ui_sort ?? 0) - (b.ui_sort ?? 0) || a.code.localeCompare(b.code)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
