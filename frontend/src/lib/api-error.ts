/**
 * The API answers a failure with `{ success: false, error: { code, message } }`.
 * ApiError carries that through so a caller can branch on a stable machine code
 * instead of pattern-matching an English sentence that may be reworded later.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  /** Correlates a client-side failure with a line in the server log. */
  readonly requestId: string | null;

  constructor(status: number, code: string, message: string, requestId: string | null = null) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }

  /**
   * A private video answers 404, never 403, so that its existence is not
   * observable. That means "not found" and "not allowed" are deliberately
   * indistinguishable here, and the UI must not claim to know which it was.
   */
  get isNotFound(): boolean {
    return this.status === 404;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }

  get isForbidden(): boolean {
    return this.status === 403;
  }

  get isRateLimited(): boolean {
    return this.status === 429;
  }

  get isValidation(): boolean {
    return this.status === 400;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
