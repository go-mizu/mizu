import { useState, useCallback } from "react"
import { NavLink, useNavigate } from "react-router-dom"
import {
  Search,
  Database,
  Globe,
  Link2,
  FileText,
  FileCode,
  FileType,
  Bot,
  BarChart3,
  GitGraph,
  Sparkles,
  Newspaper,
  Sun,
  Moon,
  Monitor,
  Github,
  Menu,
  X,
  ChevronDown,
  ChevronRight,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useTheme } from "./theme-provider"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"

interface NavItem {
  label: string
  to: string
  icon: React.ReactNode
}

interface NavSection {
  title: string
  items: NavItem[]
}

const sections: NavSection[] = [
  {
    title: "BROWSE",
    items: [
      { label: "Crawls", to: "/crawls", icon: <Database className="h-4 w-4" /> },
      { label: "Domains", to: "/domains", icon: <Globe className="h-4 w-4" /> },
      { label: "URL Lookup", to: "/url", icon: <Link2 className="h-4 w-4" /> },
    ],
  },
  {
    title: "DATA",
    items: [
      { label: "WARC Viewer", to: "/view", icon: <FileText className="h-4 w-4" /> },
      { label: "WAT Browser", to: "/wat", icon: <FileCode className="h-4 w-4" /> },
      { label: "WET Browser", to: "/wet", icon: <FileType className="h-4 w-4" /> },
      { label: "Robots.txt", to: "/robots", icon: <Bot className="h-4 w-4" /> },
    ],
  },
  {
    title: "ANALYTICS",
    items: [
      { label: "Statistics", to: "/stats", icon: <BarChart3 className="h-4 w-4" /> },
      { label: "Web Graph", to: "/graph", icon: <GitGraph className="h-4 w-4" /> },
      { label: "Quality", to: "/quality", icon: <Sparkles className="h-4 w-4" /> },
    ],
  },
  {
    title: "NEWS",
    items: [
      { label: "CC-NEWS", to: "/news", icon: <Newspaper className="h-4 w-4" /> },
    ],
  },
]

export function Sidebar() {
  const [isOpen, setIsOpen] = useState(false)
  const [collapsedSections, setCollapsedSections] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState("")
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()

  const toggleSection = useCallback((title: string) => {
    setCollapsedSections((prev) => {
      const next = new Set(prev)
      if (next.has(title)) {
        next.delete(title)
      } else {
        next.add(title)
      }
      return next
    })
  }, [])

  const handleSearch = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      const q = searchQuery.trim()
      if (!q) return
      navigate(`/search?q=${encodeURIComponent(q)}`)
      setSearchQuery("")
      setIsOpen(false)
    },
    [searchQuery, navigate]
  )

  const cycleTheme = useCallback(() => {
    const order: Array<"light" | "dark" | "system"> = ["light", "dark", "system"]
    const idx = order.indexOf(theme)
    setTheme(order[(idx + 1) % order.length])
  }, [theme, setTheme])

  const themeIcon =
    theme === "dark" ? (
      <Moon className="h-4 w-4" />
    ) : theme === "light" ? (
      <Sun className="h-4 w-4" />
    ) : (
      <Monitor className="h-4 w-4" />
    )

  const sidebarContent = (
    <div className="flex h-full flex-col">
      {/* Logo */}
      <div className="flex h-14 items-center gap-2 px-4 border-b border-sidebar-border">
        <NavLink
          to="/"
          className="flex items-center gap-2 font-semibold text-sidebar-foreground"
          onClick={() => setIsOpen(false)}
        >
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground text-xs font-bold">
            CC
          </div>
          <span className="text-sm">Common Crawl Viewer</span>
        </NavLink>
      </div>

      {/* Search */}
      <div className="p-3">
        <form onSubmit={handleSearch}>
          <div className="relative">
            <Search className="absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-sidebar-muted" />
            <Input
              type="text"
              placeholder="Search URL or domain..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="h-8 pl-8 text-xs bg-background border-sidebar-border"
            />
          </div>
        </form>
      </div>

      {/* Navigation */}
      <nav className="flex-1 overflow-y-auto px-2 pb-2">
        {sections.map((section) => {
          const isCollapsed = collapsedSections.has(section.title)
          return (
            <div key={section.title} className="mb-1">
              <button
                onClick={() => toggleSection(section.title)}
                className="flex w-full items-center gap-1 px-2 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-sidebar-muted hover:text-sidebar-foreground transition-colors"
              >
                {isCollapsed ? (
                  <ChevronRight className="h-3 w-3" />
                ) : (
                  <ChevronDown className="h-3 w-3" />
                )}
                {section.title}
              </button>
              {!isCollapsed && (
                <div className="space-y-0.5">
                  {section.items.map((item) => (
                    <NavLink
                      key={item.to}
                      to={item.to}
                      onClick={() => setIsOpen(false)}
                      className={({ isActive }) =>
                        cn(
                          "flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors",
                          isActive
                            ? "bg-primary/10 text-primary font-medium"
                            : "text-sidebar-foreground/80 hover:bg-sidebar-border/50 hover:text-sidebar-foreground"
                        )
                      }
                    >
                      {item.icon}
                      {item.label}
                    </NavLink>
                  ))}
                </div>
              )}
            </div>
          )
        })}
      </nav>

      {/* Footer */}
      <div className="border-t border-sidebar-border p-2">
        <div className="flex items-center justify-between px-1">
          <Button
            variant="ghost"
            size="icon"
            onClick={cycleTheme}
            className="h-8 w-8 text-sidebar-muted hover:text-sidebar-foreground"
            data-tooltip={`Theme: ${theme}`}
          >
            {themeIcon}
          </Button>
          <a
            href="https://github.com/commoncrawl"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex h-8 w-8 items-center justify-center rounded-md text-sidebar-muted hover:text-sidebar-foreground hover:bg-sidebar-border/50 transition-colors"
          >
            <Github className="h-4 w-4" />
          </a>
        </div>
      </div>
    </div>
  )

  return (
    <>
      {/* Mobile hamburger */}
      <Button
        variant="ghost"
        size="icon"
        className="fixed top-3 left-3 z-50 md:hidden"
        onClick={() => setIsOpen(!isOpen)}
      >
        {isOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
      </Button>

      {/* Mobile overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 md:hidden"
          onClick={() => setIsOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          "fixed inset-y-0 left-0 z-40 w-60 border-r border-sidebar-border bg-sidebar transition-transform duration-200 md:static md:translate-x-0",
          isOpen ? "translate-x-0" : "-translate-x-full"
        )}
      >
        {sidebarContent}
      </aside>
    </>
  )
}
