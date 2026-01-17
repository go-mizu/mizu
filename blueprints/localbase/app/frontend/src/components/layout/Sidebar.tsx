import { Link, useLocation } from 'react-router-dom';
import {
  Box,
  NavLink,
  Stack,
  Group,
  Text,
  Badge,
  Divider,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import {
  IconDatabase,
  IconUsers,
  IconFolder,
  IconBolt,
  IconCode,
  IconApi,
  IconSettings,
  IconDashboard,
  IconTable,
  IconChevronLeft,
  IconChevronRight,
} from '@tabler/icons-react';
import { useAppStore } from '../../stores/appStore';

const navItems = [
  { icon: IconDashboard, label: 'Dashboard', path: '/' },
  { icon: IconTable, label: 'Table Editor', path: '/table-editor' },
  { icon: IconCode, label: 'SQL Editor', path: '/sql-editor' },
  { icon: IconUsers, label: 'Authentication', path: '/auth/users' },
  { icon: IconFolder, label: 'Storage', path: '/storage' },
  { icon: IconBolt, label: 'Realtime', path: '/realtime' },
  { icon: IconCode, label: 'Edge Functions', path: '/functions' },
  { icon: IconApi, label: 'API Docs', path: '/api-docs' },
];

const bottomNavItems = [
  { icon: IconSettings, label: 'Settings', path: '/settings' },
];

export function Sidebar() {
  const location = useLocation();
  const { sidebarCollapsed, toggleSidebar, projectName } = useAppStore();

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  return (
    <Box
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
      }}
    >
      {/* Logo / Project Name */}
      <Box p="md" pb="sm">
        <Group gap="xs" wrap="nowrap">
          <Box
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background: 'linear-gradient(135deg, #3ECF8E 0%, #24B47E 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <IconDatabase size={18} color="white" />
          </Box>
          {!sidebarCollapsed && (
            <Box>
              <Text fw={600} size="sm" truncate>
                {projectName}
              </Text>
              <Badge size="xs" variant="light" color="green">
                Local
              </Badge>
            </Box>
          )}
        </Group>
      </Box>

      <Divider />

      {/* Main Navigation */}
      <Box p="xs" style={{ flex: 1, overflow: 'auto' }}>
        <Stack gap={2}>
          {navItems.map((item) => (
            <Tooltip
              key={item.path}
              label={item.label}
              position="right"
              disabled={!sidebarCollapsed}
              withArrow
            >
              <NavLink
                component={Link}
                to={item.path}
                label={sidebarCollapsed ? undefined : item.label}
                leftSection={<item.icon size={18} stroke={1.5} />}
                active={isActive(item.path)}
                style={{
                  borderRadius: 6,
                }}
              />
            </Tooltip>
          ))}
        </Stack>
      </Box>

      <Divider />

      {/* Bottom Navigation */}
      <Box p="xs">
        <Stack gap={2}>
          {bottomNavItems.map((item) => (
            <Tooltip
              key={item.path}
              label={item.label}
              position="right"
              disabled={!sidebarCollapsed}
              withArrow
            >
              <NavLink
                component={Link}
                to={item.path}
                label={sidebarCollapsed ? undefined : item.label}
                leftSection={<item.icon size={18} stroke={1.5} />}
                active={isActive(item.path)}
                style={{
                  borderRadius: 6,
                }}
              />
            </Tooltip>
          ))}

          {/* Collapse Toggle */}
          <Tooltip
            label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            position="right"
            withArrow
          >
            <ActionIcon
              variant="subtle"
              color="gray"
              onClick={toggleSidebar}
              style={{ width: '100%', justifyContent: 'flex-start', padding: '8px 12px' }}
              h={36}
            >
              {sidebarCollapsed ? (
                <IconChevronRight size={18} />
              ) : (
                <Group gap="xs">
                  <IconChevronLeft size={18} />
                  {!sidebarCollapsed && (
                    <Text size="sm" c="dimmed">
                      Collapse
                    </Text>
                  )}
                </Group>
              )}
            </ActionIcon>
          </Tooltip>
        </Stack>
      </Box>
    </Box>
  );
}
