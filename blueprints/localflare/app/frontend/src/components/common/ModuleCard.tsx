import { useState, type ReactNode } from 'react'
import {
  Paper,
  Group,
  Text,
  ActionIcon,
  Collapse,
  Box,
  ThemeIcon,
  Tooltip,
  Divider,
} from '@mantine/core'
import { IconHelp, IconChevronDown, IconChevronUp, IconInfoCircle } from '@tabler/icons-react'

interface ModuleCardProps {
  /** Module title */
  title: string
  /** Optional description shown below title */
  description?: string
  /** Icon displayed next to title */
  icon?: ReactNode
  /** Inline help content shown in collapsible panel */
  helpContent?: ReactNode
  /** Control elements displayed on the right side of the header */
  controls?: ReactNode
  /** Main content of the module */
  children: ReactNode
  /** Whether the module content is collapsible */
  collapsible?: boolean
  /** Whether the module starts collapsed (only if collapsible) */
  defaultCollapsed?: boolean
  /** Optional footer content */
  footer?: ReactNode
  /** Custom padding for the content area */
  contentPadding?: string | number
  /** Whether to show a border */
  withBorder?: boolean
}

export function ModuleCard({
  title,
  description,
  icon,
  helpContent,
  controls,
  children,
  collapsible = false,
  defaultCollapsed = false,
  footer,
  contentPadding = 'md',
  withBorder = true,
}: ModuleCardProps) {
  const [helpOpen, setHelpOpen] = useState(false)
  const [contentOpen, setContentOpen] = useState(!defaultCollapsed)

  return (
    <Paper radius="md" withBorder={withBorder}>
      {/* Header */}
      <Box p="md" pb={0}>
        <Group justify="space-between" wrap="nowrap">
          <Group gap="sm" wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
            {collapsible && (
              <ActionIcon
                variant="subtle"
                size="sm"
                onClick={() => setContentOpen(!contentOpen)}
                aria-label={contentOpen ? 'Collapse' : 'Expand'}
              >
                {contentOpen ? <IconChevronUp size={16} /> : <IconChevronDown size={16} />}
              </ActionIcon>
            )}

            {icon && (
              <ThemeIcon variant="light" color="orange" size="md" radius="md">
                {icon}
              </ThemeIcon>
            )}

            <Box style={{ minWidth: 0, flex: 1 }}>
              <Group gap="xs" wrap="nowrap">
                <Text fw={600} size="sm" truncate>
                  {title}
                </Text>
                {helpContent && (
                  <Tooltip label={helpOpen ? 'Hide help' : 'Show help'} position="top">
                    <ActionIcon
                      variant="subtle"
                      size="xs"
                      onClick={() => setHelpOpen(!helpOpen)}
                      color={helpOpen ? 'orange' : 'gray'}
                      aria-label="Toggle help"
                    >
                      <IconHelp size={14} />
                    </ActionIcon>
                  </Tooltip>
                )}
              </Group>
              {description && (
                <Text size="xs" c="dimmed" lineClamp={2}>
                  {description}
                </Text>
              )}
            </Box>
          </Group>

          {controls && <Group gap="sm" wrap="nowrap">{controls}</Group>}
        </Group>
      </Box>

      {/* Inline Help Panel */}
      <Collapse in={helpOpen}>
        <Box px="md" pt="sm">
          <Paper
            p="sm"
            radius="sm"
            style={{
              backgroundColor: 'var(--mantine-color-marine-9)',
              borderLeft: '3px solid var(--mantine-color-marine-5)',
            }}
          >
            <Group gap="xs" align="flex-start" wrap="nowrap">
              <IconInfoCircle size={16} color="var(--mantine-color-marine-5)" style={{ flexShrink: 0, marginTop: 2 }} />
              <Box style={{ flex: 1 }}>
                {typeof helpContent === 'string' ? (
                  <Text size="sm" c="gray.3">
                    {helpContent}
                  </Text>
                ) : (
                  helpContent
                )}
              </Box>
            </Group>
          </Paper>
        </Box>
      </Collapse>

      {/* Content */}
      <Collapse in={contentOpen}>
        <Box p={contentPadding} pt="md">
          {children}
        </Box>
      </Collapse>

      {/* Footer */}
      {footer && contentOpen && (
        <>
          <Divider />
          <Box p="sm" bg="dark.7">
            {footer}
          </Box>
        </>
      )}
    </Paper>
  )
}

/** Preset for a module with a simple toggle control */
interface ToggleModuleCardProps extends Omit<ModuleCardProps, 'controls'> {
  enabled: boolean
  onToggle: (enabled: boolean) => void
}

export function ToggleModuleCard({
  enabled,
  onToggle,
  ...props
}: ToggleModuleCardProps) {
  return (
    <ModuleCard
      {...props}
      controls={
        <ActionIcon
          variant={enabled ? 'filled' : 'subtle'}
          color={enabled ? 'green' : 'gray'}
          size="sm"
          onClick={() => onToggle(!enabled)}
          aria-label={enabled ? 'Disable' : 'Enable'}
        >
          <Box
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              backgroundColor: enabled
                ? 'var(--mantine-color-green-5)'
                : 'var(--mantine-color-gray-5)',
            }}
          />
        </ActionIcon>
      }
    />
  )
}
