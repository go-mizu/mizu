import { useState } from "react"
import { ChevronDown } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Definition, SynonymGroup } from "@/api/client"

interface CollapsibleSectionProps {
  title: string
  defaultOpen?: boolean
  children: React.ReactNode
}

function CollapsibleSection({ title, defaultOpen = true, children }: CollapsibleSectionProps) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="border-t border-border">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-4 py-2.5 text-sm font-medium hover:bg-secondary/50 transition-colors"
      >
        {title}
        <ChevronDown
          className={cn("h-4 w-4 transition-transform", open && "rotate-180")}
        />
      </button>
      {open && <div className="px-4 pb-3">{children}</div>}
    </div>
  )
}

interface RichPanelProps {
  definitions: Definition[] | null
  synonyms: SynonymGroup[] | null
  examples: string[] | null
}

export function RichPanel({ definitions, synonyms, examples }: RichPanelProps) {
  const hasDefs = definitions && definitions.length > 0
  const hasSyns = synonyms && synonyms.length > 0
  const hasExamples = examples && examples.length > 0

  if (!hasDefs && !hasSyns && !hasExamples) return null

  return (
    <div className="mt-4 rounded-lg border border-border bg-card overflow-hidden">
      {hasDefs && (
        <CollapsibleSection title="Definitions">
          <div className="space-y-3">
            {definitions!.map((group, gi) => (
              <div key={gi}>
                <span className="inline-block rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground mb-1.5">
                  {group.partOfSpeech}
                </span>
                <ol className="list-decimal list-inside space-y-1.5 text-sm">
                  {group.entries.map((entry, ei) => (
                    <li key={ei}>
                      {entry.definition}
                      {entry.example && (
                        <p className="ml-5 mt-0.5 text-muted-foreground italic">
                          &ldquo;{entry.example}&rdquo;
                        </p>
                      )}
                    </li>
                  ))}
                </ol>
              </div>
            ))}
          </div>
        </CollapsibleSection>
      )}

      {hasSyns && (
        <CollapsibleSection title="Synonyms">
          <div className="space-y-3">
            {synonyms!.map((group, gi) => (
              <div key={gi}>
                <span className="inline-block rounded bg-secondary px-2 py-0.5 text-xs font-medium text-secondary-foreground mb-1.5">
                  {group.partOfSpeech}
                </span>
                <div className="flex flex-wrap gap-1.5">
                  {group.entries.flat().map((word, wi) => (
                    <span
                      key={wi}
                      className="inline-block rounded-full border border-border bg-background px-2.5 py-0.5 text-xs"
                    >
                      {word}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </CollapsibleSection>
      )}

      {hasExamples && (
        <CollapsibleSection title="Examples">
          <ul className="list-disc list-inside space-y-1.5 text-sm">
            {examples!.map((ex, i) => (
              <li key={i} dangerouslySetInnerHTML={{ __html: ex }} />
            ))}
          </ul>
        </CollapsibleSection>
      )}
    </div>
  )
}
