import { PDFDownloadLink } from "@react-pdf/renderer";
import { Download, FileDown, Loader2 } from "lucide-react";
import { buttonVariants } from "@/components/ui/Button";
import { InvoiceDocument } from "@/components/invoice/InvoiceDocument";

export default function InvoicePdfDownloadContent({ invoice, settings, lang, publicView, label }) {
  return (
    <PDFDownloadLink
      document={<InvoiceDocument invoice={invoice} settings={settings} lang={lang} />}
      fileName={`${invoice.invoice_number}.pdf`}
    >
      {({ loading }) => (
        <span
          className={buttonVariants({
            variant: publicView ? "accent" : "outline",
            size: "md",
            className: publicView ? "w-full" : undefined,
          })}
        >
          {loading ? (
            <Loader2 size={publicView ? 14 : 15} className="animate-spin" />
          ) : publicView ? (
            <FileDown size={14} />
          ) : (
            <Download size={15} />
          )}
          {label}
        </span>
      )}
    </PDFDownloadLink>
  );
}
