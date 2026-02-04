import { useNavigate } from "react-router-dom";
import { Inbox } from "lucide-react";
import EmailRow from "./EmailRow";
import { useEmailStore } from "../store";
import type { Email } from "../types";

export default function EmailList() {
  const emails = useEmailStore((s) => s.emails);
  const loading = useEmailStore((s) => s.loading);
  const selectEmail = useEmailStore((s) => s.selectEmail);
  const currentLabel = useEmailStore((s) => s.currentLabel);
  const navigate = useNavigate();

  const handleClick = (email: Email) => {
    selectEmail(email);
    navigate(`/email/${email.id}`);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gmail-blue border-t-transparent" />
      </div>
    );
  }

  if (emails.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-gmail-text-secondary">
        <Inbox className="mb-4 h-16 w-16 text-gray-300" />
        <p className="text-lg font-medium">
          {currentLabel === "inbox"
            ? "Your inbox is empty"
            : `No messages in ${currentLabel}`}
        </p>
        <p className="mt-1 text-sm">
          {currentLabel === "inbox"
            ? "Emails that arrive here will appear in your inbox."
            : "Messages that match this label will appear here."}
        </p>
      </div>
    );
  }

  return (
    <div className="divide-y divide-transparent">
      {emails.map((email) => (
        <EmailRow
          key={email.id}
          email={email}
          onClick={() => handleClick(email)}
        />
      ))}
    </div>
  );
}
