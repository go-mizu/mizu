import type {
  CalculatorResult,
  UnitConversionResult,
  CurrencyResult,
  WeatherResult,
  DefinitionResult,
  TimeResult,
} from '../types';
import type { CacheStore } from '../store/cache';

// ========== Safe math evaluator (no eval) ==========

type Token =
  | { type: 'number'; value: number }
  | { type: 'op'; value: string }
  | { type: 'lparen' }
  | { type: 'rparen' }
  | { type: 'func'; value: string }
  | { type: 'comma' };

const CONSTANTS: Record<string, number> = {
  pi: Math.PI,
  e: Math.E,
};

const FUNCTIONS: Record<string, (args: number[]) => number> = {
  sqrt: (a) => Math.sqrt(a[0]),
  sin: (a) => Math.sin(a[0]),
  cos: (a) => Math.cos(a[0]),
  tan: (a) => Math.tan(a[0]),
  log: (a) => Math.log10(a[0]),
  ln: (a) => Math.log(a[0]),
  abs: (a) => Math.abs(a[0]),
  ceil: (a) => Math.ceil(a[0]),
  floor: (a) => Math.floor(a[0]),
  round: (a) => Math.round(a[0]),
  pow: (a) => Math.pow(a[0], a[1]),
  min: (a) => Math.min(...a),
  max: (a) => Math.max(...a),
};

function tokenize(expr: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  const s = expr.replace(/\s+/g, '');

  while (i < s.length) {
    const ch = s[i];

    // Number (including decimals)
    if (/\d/.test(ch) || (ch === '.' && i + 1 < s.length && /\d/.test(s[i + 1]))) {
      let num = '';
      while (i < s.length && (/\d/.test(s[i]) || s[i] === '.')) {
        num += s[i];
        i++;
      }
      tokens.push({ type: 'number', value: parseFloat(num) });
      continue;
    }

    // Alpha (function name or constant)
    if (/[a-zA-Z_]/.test(ch)) {
      let name = '';
      while (i < s.length && /[a-zA-Z_0-9]/.test(s[i])) {
        name += s[i];
        i++;
      }
      const lower = name.toLowerCase();
      if (CONSTANTS[lower] !== undefined) {
        tokens.push({ type: 'number', value: CONSTANTS[lower] });
      } else if (FUNCTIONS[lower]) {
        tokens.push({ type: 'func', value: lower });
      } else {
        throw new Error(`Unknown identifier: ${name}`);
      }
      continue;
    }

    if (ch === '(') {
      tokens.push({ type: 'lparen' });
      i++;
      continue;
    }

    if (ch === ')') {
      tokens.push({ type: 'rparen' });
      i++;
      continue;
    }

    if (ch === ',') {
      tokens.push({ type: 'comma' });
      i++;
      continue;
    }

    if ('+-*/%^'.includes(ch)) {
      tokens.push({ type: 'op', value: ch });
      i++;
      continue;
    }

    throw new Error(`Unexpected character: ${ch}`);
  }

  return tokens;
}

// Recursive descent parser
// Grammar:
//   expr     -> term (('+' | '-') term)*
//   term     -> exponent (('*' | '/' | '%') exponent)*
//   exponent -> unary ('^' unary)*
//   unary    -> ('-' | '+') unary | primary
//   primary  -> NUMBER | '(' expr ')' | FUNC '(' args ')'
//   args     -> expr (',' expr)*

class Parser {
  private tokens: Token[];
  private pos: number;

  constructor(tokens: Token[]) {
    this.tokens = tokens;
    this.pos = 0;
  }

  parse(): number {
    const result = this.expr();
    if (this.pos < this.tokens.length) {
      throw new Error('Unexpected token at end of expression');
    }
    return result;
  }

  private peek(): Token | undefined {
    return this.tokens[this.pos];
  }

  private consume(): Token {
    const tok = this.tokens[this.pos];
    this.pos++;
    return tok;
  }

