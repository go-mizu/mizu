import { Container, Stack, Group, Text, Anchor, ActionIcon } from '@mantine/core'
import { IconSettings, IconHistory } from '@tabler/icons-react'
import { Link } from 'react-router-dom'
import { SearchBox } from '../components/SearchBox'

export default function HomePage() {
  return (
    <div className="min-h-screen flex flex-col">
      {/* Header */}
      <header className="flex justify-end p-4 gap-4">
        <Link to="/history">
          <ActionIcon variant="subtle" color="gray" size="lg">
            <IconHistory size={20} />
          </ActionIcon>
        </Link>
        <Link to="/settings">
          <ActionIcon variant="subtle" color="gray" size="lg">
            <IconSettings size={20} />
          </ActionIcon>
        </Link>
      </header>

      {/* Main content */}
      <main className="flex-1 flex items-center justify-center -mt-20">
        <Container size="md" className="w-full">
          <Stack align="center" gap="xl">
            {/* Logo */}
            <div className="text-center">
              <Text
                size="72px"
                fw={700}
                className="tracking-tight"
                style={{
                  background: 'linear-gradient(90deg, #4285F4, #EA4335, #FBBC05, #34A853)',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                }}
              >
                Search
              </Text>
            </div>

            {/* Search box */}
            <SearchBox size="lg" autoFocus />

            {/* Quick links */}
            <Group gap="xl" mt="xl">
              <Anchor component={Link} to="/images" size="sm" c="dimmed">
                Images
              </Anchor>
              <Anchor component={Link} to="/search?time=day" size="sm" c="dimmed">
                News
              </Anchor>
              <Anchor component={Link} to="/settings" size="sm" c="dimmed">
                Settings
              </Anchor>
            </Group>
          </Stack>
        </Container>
      </main>

      {/* Footer */}
      <footer className="p-4 border-t bg-gray-50">
        <Container size="lg">
          <Group justify="space-between">
            <Group gap="lg">
              <Text size="xs" c="dimmed">Built with Go + React</Text>
            </Group>
            <Group gap="lg">
              <Anchor size="xs" c="dimmed" href="#">Privacy</Anchor>
              <Anchor size="xs" c="dimmed" href="#">Terms</Anchor>
            </Group>
          </Group>
        </Container>
      </footer>
    </div>
  )
}
