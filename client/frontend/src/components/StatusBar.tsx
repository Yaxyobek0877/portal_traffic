import React from "react";
import { Activity, Users, Wifi } from "lucide-react";
import type { PeerView } from "../types";
import { rttLabel } from "../lib/format";

type Props = {
  ownVip: string;
  peers: PeerView[];
  signalingUrl: string;
};

export function StatusBar({ ownVip, peers, signalingUrl }: Props) {
  const connected = peers.filter((p) => p.state === "connected").length;
  const avgRtt = (() => {
    const v = peers.filter((p) => p.rttMs > 0).map((p) => p.rttMs);
    if (v.length === 0) return 0;
    return v.reduce((a, b) => a + b, 0) / v.length;
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
      </div>
      <div className="flex items-center gap-1.5 truncate max-w-[50%]">
        <Wifi className="w-3 h-3 text-emerald-400" />
        <span className="truncate text-zinc-500">{signalingUrl}</span>
      </div>
    </div>
  );
}