  private isOpToken(tok: Token | undefined, ops: string[]): tok is { type: 'op'; value: string } {
    return tok?.type === 'op' && 'value' in tok && ops.includes(tok.value);
  }

  private expr(): number {
    let left = this.term();
    while (this.isOpToken(this.peek(), ['+', '-'])) {
      const tok = this.consume() as { type: 'op'; value: string };
      const right = this.term();
      left = tok.value === '+' ? left + right : left - right;
    }
    return left;
  }

  private term(): number {
    let left = this.exponent();
    while (this.isOpToken(this.peek(), ['*', '/', '%'])) {
      const tok = this.consume() as { type: 'op'; value: string };
      const right = this.exponent();
      if (tok.value === '*') left = left * right;
      else if (tok.value === '/') {
        if (right === 0) throw new Error('Division by zero');
        left = left / right;
      } else left = left % right;
    }
    return left;
  }

  private exponent(): number {
    let base = this.unary();
    while (this.isOpToken(this.peek(), ['^'])) {
      this.consume();
      const exp = this.unary();
      base = Math.pow(base, exp);
    }
    return base;
  }

  private unary(): number {
    if (this.isOpToken(this.peek(), ['-'])) {
      this.consume();
      return -this.unary();
    }
    if (this.isOpToken(this.peek(), ['+'])) {
      this.consume();
      return this.unary();
    }
    return this.primary();
  }

  private primary(): number {
    const tok = this.peek();

    if (!tok) {
      throw new Error('Unexpected end of expression');
    }

    if (tok.type === 'number') {
      this.consume();
      return tok.value;
    }

    if (tok.type === 'func') {
      const funcName = tok.value;
      this.consume();
      const lparen = this.peek();
      if (lparen?.type !== 'lparen') {
        throw new Error(`Expected '(' after function ${funcName}`);
      }
      this.consume();
      const args = this.parseArgs();
      const rparen = this.peek();
      if (rparen?.type !== 'rparen') {
        throw new Error(`Expected ')' after function arguments`);
      }
      this.consume();
      const fn = FUNCTIONS[funcName];
      if (!fn) throw new Error(`Unknown function: ${funcName}`);
      return fn(args);
    }

    if (tok.type === 'lparen') {
      this.consume();
      const val = this.expr();
      const rparen = this.peek();
      if (rparen?.type !== 'rparen') {
        throw new Error(`Expected ')'`);
      }
      this.consume();
      return val;
    }

    throw new Error(`Unexpected token: ${JSON.stringify(tok)}`);
  }

  private parseArgs(): number[] {
    const args: number[] = [];
    if (this.peek()?.type === 'rparen') {
      return args;
    }
    args.push(this.expr());
    while (this.peek()?.type === 'comma') {
      this.consume();
      args.push(this.expr());
    }
    return args;
  }
}

function formatNumber(n: number): string {
  if (!isFinite(n)) return String(n);
  const abs = Math.abs(n);
  if (abs >= 1e15 || (abs < 1e-6 && abs > 0)) {
    return n.toExponential(6);
  }
  // Up to 10 decimal places, strip trailing zeros
  const fixed = n.toFixed(10).replace(/\.?0+$/, '');
  // Add commas to integer part
  const parts = fixed.split('.');
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  return parts.join('.');
}

// ========== Unit conversion ==========

interface UnitDef {
  category: string;
  toBase: (v: number) => number;
  fromBase: (v: number) => number;
}

function linearUnit(category: string, factor: number): UnitDef {
  return {
    category,
    toBase: (v) => v * factor,
    fromBase: (v) => v / factor,
  };
}

