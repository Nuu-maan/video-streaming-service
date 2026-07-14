import { z } from "zod";

/**
 * A Go duration string, which is what the ban endpoint expects: a sequence of
 * number+unit pairs like `72h`, `30m`, `1h30m`. Go's own parser accepts
 * `ns|us|µs|ms|s|m|h` and nothing longer — there is no `d` for days, which is
 * the mistake everybody makes, so seven days is `168h`.
 *
 * Validating it here rather than letting the API do it is not about trust: it
 * is about the moderator getting the answer while the form is still open,
 * instead of after a round trip that returns a 400 they have to decode.
 */
const GO_DURATION = /^(\d+(\.\d+)?(ns|us|µs|ms|s|m|h))+$/;

/** Zero is not a ban. `0h` parses fine but the API rejects a non-positive duration. */
const NON_ZERO = /[1-9]/;

export const MAX_BAN_REASON = 500;

/**
 * The ban form. An empty duration is the permanent ban — that is the API's own
 * convention, and the form surfaces it as an explicit choice rather than as an
 * empty field the moderator has to intuit.
 */
export const banSchema = z.object({
  user_id: z.string().trim().pipe(z.uuid("That isn't a valid user ID.")),
  reason: z
    .string()
    .trim()
    .min(1, "Say why. The reason is recorded against the account.")
    .max(MAX_BAN_REASON, `Keep the reason under ${MAX_BAN_REASON} characters.`),
  duration: z
    .string()
    .trim()
    .refine((value) => value === "" || GO_DURATION.test(value), {
      message: 'Use a Go duration like "72h" or "1h30m". Days aren\'t a unit — 7 days is "168h".',
    })
    .refine((value) => value === "" || NON_ZERO.test(value), {
      message: "A ban has to last longer than zero. Leave it empty for permanent.",
    }),
});

export type BanInput = z.infer<typeof banSchema>;

export const unbanSchema = z.object({
  user_id: z.string().trim().pipe(z.uuid("That isn't a valid user ID.")),
});

/**
 * The durations a moderator actually reaches for, so the common case is a click
 * and the uncommon one is still typeable. Empty string = permanent.
 */
export const banDurations = [
  { value: "24h", label: "24 hours" },
  { value: "72h", label: "3 days" },
  { value: "168h", label: "7 days" },
  { value: "720h", label: "30 days" },
  { value: "", label: "Permanent" },
] as const;

/** The four things a moderator can do with a report, in the order they escalate. */
export const reviewActions = [
  {
    value: "dismiss",
    label: "Dismiss",
    /** What the button promises, spelled out in the confirmation. */
    consequence: "Closes the report and leaves the content untouched. Use this when the report is unfounded.",
    destructive: false,
  },
  {
    value: "warn_user",
    label: "Warn user",
    consequence: "Closes the report and records a warning against the account. The content stays up.",
    destructive: false,
  },
  {
    value: "delete_video",
    label: "Delete video",
    consequence:
      "Permanently deletes the reported video — the file, every transcoded rendition, and all of its views, likes and comments. This cannot be undone.",
    destructive: true,
  },
  {
    value: "ban_user",
    label: "Ban user",
    consequence:
      "Permanently bans the reported account and closes the report. Requires the manage_users permission; without it the API refuses the review.",
    destructive: true,
  },
] as const;
