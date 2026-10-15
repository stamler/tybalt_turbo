# UI Design System Proposal

Status: accepted 2026-10-15 with the section 7 recommendations. Phases 1 (rules and check), 2 (building blocks and the `/dev/components` gallery) and 3 (job details pilot) are implemented; phases 4–6 are not.

## 1. Why

Pages have been built one at a time, mostly by AI models, each seeing only the page in front of it. Where a shared component exists it is mostly used (`DSActionButton` 238 uses, `DsLabel` 31, `DSList` 39). Where none exists, every page invents its own:

| Pattern          | Shared component today                     | What the code does                                                                                                                                                                                                                |
| ---------------- | ------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Modal dialog     | `DSPopover` (confirm dialog only)          | 7 more hand-built overlays (`RejectModal`, `ShareModal`, `CumulativePOOverflowModal`, `BranchResourceFlowDialog`, `StoredFileHashRepairPopover`, `JobValueProposal`, the review dialog in `JobValueStatus`) at `z-50` or `z-9999` |
| Anchored flyout  | none                                       | 4 hand-built click-away panels (`JobValueLine` history, `VersionInfo`, `DSCurrencyInput` guidance, `TimeEntriesEditor`)                                                                                                           |
| Button           | `DSActionButton`                           | 144 raw `<button>`s in 58 files, about 20 background colours                                                                                                                                                                      |
| Notice box       | `DsEditingDisabledBanner` (single purpose) | 27 files hand-tint boxes: 18 amber, 5 yellow, the rest red/blue/green                                                                                                                                                             |
| Label: value row | none                                       | written by hand ~30 times each on the job, expense, PO and admin-profile detail pages                                                                                                                                             |
| Page header      | none                                       | 11 different `<h1>` class strings; 8 detail pages use about 4 layouts                                                                                                                                                             |
| Status chip      | `DsLabel`                                  | status-to-colour ternaries repeated in 3 files                                                                                                                                                                                    |
| Collapsible      | `DSCollapsible` (1 use)                    | the job page hand-builds a `<details>` with identical classes                                                                                                                                                                     |
| Link             | none                                       | `text-blue-600 hover:underline` (45) vs `text-blue-700 underline` (45), plus 4 other variants                                                                                                                                     |

Palette and shape drift too: `neutral` 623 uses, `gray` 26, `slate` 17; `rounded-sm` 222, `rounded` 51, `rounded-lg` 30, `rounded-md` 18. `DsExternalLinkButton` and `QueryBox` have no importers.

The root cause is that the house style is not written anywhere a model reads it, and nothing checks it. This proposal (a) writes down the look we already mostly have, (b) fills the gaps in the component set, (c) puts the rules in `ui/CLAUDE.md`, and (d) adds a check so drift is caught at lint time.

## 2. Principles

1. **Reuse before writing.** Before writing markup, look for a `DS` component or a sibling page that already does it.
2. **A new pattern becomes a shared component.** If nothing fits, add or extend a component in `lib/components/` rather than styling the page. Extend with a prop before forking a near-copy.
3. **Pages compose, components style.** Pages may use layout utilities (`flex`, `grid`, `gap-*`, `space-y-*`, `p-*`, widths). Colour, border, radius, shadow and type-size classes belong inside components.
4. **One way to do each thing.** When two styles exist, pick the majority and migrate the rest.
5. **Write down what exists.** The canonical look below is the app's current majority style, not a redesign.

## 3. Canonical look

### Colour

- **Greys:** `neutral` only. No `gray`, `slate`, `zinc`, `stone`.
- **Tones** (what a colour means). Components take a `tone` prop, never a raw colour:

  | Tone      | Tailwind colour | Use for                                       |
  | --------- | --------------- | --------------------------------------------- |
  | `neutral` | neutral         | default, closed/cancelled states              |
  | `info`    | blue            | links, primary actions, informational notices |
  | `success` | green           | active/approved states, create actions        |
  | `warning` | amber           | needs attention, pending, mismatches          |
  | `danger`  | red             | rejected, destructive actions, errors         |

- **Categorical colours** (`cyan`, `teal`, `purple`, `orange`, `yellow`) are allowed only on `DsLabel` and filter chips, where colour distinguishes kinds of thing and carries no meaning. `yellow` also stays as `DSActionButton`'s existing default.
- **Static class maps.** Components look up full class strings from a map (`{ warning: "border-amber-300 bg-amber-50 text-amber-900" }`) instead of building `bg-${color}-200`. Tailwind then sees every class, and the `@source inline(...)` safelist in `app.css` can shrink as components migrate.

