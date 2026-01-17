import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface AppState {
  // Sidebar state
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  setSidebarCollapsed: (collapsed: boolean) => void;

  // Project info
  projectName: string;
  setProjectName: (name: string) => void;

  // API key
  serviceKey: string;
  setServiceKey: (key: string) => void;

  // Theme (always light for Supabase style)
  theme: 'light';

  // Loading states
  globalLoading: boolean;
  setGlobalLoading: (loading: boolean) => void;

  // Error state
  globalError: string | null;
  setGlobalError: (error: string | null) => void;
  clearGlobalError: () => void;

  // Storage page state
  selectedBucket: string | null;
  setSelectedBucket: (bucket: string | null) => void;
  currentPath: string;
  setCurrentPath: (path: string) => void;

  // Database page state
  selectedSchema: string;
  setSelectedSchema: (schema: string) => void;
  selectedTable: string | null;
  setSelectedTable: (table: string | null) => void;

  // SQL Editor state
  savedQueries: { id: string; name: string; query: string }[];
  addSavedQuery: (name: string, query: string) => void;
  removeSavedQuery: (id: string) => void;
}

// Default Supabase local development service key
const DEFAULT_SERVICE_KEY = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZS1kZW1vIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImV4cCI6MTk4MzgxMjk5Nn0.EGIM96RAZx35lJzdJsyH-qQwv8Hdp7fsn3W0YpN81IU';

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      // Sidebar
      sidebarCollapsed: false,
      toggleSidebar: () => set((state) => ({ sidebarCollapsed: !state.sidebarCollapsed })),
      setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),

      // Project
      projectName: 'localbase',
      setProjectName: (name) => set({ projectName: name }),

      // API Key
      serviceKey: DEFAULT_SERVICE_KEY,
      setServiceKey: (key) => set({ serviceKey: key }),

      // Theme
      theme: 'light',

      // Loading
      globalLoading: false,
      setGlobalLoading: (loading) => set({ globalLoading: loading }),

      // Error
      globalError: null,
      setGlobalError: (error) => set({ globalError: error }),
      clearGlobalError: () => set({ globalError: null }),

      // Storage
      selectedBucket: null,
      setSelectedBucket: (bucket) => set({ selectedBucket: bucket, currentPath: '' }),
      currentPath: '',
      setCurrentPath: (path) => set({ currentPath: path }),

      // Database
      selectedSchema: 'public',
      setSelectedSchema: (schema) => set({ selectedSchema: schema, selectedTable: null }),
      selectedTable: null,
      setSelectedTable: (table) => set({ selectedTable: table }),

      // SQL Editor
      savedQueries: [],
      addSavedQuery: (name, query) =>
        set((state) => ({
          savedQueries: [
            ...state.savedQueries,
            { id: crypto.randomUUID(), name, query },
          ],
        })),
      removeSavedQuery: (id) =>
        set((state) => ({
          savedQueries: state.savedQueries.filter((q) => q.id !== id),
        })),
    }),
    {
      name: 'localbase-app-storage',
      partialize: (state) => ({
        sidebarCollapsed: state.sidebarCollapsed,
        serviceKey: state.serviceKey,
        savedQueries: state.savedQueries,
      }),
    }
  )
);
