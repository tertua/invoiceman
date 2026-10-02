// Stroke-based empty-state illustrations. Inline SVG (no assets, no network),
// drawn with `currentColor` so the caller tints them with a token class. Kept
// tiny on purpose: they land in the entry chunk (see check:bundles).
//
// Dev path — empty-state art
//   [done]  four small line illustrations for the app's empty lists
//   [later] a shared <Illustration> wrapper if the set grows past a screen
// Seam: every component takes { className, size } and renders nothing else;
// swap the art without touching EmptyState or the pages.
// End dev path

function Frame({ children, size = 120 }) {
  return (
    <svg
      viewBox="0 0 120 96"
      width={size}
      height={(size * 96) / 120}
      fill="none"
      stroke="currentColor"
      strokeWidth="2.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

export function WelcomeIllo(props) {
  return (
    <Frame {...props}>
      <rect x="34" y="18" width="42" height="56" rx="5" />
      <path d="M42 32h26M42 42h26M42 52h16" />
      <circle cx="82" cy="62" r="14" />
      <path d="M82 56v12M76 62h12" />
    </Frame>
  );
}

export function InvoicesIllo(props) {
  return (
    <Frame {...props}>
      <rect x="26" y="24" width="44" height="54" rx="5" />
      <path d="M34 38h28M34 48h28M34 58h18" />
      <path d="M62 16h30l8 8v54a4 4 0 0 1-4 4H62" />
    </Frame>
  );
}

export function ClientsIllo(props) {
  return (
    <Frame {...props}>
      <circle cx="44" cy="40" r="12" />
      <path d="M24 78c0-11 9-20 20-20s20 9 20 20" />
      <circle cx="84" cy="34" r="9" />
      <path d="M70 62c3-5 8-8 14-8 8 0 14 6 14 14" />
    </Frame>
  );
}

export function ExpensesIllo(props) {
  return (
    <Frame {...props}>
      <path d="M40 14h40v68l-5-4-5 4-5-4-5 4-5-4-5 4-5-4-5 4z" />
      <path d="M50 34h20M50 44h20M50 54h12" />
    </Frame>
  );
}

export const EMPTY_ILLO = {
  welcome: WelcomeIllo,
  invoices: InvoicesIllo,
  clients: ClientsIllo,
  expenses: ExpensesIllo,
};
