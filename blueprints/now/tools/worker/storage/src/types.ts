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

export interface ObjectRow {
  id: string;
  owner: string;
  path: string;
  name: string;
  is_folder: number;
  content_type: string;
  size: number;
  r2_key: string;
  starred: number;
  trashed_at: number | null;
  accessed_at: number | null;
  description: string;
  created_at: number;
  updated_at: number;
}

export interface ShareRow {
  id: string;
  object_id: string;
  owner: string;
  grantee: string;
  permission: string;
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

export interface PublicLinkRow {
  id: string;
  object_id: string;
  owner: string;
  token: string;
  permission: string;
  password_hash: string | null;
  expires_at: number | null;
  max_downloads: number | null;
  download_count: number;
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
