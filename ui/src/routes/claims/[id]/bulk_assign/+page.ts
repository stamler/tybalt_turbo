import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { ClaimAssignableUsers } from "$lib/svelte-types";

export const load: PageLoad = async ({ params }) => {
  try {
    const item = (await pb.send(`/api/claims/${params.id}/assignable_users`, {
      method: "GET",
    })) as ClaimAssignableUsers;
    return { item, error: null };
  } catch (error) {
    console.error(`loading claim assignable users: ${error}`);
    return { item: null, error: "You do not have permission to bulk assign this claim." };
  }
};
