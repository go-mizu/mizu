import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  NavLink, Stack, Text, Box, TextInput, Divider, Group, UnstyledButton,
  Collapse, ActionIcon, Tooltip, Menu, ScrollArea
} from '@mantine/core'
import {
  IconHome, IconFolder, IconPlus, IconLayoutDashboard, IconSettings, IconSearch,
  IconDatabase, IconUsers, IconChevronRight, IconChevronDown, IconLogout,
  IconFileAnalytics, IconChartBar
} from '@tabler/icons-react'
import { useCollections, useCurrentUser, useLogout } from '../../api/hooks'
import { useUIStore } from '../../stores/uiStore'

const sidebarTheme = {
  bg: '#2e353b',
  text: 'rgba(255, 255, 255, 0.7)',
  textHover: '#ffffff',
  active: '#509EE3',
  border: 'rgba(255, 255, 255, 0.1)',
  inputBg: 'rgba(255, 255, 255, 0.1)',
}

const browseItems = [
  { icon: IconDatabase, label: 'Databases', path: '/browse/databases' },
  { icon: IconFileAnalytics, label: 'Models', path: '/browse/models' },
  { icon: IconChartBar, label: 'Metrics', path: '/browse/metrics' },
]

const adminItems = [
  { icon: IconDatabase, label: 'Data Model', path: '/admin/datamodel' },
  { icon: IconUsers, label: 'People', path: '/admin/people' },
  { icon: IconSettings, label: 'Settings', path: '/admin/settings' },
]

interface CollectionItemProps {
  id: string
  name: string
  color?: string
  depth?: number
  collections: any[]
}

