export interface Env {
  DB: D1Database;
  CRAWL_QUEUE: Queue;
  AUTH_TOKEN: string;
}

// --- Request types ---

export interface CrawlRequest {
  url: string;
  limit?: number;
  depth?: number;
  formats?: ("html" | "markdown")[];
  userAgent?: string;
  setExtraHTTPHeaders?: Record<string, string>;
  options?: CrawlOptions;
}

export interface CrawlOptions {
  includeSubdomains?: boolean;
  includeExternalLinks?: boolean;
  includePatterns?: string[];
  excludePatterns?: string[];
}

// --- Response types ---

export interface ApiResponse<T = unknown> {
  success: boolean;
  result: T | null;
  errors?: ApiError[];
}

export interface ApiError {
  code: number;
  message: string;
}

export interface JobResult {
  id: string;
  status: JobStatus;
  total: number;
  finished: number;
  cursor: number;
  records: PageRecord[];
}

export interface PageRecord {
  url: string;
  status: RecordStatus;
  markdown: string | null;
  html: string | null;
  metadata: {
    status: number;
    title: string;
    url: string;
  };
}

export type JobStatus = "running" | "completed" | "errored" | "cancelled_by_user";
export type RecordStatus = "queued" | "completed" | "errored" | "skipped";

// --- DB row types ---

export interface JobRow {
  id: string;
  url: string;
  status: JobStatus;
  config: string;
  total: number;
  finished: number;
  created_at: number;
  updated_at: number;
}

export interface PageRow {
  id: number;
  job_id: string;
  url: string;
  status: RecordStatus;
  http_status: number;
  title: string;
  html: string | null;
  markdown: string | null;
  depth: number;
  created_at: number;
}

// --- Queue message ---

export interface CrawlMessage {
  jobId: string;
  url: string;
  depth: number;
}

// --- Parsed config stored in jobs.config ---

export interface JobConfig {
  url: string;
  limit: number;
  depth: number;
  formats: ("html" | "markdown")[];
  userAgent: string;
  extraHeaders: Record<string, string>;
  options: Required<CrawlOptions>;
}
