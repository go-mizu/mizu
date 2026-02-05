/**
 * Odysee/LBRY Search Engine adapter.
 *
 * Uses the Odysee Lighthouse API for video search.
 * Lighthouse is LBRY's search API.
 * No API key required.
 *
 * API Documentation: https://lbry.tech/api/lighthouse
 */

import type {
  OnlineEngine,
  EngineParams,
  RequestConfig,
  EngineResults,
  Category,
} from './engine';
import { newEngineResults } from './engine';

// ========== Odysee/LBRY API Types ==========

interface OdyseeClaimValue {
  title?: string;
  description?: string;
  thumbnail?: {
    url?: string;
  };
  video?: {
    duration?: number;
    width?: number;
    height?: number;
  };
  release_time?: number;
}

interface OdyseeChannel {
  name?: string;
  claim_id?: string;
}

interface OdyseeClaim {
  claimId?: string;
  claim_id?: string;
  name?: string;
  title?: string;
  description?: string;
  thumbnail_url?: string;
  channel?: string;
  channel_claim_id?: string;
  duration?: number;
  fee?: number;
  release_time?: number;
  effective_amount?: number;
  // Nested value structure from API
  value?: OdyseeClaimValue;
  signing_channel?: OdyseeChannel;
}

// Reserved for typed API response handling
// interface OdyseeSearchResponse {
//   // Lighthouse API returns array directly
//   [index: number]: OdyseeClaim;
//   length?: number;
// }

// Time range calculation (Unix timestamps)
function calculateReleaseTime(timeRange: string): number | null {
  const now = Math.floor(Date.now() / 1000);
  let offset: number;

  switch (timeRange) {
    case 'day':
      offset = 24 * 60 * 60;
      break;
    case 'week':
      offset = 7 * 24 * 60 * 60;
      break;
    case 'month':
      offset = 30 * 24 * 60 * 60;
      break;
    case 'year':
      offset = 365 * 24 * 60 * 60;
      break;
    default:
      return null;
  }

  return now - offset;
}

export class OdyseeEngine implements OnlineEngine {
  name = 'odysee';
  shortcut = 'od';
  categories: Category[] = ['videos'];
  supportsPaging = true;
  maxPage = 10;
  timeout = 8000;
  weight = 0.75;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('s', query);
    searchParams.set('size', '20');
    searchParams.set('from', ((params.page - 1) * 20).toString());
    searchParams.set('mediaType', 'video');
    searchParams.set('free_only', 'true');

    // NSFW filter based on safe search
    if (params.safeSearch >= 1) {
      searchParams.set('nsfw', 'false');
    }

    // Time range filter
    if (params.timeRange) {
      const releaseTime = calculateReleaseTime(params.timeRange);
      if (releaseTime) {
        searchParams.set('release_time', `>${releaseTime}`);
      }
    }

    return {
      url: `https://lighthouse.odysee.tv/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    let data: OdyseeClaim[];
    try {
      data = JSON.parse(body);
    } catch {
      return results;
    }

    if (!Array.isArray(data)) {
      return results;
    }

    for (const claim of data) {
      const claimId = claim.claimId || claim.claim_id;
      const name = claim.name;

      if (!claimId || !name) continue;

      // Get title from claim or nested value
      const title =
        claim.title ||
        claim.value?.title ||
        name.replace(/-/g, ' ');

      // Get description
      const description =
        claim.description ||
        claim.value?.description ||
        '';

      // Get thumbnail
      const thumbnailUrl =
        claim.thumbnail_url ||
        claim.value?.thumbnail?.url ||
        '';

      // Get duration (in seconds)
      const durationSeconds =
        claim.duration ||
        claim.value?.video?.duration ||
        0;
      const duration = durationSeconds ? this.formatSeconds(durationSeconds) : '';

      // Get channel name
      const channel =
        claim.channel ||
        claim.signing_channel?.name?.replace('@', '') ||
        '';

      // Build Odysee URL
      const channelPart = claim.channel || claim.signing_channel?.name || '';
      let videoUrl: string;
      if (channelPart) {
        videoUrl = `https://odysee.com/${channelPart}/${name}:${claimId.slice(0, 1)}`;
      } else {
        videoUrl = `https://odysee.com/${name}:${claimId}`;
      }

      // Build embed URL
      const embedUrl = `https://odysee.com/$/embed/${name}/${claimId}`;

      // Get release time for published date
      const releaseTime = claim.release_time || claim.value?.release_time;
      const publishedAt = releaseTime
        ? new Date(releaseTime * 1000).toISOString()
        : '';

      // Effective amount as a proxy for views (not exact but indicative of popularity)
      const views = claim.effective_amount
        ? Math.round(claim.effective_amount)
        : 0;

      results.results.push({
        url: videoUrl,
        title,
        content: description,
        engine: this.name,
        score: this.weight,
        category: 'videos',
        template: 'videos',
        duration,
        embedUrl,
        thumbnailUrl,
        channel,
        views,
        publishedAt,
      });
    }

    return results;
  }

  private formatSeconds(totalSeconds: number): string {
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const secs = Math.floor(totalSeconds % 60);

    if (hours > 0) {
      return `${hours}:${minutes.toString().padStart(2, '0')}:${secs
        .toString()
        .padStart(2, '0')}`;
    }
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
  }
}
