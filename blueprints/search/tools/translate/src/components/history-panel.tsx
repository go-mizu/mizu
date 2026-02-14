import { X, Trash2, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useTranslateStore, type HistoryEntry } from "@/stores/translate"

export function HistoryPanel() {
  const { history, removeFromHistory, clearHistory, restoreFromHistory } = useTranslateStore()

  if (history.length === 0) return null

  return (
    <div className="mt-8">
      <div className="flex items-center justify-between mb-3">
        <h2 className="text-sm font-medium text-muted-foreground">Recent translations</h2>
        <Button variant="ghost" size="sm" onClick={clearHistory} className="text-xs text-muted-foreground hover:text-destructive">
          <Trash2 className="h-3.5 w-3.5 mr-1" />
          Clear all
        </Button>
      </div>
      <div className="space-y-2">
        {history.map((entry: HistoryEntry) => (
          <div
            key={entry.id}
            className="group flex items-start gap-3 rounded-lg border border-border bg-card p-3 hover:bg-secondary/30 cursor-pointer transition-colors"
            onClick={() => restoreFromHistory(entry)}
          >
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 text-xs text-muted-foreground mb-1">
                <span>{entry.sourceLang}</span>
                <ArrowRight className="h-3 w-3" />
                <span>{entry.targetLang}</span>
              </div>
              <p className="text-sm truncate">{entry.sourceText}</p>
              <p className="text-sm text-muted-foreground truncate">{entry.translation}</p>
            </div>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation()
                removeFromHistory(entry.id)
              }}
              className="opacity-0 group-hover:opacity-100 p-1 rounded hover:bg-destructive/10 hover:text-destructive transition-all"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}