const UNITS: Record<string, UnitDef> = {
  // Length (base: meters)
  mm: linearUnit('length', 0.001),
  cm: linearUnit('length', 0.01),
  m: linearUnit('length', 1),
  km: linearUnit('length', 1000),
  in: linearUnit('length', 0.0254),
  ft: linearUnit('length', 0.3048),
  yd: linearUnit('length', 0.9144),
  mi: linearUnit('length', 1609.344),

  // Weight (base: grams)
  mg: linearUnit('weight', 0.001),
  g: linearUnit('weight', 1),
  kg: linearUnit('weight', 1000),
  lb: linearUnit('weight', 453.592),
  oz: linearUnit('weight', 28.3495),
  ton: linearUnit('weight', 907185),

  // Temperature (base: celsius)
  c: {
    category: 'temperature',
    toBase: (v) => v,
    fromBase: (v) => v,
  },
  f: {
    category: 'temperature',
    toBase: (v) => (v - 32) * (5 / 9),
    fromBase: (v) => v * (9 / 5) + 32,
  },
  k: {
    category: 'temperature',
    toBase: (v) => v - 273.15,
    fromBase: (v) => v + 273.15,
  },

  // Volume (base: milliliters)
  ml: linearUnit('volume', 1),
  l: linearUnit('volume', 1000),
  gal: linearUnit('volume', 3785.41),
  qt: linearUnit('volume', 946.353),
  pt: linearUnit('volume', 473.176),
  cup: linearUnit('volume', 236.588),
  tbsp: linearUnit('volume', 14.7868),
  tsp: linearUnit('volume', 4.92892),
  fl_oz: linearUnit('volume', 29.5735),

  // Area (base: square meters)
  mm2: linearUnit('area', 1e-6),
  cm2: linearUnit('area', 1e-4),
  m2: linearUnit('area', 1),
  km2: linearUnit('area', 1e6),
  in2: linearUnit('area', 6.4516e-4),
  ft2: linearUnit('area', 0.092903),
  acre: linearUnit('area', 4046.86),
  hectare: linearUnit('area', 10000),

  // Speed (base: m/s)
  'm/s': linearUnit('speed', 1),
  'km/h': linearUnit('speed', 1 / 3.6),
  mph: linearUnit('speed', 0.44704),
  knots: linearUnit('speed', 0.514444),

  // Data (base: bytes)
  b: linearUnit('data', 1),
  kb: linearUnit('data', 1024),
  mb: linearUnit('data', 1024 ** 2),
  gb: linearUnit('data', 1024 ** 3),
  tb: linearUnit('data', 1024 ** 4),
  pb: linearUnit('data', 1024 ** 5),

  // Time (base: seconds)
  ms: linearUnit('time', 0.001),
  s: linearUnit('time', 1),
  min: linearUnit('time', 60),
  hr: linearUnit('time', 3600),
  day: linearUnit('time', 86400),
  week: linearUnit('time', 604800),
  month: linearUnit('time', 2592000),
  year: linearUnit('time', 31536000),
};

// ========== Timezone mapping ==========

