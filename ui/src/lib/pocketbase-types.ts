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
  AppConfig = "app_config",
  AdminProfilesAugmented = "admin_profiles_augmented",
  Branches = "branches",
  Categories = "categories",
  Claims = "claims",
  ClientContacts = "client_contacts",
  ClientNotes = "client_notes",
  Clients = "clients",
  Divisions = "divisions",
  RateRoles = "rate_roles",
  ExpenseAllowanceTotals = "expense_allowance_totals",
  ExpenseMileageTotals = "expense_mileage_totals",
  ExpenseRates = "expense_rates",
  Expenses = "expenses",
  ExpensesAugmented = "expenses_augmented",
  Jobs = "jobs",
  Managers = "managers",
  MileageResetDates = "mileage_reset_dates",
  NotificationTemplates = "notification_templates",
  Notifications = "notifications",
  PayrollReportWeekEndings = "payroll_report_week_endings",
  PayrollYearEndDates = "payroll_year_end_dates",
  PendingItemsForQualifiedPoSecondApprovers = "pending_items_for_qualified_po_second_approvers",
  PoApprovalThresholds = "po_approval_thresholds",
  PoApproverProps = "po_approver_props",
  Profiles = "profiles",
  PurchaseOrders = "purchase_orders",
  PurchaseOrdersAugmented = "purchase_orders_augmented",
  TimeAmendments = "time_amendments",
  TimeAmendmentsAugmented = "time_amendments_augmented",
  TimeEntries = "time_entries",
  TimeOff = "time_off",
  TimeReportWeekEndings = "time_report_week_endings",
  TimeSheetReviewers = "time_sheet_reviewers",
  TimeSheets = "time_sheets",
  TimeTracking = "time_tracking",
  TimeTypes = "time_types",
  UserClaims = "user_claims",
  UserPoPermissionData = "user_po_permission_data",
  Users = "users",
  Vendors = "vendors",
  ZipCache = "zip_cache",
}

// Alias types for improved usability
export type IsoDateString = string;
export type RecordIdString = string;
export type HTMLString = string;

type ExpandType<T> = unknown extends T
  ? T extends unknown
    ? { expand: unknown }
    : { expand: T }
  : { expand: T };

// System fields
export type BaseSystemFields<T = unknown> = {
  id: RecordIdString;
  collectionId: string;
  collectionName: Collections;
} & ExpandType<T>;

export type AuthSystemFields<T = unknown> = {
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
  _imported: boolean;
  active: boolean;
  allow_personal_reimbursement: boolean;
  created: IsoDateString;
  default_branch: RecordIdString;
  default_charge_out_rate: number;
  id: string;
  job_title: string;
  mobile_phone: string;
  off_rotation_permitted: boolean;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  payroll_id: string;
  personal_vehicle_insurance_expiry: string;
  salary: boolean;
  skip_min_time_check: AdminProfilesSkipMinTimeCheckOptions;
  time_sheet_expected: boolean;
  uid: RecordIdString;
  untracked_time_off: boolean;
  updated: IsoDateString;
  work_week_hours: number;
};

export enum AdminProfilesAugmentedSkipMinTimeCheckOptions {
  "no" = "no",
  "on_next_bundle" = "on_next_bundle",
  "yes" = "yes",
}
export type AdminProfilesAugmentedRecord = {
  active?: boolean;
  allow_personal_reimbursement: boolean;
  default_branch: RecordIdString;
  default_charge_out_rate: number;
  given_name: string;
  id: string;
  job_title: string;
  mobile_phone: string;
  off_rotation_permitted: boolean;
  opening_date: string;
  opening_op: number;
  opening_ov: number;
  payroll_id: string;
  personal_vehicle_insurance_expiry: string;
  po_approver_divisions: null | string[];
  po_approver_max_amount: null | number;
  po_approver_props_id: null | string;
  salary: boolean;
  skip_min_time_check: AdminProfilesAugmentedSkipMinTimeCheckOptions;
  surname: string;
  time_sheet_expected: boolean;
  uid: RecordIdString;
  untracked_time_off: boolean;
  work_week_hours: number;
};

export type AppConfigRecord<Tvalue = unknown> = {
  created: IsoDateString;
  description?: string;
  id: string;
  key: string;
  updated: IsoDateString;
  value: null | Tvalue;
};

