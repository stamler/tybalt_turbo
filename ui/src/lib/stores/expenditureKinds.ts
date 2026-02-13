// Shared expenditure kinds collection store.
//
// Why this exists:
// Editors need the same kind list for labels/defaulting/toggles. Loading once via
// a shared store avoids repeated fetch logic and keeps UI behavior consistent.
import type { ExpenditureKindsResponse } from "$lib/pocketbase-types";
import { createCollectionStore } from "./collectionStore";
import { pb } from "$lib/pocketbase";

export const expenditureKinds = createCollectionStore<ExpenditureKindsResponse>(
  "expenditure_kinds",
  {
    sort: "ui_order",
    requestKey: "expenditure_kinds",
  },
  {
    fields: ["name", "en_ui_label", "description"],
    storeFields: ["id", "name", "en_ui_label", "description"],
  },
  async (item) => {
    const fullRecord = await pb
      .collection("expenditure_kinds")
      .getOne<ExpenditureKindsResponse>(item.id);
    expenditureKinds.update((state) => ({
      ...state,
      items: [...state.items, fullRecord],
      index: state.index?.add(fullRecord) || state.index,
    }));
  },
  async (item) => {
    const fullRecord = await pb
      .collection("expenditure_kinds")
      .getOne<ExpenditureKindsResponse>(item.id);
    expenditureKinds.update((state) => ({
      ...state,
      items: state.items.map((i) => (i.id === item.id ? fullRecord : i)),
      index: state.index?.replace(fullRecord) || state.index,
    }));
  },
);
