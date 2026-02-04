import { create } from "zustand";
import type { Email, Label, Settings, ComposeRequest } from "./types";
import * as api from "./api";

interface EmailStore {
  emails: Email[];
  selectedEmail: Email | null;
  selectedEmails: Set<string>;
  currentLabel: string;
  searchQuery: string;
  page: number;
  perPage: number;
  total: number;
  totalPages: number;
  loading: boolean;
  composeOpen: boolean;
  composeData: (Partial<ComposeRequest> & { mode?: string; email_id?: string }) | null;
  composeMode: "new" | "reply" | "reply-all" | "forward";

  fetchEmails: () => Promise<void>;
  selectEmail: (email: Email | null) => void;
  toggleSelect: (id: string) => void;
  selectAll: () => void;
  deselectAll: () => void;
  setLabel: (label: string) => void;
  setSearch: (query: string) => void;
  nextPage: () => void;
  prevPage: () => void;
  openCompose: (data?: Partial<ComposeRequest> & { mode?: string; email_id?: string }) => void;
  closeCompose: () => void;
  openReply: (email: Email) => void;
  openForward: (email: Email) => void;
  refreshEmails: () => Promise<void>;
}

export const useEmailStore = create<EmailStore>((set, get) => ({
  emails: [],
  selectedEmail: null,
  selectedEmails: new Set(),
  currentLabel: "inbox",
  searchQuery: "",
  page: 1,
  perPage: 50,
  total: 0,
  totalPages: 0,
  loading: false,
  composeOpen: false,
  composeData: null,
  composeMode: "new",

  fetchEmails: async () => {
    const state = get();
    set({ loading: true });
    try {
      const response = await api.fetchEmails({
        label: state.currentLabel,
        page: state.page,
        per_page: state.perPage,
        q: state.searchQuery || undefined,
      });
      set({
        emails: response.emails ?? [],
        total: response.total,
        totalPages: response.total_pages,
        loading: false,
      });
    } catch {
      set({ loading: false, emails: [] });
    }
  },

  selectEmail: (email) => set({ selectedEmail: email }),

  toggleSelect: (id) => {
    const selected = new Set(get().selectedEmails);
    if (selected.has(id)) {
      selected.delete(id);
    } else {
      selected.add(id);
    }
    set({ selectedEmails: selected });
  },

  selectAll: () => {
    const ids = new Set(get().emails.map((e) => e.id));
    set({ selectedEmails: ids });
  },

  deselectAll: () => set({ selectedEmails: new Set() }),

  setLabel: (label) =>
    set({ currentLabel: label, page: 1, searchQuery: "", selectedEmail: null }),

  setSearch: (query) =>
    set({ searchQuery: query, page: 1, selectedEmail: null }),

  nextPage: () => {
    const state = get();
    if (state.page < state.totalPages) {
      set({ page: state.page + 1 });
    }
  },

  prevPage: () => {
    const state = get();
    if (state.page > 1) {
      set({ page: state.page - 1 });
    }
  },

  openCompose: (data) =>
    set({
      composeOpen: true,
      composeData: data ?? null,
      composeMode: "new",
    }),

  closeCompose: () =>
    set({ composeOpen: false, composeData: null, composeMode: "new" }),

  openReply: (email) => {
    const replyTo = email.from_address;
    const replyName = email.from_name;
    set({
      composeOpen: true,
      composeMode: "reply",
      composeData: {
        mode: "reply",
        email_id: email.id,
        to: [{ address: replyTo, name: replyName }],
        subject: email.subject.startsWith("Re:")
          ? email.subject
          : `Re: ${email.subject}`,
        in_reply_to: email.message_id,
        thread_id: email.thread_id,
        body_html: `<br/><br/><div style="border-left:1px solid #ccc;padding-left:12px;margin-left:0;color:#666"><p>On ${new Date(email.received_at).toLocaleDateString()}, ${email.from_name} &lt;${email.from_address}&gt; wrote:</p>${email.body_html ?? email.body_text ?? ""}</div>`,
        body_text: "",
      },
    });
  },

  openForward: (email) => {
    set({
      composeOpen: true,
      composeMode: "forward",
      composeData: {
        mode: "forward",
        email_id: email.id,
        to: [],
        subject: email.subject.startsWith("Fwd:")
          ? email.subject
          : `Fwd: ${email.subject}`,
        body_html: `<br/><br/><div style="border-left:1px solid #ccc;padding-left:12px;margin-left:0;color:#666"><p>---------- Forwarded message ----------</p><p>From: ${email.from_name} &lt;${email.from_address}&gt;</p><p>Date: ${new Date(email.received_at).toLocaleDateString()}</p><p>Subject: ${email.subject}</p><p>To: ${email.to_addresses.map((r) => r.address).join(", ")}</p><br/>${email.body_html ?? email.body_text ?? ""}</div>`,
        body_text: "",
      },
    });
  },

  refreshEmails: async () => {
    await get().fetchEmails();
  },
}));

interface LabelStore {
  labels: Label[];
  loading: boolean;
  fetchLabels: () => Promise<void>;
}

export const useLabelStore = create<LabelStore>((set) => ({
  labels: [],
  loading: false,

  fetchLabels: async () => {
    set({ loading: true });
    try {
      const labels = await api.fetchLabels();
      set({ labels: labels ?? [], loading: false });
    } catch {
      set({ loading: false });
    }
  },
}));

interface SettingsStore {
  settings: Settings | null;
  loading: boolean;
  fetchSettings: () => Promise<void>;
  updateSettings: (data: Partial<Settings>) => Promise<void>;
}

export const useSettingsStore = create<SettingsStore>((set, get) => ({
  settings: null,
  loading: false,

  fetchSettings: async () => {
    set({ loading: true });
    try {
      const settings = await api.getSettings();
      set({ settings, loading: false });
    } catch {
      set({ loading: false });
    }
  },

  updateSettings: async (data) => {
    const current = get().settings;
    if (!current) return;
    try {
      const updated = await api.updateSettings({ ...current, ...data });
      set({ settings: updated });
    } catch {
      // Keep current settings on error
    }
  },
}));
