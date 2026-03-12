/**
 * Convert a glob pattern to a RegExp.
 * Supports: `*` (any char except `/`) and `**` (any char including `/`).
 */
export function globToRegex(pattern: string): RegExp {
  let regex = "";
  let i = 0;
  while (i < pattern.length) {
    const ch = pattern[i];
    if (ch === "*") {
      if (pattern[i + 1] === "*") {
        regex += ".*";
        i += 2;
      } else {
        regex += "[^/]*";
        i++;
      }
    } else if (".+^${}()|[]\\".includes(ch)) {
      regex += "\\" + ch;
      i++;
    } else {
      regex += ch;
      i++;
    }
  }
  return new RegExp("^" + regex + "$");
}

export function matchesAnyPattern(url: string, patterns: string[]): boolean {
  return patterns.some((p) => globToRegex(p).test(url));
}
