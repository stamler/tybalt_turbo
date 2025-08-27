import type { PageLoad } from "./$types";
import type { AdminProfilesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params }) => {
  try {
    type AdminProfilesAugmented = AdminProfilesResponse & { given_name: string; surname: string };
    const item = await pb
      .collection("admin_profiles_augmented")
      .getOne<AdminProfilesAugmented>(params.id);
    return { item };
  } catch (error) {
    console.error(`error loading admin_profile details: ${error}`);
    return { item: null };
  }
};
