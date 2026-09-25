// Stable AR-aging bucket keys returned by GET /api/reports.
export const AGING_BUCKET_KEYS = ["current", "d1_30", "d31_60", "d61_90", "d90_plus"];

// Legacy display labels once sent by the API; mapped so cached responses
// still resolve to a translatable key.
const LEGACY_AGING_LABELS = {
  Current: "current",
  "1-30 days": "d1_30",
  "31-60 days": "d31_60",
  "61-90 days": "d61_90",
  "90+ days": "d90_plus",
};

export function agingBucketKey(point) {
  const raw = point?.key || point?.bucket;
  if (!raw) return "current";
  if (AGING_BUCKET_KEYS.includes(raw)) return raw;
  return LEGACY_AGING_LABELS[raw] || raw;
}

// Replace each point's chart label with its localized text. Must never
// throw on missing/invalid input — fall back to the raw value.
export function localizeAgingBuckets(aging, translate) {
  if (!Array.isArray(aging)) return [];
  return aging.map((a) => {
    if (!a || typeof a !== "object") return a;
    const key = agingBucketKey(a);
    let label = a.bucket;
    try {
      label = translate(`aging.${key}`);
    } catch {
      /* keep raw label */
    }
    return { ...a, key, bucket: label || a.bucket };
  });
}
