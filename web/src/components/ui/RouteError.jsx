import { AlertTriangle, RotateCcw, ArrowLeft } from "lucide-react";
import { useNavigate, useRouteError } from "react-router-dom";

export default function RouteError() {
  const error = useRouteError();
  const nav = useNavigate();
  const message = error?.message || String(error || "Something went wrong");

  return (
    <div className="min-h-screen flex items-center justify-center p-6 bg-[var(--bg)]">
      <div className="w-full max-w-md text-center">
        <div className="mx-auto h-14 w-14 rounded-3xl bg-[var(--danger)]/12 text-[var(--danger)] flex items-center justify-center mb-5">
          <AlertTriangle size={26} />
        </div>
        <h2 className="font-display text-xl font-semibold tracking-tight text-[var(--ink)]">
          This page hit a snag
        </h2>
        <p className="text-sm text-[var(--ink-muted)] mt-2 break-words">{message}</p>
        <div className="flex items-center justify-center gap-2 mt-6">
          <button type="button"
            onClick={() => nav(-1)}
            className="inline-flex items-center gap-1.5 h-10 px-4 rounded-full text-sm font-semibold border border-[var(--border)] bg-[var(--surface)] text-[var(--ink)]"
          >
            <ArrowLeft size={15} /> Go back
          </button>
          <button type="button"
            onClick={() => window.location.reload()}
            className="inline-flex items-center gap-1.5 h-10 px-4 rounded-full text-sm font-semibold bg-[var(--ink)] text-[var(--bg)]"
          >
            <RotateCcw size={15} /> Reload
          </button>
        </div>
      </div>
    </div>
  );
}
