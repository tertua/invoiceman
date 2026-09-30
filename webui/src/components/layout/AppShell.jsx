import { useCallback, useEffect, useState } from "react";
import { Outlet, useLocation } from "react-router-dom";
import { AnimatePresence, MotionConfig, motion } from "framer-motion";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";
import { CommandPalette } from "./CommandPalette";
import { useSettings } from "@/hooks/useSettings";
import { useLang } from "@/context/LangContext";
import { setDefaultCurrency } from "@/lib/utils";

export function AppShell() {
  const location = useLocation();
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [currency, setCurrency] = useState("IDR");
  const { data: settings } = useSettings();
  const { lang, setLang } = useLang();

  const openPalette = useCallback(() => setPaletteOpen(true), []);
  const closePalette = useCallback(() => setPaletteOpen(false), []);

  useEffect(() => {
    const nextCurrency = settings?.currency || "IDR";
    setDefaultCurrency(nextCurrency);
    setCurrency(nextCurrency);
    const srv = settings?.language;
    if ((srv === "en" || srv === "id") && srv !== lang) setLang(srv);
  }, [settings?.currency, settings?.language, lang, setLang]);

  useEffect(() => { window.scrollTo({ top: 0, left: 0, behavior: "instant" }); }, [location.pathname]);

  useEffect(() => {
    function onKey(e) {
      const isK = e.key === "k" || e.key === "K";
      if (isK && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setPaletteOpen((v) => !v);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // close on route change
  useEffect(() => { setPaletteOpen(false); }, [location.pathname]);

  return (
    <div className="min-h-screen flex bg-[var(--bg)]">
      <Sidebar />
      <main className="flex-1 px-6 md:px-8 py-6 max-w-[1600px] mx-auto w-full">
        <Topbar onOpenPalette={openPalette} />
        <MotionConfig reducedMotion="user"><AnimatePresence mode="wait">
          <motion.div
            key={`${location.pathname}:${currency}`}
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
          >
            <Outlet />
          </motion.div>
        </AnimatePresence></MotionConfig>
      </main>
      <CommandPalette open={paletteOpen} onClose={closePalette} />
    </div>
  );
}
