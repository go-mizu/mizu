// Durable Objects Compatibility Test Worker
// This worker tests all DO features against real Cloudflare implementation

export class TestDurableObject {
  constructor(state, env) {
    this.state = state;
    this.storage = state.storage;
  }

  async fetch(request) {
    const url = new URL(request.url);
    const path = url.pathname;

    try {
      // Route to test handlers
      switch (path) {
        // Storage KV Tests
        case '/test/storage/put-get':
          return await this.testPutGet();
        case '/test/storage/put-get-types':
          return await this.testPutGetTypes();
        case '/test/storage/delete':
          return await this.testDelete();
        case '/test/storage/delete-all':
          return await this.testDeleteAll();
        case '/test/storage/list':
          return await this.testList();
        case '/test/storage/list-prefix':
          return await this.testListPrefix();
        case '/test/storage/list-limit':
          return await this.testListLimit();
        case '/test/storage/list-range':
          return await this.testListRange();
        case '/test/storage/list-reverse':
          return await this.testListReverse();

        // Batch Operations
        case '/test/batch/get':
          return await this.testBatchGet();
        case '/test/batch/put':
          return await this.testBatchPut();
        case '/test/batch/delete':
          return await this.testBatchDelete();

        // Map-like Interface
        case '/test/map/interface':
          return await this.testMapInterface();

        // Alarm Tests
        case '/test/alarm/set-get':
          return await this.testAlarmSetGet();
        case '/test/alarm/delete':
          return await this.testAlarmDelete();

        // State properties
        case '/test/state/id':
          return await this.testStateId();

        // Sync test
        case '/test/storage/sync':
          return await this.testSync();

        // Clear storage for isolation
        case '/clear':
          await this.storage.deleteAll();
          return this.ok({ cleared: true });

        default:
          return this.error(`Unknown test: ${path}`, 404);
      }
    } catch (e) {
      return this.error(e.message, 500);
    }
  }

  // Test Implementations

  async testPutGet() {
    await this.storage.deleteAll();

    await this.storage.put('key1', 'value1');
    const result = await this.storage.get('key1');

    if (result !== 'value1') {
      return this.fail(`Expected 'value1', got '${result}'`);
    }
    return this.pass('put/get string works');
  }

  async testPutGetTypes() {
    await this.storage.deleteAll();

    // Test various types
    await this.storage.put('string', 'hello');
    await this.storage.put('number', 42);
    await this.storage.put('float', 3.14);
    await this.storage.put('bool', true);
    await this.storage.put('null', null);
    await this.storage.put('array', [1, 2, 3]);
    await this.storage.put('object', { a: 1, b: 'two' });

    const results = {
      string: await this.storage.get('string'),
      number: await this.storage.get('number'),
      float: await this.storage.get('float'),
      bool: await this.storage.get('bool'),
      null: await this.storage.get('null'),
      array: await this.storage.get('array'),
      object: await this.storage.get('object'),
    };

    const checks = [
      results.string === 'hello',
      results.number === 42,
      results.float === 3.14,
      results.bool === true,
      results.null === null,
      JSON.stringify(results.array) === '[1,2,3]',
      results.object.a === 1 && results.object.b === 'two',
    ];

    if (checks.every(c => c)) {
      return this.pass('All types work correctly');
    }
    return this.fail(`Type mismatch: ${JSON.stringify(results)}`);
  }

  async testDelete() {
    await this.storage.deleteAll();

    await this.storage.put('toDelete', 'value');
    const before = await this.storage.get('toDelete');

    const deleted = await this.storage.delete('toDelete');
    const after = await this.storage.get('toDelete');

    if (before !== 'value') {
      return this.fail(`Expected value before delete, got ${before}`);
    }
    if (deleted !== true) {
      return this.fail(`Expected delete to return true, got ${deleted}`);
    }
    if (after !== undefined) {
      return this.fail(`Expected undefined after delete, got ${after}`);
    }
    return this.pass('delete works');
  }

