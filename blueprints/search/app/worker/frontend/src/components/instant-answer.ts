import type { InstantAnswer } from '../api';

const ICON_CALC = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="20" x="4" y="2" rx="2"/><line x1="8" x2="16" y1="6" y2="6"/><line x1="16" x2="16" y1="14" y2="18"/><path d="M16 10h.01"/><path d="M12 10h.01"/><path d="M8 10h.01"/><path d="M12 14h.01"/><path d="M8 14h.01"/><path d="M12 18h.01"/><path d="M8 18h.01"/></svg>`;
const ICON_CONVERT = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M8 3 4 7l4 4"/><path d="M4 7h16"/><path d="m16 21 4-4-4-4"/><path d="M20 17H4"/></svg>`;
const ICON_CURRENCY = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="12" x2="12" y1="2" y2="22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/></svg>`;
const ICON_WEATHER_SUN = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#FBBC05" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="4"/><path d="M12 2v2"/><path d="M12 20v2"/><path d="m4.93 4.93 1.41 1.41"/><path d="m17.66 17.66 1.41 1.41"/><path d="M2 12h2"/><path d="M20 12h2"/><path d="m6.34 17.66-1.41 1.41"/><path d="m19.07 4.93-1.41 1.41"/></svg>`;
const ICON_WEATHER_CLOUD = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#5f6368" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>`;
const ICON_WEATHER_RAIN = `<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#4285F4" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 14.899A7 7 0 1 1 15.71 8h1.79a4.5 4.5 0 0 1 2.5 8.242"/><path d="M16 14v6"/><path d="M8 14v6"/><path d="M12 16v6"/></svg>`;
const ICON_BOOK = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20"/></svg>`;
const ICON_CLOCK = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>`;

export function renderInstantAnswer(answer: InstantAnswer): string {
  switch (answer.type) {
    case 'calculator':
      return renderCalculator(answer);
    case 'unit_conversion':
      return renderConversion(answer);
    case 'currency':
      return renderCurrency(answer);
    case 'weather':
      return renderWeather(answer);
    case 'definition':
      return renderDefinition(answer);
    case 'time':
      return renderTime(answer);
    default:
      return renderGeneric(answer);
  }
}

function renderCalculator(answer: InstantAnswer): string {
  const data = answer.data || {};
  const expression = data.expression || answer.query || '';
  const result = data.formatted || data.result || answer.result || '';

  return `
    <div class="instant-card border-l-4 border-l-blue">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ICON_CALC}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${escapeHtml(expression)} = ${escapeHtml(String(result))}</div>
    </div>
  `;
}

function renderConversion(answer: InstantAnswer): string {
  const data = answer.data || {};
  const fromVal = data.from_value ?? '';
  const fromUnit = data.from_unit ?? '';
  const toVal = data.to_value ?? '';
  const toUnit = data.to_unit ?? '';
  const category = data.category ?? '';

  return `
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ICON_CONVERT}
        <span class="instant-type">Unit Conversion${category ? ` -- ${escapeHtml(category)}` : ''}</span>
      </div>
      <div class="instant-result">${escapeHtml(String(fromVal))} ${escapeHtml(fromUnit)} = ${escapeHtml(String(toVal))} ${escapeHtml(toUnit)}</div>
      ${data.formatted ? `<div class="instant-sub">${escapeHtml(data.formatted)}</div>` : ''}
    </div>
  `;
}

function renderCurrency(answer: InstantAnswer): string {
  const data = answer.data || {};
  const fromVal = data.from_value ?? '';
  const fromCur = data.from_currency ?? '';
  const toVal = data.to_value ?? '';
  const toCur = data.to_currency ?? '';
  const rate = data.rate ?? '';

  return `
    <div class="instant-card border-l-4 border-l-yellow">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ICON_CURRENCY}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${escapeHtml(String(fromVal))} ${escapeHtml(fromCur)} = ${escapeHtml(String(toVal))} ${escapeHtml(toCur)}</div>
      ${rate ? `<div class="instant-sub">1 ${escapeHtml(fromCur)} = ${escapeHtml(String(rate))} ${escapeHtml(toCur)}</div>` : ''}
    </div>
  `;
}

function renderWeather(answer: InstantAnswer): string {
  const data = answer.data || {};
  const location = data.location || '';
  const temp = data.temperature ?? '';
  const condition = (data.condition || '').toLowerCase();
  const humidity = data.humidity || '';
  const wind = data.wind || '';

  let weatherIcon = ICON_WEATHER_SUN;
  if (condition.includes('cloud') || condition.includes('overcast')) {
    weatherIcon = ICON_WEATHER_CLOUD;
  } else if (condition.includes('rain') || condition.includes('drizzle') || condition.includes('storm')) {
    weatherIcon = ICON_WEATHER_RAIN;
  }

  return `
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">Weather</div>
      <div class="flex items-center gap-4 mb-3">
        <div>${weatherIcon}</div>
        <div>
          <div class="text-2xl font-semibold text-primary">${escapeHtml(String(temp))}&deg;</div>
          <div class="text-secondary capitalize">${escapeHtml(data.condition || '')}</div>
        </div>
      </div>
      <div class="text-sm font-medium text-primary mb-2">${escapeHtml(location)}</div>
      <div class="flex gap-6 text-sm text-tertiary">
        ${humidity ? `<span>Humidity: ${escapeHtml(humidity)}</span>` : ''}
        ${wind ? `<span>Wind: ${escapeHtml(wind)}</span>` : ''}
      </div>
    </div>
  `;
}

function renderDefinition(answer: InstantAnswer): string {
  const data = answer.data || {};
  const word = data.word || answer.query || '';
  const phonetic = data.phonetic || '';
  const pos = data.part_of_speech || '';
  const definitions: string[] = data.definitions || [];
  const synonyms: string[] = data.synonyms || [];

  return `
    <div class="instant-card border-l-4 border-l-red">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ICON_BOOK}
        <span class="instant-type">Definition</span>
      </div>
      <div class="flex items-baseline gap-3 mb-1">
        <span class="text-xl font-semibold text-primary">${escapeHtml(word)}</span>
        ${phonetic ? `<span class="text-tertiary text-sm">${escapeHtml(phonetic)}</span>` : ''}
      </div>
      ${pos ? `<div class="text-sm italic text-secondary mb-2">${escapeHtml(pos)}</div>` : ''}
      ${
        definitions.length > 0
          ? `<ol class="list-decimal list-inside space-y-1 text-sm text-snippet mb-3">
              ${definitions.map((d) => `<li>${escapeHtml(d)}</li>`).join('')}
             </ol>`
          : ''
      }
      ${
        synonyms.length > 0
          ? `<div class="text-sm">
              <span class="text-tertiary">Synonyms: </span>
              <span class="text-secondary">${synonyms.map((s) => escapeHtml(s)).join(', ')}</span>
             </div>`
          : ''
      }
    </div>
  `;
}

function renderTime(answer: InstantAnswer): string {
  const data = answer.data || {};
  const location = data.location || '';
  const time = data.time || '';
  const date = data.date || '';
  const timezone = data.timezone || '';

  return `
    <div class="instant-card border-l-4 border-l-green">
      <div class="flex items-center gap-2 mb-2 text-tertiary">
        ${ICON_CLOCK}
        <span class="instant-type">Time</span>
      </div>
      <div class="text-sm font-medium text-secondary mb-1">${escapeHtml(location)}</div>
      <div class="text-4xl font-semibold text-primary mb-1">${escapeHtml(time)}</div>
      <div class="text-sm text-tertiary">${escapeHtml(date)}</div>
      ${timezone ? `<div class="text-xs text-light mt-1">${escapeHtml(timezone)}</div>` : ''}
    </div>
  `;
}

function renderGeneric(answer: InstantAnswer): string {
  return `
    <div class="instant-card border-l-4 border-l-blue">
      <div class="instant-type mb-2">${escapeHtml(answer.type)}</div>
      <div class="instant-result">${escapeHtml(answer.result)}</div>
    </div>
  `;
}

function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
