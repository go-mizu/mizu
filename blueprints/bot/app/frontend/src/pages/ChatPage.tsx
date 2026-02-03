import { useState, useEffect, useRef, useCallback, KeyboardEvent } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Gateway } from '../lib/gateway';
import { copyToClipboard } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

interface Message {
  id?: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt?: string;
  timestamp?: number;
  agentId?: string;
  stopReason?: string;
}

interface Session {
  id: string;
  title?: string;
  displayName?: string;
  channelType?: string;
  updatedAt?: string;
}

interface MessageGroup {
  role: 'user' | 'assistant';
  messages: Message[];
}

interface ChatPageProps {
  gw: Gateway;
}

/** Extract text from OpenClaw content format (array of {type, text} blocks or plain string). */
function extractText(message?: Record<string, unknown>): string {
  if (!message) return '';
  const content = message.content;
  if (typeof content === 'string') return content;
  if (Array.isArray(content)) {
    return content
      .filter((b: Record<string, unknown>) => b.type === 'text')
      .map((b: Record<string, unknown>) => b.text as string)
      .join('');
  }
  return '';
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

function formatTime(iso?: string, ts?: number): string {
  const date = ts ? new Date(ts) : iso ? new Date(iso) : null;
  if (!date) return '';
  try {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  } catch {
    return '';
  }
}

export function ChatPage({ gw }: ChatPageProps) {
  const { toast } = useToast();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [sessionId, setSessionId] = useState('');
  const [currentSessionKey, setCurrentSessionKey] = useState('agent:main:main');
  const [messages, setMessages] = useState<Message[]>([]);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [streamingText, setStreamingText] = useState('');
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

  const loadMessages = useCallback(async (sid: string, skey: string) => {
    if (!sid && !skey) {
      setMessages([]);
      return;
    }
    try {
      // Use sessionKey if available (OpenClaw compat), fall back to sessionId
      const params: Record<string, unknown> = { limit: 200 };
      if (skey) params.sessionKey = skey;
      if (sid) params.sessionId = sid;
      const res = await gw.rpc('chat.history', params);
      const rawList = (res.messages ?? []) as Record<string, unknown>[];
      // Convert OpenClaw format messages to local format
      const normalized: Message[] = rawList.map((m) => {
        const content = typeof m.content === 'string'
          ? m.content as string
          : extractText(m);
        return {
          role: m.role as 'user' | 'assistant',
          content,
          timestamp: m.timestamp as number | undefined,
          createdAt: m.createdAt as string | undefined,
          stopReason: m.stopReason as string | undefined,
        };
      });
      setMessages(normalized);
      // Capture sessionId from response if we used sessionKey
      if (res.sessionId && !sid) {
        setSessionId(res.sessionId as string);
      }
    } catch {
      setMessages([]);
    }
  }, [gw]);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  useEffect(() => {
    loadMessages(sessionId, currentSessionKey);
  }, [sessionId, currentSessionKey, loadMessages]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, sending, streaming, streamingText]);

  // OpenClaw-format event listener (single 'chat' event with state discriminator)
  useEffect(() => {
    const unsubChat = gw.on('event:chat', (payload: unknown) => {
      const data = payload as Record<string, unknown>;
      if (!data) return;
      // Only process events for current session
      if (data.sessionKey && data.sessionKey !== currentSessionKey) return;

      switch (data.state) {
        case 'delta': {
          const text = extractText(data.message as Record<string, unknown>);
          if (text) {
            setStreamingText(text);
            setStreaming(true);
          }
          break;
        }
        case 'final': {
          const message = data.message as Record<string, unknown> | undefined;
          if (message) {
            const content = extractText(message);
            const role = (message.role as string) || 'assistant';
            if (content) {
              setMessages(prev => {
                // Deduplicate: check if we already have this content as last message
                const last = prev[prev.length - 1];
                if (last && last.role === role && last.content === content) return prev;
                return [...prev, {
                  role: role as 'user' | 'assistant',
                  content,
                  timestamp: message.timestamp as number | undefined,
                  stopReason: message.stopReason as string | undefined,
                }];
              });
            }
          }
          setStreamingText('');
          setStreaming(false);
          break;
        }
        case 'aborted':
          setStreamingText('');
          setStreaming(false);
          break;
        case 'error': {
          setStreamingText('');
          setStreaming(false);
          const errMsg = (data.errorMessage as string) || 'Chat request failed';
          setMessages(prev => [...prev, {
            role: 'assistant' as const,
            content: `Error: ${errMsg}`,
          }]);
          break;
        }
      }
    });

    // Legacy event listeners (backward compat during transition)
    const unsubMessage = gw.on('event:chat.message', (payload: unknown) => {
      const data = payload as { sessionId?: string; message?: Message };
      if (!data?.message) return;
      if (data.sessionId === sessionId || !sessionId) {
        setMessages((prev) => {
          if (data.message!.id && prev.some((m) => m.id === data.message!.id)) return prev;
          return [...prev, data.message!];
        });
      }
    });

    const unsubTyping = gw.on('event:chat.typing', (payload: unknown) => {
      const data = payload as { sessionId?: string };
      if (data?.sessionId === sessionId || !sessionId) {
        setStreaming(true);
      }
    });

    const unsubDone = gw.on('event:chat.done', (payload: unknown) => {
      const data = payload as { sessionId?: string };
      if (data?.sessionId === sessionId || !sessionId) {
        setStreaming(false);
      }
    });

    return () => {
      unsubChat();
      unsubMessage();
      unsubTyping();
      unsubDone();
    };
  }, [gw, sessionId, currentSessionKey]);

  const handleSend = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || sending) return;

    // Add user message locally for immediate feedback
    const userMsg: Message = { role: 'user', content: trimmed, timestamp: Date.now() };
    setMessages(prev => [...prev, userMsg]);
    setText('');
    setSending(true);
    setStreaming(true);

    // Generate idempotency key (OpenClaw compat)
    const idempotencyKey = crypto.randomUUID();

    try {
      const res = await gw.rpc('chat.send', {
        sessionKey: currentSessionKey,
        message: trimmed,
        idempotencyKey,
        agentId: 'main',
      });

      // Capture session ID from first response
      if (res.sessionId && !sessionId) {
        setSessionId(res.sessionId as string);
        loadSessions();
      }

      // The broadcast events handle adding messages, but as a fallback
      // ensure the assistant response is shown if broadcast didn't fire
      const reply = (res.content ?? '') as string;
      if (reply) {
        setMessages((prev) => {
          if (res.messageId && prev.some((m) => m.id === res.messageId)) return prev;
          // Also check if content already present from event
          const last = prev[prev.length - 1];
          if (last && last.role === 'assistant' && last.content === reply) return prev;
          return [...prev, {
            id: res.messageId as string | undefined,
            role: 'assistant' as const,
            content: reply,
          }];
        });
      }
    } catch (err) {
      const errorMsg: Message = {
        role: 'assistant',
        content: `Error: ${err instanceof Error ? err.message : 'unknown error'}`,
      };
      setMessages((prev) => [...prev, errorMsg]);
    } finally {
      setSending(false);
      setStreaming(false);
      setStreamingText('');
      textareaRef.current?.focus();
    }
  }, [text, sending, gw, sessionId, currentSessionKey, loadSessions]);

  const handleAbort = useCallback(async () => {
    try {
      await gw.rpc('chat.abort', { sessionKey: currentSessionKey });
    } catch {
      // ignore abort errors
    }
    setStreaming(false);
    setSending(false);
    setStreamingText('');
  }, [gw, currentSessionKey]);

  const handleNewConversation = useCallback(async () => {
    try {
      await gw.rpc('chat.new', { agentId: 'main' });
    } catch {
      // ignore errors
    }
    setSessionId('');
    setCurrentSessionKey('agent:main:main');
    setMessages([]);
    setStreamingText('');
    loadSessions();
    textareaRef.current?.focus();
  }, [gw, loadSessions]);

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
      setStreamingText('');
      setSessionId(newSid);
      // Build sessionKey from session ID for OpenClaw compat
      if (newSid) {
        setCurrentSessionKey('agent:main:' + newSid);
      } else {
        setCurrentSessionKey('agent:main:main');
      }
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
              {s.channelType && s.channelType !== 'webhook' ? ` [${s.channelType}]` : ''}
            </option>
          ))}
        </select>
        <button
          className="btn btn--sm chat-new-btn"
          onClick={handleNewConversation}
          title="Start a new conversation"
        >
          <Icon name="plus" size={14} />
          <span>New</span>
        </button>
      </div>

      <div className="chat-messages">
        {groups.length === 0 && !sending && !streamingText && (
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
                !streamingText &&
                msg.role === 'assistant' &&
                gi === groups.length - 1 &&
                mi === group.messages.length - 1;
              return (
                <div key={msg.id || `${gi}-${mi}`} className="chat-bubble-wrapper">
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
                  {(msg.createdAt || msg.timestamp) && (
                    <div className="chat-stamp">{formatTime(msg.createdAt, msg.timestamp)}</div>
                  )}
                </div>
              );
            })}
          </div>
        ))}

        {/* Streaming delta text display */}
        {streaming && streamingText && (
          <div className="chat-group chat-group--assistant">
            <div className="chat-group-label">Assistant</div>
            <div className="chat-bubble-wrapper">
              <div className="chat-bubble chat-bubble--assistant streaming">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>
                  {streamingText}
                </ReactMarkdown>
              </div>
            </div>
          </div>
        )}

        {/* Typing indicator (no delta text yet) */}
        {streaming && !streamingText && messages.length > 0 && (
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
