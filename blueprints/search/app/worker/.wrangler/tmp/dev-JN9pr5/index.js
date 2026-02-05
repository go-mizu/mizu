var __defProp = Object.defineProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/_internal/utils.mjs
// @__NO_SIDE_EFFECTS__
function createNotImplementedError(name) {
  return new Error(`[unenv] ${name} is not implemented yet!`);
}
__name(createNotImplementedError, "createNotImplementedError");
// @__NO_SIDE_EFFECTS__
function notImplemented(name) {
  const fn = /* @__PURE__ */ __name(() => {
    throw /* @__PURE__ */ createNotImplementedError(name);
  }, "fn");
  return Object.assign(fn, { __unenv__: true });
}
__name(notImplemented, "notImplemented");

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/perf_hooks/performance.mjs
var _timeOrigin = globalThis.performance?.timeOrigin ?? Date.now();
var _performanceNow = globalThis.performance?.now ? globalThis.performance.now.bind(globalThis.performance) : () => Date.now() - _timeOrigin;
var nodeTiming = {
  name: "node",
  entryType: "node",
  startTime: 0,
  duration: 0,
  nodeStart: 0,
  v8Start: 0,
  bootstrapComplete: 0,
  environment: 0,
  loopStart: 0,
  loopExit: 0,
  idleTime: 0,
  uvMetricsInfo: {
    loopCount: 0,
    events: 0,
    eventsWaiting: 0
  },
  detail: void 0,
  toJSON() {
    return this;
  }
};
var PerformanceEntry = class {
  static {
    __name(this, "PerformanceEntry");
  }
  __unenv__ = true;
  detail;
  entryType = "event";
  name;
  startTime;
  constructor(name, options) {
    this.name = name;
    this.startTime = options?.startTime || _performanceNow();
    this.detail = options?.detail;
  }
  get duration() {
    return _performanceNow() - this.startTime;
  }
  toJSON() {
    return {
      name: this.name,
      entryType: this.entryType,
      startTime: this.startTime,
      duration: this.duration,
      detail: this.detail
    };
  }
};
var PerformanceMark = class PerformanceMark2 extends PerformanceEntry {
  static {
    __name(this, "PerformanceMark");
  }
  entryType = "mark";
  constructor() {
    super(...arguments);
  }
  get duration() {
    return 0;
  }
};
var PerformanceMeasure = class extends PerformanceEntry {
  static {
    __name(this, "PerformanceMeasure");
  }
  entryType = "measure";
};
var PerformanceResourceTiming = class extends PerformanceEntry {
  static {
    __name(this, "PerformanceResourceTiming");
  }
  entryType = "resource";
  serverTiming = [];
  connectEnd = 0;
  connectStart = 0;
  decodedBodySize = 0;
  domainLookupEnd = 0;
  domainLookupStart = 0;
  encodedBodySize = 0;
  fetchStart = 0;
  initiatorType = "";
  name = "";
  nextHopProtocol = "";
  redirectEnd = 0;
  redirectStart = 0;
  requestStart = 0;
  responseEnd = 0;
  responseStart = 0;
  secureConnectionStart = 0;
  startTime = 0;
  transferSize = 0;
  workerStart = 0;
  responseStatus = 0;
};
var PerformanceObserverEntryList = class {
  static {
    __name(this, "PerformanceObserverEntryList");
  }
  __unenv__ = true;
  getEntries() {
    return [];
  }
  getEntriesByName(_name, _type) {
    return [];
  }
  getEntriesByType(type) {
    return [];
  }
};
var Performance = class {
  static {
    __name(this, "Performance");
  }
  __unenv__ = true;
  timeOrigin = _timeOrigin;
  eventCounts = /* @__PURE__ */ new Map();
  _entries = [];
  _resourceTimingBufferSize = 0;
  navigation = void 0;
  timing = void 0;
  timerify(_fn, _options) {
    throw createNotImplementedError("Performance.timerify");
  }
  get nodeTiming() {
    return nodeTiming;
  }
  eventLoopUtilization() {
    return {};
  }
  markResourceTiming() {
    return new PerformanceResourceTiming("");
  }
  onresourcetimingbufferfull = null;
  now() {
    if (this.timeOrigin === _timeOrigin) {
      return _performanceNow();
    }
    return Date.now() - this.timeOrigin;
  }
  clearMarks(markName) {
    this._entries = markName ? this._entries.filter((e) => e.name !== markName) : this._entries.filter((e) => e.entryType !== "mark");
  }
  clearMeasures(measureName) {
    this._entries = measureName ? this._entries.filter((e) => e.name !== measureName) : this._entries.filter((e) => e.entryType !== "measure");
  }
  clearResourceTimings() {
    this._entries = this._entries.filter((e) => e.entryType !== "resource" || e.entryType !== "navigation");
  }
  getEntries() {
    return this._entries;
  }
  getEntriesByName(name, type) {
    return this._entries.filter((e) => e.name === name && (!type || e.entryType === type));
  }
  getEntriesByType(type) {
    return this._entries.filter((e) => e.entryType === type);
  }
  mark(name, options) {
    const entry = new PerformanceMark(name, options);
    this._entries.push(entry);
    return entry;
  }
  measure(measureName, startOrMeasureOptions, endMark) {
    let start;
    let end;
    if (typeof startOrMeasureOptions === "string") {
      start = this.getEntriesByName(startOrMeasureOptions, "mark")[0]?.startTime;
      end = this.getEntriesByName(endMark, "mark")[0]?.startTime;
    } else {
      start = Number.parseFloat(startOrMeasureOptions?.start) || this.now();
      end = Number.parseFloat(startOrMeasureOptions?.end) || this.now();
    }
    const entry = new PerformanceMeasure(measureName, {
      startTime: start,
      detail: {
        start,
        end
      }
    });
    this._entries.push(entry);
    return entry;
  }
  setResourceTimingBufferSize(maxSize) {
    this._resourceTimingBufferSize = maxSize;
  }
  addEventListener(type, listener, options) {
    throw createNotImplementedError("Performance.addEventListener");
  }
  removeEventListener(type, listener, options) {
    throw createNotImplementedError("Performance.removeEventListener");
  }
  dispatchEvent(event) {
    throw createNotImplementedError("Performance.dispatchEvent");
  }
  toJSON() {
    return this;
  }
};
var PerformanceObserver = class {
  static {
    __name(this, "PerformanceObserver");
  }
  __unenv__ = true;
  static supportedEntryTypes = [];
  _callback = null;
  constructor(callback) {
    this._callback = callback;
  }
  takeRecords() {
    return [];
  }
  disconnect() {
    throw createNotImplementedError("PerformanceObserver.disconnect");
  }
  observe(options) {
    throw createNotImplementedError("PerformanceObserver.observe");
  }
  bind(fn) {
    return fn;
  }
  runInAsyncScope(fn, thisArg, ...args) {
    return fn.call(thisArg, ...args);
  }
  asyncId() {
    return 0;
  }
  triggerAsyncId() {
    return 0;
  }
  emitDestroy() {
    return this;
  }
};
var performance2 = globalThis.performance && "addEventListener" in globalThis.performance ? globalThis.performance : new Performance();

// node_modules/.pnpm/@cloudflare+unenv-preset@2.12.0_unenv@2.0.0-rc.24_workerd@1.20260131.0/node_modules/@cloudflare/unenv-preset/dist/runtime/polyfill/performance.mjs
globalThis.performance = performance2;
globalThis.Performance = Performance;
globalThis.PerformanceEntry = PerformanceEntry;
globalThis.PerformanceMark = PerformanceMark;
globalThis.PerformanceMeasure = PerformanceMeasure;
globalThis.PerformanceObserver = PerformanceObserver;
globalThis.PerformanceObserverEntryList = PerformanceObserverEntryList;
globalThis.PerformanceResourceTiming = PerformanceResourceTiming;

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/process/hrtime.mjs
var hrtime = /* @__PURE__ */ Object.assign(/* @__PURE__ */ __name(function hrtime2(startTime2) {
  const now = Date.now();
  const seconds = Math.trunc(now / 1e3);
  const nanos = now % 1e3 * 1e6;
  if (startTime2) {
    let diffSeconds = seconds - startTime2[0];
    let diffNanos = nanos - startTime2[0];
    if (diffNanos < 0) {
      diffSeconds = diffSeconds - 1;
      diffNanos = 1e9 + diffNanos;
    }
    return [diffSeconds, diffNanos];
  }
  return [seconds, nanos];
}, "hrtime"), { bigint: /* @__PURE__ */ __name(function bigint() {
  return BigInt(Date.now() * 1e6);
}, "bigint") });

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/process/process.mjs
import { EventEmitter } from "node:events";

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/tty/read-stream.mjs
var ReadStream = class {
  static {
    __name(this, "ReadStream");
  }
  fd;
  isRaw = false;
  isTTY = false;
  constructor(fd) {
    this.fd = fd;
  }
  setRawMode(mode) {
    this.isRaw = mode;
    return this;
  }
};

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/tty/write-stream.mjs
var WriteStream = class {
  static {
    __name(this, "WriteStream");
  }
  fd;
  columns = 80;
  rows = 24;
  isTTY = false;
  constructor(fd) {
    this.fd = fd;
  }
  clearLine(dir, callback) {
    callback && callback();
    return false;
  }
  clearScreenDown(callback) {
    callback && callback();
    return false;
  }
  cursorTo(x, y, callback) {
    callback && typeof callback === "function" && callback();
    return false;
  }
  moveCursor(dx, dy, callback) {
    callback && callback();
    return false;
  }
  getColorDepth(env2) {
    return 1;
  }
  hasColors(count, env2) {
    return false;
  }
  getWindowSize() {
    return [this.columns, this.rows];
  }
  write(str, encoding, cb) {
    if (str instanceof Uint8Array) {
      str = new TextDecoder().decode(str);
    }
    try {
      console.log(str);
    } catch {
    }
    cb && typeof cb === "function" && cb();
    return false;
  }
};

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/process/node-version.mjs
var NODE_VERSION = "22.14.0";

// node_modules/.pnpm/unenv@2.0.0-rc.24/node_modules/unenv/dist/runtime/node/internal/process/process.mjs
var Process = class _Process extends EventEmitter {
  static {
    __name(this, "Process");
  }
  env;
  hrtime;
  nextTick;
  constructor(impl) {
    super();
    this.env = impl.env;
    this.hrtime = impl.hrtime;
    this.nextTick = impl.nextTick;
    for (const prop of [...Object.getOwnPropertyNames(_Process.prototype), ...Object.getOwnPropertyNames(EventEmitter.prototype)]) {
      const value = this[prop];
      if (typeof value === "function") {
        this[prop] = value.bind(this);
      }
    }
  }
  // --- event emitter ---
  emitWarning(warning, type, code) {
    console.warn(`${code ? `[${code}] ` : ""}${type ? `${type}: ` : ""}${warning}`);
  }
  emit(...args) {
    return super.emit(...args);
  }
  listeners(eventName) {
    return super.listeners(eventName);
  }
  // --- stdio (lazy initializers) ---
  #stdin;
  #stdout;
  #stderr;
  get stdin() {
    return this.#stdin ??= new ReadStream(0);
  }
  get stdout() {
    return this.#stdout ??= new WriteStream(1);
  }
  get stderr() {
    return this.#stderr ??= new WriteStream(2);
  }
  // --- cwd ---
  #cwd = "/";
  chdir(cwd2) {
    this.#cwd = cwd2;
  }
  cwd() {
    return this.#cwd;
  }
  // --- dummy props and getters ---
  arch = "";
  platform = "";
  argv = [];
  argv0 = "";
  execArgv = [];
  execPath = "";
  title = "";
  pid = 200;
  ppid = 100;
  get version() {
    return `v${NODE_VERSION}`;
  }
  get versions() {
    return { node: NODE_VERSION };
  }
  get allowedNodeEnvironmentFlags() {
    return /* @__PURE__ */ new Set();
  }
  get sourceMapsEnabled() {
    return false;
  }
  get debugPort() {
    return 0;
  }
  get throwDeprecation() {
    return false;
  }
  get traceDeprecation() {
    return false;
  }
  get features() {
    return {};
  }
  get release() {
    return {};
  }
  get connected() {
    return false;
  }
  get config() {
    return {};
  }
  get moduleLoadList() {
    return [];
  }
  constrainedMemory() {
    return 0;
  }
  availableMemory() {
    return 0;
  }
  uptime() {
    return 0;
  }
  resourceUsage() {
    return {};
  }
  // --- noop methods ---
  ref() {
  }
  unref() {
  }
  // --- unimplemented methods ---
  umask() {
    throw createNotImplementedError("process.umask");
  }
  getBuiltinModule() {
    return void 0;
  }
  getActiveResourcesInfo() {
    throw createNotImplementedError("process.getActiveResourcesInfo");
  }
  exit() {
    throw createNotImplementedError("process.exit");
  }
  reallyExit() {
    throw createNotImplementedError("process.reallyExit");
  }
  kill() {
    throw createNotImplementedError("process.kill");
  }
  abort() {
    throw createNotImplementedError("process.abort");
  }
  dlopen() {
    throw createNotImplementedError("process.dlopen");
  }
  setSourceMapsEnabled() {
    throw createNotImplementedError("process.setSourceMapsEnabled");
  }
  loadEnvFile() {
    throw createNotImplementedError("process.loadEnvFile");
  }
  disconnect() {
    throw createNotImplementedError("process.disconnect");
  }
  cpuUsage() {
    throw createNotImplementedError("process.cpuUsage");
  }
  setUncaughtExceptionCaptureCallback() {
    throw createNotImplementedError("process.setUncaughtExceptionCaptureCallback");
  }
  hasUncaughtExceptionCaptureCallback() {
    throw createNotImplementedError("process.hasUncaughtExceptionCaptureCallback");
  }
  initgroups() {
    throw createNotImplementedError("process.initgroups");
  }
  openStdin() {
    throw createNotImplementedError("process.openStdin");
  }
  assert() {
    throw createNotImplementedError("process.assert");
  }
  binding() {
    throw createNotImplementedError("process.binding");
  }
  // --- attached interfaces ---
  permission = { has: /* @__PURE__ */ notImplemented("process.permission.has") };
  report = {
    directory: "",
    filename: "",
    signal: "SIGUSR2",
    compact: false,
    reportOnFatalError: false,
    reportOnSignal: false,
    reportOnUncaughtException: false,
    getReport: /* @__PURE__ */ notImplemented("process.report.getReport"),
    writeReport: /* @__PURE__ */ notImplemented("process.report.writeReport")
  };
  finalization = {
    register: /* @__PURE__ */ notImplemented("process.finalization.register"),
    unregister: /* @__PURE__ */ notImplemented("process.finalization.unregister"),
    registerBeforeExit: /* @__PURE__ */ notImplemented("process.finalization.registerBeforeExit")
  };
  memoryUsage = Object.assign(() => ({
    arrayBuffers: 0,
    rss: 0,
    external: 0,
    heapTotal: 0,
    heapUsed: 0
  }), { rss: /* @__PURE__ */ __name(() => 0, "rss") });
  // --- undefined props ---
  mainModule = void 0;
  domain = void 0;
  // optional
  send = void 0;
  exitCode = void 0;
  channel = void 0;
  getegid = void 0;
  geteuid = void 0;
  getgid = void 0;
  getgroups = void 0;
  getuid = void 0;
  setegid = void 0;
  seteuid = void 0;
  setgid = void 0;
  setgroups = void 0;
  setuid = void 0;
  // internals
  _events = void 0;
  _eventsCount = void 0;
  _exiting = void 0;
  _maxListeners = void 0;
  _debugEnd = void 0;
  _debugProcess = void 0;
  _fatalException = void 0;
  _getActiveHandles = void 0;
  _getActiveRequests = void 0;
  _kill = void 0;
  _preload_modules = void 0;
  _rawDebug = void 0;
  _startProfilerIdleNotifier = void 0;
  _stopProfilerIdleNotifier = void 0;
  _tickCallback = void 0;
  _disconnect = void 0;
  _handleQueue = void 0;
  _pendingMessage = void 0;
  _channel = void 0;
  _send = void 0;
  _linkedBinding = void 0;
};

// node_modules/.pnpm/@cloudflare+unenv-preset@2.12.0_unenv@2.0.0-rc.24_workerd@1.20260131.0/node_modules/@cloudflare/unenv-preset/dist/runtime/node/process.mjs
var globalProcess = globalThis["process"];
var getBuiltinModule = globalProcess.getBuiltinModule;
var workerdProcess = getBuiltinModule("node:process");
var isWorkerdProcessV2 = globalThis.Cloudflare.compatibilityFlags.enable_nodejs_process_v2;
var unenvProcess = new Process({
  env: globalProcess.env,
  // `hrtime` is only available from workerd process v2
  hrtime: isWorkerdProcessV2 ? workerdProcess.hrtime : hrtime,
  // `nextTick` is available from workerd process v1
  nextTick: workerdProcess.nextTick
});
var { exit, features, platform } = workerdProcess;
var {
  // Always implemented by workerd
  env,
  // Only implemented in workerd v2
  hrtime: hrtime3,
  // Always implemented by workerd
  nextTick
} = unenvProcess;
var {
  _channel,
  _disconnect,
  _events,
  _eventsCount,
  _handleQueue,
  _maxListeners,
  _pendingMessage,
  _send,
  assert,
  disconnect,
  mainModule
} = unenvProcess;
var {
  // @ts-expect-error `_debugEnd` is missing typings
  _debugEnd,
  // @ts-expect-error `_debugProcess` is missing typings
  _debugProcess,
  // @ts-expect-error `_exiting` is missing typings
  _exiting,
  // @ts-expect-error `_fatalException` is missing typings
  _fatalException,
  // @ts-expect-error `_getActiveHandles` is missing typings
  _getActiveHandles,
  // @ts-expect-error `_getActiveRequests` is missing typings
  _getActiveRequests,
  // @ts-expect-error `_kill` is missing typings
  _kill,
  // @ts-expect-error `_linkedBinding` is missing typings
  _linkedBinding,
  // @ts-expect-error `_preload_modules` is missing typings
  _preload_modules,
  // @ts-expect-error `_rawDebug` is missing typings
  _rawDebug,
  // @ts-expect-error `_startProfilerIdleNotifier` is missing typings
  _startProfilerIdleNotifier,
  // @ts-expect-error `_stopProfilerIdleNotifier` is missing typings
  _stopProfilerIdleNotifier,
  // @ts-expect-error `_tickCallback` is missing typings
  _tickCallback,
  abort,
  addListener,
  allowedNodeEnvironmentFlags,
  arch,
  argv,
  argv0,
  availableMemory,
  // @ts-expect-error `binding` is missing typings
  binding,
  channel,
  chdir,
  config,
  connected,
  constrainedMemory,
  cpuUsage,
  cwd,
  debugPort,
  dlopen,
  // @ts-expect-error `domain` is missing typings
  domain,
  emit,
  emitWarning,
  eventNames,
  execArgv,
  execPath,
  exitCode,
  finalization,
  getActiveResourcesInfo,
  getegid,
  geteuid,
  getgid,
  getgroups,
  getMaxListeners,
  getuid,
  hasUncaughtExceptionCaptureCallback,
  // @ts-expect-error `initgroups` is missing typings
  initgroups,
  kill,
  listenerCount,
  listeners,
  loadEnvFile,
  memoryUsage,
  // @ts-expect-error `moduleLoadList` is missing typings
  moduleLoadList,
  off,
  on,
  once,
  // @ts-expect-error `openStdin` is missing typings
  openStdin,
  permission,
  pid,
  ppid,
  prependListener,
  prependOnceListener,
  rawListeners,
  // @ts-expect-error `reallyExit` is missing typings
  reallyExit,
  ref,
  release,
  removeAllListeners,
  removeListener,
  report,
  resourceUsage,
  send,
  setegid,
  seteuid,
  setgid,
  setgroups,
  setMaxListeners,
  setSourceMapsEnabled,
  setuid,
  setUncaughtExceptionCaptureCallback,
  sourceMapsEnabled,
  stderr,
  stdin,
  stdout,
  throwDeprecation,
  title,
  traceDeprecation,
  umask,
  unref,
  uptime,
  version,
  versions
} = isWorkerdProcessV2 ? workerdProcess : unenvProcess;
var _process = {
  abort,
  addListener,
  allowedNodeEnvironmentFlags,
  hasUncaughtExceptionCaptureCallback,
  setUncaughtExceptionCaptureCallback,
  loadEnvFile,
  sourceMapsEnabled,
  arch,
  argv,
  argv0,
  chdir,
  config,
  connected,
  constrainedMemory,
  availableMemory,
  cpuUsage,
  cwd,
  debugPort,
  dlopen,
  disconnect,
  emit,
  emitWarning,
  env,
  eventNames,
  execArgv,
  execPath,
  exit,
  finalization,
  features,
  getBuiltinModule,
  getActiveResourcesInfo,
  getMaxListeners,
  hrtime: hrtime3,
  kill,
  listeners,
  listenerCount,
  memoryUsage,
  nextTick,
  on,
  off,
  once,
  pid,
  platform,
  ppid,
  prependListener,
  prependOnceListener,
  rawListeners,
  release,
  removeAllListeners,
  removeListener,
  report,
  resourceUsage,
  setMaxListeners,
  setSourceMapsEnabled,
  stderr,
  stdin,
  stdout,
  title,
  throwDeprecation,
  traceDeprecation,
  umask,
  uptime,
  version,
  versions,
  // @ts-expect-error old API
  domain,
  initgroups,
  moduleLoadList,
  reallyExit,
  openStdin,
  assert,
  binding,
  send,
  exitCode,
  channel,
  getegid,
  geteuid,
  getgid,
  getgroups,
  getuid,
  setegid,
  seteuid,
  setgid,
  setgroups,
  setuid,
  permission,
  mainModule,
  _events,
  _eventsCount,
  _exiting,
  _maxListeners,
  _debugEnd,
  _debugProcess,
  _fatalException,
  _getActiveHandles,
  _getActiveRequests,
  _kill,
  _preload_modules,
  _rawDebug,
  _startProfilerIdleNotifier,
  _stopProfilerIdleNotifier,
  _tickCallback,
  _disconnect,
  _handleQueue,
  _pendingMessage,
  _channel,
  _send,
  _linkedBinding
};
var process_default = _process;

