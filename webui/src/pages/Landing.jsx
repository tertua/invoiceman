import { useEffect } from "react";
import { Link } from "react-router-dom";
import { motion } from "framer-motion";
import {
  ArrowRight,
  ArrowUpRight,
  ScanLine,
  Sparkles,
  BellRing,
  PenLine,
  FileText,
  Users,
  BarChart3,
  ShieldCheck,
  Wallet,
  Receipt,
  CheckCircle2,
  TrendingUp,
  Check,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useLang } from "@/context/LangContext";
import { useAppName, useAllowRegistration } from "@/hooks/useConfig";
import AILogo from "@/components/layout/AILogo";

const TEAL = "var(--accent)";
const TEAL_DARK = "var(--accent-strong)";

export default function Landing() {
  useEffect(() => {
    const prev = document.documentElement.getAttribute("data-theme");
    document.documentElement.setAttribute("data-theme", "light");
    return () => {
      if (prev) document.documentElement.setAttribute("data-theme", prev);
    };
  }, []);

  return (
    <div className="min-h-screen bg-[var(--surface)] text-[var(--ink)] overflow-x-clip antialiased">
      <Nav />
      <Hero />
      <Marquee />
      <AISection />
      <CoreSection />
      <CTASection />
      <Footer />
    </div>
  );
}

