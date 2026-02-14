import * as React from "react"
import { cn } from "@/lib/utils"

interface TooltipProps {
  content: string
  children: React.ReactNode
  side?: "top" | "bottom"
  className?: string
}

function Tooltip({ content, children, side = "top", className }: TooltipProps) {
  return (
    <span
      className={cn("relative inline-flex group", className)}
      data-tooltip={content}
    >
      {children}
      <span
        className={cn(
          "pointer-events-none absolute left-1/2 -translate-x-1/2 z-50 whitespace-nowrap rounded-md border border-border bg-popover px-2 py-1 text-xs text-popover-foreground shadow-md opacity-0 transition-opacity group-hover:opacity-100",
          side === "top" ? "bottom-full mb-1.5" : "top-full mt-1.5"
        )}
      >
        {content}
      </span>
    </span>
  )
}

export { Tooltip }
