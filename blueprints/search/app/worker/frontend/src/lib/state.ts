export type Listener<T> = (state: T) => void;

export interface Store<T> {
  get(): T;
  set(partial: Partial<T>): void;
  subscribe(listener: Listener<T>): () => void;
}

export function createStore<T extends Record<string, any>>(initial: T): Store<T> {
  let state = { ...initial };
  const listeners = new Set<Listener<T>>();

  return {
    get(): T {
      return state;
    },

    set(partial: Partial<T>): void {
      state = { ...state, ...partial };
      listeners.forEach((fn) => fn(state));
    },

    subscribe(listener: Listener<T>): () => void {
      listeners.add(listener);
      return () => {
        listeners.delete(listener);
      };
    },
  };
}

export interface SearchSettings {
  safe_search: string;
  results_per_page: number;
  region: string;
  language: string;
  theme: string;
  open_in_new_tab: boolean;
  show_thumbnails: boolean;
}

export interface AppState {
  recentSearches: string[];
  settings: SearchSettings;
}

const STORAGE_KEY = 'mizu_search_state';

function loadState(): AppState {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      return JSON.parse(saved);
    }
  } catch {
    // ignore
  }
  return {
    recentSearches: [],
    settings: {
      safe_search: 'moderate',
      results_per_page: 10,
      region: 'auto',
      language: 'en',
      theme: 'light',
      open_in_new_tab: false,
      show_thumbnails: true,
    },
  };
}

export const appState = createStore<AppState>(loadState());

appState.subscribe((state) => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // ignore
  }
});

export function addRecentSearch(query: string): void {
  const state = appState.get();
  const searches = [query, ...state.recentSearches.filter((s) => s !== query)].slice(0, 20);
  appState.set({ recentSearches: searches });
}
