import React from "react";
import { motion } from "framer-motion";
import { Crown, Wifi, WifiOff } from "lucide-react";
import type { PeerView } from "../types";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { rttLabel, shortId } from "../lib/format";

type Props = {
  peer: PeerView;
  highlighted?: boolean;
  onClick?: () => void;
};

export function PeerCard({ peer, highlighted, onClick }: Props) {
  const color = avatarColor(peer.nickname || peer.peerId);
  const initial = avatarInitial(peer.nickname || peer.peerId);
  const dot =
    peer.state === "connected"
      ? "bg-emerald-400 shadow-[0_0_10px_#34d399]"
      : peer.state === "failed"
      ? "bg-rose-400 shadow-[0_0_10px_#fb7185]"
      : peer.state === "closed"
      ? "bg-zinc-500"
      : "bg-amber-400 shadow-[0_0_10px_#fbbf24]";

  return (
    <motion.button
      type="button"
      onClick={onClick}
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.95 }}
      whileHover={{ scale: 1.01 }}
      transition={{ type: "spring", stiffness: 300, damping: 30 }}
      className={`panel rounded-card w-full text-left p-3 flex items-center gap-3 transition-colors ${
        highlighted ? "border-accent2/50 bg-white/[0.06]" : ""
      }`}
    >
      <div
        className="w-9 h-9 rounded-full flex items-center justify-center font-semibold text-sm text-white shrink-0"
        style={{ background: `linear-gradient(135deg, ${color} 0%, #6366f1 100%)` }}
      >
        {initial}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <div className="font-medium truncate text-sm">{peer.nickname || shortId(peer.peerId)}</div>
          {peer.isOwner && <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />}
        </div>
        <div className="text-xs text-zinc-400 font-mono mt-0.5">{peer.virtualIp}</div>
      </div>

      <div className="flex flex-col items-end gap-1 shrink-0">
        <div className="flex items-center gap-1.5">
          <span className={`w-2 h-2 rounded-full ${dot}`} />
          {peer.state === "connected" ? (
            <Wifi className="w-3.5 h-3.5 text-emerald-400/70" strokeWidth={2} />
          ) : (
            <WifiOff className="w-3.5 h-3.5 text-amber-400/70" strokeWidth={2} />
          )}
        </div>
        <div className="text-[11px] text-zinc-500 font-mono">{rttLabel(peer.rttMs)}</div>
      </div>
    </motion.button>
  );
}
