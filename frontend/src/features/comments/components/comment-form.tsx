"use client";

import { LoaderCircle } from "lucide-react";
import { useEffect, useRef, useState, useTransition } from "react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  COMMENT_COUNTER_THRESHOLD,
  MAX_COMMENT_LENGTH,
} from "@/features/comments/schemas";
import type { CommentViewer } from "@/features/comments/types";
import { cn } from "@/lib/utils";

interface CommentFormProps {
  viewer: CommentViewer;
  /** Resolve true to clear the field; false leaves the draft intact so nothing is lost. */
  onSubmit: (content: string) => Promise<boolean>;
  placeholder?: string;
  submitLabel?: string;
  initialValue?: string;
  onCancel?: () => void;
  autoFocus?: boolean;
  /** Hides the avatar — replies and edit boxes are already inside an avatar's column. */
  compact?: boolean;
  className?: string;
}

/**
 * The one composer, used for a new comment, a reply, and an edit.
 *
 * The action buttons stay hidden until the field is focused or holds a draft:
 * an idle thread shows one quiet input, not a form. ⌘/Ctrl+Enter submits,
 * Escape cancels — the two shortcuts everyone already has in their fingers.
 *
 * The character counter appears only in the last 500 characters. A counter that
 * is always on is noise; one that arrives at 9,500 is a warning.
 */
export function CommentForm({
  viewer,
  onSubmit,
  placeholder = "Add a comment…",
  submitLabel = "Comment",
  initialValue = "",
  onCancel,
  autoFocus = false,
  compact = false,
  className,
}: CommentFormProps) {
  const [value, setValue] = useState(initialValue);
  const [focused, setFocused] = useState(false);
  const [pending, startTransition] = useTransition();
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (!autoFocus) return;
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.focus();
    // Land the caret at the end of an existing draft, not in front of it.
    textarea.setSelectionRange(textarea.value.length, textarea.value.length);
  }, [autoFocus]);

  const trimmed = value.trim();
  const canSubmit = trimmed.length > 0 && trimmed.length <= MAX_COMMENT_LENGTH && !pending;
  const showActions = focused || value.length > 0 || Boolean(onCancel);
  const remaining = MAX_COMMENT_LENGTH - value.length;
  const showCounter = value.length >= COMMENT_COUNTER_THRESHOLD;

  /**
   * Two announcements over the life of a draft, not one per character. The
   * string only changes when a threshold is crossed, and a live region is silent
   * when its content is unchanged — so typing through the middle of the range
   * says nothing at all, which is the point.
   */
  const countdownAnnouncement =
    remaining <= 0
      ? "No characters left."
      : remaining <= 100
        ? "100 characters left."
        : "";

  function submit() {
    if (!canSubmit) return;
    startTransition(async () => {
      const cleared = await onSubmit(trimmed);
      if (cleared) setValue("");
    });
  }

  function handleKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      submit();
      return;
    }
    if (event.key === "Escape" && onCancel) {
      event.preventDefault();
      onCancel();
    }
  }

  const initial = viewer.username.slice(0, 1).toUpperCase();

  return (
    <form
      className={cn("flex gap-3", className)}
      onSubmit={(event) => {
        event.preventDefault();
        submit();
      }}
    >
      {compact ? null : (
        <Avatar className="mt-0.5 size-9 shrink-0">
          {viewer.avatarUrl ? <AvatarImage src={viewer.avatarUrl} alt="" /> : null}
          <AvatarFallback className="bg-brand-800 text-xs font-medium text-brand-100">
            {initial}
          </AvatarFallback>
        </Avatar>
      )}

      <div className="min-w-0 flex-1">
        <Textarea
          ref={textareaRef}
          value={value}
          onChange={(event) => setValue(event.target.value.slice(0, MAX_COMMENT_LENGTH))}
          onFocus={() => setFocused(true)}
          onKeyDown={handleKeyDown}
          disabled={pending}
          placeholder={placeholder}
          rows={compact ? 2 : 1}
          aria-label={submitLabel}
          className="field-sizing-content max-h-64 min-h-9 resize-none"
        />

        {/*
         * The counter is visible, and it is NOT a live region. It used to be —
         * which meant the screen reader interrupted the person with a new number
         * on every single keystroke, while they were mid-sentence. A live region
         * must never be wired to a per-keystroke value.
         *
         * What IS announced is the threshold: once at "100 left", once at "no
         * characters left". Those are the two moments the number actually
         * changes what the writer should do.
         */}
        <span role="status" aria-live="polite" className="sr-only">
          {countdownAnnouncement}
        </span>

        {showActions ? (
          <div className="mt-2 flex items-center justify-end gap-2">
            {showCounter ? (
              <span
                className={cn(
                  "mr-auto text-xs tabular-nums",
                  remaining <= 0 ? "text-destructive" : "text-muted-foreground",
                )}
              >
                {remaining.toLocaleString()} left
              </span>
            ) : null}

            {onCancel ? (
              <Button type="button" variant="ghost" size="sm" disabled={pending} onClick={onCancel}>
                Cancel
              </Button>
            ) : null}

            <Button type="submit" size="sm" disabled={!canSubmit}>
              {pending ? <LoaderCircle aria-hidden className="animate-spin" /> : null}
              {submitLabel}
            </Button>
          </div>
        ) : null}
      </div>
    </form>
  );
}
