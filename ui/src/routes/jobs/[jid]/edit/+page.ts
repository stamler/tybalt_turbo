import type { JobsRecord } from "$lib/pocketbase-types";
import { error } from "@sveltejs/kit";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";
import { JobsStatusOptions } from "$lib/pocketbase-types";

export const load: PageLoad<JobsPageData> = async ({ params }) => {
  const defaultItem: Partial<JobsRecord> = {
    number: "",
    description: "",
    client: "",
    contact: "",
    manager: "",
    alternate_manager: "",
    fn_agreement: false,
    status: JobsStatusOptions.Active,
    proposal: "",
    divisions: [],
    job_owner: "",
    branch: "",
    location: "",
    authorizing_document: "",
    client_po: "",
    client_reference_number: "",
    project_award_date: "",
    project_completion_date: "",
    proposal_opening_date: "",
    proposal_submission_due_date: "",
  };
  let item: JobsRecord;
  try {
    item = await pb.collection("jobs").getOne(params.jid);
    const categories = await pb.collection("categories").getFullList({
      filter: `job="${params.jid}"`,
      sort: "name",
    });
    return {
      item: {
        ...defaultItem,
        ...item,
        divisions: item.divisions ?? [],
        alternate_manager: item.alternate_manager ?? "",
        proposal: item.proposal ?? "",
        job_owner: item.job_owner ?? "",
        branch: item.branch ?? "",
      } as JobsRecord,
      editing: true,
      id: params.jid,
      categories,
    };
  } catch (cause) {
    const status = (cause as { status?: number }).status;
    throw error(status === 403 || status === 404 ? status : 500, "Could not load this job.");
  }
};
