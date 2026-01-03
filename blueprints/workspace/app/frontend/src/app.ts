// App initialization and vanilla JS functionality
import { api } from './api/client'

export function initApp() {
  // Initialize dropdowns
  initDropdowns()

  // Initialize page title editing
  initPageTitleEditing()

  // Initialize sidebar
  initSidebar()

  // Initialize keyboard shortcuts
  initKeyboardShortcuts()

  // Initialize favorite button
  initFavoriteButton()

  // Initialize search
  initSearch()

  // Initialize settings
  initSettings()

  // Initialize quick actions
  initQuickActions()

  // Initialize theme
  initTheme()
}

// Dropdown functionality
function initDropdowns() {
  function setupDropdown(toggleId: string, dropdownId: string) {
    const toggle = document.getElementById(toggleId)
    const dropdown = document.getElementById(dropdownId)
    if (!toggle || !dropdown) return

    toggle.addEventListener('click', (e) => {
      e.stopPropagation()
      dropdown.classList.toggle('open')
    })

    document.addEventListener('click', () => {
      dropdown.classList.remove('open')
    })
  }

  setupDropdown('workspace-switcher', 'workspace-dropdown')
  setupDropdown('user-menu', 'user-dropdown')
}

// Page title editing
function initPageTitleEditing() {
  const pageTitle = document.getElementById('page-title')
  if (!pageTitle) return

  let debounceTimer: ReturnType<typeof setTimeout>

  pageTitle.addEventListener('input', () => {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      const pageView = document.querySelector('.page-view') as HTMLElement
      const pageId = pageView?.dataset.pageId
      if (pageId) {
        try {
          await api.patch(`/pages/${pageId}`, {
            title: pageTitle.textContent,
          })
        } catch (err) {
          console.error('Failed to update title:', err)
        }
      }
    }, 500)
  })

  // Handle Enter key
  pageTitle.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      pageTitle.blur()
    }
  })
}

// Sidebar functionality
function initSidebar() {
  // Toggle sidebar
  const sidebarToggle = document.getElementById('sidebar-toggle')
  const sidebar = document.querySelector('.sidebar')

  if (sidebarToggle && sidebar) {
    sidebarToggle.addEventListener('click', () => {
      sidebar.classList.toggle('collapsed')
      localStorage.setItem('sidebar-collapsed', sidebar.classList.contains('collapsed').toString())
    })

    // Restore state
    if (localStorage.getItem('sidebar-collapsed') === 'true') {
      sidebar.classList.add('collapsed')
    }
  }

  // Page tree expansion
  document.querySelectorAll('.page-tree-toggle').forEach((toggle) => {
    toggle.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
      const parent = toggle.closest('.page-item')
      parent?.classList.toggle('expanded')
    })
  })

  // Add page button
  const addPageBtn = document.getElementById('add-page-btn')
  if (addPageBtn) {
    addPageBtn.addEventListener('click', async () => {
      const workspaceId = addPageBtn.dataset.workspace
      if (!workspaceId) return

      try {
        const page = await api.post<{ id: string }>('/pages', {
          workspace_id: workspaceId,
          title: 'Untitled',
          parent_type: 'workspace',
          parent_id: workspaceId,
        })
        const workspaceSlug = window.location.pathname.split('/')[2]
        window.location.href = `/w/${workspaceSlug}/p/${page.id}`
      } catch (err) {
        console.error('Failed to create page:', err)
      }
    })
  }
}

// Keyboard shortcuts
function initKeyboardShortcuts() {
  document.addEventListener('keydown', (e) => {
    // Cmd/Ctrl + K: Quick search
    if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
      e.preventDefault()
      const searchInput = document.getElementById('search-input') as HTMLInputElement
      if (searchInput) {
        searchInput.focus()
      } else {
        // Navigate to search page
        const workspaceSlug = window.location.pathname.split('/')[2]
        if (workspaceSlug) {
          window.location.href = `/w/${workspaceSlug}/search`
        }
      }
    }

    // Cmd/Ctrl + N: New page
    if ((e.metaKey || e.ctrlKey) && e.key === 'n') {
      e.preventDefault()
      const addPageBtn = document.getElementById('add-page-btn') as HTMLButtonElement
      if (addPageBtn) {
        addPageBtn.click()
      }
    }

    // Cmd/Ctrl + \\: Toggle sidebar
    if ((e.metaKey || e.ctrlKey) && e.key === '\\') {
      e.preventDefault()
      const sidebarToggle = document.getElementById('sidebar-toggle') as HTMLButtonElement
      if (sidebarToggle) {
        sidebarToggle.click()
      }
    }
  })
}

