import { pb } from "$lib/pocketbase";
import type { PageLoad } from "./$types";

export const load: PageLoad = async () => {
  let dates: string[] = [];

  try {
    const response = await fetch(`${pb.baseUrl}/api/reports/payables_spreadsheet_dates`, {
      headers: pb.authStore.isValid ? { Authorization: pb.authStore.token } : {},
    });
    if (response.ok) {
      dates = await response.json();
    }
  } catch (error) {
    console.error(`loading payables spreadsheet dates: ${error}`);
  }

  return { dates };
};
