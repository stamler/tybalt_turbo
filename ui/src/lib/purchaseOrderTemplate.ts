import type { PurchaseOrdersRecord, PurchaseOrdersResponse } from "$lib/pocketbase-types";
import {
  PurchaseOrdersFrequencyOptions,
  PurchaseOrdersPaymentTypeOptions,
  PurchaseOrdersStatusOptions,
  PurchaseOrdersTypeOptions,
} from "$lib/pocketbase-types";

type PurchaseOrderTemplateSource = Partial<PurchaseOrdersRecord | PurchaseOrdersResponse>;

export function blankPurchaseOrder(date = new Date()): Partial<PurchaseOrdersRecord> {
  return {
    po_number: "",
    status: PurchaseOrdersStatusOptions.Unapproved,
    uid: "",
    type: PurchaseOrdersTypeOptions["One-Time"],
    date: date.toISOString().split("T")[0],
    end_date: "",
    frequency: PurchaseOrdersFrequencyOptions.Weekly,
    division: "",
    description: "",
    total: 0,
    currency: "",
    approval_total_home: 0,
    covered_within_project_budget: false,
    payment_type: PurchaseOrdersPaymentTypeOptions.OnAccount,
    vendor: "",
    job: "",
    category: "",
    kind: "",
  };
}

export function purchaseOrderFromTemplate(
  template: PurchaseOrderTemplateSource,
): Partial<PurchaseOrdersRecord> {
  const item = blankPurchaseOrder();

  return {
    ...item,
    type: template.type ?? item.type,
    date: template.date ?? item.date,
    end_date: template.end_date ?? item.end_date,
    frequency: template.frequency ?? item.frequency,
    division: template.division ?? item.division,
    description: template.description ?? item.description,
    total: template.total ?? item.total,
    currency: template.currency ?? item.currency,
    covered_within_project_budget:
      template.covered_within_project_budget ?? item.covered_within_project_budget,
    payment_type: template.payment_type ?? item.payment_type,
    vendor: template.vendor ?? item.vendor,
    job: template.job ?? item.job,
    category: template.category ?? item.category,
    kind: template.kind ?? item.kind,
    branch: template.branch ?? "",
    parent_po: "",
    approver: "",
    priority_second_approver: "",
    attachment: "",
  };
}
