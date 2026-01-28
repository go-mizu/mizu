import { useEffect, useRef, useCallback, useState } from 'react'

interface UseInfiniteScrollOptions {
  threshold?: number // pixels from bottom to trigger
  enabled?: boolean
}

interface UseInfiniteScrollReturn {
  containerRef: React.RefObject<HTMLDivElement | null>
  isLoading: boolean
  setIsLoading: (loading: boolean) => void
  hasMore: boolean
  setHasMore: (hasMore: boolean) => void
  reset: () => void
}

export function useInfiniteScroll(
  onLoadMore: () => Promise<void>,
  options: UseInfiniteScrollOptions = {}
): UseInfiniteScrollReturn {
  const { threshold = 200, enabled = true } = options
  const containerRef = useRef<HTMLDivElement>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [hasMore, setHasMore] = useState(true)
  const loadingRef = useRef(false)

  const handleScroll = useCallback(async () => {
    if (!enabled || loadingRef.current || !hasMore) return

    const scrollTop = window.scrollY
    const scrollHeight = document.documentElement.scrollHeight
    const clientHeight = window.innerHeight

    if (scrollHeight - scrollTop - clientHeight < threshold) {
      loadingRef.current = true
      setIsLoading(true)
      try {
        await onLoadMore()
      } finally {
        loadingRef.current = false
        setIsLoading(false)
      }
    }
  }, [enabled, hasMore, threshold, onLoadMore])

  useEffect(() => {
    if (!enabled) return

    window.addEventListener('scroll', handleScroll, { passive: true })
    return () => window.removeEventListener('scroll', handleScroll)
  }, [enabled, handleScroll])

  const reset = useCallback(() => {
    setHasMore(true)
    setIsLoading(false)
    loadingRef.current = false
  }, [])

  return {
    containerRef,
    isLoading,
    setIsLoading,
    hasMore,
    setHasMore,
    reset,
  }
}
