import type { KnowledgePanel } from '../api';

const ICON_EXTERNAL = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"/><polyline points="15 3 21 3 21 9"/><line x1="10" x2="21" y1="14" y2="3"/></svg>`;

export function renderKnowledgePanel(panel: KnowledgePanel): string {
  const imageHtml = panel.image
    ? `<img class="kp-image" src="${escapeAttr(panel.image)}" alt="${escapeAttr(panel.title)}" loading="lazy" onerror="this.style.display='none'" />`
    : '';

  const factsHtml =
    panel.facts && panel.facts.length > 0
      ? `<table class="kp-facts">
          <tbody>
            ${panel.facts
              .map(
                (f) => `
              <tr>
                <td class="fact-label">${escapeHtml(f.label)}</td>
                <td class="fact-value">${escapeHtml(f.value)}</td>
              </tr>
            `
              )
              .join('')}
          </tbody>
        </table>`
      : '';

  const linksHtml =
    panel.links && panel.links.length > 0
      ? `<div class="kp-links">
          ${panel.links
            .map(
              (l) => `
            <a class="kp-link" href="${escapeAttr(l.url)}" target="_blank" rel="noopener">
              ${ICON_EXTERNAL}
              <span>${escapeHtml(l.title)}</span>
            </a>
          `
            )
            .join('')}
        </div>`
      : '';

  const sourceHtml = panel.source
    ? `<div class="kp-source">Source: ${escapeHtml(panel.source)}</div>`
    : '';

  return `
    <div class="knowledge-panel" id="knowledge-panel">
      ${imageHtml}
      <div class="kp-title">${escapeHtml(panel.title)}</div>
      ${panel.subtitle ? `<div class="kp-subtitle">${escapeHtml(panel.subtitle)}</div>` : ''}
      <div class="kp-description">${escapeHtml(panel.description)}</div>
      ${factsHtml}
      ${linksHtml}
      ${sourceHtml}
    </div>
  `;
}

export function initKnowledgePanel(): void {
  // Knowledge panel is mostly static content with external links
  // No special initialization needed beyond the rendered HTML
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
