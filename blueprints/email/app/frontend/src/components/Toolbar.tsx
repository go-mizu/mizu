import { useState, useRef, useEffect, useCallback } from "react";
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
  ChevronDown,
} from "lucide-react";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";
import { showToast } from "./Toast";

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

  const [selectDropdownOpen, setSelectDropdownOpen] = useState(false);
  const [labelDropdownOpen, setLabelDropdownOpen] = useState(false);
  const [moreDropdownOpen, setMoreDropdownOpen] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const selectDropdownRef = useRef<HTMLDivElement>(null);
  const labelDropdownRef = useRef<HTMLDivElement>(null);
  const moreDropdownRef = useRef<HTMLDivElement>(null);

  const hasSelected = selectedEmails.size > 0;
  const allSelected =
    emails.length > 0 && selectedEmails.size === emails.length;
  const someSelected = selectedEmails.size > 0 && !allSelected;

  const start = total > 0 ? (page - 1) * perPage + 1 : 0;
  const end = Math.min(page * perPage, total);

  // Close all dropdowns on click outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        selectDropdownRef.current &&
        !selectDropdownRef.current.contains(e.target as Node)
      ) {
        setSelectDropdownOpen(false);
      }
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

  const handleBatchAction = useCallback(
    async (action: string) => {
      if (selectedEmails.size === 0) return;
      try {
        await api.batchEmails({
          ids: Array.from(selectedEmails),
          action,
        });
        const count = selectedEmails.size;
        deselectAll();
        refreshEmails();
        switch (action) {
          case "archive":
            showToast(`${count} conversation${count > 1 ? "s" : ""} archived`);
            break;
          case "trash":
            showToast(
              `${count} conversation${count > 1 ? "s" : ""} moved to Trash`
            );
            break;
          case "read":
            showToast(
              `${count} conversation${count > 1 ? "s" : ""} marked as read`
            );
            break;
          case "unread":
            showToast(
              `${count} conversation${count > 1 ? "s" : ""} marked as unread`
            );
            break;
          case "star":
            showToast("Starred");
            break;
          case "unstar":
            showToast("Star removed");
            break;
          case "important":
            showToast("Marked as important");
            break;
          case "unimportant":
            showToast("Marked as not important");
            break;
        }
      } catch {
        showToast("Action failed. Please try again.");
      }
    },
    [selectedEmails, deselectAll, refreshEmails]
  );

  const handleMoveToLabel = useCallback(
    async (labelId: string, labelName: string) => {
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
        showToast(`Moved to ${labelName}`);
      } catch {
        showToast("Failed to move. Please try again.");
      }
    },
    [selectedEmails, deselectAll, refreshEmails]
  );

  function handleCheckboxClick() {
    if (hasSelected) {
      deselectAll();
    } else {
      selectAll();
    }
  }

  async function handleRefresh() {
    setRefreshing(true);
    await refreshEmails();
    // Keep spinning for at least 600ms for visual feedback
    setTimeout(() => setRefreshing(false), 600);
  }

  function handleSelectOption(option: "all" | "none" | "read" | "unread" | "starred" | "unstarred") {
    switch (option) {
      case "all":
        selectAll();
        break;
      case "none":
        deselectAll();
        break;
      case "read": {
        deselectAll();
        const readIds = new Set(
          emails.filter((e) => e.is_read).map((e) => e.id)
        );
        readIds.forEach((id) => useEmailStore.getState().toggleSelect(id));
        break;
      }
      case "unread": {
        deselectAll();
        const unreadIds = new Set(
          emails.filter((e) => !e.is_read).map((e) => e.id)
        );
        unreadIds.forEach((id) => useEmailStore.getState().toggleSelect(id));
        break;
      }
      case "starred": {
        deselectAll();
        const starredIds = new Set(
          emails.filter((e) => e.is_starred).map((e) => e.id)
        );
        starredIds.forEach((id) => useEmailStore.getState().toggleSelect(id));
        break;
      }
      case "unstarred": {
        deselectAll();
        const unstarredIds = new Set(
          emails.filter((e) => !e.is_starred).map((e) => e.id)
        );
        unstarredIds.forEach((id) =>
          useEmailStore.getState().toggleSelect(id)
        );
        break;
      }
    }
    setSelectDropdownOpen(false);
  }

  const CheckIcon = allSelected
    ? CheckSquare
    : someSelected
      ? MinusSquare
      : Square;

  const visibleLabels = labels.filter((l) => l.visible);

  return (
    <div className="toolbar-shadow flex h-10 items-center justify-between border-b border-gmail-border px-2">
      {/* Left side */}
      <div className="flex items-center gap-0.5">
        {/* Checkbox + dropdown arrow */}
        <div className="relative flex items-center" ref={selectDropdownRef}>
          <button
            onClick={handleCheckboxClick}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
            aria-label={hasSelected ? "Deselect all" : "Select all"}
          >
            <CheckIcon
              className={`h-[18px] w-[18px] ${
                hasSelected
                  ? "text-gmail-text-primary"
                  : "text-gmail-text-secondary"
              }`}
            />
          </button>
          <button
            onClick={() => setSelectDropdownOpen(!selectDropdownOpen)}
            className="flex h-8 w-4 items-center justify-center rounded hover:bg-gray-100"
            aria-label="Select options"
          >
            <ChevronDown className="h-3.5 w-3.5 text-gmail-text-secondary" />
          </button>
          {selectDropdownOpen && (
            <div className="absolute left-0 top-full z-50 mt-1 min-w-[120px] rounded-lg border border-gmail-border bg-white py-1 shadow-lg">
              {(
                [
                  { key: "all", label: "All" },
                  { key: "none", label: "None" },
                  { key: "read", label: "Read" },
                  { key: "unread", label: "Unread" },
                  { key: "starred", label: "Starred" },
                  { key: "unstarred", label: "Unstarred" },
                ] as const
              ).map((opt) => (
                <button
                  key={opt.key}
                  onClick={() => handleSelectOption(opt.key)}
                  className="flex w-full items-center px-3 py-1.5 text-sm text-gmail-text-primary hover:bg-gray-100"
                >
                  {opt.label}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Divider */}
        <div className="mx-0.5 h-5 w-px bg-gmail-border" />

        {hasSelected ? (
          <>
            {/* Archive */}
            <button
              onClick={() => handleBatchAction("archive")}
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Archive"
            >
              <Archive className="h-[18px] w-[18px] text-gmail-text-secondary" />
            </button>

            {/* Delete */}
            <button
              onClick={() => handleBatchAction("trash")}
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Delete"
            >
              <Trash2 className="h-[18px] w-[18px] text-gmail-text-secondary" />
            </button>

            {/* Mark as read */}
            <button
              onClick={() => handleBatchAction("read")}
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Mark as read"
            >
              <MailOpen className="h-[18px] w-[18px] text-gmail-text-secondary" />
            </button>

            {/* Mark as unread */}
            <button
              onClick={() => handleBatchAction("unread")}
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Mark as unread"
            >
              <Mail className="h-[18px] w-[18px] text-gmail-text-secondary" />
            </button>

            {/* Label dropdown */}
            <div className="relative" ref={labelDropdownRef}>
              <button
                onClick={() => setLabelDropdownOpen(!labelDropdownOpen)}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
                title="Label"
              >
                <Tag className="h-[18px] w-[18px] text-gmail-text-secondary" />
              </button>
              {labelDropdownOpen && (
                <div className="absolute left-0 top-full z-50 mt-1 min-w-[200px] rounded-lg border border-gmail-border bg-white py-1 shadow-lg">
                  <div className="px-3 py-2 text-xs font-medium text-gmail-text-secondary">
                    Label as:
                  </div>
                  <div className="max-h-[240px] overflow-y-auto">
                    {visibleLabels.map((label) => (
                      <button
                        key={label.id}
                        onClick={() =>
                          handleMoveToLabel(label.id, label.name)
                        }
                        className="flex w-full items-center gap-2 px-3 py-1.5 text-sm text-gmail-text-primary hover:bg-gray-100"
                      >
                        {label.color && (
                          <span
                            className="inline-block h-2.5 w-2.5 flex-shrink-0 rounded-full"
                            style={{ backgroundColor: label.color }}
                          />
                        )}
                        <span className="truncate">{label.name}</span>
                      </button>
                    ))}
                    {visibleLabels.length === 0 && (
                      <div className="px-3 py-2 text-xs text-gmail-text-secondary">
                        No labels
                      </div>
                    )}
                  </div>
                </div>
              )}
            </div>

            {/* More actions dropdown */}
            <div className="relative" ref={moreDropdownRef}>
              <button
                onClick={() => setMoreDropdownOpen(!moreDropdownOpen)}
                className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
                title="More"
              >
                <MoreVertical className="h-[18px] w-[18px] text-gmail-text-secondary" />
              </button>
              {moreDropdownOpen && (
                <div className="absolute left-0 top-full z-50 mt-1 min-w-[200px] rounded-lg border border-gmail-border bg-white py-1 shadow-lg">
                  <button
                    onClick={() => {
                      handleBatchAction("star");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-2 text-sm text-gmail-text-primary hover:bg-gray-100"
                  >
                    Star
                  </button>
                  <button
                    onClick={() => {
                      handleBatchAction("unstar");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-2 text-sm text-gmail-text-primary hover:bg-gray-100"
                  >
                    Remove star
                  </button>
                  <div className="my-1 border-t border-gmail-border" />
                  <button
                    onClick={() => {
                      handleBatchAction("important");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-2 text-sm text-gmail-text-primary hover:bg-gray-100"
                  >
                    Mark as important
                  </button>
                  <button
                    onClick={() => {
                      handleBatchAction("unimportant");
                      setMoreDropdownOpen(false);
                    }}
                    className="flex w-full items-center px-3 py-2 text-sm text-gmail-text-primary hover:bg-gray-100"
                  >
                    Mark as not important
                  </button>
                </div>
              )}
            </div>
          </>
        ) : (
          /* Refresh button (only when nothing selected) */
          <button
            onClick={handleRefresh}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
            title="Refresh"
          >
            <RefreshCw
              className={`h-[18px] w-[18px] text-gmail-text-secondary ${
                refreshing ? "animate-spin" : ""
              }`}
            />
          </button>
        )}
      </div>

      {/* Right side: Pagination */}
      {total > 0 && (
        <div className="flex items-center gap-1">
          <span className="mr-2 text-xs text-gmail-text-secondary">
            {start}&ndash;{end} of {total}
          </span>
          <button
            onClick={prevPage}
            disabled={page <= 1}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100 disabled:opacity-30 disabled:hover:bg-transparent"
            aria-label="Newer"
          >
            <ChevronLeft className="h-[18px] w-[18px] text-gmail-text-secondary" />
          </button>
          <button
            onClick={nextPage}
            disabled={page >= totalPages}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100 disabled:opacity-30 disabled:hover:bg-transparent"
            aria-label="Older"
          >
            <ChevronRight className="h-[18px] w-[18px] text-gmail-text-secondary" />
          </button>
        </div>
      )}
    </div>
  );
}
