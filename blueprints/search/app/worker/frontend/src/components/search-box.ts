import { api } from '../api';
import type { Suggestion, Bang } from '../api';
import { appState } from '../lib/state';

const ICON_SEARCH = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>`;
const ICON_CLEAR = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 6 6 18"/><path d="m6 6 12 12"/></svg>`;
const ICON_MIC = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2a3 3 0 0 0-3 3v7a3 3 0 0 0 6 0V5a3 3 0 0 0-3-3Z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" x2="12" y1="19" y2="22"/></svg>`;
const ICON_CAMERA = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z"/><circle cx="12" cy="13" r="3"/></svg>`;
const ICON_HISTORY = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/><path d="M12 7v5l4 2"/></svg>`;
const ICON_BANG = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2 3 14h9l-1 8 10-12h-9l1-8z"/></svg>`;

export interface SearchBoxOptions {
  size: 'lg' | 'sm';
  initialValue?: string;
  autofocus?: boolean;
}

export function renderSearchBox(options: SearchBoxOptions): string {
  const sizeClass = options.size === 'lg' ? 'search-box-lg' : 'search-box-sm';
  const value = options.initialValue ? escapeAttr(options.initialValue) : '';
  const clearDisplay = options.initialValue ? '' : 'hidden';

  return `
    <div id="search-box-wrapper" class="relative w-full flex justify-center">
      <div id="search-box" class="search-box ${sizeClass}">
        <span class="text-light mr-3 flex-shrink-0">${ICON_SEARCH}</span>
        <input
          id="search-input"
          type="text"
          value="${value}"
          placeholder="Search the web"
          autocomplete="off"
          spellcheck="false"
          ${options.autofocus ? 'autofocus' : ''}
        />
        <button id="search-clear-btn" class="text-secondary hover:text-primary p-1 flex-shrink-0 ${clearDisplay}" type="button" aria-label="Clear">
          ${ICON_CLEAR}
        </button>
        <span class="mx-1 w-px h-5 bg-border flex-shrink-0"></span>
        <button id="voice-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Voice search">
          ${ICON_MIC}
        </button>
        <button id="camera-search-btn" class="text-light hover:text-secondary p-1 flex-shrink-0" type="button" aria-label="Image search">
          ${ICON_CAMERA}
        </button>
      </div>
      <div id="autocomplete-dropdown" class="autocomplete-dropdown hidden"></div>
    </div>
  `;
}

