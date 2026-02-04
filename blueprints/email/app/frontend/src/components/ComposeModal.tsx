import { useState, useRef, useEffect, useCallback } from "react";
import {
  X,
  Minus,
  Maximize2,
  Minimize2,
  Trash2,
  Paperclip,
  Send,
  Clock,
  ChevronDown,
} from "lucide-react";
import { useEmailStore, useLabelStore, useSettingsStore } from "../store";
import {
  sendEmail,
  replyEmail,
  replyAllEmail,
  forwardEmail,
  saveDraft,
  fetchContacts,
  scheduleEmail,
} from "../api";
import { showToast } from "./Toast";
import RichTextEditor from "./RichTextEditor";
import Avatar from "./Avatar";
import type { Recipient, Contact, Attachment } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type ComposeState = "normal" | "minimized" | "maximized";

interface ComposeData {
  mode?: "new" | "reply" | "reply-all" | "forward";
  email_id?: string;
  to?: Recipient[];
  cc?: Recipient[];
  bcc?: Recipient[];
  subject?: string;
  body_html?: string;
  body_text?: string;
  in_reply_to?: string;
  thread_id?: string;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

// ---------------------------------------------------------------------------
// RecipientField sub-component
// ---------------------------------------------------------------------------

interface RecipientFieldProps {
  label: string;
  field: "to" | "cc" | "bcc";
  recipients: Recipient[];
  show?: boolean;
  inputValue: string;
  activeField: "to" | "cc" | "bcc" | null;
  suggestions: Contact[];
  inputRef?: React.RefObject<HTMLInputElement | null>;
  onInputChange: (value: string, field: "to" | "cc" | "bcc") => void;
  onKeyDown: (e: React.KeyboardEvent, field: "to" | "cc" | "bcc") => void;
  onFocus: (field: "to" | "cc" | "bcc") => void;
  onBlur: (field: "to" | "cc" | "bcc") => void;
  onRemove: (field: "to" | "cc" | "bcc", index: number) => void;
  onSelectContact: (contact: Contact, field: "to" | "cc" | "bcc") => void;
  showCc: boolean;
  showBcc: boolean;
  onShowCc: () => void;
  onShowBcc: () => void;
}

function RecipientField({
  label,
  field,
  recipients,
  show,
  inputValue,
  activeField,
  suggestions,
  inputRef,
  onInputChange,
  onKeyDown,
  onFocus,
  onBlur,
  onRemove,
  onSelectContact,
  showCc,
  showBcc,
  onShowCc,
  onShowBcc,
}: RecipientFieldProps) {
  if (show === false) return null;

  return (
    <div className="flex items-start gap-2 border-b border-gray-100 px-4 py-1.5">
      <span className="w-8 pt-1 text-sm text-gray-500">{label}</span>
      <div className="flex flex-1 flex-wrap items-center gap-1">
        {recipients.map((r, i) => (
          <span
            key={`${r.address}-${i}`}
            className="inline-flex items-center gap-1 rounded-full bg-gray-100 px-2 py-0.5 text-sm"
          >
            <span>{r.name || r.address}</span>
            <button
              onClick={() => onRemove(field, i)}
              className="text-gray-400 hover:text-gray-600"
            >
              <X size={14} />
            </button>
          </span>
        ))}
        <div className="relative min-w-[120px] flex-1">
          <input
            ref={field === "to" ? inputRef : undefined}
            type="text"
            value={activeField === field ? inputValue : ""}
            onChange={(e) => onInputChange(e.target.value, field)}
            onKeyDown={(e) => onKeyDown(e, field)}
            onFocus={() => onFocus(field)}
            onBlur={() => onBlur(field)}
            className="w-full bg-transparent py-1 text-sm outline-none"
            placeholder=""
          />
          {/* Contact autocomplete dropdown */}
          {activeField === field && suggestions.length > 0 && (
            <div className="absolute left-0 top-full z-50 max-h-48 w-72 overflow-y-auto rounded-lg border border-gray-200 bg-white shadow-lg">
              {suggestions.map((contact) => (
                <button
                  key={contact.id}
                  onMouseDown={(e) => {
                    e.preventDefault();
                    onSelectContact(contact, field);
                  }}
                  className="flex w-full items-center gap-2 px-3 py-2 text-left hover:bg-gray-50"
                >
                  <Avatar
                    name={contact.name || contact.email}
                    email={contact.email}
                    size={28}
                  />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">
                      {contact.name || contact.email}
                    </div>
                    {contact.name && (
                      <div className="truncate text-xs text-gray-500">
                        {contact.email}
                      </div>
                    )}
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>
      {field === "to" && (
        <div className="flex gap-1 pt-1 text-sm text-gray-500">
          {!showCc && (
            <button onClick={onShowCc} className="hover:text-gray-700">
              Cc
            </button>
          )}
          {!showBcc && (
            <button onClick={onShowBcc} className="hover:text-gray-700">
              Bcc
            </button>
          )}
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// ComposeModal
// ---------------------------------------------------------------------------

export default function ComposeModal() {
  const composeMode = useEmailStore((s) => s.composeMode);
  const composeData = useEmailStore((s) => s.composeData) as ComposeData | null;
  const closeCompose = useEmailStore((s) => s.closeCompose);
  const fetchEmails = useEmailStore((s) => s.fetchEmails);
  const { fetchLabels } = useLabelStore();
  const settings = useSettingsStore((s) => s.settings);

  const [to, setTo] = useState<Recipient[]>([]);
  const [cc, setCc] = useState<Recipient[]>([]);
  const [bcc, setBcc] = useState<Recipient[]>([]);
  const [subject, setSubject] = useState("");
  const [bodyHtml, setBodyHtml] = useState("");
  const [showCc, setShowCc] = useState(false);
  const [showBcc, setShowBcc] = useState(false);
  const [sending, setSending] = useState(false);
  const [windowState, setWindowState] = useState<ComposeState>("normal");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [suggestions, setSuggestions] = useState<Contact[]>([]);
  const [activeField, setActiveField] = useState<
    "to" | "cc" | "bcc" | null
  >(null);
  const [inputValue, setInputValue] = useState("");

  const [scheduleOpen, setScheduleOpen] = useState(false);

  const toInputRef = useRef<HTMLInputElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const scheduleRef = useRef<HTMLDivElement>(null);

  // ---- Populate fields when compose opens ----
  useEffect(() => {
    if (composeMode && composeData) {
      setTo(composeData.to || []);
      setCc(composeData.cc || []);
      setBcc(composeData.bcc || []);
      setSubject(composeData.subject || "");

      // Build initial body: for new messages, auto-insert signature
      let initialBody = composeData.body_html || "";
      if (
        (!composeData.mode || composeData.mode === "new") &&
        settings?.signature &&
        !initialBody
      ) {
        initialBody = `<br/><br/><div style="color:#5f6368">--<br/>${settings.signature}</div>`;
      }
      setBodyHtml(initialBody);

      setShowCc((composeData.cc || []).length > 0);
      setShowBcc((composeData.bcc || []).length > 0);
      setAttachments([]);
      setWindowState("normal");

      if ((composeData.to || []).length === 0) {
        setTimeout(() => toInputRef.current?.focus(), 100);
      }
    }
  }, [composeMode, composeData, settings?.signature]);

  // ---- Contact autocomplete with debounce ----
  const searchContacts = useCallback(
    async (query: string) => {
      if (query.length < 1) {
        setSuggestions([]);
        return;
      }
      try {
        const contacts = await fetchContacts(query);
        setSuggestions(
          contacts.filter(
            (c) =>
              !to.some((r) => r.address === c.email) &&
              !cc.some((r) => r.address === c.email) &&
              !bcc.some((r) => r.address === c.email)
          )
        );
      } catch {
        setSuggestions([]);
      }
    },
    [to, cc, bcc]
  );

  useEffect(() => {
    const timer = setTimeout(() => {
      if (inputValue) searchContacts(inputValue);
    }, 200);
    return () => clearTimeout(timer);
  }, [inputValue, searchContacts]);

  // ---- Bail if compose is not open ----
  if (!composeMode) return null;

  // ---- Recipient helpers ----

  const addRecipient = (field: "to" | "cc" | "bcc", value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return;
    const match = trimmed.match(/^(.+?)\s*<(.+?)>$/);
    const recipient: Recipient =
      match && match[1] && match[2]
        ? { name: match[1].trim(), address: match[2].trim() }
        : { address: trimmed };
    const setter = field === "to" ? setTo : field === "cc" ? setCc : setBcc;
    setter((prev) =>
      prev.some((r) => r.address === recipient.address)
        ? prev
        : [...prev, recipient]
    );
    setInputValue("");
    setSuggestions([]);
  };

  const selectContact = (
    contact: Contact,
    field: "to" | "cc" | "bcc"
  ) => {
    const recipient: Recipient = {
      name: contact.name,
      address: contact.email,
    };
    const setter = field === "to" ? setTo : field === "cc" ? setCc : setBcc;
    setter((prev) =>
      prev.some((r) => r.address === recipient.address)
        ? prev
        : [...prev, recipient]
    );
    setInputValue("");
    setSuggestions([]);
  };

  const removeRecipient = (field: "to" | "cc" | "bcc", index: number) => {
    const setter = field === "to" ? setTo : field === "cc" ? setCc : setBcc;
    setter((prev) => prev.filter((_, i) => i !== index));
  };

  const handleKeyDown = (
    e: React.KeyboardEvent,
    field: "to" | "cc" | "bcc"
  ) => {
    if (["Enter", "Tab", ","].includes(e.key) && inputValue.trim()) {
      e.preventDefault();
      addRecipient(field, inputValue);
    }
  };

  const handleInputChange = (
    value: string,
    field: "to" | "cc" | "bcc"
  ) => {
    setInputValue(value);
    setActiveField(field);
  };

  const handleFieldFocus = (field: "to" | "cc" | "bcc") => {
    setActiveField(field);
  };

  const handleFieldBlur = (field: "to" | "cc" | "bcc") => {
    setTimeout(() => {
      if (inputValue.trim()) addRecipient(field, inputValue);
      setSuggestions([]);
      setActiveField(null);
    }, 200);
  };

  // ---- Attachment handlers ----

  const handleAttach = () => fileInputRef.current?.click();

  const handleFileChange = async (
    e: React.ChangeEvent<HTMLInputElement>
  ) => {
    const files = e.target.files;
    if (!files) return;
    for (const file of Array.from(files)) {
      const attachment: Attachment = {
        id: `local-${Date.now()}-${file.name}`,
        email_id: "",
        filename: file.name,
        content_type: file.type || "application/octet-stream",
        size_bytes: file.size,
        created_at: new Date().toISOString(),
      };
      setAttachments((prev) => [...prev, attachment]);
    }
    e.target.value = "";
  };

  // ---- Send ----

  const handleSend = async () => {
    if (to.length === 0) return;
    setSending(true);
    try {
      const bodyText = bodyHtml
        .replace(/<[^>]*>/g, " ")
        .replace(/\s+/g, " ")
        .trim();
      const data = {
        to,
        cc: showCc ? cc : [],
        bcc: showBcc ? bcc : [],
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        is_draft: false,
        in_reply_to: composeData?.in_reply_to || "",
        thread_id: composeData?.thread_id || "",
      };

      const mode = composeData?.mode || composeMode;
      if (mode === "reply" && composeData?.email_id) {
        await replyEmail(composeData.email_id, data);
      } else if (mode === "reply-all" && composeData?.email_id) {
        await replyAllEmail(composeData.email_id, data);
      } else if (mode === "forward" && composeData?.email_id) {
        await forwardEmail(composeData.email_id, data);
      } else {
        await sendEmail(data);
      }

      showToast("Message sent", {
        action: {
          label: "Undo",
          onClick: () =>
            showToast("Undo not available for this message"),
        },
      });
      closeCompose();
      fetchEmails();
      fetchLabels();
    } catch {
      showToast("Failed to send message");
    } finally {
      setSending(false);
    }
  };

  // ---- Schedule send ----

  const handleScheduleSend = async (option: string) => {
    setScheduleOpen(false);
    if (to.length === 0) return;
    setSending(true);
    try {
      const bodyText = bodyHtml.replace(/<[^>]*>/g, " ").replace(/\s+/g, " ").trim();
      const data = {
        to,
        cc: showCc ? cc : [],
        bcc: showBcc ? bcc : [],
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        is_draft: true,
        in_reply_to: composeData?.in_reply_to || "",
        thread_id: composeData?.thread_id || "",
      };

      const result = await saveDraft(data);

      const now = new Date();
      let sendAt: Date;
      switch (option) {
        case "tomorrow_morning":
          sendAt = new Date(now);
          sendAt.setDate(sendAt.getDate() + 1);
          sendAt.setHours(8, 0, 0, 0);
          break;
        case "tomorrow_afternoon":
          sendAt = new Date(now);
          sendAt.setDate(sendAt.getDate() + 1);
          sendAt.setHours(13, 0, 0, 0);
          break;
        case "monday_morning": {
          sendAt = new Date(now);
          const dow = sendAt.getDay();
          const daysUntilMon = (1 - dow + 7) % 7 || 7;
          sendAt.setDate(sendAt.getDate() + daysUntilMon);
          sendAt.setHours(8, 0, 0, 0);
          break;
        }
        default:
          return;
      }

      const emailResult = result as { email?: { id: string }; id?: string };
      const draftId = emailResult?.email?.id || emailResult?.id;
      if (draftId) {
        await scheduleEmail(draftId, sendAt.toISOString());
      }

      showToast(`Send scheduled for ${sendAt.toLocaleDateString("en-US", { weekday: "short", month: "short", day: "numeric" })} at ${sendAt.toLocaleTimeString("en-US", { hour: "numeric", minute: "2-digit" })}`);
      closeCompose();
      fetchEmails();
      fetchLabels();
    } catch {
      showToast("Failed to schedule send");
    } finally {
      setSending(false);
    }
  };

  // ---- Discard ----

  const handleDiscard = () => {
    closeCompose();
    showToast("Draft discarded");
  };

  // ---- Save draft on close (X) ----

  const handleSaveDraft = async () => {
    try {
      const bodyText = bodyHtml
        .replace(/<[^>]*>/g, " ")
        .replace(/\s+/g, " ")
        .trim();
      await saveDraft({
        to,
        cc,
        bcc,
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        is_draft: true,
        in_reply_to: composeData?.in_reply_to || "",
        thread_id: composeData?.thread_id || "",
      });
      showToast("Draft saved");
    } catch {
      // ignore
    }
    closeCompose();
  };

  // ---- Title ----

  const title =
    composeData?.mode === "reply" || composeMode === "reply"
      ? "Reply"
      : composeData?.mode === "reply-all"
        ? "Reply All"
        : composeData?.mode === "forward" || composeMode === "forward"
          ? "Forward"
          : "New Message";

  // ---- Position / size classes ----

  const positionClass =
    windowState === "maximized" ? "inset-4" : "bottom-0 right-4 w-[560px]";

  const heightStyle: React.CSSProperties =
    windowState === "minimized"
      ? { height: "auto" }
      : windowState === "maximized"
        ? {}
        : { height: 520 };

  // ---- Render ----

  return (
    <div
      className={`compose-animate compose-shadow fixed ${positionClass} z-50 flex flex-col rounded-t-lg bg-white`}
      style={heightStyle}
    >
      {/* ============================================================
         Dark header bar
         ============================================================ */}
      <div
        className="flex cursor-pointer items-center justify-between rounded-t-lg bg-[#404040] px-4 py-2"
        onClick={() =>
          windowState === "minimized" && setWindowState("normal")
        }
      >
        <span className="text-sm font-medium text-white">{title}</span>
        <div className="flex items-center gap-1">
          <button
            onClick={(e) => {
              e.stopPropagation();
              setWindowState(
                windowState === "minimized" ? "normal" : "minimized"
              );
            }}
            className="rounded p-1 text-white hover:bg-white/10"
          >
            <Minus size={16} />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              setWindowState(
                windowState === "maximized" ? "normal" : "maximized"
              );
            }}
            className="rounded p-1 text-white hover:bg-white/10"
          >
            {windowState === "maximized" ? (
              <Minimize2 size={16} />
            ) : (
              <Maximize2 size={16} />
            )}
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              handleSaveDraft();
            }}
            className="rounded p-1 text-white hover:bg-white/10"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* ============================================================
         Body (hidden when minimized)
         ============================================================ */}
      {windowState !== "minimized" && (
        <>
          {/* To / Cc / Bcc fields */}
          <RecipientField
            label="To"
            field="to"
            recipients={to}
            inputValue={inputValue}
            activeField={activeField}
            suggestions={suggestions}
            inputRef={toInputRef}
            onInputChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onFocus={handleFieldFocus}
            onBlur={handleFieldBlur}
            onRemove={removeRecipient}
            onSelectContact={selectContact}
            showCc={showCc}
            showBcc={showBcc}
            onShowCc={() => setShowCc(true)}
            onShowBcc={() => setShowBcc(true)}
          />
          <RecipientField
            label="Cc"
            field="cc"
            recipients={cc}
            show={showCc}
            inputValue={inputValue}
            activeField={activeField}
            suggestions={suggestions}
            onInputChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onFocus={handleFieldFocus}
            onBlur={handleFieldBlur}
            onRemove={removeRecipient}
            onSelectContact={selectContact}
            showCc={showCc}
            showBcc={showBcc}
            onShowCc={() => setShowCc(true)}
            onShowBcc={() => setShowBcc(true)}
          />
          <RecipientField
            label="Bcc"
            field="bcc"
            recipients={bcc}
            show={showBcc}
            inputValue={inputValue}
            activeField={activeField}
            suggestions={suggestions}
            onInputChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onFocus={handleFieldFocus}
            onBlur={handleFieldBlur}
            onRemove={removeRecipient}
            onSelectContact={selectContact}
            showCc={showCc}
            showBcc={showBcc}
            onShowCc={() => setShowCc(true)}
            onShowBcc={() => setShowBcc(true)}
          />

          {/* Subject */}
          <input
            type="text"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="Subject"
            className="border-b border-gray-100 px-4 py-2 text-sm outline-none"
          />

          {/* Rich text editor */}
          <div className="flex-1 overflow-y-auto">
            <RichTextEditor
              content={bodyHtml}
              onChange={setBodyHtml}
              placeholder="Compose your email..."
              autoFocus={to.length > 0}
            />
          </div>

          {/* Attachment list */}
          {attachments.length > 0 && (
            <div className="flex flex-wrap gap-2 border-t border-gray-100 px-4 py-2">
              {attachments.map((a) => (
                <div
                  key={a.id}
                  className="flex items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-1.5 text-sm"
                >
                  <Paperclip size={14} className="text-gray-400" />
                  <span className="max-w-[150px] truncate">
                    {a.filename}
                  </span>
                  <span className="text-xs text-gray-400">
                    {formatSize(a.size_bytes)}
                  </span>
                  <button
                    onClick={() =>
                      setAttachments((prev) =>
                        prev.filter((at) => at.id !== a.id)
                      )
                    }
                    className="text-gray-400 hover:text-gray-600"
                  >
                    <X size={14} />
                  </button>
                </div>
              ))}
            </div>
          )}

          {/* ============================================================
             Footer: Send, Attach, Discard
             ============================================================ */}
          <div className="flex items-center gap-2 border-t border-gray-100 px-4 py-2">
            <div className="relative flex" ref={scheduleRef}>
              <button
                onClick={handleSend}
                disabled={sending || to.length === 0}
                className="flex items-center gap-2 rounded-l-full bg-[#1A73E8] px-5 py-2 text-sm font-medium text-white hover:bg-[#1765CC] hover:shadow-md disabled:cursor-not-allowed disabled:opacity-50"
              >
                {sending ? (
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
                ) : (
                  <Send size={16} />
                )}
                Send
              </button>
              <button
                onClick={() => setScheduleOpen((v) => !v)}
                disabled={sending || to.length === 0}
                className="flex items-center rounded-r-full border-l border-blue-400/30 bg-[#1A73E8] px-2 py-2 text-white hover:bg-[#1765CC] disabled:cursor-not-allowed disabled:opacity-50"
                title="Schedule send"
              >
                <ChevronDown size={14} />
              </button>
              {scheduleOpen && (
                <div className="absolute bottom-full left-0 z-50 mb-1 w-56 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
                  <div className="px-3 py-1.5 text-xs font-medium text-gray-500">
                    <Clock size={12} className="mr-1 inline" />
                    Schedule send
                  </div>
                  {([
                    { key: "tomorrow_morning", label: "Tomorrow morning", sub: "8:00 AM" },
                    { key: "tomorrow_afternoon", label: "Tomorrow afternoon", sub: "1:00 PM" },
                    { key: "monday_morning", label: "Monday morning", sub: "8:00 AM" },
                  ] as const).map((opt) => (
                    <button
                      key={opt.key}
                      onClick={() => handleScheduleSend(opt.key)}
                      className="flex w-full items-center justify-between px-3 py-2 text-sm hover:bg-gray-50"
                    >
                      <span>{opt.label}</span>
                      <span className="text-xs text-gray-400">{opt.sub}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>

            <input
              ref={fileInputRef}
              type="file"
              multiple
              className="hidden"
              onChange={handleFileChange}
            />
            <button
              onClick={handleAttach}
              className="rounded-full p-2 text-gray-600 hover:bg-gray-100"
              title="Attach files"
            >
              <Paperclip size={20} />
            </button>

            <div className="flex-1" />

            <button
              onClick={handleDiscard}
              className="rounded-full p-2 text-gray-600 hover:bg-gray-100"
              title="Discard"
            >
              <Trash2 size={20} />
            </button>
          </div>
        </>
      )}
    </div>
  );
}
