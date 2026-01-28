import type {
  TimeEntriesRecord,
  PurchaseOrdersRecord,
  PurchaseOrdersResponse,
  ExpensesRecord,
  ExpensesResponse,
  JobsRecord,
  CategoriesResponse,
  ClientsRecord,
  ClientContactsResponse,
  VendorsRecord,
  VendorsResponse,
  TimeAmendmentsRecord,
  AdminProfilesRecord,
  AdminProfilesAugmentedResponse,
  ClientDetails,
  ClientNotesResponse,
  ExpensesAugmentedResponse,
} from "$lib/pocketbase-types";

export interface PageData<T> {
  item: T;
  editing: boolean;
  id: string | null;
}

export type TimeEntriesPageData = PageData<TimeEntriesRecord>;
export type TimeAmendmentsPageData = PageData<TimeAmendmentsRecord>;
export type PurchaseOrdersPageData = PageData<PurchaseOrdersRecord | PurchaseOrdersResponse> & {
  parent_po_number?: string;
};
export type ExpensesPageData = PageData<ExpensesRecord | ExpensesResponse>;
export type JobsPageData = PageData<JobsRecord> & {
  categories: CategoriesResponse[];
  // When creating a project from a proposal, this contains any existing projects that already reference the proposal
  existingReferencingProjects?: { id: string; number: string }[];
};
export type ClientsPageData = PageData<ClientsRecord> & {
  client_contacts: ClientContactsResponse[];
};
export type ClientDetailsPageData = {
  client: ClientDetails;
  jobs: any[];
  notes: ClientNotesResponse[];
  noteJobs: JobsRecord[];
  tab: string;
  page: number;
  totalPages: number;
  projectsPage: number;
  proposalsPage: number;
  ownerPage: number;
  counts: {
    projects: number;
    proposals: number;
    owner: number;
  };
};
export type VendorsPageData = PageData<VendorsRecord | VendorsResponse>;
export type AdminProfilesPageData = PageData<AdminProfilesRecord | AdminProfilesAugmentedResponse>;

// Expenses list pages use API endpoints returning augmented rows plus pagination metadata
export type ExpensesListData = {
  items: ExpensesAugmentedResponse[];
  createdItemIsVisible?: (record: ExpensesResponse) => boolean;
  totalPages?: number;
  limit?: number;
};
