import { useMemo } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import {
  Group, Title, Text, Button, Paper, SimpleGrid, Box, Loader,
  Breadcrumbs, Anchor, ActionIcon, ThemeIcon, Tooltip, Stack, Menu, Divider
} from '@mantine/core'
import {
  IconArrowLeft, IconBolt, IconDeviceFloppy, IconDotsVertical,
  IconZoomIn, IconZoomOut, IconLink, IconDatabase, IconRefresh
} from '@tabler/icons-react'
import { PageContainer, EmptyState, LoadingState } from '../components/ui'
import Visualization from '../components/visualizations'
import { useDataSource, useTables } from '../api/hooks'
import { useXRayTable, useXRayField, useSaveXRay } from '../api/hooks/xray'

export default function XRay() {
  const navigate = useNavigate()
  const { datasourceId, tableId, columnId } = useParams()
  const [searchParams] = useSearchParams()
  const type = searchParams.get('type') || 'table'

  // Fetch metadata
  const { data: datasource, isLoading: loadingDs } = useDataSource(datasourceId || '')
  const { data: tables } = useTables(datasourceId || '')
  const table = tables?.find(t => t.id === tableId)

  // Fetch X-ray data
  const {
    data: tableXRay,
    isLoading: loadingTableXRay,
    refetch: refetchTableXRay,
    isFetching: fetchingTableXRay
  } = useXRayTable(datasourceId || '', tableId || '', type === 'table')

  const {
    data: fieldXRay,
    isLoading: loadingFieldXRay,
    refetch: refetchFieldXRay,
    isFetching: fetchingFieldXRay
  } = useXRayField(datasourceId || '', columnId || '', type === 'field')

  const xray = type === 'field' ? fieldXRay : tableXRay
  const isLoading = type === 'field' ? loadingFieldXRay : loadingTableXRay
  const isFetching = type === 'field' ? fetchingFieldXRay : fetchingTableXRay

  const saveXRay = useSaveXRay()

  const handleSave = async () => {
    if (!xray) return
    try {
      const result = await saveXRay.mutateAsync({ xray })
      navigate(`/dashboard/${result.dashboard_id}`)
    } catch (error) {
      console.error('Failed to save X-ray:', error)
    }
  }

  const handleRefresh = () => {
    if (type === 'field') {
      refetchFieldXRay()
    } else {
      refetchTableXRay()
    }
  }

  if (loadingDs || isLoading) {
    return (
      <PageContainer>
        <LoadingState message="Generating X-ray insights..." />
      </PageContainer>
    )
  }

  if (!datasource) {
    return (
      <PageContainer>
        <EmptyState
          icon={<IconDatabase size={32} strokeWidth={1.5} />}
          iconColor="var(--color-warning)"
          title="Database not found"
          description="The requested database could not be found."
          action={<Button onClick={() => navigate('/browse/databases')}>Back to Databases</Button>}
        />
      </PageContainer>
    )
  }

  if (!xray) {
    return (
      <PageContainer>
        <EmptyState
          icon={<IconBolt size={32} strokeWidth={1.5} />}
          iconColor="var(--color-brand)"
          title="Unable to generate X-ray"
          description="Could not generate insights for this data."
          action={<Button onClick={() => navigate(-1)}>Go Back</Button>}
        />
      </PageContainer>
    )
  }

  return (
    <PageContainer>
      {/* Header */}
      <Group justify="space-between" mb="xl">
        <div>
          <Breadcrumbs mb="xs">
            <Anchor onClick={() => navigate('/browse/databases')}>Databases</Anchor>
            <Anchor onClick={() => navigate(`/browse/database/${datasourceId}`)}>
              {datasource.name}
            </Anchor>
            {table && (
              <Anchor onClick={() => navigate(`/browse/database/${datasourceId}/table/${tableId}`)}>
                {table.display_name || table.name}
              </Anchor>
            )}
            <Text fw={600}>X-ray</Text>
          </Breadcrumbs>
          <Group gap="sm">
            <ActionIcon variant="subtle" onClick={() => navigate(-1)}>
              <IconArrowLeft size={20} />
            </ActionIcon>
            <ThemeIcon size={40} radius="md" variant="gradient" gradient={{ from: 'yellow', to: 'orange' }}>
              <IconBolt size={20} />
            </ThemeIcon>
            <div>
              <Title order={2}>{xray.title}</Title>
              <Text size="sm" c="dimmed">{xray.description}</Text>
            </div>
          </Group>
        </div>
        <Group gap="sm">
          <Tooltip label="Refresh insights">
            <ActionIcon
              variant="light"
              size="lg"
              onClick={handleRefresh}
              loading={isFetching}
            >
              <IconRefresh size={18} />
            </ActionIcon>
          </Tooltip>
          <Button
            leftSection={<IconDeviceFloppy size={16} />}
            onClick={handleSave}
            loading={saveXRay.isPending}
          >
            Save as Dashboard
          </Button>
        </Group>
      </Group>

      {/* Stats Summary */}
      {xray.stats && (
        <SimpleGrid cols={{ base: 2, sm: 4 }} mb="xl">
          <Paper withBorder p="md" radius="md">
            <Text size="xs" c="dimmed" tt="uppercase" fw={500}>Total Records</Text>
            <Text size="xl" fw={700}>{xray.stats.row_count?.toLocaleString() || '-'}</Text>
          </Paper>
          <Paper withBorder p="md" radius="md">
            <Text size="xs" c="dimmed" tt="uppercase" fw={500}>Columns</Text>
            <Text size="xl" fw={700}>{xray.stats.column_count || '-'}</Text>
          </Paper>
          <Paper withBorder p="md" radius="md">
            <Text size="xs" c="dimmed" tt="uppercase" fw={500}>Nullable Columns</Text>
            <Text size="xl" fw={700}>{xray.stats.nullable_count || '-'}</Text>
          </Paper>
          <Paper withBorder p="md" radius="md">
            <Text size="xs" c="dimmed" tt="uppercase" fw={500}>Generated</Text>
            <Text size="sm" mt={4}>
              {new Date(xray.generated_at).toLocaleString()}
            </Text>
          </Paper>
        </SimpleGrid>
      )}

      {/* Navigation Sidebar + Cards Grid */}
      <Group align="flex-start" gap="xl">
        {/* Navigation Panel */}
        {xray.navigation && xray.navigation.length > 0 && (
          <Paper withBorder p="md" radius="md" w={250} style={{ flexShrink: 0 }}>
            <Text size="sm" fw={600} mb="md">Explore Further</Text>
            <Stack gap="xs">
              {xray.navigation.filter(n => n.type === 'zoom_out').length > 0 && (
                <>
                  <Text size="xs" c="dimmed" tt="uppercase">Zoom Out</Text>
                  {xray.navigation.filter(n => n.type === 'zoom_out').map((nav, i) => (
                    <Button
                      key={`out-${i}`}
                      variant="subtle"
                      size="xs"
                      leftSection={<IconZoomOut size={14} />}
                      onClick={() => {
                        if (nav.target_type === 'table') {
                          navigate(`/xray/${datasourceId}/table/${nav.target_id}`)
                        }
                      }}
                      justify="flex-start"
                      fullWidth
                    >
                      {nav.label}
                    </Button>
                  ))}
                </>
              )}

              {xray.navigation.filter(n => n.type === 'zoom_in').slice(0, 5).length > 0 && (
                <>
                  <Divider my="xs" />
                  <Text size="xs" c="dimmed" tt="uppercase">Zoom In</Text>
                  {xray.navigation.filter(n => n.type === 'zoom_in').slice(0, 5).map((nav, i) => (
                    <Button
                      key={`in-${i}`}
                      variant="subtle"
                      size="xs"
                      leftSection={<IconZoomIn size={14} />}
                      onClick={() => {
                        if (nav.target_type === 'field') {
                          navigate(`/xray/${datasourceId}/field/${nav.target_id}?type=field`)
                        }
                      }}
                      justify="flex-start"
                      fullWidth
                    >
                      {nav.label}
                    </Button>
                  ))}
                  {xray.navigation.filter(n => n.type === 'zoom_in').length > 5 && (
                    <Text size="xs" c="dimmed" ta="center">
                      +{xray.navigation.filter(n => n.type === 'zoom_in').length - 5} more fields
                    </Text>
                  )}
                </>
              )}

              {xray.navigation.filter(n => n.type === 'related').length > 0 && (
                <>
                  <Divider my="xs" />
                  <Text size="xs" c="dimmed" tt="uppercase">Related</Text>
                  {xray.navigation.filter(n => n.type === 'related').map((nav, i) => (
                    <Button
                      key={`rel-${i}`}
                      variant="subtle"
                      size="xs"
                      leftSection={<IconLink size={14} />}
                      onClick={() => {
                        // For related tables, we'd need to resolve the table ID
                        console.log('Navigate to related:', nav)
                      }}
                      justify="flex-start"
                      fullWidth
                    >
                      {nav.label}
                    </Button>
                  ))}
                </>
              )}
            </Stack>
          </Paper>
        )}

        {/* Cards Grid */}
        <Box style={{ flex: 1, minWidth: 0 }}>
          <XRayCardsGrid cards={xray.cards || []} />
        </Box>
      </Group>
    </PageContainer>
  )
}

