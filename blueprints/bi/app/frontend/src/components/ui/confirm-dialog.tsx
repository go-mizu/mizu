import { Modal, Group, Button, Text, ThemeIcon, Stack } from '@mantine/core'
import { IconAlertTriangle, IconTrash, IconCheck } from '@tabler/icons-react'

// =============================================================================
// CONFIRM DIALOG - Standardized confirmation modal
// =============================================================================

export type ConfirmDialogVariant = 'danger' | 'warning' | 'info'

export interface ConfirmDialogProps {
  /** Whether the modal is open */
  opened: boolean
  /** Close handler */
  onClose: () => void
  /** Confirm handler */
  onConfirm: () => void
  /** Dialog title */
  title: string
  /** Dialog message */
  message: string | React.ReactNode
  /** Confirm button text */
  confirmLabel?: string
  /** Cancel button text */
  cancelLabel?: string
  /** Dialog variant affects colors */
  variant?: ConfirmDialogVariant
  /** Loading state for confirm button */
  loading?: boolean
  /** Custom icon */
  icon?: React.ReactNode
}

const variantStyles = {
  danger: {
    color: 'var(--color-error)',
    buttonColor: 'red',
    icon: IconTrash,
  },
  warning: {
    color: 'var(--color-warning)',
    buttonColor: 'yellow',
    icon: IconAlertTriangle,
  },
  info: {
    color: 'var(--color-primary)',
    buttonColor: 'blue',
    icon: IconCheck,
  },
}

export function ConfirmDialog({
  opened,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'danger',
  loading = false,
  icon,
}: ConfirmDialogProps) {
  const styles = variantStyles[variant]
  const IconComponent = styles.icon

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={title}
      centered
      size="sm"
    >
      <Stack gap="md">
        <Group gap="md" wrap="nowrap">
          <ThemeIcon
            size={48}
            radius="xl"
            variant="light"
            style={{
              backgroundColor: `${styles.color}15`,
              color: styles.color,
            }}
          >
            {icon || <IconComponent size={24} strokeWidth={1.5} />}
          </ThemeIcon>
          <Text
            size="sm"
            style={{ color: 'var(--color-foreground)', flex: 1 }}
          >
            {message}
          </Text>
        </Group>

        <Group justify="flex-end" gap="sm" mt="md">
          <Button
            variant="subtle"
            color="gray"
            onClick={onClose}
            disabled={loading}
          >
            {cancelLabel}
          </Button>
          <Button
            color={styles.buttonColor}
            onClick={() => {
              onConfirm()
              // Don't auto-close if loading - let the caller handle it
              if (!loading) {
                onClose()
              }
            }}
            loading={loading}
          >
            {confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  )
}

// =============================================================================
// USE CONFIRM DIALOG - Hook for easier usage
// =============================================================================

import { useState, useCallback } from 'react'

export interface UseConfirmDialogOptions {
  /** Default confirm handler */
  onConfirm?: () => void | Promise<void>
}

export function useConfirmDialog(options: UseConfirmDialogOptions = {}) {
  const [opened, setOpened] = useState(false)
  const [loading, setLoading] = useState(false)
  const [config, setConfig] = useState<Partial<ConfirmDialogProps>>({})

  const open = useCallback((dialogConfig?: Partial<ConfirmDialogProps>) => {
    setConfig(dialogConfig || {})
    setOpened(true)
  }, [])

  const close = useCallback(() => {
    setOpened(false)
    setLoading(false)
    setConfig({})
  }, [])

  const confirm = useCallback(async () => {
    setLoading(true)
    try {
      if (config.onConfirm) {
        await config.onConfirm()
      } else if (options.onConfirm) {
        await options.onConfirm()
      }
      close()
    } catch (error) {
      setLoading(false)
      throw error
    }
  }, [config.onConfirm, options.onConfirm, close])

  const dialogProps: ConfirmDialogProps = {
    opened,
    onClose: close,
    onConfirm: confirm,
    title: config.title || 'Confirm',
    message: config.message || 'Are you sure?',
    confirmLabel: config.confirmLabel,
    cancelLabel: config.cancelLabel,
    variant: config.variant,
    loading,
    icon: config.icon,
  }

  return {
    open,
    close,
    dialogProps,
    Dialog: () => <ConfirmDialog {...dialogProps} />,
  }
}