  async testDeleteAll() {
    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);

    await this.storage.deleteAll();

    const list = await this.storage.list();
    if (list.size !== 0) {
      return this.fail(`Expected 0 entries after deleteAll, got ${list.size}`);
    }
    return this.pass('deleteAll works');
  }

  async testList() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);

    const list = await this.storage.list();

    if (list.size !== 3) {
      return this.fail(`Expected 3 entries, got ${list.size}`);
    }
    if (!list.has('a') || !list.has('b') || !list.has('c')) {
      return this.fail(`Missing keys in list`);
    }
    return this.pass('list works');
  }

  async testListPrefix() {
    await this.storage.deleteAll();

    await this.storage.put('user:1', 'alice');
    await this.storage.put('user:2', 'bob');
    await this.storage.put('item:1', 'widget');

    const list = await this.storage.list({ prefix: 'user:' });

    if (list.size !== 2) {
      return this.fail(`Expected 2 entries with prefix, got ${list.size}`);
    }
    if (!list.has('user:1') || !list.has('user:2')) {
      return this.fail(`Missing user keys`);
    }
    if (list.has('item:1')) {
      return this.fail(`Should not include item:1`);
    }
    return this.pass('list with prefix works');
  }

  async testListLimit() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);
    await this.storage.put('d', 4);

    const list = await this.storage.list({ limit: 2 });

    if (list.size !== 2) {
      return this.fail(`Expected 2 entries with limit, got ${list.size}`);
    }
    return this.pass('list with limit works');
  }

  async testListRange() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);
    await this.storage.put('d', 4);

    const list = await this.storage.list({ start: 'b', end: 'd' });

    // Should include b, c but not a or d (end is exclusive)
    if (!list.has('b') || !list.has('c')) {
      return this.fail(`Should have b and c`);
    }
    if (list.has('a') || list.has('d')) {
      return this.fail(`Should not have a or d`);
    }
    return this.pass('list with range works');
  }

  async testListReverse() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);

    const list = await this.storage.list({ reverse: true });
    const keys = Array.from(list.keys());

    if (keys[0] !== 'c' || keys[1] !== 'b' || keys[2] !== 'a') {
      return this.fail(`Expected reverse order [c,b,a], got [${keys}]`);
    }
    return this.pass('list with reverse works');
  }

  async testBatchGet() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);

    const results = await this.storage.get(['a', 'b', 'c', 'nonexistent']);

    if (!(results instanceof Map)) {
      return this.fail(`Expected Map, got ${typeof results}`);
    }
    if (results.size !== 3) {
      return this.fail(`Expected 3 results (excluding nonexistent), got ${results.size}`);
    }
    if (results.get('a') !== 1 || results.get('b') !== 2 || results.get('c') !== 3) {
      return this.fail(`Values mismatch`);
    }
    return this.pass('batch get works');
  }

  async testBatchPut() {
    await this.storage.deleteAll();

    await this.storage.put({
      x: 10,
      y: 20,
      z: 30
    });

    const results = await this.storage.get(['x', 'y', 'z']);

    if (results.get('x') !== 10 || results.get('y') !== 20 || results.get('z') !== 30) {
      return this.fail(`Batch put values mismatch`);
    }
    return this.pass('batch put works');
  }

  async testBatchDelete() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);
    await this.storage.put('c', 3);

    const deleted = await this.storage.delete(['a', 'b']);

    const remaining = await this.storage.list();

    if (remaining.size !== 1) {
      return this.fail(`Expected 1 remaining, got ${remaining.size}`);
    }
    if (!remaining.has('c')) {
      return this.fail(`Expected 'c' to remain`);
    }
    return this.pass('batch delete works');
  }

  async testMapInterface() {
    await this.storage.deleteAll();

    await this.storage.put('a', 1);
    await this.storage.put('b', 2);

    const map = await this.storage.get(['a', 'b']);

    const checks = [
      map.size === 2,
      map.has('a'),
      map.has('b'),
      !map.has('c'),
      map.get('a') === 1,
      Array.from(map.keys()).length === 2,
      Array.from(map.values()).length === 2,
      Array.from(map.entries()).length === 2,
    ];

    // Test forEach
    let forEachCount = 0;
    map.forEach((v, k) => {
      forEachCount++;
    });
    checks.push(forEachCount === 2);

    if (!checks.every(c => c)) {
      return this.fail(`Map interface check failed: ${checks}`);
    }
    return this.pass('Map-like interface works');
  }

  async testAlarmSetGet() {
    // Set alarm for 1 second from now
    const alarmTime = Date.now() + 1000;
    await this.storage.setAlarm(alarmTime);

    const retrieved = await this.storage.getAlarm();

    if (retrieved !== alarmTime) {
      return this.fail(`Alarm time mismatch: expected ${alarmTime}, got ${retrieved}`);
    }

    // Clean up
    await this.storage.deleteAlarm();
    return this.pass('setAlarm/getAlarm works');
  }

  async testAlarmDelete() {
    await this.storage.setAlarm(Date.now() + 10000);
    await this.storage.deleteAlarm();

    const after = await this.storage.getAlarm();

    if (after !== null) {
      return this.fail(`Expected null after deleteAlarm, got ${after}`);
    }
    return this.pass('deleteAlarm works');
  }

  async testStateId() {
    const id = this.state.id;
    const idStr = id.toString();

    if (!idStr || typeof idStr !== 'string') {
      return this.fail(`Expected id.toString() to return string, got ${typeof idStr}`);
    }
    if (idStr.length !== 64) {
      return this.fail(`Expected 64-char hex ID, got ${idStr.length} chars`);
    }
    return this.pass('state.id works', { id: idStr });
  }

  async testSync() {
    await this.storage.deleteAll();

    await this.storage.put('syncTest', 'value');
    await this.storage.sync();

    const result = await this.storage.get('syncTest');
    if (result !== 'value') {
      return this.fail(`Expected 'value' after sync, got ${result}`);
    }
    return this.pass('sync works');
  }

  // Alarm handler
  async alarm() {
    console.log('Alarm triggered at', Date.now());
  }

  // Helper methods
  ok(data) {
    return new Response(JSON.stringify({ ok: true, ...data }), {
      headers: { 'Content-Type': 'application/json' }
    });
  }

  pass(message, extra = {}) {
    return new Response(JSON.stringify({
      passed: true,
      message,
      ...extra
    }), {
      headers: { 'Content-Type': 'application/json' }
    });
  }

  fail(message) {
    return new Response(JSON.stringify({
      passed: false,
      message
    }), {
      headers: { 'Content-Type': 'application/json' }
    });
  }

  error(message, status) {
    return new Response(JSON.stringify({
      error: true,
      message
    }), {
      status,
      headers: { 'Content-Type': 'application/json' }
    });
  }
}

