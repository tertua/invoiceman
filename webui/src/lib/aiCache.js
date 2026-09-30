// AI text caches (business summary, payment reminder drafts) persist in
// localStorage so the last generation survives menu switches and reloads.
//
// Every key carries the active organization as well as the user: the stored
// text describes one tenant's figures, so a key without the org would surface
// the previous tenant's summary after a switch. Keys are therefore only built
// (and only read or written) when the org is known.

const SUMMARY_PREFIX = "tupay:ai-summary:";
const REMINDER_PREFIX = "tupay:ai-reminder:";

// Total ":"-separated segments of a tenant-scoped key, counting the
// "tupay:ai-summary:" prefix as two. Everything shorter predates the org
// segment and is pruned.
//   summary:  tupay | ai-summary: | user | org | lang              = 5
//   reminder: tupay | ai-reminder: | user | org | invoice | tone | lang = 7
const SUMMARY_SEGMENTS = 5;
const REMINDER_SEGMENTS = 7;

function normLang(value) {
  return value === "id" ? "id" : "en";
}

function summaryKey(userId, orgId, language) {
  return `${SUMMARY_PREFIX}${userId || "anon"}:${orgId}:${normLang(language)}`;
}

function reminderKey(userId, orgId, invoiceId, tone, language) {
  return `${REMINDER_PREFIX}${userId || "anon"}:${orgId}:${invoiceId}:${tone}:${normLang(language)}`;
}

function read(key) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : null;
  } catch {
    return null;
  }
}

// write is best-effort: private mode or a full quota must not break generation.
function write(key, value) {
  try {
    localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* storage unavailable — the in-memory value still renders */
  }
}

// loadAiSummary returns the cached summary for this user+org+language, or ""
// when absent or when the org is unknown (an unscoped read could leak).
export function loadAiSummary(userId, orgId, language) {
  if (!orgId) return "";
  const parsed = read(summaryKey(userId, orgId, language));
  return typeof parsed?.summary === "string" ? parsed.summary : "";
}

// saveAiSummary persists the summary; a call without an org is dropped rather
// than written under an ambiguous key.
export function saveAiSummary(userId, orgId, language, summary) {
  if (!orgId) return;
  write(summaryKey(userId, orgId, language), { summary, at: Date.now() });
}

// loadAiReminder returns the cached {subject, body} draft, or null.
export function loadAiReminder(userId, orgId, invoiceId, tone, language) {
  if (!orgId) return null;
  const parsed = read(reminderKey(userId, orgId, invoiceId, tone, language));
  if (typeof parsed?.draft?.subject === "string" && typeof parsed?.draft?.body === "string") {
    return parsed.draft;
  }
  return null;
}

// saveAiReminder persists a generated draft; a call without an org is dropped.
export function saveAiReminder(userId, orgId, invoiceId, tone, language, draft) {
  if (!orgId) return;
  write(reminderKey(userId, orgId, invoiceId, tone, language), { draft, at: Date.now() });
}

// pruneUnscopedAiCaches drops every AI cache that predates tenant scoping
// (user+language only, or the anon variants). They can never be read again
// once keys carry the org, so the space is reclaimed instead of left behind.
export function pruneUnscopedAiCaches() {
  try {
    const doomed = [];
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      const segments = key?.split(":").length ?? 0;
      const unscoped =
        (key?.startsWith(SUMMARY_PREFIX) && segments < SUMMARY_SEGMENTS) ||
        (key?.startsWith(REMINDER_PREFIX) && segments < REMINDER_SEGMENTS);
      if (unscoped) doomed.push(key);
    }
    doomed.forEach((key) => localStorage.removeItem(key));
  } catch {
    /* private mode — nothing to prune */
  }
}
