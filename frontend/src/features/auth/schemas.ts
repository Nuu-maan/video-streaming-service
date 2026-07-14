import { z } from "zod";

/**
 * One schema per auth flow, shared by the client form and the Server Action.
 * The action re-validates every submission with the same schema the form used,
 * so the two can never drift apart — and the server never trusts the client.
 *
 * Password strength is deliberately NOT modelled here. The API enforces its own
 * policy and returns a precise message on 400; duplicating it client-side would
 * create a second, subtly different policy that rejects passwords the server
 * would accept (or worse, the reverse). We only require the field to be filled.
 */

/** Mirrors the API's own username rule: `^[a-zA-Z0-9_]{3,30}$`. */
const username = z
  .string()
  .trim()
  .min(3, "At least 3 characters.")
  .max(30, "At most 30 characters.")
  .regex(/^[a-zA-Z0-9_]+$/, "Letters, numbers and underscores only.");

const email = z
  .string()
  .trim()
  .min(1, "Enter your email address.")
  .pipe(z.email("Enter a valid email address."));

export const loginSchema = z.object({
  identifier: z.string().trim().min(1, "Enter your username or email."),
  password: z.string().min(1, "Enter your password."),
});

export const registerSchema = z.object({
  username,
  email,
  password: z.string().min(1, "Choose a password."),
});

export const forgotPasswordSchema = z.object({
  email,
});

export const resetPasswordSchema = z.object({
  token: z.string().min(1),
  password: z.string().min(1, "Choose a new password."),
});

export const changePasswordSchema = z.object({
  current_password: z.string().min(1, "Enter your current password."),
  new_password: z.string().min(1, "Choose a new password."),
});

export const verifyEmailSchema = z.object({
  token: z.string().min(1),
});

export type LoginInput = z.infer<typeof loginSchema>;
export type RegisterInput = z.infer<typeof registerSchema>;
export type ForgotPasswordInput = z.infer<typeof forgotPasswordSchema>;
export type ResetPasswordInput = z.infer<typeof resetPasswordSchema>;
export type ChangePasswordInput = z.infer<typeof changePasswordSchema>;

/**
 * The state every auth action returns to `useActionState`.
 *
 * `fieldErrors` is keyed by input `name`, so a form can pin a message to the
 * field it belongs to; `error` is the form-level failure ("incorrect password",
 * "slow down"); `message` is the confirmation copy for flows that end on the
 * same page (forgot / reset / change password) rather than in a redirect.
 */
export interface AuthFormState {
  status: "idle" | "error" | "success";
  error?: string;
  fieldErrors?: Record<string, string>;
  message?: string;
}

export const idleAuthFormState: AuthFormState = { status: "idle" };
