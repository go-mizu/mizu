import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from 'react'
import { AppState, AppStateStatus } from 'react-native'

interface NetworkState {
  isOnline: boolean
  checkNow: () => Promise<boolean>
}

const NetworkContext = createContext<NetworkState>({ isOnline: true, checkNow: async () => true })

const HEARTBEAT_URL = 'https://x-viewer.go-mizu.workers.dev/api/health'
const CHECK_INTERVAL = 30_000 // 30s

export function NetworkProvider({ children }: { children: React.ReactNode }) {
  const [isOnline, setIsOnline] = useState(true)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const checkNow = useCallback(async (): Promise<boolean> => {
    try {
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), 5000)
      const resp = await fetch(HEARTBEAT_URL, { signal: controller.signal })
      clearTimeout(timer)
      const online = resp.ok
      setIsOnline(online)
      return online
    } catch {
      setIsOnline(false)
      return false
    }
  }, [])

  useEffect(() => {
    checkNow()
    intervalRef.current = setInterval(checkNow, CHECK_INTERVAL)
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [checkNow])

  // Re-check when app comes to foreground
  useEffect(() => {
    const sub = AppState.addEventListener('change', (state: AppStateStatus) => {
      if (state === 'active') checkNow()
    })
    return () => sub.remove()
  }, [checkNow])

  return React.createElement(
    NetworkContext.Provider,
    { value: { isOnline, checkNow } },
    children
  )
}

export function useNetwork(): NetworkState {
  return useContext(NetworkContext)
}
