import { Badge } from "@/components/ui/badge"
import { statusColor } from "@/lib/utils"

interface StatusBadgeProps {
  status: string
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const code = parseInt(status)
  const variant = statusColor(status)

  let label = status
  if (code >= 200 && code < 300) label = status
  else if (code >= 300 && code < 400) label = `${status} redirect`
  else if (code >= 400 && code < 500) label = `${status} client error`
  else if (code >= 500) label = `${status} server error`

  return (
    <Badge variant={variant} className={className}>
      {label}
    </Badge>
  )
}
