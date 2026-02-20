import type {
  TimeEntriesResponse,
  CategoriesResponse,
  ClientContactsResponse,
  BaseSystemFields,
  PurchaseOrdersAugmentedResponse,
  ProfilesResponse,
} from "$lib/pocketbase-types";
import { type UnsubscribeFunc } from "pocketbase";
import { pb } from "$lib/pocketbase";
import { get } from "svelte/store";
import { globalStore } from "$lib/stores/global";

export const DATE_INPUT_MIN = "2024-06-01";

function toIsoDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function dateInputMaxMonthsAhead(monthsAhead: number): string {
  const today = new Date();
  const target = new Date(today);
  target.setMonth(target.getMonth() + monthsAhead);
  return toIsoDate(target);
}

export interface TimeSheetTallyQueryRow {
  id: string;
  approved: string;
  bank_entry_dates: string[];
  division_names: string[];
  divisions: string[];
  job_numbers: string[];
  meals_hours: number;
  non_work_total_hours: number;
  ob_hours: number;
  off_rotation_dates: string[];
  off_week_dates: string[];
  op_hours: number;
  os_hours: number;
  ov_hours: number;
  rb_hours: number;
  payout_request_amount: number;
  payout_request_dates: string[];
  rejected: string;
  rejection_reason: string;
  salary: string;
  time_type_names: string[];
  time_types: string[];
  week_ending: string;
  work_hours: number;
  work_job_hours: number;
  work_total_hours: number;
  work_week_hours: number;
  given_name: string;
  surname: string;
  approver: string;
  committer: string;
  committed: string;
  approver_name: string;
  committer_name: string;
}

export interface TimeEntriesSummary {
  workHoursTally: {
    jobHours: number;
    hours: number;
    total: number;
  };
  nonWorkHoursTally: {
    total: number;
    [key: string]: number;
  };
  mealsHoursTally: number;
  bankEntries: TimeEntriesResponse[];
  payoutRequests: TimeEntriesResponse[];
  offRotationDates: string[];
  offWeek: string[];

  // these are new and in the firestore system they existed as properties of the
  // TimeSheets documents but are now denormalized and derived from the records
  // in time_entries
  jobsTally: {
    [key: string]: number;
  };
  divisionsTally: {
    [key: string]: string;
  };
}

export function calculateTally(entries: TimeEntriesResponse[]): TimeEntriesSummary {
  const tallies: TimeEntriesSummary = {
    workHoursTally: { jobHours: 0, hours: 0, total: 0 },
    nonWorkHoursTally: { total: 0 },
    mealsHoursTally: 0,
    bankEntries: [],
    payoutRequests: [],
    offRotationDates: [],
    offWeek: [],
    jobsTally: {},
    divisionsTally: {},
  };

  entries.forEach((item) => {
    if (!item.expand) {
      alert("Error: expand field is missing from time entry record.");
      return;
    }
    const timeType = item.expand?.time_type.code;

    if (timeType === "R" || timeType === "RT") {
      if (item.expand.division) {
        tallies.divisionsTally[item.expand.division.code] = item.expand.division.name;
      }
      if (item.job === "") {
        tallies.workHoursTally.hours += item.hours;
      } else {
        tallies.workHoursTally.jobHours += item.hours;
        // tally jobs
        tallies.jobsTally[item.expand.job.number] =
          (tallies.jobsTally[item.expand.job.number] || 0) + item.hours;
      }
      tallies.workHoursTally.total += item.hours;
      tallies.mealsHoursTally += item.meals_hours;
    } else if (timeType === "OR") {
      tallies.offRotationDates.push(item.date);
    } else if (timeType === "OW") {
      tallies.offWeek.push(item.date);
    } else if (timeType === "OTO") {
      tallies.payoutRequests.push(item);
    } else if (timeType === "RB") {
      tallies.bankEntries.push(item);
    } else {
      tallies.nonWorkHoursTally[timeType] = (tallies.nonWorkHoursTally[timeType] || 0) + item.hours;
      tallies.nonWorkHoursTally.total += item.hours;
    }
  });

  return tallies;
}

export function shortDate(dateString: string, includeYear = false) {
  const months = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];
  const dateParts = dateString.split("-");
  // const year = dateParts[0];
  const month = months[parseInt(dateParts[1], 10) - 1];
  const day = parseInt(dateParts[2], 10);
  return `${month} ${day}${includeYear ? `, ${dateParts[0]}` : ""}`;
}

