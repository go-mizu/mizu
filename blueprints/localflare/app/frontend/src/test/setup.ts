import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, beforeAll, vi } from 'vitest'

// Cleanup after each test
afterEach(() => {
  cleanup()
})

// Mock window.matchMedia for Mantine components
beforeAll(() => {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })

  // Mock ResizeObserver as a class
  class MockResizeObserver {
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
  }
  global.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver

  // Mock IntersectionObserver as a class
  class MockIntersectionObserver {
    readonly root: Element | null = null
    readonly rootMargin: string = ''
    readonly thresholds: readonly number[] = []
    observe = vi.fn()
    unobserve = vi.fn()
    disconnect = vi.fn()
    takeRecords = vi.fn(() => [])
  }
  global.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

  // Mock scrollTo
  window.scrollTo = vi.fn()

  // Mock getComputedStyle for Mantine
  const originalGetComputedStyle = window.getComputedStyle
  window.getComputedStyle = vi.fn().mockImplementation((element: Element) => {
    return {
      ...originalGetComputedStyle(element),
      getPropertyValue: vi.fn().mockReturnValue(''),
    }
  })
})

// Suppress console errors for expected test failures
const originalConsoleError = console.error
console.error = (...args: unknown[]) => {
  // Suppress React 18 hydration warnings in tests
  if (
    typeof args[0] === 'string' &&
    (args[0].includes('Warning: ReactDOM.render') ||
      args[0].includes('Warning: An update to') ||
      args[0].includes('act(...)'))
  ) {
    return
  }
  originalConsoleError.apply(console, args)
}
