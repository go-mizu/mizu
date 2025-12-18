/**
 * LiveView - Client-side live view implementation
 *
 * Features:
 * - Automatic event binding via data-live-* attributes
 * - DOM patching for efficient updates
 * - Mount/unmount lifecycle
 */
class LiveView {
    constructor(options) {
        this.socket = options.socket;
        this.viewId = options.viewId;
        this.element = options.element;
        this.topic = `view:${this.viewId}`;
        this.sessionId = null;
        this.mounted = false;
        this.refCounter = 0;
        this.pendingRefs = new Map();
        this.boundHandlers = [];

        // Register message handler
        this.socket.register(this.topic, (msg) => this.handleMessage(msg));
    }

    /**
     * Mount the live view
     */
    mount(params = {}) {
        const ref = this.nextRef();
        return new Promise((resolve, reject) => {
            this.pendingRefs.set(ref, { resolve, reject, type: 'mount' });
            this.socket.send({
                type: 'mount',
                topic: this.topic,
                ref: ref,
                body: JSON.stringify({
                    viewId: this.viewId,
                    params: params
                })
            });
        }).then((result) => {
            this.sessionId = result.sessionId;
            this.mounted = true;
            this.bindEvents();
            return result;
        });
    }

    /**
     * Unmount the live view
     */
    unmount() {
        this.socket.unregister(this.topic);
        this.unbindEvents();
        this.mounted = false;
    }

    /**
     * Send an event to the server
     */
    sendEvent(event) {
        if (!this.mounted) {
            console.warn('View not mounted');
            return;
        }

        this.socket.send({
            type: 'event',
            topic: this.topic,
            ref: this.nextRef(),
            body: JSON.stringify(event)
        });
    }

    /**
     * Handle incoming messages
     */
    handleMessage(msg) {
        // Handle pending refs (mount response, etc.)
        if (msg.ref && this.pendingRefs.has(msg.ref)) {
            const pending = this.pendingRefs.get(msg.ref);
            this.pendingRefs.delete(msg.ref);

            if (msg.type === 'error') {
                const body = typeof msg.body === 'string' ? JSON.parse(msg.body) : msg.body;
                pending.reject(new Error(body.message || 'Unknown error'));
            } else {
                const body = typeof msg.body === 'string' ? JSON.parse(msg.body) : msg.body;
                pending.resolve(body);
            }
            return;
        }

        // Handle patches
        if (msg.type === 'patch') {
            const body = typeof msg.body === 'string' ? JSON.parse(msg.body) : msg.body;
            if (body.patches) {
                this.applyPatches(body.patches);
            }
        }
    }

    /**
     * Apply DOM patches
     */
    applyPatches(patches) {
        for (const patch of patches) {
            this.applyPatch(patch);
        }
    }

    /**
     * Apply a single DOM patch
     */
    applyPatch(patch) {
        const el = document.querySelector(patch.target);
        if (!el) {
            console.warn('Patch target not found:', patch.target);
            return;
        }

        switch (patch.op) {
            case 'replace':
                el.innerHTML = patch.html;
                break;
            case 'outer':
                el.outerHTML = patch.html;
                this.rebindEvents();
                break;
            case 'append':
                el.insertAdjacentHTML('beforeend', patch.html);
                break;
            case 'prepend':
                el.insertAdjacentHTML('afterbegin', patch.html);
                break;
            case 'remove':
                el.remove();
                break;
            case 'attr':
                el.setAttribute(patch.attr, patch.value);
                break;
            case 'removeAttr':
                el.removeAttribute(patch.attr);
                break;
            case 'addClass':
                el.classList.add(patch.value);
                break;
            case 'removeClass':
                el.classList.remove(patch.value);
                break;
            case 'toggleClass':
                el.classList.toggle(patch.value);
                break;
            case 'show':
                el.hidden = false;
                break;
            case 'hide':
                el.hidden = true;
                break;
            case 'focus':
                el.focus();
                break;
            case 'blur':
                el.blur();
                break;
            case 'redirect':
                window.location.href = patch.value;
                break;
            case 'reload':
                window.location.reload();
                break;
            default:
                console.warn('Unknown patch operation:', patch.op);
        }
    }

