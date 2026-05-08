// PeerServicesGrid — the central "what services are in this room?"
// view.
//
// User asked for THEIR own opened services to live in the same grid
// as other peers' (so 'mening' and 'boshqalar' are side by side, not
// in two different columns). Layout:
//
//   ┌─ Servislar (bu xonada) ─────────────────┐
//   │                                          │
//   │ ┌─ Siz ──────────────────────────────┐  │
//   │ │ ● kamera  TCP :554 → 192.168.x:554 │  │
//   │ │   (LAN, health=ok)    pencil pause │  │
//   │ │ ● ssh     TCP :22                  │  │
//   │ └────────────────────────────────────┘  │
//   │                                          │
//   │ ┌─ Alice ────────────────────────────┐  │
//   │ │ ● minecraft TCP :25565   [Ulash]   │  │
//   │ └────────────────────────────────────┘  │
//   │                                          │
//   │ ┌─ Bob ──────────────────────────────┐  │
//   │ │ ● cs2 UDP :27015         [Ulash]   │  │
//   │ └────────────────────────────────────┘  │
//   └──────────────────────────────────────────┘
//
// Own services use the existing ExposedServiceRow (target editor /
// pause / remove); peer services use the dial-button card row.
// Sections are visually distinct so the user always knows which is
// theirs vs. someone else's.

import React, { useState } from "react";
import { Link2, Globe, Copy, Check, Crown } from "lucide-react";
import type { PeerView, ServiceView } from "../types";
import { app } from "../lib/wails";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { shortId } from "../lib/format";
import { ExposedServiceRow } from "./ExposedServiceRow";

type Props = {
  // Own services (the user has opened these to the room).
  ownServices: ServiceView[];
  ownNickname: string;
  ownVip: string;

  // Other peers and their announced services.
  peers: PeerView[];

  // Callbacks for own-service actions. Hoisted to the parent so the
  // grid doesn't need to know about Wails directly.
  onTogglePause: (s: ServiceView) => void;
  onRemoveOwn: (port: number) => void;
  onRetargetOwn: (s: ServiceView, target: string) => Promise<boolean>;
};

export function PeerServicesGrid({
  ownServices,
  ownNickname,
  ownVip,
  peers,
  onTogglePause,
  onRemoveOwn,
  onRetargetOwn,
}: Props) {
  const live = peers.filter((p) => p.state !== "closed");
  const peerCount = live.filter((p) => p.services.length > 0).length;
  const peerServiceCount = live.reduce((n, p) => n + p.services.length, 0);
  const totalServices = ownServices.length + peerServiceCount;
  const empty = totalServices === 0;

  return (
    <div className="h-full flex flex-col min-h-0">
      <div className="px-5 pt-5 pb-3 border-b border-white/5">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <h2 className="font-semibold text-base flex items-center gap-2">
            <Link2 className="w-4 h-4 text-cyan-400" strokeWidth={2} />
            Servislar
          </h2>
          {!empty && (
            <span className="text-xs text-zinc-500 font-mono">
              {totalServices} ta · {ownServices.length > 0 ? `siz ${ownServices.length}` : "siz 0"}
              {peerCount > 0 ? ` · ${peerCount} peer` : ""}
            </span>
          )}
        </div>
        <p className="text-xs text-zinc-500 mt-1">
          Bu xonadagi barcha ochilgan portlar. <span className="text-violet-300">Ulash</span> bossangiz lokal alias ochiladi va shu manzilga ulansangiz mesh orqali peer servisiga yo'naltiriladi.
        </p>
      </div>

      <div className="flex-1 overflow-y-auto p-5 space-y-4">
        {empty && (
          <div className="text-sm text-zinc-500 text-center py-12 panel rounded-card">
            Hozircha hech kim biror port ochmagan.
            <div className="mt-1.5 text-xs text-zinc-600">
              Yuqoridagi forma orqali o'zingiz port oching, yoki do'stingiz kirib bir narsa ulashishini kuting.
            </div>
          </div>
        )}

        {/* Section 1: SIZ — the user's own opened services. Renders
            ExposedServiceRow for each so the pencil/pause/remove
            controls and the cyan LAN pill are identical to what
            ServicesPanel used to show. */}
        {ownServices.length > 0 && (
          <OwnSection
            services={ownServices}
            nickname={ownNickname}
            vip={ownVip}
            onTogglePause={onTogglePause}
            onRemove={onRemoveOwn}
            onRetarget={onRetargetOwn}
          />
        )}

        {/* Section 2..N: per-peer services with dial buttons. */}
        {live.map((peer) =>
          peer.services.length === 0 ? null : (
            <PeerSection key={peer.peerId} peer={peer} />
          ),
        )}
      </div>
    </div>
  );
}

