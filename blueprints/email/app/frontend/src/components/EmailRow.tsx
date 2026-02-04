import { useCallback } from "react";
import {
  Star,
  ChevronRight,
  Paperclip,
  Archive,
  Trash2,
  MailOpen,
  Clock,
} from "lucide-react";
import type { Email } from "../types";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";

interface EmailRowProps {
  email: Email;
  onClick: () => void;
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  if (isToday) {
    return date.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    });
  }

  const isThisYear = date.getFullYear() === now.getFullYear();
  if (isThisYear) {
    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
    });
  }

  return date.toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export default function EmailRow({ email, onClick }: EmailRowProps) {
  const selectedEmails = useEmailStore((s) => s.selectedEmails);
  const toggleSelect = useEmailStore((s) => s.toggleSelect);
  const refreshEmails = useEmailStore((s) => s.refreshEmails);
  const labels = useLabelStore((s) => s.labels);

  const isSelected = selectedEmails.has(email.id);
  const isUnread = !email.is_read;

  const handleCheckboxClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      toggleSelect(email.id);
    },
    [email.id, toggleSelect]
  );

  const handleStarClick = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.updateEmail(email.id, { is_starred: !email.is_starred });
        refreshEmails();
      } catch {
        // Handle error silently
      }
    },
    [email.id, email.is_starred, refreshEmails]
  );

  const handleImportantClick = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.updateEmail(email.id, {
          is_important: !email.is_important,
        });
        refreshEmails();
      } catch {
        // Handle error silently
      }
    },
    [email.id, email.is_important, refreshEmails]
  );

  const handleArchive = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.batchEmails({ ids: [email.id], action: "archive" });
        refreshEmails();
      } catch {
        // Handle error silently
      }
    },
    [email.id, refreshEmails]
  );

  const handleDelete = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.batchEmails({ ids: [email.id], action: "trash" });
        refreshEmails();
      } catch {
        // Handle error silently
      }
    },
    [email.id, refreshEmails]
  );

  const handleMarkRead = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.updateEmail(email.id, { is_read: !email.is_read });
        refreshEmails();
      } catch {
        // Handle error silently
      }
    },
    [email.id, email.is_read, refreshEmails]
  );

  const emailLabels = (email.labels ?? [])
    .map((id) => labels.find((l) => l.id === id))
    .filter((l) => l && l.type === "user");

  const senderDisplay = email.from_name || email.from_address;

  return (
    <div
      onClick={onClick}
      className={`email-row-hover group flex cursor-pointer items-center border-b border-gray-100 px-2 ${
        isSelected
          ? "bg-[#C2DBFF]"
          : isUnread
            ? "bg-white"
            : "bg-[#F2F6FC]"
      } hover:shadow-[inset_1px_0_0_#dadce0,inset_-1px_0_0_#dadce0,0_1px_2px_0_rgba(60,64,67,.3),0_1px_3px_1px_rgba(60,64,67,.15)]`}
      style={{ height: "40px" }}
    >
      {/* Checkbox */}
      <div
        onClick={handleCheckboxClick}
        className="flex h-10 w-10 flex-shrink-0 items-center justify-center"
      >
        <div
          className={`flex h-[18px] w-[18px] items-center justify-center rounded-sm border ${
            isSelected
              ? "border-gmail-blue bg-gmail-blue"
              : "border-gray-400 bg-white"
          }`}
        >
          {isSelected && (
            <svg
              viewBox="0 0 24 24"
              className="h-3.5 w-3.5 text-white"
              fill="none"
              stroke="currentColor"
              strokeWidth="3"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          )}
        </div>
      </div>

      {/* Star */}
      <button
        onClick={handleStarClick}
        className="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full hover:bg-gray-200"
      >
        <Star
          className={`h-[18px] w-[18px] ${
            email.is_starred
              ? "fill-gmail-star text-gmail-star"
              : "text-gray-400"
          }`}
        />
      </button>

      {/* Important marker */}
      <button
        onClick={handleImportantClick}
        className="flex h-8 w-5 flex-shrink-0 items-center justify-center"
      >
        <ChevronRight
          className={`h-4 w-4 ${
            email.is_important
              ? "fill-gmail-important text-gmail-important"
              : "text-transparent group-hover:text-gray-400"
          }`}
        />
      </button>

      {/* Sender */}
      <div
        className={`w-[200px] flex-shrink-0 truncate pr-4 text-sm ${
          isUnread ? "font-bold text-gmail-text-primary" : "text-gmail-text-primary"
        }`}
      >
        {senderDisplay}
      </div>

      {/* Subject + Snippet + Labels */}
      <div className="flex min-w-0 flex-1 items-center gap-1 overflow-hidden pr-2">
        {/* Label chips */}
        {emailLabels.map(
          (label) =>
            label && (
              <span
                key={label.id}
                className="label-chip flex-shrink-0"
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

        <span
          className={`truncate text-sm ${
            isUnread ? "font-bold text-gmail-text-primary" : "text-gmail-text-primary"
          }`}
        >
          {email.subject || "(no subject)"}
        </span>
        {email.snippet && (
          <span className="truncate text-sm text-gmail-text-snippet">
            {" "}
            - {email.snippet}
          </span>
        )}
      </div>

      {/* Inline actions (shown on hover) */}
      <div className="email-row-actions flex flex-shrink-0 items-center gap-0.5 opacity-0 transition-opacity">
        <button
          onClick={handleArchive}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
          title="Archive"
        >
          <Archive className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          onClick={handleDelete}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
          title="Delete"
        >
          <Trash2 className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          onClick={handleMarkRead}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
          title={isUnread ? "Mark as read" : "Mark as unread"}
        >
          <MailOpen className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
        <button
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
          title="Snooze"
        >
          <Clock className="h-[18px] w-[18px] text-gmail-text-secondary" />
        </button>
      </div>

      {/* Attachment icon */}
      {email.has_attachments && (
        <Paperclip className="mr-2 h-4 w-4 flex-shrink-0 text-gmail-text-secondary" />
      )}

      {/* Date */}
      <div
        className={`email-row-date w-[80px] flex-shrink-0 text-right text-xs transition-opacity ${
          isUnread ? "font-bold text-gmail-text-primary" : "text-gmail-text-secondary"
        }`}
      >
        {formatDate(email.received_at)}
      </div>
    </div>
  );
}
