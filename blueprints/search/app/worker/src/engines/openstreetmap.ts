/**
 * OpenStreetMap/Nominatim Search Engine adapter.
 *
 * Uses Nominatim for geocoding and place search.
 * Returns location results with coordinates and map data.
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

// ========== Nominatim API Types ==========

interface NominatimPlace {
  place_id: number;
  licence: string;
  osm_type: string;
  osm_id: number;
  lat: string;
  lon: string;
  class: string;
  type: string;
  place_rank: number;
  importance: number;
  addresstype: string;
  name: string;
  display_name: string;
  boundingbox: [string, string, string, string]; // [south, north, west, east]
  address?: {
    house_number?: string;
    road?: string;
    suburb?: string;
    city?: string;
    town?: string;
    village?: string;
    county?: string;
    state?: string;
    postcode?: string;
    country?: string;
    country_code?: string;
  };
  extratags?: {
    website?: string;
    phone?: string;
    opening_hours?: string;
    wikipedia?: string;
    wikidata?: string;
    population?: string;
    elevation?: string;
  };
  namedetails?: Record<string, string>;
  icon?: string;
}

const USER_AGENT = 'MizuSearch/1.0 (https://github.com/mizu)';

// Nominatim public server
const NOMINATIM_URL = 'https://nominatim.openstreetmap.org';

// Map class types to human-readable categories
const classLabels: Record<string, string> = {
  place: 'Place',
  boundary: 'Administrative Boundary',
  highway: 'Road/Highway',
  building: 'Building',
  amenity: 'Amenity',
  shop: 'Shop',
  tourism: 'Tourism',
  natural: 'Natural Feature',
  waterway: 'Waterway',
  landuse: 'Land Use',
  railway: 'Railway',
  aeroway: 'Airport/Aeroway',
  leisure: 'Leisure',
  historic: 'Historic Site',
  office: 'Office',
  craft: 'Craft',
  emergency: 'Emergency Service',
  healthcare: 'Healthcare',
  man_made: 'Man-made Structure',
  military: 'Military',
  power: 'Power Infrastructure',
};

// Map type to human-readable labels
const typeLabels: Record<string, string> = {
  city: 'City',
  town: 'Town',
  village: 'Village',
  suburb: 'Suburb',
  neighbourhood: 'Neighborhood',
  hamlet: 'Hamlet',
  isolated_dwelling: 'Isolated Dwelling',
  country: 'Country',
  state: 'State/Province',
  county: 'County',
  municipality: 'Municipality',
  administrative: 'Administrative Area',
  continent: 'Continent',
  island: 'Island',
  peak: 'Mountain Peak',
  volcano: 'Volcano',
  river: 'River',
  lake: 'Lake',
  ocean: 'Ocean',
  sea: 'Sea',
  bay: 'Bay',
  beach: 'Beach',
  forest: 'Forest',
  park: 'Park',
  nature_reserve: 'Nature Reserve',
  restaurant: 'Restaurant',
  cafe: 'Cafe',
  hotel: 'Hotel',
  hospital: 'Hospital',
  school: 'School',
  university: 'University',
  museum: 'Museum',
  library: 'Library',
  theatre: 'Theatre',
  cinema: 'Cinema',
  stadium: 'Stadium',
  airport: 'Airport',
  station: 'Train Station',
  bus_station: 'Bus Station',
};

export class OpenStreetMapEngine implements OnlineEngine {
  name = 'openstreetmap';
  shortcut = 'osm';
  categories: Category[] = ['general'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 10_000;
  weight = 0.9;
  disabled = false;

  private limit: number;

  constructor(options?: { limit?: number }) {
    this.limit = options?.limit ?? 20;
  }

  buildRequest(query: string, params: EngineParams): RequestConfig {
    const searchParams = new URLSearchParams();
    searchParams.set('q', query);
    searchParams.set('format', 'jsonv2');
    searchParams.set('limit', this.limit.toString());
    searchParams.set('addressdetails', '1');
    searchParams.set('extratags', '1');
    searchParams.set('namedetails', '1');
    searchParams.set('dedupe', '1');

    // Language preference
    const lang = params.locale.split('-')[0] || 'en';
    searchParams.set('accept-language', lang);

    return {
      url: `${NOMINATIM_URL}/search?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const places: NominatimPlace[] = JSON.parse(body);

      if (!Array.isArray(places)) {
        return results;
      }

      for (const place of places) {
        const result = this.parsePlace(place);
        if (result) {
          results.results.push(result);
        }
      }
    } catch {
      // JSON parse failed
    }

    return results;
  }

  private parsePlace(place: NominatimPlace): EngineResults['results'][0] | null {
    if (!place.display_name) return null;

    // Build title - use name if available, otherwise the display name
    const title = place.name || place.display_name.split(',')[0];

    // Build URL to OpenStreetMap
    const osmUrl = `https://www.openstreetmap.org/${place.osm_type}/${place.osm_id}`;

    // Get coordinates
    const lat = parseFloat(place.lat);
    const lon = parseFloat(place.lon);

    // Determine place type label
    const classLabel = classLabels[place.class] || place.class;
    const typeLabel = typeLabels[place.type] || place.type.replace(/_/g, ' ');

    // Build content with location details
    const contentParts: string[] = [];

    // Add type info
    contentParts.push(`${typeLabel} (${classLabel})`);

    // Add coordinates
    contentParts.push(`${lat.toFixed(5)}, ${lon.toFixed(5)}`);

    // Add address parts
    const addr = place.address;
    if (addr) {
      const addressParts: string[] = [];
      if (addr.city || addr.town || addr.village) {
        addressParts.push(addr.city || addr.town || addr.village || '');
      }
      if (addr.state) {
        addressParts.push(addr.state);
      }
      if (addr.country) {
        addressParts.push(addr.country);
      }
      if (addressParts.length > 0) {
        contentParts.push(addressParts.join(', '));
      }
    }

    // Add population if available
    if (place.extratags?.population) {
      contentParts.push(`Pop: ${this.formatNumber(parseInt(place.extratags.population, 10))}`);
    }

    const content = contentParts.join(' | ');

    // Build thumbnail URL (static map image)
    const thumbnailUrl = this.buildMapThumbnail(lat, lon);

    return {
      url: osmUrl,
      title: decodeHtmlEntities(title),
      content,
      engine: this.name,
      score: this.weight * place.importance,
      category: 'general',
      template: 'images',
      thumbnailUrl,
      source: 'OpenStreetMap',
      metadata: {
        placeId: place.place_id,
        osmType: place.osm_type,
        osmId: place.osm_id,
        latitude: lat,
        longitude: lon,
        boundingBox: place.boundingbox,
        class: place.class,
        type: place.type,
        placeRank: place.place_rank,
        importance: place.importance,
        addressType: place.addresstype,
        address: place.address,
        website: place.extratags?.website,
        phone: place.extratags?.phone,
        openingHours: place.extratags?.opening_hours,
        wikipedia: place.extratags?.wikipedia,
        wikidata: place.extratags?.wikidata,
        population: place.extratags?.population,
        elevation: place.extratags?.elevation,
      },
    };
  }

  private buildMapThumbnail(lat: number, lon: number): string {
    // Use OpenStreetMap static map tile
    // Calculate tile coordinates for zoom level 14
    const zoom = 14;
    const n = Math.pow(2, zoom);
    const x = Math.floor(((lon + 180) / 360) * n);
    const latRad = (lat * Math.PI) / 180;
    const y = Math.floor(((1 - Math.log(Math.tan(latRad) + 1 / Math.cos(latRad)) / Math.PI) / 2) * n);

    return `https://tile.openstreetmap.org/${zoom}/${x}/${y}.png`;
  }

  private formatNumber(num: number): string {
    if (num >= 1_000_000) {
      return `${(num / 1_000_000).toFixed(1)}M`;
    }
    if (num >= 1_000) {
      return `${(num / 1_000).toFixed(1)}K`;
    }
    return num.toString();
  }
}

/**
 * OpenStreetMap Reverse Geocoding Engine.
 * Converts coordinates to place names.
 */
