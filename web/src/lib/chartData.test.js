import { test } from "node:test";
import assert from "node:assert/strict";
import { chartNumbers } from "./chartData.js";

// Mirror Recharts' own slice total: `sum + (isNumber(val) ? val : 0)`, where
// isNumber accepts only the number type (recharts/es6/util/DataUtils). This
// encodes *why* coercion is required, so the spec fails loudly if the
// contract ever changes.
const rechartsIsNumber = (value) => typeof value === "number" || value instanceof Number;
function pieSum(rows, dataKey) {
  return rows.reduce((total, row) => total + (rechartsIsNumber(row[dataKey]) ? row[dataKey] : 0), 0);
}

// The backend serializes models.Money (shopspring decimal) as JSON strings.
const apiStatusBreakdown = [
  { key: "draft", name: "Draft", value: "390720" },
  { key: "pending", name: "Pending", value: "111000" },
  { key: "paid", name: "Paid", value: "0" },
];

test("decimal-string money renders an empty Recharts pie without coercion", () => {
  assert.equal(pieSum(apiStatusBreakdown, "value"), 0);
});

test("chartNumbers coerces decimal-string money so the pie renders", () => {
  const coerced = chartNumbers(apiStatusBreakdown, ["value"]);
  assert.equal(pieSum(coerced, "value"), 501720);
  assert.deepEqual(
    coerced.map((row) => row.value),
    [390720, 111000, 0]
  );
});

test("chartNumbers preserves labels, keys and extra fields", () => {
  const [first] = chartNumbers(apiStatusBreakdown, ["value"]);
  assert.equal(first.key, "draft");
  assert.equal(first.name, "Draft");
  assert.equal(typeof first.value, "number");
});

test("chartNumbers coerces every requested key", () => {
  const rows = [{ revenue: "10", expenses: "2.5", label: "Jan" }];
  const [row] = chartNumbers(rows, ["revenue", "expenses"]);
  assert.equal(row.revenue, 10);
  assert.equal(row.expenses, 2.5);
  assert.equal(row.label, "Jan");
});

test("chartNumbers defaults missing or unparsable fields to 0", () => {
  const rows = [{ value: undefined }, { value: "not-a-number" }, {}];
  assert.deepEqual(
    chartNumbers(rows, ["value"]).map((row) => row.value),
    [0, 0, 0]
  );
});

test("chartNumbers never throws on non-array input", () => {
  assert.deepEqual(chartNumbers(undefined, ["value"]), []);
  assert.deepEqual(chartNumbers(null, ["value"]), []);
});

test("chartNumbers passes through non-object rows untouched", () => {
  assert.deepEqual(chartNumbers([1, null, "x"], ["value"]), [1, null, "x"]);
});

test("chartNumbers does not mutate its input", () => {
  const rows = [{ value: "5" }];
  chartNumbers(rows, ["value"]);
  assert.equal(rows[0].value, "5");
});
