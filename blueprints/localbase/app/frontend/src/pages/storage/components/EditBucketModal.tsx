import { useState, useEffect } from 'react';
import {
  Modal,
  Stack,
  TextInput,
  Switch,
  NumberInput,
  MultiSelect,
  Button,
  Group,
  Text,
  Divider,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { storageApi } from '../../../api';
import type { Bucket } from '../../../types';

const COMMON_MIME_TYPES = [
  { value: 'image/*', label: 'All images (image/*)' },
  { value: 'video/*', label: 'All videos (video/*)' },
  { value: 'audio/*', label: 'All audio (audio/*)' },
  { value: 'application/pdf', label: 'PDF documents' },
  { value: 'text/*', label: 'All text files (text/*)' },
  { value: 'application/json', label: 'JSON files' },
  { value: 'application/zip', label: 'ZIP archives' },
  { value: '*/*', label: 'All file types (*/*) ' },
];

const SIZE_OPTIONS = [
  { value: 1024 * 1024, label: '1 MB' },
  { value: 5 * 1024 * 1024, label: '5 MB' },
  { value: 10 * 1024 * 1024, label: '10 MB' },
  { value: 50 * 1024 * 1024, label: '50 MB' },
  { value: 100 * 1024 * 1024, label: '100 MB' },
  { value: 500 * 1024 * 1024, label: '500 MB' },
];

interface EditBucketModalProps {
  bucket: Bucket | null;
  opened: boolean;
  onClose: () => void;
  onSave: () => void;
}

export function EditBucketModal({
  bucket,
  opened,
  onClose,
  onSave,
}: EditBucketModalProps) {
  const [isPublic, setIsPublic] = useState(false);
  const [fileSizeLimit, setFileSizeLimit] = useState<number | null>(null);
  const [noSizeLimit, setNoSizeLimit] = useState(true);
  const [allowedMimeTypes, setAllowedMimeTypes] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  // Reset form when bucket changes
  useEffect(() => {
    if (bucket) {
      setIsPublic(bucket.public);
      setFileSizeLimit(bucket.file_size_limit || null);
      setNoSizeLimit(!bucket.file_size_limit);
      setAllowedMimeTypes(bucket.allowed_mime_types || []);
    }
  }, [bucket]);

  const handleSave = async () => {
    if (!bucket) return;

    setLoading(true);
    try {
      await storageApi.updateBucket(bucket.id, {
        public: isPublic,
        file_size_limit: noSizeLimit ? undefined : (fileSizeLimit || undefined),
        allowed_mime_types: allowedMimeTypes.length > 0 ? allowedMimeTypes : undefined,
      });
      notifications.show({
        title: 'Success',
        message: 'Bucket settings updated',
        color: 'green',
      });
      onSave();
      onClose();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to update bucket',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  };

  if (!bucket) return null;

  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Edit Bucket Settings"
      size="md"
    >
      <Stack gap="md">
        <TextInput
          label="Bucket name"
          value={bucket.name}
          disabled
          description="Bucket name cannot be changed after creation"
        />

        <Divider my="xs" />

        <Switch
          label="Public bucket"
          description="Allow public access to files without authentication"
          checked={isPublic}
          onChange={(e) => setIsPublic(e.currentTarget.checked)}
        />

        <Divider my="xs" />

        <Stack gap="xs">
          <Text size="sm" fw={500}>File size limit</Text>
          <Switch
            label="No limit"
            checked={noSizeLimit}
            onChange={(e) => setNoSizeLimit(e.currentTarget.checked)}
          />
          {!noSizeLimit && (
            <NumberInput
              placeholder="Max file size in bytes"
              value={fileSizeLimit || ''}
              onChange={(val) => setFileSizeLimit(val as number)}
              min={1024}
              step={1024 * 1024}
              suffix=" bytes"
              description={
                fileSizeLimit
                  ? `Approximately ${formatBytes(fileSizeLimit)}`
                  : 'Enter maximum file size'
              }
            />
          )}
          {!noSizeLimit && (
            <Group gap="xs">
              {SIZE_OPTIONS.map((opt) => (
                <Button
                  key={opt.value}
                  size="xs"
                  variant={fileSizeLimit === opt.value ? 'filled' : 'outline'}
                  onClick={() => setFileSizeLimit(opt.value)}
                >
                  {opt.label}
                </Button>
              ))}
            </Group>
          )}
        </Stack>

        <Divider my="xs" />

        <MultiSelect
          label="Allowed MIME types"
          description="Leave empty to allow all file types"
          placeholder="Select or type MIME types"
          data={COMMON_MIME_TYPES}
          value={allowedMimeTypes}
          onChange={setAllowedMimeTypes}
          searchable
          clearable
        />

        <Divider my="md" />

        <Group justify="flex-end">
          <Button variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={handleSave} loading={loading}>
            Save changes
          </Button>
        </Group>
      </Stack>
    </Modal>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}
