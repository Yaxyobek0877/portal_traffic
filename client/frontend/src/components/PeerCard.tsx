import React from "react";
import { motion } from "framer-motion";
import { Crown, Wifi, WifiOff, Zap, Cloud, X } from "lucide-react";
import type { PeerView } from "../types";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { rttLabel, shortId } from "../lib/format";

type Props = {
  peer: PeerView;
  highlighted?: boolean;
  onClick?: () => void;
  // Optional explicit-forget action. When set, an X appears on hover
  // for offline peers so the user can prune their list.
  onForget?: () => void;
};

export function PeerCard({ peer, highlighted, onClick, onForget }: Props) {
  const color = avatarColor(peer.nickname || peer.peerId);
  const initial = avatarInitial(peer.nickname || peer.peerId);
  const offline = peer.state === "closed";
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
      initial={{ opacity: 0, y: 12, scale: 0.92, boxShadow: "0 0 0px 0px rgba(139,92,246,0)" }}
      animate={{
        opacity: 1, y: 0, scale: 1,
        boxShadow: peer.state === "connected"
          ? ["0 0 0px 0px rgba(139,92,246,0)", "0 0 20px -4px rgba(139,92,246,0.45)", "0 0 0px 0px rgba(139,92,246,0)"]
          : "0 0 0px 0px rgba(139,92,246,0)",
      }}
      exit={{ opacity: 0, scale: 0.92, x: -6 }}
      whileHover={{ scale: 1.015, y: -1 }}
      transition={{
        type: "spring", stiffness: 300, damping: 28,
        boxShadow: { duration: 1.4 },
      }}
      className={`panel rounded-card w-full text-left p-3 flex items-center gap-3 transition-colors group relative ${
        highlighted ? "border-accent2/50 bg-white/[0.06]" : ""
      } ${offline ? "opacity-60" : ""}`}
    >
      <div
        className={`w-9 h-9 rounded-full flex items-center justify-center font-semibold text-sm text-white shrink-0 ${
          offline ? "grayscale" : ""
        }`}
        style={{ background: `linear-gradient(135deg, ${color} 0%, #6366f1 100%)` }}
      >
        {initial}
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-1.5">
          <div className="font-medium truncate text-sm">{peer.nickname || shortId(peer.peerId)}</div>
          {peer.isOwner && <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />}
          {offline && (
            <span className="text-[9px] uppercase tracking-wider text-zinc-500 shrink-0">
              offline
            </span>
          )}
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
        {peer.state === "connected" && peer.transport && (
          <TransportBadge transport={peer.transport} />
        )}
      </div>

      {/* Forget button — only renders for offline peers and only when
          the parent supplies onForget (i.e. the row is in a list
          where pruning makes sense). Hover-only so the row stays
          quiet at rest. Stops click bubbling so 'Forget' doesn't
          also trigger the card's onClick. */}
      {offline && onForget && (
        <span
          role="button"
          aria-label="Forget"
          onClick={(e) => {
            e.stopPropagation();
            e.preventDefault();
            onForget();
          }}
          className="absolute top-1.5 right-1.5 p-1 rounded text-zinc-500 hover:text-rose-300 hover:bg-rose-500/10 opacity-0 group-hover:opacity-100 transition cursor-pointer"
          title="Ro'yxatdan o'chirish"
        >
          <X className="w-3 h-3" />
        </span>
      )}
    </motion.button>
  );
}

function TransportBadge({ transport }: { transport: "direct" | "relay" }) {
  if (transport === "direct") {
    return (
      <div
        title="To'g'ridan-to'g'ri P2P (host/srflx) — trafik hech qanday server orqali o'tmayapti"
        className="flex items-center gap-1 text-[10px] font-medium text-emerald-300/90 bg-emerald-400/10 border border-emerald-400/20 rounded px-1.5 py-0.5"
      >
        <Zap className="w-3 h-3" strokeWidth={2.5} />
        <span>P2P</span>
      </div>
    );
  }
  return (
    <div
      title="TURN serveri orqali relay qilinmoqda — trafik shifrlangan, lekin TURN serveridan o'tadi"
      className="flex items-center gap-1 text-[10px] font-medium text-amber-300/90 bg-amber-400/10 border border-amber-400/20 rounded px-1.5 py-0.5"
    >
      <Cloud className="w-3 h-3" strokeWidth={2.5} />
      <span>TURN</span>
    </div>
  );
}
