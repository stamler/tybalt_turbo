import type { TimeAmendmentsRecord } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { TimeAmendmentsPageData } from "$lib/svelte-types";

export const load: PageLoad<TimeAmendmentsPageData> = async () => {
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
  return { item: { ...defaultItem } as TimeAmendmentsRecord, editing: false, id: null };
};
