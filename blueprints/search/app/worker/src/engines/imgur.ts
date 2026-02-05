/**
 * Imgur Image Search Engine adapter.
 *
 * Searches Imgur's gallery for images matching the query.
 * Uses public endpoints without API key.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';
import { decodeHtmlEntities } from '../lib/html-parser';

// ========== Imgur Data Types ==========

interface ImgurImage {
  id: string;
  title?: string;
  description?: string;
  datetime?: number;
  type?: string;
  animated?: boolean;
  width?: number;
  height?: number;
  size?: number;
  views?: number;
  bandwidth?: number;
  vote?: string | null;
  favorite?: boolean;
  nsfw?: boolean | null;
  section?: string;
  account_url?: string;
  account_id?: number | null;
  is_ad?: boolean;
  in_most_viral?: boolean;
  has_sound?: boolean;
  tags?: string[];
  ad_type?: number;
  ad_url?: string;
  edited?: string;
  in_gallery?: boolean;
  link?: string;
  mp4?: string;
  gifv?: string;
  hls?: string;
  mp4_size?: number;
  looping?: boolean;
  comment_count?: number;
  favorite_count?: number;
  ups?: number;
  downs?: number;
  points?: number;
  score?: number;
  images?: ImgurImage[];
  cover?: string;
  cover_width?: number;
  cover_height?: number;
  images_count?: number;
}

interface ImgurSearchResponse {
  data?: ImgurImage[];
  success?: boolean;
  status?: number;
}

// ========== Sort Mapping ==========

const imgurSortMap: Record<string, string> = {
  relevance: 'top',
  newest: 'time',
  viral: 'viral',
};

// ========== Time Window Mapping ==========

const imgurTimeMap: Record<string, string> = {
  day: 'day',
  week: 'week',
  month: 'month',
  year: 'year',
};

export class ImgurEngine implements OnlineEngine {
  name = 'imgur';
  shortcut = 'im';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 50;
  timeout = 8000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Use Imgur's gallery search endpoint
    const sort = 'time';
    const window = imgurTimeMap[params.timeRange] || 'all';
    const page = params.page - 1; // Imgur uses 0-based pagination

    // Build the search URL
    const searchUrl = `https://api.imgur.com/3/gallery/search/${sort}/${window}/${page}?q=${encodeURIComponent(query)}`;

    return {
      url: searchUrl,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        // Imgur's public client ID for anonymous access
        Authorization: 'Client-ID 546c25a59c58ad7',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    try {
      const data: ImgurSearchResponse = JSON.parse(body);

      if (!data.success || !data.data) {
        // Try alternate parsing
        return this.parseHtmlResponse(body, params);
      }

      for (const item of data.data) {
        // Skip NSFW content if safe search is enabled
        if (params.safeSearch >= 1 && item.nsfw) continue;

        // Handle albums (multiple images)
        if (item.images && item.images.length > 0) {
          // Use cover image or first image
          const coverImage = item.images.find((img) => img.id === item.cover) || item.images[0];
          this.processImage(item, coverImage, results, filters);
        } else if (item.link) {
          // Single image
          this.processImage(item, item, results, filters);
        }
      }
    } catch {
      // JSON parse failed, try HTML parsing
      return this.parseHtmlResponse(body, params);
    }

    return results;
  }

  private processImage(
    galleryItem: ImgurImage,
    image: ImgurImage,
    results: EngineResults,
    filters?: EngineParams['imageFilters']
  ): void {
    const width = image.width || galleryItem.cover_width || 0;
    const height = image.height || galleryItem.cover_height || 0;

    // Skip animated content if looking for static images
    if (filters?.type === 'photo' && image.animated) return;
    if (filters?.type === 'animated' && !image.animated) return;

    // Client-side filtering
    if (filters && width && height) {
      if (filters.minWidth && width < filters.minWidth) return;
      if (filters.minHeight && height < filters.minHeight) return;
      if (filters.maxWidth && width > filters.maxWidth) return;
      if (filters.maxHeight && height > filters.maxHeight) return;

      // Size category filter
      if (filters.size && filters.size !== 'any') {
        const maxDim = Math.max(width, height);
        if (filters.size === 'large' && maxDim < 1920) return;
        if (filters.size === 'medium' && (maxDim < 800 || maxDim > 1920)) return;
        if (filters.size === 'small' && (maxDim < 300 || maxDim > 800)) return;
        if (filters.size === 'icon' && maxDim > 300) return;
      }

      // Aspect ratio filter
      if (filters.aspect && filters.aspect !== 'any') {
        const ratio = width / height;
        if (filters.aspect === 'tall' && ratio > 0.9) return;
        if (filters.aspect === 'wide' && ratio < 1.1) return;
        if (filters.aspect === 'square' && (ratio < 0.8 || ratio > 1.2)) return;
        if (filters.aspect === 'panoramic' && ratio < 2.0) return;
      }
    }

    // Build URLs
    const imageId = image.id || galleryItem.id;
    const galleryId = galleryItem.id;

    // Direct image link
    let imageUrl = image.link || `https://i.imgur.com/${imageId}.jpg`;
    // For animated content, prefer mp4 or gifv
    if (image.animated && image.mp4) {
      // Use the static thumbnail instead of video
      imageUrl = `https://i.imgur.com/${imageId}h.jpg`;
    }

    // Thumbnail URL (medium size)
    const thumbnailUrl = `https://i.imgur.com/${imageId}m.jpg`;

    // Gallery page URL
    const pageUrl = galleryItem.in_gallery
      ? `https://imgur.com/gallery/${galleryId}`
      : `https://imgur.com/${imageId}`;

    results.results.push({
      url: pageUrl,
      title: galleryItem.title || image.title || 'Imgur Image',
      content: galleryItem.description || image.description || '',
      engine: this.name,
      score: this.weight,
      category: 'images',
      template: 'images',
      imageUrl,
      thumbnailUrl,
      resolution: width && height ? `${width}x${height}` : '',
      source: galleryItem.account_url || 'Imgur',
      metadata: {
        views: galleryItem.views || image.views,
        points: galleryItem.points,
        animated: image.animated,
      },
    });
  }

  private parseHtmlResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    // Try to find embedded JSON data in HTML
    const jsonMatch = body.match(/window\.postDataJSON\s*=\s*"([^"]+)"/);
    if (jsonMatch) {
      try {
        const unescaped = jsonMatch[1]
          .replace(/\\"/g, '"')
          .replace(/\\\\/g, '\\');
        const data = JSON.parse(unescaped);

        if (Array.isArray(data)) {
          for (const item of data) {
            if (item.id && (item.link || item.hash)) {
              this.processImage(item, item, results, filters);
            }
          }
          if (results.results.length > 0) return results;
        }
      } catch {
        // Continue to HTML parsing
      }
    }

    // Fallback: parse HTML directly
    this.extractFromHtml(body, results);

    return results;
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Look for image posts in HTML
    const postRegex = /data-post-id="([^"]+)"[^>]*data-post-title="([^"]*)"/g;
    let match: RegExpExecArray | null;
    const seen = new Set<string>();

    while ((match = postRegex.exec(body)) !== null) {
      const postId = match[1];
      if (seen.has(postId)) continue;
      seen.add(postId);

      const title = decodeHtmlEntities(match[2]) || 'Imgur Image';

      results.results.push({
        url: `https://imgur.com/gallery/${postId}`,
        title,
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl: `https://i.imgur.com/${postId}.jpg`,
        thumbnailUrl: `https://i.imgur.com/${postId}m.jpg`,
        source: 'Imgur',
      });
    }

    // Also look for direct image links
    const imgRegex = /href="(https?:\/\/(?:i\.)?imgur\.com\/(\w+)(?:\.\w+)?)"[^>]*>/g;
    while ((match = imgRegex.exec(body)) !== null) {
      const imageId = match[2];
      if (seen.has(imageId)) continue;
      if (imageId.length < 5) continue; // Skip short IDs that are likely not images
      seen.add(imageId);

      results.results.push({
        url: `https://imgur.com/${imageId}`,
        title: 'Imgur Image',
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl: `https://i.imgur.com/${imageId}.jpg`,
        thumbnailUrl: `https://i.imgur.com/${imageId}m.jpg`,
        source: 'Imgur',
      });
    }
  }
}
