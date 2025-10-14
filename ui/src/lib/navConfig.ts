export interface NavButton {
  action: string;
  icon: string;
  title: string;
  color?: string;
}

export interface NavItem {
  label: string;
  href: string;
  buttons: NavButton[];
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
        buttons: [
          {
            action: "/time/entries/add",
            icon: "feather:plus-circle",
            title: "New Entry",
            color: "green",
          },
        ],
      },
      { label: "Sheets", href: "/time/sheets/list", buttons: [] },
      { label: "Pending My Approval", href: "/time/sheets/pending", buttons: [] },
      { label: "Approved By Me", href: "/time/sheets/approved", buttons: [] },
      { label: "Shared with Me", href: "/time/sheets/shared", buttons: [] },
      { label: "Tracking", href: "/time/tracking", buttons: [] },
      {
        label: "Amendments",
        href: "/time/amendments/list",
        buttons: [
          {
            action: "/time/amendments/pending",
            icon: "feather:list",
            title: "Pending Amendments",
            color: "purple",
          },
          {
            action: "/time/amendments/add",
            icon: "feather:plus-circle",
            title: "New Amendment",
            color: "green",
          },
        ],
      },
      { label: "Time Off", href: "/time/off", buttons: [] },
    ],
  },
  {
    title: "Purchase Orders",
    items: [
      {
        label: "My Purchase Orders",
        href: "/pos/list",
        buttons: [
          {
            action: "/pos/add",
            icon: "feather:plus-circle",
            title: "New PO",
            color: "green",
          },
        ],
      },
      { label: "Pending My Approval", href: "/pos/pending", buttons: [] },
      { label: "All Active", href: "/pos/active", buttons: [] },
      { label: "Stale", href: "/pos/stale", buttons: [] },
    ],
  },
  {
    title: "Expenses",
    items: [
      {
        label: "My Expenses",
        href: "/expenses/list",
        buttons: [
          {
            action: "/expenses/add",
            icon: "feather:plus-circle",
            title: "New Expense",
            color: "green",
          },
        ],
      },
      { label: "Pending My Approval", href: "/expenses/pending", buttons: [] },
      { label: "Approved By Me", href: "/expenses/approved", buttons: [] },
    ],
  },
  {
    title: "Business",
    items: [
      {
        label: "Jobs",
        href: "/jobs/list",
        buttons: [
          {
            action: "/jobs/map",
            icon: "feather:map",
            title: "Jobs Map",
            color: "blue",
          },
          {
            action: "/jobs/add",
            icon: "feather:plus-circle",
            title: "New Job",
            color: "green",
          },
        ],
      },
      {
        label: "Clients",
        href: "/clients/list",
        buttons: [
          {
            action: "/clients/add",
            icon: "feather:plus-circle",
            title: "New Client",
            color: "green",
          },
        ],
      },
      {
        label: "Vendors",
        href: "/vendors/list",
        buttons: [
          {
            action: "/vendors/add",
            icon: "feather:plus-circle",
            title: "New Vendor",
            color: "green",
          },
        ],
      },
      { label: "Absorb Actions", href: "/absorb/actions", buttons: [] },
      { label: "Admin Profiles", href: "/admin_profiles/list", buttons: [] },
    ],
  },
  {
    title: "Reports",
    items: [
      { label: "Payroll", href: "/reports/payroll", buttons: [] },
      { label: "Weekly", href: "/reports/weekly", buttons: [] },
    ],
  },
  {
    title: "Settings",
    items: [
      { label: "Time Types", href: "/timetypes", buttons: [] },
      { label: "Divisions", href: "/divisions", buttons: [] },
    ],
  },
];
