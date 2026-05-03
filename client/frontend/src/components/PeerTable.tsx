import React from "react";
import { Crown, Zap, Cloud, Wifi, WifiOff, Loader2 } from "lucide-react";
import type { PeerView } from "../types";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { rttLabel, shortId } from "../lib/format";

type Props = {
  selfNickname: string;
  selfVip: string;
  peers: PeerView[];
  hovered?: string | null;
  onHover?: (peerId: string | null) => void;
};

function bytesShort(n: number): string {
  if (!n || n < 1024) return `${n || 0} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 * 1024 * 1024) return `${(n / 1024 / 1024).toFixed(1)} MB`;
  return `${(n / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

export function PeerTable({ selfNickname, selfVip, peers, hovered, onHover }: Props) {
  const others = peers.filter((p) => p.peerId);

  return (
    <div className="w-full max-w-[860px] mx-auto">
      <div className="panel rounded-card overflow-hidden">
        <div className="px-4 py-2.5 border-b border-white/5 flex items-center justify-between">
          <div className="text-xs uppercase tracking-widest text-zinc-400">Mesh</div>
          <div className="text-xs text-zinc-500">
            {others.length === 0
              ? "Faqat siz"
              : `${others.length + 1} ulangan`}
          </div>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="text-[11px] uppercase tracking-wider text-zinc-500 border-b border-white/5">
                <th className="text-left font-medium px-4 py-2">Peer</th>
                <th className="text-left font-medium px-3 py-2">VIP</th>
                <th className="text-left font-medium px-3 py-2">Holat</th>
                <th className="text-left font-medium px-3 py-2">RTT</th>
                <th className="text-left font-medium px-3 py-2">Yo'l</th>
                <th className="text-right font-medium px-4 py-2">Trafik</th>
              </tr>
            </thead>
            <tbody>
              <SelfRow nickname={selfNickname} vip={selfVip} />
              {others.map((p) => (
                <PeerRow
                  key={p.peerId}
                  peer={p}
                  highlighted={hovered === p.peerId}
                  onMouseEnter={() => onHover?.(p.peerId)}
                  onMouseLeave={() => onHover?.(null)}
                />
              ))}
            </tbody>
          </table>
        </div>

        {others.length === 0 && (
          <div className="px-4 py-8 text-center border-t border-white/5">
            <div className="text-xs uppercase tracking-widest text-violet-300/70 mb-1.5">
              Kutilmoqda
            </div>
            <div className="text-sm text-zinc-400">
              Hech kim hali qo'shilmagan.
            </div>
            <div className="text-xs text-zinc-500 mt-1">
              ID + KOD ni do'stingizga ulashing — meshda ko'rinadi.
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

function SelfRow({ nickname, vip }: { nickname: string; vip: string }) {
  const label = nickname || "siz";
  const color = avatarColor(label);
  const initial = avatarInitial(label);
  return (
    <tr className="border-b border-white/5 bg-violet-500/[0.04]">
      <td className="px-4 py-2.5">
        <div className="flex items-center gap-2.5 min-w-0">
          <div
            className="w-7 h-7 rounded-full flex items-center justify-center font-semibold text-xs text-white shrink-0"
            style={{ background: `linear-gradient(135deg, ${color} 0%, #6366f1 100%)` }}
          >
            {initial}
          </div>
          <div className="min-w-0">
            <div className="font-medium truncate">
              {label} <span className="text-zinc-500 text-xs font-normal">(siz)</span>
            </div>
          </div>
        </div>
      </td>
      <td className="px-3 py-2.5 font-mono text-xs text-zinc-300">{vip}</td>
      <td className="px-3 py-2.5">
        <span className="inline-flex items-center gap-1 text-emerald-400 text-xs">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
          aktiv
        </span>
      </td>
      <td className="px-3 py-2.5 text-zinc-500 text-xs font-mono">—</td>
      <td className="px-3 py-2.5 text-zinc-500 text-xs">—</td>
      <td className="px-4 py-2.5 text-right text-xs text-zinc-500 font-mono">—</td>
    </tr>
  );
}

function PeerRow({
  peer,
  highlighted,
  onMouseEnter,
  onMouseLeave,
}: {
  peer: PeerView;
  highlighted: boolean;
  onMouseEnter: () => void;
  onMouseLeave: () => void;
}) {
  const color = avatarColor(peer.nickname || peer.peerId);
  const initial = avatarInitial(peer.nickname || peer.peerId);
  const stateNode = (() => {
    switch (peer.state) {
      case "connected":
        return (
          <span className="inline-flex items-center gap-1 text-emerald-400 text-xs">
            <Wifi className="w-3 h-3" strokeWidth={2.5} /> ulangan
          </span>
        );
      case "connecting":
        return (
          <span className="inline-flex items-center gap-1 text-amber-400 text-xs">
            <Loader2 className="w-3 h-3 animate-spin" strokeWidth={2.5} /> ulanmoqda
          </span>
        );
      case "failed":
        return (
          <span className="inline-flex items-center gap-1 text-rose-400 text-xs">
            <WifiOff className="w-3 h-3" strokeWidth={2.5} /> uzilgan
          </span>
        );
      default:
        return (
          <span className="inline-flex items-center gap-1 text-zinc-500 text-xs">
            <WifiOff className="w-3 h-3" strokeWidth={2.5} /> yopiq
          </span>
        );
    }
  })();

  return (
    <tr
      onMouseEnter={onMouseEnter}
      onMouseLeave={onMouseLeave}
      className={`border-b border-white/5 last:border-b-0 transition-colors ${
        highlighted ? "bg-white/[0.05]" : "hover:bg-white/[0.03]"
      }`}
    >
      <td className="px-4 py-2.5">
        <div className="flex items-center gap-2.5 min-w-0">
          <div
            className="w-7 h-7 rounded-full flex items-center justify-center font-semibold text-xs text-white shrink-0"
            style={{ background: `linear-gradient(135deg, ${color} 0%, #6366f1 100%)` }}
          >
            {initial}
          </div>
          <div className="min-w-0 flex items-center gap-1.5">
            <span className="font-medium truncate">
              {peer.nickname || shortId(peer.peerId)}
            </span>
            {peer.isOwner && (
              <Crown className="w-3 h-3 text-amber-400 shrink-0" strokeWidth={2.5} />
            )}
          </div>
        </div>
      </td>
      <td className="px-3 py-2.5 font-mono text-xs text-zinc-300">{peer.virtualIp}</td>
      <td className="px-3 py-2.5">{stateNode}</td>
      <td className="px-3 py-2.5 font-mono text-xs text-zinc-300">
        {rttLabel(peer.rttMs)}
      </td>
      <td className="px-3 py-2.5">
        {peer.state === "connected" && peer.transport ? (
          peer.transport === "direct" ? (
            <span
              title="To'g'ridan-to'g'ri P2P (host/srflx) — trafik server orqali emas"
              className="inline-flex items-center gap-1 text-[10px] font-medium text-emerald-300/90 bg-emerald-400/10 border border-emerald-400/20 rounded px-1.5 py-0.5"
            >
              <Zap className="w-3 h-3" strokeWidth={2.5} /> P2P
            </span>
          ) : (
            <span
              title="TURN serveri orqali relay qilinmoqda — shifrlangan, lekin TURN'dan o'tadi"
              className="inline-flex items-center gap-1 text-[10px] font-medium text-amber-300/90 bg-amber-400/10 border border-amber-400/20 rounded px-1.5 py-0.5"
            >
              <Cloud className="w-3 h-3" strokeWidth={2.5} /> TURN
            </span>
          )
        ) : (
          <span className="text-zinc-600 text-xs">—</span>
        )}
      </td>
      <td className="px-4 py-2.5 text-right font-mono text-xs text-zinc-400 whitespace-nowrap">
        <span className="text-violet-300/80">↑ {bytesShort(peer.bytesSent)}</span>
        <span className="mx-1.5 text-zinc-600">·</span>
        <span className="text-cyan-300/80">↓ {bytesShort(peer.bytesRecv)}</span>
      </td>
    </tr>
  );
}
