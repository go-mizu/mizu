/**
 * Flickr Image Search Engine adapter.
 *
 * Uses Flickr's public API endpoints for searching photos.
 * No API key required for basic public photo searches.
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Flickr API Types ==========

interface FlickrPhoto {
  id: string;
  owner: string;
  secret: string;
  server: string;
  farm: number;
  title: string;
  ispublic: number;
  isfriend: number;
  isfamily: number;
  ownername?: string;
  description?: { _content: string };
  o_width?: string;
  o_height?: string;
  url_o?: string;
  url_l?: string;
  url_m?: string;
  url_s?: string;
  url_t?: string;
  url_sq?: string;
  height_o?: string;
  width_o?: string;
  height_l?: string;
  width_l?: string;
  height_m?: string;
  width_m?: string;
}

interface FlickrSearchResponse {
  photos?: {
    page: number;
    pages: number;
    perpage: number;
    total: number;
    photo: FlickrPhoto[];
  };
  stat?: string;
  message?: string;
}

// ========== Size Filter Mapping ==========

const flickrSizeMap: Record<string, string> = {
  large: 'l', // 1024 on longest side
  medium: 'm', // 500 on longest side
  small: 's', // 240 on longest side
  icon: 't', // 100 on longest side
};

// ========== Aspect Filter Mapping ==========

const flickrAspectMap: Record<string, string> = {
  square: 'square',
  tall: 'portrait',
  wide: 'landscape',
  panoramic: 'panorama',
};

// ========== Color Filter Mapping ==========

const flickrColorMap: Record<string, string> = {
  red: '0',
  orange: '1',
  yellow: '2',
  green: '3',
  teal: '4',
  blue: '5',
  purple: '6',
  pink: '7',
  white: '8',
  gray: '9',
  black: 'a',
  brown: 'b',
};

export class FlickrEngine implements OnlineEngine {
  name = 'flickr';
  shortcut = 'fl';
  categories: Category[] = ['images'];
  supportsPaging = true;
  maxPage = 100;
  timeout = 8000;
  weight = 0.85;
  disabled = false;

  private readonly perPage = 20;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();

    // Use Flickr's public JSON feed API
    searchParams.set('method', 'flickr.photos.search');
    searchParams.set('api_key', ''); // Empty key works for public search
    searchParams.set('format', 'json');
    searchParams.set('nojsoncallback', '1');
    searchParams.set('text', query);
    searchParams.set('per_page', this.perPage.toString());
    searchParams.set('page', params.page.toString());
    searchParams.set('extras', 'url_o,url_l,url_m,url_s,url_t,description,owner_name,o_dims');

    // Safe search level
    // Flickr: 1 = safe, 2 = moderate, 3 = restricted
    const safeLevel = params.safeSearch === 0 ? 3 : params.safeSearch === 1 ? 2 : 1;
    searchParams.set('safe_search', safeLevel.toString());

    // Apply image filters
    const filters = params.imageFilters;
    if (filters) {
      // Color code filter
      if (filters.color && filters.color !== 'any' && flickrColorMap[filters.color]) {
        searchParams.set('color_codes', flickrColorMap[filters.color]);
      }

      // Content type (photos only, exclude screenshots/screencasts)
      if (filters.type === 'photo') {
        searchParams.set('content_type', '1');
      }

      // Aspect ratio (orientation)
      if (filters.aspect && filters.aspect !== 'any' && flickrAspectMap[filters.aspect]) {
        searchParams.set('orientation', flickrAspectMap[filters.aspect]);
      }

      // License filter for Creative Commons
      if (filters.rights === 'creative_commons') {
        // 1-8 are various CC licenses
        searchParams.set('license', '1,2,3,4,5,6,7,8');
      } else if (filters.rights === 'commercial') {
        // 4,5,6,7 allow commercial use
        searchParams.set('license', '4,5,6,7');
      }

      // Minimum dimensions
      if (filters.minWidth) {
        searchParams.set('min_upload_date', ''); // Required param for dimension filtering
      }
    }

    // Time range filter
    if (params.timeRange) {
      const now = Math.floor(Date.now() / 1000);
      let minDate = 0;
      switch (params.timeRange) {
        case 'day':
          minDate = now - 86400;
          break;
        case 'week':
          minDate = now - 604800;
          break;
        case 'month':
          minDate = now - 2592000;
          break;
        case 'year':
          minDate = now - 31536000;
          break;
      }
      if (minDate > 0) {
        searchParams.set('min_upload_date', minDate.toString());
      }
    }

    // Sort by interestingness (relevance) or date
    searchParams.set('sort', 'relevance');

    return {
      url: `https://www.flickr.com/services/rest/?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, params: EngineParams): EngineResults {
    const results = newEngineResults();
    const filters = params.imageFilters;

    try {
      const data: FlickrSearchResponse = JSON.parse(body);

      if (data.stat !== 'ok' || !data.photos?.photo) {
        return results;
      }

      for (const photo of data.photos.photo) {
        // Build image URLs
        // Flickr URL format: https://live.staticflickr.com/{server-id}/{id}_{secret}_{size}.jpg
        const baseUrl = `https://live.staticflickr.com/${photo.server}/${photo.id}_${photo.secret}`;

        // Get the best available image URL
        let imageUrl = photo.url_o || photo.url_l || photo.url_m || `${baseUrl}_b.jpg`;
        let thumbnailUrl = photo.url_s || photo.url_t || `${baseUrl}_m.jpg`;

        // Determine resolution
        let width = parseInt(photo.o_width || photo.width_o || photo.width_l || '0', 10);
        let height = parseInt(photo.o_height || photo.height_o || photo.height_l || '0', 10);

        // Client-side size filtering
        if (filters) {
          if (filters.minWidth && width && width < filters.minWidth) continue;
          if (filters.minHeight && height && height < filters.minHeight) continue;
          if (filters.maxWidth && width && width > filters.maxWidth) continue;
          if (filters.maxHeight && height && height > filters.maxHeight) continue;

          // Size category filter (approximate)
          if (filters.size && filters.size !== 'any') {
            const minDim = Math.max(width, height);
            if (filters.size === 'large' && minDim < 1024) continue;
            if (filters.size === 'medium' && (minDim < 500 || minDim > 1024)) continue;
            if (filters.size === 'small' && (minDim < 200 || minDim > 500)) continue;
            if (filters.size === 'icon' && minDim > 200) continue;
          }
        }

        const resolution = width && height ? `${width}x${height}` : '';
        const pageUrl = `https://www.flickr.com/photos/${photo.owner}/${photo.id}`;

        results.results.push({
          url: pageUrl,
          title: photo.title || 'Untitled',
          content: photo.description?._content || '',
          engine: this.name,
          score: this.weight,
          category: 'images',
          template: 'images',
          imageUrl,
          thumbnailUrl,
          resolution,
          source: photo.ownername || photo.owner,
        });
      }
    } catch {
      // JSON parse failed, try alternate extraction
      this.extractFromHtml(body, results);
    }

    return results;
  }

  private extractFromHtml(body: string, results: EngineResults): void {
    // Fallback: try to extract from HTML if JSON fails
    const photoRegex = /data-photo-id="(\d+)"[^>]*data-owner-nsid="([^"]+)"[^>]*title="([^"]*)"/g;
    let match: RegExpExecArray | null;

    while ((match = photoRegex.exec(body)) !== null) {
      const photoId = match[1];
      const ownerId = match[2];
      const title = match[3];

      results.results.push({
        url: `https://www.flickr.com/photos/${ownerId}/${photoId}`,
        title: title || 'Untitled',
        content: '',
        engine: this.name,
        score: this.weight,
        category: 'images',
        template: 'images',
        imageUrl: `https://live.staticflickr.com/${photoId}_b.jpg`,
        thumbnailUrl: `https://live.staticflickr.com/${photoId}_m.jpg`,
        source: 'Flickr',
      });
    }
  }
}
