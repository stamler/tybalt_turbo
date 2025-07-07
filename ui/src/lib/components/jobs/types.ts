export interface FilterDef {
  type: string;
  label: string;
  queryParam?: string;
  summaryProperty: string;
  valueProperty: string;
  displayProperty: string;
  color: "blue" | "green" | "purple" | "teal" | "yellow" | "red" | "gray";
}
