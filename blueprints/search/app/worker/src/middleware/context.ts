/**
 * Context middleware for dependency injection.
 * Initializes services once per worker instance using lazy singleton pattern.
 */

import { createMiddleware } from 'hono/factory';
import { CacheStore } from '../store/cache';
import { KVStore } from '../store/kv';
import { createDefaultMetaSearch, type MetaSearch } from '../engines/metasearch';
import { SearchService } from '../services/search';
import { BangService } from '../services/bang';
import { InstantService } from '../services/instant';
import { KnowledgeService } from '../services/knowledge';
import { SuggestService } from '../services/suggest';
import type { Env } from '../types';

/**
 * Service container holding all initialized services.
 */
export interface ServiceContainer {
  readonly cache: CacheStore;
  readonly kv: KVStore;
  readonly metasearch: MetaSearch;
  readonly search: SearchService;
  readonly bang: BangService;
  readonly instant: InstantService;
  readonly knowledge: KnowledgeService;
  readonly suggest: SuggestService;
}

/**
 * Cache for service containers, keyed by KV namespace.
 * Using WeakMap to allow garbage collection when KV namespace is no longer referenced.
 */
const containerCache = new WeakMap<KVNamespace, ServiceContainer>();

/**
 * Create or retrieve the service container for a given KV namespace.
 * Services are lazily initialized on first access.
 */
export function getServiceContainer(kv: KVNamespace): ServiceContainer {
  // Check cache first
  let container = containerCache.get(kv);
  if (container) {
    return container;
  }

  // Initialize all services
  const cache = new CacheStore(kv);
  const kvStore = new KVStore(kv);
  const metasearch = createDefaultMetaSearch();
  const bang = new BangService(kvStore);
  const instant = new InstantService(cache);
  const knowledge = new KnowledgeService(cache);
  const suggest = new SuggestService(cache);
  const search = new SearchService(
    metasearch,
    cache,
    kvStore,
    bang,
    instant,
    knowledge
  );

  container = {
    cache,
    kv: kvStore,
    metasearch,
    search,
    bang,
    instant,
    knowledge,
    suggest,
  };

  // Store in cache for future requests
  containerCache.set(kv, container);

  return container;
}

/**
 * Context middleware that injects the service container into the request context.
 * Services are initialized once per worker instance and reused across requests.
 *
 * @example
 * ```typescript
 * // In routes:
 * app.get('/search', async (c) => {
 *   const services = getServices(c.env.SEARCH_KV);
 *   const results = await services.search.search(query, options);
 *   return c.json(results);
 * });
 * ```
 */
export const contextMiddleware = createMiddleware<{
  Bindings: Env;
  Variables: { services: ServiceContainer };
}>(async (c, next) => {
  const services = getServiceContainer(c.env.SEARCH_KV);
  c.set('services', services);
  return next();
});

/**
 * Helper to get services directly from the KV namespace.
 * Use this when you don't have access to the middleware-injected context.
 */
export function getServices(kv: KVNamespace): ServiceContainer {
  return getServiceContainer(kv);
}
