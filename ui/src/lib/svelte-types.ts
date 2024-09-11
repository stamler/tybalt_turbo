import type { TimeEntriesRecord, PurchaseOrdersRecord } from "$lib/pocketbase-types";

export interface TimeEntriesPageData {
  item: TimeEntriesRecord;
  editing: boolean;
  id: string | null;
}

export interface PurchaseOrdersPageData {
  item: PurchaseOrdersRecord;
  editing: boolean;
  id: string | null;
}
