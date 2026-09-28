# Frontend Hooks and State

## Rules
- TanStack Query for all server state (useQuery, useMutation, useQueryClient)
- Query keys as functions: invoicesKey(params), invoiceKey(id)
- keepPreviousData for list queries (prevents flash on pagination)
- Invalidate related queries on mutation success (invoices, dashboard, clients)
- apiClient from @/api/http.js (centralized axios with interceptors)
- localStorage for user preferences (language, cached AI text)
- No global state library (Context + TanStack Query sufficient)

## Examples

### Query Hook Pattern
```js
// webui/src/hooks/useInvoices.js:4-18
export const invoicesKey = (params) => ["invoices", params || {}];
export const invoiceKey = (id) => ["invoice", id];

export function useInvoices(params) {
  return useQuery({
    queryKey: invoicesKey(params),
    queryFn: () => invoicesApi.list(params),
    placeholderData: keepPreviousData,
  });
}

export function useInvoice(id) {
  return useQuery({
    queryKey: invoiceKey(id),
    queryFn: () => invoicesApi.get(id),
    enabled: !!id,
  });
}
```

### Mutation with Invalidation
```js
// webui/src/hooks/useInvoices.js:20-37
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

export function useUpdateInvoice() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, payload }) => invoicesApi.update(id, payload),
```

### PDF Render with State Management
```jsx
// webui/src/components/invoice/InvoicePdfDownloadContent.jsx:9-45
export default function InvoicePdfDownloadContent({ invoice, settings, lang, publicView, label }) {
  const [state, setState] = useState({ loading: true, url: "", error: false });
  const downloadRef = useRef(null);

  useEffect(() => {
    let cancelled = false;
    let objectUrl = "";
    const timeout = window.setTimeout(() => {
      if (!cancelled) setState({ loading: false, url: "", error: true });
    }, 15000);

    let renderPromise;
    try {
      renderPromise = pdf(createElement(InvoiceDocument, { invoice, settings, lang })).toBlob();
    } catch {
      window.clearTimeout(timeout);
      setState({ loading: false, url: "", error: true });
      return () => window.clearTimeout(timeout);
    }

    renderPromise
      .then((blob) => {
        if (cancelled) return;
        objectUrl = URL.createObjectURL(blob);
        window.clearTimeout(timeout);
        setState({ loading: false, url: objectUrl, error: false });
      })
```

## Anti-patterns

### ❌ Fetching in useEffect
```js
// BAD: manual fetch in useEffect
useEffect(() => {
  fetch("/api/invoices").then(r => r.json()).then(setInvoices);
}, []);

// GOOD: TanStack Query
const { data: invoices } = useInvoices();
```

### ❌ Stale Data After Mutation
```js
// BAD: mutation without invalidation
const mutation = useMutation({
  mutationFn: createInvoice,
});

// GOOD: invalidate related queries
const mutation = useMutation({
  mutationFn: createInvoice,
  onSuccess: () => queryClient.invalidateQueries(["invoices"]),
});
```
