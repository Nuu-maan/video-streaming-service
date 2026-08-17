"use server";

import { labelForReportType, reportSchema } from "@/features/reports/schemas";
import type { ActionFailure, ReportInput, ReportResult } from "@/features/reports/types";
import { api } from "@/lib/api-client";
import { isApiError } from "@/lib/api-error";

function fail(error: unknown): ActionFailure {
  if (isApiError(error)) {
    /* Filing the same report twice is a 409. That is not a failure of the
       system — it is the system telling us the report already landed. Say so. */
    if (error.status === 409) {
      return { ok: false, code: "DUPLICATE", message: "You already reported this." };
    }
    if (error.isUnauthorized) {
      return { ok: false, code: "UNAUTHORIZED", message: "Sign in to report content." };
    }
    if (error.isRateLimited) {
      return { ok: false, code: "RATE_LIMITED", message: "Slow down a moment, then try again." };
    }
    if (error.isNotFound) {
      return { ok: false, code: "NOT_FOUND", message: "That content no longer exists." };
    }
    return { ok: false, code: error.code, message: error.message };
  }
  return { ok: false, code: "UNKNOWN", message: "Something went wrong. Please try again." };
}

export async function submitReport(input: ReportInput): Promise<ReportResult> {
  const parsed = reportSchema.safeParse({
    report_type: input.report_type,
    description: input.description,
  });
  if (!parsed.success) {
    return { ok: false, code: "VALIDATION", message: parsed.error.issues[0].message };
  }

  const { target } = input;
  const body = {
    report_type: parsed.data.report_type,
    // `reason` is required and free-text; the chosen label is the honest summary.
    reason: labelForReportType(parsed.data.report_type),
    description: parsed.data.description || undefined,
    video_id: target.kind === "video" ? target.id : undefined,
    comment_id: target.kind === "comment" ? target.id : undefined,
    user_id: target.kind === "user" ? target.id : undefined,
  };

  try {
    await api.post("/reports", { body });
  } catch (error) {
    return fail(error);
  }
  return { ok: true };
}
