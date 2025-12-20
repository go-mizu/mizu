import { MizuError } from './errors';

export interface TransportRequest {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: string;
  timeout: number;
}

export interface TransportResponse {
  statusCode: number;
  headers: Record<string, string>;
  body: string;
}

export interface Transport {
  execute(request: TransportRequest): Promise<TransportResponse>;
}

export interface RequestInterceptor {
  intercept(request: TransportRequest): Promise<TransportRequest>;
}

/**
 * HTTP-based transport implementation using fetch
 */
export class HttpTransport implements Transport {
  private interceptors: RequestInterceptor[] = [];

  /** Adds a request interceptor */
  addInterceptor(interceptor: RequestInterceptor): void {
    this.interceptors.push(interceptor);
  }

  async execute(request: TransportRequest): Promise<TransportResponse> {
    let req = request;

    // Apply interceptors
    for (const interceptor of this.interceptors) {
      req = await interceptor.intercept(req);
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), req.timeout);

    try {
      const response = await fetch(req.url, {
        method: req.method,
        headers: req.headers,
        body: req.body,
        signal: controller.signal,
        credentials: 'include',
      });

      clearTimeout(timeoutId);

      const body = await response.text();
      const headers: Record<string, string> = {};
      response.headers.forEach((value, key) => {
        headers[key] = value;
      });

      return {
        statusCode: response.status,
        headers,
        body,
      };
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof Error && error.name === 'AbortError') {
        throw MizuError.network(new Error('Request timed out'));
      }
      throw MizuError.network(error instanceof Error ? error : new Error(String(error)));
    }
  }
}

/**
 * Logging interceptor for debugging
 */
export class LoggingInterceptor implements RequestInterceptor {
  async intercept(request: TransportRequest): Promise<TransportRequest> {
    if (import.meta.env.DEV) {
      console.log(`[Mizu] ${request.method} ${request.url}`);
    }
    return request;
  }
}

/**
 * Retry interceptor with exponential backoff
 */
export class RetryInterceptor implements RequestInterceptor {
  constructor(
    public maxRetries: number = 3,
    public baseDelay: number = 1000
  ) {}

  async intercept(request: TransportRequest): Promise<TransportRequest> {
    return request;
  }
}