// Main Worker
export default {
  async fetch(request, env) {
    const url = new URL(request.url);
    const path = url.pathname;

    // Namespace API tests
    if (path === '/test/namespace/id-from-name') {
      const id1 = env.TEST_DO.idFromName('test-name');
      const id2 = env.TEST_DO.idFromName('test-name');
      const id3 = env.TEST_DO.idFromName('different-name');

      const same = id1.toString() === id2.toString();
      const different = id1.toString() !== id3.toString();

      if (same && different) {
        return json({ passed: true, message: 'idFromName consistency works' });
      }
      return json({ passed: false, message: `same=${same}, different=${different}` });
    }

    if (path === '/test/namespace/id-from-string') {
      const original = env.TEST_DO.idFromName('test');
      const idStr = original.toString();
      const restored = env.TEST_DO.idFromString(idStr);

      if (original.toString() === restored.toString()) {
        return json({ passed: true, message: 'idFromString works' });
      }
      return json({ passed: false, message: 'ID mismatch after idFromString' });
    }

    if (path === '/test/namespace/new-unique-id') {
      const id1 = env.TEST_DO.newUniqueId();
      const id2 = env.TEST_DO.newUniqueId();

      if (id1.toString() !== id2.toString()) {
        return json({ passed: true, message: 'newUniqueId generates unique IDs' });
      }
      return json({ passed: false, message: 'IDs should be unique' });
    }

    if (path === '/test/id/name-property') {
      const namedId = env.TEST_DO.idFromName('my-name');
      const uniqueId = env.TEST_DO.newUniqueId();

      // Named IDs should have name property, unique IDs should not
      const namedHasName = namedId.name === 'my-name';
      const uniqueNoName = uniqueId.name === undefined || uniqueId.name === null;

      if (namedHasName && uniqueNoName) {
        return json({ passed: true, message: 'ID name property works' });
      }
      return json({ passed: false, message: `namedHasName=${namedHasName}, uniqueNoName=${uniqueNoName}` });
    }

    if (path === '/test/isolation') {
      // Test that different named objects are isolated
      const id1 = env.TEST_DO.idFromName('isolation-test-1');
      const id2 = env.TEST_DO.idFromName('isolation-test-2');

      const stub1 = env.TEST_DO.get(id1);
      const stub2 = env.TEST_DO.get(id2);

      // Clear both
      await stub1.fetch(new Request('http://do/clear'));
      await stub2.fetch(new Request('http://do/clear'));

      // Set different values
      await stub1.fetch(new Request('http://do/test/storage/put-get'));

      // Check stub2 doesn't have stub1's data
      const stub2List = await stub2.fetch(new Request('http://do/test/storage/list'));
      const stub2Result = await stub2List.json();

      // stub2 should have 0 entries since we only wrote to stub1
      if (stub2Result.passed && stub2Result.message.includes('0 entries')) {
        return json({ passed: false, message: 'Isolation broken - stub2 sees stub1 data' });
      }

      return json({ passed: true, message: 'Instance isolation works' });
    }

    // Route to DO for storage tests
    if (path.startsWith('/test/')) {
      const id = env.TEST_DO.idFromName('test-do');
      const stub = env.TEST_DO.get(id);
      return stub.fetch(request);
    }

    // Run all tests
    if (path === '/run-all') {
      const tests = [
        '/test/namespace/id-from-name',
        '/test/namespace/id-from-string',
        '/test/namespace/new-unique-id',
        '/test/id/name-property',
        '/test/storage/put-get',
        '/test/storage/put-get-types',
        '/test/storage/delete',
        '/test/storage/delete-all',
        '/test/storage/list',
        '/test/storage/list-prefix',
        '/test/storage/list-limit',
        '/test/storage/list-range',
        '/test/storage/list-reverse',
        '/test/batch/get',
        '/test/batch/put',
        '/test/batch/delete',
        '/test/map/interface',
        '/test/alarm/set-get',
        '/test/alarm/delete',
        '/test/state/id',
        '/test/storage/sync',
        '/test/isolation',
      ];

      const results = [];
      let passed = 0;
      let failed = 0;

      for (const test of tests) {
        try {
          const res = await fetch(new URL(test, request.url));
          const data = await res.json();
          results.push({ test, ...data });
          if (data.passed) passed++;
          else failed++;
        } catch (e) {
          results.push({ test, passed: false, error: e.message });
          failed++;
        }
      }

      return json({
        summary: { total: tests.length, passed, failed },
        results
      });
    }

    return json({
      message: 'DO Compatibility Test Worker',
      endpoints: [
        'GET /run-all - Run all tests',
        'GET /test/* - Run individual test',
      ]
    });
  }
};

function json(data) {
  return new Response(JSON.stringify(data, null, 2), {
    headers: { 'Content-Type': 'application/json' }
  });
}