export function formatDateTime(value: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const datePart = date.toLocaleDateString("en-US", {
    month: "short",
    day: "2-digit",
    year: "numeric",
  });
  const timePart = date.toLocaleTimeString();
  return `${datePart} ${timePart}`;
}
/*
export const hoursWorked = function (item: TimeSheetTally) {
  let workedHours = 0;
  workedHours += item.workHoursTally.hours;
  workedHours += item.workHoursTally.jobHours;
  if (workedHours > 0) {
    return `${workedHours} hours worked`;
  } else {
    return "no work";
  }
};

export const jobs = function (item: TimeSheetTally) {
  const jobs = Object.keys(item.jobsTally).sort().join(", ");
  if (jobs.length > 0) {
    return `jobs: ${jobs}`;
  } else {
    return;
  }
};

export const divisions = function (item: TimeSheetTally) {
  const divisions = Object.keys(item.divisionsTally).sort().join(", ");
  if (divisions.length > 0) {
    return `divisions: ${divisions}`;
  } else {
    return;
  }
};

export const payoutRequests = function (item: TimeSheetTally) {
  const totalPayoutRequests = item.payoutRequests.reduce(
    (sum, request) => sum + request.payout_request_amount,
    0,
  );
  if (totalPayoutRequests > 0) {
    return `$${totalPayoutRequests.toFixed(2)} in payout requests`;
  } else {
    return "no payout requests";
  }
};
*/
const nFormatter = function (num: number, digits: number) {
  const lookup = [
    { value: 1, symbol: "" },
    { value: 1e3, symbol: "k" },
    { value: 1e6, symbol: "M" },
    { value: 1e9, symbol: "G" },
    { value: 1e12, symbol: "T" },
    { value: 1e15, symbol: "P" },
    { value: 1e18, symbol: "E" },
  ];
  const regexp = /\.0+$|(?<=\.[0-9]*[1-9])0+$/;
  const item = lookup
    .slice()
    .reverse()
    .find((item) => num >= item.value);
  return item ? (num / item.value).toFixed(digits).replace(regexp, "").concat(item.symbol) : "0";
};

export const formatDollars = function <T>(value: T) {
  if (typeof value !== "number") {
    return value;
  }
  // add thousands separator and round to 2 decimal places
  // return "$" + value.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return "$" + nFormatter(value, 1);
};

