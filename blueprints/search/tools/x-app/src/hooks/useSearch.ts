import { useState, useCallback, useRef } from 'react'
import type { Tweet, Profile } from '../api/types'
import { fetchSearch } from '../api/client'
import { cacheGet, cacheSet } from '../cache/store'
import { SearchPeople, CACHE_SEARCH } from '../api/config'

export function useSearch(query: string, mode: string) {
  const [tweets, setTweets] = useState<Tweet[]>([])
  const [users, setUsers] = useState<Profile[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cursorRef = useRef('')
  const hasMoreRef = useRef(true)

  const fetchPage = useCallback(async (cursor: string, isRefresh = false) => {
    if (!query) return
    if (!isRefresh && !hasMoreRef.current && cursor) return
    setLoading(true)
    setError(null)

    const isPeople = mode === SearchPeople
    const cacheKey = `search:${query}:${mode}:${cursor}`

    try {
      if (!isRefresh) {
        const cached = await cacheGet<any>(cacheKey)
        if (cached) {
          if (isPeople) {
            setUsers(cursor ? prev => [...prev, ...cached.users] : cached.users)
            cursorRef.current = cached.cursor
          } else {
            setTweets(cursor ? prev => [...prev, ...cached.tweets] : cached.tweets)
            cursorRef.current = cached.cursor
          }
          hasMoreRef.current = !!cached.cursor
          setLoading(false)
          return
        }
      }

      const result = await fetchSearch(query, mode, cursor || undefined)

      if (isPeople) {
        const userList = result.users || []
        if (cursor && !isRefresh) {
          setUsers(prev => [...prev, ...userList])
        } else {
          setUsers(userList)
        }
        cursorRef.current = result.cursor
        hasMoreRef.current = !!result.cursor
        await cacheSet(cacheKey, result, CACHE_SEARCH)
      } else {
        const tweetList = result.tweets || []
        if (cursor && !isRefresh) {
          setTweets(prev => [...prev, ...tweetList])
        } else {
          setTweets(tweetList)
        }
        cursorRef.current = result.cursor
        hasMoreRef.current = !!result.cursor
        await cacheSet(cacheKey, result, CACHE_SEARCH)
      }
    } catch (e: any) {
      setError(e.message || 'Search failed')
    }
    setLoading(false)
  }, [query, mode])

  const loadMore = useCallback(() => {
    if (!loading && hasMoreRef.current && cursorRef.current) {
      fetchPage(cursorRef.current)
    }
  }, [loading, fetchPage])

  const refresh = useCallback(() => {
    cursorRef.current = ''
    hasMoreRef.current = true
    setTweets([])
    setUsers([])
    fetchPage('', true)
  }, [fetchPage])

  return { tweets, users, loading, error, loadMore, refresh, fetchInitial: () => fetchPage('') }
}
