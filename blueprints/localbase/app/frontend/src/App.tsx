import { Routes, Route, Link, useLocation } from 'react-router-dom'
import {
  AppShell,
  NavLink,
  Group,
  Text,
  Box,
  Stack,
  Badge,
  Container,
  Title,
  Card,
  SimpleGrid,
  ThemeIcon,
} from '@mantine/core'
import {
  IconDatabase,
  IconUsers,
  IconFolder,
  IconBolt,
  IconCode,
  IconApi,
  IconSettings,
  IconDashboard,
  IconBrandSupabase,
} from '@tabler/icons-react'

// Dashboard Home Page
function DashboardHome() {
  return (
    <Container size="xl" py="xl">
      <Title order={2} mb="lg">Dashboard</Title>
      <SimpleGrid cols={{ base: 1, sm: 2, lg: 4 }} spacing="lg">
        <StatCard
          title="Users"
          value="0"
          icon={<IconUsers size={24} />}
          color="blue"
        />
        <StatCard
          title="Tables"
          value="0"
          icon={<IconDatabase size={24} />}
          color="violet"
        />
        <StatCard
          title="Storage Buckets"
          value="0"
          icon={<IconFolder size={24} />}
          color="orange"
        />
        <StatCard
          title="Edge Functions"
          value="0"
          icon={<IconCode size={24} />}
          color="green"
        />
      </SimpleGrid>
    </Container>
  )
}

// Stat Card Component
function StatCard({
  title,
  value,
  icon,
  color,
}: {
  title: string
  value: string
  icon: React.ReactNode
  color: string
}) {
  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder>
      <Group justify="space-between" mb="xs">
        <Text size="sm" c="dimmed">{title}</Text>
        <ThemeIcon size="lg" variant="light" color={color}>
          {icon}
        </ThemeIcon>
      </Group>
      <Text size="xl" fw={700}>{value}</Text>
    </Card>
  )
}

// Placeholder pages
function TableEditorPage() {
  return <PagePlaceholder title="Table Editor" description="Visual database management" />
}

function SQLEditorPage() {
  return <PagePlaceholder title="SQL Editor" description="Write and execute SQL queries" />
}

function AuthUsersPage() {
  return <PagePlaceholder title="Users" description="Manage authenticated users" />
}

function StoragePage() {
  return <PagePlaceholder title="Storage" description="File storage management" />
}

function RealtimePage() {
  return <PagePlaceholder title="Realtime" description="WebSocket connections" />
}

function FunctionsPage() {
  return <PagePlaceholder title="Edge Functions" description="Serverless functions" />
}

function APIDocsPage() {
  return <PagePlaceholder title="API Docs" description="Auto-generated API documentation" />
}

function SettingsPage() {
  return <PagePlaceholder title="Settings" description="Project configuration" />
}

function PagePlaceholder({ title, description }: { title: string; description: string }) {
  return (
    <Container size="xl" py="xl">
      <Title order={2}>{title}</Title>
      <Text c="dimmed" mt="xs">{description}</Text>
      <Card shadow="sm" padding="xl" radius="md" withBorder mt="lg">
        <Text c="dimmed" ta="center">
          This page is under development. Check back soon!
        </Text>
      </Card>
    </Container>
  )
}

// Navigation items
const navItems = [
  { icon: IconDashboard, label: 'Dashboard', path: '/' },
  { icon: IconDatabase, label: 'Table Editor', path: '/table-editor' },
  { icon: IconCode, label: 'SQL Editor', path: '/sql-editor' },
  { icon: IconUsers, label: 'Authentication', path: '/auth/users' },
  { icon: IconFolder, label: 'Storage', path: '/storage' },
  { icon: IconBolt, label: 'Realtime', path: '/realtime' },
  { icon: IconCode, label: 'Edge Functions', path: '/functions' },
  { icon: IconApi, label: 'API Docs', path: '/api-docs' },
  { icon: IconSettings, label: 'Settings', path: '/settings' },
]

export default function App() {
  const location = useLocation()

  return (
    <AppShell
      navbar={{ width: 250, breakpoint: 'sm' }}
      padding="md"
    >
      <AppShell.Navbar p="md">
        <Box mb="lg">
          <Group gap="xs">
            <IconBrandSupabase size={28} color="#3ECF8E" />
            <Text fw={700} size="lg">Localbase</Text>
          </Group>
          <Badge size="xs" variant="light" color="green" mt="xs">
            Development
          </Badge>
        </Box>

        <Stack gap={4}>
          {navItems.map((item) => (
            <NavLink
              key={item.path}
              component={Link}
              to={item.path}
              label={item.label}
              leftSection={<item.icon size={18} />}
              active={location.pathname === item.path}
              variant="light"
            />
          ))}
        </Stack>
      </AppShell.Navbar>

      <AppShell.Main>
        <Routes>
          <Route path="/" element={<DashboardHome />} />
          <Route path="/table-editor" element={<TableEditorPage />} />
          <Route path="/sql-editor" element={<SQLEditorPage />} />
          <Route path="/auth/users" element={<AuthUsersPage />} />
          <Route path="/storage" element={<StoragePage />} />
          <Route path="/realtime" element={<RealtimePage />} />
          <Route path="/functions" element={<FunctionsPage />} />
          <Route path="/api-docs" element={<APIDocsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
        </Routes>
      </AppShell.Main>
    </AppShell>
  )
}
