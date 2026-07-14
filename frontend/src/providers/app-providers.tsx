import { Toaster } from "@/components/ui/sonner";
import { ThemeProvider } from "@/providers/theme-provider";

/**
 * Everything the root layout wraps the app in, composed once. Children pass
 * straight through, so Server Components below stay Server Components.
 */
export function AppProviders({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider>
      {children}
      <Toaster position="bottom-right" />
    </ThemeProvider>
  );
}
