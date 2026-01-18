// API Client Configuration
const API_BASE = import.meta.env.VITE_API_URL || '/api/v1'

interface RequestOptions extends RequestInit {
  params?: Record<string, string | number | boolean>
}

class ApiClient {
  private baseUrl: string

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    }

    // Get auth token and user ID from localStorage
    const token = localStorage.getItem('auth_token')
    const userId = localStorage.getItem('user_id')

    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    if (userId) {
      headers['X-User-ID'] = userId
    }

    return headers
  }

  private buildUrl(path: string, params?: Record<string, string | number | boolean>): string {
    const url = new URL(`${this.baseUrl}${path}`, window.location.origin)
    if (params) {
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          url.searchParams.append(key, String(value))
        }
      })
    }
    return url.toString()
  }

  async get<T>(path: string, options?: RequestOptions): Promise<T> {
    const url = this.buildUrl(path, options?.params)
    const response = await fetch(url, {
      method: 'GET',
      headers: this.getHeaders(),
      ...options,
    })

    if (!response.ok) {
      if (response.status === 401) {
        // Handle unauthorized - redirect to login
        window.location.href = '/login'
      }
      throw new Error(`API Error: ${response.status} ${response.statusText}`)
    }

    return response.json()
  }

  async post<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    const url = this.buildUrl(path, options?.params)
    const response = await fetch(url, {
      method: 'POST',
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
      ...options,
    })

    if (!response.ok) {
      if (response.status === 401) {
        window.location.href = '/login'
      }
      const error = await response.json().catch(() => ({ error: 'Unknown error' }))
      throw new Error(error.error || `API Error: ${response.status}`)
    }

    return response.json()
  }

  async put<T>(path: string, body?: unknown, options?: RequestOptions): Promise<T> {
    const url = this.buildUrl(path, options?.params)
    const response = await fetch(url, {
      method: 'PUT',
      headers: this.getHeaders(),
      body: body ? JSON.stringify(body) : undefined,
      ...options,
    })

    if (!response.ok) {
      if (response.status === 401) {
        window.location.href = '/login'
      }
      throw new Error(`API Error: ${response.status}`)
    }

    return response.json()
  }

  async delete<T>(path: string, options?: RequestOptions): Promise<T> {
    const url = this.buildUrl(path, options?.params)
    const response = await fetch(url, {
      method: 'DELETE',
      headers: this.getHeaders(),
      ...options,
    })

    if (!response.ok) {
      if (response.status === 401) {
        window.location.href = '/login'
      }
      throw new Error(`API Error: ${response.status}`)
    }

    return response.json()
  }
}

export const api = new ApiClient(API_BASE)

// Type definitions
export interface Language {
  id: string
  name: string
  native_name: string
  flag_emoji: string
  rtl: boolean
  enabled: boolean
}

export interface Course {
  id: string
  from_language_id: string
  learning_language_id: string
  title: string
  description: string
  total_units: number
  cefr_level: string
  enabled: boolean
}

export interface Unit {
  id: string
  course_id: string
  position: number
  title: string
  description: string
  guidebook_content?: string
  icon_url?: string
  skills: Skill[]
}

export interface Skill {
  id: string
  unit_id: string
  position: number
  name: string
  icon_name: string
  levels: number
  lexemes_count: number
  lessons?: Lesson[]
}

export interface Lesson {
  id: string
  skill_id: string
  level: number
  position: number
  exercise_count: number
  exercises?: Exercise[]
}

export interface Exercise {
  id: string
  lesson_id: string
  type: ExerciseType
  prompt: string
  correct_answer: string
  choices?: string[]
  audio_url?: string
  image_url?: string
  hints?: string[]
  difficulty: number
}

export type ExerciseType =
  | 'translation'
  | 'multiple_choice'
  | 'word_bank'
  | 'listening'
  | 'fill_blank'
  | 'match_pairs'

export interface LessonSession {
  id: string
  user_id: string
  lesson_id: string
  started_at: string
  completed_at?: string
  xp_earned: number
  mistakes_count: number
  hearts_lost: number
  is_perfect: boolean
}

export interface UserSkill {
  user_id: string
  skill_id: string
  crown_level: number
  is_legendary: boolean
  strength: number
  last_practiced_at?: string
  next_review_at?: string
}

export interface AnswerResult {
  correct: boolean
  correct_answer: string
  xp_earned?: number
}

export interface LessonResult {
  xp_earned: number
  is_perfect: boolean
  hearts_remaining: number
  achievements_unlocked?: string[]
}

// API functions
export const coursesApi = {
  listLanguages: () =>
    api.get<Language[]>('/languages'),

  listCourses: (fromLang: string) =>
    api.get<Course[]>('/courses', { params: { from: fromLang } }),

  getCourse: (id: string) =>
    api.get<Course>(`/courses/${id}`),

  getCoursePath: (courseId: string) =>
    api.get<Unit[]>(`/courses/${courseId}/path`),

  enrollCourse: (courseId: string) =>
    api.post(`/courses/${courseId}/enroll`),
}

export const lessonsApi = {
  getLesson: (id: string) =>
    api.get<Lesson>(`/lessons/${id}`),

  startLesson: (id: string) =>
    api.post<LessonSession>(`/lessons/${id}/start`),

  completeLesson: (id: string, data: { mistakes_count: number; hearts_lost: number }) =>
    api.post<LessonResult>(`/lessons/${id}/complete`, data),

  answerExercise: (id: string, answer: string) =>
    api.post<AnswerResult>(`/exercises/${id}/answer`, { answer }),
}

export const progressApi = {
  getUserSkills: (courseId: string) =>
    api.get<UserSkill[]>('/progress/skills', { params: { course_id: courseId } }),
}
