// Per-peer announced-services list with dial buttons.
//
// Lives under each PeerCard in the members column. Replaces the
// "Boshqa peerlardagi servislar" block that used to be at the bottom
// of the right tab — putting each service next to the peer that
// announced it shortens the "I want to connect to alice's server"
// path from "scroll to find services tab → find alice → find row"
// to "find alice → click Ulash". One section instead of two.
//
// On a successful Ulash the row swaps the button for a copy-able
// pill carrying the local alias (e.g. 127.0.0.1:27015). The pill
// is amber-tinted when the local port had to differ from the remote
// port (something local was already on the same number), green when
// it matches — the proxy.DialPreferringPort logic prefers a mirror
// alias and falls back to OS-pick.

import React, { useState } from "react";
import { Globe, Copy, Check } from "lucide-react";
import type { PeerView, ServiceView } from "../types";
import { app } from "../lib/wails";

export function PeerServicesList({ peer }: { peer: PeerView }) {
  const [dialing, setDialing] = useState<string | null>(null);
  const [dialedAddrs, setDialedAddrs] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const dial = async (svc: ServiceView) => {
    const key = `${svc.protocol}:${svc.port}`;
    setDialing(key);
    setError(null);
    try {
      const addr = await app.DialService(
        peer.peerId,
        svc.protocol as "tcp" | "udp",
        svc.port,
        0,
      );
      setDialedAddrs((s) => ({ ...s, [key]: addr }));
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setDialing(null);
    }
  };

  return (
    <div className="ml-3 pl-3 border-l border-white/[0.06] space-y-1">
      {peer.services.map((s) => {
        const key = `${s.protocol}:${s.port}`;
        const local = dialedAddrs[key];
        const isDialing = dialing === key;
        return (
          <div
            key={key}
            className="flex items-center gap-2 text-xs py-1"
          >
            <Globe
              className="w-3 h-3 text-cyan-400 shrink-0"
              strokeWidth={2}
            />
            <span className="truncate flex-1 min-w-0 text-zinc-300">
              {s.name}
            </span>
            <span
              className={`text-[9px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
                s.protocol === "udp"
                  ? "bg-cyan-400/10 text-cyan-300"
                  : "bg-violet-400/10 text-violet-300"
              }`}
            >
              {s.protocol.toUpperCase()}
            </span>
            <span className="font-mono text-[10px] text-zinc-500 shrink-0">
              :{s.port}
            </span>
            {local ? (
              <DialedPill addr={local} expectedPort={s.port} />
            ) : (
              <button
                disabled={isDialing}
                onClick={() => dial(s)}
                className="px-2 py-0.5 rounded bg-violet-500/15 text-violet-300 hover:bg-violet-500/25 text-[10px] font-medium disabled:opacity-50 shrink-0"
              >
                {isDialing ? "…" : "Ulash"}
              </button>
            )}
          </div>
        );
      })}
      {error && (
        <div className="text-[10px] text-rose-400 truncate" title={error}>
          {error}
        </div>
      )}
    </div>
  );
}

// DialedPill — local alias copy-pill. Same shape as the one in
// ServicesPanel but inlined here so PeerServicesList can stand alone.
function DialedPill({
  addr,
  expectedPort,
}: {
  addr: string;
  expectedPort: number;
}) {
  const [copied, setCopied] = useState(false);
  const localPort = parseInt(addr.split(":").pop() || "0", 10);
  const matches = localPort === expectedPort;
  const tint = matches
    ? "bg-emerald-500/15 hover:bg-emerald-500/25 text-emerald-300"
    : "bg-amber-500/15 hover:bg-amber-500/25 text-amber-300";
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(addr);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {}
  };
  return (
    <button
      onClick={copy}
      className={`flex items-center gap-1 px-1.5 py-0.5 rounded shrink-0 ${tint}`}
      title={
        matches
          ? `Lokal alias: ${addr}. Mesh orqali peer servisiga yo'naltiriladi.`
          : `Lokal :${expectedPort} band edi — Portal :${localPort} ni tanladi.`
      }
    >
      <span className="font-mono text-[10px]">{addr}</span>
      {copied ? (
        <Check className="w-2.5 h-2.5" strokeWidth={2.5} />
      ) : (
        <Copy className="w-2.5 h-2.5" strokeWidth={2} />
      )}
    </button>
  );
}