### Surfaces and shape

| Element                      | Classes                                                                                       |
| ---------------------------- | --------------------------------------------------------------------------------------------- |
| Page                         | `mx-auto space-y-4 p-4`                                                                       |
| Card (grey box)              | `rounded-sm bg-neutral-100 p-4`                                                               |
| Card header strip (optional) | `bg-neutral-200 px-4 py-2`                                                                    |
| Borders                      | `border-neutral-300`                                                                          |
| Radius                       | `rounded-sm` everywhere; `rounded-full` only for pills; `rounded-lg` only for the modal panel |
| Shadow                       | none, except modal panel (`shadow-xl`) and flyouts (`shadow-lg`)                              |

### Type

| Role                 | Classes                                      |
| -------------------- | -------------------------------------------- |
| Page title (`h1`)    | `text-2xl font-bold`                         |
| Section title (`h2`) | `text-lg font-semibold`                      |
| Card title (`h3`)    | `font-semibold`                              |
| Field label          | `text-sm text-neutral-600` (see decision D1) |
| Meta and hints       | `text-sm text-neutral-600`                   |

### Links, icons, values

- **Links:** one `link` utility defined with `@utility` in `app.css`: `text-blue-700 hover:underline`, with a visible focus outline. Blue-700 is the more legible of the two current shades on grey cards.
- **Icons:** `mdi:` via `@iconify/svelte`; 24px in buttons, 18px inline in text.
- **Dates:** always `shortDate()`; never print a raw date string.
- **Money:** always `formatCurrency()` / `formatCurrencyAmount()`.
- **Empty values:** omit the field. Use `—` only where a grid position must hold its place.
- **Booleans:** `Yes` / `No` in a field. A flag that is rarely true and worth noticing may instead be a label in the page header.

## 4. Building blocks

All new components use the `DS` prefix (decision D5). Existing components keep their names for now.

### Existing components: keep, with small changes

**`DSActionButton`**, the only button. Add a `variant` prop:

- `"chip"` (default): today's compact button.
- `"primary"`: filled blue, white text, `px-4 py-2`. For form submits and dialog confirms.
- `"danger"`: filled red. For destructive confirms.
- `"secondary"`: filled grey. For Cancel beside a primary or danger button (added in phase 2 so `DSPopover` keeps its look).
- `"plain"`: text-only, styled like a link.

Colour classes move into a static map. This replaces the raw buttons inside `DSPopover` and most of the 144 raw buttons elsewhere.

**`DsLabel`**:

- Add `size: "md" | "sm"`. `sm` replaces the hand-built `text-xs py-0.5` amber chips on the expense page.
- Accept `tone` alongside `color`, and move colours to a static map.

**`DSPopover`**: becomes a thin wrapper over the new `DSModal` (Cancel plus submit footer). Same API, so the 11 call sites don't change.

**`DSCollapsible`**: use it wherever `<details>` appears.

**`DSTabBar`**: swap `text-gray-700` for `text-neutral-700`.

**`DsEditingDisabledBanner`**: becomes `<DSCallout tone="warning">` internally.

### New components

#### `DSPageHeader`

The top of every detail page. It replaces "X Details" titles: the title is the thing itself.

```ts
{
  title: string;            // "25-123", "PO 2410-0042", "Jane Doe"
  subtitle?: string;        // description, job title, …
  badges?: Snippet;         // status and kind labels
  context?: Snippet;        // one muted line: "Subjob of 25-100 · from proposal P25-033"
  actions?: Snippet;        // frequent actions, right-aligned DSActionButtons
  menu?: Snippet;           // rare and admin actions, shown in a "⋯" DSFlyout as DSMenuItems
}
```

On mobile, actions wrap below the title. The `⋯` menu appears only when `menu` renders something.

#### `DSCard`

A grey box with an optional title row.

```ts
{
  title?: string;
  actions?: Snippet;        // right side of the title row
  span?: 1 | 2 | "full";    // column span inside a DSCardGrid
  children: Snippet;
}
```

`DSCardGrid` wraps cards: `grid gap-4 md:grid-cols-2 lg:grid-cols-3`, one column on phones.

#### `DSSection`

A page-level section with an `h2` heading and optional actions, and no box. For "Notes", "Claims", "Visible Expenses", "Rate Entries".

```ts
{ title: string; actions?: Snippet; children: Snippet }
```

#### `DSFieldList` and `DSField`

Semantic `<dl>` / `<dt>` / `<dd>`. This replaces every hand-written `<span class="font-semibold">Label:</span> value`.

