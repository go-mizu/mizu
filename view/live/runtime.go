package live

// clientRuntime is the JavaScript client runtime.
// It handles:
// - WebSocket connection and reconnection
// - Event capture and dispatch
// - DOM patching via morphdom-style updates
// - Loading states
// - Client commands
const clientRuntime = `
(function() {
  'use strict';

  // MizuLive global object
  window.MizuLive = {
    version: '1.0.0',
    debug: false,
    conn: null,
    sessionId: null,
    url: null,
    reconnectAttempts: 0,
    maxReconnectAttempts: 10,
    reconnectDelays: [1000, 2000, 4000, 8000, 16000, 30000],
    pendingEvents: new Map(),
    eventRef: 0,
    debounceTimers: new Map(),
    throttleTimers: new Map(),

    log: function(...args) {
      if (this.debug) console.log('[MizuLive]', ...args);
    },

    connect: function(options) {
      this.url = options.url || window.location.pathname;
      this.sessionId = options.sessionId || null;
      this.debug = options.debug || false;

      this.setupEventListeners();
      this.openConnection();
    },

    openConnection: function() {
      var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      var wsUrl = protocol + '//' + window.location.host + '/_live/websocket';

      this.log('Connecting to', wsUrl);

      try {
        this.conn = new WebSocket(wsUrl);
        this.conn.onopen = this.onOpen.bind(this);
        this.conn.onclose = this.onClose.bind(this);
        this.conn.onerror = this.onError.bind(this);
        this.conn.onmessage = this.onMessage.bind(this);
      } catch (e) {
        this.log('Connection error:', e);
        this.scheduleReconnect();
      }
    },

    onOpen: function() {
      this.log('Connected');
      this.reconnectAttempts = 0;

      // Send JOIN message
      this.send({
        type: 1, // MsgTypeJoin
        ref: 1,
        payload: {
          token: this.getCSRFToken(),
          url: this.url,
          session: this.sessionId,
          reconnect: this.sessionId !== null
        }
      });

      this.setConnectedState(true);
    },

    onClose: function(event) {
      this.log('Disconnected', event.code, event.reason);
      this.conn = null;
      this.setConnectedState(false);

      if (event.code !== 1000) {
        this.scheduleReconnect();
      }
    },

    onError: function(event) {
      this.log('Error:', event);
    },

    onMessage: function(event) {
      try {
        var msg = JSON.parse(event.data);
        this.handleMessage(msg);
      } catch (e) {
        this.log('Parse error:', e);
      }
    },

    handleMessage: function(msg) {
      this.log('Received:', msg.type, msg);

      switch (msg.type) {
        case 4: // MsgTypeHeartbeat
          this.handleHeartbeat(msg);
          break;
        case 5: // MsgTypeReply
          this.handleReply(msg);
          break;
        case 6: // MsgTypePatch
          this.handlePatch(msg);
          break;
        case 7: // MsgTypeCommand
          this.handleCommands(msg);
          break;
        case 8: // MsgTypeError
          this.handleError(msg);
          break;
        case 10: // MsgTypeClose
          this.handleClose(msg);
          break;
      }
    },

    handleHeartbeat: function(msg) {
      // Pong already received, nothing to do
    },

    handleReply: function(msg) {
      var payload = msg.payload || {};

      if (payload.session_id) {
        this.sessionId = payload.session_id;
      }

      if (payload.rendered) {
        for (var id in payload.rendered) {
          this.patchRegion(id, payload.rendered[id]);
        }
      }

      // Clear loading state for this ref
      var pending = this.pendingEvents.get(msg.ref);
      if (pending) {
        this.clearLoading(pending.element);
        this.pendingEvents.delete(msg.ref);
      }
    },

    handlePatch: function(msg) {
      var payload = msg.payload || {};
      var regions = payload.regions || [];

      for (var i = 0; i < regions.length; i++) {
        var region = regions[i];
        this.patchRegion(region.id, region.html, region.action);
      }

      if (payload.title) {
        document.title = payload.title;
      }
    },

    handleCommands: function(msg) {
      var payload = msg.payload || {};
      var commands = payload.commands || [];

      for (var i = 0; i < commands.length; i++) {
        this.executeCommand(commands[i]);
      }
    },

    handleError: function(msg) {
      var payload = msg.payload || {};
      console.error('[MizuLive] Error:', payload.code, payload.message);
    },

    handleClose: function(msg) {
      var payload = msg.payload || {};
      this.log('Server closed connection:', payload.reason, payload.message);

      if (this.conn) {
        this.conn.close(1000, 'Server requested close');
      }
    },

    patchRegion: function(id, html, action) {
      var el = document.getElementById(id);
      if (!el) {
        this.log('Region not found:', id);
        return;
      }

      action = action || 'morph';

      switch (action) {
        case 'replace':
        case 'morph':
          this.morphElement(el, html);
          break;
        case 'append':
          el.insertAdjacentHTML('beforeend', html);
          break;
        case 'prepend':
          el.insertAdjacentHTML('afterbegin', html);
          break;
        case 'before':
          el.insertAdjacentHTML('beforebegin', html);
          break;
        case 'after':
          el.insertAdjacentHTML('afterend', html);
          break;
        case 'remove':
          el.remove();
          break;
      }
    },

    morphElement: function(el, html) {
      // Simple innerHTML replacement with focus preservation
      var focused = document.activeElement;
      var focusId = focused ? focused.id : null;
      var focusValue = focused && focused.value !== undefined ? focused.value : null;
      var selStart = focused && focused.selectionStart !== undefined ? focused.selectionStart : null;
      var selEnd = focused && focused.selectionEnd !== undefined ? focused.selectionEnd : null;

      el.innerHTML = html;

      // Restore focus if possible
      if (focusId) {
        var newFocused = document.getElementById(focusId);
        if (newFocused) {
          newFocused.focus();
          if (focusValue !== null && newFocused.value !== undefined) {
            // Restore value if needed
          }
          if (selStart !== null && newFocused.setSelectionRange) {
            try {
              newFocused.setSelectionRange(selStart, selEnd);
            } catch (e) {}
          }
        }
      }
    },

    executeCommand: function(cmd) {
      var data = cmd.data || {};

      switch (cmd.cmd) {
        case 'redirect':
          if (data.replace) {
            window.location.replace(data.to);
          } else {
            window.location.href = data.to;
          }
          break;
        case 'focus':
          var focusEl = document.querySelector(data.selector);
          if (focusEl) focusEl.focus();
          break;
        case 'scroll':
          var scrollEl = data.selector ? document.querySelector(data.selector) : null;
          if (scrollEl) {
            scrollEl.scrollIntoView({ block: data.block || 'start', behavior: 'smooth' });
          } else {
            window.scrollTo({ top: 0, behavior: 'smooth' });
          }
          break;
        case 'download':
          var a = document.createElement('a');
          a.href = data.url;
          a.download = data.filename || '';
          document.body.appendChild(a);
          a.click();
          document.body.removeChild(a);
          break;
        case 'js':
          try {
            var fn = new Function('args', data.code);
            fn(data.args || {});
          } catch (e) {
            console.error('[MizuLive] JS error:', e);
          }
          break;
        case 'title':
          document.title = data.title;
          break;
        case 'add_class':
          var addEl = document.querySelector(data.selector);
          if (addEl) addEl.classList.add(...data.class.split(' '));
          break;
        case 'remove_class':
          var remEl = document.querySelector(data.selector);
          if (remEl) remEl.classList.remove(...data.class.split(' '));
          break;
        case 'toggle_class':
          var togEl = document.querySelector(data.selector);
          if (togEl) {
            data.class.split(' ').forEach(function(c) {
              togEl.classList.toggle(c);
            });
          }
          break;
        case 'set_attr':
          var attrEl = document.querySelector(data.selector);
          if (attrEl) attrEl.setAttribute(data.name, data.value);
          break;
        case 'remove_attr':
          var remAttrEl = document.querySelector(data.selector);
          if (remAttrEl) remAttrEl.removeAttribute(data.name);
          break;
      }
    },

    scheduleReconnect: function() {
      if (this.reconnectAttempts >= this.maxReconnectAttempts) {
        this.log('Max reconnect attempts reached');
        return;
      }

      var delay = this.reconnectDelays[Math.min(this.reconnectAttempts, this.reconnectDelays.length - 1)];
      this.reconnectAttempts++;

      this.log('Reconnecting in', delay, 'ms (attempt', this.reconnectAttempts + ')');

      setTimeout(this.openConnection.bind(this), delay);
    },

    send: function(msg) {
      if (!this.conn || this.conn.readyState !== WebSocket.OPEN) {
        this.log('Cannot send, not connected');
        return false;
      }

      this.conn.send(JSON.stringify(msg));
      return true;
    },

    sendEvent: function(name, element, values, form) {
      var ref = ++this.eventRef;

      var payload = {
        name: name,
        values: values || {},
        form: form || null,
        meta: {
          shift: false,
          ctrl: false,
          alt: false,
          meta: false
        }
      };

      var target = element.dataset.lvTarget;
      if (target) {
        payload.target = target;
      }

      this.setLoading(element);
      this.pendingEvents.set(ref, { element: element, name: name });

      this.send({
        type: 3, // MsgTypeEvent
        ref: ref,
        payload: payload
      });
    },

    setupEventListeners: function() {
      var self = this;

      // Click events
      document.addEventListener('click', function(e) {
        var el = e.target.closest('[data-lv-click]');
        if (!el) return;

        e.preventDefault();
        if (el.dataset.lvStop) e.stopPropagation();

        var values = self.extractValues(el);
        self.sendEvent(el.dataset.lvClick, el, values);
      }, true);

      // Submit events
      document.addEventListener('submit', function(e) {
        var el = e.target.closest('[data-lv-submit]');
        if (!el) return;

        e.preventDefault();
        if (el.dataset.lvPrevent !== undefined) e.preventDefault();

        var values = self.extractValues(el);
        var form = self.extractFormData(el);
        self.sendEvent(el.dataset.lvSubmit, el, values, form);
      }, true);

      // Change events
      document.addEventListener('change', function(e) {
        var el = e.target.closest('[data-lv-change]');
        if (!el) return;

        var debounce = parseInt(el.dataset.lvDebounce || '0');
        var throttle = parseInt(el.dataset.lvThrottle || '0');

        var doSend = function() {
          var values = self.extractValues(el);
          var form = {};
          form[el.name || 'value'] = [el.value];
          self.sendEvent(el.dataset.lvChange, el, values, form);
        };

        if (debounce > 0) {
          self.debounce(el, doSend, debounce);
        } else if (throttle > 0) {
          self.throttle(el, doSend, throttle);
        } else {
          doSend();
        }
      }, true);

      // Input events (for live typing)
      document.addEventListener('input', function(e) {
        var el = e.target.closest('[data-lv-change]');
        if (!el) return;

        var debounce = parseInt(el.dataset.lvDebounce || '150');

        self.debounce(el, function() {
          var values = self.extractValues(el);
          var form = {};
          form[el.name || 'value'] = [el.value];
          self.sendEvent(el.dataset.lvChange, el, values, form);
        }, debounce);
      }, true);

      // Keydown events
      document.addEventListener('keydown', function(e) {
        var el = e.target.closest('[data-lv-keydown]');
        if (!el) return;

        var key = el.dataset.lvKey;
        if (key && e.key !== key) return;

        e.preventDefault();
        var values = self.extractValues(el);
        values.key = e.key;
        self.sendEvent(el.dataset.lvKeydown, el, values);
      }, true);

      // Keyup events
      document.addEventListener('keyup', function(e) {
        var el = e.target.closest('[data-lv-keyup]');
        if (!el) return;

        var key = el.dataset.lvKey;
        if (key && e.key !== key) return;

        var values = self.extractValues(el);
        values.key = e.key;
        self.sendEvent(el.dataset.lvKeyup, el, values);
      }, true);

      // Focus events
      document.addEventListener('focus', function(e) {
        var el = e.target.closest('[data-lv-focus]');
        if (!el) return;

        var values = self.extractValues(el);
        self.sendEvent(el.dataset.lvFocus, el, values);
      }, true);

      // Blur events
      document.addEventListener('blur', function(e) {
        var el = e.target.closest('[data-lv-blur]');
        if (!el) return;

        var values = self.extractValues(el);
        self.sendEvent(el.dataset.lvBlur, el, values);
      }, true);
    },

    extractValues: function(el) {
      var values = {};
      for (var key in el.dataset) {
        if (key.startsWith('lvValue')) {
          var name = key.substring(7).toLowerCase();
          if (name.length > 0) {
            name = name.charAt(0).toLowerCase() + name.slice(1);
            // Convert camelCase to kebab-case for the key
            name = name.replace(/([A-Z])/g, function(m) { return '-' + m.toLowerCase(); });
            values[name] = el.dataset[key];
          }
        }
      }
      return values;
    },

    extractFormData: function(form) {
      var data = {};
      var formData = new FormData(form);
      for (var pair of formData.entries()) {
        if (!data[pair[0]]) {
          data[pair[0]] = [];
        }
        data[pair[0]].push(pair[1]);
      }
      return data;
    },

    debounce: function(el, fn, delay) {
      var id = el.id || Math.random().toString(36);
      clearTimeout(this.debounceTimers.get(id));
      this.debounceTimers.set(id, setTimeout(fn, delay));
    },

    throttle: function(el, fn, delay) {
      var id = el.id || Math.random().toString(36);
      if (this.throttleTimers.has(id)) return;
      this.throttleTimers.set(id, true);
      fn();
      setTimeout(function() {
        this.throttleTimers.delete(id);
      }.bind(this), delay);
    },

    setLoading: function(el) {
      var target = el.dataset.lvLoadingTarget
        ? document.querySelector(el.dataset.lvLoadingTarget)
        : el;

      if (el.dataset.lvLoadingClass) {
        target.classList.add(...el.dataset.lvLoadingClass.split(' '));
      }

      target.querySelectorAll('[data-lv-loading-show]').forEach(function(e) {
        e.hidden = false;
      });
      target.querySelectorAll('[data-lv-loading-hide]').forEach(function(e) {
        e.hidden = true;
      });
      target.querySelectorAll('[data-lv-loading-disable]').forEach(function(e) {
        e.disabled = true;
      });
    },

    clearLoading: function(el) {
      if (!el) return;

      var target = el.dataset.lvLoadingTarget
        ? document.querySelector(el.dataset.lvLoadingTarget)
        : el;

      if (el.dataset.lvLoadingClass) {
        target.classList.remove(...el.dataset.lvLoadingClass.split(' '));
      }

      target.querySelectorAll('[data-lv-loading-show]').forEach(function(e) {
        e.hidden = true;
      });
      target.querySelectorAll('[data-lv-loading-hide]').forEach(function(e) {
        e.hidden = false;
      });
      target.querySelectorAll('[data-lv-loading-disable]').forEach(function(e) {
        e.disabled = false;
      });
    },

    setConnectedState: function(connected) {
      var root = document.querySelector('[data-lv]');
      if (root) {
        if (connected) {
          root.classList.add('lv-connected');
          root.classList.remove('lv-disconnected');
        } else {
          root.classList.remove('lv-connected');
          root.classList.add('lv-disconnected');
        }
      }
    },

    getCSRFToken: function() {
      var meta = document.querySelector('meta[name="csrf-token"]');
      if (meta) return meta.getAttribute('content');
      var input = document.querySelector('input[name="_csrf"]');
      if (input) return input.value;
      return '';
    }
  };

  // Auto-connect if data-lv element exists
  document.addEventListener('DOMContentLoaded', function() {
    if (document.querySelector('[data-lv]')) {
      // Check if already configured
      if (!window.MizuLive.conn) {
        window.MizuLive.connect({
          url: window.location.pathname
        });
      }
    }
  });
})();
`
