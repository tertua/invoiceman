import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, Check, Copy, Link2, Loader2, Pencil, Plus, Trash2, UserPlus, Users, X } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardHeader, CardTitle, CardDescription } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { QueryError } from "@/components/ui/QueryError";
import { authApi } from "@/api/auth";
import { orgsApi } from "@/api/orgs";
import { useAuth } from "@/context/AuthContext";
import { useLang } from "@/context/LangContext";
import { orgsKey, useActivateOrg, useOrgInvites, useOrgMembers, useOrgMe } from "@/hooks/useOrgs";
import { formatDate } from "@/lib/utils";

const SELECT_CLASS =
  "h-10 w-full rounded-full border border-[var(--border)] bg-[var(--surface)] px-4 text-sm text-[var(--ink)] outline-none focus:border-[var(--accent)]/50 focus:ring-2 focus:ring-[var(--accent)]/15";

// Invite link URL: the create call returns /invite/<token>, the list only ships the token.
const inviteUrl = (inv) => new URL(inv.url || `/invite/${inv.token}`, window.location.origin).href;

// Revoke only helps while an invite is pending; accepted/revoked/expired are final.
const inviteState = (inv) =>
  inv.revoked_at ? "revoked" : inv.accepted_by ? "accepted" : new Date(inv.expires_at) < new Date() ? "expired" : "pending";

const INVITE_STATE = {
  pending: { key: "org.statePending", tone: "warning" },
  accepted: { key: "org.stateAccepted", tone: "success" },
  revoked: { key: "org.stateRevoked", tone: "danger" },
  expired: { key: "org.stateExpired", tone: "neutral" },
};

// Shared error toast for org mutations: 401 is handled globally by the auth-expired broadcast.
const failToast = (t, key) => (err) => {
  if (err?.status !== 401) toast.error(t(key), { description: err?.message });
};

function SectionHead({ icon: Icon, title, desc }) {
  return (
    <CardHeader>
      <div>
        <CardTitle className="text-base flex items-center gap-2"><Icon size={16} className="text-[var(--accent-strong)]" />{title}</CardTitle>
        {desc && <CardDescription className="mt-1">{desc}</CardDescription>}
      </div>
    </CardHeader>
  );
}

// Switcher rendered only for members of more than one org (POST /orgs/:id/activate).
function ActiveOrgCard({ org, memberships }) {
  const { t } = useLang();
  const activate = useActivateOrg();
  if (memberships.length < 2) return null;
  function onChange(e) {
    const next = memberships.find((m) => m.org_id === e.target.value);
    if (!next || next.org_id === org?.id || !window.confirm(t("org.switchConfirm", { name: next.name }))) return;
    activate.mutate(next.org_id, {
      onSuccess: () => toast.success(t("org.activated", { name: next.name })),
      onError: failToast(t, "org.activateFailed"),
    });
  }
  return (
    <Card padding="lg">
      <SectionHead icon={Building2} title={t("org.activeOrg")} desc={t("org.activeOrgDesc")} />
      <div className="flex items-center gap-3">
        <select className={SELECT_CLASS} value={org?.id || ""} onChange={onChange} disabled={activate.isPending}>
          {memberships.map((m) => <option key={m.org_id} value={m.org_id}>{m.name}</option>)}
        </select>
        <Badge tone="accent">{t("org.activeBadge")}</Badge>
      </div>
    </Card>
  );
}

// Rename form (owner only, PATCH /orgs/:id); staff sees the read-only hint instead.
function RenameCard({ org, isOwner }) {
  const { t } = useLang();
  const qc = useQueryClient();
  const [name, setName] = useState("");
  useEffect(() => { setName(org?.name || ""); }, [org?.name]);
  const rename = useMutation({
    mutationFn: (value) => orgsApi.rename(org.id, { name: value }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: orgsKey("me") });
      qc.invalidateQueries({ queryKey: ["auth", "org"] });
      toast.success(t("org.renamed"));
    },
    onError: failToast(t, "org.renameFailed"),
  });
  const dirty = name.trim().length > 0 && name.trim() !== (org?.name || "");
  return (
    <Card padding="lg">
      <SectionHead icon={Pencil} title={t("org.renameTitle")} desc={t("org.renameDesc")} />
      {!isOwner && <p className="mt-3 text-xs text-[var(--ink-muted)]">{t("org.settingsOwnerHint")}</p>}
      <form onSubmit={(e) => { e.preventDefault(); if (dirty) rename.mutate(name.trim()); }} className="mt-4 flex flex-col gap-3 sm:flex-row">
        <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={t("org.namePlaceholder")} disabled={!isOwner} maxLength={80} className="flex-1" />
        <Button type="submit" variant="accent" disabled={!isOwner || !dirty || rename.isPending}>
          {rename.isPending ? <Loader2 size={14} className="animate-spin" /> : null}{t("org.rename")}
        </Button>
      </form>
      {isOwner && name && !dirty && <p className="mt-2 text-[11px] text-[var(--danger)]">{t("org.renameSame")}</p>}
    </Card>
  );
}

