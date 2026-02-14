import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"

export function useCrawls() {
  return useQuery({
    queryKey: ["crawls"],
    queryFn: api.getCrawls,
    staleTime: 10 * 60 * 1000,
  })
}

export function useCrawl(id: string) {
  return useQuery({
    queryKey: ["crawl", id],
    queryFn: () => api.getCrawl(id),
    enabled: !!id,
  })
}

export function useCrawlFiles(id: string, page = 0) {
  return useQuery({
    queryKey: ["crawl-files", id, page],
    queryFn: () => api.getCrawlFiles(id, page),
    enabled: !!id,
  })
}

export function useCrawlCDX(id: string, prefix: string, page = 0) {
  return useQuery({
    queryKey: ["crawl-cdx", id, prefix, page],
    queryFn: () => api.getCrawlCDX(id, prefix, page),
    enabled: !!id,
  })
}

export function useDomains(crawl?: string, page = 0) {
  return useQuery({
    queryKey: ["domains", crawl, page],
    queryFn: () => api.getDomains(crawl, page),
  })
}

export function useDomain(domain: string, crawl?: string, page = 0) {
  return useQuery({
    queryKey: ["domain", domain, crawl, page],
    queryFn: () => api.getDomain(domain, crawl, page),
    enabled: !!domain,
  })
}

export function useURLLookup(url: string, crawl?: string) {
  return useQuery({
    queryKey: ["url", url, crawl],
    queryFn: () => api.lookupURL(url, crawl),
    enabled: !!url,
  })
}

export function useWARCView(
  file: string,
  offset: number,
  length: number
) {
  return useQuery({
    queryKey: ["view", file, offset, length],
    queryFn: () => api.getView(file, offset, length),
    enabled: !!file && offset >= 0 && length > 0,
  })
}

// Alias for backward compatibility
export const useView = useWARCView

export function useWAT(file: string, offset: number, length: number) {
  return useQuery({
    queryKey: ["wat", file, offset, length],
    queryFn: () => api.getWAT(file, offset, length),
    enabled: !!file && offset >= 0 && length > 0,
  })
}

export function useWET(file: string, offset: number, length: number) {
  return useQuery({
    queryKey: ["wet", file, offset, length],
    queryFn: () => api.getWET(file, offset, length),
    enabled: !!file && offset >= 0 && length > 0,
  })
}

export function useRobots(domain: string, crawl?: string) {
  return useQuery({
    queryKey: ["robots", domain, crawl],
    queryFn: () => api.getRobots(domain, crawl),
    enabled: !!domain,
  })
}

export function useStats(crawl?: string) {
  return useQuery({
    queryKey: ["stats", crawl],
    queryFn: () => api.getStats(crawl),
  })
}

export function useStatsTrends() {
  return useQuery({
    queryKey: ["stats-trends"],
    queryFn: api.getStatsTrends,
    staleTime: 30 * 60 * 1000,
  })
}

export function useGraph(crawl?: string, type?: string) {
  return useQuery({
    queryKey: ["graph", crawl, type],
    queryFn: () => api.getGraph(crawl, type),
  })
}

export function useGraphRank(domain: string) {
  return useQuery({
    queryKey: ["graph-rank", domain],
    queryFn: () => api.getGraphRank(domain),
    enabled: !!domain,
  })
}

export function useAnnotations(domain: string) {
  return useQuery({
    queryKey: ["annotations", domain],
    queryFn: () => api.getAnnotations(domain),
    enabled: !!domain,
  })
}

export function useNews(date?: string, page = 0) {
  return useQuery({
    queryKey: ["news", date, page],
    queryFn: () => api.getNews(date, page),
  })
}

export function useNewsDates() {
  return useQuery({
    queryKey: ["news-dates"],
    queryFn: api.getNewsDates,
    staleTime: 30 * 60 * 1000,
  })
}

export function useSearch(q: string, type?: string) {
  return useQuery({
    queryKey: ["search", q, type],
    queryFn: () => api.search(q, type),
    enabled: q.length > 0,
  })
}

export function usePopular() {
  return useQuery({
    queryKey: ["popular"],
    queryFn: api.getPopular,
    staleTime: 10 * 60 * 1000,
  })
}

export function useUsage() {
  return useQuery({
    queryKey: ["usage"],
    queryFn: api.getUsage,
    staleTime: 60 * 1000,
  })
}