export const formatCurrency = (value: number | null | undefined) => {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return "$0.00";
  }
  return new Intl.NumberFormat("en-CA", {
    style: "currency",
    currency: "CAD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
};

export const formatPercent = function <T>(value: T) {
  if (typeof value !== "number") {
    return value;
  }
  return value.toFixed(2) + "%";
};

export const formatNumber = function <T>(value: T) {
  if (typeof value !== "number") {
    return value;
  }
  return nFormatter(value, 1);
};

// Fetch categories for the given job
export async function fetchCategories(jobId: string): Promise<CategoriesResponse[]> {
  // if jobId is an empty string, return an empty array
  if (jobId === "") {
    return Promise.resolve([] as CategoriesResponse[]);
  }

  try {
    return pb.collection("categories").getFullList({
      filter: `job="${jobId}"`,
      sort: "name",
    });
  } catch (error) {
    console.error("Error fetching categories:", error);
    return Promise.resolve([] as CategoriesResponse[]);
  }
}

// Build a small synchronizer for "selected job -> category options" form state.
//
// Why this exists:
// Multiple editors (PO, Expense, Time Entry, Time Amendment) had nearly identical
// reactive effects that called `fetchCategories(item.job)` and assigned the result.
// That duplication made behavior easy to drift and hard to update safely.
//
// What this does:
// - Returns a function that accepts the current job id.
// - Starts a categories fetch when a truthy job id is provided.
// - Ignores stale responses by using a monotonic request id, so rapid job changes
//   cannot apply out-of-order data.
// - Does not clear categories when job id is empty; callers keep existing behavior
//   and decide when/how category state is reset on save.
//
// Usage pattern:
// `const syncCategoriesForJob = createJobCategoriesSync((rows) => (categories = rows));`
// then inside an effect:
// `$effect(() => { syncCategoriesForJob(item.job); });`
export function createJobCategoriesSync(
  applyCategories: (rows: CategoriesResponse[]) => void,
): (jobId: string | undefined | null) => void {
  let requestId = 0;
  return (jobId: string | undefined | null) => {
    const currentRequestId = ++requestId;
    if (!jobId) return;
    fetchCategories(jobId).then((rows) => {
      if (currentRequestId !== requestId) return;
      applyCategories(rows);
    });
  };
}

// Fetch associated client_contacts
export async function fetchClientContacts(clientId: string): Promise<ClientContactsResponse[]> {
  // if clientId is an empty string, return an empty array
  if (clientId === "") {
    return Promise.resolve([] as ClientContactsResponse[]);
  }

  try {
    return pb.collection("client_contacts").getFullList({
      filter: `client="${clientId}"`,
      sort: "surname,given_name",
    });
  } catch (error) {
    console.error("Error fetching client_contacts:", error);
    return Promise.resolve([] as ClientContactsResponse[]);
  }
}

function csvEscape(value: unknown): string {
  if (value === null || value === undefined) return "";
  return `"${String(value).replace(/"/g, '""')}"`;
}

function rowsToCsv(rows: Array<Record<string, unknown>>, headers: string[]): string {
  const lines = [headers.map((h) => csvEscape(h)).join(",")];
  for (const row of rows) {
    lines.push(headers.map((h) => csvEscape(row[h])).join(","));
  }
  return lines.join("\n");
}

export function downloadCsvRows(
  fileName: string,
  rows: Array<Record<string, unknown>>,
  headers?: string[],
) {
  if (!rows || rows.length === 0) return;
  const resolvedHeaders = headers && headers.length > 0 ? headers : Object.keys(rows[0] ?? {});
  if (resolvedHeaders.length === 0) return;

  const csv = rowsToCsv(rows, resolvedHeaders);
  const blob = new Blob([csv], { type: "text/csv" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  document.body.removeChild(anchor);
  URL.revokeObjectURL(url);
}

export async function downloadCSV(endpoint: string, fileName: string) {
  try {
    // Prepare headers, including Authorization if the user is logged in
    const headers: HeadersInit = {};
    if (pb.authStore.isValid) {
      headers["Authorization"] = pb.authStore.token;
    }

    // Use standard fetch API
    const response = await fetch(endpoint, {
      method: "GET",
      headers: headers,
    });

    if (!response.ok) {
      throw new Error(`HTTP error ${response.status}: ${await response.text()}`);
    }

    // Get the response body as text
    const csvString = await response.text();

    if (typeof csvString !== "string") {
      throw new Error("Received non-string response from server.");
    }

    // Create a Blob from the string response
    const timeReportCSV = new Blob([csvString], { type: "text/csv" });

    // Create an object URL from the Blob
    const blobUrl = URL.createObjectURL(timeReportCSV);

    // Create a temporary anchor element to trigger the download
    const anchor = document.createElement("a");
    anchor.href = blobUrl;
    // Suggest a filename (the browser might override it, but it's good practice)
    // We can construct a dynamic filename based on the inputs
    anchor.download = fileName;
    document.body.appendChild(anchor); // Append to body to make it clickable
    anchor.click(); // Programmatically click the anchor to trigger download
    document.body.removeChild(anchor); // Clean up the anchor element

    // Revoke the object URL to free up memory, potentially after a small delay
    setTimeout(() => URL.revokeObjectURL(blobUrl), 100);
  } catch (error) {
    console.error("Error fetching CSV:", error);
    // Optionally: Show an error message to the user
  }
}

export async function downloadZip(endpoint: string, fileName: string) {
  try {
    // Prepare headers, including Authorization if the user is logged in
    const headers: HeadersInit = {};
    if (pb.authStore.isValid) {
      headers["Authorization"] = pb.authStore.token;
    }

    // First fetch: get the URL of the zip file
    const urlResponse = await fetch(endpoint, {
      method: "GET",
      headers: headers,
    });

    if (!urlResponse.ok) {
      throw new Error(
        `HTTP error ${urlResponse.status} while fetching file URL: ${await urlResponse.text()}`,
      );
    }

    const responseData = await urlResponse.json();
    const fileUrl = responseData.url;

    if (typeof fileUrl !== "string") {
      throw new Error("Invalid file URL received from server.");
    }

    // Second fetch: get the actual file blob from the retrieved URL
    // Note: Depending on the source of fileUrl, it might or might not need auth headers.
    // For simplicity, not adding auth headers here. If needed, they can be added.
    const fileResponse = await fetch(`${pb.baseUrl}/api/files/${fileUrl}`);

    if (!fileResponse.ok) {
      throw new Error(
        `HTTP error ${fileResponse.status} while downloading file: ${await fileResponse.text()}`,
      );
    }

    // Get the response body as a blob
    const zipBlob = await fileResponse.blob();

    // Create an object URL from the Blob
    const blobUrl = URL.createObjectURL(zipBlob);

    // Create a temporary anchor element to trigger the download
    const anchor = document.createElement("a");
    anchor.href = blobUrl;
    // Suggest a filename
    anchor.download = fileName;
    document.body.appendChild(anchor); // Append to body to make it clickable
    anchor.click(); // Programmatically click the anchor to trigger download
    document.body.removeChild(anchor); // Clean up the anchor element

    // Revoke the object URL to free up memory, potentially after a small delay
    setTimeout(() => URL.revokeObjectURL(blobUrl), 100);
  } catch (error) {
    console.error("Error fetching zip:", error);
    throw error; // Re-throw the error so it can be caught by the caller
  }
}

// --- Defaults helpers ---
const appliedDivisionOnce = new WeakSet<object>();
const appliedRoleOnce = new WeakSet<object>();
const lastUidAppliedForItem = new WeakMap<object, string>();
export function applyDefaultDivisionOnce(
  item: { division?: string } | null | undefined,
  editing: boolean,
  uid?: string,
) {
  if (!item || editing) return;

  // When a subject uid is provided (e.g., Time Amendments), allow re-applying
  // for different users, but avoid duplicate fetches for the same uid.
  if (uid && uid !== "") {
    if (item.division && item.division !== "") return;
    const lastUid = lastUidAppliedForItem.get(item as object);
    if (lastUid === uid) return;
    lastUidAppliedForItem.set(item as object, uid);
    (async () => {
      try {
        const prof = await pb
          .collection("profiles")
          .getFirstListItem<ProfilesResponse>(pb.filter("uid={:uid}", { uid }));
        const dd = prof?.default_division ?? "";
        if ((!item.division || item.division === "") && dd) {
          item.division = dd as string;
        }
      } catch {
        // noop
      }
    })();
    return;
  }

  // Caller default division: apply only once per item instance
  if (appliedDivisionOnce.has(item as object)) return;
  const dd = get(globalStore)?.profile?.default_division ?? "";
  if ((!item.division || item.division === "") && dd) {
    item.division = dd as string;
    appliedDivisionOnce.add(item as object);
  }
}

export function applyDefaultRoleOnce(item: { role?: string } | null | undefined, editing: boolean) {
  if (!item || editing) return;

  // Apply default role only once per item instance
  if (appliedRoleOnce.has(item as object)) return;
  const dr = get(globalStore)?.profile?.default_role ?? "";
  if ((!item.role || item.role === "") && dr) {
    item.role = dr as string;
    appliedRoleOnce.add(item as object);
  }
}

// Like previously existing augmentedProxySubscription but uses a loader callback instead of a view name,
// so callers can source augmented rows from a custom API endpoint.
export function proxySubscriptionWithLoader<
  CollectionResponse extends BaseSystemFields,
  ViewResponse extends { id: string },
>(
  localArray: ViewResponse[],
  collectionName: string,
  loadAugmented: (id: string) => Promise<ViewResponse>,
  updateCallback: (newArray: ViewResponse[]) => void,
  createdItemIsVisible: undefined | ((record: CollectionResponse) => boolean) = undefined,
): Promise<UnsubscribeFunc> {
  const isNotFoundError = (error: any): boolean =>
    Boolean(error?.status === 404 || error?.response?.status === 404);

  return pb.collection(collectionName).subscribe<CollectionResponse>("*", async (e) => {
    if (!Array.isArray(localArray)) return;
    const id = e.record.id;
    let augmentedRecord: ViewResponse;
    switch (e.action) {
      case "create":
        if (createdItemIsVisible !== undefined && !createdItemIsVisible(e.record)) {
          return;
        }
        try {
          augmentedRecord = await loadAugmented(id);
        } catch (error) {
          if (isNotFoundError(error)) {
            return;
          }
          console.error("Error loading augmented record on create:", error);
          return;
        }
        localArray = [augmentedRecord, ...localArray];
        break;
      case "update":
        // If the updated record no longer matches the page's visibility predicate,
        // remove it from the local array instead of keeping it.
        if (createdItemIsVisible !== undefined && !createdItemIsVisible(e.record)) {
          localArray = localArray.filter((item) => item.id !== e.record.id);
          break;
        }
        try {
          augmentedRecord = await loadAugmented(id);
        } catch (error) {
          if (isNotFoundError(error)) {
            localArray = localArray.filter((item) => item.id !== e.record.id);
            break;
          }
          console.error("Error loading augmented record on update:", error);
          break;
        }
        localArray = localArray.map((item) => (item.id === e.record.id ? augmentedRecord : item));
        break;
      case "delete":
        localArray = localArray.filter((item) => item.id !== e.record.id);
        break;
    }
    updateCallback(localArray);
  });
}

export function poActiveDate(record: PurchaseOrdersAugmentedResponse): string {
  // return the first 10 characters the second_approval (if not undefined) or of
  // the approved.
  if (record.second_approval) {
    return record.second_approval.substring(0, 10);
  }
  return record.approved.substring(0, 10);
}

// Get the appropriate redirect URL after absorb operations
export async function getAbsorbRedirectUrl(
  collectionName: string,
  targetRecordId?: string,
  clientId?: string,
): Promise<string> {
  if (collectionName === "client_contacts") {
    // For client_contacts, use provided clientId or fetch from contact record
    let finalClientId = clientId;
    if (!finalClientId && targetRecordId) {
      const contact = await pb.collection("client_contacts").getOne(targetRecordId);
      finalClientId = contact.client;
    }
    return `/clients/${finalClientId}/edit`;
  } else {
    // For other collections, redirect to the list page
    return `/${collectionName}/list`;
  }
}
