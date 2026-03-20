/**
 * TUS protocol helpers — metadata parsing and ID generation.
 */

const TUS_VERSION = "1.0.0";
const TUS_EXTENSIONS = "creation,creation-with-upload,termination,expiration";
const TUS_EXPIRY_MS = 24 * 60 * 60 * 1000; // 24 hours

export { TUS_VERSION, TUS_EXTENSIONS, TUS_EXPIRY_MS };

/** Generate a TUS upload ID. */
export function tusUploadId(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return `tu_${Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("")}`;
}

/**
 * Parse the Upload-Metadata header.
 * Format: "key base64val,key2 base64val2"
 * Keys without values are allowed (value will be empty string).
 */
export function parseUploadMetadata(header: string): Record<string, string> {
  const result: Record<string, string> = {};
  if (!header) return result;

  for (const pair of header.split(",")) {
    const trimmed = pair.trim();
    if (!trimmed) continue;
    const spaceIdx = trimmed.indexOf(" ");
    if (spaceIdx === -1) {
      result[trimmed] = "";
    } else {
      const key = trimmed.slice(0, spaceIdx);
      const b64 = trimmed.slice(spaceIdx + 1).trim();
      try {
        result[key] = atob(b64);
      } catch {
        result[key] = "";
      }
    }
  }

  return result;
}

/** Format an RFC 7231 date for Upload-Expires. */
export function formatExpires(ts: number): string {
  return new Date(ts).toUTCString();
}
