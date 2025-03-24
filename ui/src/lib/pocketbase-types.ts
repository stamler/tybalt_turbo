/**
 * This file was @generated using pocketbase-typegen
 */

import type PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";

export enum Collections {
  Authorigins = "_authOrigins",
  Externalauths = "_externalAuths",
  Mfas = "_mfas",
  Otps = "_otps",
  Superusers = "_superusers",
  AbsorbActions = "absorb_actions",
  AdminProfiles = "admin_profiles",
  Categories = "categories",
  Claims = "claims",
  ClientContacts = "client_contacts",
  Clients = "clients",
  Divisions = "divisions",
  ExpenseRates = "expense_rates",
  Expenses = "expenses",
  Jobs = "jobs",
  Managers = "managers",
  PayrollYearEndDates = "payroll_year_end_dates",
  PoApprovalThresholds = "po_approval_thresholds",
  PoApproverProps = "po_approver_props",
  Profiles = "profiles",
  PurchaseOrderThresholds = "purchase_order_thresholds",
  PurchaseOrders = "purchase_orders",
  TimeAmendments = "time_amendments",
  TimeEntries = "time_entries",
  TimeOff = "time_off",
  TimeSheetReviewers = "time_sheet_reviewers",
  TimeSheets = "time_sheets",
  TimeTypes = "time_types",
  UserClaims = "user_claims",
  Users = "users",
  Vendors = "vendors",
}

// Alias types for improved usability
export type IsoDateString = string;
export type RecordIdString = string;
export type HTMLString = string;

// System fields
export type BaseSystemFields<T = never> = {
  id: RecordIdString;
  collectionId: string;
  collectionName: Collections;
  expand: T;
};

export type AuthSystemFields<T = never> = {
  email: string;
  emailVisibility: boolean;
  username: string;
  verified: boolean;
} & BaseSystemFields<T>;

// Record types for each collection

export type AuthoriginsRecord = {
  collectionRef: string;
  created: IsoDateString;
  fingerprint: string;
  id: string;
  recordRef: string;
  updated: IsoDateString;
};

export type ExternalauthsRecord = {
  collectionRef: string;
  created: IsoDateString;
  id: string;
  provider: string;
  providerId: string;
  recordRef: string;
  updated: IsoDateString;
};

export type MfasRecord = {
  collectionRef: string;
  created: IsoDateString;
  id: string;
  method: string;
  recordRef: string;
  updated: IsoDateString;
};

export type OtpsRecord = {
  collectionRef: string;
  created: IsoDateString;
  id: string;
  password: string;
  recordRef: string;
  sentTo: string;
  updated: IsoDateString;
};

export type SuperusersRecord = {
  created: IsoDateString;
  email: string;
  emailVisibility: boolean;
  id: string;
  password: string;
  tokenKey: string;
  updated: IsoDateString;
  verified: boolean;
};

export type AbsorbActionsRecord<Tabsorbed_records = unknown, Tupdated_references = unknown> = {
  absorbed_records: null | Tabsorbed_records;
  collection_name: string;
  created: IsoDateString;
  id: string;
  target_id: string;
  updated: IsoDateString;
  updated_references: null | Tupdated_references;
};

export enum AdminProfilesSkipMinTimeCheckOptions {
  "no" = "no",
  "on_next_bundle" = "on_next_bundle",
  "yes" = "yes",
}
export type AdminProfilesRecord = {
  created: IsoDateString;
  default_charge_out_rate: number;
  id: string;
  off_rotation_permitted: boolean;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  payroll_id: string;
  salary: boolean;
  skip_min_time_check: AdminProfilesSkipMinTimeCheckOptions;
  uid: RecordIdString;
  updated: IsoDateString;
  work_week_hours: number;
};

export type CategoriesRecord = {
  created: IsoDateString;
  id: string;
  job: RecordIdString;
  name: string;
  updated: IsoDateString;
};

