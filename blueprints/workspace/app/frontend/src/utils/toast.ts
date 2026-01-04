import toast, { ToastOptions } from 'react-hot-toast'

// Custom toast styles matching our design system
const baseOptions: ToastOptions = {
  duration: 4000,
  style: {
    padding: '12px 16px',
    borderRadius: 'var(--radius-md)',
    background: 'var(--bg-primary)',
    color: 'var(--text-primary)',
    border: '1px solid var(--border-color)',
    boxShadow: 'var(--shadow-lg)',
    fontSize: '14px',
  },
}

export const showToast = {
  success: (message: string, options?: ToastOptions) =>
    toast.success(message, {
      ...baseOptions,
      ...options,
      style: {
        ...baseOptions.style,
        borderLeft: '3px solid var(--success-color)',
        ...options?.style,
      },
      iconTheme: {
        primary: 'var(--success-color)',
        secondary: '#fff',
      },
    }),

  error: (message: string, options?: ToastOptions) =>
    toast.error(message, {
      ...baseOptions,
      duration: 5000,
      ...options,
      style: {
        ...baseOptions.style,
        borderLeft: '3px solid var(--danger-color)',
        ...options?.style,
      },
      iconTheme: {
        primary: 'var(--danger-color)',
        secondary: '#fff',
      },
    }),

  info: (message: string, options?: ToastOptions) =>
    toast(message, {
      ...baseOptions,
      ...options,
      style: {
        ...baseOptions.style,
        borderLeft: '3px solid var(--accent-color)',
        ...options?.style,
      },
      icon: 'ℹ️',
    }),

  warning: (message: string, options?: ToastOptions) =>
    toast(message, {
      ...baseOptions,
      ...options,
      style: {
        ...baseOptions.style,
        borderLeft: '3px solid #f59e0b',
        ...options?.style,
      },
      icon: '⚠️',
    }),

  loading: (message: string, options?: ToastOptions) =>
    toast.loading(message, {
      ...baseOptions,
      ...options,
    }),

  dismiss: toast.dismiss,

  promise: <T,>(
    promise: Promise<T>,
    messages: {
      loading: string
      success: string | ((data: T) => string)
      error: string | ((err: Error) => string)
    },
    options?: ToastOptions
  ) =>
    toast.promise(promise, messages, {
      ...baseOptions,
      ...options,
    }),
}

export default showToast
