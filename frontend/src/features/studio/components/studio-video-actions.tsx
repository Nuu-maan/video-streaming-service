"use client";

import { Check, Copy, EllipsisVertical, ExternalLink, Trash2 } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";

import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { routes } from "@/config/routes";
import { deleteVideo } from "@/features/studio/actions";
import type { StudioVideoRow } from "@/features/studio/types";

interface StudioVideoActionsProps {
  video: StudioVideoRow;
}

/**
 * The per-row menu: view, copy link, delete.
 *
 * The dialog is deliberately *not* nested inside the dropdown's item. A menu
 * item closes the menu on click, and a dialog mounted inside a closing menu is
 * unmounted mid-open; so the menu sets state, closes, and the dialog is
 * rendered as its sibling in controlled mode.
 */
export function StudioVideoActions({ video }: StudioVideoActionsProps) {
  const [confirming, setConfirming] = useState(false);
  const [copied, setCopied] = useState(false);

  async function copyLink() {
    const url = `${window.location.origin}${routes.video(video.id)}`;
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      // The check mark is the confirmation; the toast carries the URL, because
      // "copied" without saying what is a promise you cannot verify.
      toast.success("Link copied", { description: url });
      window.setTimeout(() => setCopied(false), 2_000);
    } catch {
      // Clipboard access is denied in some contexts (no permission, insecure
      // origin). Failing silently would look like the copy worked.
      toast.error("Couldn't copy the link", { description: url });
    }
  }

  async function confirmDelete() {
    const result = await deleteVideo(video.id);
    if (result.ok) {
      toast.success("Video deleted", { description: `“${video.title}” and its files are gone.` });
      return;
    }
    toast.error("Couldn't delete that video", { description: result.message });
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="icon-sm" aria-label={`Actions for “${video.title}”`}>
            <EllipsisVertical aria-hidden />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-48">
          <DropdownMenuItem asChild disabled={video.status !== "ready"}>
            <Link href={routes.video(video.id)}>
              <ExternalLink aria-hidden />
              View
            </Link>
          </DropdownMenuItem>
          <DropdownMenuItem onSelect={copyLink}>
            {copied ? <Check aria-hidden /> : <Copy aria-hidden />}
            Copy link
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant="destructive" onSelect={() => setConfirming(true)}>
            <Trash2 aria-hidden />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <ConfirmDialog
        open={confirming}
        onOpenChange={setConfirming}
        title={`Delete “${video.title}”?`}
        description="This removes the video file, every transcoded rendition, and all of its views, likes and comments. It cannot be undone."
        confirmLabel="Delete forever"
        destructive
        onConfirm={confirmDelete}
      />
    </>
  );
}
