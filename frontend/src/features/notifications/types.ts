import type { Notification } from "@/types/common";

export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type NotificationResult = { ok: true } | ActionFailure;
export type MarkAllResult = { ok: true; marked: number } | ActionFailure;

/** Notifications from one calendar day, under the heading a reader would expect. */
export interface NotificationDay {
  /** "Today", "Yesterday", or an absolute date. */
  label: string;
  /** The day itself, for a stable React key. */
  key: string;
  items: Notification[];
}
