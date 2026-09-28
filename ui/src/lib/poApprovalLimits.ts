export function canViewPOApprovers(claims: string[]): boolean {
  return claims.includes("report") || claims.includes("po_approver");
}

export type POApprovalLimitField =
  | "max_amount"
  | "project_max"
  | "sponsorship_max"
  | "staff_and_social_max"
  | "media_and_event_max"
  | "computer_max";

export type POApprovalLimitsRow = Record<POApprovalLimitField, number> & {
  id: string;
  given_name: string;
  surname: string;
  configured: boolean;
  divisions: string[];
};

export interface POApprovalLimitsResponse {
  items: POApprovalLimitsRow[];
  categories: {
    id: string;
    label: string;
    limit_field: POApprovalLimitField;
    second_approval_threshold: number;
  }[];
  divisions: { id: string; code: string; name: string; active: boolean }[];
}
