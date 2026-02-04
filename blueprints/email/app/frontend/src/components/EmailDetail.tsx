import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Archive,
  Trash2,
  Mail,
  MailOpen,
  Tag,
  MoreVertical,
  Star,
  Reply,
  ReplyAll,
  Forward,
  ChevronDown,
  ChevronUp,
  Paperclip,
  Download,
  Printer,
  ExternalLink,
} from "lucide-react";
import DOMPurify from "dompurify";
import Avatar from "./Avatar";
import { useEmailStore, useLabelStore, useSettingsStore } from "../store";
import * as api from "../api";
import { showToast } from "./Toast";
import type { Email, Attachment } from "../types";

interface EmailDetailProps {
  emailId: string;
}

function formatFullDate(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString("en-US", {
    weekday: "short",
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    hour12: true,
  });
}

function sanitizeHtml(html: string): string {
  return DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      "p", "br", "b", "i", "u", "strong", "em", "a", "ul", "ol", "li",
      "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "pre", "code",
      "span", "div", "table", "tr", "td", "th", "thead", "tbody", "img",
      "hr", "sub", "sup",
    ],
    ALLOWED_ATTR: [
      "href", "target", "rel", "style", "class", "src", "alt", "width",
      "height", "align", "valign", "border", "cellpadding", "cellspacing",
      "colspan", "rowspan",
    ],
    ALLOW_DATA_ATTR: false,
  });
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

interface EmailMessageProps {
  email: Email;
  isExpanded: boolean;
  onToggle: () => void;
  isLast: boolean;
  attachments: Attachment[];
  onDownload: (attachment: Attachment) => void;
  onReplyAll: (email: Email) => void;
}