export type ClaimsRecord = {
  created: IsoDateString;
  description: string;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type ClientContactsRecord = {
  client: RecordIdString;
  created: IsoDateString;
  email: string;
  given_name: string;
  id: string;
  surname: string;
  updated: IsoDateString;
};

export type ClientsRecord = {
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type DivisionsRecord = {
  code: string;
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type ExpenseRatesRecord<Tmileage = unknown> = {
  breakfast: number;
  created: IsoDateString;
  dinner: number;
  effective_date: string;
  id: string;
  lodging: number;
  lunch: number;
  mileage: null | Tmileage;
  updated: IsoDateString;
};

export enum ExpensesPaymentTypeOptions {
  "OnAccount" = "OnAccount",
  "Expense" = "Expense",
  "CorporateCreditCard" = "CorporateCreditCard",
  "Allowance" = "Allowance",
  "FuelCard" = "FuelCard",
  "Mileage" = "Mileage",
  "PersonalReimbursement" = "PersonalReimbursement",
}

export enum ExpensesAllowanceTypesOptions {
  "Lodging" = "Lodging",
  "Breakfast" = "Breakfast",
  "Lunch" = "Lunch",
  "Dinner" = "Dinner",
}
export type ExpensesRecord = {
  allowance_types: ExpensesAllowanceTypesOptions[];
  approved: IsoDateString;
  approver: RecordIdString;
  attachment: string;
  category: RecordIdString;
  cc_last_4_digits: string;
  committed: IsoDateString;
  committed_week_ending: string;
  committer: RecordIdString;
  created: IsoDateString;
  date: string;
  description: string;
  distance: number;
  division: RecordIdString;
  id: string;
  job: RecordIdString;
  pay_period_ending: string;
  payment_type: ExpensesPaymentTypeOptions;
  purchase_order: RecordIdString;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  submitted: boolean;
  total: number;
  uid: RecordIdString;
  updated: IsoDateString;
  vendor: RecordIdString;
};

export type JobsRecord = {
  client: RecordIdString;
  contact: RecordIdString;
  created: IsoDateString;
  description: string;
  id: string;
  manager: RecordIdString;
  number: string;
  updated: IsoDateString;
};

export type ManagersRecord = {
  given_name: string;
  id: string;
  surname: string;
};

export type PayrollYearEndDatesRecord = {
  created: IsoDateString;
  date: string;
  id: string;
  updated: IsoDateString;
};

export type PoApprovalThresholdsRecord = {
  created: IsoDateString;
  description: string;
  id: string;
  threshold: number;
  updated: IsoDateString;
};

export type PoApproverPropsRecord = {
  created: IsoDateString;
  divisions: RecordIdString[];
  id: string;
  max_amount: number;
  updated: IsoDateString;
  user_claim: RecordIdString;
};

export type ProfilesRecord = {
  alternate_manager: RecordIdString;
  created: IsoDateString;
  default_division: RecordIdString;
  given_name: string;
  id: string;
  manager: RecordIdString;
  surname: string;
  uid: RecordIdString;
  updated: IsoDateString;
};

export type PurchaseOrderThresholdsRecord = {
  approval_total: number;
  id: string;
  lower_threshold: number;
  upper_threshold: number;
};

export enum PurchaseOrdersStatusOptions {
  "Unapproved" = "Unapproved",
  "Active" = "Active",
  "Cancelled" = "Cancelled",
  "Closed" = "Closed",
}

export enum PurchaseOrdersTypeOptions {
  "Normal" = "Normal",
  "Cumulative" = "Cumulative",
  "Recurring" = "Recurring",
}

export enum PurchaseOrdersFrequencyOptions {
  "Weekly" = "Weekly",
  "Biweekly" = "Biweekly",
  "Monthly" = "Monthly",
}

export enum PurchaseOrdersPaymentTypeOptions {
  "OnAccount" = "OnAccount",
  "Expense" = "Expense",
  "CorporateCreditCard" = "CorporateCreditCard",
}
export type PurchaseOrdersRecord = {
  approval_total: number;
  approved: IsoDateString;
  approver: RecordIdString;
  attachment: string;
  cancelled: IsoDateString;
  canceller: RecordIdString;
  category: RecordIdString;
  closed: IsoDateString;
  closed_by_system: boolean;
  closer: RecordIdString;
  created: IsoDateString;
  date: string;
  description: string;
  division: RecordIdString;
  end_date: string;
  frequency: PurchaseOrdersFrequencyOptions;
  id: string;
  job: RecordIdString;
  parent_po: RecordIdString;
  payment_type: PurchaseOrdersPaymentTypeOptions;
  po_number: string;
  priority_second_approver: RecordIdString;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  second_approval: IsoDateString;
  second_approver: RecordIdString;
  second_approver_claim: RecordIdString;
  status: PurchaseOrdersStatusOptions;
  total: number;
  type: PurchaseOrdersTypeOptions;
  uid: RecordIdString;
  updated: IsoDateString;
  vendor: RecordIdString;
};

export type TimeAmendmentsRecord = {
  category: RecordIdString;
  committed: IsoDateString;
  committed_week_ending: string;
  committer: RecordIdString;
  created: IsoDateString;
  creator: RecordIdString;
  date: string;
  description: string;
  division: RecordIdString;
  hours: number;
  id: string;
  job: RecordIdString;
  meals_hours: number;
  payout_request_amount: number;
  skip_tsid_check: boolean;
  time_type: RecordIdString;
  tsid: RecordIdString;
  uid: RecordIdString;
  updated: IsoDateString;
  week_ending: string;
  work_record: string;
};

export type TimeEntriesRecord = {
  category: RecordIdString;
  created: IsoDateString;
  date: string;
  description: string;
  division: RecordIdString;
  hours: number;
  id: string;
  job: RecordIdString;
  meals_hours: number;
  payout_request_amount: number;
  time_type: RecordIdString;
  tsid: RecordIdString;
  uid: RecordIdString;
  updated: IsoDateString;
  week_ending: string;
  work_record: string;
};

export type TimeOffRecord = {
  id: string;
  last_op: string;
  last_ov: string;
  manager: string;
  manager_uid: RecordIdString;
  name: string;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  timesheet_op: number;
  timesheet_ov: number;
  used_op: number;
  used_ov: number;
};

export type TimeSheetReviewersRecord = {
  created: IsoDateString;
  id: string;
  reviewed: IsoDateString;
  reviewer: RecordIdString;
  time_sheet: RecordIdString;
  updated: IsoDateString;
};

export type TimeSheetsRecord = {
  approved: IsoDateString;
  approver: RecordIdString;
  committed: IsoDateString;
  committer: RecordIdString;
  created: IsoDateString;
  id: string;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  salary: boolean;
  submitted: boolean;
  uid: RecordIdString;
  updated: IsoDateString;
  week_ending: string;
  work_week_hours: number;
};

export type TimeTypesRecord = {
  allowed_fields: null | string[];
  code: string;
  created: IsoDateString;
  description: string;
  id: string;
  name: string;
  required_fields: null | string[];
  updated: IsoDateString;
};

export type UserClaimsRecord = {
  cid: RecordIdString;
  created: IsoDateString;
  id: string;
  uid: RecordIdString;
  updated: IsoDateString;
};

export type UsersRecord = {
  created: IsoDateString;
  email: string;
  emailVisibility: boolean;
  id: string;
  name: string;
  password: string;
  tokenKey: string;
  updated: IsoDateString;
  username: string;
  verified: boolean;
};

export enum VendorsStatusOptions {
  "Active" = "Active",
  "Inactive" = "Inactive",
}
export type VendorsRecord = {
  alias: string;
  created: IsoDateString;
  id: string;
  name: string;
  status: VendorsStatusOptions;
  updated: IsoDateString;
};

type TimeEntriesRecordExpands = {
  category: CategoriesRecord;
  division: DivisionsRecord;
  job: JobsRecord;
  time_type: TimeTypesRecord;
};

type TimeAmendmentsRecordExpands = {
  category: CategoriesResponse;
  committer: UsersResponse;
  creator: UsersResponse;
  uid: UsersResponse;
  division: DivisionsRecord;
  job: JobsRecord;
  time_type: TimeTypesRecord;
};

type TimeSheetReviewersRecordExpands = {
  reviewer: ManagersRecord;
  time_sheet: TimeSheetsRecord;
};

type ExpensesRecordExpands = {
  approver: UsersResponse;
  category: CategoriesRecord;
  division: DivisionsRecord;
  job: JobsRecord;
  purchase_order: PurchaseOrdersRecord;
  rejector: UsersResponse;
  uid: UsersResponse;
  vendor: VendorsRecord;
};

type PurchaseOrdersRecordExpands = {
  approver: UsersResponse;
  category: CategoriesResponse;
  division: DivisionsResponse;
  job: JobsResponse;
  parent_po: PurchaseOrdersResponse;
  rejector: UsersResponse;
  second_approver: UsersResponse;
  second_approver_claim: ClaimsResponse;
  type: PurchaseOrdersTypeOptions;
  uid: UsersResponse;
  vendor: VendorsResponse;
  priority_second_approver: UsersResponse;
};

type UsersRecordExpands = {
  profiles_via_uid: ProfilesResponse;
};

type JobsRecordExpands = {
  categories_via_job: CategoriesResponse[];
  client: ClientsResponse;
};

type ClientsRecordExpands = {
  client_contacts_via_client: ClientContactsResponse[];
};

// Response types include system fields and match responses from the PocketBase API
export type AuthoriginsResponse<Texpand = unknown> = Required<AuthoriginsRecord> &
  BaseSystemFields<Texpand>;
export type ExternalauthsResponse<Texpand = unknown> = Required<ExternalauthsRecord> &
  BaseSystemFields<Texpand>;
export type MfasResponse<Texpand = unknown> = Required<MfasRecord> & BaseSystemFields<Texpand>;
export type OtpsResponse<Texpand = unknown> = Required<OtpsRecord> & BaseSystemFields<Texpand>;
export type SuperusersResponse<Texpand = unknown> = Required<SuperusersRecord> &
  AuthSystemFields<Texpand>;
export type AbsorbActionsResponse<
  Tabsorbed_records = unknown,
  Tupdated_references = unknown,
  Texpand = unknown,
> = Required<AbsorbActionsRecord<Tabsorbed_records, Tupdated_references>> &
  BaseSystemFields<Texpand>;
export type AdminProfilesResponse<Texpand = unknown> = Required<AdminProfilesRecord> &
  BaseSystemFields<Texpand>;
export type CategoriesResponse<Texpand = unknown> = Required<CategoriesRecord> &
  BaseSystemFields<Texpand>;
export type ClaimsResponse<Texpand = unknown> = Required<ClaimsRecord> & BaseSystemFields<Texpand>;
export type ClientContactsResponse<Texpand = unknown> = Required<ClientContactsRecord> &
  BaseSystemFields<Texpand>;
export type ClientsResponse<Texpand = ClientsRecordExpands> = Required<ClientsRecord> &
  BaseSystemFields<Texpand>;
export type DivisionsResponse<Texpand = unknown> = Required<DivisionsRecord> &
  BaseSystemFields<Texpand>;
export type ExpenseRatesResponse<Tmileage = unknown, Texpand = unknown> = Required<
  ExpenseRatesRecord<Tmileage>
> &
  BaseSystemFields<Texpand>;
export type ExpensesResponse<Texpand = ExpensesRecordExpands> = Required<ExpensesRecord> &
  BaseSystemFields<Texpand>;
export type JobsResponse<Texpand = JobsRecordExpands> = Required<JobsRecord> &
  BaseSystemFields<Texpand>;
export type ManagersResponse<Texpand = unknown> = Required<ManagersRecord> &
  BaseSystemFields<Texpand>;
export type PayrollYearEndDatesResponse<Texpand = unknown> = Required<PayrollYearEndDatesRecord> &
  BaseSystemFields<Texpand>;
export type PoApprovalThresholdsResponse<Texpand = unknown> = Required<PoApprovalThresholdsRecord> &
  BaseSystemFields<Texpand>;
export type PoApproverPropsResponse<Texpand = unknown> = Required<PoApproverPropsRecord> &
  BaseSystemFields<Texpand>;
export type ProfilesResponse<Texpand = unknown> = Required<ProfilesRecord> &
  BaseSystemFields<Texpand>;
export type PurchaseOrderThresholdsResponse<Texpand = unknown> =
  Required<PurchaseOrderThresholdsRecord> & BaseSystemFields<Texpand>;
export type PurchaseOrdersResponse<Texpand = PurchaseOrdersRecordExpands> =
  Required<PurchaseOrdersRecord> & BaseSystemFields<Texpand>;
export type TimeAmendmentsResponse<Texpand = TimeAmendmentsRecordExpands> =
  Required<TimeAmendmentsRecord> & BaseSystemFields<Texpand>;
export type TimeEntriesResponse<Texpand = TimeEntriesRecordExpands> = Required<TimeEntriesRecord> &
  BaseSystemFields<Texpand>;
export type TimeOffResponse<Texpand = unknown> = Required<TimeOffRecord> &
  BaseSystemFields<Texpand>;
export type TimeSheetReviewersResponse<Texpand = TimeSheetReviewersRecordExpands> =
  Required<TimeSheetReviewersRecord> & BaseSystemFields<Texpand>;
export type TimeSheetsResponse<Texpand = unknown> = Required<TimeSheetsRecord> &
  BaseSystemFields<Texpand>;
export type TimeTypesResponse<Texpand = unknown> = Required<TimeTypesRecord> &
  BaseSystemFields<Texpand>;
export type UserClaimsResponse<Texpand = unknown> = Required<UserClaimsRecord> &
  BaseSystemFields<Texpand>;
export type UsersResponse<Texpand = UsersRecordExpands> = Required<UsersRecord> &
  AuthSystemFields<Texpand>;
export type VendorsResponse<Texpand = unknown> = Required<VendorsRecord> &
  BaseSystemFields<Texpand>;

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
  _authOrigins: AuthoriginsRecord;
  _externalAuths: ExternalauthsRecord;
  _mfas: MfasRecord;
  _otps: OtpsRecord;
  _superusers: SuperusersRecord;
  absorb_actions: AbsorbActionsRecord;
  admin_profiles: AdminProfilesRecord;
  categories: CategoriesRecord;
  claims: ClaimsRecord;
  client_contacts: ClientContactsRecord;
  clients: ClientsRecord;
  divisions: DivisionsRecord;
  expense_rates: ExpenseRatesRecord;
  expenses: ExpensesRecord;
  jobs: JobsRecord;
  managers: ManagersRecord;
  payroll_year_end_dates: PayrollYearEndDatesRecord;
  po_approval_thresholds: PoApprovalThresholdsRecord;
  po_approver_props: PoApproverPropsRecord;
  profiles: ProfilesRecord;
  purchase_order_thresholds: PurchaseOrderThresholdsRecord;
  purchase_orders: PurchaseOrdersRecord;
  time_amendments: TimeAmendmentsRecord;
  time_entries: TimeEntriesRecord;
  time_off: TimeOffRecord;
  time_sheet_reviewers: TimeSheetReviewersRecord;
  time_sheets: TimeSheetsRecord;
  time_types: TimeTypesRecord;
  user_claims: UserClaimsRecord;
  users: UsersRecord;
  vendors: VendorsRecord;
};

export type CollectionResponses = {
  _authOrigins: AuthoriginsResponse;
  _externalAuths: ExternalauthsResponse;
  _mfas: MfasResponse;
  _otps: OtpsResponse;
  _superusers: SuperusersResponse;
  absorb_actions: AbsorbActionsResponse;
  admin_profiles: AdminProfilesResponse;
  categories: CategoriesResponse;
  claims: ClaimsResponse;
  client_contacts: ClientContactsResponse;
  clients: ClientsResponse;
  divisions: DivisionsResponse;
  expense_rates: ExpenseRatesResponse;
  expenses: ExpensesResponse;
  jobs: JobsResponse;
  managers: ManagersResponse;
  payroll_year_end_dates: PayrollYearEndDatesResponse;
  po_approval_thresholds: PoApprovalThresholdsResponse;
  po_approver_props: PoApproverPropsResponse;
  profiles: ProfilesResponse;
  purchase_order_thresholds: PurchaseOrderThresholdsResponse;
  purchase_orders: PurchaseOrdersResponse;
  time_amendments: TimeAmendmentsResponse;
  time_entries: TimeEntriesResponse;
  time_off: TimeOffResponse;
  time_sheet_reviewers: TimeSheetReviewersResponse;
  time_sheets: TimeSheetsResponse;
  time_types: TimeTypesResponse;
  user_claims: UserClaimsResponse;
  users: UsersResponse;
  vendors: VendorsResponse;
};

// Type for usage with type asserted PocketBase instance
// https://github.com/pocketbase/js-sdk#specify-typescript-definitions

export type TypedPocketBase = PocketBase & {
  collection(idOrName: "_authOrigins"): RecordService<AuthoriginsResponse>;
  collection(idOrName: "_externalAuths"): RecordService<ExternalauthsResponse>;
  collection(idOrName: "_mfas"): RecordService<MfasResponse>;
  collection(idOrName: "_otps"): RecordService<OtpsResponse>;
  collection(idOrName: "_superusers"): RecordService<SuperusersResponse>;
  collection(idOrName: "absorb_actions"): RecordService<AbsorbActionsResponse>;
  collection(idOrName: "admin_profiles"): RecordService<AdminProfilesResponse>;
  collection(idOrName: "categories"): RecordService<CategoriesResponse>;
  collection(idOrName: "claims"): RecordService<ClaimsResponse>;
  collection(idOrName: "client_contacts"): RecordService<ClientContactsResponse>;
  collection(idOrName: "clients"): RecordService<ClientsResponse>;
  collection(idOrName: "divisions"): RecordService<DivisionsResponse>;
  collection(idOrName: "expense_rates"): RecordService<ExpenseRatesResponse>;
  collection(idOrName: "expenses"): RecordService<ExpensesResponse>;
  collection(idOrName: "jobs"): RecordService<JobsResponse>;
  collection(idOrName: "managers"): RecordService<ManagersResponse>;
  collection(idOrName: "payroll_year_end_dates"): RecordService<PayrollYearEndDatesResponse>;
  collection(idOrName: "po_approval_thresholds"): RecordService<PoApprovalThresholdsResponse>;
  collection(idOrName: "po_approver_props"): RecordService<PoApproverPropsResponse>;
  collection(idOrName: "profiles"): RecordService<ProfilesResponse>;
  collection(idOrName: "purchase_order_thresholds"): RecordService<PurchaseOrderThresholdsResponse>;
  collection(idOrName: "purchase_orders"): RecordService<PurchaseOrdersResponse>;
  collection(idOrName: "time_amendments"): RecordService<TimeAmendmentsResponse>;
  collection(idOrName: "time_entries"): RecordService<TimeEntriesResponse>;
  collection(idOrName: "time_off"): RecordService<TimeOffResponse>;
  collection(idOrName: "time_sheet_reviewers"): RecordService<TimeSheetReviewersResponse>;
  collection(idOrName: "time_sheets"): RecordService<TimeSheetsResponse>;
  collection(idOrName: "time_types"): RecordService<TimeTypesResponse>;
  collection(idOrName: "user_claims"): RecordService<UserClaimsResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
  collection(idOrName: "vendors"): RecordService<VendorsResponse>;
};

export type SelectOption = { id: string | number };

// Type guards
export function isBaseSystemFields(item: unknown): item is BaseSystemFields {
  return (
    !!item &&
    typeof item === "object" &&
    "expand" in item &&
    !!item.expand &&
    "id" in item &&
    typeof item.id === "string"
  );
}

export function isExpensesRecord(item: unknown): item is ExpensesRecord {
  return !!item && typeof item === "object" && "uid" in item && typeof item.uid === "string";
}

export function isExpensesResponse(item: unknown): item is ExpensesResponse {
  return isBaseSystemFields(item) && isExpensesRecord(item);
}
