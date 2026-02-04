import { useState, useRef, useEffect } from "react";
import {
  Archive,
  Trash2,
  Mail,
  MailOpen,
  Tag,
  MoreVertical,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Square,
  CheckSquare,
  MinusSquare,
} from "lucide-react";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";

export default function Toolbar() {
  const emails = useEmailStore((s) => s.emails);
  const selectedEmails = useEmailStore((s) => s.selectedEmails);
  const selectAll = useEmailStore((s) => s.selectAll);
  const deselectAll = useEmailStore((s) => s.deselectAll);
  const refreshEmails = useEmailStore((s) => s.refreshEmails);
  const page = useEmailStore((s) => s.page);
  const perPage = useEmailStore((s) => s.perPage);
  const total = useEmailStore((s) => s.total);
  const totalPages = useEmailStore((s) => s.totalPages);
  const nextPage = useEmailStore((s) => s.nextPage);
  const prevPage = useEmailStore((s) => s.prevPage);
  const labels = useLabelStore((s) => s.labels);

  const [labelDropdownOpen, setLabelDropdownOpen] = useState(false);
  const [moreDropdownOpen, setMoreDropdownOpen] = useState(false);
  const labelDropdownRef = useRef<HTMLDivElement>(null);
  const moreDropdownRef = useRef<HTMLDivElement>(null);

  const hasSelected = selectedEmails.size > 0;
  const allSelected =
    emails.length > 0 && selectedEmails.size === emails.length;
  const someSelected = selectedEmails.size > 0 && !allSelected;

  const start = (page - 1) * perPage + 1;
  const end = Math.min(page * perPage, total);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        labelDropdownRef.current &&
        !labelDropdownRef.current.contains(e.target as Node)
      ) {
        setLabelDropdownOpen(false);
      }
      if (
        moreDropdownRef.current &&
        !moreDropdownRef.current.contains(e.target as Node)
      ) {
        setMoreDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  async function handleBatchAction(action: string) {
    if (selectedEmails.size === 0) return;
    try {
      await api.batchEmails({
        ids: Array.from(selectedEmails),
        action,
      });
      deselectAll();
      refreshEmails();
    } catch {
      // Handle error silently
    }
  }

  async function handleMoveToLabel(labelId: string) {
    if (selectedEmails.size === 0) return;
    try {
      await api.batchEmails({
        ids: Array.from(selectedEmails),
        action: "label",
        label_id: labelId,
      });
      deselectAll();
      setLabelDropdownOpen(false);
      refreshEmails();
    } catch {
      // Handle error silently
    }
  }

  function handleCheckboxClick() {
    if (hasSelected) {
      deselectAll();
    } else {
      selectAll();
    }
  }

  const CheckIcon = allSelected
    ? CheckSquare
    : someSelected
      ? MinusSquare
      : Square;

  return (
    <div className="toolbar-shadow flex h-10 items-center justify-between border-b border-gmail-border px-2">
      <div className="flex items-center gap-0.5">
        {/* Checkbox */}
        <button
          onClick={handleCheckboxClick}
          className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
          aria-label={hasSelected ? "Deselect all" : "Select all"}
        >
          <CheckIcon
            className={`h-5 w-5 ${hasSelected ? "text-gmail-text-primary" : "text-gmail-text-secondary"}`}
          />
        </button>

        {hasSelected ? (
          <>
            {/* Archive */}
            <button
              onClick={() => handleBatchAction("archive")}
              className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
              title="Archive"
            >
              <Archive className="h-4 w-4 text-gmail-text-secondary" />
            </button>

            {/* Delete */}
            <button
              onClick={() => handleBatchAction("trash")}
              className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
              title="Delete"
            >
              <Trash2 className="h-4 w-4 text-gmail-text-secondary" />
            </button>

            {/* Mark read/unread */}
            <button
              onClick={() => handleBatchAction("read")}
              className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
              title="Mark as read"
            >
              <MailOpen className="h-4 w-4 text-gmail-text-secondary" />
            </button>
            <button
              onClick={() => handleBatchAction("unread")}
              className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
              title="Mark as unread"
            >
              <Mail className="h-4 w-4 text-gmail-text-secondary" />
            </button>

            {/* Move to label */}
            <div className="relative" ref={labelDropdownRef}>
              <button
                onClick={() => setLabelDropdownOpen(!labelDropdownOpen)}
                className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
                title="Move to"
              >
                <Tag className="h-4 w-4 text-gmail-text-secondary" />
              </button>
              {labelDropdownOpen && (
                <div className="absolute left-0 top-full z-50 mt-1 min-w-[180px] rounded-lg border border-gmail-border bg-white py-1 shadow-lg">
                  <div className="px-3 py-1.5 text-xs font-medium text-gmail-text-secondary">
                    Move to
                  </div>
                  {labels
                    .filter((l) => l.visible)
                    .map((label) => (
                      <button
                        key={label.id}
                        onClick={() => handleMoveToLabel(label.id)}
                        className="flex w-full items-center px-3 py-1.5 text-sm hover:bg-gray-100"
                      >
                        {label.color && (
                          <span
                            className="mr-2 inline-block h-2.5 w-2.5 rounded-full"
                            style={{ backgroundColor: label.color }}
                          />
                        )}
                        {label.name}
                      </button>
                    ))}
                </div>
              )}
            </div>

            {/* More actions */}
            <div className="relative" ref={moreDropdownRef}>
              <button
                onClick={() => setMoreDropdownOpen(!moreDropdownOpen)}
                className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
                title="More"
              >
                <MoreVertical className="h-4 w-4 text-gmail-text-secondary" />
              </button>
              {moreDropdownOpen && (
                <div className="absolute left-0 top-full z-50 mt-1 min-w-[180px] rounded-lg border border-gmail-border bg-white py-1 shadow-lg">
                  <button
                    onClick={() => {
                      handleBatchAction("star");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-1.5 text-sm hover:bg-gray-100"
                  >
                    Star
                  </button>
                  <button
                    onClick={() => {
                      handleBatchAction("unstar");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-1.5 text-sm hover:bg-gray-100"
                  >
                    Remove star
                  </button>
                  <button
                    onClick={() => {
                      handleBatchAction("important");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-1.5 text-sm hover:bg-gray-100"
                  >
                    Mark as important
                  </button>
                  <button
                    onClick={() => {
                      handleBatchAction("unimportant");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-1.5 text-sm hover:bg-gray-100"
                  >
                    Mark as not important
                  </button>
                </div>
              )}
            </div>
          </>
        ) : (
          <button
            onClick={() => refreshEmails()}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100"
            title="Refresh"
          >
            <RefreshCw className="h-4 w-4 text-gmail-text-secondary" />
          </button>
        )}
      </div>

      {/* Pagination */}
      {total > 0 && (
        <div className="flex items-center gap-1">
          <span className="mr-2 text-xs text-gmail-text-secondary">
            {start}-{end} of {total}
          </span>
          <button
            onClick={prevPage}
            disabled={page <= 1}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100 disabled:opacity-30 disabled:hover:bg-transparent"
            aria-label="Previous page"
          >
            <ChevronLeft className="h-5 w-5 text-gmail-text-secondary" />
          </button>
          <button
            onClick={nextPage}
            disabled={page >= totalPages}
            className="flex h-8 w-8 items-center justify-center rounded hover:bg-gray-100 disabled:opacity-30 disabled:hover:bg-transparent"
            aria-label="Next page"
          >
            <ChevronRight className="h-5 w-5 text-gmail-text-secondary" />
          </button>
        </div>
      )}
    </div>
  );
}
