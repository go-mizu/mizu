/**
 * SyncClient - Offline-first data synchronization client
 *
 * Implements the Mizu sync protocol with:
 * - Mutation queue with localStorage persistence
 * - Optimistic updates
 * - Cursor-based incremental sync
 * - Conflict resolution via server re-sync
 */
class SyncClient {
    constructor(options) {
        this.baseURL = options.baseURL || '/_sync';
        this.scope = options.scope || '_default';
        this.cursor = options.cursor || 0;
        this.onSync = options.onSync || (() => {});
        this.onError = options.onError || console.error;
        this.onOnline = options.onOnline || (() => {});
        this.onOffline = options.onOffline || (() => {});

        // Local store for entities
        this.store = new Map();

        // Mutation queue
        this.queue = [];
        this.clientId = this.loadClientId();
        this.seq = 0;

        // State
        this.online = navigator.onLine;
        this.syncing = false;

        // Load persisted state
        this.loadState();

        // Listen for online/offline events
        window.addEventListener('online', () => this.setOnline(true));
        window.addEventListener('offline', () => this.setOnline(false));
    }

    /**
     * Queue a mutation for sync
     */
    mutate(name, args) {
        const mutation = {
            id: this.generateId(),
            name: name,
            scope: this.scope,
            client: this.clientId,
            seq: ++this.seq,
            args: args,
            created_at: new Date().toISOString()
        };

        this.queue.push(mutation);
        this.saveState();

        // Schedule sync
        this.scheduleSync();

        return mutation.id;
    }

    /**
     * Sync with server
     */
    async sync() {
        if (this.syncing) return;
        if (!this.online) return;

        this.syncing = true;

        try {
            // Push pending mutations
            await this.push();

            // Pull changes
            await this.pull();

            this.onSync(this.cursor);
        } catch (err) {
            this.onError(err);
        } finally {
            this.syncing = false;
        }
    }

    /**
     * Push pending mutations to server
     */
    async push() {
        if (this.queue.length === 0) return;

        const response = await fetch(`${this.baseURL}/push`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mutations: this.queue })
        });

        if (!response.ok) {
            throw new Error(`Push failed: ${response.status}`);
        }

        const data = await response.json();

        // Process results
        let maxCursor = this.cursor;
        const completed = [];

        for (let i = 0; i < data.results.length && i < this.queue.length; i++) {
            const result = data.results[i];
            const mutation = this.queue[i];

            if (result.ok) {
                completed.push(mutation.id);
                if (result.cursor > maxCursor) {
                    maxCursor = result.cursor;
                }
            } else if (result.code === 'conflict') {
                // Need full re-sync
                await this.snapshot();
                return;
            }
        }

        // Remove completed mutations
        this.queue = this.queue.filter(m => !completed.includes(m.id));
        this.cursor = maxCursor;
        this.saveState();
    }

    /**
     * Pull changes from server
     */
    async pull() {
        let hasMore = true;

        while (hasMore) {
            const response = await fetch(`${this.baseURL}/pull`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    scope: this.scope,
                    cursor: this.cursor,
                    limit: 100
                })
            });

            if (response.status === 410) {
                // Cursor too old, need full sync
                await this.snapshot();
                return;
            }

            if (!response.ok) {
                throw new Error(`Pull failed: ${response.status}`);
            }

            const data = await response.json();

            // Apply changes
            for (const change of data.changes) {
                this.applyChange(change);
                if (change.cursor > this.cursor) {
                    this.cursor = change.cursor;
                }
            }

            hasMore = data.has_more;
        }

        this.saveState();
    }

    /**
     * Get full snapshot from server
     */
    async snapshot() {
        const response = await fetch(`${this.baseURL}/snapshot`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ scope: this.scope })
        });

        if (!response.ok) {
            throw new Error(`Snapshot failed: ${response.status}`);
        }

        const data = await response.json();

        // Replace local store
        this.store.clear();
        for (const [entity, items] of Object.entries(data.data)) {
            for (const [id, bytes] of Object.entries(items)) {
                const key = `${entity}:${id}`;
                // Decode base64 JSON
                const json = atob(bytes);
                const value = JSON.parse(json);
                this.store.set(key, value);
            }
        }

        this.cursor = data.cursor;
        this.saveState();
    }

    /**
     * Apply a change to local store
     */
    applyChange(change) {
        const key = `${change.entity}:${change.id}`;

        switch (change.op) {
            case 'create':
            case 'update':
                if (change.data) {
                    const json = atob(change.data);
                    const value = JSON.parse(json);
                    this.store.set(key, value);
                }
                break;
            case 'delete':
                this.store.delete(key);
                break;
        }
    }

    /**
     * Get entity from local store
     */
    get(entity, id) {
        const key = `${entity}:${id}`;
        return this.store.get(key);
    }

    /**
     * Get all entities of a type
     */
    all(entity) {
        const prefix = `${entity}:`;
        const result = [];
        for (const [key, value] of this.store) {
            if (key.startsWith(prefix)) {
                result.push(value);
            }
        }
        return result;
    }

    /**
     * Set entity in local store (optimistic)
     */
    set(entity, id, value) {
        const key = `${entity}:${id}`;
        this.store.set(key, value);
    }

    /**
     * Delete entity from local store (optimistic)
     */
    delete(entity, id) {
        const key = `${entity}:${id}`;
        this.store.delete(key);
    }

    // Private methods

    scheduleSync() {
        if (this.syncTimeout) return;
        this.syncTimeout = setTimeout(() => {
            this.syncTimeout = null;
            this.sync();
        }, 100);
    }

    setOnline(online) {
        if (this.online === online) return;
        this.online = online;

        if (online) {
            this.onOnline();
            this.sync();
        } else {
            this.onOffline();
        }
    }

    generateId() {
        const bytes = new Uint8Array(16);
        crypto.getRandomValues(bytes);
        return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
    }

    loadClientId() {
        let id = localStorage.getItem('sync_client_id');
        if (!id) {
            id = this.generateId();
            localStorage.setItem('sync_client_id', id);
        }
        return id;
    }

    loadState() {
        try {
            const state = localStorage.getItem(`sync_state_${this.scope}`);
            if (state) {
                const parsed = JSON.parse(state);
                this.cursor = parsed.cursor || 0;
                this.queue = parsed.queue || [];
                this.seq = parsed.seq || 0;
            }
        } catch (e) {
            console.error('Failed to load sync state:', e);
        }
    }

    saveState() {
        try {
            const state = {
                cursor: this.cursor,
                queue: this.queue,
                seq: this.seq
            };
            localStorage.setItem(`sync_state_${this.scope}`, JSON.stringify(state));
        } catch (e) {
            console.error('Failed to save sync state:', e);
        }
    }
}
