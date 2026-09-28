import { Toaster } from "sonner";
import { useTheme } from "@/context/ThemeContext";

export function AppToaster() {
  const { theme } = useTheme();
  return (
    <Toaster
      theme={theme}
      position="top-center"
      closeButton
      toastOptions={{
        duration: 4200,
        style: {
          background: "var(--surface)",
          border: "1px solid var(--border)",
          color: "var(--ink)",
        },
      }}
    />
  );
}
