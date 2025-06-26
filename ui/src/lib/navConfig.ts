export interface NavButton {
  action: string;
  icon: string;
  title: string;
  color?: string;
}

export interface NavItem {
  label: string;
  href: string;
  button?: NavButton;
}

export interface NavSection {
  title: string;
  items: NavItem[];
}

export const navSections: NavSection[] = [
  {
    title: "Time Management",
    items: [
      {
        label: "Entries",
        href: "/time/entries/list",
        button: {
          action: "/time/entries/add",
          icon: "feather:plus-circle",
          title: "New Entry",
          color: "green",
        },
      },
      { label: "Sheets", href: "/time/sheets/list" },
      { label: "Pending My Approval", href: "/time/sheets/pending" },
      { label: "Approved By Me", href: "/time/sheets/approved" },
      { label: "Shared with Me", href: "/time/sheets/shared" },
      {
        label: "Amendments",
        href: "/time/amendments/list",
        button: {
          action: "/time/amendments/add",
          icon: "feather:plus-circle",
          title: "New Amendment",
          color: "green",
        },
      },
      { label: "Time Off", href: "/time/off" },
    ],
  },
  {
    title: "Purchase Orders",
    items: [
      {
        label: "My Purchase Orders",
        href: "/pos/list",
        button: {
          action: "/pos/add",
          icon: "feather:plus-circle",
          title: "New PO",
          color: "green",
        },
      },
      { label: "Pending My Approval", href: "/pos/pending" },
      { label: "All Active", href: "/pos/active" },
      { label: "Stale", href: "/pos/stale" },
    ],
  },
  {
    title: "Expenses",
    items: [
      {
        label: "My Expenses",
        href: "/expenses/list",
        button: {
          action: "/expenses/add",
          icon: "feather:plus-circle",
          title: "New Expense",
          color: "green",
        },
      },
      { label: "Pending My Approval", href: "/expenses/pending" },
      { label: "Approved By Me", href: "/expenses/approved" },
    ],
  },
  {
    title: "Business",
    items: [
      {
        label: "Jobs",
        href: "/jobs/list",
        button: {
          action: "/jobs/add",
          icon: "feather:plus-circle",
          title: "New Job",
          color: "green",
        },
      },
      {
        label: "Clients",
        href: "/clients/list",
        button: {
          action: "/clients/add",
          icon: "feather:plus-circle",
          title: "New Client",
          color: "green",
        },
      },
      {
        label: "Vendors",
        href: "/vendors/list",
        button: {
          action: "/vendors/add",
          icon: "feather:plus-circle",
          title: "New Vendor",
          color: "green",
        },
      },
    ],
  },
  {
    title: "Reports",
    items: [
      { label: "Payroll", href: "/reports/payroll" },
      { label: "Weekly", href: "/reports/weekly" },
    ],
  },
  {
    title: "Settings",
    items: [
      { label: "Time Types", href: "/timetypes" },
      { label: "Divisions", href: "/divisions" },
    ],
  },
]; 