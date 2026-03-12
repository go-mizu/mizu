export interface Env {
  DB: D1Database;
  CRAWL_QUEUE: Queue;
  AUTH_TOKEN: string;
  // CF Browser Rendering credentials (optional Worker secrets)
  CF_ACCOUNT_ID?: string;
  CF_API_TOKEN?: string;
  // Optional Workers AI binding for /api/json fallback
  AI?: any;
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

// ── Shared option types ──────────────────────────────────────────────────────

export interface GotoOptions {
  waitUntil?: "domcontentloaded" | "networkidle0" | "networkidle2";
  timeout?: number;
}

export interface Cookie {
  name: string;
  value: string;
  domain?: string;
  path?: string;
  secure?: boolean;
  httpOnly?: boolean;
}

export interface AuthCredentials {
  username: string;
  password: string;
}

export interface Viewport {
  width?: number;
  height?: number;
  deviceScaleFactor?: number;
}

export interface WaitForSelector {
  selector: string;
  timeout?: number;
  visible?: boolean;
}

export interface ScriptTag {
  content: string;
}

export interface StyleTag {
  content?: string;
  url?: string;
}

// Shared fields present on all single-URL endpoint requests
export interface BaseRequest {
  url?: string;
  html?: string;
  gotoOptions?: GotoOptions;
  cookies?: Cookie[];
  authenticate?: AuthCredentials;
  setExtraHTTPHeaders?: Record<string, string>;
  userAgent?: string;
  viewport?: Viewport;
  waitForSelector?: string | WaitForSelector;
  addScriptTag?: ScriptTag[];
  addStyleTag?: StyleTag[];
  setJavaScriptEnabled?: boolean;
  rejectResourceTypes?: string[];
  rejectRequestPattern?: string[];
  allowResourceTypes?: string[];
  allowRequestPattern?: string[];
}

// ── /api/content ─────────────────────────────────────────────────────────────
export type ContentRequest = BaseRequest;

// ── /api/screenshot ───────────────────────────────────────────────────────────
export interface ScreenshotOptions {
  type?: "png" | "jpeg";
  quality?: number;
  fullPage?: boolean;
  omitBackground?: boolean;
  clip?: { x: number; y: number; width: number; height: number };
  captureBeyondViewport?: boolean;
}

export interface ScreenshotRequest extends BaseRequest {
  screenshotOptions?: ScreenshotOptions;
  selector?: string;
}

// ── /api/pdf ──────────────────────────────────────────────────────────────────
export interface PdfOptions {
  format?: string;
  landscape?: boolean;
  printBackground?: boolean;
  preferCSSPageSize?: boolean;
  scale?: number;
  displayHeaderFooter?: boolean;
  headerTemplate?: string;
  footerTemplate?: string;
  margin?: { top?: string; bottom?: string; left?: string; right?: string };
  timeout?: number;
}

export interface PdfRequest extends BaseRequest {
  pdfOptions?: PdfOptions;
}

// ── /api/markdown ─────────────────────────────────────────────────────────────
export type MarkdownRequest = BaseRequest;

// ── /api/snapshot ─────────────────────────────────────────────────────────────
export interface SnapshotScreenshotOptions {
  fullPage?: boolean;
}

export interface SnapshotRequest extends BaseRequest {
  screenshotOptions?: SnapshotScreenshotOptions;
}

export interface SnapshotResult {
  content: string;
  screenshot: string | null;
}

// ── /api/scrape ───────────────────────────────────────────────────────────────
export interface ScrapeElement {
  selector: string;
}

export interface ScrapeRequest extends BaseRequest {
  elements: ScrapeElement[];
}

export interface ScrapeNodeResult {
  text: string;
  html: string;
  attributes: Array<{ name: string; value: string }>;
  height: number;
  width: number;
  top: number;
  left: number;
}

export interface ScrapeSelectorResult {
  selector: string;
  results: ScrapeNodeResult[];
}

// ── /api/json ─────────────────────────────────────────────────────────────────
export interface CustomAiModel {
  model: string;
  authorization: string;
}

export interface JsonRequest extends BaseRequest {
  prompt?: string;
  response_format?: {
    type: "json_schema";
    schema: Record<string, unknown>;
  };
  custom_ai?: CustomAiModel[];
}

// ── /api/links ────────────────────────────────────────────────────────────────
export interface LinksRequest extends BaseRequest {
  visibleLinksOnly?: boolean;
  excludeExternalLinks?: boolean;
}

// ── D1 cache row ──────────────────────────────────────────────────────────────
export interface PageCacheRow {
  url: string;
  endpoint: string;
  params_hash: string;
  html: string | null;
  markdown: string | null;
  result: string | null;
  title: string | null;
  created_at: number;
}
