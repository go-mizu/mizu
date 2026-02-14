import { useState, useRef, useEffect, useCallback } from "react"
import { ChevronDown, Search } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Language } from "@/api/client"

interface LanguageSelectProps {
  languages: Language[]
  value: string
  onChange: (code: string) => void
  showAutoDetect?: boolean
  detectedLanguage?: string | null
  className?: string
}

export function LanguageSelect({
  languages,
  value,
  onChange,
  showAutoDetect = false,
  detectedLanguage,
  className,
}: LanguageSelectProps) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState("")
  const containerRef = useRef<HTMLDivElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)

  const handleClickOutside = useCallback((e: MouseEvent) => {
    if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
      setOpen(false)
      setSearch("")
    }
  }, [])

  useEffect(() => {
    if (open) {
      document.addEventListener("mousedown", handleClickOutside)
      setTimeout(() => searchRef.current?.focus(), 0)
    }
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [open, handleClickOutside])

  // Source panel: show auto (already in list from API). Target panel: filter it out.
  const allOptions: Language[] = showAutoDetect
    ? languages
    : languages.filter((l) => l.code !== "auto")

  const filtered = search
    ? allOptions.filter(
        (l) =>
          l.name.toLowerCase().includes(search.toLowerCase()) ||
          l.code.toLowerCase().includes(search.toLowerCase())
      )
    : allOptions

  const selectedLabel =
    value === "auto"
      ? detectedLanguage
        ? `Auto-detect (${detectedLanguage})`
        : "Auto-detect"
      : languages.find((l) => l.code === value)?.name || value

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex h-9 w-full items-center justify-between rounded-md border border-input bg-background px-3 py-2 text-sm shadow-sm hover:bg-secondary/50 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
      >
        <span className="truncate">{selectedLabel}</span>
        <ChevronDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
      </button>

      {open && (
        <div className="absolute z-50 mt-1 w-full min-w-[220px] rounded-md border border-border bg-popover shadow-lg">
          <div className="flex items-center border-b border-border px-3 py-2">
            <Search className="mr-2 h-4 w-4 shrink-0 opacity-50" />
            <input
              ref={searchRef}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search languages..."
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
            />
          </div>
          <div className="max-h-[280px] overflow-y-auto p-1">
            {filtered.length === 0 ? (
              <div className="px-3 py-6 text-center text-sm text-muted-foreground">
                No language found.
              </div>
            ) : (
              filtered.map((lang) => (
                <button
                  key={lang.code}
                  type="button"
                  onClick={() => {
                    onChange(lang.code)
                    setOpen(false)
                    setSearch("")
                  }}
                  className={cn(
                    "flex w-full items-center justify-between rounded-sm px-3 py-1.5 text-sm hover:bg-secondary",
                    value === lang.code && "bg-secondary font-medium"
                  )}
                >
                  <span>{lang.name}</span>
                  <span className="text-xs text-muted-foreground">{lang.code}</span>
                </button>
              ))
            )}
          </div>
        </div>
      )}
    </div>
  )
}
