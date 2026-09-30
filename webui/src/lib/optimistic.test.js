import { test } from "node:test";
import assert from "node:assert/strict";
import { QueryClient } from "@tanstack/react-query";

import { snapshotQueries, restoreSnapshots, patchList, removeFromList, replaceInList } from "./optimistic.js";

test("snapshot + restore reverts a patched list", () => {
  const qc = new QueryClient();
  qc.setQueryData(["items"], [{ id: "a" }, { id: "b" }]);
  return snapshotQueries(qc, [["items"]]).then((snap) => {
    removeFromList(qc, ["items"], (x) => x.id === "a");
    assert.equal(qc.getQueryData(["items"]).length, 1);
    restoreSnapshots(qc, snap);
    assert.equal(qc.getQueryData(["items"]).length, 2);
  });
});

test("restore skips keys that were never cached", () => {
  const qc = new QueryClient();
  restoreSnapshots(qc, [["missing", undefined]]);
  assert.equal(qc.getQueryData(["missing"]), undefined);
});

test("patchList only touches array caches", () => {
  const qc = new QueryClient();
  qc.setQueryData(["x"], { not: "array" });
  patchList(qc, ["x"], (list) => list);
  assert.deepEqual(qc.getQueryData(["x"]), { not: "array" });
});

test("replaceInList keeps the item position when merging", () => {
  const qc = new QueryClient();
  qc.setQueryData(["items"], [{ id: "a", name: "A" }, { id: "b", name: "B" }]);
  replaceInList(qc, ["items"], (x) => x.id === "b", { name: "B2" });
  assert.deepEqual(qc.getQueryData(["items"]), [{ id: "a", name: "A" }, { id: "b", name: "B2" }]);
});

test("setQueriesData prefix-matches every param variant", () => {
  const qc = new QueryClient();
  qc.setQueryData(["invoices", { status: "draft" }], [{ id: "i1", status: "draft" }]);
  qc.setQueryData(["invoices", { status: "sent" }], [{ id: "i2", status: "sent" }]);
  qc.setQueriesData({ queryKey: ["invoices"] }, (old) => old?.map((x) => ({ ...x, patched: true })));
  assert.equal(qc.getQueryData(["invoices", { status: "draft" }])[0].patched, true);
  assert.equal(qc.getQueryData(["invoices", { status: "sent" }])[0].patched, true);
});
