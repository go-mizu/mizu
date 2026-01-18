import { Modal, Text, Group, Button, Stack } from '@mantine/core';
import { IconAlertTriangle } from '@tabler/icons-react';

interface ConfirmModalProps {
  opened: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  danger?: boolean;
  loading?: boolean;
}

export function ConfirmModal({
  opened,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  danger = false,
  loading = false,
}: ConfirmModalProps) {
  return (
    <Modal opened={opened} onClose={onClose} title={title} centered size="sm">
      <Stack gap="lg">
        {danger && (
          <Group gap="sm" align="flex-start">
            <IconAlertTriangle size={20} color="var(--lb-error)" />
            <Text
              size="sm"
              style={{
                flex: 1,
                color: 'var(--lb-text-primary)',
              }}
            >
              {message}
            </Text>
          </Group>
        )}
        {!danger && (
          <Text
            size="sm"
            style={{
              color: 'var(--lb-text-primary)',
            }}
          >
            {message}
          </Text>
        )}

        <Group justify="flex-end" gap="sm">
          <Button
            variant="outline"
            onClick={onClose}
            disabled={loading}
            style={{
              borderColor: 'var(--lb-border-default)',
            }}
          >
            {cancelLabel}
          </Button>
          <Button
            color={danger ? 'red' : 'green'}
            onClick={onConfirm}
            loading={loading}
          >
            {confirmLabel}
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
