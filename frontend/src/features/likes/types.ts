/** The caller's current rating on a video. One row server-side: liking after disliking flips it. */
export type Rating = "like" | "dislike";

export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type LikeActionResult = { ok: true } | ActionFailure;