```ts
// DSFieldList
{ columns?: 1 | 2; children: Snippet }        // 2 = two columns from md up

// DSField
{
  label: string;
  value?: string | number | null;   // omitted entirely when empty, unless showEmpty
  href?: string;                    // renders value as a link
  hint?: string;                    // muted suffix, e.g. "as of Oct 1, 2026"
  showEmpty?: boolean;              // render "—" instead of hiding
  children?: Snippet;               // custom value (chips, buttons) instead of value
}
```

#### `DSCallout`

The one notice box. It replaces the 27 hand-tinted boxes and the value-change banner, and sits under `DsEditingDisabledBanner`.

```ts
{
  tone: "info" | "success" | "warning" | "danger";
  title?: string;
  icon?: string | false;    // default per tone (mdi:information-outline, mdi:alert-outline, …)
  actions?: Snippet;        // buttons, right side or wrapped below
  children?: Snippet;
}
```

#### `DSModal`

The one modal shell, built on native `<dialog>` with `showModal()`. That gives the browser's top layer (no more z-index guessing), focus trapping, Escape to close, and `inert` page content.

```ts
{
  show: boolean;            // $bindable
  title: string;
  subtitle?: string;
  size?: "sm" | "md" | "lg";       // max-w-md / max-w-lg / max-w-2xl
  dismissible?: boolean;           // backdrop click and Escape close it (default true; false while submitting)
  onclose?: () => void;            // called instead of setting show=false on Escape or backdrop click
  children: Snippet;
  footer?: Snippet;                // usually DSActionButtons, right-aligned
}
```

Migrate in this order: `JobValueProposal`, the `JobValueStatus` review dialog, `StoredFileHashRepairPopover`, `RejectModal`, `ShareModal`, `CumulativePOOverflowModal`, `BranchResourceFlowDialog`.

#### `DSFlyout` and `DSMenuItem`

An anchored panel that closes on outside click or Escape. It replaces the four hand-built click-away panels and powers the `⋯` menu in `DSPageHeader`.

```ts
// DSFlyout
{
  open?: boolean;           // $bindable
  label: string;            // accessible name for the trigger and panel
  align?: "start" | "end";
  hideWhenEmpty?: boolean;  // hide the trigger while no DSMenuItem is inside (DSPageHeader's menu)
  trigger: Snippet<[{ toggle: () => void; open: boolean }]>;
  children: Snippet;
}

// DSMenuItem
{
  label: string;
  icon?: string;
  action: (() => void) | string;   // same contract as DSActionButton
  tone?: "neutral" | "danger";
}
```

#### `DSStatusLabel`

One map from record status to tone, shared by jobs, POs and expenses. Expenses have no status field; pass the display status the page already derives from the submitted, approved, committed and rejected timestamps.

```ts
{ status: string; kind?: "job" | "po" | "expense" }
```

- **Success:** Active, Awarded (jobs); Active (POs); Approved, Committed (expenses).
- **Warning:** Submitted, In Progress (jobs); Unapproved (POs); Submitted (expenses).
- **Danger:** Not Awarded (jobs); Rejected (POs and expenses).
- **Neutral:** Closed, Cancelled, No Bid.

This removes the ternaries in `PurchaseOrdersList`, `pos/[poid]/details` and `pos/search`.

### Component gallery

A `/dev/components` route, visible only to the admin claim, renders every `DS` component in each variant with sample data. People can see what exists. Models can be pointed at it, or at its source, as the reference.

### Enforcement: `ui/scripts/check-ui.mjs`

`npm run check:ui`, also the first step of `npm run lint`. It scans every `.svelte` file under `ui/src` except the `DS*` / `Ds*` components at the top of `src/lib/components/`, which own the styling.

| Rule                 | Flags                                                            | Use instead            | Status          |
| -------------------- | ---------------------------------------------------------------- | ---------------------- | --------------- |
| `raw-button`         | `<button`                                                        | `DSActionButton`       | error           |
| `raw-details`        | `<details`                                                       | `DSCollapsible`        | error           |
| `off-palette-grey`   | `*-gray-*`, `*-slate-*`, `*-zinc-*`, `*-stone-*`                 | `neutral`              | error           |
| `off-scale-radius`   | bare `rounded`, `rounded-(md\|lg\|xl\|2xl\|3xl)`                 | `rounded-sm`           | error           |
| `hand-styled-link`   | a class with `text-blue-NNN` and `underline` / `hover:underline` | `class="link"`         | error           |
| `hand-built-overlay` | `fixed inset-0`                                                  | `DSModal` / `DSFlyout` | error (phase 2) |
| `hand-tinted-box`    | `bg-(amber\|yellow\|red\|green\|blue\|…)-50` (not `hover:`)      | `DSCallout`            | error (phase 2) |
| `hand-written-field` | `<span class="…font-semibold…">Label:</span>`                    | `DSField`              | error (phase 2) |
| `raw-page-title`     | `<h1`                                                            | `DSPageHeader`         | error (phase 2) |

