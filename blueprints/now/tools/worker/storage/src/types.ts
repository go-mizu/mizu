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
