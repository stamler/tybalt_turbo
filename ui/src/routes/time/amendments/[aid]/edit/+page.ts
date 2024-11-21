import type { TimeAmendmentsRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { TimeAmendmentsPageData } from "$lib/svelte-types";

export const load: PageLoad<TimeAmendmentsPageData> = async ({ params }) => {
  const defaultItem = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    time_type: "sdyfl3q7j7ap849",
    division: "vccd5fo56ctbigh",
    description: "",
    job: "",
    work_record: "",
    hours: 0,
    meals_hours: 0,
    payout_request_amount: 0,
    category: "",
    week_ending: "2006-01-02",
  };
  let item: TimeAmendmentsRecord;

  try {
    item = await pb.collection("time_amendments").getOne(params.aid);
    return { item, editing: true, id: params.aid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as TimeAmendmentsRecord, editing: false, id: null };
  }
};
