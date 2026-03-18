const TYPES: Record<string, string> = {
  html: "text/html",
  htm: "text/html",
  css: "text/css",
  js: "application/javascript",
  mjs: "application/javascript",
  json: "application/json",
  xml: "application/xml",
  txt: "text/plain",
  md: "text/markdown",
  csv: "text/csv",
  svg: "image/svg+xml",
  png: "image/png",
  jpg: "image/jpeg",
  jpeg: "image/jpeg",
  gif: "image/gif",
  webp: "image/webp",
  ico: "image/x-icon",
  pdf: "application/pdf",
  zip: "application/zip",
  gz: "application/gzip",
  tar: "application/x-tar",
  mp3: "audio/mpeg",
  mp4: "video/mp4",
  webm: "video/webm",
  wasm: "application/wasm",
  woff: "font/woff",
  woff2: "font/woff2",
  ttf: "font/ttf",
  otf: "font/otf",
  yaml: "text/yaml",
  yml: "text/yaml",
  toml: "text/plain",
  rs: "text/plain",
  go: "text/plain",
  py: "text/plain",
  ts: "text/plain",
  tsx: "text/plain",
  jsx: "text/plain",
  sh: "text/plain",
  sql: "text/plain",
  log: "text/plain",
  env: "text/plain",
  dockerfile: "text/plain",
};

export function mimeFromName(name: string): string {
  const ext = name.split(".").pop()?.toLowerCase() || "";
  return TYPES[ext] || "application/octet-stream";
}

const INLINE_PREFIXES = ["text/", "image/", "video/", "audio/", "application/pdf", "application/json"];

export function isInlineType(ct: string): boolean {
  return INLINE_PREFIXES.some((p) => ct.startsWith(p));
}
