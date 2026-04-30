import React from "react";
import { Activity, ArrowDown, ArrowUp, Globe, Users, Wifi } from "lucide-react";
import type { NATResult, PeerView } from "../types";
import { rttLabel } from "../lib/format";

function bytesShort(n: number): string {
  if (n < 1024) return `${n}B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(0)}K`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)}M`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)}G`;
}

type Props = {
  ownVip: string;
  peers: PeerView[];
  signalingUrl: string;
  nat: NATResult | null;
};

export function StatusBar({ ownVip, peers, signalingUrl, nat }: Props) {
  const connected = peers.filter((p) => p.state === "connected").length;
  const avgRtt = (() => {
    const v = peers.filter((p) => p.rttMs > 0).map((p) => p.rttMs);
    if (v.length === 0) return 0;
    return v.reduce((a, b) => a + b, 0) / v.length;
  })();
  const totalSent = peers.reduce((acc, p) => acc + (p.bytesSent || 0), 0);
  const totalRecv = peers.reduce((acc, p) => acc + (p.bytesRecv || 0), 0);

  const natBadge = (() => {
    if (!nat) return null;
    const color =
      nat.type === 1
        ? "text-emerald-400"
        : nat.type === 2
        ? "text-amber-400"
        : nat.type === 3
        ? "text-rose-400"
        : "text-zinc-500";
    return (
      <span title={`Aniqlandi: ${nat.detectedAt}`} className={`flex items-center gap-1.5 ${color}`}>
        <Globe className="w-3 h-3" /> {nat.label}
      </span>
    );
  })();

  return (
    <div className="panel border-t border-white/5 px-4 py-2 flex items-center justify-between text-xs text-zinc-400 font-mono">
      <div className="flex items-center gap-4">
        <span>
          <span className="text-zinc-600">vip </span>
          <span className="text-zinc-200">{ownVip || "—"}</span>
        </span>
        <span className="flex items-center gap-1.5">
          <Users className="w-3 h-3" />
          {connected}/{peers.length}
        </span>
        <span className="flex items-center gap-1.5">
          <Activity className="w-3 h-3" />
          {rttLabel(avgRtt)}
        </span>
        <span className="flex items-center gap-1" title="Mesh ustida yuborilgan / qabul qilingan baytlar">
          <ArrowUp className="w-3 h-3 text-violet-300" /> {bytesShort(totalSent)}
          <ArrowDown className="w-3 h-3 text-cyan-300 ml-1" /> {bytesShort(totalRecv)}
        </span>
        {natBadge}
      </div>
      <div className="flex items-center gap-1.5 truncate max-w-[50%]">
        <Wifi className="w-3 h-3 text-emerald-400" />
        <span className="truncate text-zinc-500">{signalingUrl}</span>
      </div>
    </div>
  );
}
