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

function normalizeNumber(value: unknown): number {
  if (typeof value === "number") return Number.isFinite(value) ? value : 0;
  if (typeof value === "string") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}

function normalizeDivisions(value: unknown): string[] {
  if (Array.isArray(value)) {
    return value.filter((id): id is string => typeof id === "string");
  }
  if (typeof value === "string" && value.trim().startsWith("[")) {
    try {
      const parsed = JSON.parse(value);
      if (Array.isArray(parsed)) {
        return parsed.filter((id): id is string => typeof id === "string");
      }
    } catch {
      // noop
    }
  }
  return [];
}

export const load: PageLoad = async ({ params }) => {
  try {
    type AdminProfilesAugmented = AdminProfilesResponse & {
      active?: boolean;
      given_name: string;
      surname: string;
      po_approver_max_amount?: number | string | null;
      po_approver_divisions?: unknown;
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

    const poApproverDivisions = normalizeDivisions(item.po_approver_divisions);
    const poApproverProps: PoApproverPropsResponse | null =
      item.po_approver_max_amount !== undefined || poApproverDivisions.length > 0
        ? {
            id: "",
            created: "",
            updated: "",
            user_claim: "",
            max_amount: normalizeNumber(item.po_approver_max_amount),
            divisions: poApproverDivisions,
          }
        : null;

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
