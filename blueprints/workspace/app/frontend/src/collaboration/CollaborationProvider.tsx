import { createContext, useContext, useEffect, useState, useCallback, useRef, ReactNode } from 'react'

// Collaboration user presence
export interface CollaboratorPresence {
  id: string
  name: string
  email: string
  avatarUrl?: string
  color: string
  cursor?: {
    blockId: string
    offset: number
  }
  selection?: {
    anchor: { blockId: string; offset: number }
    focus: { blockId: string; offset: number }
  }
  lastActive: Date
}

// Collaboration event types
type CollaborationEvent =
  | { type: 'user_joined'; user: CollaboratorPresence }
  | { type: 'user_left'; userId: string }
  | { type: 'cursor_move'; userId: string; cursor: CollaboratorPresence['cursor'] }
  | { type: 'selection_change'; userId: string; selection: CollaboratorPresence['selection'] }
  | { type: 'content_update'; changes: unknown[] }
  | { type: 'sync_request' }
  | { type: 'sync_response'; content: unknown }

// Collaboration context
interface CollaborationContextValue {
  isConnected: boolean
  collaborators: Map<string, CollaboratorPresence>
  currentUser: CollaboratorPresence | null
  connect: (pageId: string, user: { id: string; name: string; email: string; avatarUrl?: string }) => void
  disconnect: () => void
  updateCursor: (blockId: string, offset: number) => void
  updateSelection: (anchor: { blockId: string; offset: number }, focus: { blockId: string; offset: number }) => void
  broadcastChange: (changes: unknown[]) => void
}

const CollaborationContext = createContext<CollaborationContextValue | null>(null)

// Preset colors for collaborators
const COLLABORATOR_COLORS = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4',
  '#FFEAA7', '#DDA0DD', '#98D8C8', '#F7DC6F',
  '#BB8FCE', '#85C1E9', '#F8B500', '#00CED1',
]

// Get consistent color for user based on their ID
function getUserColor(userId: string): string {
  let hash = 0
  for (let i = 0; i < userId.length; i++) {
    const char = userId.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash
  }
  return COLLABORATOR_COLORS[Math.abs(hash) % COLLABORATOR_COLORS.length]
}

interface CollaborationProviderProps {
  children: ReactNode
  wsUrl?: string
}

