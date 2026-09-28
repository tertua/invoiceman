import { pdf } from "@react-pdf/renderer";
import { createElement, useEffect, useRef, useState } from "react";
import { Download, FileDown, Loader2 } from "lucide-react";
import { buttonVariants } from "@/components/ui/Button";
import { InvoiceDocument } from "@/components/invoice/InvoiceDocument";
import { t } from "@/lib/i18n";

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
      .catch(() => {
        if (cancelled) return;
        window.clearTimeout(timeout);
        setState({ loading: false, url: "", error: true });
      });

    return () => {
      cancelled = true;
      window.clearTimeout(timeout);
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [invoice, settings, lang]);

  useEffect(() => {
    if (!state.url || !downloadRef.current) return;
    downloadRef.current.click();
  }, [state.url]);

  if (state.error) {
    return (
      <span className="inline-flex items-center text-xs text-[var(--danger)]">
        {t(lang, "public.pdfFailed")}
      </span>
    );
  }

  return (
    <a
      ref={downloadRef}
      href={state.url || undefined}
      download={`${invoice.invoice_number}.pdf`}
      aria-disabled={!state.url}
      className={buttonVariants({
        variant: publicView ? "accent" : "outline",
        size: "md",
        className: publicView ? "w-full" : undefined,
      })}
    >
      {state.loading ? <Loader2 size={publicView ? 14 : 15} className="animate-spin" /> : publicView ? <FileDown size={14} /> : <Download size={15} />}
      {state.loading ? t(lang, "public.pdfPreparing") : label}
    </a>
  );
}
