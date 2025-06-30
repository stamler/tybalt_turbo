import type { PageLoad } from "./$types";
import { pb } from "$lib/pocketbase";

export const load: PageLoad = async ({ params, url }) => {
  const clientId = params.cid;
  const tab = url.searchParams.get("tab") ?? "projects";
  const projectsPageParam = Number(url.searchParams.get("projectsPage") ?? "1");
  const proposalsPageParam = Number(url.searchParams.get("proposalsPage") ?? "1");
  const perPage = 10;

  // fetch core data
  const client = await pb.collection("clients").getOne(clientId);
  const contacts = await pb.collection("client_contacts").getFullList({
    filter: `client='${clientId}'`,
    sort: "surname,given_name",
  });

  // Server-side pagination for jobs (projects vs proposals)
  const proposalsFilter = `client='${clientId}' && number ~ 'P%'`;
  const projectsFilter = `client='${clientId}' && number !~ 'P%'`;

  const activePage = tab === "proposals" ? proposalsPageParam : projectsPageParam;
  const activeFilter = tab === "proposals" ? proposalsFilter : projectsFilter;

  // fetch the current page for the active tab
  const activeList = await pb.collection("jobs").getList(activePage, perPage, {
    filter: activeFilter,
    sort: "-number",
  });

  // fetch a minimal list to obtain the total count for the other tab
  const otherFilter = tab === "proposals" ? projectsFilter : proposalsFilter;
  const otherList = await pb.collection("jobs").getList(1, 1, {
    filter: otherFilter,
  });

  const totalPages = Math.max(1, Math.ceil(activeList.totalItems / perPage));

  return {
    client,
    contacts,
    jobs: activeList.items,
    tab,
    page: activePage,
    totalPages,
    projectsPage: tab === "projects" ? activePage : projectsPageParam,
    proposalsPage: tab === "proposals" ? activePage : proposalsPageParam,
    counts: {
      projects: tab === "projects" ? activeList.totalItems : otherList.totalItems,
      proposals: tab === "proposals" ? activeList.totalItems : otherList.totalItems,
    },
  };
};
