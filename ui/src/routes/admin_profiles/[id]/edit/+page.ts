import type { PageLoad } from "./$types";
import {
  AdminProfilesAugmentedSkipMinTimeCheckOptions,
  Collections,
} from "$lib/pocketbase-types";
import type {
  AdminProfilesAugmentedResponse,
  DivisionsResponse,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { AdminProfilesEditPageData } from "$lib/svelte-types";

export const load: PageLoad<AdminProfilesEditPageData & { divisions: DivisionsResponse[] }> = async ({
  params,
}) => {
  const defaultItem = {
    id: "",
    collectionId: "",
    collectionName: Collections.AdminProfilesAugmented,
    expand: {},
    uid: "",
    active: true,
    work_week_hours: 40,
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
    given_name: "",
    surname: "",
    po_approver_props_id: null,
    po_approver_max_amount: null,
    po_approver_project_max: null,
    po_approver_sponsorship_max: null,
    po_approver_staff_and_social_max: null,
    po_approver_media_and_event_max: null,
    po_approver_computer_max: null,
    po_approver_divisions: null,
    default_branch: "",
  } satisfies AdminProfilesAugmentedResponse;

  try {
    const [item, divisions] = await Promise.all([
      pb.collection("admin_profiles_augmented").getOne<AdminProfilesAugmentedResponse>(params.id),
      pb.collection("divisions").getFullList<DivisionsResponse>({ sort: "code" }),
    ]);

    return { item, editing: true, id: params.id, divisions };
  } catch (error) {
    console.error(`error loading admin_profile, returning default item: ${error}`);
    return { item: { ...defaultItem }, editing: false, id: null, divisions: [] };
  }
};
