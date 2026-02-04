import { useEffect, useState, useCallback } from "react";
import { X } from "lucide-react";

interface ToastData {
  id: string;
  message: string;
  action?: { label: string; onClick: () => void };
  duration?: number;
}

type ToastListener = (toast: ToastData) => void;

let toastListeners: ToastListener[] = [];

export function showToast(
  message: string,
  options?: {
    action?: { label: string; onClick: () => void };
    duration?: number;
  }
) {
  const toast: ToastData = {
    id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    message,
    action: options?.action,
    duration: options?.duration,
  };
  toastListeners.forEach((fn) => fn(toast));
}

export default function ToastContainer() {
  const [toasts, setToasts] = useState<ToastData[]>([]);

  useEffect(() => {
    const handler: ToastListener = (toast) => {
      setToasts((prev) => [...prev, toast]);

      const autoDismissMs = toast.duration ?? 5000;
      setTimeout(() => {
        setToasts((prev) => prev.filter((t) => t.id !== toast.id));
      }, autoDismissMs);
    };

    toastListeners.push(handler);
    return () => {
      toastListeners = toastListeners.filter((fn) => fn !== handler);
    };
  }, []);

  const dismiss = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-4 left-4 z-[9999] flex flex-col gap-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="compose-animate flex items-center gap-3 rounded-lg bg-[#323232] px-4 py-3 text-sm text-white shadow-lg"
          style={{ minWidth: 300, maxWidth: 480 }}
        >
          <span className="flex-1">{toast.message}</span>

          {toast.action && (
            <button
              onClick={() => {
                toast.action!.onClick();
                dismiss(toast.id);
              }}
              className="whitespace-nowrap font-medium text-[#8AB4F8] hover:underline"
            >
              {toast.action.label}
            </button>
          )}

          <button
            onClick={() => dismiss(toast.id)}
            className="flex-shrink-0 text-gray-400 hover:text-white"
            aria-label="Dismiss"
          >
            <X size={16} />
          </button>
        </div>
      ))}
    </div>
  );
}
