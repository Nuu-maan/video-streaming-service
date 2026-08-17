"use client";

import { Check, Copy, Globe, Link2, Lock, Share2 } from "lucide-react";
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
import type { Video } from "@/types/common";

interface ShareDialogProps {
  url: string;
  title: string;
  visibility: Video["visibility"];
}

const COPIED_RESET_MS = 2_000;

/**
 * Who can open this, and the link to give them.
 *
 * Share is a primary action here rather than an afterthought icon: this is a
 * workspace, so handing a video to a specific person is the point of putting it
 * in one, and burying that behind an overflow menu would hide the product's
 * whole reason for having three visibility states.
 *
 * The visibility is stated plainly, and stated FIRST, because it is the fact
 * that decides whether the link below will work for the person you send it to.
 * A private video says so before you copy anything — handing someone a link that
 * 404s for them without warning is the kind of small betrayal that makes people
 * stop trusting a share button.
 *
 * It reports; it does not change. The API has no endpoint for editing a video
 * after upload — no PATCH /videos/:id exists — so visibility is fixed at the
 * moment of upload. A control here would be a lie. See the note in the upload
 * form, which is where the decision actually gets made.
 */
const VISIBILITY = {
  public: {
    icon: Globe,
    label: "Public",
    detail: "Anyone with the link can watch, and it can be found by search.",
  },
  unlisted: {
    icon: Link2,
    label: "Unlisted",
    detail: "Anyone with the link can watch. It stays out of search and listings.",
  },
  private: {
    icon: Lock,
    label: "Private",
    detail: "Only you can open this. Nobody you send the link to will be able to watch it.",
  },
} as const;

export function ShareDialog({ url, title, visibility }: ShareDialogProps) {
  const [copied, setCopied] = useState(false);

  const state = VISIBILITY[visibility] ?? VISIBILITY.private;
  const Icon = state.icon;

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
        <Button size="sm" className="rounded-full">
          <Share2 aria-hidden />
          Share
        </Button>
      </DialogTrigger>

      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Share this video</DialogTitle>
          <DialogDescription>Check who can open it before you send the link.</DialogDescription>
        </DialogHeader>

        <div className="flex items-start gap-3 rounded-lg bg-muted/60 p-3 ring-1 ring-border/60 ring-inset">
          <Icon aria-hidden className="mt-0.5 size-4 shrink-0 text-muted-foreground" />
          <div className="min-w-0">
            <p className="text-sm font-medium">{state.label}</p>
            <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">{state.detail}</p>
          </div>
        </div>

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
