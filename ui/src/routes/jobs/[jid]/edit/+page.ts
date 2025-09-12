import type { JobsRecord, CategoriesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { JobsPageData } from "$lib/svelte-types";

export const load: PageLoad<JobsPageData> = async ({ params }) => {
  const defaultItem = {
    number: "",
    description: "",
    client: "",
    contact: "",
    manager: "",
    location: "",
  };
  const defaultCategories = [] as CategoriesResponse[];
  let item: JobsRecord;
  try {
    item = await pb.collection("jobs").getOne(params.jid);
    const categories = await pb.collection("categories").getFullList({
      filter: `job="${params.jid}"`,
      sort: "name",
    });
    return { item, editing: true, id: params.jid, categories };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return {
      item: { ...defaultItem } as JobsRecord,
      editing: false,
      id: null,
      categories: defaultCategories,
    };
  }
};
