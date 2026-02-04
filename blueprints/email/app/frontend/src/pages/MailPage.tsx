import { useEffect, useCallback } from "react";
import { useParams, useNavigate } from "react-router-dom";
import Toolbar from "../components/Toolbar";
import EmailList from "../components/EmailList";
import EmailDetail from "../components/EmailDetail";
import { useEmailStore } from "../store";
import * as api from "../api";
import { showToast } from "../components/Toast";

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
  const emails = useEmailStore((s) => s.emails);
  const page = useEmailStore((s) => s.page);
  const searchQuery = useEmailStore((s) => s.searchQuery);

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

        case "f":
          if (selectedEmail) {
            e.preventDefault();
            openForward(selectedEmail);
          }
          break;

        case "e":
          if (selectedEmail) {
            e.preventDefault();
            api
              .batchEmails({ ids: [selectedEmail.id], action: "archive" })
              .then(() => {
                showToast("Conversation archived");
                selectEmail(null);
                navigate(-1);
                fetchEmails();
              });
          }
          break;

        case "#":
          if (selectedEmail) {
            e.preventDefault();
            api
              .batchEmails({ ids: [selectedEmail.id], action: "trash" })
              .then(() => {
                showToast("Conversation moved to Trash");
                selectEmail(null);
                navigate(-1);
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
          if (selectedEmail) {
            selectEmail(null);
          }
          break;

        case "/":
          e.preventDefault();
          const searchInput = document.querySelector(
            'input[placeholder="Search mail"]'
          ) as HTMLInputElement;
          if (searchInput) searchInput.focus();
          break;
      }
    },
    [
      openCompose,
      selectedEmail,
      openReply,
      openForward,
      selectEmail,
      emails,
      emailId,
      currentLabel,
      navigate,
      fetchEmails,
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
    </div>
  );
}