function EmailMessage({
  email,
  isExpanded,
  onToggle,
  isLast,
  attachments,
  onDownload,
  onReplyAll,
}: EmailMessageProps) {
  const openReply = useEmailStore((s) => s.openReply);
  const openForward = useEmailStore((s) => s.openForward);
  const refreshEmails = useEmailStore((s) => s.refreshEmails);

  const handleStarClick = useCallback(async () => {
    try {
      await api.updateEmail(email.id, { is_starred: !email.is_starred });
      refreshEmails();
    } catch {
      // Handle error silently
    }
  }, [email.id, email.is_starred, refreshEmails]);

  const recipients = [
    ...(email.to_addresses ?? []).map((r) => r.name || r.address),
  ].join(", ");

  const ccRecipients = (email.cc_addresses ?? [])
    .map((r) => r.name || r.address)
    .join(", ");

  return (
    <div className={`${!isLast ? "border-b border-gray-200" : ""}`}>
      {/* Message header */}
      <div
        className="flex cursor-pointer items-start gap-3 px-6 py-3 hover:bg-gray-50"
        onClick={onToggle}
      >
        <Avatar name={email.from_name} email={email.from_address} size={40} />

        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium text-gmail-text-primary">
                {email.from_name || email.from_address}
              </span>
              {isExpanded && (
                <span className="text-xs text-gmail-text-secondary">
                  &lt;{email.from_address}&gt;
                </span>
              )}
            </div>
            <div className="flex items-center gap-1">
              {email.has_attachments && (
                <Paperclip className="h-3.5 w-3.5 text-gmail-text-secondary" />
              )}
              <span className="text-xs text-gmail-text-secondary">
                {formatFullDate(email.received_at)}
              </span>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  handleStarClick();
                }}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
              >
                <Star
                  className={`h-4 w-4 ${
                    email.is_starred
                      ? "fill-gmail-star text-gmail-star"
                      : "text-gray-400"
                  }`}
                />
              </button>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  openReply(email);
                }}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
              >
                <Reply className="h-4 w-4 text-gmail-text-secondary" />
              </button>
              <button
                onClick={(e) => e.stopPropagation()}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
              >
                <MoreVertical className="h-4 w-4 text-gmail-text-secondary" />
              </button>
            </div>
          </div>

          {isExpanded && (
            <div className="mt-0.5 text-xs text-gmail-text-secondary">
              to {recipients}
              {ccRecipients && <span>, cc: {ccRecipients}</span>}
              <button className="ml-1 inline-flex items-center">
                <ChevronDown className="h-3 w-3" />
              </button>
            </div>
          )}

          {!isExpanded && (
            <p className="mt-0.5 truncate text-sm text-gmail-text-secondary">
              {email.snippet}
            </p>
          )}
        </div>
      </div>

      {/* Message body */}
      {isExpanded && (
        <div className="px-6 pb-4">
          <div className="pl-[52px]">
            {email.body_html ? (
              <div
                className="prose prose-sm max-w-none text-sm text-gmail-text-primary"
                dangerouslySetInnerHTML={{
                  __html: sanitizeHtml(email.body_html),
                }}
              />
            ) : (
              <div className="whitespace-pre-wrap text-sm text-gmail-text-primary">
                {email.body_text}
              </div>
            )}

            {/* Attachments */}
            {attachments.length > 0 && (
              <div className="mt-4 border-t border-gray-200 pt-3">
                <div className="flex flex-wrap gap-2">
                  {attachments.map((a) => (
                    <div
                      key={a.id}
                      className="flex items-center gap-2 rounded-2xl border border-gray-300 px-4 py-2 hover:bg-gray-50 cursor-pointer group"
                      onClick={() => onDownload(a)}
                    >
                      <Paperclip className="h-4 w-4 text-gmail-text-secondary" />
                      <div>
                        <span className="text-sm text-gmail-text-primary group-hover:text-blue-600">
                          {a.filename}
                        </span>
                        <span className="text-xs text-gmail-text-secondary ml-2">
                          {formatSize(a.size_bytes)}
                        </span>
                      </div>
                      <button className="ml-2 flex h-6 w-6 items-center justify-center rounded-full hover:bg-gray-200">
                        <Download className="h-3.5 w-3.5 text-gmail-text-secondary group-hover:text-blue-600" />
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Reply/Forward buttons at bottom of last expanded message */}
            {isLast && (
              <div className="mt-6 flex gap-2">
                <button
                  onClick={() => openReply(email)}
                  className="flex items-center gap-2 rounded-full border border-gray-300 px-6 py-2.5 text-sm font-medium text-gmail-text-primary hover:bg-gray-100 hover:shadow-sm"
                >
                  <Reply className="h-4 w-4" />
                  Reply
                </button>
                <button
                  onClick={() => onReplyAll(email)}
                  className="flex items-center gap-2 rounded-full border border-gray-300 px-6 py-2.5 text-sm font-medium text-gmail-text-primary hover:bg-gray-100 hover:shadow-sm"
                >
                  <ReplyAll className="h-4 w-4" />
                  Reply All
                </button>
                <button
                  onClick={() => openForward(email)}
                  className="flex items-center gap-2 rounded-full border border-gray-300 px-6 py-2.5 text-sm font-medium text-gmail-text-primary hover:bg-gray-100 hover:shadow-sm"
                >
                  <Forward className="h-4 w-4" />
                  Forward
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

export default function EmailDetail({ emailId }: EmailDetailProps) {
  const navigate = useNavigate();
  const selectedEmail = useEmailStore((s) => s.selectedEmail);
  const selectEmail = useEmailStore((s) => s.selectEmail);
  const refreshEmails = useEmailStore((s) => s.refreshEmails);
  const openCompose = useEmailStore((s) => s.openCompose);
  const labels = useLabelStore((s) => s.labels);
  const settings = useSettingsStore((s) => s.settings);
  const [email, setEmail] = useState<Email | null>(selectedEmail);
  const [threadEmails, setThreadEmails] = useState<Email[]>([]);
  const [expandedEmails, setExpandedEmails] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);
  const [attachmentMap, setAttachmentMap] = useState<Record<string, Attachment[]>>({});
  const [showLabelMenu, setShowLabelMenu] = useState(false);

  useEffect(() => {
    async function loadEmail() {
      setLoading(true);
      try {
        const fetched = await api.getEmail(emailId);
        setEmail(fetched);
        selectEmail(fetched);

        // Mark as read
        if (!fetched.is_read) {
          await api.updateEmail(emailId, { is_read: true });
        }

        // Try to load thread
        if (fetched.thread_id) {
          try {
            const thread = await api.getThread(fetched.thread_id);
            setThreadEmails(thread.emails ?? [fetched]);
            // Expand the last email by default
            const lastEmail = thread.emails?.[thread.emails.length - 1];
            if (lastEmail) {
              setExpandedEmails(new Set([lastEmail.id]));
            }
          } catch {
            setThreadEmails([fetched]);
            setExpandedEmails(new Set([fetched.id]));
          }
        } else {
          setThreadEmails([fetched]);
          setExpandedEmails(new Set([fetched.id]));
        }
      } catch {
        // Handle error silently
      } finally {
        setLoading(false);
      }
    }

    loadEmail();
  }, [emailId, selectEmail]);

  // Fetch attachments for emails that have them
  useEffect(() => {
    threadEmails.filter((e) => e.has_attachments).forEach(async (threadEmail) => {
      try {
        const atts = await api.listAttachments(threadEmail.id);
        setAttachmentMap((prev) => ({ ...prev, [threadEmail.id]: atts }));
      } catch {
        // ignore
      }
    });
  }, [threadEmails]);

  const handleBack = useCallback(() => {
    selectEmail(null);
    navigate(-1);
  }, [selectEmail, navigate]);

  const handleArchive = useCallback(async () => {
    if (!email) return;
    try {
      await api.batchEmails({ ids: [email.id], action: "archive" });
      showToast("Conversation archived", {
        action: {
          label: "Undo",
          onClick: () =>
            api
              .batchEmails({ ids: [email.id], action: "add_label", label_id: "inbox" })
              .then(refreshEmails),
        },
      });
      refreshEmails();
      handleBack();
    } catch {
      showToast("Failed to archive");
    }
  }, [email, refreshEmails, handleBack]);

  const handleDelete = useCallback(async () => {
    if (!email) return;
    try {
      await api.batchEmails({ ids: [email.id], action: "trash" });
      showToast("Conversation moved to Trash");
      refreshEmails();
      handleBack();
    } catch {
      showToast("Failed to delete");
    }
  }, [email, refreshEmails, handleBack]);

  const handleToggleRead = useCallback(async () => {
    if (!email) return;
    try {
      await api.updateEmail(email.id, { is_read: !email.is_read });
      refreshEmails();
      handleBack();
    } catch {
      // Handle error silently
    }
  }, [email, refreshEmails, handleBack]);

  const handlePrint = useCallback(() => {
    window.print();
  }, []);

  const handleAddLabel = useCallback(
    async (labelId: string) => {
      if (!email) return;
      await api.batchEmails({ ids: [email.id], action: "add_label", label_id: labelId });
      showToast("Label added");
      setShowLabelMenu(false);
      refreshEmails();
    },
    [email, refreshEmails]
  );

  const handleDownload = useCallback(async (attachment: Attachment) => {
    try {
      const resp = await api.downloadAttachment(attachment.id);
      const blob = await resp.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = attachment.filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      showToast("Failed to download attachment");
    }
  }, []);

  const handleReplyAll = useCallback(
    (replyEmail: Email) => {
      const myAddress = settings?.email_address || "me@email.local";

      const toRecipients: { name?: string; address: string }[] = [
        { name: replyEmail.from_name, address: replyEmail.from_address },
      ];
      replyEmail.to_addresses?.forEach((r) => {
        if (r.address.toLowerCase() !== myAddress.toLowerCase()) {
          toRecipients.push(r);
        }
      });
      const ccRecipients = (replyEmail.cc_addresses || []).filter(
        (r) => r.address.toLowerCase() !== myAddress.toLowerCase()
      );

      const quoted = replyEmail.body_html || replyEmail.body_text || "";
      const body = `<br/><div style="border-left:1px solid #ccc;padding-left:12px;margin-left:0;color:#5f6368"><p>On ${new Date(replyEmail.received_at).toLocaleString()}, ${replyEmail.from_name || replyEmail.from_address} wrote:</p>${quoted}</div>`;
      let subj = replyEmail.subject;
      if (!subj.toLowerCase().startsWith("re:")) subj = "Re: " + subj;

      openCompose({
        to: toRecipients,
        cc: ccRecipients,
        subject: subj,
        body_html: body,
        in_reply_to: replyEmail.message_id,
        thread_id: replyEmail.thread_id,
      });
    },
    [settings, openCompose]
  );

  const toggleExpanded = useCallback((id: string) => {
    setExpandedEmails((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gmail-blue border-t-transparent" />
      </div>
    );
  }

  if (!email) {
    return (
      <div className="flex items-center justify-center py-20 text-gmail-text-secondary">
        Email not found
      </div>
    );
  }

  const emailLabels = (email.labels ?? [])
    .map((id) => labels.find((l) => l.id === id))
    .filter(Boolean);

  const userLabels = labels.filter((l) => l.type === "user");

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Actions toolbar */}
      <div className="flex h-10 flex-shrink-0 items-center gap-0.5 border-b border-gmail-border px-2">
        <button
          onClick={handleBack}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Back to inbox"
        >
          <ArrowLeft className="h-5 w-5 text-gmail-text-secondary" />
        </button>
        <button
          onClick={handleArchive}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Archive"
        >
          <Archive className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          onClick={handleDelete}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Delete"
        >
          <Trash2 className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          onClick={handleToggleRead}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title={email.is_read ? "Mark as unread" : "Mark as read"}
        >
          {email.is_read ? (
            <MailOpen className="h-[18px] w-[18px] text-gmail-text-secondary" />
          ) : (
            <Mail className="h-[18px] w-[18px] text-gmail-text-secondary" />
          )}
        </button>
        <div className="relative">
          <button
            onClick={() => setShowLabelMenu(!showLabelMenu)}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
            title="Labels"
          >
            <Tag className="h-[18px] w-[18px] text-gmail-text-secondary" />
          </button>
          {showLabelMenu && (
            <div className="absolute top-full left-0 mt-1 w-48 bg-white shadow-lg rounded-lg border border-gray-200 z-50">
              <div className="py-1">
                <div className="px-3 py-1.5 text-xs font-medium text-gray-500 uppercase">
                  Label as
                </div>
                {userLabels.map((label) => (
                  <button
                    key={label.id}
                    onClick={() => handleAddLabel(label.id)}
                    className="w-full px-3 py-1.5 text-sm text-left hover:bg-gray-50 flex items-center gap-2"
                  >
                    <span
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: label.color || "#9AA0A6" }}
                    />
                    {label.name}
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
        <button
          onClick={handlePrint}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Print"
        >
          <Printer className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Open in new window"
        >
          <ExternalLink className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="More"
        >
          <MoreVertical className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
      </div>

      {/* Email content */}
      <div className="flex-1 overflow-y-auto">
        {/* Subject line */}
        <div className="flex items-center gap-2 px-6 py-4">
          <h1
            className="flex-1 text-xl text-gmail-text-primary"
            style={{ fontFamily: "'Google Sans', sans-serif" }}
          >
            {email.subject || "(no subject)"}
          </h1>
          {threadEmails.length > 1 && (
            <button
              className="flex-shrink-0 text-xs text-gmail-text-secondary hover:text-gmail-text-primary flex items-center gap-1"
              onClick={() => {
                if (expandedEmails.size === threadEmails.length) {
                  const last = threadEmails[threadEmails.length - 1];
                  setExpandedEmails(new Set(last ? [last.id] : []));
                } else {
                  setExpandedEmails(new Set(threadEmails.map((e) => e.id)));
                }
              }}
            >
              {expandedEmails.size === threadEmails.length ? (
                <ChevronUp className="h-4 w-4" />
              ) : (
                <ChevronDown className="h-4 w-4" />
              )}
              {threadEmails.length} messages
            </button>
          )}
        </div>

        {/* Label chips */}
        {emailLabels.length > 0 && (
          <div className="flex flex-wrap gap-1 px-6 pb-2">
            {emailLabels.map(
              (label) =>
                label && (
                  <span
                    key={label.id}
                    className="label-chip"
                    style={{
                      backgroundColor: label.color
                        ? `${label.color}22`
                        : "#e8eaed",
                      color: label.color || "#5f6368",
                    }}
                  >
                    {label.name}
                  </span>
                )
            )}
          </div>
        )}

        {/* Thread messages */}
        <div>
          {threadEmails.map((threadEmail, index) => (
            <EmailMessage
              key={threadEmail.id}
              email={threadEmail}
              isExpanded={expandedEmails.has(threadEmail.id)}
              onToggle={() => toggleExpanded(threadEmail.id)}
              isLast={index === threadEmails.length - 1}
              attachments={attachmentMap[threadEmail.id] || []}
              onDownload={handleDownload}
              onReplyAll={handleReplyAll}
            />
          ))}
        </div>

        {/* Bottom padding */}
        <div className="h-8" />
      </div>
    </div>
  );
}
