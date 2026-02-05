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
    <div class="instant-card calculator">
      <div class="flex items-center gap-2 text-tertiary">
        ${ICON_CALC}
        <span class="instant-type">Calculator</span>
      </div>
      <div class="instant-result">${escapeHtml(String(result))}</div>
      <div class="instant-sub">${escapeHtml(expression)}</div>
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
    <div class="instant-card conversion">
      <div class="flex items-center gap-2 text-tertiary">
        ${ICON_CONVERT}
        <span class="instant-type">Unit Conversion${category ? ` - ${escapeHtml(category)}` : ''}</span>
      </div>
      <div class="instant-result">${escapeHtml(String(toVal))} ${escapeHtml(toUnit)}</div>
      <div class="instant-sub">${escapeHtml(String(fromVal))} ${escapeHtml(fromUnit)}</div>
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
    <div class="instant-card currency">
      <div class="flex items-center gap-2 text-tertiary">
        ${ICON_CURRENCY}
        <span class="instant-type">Currency</span>
      </div>
      <div class="instant-result">${escapeHtml(String(toVal))} ${escapeHtml(toCur)}</div>
      ${rate ? `<div class="currency-rate">1 ${escapeHtml(fromCur)} = ${escapeHtml(String(rate))} ${escapeHtml(toCur)}</div>` : ''}
      <div class="currency-updated">${escapeHtml(String(fromVal))} ${escapeHtml(fromCur)}</div>
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

  // Build meta items
  const metaItems: string[] = [];
  if (humidity) metaItems.push(`Humidity: ${escapeHtml(humidity)}`);
  if (wind) metaItems.push(`Wind: ${escapeHtml(wind)}`);

  return `
    <div class="instant-card weather">
      <div class="weather-main">
        <div class="weather-icon">${weatherIcon}</div>
        <div class="weather-temp">${escapeHtml(String(temp))}<sup>°</sup></div>
      </div>
      <div class="weather-details">
        <div class="weather-condition">${escapeHtml(data.condition || '')}</div>
        <div class="weather-location">${escapeHtml(location)}</div>
        ${metaItems.length > 0 ? `<div class="weather-meta">${metaItems.join(' · ')}</div>` : ''}
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
  const example = data.example || '';

  // Speaker icon for pronunciation button
  const ICON_SPEAKER = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><path d="M15.54 8.46a5 5 0 0 1 0 7.07"/></svg>`;

  return `
    <div class="instant-card definition">
      <div class="flex items-center gap-2 text-tertiary">
        ${ICON_BOOK}
        <span class="instant-type">Definition</span>
      </div>
      <div class="word">
        <span>${escapeHtml(word)}</span>
        <button class="pronunciation-btn" title="Listen to pronunciation" aria-label="Listen to pronunciation">
          ${ICON_SPEAKER}
        </button>
      </div>
      ${phonetic ? `<div class="phonetic">${escapeHtml(phonetic)}</div>` : ''}
      ${pos ? `<div class="part-of-speech">${escapeHtml(pos)}</div>` : ''}
      ${
        definitions.length > 0
          ? definitions.map((d, i) => `<div class="definition-text">${i + 1}. ${escapeHtml(d)}</div>`).join('')
          : ''
      }
      ${example ? `<div class="definition-example">"${escapeHtml(example)}"</div>` : ''}
      ${
        synonyms.length > 0
          ? `<div class="mt-3 text-sm">
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
    <div class="instant-card time">
      <div class="flex items-center gap-2 text-tertiary">
        ${ICON_CLOCK}
        <span class="instant-type">Time</span>
      </div>
      <div class="time-display">${escapeHtml(time)}</div>
      <div class="time-location">${escapeHtml(location)}</div>
      <div class="time-date">${escapeHtml(date)}</div>
      ${timezone ? `<div class="time-timezone">${escapeHtml(timezone)}</div>` : ''}
    </div>
  `;
}

function renderGeneric(answer: InstantAnswer): string {
  return `
    <div class="instant-card">
      <div class="instant-type">${escapeHtml(answer.type)}</div>
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
