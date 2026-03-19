export interface Env {
  DB: D1Database;
  BUCKET: R2Bucket;
  RESEND_API_KEY?: string;
  R2_ACCESS_KEY_ID?: string;
  R2_SECRET_ACCESS_KEY?: string;
  CF_ACCOUNT_ID?: string;
  R2_BUCKET_NAME?: string;
}

export interface Variables {
  actor: string;
  authType: "session" | "apikey";
  scopes: string;
  pathPrefix: string;
}

export interface ActorRow {
  actor: string;
  type: "human" | "agent";
  public_key: string | null;
  email: string | null;
  bio: string;
  created_at: number;
}

export interface SessionRow {
  token: string;
  actor: string;
  expires_at: number;
}

export interface BucketRow {
  id: string;
  owner: string;
  name: string;
  public: number;
  file_size_limit: number | null;
  allowed_mime_types: string | null;
  created_at: number;
  updated_at: number;
}

export interface ObjectRow {
  id: string;
  owner: string;
  bucket_id: string;
  path: string;
  name: string;
  content_type: string;
  size: number;
  r2_key: string;
  metadata: string;
  accessed_at: number | null;
  created_at: number;
  updated_at: number;
}

export interface SignedUrlRow {
  id: string;
  owner: string;
  bucket_id: string;
  path: string;
  token: string;
  type: "download" | "upload";
  expires_at: number;
  created_at: number;
}

export interface ApiKeyRow {
  id: string;
  actor: string;
  token_hash: string;
  name: string;
  scopes: string;
  path_prefix: string;
  expires_at: number | null;
  last_used_at: number | null;
  created_at: number;
}

export interface ChallengeRequest {
  actor: string;
}

export interface VerifyRequest {
  challenge_id: string;
  actor: string;
  signature: string;
}
