import type { AdminProfilesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  try {
    type AdminProfilesAugmented = AdminProfilesResponse & {
      given_name: string;
      surname: string;
    };
    const items = await pb
      .collection("admin_profiles_augmented")
      .getFullList<AdminProfilesAugmented>({ sort: "surname,given_name" });
    return { items };
  } catch (error) {
    console.error(`loading admin_profiles list: ${error}`);
    return { items: [] as AdminProfilesResponse[] };
  }
};
