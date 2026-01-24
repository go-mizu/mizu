import { Container, Stack, Text, Paper, Select, Switch, Button, Group, ActionIcon } from '@mantine/core'
import { IconArrowLeft } from '@tabler/icons-react'
import { Link } from 'react-router-dom'
import { useSearchStore } from '../stores/searchStore'

const REGIONS = [
  { value: 'us', label: 'United States' },
  { value: 'uk', label: 'United Kingdom' },
  { value: 'ca', label: 'Canada' },
  { value: 'au', label: 'Australia' },
  { value: 'de', label: 'Germany' },
  { value: 'fr', label: 'France' },
  { value: 'jp', label: 'Japan' },
  { value: 'in', label: 'India' },
  { value: 'br', label: 'Brazil' },
  { value: 'mx', label: 'Mexico' },
]

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'es', label: 'Spanish' },
  { value: 'fr', label: 'French' },
  { value: 'de', label: 'German' },
  { value: 'it', label: 'Italian' },
  { value: 'pt', label: 'Portuguese' },
  { value: 'ja', label: 'Japanese' },
  { value: 'zh', label: 'Chinese' },
  { value: 'ko', label: 'Korean' },
  { value: 'ar', label: 'Arabic' },
]

const RESULTS_PER_PAGE = [
  { value: '10', label: '10 results' },
  { value: '20', label: '20 results' },
  { value: '30', label: '30 results' },
  { value: '50', label: '50 results' },
]

const SAFE_SEARCH = [
  { value: 'off', label: 'Off' },
  { value: 'moderate', label: 'Moderate' },
  { value: 'strict', label: 'Strict' },
]

export default function SettingsPage() {
  const { settings, updateSettings, clearRecentSearches } = useSearchStore()

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white border-b">
        <Container size="md" className="py-4">
          <Group>
            <Link to="/">
              <ActionIcon variant="subtle" color="gray" size="lg">
                <IconArrowLeft size={20} />
              </ActionIcon>
            </Link>
            <Text size="xl" fw={600}>
              Search Settings
            </Text>
          </Group>
        </Container>
      </header>

      {/* Main content */}
      <main>
        <Container size="md" className="py-6">
          <Stack gap="lg">
            {/* Search preferences */}
            <Paper p="lg" withBorder>
              <Text fw={600} mb="md">
                Search Preferences
              </Text>

              <Stack gap="md">
                <Select
                  label="Region"
                  description="Show results relevant to your region"
                  value={settings.region}
                  onChange={(value) => updateSettings({ region: value || 'us' })}
                  data={REGIONS}
                />

                <Select
                  label="Language"
                  description="Preferred language for search results"
                  value={settings.language}
                  onChange={(value) => updateSettings({ language: value || 'en' })}
                  data={LANGUAGES}
                />

                <Select
                  label="Results per page"
                  description="Number of results to show per page"
                  value={String(settings.results_per_page)}
                  onChange={(value) => updateSettings({ results_per_page: parseInt(value || '10', 10) })}
                  data={RESULTS_PER_PAGE}
                />

                <Select
                  label="SafeSearch"
                  description="Filter explicit content from results"
                  value={settings.safe_search}
                  onChange={(value) => updateSettings({ safe_search: value || 'moderate' })}
                  data={SAFE_SEARCH}
                />
              </Stack>
            </Paper>

            {/* Display settings */}
            <Paper p="lg" withBorder>
              <Text fw={600} mb="md">
                Display Settings
              </Text>

              <Stack gap="md">
                <Switch
                  label="Open links in new tab"
                  description="Open search result links in a new browser tab"
                  checked={settings.open_in_new_tab}
                  onChange={(e) => updateSettings({ open_in_new_tab: e.currentTarget.checked })}
                />

                <Switch
                  label="Show instant answers"
                  description="Display instant answers for calculations, conversions, etc."
                  checked={settings.show_instant_answers}
                  onChange={(e) => updateSettings({ show_instant_answers: e.currentTarget.checked })}
                />

                <Switch
                  label="Show knowledge panels"
                  description="Display knowledge panels for people, places, and things"
                  checked={settings.show_knowledge_panel}
                  onChange={(e) => updateSettings({ show_knowledge_panel: e.currentTarget.checked })}
                />
              </Stack>
            </Paper>

            {/* Privacy */}
            <Paper p="lg" withBorder>
              <Text fw={600} mb="md">
                Privacy
              </Text>

              <Stack gap="md">
                <Switch
                  label="Save search history"
                  description="Keep a record of your searches for quick access"
                  checked={settings.save_history}
                  onChange={(e) => updateSettings({ save_history: e.currentTarget.checked })}
                />

                <Switch
                  label="Enable autocomplete"
                  description="Show search suggestions as you type"
                  checked={settings.autocomplete_enabled}
                  onChange={(e) => updateSettings({ autocomplete_enabled: e.currentTarget.checked })}
                />

                <div>
                  <Button
                    variant="outline"
                    color="red"
                    onClick={clearRecentSearches}
                  >
                    Clear search history
                  </Button>
                  <Text size="xs" c="dimmed" mt="xs">
                    This will remove all your recent searches
                  </Text>
                </div>
              </Stack>
            </Paper>

            {/* Domain preferences info */}
            <Paper p="lg" withBorder>
              <Text fw={600} mb="md">
                Domain Preferences
              </Text>

              <Text size="sm" c="dimmed">
                You can upvote, downvote, or block specific domains directly from search results.
                Upvoted domains will appear higher in results, downvoted domains will appear lower,
                and blocked domains will be hidden entirely.
              </Text>

              <Text size="sm" c="dimmed" mt="sm">
                Look for the domain preference icons next to each search result.
              </Text>
            </Paper>

            {/* Lenses info */}
            <Paper p="lg" withBorder>
              <Text fw={600} mb="md">
                Search Lenses
              </Text>

              <Text size="sm" c="dimmed">
                Search lenses allow you to filter results to specific types of content or domains.
                Use them to focus your search on programming resources, academic papers, news, or other categories.
              </Text>

              <Text size="sm" c="dimmed" mt="sm">
                Add a lens filter to your search using the filter dropdown on the results page.
              </Text>
            </Paper>
          </Stack>
        </Container>
      </main>
    </div>
  )
}
