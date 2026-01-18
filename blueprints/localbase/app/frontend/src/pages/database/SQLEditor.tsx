import { useState, useCallback, useRef } from 'react';
import {
  Box,
  Button,
  Group,
  Text,
  Stack,
  Paper,
  Table,
  ScrollArea,
  Badge,
  ActionIcon,
  Tooltip,
  TextInput,
  Collapse,
  Menu,
  Select,
  Modal,
  SegmentedControl,
  Kbd,
  Divider,
  UnstyledButton,
  Center,
  Loader,
  Drawer,
  Slider,
  Switch,
  Textarea,
  ThemeIcon,
} from '@mantine/core';
import {
  IconTrash,
  IconDownload,
  IconPlus,
  IconX,
  IconChevronRight,
  IconChevronDown,
  IconSearch,
  IconStar,
  IconStarFilled,
  IconTemplate,
  IconBook,
  IconChartBar,
  IconFileAnalytics,
  IconDotsVertical,
  IconCopy,
  IconFolder,
  IconFile,
  IconHistory,
  IconPlayerPlay,
  IconAdjustments,
  IconCode,
  IconBraces,
  IconTableExport,
  IconMinus,
  IconCircleCheck,
  IconCircleX,
  IconWand,
  IconSparkles,
} from '@tabler/icons-react';
import Editor, { Monaco } from '@monaco-editor/react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { databaseApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { QueryResult, QueryHistoryEntry } from '../../types';

interface QueryTab {
  id: string;
  name: string;
  query: string;
  selectedText: string | null;
  isDirty: boolean;
  isTransient: boolean;
  result: QueryResult | null;
  error: string | null;
  loading: boolean;
  executionRole: string;
}

// Query templates
const QUERY_TEMPLATES = [
  {
    category: 'Common',
    items: [
      { name: 'Select all', query: 'SELECT * FROM table_name LIMIT 100;' },
      { name: 'Count rows', query: 'SELECT COUNT(*) FROM table_name;' },
      { name: 'Insert row', query: "INSERT INTO table_name (column1, column2)\nVALUES ('value1', 'value2');" },
      { name: 'Update row', query: "UPDATE table_name\nSET column1 = 'new_value'\nWHERE id = 1;" },
      { name: 'Delete row', query: 'DELETE FROM table_name WHERE id = 1;' },
    ],
  },
  {
    category: 'Auth',
    items: [
      { name: 'List users', query: 'SELECT id, email, created_at FROM auth.users ORDER BY created_at DESC LIMIT 100;' },
      { name: 'User by email', query: "SELECT * FROM auth.users WHERE email = 'user@example.com';" },
    ],
  },
  {
    category: 'Database',
    items: [
      { name: 'List tables', query: "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public';" },
      { name: 'Table columns', query: "SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'table_name';" },
      { name: 'Table size', query: "SELECT pg_size_pretty(pg_total_relation_size('table_name'));" },
      { name: 'Active connections', query: 'SELECT count(*) FROM pg_stat_activity;' },
    ],
  },
  {
    category: 'RLS',
    items: [
      { name: 'List policies', query: 'SELECT * FROM pg_policies;' },
      { name: 'Enable RLS', query: 'ALTER TABLE table_name ENABLE ROW LEVEL SECURITY;' },
      { name: 'Create policy', query: "CREATE POLICY policy_name ON table_name\nFOR SELECT\nUSING (auth.uid() = user_id);" },
    ],
  },
];

// Quick start queries
const QUICKSTARTS = [
  { name: 'Hello World', query: "SELECT 'Hello, World!' AS greeting;" },
  { name: 'Current time', query: 'SELECT NOW() AS current_time;' },
  { name: 'PostgreSQL version', query: 'SELECT version();' },
  { name: 'Database size', query: "SELECT pg_size_pretty(pg_database_size(current_database())) AS db_size;" },
];

