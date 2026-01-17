import { useEffect } from 'react';
import {
  SimpleGrid,
  Card,
  Text,
  Group,
  ThemeIcon,
  Skeleton,
  Badge,
  Box,
  Stack,
} from '@mantine/core';
import {
  IconUsers,
  IconDatabase,
  IconFolder,
  IconCode,
  IconCheck,
  IconX,
} from '@tabler/icons-react';
import { PageContainer } from '../components/layout/PageContainer';
import { useApi } from '../hooks/useApi';
import { dashboardApi } from '../api';
import type { DashboardStats, HealthStatus } from '../types';

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
  loading?: boolean;
  subtitle?: string;
}

function StatCard({ title, value, icon, color, loading, subtitle }: StatCardProps) {
  return (
    <Card className="supabase-stat-card">
      <Group justify="space-between" mb="xs">
        <Text size="sm" c="dimmed" fw={500}>
          {title}
        </Text>
        <ThemeIcon size="lg" variant="light" color={color} radius="md">
          {icon}
        </ThemeIcon>
      </Group>
      {loading ? (
        <Skeleton height={36} width={60} />
      ) : (
        <>
          <Text className="supabase-stat-value">{value}</Text>
          {subtitle && (
            <Text size="xs" c="dimmed" mt={4}>
              {subtitle}
            </Text>
          )}
        </>
      )}
    </Card>
  );
}

interface ServiceStatusProps {
  name: string;
  healthy: boolean;
}

function ServiceStatus({ name, healthy }: ServiceStatusProps) {
  return (
    <Group justify="space-between" py="xs">
      <Text size="sm">{name}</Text>
      <Badge
        variant="light"
        color={healthy ? 'green' : 'red'}
        leftSection={healthy ? <IconCheck size={12} /> : <IconX size={12} />}
      >
        {healthy ? 'Healthy' : 'Unhealthy'}
      </Badge>
    </Group>
  );
}

export function Dashboard() {
  const {
    data: stats,
    loading: statsLoading,
    execute: fetchStats,
  } = useApi<DashboardStats>(() => dashboardApi.getStats(), {
    showErrorNotification: false,
  });

  const {
    data: health,
    loading: healthLoading,
    execute: fetchHealth,
  } = useApi<HealthStatus>(() => dashboardApi.getHealth(), {
    showErrorNotification: false,
  });

  useEffect(() => {
    fetchStats();
    fetchHealth();
  }, []);

  return (
    <PageContainer title="Dashboard" description="Overview of your Localbase project">
      {/* Stats Cards */}
      <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="lg" mb="xl">
        <StatCard
          title="Users"
          value={stats?.users?.total ?? 0}
          icon={<IconUsers size={20} />}
          color="blue"
          loading={statsLoading}
          subtitle="Total registered users"
        />
        <StatCard
          title="Tables"
          value={stats?.database?.tables ?? 0}
          icon={<IconDatabase size={20} />}
          color="violet"
          loading={statsLoading}
          subtitle="In public schema"
        />
        <StatCard
          title="Storage Buckets"
          value={stats?.storage?.buckets ?? 0}
          icon={<IconFolder size={20} />}
          color="orange"
          loading={statsLoading}
          subtitle="Total buckets"
        />
        <StatCard
          title="Edge Functions"
          value={stats?.functions?.active ?? 0}
          icon={<IconCode size={20} />}
          color="green"
          loading={statsLoading}
          subtitle={`${stats?.functions?.total ?? 0} total`}
        />
      </SimpleGrid>

      {/* Service Health */}
      <SimpleGrid cols={{ base: 1, lg: 2 }} spacing="lg">
        <Card className="supabase-section">
          <Text fw={600} mb="md">
            Service Status
          </Text>
          {healthLoading ? (
            <Stack gap="xs">
              <Skeleton height={36} />
              <Skeleton height={36} />
              <Skeleton height={36} />
              <Skeleton height={36} />
            </Stack>
          ) : health ? (
            <Stack gap={0}>
              <ServiceStatus name="Database" healthy={health.services.database} />
              <ServiceStatus name="Auth" healthy={health.services.auth} />
              <ServiceStatus name="Storage" healthy={health.services.storage} />
              <ServiceStatus name="Realtime" healthy={health.services.realtime} />
            </Stack>
          ) : (
            <Text c="dimmed" size="sm">
              Unable to fetch service status
            </Text>
          )}
        </Card>

        <Card className="supabase-section">
          <Text fw={600} mb="md">
            Quick Links
          </Text>
          <Stack gap="sm">
            <QuickLink
              title="Table Editor"
              description="View and edit your database tables"
              href="/table-editor"
            />
            <QuickLink
              title="SQL Editor"
              description="Run SQL queries on your database"
              href="/sql-editor"
            />
            <QuickLink
              title="Authentication"
              description="Manage users and authentication"
              href="/auth/users"
            />
            <QuickLink
              title="Storage"
              description="Upload and manage files"
              href="/storage"
            />
          </Stack>
        </Card>
      </SimpleGrid>
    </PageContainer>
  );
}

function QuickLink({
  title,
  description,
  href,
}: {
  title: string;
  description: string;
  href: string;
}) {
  return (
    <Box
      component="a"
      href={href}
      style={{
        display: 'block',
        padding: '12px',
        borderRadius: 6,
        border: '1px solid var(--supabase-border)',
        textDecoration: 'none',
        color: 'inherit',
        transition: 'background-color 0.15s',
      }}
      onMouseEnter={(e) => {
        e.currentTarget.style.backgroundColor = 'var(--supabase-bg-surface)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.backgroundColor = 'transparent';
      }}
    >
      <Text size="sm" fw={500}>
        {title}
      </Text>
      <Text size="xs" c="dimmed">
        {description}
      </Text>
    </Box>
  );
}
