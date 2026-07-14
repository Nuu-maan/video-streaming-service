"use client";

import { Check, Copy, Share2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { IconSwap } from "@/components/common/icon-swap";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";

interface ShareDialogProps {
  url: string;
  title: string;
  /** Unlisted and private videos are shareable only in the sense that the link exists. */
  visibility: "public" | "unlisted" | "private";
}

const COPIED_RESET_MS = 2_000;

/**
 * Share: a link, a copy button, and the OS share sheet where there is one.
 *
 * The copy button confirms in place — a checkmark that swaps for the copy icon —
 * rather than only firing a toast. The toast is for the case where the click
 * lands and the eye is elsewhere; the icon is for the case where it isn't.
 *
 * A private video says so. Handing someone a link that will 404 for them without
 * warning is the kind of small betrayal that makes people stop trusting a share
 * button.
 */
export function ShareDialog({ url, title, visibility }: ShareDialogProps) {
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
      toast.success("Link copied");
      setTimeout(() => setCopied(false), COPIED_RESET_MS);
    } catch {
      toast.error("Couldn't copy the link. Select it and copy manually.");
    }
  };

  const shareNatively = async () => {
    if (!navigator.share) return;
    try {
      await navigator.share({ title, url });
    } catch {
      // A cancelled share sheet throws. That is a decision, not a failure.
    }
  };

  return (
    <Dialog>
      <DialogTrigger asChild>
        <Button variant="secondary" size="sm" className="rounded-full">
          <Share2 aria-hidden />
          Share
        </Button>
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Share this video</DialogTitle>
          <DialogDescription>
            {visibility === "private"
              ? "This video is private. Only you can open this link — make it unlisted or public before sharing it."
              : visibility === "unlisted"
                ? "This video is unlisted. Anyone with the link can watch it, but it won't appear in search."
                : "Anyone with this link can watch."}
          </DialogDescription>
        </DialogHeader>

        <div className="flex items-center gap-2">
          <Input
            readOnly
            value={url}
            aria-label="Video link"
            onFocus={(event) => event.currentTarget.select()}
            className="font-mono text-xs"
          />
          <Button onClick={copy} size="icon" aria-label={copied ? "Link copied" : "Copy link"} className="shrink-0">
            <IconSwap
              active={copied}
              from={<Copy aria-hidden className="size-4" />}
              to={<Check aria-hidden className="size-4" />}
            />
          </Button>
        </div>

        {typeof navigator !== "undefined" && "share" in navigator ? (
          <Button variant="outline" onClick={shareNatively} className="w-full">
            <Share2 aria-hidden />
            Share via…
          </Button>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
