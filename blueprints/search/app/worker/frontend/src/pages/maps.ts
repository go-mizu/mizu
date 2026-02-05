import { Router } from '../lib/router';
import { api } from '../api';
import type { SearchResult } from '../api';
import { addRecentSearch } from '../lib/state';
import { renderSearchBox, initSearchBox } from '../components/search-box';
import { renderTabs, initTabs } from '../components/tabs';

const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_EXTERNAL = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>`;

export function renderMapsPage(query: string): string {
  return `
    <div class="min-h-screen flex flex-col">
      <header class="search-header">
        <div class="search-header-row">
          <a href="/" data-link class="search-logo">
            <span style="color: #2563eb">M</span><span style="color: #ef4444">i</span><span style="color: #f59e0b">z</span><span style="color: #22c55e">u</span>
          </a>
          <div class="search-header-box">
            ${renderSearchBox({ size: 'sm', initialValue: query })}
          </div>
          <a href="/settings" data-link class="search-box-btn" aria-label="Settings">
            ${ICON_SETTINGS}
          </a>
        </div>
        <div class="search-tabs-row">
          ${renderTabs({ query, active: 'maps' })}
        </div>
      </header>
      <main class="flex-1 flex flex-col lg:flex-row">
        <!-- Map area -->
        <div id="map-container" class="h-[300px] lg:h-auto lg:flex-1 bg-surface relative">
          <iframe id="map-iframe" class="w-full h-full border-0" src="" title="Map"></iframe>
        </div>
        <!-- Results sidebar -->
        <div id="maps-content" class="lg:w-[400px] lg:border-l border-border overflow-y-auto">
          <div class="flex items-center justify-center py-16">
            <div class="spinner"></div>
          </div>
        </div>
      </main>
    </div>
  `;
}

export function initMapsPage(router: Router, query: string): void {
  initSearchBox((q) => router.navigate(`/maps?q=${encodeURIComponent(q)}`));
  initTabs();
  if (query) addRecentSearch(query);
  fetchAndRenderMaps(query);
}

async function fetchAndRenderMaps(query: string): Promise<void> {
  const content = document.getElementById('maps-content');
  const mapIframe = document.getElementById('map-iframe') as HTMLIFrameElement;
  if (!content || !query) return;

  // Set initial map view with search query
  if (mapIframe) {
    const osmUrl = `https://www.openstreetmap.org/export/embed.html?bbox=-180,-90,180,90&layer=mapnik&marker=0,0`;
    mapIframe.src = osmUrl;
  }

  try {
    const response = await api.searchMaps(query);
    const results = response.results;

    if (results.length === 0) {
      content.innerHTML = `<div class="p-4 text-secondary">No locations found for "${escapeHtml(query)}"</div>`;
      return;
    }

    // Update map to show first result
    const firstResult = results[0];
    const lat = firstResult.metadata?.lat || 0;
    const lon = firstResult.metadata?.lon || 0;
    if (mapIframe && lat && lon) {
      const bbox = `${lon-0.1},${lat-0.1},${lon+0.1},${lat+0.1}`;
      mapIframe.src = `https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik&marker=${lat},${lon}`;
    }

    content.innerHTML = `
      <div class="p-4">
        <div class="text-xs text-tertiary mb-4">${results.length} locations found</div>
        <div class="space-y-3">
          ${results.map((r, i) => renderLocationCard(r, i)).join('')}
        </div>
      </div>
    `;

    // Add click handlers to location cards
    content.querySelectorAll('.location-card').forEach((card) => {
      card.addEventListener('click', () => {
        const lat = (card as HTMLElement).dataset.lat;
        const lon = (card as HTMLElement).dataset.lon;
        if (lat && lon && mapIframe) {
          const bbox = `${parseFloat(lon)-0.05},${parseFloat(lat)-0.05},${parseFloat(lon)+0.05},${parseFloat(lat)+0.05}`;
          mapIframe.src = `https://www.openstreetmap.org/export/embed.html?bbox=${bbox}&layer=mapnik&marker=${lat},${lon}`;
        }
      });
    });
  } catch (err) {
    content.innerHTML = `<div class="p-4 text-red text-sm">Failed to load results. ${escapeHtml(String(err))}</div>`;
  }
}

function renderLocationCard(result: SearchResult, index: number): string {
  const lat = result.metadata?.lat || 0;
  const lon = result.metadata?.lon || 0;
  const type = result.metadata?.type || 'place';
  const address = result.content || '';

  return `
    <article class="location-card bg-white border border-border rounded-lg p-3 cursor-pointer hover:shadow-md transition-shadow"
             data-lat="${lat}" data-lon="${lon}">
      <div class="flex items-start gap-3">
        <span class="flex-shrink-0 w-8 h-8 rounded-full bg-red-500 text-white flex items-center justify-center text-sm font-medium">
          ${index + 1}
        </span>
        <div class="flex-1 min-w-0">
          <h3 class="font-medium text-primary text-sm">${escapeHtml(result.title)}</h3>
          <p class="text-xs text-tertiary mt-0.5 capitalize">${escapeHtml(type)}</p>
          ${address ? `<p class="text-xs text-secondary mt-1 line-clamp-2">${escapeHtml(address)}</p>` : ''}
          <p class="text-xs text-tertiary mt-1">${lat.toFixed(5)}, ${lon.toFixed(5)}</p>
          <div class="flex items-center gap-2 mt-2">
            <a href="${escapeAttr(result.url)}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              View on OSM ${ICON_EXTERNAL}
            </a>
            <a href="https://www.google.com/maps?q=${lat},${lon}" target="_blank" rel="noopener"
               class="text-xs text-blue hover:underline flex items-center gap-1"
               onclick="event.stopPropagation()">
              Google Maps ${ICON_EXTERNAL}
            </a>
          </div>
        </div>
      </div>
    </article>
  `;
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
