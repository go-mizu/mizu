import { useEffect, useState, useCallback, useRef } from 'react';
import {
  Box,
  Button,
  Modal,
  TextInput,
  Switch,
  Stack,
  Group,
  Text,
  ActionIcon,
  Menu,
  Breadcrumbs,
  Anchor,
  Paper,
  Loader,
  Center,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Dropzone } from '@mantine/dropzone';
import {
  IconPlus,
  IconDotsVertical,
  IconTrash,
  IconFolder,
  IconFile,
  IconUpload,
  IconDownload,
  IconLink,
  IconArrowLeft,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { PublicBadge } from '../../components/common/StatusBadge';
import { storageApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { Bucket, StorageObject } from '../../types';

export function StoragePage() {
  const { selectedBucket, setSelectedBucket, currentPath, setCurrentPath } = useAppStore();

  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [objects, setObjects] = useState<StorageObject[]>([]);
  const [loading, setLoading] = useState(true);
  const [objectsLoading, setObjectsLoading] = useState(false);
  const [uploadLoading, setUploadLoading] = useState(false);

  const [createBucketOpened, { open: openCreateBucket, close: closeCreateBucket }] =
    useDisclosure(false);
  const [deleteBucketOpened, { open: openDeleteBucket, close: closeDeleteBucket }] =
    useDisclosure(false);
  const [deleteObjectOpened, { open: openDeleteObject, close: closeDeleteObject }] =
    useDisclosure(false);

  const [bucketForm, setBucketForm] = useState({ name: '', public: false });
  const [selectedObject, setSelectedObject] = useState<StorageObject | null>(null);
  const [formLoading, setFormLoading] = useState(false);

  const dropzoneRef = useRef<() => void>(null);

  const fetchBuckets = useCallback(async () => {
    setLoading(true);
    try {
      const data = await storageApi.listBuckets();
      setBuckets(data);
      if (data.length > 0 && !selectedBucket) {
        setSelectedBucket(data[0].id);
      }
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load buckets',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [selectedBucket, setSelectedBucket]);

  const fetchObjects = useCallback(async () => {
    if (!selectedBucket) return;

    setObjectsLoading(true);
    try {
      const data = await storageApi.listObjects(selectedBucket, {
        prefix: currentPath,
        limit: 100,
      });
      setObjects(data);
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to load objects',
        color: 'red',
      });
      setObjects([]);
    } finally {
      setObjectsLoading(false);
    }
  }, [selectedBucket, currentPath]);

  useEffect(() => {
    fetchBuckets();
  }, [fetchBuckets]);

  useEffect(() => {
    if (selectedBucket) {
      fetchObjects();
    }
  }, [selectedBucket, currentPath, fetchObjects]);

  const handleCreateBucket = async () => {
    if (!bucketForm.name) {
      notifications.show({
        title: 'Validation Error',
        message: 'Bucket name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      await storageApi.createBucket({
        name: bucketForm.name,
        public: bucketForm.public,
      });
      notifications.show({
        title: 'Success',
        message: 'Bucket created successfully',
        color: 'green',
      });
      closeCreateBucket();
      setBucketForm({ name: '', public: false });
      fetchBuckets();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create bucket',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDeleteBucket = async () => {
    if (!selectedBucket) return;

    setFormLoading(true);
    try {
      await storageApi.deleteBucket(selectedBucket);
      notifications.show({
        title: 'Success',
        message: 'Bucket deleted successfully',
        color: 'green',
      });
      closeDeleteBucket();
      setSelectedBucket(null);
      fetchBuckets();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete bucket',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleUpload = async (files: File[]) => {
    if (!selectedBucket || files.length === 0) return;

    setUploadLoading(true);
    try {
      for (const file of files) {
        const path = currentPath ? `${currentPath}/${file.name}` : file.name;
        await storageApi.uploadObject(selectedBucket, path, file);
      }
      notifications.show({
        title: 'Success',
        message: `${files.length} file(s) uploaded successfully`,
        color: 'green',
      });
      fetchObjects();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to upload files',
        color: 'red',
      });
    } finally {
      setUploadLoading(false);
    }
  };

  const handleDeleteObject = async () => {
    if (!selectedBucket || !selectedObject) return;

    setFormLoading(true);
    try {
      await storageApi.deleteObject(selectedBucket, selectedObject.name);
      notifications.show({
        title: 'Success',
        message: 'File deleted successfully',
        color: 'green',
      });
      closeDeleteObject();
      setSelectedObject(null);
      fetchObjects();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to delete file',
        color: 'red',
      });
    } finally {
      setFormLoading(false);
    }
  };

  const handleDownload = (obj: StorageObject) => {
    if (!selectedBucket) return;
    const url = storageApi.downloadObjectUrl(selectedBucket, obj.name);
    window.open(url, '_blank');
  };

  const handleCopyUrl = async (obj: StorageObject) => {
    if (!selectedBucket) return;
    const bucket = buckets.find((b) => b.id === selectedBucket);
    let url: string;

    if (bucket?.public) {
      url = `${window.location.origin}${storageApi.getPublicUrl(selectedBucket, obj.name)}`;
    } else {
      try {
        const result = await storageApi.createSignedUrl(selectedBucket, obj.name, 3600);
        url = result.signedURL;
      } catch {
        url = storageApi.downloadObjectUrl(selectedBucket, obj.name);
      }
    }

    navigator.clipboard.writeText(url);
    notifications.show({
      title: 'Copied',
      message: 'URL copied to clipboard',
      color: 'green',
    });
  };

  const navigateToFolder = (folderPath: string) => {
    setCurrentPath(folderPath);
  };

  const navigateUp = () => {
    if (!currentPath) return;
    const parts = currentPath.split('/');
    parts.pop();
    setCurrentPath(parts.join('/'));
  };

  const breadcrumbParts = currentPath ? currentPath.split('/') : [];

  const currentBucket = buckets.find((b) => b.id === selectedBucket);

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  const isFolder = (obj: StorageObject) => {
    return obj.name.endsWith('/') || !obj.content_type;
  };

  return (
    <PageContainer
      title="Storage"
      description="Manage files and buckets"
      fullWidth
      noPadding
    >
      <Box style={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
        {/* Bucket Sidebar */}
        <Box
          style={{
            width: 280,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Box p="md" pb="sm">
            <Group justify="space-between" mb="sm">
              <Text fw={600} size="sm">
                Buckets
              </Text>
              <ActionIcon variant="subtle" onClick={openCreateBucket} size="sm">
                <IconPlus size={16} />
              </ActionIcon>
            </Group>
          </Box>

          <Box style={{ flex: 1, overflow: 'auto' }} px="sm" pb="sm">
            {loading ? (
              <Center py="xl">
                <Loader size="sm" />
              </Center>
            ) : buckets.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="xl">
                No buckets yet
              </Text>
            ) : (
              <Stack gap={4}>
                {buckets.map((bucket) => (
                  <Paper
                    key={bucket.id}
                    p="xs"
                    style={{
                      cursor: 'pointer',
                      backgroundColor:
                        selectedBucket === bucket.id
                          ? 'var(--supabase-brand-light)'
                          : 'transparent',
                      borderRadius: 6,
                    }}
                    onClick={() => setSelectedBucket(bucket.id)}
                  >
                    <Group justify="space-between" wrap="nowrap">
                      <Group gap="xs" wrap="nowrap">
                        <IconFolder size={16} />
                        <Text size="sm" truncate style={{ maxWidth: 150 }}>
                          {bucket.name}
                        </Text>
                      </Group>
                      <PublicBadge isPublic={bucket.public} />
                    </Group>
                  </Paper>
                ))}
              </Stack>
            )}
          </Box>
        </Box>

        {/* File Browser */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {selectedBucket ? (
            <>
              {/* Toolbar */}
              <Box
                p="md"
                style={{ borderBottom: '1px solid var(--supabase-border)' }}
              >
                <Group justify="space-between">
                  <Group gap="xs">
                    {currentPath && (
                      <ActionIcon variant="subtle" onClick={navigateUp}>
                        <IconArrowLeft size={18} />
                      </ActionIcon>
                    )}
                    <Breadcrumbs>
                      <Anchor
                        size="sm"
                        onClick={() => setCurrentPath('')}
                        style={{ cursor: 'pointer' }}
                      >
                        {currentBucket?.name || selectedBucket}
                      </Anchor>
                      {breadcrumbParts.map((part, index) => (
                        <Anchor
                          key={index}
                          size="sm"
                          onClick={() =>
                            navigateToFolder(breadcrumbParts.slice(0, index + 1).join('/'))
                          }
                          style={{ cursor: 'pointer' }}
                        >
                          {part}
                        </Anchor>
                      ))}
                    </Breadcrumbs>
                  </Group>

                  <Group gap="sm">
                    <Button
                      leftSection={<IconUpload size={16} />}
                      onClick={() => dropzoneRef.current?.()}
                      loading={uploadLoading}
                      size="sm"
                    >
                      Upload
                    </Button>
                    <Menu position="bottom-end">
                      <Menu.Target>
                        <ActionIcon variant="subtle">
                          <IconDotsVertical size={18} />
                        </ActionIcon>
                      </Menu.Target>
                      <Menu.Dropdown>
                        <Menu.Item
                          color="red"
                          leftSection={<IconTrash size={14} />}
                          onClick={openDeleteBucket}
                        >
                          Delete bucket
                        </Menu.Item>
                      </Menu.Dropdown>
                    </Menu>
                  </Group>
                </Group>
              </Box>

              {/* File List */}
              <Box style={{ flex: 1, overflow: 'auto' }} p="md">
                <Dropzone
                  onDrop={handleUpload}
                  openRef={dropzoneRef}
                  activateOnClick={false}
                  styles={{
                    root: {
                      border: 'none',
                      backgroundColor: 'transparent',
                      minHeight: '100%',
                    },
                  }}
                >
                  {objectsLoading ? (
                    <Center py="xl">
                      <Loader size="sm" />
                    </Center>
                  ) : objects.length === 0 ? (
                    <EmptyState
                      icon={<IconFolder size={32} />}
                      title="No files"
                      description="Upload files or create a folder to get started"
                      action={{
                        label: 'Upload files',
                        onClick: () => dropzoneRef.current?.(),
                      }}
                    />
                  ) : (
                    <Stack gap={4}>
                      {objects.map((obj) => (
                        <Paper
                          key={obj.id}
                          p="sm"
                          style={{
                            cursor: isFolder(obj) ? 'pointer' : 'default',
                            border: '1px solid var(--supabase-border)',
                            borderRadius: 6,
                          }}
                          onClick={() => {
                            if (isFolder(obj)) {
                              navigateToFolder(obj.name.replace(/\/$/, ''));
                            }
                          }}
                        >
                          <Group justify="space-between" wrap="nowrap">
                            <Group gap="sm" wrap="nowrap">
                              {isFolder(obj) ? (
                                <IconFolder size={20} color="var(--supabase-brand)" />
                              ) : (
                                <IconFile size={20} />
                              )}
                              <Box>
                                <Text size="sm" fw={500} truncate style={{ maxWidth: 400 }}>
                                  {obj.name.split('/').pop() || obj.name}
                                </Text>
                                {!isFolder(obj) && (
                                  <Text size="xs" c="dimmed">
                                    {formatFileSize(obj.size)} • {obj.content_type || 'Unknown'}
                                  </Text>
                                )}
                              </Box>
                            </Group>

                            {!isFolder(obj) && (
                              <Group gap="xs">
                                <Tooltip label="Download">
                                  <ActionIcon
                                    variant="subtle"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      handleDownload(obj);
                                    }}
                                  >
                                    <IconDownload size={16} />
                                  </ActionIcon>
                                </Tooltip>
                                <Tooltip label="Copy URL">
                                  <ActionIcon
                                    variant="subtle"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      handleCopyUrl(obj);
                                    }}
                                  >
                                    <IconLink size={16} />
                                  </ActionIcon>
                                </Tooltip>
                                <Menu position="bottom-end">
                                  <Menu.Target>
                                    <ActionIcon
                                      variant="subtle"
                                      onClick={(e) => e.stopPropagation()}
                                    >
                                      <IconDotsVertical size={16} />
                                    </ActionIcon>
                                  </Menu.Target>
                                  <Menu.Dropdown>
                                    <Menu.Item
                                      color="red"
                                      leftSection={<IconTrash size={14} />}
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        setSelectedObject(obj);
                                        openDeleteObject();
                                      }}
                                    >
                                      Delete
                                    </Menu.Item>
                                  </Menu.Dropdown>
                                </Menu>
                              </Group>
                            )}
                          </Group>
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </Dropzone>
              </Box>
            </>
          ) : (
            <Center style={{ flex: 1 }}>
              <EmptyState
                icon={<IconFolder size={32} />}
                title="Select a bucket"
                description="Choose a bucket from the sidebar or create a new one"
                action={{
                  label: 'Create bucket',
                  onClick: openCreateBucket,
                }}
              />
            </Center>
          )}
        </Box>
      </Box>

      {/* Create Bucket Modal */}
      <Modal opened={createBucketOpened} onClose={closeCreateBucket} title="Create bucket" size="md">
        <Stack gap="md">
          <TextInput
            label="Name"
            placeholder="my-bucket"
            value={bucketForm.name}
            onChange={(e) => setBucketForm({ ...bucketForm, name: e.target.value })}
            required
          />
          <Switch
            label="Public bucket"
            description="Allow public access to files without authentication"
            checked={bucketForm.public}
            onChange={(e) => setBucketForm({ ...bucketForm, public: e.target.checked })}
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateBucket}>
              Cancel
            </Button>
            <Button onClick={handleCreateBucket} loading={formLoading}>
              Create bucket
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Delete Bucket Confirmation */}
      <ConfirmModal
        opened={deleteBucketOpened}
        onClose={closeDeleteBucket}
        onConfirm={handleDeleteBucket}
        title="Delete bucket"
        message={`Are you sure you want to delete "${currentBucket?.name}"? All files in this bucket will be permanently deleted.`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />

      {/* Delete Object Confirmation */}
      <ConfirmModal
        opened={deleteObjectOpened}
        onClose={closeDeleteObject}
        onConfirm={handleDeleteObject}
        title="Delete file"
        message={`Are you sure you want to delete "${selectedObject?.name.split('/').pop()}"?`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
