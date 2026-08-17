"use client";

import { formatDuration } from "@/lib/format";
import { cn } from "@/lib/utils";

interface NoteTimestampProps {
  seconds: number;
  /** Omit where there is no player to drive — the chip then reads, but does not act. */
  onSeek?: (seconds: number) => void;
  className?: string;
}

/**
 * The anchor chip: `2:14`, and clicking it takes you there.
 *
 * Tabular numerals, because a column of these is read as a column — proportional
 * digits make `1:11` and `2:04` different widths and the left edge of the text
 * beside them ripples. It is the whole reason the timestamps line up.
 *
 * When there is no `onSeek` this renders a `<span>`, not a disabled button. A
 * disabled button is still announced as a button, which promises an action that
 * does not exist; a span just says what time it is.
 */
export function NoteTimestamp({ seconds, onSeek, className }: NoteTimestampProps) {
  const label = formatDuration(seconds);

  const shared = cn(
    "inline-flex shrink-0 items-center rounded-md px-1.5 py-0.5 font-mono text-[0.6875rem] leading-none font-medium tabular-nums",
    className,
  );

  if (!onSeek) {
    return <span className={cn(shared, "bg-muted text-muted-foreground")}>{label}</span>;
  }

  return (
    <button
      type="button"
      onClick={() => onSeek(seconds)}
      title={`Jump to ${label}`}
      aria-label={`Jump to ${label}`}
      className={cn(
        shared,
        "bg-brand-500/12 text-brand-700 outline-none dark:text-brand-300",
        "transition-[background-color,color,scale] duration-(--motion-fast) ease-out-quart",
        "hover:bg-brand-500/20 hover:text-brand-800 dark:hover:text-brand-200",
        "focus-visible:ring-2 focus-visible:ring-ring/60 active:scale-95",
      )}
    >
      {label}
    </button>
  );
}
