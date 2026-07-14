import { isApiError } from "@/lib/api-error";

interface ErrorCopy {
  title: string;
  description: string;
}

/**
 * Turns a failed admin read into the sentence the moderator should read.
 *
 * Three cases are worth telling apart, and the API distinguishes exactly these
 * three:
 *
 *  - **403** — the role got them through the door, but this particular endpoint
 *    is gated on a permission (`view_analytics`, `moderate_content`,
 *    `manage_users`) their account does not carry. "You don't have permission"
 *    is the honest answer here, and unlike the video case it is safe to say:
 *    the admin area's existence is not a secret from someone already inside it.
 *  - **429** — they are hammering the API. "Slow down", never "something went
 *    wrong", because the fix is to wait and a generic error invites a retry
 *    loop that makes it worse.
 *  - **everything else** — an honest shrug. Inventing a cause is worse than
 *    admitting there isn't one to hand.
 *
 * `needs` names the permission so the reader knows what to go and ask for,
 * rather than filing a bug against a page that is working as designed.
 */
export function errorCopy(error: unknown, subject: string, needs?: string): ErrorCopy {
  if (isApiError(error)) {
    if (error.isRateLimited) {
      return {
        title: "Slow down a moment",
        description: "You're making requests faster than the API allows. Wait a minute, then reload.",
      };
    }
    if (error.isForbidden) {
      return {
        title: `You can't view ${subject}`,
        description: needs
          ? `Your account is missing the ${needs} permission. Ask an admin who has it to grant it.`
          : "Your account doesn't carry the permission this section needs.",
      };
    }
  }

  return {
    title: `Couldn't load ${subject}`,
    description: "Something went wrong talking to the API. Reload the page to try again.",
  };
}
