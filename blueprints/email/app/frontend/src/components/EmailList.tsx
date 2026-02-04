import { useNavigate } from "react-router-dom";
import { Inbox } from "lucide-react";
import EmailRow from "./EmailRow";
import { useEmailStore } from "../store";
import type { Email } from "../types";

function SkeletonRow() {
  return (
    <div className="flex h-10 items-center px-2" style={{ height: "40px" }}>
      {/* Checkbox skeleton */}
      <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center">
        <div className="h-[18px] w-[18px] animate-pulse rounded-sm bg-gray-200" />
      </div>
      {/* Star skeleton */}
      <div className="flex h-8 w-8 flex-shrink-0 items-center justify-center">
        <div className="h-[18px] w-[18px] animate-pulse rounded-full bg-gray-200" />
      </div>
      {/* Important marker skeleton */}
      <div className="h-8 w-5 flex-shrink-0" />
      {/* Sender skeleton */}
      <div className="w-[200px] flex-shrink-0 pr-4">
        <div className="h-3.5 w-28 animate-pulse rounded bg-gray-200" />
      </div>
      {/* Subject + snippet skeleton */}
      <div className="flex min-w-0 flex-1 items-center gap-2 pr-2">
        <div className="h-3.5 w-40 animate-pulse rounded bg-gray-200" />
        <div className="h-3.5 w-64 animate-pulse rounded bg-gray-100" />
      </div>
      {/* Date skeleton */}
      <div className="w-[80px] flex-shrink-0">
        <div className="ml-auto h-3 w-12 animate-pulse rounded bg-gray-200" />
      </div>
    </div>
  );
}

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
      <div className="divide-y divide-transparent">
        {Array.from({ length: 8 }).map((_, i) => (
          <SkeletonRow key={i} />
        ))}
      </div>
    );
  }

  if (emails.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-gmail-text-secondary">
        <Inbox className="mb-4 h-16 w-16 text-gray-300" strokeWidth={1} />
        <p
          className="text-xl font-medium text-gmail-text-primary"
          style={{ fontFamily: "'Google Sans', 'Roboto', sans-serif" }}
        >
          {currentLabel === "inbox"
            ? "Your inbox is empty"
            : `No messages in ${currentLabel}`}
        </p>
        <p className="mt-2 text-sm text-gmail-text-secondary">
          {currentLabel === "inbox"
            ? "Emails that arrive in your inbox will appear here."
            : "Messages with this label will appear here."}
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
