import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";
import { error } from "@sveltejs/kit";

export const load: PageLoad = async ({ params, url }) => {
  const vendorId = params.vid;
  const tab = url.searchParams.get("tab") ?? "purchase_orders";
  const poPageParam = Number(url.searchParams.get("poPage") ?? "1");
  const expPageParam = Number(url.searchParams.get("expPage") ?? "1");
  const perPage = 10;

  try {
    // fetch vendor core data
    const vendor = await pb.collection("vendors").getOne(vendorId);

    // filters
    const poFilter = `vendor='${vendorId}' && (status='Active' || status='Closed')`;
    const expFilter = `vendor='${vendorId}' && committed != ''`;

    // decide which collection & filter to run for active tab
    const isExpensesTab = tab === "expenses";
    const activePage = isExpensesTab ? expPageParam : poPageParam;
    const activeFilter = isExpensesTab ? expFilter : poFilter;
    const activeCollection = isExpensesTab ? "expenses" : "purchase_orders";

    const activeList = await pb.collection(activeCollection).getList(activePage, perPage, {
      filter: activeFilter,
      sort: "-date",
    });

    // Fetch total count for the other tab with minimal query
    const otherCollection = isExpensesTab ? "purchase_orders" : "expenses";
    const otherFilter = isExpensesTab ? poFilter : expFilter;
    const otherList = await pb.collection(otherCollection).getList(1, 1, {
      filter: otherFilter,
    });

    const totalPages = Math.max(1, Math.ceil(activeList.totalItems / perPage));

    return {
      vendor,
      tab,
      page: activePage,
      totalPages,
      poPage: isExpensesTab ? poPageParam : activePage,
      expPage: isExpensesTab ? activePage : expPageParam,
      purchaseOrders: isExpensesTab ? [] : activeList.items,
      expenses: isExpensesTab ? activeList.items : [],
      counts: {
        purchase_orders: isExpensesTab ? otherList.totalItems : activeList.totalItems,
        expenses: isExpensesTab ? activeList.totalItems : otherList.totalItems,
      },
    };
  } catch (err) {
    console.error(`loading vendor details: ${err}`);
    throw error(404, `Failed to load vendor details: ${err}`);
  }
};
