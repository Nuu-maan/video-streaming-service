export interface ActionFailure {
  ok: false;
  code: string;
  message: string;
}

export type SubscribeResult = { ok: true; subscribed: boolean } | ActionFailure;
