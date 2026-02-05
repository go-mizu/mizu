const ICON_CHEVRON = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="m6 9 6 6 6-6"/></svg>`;

interface PAQuestion {
  question: string;
  answer?: string;
  source?: string;
  url?: string;
}

export function renderPeopleAlsoAsk(questions: PAQuestion[]): string {
  if (!questions || questions.length === 0) return '';

  return `
    <div class="paa-container">
      <h3 class="paa-title">People also ask</h3>
      <div class="paa-list">
        ${questions.map((q, i) => `
          <div class="paa-item" data-index="${i}">
            <button class="paa-question" aria-expanded="false">
              <span>${escapeHtml(q.question)}</span>
              <span class="paa-chevron">${ICON_CHEVRON}</span>
            </button>
            <div class="paa-answer hidden">
              ${q.answer ? `<p class="paa-answer-text">${escapeHtml(q.answer)}</p>` : '<p class="paa-loading">Loading...</p>'}
              ${q.source && q.url ? `
                <a href="${escapeAttr(q.url)}" target="_blank" class="paa-source">
                  ${escapeHtml(q.source)}
                </a>
              ` : ''}
            </div>
          </div>
        `).join('')}
      </div>
    </div>
  `;
}

export function initPeopleAlsoAsk(): void {
  const container = document.querySelector('.paa-container');
  if (!container) return;

  container.querySelectorAll('.paa-item').forEach((item) => {
    const btn = item.querySelector('.paa-question');
    const answer = item.querySelector('.paa-answer');

    btn?.addEventListener('click', () => {
      const isExpanded = btn.getAttribute('aria-expanded') === 'true';

      // Collapse all others
      container.querySelectorAll('.paa-item').forEach((other) => {
        if (other !== item) {
          other.querySelector('.paa-question')?.setAttribute('aria-expanded', 'false');
          other.querySelector('.paa-answer')?.classList.add('hidden');
          other.querySelector('.paa-chevron')?.classList.remove('rotated');
        }
      });

      // Toggle this one
      btn.setAttribute('aria-expanded', String(!isExpanded));
      answer?.classList.toggle('hidden', isExpanded);
      item.querySelector('.paa-chevron')?.classList.toggle('rotated', !isExpanded);
    });
  });
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
