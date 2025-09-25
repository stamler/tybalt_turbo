import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params, url }) => {
  const clientId = params.cid;
  const tab = url.searchParams.get("tab") ?? "projects";
  const projectsPageParam = Number(url.searchParams.get("projectsPage") ?? "1");
  const proposalsPageParam = Number(url.searchParams.get("proposalsPage") ?? "1");
  const ownerPageParam = Number(url.searchParams.get("ownerPage") ?? "1");
  const perPage = 10;

  // fetch core data via API to avoid PocketBase expands
  const client = await pb.send(`/api/clients/${clientId}`, { method: "GET" });

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
