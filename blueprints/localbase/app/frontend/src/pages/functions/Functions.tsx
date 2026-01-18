import { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Title,
  Text,
  Button,
  Group,
  Stack,
  Paper,
  Badge,
  ActionIcon,
  TextInput,
  Tabs,
  Modal,
  Select,
  Switch,
  ScrollArea,
  Loader,
  Menu,
  Tooltip,
  Textarea,
  SegmentedControl,
  Code,
  Divider,
  ThemeIcon,
  SimpleGrid,
  Card,
  CopyButton,
  Skeleton,
  Collapse,
} from '@mantine/core';
import {
  IconPlus,
  IconCode,
  IconPlayerPlay,
  IconRocket,
  IconTrash,
  IconSettings,
  IconHistory,
  IconChartBar,
  IconLock,
  IconLockOpen,
  IconDownload,
  IconDotsVertical,
  IconRefresh,
  IconCheck,
  IconX,
  IconCopy,
  IconBolt,
  IconBrandStripe,
  IconRobot,
  IconMail,
  IconUpload,
  IconShield,
  IconClock,
  IconDatabase,
  IconKey,
  IconFileCode,
  IconTerminal2,
} from '@tabler/icons-react';
import Editor from '@monaco-editor/react';
import { functionsApi } from '../../api/functions';
import type {
  EdgeFunction,
  Deployment,
  FunctionLog,
  FunctionMetrics,
  FunctionTemplate,
  Secret,
} from '../../types';
import { AreaChart, Area, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer, CartesianGrid } from 'recharts';

// Template icons mapping
const templateIcons: Record<string, React.ReactNode> = {
  'wave': <IconBolt size={20} />,
  'credit-card': <IconBrandStripe size={20} />,
  'robot': <IconRobot size={20} />,
  'mail': <IconMail size={20} />,
  'upload': <IconUpload size={20} />,
  'shield': <IconShield size={20} />,
  'clock': <IconClock size={20} />,
  'database': <IconDatabase size={20} />,
};

