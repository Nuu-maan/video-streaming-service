"use client";

import { Trash2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import { clearHistory } from "@/features/history/actions";

/**
 * Behind a confirmation, because it is irreversible and the API has no undo —
 * and the copy names what is destroyed rather than asking "are you sure?", which
 * is a question nobody reads.
 */
export function ClearHistoryButton() {
  const router = useRouter();

  return (
    <ConfirmDialog
      trigger={
        <Button variant="outline" size="sm">
          <Trash2 aria-hidden />
          Clear history
        </Button>
      }
      title="Clear your entire watch history?"
      description="Every video you have watched is removed from this list, and each one loses its resume position — videos you were part-way through will start again from the beginning. This cannot be undone."
      confirmLabel="Clear history"
      destructive
      onConfirm={async () => {
        const result = await clearHistory();
        if (!result.ok) {
          toast.error(result.message);
          return;
        }
        router.refresh();
        toast.success("Watch history cleared.");
      }}
    />
  );
}
