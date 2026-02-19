import type { AdminProfilesAugmentedResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  try {
    const items = await pb
      .collection("admin_profiles_augmented")
      .getFullList<AdminProfilesAugmentedResponse>({
        filter: "po_approver_props_id != null && active = true",
        sort: "surname,given_name",
      });
    return { items };
  } catch (error) {
    console.error(`loading po approvers list: ${error}`);
    return { items: [] as AdminProfilesAugmentedResponse[] };
  }
};
