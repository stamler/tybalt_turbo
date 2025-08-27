import type { PageLoad } from "./$types";
import type { AdminProfilesRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { AdminProfilesPageData } from "$lib/svelte-types";

export const load: PageLoad<AdminProfilesPageData> = async ({ params }) => {
  const defaultItem = {
    uid: "",
    work_week_hours: 40,
    salary: false,
    default_charge_out_rate: 0,
    off_rotation_permitted: false,
    skip_min_time_check: "no",
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
  } as unknown as AdminProfilesRecord;

  try {
    // Load from augmented view for display fields (e.g., names)
    const item = await pb
      .collection("admin_profiles_augmented")
      .getOne<AdminProfilesRecord & { given_name?: string; surname?: string }>(params.id);
    return { item, editing: true, id: params.id };
  } catch (error) {
    console.error(`error loading admin_profile, returning default item: ${error}`);
    return { item: { ...defaultItem }, editing: false, id: null };
  }
};
