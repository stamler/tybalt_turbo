/**
 * This file was @generated using pocketbase-typegen
 */

import type PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";

export enum Collections {
  AdminProfiles = "admin_profiles",
  Categories = "categories",
  Claims = "claims",
  Clients = "clients",
  Contacts = "contacts",
  Divisions = "divisions",
  ExpenseRates = "expense_rates",
  Expenses = "expenses",
  Jobs = "jobs",
  Managers = "managers",
  PayrollYearEndDates = "payroll_year_end_dates",
  PoApprovers = "po_approvers",
  Profiles = "profiles",
  PurchaseOrders = "purchase_orders",
  TimeAmendments = "time_amendments",
  TimeEntries = "time_entries",
  TimeOff = "time_off",
  TimeSheetReviewers = "time_sheet_reviewers",
  TimeSheets = "time_sheets",
  TimeTypes = "time_types",
  UserClaims = "user_claims",
  Users = "users",
}

// Alias types for improved usability
export type IsoDateString = string;
export type RecordIdString = string;
export type HTMLString = string;

// System fields
export type BaseSystemFields<T = never> = {
  id: RecordIdString;
  created: IsoDateString;
  updated: IsoDateString;
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

export enum AdminProfilesSkipMinTimeCheckOptions {
  "no" = "no",
  "on_next_bundle" = "on_next_bundle",
  "yes" = "yes",
}
export type AdminProfilesRecord = {
  default_charge_out_rate: number;
  off_rotation_permitted: boolean;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  payroll_id: string;
  salary: boolean;
  skip_min_time_check: AdminProfilesSkipMinTimeCheckOptions;
  uid: RecordIdString;
  work_week_hours: number;
};

export type CategoriesRecord = {
  job: RecordIdString;
  name: string;
};

export type ClaimsRecord = {
  description: string;
  name: string;
};

export type ClientsRecord = {
  name: string;
};

export type ContactsRecord = {
  client: RecordIdString;
  email: string;
  given_name: string;
  surname: string;
};

export type DivisionsRecord = {
  code: string;
  name: string;
};

export type ExpenseRatesRecord<Tmileage = unknown> = {
  breakfast: number;
  dinner: number;
  effective_date: string;
  lodging: number;
  lunch: number;
  mileage: null | Tmileage;
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
  date: string;
  description: string;
  distance: number;
  division: RecordIdString;
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
  vendor_name: string;
};

export type JobsRecord = {
  client: RecordIdString;
  contact: RecordIdString;
  description: string;
  manager: RecordIdString;
  number: string;
};

export type ManagersRecord = {
  given_name: string;
  surname: string;
};

export type PayrollYearEndDatesRecord = {
  date: string;
};

export type PoApproversRecord = {
  divisions: null | string[];
  given_name: string;
  surname: string;
};

export type ProfilesRecord = {
  alternate_manager: RecordIdString;
  default_division: RecordIdString;
  given_name: string;
  manager: RecordIdString;
  surname: string;
  uid: RecordIdString;
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
  approved: IsoDateString;
  approver: RecordIdString;
  attachment: string;
  cancelled: IsoDateString;
  canceller: RecordIdString;
  category: RecordIdString;
  date: string;
  description: string;
  division: RecordIdString;
  end_date: string;
  frequency: PurchaseOrdersFrequencyOptions;
  job: RecordIdString;
  payment_type: PurchaseOrdersPaymentTypeOptions;
  po_number: string;
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
  vendor_name: string;
};

export type TimeAmendmentsRecord = {
  category: RecordIdString;
  committed: IsoDateString;
  committed_week_ending: string;
  committer: RecordIdString;
  creator: RecordIdString;
  date: string;
  description: string;
  division: RecordIdString;
  hours: number;
  job: RecordIdString;
  meals_hours: number;
  payout_request_amount: number;
  salary: boolean;
  time_type: RecordIdString;
  tsid: RecordIdString;
  uid: RecordIdString;
  week_ending: string;
  work_record: string;
  work_week_hours: number;
};

export type TimeEntriesRecord = {
  category: RecordIdString;
  date: string;
  description: string;
  division: RecordIdString;
  hours: number;
  job: RecordIdString;
  meals_hours: number;
  payout_request_amount: number;
  time_type: RecordIdString;
  tsid: RecordIdString;
  uid: RecordIdString;
  week_ending: string;
  work_record: string;
};

export type TimeOffRecord = {
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
  reviewed: IsoDateString;
  reviewer: RecordIdString;
  time_sheet: RecordIdString;
};

export type TimeSheetsRecord = {
  approved: IsoDateString;
  approver: RecordIdString;
  committed: IsoDateString;
  committer: RecordIdString;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  salary: boolean;
  submitted: boolean;
  uid: RecordIdString;
  week_ending: string;
  work_week_hours: number;
};

export type TimeTypesRecord = {
  allowed_fields: null | string[];
  code: string;
  description: string;
  name: string;
  required_fields: null | string[];
};

export type UserClaimsRecord = {
  cid: RecordIdString;
  payload: null | string[];
  uid: RecordIdString;
};

export type UsersRecord = {
  name: string;
};

type TimeEntriesRecordExpands = {
  category: CategoriesRecord;
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
};

type PurchaseOrdersRecordExpands = {
  approver: UsersResponse;
  category: CategoriesRecord;
  division: DivisionsRecord;
  job: JobsResponse;
  rejector: UsersResponse;
  second_approver: UsersResponse;
  second_approver_claim: ClaimsResponse;
  type: PurchaseOrdersTypeOptions;
  uid: UsersResponse;
};

type UsersRecordExpands = {
  profiles_via_uid: ProfilesResponse;
};

type JobsRecordExpands = {
  categories_via_job: CategoriesResponse[];
  client: ClientsResponse;
};

type ClientsRecordExpands = {
  contacts_via_client: ContactsResponse[];
};

// Response types include system fields and match responses from the PocketBase API
export type AdminProfilesResponse<Texpand = unknown> = Required<AdminProfilesRecord> &
  BaseSystemFields<Texpand>;
export type CategoriesResponse<Texpand = unknown> = Required<CategoriesRecord> &
  BaseSystemFields<Texpand>;
export type ClaimsResponse<Texpand = unknown> = Required<ClaimsRecord> & BaseSystemFields<Texpand>;
export type ClientsResponse<Texpand = ClientsRecordExpands> = Required<ClientsRecord> &
  BaseSystemFields<Texpand>;
export type ContactsResponse<Texpand = unknown> = Required<ContactsRecord> &
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
export type PoApproversResponse<Texpand = unknown> = Required<PoApproversRecord> &
  BaseSystemFields<Texpand>;
export type ProfilesResponse<Texpand = unknown> = Required<ProfilesRecord> &
  BaseSystemFields<Texpand>;
export type PurchaseOrdersResponse<Texpand = PurchaseOrdersRecordExpands> =
  Required<PurchaseOrdersRecord> & BaseSystemFields<Texpand>;
export type TimeAmendmentsResponse<Texpand = unknown> = Required<TimeAmendmentsRecord> &
  BaseSystemFields<Texpand>;
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

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
  admin_profiles: AdminProfilesRecord;
  categories: CategoriesRecord;
  claims: ClaimsRecord;
  clients: ClientsRecord;
  contacts: ContactsRecord;
  divisions: DivisionsRecord;
  expense_rates: ExpenseRatesRecord;
  expenses: ExpensesRecord;
  jobs: JobsRecord;
  managers: ManagersRecord;
  payroll_year_end_dates: PayrollYearEndDatesRecord;
  po_approvers: PoApproversRecord;
  profiles: ProfilesRecord;
  purchase_orders: PurchaseOrdersRecord;
  time_amendments: TimeAmendmentsRecord;
  time_entries: TimeEntriesRecord;
  time_off: TimeOffRecord;
  time_sheet_reviewers: TimeSheetReviewersRecord;
  time_sheets: TimeSheetsRecord;
  time_types: TimeTypesRecord;
  user_claims: UserClaimsRecord;
  users: UsersRecord;
};