export function SQLEditorPage() {
  const { savedQueries, addSavedQuery, removeSavedQuery } = useAppStore();

  // Tabs state
  const [tabs, setTabs] = useState<QueryTab[]>([
    {
      id: crypto.randomUUID(),
      name: 'New query',
      query: '',
      selectedText: null,
      isDirty: false,
      isTransient: true,
      result: null,
      error: null,
      loading: false,
      executionRole: 'postgres',
    },
  ]);
  const [activeTabId, setActiveTabId] = useState(tabs[0].id);

  // Sidebar sections
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(['PRIVATE'])
  );
  const [searchQuery, setSearchQuery] = useState('');

  // Result tabs
  const [resultTab, setResultTab] = useState<'results' | 'explain' | 'chart'>('results');

  // Save modal
  const [saveModalOpened, setSaveModalOpened] = useState(false);
  const [newQueryName, setNewQueryName] = useState('');

  // Role selection
  const [selectedRole, setSelectedRole] = useState('postgres');

  // Favorites (local state, could be persisted)
  const [favorites, setFavorites] = useState<Set<string>>(new Set());

  // Query history
  const [historyDrawerOpened, setHistoryDrawerOpened] = useState(false);
  const [queryHistory, setQueryHistory] = useState<QueryHistoryEntry[]>([]);
  const [historyLoading, setHistoryLoading] = useState(false);

  // Editor settings
  const [fontSize, setFontSize] = useState(14);
  const [settingsOpened, setSettingsOpened] = useState(false);
  const [wordWrap, setWordWrap] = useState(true);
  const [minimap, setMinimap] = useState(false);

  // AI Assistant
  const [aiModalOpened, setAiModalOpened] = useState(false);
  const [aiPrompt, setAiPrompt] = useState('');

  // Results panel
  const [resultsPanelCollapsed, setResultsPanelCollapsed] = useState(false);

  // Editor ref
  const editorRef = useRef<any>(null);
  const monacoRef = useRef<Monaco | null>(null);

  // Get active tab
  const activeTab = tabs.find((t) => t.id === activeTabId) || tabs[0];

  // Load query history
  const loadQueryHistory = useCallback(async () => {
    setHistoryLoading(true);
    try {
      const history = await databaseApi.getQueryHistory(100, 0);
      setQueryHistory(history);
    } catch (err) {
      console.error('Failed to load query history:', err);
    } finally {
      setHistoryLoading(false);
    }
  }, []);

  // Toggle section
  const toggleSection = (section: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(section)) {
        next.delete(section);
      } else {
        next.add(section);
      }
      return next;
    });
  };

  // Create new tab
  const createTab = () => {
    const newTab: QueryTab = {
      id: crypto.randomUUID(),
      name: 'New query',
      query: '',
      selectedText: null,
      isDirty: false,
      isTransient: true,
      result: null,
      error: null,
      loading: false,
      executionRole: selectedRole,
    };
    setTabs([...tabs, newTab]);
    setActiveTabId(newTab.id);
  };

  // Close tab
  const closeTab = (id: string) => {
    if (tabs.length === 1) {
      // Don't close the last tab, just reset it
      setTabs([
        {
          id: crypto.randomUUID(),
          name: 'New query',
          query: '',
          selectedText: null,
          isDirty: false,
          isTransient: true,
          result: null,
          error: null,
          loading: false,
          executionRole: selectedRole,
        },
      ]);
      setActiveTabId(tabs[0].id);
      return;
    }

    const index = tabs.findIndex((t) => t.id === id);
    const newTabs = tabs.filter((t) => t.id !== id);
    setTabs(newTabs);

    if (activeTabId === id) {
      // Switch to the previous tab or the first one
      const newIndex = Math.max(0, index - 1);
      setActiveTabId(newTabs[newIndex].id);
    }
  };

  // Update tab query
  const updateTabQuery = (query: string) => {
    setTabs((prev) =>
      prev.map((t) =>
        t.id === activeTabId
          ? { ...t, query, isDirty: query !== '' }
          : t
      )
    );
  };

  // Update selected text
  const updateSelectedText = () => {
    if (editorRef.current) {
      const selection = editorRef.current.getSelection();
      const model = editorRef.current.getModel();
      if (selection && model && !selection.isEmpty()) {
        const selectedText = model.getValueInRange(selection);
        setTabs((prev) =>
          prev.map((t) =>
            t.id === activeTabId
              ? { ...t, selectedText }
              : t
          )
        );
      } else {
        setTabs((prev) =>
          prev.map((t) =>
            t.id === activeTabId
              ? { ...t, selectedText: null }
              : t
          )
        );
      }
    }
  };

  // Execute query
  const executeQuery = useCallback(async (explain = false) => {
    // Use selected text if available, otherwise use full query
    const queryToRun = activeTab.selectedText || activeTab.query;

    if (!queryToRun.trim()) {
      notifications.show({
        title: 'Error',
        message: 'Please enter a query',
        color: 'red',
      });
      return;
    }

    setTabs((prev) =>
      prev.map((t) =>
        t.id === activeTabId
          ? { ...t, loading: true, error: null, result: null }
          : t
      )
    );

    try {
      const data = await databaseApi.executeQuery(queryToRun, {
        role: selectedRole,
        explain,
      });
      setTabs((prev) =>
        prev.map((t) =>
          t.id === activeTabId
            ? { ...t, loading: false, result: data }
            : t
        )
      );

      // Switch to explain tab if explain was requested
      if (explain) {
        setResultTab('explain');
      } else {
        setResultTab('results');
      }
    } catch (err: any) {
      const errorMessage = err.message || 'Query execution failed';
      setTabs((prev) =>
        prev.map((t) =>
          t.id === activeTabId
            ? { ...t, loading: false, error: errorMessage }
            : t
        )
      );
      notifications.show({
        title: 'Query Error',
        message: errorMessage,
        color: 'red',
      });
    }
  }, [activeTab, activeTabId, selectedRole]);

  // Save query
  const handleSaveQuery = () => {
    if (!newQueryName.trim()) {
      notifications.show({
        title: 'Error',
        message: 'Please enter a name',
        color: 'red',
      });
      return;
    }

    addSavedQuery(newQueryName, activeTab.query);
    setSaveModalOpened(false);
    setNewQueryName('');
    notifications.show({
      title: 'Saved',
      message: 'Query saved successfully',
      color: 'green',
    });

    // Update tab name and make it permanent
    setTabs((prev) =>
      prev.map((t) =>
        t.id === activeTabId
          ? { ...t, name: newQueryName, isDirty: false, isTransient: false }
          : t
      )
    );
  };

  // Load saved query into current tab
  const loadQuery = (name: string, query: string) => {
    setTabs((prev) =>
      prev.map((t) =>
        t.id === activeTabId
          ? { ...t, name, query, isDirty: false, isTransient: false }
          : t
      )
    );
  };

  // Load query into new tab
  const loadQueryNewTab = (name: string, query: string) => {
    const newTab: QueryTab = {
      id: crypto.randomUUID(),
      name,
      query,
      selectedText: null,
      isDirty: false,
      isTransient: false,
      result: null,
      error: null,
      loading: false,
      executionRole: selectedRole,
    };
    setTabs([...tabs, newTab]);
    setActiveTabId(newTab.id);
  };

  // Toggle favorite
  const toggleFavorite = (id: string) => {
    setFavorites((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  // Export results as CSV
  const exportResultsCSV = () => {
    if (!activeTab.result) return;

    const csv = [
      activeTab.result.columns.join(','),
      ...activeTab.result.rows.map((row) =>
        activeTab.result!.columns.map((col) => JSON.stringify(row[col] ?? '')).join(',')
      ),
    ].join('\n');

    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'query_results.csv';
    a.click();
    URL.revokeObjectURL(url);
  };

  // Export results as JSON
  const exportResultsJSON = () => {
    if (!activeTab.result) return;

    const json = JSON.stringify(activeTab.result.rows, null, 2);
    const blob = new Blob([json], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'query_results.json';
    a.click();
    URL.revokeObjectURL(url);
  };

  // Copy as SQL INSERT
  const copyAsSQLInsert = () => {
    if (!activeTab.result || activeTab.result.rows.length === 0) return;

    const columns = activeTab.result.columns;
    const inserts = activeTab.result.rows.map((row) => {
      const values = columns.map((col) => {
        const val = row[col];
        if (val === null) return 'NULL';
        if (typeof val === 'string') return `'${val.replace(/'/g, "''")}'`;
        if (typeof val === 'boolean') return val ? 'TRUE' : 'FALSE';
        if (typeof val === 'object') return `'${JSON.stringify(val).replace(/'/g, "''")}'`;
        return String(val);
      });
      return `INSERT INTO table_name (${columns.join(', ')}) VALUES (${values.join(', ')});`;
    });

    navigator.clipboard.writeText(inserts.join('\n'));
    notifications.show({
      title: 'Copied',
      message: 'SQL INSERT statements copied to clipboard',
      color: 'green',
    });
  };

  // Filter saved queries
  const filteredQueries = savedQueries.filter(
    (q) =>
      q.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      q.query.toLowerCase().includes(searchQuery.toLowerCase())
  );

  // Get favorite queries
  const favoriteQueries = savedQueries.filter((q) => favorites.has(q.id));

  // Handle editor mount
  const handleEditorMount = (editor: any, monaco: Monaco) => {
    editorRef.current = editor;
    monacoRef.current = monaco;

    // Ctrl/Cmd + Enter to run
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter,
      () => executeQuery()
    );

    // Ctrl/Cmd + K for AI assistant
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyK,
      () => setAiModalOpened(true)
    );

    // Ctrl/Cmd + S to save
    editor.addCommand(
      monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS,
      () => setSaveModalOpened(true)
    );

    // Track selection changes
    editor.onDidChangeCursorSelection(() => {
      updateSelectedText();
    });
  };

  // Format duration
  const formatDuration = (ms: number) => {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
    if (ms < 1000) return `${ms.toFixed(2)}ms`;
    return `${(ms / 1000).toFixed(2)}s`;
  };

  // Format date
  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();

    if (diff < 60000) return 'Just now';
    if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`;
    if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <PageContainer title="SQL Editor" description="" fullWidth noPadding noHeader>
      <Box style={{ display: 'flex', height: 'calc(100vh - 60px)' }}>
        {/* Left Sidebar */}
        <Box
          style={{
            width: 280,
            minWidth: 280,
            borderRight: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
            backgroundColor: 'var(--supabase-bg-surface)',
          }}
        >
          {/* Search */}
          <Box p="sm" pb={0}>
            <TextInput
              placeholder="Search queries..."
              size="xs"
              leftSection={<IconSearch size={14} />}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              rightSection={
                <ActionIcon size="xs" variant="subtle" onClick={createTab}>
                  <IconPlus size={14} />
                </ActionIcon>
              }
            />
          </Box>

          <ScrollArea style={{ flex: 1 }} p="sm">
            <Stack gap="xs">
              {/* SHARED Section */}
              <Box>
                <UnstyledButton
                  onClick={() => toggleSection('SHARED')}
                  style={{ width: '100%' }}
                >
                  <Group gap="xs" py={4}>
                    {expandedSections.has('SHARED') ? (
                      <IconChevronDown size={12} />
                    ) : (
                      <IconChevronRight size={12} />
                    )}
                    <Text size="xs" fw={600} c="dimmed" style={{ letterSpacing: 0.5 }}>
                      SHARED
                    </Text>
                  </Group>
                </UnstyledButton>
                <Collapse in={expandedSections.has('SHARED')}>
                  <Box pl="md" py="xs">
                    <Text size="xs" c="dimmed">
                      No shared queries yet
                    </Text>
                  </Box>
                </Collapse>
              </Box>

              {/* FAVORITES Section */}
              <Box>
                <UnstyledButton
                  onClick={() => toggleSection('FAVORITES')}
                  style={{ width: '100%' }}
                >
                  <Group gap="xs" py={4}>
                    {expandedSections.has('FAVORITES') ? (
                      <IconChevronDown size={12} />
                    ) : (
                      <IconChevronRight size={12} />
                    )}
                    <Text size="xs" fw={600} c="dimmed" style={{ letterSpacing: 0.5 }}>
                      FAVORITES
                    </Text>
                  </Group>
                </UnstyledButton>
                <Collapse in={expandedSections.has('FAVORITES')}>
                  <Stack gap={2} pl="xs" py="xs">
                    {favoriteQueries.length === 0 ? (
                      <Text size="xs" c="dimmed" pl="md">
                        No favorites yet
                      </Text>
                    ) : (
                      favoriteQueries.map((q) => (
                        <QueryItem
                          key={q.id}
                          query={q}
                          isFavorite={true}
                          onLoad={() => loadQuery(q.name, q.query)}
                          onLoadNewTab={() => loadQueryNewTab(q.name, q.query)}
                          onToggleFavorite={() => toggleFavorite(q.id)}
                          onDelete={() => removeSavedQuery(q.id)}
                        />
                      ))
                    )}
                  </Stack>
                </Collapse>
              </Box>

              {/* PRIVATE Section */}
              <Box>
                <UnstyledButton
                  onClick={() => toggleSection('PRIVATE')}
                  style={{ width: '100%' }}
                >
                  <Group gap="xs" py={4}>
                    {expandedSections.has('PRIVATE') ? (
                      <IconChevronDown size={12} />
                    ) : (
                      <IconChevronRight size={12} />
                    )}
                    <Text size="xs" fw={600} c="dimmed" style={{ letterSpacing: 0.5 }}>
                      PRIVATE
                    </Text>
                  </Group>
                </UnstyledButton>
                <Collapse in={expandedSections.has('PRIVATE')}>
                  <Stack gap={2} pl="xs" py="xs">
                    {filteredQueries.length === 0 ? (
                      <Box py="md" ta="center">
                        <Text size="xs" c="dimmed" mb="xs">
                          No private queries created yet
                        </Text>
                        <Text size="xs" c="dimmed">
                          Save queries to see them here
                        </Text>
                      </Box>
                    ) : (
                      filteredQueries.map((q) => (
                        <QueryItem
                          key={q.id}
                          query={q}
                          isFavorite={favorites.has(q.id)}
                          onLoad={() => loadQuery(q.name, q.query)}
                          onLoadNewTab={() => loadQueryNewTab(q.name, q.query)}
                          onToggleFavorite={() => toggleFavorite(q.id)}
                          onDelete={() => removeSavedQuery(q.id)}
                        />
                      ))
                    )}
                  </Stack>
                </Collapse>
              </Box>

              <Divider my="xs" />

              {/* COMMUNITY Section */}
              <Box>
                <UnstyledButton
                  onClick={() => toggleSection('COMMUNITY')}
                  style={{ width: '100%' }}
                >
                  <Group gap="xs" py={4}>
                    {expandedSections.has('COMMUNITY') ? (
                      <IconChevronDown size={12} />
                    ) : (
                      <IconChevronRight size={12} />
                    )}
                    <Text size="xs" fw={600} c="dimmed" style={{ letterSpacing: 0.5 }}>
                      COMMUNITY
                    </Text>
                  </Group>
                </UnstyledButton>
                <Collapse in={expandedSections.has('COMMUNITY')}>
                  <Stack gap={2} pl="xs" py="xs">
                    {/* Templates */}
                    <UnstyledButton
                      onClick={() => toggleSection('Templates')}
                      style={{
                        padding: '6px 8px',
                        borderRadius: 4,
                      }}
                    >
                      <Group gap="xs">
                        <IconTemplate size={14} />
                        <Text size="sm">Templates</Text>
                      </Group>
                    </UnstyledButton>
                    <Collapse in={expandedSections.has('Templates')}>
                      <Stack gap={2} pl="md">
                        {QUERY_TEMPLATES.map((category) => (
                          <Box key={category.category}>
                            <Text size="xs" c="dimmed" fw={500} py={4}>
                              {category.category}
                            </Text>
                            {category.items.map((item) => (
                              <UnstyledButton
                                key={item.name}
                                onClick={() => loadQuery(item.name, item.query)}
                                style={{
                                  display: 'block',
                                  width: '100%',
                                  padding: '4px 8px',
                                  borderRadius: 4,
                                }}
                              >
                                <Text size="xs">{item.name}</Text>
                              </UnstyledButton>
                            ))}
                          </Box>
                        ))}
                      </Stack>
                    </Collapse>

                    {/* Quickstarts */}
                    <UnstyledButton
                      onClick={() => toggleSection('Quickstarts')}
                      style={{
                        padding: '6px 8px',
                        borderRadius: 4,
                      }}
                    >
                      <Group gap="xs">
                        <IconBook size={14} />
                        <Text size="sm">Quickstarts</Text>
                      </Group>
                    </UnstyledButton>
                    <Collapse in={expandedSections.has('Quickstarts')}>
                      <Stack gap={2} pl="md">
                        {QUICKSTARTS.map((item) => (
                          <UnstyledButton
                            key={item.name}
                            onClick={() => loadQuery(item.name, item.query)}
                            style={{
                              display: 'block',
                              width: '100%',
                              padding: '4px 8px',
                              borderRadius: 4,
                            }}
                          >
                            <Text size="xs">{item.name}</Text>
                          </UnstyledButton>
                        ))}
                      </Stack>
                    </Collapse>
                  </Stack>
                </Collapse>
              </Box>
            </Stack>
          </ScrollArea>

          {/* Running queries / History button */}
          <Box p="sm" style={{ borderTop: '1px solid var(--supabase-border)' }}>
            <Button
              variant="subtle"
              fullWidth
              size="xs"
              leftSection={<IconHistory size={14} />}
              onClick={() => {
                loadQueryHistory();
                setHistoryDrawerOpened(true);
              }}
            >
              Query History
            </Button>
          </Box>
        </Box>

        {/* Main Editor Area */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', minWidth: 0 }}>
          {/* Tab Bar */}
          <Box
            style={{
              borderBottom: '1px solid var(--supabase-border)',
              backgroundColor: 'var(--supabase-bg)',
              display: 'flex',
              alignItems: 'center',
            }}
          >
            <ScrollArea style={{ flex: 1 }} scrollbarSize={4} type="scroll">
              <Group gap={0} wrap="nowrap" px="xs">
                {tabs.map((tab) => (
                  <Box
                    key={tab.id}
                    onClick={() => setActiveTabId(tab.id)}
                    onDoubleClick={() => {
                      // Make transient tab permanent on double-click
                      setTabs((prev) =>
                        prev.map((t) =>
                          t.id === tab.id ? { ...t, isTransient: false } : t
                        )
                      );
                    }}
                    style={{
                      padding: '8px 12px',
                      cursor: 'pointer',
                      borderBottom:
                        activeTabId === tab.id
                          ? '2px solid var(--supabase-brand)'
                          : '2px solid transparent',
                      backgroundColor:
                        activeTabId === tab.id
                          ? 'var(--supabase-bg-surface)'
                          : 'transparent',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 8,
                      whiteSpace: 'nowrap',
                    }}
                  >
                    <IconFile size={14} />
                    <Text size="sm" style={{ fontStyle: tab.isTransient ? 'italic' : 'normal' }}>
                      {tab.name}
                      {tab.isDirty && ' *'}
                    </Text>
                    <ActionIcon
                      size="xs"
                      variant="subtle"
                      onClick={(e) => {
                        e.stopPropagation();
                        closeTab(tab.id);
                      }}
                    >
                      <IconX size={12} />
                    </ActionIcon>
                  </Box>
                ))}

                {/* New Tab Button */}
                <Tooltip label="New query">
                  <ActionIcon
                    variant="subtle"
                    size="sm"
                    onClick={createTab}
                    mx="xs"
                  >
                    <IconPlus size={16} />
                  </ActionIcon>
                </Tooltip>
              </Group>
            </ScrollArea>
          </Box>

          {/* Editor Toolbar */}
          <Box
            px="sm"
            py={6}
            style={{
              borderBottom: '1px solid var(--supabase-border)',
              backgroundColor: 'var(--supabase-bg)',
            }}
          >
            <Group justify="space-between">
              <Group gap="xs">
                <Tooltip label="AI Assistant (⌘K)">
                  <ActionIcon variant="subtle" size="sm" onClick={() => setAiModalOpened(true)}>
                    <IconSparkles size={16} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Format SQL">
                  <ActionIcon variant="subtle" size="sm">
                    <IconCode size={16} />
                  </ActionIcon>
                </Tooltip>
                <Divider orientation="vertical" />
                <Tooltip label="Decrease font size">
                  <ActionIcon
                    variant="subtle"
                    size="sm"
                    onClick={() => setFontSize((s) => Math.max(10, s - 1))}
                  >
                    <IconMinus size={14} />
                  </ActionIcon>
                </Tooltip>
                <Text size="xs" w={40} ta="center">{fontSize}px</Text>
                <Tooltip label="Increase font size">
                  <ActionIcon
                    variant="subtle"
                    size="sm"
                    onClick={() => setFontSize((s) => Math.min(24, s + 1))}
                  >
                    <IconPlus size={14} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip label="Settings">
                  <ActionIcon variant="subtle" size="sm" onClick={() => setSettingsOpened(true)}>
                    <IconAdjustments size={16} />
                  </ActionIcon>
                </Tooltip>
              </Group>
              <Group gap="xs">
                {activeTab.selectedText && (
                  <Badge size="xs" variant="light" color="blue">
                    Selection active
                  </Badge>
                )}
                <Button
                  size="xs"
                  variant="subtle"
                  leftSection={<IconDownload size={14} />}
                  onClick={() => setSaveModalOpened(true)}
                >
                  Save
                </Button>
              </Group>
            </Group>
          </Box>

          {/* Editor */}
          <Box style={{ flex: resultsPanelCollapsed ? 1 : '0 0 50%', borderBottom: '1px solid var(--supabase-border)' }}>
            <Editor
              height="100%"
              defaultLanguage="sql"
              value={activeTab.query}
              onChange={(value) => updateTabQuery(value || '')}
              theme="vs-light"
              options={{
                minimap: { enabled: minimap },
                fontSize,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                wordWrap: wordWrap ? 'on' : 'off',
              }}
              onMount={handleEditorMount}
            />
          </Box>

          {/* Results Panel */}
          <Box style={{ flex: resultsPanelCollapsed ? 0 : 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Results Header */}
            <Box
              px="sm"
              py="xs"
              style={{
                borderBottom: resultsPanelCollapsed ? 'none' : '1px solid var(--supabase-border)',
                backgroundColor: 'var(--supabase-bg)',
              }}
            >
              <Group justify="space-between">
                <Group gap="xs">
                  <ActionIcon
                    size="xs"
                    variant="subtle"
                    onClick={() => setResultsPanelCollapsed(!resultsPanelCollapsed)}
                  >
                    {resultsPanelCollapsed ? <IconChevronDown size={14} /> : <IconChevronRight size={14} style={{ transform: 'rotate(90deg)' }} />}
                  </ActionIcon>
                  <SegmentedControl
                    size="xs"
                    value={resultTab}
                    onChange={(v) => setResultTab(v as 'results' | 'explain' | 'chart')}
                    data={[
                      { label: 'Results', value: 'results' },
                      { label: 'Explain', value: 'explain' },
                      { label: 'Chart', value: 'chart' },
                    ]}
                  />
                  {activeTab.result && (
                    <Badge size="sm" variant="light" color="green">
                      {activeTab.result.row_count} rows in {formatDuration(activeTab.result.duration_ms)}
                    </Badge>
                  )}
                </Group>
                <Group gap="xs">
                  {activeTab.result && activeTab.result.rows.length > 0 && (
                    <>
                      <Menu position="bottom-end">
                        <Menu.Target>
                          <Button size="xs" variant="subtle" leftSection={<IconCopy size={14} />}>
                            Copy
                          </Button>
                        </Menu.Target>
                        <Menu.Dropdown>
                          <Menu.Item
                            leftSection={<IconCopy size={14} />}
                            onClick={() => {
                              navigator.clipboard.writeText(JSON.stringify(activeTab.result?.rows, null, 2));
                              notifications.show({ message: 'Copied as JSON', color: 'green' });
                            }}
                          >
                            Copy as JSON
                          </Menu.Item>
                          <Menu.Item
                            leftSection={<IconTableExport size={14} />}
                            onClick={copyAsSQLInsert}
                          >
                            Copy as SQL INSERT
                          </Menu.Item>
                        </Menu.Dropdown>
                      </Menu>
                      <Menu position="bottom-end">
                        <Menu.Target>
                          <Button size="xs" variant="subtle" leftSection={<IconDownload size={14} />}>
                            Export
                          </Button>
                        </Menu.Target>
                        <Menu.Dropdown>
                          <Menu.Item
                            leftSection={<IconDownload size={14} />}
                            onClick={exportResultsCSV}
                          >
                            Export as CSV
                          </Menu.Item>
                          <Menu.Item
                            leftSection={<IconBraces size={14} />}
                            onClick={exportResultsJSON}
                          >
                            Export as JSON
                          </Menu.Item>
                        </Menu.Dropdown>
                      </Menu>
                    </>
                  )}
                  <Divider orientation="vertical" />
                  <Select
                    size="xs"
                    value="primary"
                    data={[{ value: 'primary', label: 'Primary Database' }]}
                    w={150}
                    styles={{ input: { fontSize: 12 } }}
                  />
                  <Select
                    size="xs"
                    value={selectedRole}
                    onChange={(v) => v && setSelectedRole(v)}
                    data={[
                      { value: 'postgres', label: 'postgres' },
                      { value: 'anon', label: 'anon' },
                      { value: 'authenticated', label: 'authenticated' },
                      { value: 'service_role', label: 'service_role' },
                    ]}
                    w={130}
                    leftSection={<Text size="xs" c="dimmed">Role</Text>}
                    styles={{ input: { fontSize: 12, paddingLeft: 40 } }}
                  />
                  <Tooltip label="Run with EXPLAIN">
                    <ActionIcon
                      variant="subtle"
                      size="sm"
                      onClick={() => executeQuery(true)}
                      loading={activeTab.loading}
                    >
                      <IconFileAnalytics size={16} />
                    </ActionIcon>
                  </Tooltip>
                  <Button
                    size="xs"
                    onClick={() => executeQuery(false)}
                    loading={activeTab.loading}
                    leftSection={<IconPlayerPlay size={14} />}
                    rightSection={
                      <Group gap={2}>
                        <Kbd size="xs">⌘</Kbd>
                        <Kbd size="xs">↵</Kbd>
                      </Group>
                    }
                  >
                    Run
                  </Button>
                </Group>
              </Group>
            </Box>

            {/* Results Content */}
            {!resultsPanelCollapsed && (
              <ScrollArea style={{ flex: 1 }}>
                {activeTab.loading ? (
                  <Center py="xl">
                    <Loader size="sm" />
                  </Center>
                ) : activeTab.error ? (
                  <Box p="md">
                    <Paper
                      p="md"
                      style={{
                        backgroundColor: 'rgba(239, 68, 68, 0.1)',
                        border: '1px solid rgba(239, 68, 68, 0.3)',
                      }}
                    >
                      <Text size="sm" c="red">
                        {activeTab.error}
                      </Text>
                    </Paper>
                  </Box>
                ) : resultTab === 'results' ? (
                  activeTab.result ? (
                    activeTab.result.rows.length > 0 ? (
                      <Table striped highlightOnHover>
                        <Table.Thead>
                          <Table.Tr>
                            {activeTab.result.columns.map((col) => (
                              <Table.Th key={col}>{col}</Table.Th>
                            ))}
                          </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                          {activeTab.result.rows.map((row, i) => (
                            <Table.Tr key={i}>
                              {activeTab.result!.columns.map((col) => (
                                <Table.Td key={col}>
                                  <Text size="sm" style={{ maxWidth: 300, fontFamily: 'monospace' }} truncate>
                                    {row[col] === null ? (
                                      <Text span c="dimmed" fs="italic">NULL</Text>
                                    ) : typeof row[col] === 'object' ? (
                                      JSON.stringify(row[col])
                                    ) : (
                                      String(row[col])
                                    )}
                                  </Text>
                                </Table.Td>
                              ))}
                            </Table.Tr>
                          ))}
                        </Table.Tbody>
                      </Table>
                    ) : (
                      <Box p="xl" ta="center">
                        <Text c="dimmed">Query executed successfully. No rows returned.</Text>
                      </Box>
                    )
                  ) : (
                    <Box p="xl" ta="center">
                      <Text c="dimmed">Click Run to execute your query.</Text>
                    </Box>
                  )
                ) : resultTab === 'explain' ? (
                  activeTab.result ? (
                    <Box p="md">
                      <Paper p="md" withBorder>
                        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', fontSize: 12 }}>
                          {JSON.stringify(activeTab.result.rows, null, 2)}
                        </pre>
                      </Paper>
                    </Box>
                  ) : (
                    <Box p="xl" ta="center">
                      <IconFileAnalytics size={32} style={{ opacity: 0.3 }} />
                      <Text c="dimmed" mt="sm">
                        Click the Explain button to see the query plan
                      </Text>
                    </Box>
                  )
                ) : (
                  <Box p="xl" ta="center">
                    <IconChartBar size={32} style={{ opacity: 0.3 }} />
                    <Text c="dimmed" mt="sm">
                      Chart visualization coming soon
                    </Text>
                  </Box>
                )}
              </ScrollArea>
            )}
          </Box>
        </Box>
      </Box>

      {/* Save Query Modal */}
      <Modal
        opened={saveModalOpened}
        onClose={() => setSaveModalOpened(false)}
        title="Save query"
        size="sm"
      >
        <Stack gap="md">
          <TextInput
            label="Query name"
            placeholder="My query"
            value={newQueryName}
            onChange={(e) => setNewQueryName(e.target.value)}
            autoFocus
          />
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setSaveModalOpened(false)}>
              Cancel
            </Button>
            <Button onClick={handleSaveQuery}>Save</Button>
          </Group>
        </Stack>
      </Modal>

      {/* Settings Modal */}
      <Modal
        opened={settingsOpened}
        onClose={() => setSettingsOpened(false)}
        title="Editor Settings"
        size="sm"
      >
        <Stack gap="md">
          <Box>
            <Text size="sm" fw={500} mb="xs">Font Size</Text>
            <Slider
              value={fontSize}
              onChange={setFontSize}
              min={10}
              max={24}
              step={1}
              marks={[
                { value: 10, label: '10' },
                { value: 14, label: '14' },
                { value: 18, label: '18' },
                { value: 24, label: '24' },
              ]}
            />
          </Box>
          <Switch
            label="Word wrap"
            checked={wordWrap}
            onChange={(e) => setWordWrap(e.currentTarget.checked)}
          />
          <Switch
            label="Show minimap"
            checked={minimap}
            onChange={(e) => setMinimap(e.currentTarget.checked)}
          />
        </Stack>
      </Modal>

      {/* AI Assistant Modal */}
      <Modal
        opened={aiModalOpened}
        onClose={() => setAiModalOpened(false)}
        title={
          <Group gap="xs">
            <ThemeIcon size="sm" variant="light" color="violet">
              <IconSparkles size={14} />
            </ThemeIcon>
            <Text>AI Assistant</Text>
          </Group>
        }
        size="md"
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            Describe what you want to query and AI will generate the SQL for you.
          </Text>
          <Textarea
            placeholder="e.g., Select all users who signed up in the last 7 days"
            value={aiPrompt}
            onChange={(e) => setAiPrompt(e.target.value)}
            minRows={3}
            autoFocus
          />
          <Text size="xs" c="dimmed">
            Examples:
          </Text>
          <Stack gap={4}>
            {[
              'Select all active users',
              'Count orders by status',
              'Find products with low inventory',
              'List users who haven\'t logged in for 30 days',
            ].map((example) => (
              <UnstyledButton
                key={example}
                onClick={() => setAiPrompt(example)}
                style={{ padding: '4px 8px', borderRadius: 4 }}
              >
                <Text size="xs" c="blue">{example}</Text>
              </UnstyledButton>
            ))}
          </Stack>
          <Group justify="flex-end">
            <Button variant="subtle" onClick={() => setAiModalOpened(false)}>
              Cancel
            </Button>
            <Button
              leftSection={<IconWand size={14} />}
              onClick={() => {
                notifications.show({
                  title: 'Coming Soon',
                  message: 'AI query generation is not yet implemented',
                  color: 'blue',
                });
              }}
            >
              Generate SQL
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Query History Drawer */}
      <Drawer
        opened={historyDrawerOpened}
        onClose={() => setHistoryDrawerOpened(false)}
        title="Query History"
        position="right"
        size="md"
      >
        <Stack gap="md">
          <Group justify="space-between">
            <TextInput
              placeholder="Search history..."
              size="xs"
              leftSection={<IconSearch size={14} />}
              style={{ flex: 1 }}
            />
            <Button
              size="xs"
              variant="subtle"
              color="red"
              onClick={async () => {
                await databaseApi.clearQueryHistory();
                setQueryHistory([]);
                notifications.show({ message: 'History cleared', color: 'green' });
              }}
            >
              Clear
            </Button>
          </Group>

          {historyLoading ? (
            <Center py="xl">
              <Loader size="sm" />
            </Center>
          ) : queryHistory.length === 0 ? (
            <Center py="xl">
              <Text c="dimmed">No query history yet</Text>
            </Center>
          ) : (
            <ScrollArea style={{ height: 'calc(100vh - 200px)' }}>
              <Stack gap="xs">
                {queryHistory.map((entry) => (
                  <Paper
                    key={entry.id}
                    p="sm"
                    withBorder
                    style={{ cursor: 'pointer' }}
                    onClick={() => {
                      loadQuery('History query', entry.query);
                      setHistoryDrawerOpened(false);
                    }}
                  >
                    <Group justify="space-between" mb={4}>
                      <Group gap="xs">
                        {entry.success ? (
                          <ThemeIcon size="xs" color="green" variant="light">
                            <IconCircleCheck size={12} />
                          </ThemeIcon>
                        ) : (
                          <ThemeIcon size="xs" color="red" variant="light">
                            <IconCircleX size={12} />
                          </ThemeIcon>
                        )}
                        <Badge size="xs" variant="light">
                          {entry.role}
                        </Badge>
                      </Group>
                      <Text size="xs" c="dimmed">
                        {formatDate(entry.executed_at)}
                      </Text>
                    </Group>
                    <Text size="xs" lineClamp={2} style={{ fontFamily: 'monospace' }}>
                      {entry.query}
                    </Text>
                    <Group gap="xs" mt={4}>
                      <Text size="xs" c="dimmed">
                        {formatDuration(entry.duration_ms)}
                      </Text>
                      {entry.success && (
                        <Text size="xs" c="dimmed">
                          • {entry.row_count} rows
                        </Text>
                      )}
                      {entry.error && (
                        <Text size="xs" c="red" lineClamp={1}>
                          {entry.error}
                        </Text>
                      )}
                    </Group>
                  </Paper>
                ))}
              </Stack>
            </ScrollArea>
          )}
        </Stack>
      </Drawer>
    </PageContainer>
  );
}

// Query item component
function QueryItem({
  query,
  isFavorite,
  onLoad,
  onLoadNewTab,
  onToggleFavorite,
  onDelete,
}: {
  query: { id: string; name: string; query: string };
  isFavorite: boolean;
  onLoad: () => void;
  onLoadNewTab: () => void;
  onToggleFavorite: () => void;
  onDelete: () => void;
}) {
  return (
    <Group
      gap={4}
      wrap="nowrap"
      style={{
        padding: '4px 8px',
        borderRadius: 4,
        cursor: 'pointer',
      }}
      className="query-item"
    >
      <IconFile size={14} style={{ flexShrink: 0 }} />
      <UnstyledButton onClick={onLoad} style={{ flex: 1, minWidth: 0 }}>
        <Text size="sm" truncate>
          {query.name}
        </Text>
      </UnstyledButton>
      <Menu position="right-start" withinPortal>
        <Menu.Target>
          <ActionIcon
            size="xs"
            variant="subtle"
            onClick={(e) => e.stopPropagation()}
          >
            <IconDotsVertical size={12} />
          </ActionIcon>
        </Menu.Target>
        <Menu.Dropdown>
          <Menu.Item
            leftSection={<IconFolder size={14} />}
            onClick={onLoad}
          >
            Open
          </Menu.Item>
          <Menu.Item
            leftSection={<IconPlus size={14} />}
            onClick={onLoadNewTab}
          >
            Open in new tab
          </Menu.Item>
          <Menu.Item
            leftSection={
              isFavorite ? <IconStarFilled size={14} /> : <IconStar size={14} />
            }
            onClick={onToggleFavorite}
          >
            {isFavorite ? 'Remove from favorites' : 'Add to favorites'}
          </Menu.Item>
          <Menu.Divider />
          <Menu.Item
            leftSection={<IconTrash size={14} />}
            color="red"
            onClick={onDelete}
          >
            Delete
          </Menu.Item>
        </Menu.Dropdown>
      </Menu>
    </Group>
  );
}
