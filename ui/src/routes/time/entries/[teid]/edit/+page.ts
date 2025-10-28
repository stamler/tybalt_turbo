import type { TimeEntriesRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import type { TimeEntriesPageData } from "$lib/svelte-types";

export const load: PageLoad<TimeEntriesPageData> = async ({ params }) => {
  const defaultItem = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    time_type: "sdyfl3q7j7ap849",
    division: "",
    description: "",
    job: "",
    work_record: "",
    hours: 0,
    meals_hours: 0,
    payout_request_amount: 0,
    category: "",
    week_ending: "2006-01-02",
  };
  let item: TimeEntriesRecord;
  try {
    item = await pb.collection("time_entries").getOne(params.teid);
    return { item, editing: true, id: params.teid };
  } catch (error) {
    console.error(`error loading data, returning default item: ${error}`);
    return { item: { ...defaultItem } as TimeEntriesRecord, editing: false, id: null };
  }
};
