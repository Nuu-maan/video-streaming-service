import { CircleCheck, TriangleAlert } from "lucide-react";

import { Alert, AlertDescription } from "@/components/ui/alert";

/**
 * The form-level result banner: the failure that is not a single field's fault
 * ("incorrect username or password", "slow down"), or the confirmation for a
 * flow that ends where it started (forgot / reset / change password).
 *
 * It mounts on the state change, so its `role="alert"` fires for a screen
 * reader at the moment it appears — which is the only reason the live region
 * works at all. It enters with a short fade and a 4px settle: enough to catch
 * the eye of someone who just pressed a button and is looking at the button,
 * not at the top of the form.
 */
export function FormAlert({ tone, children }: { tone: "error" | "success"; children: React.ReactNode }) {
  const Icon = tone === "error" ? TriangleAlert : CircleCheck;

  return (
    <Alert
      variant={tone === "error" ? "destructive" : "default"}
      className="animate-in fade-in-0 slide-in-from-top-1 shadow-border duration-(--motion-medium) ease-out-quart"
    >
      <Icon aria-hidden />
      <AlertDescription className={tone === "error" ? "text-destructive/90" : "text-foreground"}>
        {children}
      </AlertDescription>
    </Alert>
  );
}