// node_modules/.pnpm/wrangler@4.62.0_@cloudflare+workers-types@4.20260205.0/node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-process
globalThis.process = process_default;

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/compose.js
var compose = /* @__PURE__ */ __name((middleware, onError, onNotFound) => {
  return (context, next) => {
    let index = -1;
    return dispatch(0);
    async function dispatch(i) {
      if (i <= index) {
        throw new Error("next() called multiple times");
      }
      index = i;
      let res;
      let isError = false;
      let handler;
      if (middleware[i]) {
        handler = middleware[i][0][0];
        context.req.routeIndex = i;
      } else {
        handler = i === middleware.length && next || void 0;
      }
      if (handler) {
        try {
          res = await handler(context, () => dispatch(i + 1));
        } catch (err) {
          if (err instanceof Error && onError) {
            context.error = err;
            res = await onError(err, context);
            isError = true;
          } else {
            throw err;
          }
        }
      } else {
        if (context.finalized === false && onNotFound) {
          res = await onNotFound(context);
        }
      }
      if (res && (context.finalized === false || isError)) {
        context.res = res;
      }
      return context;
    }
    __name(dispatch, "dispatch");
  };
}, "compose");

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/request/constants.js
var GET_MATCH_RESULT = /* @__PURE__ */ Symbol();

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/utils/body.js
var parseBody = /* @__PURE__ */ __name(async (request, options = /* @__PURE__ */ Object.create(null)) => {
  const { all = false, dot = false } = options;
  const headers = request instanceof HonoRequest ? request.raw.headers : request.headers;
  const contentType = headers.get("Content-Type");
  if (contentType?.startsWith("multipart/form-data") || contentType?.startsWith("application/x-www-form-urlencoded")) {
    return parseFormData(request, { all, dot });
  }
  return {};
}, "parseBody");
async function parseFormData(request, options) {
  const formData = await request.formData();
  if (formData) {
    return convertFormDataToBodyData(formData, options);
  }
  return {};
}
__name(parseFormData, "parseFormData");
function convertFormDataToBodyData(formData, options) {
  const form = /* @__PURE__ */ Object.create(null);
  formData.forEach((value, key) => {
    const shouldParseAllValues = options.all || key.endsWith("[]");
    if (!shouldParseAllValues) {
      form[key] = value;
    } else {
      handleParsingAllValues(form, key, value);
    }
  });
  if (options.dot) {
    Object.entries(form).forEach(([key, value]) => {
      const shouldParseDotValues = key.includes(".");
      if (shouldParseDotValues) {
        handleParsingNestedValues(form, key, value);
        delete form[key];
      }
    });
  }
  return form;
}
__name(convertFormDataToBodyData, "convertFormDataToBodyData");
var handleParsingAllValues = /* @__PURE__ */ __name((form, key, value) => {
  if (form[key] !== void 0) {
    if (Array.isArray(form[key])) {
      ;
      form[key].push(value);
    } else {
      form[key] = [form[key], value];
    }
  } else {
    if (!key.endsWith("[]")) {
      form[key] = value;
    } else {
      form[key] = [value];
    }
  }
}, "handleParsingAllValues");
var handleParsingNestedValues = /* @__PURE__ */ __name((form, key, value) => {
  let nestedForm = form;
  const keys = key.split(".");
  keys.forEach((key2, index) => {
    if (index === keys.length - 1) {
      nestedForm[key2] = value;
    } else {
      if (!nestedForm[key2] || typeof nestedForm[key2] !== "object" || Array.isArray(nestedForm[key2]) || nestedForm[key2] instanceof File) {
        nestedForm[key2] = /* @__PURE__ */ Object.create(null);
      }
      nestedForm = nestedForm[key2];
    }
  });
}, "handleParsingNestedValues");

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/utils/url.js
var splitPath = /* @__PURE__ */ __name((path) => {
  const paths = path.split("/");
  if (paths[0] === "") {
    paths.shift();
  }
  return paths;
}, "splitPath");
var splitRoutingPath = /* @__PURE__ */ __name((routePath) => {
  const { groups, path } = extractGroupsFromPath(routePath);
  const paths = splitPath(path);
  return replaceGroupMarks(paths, groups);
}, "splitRoutingPath");
var extractGroupsFromPath = /* @__PURE__ */ __name((path) => {
  const groups = [];
  path = path.replace(/\{[^}]+\}/g, (match2, index) => {
    const mark = `@${index}`;
    groups.push([mark, match2]);
    return mark;
  });
  return { groups, path };
}, "extractGroupsFromPath");
var replaceGroupMarks = /* @__PURE__ */ __name((paths, groups) => {
  for (let i = groups.length - 1; i >= 0; i--) {
    const [mark] = groups[i];
    for (let j = paths.length - 1; j >= 0; j--) {
      if (paths[j].includes(mark)) {
        paths[j] = paths[j].replace(mark, groups[i][1]);
        break;
      }
    }
  }
  return paths;
}, "replaceGroupMarks");
var patternCache = {};
var getPattern = /* @__PURE__ */ __name((label, next) => {
  if (label === "*") {
    return "*";
  }
  const match2 = label.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
  if (match2) {
    const cacheKey = `${label}#${next}`;
    if (!patternCache[cacheKey]) {
      if (match2[2]) {
        patternCache[cacheKey] = next && next[0] !== ":" && next[0] !== "*" ? [cacheKey, match2[1], new RegExp(`^${match2[2]}(?=/${next})`)] : [label, match2[1], new RegExp(`^${match2[2]}$`)];
      } else {
        patternCache[cacheKey] = [label, match2[1], true];
      }
    }
    return patternCache[cacheKey];
  }
  return null;
}, "getPattern");
var tryDecode = /* @__PURE__ */ __name((str, decoder) => {
  try {
    return decoder(str);
  } catch {
    return str.replace(/(?:%[0-9A-Fa-f]{2})+/g, (match2) => {
      try {
        return decoder(match2);
      } catch {
        return match2;
      }
    });
  }
}, "tryDecode");
var tryDecodeURI = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURI), "tryDecodeURI");
var getPath = /* @__PURE__ */ __name((request) => {
  const url = request.url;
  const start = url.indexOf("/", url.indexOf(":") + 4);
  let i = start;
  for (; i < url.length; i++) {
    const charCode = url.charCodeAt(i);
    if (charCode === 37) {
      const queryIndex = url.indexOf("?", i);
      const path = url.slice(start, queryIndex === -1 ? void 0 : queryIndex);
      return tryDecodeURI(path.includes("%25") ? path.replace(/%25/g, "%2525") : path);
    } else if (charCode === 63) {
      break;
    }
  }
  return url.slice(start, i);
}, "getPath");
var getPathNoStrict = /* @__PURE__ */ __name((request) => {
  const result = getPath(request);
  return result.length > 1 && result.at(-1) === "/" ? result.slice(0, -1) : result;
}, "getPathNoStrict");
var mergePath = /* @__PURE__ */ __name((base, sub, ...rest) => {
  if (rest.length) {
    sub = mergePath(sub, ...rest);
  }
  return `${base?.[0] === "/" ? "" : "/"}${base}${sub === "/" ? "" : `${base?.at(-1) === "/" ? "" : "/"}${sub?.[0] === "/" ? sub.slice(1) : sub}`}`;
}, "mergePath");
var checkOptionalParameter = /* @__PURE__ */ __name((path) => {
  if (path.charCodeAt(path.length - 1) !== 63 || !path.includes(":")) {
    return null;
  }
  const segments = path.split("/");
  const results = [];
  let basePath = "";
  segments.forEach((segment) => {
    if (segment !== "" && !/\:/.test(segment)) {
      basePath += "/" + segment;
    } else if (/\:/.test(segment)) {
      if (/\?/.test(segment)) {
        if (results.length === 0 && basePath === "") {
          results.push("/");
        } else {
          results.push(basePath);
        }
        const optionalSegment = segment.replace("?", "");
        basePath += "/" + optionalSegment;
        results.push(basePath);
      } else {
        basePath += "/" + segment;
      }
    }
  });
  return results.filter((v, i, a) => a.indexOf(v) === i);
}, "checkOptionalParameter");
var _decodeURI = /* @__PURE__ */ __name((value) => {
  if (!/[%+]/.test(value)) {
    return value;
  }
  if (value.indexOf("+") !== -1) {
    value = value.replace(/\+/g, " ");
  }
  return value.indexOf("%") !== -1 ? tryDecode(value, decodeURIComponent_) : value;
}, "_decodeURI");
var _getQueryParam = /* @__PURE__ */ __name((url, key, multiple) => {
  let encoded;
  if (!multiple && key && !/[%+]/.test(key)) {
    let keyIndex2 = url.indexOf("?", 8);
    if (keyIndex2 === -1) {
      return void 0;
    }
    if (!url.startsWith(key, keyIndex2 + 1)) {
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    while (keyIndex2 !== -1) {
      const trailingKeyCode = url.charCodeAt(keyIndex2 + key.length + 1);
      if (trailingKeyCode === 61) {
        const valueIndex = keyIndex2 + key.length + 2;
        const endIndex = url.indexOf("&", valueIndex);
        return _decodeURI(url.slice(valueIndex, endIndex === -1 ? void 0 : endIndex));
      } else if (trailingKeyCode == 38 || isNaN(trailingKeyCode)) {
        return "";
      }
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    encoded = /[%+]/.test(url);
    if (!encoded) {
      return void 0;
    }
  }
  const results = {};
  encoded ??= /[%+]/.test(url);
  let keyIndex = url.indexOf("?", 8);
  while (keyIndex !== -1) {
    const nextKeyIndex = url.indexOf("&", keyIndex + 1);
    let valueIndex = url.indexOf("=", keyIndex);
    if (valueIndex > nextKeyIndex && nextKeyIndex !== -1) {
      valueIndex = -1;
    }
    let name = url.slice(
      keyIndex + 1,
      valueIndex === -1 ? nextKeyIndex === -1 ? void 0 : nextKeyIndex : valueIndex
    );
    if (encoded) {
      name = _decodeURI(name);
    }
    keyIndex = nextKeyIndex;
    if (name === "") {
      continue;
    }
    let value;
    if (valueIndex === -1) {
      value = "";
    } else {
      value = url.slice(valueIndex + 1, nextKeyIndex === -1 ? void 0 : nextKeyIndex);
      if (encoded) {
        value = _decodeURI(value);
      }
    }
    if (multiple) {
      if (!(results[name] && Array.isArray(results[name]))) {
        results[name] = [];
      }
      ;
      results[name].push(value);
    } else {
      results[name] ??= value;
    }
  }
  return key ? results[key] : results;
}, "_getQueryParam");
var getQueryParam = _getQueryParam;
var getQueryParams = /* @__PURE__ */ __name((url, key) => {
  return _getQueryParam(url, key, true);
}, "getQueryParams");
var decodeURIComponent_ = decodeURIComponent;

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/request.js
var tryDecodeURIComponent = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURIComponent_), "tryDecodeURIComponent");
var HonoRequest = class {
  static {
    __name(this, "HonoRequest");
  }
  /**
   * `.raw` can get the raw Request object.
   *
   * @see {@link https://hono.dev/docs/api/request#raw}
   *
   * @example
   * ```ts
   * // For Cloudflare Workers
   * app.post('/', async (c) => {
   *   const metadata = c.req.raw.cf?.hostMetadata?
   *   ...
   * })
   * ```
   */
  raw;
  #validatedData;
  // Short name of validatedData
  #matchResult;
  routeIndex = 0;
  /**
   * `.path` can get the pathname of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#path}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const pathname = c.req.path // `/about/me`
   * })
   * ```
   */
  path;
  bodyCache = {};
  constructor(request, path = "/", matchResult = [[]]) {
    this.raw = request;
    this.path = path;
    this.#matchResult = matchResult;
    this.#validatedData = {};
  }
  param(key) {
    return key ? this.#getDecodedParam(key) : this.#getAllDecodedParams();
  }
  #getDecodedParam(key) {
    const paramKey = this.#matchResult[0][this.routeIndex][1][key];
    const param = this.#getParamValue(paramKey);
    return param && /\%/.test(param) ? tryDecodeURIComponent(param) : param;
  }
  #getAllDecodedParams() {
    const decoded = {};
    const keys = Object.keys(this.#matchResult[0][this.routeIndex][1]);
    for (const key of keys) {
      const value = this.#getParamValue(this.#matchResult[0][this.routeIndex][1][key]);
      if (value !== void 0) {
        decoded[key] = /\%/.test(value) ? tryDecodeURIComponent(value) : value;
      }
    }
    return decoded;
  }
  #getParamValue(paramKey) {
    return this.#matchResult[1] ? this.#matchResult[1][paramKey] : paramKey;
  }
  query(key) {
    return getQueryParam(this.url, key);
  }
  queries(key) {
    return getQueryParams(this.url, key);
  }
  header(name) {
    if (name) {
      return this.raw.headers.get(name) ?? void 0;
    }
    const headerData = {};
    this.raw.headers.forEach((value, key) => {
      headerData[key] = value;
    });
    return headerData;
  }
  async parseBody(options) {
    return this.bodyCache.parsedBody ??= await parseBody(this, options);
  }
  #cachedBody = /* @__PURE__ */ __name((key) => {
    const { bodyCache, raw: raw2 } = this;
    const cachedBody = bodyCache[key];
    if (cachedBody) {
      return cachedBody;
    }
    const anyCachedKey = Object.keys(bodyCache)[0];
    if (anyCachedKey) {
      return bodyCache[anyCachedKey].then((body) => {
        if (anyCachedKey === "json") {
          body = JSON.stringify(body);
        }
        return new Response(body)[key]();
      });
    }
    return bodyCache[key] = raw2[key]();
  }, "#cachedBody");
  /**
   * `.json()` can parse Request body of type `application/json`
   *
   * @see {@link https://hono.dev/docs/api/request#json}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.json()
   * })
   * ```
   */
  json() {
    return this.#cachedBody("text").then((text) => JSON.parse(text));
  }
  /**
   * `.text()` can parse Request body of type `text/plain`
   *
   * @see {@link https://hono.dev/docs/api/request#text}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.text()
   * })
   * ```
   */
  text() {
    return this.#cachedBody("text");
  }
  /**
   * `.arrayBuffer()` parse Request body as an `ArrayBuffer`
   *
   * @see {@link https://hono.dev/docs/api/request#arraybuffer}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.arrayBuffer()
   * })
   * ```
   */
  arrayBuffer() {
    return this.#cachedBody("arrayBuffer");
  }
  /**
   * Parses the request body as a `Blob`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.blob();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#blob
   */
  blob() {
    return this.#cachedBody("blob");
  }
  /**
   * Parses the request body as `FormData`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.formData();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#formdata
   */
  formData() {
    return this.#cachedBody("formData");
  }
  /**
   * Adds validated data to the request.
   *
   * @param target - The target of the validation.
   * @param data - The validated data to add.
   */
  addValidatedData(target, data) {
    this.#validatedData[target] = data;
  }
  valid(target) {
    return this.#validatedData[target];
  }
  /**
   * `.url()` can get the request url strings.
   *
   * @see {@link https://hono.dev/docs/api/request#url}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const url = c.req.url // `http://localhost:8787/about/me`
   *   ...
   * })
   * ```
   */
  get url() {
    return this.raw.url;
  }
  /**
   * `.method()` can get the method name of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#method}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const method = c.req.method // `GET`
   * })
   * ```
   */
  get method() {
    return this.raw.method;
  }
  get [GET_MATCH_RESULT]() {
    return this.#matchResult;
  }
  /**
   * `.matchedRoutes()` can return a matched route in the handler
   *
   * @deprecated
   *
   * Use matchedRoutes helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#matchedroutes}
   *
   * @example
   * ```ts
   * app.use('*', async function logger(c, next) {
   *   await next()
   *   c.req.matchedRoutes.forEach(({ handler, method, path }, i) => {
   *     const name = handler.name || (handler.length < 2 ? '[handler]' : '[middleware]')
   *     console.log(
   *       method,
   *       ' ',
   *       path,
   *       ' '.repeat(Math.max(10 - path.length, 0)),
   *       name,
   *       i === c.req.routeIndex ? '<- respond from here' : ''
   *     )
   *   })
   * })
   * ```
   */
  get matchedRoutes() {
    return this.#matchResult[0].map(([[, route]]) => route);
  }
  /**
   * `routePath()` can retrieve the path registered within the handler
   *
   * @deprecated
   *
   * Use routePath helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#routepath}
   *
   * @example
   * ```ts
   * app.get('/posts/:id', (c) => {
   *   return c.json({ path: c.req.routePath })
   * })
   * ```
   */
  get routePath() {
    return this.#matchResult[0].map(([[, route]]) => route)[this.routeIndex].path;
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/utils/html.js
var HtmlEscapedCallbackPhase = {
  Stringify: 1,
  BeforeStream: 2,
  Stream: 3
};
var raw = /* @__PURE__ */ __name((value, callbacks) => {
  const escapedString = new String(value);
  escapedString.isEscaped = true;
  escapedString.callbacks = callbacks;
  return escapedString;
}, "raw");
var resolveCallback = /* @__PURE__ */ __name(async (str, phase, preserveCallbacks, context, buffer) => {
  if (typeof str === "object" && !(str instanceof String)) {
    if (!(str instanceof Promise)) {
      str = str.toString();
    }
    if (str instanceof Promise) {
      str = await str;
    }
  }
  const callbacks = str.callbacks;
  if (!callbacks?.length) {
    return Promise.resolve(str);
  }
  if (buffer) {
    buffer[0] += str;
  } else {
    buffer = [str];
  }
  const resStr = Promise.all(callbacks.map((c) => c({ phase, buffer, context }))).then(
    (res) => Promise.all(
      res.filter(Boolean).map((str2) => resolveCallback(str2, phase, false, context, buffer))
    ).then(() => buffer[0])
  );
  if (preserveCallbacks) {
    return raw(await resStr, callbacks);
  } else {
    return resStr;
  }
}, "resolveCallback");

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/context.js
var TEXT_PLAIN = "text/plain; charset=UTF-8";
var setDefaultContentType = /* @__PURE__ */ __name((contentType, headers) => {
  return {
    "Content-Type": contentType,
    ...headers
  };
}, "setDefaultContentType");
var Context = class {
  static {
    __name(this, "Context");
  }
  #rawRequest;
  #req;
  /**
   * `.env` can get bindings (environment variables, secrets, KV namespaces, D1 database, R2 bucket etc.) in Cloudflare Workers.
   *
   * @see {@link https://hono.dev/docs/api/context#env}
   *
   * @example
   * ```ts
   * // Environment object for Cloudflare Workers
   * app.get('*', async c => {
   *   const counter = c.env.COUNTER
   * })
   * ```
   */
  env = {};
  #var;
  finalized = false;
  /**
   * `.error` can get the error object from the middleware if the Handler throws an error.
   *
   * @see {@link https://hono.dev/docs/api/context#error}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   await next()
   *   if (c.error) {
   *     // do something...
   *   }
   * })
   * ```
   */
  error;
  #status;
  #executionCtx;
  #res;
  #layout;
  #renderer;
  #notFoundHandler;
  #preparedHeaders;
  #matchResult;
  #path;
  /**
   * Creates an instance of the Context class.
   *
   * @param req - The Request object.
   * @param options - Optional configuration options for the context.
   */
  constructor(req, options) {
    this.#rawRequest = req;
    if (options) {
      this.#executionCtx = options.executionCtx;
      this.env = options.env;
      this.#notFoundHandler = options.notFoundHandler;
      this.#path = options.path;
      this.#matchResult = options.matchResult;
    }
  }
  /**
   * `.req` is the instance of {@link HonoRequest}.
   */
  get req() {
    this.#req ??= new HonoRequest(this.#rawRequest, this.#path, this.#matchResult);
    return this.#req;
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#event}
   * The FetchEvent associated with the current request.
   *
   * @throws Will throw an error if the context does not have a FetchEvent.
   */
  get event() {
    if (this.#executionCtx && "respondWith" in this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no FetchEvent");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#executionctx}
   * The ExecutionContext associated with the current request.
   *
   * @throws Will throw an error if the context does not have an ExecutionContext.
   */
  get executionCtx() {
    if (this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no ExecutionContext");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#res}
   * The Response object for the current request.
   */
  get res() {
    return this.#res ||= new Response(null, {
      headers: this.#preparedHeaders ??= new Headers()
    });
  }
  /**
   * Sets the Response object for the current request.
   *
   * @param _res - The Response object to set.
   */
  set res(_res) {
    if (this.#res && _res) {
      _res = new Response(_res.body, _res);
      for (const [k, v] of this.#res.headers.entries()) {
        if (k === "content-type") {
          continue;
        }
        if (k === "set-cookie") {
          const cookies = this.#res.headers.getSetCookie();
          _res.headers.delete("set-cookie");
          for (const cookie of cookies) {
            _res.headers.append("set-cookie", cookie);
          }
        } else {
          _res.headers.set(k, v);
        }
      }
    }
    this.#res = _res;
    this.finalized = true;
  }
  /**
   * `.render()` can create a response within a layout.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   return c.render('Hello!')
   * })
   * ```
   */
  render = /* @__PURE__ */ __name((...args) => {
    this.#renderer ??= (content) => this.html(content);
    return this.#renderer(...args);
  }, "render");
  /**
   * Sets the layout for the response.
   *
   * @param layout - The layout to set.
   * @returns The layout function.
   */
  setLayout = /* @__PURE__ */ __name((layout) => this.#layout = layout, "setLayout");
  /**
   * Gets the current layout for the response.
   *
   * @returns The current layout function.
   */
  getLayout = /* @__PURE__ */ __name(() => this.#layout, "getLayout");
  /**
   * `.setRenderer()` can set the layout in the custom middleware.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```tsx
   * app.use('*', async (c, next) => {
   *   c.setRenderer((content) => {
   *     return c.html(
   *       <html>
   *         <body>
   *           <p>{content}</p>
   *         </body>
   *       </html>
   *     )
   *   })
   *   await next()
   * })
   * ```
   */
  setRenderer = /* @__PURE__ */ __name((renderer) => {
    this.#renderer = renderer;
  }, "setRenderer");
  /**
   * `.header()` can set headers.
   *
   * @see {@link https://hono.dev/docs/api/context#header}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  header = /* @__PURE__ */ __name((name, value, options) => {
    if (this.finalized) {
      this.#res = new Response(this.#res.body, this.#res);
    }
    const headers = this.#res ? this.#res.headers : this.#preparedHeaders ??= new Headers();
    if (value === void 0) {
      headers.delete(name);
    } else if (options?.append) {
      headers.append(name, value);
    } else {
      headers.set(name, value);
    }
  }, "header");
  status = /* @__PURE__ */ __name((status) => {
    this.#status = status;
  }, "status");
  /**
   * `.set()` can set the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   c.set('message', 'Hono is hot!!')
   *   await next()
   * })
   * ```
   */
  set = /* @__PURE__ */ __name((key, value) => {
    this.#var ??= /* @__PURE__ */ new Map();
    this.#var.set(key, value);
  }, "set");
  /**
   * `.get()` can use the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   const message = c.get('message')
   *   return c.text(`The message is "${message}"`)
   * })
   * ```
   */
  get = /* @__PURE__ */ __name((key) => {
    return this.#var ? this.#var.get(key) : void 0;
  }, "get");
  /**
   * `.var` can access the value of a variable.
   *
   * @see {@link https://hono.dev/docs/api/context#var}
   *
   * @example
   * ```ts
   * const result = c.var.client.oneMethod()
   * ```
   */
  // c.var.propName is a read-only
  get var() {
    if (!this.#var) {
      return {};
    }
    return Object.fromEntries(this.#var);
  }
  #newResponse(data, arg, headers) {
    const responseHeaders = this.#res ? new Headers(this.#res.headers) : this.#preparedHeaders ?? new Headers();
    if (typeof arg === "object" && "headers" in arg) {
      const argHeaders = arg.headers instanceof Headers ? arg.headers : new Headers(arg.headers);
      for (const [key, value] of argHeaders) {
        if (key.toLowerCase() === "set-cookie") {
          responseHeaders.append(key, value);
        } else {
          responseHeaders.set(key, value);
        }
      }
    }
    if (headers) {
      for (const [k, v] of Object.entries(headers)) {
        if (typeof v === "string") {
          responseHeaders.set(k, v);
        } else {
          responseHeaders.delete(k);
          for (const v2 of v) {
            responseHeaders.append(k, v2);
          }
        }
      }
    }
    const status = typeof arg === "number" ? arg : arg?.status ?? this.#status;
    return new Response(data, { status, headers: responseHeaders });
  }
  newResponse = /* @__PURE__ */ __name((...args) => this.#newResponse(...args), "newResponse");
  /**
   * `.body()` can return the HTTP response.
   * You can set headers with `.header()` and set HTTP status code with `.status`.
   * This can also be set in `.text()`, `.json()` and so on.
   *
   * @see {@link https://hono.dev/docs/api/context#body}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *   // Set HTTP status code
   *   c.status(201)
   *
   *   // Return the response body
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  body = /* @__PURE__ */ __name((data, arg, headers) => this.#newResponse(data, arg, headers), "body");
  /**
   * `.text()` can render text as `Content-Type:text/plain`.
   *
   * @see {@link https://hono.dev/docs/api/context#text}
   *
   * @example
   * ```ts
   * app.get('/say', (c) => {
   *   return c.text('Hello!')
   * })
   * ```
   */
  text = /* @__PURE__ */ __name((text, arg, headers) => {
    return !this.#preparedHeaders && !this.#status && !arg && !headers && !this.finalized ? new Response(text) : this.#newResponse(
      text,
      arg,
      setDefaultContentType(TEXT_PLAIN, headers)
    );
  }, "text");
  /**
   * `.json()` can render JSON as `Content-Type:application/json`.
   *
   * @see {@link https://hono.dev/docs/api/context#json}
   *
   * @example
   * ```ts
   * app.get('/api', (c) => {
   *   return c.json({ message: 'Hello!' })
   * })
   * ```
   */
  json = /* @__PURE__ */ __name((object, arg, headers) => {
    return this.#newResponse(
      JSON.stringify(object),
      arg,
      setDefaultContentType("application/json", headers)
    );
  }, "json");
  html = /* @__PURE__ */ __name((html, arg, headers) => {
    const res = /* @__PURE__ */ __name((html2) => this.#newResponse(html2, arg, setDefaultContentType("text/html; charset=UTF-8", headers)), "res");
    return typeof html === "object" ? resolveCallback(html, HtmlEscapedCallbackPhase.Stringify, false, {}).then(res) : res(html);
  }, "html");
  /**
   * `.redirect()` can Redirect, default status code is 302.
   *
   * @see {@link https://hono.dev/docs/api/context#redirect}
   *
   * @example
   * ```ts
   * app.get('/redirect', (c) => {
   *   return c.redirect('/')
   * })
   * app.get('/redirect-permanently', (c) => {
   *   return c.redirect('/', 301)
   * })
   * ```
   */
  redirect = /* @__PURE__ */ __name((location, status) => {
    const locationString = String(location);
    this.header(
      "Location",
      // Multibyes should be encoded
      // eslint-disable-next-line no-control-regex
      !/[^\x00-\xFF]/.test(locationString) ? locationString : encodeURI(locationString)
    );
    return this.newResponse(null, status ?? 302);
  }, "redirect");
  /**
   * `.notFound()` can return the Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/context#notfound}
   *
   * @example
   * ```ts
   * app.get('/notfound', (c) => {
   *   return c.notFound()
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name(() => {
    this.#notFoundHandler ??= () => new Response();
    return this.#notFoundHandler(this);
  }, "notFound");
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router.js
var METHOD_NAME_ALL = "ALL";
var METHOD_NAME_ALL_LOWERCASE = "all";
var METHODS = ["get", "post", "put", "delete", "options", "patch"];
var MESSAGE_MATCHER_IS_ALREADY_BUILT = "Can not add a route since the matcher is already built.";
var UnsupportedPathError = class extends Error {
  static {
    __name(this, "UnsupportedPathError");
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/utils/constants.js
var COMPOSED_HANDLER = "__COMPOSED_HANDLER";

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/hono-base.js
var notFoundHandler = /* @__PURE__ */ __name((c) => {
  return c.text("404 Not Found", 404);
}, "notFoundHandler");
var errorHandler = /* @__PURE__ */ __name((err, c) => {
  if ("getResponse" in err) {
    const res = err.getResponse();
    return c.newResponse(res.body, res);
  }
  console.error(err);
  return c.text("Internal Server Error", 500);
}, "errorHandler");
var Hono = class _Hono {
  static {
    __name(this, "_Hono");
  }
  get;
  post;
  put;
  delete;
  options;
  patch;
  all;
  on;
  use;
  /*
    This class is like an abstract class and does not have a router.
    To use it, inherit the class and implement router in the constructor.
  */
  router;
  getPath;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  _basePath = "/";
  #path = "/";
  routes = [];
  constructor(options = {}) {
    const allMethods = [...METHODS, METHOD_NAME_ALL_LOWERCASE];
    allMethods.forEach((method) => {
      this[method] = (args1, ...args) => {
        if (typeof args1 === "string") {
          this.#path = args1;
        } else {
          this.#addRoute(method, this.#path, args1);
        }
        args.forEach((handler) => {
          this.#addRoute(method, this.#path, handler);
        });
        return this;
      };
    });
    this.on = (method, path, ...handlers) => {
      for (const p of [path].flat()) {
        this.#path = p;
        for (const m of [method].flat()) {
          handlers.map((handler) => {
            this.#addRoute(m.toUpperCase(), this.#path, handler);
          });
        }
      }
      return this;
    };
    this.use = (arg1, ...handlers) => {
      if (typeof arg1 === "string") {
        this.#path = arg1;
      } else {
        this.#path = "*";
        handlers.unshift(arg1);
      }
      handlers.forEach((handler) => {
        this.#addRoute(METHOD_NAME_ALL, this.#path, handler);
      });
      return this;
    };
    const { strict, ...optionsWithoutStrict } = options;
    Object.assign(this, optionsWithoutStrict);
    this.getPath = strict ?? true ? options.getPath ?? getPath : getPathNoStrict;
  }
  #clone() {
    const clone = new _Hono({
      router: this.router,
      getPath: this.getPath
    });
    clone.errorHandler = this.errorHandler;
    clone.#notFoundHandler = this.#notFoundHandler;
    clone.routes = this.routes;
    return clone;
  }
  #notFoundHandler = notFoundHandler;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  errorHandler = errorHandler;
  /**
   * `.route()` allows grouping other Hono instance in routes.
   *
   * @see {@link https://hono.dev/docs/api/routing#grouping}
   *
   * @param {string} path - base Path
   * @param {Hono} app - other Hono instance
   * @returns {Hono} routed Hono instance
   *
   * @example
   * ```ts
   * const app = new Hono()
   * const app2 = new Hono()
   *
   * app2.get("/user", (c) => c.text("user"))
   * app.route("/api", app2) // GET /api/user
   * ```
   */
  route(path, app12) {
    const subApp = this.basePath(path);
    app12.routes.map((r) => {
      let handler;
      if (app12.errorHandler === errorHandler) {
        handler = r.handler;
      } else {
        handler = /* @__PURE__ */ __name(async (c, next) => (await compose([], app12.errorHandler)(c, () => r.handler(c, next))).res, "handler");
        handler[COMPOSED_HANDLER] = r.handler;
      }
      subApp.#addRoute(r.method, r.path, handler);
    });
    return this;
  }
  /**
   * `.basePath()` allows base paths to be specified.
   *
   * @see {@link https://hono.dev/docs/api/routing#base-path}
   *
   * @param {string} path - base Path
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * const api = new Hono().basePath('/api')
   * ```
   */
  basePath(path) {
    const subApp = this.#clone();
    subApp._basePath = mergePath(this._basePath, path);
    return subApp;
  }
  /**
   * `.onError()` handles an error and returns a customized Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#error-handling}
   *
   * @param {ErrorHandler} handler - request Handler for error
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.onError((err, c) => {
   *   console.error(`${err}`)
   *   return c.text('Custom Error Message', 500)
   * })
   * ```
   */
  onError = /* @__PURE__ */ __name((handler) => {
    this.errorHandler = handler;
    return this;
  }, "onError");
  /**
   * `.notFound()` allows you to customize a Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#not-found}
   *
   * @param {NotFoundHandler} handler - request handler for not-found
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.notFound((c) => {
   *   return c.text('Custom 404 Message', 404)
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name((handler) => {
    this.#notFoundHandler = handler;
    return this;
  }, "notFound");
  /**
   * `.mount()` allows you to mount applications built with other frameworks into your Hono application.
   *
   * @see {@link https://hono.dev/docs/api/hono#mount}
   *
   * @param {string} path - base Path
   * @param {Function} applicationHandler - other Request Handler
   * @param {MountOptions} [options] - options of `.mount()`
   * @returns {Hono} mounted Hono instance
   *
   * @example
   * ```ts
   * import { Router as IttyRouter } from 'itty-router'
   * import { Hono } from 'hono'
   * // Create itty-router application
   * const ittyRouter = IttyRouter()
   * // GET /itty-router/hello
   * ittyRouter.get('/hello', () => new Response('Hello from itty-router'))
   *
   * const app = new Hono()
   * app.mount('/itty-router', ittyRouter.handle)
   * ```
   *
   * @example
   * ```ts
   * const app = new Hono()
   * // Send the request to another application without modification.
   * app.mount('/app', anotherApp, {
   *   replaceRequest: (req) => req,
   * })
   * ```
   */
  mount(path, applicationHandler, options) {
    let replaceRequest;
    let optionHandler;
    if (options) {
      if (typeof options === "function") {
        optionHandler = options;
      } else {
        optionHandler = options.optionHandler;
        if (options.replaceRequest === false) {
          replaceRequest = /* @__PURE__ */ __name((request) => request, "replaceRequest");
        } else {
          replaceRequest = options.replaceRequest;
        }
      }
    }
    const getOptions = optionHandler ? (c) => {
      const options2 = optionHandler(c);
      return Array.isArray(options2) ? options2 : [options2];
    } : (c) => {
      let executionContext = void 0;
      try {
        executionContext = c.executionCtx;
      } catch {
      }
      return [c.env, executionContext];
    };
    replaceRequest ||= (() => {
      const mergedPath = mergePath(this._basePath, path);
      const pathPrefixLength = mergedPath === "/" ? 0 : mergedPath.length;
      return (request) => {
        const url = new URL(request.url);
        url.pathname = url.pathname.slice(pathPrefixLength) || "/";
        return new Request(url, request);
      };
    })();
    const handler = /* @__PURE__ */ __name(async (c, next) => {
      const res = await applicationHandler(replaceRequest(c.req.raw), ...getOptions(c));
      if (res) {
        return res;
      }
      await next();
    }, "handler");
    this.#addRoute(METHOD_NAME_ALL, mergePath(path, "*"), handler);
    return this;
  }
  #addRoute(method, path, handler) {
    method = method.toUpperCase();
    path = mergePath(this._basePath, path);
    const r = { basePath: this._basePath, path, method, handler };
    this.router.add(method, path, [handler, r]);
    this.routes.push(r);
  }
  #handleError(err, c) {
    if (err instanceof Error) {
      return this.errorHandler(err, c);
    }
    throw err;
  }
  #dispatch(request, executionCtx, env2, method) {
    if (method === "HEAD") {
      return (async () => new Response(null, await this.#dispatch(request, executionCtx, env2, "GET")))();
    }
    const path = this.getPath(request, { env: env2 });
    const matchResult = this.router.match(method, path);
    const c = new Context(request, {
      path,
      matchResult,
      env: env2,
      executionCtx,
      notFoundHandler: this.#notFoundHandler
    });
    if (matchResult[0].length === 1) {
      let res;
      try {
        res = matchResult[0][0][0][0](c, async () => {
          c.res = await this.#notFoundHandler(c);
        });
      } catch (err) {
        return this.#handleError(err, c);
      }
      return res instanceof Promise ? res.then(
        (resolved) => resolved || (c.finalized ? c.res : this.#notFoundHandler(c))
      ).catch((err) => this.#handleError(err, c)) : res ?? this.#notFoundHandler(c);
    }
    const composed = compose(matchResult[0], this.errorHandler, this.#notFoundHandler);
    return (async () => {
      try {
        const context = await composed(c);
        if (!context.finalized) {
          throw new Error(
            "Context is not finalized. Did you forget to return a Response object or `await next()`?"
          );
        }
        return context.res;
      } catch (err) {
        return this.#handleError(err, c);
      }
    })();
  }
  /**
   * `.fetch()` will be entry point of your app.
   *
   * @see {@link https://hono.dev/docs/api/hono#fetch}
   *
   * @param {Request} request - request Object of request
   * @param {Env} Env - env Object
   * @param {ExecutionContext} - context of execution
   * @returns {Response | Promise<Response>} response of request
   *
   */
  fetch = /* @__PURE__ */ __name((request, ...rest) => {
    return this.#dispatch(request, rest[1], rest[0], request.method);
  }, "fetch");
  /**
   * `.request()` is a useful method for testing.
   * You can pass a URL or pathname to send a GET request.
   * app will return a Response object.
   * ```ts
   * test('GET /hello is ok', async () => {
   *   const res = await app.request('/hello')
   *   expect(res.status).toBe(200)
   * })
   * ```
   * @see https://hono.dev/docs/api/hono#request
   */
  request = /* @__PURE__ */ __name((input, requestInit, Env, executionCtx) => {
    if (input instanceof Request) {
      return this.fetch(requestInit ? new Request(input, requestInit) : input, Env, executionCtx);
    }
    input = input.toString();
    return this.fetch(
      new Request(
        /^https?:\/\//.test(input) ? input : `http://localhost${mergePath("/", input)}`,
        requestInit
      ),
      Env,
      executionCtx
    );
  }, "request");
  /**
   * `.fire()` automatically adds a global fetch event listener.
   * This can be useful for environments that adhere to the Service Worker API, such as non-ES module Cloudflare Workers.
   * @deprecated
   * Use `fire` from `hono/service-worker` instead.
   * ```ts
   * import { Hono } from 'hono'
   * import { fire } from 'hono/service-worker'
   *
   * const app = new Hono()
   * // ...
   * fire(app)
   * ```
   * @see https://hono.dev/docs/api/hono#fire
   * @see https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API
   * @see https://developers.cloudflare.com/workers/reference/migrate-to-module-workers/
   */
  fire = /* @__PURE__ */ __name(() => {
    addEventListener("fetch", (event) => {
      event.respondWith(this.#dispatch(event.request, event, void 0, event.request.method));
    });
  }, "fire");
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/reg-exp-router/matcher.js
var emptyParam = [];
function match(method, path) {
  const matchers = this.buildAllMatchers();
  const match2 = /* @__PURE__ */ __name(((method2, path2) => {
    const matcher = matchers[method2] || matchers[METHOD_NAME_ALL];
    const staticMatch = matcher[2][path2];
    if (staticMatch) {
      return staticMatch;
    }
    const match3 = path2.match(matcher[0]);
    if (!match3) {
      return [[], emptyParam];
    }
    const index = match3.indexOf("", 1);
    return [matcher[1][index], match3];
  }), "match2");
  this.match = match2;
  return match2(method, path);
}
__name(match, "match");

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/reg-exp-router/node.js
var LABEL_REG_EXP_STR = "[^/]+";
var ONLY_WILDCARD_REG_EXP_STR = ".*";
var TAIL_WILDCARD_REG_EXP_STR = "(?:|/.*)";
var PATH_ERROR = /* @__PURE__ */ Symbol();
var regExpMetaChars = new Set(".\\+*[^]$()");
function compareKey(a, b) {
  if (a.length === 1) {
    return b.length === 1 ? a < b ? -1 : 1 : -1;
  }
  if (b.length === 1) {
    return 1;
  }
  if (a === ONLY_WILDCARD_REG_EXP_STR || a === TAIL_WILDCARD_REG_EXP_STR) {
    return 1;
  } else if (b === ONLY_WILDCARD_REG_EXP_STR || b === TAIL_WILDCARD_REG_EXP_STR) {
    return -1;
  }
  if (a === LABEL_REG_EXP_STR) {
    return 1;
  } else if (b === LABEL_REG_EXP_STR) {
    return -1;
  }
  return a.length === b.length ? a < b ? -1 : 1 : b.length - a.length;
}
__name(compareKey, "compareKey");
var Node = class _Node {
  static {
    __name(this, "_Node");
  }
  #index;
  #varIndex;
  #children = /* @__PURE__ */ Object.create(null);
  insert(tokens, index, paramMap, context, pathErrorCheckOnly) {
    if (tokens.length === 0) {
      if (this.#index !== void 0) {
        throw PATH_ERROR;
      }
      if (pathErrorCheckOnly) {
        return;
      }
      this.#index = index;
      return;
    }
    const [token, ...restTokens] = tokens;
    const pattern = token === "*" ? restTokens.length === 0 ? ["", "", ONLY_WILDCARD_REG_EXP_STR] : ["", "", LABEL_REG_EXP_STR] : token === "/*" ? ["", "", TAIL_WILDCARD_REG_EXP_STR] : token.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
    let node;
    if (pattern) {
      const name = pattern[1];
      let regexpStr = pattern[2] || LABEL_REG_EXP_STR;
      if (name && pattern[2]) {
        if (regexpStr === ".*") {
          throw PATH_ERROR;
        }
        regexpStr = regexpStr.replace(/^\((?!\?:)(?=[^)]+\)$)/, "(?:");
        if (/\((?!\?:)/.test(regexpStr)) {
          throw PATH_ERROR;
        }
      }
      node = this.#children[regexpStr];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[regexpStr] = new _Node();
        if (name !== "") {
          node.#varIndex = context.varIndex++;
        }
      }
      if (!pathErrorCheckOnly && name !== "") {
        paramMap.push([name, node.#varIndex]);
      }
    } else {
      node = this.#children[token];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k.length > 1 && k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[token] = new _Node();
      }
    }
    node.insert(restTokens, index, paramMap, context, pathErrorCheckOnly);
  }
  buildRegExpStr() {
    const childKeys = Object.keys(this.#children).sort(compareKey);
    const strList = childKeys.map((k) => {
      const c = this.#children[k];
      return (typeof c.#varIndex === "number" ? `(${k})@${c.#varIndex}` : regExpMetaChars.has(k) ? `\\${k}` : k) + c.buildRegExpStr();
    });
    if (typeof this.#index === "number") {
      strList.unshift(`#${this.#index}`);
    }
    if (strList.length === 0) {
      return "";
    }
    if (strList.length === 1) {
      return strList[0];
    }
    return "(?:" + strList.join("|") + ")";
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/reg-exp-router/trie.js
var Trie = class {
  static {
    __name(this, "Trie");
  }
  #context = { varIndex: 0 };
  #root = new Node();
  insert(path, index, pathErrorCheckOnly) {
    const paramAssoc = [];
    const groups = [];
    for (let i = 0; ; ) {
      let replaced = false;
      path = path.replace(/\{[^}]+\}/g, (m) => {
        const mark = `@\\${i}`;
        groups[i] = [mark, m];
        i++;
        replaced = true;
        return mark;
      });
      if (!replaced) {
        break;
      }
    }
    const tokens = path.match(/(?::[^\/]+)|(?:\/\*$)|./g) || [];
    for (let i = groups.length - 1; i >= 0; i--) {
      const [mark] = groups[i];
      for (let j = tokens.length - 1; j >= 0; j--) {
        if (tokens[j].indexOf(mark) !== -1) {
          tokens[j] = tokens[j].replace(mark, groups[i][1]);
          break;
        }
      }
    }
    this.#root.insert(tokens, index, paramAssoc, this.#context, pathErrorCheckOnly);
    return paramAssoc;
  }
  buildRegExp() {
    let regexp = this.#root.buildRegExpStr();
    if (regexp === "") {
      return [/^$/, [], []];
    }
    let captureIndex = 0;
    const indexReplacementMap = [];
    const paramReplacementMap = [];
    regexp = regexp.replace(/#(\d+)|@(\d+)|\.\*\$/g, (_, handlerIndex, paramIndex) => {
      if (handlerIndex !== void 0) {
        indexReplacementMap[++captureIndex] = Number(handlerIndex);
        return "$()";
      }
      if (paramIndex !== void 0) {
        paramReplacementMap[Number(paramIndex)] = ++captureIndex;
        return "";
      }
      return "";
    });
    return [new RegExp(`^${regexp}`), indexReplacementMap, paramReplacementMap];
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/reg-exp-router/router.js
var nullMatcher = [/^$/, [], /* @__PURE__ */ Object.create(null)];
var wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
function buildWildcardRegExp(path) {
  return wildcardRegExpCache[path] ??= new RegExp(
    path === "*" ? "" : `^${path.replace(
      /\/\*$|([.\\+*[^\]$()])/g,
      (_, metaChar) => metaChar ? `\\${metaChar}` : "(?:|/.*)"
    )}$`
  );
}
__name(buildWildcardRegExp, "buildWildcardRegExp");
function clearWildcardRegExpCache() {
  wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
}
__name(clearWildcardRegExpCache, "clearWildcardRegExpCache");
function buildMatcherFromPreprocessedRoutes(routes) {
  const trie = new Trie();
  const handlerData = [];
  if (routes.length === 0) {
    return nullMatcher;
  }
  const routesWithStaticPathFlag = routes.map(
    (route) => [!/\*|\/:/.test(route[0]), ...route]
  ).sort(
    ([isStaticA, pathA], [isStaticB, pathB]) => isStaticA ? 1 : isStaticB ? -1 : pathA.length - pathB.length
  );
  const staticMap = /* @__PURE__ */ Object.create(null);
  for (let i = 0, j = -1, len = routesWithStaticPathFlag.length; i < len; i++) {
    const [pathErrorCheckOnly, path, handlers] = routesWithStaticPathFlag[i];
    if (pathErrorCheckOnly) {
      staticMap[path] = [handlers.map(([h]) => [h, /* @__PURE__ */ Object.create(null)]), emptyParam];
    } else {
      j++;
    }
    let paramAssoc;
    try {
      paramAssoc = trie.insert(path, j, pathErrorCheckOnly);
    } catch (e) {
      throw e === PATH_ERROR ? new UnsupportedPathError(path) : e;
    }
    if (pathErrorCheckOnly) {
      continue;
    }
    handlerData[j] = handlers.map(([h, paramCount]) => {
      const paramIndexMap = /* @__PURE__ */ Object.create(null);
      paramCount -= 1;
      for (; paramCount >= 0; paramCount--) {
        const [key, value] = paramAssoc[paramCount];
        paramIndexMap[key] = value;
      }
      return [h, paramIndexMap];
    });
  }
  const [regexp, indexReplacementMap, paramReplacementMap] = trie.buildRegExp();
  for (let i = 0, len = handlerData.length; i < len; i++) {
    for (let j = 0, len2 = handlerData[i].length; j < len2; j++) {
      const map = handlerData[i][j]?.[1];
      if (!map) {
        continue;
      }
      const keys = Object.keys(map);
      for (let k = 0, len3 = keys.length; k < len3; k++) {
        map[keys[k]] = paramReplacementMap[map[keys[k]]];
      }
    }
  }
  const handlerMap = [];
  for (const i in indexReplacementMap) {
    handlerMap[i] = handlerData[indexReplacementMap[i]];
  }
  return [regexp, handlerMap, staticMap];
}
__name(buildMatcherFromPreprocessedRoutes, "buildMatcherFromPreprocessedRoutes");
function findMiddleware(middleware, path) {
  if (!middleware) {
    return void 0;
  }
  for (const k of Object.keys(middleware).sort((a, b) => b.length - a.length)) {
    if (buildWildcardRegExp(k).test(path)) {
      return [...middleware[k]];
    }
  }
  return void 0;
}
__name(findMiddleware, "findMiddleware");
var RegExpRouter = class {
  static {
    __name(this, "RegExpRouter");
  }
  name = "RegExpRouter";
  #middleware;
  #routes;
  constructor() {
    this.#middleware = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
    this.#routes = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
  }
  add(method, path, handler) {
    const middleware = this.#middleware;
    const routes = this.#routes;
    if (!middleware || !routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    if (!middleware[method]) {
      ;
      [middleware, routes].forEach((handlerMap) => {
        handlerMap[method] = /* @__PURE__ */ Object.create(null);
        Object.keys(handlerMap[METHOD_NAME_ALL]).forEach((p) => {
          handlerMap[method][p] = [...handlerMap[METHOD_NAME_ALL][p]];
        });
      });
    }
    if (path === "/*") {
      path = "*";
    }
    const paramCount = (path.match(/\/:/g) || []).length;
    if (/\*$/.test(path)) {
      const re = buildWildcardRegExp(path);
      if (method === METHOD_NAME_ALL) {
        Object.keys(middleware).forEach((m) => {
          middleware[m][path] ||= findMiddleware(middleware[m], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
        });
      } else {
        middleware[method][path] ||= findMiddleware(middleware[method], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
      }
      Object.keys(middleware).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(middleware[m]).forEach((p) => {
            re.test(p) && middleware[m][p].push([handler, paramCount]);
          });
        }
      });
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(routes[m]).forEach(
            (p) => re.test(p) && routes[m][p].push([handler, paramCount])
          );
        }
      });
      return;
    }
    const paths = checkOptionalParameter(path) || [path];
    for (let i = 0, len = paths.length; i < len; i++) {
      const path2 = paths[i];
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          routes[m][path2] ||= [
            ...findMiddleware(middleware[m], path2) || findMiddleware(middleware[METHOD_NAME_ALL], path2) || []
          ];
          routes[m][path2].push([handler, paramCount - len + i + 1]);
        }
      });
    }
  }
  match = match;
  buildAllMatchers() {
    const matchers = /* @__PURE__ */ Object.create(null);
    Object.keys(this.#routes).concat(Object.keys(this.#middleware)).forEach((method) => {
      matchers[method] ||= this.#buildMatcher(method);
    });
    this.#middleware = this.#routes = void 0;
    clearWildcardRegExpCache();
    return matchers;
  }
  #buildMatcher(method) {
    const routes = [];
    let hasOwnRoute = method === METHOD_NAME_ALL;
    [this.#middleware, this.#routes].forEach((r) => {
      const ownRoute = r[method] ? Object.keys(r[method]).map((path) => [path, r[method][path]]) : [];
      if (ownRoute.length !== 0) {
        hasOwnRoute ||= true;
        routes.push(...ownRoute);
      } else if (method !== METHOD_NAME_ALL) {
        routes.push(
          ...Object.keys(r[METHOD_NAME_ALL]).map((path) => [path, r[METHOD_NAME_ALL][path]])
        );
      }
    });
    if (!hasOwnRoute) {
      return null;
    } else {
      return buildMatcherFromPreprocessedRoutes(routes);
    }
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/smart-router/router.js
var SmartRouter = class {
  static {
    __name(this, "SmartRouter");
  }
  name = "SmartRouter";
  #routers = [];
  #routes = [];
  constructor(init) {
    this.#routers = init.routers;
  }
  add(method, path, handler) {
    if (!this.#routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    this.#routes.push([method, path, handler]);
  }
  match(method, path) {
    if (!this.#routes) {
      throw new Error("Fatal error");
    }
    const routers = this.#routers;
    const routes = this.#routes;
    const len = routers.length;
    let i = 0;
    let res;
    for (; i < len; i++) {
      const router = routers[i];
      try {
        for (let i2 = 0, len2 = routes.length; i2 < len2; i2++) {
          router.add(...routes[i2]);
        }
        res = router.match(method, path);
      } catch (e) {
        if (e instanceof UnsupportedPathError) {
          continue;
        }
        throw e;
      }
      this.match = router.match.bind(router);
      this.#routers = [router];
      this.#routes = void 0;
      break;
    }
    if (i === len) {
      throw new Error("Fatal error");
    }
    this.name = `SmartRouter + ${this.activeRouter.name}`;
    return res;
  }
  get activeRouter() {
    if (this.#routes || this.#routers.length !== 1) {
      throw new Error("No active router has been determined yet.");
    }
    return this.#routers[0];
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/trie-router/node.js
var emptyParams = /* @__PURE__ */ Object.create(null);
var Node2 = class _Node2 {
  static {
    __name(this, "_Node");
  }
  #methods;
  #children;
  #patterns;
  #order = 0;
  #params = emptyParams;
  constructor(method, handler, children) {
    this.#children = children || /* @__PURE__ */ Object.create(null);
    this.#methods = [];
    if (method && handler) {
      const m = /* @__PURE__ */ Object.create(null);
      m[method] = { handler, possibleKeys: [], score: 0 };
      this.#methods = [m];
    }
    this.#patterns = [];
  }
  insert(method, path, handler) {
    this.#order = ++this.#order;
    let curNode = this;
    const parts = splitRoutingPath(path);
    const possibleKeys = [];
    for (let i = 0, len = parts.length; i < len; i++) {
      const p = parts[i];
      const nextP = parts[i + 1];
      const pattern = getPattern(p, nextP);
      const key = Array.isArray(pattern) ? pattern[0] : p;
      if (key in curNode.#children) {
        curNode = curNode.#children[key];
        if (pattern) {
          possibleKeys.push(pattern[1]);
        }
        continue;
      }
      curNode.#children[key] = new _Node2();
      if (pattern) {
        curNode.#patterns.push(pattern);
        possibleKeys.push(pattern[1]);
      }
      curNode = curNode.#children[key];
    }
    curNode.#methods.push({
      [method]: {
        handler,
        possibleKeys: possibleKeys.filter((v, i, a) => a.indexOf(v) === i),
        score: this.#order
      }
    });
    return curNode;
  }
  #getHandlerSets(node, method, nodeParams, params) {
    const handlerSets = [];
    for (let i = 0, len = node.#methods.length; i < len; i++) {
      const m = node.#methods[i];
      const handlerSet = m[method] || m[METHOD_NAME_ALL];
      const processedSet = {};
      if (handlerSet !== void 0) {
        handlerSet.params = /* @__PURE__ */ Object.create(null);
        handlerSets.push(handlerSet);
        if (nodeParams !== emptyParams || params && params !== emptyParams) {
          for (let i2 = 0, len2 = handlerSet.possibleKeys.length; i2 < len2; i2++) {
            const key = handlerSet.possibleKeys[i2];
            const processed = processedSet[handlerSet.score];
            handlerSet.params[key] = params?.[key] && !processed ? params[key] : nodeParams[key] ?? params?.[key];
            processedSet[handlerSet.score] = true;
          }
        }
      }
    }
    return handlerSets;
  }
  search(method, path) {
    const handlerSets = [];
    this.#params = emptyParams;
    const curNode = this;
    let curNodes = [curNode];
    const parts = splitPath(path);
    const curNodesQueue = [];
    for (let i = 0, len = parts.length; i < len; i++) {
      const part = parts[i];
      const isLast = i === len - 1;
      const tempNodes = [];
      for (let j = 0, len2 = curNodes.length; j < len2; j++) {
        const node = curNodes[j];
        const nextNode = node.#children[part];
        if (nextNode) {
          nextNode.#params = node.#params;
          if (isLast) {
            if (nextNode.#children["*"]) {
              handlerSets.push(
                ...this.#getHandlerSets(nextNode.#children["*"], method, node.#params)
              );
            }
            handlerSets.push(...this.#getHandlerSets(nextNode, method, node.#params));
          } else {
            tempNodes.push(nextNode);
          }
        }
        for (let k = 0, len3 = node.#patterns.length; k < len3; k++) {
          const pattern = node.#patterns[k];
          const params = node.#params === emptyParams ? {} : { ...node.#params };
          if (pattern === "*") {
            const astNode = node.#children["*"];
            if (astNode) {
              handlerSets.push(...this.#getHandlerSets(astNode, method, node.#params));
              astNode.#params = params;
              tempNodes.push(astNode);
            }
            continue;
          }
          const [key, name, matcher] = pattern;
          if (!part && !(matcher instanceof RegExp)) {
            continue;
          }
          const child = node.#children[key];
          const restPathString = parts.slice(i).join("/");
          if (matcher instanceof RegExp) {
            const m = matcher.exec(restPathString);
            if (m) {
              params[name] = m[0];
              handlerSets.push(...this.#getHandlerSets(child, method, node.#params, params));
              if (Object.keys(child.#children).length) {
                child.#params = params;
                const componentCount = m[0].match(/\//)?.length ?? 0;
                const targetCurNodes = curNodesQueue[componentCount] ||= [];
                targetCurNodes.push(child);
              }
              continue;
            }
          }
          if (matcher === true || matcher.test(part)) {
            params[name] = part;
            if (isLast) {
              handlerSets.push(...this.#getHandlerSets(child, method, params, node.#params));
              if (child.#children["*"]) {
                handlerSets.push(
                  ...this.#getHandlerSets(child.#children["*"], method, params, node.#params)
                );
              }
            } else {
              child.#params = params;
              tempNodes.push(child);
            }
          }
        }
      }
      curNodes = tempNodes.concat(curNodesQueue.shift() ?? []);
    }
    if (handlerSets.length > 1) {
      handlerSets.sort((a, b) => {
        return a.score - b.score;
      });
    }
    return [handlerSets.map(({ handler, params }) => [handler, params])];
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/router/trie-router/router.js
var TrieRouter = class {
  static {
    __name(this, "TrieRouter");
  }
  name = "TrieRouter";
  #node;
  constructor() {
    this.#node = new Node2();
  }
  add(method, path, handler) {
    const results = checkOptionalParameter(path);
    if (results) {
      for (let i = 0, len = results.length; i < len; i++) {
        this.#node.insert(method, results[i], handler);
      }
      return;
    }
    this.#node.insert(method, path, handler);
  }
  match(method, path) {
    return this.#node.search(method, path);
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/hono.js
var Hono2 = class extends Hono {
  static {
    __name(this, "Hono");
  }
  /**
   * Creates an instance of the Hono class.
   *
   * @param options - Optional configuration options for the Hono instance.
   */
  constructor(options = {}) {
    super(options);
    this.router = options.router ?? new SmartRouter({
      routers: [new RegExpRouter(), new TrieRouter()]
    });
  }
};

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/middleware/cors/index.js
var cors = /* @__PURE__ */ __name((options) => {
  const defaults = {
    origin: "*",
    allowMethods: ["GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"],
    allowHeaders: [],
    exposeHeaders: []
  };
  const opts = {
    ...defaults,
    ...options
  };
  const findAllowOrigin = ((optsOrigin) => {
    if (typeof optsOrigin === "string") {
      if (optsOrigin === "*") {
        return () => optsOrigin;
      } else {
        return (origin) => optsOrigin === origin ? origin : null;
      }
    } else if (typeof optsOrigin === "function") {
      return optsOrigin;
    } else {
      return (origin) => optsOrigin.includes(origin) ? origin : null;
    }
  })(opts.origin);
  const findAllowMethods = ((optsAllowMethods) => {
    if (typeof optsAllowMethods === "function") {
      return optsAllowMethods;
    } else if (Array.isArray(optsAllowMethods)) {
      return () => optsAllowMethods;
    } else {
      return () => [];
    }
  })(opts.allowMethods);
  return /* @__PURE__ */ __name(async function cors2(c, next) {
    function set(key, value) {
      c.res.headers.set(key, value);
    }
    __name(set, "set");
    const allowOrigin = await findAllowOrigin(c.req.header("origin") || "", c);
    if (allowOrigin) {
      set("Access-Control-Allow-Origin", allowOrigin);
    }
    if (opts.credentials) {
      set("Access-Control-Allow-Credentials", "true");
    }
    if (opts.exposeHeaders?.length) {
      set("Access-Control-Expose-Headers", opts.exposeHeaders.join(","));
    }
    if (c.req.method === "OPTIONS") {
      if (opts.origin !== "*") {
        set("Vary", "Origin");
      }
      if (opts.maxAge != null) {
        set("Access-Control-Max-Age", opts.maxAge.toString());
      }
      const allowMethods = await findAllowMethods(c.req.header("origin") || "", c);
      if (allowMethods.length) {
        set("Access-Control-Allow-Methods", allowMethods.join(","));
      }
      let headers = opts.allowHeaders;
      if (!headers?.length) {
        const requestHeaders = c.req.header("Access-Control-Request-Headers");
        if (requestHeaders) {
          headers = requestHeaders.split(/\s*,\s*/);
        }
      }
      if (headers?.length) {
        set("Access-Control-Allow-Headers", headers.join(","));
        c.res.headers.append("Vary", "Access-Control-Request-Headers");
      }
      c.res.headers.delete("Content-Length");
      c.res.headers.delete("Content-Type");
      return new Response(null, {
        headers: c.res.headers,
        status: 204,
        statusText: "No Content"
      });
    }
    await next();
    if (opts.origin !== "*") {
      c.header("Vary", "Origin", { append: true });
    }
  }, "cors2");
}, "cors");

// node_modules/.pnpm/hono@4.11.7/node_modules/hono/dist/middleware/timing/timing.js
var getTime = /* @__PURE__ */ __name(() => {
  try {
    return performance.now();
  } catch {
  }
  return Date.now();
}, "getTime");
var timing = /* @__PURE__ */ __name((config2) => {
  const options = {
    total: true,
    enabled: true,
    totalDescription: "Total Response Time",
    autoEnd: true,
    crossOrigin: false,
    ...config2
  };
  return /* @__PURE__ */ __name(async function timing2(c, next) {
    const headers = [];
    const timers = /* @__PURE__ */ new Map();
    if (c.get("metric")) {
      return await next();
    }
    c.set("metric", { headers, timers });
    if (options.total) {
      startTime(c, "total", options.totalDescription);
    }
    await next();
    if (options.total) {
      endTime(c, "total");
    }
    if (options.autoEnd) {
      timers.forEach((_, key) => endTime(c, key));
    }
    const enabled = typeof options.enabled === "function" ? options.enabled(c) : options.enabled;
    if (enabled) {
      c.res.headers.append("Server-Timing", headers.join(","));
      const crossOrigin = typeof options.crossOrigin === "function" ? options.crossOrigin(c) : options.crossOrigin;
      if (crossOrigin) {
        c.res.headers.append(
          "Timing-Allow-Origin",
          typeof crossOrigin === "string" ? crossOrigin : "*"
        );
      }
    }
  }, "timing2");
}, "timing");
var setMetric = /* @__PURE__ */ __name((c, name, valueDescription, description, precision) => {
  const metrics = c.get("metric");
  if (!metrics) {
    console.warn("Metrics not initialized! Please add the `timing()` middleware to this route!");
    return;
  }
  if (typeof valueDescription === "number") {
    const dur = valueDescription.toFixed(precision || 1);
    const metric = description ? `${name};dur=${dur};desc="${description}"` : `${name};dur=${dur}`;
    metrics.headers.push(metric);
  } else {
    const metric = valueDescription ? `${name};desc="${valueDescription}"` : `${name}`;
    metrics.headers.push(metric);
  }
}, "setMetric");
var startTime = /* @__PURE__ */ __name((c, name, description) => {
  const metrics = c.get("metric");
  if (!metrics) {
    console.warn("Metrics not initialized! Please add the `timing()` middleware to this route!");
    return;
  }
  metrics.timers.set(name, { description, start: getTime() });
}, "startTime");
var endTime = /* @__PURE__ */ __name((c, name, precision) => {
  const metrics = c.get("metric");
  if (!metrics) {
    console.warn("Metrics not initialized! Please add the `timing()` middleware to this route!");
    return;
  }
  const timer = metrics.timers.get(name);
  if (!timer) {
    console.warn(`Timer "${name}" does not exist!`);
    return;
  }
  const { description, start } = timer;
  const duration = getTime() - start;
  setMetric(c, name, duration, description, precision);
  metrics.timers.delete(name);
}, "endTime");

// src/routes/health.ts
var app = new Hono2();
app.get("/", (c) => {
  return c.json({ status: "ok" });
});
var health_default = app;

// src/engines/engine.ts
function newEngineResults() {
  return {
    results: [],
    suggestions: [],
    corrections: [],
    engineData: {}
  };
}
__name(newEngineResults, "newEngineResults");
async function executeEngine(engine, query, params) {
  const config2 = engine.buildRequest(query, params);
  const headers = new Headers(config2.headers);
  if (config2.cookies.length > 0) {
    headers.set("Cookie", config2.cookies.join("; "));
  }
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), engine.timeout);
  try {
    const response = await fetch(config2.url, {
      method: config2.method,
      headers,
      body: config2.body || void 0,
      signal: controller.signal,
      redirect: "follow"
    });
    if (!response.ok) {
      throw new Error(
        `${engine.name}: HTTP ${response.status} ${response.statusText}`
      );
    }
    const body = await response.text();
    const results = engine.parseResponse(body, params);
    for (const r of results.results) {
      r.engine = engine.name;
      if (r.score === 0) {
        r.score = engine.weight;
      }
    }
    return results;
  } finally {
    clearTimeout(timeoutId);
  }
}
__name(executeEngine, "executeEngine");

// src/lib/html-parser.ts
function decodeHtmlEntities(text) {
  let result = text;
  result = result.replace(/&amp;/g, "&");
  result = result.replace(/&lt;/g, "<");
  result = result.replace(/&gt;/g, ">");
  result = result.replace(/&quot;/g, '"');
  result = result.replace(/&#39;/g, "'");
  result = result.replace(/&apos;/g, "'");
  result = result.replace(/&nbsp;/g, " ");
  result = result.replace(/&mdash;/g, "\u2014");
  result = result.replace(/&ndash;/g, "\u2013");
  result = result.replace(/&laquo;/g, "\xAB");
  result = result.replace(/&raquo;/g, "\xBB");
  result = result.replace(/&hellip;/g, "\u2026");
  result = result.replace(/&copy;/g, "\xA9");
  result = result.replace(/&reg;/g, "\xAE");
  result = result.replace(/&trade;/g, "\u2122");
  result = result.replace(/&#(\d+);/g, (_match, code) => {
    const num = parseInt(code, 10);
    return num > 0 && num < 1114111 ? String.fromCodePoint(num) : "";
  });
  result = result.replace(/&#x([0-9a-fA-F]+);/g, (_match, code) => {
    const num = parseInt(code, 16);
    return num > 0 && num < 1114111 ? String.fromCodePoint(num) : "";
  });
  return result;
}
__name(decodeHtmlEntities, "decodeHtmlEntities");
function extractText(html) {
  let result = html.replace(/<script[^>]*>[\s\S]*?<\/script>/gi, "");
  result = result.replace(/<style[^>]*>[\s\S]*?<\/style>/gi, "");
  result = result.replace(/<br\s*\/?>/gi, " ");
  result = result.replace(/<\/?(p|div|li|h[1-6]|tr|td|th)\b[^>]*>/gi, " ");
  result = result.replace(/<[^>]+>/g, "");
  result = decodeHtmlEntities(result);
  result = result.replace(/\s+/g, " ").trim();
  return result;
}
__name(extractText, "extractText");
function findElements(html, selector) {
  let tag = "";
  let className = "";
  let idName = "";
  let attrName = "";
  let attrValue = "";
  const attrMatch = selector.match(
    /^(\w+)?\[([a-zA-Z0-9_-]+)\s*=\s*['"]([^'"]*)['"]\]$/
  );
  if (attrMatch) {
    tag = attrMatch[1] || "";
    attrName = attrMatch[2];
    attrValue = attrMatch[3];
  } else if (selector.includes("#")) {
    const parts = selector.split("#");
    tag = parts[0] || "";
    idName = parts[1];
  } else if (selector.includes(".")) {
    const parts = selector.split(".");
    tag = parts[0] || "";
    className = parts[1];
  } else {
    tag = selector;
  }
  const results = [];
  const tagToSearch = tag || "[a-zA-Z][a-zA-Z0-9]*";
  let openPattern;
  if (attrName) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\b${attrName}\\s*=\\s*["']${escapeRegex(attrValue)}["'][^>]*>`,
      "gi"
    );
  } else if (idName) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\bid\\s*=\\s*["']${escapeRegex(idName)}["'][^>]*>`,
      "gi"
    );
  } else if (className) {
    openPattern = new RegExp(
      `<(${tagToSearch})\\b[^>]*?\\bclass\\s*=\\s*["'][^"']*\\b${escapeRegex(className)}\\b[^"']*["'][^>]*>`,
      "gi"
    );
  } else {
    openPattern = new RegExp(`<(${tagToSearch})\\b[^>]*>`, "gi");
  }
  let openMatch;
  while ((openMatch = openPattern.exec(html)) !== null) {
    const startIndex = openMatch.index;
    const matchedTag = openMatch[1].toLowerCase();
    if (openMatch[0].endsWith("/>")) {
      results.push(openMatch[0]);
      continue;
    }
    let depth = 1;
    let searchPos = startIndex + openMatch[0].length;
    const closeTagPattern = new RegExp(
      `<(/?)${matchedTag}\\b[^>]*>`,
      "gi"
    );
    closeTagPattern.lastIndex = searchPos;
    let closeMatch;
    while (depth > 0 && (closeMatch = closeTagPattern.exec(html)) !== null) {
      if (closeMatch[1] === "/") {
        depth--;
      } else if (!closeMatch[0].endsWith("/>")) {
        depth++;
      }
      if (depth === 0) {
        const endIndex = closeMatch.index + closeMatch[0].length;
        results.push(html.slice(startIndex, endIndex));
      }
    }
    if (depth > 0) {
      const maxChunk = Math.min(startIndex + 1e4, html.length);
      results.push(html.slice(startIndex, maxChunk));
    }
  }
  return results;
}
__name(findElements, "findElements");
function escapeRegex(str) {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
__name(escapeRegex, "escapeRegex");

// src/engines/google.ts
var gsaUserAgents = [
  "Mozilla/5.0 (iPhone; CPU iPhone OS 17_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_3_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
  "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/399.2.845414227 Mobile/15E148 Safari/604.1",
  "Mozilla/5.0 (Linux; Android 14; SM-S928B Build/UP1A.231005.007) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.230 Mobile Safari/537.36 GSA/15.3.36.28.arm64",
  "Mozilla/5.0 (Linux; Android 13; Pixel 7 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.6099.144 Mobile Safari/537.36 GSA/14.50.15.29.arm64"
];
var ARC_ID_RANGE = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-";
var cachedArcId = "";
var arcIdTimestamp = 0;
function generateArcId() {
  const bytes = new Uint8Array(23);
  crypto.getRandomValues(bytes);
  let result = "";
  for (let i = 0; i < 23; i++) {
    result += ARC_ID_RANGE[bytes[i] % ARC_ID_RANGE.length];
  }
  return result;
}
__name(generateArcId, "generateArcId");
function getArcId() {
  const now = Date.now();
  if (!cachedArcId || now - arcIdTimestamp > 36e5) {
    cachedArcId = generateArcId();
    arcIdTimestamp = now;
  }
  return cachedArcId;
}
__name(getArcId, "getArcId");
function uiAsync(start) {
  const startPadded = start.toString().padStart(2, "0");
  const arcId = `arc_id:srp_${getArcId()}_1${startPadded}`;
  return `${arcId},use_ac:true,_fmt:prog`;
}
__name(uiAsync, "uiAsync");
function getRandomGSAUserAgent() {
  const idx = Math.floor(Math.random() * gsaUserAgents.length);
  return gsaUserAgents[idx];
}
__name(getRandomGSAUserAgent, "getRandomGSAUserAgent");
var timeRangeMap = {
  day: "d",
  week: "w",
  month: "m",
  year: "y"
};
var safeSearchMap = {
  0: "off",
  1: "medium",
  2: "high"
};
function unwrapGoogleUrl(href) {
  if (href.startsWith("/url?")) {
    const match2 = href.match(/[?&]q=([^&]+)/);
    if (match2) {
      let decoded = decodeURIComponent(match2[1]);
      const saIdx = decoded.indexOf("&sa=U");
      if (saIdx > 0) {
        decoded = decoded.slice(0, saIdx);
      }
      return decoded;
    }
    const urlMatch = href.match(/[?&]url=([^&]+)/);
    if (urlMatch) {
      return decodeURIComponent(urlMatch[1]);
    }
  }
  return href;
}
__name(unwrapGoogleUrl, "unwrapGoogleUrl");
var GoogleEngine = class {
  static {
    __name(this, "GoogleEngine");
  }
  name = "google";
  shortcut = "g";
  categories = ["general"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const start = (params.page - 1) * 10;
    const asyncParam = uiAsync(start);
    const locale = params.locale || "en-US";
    const parts = locale.split("-");
    const langCode = parts[0] || "en";
    const regionCode = parts[1] || "US";
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("hl", `${langCode}-${regionCode}`);
    if (locale !== "all") {
      searchParams.set("lr", `lang_${langCode}`);
    }
    if (locale.includes("-")) {
      searchParams.set("cr", `country${regionCode}`);
    }
    searchParams.set("ie", "utf8");
    searchParams.set("oe", "utf8");
    searchParams.set("filter", "0");
    searchParams.set("start", start.toString());
    searchParams.set("asearch", "arc");
    searchParams.set("async", asyncParam);
    if (params.timeRange && timeRangeMap[params.timeRange]) {
      searchParams.set("tbs", `qdr:${timeRangeMap[params.timeRange]}`);
    }
    const safeValue = safeSearchMap[params.safeSearch];
    if (safeValue) {
      searchParams.set("safe", safeValue);
    }
    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": getRandomGSAUserAgent(),
        Accept: "*/*"
      },
      cookies: ["CONSENT=YES+"]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    if (body.includes("sorry.google.com") || body.includes("/sorry/")) {
      return results;
    }
    const mjjYudElements = findElements(body, "div.MjjYud");
    for (const el of mjjYudElements) {
      const result = this.parseMjjYudResult(el);
      if (result) {
        results.results.push(result);
      }
    }
    if (results.results.length === 0) {
      const gElements = findElements(body, "div.g");
      for (const el of gElements) {
        if (el.includes("g-blk")) continue;
        const result = this.parseGResult(el);
        if (result) {
          results.results.push(result);
        }
      }
    }
    const suggestionElements = findElements(body, "div.ouy7Mc");
    for (const el of suggestionElements) {
      const linkPattern = /<a\b[^>]*>([^<]*(?:<[^/][^>]*>[^<]*)*)<\/a>/gi;
      let match2;
      while ((match2 = linkPattern.exec(el)) !== null) {
        const text = extractText(match2[1]).trim();
        if (text) {
          results.suggestions.push(text);
        }
      }
    }
    return results;
  }
  parseMjjYudResult(html) {
    let title2 = "";
    const roleLinkElements = findElements(html, "div[role='link']");
    if (roleLinkElements.length > 0) {
      title2 = extractText(roleLinkElements[0]).trim();
    }
    let url = "";
    const hrefMatch = html.match(
      /<a\b[^>]*?\bhref\s*=\s*"([^"]+)"/i
    );
    if (hrefMatch) {
      const href = decodeHtmlEntities(hrefMatch[1]);
      if (!href.startsWith("#")) {
        if (href.startsWith("/url?")) {
          url = unwrapGoogleUrl(href);
        } else if (href.startsWith("http://") || href.startsWith("https://")) {
          url = href;
        }
      }
    }
    if (!url || !title2) return null;
    if (url.includes("google.com") && !url.includes("translate.google")) {
      return null;
    }
    let content = "";
    const sncfElements = findElements(html, "div[data-sncf='1']");
    if (sncfElements.length > 0) {
      let cleaned = sncfElements[0].replace(
        /<script[^>]*>[\s\S]*?<\/script>/gi,
        ""
      );
      content = extractText(cleaned).trim();
    }
    return {
      url,
      title: title2,
      content,
      engine: this.name,
      score: this.weight,
      category: "general"
    };
  }
  parseGResult(html) {
    let url = "";
    let title2 = "";
    const linkMatch = html.match(
      /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+|\/url\?[^"]+)"/i
    );
    if (linkMatch) {
      const href = decodeHtmlEntities(linkMatch[1]);
      if (href.startsWith("/url?")) {
        url = unwrapGoogleUrl(href);
      } else {
        url = href;
      }
    }
    if (!url) return null;
    if (url.includes("google.com/search") || url.includes("google.com") && !url.includes("translate.google")) {
      return null;
    }
    const h3Match = html.match(/<h3[^>]*>([\s\S]*?)<\/h3>/i);
    if (h3Match) {
      title2 = extractText(h3Match[1]).trim();
    }
    if (!title2) {
      const aContent = html.match(/<a\b[^>]*>([\s\S]*?)<\/a>/i);
      if (aContent) {
        title2 = extractText(aContent[1]).trim();
      }
    }
    if (!title2) return null;
    let content = "";
    const snippetPatterns = [
      /class="[^"]*VwiC3b[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /class="[^"]*IsZvec[^"]*"[^>]*>([\s\S]*?)<\/(?:div|span)>/i,
      /data-sncf="1"[^>]*>([\s\S]*?)<\/div>/i
    ];
    for (const pattern of snippetPatterns) {
      const match2 = html.match(pattern);
      if (match2) {
        const text = extractText(match2[1]).trim();
        if (text.length > content.length) {
          content = text;
        }
      }
    }
    return {
      url,
      title: title2,
      content,
      engine: this.name,
      score: this.weight,
      category: "general"
    };
  }
};
var GoogleImagesEngine = class {
  static {
    __name(this, "GoogleImagesEngine");
  }
  name = "google images";
  shortcut = "gi";
  categories = ["images"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("tbm", "isch");
    searchParams.set("asearch", "isch");
    searchParams.set("hl", "en");
    searchParams.set("safe", "off");
    const ijn = params.page - 1;
    searchParams.set("async", `_fmt:json,p:1,ijn:${ijn}`);
    return {
      url: `https://www.google.com/search?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": getRandomGSAUserAgent(),
        Accept: "*/*"
      },
      cookies: ["CONSENT=YES+"]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const jsonStart = body.indexOf('{"ischj"');
    if (jsonStart !== -1) {
      let jsonEnd = body.indexOf("\n", jsonStart);
      if (jsonEnd === -1) jsonEnd = body.length;
      const jsonStr = body.slice(jsonStart, jsonEnd);
      try {
        const data = JSON.parse(jsonStr);
        if (data.ischj?.metadata) {
          for (const item of data.ischj.metadata) {
            if (item.original_image?.url) {
              results.results.push({
                url: item.result?.referrer_url || "",
                title: item.result?.page_title || "",
                content: "",
                engine: this.name,
                score: this.weight,
                category: "images",
                template: "images",
                imageUrl: item.original_image.url,
                thumbnailUrl: item.thumbnail?.url || "",
                source: item.result?.site_title || "",
                resolution: `${item.original_image.width || 0}x${item.original_image.height || 0}`
              });
            }
          }
        }
      } catch {
      }
    }
    if (results.results.length === 0) {
      const re = /\["(https:\/\/[^"]+\.(?:jpg|jpeg|png|gif|webp)[^"]*)",(\d+),(\d+)\]/g;
      let match2;
      while ((match2 = re.exec(body)) !== null) {
        results.results.push({
          url: match2[1],
          title: "",
          content: "",
          engine: this.name,
          score: this.weight,
          category: "images",
          template: "images",
          imageUrl: match2[1],
          resolution: `${match2[2]}x${match2[3]}`
        });
      }
    }
    return results;
  }
};

// src/engines/bing.ts
var BING_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36";
function decodeBingUrl(bingUrl) {
  try {
    const parsed = new URL(bingUrl);
    const paramU = parsed.searchParams.get("u");
    if (!paramU) return bingUrl;
    if (paramU.length > 2 && paramU.startsWith("a1")) {
      let encoded = paramU.slice(2);
      encoded = encoded.replace(/-/g, "+").replace(/_/g, "/");
      const padding = 4 - encoded.length % 4;
      if (padding < 4) {
        encoded += "=".repeat(padding);
      }
      try {
        const decoded = atob(encoded);
        return decoded;
      } catch {
        return bingUrl;
      }
    }
  } catch {
  }
  return bingUrl;
}
__name(decodeBingUrl, "decodeBingUrl");
var generalTimeRange = {
  day: "1",
  week: "2",
  month: "3",
  year: "5"
};
var imagesTimeRange = {
  day: 1440,
  week: 10080,
  month: 44640,
  year: 525600
};
var newsTimeRange = {
  day: "4",
  week: "7",
  month: "9"
};
var BingEngine = class {
  static {
    __name(this, "BingEngine");
  }
  name = "bing";
  shortcut = "b";
  categories = ["general"];
  supportsPaging = true;
  maxPage = 200;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const locale = params.locale || "en-US";
    const parts = locale.split("-");
    const lang = (parts[0] || "en").toLowerCase();
    const region = `${lang}-${(parts[1] || "us").toLowerCase()}`;
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("pq", query);
    if (params.page > 1) {
      const first = (params.page - 1) * 10 + 1;
      searchParams.set("first", first.toString());
      if (params.page === 2) {
        searchParams.set("FORM", "PERE");
      } else {
        searchParams.set("FORM", `PERE${params.page - 2}`);
      }
    }
    if (params.timeRange && generalTimeRange[params.timeRange]) {
      const tr = generalTimeRange[params.timeRange];
      if (params.timeRange === "year") {
        const unixDay = Math.floor(Date.now() / 864e5);
        searchParams.set(
          "filters",
          `ex1:"ez${tr}_${unixDay - 365}_${unixDay}"`
        );
      } else {
        searchParams.set("filters", `ex1:"ez${tr}"`);
      }
    }
    const cookies = [
      `_EDGE_CD=m=${region}&u=${lang}`,
      `_EDGE_S=mkt=${region}&ui=${lang}`,
      `SRCHHPGUSR=SRCHLANG=${lang}`
    ];
    return {
      url: `https://www.bing.com/search?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": BING_USER_AGENT,
        Accept: "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
        DNT: "1",
        "Upgrade-Insecure-Requests": "1"
      },
      cookies
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const algoItems = findElements(body, "li.b_algo");
    for (const item of algoItems) {
      const result = this.parseResult(item);
      if (result) {
        results.results.push(result);
      }
    }
    return results;
  }
  parseResult(html) {
    let url = "";
    let title2 = "";
    let content = "";
    const h2Match = html.match(/<h2[^>]*>([\s\S]*?)<\/h2>/i);
    if (h2Match) {
      const h2Content = h2Match[1];
      const linkMatch = h2Content.match(
        /<a\b[^>]*?\bhref\s*=\s*"([^"]+)"[^>]*>([\s\S]*?)<\/a>/i
      );
      if (linkMatch) {
        const href = decodeHtmlEntities(linkMatch[1]);
        if (href.startsWith("https://www.bing.com/ck/a?")) {
          url = decodeBingUrl(href);
        } else if (href.startsWith("http")) {
          url = href;
        }
        title2 = extractText(linkMatch[2]).trim();
      }
    }
    if (!url || !title2) return null;
    const pMatch = html.match(/<p[^>]*>([\s\S]*?)<\/p>/i);
    if (pMatch) {
      content = extractText(pMatch[1]).trim();
    }
    if (!content) {
      const captionElements = findElements(html, "div.b_caption");
      if (captionElements.length > 0) {
        content = extractText(captionElements[0]).trim();
      }
    }
    return {
      url,
      title: title2,
      content,
      engine: this.name,
      score: this.weight,
      category: "general"
    };
  }
};
var BingImagesEngine = class {
  static {
    __name(this, "BingImagesEngine");
  }
  name = "bing images";
  shortcut = "bi";
  categories = ["images"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("async", "1");
    searchParams.set("count", "35");
    let first = 1;
    if (params.page > 1) {
      first = (params.page - 1) * 35 + 1;
    }
    searchParams.set("first", first.toString());
    if (params.timeRange && imagesTimeRange[params.timeRange]) {
      searchParams.set(
        "qft",
        `filterui:age-lt${imagesTimeRange[params.timeRange]}`
      );
    }
    return {
      url: `https://www.bing.com/images/async?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": BING_USER_AGENT,
        Accept: "text/html"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const iuscPattern = /class="iusc"[^>]*m="([^"]+)"/g;
    let match2;
    while ((match2 = iuscPattern.exec(body)) !== null) {
      let jsonStr = match2[1].replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/&lt;/g, "<").replace(/&gt;/g, ">");
      try {
        const metadata = JSON.parse(jsonStr);
        if (metadata.murl) {
          results.results.push({
            url: metadata.purl || "",
            title: metadata.t || "",
            content: metadata.desc || "",
            engine: this.name,
            score: this.weight,
            category: "images",
            template: "images",
            imageUrl: metadata.murl,
            thumbnailUrl: metadata.turl || ""
          });
        }
      } catch {
      }
    }
    if (results.results.length === 0) {
      const fallbackPattern = /murl&quot;:&quot;([^&]+)&quot;/g;
      let fbMatch;
      while ((fbMatch = fallbackPattern.exec(body)) !== null) {
        const imgUrl = decodeURIComponent(fbMatch[1]);
        if (imgUrl) {
          results.results.push({
            url: imgUrl,
            title: "",
            content: "",
            engine: this.name,
            score: this.weight,
            category: "images",
            template: "images",
            imageUrl: imgUrl
          });
        }
      }
    }
    return results;
  }
};
var BingNewsEngine = class {
  static {
    __name(this, "BingNewsEngine");
  }
  name = "bing news";
  shortcut = "bn";
  categories = ["news"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const locale = params.locale || "en-US";
    const parts = locale.split("-");
    const lang = (parts[0] || "en").toLowerCase();
    const country = (parts[1] || "us").toLowerCase();
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("InfiniteScroll", "1");
    searchParams.set("form", "PTFTNR");
    searchParams.set("setlang", lang);
    searchParams.set("cc", country);
    let first = 1;
    let sfx = 0;
    if (params.page > 1) {
      first = (params.page - 1) * 10 + 1;
      sfx = params.page - 1;
    }
    searchParams.set("first", first.toString());
    searchParams.set("SFX", sfx.toString());
    if (params.timeRange && newsTimeRange[params.timeRange]) {
      searchParams.set(
        "qft",
        `interval="${newsTimeRange[params.timeRange]}"`
      );
    }
    return {
      url: `https://www.bing.com/news/infinitescrollajax?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": BING_USER_AGENT,
        Accept: "text/html"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const newsItems = findElements(body, "div.newsitem");
    const newsCards = findElements(body, "div.news-card");
    const allItems = [...newsItems, ...newsCards];
    for (const item of allItems) {
      const result = this.parseNewsResult(item);
      if (result) {
        results.results.push(result);
      }
    }
    return results;
  }
  parseNewsResult(html) {
    let url = "";
    let title2 = "";
    let content = "";
    let source = "";
    let thumbnailUrl = "";
    const linkPattern = /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let linkMatch;
    while ((linkMatch = linkPattern.exec(html)) !== null) {
      const linkText = extractText(linkMatch[2]).trim();
      if (linkText && !url) {
        url = decodeHtmlEntities(linkMatch[1]);
        title2 = linkText;
        break;
      }
    }
    if (!url) return null;
    const snippetElements = findElements(html, "div.snippet");
    const summaryElements = findElements(html, "div.summary");
    const snippetHtml = snippetElements[0] || summaryElements[0] || "";
    if (snippetHtml) {
      content = extractText(snippetHtml).trim();
    }
    const imgMatch = html.match(
      /<img\b[^>]*?\bsrc\s*=\s*"([^"]+)"/i
    );
    if (imgMatch && !imgMatch[1].startsWith("data:image")) {
      thumbnailUrl = imgMatch[1];
      if (!thumbnailUrl.startsWith("http")) {
        thumbnailUrl = "https://www.bing.com" + thumbnailUrl;
      }
    }
    const sourceElements = findElements(html, "div.source");
    if (sourceElements.length > 0) {
      source = extractText(sourceElements[0]).trim();
    }
    return {
      url,
      title: title2,
      content,
      engine: this.name,
      score: this.weight,
      category: "news",
      template: "news",
      thumbnailUrl,
      source
    };
  }
};

// src/engines/duckduckgo.ts
var DDG_USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36";
var ddgRegions = {
  "en-US": "us-en",
  "en-GB": "uk-en",
  "de-DE": "de-de",
  "fr-FR": "fr-fr",
  "es-ES": "es-es",
  "it-IT": "it-it",
  "ja-JP": "jp-jp",
  "ko-KR": "kr-kr",
  "zh-CN": "cn-zh",
  "ru-RU": "ru-ru"
};
function getDdgRegion(locale) {
  return ddgRegions[locale] || "wt-wt";
}
__name(getDdgRegion, "getDdgRegion");
var secFetchHeaders = {
  "Sec-Fetch-Dest": "document",
  "Sec-Fetch-Mode": "navigate",
  "Sec-Fetch-Site": "same-origin",
  "Sec-Fetch-User": "?1",
  "Upgrade-Insecure-Requests": "1"
};
var DuckDuckGoImagesEngine = class {
  static {
    __name(this, "DuckDuckGoImagesEngine");
  }
  name = "duckduckgo images";
  shortcut = "ddi";
  categories = ["images"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  /**
   * NOTE: This engine requires an async VQD fetch before building the request.
   * The caller should pre-fetch VQD and pass it in params.engineData['vqd'].
   * If not provided, executeEngine will fail. Use the companion helper
   * `prepareVqd(query, locale)` before calling executeEngine.
   */
  buildRequest(query, params) {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData["vqd"] || "";
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("o", "json");
    searchParams.set("l", region);
    searchParams.set("f", ",,,,,");
    searchParams.set("vqd", vqd);
    if (params.page > 1) {
      searchParams.set("s", ((params.page - 1) * 100).toString());
    }
    if (params.safeSearch === 0) {
      searchParams.set("p", "-1");
    } else if (params.safeSearch === 2) {
      searchParams.set("p", "1");
    }
    return {
      url: `https://duckduckgo.com/i.js?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": DDG_USER_AGENT,
        Accept: "application/json, text/javascript, */*; q=0.01",
        Referer: "https://duckduckgo.com/",
        "X-Requested-With": "XMLHttpRequest",
        ...secFetchHeaders
      },
      cookies: [`l=${region}`, `ah=${region}`]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const jsonStart = body.indexOf("{");
    if (jsonStart === -1) return results;
    try {
      const data = JSON.parse(body.slice(jsonStart));
      if (data.results) {
        for (const item of data.results) {
          if (item.image) {
            results.results.push({
              url: item.url || "",
              title: item.title || "",
              content: "",
              engine: this.name,
              score: this.weight,
              category: "images",
              template: "images",
              imageUrl: item.image,
              thumbnailUrl: item.thumbnail || "",
              source: item.source || "",
              resolution: `${item.width || 0}x${item.height || 0}`
            });
          }
        }
      }
    } catch {
    }
    return results;
  }
};
var DuckDuckGoVideosEngine = class {
  static {
    __name(this, "DuckDuckGoVideosEngine");
  }
  name = "duckduckgo videos";
  shortcut = "ddv";
  categories = ["videos"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData["vqd"] || "";
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("o", "json");
    searchParams.set("l", region);
    searchParams.set("f", ",,,,,");
    searchParams.set("vqd", vqd);
    if (params.page > 1) {
      searchParams.set("s", ((params.page - 1) * 60).toString());
    }
    if (params.safeSearch === 0) {
      searchParams.set("p", "-1");
    } else if (params.safeSearch === 2) {
      searchParams.set("p", "1");
    }
    return {
      url: `https://duckduckgo.com/v.js?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": DDG_USER_AGENT,
        Accept: "application/json, text/javascript, */*; q=0.01",
        Referer: "https://duckduckgo.com/",
        "X-Requested-With": "XMLHttpRequest",
        ...secFetchHeaders
      },
      cookies: [`l=${region}`]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const jsonStart = body.indexOf("{");
    if (jsonStart === -1) return results;
    try {
      const data = JSON.parse(body.slice(jsonStart));
      if (data.results) {
        for (const item of data.results) {
          let thumbnail = item.images?.medium || item.images?.small || item.images?.large || "";
          let content = item.description || "";
          if (item.uploader && content) {
            content = `by ${item.uploader} - ${content}`;
          } else if (item.uploader) {
            content = `by ${item.uploader}`;
          }
          results.results.push({
            url: item.content || "",
            title: item.title || "",
            content,
            engine: this.name,
            score: this.weight,
            category: "videos",
            template: "videos",
            duration: item.duration || "",
            source: item.provider || "",
            thumbnailUrl: thumbnail,
            embedUrl: item.embed_url || ""
          });
        }
      }
    } catch {
    }
    return results;
  }
};
var DuckDuckGoNewsEngine = class {
  static {
    __name(this, "DuckDuckGoNewsEngine");
  }
  name = "duckduckgo news";
  shortcut = "ddn";
  categories = ["news"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 1e4;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const region = getDdgRegion(params.locale);
    const vqd = params.engineData["vqd"] || "";
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("o", "json");
    searchParams.set("l", region);
    searchParams.set("f", ",,,,,");
    searchParams.set("vqd", vqd);
    if (params.page > 1) {
      searchParams.set("s", ((params.page - 1) * 30).toString());
    }
    return {
      url: `https://duckduckgo.com/news.js?${searchParams.toString()}`,
      method: "GET",
      headers: {
        "User-Agent": DDG_USER_AGENT,
        Accept: "application/json, text/javascript, */*; q=0.01",
        Referer: "https://duckduckgo.com/",
        "X-Requested-With": "XMLHttpRequest",
        ...secFetchHeaders
      },
      cookies: [`l=${region}`]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const jsonStart = body.indexOf("{");
    if (jsonStart === -1) return results;
    try {
      const data = JSON.parse(body.slice(jsonStart));
      if (data.results) {
        for (const item of data.results) {
          let publishedAt = "";
          if (item.date && item.date > 0) {
            publishedAt = new Date(item.date * 1e3).toISOString();
          }
          results.results.push({
            url: item.url || "",
            title: item.title || "",
            content: item.excerpt || "",
            engine: this.name,
            score: this.weight,
            category: "news",
            template: "news",
            source: item.source || "",
            thumbnailUrl: item.image || "",
            publishedAt
          });
        }
      }
    } catch {
    }
    return results;
  }
};

// src/engines/brave.ts
var timeRangeMap2 = {
  day: "pd",
  week: "pw",
  month: "pm",
  year: "py"
};
var BraveEngine = class {
  static {
    __name(this, "BraveEngine");
  }
  name = "brave";
  shortcut = "br";
  categories = ["general"];
  supportsPaging = true;
  maxPage = 50;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("source", "web");
    if (params.page > 1) {
      searchParams.set("offset", (params.page - 1).toString());
    }
    if (params.timeRange && timeRangeMap2[params.timeRange]) {
      searchParams.set("tf", timeRangeMap2[params.timeRange]);
    }
    let safeValue = "moderate";
    if (params.safeSearch === 2) {
      safeValue = "strict";
    } else if (params.safeSearch === 0) {
      safeValue = "off";
    }
    return {
      url: `https://search.brave.com/search?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "text/html",
        "Accept-Language": "en-US,en;q=0.9"
      },
      cookies: [`safesearch=${safeValue}`]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const snippetElements = findElements(body, "div.snippet");
    for (const el of snippetElements) {
      const result = this.parseResult(el);
      if (result) {
        results.results.push(result);
      }
    }
    return results;
  }
  parseResult(html) {
    let url = "";
    let title2 = "";
    let content = "";
    const linkPattern = /<a\b[^>]*?\bhref\s*=\s*"(https?:\/\/[^"]+)"[^>]*>([\s\S]*?)<\/a>/gi;
    let linkMatch;
    while ((linkMatch = linkPattern.exec(html)) !== null) {
      const href = decodeHtmlEntities(linkMatch[1]);
      if (href.startsWith("http")) {
        url = href;
        title2 = extractText(linkMatch[2]).trim();
        break;
      }
    }
    if (!url || !title2) return null;
    const contentElements = findElements(html, "div.content");
    if (contentElements.length > 0) {
      const text = extractText(contentElements[0]).trim();
      if (text.length > content.length) {
        content = text;
      }
    }
    const descElements = findElements(html, "div.snippet-description");
    if (descElements.length > 0) {
      const text = extractText(descElements[0]).trim();
      if (text.length > content.length) {
        content = text;
      }
    }
    return {
      url,
      title: title2,
      content,
      engine: this.name,
      score: this.weight,
      category: "general"
    };
  }
};

// src/engines/wikipedia.ts
var languageMap = {
  en: "en",
  de: "de",
  fr: "fr",
  es: "es",
  it: "it",
  pt: "pt",
  ja: "ja",
  ko: "ko",
  zh: "zh",
  ru: "ru",
  ar: "ar",
  hi: "hi",
  nl: "nl",
  pl: "pl",
  sv: "sv",
  vi: "vi",
  uk: "uk",
  he: "he",
  id: "id",
  cs: "cs",
  fi: "fi",
  da: "da",
  no: "no",
  hu: "hu",
  ro: "ro",
  tr: "tr",
  th: "th",
  el: "el",
  fa: "fa",
  ca: "ca"
};
function stripHtmlTags(s) {
  let result = s;
  let start = result.indexOf("<");
  while (start !== -1) {
    const end = result.indexOf(">", start);
    if (end === -1) break;
    result = result.slice(0, start) + result.slice(end + 1);
    start = result.indexOf("<");
  }
  result = result.replace(/&quot;/g, '"');
  result = result.replace(/&amp;/g, "&");
  result = result.replace(/&lt;/g, "<");
  result = result.replace(/&gt;/g, ">");
  result = result.replace(/&#39;/g, "'");
  result = result.replace(/&nbsp;/g, " ");
  return result.trim();
}
__name(stripHtmlTags, "stripHtmlTags");
var WikipediaEngine = class {
  static {
    __name(this, "WikipediaEngine");
  }
  name = "wikipedia";
  shortcut = "w";
  categories = ["general"];
  supportsPaging = true;
  maxPage = 10;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    let lang = "en";
    if (params.locale) {
      const parts = params.locale.split("-");
      const langCode = parts[0].toLowerCase();
      if (languageMap[langCode]) {
        lang = languageMap[langCode];
      }
    }
    const searchParams = new URLSearchParams();
    searchParams.set("action", "query");
    searchParams.set("list", "search");
    searchParams.set("srsearch", query);
    searchParams.set("srwhat", "text");
    searchParams.set("srlimit", "10");
    searchParams.set("srprop", "snippet|titlesnippet|timestamp");
    searchParams.set("format", "json");
    searchParams.set("utf8", "1");
    if (params.page > 1) {
      searchParams.set("sroffset", ((params.page - 1) * 10).toString());
    }
    return {
      url: `https://${lang}.wikipedia.org/w/api.php?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "application/json",
        "User-Agent": "MizuSearch/1.0 (https://github.com/go-mizu/mizu; mizu@example.com)"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    try {
      const data = JSON.parse(body);
      if (!data.query?.search) return results;
      const lang = "en";
      for (const item of data.query.search) {
        if (!item.title) continue;
        const articleUrl = `https://${lang}.wikipedia.org/wiki/${encodeURIComponent(item.title.replace(/ /g, "_"))}`;
        const snippet = stripHtmlTags(item.snippet || "");
        results.results.push({
          url: articleUrl,
          title: item.title,
          content: snippet,
          engine: this.name,
          score: this.weight,
          category: "general"
        });
      }
    } catch {
    }
    return results;
  }
};

// src/engines/youtube.ts
var timeRangeMap3 = {
  day: "Ag",
  week: "Aw",
  month: "BA",
  year: "BQ"
};
var YouTubeEngine = class {
  static {
    __name(this, "YouTubeEngine");
  }
  name = "youtube";
  shortcut = "yt";
  categories = ["videos"];
  supportsPaging = true;
  maxPage = 5;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const searchParams = new URLSearchParams();
    searchParams.set("search_query", query);
    if (params.timeRange && timeRangeMap3[params.timeRange]) {
      const sp = `EgIIA${timeRangeMap3[params.timeRange]}%3D%3D`;
      searchParams.set("sp", sp);
    }
    return {
      url: `https://www.youtube.com/results?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "text/html",
        "Accept-Language": "en-US,en;q=0.9"
      },
      cookies: ["CONSENT=YES+"]
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const match2 = body.match(
      /var ytInitialData = ({.+?});<\/script>/
    );
    if (!match2) return results;
    let data;
    try {
      data = JSON.parse(match2[1]);
    } catch {
      return results;
    }
    const sectionContents = data.contents?.twoColumnSearchResultsRenderer?.primaryContents?.sectionListRenderer?.contents;
    if (!sectionContents) return results;
    for (const section of sectionContents) {
      const items = section.itemSectionRenderer?.contents;
      if (!items) continue;
      for (const item of items) {
        const vr = item.videoRenderer;
        if (!vr?.videoId) continue;
        let title2 = "";
        if (vr.title?.runs) {
          title2 = vr.title.runs.map((r) => r.text || "").join("");
        }
        let description = "";
        if (vr.descriptionSnippet?.runs) {
          description = vr.descriptionSnippet.runs.map((r) => r.text || "").join("");
        }
        let channel2 = "";
        if (vr.ownerText?.runs && vr.ownerText.runs.length > 0) {
          channel2 = vr.ownerText.runs[0].text || "";
        }
        let thumbnailUrl = "";
        const thumbnails = vr.thumbnail?.thumbnails;
        if (thumbnails && thumbnails.length > 0) {
          thumbnailUrl = thumbnails[thumbnails.length - 1].url || "";
        }
        const duration = vr.lengthText?.simpleText || "";
        let views = 0;
        const viewText = vr.viewCountText?.simpleText || "";
        const viewMatch = viewText.match(/([\d,]+)\s*view/i);
        if (viewMatch) {
          views = parseInt(viewMatch[1].replace(/,/g, ""), 10) || 0;
        }
        const videoUrl = `https://www.youtube.com/watch?v=${vr.videoId}`;
        const embedUrl = `https://www.youtube-nocookie.com/embed/${vr.videoId}`;
        results.results.push({
          url: videoUrl,
          title: title2,
          content: description,
          engine: this.name,
          score: this.weight,
          category: "videos",
          template: "videos",
          duration,
          embedUrl,
          thumbnailUrl,
          channel: channel2,
          views
        });
      }
    }
    return results;
  }
};

// src/engines/reddit.ts
var INVALID_THUMBNAILS = /* @__PURE__ */ new Set(["self", "default", "nsfw", "spoiler", ""]);
function isValidUrl(s) {
  if (!s || INVALID_THUMBNAILS.has(s)) return false;
  try {
    const url = new URL(s);
    return url.protocol === "http:" || url.protocol === "https:";
  } catch {
    return false;
  }
}
__name(isValidUrl, "isValidUrl");
function truncateText(s, maxLen) {
  if (s.length <= maxLen) return s;
  return s.slice(0, maxLen) + "...";
}
__name(truncateText, "truncateText");
var RedditEngine = class {
  static {
    __name(this, "RedditEngine");
  }
  name = "reddit";
  shortcut = "re";
  categories = ["social"];
  supportsPaging = false;
  maxPage = 1;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, _params) {
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("limit", "25");
    return {
      url: `https://www.reddit.com/search.json?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "application/json",
        "User-Agent": "Mozilla/5.0 (compatible; SearXNG)"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    try {
      const data = JSON.parse(body);
      if (!data.data?.children) return results;
      for (const child of data.data.children) {
        const post = child.data;
        if (!post?.permalink || !post.title) continue;
        const url = `https://www.reddit.com${post.permalink}`;
        const content = truncateText(post.selftext || "", 500);
        let publishedAt = "";
        if (post.created_utc && post.created_utc > 0) {
          publishedAt = new Date(post.created_utc * 1e3).toISOString();
        }
        let thumbnailUrl = "";
        let imageUrl = "";
        if (isValidUrl(post.thumbnail || "")) {
          thumbnailUrl = post.thumbnail;
          if (isValidUrl(post.url || "")) {
            imageUrl = post.url;
          }
        }
        results.results.push({
          url,
          title: post.title,
          content,
          engine: this.name,
          score: this.weight,
          category: "social",
          publishedAt,
          thumbnailUrl,
          imageUrl: imageUrl || void 0,
          template: thumbnailUrl ? "images" : void 0,
          source: post.subreddit ? `r/${post.subreddit}` : void 0,
          channel: post.author || void 0
        });
      }
    } catch {
    }
    return results;
  }
};

// src/lib/xml-parser.ts
function getElementsByTagName(xml, tagName) {
  const results = [];
  const openPattern = new RegExp(`<${tagName}(?:\\s[^>]*)?>`, "gi");
  const closeTag = `</${tagName}>`;
  let openMatch;
  while ((openMatch = openPattern.exec(xml)) !== null) {
    const contentStart = openMatch.index + openMatch[0].length;
    if (openMatch[0].endsWith("/>")) {
      results.push("");
      continue;
    }
    let depth = 1;
    let pos = contentStart;
    while (depth > 0 && pos < xml.length) {
      const nextOpen = xml.indexOf(`<${tagName}`, pos);
      const nextClose = xml.toLowerCase().indexOf(closeTag.toLowerCase(), pos);
      if (nextClose === -1) {
        break;
      }
      if (nextOpen !== -1 && nextOpen < nextClose) {
        const afterTag = xml[nextOpen + tagName.length + 1];
        if (afterTag === ">" || afterTag === " " || afterTag === "/" || afterTag === "\n" || afterTag === "\r" || afterTag === "	") {
          const openEnd = xml.indexOf(">", nextOpen);
          if (openEnd !== -1 && xml[openEnd - 1] !== "/") {
            depth++;
          }
        }
        pos = nextOpen + tagName.length + 1;
      } else {
        depth--;
        if (depth === 0) {
          results.push(xml.slice(contentStart, nextClose));
        }
        pos = nextClose + closeTag.length;
      }
    }
  }
  return results;
}
__name(getElementsByTagName, "getElementsByTagName");
function getTextContent(xml, tagName) {
  const elements = getElementsByTagName(xml, tagName);
  if (elements.length === 0) {
    return "";
  }
  const stripped = elements[0].replace(/<[^>]+>/g, "");
  return decodeHtmlEntities(stripped).trim();
}
__name(getTextContent, "getTextContent");
function getElementAttribute(xml, tagName, attrName) {
  const results = [];
  const pattern = new RegExp(
    `<${tagName}\\b[^>]*?\\b${attrName}\\s*=\\s*(?:"([^"]*)"|'([^']*)')`,
    "gi"
  );
  let match2;
  while ((match2 = pattern.exec(xml)) !== null) {
    results.push(decodeHtmlEntities(match2[1] ?? match2[2] ?? ""));
  }
  return results;
}
__name(getElementAttribute, "getElementAttribute");

// src/engines/arxiv.ts
var ArxivEngine = class {
  static {
    __name(this, "ArxivEngine");
  }
  name = "arxiv";
  shortcut = "arx";
  categories = ["science"];
  supportsPaging = true;
  maxPage = 10;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, params) {
    const maxResults = 10;
    let start = 0;
    if (params.page > 1) {
      start = (params.page - 1) * maxResults;
    }
    const searchParams = new URLSearchParams();
    searchParams.set("search_query", `all:${query}`);
    searchParams.set("start", start.toString());
    searchParams.set("max_results", maxResults.toString());
    return {
      url: `https://export.arxiv.org/api/query?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "application/atom+xml"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    const entries = getElementsByTagName(body, "entry");
    for (const entry of entries) {
      const id = getTextContent(entry, "id").trim();
      if (!id) continue;
      const title2 = getTextContent(entry, "title").replace(/\s+/g, " ").trim();
      const summary = getTextContent(entry, "summary").replace(/\s+/g, " ").trim();
      const authorNames = getElementsByTagName(entry, "author");
      const authors = [];
      for (const authorXml of authorNames) {
        const name = getTextContent(authorXml, "name").trim();
        if (name) {
          authors.push(name);
        }
      }
      const published = getTextContent(entry, "published").trim();
      let publishedAt = "";
      if (published) {
        try {
          publishedAt = new Date(published).toISOString();
        } catch {
          publishedAt = published;
        }
      }
      const linkHrefs = getElementAttribute(entry, "link", "href");
      const linkTitles = getElementAttribute(entry, "link", "title");
      let pdfUrl = "";
      for (let i = 0; i < linkTitles.length; i++) {
        if (linkTitles[i] === "pdf" && linkHrefs[i]) {
          pdfUrl = linkHrefs[i];
          break;
        }
      }
      const doi = getTextContent(entry, "doi").trim();
      const journal = getTextContent(entry, "journal_ref").trim();
      let content = summary;
      if (pdfUrl) {
        content += ` [PDF: ${pdfUrl}]`;
      }
      results.results.push({
        url: id,
        title: title2,
        content,
        engine: this.name,
        score: this.weight,
        category: "science",
        template: "paper",
        authors,
        publishedAt,
        doi: doi || void 0,
        journal: journal || void 0
      });
    }
    return results;
  }
};

// src/engines/github.ts
var GitHubEngine = class {
  static {
    __name(this, "GitHubEngine");
  }
  name = "github";
  shortcut = "gh";
  categories = ["it"];
  supportsPaging = false;
  maxPage = 1;
  timeout = 5e3;
  weight = 1;
  disabled = false;
  buildRequest(query, _params) {
    const searchParams = new URLSearchParams();
    searchParams.set("q", query);
    searchParams.set("sort", "stars");
    searchParams.set("order", "desc");
    return {
      url: `https://api.github.com/search/repositories?${searchParams.toString()}`,
      method: "GET",
      headers: {
        Accept: "application/vnd.github.preview.text-match+json",
        "User-Agent": "SearXNG"
      },
      cookies: []
    };
  }
  parseResponse(body, _params) {
    const results = newEngineResults();
    try {
      const data = JSON.parse(body);
      if (!data.items) return results;
      for (const item of data.items) {
        if (!item.html_url || !item.full_name) continue;
        let content = item.description || "";
        const meta = [];
        if (item.stargazers_count !== void 0 && item.stargazers_count > 0) {
          meta.push(`${formatStars(item.stargazers_count)} stars`);
        }
        if (item.language) {
          meta.push(item.language);
        }
        if (item.topics && item.topics.length > 0) {
          meta.push(item.topics.slice(0, 5).join(", "));
        }
        if (meta.length > 0) {
          content = content ? `${content} | ${meta.join(" | ")}` : meta.join(" | ");
        }
        let publishedAt = "";
        if (item.updated_at) {
          try {
            publishedAt = new Date(item.updated_at).toISOString();
          } catch {
            publishedAt = item.updated_at;
          }
        }
        results.results.push({
          url: item.html_url,
          title: item.full_name,
          content,
          engine: this.name,
          score: this.weight,
          category: "it",
          template: "packages",
          thumbnailUrl: item.owner?.avatar_url || "",
          publishedAt,
          stars: item.stargazers_count || 0,
          language: item.language || "",
          topics: item.topics || []
        });
      }
    } catch {
    }
    return results;
  }
};
function formatStars(count) {
  if (count >= 1e6) {
    return (count / 1e6).toFixed(1) + "M";
  }
  if (count >= 1e3) {
    return (count / 1e3).toFixed(1) + "k";
  }
  return count.toString();
}
__name(formatStars, "formatStars");

// src/engines/metasearch.ts
var MetaSearch = class {
  static {
    __name(this, "MetaSearch");
  }
  engines = /* @__PURE__ */ new Map();
  /**
   * Register an engine in the registry.
   */
  register(engine) {
    this.engines.set(engine.name, engine);
  }
  /**
   * Get an engine by name.
   */
  get(name) {
    return this.engines.get(name);
  }
  /**
   * Get all engines for a given category.
   * Only returns enabled engines.
   */
  getByCategory(category) {
    const matched = [];
    for (const engine of this.engines.values()) {
      if (engine.disabled) continue;
      if (engine.categories.includes(category)) {
        matched.push(engine);
      }
    }
    return matched;
  }
  /**
   * Get all registered engine names.
   */
  listEngines() {
    return Array.from(this.engines.keys());
  }
  /**
   * Perform a metasearch across all engines in the given category.
   *
   * 1. Gets engines for the category
   * 2. Executes all in parallel with Promise.allSettled
   * 3. Collects results and suggestions
   * 4. Deduplicates by URL (merges scores for duplicates)
   * 5. Sorts by score descending
   * 6. Returns paginated results with metadata
   */
  async search(query, category, params) {
    const engines = this.getByCategory(category);
    if (engines.length === 0) {
      return {
        results: [],
        suggestions: [],
        corrections: [],
        totalEngines: 0,
        successfulEngines: 0,
        failedEngines: []
      };
    }
    const promises = engines.map(
      (engine) => executeEngine(engine, query, params).then(
        (result) => ({ engine: engine.name, result, error: null }),
        (error) => ({
          engine: engine.name,
          result: null,
          error: error instanceof Error ? error.message : String(error)
        })
      )
    );
    const settled = await Promise.allSettled(promises);
    const allResults = [];
    const allSuggestions = [];
    const allCorrections = [];
    const failedEngines = [];
    let successfulEngines = 0;
    for (const outcome of settled) {
      if (outcome.status === "rejected") {
        continue;
      }
      const { engine: engineName, result, error } = outcome.value;
      if (error || !result) {
        failedEngines.push(engineName);
        continue;
      }
      successfulEngines++;
      allResults.push(...result.results);
      allSuggestions.push(...result.suggestions);
      allCorrections.push(...result.corrections);
    }
    const deduped = this.deduplicateResults(allResults);
    deduped.sort((a, b) => b.score - a.score);
    const uniqueSuggestions = [...new Set(allSuggestions)];
    const uniqueCorrections = [...new Set(allCorrections)];
    return {
      results: deduped,
      suggestions: uniqueSuggestions,
      corrections: uniqueCorrections,
      totalEngines: engines.length,
      successfulEngines,
      failedEngines
    };
  }
  /**
   * Deduplicate results by URL.
   * When two results share the same URL, merge their scores
   * (additive) and keep the one with more content.
   */
  deduplicateResults(results) {
    const urlMap = /* @__PURE__ */ new Map();
    for (const result of results) {
      const normalizedUrl = this.normalizeUrl(result.url);
      if (!normalizedUrl) continue;
      const existing = urlMap.get(normalizedUrl);
      if (existing) {
        existing.score += result.score;
        if (result.content.length > existing.content.length) {
          existing.content = result.content;
        }
        if (!existing.title && result.title) {
          existing.title = result.title;
        }
        if (!existing.thumbnailUrl && result.thumbnailUrl) {
          existing.thumbnailUrl = result.thumbnailUrl;
        }
      } else {
        urlMap.set(normalizedUrl, { ...result });
      }
    }
    return Array.from(urlMap.values());
  }
  /**
   * Normalize a URL for deduplication.
   * Strips trailing slashes, removes www. prefix, lowercases host.
   */
  normalizeUrl(url) {
    if (!url) return "";
    try {
      const parsed = new URL(url);
      let host = parsed.hostname.toLowerCase();
      if (host.startsWith("www.")) {
        host = host.slice(4);
      }
      let path = parsed.pathname;
      if (path.endsWith("/") && path.length > 1) {
        path = path.slice(0, -1);
      }
      return `${parsed.protocol}//${host}${path}${parsed.search}`;
    } catch {
      return url.toLowerCase();
    }
  }
};
function createDefaultMetaSearch() {
  const ms = new MetaSearch();
  ms.register(new GoogleEngine());
  ms.register(new BingEngine());
  ms.register(new BraveEngine());
  ms.register(new WikipediaEngine());
  ms.register(new GoogleImagesEngine());
  ms.register(new BingImagesEngine());
  ms.register(new DuckDuckGoImagesEngine());
  ms.register(new YouTubeEngine());
  ms.register(new DuckDuckGoVideosEngine());
  ms.register(new BingNewsEngine());
  ms.register(new DuckDuckGoNewsEngine());
  ms.register(new ArxivEngine());
  ms.register(new GitHubEngine());
  ms.register(new RedditEngine());
  return ms;
}
__name(createDefaultMetaSearch, "createDefaultMetaSearch");

// src/store/cache.ts
var TTL_SEARCH = 300;
var TTL_SUGGEST = 60;
var TTL_KNOWLEDGE = 3600;
var TTL_INSTANT = 600;
var CacheStore = class {
  static {
    __name(this, "CacheStore");
  }
  kv;
  constructor(kv) {
    this.kv = kv;
  }
  // --- Search results cache ---
  async getSearch(hash) {
    const raw2 = await this.kv.get(`cache:search:${hash}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async setSearch(hash, response) {
    await this.kv.put(`cache:search:${hash}`, JSON.stringify(response), {
      expirationTtl: TTL_SEARCH
    });
  }
  // --- Suggestions cache ---
  async getSuggest(hash) {
    const raw2 = await this.kv.get(`cache:suggest:${hash}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async setSuggest(hash, suggestions) {
    await this.kv.put(`cache:suggest:${hash}`, JSON.stringify(suggestions), {
      expirationTtl: TTL_SUGGEST
    });
  }
  // --- Knowledge panel cache ---
  async getKnowledge(query) {
    const raw2 = await this.kv.get(`cache:knowledge:${query}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async setKnowledge(query, panel) {
    await this.kv.put(`cache:knowledge:${query}`, JSON.stringify(panel), {
      expirationTtl: TTL_KNOWLEDGE
    });
  }
  // --- Instant answer cache ---
  async getInstant(hash) {
    const raw2 = await this.kv.get(`cache:instant:${hash}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async setInstant(hash, answer) {
    await this.kv.put(`cache:instant:${hash}`, JSON.stringify(answer), {
      expirationTtl: TTL_INSTANT
    });
  }
};

// src/store/kv.ts
var DEFAULT_SETTINGS = {
  safe_search: "moderate",
  results_per_page: 10,
  region: "",
  language: "en",
  theme: "system",
  open_in_new_tab: false,
  show_thumbnails: true
};
var DEFAULT_WIDGET_SETTINGS = {
  calculator: true,
  unit_converter: true,
  currency: true,
  weather: true,
  dictionary: true,
  time_zones: true,
  knowledge_panel: true
};
var MAX_HISTORY = 100;
var KVStore = class {
  static {
    __name(this, "KVStore");
  }
  kv;
  constructor(kv) {
    this.kv = kv;
  }
  // --- Settings ---
  async getSettings() {
    const raw2 = await this.kv.get("settings:default");
    if (!raw2) {
      return { ...DEFAULT_SETTINGS };
    }
    return JSON.parse(raw2);
  }
  async updateSettings(settings) {
    const current = await this.getSettings();
    const merged = { ...current, ...settings };
    await this.kv.put("settings:default", JSON.stringify(merged));
    return merged;
  }
  // --- Preferences ---
  async listPreferences() {
    const index = await this.getIndex("preferences:_index");
    const preferences = [];
    for (const domain2 of index) {
      const pref = await this.getPreference(domain2);
      if (pref) {
        preferences.push(pref);
      }
    }
    return preferences;
  }
  async getPreference(domain2) {
    const raw2 = await this.kv.get(`preferences:${domain2}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async setPreference(pref) {
    await this.kv.put(`preferences:${pref.domain}`, JSON.stringify(pref));
    await this.addToIndex("preferences:_index", pref.domain);
  }
  async deletePreference(domain2) {
    await this.kv.delete(`preferences:${domain2}`);
    await this.removeFromIndex("preferences:_index", domain2);
  }
  // --- Lenses ---
  async listLenses() {
    const index = await this.getIndex("lenses:_index");
    const lenses = [];
    for (const id of index) {
      const lens = await this.getLens(id);
      if (lens) {
        lenses.push(lens);
      }
    }
    return lenses;
  }
  async getLens(id) {
    const raw2 = await this.kv.get(`lenses:${id}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async createLens(lens) {
    await this.kv.put(`lenses:${lens.id}`, JSON.stringify(lens));
    await this.addToIndex("lenses:_index", lens.id);
  }
  async updateLens(id, lens) {
    const current = await this.getLens(id);
    if (!current) return null;
    const updated = {
      ...current,
      ...lens,
      id,
      updated_at: (/* @__PURE__ */ new Date()).toISOString()
    };
    await this.kv.put(`lenses:${id}`, JSON.stringify(updated));
    return updated;
  }
  async deleteLens(id) {
    await this.kv.delete(`lenses:${id}`);
    await this.removeFromIndex("lenses:_index", id);
  }
  // --- History ---
  async listHistory(limit) {
    const index = await this.getIndex("history:_index");
    const sliced = limit ? index.slice(0, limit) : index;
    const entries = [];
    for (const id of sliced) {
      const entry = await this.getHistoryEntry(id);
      if (entry) {
        entries.push(entry);
      }
    }
    return entries;
  }
  async getHistoryEntry(id) {
    const raw2 = await this.kv.get(`history:${id}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async addHistory(entry) {
    await this.kv.put(`history:${entry.id}`, JSON.stringify(entry));
    const index = await this.getIndex("history:_index");
    const updated = [entry.id, ...index.filter((id) => id !== entry.id)];
    if (updated.length > MAX_HISTORY) {
      const removed = updated.slice(MAX_HISTORY);
      for (const id of removed) {
        await this.kv.delete(`history:${id}`);
      }
    }
    await this.setIndex("history:_index", updated.slice(0, MAX_HISTORY));
  }
  async deleteHistory(id) {
    await this.kv.delete(`history:${id}`);
    await this.removeFromIndex("history:_index", id);
  }
  async clearHistory() {
    const index = await this.getIndex("history:_index");
    for (const id of index) {
      await this.kv.delete(`history:${id}`);
    }
    await this.setIndex("history:_index", []);
  }
  // --- Bangs ---
  async listBangs() {
    const index = await this.getIndex("bangs:_index");
    const bangs = [];
    for (const trigger of index) {
      const bang = await this.getBang(trigger);
      if (bang) {
        bangs.push(bang);
      }
    }
    return bangs;
  }
  async getBang(trigger) {
    const raw2 = await this.kv.get(`bangs:${trigger}`);
    if (!raw2) return null;
    return JSON.parse(raw2);
  }
  async createBang(bang) {
    await this.kv.put(`bangs:${bang.trigger}`, JSON.stringify(bang));
    await this.addToIndex("bangs:_index", bang.trigger);
    if (!bang.is_builtin) {
      await this.addToIndex("bangs:_custom", bang.trigger);
    }
  }
  async deleteBang(trigger) {
    await this.kv.delete(`bangs:${trigger}`);
    await this.removeFromIndex("bangs:_index", trigger);
    await this.removeFromIndex("bangs:_custom", trigger);
  }
  // --- Widget Settings ---
  async getWidgetSettings() {
    const raw2 = await this.kv.get("widgets:settings");
    if (!raw2) {
      return { ...DEFAULT_WIDGET_SETTINGS };
    }
    return JSON.parse(raw2);
  }
  async updateWidgetSettings(settings) {
    const current = await this.getWidgetSettings();
    const merged = { ...current, ...settings };
    await this.kv.put("widgets:settings", JSON.stringify(merged));
    return merged;
  }
  // --- Index helpers ---
  async getIndex(key) {
    const raw2 = await this.kv.get(key);
    if (!raw2) return [];
    return JSON.parse(raw2);
  }
  async setIndex(key, index) {
    await this.kv.put(key, JSON.stringify(index));
  }
  async addToIndex(key, value) {
    const index = await this.getIndex(key);
    if (!index.includes(value)) {
      index.push(value);
      await this.setIndex(key, index);
    }
  }
  async removeFromIndex(key, value) {
    const index = await this.getIndex(key);
    const filtered = index.filter((item) => item !== value);
    if (filtered.length !== index.length) {
      await this.setIndex(key, filtered);
    }
  }
};

// src/services/search.ts
function parseTimeRange(tr) {
  if (tr === "day" || tr === "week" || tr === "month" || tr === "year") {
    return tr;
  }
  return "";
}
__name(parseTimeRange, "parseTimeRange");
function toEngineParams(options) {
  let category = "general";
  if (options.file_type === "image") {
    category = "images";
  } else if (options.file_type === "video") {
    category = "videos";
  } else if (options.file_type === "news") {
    category = "news";
  }
  return {
    category,
    params: {
      page: options.page,
      locale: options.language ?? "en",
      timeRange: parseTimeRange(options.time_range),
      safeSearch: options.safe_search === "strict" ? 2 : options.safe_search === "off" ? 0 : 1,
      engineData: {}
    }
  };
}
__name(toEngineParams, "toEngineParams");
function toSearchResult(r, index) {
  return {
    id: `${Date.now().toString(36)}-${index}`,
    url: r.url,
    title: r.title,
    snippet: r.content,
    domain: extractDomain(r.url),
    thumbnail: r.thumbnailUrl ? { url: r.thumbnailUrl } : void 0,
    published: r.publishedAt,
    score: r.score,
    crawled_at: (/* @__PURE__ */ new Date()).toISOString(),
    engine: r.engine,
    engines: [r.engine]
  };
}
__name(toSearchResult, "toSearchResult");
function extractDomain(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return "";
  }
}
__name(extractDomain, "extractDomain");
var CALC_PATTERN = /^\d+[\s]*[+\-*/^%][\s]*\d+/;
var FUNC_PATTERN = /^(sqrt|sin|cos|tan|log|ln|abs|ceil|floor|round)\s*\(/i;
var UNIT_PATTERN = /^(\d+\.?\d*)\s*(mm|cm|m|km|in|ft|yd|mi|mg|g|kg|lb|oz|ton|c|f|k|ml|l|gal|qt|pt|cup|tbsp|tsp|fl_oz|mm2|cm2|m2|km2|in2|ft2|acre|hectare|m\/s|km\/h|mph|knots|b|kb|mb|gb|tb|pb|ms|s|min|hr|day|week|month|year)\s+(to|in)\s+(mm|cm|m|km|in|ft|yd|mi|mg|g|kg|lb|oz|ton|c|f|k|ml|l|gal|qt|pt|cup|tbsp|tsp|fl_oz|mm2|cm2|m2|km2|in2|ft2|acre|hectare|m\/s|km\/h|mph|knots|b|kb|mb|gb|tb|pb|ms|s|min|hr|day|week|month|year)$/i;
var CURRENCY_PATTERN = /^(\d+\.?\d*)\s*(usd|eur|gbp|jpy|cad|aud|chf|cny|inr|krw|brl|mxn|sgd|hkd|nzd|sek|nok|dkk|pln|zar|try|thb|idr|php|czk|ils|clp|myr|twd|ars|cop|sar|aed|egp|vnd|bgn|hrk|huf|isk|ron|rub)\s+(to|in)\s+(usd|eur|gbp|jpy|cad|aud|chf|cny|inr|krw|brl|mxn|sgd|hkd|nzd|sek|nok|dkk|pln|zar|try|thb|idr|php|czk|ils|clp|myr|twd|ars|cop|sar|aed|egp|vnd|bgn|hrk|huf|isk|ron|rub)$/i;
var WEATHER_PATTERN = /^weather\s+(in\s+)?(.+)/i;
var DEFINE_PATTERN = /^(?:define|meaning\s+of)\s+(.+)/i;
var TIME_PATTERN = /^(?:time\s+in|what\s+time.*in)\s+(.+)/i;
function generateId() {
  const timestamp = Date.now().toString(36);
  const random = Math.random().toString(36).substring(2, 8);
  return `${timestamp}-${random}`;
}
__name(generateId, "generateId");
function hashSearchKey(query, options) {
  const key = `${query}|${options.page}|${options.per_page}|${options.time_range ?? ""}|${options.region ?? ""}|${options.language ?? ""}|${options.safe_search ?? ""}|${options.site ?? ""}|${options.lens ?? ""}`;
  let hash = 0;
  for (let i = 0; i < key.length; i++) {
    const char = key.charCodeAt(i);
    hash = (hash << 5) - hash + char | 0;
  }
  return Math.abs(hash).toString(36);
}
__name(hashSearchKey, "hashSearchKey");
var SearchService = class {
  static {
    __name(this, "SearchService");
  }
  metasearch;
  cache;
  kvStore;
  bangService;
  instantService;
  knowledgeService;
  constructor(metasearch, cache, kvStore, bangService, instantService, knowledgeService) {
    this.metasearch = metasearch;
    this.cache = cache;
    this.kvStore = kvStore;
    this.bangService = bangService;
    this.instantService = instantService;
    this.knowledgeService = knowledgeService;
  }
  /**
   * Main search method. Handles bang redirects, caching, instant answers,
   * knowledge panels, and metasearch aggregation.
   */
  async search(query, options) {
    const startTime2 = Date.now();
    const trimmedQuery = query.trim();
    if (!trimmedQuery) {
      return this.emptyResponse(trimmedQuery, options, 0);
    }
    const bangResult = await this.bangService.parse(trimmedQuery);
    if (bangResult.redirect) {
      return {
        ...this.emptyResponse(bangResult.query, options, Date.now() - startTime2),
        redirect: bangResult.redirect,
        bang: bangResult.bang?.trigger,
        category: bangResult.category
      };
    }
    const searchQuery = bangResult.query;
    const cacheHash = hashSearchKey(searchQuery, options);
    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }
    const { category, params } = toEngineParams(options);
    const [instantAnswer, knowledgePanel, metaResult] = await Promise.all([
      this.detectInstantAnswer(searchQuery),
      options.page === 1 ? this.knowledgeService.getPanel(searchQuery) : Promise.resolve(null),
      this.metasearch.search(searchQuery, category, params)
    ]);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;
    const response = {
      query: searchQuery,
      corrected_query: metaResult.corrections[0],
      total_results: totalResults,
      results: paginatedResults,
      suggestions: metaResult.suggestions,
      instant_answer: instantAnswer ?? void 0,
      knowledge_panel: knowledgePanel ?? void 0,
      search_time_ms: Date.now() - startTime2,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore
    };
    await this.cache.setSearch(cacheHash, response);
    this.addToHistory(searchQuery, totalResults).catch(() => {
    });
    return response;
  }
  /**
   * Search for images.
   */
  async searchImages(query, options) {
    const startTime2 = Date.now();
    const imageOptions = { ...options, file_type: "image" };
    const cacheHash = hashSearchKey(`img:${query}`, imageOptions);
    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }
    const { params } = toEngineParams(imageOptions);
    const metaResult = await this.metasearch.search(query, "images", params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;
    const response = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      search_time_ms: Date.now() - startTime2,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore
    };
    await this.cache.setSearch(cacheHash, response);
    return response;
  }
  /**
   * Search for videos.
   */
  async searchVideos(query, options) {
    const startTime2 = Date.now();
    const videoOptions = { ...options, file_type: "video" };
    const cacheHash = hashSearchKey(`vid:${query}`, videoOptions);
    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }
    const { params } = toEngineParams(videoOptions);
    const metaResult = await this.metasearch.search(query, "videos", params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;
    const response = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      search_time_ms: Date.now() - startTime2,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore
    };
    await this.cache.setSearch(cacheHash, response);
    return response;
  }
  /**
   * Search for news.
   */
  async searchNews(query, options) {
    const startTime2 = Date.now();
    const newsOptions = { ...options, file_type: "news" };
    const cacheHash = hashSearchKey(`news:${query}`, newsOptions);
    const cachedResponse = await this.cache.getSearch(cacheHash);
    if (cachedResponse) {
      return cachedResponse;
    }
    const { params } = toEngineParams(newsOptions);
    const metaResult = await this.metasearch.search(query, "news", params);
    const allResults = metaResult.results.map(toSearchResult);
    const startIndex = (options.page - 1) * options.per_page;
    const endIndex = startIndex + options.per_page;
    const paginatedResults = allResults.slice(startIndex, endIndex);
    const totalResults = allResults.length;
    const hasMore = endIndex < totalResults;
    const response = {
      query,
      total_results: totalResults,
      results: paginatedResults,
      search_time_ms: Date.now() - startTime2,
      page: options.page,
      per_page: options.per_page,
      has_more: hasMore
    };
    await this.cache.setSearch(cacheHash, response);
    return response;
  }
  /**
   * Detect and compute instant answers based on query patterns.
   */
  async detectInstantAnswer(query) {
    try {
      if (CALC_PATTERN.test(query) || FUNC_PATTERN.test(query)) {
        const result = this.instantService.calculate(query);
        return {
          type: "calculator",
          query,
          result: result.formatted,
          data: result
        };
      }
      if (UNIT_PATTERN.test(query)) {
        const result = this.instantService.convert(query);
        return {
          type: "unit_conversion",
          query,
          result: `${result.from_value} ${result.from_unit} = ${result.to_value} ${result.to_unit}`,
          data: result
        };
      }
      if (CURRENCY_PATTERN.test(query)) {
        const result = await this.instantService.currency(query);
        return {
          type: "currency",
          query,
          result: `${result.from_amount} ${result.from_currency} = ${result.to_amount.toFixed(2)} ${result.to_currency}`,
          data: result
        };
      }
      const weatherMatch = query.match(WEATHER_PATTERN);
      if (weatherMatch) {
        const location = weatherMatch[2].trim();
        const result = await this.instantService.weather(location);
        return {
          type: "weather",
          query,
          result: `${result.temperature}${result.unit} ${result.condition} in ${result.location}`,
          data: result
        };
      }
      const defineMatch = query.match(DEFINE_PATTERN);
      if (defineMatch) {
        const word = defineMatch[1].trim();
        const result = await this.instantService.define(word);
        return {
          type: "definition",
          query,
          result: result.definitions[0] ?? "",
          data: result
        };
      }
      const timeMatch = query.match(TIME_PATTERN);
      if (timeMatch) {
        const location = timeMatch[1].trim();
        const result = this.instantService.time(location);
        return {
          type: "time",
          query,
          result: `${result.time} in ${result.location}`,
          data: result
        };
      }
      return null;
    } catch {
      return null;
    }
  }
  /**
   * Add a search query to history.
   */
  async addToHistory(query, totalResults) {
    const entry = {
      id: generateId(),
      query,
      results: totalResults,
      searched_at: (/* @__PURE__ */ new Date()).toISOString()
    };
    await this.kvStore.addHistory(entry);
  }
  /**
   * Build an empty search response.
   */
  emptyResponse(query, options, timeMs) {
    return {
      query,
      total_results: 0,
      results: [],
      search_time_ms: timeMs,
      page: options.page,
      per_page: options.per_page,
      has_more: false
    };
  }
};

// src/services/bang.ts
var BUILTIN_BANGS = [
  { trigger: "g", name: "Google", url_template: "https://www.google.com/search?q={query}", category: "search", is_builtin: true },
  { trigger: "ddg", name: "DuckDuckGo", url_template: "https://duckduckgo.com/?q={query}", category: "search", is_builtin: true },
  { trigger: "b", name: "Bing", url_template: "https://www.bing.com/search?q={query}", category: "search", is_builtin: true },
  { trigger: "yt", name: "YouTube", url_template: "https://www.youtube.com/results?search_query={query}", category: "video", is_builtin: true },
  { trigger: "w", name: "Wikipedia", url_template: "https://en.wikipedia.org/wiki/Special:Search?search={query}", category: "reference", is_builtin: true },
  { trigger: "r", name: "Reddit", url_template: "https://www.reddit.com/search/?q={query}", category: "social", is_builtin: true },
  { trigger: "gh", name: "GitHub", url_template: "https://github.com/search?q={query}", category: "code", is_builtin: true },
  { trigger: "so", name: "Stack Overflow", url_template: "https://stackoverflow.com/search?q={query}", category: "code", is_builtin: true },
  { trigger: "npm", name: "npm", url_template: "https://www.npmjs.com/search?q={query}", category: "code", is_builtin: true },
  { trigger: "amz", name: "Amazon", url_template: "https://www.amazon.com/s?k={query}", category: "shopping", is_builtin: true },
  { trigger: "imdb", name: "IMDb", url_template: "https://www.imdb.com/find?q={query}", category: "media", is_builtin: true },
  { trigger: "mdn", name: "MDN", url_template: "https://developer.mozilla.org/en-US/search?q={query}", category: "code", is_builtin: true },
  { trigger: "i", name: "Images", url_template: "/images?q={query}", category: "internal", is_builtin: true },
  { trigger: "n", name: "News", url_template: "/news?q={query}", category: "internal", is_builtin: true },
  { trigger: "v", name: "Videos", url_template: "/videos?q={query}", category: "internal", is_builtin: true }
];
var BUILTIN_MAP = /* @__PURE__ */ new Map();
for (const bang of BUILTIN_BANGS) {
  BUILTIN_MAP.set(bang.trigger, bang);
}
var BangService = class {
  static {
    __name(this, "BangService");
  }
  kvStore;
  constructor(kvStore) {
    this.kvStore = kvStore;
  }
  /**
   * Parse a query for bang commands.
   * Bangs can appear at the start (!g query) or end (query !g) of the query.
   * Returns a redirect URL for external bangs, a category for internal bangs,
   * or the cleaned query if no bang matches.
   */
  async parse(query) {
    const trimmed = query.trim();
    if (!trimmed) {
      return { query: trimmed };
    }
    let trigger = null;
    let cleanQuery;
    const startMatch = trimmed.match(/^!(\S+)\s*(.*)/);
    if (startMatch) {
      trigger = startMatch[1].toLowerCase();
      cleanQuery = startMatch[2].trim();
    } else {
      const endMatch = trimmed.match(/(.*)\s+!(\S+)$/);
      if (endMatch) {
        trigger = endMatch[2].toLowerCase();
        cleanQuery = endMatch[1].trim();
      } else {
        return { query: trimmed };
      }
    }
    let bangData = BUILTIN_MAP.get(trigger);
    if (!bangData) {
      const customBang = await this.kvStore.getBang(trigger);
      if (customBang) {
        bangData = customBang;
      }
    }
    if (!bangData) {
      return { query: trimmed };
    }
    const encodedQuery = encodeURIComponent(cleanQuery || "");
    const url = bangData.url_template.replace("{query}", encodedQuery);
    if (url.startsWith("/")) {
      return {
        query: cleanQuery,
        bang: { name: bangData.name, trigger: bangData.trigger },
        category: bangData.category,
        redirect: url
      };
    }
    return {
      redirect: url,
      bang: { name: bangData.name, trigger: bangData.trigger },
      query: cleanQuery
    };
  }
  /**
   * List all bangs: built-in bangs combined with custom bangs from KV.
   */
  async listBangs() {
    const now = (/* @__PURE__ */ new Date()).toISOString();
    const builtins = BUILTIN_BANGS.map((b, idx) => ({
      ...b,
      id: idx + 1,
      created_at: now
    }));
    const customBangs = await this.kvStore.listBangs();
    return [...builtins, ...customBangs];
  }
  /**
   * Create a custom bang and persist to KV.
   */
  async createBang(bang) {
    if (BUILTIN_MAP.has(bang.trigger)) {
      throw new Error(`Cannot override built-in bang: !${bang.trigger}`);
    }
    await this.kvStore.createBang(bang);
  }
  /**
   * Delete a custom bang from KV by trigger.
   */
  async deleteBang(trigger) {
    if (BUILTIN_MAP.has(trigger)) {
      throw new Error(`Cannot delete built-in bang: !${trigger}`);
    }
    await this.kvStore.deleteBang(trigger);
  }
};

// src/services/instant.ts
var CONSTANTS = {
  pi: Math.PI,
  e: Math.E
};
var FUNCTIONS = {
  sqrt: /* @__PURE__ */ __name((a) => Math.sqrt(a[0]), "sqrt"),
  sin: /* @__PURE__ */ __name((a) => Math.sin(a[0]), "sin"),
  cos: /* @__PURE__ */ __name((a) => Math.cos(a[0]), "cos"),
  tan: /* @__PURE__ */ __name((a) => Math.tan(a[0]), "tan"),
  log: /* @__PURE__ */ __name((a) => Math.log10(a[0]), "log"),
  ln: /* @__PURE__ */ __name((a) => Math.log(a[0]), "ln"),
  abs: /* @__PURE__ */ __name((a) => Math.abs(a[0]), "abs"),
  ceil: /* @__PURE__ */ __name((a) => Math.ceil(a[0]), "ceil"),
  floor: /* @__PURE__ */ __name((a) => Math.floor(a[0]), "floor"),
  round: /* @__PURE__ */ __name((a) => Math.round(a[0]), "round"),
  pow: /* @__PURE__ */ __name((a) => Math.pow(a[0], a[1]), "pow"),
  min: /* @__PURE__ */ __name((a) => Math.min(...a), "min"),
  max: /* @__PURE__ */ __name((a) => Math.max(...a), "max")
};
function tokenize(expr) {
  const tokens = [];
  let i = 0;
  const s = expr.replace(/\s+/g, "");
  while (i < s.length) {
    const ch = s[i];
    if (/\d/.test(ch) || ch === "." && i + 1 < s.length && /\d/.test(s[i + 1])) {
      let num = "";
      while (i < s.length && (/\d/.test(s[i]) || s[i] === ".")) {
        num += s[i];
        i++;
      }
      tokens.push({ type: "number", value: parseFloat(num) });
      continue;
    }
    if (/[a-zA-Z_]/.test(ch)) {
      let name = "";
      while (i < s.length && /[a-zA-Z_0-9]/.test(s[i])) {
        name += s[i];
        i++;
      }
      const lower = name.toLowerCase();
      if (CONSTANTS[lower] !== void 0) {
        tokens.push({ type: "number", value: CONSTANTS[lower] });
      } else if (FUNCTIONS[lower]) {
        tokens.push({ type: "func", value: lower });
      } else {
        throw new Error(`Unknown identifier: ${name}`);
      }
      continue;
    }
    if (ch === "(") {
      tokens.push({ type: "lparen" });
      i++;
      continue;
    }
    if (ch === ")") {
      tokens.push({ type: "rparen" });
      i++;
      continue;
    }
    if (ch === ",") {
      tokens.push({ type: "comma" });
      i++;
      continue;
    }
    if ("+-*/%^".includes(ch)) {
      tokens.push({ type: "op", value: ch });
      i++;
      continue;
    }
    throw new Error(`Unexpected character: ${ch}`);
  }
  return tokens;
}
__name(tokenize, "tokenize");
var Parser = class {
  static {
    __name(this, "Parser");
  }
  tokens;
  pos;
  constructor(tokens) {
    this.tokens = tokens;
    this.pos = 0;
  }
  parse() {
    const result = this.expr();
    if (this.pos < this.tokens.length) {
      throw new Error("Unexpected token at end of expression");
    }
    return result;
  }
  peek() {
    return this.tokens[this.pos];
  }
  consume() {
    const tok = this.tokens[this.pos];
    this.pos++;
    return tok;
  }
  isOpToken(tok, ops) {
    return tok?.type === "op" && "value" in tok && ops.includes(tok.value);
  }
  expr() {
    let left = this.term();
    while (this.isOpToken(this.peek(), ["+", "-"])) {
      const tok = this.consume();
      const right = this.term();
      left = tok.value === "+" ? left + right : left - right;
    }
    return left;
  }
  term() {
    let left = this.exponent();
    while (this.isOpToken(this.peek(), ["*", "/", "%"])) {
      const tok = this.consume();
      const right = this.exponent();
      if (tok.value === "*") left = left * right;
      else if (tok.value === "/") {
        if (right === 0) throw new Error("Division by zero");
        left = left / right;
      } else left = left % right;
    }
    return left;
  }
  exponent() {
    let base = this.unary();
    while (this.isOpToken(this.peek(), ["^"])) {
      this.consume();
      const exp = this.unary();
      base = Math.pow(base, exp);
    }
    return base;
  }
  unary() {
    if (this.isOpToken(this.peek(), ["-"])) {
      this.consume();
      return -this.unary();
    }
    if (this.isOpToken(this.peek(), ["+"])) {
      this.consume();
      return this.unary();
    }
    return this.primary();
  }
  primary() {
    const tok = this.peek();
    if (!tok) {
      throw new Error("Unexpected end of expression");
    }
    if (tok.type === "number") {
      this.consume();
      return tok.value;
    }
    if (tok.type === "func") {
      const funcName = tok.value;
      this.consume();
      const lparen = this.peek();
      if (lparen?.type !== "lparen") {
        throw new Error(`Expected '(' after function ${funcName}`);
      }
      this.consume();
      const args = this.parseArgs();
      const rparen = this.peek();
      if (rparen?.type !== "rparen") {
        throw new Error(`Expected ')' after function arguments`);
      }
      this.consume();
      const fn = FUNCTIONS[funcName];
      if (!fn) throw new Error(`Unknown function: ${funcName}`);
      return fn(args);
    }
    if (tok.type === "lparen") {
      this.consume();
      const val = this.expr();
      const rparen = this.peek();
      if (rparen?.type !== "rparen") {
        throw new Error(`Expected ')'`);
      }
      this.consume();
      return val;
    }
    throw new Error(`Unexpected token: ${JSON.stringify(tok)}`);
  }
  parseArgs() {
    const args = [];
    if (this.peek()?.type === "rparen") {
      return args;
    }
    args.push(this.expr());
    while (this.peek()?.type === "comma") {
      this.consume();
      args.push(this.expr());
    }
    return args;
  }
};
function formatNumber(n) {
  if (!isFinite(n)) return String(n);
  const abs = Math.abs(n);
  if (abs >= 1e15 || abs < 1e-6 && abs > 0) {
    return n.toExponential(6);
  }
  const fixed = n.toFixed(10).replace(/\.?0+$/, "");
  const parts = fixed.split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}
__name(formatNumber, "formatNumber");
function linearUnit(category, factor) {
  return {
    category,
    toBase: /* @__PURE__ */ __name((v) => v * factor, "toBase"),
    fromBase: /* @__PURE__ */ __name((v) => v / factor, "fromBase")
  };
}
__name(linearUnit, "linearUnit");
var UNITS = {
  // Length (base: meters)
  mm: linearUnit("length", 1e-3),
  cm: linearUnit("length", 0.01),
  m: linearUnit("length", 1),
  km: linearUnit("length", 1e3),
  in: linearUnit("length", 0.0254),
  ft: linearUnit("length", 0.3048),
  yd: linearUnit("length", 0.9144),
  mi: linearUnit("length", 1609.344),
  // Weight (base: grams)
  mg: linearUnit("weight", 1e-3),
  g: linearUnit("weight", 1),
  kg: linearUnit("weight", 1e3),
  lb: linearUnit("weight", 453.592),
  oz: linearUnit("weight", 28.3495),
  ton: linearUnit("weight", 907185),
  // Temperature (base: celsius)
  c: {
    category: "temperature",
    toBase: /* @__PURE__ */ __name((v) => v, "toBase"),
    fromBase: /* @__PURE__ */ __name((v) => v, "fromBase")
  },
  f: {
    category: "temperature",
    toBase: /* @__PURE__ */ __name((v) => (v - 32) * (5 / 9), "toBase"),
    fromBase: /* @__PURE__ */ __name((v) => v * (9 / 5) + 32, "fromBase")
  },
  k: {
    category: "temperature",
    toBase: /* @__PURE__ */ __name((v) => v - 273.15, "toBase"),
    fromBase: /* @__PURE__ */ __name((v) => v + 273.15, "fromBase")
  },
  // Volume (base: milliliters)
  ml: linearUnit("volume", 1),
  l: linearUnit("volume", 1e3),
  gal: linearUnit("volume", 3785.41),
  qt: linearUnit("volume", 946.353),
  pt: linearUnit("volume", 473.176),
  cup: linearUnit("volume", 236.588),
  tbsp: linearUnit("volume", 14.7868),
  tsp: linearUnit("volume", 4.92892),
  fl_oz: linearUnit("volume", 29.5735),
  // Area (base: square meters)
  mm2: linearUnit("area", 1e-6),
  cm2: linearUnit("area", 1e-4),
  m2: linearUnit("area", 1),
  km2: linearUnit("area", 1e6),
  in2: linearUnit("area", 64516e-8),
  ft2: linearUnit("area", 0.092903),
  acre: linearUnit("area", 4046.86),
  hectare: linearUnit("area", 1e4),
  // Speed (base: m/s)
  "m/s": linearUnit("speed", 1),
  "km/h": linearUnit("speed", 1 / 3.6),
  mph: linearUnit("speed", 0.44704),
  knots: linearUnit("speed", 0.514444),
  // Data (base: bytes)
  b: linearUnit("data", 1),
  kb: linearUnit("data", 1024),
  mb: linearUnit("data", 1024 ** 2),
  gb: linearUnit("data", 1024 ** 3),
  tb: linearUnit("data", 1024 ** 4),
  pb: linearUnit("data", 1024 ** 5),
  // Time (base: seconds)
  ms: linearUnit("time", 1e-3),
  s: linearUnit("time", 1),
  min: linearUnit("time", 60),
  hr: linearUnit("time", 3600),
  day: linearUnit("time", 86400),
  week: linearUnit("time", 604800),
  month: linearUnit("time", 2592e3),
  year: linearUnit("time", 31536e3)
};
var TIMEZONE_MAP = {
  "new york": "America/New_York",
  "los angeles": "America/Los_Angeles",
  "chicago": "America/Chicago",
  "denver": "America/Denver",
  "london": "Europe/London",
  "paris": "Europe/Paris",
  "berlin": "Europe/Berlin",
  "tokyo": "Asia/Tokyo",
  "sydney": "Australia/Sydney",
  "moscow": "Europe/Moscow",
  "dubai": "Asia/Dubai",
  "mumbai": "Asia/Kolkata",
  "delhi": "Asia/Kolkata",
  "shanghai": "Asia/Shanghai",
  "beijing": "Asia/Shanghai",
  "hong kong": "Asia/Hong_Kong",
  "singapore": "Asia/Singapore",
  "seoul": "Asia/Seoul",
  "toronto": "America/Toronto",
  "vancouver": "America/Vancouver",
  "sao paulo": "America/Sao_Paulo",
  "mexico city": "America/Mexico_City",
  "cairo": "Africa/Cairo",
  "johannesburg": "Africa/Johannesburg",
  "istanbul": "Europe/Istanbul",
  "bangkok": "Asia/Bangkok",
  "jakarta": "Asia/Jakarta",
  "amsterdam": "Europe/Amsterdam",
  "rome": "Europe/Rome",
  "madrid": "Europe/Madrid",
  "lisbon": "Europe/Lisbon",
  "zurich": "Europe/Zurich",
  "vienna": "Europe/Vienna",
  "warsaw": "Europe/Warsaw",
  "athens": "Europe/Athens",
  "helsinki": "Europe/Helsinki",
  "stockholm": "Europe/Stockholm",
  "oslo": "Europe/Oslo",
  "copenhagen": "Europe/Copenhagen",
  "honolulu": "Pacific/Honolulu",
  "anchorage": "America/Anchorage",
  "phoenix": "America/Phoenix",
  "auckland": "Pacific/Auckland",
  "perth": "Australia/Perth",
  "melbourne": "Australia/Melbourne",
  "brisbane": "Australia/Brisbane",
  "est": "America/New_York",
  "cst": "America/Chicago",
  "mst": "America/Denver",
  "pst": "America/Los_Angeles",
  "gmt": "Europe/London",
  "utc": "UTC",
  "cet": "Europe/Paris",
  "jst": "Asia/Tokyo",
  "ist": "Asia/Kolkata",
  "aest": "Australia/Sydney"
};
var SUPPORTED_CURRENCIES = /* @__PURE__ */ new Set([
  "usd",
  "eur",
  "gbp",
  "jpy",
  "cad",
  "aud",
  "chf",
  "cny",
  "inr",
  "krw",
  "brl",
  "mxn",
  "sgd",
  "hkd",
  "nzd",
  "sek",
  "nok",
  "dkk",
  "pln",
  "zar",
  "try",
  "thb",
  "idr",
  "php",
  "czk",
  "ils",
  "clp",
  "myr",
  "twd",
  "ars",
  "cop",
  "sar",
  "aed",
  "egp",
  "vnd",
  "bgn",
  "hrk",
  "huf",
  "isk",
  "ron",
  "rub"
]);
var WEATHER_ICONS = {
  "sunny": "sun",
  "clear": "sun",
  "partly cloudy": "cloud-sun",
  "cloudy": "cloud",
  "overcast": "cloud",
  "mist": "smog",
  "fog": "smog",
  "patchy rain possible": "cloud-rain",
  "patchy rain nearby": "cloud-rain",
  "light rain": "cloud-rain",
  "moderate rain": "cloud-showers-heavy",
  "heavy rain": "cloud-showers-heavy",
  "light snow": "snowflake",
  "moderate snow": "snowflake",
  "heavy snow": "snowflake",
  "thunderstorm": "bolt",
  "blizzard": "snowflake"
};
function weatherIcon(condition) {
  const lower = condition.toLowerCase();
  for (const [key, icon] of Object.entries(WEATHER_ICONS)) {
    if (lower.includes(key)) return icon;
  }
  return "cloud";
}
__name(weatherIcon, "weatherIcon");
var InstantService = class {
  static {
    __name(this, "InstantService");
  }
  cache;
  currencyRates;
  constructor(cache) {
    this.cache = cache;
    this.currencyRates = /* @__PURE__ */ new Map();
  }
  calculate(expr) {
    const tokens = tokenize(expr);
    const parser = new Parser(tokens);
    const result = parser.parse();
    return {
      expression: expr,
      result,
      formatted: formatNumber(result)
    };
  }
  convert(expr) {
    const match2 = expr.match(/^([\d.]+)\s*([a-zA-Z/_2]+)\s+(?:to|in)\s+([a-zA-Z/_2]+)$/i);
    if (!match2) {
      throw new Error("Invalid conversion format. Use: <number> <unit> to <unit>");
    }
    const value = parseFloat(match2[1]);
    const fromUnitKey = match2[2].toLowerCase();
    const toUnitKey = match2[3].toLowerCase();
    const fromUnit = UNITS[fromUnitKey];
    const toUnit = UNITS[toUnitKey];
    if (!fromUnit) throw new Error(`Unknown unit: ${match2[2]}`);
    if (!toUnit) throw new Error(`Unknown unit: ${match2[3]}`);
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
      category: fromUnit.category
    };
  }
  async currency(expr) {
    const match2 = expr.match(/^([\d.]+)\s*([a-zA-Z]{3})\s+(?:to|in)\s+([a-zA-Z]{3})$/i);
    if (!match2) {
      throw new Error("Invalid currency format. Use: <number> <currency> to <currency>");
    }
    const value = parseFloat(match2[1]);
    const fromCurrency = match2[2].toUpperCase();
    const toCurrency = match2[3].toUpperCase();
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
      updated_at: (/* @__PURE__ */ new Date()).toISOString()
    };
  }
  async fetchRate(from, to) {
    const cacheKey = `${from}_${to}`;
    const cached = this.currencyRates.get(cacheKey);
    const now = Date.now();
    if (cached && now - cached.fetched_at < 36e5) {
      return cached.rates[to] ?? 1;
    }
    const kvCached = await this.cache.getInstant(`currency:${cacheKey}`);
    if (kvCached && kvCached.data) {
      const rateData = kvCached.data;
      const rate2 = rateData.rates[to];
      if (rate2 !== void 0) {
        this.currencyRates.set(cacheKey, { rates: rateData.rates, fetched_at: now });
        return rate2;
      }
    }
    const url = `https://api.frankfurter.app/latest?from=${from}&to=${to}`;
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Currency API error: ${response.status}`);
    }
    const data = await response.json();
    this.currencyRates.set(cacheKey, { rates: data.rates, fetched_at: now });
    await this.cache.setInstant(`currency:${cacheKey}`, {
      type: "currency_rates",
      query: cacheKey,
      result: JSON.stringify(data.rates),
      data: { rates: data.rates }
    });
    const rate = data.rates[to];
    if (rate === void 0) {
      throw new Error(`No rate found for ${from} to ${to}`);
    }
    return rate;
  }
  async weather(location) {
    const encoded = encodeURIComponent(location.trim());
    const url = `https://wttr.in/${encoded}?format=j1`;
    const response = await fetch(url, {
      headers: { "User-Agent": "mizu-search/1.0" }
    });
    if (!response.ok) {
      throw new Error(`Weather API error: ${response.status}`);
    }
    const data = await response.json();
    const current = data.current_condition[0];
    const area = data.nearest_area?.[0];
    const areaName = area?.areaName?.[0]?.value ?? location;
    const country = area?.country?.[0]?.value ?? "";
    const displayLocation = country ? `${areaName}, ${country}` : areaName;
    const condition = current.weatherDesc[0]?.value ?? "Unknown";
    return {
      location: displayLocation,
      temperature: parseInt(current.temp_C, 10),
      unit: "C",
      condition,
      humidity: parseInt(current.humidity, 10),
      wind_speed: parseInt(current.windspeedKmph, 10),
      wind_unit: "km/h",
      icon: weatherIcon(condition)
    };
  }
  async define(word) {
    const encoded = encodeURIComponent(word.trim().toLowerCase());
    const url = `https://api.dictionaryapi.dev/api/v2/entries/en/${encoded}`;
    const response = await fetch(url);
    if (!response.ok) {
      throw new Error(`Dictionary API error: ${response.status}`);
    }
    const data = await response.json();
    if (!data.length) {
      throw new Error(`No definition found for: ${word}`);
    }
    const entry = data[0];
    const firstMeaning = entry.meanings[0];
    const phonetic = entry.phonetic ?? entry.phonetics?.find((p) => p.text)?.text ?? void 0;
    const examples = [];
    for (const def of firstMeaning?.definitions ?? []) {
      if (def.example) {
        examples.push(def.example);
      }
    }
    const allAntonyms = [];
    for (const meaning of entry.meanings) {
      allAntonyms.push(...meaning.antonyms);
    }
    return {
      word: entry.word,
      phonetic,
      part_of_speech: firstMeaning?.partOfSpeech ?? "unknown",
      definitions: firstMeaning?.definitions.map((d) => d.definition).slice(0, 5) ?? [],
      synonyms: firstMeaning?.synonyms?.slice(0, 10),
      antonyms: [...new Set(allAntonyms)].slice(0, 10),
      examples: examples.slice(0, 3)
    };
  }
  time(location) {
    const normalized = location.trim().toLowerCase();
    const timezone = TIMEZONE_MAP[normalized];
    if (!timezone) {
      try {
        new Intl.DateTimeFormat("en-US", { timeZone: location.trim() });
        return this.formatTime(location.trim(), location.trim());
      } catch {
        throw new Error(`Unknown location or timezone: ${location}`);
      }
    }
    return this.formatTime(location.trim(), timezone);
  }
  formatTime(location, timezone) {
    const now = /* @__PURE__ */ new Date();
    const timeFormatter = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: true
    });
    const dateFormatter = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      weekday: "long",
      year: "numeric",
      month: "long",
      day: "numeric"
    });
    const offsetFormatter = new Intl.DateTimeFormat("en-US", {
      timeZone: timezone,
      timeZoneName: "longOffset"
    });
    const offsetParts = offsetFormatter.formatToParts(now);
    const offsetPart = offsetParts.find((p) => p.type === "timeZoneName");
    const offset = offsetPart?.value ?? timezone;
    return {
      location: location.charAt(0).toUpperCase() + location.slice(1),
      time: timeFormatter.format(now),
      date: dateFormatter.format(now),
      timezone,
      offset
    };
  }
};

// src/services/knowledge.ts
var WIKIDATA_PROPERTIES = {
  P569: "Born",
  P570: "Died",
  P19: "Place of birth",
  P20: "Place of death",
  P27: "Nationality",
  P106: "Occupation",
  P1412: "Languages spoken",
  P17: "Country",
  P36: "Capital",
  P1082: "Population",
  P571: "Founded",
  P112: "Founded by",
  P159: "Headquarters",
  P452: "Industry",
  P856: "Official website",
  P1448: "Official name",
  P18: "Image"
};
var KnowledgeService = class {
  static {
    __name(this, "KnowledgeService");
  }
  cache;
  constructor(cache) {
    this.cache = cache;
  }
  /**
   * Get a knowledge panel for a query.
   * Checks cache first, then tries Wikipedia and Wikidata APIs.
   */
  async getPanel(query) {
    const normalizedQuery = query.trim().toLowerCase();
    if (!normalizedQuery) {
      return null;
    }
    const cached = await this.cache.getKnowledge(normalizedQuery);
    if (cached) {
      return cached;
    }
    const panel = await this.fetchWikipediaPanel(normalizedQuery);
    if (!panel) {
      return null;
    }
    const enrichedPanel = await this.enrichWithWikidata(panel, normalizedQuery);
    await this.cache.setKnowledge(normalizedQuery, enrichedPanel);
    return enrichedPanel;
  }
  async fetchWikipediaPanel(query) {
    const encoded = encodeURIComponent(query.replace(/\s+/g, "_"));
    const url = `https://en.wikipedia.org/api/rest_v1/page/summary/${encoded}`;
    try {
      const response = await fetch(url, {
        headers: {
          "User-Agent": "mizu-search/1.0 (search engine)",
          "Accept": "application/json"
        }
      });
      if (!response.ok) {
        return this.searchWikipedia(query);
      }
      const data = await response.json();
      if (data.type === "disambiguation" || data.type === "no-extract") {
        return this.searchWikipedia(query);
      }
      if (!data.extract || data.extract.length < 20) {
        return null;
      }
      const links = [];
      if (data.content_urls?.desktop?.page) {
        links.push({
          title: "Wikipedia",
          url: data.content_urls.desktop.page,
          icon: "wikipedia"
        });
      }
      return {
        title: data.title,
        subtitle: data.description,
        description: data.extract,
        image: data.thumbnail?.source,
        facts: [],
        links,
        source: "Wikipedia"
      };
    } catch {
      return null;
    }
  }
  /**
   * Fallback: search Wikipedia and use the first result.
   */
  async searchWikipedia(query) {
    const encoded = encodeURIComponent(query);
    const url = `https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=${encoded}&format=json&utf8=1&srlimit=1&srprop=snippet`;
    try {
      const response = await fetch(url, {
        headers: {
          "User-Agent": "mizu-search/1.0 (search engine)",
          "Accept": "application/json"
        }
      });
      if (!response.ok) return null;
      const data = await response.json();
      const results = data.query?.search;
      if (!results || results.length === 0) return null;
      const firstResult = results[0];
      const titleEncoded = encodeURIComponent(firstResult.title.replace(/\s+/g, "_"));
      const summaryUrl = `https://en.wikipedia.org/api/rest_v1/page/summary/${titleEncoded}`;
      const summaryResponse = await fetch(summaryUrl, {
        headers: {
          "User-Agent": "mizu-search/1.0 (search engine)",
          "Accept": "application/json"
        }
      });
      if (!summaryResponse.ok) return null;
      const summaryData = await summaryResponse.json();
      if (!summaryData.extract || summaryData.extract.length < 20) {
        return null;
      }
      const links = [];
      if (summaryData.content_urls?.desktop?.page) {
        links.push({
          title: "Wikipedia",
          url: summaryData.content_urls.desktop.page,
          icon: "wikipedia"
        });
      }
      return {
        title: summaryData.title,
        subtitle: summaryData.description,
        description: summaryData.extract,
        image: summaryData.thumbnail?.source,
        facts: [],
        links,
        source: "Wikipedia"
      };
    } catch {
      return null;
    }
  }
  /**
   * Enrich a knowledge panel with Wikidata structured facts.
   */
  async enrichWithWikidata(panel, query) {
    try {
      const searchEncoded = encodeURIComponent(query);
      const searchUrl = `https://www.wikidata.org/w/api.php?action=wbsearchentities&search=${searchEncoded}&language=en&format=json&limit=1`;
      const searchResponse = await fetch(searchUrl, {
        headers: {
          "User-Agent": "mizu-search/1.0 (search engine)",
          "Accept": "application/json"
        }
      });
      if (!searchResponse.ok) return panel;
      const searchData = await searchResponse.json();
      if (!searchData.search || searchData.search.length === 0) {
        return panel;
      }
      const entityId = searchData.search[0].id;
      const entityUrl = `https://www.wikidata.org/w/api.php?action=wbgetentities&ids=${entityId}&languages=en&format=json&props=claims|labels|descriptions|sitelinks`;
      const entityResponse = await fetch(entityUrl, {
        headers: {
          "User-Agent": "mizu-search/1.0 (search engine)",
          "Accept": "application/json"
        }
      });
      if (!entityResponse.ok) return panel;
      const entityData = await entityResponse.json();
      const entity = entityData.entities[entityId];
      if (!entity || !entity.claims) return panel;
      const facts = [];
      for (const [propId, label] of Object.entries(WIKIDATA_PROPERTIES)) {
        const claims = entity.claims[propId];
        if (!claims || claims.length === 0) continue;
        const claim = claims[0];
        const value = this.extractClaimValue(claim);
        if (value) {
          facts.push({ label, value });
        }
      }
      const links = [...panel.links ?? []];
      links.push({
        title: "Wikidata",
        url: `https://www.wikidata.org/wiki/${entityId}`,
        icon: "wikidata"
      });
      return {
        ...panel,
        facts: facts.length > 0 ? facts : panel.facts,
        links
      };
    } catch {
      return panel;
    }
  }
  /**
   * Extract a human-readable value from a Wikidata claim.
   */
  extractClaimValue(claim) {
    const datavalue = claim.mainsnak?.datavalue;
    if (!datavalue) return null;
    switch (datavalue.type) {
      case "string":
        return typeof datavalue.value === "string" ? datavalue.value : null;
      case "monolingualtext":
        if (typeof datavalue.value === "object" && datavalue.value?.text) {
          return datavalue.value.text;
        }
        return null;
      case "quantity":
        if (typeof datavalue.value === "object" && datavalue.value?.amount) {
          const amount = datavalue.value.amount.replace(/^\+/, "");
          const num = parseFloat(amount);
          if (!isNaN(num)) {
            return num.toLocaleString("en-US");
          }
          return amount;
        }
        return null;
      case "time":
        if (typeof datavalue.value === "object" && datavalue.value?.time) {
          const timeStr = datavalue.value.time;
          const match2 = timeStr.match(/^\+?(-?\d{4})-(\d{2})-(\d{2})/);
          if (match2) {
            const year = parseInt(match2[1], 10);
            const month = parseInt(match2[2], 10);
            const day = parseInt(match2[3], 10);
            if (month > 0 && day > 0) {
              const date = new Date(year, month - 1, day);
              return date.toLocaleDateString("en-US", {
                year: "numeric",
                month: "long",
                day: "numeric"
              });
            }
            return String(year);
          }
          return null;
        }
        return null;
      case "wikibase-entityid":
        if (typeof datavalue.value === "object" && datavalue.value?.id) {
          return datavalue.value.id;
        }
        return null;
      default:
        if (typeof datavalue.value === "string") {
          return datavalue.value;
        }
        return null;
    }
  }
};

// src/routes/search.ts
function extractSearchOptions(c) {
  return {
    page: parseInt(c.req.query("page") ?? "1", 10),
    per_page: parseInt(c.req.query("per_page") ?? "10", 10),
    time_range: c.req.query("time") ?? "",
    region: c.req.query("region") ?? "",
    language: c.req.query("lang") ?? "en",
    safe_search: c.req.query("safe") ?? "moderate"
  };
}
__name(extractSearchOptions, "extractSearchOptions");
function createServices(kv) {
  const cache = new CacheStore(kv);
  const kvStore = new KVStore(kv);
  const metaSearch = createDefaultMetaSearch();
  const bangService = new BangService(kvStore);
  const instantService = new InstantService(cache);
  const knowledgeService = new KnowledgeService(cache);
  const searchService = new SearchService(metaSearch, cache, kvStore, bangService, instantService, knowledgeService);
  return { searchService };
}
__name(createServices, "createServices");
var app2 = new Hono2();
app2.get("/", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const options = extractSearchOptions(c);
  const { searchService } = createServices(c.env.SEARCH_KV);
  const results = await searchService.search(q, options);
  return c.json(results);
});
app2.get("/images", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const options = extractSearchOptions(c);
  const { searchService } = createServices(c.env.SEARCH_KV);
  const results = await searchService.searchImages(q, options);
  return c.json(results);
});
app2.get("/videos", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const options = extractSearchOptions(c);
  const { searchService } = createServices(c.env.SEARCH_KV);
  const results = await searchService.searchVideos(q, options);
  return c.json(results);
});
app2.get("/news", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const options = extractSearchOptions(c);
  const { searchService } = createServices(c.env.SEARCH_KV);
  const results = await searchService.searchNews(q, options);
  return c.json(results);
});
var search_default = app2;

// src/services/suggest.ts
var TRENDING_SUGGESTIONS = [
  { text: "artificial intelligence news", type: "trending", frequency: 95 },
  { text: "climate change solutions", type: "trending", frequency: 88 },
  { text: "programming tutorials", type: "trending", frequency: 85 },
  { text: "space exploration updates", type: "trending", frequency: 82 },
  { text: "healthy recipes", type: "trending", frequency: 78 },
  { text: "web development frameworks", type: "trending", frequency: 76 },
  { text: "machine learning projects", type: "trending", frequency: 74 },
  { text: "renewable energy technology", type: "trending", frequency: 72 },
  { text: "open source software", type: "trending", frequency: 70 },
  { text: "cybersecurity best practices", type: "trending", frequency: 68 }
];
var SuggestService = class {
  static {
    __name(this, "SuggestService");
  }
  cache;
  constructor(cache) {
    this.cache = cache;
  }
  /**
   * Get search suggestions for a query.
   * Checks cache first, then fetches from Google suggestions API.
   */
  async suggest(query) {
    const trimmed = query.trim();
    if (!trimmed) {
      return [];
    }
    const hash = this.hashQuery(trimmed);
    const cached = await this.cache.getSuggest(hash);
    if (cached) {
      return cached;
    }
    const encoded = encodeURIComponent(trimmed);
    const url = `https://suggestqueries.google.com/complete/search?client=firefox&q=${encoded}`;
    try {
      const response = await fetch(url, {
        headers: {
          "User-Agent": "Mozilla/5.0 (compatible; mizu-search/1.0)"
        }
      });
      if (!response.ok) {
        return [];
      }
      const data = await response.json();
      if (!Array.isArray(data) || !Array.isArray(data[1])) {
        return [];
      }
      const suggestions = data[1].map((text) => ({
        text,
        type: "query"
      }));
      await this.cache.setSuggest(hash, suggestions);
      return suggestions;
    } catch {
      return [];
    }
  }
  /**
   * Return a list of trending search suggestions.
   */
  async trending() {
    return [...TRENDING_SUGGESTIONS];
  }
  /**
   * Simple hash function for cache keys.
   * Uses a basic string hash suitable for KV cache keys.
   */
  hashQuery(query) {
    let hash = 0;
    for (let i = 0; i < query.length; i++) {
      const char = query.charCodeAt(i);
      hash = (hash << 5) - hash + char | 0;
    }
    return Math.abs(hash).toString(36);
  }
};

// src/routes/suggest.ts
var app3 = new Hono2();
app3.get("/", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const suggestService = new SuggestService(cache);
  const suggestions = await suggestService.suggest(q);
  return c.json(suggestions);
});
app3.get("/trending", async (c) => {
  const cache = new CacheStore(c.env.SEARCH_KV);
  const suggestService = new SuggestService(cache);
  const trending = await suggestService.trending();
  return c.json(trending);
});
var suggest_default = app3;

// src/routes/instant.ts
var app4 = new Hono2();
app4.get("/calculate", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = instantService.calculate(q);
  return c.json({
    type: "calculation",
    query: q,
    answer: result
  });
});
app4.get("/convert", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = instantService.convert(q);
  return c.json({
    type: "conversion",
    query: q,
    answer: result
  });
});
app4.get("/currency", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = await instantService.currency(q);
  return c.json({
    type: "currency",
    query: q,
    answer: result
  });
});
app4.get("/weather", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = await instantService.weather(q);
  return c.json({
    type: "weather",
    query: q,
    answer: result
  });
});
app4.get("/define", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = await instantService.define(q);
  return c.json({
    type: "definition",
    query: q,
    answer: result
  });
});
app4.get("/time", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const instantService = new InstantService(cache);
  const result = instantService.time(q);
  return c.json({
    type: "time",
    query: q,
    answer: result
  });
});
var instant_default = app4;

