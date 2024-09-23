import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";

export const load: PageLoad<JobsPageData> = async () => {
  const defaultItem = {
    number: "",
    description: "",
  };

  const defaultCategories = [] as CategoriesResponse[];
  return {
    item: { ...defaultItem } as JobsRecord,
    editing: false,
    id: null,
    categories: defaultCategories,
  };
};
