"use client";

import { cn } from "@/lib/utils";

interface PlayerButtonProps extends React.ComponentProps<"button"> {
  /** Required: every control here is an icon, and an icon alone says nothing. */
  label: string;
}

/**
 * The one control button in the player chrome.
 *
 * It lives on top of arbitrary video frames, so it cannot rely on the app's
 * surface tokens — it is white-on-video, with a hover wash rather than a border.
 * The visual glyph is 20px but the hit area is 40px, which is the smallest thing
 * a mouse should ever be asked to hit, and the focus ring is white for the same
 * reason the text is: the background is a picture, not a colour we chose.
 */
export function PlayerButton({ label, className, children, ...props }: PlayerButtonProps) {
  return (
    <button
      type="button"
      aria-label={label}
      title={label}
      className={cn(
        "inline-flex size-10 shrink-0 items-center justify-center rounded-lg text-white/90 outline-none",
        "transition-[background-color,color,scale] duration-(--motion-fast) ease-out-quart",
        "hover:bg-white/15 hover:text-white",
        "focus-visible:ring-2 focus-visible:ring-white/80",
        "active:scale-96",
        "[&_svg]:size-5 [&_svg]:shrink-0",
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