// src/routes/knowledge.ts
var app5 = new Hono2();
app5.get("/:query", async (c) => {
  const query = c.req.param("query");
  if (!query) {
    return c.json({ error: "Missing required parameter: query" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const knowledgeService = new KnowledgeService(cache);
  const panel = await knowledgeService.getPanel(query);
  if (!panel) {
    return c.json({ error: "No knowledge panel found" }, 404);
  }
  return c.json(panel);
});
var knowledge_default = app5;

// src/lib/utils.ts
function generateId2() {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return Array.from(bytes).map((b) => b.toString(16).padStart(2, "0")).join("");
}
__name(generateId2, "generateId");

// src/routes/preferences.ts
var app6 = new Hono2();
app6.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const preferences = await kvStore.listPreferences();
  return c.json(preferences);
});
app6.post("/", async (c) => {
  const body = await c.req.json();
  if (!body.domain || !body.action) {
    return c.json({ error: "Missing required fields: domain, action" }, 400);
  }
  const pref = {
    id: generateId2(),
    domain: body.domain,
    action: body.action,
    level: body.level ?? 0,
    created_at: (/* @__PURE__ */ new Date()).toISOString()
  };
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.setPreference(pref);
  return c.json({ success: true });
});
app6.delete("/:domain", async (c) => {
  const domain2 = c.req.param("domain");
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.deletePreference(domain2);
  return c.json({ success: true });
});
var preferences_default = app6;

// src/routes/lenses.ts
var app7 = new Hono2();
app7.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const lenses = await kvStore.listLenses();
  return c.json(lenses);
});
app7.post("/", async (c) => {
  const body = await c.req.json();
  const now = (/* @__PURE__ */ new Date()).toISOString();
  const lens = {
    id: generateId2(),
    name: body.name ?? "Untitled",
    description: body.description,
    domains: body.domains,
    exclude: body.exclude,
    include_keywords: body.include_keywords,
    exclude_keywords: body.exclude_keywords,
    keywords: body.keywords,
    region: body.region,
    file_type: body.file_type,
    date_before: body.date_before,
    date_after: body.date_after,
    is_public: body.is_public ?? false,
    is_built_in: false,
    is_shared: body.is_shared ?? false,
    share_link: body.share_link,
    user_id: body.user_id,
    created_at: now,
    updated_at: now
  };
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.createLens(lens);
  return c.json(lens, 201);
});
app7.get("/:id", async (c) => {
  const id = c.req.param("id");
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const lens = await kvStore.getLens(id);
  if (!lens) {
    return c.json({ error: "Lens not found" }, 404);
  }
  return c.json(lens);
});
app7.put("/:id", async (c) => {
  const id = c.req.param("id");
  const body = await c.req.json();
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const lens = await kvStore.updateLens(id, body);
  return c.json(lens);
});
app7.delete("/:id", async (c) => {
  const id = c.req.param("id");
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.deleteLens(id);
  return c.json({ success: true });
});
var lenses_default = app7;

