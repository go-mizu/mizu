import { useRef, useState } from 'react';
import {
  Box,
  Group,
  Text,
  ActionIcon,
  Menu,
  Tooltip,
  CloseButton,
} from '@mantine/core';
import { IconTable, IconPlus } from '@tabler/icons-react';

export interface TableTab {
  id: string;
  schema: string;
  table: string;
  isTransient: boolean;
  isDirty?: boolean;
}

interface TableTabsProps {
  tabs: TableTab[];
  activeTabId: string | null;
  onTabClick: (tabId: string) => void;
  onTabDoubleClick: (tabId: string) => void;
  onTabClose: (tabId: string) => void;
  onAddTab?: () => void;
}

export function TableTabs({
  tabs,
  activeTabId,
  onTabClick,
  onTabDoubleClick,
  onTabClose,
  onAddTab,
}: TableTabsProps) {
  const [contextMenuTab, setContextMenuTab] = useState<string | null>(null);
  const tabsRef = useRef<HTMLDivElement>(null);

  const handleContextMenu = (e: React.MouseEvent, tabId: string) => {
    e.preventDefault();
    setContextMenuTab(tabId);
  };

  const handleCloseOthers = (tabId: string) => {
    tabs.forEach((tab) => {
      if (tab.id !== tabId) {
        onTabClose(tab.id);
      }
    });
    setContextMenuTab(null);
  };

  const handleCloseToRight = (tabId: string) => {
    const tabIndex = tabs.findIndex((t) => t.id === tabId);
    tabs.forEach((tab, index) => {
      if (index > tabIndex) {
        onTabClose(tab.id);
      }
    });
    setContextMenuTab(null);
  };

  if (tabs.length === 0) {
    return null;
  }

  return (
    <Box
      ref={tabsRef}
      style={{
        borderBottom: '1px solid var(--supabase-border)',
        backgroundColor: 'var(--supabase-bg)',
        display: 'flex',
        alignItems: 'center',
        paddingLeft: 12,
        paddingRight: 8,
        minHeight: 40,
        gap: 0,
        overflowX: 'auto',
      }}
    >
      <Group gap={0} style={{ flex: 1, minWidth: 0 }} wrap="nowrap">
        {tabs.map((tab) => {
          const isActive = tab.id === activeTabId;
          return (
            <Menu
              key={tab.id}
              opened={contextMenuTab === tab.id}
              onClose={() => setContextMenuTab(null)}
              position="bottom-start"
              shadow="md"
              withinPortal
            >
              <Menu.Target>
                <Box
                  px="sm"
                  py={8}
                  onContextMenu={(e) => handleContextMenu(e, tab.id)}
                  onClick={() => onTabClick(tab.id)}
                  onDoubleClick={() => onTabDoubleClick(tab.id)}
                  style={{
                    borderBottom: isActive ? '2px solid var(--supabase-brand)' : '2px solid transparent',
                    marginBottom: -1,
                    cursor: 'pointer',
                    backgroundColor: isActive ? 'transparent' : 'transparent',
                    transition: 'background-color 0.1s ease',
                    maxWidth: 200,
                    minWidth: 0,
                    display: 'flex',
                    alignItems: 'center',
                    gap: 6,
                  }}
                  onMouseEnter={(e) => {
                    if (!isActive) {
                      e.currentTarget.style.backgroundColor = 'var(--supabase-bg-surface)';
                    }
                  }}
                  onMouseLeave={(e) => {
                    if (!isActive) {
                      e.currentTarget.style.backgroundColor = 'transparent';
                    }
                  }}
                >
                  <Group gap={6} wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
                    <IconTable
                      size={14}
                      style={{
                        flexShrink: 0,
                        color: isActive ? 'var(--supabase-brand)' : 'var(--supabase-text-muted)',
                      }}
                    />
                    <Text
                      size="sm"
                      fw={isActive ? 500 : 400}
                      truncate
                      style={{
                        color: isActive ? 'var(--supabase-brand)' : 'var(--supabase-text)',
                        fontStyle: tab.isTransient ? 'italic' : 'normal',
                      }}
                    >
                      {tab.table}
                    </Text>
                    {tab.isDirty && (
                      <Box
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: '50%',
                          backgroundColor: 'var(--supabase-warning)',
                          flexShrink: 0,
                        }}
                      />
                    )}
                  </Group>
                  <CloseButton
                    size="xs"
                    variant="subtle"
                    onClick={(e) => {
                      e.stopPropagation();
                      onTabClose(tab.id);
                    }}
                    style={{
                      opacity: isActive ? 0.7 : 0.3,
                      flexShrink: 0,
                    }}
                  />
                </Box>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Item onClick={() => onTabClose(tab.id)}>
                  Close
                </Menu.Item>
                <Menu.Item onClick={() => handleCloseOthers(tab.id)}>
                  Close Others
                </Menu.Item>
                <Menu.Item onClick={() => handleCloseToRight(tab.id)}>
                  Close to the Right
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          );
        })}
      </Group>

      {onAddTab && (
        <Tooltip label="Open table in new tab">
          <ActionIcon
            variant="subtle"
            size="sm"
            ml={4}
            onClick={onAddTab}
            style={{ opacity: 0.5, flexShrink: 0 }}
          >
            <IconPlus size={14} />
          </ActionIcon>
        </Tooltip>
      )}
    </Box>
  );
}
