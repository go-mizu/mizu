import { useState, useRef, useCallback, useEffect } from "react";
import {
  X,
  Minus,
  Maximize2,
  Minimize2,
  Trash2,
  Paperclip,
  MoreVertical,
  Send,
  Type,
} from "lucide-react";
import { useEmailStore } from "../store";
import * as api from "../api";
import type { Recipient } from "../types";

type ComposeState = "normal" | "minimized" | "maximized";

export default function ComposeModal() {
  const closeCompose = useEmailStore((s) => s.closeCompose);
  const composeData = useEmailStore((s) => s.composeData);
  const composeMode = useEmailStore((s) => s.composeMode);
  const refreshEmails = useEmailStore((s) => s.refreshEmails);

  const [state, setState] = useState<ComposeState>("normal");
  const [to, setTo] = useState<Recipient[]>(composeData?.to ?? []);
  const [cc, setCc] = useState<Recipient[]>(composeData?.cc ?? []);
  const [bcc, setBcc] = useState<Recipient[]>(composeData?.bcc ?? []);
  const [showCc, setShowCc] = useState((composeData?.cc?.length ?? 0) > 0);
  const [showBcc, setShowBcc] = useState((composeData?.bcc?.length ?? 0) > 0);
  const [subject, setSubject] = useState(composeData?.subject ?? "");
  const [body, setBody] = useState(composeData?.body_html ?? composeData?.body_text ?? "");
  const [toInput, setToInput] = useState("");
  const [ccInput, setCcInput] = useState("");
  const [bccInput, setBccInput] = useState("");
  const [sending, setSending] = useState(false);

  const bodyRef = useRef<HTMLTextAreaElement>(null);
  const toInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    // Focus the To field on open, unless it's a reply/forward with pre-filled To
    if (to.length === 0 && toInputRef.current) {
      toInputRef.current.focus();
    } else if (bodyRef.current) {
      bodyRef.current.focus();
    }
  }, [to.length]);

  const addRecipient = useCallback(
    (
      input: string,
      setter: React.Dispatch<React.SetStateAction<Recipient[]>>,
      inputSetter: React.Dispatch<React.SetStateAction<string>>
    ) => {
      const trimmed = input.trim();
      if (!trimmed) return;

      // Parse "Name <email>" format or plain email
      const match = trimmed.match(/^(.+?)\s*<(.+?)>$/);
      if (match) {
        setter((prev) => [
          ...prev,
          { name: match[1]!.trim(), address: match[2]!.trim() },
        ]);
      } else {
        setter((prev) => [...prev, { address: trimmed }]);
      }
      inputSetter("");
    },
    []
  );

  const removeRecipient = useCallback(
    (
      index: number,
      setter: React.Dispatch<React.SetStateAction<Recipient[]>>
    ) => {
      setter((prev) => prev.filter((_, i) => i !== index));
    },
    []
  );

  const handleSend = useCallback(async () => {
    if (to.length === 0) return;
    setSending(true);
    try {
      await api.createEmail({
        to,
        cc: showCc ? cc : undefined,
        bcc: showBcc ? bcc : undefined,
        subject,
        body_html: body,
        body_text: body.replace(/<[^>]*>/g, ""),
        in_reply_to: composeData?.in_reply_to,
        thread_id: composeData?.thread_id,
        is_draft: false,
      });
      closeCompose();
      refreshEmails();
    } catch {
      setSending(false);
    }
  }, [
    to,
    cc,
    bcc,
    showCc,
    showBcc,
    subject,
    body,
    composeData,
    closeCompose,
    refreshEmails,
  ]);

  const handleDiscard = useCallback(() => {
    closeCompose();
  }, [closeCompose]);

  const handleSaveDraft = useCallback(async () => {
    try {
      await api.saveDraft({
        to,
        cc: showCc ? cc : undefined,
        bcc: showBcc ? bcc : undefined,
        subject,
        body_html: body,
        body_text: body.replace(/<[^>]*>/g, ""),
        in_reply_to: composeData?.in_reply_to,
        thread_id: composeData?.thread_id,
        is_draft: true,
      });
    } catch {
      // Handle error silently
    }
    closeCompose();
  }, [to, cc, bcc, showCc, showBcc, subject, body, composeData, closeCompose]);

  const handleKeyDown = useCallback(
    (
      e: React.KeyboardEvent<HTMLInputElement>,
      input: string,
      setter: React.Dispatch<React.SetStateAction<Recipient[]>>,
      inputSetter: React.Dispatch<React.SetStateAction<string>>
    ) => {
      if (e.key === "Enter" || e.key === "Tab" || e.key === ",") {
        e.preventDefault();
        addRecipient(input, setter, inputSetter);
      }
    },
    [addRecipient]
  );

  const title =
    composeMode === "reply"
      ? "Reply"
      : composeMode === "forward"
        ? "Forward"
        : "New Message";

  if (state === "minimized") {
    return (
      <div className="fixed bottom-0 right-20 z-50 w-[280px]">
        <div
          className="compose-shadow flex h-10 cursor-pointer items-center justify-between rounded-t-lg bg-[#404040] px-3"
          onClick={() => setState("normal")}
        >
          <span className="text-sm font-medium text-white">{title}</span>
          <div className="flex items-center gap-1">
            <button
              onClick={(e) => {
                e.stopPropagation();
                setState("normal");
              }}
              className="flex h-6 w-6 items-center justify-center rounded hover:bg-gray-600"
            >
              <Maximize2 className="h-3.5 w-3.5 text-gray-300" />
            </button>
            <button
              onClick={(e) => {
                e.stopPropagation();
                handleSaveDraft();
              }}
              className="flex h-6 w-6 items-center justify-center rounded hover:bg-gray-600"
            >
              <X className="h-3.5 w-3.5 text-gray-300" />
            </button>
          </div>
        </div>
      </div>
    );
  }

  const isMaximized = state === "maximized";

  return (
    <div
      className={`fixed z-50 flex flex-col compose-shadow compose-animate ${
        isMaximized
          ? "bottom-0 left-[10%] right-[10%] top-[5%] rounded-t-lg"
          : "bottom-0 right-4 w-[560px] rounded-t-lg"
      }`}
      style={isMaximized ? undefined : { height: "520px" }}
    >
      {/* Header bar */}
      <div className="flex h-10 flex-shrink-0 items-center justify-between rounded-t-lg bg-[#404040] px-3">
        <span className="text-sm font-medium text-white">{title}</span>
        <div className="flex items-center gap-0.5">
          <button
            onClick={() => setState("minimized")}
            className="flex h-6 w-6 items-center justify-center rounded hover:bg-gray-600"
            title="Minimize"
          >
            <Minus className="h-4 w-4 text-gray-300" />
          </button>
          <button
            onClick={() =>
              setState(isMaximized ? "normal" : "maximized")
            }
            className="flex h-6 w-6 items-center justify-center rounded hover:bg-gray-600"
            title={isMaximized ? "Restore" : "Full screen"}
          >
            {isMaximized ? (
              <Minimize2 className="h-3.5 w-3.5 text-gray-300" />
            ) : (
              <Maximize2 className="h-3.5 w-3.5 text-gray-300" />
            )}
          </button>
          <button
            onClick={handleSaveDraft}
            className="flex h-6 w-6 items-center justify-center rounded hover:bg-gray-600"
            title="Save & close"
          >
            <X className="h-4 w-4 text-gray-300" />
          </button>
        </div>
      </div>

      {/* Form body */}
      <div className="flex flex-1 flex-col overflow-hidden bg-white">
        {/* To field */}
        <div className="flex items-center border-b border-gray-200 px-4 py-1">
          <span className="mr-2 text-sm text-gmail-text-secondary">To</span>
          <div className="flex flex-1 flex-wrap items-center gap-1">
            {to.map((r, i) => (
              <span
                key={i}
                className="flex items-center gap-1 rounded-full bg-[#E8EAED] px-2 py-0.5 text-sm"
              >
                {r.name || r.address}
                <button
                  onClick={() => removeRecipient(i, setTo)}
                  className="ml-0.5 text-gmail-text-secondary hover:text-gmail-text-primary"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
            <input
              ref={toInputRef}
              type="text"
              value={toInput}
              onChange={(e) => setToInput(e.target.value)}
              onKeyDown={(e) =>
                handleKeyDown(e, toInput, setTo, setToInput)
              }
              onBlur={() => addRecipient(toInput, setTo, setToInput)}
              className="min-w-[120px] flex-1 bg-transparent py-1 text-sm outline-none"
              placeholder={to.length === 0 ? "Recipients" : ""}
            />
          </div>
          <div className="flex items-center gap-1 text-xs">
            {!showCc && (
              <button
                onClick={() => setShowCc(true)}
                className="text-gmail-text-secondary hover:text-gmail-text-primary"
              >
                Cc
              </button>
            )}
            {!showBcc && (
              <button
                onClick={() => setShowBcc(true)}
                className="text-gmail-text-secondary hover:text-gmail-text-primary"
              >
                Bcc
              </button>
            )}
          </div>
        </div>

        {/* CC field */}
        {showCc && (
          <div className="flex items-center border-b border-gray-200 px-4 py-1">
            <span className="mr-2 text-sm text-gmail-text-secondary">Cc</span>
            <div className="flex flex-1 flex-wrap items-center gap-1">
              {cc.map((r, i) => (
                <span
                  key={i}
                  className="flex items-center gap-1 rounded-full bg-[#E8EAED] px-2 py-0.5 text-sm"
                >
                  {r.name || r.address}
                  <button
                    onClick={() => removeRecipient(i, setCc)}
                    className="ml-0.5 text-gmail-text-secondary hover:text-gmail-text-primary"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
              <input
                type="text"
                value={ccInput}
                onChange={(e) => setCcInput(e.target.value)}
                onKeyDown={(e) =>
                  handleKeyDown(e, ccInput, setCc, setCcInput)
                }
                onBlur={() => addRecipient(ccInput, setCc, setCcInput)}
                className="min-w-[120px] flex-1 bg-transparent py-1 text-sm outline-none"
              />
            </div>
          </div>
        )}

        {/* BCC field */}
        {showBcc && (
          <div className="flex items-center border-b border-gray-200 px-4 py-1">
            <span className="mr-2 text-sm text-gmail-text-secondary">
              Bcc
            </span>
            <div className="flex flex-1 flex-wrap items-center gap-1">
              {bcc.map((r, i) => (
                <span
                  key={i}
                  className="flex items-center gap-1 rounded-full bg-[#E8EAED] px-2 py-0.5 text-sm"
                >
                  {r.name || r.address}
                  <button
                    onClick={() => removeRecipient(i, setBcc)}
                    className="ml-0.5 text-gmail-text-secondary hover:text-gmail-text-primary"
                  >
                    <X className="h-3 w-3" />
                  </button>
                </span>
              ))}
              <input
                type="text"
                value={bccInput}
                onChange={(e) => setBccInput(e.target.value)}
                onKeyDown={(e) =>
                  handleKeyDown(e, bccInput, setBcc, setBccInput)
                }
                onBlur={() => addRecipient(bccInput, setBcc, setBccInput)}
                className="min-w-[120px] flex-1 bg-transparent py-1 text-sm outline-none"
              />
            </div>
          </div>
        )}

        {/* Subject field */}
        <div className="border-b border-gray-200 px-4 py-1">
          <input
            type="text"
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="Subject"
            className="w-full bg-transparent py-1 text-sm outline-none"
          />
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-4 pt-2">
          <textarea
            ref={bodyRef}
            value={body}
            onChange={(e) => setBody(e.target.value)}
            className="h-full w-full resize-none bg-transparent text-sm outline-none"
            placeholder="Compose email"
          />
        </div>

        {/* Bottom toolbar */}
        <div className="flex items-center justify-between border-t border-gray-200 px-2 py-1.5">
          <div className="flex items-center gap-0.5">
            <button
              onClick={handleSend}
              disabled={to.length === 0 || sending}
              className="flex items-center gap-2 rounded-full bg-gmail-blue px-6 py-2 text-sm font-medium text-white hover:bg-gmail-blue-hover hover:shadow-sm disabled:opacity-50"
            >
              {sending ? (
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              Send
            </button>
            <button
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Formatting options"
            >
              <Type className="h-4 w-4 text-gmail-text-secondary" />
            </button>
            <button
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="Attach files"
            >
              <Paperclip className="h-4 w-4 text-gmail-text-secondary" />
            </button>
            <button
              className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
              title="More options"
            >
              <MoreVertical className="h-4 w-4 text-gmail-text-secondary" />
            </button>
          </div>
          <button
            onClick={handleDiscard}
            className="flex h-8 w-8 items-center justify-center rounded-full hover:bg-gray-100"
            title="Discard draft"
          >
            <Trash2 className="h-4 w-4 text-gmail-text-secondary" />
          </button>
        </div>
      </div>
    </div>
  );
}