const TIMEZONE_MAP: Record<string, string> = {
  'new york': 'America/New_York',
  'los angeles': 'America/Los_Angeles',
  'chicago': 'America/Chicago',
  'denver': 'America/Denver',
  'london': 'Europe/London',
  'paris': 'Europe/Paris',
  'berlin': 'Europe/Berlin',
  'tokyo': 'Asia/Tokyo',
  'sydney': 'Australia/Sydney',
  'moscow': 'Europe/Moscow',
  'dubai': 'Asia/Dubai',
  'mumbai': 'Asia/Kolkata',
  'delhi': 'Asia/Kolkata',
  'shanghai': 'Asia/Shanghai',
  'beijing': 'Asia/Shanghai',
  'hong kong': 'Asia/Hong_Kong',
  'singapore': 'Asia/Singapore',
  'seoul': 'Asia/Seoul',
  'toronto': 'America/Toronto',
  'vancouver': 'America/Vancouver',
  'sao paulo': 'America/Sao_Paulo',
  'mexico city': 'America/Mexico_City',
  'cairo': 'Africa/Cairo',
  'johannesburg': 'Africa/Johannesburg',
  'istanbul': 'Europe/Istanbul',
  'bangkok': 'Asia/Bangkok',
  'jakarta': 'Asia/Jakarta',
  'amsterdam': 'Europe/Amsterdam',
  'rome': 'Europe/Rome',
  'madrid': 'Europe/Madrid',
  'lisbon': 'Europe/Lisbon',
  'zurich': 'Europe/Zurich',
  'vienna': 'Europe/Vienna',
  'warsaw': 'Europe/Warsaw',
  'athens': 'Europe/Athens',
  'helsinki': 'Europe/Helsinki',
  'stockholm': 'Europe/Stockholm',
  'oslo': 'Europe/Oslo',
  'copenhagen': 'Europe/Copenhagen',
  'honolulu': 'Pacific/Honolulu',
  'anchorage': 'America/Anchorage',
  'phoenix': 'America/Phoenix',
  'auckland': 'Pacific/Auckland',
  'perth': 'Australia/Perth',
  'melbourne': 'Australia/Melbourne',
  'brisbane': 'Australia/Brisbane',
  'est': 'America/New_York',
  'cst': 'America/Chicago',
  'mst': 'America/Denver',
  'pst': 'America/Los_Angeles',
  'gmt': 'Europe/London',
  'utc': 'UTC',
  'cet': 'Europe/Paris',
  'jst': 'Asia/Tokyo',
  'ist': 'Asia/Kolkata',
  'aest': 'Australia/Sydney',
};

// ========== Supported currencies ==========

const SUPPORTED_CURRENCIES = new Set([
  'usd', 'eur', 'gbp', 'jpy', 'cad', 'aud', 'chf', 'cny', 'inr',
  'krw', 'brl', 'mxn', 'sgd', 'hkd', 'nzd', 'sek', 'nok', 'dkk',
  'pln', 'zar', 'try', 'thb', 'idr', 'php', 'czk', 'ils', 'clp',
  'myr', 'twd', 'ars', 'cop', 'sar', 'aed', 'egp', 'vnd', 'bgn',
  'hrk', 'huf', 'isk', 'ron', 'rub',
]);

// Weather condition to icon mapping
const WEATHER_ICONS: Record<string, string> = {
  'sunny': 'sun',
  'clear': 'sun',
  'partly cloudy': 'cloud-sun',
  'cloudy': 'cloud',
  'overcast': 'cloud',
  'mist': 'smog',
  'fog': 'smog',
  'patchy rain possible': 'cloud-rain',
  'patchy rain nearby': 'cloud-rain',
  'light rain': 'cloud-rain',
  'moderate rain': 'cloud-showers-heavy',
  'heavy rain': 'cloud-showers-heavy',
  'light snow': 'snowflake',
  'moderate snow': 'snowflake',
  'heavy snow': 'snowflake',
  'thunderstorm': 'bolt',
  'blizzard': 'snowflake',
};

function weatherIcon(condition: string): string {
  const lower = condition.toLowerCase();
  for (const [key, icon] of Object.entries(WEATHER_ICONS)) {
    if (lower.includes(key)) return icon;
  }
  return 'cloud';
}

export class InstantService {
  private cache: CacheStore;
  private currencyRates: Map<string, { rates: Record<string, number>; fetched_at: number }>;

  constructor(cache: CacheStore) {
    this.cache = cache;
    this.currencyRates = new Map();
  }

  calculate(expr: string): CalculatorResult {
    const tokens = tokenize(expr);
    const parser = new Parser(tokens);
    const result = parser.parse();

    return {
      expression: expr,
      result,
      formatted: formatNumber(result),
    };
  }

