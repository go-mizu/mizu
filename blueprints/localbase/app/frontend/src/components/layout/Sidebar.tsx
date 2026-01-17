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
  Collapse,
  TextInput,
} from '@mantine/core';
import {
  IconDatabase,
  IconUsers,
  IconFolder,
  IconBolt,
  IconCode,
  IconApi,
  IconSettings,
  IconLayoutDashboard,
  IconTable,
  IconChevronLeft,
  IconChevronRight,
  IconChevronDown,
  IconShield,
  IconList,
  IconEye,
  IconUserShield,
  IconFileText,
  IconTerminal2,
  IconBroadcast,
  IconAlertCircle,
  IconPlugConnected,
  IconSchema,
  IconSearch,
} from '@tabler/icons-react';
import { useAppStore } from '../../stores/appStore';
import { useState } from 'react';

interface NavItem {
  icon: typeof IconLayoutDashboard;
  label: string;
  path: string;
  children?: NavItem[];
  badge?: string;
  badgeColor?: string;
}

const navItems: NavItem[] = [
  { icon: IconLayoutDashboard, label: 'Project Overview', path: '/' },
  { icon: IconTable, label: 'Table Editor', path: '/table-editor' },
  { icon: IconCode, label: 'SQL Editor', path: '/sql-editor' },
  {
    icon: IconDatabase,
    label: 'Database',
    path: '/database',
    children: [
      { icon: IconSchema, label: 'Schema Visualizer', path: '/database/schema-visualizer' },
      { icon: IconTable, label: 'Tables', path: '/table-editor' },
      { icon: IconEye, label: 'Views', path: '/database/views' },
      { icon: IconTerminal2, label: 'Functions', path: '/database/functions' },
      { icon: IconBolt, label: 'Triggers', path: '/database/triggers' },
      { icon: IconUserShield, label: 'Roles', path: '/database/roles' },
      { icon: IconShield, label: 'Policies', path: '/database/policies' },
      { icon: IconList, label: 'Indexes', path: '/database/indexes' },
    ],
  },
  { icon: IconUsers, label: 'Authentication', path: '/auth/users' },
  { icon: IconFolder, label: 'Storage', path: '/storage' },
  { icon: IconTerminal2, label: 'Edge Functions', path: '/functions' },
  { icon: IconBroadcast, label: 'Realtime', path: '/realtime' },
];

const toolsItems: NavItem[] = [
  { icon: IconAlertCircle, label: 'Advisors', path: '/advisors', badge: 'New', badgeColor: 'green' },
  { icon: IconFileText, label: 'Logs', path: '/logs' },
  { icon: IconApi, label: 'API Docs', path: '/api-docs' },
  { icon: IconPlugConnected, label: 'Integrations', path: '/integrations', badge: 'New', badgeColor: 'green' },
];

const bottomNavItems: NavItem[] = [
  { icon: IconSettings, label: 'Project Settings', path: '/settings' },
];

