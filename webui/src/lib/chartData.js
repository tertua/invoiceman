// Recharts draws nothing for a Pie whose dataKey holds a string: its slice
// total is `sum + (isNumber(val) ? val : 0)` and `isNumber` only accepts the
// number type. Money crosses the API as fixed-point decimal strings (backend
// `models.Money`), so pie data must be coerced before rendering. Other chart
// types survive on d3's `+x` coercion, but coercing here is harmless and keeps
// every series consistent. Labels and keys are preserved; only listed numeric
// fields are converted, and missing fields become 0.
export function chartNumbers(rows, keys) {
  if (!Array.isArray(rows)) return [];
  return rows.map((row) => {
    if (!row || typeof row !== "object") return row;
    const next = { ...row };
    for (const key of keys) next[key] = Number(next[key]) || 0;
    return next;
  });
}
