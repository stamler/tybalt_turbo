import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import { JobsStatusOptions } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

// Helper to get today's date in YYYY-MM-DD format
function getTodayDateString(): string {
  const today = new Date();
  const year = today.getFullYear();
  const month = String(today.getMonth() + 1).padStart(2, "0");
  const day = String(today.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export const load: PageLoad<JobsPageData> = async ({ params, url }) => {
  const details = await pb.send(`/api/jobs/${params.proposal}/details`, { method: "GET" });

  // Check if setAwardToday query param is present
  const setAwardToday = url.searchParams.get("setAwardToday") === "true";

  // Extract any existing projects that already reference this proposal
  const existingReferencingProjects: { id: string; number: string }[] = Array.isArray(
    details?.projects,
  )
    ? details.projects
    : [];

  const item: Partial<JobsRecord> & { _prefilled_from_proposal: true } = {
    number: "",
    description: details?.description ?? "",
    client: details?.client?.id ?? "",
    contact: details?.contact?.id ?? "",
    manager: details?.manager?.id ?? "",
    alternate_manager: details?.alternate_manager?.id ?? "",
    divisions: Array.isArray(details?.divisions)
      ? details.divisions.map((d: { id: string }) => d.id)
      : [],
    location: details?.location ?? "",
    branch: details?.branch_id ?? "",
    job_owner: details?.job_owner?.id ?? "",
    fn_agreement: details?.fn_agreement ?? false,
    status: JobsStatusOptions.Active,
    proposal: params.proposal,
    authorizing_document: "",
    client_po: "",
    client_reference_number: "",
    // Set project_award_date to today if setAwardToday param is present
    project_award_date: setAwardToday ? getTodayDateString() : "",
    proposal_opening_date: "",
    proposal_submission_due_date: "",
    // Copy proposal_value to project_value (user can edit before saving)
    project_value: details?.proposal_value ?? 0,
    // Copy time_and_materials flag from proposal
    time_and_materials: details?.time_and_materials ?? false,
    _prefilled_from_proposal: true,
  };

  return {
    item: item as JobsRecord,
    editing: false,
    id: null,
    categories: [] as CategoriesResponse[],
    existingReferencingProjects,
  };
};