  convert(expr: string): UnitConversionResult {
    const match = expr.match(/^([\d.]+)\s*([a-zA-Z/_2]+)\s+(?:to|in)\s+([a-zA-Z/_2]+)$/i);
    if (!match) {
      throw new Error('Invalid conversion format. Use: <number> <unit> to <unit>');
    }

    const value = parseFloat(match[1]);
    const fromUnitKey = match[2].toLowerCase();
    const toUnitKey = match[3].toLowerCase();

    const fromUnit = UNITS[fromUnitKey];
    const toUnit = UNITS[toUnitKey];

    if (!fromUnit) throw new Error(`Unknown unit: ${match[2]}`);
    if (!toUnit) throw new Error(`Unknown unit: ${match[3]}`);
    if (fromUnit.category !== toUnit.category) {
      throw new Error(`Cannot convert between ${fromUnit.category} and ${toUnit.category}`);
    }

    const baseValue = fromUnit.toBase(value);
    const result = toUnit.fromBase(baseValue);

    return {
      from_value: value,
      from_unit: fromUnitKey,
      to_value: result,
      to_unit: toUnitKey,
      category: fromUnit.category,
    };
  }

  async currency(expr: string): Promise<CurrencyResult> {
    const match = expr.match(/^([\d.]+)\s*([a-zA-Z]{3})\s+(?:to|in)\s+([a-zA-Z]{3})$/i);
    if (!match) {
      throw new Error('Invalid currency format. Use: <number> <currency> to <currency>');
    }

    const value = parseFloat(match[1]);
    const fromCurrency = match[2].toUpperCase();
    const toCurrency = match[3].toUpperCase();

    if (!SUPPORTED_CURRENCIES.has(fromCurrency.toLowerCase())) {
      throw new Error(`Unsupported currency: ${fromCurrency}`);
    }
    if (!SUPPORTED_CURRENCIES.has(toCurrency.toLowerCase())) {
      throw new Error(`Unsupported currency: ${toCurrency}`);
    }

    const rate = await this.fetchRate(fromCurrency, toCurrency);
    const result = value * rate;

    return {
      from_amount: value,
      from_currency: fromCurrency,
      to_amount: result,
      to_currency: toCurrency,
      rate,
      updated_at: new Date().toISOString(),
    };
  }

  private async fetchRate(from: string, to: string): Promise<number> {
    const cacheKey = `${from}_${to}`;

    // Check in-memory cache first
    const cached = this.currencyRates.get(cacheKey);
    const now = Date.now();
    if (cached && now - cached.fetched_at < 3600000) {
      return cached.rates[to] ?? 1;
    }

    // Check KV cache (persists across requests within TTL)
    const kvCached = await this.cache.getInstant(`currency:${cacheKey}`);
    if (kvCached && kvCached.data) {
      const rateData = kvCached.data as { rates: Record<string, number> };
      const rate = rateData.rates[to];
      if (rate !== undefined) {
        this.currencyRates.set(cacheKey, { rates: rateData.rates, fetched_at: now });
        return rate;
      }
    }

    const url = `https://api.frankfurter.app/latest?from=${from}&to=${to}`;
    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Currency API error: ${response.status}`);
    }

    const data = (await response.json()) as { rates: Record<string, number> };

    // Store in both in-memory and KV caches
    this.currencyRates.set(cacheKey, { rates: data.rates, fetched_at: now });
    await this.cache.setInstant(`currency:${cacheKey}`, {
      type: 'currency_rates',
      query: cacheKey,
      result: JSON.stringify(data.rates),
      data: { rates: data.rates },
    });

    const rate = data.rates[to];
    if (rate === undefined) {
      throw new Error(`No rate found for ${from} to ${to}`);
    }
    return rate;
  }

  async weather(location: string): Promise<WeatherResult> {
    const encoded = encodeURIComponent(location.trim());
    const url = `https://wttr.in/${encoded}?format=j1`;

    const response = await fetch(url, {
      headers: { 'User-Agent': 'mizu-search/1.0' },
    });

    if (!response.ok) {
      throw new Error(`Weather API error: ${response.status}`);
    }

    const data = (await response.json()) as {
      current_condition: Array<{
        temp_C: string;
        weatherDesc: Array<{ value: string }>;
        humidity: string;
        windspeedKmph: string;
        winddir16Point: string;
      }>;
      nearest_area: Array<{
        areaName: Array<{ value: string }>;
        country: Array<{ value: string }>;
      }>;
    };

    const current = data.current_condition[0];
    const area = data.nearest_area?.[0];
    const areaName = area?.areaName?.[0]?.value ?? location;
    const country = area?.country?.[0]?.value ?? '';
    const displayLocation = country ? `${areaName}, ${country}` : areaName;
    const condition = current.weatherDesc[0]?.value ?? 'Unknown';

    return {
      location: displayLocation,
      temperature: parseInt(current.temp_C, 10),
      unit: 'C',
      condition,
      humidity: parseInt(current.humidity, 10),
      wind_speed: parseInt(current.windspeedKmph, 10),
      wind_unit: 'km/h',
      icon: weatherIcon(condition),
    };
  }

