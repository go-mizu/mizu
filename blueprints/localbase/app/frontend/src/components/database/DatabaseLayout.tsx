import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import {
  Box,
  NavLink,
  Stack,
  Text,
  Group,
  Badge,
  Select,
  Divider,
  Paper,
  Skeleton,
  ScrollArea,
} from '@mantine/core';
import {
  IconTable,
  IconCode,
  IconSchema,
  IconShield,
  IconUserShield,
  IconList,
  IconEye,
  IconBolt,
  IconTerminal2,
  IconPuzzle,
  IconDatabase,
} from '@tabler/icons-react';
import { databaseApi, type DatabaseOverview } from '../../api/database';

interface NavItem {
  icon: typeof IconTable;
  label: string;
  path: string;
  badge?: string;
  badgeColor?: string;
  count?: number;
}

const databaseNavItems: NavItem[] = [
  { icon: IconTable, label: 'Tables', path: '/database/tables' },
  { icon: IconCode, label: 'SQL Editor', path: '/database/sql' },
  { icon: IconSchema, label: 'Schema Visualizer', path: '/database/schema' },
  { icon: IconShield, label: 'Policies', path: '/database/policies' },
  { icon: IconUserShield, label: 'Roles', path: '/database/roles' },
  { icon: IconList, label: 'Indexes', path: '/database/indexes' },
  { icon: IconEye, label: 'Views', path: '/database/views' },
  { icon: IconBolt, label: 'Triggers', path: '/database/triggers' },
  { icon: IconTerminal2, label: 'Functions', path: '/database/functions' },
  { icon: IconPuzzle, label: 'Extensions', path: '/database/extensions' },
];

interface DatabaseLayoutProps {
  children?: React.ReactNode;
}

export function DatabaseLayout({ children }: DatabaseLayoutProps) {
  const location = useLocation();
  const [overview, setOverview] = useState<DatabaseOverview | null>(null);
  const [schemas, setSchemas] = useState<string[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<string>('public');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [overviewData, schemasData] = await Promise.all([
          databaseApi.getOverview(),
          databaseApi.listSchemas(),
        ]);
        setOverview(overviewData);
        setSchemas(schemasData);
      } catch (error) {
        console.error('Failed to fetch database overview:', error);
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, []);

  const isActive = (path: string) => {
    return location.pathname === path || location.pathname.startsWith(path + '/');
  };

  // Get counts for nav items from overview
  const getNavItemCount = (label: string): number | undefined => {
    if (!overview) return undefined;
    switch (label) {
      case 'Tables':
        return overview.total_tables;
      case 'Views':
        return overview.total_views;
      case 'Indexes':
        return overview.total_indexes;
      case 'Policies':
        return overview.total_policies;
      case 'Functions':
        return overview.total_functions;
      default:
        return undefined;
    }
  };

  return (
    <Box style={{ display: 'flex', height: '100%' }}>
      {/* Database Sidebar */}
      <Paper
        style={{
          width: 240,
          minWidth: 240,
          height: '100%',
          borderRight: '1px solid var(--mantine-color-default-border)',
          display: 'flex',
          flexDirection: 'column',
          backgroundColor: 'var(--mantine-color-body)',
        }}
      >
        {/* Header */}
        <Box p="md" pb="sm">
          <Group gap="xs" mb="sm">
            <IconDatabase size={20} style={{ color: 'var(--mantine-color-green-6)' }} />
            <Text fw={600} size="sm">Database</Text>
          </Group>

          {/* Schema Selector */}
          <Select
            size="xs"
            placeholder="Select schema"
            value={selectedSchema}
            onChange={(value) => setSelectedSchema(value || 'public')}
            data={schemas.map((s) => ({ value: s, label: s }))}
            styles={{
              input: {
                fontSize: '12px',
              },
            }}
          />
        </Box>

        <Divider />

        {/* Overview Stats */}
        <Box px="md" py="sm">
          {loading ? (
            <Stack gap="xs">
              <Skeleton height={14} width="60%" />
              <Skeleton height={14} width="80%" />
            </Stack>
          ) : overview ? (
            <Stack gap={4}>
              <Group gap="xs" justify="space-between">
                <Text size="xs" c="dimmed">Size</Text>
                <Text size="xs" fw={500}>{overview.database_size}</Text>
              </Group>
              <Group gap="xs" justify="space-between">
                <Text size="xs" c="dimmed">Connections</Text>
                <Text size="xs" fw={500}>{overview.connection_count}</Text>
              </Group>
            </Stack>
          ) : null}
        </Box>

        <Divider />

        {/* Navigation */}
        <ScrollArea style={{ flex: 1 }} p="xs">
          <Stack gap={2}>
            {databaseNavItems.map((item) => {
              const count = getNavItemCount(item.label);
              return (
                <NavLink
                  key={item.path}
                  component={Link}
                  to={item.path}
                  label={item.label}
                  leftSection={<item.icon size={16} stroke={1.5} />}
                  rightSection={
                    count !== undefined ? (
                      <Badge size="xs" variant="light" color="gray" radius="sm">
                        {count}
                      </Badge>
                    ) : item.badge ? (
                      <Badge size="xs" variant="light" color={item.badgeColor || 'gray'}>
                        {item.badge}
                      </Badge>
                    ) : undefined
                  }
                  active={isActive(item.path)}
                  style={{ borderRadius: 6 }}
                  styles={{
                    root: {
                      fontSize: '13px',
                    },
                  }}
                />
              );
            })}
          </Stack>
        </ScrollArea>
      </Paper>

      {/* Main Content */}
      <Box style={{ flex: 1, overflow: 'auto' }}>
        {children || <Outlet />}
      </Box>
    </Box>
  );
}
