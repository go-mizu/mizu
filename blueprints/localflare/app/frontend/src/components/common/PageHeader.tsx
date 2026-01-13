import { Group, Title, ActionIcon, Text, Stack, Breadcrumbs } from '@mantine/core'
import { IconArrowLeft } from '@tabler/icons-react'
import { useNavigate, Link } from 'react-router-dom'
import type { ReactNode } from 'react'

interface Breadcrumb {
  label: string
  path?: string
}

interface PageHeaderProps {
  title: string
  subtitle?: string
  breadcrumbs?: Breadcrumb[]
  backPath?: string
  actions?: ReactNode
}

export function PageHeader({ title, subtitle, breadcrumbs, backPath, actions }: PageHeaderProps) {
  const navigate = useNavigate()

  return (
    <Stack gap="xs" mb="lg">
      {breadcrumbs && breadcrumbs.length > 0 && (
        <Breadcrumbs separator="/">
          {breadcrumbs.map((crumb, idx) => (
            crumb.path ? (
              <Link
                key={idx}
                to={crumb.path}
                style={{
                  fontSize: 'var(--mantine-font-size-sm)',
                  color: 'var(--mantine-color-dimmed)',
                  textDecoration: 'none',
                }}
              >
                {crumb.label}
              </Link>
            ) : (
              <Text key={idx} size="sm" c="dimmed">
                {crumb.label}
              </Text>
            )
          ))}
        </Breadcrumbs>
      )}

      <Group justify="space-between" align="flex-start">
        <Group gap="md">
          {backPath && (
            <ActionIcon
              variant="subtle"
              size="lg"
              onClick={() => navigate(backPath)}
              aria-label="Go back"
            >
              <IconArrowLeft size={20} />
            </ActionIcon>
          )}
          <Stack gap={2}>
            <Title order={2}>{title}</Title>
            {subtitle && (
              <Text size="sm" c="dimmed">
                {subtitle}
              </Text>
            )}
          </Stack>
        </Group>
        {actions && <Group gap="sm">{actions}</Group>}
      </Group>
    </Stack>
  )
}
