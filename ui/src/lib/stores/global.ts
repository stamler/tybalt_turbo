/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type { TimeTypesRecord, DivisionsRecord, JobsRecord } from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { ClientResponseError } from "pocketbase";

const { subscribe, set, update } = writable({
  timetypes: [] as TimeTypesRecord[],
  divisions: [] as DivisionsRecord[],
  jobs: [] as JobsRecord[],
  isLoading: false,
  error: null as ClientResponseError | null,
});

const loadData = async () => {
  update((state) => ({ ...state, isLoading: true, error: null }));
  try {
    const [timetypes, divisions, jobs] = await Promise.all([
      pb.collection("time_types").getFullList<TimeTypesRecord>({ sort: "code", requestKey: "tt" }),
      pb.collection("divisions").getFullList<DivisionsRecord>({ sort: "code", requestKey: "div" }),
      pb.collection("jobs").getFullList<JobsRecord>({ sort: "-number", requestKey: "job" }),
    ]);
    set({ timetypes, divisions, jobs, isLoading: false, error: null });
  } catch (error: unknown) {
    const typedErr = error as ClientResponseError;
    console.error("Error loading data:", typedErr);
    update((state) => ({ ...state, isLoading: false, error: typedErr }));
  }
};

loadData();

export const globalStore = {
  subscribe,
  refresh: loadData,
};