/* ─────────────────────────── Nav ─────────────────────────── */
function Nav() {
  const { t } = useLang();
  const appName = useAppName();
  const allowRegistration = useAllowRegistration();
  return (
    <header className="sticky top-0 z-30 backdrop-blur-xl bg-[var(--surface)]/70 border-b border-[var(--border)]">
      <div className="max-w-[1400px] mx-auto px-5 h-16 flex items-center justify-between">
        <div className="flex items-center gap-2.5">
          <AILogo label={appName} />
          <span className="font-display font-semibold text-lg">{appName}</span>
        </div>
        <div className="flex items-center gap-2">
          <Link to="/login" className="h-10 px-4 rounded-full text-sm font-semibold hover:bg-[var(--surface-2)] flex items-center transition-colors">
            {t("landing.nav.signIn")}
          </Link>
          {allowRegistration && (
            <Link
              to="/register"
              className="group h-10 px-5 rounded-full text-sm font-semibold text-white flex items-center gap-1.5 transition-all hover:-translate-y-px"
              style={{ background: "linear-gradient(135deg,var(--accent-hero-2),var(--accent) 50%,var(--accent-strong))", boxShadow: "0 8px 24px -8px color-mix(in srgb, var(--accent) 60%, transparent)" }}
            >
              {t("landing.nav.getStarted")} <ArrowRight size={15} className="group-hover:translate-x-0.5 transition-transform" />
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}

/* ─────────────────────────── Hero ─────────────────────────── */
function Hero() {
  const { t } = useLang();
  const allowRegistration = useAllowRegistration();
  return (
    <section className="relative overflow-hidden">
      <div className="absolute -top-40 -left-40 w-[560px] h-[560px] rounded-full pointer-events-none"
        style={{ background: "radial-gradient(circle, color-mix(in srgb, var(--accent) 22%, transparent), transparent 70%)" }} />
      <div className="absolute top-20 right-0 w-[520px] h-[520px] rounded-full pointer-events-none"
        style={{ background: "radial-gradient(circle, color-mix(in srgb, var(--accent) 16%, transparent), transparent 70%)" }} />

      <div className="relative max-w-[1400px] mx-auto px-6 lg:px-10 grid lg:grid-cols-[1fr_1fr] xl:grid-cols-[1fr_1.35fr] gap-10 items-center pt-16 lg:pt-24 pb-16">
        <motion.div
          initial={{ opacity: 0, y: 18 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: [0.16, 1, 0.3, 1] }}
        >
          <span className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-[var(--surface)]/80 border border-[var(--border)] text-[var(--accent-strong)] text-xs font-semibold shadow-sm">
            <Sparkles size={13} /> {t("landing.hero.badge")}
          </span>
          <h1 className="font-serif text-[clamp(40px,6.4vw,68px)] font-semibold leading-[0.98] tracking-tight mt-6">
            {t("landing.hero.t1")}
            <br />
            <span style={{ background: "linear-gradient(120deg,var(--accent-strong),var(--accent) 55%,var(--accent-hero-2))", WebkitBackgroundClip: "text", backgroundClip: "text", color: "transparent" }}>
              {t("landing.hero.t2")}
            </span>
          </h1>
          <p className="text-lg text-[var(--ink-muted)] mt-6 max-w-lg leading-relaxed">
            {t("landing.hero.desc")}
          </p>
          <div className="flex items-center gap-3 mt-8">
            {allowRegistration && (
              <Link to="/register"
                className="group h-12 px-7 rounded-full text-sm font-semibold text-white flex items-center gap-2 transition-all hover:-translate-y-px"
                style={{ background: "linear-gradient(135deg,var(--accent-hero-2),var(--accent) 50%,var(--accent-strong))" }}>
                {t("landing.hero.startFree")} <ArrowRight size={16} className="group-hover:translate-x-0.5 transition-transform" />
              </Link>
            )}
            <Link to="/login" className="h-12 px-6 rounded-full text-sm font-semibold border border-[var(--border)] bg-[var(--surface)] hover:bg-[var(--surface-2)] flex items-center transition-colors">
              {t("landing.nav.signIn")}
            </Link>
          </div>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-2 mt-8">
            {[t("landing.hero.feat1"), t("landing.hero.feat2"), t("landing.hero.feat3"), t("landing.hero.feat4")].map((f) => (
              <span key={f} className="inline-flex items-center gap-1.5 text-[13px] font-medium text-[var(--ink-muted)]">
                <Check size={14} className="text-[var(--accent)]" /> {f}
              </span>
            ))}
          </div>
        </motion.div>

        <div className="hidden lg:block">
          <InvoiceWall />
        </div>
      </div>
    </section>
  );
}

/* ───────────────── Scrolling invoice wall ───────────────── */
function InvoiceWall() {
  const colA = [<InvoiceCard key="a1" />, <RevenueCard key="a2" />, <PaymentCard key="a3" />];
  const colB = [<ReceiptCard key="b1" />, <ReminderCard key="b2" />, <PaidCard key="b3" />];
  const colC = [<ClientCard key="c1" />, <StatCard2 key="c2" />, <ExpenseCard key="c3" />];

  return (
    <div
      className="relative h-[600px] overflow-hidden"
      style={{
        maskImage: "linear-gradient(to bottom, transparent 0%, black 12%, black 88%, transparent 100%)",
        WebkitMaskImage: "linear-gradient(to bottom, transparent 0%, black 12%, black 88%, transparent 100%)",
      }}
    >
      <div className="absolute inset-0 flex justify-center gap-4">
        <ScrollColumn cards={colA} direction="up" duration={30} />
        <ScrollColumn cards={colB} direction="down" duration={36} />
        <ScrollColumn cards={colC} direction="up" duration={44} className="hidden xl:block" />
      </div>
    </div>
  );
}

function ScrollColumn({ cards, direction, duration, className }) {
  const doubled = [...cards, ...cards];
  const from = direction === "up" ? "0%" : "-50%";
  const to = direction === "up" ? "-50%" : "0%";
  return (
    <div className={cn("w-[228px] shrink-0", className)}>
      <motion.div
        className="flex flex-col gap-3.5"
        animate={{ y: [from, to] }}
        transition={{ duration, repeat: Infinity, ease: "linear" }}
      >
        {doubled.map((c, i) => (
          <div key={i}>{c}</div>
        ))}
      </motion.div>
    </div>
  );
}

function WallCard({ children, className }) {
  return (
    <div className={cn("rounded-3xl bg-[var(--surface)] border border-[var(--border)] shadow-card p-4", className)}>
      {children}
    </div>
  );
}
const WallLabel = ({ children }) => <div className="text-[10px] uppercase tracking-wider text-[var(--ink-muted)] font-semibold">{children}</div>;
const CardFoot = ({ children }) => (
  <div className="mt-3.5 pt-3 border-t border-[var(--border)] flex items-center gap-2">
    <span className="h-4 w-4 rounded-md" style={{ background: `linear-gradient(135deg,${TEAL},${TEAL_DARK})` }} />
    <span className="text-[11px] font-medium text-[var(--ink-muted)]">{children}</span>
  </div>
);

function Pill({ children, tone = "teal" }) {
  const cls = tone === "rose" ? "bg-[var(--danger)]/10 text-[var(--danger)]" : tone === "amber" ? "bg-[var(--warning)]/15 text-[var(--warning)]" : "bg-[var(--accent-soft)] text-[var(--accent-strong)]";
  return <span className={cn("inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold", cls)}>{children}</span>;
}

function InvoiceCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-start justify-between mb-3">
        <div><WallLabel>{t("landing.wall.invoice")}</WallLabel><div className="text-[15px] font-bold text-[var(--ink)] mt-1 tabular-nums">INV-0042</div></div>
        <Pill>{t("landing.wall.sent")}</Pill>
      </div>
      {[["Design sprint", "$3,200"], ["Development · 24h", "$2,280"]].map(([d, a]) => (
        <div key={d} className="flex items-center justify-between text-[12px] py-0.5"><span className="text-[var(--ink-muted)]">{d}</span><span className="text-[var(--ink)] font-semibold tabular-nums">{a}</span></div>
      ))}
      <div className="flex items-center justify-between mt-2.5 pt-2 border-t border-[var(--border)]">
        <span className="text-[10px] uppercase tracking-wide text-[var(--ink-muted)] font-semibold">{t("landing.wall.total")}</span>
        <span className="text-[17px] font-bold tabular-nums" style={{ color: TEAL_DARK }}>$5,480</span>
      </div>
      <CardFoot>Sherly Retail Group</CardFoot>
    </WallCard>
  );
}

function RevenueCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-start justify-between mb-3">
        <div><WallLabel>{t("landing.wall.revenue")}</WallLabel><div className="text-[26px] font-bold text-[var(--ink)] mt-1 tabular-nums">$311K</div></div>
        <Pill><TrendingUp size={10} strokeWidth={2.5} /> +12%</Pill>
      </div>
      <div className="flex items-end gap-1.5 h-12">
        {[42, 58, 50, 72, 63, 88].map((h, i) => (
          <div key={i} className="flex-1 rounded-t-md" style={{ height: `${h}%`, background: `linear-gradient(180deg,var(--accent-hero-2),${TEAL_DARK})`, opacity: 0.45 + i * 0.09 }} />
        ))}
      </div>
      <CardFoot>{t("landing.wall.last6")}</CardFoot>
    </WallCard>
  );
}

function ReceiptCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-start justify-between mb-2.5"><WallLabel>{t("landing.wall.scan")}</WallLabel><Pill><ScanLine size={10} strokeWidth={2.5} /> {t("landing.wall.parsed")}</Pill></div>
      <div className="rounded-xl p-3 bg-[var(--accent-soft)]">
        <div className="text-[9px] uppercase tracking-wide font-semibold mb-1 text-[var(--accent-strong)]">{t("landing.wall.extracted")}</div>
        <div className="text-[13px] font-semibold text-[var(--ink)]">Adobe Inc.</div>
        <div className="flex items-center justify-between text-[12px] text-[var(--ink-muted)] mt-1"><span>Creative Cloud ×1</span><span className="tabular-nums font-bold text-[var(--ink)]">$54.99</span></div>
      </div>
      <CardFoot>{t("landing.wall.imageToInvoice")}</CardFoot>
    </WallCard>
  );
}

function PaymentCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-center gap-2.5">
        <div className="h-9 w-9 rounded-xl flex items-center justify-center bg-[var(--success)]/12"><CheckCircle2 size={17} className="text-[var(--success)]" /></div>
        <div><WallLabel>{t("landing.wall.paymentReceived")}</WallLabel><div className="text-[17px] font-bold text-[var(--ink)] tabular-nums">$7,595.00</div></div>
      </div>
      <div className="flex items-center justify-between text-[11px] text-[var(--ink-muted)] mt-3"><span>INV-0038 · Harbor & Co.</span><span>{t("landing.wall.bankTransfer")}</span></div>
    </WallCard>
  );
}

function ReminderCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-start justify-between mb-2.5"><WallLabel>{t("landing.wall.reminder")}</WallLabel><Pill><Sparkles size={10} strokeWidth={2.5} /> {t("landing.wall.drafted")}</Pill></div>
      <div className="rounded-xl bg-[var(--surface-2)] border border-[var(--border)] p-3">
        <div className="flex items-center gap-1.5 mb-1"><BellRing size={12} style={{ color: TEAL_DARK }} /><span className="text-[12px] font-semibold text-[var(--ink)]">{t("landing.wall.friendlyNudge")}</span></div>
        <p className="text-[11.5px] text-[var(--ink-muted)] leading-snug">{t("landing.wall.reminderQuote")}</p>
      </div>
      <CardFoot>{t("landing.wall.oneClick")}</CardFoot>
    </WallCard>
  );
}

function PaidCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-start justify-between mb-3"><WallLabel>{t("landing.wall.paidMonth")}</WallLabel><Pill><Check size={10} strokeWidth={3} /> {t("landing.wall.onTrack")}</Pill></div>
      <div className="text-[28px] font-bold text-[var(--ink)] tabular-nums">$42,180</div>
      <div className="flex items-center gap-1 mt-2.5">
        {Array.from({ length: 8 }).map((_, i) => <span key={i} className="h-2 flex-1 rounded-full" style={{ background: i < 6 ? TEAL : "var(--surface-2)" }} />)}
      </div>
      <CardFoot>{t("landing.wall.invoicesPaid", { a: "6", b: "8" })}</CardFoot>
    </WallCard>
  );
}

function ClientCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-center gap-2.5 mb-3">
        <div className="h-9 w-9 rounded-full flex items-center justify-center text-white font-bold text-sm" style={{ background: `linear-gradient(135deg,${TEAL},${TEAL_DARK})` }}>B</div>
        <div><div className="text-[13px] font-semibold text-[var(--ink)]">Brightline Studios</div><div className="text-[11px] text-[var(--ink-muted)]">New York, NY</div></div>
      </div>
      <div className="grid grid-cols-2 gap-2">
        <div><div className="text-[9px] uppercase tracking-wide text-[var(--ink-muted)] font-semibold">{t("landing.wall.billed")}</div><div className="text-[13px] font-bold text-[var(--ink)] tabular-nums">$18.4K</div></div>
        <div><div className="text-[9px] uppercase tracking-wide text-[var(--ink-muted)] font-semibold">{t("landing.wall.owed")}</div><div className="text-[13px] font-bold tabular-nums text-[var(--warning)]">$3.8K</div></div>
      </div>
    </WallCard>
  );
}

function StatCard2() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-center gap-2.5">
        <div className="h-9 w-9 rounded-xl flex items-center justify-center bg-[var(--accent-soft)]"><Wallet size={16} className="text-[var(--accent-strong)]" /></div>
        <div><WallLabel>{t("landing.wall.outstanding")}</WallLabel><div className="text-[17px] font-bold text-[var(--ink)] tabular-nums">$23,760</div></div>
      </div>
      <div className="flex items-center justify-between text-[11px] mt-3"><span className="text-[var(--ink-muted)]">{t("landing.wall.openInvoices", { n: "12" })}</span><Pill tone="rose">{t("landing.wall.overdue", { n: "3" })}</Pill></div>
    </WallCard>
  );
}

function ExpenseCard() {
  const { t } = useLang();
  return (
    <WallCard>
      <div className="flex items-center gap-2.5">
        <div className="h-9 w-9 rounded-xl flex items-center justify-center bg-[var(--warning)]/15"><Receipt size={16} className="text-[var(--warning)]" /></div>
        <div><WallLabel>{t("landing.wall.expense")}</WallLabel><div className="text-[14px] font-bold text-[var(--ink)]">AWS · Hosting</div></div>
      </div>
      <div className="flex items-center justify-between text-[12px] mt-3"><span className="text-[var(--ink-muted)]">Jul 2026 · card ****3140</span><span className="tabular-nums font-bold text-[var(--ink)]">$128.40</span></div>
    </WallCard>
  );
}

