/**
 * Display formatting. Every one of these is a pure function of its input so it
 * renders identically on the server and the client — a formatter that reads the
 * clock or the locale from the environment produces different HTML in each place
 * and hydration fails.
 */

/** 754 -> "12:34", 3754 -> "1:02:34". Hours only appear when there are hours. */
export function formatDuration(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds < 0) return "0:00";

  const seconds = Math.floor(totalSeconds % 60);
  const minutes = Math.floor((totalSeconds / 60) % 60);
  const hours = Math.floor(totalSeconds / 3600);

  const paddedSeconds = seconds.toString().padStart(2, "0");
  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, "0")}:${paddedSeconds}`;
  }
  return `${minutes}:${paddedSeconds}`;
}

/** 1 -> "1 view", 1_500 -> "1.5K views", 2_400_000 -> "2.4M views". */
export function formatCount(value: number, singular: string, plural = `${singular}s`): string {
  const label = value === 1 ? singular : plural;
  return `${formatCompact(value)} ${label}`;
}

export function formatCompact(value: number): string {
  if (!Number.isFinite(value)) return "0";
  if (value < 1_000) return String(value);
  if (value < 1_000_000) return `${trim(value / 1_000)}K`;
  if (value < 1_000_000_000) return `${trim(value / 1_000_000)}M`;
  return `${trim(value / 1_000_000_000)}B`;
}

function trim(value: number): string {
  // 1.0K reads worse than 1K, but 1.5K carries real information.
  const rounded = Math.round(value * 10) / 10;
  return rounded % 1 === 0 ? String(Math.trunc(rounded)) : rounded.toFixed(1);
}

/**
 * "3 days ago". Takes `now` as an argument rather than calling Date.now()
 * internally: a server-rendered "2 minutes ago" and a client-rendered one
 * computed a moment later disagree, and React reports a hydration mismatch.
 * Callers that need live-updating relative time re-render with a new `now`.
 */
export function formatRelativeTime(date: string | Date, now: Date = new Date()): string {
  const then = typeof date === "string" ? new Date(date) : date;
  const seconds = Math.round((now.getTime() - then.getTime()) / 1000);

  if (!Number.isFinite(seconds)) return "";
  if (seconds < 60) return "just now";

  const units: Array<[Intl.RelativeTimeFormatUnit, number]> = [
    ["year", 60 * 60 * 24 * 365],
    ["month", 60 * 60 * 24 * 30],
    ["week", 60 * 60 * 24 * 7],
    ["day", 60 * 60 * 24],
    ["hour", 60 * 60],
    ["minute", 60],
  ];

  const formatter = new Intl.RelativeTimeFormat("en", { numeric: "auto" });
  for (const [unit, secondsPerUnit] of units) {
    const value = Math.floor(seconds / secondsPerUnit);
    if (value >= 1) {
      return formatter.format(-value, unit);
    }
  }
  return "just now";
}

/** 286_866 -> "280 KB". Binary units, because that is what a file browser shows. */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";

  const units = ["B", "KB", "MB", "GB", "TB"];
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / Math.pow(1024, exponent);

  return `${value >= 10 || exponent === 0 ? Math.round(value) : value.toFixed(1)} ${units[exponent]}`;
}

/** Absolute date for a tooltip, where "3 days ago" is not precise enough. */
export function formatDate(date: string | Date): string {
  const value = typeof date === "string" ? new Date(date) : date;
  return new Intl.DateTimeFormat("en", { dateStyle: "medium" }).format(value);
}
