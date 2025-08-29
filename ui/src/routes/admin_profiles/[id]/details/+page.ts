import type { PageLoad } from "./$types";
import type {
  AdminProfilesResponse,
  ClaimsResponse,
  UserClaimsResponse,
  BranchesResponse,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params }) => {
  try {
    type AdminProfilesAugmented = AdminProfilesResponse & { given_name: string; surname: string };
    const item = await pb
      .collection("admin_profiles_augmented")
      .getOne<AdminProfilesAugmented>(params.id);

    // Load default branch name if set
    let defaultBranch: { id: string; name: string } | null = null;
    try {
      const defaultBranchId = (item as any)?.default_branch as string | undefined;
      if (defaultBranchId) {
        const b = await pb.collection("branches").getOne<BranchesResponse>(defaultBranchId);
        if (b) defaultBranch = { id: b.id, name: b.name };
      }
    } catch {
      // noop
    }

    // Load claims for this user's uid and expand to include claim names
    let claims: Array<{ id: string; name: string }> = [];
    if (item?.uid) {
      try {
        const list = await pb.collection("user_claims").getFullList<UserClaimsResponse>({
          filter: `uid="${item.uid}"`,
          expand: "cid",
        });
        claims = list
          .map((uc) => uc.expand?.cid)
          .filter((c): c is ClaimsResponse => !!c)
          .map((c) => ({ id: c.id, name: c.name }));
      } catch {
        // noop
      }
    }

    return { item, claims, defaultBranch };
  } catch (error) {
    console.error(`error loading admin_profile details: ${error}`);
    return { item: null, claims: [], defaultBranch: null };
  }
};
