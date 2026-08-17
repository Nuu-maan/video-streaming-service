import { cn } from "@/lib/utils";

interface IconSwapProps {
  /** Shown while `active` is false. This one defines the layout box. */
  from: React.ReactNode;
  /** Shown while `active` is true. Overlaid, so it never affects layout. */
  to: React.ReactNode;
  active: boolean;
  /** Sizes the box. Defaults to `size-4`, which is the app's icon size. */
  className?: string;
}

/*
 * The house icon cross-fade, in one place.
 *
 * Both icons stay mounted; one is absolutely positioned over the other. Toggling
 * `active` cross-fades them — the entering icon scales up from 0.25 while the
 * exiting one scales down to it, both carrying opacity and a 4px blur. Because
 * neither ever unmounts, the swap has a real exit as well as a real enter, and
 * the button never flashes empty. The blur is doing quiet work: without it you
 * see two distinct glyphs overlapping mid-fade; with it, the eye reads one shape
 * transforming into another.
 *
 * Values are the ones the design system fixes (scale 0.25 → 1, opacity 0 → 1,
 * blur 4px → 0, `ease-swap`) and are not knobs.
 *
 * WHEN NOT TO USE THIS. An icon swap is *state indication*, and state indication
 * is worth animating only when the state change is worth noticing. A control the
 * viewer hits dozens of times a session — the player's play/pause, which is the
 * space bar — must stay instant: animation on a key you press constantly reads as
 * lag, not polish. Subscribe, fullscreen, copy-to-clipboard: rare, deliberate,
 * and carrying a state the user wants confirmed. Those animate.
 */
const LAYER = "flex items-center justify-center transition-[opacity,filter,scale] duration-(--motion-medium) ease-swap";
const SHOWN = "scale-100 opacity-100 blur-none";
const HIDDEN = "scale-[0.25] opacity-0 blur-[4px]";

export function IconSwap({ from, to, active, className }: IconSwapProps) {
  return (
    <span aria-hidden className={cn("relative flex size-4 shrink-0 items-center justify-center", className)}>
      <span className={cn(LAYER, active ? HIDDEN : SHOWN)}>{from}</span>
      <span className={cn("absolute inset-0", LAYER, active ? SHOWN : HIDDEN)}>{to}</span>
    </span>
  );
}
