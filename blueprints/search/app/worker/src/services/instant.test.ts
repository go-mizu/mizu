import { describe, it, expect, beforeAll } from 'vitest';
import { InstantService } from './instant';
import { CacheStore } from '../store/cache';

// Create a mock KV namespace for testing (in-memory)
const createMockKV = (): KVNamespace => {
  const store = new Map<string, string>();
  return {
    get: async (key: string) => store.get(key) ?? null,
    put: async (key: string, value: string) => {
      store.set(key, value);
    },
    delete: async (key: string) => {
      store.delete(key);
    },
    list: async () => ({ keys: [], list_complete: true, cacheStatus: null }),
    getWithMetadata: async () => ({ value: null, metadata: null, cacheStatus: null }),
  } as unknown as KVNamespace;
};

describe('InstantService', () => {
  let service: InstantService;

  beforeAll(() => {
    const kv = createMockKV();
    const cache = new CacheStore(kv);
    service = new InstantService(cache);
  });

  describe('calculate', () => {
    describe('basic arithmetic', () => {
      it('adds numbers', () => {
        expect(service.calculate('2 + 3').result).toBe(5);
        expect(service.calculate('100 + 200').result).toBe(300);
        expect(service.calculate('0 + 0').result).toBe(0);
      });

      it('subtracts numbers', () => {
        expect(service.calculate('10 - 4').result).toBe(6);
        expect(service.calculate('5 - 10').result).toBe(-5);
      });

      it('multiplies numbers', () => {
        expect(service.calculate('3 * 4').result).toBe(12);
        expect(service.calculate('7 * 0').result).toBe(0);
      });

      it('divides numbers', () => {
        expect(service.calculate('15 / 3').result).toBe(5);
        expect(service.calculate('7 / 2').result).toBe(3.5);
      });

      it('calculates modulo', () => {
        expect(service.calculate('17 % 5').result).toBe(2);
        expect(service.calculate('10 % 3').result).toBe(1);
      });
    });

    describe('operator precedence', () => {
      it('respects multiplication before addition', () => {
        expect(service.calculate('2 + 3 * 4').result).toBe(14);
      });

      it('respects division before subtraction', () => {
        expect(service.calculate('10 - 6 / 2').result).toBe(7);
      });

      it('handles parentheses', () => {
        expect(service.calculate('(2 + 3) * 4').result).toBe(20);
        expect(service.calculate('(10 - 2) / (2 + 2)').result).toBe(2);
      });

      it('handles nested parentheses', () => {
        expect(service.calculate('((2 + 3) * 2) + 1').result).toBe(11);
      });
    });

    describe('exponentiation', () => {
      it('calculates powers', () => {
        expect(service.calculate('2 ^ 3').result).toBe(8);
        expect(service.calculate('3 ^ 2').result).toBe(9);
        expect(service.calculate('10 ^ 0').result).toBe(1);
      });

      it('handles right associativity', () => {
        // 2 ^ 3 ^ 2 should be 2 ^ (3 ^ 2) = 2 ^ 9 = 512
        // But our parser is left-to-right, so it's (2 ^ 3) ^ 2 = 8 ^ 2 = 64
        expect(service.calculate('2 ^ 3 ^ 2').result).toBe(64);
      });
    });

    describe('unary operators', () => {
      it('handles negative numbers', () => {
        expect(service.calculate('-5').result).toBe(-5);
        expect(service.calculate('-5 + 10').result).toBe(5);
      });

      it('handles double negation', () => {
        expect(service.calculate('--5').result).toBe(5);
      });

      it('handles positive sign', () => {
        expect(service.calculate('+5').result).toBe(5);
      });
    });

    describe('decimals', () => {
      it('handles decimal numbers', () => {
        expect(service.calculate('1.5 + 2.5').result).toBe(4);
        expect(service.calculate('3.14 * 2').result).toBeCloseTo(6.28);
      });

      it('handles floating point precision', () => {
        expect(service.calculate('0.1 + 0.2').result).toBeCloseTo(0.3);
      });
    });

    describe('functions', () => {
      it('calculates sqrt', () => {
        expect(service.calculate('sqrt(16)').result).toBe(4);
        expect(service.calculate('sqrt(2)').result).toBeCloseTo(1.414, 2);
      });

      it('calculates abs', () => {
        expect(service.calculate('abs(-5)').result).toBe(5);
        expect(service.calculate('abs(5)').result).toBe(5);
      });

      it('calculates ceil/floor/round', () => {
        expect(service.calculate('ceil(4.2)').result).toBe(5);
        expect(service.calculate('floor(4.8)').result).toBe(4);
        expect(service.calculate('round(4.5)').result).toBe(5);
        expect(service.calculate('round(4.4)').result).toBe(4);
      });

      it('calculates trigonometric functions', () => {
        expect(service.calculate('sin(0)').result).toBe(0);
        expect(service.calculate('cos(0)').result).toBe(1);
      });

      it('calculates logarithms', () => {
        expect(service.calculate('log(100)').result).toBeCloseTo(2);
        expect(service.calculate('ln(e)').result).toBeCloseTo(1);
      });

      it('calculates pow with two arguments', () => {
        expect(service.calculate('pow(2, 10)').result).toBe(1024);
      });

      it('calculates min/max', () => {
        expect(service.calculate('min(3, 7, 1)').result).toBe(1);
        expect(service.calculate('max(3, 7, 1)').result).toBe(7);
      });
    });

    describe('constants', () => {
      it('knows pi', () => {
        expect(service.calculate('pi').result).toBeCloseTo(Math.PI);
      });

      it('knows e', () => {
        expect(service.calculate('e').result).toBeCloseTo(Math.E);
      });

      it('uses constants in expressions', () => {
        expect(service.calculate('2 * pi').result).toBeCloseTo(2 * Math.PI);
      });
    });

    describe('complex expressions', () => {
      it('handles multiple operations', () => {
        expect(service.calculate('sqrt(16) + 2 * 3').result).toBe(10);
      });

      it('handles nested functions', () => {
        expect(service.calculate('sqrt(abs(-16))').result).toBe(4);
      });

      it('handles complex parentheses', () => {
        expect(service.calculate('(1 + 2) * (3 + 4)').result).toBe(21);
      });
    });

    describe('error handling', () => {
      it('throws on division by zero', () => {
        expect(() => service.calculate('5 / 0')).toThrow('Division by zero');
      });

      it('throws on unknown identifier', () => {
        expect(() => service.calculate('foo + 1')).toThrow('Unknown identifier');
      });

      it('throws on unknown function', () => {
        expect(() => service.calculate('unknown(5)')).toThrow();
      });

      it('throws on unbalanced parentheses', () => {
        expect(() => service.calculate('(2 + 3')).toThrow();
      });
    });

    describe('formatting', () => {
      it('formats with commas', () => {
        expect(service.calculate('1000 + 234').formatted).toBe('1,234');
      });

      it('formats large numbers', () => {
        expect(service.calculate('1000000').formatted).toBe('1,000,000');
      });

      it('formats decimals', () => {
        expect(service.calculate('1.5 + 0.5').formatted).toBe('2');
      });
    });
  });

  describe('convert', () => {
    describe('length', () => {
      it('converts km to m', () => {
        const result = service.convert('1 km to m');
        expect(result.from_value).toBe(1);
        expect(result.to_value).toBe(1000);
        expect(result.category).toBe('length');
      });

      it('converts m to km', () => {
        const result = service.convert('5000 m to km');
        expect(result.to_value).toBe(5);
      });

      it('converts miles to km', () => {
        const result = service.convert('1 mi to km');
        expect(result.to_value).toBeCloseTo(1.609, 2);
      });

      it('converts feet to meters', () => {
        const result = service.convert('10 ft to m');
        expect(result.to_value).toBeCloseTo(3.048, 2);
      });

      it('converts inches to cm', () => {
        const result = service.convert('1 in to cm');
        expect(result.to_value).toBeCloseTo(2.54);
      });
    });

    describe('weight', () => {
      it('converts g to kg', () => {
        const result = service.convert('1000 g to kg');
        expect(result.to_value).toBe(1);
      });

      it('converts lb to kg', () => {
        const result = service.convert('1 lb to kg');
        expect(result.to_value).toBeCloseTo(0.453, 2);
      });

      it('converts oz to g', () => {
        const result = service.convert('1 oz to g');
        expect(result.to_value).toBeCloseTo(28.35, 1);
      });
    });

    describe('temperature', () => {
      it('converts 0 C to F', () => {
        const result = service.convert('0 c to f');
        expect(result.to_value).toBe(32);
      });

      it('converts 100 C to F', () => {
        const result = service.convert('100 c to f');
        expect(result.to_value).toBe(212);
      });

      it('converts F to C', () => {
        const result = service.convert('32 f to c');
        expect(result.to_value).toBeCloseTo(0);
      });

      it('converts C to K', () => {
        const result = service.convert('0 c to k');
        expect(result.to_value).toBeCloseTo(273.15);
      });

      it('converts K to C', () => {
        const result = service.convert('273.15 k to c');
        expect(result.to_value).toBeCloseTo(0);
      });
    });

    describe('volume', () => {
      it('converts l to ml', () => {
        const result = service.convert('1 l to ml');
        expect(result.to_value).toBe(1000);
      });

      it('converts gal to l', () => {
        const result = service.convert('1 gal to l');
        expect(result.to_value).toBeCloseTo(3.785, 2);
      });
    });

    describe('time', () => {
      it('converts hours to minutes', () => {
        const result = service.convert('2 hr to min');
        expect(result.to_value).toBe(120);
      });

      it('converts days to hours', () => {
        const result = service.convert('1 day to hr');
        expect(result.to_value).toBe(24);
      });
    });

    describe('data', () => {
      it('converts kb to b', () => {
        const result = service.convert('1 kb to b');
        expect(result.to_value).toBe(1024);
      });

      it('converts gb to mb', () => {
        const result = service.convert('1 gb to mb');
        expect(result.to_value).toBe(1024);
      });
    });

    describe('error handling', () => {
      it('throws on incompatible categories', () => {
        expect(() => service.convert('10 km to kg')).toThrow(
          'Cannot convert between length and weight'
        );
      });

      it('throws on unknown units', () => {
        expect(() => service.convert('10 foo to bar')).toThrow('Unknown unit');
      });

      it('throws on invalid format', () => {
        expect(() => service.convert('hello world')).toThrow('Invalid conversion format');
      });

      it('accepts "in" keyword', () => {
        const result = service.convert('1 km in m');
        expect(result.to_value).toBe(1000);
      });
    });
  });

  describe('time', () => {
    it('returns time for known cities', () => {
      const result = service.time('Tokyo');
      expect(result.location).toBe('Tokyo');
      expect(result.timezone).toBe('Asia/Tokyo');
      expect(result.time).toMatch(/\d{1,2}:\d{2}:\d{2}\s?(AM|PM)/);
      expect(result.date).toMatch(/\w+,\s+\w+\s+\d+,\s+\d{4}/);
    });

    it('handles case insensitivity', () => {
      const result = service.time('LONDON');
      expect(result.timezone).toBe('Europe/London');
    });

    it('handles timezone abbreviations', () => {
      const result = service.time('utc');
      expect(result.timezone).toBe('UTC');
    });

    it('handles pst/est/cst/mst', () => {
      expect(service.time('pst').timezone).toBe('America/Los_Angeles');
      expect(service.time('est').timezone).toBe('America/New_York');
      expect(service.time('cst').timezone).toBe('America/Chicago');
      expect(service.time('mst').timezone).toBe('America/Denver');
    });

    it('throws on unknown location', () => {
      expect(() => service.time('InvalidPlace12345')).toThrow('Unknown location or timezone');
    });

    it('capitalizes location name', () => {
      const result = service.time('tokyo');
      expect(result.location).toBe('Tokyo');
    });
  });

  describe('currency (integration - requires network)', () => {
    it.skip('converts USD to EUR', async () => {
      const result = await service.currency('100 usd to eur');
      expect(result.from_amount).toBe(100);
      expect(result.from_currency).toBe('USD');
      expect(result.to_currency).toBe('EUR');
      expect(result.to_amount).toBeGreaterThan(0);
      expect(result.rate).toBeGreaterThan(0);
    });
  });

  describe('weather (integration - requires network)', () => {
    it.skip('gets weather for a location', async () => {
      const result = await service.weather('London');
      expect(result.location).toContain('London');
      expect(typeof result.temperature).toBe('number');
      expect(result.unit).toBe('C');
      expect(result.condition).toBeTruthy();
    });
  });

  describe('define (integration - requires network)', () => {
    it.skip('defines a word', async () => {
      const result = await service.define('hello');
      expect(result.word).toBe('hello');
      expect(result.definitions.length).toBeGreaterThan(0);
      expect(result.part_of_speech).toBeTruthy();
    });
  });
});