/* ───────────────── Marquee strip ───────────────── */
function Marquee() {
  const { t } = useLang();
  const items = t("landing.marquee.items");
  const doubled = [...items, ...items];
  return (
    <div className="border-y border-[var(--border)] bg-[var(--surface-2)] py-4 overflow-hidden">
      <motion.div className="flex gap-3 w-max" animate={{ x: ["0%", "-50%"] }} transition={{ duration: 28, repeat: Infinity, ease: "linear" }}>
        {doubled.map((t, i) => (
          <span key={i} className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full bg-[var(--surface)] border border-[var(--border)] text-sm font-medium text-[var(--ink-muted)] shadow-card whitespace-nowrap">
            <span className="h-1.5 w-1.5 rounded-full" style={{ background: TEAL }} /> {t}
          </span>
        ))}
      </motion.div>
    </div>
  );
}

/* ───────────────── AI section ───────────────── */
function AISection() {
  const { t } = useLang();
  const AI_FEATURES = [
    { icon: ScanLine, title: t("landing.ai.f1.title"), desc: t("landing.ai.f1.desc") },
    { icon: Sparkles, title: t("landing.ai.f2.title"), desc: t("landing.ai.f2.desc") },
    { icon: BellRing, title: t("landing.ai.f3.title"), desc: t("landing.ai.f3.desc") },
    { icon: PenLine, title: t("landing.ai.f4.title"), desc: t("landing.ai.f4.desc") },
  ];

  return (
    <section className="max-w-[1400px] mx-auto px-5 py-24">
      <SectionHead eyebrow={t("landing.ai.eyebrow")} title={t("landing.ai.title")} sub={t("landing.ai.sub")} />
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5 mt-14">
        {AI_FEATURES.map((f, i) => (
          <motion.div key={f.title}
            initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true }}
            transition={{ duration: 0.5, delay: i * 0.08 }}
            className="group relative p-6 rounded-3xl bg-[var(--surface)] border border-[var(--border)] shadow-card hover:shadow-hover hover:-translate-y-1.5 transition-all duration-300 overflow-hidden"
          >
            <div className="absolute -top-8 -right-8 w-24 h-24 rounded-full opacity-0 group-hover:opacity-100 transition-opacity" style={{ background: "radial-gradient(circle, color-mix(in srgb, var(--accent) 18%, transparent), transparent 70%)" }} />
            <div className="relative h-12 w-12 rounded-2xl flex items-center justify-center text-white" style={{ background: `linear-gradient(135deg,var(--accent),${TEAL_DARK})`, boxShadow: "0 8px 20px -6px color-mix(in srgb, var(--accent) 60%, transparent)" }}>
              <f.icon size={22} />
            </div>
            <div className="text-[15px] font-bold text-[var(--ink)] mt-5">{f.title}</div>
            <p className="text-sm text-[var(--ink-muted)] mt-2 leading-relaxed">{f.desc}</p>
          </motion.div>
        ))}
      </div>
    </section>
  );
}

