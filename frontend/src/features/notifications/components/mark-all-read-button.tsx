"use client";

import { CheckCheck, LoaderCircle } from "lucide-react";
import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import { markAllNotificationsRead } from "@/features/notifications/actions";

export function MarkAllReadButton({ disabled = false }: { disabled?: boolean }) {
  const router = useRouter();
  const [pending, startTransition] = useTransition();

  function handle() {
    startTransition(async () => {
      const result = await markAllNotificationsRead();
      if (!result.ok) {
        toast.error(result.message);
        return;
      }
      router.refresh();
      toast.success(
        result.marked === 0
          ? "Nothing left to read."
          : `Marked ${result.marked} ${result.marked === 1 ? "notification" : "notifications"} read.`,
      );
    });
  }

  return (
    <Button variant="outline" size="sm" disabled={disabled || pending} onClick={handle}>
      {pending ? (
        <LoaderCircle aria-hidden className="animate-spin" />
      ) : (
        <CheckCheck aria-hidden />
      )}
      Mark all read
    </Button>
  );
}
