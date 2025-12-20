export interface APIError {
  code: string;
  message: string;
  details?: Record<string, unknown>;
  traceId?: string;
}

export namespace APIError {
  export function fromJson(json: Record<string, unknown>): APIError {
    return {
      code: json.code as string,
      message: json.message as string,
      details: json.details as Record<string, unknown> | undefined,
      traceId: json.trace_id as string | undefined,
    };
  }
}

export type MizuErrorType =
  | 'invalid_response'
  | 'http'
  | 'api'
  | 'network'
  | 'encoding'
  | 'decoding'
  | 'unauthorized'
  | 'token_expired';

export class MizuError extends Error {
  readonly type: MizuErrorType;
  readonly cause?: unknown;
  readonly apiError?: APIError;
  readonly statusCode?: number;

  private constructor(type: MizuErrorType, message: string, cause?: unknown) {
    super(message);
    this.name = 'MizuError';
    this.type = type;
    this.cause = cause;
  }

  static invalidResponse(): MizuError {
    return new MizuError('invalid_response', 'Invalid server response');
  }

  static http(statusCode: number, body: string): MizuError {
    const error = new MizuError('http', `HTTP error ${statusCode}`, body);
    (error as { statusCode: number }).statusCode = statusCode;
    return error;
  }

  static api(apiError: APIError): MizuError {
    const error = new MizuError('api', apiError.message, apiError);
    (error as { apiError: APIError }).apiError = apiError;
    return error;
  }

  static network(cause: Error): MizuError {
    return new MizuError('network', 'Network error', cause);
  }

  static encoding(cause: Error): MizuError {
    return new MizuError('encoding', 'Encoding error', cause);
  }

  static decoding(cause: Error): MizuError {
    return new MizuError('decoding', 'Decoding error', cause);
  }

  static unauthorized(): MizuError {
    return new MizuError('unauthorized', 'Unauthorized');
  }

  static tokenExpired(): MizuError {
    return new MizuError('token_expired', 'Token expired');
  }

  get isInvalidResponse(): boolean {
    return this.type === 'invalid_response';
  }

  get isHttp(): boolean {
    return this.type === 'http';
  }

  get isApi(): boolean {
    return this.type === 'api';
  }

  get isNetwork(): boolean {
    return this.type === 'network';
  }

  get isUnauthorized(): boolean {
    return this.type === 'unauthorized';
  }

  get isTokenExpired(): boolean {
    return this.type === 'token_expired';
  }
}
