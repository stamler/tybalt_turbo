import type { ExpensesRecord } from "$lib/pocketbase-types";
import { ExpensesPaymentTypeOptions } from "$lib/pocketbase-types";
import type { PageLoad } from "./$types";
import type { ExpensesPageData } from "$lib/svelte-types";

export const load: PageLoad<ExpensesPageData> = async () => {
  const defaultItem: Partial<ExpensesRecord> = {
    uid: "",
    // date in YYYY-MM-DD format
    date: new Date().toISOString().split("T")[0],
    division: "",
    description: "",
    payment_type: ExpensesPaymentTypeOptions.OnAccount,
    purchase_order: "",
    vendor: "",
    job: "",
    category: "",
    kind: "",
    allowance_types: [],
    // approver is configured as not required in pocketbase so we do not have to
    // set it here, but is set by the server side hook
  };
  return { item: { ...defaultItem } as ExpensesRecord, editing: false, id: null };
};
