import { useState, useEffect, useRef, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { AlertTriangle, Trash2, X, Info, AlertCircle } from 'lucide-react'

type DialogVariant = 'danger' | 'warning' | 'info'

interface ConfirmDialogProps {
  isOpen: boolean
  onClose: () => void
  onConfirm: () => void | Promise<void>
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: DialogVariant
  isLoading?: boolean
}

const variantConfig = {
  danger: {
    icon: Trash2,
    iconBg: 'var(--danger-bg)',
    iconColor: 'var(--danger-color)',
    buttonClass: 'btn-danger',
  },
  warning: {
    icon: AlertTriangle,
    iconBg: '#fef3c7',
    iconColor: '#d97706',
    buttonClass: 'btn-primary',
  },
  info: {
    icon: Info,
    iconBg: 'var(--accent-bg)',
    iconColor: 'var(--accent-color)',
    buttonClass: 'btn-primary',
  },
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'danger',
  isLoading = false,
}: ConfirmDialogProps) {
  const [internalLoading, setInternalLoading] = useState(false)
  const cancelRef = useRef<HTMLButtonElement>(null)
  const config = variantConfig[variant]
  const IconComponent = config.icon

  // Focus cancel button on open
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => cancelRef.current?.focus(), 50)
    }
  }, [isOpen])

  // Close on Escape
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isOpen && !isLoading && !internalLoading) {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, isLoading, internalLoading, onClose])

  const handleConfirm = useCallback(async () => {
    try {
      setInternalLoading(true)
      await onConfirm()
      onClose()
    } catch (err) {
      console.error('Confirm action failed:', err)
    } finally {
      setInternalLoading(false)
    }
  }, [onConfirm, onClose])

  if (!isOpen) return null

  const loading = isLoading || internalLoading

  return createPortal(
    <div className="modal-backdrop" onClick={loading ? undefined : onClose}>
      <div
        className="modal-content confirm-dialog"
        onClick={(e) => e.stopPropagation()}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-title"
        aria-describedby="confirm-message"
      >
        <div className="modal-body">
          <div
            className="confirm-icon"
            style={{
              background: config.iconBg,
              color: config.iconColor,
            }}
          >
            <IconComponent size={24} />
          </div>
          <h3 id="confirm-title" className="confirm-title">
            {title}
          </h3>
          <p id="confirm-message" className="confirm-message">
            {message}
          </p>
        </div>
        <div className="modal-footer" style={{ justifyContent: 'center' }}>
          <button
            ref={cancelRef}
            className="btn btn-secondary"
            onClick={onClose}
            disabled={loading}
          >
            {cancelText}
          </button>
          <button
            className={`btn ${config.buttonClass} ${loading ? 'loading' : ''}`}
            onClick={handleConfirm}
            disabled={loading}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>,
    document.body
  )
}

// Hook for easier usage
interface UseConfirmOptions {
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: DialogVariant
}

interface ConfirmState {
  isOpen: boolean
  options: UseConfirmOptions | null
  resolve: ((value: boolean) => void) | null
}

export function useConfirm() {
  const [state, setState] = useState<ConfirmState>({
    isOpen: false,
    options: null,
    resolve: null,
  })

  const confirm = useCallback((options: UseConfirmOptions): Promise<boolean> => {
    return new Promise((resolve) => {
      setState({
        isOpen: true,
        options,
        resolve,
      })
    })
  }, [])

  const handleClose = useCallback(() => {
    state.resolve?.(false)
    setState({ isOpen: false, options: null, resolve: null })
  }, [state.resolve])

  const handleConfirm = useCallback(() => {
    state.resolve?.(true)
    setState({ isOpen: false, options: null, resolve: null })
  }, [state.resolve])

  const dialogProps = state.options
    ? {
        isOpen: state.isOpen,
        onClose: handleClose,
        onConfirm: handleConfirm,
        title: state.options.title,
        message: state.options.message,
        confirmText: state.options.confirmText,
        cancelText: state.options.cancelText,
        variant: state.options.variant,
      }
    : null

  return { confirm, dialogProps, ConfirmDialog }
}

// Alert dialog (single button)
interface AlertDialogProps {
  isOpen: boolean
  onClose: () => void
  title: string
  message: string
  buttonText?: string
  variant?: DialogVariant
}

export function AlertDialog({
  isOpen,
  onClose,
  title,
  message,
  buttonText = 'OK',
  variant = 'info',
}: AlertDialogProps) {
  const buttonRef = useRef<HTMLButtonElement>(null)
  const config = variantConfig[variant]
  const IconComponent = config.icon

  useEffect(() => {
    if (isOpen) {
      setTimeout(() => buttonRef.current?.focus(), 50)
    }
  }, [isOpen])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.key === 'Escape' || e.key === 'Enter') && isOpen) {
        onClose()
      }
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return createPortal(
    <div className="modal-backdrop" onClick={onClose}>
      <div
        className="modal-content confirm-dialog"
        onClick={(e) => e.stopPropagation()}
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="alert-title"
        aria-describedby="alert-message"
      >
        <div className="modal-body">
          <div
            className="confirm-icon"
            style={{
              background: config.iconBg,
              color: config.iconColor,
            }}
          >
            <IconComponent size={24} />
          </div>
          <h3 id="alert-title" className="confirm-title">
            {title}
          </h3>
          <p id="alert-message" className="confirm-message">
            {message}
          </p>
        </div>
        <div className="modal-footer" style={{ justifyContent: 'center' }}>
          <button
            ref={buttonRef}
            className={`btn ${config.buttonClass}`}
            onClick={onClose}
          >
            {buttonText}
          </button>
        </div>
      </div>
    </div>,
    document.body
  )
}

export function useAlert() {
  const [state, setState] = useState<{
    isOpen: boolean
    options: Omit<AlertDialogProps, 'isOpen' | 'onClose'> | null
  }>({
    isOpen: false,
    options: null,
  })

  const alert = useCallback(
    (options: Omit<AlertDialogProps, 'isOpen' | 'onClose'>): void => {
      setState({
        isOpen: true,
        options,
      })
    },
    []
  )

  const handleClose = useCallback(() => {
    setState({ isOpen: false, options: null })
  }, [])

  const dialogProps = state.options
    ? {
        isOpen: state.isOpen,
        onClose: handleClose,
        ...state.options,
      }
    : null

  return { alert, dialogProps, AlertDialog }
}