export class OpenStreetMapReverseEngine implements OnlineEngine {
  name = 'openstreetmap reverse';
  shortcut = 'osmr';
  categories: Category[] = ['general'];
  supportsPaging = false;
  maxPage = 1;
  timeout = 10_000;
  weight = 0.85;
  disabled = false;

  buildRequest(query: string, params: EngineParams): RequestConfig {
    // Parse coordinates from query (format: "lat,lon" or "lat lon")
    const coords = query.replace(',', ' ').trim().split(/\s+/);

    if (coords.length !== 2) {
      // Return empty request if coordinates are invalid
      return {
        url: '',
        method: 'GET',
        headers: {},
        cookies: [],
      };
    }

    const lat = coords[0];
    const lon = coords[1];

    const searchParams = new URLSearchParams();
    searchParams.set('lat', lat);
    searchParams.set('lon', lon);
    searchParams.set('format', 'jsonv2');
    searchParams.set('addressdetails', '1');
    searchParams.set('extratags', '1');
    searchParams.set('zoom', '18');

    const lang = params.locale.split('-')[0] || 'en';
    searchParams.set('accept-language', lang);

    return {
      url: `${NOMINATIM_URL}/reverse?${searchParams.toString()}`,
      method: 'GET',
      headers: {
        Accept: 'application/json',
        'User-Agent': USER_AGENT,
      },
      cookies: [],
    };
  }

  parseResponse(body: string, _params: EngineParams): EngineResults {
    const results = newEngineResults();

    try {
      const place: NominatimPlace = JSON.parse(body);

      if (place.display_name) {
        const osmUrl = `https://www.openstreetmap.org/${place.osm_type}/${place.osm_id}`;
        const lat = parseFloat(place.lat);
        const lon = parseFloat(place.lon);

        results.results.push({
          url: osmUrl,
          title: place.name || place.display_name.split(',')[0],
          content: place.display_name,
          engine: this.name,
          score: this.weight,
          category: 'general',
          source: 'OpenStreetMap',
          metadata: {
            placeId: place.place_id,
            osmType: place.osm_type,
            osmId: place.osm_id,
            latitude: lat,
            longitude: lon,
            address: place.address,
          },
        });
      }
    } catch {
      // Parse error
    }

    return results;
  }
}
