import * as React from "react"
import { ChevronLeft, ChevronRight } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "./button"

interface PaginationProps {
  page?: number
  currentPage?: number
  totalPages: number
  onPageChange: (page: number) => void
  className?: string
}

function Pagination({ page: pageProp, currentPage, totalPages, onPageChange, className }: PaginationProps) {
  const page = pageProp ?? currentPage ?? 0
  if (totalPages <= 1) return null

  return (
    <div
      className={cn(
        "flex items-center justify-between gap-4 pt-4",
        className
      )}
    >
      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 0}
        className="gap-1"
      >
        <ChevronLeft className="h-4 w-4" />
        Previous
      </Button>

      <span className="text-sm text-muted-foreground">
        Page {page + 1} of {totalPages}
      </span>

      <Button
        variant="outline"
        size="sm"
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages - 1}
        className="gap-1"
      >
        Next
        <ChevronRight className="h-4 w-4" />
      </Button>
    </div>
  )
}

export { Pagination }
