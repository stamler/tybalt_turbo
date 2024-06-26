/**
 * The global store is used to load data that is used in multiple places in the
 * app.
 */

import type {
  TimeTypesRecord,
  DivisionsRecord,
  JobsRecord,
  ManagersRecord,
} from "$lib/pocketbase-types";
import { writable } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { ClientResponseError } from "pocketbase";

const { subscribe, set, update } = writable({
  timetypes: [] as TimeTypesRecord[],
  divisions: [] as DivisionsRecord[],
  managers: [] as ManagersRecord[],
  jobs: [] as JobsRecord[],
  isLoading: false,
  error: null as ClientResponseError | null,
});

const loadData = async () => {
  update((state) => ({ ...state, isLoading: true, error: null }));
  try {
    const [timetypes, divisions, jobs, managers] = await Promise.all([
      pb.collection("time_types").getFullList<TimeTypesRecord>({ sort: "code", requestKey: "tt" }),
      pb.collection("divisions").getFullList<DivisionsRecord>({ sort: "code", requestKey: "div" }),
      pb.collection("jobs").getFullList<JobsRecord>({ sort: "-number", requestKey: "job" }),
      // managers are all users with a tapr (time approver) claim
      pb.collection("managers").getFullList<ManagersRecord>({ requestKey: "manager" }),
    ]);
    set({ timetypes, divisions, jobs, managers, isLoading: false, error: null });
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
