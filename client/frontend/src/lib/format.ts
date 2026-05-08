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
// onto the auth nickname using '.' as the separator (the deployed
// signaling server's nickname validator rejects '@', whitespace,
// Unicode, etc — '.' is the most readable of the three characters
// the validator does allow: `_`, `-`, `.`).
//
// We split on the LAST '.' so a username that itself contains a
// dot (e.g. 'john.doe') still works — only the trailing '.<device>'
// is treated as the device tag. When the input has no dot we just
// return the whole string as the nickname; older clients without
// device-name support land here naturally.
export function splitNicknameAndDevice(combined: string): {
  nickname: string;
  device: string;
} {
  if (!combined) return { nickname: "", device: "" };
  const i = combined.lastIndexOf(".");
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
