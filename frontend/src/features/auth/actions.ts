"use server";

import { redirect } from "next/navigation";
import type { z } from "zod";

import { routes } from "@/config/routes";
import {
  changePasswordSchema,
  forgotPasswordSchema,
  loginSchema,
  registerSchema,
  resetPasswordSchema,
  verifyEmailSchema,
  type AuthFormState,
} from "@/features/auth/schemas";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";
import { clearSession, setSession } from "@/lib/session";
import type { TokenPair } from "@/types/common";

/**
 * Every action here re-validates its input with the same zod schema the form
 * used. A Server Function is a public POST endpoint — Next's docs are explicit
 * that the proxy matcher does not protect it — so nothing below trusts a byte
 * that arrived from the client.
 */

/** First message per field, keyed by input name, for inline display. */
function fieldErrorsOf(error: z.ZodError): Record<string, string> {
  const out: Record<string, string> = {};
  for (const issue of error.issues) {
    const key = issue.path.map(String).join(".");
    if (key && !(key in out)) out[key] = issue.message;
  }
  return out;
}

/**
 * Only ever redirect to an internal path. `?next=` is attacker-writable, and
 * without this check a crafted link could bounce a fresh session to another
 * origin ("//evil.example" is a protocol-relative URL, not a path).
 */
function safeNext(value: FormDataEntryValue | null): string | null {
  if (typeof value !== "string") return null;
  if (!value.startsWith("/") || value.startsWith("//") || value.includes("\\")) return null;
  return value;
}

/**
 * Maps an ApiError to copy a person can act on. `overrides` lets each action
 * name the statuses it knows better (401 on login is "wrong password"; 401 on
 * change-password is "session expired"). A 400 surfaces the API's own message —
 * it enforces password strength, and its wording is the policy.
 */
function messageFor(error: unknown, overrides: Record<number, string> = {}): string {
  if (isApiError(error)) {
    const override = overrides[error.status];
    if (override) return override;
    if (error.isRateLimited) return "Too many attempts. Wait a minute and try again.";
    if (error.status === 503) return "The sign-in service is briefly unavailable. Try again in a moment.";
    return error.message;
  }
  return "Something went wrong. Try again.";
}

export async function login(prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const parsed = loginSchema.safeParse({
    identifier: formData.get("identifier") ?? "",
    password: formData.get("password") ?? "",
  });
  if (!parsed.success) {
    return { status: "error", fieldErrors: fieldErrorsOf(parsed.error) };
  }

  try {
    const tokens = await api.post<TokenPair>("/auth/login", { body: parsed.data, auth: false });
    await setSession(tokens);
  } catch (error) {
    return {
      status: "error",
      /* The API answers unknown account and wrong password identically; so do
         we. Naming which half was wrong would let account existence be probed.
         403 is the one status that is *not* ambiguous — the API only sends it
         for USER_BANNED — so it gets its own copy. */
      error: messageFor(error, {
        401: "Incorrect username or password.",
        403: "This account has been suspended.",
      }),
    };
  }

  redirect(safeNext(formData.get("next")) ?? routes.home);
}

export async function register(prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const parsed = registerSchema.safeParse({
    username: formData.get("username") ?? "",
    email: formData.get("email") ?? "",
    password: formData.get("password") ?? "",
  });
  if (!parsed.success) {
    return { status: "error", fieldErrors: fieldErrorsOf(parsed.error) };
  }

  try {
    const tokens = await api.post<TokenPair>("/auth/register", { body: parsed.data, auth: false });
    await setSession(tokens);
  } catch (error) {
    return {
      status: "error",
      error: messageFor(error, {
        409: "That username or email is already taken.",
        403: "This account has been suspended.",
      }),
    };
  }

  redirect(safeNext(formData.get("next")) ?? routes.home);
}

export async function logout(): Promise<void> {
  try {
    await api.post("/auth/logout");
  } catch {
    /* Best-effort: a token the server will not revoke (network down, already
       expired) must still be dropped locally, or sign-out becomes impossible
       exactly when the session is most broken. */
  }
  await clearSession();
  redirect(routes.home);
}

