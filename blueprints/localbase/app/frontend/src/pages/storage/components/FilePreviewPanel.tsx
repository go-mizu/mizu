import { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Text,
  Group,
  CloseButton,
  Divider,
  Image,
  Code,
  ScrollArea,
  Center,
  Loader,
  Menu,
  Stack,
} from '@mantine/core';
import {
  IconDownload,
  IconLink,
  IconTrash,
  IconChevronDown,
  IconExternalLink,
  IconPhoto,
  IconVideo,
  IconMusic,
  IconFile,
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
  IconFileSpreadsheet,
  IconPresentation,
  IconFileTypeDocx,
  IconFileTypeCsv,
  IconFileTypeXml,
  IconFileTypeSql,
  IconBrandRust,
  IconBrandDocker,
  IconTerminal2,
  IconSettingsCode,
} from '@tabler/icons-react';
import { storageApi } from '../../../api';
import type { Bucket, StorageObject } from '../../../types';

function getFileIcon(file: StorageObject) {
  const contentType = file.content_type || '';
  const ext = file.name.split('.').pop()?.toLowerCase();
  const fileName = file.name.split('/').pop()?.toLowerCase() || '';

  if (contentType.startsWith('image/')) return IconPhoto;
  if (contentType.startsWith('video/')) return IconVideo;
  if (contentType.startsWith('audio/')) return IconMusic;
  if (contentType === 'application/pdf') return IconFileTypePdf;
  if (contentType === 'application/json') return IconBraces;
  if (contentType === 'text/csv' || contentType === 'application/csv') return IconFileTypeCsv;
  if (contentType === 'application/xml' || contentType === 'text/xml') return IconFileTypeXml;
  if (contentType === 'application/sql' || contentType === 'text/x-sql') return IconFileTypeSql;
  if (contentType.includes('spreadsheet') || contentType.includes('excel')) return IconFileSpreadsheet;
  if (contentType.includes('presentation') || contentType.includes('powerpoint')) return IconPresentation;
  if (contentType.includes('wordprocessing') || contentType.includes('msword')) return IconFileTypeDocx;

  if (fileName === 'dockerfile') return IconBrandDocker;
  if (fileName === 'makefile') return IconTerminal2;

  switch (ext) {
    case 'js':
    case 'jsx':
    case 'ts':
    case 'tsx':
      return IconBrandJavascript;
    case 'py':
      return IconBrandPython;
    case 'rs':
      return IconBrandRust;
    case 'go':
    case 'java':
    case 'c':
    case 'cpp':
    case 'h':
    case 'hpp':
    case 'swift':
    case 'kt':
    case 'rb':
    case 'php':
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
    case 'tgz':
      return IconFileZip;
    case 'txt':
    case 'log':
      return IconFileText;
    case 'pdf':
      return IconFileTypePdf;
    case 'csv':
      return IconFileTypeCsv;
    case 'xml':
      return IconFileTypeXml;
    case 'sql':
      return IconFileTypeSql;
    case 'xlsx':
    case 'xls':
      return IconFileSpreadsheet;
    case 'pptx':
    case 'ppt':
      return IconPresentation;
    case 'docx':
    case 'doc':
      return IconFileTypeDocx;
    case 'yaml':
    case 'yml':
    case 'toml':
    case 'ini':
    case 'conf':
    case 'cfg':
      return IconSettingsCode;
    case 'sh':
    case 'bash':
    case 'zsh':
      return IconTerminal2;
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
    'text/csv',
    'text/markdown',
    'text/typescript',
    'text/x-python',
    'text/x-go',
    'text/x-rust',
    'text/x-java',
    'application/json',
    'application/javascript',
    'application/xml',
    'application/sql',
    'application/x-yaml',
    'application/toml',
  ];

  if (contentType && textTypes.some(t => contentType.includes(t))) return true;

  const ext = name.split('.').pop()?.toLowerCase();
  const fileName = name.split('/').pop()?.toLowerCase() || '';

  const specialFiles = ['dockerfile', 'makefile', 'gemfile', 'rakefile', 'procfile'];
  if (specialFiles.includes(fileName)) return true;

  const textExtensions = [
    'txt', 'md', 'mdx', 'json', 'js', 'jsx', 'ts', 'tsx', 'py', 'go', 'rs',
    'java', 'c', 'cpp', 'h', 'hpp', 'css', 'scss', 'less', 'html', 'htm',
    'xml', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'sh', 'bash', 'zsh',
    'sql', 'graphql', 'vue', 'svelte', 'astro', 'rb', 'php', 'swift', 'kt',
    'gradle', 'env', 'gitignore', 'log', 'csv', 'tsv', 'properties',
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

interface FilePreviewPanelProps {
  file: StorageObject;
  bucket: Bucket;
  onClose: () => void;
  onDownload: () => void;
  onDelete: () => void;
  onCopyUrl: (type: 'public' | 'signed') => void;
}

export function FilePreviewPanel({
  file,
  bucket,
  onClose,
  onDownload,
  onDelete,
  onCopyUrl,
}: FilePreviewPanelProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [loadingPreview, setLoadingPreview] = useState(false);

  const FileIcon = getFileIcon(file);
  const fileName = file.name.split('/').pop() || file.name;
  const contentType = file.content_type || 'Unknown';

  useEffect(() => {
    if (bucket) {
      const url = storageApi.downloadObjectUrl(bucket.id, file.name);
      setPreviewUrl(url);

      if (isPreviewableText(file.content_type, file.name)) {
        setLoadingPreview(true);
        fetch(url)
          .then((res) => res.text())
          .then((text) => {
            setTextContent(text.slice(0, 10000));
          })
          .catch(() => setTextContent(null))
          .finally(() => setLoadingPreview(false));
      } else {
        setTextContent(null);
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
        minWidth: 320,
        borderLeft: '1px solid var(--supabase-border)',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: 'var(--supabase-bg)',
        flexShrink: 0,
      }}
    >
      {/* Header */}
      <Box
        px="md"
        py="xs"
        style={{
          borderBottom: '1px solid var(--supabase-border)',
          minHeight: 40,
          display: 'flex',
          alignItems: 'center',
        }}
      >
        <Group justify="flex-end" style={{ width: '100%' }}>
          <CloseButton onClick={onClose} />
        </Group>
      </Box>

      {/* Preview Area */}
      <ScrollArea style={{ flex: 1 }} offsetScrollbars>
        <Box p="md">
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
                {bucket.public && (
                  <Menu.Item
                    leftSection={<IconLink size={14} />}
                    onClick={() => onCopyUrl('public')}
                  >
                    Copy public URL
                  </Menu.Item>
                )}
                <Menu.Item
                  leftSection={<IconLink size={14} />}
                  onClick={() => onCopyUrl('signed')}
                >
                  Copy signed URL (1 hour)
                </Menu.Item>
                <Menu.Item
                  leftSection={<IconExternalLink size={14} />}
                  onClick={() => {
                    const url = bucket.public
                      ? `${window.location.origin}${storageApi.getPublicUrl(bucket.id, file.name)}`
                      : storageApi.downloadObjectUrl(bucket.id, file.name);
                    window.open(url, '_blank');
                  }}
                >
                  Open in new tab
                </Menu.Item>
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
      </ScrollArea>
    </Box>
  );
}
