import { cn } from "@/lib/utils"
import { Card, CardContent } from "@/components/ui/card"
import { TrendingUp, TrendingDown, Minus } from "lucide-react"

interface KPICardProps {
  label?: string
  title?: string
  value: string | number
  icon?: React.ReactNode
  trend?: {
    value: number
    label?: string
  }
  className?: string
}

export function KPICard({ label, title, value, icon, trend, className }: KPICardProps) {
  const displayLabel = label || title || ""
  const trendDirection =
    trend && trend.value > 0
      ? "up"
      : trend && trend.value < 0
        ? "down"
        : "flat"

  return (
    <Card className={cn("", className)}>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
              {displayLabel}
            </p>
            <p className="text-2xl font-semibold tracking-tight">{value}</p>
          </div>
          {icon && (
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-primary/10 text-primary">
              {icon}
            </div>
          )}
        </div>

        {trend && (
          <div className="mt-3 flex items-center gap-1.5 text-xs">
            {trendDirection === "up" && (
              <TrendingUp className="h-3.5 w-3.5 text-success" />
            )}
            {trendDirection === "down" && (
              <TrendingDown className="h-3.5 w-3.5 text-error" />
            )}
            {trendDirection === "flat" && (
              <Minus className="h-3.5 w-3.5 text-muted-foreground" />
            )}
            <span
              className={cn(
                "font-medium",
                trendDirection === "up" && "text-success",
                trendDirection === "down" && "text-error",
                trendDirection === "flat" && "text-muted-foreground"
              )}
            >
              {trend.value > 0 ? "+" : ""}
              {trend.value.toFixed(1)}%
            </span>
            {trend.label && (
              <span className="text-muted-foreground">{trend.label}</span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
