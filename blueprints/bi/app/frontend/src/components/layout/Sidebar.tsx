import { NavLink, Stack, Text, Box, TextInput, Divider } from '@mantine/core'
import { IconHome, IconFolder, IconPlus, IconLayoutDashboard, IconSettings, IconSearch, IconDatabase } from '@tabler/icons-react'
import { useNavigate, useLocation } from 'react-router-dom'

export default function Sidebar() {
  const navigate = useNavigate()
  const location = useLocation()

  const isActive = (path: string) => location.pathname === path || location.pathname.startsWith(path + '/')

  return (
    <Stack h="100%" bg="#2e353b" p="md" gap="sm">
      {/* Logo */}
      <Text
        size="xl"
        fw={700}
        c="#509EE3"
        mb="sm"
        style={{ cursor: 'pointer' }}
        onClick={() => navigate('/')}
      >
        BI
      </Text>

      {/* Search */}
      <TextInput
        placeholder="Search..."
        leftSection={<IconSearch size={16} />}
        size="sm"
        styles={{
          input: {
            backgroundColor: 'rgba(255,255,255,0.1)',
            border: 'none',
            color: 'white',
            '&::placeholder': { color: 'rgba(255,255,255,0.5)' },
          },
        }}
      />

      <Divider color="rgba(255,255,255,0.1)" my="sm" />

      {/* Navigation */}
      <Stack gap={4}>
        <NavLink
          label="Home"
          leftSection={<IconHome size={18} />}
          active={location.pathname === '/'}
          onClick={() => navigate('/')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />

        <NavLink
          label="Browse"
          leftSection={<IconFolder size={18} />}
          active={isActive('/browse')}
          onClick={() => navigate('/browse')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />

        <Divider color="rgba(255,255,255,0.1)" my="sm" />

        <NavLink
          label="New Question"
          leftSection={<IconPlus size={18} />}
          active={isActive('/question/new')}
          onClick={() => navigate('/question/new')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />

        <NavLink
          label="New Dashboard"
          leftSection={<IconLayoutDashboard size={18} />}
          active={isActive('/dashboard/new')}
          onClick={() => navigate('/dashboard/new')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />
      </Stack>

      {/* Bottom section */}
      <Box mt="auto">
        <Divider color="rgba(255,255,255,0.1)" mb="sm" />
        <NavLink
          label="Data Model"
          leftSection={<IconDatabase size={18} />}
          active={isActive('/admin/datamodel')}
          onClick={() => navigate('/admin/datamodel')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />
        <NavLink
          label="Settings"
          leftSection={<IconSettings size={18} />}
          active={isActive('/admin/settings')}
          onClick={() => navigate('/admin/settings')}
          styles={{
            root: {
              borderRadius: 6,
              color: 'rgba(255,255,255,0.7)',
              '&[data-active]': {
                backgroundColor: '#509EE3',
                color: 'white',
              },
              '&:hover': {
                backgroundColor: 'rgba(255,255,255,0.1)',
              },
            },
          }}
        />
      </Box>
    </Stack>
  )
}
