import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import {
  ArrowLeft,
  Archive,
  Trash2,
  Mail,
  Tag,
  MoreVertical,
  Star,
  Reply,
  Forward,
  ChevronDown,
  ChevronUp,
  Paperclip,
  Download,
  Printer,
  ExternalLink,
} from "lucide-react";
import Avatar from "./Avatar";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";
import type { Email } from "../types";

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

function basicSanitize(html: string): string {
  // Remove script tags, event handlers, and javascript: URLs
  return html
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, "")
    .replace(/on\w+\s*=\s*("[^"]*"|'[^']*'|[^\s>]*)/gi, "")
    .replace(/javascript\s*:/gi, "");
}

interface EmailMessageProps {
  email: Email;
  isExpanded: boolean;
  onToggle: () => void;
  isLast: boolean;
}

function EmailMessage({
  email,
  isExpanded,
  onToggle,
  isLast,
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
              {!isExpanded && (
                <span className="text-xs text-gmail-text-secondary">
                  &lt;{email.from_address}&gt;
                </span>
              )}
            </div>
            <div className="flex items-center gap-1">
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
                  __html: basicSanitize(email.body_html),
                }}
              />
            ) : (
              <div className="whitespace-pre-wrap text-sm text-gmail-text-primary">
                {email.body_text}
              </div>
            )}

            {/* Attachments */}
            {email.has_attachments && (
              <div className="mt-4 border-t border-gray-200 pt-3">
                <div className="flex flex-wrap gap-2">
                  <div className="flex items-center gap-2 rounded-2xl border border-gray-300 px-4 py-2 hover:bg-gray-50">
                    <Paperclip className="h-4 w-4 text-gmail-text-secondary" />
                    <span className="text-sm text-gmail-text-primary">
                      Attachment
                    </span>
                    <button className="ml-2 flex h-6 w-6 items-center justify-center rounded-full hover:bg-gray-200">
                      <Download className="h-3.5 w-3.5 text-gmail-text-secondary" />
                    </button>
                  </div>
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
  const labels = useLabelStore((s) => s.labels);
  const [email, setEmail] = useState<Email | null>(selectedEmail);
  const [threadEmails, setThreadEmails] = useState<Email[]>([]);
  const [expandedEmails, setExpandedEmails] = useState<Set<string>>(new Set());
  const [loading, setLoading] = useState(false);

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

  const handleBack = useCallback(() => {
    selectEmail(null);
    navigate(-1);
  }, [selectEmail, navigate]);

  const handleArchive = useCallback(async () => {
    if (!email) return;
    try {
      await api.batchEmails({ ids: [email.id], action: "archive" });
      refreshEmails();
      handleBack();
    } catch {
      // Handle error silently
    }
  }, [email, refreshEmails, handleBack]);

  const handleDelete = useCallback(async () => {
    if (!email) return;
    try {
      await api.batchEmails({ ids: [email.id], action: "trash" });
      refreshEmails();
      handleBack();
    } catch {
      // Handle error silently
    }
  }, [email, refreshEmails, handleBack]);

  const handleMarkUnread = useCallback(async () => {
    if (!email) return;
    try {
      await api.updateEmail(email.id, { is_read: false });
      refreshEmails();
      handleBack();
    } catch {
      // Handle error silently
    }
  }, [email, refreshEmails, handleBack]);

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
          onClick={handleMarkUnread}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Mark as unread"
        >
          <Mail className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
          title="Labels"
        >
          <Tag className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
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
            <span className="flex-shrink-0 text-sm text-gmail-text-secondary">
              {threadEmails.length} messages
            </span>
          )}
          <button
            className="flex-shrink-0"
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
              <ChevronUp className="h-5 w-5 text-gmail-text-secondary" />
            ) : (
              <ChevronDown className="h-5 w-5 text-gmail-text-secondary" />
            )}
          </button>
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
            />
          ))}
        </div>

        {/* Bottom padding */}
        <div className="h-8" />
      </div>
    </div>
  );
}
