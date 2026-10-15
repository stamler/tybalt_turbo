export interface FilterDef {
  type: string;
  label: string;
  queryParam?: string;
  summaryProperty: string;
  valueProperty: string;
  displayProperty: string;
  color: "blue" | "green" | "purple" | "teal" | "yellow" | "red" | "gray";
}

// The response of GET /api/jobs/{id}/details (JobDetails in
// app/routes/job_details_api.go). Empty strings and zero-value objects mean
// "not set"; lists may be null when empty.
export interface JobDetailsPerson {
  id: string;
  given_name: string;
  surname: string;
}

export interface JobDetailsClient {
  id: string;
  name: string;
  alias: string;
}

export interface JobRef {
  id: string;
  number: string;
}

export interface JobDetails {
  id: string;
  number: string;
  description: string;
  status: string;
  imported: boolean;
  parent_id: string;
  parent_number: string;
  location: string;
  authorizing_document: string;
  client_po: string;
  client_reference_number: string;
  client: JobDetailsClient;
  contact: JobDetailsPerson;
  manager: JobDetailsPerson;
  alternate_manager: JobDetailsPerson;
  job_owner: JobDetailsClient;
  proposal_id: string;
  proposal_number: string;
  branch_id: string;
  branch_code: string;
  branch_name: string;
  rate_sheet: { id: string; name: string; revision: number };
  fn_agreement: boolean;
  project_award_date: string;
  project_completion_date: string;
  invoicing_information: string;
  invoicing_profile_name: string;
  invoice_contact_name: string;
  invoice_contact_email: string;
  invoice_instructions: string;
  request_creator: string;
  request_created: string;
  request_submitted: string;
  created_by: string;
  request_creator_name: string;
  created_by_name: string;
  creation_completed: string;
  proposal_opening_date: string;
  proposal_submission_due_date: string;
  proposal_value: number;
  project_value: number;
  time_and_materials: boolean;
  outstanding_balance: number;
  outstanding_balance_date: string;
  allocations:
    { id: string; division: { id: string; code: string; name: string }; hours: number }[] | null;
  projects: JobRef[] | null;
  children: JobRef[] | null;
  categories: { id: string; name: string }[] | null;
}
