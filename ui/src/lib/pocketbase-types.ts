/**
 * This file was @generated using pocketbase-typegen
 */

import type PocketBase from "pocketbase";
import type { RecordService } from "pocketbase";

export enum Collections {
  Divisions = "divisions",
  Jobs = "jobs",
  Profiles = "profiles",
  TimeEntries = "time_entries",
  TimeTypes = "time_types",
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
  expand?: T;
};

export type AuthSystemFields<T = never> = {
  email: string;
  emailVisibility: boolean;
  username: string;
  verified: boolean;
} & BaseSystemFields<T>;

// Record types for each collection

export type DivisionsRecord = {
  id: RecordIdString;
  code: string;
  name: string;
};

export type JobsRecord = {
  id: RecordIdString;
  number: string;
  description: string;
};

export type ProfilesRecord = {
  id: RecordIdString;
  given_name: string;
  surname: string;
  manager: RecordIdString;
  alternate_manager: RecordIdString;
  default_division: RecordIdString;
  uid: RecordIdString;
};

export type TimeEntriesRecord = {
  id?: RecordIdString;
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  expand?: Record<string, any>;
};

export type TimeTypesRecord<Tfields = unknown> = {
  id: RecordIdString;
  allowed_fields?: null | Tfields;
  required_fields?: null | Tfields;
  code: string;
  description?: string;
  name: string;
};

export type UsersRecord = {
  name: string;
  opening_ov: number;
  opening_op: number;
  opening_date: string;
  untracked_time_off: boolean;
  default_charge_out_rate: number;
};

export type ManagersRecord = {
  id: RecordIdString;
  given_name: string;
  surname: string;
};

// Response types include system fields and match responses from the PocketBase API
export type DivisionsResponse<Texpand = unknown> = Required<DivisionsRecord> &
  BaseSystemFields<Texpand>;
export type JobsResponse<Texpand = unknown> = Required<JobsRecord> & BaseSystemFields<Texpand>;
export type ProfilesResponse<Texpand = unknown> = Required<ProfilesRecord> &
  BaseSystemFields<Texpand>;
export type TimeEntriesResponse<Texpand = unknown> = Required<TimeEntriesRecord> &
  BaseSystemFields<Texpand>;
export type TimeTypesResponse<Tfields = unknown, Texpand = unknown> = Required<
  TimeTypesRecord<Tfields>
> &
  BaseSystemFields<Texpand>;
export type UsersResponse<Texpand = unknown> = Required<UsersRecord> & AuthSystemFields<Texpand>;

// Types containing all Records and Responses, useful for creating typing helper functions

export type CollectionRecords = {
  divisions: DivisionsRecord;
  jobs: JobsRecord;
  profiles: ProfilesRecord;
  time_entries: TimeEntriesRecord;
  time_types: TimeTypesRecord;
  users: UsersRecord;
};

export type CollectionResponses = {
  divisions: DivisionsResponse;
  jobs: JobsResponse;
  profiles: ProfilesResponse;
  time_entries: TimeEntriesResponse;
  time_types: TimeTypesResponse;
  users: UsersResponse;
};

// Type for usage with type asserted PocketBase instance
// https://github.com/pocketbase/js-sdk#specify-typescript-definitions

export type TypedPocketBase = PocketBase & {
  collection(idOrName: "divisions"): RecordService<DivisionsResponse>;
  collection(idOrName: "jobs"): RecordService<JobsResponse>;
  collection(idOrName: "profiles"): RecordService<ProfilesResponse>;
  collection(idOrName: "time_entries"): RecordService<TimeEntriesResponse>;
  collection(idOrName: "time_types"): RecordService<TimeTypesResponse>;
  collection(idOrName: "users"): RecordService<UsersResponse>;
};

export type HasId = { id: RecordIdString };