// src/routes/history.ts
var app8 = new Hono2();
app8.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const history = await kvStore.listHistory();
  return c.json(history);
});
app8.delete("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.clearHistory();
  return c.json({ success: true });
});
app8.delete("/:id", async (c) => {
  const id = c.req.param("id");
  const kvStore = new KVStore(c.env.SEARCH_KV);
  await kvStore.deleteHistory(id);
  return c.json({ success: true });
});
var history_default = app8;

// src/routes/settings.ts
var app9 = new Hono2();
app9.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const settings = await kvStore.getSettings();
  return c.json(settings);
});
app9.put("/", async (c) => {
  const body = await c.req.json();
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const settings = await kvStore.updateSettings(body);
  return c.json(settings);
});
var settings_default = app9;

// src/routes/bangs.ts
var app10 = new Hono2();
app10.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const bangService = new BangService(kvStore);
  const bangs = await bangService.listBangs();
  return c.json(bangs);
});
app10.get("/parse", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const bangService = new BangService(kvStore);
  const result = await bangService.parse(q);
  return c.json(result);
});
app10.post("/", async (c) => {
  const body = await c.req.json();
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const bangService = new BangService(kvStore);
  const bang = await bangService.createBang(body);
  return c.json(bang, 201);
});
app10.delete("/:id", async (c) => {
  const id = c.req.param("id");
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const bangService = new BangService(kvStore);
  await bangService.deleteBang(id);
  return c.json({ success: true });
});
var bangs_default = app10;

