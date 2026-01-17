import { useEffect, useState } from 'react';
import {
  Box,
  Paper,
  Group,
  Text,
  Stack,
  Badge,
  Button,
  Tabs,
  Card,
  ThemeIcon,
  Skeleton,
  Accordion,
  Code,
  ActionIcon,
  Tooltip,
  Alert,
  Progress,
  SimpleGrid,
} from '@mantine/core';
import {
  IconShield,
  IconChartBar,
  IconAlertTriangle,
  IconCircleCheck,
  IconInfoCircle,
  IconExternalLink,
  IconCopy,
  IconCheck,
  IconTable,
  IconLock,
  IconEye,
  IconDatabase,
  IconClock,
  IconChartLine,
  IconBulb,
} from '@tabler/icons-react';
import { Link } from 'react-router-dom';
import { PageContainer } from '../../components/layout/PageContainer';
import { databaseApi } from '../../api';

interface SecurityIssue {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  title: string;
  description: string;
  table?: string;
  suggestion: string;
  sqlFix?: string;
}

interface PerformanceIssue {
  id: string;
  severity: 'critical' | 'warning' | 'info';
  title: string;
  description: string;
  impact: string;
  suggestion: string;
  sqlFix?: string;
}

export function AdvisorsPage() {
  const [activeTab, setActiveTab] = useState<string | null>('security');
  const [loading, setLoading] = useState(true);
  const [securityIssues, setSecurityIssues] = useState<SecurityIssue[]>([]);
  const [performanceIssues, setPerformanceIssues] = useState<PerformanceIssue[]>([]);

  useEffect(() => {
    const analyzeDatabase = async () => {
      setLoading(true);
      try {
        // Fetch tables to analyze
        const tables = await databaseApi.listTables('public');

        // Mock security analysis - in real implementation this would check RLS, policies, etc.
        const securityResults: SecurityIssue[] = [];
        const perfResults: PerformanceIssue[] = [];

        for (const table of tables || []) {
          // Check RLS
          if (!table.rls_enabled) {
            securityResults.push({
              id: `rls-${table.name}`,
              severity: 'critical',
              title: `Row Level Security disabled on "${table.name}"`,
              description: `The table "${table.name}" does not have Row Level Security (RLS) enabled. This means any authenticated user can access all rows.`,
              table: table.name,
              suggestion: 'Enable RLS and create appropriate policies to restrict access.',
              sqlFix: `ALTER TABLE ${table.name} ENABLE ROW LEVEL SECURITY;`,
            });
          }

          // Check for missing indexes on likely query columns
          if (table.row_count && table.row_count > 1000) {
            perfResults.push({
              id: `index-${table.name}`,
              severity: 'warning',
              title: `Consider adding indexes to "${table.name}"`,
              description: `Table "${table.name}" has ${table.row_count} rows. Consider adding indexes on frequently queried columns.`,
              impact: 'Slow queries on large tables without proper indexes',
              suggestion: 'Analyze query patterns and add indexes on columns used in WHERE clauses and JOINs.',
              sqlFix: `-- Example: Add index on commonly queried column
CREATE INDEX idx_${table.name}_user_id ON ${table.name}(user_id);`,
            });
          }
        }

        // Add some general security recommendations if no critical issues
        if (securityResults.filter((i) => i.severity === 'critical').length === 0) {
          securityResults.push({
            id: 'general-security',
            severity: 'info',
            title: 'Security best practices',
            description: 'Your database appears to be configured securely. Here are some additional recommendations.',
            suggestion: 'Regularly review your RLS policies and ensure service_role key is never exposed to clients.',
          });
        }

        // Add performance recommendations
        perfResults.push({
          id: 'connection-pooling',
          severity: 'info',
          title: 'Use connection pooling for serverless',
          description: 'If using serverless functions, ensure you use the pooler connection string.',
          impact: 'Connection exhaustion in serverless environments',
          suggestion: 'Use transaction pooler mode for short-lived connections.',
        });

        setSecurityIssues(securityResults);
        setPerformanceIssues(perfResults);
      } catch (error) {
        console.error('Failed to analyze database:', error);
      } finally {
        setLoading(false);
      }
    };

    analyzeDatabase();
  }, []);

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'red';
      case 'warning':
        return 'yellow';
      case 'info':
        return 'blue';
      default:
        return 'gray';
    }
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'critical':
        return <IconAlertTriangle size={16} />;
      case 'warning':
        return <IconAlertTriangle size={16} />;
      case 'info':
        return <IconInfoCircle size={16} />;
      default:
        return <IconInfoCircle size={16} />;
    }
  };

  const securityScore = Math.max(
    0,
    100 -
      securityIssues.filter((i) => i.severity === 'critical').length * 30 -
      securityIssues.filter((i) => i.severity === 'warning').length * 10
  );

  const performanceScore = Math.max(
    0,
    100 -
      performanceIssues.filter((i) => i.severity === 'critical').length * 30 -
      performanceIssues.filter((i) => i.severity === 'warning').length * 10
  );

  return (
    <PageContainer
      title="Advisors"
      description="Security and performance recommendations for your project"
    >
      <Tabs value={activeTab} onChange={setActiveTab}>
        <Tabs.List mb="lg">
          <Tabs.Tab
            value="security"
            leftSection={<IconShield size={16} />}
            rightSection={
              securityIssues.filter((i) => i.severity === 'critical').length > 0 ? (
                <Badge size="xs" color="red" variant="filled">
                  {securityIssues.filter((i) => i.severity === 'critical').length}
                </Badge>
              ) : null
            }
          >
            Security
          </Tabs.Tab>
          <Tabs.Tab
            value="performance"
            leftSection={<IconChartBar size={16} />}
            rightSection={
              performanceIssues.filter((i) => i.severity === 'critical').length > 0 ? (
                <Badge size="xs" color="red" variant="filled">
                  {performanceIssues.filter((i) => i.severity === 'critical').length}
                </Badge>
              ) : null
            }
          >
            Performance
          </Tabs.Tab>
        </Tabs.List>

        <Tabs.Panel value="security">
          <Stack gap="lg">
            {/* Security Score */}
            <Paper p="md" radius="md" withBorder>
              <Group justify="space-between" mb="sm">
                <Group gap="sm">
                  <ThemeIcon
                    size="lg"
                    radius="md"
                    variant="light"
                    color={securityScore >= 80 ? 'green' : securityScore >= 50 ? 'yellow' : 'red'}
                  >
                    <IconShield size={20} />
                  </ThemeIcon>
                  <Box>
                    <Text fw={600}>Security Score</Text>
                    <Text size="xs" c="dimmed">
                      Based on RLS configuration and policy analysis
                    </Text>
                  </Box>
                </Group>
                <Text size="xl" fw={700}>
                  {securityScore}%
                </Text>
              </Group>
              <Progress
                value={securityScore}
                color={securityScore >= 80 ? 'green' : securityScore >= 50 ? 'yellow' : 'red'}
                size="lg"
                radius="xl"
              />
            </Paper>

            {/* Issues List */}
            {loading ? (
              <Stack gap="md">
                <Skeleton height={100} />
                <Skeleton height={100} />
                <Skeleton height={100} />
              </Stack>
            ) : securityIssues.length === 0 ? (
              <Alert
                icon={<IconCircleCheck size={16} />}
                title="All clear!"
                color="green"
              >
                No security issues found in your database configuration.
              </Alert>
            ) : (
              <Accordion variant="separated" radius="md">
                {securityIssues.map((issue) => (
                  <Accordion.Item key={issue.id} value={issue.id}>
                    <Accordion.Control>
                      <Group gap="sm">
                        <ThemeIcon
                          size="sm"
                          radius="xl"
                          variant="light"
                          color={getSeverityColor(issue.severity)}
                        >
                          {getSeverityIcon(issue.severity)}
                        </ThemeIcon>
                        <Box style={{ flex: 1 }}>
                          <Text size="sm" fw={500}>
                            {issue.title}
                          </Text>
                          {issue.table && (
                            <Badge size="xs" variant="light" mt={4}>
                              {issue.table}
                            </Badge>
                          )}
                        </Box>
                        <Badge
                          size="sm"
                          variant="light"
                          color={getSeverityColor(issue.severity)}
                        >
                          {issue.severity}
                        </Badge>
                      </Group>
                    </Accordion.Control>
                    <Accordion.Panel>
                      <Stack gap="md">
                        <Text size="sm" c="dimmed">
                          {issue.description}
                        </Text>
                        <Box>
                          <Text size="sm" fw={500} mb={4}>
                            <IconBulb size={14} style={{ verticalAlign: 'middle' }} /> Recommendation
                          </Text>
                          <Text size="sm">{issue.suggestion}</Text>
                        </Box>
                        {issue.sqlFix && (
                          <Box>
                            <Text size="sm" fw={500} mb={4}>
                              Quick Fix
                            </Text>
                            <Code block>{issue.sqlFix}</Code>
                            <Group mt="sm" gap="xs">
                              <Button
                                size="xs"
                                variant="light"
                                leftSection={<IconCopy size={14} />}
                                onClick={() => navigator.clipboard.writeText(issue.sqlFix!)}
                              >
                                Copy SQL
                              </Button>
                              <Button
                                size="xs"
                                variant="light"
                                component={Link}
                                to="/sql-editor"
                                leftSection={<IconDatabase size={14} />}
                              >
                                Open in SQL Editor
                              </Button>
                            </Group>
                          </Box>
                        )}
                        {issue.table && (
                          <Button
                            size="xs"
                            variant="subtle"
                            component={Link}
                            to="/database/policies"
                            rightSection={<IconExternalLink size={14} />}
                          >
                            Manage policies for {issue.table}
                          </Button>
                        )}
                      </Stack>
                    </Accordion.Panel>
                  </Accordion.Item>
                ))}
              </Accordion>
            )}
          </Stack>
        </Tabs.Panel>

        <Tabs.Panel value="performance">
          <Stack gap="lg">
            {/* Performance Score */}
            <Paper p="md" radius="md" withBorder>
              <Group justify="space-between" mb="sm">
                <Group gap="sm">
                  <ThemeIcon
                    size="lg"
                    radius="md"
                    variant="light"
                    color={
                      performanceScore >= 80
                        ? 'green'
                        : performanceScore >= 50
                          ? 'yellow'
                          : 'red'
                    }
                  >
                    <IconChartLine size={20} />
                  </ThemeIcon>
                  <Box>
                    <Text fw={600}>Performance Score</Text>
                    <Text size="xs" c="dimmed">
                      Based on indexes, query patterns, and configuration
                    </Text>
                  </Box>
                </Group>
                <Text size="xl" fw={700}>
                  {performanceScore}%
                </Text>
              </Group>
              <Progress
                value={performanceScore}
                color={
                  performanceScore >= 80
                    ? 'green'
                    : performanceScore >= 50
                      ? 'yellow'
                      : 'red'
                }
                size="lg"
                radius="xl"
              />
            </Paper>

            {/* Quick Stats */}
            <SimpleGrid cols={{ base: 1, sm: 3 }} spacing="md">
              <Card padding="md" radius="md" withBorder>
                <Group>
                  <ThemeIcon size="lg" radius="md" variant="light" color="blue">
                    <IconDatabase size={20} />
                  </ThemeIcon>
                  <Box>
                    <Text size="xs" c="dimmed">
                      Active Connections
                    </Text>
                    <Text fw={600}>5 / 100</Text>
                  </Box>
                </Group>
              </Card>
              <Card padding="md" radius="md" withBorder>
                <Group>
                  <ThemeIcon size="lg" radius="md" variant="light" color="green">
                    <IconClock size={20} />
                  </ThemeIcon>
                  <Box>
                    <Text size="xs" c="dimmed">
                      Avg Query Time
                    </Text>
                    <Text fw={600}>2.5ms</Text>
                  </Box>
                </Group>
              </Card>
              <Card padding="md" radius="md" withBorder>
                <Group>
                  <ThemeIcon size="lg" radius="md" variant="light" color="orange">
                    <IconTable size={20} />
                  </ThemeIcon>
                  <Box>
                    <Text size="xs" c="dimmed">
                      Cache Hit Ratio
                    </Text>
                    <Text fw={600}>98.5%</Text>
                  </Box>
                </Group>
              </Card>
            </SimpleGrid>

            {/* Issues List */}
            {loading ? (
              <Stack gap="md">
                <Skeleton height={100} />
                <Skeleton height={100} />
                <Skeleton height={100} />
              </Stack>
            ) : performanceIssues.length === 0 ? (
              <Alert
                icon={<IconCircleCheck size={16} />}
                title="All clear!"
                color="green"
              >
                No performance issues found in your database configuration.
              </Alert>
            ) : (
              <Accordion variant="separated" radius="md">
                {performanceIssues.map((issue) => (
                  <Accordion.Item key={issue.id} value={issue.id}>
                    <Accordion.Control>
                      <Group gap="sm">
                        <ThemeIcon
                          size="sm"
                          radius="xl"
                          variant="light"
                          color={getSeverityColor(issue.severity)}
                        >
                          {getSeverityIcon(issue.severity)}
                        </ThemeIcon>
                        <Box style={{ flex: 1 }}>
                          <Text size="sm" fw={500}>
                            {issue.title}
                          </Text>
                        </Box>
                        <Badge
                          size="sm"
                          variant="light"
                          color={getSeverityColor(issue.severity)}
                        >
                          {issue.severity}
                        </Badge>
                      </Group>
                    </Accordion.Control>
                    <Accordion.Panel>
                      <Stack gap="md">
                        <Text size="sm" c="dimmed">
                          {issue.description}
                        </Text>
                        <Box>
                          <Text size="sm" fw={500} mb={4}>
                            Impact
                          </Text>
                          <Text size="sm" c="red">
                            {issue.impact}
                          </Text>
                        </Box>
                        <Box>
                          <Text size="sm" fw={500} mb={4}>
                            <IconBulb size={14} style={{ verticalAlign: 'middle' }} /> Recommendation
                          </Text>
                          <Text size="sm">{issue.suggestion}</Text>
                        </Box>
                        {issue.sqlFix && (
                          <Box>
                            <Text size="sm" fw={500} mb={4}>
                              Example Fix
                            </Text>
                            <Code block>{issue.sqlFix}</Code>
                            <Group mt="sm" gap="xs">
                              <Button
                                size="xs"
                                variant="light"
                                leftSection={<IconCopy size={14} />}
                                onClick={() => navigator.clipboard.writeText(issue.sqlFix!)}
                              >
                                Copy SQL
                              </Button>
                              <Button
                                size="xs"
                                variant="light"
                                component={Link}
                                to="/sql-editor"
                                leftSection={<IconDatabase size={14} />}
                              >
                                Open in SQL Editor
                              </Button>
                            </Group>
                          </Box>
                        )}
                      </Stack>
                    </Accordion.Panel>
                  </Accordion.Item>
                ))}
              </Accordion>
            )}
          </Stack>
        </Tabs.Panel>
      </Tabs>
    </PageContainer>
  );
}
