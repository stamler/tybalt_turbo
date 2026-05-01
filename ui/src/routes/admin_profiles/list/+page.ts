import { AdminProfilesAugmentedSkipMinTimeCheckOptions, Collections } from "$lib/pocketbase-types";
import type { AdminProfilesAugmentedResponse, AdminProfilesResponse } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

type AdminProfileIdentityListRow = {
  id: string;
  uid: string;
  active: boolean;
  legacy_uid: string;
  email: string;
  name: string;
  given_name: string;
  surname: string;
  provider_count: number;
};

function identityRowToListItem(row: AdminProfileIdentityListRow): AdminProfilesAugmentedResponse & {
  legacy_uid: string;
  provider_count: number;
  email: string;
  name: string;
  can_toggle_active?: boolean;
} {
  return {
    id: row.id,
    collectionId: "",
    collectionName: Collections.AdminProfilesAugmented,
    expand: {},
    uid: row.uid,
    active: row.active,
    work_week_hours: 0,
    salary: false,
    default_charge_out_rate: 0,
    off_rotation_permitted: false,
    skip_min_time_check: AdminProfilesAugmentedSkipMinTimeCheckOptions.no,
    opening_date: "",
    opening_op: 0,
    opening_ov: 0,
    payroll_id: "",
    untracked_time_off: false,
    time_sheet_expected: false,
    allow_personal_reimbursement: false,
    mobile_phone: "",
    job_title: "",
    personal_vehicle_insurance_expiry: "",
    default_branch: "",
    given_name: row.given_name,
    surname: row.surname,
    po_approver_props_id: null,
    po_approver_user_claim_id: null,
    po_approver_max_amount: null,
    po_approver_project_max: null,
    po_approver_sponsorship_max: null,
    po_approver_staff_and_social_max: null,
    po_approver_media_and_event_max: null,
    po_approver_computer_max: null,
    po_approver_divisions: null,
    legacy_uid: row.legacy_uid,
    provider_count: row.provider_count,
    email: row.email,
    name: row.name,
  };
}

type AdminProfileActiveToggleTargetsResponse = {
  admin_profile_ids: string[];
};

type AdminProfileListItem = AdminProfilesAugmentedResponse & {
  legacy_uid?: string;
  provider_count?: number;
  email?: string;
  name?: string;
  can_toggle_active?: boolean;
};

async function loadIdentityItems() {
  const identityItems = (await pb.send("/api/admin_profiles/identity", {
    method: "GET",
  })) as AdminProfileIdentityListRow[];
  return identityItems.map(identityRowToListItem);
}

async function loadActiveToggleTargetIDs(): Promise<Set<string>> {
  try {
    const response = (await pb.send("/api/admin_profiles/active_toggle_targets", {
      method: "GET",
    })) as AdminProfileActiveToggleTargetsResponse;
    return new Set(response.admin_profile_ids);
  } catch {
    return new Set();
  }
}

function withActiveToggleEligibility(
  items: AdminProfileListItem[],
  activeToggleTargetIDs: Set<string>,
): AdminProfileListItem[] {
  return items.map((item) => ({
    ...item,
    can_toggle_active: activeToggleTargetIDs.has(item.id),
  }));
}

export const load: PageLoad = async () => {
  const activeToggleTargetIDs = await loadActiveToggleTargetIDs();

  try {
    const items = await pb
      .collection("admin_profiles_augmented")
      .getFullList<AdminProfilesAugmentedResponse>({ sort: "surname,given_name" });
    if (items.length === 0) {
      try {
        const identityItems = await loadIdentityItems();
        if (identityItems.length > 0) {
          return { items: withActiveToggleEligibility(identityItems, activeToggleTargetIDs) };
        }
      } catch {
        // Non-IT users can legitimately see an empty augmented list; keep it.
      }
    }
    return { items: withActiveToggleEligibility(items, activeToggleTargetIDs) };
  } catch (error) {
    try {
      return { items: withActiveToggleEligibility(await loadIdentityItems(), activeToggleTargetIDs) };
    } catch {
      console.error(`loading admin_profiles list: ${error}`);
      return { items: [] as AdminProfilesResponse[] };
    }
  }
};
