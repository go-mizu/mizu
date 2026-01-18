import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface User {
  id: string
  email: string
  username: string
  display_name: string
  avatar_url?: string
  xp_total: number
  gems: number
  hearts: number
  streak_days: number
  is_premium: boolean
  daily_goal_minutes: number
  active_course_id?: string
  native_language_id?: string
}

interface AuthState {
  user: User | null
  token: string | null
  isAuthenticated: boolean
  login: (email: string, password: string) => Promise<void>
  signup: (email: string, username: string, password: string) => Promise<void>
  logout: () => void
  updateUser: (updates: Partial<User>) => void
  setActiveCourse: (courseId: string) => Promise<void>
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      isAuthenticated: false,

      login: async (email: string, password: string) => {
        const response = await fetch('/api/v1/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email, password }),
        })

        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.error || 'Login failed')
        }

        const data = await response.json()
        // Store in localStorage for API client
        localStorage.setItem('auth_token', data.access_token)
        localStorage.setItem('user_id', data.user.id)
        set({
          user: data.user,
          token: data.access_token,
          isAuthenticated: true,
        })
      },

      signup: async (email: string, username: string, password: string) => {
        const response = await fetch('/api/v1/auth/signup', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ email, username, password }),
        })

        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.error || 'Signup failed')
        }

        const data = await response.json()
        // Store in localStorage for API client
        localStorage.setItem('auth_token', data.access_token)
        localStorage.setItem('user_id', data.user.id)
        set({
          user: data.user,
          token: data.access_token,
          isAuthenticated: true,
        })
      },

      logout: () => {
        // Clear localStorage
        localStorage.removeItem('auth_token')
        localStorage.removeItem('user_id')
        set({
          user: null,
          token: null,
          isAuthenticated: false,
        })
      },

      updateUser: (updates: Partial<User>) => {
        const current = get().user
        if (current) {
          set({ user: { ...current, ...updates } })
        }
      },

      setActiveCourse: async (courseId: string) => {
        const token = get().token
        const userId = get().user?.id

        const response = await fetch('/api/v1/users/me/course', {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
            ...(token ? { Authorization: `Bearer ${token}` } : {}),
            ...(userId ? { 'X-User-ID': userId } : {}),
          },
          body: JSON.stringify({ course_id: courseId }),
        })

        if (!response.ok) {
          const error = await response.json()
          throw new Error(error.error || 'Failed to set active course')
        }

        const user = await response.json()
        set({ user })
      },
    }),
    {
      name: 'lingo-auth',
      partialize: (state) => ({
        user: state.user,
        token: state.token,
        isAuthenticated: state.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        // Sync localStorage keys when store is rehydrated from persistence
        // This ensures the API client has the auth headers it needs
        if (state?.token && state?.user?.id) {
          localStorage.setItem('auth_token', state.token)
          localStorage.setItem('user_id', state.user.id)
        }
      },
    }
  )
)
