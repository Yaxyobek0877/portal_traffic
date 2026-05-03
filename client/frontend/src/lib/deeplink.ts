// Portal deeplink scheme: portal://
//
// Format examples:
//   portal://join/428591
//   portal://join/428591?code=739204
//   portal://join/428591?code=X3K9-A2F1   (PCP-1 alphanumeric codes)
//
// Future schemes can ride on the same prefix:
//   portal://verify/<fingerprint>         — verify another peer's identity
//   portal://invite/<token>               — magic-link invite (server-issued)
//
// We accept several legacy formats too:
//   portal://<id>:<code>                  — early v0.4 alpha
//   portal://<id>                         — id only
//
// All accepted forms produce a normalised { portalId, code? } pair.

export type ParsedInvite = {
  portalId: string;
  code?: string;
};

const ID_RE = /^[A-Za-z0-9_-]{1,32}$/;
// Code regex covers both legacy 6-digit codes and new PCP-1 XXXX-YYYY codes.
const CODE_RE = /^[A-Za-z0-9-]{4,16}$/;

export function parseInvite(input: string): ParsedInvite | null {
  if (!input) return null;
  const trimmed = input.trim();

  // Try as URL first.
  if (trimmed.toLowerCase().startsWith("portal://")) {
    const tail = trimmed.slice("portal://".length);
    return parseUrlTail(tail);
  }

  // Heuristic: bare ID + code separated by space, comma, slash, or
  // newline. Useful for "ID 428591 code 739204" pasted from a chat
  // app or a text-based invite.
  const compact = trimmed
    .replace(/\bID[:\s]*/gi, " ")
    .replace(/\bKOD?[:\s]*/gi, " ")
    .replace(/\bCODE[:\s]*/gi, " ");
  const tokens = compact.split(/[\s,/]+/).filter((s) => s);
  // Find first token that looks like an ID and the next that looks
  // like a code.
  for (let i = 0; i < tokens.length; i++) {
    if (ID_RE.test(tokens[i])) {
      const id = tokens[i];
      const next = tokens[i + 1];
      if (next && CODE_RE.test(next)) {
        return { portalId: id, code: next };
      }
      return { portalId: id };
    }
  }
  return null;
}

function parseUrlTail(tail: string): ParsedInvite | null {
  // Strip an optional trailing slash. Then look for the action prefix
  // ("join/...") if present.
  const cleaned = tail.replace(/\/+$/, "");

  // portal://join/<id>?code=<code>
  if (cleaned.toLowerCase().startsWith("join/")) {
    const rest = cleaned.slice(5);
    const [pathPart, queryPart] = splitOnce(rest, "?");
    if (!ID_RE.test(pathPart)) return null;
    const code = queryFor(queryPart, "code");
    if (code && !CODE_RE.test(code)) return null;
    return code ? { portalId: pathPart, code } : { portalId: pathPart };
  }

  // Legacy: portal://<id>:<code> or portal://<id>
  const [head, query] = splitOnce(cleaned, "?");
  const [id, codeRaw] = splitOnce(head, ":");
  if (!ID_RE.test(id)) return null;
  const code = codeRaw || queryFor(query, "code");
  if (code && !CODE_RE.test(code)) return null;
  return code ? { portalId: id, code } : { portalId: id };
}

function splitOnce(s: string, sep: string): [string, string] {
  const i = s.indexOf(sep);
  if (i < 0) return [s, ""];
  return [s.slice(0, i), s.slice(i + 1)];
}

function queryFor(query: string, key: string): string {
  if (!query) return "";
  for (const part of query.split("&")) {
    const [k, v = ""] = splitOnce(part, "=");
    if (k.toLowerCase() === key.toLowerCase()) {
      try {
        return decodeURIComponent(v);
      } catch {
        return v;
      }
    }
  }
  return "";
}
