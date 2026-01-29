import { writable, get } from "svelte/store";
import { pb } from "$lib/pocketbase";
import { authStore } from "./auth";
import { globalStore } from "./global";
import MiniSearch from "minisearch";

export interface BusdevLead {
  id: string;
  given_name: string;
  surname: string;
  email: string;
}

interface BusdevLeadsState {
  items: BusdevLead[];
  index: MiniSearch<BusdevLead> | null;
  loading: boolean;
  initialized: boolean;
  error: string | null;
}

const initialState: BusdevLeadsState = {
  items: [],
  index: null,
  loading: false,
  initialized: false,
  error: null,
};

function createBusdevLeadsStore() {
  const { subscribe, set, update } = writable<BusdevLeadsState>(initialState);

  let initPromise: Promise<void> | null = null;

  async function load() {
    const auth = get(authStore);
    if (!auth?.isValid) {
      return;
    }

    update((s) => ({ ...s, loading: true, error: null }));

    try {
      const list = (await pb.send("/api/clients/devleads", { method: "GET" })) as BusdevLead[];

      // Build MiniSearch index if there are enough items
      let index: MiniSearch<BusdevLead> | null = null;
      if (list.length > 10) {
        index = new MiniSearch({
          fields: ["surname", "given_name", "id"],
          storeFields: ["surname", "given_name", "id"],
        });
        index.addAll(list);
      }

      update((s) => ({
        ...s,
        items: list,
        index,
        loading: false,
        initialized: true,
      }));
    } catch (error: unknown) {
      globalStore.addError(`Error loading business development leads: ${String(error)}`);
      update((s) => ({
        ...s,
        loading: false,
        error: String(error),
      }));
    }
  }

  function init() {
    const state = get({ subscribe });
    if (state.initialized || initPromise) {
      return initPromise ?? Promise.resolve();
    }
    initPromise = load();
    return initPromise;
  }

  return {
    subscribe,
    init,
    refresh: load,
  };
}

export const busdevLeads = createBusdevLeadsStore();
