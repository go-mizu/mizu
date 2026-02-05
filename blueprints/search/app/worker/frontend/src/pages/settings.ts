import { Router } from '../lib/router';
import { api } from '../api';
import { appState } from '../lib/state';
import type { SearchSettings } from '../lib/state';

const ICON_ARROW_LEFT = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m12 19-7-7 7-7"/><path d="M19 12H5"/></svg>`;

const REGIONS = [
  { value: 'auto', label: 'Auto-detect' },
  { value: 'us', label: 'United States' },
  { value: 'gb', label: 'United Kingdom' },
  { value: 'de', label: 'Germany' },
  { value: 'fr', label: 'France' },
  { value: 'es', label: 'Spain' },
  { value: 'it', label: 'Italy' },
  { value: 'nl', label: 'Netherlands' },
  { value: 'pl', label: 'Poland' },
  { value: 'br', label: 'Brazil' },
  { value: 'ca', label: 'Canada' },
  { value: 'au', label: 'Australia' },
  { value: 'in', label: 'India' },
  { value: 'jp', label: 'Japan' },
  { value: 'kr', label: 'South Korea' },
  { value: 'cn', label: 'China' },
  { value: 'ru', label: 'Russia' },
];

const LANGUAGES = [
  { value: 'en', label: 'English' },
  { value: 'de', label: 'German (Deutsch)' },
  { value: 'fr', label: 'French (Fran\u00e7ais)' },
  { value: 'es', label: 'Spanish (Espa\u00f1ol)' },
  { value: 'it', label: 'Italian (Italiano)' },
  { value: 'pt', label: 'Portuguese (Portugu\u00eas)' },
  { value: 'nl', label: 'Dutch (Nederlands)' },
  { value: 'pl', label: 'Polish (Polski)' },
  { value: 'ja', label: 'Japanese' },
  { value: 'ko', label: 'Korean' },
  { value: 'zh', label: 'Chinese' },
  { value: 'ru', label: 'Russian' },
  { value: 'ar', label: 'Arabic' },
  { value: 'hi', label: 'Hindi' },
];

export function renderSettingsPage(): string {
  const settings = appState.get().settings;

  return `
    <div class="min-h-screen bg-white">
      <!-- Header -->
      <header class="border-b border-border">
        <div class="max-w-[700px] mx-auto px-4 py-4 flex items-center gap-4">
          <a href="/" data-link class="text-tertiary hover:text-primary transition-colors" aria-label="Back">
            ${ICON_ARROW_LEFT}
          </a>
          <h1 class="text-xl font-semibold text-primary">Settings</h1>
        </div>
      </header>

      <!-- Form -->
      <main class="max-w-[700px] mx-auto px-4 py-8">
        <form id="settings-form">
          <!-- Safe Search -->
          <div class="settings-section">
            <h3>Safe Search</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="safe_search" value="off" ${settings.safe_search === 'off' ? 'checked' : ''} />
                <span>Off</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="moderate" ${settings.safe_search === 'moderate' ? 'checked' : ''} />
                <span>Moderate</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="safe_search" value="strict" ${settings.safe_search === 'strict' ? 'checked' : ''} />
                <span>Strict</span>
              </label>
            </div>
          </div>

          <!-- Results per page -->
          <div class="settings-section">
            <h3>Results per page</h3>
            <select name="results_per_page" class="settings-select">
              ${[10, 20, 30, 50]
                .map(
                  (n) => `<option value="${n}" ${settings.results_per_page === n ? 'selected' : ''}>${n}</option>`
                )
                .join('')}
            </select>
          </div>

          <!-- Region -->
          <div class="settings-section">
            <h3>Region</h3>
            <select name="region" class="settings-select">
              ${REGIONS.map(
                (r) => `<option value="${r.value}" ${settings.region === r.value ? 'selected' : ''}>${escapeHtml(r.label)}</option>`
              ).join('')}
            </select>
          </div>

          <!-- Language -->
          <div class="settings-section">
            <h3>Language</h3>
            <select name="language" class="settings-select">
              ${LANGUAGES.map(
                (l) => `<option value="${l.value}" ${settings.language === l.value ? 'selected' : ''}>${escapeHtml(l.label)}</option>`
              ).join('')}
            </select>
          </div>

          <!-- Theme -->
          <div class="settings-section">
            <h3>Theme</h3>
            <div class="space-y-1">
              <label class="settings-label">
                <input type="radio" name="theme" value="light" ${settings.theme === 'light' ? 'checked' : ''} />
                <span>Light</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="dark" ${settings.theme === 'dark' ? 'checked' : ''} />
                <span>Dark</span>
              </label>
              <label class="settings-label">
                <input type="radio" name="theme" value="system" ${settings.theme === 'system' ? 'checked' : ''} />
                <span>System</span>
              </label>
            </div>
          </div>

          <!-- Open in new tab -->
          <div class="settings-section">
            <h3>Behavior</h3>
            <label class="settings-label">
              <input type="checkbox" name="open_in_new_tab" ${settings.open_in_new_tab ? 'checked' : ''} />
              <span>Open results in new tab</span>
            </label>
            <label class="settings-label">
              <input type="checkbox" name="show_thumbnails" ${settings.show_thumbnails ? 'checked' : ''} />
              <span>Show thumbnails in results</span>
            </label>
          </div>

          <!-- Save Button -->
          <div class="flex items-center gap-4 pt-4">
            <button type="submit" id="settings-save-btn"
                    class="px-6 py-2.5 bg-blue text-white rounded-lg font-medium text-sm hover:bg-blue/90 transition-colors cursor-pointer">
              Save settings
            </button>
            <span id="settings-status" class="text-sm text-green hidden">Settings saved!</span>
          </div>
        </form>
      </main>
    </div>
  `;
}

export function initSettingsPage(router: Router): void {
  const form = document.getElementById('settings-form') as HTMLFormElement | null;
  const statusEl = document.getElementById('settings-status');

  if (!form) return;

  form.addEventListener('submit', async (e) => {
    e.preventDefault();

    const formData = new FormData(form);
    const newSettings: SearchSettings = {
      safe_search: formData.get('safe_search') as string || 'moderate',
      results_per_page: parseInt(formData.get('results_per_page') as string) || 10,
      region: formData.get('region') as string || 'auto',
      language: formData.get('language') as string || 'en',
      theme: formData.get('theme') as string || 'light',
      open_in_new_tab: formData.has('open_in_new_tab'),
      show_thumbnails: formData.has('show_thumbnails'),
    };

    // Save to local state
    appState.set({ settings: newSettings });

    // Try to save to server
    try {
      await api.updateSettings(newSettings);
    } catch {
      // Server save failed, but local save succeeded
    }

    // Show success message
    if (statusEl) {
      statusEl.classList.remove('hidden');
      setTimeout(() => {
        statusEl.classList.add('hidden');
      }, 2000);
    }
  });
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}
