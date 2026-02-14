import { useState, useEffect, useCallback, useRef } from "react"
import { useQuery } from "@tanstack/react-query"
import { Languages, ArrowLeftRight, Copy, Check, Moon, Sun, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Skeleton } from "@/components/ui/skeleton"
import { LanguageSelect } from "@/components/language-select"
import { AudioButton } from "@/components/audio-button"
import { RichPanel } from "@/components/rich-panel"
import { HistoryPanel } from "@/components/history-panel"
import { translateText, fetchLanguages } from "@/api/client"
import { useTranslateStore } from "@/stores/translate"

// --- Theme hook ---
function useTheme() {
  const [dark, setDark] = useState(() => {
    if (typeof window === "undefined") return false
    const stored = localStorage.getItem("theme")
    if (stored) return stored === "dark"
    return window.matchMedia("(prefers-color-scheme: dark)").matches
  })

  useEffect(() => {
    const root = document.documentElement
    if (dark) {
      root.classList.add("dark")
    } else {
      root.classList.remove("dark")
    }
    localStorage.setItem("theme", dark ? "dark" : "light")
  }, [dark])

  return { dark, toggle: () => setDark((d) => !d) }
}

// --- Translator component ---
function Translator() {
  const {
    sourceText, sourceLang, targetLang, result, loading, error,
    setSourceText, setSourceLang, setTargetLang, swapLanguages,
    setResult, setLoading, setError, addToHistory,
  } = useTranslateStore()

  const [copied, setCopied] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const { data: languages = [] } = useQuery({
    queryKey: ["languages"],
    queryFn: fetchLanguages,
  })

  // Debounced auto-translate
  useEffect(() => {
    if (timerRef.current) clearTimeout(timerRef.current)

    if (!sourceText.trim()) {
      setResult(null)
      setError(null)
      return
    }

    timerRef.current = setTimeout(async () => {
      setLoading(true)
      setError(null)
      try {
        const res = await translateText(sourceText, sourceLang, targetLang)
        setResult(res)
        addToHistory({
          id: crypto.randomUUID(),
          sourceText: sourceText.slice(0, 200),
          translation: res.translation.slice(0, 200),
          sourceLang: res.detectedLanguage || sourceLang,
          targetLang,
          timestamp: Date.now(),
        })
      } catch (err) {
        setError(err instanceof Error ? err.message : "Translation failed")
        setResult(null)
      } finally {
        setLoading(false)
      }
    }, 500)

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sourceText, sourceLang, targetLang])

  const handleCopy = useCallback(async () => {
    if (!result?.translation) return
    await navigator.clipboard.writeText(result.translation)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }, [result])

  const detectedLangName = result?.detectedLanguage
    ? languages.find((l) => l.code === result.detectedLanguage)?.name || result.detectedLanguage
    : null

  return (
    <div>
      <div className="grid grid-cols-1 md:grid-cols-[1fr_auto_1fr] gap-4">
        {/* Source panel */}
        <div className="space-y-2">
          <LanguageSelect
            languages={languages}
            value={sourceLang}
            onChange={setSourceLang}
            showAutoDetect
            detectedLanguage={detectedLangName}
          />
          <Textarea
            value={sourceText}
            onChange={(e) => { if (e.target.value.length <= 5000) setSourceText(e.target.value) }}
            maxLength={5000}
            placeholder="Enter text to translate..."
            className="min-h-[180px]"
          />
          <div className="flex items-center justify-between">
            <AudioButton
              lang={sourceLang === "auto" ? (result?.detectedLanguage || "en") : sourceLang}
              text={sourceText}
            />
            <span className="text-xs text-muted-foreground">
              {sourceText.length.toLocaleString()}/5,000
            </span>
          </div>
        </div>

        {/* Swap button */}
        <div className="flex items-center justify-center md:pt-10">
          <Button
            variant="outline"
            size="icon"
            onClick={swapLanguages}
            disabled={sourceLang === "auto"}
            title="Swap languages"
          >
            <ArrowLeftRight className="h-4 w-4" />
          </Button>
        </div>

        {/* Target panel */}
        <div className="space-y-2">
          <LanguageSelect
            languages={languages}
            value={targetLang}
            onChange={setTargetLang}
          />
          {loading ? (
            <div className="min-h-[180px] rounded-md border border-input bg-secondary/30 p-3 space-y-2">
              <Skeleton className="h-4 w-3/4" />
              <Skeleton className="h-4 w-1/2" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          ) : (
            <Textarea
              value={result?.translation || ""}
              readOnly
              placeholder="Translation will appear here..."
              className="min-h-[180px] bg-secondary/30"
            />
          )}
          <div className="flex items-center justify-between">
            <AudioButton lang={targetLang} text={result?.translation || ""} />
            <Button
              variant="ghost"
              size="sm"
              onClick={handleCopy}
              disabled={!result?.translation}
              className="text-xs"
            >
              {copied ? (
                <>
                  <Check className="h-3.5 w-3.5 mr-1" />
                  Copied
                </>
              ) : (
                <>
                  <Copy className="h-3.5 w-3.5 mr-1" />
                  Copy
                </>
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Error banner */}
      {error && (
        <div className="mt-4 flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          <AlertCircle className="h-4 w-4 shrink-0" />
          {error}
        </div>
      )}

      {/* Pronunciation */}
      {result?.pronunciation && (result.pronunciation.sourcePhonetic || result.pronunciation.targetTranslit) && (
        <div className="mt-3 flex flex-wrap gap-4 text-sm text-muted-foreground">
          {result.pronunciation.sourcePhonetic && (
            <span>Source: <span className="italic">{result.pronunciation.sourcePhonetic}</span></span>
          )}
          {result.pronunciation.targetTranslit && (
            <span>Target: <span className="italic">{result.pronunciation.targetTranslit}</span></span>
          )}
        </div>
      )}

      {/* Alternatives */}
      {result?.alternatives && result.alternatives.length > 0 && (
        <div className="mt-3">
          <span className="text-xs font-medium text-muted-foreground">Alternatives: </span>
          <span className="text-sm">{result.alternatives.join(", ")}</span>
        </div>
      )}

      {/* Rich panel for definitions/synonyms/examples */}
      {result && (
        <RichPanel
          definitions={result.definitions}
          synonyms={result.synonyms}
          examples={result.examples}
        />
      )}

      {/* Provider badge */}
      {result?.provider && (
        <div className="mt-3 text-xs text-muted-foreground">
          Powered by {result.provider}
        </div>
      )}
    </div>
  )
}

// --- App ---
export default function App() {
  const { dark, toggle } = useTheme()

  return (
    <div className="min-h-screen bg-background text-foreground">
      {/* Header */}
      <header className="sticky top-0 z-40 border-b border-border bg-background/80 backdrop-blur-sm">
        <div className="mx-auto flex h-14 max-w-5xl items-center justify-between px-4">
          <div className="flex items-center gap-2">
            <Languages className="h-5 w-5 text-primary" />
            <h1 className="text-lg font-semibold">Translate</h1>
          </div>
          <Button variant="ghost" size="icon" onClick={toggle} title="Toggle theme">
            {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
          </Button>
        </div>
      </header>

      {/* Main */}
      <main className="mx-auto max-w-5xl px-4 py-6">
        <Translator />
        <HistoryPanel />
      </main>
    </div>
  )
}
