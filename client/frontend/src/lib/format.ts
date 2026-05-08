export function rttLabel(ms: number): string {
  if (ms <= 0) return "—";
  if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`;
  if (ms < 100) return `${ms.toFixed(1)}ms`;
  return `${Math.round(ms)}ms`;
}

export function shortId(id: string): string {
  if (!id) return "";
  return id.length <= 8 ? id : id.slice(0, 8);
}

// splitNicknameAndDevice un-flattens what the Go side packs into the
// mesh nickname. createPortal / joinPortal stamp the device label
// onto the auth nickname using '@' as the separator (server-side
// regex rejects whitespace and Unicode glyphs like '·'); the
// frontend cards prefer the prettier 'username · device' form, so
// here we split on the LAST '@' so the username can itself contain
// an '@' (extremely rare but cheap to handle). When the input
// doesn't carry a separator we just return the whole string as
// the nickname and an empty device tag — older clients without
// device-name support land here naturally.
export function splitNicknameAndDevice(combined: string): {
  nickname: string;
  device: string;
} {
  if (!combined) return { nickname: "", device: "" };
  const i = combined.lastIndexOf("@");
  if (i <= 0) return { nickname: combined, device: "" };
  return {
    nickname: combined.slice(0, i),
    device: combined.slice(i + 1),
  };
}

export function timeOfDay(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } catch {
    return "";
  }
}