// X-Ray Cards Grid Component
function XRayCardsGrid({ cards }: { cards: XRayCard[] }) {
  // Group cards by row
  const cardsByRow = useMemo(() => {
    const rows: Record<number, XRayCard[]> = {}
    cards.forEach(card => {
      const row = card.row || 0
      if (!rows[row]) rows[row] = []
      rows[row].push(card)
    })
    // Sort cards within each row by column
    Object.values(rows).forEach(rowCards => {
      rowCards.sort((a, b) => (a.col || 0) - (b.col || 0))
    })
    return rows
  }, [cards])

  const rowNumbers = Object.keys(cardsByRow).map(Number).sort((a, b) => a - b)

  if (cards.length === 0) {
    return (
      <Paper withBorder p="xl" ta="center">
        <Text c="dimmed">No insights generated</Text>
      </Paper>
    )
  }

  return (
    <Stack gap="md">
      {rowNumbers.map(rowNum => (
        <Group key={rowNum} gap="md" align="stretch" grow>
          {cardsByRow[rowNum].map(card => (
            <XRayCardComponent key={card.id} card={card} />
          ))}
        </Group>
      ))}
    </Stack>
  )
}

// Individual X-Ray Card Component
function XRayCardComponent({ card }: { card: XRayCard }) {
  const heightMultiplier = 80 // pixels per height unit
  const minHeight = (card.height || 2) * heightMultiplier

  // Convert card data to visualization format
  const visualizationData = useMemo(() => {
    if (!card.data) return null

    return {
      columns: card.data.columns || [],
      rows: card.data.rows || [],
      row_count: card.data.row_count || 0,
      duration_ms: 0
    }
  }, [card.data])

  return (
    <Paper
      withBorder
      p="md"
      radius="md"
      style={{
        minHeight,
        flex: card.width || 6,
        display: 'flex',
        flexDirection: 'column'
      }}
    >
      {/* Card Header */}
      <Group justify="space-between" mb="sm">
        <div>
          <Text size="sm" fw={600}>{card.title}</Text>
          {card.description && (
            <Text size="xs" c="dimmed">{card.description}</Text>
          )}
        </div>
        <Menu withinPortal position="bottom-end">
          <Menu.Target>
            <ActionIcon variant="subtle" size="sm">
              <IconDotsVertical size={14} />
            </ActionIcon>
          </Menu.Target>
          <Menu.Dropdown>
            <Menu.Item leftSection={<IconZoomIn size={14} />}>
              Explore
            </Menu.Item>
            <Menu.Item leftSection={<IconDeviceFloppy size={14} />}>
              Save as Question
            </Menu.Item>
          </Menu.Dropdown>
        </Menu>
      </Group>

      {/* Card Content */}
      <Box style={{ flex: 1, minHeight: minHeight - 60 }}>
        {card.data && visualizationData ? (
          <Visualization
            result={visualizationData}
            visualization={{
              type: card.visualization as any || 'table',
              settings: card.settings || {}
            }}
            height={minHeight - 60}
            showLegend={false}
          />
        ) : (
          <Box
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              height: '100%'
            }}
          >
            <Loader size="sm" />
          </Box>
        )}
      </Box>

      {/* Query preview (collapsed) */}
      {card.query && (
        <Tooltip label={card.query} multiline w={400}>
          <Text
            size="xs"
            c="dimmed"
            mt="sm"
            style={{
              fontFamily: 'monospace',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap'
            }}
          >
            {card.query.length > 60 ? card.query.slice(0, 60) + '...' : card.query}
          </Text>
        </Tooltip>
      )}
    </Paper>
  )
}

// Types
interface XRayCard {
  id: string
  title: string
  description?: string
  visualization: string
  query?: string
  data?: {
    columns: { name: string; display_name: string; type: string }[]
    rows: Record<string, any>[]
    row_count: number
  }
  width?: number
  height?: number
  row?: number
  col?: number
  settings?: Record<string, any>
}
