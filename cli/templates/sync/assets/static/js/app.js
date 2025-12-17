/**
 * TodoApp - Main application controller
 *
 * Integrates:
 * - SyncClient for offline-first data sync
 * - LiveClient for real-time updates
 * - DOM manipulation for UI updates
 */
class TodoApp {
    constructor(options) {
        this.scope = options.scope;

        // Initialize sync client
        this.sync = new SyncClient({
            baseURL: options.baseURL,
            scope: options.scope,
            cursor: options.cursor,
            onSync: (cursor) => this.handleSync(cursor),
            onError: (err) => this.handleError(err),
            onOnline: () => this.setOnlineStatus(true),
            onOffline: () => this.setOnlineStatus(false)
        });

        // Initialize live client
        this.live = new LiveClient({
            url: options.wsURL,
            onMessage: (msg) => this.handleLiveMessage(msg),
            onConnect: () => this.handleLiveConnect(),
            onDisconnect: () => this.handleLiveDisconnect()
        });

        // Connect to live server
        this.live.connect();

        // Initial sync
        this.sync.sync();
    }

    /**
     * Handle form submission
     */
    handleSubmit(event) {
        event.preventDefault();

        const form = event.target;
        const input = form.querySelector('input[name="title"]');
        const title = input.value.trim();

        if (!title) return false;

        // Create todo
        this.createTodo(title);

        // Clear input
        input.value = '';

        return false;
    }

    /**
     * Create a new todo
     */
    createTodo(title) {
        const id = this.generateId();
        const todo = {
            id: id,
            title: title,
            done: false,
            created_at: new Date().toISOString()
        };

        // Optimistic update
        this.sync.set('todo', id, todo);
        this.addTodoToDOM(todo);
        this.updateCount();

        // Queue mutation
        this.sync.mutate('todo.create', { id: id, title: title });
    }

    /**
     * Toggle todo done status
     */
    toggleTodo(id) {
        const todo = this.sync.get('todo', id);
        if (!todo) return;

        // Optimistic update
        todo.done = !todo.done;
        this.sync.set('todo', id, todo);
        this.updateTodoInDOM(todo);

        // Queue mutation
        this.sync.mutate('todo.toggle', { id: id });
    }

    /**
     * Delete a todo
     */
    deleteTodo(id) {
        // Optimistic update
        this.sync.delete('todo', id);
        this.removeTodoFromDOM(id);
        this.updateCount();

        // Queue mutation
        this.sync.mutate('todo.delete', { id: id });
    }

    // DOM manipulation

    addTodoToDOM(todo) {
        const list = document.getElementById('todo-list');
        const empty = list.querySelector('.empty-state');
        if (empty) {
            empty.remove();
        }

        const html = this.renderTodoItem(todo);
        const temp = document.createElement('div');
        temp.innerHTML = html;
        const element = temp.firstElementChild;
        element.classList.add('todo-item-enter');
        list.appendChild(element);
    }

    updateTodoInDOM(todo) {
        const element = document.querySelector(`.todo-item[data-id="${todo.id}"]`);
        if (!element) return;

        const checkbox = element.querySelector('input[type="checkbox"]');
        const title = element.querySelector('.todo-title');

        checkbox.checked = todo.done;
        if (todo.done) {
            element.classList.add('done');
        } else {
            element.classList.remove('done');
        }
        title.textContent = todo.title;
    }

    removeTodoFromDOM(id) {
        const element = document.querySelector(`.todo-item[data-id="${id}"]`);
        if (!element) return;

        element.classList.add('todo-item-leave');
        setTimeout(() => {
            element.remove();
            this.checkEmpty();
        }, 200);
    }

    checkEmpty() {
        const list = document.getElementById('todo-list');
        const items = list.querySelectorAll('.todo-item');
        if (items.length === 0) {
            list.innerHTML = '<p class="empty-state">No todos yet. Add one above!</p>';
        }
    }

    updateCount() {
        const todos = this.sync.all('todo');
        const active = todos.filter(t => !t.done).length;
        const counter = document.getElementById('active-count');
        if (counter) {
            counter.textContent = active;
        }
    }

    renderTodoItem(todo) {
        const done = todo.done ? 'done' : '';
        const checked = todo.done ? 'checked' : '';
        return `
            <div class="todo-item ${done}" data-id="${todo.id}">
                <input type="checkbox" ${checked} onchange="todoApp.toggleTodo('${todo.id}')">
                <span class="todo-title">${this.escapeHtml(todo.title)}</span>
                <button class="delete-btn" onclick="todoApp.deleteTodo('${todo.id}')" aria-label="Delete">
                    &times;
                </button>
            </div>
        `;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Event handlers

    handleSync(cursor) {
        console.log('Synced to cursor:', cursor);
        this.refreshUI();
    }

    handleError(err) {
        console.error('Sync error:', err);
    }

    handleLiveMessage(msg) {
        if (msg.type === 'sync' && msg.topic === `sync:${this.scope}`) {
            // Server notified us of changes, trigger sync
            this.sync.sync();
        }
    }

    handleLiveConnect() {
        console.log('Live connected');
        // Subscribe to sync notifications
        this.live.subscribe(`sync:${this.scope}`);
        this.setOnlineStatus(true);
    }

    handleLiveDisconnect() {
        console.log('Live disconnected');
    }

    setOnlineStatus(online) {
        const status = document.getElementById('sync-status');
        if (!status) return;

        const indicator = status.querySelector('.status-indicator');
        const text = status.querySelector('.status-text');

        if (online) {
            indicator.classList.remove('offline', 'syncing');
            indicator.classList.add('online');
            text.textContent = 'Synced';
        } else {
            indicator.classList.remove('online', 'syncing');
            indicator.classList.add('offline');
            text.textContent = 'Offline';
        }
    }

    setSyncingStatus() {
        const status = document.getElementById('sync-status');
        if (!status) return;

        const indicator = status.querySelector('.status-indicator');
        const text = status.querySelector('.status-text');

        indicator.classList.remove('online', 'offline');
        indicator.classList.add('syncing');
        text.textContent = 'Syncing...';
    }

    refreshUI() {
        // Refresh the todo list from sync store
        const list = document.getElementById('todo-list');
        const todos = this.sync.all('todo');

        if (todos.length === 0) {
            list.innerHTML = '<p class="empty-state">No todos yet. Add one above!</p>';
        } else {
            // Sort by created_at
            todos.sort((a, b) => new Date(a.created_at) - new Date(b.created_at));
            list.innerHTML = todos.map(t => this.renderTodoItem(t)).join('');
        }

        this.updateCount();
    }

    generateId() {
        const bytes = new Uint8Array(16);
        crypto.getRandomValues(bytes);
        return Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join('');
    }
}