/* ───────────────── Core section ───────────────── */
function CoreSection() {
  const { t } = useLang();
  const CORE = [
    { icon: FileText, title: t("landing.core.f1.title"), desc: t("landing.core.f1.desc") },
    { icon: Users, title: t("landing.core.f2.title"), desc: t("landing.core.f2.desc") },
    { icon: Wallet, title: t("landing.core.f3.title"), desc: t("landing.core.f3.desc") },
    { icon: BarChart3, title: t("landing.core.f4.title"), desc: t("landing.core.f4.desc") },
    { icon: Receipt, title: t("landing.core.f5.title"), desc: t("landing.core.f5.desc") },
    { icon: ShieldCheck, title: t("landing.core.f6.title"), desc: t("landing.core.f6.desc") },
  ];

  return (
    <section className="relative py-24 overflow-hidden">
      <div className="absolute inset-0 bg-[var(--surface-2)] border-y border-[var(--border)]" />
      <div className="relative max-w-[1400px] mx-auto px-5">
        <SectionHead eyebrow={t("landing.core.eyebrow")} title={t("landing.core.title")} sub={t("landing.core.sub")} />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5 mt-14">
          {CORE.map((f, i) => (
            <motion.div key={f.title}
              initial={{ opacity: 0, y: 20 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true }}
              transition={{ duration: 0.45, delay: (i % 3) * 0.08 }}
              className="group p-6 rounded-3xl bg-[var(--surface)] border border-[var(--border)] hover:border-[var(--accent)]/30 shadow-card hover:shadow-hover transition-all duration-300"
            >
              <div className="h-11 w-11 rounded-2xl flex items-center justify-center bg-[var(--accent-soft)] text-[var(--accent-strong)] group-hover:scale-110 transition-transform">
                <f.icon size={20} />
              </div>
              <div className="text-[15px] font-bold text-[var(--ink)] mt-4 flex items-center gap-1.5">
                {f.title}
                <ArrowUpRight size={15} className="text-[var(--ink-muted)]/40 group-hover:text-[var(--accent)] group-hover:translate-x-0.5 group-hover:-translate-y-0.5 transition-all" />
              </div>
              <p className="text-sm text-[var(--ink-muted)] mt-2 leading-relaxed">{f.desc}</p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ───────────────── CTA ───────────────── */
function CTASection() {
  const { t } = useLang();
  const allowRegistration = useAllowRegistration();
  const ctaTo = allowRegistration ? "/register" : "/login";
  return (
    <section className="max-w-[1400px] mx-auto px-5 py-24">
      <div className="relative rounded-3xl px-8 py-24 text-center text-white overflow-hidden"
        style={{ background: "linear-gradient(135deg,var(--accent-strong) 0%,var(--accent) 45%,var(--accent-hero) 100%)", boxShadow: "0 40px 80px -30px color-mix(in srgb, var(--accent-strong) 50%, transparent)" }}>
        <div className="absolute -top-28 -right-24 w-96 h-96 rounded-full" style={{ background: "radial-gradient(circle, color-mix(in srgb, white 40%, transparent), transparent 70%)", opacity: 0.25 }} />
        <div className="absolute -bottom-32 -left-24 w-96 h-96 rounded-full" style={{ background: "radial-gradient(circle, color-mix(in srgb, white 40%, transparent), transparent 70%)", opacity: 0.2 }} />

        <div className="hidden xl:block absolute left-8 top-0 bottom-0 w-[200px] py-6">
          <CtaColumn direction="up" duration={26} />
        </div>
        <div className="hidden xl:block absolute right-8 top-0 bottom-0 w-[200px] py-6">
          <CtaColumn direction="down" duration={30} />
        </div>

        <div className="relative z-10">
          <span className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-white/10 border border-white/15 text-white/90 text-xs font-semibold backdrop-blur-md">
            <Sparkles size={13} /> {t("landing.cta.badge")}
          </span>
          <h2 className="font-serif text-[clamp(30px,4.5vw,48px)] font-semibold tracking-tight mt-6">
            {t("landing.cta.title")}
          </h2>
          <p className="text-white/75 mt-4 max-w-md mx-auto text-lg">
            {t("landing.cta.sub")}
          </p>
          <Link to={ctaTo}
            className="group inline-flex items-center gap-2 mt-9 h-13 px-8 py-4 rounded-full bg-[var(--surface)] text-[var(--accent-strong)] text-sm font-bold transition-all hover:-translate-y-px hover:shadow-hover">
            {t("landing.cta.button")} <ArrowRight size={16} className="group-hover:translate-x-0.5 transition-transform" />
          </Link>
        </div>
      </div>
    </section>
  );
}

function CtaColumn({ direction, duration }) {
  const cards = [<CtaInv key="1" />, <CtaPay key="2" />, <CtaReceipt key="3" />, <CtaRevenue key="4" />];
  const doubled = [...cards, ...cards];
  const from = direction === "up" ? "0%" : "-50%";
  const to = direction === "up" ? "-50%" : "0%";
  const fade = "linear-gradient(to bottom, transparent 0%, black 16%, black 84%, transparent 100%)";
  return (
    <div className="h-full" style={{ maskImage: fade, WebkitMaskImage: fade }}>
      <motion.div className="flex flex-col gap-3" animate={{ y: [from, to] }} transition={{ duration, repeat: Infinity, ease: "linear" }}>
        {doubled.map((c, i) => <div key={i}>{c}</div>)}
      </motion.div>
    </div>
  );
}

function CtaCard({ children }) {
  return <div className="rounded-2xl bg-white/10 border border-white/15 backdrop-blur-md p-3.5 text-white" style={{ boxShadow: "0 10px 30px -12px rgba(0,0,0,0.4)" }}>{children}</div>;
}
function CtaInv() {
  const { t } = useLang();
  return (
    <CtaCard>
      <div className="flex items-center justify-between">
        <span className="text-[12px] font-bold tabular-nums text-white/90">INV-0042</span>
        <span className="text-[9px] font-semibold px-1.5 py-0.5 rounded-full bg-white/20">{t("landing.ctaCard.paid")}</span>
      </div>
      <div className="text-[17px] font-bold tabular-nums mt-1">$5,480</div>
    </CtaCard>
  );
}
function CtaPay() {
  const { t } = useLang();
  return (
    <CtaCard>
      <div className="text-[9px] uppercase tracking-wide text-white/60 font-semibold">{t("landing.ctaCard.paymentReceived")}</div>
      <div className="text-[16px] font-bold tabular-nums mt-0.5">$7,595.00</div>
    </CtaCard>
  );
}
function CtaReceipt() {
  const { t } = useLang();
  return (
    <CtaCard>
      <div className="text-[9px] uppercase tracking-wide text-white/60 font-semibold">{t("landing.ctaCard.aiParsed")}</div>
      <div className="flex items-center justify-between mt-1"><span className="text-[12px] font-semibold">Adobe Inc.</span><span className="text-[13px] font-bold tabular-nums">$54.99</span></div>
    </CtaCard>
  );
}
function CtaRevenue() {
  const { t } = useLang();
  return (
    <CtaCard>
      <div className="text-[9px] uppercase tracking-wide text-white/60 font-semibold">{t("landing.ctaCard.revenue")}</div>
      <div className="text-[19px] font-bold tabular-nums mt-0.5">$311K</div>
    </CtaCard>
  );
}

/* ───────────────── Footer ───────────────── */
function Footer() {
  const { t } = useLang();
  const appName = useAppName();
  const allowRegistration = useAllowRegistration();
  const year = new Date().getFullYear();
  return (
    <footer className="border-t border-[var(--border)]">
      <div className="max-w-[1400px] mx-auto px-5 py-10 flex flex-col sm:flex-row items-center justify-between gap-4">
        <div className="flex items-center gap-2.5">
          <AILogo label={appName} />
          <span className="font-display font-semibold">{appName}</span>
        </div>
        <span className="text-sm text-[var(--ink-muted)]">{t("landing.footer.copyright", { year, app: appName })}</span>
        <div className="flex items-center gap-3">
          <Link to="/login" className="text-sm font-semibold text-[var(--accent-strong)] hover:underline">{t("landing.footer.signIn")}</Link>
          {allowRegistration && (
            <Link to="/register" className="text-sm font-semibold text-[var(--accent-strong)] hover:underline">{t("landing.footer.getStarted")}</Link>
          )}
        </div>
      </div>
    </footer>
  );
}

/* ───────────────── shared ───────────────── */
function SectionHead({ eyebrow, title, sub }) {
  return (
    <div className="text-center max-w-2xl mx-auto">
      <span className="inline-block text-[11px] font-bold uppercase tracking-[0.18em] text-[var(--accent)] mb-3">{eyebrow}</span>
      <h2 className="font-serif text-[clamp(28px,4vw,42px)] font-semibold tracking-tight text-[var(--ink)]">{title}</h2>
      <p className="text-[var(--ink-muted)] mt-3 text-lg">{sub}</p>
    </div>
  );
}