// src/routes/widgets.ts
var CHEATSHEETS = {
  javascript: {
    language: "javascript",
    title: "JavaScript Cheatsheet",
    sections: [
      {
        title: "Variables & Types",
        items: [
          { syntax: "const x = 1", description: "Declare a block-scoped constant" },
          { syntax: "let y = 2", description: "Declare a block-scoped variable" },
          { syntax: "typeof x", description: "Returns the type of a variable as a string" },
          { syntax: "Array.isArray(arr)", description: "Check if a value is an array" },
          { syntax: "x ?? fallback", description: "Nullish coalescing - use fallback if x is null/undefined" },
          { syntax: "x?.prop", description: "Optional chaining - access property safely" }
        ]
      },
      {
        title: "Functions",
        items: [
          { syntax: "function name(a, b) {}", description: "Function declaration with hoisting" },
          { syntax: "const fn = (a, b) => a + b", description: "Arrow function expression" },
          { syntax: "const fn = (...args) => {}", description: "Rest parameters collect remaining arguments" },
          { syntax: "fn(a, ...arr)", description: "Spread syntax expands array into arguments" },
          { syntax: "function fn(a = 1) {}", description: "Default parameter values" },
          { syntax: "const { a, b } = obj", description: "Destructuring assignment from objects" }
        ]
      },
      {
        title: "Arrays",
        items: [
          { syntax: "arr.map(x => x * 2)", description: "Create new array by transforming each element" },
          { syntax: "arr.filter(x => x > 0)", description: "Create new array with elements passing a test" },
          { syntax: "arr.reduce((acc, x) => acc + x, 0)", description: "Reduce array to single value" },
          { syntax: "arr.find(x => x.id === 1)", description: "Find first element matching condition" },
          { syntax: "arr.some(x => x > 5)", description: "Test if any element passes condition" },
          { syntax: "arr.flat(depth)", description: "Flatten nested arrays to specified depth" },
          { syntax: "[...arr1, ...arr2]", description: "Concatenate arrays using spread" }
        ]
      },
      {
        title: "Async",
        items: [
          { syntax: "async function fn() {}", description: "Declare async function returning a Promise" },
          { syntax: "const result = await promise", description: "Wait for promise to resolve" },
          { syntax: "Promise.all([p1, p2])", description: "Wait for all promises to resolve" },
          { syntax: "Promise.race([p1, p2])", description: "Resolve with the first settled promise" },
          { syntax: "try { await fn() } catch(e) {}", description: "Handle async errors with try/catch" }
        ]
      }
    ]
  },
  python: {
    language: "python",
    title: "Python Cheatsheet",
    sections: [
      {
        title: "Variables & Types",
        items: [
          { syntax: "x: int = 10", description: "Variable with type annotation" },
          { syntax: "type(x)", description: "Get the type of a variable" },
          { syntax: "isinstance(x, int)", description: "Check if object is instance of a type" },
          { syntax: "x = y if cond else z", description: "Ternary conditional expression" },
          { syntax: "a, b = 1, 2", description: "Multiple assignment / tuple unpacking" },
          { syntax: 'f"Hello {name}"', description: "F-string for string interpolation" }
        ]
      },
      {
        title: "Collections",
        items: [
          { syntax: "[x*2 for x in lst]", description: "List comprehension" },
          { syntax: "{k: v for k, v in items}", description: "Dictionary comprehension" },
          { syntax: "set(lst)", description: "Create a set from an iterable" },
          { syntax: "dict.get(key, default)", description: "Get value with a default fallback" },
          { syntax: "lst.sort(key=lambda x: x.name)", description: "Sort list in-place by key function" },
          { syntax: "enumerate(lst)", description: "Iterate with index and value" },
          { syntax: "zip(lst1, lst2)", description: "Iterate over multiple lists in parallel" }
        ]
      },
      {
        title: "Functions & Classes",
        items: [
          { syntax: "def fn(a: int, b: int = 0) -> int:", description: "Function with type hints and default" },
          { syntax: "lambda x: x * 2", description: "Anonymous function expression" },
          { syntax: "*args, **kwargs", description: "Variadic positional and keyword arguments" },
          { syntax: "@decorator", description: "Apply a decorator to a function or class" },
          { syntax: "class Foo(Bar):", description: "Class definition with inheritance" },
          { syntax: "@property", description: "Define a getter property on a class" }
        ]
      },
      {
        title: "Control Flow",
        items: [
          { syntax: "for x in range(10):", description: "Loop over a range of numbers" },
          { syntax: "while cond:", description: "Loop while condition is true" },
          { syntax: "with open(f) as fh:", description: "Context manager for resource handling" },
          { syntax: "try: ... except E as e:", description: "Exception handling with specific type" },
          { syntax: "match value: case pattern:", description: "Structural pattern matching (3.10+)" }
        ]
      }
    ]
  },
  go: {
    language: "go",
    title: "Go Cheatsheet",
    sections: [
      {
        title: "Variables & Types",
        items: [
          { syntax: "var x int = 10", description: "Explicit variable declaration with type" },
          { syntax: "x := 10", description: "Short variable declaration with type inference" },
          { syntax: "const Pi = 3.14", description: "Constant declaration" },
          { syntax: "type Point struct { X, Y int }", description: "Define a struct type" },
          { syntax: "type Reader interface { Read([]byte) (int, error) }", description: "Define an interface" },
          { syntax: "p := &Point{1, 2}", description: "Create pointer to struct literal" }
        ]
      },
      {
        title: "Functions & Methods",
        items: [
          { syntax: "func add(a, b int) int {}", description: "Function with parameters and return type" },
          { syntax: "func div(a, b int) (int, error) {}", description: "Function with multiple return values" },
          { syntax: "func (p *Point) Scale(f int) {}", description: "Method with pointer receiver" },
          { syntax: "func process(fn func(int) int) {}", description: "Function as parameter" },
          { syntax: "defer file.Close()", description: "Defer execution until function returns" },
          { syntax: "func variadic(nums ...int) {}", description: "Variadic function parameters" }
        ]
      },
      {
        title: "Concurrency",
        items: [
          { syntax: "go fn()", description: "Launch a goroutine" },
          { syntax: "ch := make(chan int)", description: "Create an unbuffered channel" },
          { syntax: "ch := make(chan int, 10)", description: "Create a buffered channel" },
          { syntax: "select { case v := <-ch: ... }", description: "Select on multiple channel operations" },
          { syntax: "var mu sync.Mutex", description: "Mutex for protecting shared state" },
          { syntax: "var wg sync.WaitGroup", description: "Wait for a group of goroutines to finish" }
        ]
      },
      {
        title: "Collections",
        items: [
          { syntax: "sl := []int{1, 2, 3}", description: "Slice literal" },
          { syntax: "sl = append(sl, 4)", description: "Append to a slice" },
          { syntax: 'm := map[string]int{"a": 1}', description: "Map literal" },
          { syntax: 'v, ok := m["key"]', description: "Map lookup with existence check" },
          { syntax: "for i, v := range sl {}", description: "Range over slice with index and value" },
          { syntax: "copy(dst, src)", description: "Copy elements between slices" }
        ]
      }
    ]
  },
  rust: {
    language: "rust",
    title: "Rust Cheatsheet",
    sections: [
      {
        title: "Variables & Types",
        items: [
          { syntax: "let x: i32 = 10;", description: "Immutable variable binding with type" },
          { syntax: "let mut y = 20;", description: "Mutable variable binding" },
          { syntax: "const MAX: u32 = 100;", description: "Compile-time constant" },
          { syntax: 'let s = String::from("hello");', description: "Create an owned String from a literal" },
          { syntax: "let r: &str = &s;", description: "Borrow as a string slice reference" },
          { syntax: "type Point = (f64, f64);", description: "Type alias for a tuple" }
        ]
      },
      {
        title: "Ownership & Borrowing",
        items: [
          { syntax: "let s2 = s1.clone();", description: "Deep clone to avoid move" },
          { syntax: "fn borrow(s: &String) {}", description: "Immutable reference parameter" },
          { syntax: "fn mutate(s: &mut String) {}", description: "Mutable reference parameter" },
          { syntax: "let r = &v[0..3];", description: "Slice reference to a portion of a collection" },
          { syntax: "Box::new(value)", description: "Heap-allocate a value with Box" },
          { syntax: "Rc::clone(&shared)", description: "Reference-counted shared ownership" }
        ]
      },
      {
        title: "Enums & Pattern Matching",
        items: [
          { syntax: "enum Option<T> { Some(T), None }", description: "Generic enum for optional values" },
          { syntax: "match value { Pat => expr }", description: "Exhaustive pattern matching" },
          { syntax: "if let Some(v) = opt { }", description: "Conditional pattern match" },
          { syntax: "value.unwrap_or(default)", description: "Unwrap Option/Result with fallback" },
          { syntax: "result?", description: "Propagate errors with the ? operator" }
        ]
      },
      {
        title: "Traits & Generics",
        items: [
          { syntax: "trait Summary { fn summarize(&self) -> String; }", description: "Define a trait with a method" },
          { syntax: "impl Summary for Article {}", description: "Implement a trait for a type" },
          { syntax: "fn print<T: Display>(val: T) {}", description: "Generic function with trait bound" },
          { syntax: "fn process(item: &dyn Summary) {}", description: "Dynamic dispatch with trait object" },
          { syntax: "#[derive(Debug, Clone)]", description: "Auto-derive trait implementations" }
        ]
      }
    ]
  },
  html: {
    language: "html",
    title: "HTML Cheatsheet",
    sections: [
      {
        title: "Document Structure",
        items: [
          { syntax: "<!DOCTYPE html>", description: "HTML5 document type declaration" },
          { syntax: '<html lang="en">', description: "Root element with language attribute" },
          { syntax: "<head>", description: "Container for metadata, links, and scripts" },
          { syntax: '<meta charset="utf-8">', description: "Character encoding declaration" },
          { syntax: '<meta name="viewport" content="width=device-width">', description: "Responsive viewport setting" },
          { syntax: '<link rel="stylesheet" href="style.css">', description: "Link external stylesheet" }
        ]
      },
      {
        title: "Semantic Elements",
        items: [
          { syntax: "<header>", description: "Introductory content or navigation container" },
          { syntax: "<nav>", description: "Navigation links section" },
          { syntax: "<main>", description: "Dominant content of the document" },
          { syntax: "<article>", description: "Self-contained composition" },
          { syntax: "<section>", description: "Thematic grouping of content" },
          { syntax: "<aside>", description: "Content tangentially related to surrounding content" },
          { syntax: "<footer>", description: "Footer for nearest section or root" }
        ]
      },
      {
        title: "Forms",
        items: [
          { syntax: '<form action="/submit" method="post">', description: "Form with action URL and method" },
          { syntax: '<input type="text" name="q" required>', description: "Text input with validation" },
          { syntax: '<input type="email" placeholder="you@example.com">', description: "Email input with placeholder" },
          { syntax: '<select name="option"><option>A</option></select>', description: "Dropdown select element" },
          { syntax: '<textarea rows="4" cols="50"></textarea>', description: "Multi-line text input" },
          { syntax: '<button type="submit">Send</button>', description: "Submit button element" }
        ]
      },
      {
        title: "Media & Embedding",
        items: [
          { syntax: '<img src="img.png" alt="Description">', description: "Image with alt text for accessibility" },
          { syntax: '<picture><source srcset="img.webp" type="image/webp"></picture>', description: "Responsive image with multiple sources" },
          { syntax: '<video src="vid.mp4" controls></video>', description: "Video element with playback controls" },
          { syntax: '<audio src="sound.mp3" controls></audio>', description: "Audio element with playback controls" },
          { syntax: '<canvas id="c" width="300" height="200"></canvas>', description: "Drawing surface for graphics" }
        ]
      }
    ]
  },
  css: {
    language: "css",
    title: "CSS Cheatsheet",
    sections: [
      {
        title: "Selectors",
        items: [
          { syntax: ".class", description: "Select elements by class name" },
          { syntax: "#id", description: "Select element by ID" },
          { syntax: "parent > child", description: "Direct child combinator" },
          { syntax: "a:hover", description: "Pseudo-class for hover state" },
          { syntax: "p::first-line", description: "Pseudo-element for first line of text" },
          { syntax: '[data-attr="value"]', description: "Attribute selector with value match" },
          { syntax: ":has(.child)", description: "Parent selector based on child (CSS4)" }
        ]
      },
      {
        title: "Layout",
        items: [
          { syntax: "display: flex;", description: "Enable flexbox layout on container" },
          { syntax: "display: grid;", description: "Enable grid layout on container" },
          { syntax: "grid-template-columns: repeat(3, 1fr);", description: "Three equal-width grid columns" },
          { syntax: "justify-content: center;", description: "Center items along main axis" },
          { syntax: "align-items: center;", description: "Center items along cross axis" },
          { syntax: "gap: 1rem;", description: "Gap between flex or grid items" }
        ]
      },
      {
        title: "Responsive",
        items: [
          { syntax: "@media (max-width: 768px) {}", description: "Media query for mobile screens" },
          { syntax: "clamp(1rem, 2vw, 3rem)", description: "Fluid value between min and max" },
          { syntax: "@container (min-width: 400px) {}", description: "Container query for component-level responsiveness" },
          { syntax: "aspect-ratio: 16 / 9;", description: "Maintain aspect ratio on element" },
          { syntax: "min-width: 0;", description: "Prevent flex/grid children from overflowing" }
        ]
      },
      {
        title: "Custom Properties & Functions",
        items: [
          { syntax: "--color: #333;", description: "Define a CSS custom property" },
          { syntax: "var(--color, fallback)", description: "Use custom property with fallback" },
          { syntax: "calc(100% - 2rem)", description: "Perform calculations in values" },
          { syntax: "color: oklch(70% 0.15 200);", description: "OKLCH color space for perceptually uniform colors" },
          { syntax: "@layer base, components;", description: "Declare cascade layers for specificity control" }
        ]
      }
    ]
  },
  sql: {
    language: "sql",
    title: "SQL Cheatsheet",
    sections: [
      {
        title: "Queries",
        items: [
          { syntax: "SELECT col FROM table WHERE cond;", description: "Basic select with condition" },
          { syntax: "SELECT DISTINCT col FROM table;", description: "Select unique values only" },
          { syntax: "SELECT * FROM t ORDER BY col DESC LIMIT 10;", description: "Sort descending and limit rows" },
          { syntax: 'SELECT * FROM t WHERE col LIKE "%pattern%";', description: "Pattern matching with wildcards" },
          { syntax: "SELECT * FROM t WHERE col IN (1, 2, 3);", description: "Match against a list of values" },
          { syntax: "SELECT * FROM t WHERE col BETWEEN 1 AND 10;", description: "Range condition inclusive" }
        ]
      },
      {
        title: "Joins & Subqueries",
        items: [
          { syntax: "SELECT * FROM a INNER JOIN b ON a.id = b.a_id;", description: "Inner join matching rows from both tables" },
          { syntax: "SELECT * FROM a LEFT JOIN b ON a.id = b.a_id;", description: "Left join keeps all rows from left table" },
          { syntax: "SELECT * FROM a CROSS JOIN b;", description: "Cartesian product of two tables" },
          { syntax: "SELECT * FROM t WHERE id IN (SELECT id FROM t2);", description: "Subquery in WHERE clause" },
          { syntax: "WITH cte AS (SELECT ...) SELECT * FROM cte;", description: "Common Table Expression (CTE)" }
        ]
      },
      {
        title: "Aggregation",
        items: [
          { syntax: "SELECT COUNT(*) FROM table;", description: "Count total rows" },
          { syntax: "SELECT col, COUNT(*) FROM t GROUP BY col;", description: "Group and count per group" },
          { syntax: "SELECT col, SUM(val) FROM t GROUP BY col HAVING SUM(val) > 100;", description: "Filter groups with HAVING" },
          { syntax: "SELECT AVG(col), MIN(col), MAX(col) FROM t;", description: "Aggregate functions for statistics" },
          { syntax: "SELECT ROW_NUMBER() OVER (ORDER BY col) FROM t;", description: "Window function for row numbering" }
        ]
      },
      {
        title: "Data Modification",
        items: [
          { syntax: "INSERT INTO t (col1, col2) VALUES (v1, v2);", description: "Insert a new row" },
          { syntax: "UPDATE t SET col = val WHERE cond;", description: "Update existing rows" },
          { syntax: "DELETE FROM t WHERE cond;", description: "Delete rows matching condition" },
          { syntax: "CREATE TABLE t (id INT PRIMARY KEY, name TEXT);", description: "Create a new table" },
          { syntax: "ALTER TABLE t ADD COLUMN col TYPE;", description: "Add a column to existing table" }
        ]
      }
    ]
  },
  bash: {
    language: "bash",
    title: "Bash Cheatsheet",
    sections: [
      {
        title: "Variables & Strings",
        items: [
          { syntax: 'VAR="value"', description: "Assign a variable (no spaces around =)" },
          { syntax: "${VAR:-default}", description: "Use default if variable is unset" },
          { syntax: "${#VAR}", description: "Get length of variable value" },
          { syntax: "${VAR//old/new}", description: "Replace all occurrences in variable" },
          { syntax: "$(command)", description: "Command substitution - capture output" },
          { syntax: '"$VAR"', description: "Double quotes preserve variable expansion" }
        ]
      },
      {
        title: "Control Flow",
        items: [
          { syntax: "if [[ $x -gt 0 ]]; then ... fi", description: "Conditional with numeric comparison" },
          { syntax: "for f in *.txt; do ... done", description: "Loop over glob pattern matches" },
          { syntax: "while read -r line; do ... done < file", description: "Read file line by line" },
          { syntax: 'case "$VAR" in pat) cmd;; esac', description: "Pattern matching switch statement" },
          { syntax: "[[ -f file ]] && echo exists", description: "Test if file exists with short-circuit" },
          { syntax: "cmd1 || cmd2", description: "Run cmd2 only if cmd1 fails" }
        ]
      },
      {
        title: "Pipes & Redirection",
        items: [
          { syntax: "cmd1 | cmd2", description: "Pipe stdout of cmd1 to stdin of cmd2" },
          { syntax: "cmd > file", description: "Redirect stdout to file (overwrite)" },
          { syntax: "cmd >> file", description: "Redirect stdout to file (append)" },
          { syntax: "cmd 2>&1", description: "Redirect stderr to stdout" },
          { syntax: "cmd < file", description: "Read stdin from file" },
          { syntax: "cmd1 | tee file | cmd2", description: "Split output to file and next command" }
        ]
      },
      {
        title: "Common Commands",
        items: [
          { syntax: 'find . -name "*.log" -mtime +7 -delete', description: "Find and delete files older than 7 days" },
          { syntax: 'grep -rn "pattern" dir/', description: "Recursively search for pattern with line numbers" },
          { syntax: "awk '{print $1}' file", description: "Print first column of each line" },
          { syntax: "sed -i 's/old/new/g' file", description: "In-place find and replace in file" },
          { syntax: "xargs -I{} cmd {}", description: "Execute command for each stdin line" }
        ]
      }
    ]
  },
  git: {
    language: "git",
    title: "Git Cheatsheet",
    sections: [
      {
        title: "Basic Commands",
        items: [
          { syntax: "git init", description: "Initialize a new repository" },
          { syntax: "git clone <url>", description: "Clone a remote repository" },
          { syntax: "git add -p", description: "Interactively stage hunks" },
          { syntax: 'git commit -m "message"', description: "Commit staged changes with message" },
          { syntax: "git status", description: "Show working tree status" },
          { syntax: "git diff --staged", description: "Show staged changes vs last commit" }
        ]
      },
      {
        title: "Branching",
        items: [
          { syntax: "git branch feature", description: "Create a new branch" },
          { syntax: "git checkout -b feature", description: "Create and switch to new branch" },
          { syntax: "git switch main", description: "Switch to an existing branch" },
          { syntax: "git merge feature", description: "Merge branch into current branch" },
          { syntax: "git rebase main", description: "Rebase current branch onto main" },
          { syntax: "git branch -d feature", description: "Delete a merged branch" }
        ]
      },
      {
        title: "Remote Operations",
        items: [
          { syntax: "git remote add origin <url>", description: "Add a remote repository" },
          { syntax: "git fetch origin", description: "Download objects and refs from remote" },
          { syntax: "git pull --rebase", description: "Fetch and rebase local commits on top" },
          { syntax: "git push -u origin feature", description: "Push branch and set upstream tracking" },
          { syntax: "git push origin --delete feature", description: "Delete a remote branch" }
        ]
      },
      {
        title: "History & Undo",
        items: [
          { syntax: "git log --oneline --graph", description: "Compact log with branch graph" },
          { syntax: "git stash", description: "Stash uncommitted changes" },
          { syntax: "git stash pop", description: "Apply and remove most recent stash" },
          { syntax: "git reset HEAD~1", description: "Undo last commit, keep changes staged" },
          { syntax: "git revert <hash>", description: "Create commit undoing a specific commit" },
          { syntax: "git cherry-pick <hash>", description: "Apply a commit from another branch" }
        ]
      }
    ]
  },
  regex: {
    language: "regex",
    title: "Regular Expressions Cheatsheet",
    sections: [
      {
        title: "Character Classes",
        items: [
          { syntax: ".", description: "Match any character except newline" },
          { syntax: "\\d", description: "Match any digit (0-9)" },
          { syntax: "\\w", description: "Match word character (letter, digit, underscore)" },
          { syntax: "\\s", description: "Match whitespace (space, tab, newline)" },
          { syntax: "[abc]", description: "Match any character in the set" },
          { syntax: "[^abc]", description: "Match any character NOT in the set" },
          { syntax: "[a-z]", description: "Match any character in the range" }
        ]
      },
      {
        title: "Quantifiers",
        items: [
          { syntax: "*", description: "Match 0 or more times (greedy)" },
          { syntax: "+", description: "Match 1 or more times (greedy)" },
          { syntax: "?", description: "Match 0 or 1 time (optional)" },
          { syntax: "{n}", description: "Match exactly n times" },
          { syntax: "{n,m}", description: "Match between n and m times" },
          { syntax: "*?", description: "Match 0 or more times (lazy/non-greedy)" }
        ]
      },
      {
        title: "Anchors & Boundaries",
        items: [
          { syntax: "^", description: "Match start of string (or line with m flag)" },
          { syntax: "$", description: "Match end of string (or line with m flag)" },
          { syntax: "\\b", description: "Match word boundary" },
          { syntax: "(?=pattern)", description: "Positive lookahead assertion" },
          { syntax: "(?!pattern)", description: "Negative lookahead assertion" },
          { syntax: "(?<=pattern)", description: "Positive lookbehind assertion" }
        ]
      },
      {
        title: "Groups & References",
        items: [
          { syntax: "(pattern)", description: "Capturing group" },
          { syntax: "(?:pattern)", description: "Non-capturing group" },
          { syntax: "(?<name>pattern)", description: "Named capturing group" },
          { syntax: "\\1", description: "Backreference to first capturing group" },
          { syntax: "a|b", description: "Alternation - match a or b" },
          { syntax: "(?i)", description: "Case-insensitive flag" }
        ]
      }
    ]
  }
};
var AVAILABLE_LANGUAGES = Object.keys(CHEATSHEETS);
var DEFAULT_WIDGET_SETTINGS2 = {
  weather: true,
  calculator: true,
  converter: true,
  dictionary: true,
  cheatsheets: true,
  relatedSearches: true,
  knowledgePanel: true
};
var widgetSettingsApp = new Hono2();
widgetSettingsApp.get("/", async (c) => {
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const settings = await kvStore.getWidgetSettings();
  return c.json(settings ?? DEFAULT_WIDGET_SETTINGS2);
});
widgetSettingsApp.put("/", async (c) => {
  const body = await c.req.json();
  const kvStore = new KVStore(c.env.SEARCH_KV);
  const settings = await kvStore.updateWidgetSettings(body);
  return c.json(settings);
});
var cheatsheetApp = new Hono2();
cheatsheetApp.get("/:language", (c) => {
  const language = c.req.param("language").toLowerCase();
  const cheatsheet = CHEATSHEETS[language];
  if (!cheatsheet) {
    return c.json(
      { error: `Cheatsheet not found for language: ${language}`, available: AVAILABLE_LANGUAGES },
      404
    );
  }
  return c.json(cheatsheet);
});
var cheatsheetsListApp = new Hono2();
cheatsheetsListApp.get("/", (c) => {
  const list = AVAILABLE_LANGUAGES.map((lang) => ({
    language: lang,
    title: CHEATSHEETS[lang].title
  }));
  return c.json(list);
});
var relatedApp = new Hono2();
relatedApp.get("/", async (c) => {
  const q = c.req.query("q") ?? "";
  if (!q) {
    return c.json({ error: "Missing required parameter: q" }, 400);
  }
  const cache = new CacheStore(c.env.SEARCH_KV);
  const suggestService = new SuggestService(cache);
  const suggestions = await suggestService.suggest(q);
  return c.json({ query: q, related: suggestions });
});

