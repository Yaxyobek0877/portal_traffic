// PeerCard — one row in the A'zolar column.
//
// After the Portal layout swap (centre = peer services, right = my
// services + chat), the members column lost the inline service list
// it briefly carried and gained the per-peer diagnostics that used
// to live in PeerTable: real-time state, RTT, P2P-vs-TURN badge,
// bytes sent/received, ICE selected-pair addresses, and an on-demand
// bandwidth probe. Everything you'd want to know "is alice's
// connection fast / direct / falling over?" is on one card.
//
// Layout:
//
//   [Avatar] Nickname (Crown)        ● state  rtt
//            10.42.x.y                [P2P|TURN]
//   ───────────────────────────────────────────
//   ↑ bytesSent      ↓ bytesRecv     [⚡ Tezligi]
//   192.168.x ↔ 10.0.y · LAN
//
// Offline peers (state==='closed') render at opacity-60 with a
// hover-X to forget. Their diagnostic rows hide because the numbers
// are stale; only the header row stays so the user remembers who
// was here.

import React, { useState } from "react";
import { motion } from "framer-motion";
import {
  Crown,
  Wifi,
  WifiOff,
  Zap,
  Cloud,
  X,
  Gauge,
  Loader2,
  Network,
} from "lucide-react";
import type { PeerView, BandwidthResult } from "../types";
import { app } from "../lib/wails";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { rttLabel, shortId, splitNicknameAndDevice } from "../lib/format";

type Props = {
  peer: PeerView;
  highlighted?: boolean;
  onForget?: () => void;
};

