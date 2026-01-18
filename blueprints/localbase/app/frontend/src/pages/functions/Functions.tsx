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
        <Text style={{ color: 'var(--lb-text-secondary)' }}>Loading functions...</Text>
      </Stack>
    );
  }

  return (
    <Box h="calc(100vh - 60px)">
      <Group justify="space-between" mb="md">
        <Title order={3} style={{ color: 'var(--lb-text-primary)' }}>Edge Functions</Title>
        <Group>
          <Button
            variant="subtle"
            leftSection={<IconKey size={16} />}
            onClick={() => setSecretsModalOpen(true)}
            style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
          >
            Manage Secrets
          </Button>
          <Button
            leftSection={<IconPlus size={16} />}
            onClick={() => setCreateModalOpen(true)}
            style={{
              background: 'var(--lb-brand)',
              borderRadius: 'var(--lb-radius-md)',
              transition: 'var(--lb-transition-fast)',
            }}
            className="lb-btn-primary"
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
          style={{
            flexShrink: 0,
            display: 'flex',
            flexDirection: 'column',
            background: 'var(--lb-bg-secondary)',
            borderColor: 'var(--lb-border-default)',
            borderRadius: 'var(--lb-radius-md)',
          }}
        >
          <Text size="xs" fw={600} mb="xs" px="xs" style={{ color: 'var(--lb-text-muted)' }}>
            FUNCTIONS
          </Text>
          <ScrollArea style={{ flex: 1 }}>
            <Stack gap={2}>
              {functions.length === 0 ? (
                <Text size="sm" ta="center" py="lg" style={{ color: 'var(--lb-text-secondary)' }}>
                  No functions yet
                </Text>
              ) : (
                functions.map(fn => (
                  <Paper
                    key={fn.id}
                    p="xs"
                    withBorder={selectedFunctionId === fn.id}
                    style={{
                      cursor: 'pointer',
                      background: selectedFunctionId === fn.id ? 'var(--lb-bg-tertiary)' : 'transparent',
                      borderColor: selectedFunctionId === fn.id ? 'var(--lb-border-strong)' : 'transparent',
                      borderRadius: 'var(--lb-radius-sm)',
                      transition: 'var(--lb-transition-fast)',
                    }}
                    onClick={() => setSelectedFunctionId(fn.id)}
                  >
                    <Group gap="xs" wrap="nowrap">
                      <ThemeIcon
                        size="xs"
                        radius="xl"
                        color={fn.status === 'active' ? 'green' : 'gray'}
                        variant="filled"
                        style={{
                          background: fn.status === 'active' ? 'var(--lb-success)' : 'var(--lb-text-muted)',
                        }}
                      >
                        <Box w={6} h={6} style={{ borderRadius: '50%', background: 'currentColor' }} />
                      </ThemeIcon>
                      <Box style={{ flex: 1, minWidth: 0 }}>
                        <Text size="sm" fw={500} truncate style={{ color: 'var(--lb-text-primary)' }}>
                          {fn.name}
                        </Text>
                        <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                          v{fn.version}
                        </Text>
                      </Box>
                      {fn.verify_jwt ? (
                        <Tooltip label="JWT Required">
                          <IconLock size={14} style={{ opacity: 0.5, color: 'var(--lb-text-muted)' }} />
                        </Tooltip>
                      ) : (
                        <Tooltip label="Public">
                          <IconLockOpen size={14} style={{ opacity: 0.5, color: 'var(--lb-text-muted)' }} />
                        </Tooltip>
                      )}
                    </Group>
                  </Paper>
                ))
              )}
            </Stack>
          </ScrollArea>

          <Divider my="sm" style={{ borderColor: 'var(--lb-border-muted)' }} />

          <Text size="xs" fw={600} mb="xs" px="xs" style={{ color: 'var(--lb-text-muted)' }}>
            SECRETS
          </Text>
          <Stack gap={2}>
            {secrets.slice(0, 3).map(secret => (
              <Text key={secret.id} size="xs" px="xs" style={{ color: 'var(--lb-text-secondary)' }}>
                {secret.name}
              </Text>
            ))}
            {secrets.length > 3 && (
              <Text size="xs" px="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                +{secrets.length - 3} more
              </Text>
            )}
            <Button
              variant="subtle"
              size="xs"
              onClick={() => setSecretsModalOpen(true)}
              fullWidth
              style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
            >
              Manage secrets
            </Button>
          </Stack>
        </Paper>

        {/* Main Content Area */}
        {selectedFunction ? (
          <Paper
            withBorder
            p={0}
            style={{
              flex: 1,
              height: '100%',
              display: 'flex',
              flexDirection: 'column',
              overflow: 'hidden',
              background: 'var(--lb-bg-secondary)',
              borderColor: 'var(--lb-border-default)',
              borderRadius: 'var(--lb-radius-md)',
            }}
          >
            {/* Header */}
            <Group justify="space-between" p="sm" style={{ borderBottom: '1px solid var(--lb-border-default)' }}>
              <Group>
                <Title order={4} style={{ color: 'var(--lb-text-primary)' }}>{selectedFunction.name}</Title>
                <Badge
                  color={selectedFunction.status === 'active' ? 'green' : 'gray'}
                  variant="light"
                  style={{
                    background: selectedFunction.status === 'active' ? 'var(--lb-success-bg)' : 'var(--lb-bg-tertiary)',
                    color: selectedFunction.status === 'active' ? 'var(--lb-success-text)' : 'var(--lb-text-secondary)',
                  }}
                >
                  {selectedFunction.status}
                </Badge>
                {isDirty && (
                  <Badge
                    color="yellow"
                    variant="light"
                    style={{
                      background: 'var(--lb-warning-bg)',
                      color: 'var(--lb-warning-text)',
                    }}
                  >
                    Unsaved changes
                  </Badge>
                )}
              </Group>
              <Group>
                <Button
                  variant="subtle"
                  leftSection={<IconPlayerPlay size={16} />}
                  onClick={() => setTestPanelOpen(!testPanelOpen)}
                  style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
                >
                  Test
                </Button>
                <Button
                  leftSection={<IconRocket size={16} />}
                  loading={isDeploying}
                  onClick={handleDeploy}
                  disabled={!isDirty && sourceCode === originalSourceCode}
                  style={{
                    background: 'var(--lb-brand)',
                    borderRadius: 'var(--lb-radius-md)',
                    transition: 'var(--lb-transition-fast)',
                  }}
                  className="lb-btn-primary"
                >
                  Deploy
                </Button>
                <Menu shadow="md" width={200}>
                  <Menu.Target>
                    <ActionIcon variant="subtle" style={{ color: 'var(--lb-text-secondary)' }}>
                      <IconDotsVertical size={16} />
                    </ActionIcon>
                  </Menu.Target>
                  <Menu.Dropdown style={{ background: 'var(--lb-bg-secondary)', borderColor: 'var(--lb-border-default)' }}>
                    <Menu.Item
                      leftSection={<IconDownload size={16} />}
                      onClick={handleDownload}
                      style={{ color: 'var(--lb-text-primary)' }}
                    >
                      Download
                    </Menu.Item>
                    <Menu.Divider style={{ borderColor: 'var(--lb-border-muted)' }} />
                    <Menu.Item
                      leftSection={<IconTrash size={16} />}
                      color="red"
                      onClick={() => handleDeleteFunction(selectedFunction.id)}
                      style={{ color: 'var(--lb-error)' }}
                    >
                      Delete function
                    </Menu.Item>
                  </Menu.Dropdown>
                </Menu>
              </Group>
            </Group>

            {/* Tabs */}
            <Tabs value={activeTab} onChange={setActiveTab} style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
              <Tabs.List px="sm" style={{ borderColor: 'var(--lb-border-default)' }}>
                <Tabs.Tab value="code" leftSection={<IconCode size={14} />} style={{ color: 'var(--lb-text-secondary)' }}>
                  Code
                </Tabs.Tab>
                <Tabs.Tab value="logs" leftSection={<IconTerminal2 size={14} />} style={{ color: 'var(--lb-text-secondary)' }}>
                  Logs
                </Tabs.Tab>
                <Tabs.Tab value="metrics" leftSection={<IconChartBar size={14} />} style={{ color: 'var(--lb-text-secondary)' }}>
                  Metrics
                </Tabs.Tab>
                <Tabs.Tab value="deployments" leftSection={<IconHistory size={14} />} style={{ color: 'var(--lb-text-secondary)' }}>
                  Deployments
                </Tabs.Tab>
                <Tabs.Tab value="settings" leftSection={<IconSettings size={14} />} style={{ color: 'var(--lb-text-secondary)' }}>
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
                        borderLeft: '1px solid var(--lb-border-default)',
                        display: 'flex',
                        flexDirection: 'column',
                        background: 'var(--lb-bg-secondary)',
                      }}
                    >
                      <Group justify="space-between" mb="md">
                        <Text fw={600} style={{ color: 'var(--lb-text-primary)' }}>Test Function</Text>
                        <ActionIcon variant="subtle" onClick={() => setTestPanelOpen(false)} style={{ color: 'var(--lb-text-secondary)' }}>
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
                            styles={{
                              input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                            }}
                          />
                          <TextInput
                            flex={1}
                            value={`/functions/v1/${selectedFunction.slug}`}
                            readOnly
                            styles={{
                              input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                            }}
                          />
                        </Group>

                        <Divider label="Headers" labelPosition="left" style={{ borderColor: 'var(--lb-border-muted)' }} />

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
                              styles={{
                                input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                              }}
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
                              styles={{
                                input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                              }}
                            />
                            <ActionIcon
                              variant="subtle"
                              color="red"
                              onClick={() => {
                                setTestHeaders(testHeaders.filter((_, i) => i !== idx));
                              }}
                              style={{ color: 'var(--lb-error)' }}
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
                          style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
                        >
                          Add header
                        </Button>

                        <Divider label="Body" labelPosition="left" style={{ borderColor: 'var(--lb-border-muted)' }} />

                        <Textarea
                          value={testBody}
                          onChange={(e) => setTestBody(e.target.value)}
                          minRows={4}
                          styles={{
                            input: {
                              fontFamily: 'monospace',
                              fontSize: 12,
                              background: 'var(--lb-bg-tertiary)',
                              borderColor: 'var(--lb-border-default)',
                            },
                          }}
                        />

                        <Button
                          fullWidth
                          leftSection={<IconPlayerPlay size={16} />}
                          loading={isTesting}
                          onClick={handleTestFunction}
                          style={{
                            background: 'var(--lb-brand)',
                            borderRadius: 'var(--lb-radius-md)',
                            transition: 'var(--lb-transition-fast)',
                          }}
                          className="lb-btn-primary"
                        >
                          Run function
                        </Button>

                        {testResponse && (
                          <>
                            <Divider label="Response" labelPosition="left" style={{ borderColor: 'var(--lb-border-muted)' }} />
                            <Paper
                              p="xs"
                              withBorder
                              style={{
                                background: 'var(--lb-bg-tertiary)',
                                borderColor: 'var(--lb-border-default)',
                                borderRadius: 'var(--lb-radius-sm)',
                              }}
                            >
                              <Group gap="xs" mb="xs">
                                <Badge
                                  color={testResponse.status < 400 ? 'green' : 'red'}
                                  variant="light"
                                  style={{
                                    background: testResponse.status < 400 ? 'var(--lb-success-bg)' : 'var(--lb-error-bg)',
                                    color: testResponse.status < 400 ? 'var(--lb-success-text)' : 'var(--lb-error-text)',
                                  }}
                                >
                                  {testResponse.status}
                                </Badge>
                                <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                                  {testResponse.duration_ms}ms
                                </Text>
                              </Group>
                              <Code block style={{ fontSize: 11, maxHeight: 200, overflow: 'auto', background: 'var(--lb-bg-primary)' }}>
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
                    <Text fw={500} style={{ color: 'var(--lb-text-primary)' }}>Execution Logs</Text>
                    <Button
                      variant="subtle"
                      size="xs"
                      leftSection={<IconRefresh size={14} />}
                      onClick={() => loadLogs(selectedFunctionId!)}
                      loading={logsLoading}
                      style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
                    >
                      Refresh
                    </Button>
                  </Group>

                  {logsLoading ? (
                    <Stack gap="xs">
                      {[1, 2, 3, 4, 5].map(i => (
                        <Skeleton key={i} height={40} style={{ borderRadius: 'var(--lb-radius-sm)' }} />
                      ))}
                    </Stack>
                  ) : logs.length === 0 ? (
                    <Stack align="center" py="xl">
                      <IconTerminal2 size={48} style={{ opacity: 0.3, color: 'var(--lb-text-muted)' }} />
                      <Text style={{ color: 'var(--lb-text-secondary)' }}>No logs yet</Text>
                      <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
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
                          style={{
                            background: log.level === 'error' ? 'var(--lb-error-bg)' : 'var(--lb-bg-tertiary)',
                            borderColor: log.level === 'error' ? 'var(--lb-error)' : 'var(--lb-border-default)',
                            borderRadius: 'var(--lb-radius-sm)',
                            opacity: log.level === 'debug' ? 0.7 : 1,
                          }}
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
                              style={{
                                background:
                                  log.level === 'error' ? 'var(--lb-error)' :
                                  log.level === 'warn' ? 'var(--lb-warning)' :
                                  log.level === 'debug' ? 'var(--lb-text-muted)' : 'var(--lb-info)',
                              }}
                            >
                              {log.level.toUpperCase()}
                            </Badge>
                            <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                              {new Date(log.timestamp).toLocaleTimeString()}
                            </Text>
                            {log.status_code && (
                              <Badge
                                size="xs"
                                variant="outline"
                                style={{ borderColor: 'var(--lb-border-default)', color: 'var(--lb-text-secondary)' }}
                              >
                                {log.status_code}
                              </Badge>
                            )}
                            {log.duration_ms && (
                              <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                                {log.duration_ms}ms
                              </Text>
                            )}
                          </Group>
                          <Text size="sm" mt={4} style={{ fontFamily: 'monospace', color: 'var(--lb-text-primary)' }}>
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
                    <Text fw={500} style={{ color: 'var(--lb-text-primary)' }}>Function Metrics</Text>
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
                      styles={{
                        root: { background: 'var(--lb-bg-tertiary)' },
                      }}
                    />
                  </Group>

                  {metricsLoading ? (
                    <Stack gap="md">
                      <Skeleton height={100} style={{ borderRadius: 'var(--lb-radius-md)' }} />
                      <Skeleton height={200} style={{ borderRadius: 'var(--lb-radius-md)' }} />
                    </Stack>
                  ) : metrics ? (
                    <Stack gap="lg">
                      <SimpleGrid cols={3}>
                        <Paper
                          withBorder
                          p="md"
                          style={{
                            background: 'var(--lb-bg-tertiary)',
                            borderColor: 'var(--lb-border-default)',
                            borderRadius: 'var(--lb-radius-md)',
                          }}
                        >
                          <Text size="xs" tt="uppercase" style={{ color: 'var(--lb-text-muted)' }}>
                            Total Invocations
                          </Text>
                          <Text size="xl" fw={600} style={{ color: 'var(--lb-text-primary)' }}>
                            {metrics.invocations.total.toLocaleString()}
                          </Text>
                        </Paper>
                        <Paper
                          withBorder
                          p="md"
                          style={{
                            background: 'var(--lb-bg-tertiary)',
                            borderColor: 'var(--lb-border-default)',
                            borderRadius: 'var(--lb-radius-md)',
                          }}
                        >
                          <Text size="xs" tt="uppercase" style={{ color: 'var(--lb-text-muted)' }}>
                            Success Rate
                          </Text>
                          <Text
                            size="xl"
                            fw={600}
                            style={{
                              color: metrics.invocations.error > 0 ? 'var(--lb-warning)' : 'var(--lb-success)',
                            }}
                          >
                            {metrics.invocations.total > 0
                              ? Math.round((metrics.invocations.success / metrics.invocations.total) * 100)
                              : 100}%
                          </Text>
                        </Paper>
                        <Paper
                          withBorder
                          p="md"
                          style={{
                            background: 'var(--lb-bg-tertiary)',
                            borderColor: 'var(--lb-border-default)',
                            borderRadius: 'var(--lb-radius-md)',
                          }}
                        >
                          <Text size="xs" tt="uppercase" style={{ color: 'var(--lb-text-muted)' }}>
                            Avg Latency
                          </Text>
                          <Text size="xl" fw={600} style={{ color: 'var(--lb-text-primary)' }}>
                            {metrics.latency.avg}ms
                          </Text>
                        </Paper>
                      </SimpleGrid>

                      <Paper
                        withBorder
                        p="md"
                        style={{
                          background: 'var(--lb-bg-tertiary)',
                          borderColor: 'var(--lb-border-default)',
                          borderRadius: 'var(--lb-radius-md)',
                        }}
                      >
                        <Text fw={500} mb="md" style={{ color: 'var(--lb-text-primary)' }}>Invocations Over Time</Text>
                        <Box h={250}>
                          <ResponsiveContainer width="100%" height="100%">
                            <AreaChart data={metrics.invocations.by_hour}>
                              <CartesianGrid strokeDasharray="3 3" stroke="var(--lb-border-muted)" />
                              <XAxis
                                dataKey="hour"
                                tickFormatter={(value) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                                stroke="var(--lb-text-muted)"
                                fontSize={11}
                              />
                              <YAxis stroke="var(--lb-text-muted)" fontSize={11} />
                              <RechartsTooltip
                                contentStyle={{
                                  background: 'var(--lb-bg-secondary)',
                                  border: '1px solid var(--lb-border-default)',
                                  borderRadius: 'var(--lb-radius-sm)',
                                }}
                              />
                              <Area
                                type="monotone"
                                dataKey="count"
                                stroke="var(--lb-brand)"
                                fill="var(--lb-brand)"
                                fillOpacity={0.2}
                              />
                            </AreaChart>
                          </ResponsiveContainer>
                        </Box>
                      </Paper>
                    </Stack>
                  ) : (
                    <Stack align="center" py="xl">
                      <IconChartBar size={48} style={{ opacity: 0.3, color: 'var(--lb-text-muted)' }} />
                      <Text style={{ color: 'var(--lb-text-secondary)' }}>No metrics data yet</Text>
                    </Stack>
                  )}
                </Box>
              </Tabs.Panel>

              <Tabs.Panel value="deployments" style={{ flex: 1 }}>
                <Box p="md" h="100%" style={{ overflow: 'auto' }}>
                  <Text fw={500} mb="md" style={{ color: 'var(--lb-text-primary)' }}>Deployment History</Text>

                  {deploymentsLoading ? (
                    <Stack gap="xs">
                      {[1, 2, 3].map(i => (
                        <Skeleton key={i} height={60} style={{ borderRadius: 'var(--lb-radius-md)' }} />
                      ))}
                    </Stack>
                  ) : deployments.length === 0 ? (
                    <Stack align="center" py="xl">
                      <IconHistory size={48} style={{ opacity: 0.3, color: 'var(--lb-text-muted)' }} />
                      <Text style={{ color: 'var(--lb-text-secondary)' }}>No deployments yet</Text>
                      <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                        Deploy your function to see version history
                      </Text>
                    </Stack>
                  ) : (
                    <Stack gap="xs">
                      {deployments.map((deployment, idx) => (
                        <Paper
                          key={deployment.id}
                          withBorder
                          p="md"
                          style={{
                            background: 'var(--lb-bg-tertiary)',
                            borderColor: 'var(--lb-border-default)',
                            borderRadius: 'var(--lb-radius-md)',
                          }}
                        >
                          <Group justify="space-between">
                            <Group>
                              <Badge
                                color={
                                  deployment.status === 'deployed' ? 'green' :
                                  deployment.status === 'deploying' ? 'blue' :
                                  deployment.status === 'failed' ? 'red' : 'gray'
                                }
                                variant="light"
                                style={{
                                  background:
                                    deployment.status === 'deployed' ? 'var(--lb-success-bg)' :
                                    deployment.status === 'deploying' ? 'var(--lb-info-bg)' :
                                    deployment.status === 'failed' ? 'var(--lb-error-bg)' : 'var(--lb-bg-tertiary)',
                                  color:
                                    deployment.status === 'deployed' ? 'var(--lb-success-text)' :
                                    deployment.status === 'deploying' ? 'var(--lb-info-text)' :
                                    deployment.status === 'failed' ? 'var(--lb-error-text)' : 'var(--lb-text-secondary)',
                                }}
                              >
                                {deployment.status}
                              </Badge>
                              <Text fw={500} style={{ color: 'var(--lb-text-primary)' }}>Version {deployment.version}</Text>
                              {idx === 0 && (
                                <Badge
                                  variant="outline"
                                  size="xs"
                                  style={{ borderColor: 'var(--lb-brand)', color: 'var(--lb-brand)' }}
                                >
                                  Current
                                </Badge>
                              )}
                            </Group>
                            <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
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
                    <Paper
                      withBorder
                      p="md"
                      style={{
                        background: 'var(--lb-bg-tertiary)',
                        borderColor: 'var(--lb-border-default)',
                        borderRadius: 'var(--lb-radius-md)',
                      }}
                    >
                      <Text fw={500} mb="md" style={{ color: 'var(--lb-text-primary)' }}>Function Settings</Text>
                      <Stack gap="md">
                        <TextInput
                          label="Function Name"
                          value={selectedFunction.name}
                          disabled
                          styles={{
                            label: { color: 'var(--lb-text-secondary)' },
                            input: { background: 'var(--lb-bg-secondary)', borderColor: 'var(--lb-border-default)' },
                          }}
                        />
                        <TextInput
                          label="Slug"
                          value={selectedFunction.slug}
                          disabled
                          description="URL path for invoking the function"
                          styles={{
                            label: { color: 'var(--lb-text-secondary)' },
                            description: { color: 'var(--lb-text-tertiary)' },
                            input: { background: 'var(--lb-bg-secondary)', borderColor: 'var(--lb-border-default)' },
                          }}
                        />
                        <TextInput
                          label="Entrypoint"
                          value={selectedFunction.entrypoint}
                          disabled
                          styles={{
                            label: { color: 'var(--lb-text-secondary)' },
                            input: { background: 'var(--lb-bg-secondary)', borderColor: 'var(--lb-border-default)' },
                          }}
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
                          styles={{
                            label: { color: 'var(--lb-text-primary)' },
                            description: { color: 'var(--lb-text-tertiary)' },
                          }}
                        />
                      </Stack>
                    </Paper>

                    <Paper
                      withBorder
                      p="md"
                      style={{
                        background: 'var(--lb-bg-tertiary)',
                        borderColor: 'var(--lb-border-default)',
                        borderRadius: 'var(--lb-radius-md)',
                      }}
                    >
                      <Text fw={500} mb="md" style={{ color: 'var(--lb-text-primary)' }}>Endpoint</Text>
                      <Group>
                        <Code style={{ flex: 1, background: 'var(--lb-bg-secondary)', color: 'var(--lb-text-primary)' }}>
                          {window.location.origin}/functions/v1/{selectedFunction.slug}
                        </Code>
                        <CopyButton value={`${window.location.origin}/functions/v1/${selectedFunction.slug}`}>
                          {({ copied, copy }) => (
                            <ActionIcon
                              variant="subtle"
                              onClick={copy}
                              style={{ color: copied ? 'var(--lb-success)' : 'var(--lb-text-secondary)' }}
                            >
                              {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
                            </ActionIcon>
                          )}
                        </CopyButton>
                      </Group>
                    </Paper>

                    <Paper
                      withBorder
                      p="md"
                      style={{
                        background: 'var(--lb-error-bg)',
                        borderColor: 'var(--lb-error)',
                        borderRadius: 'var(--lb-radius-md)',
                      }}
                    >
                      <Text fw={500} mb="md" style={{ color: 'var(--lb-error-text)' }}>Danger Zone</Text>
                      <Button
                        color="red"
                        variant="outline"
                        leftSection={<IconTrash size={16} />}
                        onClick={() => handleDeleteFunction(selectedFunction.id)}
                        style={{
                          borderColor: 'var(--lb-error)',
                          color: 'var(--lb-error)',
                          transition: 'var(--lb-transition-fast)',
                        }}
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
          <Paper
            withBorder
            p="xl"
            style={{
              flex: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: 'var(--lb-bg-secondary)',
              borderColor: 'var(--lb-border-default)',
              borderRadius: 'var(--lb-radius-md)',
            }}
          >
            <Stack align="center">
              <IconFileCode size={64} style={{ opacity: 0.3, color: 'var(--lb-text-muted)' }} />
              <Text size="lg" style={{ color: 'var(--lb-text-secondary)' }}>
                Select a function or create a new one
              </Text>
              <Button
                leftSection={<IconPlus size={16} />}
                onClick={() => setCreateModalOpen(true)}
                style={{
                  background: 'var(--lb-brand)',
                  borderRadius: 'var(--lb-radius-md)',
                  transition: 'var(--lb-transition-fast)',
                }}
                className="lb-btn-primary"
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
        styles={{
          header: { background: 'var(--lb-bg-secondary)', borderBottom: '1px solid var(--lb-border-default)' },
          title: { color: 'var(--lb-text-primary)', fontWeight: 600 },
          body: { background: 'var(--lb-bg-secondary)' },
          content: { borderRadius: 'var(--lb-radius-lg)' },
        }}
      >
        <Stack gap="md">
          <TextInput
            label="Function name"
            placeholder="my-function"
            value={newFunctionName}
            onChange={(e) => setNewFunctionName(e.target.value)}
            description="Use lowercase letters, numbers, and hyphens"
            styles={{
              label: { color: 'var(--lb-text-primary)' },
              description: { color: 'var(--lb-text-tertiary)' },
              input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
            }}
          />

          <Switch
            label="Require JWT verification"
            description="Requests must include a valid JWT token"
            checked={verifyJwt}
            onChange={(e) => setVerifyJwt(e.target.checked)}
            styles={{
              label: { color: 'var(--lb-text-primary)' },
              description: { color: 'var(--lb-text-tertiary)' },
            }}
          />

          <Divider label="Start from template" labelPosition="center" style={{ borderColor: 'var(--lb-border-muted)' }} />

          <SimpleGrid cols={2}>
            {templates.map(template => (
              <Card
                key={template.id}
                withBorder
                padding="sm"
                style={{
                  cursor: 'pointer',
                  background: selectedTemplate === template.id ? 'var(--lb-brand-light)' : 'var(--lb-bg-tertiary)',
                  borderColor: selectedTemplate === template.id ? 'var(--lb-brand)' : 'var(--lb-border-default)',
                  borderRadius: 'var(--lb-radius-md)',
                  transition: 'var(--lb-transition-fast)',
                }}
                onClick={() => setSelectedTemplate(selectedTemplate === template.id ? null : template.id)}
              >
                <Group gap="sm">
                  <ThemeIcon
                    size="lg"
                    variant="light"
                    style={{
                      background: 'var(--lb-brand-light)',
                      color: 'var(--lb-brand)',
                    }}
                  >
                    {templateIcons[template.icon] || <IconCode size={20} />}
                  </ThemeIcon>
                  <Box>
                    <Text size="sm" fw={500} style={{ color: 'var(--lb-text-primary)' }}>{template.name}</Text>
                    <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>{template.description}</Text>
                  </Box>
                </Group>
              </Card>
            ))}
          </SimpleGrid>

          <Group justify="flex-end" mt="md">
            <Button
              variant="subtle"
              onClick={() => setCreateModalOpen(false)}
              style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleCreateFunction}
              disabled={!newFunctionName.trim()}
              style={{
                background: 'var(--lb-brand)',
                borderRadius: 'var(--lb-radius-md)',
                transition: 'var(--lb-transition-fast)',
              }}
              className="lb-btn-primary"
            >
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
        styles={{
          header: { background: 'var(--lb-bg-secondary)', borderBottom: '1px solid var(--lb-border-default)' },
          title: { color: 'var(--lb-text-primary)', fontWeight: 600 },
          body: { background: 'var(--lb-bg-secondary)' },
          content: { borderRadius: 'var(--lb-radius-lg)' },
        }}
      >
        <Stack gap="md">
          <Text size="sm" style={{ color: 'var(--lb-text-secondary)' }}>
            Secrets are available to all Edge Functions as environment variables.
          </Text>

          <Paper
            withBorder
            p="md"
            style={{
              background: 'var(--lb-bg-tertiary)',
              borderColor: 'var(--lb-border-default)',
              borderRadius: 'var(--lb-radius-md)',
            }}
          >
            <Stack gap="sm">
              {secrets.length === 0 ? (
                <Text size="sm" ta="center" py="md" style={{ color: 'var(--lb-text-secondary)' }}>
                  No secrets yet
                </Text>
              ) : (
                secrets.map(secret => (
                  <Group key={secret.id} justify="space-between">
                    <Text size="sm" ff="monospace" style={{ color: 'var(--lb-text-primary)' }}>{secret.name}</Text>
                    <Group gap="xs">
                      <Text size="xs" style={{ color: 'var(--lb-text-tertiary)' }}>
                        Added {new Date(secret.created_at).toLocaleDateString()}
                      </Text>
                      <ActionIcon
                        variant="subtle"
                        color="red"
                        size="sm"
                        onClick={() => handleDeleteSecret(secret.name)}
                        style={{ color: 'var(--lb-error)' }}
                      >
                        <IconTrash size={14} />
                      </ActionIcon>
                    </Group>
                  </Group>
                ))
              )}
            </Stack>
          </Paper>

          <Divider style={{ borderColor: 'var(--lb-border-muted)' }} />

          {bulkSecretsMode ? (
            <>
              <Textarea
                label="Bulk add secrets (.env format)"
                placeholder="SECRET_KEY=value&#10;ANOTHER_KEY=another_value"
                value={bulkSecretsText}
                onChange={(e) => setBulkSecretsText(e.target.value)}
                minRows={6}
                styles={{
                  label: { color: 'var(--lb-text-primary)' },
                  input: { fontFamily: 'monospace', background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                }}
              />
              <Group justify="flex-end">
                <Button
                  variant="subtle"
                  onClick={() => setBulkSecretsMode(false)}
                  style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleBulkUpdateSecrets}
                  style={{
                    background: 'var(--lb-brand)',
                    borderRadius: 'var(--lb-radius-md)',
                    transition: 'var(--lb-transition-fast)',
                  }}
                  className="lb-btn-primary"
                >
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
                  styles={{
                    input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                  }}
                />
                <TextInput
                  placeholder="Secret value"
                  type="password"
                  value={newSecretValue}
                  onChange={(e) => setNewSecretValue(e.target.value)}
                  style={{ flex: 1 }}
                  styles={{
                    input: { background: 'var(--lb-bg-tertiary)', borderColor: 'var(--lb-border-default)' },
                  }}
                />
                <ActionIcon
                  variant="filled"
                  onClick={handleAddSecret}
                  disabled={!newSecretName.trim() || !newSecretValue.trim()}
                  style={{ background: 'var(--lb-brand)', transition: 'var(--lb-transition-fast)' }}
                >
                  <IconPlus size={16} />
                </ActionIcon>
              </Group>
              <Button
                variant="subtle"
                onClick={() => setBulkSecretsMode(true)}
                style={{ color: 'var(--lb-text-secondary)', transition: 'var(--lb-transition-fast)' }}
              >
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
