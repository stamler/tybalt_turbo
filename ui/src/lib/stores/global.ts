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
import MiniSearch from 'minisearch'

const { subscribe, update } = writable({
  lastRefresh: new Date(),
  maxAge: 5 * 60 * 1000, // 5 minutes
  timetypes: [] as TimeTypesRecord[],
  divisions: [] as DivisionsRecord[],
  managers: [] as ManagersRecord[],
  jobs: [] as JobsRecord[],
  jobsIndex: null as MiniSearch<JobsRecord> | null,
  isLoading: false,
  error: null as ClientResponseError | null,
});

const loadData = async (lastRefresh: Date) => {
  update((state) => ({ ...state, isLoading: true, error: null }));
  try {
    const [timetypes, divisions, jobs, managers] = await Promise.all([
      pb.collection("time_types").getFullList<TimeTypesRecord>({ sort: "code", requestKey: "tt" }),
      pb.collection("divisions").getFullList<DivisionsRecord>({ sort: "code", requestKey: "div" }),
      pb.collection("jobs").getFullList<JobsRecord>({ sort: "-number", requestKey: "job" }),
      // managers are all users with a tapr (time approver) claim
      pb.collection("managers").getFullList<ManagersRecord>({ requestKey: "manager" }),
    ]);
    // populate the minisearch index for jobs
    const jobsIndex = new MiniSearch<JobsRecord>({
      // id must be indexed because we're using search in the DSAutoComplete
      // component to get an existing job by id and display it in the ui.
      fields: ['id', 'number', 'description'],
      storeFields: ['id', 'number', 'description'],
    });
    jobsIndex.addAll(jobs);

    // update the store
    update((state) => ({
      ...state,
      timetypes,
      divisions,
      jobs,
      jobsIndex,
      managers,
      lastRefresh,
      isLoading: false,
      error: null,
    }));
  } catch (error: unknown) {
    const typedErr = error as ClientResponseError;
    console.error("Error loading data:", typedErr);
    update((state) => ({ ...state, isLoading: false, error: typedErr }));
  }
};

const refresh = async (immediate = false) => {
  // if immediate is true, we will refresh even if the data is not stale
  update((state) => {
    const now = new Date();
    if (!immediate && now.getTime() - state.lastRefresh.getTime() < state.maxAge) {
      return state;
    }
    loadData(now);
    return { ...state, lastRefresh: now };
  });
};
refresh(true);

export const globalStore = {
  subscribe,
  refresh,
};