// Favorite button
function initFavoriteButton() {
  const favoriteBtn = document.getElementById('favorite-btn')
  if (!favoriteBtn) return

  favoriteBtn.addEventListener('click', async () => {
    const pageView = document.querySelector('.page-view') as HTMLElement
    const pageId = pageView?.dataset.pageId
    if (!pageId) return

    try {
      await api.post('/favorites', {
        target_type: 'page',
        target_id: pageId,
      })
      favoriteBtn.classList.toggle('active')
    } catch (err) {
      console.error('Failed to toggle favorite:', err)
    }
  })
}

// Search functionality
function initSearch() {
  const searchInput = document.getElementById('search-input') as HTMLInputElement
  const searchResults = document.getElementById('search-results')

  if (!searchInput || !searchResults) return

  let debounceTimer: ReturnType<typeof setTimeout>

  searchInput.addEventListener('input', () => {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(async () => {
      const query = searchInput.value.trim()
      if (query.length < 2) {
        searchResults.innerHTML = `
          <div class="search-empty">
            <div class="empty-icon">🔍</div>
            <p>Search for pages, databases, and content</p>
          </div>
        `
        return
      }

      searchResults.innerHTML = '<div class="results-loading">Searching...</div>'

      try {
        const workspaceSlug = window.location.pathname.split('/')[2]
        const ws = await api.get<{ id: string }>(`/workspaces/${workspaceSlug}`)
        const results = await api.get<{ id: string; title: string; icon?: string; snippet?: string }[]>(
          `/search?workspace_id=${ws.id}&q=${encodeURIComponent(query)}`
        )

        if (results.length === 0) {
          searchResults.innerHTML = '<div class="search-empty"><p>No results found</p></div>'
          return
        }

        let html = '<div class="results-list">'
        for (const result of results) {
          html += `
            <a href="/w/${workspaceSlug}/p/${result.id}" class="result-item">
              <span class="result-icon">${result.icon || '📄'}</span>
              <div class="result-info">
                <div class="result-title">${escapeHtml(result.title)}</div>
                ${result.snippet ? `<div class="result-snippet">${escapeHtml(result.snippet)}</div>` : ''}
              </div>
            </a>
          `
        }
        html += '</div>'
        searchResults.innerHTML = html

        // Save to recent searches
        saveRecentSearch(query)
      } catch (err) {
        console.error('Search failed:', err)
        searchResults.innerHTML = '<div class="search-empty"><p>Search failed</p></div>'
      }
    }, 300)
  })

  // Load recent searches
  loadRecentSearches()
}

function saveRecentSearch(query: string) {
  const key = 'workspace_recent_searches'
  let searches = JSON.parse(localStorage.getItem(key) || '[]')
  searches = searches.filter((s: string) => s !== query)
  searches.unshift(query)
  searches = searches.slice(0, 5)
  localStorage.setItem(key, JSON.stringify(searches))
}

function loadRecentSearches() {
  const container = document.querySelector('#recent-searches .recent-list')
  if (!container) return

  const searches = JSON.parse(localStorage.getItem('workspace_recent_searches') || '[]')
  if (searches.length === 0) {
    container.innerHTML = '<p class="empty-hint">No recent searches</p>'
    return
  }

  let html = ''
  for (const query of searches) {
    html += `<button class="recent-item" data-query="${escapeHtml(query)}">${escapeHtml(query)}</button>`
  }
  container.innerHTML = html

  container.addEventListener('click', (e) => {
    const item = (e.target as HTMLElement).closest('.recent-item') as HTMLElement
    if (item) {
      const searchInput = document.getElementById('search-input') as HTMLInputElement
      if (searchInput) {
        searchInput.value = item.dataset.query || ''
        searchInput.dispatchEvent(new Event('input'))
      }
    }
  })
}

