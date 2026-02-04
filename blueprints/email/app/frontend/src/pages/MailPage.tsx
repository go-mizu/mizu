import { useEffect, useCallback } from "react";
import { useParams } from "react-router-dom";
import Toolbar from "../components/Toolbar";
import EmailList from "../components/EmailList";
import EmailDetail from "../components/EmailDetail";
import { useEmailStore } from "../store";

export default function MailPage() {
  const { labelId, emailId } = useParams<{
    labelId?: string;
    emailId?: string;
  }>();

  const fetchEmails = useEmailStore((s) => s.fetchEmails);
  const setLabel = useEmailStore((s) => s.setLabel);
  const currentLabel = useEmailStore((s) => s.currentLabel);
  const openCompose = useEmailStore((s) => s.openCompose);
  const selectedEmail = useEmailStore((s) => s.selectedEmail);
  const selectEmail = useEmailStore((s) => s.selectEmail);
  const openReply = useEmailStore((s) => s.openReply);
  const page = useEmailStore((s) => s.page);
  const searchQuery = useEmailStore((s) => s.searchQuery);

  // Update label from URL
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
      // Skip if user is typing in an input
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
        case "e":
          // Archive shortcut handled by toolbar
          break;
        case "#":
          // Delete shortcut handled by toolbar
          break;
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
    [openCompose, selectedEmail, openReply, selectEmail]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyboard);
    return () => document.removeEventListener("keydown", handleKeyboard);
  }, [handleKeyboard]);

  // If viewing a specific email
  if (emailId) {
    return (
      <div className="flex h-full flex-col">
        <EmailDetail emailId={emailId} />
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col">
      <Toolbar />
      <div className="flex-1 overflow-y-auto">
        <EmailList />
      </div>
    </div>
  );
}