export function CollaborationProvider({ children, wsUrl = '/ws/collaborate' }: CollaborationProviderProps) {
  const [isConnected, setIsConnected] = useState(false)
  const [collaborators, setCollaborators] = useState<Map<string, CollaboratorPresence>>(new Map())
  const [currentUser, setCurrentUser] = useState<CollaboratorPresence | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const heartbeatIntervalRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined)
  const pageIdRef = useRef<string>('')

  // Send message through WebSocket
  const sendMessage = useCallback((event: CollaborationEvent) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(event))
    }
  }, [])

  // Handle incoming WebSocket messages
  const handleMessage = useCallback((event: MessageEvent) => {
    try {
      const data = JSON.parse(event.data) as CollaborationEvent

      switch (data.type) {
        case 'user_joined':
          setCollaborators((prev) => {
            const updated = new Map(prev)
            updated.set(data.user.id, data.user)
            return updated
          })
          break

        case 'user_left':
          setCollaborators((prev) => {
            const updated = new Map(prev)
            updated.delete(data.userId)
            return updated
          })
          break

        case 'cursor_move':
          setCollaborators((prev) => {
            const updated = new Map(prev)
            const user = updated.get(data.userId)
            if (user) {
              updated.set(data.userId, {
                ...user,
                cursor: data.cursor,
                lastActive: new Date(),
              })
            }
            return updated
          })
          break

        case 'selection_change':
          setCollaborators((prev) => {
            const updated = new Map(prev)
            const user = updated.get(data.userId)
            if (user) {
              updated.set(data.userId, {
                ...user,
                selection: data.selection,
                lastActive: new Date(),
              })
            }
            return updated
          })
          break

        case 'content_update':
          // Emit content update event for the editor to handle
          window.dispatchEvent(new CustomEvent('collaboration:content_update', {
            detail: { changes: data.changes }
          }))
          break

        case 'sync_response':
          // Handle sync response
          window.dispatchEvent(new CustomEvent('collaboration:sync', {
            detail: { content: data.content }
          }))
          break
      }
    } catch (err) {
      console.error('Failed to parse collaboration message:', err)
    }
  }, [])

  // Connect to collaboration server
  const connect = useCallback((pageId: string, user: { id: string; name: string; email: string; avatarUrl?: string }) => {
    // Disconnect existing connection
    if (wsRef.current) {
      wsRef.current.close()
    }

    pageIdRef.current = pageId

    // Create user presence
    const presence: CollaboratorPresence = {
      ...user,
      color: getUserColor(user.id),
      lastActive: new Date(),
    }
    setCurrentUser(presence)

    // Build WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}${wsUrl}?pageId=${pageId}&userId=${user.id}`

    // Create WebSocket connection
    const ws = new WebSocket(url)
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      console.log('Collaboration connected')

      // Send join event
      sendMessage({ type: 'user_joined', user: presence })

      // Request sync
      sendMessage({ type: 'sync_request' })

      // Start heartbeat
      heartbeatIntervalRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'heartbeat' }))
        }
      }, 30000)
    }

    ws.onmessage = handleMessage

    ws.onclose = () => {
      setIsConnected(false)
      console.log('Collaboration disconnected')

      // Clear heartbeat
      if (heartbeatIntervalRef.current) {
        clearInterval(heartbeatIntervalRef.current)
      }

      // Attempt reconnection
      reconnectTimeoutRef.current = setTimeout(() => {
        if (pageIdRef.current && currentUser) {
          connect(pageIdRef.current, currentUser)
        }
      }, 3000)
    }

    ws.onerror = (err) => {
      console.error('Collaboration WebSocket error:', err)
    }
  }, [wsUrl, sendMessage, handleMessage, currentUser])

  // Disconnect from collaboration server
  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
    }
    if (heartbeatIntervalRef.current) {
      clearInterval(heartbeatIntervalRef.current)
    }
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }
    setIsConnected(false)
    setCollaborators(new Map())
    setCurrentUser(null)
    pageIdRef.current = ''
  }, [])

  // Update cursor position
  const updateCursor = useCallback((blockId: string, offset: number) => {
    if (currentUser) {
      sendMessage({
        type: 'cursor_move',
        userId: currentUser.id,
        cursor: { blockId, offset },
      })
    }
  }, [currentUser, sendMessage])

  // Update selection
  const updateSelection = useCallback((
    anchor: { blockId: string; offset: number },
    focus: { blockId: string; offset: number }
  ) => {
    if (currentUser) {
      sendMessage({
        type: 'selection_change',
        userId: currentUser.id,
        selection: { anchor, focus },
      })
    }
  }, [currentUser, sendMessage])

  // Broadcast content changes
  const broadcastChange = useCallback((changes: unknown[]) => {
    sendMessage({
      type: 'content_update',
      changes,
    })
  }, [sendMessage])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect()
    }
  }, [disconnect])

  const value: CollaborationContextValue = {
    isConnected,
    collaborators,
    currentUser,
    connect,
    disconnect,
    updateCursor,
    updateSelection,
    broadcastChange,
  }

  return (
    <CollaborationContext.Provider value={value}>
      {children}
    </CollaborationContext.Provider>
  )
}

// Hook to use collaboration context
export function useCollaboration() {
  const context = useContext(CollaborationContext)
  if (!context) {
    throw new Error('useCollaboration must be used within a CollaborationProvider')
  }
  return context
}

// Hook to get active collaborators (excluding self)
export function useActiveCollaborators() {
  const { collaborators, currentUser } = useCollaboration()

  return Array.from(collaborators.values()).filter(
    (c) => c.id !== currentUser?.id &&
           Date.now() - new Date(c.lastActive).getTime() < 60000 // Active within 1 minute
  )
}
