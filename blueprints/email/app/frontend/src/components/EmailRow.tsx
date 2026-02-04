import { useCallback, useState, useRef, useEffect } from "react";
import {
  Star,
  ChevronRight,
  Paperclip,
  Archive,
  Trash2,
  MailOpen,
  Mail,
  Clock,
  Reply,
  Forward,
  VolumeX,
} from "lucide-react";
import type { Email } from "../types";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";
import { showToast } from "./Toast";

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
  const openReply = useEmailStore((s) => s.openReply);
  const openForward = useEmailStore((s) => s.openForward);
  const labels = useLabelStore((s) => s.labels);

  const [starAnimating, setStarAnimating] = useState(false);
  const [snoozeOpen, setSnoozeOpen] = useState(false);
  const [contextMenu, setContextMenu] = useState<{ x: number; y: number } | null>(null);
  const snoozeRef = useRef<HTMLDivElement>(null);
  const contextRef = useRef<HTMLDivElement>(null);

  // Close menus on outside click
  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (snoozeRef.current && !snoozeRef.current.contains(e.target as Node)) setSnoozeOpen(false);
      if (contextRef.current && !contextRef.current.contains(e.target as Node)) setContextMenu(null);
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const isSelected = selectedEmails.has(email.id);
  const hasAnySelected = selectedEmails.size > 0;
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
      setStarAnimating(true);
      setTimeout(() => setStarAnimating(false), 300);
      try {
        await api.updateEmail(email.id, { is_starred: !email.is_starred });
        refreshEmails();
      } catch {
        showToast("Failed to update star");
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
        showToast("Failed to update importance");
      }
    },
    [email.id, email.is_important, refreshEmails]
  );

  const handleArchive = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.batchEmails({ ids: [email.id], action: "archive" });
        showToast("Conversation archived", {
          action: {
            label: "Undo",
            onClick: () => {
              api
                .batchEmails({ ids: [email.id], action: "unarchive" })
                .then(() => refreshEmails());
            },
          },
        });
        refreshEmails();
      } catch {
        showToast("Failed to archive");
      }
    },
    [email.id, refreshEmails]
  );

  const handleDelete = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.batchEmails({ ids: [email.id], action: "trash" });
        showToast("Conversation moved to Trash", {
          action: {
            label: "Undo",
            onClick: () => {
              api
                .batchEmails({ ids: [email.id], action: "untrash" })
                .then(() => refreshEmails());
            },
          },
        });
        refreshEmails();
      } catch {
        showToast("Failed to delete");
      }
    },
    [email.id, refreshEmails]
  );

  const handleToggleRead = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      try {
        await api.updateEmail(email.id, { is_read: !email.is_read });
        refreshEmails();
      } catch {
        showToast("Failed to update read status");
      }
    },
    [email.id, email.is_read, refreshEmails]
  );

  const handleSnoozeOption = useCallback(
    async (option: string) => {
      setSnoozeOpen(false);
      const now = new Date();
      let until: Date;
      switch (option) {
        case "tomorrow_morning":
          until = new Date(now);
          until.setDate(until.getDate() + 1);
          until.setHours(8, 0, 0, 0);
          break;
        case "tomorrow_afternoon":
          until = new Date(now);
          until.setDate(until.getDate() + 1);
          until.setHours(13, 0, 0, 0);
          break;
        case "this_weekend": {
          until = new Date(now);
          const dayOfWeek = until.getDay();
          const daysUntilSat = (6 - dayOfWeek + 7) % 7 || 7;
          until.setDate(until.getDate() + daysUntilSat);
          until.setHours(9, 0, 0, 0);
          break;
        }
        case "next_week": {
          until = new Date(now);
          const dow = until.getDay();
          const daysUntilMon = (1 - dow + 7) % 7 || 7;
          until.setDate(until.getDate() + daysUntilMon);
          until.setHours(8, 0, 0, 0);
          break;
        }
        default:
          return;
      }
      try {
        await api.snoozeEmail(email.id, until.toISOString());
        const label = option.replace(/_/g, " ").replace(/^\w/, (c) => c.toUpperCase());
        showToast(`Snoozed until ${label}`);
        refreshEmails();
      } catch {
        showToast("Failed to snooze");
      }
    },
    [email.id, refreshEmails]
  );

  const handleSnoozeClick = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      setSnoozeOpen((v) => !v);
    },
    []
  );

  const handleContextMenu = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setContextMenu({ x: e.clientX, y: e.clientY });
    },
    []
  );

  const handleMute = useCallback(
    async () => {
      setContextMenu(null);
      try {
        await api.muteEmail(email.id);
        showToast("Conversation muted");
        refreshEmails();
      } catch {
        showToast("Failed to mute");
      }
    },
    [email.id, refreshEmails]
  );

  // Resolve label objects from IDs
  const emailLabels = (email.labels ?? [])
    .map((id) => labels.find((l) => l.id === id))
    .filter((l) => l && l.type === "user");

  const senderDisplay = email.from_name || email.from_address;

  // Row background color
  const bgClass = isSelected
    ? "bg-[#C2DBFF]"
    : isUnread
      ? "bg-white"
      : "bg-[#F2F6FC]";

  return (
    <div
      onClick={onClick}
      onContextMenu={handleContextMenu}
      className={`email-row-hover group relative flex cursor-pointer items-center border-b border-gray-100 px-2 ${bgClass} hover:shadow-[inset_1px_0_0_#dadce0,inset_-1px_0_0_#dadce0,0_1px_2px_0_rgba(60,64,67,.3),0_1px_3px_1px_rgba(60,64,67,.15)]`}
      style={{ height: "40px" }}
    >
      {/* Checkbox - hidden by default, visible on hover or when any emails are selected */}
      <div
        onClick={handleCheckboxClick}
        className={`flex h-10 w-10 flex-shrink-0 items-center justify-center ${
          hasAnySelected ? "visible" : "invisible group-hover:visible"
        }`}
      >
        <div
          className={`flex h-[18px] w-[18px] items-center justify-center rounded-sm border transition-colors ${
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
          className={`h-[18px] w-[18px] transition-transform ${
            starAnimating ? "scale-125" : "scale-100"
          } ${
            email.is_starred
              ? "fill-gmail-star text-gmail-star"
              : "text-gray-400"
          }`}
          style={{ transitionDuration: "150ms" }}
        />
      </button>

      {/* Important marker */}
      <button
        onClick={handleImportantClick}
        className="flex h-8 w-5 flex-shrink-0 items-center justify-center"
      >
        <ChevronRight
          className={`h-4 w-4 transition-colors ${
            email.is_important
              ? "fill-gmail-important text-gmail-important"
              : "text-transparent group-hover:text-gray-400"
          }`}
        />
      </button>

      {/* Sender */}
      <div
        className={`w-[200px] flex-shrink-0 truncate pr-4 text-sm ${
          isUnread
            ? "font-bold text-gmail-text-primary"
            : "text-gmail-text-primary"
        }`}
      >
        {senderDisplay}
        {/* Thread count slot (visual placeholder for future thread grouping) */}
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
            isUnread
              ? "font-bold text-gmail-text-primary"
              : "text-gmail-text-primary"
          }`}
        >
          {email.subject || "(no subject)"}
        </span>
        {email.snippet && (
          <span className="truncate text-sm text-gmail-text-snippet">
            &nbsp;-&nbsp;{email.snippet}
          </span>
        )}
      </div>

      {/* Hover actions - absolutely positioned, replace date on hover */}
      <div className="email-row-actions absolute right-2 flex flex-shrink-0 items-center gap-0.5 bg-inherit opacity-0 transition-opacity">
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
          onClick={handleToggleRead}
          className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
          title={isUnread ? "Mark as read" : "Mark as unread"}
        >
          {isUnread ? (
            <MailOpen className="h-[18px] w-[18px] text-gmail-text-secondary" />
          ) : (
            <Mail className="h-[18px] w-[18px] text-gmail-text-secondary" />
          )}
        </button>
        <div className="relative" ref={snoozeRef}>
          <button
            onClick={handleSnoozeClick}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-200"
            title="Snooze"
          >
            <Clock className="h-[18px] w-[18px] text-gmail-text-secondary" />
          </button>
          {snoozeOpen && (
            <div className="absolute right-0 top-full z-50 mt-1 w-52 rounded-lg border border-gray-200 bg-white py-1 shadow-lg">
              <div className="px-3 py-1.5 text-xs font-medium text-gray-500">Snooze until</div>
              {([
                { key: "tomorrow_morning", label: "Tomorrow morning", sub: "8:00 AM" },
                { key: "tomorrow_afternoon", label: "Tomorrow afternoon", sub: "1:00 PM" },
                { key: "this_weekend", label: "This weekend", sub: "Sat 9:00 AM" },
                { key: "next_week", label: "Next week", sub: "Mon 8:00 AM" },
              ] as const).map((opt) => (
                <button
                  key={opt.key}
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSnoozeOption(opt.key);
                  }}
                  className="flex w-full items-center justify-between px-3 py-2 text-sm hover:bg-gray-50"
                >
                  <span className="text-gmail-text-primary">{opt.label}</span>
                  <span className="text-xs text-gray-400">{opt.sub}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Context menu */}
      {contextMenu && (
        <div
          ref={contextRef}
          className="fixed z-[100] w-52 rounded-lg border border-gray-200 bg-white py-1 shadow-lg"
          style={{ left: contextMenu.x, top: contextMenu.y }}
        >
          <button onClick={() => { setContextMenu(null); openReply(email); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <Reply className="h-4 w-4 text-gray-500" /> Reply
          </button>
          <button onClick={() => { setContextMenu(null); openForward(email); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <Forward className="h-4 w-4 text-gray-500" /> Forward
          </button>
          <div className="my-1 border-t border-gray-100" />
          <button onClick={(e) => { setContextMenu(null); handleArchive(e); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <Archive className="h-4 w-4 text-gray-500" /> Archive
          </button>
          <button onClick={(e) => { setContextMenu(null); handleDelete(e); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <Trash2 className="h-4 w-4 text-gray-500" /> Delete
          </button>
          <button onClick={(e) => { setContextMenu(null); handleToggleRead(e); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            {isUnread ? <MailOpen className="h-4 w-4 text-gray-500" /> : <Mail className="h-4 w-4 text-gray-500" />}
            {isUnread ? "Mark as read" : "Mark as unread"}
          </button>
          <button onClick={(e) => { setContextMenu(null); handleStarClick(e); }} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <Star className="h-4 w-4 text-gray-500" /> {email.is_starred ? "Unstar" : "Star"}
          </button>
          <button onClick={() => handleMute()} className="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-gray-50">
            <VolumeX className="h-4 w-4 text-gray-500" /> Mute
          </button>
        </div>
      )}

      {/* Attachment icon */}
      {email.has_attachments && (
        <Paperclip className="mr-2 h-4 w-4 flex-shrink-0 text-gmail-text-secondary" />
      )}

      {/* Date - hidden on hover when actions appear */}
      <div
        className={`email-row-date w-[80px] flex-shrink-0 text-right text-xs transition-opacity ${
          isUnread
            ? "font-bold text-gmail-text-primary"
            : "text-gmail-text-secondary"
        }`}
      >
        {formatDate(email.received_at)}
      </div>
    </div>
  );
}
