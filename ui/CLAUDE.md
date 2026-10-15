# UI conventions (ui/)

The app must look like one product. Consistency comes from reusing shared
components, not from re-creating their look with Tailwind on each page.
Full rationale and component APIs: ../descriptions/ui_design_system.md.

## Before writing any markup

1. Find the closest existing page that does something similar and match it.
2. Use the components below. If none fits, extend one with a prop, or add a
   new `DS*` component in src/lib/components/ and use that. Never style a
   one-off version on the page, and never copy a component's markup into a
   page to tweak it.
3. Pages may use layout utilities only (flex, grid, gap, space, padding,
   width). Colours, borders, radius, shadows and text sizes live inside
   components.

## Which component for what

| Need                                | Use                                                                                                                                    |
| ----------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| Any button                          | DSActionButton (`variant`: chip, primary, danger, secondary, plain)                                                                    |
| Top of a detail page                | DSPageHeader (title, badges, context line, actions, ⋯ menu)                                                                            |
| Grey box of related fields          | DSCard inside DSCardGrid                                                                                                               |
| Page section with a heading, no box | DSSection                                                                                                                              |
| Label and value                     | DSFieldList / DSField                                                                                                                  |
| Notice, warning or error box        | DSCallout (`tone`: info, success, warning, danger)                                                                                     |
| Confirm or small form dialog        | DSPopover                                                                                                                              |
| Any other modal                     | DSModal                                                                                                                                |
| Anchored panel or menu              | DSFlyout with DSMenuItem                                                                                                               |
| Chip or tag                         | DsLabel (`tone` for meaning, `color` for categories, `size="sm"`)                                                                      |
| Job, PO or expense status           | DSStatusLabel                                                                                                                          |
| Show/hide block                     | DSCollapsible                                                                                                                          |
| Tabs                                | DSTabBar                                                                                                                               |
| Lists of records                    | DSList / DSSearchList                                                                                                                  |
| Inputs                              | DSTextInput, DSSelector, DSAutoComplete, DSDateInput, DSCurrencyInput, DSToggle, DsCheck, DsFileSelect, DSLocationPicker, DSTokenInput |
| Files                               | DsFileLink, DsFileSelect                                                                                                               |
| Editing-disabled notice             | DsEditingDisabledBanner                                                                                                                |

Every component is shown in each variant at /dev/components (admins only;
source in src/routes/dev/components/+page.svelte).

## Look

- Greys are `neutral` only. Never gray, slate, zinc or stone.
- Meaning maps to colour: warning = amber, danger = red, success = green,
  info = blue. Components should take a `tone`, not a raw colour.
- Radius is `rounded-sm`. Only pills use `rounded-full`; only a modal panel
  uses `rounded-lg`.
- Links use the `link` utility: `<a class="link" href=...>`.
- Dates go through shortDate(); money through formatCurrency() or
  formatCurrencyAmount() (src/lib/utilities.ts). Never print raw date
  strings or numbers.
- Hide empty fields rather than printing "No X" or a blank label.
- A detail page's title is the record itself (its number or name), not
  "X Details".

## Never

- Hand-build an overlay (`fixed inset-0`) or a z-index stack; use DSModal
  or DSFlyout.
- Write a raw `<button>`; use DSActionButton.
- Write `<span class="font-semibold">Label:</span> value` rows; use DSField.
- Hand-tint a notice box (`bg-amber-50` and friends); use DSCallout.

## The check

`npm run check:ui` (also the first step of `npm run lint`) runs
scripts/check-ui.mjs. It fails when a file gains a pattern above the
recorded baseline in scripts/check-ui-baseline.json. Run it after UI changes.

- Fix a failure by using the component it names.
- A deliberate exception needs `ui-check-allow: <reason>` on the same line
  or the line above (`<!-- ui-check-allow: ... -->` in markup).
- After removing violations, run `node scripts/check-ui.mjs --update` to
  lower the baseline. Never edit the baseline by hand or run `--init` to
  make a failure go away.

## Where things are

- Shared components: src/lib/components/ (domain folders: clients/, jobs/)
- Generated PocketBase types: src/lib/pocketbase-types.ts
- Formatting helpers: src/lib/utilities.ts
- Global CSS and the `link` utility: src/app.css
