import type { TimeEntriesResponse } from "$lib/pocketbase-types";

export function calculateTallies(items: TimeEntriesResponse[]) {
  const tallies = {
    workHoursTally: { jobHours: 0, hours: 0, total: 0 },
    nonWorkHoursTally: { total: 0 } as Record<string, number>,
    mealsHoursTally: 0,
    bankEntries: [] as TimeEntriesResponse[],
    payoutRequests: [] as TimeEntriesResponse[],
    offRotationDates: [] as string[],
    offWeek: [] as string[],
  };

  items.forEach((item) => {
    if (!item.expand) {
      alert("Error: expand field is missing from time entry record.");
      return;
    }
    const timeType = item.expand?.time_type.code;

    if (timeType === "R" || timeType === "RT") {
      if (item.job === "") {
        tallies.workHoursTally.hours += item.hours;
      } else {
        tallies.workHoursTally.jobHours += item.hours;
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