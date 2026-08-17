import type { ReportType } from "@/features/reports/schemas";

/** What is being reported. Exactly one of these is set on a report. */
export type ReportTarget =
  | { kind: "video"; id: string }
  | { kind: "comment"; id: string }
  | { kind: "user"; id: string };

export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type ReportResult = { ok: true } | ActionFailure;

export interface ReportInput {
  target: ReportTarget;
  report_type: ReportType;
  description?: string;
}
