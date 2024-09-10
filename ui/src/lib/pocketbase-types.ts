/**
 * This file was @generated using pocketbase-typegen
 */

import type PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";

export enum Collections {
  AdminProfiles = "admin_profiles",
  Claims = "claims",
  Divisions = "divisions",
  Jobs = "jobs",
  Managers = "managers",
  PayrollYearEndDates = "payroll_year_end_dates",
  Profiles = "profiles",
  PurchaseOrders = "purchase_orders",
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
  salary: boolean;
  uid: RecordIdString;
  work_week_hours: number;
  off_rotation_permitted: boolean;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  payroll_id: string;
  skip_min_time_check: AdminProfilesSkipMinTimeCheckOptions;
};

export type ClaimsRecord = {
  description: string;
  name: string;
};

export type DivisionsRecord = {
  code: string;
  name: string;
};

export type JobsRecord = {
  number: string;
  description: string;
};

export type ProfilesRecord = {
  given_name: string;
  surname: string;
  manager: RecordIdString;
  alternate_manager: RecordIdString;
  default_division: RecordIdString;
  uid: RecordIdString;
};

export type PayrollYearEndDatesRecord = {
  date: string;
};

export enum PurchaseOrdersStatusOptions {
  "Unapproved" = "Unapproved",
  "Active" = "Active",
  "Cancelled" = "Cancelled",
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
  date: string;
  description: string;
  division: RecordIdString;
  end_date: string;
  frequency: PurchaseOrdersFrequencyOptions;
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

export type TimeEntriesRecord = {
  category: string;
  date: string;
  description: string;
  division: RecordIdString;
  hours: number;
  job: RecordIdString;
  meals_hours: number;
  payout_request_amount: number;
  time_type: RecordIdString;
  uid: RecordIdString;
  week_ending: string;
  work_record: string;
  tsid: RecordIdString;
};

export type TimeSheetsRecord = {
  approved: IsoDateString;
  approver: RecordIdString;
  locked: boolean;
  locker: RecordIdString;
  rejected: boolean;
  rejection_reason: string;
  rejector: RecordIdString;
  submitted: boolean;
  uid: RecordIdString;
  salary: boolean;
  week_ending: string;
  work_week_hours: number;
};

export type TimeOffRecord = {
  name: string;
  last_op: string;
  last_ov: string;
  manager: string;
  manager_uid: RecordIdString;
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

export type TimeTypesRecord = {
  allowed_fields: null | string[];
  required_fields: null | string[];
  code: string;
  description: string;
  name: string;
};

export type UserClaimsRecord = {
  cid: RecordIdString;
  uid: RecordIdString;
};

export type UsersRecord = {
  name: string;
};

export type ManagersRecord = {
  given_name: string;
  surname: string;
};

type TimeEntriesRecordExpands = {
  time_type: TimeTypesRecord;
  division: DivisionsRecord;
  job: JobsRecord;
};

type TimeSheetReviewersRecordExpands = {
  time_sheet: TimeSheetsRecord;
  reviewer: ManagersRecord;
};

// Response types include system fields and match responses from the PocketBase API
export type AdminProfilesResponse<Texpand = unknown> = Required<AdminProfilesRecord> &
  BaseSystemFields<Texpand>;
export type ClaimsResponse<Texpand = unknown> = Required<ClaimsRecord> & BaseSystemFields<Texpand>;
export type DivisionsResponse<Texpand = unknown> = Required<DivisionsRecord> &
  BaseSystemFields<Texpand>;
export type JobsResponse<Texpand = unknown> = Required<JobsRecord> & BaseSystemFields<Texpand>;
export type ManagersResponse<Texpand = unknown> = Required<ManagersRecord> &
  BaseSystemFields<Texpand>;
export type PayrollYearEndDatesResponse<Texpand = unknown> = Required<PayrollYearEndDatesRecord> &
  BaseSystemFields<Texpand>;
export type ProfilesResponse<Texpand = unknown> = Required<ProfilesRecord> &
  BaseSystemFields<Texpand>;
export type PurchaseOrdersResponse<Texpand = unknown> = Required<PurchaseOrdersRecord> &
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
export type UsersResponse<Texpand = unknown> = Required<UsersRecord> & AuthSystemFields<Texpand>;

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
  admin_profiles: AdminProfilesRecord;
  claims: ClaimsRecord;
  divisions: DivisionsRecord;
  jobs: JobsRecord;
  managers: ManagersRecord;
  payroll_year_end_dates: PayrollYearEndDatesRecord;
  profiles: ProfilesRecord;
  purchase_orders: PurchaseOrdersRecord;
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
  claims: ClaimsResponse;
  divisions: DivisionsResponse;
  jobs: JobsResponse;
  managers: ManagersResponse;
  payroll_year_end_dates: PayrollYearEndDatesResponse;
  profiles: ProfilesResponse;
  purchase_orders: PurchaseOrdersResponse;
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
  collection(idOrName: "claims"): RecordService<ClaimsResponse>;
  collection(idOrName: "divisions"): RecordService<DivisionsResponse>;
  collection(idOrName: "jobs"): RecordService<JobsResponse>;
  collection(idOrName: "managers"): RecordService<ManagersResponse>;
  collection(idOrName: "payroll_year_end_dates"): RecordService<PayrollYearEndDatesResponse>;
  collection(idOrName: "profiles"): RecordService<ProfilesResponse>;
  collection(idOrName: "purchase_orders"): RecordService<PurchaseOrdersResponse>;
  collection(idOrName: "time_entries"): RecordService<TimeEntriesResponse>;
  collection(idOrName: "time_off"): RecordService<TimeOffResponse>;
  collection(idOrName: "time_sheet_reviewers"): RecordService<TimeSheetReviewersResponse>;
  collection(idOrName: "time_sheets"): RecordService<TimeSheetsResponse>;
  collection(idOrName: "time_types"): RecordService<TimeTypesResponse>;
  collection(idOrName: "user_claims"): RecordService<UserClaimsResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
};

export type HasId = { id: RecordIdString };
