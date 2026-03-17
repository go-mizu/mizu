export interface Env {
  DB: D1Database;
}

// Hono variables set by middleware
export type Variables = {
  actor: string;
};

// --- Domain types ---

export interface Actor {
  actor: string;
  type: "human" | "agent";
  created_at: string;
}

export interface Chat {
  id: string;
  kind: "direct" | "room";
  title: string;
  created_at: string;
  last_message?: MessageSummary;
  unread_count?: number;
}

export interface MessageSummary {
  id: string;
  text: string;
  actor: string;
  created_at: string;
}

export interface Message {
  id: string;
  chat_id: string;
  actor: string;
  text: string;
  created_at: string;
}

export interface Member {
  actor: string;
  role: string;
}

// --- DB row types ---

export interface ActorRow {
  actor: string;
  type: string;
  public_key: string;
  created_at: number;
}

export interface ChallengeRow {
  id: string;
  actor: string;
  nonce: string;
  expires_at: number;
}

export interface SessionRow {
  token: string;
  actor: string;
  expires_at: number;
}

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
  role: string;
  joined_at: number;
}

export interface MessageRow {
  id: string;
  chat_id: string;
  actor: string;
  text: string;
  client_id: string | null;
  created_at: number;
}

// --- Request types ---

export interface RegisterActorRequest {
  actor: string;
  public_key: string;
  type: "human" | "agent";
}

export interface ChallengeRequest {
  actor: string;
}

export interface VerifyRequest {
  challenge_id: string;
  actor: string;
  signature: string;
}

export interface CreateChatRequest {
  kind: "direct" | "room";
  title?: string;
  peer?: string;
}

export interface SendMessageRequest {
  to?: string;
  chat_id?: string;
  text: string;
  client_id?: string;
}

export interface SendMessageExplicitRequest {
  text: string;
  client_id?: string;
}

export interface AddMemberRequest {
  actor: string;
}

export interface MagicLinkRequest {
  email: string;
}

export interface MagicTokenRow {
  token: string;
  email: string;
  actor: string | null;
  expires_at: number;
}
