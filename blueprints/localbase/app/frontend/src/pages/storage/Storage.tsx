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
  Paper,
  Loader,
  Center,
  Divider,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { Dropzone } from '@mantine/dropzone';
import {
  IconPlus,
  IconFolder,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { PublicBadge } from '../../components/common/StatusBadge';
import { storageApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { Bucket, StorageObject } from '../../types';

import { MillerColumnBrowser, FilePreviewPanel, StorageToolbar } from './components';
import type { ViewMode } from './components';
import { useMillerNavigation } from './hooks/useMillerNavigation';

export function StoragePage() {
  const { selectedBucket, setSelectedBucket } = useAppStore();

  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploadLoading, setUploadLoading] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('columns');

  const [createBucketOpened, { open: openCreateBucket, close: closeCreateBucket }] =
    useDisclosure(false);
  const [deleteBucketOpened, { open: openDeleteBucket, close: closeDeleteBucket }] =
    useDisclosure(false);
  const [deleteObjectOpened, { open: openDeleteObject, close: closeDeleteObject }] =
    useDisclosure(false);
  const [createFolderOpened, { open: openCreateFolder, close: closeCreateFolder }] =
    useDisclosure(false);

  const [bucketForm, setBucketForm] = useState({ name: '', public: false });
  const [folderName, setFolderName] = useState('');
  const [formLoading, setFormLoading] = useState(false);

  const dropzoneRef = useRef<() => void>(null);

  // Miller column navigation
  const {
    columns,
    selectedFile,
    navigateToFolder,
    selectItem,
    navigateBack,
    refreshAll,
    clearSelection,
    currentPath,
  } = useMillerNavigation({
    bucketId: selectedBucket,
  });

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

  useEffect(() => {
    fetchBuckets();
  }, [fetchBuckets]);

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
      refreshAll();
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
    if (!selectedBucket || !selectedFile) return;

    setFormLoading(true);
    try {
      await storageApi.deleteObject(selectedBucket, selectedFile.name);
      notifications.show({
        title: 'Success',
        message: 'File deleted successfully',
        color: 'green',
      });
      closeDeleteObject();
      clearSelection();
      refreshAll();
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

  const handleCreateFolder = async () => {
    if (!selectedBucket || !folderName.trim()) {
      notifications.show({
        title: 'Validation Error',
        message: 'Folder name is required',
        color: 'red',
      });
      return;
    }

    setFormLoading(true);
    try {
      const path = currentPath
        ? `${currentPath}/${folderName.trim()}/.keep`
        : `${folderName.trim()}/.keep`;

      const emptyFile = new File([''], '.keep', { type: 'text/plain' });
      await storageApi.uploadObject(selectedBucket, path, emptyFile);

      notifications.show({
        title: 'Success',
        message: 'Folder created successfully',
        color: 'green',
      });
      closeCreateFolder();
      setFolderName('');
      refreshAll();
    } catch (error: any) {
      notifications.show({
        title: 'Error',
        message: error.message || 'Failed to create folder',
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

  const handleCopyUrl = async (obj: StorageObject, type: 'public' | 'signed') => {
    if (!selectedBucket) return;
    const bucket = buckets.find((b) => b.id === selectedBucket);
    let url: string;

    if (type === 'public' && bucket?.public) {
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

  const handleSearch = async (query: string) => {
    if (!selectedBucket) return;

    try {
      const results = await storageApi.listObjects(selectedBucket, {
        prefix: '',
        search: query,
        limit: 100,
      });

      // Show results in notification for now (can be enhanced later)
      notifications.show({
        title: 'Search Results',
        message: `Found ${results.length} matching files`,
        color: 'blue',
      });
    } catch (error: any) {
      notifications.show({
        title: 'Search Error',
        message: error.message || 'Failed to search files',
        color: 'red',
      });
    }
  };

  const handleNavigateToPath = (path: string) => {
    if (path === '') {
      // Reset to root - handled by refreshAll
      refreshAll();
    } else {
      // Navigate to specific path
      const parts = path.split('/');
      // Find the column index for this path
      const columnIndex = columns.findIndex(col => col.path === path);
      if (columnIndex >= 0) {
        // Already have this column, just clear selection
        clearSelection();
      } else {
        // Need to navigate to this path - find parent column
        const parentPath = parts.slice(0, -1).join('/');
        const parentIndex = columns.findIndex(col => col.path === parentPath);
        if (parentIndex >= 0) {
          navigateToFolder(path, parentIndex);
        }
      }
    }
  };

  const currentBucket = buckets.find((b) => b.id === selectedBucket);

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
            width: 240,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: 'var(--supabase-bg)',
            flexShrink: 0,
          }}
        >
          {/* MANAGE Section */}
          <Box px="md" pt="md">
            <Text
              size="xs"
              fw={600}
              tt="uppercase"
              c="dimmed"
              style={{ letterSpacing: '0.05em', fontSize: '0.6875rem' }}
            >
              Manage
            </Text>
          </Box>

          <Box p="xs">
            <Stack gap={2}>
              <Paper
                p="xs"
                style={{
                  cursor: 'pointer',
                  backgroundColor: 'var(--supabase-brand-light)',
                  borderRadius: 6,
                }}
              >
                <Group gap="xs">
                  <IconFolder size={16} color="var(--supabase-brand)" />
                  <Text size="sm" fw={500}>
                    Files
                  </Text>
                </Group>
              </Paper>
            </Stack>
          </Box>

          <Divider mx="md" my="xs" />

          {/* Buckets List */}
          <Box px="md" pt="xs">
            <Group justify="space-between" mb="xs">
              <Text size="xs" c="dimmed">
                Buckets
              </Text>
              <ActionIcon variant="subtle" onClick={openCreateBucket} size="sm">
                <IconPlus size={14} />
              </ActionIcon>
            </Group>
          </Box>

          <Box style={{ flex: 1, overflow: 'auto' }} px="xs" pb="sm">
            {loading ? (
              <Center py="xl">
                <Loader size="sm" />
              </Center>
            ) : buckets.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="xl">
                No buckets yet
              </Text>
            ) : (
              <Stack gap={2}>
                {buckets.map((bucket) => (
                  <Paper
                    key={bucket.id}
                    p="xs"
                    style={{
                      cursor: 'pointer',
                      backgroundColor:
                        selectedBucket === bucket.id
                          ? 'var(--supabase-bg-surface)'
                          : 'transparent',
                      borderRadius: 6,
                    }}
                    onClick={() => {
                      setSelectedBucket(bucket.id);
                      clearSelection();
                    }}
                  >
                    <Group justify="space-between" wrap="nowrap">
                      <Group gap="xs" wrap="nowrap">
                        <IconFolder size={16} color="var(--supabase-text-muted)" />
                        <Text size="sm" truncate style={{ maxWidth: 120 }}>
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

        {/* Main Content Area */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {selectedBucket && currentBucket ? (
            <>
              {/* Toolbar */}
              <StorageToolbar
                bucketName={currentBucket.name}
                currentPath={currentPath}
                viewMode={viewMode}
                onRefresh={refreshAll}
                onUpload={() => dropzoneRef.current?.()}
                onCreateFolder={openCreateFolder}
                onDeleteBucket={openDeleteBucket}
                onViewModeChange={setViewMode}
                onSearch={handleSearch}
                onNavigateToPath={handleNavigateToPath}
                uploadLoading={uploadLoading}
              />

              {/* File Browser */}
              <Box style={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
                <Dropzone
                  onDrop={handleUpload}
                  openRef={dropzoneRef}
                  activateOnClick={false}
                  styles={{
                    root: {
                      border: 'none',
                      backgroundColor: 'transparent',
                      flex: 1,
                      display: 'flex',
                      minWidth: 0,
                      padding: 0,
                      pointerEvents: 'auto',
                    },
                    inner: {
                      pointerEvents: 'none',
                      display: 'flex',
                      flex: 1,
                      padding: 0,
                      minWidth: 0,
                    },
                  }}
                >
                  {columns.length === 0 || (columns.length === 1 && columns[0].items.length === 0 && !columns[0].loading) ? (
                    <Center style={{ flex: 1, pointerEvents: 'auto' }}>
                      <EmptyState
                        icon={<IconFolder size={32} />}
                        title="No files"
                        description="Upload files or create a folder to get started"
                        action={{
                          label: 'Upload files',
                          onClick: () => dropzoneRef.current?.(),
                        }}
                      />
                    </Center>
                  ) : (
                    <MillerColumnBrowser
                      columns={columns}
                      onItemSelect={selectItem}
                      onBack={navigateBack}
                    />
                  )}
                </Dropzone>

                {/* File Preview Panel */}
                {selectedFile && currentBucket && (
                  <FilePreviewPanel
                    file={selectedFile}
                    bucket={currentBucket}
                    onClose={clearSelection}
                    onDownload={() => handleDownload(selectedFile)}
                    onDelete={openDeleteObject}
                    onCopyUrl={(type) => handleCopyUrl(selectedFile, type)}
                  />
                )}
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

      {/* Create Folder Modal */}
      <Modal opened={createFolderOpened} onClose={closeCreateFolder} title="Create folder" size="md">
        <Stack gap="md">
          <TextInput
            label="Folder name"
            placeholder="my-folder"
            value={folderName}
            onChange={(e) => setFolderName(e.target.value)}
            required
          />
          <Group justify="flex-end" mt="md">
            <Button variant="outline" onClick={closeCreateFolder}>
              Cancel
            </Button>
            <Button onClick={handleCreateFolder} loading={formLoading}>
              Create folder
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
        message={`Are you sure you want to delete "${selectedFile?.name.split('/').pop()}"?`}
        confirmLabel="Delete"
        danger
        loading={formLoading}
      />
    </PageContainer>
  );
}
