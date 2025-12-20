export function createApp() {
  return {
    route: window.location.pathname,
    message: '',
    loading: true,

    init() {
      // Handle browser back/forward
      window.addEventListener('popstate', () => {
        this.route = window.location.pathname
        this.onRouteChange()
      })

      // Initial route load
      this.onRouteChange()
    },

    navigate(path: string) {
      if (this.route === path) return
      window.history.pushState({}, '', path)
      this.route = path
      this.onRouteChange()
    },

    onRouteChange() {
      if (this.route === '/') {
        this.fetchMessage()
      }
    },

    async fetchMessage() {
      this.loading = true
      try {
        const res = await fetch('/api/hello')
        const data = await res.json()
        this.message = data.message
      } catch {
        this.message = 'Failed to load message'
      } finally {
        this.loading = false
      }
    }
  }
}