Existing occurrences are counted per file in `ui/scripts/check-ui-baseline.json`. A file that goes above its count fails (or warns, for rules whose replacement isn't built yet). `--update` lowers counts after a cleanup but never raises one or adds a file; `--init` rewrites the baseline and is for humans only, for example after moving a file. A deliberate exception gets `ui-check-allow: <reason>` on the same line or the line above. When a planned component lands, flip its rule from `pending` to an error.

Baseline at adoption (2026-10-15):

| Rule               | Files | Occurrences |
| ------------------ | ----: | ----------: |
| raw-button         |    52 |         135 |
| raw-details        |     2 |           3 |
| off-palette-grey   |    17 |          37 |
| off-scale-radius   |    31 |          98 |
| hand-styled-link   |    47 |         100 |
| hand-built-overlay |    11 |          14 |
| hand-tinted-box    |    34 |          62 |
| hand-written-field |     7 |         117 |
| raw-page-title     |    40 |          40 |

## 5. `ui/CLAUDE.md`

Implemented in phase 1; see `ui/CLAUDE.md` for the current text. Claude Code loads a nested `CLAUDE.md` whenever it works on files under that directory, so the rules reach every model that touches the UI. Until a planned component exists, the file lists it as planned and tells models to match the nearest existing page rather than build the component as a side effect of another task. As each component lands, move it from the "planned" paragraph into the component table.

## 6. Pilot: job details page

`routes/jobs/[jid]/details/+page.svelte` (1045 lines) is the pilot because it needs every new component.

### Layout

```
┌───────────────────────────────────────────────────────────────────────────┐
│ 25-123  [Active] [T&M]                          [✎ Edit] [+ Subjob] [⋯]  │
│ Highway 7 bridge inspection                                               │
│ Subjob of 25-100 · from proposal P25-033                                  │
└───────────────────────────────────────────────────────────────────────────┘
 ⚠ Value change to $480,000 pending · awaiting R. Singh      [Review] [Correct]

┌ Client & team ──────────┐ ┌ Value & contract ───────┐ ┌ Invoicing ───── Open ↗ ┐
│ Client   Acme Municipal │ │ Value    $450,000 ⬇ ⟲ … │ │ Municipal AP           │
│ Contact  Sam Lee        │ │ Outstanding $12,300     │ │ Jane Doe ·             │
│ Owner    Acme Municipal │ │          as of Oct 1    │ │ jane@acme.ca           │
│ Manager  Pat Kim        │ │ Rate sheet 2026 (rev 2) │ │ Quote PO 4411 on every │
│ Alternate  Chris Wu     │ │ Client PO  4411         │ │ invoice; send PDF …    │
│ Branch   Thunder Bay    │ │ Client ref  R-2290      │ └────────────────────────┘
└─────────────────────────┘ │ Authorizing  Letter     │
                            │ FN agreement  No        │
┌ Structure ──────────────┐ └─────────────────────────┘ ┌ Dates ─────────────────┐
│ Children  25-123-01, -02│                             │ Awarded   Mar 4, 2026  │
│ Divisions [Civil · 120h]│                             │ Completion Nov 30, 2026│
│ Categories [Bridge] …   │                             │ Requested by … on …    │
└─────────────────────────┘                             └────────────────────────┘
▸ Location (collapsed map)

Notes ─────────────────────────────────────────────────────── [+ Add note]
…

[Time] [Expenses] [Active POs] [WIP]
```

### Where every current field goes

| Current element                                                                                                                                 | New home                                                                                                 |
| ----------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| "Job Details" H1                                                                                                                                | removed; `DSPageHeader` title is the job number                                                          |
| Job Number, Description                                                                                                                         | header title, subtitle                                                                                   |
| "proposal" label, Status                                                                                                                        | header badges (`DsLabel`, `DSStatusLabel`)                                                               |
| Time & Materials                                                                                                                                | header badge "T&M" when yes; otherwise omitted                                                           |
| Parent Job, Proposal (on projects)                                                                                                              | header context line                                                                                      |
| Edit, Create Subjob, Award, Create project from proposal                                                                                        | header actions                                                                                           |
| Change Number, Change Client, Close imported project                                                                                            | header `⋯` menu                                                                                          |
| `JobValueStatus` banner                                                                                                                         | `DSCallout` directly under the header                                                                    |
| Client, Contact, Job Owner, Manager, Alternate Manager, Branch                                                                                  | Client & team card                                                                                       |
| Project Value + `JobValueLine` actions, Outstanding Balance, Rate Sheet, Client PO, Client Reference Number, Authorizing Document, FN Agreement | Value & contract card                                                                                    |
| Proposal Value (proposals)                                                                                                                      | Value card, replacing the project fields; proposal opening and submission-due dates go in the Dates card |
| Invoicing profile block                                                                                                                         | Invoicing card (below)                                                                                   |
| Children, Projects (on proposals), Divisions, Categories                                                                                        | Structure card; divisions become chips ("Civil · 120 h")                                                 |
| Project Award Date, Completion Date, Project Request line                                                                                       | Dates card, all via `shortDate`                                                                          |
| Location map                                                                                                                                    | `DSCollapsible` "Location", closed by default                                                            |
| "Additional Details" `<details>`                                                                                                                | removed (decision D3)                                                                                    |
| `ClientNotesSection` inside the grey box                                                                                                        | its own `DSSection` "Notes"                                                                              |
| Tabs                                                                                                                                            | unchanged                                                                                                |

### Invoicing card

```svelte
<DSCard title="Invoicing">
  {#snippet actions()}<a class="link" href={profileHref}>Open profile</a>{/snippet}
  <DSFieldList>
    <DSField label="Profile" value={job.invoicing_profile_name} />
    <DSField label="Send to">{contactName} · <a class="link" href={`mailto:${email}`}>{email}</a></DSField>
  </DSFieldList>
  <InvoicingInstructions text={job.invoice_instructions} />   <!-- renders nothing when empty -->
</DSCard>
```

- When a project has no invoicing profile, the card shows a muted "No invoicing profile" so the gap is visible.
- Proposals don't show the card.
- `InvoicingInstructions` stops rendering "No additional instructions". The client page passes a prop if it still wants that text.

### Breaking up the file

- `lib/components/jobs/JobHeader.svelte`: header, actions, menu, and the change-number, change-client and close-project dialogs they open.
- `lib/components/jobs/JobSummary.svelte`: the card grid.
- `lib/components/jobs/JobInvoicingCard.svelte`.
- The page keeps loading, tabs and filter definitions, and should land well under 400 lines.

Out of scope for the pilot, but noted: the Staff and Divisions summary sub-tabs repeat the same start/end date inputs. They could share one control.

## 7. Decisions (all accepted as recommended, 2026-10-15)

| #   | Question                                                                                          | Decision                                                                                           |
| --- | ------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------- |
| D1  | Field style: label stacked above value (small muted label) or today's inline bold `Label: value`? | Stacked. It aligns in card grids and scans faster. This is the one deliberate visual change.       |
| D2  | Warning colour: amber or yellow?                                                                  | Amber (18 of 23 existing warning boxes). Yellow stays for `DSActionButton` and categorical labels. |
| D3  | Drop the "Additional Details" collapse?                                                           | Yes on desktop. Compact cards keep the tabs near the fold.                                         |
| D4  | Keep the name `DSPopover` for a modal confirm dialog?                                             | Keep it for now (11 call sites); consider `DSConfirm` in a later mechanical rename.                |
| D5  | Prefix: `DS` or `Ds`? Both exist today.                                                           | `DS` for new components. Rename the `Ds*` files in one mechanical commit later, or leave them.     |
| D6  | Baseline-ratchet lint, or fix everything before turning the check on?                             | Ratchet. It blocks new drift from day one without a big-bang migration.                            |

## 8. Rollout

Each phase is its own commit or PR.

1. **Rules and check (done):** `ui/CLAUDE.md`, the `link` utility, `ui/scripts/check-ui.mjs` with its baseline. No visual change.
2. **Building blocks:** the new components, the `DSActionButton` / `DsLabel` / `DSPopover` changes, and the gallery route.
3. **Pilot (done):** job details, as specified in section 6. It lives in `JobHeader`, `JobSummary` and `JobInvoicingCard` under `src/lib/components/jobs/`.
4. **Other detail pages:** expense, PO, admin profile, rate sheet, vendor, claims, onto `DSPageHeader` / `DSCard` / `DSField`.
5. **Overlays:** the seven modals and four flyouts move onto `DSModal` / `DSFlyout`.
6. **Sweep:** remaining raw buttons, callouts, grey shades and radii. Delete unused components, trim the `app.css` safelist, and empty the baseline.
