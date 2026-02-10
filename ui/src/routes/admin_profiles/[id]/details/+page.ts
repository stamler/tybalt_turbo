import type { PageLoad } from "./$types";
import type {
  AdminProfilesResponse,
  ClaimsResponse,
  UserClaimsResponse,
  BranchesResponse,
  DivisionsResponse,
  PoApproverPropsResponse,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params }) => {
  try {
    type AdminProfilesAugmented = AdminProfilesResponse & {
      active?: boolean;
      given_name: string;
      surname: string;
    };
    const item = await pb
      .collection("admin_profiles_augmented")
      .getOne<AdminProfilesAugmented>(params.id);

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

    let claims: Array<{ id: string; name: string }> = [];
    let poApproverProps: PoApproverPropsResponse | null = null;
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

        const poApproverUserClaim = list.find((uc) => uc.expand?.cid?.name === "po_approver");
        if (poApproverUserClaim?.id) {
          const props = await pb.collection("po_approver_props").getFullList<PoApproverPropsResponse>({
            filter: `user_claim="${poApproverUserClaim.id}"`,
          });
          if (props.length > 0) {
            poApproverProps = props[0];
            poApproverDivisions = Array.isArray(poApproverProps.divisions)
              ? [...poApproverProps.divisions]
              : [];
          }
        }
      } catch {
        // noop
      }
    }

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
      poApproverProps,
      poApproverDivisions: poApproverDivisionsMap,
    };
  } catch (error) {
    console.error(`error loading admin_profile details: ${error}`);
    return {
      item: null,
      claims: [],
      defaultBranch: null,
      poApproverProps: null,
      poApproverDivisions: new Map<string, DivisionsResponse>(),
    };
  }
};