function CollectionItem({ id, name, color, depth = 0, collections }: CollectionItemProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const [expanded, setExpanded] = useState(false)
  const children = collections?.filter(c => c.parent_id === id) || []
  const hasChildren = children.length > 0
  const isActive = location.pathname === `/browse/${id}`

  return (
    <Box>
      <UnstyledButton
        onClick={() => navigate(`/browse/${id}`)}
        px="md"
        py={6}
        ml={depth * 16}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 8,
          color: isActive ? '#fff' : sidebarTheme.text,
          backgroundColor: isActive ? sidebarTheme.active : 'transparent',
          borderRadius: 6,
          width: `calc(100% - ${depth * 16}px)`,
        }}
        onMouseEnter={(e) => {
          if (!isActive) e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.1)'
        }}
        onMouseLeave={(e) => {
          if (!isActive) e.currentTarget.style.backgroundColor = 'transparent'
        }}
      >
        {hasChildren && (
          <ActionIcon
            size="xs"
            variant="transparent"
            color="gray"
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
          >
            {expanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
          </ActionIcon>
        )}
        {!hasChildren && <Box w={18} />}
        <IconFolder size={16} style={{ color: color || sidebarTheme.text }} />
        <Text size="xs" truncate style={{ flex: 1 }}>{name}</Text>
      </UnstyledButton>

      {hasChildren && (
        <Collapse in={expanded}>
          {children.map(child => (
            <CollectionItem
              key={child.id}
              id={child.id}
              name={child.name}
              color={child.color}
              depth={depth + 1}
              collections={collections}
            />
          ))}
        </Collapse>
      )}
    </Box>
  )
}

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()
  const { data: user } = useCurrentUser()
  const { mutate: logout } = useLogout()
  const { data: collections } = useCollections()
  const { openCommandPalette } = useUIStore()

  const rootCollections = collections?.filter(c => !c.parent_id) || []
  const [browseExpanded, setBrowseExpanded] = useState(location.pathname.startsWith('/browse'))
  const [adminExpanded, setAdminExpanded] = useState(location.pathname.startsWith('/admin'))

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + '/')

  const navLinkStyles = {
    root: {
      borderRadius: 6,
      color: sidebarTheme.text,
      '&[data-active]': {
        backgroundColor: sidebarTheme.active,
        color: 'white',
      },
      '&:hover': {
        backgroundColor: 'rgba(255,255,255,0.1)',
      },
    },
  }

  return (
    <Box
      component="aside"
      w={260}
      h="100vh"
      style={{
        backgroundColor: sidebarTheme.bg,
        display: 'flex',
        flexDirection: 'column',
        borderRight: `1px solid ${sidebarTheme.border}`,
      }}
    >
      {/* Logo */}
      <Group px="md" py="md" style={{ borderBottom: `1px solid ${sidebarTheme.border}` }}>
        <UnstyledButton onClick={() => navigate('/')}>
          <Text size="xl" fw={700} style={{ color: sidebarTheme.active }}>BI</Text>
        </UnstyledButton>
      </Group>

      {/* Search */}
      <Box px="md" py="sm">
        <TextInput
          placeholder="Search... (Ctrl+K)"
          leftSection={<IconSearch size={16} />}
          size="xs"
          onClick={openCommandPalette}
          readOnly
          styles={{
            input: {
              backgroundColor: sidebarTheme.inputBg,
              border: 'none',
              color: sidebarTheme.text,
              cursor: 'pointer',
            },
          }}
        />
      </Box>

      {/* Main Navigation */}
      <ScrollArea style={{ flex: 1 }} px="sm">
        <Stack gap={4}>
          <NavLink
            label="Home"
            leftSection={<IconHome size={18} />}
            active={location.pathname === '/'}
            onClick={() => navigate('/')}
            styles={navLinkStyles}
          />

          <NavLink
            label="Browse"
            leftSection={<IconFolder size={18} />}
            active={isActive('/browse') && location.pathname === '/browse'}
            onClick={() => {
              navigate('/browse')
              setBrowseExpanded(!browseExpanded)
            }}
            rightSection={browseExpanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
            styles={navLinkStyles}
          />

          <Collapse in={browseExpanded}>
            <Stack gap={2} ml="md">
              {browseItems.map(item => (
                <NavLink
                  key={item.path}
                  label={item.label}
                  leftSection={<item.icon size={16} />}
                  active={isActive(item.path)}
                  onClick={() => navigate(item.path)}
                  styles={navLinkStyles}
                />
              ))}
            </Stack>
          </Collapse>

          <Divider color={sidebarTheme.border} my="sm" />

          <NavLink
            label="New Question"
            leftSection={<IconPlus size={18} />}
            active={isActive('/question/new')}
            onClick={() => navigate('/question/new')}
            styles={navLinkStyles}
          />

          <NavLink
            label="New Dashboard"
            leftSection={<IconLayoutDashboard size={18} />}
            active={isActive('/dashboard/new')}
            onClick={() => navigate('/dashboard/new')}
            styles={navLinkStyles}
          />

          {/* Collections Section */}
          {rootCollections.length > 0 && (
            <>
              <Divider color={sidebarTheme.border} my="sm" />
              <Group px="md" justify="space-between">
                <Text size="xs" c="dimmed" tt="uppercase" fw={600}>Collections</Text>
                <Tooltip label="New collection">
                  <ActionIcon
                    size="xs"
                    variant="transparent"
                    color="gray"
                    onClick={() => navigate('/collection/new')}
                  >
                    <IconPlus size={14} />
                  </ActionIcon>
                </Tooltip>
              </Group>
              <Stack gap={2}>
                {rootCollections.map(collection => (
                  <CollectionItem
                    key={collection.id}
                    id={collection.id}
                    name={collection.name}
                    color={collection.color}
                    collections={collections || []}
                  />
                ))}
              </Stack>
            </>
          )}
        </Stack>
      </ScrollArea>

      {/* Admin Section */}
      {user?.role === 'admin' && (
        <Box px="sm" pb="sm">
          <Divider color={sidebarTheme.border} mb="sm" />
          <NavLink
            label="Admin"
            leftSection={<IconSettings size={18} />}
            onClick={() => setAdminExpanded(!adminExpanded)}
            rightSection={adminExpanded ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
            styles={navLinkStyles}
          />
          <Collapse in={adminExpanded}>
            <Stack gap={2} ml="md">
              {adminItems.map(item => (
                <NavLink
                  key={item.path}
                  label={item.label}
                  leftSection={<item.icon size={16} />}
                  active={isActive(item.path)}
                  onClick={() => navigate(item.path)}
                  styles={navLinkStyles}
                />
              ))}
            </Stack>
          </Collapse>
        </Box>
      )}

      {/* User Section */}
      <Box px="md" py="sm" style={{ borderTop: `1px solid ${sidebarTheme.border}` }}>
        <Menu position="top-start" width={200}>
          <Menu.Target>
            <UnstyledButton
              w="100%"
              p="xs"
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: 12,
                borderRadius: 6,
                color: sidebarTheme.text,
              }}
              onMouseEnter={(e) => e.currentTarget.style.backgroundColor = 'rgba(255,255,255,0.1)'}
              onMouseLeave={(e) => e.currentTarget.style.backgroundColor = 'transparent'}
            >
              <Box
                w={32}
                h={32}
                style={{
                  backgroundColor: sidebarTheme.active,
                  borderRadius: '50%',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Text size="sm" fw={600} c="white">
                  {user?.name?.charAt(0).toUpperCase() || 'U'}
                </Text>
              </Box>
              <Box style={{ flex: 1 }}>
                <Text size="sm" c="white" lineClamp={1}>{user?.name || 'User'}</Text>
                <Text size="xs" c="dimmed" lineClamp={1}>{user?.email}</Text>
              </Box>
            </UnstyledButton>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item leftSection={<IconSettings size={14} />} onClick={() => navigate('/admin/settings')}>
              Settings
            </Menu.Item>
            <Menu.Divider />
            <Menu.Item leftSection={<IconLogout size={14} />} color="red" onClick={() => logout()}>
              Log out
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Box>
    </Box>
  )
}