export type CollectionResponses = {
  admin_profiles: AdminProfilesResponse;
  categories: CategoriesResponse;
  claims: ClaimsResponse;
  clients: ClientsResponse;
  contacts: ContactsResponse;
  divisions: DivisionsResponse;
  expense_rates: ExpenseRatesResponse;
  expenses: ExpensesResponse;
  jobs: JobsResponse;
  managers: ManagersResponse;
  payroll_year_end_dates: PayrollYearEndDatesResponse;
  po_approvers: PoApproversResponse;
  profiles: ProfilesResponse;
  purchase_orders: PurchaseOrdersResponse;
  time_amendments: TimeAmendmentsResponse;
  time_entries: TimeEntriesResponse;
  time_off: TimeOffResponse;
  time_sheet_reviewers: TimeSheetReviewersResponse;
  time_sheets: TimeSheetsResponse;
  time_types: TimeTypesResponse;
  user_claims: UserClaimsResponse;
  users: UsersResponse;
};

// Type for usage with type asserted PocketBase instance
// https://github.com/pocketbase/js-sdk#specify-typescript-definitions

export type TypedPocketBase = PocketBase & {
  collection(idOrName: "admin_profiles"): RecordService<AdminProfilesResponse>;
  collection(idOrName: "categories"): RecordService<CategoriesResponse>;
  collection(idOrName: "claims"): RecordService<ClaimsResponse>;
  collection(idOrName: "clients"): RecordService<ClientsResponse>;
  collection(idOrName: "contacts"): RecordService<ContactsResponse>;
  collection(idOrName: "divisions"): RecordService<DivisionsResponse>;
  collection(idOrName: "expense_rates"): RecordService<ExpenseRatesResponse>;
  collection(idOrName: "expenses"): RecordService<ExpensesResponse>;
  collection(idOrName: "jobs"): RecordService<JobsResponse>;
  collection(idOrName: "managers"): RecordService<ManagersResponse>;
  collection(idOrName: "payroll_year_end_dates"): RecordService<PayrollYearEndDatesResponse>;
  collection(idOrName: "po_approvers"): RecordService<PoApproversResponse>;
  collection(idOrName: "profiles"): RecordService<ProfilesResponse>;
  collection(idOrName: "purchase_orders"): RecordService<PurchaseOrdersResponse>;
  collection(idOrName: "time_amendments"): RecordService<TimeAmendmentsResponse>;
  collection(idOrName: "time_entries"): RecordService<TimeEntriesResponse>;
  collection(idOrName: "time_off"): RecordService<TimeOffResponse>;
  collection(idOrName: "time_sheet_reviewers"): RecordService<TimeSheetReviewersResponse>;
  collection(idOrName: "time_sheets"): RecordService<TimeSheetsResponse>;
  collection(idOrName: "time_types"): RecordService<TimeTypesResponse>;
  collection(idOrName: "user_claims"): RecordService<UserClaimsResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
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