// Member roster (GET /orgs/members): the payload is user_id + role only, so the UUID stands in for a name.
function MembersCard() {
  const { t } = useLang();
  const { user } = useAuth();
  const { data: members = [], isLoading, error } = useOrgMembers();
  const row = (m) => (
    <div key={m.user_id} className="flex items-center justify-between gap-3 py-3">
      <div className="min-w-0">
        <div className="truncate text-sm font-semibold text-[var(--ink)]" title={m.user_id}>{m.user_id === user?.id ? user?.name || t("org.you") : m.user_id}</div>
        <div className="text-[11px] text-[var(--ink-muted)]">{m.role === "owner" ? t("org.roleOwnerDesc") : t("org.roleStaffDesc")}</div>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {m.user_id === user?.id && <Badge tone="ink">{t("org.you")}</Badge>}
        <Badge tone={m.role === "owner" ? "accent" : "neutral"}>{m.role === "owner" ? t("org.roleOwner") : t("org.roleStaff")}</Badge>
      </div>
    </div>
  );
  return (
    <Card padding="lg">
      <SectionHead icon={Users} title={t("org.members")} desc={`${t("org.membersDesc")} · ${t("org.membersCount", { count: members.length })}`} />
      {isLoading && <div className="flex justify-center py-6"><Loader2 size={18} className="animate-spin text-[var(--ink-muted)]" /></div>}
      {!isLoading && <QueryError error={error} />}
      {!isLoading && !error && (members.length
        ? <div className="mt-1 divide-y divide-[var(--border)]">{members.map(row)}</div>
        : <div className="mt-1 py-3 text-sm text-[var(--ink-muted)]">{t("org.empty")}<p className="mt-1 text-xs">{t("org.emptyDesc")}</p></div>)}
    </Card>
  );
}

