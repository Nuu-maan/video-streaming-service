"use client";

import { LogOut } from "lucide-react";
import { useTransition } from "react";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import { logoutAll } from "@/features/auth/actions";

/**
 * Revokes every token the account has, on every device, and redirects to the
 * sign-in page. Genuinely irreversible for the other sessions, so it is behind
 * a confirmation — one of the few places that earns one.
 */
export function SignOutEverywhere() {
  const [, startTransition] = useTransition();

  return (
    <ConfirmDialog
      title="Sign out everywhere?"
      description="Every device signed in to this account will be signed out, including this one."
      confirmLabel="Sign out everywhere"
      destructive
      onConfirm={() =>
        new Promise<void>((resolve) => {
          startTransition(async () => {
            // logoutAll redirects, so this promise resolves only if it somehow does not.
            await logoutAll();
            resolve();
          });
        })
      }
      trigger={
        <Button variant="outline">
          <LogOut aria-hidden />
          Sign out everywhere
        </Button>
      }
    />
  );
}
