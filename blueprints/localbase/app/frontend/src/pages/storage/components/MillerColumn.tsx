import { Box, Text, Center, Loader, Stack, ActionIcon, ScrollArea } from '@mantine/core';
import {
  IconArrowLeft,
  IconFolder,
  IconFile,
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
  IconChevronRight,
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
import type { StorageObject } from '../../../types';

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

function isFolder(obj: StorageObject): boolean {
  return obj.name.endsWith('/') || !obj.content_type;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

interface MillerColumnProps {
  path: string;
  items: StorageObject[];
  selectedItem: string | null;
  loading: boolean;
  error: string | null;
  columnIndex: number;
  showBackButton: boolean;
  onItemSelect: (item: StorageObject, columnIndex: number) => void;
  onBack: (columnIndex: number) => void;
}

export function MillerColumn({
  path,
  items,
  selectedItem,
  loading,
  error,
  columnIndex,
  showBackButton,
  onItemSelect,
  onBack,
}: MillerColumnProps) {
  const folderName = path ? path.split('/').pop() || path : 'Root';

  return (
    <Box
      style={{
        minWidth: 220,
        maxWidth: 280,
        width: 240,
        borderRight: '1px solid var(--supabase-border)',
        display: 'flex',
        flexDirection: 'column',
        backgroundColor: 'var(--supabase-bg)',
        flexShrink: 0,
      }}
    >
      {/* Column Header */}
      <Box
        px="xs"
        py={6}
        style={{
          borderBottom: '1px solid var(--supabase-border)',
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          minHeight: 36,
        }}
      >
        {showBackButton && (
          <ActionIcon
            variant="subtle"
            size="sm"
            onClick={() => onBack(columnIndex)}
            style={{ flexShrink: 0 }}
          >
            <IconArrowLeft size={16} />
          </ActionIcon>
        )}
        <Text size="sm" fw={500} truncate style={{ flex: 1 }}>
          {folderName}
        </Text>
      </Box>

      {/* Column Content */}
      <ScrollArea style={{ flex: 1 }} scrollbarSize={6}>
        {loading ? (
          <Center py="xl">
            <Loader size="sm" />
          </Center>
        ) : error ? (
          <Center py="xl">
            <Text size="sm" c="red">{error}</Text>
          </Center>
        ) : items.length === 0 ? (
          <Center py="xl">
            <Text size="sm" c="dimmed">Empty folder</Text>
          </Center>
        ) : (
          <Stack gap={0}>
            {items.map((item) => {
              const isFolderItem = isFolder(item);
              const ItemIcon = isFolderItem ? IconFolder : getFileIcon(item);
              const displayName = item.name.split('/').filter(Boolean).pop() || item.name;
              const isSelected = selectedItem === item.name;

              return (
                <Box
                  key={item.id || item.name}
                  px="xs"
                  py={6}
                  style={{
                    cursor: 'pointer',
                    borderLeft: isSelected ? '2px solid var(--supabase-brand)' : '2px solid transparent',
                    backgroundColor: isSelected ? 'var(--supabase-brand-light)' : 'transparent',
                    display: 'flex',
                    alignItems: 'center',
                    gap: 8,
                    transition: 'background-color 0.1s ease',
                  }}
                  onClick={() => onItemSelect(item, columnIndex)}
                  onMouseEnter={(e) => {
                    if (!isSelected) {
                      e.currentTarget.style.backgroundColor = 'var(--supabase-bg-surface)';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!isSelected) {
                      e.currentTarget.style.backgroundColor = 'transparent';
                    }
                  }}
                >
                  <ItemIcon
                    size={18}
                    color={isFolderItem ? 'var(--supabase-brand)' : 'var(--supabase-text-muted)'}
                    stroke={1.5}
                    style={{ flexShrink: 0 }}
                  />
                  <Box style={{ flex: 1, minWidth: 0 }}>
                    <Text size="sm" truncate>
                      {displayName}
                    </Text>
                    {!isFolderItem && item.size > 0 && (
                      <Text size="xs" c="dimmed">
                        {formatFileSize(item.size)}
                      </Text>
                    )}
                  </Box>
                  {isFolderItem && (
                    <IconChevronRight
                      size={14}
                      color="var(--supabase-text-muted)"
                      style={{ flexShrink: 0 }}
                    />
                  )}
                </Box>
              );
            })}
          </Stack>
        )}
      </ScrollArea>
    </Box>
  );
}
