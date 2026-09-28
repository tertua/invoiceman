# Frontend Stack

## Rules
- React 19.3 with Vite 8 for development and build
- TanStack Query for server state management
- Tailwind CSS 4 for styling
- Axios centralized in webui/src/api/http.js (CI enforces no other axios imports)
- @react-pdf/renderer restricted to 2 PDF files (CI enforces)
- recharts restricted to 3 chart components (CI enforces)
- @ alias maps to webui/src for all imports
- i18n structure: user strings in i18n.en.js and i18n.id.js, re-exported via i18n.js

## Examples

### Centralized Axios Client with CSRF
```js
// webui/src/api/http.js:29-56
export const apiClient = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
  headers: { "Content-Type": "application/json" },
});

function csrfToken() {
  try {
    const m = document.cookie.match(/(?:^|; )csrf_token=([^;]*)/);
    return m ? decodeURIComponent(m[1]) : "";
  } catch {
    return "";
  }
}

apiClient.interceptors.request.use((config) => {
  config.headers = config.headers ?? {};
  const token = csrfToken();
  if (token) config.headers["X-CSRF-Token"] = token;
  config.headers["X-Locale"] = appLocale();
  return config;
});

function appLocale() {
  try {
    return localStorage.getItem("arr-lang") === "id" ? "id" : "en";
  } catch {
    return "en";
  }
}
```

### TanStack Query Hook Pattern
```js
// webui/src/hooks/useInvoices.js:7-27
export function useInvoices(params) {
  return useQuery({
    queryKey: invoicesKey(params),
    queryFn: () => invoicesApi.list(params),
    placeholderData: keepPreviousData,
  });
}

function invalidateAll(qc) {
  qc.invalidateQueries({ queryKey: ["invoices"] });
  qc.invalidateQueries({ queryKey: ["dashboard"] });
  qc.invalidateQueries({ queryKey: ["clients"] });
}

export function useCreateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (payload) => invoicesApi.create(payload),
    onSuccess: () => invalidateAll(qc),
  });
}
```

### Restricted Import: React-PDF
```jsx
// webui/src/components/invoice/InvoicePdfDownloadContent.jsx:1-2
import { pdf } from "@react-pdf/renderer";
import { createElement, useEffect, useRef, useState } from "react";
```

## Anti-patterns

### ❌ Axios Outside http.js
```js
// BAD: direct axios import (CI fails check:bundles:strict)
import axios from "axios";
axios.get("/api/invoices");
```

### ❌ User Strings Inline
```jsx
// BAD: hardcoded text
<button>Download Invoice</button>

// GOOD: i18n key
<button>{t(lang, "invoice.download")}</button>
```