// Settings functionality
function initSettings() {
  // Settings navigation
  const settingsNav = document.querySelectorAll('.settings-nav-item')
  const settingsSections = document.querySelectorAll('.settings-section')

  settingsNav.forEach((navItem) => {
    navItem.addEventListener('click', () => {
      const section = (navItem as HTMLElement).dataset.section

      settingsNav.forEach((n) => n.classList.remove('active'))
      navItem.classList.add('active')

      settingsSections.forEach((s) => {
        s.classList.toggle('active', s.id === `section-${section}`)
      })
    })
  })

  // Workspace settings form
  const workspaceSettingsForm = document.getElementById('workspace-settings-form')
  if (workspaceSettingsForm) {
    workspaceSettingsForm.addEventListener('submit', async (e) => {
      e.preventDefault()
      const workspaceSlug = window.location.pathname.split('/')[2]

      try {
        const ws = await api.get<{ id: string }>(`/workspaces/${workspaceSlug}`)
        await api.put(`/workspaces/${ws.id}`, {
          name: (document.getElementById('ws-name') as HTMLInputElement).value,
          slug: (document.getElementById('ws-slug') as HTMLInputElement).value,
        })

        const newSlug = (document.getElementById('ws-slug') as HTMLInputElement).value
        if (newSlug !== workspaceSlug) {
          window.location.href = `/w/${newSlug}/settings`
        }
      } catch (err) {
        console.error('Failed to update workspace:', err)
        alert('Failed to update workspace')
      }
    })
  }

  // Invite form
  const inviteForm = document.getElementById('invite-form')
  if (inviteForm) {
    inviteForm.addEventListener('submit', async (e) => {
      e.preventDefault()
      const workspaceSlug = window.location.pathname.split('/')[2]

      try {
        const ws = await api.get<{ id: string }>(`/workspaces/${workspaceSlug}`)
        await api.post(`/workspaces/${ws.id}/members`, {
          email: (document.getElementById('invite-email') as HTMLInputElement).value,
          role: (document.getElementById('invite-role') as HTMLSelectElement).value,
        })
        window.location.reload()
      } catch (err) {
        console.error('Failed to invite member:', err)
        alert('Failed to invite member')
      }
    })
  }

  // Delete workspace
  const deleteWorkspaceBtn = document.getElementById('delete-workspace-btn')
  if (deleteWorkspaceBtn) {
    deleteWorkspaceBtn.addEventListener('click', async () => {
      if (!confirm('Are you sure you want to delete this workspace? This action cannot be undone.')) {
        return
      }

      const workspaceSlug = window.location.pathname.split('/')[2]

      try {
        const ws = await api.get<{ id: string }>(`/workspaces/${workspaceSlug}`)
        await api.delete(`/workspaces/${ws.id}`)
        window.location.href = '/app'
      } catch (err) {
        console.error('Failed to delete workspace:', err)
        alert('Failed to delete workspace')
      }
    })
  }
}

// Quick actions
function initQuickActions() {
  const newPageAction = document.getElementById('new-page-action')
  if (newPageAction) {
    newPageAction.addEventListener('click', async () => {
      const workspaceId = newPageAction.dataset.workspace
      if (!workspaceId) return

      try {
        const page = await api.post<{ id: string }>('/pages', {
          workspace_id: workspaceId,
          title: 'Untitled',
          parent_type: 'workspace',
          parent_id: workspaceId,
        })
        const workspaceSlug = window.location.pathname.split('/')[2]
        window.location.href = `/w/${workspaceSlug}/p/${page.id}`
      } catch (err) {
        console.error('Failed to create page:', err)
      }
    })
  }

  const newDatabaseAction = document.getElementById('new-database-action')
  if (newDatabaseAction) {
    newDatabaseAction.addEventListener('click', async () => {
      const workspaceId = newDatabaseAction.dataset.workspace
      if (!workspaceId) return

      try {
        const db = await api.post<{ id: string }>('/databases', {
          workspace_id: workspaceId,
          name: 'Untitled Database',
        })
        const workspaceSlug = window.location.pathname.split('/')[2]
        window.location.href = `/w/${workspaceSlug}/d/${db.id}`
      } catch (err) {
        console.error('Failed to create database:', err)
      }
    })
  }
}

// Theme functionality
function initTheme() {
  function setTheme(theme: 'light' | 'dark') {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem('theme', theme)
  }

  // Theme toggle button
  const themeToggle = document.getElementById('theme-toggle')
  if (themeToggle) {
    themeToggle.addEventListener('click', () => {
      const current = document.documentElement.getAttribute('data-theme') || 'light'
      const next = current === 'dark' ? 'light' : 'dark'
      setTheme(next)
    })
  }
}

// Helper functions
function escapeHtml(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}
