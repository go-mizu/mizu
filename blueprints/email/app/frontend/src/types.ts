export interface Recipient {
  name?: string;
  address: string;
}

export interface Email {
  id: string;
  thread_id: string;
  message_id: string;
  in_reply_to?: string;
  references?: string[];
  from_address: string;
  from_name: string;
  to_addresses: Recipient[];
  cc_addresses?: Recipient[];
  bcc_addresses?: Recipient[];
  subject: string;
  body_text?: string;
  body_html?: string;
  snippet: string;
  is_read: boolean;
  is_starred: boolean;
  is_important: boolean;
  is_draft: boolean;
  is_sent: boolean;
  has_attachments: boolean;
  size_bytes: number;
  labels?: string[];
  snoozed_until?: string;
  scheduled_at?: string;
  is_muted: boolean;
  sent_at?: string;
  received_at: string;
  created_at: string;
  updated_at: string;
}

export interface Thread {
  id: string;
  subject: string;
  snippet: string;
  emails: Email[];
  email_count: number;
  unread_count: number;
  is_starred: boolean;
  is_important: boolean;
  labels: string[];
  last_email_at: string;
}

export interface Attachment {
  id: string;
  email_id: string;
  filename: string;
  content_type: string;
  size_bytes: number;
  created_at: string;
}

export interface Label {
  id: string;
  name: string;
  color?: string;
  type: "system" | "user";
  visible: boolean;
  position: number;
  unread_count: number;
  total_count: number;
  created_at: string;
}

export interface Contact {
  id: string;
  email: string;
  name: string;
  avatar_url?: string;
  is_frequent: boolean;
  last_contacted?: string;
  contact_count: number;
  created_at: string;
}

export interface Settings {
  id: number;
  display_name: string;
  email_address: string;
  signature: string;
  theme: string;
  density: string;
  conversation_view: boolean;
  auto_advance: string;
  undo_send_seconds: number;
}

export interface ComposeRequest {
  to: Recipient[];
  cc?: Recipient[];
  bcc?: Recipient[];
  subject: string;
  body_html: string;
  body_text: string;
  in_reply_to?: string;
  thread_id?: string;
  is_draft: boolean;
}

export interface BatchAction {
  ids: string[];
  action: string;
  label_id?: string;
}

export interface EmailListResponse {
  emails: Email[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface ThreadListResponse {
  threads: Thread[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}
