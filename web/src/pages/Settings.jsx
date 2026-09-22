import { useEffect, useRef, useState } from "react";
import { Sun, Moon, Check, Upload, Building2, Loader2, Languages } from "lucide-react";
import { PageHeader } from "@/components/layout/PageHeader";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Button } from "@/components/ui/Button";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/Tabs";
import { useAuth } from "@/context/AuthContext";
import { useTheme } from "@/context/ThemeContext";
import { useLang } from "@/context/LangContext";
import { useToast } from "@/context/UIContext";
import { QueryError } from "@/components/ui/QueryError";
import { authApi } from "@/api/auth";
import { useSettings, useUpdateSettings, useUploadLogo } from "@/hooks/useSettings";
import { CURRENCIES, cn } from "@/lib/utils";

function FieldLabel({ children }) {
  return (
    <label className="text-xs font-medium text-[var(--ink-muted)] mb-1.5 block">
      {children}
    </label>
  );
}

function CompanySection() {
  const { t } = useLang();
  const { user } = useAuth();
  const readOnly = user?.role === "moderator";
  const { data: settings, error: settingsError } = useSettings();
  const update = useUpdateSettings();
  const uploadLogo = useUploadLogo();
  const toast = useToast();
  const fileRef = useRef(null);
  const [form, setForm] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (settings && !form) {
      setForm({
        company_name: settings.company_name || "",
        email: settings.email || "",
        phone: settings.phone || "",
        address: settings.address || "",
        logo_url: settings.logo_url || "",
        currency: settings.currency || "IDR",
        tax_rate: Number(settings.tax_rate) || 0,
        invoice_prefix: settings.invoice_prefix || "INV-",
      });
    }
  }, [settings, form]);

  if (settingsError) {
    return <QueryError error={settingsError} />;
  }
  if (!form) {
    return (
      <div className="flex items-center py-16 justify-center text-[var(--ink-muted)]">
        <Loader2 className="animate-spin" size={18} />
      </div>
    );
  }

  const set = (k) => (e) => setForm((f) => ({ ...f, [k]: e.target.value }));

  async function onLogoPick(e) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    if (file.size > 400_000) {
      toast.error(t("settings.logoTooLarge"), t("settings.logoTooLargeDesc"));
      return;
    }
    try {
      const updated = await uploadLogo.mutateAsync(file);
      setForm((f) => ({ ...f, logo_url: updated.logo_url || "" }));
    } catch (err) {
      if (err?.status !== 401) toast.error(t("settings.logoFailed"), err?.message);
    }
  }

  async function onSave(e) {
    e.preventDefault();
    setSaving(true);
    try {
      await update.mutateAsync({ ...form, tax_rate: Number(form.tax_rate) || 0 });
      toast.success(t("settings.companySaved"));
    } catch (err) {
      if (err?.status !== 401) toast.error(t("settings.saveFailed"), err?.message);
    } finally {
      setSaving(false);
    }
  }

  const selectClass =
    "h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15";

  return (
    <form onSubmit={onSave} className="space-y-5 max-w-2xl">
      {readOnly && (
        <div className="rounded-2xl border border-[var(--border)] bg-[var(--surface-2)] px-4 py-3 text-sm text-[var(--ink-muted)]">
          {t("settings.moderatorReadOnly")}
        </div>
      )}
      <fieldset disabled={readOnly} className="space-y-5">
      <Card padding="lg">
        <CardHeader>
          <div>
            <CardTitle className="text-base">{t("settings.companyProfile")}</CardTitle>
            <CardDescription className="mt-1">
              {t("settings.companyProfileDesc")}
            </CardDescription>
          </div>
        </CardHeader>

        <div className="flex items-center gap-4 mb-5">
          <div className="h-16 w-16 rounded-2xl border border-[var(--border)] bg-[var(--surface-2)] flex items-center justify-center overflow-hidden shrink-0">
            {form.logo_url ? (
              <img src={form.logo_url} alt="logo" className="h-full w-full object-contain" />
            ) : (
              <Building2 size={22} className="text-[var(--ink-muted)]" />
            )}
          </div>
          <div>
            <input ref={fileRef} type="file" accept="image/*" className="hidden" onChange={onLogoPick} />
            <Button type="button" variant="outline" size="sm" onClick={() => fileRef.current?.click()}>
              <Upload size={14} /> {t("settings.uploadLogo")}
            </Button>
            {form.logo_url && (
              <button
                type="button"
                onClick={() => setForm((f) => ({ ...f, logo_url: "" }))}
                className="ml-2 text-xs text-[var(--danger)] font-semibold"
              >
                {t("settings.remove")}
              </button>
            )}
            <p className="text-[11px] text-[var(--ink-muted)] mt-1.5">{t("settings.logoHint")}</p>
          </div>
        </div>

        <div className="space-y-4">
          <div>
            <FieldLabel>{t("settings.companyName")}</FieldLabel>
            <Input value={form.company_name} onChange={set("company_name")} placeholder="Your Company LLC" />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <FieldLabel>{t("settings.billingEmail")}</FieldLabel>
              <Input type="email" value={form.email} onChange={set("email")} placeholder="billing@you.com" />
            </div>
            <div>
              <FieldLabel>{t("common.phone")}</FieldLabel>
              <Input value={form.phone} onChange={set("phone")} placeholder="+6281234567890" />
            </div>
          </div>
          <div>
            <FieldLabel>{t("common.address")}</FieldLabel>
            <Input value={form.address} onChange={set("address")} placeholder="123 Main St, City, State" />
          </div>
        </div>
      </Card>

      <Card padding="lg">
        <CardHeader>
          <div>
            <CardTitle className="text-base">{t("settings.defaults")}</CardTitle>
            <CardDescription className="mt-1">
              {t("settings.defaultsDesc")}
            </CardDescription>
          </div>
        </CardHeader>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <FieldLabel>{t("settings.defaultCurrency")}</FieldLabel>
            <select className={selectClass} value={form.currency} onChange={set("currency")}>
              {CURRENCIES.map((c) => (
                <option key={c.code} value={c.code}>
                  {c.code}
                </option>
              ))}
            </select>
          </div>
          <div>
            <FieldLabel>{t("settings.defaultTax")}</FieldLabel>
            <Input type="number" min="0" step="0.1" value={form.tax_rate} onChange={set("tax_rate")} className="tabular" />
          </div>
          <div>
            <FieldLabel>{t("settings.prefix")}</FieldLabel>
            <Input value={form.invoice_prefix} onChange={set("invoice_prefix")} placeholder="INV-" />
          </div>
        </div>
      </Card>
      </fieldset>

      <div className="flex justify-end">
        <Button type="submit" variant="accent" disabled={saving || readOnly}>
          {saving && <Loader2 size={14} className="animate-spin" />}
          {t("settings.saveCompany")}
        </Button>
      </div>
    </form>
  );
}

