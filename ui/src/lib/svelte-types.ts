import type {
  TimeEntriesRecord,
  PurchaseOrdersRecord,
  ExpensesRecord,
  ExpensesResponse,
  JobsRecord,
  CategoriesResponse,
  ClientsRecord,
  ContactsResponse,
} from "$lib/pocketbase-types";

export interface PageData<T> {
  item: T;
  editing: boolean;
  id: string | null;
}

export type TimeEntriesPageData = PageData<TimeEntriesRecord>;
export type PurchaseOrdersPageData = PageData<PurchaseOrdersRecord>;
export type ExpensesPageData = PageData<ExpensesRecord | ExpensesResponse>;
export type JobsPageData = PageData<JobsRecord> & { categories: CategoriesResponse[] };
export type ClientsPageData = PageData<ClientsRecord> & { contacts: ContactsResponse[] };