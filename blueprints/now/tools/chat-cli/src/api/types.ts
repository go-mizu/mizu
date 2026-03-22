export interface Chat {
  id: string;
  kind: string;
  title: string;
  creator: string;
  peer?: string;
  created_at: string;
}

export interface Message {
  id: string;
  chat: string;
  actor: string;
  text: string;
  created_at: string;
}

export interface RegisterResponse {
  actor: string;
  recovery_code: string;
}

export interface ListResponse<T> {
  items: T[];
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: string,
  ) {
    super(`HTTP ${status}: ${body}`);
    this.name = "ApiError";
  }
}

export class AuthError extends ApiError {
  constructor(status: number, body: string) {
    super(status, body);
    this.name = "AuthError";
  }
}

export class RateLimitError extends ApiError {
  public retryAfter: number;
  constructor(status: number, body: string, retryAfter: number) {
    super(status, body);
    this.name = "RateLimitError";
    this.retryAfter = retryAfter;
  }
}
