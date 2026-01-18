import { Badge } from '@mantine/core';

type StatusType = 'success' | 'warning' | 'error' | 'info' | 'default' | 'active' | 'inactive';

interface StatusBadgeProps {
  status: StatusType | string;
  label?: string;
}

const statusConfig: Record<StatusType, { color: string; label: string }> = {
  success: { color: 'green', label: 'Success' },
  warning: { color: 'yellow', label: 'Warning' },
  error: { color: 'red', label: 'Error' },
  info: { color: 'blue', label: 'Info' },
  default: { color: 'gray', label: 'Default' },
  active: { color: 'green', label: 'Active' },
  inactive: { color: 'gray', label: 'Inactive' },
};

export function StatusBadge({ status, label }: StatusBadgeProps) {
  const config = statusConfig[status as StatusType] || statusConfig.default;

  return (
    <Badge
      variant="light"
      color={config.color}
      size="sm"
      style={{
        textTransform: 'none',
        fontWeight: 500,
      }}
    >
      {label || config.label}
    </Badge>
  );
}

// Specialized badges
export function PublicBadge({ isPublic }: { isPublic: boolean }) {
  return (
    <Badge
      variant="light"
      color={isPublic ? 'blue' : 'gray'}
      size="sm"
      style={{
        textTransform: 'none',
        fontWeight: 500,
      }}
    >
      {isPublic ? 'Public' : 'Private'}
    </Badge>
  );
}

export function VerifiedBadge({ verified }: { verified: boolean }) {
  return verified ? (
    <Badge
      variant="light"
      color="green"
      size="xs"
      style={{
        textTransform: 'none',
        fontWeight: 500,
      }}
    >
      Verified
    </Badge>
  ) : (
    <Badge
      variant="light"
      color="yellow"
      size="xs"
      style={{
        textTransform: 'none',
        fontWeight: 500,
      }}
    >
      Unverified
    </Badge>
  );
}

export function RoleBadge({ role }: { role: string }) {
  const color = role === 'service_role' ? 'red' : role === 'authenticated' ? 'blue' : 'gray';
  return (
    <Badge
      variant="light"
      color={color}
      size="sm"
      style={{
        textTransform: 'none',
        fontWeight: 500,
      }}
    >
      {role}
    </Badge>
  );
}
