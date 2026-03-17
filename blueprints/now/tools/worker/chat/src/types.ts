export interface Env {
  DB: D1Database;
}

// Hono variables set by middleware
export type Variables = {
  actor: string;
};

// --- Domain types ---

export interface Chat {
  id: string;
  kind: string;
  title: string;
  creator: string;
  peer?: string;       // only for kind: "direct" — the other actor
  created_at: string;  // ISO 8601
}

export interface DmRequest {
  peer: string;
}

export interface Message {
  id: string;
  chat: string;
  actor: string;
  text: string;
  created_at: string; // ISO 8601
}

// --- Request types ---

export interface CreateChatRequest {
  kind: string;
  title?: string;
  visibility?: string;
}

export interface SendMessageRequest {
  text: string;
}

// --- DB row types ---

export interface ChatRow {
  id: string;
  kind: string;
  title: string;
  creator: string;
  visibility: string;
  created_at: number;
}

export interface MemberRow {
  chat_id: string;
  actor: string;
  joined_at: number;
}

export interface MessageRow {
  id: string;
  chat_id: string;
  actor: string;
  text: string;
  created_at: number;
}

// --- Registration & key management ---

export interface RegisterRequest {
  actor: string;
  public_key: string;
}

export interface RotateKeyRequest {
  actor: string;
  recovery_code: string;
  new_public_key: string;
}

export interface RotateRecoveryRequest {
  actor: string;
  recovery_code: string;
}

export interface DeleteActorRequest {
  actor: string;
  recovery_code: string;
}
