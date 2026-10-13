import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { ClaimHolder } from "$lib/svelte-types";
import type { ProjectAuthorizationPropsResponse } from "$lib/pocketbase-types";

// Load holders from the claim details route, which reads their profiles in
// SQL. Expanding each holder's user record applied the users view rule, whose
// expense and purchase order lookups made the request hang on production data.
export const load: PageLoad = async () => {
  const claim = await pb.collection("claims").getFirstListItem('name="project_authorization"');
  const [details, props] = await Promise.all([
    pb.send<{ holders: Omit<ClaimHolder, "id">[] }>(`/api/claims/${claim.id}`, { method: "GET" }),
    pb.collection("project_authorization_props").getFullList<ProjectAuthorizationPropsResponse>(),
  ]);
  return { claimId: claim.id, holders: details.holders, props };
};
