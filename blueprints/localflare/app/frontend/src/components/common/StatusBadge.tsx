import { Badge, Group, Box } from '@mantine/core'

interface StatusBadgeProps {
  status: 'active' | 'inactive' | 'error' | 'cached' | 'pending' | 'success' | 'failed' | 'running' | 'online' | 'offline' | 'degraded' | 'connected' | 'disconnected' | 'idle' | 'enabled' | 'disabled' | 'paused' | 'ok' | 'revoked' | 'ready' | 'pendingupload' | 'queued' | 'inprogress' | 'public' | 'private' | 'building'
  label?: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
}

const statusConfig: Record<StatusBadgeProps['status'], { color: string; label: string }> = {
  active: { color: 'green', label: 'Active' },
  inactive: { color: 'gray', label: 'Inactive' },
  error: { color: 'red', label: 'Error' },
  cached: { color: 'blue', label: 'Cached' },
  pending: { color: 'yellow', label: 'Pending' },
  success: { color: 'green', label: 'Success' },
  failed: { color: 'red', label: 'Failed' },
  running: { color: 'blue', label: 'Running' },
  online: { color: 'green', label: 'Online' },
  offline: { color: 'red', label: 'Offline' },
  degraded: { color: 'yellow', label: 'Degraded' },
  connected: { color: 'green', label: 'Connected' },
  disconnected: { color: 'red', label: 'Disconnected' },
  idle: { color: 'gray', label: 'Idle' },
  enabled: { color: 'green', label: 'Enabled' },
  disabled: { color: 'gray', label: 'Disabled' },
  paused: { color: 'yellow', label: 'Paused' },
  ok: { color: 'green', label: 'OK' },
  revoked: { color: 'red', label: 'Revoked' },
  ready: { color: 'green', label: 'Ready' },
  pendingupload: { color: 'yellow', label: 'Pending Upload' },
  queued: { color: 'blue', label: 'Queued' },
  inprogress: { color: 'blue', label: 'In Progress' },
  public: { color: 'green', label: 'Public' },
  private: { color: 'gray', label: 'Private' },
  building: { color: 'blue', label: 'Building' },
}

export function StatusBadge({ status, label, size = 'sm' }: StatusBadgeProps) {
  const config = statusConfig[status]

  return (
    <Badge
      size={size}
      color={config.color}
      variant="dot"
      radius="sm"
    >
      {label || config.label}
    </Badge>
  )
}

export function StatusDot({ status }: { status: StatusBadgeProps['status'] }) {
  const config = statusConfig[status]

  return (
    <Group gap={6}>
      <Box
        style={{
          width: 8,
          height: 8,
          borderRadius: '50%',
          backgroundColor: `var(--mantine-color-${config.color}-6)`,
        }}
      />
    </Group>
  )
}
