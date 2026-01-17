import { useState, useCallback } from 'react';
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
} from '@mantine/core';
import {
  IconPlayerPlay,
  IconTrash,
  IconClock,
  IconCode,
  IconDownload,
} from '@tabler/icons-react';
import Editor from '@monaco-editor/react';
import { notifications } from '@mantine/notifications';
import { PageContainer } from '../../components/layout/PageContainer';
import { databaseApi } from '../../api';
import { useAppStore } from '../../stores/appStore';
import type { QueryResult } from '../../types';

export function SQLEditorPage() {
  const { savedQueries, addSavedQuery, removeSavedQuery } = useAppStore();

  const [query, setQuery] = useState('SELECT * FROM auth.users LIMIT 10;');
  const [result, setResult] = useState<QueryResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const executeQuery = useCallback(async () => {
    if (!query.trim()) {
      notifications.show({
        title: 'Error',
        message: 'Please enter a query',
        color: 'red',
      });
      return;
    }

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const data = await databaseApi.executeQuery(query);
      setResult(data);
    } catch (err: any) {
      setError(err.message || 'Query execution failed');
      notifications.show({
        title: 'Query Error',
        message: err.message || 'Query execution failed',
        color: 'red',
      });
    } finally {
      setLoading(false);
    }
  }, [query]);

  const handleSaveQuery = () => {
    const name = prompt('Enter a name for this query:');
    if (name) {
      addSavedQuery(name, query);
      notifications.show({
        title: 'Saved',
        message: 'Query saved successfully',
        color: 'green',
      });
    }
  };

  const loadQuery = (savedQuery: string) => {
    setQuery(savedQuery);
  };

  const exportResults = () => {
    if (!result) return;

    const csv = [
      result.columns.join(','),
      ...result.rows.map((row) =>
        result.columns.map((col) => JSON.stringify(row[col] ?? '')).join(',')
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

  return (
    <PageContainer title="SQL Editor" description="Run SQL queries on your database" fullWidth noPadding>
      <Box style={{ display: 'flex', height: 'calc(100vh - 140px)' }}>
        {/* Main Editor Area */}
        <Box style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {/* Editor */}
          <Box style={{ flex: '0 0 40%', borderBottom: '1px solid var(--supabase-border)' }}>
            <Box
              p="sm"
              style={{
                borderBottom: '1px solid var(--supabase-border)',
                backgroundColor: 'var(--supabase-bg-sidebar)',
              }}
            >
              <Group justify="space-between">
                <Group gap="xs">
                  <IconCode size={18} />
                  <Text size="sm" fw={500}>
                    Query
                  </Text>
                </Group>
                <Group gap="sm">
                  <Button variant="subtle" size="xs" onClick={handleSaveQuery}>
                    Save query
                  </Button>
                  <Button
                    leftSection={<IconPlayerPlay size={14} />}
                    size="xs"
                    onClick={executeQuery}
                    loading={loading}
                  >
                    Run (Ctrl+Enter)
                  </Button>
                </Group>
              </Group>
            </Box>
            <Editor
              height="calc(100% - 50px)"
              defaultLanguage="sql"
              value={query}
              onChange={(value) => setQuery(value || '')}
              theme="vs-light"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
                lineNumbers: 'on',
                scrollBeyondLastLine: false,
                automaticLayout: true,
                tabSize: 2,
                wordWrap: 'on',
              }}
              onMount={(editor) => {
                editor.addCommand(
                  // Ctrl/Cmd + Enter
                  2048 | 3, // KeyMod.CtrlCmd | KeyCode.Enter
                  () => executeQuery()
                );
              }}
            />
          </Box>

          {/* Results */}
          <Box style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            <Box
              p="sm"
              style={{
                borderBottom: '1px solid var(--supabase-border)',
                backgroundColor: 'var(--supabase-bg-sidebar)',
              }}
            >
              <Group justify="space-between">
                <Group gap="xs">
                  <Text size="sm" fw={500}>
                    Results
                  </Text>
                  {result && (
                    <Badge size="sm" variant="light" color="green">
                      {result.row_count} rows in {result.duration_ms.toFixed(2)}ms
                    </Badge>
                  )}
                </Group>
                {result && result.rows.length > 0 && (
                  <Tooltip label="Export as CSV">
                    <ActionIcon variant="subtle" size="sm" onClick={exportResults}>
                      <IconDownload size={16} />
                    </ActionIcon>
                  </Tooltip>
                )}
              </Group>
            </Box>

            <ScrollArea style={{ flex: 1 }}>
              {error ? (
                <Box p="md">
                  <Paper
                    p="md"
                    style={{
                      backgroundColor: 'rgba(239, 68, 68, 0.1)',
                      border: '1px solid rgba(239, 68, 68, 0.3)',
                    }}
                  >
                    <Text size="sm" c="red">
                      {error}
                    </Text>
                  </Paper>
                </Box>
              ) : result ? (
                result.rows.length > 0 ? (
                  <Table striped highlightOnHover>
                    <Table.Thead>
                      <Table.Tr>
                        {result.columns.map((col) => (
                          <Table.Th key={col}>{col}</Table.Th>
                        ))}
                      </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                      {result.rows.map((row, i) => (
                        <Table.Tr key={i}>
                          {result.columns.map((col) => (
                            <Table.Td key={col}>
                              <Text size="sm" style={{ maxWidth: 300 }} truncate>
                                {row[col] === null
                                  ? 'NULL'
                                  : typeof row[col] === 'object'
                                    ? JSON.stringify(row[col])
                                    : String(row[col])}
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
                  <Text c="dimmed">Run a query to see results</Text>
                </Box>
              )}
            </ScrollArea>
          </Box>
        </Box>

        {/* Saved Queries Sidebar */}
        <Box
          style={{
            width: 280,
            borderLeft: '1px solid var(--supabase-border)',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Box
            p="md"
            style={{
              borderBottom: '1px solid var(--supabase-border)',
            }}
          >
            <Group gap="xs">
              <IconClock size={18} />
              <Text size="sm" fw={500}>
                Saved Queries
              </Text>
            </Group>
          </Box>

          <ScrollArea style={{ flex: 1 }} p="sm">
            {savedQueries.length === 0 ? (
              <Text size="sm" c="dimmed" ta="center" py="xl">
                No saved queries
              </Text>
            ) : (
              <Stack gap="xs">
                {savedQueries.map((sq) => (
                  <Paper
                    key={sq.id}
                    p="sm"
                    style={{
                      cursor: 'pointer',
                      border: '1px solid var(--supabase-border)',
                    }}
                    onClick={() => loadQuery(sq.query)}
                  >
                    <Group justify="space-between" wrap="nowrap">
                      <Box style={{ overflow: 'hidden' }}>
                        <Text size="sm" fw={500} truncate>
                          {sq.name}
                        </Text>
                        <Text size="xs" c="dimmed" truncate>
                          {sq.query.slice(0, 50)}...
                        </Text>
                      </Box>
                      <ActionIcon
                        variant="subtle"
                        color="red"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          removeSavedQuery(sq.id);
                        }}
                      >
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Paper>
                ))}
              </Stack>
            )}
          </ScrollArea>
        </Box>
      </Box>
    </PageContainer>
  );
}
