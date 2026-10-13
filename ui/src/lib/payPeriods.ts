function parseIsoDate(dateString: string) {
  return new Date(`${dateString}T12:00:00`);
}

function toIsoDate(date: Date) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

// Pay periods are two weeks long and end on a Saturday, so they start on the
// Sunday 13 days earlier.
export function payPeriodStart(payPeriodEnding: string) {
  const date = parseIsoDate(payPeriodEnding);
  date.setDate(date.getDate() - 13);
  return toIsoDate(date);
}

export function isUpcomingPayDate(payDate: string) {
  return payDate > toIsoDate(new Date());
}
