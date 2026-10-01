import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  amountInCents,
  authorizedAmountText,
  defaultPOPrintOptions,
  poPrintAcknowledgementKey,
  poPrintSearchParams,
  poPrintStorageKey,
  readPOPrintRequest,
  poPrintWarning,
  readPOPrintOptions,
  validatePOPrintOptions,
  type PrintablePO,
  type POPrintOptions,
} from "../src/lib/poPrint.ts";

const fixtures: Record<string, PrintablePO> = JSON.parse(
  readFileSync(new URL("./fixtures/poPrint.json", import.meta.url), "utf8"),
);
const plain = (amount: string): POPrintOptions => ({ amount, plusTax: false, plusShipping: false });
const normalized = (text: string) => text.replace(/\s/g, " ");

test("default amount uses the limit for each PO type without changing the PO", () => {
  for (const [name, amount] of [
    ["oneTime", "10000.00"],
    ["cumulative", "7000.00"],
    ["recurring", "500.00"],
    ["foreign", "100.10"],
  ]) {
    const po = fixtures[name];
    const original = structuredClone(po);
    assert.deepEqual(readPOPrintOptions(po, new URLSearchParams()), plain(amount));
    assert.deepEqual(po, original);
  }
});

test("valid positive amounts use exact cents and invalid amounts are rejected", () => {
  for (const amount of ["0.01", "100.10", "9999.99", "10000.00"]) {
    assert.equal(validatePOPrintOptions(fixtures.oneTime, plain(amount)), null);
  }
  assert.equal(amountInCents("0.29"), 29);
  for (const amount of [
    "",
    " ",
    " 1 ",
    "0",
    "-1",
    "abc",
    "NaN",
    "Infinity",
    "1e3",
    "0x10",
    "1,000",
    "1.001",
    "10000.01",
    "9007199254740991",
  ]) {
    assert.ok(validatePOPrintOptions(fixtures.oneTime, plain(amount)), amount);
  }
  for (const name of ["exhausted", "overdrawn", "closed"]) {
    assert.ok(validatePOPrintOptions(fixtures[name], plain("1")));
    assert.throws(() => readPOPrintOptions(fixtures[name], new URLSearchParams()));
  }
});

test("all charge combinations have correct text and share one allowance", () => {
  for (const [plusTax, plusShipping, suffix, warning] of [
    [false, false, "", null],
    [true, false, " plus tax", "Tax"],
    [false, true, " plus shipping", "Shipping"],
    [true, true, " plus tax and shipping", "Tax and shipping combined"],
  ] as const) {
    const options = { amount: "8000", plusTax, plusShipping };
    assert.equal(
      normalized(authorizedAmountText(fixtures.oneTime, options)),
      `CAD 8,000.00${suffix}`,
    );
    const text = poPrintWarning(fixtures.oneTime, options);
    if (warning) {
      assert.ok(normalized(text!).startsWith(`${warning} must not exceed CAD 2,000.00.`));
      assert.match(text!, /new PO approved before authorizing/);
      assert.ok(validatePOPrintOptions(fixtures.oneTime, { ...options, amount: "10000" }));
    } else assert.equal(text, null);
    assert.deepEqual(
      readPOPrintOptions(fixtures.oneTime, poPrintSearchParams(fixtures.oneTime, options)),
      options,
    );
  }
});

test("cumulative, recurring, and foreign warnings use the right amount and period", () => {
  const both = { amount: "6000", plusTax: true, plusShipping: true };
  assert.match(normalized(poPrintWarning(fixtures.cumulative, both)!), /CAD 1,000.00/);
  assert.ok(validatePOPrintOptions(fixtures.cumulative, plain("7000.01")));
  assert.match(
    normalized(poPrintWarning(fixtures.foreign, { ...both, amount: "100" })!),
    /USD 0.10/,
  );
  assert.match(
    normalized(poPrintWarning(fixtures.recurring, { ...both, amount: "400" })!),
    /CAD 100.00 per period/,
  );
  assert.equal(
    normalized(authorizedAmountText(fixtures.recurring, { ...both, amount: "400" })),
    "CAD 400.00 plus tax and shipping / Monthly",
  );
});

test("direct print links reject incomplete, duplicate, malformed, and unacknowledged options", () => {
  for (const query of [
    "amount=8000",
    "plusTax=1",
    "amount=8000&plusTax=1&plusShipping=0&limit=10000&acknowledged=0",
    "amount=8000&plusTax=true&plusShipping=0&limit=10000&acknowledged=1",
    "amount=8000&plusTax=0&plusShipping=2&limit=10000&acknowledged=0",
    "amount=8000&plusTax=0&plusShipping=0&limit=NaN&acknowledged=0",
  ])
    assert.throws(() => readPOPrintOptions(fixtures.oneTime, new URLSearchParams(query)), query);
  const valid = poPrintSearchParams(fixtures.oneTime, plain("8000"));
  for (const key of [...valid.keys()]) {
    const duplicate = new URLSearchParams(valid);
    duplicate.append(key, valid.get(key)!);
    assert.throws(() => readPOPrintOptions(fixtures.oneTime, duplicate), key);
  }
  assert.throws(() =>
    readPOPrintOptions(fixtures.oneTime, poPrintSearchParams(fixtures.oneTime, plain("10000.01"))),
  );
});

test("fresh balances prevent stale printing and require a new warning acknowledgement", () => {
  const options = { amount: "6000", plusTax: true, plusShipping: true };
  const params = poPrintSearchParams(fixtures.cumulative, options);
  for (const print_max_amount of [6500, 7500, 0]) {
    // Model a new API response after another expense was committed or uncommitted.
    const currentPO = { ...fixtures.cumulative, print_max_amount };
    assert.throws(() => readPOPrintOptions(currentPO, params), /changed/);
  }
  assert.throws(
    () =>
      readPOPrintOptions(fixtures.cumulative, poPrintSearchParams(fixtures.oneTime, plain("8000"))),
    /exceeds/,
  );
  const key = poPrintAcknowledgementKey(fixtures.cumulative, options);
  for (const change of [{ amount: "5999" }, { plusTax: false }, { plusShipping: false }]) {
    assert.notEqual(poPrintAcknowledgementKey(fixtures.cumulative, { ...options, ...change }), key);
  }
  assert.notEqual(poPrintAcknowledgementKey(fixtures.oneTime, options), key);
});

test("print URLs contain only a key and tab-local options are validated again", () => {
  const token = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee";
  const po = fixtures.cumulative;
  const options = { amount: "6000", plusTax: true, plusShipping: true };
  const key = poPrintStorageKey("printpo", token);
  const saved = poPrintSearchParams(po, options).toString();
  const storage = { getItem: (name: string) => (name === key ? saved : null) };
  const params = new URLSearchParams({ print: token });
  assert.deepEqual(readPOPrintRequest(po, "printpo", params, storage), options);
  assert.throws(() => readPOPrintRequest(po, "anotherpo", params, storage), /not available/);
  assert.throws(
    () => readPOPrintRequest(po, "printpo", params, { getItem: () => null }),
    /not available/,
  );
  for (const query of [
    "print=invalid",
    `print=${token}&print=${token}`,
    `print=${token}&limit=7000`,
    "amount=6000",
  ]) {
    assert.throws(() => readPOPrintRequest(po, "printpo", new URLSearchParams(query), storage));
  }
  assert.throws(
    () => readPOPrintRequest(po, "printpo", params, { getItem: () => "amount=8000" }),
    /Invalid/,
  );
  assert.deepEqual(
    readPOPrintRequest(po, "printpo", new URLSearchParams(), storage),
    defaultPOPrintOptions(po),
  );
});
