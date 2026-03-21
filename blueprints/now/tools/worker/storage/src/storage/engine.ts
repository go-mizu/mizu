// ── Storage engine interface ──────────────────────────────────────────
// Abstract, driver-agnostic contract for all storage operations.
// Every mutating operation (write, move, delete) produces a transaction (tx).

// ── Types ────────────────────────────────────────────────────────────

export interface FileEntry {
  name: string;
  type: string;        // MIME type or 'directory'
  size?: number;
  updated_at?: number;
  tx?: number;
  tx_time?: number;
}

export interface FileMeta {
  path: string;
  name: string;
  size: number;
  type: string;
  tx: number;
  tx_time: number;
}

export interface WriteResult {
  tx: number;
  time: number;
  size: number;
}

export interface MutationResult {
  tx: number;
  time: number;
}

export interface DeleteResult extends MutationResult {
  deleted: number;
}

export interface ReadResult {
  body: ReadableStream | ArrayBuffer;
  meta: FileMeta;
}

export interface SearchResult {
  path: string;
  name: string;
  size: number;
  type: string;
  tx: number;
}

export interface StorageEvent {
  tx: number;
  action: "write" | "move" | "delete";
  path: string;
  size: number;
  msg: string | null;
  ts: number;
  meta: Record<string, string> | null;
}

export interface ListOptions {
  prefix?: string;
  limit?: number;
  offset?: number;
}

export interface LogOptions {
  path?: string;
  since_tx?: number;
  limit?: number;
}

// ── Abstract engine ──────────────────────────────────────────────────

export interface StorageEngine {
  /** Write a file. Returns tx number and timestamp. */
  write(
    actor: string,
    path: string,
    body: ArrayBuffer | ReadableStream,
    contentType: string,
    msg?: string,
  ): Promise<WriteResult>;

  /** Move/rename a file. No blob copy — only metadata changes. */
  move(
    actor: string,
    from: string,
    to: string,
    msg?: string,
  ): Promise<MutationResult>;

  /** Delete file(s). Paths ending with / delete recursively. */
  delete(
    actor: string,
    paths: string[],
    msg?: string,
  ): Promise<DeleteResult>;

  /** Read file content + metadata. */
  read(actor: string, path: string): Promise<ReadResult | null>;

  /** Check if a file exists and get metadata (no body). */
  head(actor: string, path: string): Promise<FileMeta | null>;

  /** List files/folders under a prefix. */
  list(
    actor: string,
    opts?: ListOptions,
  ): Promise<{ entries: FileEntry[]; truncated: boolean }>;

  /** Search files by name/path. */
  search(
    actor: string,
    query: string,
    opts?: { limit?: number; prefix?: string },
  ): Promise<SearchResult[]>;

  /** Aggregate storage stats. */
  stats(actor: string): Promise<{ files: number; bytes: number }>;

  /** Get all file path/name pairs (for search indexing). */
  allNames(actor: string): Promise<{ path: string; name: string }[]>;

  /** Get event log for an actor. */
  log(actor: string, opts?: LogOptions): Promise<StorageEvent[]>;

  /** Generate a presigned GET URL for a file. */
  presignRead(
    actor: string,
    path: string,
    expiresIn?: number,
  ): Promise<string | null>;

  /** Generate a presigned PUT URL for uploading. */
  presignUpload(
    actor: string,
    path: string,
    contentType: string,
    expiresIn?: number,
  ): Promise<string>;

  /**
   * Confirm a presigned upload completed.
   * Called after the client PUTs to the presigned URL.
   */
  confirmUpload(
    actor: string,
    path: string,
    msg?: string,
  ): Promise<WriteResult>;

  /** Initiate a multipart upload. Returns upload_id and part URLs. */
  initiateMultipart(
    actor: string,
    path: string,
    contentType: string,
    partCount: number,
  ): Promise<{
    upload_id: string;
    part_urls: string[];
    expires_in: number;
  }>;

  /** Complete a multipart upload by assembling parts. */
  completeMultipart(
    actor: string,
    path: string,
    uploadId: string,
    parts: { part_number: number; etag: string }[],
    msg?: string,
  ): Promise<WriteResult>;

  /** Abort a multipart upload. */
  abortMultipart(
    actor: string,
    path: string,
    uploadId: string,
  ): Promise<void>;
}
