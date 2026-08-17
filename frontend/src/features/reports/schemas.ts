import { z } from "zod";

/**
 * The API's ReportType enum, paired with the words a person would actually
 * choose. The machine value goes in `report_type`; the label doubles as the
 * required free-text `reason`, so a reporter who has nothing to add still files
 * a report the moderation queue can read at a glance.
 */
export const reportReasons = [
  { value: "spam", label: "Spam or misleading" },
  { value: "harassment", label: "Harassment or bullying" },
  { value: "hate_speech", label: "Hate speech" },
  { value: "violence", label: "Violence or dangerous acts" },
  { value: "copyright", label: "Copyright infringement" },
  { value: "nudity", label: "Nudity or sexual content" },
  { value: "misinformation", label: "Misinformation" },
  { value: "other", label: "Something else" },
] as const;

export type ReportType = (typeof reportReasons)[number]["value"];

const reportTypeValues = reportReasons.map((reason) => reason.value) as [ReportType, ...ReportType[]];

export const MAX_REPORT_DESCRIPTION = 1000;

export const reportSchema = z.object({
  report_type: z.enum(reportTypeValues, { message: "Choose a reason." }),
  description: z.string().trim().max(MAX_REPORT_DESCRIPTION).optional(),
});

export function labelForReportType(value: ReportType): string {
  return reportReasons.find((reason) => reason.value === value)?.label ?? value;
}
