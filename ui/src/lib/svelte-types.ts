import type {
  TimeEntriesRecord,
  PurchaseOrdersRecord,
  PurchaseOrdersResponse,
  PurchaseOrdersPaymentTypeOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
  PoApproversResponse,
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
  PurchaseOrdersAugmentedResponse,
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
export type SecondApproverStatus =
  | "not_required"
  | "requester_qualifies"
  | "candidates_available"
  | "required_no_candidates";
export type SecondApproversResponse = {
  approvers: PoApproversResponse[];
  meta: {
    second_approval_required: boolean;
    requester_qualifies: boolean;
    status: SecondApproverStatus;
    reason_code: string;
    reason_message: string;
    evaluated_amount: number;
    second_approval_threshold: number;
    limit_column: string;
    second_stage_timeout_hours: number;
  };
};
export type PurchaseOrderDetailsPageData = {
  po: PurchaseOrdersAugmentedResponse;
  expenses: ExpensesAugmentedResponse[];
  secondApproverDiagnostics: SecondApproversResponse | null;
  canApproveOrReject: boolean;
};
export type LinkedPurchaseOrderSummary = {
  id: string;
  po_number: string;
  type: PurchaseOrdersTypeOptions;
  payment_type: PurchaseOrdersPaymentTypeOptions;
  status: PurchaseOrdersStatusOptions;
  recurring_expected_occurrences?: number;
  recurring_remaining_occurrences?: number;
  cumulative_remaining_balance?: number;
};
export type ExpensesPageData = PageData<ExpensesRecord | ExpensesResponse> & {
  linked_purchase_order?: LinkedPurchaseOrderSummary;
};
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
export type AdminProfilesEditPageData = PageData<AdminProfilesAugmentedResponse>;

// Expenses list pages use API endpoints returning augmented rows plus pagination metadata
export type ExpensesListData = {
  items: ExpensesAugmentedResponse[];
  createdItemIsVisible?: (record: ExpensesResponse) => boolean;
  totalPages?: number;
  limit?: number;
};

// Claims types returned by custom API endpoints
export interface ClaimListItem {
  id: string;
  name: string;
  description: string;
  holder_count: number;
}

export interface ClaimHolder {
  id: string;
  admin_profile_id: string;
  given_name: string;
  surname: string;
}

export interface ClaimDetails {
  id: string;
  name: string;
  description: string;
  holders: ClaimHolder[];
}
