import { Link } from "react-router-dom";
import { AlertTriangle, FileX } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { Button } from "@/components/ui/Button";
import { t } from "@/lib/i18n";

export function PublicErrorCard({ lang, status, message }) {
  const notFound = status === 404;
  const Icon = notFound ? FileX : AlertTriangle;

  return (
    <Card padding="lg" className="max-w-md w-full flex flex-col items-center text-center py-12">
      <div className="h-14 w-14 rounded-2xl bg-[var(--accent-soft)] text-[var(--accent-strong)] flex items-center justify-center mb-4">
        <Icon size={26} />
      </div>
      <div className="font-display text-lg font-semibold tracking-tight">
        {notFound ? t(lang, "public.notFoundTitle") : t(lang, "public.errorTitle")}
      </div>
      <p className="text-sm text-[var(--ink-muted)] mt-1.5 max-w-sm">
        {notFound ? t(lang, "public.notFoundDesc") : (message || t(lang, "public.invalidLink"))}
      </p>
      <div className="mt-6">
        <Link to="/">
          <Button variant="outline" size="sm">{t(lang, "public.backHome")}</Button>
        </Link>
      </div>
    </Card>
  );
}
