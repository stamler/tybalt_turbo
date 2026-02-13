import type { PageLoad } from "./$types";
import type {
  AdminProfilesAugmentedResponse,
  ClaimsResponse,
  UserClaimsResponse,
  BranchesResponse,
  DivisionsResponse,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params }) => {
  try {
    const item = await pb
      .collection("admin_profiles_augmented")
      .getOne<AdminProfilesAugmentedResponse>(params.id);

    let defaultBranch: { id: string; name: string } | null = null;
    try {
      const defaultBranchId = item.default_branch;
      if (defaultBranchId) {
        const b = await pb.collection("branches").getOne<BranchesResponse>(defaultBranchId);
        if (b) defaultBranch = { id: b.id, name: b.name };
      }
    } catch {
      // noop
    }

    let claims: Array<{ id: string; name: string }> = [];
    let poApproverDivisions: string[] = [];
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
    poApproverDivisions = Array.isArray(item.po_approver_divisions)
      ? item.po_approver_divisions
      : [];

    let poApproverDivisionsMap = new Map<string, DivisionsResponse>();
    if (poApproverDivisions.length > 0) {
      try {
        const divisions = await pb.collection("divisions").getFullList<DivisionsResponse>({
          filter: poApproverDivisions.map((id) => `id="${id}"`).join(" || "),
          sort: "code",
        });
        poApproverDivisionsMap = new Map(divisions.map((division) => [division.id, division]));
      } catch {
        // noop
      }
    }

    return {
      item,
      claims,
      defaultBranch,
      poApproverDivisions: poApproverDivisionsMap,
    };
  } catch (error) {
    console.error(`error loading admin_profile details: ${error}`);
    return {
      item: null,
      claims: [],
      defaultBranch: null,
      poApproverDivisions: new Map<string, DivisionsResponse>(),
    };
  }
};
