import { useState, useEffect, useRef, useCallback, KeyboardEvent, ChangeEvent } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Gateway } from '../lib/gateway';
import { copyToClipboard } from '../lib/utils';
import { Icon } from '../components/Icon';
import { useToast } from '../components/Toast';

type ThinkingLevel = 'default' | 'low' | 'medium' | 'high';

interface Message {
  id?: string;
  role: 'user' | 'assistant';
  content: string;
  thinking?: string;
  createdAt?: string;
  timestamp?: number;
  agentId?: string;
  stopReason?: string;
  usage?: number;
}

interface ActiveTool {
  name: string;
  startedAt: number;
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

/** Extract thinking text from content block arrays. */
function extractThinking(message?: Record<string, unknown>): string {
  if (!message) return '';
  const content = message.content;
  if (Array.isArray(content)) {
    return content
      .filter((b: Record<string, unknown>) => b.type === 'thinking')
      .map((b: Record<string, unknown>) => (b.thinking as string) || (b.text as string) || '')
      .join('');
  }
  return '';
}

/** Extract total token usage from message or response data. */
function extractUsage(data?: Record<string, unknown>): number | undefined {
  if (!data) return undefined;
  const usage = data.usage as Record<string, unknown> | undefined;
  if (!usage) return undefined;
  // Try common token count fields
  const total = usage.total_tokens ?? usage.totalTokens ?? usage.output_tokens ?? usage.outputTokens;
  return typeof total === 'number' ? total : undefined;
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
  const [focusMode, setFocusMode] = useState(() => localStorage.getItem('openbot-chat-focus') === 'true');
  const [thinkingLevel, setThinkingLevel] = useState<ThinkingLevel>('default');
  const [streamingThinking, setStreamingThinking] = useState('');
  const [activeTools, setActiveTools] = useState<ActiveTool[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'reconnecting'>('connected');
  const [expandedThinking, setExpandedThinking] = useState<Record<string, boolean>>({});
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

  // Focus mode: toggle shell class and persist preference
  useEffect(() => {
    const shell = document.querySelector('.shell');
    if (shell) {
      if (focusMode) {
        shell.classList.add('shell--chat-focus');
      } else {
        shell.classList.remove('shell--chat-focus');
      }
    }
    localStorage.setItem('openbot-chat-focus', String(focusMode));
    return () => {
      const el = document.querySelector('.shell');
      if (el) el.classList.remove('shell--chat-focus');
    };
  }, [focusMode]);

  // Auto-resize textarea
  const handleTextareaInput = useCallback((e: ChangeEvent<HTMLTextAreaElement>) => {
    setText(e.target.value);
    const ta = e.target;
    ta.style.height = 'auto';
    ta.style.height = Math.min(ta.scrollHeight, 150) + 'px';
  }, []);

  const toggleFocus = useCallback(() => {
    setFocusMode(prev => !prev);
  }, []);

  const handleThinkingChange = useCallback((e: ChangeEvent<HTMLSelectElement>) => {
    setThinkingLevel(e.target.value as ThinkingLevel);
  }, []);

  // OpenClaw-format event listener (single 'chat' event with state discriminator)
  useEffect(() => {
    const unsubChat = gw.on('event:chat', (payload: unknown) => {
      const data = payload as Record<string, unknown>;
      if (!data) return;
      // Only process events for current session
      if (data.sessionKey && data.sessionKey !== currentSessionKey) return;

      switch (data.state) {
        case 'delta': {
          const msg = data.message as Record<string, unknown>;
          const text = extractText(msg);
          const thinking = extractThinking(msg);
          if (text) {
            setStreamingText(text);
            setStreaming(true);
          }
          if (thinking) {
            setStreamingThinking(thinking);
          }
          break;
        }
        case 'final': {
          const message = data.message as Record<string, unknown> | undefined;
          if (message) {
            const content = extractText(message);
            const thinking = extractThinking(message);
            const role = (message.role as string) || 'assistant';
            const usage = extractUsage(data as Record<string, unknown>)
              ?? extractUsage(message);
            if (content) {
              setMessages(prev => {
                // Deduplicate: check if we already have this content as last message
                const last = prev[prev.length - 1];
                if (last && last.role === role && last.content === content) return prev;
                return [...prev, {
                  role: role as 'user' | 'assistant',
                  content,
                  thinking: thinking || undefined,
                  timestamp: message.timestamp as number | undefined,
                  stopReason: message.stopReason as string | undefined,
                  usage,
                }];
              });
            }
          }
          setStreamingText('');
          setStreamingThinking('');
          setStreaming(false);
          setActiveTools([]);
          break;
        }
        case 'aborted':
          setStreamingText('');
          setStreamingThinking('');
          setStreaming(false);
          setActiveTools([]);
          break;
        case 'error': {
          setStreamingText('');
          setStreamingThinking('');
          setStreaming(false);
          setActiveTools([]);
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

    // Agent event listener for tool call indicators
    const unsubAgent = gw.on('event:agent', (payload: unknown) => {
      const data = payload as Record<string, unknown>;
      if (!data) return;
      if (data.sessionKey && data.sessionKey !== currentSessionKey) return;

      if (data.type === 'tool_start') {
        const toolName = (data.tool as string) || (data.name as string) || 'tool';
        setActiveTools(prev => {
          if (prev.some(t => t.name === toolName)) return prev;
          return [...prev, { name: toolName, startedAt: Date.now() }];
        });
      } else if (data.type === 'tool_end') {
        const toolName = (data.tool as string) || (data.name as string) || 'tool';
        setActiveTools(prev => prev.filter(t => t.name !== toolName));
      }
    });

    // Connection status listeners
    const unsubDisconnected = gw.on('disconnected', () => {
      setConnectionStatus('reconnecting');
    });
    const unsubReconnected = gw.on('reconnected', () => {
      setConnectionStatus('connected');
    });
    const unsubConnected = gw.on('connected', () => {
      setConnectionStatus('connected');
    });

    return () => {
      unsubChat();
      unsubMessage();
      unsubTyping();
      unsubDone();
      unsubAgent();
      unsubDisconnected();
      unsubReconnected();
      unsubConnected();
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

    // Reset textarea height after clearing
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
    }

    // Generate idempotency key (OpenClaw compat)
    const idempotencyKey = crypto.randomUUID();

    try {
      const sendParams: Record<string, unknown> = {
        sessionKey: currentSessionKey,
        message: trimmed,
        idempotencyKey,
        agentId: 'main',
      };
      if (thinkingLevel !== 'default') {
        sendParams.thinkingLevel = thinkingLevel;
      }
      const res = await gw.rpc('chat.send', sendParams);

      // Capture session ID from first response
      if (res.sessionId && !sessionId) {
        setSessionId(res.sessionId as string);
        loadSessions();
      }

      // chat.send now returns {runId, status: "started"} without content.
      // Assistant messages arrive via event:chat events (delta/final), so
      // we do NOT try to display content from the RPC response.
    } catch (err) {
      const errorMsg: Message = {
        role: 'assistant',
        content: `Error: ${err instanceof Error ? err.message : 'unknown error'}`,
      };
      setMessages((prev) => [...prev, errorMsg]);
    } finally {
      setSending(false);
      // Note: streaming/streamingText are cleared by the 'final' event,
      // but we clear sending here so the input re-enables immediately.
      textareaRef.current?.focus();
    }
  }, [text, sending, gw, sessionId, currentSessionKey, loadSessions, thinkingLevel]);

  const handleAbort = useCallback(async () => {
    try {
      await gw.rpc('chat.abort', { sessionKey: currentSessionKey });
    } catch {
      // ignore abort errors
    }
    setStreaming(false);
    setSending(false);
    setStreamingText('');
    setStreamingThinking('');
    setActiveTools([]);
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

  const toggleThinking = useCallback((key: string) => {
    setExpandedThinking(prev => ({ ...prev, [key]: !prev[key] }));
  }, []);

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
        <select
          className="chat-thinking-select"
          value={thinkingLevel}
          onChange={handleThinkingChange}
          title="Thinking level"
        >
          <option value="default">Thinking: Default</option>
          <option value="low">Thinking: Low</option>
          <option value="medium">Thinking: Medium</option>
          <option value="high">Thinking: High</option>
        </select>
        <button
          className={`chat-focus-btn${focusMode ? ' active' : ''}`}
          onClick={toggleFocus}
          title={focusMode ? 'Exit focus mode' : 'Enter focus mode'}
        >
          <Icon name={focusMode ? 'minimize' : 'maximize'} size={14} />
          <span>{focusMode ? 'Unfocus' : 'Focus'}</span>
        </button>
      </div>

      {connectionStatus === 'reconnecting' && (
        <div style={{
          padding: '4px 12px',
          background: '#fef3cd',
          color: '#856404',
          fontSize: '12px',
          textAlign: 'center',
          borderBottom: '1px solid #ffc107',
        }}>
          Reconnecting...
        </div>
      )}

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
            <div className="chat-group-row">
              <div className={`chat-avatar ${group.role}`}>
                {group.role === 'user' ? 'Y' : 'AI'}
              </div>
              <div className="chat-group-content">
                {group.messages.map((msg, mi) => {
                  const isLastAssistant =
                    streaming &&
                    !streamingText &&
                    msg.role === 'assistant' &&
                    gi === groups.length - 1 &&
                    mi === group.messages.length - 1;
                  const thinkingKey = `${gi}-${mi}`;
                  const isThinkingExpanded = expandedThinking[thinkingKey] ?? false;
                  return (
                    <div key={msg.id || `${gi}-${mi}`} className="chat-bubble-wrapper">
                      {/* Thinking block (collapsible) */}
                      {msg.thinking && (
                        <div style={{ marginBottom: '4px' }}>
                          <button
                            onClick={() => toggleThinking(thinkingKey)}
                            style={{
                              background: 'none',
                              border: '1px solid var(--color-border, #ddd)',
                              borderRadius: '4px',
                              padding: '2px 8px',
                              fontSize: '11px',
                              color: 'var(--color-text-muted, #888)',
                              cursor: 'pointer',
                            }}
                          >
                            {isThinkingExpanded
                              ? 'Hide thinking'
                              : `Show thinking (${msg.thinking.length} chars)`}
                          </button>
                          {isThinkingExpanded && (
                            <div style={{
                              marginTop: '4px',
                              padding: '8px',
                              background: 'var(--color-bg-subtle, #f6f6f6)',
                              border: '1px solid var(--color-border, #ddd)',
                              borderRadius: '4px',
                              fontSize: '12px',
                              color: 'var(--color-text-muted, #888)',
                              whiteSpace: 'pre-wrap',
                              maxHeight: '300px',
                              overflow: 'auto',
                            }}>
                              {msg.thinking}
                            </div>
                          )}
                        </div>
                      )}
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
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px', flexWrap: 'wrap' }}>
                        {(msg.createdAt || msg.timestamp) && (
                          <div className="chat-stamp">{formatTime(msg.createdAt, msg.timestamp)}</div>
                        )}
                        {msg.usage != null && (
                          <span style={{
                            fontSize: '10px',
                            color: 'var(--color-text-muted, #999)',
                            background: 'var(--color-bg-subtle, #f0f0f0)',
                            padding: '1px 6px',
                            borderRadius: '8px',
                          }}>
                            {msg.usage} tokens
                          </span>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          </div>
        ))}

        {/* Streaming delta text display */}
        {streaming && streamingText && (
          <div className="chat-group chat-group--assistant">
            <div className="chat-group-label">Assistant</div>
            <div className="chat-group-row">
              <div className="chat-avatar assistant">AI</div>
              <div className="chat-group-content">
                {/* Streaming thinking indicator */}
                {streamingThinking && (
                  <div style={{ marginBottom: '4px' }}>
                    <button
                      onClick={() => toggleThinking('streaming')}
                      style={{
                        background: 'none',
                        border: '1px solid var(--color-border, #ddd)',
                        borderRadius: '4px',
                        padding: '2px 8px',
                        fontSize: '11px',
                        color: 'var(--color-text-muted, #888)',
                        cursor: 'pointer',
                      }}
                    >
                      {expandedThinking['streaming']
                        ? 'Hide thinking'
                        : `Show thinking (${streamingThinking.length} chars)`}
                    </button>
                    {expandedThinking['streaming'] && (
                      <div style={{
                        marginTop: '4px',
                        padding: '8px',
                        background: 'var(--color-bg-subtle, #f6f6f6)',
                        border: '1px solid var(--color-border, #ddd)',
                        borderRadius: '4px',
                        fontSize: '12px',
                        color: 'var(--color-text-muted, #888)',
                        whiteSpace: 'pre-wrap',
                        maxHeight: '200px',
                        overflow: 'auto',
                      }}>
                        {streamingThinking}
                      </div>
                    )}
                  </div>
                )}
                <div className="chat-bubble-wrapper">
                  <div className="chat-bubble chat-bubble--assistant streaming">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {streamingText}
                    </ReactMarkdown>
                  </div>
                  {/* Active tool badges */}
                  {activeTools.length > 0 && (
                    <div style={{
                      display: 'flex',
                      gap: '4px',
                      flexWrap: 'wrap',
                      marginTop: '4px',
                    }}>
                      {activeTools.map(tool => (
                        <span key={tool.name} style={{
                          fontSize: '10px',
                          padding: '2px 8px',
                          borderRadius: '10px',
                          background: 'var(--color-accent, #4f46e5)',
                          color: '#fff',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}>
                          <span style={{
                            width: '6px',
                            height: '6px',
                            borderRadius: '50%',
                            background: '#4ade80',
                            display: 'inline-block',
                          }} />
                          {tool.name}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Typing indicator (no delta text yet) */}
        {streaming && !streamingText && messages.length > 0 && (
          <div className="chat-group chat-group--assistant">
            <div className="chat-group-label">Assistant</div>
            <div className="chat-group-row">
              <div className="chat-avatar assistant">AI</div>
              <div className="chat-group-content">
                <div className="chat-bubble-wrapper">
                  <div className="chat-bubble streaming">
                    <span className="chat-reading-dots">
                      <span />
                      <span />
                      <span />
                    </span>
                  </div>
                  {/* Active tool badges during typing */}
                  {activeTools.length > 0 && (
                    <div style={{
                      display: 'flex',
                      gap: '4px',
                      flexWrap: 'wrap',
                      marginTop: '4px',
                    }}>
                      {activeTools.map(tool => (
                        <span key={tool.name} style={{
                          fontSize: '10px',
                          padding: '2px 8px',
                          borderRadius: '10px',
                          background: 'var(--color-accent, #4f46e5)',
                          color: '#fff',
                          display: 'inline-flex',
                          alignItems: 'center',
                          gap: '4px',
                        }}>
                          <span style={{
                            width: '6px',
                            height: '6px',
                            borderRadius: '50%',
                            background: '#4ade80',
                            display: 'inline-block',
                          }} />
                          {tool.name}
                        </span>
                      ))}
                    </div>
                  )}
                </div>
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
          onChange={handleTextareaInput}
          onKeyDown={handleKeyDown}
          placeholder="Type a message... (Enter to send, Shift+Enter for newline)"
          rows={1}
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
