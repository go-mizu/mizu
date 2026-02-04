import { useEffect, useState, useCallback } from 'react';
import { X } from 'lucide-react';

interface Toast {
  id: string;
  message: string;
  action?: { label: string; onClick: () => void };
  duration?: number;
}

let toastListeners: ((toast: Toast) => void)[] = [];

export function showToast(message: string, options?: { action?: { label: string; onClick: () => void }; duration?: number }) {
  const toast: Toast = { id: Date.now().toString(), message, ...options };
  toastListeners.forEach(fn => fn(toast));
}

export default function ToastContainer() {
  const [toasts, setToasts] = useState<Toast[]>([]);

  useEffect(() => {
    const handler = (toast: Toast) => {
      setToasts(prev => [...prev, toast]);
      setTimeout(() => {
        setToasts(prev => prev.filter(t => t.id !== toast.id));
      }, toast.duration || 5000);
    };
    toastListeners.push(handler);
    return () => { toastListeners = toastListeners.filter(fn => fn !== handler); };
  }, []);

  const dismiss = useCallback((id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  }, []);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 left-4 z-[9999] flex flex-col gap-2">
      {toasts.map(toast => (
        <div key={toast.id} className="flex items-center gap-3 bg-[#323232] text-white px-4 py-3 rounded-lg shadow-lg text-sm min-w-[300px] compose-animate">
          <span className="flex-1">{toast.message}</span>
          {toast.action && (
            <button onClick={() => { toast.action!.onClick(); dismiss(toast.id); }} className="text-[#8ab4f8] font-medium hover:underline whitespace-nowrap">
              {toast.action.label}
            </button>
          )}
          <button onClick={() => dismiss(toast.id)} className="text-gray-400 hover:text-white">
            <X size={16} />
          </button>
        </div>
      ))}
    </div>
  );
}
