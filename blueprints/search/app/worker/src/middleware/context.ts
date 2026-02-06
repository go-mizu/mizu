/**
 * Context middleware for dependency injection.
 * Initializes services once per worker instance using lazy singleton pattern.
 * Engine secrets are resolved from both env vars and KV storage.
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
 * Cache for service containers keyed by KV namespace.
 */
const containerCache = new WeakMap<KVNamespace, ServiceContainer>();

/**
 * Whether KV secrets have been loaded into the container.
 */
const kvSecretsLoaded = new WeakSet<KVNamespace>();

/**
 * Extract engine secrets from Cloudflare env bindings.
 */
function extractEngineSecrets(env: Record<string, unknown>): Record<string, string> {
  const secrets: Record<string, string> = {};
  if (typeof env.JINA_API_KEY === 'string' && env.JINA_API_KEY) {
    secrets['jina_api_key'] = env.JINA_API_KEY;
  }
  return secrets;
}

/**
 * Resolve engine secrets from KV storage (for keys stored via key rotation API).
 */
async function resolveKVSecrets(kv: KVNamespace): Promise<Record<string, string>> {
  const secrets: Record<string, string> = {};

  try {
    const jinaRaw = await kv.get('secrets:jina_api_key');
    if (jinaRaw) {
      try {
        const data = JSON.parse(jinaRaw) as { key: string };
        if (data.key) secrets['jina_api_key'] = data.key;
      } catch {
        if (jinaRaw.startsWith('jina_')) secrets['jina_api_key'] = jinaRaw;
      }
    }
  } catch {
    // KV read failed, skip
  }

  return secrets;
}

/**
 * Create or retrieve the service container for a given KV namespace.
 */
export function getServiceContainer(kv: KVNamespace, engineSecrets: Record<string, string> = {}): ServiceContainer {
  let container = containerCache.get(kv);
  if (container) {
    return container;
  }

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
    knowledge,
    engineSecrets
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

  containerCache.set(kv, container);
  return container;
}

/**
 * Context middleware that injects the service container into the request context.
 * On first invocation, resolves engine secrets from both env and KV.
 */
export const contextMiddleware = createMiddleware<{
  Bindings: Env;
  Variables: { services: ServiceContainer };
}>(async (c, next) => {
  // Extract env secrets
  const envSecrets = extractEngineSecrets(c.env as unknown as Record<string, unknown>);

  // Get or create the container
  const services = getServiceContainer(c.env.SEARCH_KV, envSecrets);

  // On first request, also resolve KV-stored secrets (async)
  if (!kvSecretsLoaded.has(c.env.SEARCH_KV)) {
    kvSecretsLoaded.add(c.env.SEARCH_KV);
    const kvSecrets = await resolveKVSecrets(c.env.SEARCH_KV);
    // KV secrets override env secrets
    if (Object.keys(kvSecrets).length > 0) {
      const merged = { ...envSecrets, ...kvSecrets };
      services.search.updateEngineSecrets(merged);
    }
  }

  c.set('services', services);
  return next();
});

/**
 * Helper to get services directly from the KV namespace.
 */
export function getServices(kv: KVNamespace, env?: Record<string, unknown>): ServiceContainer {
  const secrets = env ? extractEngineSecrets(env) : {};
  return getServiceContainer(kv, secrets);
}
