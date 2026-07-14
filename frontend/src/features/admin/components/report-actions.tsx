"use client";

import { Ban, Check, MessageSquareWarning, Trash2 } from "lucide-react";
import { useId, useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { reviewReport } from "@/features/admin/actions";
import { reviewActions } from "@/features/admin/schemas";
import type { ReviewAction } from "@/features/admin/types";

interface ReportActionsProps {
  reportId: string;
  /** Whether the report actually points at a video. Without one there is nothing to delete. */
  hasVideo: boolean;
}

const ICONS: Record<ReviewAction, typeof Check> = {
  dismiss: Check,
  warn_user: MessageSquareWarning,
  delete_video: Trash2,
  ban_user: Ban,
};

/**
 * The four ways a report ends.
 *
 * Every one of them routes through ConfirmDialog — including the two that are
 * not destructive — because all four are terminal: the report leaves the queue
 * and there is no undo button anywhere in this API. The dialog names the exact
 * consequence rather than asking "are you sure?", which is a question nobody has
 * ever answered by thinking.
 *
 * `delete_video` is hidden, not disabled, when the report has no video attached:
 * a report against a comment or an account has nothing for it to delete, and a
 * greyed-out button invites the reader to wonder what they did wrong.
 *
 * The notes field lives out here rather than inside the dialog because it
 * applies to whichever action is chosen — it is the moderator's reasoning, and
 * it gets recorded against the review no matter which of the four they land on.
 */
export function ReportActions({ reportId, hasVideo }: ReportActionsProps) {
  const notesId = useId();
  const [notes, setNotes] = useState("");
  const [pending, setPending] = useState<ReviewAction | null>(null);

  const available = reviewActions.filter(
    (action) => action.value !== "delete_video" || hasVideo,
  );
  const active = available.find((action) => action.value === pending) ?? null;

  async function confirm(action: ReviewAction) {
    const result = await reviewReport(reportId, action, notes);
    if (result.ok) {
      toast.success(result.message);
      return;
    }
    // A refused ban is the case worth getting right: the API 403s a moderator
    // who lacks `manage_users`, and the action translates that into "you can't
    // ban users" rather than a generic failure the reader would retry forever.
    toast.error("Couldn't review that report", { description: result.message });
  }

  return (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <Label htmlFor={notesId} className="text-xs text-muted-foreground">
          Notes (optional — recorded with your decision)
        </Label>
        <Textarea
          id={notesId}
          value={notes}
          onChange={(event) => setNotes(event.target.value)}
          rows={2}
          placeholder="Why you're taking this action."
          className="resize-none text-sm"
        />
      </div>

      <div className="flex flex-wrap gap-2">
        {available.map((action) => {
          const Icon = ICONS[action.value];
          return (
            <Button
              key={action.value}
              variant={action.destructive ? "destructive" : "outline"}
              size="sm"
              onClick={() => setPending(action.value)}
            >
              <Icon aria-hidden data-icon="inline-start" />
              {action.label}
            </Button>
          );
        })}
      </div>

      <ConfirmDialog
        open={active !== null}
        onOpenChange={(open) => (open ? undefined : setPending(null))}
        title={active ? `${active.label}?` : ""}
        description={active?.consequence}
        confirmLabel={active?.label ?? "Confirm"}
        destructive={active?.destructive}
        onConfirm={async () => {
          if (active) await confirm(active.value);
        }}
      />
    </div>
  );
}
