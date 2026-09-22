export type JobWIP = {
  project_value: number;
  as_of: string;
  no_rate_sheet: boolean;
  time_value: number;
  hours: number;
  estimated_hours: number;
  unpriced_hours: number;
  expense_value: number;
  unpriced_expenses: number;
  po_value: number;
  unpriced_pos: number;
  estimated_pos: number;
};

// Pricing and PO balances come from SQL. These controls only select factors
// and their display denominators; they never change the PO expense deduction.
export function wipView(data: JobWIP, includeExpenses: boolean, includePOs: boolean) {
  const factors = [
    {
      key: "time",
      name: "Time value",
      value: data.time_value,
      included: true,
      partial: data.unpriced_hours > 0,
      estimated: data.estimated_hours > 0,
      color: "#2563eb",
    },
    {
      key: "expenses",
      name: "Expenses",
      value: data.expense_value,
      included: includeExpenses,
      partial: data.unpriced_expenses > 0,
      estimated: false,
      color: "#0f766e",
    },
    {
      key: "pos",
      name: "Remaining active POs",
      value: data.po_value,
      included: includePOs,
      partial: data.unpriced_pos > 0,
      estimated: data.estimated_pos > 0,
      color: "#b45309",
    },
  ];
  const included = factors.filter((factor) => factor.included);
  const total = Math.round(included.reduce((sum, factor) => sum + factor.value, 0) * 100) / 100;
  const partial = included.some((factor) => factor.partial);
  const estimated = included.some((factor) => factor.estimated);
  const percent = !partial && data.project_value > 0 ? (total * 100) / data.project_value : null;
  // Signed corrections can be reported in the table, but cannot be drawn as
  // pie slices or positive stacked segments.
  const chartable = !partial && included.every((factor) => factor.value >= 0);
  const scale = Math.max(data.project_value, total, 0);
  let offset = 0;
  const rows = factors.map((factor) => {
    const share = factor.included && total > 0 && chartable ? (factor.value * 100) / total : 0;
    const row = {
      ...factor,
      share,
      offset,
      percent:
        factor.included && !factor.partial && data.project_value > 0
          ? (factor.value * 100) / data.project_value
          : null,
      width: factor.included && scale > 0 && chartable ? (factor.value * 100) / scale : 0,
    };
    offset += share;
    return row;
  });
  return {
    rows,
    total,
    partial,
    estimated,
    percent,
    chartable,
    balance: partial ? null : Math.round((data.project_value - total) * 100) / 100,
    budgetMarker: scale > 0 ? (data.project_value * 100) / scale : 0,
  };
}
