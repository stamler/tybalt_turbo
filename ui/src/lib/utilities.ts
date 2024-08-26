import type { TimeEntriesResponse, TimeSheetsResponse, BaseSystemFields, IsoDateString } from "$lib/pocketbase-types";
import { Collections } from "$lib/pocketbase-types";

export interface TimeSheetTally extends BaseSystemFields {
  // These TimeSheetRecord-specific properties will be "" if there is no
  // corresponding time sheet record for the time entries
  week_ending: string;
  salary: boolean;
  work_week_hours: number;

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
    created = arg.created;
    updated = arg.updated;
    collectionId = arg.collectionId;
    collectionName = arg.collectionName;
    items = (arg.expand as { "time_entries(tsid)": TimeEntriesResponse[]})["time_entries(tsid)"]
  }
  const tallies: TimeSheetTally = {
    id,
    week_ending,
    salary,
    work_week_hours,
    created,
    updated,
    collectionId,
    collectionName,
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
        // tally jobs. TODO: include other details about the job besides the
        // number of hours
        tallies.jobsTally[item.expand.job.number] = (tallies.jobsTally[item.expand.job.number] || 0) + item.hours;
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
      tallies.nonWorkHoursTally[timeType] =
        (tallies.nonWorkHoursTally[timeType] || 0) + item.hours;
      tallies.nonWorkHoursTally.total += item.hours;
    }
  });

  return tallies;
}

export function shortDate(dateString: string) {
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  const dateParts = dateString.split("-");
  // const year = dateParts[0];
  const month = months[parseInt(dateParts[1], 10) - 1];
  const day = parseInt(dateParts[2], 10);
  return `${month} ${day}`;
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

export const hoursOff = function (item: TimeSheetTally) {
  let hoursOff = 0;
  for (const timetype in item.nonWorkHoursTally) {
    hoursOff += item.nonWorkHoursTally[timetype];
  }
  if (hoursOff > 0) {
    return `${hoursOff} hours off`;
  } else {
    return "no time off";
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
