import { useState, useRef, useEffect, useCallback } from 'react';
import { X, Minus, Maximize2, Minimize2, Trash2, Paperclip, Send } from 'lucide-react';
import { useEmailStore, useLabelStore } from '../store';
import { sendEmail, replyEmail, replyAllEmail, forwardEmail, saveDraft, fetchContacts } from '../api';
import { showToast } from './Toast';
import RichTextEditor from './RichTextEditor';
import type { Recipient, Contact, Attachment } from '../types';

type ComposeState = 'normal' | 'minimized' | 'maximized';

interface ComposeData {
  mode?: 'new' | 'reply' | 'reply-all' | 'forward';
  email_id?: string;
  to?: Recipient[];
  cc?: Recipient[];
  bcc?: Recipient[];
  subject?: string;
  body_html?: string;
  body_text?: string;
  in_reply_to?: string;
  thread_id?: string;
}

export default function ComposeModal() {
  const composeMode = useEmailStore((s) => s.composeMode);
  const composeData = useEmailStore((s) => s.composeData) as ComposeData | null;
  const closeCompose = useEmailStore((s) => s.closeCompose);
  const fetchEmails = useEmailStore((s) => s.fetchEmails);
  const { fetchLabels } = useLabelStore();

  const [to, setTo] = useState<Recipient[]>([]);
  const [cc, setCc] = useState<Recipient[]>([]);
  const [bcc, setBcc] = useState<Recipient[]>([]);
  const [subject, setSubject] = useState('');
  const [bodyHtml, setBodyHtml] = useState('');
  const [showCc, setShowCc] = useState(false);
  const [showBcc, setShowBcc] = useState(false);
  const [sending, setSending] = useState(false);
  const [state, setState] = useState<ComposeState>('normal');
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [suggestions, setSuggestions] = useState<Contact[]>([]);
  const [activeField, setActiveField] = useState<'to' | 'cc' | 'bcc' | null>(null);
  const [inputValue, setInputValue] = useState('');

  const toInputRef = useRef<HTMLInputElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (composeMode && composeData) {
      setTo(composeData.to || []);
      setCc(composeData.cc || []);
      setBcc(composeData.bcc || []);
      setSubject(composeData.subject || '');
      setBodyHtml(composeData.body_html || '');
      setShowCc((composeData.cc || []).length > 0);
      setShowBcc((composeData.bcc || []).length > 0);
      setAttachments([]);
      setState('normal');

      if ((composeData.to || []).length === 0) {
        setTimeout(() => toInputRef.current?.focus(), 100);
      }
    }
  }, [composeMode, composeData]);

  const searchContacts = useCallback(async (query: string) => {
    if (query.length < 1) { setSuggestions([]); return; }
    try {
      const contacts = await fetchContacts(query);
      setSuggestions(contacts.filter(c =>
        !to.some(r => r.address === c.email) &&
        !cc.some(r => r.address === c.email) &&
        !bcc.some(r => r.address === c.email)
      ));
    } catch { setSuggestions([]); }
  }, [to, cc, bcc]);

  useEffect(() => {
    const timer = setTimeout(() => { if (inputValue) searchContacts(inputValue); }, 200);
    return () => clearTimeout(timer);
  }, [inputValue, searchContacts]);

  if (!composeMode) return null;

  const addRecipient = (field: 'to' | 'cc' | 'bcc', value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return;
    const match = trimmed.match(/^(.+?)\s*<(.+?)>$/);
    const recipient: Recipient = match && match[1] && match[2] ? { name: match[1].trim(), address: match[2].trim() } : { address: trimmed };
    const setter = field === 'to' ? setTo : field === 'cc' ? setCc : setBcc;
    setter(prev => prev.some(r => r.address === recipient.address) ? prev : [...prev, recipient]);
    setInputValue('');
    setSuggestions([]);
  };

  const selectContact = (contact: Contact, field: 'to' | 'cc' | 'bcc') => {
    const recipient: Recipient = { name: contact.name, address: contact.email };
    const setter = field === 'to' ? setTo : field === 'cc' ? setCc : setBcc;
    setter(prev => prev.some(r => r.address === recipient.address) ? prev : [...prev, recipient]);
    setInputValue('');
    setSuggestions([]);
  };

  const removeRecipient = (field: 'to' | 'cc' | 'bcc', index: number) => {
    const setter = field === 'to' ? setTo : field === 'cc' ? setCc : setBcc;
    setter(prev => prev.filter((_, i) => i !== index));
  };

  const handleKeyDown = (e: React.KeyboardEvent, field: 'to' | 'cc' | 'bcc') => {
    if (['Enter', 'Tab', ','].includes(e.key) && inputValue.trim()) {
      e.preventDefault();
      addRecipient(field, inputValue);
    }
  };

  const handleAttach = () => fileInputRef.current?.click();

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    for (const file of Array.from(files)) {
      try {
        const attachment: Attachment = {
          id: `local-${Date.now()}-${file.name}`,
          email_id: '',
          filename: file.name,
          content_type: file.type || 'application/octet-stream',
          size_bytes: file.size,
          created_at: new Date().toISOString(),
        };
        setAttachments(prev => [...prev, attachment]);
      } catch {
        showToast('Failed to attach file');
      }
    }
    e.target.value = '';
  };

  const handleSend = async () => {
    if (to.length === 0) return;
    setSending(true);
    try {
      const bodyText = bodyHtml.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim();
      const data = {
        to,
        cc: showCc ? cc : [],
        bcc: showBcc ? bcc : [],
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        is_draft: false,
        in_reply_to: composeData?.in_reply_to || '',
        thread_id: composeData?.thread_id || '',
      };

      const mode = composeData?.mode || composeMode;
      if (mode === 'reply' && composeData?.email_id) {
        await replyEmail(composeData.email_id, data);
      } else if (mode === 'reply-all' && composeData?.email_id) {
        await replyAllEmail(composeData.email_id, data);
      } else if (mode === 'forward' && composeData?.email_id) {
        await forwardEmail(composeData.email_id, data);
      } else {
        await sendEmail(data);
      }
      showToast('Message sent', { action: { label: 'Undo', onClick: () => showToast('Undo not available for this message') } });
      closeCompose();
      fetchEmails();
      fetchLabels();
    } catch {
      showToast('Failed to send message');
    } finally {
      setSending(false);
    }
  };

  const handleDiscard = () => {
    closeCompose();
    showToast('Draft discarded');
  };

  const handleSaveDraft = async () => {
    try {
      const bodyText = bodyHtml.replace(/<[^>]*>/g, ' ').replace(/\s+/g, ' ').trim();
      await saveDraft({
        to,
        cc,
        bcc,
        subject,
        body_html: bodyHtml,
        body_text: bodyText,
        is_draft: true,
        in_reply_to: composeData?.in_reply_to || '',
        thread_id: composeData?.thread_id || '',
      });
      showToast('Draft saved');
    } catch { /* ignore */ }
    closeCompose();
  };

  const formatSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  };

  const RecipientField = ({ label, field, recipients, show }: { label: string; field: 'to' | 'cc' | 'bcc'; recipients: Recipient[]; show?: boolean }) => {
    if (show === false) return null;
    return (
      <div className="flex items-start gap-2 px-4 py-1.5 border-b border-gray-100">
        <span className="text-sm text-gray-500 pt-1 w-8">{label}</span>
        <div className="flex-1 flex flex-wrap items-center gap-1">
          {recipients.map((r, i) => (
            <span key={i} className="inline-flex items-center gap-1 px-2 py-0.5 bg-gray-100 rounded-full text-sm">
              <span>{r.name || r.address}</span>
              <button onClick={() => removeRecipient(field, i)} className="text-gray-400 hover:text-gray-600"><X size={14} /></button>
            </span>
          ))}
          <div className="relative flex-1 min-w-[120px]">
            <input
              ref={field === 'to' ? toInputRef : undefined}
              type="text"
              value={activeField === field ? inputValue : ''}
              onChange={e => { setInputValue(e.target.value); setActiveField(field); }}
              onKeyDown={e => handleKeyDown(e, field)}
              onFocus={() => setActiveField(field)}
              onBlur={() => { setTimeout(() => { if (inputValue.trim()) addRecipient(field, inputValue); setSuggestions([]); setActiveField(null); }, 200); }}
              className="w-full text-sm outline-none bg-transparent py-1"
              placeholder=""
            />
            {activeField === field && suggestions.length > 0 && (
              <div className="absolute top-full left-0 w-64 bg-white shadow-lg rounded-lg border border-gray-200 z-50 max-h-48 overflow-y-auto">
                {suggestions.map(contact => (
                  <button key={contact.id} onMouseDown={e => { e.preventDefault(); selectContact(contact, field); }} className="w-full px-3 py-2 text-left hover:bg-gray-50 flex items-center gap-2">
                    <div className="w-7 h-7 rounded-full bg-blue-100 text-blue-700 flex items-center justify-center text-xs font-medium">
                      {(contact.name || contact.email)[0]?.toUpperCase()}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{contact.name || contact.email}</div>
                      {contact.name && <div className="text-xs text-gray-500 truncate">{contact.email}</div>}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>
        {field === 'to' && (
          <div className="flex gap-1 text-sm text-gray-500 pt-1">
            {!showCc && <button onClick={() => setShowCc(true)} className="hover:text-gray-700">Cc</button>}
            {!showBcc && <button onClick={() => setShowBcc(true)} className="hover:text-gray-700">Bcc</button>}
          </div>
        )}
      </div>
    );
  };

  const title = composeData?.mode === 'reply' || composeMode === 'reply' ? 'Reply'
    : composeData?.mode === 'reply-all' ? 'Reply All'
    : composeData?.mode === 'forward' || composeMode === 'forward' ? 'Forward'
    : 'New Message';

  const widthClass = state === 'maximized' ? 'inset-4' : 'bottom-0 right-4 w-[560px]';
  const heightClass = state === 'maximized' ? '' : state === 'minimized' ? '' : 'h-[520px]';

  return (
    <div className={`fixed ${widthClass} z-50 flex flex-col bg-white rounded-t-lg compose-shadow compose-animate ${heightClass}`} style={state === 'minimized' ? { height: 'auto' } : undefined}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-2 bg-[#404040] rounded-t-lg cursor-pointer" onClick={() => state === 'minimized' && setState('normal')}>
        <span className="text-sm font-medium text-white">{title}</span>
        <div className="flex items-center gap-1">
          <button onClick={e => { e.stopPropagation(); setState(state === 'minimized' ? 'normal' : 'minimized'); }} className="p-1 hover:bg-white/10 rounded text-white"><Minus size={16} /></button>
          <button onClick={e => { e.stopPropagation(); setState(state === 'maximized' ? 'normal' : 'maximized'); }} className="p-1 hover:bg-white/10 rounded text-white">{state === 'maximized' ? <Minimize2 size={16} /> : <Maximize2 size={16} />}</button>
          <button onClick={e => { e.stopPropagation(); handleSaveDraft(); }} className="p-1 hover:bg-white/10 rounded text-white"><X size={16} /></button>
        </div>
      </div>

      {state !== 'minimized' && (
        <>
          <RecipientField label="To" field="to" recipients={to} />
          <RecipientField label="Cc" field="cc" recipients={cc} show={showCc} />
          <RecipientField label="Bcc" field="bcc" recipients={bcc} show={showBcc} />
          <input type="text" value={subject} onChange={e => setSubject(e.target.value)} placeholder="Subject" className="px-4 py-2 text-sm border-b border-gray-100 outline-none" />

          <div className="flex-1 overflow-y-auto">
            <RichTextEditor
              content={bodyHtml}
              onChange={setBodyHtml}
              placeholder="Compose your email..."
              autoFocus={to.length > 0}
            />
          </div>

          {/* Attachments */}
          {attachments.length > 0 && (
            <div className="px-4 py-2 border-t border-gray-100 flex flex-wrap gap-2">
              {attachments.map(a => (
                <div key={a.id} className="flex items-center gap-2 px-3 py-1.5 bg-gray-50 rounded-lg border border-gray-200 text-sm">
                  <Paperclip size={14} className="text-gray-400" />
                  <span className="truncate max-w-[150px]">{a.filename}</span>
                  <span className="text-gray-400 text-xs">{formatSize(a.size_bytes)}</span>
                  <button onClick={() => setAttachments(prev => prev.filter(at => at.id !== a.id))} className="text-gray-400 hover:text-gray-600"><X size={14} /></button>
                </div>
              ))}
            </div>
          )}

          {/* Footer */}
          <div className="flex items-center gap-2 px-4 py-2 border-t border-gray-100">
            <button onClick={handleSend} disabled={sending || to.length === 0} className="px-6 py-2 bg-[#1a73e8] text-white text-sm font-medium rounded-full hover:bg-[#1765cc] hover:shadow-md disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2">
              {sending ? <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <Send size={16} />}
              Send
            </button>
            <input ref={fileInputRef} type="file" multiple className="hidden" onChange={handleFileChange} />
            <button onClick={handleAttach} className="p-2 hover:bg-gray-100 rounded-full text-gray-600" title="Attach files">
              <Paperclip size={20} />
            </button>
            <div className="flex-1" />
            <button onClick={handleDiscard} className="p-2 hover:bg-gray-100 rounded-full text-gray-600" title="Discard"><Trash2 size={20} /></button>
          </div>
        </>
      )}
    </div>
  );
}
