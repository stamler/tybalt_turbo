import { error } from "@sveltejs/kit";
import { ClientResponseError } from "pocketbase";
import { pb } from "$lib/pocketbase";
import type { JobsRecord } from "$lib/pocketbase-types";
import type { ClientNote } from "$lib/types/notes";
import { globalStore } from "$lib/stores/global";
import {
  parseClientWorkspaceQuery,
  positivePage,
  type WorkspaceClient,
  type ClientWorkspaceTab,
} from "$lib/clientWorkspace";

export async function loadClientWorkspace(
  { params, url }: { params: { cid: string }; url: URL },
  tab: ClientWorkspaceTab,
) {
  const clientId = params.cid;
  const { view } = parseClientWorkspaceQuery(url.searchParams);
  await globalStore.ensureClaims();
  // Route loads and hover preloads can read the same endpoint at once.
  // SvelteKit discards stale results; SDK cancellation must not cancel another load.
  let client: WorkspaceClient;
  try {
    client = await pb.send<WorkspaceClient>(`/api/clients/${clientId}`, {
      method: "GET",
      requestKey: null,
    });
  } catch (failure) {
    if (failure instanceof ClientResponseError && failure.status === 404)
      error(404, "Client not found.");
    throw failure;
  }
  const base = { client, tab, view };
  // Load only what the open tab shows, so switching tabs stays quick.
  if (tab === "jobs") return { ...base, ...(await loadJobLists(clientId, view, url)), ...noNotes };
  if (tab === "notes") return { ...base, ...noJobs, ...(await loadNotes(clientId)) };
  return { ...base, ...noJobs, ...noNotes };
}

const perPage = 20;
const noJobs = {
  jobs: [] as JobsRecord[],
  page: 1,
  totalPages: 1,
  counts: { projects: 0, proposals: 0, owner: 0 },
};
const noNotes = { notes: [] as ClientNote[], noteJobs: [] as JobsRecord[] };

async function loadNotes(clientId: string) {
  const [notes, noteJobs] = await Promise.all([
    pb.send<ClientNote[]>(`/api/clients/${clientId}/notes`, { method: "GET", requestKey: null }),
    pb.collection("jobs").getFullList<JobsRecord>({
      filter: pb.filter("client={:client} || job_owner={:client}", { client: clientId }),
      sort: "-created",
      fields: "id,number,description",
      requestKey: null,
    }),
  ]);
  return { notes, noteJobs };
}

async function loadJobLists(clientId: string, view: "projects" | "proposals" | "owner", url: URL) {
  const pages = {
    projects: positivePage(url.searchParams.get("projectsPage")),
    proposals: positivePage(url.searchParams.get("proposalsPage")),
    owner: positivePage(url.searchParams.get("ownerPage")),
  };
  const filters = {
    proposals: pb.filter("client={:client} && number ~ 'P%'", { client: clientId }),
    projects: pb.filter("client={:client} && number !~ 'P%'", { client: clientId }),
    owner: pb.filter("job_owner={:client}", { client: clientId }),
  };
  const [projects, proposals, owner] = await Promise.all(
    (["projects", "proposals", "owner"] as const).map((key) =>
      pb
        .collection("jobs")
        .getList<JobsRecord>(key === view ? pages[key] : 1, key === view ? perPage : 1, {
          filter: filters[key],
          sort: "-number",
          requestKey: null,
        }),
    ),
  );
  const lists = { projects, proposals, owner };
  let active = lists[view];
  const totalPages = Math.max(1, Math.ceil(active.totalItems / perPage));
  // A removed job or stale bookmark can leave the requested page beyond the last page.
  if (pages[view] > totalPages) {
    pages[view] = totalPages;
    active = await pb.collection("jobs").getList<JobsRecord>(totalPages, perPage, {
      filter: filters[view],
      sort: "-number",
      requestKey: null,
    });
  }
  return {
    jobs: active.items,
    page: pages[view],
    totalPages,
    counts: {
      projects: projects.totalItems,
      proposals: proposals.totalItems,
      owner: owner.totalItems,
    },
  };
}

export type ClientWorkspaceData = Awaited<ReturnType<typeof loadClientWorkspace>>;
