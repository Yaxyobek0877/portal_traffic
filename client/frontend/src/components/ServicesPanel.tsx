import React, { useEffect, useState } from "react";
import { Plus, Server, Link2, Trash2, Globe, Search, Zap, Copy, Check } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { LocalListener, PeerView, ServiceView } from "../types";
import { app } from "../lib/wails";
import { shortId } from "../lib/format";

type Props = {
  localServices: ServiceView[];
  peers: PeerView[];
  refreshLocalServices: () => Promise<void>;
};

export function ServicesPanel({ localServices, peers, refreshLocalServices }: Props) {
  const [name, setName] = useState("");
  const [port, setPort] = useState<number | "">("");
  const [proto, setProto] = useState<"tcp" | "udp">("tcp");
  const [target, setTarget] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [error, setError] = useState("");
  const [dialing, setDialing] = useState<{ peerId: string; port: number; protocol: string } | null>(null);
  const [dialedAddrs, setDialedAddrs] = useState<Record<string, string>>({});
  const [detected, setDetected] = useState<LocalListener[]>([]);
  const [scanning, setScanning] = useState(false);

  const refreshDetected = async () => {
    setScanning(true);
    try {
      const list = await app.LocalListeners();
      setDetected(list);
    } finally {
      setScanning(false);
    }
  };

  useEffect(() => {
    refreshDetected();
  }, []);

  const exposeDetected = async (l: LocalListener) => {
    try {
      // Detected listeners are by definition local; target stays empty
      // → defaults to 127.0.0.1:port on the Go side.
      await app.ExposeService(l.process || `${l.protocol}:${l.port}`, l.protocol, l.port, "");
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const submitExpose = async () => {
    setError("");
    if (typeof port !== "number" || port < 1 || port > 65535) {
      setError("Port 1–65535 oralig'ida bo'lishi kerak");
      return;
    }
    const trimmedTarget = target.trim();
    if (trimmedTarget && !/^[\w.\-]+:\d{1,5}$/.test(trimmedTarget)) {
      setError("Target host:port shaklida bo'lishi kerak (masalan 192.168.1.100:554)");
      return;
    }
    try {
      await app.ExposeService(name || `${proto}:${port}`, proto, port, trimmedTarget);
      setName("");
      setPort("");
      setTarget("");
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const removeExposed = async (p: number) => {
    try {
      await app.UnexposeService(p);
      await refreshLocalServices();
    } catch {}
  };

  const dialPeerService = async (peer: PeerView, svc: ServiceView) => {
    setDialing({ peerId: peer.peerId, port: svc.port, protocol: svc.protocol });
    try {
      const addr = await app.DialService(peer.peerId, svc.protocol as "tcp" | "udp", svc.port, 0);
      setDialedAddrs((s) => ({ ...s, [`${peer.peerId}:${svc.protocol}:${svc.port}`]: addr }));
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setDialing(null);
    }
  };

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-white/5">
        <div className="flex items-center gap-2 mb-3">
          <Server className="w-4 h-4 text-violet-400" strokeWidth={2} />
          <h3 className="font-semibold text-sm">Mening servislarim</h3>
        </div>
        <p className="text-xs text-zinc-500 mb-3">
          Lokal portni mesh ga oching — boshqa peerlar to'g'ridan-to'g'ri ulana oladi.
        </p>
        <div className="flex flex-wrap gap-2 items-stretch">
          <input
            type="text"
            placeholder="Nom (minecraft / cs2)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input-base text-sm flex-1 min-w-[140px]"
          />
          <div className="flex gap-2 items-stretch">
            <div className="flex rounded overflow-hidden border border-white/10 text-[11px] font-mono shrink-0">
              <button
                type="button"
                onClick={() => setProto("tcp")}
                className={`px-2 ${proto === "tcp" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                title="HTTP, SSH, Minecraft Java — TCP"
              >
                TCP
              </button>
              <button
                type="button"
                onClick={() => setProto("udp")}
                className={`px-2 ${proto === "udp" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                title="CS2, Valorant, Minecraft Bedrock — UDP"
              >
                UDP
              </button>
            </div>
            <input
              type="number"
              placeholder="Port"
              value={port}
              onChange={(e) => setPort(e.target.value === "" ? "" : Number(e.target.value))}
              className="input-base w-24 text-sm font-mono"
            />
            <button
              onClick={submitExpose}
              className="btn-primary rounded-btn px-3 flex items-center gap-1 text-sm shrink-0"
            >
              <Plus className="w-4 h-4" strokeWidth={2} />
              Och
            </button>
          </div>
        </div>
        <button
          type="button"
          onClick={() => setShowAdvanced((v) => !v)}
          className="mt-2 text-[11px] text-zinc-400 hover:text-zinc-200"
        >
          {showAdvanced ? "− LAN target" : "+ LAN qurilma (NVR / kamera / printer)"}
        </button>
        {showAdvanced && (
          <div className="mt-2">
            <input
              type="text"
              placeholder="192.168.1.100:554  (bo'sh = localhost)"
              value={target}
              onChange={(e) => setTarget(e.target.value)}
              className="input-base text-sm w-full font-mono"
            />
            <p className="text-[10px] text-zinc-500 mt-1 leading-relaxed">
              Tarmoqdagi boshqa qurilmaga forward qilish. Misol: RTSP kamera{" "}
              <code className="font-mono text-zinc-400">192.168.1.100:554</code>,
              NVR <code className="font-mono text-zinc-400">192.168.1.50:8000</code>,
              printer <code className="font-mono text-zinc-400">192.168.1.7:631</code>.
              Bo'sh qoldirilsa, lokalda turibdi deb qabul qilinadi.
            </p>
          </div>
        )}
        {error && <div className="text-xs text-rose-400 mt-2">{error}</div>}

        <AnimatePresence>
          {localServices.length > 0 && (
            <motion.div layout className="mt-3 space-y-1.5">
              {localServices.map((s) => (
                <motion.div
                  key={`${s.protocol}:${s.port}`}
                  layout
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  className="panel rounded-input px-3 py-2 flex items-center gap-2 text-sm"
                >
                  <Globe className="w-3.5 h-3.5 text-emerald-400 shrink-0" strokeWidth={2} />
                  <span className="font-medium truncate flex-1 min-w-0">{s.name}</span>
                  <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
                    s.protocol === "udp" ? "bg-cyan-400/10 text-cyan-300" : "bg-violet-400/10 text-violet-300"
                  }`}>
                    {s.protocol.toUpperCase()}
                  </span>
                  <span className="font-mono text-xs text-zinc-500 shrink-0">:{s.port}</span>
                  <button
                    onClick={() => removeExposed(s.port)}
                    className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-rose-400 shrink-0"
                    title="Olib tashlash"
                  >
                    <Trash2 className="w-3.5 h-3.5" strokeWidth={2} />
                  </button>
                </motion.div>
              ))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-5">
        <div>
          <div className="flex items-center justify-between mb-2.5">
            <h3 className="font-semibold text-sm flex items-center gap-2">
              <Search className="w-4 h-4 text-amber-400" strokeWidth={2} />
              Lokalda topilgan portlar
            </h3>
            <button
              onClick={refreshDetected}
              disabled={scanning}
              className="text-[11px] text-zinc-400 hover:text-white disabled:opacity-50"
            >
              {scanning ? "Skanerlanmoqda..." : "Yangilash"}
            </button>
          </div>
          {detected.length === 0 && !scanning && (
            <div className="text-xs text-zinc-500 panel rounded-input px-3 py-3 text-center">
              Lokalda port topilmadi (yoki barchasi tizim portlari).
            </div>
          )}
          <div className="space-y-1.5">
            {detected.map((d) => {
              const exposed = localServices.some(
                (s) => s.port === d.port && s.protocol === d.protocol,
              );
              return (
                <div
                  key={`${d.protocol}:${d.port}`}
                  className="panel rounded-input px-3 py-2 flex items-center gap-2 text-sm"
                >
                  <Zap className="w-3.5 h-3.5 text-amber-400 shrink-0" strokeWidth={2} />
                  <span className="truncate flex-1 min-w-0">{d.process || "?"}</span>
                  <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
                    d.protocol === "udp" ? "bg-cyan-400/10 text-cyan-300" : "bg-violet-400/10 text-violet-300"
                  }`}>
                    {d.protocol.toUpperCase()}
                  </span>
                  <span className="font-mono text-xs text-zinc-500 shrink-0">:{d.port}</span>
                  {exposed ? (
                    <span className="text-[11px] text-emerald-400 font-medium shrink-0">ochilgan ✓</span>
                  ) : (
                    <button
                      onClick={() => exposeDetected(d)}
                      className="px-2 py-1 rounded text-[11px] font-medium bg-violet-500/15 text-violet-300 hover:bg-violet-500/25 shrink-0 whitespace-nowrap"
                    >
                      Och
                    </button>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <div>
          <h3 className="font-semibold text-sm mb-3 flex items-center gap-2">
            <Link2 className="w-4 h-4 text-cyan-400" strokeWidth={2} />
            Boshqa peerlardagi servislar
          </h3>
          {peers.every((p) => p.services.length === 0) && (
            <div className="text-xs text-zinc-500 text-center py-6 panel rounded-input">
              Hozircha hech kim servis e'lon qilmagan.
            </div>
          )}
          <div className="space-y-3">
          {peers.map((p) =>
            p.services.length === 0 ? null : (
              <div key={p.peerId} className="panel rounded-card p-3">
                <div className="flex items-center gap-2 mb-2 min-w-0">
                  <span className="text-sm font-medium truncate min-w-0">{p.nickname || shortId(p.peerId)}</span>
                  <span className="text-xs text-zinc-500 font-mono shrink-0">{p.virtualIp}</span>
                </div>
                <div className="space-y-1.5">
                  {p.services.map((s) => {
                    const key = `${p.peerId}:${s.protocol}:${s.port}`;
                    const local = dialedAddrs[key];
                    const isDialing = dialing?.peerId === p.peerId &&
                      dialing?.port === s.port && dialing?.protocol === s.protocol;
                    return (
                      <div
                        key={`${s.protocol}:${s.port}`}
                        className="flex items-center gap-2 text-xs bg-black/20 rounded p-2"
                      >
                        <Globe className="w-3 h-3 text-cyan-400 shrink-0" strokeWidth={2} />
                        <span className="truncate flex-1 min-w-0">{s.name}</span>
                        <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
                          s.protocol === "udp" ? "bg-cyan-400/10 text-cyan-300" : "bg-violet-400/10 text-violet-300"
                        }`}>
                          {s.protocol.toUpperCase()}
                        </span>
                        <span className="font-mono text-zinc-500 shrink-0">:{s.port}</span>
                        {local ? (
                          <DialedPill addr={local} expectedPort={s.port} />
                        ) : (
                          <button
                            disabled={isDialing}
                            onClick={() => dialPeerService(p, s)}
                            className="px-2 py-1 rounded bg-violet-500/15 text-violet-300 hover:bg-violet-500/25 text-[11px] font-medium disabled:opacity-50 shrink-0"
                          >
                            Ulash
                          </button>
                        )}
                      </div>
                    );
                  })}
                </div>
              </div>
            )
          )}
          </div>
        </div>
      </div>
    </div>
  );
}

// DialedPill is shown after a successful Dial. The address shown is a
// LOCAL listener on this machine; connecting to it gets forwarded
// through the mesh to the peer's exposed port. We label it "lokal" so
// users don't mistake it for the remote endpoint.
//
// `expectedPort` is the remote port. We try to bind a matching local
// port (so 127.0.0.1:5000 mirrors peer:5000); when that fails because
// the user already has something on the same port locally, the proxy
// falls back to OS-pick — we surface that mismatch with an amber tone
// so the user knows the random number isn't a bug.
function DialedPill({ addr, expectedPort }: { addr: string; expectedPort: number }) {
  const [copied, setCopied] = useState(false);
  const localPort = parseInt(addr.split(":").pop() || "0", 10);
  const portMatches = localPort === expectedPort;
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(addr);
      setCopied(true);
      setTimeout(() => setCopied(false), 1400);
    } catch {}
  };
  const tint = portMatches
    ? "bg-emerald-500/15 hover:bg-emerald-500/25 text-emerald-300"
    : "bg-amber-500/15 hover:bg-amber-500/25 text-amber-300";
  const labelTint = portMatches ? "text-emerald-300/60" : "text-amber-300/70";
  const iconTint = portMatches ? "text-emerald-300/60 group-hover:text-emerald-300" : "text-amber-300/70 group-hover:text-amber-300";
  const title = portMatches
    ? `Lokal alias — shu manzilga ulansangiz, mesh orqali peer servisiga yo'naltiriladi. Nusxa olish: ${addr}`
    : `Lokal alias. Sizda :${expectedPort} band edi — Portal :${localPort} ni tanladi. Mesh orqali peer servisiga yo'naltiriladi. Nusxa olish: ${addr}`;
  return (
    <button
      onClick={copy}
      className={`group flex items-center gap-1.5 px-2 py-1 rounded transition-colors shrink-0 ${tint}`}
      title={title}
    >
      <span className={`text-[10px] uppercase tracking-wider ${labelTint}`}>
        {portMatches ? "lokal" : "lokal*"}
      </span>
      <span className="font-mono text-[11px]">{addr}</span>
      {copied ? (
        <Check className="w-3 h-3" strokeWidth={2.5} />
      ) : (
        <Copy className={`w-3 h-3 ${iconTint}`} strokeWidth={2} />
      )}
    </button>
  );
}
