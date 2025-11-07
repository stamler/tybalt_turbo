import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import { JobsStatusOptions } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad<JobsPageData> = async ({ params }) => {
  const details = await pb.send(`/api/jobs/${params.proposal}/details`, { method: "GET" });

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
    project_award_date: "",
    proposal_opening_date: "",
    proposal_submission_due_date: "",
    _prefilled_from_proposal: true,
  };

  return {
    item: item as JobsRecord,
    editing: false,
    id: null,
    categories: [] as CategoriesResponse[],
  };
};