function ProfileSection() {
  const { user, updateProfile } = useAuth();
  const { t } = useLang();
  const toast = useToast();
  const [name, setName] = useState(user?.name || "");
  const [saving, setSaving] = useState(false);

  const dirty = name.trim() !== (user?.name || "") && name.trim().length > 0;

  async function onSave(e) {
    e.preventDefault();
    if (!dirty) return;
    setSaving(true);
    try {
      await updateProfile({ name: name.trim() });
      toast.success(t("settings.profileUpdated"));
    } catch (err) {
      if (err?.status !== 401) toast.error(t("settings.profileFailed"), err?.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card padding="lg" className="max-w-2xl">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.yourAccount")}</CardTitle>
          <CardDescription className="mt-1">
            {t("settings.accountDesc")}
          </CardDescription>
        </div>
      </CardHeader>

      <form onSubmit={onSave} className="space-y-4">
        <div className="flex items-center gap-4">
          <div className="h-14 w-14 rounded-full bg-[var(--accent-soft)] text-[var(--accent-strong)] font-semibold flex items-center justify-center text-lg ring-2 ring-[var(--surface)] shrink-0">
            {(user?.name?.[0] || "?").toUpperCase()}
          </div>
          <div className="text-xs text-[var(--ink-muted)]">{t("settings.avatarHint")}</div>
        </div>

        <div>
          <FieldLabel>{t("common.name")}</FieldLabel>
          <Input value={name} onChange={(e) => setName(e.target.value)} maxLength={80} placeholder={t("settings.namePlaceholder")} />
        </div>

        <div>
          <FieldLabel>{t("common.email")}</FieldLabel>
          <Input value={user?.email || ""} disabled />
          <p className="text-[11px] text-[var(--ink-muted)] mt-1.5">{t("settings.emailNoChange")}</p>
        </div>

        <div className="flex justify-end pt-2">
          <Button type="submit" disabled={!dirty || saving}>
            {saving ? t("common.saving") : t("common.saveChanges")}
          </Button>
        </div>
      </form>
    </Card>
  );
}

function ThemeOption({ value, label, icon: Icon, current, onSelect, desc }) {
  const active = current === value;
  return (
    <button
      type="button"
      onClick={() => onSelect(value)}
      className={cn(
        "relative flex-1 flex flex-col items-start gap-3 p-4 rounded-2xl border text-left transition-all",
        active
          ? "border-[var(--accent)] bg-[var(--accent-soft)]"
          : "border-[var(--border)] bg-[var(--surface)] hover:bg-[var(--surface-2)]"
      )}
    >
      <div
        className={cn(
          "h-9 w-9 rounded-xl flex items-center justify-center",
          active ? "bg-[var(--accent-strong)] text-white" : "bg-[var(--surface-2)] text-[var(--ink-muted)]"
        )}
      >
        <Icon size={16} />
      </div>
      <div>
        <div className="text-sm font-semibold text-[var(--ink)]">{label}</div>
        {desc && (
          <div className="text-[11px] text-[var(--ink-muted)] mt-0.5">{desc}</div>
        )}
      </div>
      {active && (
        <span className="absolute top-3 right-3 h-5 w-5 rounded-full bg-[var(--accent-strong)] text-white flex items-center justify-center">
          <Check size={12} />
        </span>
      )}
    </button>
  );
}

function AppearanceSection() {
  const { theme, setTheme } = useTheme();
  const { t } = useLang();
  return (
    <Card padding="lg" className="max-w-2xl">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.appearance")}</CardTitle>
          <CardDescription className="mt-1">
            {t("settings.appearanceDesc")}
          </CardDescription>
        </div>
      </CardHeader>

      <div className="flex gap-3">
        <ThemeOption value="light" label={t("settings.light")} icon={Sun} current={theme} onSelect={setTheme} desc={t("settings.lightDesc")} />
        <ThemeOption value="dark" label={t("settings.dark")} icon={Moon} current={theme} onSelect={setTheme} desc={t("settings.darkDesc")} />
      </div>
    </Card>
  );
}

function LanguageSection() {
  const { lang, setLang, t } = useLang();
  return (
    <Card padding="lg" className="max-w-2xl">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.language")}</CardTitle>
          <CardDescription className="mt-1">
            {t("settings.languageDesc")}
          </CardDescription>
        </div>
      </CardHeader>

      <div className="flex gap-3">
        <ThemeOption value="en" label="English" icon={Languages} current={lang} onSelect={setLang} />
        <ThemeOption value="id" label="Bahasa Indonesia" icon={Languages} current={lang} onSelect={setLang} />
      </div>
    </Card>
  );
}

function PasswordSection() {
  const { t } = useLang();
  const toast = useToast();
  const [currentPassword, setCurrent] = useState("");
  const [newPassword, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [saving, setSaving] = useState(false);

  const newTooShort = newPassword.length > 0 && newPassword.length < 8;
  const mismatch = confirm.length > 0 && confirm !== newPassword;
  const canSubmit =
    currentPassword.length > 0 && newPassword.length >= 8 && confirm === newPassword && !saving;

  async function onSubmit(e) {
    e.preventDefault();
    if (!canSubmit) return;
    setSaving(true);
    try {
      await authApi.changePassword({ currentPassword, newPassword });
      toast.success(t("settings.passwordChanged"));
      setCurrent("");
      setNext("");
      setConfirm("");
    } catch (err) {
      if (err?.status !== 401) toast.error(t("settings.passwordFailed"), err?.message);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card padding="lg" className="max-w-2xl">
      <CardHeader>
        <div>
          <CardTitle className="text-base">{t("settings.passwordTitle")}</CardTitle>
          <CardDescription className="mt-1">
            {t("settings.passwordDesc")}
          </CardDescription>
        </div>
      </CardHeader>

      <form onSubmit={onSubmit} className="space-y-4">
        <div>
          <FieldLabel>{t("settings.currentPassword")}</FieldLabel>
          <Input type="password" value={currentPassword} onChange={(e) => setCurrent(e.target.value)} autoComplete="current-password" />
        </div>

        <div>
          <FieldLabel>{t("settings.newPassword")}</FieldLabel>
          <Input type="password" value={newPassword} onChange={(e) => setNext(e.target.value)} autoComplete="new-password" />
          {newTooShort && <p className="text-[11px] text-[var(--danger)] mt-1.5">{t("settings.minLength")}</p>}
        </div>

        <div>
          <FieldLabel>{t("settings.confirmPassword")}</FieldLabel>
          <Input type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" />
          {mismatch && <p className="text-[11px] text-[var(--danger)] mt-1.5">{t("settings.passwordsMismatch")}</p>}
        </div>

        <div className="flex justify-end pt-2">
          <Button type="submit" disabled={!canSubmit}>
            {saving ? t("common.updating") : t("settings.updatePassword")}
          </Button>
        </div>
      </form>
    </Card>
  );
}

export default function Settings() {
  const { t } = useLang();
  const [tab, setTab] = useState("company");

  return (
    <div className="space-y-6">
      <PageHeader title={t("settings.title")} description={t("settings.desc")} />

      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="company">{t("settings.tabCompany")}</TabsTrigger>
          <TabsTrigger value="profile">{t("settings.tabAccount")}</TabsTrigger>
          <TabsTrigger value="appearance">{t("settings.tabAppearance")}</TabsTrigger>
          <TabsTrigger value="password">{t("settings.tabPassword")}</TabsTrigger>
        </TabsList>

        <div className="mt-6">
          <TabsContent value="company">
            <CompanySection />
          </TabsContent>
          <TabsContent value="profile">
            <ProfileSection />
          </TabsContent>
          <TabsContent value="appearance">
            <div className="space-y-5">
              <AppearanceSection />
              <LanguageSection />
            </div>
          </TabsContent>
          <TabsContent value="password">
            <PasswordSection />
          </TabsContent>
        </div>
      </Tabs>
    </div>
  );
}