export function FunctionsPage() {
  // State
  const [functions, setFunctions] = useState<EdgeFunction[]>([]);
  const [selectedFunctionId, setSelectedFunctionId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [sourceCode, setSourceCode] = useState('');
  const [originalSourceCode, setOriginalSourceCode] = useState('');
  const [isDirty, setIsDirty] = useState(false);
  const [isDeploying, setIsDeploying] = useState(false);
  const [activeTab, setActiveTab] = useState<string | null>('code');

  // Secrets
  const [secrets, setSecrets] = useState<Secret[]>([]);
  const [secretsModalOpen, setSecretsModalOpen] = useState(false);
  const [newSecretName, setNewSecretName] = useState('');
  const [newSecretValue, setNewSecretValue] = useState('');
  const [bulkSecretsMode, setBulkSecretsMode] = useState(false);
  const [bulkSecretsText, setBulkSecretsText] = useState('');

  // Create function modal
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [newFunctionName, setNewFunctionName] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState<string | null>(null);
  const [verifyJwt, setVerifyJwt] = useState(true);
  const [templates, setTemplates] = useState<FunctionTemplate[]>([]);

  // Testing
  const [testPanelOpen, setTestPanelOpen] = useState(false);
  const [testMethod, setTestMethod] = useState('POST');
  const [testHeaders, setTestHeaders] = useState<Array<{ key: string; value: string }>>([
    { key: 'Content-Type', value: 'application/json' },
  ]);
  const [testBody, setTestBody] = useState('{\n  "name": "World"\n}');
  const [testResponse, setTestResponse] = useState<any>(null);
  const [isTesting, setIsTesting] = useState(false);

  // Logs and metrics
  const [logs, setLogs] = useState<FunctionLog[]>([]);
  const [logsLoading, setLogsLoading] = useState(false);
  const [metrics, setMetrics] = useState<FunctionMetrics | null>(null);
  const [metricsLoading, setMetricsLoading] = useState(false);
  const [metricsPeriod, setMetricsPeriod] = useState('24h');

  // Deployments
  const [deployments, setDeployments] = useState<Deployment[]>([]);
  const [deploymentsLoading, setDeploymentsLoading] = useState(false);

  const selectedFunction = functions.find(f => f.id === selectedFunctionId);

  // Load functions
  const loadFunctions = useCallback(async () => {
    try {
      const data = await functionsApi.listFunctions();
      setFunctions(data);
      if (data.length > 0 && !selectedFunctionId) {
        setSelectedFunctionId(data[0].id);
      }
    } catch (error) {
      console.error('Failed to load functions:', error);
    } finally {
      setLoading(false);
    }
  }, [selectedFunctionId]);

  // Load source code when function is selected
  const loadSourceCode = useCallback(async (id: string) => {
    try {
      const source = await functionsApi.getSource(id);
      setSourceCode(source.source_code);
      setOriginalSourceCode(source.source_code);
      setIsDirty(source.is_draft);
    } catch (error) {
      console.error('Failed to load source:', error);
    }
  }, []);

  // Load logs
  const loadLogs = useCallback(async (id: string) => {
    setLogsLoading(true);
    try {
      const data = await functionsApi.getLogs(id, { limit: 100 });
      setLogs(data.logs || []);
    } catch (error) {
      console.error('Failed to load logs:', error);
    } finally {
      setLogsLoading(false);
    }
  }, []);

  // Load metrics
  const loadMetrics = useCallback(async (id: string, period: string) => {
    setMetricsLoading(true);
    try {
      const data = await functionsApi.getMetrics(id, period);
      setMetrics(data);
    } catch (error) {
      console.error('Failed to load metrics:', error);
    } finally {
      setMetricsLoading(false);
    }
  }, []);

  // Load deployments
  const loadDeployments = useCallback(async (id: string) => {
    setDeploymentsLoading(true);
    try {
      const data = await functionsApi.listDeployments(id);
      setDeployments(data || []);
    } catch (error) {
      console.error('Failed to load deployments:', error);
    } finally {
      setDeploymentsLoading(false);
    }
  }, []);

  // Load templates
  const loadTemplates = useCallback(async () => {
    try {
      const data = await functionsApi.listTemplates();
      setTemplates(data.templates || []);
    } catch (error) {
      console.error('Failed to load templates:', error);
    }
  }, []);

  // Load secrets
  const loadSecrets = useCallback(async () => {
    try {
      const data = await functionsApi.listSecrets();
      setSecrets(data || []);
    } catch (error) {
      console.error('Failed to load secrets:', error);
    }
  }, []);

  // Initial load
  useEffect(() => {
    loadFunctions();
    loadTemplates();
    loadSecrets();
  }, [loadFunctions, loadTemplates, loadSecrets]);

  // Load data when function is selected
  useEffect(() => {
    if (selectedFunctionId) {
      loadSourceCode(selectedFunctionId);
      if (activeTab === 'logs') {
        loadLogs(selectedFunctionId);
      } else if (activeTab === 'metrics') {
        loadMetrics(selectedFunctionId, metricsPeriod);
      } else if (activeTab === 'deployments') {
        loadDeployments(selectedFunctionId);
      }
    }
  }, [selectedFunctionId, activeTab, metricsPeriod, loadSourceCode, loadLogs, loadMetrics, loadDeployments]);

  // Handle code change
  const handleCodeChange = useCallback((value: string | undefined) => {
    if (value !== undefined) {
      setSourceCode(value);
      setIsDirty(value !== originalSourceCode);
    }
  }, [originalSourceCode]);

  // Deploy function
  const handleDeploy = useCallback(async () => {
    if (!selectedFunctionId) return;
    setIsDeploying(true);
    try {
      await functionsApi.deployFunction(selectedFunctionId, {
        source_code: sourceCode,
      });
      setOriginalSourceCode(sourceCode);
      setIsDirty(false);
      await loadFunctions();
      await loadDeployments(selectedFunctionId);
    } catch (error) {
      console.error('Failed to deploy:', error);
    } finally {
      setIsDeploying(false);
    }
  }, [selectedFunctionId, sourceCode, loadFunctions, loadDeployments]);

  // Create function
  const handleCreateFunction = useCallback(async () => {
    if (!newFunctionName.trim()) return;

    try {
      const fn = await functionsApi.createFunction({
        name: newFunctionName.trim().replace(/\s+/g, '-').toLowerCase(),
        verify_jwt: verifyJwt,
        template_id: selectedTemplate || undefined,
      });

      setCreateModalOpen(false);
      setNewFunctionName('');
      setSelectedTemplate(null);
      setVerifyJwt(true);

      await loadFunctions();
      setSelectedFunctionId(fn.id);
    } catch (error) {
      console.error('Failed to create function:', error);
    }
  }, [newFunctionName, verifyJwt, selectedTemplate, loadFunctions]);

  // Delete function
  const handleDeleteFunction = useCallback(async (id: string) => {
    if (!confirm('Are you sure you want to delete this function?')) return;

    try {
      await functionsApi.deleteFunction(id);
      if (selectedFunctionId === id) {
        setSelectedFunctionId(null);
      }
      await loadFunctions();
    } catch (error) {
      console.error('Failed to delete function:', error);
    }
  }, [selectedFunctionId, loadFunctions]);

  // Test function
  const handleTestFunction = useCallback(async () => {
    if (!selectedFunctionId) return;
    setIsTesting(true);
    setTestResponse(null);

    try {
      const headers: Record<string, string> = {};
      testHeaders.forEach(h => {
        if (h.key && h.value) {
          headers[h.key] = h.value;
        }
      });

      let body;
      try {
        body = JSON.parse(testBody);
      } catch {
        body = testBody;
      }

      const response = await functionsApi.testFunction(selectedFunctionId, {
        method: testMethod,
        path: '/',
        headers,
        body,
      });

      setTestResponse(response);
    } catch (error) {
      console.error('Test failed:', error);
      setTestResponse({ error: 'Test failed' });
    } finally {
      setIsTesting(false);
    }
  }, [selectedFunctionId, testMethod, testHeaders, testBody]);

  // Add secret
  const handleAddSecret = useCallback(async () => {
    if (!newSecretName.trim() || !newSecretValue.trim()) return;

    try {
      await functionsApi.createSecret({
        name: newSecretName.trim(),
        value: newSecretValue.trim(),
      });
      setNewSecretName('');
      setNewSecretValue('');
      await loadSecrets();
    } catch (error) {
      console.error('Failed to create secret:', error);
    }
  }, [newSecretName, newSecretValue, loadSecrets]);

  // Delete secret
  const handleDeleteSecret = useCallback(async (name: string) => {
    try {
      await functionsApi.deleteSecret(name);
      await loadSecrets();
    } catch (error) {
      console.error('Failed to delete secret:', error);
    }
  }, [loadSecrets]);

  // Bulk update secrets
  const handleBulkUpdateSecrets = useCallback(async () => {
    const lines = bulkSecretsText.split('\n').filter(line => line.trim());
    const secretsList: Array<{ name: string; value: string }> = [];

    for (const line of lines) {
      const [name, ...valueParts] = line.split('=');
      if (name && valueParts.length > 0) {
        secretsList.push({
          name: name.trim(),
          value: valueParts.join('=').trim(),
        });
      }
    }

    if (secretsList.length === 0) return;

    try {
      await functionsApi.bulkUpdateSecrets({ secrets: secretsList });
      setBulkSecretsText('');
      setBulkSecretsMode(false);
      await loadSecrets();
    } catch (error) {
      console.error('Failed to bulk update secrets:', error);
    }
  }, [bulkSecretsText, loadSecrets]);

  // Download function
  const handleDownload = useCallback(async () => {
    if (!selectedFunctionId || !selectedFunction) return;

    try {
      const content = await functionsApi.downloadFunction(selectedFunctionId);
      const blob = new Blob([content], { type: 'application/typescript' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${selectedFunction.slug}.ts`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Failed to download function:', error);
    }
  }, [selectedFunctionId, selectedFunction]);

  if (loading) {
    return (
      <Stack align="center" justify="center" h={400}>
        <Loader size="lg" />
        <Text c="dimmed">Loading functions...</Text>
      </Stack>
    );
  }

  return (
    <Box h="calc(100vh - 60px)">
      <Group justify="space-between" mb="md">
        <Title order={3}>Edge Functions</Title>
        <Group>
          <Button
            variant="subtle"
            leftSection={<IconKey size={16} />}
            onClick={() => setSecretsModalOpen(true)}
          >
            Manage Secrets
          </Button>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setCreateModalOpen(true)}
          >
            Create function
          </Button>
        </Group>
      </Group>

      <Group align="flex-start" gap="md" wrap="nowrap" style={{ height: 'calc(100% - 50px)' }}>
        {/* Function List Sidebar */}
        <Paper
          withBorder
          p="xs"
          w={260}
          h="100%"
          style={{ flexShrink: 0, display: 'flex', flexDirection: 'column' }}
        >
          <Text size="xs" fw={600} c="dimmed" mb="xs" px="xs">
            FUNCTIONS
          </Text>
          <ScrollArea style={{ flex: 1 }}>
            <Stack gap={2}>
              {functions.length === 0 ? (
                <Text size="sm" c="dimmed" ta="center" py="lg">
                  No functions yet
                </Text>
              ) : (
                functions.map(fn => (
                  <Paper
                    key={fn.id}
                    p="xs"
                    withBorder={selectedFunctionId === fn.id}
                    bg={selectedFunctionId === fn.id ? 'var(--mantine-color-dark-6)' : 'transparent'}
                    style={{ cursor: 'pointer' }}
                    onClick={() => setSelectedFunctionId(fn.id)}
                  >
                    <Group gap="xs" wrap="nowrap">
                      <ThemeIcon
                        size="xs"
                        radius="xl"
                        color={fn.status === 'active' ? 'green' : 'gray'}
                        variant="filled"
                      >
                        <Box w={6} h={6} style={{ borderRadius: '50%', background: 'currentColor' }} />
                      </ThemeIcon>
                      <Box style={{ flex: 1, minWidth: 0 }}>
                        <Text size="sm" fw={500} truncate>
                          {fn.name}
                        </Text>
                        <Text size="xs" c="dimmed">
                          v{fn.version}
                        </Text>
                      </Box>
                      {fn.verify_jwt ? (
                        <Tooltip label="JWT Required">
                          <IconLock size={14} style={{ opacity: 0.5 }} />
                        </Tooltip>
                      ) : (
                        <Tooltip label="Public">
                          <IconLockOpen size={14} style={{ opacity: 0.5 }} />
                        </Tooltip>
                      )}
                    </Group>
                  </Paper>
                ))
              )}
            </Stack>
          </ScrollArea>

          <Divider my="sm" />

          <Text size="xs" fw={600} c="dimmed" mb="xs" px="xs">
            SECRETS
          </Text>
          <Stack gap={2}>
            {secrets.slice(0, 3).map(secret => (
              <Text key={secret.id} size="xs" c="dimmed" px="xs">
                {secret.name}
              </Text>
            ))}
            {secrets.length > 3 && (
              <Text size="xs" c="dimmed" px="xs">
                +{secrets.length - 3} more
              </Text>
            )}
            <Button
              variant="subtle"
              size="xs"
              onClick={() => setSecretsModalOpen(true)}
              fullWidth
            >
              Manage secrets
            </Button>
          </Stack>
        </Paper>

        {/* Main Content Area */}
        {selectedFunction ? (
          <Paper withBorder p={0} style={{ flex: 1, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
            {/* Header */}
            <Group justify="space-between" p="sm" style={{ borderBottom: '1px solid var(--mantine-color-dark-4)' }}>
              <Group>
                <Title order={4}>{selectedFunction.name}</Title>
                <Badge
                  color={selectedFunction.status === 'active' ? 'green' : 'gray'}
                  variant="light"
                >
                  {selectedFunction.status}
                </Badge>
                {isDirty && (
                  <Badge color="yellow" variant="light">
                    Unsaved changes
                  </Badge>
                )}
              </Group>
              <Group>
                <Button
                  variant="subtle"
                  leftSection={<IconPlayerPlay size={16} />}
                  onClick={() => setTestPanelOpen(!testPanelOpen)}
                >
                  Test
                </Button>
                <Button
                  leftSection={<IconRocket size={16} />}
                  loading={isDeploying}
                  onClick={handleDeploy}
                  disabled={!isDirty && sourceCode === originalSourceCode}
                >
                  Deploy
                </Button>
                <Menu shadow="md" width={200}>
                  <Menu.Target>
                    <ActionIcon variant="subtle">
                      <IconDotsVertical size={16} />
                    </ActionIcon>
                  </Menu.Target>
                  <Menu.Dropdown>
                    <Menu.Item
                      leftSection={<IconDownload size={16} />}
                      onClick={handleDownload}
                    >
                      Download
                    </Menu.Item>
                    <Menu.Divider />
                    <Menu.Item
                      leftSection={<IconTrash size={16} />}
                      color="red"
                      onClick={() => handleDeleteFunction(selectedFunction.id)}
                    >
                      Delete function
                    </Menu.Item>
                  </Menu.Dropdown>
                </Menu>
              </Group>
            </Group>

            {/* Tabs */}
            <Tabs value={activeTab} onChange={setActiveTab} style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <Tabs.List px="sm">
                <Tabs.Tab value="code" leftSection={<IconCode size={14} />}>
                  Code
                </Tabs.Tab>
                <Tabs.Tab value="logs" leftSection={<IconTerminal2 size={14} />}>
                  Logs
                </Tabs.Tab>
                <Tabs.Tab value="metrics" leftSection={<IconChartBar size={14} />}>
                  Metrics
                </Tabs.Tab>
                <Tabs.Tab value="deployments" leftSection={<IconHistory size={14} />}>
                  Deployments
                </Tabs.Tab>
                <Tabs.Tab value="settings" leftSection={<IconSettings size={14} />}>
                  Settings
                </Tabs.Tab>
              </Tabs.List>

              <Tabs.Panel value="code" style={{ flex: 1, position: 'relative' }}>
                <Box style={{ position: 'absolute', inset: 0, display: 'flex' }}>
                  <Box style={{ flex: 1, position: 'relative' }}>
                    <Editor
                      height="100%"
                      language="typescript"
                      theme="vs-dark"
                      value={sourceCode}
                      onChange={handleCodeChange}
                      options={{
                        minimap: { enabled: false },
                        fontSize: 13,
                        lineNumbers: 'on',
                        scrollBeyondLastLine: false,
                        wordWrap: 'on',
                        automaticLayout: true,
                        padding: { top: 10 },
                      }}
                    />
                  </Box>

                  {/* Test Panel (collapsible) */}
                  <Collapse in={testPanelOpen} transitionDuration={200}>
                    <Paper
                      w={400}
                      h="100%"
                      p="md"
                      style={{
                        borderLeft: '1px solid var(--mantine-color-dark-4)',
                        display: 'flex',
                        flexDirection: 'column',
                      }}
                    >
                      <Group justify="space-between" mb="md">
                        <Text fw={600}>Test Function</Text>
                        <ActionIcon variant="subtle" onClick={() => setTestPanelOpen(false)}>
                          <IconX size={16} />
                        </ActionIcon>
                      </Group>

                      <Stack gap="sm" style={{ flex: 1, overflow: 'auto' }}>
                        <Group gap="xs">
                          <Select
                            w={100}
                            value={testMethod}
                            onChange={(v) => setTestMethod(v || 'POST')}
                            data={['GET', 'POST', 'PUT', 'PATCH', 'DELETE']}
                          />
                          <TextInput
                            flex={1}
                            value={`/functions/v1/${selectedFunction.slug}`}
                            readOnly
                          />
                        </Group>

                        <Divider label="Headers" labelPosition="left" />

                        {testHeaders.map((header, idx) => (
                          <Group key={idx} gap="xs">
                            <TextInput
                              placeholder="Key"
                              value={header.key}
                              onChange={(e) => {
                                const newHeaders = [...testHeaders];
                                newHeaders[idx].key = e.target.value;
                                setTestHeaders(newHeaders);
                              }}
                              style={{ flex: 1 }}
                            />
                            <TextInput
                              placeholder="Value"
                              value={header.value}
                              onChange={(e) => {
                                const newHeaders = [...testHeaders];
                                newHeaders[idx].value = e.target.value;
                                setTestHeaders(newHeaders);
                              }}
                              style={{ flex: 1 }}
                            />
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              onClick={() => {
                                setTestHeaders(testHeaders.filter((_, i) => i !== idx));
                              }}
                            >
                              <IconX size={14} />
                            </ActionIcon>
                          </Group>
                        ))}
                        <Button
                          variant="subtle"
                          size="xs"
                          leftSection={<IconPlus size={14} />}
                          onClick={() => setTestHeaders([...testHeaders, { key: '', value: '' }])}
                        >
                          Add header
                        </Button>

                        <Divider label="Body" labelPosition="left" />

                        <Textarea
                          value={testBody}
                          onChange={(e) => setTestBody(e.target.value)}
                          minRows={4}
                          styles={{ input: { fontFamily: 'monospace', fontSize: 12 } }}
                        />

                        <Button
                          fullWidth
                          leftSection={<IconPlayerPlay size={16} />}
                          loading={isTesting}
                          onClick={handleTestFunction}
                        >
                          Run function
                        </Button>

                        {testResponse && (
                          <>
                            <Divider label="Response" labelPosition="left" />
                            <Paper p="xs" withBorder bg="dark.8">
                              <Group gap="xs" mb="xs">
                                <Badge
                                  color={testResponse.status < 400 ? 'green' : 'red'}
                                  variant="light"
                                >
                                  {testResponse.status}
                                </Badge>
                                <Text size="xs" c="dimmed">
                                  {testResponse.duration_ms}ms
                                </Text>
                              </Group>
                              <Code block style={{ fontSize: 11, maxHeight: 200, overflow: 'auto' }}>
                                {JSON.stringify(testResponse.body, null, 2)}
                              </Code>
                            </Paper>
                          </>
                        )}
                      </Stack>
                    </Paper>
                  </Collapse>
                </Box>
              </Tabs.Panel>

              <Tabs.Panel value="logs" style={{ flex: 1 }}>
                <Box p="md" h="100%" style={{ overflow: 'auto' }}>
                  <Group justify="space-between" mb="md">
                    <Text fw={500}>Execution Logs</Text>
                    <Button
                      variant="subtle"
                      size="xs"
                      leftSection={<IconRefresh size={14} />}
                      onClick={() => loadLogs(selectedFunctionId!)}
                      loading={logsLoading}
                    >
                      Refresh
                    </Button>
                  </Group>

                  {logsLoading ? (
                    <Stack gap="xs">
                      {[1, 2, 3, 4, 5].map(i => (
                        <Skeleton key={i} height={40} />
                      ))}
                    </Stack>
                  ) : logs.length === 0 ? (
                    <Stack align="center" py="xl">
                      <IconTerminal2 size={48} style={{ opacity: 0.3 }} />
                      <Text c="dimmed">No logs yet</Text>
                      <Text size="xs" c="dimmed">
                        Logs will appear here when the function is invoked
                      </Text>
                    </Stack>
                  ) : (
                    <Stack gap={4}>
                      {logs.map(log => (
                        <Paper
                          key={log.id}
                          p="xs"
                          withBorder
                          bg={log.level === 'error' ? 'red.9' : 'dark.7'}
                          style={{ opacity: log.level === 'debug' ? 0.7 : 1 }}
                        >
                          <Group gap="xs">
                            <Badge
                              size="xs"
                              color={
                                log.level === 'error' ? 'red' :
                                log.level === 'warn' ? 'yellow' :
                                log.level === 'debug' ? 'gray' : 'blue'
                              }
                              variant="filled"
                            >
                              {log.level.toUpperCase()}
                            </Badge>
                            <Text size="xs" c="dimmed">
                              {new Date(log.timestamp).toLocaleTimeString()}
                            </Text>
                            {log.status_code && (
                              <Badge size="xs" variant="outline">
                                {log.status_code}
                              </Badge>
                            )}
                            {log.duration_ms && (
                              <Text size="xs" c="dimmed">
                                {log.duration_ms}ms
                              </Text>
                            )}
                          </Group>
                          <Text size="sm" mt={4} style={{ fontFamily: 'monospace' }}>
                            {log.message}
                          </Text>
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </Box>
              </Tabs.Panel>

              <Tabs.Panel value="metrics" style={{ flex: 1 }}>
                <Box p="md" h="100%" style={{ overflow: 'auto' }}>
                  <Group justify="space-between" mb="lg">
                    <Text fw={500}>Function Metrics</Text>
                    <SegmentedControl
                      size="xs"
                      value={metricsPeriod}
                      onChange={setMetricsPeriod}
                      data={[
                        { label: '1h', value: '1h' },
                        { label: '24h', value: '24h' },
                        { label: '7d', value: '7d' },
                        { label: '30d', value: '30d' },
                      ]}
                    />
                  </Group>

                  {metricsLoading ? (
                    <Stack gap="md">
                      <Skeleton height={100} />
                      <Skeleton height={200} />
                    </Stack>
                  ) : metrics ? (
                    <Stack gap="lg">
                      <SimpleGrid cols={3}>
                        <Paper withBorder p="md">
                          <Text size="xs" c="dimmed" tt="uppercase">
                            Total Invocations
                          </Text>
                          <Text size="xl" fw={600}>
                            {metrics.invocations.total.toLocaleString()}
                          </Text>
                        </Paper>
                        <Paper withBorder p="md">
                          <Text size="xs" c="dimmed" tt="uppercase">
                            Success Rate
                          </Text>
                          <Text size="xl" fw={600} c={metrics.invocations.error > 0 ? 'yellow' : 'green'}>
                            {metrics.invocations.total > 0
                              ? Math.round((metrics.invocations.success / metrics.invocations.total) * 100)
                              : 100}%
                          </Text>
                        </Paper>
                        <Paper withBorder p="md">
                          <Text size="xs" c="dimmed" tt="uppercase">
                            Avg Latency
                          </Text>
                          <Text size="xl" fw={600}>
                            {metrics.latency.avg}ms
                          </Text>
                        </Paper>
                      </SimpleGrid>

                      <Paper withBorder p="md">
                        <Text fw={500} mb="md">Invocations Over Time</Text>
                        <Box h={250}>
                          <ResponsiveContainer width="100%" height="100%">
                            <AreaChart data={metrics.invocations.by_hour}>
                              <CartesianGrid strokeDasharray="3 3" stroke="var(--mantine-color-dark-4)" />
                              <XAxis
                                dataKey="hour"
                                tickFormatter={(value) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                stroke="var(--mantine-color-dimmed)"
                                fontSize={11}
                              />
                              <YAxis stroke="var(--mantine-color-dimmed)" fontSize={11} />
                              <RechartsTooltip
                                contentStyle={{
                                  background: 'var(--mantine-color-dark-7)',
                                  border: '1px solid var(--mantine-color-dark-4)',
                                  borderRadius: 4,
                                }}
                              />
                              <Area
                                type="monotone"
                                dataKey="count"
                                stroke="#3ECF8E"
                                fill="#3ECF8E"
                                fillOpacity={0.2}
                              />
                            </AreaChart>
                          </ResponsiveContainer>
                        </Box>
                      </Paper>
                    </Stack>
                  ) : (
                    <Stack align="center" py="xl">
                      <IconChartBar size={48} style={{ opacity: 0.3 }} />
                      <Text c="dimmed">No metrics data yet</Text>
                    </Stack>
                  )}
                </Box>
              </Tabs.Panel>

              <Tabs.Panel value="deployments" style={{ flex: 1 }}>
                <Box p="md" h="100%" style={{ overflow: 'auto' }}>
                  <Text fw={500} mb="md">Deployment History</Text>

                  {deploymentsLoading ? (
                    <Stack gap="xs">
                      {[1, 2, 3].map(i => (
                        <Skeleton key={i} height={60} />
                      ))}
                    </Stack>
                  ) : deployments.length === 0 ? (
                    <Stack align="center" py="xl">
                      <IconHistory size={48} style={{ opacity: 0.3 }} />
                      <Text c="dimmed">No deployments yet</Text>
                      <Text size="xs" c="dimmed">
                        Deploy your function to see version history
                      </Text>
                    </Stack>
                  ) : (
                    <Stack gap="xs">
                      {deployments.map((deployment, idx) => (
                        <Paper key={deployment.id} withBorder p="md">
                          <Group justify="space-between">
                            <Group>
                              <Badge
                                color={
                                  deployment.status === 'deployed' ? 'green' :
                                  deployment.status === 'deploying' ? 'blue' :
                                  deployment.status === 'failed' ? 'red' : 'gray'
                                }
                                variant="light"
                              >
                                {deployment.status}
                              </Badge>
                              <Text fw={500}>Version {deployment.version}</Text>
                              {idx === 0 && (
                                <Badge variant="outline" size="xs">
                                  Current
                                </Badge>
                              )}
                            </Group>
                            <Text size="xs" c="dimmed">
                              {new Date(deployment.deployed_at).toLocaleString()}
                            </Text>
                          </Group>
                        </Paper>
                      ))}
                    </Stack>
                  )}
                </Box>
              </Tabs.Panel>

              <Tabs.Panel value="settings" style={{ flex: 1 }}>
                <Box p="md" maw={600}>
                  <Stack gap="lg">
                    <Paper withBorder p="md">
                      <Text fw={500} mb="md">Function Settings</Text>
                      <Stack gap="md">
                        <TextInput
                          label="Function Name"
                          value={selectedFunction.name}
                          disabled
                        />
                        <TextInput
                          label="Slug"
                          value={selectedFunction.slug}
                          disabled
                          description="URL path for invoking the function"
                        />
                        <TextInput
                          label="Entrypoint"
                          value={selectedFunction.entrypoint}
                          disabled
                        />
                        <Switch
                          label="Require JWT verification"
                          description="When enabled, requests must include a valid JWT token"
                          checked={selectedFunction.verify_jwt}
                          onChange={async (e) => {
                            await functionsApi.updateFunction(selectedFunction.id, {
                              verify_jwt: e.target.checked,
                            });
                            await loadFunctions();
                          }}
                        />
                      </Stack>
                    </Paper>

                    <Paper withBorder p="md">
                      <Text fw={500} mb="md">Endpoint</Text>
                      <Group>
                        <Code style={{ flex: 1 }}>
                          {window.location.origin}/functions/v1/{selectedFunction.slug}
                        </Code>
                        <CopyButton value={`${window.location.origin}/functions/v1/${selectedFunction.slug}`}>
                          {({ copied, copy }) => (
                            <ActionIcon variant="subtle" onClick={copy}>
                              {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                            </ActionIcon>
                          )}
                        </CopyButton>
                      </Group>
                    </Paper>

                    <Paper withBorder p="md" bg="red.9">
                      <Text fw={500} mb="md" c="red.3">Danger Zone</Text>
                      <Button
                        color="red"
                        variant="outline"
                        leftSection={<IconTrash size={16} />}
                        onClick={() => handleDeleteFunction(selectedFunction.id)}
                      >
                        Delete function
                      </Button>
                    </Paper>
                  </Stack>
                </Box>
              </Tabs.Panel>
            </Tabs>
          </Paper>
        ) : (
          <Paper withBorder p="xl" style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Stack align="center">
              <IconFileCode size={64} style={{ opacity: 0.3 }} />
              <Text size="lg" c="dimmed">
                Select a function or create a new one
              </Text>
              <Button
                leftSection={<IconPlus size={16} />}
                onClick={() => setCreateModalOpen(true)}
              >
                Create function
              </Button>
            </Stack>
          </Paper>
        )}
      </Group>

      {/* Create Function Modal */}
      <Modal
        opened={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        title="Create Edge Function"
        size="lg"
      >
        <Stack gap="md">
          <TextInput
            label="Function name"
            placeholder="my-function"
            value={newFunctionName}
            onChange={(e) => setNewFunctionName(e.target.value)}
            description="Use lowercase letters, numbers, and hyphens"
          />

          <Switch
            label="Require JWT verification"
            description="Requests must include a valid JWT token"
            checked={verifyJwt}
            onChange={(e) => setVerifyJwt(e.target.checked)}
          />

          <Divider label="Start from template" labelPosition="center" />

          <SimpleGrid cols={2}>
            {templates.map(template => (
              <Card
                key={template.id}
                withBorder
                padding="sm"
                style={{
                  cursor: 'pointer',
                  borderColor: selectedTemplate === template.id ? 'var(--mantine-color-blue-6)' : undefined,
                }}
                onClick={() => setSelectedTemplate(selectedTemplate === template.id ? null : template.id)}
              >
                <Group gap="sm">
                  <ThemeIcon size="lg" variant="light" color="blue">
                    {templateIcons[template.icon] || <IconCode size={20} />}
                  </ThemeIcon>
                  <Box>
                    <Text size="sm" fw={500}>{template.name}</Text>
                    <Text size="xs" c="dimmed">{template.description}</Text>
                  </Box>
                </Group>
              </Card>
            ))}
          </SimpleGrid>

          <Group justify="flex-end" mt="md">
            <Button variant="subtle" onClick={() => setCreateModalOpen(false)}>
              Cancel
            </Button>
            <Button onClick={handleCreateFunction} disabled={!newFunctionName.trim()}>
              Create function
            </Button>
          </Group>
        </Stack>
      </Modal>

      {/* Secrets Modal */}
      <Modal
        opened={secretsModalOpen}
        onClose={() => setSecretsModalOpen(false)}
        title="Manage Secrets"
        size="lg"
      >
        <Stack gap="md">
          <Text size="sm" c="dimmed">
            Secrets are available to all Edge Functions as environment variables.
          </Text>

          <Paper withBorder p="md">
            <Stack gap="sm">
              {secrets.length === 0 ? (
                <Text size="sm" c="dimmed" ta="center" py="md">
                  No secrets yet
                </Text>
              ) : (
                secrets.map(secret => (
                  <Group key={secret.id} justify="space-between">
                    <Text size="sm" ff="monospace">{secret.name}</Text>
                    <Group gap="xs">
                      <Text size="xs" c="dimmed">
                        Added {new Date(secret.created_at).toLocaleDateString()}
                      </Text>
                      <ActionIcon
                        variant="subtle"
                        color="red"
                        size="sm"
                        onClick={() => handleDeleteSecret(secret.name)}
                      >
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Group>
                ))
              )}
            </Stack>
          </Paper>

          <Divider />

          {bulkSecretsMode ? (
            <>
              <Textarea
                label="Bulk add secrets (.env format)"
                placeholder="SECRET_KEY=value&#10;ANOTHER_KEY=another_value"
                value={bulkSecretsText}
                onChange={(e) => setBulkSecretsText(e.target.value)}
                minRows={6}
                styles={{ input: { fontFamily: 'monospace' } }}
              />
              <Group justify="flex-end">
                <Button variant="subtle" onClick={() => setBulkSecretsMode(false)}>
                  Cancel
                </Button>
                <Button onClick={handleBulkUpdateSecrets}>
                  Add secrets
                </Button>
              </Group>
            </>
          ) : (
            <>
              <Group>
                <TextInput
                  placeholder="Secret name"
                  value={newSecretName}
                  onChange={(e) => setNewSecretName(e.target.value)}
                  style={{ flex: 1 }}
                />
                <TextInput
                  placeholder="Secret value"
                  type="password"
                  value={newSecretValue}
                  onChange={(e) => setNewSecretValue(e.target.value)}
                  style={{ flex: 1 }}
                />
                <ActionIcon
                  variant="filled"
                  onClick={handleAddSecret}
                  disabled={!newSecretName.trim() || !newSecretValue.trim()}
                >
                  <IconPlus size={16} />
                </ActionIcon>
              </Group>
              <Button variant="subtle" onClick={() => setBulkSecretsMode(true)}>
                Bulk add from .env
              </Button>
            </>
          )}
        </Stack>
      </Modal>
    </Box>
  );
}

export default FunctionsPage;