// src/index.ts
var app11 = new Hono2();
app11.use("*", cors());
app11.use("*", timing());
app11.route("/health", health_default);
app11.route("/api/search", search_default);
app11.route("/api/suggest", suggest_default);
app11.route("/api/instant", instant_default);
app11.route("/api/knowledge", knowledge_default);
app11.route("/api/preferences", preferences_default);
app11.route("/api/lenses", lenses_default);
app11.route("/api/history", history_default);
app11.route("/api/settings", settings_default);
app11.route("/api/bangs", bangs_default);
app11.route("/api/widgets", widgetSettingsApp);
app11.route("/api/cheatsheet", cheatsheetApp);
app11.route("/api/cheatsheets", cheatsheetsListApp);
app11.route("/api/related", relatedApp);
app11.get("*", async (c) => {
  return c.html('<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0;url=/"></head><body></body></html>');
});
var src_default = app11;

// node_modules/.pnpm/wrangler@4.62.0_@cloudflare+workers-types@4.20260205.0/node_modules/wrangler/templates/middleware/middleware-ensure-req-body-drained.ts
var drainBody = /* @__PURE__ */ __name(async (request, env2, _ctx, middlewareCtx) => {
  try {
    return await middlewareCtx.next(request, env2);
  } finally {
    try {
      if (request.body !== null && !request.bodyUsed) {
        const reader = request.body.getReader();
        while (!(await reader.read()).done) {
        }
      }
    } catch (e) {
      console.error("Failed to drain the unused request body.", e);
    }
  }
}, "drainBody");
var middleware_ensure_req_body_drained_default = drainBody;

