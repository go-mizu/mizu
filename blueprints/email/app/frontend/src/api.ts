import type {
  Email,
  EmailListResponse,
  ThreadListResponse,
  Thread,
  Label,
  Contact,
  Settings,
  ComposeRequest,
  BatchAction,
} from "./types";

const API_BASE = "/api";

async function request<T>(
  path: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...options?.headers,
    },
  });
  if (!res.ok) {
    const text = await res.text().catch(() => "Unknown error");
    throw new Error(`API error ${res.status}: ${text}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

// Emails
export async function fetchEmails(params: {
  label?: string;
  page?: number;
  per_page?: number;
  q?: string;
}): Promise<EmailListResponse> {
  const searchParams = new URLSearchParams();
  if (params.label) searchParams.set("label", params.label);
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.q) searchParams.set("q", params.q);
  const qs = searchParams.toString();
  return request<EmailListResponse>(`/emails${qs ? `?${qs}` : ""}`);
}

export async function getEmail(id: string): Promise<Email> {
  return request<Email>(`/emails/${id}`);
}

export async function createEmail(data: ComposeRequest): Promise<Email> {
  return request<Email>("/emails", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateEmail(
  id: string,
  data: Partial<Email>
): Promise<Email> {
  return request<Email>(`/emails/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteEmail(
  id: string,
  permanent = false
): Promise<void> {
  const qs = permanent ? "?permanent=true" : "";
  return request<void>(`/emails/${id}${qs}`, {
    method: "DELETE",
  });
}

export async function replyEmail(
  id: string,
  data: ComposeRequest
): Promise<Email> {
  return request<Email>(`/emails/${id}/reply`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function forwardEmail(
  id: string,
  data: ComposeRequest
): Promise<Email> {
  return request<Email>(`/emails/${id}/forward`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function batchEmails(action: BatchAction): Promise<void> {
  return request<void>("/emails/batch", {
    method: "POST",
    body: JSON.stringify(action),
  });
}

export async function searchEmails(
  query: string,
  page = 1,
  perPage = 50
): Promise<EmailListResponse> {
  return fetchEmails({ q: query, page, per_page: perPage });
}

// Threads
export async function fetchThreads(params: {
  label?: string;
  page?: number;
  per_page?: number;
  q?: string;
}): Promise<ThreadListResponse> {
  const searchParams = new URLSearchParams();
  if (params.label) searchParams.set("label", params.label);
  if (params.page) searchParams.set("page", String(params.page));
  if (params.per_page) searchParams.set("per_page", String(params.per_page));
  if (params.q) searchParams.set("q", params.q);
  const qs = searchParams.toString();
  return request<ThreadListResponse>(`/threads${qs ? `?${qs}` : ""}`);
}

export async function getThread(id: string): Promise<Thread> {
  return request<Thread>(`/threads/${id}`);
}

// Labels
export async function fetchLabels(): Promise<Label[]> {
  return request<Label[]>("/labels");
}

export async function createLabel(
  data: Partial<Label>
): Promise<Label> {
  return request<Label>("/labels", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateLabel(
  id: string,
  data: Partial<Label>
): Promise<Label> {
  return request<Label>(`/labels/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteLabel(id: string): Promise<void> {
  return request<void>(`/labels/${id}`, {
    method: "DELETE",
  });
}

// Contacts
export async function fetchContacts(query?: string): Promise<Contact[]> {
  const qs = query ? `?q=${encodeURIComponent(query)}` : "";
  return request<Contact[]>(`/contacts${qs}`);
}

export async function createContact(
  data: Partial<Contact>
): Promise<Contact> {
  return request<Contact>("/contacts", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function updateContact(
  id: string,
  data: Partial<Contact>
): Promise<Contact> {
  return request<Contact>(`/contacts/${id}`, {
    method: "PATCH",
    body: JSON.stringify(data),
  });
}

export async function deleteContact(id: string): Promise<void> {
  return request<void>(`/contacts/${id}`, {
    method: "DELETE",
  });
}

// Settings
export async function getSettings(): Promise<Settings> {
  return request<Settings>("/settings");
}

export async function updateSettings(
  data: Partial<Settings>
): Promise<Settings> {
  return request<Settings>("/settings", {
    method: "PUT",
    body: JSON.stringify(data),
  });
}

// Drafts
export async function saveDraft(data: ComposeRequest): Promise<Email> {
  return createEmail({ ...data, is_draft: true });
}

export async function updateDraft(
  id: string,
  data: Partial<ComposeRequest>
): Promise<Email> {
  return request<Email>(`/emails/${id}`, {
    method: "PATCH",
    body: JSON.stringify({ ...data, is_draft: true }),
  });
}

export async function deleteDraft(id: string): Promise<void> {
  return deleteEmail(id, true);
}

export async function sendDraft(id: string): Promise<Email> {
  return request<Email>(`/emails/${id}/send`, {
    method: "POST",
  });
}
