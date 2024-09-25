import type {
  TimeEntriesRecord,
  PurchaseOrdersRecord,
  ExpensesRecord,
  JobsRecord,
  CategoriesResponse,
} from "$lib/pocketbase-types";

export interface PageData<T> {
  item: T;
  editing: boolean;
  id: string | null;
}

export type TimeEntriesPageData = PageData<TimeEntriesRecord>;
export type PurchaseOrdersPageData = PageData<PurchaseOrdersRecord>;
export type ExpensesPageData = PageData<ExpensesRecord>;
export type JobsPageData = PageData<JobsRecord> & { categories: CategoriesResponse[] };
