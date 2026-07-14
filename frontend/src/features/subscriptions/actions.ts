"use server";

import { revalidatePath } from "next/cache";

import { routes } from "@/config/routes";
import { getCurrentUser } from "@/features/auth/current-user";
import type { ActionFailure, SubscribeResult } from "@/features/subscriptions/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to subscribe." };
    }
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    if (error.isNotFound) {
      return { ok: false, code: "NOT_FOUND", message: "That channel no longer exists." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

/**
 * Subscribing to yourself is a 400 on the API. We stop it here first — a
 * self-subscribe is a UI bug, not a user error, and a validation toast is a
 * poor way to learn about one. The button hides itself on your own channel;
 * this is the belt to that pair of braces, because a Server Function is a
 * public POST endpoint and the caller is not to be trusted.
 */
export async function subscribe(userId: string): Promise<SubscribeResult> {
  const me = await getCurrentUser();
  if (!me) return { ok: false, code: "UNAUTHORIZED", message: "Sign in to subscribe." };
  if (me.id === userId) {
    return { ok: false, code: "SELF_SUBSCRIBE", message: "You can't subscribe to your own channel." };
  }

  try {
    await api.post(`/users/${userId}/subscribe`);
  } catch (error) {
    return fail(error);
  }
  revalidatePath(routes.subscriptions);
  return { ok: true, subscribed: true };
}

export async function unsubscribe(userId: string): Promise<SubscribeResult> {
  try {
    await api.delete(`/users/${userId}/subscribe`);
  } catch (error) {
    // Not subscribed is a 404 — and it is exactly where we were headed.
    if (!(isApiError(error) && error.isNotFound)) return fail(error);
  }
  revalidatePath(routes.subscriptions);
  return { ok: true, subscribed: false };
}

export async function toggleSubscription(
  userId: string,
  subscribed: boolean,
): Promise<SubscribeResult> {
  return subscribed ? unsubscribe(userId) : subscribe(userId);
}