export type BranchesRecord = {
  code: string;
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type CategoriesRecord = {
  _imported: boolean;
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
  _imported: boolean;
  client: RecordIdString;
  created: IsoDateString;
  email: string;
  given_name: string;
  id: string;
  surname: string;
  updated: IsoDateString;
};

export enum ClientNotesJobStatusChangedToOptions {
  "No Bid" = "No Bid",
  "Cancelled" = "Cancelled",
}
export type ClientNotesRecord = {
  client: RecordIdString;
  created: IsoDateString;
  id: string;
  job: RecordIdString;
  job_not_applicable: boolean;
  job_status_changed_to?: ClientNotesJobStatusChangedToOptions;
  note: string;
  uid: RecordIdString;
  updated: IsoDateString;
};

export type ClientsRecord = {
  _imported: boolean;
  business_development_lead: RecordIdString;
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type DivisionsRecord = {
  active: boolean;
  code: string;
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export type ExpenditureKindsRecord = {
  created: IsoDateString;
  description: string;
  en_ui_label: string;
  id: string;
  name: string;
  updated: IsoDateString;
};

export enum ExpenseAllowanceTotalsPaymentTypeOptions {
  "OnAccount" = "OnAccount",
  "Expense" = "Expense",
  "CorporateCreditCard" = "CorporateCreditCard",
  "Allowance" = "Allowance",
  "FuelCard" = "FuelCard",
  "Mileage" = "Mileage",
  "PersonalReimbursement" = "PersonalReimbursement",
}

export enum ExpenseAllowanceTotalsAllowanceTypesOptions {
  "Lodging" = "Lodging",
  "Breakfast" = "Breakfast",
  "Lunch" = "Lunch",
  "Dinner" = "Dinner",
}
export type ExpenseAllowanceTotalsRecord = {
  allowance_description: string;
  allowance_rates_effective_date: string;
  allowance_total: number;
  allowance_types: ExpenseAllowanceTotalsAllowanceTypesOptions[];
  breakfast_rate: number;
  date: string;
  dinner_rate: number;
  id: string;
  lodging_rate: number;
  lunch_rate: number;
  mileage: number;
  payment_type: ExpenseAllowanceTotalsPaymentTypeOptions;
};

export type ExpenseMileageTotalsRecord = {
  cumulative: number;
  date: string;
  distance: number;
  effective_date: string;
  id: string;
  mileage_total: number;
  reset_mileage_date: string;
  uid: string;
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
  _imported: boolean;
  allowance_types: ExpensesAllowanceTypesOptions[];
  approved: IsoDateString;
  approver: RecordIdString;
  attachment: string;
  attachment_hash: string;
  branch: RecordIdString;
  category: RecordIdString;
  kind: RecordIdString;
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

export enum ExpensesAugmentedPaymentTypeOptions {
  "OnAccount" = "OnAccount",
  "Expense" = "Expense",
  "CorporateCreditCard" = "CorporateCreditCard",
  "Allowance" = "Allowance",
  "FuelCard" = "FuelCard",
  "Mileage" = "Mileage",
  "PersonalReimbursement" = "PersonalReimbursement",
}

export enum ExpensesAugmentedAllowanceTypesOptions {
  "Lodging" = "Lodging",
  "Breakfast" = "Breakfast",
  "Lunch" = "Lunch",
  "Dinner" = "Dinner",
}
export type ExpensesAugmentedRecord = {
  allowance_types: ExpensesAugmentedAllowanceTypesOptions[];
  approved: IsoDateString;
  approver: RecordIdString;
  approver_name: string;
  attachment: string;
  category: RecordIdString;
  category_name: string;
  kind: RecordIdString;
  cc_last_4_digits: string;
  client_name: string;
  committed: IsoDateString;
  committed_week_ending: string;
  committer: RecordIdString;
  date: string;
  description: string;
  distance: number;
  division: RecordIdString;
  division_code: string;
  division_name: string;
  id: string;
  job: RecordIdString;
  job_description: string;
  job_number: string;
  pay_period_ending: string;
  payment_type: ExpensesAugmentedPaymentTypeOptions;
  purchase_order: RecordIdString;
  purchase_order_number: string;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  rejector_name: string;
  submitted: boolean;
  total: number;
  uid: RecordIdString;
  uid_name: string;
  vendor: RecordIdString;
  vendor_alias: string;
  vendor_name: string;
};

export enum JobsStatusOptions {
  "Active" = "Active",
  "Closed" = "Closed",
  "Cancelled" = "Cancelled",
  "Awarded" = "Awarded",
  "Not Awarded" = "Not Awarded",
  "Submitted" = "Submitted",
  "In Progress" = "In Progress",
  "No Bid" = "No Bid",
}
export type JobsRecord = {
  _imported: boolean;
  alternate_manager: RecordIdString;
  authorizing_document: string;
  branch: RecordIdString;
  client: RecordIdString;
  client_po: string;
  client_reference_number: string;
  contact: RecordIdString;
  created: IsoDateString;
  description: string;
  divisions: RecordIdString[];
  fn_agreement: boolean;
  id: string;
  job_owner: RecordIdString;
  location: string;
  manager: RecordIdString;
  number: string;
  outstanding_balance: number;
  outstanding_balance_date: string;
  project_award_date: string;
  project_value: number;
  proposal: RecordIdString;
  proposal_opening_date: string;
  proposal_submission_due_date: string;
  proposal_value: number;
  status: JobsStatusOptions;
  time_and_materials: boolean;
  updated: IsoDateString;
};

export type ManagersRecord = {
  given_name: string;
  id: string;
  surname: string;
};

export type MileageResetDatesRecord = {
  _imported: boolean;
  created: IsoDateString;
  date: string;
  id: string;
  updated: IsoDateString;
};

export type NotificationTemplatesRecord = {
  code: string;
  created: IsoDateString;
  description: string;
  html_email: string;
  id: string;
  subject: string;
  text_email: string;
  updated: IsoDateString;
};

export enum NotificationsStatusOptions {
  "pending" = "pending",
  "inflight" = "inflight",
  "sent" = "sent",
  "error" = "error",
}
export type NotificationsRecord<Tdata = unknown> = {
  created: IsoDateString;
  data: null | Tdata;
  error: string;
  id: string;
  recipient: RecordIdString;
  status: NotificationsStatusOptions;
  status_updated: IsoDateString;
  system_notification: boolean;
  template: RecordIdString;
  updated: IsoDateString;
  user: RecordIdString;
};

export type PayrollReportWeekEndingsRecord = {
  id: string;
  week_ending: string;
};

export type PayrollYearEndDatesRecord = {
  created: IsoDateString;
  date: string;
  id: string;
  updated: IsoDateString;
};

export type PendingItemsForQualifiedPoSecondApproversRecord = {
  id: string;
  num_pos_qualified: number;
};

export type PoApprovalThresholdsRecord = {
  created: IsoDateString;
  description: string;
  id: string;
  threshold: number;
  updated: IsoDateString;
};

export type PoApproverPropsRecord = {
  computer_max: number;
  created: IsoDateString;
  divisions: RecordIdString[];
  id: string;
  media_and_event_max: number;
  max_amount: number;
  project_max: number;
  sponsorship_max: number;
  staff_and_social_max: number;
  updated: IsoDateString;
  user_claim: RecordIdString;
};

export type RateRolesRecord = {
  created: IsoDateString;
  id: string;
  name: string;
  updated: IsoDateString;
};

export enum ProfilesNotificationTypeOptions {
  "email_text" = "email_text",
  "email_html" = "email_html",
}
export type ProfilesRecord = {
  _imported: boolean;
  alternate_manager: RecordIdString;
  created: IsoDateString;
  default_division: RecordIdString;
  default_role: RecordIdString;
  do_not_accept_submissions: boolean;
  given_name: string;
  id: string;
  manager: RecordIdString;
  notification_type: ProfilesNotificationTypeOptions;
  surname: string;
  uid: RecordIdString;
  updated: IsoDateString;
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
  _imported: boolean;
  approval_total: number;
  approved: IsoDateString;
  approver: RecordIdString;
  attachment: string;
  branch: RecordIdString;
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
  kind: RecordIdString;
  parent_po: RecordIdString;
  payment_type: PurchaseOrdersPaymentTypeOptions;
  po_number: string;
  priority_second_approver: RecordIdString;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  second_approval: IsoDateString;
  second_approver: RecordIdString;
  status: PurchaseOrdersStatusOptions;
  total: number;
  type: PurchaseOrdersTypeOptions;
  uid: RecordIdString;
  updated: IsoDateString;
  vendor: RecordIdString;
};

export enum PurchaseOrdersAugmentedStatusOptions {
  "Unapproved" = "Unapproved",
  "Active" = "Active",
  "Cancelled" = "Cancelled",
  "Closed" = "Closed",
}

export enum PurchaseOrdersAugmentedTypeOptions {
  "Normal" = "Normal",
  "Cumulative" = "Cumulative",
  "Recurring" = "Recurring",
}

export enum PurchaseOrdersAugmentedFrequencyOptions {
  "Weekly" = "Weekly",
  "Biweekly" = "Biweekly",
  "Monthly" = "Monthly",
}

export enum PurchaseOrdersAugmentedPaymentTypeOptions {
  "OnAccount" = "OnAccount",
  "Expense" = "Expense",
  "CorporateCreditCard" = "CorporateCreditCard",
}
export type PurchaseOrdersAugmentedRecord = {
  approval_total: number;
  approved: IsoDateString;
  approver: RecordIdString;
  approver_name: string;
  attachment: string;
  cancelled: IsoDateString;
  canceller: RecordIdString;
  category: RecordIdString;
  category_name: string;
  client_id: RecordIdString;
  client_name: string;
  closed: IsoDateString;
  closed_by_system: boolean;
  closer: RecordIdString;
  committed_expenses_count: number;
  created: IsoDateString;
  date: string;
  description: string;
  division: RecordIdString;
  division_code: string;
  division_name: string;
  end_date: string;
  frequency: PurchaseOrdersAugmentedFrequencyOptions;
  id: string;
  job: RecordIdString;
  job_description: string;
  job_number: string;
  kind: RecordIdString;
  lower_threshold: number;
  parent_po: RecordIdString;
  parent_po_number: string;
  payment_type: PurchaseOrdersAugmentedPaymentTypeOptions;
  po_number: string;
  priority_second_approver: RecordIdString;
  priority_second_approver_name: string;
  rejected: IsoDateString;
  rejection_reason: string;
  rejector: RecordIdString;
  rejector_name: string;
  second_approval: IsoDateString;
  second_approver: RecordIdString;
  second_approver_name: string;
  status: PurchaseOrdersAugmentedStatusOptions;
  total: number;
  type: PurchaseOrdersAugmentedTypeOptions;
  uid: RecordIdString;
  uid_name: string;
  updated: IsoDateString;
  upper_threshold: number;
  vendor: RecordIdString;
  vendor_alias: string;
  vendor_name: string;
};

export type TimeAmendmentsRecord = {
  _imported: boolean;
  branch: RecordIdString;
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

export type TimeAmendmentsAugmentedRecord = {
  category: RecordIdString;
  category_name: string;
  committed: IsoDateString;
  committed_week_ending: string;
  committer: RecordIdString;
  committer_name: string;
  creator: RecordIdString;
  creator_name: string;
  date: string;
  description: string;
  division: RecordIdString;
  division_code: string;
  division_name: string;
  hours: number;
  id: string;
  job: RecordIdString;
  job_description: string;
  job_number: string;
  meals_hours: number;
  payout_request_amount: number;
  skip_tsid_check: boolean;
  time_type: RecordIdString;
  time_type_code: string;
  time_type_name: string;
  tsid: RecordIdString;
  uid: RecordIdString;
  uid_name: string;
  week_ending: string;
  work_record: string;
};

export type TimeEntriesRecord = {
  _imported: boolean;
  branch: RecordIdString;
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
  role: RecordIdString;
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

export type TimeReportWeekEndingsRecord = {
  id: string;
  week_ending: string;
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
  _imported: boolean;
  approved: IsoDateString;
  approver: RecordIdString;
  committed: IsoDateString;
  committer: RecordIdString;
  created: IsoDateString;
  id: string;
  payroll_id: string;
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

export type TimeTrackingRecord = {
  approved_count: number;
  committed_count: number;
  id: string;
  submitted_count: number;
  week_ending: string;
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
  _imported: boolean;
  cid: RecordIdString;
  created: IsoDateString;
  id: string;
  uid: RecordIdString;
  updated: IsoDateString;
};

export type UserPoPermissionDataRecord = {
  claims: string[];
  divisions: string[];
  id: string;
  lower_threshold: number;
  max_amount: number;
  upper_threshold: number;
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
  _imported: boolean;
  alias: string;
  created: IsoDateString;
  id: string;
  name: string;
  status: VendorsStatusOptions;
  updated: IsoDateString;
};

export type ZipCacheRecord<Tfilenames = unknown, Thashes = unknown> = {
  class: string;
  created: IsoDateString;
  filenames: null | Tfilenames;
  hashes: null | Thashes;
  id: string;
  key: string;
  updated: IsoDateString;
  zip: string;
};

type TimeEntriesRecordExpands = {
  category: CategoriesRecord;
  division: DivisionsRecord;
  job: JobsRecord;
  role: RateRolesRecord;
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
  reviewer: ManagersResponse;
  time_sheet: TimeSheetsResponse;
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
  type: PurchaseOrdersTypeOptions;
  uid: UsersResponse;
  vendor: VendorsResponse;
  priority_second_approver: UsersResponse;
};

type UsersRecordExpands = {
  profiles_via_uid: ProfilesResponse;
};

type ManagersRecordExpands = {
  profiles_via_uid: ProfilesResponse;
};

type UserClaimsRecordExpands = {
  cid: ClaimsResponse;
};

type JobsRecordExpands = {
  categories_via_job: CategoriesResponse[];
  client: ClientsResponse;
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
export type AdminProfilesAugmentedResponse<Texpand = unknown> =
  Required<AdminProfilesAugmentedRecord> & BaseSystemFields<Texpand>;
export type AppConfigResponse<Tvalue = unknown, Texpand = unknown> = Required<
  AppConfigRecord<Tvalue>
> &
  BaseSystemFields<Texpand>;
export type BranchesResponse<Texpand = unknown> = Required<BranchesRecord> &
  BaseSystemFields<Texpand>;
export type CategoriesResponse<Texpand = unknown> = Required<CategoriesRecord> &
  BaseSystemFields<Texpand>;
export type ClaimsResponse<Texpand = unknown> = Required<ClaimsRecord> & BaseSystemFields<Texpand>;
export type ClientContactsResponse<Texpand = unknown> = Required<ClientContactsRecord> &
  BaseSystemFields<Texpand>;
export type ClientNotesResponse<Texpand = unknown> = Required<ClientNotesRecord> &
  BaseSystemFields<Texpand>;
export type ClientsResponse<Texpand = unknown> = Required<ClientsRecord> &
  BaseSystemFields<Texpand>;
export type DivisionsResponse<Texpand = unknown> = Required<DivisionsRecord> &
  BaseSystemFields<Texpand>;
export type ExpenditureKindsResponse<Texpand = unknown> = Required<ExpenditureKindsRecord> &
  BaseSystemFields<Texpand>;
export type ExpenseAllowanceTotalsResponse<Texpand = unknown> =
  Required<ExpenseAllowanceTotalsRecord> & BaseSystemFields<Texpand>;
export type ExpenseMileageTotalsResponse<Texpand = unknown> = Required<ExpenseMileageTotalsRecord> &
  BaseSystemFields<Texpand>;
export type ExpenseRatesResponse<Texpand = unknown> = Required<ExpenseRatesRecord> &
  BaseSystemFields<Texpand>;
export type ExpensesResponse<Texpand = ExpensesRecordExpands> = Required<ExpensesRecord> &
  BaseSystemFields<Texpand>;
export type ExpensesAugmentedResponse<Texpand = unknown> = Required<ExpensesAugmentedRecord> &
  BaseSystemFields<Texpand>;
export type JobsResponse<Texpand = JobsRecordExpands> = Required<JobsRecord> &
  BaseSystemFields<Texpand>;
export type ManagersResponse<Texpand = ManagersRecordExpands> = Required<ManagersRecord> &
  BaseSystemFields<Texpand>;
export type MileageResetDatesResponse<Texpand = unknown> = Required<MileageResetDatesRecord> &
  BaseSystemFields<Texpand>;
export type NotificationTemplatesResponse<Texpand = unknown> =
  Required<NotificationTemplatesRecord> & BaseSystemFields<Texpand>;
export type NotificationsResponse<Tdata = unknown, Texpand = unknown> = Required<
  NotificationsRecord<Tdata>
> &
  BaseSystemFields<Texpand>;
export type PayrollReportWeekEndingsResponse<Texpand = unknown> =
  Required<PayrollReportWeekEndingsRecord> & BaseSystemFields<Texpand>;
export type PayrollYearEndDatesResponse<Texpand = unknown> = Required<PayrollYearEndDatesRecord> &
  BaseSystemFields<Texpand>;
export type PendingItemsForQualifiedPoSecondApproversResponse<Texpand = unknown> =
  Required<PendingItemsForQualifiedPoSecondApproversRecord> & BaseSystemFields<Texpand>;
export type PoApprovalThresholdsResponse<Texpand = unknown> = Required<PoApprovalThresholdsRecord> &
  BaseSystemFields<Texpand>;
export type PoApproverPropsResponse<Texpand = unknown> = Required<PoApproverPropsRecord> &
  BaseSystemFields<Texpand>;
export type RateRolesResponse<Texpand = unknown> = Required<RateRolesRecord> &
  BaseSystemFields<Texpand>;
export type ProfilesResponse<Texpand = unknown> = Required<ProfilesRecord> &
  BaseSystemFields<Texpand>;
export type PurchaseOrdersResponse<Texpand = PurchaseOrdersRecordExpands> =
  Required<PurchaseOrdersRecord> & BaseSystemFields<Texpand>;
export type PurchaseOrdersAugmentedResponse<Texpand = unknown> =
  Required<PurchaseOrdersAugmentedRecord> & BaseSystemFields<Texpand>;
export type TimeAmendmentsResponse<Texpand = TimeAmendmentsRecordExpands> =
  Required<TimeAmendmentsRecord> & BaseSystemFields<Texpand>;
export type TimeAmendmentsAugmentedResponse<Texpand = unknown> =
  Required<TimeAmendmentsAugmentedRecord> & BaseSystemFields<Texpand>;
export type TimeEntriesResponse<Texpand = TimeEntriesRecordExpands> = Required<TimeEntriesRecord> &
  BaseSystemFields<Texpand>;
export type TimeOffResponse<Texpand = unknown> = Required<TimeOffRecord> &
  BaseSystemFields<Texpand>;
export type TimeReportWeekEndingsResponse<Texpand = unknown> =
  Required<TimeReportWeekEndingsRecord> & BaseSystemFields<Texpand>;
export type TimeSheetReviewersResponse<Texpand = TimeSheetReviewersRecordExpands> =
  Required<TimeSheetReviewersRecord> & BaseSystemFields<Texpand>;
export type TimeSheetsResponse<Texpand = unknown> = Required<TimeSheetsRecord> &
  BaseSystemFields<Texpand>;
export type TimeTrackingResponse<Texpand = unknown> = Required<TimeTrackingRecord> &
  BaseSystemFields<Texpand>;
export type TimeTypesResponse<Texpand = unknown> = Required<TimeTypesRecord> &
  BaseSystemFields<Texpand>;
export type UserClaimsResponse<Texpand = UserClaimsRecordExpands> = Required<UserClaimsRecord> &
  BaseSystemFields<Texpand>;
export type UserPoPermissionDataResponse<Texpand = unknown> = Required<UserPoPermissionDataRecord> &
  BaseSystemFields<Texpand>;
export type UsersResponse<Texpand = UsersRecordExpands> = Required<UsersRecord> &
  AuthSystemFields<Texpand>;
export type VendorsResponse<Texpand = unknown> = Required<VendorsRecord> &
  BaseSystemFields<Texpand>;
export type ZipCacheResponse<Tfilenames = unknown, Thashes = unknown, Texpand = unknown> = Required<
  ZipCacheRecord<Tfilenames, Thashes>
> &
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
  admin_profiles_augmented: AdminProfilesAugmentedRecord;
  app_config: AppConfigRecord;
  branches: BranchesRecord;
  categories: CategoriesRecord;
  claims: ClaimsRecord;
  client_contacts: ClientContactsRecord;
  client_notes: ClientNotesRecord;
  clients: ClientsRecord;
  divisions: DivisionsRecord;
  expenditure_kinds: ExpenditureKindsRecord;
  expense_allowance_totals: ExpenseAllowanceTotalsRecord;
  expense_mileage_totals: ExpenseMileageTotalsRecord;
  expense_rates: ExpenseRatesRecord;
  expenses: ExpensesRecord;
  expenses_augmented: ExpensesAugmentedRecord;
  jobs: JobsRecord;
  managers: ManagersRecord;
  mileage_reset_dates: MileageResetDatesRecord;
  notification_templates: NotificationTemplatesRecord;
  notifications: NotificationsRecord;
  payroll_report_week_endings: PayrollReportWeekEndingsRecord;
  payroll_year_end_dates: PayrollYearEndDatesRecord;
  pending_items_for_qualified_po_second_approvers: PendingItemsForQualifiedPoSecondApproversRecord;
  po_approval_thresholds: PoApprovalThresholdsRecord;
  po_approver_props: PoApproverPropsRecord;
  rate_roles: RateRolesRecord;
  profiles: ProfilesRecord;
  purchase_orders: PurchaseOrdersRecord;
  purchase_orders_augmented: PurchaseOrdersAugmentedRecord;
  time_amendments: TimeAmendmentsRecord;
  time_amendments_augmented: TimeAmendmentsAugmentedRecord;
  time_entries: TimeEntriesRecord;
  time_off: TimeOffRecord;
  time_report_week_endings: TimeReportWeekEndingsRecord;
  time_sheet_reviewers: TimeSheetReviewersRecord;
  time_sheets: TimeSheetsRecord;
  time_tracking: TimeTrackingRecord;
  time_types: TimeTypesRecord;
  user_claims: UserClaimsRecord;
  user_po_permission_data: UserPoPermissionDataRecord;
  users: UsersRecord;
  vendors: VendorsRecord;
  zip_cache: ZipCacheRecord;
};

export type CollectionResponses = {
  _authOrigins: AuthoriginsResponse;
  _externalAuths: ExternalauthsResponse;
  _mfas: MfasResponse;
  _otps: OtpsResponse;
  _superusers: SuperusersResponse;
  absorb_actions: AbsorbActionsResponse;
  admin_profiles: AdminProfilesResponse;
  admin_profiles_augmented: AdminProfilesAugmentedResponse;
  app_config: AppConfigResponse;
  branches: BranchesResponse;
  categories: CategoriesResponse;
  claims: ClaimsResponse;
  client_contacts: ClientContactsResponse;
  client_notes: ClientNotesResponse;
  clients: ClientsResponse;
  divisions: DivisionsResponse;
  expenditure_kinds: ExpenditureKindsResponse;
  expense_allowance_totals: ExpenseAllowanceTotalsResponse;
  expense_mileage_totals: ExpenseMileageTotalsResponse;
  expense_rates: ExpenseRatesResponse;
  expenses: ExpensesResponse;
  expenses_augmented: ExpensesAugmentedResponse;
  jobs: JobsResponse;
  managers: ManagersResponse;
  mileage_reset_dates: MileageResetDatesResponse;
  notification_templates: NotificationTemplatesResponse;
  notifications: NotificationsResponse;
  payroll_report_week_endings: PayrollReportWeekEndingsResponse;
  payroll_year_end_dates: PayrollYearEndDatesResponse;
  pending_items_for_qualified_po_second_approvers: PendingItemsForQualifiedPoSecondApproversResponse;
  po_approval_thresholds: PoApprovalThresholdsResponse;
  po_approver_props: PoApproverPropsResponse;
  rate_roles: RateRolesResponse;
  profiles: ProfilesResponse;
  purchase_orders: PurchaseOrdersResponse;
  purchase_orders_augmented: PurchaseOrdersAugmentedResponse;
  time_amendments: TimeAmendmentsResponse;
  time_amendments_augmented: TimeAmendmentsAugmentedResponse;
  time_entries: TimeEntriesResponse;
  time_off: TimeOffResponse;
  time_report_week_endings: TimeReportWeekEndingsResponse;
  time_sheet_reviewers: TimeSheetReviewersResponse;
  time_sheets: TimeSheetsResponse;
  time_tracking: TimeTrackingResponse;
  time_types: TimeTypesResponse;
  user_claims: UserClaimsResponse;
  user_po_permission_data: UserPoPermissionDataResponse;
  users: UsersResponse;
  vendors: VendorsResponse;
  zip_cache: ZipCacheResponse;
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
  collection(idOrName: "admin_profiles_augmented"): RecordService<AdminProfilesAugmentedResponse>;
  collection(idOrName: "app_config"): RecordService<AppConfigResponse>;
  collection(idOrName: "branches"): RecordService<BranchesResponse>;
  collection(idOrName: "categories"): RecordService<CategoriesResponse>;
  collection(idOrName: "claims"): RecordService<ClaimsResponse>;
  collection(idOrName: "client_contacts"): RecordService<ClientContactsResponse>;
  collection(idOrName: "client_notes"): RecordService<ClientNotesResponse>;
  collection(idOrName: "clients"): RecordService<ClientsResponse>;
  collection(idOrName: "divisions"): RecordService<DivisionsResponse>;
  collection(idOrName: "expense_allowance_totals"): RecordService<ExpenseAllowanceTotalsResponse>;
  collection(idOrName: "expense_mileage_totals"): RecordService<ExpenseMileageTotalsResponse>;
  collection(idOrName: "expense_rates"): RecordService<ExpenseRatesResponse>;
  collection(idOrName: "expenses"): RecordService<ExpensesResponse>;
  collection(idOrName: "expenses_augmented"): RecordService<ExpensesAugmentedResponse>;
  collection(idOrName: "jobs"): RecordService<JobsResponse>;
  collection(idOrName: "managers"): RecordService<ManagersResponse>;
  collection(idOrName: "mileage_reset_dates"): RecordService<MileageResetDatesResponse>;
  collection(idOrName: "notification_templates"): RecordService<NotificationTemplatesResponse>;
  collection(idOrName: "notifications"): RecordService<NotificationsResponse>;
  collection(
    idOrName: "payroll_report_week_endings",
  ): RecordService<PayrollReportWeekEndingsResponse>;
  collection(idOrName: "payroll_year_end_dates"): RecordService<PayrollYearEndDatesResponse>;
  collection(
    idOrName: "pending_items_for_qualified_po_second_approvers",
  ): RecordService<PendingItemsForQualifiedPoSecondApproversResponse>;
  collection(idOrName: "po_approval_thresholds"): RecordService<PoApprovalThresholdsResponse>;
  collection(idOrName: "po_approver_props"): RecordService<PoApproverPropsResponse>;
  collection(idOrName: "rate_roles"): RecordService<RateRolesResponse>;
  collection(idOrName: "profiles"): RecordService<ProfilesResponse>;
  collection(idOrName: "purchase_orders"): RecordService<PurchaseOrdersResponse>;
  collection(idOrName: "purchase_orders_augmented"): RecordService<PurchaseOrdersAugmentedResponse>;
  collection(idOrName: "time_amendments"): RecordService<TimeAmendmentsResponse>;
  collection(idOrName: "time_amendments_augmented"): RecordService<TimeAmendmentsAugmentedResponse>;
  collection(idOrName: "time_entries"): RecordService<TimeEntriesResponse>;
  collection(idOrName: "time_off"): RecordService<TimeOffResponse>;
  collection(idOrName: "time_report_week_endings"): RecordService<TimeReportWeekEndingsResponse>;
  collection(idOrName: "time_sheet_reviewers"): RecordService<TimeSheetReviewersResponse>;
  collection(idOrName: "time_sheets"): RecordService<TimeSheetsResponse>;
  collection(idOrName: "time_tracking"): RecordService<TimeTrackingResponse>;
  collection(idOrName: "time_types"): RecordService<TimeTypesResponse>;
  collection(idOrName: "user_claims"): RecordService<UserClaimsResponse>;
  collection(idOrName: "user_po_permission_data"): RecordService<UserPoPermissionDataResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
  collection(idOrName: "vendors"): RecordService<VendorsResponse>;
  collection(idOrName: "zip_cache"): RecordService<ZipCacheResponse>;
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

// This is defined in the app/utilities/po_approvers.go file
export type PoApproversResponse = {
  id: string;
  given_name: string;
  surname: string;
};

export type ClientDetails = {
  id: string;
  name: string;
  business_development_lead: string;
  lead_given_name: string;
  lead_surname: string;
  lead_email: string;
  outstanding_balance: number;
  outstanding_balance_date: string;
  contacts: {
    id: string;
    given_name: string;
    surname: string;
    email: string;
  }[];
  referencing_jobs_count: number;
};