export function Sidebar() {
  const location = useLocation();
  const { sidebarCollapsed, toggleSidebar, projectName } = useAppStore();
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['Database']));
  const [searchQuery, setSearchQuery] = useState('');

  const isActive = (path: string) => {
    if (path === '/') {
      return location.pathname === '/';
    }
    return location.pathname.startsWith(path);
  };

  const toggleSection = (label: string) => {
    setExpandedSections((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(label)) {
        newSet.delete(label);
      } else {
        newSet.add(label);
      }
      return newSet;
    });
  };

  const renderNavItem = (item: NavItem, _depth = 0) => {
    const hasChildren = item.children && item.children.length > 0;
    const isExpanded = expandedSections.has(item.label);
    const isChildActive = hasChildren && item.children?.some((child) => isActive(child.path));

    if (hasChildren && !sidebarCollapsed) {
      return (
        <Box key={item.path}>
          <NavLink
            label={item.label}
            leftSection={<item.icon size={18} stroke={1.5} />}
            rightSection={
              <IconChevronDown
                size={14}
                style={{
                  transform: isExpanded ? 'rotate(180deg)' : 'rotate(0deg)',
                  transition: 'transform 200ms ease',
                }}
              />
            }
            onClick={() => toggleSection(item.label)}
            active={isChildActive}
            style={{ borderRadius: 6 }}
          />
          <Collapse in={isExpanded}>
            <Stack gap={2} pl="md" mt={4}>
              {item.children?.map((child) => (
                <Tooltip
                  key={child.path}
                  label={child.label}
                  position="right"
                  disabled={!sidebarCollapsed}
                  withArrow
                >
                  <NavLink
                    component={Link}
                    to={child.path}
                    label={child.label}
                    leftSection={<child.icon size={16} stroke={1.5} />}
                    active={isActive(child.path)}
                    style={{ borderRadius: 6 }}
                  />
                </Tooltip>
              ))}
            </Stack>
          </Collapse>
        </Box>
      );
    }

    // For collapsed sidebar or items without children
    if (hasChildren && sidebarCollapsed) {
      return (
        <Tooltip key={item.path} label={item.label} position="right" withArrow>
          <NavLink
            component={Link}
            to={item.children![0].path}
            leftSection={<item.icon size={18} stroke={1.5} />}
            active={isChildActive}
            style={{ borderRadius: 6 }}
          />
        </Tooltip>
      );
    }

    return (
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
          rightSection={
            item.badge && !sidebarCollapsed ? (
              <Badge size="xs" variant="light" color={item.badgeColor || 'gray'}>
                {item.badge}
              </Badge>
            ) : undefined
          }
          active={isActive(item.path)}
          style={{ borderRadius: 6 }}
        />
      </Tooltip>
    );
  };

  return (
    <Box
      style={{
        display: 'flex',
        flexDirection: 'column',
        height: '100%',
        backgroundColor: 'var(--supabase-sidebar-bg)',
      }}
    >
      {/* Logo / Project Name */}
      <Box
        p="md"
        pb="sm"
        style={{
          borderBottom: '1px solid var(--supabase-sidebar-divider)',
        }}
      >
        <Group gap="xs" wrap="nowrap">
          <Box
            style={{
              width: 32,
              height: 32,
              borderRadius: 8,
              background: 'linear-gradient(135deg, #3ECF8E 0%, #1C9B5E 100%)',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              flexShrink: 0,
            }}
          >
            <IconDatabase size={18} color="white" />
          </Box>
          {!sidebarCollapsed && (
            <Box style={{ minWidth: 0, flex: 1 }}>
              <Text
                fw={600}
                size="sm"
                truncate
                style={{ color: 'var(--supabase-sidebar-text-active)' }}
              >
                {projectName}
              </Text>
              <Badge
                size="xs"
                variant="light"
                color="green"
                style={{
                  backgroundColor: 'rgba(62, 207, 142, 0.2)',
                  color: 'var(--supabase-brand)',
                  marginTop: 2,
                }}
              >
                Local
              </Badge>
            </Box>
          )}
        </Group>
      </Box>

      {/* Search (visible when expanded) */}
      {!sidebarCollapsed && (
        <Box px="sm" pt="sm">
          <TextInput
            size="xs"
            placeholder="Search..."
            leftSection={<IconSearch size={14} />}
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            styles={{
              input: {
                backgroundColor: 'var(--supabase-sidebar-bg-hover)',
                borderColor: 'var(--supabase-sidebar-border)',
                color: 'var(--supabase-sidebar-text-active)',
                '&::placeholder': {
                  color: 'var(--supabase-sidebar-text)',
                },
              },
            }}
          />
        </Box>
      )}

      {/* Main Navigation */}
      <Box p="xs" style={{ flex: 1, overflow: 'auto' }}>
        <Stack gap={2}>
          {navItems
            .filter((item) =>
              searchQuery === '' ||
              item.label.toLowerCase().includes(searchQuery.toLowerCase())
            )
            .map((item) => renderNavItem(item))}
        </Stack>

        {/* Divider before tools */}
        <Divider my="sm" style={{ borderColor: 'var(--supabase-sidebar-divider)' }} />

        {/* Tools Section */}
        {!sidebarCollapsed && (
          <Text
            size="xs"
            fw={600}
            tt="uppercase"
            c="dimmed"
            px="sm"
            mb="xs"
            style={{
              color: 'var(--supabase-sidebar-text)',
              letterSpacing: '0.05em',
              fontSize: '0.6875rem',
            }}
          >
            Tools
          </Text>
        )}
        <Stack gap={2}>
          {toolsItems.map((item) => renderNavItem(item))}
        </Stack>
      </Box>

      <Divider style={{ borderColor: 'var(--supabase-sidebar-divider)' }} />

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
              onClick={toggleSidebar}
              style={{
                width: '100%',
                justifyContent: sidebarCollapsed ? 'center' : 'flex-start',
                padding: '8px 12px',
                color: 'var(--supabase-sidebar-text)',
              }}
              h={36}
            >
              {sidebarCollapsed ? (
                <IconChevronRight size={18} />
              ) : (
                <Group gap="xs">
                  <IconChevronLeft size={18} />
                  <Text
                    size="sm"
                    style={{ color: 'var(--supabase-sidebar-text)' }}
                  >
                    Collapse
                  </Text>
                </Group>
              )}
            </ActionIcon>
          </Tooltip>
        </Stack>
      </Box>
    </Box>
  );
}
