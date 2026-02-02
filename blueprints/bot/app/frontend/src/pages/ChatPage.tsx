import { useState, useEffect, useRef, useCallback, KeyboardEvent } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Gateway } from '../lib/gateway';
import { copyToClipboard } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Message {
  role: 'user' | 'assistant';
  content: string;
}

interface Session {
  id: string;
  title?: string;
  displayName?: string;
}

interface MessageGroup {
  role: 'user' | 'assistant';
  messages: Message[];
}

interface ChatPageProps {
  gw: Gateway;
}

function groupMessages(messages: Message[]): MessageGroup[] {
  const groups: MessageGroup[] = [];
  for (const msg of messages) {
    const last = groups[groups.length - 1];
    if (last && last.role === msg.role) {
      last.messages.push(msg);
    } else {
      groups.push({ role: msg.role, messages: [msg] });
    }
  }
  return groups;
}

export function ChatPage({ gw }: ChatPageProps) {
  const { toast } = useToast();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [sessionId, setSessionId] = useState('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const loadSessions = useCallback(async () => {
    try {
      const res = await gw.rpc('sessions.list', { limit: 50 });
      const list = (res.sessions ?? []) as Session[];
      setSessions(list);
    } catch {
      // ignore load errors
    }
  }, [gw]);

  const loadMessages = useCallback(async (sid: string) => {
    if (!sid) {
      setMessages([]);
      return;
    }
    try {
      const res = await gw.rpc('chat.history', { sessionId: sid, limit: 50 });
      const list = (res.messages ?? []) as Message[];
      setMessages(list);
    } catch {
      setMessages([]);
    }
  }, [gw]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  useEffect(() => {
    loadMessages(sessionId);
  }, [sessionId, loadMessages]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, sending, streaming]);

  const handleSend = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || sending) return;

    const userMsg: Message = { role: 'user', content: trimmed };
    setMessages((prev) => [...prev, userMsg]);
    setText('');
    setSending(true);
    setStreaming(true);

    try {
      const res = await gw.rpc('chat.send', { sessionId, message: trimmed });
      const reply = (res.content ?? res.message ?? res.response ?? '') as string;
      const assistantMsg: Message = { role: 'assistant', content: reply };
      setMessages((prev) => [...prev, assistantMsg]);
    } catch (err) {
      const errorMsg: Message = {
        role: 'assistant',
        content: `Error: ${err instanceof Error ? err.message : 'unknown error'}`,
      };
      setMessages((prev) => [...prev, errorMsg]);
    } finally {
      setSending(false);
      setStreaming(false);
      textareaRef.current?.focus();
    }
  }, [text, sending, gw, sessionId]);

  const handleAbort = useCallback(async () => {
    try {
      await gw.rpc('chat.abort');
    } catch {
      // ignore abort errors
    }
    setStreaming(false);
    setSending(false);
  }, [gw]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        handleSend();
      }
    },
    [handleSend],
  );

  const handleCopy = useCallback(
    async (content: string) => {
      try {
        await copyToClipboard(content);
        toast('Copied!', 'success');
      } catch {
        toast('Failed to copy', 'error');
      }
    },
    [toast],
  );

  const handleSessionChange = useCallback(
    (e: React.ChangeEvent<HTMLSelectElement>) => {
      const newSid = e.target.value;
      setMessages([]);
      setSessionId(newSid);
    },
    [],
  );

  const groups = groupMessages(messages);

  return (
    <div className="chat-page">
      <div className="chat-session-bar">
        <label htmlFor="session-select">Session:</label>
        <select
          id="session-select"
          value={sessionId}
          onChange={handleSessionChange}
        >
          <option value="">New conversation</option>
          {sessions.map((s) => (
            <option key={s.id} value={s.id}>
              {s.displayName || s.title || s.id}
            </option>
          ))}
        </select>
      </div>

      <div className="chat-messages">
        {groups.length === 0 && !sending && (
          <div className="chat-empty">
            <Icon name="messageSquare" size={48} />
            <p>No messages yet. Start a conversation below.</p>
          </div>
        )}

        {groups.map((group, gi) => (
          <div key={gi} className={`chat-group chat-group--${group.role}`}>
            <div className="chat-group-label">
              {group.role === 'user' ? 'You' : 'Assistant'}
            </div>
            {group.messages.map((msg, mi) => {
              const isLastAssistant =
                streaming &&
                msg.role === 'assistant' &&
                gi === groups.length - 1 &&
                mi === group.messages.length - 1;
              return (
                <div key={`${gi}-${mi}`} className="chat-bubble-wrapper">
                  <div
                    className={`chat-bubble chat-bubble--${msg.role}${isLastAssistant ? ' streaming' : ''}`}
                  >
                    {msg.role === 'assistant' ? (
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>
                        {msg.content}
                      </ReactMarkdown>
                    ) : (
                      <span>{msg.content}</span>
                    )}
                    <button
                      className="chat-copy-btn"
                      onClick={() => handleCopy(msg.content)}
                      title="Copy to clipboard"
                    >
                      <Icon name="copy" size={14} />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        ))}

        {streaming && (
          <div className="chat-group chat-group--assistant">
            <div className="chat-group-label">Assistant</div>
            <div className="chat-bubble-wrapper">
              <div className="chat-bubble streaming">
                <span className="chat-reading-dots">
                  <span />
                  <span />
                  <span />
                </span>
              </div>
            </div>
          </div>
        )}

        <div ref={bottomRef} />
      </div>

      <div className="chat-input-bar">
        <textarea
          ref={textareaRef}
          className="chat-textarea"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Type a message... (Enter to send, Shift+Enter for newline)"
          rows={2}
          disabled={sending}
        />
        {streaming ? (
          <button
            className="chat-send-btn chat-abort-btn"
            onClick={handleAbort}
            title="Stop generation"
          >
            <Icon name="x" size={18} />
            <span>Stop</span>
          </button>
        ) : (
          <button
            className="chat-send-btn"
            onClick={handleSend}
            disabled={sending || !text.trim()}
            title="Send message"
          >
            <Icon name="send" size={18} />
          </button>
        )}
      </div>
    </div>
  );
}
