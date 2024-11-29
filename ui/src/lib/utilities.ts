import type {
  TimeEntriesResponse,
  TimeSheetsResponse,
  BaseSystemFields,
  IsoDateString,
  CategoriesResponse,
  ClientContactsResponse,
} from "$lib/pocketbase-types";
import { Collections } from "$lib/pocketbase-types";
import { pb } from "$lib/pocketbase";
import flatpickr from "flatpickr";

export interface TimeSheetTally extends BaseSystemFields {
  // These TimeSheetRecord-specific properties will be "" if there is no
  // corresponding time sheet record for the time entries
  week_ending: string;
  salary: boolean;
  work_week_hours: number;
  rejected: IsoDateString;
  rejection_reason: string;
  approved: IsoDateString;

  // These are the tallies for the time entries
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

export function calculateTallies(arg: TimeSheetsResponse | TimeEntriesResponse[]): TimeSheetTally {
  let items: TimeEntriesResponse[];
  let id: string = "";
  let week_ending: string = "";
  let salary: boolean = false;
  let work_week_hours: number = 0;
  let rejected: IsoDateString = "";
  let rejection_reason: string = "";
  let approved: IsoDateString = "";
  let created: IsoDateString = "";
  let updated: IsoDateString = "";
  let collectionId: string = "";
  let collectionName = Collections.TimeSheets;
  if (Array.isArray(arg)) {
    items = arg;
  } else {
    // Existing assignment for TimeSheetsResponse case
    id = arg.id;
    week_ending = arg.week_ending;
    salary = arg.salary;
    work_week_hours = arg.work_week_hours;
    rejected = arg.rejected;
    rejection_reason = arg.rejection_reason;
    created = arg.created;
    updated = arg.updated;
    collectionId = arg.collectionId;
    collectionName = arg.collectionName;
    approved = arg.approved;
    items = (arg.expand as { time_entries_via_tsid: TimeEntriesResponse[] })[
      "time_entries_via_tsid"
    ];
  }
  const tallies: TimeSheetTally = {
    id,
    week_ending,
    salary,
    work_week_hours,
    rejected,
    rejection_reason,
    created,
    updated,
    collectionId,
    collectionName,
    approved,
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

  items.forEach((item) => {
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

export function flatpickrAction(node: HTMLElement, options: flatpickr.Options.Options = {}) {
  const instance = flatpickr(node, {
    minDate: "2024-06-01",
    maxDate: new Date(new Date().setMonth(new Date().getMonth() + 15)),
    enableTime: false,
    dateFormat: "Y-m-d",
    ...options,
  });
  return {
    destroy() {
      instance.destroy();
    },
  };
}

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
