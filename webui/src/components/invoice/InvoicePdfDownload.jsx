import { lazy, Suspense, useState } from "react";
import { Download, FileDown, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/Button";

const PdfDownloadContent = lazy(() => import("./InvoicePdfDownloadContent"));

export function InvoicePdfDownload({ invoice, settings, lang, publicView = false, label }) {
  const [active, setActive] = useState(false);
  const fallback = (
    <Button variant="accent" className={publicView ? "w-full" : undefined} disabled>
      <Loader2 size={publicView ? 14 : 15} className="animate-spin" />
      {label}
    </Button>
  );

  if (!active) {
    return (
      <Button
        variant={publicView ? "accent" : "outline"}
        className={publicView ? "w-full" : undefined}
        onClick={() => setActive(true)}
      >
        {publicView ? <FileDown size={14} /> : <Download size={15} />}
        {label}
      </Button>
    );
  }

  return (
    <Suspense fallback={fallback}>
      <PdfDownloadContent
        invoice={invoice}
        settings={settings}
        lang={lang}
        publicView={publicView}
        label={label}
      />
    </Suspense>
  );
}
