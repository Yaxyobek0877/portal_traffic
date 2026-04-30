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

export function timeOfDay(iso: string): string {
  try {
    const d = new Date(iso);
    return d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  } catch {
    return "";
  }
}
