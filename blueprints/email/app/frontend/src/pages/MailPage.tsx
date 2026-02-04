import { useEffect, useCallback, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { X } from "lucide-react";
import Toolbar from "../components/Toolbar";
import EmailList from "../components/EmailList";
import EmailDetail from "../components/EmailDetail";
import { useEmailStore } from "../store";
import * as api from "../api";
import { showToast } from "../components/Toast";

// Keyboard shortcuts help data
const SHORTCUT_GROUPS = [
  {
    title: "Navigation",
    shortcuts: [
      { keys: "j", desc: "Newer conversation" },
      { keys: "k", desc: "Older conversation" },
      { keys: "u", desc: "Back to list" },
      { keys: "/", desc: "Search" },
      { keys: "Esc", desc: "Deselect" },
    ],
  },
  {
    title: "Actions",
    shortcuts: [
      { keys: "c", desc: "Compose" },
      { keys: "r", desc: "Reply" },
      { keys: "f", desc: "Forward" },
      { keys: "a", desc: "Reply all" },
      { keys: "e", desc: "Archive" },
      { keys: "#", desc: "Delete" },
      { keys: "s", desc: "Star/unstar" },
    ],
  },
  {
    title: "Selection",
    shortcuts: [
      { keys: "x", desc: "Select conversation" },
      { keys: "Shift+I", desc: "Mark as read" },
      { keys: "Shift+U", desc: "Mark as unread" },
      { keys: "z", desc: "Undo last action" },
      { keys: "?", desc: "Show shortcuts" },
    ],
  },
];

function ShortcutsModal({ onClose }: { onClose: () => void }) {
  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/40" onClick={onClose}>
      <div
        className="max-h-[80vh] w-[560px] overflow-y-auto rounded-2xl bg-white p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-medium text-gmail-text-primary">Keyboard shortcuts</h2>
          <button onClick={onClose} className="rounded-full p-1 hover:bg-gray-100">
            <X className="h-5 w-5 text-gray-500" />
          </button>
        </div>
        <div className="grid grid-cols-3 gap-6">
          {SHORTCUT_GROUPS.map((group) => (
            <div key={group.title}>
              <h3 className="mb-2 text-sm font-medium text-gmail-text-secondary">{group.title}</h3>
              <div className="space-y-1.5">
                {group.shortcuts.map((s) => (
                  <div key={s.keys} className="flex items-center justify-between gap-3">
                    <kbd className="inline-flex min-w-[28px] items-center justify-center rounded border border-gray-300 bg-gray-50 px-1.5 py-0.5 font-mono text-xs text-gmail-text-primary">
                      {s.keys}
                    </kbd>
                    <span className="flex-1 text-sm text-gmail-text-secondary">{s.desc}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default function MailPage() {
  const { labelId, emailId } = useParams<{
    labelId?: string;
    emailId?: string;
  }>();
  const navigate = useNavigate();

  const fetchEmails = useEmailStore((s) => s.fetchEmails);
  const setLabel = useEmailStore((s) => s.setLabel);
  const currentLabel = useEmailStore((s) => s.currentLabel);
  const openCompose = useEmailStore((s) => s.openCompose);
  const selectedEmail = useEmailStore((s) => s.selectedEmail);
  const selectEmail = useEmailStore((s) => s.selectEmail);
  const openReply = useEmailStore((s) => s.openReply);
  const openForward = useEmailStore((s) => s.openForward);
  const toggleSelect = useEmailStore((s) => s.toggleSelect);
  const emails = useEmailStore((s) => s.emails);
  const page = useEmailStore((s) => s.page);
  const searchQuery = useEmailStore((s) => s.searchQuery);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const [lastAction, setLastAction] = useState<{ ids: string[]; action: string; label?: string } | null>(null);

  // Update label from URL param
  useEffect(() => {
    const newLabel = labelId ?? "inbox";
    if (newLabel !== currentLabel) {
      setLabel(newLabel);
    }
  }, [labelId, currentLabel, setLabel]);

  // Fetch emails when label, page, or search changes
  useEffect(() => {
    fetchEmails();
  }, [currentLabel, page, searchQuery, fetchEmails]);

  // Keyboard shortcuts
  const handleKeyboard = useCallback(
    (e: KeyboardEvent) => {
      // Skip if user is typing in an input, textarea, or contenteditable
      const target = e.target as HTMLElement;
      if (
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.isContentEditable
      ) {
        return;
      }

      // Shift combinations
      if (e.shiftKey) {
        switch (e.key) {
          case "I":
            // Mark as read
            if (selectedEmail) {
              e.preventDefault();
              api.updateEmail(selectedEmail.id, { is_read: true }).then(() => {
                showToast("Marked as read");
                fetchEmails();
              });
            }
            return;
          case "U":
            // Mark as unread
            if (selectedEmail) {
              e.preventDefault();
              api.updateEmail(selectedEmail.id, { is_read: false }).then(() => {
                showToast("Marked as unread");
                fetchEmails();
              });
            }
            return;
        }
      }

      switch (e.key) {
        case "c":
          e.preventDefault();
          openCompose();
          break;

        case "r":
          if (selectedEmail) {
            e.preventDefault();
            openReply(selectedEmail);
          }
          break;

        case "a":
          // Archive from inbox (like Gmail 'a' for reply-all in detail, but archive in list)
          if (selectedEmail && !emailId) {
            e.preventDefault();
            const action = { ids: [selectedEmail.id], action: "archive" };
            setLastAction(action);
            api.batchEmails(action).then(() => {
              showToast("Conversation archived", {
                action: {
                  label: "Undo",
                  onClick: () => {
                    api.batchEmails({ ids: [selectedEmail.id], action: "unarchive" }).then(() => fetchEmails());
                  },
                },
              });
              selectEmail(null);
              fetchEmails();
            });
          }
          break;

        case "f":
          if (selectedEmail) {
            e.preventDefault();
            openForward(selectedEmail);
          }
          break;

        case "e":
          if (selectedEmail) {
            e.preventDefault();
            const action = { ids: [selectedEmail.id], action: "archive" as const };
            setLastAction(action);
            api
              .batchEmails(action)
              .then(() => {
                showToast("Conversation archived");
                selectEmail(null);
                if (emailId) navigate(-1);
                fetchEmails();
              });
          }
          break;

        case "#":
          if (selectedEmail) {
            e.preventDefault();
            const action = { ids: [selectedEmail.id], action: "trash" as const };
            setLastAction(action);
            api
              .batchEmails(action)
              .then(() => {
                showToast("Conversation moved to Trash");
                selectEmail(null);
                if (emailId) navigate(-1);
                fetchEmails();
              });
          }
          break;

        case "s":
          if (selectedEmail) {
            e.preventDefault();
            api
              .updateEmail(selectedEmail.id, {
                is_starred: !selectedEmail.is_starred,
              })
              .then(() => {
                fetchEmails();
              });
          }
          break;

        case "x":
          // Toggle select current email
          if (selectedEmail) {
            e.preventDefault();
            toggleSelect(selectedEmail.id);
          }
          break;

        case "u":
          // Back to list
          if (emailId) {
            e.preventDefault();
            selectEmail(null);
            navigate(-1);
          }
          break;

        case "z":
          // Undo last action
          if (lastAction) {
            e.preventDefault();
            const undoAction = lastAction.action === "archive" ? "unarchive"
              : lastAction.action === "trash" ? "untrash"
              : null;
            if (undoAction) {
              api.batchEmails({ ids: lastAction.ids, action: undoAction }).then(() => {
                showToast("Action undone");
                setLastAction(null);
                fetchEmails();
              });
            }
          }
          break;

        case "?":
          e.preventDefault();
          setShowShortcuts(true);
          break;

        case "j": {
          // Move to next email in list
          e.preventDefault();
          if (!emailId && emails.length > 0) {
            const currentIdx = selectedEmail
              ? emails.findIndex((em) => em.id === selectedEmail.id)
              : -1;
            const nextIdx = Math.min(currentIdx + 1, emails.length - 1);
            const nextEmail = emails[nextIdx];
            if (nextEmail) {
              selectEmail(nextEmail);
            }
          }
          break;
        }

        case "k": {
          // Move to previous email in list
          e.preventDefault();
          if (!emailId && emails.length > 0) {
            const currentIdx = selectedEmail
              ? emails.findIndex((em) => em.id === selectedEmail.id)
              : emails.length;
            const prevIdx = Math.max(currentIdx - 1, 0);
            const prevEmail = emails[prevIdx];
            if (prevEmail) {
              selectEmail(prevEmail);
            }
          }
          break;
        }

        case "Escape":
          if (showShortcuts) {
            setShowShortcuts(false);
          } else if (selectedEmail) {
            selectEmail(null);
          }
          break;

        case "/":
          e.preventDefault();
          {
            const searchInput = document.querySelector(
              'input[placeholder="Search mail"]'
            ) as HTMLInputElement;
            if (searchInput) searchInput.focus();
          }
          break;
      }
    },
    [
      openCompose,
      selectedEmail,
      openReply,
      openForward,
      selectEmail,
      toggleSelect,
      emails,
      emailId,
      currentLabel,
      navigate,
      fetchEmails,
      showShortcuts,
      lastAction,
    ]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyboard);
    return () => document.removeEventListener("keydown", handleKeyboard);
  }, [handleKeyboard]);

  // If URL has emailId param, show full EmailDetail view
  if (emailId) {
    return (
      <div className="flex h-full flex-col">
        <EmailDetail emailId={emailId} />
        {showShortcuts && <ShortcutsModal onClose={() => setShowShortcuts(false)} />}
      </div>
    );
  }

  // Otherwise show Toolbar + EmailList
  return (
    <div className="flex h-full flex-col">
      <Toolbar />
      <div className="flex-1 overflow-y-auto">
        <EmailList />
      </div>
      {showShortcuts && <ShortcutsModal onClose={() => setShowShortcuts(false)} />}
    </div>
  );
}
