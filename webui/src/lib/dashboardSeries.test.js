import { test } from "node:test";
import assert from "node:assert/strict";
import { revenueSpark, monthOverMonthDelta } from "./dashboardSeries.js";

test("revenueSpark coerces decimal strings to numbers", () => {
  const spark = revenueSpark([{ revenue: "100.50" }, { revenue: "200" }]);
  assert.deepEqual(spark, [{ v: 100.5 }, { v: 200 }]);
});

test("revenueSpark caps length at the last 6 points", () => {
  const series = Array.from({ length: 9 }, (_, i) => ({ revenue: String(i) }));
  const spark = revenueSpark(series);
  assert.equal(spark.length, 6);
  assert.deepEqual(spark[0], { v: 3 });
});

test("revenueSpark tolerates missing or invalid input", () => {
  assert.deepEqual(revenueSpark(null), []);
  assert.deepEqual(revenueSpark(undefined), []);
  assert.deepEqual(revenueSpark([{ revenue: "nope" }]), [{ v: 0 }]);
});

test("monthOverMonthDelta reports growth and decline", () => {
  assert.equal(monthOverMonthDelta([{ revenue: "100" }, { revenue: "150" }]), 50);
  assert.equal(monthOverMonthDelta([{ revenue: "200" }, { revenue: "150" }]), -25);
});

test("monthOverMonthDelta rounds to the nearest integer", () => {
  assert.equal(monthOverMonthDelta([{ revenue: "3" }, { revenue: "4" }]), 33);
});

test("monthOverMonthDelta returns null when there is no baseline", () => {
  assert.equal(monthOverMonthDelta([{ revenue: "0" }, { revenue: "150" }]), null);
  assert.equal(monthOverMonthDelta([{ revenue: "150" }]), null);
  assert.equal(monthOverMonthDelta(null), null);
});
