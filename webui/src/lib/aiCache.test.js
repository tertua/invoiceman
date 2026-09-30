import { test, beforeEach } from "node:test";
import assert from "node:assert/strict";

// Minimal localStorage stub: node --test runs without a DOM.
class MemoryStorage {
  #map = new Map();
  get length() {
    return this.#map.size;
  }
  key(i) {
    return [...this.#map.keys()][i] ?? null;
  }
  getItem(k) {
    return this.#map.has(k) ? this.#map.get(k) : null;
  }
  setItem(k, v) {
    this.#map.set(k, String(v));
  }
  removeItem(k) {
    this.#map.delete(k);
  }
}
globalThis.localStorage = new MemoryStorage();

const {
  loadAiSummary,
  saveAiSummary,
  loadAiReminder,
  saveAiReminder,
  pruneUnscopedAiCaches,
} = await import("./aiCache.js");

beforeEach(() => {
  for (let i = localStorage.length - 1; i >= 0; i--) localStorage.removeItem(localStorage.key(i));
});

test("summary cache round-trips per user, org and language", () => {
  saveAiSummary("u1", "org-a", "en", "Org A summary");
  assert.equal(loadAiSummary("u1", "org-a", "en"), "Org A summary");
  // A different org is a different cache: no summary yet, not org A's text.
  assert.equal(loadAiSummary("u1", "org-b", "en"), "");
  assert.equal(loadAiSummary("u1", "org-a", "id"), "");
  assert.equal(loadAiSummary("u2", "org-a", "en"), "");
});

test("switching org never surfaces the previous tenant's summary", () => {
  saveAiSummary("u1", "org-a", "en", "Org A summary");
  saveAiSummary("u1", "org-b", "en", "Org B summary");
  assert.equal(loadAiSummary("u1", "org-b", "en"), "Org B summary");
  assert.equal(loadAiSummary("u1", "org-a", "en"), "Org A summary");
});

test("summary without an org is neither read nor written", () => {
  saveAiSummary("u1", undefined, "en", "leaky");
  assert.equal(localStorage.length, 0);
  assert.equal(loadAiSummary("u1", undefined, "en"), "");
});

test("reminder cache is scoped by org, invoice and tone", () => {
  const draft = { subject: "Reminder", body: "Please pay" };
  saveAiReminder("u1", "org-a", "inv-1", "friendly", "en", draft);
  assert.deepEqual(loadAiReminder("u1", "org-a", "inv-1", "friendly", "en"), draft);
  assert.equal(loadAiReminder("u1", "org-b", "inv-1", "friendly", "en"), null);
  assert.equal(loadAiReminder("u1", "org-a", "inv-2", "friendly", "en"), null);
  assert.equal(loadAiReminder("u1", "org-a", "inv-1", "firm", "en"), null);
  assert.equal(loadAiReminder("u1", "org-a", "inv-1", "friendly", "id"), null);
});

test("reminder without an org is neither read nor written", () => {
  saveAiReminder("u1", undefined, "inv-1", "friendly", "en", { subject: "s", body: "b" });
  assert.equal(localStorage.length, 0);
  assert.equal(loadAiReminder("u1", undefined, "inv-1", "friendly", "en"), null);
});

test("prune drops unscoped legacy keys and keeps tenant-scoped ones", () => {
  localStorage.setItem("tupay:ai-summary:u1:en", JSON.stringify({ summary: "legacy" }));
  localStorage.setItem("tupay:ai-summary:anon:en", JSON.stringify({ summary: "legacy anon" }));
  localStorage.setItem("tupay:ai-reminder:u1:inv-1:friendly:en", JSON.stringify({ draft: {} }));
  localStorage.setItem("tupay:ai-reminder:anon:inv-1:friendly:en", JSON.stringify({ draft: {} }));
  saveAiSummary("u1", "org-a", "en", "current");
  saveAiReminder("u1", "org-a", "inv-1", "friendly", "en", { subject: "s", body: "b" });
  localStorage.setItem("arr-lang", "en");

  pruneUnscopedAiCaches();

  assert.equal(localStorage.getItem("tupay:ai-summary:u1:en"), null);
  assert.equal(localStorage.getItem("tupay:ai-summary:anon:en"), null);
  assert.equal(localStorage.getItem("tupay:ai-reminder:u1:inv-1:friendly:en"), null);
  assert.equal(localStorage.getItem("tupay:ai-reminder:anon:inv-1:friendly:en"), null);
  assert.equal(loadAiSummary("u1", "org-a", "en"), "current");
  assert.deepEqual(loadAiReminder("u1", "org-a", "inv-1", "friendly", "en"), { subject: "s", body: "b" });
  assert.equal(localStorage.getItem("arr-lang"), "en", "unrelated keys survive");
});
