// Dashboard stat helpers — pure functions so node:test can cover the money +
// delta math without a DOM.
//
// Money on the wire is a decimal *string* (models.Money). We coerce with
// Number() for the sparkline/delta **visual only**; the string is still the
// source of truth for every amount we display or send back.

// revenueSpark maps the dashboard revenueSeries to the { v } point shape that
// StatCard's MiniLine/MiniBars read. Missing/invalid revenue degrades to 0.
export function revenueSpark(series) {
  if (!Array.isArray(series)) return [];
  return series.slice(-6).map((point) => ({ v: Number(point?.revenue) || 0 }));
}

// monthOverMonthDelta returns the rounded % change of the last month vs the
// one before it (e.g. -12 for a 12% drop). Returns null when either month has
// no usable revenue, so the badge is simply omitted rather than showing a
// meaningless +100% from a zero base.
export function monthOverMonthDelta(series) {
  if (!Array.isArray(series) || series.length < 2) return null;
  const last = Number(series[series.length - 1]?.revenue) || 0;
  const prev = Number(series[series.length - 2]?.revenue) || 0;
  if (prev === 0) return null;
  return Math.round(((last - prev) / prev) * 100);
}