    /**
     * Bind event handlers to data-live-* elements
     */
    bindEvents() {
        this.unbindEvents();

        // Click events
        this.element.querySelectorAll('[data-live-click]').forEach(el => {
            const handler = (e) => {
                e.preventDefault();
                this.sendEvent({
                    type: 'click',
                    target: el.dataset.liveClick,
                    value: el.dataset.value || '',
                    data: this.getDataAttributes(el)
                });
            };
            el.addEventListener('click', handler);
            this.boundHandlers.push({ el, event: 'click', handler });
        });

        // Input events with optional debounce
        this.element.querySelectorAll('[data-live-input]').forEach(el => {
            const debounce = parseInt(el.dataset.liveDebounce) || 0;
            let timeout;
            const handler = (e) => {
                clearTimeout(timeout);
                timeout = setTimeout(() => {
                    this.sendEvent({
                        type: 'input',
                        target: el.dataset.liveInput,
                        value: el.value,
                        data: this.getDataAttributes(el)
                    });
                }, debounce);
            };
            el.addEventListener('input', handler);
            this.boundHandlers.push({ el, event: 'input', handler });
        });

        // Change events
        this.element.querySelectorAll('[data-live-change]').forEach(el => {
            const handler = (e) => {
                this.sendEvent({
                    type: 'change',
                    target: el.dataset.liveChange,
                    value: el.type === 'checkbox' ? el.checked : el.value,
                    data: this.getDataAttributes(el)
                });
            };
            el.addEventListener('change', handler);
            this.boundHandlers.push({ el, event: 'change', handler });
        });

        // Form submit events
        this.element.querySelectorAll('[data-live-submit]').forEach(form => {
            const handler = (e) => {
                e.preventDefault();
                const formData = new FormData(form);
                this.sendEvent({
                    type: 'submit',
                    target: form.dataset.liveSubmit,
                    data: Object.fromEntries(formData)
                });
            };
            form.addEventListener('submit', handler);
            this.boundHandlers.push({ el: form, event: 'submit', handler });
        });

        // Key events
        this.element.querySelectorAll('[data-live-keydown]').forEach(el => {
            const targetKey = el.dataset.liveKey;
            const handler = (e) => {
                if (!targetKey || e.key === targetKey) {
                    this.sendEvent({
                        type: 'keydown',
                        target: el.dataset.liveKeydown,
                        value: el.value,
                        data: { key: e.key, ...this.getDataAttributes(el) }
                    });
                }
            };
            el.addEventListener('keydown', handler);
            this.boundHandlers.push({ el, event: 'keydown', handler });
        });

        // Focus events
        this.element.querySelectorAll('[data-live-focus]').forEach(el => {
            const handler = (e) => {
                this.sendEvent({
                    type: 'focus',
                    target: el.dataset.liveFocus,
                    value: el.value || '',
                    data: this.getDataAttributes(el)
                });
            };
            el.addEventListener('focus', handler);
            this.boundHandlers.push({ el, event: 'focus', handler });
        });

        // Blur events
        this.element.querySelectorAll('[data-live-blur]').forEach(el => {
            const handler = (e) => {
                this.sendEvent({
                    type: 'blur',
                    target: el.dataset.liveBlur,
                    value: el.value || '',
                    data: this.getDataAttributes(el)
                });
            };
            el.addEventListener('blur', handler);
            this.boundHandlers.push({ el, event: 'blur', handler });
        });
    }

    /**
     * Unbind all event handlers
     */
    unbindEvents() {
        for (const { el, event, handler } of this.boundHandlers) {
            el.removeEventListener(event, handler);
        }
        this.boundHandlers = [];
    }

    /**
     * Rebind events after outer replace
     */
    rebindEvents() {
        this.element = document.querySelector(`[data-live-view="${this.viewId}"]`);
        if (this.element) {
            this.bindEvents();
        }
    }

    /**
     * Get data-* attributes from an element (excluding data-live-*)
     */
    getDataAttributes(el) {
        const data = {};
        for (const attr of el.attributes) {
            if (attr.name.startsWith('data-') && !attr.name.startsWith('data-live-')) {
                const key = attr.name.slice(5).replace(/-./g, x => x[1].toUpperCase());
                // Try to parse as number
                const value = attr.value;
                const num = parseFloat(value);
                data[key] = !isNaN(num) && isFinite(num) ? num : value;
            }
        }
        return data;
    }

    /**
     * Generate unique ref
     */
    nextRef() {
        return `${this.viewId}_${++this.refCounter}`;
    }
}
