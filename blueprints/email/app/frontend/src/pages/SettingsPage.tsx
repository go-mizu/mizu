import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowLeft, Plus, Trash2, Check } from "lucide-react";
import { useSettingsStore, useLabelStore } from "../store";
import * as api from "../api";
import type { Label } from "../types";

type SettingsTab = "general" | "labels" | "theme";

const LABEL_COLORS = [
  "#EA4335",
  "#FBBC04",
  "#34A853",
  "#1A73E8",
  "#FF6D01",
  "#46BDC6",
  "#AF5CF7",
  "#F439A0",
  "#9334E6",
  "#E8710A",
  "#A0C3FF",
  "#71C287",
];

const THEME_OPTIONS = [
  { value: "light", label: "Light", bg: "bg-white", border: "border-gray-300" },
  {
    value: "dark",
    label: "Dark",
    bg: "bg-gray-900",
    border: "border-gray-700",
  },
  {
    value: "system",
    label: "System default",
    bg: "bg-gradient-to-r from-white to-gray-900",
    border: "border-gray-400",
  },
];

const DENSITY_OPTIONS = [
  { value: "default", label: "Default" },
  { value: "comfortable", label: "Comfortable" },
  { value: "compact", label: "Compact" },
];

const AUTO_ADVANCE_OPTIONS = [
  { value: "newer", label: "Go to newer conversation" },
  { value: "older", label: "Go to older conversation" },
  { value: "list", label: "Go back to threadlist" },
];

const UNDO_SEND_OPTIONS = [5, 10, 20, 30];

