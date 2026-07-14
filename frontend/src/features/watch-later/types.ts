export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type WatchLaterResult = { ok: true; saved: boolean } | ActionFailure;
