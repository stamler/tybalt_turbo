import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";
import type { JobsRecord } from "$lib/pocketbase-types";

export type ClientNote = {
  id: string;
  created: string;
  note: string;
  job: null | {
    id: string;
    number: string;
    description: string;
  };
  author: {
    id: string;
    email: string;
    given_name: string;
    surname: string;
  };
};

export const load: PageLoad = async ({ params, url }) => {
  const clientId = params.cid;
  const tab = url.searchParams.get("tab") ?? "projects";
  const projectsPageParam = Number(url.searchParams.get("projectsPage") ?? "1");
  const proposalsPageParam = Number(url.searchParams.get("proposalsPage") ?? "1");
  const ownerPageParam = Number(url.searchParams.get("ownerPage") ?? "1");
  const perPage = 10;

  // fetch core data via API to avoid PocketBase expands
  const client = await pb.send(`/api/clients/${clientId}`, { method: "GET" });

  // Load client notes with author/job expands
  const notes = (await pb.send(`/api/clients/${clientId}/notes`, {
    method: "GET",
  })) as ClientNote[];

  const noteJobs = await pb.collection("jobs").getFullList<JobsRecord>(200, {
    filter: `client="${clientId}" || job_owner="${clientId}"`,
    sort: "-created",
    fields: "id,number,description",
  });

  // Server-side pagination for jobs (projects vs proposals)
  const proposalsFilter = `client='${clientId}' && number ~ 'P%'`;
  const projectsFilter = `client='${clientId}' && number !~ 'P%'`;
  const ownerFilter = `job_owner='${clientId}'`;

  const activePage =
    tab === "proposals" ? proposalsPageParam : tab === "owner" ? ownerPageParam : projectsPageParam;
  const activeFilter =
    tab === "proposals" ? proposalsFilter : tab === "owner" ? ownerFilter : projectsFilter;

  // fetch the current page for the active tab
  const activeList = await pb.collection("jobs").getList(activePage, perPage, {
    filter: activeFilter,
    sort: "-number",
  });

  // fetch a minimal list to obtain the total count for the other tab
  // fetch minimal lists to obtain total counts for the other tabs
  const otherProjectsList = await pb.collection("jobs").getList(1, 1, {
    filter: projectsFilter,
  });
  const otherProposalsList = await pb.collection("jobs").getList(1, 1, {
    filter: proposalsFilter,
  });
  const otherOwnerList = await pb.collection("jobs").getList(1, 1, {
    filter: ownerFilter,
  });

  const totalPages = Math.max(1, Math.ceil(activeList.totalItems / perPage));

  return {
    client,
    referencingJobsCount: client.referencing_jobs_count,
    jobs: activeList.items,
    notes,
    noteJobs,
    tab,
    page: activePage,
    totalPages,
    projectsPage: tab === "projects" ? activePage : projectsPageParam,
    proposalsPage: tab === "proposals" ? activePage : proposalsPageParam,
    ownerPage: tab === "owner" ? activePage : ownerPageParam,
    counts: {
      projects: tab === "projects" ? activeList.totalItems : otherProjectsList.totalItems,
      proposals: tab === "proposals" ? activeList.totalItems : otherProposalsList.totalItems,
      owner: tab === "owner" ? activeList.totalItems : otherOwnerList.totalItems,
    },
  };
};