export default function SettingsPage() {
  const navigate = useNavigate();
  const settings = useSettingsStore((s) => s.settings);
  const updateSettings = useSettingsStore((s) => s.updateSettings);
  const labels = useLabelStore((s) => s.labels);
  const fetchLabels = useLabelStore((s) => s.fetchLabels);

  const [tab, setTab] = useState<SettingsTab>("general");
  const [displayName, setDisplayName] = useState("");
  const [emailAddress, setEmailAddress] = useState("");
  const [signature, setSignature] = useState("");
  const [conversationView, setConversationView] = useState(true);
  const [autoAdvance, setAutoAdvance] = useState("newer");
  const [undoSendSeconds, setUndoSendSeconds] = useState(5);
  const [density, setDensity] = useState("default");
  const [theme, setTheme] = useState("light");
  const [saved, setSaved] = useState(false);

  const [newLabelName, setNewLabelName] = useState("");
  const [newLabelColor, setNewLabelColor] = useState(LABEL_COLORS[0]!);
  const [addingLabel, setAddingLabel] = useState(false);
  const [editingLabelId, setEditingLabelId] = useState<string | null>(null);

  useEffect(() => {
    if (settings) {
      setDisplayName(settings.display_name);
      setEmailAddress(settings.email_address);
      setSignature(settings.signature);
      setConversationView(settings.conversation_view);
      setAutoAdvance(settings.auto_advance);
      setUndoSendSeconds(settings.undo_send_seconds);
      setDensity(settings.density);
      setTheme(settings.theme);
    }
  }, [settings]);

  const handleSave = useCallback(async () => {
    await updateSettings({
      display_name: displayName,
      email_address: emailAddress,
      signature,
      conversation_view: conversationView,
      auto_advance: autoAdvance,
      undo_send_seconds: undoSendSeconds,
      density,
      theme,
    });
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  }, [
    displayName,
    emailAddress,
    signature,
    conversationView,
    autoAdvance,
    undoSendSeconds,
    density,
    theme,
    updateSettings,
  ]);

  const handleCreateLabel = useCallback(async () => {
    if (!newLabelName.trim()) return;
    try {
      await api.createLabel({
        name: newLabelName.trim(),
        color: newLabelColor,
        type: "user",
        visible: true,
      });
      setNewLabelName("");
      setAddingLabel(false);
      fetchLabels();
    } catch {
      // Handle error silently
    }
  }, [newLabelName, newLabelColor, fetchLabels]);

  const handleDeleteLabel = useCallback(
    async (id: string) => {
      try {
        await api.deleteLabel(id);
        fetchLabels();
      } catch {
        // Handle error silently
      }
    },
    [fetchLabels]
  );

  const handleToggleLabelVisibility = useCallback(
    async (label: Label) => {
      try {
        await api.updateLabel(label.id, { visible: !label.visible });
        fetchLabels();
      } catch {
        // Handle error silently
      }
    },
    [fetchLabels]
  );

  const handleUpdateLabelColor = useCallback(
    async (label: Label, color: string) => {
      try {
        await api.updateLabel(label.id, { color });
        setEditingLabelId(null);
        fetchLabels();
      } catch {
        // Handle error silently
      }
    },
    [fetchLabels]
  );

  const tabs = [
    { id: "general" as const, label: "General" },
    { id: "labels" as const, label: "Labels" },
    { id: "theme" as const, label: "Theme" },
  ];

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <div className="flex h-14 items-center gap-3 border-b border-gmail-border px-4">
        <button
          onClick={() => navigate("/")}
          className="flex h-10 w-10 items-center justify-center rounded-full hover:bg-gray-100"
          aria-label="Back to inbox"
        >
          <ArrowLeft className="h-5 w-5 text-gmail-text-secondary" />
        </button>
        <h1
          className="text-xl text-gmail-text-primary"
          style={{ fontFamily: "'Google Sans', sans-serif" }}
        >
          Settings
        </h1>
      </div>

      {/* Tab bar */}
      <div className="flex border-b border-gmail-border">
        {tabs.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`px-6 py-3 text-sm font-medium transition-colors ${
              tab === t.id
                ? "border-b-2 border-gmail-blue text-gmail-blue"
                : "text-gmail-text-secondary hover:text-gmail-text-primary"
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto max-w-2xl">
          {/* General tab */}
          {tab === "general" && (
            <div className="space-y-6">
              {/* Display Name */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gmail-text-primary">
                  Display Name
                </label>
                <input
                  type="text"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gmail-blue focus:ring-1 focus:ring-gmail-blue"
                />
              </div>

              {/* Email Address */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gmail-text-primary">
                  Email Address
                </label>
                <input
                  type="email"
                  value={emailAddress}
                  onChange={(e) => setEmailAddress(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gmail-blue focus:ring-1 focus:ring-gmail-blue"
                />
              </div>

              {/* Signature */}
              <div>
                <label className="mb-1.5 block text-sm font-medium text-gmail-text-primary">
                  Signature
                </label>
                <textarea
                  value={signature}
                  onChange={(e) => setSignature(e.target.value)}
                  rows={5}
                  className="w-full rounded-lg border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gmail-blue focus:ring-1 focus:ring-gmail-blue"
                  placeholder="Create a signature that will be added to the end of your emails"
                />
              </div>

              {/* Conversation View toggle */}
              <div className="flex items-center justify-between rounded-lg border border-gray-200 p-4">
                <div>
                  <p className="text-sm font-medium text-gmail-text-primary">
                    Conversation view
                  </p>
                  <p className="mt-0.5 text-xs text-gmail-text-secondary">
                    Group emails in the same conversation together
                  </p>
                </div>
                <button
                  onClick={() => setConversationView(!conversationView)}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                    conversationView ? "bg-gmail-blue" : "bg-gray-300"
                  }`}
                >
                  <span
                    className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                      conversationView ? "translate-x-6" : "translate-x-1"
                    }`}
                  />
                </button>
              </div>

              {/* Display Density selector */}
              <div>
                <label className="mb-2 block text-sm font-medium text-gmail-text-primary">
                  Display density
                </label>
                <div className="flex gap-3">
                  {DENSITY_OPTIONS.map((option) => (
                    <button
                      key={option.value}
                      onClick={() => setDensity(option.value)}
                      className={`flex-1 rounded-lg border px-4 py-3 text-sm transition-colors ${
                        density === option.value
                          ? "border-gmail-blue bg-blue-50 text-gmail-blue"
                          : "border-gray-300 text-gmail-text-primary hover:bg-gray-50"
                      }`}
                    >
                      {option.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Auto-advance radio buttons */}
              <div>
                <label className="mb-2 block text-sm font-medium text-gmail-text-primary">
                  Auto-advance
                </label>
                <p className="mb-2 text-xs text-gmail-text-secondary">
                  After you delete, archive, or mute a conversation, choose
                  which conversation to show next
                </p>
                <div className="space-y-1.5">
                  {AUTO_ADVANCE_OPTIONS.map((option) => (
                    <label
                      key={option.value}
                      className="flex cursor-pointer items-center gap-3 rounded-lg px-3 py-2 hover:bg-gray-50"
                    >
                      <input
                        type="radio"
                        name="auto_advance"
                        value={option.value}
                        checked={autoAdvance === option.value}
                        onChange={(e) => setAutoAdvance(e.target.value)}
                        className="h-4 w-4 text-gmail-blue accent-gmail-blue"
                      />
                      <span className="text-sm text-gmail-text-primary">
                        {option.label}
                      </span>
                    </label>
                  ))}
                </div>
              </div>

              {/* Undo Send seconds selector */}
              <div>
                <label className="mb-2 block text-sm font-medium text-gmail-text-primary">
                  Undo Send
                </label>
                <p className="mb-2 text-xs text-gmail-text-secondary">
                  Send cancellation period
                </p>
                <div className="flex gap-2">
                  {UNDO_SEND_OPTIONS.map((seconds) => (
                    <button
                      key={seconds}
                      onClick={() => setUndoSendSeconds(seconds)}
                      className={`rounded-lg border px-4 py-2 text-sm transition-colors ${
                        undoSendSeconds === seconds
                          ? "border-gmail-blue bg-blue-50 text-gmail-blue"
                          : "border-gray-300 text-gmail-text-primary hover:bg-gray-50"
                      }`}
                    >
                      {seconds} seconds
                    </button>
                  ))}
                </div>
              </div>

              {/* Save button (blue pill) */}
              <div className="flex items-center gap-3 pt-2">
                <button
                  onClick={handleSave}
                  className="flex items-center gap-2 rounded-full bg-gmail-blue px-8 py-2.5 text-sm font-medium text-white hover:bg-gmail-blue-hover hover:shadow-sm"
                >
                  {saved && <Check className="h-4 w-4" />}
                  {saved ? "Saved" : "Save changes"}
                </button>
              </div>
            </div>
          )}

          {/* Labels tab */}
          {tab === "labels" && (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <h2 className="text-base font-medium text-gmail-text-primary">
                  Labels
                </h2>
                <button
                  onClick={() => setAddingLabel(true)}
                  className="flex items-center gap-1.5 rounded-full border border-gray-300 px-4 py-2 text-sm text-gmail-text-primary hover:bg-gray-50"
                >
                  <Plus className="h-4 w-4" />
                  Create new label
                </button>
              </div>

              {/* Inline creation form */}
              {addingLabel && (
                <div className="flex items-end gap-3 rounded-lg border border-gmail-blue bg-blue-50 p-4">
                  <div className="flex-1">
                    <label className="mb-1 block text-xs font-medium text-gmail-text-primary">
                      Label name
                    </label>
                    <input
                      type="text"
                      value={newLabelName}
                      onChange={(e) => setNewLabelName(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") handleCreateLabel();
                        if (e.key === "Escape") setAddingLabel(false);
                      }}
                      className="w-full rounded border border-gray-300 px-3 py-1.5 text-sm outline-none focus:border-gmail-blue"
                      autoFocus
                    />
                  </div>
                  <div>
                    <label className="mb-1 block text-xs font-medium text-gmail-text-primary">
                      Color
                    </label>
                    <div className="flex gap-1">
                      {LABEL_COLORS.map((color) => (
                        <button
                          key={color}
                          onClick={() => setNewLabelColor(color)}
                          className={`h-6 w-6 rounded-full border-2 ${
                            newLabelColor === color
                              ? "border-gmail-text-primary"
                              : "border-transparent"
                          }`}
                          style={{ backgroundColor: color }}
                        />
                      ))}
                    </div>
                  </div>
                  <button
                    onClick={handleCreateLabel}
                    className="rounded-full bg-gmail-blue px-5 py-1.5 text-sm text-white hover:bg-gmail-blue-hover"
                  >
                    Create
                  </button>
                  <button
                    onClick={() => setAddingLabel(false)}
                    className="rounded-full border border-gray-300 px-5 py-1.5 text-sm hover:bg-gray-100"
                  >
                    Cancel
                  </button>
                </div>
              )}

              {/* System labels list */}
              <div>
                <h3 className="mb-2 text-sm font-medium text-gmail-text-secondary">
                  System labels
                </h3>
                <div className="divide-y divide-gray-100 rounded-lg border border-gray-200">
                  {labels
                    .filter((l) => l.type === "system")
                    .map((label) => (
                      <div
                        key={label.id}
                        className="flex items-center justify-between px-4 py-3"
                      >
                        <span className="text-sm text-gmail-text-primary">
                          {label.name}
                        </span>
                        <div className="flex items-center gap-3">
                          <span className="text-xs text-gmail-text-secondary">
                            {label.total_count} messages
                          </span>
                          <button
                            onClick={() => handleToggleLabelVisibility(label)}
                            className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                              label.visible ? "bg-gmail-blue" : "bg-gray-300"
                            }`}
                          >
                            <span
                              className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                                label.visible
                                  ? "translate-x-4.5"
                                  : "translate-x-0.5"
                              }`}
                            />
                          </button>
                        </div>
                      </div>
                    ))}
                </div>
              </div>

              {/* Custom labels list */}
              <div>
                <h3 className="mb-2 text-sm font-medium text-gmail-text-secondary">
                  Custom labels
                </h3>
                {labels.filter((l) => l.type === "user").length === 0 ? (
                  <p className="rounded-lg border border-gray-200 px-4 py-6 text-center text-sm text-gmail-text-secondary">
                    No custom labels yet. Create one above.
                  </p>
                ) : (
                  <div className="divide-y divide-gray-100 rounded-lg border border-gray-200">
                    {labels
                      .filter((l) => l.type === "user")
                      .map((label) => (
                        <div
                          key={label.id}
                          className="relative flex items-center justify-between px-4 py-3"
                        >
                          <div className="flex items-center gap-3">
                            <button
                              onClick={() =>
                                setEditingLabelId(
                                  editingLabelId === label.id
                                    ? null
                                    : label.id
                                )
                              }
                              className="h-4 w-4 rounded-full border border-gray-300"
                              style={{
                                backgroundColor: label.color || "#5F6368",
                              }}
                            />
                            <span className="text-sm text-gmail-text-primary">
                              {label.name}
                            </span>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-xs text-gmail-text-secondary">
                              {label.total_count} messages
                            </span>
                            <button
                              onClick={() =>
                                handleToggleLabelVisibility(label)
                              }
                              className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                                label.visible
                                  ? "bg-gmail-blue"
                                  : "bg-gray-300"
                              }`}
                            >
                              <span
                                className={`inline-block h-3.5 w-3.5 transform rounded-full bg-white transition-transform ${
                                  label.visible
                                    ? "translate-x-4.5"
                                    : "translate-x-0.5"
                                }`}
                              />
                            </button>
                            <button
                              onClick={() => handleDeleteLabel(label.id)}
                              className="flex h-7 w-7 items-center justify-center rounded-full text-gmail-text-secondary hover:bg-gray-100 hover:text-red-500"
                              title="Delete label"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          </div>
                          {/* Color picker dropdown */}
                          {editingLabelId === label.id && (
                            <div className="absolute right-4 top-full z-10 mt-1 rounded-lg border border-gray-200 bg-white p-3 shadow-lg">
                              <p className="mb-2 text-xs font-medium text-gmail-text-secondary">
                                Choose color
                              </p>
                              <div className="flex gap-1.5">
                                {LABEL_COLORS.map((color) => (
                                  <button
                                    key={color}
                                    onClick={() =>
                                      handleUpdateLabelColor(label, color)
                                    }
                                    className={`h-6 w-6 rounded-full border-2 ${
                                      label.color === color
                                        ? "border-gmail-text-primary"
                                        : "border-transparent hover:border-gray-400"
                                    }`}
                                    style={{ backgroundColor: color }}
                                  />
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Theme tab */}
          {tab === "theme" && (
            <div className="space-y-6">
              <div>
                <h2 className="mb-4 text-base font-medium text-gmail-text-primary">
                  Theme
                </h2>
                <div className="flex gap-4">
                  {THEME_OPTIONS.map((option) => (
                    <button
                      key={option.value}
                      onClick={() => setTheme(option.value)}
                      className={`flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all ${
                        theme === option.value
                          ? "border-gmail-blue shadow-sm"
                          : "border-gray-200 hover:border-gray-400"
                      }`}
                    >
                      <div
                        className={`h-20 w-28 rounded-lg ${option.bg} ${option.border} border`}
                      />
                      <span
                        className={`text-sm font-medium ${
                          theme === option.value
                            ? "text-gmail-blue"
                            : "text-gmail-text-primary"
                        }`}
                      >
                        {option.label}
                      </span>
                    </button>
                  ))}
                </div>
              </div>

              {/* Save button */}
              <div className="flex items-center gap-3 pt-4">
                <button
                  onClick={handleSave}
                  className="flex items-center gap-2 rounded-full bg-gmail-blue px-8 py-2.5 text-sm font-medium text-white hover:bg-gmail-blue-hover hover:shadow-sm"
                >
                  {saved && <Check className="h-4 w-4" />}
                  {saved ? "Saved" : "Save changes"}
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