export function initSearchBox(onSearch: (query: string) => void): void {
  const input = document.getElementById('search-input') as HTMLInputElement | null;
  const clearBtn = document.getElementById('search-clear-btn') as HTMLElement | null;
  const dropdown = document.getElementById('autocomplete-dropdown') as HTMLElement | null;
  const wrapper = document.getElementById('search-box-wrapper') as HTMLElement | null;

  if (!input || !clearBtn || !dropdown || !wrapper) return;

  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let suggestions: SuggestionItem[] = [];
  let activeIndex = -1;
  let isOpen = false;

  interface SuggestionItem {
    text: string;
    type: 'suggestion' | 'recent' | 'bang';
    icon: string;
    prefix?: string;
  }

  function showDropdown(items: SuggestionItem[]): void {
    suggestions = items;
    activeIndex = -1;
    if (items.length === 0) {
      hideDropdown();
      return;
    }
    isOpen = true;
    dropdown!.innerHTML = items
      .map(
        (item, i) => `
        <div class="autocomplete-item ${i === activeIndex ? 'active' : ''}" data-index="${i}">
          <span class="suggestion-icon">${item.icon}</span>
          ${item.prefix ? `<span class="bang-trigger">${escapeHtml(item.prefix)}</span>` : ''}
          <span>${escapeHtml(item.text)}</span>
        </div>
      `
      )
      .join('');
    dropdown!.classList.remove('hidden');
    dropdown!.classList.add('has-items');

    dropdown!.querySelectorAll('.autocomplete-item').forEach((el) => {
      el.addEventListener('mousedown', (e) => {
        e.preventDefault();
        const idx = parseInt((el as HTMLElement).dataset.index || '0');
        selectItem(idx);
      });
      el.addEventListener('mouseenter', () => {
        const idx = parseInt((el as HTMLElement).dataset.index || '0');
        updateActive(idx);
      });
    });
  }

  function hideDropdown(): void {
    isOpen = false;
    dropdown!.classList.add('hidden');
    dropdown!.classList.remove('has-items');
    dropdown!.innerHTML = '';
    suggestions = [];
    activeIndex = -1;
  }

  function updateActive(index: number): void {
    activeIndex = index;
    dropdown!.querySelectorAll('.autocomplete-item').forEach((el, i) => {
      el.classList.toggle('active', i === index);
    });
  }

  function selectItem(index: number): void {
    const item = suggestions[index];
    if (!item) return;
    if (item.type === 'bang' && item.prefix) {
      input!.value = item.prefix + ' ';
      input!.focus();
      fetchSuggestions(item.prefix + ' ');
    } else {
      input!.value = item.text;
      hideDropdown();
      doSearch(item.text);
    }
  }

  function doSearch(query: string): void {
    const q = query.trim();
    if (!q) return;
    hideDropdown();
    onSearch(q);
  }

  async function fetchSuggestions(value: string): Promise<void> {
    const trimmed = value.trim();
    if (!trimmed) {
      showRecentSearches();
      return;
    }

    // Bang detection
    if (trimmed.startsWith('!')) {
      try {
        const bangs = await api.getBangs();
        const filtered = bangs
          .filter((b: Bang) => b.trigger.startsWith(trimmed) || b.name.toLowerCase().includes(trimmed.slice(1).toLowerCase()))
          .slice(0, 8);
        if (filtered.length > 0) {
          showDropdown(
            filtered.map((b: Bang) => ({
              text: b.name,
              type: 'bang' as const,
              icon: ICON_BANG,
              prefix: b.trigger,
            }))
          );
          return;
        }
      } catch {
        // fall through to regular suggestions
      }
    }

    try {
      const results = await api.suggest(trimmed);
      if (input!.value.trim() !== trimmed) return; // stale

      const items: SuggestionItem[] = results.map((s: Suggestion) => ({
        text: s.text,
        type: 'suggestion' as const,
        icon: ICON_SEARCH,
      }));

      if (items.length === 0) {
        showRecentSearches(trimmed);
      } else {
        showDropdown(items);
      }
    } catch {
      showRecentSearches(trimmed);
    }
  }

  function showRecentSearches(filter?: string): void {
    const state = appState.get();
    let recent = state.recentSearches;
    if (filter) {
      recent = recent.filter((s) => s.toLowerCase().includes(filter.toLowerCase()));
    }
    if (recent.length === 0) {
      hideDropdown();
      return;
    }
    showDropdown(
      recent.slice(0, 8).map((s) => ({
        text: s,
        type: 'recent' as const,
        icon: ICON_HISTORY,
      }))
    );
  }

  // Input events
  input.addEventListener('input', () => {
    const val = input!.value;
    clearBtn!.classList.toggle('hidden', val.length === 0);

    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => fetchSuggestions(val), 150);
  });

  input.addEventListener('focus', () => {
    if (input!.value.trim()) {
      fetchSuggestions(input!.value);
    } else {
      showRecentSearches();
    }
  });

  input.addEventListener('keydown', (e) => {
    if (!isOpen) {
      if (e.key === 'Enter') {
        doSearch(input!.value);
        return;
      }
      if (e.key === 'ArrowDown') {
        fetchSuggestions(input!.value);
        return;
      }
      return;
    }

    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        updateActive(Math.min(activeIndex + 1, suggestions.length - 1));
        break;
      case 'ArrowUp':
        e.preventDefault();
        updateActive(Math.max(activeIndex - 1, -1));
        break;
      case 'Enter':
        e.preventDefault();
        if (activeIndex >= 0) {
          selectItem(activeIndex);
        } else {
          doSearch(input!.value);
        }
        break;
      case 'Escape':
        hideDropdown();
        break;
      case 'Tab':
        hideDropdown();
        break;
    }
  });

  input.addEventListener('blur', () => {
    // Slight delay to allow click on dropdown items
    setTimeout(() => hideDropdown(), 200);
  });

  clearBtn.addEventListener('click', () => {
    input!.value = '';
    clearBtn!.classList.add('hidden');
    input!.focus();
    showRecentSearches();
  });

  // Get voice button and initialize voice search
  const voiceBtn = document.getElementById('voice-search-btn') as HTMLElement | null;

  if (voiceBtn) {
    initVoiceSearch(voiceBtn, input, (text) => {
      input!.value = text;
      clearBtn!.classList.remove('hidden');
      doSearch(text);
    });
  }

  // Get camera button and initialize image search
  const cameraBtn = document.getElementById('camera-search-btn') as HTMLElement | null;

  if (cameraBtn) {
    cameraBtn.addEventListener('click', () => {
      // Check if we're already on the images page (reverse-modal exists)
      const modal = document.getElementById('reverse-modal');
      if (modal) {
        // Already on images page, just open the modal
        modal.classList.remove('hidden');
      } else {
        // Navigate to images page with reverse param to auto-open modal
        window.dispatchEvent(new CustomEvent('router:navigate', {
          detail: { path: '/images?reverse=1' }
        }));
      }
    });
  }
}

function initVoiceSearch(
  button: HTMLElement,
  input: HTMLInputElement,
  onResult: (text: string) => void
): void {
  // Check for browser support
  const SpeechRecognition = (window as any).SpeechRecognition || (window as any).webkitSpeechRecognition;

  if (!SpeechRecognition) {
    button.style.display = 'none'; // Hide if not supported
    return;
  }

  let isListening = false;
  let recognition: any = null;

  button.addEventListener('click', () => {
    if (isListening) {
      stopListening();
    } else {
      startListening();
    }
  });

  function startListening(): void {
    recognition = new SpeechRecognition();
    recognition.continuous = false;
    recognition.interimResults = true;
    recognition.lang = 'en-US';

    recognition.onstart = () => {
      isListening = true;
      button.classList.add('listening');
      button.style.color = '#ea4335'; // Red color while listening
    };

    recognition.onresult = (event: any) => {
      const transcript = Array.from(event.results)
        .map((result: any) => result[0].transcript)
        .join('');

      input.value = transcript;

      // If final result, trigger search
      if (event.results[0].isFinal) {
        stopListening();
        onResult(transcript);
      }
    };

    recognition.onerror = (event: any) => {
      console.error('Speech recognition error:', event.error);
      stopListening();

      if (event.error === 'not-allowed') {
        alert('Microphone access denied. Please allow microphone access to use voice search.');
      }
    };

    recognition.onend = () => {
      stopListening();
    };

    try {
      recognition.start();
    } catch (e) {
      console.error('Failed to start speech recognition:', e);
      stopListening();
    }
  }

  function stopListening(): void {
    isListening = false;
    button.classList.remove('listening');
    button.style.color = '';

    if (recognition) {
      try {
        recognition.stop();
      } catch (e) {
        // Ignore errors when stopping
      }
      recognition = null;
    }
  }
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}