// bytesShort formats a byte count compactly. Identical helper to
// the one in the old PeerTable; kept local so PeerCard doesn't
// reach into a sibling component file.
function bytesShort(n: number): string {
  if (!n || n < 1024) return `${n || 0} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

// isPrivateAddr / classifyPath: copied from the old PeerTable so
// the LAN/Internet/TURN label can render inline. The classifier
// looks at whether both selected ICE-pair endpoints are RFC1918,
// link-local, or carrier-NAT — both private = LAN, both public =
// real internet.
function isPrivateAddr(addr: string): boolean {
  if (!addr) return false;
  let host = addr;
  if (host.startsWith("[")) {
    const end = host.indexOf("]");
    host = end > 0 ? host.slice(1, end) : host;
  } else {
    const i = host.lastIndexOf(":");
    if (i > 0) host = host.slice(0, i);
  }
  if (host.startsWith("10.") || host.startsWith("192.168.")) return true;
  if (host.startsWith("172.")) {
    const second = parseInt(host.split(".")[1] || "0", 10);
    if (second >= 16 && second <= 31) return true;
  }
  if (host === "127.0.0.1" || host.startsWith("169.254.")) return true;
  if (host.startsWith("fe80:") || host.startsWith("fc") || host.startsWith("fd")) return true;
  return false;
}

function classifyPath(
  peer: PeerView,
): { label: string; tone: "lan" | "internet" | "relay" | "unknown" } {
  if (peer.transport === "relay") return { label: "TURN relay", tone: "relay" };
  if (!peer.pathLocalAddr || !peer.pathRemoteAddr)
    return { label: "—", tone: "unknown" };
  const localPriv = isPrivateAddr(peer.pathLocalAddr);
  const remotePriv = isPrivateAddr(peer.pathRemoteAddr);
  if (localPriv && remotePriv) return { label: "LAN", tone: "lan" };
  if (!localPriv && !remotePriv) return { label: "Internet", tone: "internet" };
  return { label: "Aralash", tone: "internet" };
}

export function PeerCard({ peer, highlighted, onForget }: Props) {
  const color = avatarColor(peer.nickname || peer.peerId);
  const initial = avatarInitial(peer.nickname || peer.peerId);
  const offline = peer.state === "closed";
  const path = classifyPath(peer);

  const [bw, setBw] = useState<BandwidthResult | null>(null);
  const [bwRunning, setBwRunning] = useState(false);
  const [bwError, setBwError] = useState<string | null>(null);

  const runSpeedTest = async () => {
    setBwError(null);
    setBwRunning(true);
    try {
      const r = await app.MeasureBandwidth(peer.peerId);
      setBw(r);
    } catch (e: any) {
      setBwError(e?.message || String(e));
    } finally {
      setBwRunning(false);
    }
  };

  const dot =
    peer.state === "connected"
      ? "bg-emerald-400 shadow-[0_0_10px_#34d399]"
      : peer.state === "failed"
      ? "bg-rose-400 shadow-[0_0_10px_#fb7185]"
      : peer.state === "closed"
      ? "bg-zinc-500"
      : "bg-amber-400 shadow-[0_0_10px_#fbbf24]";

  const stateLabel = (() => {
    switch (peer.state) {
      case "connected":
        return "ulangan";
      case "connecting":
        return "ulanmoqda";
      case "failed":
        return "uzilgan";
      case "closed":
        return "offline";
      default:
        return "—";
    }
  })();

  const stateToneClass = {
    connected: "text-emerald-400",
    connecting: "text-amber-400",
    failed: "text-rose-400",
    closed: "text-zinc-500",
  }[peer.state || "closed"];

  const pathToneClass = {
    lan: "text-emerald-300/80",
    internet: "text-cyan-300/80",
    relay: "text-amber-300/80",
    unknown: "text-zinc-600",
  }[path.tone];

  const showDiag = !offline; // hide stale numbers on offline rows
  const showPath =
    peer.state === "connected" && (peer.pathLocalAddr || peer.pathRemoteAddr);

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 12, scale: 0.92 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, scale: 0.92, x: -6 }}
      transition={{ type: "spring", stiffness: 300, damping: 28 }}
      className={`panel rounded-card p-3 group relative transition-colors ${
        highlighted ? "border-accent2/50 bg-white/[0.06]" : ""
      } ${offline ? "opacity-60" : ""}`}
    >
      {/* Header row — avatar / name / state */}
      <div className="flex items-center gap-3">
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
            {(() => {
              // 'username@device' on the wire → 'username · device'
              // in the UI. See lib/format.splitNicknameAndDevice.
              const { nickname, device } = splitNicknameAndDevice(
                peer.nickname || shortId(peer.peerId),
              );
              return (
                <>
                  <div className="font-medium truncate text-sm">{nickname}</div>
                  {device && (
                    <span className="text-[10px] text-zinc-500 font-mono shrink-0">
                      · {device}
                    </span>
                  )}
                </>
              );
            })()}
            {peer.isOwner && (
              <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />
            )}
          </div>
          <div className="text-xs text-zinc-400 font-mono mt-0.5">
            {peer.virtualIp}
          </div>
        </div>

        <div className="flex flex-col items-end gap-1 shrink-0">
          <div className="flex items-center gap-1.5">
            <span className={`w-2 h-2 rounded-full ${dot}`} />
            <span className={`text-[10px] uppercase tracking-wider ${stateToneClass}`}>
              {stateLabel}
            </span>
          </div>
          {peer.state === "connected" && (
            <div className="text-[11px] text-zinc-500 font-mono">
              {rttLabel(peer.rttMs)}
            </div>
          )}
        </div>
      </div>

      {/* Diagnostic block — transport badge + bandwidth + traffic
          totals + path. Hidden for offline peers since the numbers
          would be stale. */}
      {showDiag && (
        <div className="mt-2.5 pt-2.5 border-t border-white/[0.04] space-y-1.5">
          {/* Transport + traffic counters on one row, with the
              speedtest button anchored to the right. */}
          <div className="flex items-center gap-2 flex-wrap">
            {peer.state === "connected" && peer.transport ? (
              peer.transport === "direct" ? (
                <span
                  title="To'g'ridan-to'g'ri P2P (host/srflx) — trafik server orqali emas"
                  className="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-300/90 bg-emerald-400/10 border border-emerald-400/20 rounded px-1.5 py-0.5 shrink-0"
                >
                  <Zap className="w-3 h-3" strokeWidth={2.5} /> P2P
                </span>
              ) : (
                <span
                  title="TURN serveri orqali relay qilinmoqda"
                  className="inline-flex items-center gap-1 text-[10px] font-medium text-amber-300/90 bg-amber-400/10 border border-amber-400/20 rounded px-1.5 py-0.5 shrink-0"
                >
                  <Cloud className="w-3 h-3" strokeWidth={2.5} /> TURN
                </span>
              )
            ) : peer.state === "connecting" ? (
              <span className="inline-flex items-center gap-1 text-[10px] text-amber-300/80 shrink-0">
                <Loader2 className="w-3 h-3 animate-spin" strokeWidth={2.5} />
                ICE tekshirilmoqda
              </span>
            ) : null}

            {/* Traffic counters — single line, monospace, tiny so
                they don't dominate the row. Updates as the mesh
                reports new totals. */}
            <span className="text-[10px] font-mono text-zinc-500 ml-auto whitespace-nowrap">
              <span className="text-violet-300/80">↑ {bytesShort(peer.bytesSent)}</span>
              <span className="mx-1 text-zinc-700">·</span>
              <span className="text-cyan-300/80">↓ {bytesShort(peer.bytesRecv)}</span>
            </span>
          </div>

          {/* Path: ICE selected-pair addresses + LAN/Internet/TURN
              tag. Only shows for connected peers; the addresses are
              what a user grep'ing for "are we even on the same Wi-Fi"
              wants to verify. */}
          {showPath && (
            <div className="flex items-center gap-1.5 text-[10px] text-zinc-500 font-mono">
              <Network className="w-3 h-3 shrink-0" strokeWidth={2} />
              <span className="truncate text-zinc-400">{peer.pathLocalAddr || "—"}</span>
              <span className="text-zinc-700 shrink-0">↔</span>
              <span className="truncate text-zinc-400">{peer.pathRemoteAddr || "—"}</span>
              <span className={`ml-1 shrink-0 ${pathToneClass}`}>· {path.label}</span>
            </div>
          )}

          {/* Bandwidth probe — last result on the left, button on
              the right. Click runs a 3s active probe and updates
              both the button label (Mbps) and a tiny stamp. */}
          {peer.state === "connected" && (
            <div className="flex items-center gap-2 pt-0.5">
              {bw && !bwRunning && (
                <span
                  className="text-[10px] font-mono text-cyan-300/80"
                  title={`Probe: ${(bw.bytesSent / 1024 / 1024).toFixed(1)} MB · ${bw.durationMs.toFixed(0)} ms`}
                >
                  {bw.mbps.toFixed(1)} Mbps
                </span>
              )}
              {bwError && (
                <span
                  className="text-[10px] text-rose-400 truncate"
                  title={bwError}
                >
                  {bwError}
                </span>
              )}
              <button
                onClick={runSpeedTest}
                disabled={bwRunning}
                className="ml-auto inline-flex items-center gap-1 text-[10px] font-medium text-violet-300 bg-violet-500/15 hover:bg-violet-500/25 disabled:opacity-50 rounded px-2 py-1 transition-colors shrink-0"
                title="3 sekundlik bandwidth probe — peer'ga binar oqim yuborib, qabul qiluvchi tomondan o'lchaydi"
              >
                {bwRunning ? (
                  <Loader2 className="w-3 h-3 animate-spin" strokeWidth={2.5} />
                ) : (
                  <Gauge className="w-3 h-3" strokeWidth={2.5} />
                )}
                {bw ? "Qayta" : bwRunning ? "tekshirilmoqda…" : "Tezligi"}
              </button>
            </div>
          )}
        </div>
      )}

      {/* Forget — top-right, hover-only, only for offline peers. */}
      {offline && onForget && (
        <button
          onClick={(e) => {
            e.stopPropagation();
            onForget();
          }}
          className="absolute top-1.5 right-1.5 p-1 rounded text-zinc-500 hover:text-rose-300 hover:bg-rose-500/10 opacity-0 group-hover:opacity-100 transition"
          title="Ro'yxatdan o'chirish"
        >
          <X className="w-3 h-3" />
        </button>
      )}
    </motion.div>
  );
}
