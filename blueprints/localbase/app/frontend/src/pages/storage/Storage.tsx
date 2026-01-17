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
  CloseButton,
  Divider,
  Image,
  Code,
  ScrollArea,
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
  IconPhoto,
  IconVideo,
  IconMusic,
  IconFileTypePdf,
  IconBrandJavascript,
  IconBrandPython,
  IconBrandHtml5,
  IconBrandCss3,
  IconFileZip,
  IconMarkdown,
  IconBraces,
  IconFileCode,
  IconFileText,
  IconRefresh,
  IconEye,
  IconFolderPlus,
  IconChevronDown,
  IconExternalLink,
} from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { EmptyState } from '../../components/common/EmptyState';
import { ConfirmModal } from '../../components/common/ConfirmModal';
import { PublicBadge } from '../../components/common/StatusBadge';
import { storageApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { Bucket, StorageObject } from '../../types';

// File type detection utilities
function getFileIcon(file: StorageObject) {
  const contentType = file.content_type || '';
  const ext = file.name.split('.').pop()?.toLowerCase();

  // By content type
  if (contentType.startsWith('image/')) return IconPhoto;
  if (contentType.startsWith('video/')) return IconVideo;
  if (contentType.startsWith('audio/')) return IconMusic;
  if (contentType === 'application/pdf') return IconFileTypePdf;
  if (contentType === 'application/json') return IconBraces;

  // By extension
  switch (ext) {
    case 'js':
    case 'jsx':
    case 'ts':
    case 'tsx':
      return IconBrandJavascript;
    case 'py':
      return IconBrandPython;
    case 'go':
    case 'rs':
    case 'java':
    case 'c':
    case 'cpp':
    case 'h':
      return IconFileCode;
    case 'md':
    case 'mdx':
      return IconMarkdown;
    case 'json':
      return IconBraces;
    case 'html':
    case 'htm':
      return IconBrandHtml5;
    case 'css':
    case 'scss':
    case 'less':
      return IconBrandCss3;
    case 'zip':
    case 'tar':
    case 'gz':
    case 'rar':
    case '7z':
      return IconFileZip;
    case 'txt':
    case 'log':
      return IconFileText;
    case 'pdf':
      return IconFileTypePdf;
    default:
      return IconFile;
  }
}

function isPreviewableImage(contentType: string | undefined): boolean {
  if (!contentType) return false;
  return contentType.startsWith('image/');
}

function isPreviewableVideo(contentType: string | undefined): boolean {
  if (!contentType) return false;
  return contentType.startsWith('video/');
}

function isPreviewableAudio(contentType: string | undefined): boolean {
  if (!contentType) return false;
  return contentType.startsWith('audio/');
}

function isPreviewableText(contentType: string | undefined, name: string): boolean {
  if (!contentType && !name) return false;

  const textTypes = [
    'text/plain',
    'text/html',
    'text/css',
    'text/javascript',
    'application/json',
    'application/javascript',
    'application/xml',
  ];

  if (contentType && textTypes.some(t => contentType.includes(t))) return true;

  const ext = name.split('.').pop()?.toLowerCase();
  const textExtensions = [
    'txt', 'md', 'mdx', 'json', 'js', 'jsx', 'ts', 'tsx', 'py', 'go', 'rs',
    'java', 'c', 'cpp', 'h', 'hpp', 'css', 'scss', 'less', 'html', 'htm',
    'xml', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'sh', 'bash', 'zsh',
    'sql', 'graphql', 'vue', 'svelte', 'astro', 'rb', 'php', 'swift', 'kt',
    'gradle', 'env', 'gitignore', 'dockerfile', 'makefile', 'log',
  ];

  return ext ? textExtensions.includes(ext) : false;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString('en-US', {
    month: 'numeric',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  });
}

// File Preview Panel Component
interface FilePreviewPanelProps {
  file: StorageObject;
  bucket: Bucket;
  onClose: () => void;
  onDownload: () => void;
  onDelete: () => void;
  onCopyUrl: () => void;
}

function FilePreviewPanel({ file, bucket, onClose, onDownload, onDelete, onCopyUrl }: FilePreviewPanelProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [loadingPreview, setLoadingPreview] = useState(false);

  const FileIcon = getFileIcon(file);
  const fileName = file.name.split('/').pop() || file.name;
  const contentType = file.content_type || 'Unknown';

  useEffect(() => {
    // Generate preview URL for the file
    if (bucket) {
      const url = storageApi.downloadObjectUrl(bucket.id, file.name);
      setPreviewUrl(url);

      // Fetch text content for text files
      if (isPreviewableText(file.content_type, file.name)) {
        setLoadingPreview(true);
        fetch(url)
          .then((res) => res.text())
          .then((text) => {
            setTextContent(text.slice(0, 10000)); // Limit preview size
          })
          .catch(() => setTextContent(null))
          .finally(() => setLoadingPreview(false));
      }
    }
  }, [file, bucket]);

  const renderPreview = () => {
    if (isPreviewableImage(file.content_type) && previewUrl) {
      return (
        <Box
          style={{
            backgroundColor: 'var(--supabase-bg-surface)',
            borderRadius: 'var(--supabase-radius-lg)',
            padding: 16,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            minHeight: 200,
          }}
        >
          <Image
            src={previewUrl}
            alt={fileName}
            fit="contain"
            style={{ maxHeight: 300 }}
            radius="md"
          />
        </Box>
      );
    }

    if (isPreviewableVideo(file.content_type) && previewUrl) {
      return (
        <Box
          style={{
            backgroundColor: 'var(--supabase-bg-surface)',
            borderRadius: 'var(--supabase-radius-lg)',
            overflow: 'hidden',
          }}
        >
          <video
            src={previewUrl}
            controls
            style={{ width: '100%', maxHeight: 300 }}
          />
        </Box>
      );
    }

    if (isPreviewableAudio(file.content_type) && previewUrl) {
      return (
        <Box
          style={{
            backgroundColor: 'var(--supabase-bg-surface)',
            borderRadius: 'var(--supabase-radius-lg)',
            padding: 24,
          }}
        >
          <Center mb="md">
            <IconMusic size={48} color="var(--supabase-text-muted)" />
          </Center>
          <audio src={previewUrl} controls style={{ width: '100%' }} />
        </Box>
      );
    }

    if (isPreviewableText(file.content_type, file.name)) {
      if (loadingPreview) {
        return (
          <Center py="xl">
            <Loader size="sm" />
          </Center>
        );
      }

      if (textContent) {
        return (
          <ScrollArea h={300}>
            <Code
              block
              style={{
                backgroundColor: 'var(--supabase-bg-surface)',
                fontSize: '0.75rem',
                lineHeight: 1.5,
              }}
            >
              {textContent}
            </Code>
          </ScrollArea>
        );
      }
    }

    // Default: Show file icon
    return (
      <Box
        style={{
          backgroundColor: 'var(--supabase-bg-surface)',
          borderRadius: 'var(--supabase-radius-lg)',
          padding: 48,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
        }}
      >
        <FileIcon size={64} color="var(--supabase-text-muted)" stroke={1} />
        <Text size="sm" c="dimmed" mt="md">
          Preview not available
        </Text>
      </Box>
    );
  };

  return (
    <Box
      style={{
        width: 320,
        borderLeft: '1px solid var(--supabase-border)',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: 'var(--supabase-bg)',
      }}
    >
      {/* Header */}
      <Box
        p="md"
        style={{ borderBottom: '1px solid var(--supabase-border)' }}
      >
        <Group justify="flex-end">
          <CloseButton onClick={onClose} />
        </Group>
      </Box>

      {/* Preview Area */}
      <Box p="md" style={{ flex: 1, overflow: 'auto' }}>
        {renderPreview()}

        {/* File Info */}
        <Box mt="lg">
          <Text fw={600} size="sm" style={{ wordBreak: 'break-word' }}>
            {fileName}
          </Text>
          <Text size="xs" c="dimmed" mt={4}>
            {contentType} - {formatFileSize(file.size)}
          </Text>
        </Box>

        {/* Metadata */}
        <Stack gap="sm" mt="lg">
          <Box>
            <Text size="xs" c="dimmed">
              Added on
            </Text>
            <Text size="sm">
              {file.created_at ? formatDate(file.created_at) : 'Unknown'}
            </Text>
          </Box>

          <Box>
            <Text size="xs" c="dimmed">
              Last modified
            </Text>
            <Text size="sm">
              {file.updated_at ? formatDate(file.updated_at) : 'Unknown'}
            </Text>
          </Box>

          {file.owner && (
            <Box>
              <Text size="xs" c="dimmed">
                Owner
              </Text>
              <Text size="sm" truncate>
                {file.owner}
              </Text>
            </Box>
          )}
        </Stack>

        <Divider my="lg" />

        {/* Actions */}
        <Stack gap="xs">
          <Button
            variant="outline"
            leftSection={<IconDownload size={16} />}
            onClick={onDownload}
            fullWidth
          >
            Download
          </Button>

          <Menu position="bottom-end" width={200}>
            <Menu.Target>
              <Button
                variant="outline"
                leftSection={<IconLink size={16} />}
                rightSection={<IconChevronDown size={14} />}
                fullWidth
              >
                Get URL
              </Button>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Item
                leftSection={<IconLink size={14} />}
                onClick={onCopyUrl}
              >
                Copy URL
              </Menu.Item>
              {bucket.public && (
                <Menu.Item
                  leftSection={<IconExternalLink size={14} />}
                  onClick={() => {
                    const url = `${window.location.origin}${storageApi.getPublicUrl(bucket.id, file.name)}`;
                    window.open(url, '_blank');
                  }}
                >
                  Open in new tab
                </Menu.Item>
              )}
            </Menu.Dropdown>
          </Menu>

          <Divider my="xs" />

          <Button
            variant="subtle"
            color="red"
            leftSection={<IconTrash size={16} />}
            onClick={onDelete}
            fullWidth
          >
            Delete file
          </Button>
        </Stack>
      </Box>
    </Box>
  );
}

export function StoragePage() {
  const { selectedBucket, setSelectedBucket, currentPath, setCurrentPath } = useAppStore();

  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [objects, setObjects] = useState<StorageObject[]>([]);
  const [loading, setLoading] = useState(true);
  const [objectsLoading, setObjectsLoading] = useState(false);
  const [uploadLoading, setUploadLoading] = useState(false);
  const [selectedFile, setSelectedFile] = useState<StorageObject | null>(null);

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
      setSelectedFile(null);
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
      setSelectedFile(null);
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

      // Create a placeholder file to create the folder
      const emptyFile = new File([''], '.keep', { type: 'text/plain' });
      await storageApi.uploadObject(selectedBucket, path, emptyFile);

      notifications.show({
        title: 'Success',
        message: 'Folder created successfully',
        color: 'green',
      });
      closeCreateFolder();
      setFolderName('');
      fetchObjects();
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
    setSelectedFile(null);
  };

  const navigateUp = () => {
    if (!currentPath) return;
    const parts = currentPath.split('/');
    parts.pop();
    setCurrentPath(parts.join('/'));
    setSelectedFile(null);
  };

  const breadcrumbParts = currentPath ? currentPath.split('/') : [];

  const currentBucket = buckets.find((b) => b.id === selectedBucket);

  const isFolder = (obj: StorageObject) => {
    return obj.name.endsWith('/') || !obj.content_type;
  };

  const handleFileClick = (obj: StorageObject) => {
    if (isFolder(obj)) {
      navigateToFolder(obj.name.replace(/\/$/, ''));
    } else {
      setSelectedFile(obj);
    }
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
            width: 240,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: 'var(--supabase-bg)',
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
                      setCurrentPath('');
                      setSelectedFile(null);
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
                        onClick={() => {
                          setCurrentPath('');
                          setSelectedFile(null);
                        }}
                        style={{ cursor: 'pointer' }}
                      >
                        {currentBucket?.name || selectedBucket}
                      </Anchor>
                      {breadcrumbParts.map((part, index) => (
                        <Anchor
                          key={index}
                          size="sm"
                          onClick={() => {
                            navigateToFolder(breadcrumbParts.slice(0, index + 1).join('/'));
                          }}
                          style={{ cursor: 'pointer' }}
                        >
                          {part}
                        </Anchor>
                      ))}
                    </Breadcrumbs>
                  </Group>

                  <Group gap="sm">
                    <Tooltip label="Reload">
                      <ActionIcon variant="subtle" onClick={fetchObjects}>
                        <IconRefresh size={18} />
                      </ActionIcon>
                    </Tooltip>
                    <Tooltip label="View">
                      <ActionIcon variant="subtle">
                        <IconEye size={18} />
                      </ActionIcon>
                    </Tooltip>
                    <Button
                      variant="outline"
                      leftSection={<IconUpload size={16} />}
                      onClick={() => dropzoneRef.current?.()}
                      loading={uploadLoading}
                      size="sm"
                    >
                      Upload files
                    </Button>
                    <Button
                      variant="outline"
                      leftSection={<IconFolderPlus size={16} />}
                      onClick={openCreateFolder}
                      size="sm"
                    >
                      Create folder
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
              <Box style={{ flex: 1, display: 'flex' }}>
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
                        {objects.map((obj) => {
                          const FileTypeIcon = isFolder(obj) ? IconFolder : getFileIcon(obj);
                          const isSelected = selectedFile?.id === obj.id;

                          return (
                            <Paper
                              key={obj.id}
                              p="sm"
                              style={{
                                cursor: 'pointer',
                                border: isSelected
                                  ? '1px solid var(--supabase-brand)'
                                  : '1px solid var(--supabase-border)',
                                backgroundColor: isSelected
                                  ? 'var(--supabase-brand-light)'
                                  : 'transparent',
                                borderRadius: 6,
                              }}
                              onClick={() => handleFileClick(obj)}
                            >
                              <Group justify="space-between" wrap="nowrap">
                                <Group gap="sm" wrap="nowrap">
                                  <FileTypeIcon
                                    size={20}
                                    color={
                                      isFolder(obj)
                                        ? 'var(--supabase-brand)'
                                        : 'var(--supabase-text-muted)'
                                    }
                                    stroke={1.5}
                                  />
                                  <Box>
                                    <Text size="sm" fw={500} truncate style={{ maxWidth: 400 }}>
                                      {obj.name.split('/').pop() || obj.name}
                                    </Text>
                                    {!isFolder(obj) && (
                                      <Text size="xs" c="dimmed">
                                        {formatFileSize(obj.size)} {obj.content_type && `- ${obj.content_type}`}
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
                                            setSelectedFile(obj);
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
                          );
                        })}
                      </Stack>
                    )}
                  </Dropzone>
                </Box>

                {/* File Preview Panel */}
                {selectedFile && currentBucket && (
                  <FilePreviewPanel
                    file={selectedFile}
                    bucket={currentBucket}
                    onClose={() => setSelectedFile(null)}
                    onDownload={() => handleDownload(selectedFile)}
                    onDelete={() => openDeleteObject()}
                    onCopyUrl={() => handleCopyUrl(selectedFile)}
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