/** The name the app shell threads into `<SiteHeader signOutAction={…}>`. */
export async function signOut(): Promise<void> {
  await logout();
}

export async function logoutAll(): Promise<void> {
  try {
    await api.post("/auth/logout-all");
  } catch {
    /* Same reasoning as logout: revocation is best-effort, dropping the local
       session is not. */
  }
  await clearSession();
  redirect(routes.login);
}

export async function forgotPassword(prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const parsed = forgotPasswordSchema.safeParse({ email: formData.get("email") ?? "" });
  if (!parsed.success) {
    return { status: "error", fieldErrors: fieldErrorsOf(parsed.error) };
  }

  try {
    await api.post("/auth/forgot-password", { body: parsed.data, auth: false });
  } catch (error) {
    return { status: "error", error: messageFor(error) };
  }

  return {
    status: "success",
    /* The API answers identically whether or not the address exists, and so
       does this copy — confirming an address has an account would leak it. */
    message: "If an account exists for that address, a reset link is on its way. Check your inbox.",
  };
}

export async function resetPassword(prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const parsed = resetPasswordSchema.safeParse({
    token: formData.get("token") ?? "",
    password: formData.get("password") ?? "",
  });
  if (!parsed.success) {
    const fieldErrors = fieldErrorsOf(parsed.error);
    /* The token travels in a hidden input; an error pinned to it would render
       nowhere. Surface it at form level instead. */
    if (fieldErrors.token) {
      return { status: "error", error: "This reset link is invalid or has expired. Request a new one." };
    }
    return { status: "error", fieldErrors };
  }

  try {
    await api.post("/auth/reset-password", { body: parsed.data, auth: false });
  } catch (error) {
    if (isApiError(error) && error.code === "INVALID_TOKEN") {
      return { status: "error", error: "This reset link is invalid or has expired. Request a new one." };
    }
    return { status: "error", error: messageFor(error) };
  }

  return { status: "success", message: "Your password has been reset. Sign in with the new one." };
}

export async function changePassword(prevState: AuthFormState, formData: FormData): Promise<AuthFormState> {
  const parsed = changePasswordSchema.safeParse({
    current_password: formData.get("current_password") ?? "",
    new_password: formData.get("new_password") ?? "",
  });
  if (!parsed.success) {
    return { status: "error", fieldErrors: fieldErrorsOf(parsed.error) };
  }

  try {
    await api.post("/me/change-password", { body: parsed.data });
  } catch (error) {
    if (isApiError(error) && error.code === "INVALID_CURRENT_PASSWORD") {
      return { status: "error", fieldErrors: { current_password: "That is not your current password." } };
    }
    return {
      status: "error",
      error: messageFor(error, { 401: "Your session has expired. Sign in again." }),
    };
  }

  return { status: "success", message: "Password changed." };
}

export async function sendVerificationEmail(): Promise<AuthFormState> {
  try {
    await api.post("/auth/verify-email/send");
  } catch (error) {
    if (isApiError(error) && error.code === "EMAIL_ALREADY_VERIFIED") {
      return { status: "success", message: "Your email is already verified." };
    }
    return {
      status: "error",
      error: messageFor(error, { 401: "Sign in to verify your email." }),
    };
  }

  return { status: "success", message: "Verification email sent. Check your inbox." };
}

export async function verifyEmail(token: string): Promise<AuthFormState> {
  const parsed = verifyEmailSchema.safeParse({ token });
  if (!parsed.success) {
    return { status: "error", error: "This verification link is invalid." };
  }

  try {
    await api.post("/auth/verify-email", { body: parsed.data, auth: false });
  } catch (error) {
    if (isApiError(error) && error.code === "INVALID_TOKEN") {
      return { status: "error", error: "This verification link is invalid or has expired." };
    }
    return { status: "error", error: messageFor(error) };
  }

  return { status: "success", message: "Your email address is verified." };
}
