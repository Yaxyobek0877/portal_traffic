// PeerServicesGrid — the central "what can I connect to?" view.
//
// Shows every announced service from every live peer as a prominent
// card with a single Ulash button. The user's primary in-portal
// task is "join my friend's game / open their NVR / SSH into their
// machine"; that path used to take three steps (find the right tab,
// scroll to the peer, click their service); now the dial is the
// front-and-centre column.
//
// Cards are grouped by peer so RTSP-on-alice and RTSP-on-bob don't
// blur together. Empty state ("hech kim hali servis e'lon qilmagan")
// renders when no live peer has any services — telling the user to
// share the portal ID + code with someone who'll expose stuff.
//
// On a successful Ulash, the card swaps the button for a
// copy-to-clipboard pill carrying the local alias (e.g.
// 127.0.0.1:27015). Pill is green when the local port mirrors the
// remote one, amber when proxy.DialPreferringPort had to OS-pick
// because the mirror was already taken on the user's side.

import React, { useState } from "react";
import { Link2, Globe, Copy, Check, Crown } from "lucide-react";
import type { PeerView, ServiceView } from "../types";
import { app } from "../lib/wails";
import { avatarColor, avatarInitial } from "../lib/avatar";
import { shortId } from "../lib/format";

export function PeerServicesGrid({ peers }: { peers: PeerView[] }) {
  const live = peers.filter((p) => p.state !== "closed");
  const total = live.reduce((n, p) => n + p.services.length, 0);

  return (
    <div className="h-full flex flex-col min-h-0">
      <div className="px-5 pt-5 pb-3 border-b border-white/5">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-base flex items-center gap-2">
            <Link2 className="w-4 h-4 text-cyan-400" strokeWidth={2} />
            Boshqalar ulashgan servislar
          </h2>
          {total > 0 && (
            <span className="text-xs text-zinc-500 font-mono">
              {total} ta servis · {live.filter((p) => p.services.length > 0).length} peer
            </span>
          )}
        </div>
        <p className="text-xs text-zinc-500 mt-1">
          Peer'lar e'lon qilgan servislar — <span className="text-violet-300">Ulash</span> bossangiz lokal alias ochiladi va shu manzilga ulansangiz mesh orqali boshqa qurilmaga yo'naltiriladi.
        </p>
      </div>

      <div className="flex-1 overflow-y-auto p-5 space-y-4">
        {total === 0 && (
          <div className="text-sm text-zinc-500 text-center py-12 panel rounded-card">
            Hozircha hech kim servis e'lon qilmagan.
            <div className="mt-1.5 text-xs text-zinc-600">
              Do'stingiz portalga kirib biror portni Och bossa, shu yerda paydo bo'ladi.
            </div>
          </div>
        )}
        {live.map((peer) =>
          peer.services.length === 0 ? null : (
            <PeerSection key={peer.peerId} peer={peer} />
          ),
        )}
      </div>
    </div>
  );
}

// PeerSection groups one peer's announced services under a header
// row carrying their avatar / nickname / VIP. Each service is its
// own card-row with a prominent Ulash button on the right.
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
          {error && <span className="text-[10px] text-rose-400 truncate" title={error}>{error}</span>}
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