// Invite links (owner only): mint a link, copy it, then revoke or scan the list.
function InvitesCard() {
  const { t } = useLang();
  const qc = useQueryClient();
  const { data: invites = [], isLoading, error } = useOrgInvites();
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("staff");
  const [link, setLink] = useState("");
  const [copied, setCopied] = useState(false);
  const create = useMutation({
    mutationFn: (payload) => orgsApi.invite(payload),
    onSuccess: (invite) => { setLink(inviteUrl(invite)); setEmail(""); qc.invalidateQueries({ queryKey: orgsKey("invites") }); toast.success(t("org.inviteCreated")); },
    onError: failToast(t, "org.inviteFailed"),
  });
  const revoke = useMutation({
    mutationFn: (inviteId) => orgsApi.revoke(inviteId),
    onSuccess: () => { qc.invalidateQueries({ queryKey: orgsKey("invites") }); toast.success(t("org.inviteRevoked")); },
    onError: failToast(t, "org.inviteRevokeFailed"),
  });
  const emailBad = email.trim() !== "" && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim());
  async function copyLink() {
    try { await navigator.clipboard.writeText(link); setCopied(true); window.setTimeout(() => setCopied(false), 1400); }
    catch { toast.error(t("org.copyFailed")); }
  }
  const inviteRow = (inv) => {
    const state = inviteState(inv);
    const meta = INVITE_STATE[state] || INVITE_STATE.pending;
    return (
      <div key={inv.id} className="flex items-center gap-3 py-3">
        <div className="min-w-0 flex-1">
          <div className="truncate text-xs text-[var(--ink)]" title={inviteUrl(inv)}>{inviteUrl(inv)}</div>
          <div className="mt-0.5 text-[11px] text-[var(--ink-muted)]">{t("org.expiresOn", { date: formatDate(inv.expires_at) })}{inv.email ? ` · ${inv.email}` : ""}</div>
        </div>
        <Badge tone={inv.role === "owner" ? "accent" : "neutral"}>{inv.role === "owner" ? t("org.roleOwner") : t("org.roleStaff")}</Badge>
        <Badge tone={meta.tone}>{t(meta.key)}</Badge>
        <Button type="button" size="iconSm" variant="ghost" onClick={() => { if (window.confirm(t("org.revokeConfirm"))) revoke.mutate(inv.id); }} disabled={revoke.isPending || state !== "pending"} aria-label={t("org.revoke")}>
          <Trash2 size={13} />
        </Button>
      </div>
    );
  };
  return (
    <Card padding="lg">
      <SectionHead icon={UserPlus} title={t("org.invites")} desc={`${t("org.invitesDesc")} ${t("org.inviteTTL")}`} />
      <form onSubmit={(e) => { e.preventDefault(); if (!emailBad) create.mutate(email.trim() ? { email: email.trim(), role } : { role }); }} className="mt-4 flex flex-col gap-3 sm:flex-row">
        <Input value={email} onChange={(e) => setEmail(e.target.value)} placeholder={t("org.inviteEmail")} type="email" className="flex-1" />
        <select aria-label={t("org.role")} value={role} onChange={(e) => setRole(e.target.value)} className={`${SELECT_CLASS} sm:w-36`}>
          <option value="staff">{t("org.roleStaff")}</option>
          <option value="owner">{t("org.roleOwner")}</option>
        </select>
        <Button type="submit" variant="accent" disabled={create.isPending || emailBad}>
          {create.isPending ? <Loader2 size={14} className="animate-spin" /> : <Plus size={14} />}{t("org.createInvite")}
        </Button>
      </form>
      {emailBad && <p className="mt-2 text-[11px] text-[var(--danger)]">{t("org.inviteEmailInvalid")}</p>}
      {link && (
        <div className="mt-4 flex items-center gap-2 rounded-2xl border border-[var(--accent)]/30 bg-[var(--accent-soft)]/30 px-3 py-2">
          <Link2 size={15} className="shrink-0 text-[var(--accent-strong)]" />
          <code className="min-w-0 flex-1 truncate text-xs text-[var(--ink)]" title={link}>{link}</code>
          <Button type="button" size="iconSm" variant="outline" onClick={copyLink} aria-label={t("org.copyLink")}>{copied ? <Check size={13} /> : <Copy size={13} />}</Button>
          <Button type="button" size="iconSm" variant="ghost" onClick={() => setLink("")} aria-label={t("common.done")}><X size={13} /></Button>
        </div>
      )}
      <p className="mt-3 text-xs text-[var(--ink-muted)]">{t("org.inviteLinkHint")}</p>
      {isLoading && <div className="mt-4 flex justify-center py-4"><Loader2 size={18} className="animate-spin text-[var(--ink-muted)]" /></div>}
      {!isLoading && (
        <div className="mt-4">
          <QueryError error={error} />
          {!error && (invites.length
            ? <div className="divide-y divide-[var(--border)]">{invites.map(inviteRow)}</div>
            : <p className="py-3 text-sm text-[var(--ink-muted)]">{t("org.inviteEmpty")}</p>)}
        </div>
      )}
    </Card>
  );
}

export default function OrgTab() {
  const { t } = useLang();
  const { data: org, isLoading, error } = useOrgMe();
  const { data: account } = useQuery({ queryKey: ["auth", "org"], queryFn: () => authApi.me().then((r) => r.org), staleTime: 60_000 });
  if (isLoading) return <div className="flex items-center justify-center py-16 text-[var(--ink-muted)]"><Loader2 size={20} className="animate-spin" /></div>;
  if (error) return <QueryError error={error} />;
  const isOwner = account?.role === "owner";
  return (
    <div className="max-w-2xl space-y-6">
      <Card padding="lg"><p className="text-sm text-[var(--ink-muted)]">{t("org.desc")}</p></Card>
      <ActiveOrgCard org={org} memberships={account?.memberships || []} />
      <RenameCard org={org} isOwner={isOwner} />
      <MembersCard />
      {isOwner && <InvitesCard />}
    </div>
  );
}
