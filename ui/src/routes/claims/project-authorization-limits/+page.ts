import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
export const load: PageLoad = async () => {
  const [claims, props] = await Promise.all([
    pb
      .collection("user_claims")
      .getFullList({ filter: 'cid.name="project_authorization"', expand: "uid" }),
    pb.collection("project_authorization_props").getFullList(),
  ]);
  const profiles = await pb.collection("profiles").getFullList();
  return { claims, props, profiles };
};
