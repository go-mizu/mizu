import { useState, useEffect } from 'react';
import {
  Modal,
  TextInput,
  Button,
  Group,
  Stack,
  Text,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { storageApi } from '../../../api';
import type { StorageObject } from '../../../types';

interface RenameModalProps {
  opened: boolean;
  item: StorageObject | null;
  bucketId: string | null;
  onClose: () => void;
  onRename: () => void;
}

export function RenameModal({
  opened,
  item,
  bucketId,
  onClose,
  onRename,
}: RenameModalProps) {
  const [newName, setNewName] = useState('');
  const [loading, setLoading] = useState(false);

  // Reset form when item changes
  useEffect(() => {
    if (item) {
      const currentName = item.name.split('/').filter(Boolean).pop() || item.name;
      setNewName(currentName);
    }
  }, [item]);

  const handleRename = async () => {
    if (!item || !bucketId || !newName.trim()) return;

    // Validate new name
    if (newName.includes('/')) {
      notifications.show({
        title: 'Invalid name',
        message: 'File name cannot contain slashes',
        color: 'red',
      });
      return;
    }

    // Build new path
    const pathParts = item.name.split('/');
    pathParts.pop(); // Remove old filename
    const newPath = pathParts.length > 0 ? `${pathParts.join('/')}/${newName.trim()}` : newName.trim();

    // Check if trying to rename to same name
    if (item.name === newPath) {
      onClose();
      return;
    }

    setLoading(true);
    try {
      await storageApi.renameObject(bucketId, item.name, newPath);
      notifications.show({
        title: 'Success',
        message: 'File renamed successfully',
        color: 'green',
      });
      onRename();
      onClose();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to rename file',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  const isFolder = item?.name.endsWith('/') || !item?.content_type;
  const currentName = item?.name.split('/').filter(Boolean).pop() || '';

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title={`Rename ${isFolder ? 'folder' : 'file'}`}
      size="md"
    >
      <Stack gap="md">
        <Text size="sm" c="dimmed">
          Current name: <strong>{currentName}</strong>
        </Text>

        <TextInput
          label="New name"
          placeholder="Enter new name"
          value={newName}
          onChange={(e) => setNewName(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              handleRename();
            }
          }}
          autoFocus
        />

        <Group justify="flex-end" mt="md">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            onClick={handleRename}
            loading={loading}
            disabled={!newName.trim() || newName === currentName}
          >
            Rename
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}
