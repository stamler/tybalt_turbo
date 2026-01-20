import type { PageLoad } from "./$types";
import type {
  AdminProfilesAugmentedResponse,
  AdminProfilesRecord,
  DivisionsResponse,
} from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { AdminProfilesPageData } from "$lib/svelte-types";

export const load: PageLoad<AdminProfilesPageData & { divisions: DivisionsResponse[] }> = async ({
  params,
}) => {
  const defaultItem = {
    uid: "",
    active: true,
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
    default_branch: "",
  } as unknown as AdminProfilesRecord;

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
