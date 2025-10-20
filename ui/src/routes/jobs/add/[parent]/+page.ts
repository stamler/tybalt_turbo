import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import { JobsStatusOptions } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";
import type { JobApiResponse } from "$lib/stores/jobs";
import { pb } from "$lib/pocketbase";

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
    project_award_date: "",
    proposal_opening_date: "",
    proposal_submission_due_date: "",
    // set parent to the route param
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
  } as any;

  // If a parent is provided, fetch it to pre-populate the client and set parent
  if (params.parent) {
    try {
      const parent: JobApiResponse = await pb.send(`/api/jobs/${params.parent}`, { method: "GET" });
      if (parent?.client_id) {
        defaultItem.client = parent.client_id;
      }
    } catch {
      // ignore errors; backend will enforce parent/client and UI will disable client anyway
    }
    // attach parent id as an extra property (JobsEditor allows extra fields)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (defaultItem as any).parent = params.parent;
  }

  const defaultCategories = [] as CategoriesResponse[];
  return {
    item: { ...defaultItem } as JobsRecord,
    editing: false,
    id: null,
    categories: defaultCategories,
  };
};