function OwnSection({
  services,
  nickname,
  vip,
  onTogglePause,
  onRemove,
  onRetarget,
}: {
  services: ServiceView[];
  nickname: string;
  vip: string;
  onTogglePause: (s: ServiceView) => void;
  onRemove: (port: number) => void;
  onRetarget: (s: ServiceView, target: string) => Promise<boolean>;
}) {
  const initial = (nickname.trim()[0] || "Y").toUpperCase();
  return (
    <section className="panel rounded-card overflow-hidden border-violet-500/20">
      <header className="px-4 py-2.5 flex items-center gap-3 border-b border-white/[0.04] bg-violet-500/[0.06]">
        <div className="w-8 h-8 rounded-full bg-gradient-to-br from-violet-500 to-cyan-400 flex items-center justify-center font-semibold text-xs text-white shrink-0">
          {initial}
        </div>
        <div className="min-w-0 flex-1">
          <div className="font-medium text-sm flex items-center gap-1.5">
            <span className="truncate">{nickname || "Siz"}</span>
            <span className="text-[10px] uppercase tracking-wider text-violet-300 shrink-0">
              siz
            </span>
          </div>
          <div className="text-[11px] text-zinc-500 font-mono">{vip}</div>
        </div>
        <span className="text-[10px] text-zinc-600 font-mono shrink-0">
          {services.length} ta
        </span>
      </header>
      <div className="p-3 space-y-1.5">
        {services.map((s) => (
          <ExposedServiceRow
            key={`${s.protocol}:${s.port}`}
            s={s}
            onTogglePause={() => onTogglePause(s)}
            onRemove={() => onRemove(s.port)}
            onRetarget={(t) => onRetarget(s, t)}
          />
        ))}
      </div>
    </section>
  );
}

function PeerSection({ peer }: { peer: PeerView }) {
  const color = avatarColor(peer.nickname || peer.peerId);
  const initial = avatarInitial(peer.nickname || peer.peerId);
  return (
    <section className="panel rounded-card overflow-hidden">
      <header className="px-4 py-2.5 flex items-center gap-3 border-b border-white/[0.04] bg-white/[0.02]">
        <div
          className="w-8 h-8 rounded-full flex items-center justify-center font-semibold text-xs text-white shrink-0"
          style={{ background: `linear-gradient(135deg, ${color} 0%, #6366f1 100%)` }}
        >
          {initial}
        </div>
        <div className="min-w-0 flex-1">
          <div className="font-medium text-sm flex items-center gap-1.5">
            <span className="truncate">{peer.nickname || shortId(peer.peerId)}</span>
            {peer.isOwner && <Crown className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />}
          </div>
          <div className="text-[11px] text-zinc-500 font-mono">{peer.virtualIp}</div>
        </div>
        <span className="text-[10px] text-zinc-600 font-mono shrink-0">
          {peer.services.length} ta
        </span>
      </header>
      <div className="divide-y divide-white/[0.04]">
        {peer.services.map((svc) => (
          <ServiceRow
            key={`${svc.protocol}:${svc.port}`}
            peer={peer}
            svc={svc}
          />
        ))}
      </div>
    </section>
  );
}

function ServiceRow({ peer, svc }: { peer: PeerView; svc: ServiceView }) {
  const [dialing, setDialing] = useState(false);
  const [addr, setAddr] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const dial = async () => {
    setDialing(true);
    setError(null);
    try {
      const a = await app.DialService(
        peer.peerId,
        svc.protocol as "tcp" | "udp",
        svc.port,
        0,
      );
      setAddr(a);
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setDialing(false);
    }
  };

  return (
    <div className="px-4 py-3 flex items-center gap-3 text-sm hover:bg-white/[0.02] flex-wrap">
      <Globe className="w-4 h-4 text-cyan-400 shrink-0" strokeWidth={2} />
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{svc.name}</div>
        <div className="flex items-center gap-2 mt-0.5">
          <span
            className={`text-[10px] font-mono px-1.5 py-0.5 rounded ${
              svc.protocol === "udp"
                ? "bg-cyan-400/10 text-cyan-300"
                : "bg-violet-400/10 text-violet-300"
            }`}
          >
            {svc.protocol.toUpperCase()}
          </span>
          <span className="font-mono text-[11px] text-zinc-500">:{svc.port}</span>
          {error && (
            <span className="text-[10px] text-rose-400 truncate" title={error}>
              {error}
            </span>
          )}
        </div>
      </div>
      {addr ? (
        <DialedPill addr={addr} expectedPort={svc.port} />
      ) : (
        <button
          disabled={dialing}
          onClick={dial}
          className="btn-primary rounded-btn px-4 py-2 text-xs font-semibold disabled:opacity-50 shrink-0"
        >
          {dialing ? "Ulanmoqda…" : "Ulash"}
        </button>
      )}
    </div>
  );
}

function DialedPill({ addr, expectedPort }: { addr: string; expectedPort: number }) {
  const [copied, setCopied] = useState(false);
  const localPort = parseInt(addr.split(":").pop() || "0", 10);
  const matches = localPort === expectedPort;
  const tint = matches
    ? "bg-emerald-500/15 hover:bg-emerald-500/25 text-emerald-300 border-emerald-500/30"
    : "bg-amber-500/15 hover:bg-amber-500/25 text-amber-300 border-amber-500/30";
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
      className={`flex items-center gap-1.5 px-3 py-2 rounded-btn border text-xs font-mono shrink-0 ${tint}`}
      title={
        matches
          ? `Lokal alias: ${addr}. Mesh orqali peer servisiga yo'naltiriladi.`
          : `Lokal :${expectedPort} band edi — Portal :${localPort} ni tanladi.`
      }
    >
      {addr}
      {copied ? (
        <Check className="w-3 h-3" strokeWidth={2.5} />
      ) : (
        <Copy className="w-3 h-3" strokeWidth={2} />
      )}
    </button>
  );
}
