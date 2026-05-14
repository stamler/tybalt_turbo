import type { PurchaseOrdersRecord } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import { blankPurchaseOrder, purchaseOrderFromTemplate } from "$lib/purchaseOrderTemplate";
import type { PageLoad } from "./$types";
import type { PurchaseOrdersPageData } from "$lib/svelte-types";
import { error } from "@sveltejs/kit";

export const load: PageLoad<PurchaseOrdersPageData> = async ({ url }) => {
  const templateId = url.searchParams.get("template");
  let defaultItem = blankPurchaseOrder();

  if (templateId !== null && templateId !== "") {
    try {
      const template = await pb.collection("purchase_orders").getOne(templateId);
      defaultItem = purchaseOrderFromTemplate(template);
    } catch (templateError) {
      console.error(`error loading purchase order template ${templateId}: ${templateError}`);
      throw error(404, "Purchase order template not found");
    }
  }

  return {
    item: { ...defaultItem } as PurchaseOrdersRecord,
    editing: false,
    id: null,
  };
};