// node_modules/.pnpm/wrangler@4.62.0_@cloudflare+workers-types@4.20260205.0/node_modules/wrangler/templates/middleware/middleware-miniflare3-json-error.ts
function reduceError(e) {
  return {
    name: e?.name,
    message: e?.message ?? String(e),
    stack: e?.stack,
    cause: e?.cause === void 0 ? void 0 : reduceError(e.cause)
  };
}
__name(reduceError, "reduceError");
var jsonError = /* @__PURE__ */ __name(async (request, env2, _ctx, middlewareCtx) => {
  try {
    return await middlewareCtx.next(request, env2);
  } catch (e) {
    const error = reduceError(e);
    return Response.json(error, {
      status: 500,
      headers: { "MF-Experimental-Error-Stack": "true" }
    });
  }
}, "jsonError");
var middleware_miniflare3_json_error_default = jsonError;

// .wrangler/tmp/bundle-a0xNuq/middleware-insertion-facade.js
var __INTERNAL_WRANGLER_MIDDLEWARE__ = [
  middleware_ensure_req_body_drained_default,
  middleware_miniflare3_json_error_default
];
var middleware_insertion_facade_default = src_default;

// node_modules/.pnpm/wrangler@4.62.0_@cloudflare+workers-types@4.20260205.0/node_modules/wrangler/templates/middleware/common.ts
var __facade_middleware__ = [];
function __facade_register__(...args) {
  __facade_middleware__.push(...args.flat());
}
__name(__facade_register__, "__facade_register__");
function __facade_invokeChain__(request, env2, ctx, dispatch, middlewareChain) {
  const [head, ...tail] = middlewareChain;
  const middlewareCtx = {
    dispatch,
    next(newRequest, newEnv) {
      return __facade_invokeChain__(newRequest, newEnv, ctx, dispatch, tail);
    }
  };
  return head(request, env2, ctx, middlewareCtx);
}
__name(__facade_invokeChain__, "__facade_invokeChain__");
function __facade_invoke__(request, env2, ctx, dispatch, finalMiddleware) {
  return __facade_invokeChain__(request, env2, ctx, dispatch, [
    ...__facade_middleware__,
    finalMiddleware
  ]);
}
__name(__facade_invoke__, "__facade_invoke__");

