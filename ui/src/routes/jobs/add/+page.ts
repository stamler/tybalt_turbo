import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import { JobsStatusOptions } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";

export const load: PageLoad<JobsPageData> = async () => {
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
  };

  const defaultCategories = [] as CategoriesResponse[];
  return {
    item: { ...defaultItem } as JobsRecord,
    editing: false,
    id: null,
    categories: defaultCategories,
  };
};
