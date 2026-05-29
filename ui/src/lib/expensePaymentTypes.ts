import { ExpensesPaymentTypeOptions, type SelectOption } from "$lib/pocketbase-types";

export type ExpensePaymentTypeOption = SelectOption & {
  id: ExpensesPaymentTypeOptions;
  name: string;
};

export const allExpensePaymentTypeOptions: ExpensePaymentTypeOption[] = [
  { id: ExpensesPaymentTypeOptions.OnAccount, name: "On Account" },
  { id: ExpensesPaymentTypeOptions.Expense, name: "Expense" },
  {
    id: ExpensesPaymentTypeOptions.CorporateCreditCard,
    name: "Corporate Credit Card",
  },
  { id: ExpensesPaymentTypeOptions.Allowance, name: "Allowance" },
  { id: ExpensesPaymentTypeOptions.FuelCard, name: "Fuel Card" },
  { id: ExpensesPaymentTypeOptions.Mileage, name: "Mileage" },
  {
    id: ExpensesPaymentTypeOptions.PersonalReimbursement,
    name: "Personal Reimbursement",
  },
];

export function isExpensePaymentType(value: string): value is ExpensesPaymentTypeOptions {
  return allExpensePaymentTypeOptions.some((option) => option.id === value);
}

export function selectableExpensePaymentTypeOptions(
  allowPersonalReimbursement: boolean,
  currentPaymentType = "",
): ExpensePaymentTypeOption[] {
  if (
    allowPersonalReimbursement ||
    currentPaymentType === ExpensesPaymentTypeOptions.PersonalReimbursement
  ) {
    return allExpensePaymentTypeOptions;
  }

  return allExpensePaymentTypeOptions.filter(
    (option) => option.id !== ExpensesPaymentTypeOptions.PersonalReimbursement,
  );
}

export function defaultableExpensePaymentTypeOptions(
  allowPersonalReimbursement: boolean,
): ExpensePaymentTypeOption[] {
  return selectableExpensePaymentTypeOptions(allowPersonalReimbursement);
}