// .wrangler/tmp/bundle-a0xNuq/middleware-loader.entry.ts
var __Facade_ScheduledController__ = class ___Facade_ScheduledController__ {
  constructor(scheduledTime, cron, noRetry) {
    this.scheduledTime = scheduledTime;
    this.cron = cron;
    this.#noRetry = noRetry;
  }
  static {
    __name(this, "__Facade_ScheduledController__");
  }
  #noRetry;
  noRetry() {
    if (!(this instanceof ___Facade_ScheduledController__)) {
      throw new TypeError("Illegal invocation");
    }
    this.#noRetry();
  }
};
function wrapExportedHandler(worker) {
  if (__INTERNAL_WRANGLER_MIDDLEWARE__ === void 0 || __INTERNAL_WRANGLER_MIDDLEWARE__.length === 0) {
    return worker;
  }
  for (const middleware of __INTERNAL_WRANGLER_MIDDLEWARE__) {
    __facade_register__(middleware);
  }
  const fetchDispatcher = /* @__PURE__ */ __name(function(request, env2, ctx) {
    if (worker.fetch === void 0) {
      throw new Error("Handler does not export a fetch() function.");
    }
    return worker.fetch(request, env2, ctx);
  }, "fetchDispatcher");
  return {
    ...worker,
    fetch(request, env2, ctx) {
      const dispatcher = /* @__PURE__ */ __name(function(type, init) {
        if (type === "scheduled" && worker.scheduled !== void 0) {
          const controller = new __Facade_ScheduledController__(
            Date.now(),
            init.cron ?? "",
            () => {
            }
          );
          return worker.scheduled(controller, env2, ctx);
        }
      }, "dispatcher");
      return __facade_invoke__(request, env2, ctx, dispatcher, fetchDispatcher);
    }
  };
}
__name(wrapExportedHandler, "wrapExportedHandler");
function wrapWorkerEntrypoint(klass) {
  if (__INTERNAL_WRANGLER_MIDDLEWARE__ === void 0 || __INTERNAL_WRANGLER_MIDDLEWARE__.length === 0) {
    return klass;
  }
  for (const middleware of __INTERNAL_WRANGLER_MIDDLEWARE__) {
    __facade_register__(middleware);
  }
  return class extends klass {
    #fetchDispatcher = /* @__PURE__ */ __name((request, env2, ctx) => {
      this.env = env2;
      this.ctx = ctx;
      if (super.fetch === void 0) {
        throw new Error("Entrypoint class does not define a fetch() function.");
      }
      return super.fetch(request);
    }, "#fetchDispatcher");
    #dispatcher = /* @__PURE__ */ __name((type, init) => {
      if (type === "scheduled" && super.scheduled !== void 0) {
        const controller = new __Facade_ScheduledController__(
          Date.now(),
          init.cron ?? "",
          () => {
          }
        );
        return super.scheduled(controller);
      }
    }, "#dispatcher");
    fetch(request) {
      return __facade_invoke__(
        request,
        this.env,
        this.ctx,
        this.#dispatcher,
        this.#fetchDispatcher
      );
    }
  };
}
__name(wrapWorkerEntrypoint, "wrapWorkerEntrypoint");
var WRAPPED_ENTRY;
if (typeof middleware_insertion_facade_default === "object") {
  WRAPPED_ENTRY = wrapExportedHandler(middleware_insertion_facade_default);
} else if (typeof middleware_insertion_facade_default === "function") {
  WRAPPED_ENTRY = wrapWorkerEntrypoint(middleware_insertion_facade_default);
}
var middleware_loader_entry_default = WRAPPED_ENTRY;
export {
  __INTERNAL_WRANGLER_MIDDLEWARE__,
  middleware_loader_entry_default as default
};
//# sourceMappingURL=index.js.map
