import { useState } from "react";
import { useNavigate, useLocation } from "react-router-dom";
import {
  Inbox,
  Star,
  Clock,
  Tag,
  Send,
  FileText,
  Mail,
  AlertCircle,
  Trash2,
  Plus,
  ChevronDown,
  ChevronUp,
  Pencil,
} from "lucide-react";
import { useEmailStore, useLabelStore } from "../store";
import * as api from "../api";

interface SidebarProps {
  collapsed: boolean;
}

const SYSTEM_LABELS = [
  { id: "inbox", name: "Inbox", icon: Inbox },
  { id: "starred", name: "Starred", icon: Star },
  { id: "snoozed", name: "Snoozed", icon: Clock },
  { id: "important", name: "Important", icon: Tag },
  { id: "sent", name: "Sent", icon: Send },
  { id: "drafts", name: "Drafts", icon: FileText },
  { id: "all", name: "All Mail", icon: Mail },
  { id: "spam", name: "Spam", icon: AlertCircle },
  { id: "trash", name: "Trash", icon: Trash2 },
];

const DEFAULT_VISIBLE_COUNT = 6;

export default function Sidebar({ collapsed }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const [showMore, setShowMore] = useState(false);
  const [creatingLabel, setCreatingLabel] = useState(false);
  const [newLabelName, setNewLabelName] = useState("");

  const currentLabel = useEmailStore((s) => s.currentLabel);
  const setLabel = useEmailStore((s) => s.setLabel);
  const openCompose = useEmailStore((s) => s.openCompose);
  const labels = useLabelStore((s) => s.labels);
  const fetchLabels = useLabelStore((s) => s.fetchLabels);

  const systemLabels = labels.filter((l) => l.type === "system");
  const userLabels = labels.filter((l) => l.type === "user");

  const displayLabels = showMore
    ? SYSTEM_LABELS
    : SYSTEM_LABELS.slice(0, DEFAULT_VISIBLE_COUNT);

  function getUnread(labelId: string): number {
    const found = systemLabels.find((l) => l.id === labelId);
    return found?.unread_count ?? 0;
  }

  function isActive(labelId: string): boolean {
    const pathLabel = location.pathname.startsWith("/label/")
      ? location.pathname.split("/label/")[1]
      : null;
    if (pathLabel) return pathLabel === labelId;
    if (location.pathname === "/" && labelId === "inbox") return true;
    return currentLabel === labelId && location.pathname === "/";
  }

  function handleLabelClick(labelId: string) {
    setLabel(labelId);
    if (labelId === "inbox") {
      navigate("/");
    } else {
      navigate(`/label/${labelId}`);
    }
  }

  async function handleCreateLabel() {
    if (!newLabelName.trim()) return;
    try {
      await api.createLabel({
        name: newLabelName.trim(),
        type: "user",
        visible: true,
      });
      setNewLabelName("");
      setCreatingLabel(false);
      fetchLabels();
    } catch {
      // Handle error silently
    }
  }

  return (
    <div className="flex h-full flex-col py-2">
      {/* ---- Compose FAB Button ---- */}
      <div className={`px-3 pb-4 ${collapsed ? "flex justify-center" : ""}`}>
        {collapsed ? (
          <button
            onClick={() => openCompose()}
            className="flex h-14 w-14 items-center justify-center rounded-2xl bg-gmail-compose-fab shadow-md transition-all hover:shadow-lg hover:brightness-95 focus-ring"
            aria-label="Compose"
            title="Compose"
          >
            <Pencil className="h-6 w-6 text-gmail-sidebar-active-text" />
          </button>
        ) : (
          <button
            onClick={() => openCompose()}
            className="flex h-14 items-center gap-3 rounded-2xl bg-gmail-compose-fab px-6 shadow-md transition-all hover:shadow-lg hover:brightness-95 focus-ring"
          >
            <Pencil className="h-6 w-6 text-gmail-sidebar-active-text" />
            <span className="font-google-sans text-sm font-medium tracking-wide text-gmail-sidebar-active-text">
              Compose
            </span>
          </button>
        )}
      </div>

      {/* ---- System Labels ---- */}
      <nav className="flex-1 space-y-px overflow-y-auto px-2">
        {displayLabels.map((item) => {
          const active = isActive(item.id);
          const Icon = item.icon;
          const unread = getUnread(item.id);

          return (
            <button
              key={item.id}
              onClick={() => handleLabelClick(item.id)}
              className={`sidebar-label-item flex w-full items-center rounded-r-full text-sm focus-ring ${
                collapsed ? "justify-center rounded-l-full px-0" : "pl-6 pr-3"
              } ${
                active
                  ? "bg-gmail-sidebar-active font-bold text-gmail-sidebar-active-text"
                  : "text-gmail-text-secondary hover:bg-gmail-sidebar-hover"
              }`}
              style={{ height: "32px" }}
              title={collapsed ? item.name : undefined}
            >
              <Icon
                className={`h-5 w-5 flex-shrink-0 ${
                  active ? "text-gmail-sidebar-active-text" : ""
                }`}
                strokeWidth={active ? 2.2 : 1.8}
              />
              {!collapsed && (
                <>
                  <span className="ml-4 flex-1 truncate text-left text-[13px]">
                    {item.name}
                  </span>
                  {unread > 0 && (
                    <span className="ml-auto text-xs font-bold tabular-nums">
                      {unread}
                    </span>
                  )}
                </>
              )}
            </button>
          );
        })}

        {/* More / Less Toggle */}
        {!collapsed && (
          <button
            onClick={() => setShowMore(!showMore)}
            className="sidebar-label-item flex w-full items-center rounded-r-full py-1 pl-6 pr-3 text-sm text-gmail-text-secondary hover:bg-gmail-sidebar-hover focus-ring"
            style={{ height: "32px" }}
          >
            {showMore ? (
              <ChevronUp className="h-5 w-5 flex-shrink-0" />
            ) : (
              <ChevronDown className="h-5 w-5 flex-shrink-0" />
            )}
            <span className="ml-4 text-[13px]">
              {showMore ? "Less" : "More"}
            </span>
          </button>
        )}

        {/* Collapsed: just show a More dot indicator */}
        {collapsed && !showMore && SYSTEM_LABELS.length > DEFAULT_VISIBLE_COUNT && (
          <button
            onClick={() => setShowMore(!showMore)}
            className="sidebar-label-item flex w-full items-center justify-center rounded-full py-1 text-gmail-text-secondary hover:bg-gmail-sidebar-hover focus-ring"
            style={{ height: "32px" }}
            title={showMore ? "Less" : "More"}
          >
            <ChevronDown className="h-5 w-5" />
          </button>
        )}

        {/* ---- Divider ---- */}
        {!collapsed && (
          <div className="mx-4 my-3 border-t border-gmail-divider" />
        )}

        {/* ---- Custom Labels Section ---- */}
        {!collapsed && (
          <>
            {/* Labels Header */}
            <div className="flex items-center justify-between px-6 py-2">
              <span className="text-[13px] font-medium text-gmail-text-secondary">
                Labels
              </span>
              <button
                onClick={() => setCreatingLabel(true)}
                className="flex h-7 w-7 items-center justify-center rounded-full hover:bg-gmail-sidebar-hover focus-ring"
                aria-label="Create new label"
                title="Create new label"
              >
                <Plus className="h-4 w-4 text-gmail-text-secondary" />
              </button>
            </div>

            {/* Inline Label Creation */}
            {creatingLabel && (
              <div className="flex items-center gap-2 px-4 py-1.5">
                <input
                  type="text"
                  value={newLabelName}
                  onChange={(e) => setNewLabelName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreateLabel();
                    if (e.key === "Escape") {
                      setCreatingLabel(false);
                      setNewLabelName("");
                    }
                  }}
                  placeholder="Label name"
                  className="flex-1 rounded-md border border-gmail-border bg-white px-2.5 py-1.5 text-sm outline-none transition-colors focus:border-gmail-blue focus:ring-1 focus:ring-gmail-blue"
                  autoFocus
                />
                <button
                  onClick={handleCreateLabel}
                  className="rounded-md px-3 py-1.5 text-xs font-medium text-gmail-blue hover:bg-gmail-blue-surface focus-ring"
                >
                  Save
                </button>
              </div>
            )}

            {/* User-Created Labels */}
            {userLabels.map((label) => {
              const active = isActive(label.id);

              return (
                <button
                  key={label.id}
                  onClick={() => handleLabelClick(label.id)}
                  className={`sidebar-label-item flex w-full items-center rounded-r-full py-1 pl-6 pr-3 text-sm focus-ring ${
                    active
                      ? "bg-gmail-sidebar-active font-bold text-gmail-sidebar-active-text"
                      : "text-gmail-text-secondary hover:bg-gmail-sidebar-hover"
                  }`}
                  style={{ height: "32px" }}
                >
                  <span
                    className="mr-4 inline-block h-3 w-3 flex-shrink-0 rounded-full ring-1 ring-black/5"
                    style={{
                      backgroundColor: label.color || "#5F6368",
                    }}
                  />
                  <span className="flex-1 truncate text-left text-[13px]">
                    {label.name}
                  </span>
                  {label.unread_count > 0 && (
                    <span className="ml-auto text-xs font-bold tabular-nums">
                      {label.unread_count}
                    </span>
                  )}
                </button>
              );
            })}
          </>
        )}
      </nav>
    </div>
  );
}
