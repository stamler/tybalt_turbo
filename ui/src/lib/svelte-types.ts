import type {
  TimeEntriesRecord,
  PurchaseOrdersRecord,
  ExpensesRecord,
  ExpensesResponse,
  JobsRecord,
  CategoriesResponse,
  ClientsRecord,
  ClientContactsResponse,
  PoApproversResponse,
  VendorsRecord,
  VendorsResponse,
  TimeAmendmentsRecord,
} from "$lib/pocketbase-types";

export interface PageData<T> {
  item: T;
  editing: boolean;
  id: string | null;
}

export type TimeEntriesPageData = PageData<TimeEntriesRecord>;
export type TimeAmendmentsPageData = PageData<TimeAmendmentsRecord>;
export type PurchaseOrdersPageData = PageData<PurchaseOrdersRecord> & {
  approvers: PoApproversResponse[];
};
export type ExpensesPageData = PageData<ExpensesRecord | ExpensesResponse>;
export type JobsPageData = PageData<JobsRecord> & { categories: CategoriesResponse[] };
export type ClientsPageData = PageData<ClientsRecord> & {
  client_contacts: ClientContactsResponse[];
};
export type VendorsPageData = PageData<VendorsRecord | VendorsResponse>;
