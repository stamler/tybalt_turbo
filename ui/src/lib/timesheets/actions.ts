export interface TimesheetActionTarget {
  uid?: string | null;
  approver?: string | null;
  approved?: string | null;
  rejected?: string | null;
  committed?: string | null;
  submitted?: boolean | null;
}

function normalized(value?: string | null): string {
  return value ?? "";
}

function isSubmitted(target: TimesheetActionTarget): boolean {
  return target.submitted ?? true;
}

export function canRecallTimesheet(target: TimesheetActionTarget, viewerId: string): boolean {
  return (
    normalized(target.uid) === viewerId &&
    normalized(target.committed) === "" &&
    ((isSubmitted(target) && normalized(target.approved) === "" && normalized(target.rejected) === "") ||
      normalized(target.rejected) !== "")
  );
}

export function canApproveTimesheet(target: TimesheetActionTarget, viewerId: string): boolean {
  return (
    normalized(target.approver) === viewerId &&
    isSubmitted(target) &&
    normalized(target.approved) === "" &&
    normalized(target.rejected) === "" &&
    normalized(target.committed) === ""
  );
}

export function canRejectTimesheet(
  target: TimesheetActionTarget,
  viewerId: string,
  hasCommitAccess: boolean,
): boolean {
  if (!isSubmitted(target) || normalized(target.committed) !== "" || normalized(target.rejected) !== "") {
    return false;
  }

  if (normalized(target.approver) === viewerId) {
    return true;
  }

  return hasCommitAccess && normalized(target.approved) !== "";
}

export function canCommitTimesheet(
  target: TimesheetActionTarget,
  hasCommitAccess: boolean,
): boolean {
  return (
    hasCommitAccess &&
    isSubmitted(target) &&
    normalized(target.approved) !== "" &&
    normalized(target.rejected) === "" &&
    normalized(target.committed) === ""
  );
}
