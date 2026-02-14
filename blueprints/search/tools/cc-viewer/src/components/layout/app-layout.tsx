import type { ReactNode } from "react"
import { Sidebar } from "./sidebar"
import { ScrollArea } from "@/components/ui/scroll-area"

interface AppLayoutProps {
  children: ReactNode
}

export function AppLayout({ children }: AppLayoutProps) {
  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar />
      <ScrollArea className="flex-1">
        <main className="mx-auto max-w-6xl px-6 py-6 md:px-8 md:py-8">
          {children}
        </main>
      </ScrollArea>
    </div>
  )
}