  async define(word: string): Promise<DefinitionResult> {
    const encoded = encodeURIComponent(word.trim().toLowerCase());
    const url = `https://api.dictionaryapi.dev/api/v2/entries/en/${encoded}`;

    const response = await fetch(url);

    if (!response.ok) {
      throw new Error(`Dictionary API error: ${response.status}`);
    }

    const data = (await response.json()) as Array<{
      word: string;
      phonetic?: string;
      phonetics?: Array<{ text?: string }>;
      meanings: Array<{
        partOfSpeech: string;
        definitions: Array<{ definition: string; example?: string }>;
        synonyms: string[];
        antonyms: string[];
      }>;
    }>;

    if (!data.length) {
      throw new Error(`No definition found for: ${word}`);
    }

    const entry = data[0];
    const firstMeaning = entry.meanings[0];
    const phonetic =
      entry.phonetic ?? entry.phonetics?.find((p) => p.text)?.text ?? undefined;

    // Collect examples from definitions
    const examples: string[] = [];
    for (const def of firstMeaning?.definitions ?? []) {
      if (def.example) {
        examples.push(def.example);
      }
    }

    // Collect antonyms from all meanings
    const allAntonyms: string[] = [];
    for (const meaning of entry.meanings) {
      allAntonyms.push(...meaning.antonyms);
    }

    return {
      word: entry.word,
      phonetic,
      part_of_speech: firstMeaning?.partOfSpeech ?? 'unknown',
      definitions: firstMeaning?.definitions.map((d) => d.definition).slice(0, 5) ?? [],
      synonyms: firstMeaning?.synonyms?.slice(0, 10),
      antonyms: [...new Set(allAntonyms)].slice(0, 10),
      examples: examples.slice(0, 3),
    };
  }

  time(location: string): TimeResult {
    const normalized = location.trim().toLowerCase();
    const timezone = TIMEZONE_MAP[normalized];

    if (!timezone) {
      // Try using the location directly as a timezone identifier
      try {
        new Intl.DateTimeFormat('en-US', { timeZone: location.trim() });
        return this.formatTime(location.trim(), location.trim());
      } catch {
        throw new Error(`Unknown location or timezone: ${location}`);
      }
    }

    return this.formatTime(location.trim(), timezone);
  }

  private formatTime(location: string, timezone: string): TimeResult {
    const now = new Date();

    const timeFormatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: true,
    });

    const dateFormatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });

    const offsetFormatter = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      timeZoneName: 'longOffset',
    });

    const offsetParts = offsetFormatter.formatToParts(now);
    const offsetPart = offsetParts.find((p) => p.type === 'timeZoneName');
    const offset = offsetPart?.value ?? timezone;

    return {
      location: location.charAt(0).toUpperCase() + location.slice(1),
      time: timeFormatter.format(now),
      date: dateFormatter.format(now),
      timezone,
      offset,
    };
  }
}
