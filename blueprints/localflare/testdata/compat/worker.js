// Cloudflare Workers Compatibility Test Suite
// Run with: wrangler dev testdata/compat/worker.js
// These tests produce JSON output that can be compared with localflare runtime

const tests = {
  // ===========================================================================
  // Request Object Tests
  // ===========================================================================
  'REQ-001': async (request) => {
    return { url: request.url, hasUrl: !!request.url };
  },

  'REQ-002': async (request) => {
    return { method: request.method };
  },

  'REQ-003': async (request) => {
    return {
      hasHeaders: request.headers instanceof Headers,
      contentType: request.headers.get('content-type'),
    };
  },

  'REQ-CF-001': async (request) => {
    const cf = request.cf || {};
    return {
      colo: cf.colo,
      country: cf.country,
      asn: cf.asn,
      timezone: cf.timezone,
      city: cf.city,
    };
  },

  // ===========================================================================
  // Response Object Tests
  // ===========================================================================
  'RES-001': async (request) => {
    const statuses = [200, 201, 204, 301, 400, 404, 500];
    const results = {};
    for (const status of statuses) {
      const resp = new Response('', { status });
      results[`status_${status}`] = resp.status;
    }
    return results;
  },

  'RES-002': async (request) => {
    const resp = new Response('body', { status: 201, statusText: 'Created' });
    return { status: resp.status, statusText: resp.statusText };
  },

  'RES-003': async (request) => {
    const results = {};
    for (const status of [200, 201, 299, 300, 400, 500]) {
      const resp = new Response('', { status });
      results[`ok_${status}`] = resp.ok;
    }
    return results;
  },

  'RES-STATIC-001': async (request) => {
    const resp = Response.json({ message: 'Hello', count: 42 });
    return {
      status: resp.status,
      contentType: resp.headers.get('content-type'),
      body: await resp.json(),
    };
  },

  'RES-STATIC-003': async (request) => {
    const resp = Response.redirect('https://example.com/new', 302);
    return {
      status: resp.status,
      location: resp.headers.get('location'),
    };
  },

  // ===========================================================================
  // Headers Object Tests
  // ===========================================================================
  'HDR-001': async (request) => {
    const headers = new Headers();
    headers.set('Content-Type', 'application/json');
    return { value: headers.get('content-type') };
  },

  'HDR-003': async (request) => {
    const headers = new Headers();
    headers.append('Accept', 'text/html');
    headers.append('Accept', 'application/json');
    return { value: headers.get('accept') };
  },

  'HDR-010': async (request) => {
    const headers = new Headers();
    headers.set('Content-Type', 'text/plain');
    return {
      lower: headers.get('content-type'),
      upper: headers.get('CONTENT-TYPE'),
      mixed: headers.get('Content-Type'),
    };
  },

  // ===========================================================================
  // URL & URLSearchParams Tests
  // ===========================================================================
  'URL-001': async (request) => {
    const url = new URL('https://user:pass@example.com:8080/path?query=1#hash');
    return {
      href: url.href,
      protocol: url.protocol,
      host: url.host,
      hostname: url.hostname,
      port: url.port,
      pathname: url.pathname,
      search: url.search,
      hash: url.hash,
      origin: url.origin,
    };
  },

  'USP-001': async (request) => {
    const url = new URL('https://example.com?name=test&count=42');
    return {
      name: url.searchParams.get('name'),
      count: url.searchParams.get('count'),
      missing: url.searchParams.get('missing'),
    };
  },

  'USP-002': async (request) => {
    const url = new URL('https://example.com?tag=a&tag=b&tag=c');
    return { tags: url.searchParams.getAll('tag') };
  },

  // ===========================================================================
  // Web Crypto API Tests
  // ===========================================================================
  'CRYPTO-001': async (request) => {
    const buffer = new Uint8Array(16);
    crypto.getRandomValues(buffer);
    return { hasNonZero: buffer.some(v => v !== 0) };
  },

  'CRYPTO-002': async (request) => {
    const uuid = crypto.randomUUID();
    return {
      uuid,
      length: uuid.length,
      parts: uuid.split('-').length,
      version: uuid[14],
    };
  },

  'DIGEST-001': async (request) => {
    const encoder = new TextEncoder();
    const data = encoder.encode('hello');
    const hash = await crypto.subtle.digest('SHA-1', data);
    const hashArray = Array.from(new Uint8Array(hash));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    return { hash: hashHex };
  },

  'DIGEST-002': async (request) => {
    const encoder = new TextEncoder();
    const data = encoder.encode('hello');
    const hash = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hash));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    return { hash: hashHex };
  },

  'DIGEST-005': async (request) => {
    const encoder = new TextEncoder();
    const data = encoder.encode('hello');
    const hash = await crypto.subtle.digest('MD5', data);
    const hashArray = Array.from(new Uint8Array(hash));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    return { hash: hashHex };
  },

  // ===========================================================================
  // Encoding API Tests
  // ===========================================================================
  'ENC-TXT-001': async (request) => {
    const encoder = new TextEncoder();
    const bytes = encoder.encode('hello');
    return {
      length: bytes.length,
      first: bytes[0],
      last: bytes[4],
    };
  },

  'ENC-TXT-002': async (request) => {
    const encoder = new TextEncoder();
    const decoder = new TextDecoder();
    const bytes = encoder.encode('Hello, World!');
    const text = decoder.decode(bytes);
    return { text };
  },

  'ENC-TXT-004': async (request) => {
    const encoder = new TextEncoder();
    const decoder = new TextDecoder();
    const text = 'Hello 世界 🌍';
    const bytes = encoder.encode(text);
    const decoded = decoder.decode(bytes);
    return { decoded };
  },

  'B64-001': async (request) => {
    return { encoded: btoa('hello world') };
  },

  'B64-002': async (request) => {
    return { decoded: atob('aGVsbG8gd29ybGQ=') };
  },

  // ===========================================================================
  // FormData API Tests
  // ===========================================================================
  'FD-001': async (request) => {
    const formData = new FormData();
    formData.append('name', 'John');
    formData.append('age', '30');
    return {
      name: formData.get('name'),
      age: formData.get('age'),
      has: formData.has('name'),
      missing: formData.has('email'),
    };
  },

  'FD-005': async (request) => {
    const formData = new FormData();
    formData.append('tag', 'a');
    formData.append('tag', 'b');
    formData.append('tag', 'c');
    return {
      tags: formData.getAll('tag'),
      single: formData.get('tag'),
    };
  },

  // ===========================================================================
  // Blob API Tests
  // ===========================================================================
  'BLOB-001': async (request) => {
    const blob = new Blob(['Hello, ', 'World!'], { type: 'text/plain' });
    return { size: blob.size, type: blob.type };
  },

  'BLOB-007': async (request) => {
    const blob = new Blob(['Hello, World!']);
    const text = await blob.text();
    return { text };
  },

  'BLOB-006': async (request) => {
    const blob = new Blob(['Hello, World!']);
    const slice = blob.slice(0, 5);
    const text = await slice.text();
    return { text };
  },

  // ===========================================================================
  // AbortController Tests
  // ===========================================================================
  'ABORT-001': async (request) => {
    const controller = new AbortController();
    return {
      hasSignal: !!controller.signal,
      aborted: controller.signal.aborted,
    };
  },

  'ABORT-003': async (request) => {
    const controller = new AbortController();
    const before = controller.signal.aborted;
    controller.abort();
    const after = controller.signal.aborted;
    return { before, after };
  },

  // ===========================================================================
  // Performance & Timers Tests
  // ===========================================================================
  'PERF-001': async (request) => {
    const start = performance.now();
    for (let i = 0; i < 1000; i++) {}
    const end = performance.now();
    return {
      startType: typeof start,
      endType: typeof end,
      elapsed: end - start >= 0,
    };
  },

  'PERF-002': async (request) => {
    const now = Date.now();
    return {
      timestamp: now,
      valid: now > 1577836800000,
      type: typeof now,
    };
  },

  // ===========================================================================
  // JSON API Tests
  // ===========================================================================
  'JSON-001': async (request) => {
    const obj = JSON.parse('{"name":"test","count":42,"active":true}');
    return { name: obj.name, count: obj.count, active: obj.active };
  },

  'JSON-002': async (request) => {
    const obj = { name: 'test', count: 42 };
    const json = JSON.stringify(obj);
    return { json };
  },

  'JSON-006': async (request) => {
    let error = null;
    try {
      JSON.parse('invalid json');
    } catch (e) {
      error = e.name;
    }
    return { error };
  },

  // ===========================================================================
  // structuredClone Tests
  // ===========================================================================
  'CLONE-001': async (request) => {
    const obj = { a: 1, b: { c: 2 } };
    const clone = structuredClone(obj);
    clone.b.c = 3;
    return { original: obj.b.c, cloned: clone.b.c };
  },

  'CLONE-002': async (request) => {
    const arr = [1, [2, 3], { nested: true }];
    const clone = structuredClone(arr);
    clone[1][0] = 99;
    return { original: arr[1][0], cloned: clone[1][0] };
  },

  'CLONE-003': async (request) => {
    const date = new Date('2024-01-15T12:00:00Z');
    const clone = structuredClone(date);
    return {
      isDate: clone instanceof Date,
      time: clone.getTime(),
      iso: clone.toISOString(),
    };
  },

  // ===========================================================================
  // Global Objects Tests
  // ===========================================================================
  'GLB-001': async (request) => {
    return {
      hasGlobalThis: typeof globalThis === 'object',
      hasResponse: typeof globalThis.Response === 'function',
      hasRequest: typeof globalThis.Request === 'function',
      hasFetch: typeof globalThis.fetch === 'function',
    };
  },

  'GLB-002': async (request) => {
    return {
      hasSelf: typeof self === 'object',
      selfEqualsGlobal: self === globalThis,
    };
  },

  // ===========================================================================
  // Modern JavaScript Features Tests
  // ===========================================================================
  'ASYNC-AWAIT': async (request) => {
    const delayedValue = async (val) => val;
    const a = await delayedValue(1);
    const b = await delayedValue(2);
    const c = await delayedValue(3);
    return { sum: a + b + c };
  },

  'PROMISE-ALL': async (request) => {
    const promises = [Promise.resolve(1), Promise.resolve(2), Promise.resolve(3)];
    const results = await Promise.all(promises);
    return { results };
  },

  'ARRAY-METHODS': async (request) => {
    const arr = [1, 2, 3, 4, 5];
    return {
      map: arr.map(x => x * 2),
      filter: arr.filter(x => x > 2),
      reduce: arr.reduce((a, b) => a + b, 0),
      find: arr.find(x => x > 3),
      some: arr.some(x => x > 3),
      every: arr.every(x => x > 0),
      includes: arr.includes(3),
    };
  },

  'OBJECT-METHODS': async (request) => {
    const obj = { a: 1, b: 2, c: 3 };
    return {
      keys: Object.keys(obj),
      values: Object.values(obj),
      entries: Object.entries(obj),
      hasOwn: Object.hasOwn(obj, 'a'),
    };
  },

  'STRING-METHODS': async (request) => {
    const str = '  Hello, World!  ';
    return {
      trim: str.trim(),
      startsWith: str.trim().startsWith('Hello'),
      endsWith: str.trim().endsWith('!'),
      includes: str.includes('World'),
      padStart: 'abc'.padStart(6, '0'),
      replaceAll: 'aaa'.replaceAll('a', 'b'),
    };
  },

  'NULLISH-COALESCING': async (request) => {
    const a = null ?? 'default';
    const b = undefined ?? 'default';
    const c = 0 ?? 'default';
    const d = '' ?? 'default';
    return { a, b, c, d };
  },

  'OPTIONAL-CHAINING': async (request) => {
    const obj = { a: { b: { c: 42 } } };
    const nullObj = null;
    return {
      existing: obj?.a?.b?.c,
      missing: obj?.x?.y?.z,
      nullSafe: nullObj?.a?.b,
    };
  },

  'SPREAD-OPERATOR': async (request) => {
    const arr1 = [1, 2, 3];
    const arr2 = [4, 5, 6];
    const obj1 = { a: 1, b: 2 };
    const obj2 = { c: 3, d: 4 };
    return {
      arraySpread: [...arr1, ...arr2],
      objectSpread: { ...obj1, ...obj2 },
      override: { ...obj1, a: 99 },
    };
  },

  'DESTRUCTURING': async (request) => {
    const { a, b, c = 'default' } = { a: 1, b: 2 };
    const [x, y, z = 'default'] = [10, 20];
    return { a, b, c, x, y, z };
  },
};

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const testId = url.searchParams.get('test');

    // If no test specified, run all tests
    if (!testId) {
      const results = {};
      for (const [id, testFn] of Object.entries(tests)) {
        try {
          results[id] = await testFn(request);
        } catch (e) {
          results[id] = { error: e.message };
        }
      }
      return Response.json(results, {
        headers: { 'Content-Type': 'application/json' },
      });
    }

    // Run specific test
    const testFn = tests[testId];
    if (!testFn) {
      return Response.json(
        { error: `Test ${testId} not found` },
        { status: 404 }
      );
    }

    try {
      const result = await testFn(request);
      return Response.json(result);
    } catch (e) {
      return Response.json({ error: e.message }, { status: 500 });
    }
  },
};
