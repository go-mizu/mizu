export type PageRenderer = (params: Record<string, string>, query: Record<string, string>) => void;

interface Route {
  pattern: string;
  segments: string[];
  renderer: PageRenderer;
}

export class Router {
  private routes: Route[] = [];
  private currentPath = '';
  private notFoundRenderer: PageRenderer | null = null;

  addRoute(pattern: string, renderer: PageRenderer): void {
    const segments = pattern.split('/').filter(Boolean);
    this.routes.push({ pattern, segments, renderer });
  }

  setNotFound(renderer: PageRenderer): void {
    this.notFoundRenderer = renderer;
  }

  navigate(path: string, replace = false): void {
    if (path === this.currentPath) return;
    if (replace) {
      history.replaceState(null, '', path);
    } else {
      history.pushState(null, '', path);
    }
    this.resolve();
  }

  start(): void {
    window.addEventListener('popstate', () => this.resolve());

    document.addEventListener('click', (e) => {
      const anchor = (e.target as HTMLElement).closest('a[data-link]');
      if (anchor) {
        e.preventDefault();
        const href = anchor.getAttribute('href');
        if (href) this.navigate(href);
      }
    });

    this.resolve();
  }

  getCurrentPath(): string {
    return this.currentPath;
  }

  private resolve(): void {
    const url = new URL(window.location.href);
    const path = url.pathname;
    const query = parseQuery(url.search);

    this.currentPath = path + url.search;

    for (const route of this.routes) {
      const params = matchRoute(route.segments, path);
      if (params !== null) {
        route.renderer(params, query);
        return;
      }
    }

    if (this.notFoundRenderer) {
      this.notFoundRenderer({}, query);
    }
  }
}

function matchRoute(routeSegments: string[], path: string): Record<string, string> | null {
  const pathSegments = path.split('/').filter(Boolean);

  if (routeSegments.length === 0 && pathSegments.length === 0) {
    return {};
  }

  if (routeSegments.length !== pathSegments.length) {
    return null;
  }

  const params: Record<string, string> = {};

  for (let i = 0; i < routeSegments.length; i++) {
    const routeSeg = routeSegments[i];
    const pathSeg = pathSegments[i];

    if (routeSeg.startsWith(':')) {
      params[routeSeg.slice(1)] = decodeURIComponent(pathSeg);
    } else if (routeSeg !== pathSeg) {
      return null;
    }
  }

  return params;
}

function parseQuery(search: string): Record<string, string> {
  const query: Record<string, string> = {};
  const params = new URLSearchParams(search);
  params.forEach((value, key) => {
    query[key] = value;
  });
  return query;
}

export function navigate(path: string): void {
  window.dispatchEvent(new CustomEvent('router:navigate', { detail: { path } }));
}
