// Optimistic-update helpers.
// Shared so the six mutation hooks do not each repeat the
// cancelQueries → getQueryData → setQueriesData boilerplate.

// Cancel in-flight queries matching each key prefix, snapshot every cached
// [queryKey, data] pair under it, and return those pairs so onError can restore
// them verbatim. Prefix-aware so param-filtered keys (["invoices", params]) are
// captured alongside exact keys (["invoice", id]).
export async function snapshotQueries(qc, keys) {
  const snapshots = [];
  for (const key of keys) {
    await qc.cancelQueries({ queryKey: key });
    snapshots.push(...qc.getQueriesData({ queryKey: key }));
  }
  return snapshots;
}

// Restore every pair captured by snapshotQueries. Skip undefined so a key that
// was never cached is left alone rather than seeded with undefined.
export function restoreSnapshots(qc, snapshots) {
  for (const [key, data] of snapshots) {
    if (data !== undefined) qc.setQueryData(key, data);
  }
}

// Patch every cached list matching a key prefix (handles param-filtered keys),
// leaving non-array caches untouched.
export function patchList(qc, key, mapFn) {
  qc.setQueriesData({ queryKey: key }, (old) => (Array.isArray(old) ? mapFn(old) : old));
}

// Remove items matching a predicate from every cached list under a key prefix.
export function removeFromList(qc, key, pred) {
  patchList(qc, key, (list) => list.filter((item) => !pred(item)));
}

// Replace one item in every cached list under a key prefix, keeping its position.
export function replaceInList(qc, key, pred, merge) {
  patchList(qc, key, (list) => list.map((item) => (pred(item) ? { ...item, ...merge } : item)));
}
