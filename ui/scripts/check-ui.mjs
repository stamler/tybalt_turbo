#!/usr/bin/env node
// Flags UI patterns that bypass the shared components (see ui/CLAUDE.md and
// descriptions/ui_design_system.md). Existing occurrences are recorded in
// check-ui-baseline.json; the check fails only when a file has more of a
// pattern than its baseline, so new drift is blocked without a big-bang fix.
//
//   node scripts/check-ui.mjs           check against the baseline
//   node scripts/check-ui.mjs --update  lower baseline counts after a cleanup
//                                       (never raises a count or adds a file)
//   node scripts/check-ui.mjs --init    rewrite the baseline from scratch
//                                       (humans only, e.g. after a file move)
//
// A deliberate exception is allowed with "ui-check-allow: <reason>" on the
// same line or the line above.

import { readFileSync, readdirSync, writeFileSync, existsSync } from "node:fs";
import { dirname, join, relative, basename } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const baselinePath = join(uiRoot, "scripts", "check-ui-baseline.json");
const scanRoot = join(uiRoot, "src");
const componentsDir = join(scanRoot, "lib", "components");

// pending: the replacement component is not built yet, so an increase is a
// warning instead of a failure. Remove it once the component lands.
const rules = [
  {
    id: "raw-button",
    pattern: /<button\b/g,
    use: "DSActionButton",
  },
  {
    id: "raw-details",
    pattern: /<details\b/g,
    use: "DSCollapsible",
  },
  {
    id: "off-palette-grey",
    pattern: /(?<![\w-])(?:[a-z-]+:)*[a-z]+(?:-[a-z]+)*-(?:gray|slate|zinc|stone)-\d{2,3}\b/g,
    use: "neutral greys (e.g. text-neutral-600)",
  },
  {
    id: "off-scale-radius",
    pattern: /(?<![\w-])(?:[a-z-]+:)*rounded(?:-(?:md|lg|xl|2xl|3xl))?(?=[\s"'`}])/g,
    use: "rounded-sm (rounded-full for pills)",
  },
  {
    id: "hand-styled-link",
    pattern:
      /class="(?=[^"]*(?<![\w:-])text-blue-\d{3}\b)(?=[^"]*(?<![\w-])(?:hover:)?underline\b)[^"]*"/g,
    use: 'class="link"',
  },
  {
    id: "hand-built-overlay",
    pattern: /\bfixed inset-0\b/g,
    use: "DSModal (DSPopover for confirms) or DSFlyout",
  },
  {
    id: "hand-tinted-box",
    pattern: /(?<![\w:-])bg-(?:amber|yellow|red|green|blue|orange|sky|emerald|rose)-50\b/g,
    use: "DSCallout",
  },
  {
    id: "hand-written-field",
    pattern: /<span class="[^"]*\bfont-semibold\b[^"]*">[^<{]+:\s*<\/span>/g,
    use: "DSFieldList / DSField",
  },
  {
    id: "raw-page-title",
    pattern: /<h1\b/g,
    use: "DSPageHeader",
  },
];

// Shared DS components own the styling, so they are not scanned.
function isScanned(path) {
  if (!path.endsWith(".svelte")) return false;
  return !(dirname(path) === componentsDir && /^ds/i.test(basename(path)));
}

function* walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) yield* walk(path);
    else if (isScanned(path)) yield path;
  }
}

function lineOf(text, index) {
  let line = 1;
  for (let i = 0; i < index; i++) if (text.charCodeAt(i) === 10) line++;
  return line;
}

// { ruleId: { file: [lines] } }
function scan() {
  const found = Object.fromEntries(rules.map((r) => [r.id, {}]));
  for (const path of walk(scanRoot)) {
    const text = readFileSync(path, "utf8");
    const lines = text.split("\n");
    const file = relative(uiRoot, path);
    for (const rule of rules) {
      for (const match of text.matchAll(rule.pattern)) {
        const line = lineOf(text, match.index);
        const allowed = [lines[line - 1], lines[line - 2]].some((l) =>
          l?.includes("ui-check-allow:"),
        );
        if (allowed) continue;
        (found[rule.id][file] ??= []).push(line);
      }
    }
  }
  return found;
}

function counts(found) {
  const out = {};
  for (const rule of rules) {
    const files = Object.keys(found[rule.id]).sort();
    out[rule.id] = Object.fromEntries(files.map((f) => [f, found[rule.id][f].length]));
  }
  return out;
}

function writeBaseline(baseline) {
  writeFileSync(baselinePath, JSON.stringify(baseline, null, 2) + "\n");
}

const args = new Set(process.argv.slice(2));
const found = scan();
const current = counts(found);

if (args.has("--init")) {
  writeBaseline(current);
  console.log(`check-ui: wrote ${relative(uiRoot, baselinePath)}`);
  process.exit(0);
}

if (!existsSync(baselinePath)) {
  console.error("check-ui: no baseline; a human should run `node scripts/check-ui.mjs --init`");
  process.exit(1);
}
const baseline = JSON.parse(readFileSync(baselinePath, "utf8"));

let failures = 0;
let warnings = 0;
let lowered = 0;
const next = {};
for (const rule of rules) {
  const was = baseline[rule.id] ?? {};
  const now = current[rule.id];
  next[rule.id] = {};
  for (const file of Object.keys(now)) {
    const allowedCount = was[file] ?? 0;
    if (now[file] > allowedCount) {
      const kind = rule.pending ? "warning" : "error";
      if (rule.pending) warnings++;
      else failures++;
      console.log(
        `${kind}: ${file}: ${rule.id} ${now[file]} (baseline ${allowedCount}) ` +
          `at lines ${found[rule.id][file].join(", ")} -> use ${rule.use}`,
      );
    }
    next[rule.id][file] = Math.min(now[file], allowedCount);
    if (next[rule.id][file] === 0) delete next[rule.id][file];
  }
  for (const file of Object.keys(was)) {
    if ((next[rule.id][file] ?? 0) < was[file]) lowered++;
  }
}

if (args.has("--update")) {
  writeBaseline(next);
  console.log(`check-ui: lowered ${lowered} baseline entr${lowered === 1 ? "y" : "ies"}`);
} else if (lowered > 0) {
  console.log(
    `check-ui: ${lowered} baseline entr${lowered === 1 ? "y is" : "ies are"} now lower; ` +
      "run `node scripts/check-ui.mjs --update` to lock in the improvement",
  );
}

if (failures > 0) {
  console.log(
    `check-ui: ${failures} new violation(s). Use the shared component named above, ` +
      "or mark a deliberate exception with `ui-check-allow: <reason>`. See ui/CLAUDE.md.",
  );
  process.exit(1);
}
console.log(`check-ui: ok${warnings ? ` (${warnings} warning(s))` : ""}`);
