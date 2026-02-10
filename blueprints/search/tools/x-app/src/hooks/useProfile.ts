import { useState, useEffect, useCallback } from 'react'
import type { Profile } from '../api/types'
import { fetchProfile } from '../api/client'
import { cacheGet, cacheSet, cacheGetStale } from '../cache/store'
import { CACHE_PROFILE } from '../api/config'
import { useNetwork } from './useNetwork'

export function useProfile(username: string) {
  const [profile, setProfile] = useState<Profile | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const { isOnline } = useNetwork()

  const load = useCallback(async () => {
    if (!username) return
    setLoading(true)
    setError(null)

    const cacheKey = `profile:${username.toLowerCase()}`

    // Try cache first
    const cached = await cacheGet<Profile>(cacheKey)
    if (cached) {
      setProfile(cached)
      setLoading(false)
      if (!isOnline) return // offline: stop here with cached data
    }

    // When offline, try stale cache if no fresh cache
    if (!isOnline) {
      if (!cached) {
        const stale = await cacheGetStale<Profile>(cacheKey)
        if (stale) {
          setProfile(stale)
        } else {
          setError('Offline — no cached data available')
        }
      }
      setLoading(false)
      return
    }

    // Fetch fresh data
    try {
      const p = await fetchProfile(username)
      setProfile(p)
      await cacheSet(cacheKey, p, CACHE_PROFILE)
    } catch (e: any) {
      if (!cached) {
        const stale = await cacheGetStale<Profile>(cacheKey)
        if (stale) {
          setProfile(stale)
        } else {
          setError(e.message || 'Failed to load profile')
        }
      }
    }
    setLoading(false)
  }, [username, isOnline])

  useEffect(() => { load() }, [load])

  return { profile, loading, error, refresh: load }
}
