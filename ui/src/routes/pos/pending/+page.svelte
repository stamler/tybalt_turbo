<script lang="ts">
  import type { PageData } from "./$types";
  import PurchaseOrdersList from "$lib/components/PurchaseOrdersList.svelte";
  import { fetchVisiblePOs } from "$lib/poVisibility";
  import type {
    PurchaseOrdersAugmentedResponse,
    PurchaseOrdersResponse,
  } from "$lib/pocketbase-types";
  import { pb } from "$lib/pocketbase";
  let { data }: { data: PageData } = $props();

  function shouldRefreshApprovedByMeAwaitingSecond(
    record: PurchaseOrdersResponse,
    currentItems: PurchaseOrdersAugmentedResponse[],
  ): boolean {
    const currentUserId = pb.authStore.record?.id ?? "";
    const isInSectionNow =
      record.status === "Unapproved" &&
      record.rejected === "" &&
      record.approver === currentUserId &&
      record.approved !== "" &&
      record.second_approval === "";

    return isInSectionNow || currentItems.some((item) => item.id === record.id);
  }
</script>

<div class="space-y-8">
  <PurchaseOrdersList inListHeader="Purchase Orders Pending My Approval" data={data.pendingData} />

  <PurchaseOrdersList
    inListHeader="Approved By Me, Awaiting Second Approval"
    data={data.approvedByMeAwaitingSecondData}
    refreshItems={() => fetchVisiblePOs("approved_by_me_awaiting_second", undefined, 20)}
    shouldRefreshOnEvent={shouldRefreshApprovedByMeAwaitingSecond}
    hideWhenEmpty={true}
  />
</div>
