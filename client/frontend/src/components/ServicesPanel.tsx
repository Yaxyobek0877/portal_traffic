import React, { useEffect, useState } from "react";
import { Plus, Server, Link2, Trash2, Globe, Search, Zap, Copy, Check, Radar, Sparkles, Pause, Play, Pencil, X } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import type { LocalListener, LANDiscovery, PeerView, ServiceView, RiskAssessment } from "../types";
import { app, subscribe } from "../lib/wails";
import { shortId } from "../lib/format";

type Props = {
  localServices: ServiceView[];
  peers: PeerView[];
  refreshLocalServices: () => Promise<void>;
};

// Common services people forward through Portal — one-click presets
// for the expose form. Order is roughly "most-asked-for first".
type Preset = {
  id: string;
  label: string;
  defaultName: string;
  protocol: "tcp" | "udp";
  port: number;
  showAdvanced?: boolean; // open the LAN-target field for hardware devices
  hint: string;
};

const presets: Preset[] = [
  { id: "minecraft-java",    label: "Minecraft Java",        defaultName: "minecraft",   protocol: "tcp", port: 25565, hint: "Vanilla Java edition default" },
  { id: "minecraft-bedrock", label: "Minecraft Bedrock",     defaultName: "minecraft-be", protocol: "udp", port: 19132, hint: "Pocket / Win10 / Switch / mobile" },
  { id: "cs2",               label: "CS2 / Source",          defaultName: "cs2",         protocol: "udp", port: 27015, hint: "Counter-Strike 2 / Source dedicated server" },
  { id: "rust",              label: "Rust",                  defaultName: "rust",        protocol: "udp", port: 28015, hint: "Facepunch's Rust server" },
  { id: "factorio",          label: "Factorio",              defaultName: "factorio",    protocol: "udp", port: 34197, hint: "" },
  { id: "terraria",          label: "Terraria",              defaultName: "terraria",    protocol: "tcp", port: 7777,  hint: "" },
  { id: "rtsp",              label: "RTSP kamera",           defaultName: "kamera",      protocol: "tcp", port: 554,   showAdvanced: true, hint: "Hikvision / Dahua / Reolink — LAN target kiriting" },
  { id: "hikvision-nvr",     label: "Hikvision NVR (SDK)",   defaultName: "nvr",         protocol: "tcp", port: 8000,  showAdvanced: true, hint: "Hikvision NVR — SDK / iVMS-4500 ishlatadigan port" },
  { id: "http-web",          label: "Web UI (HTTP)",         defaultName: "web",         protocol: "tcp", port: 80,    showAdvanced: true, hint: "NVR, smart home, router admin" },
  { id: "https-web",         label: "Web UI (HTTPS)",        defaultName: "web",         protocol: "tcp", port: 443,   showAdvanced: true, hint: "" },
  { id: "ssh",               label: "SSH",                   defaultName: "ssh",         protocol: "tcp", port: 22,    hint: "Masofadan terminal — kuchli kalit shart" },
  { id: "vite",              label: "Vite dev server",       defaultName: "vite",        protocol: "tcp", port: 5173,  hint: "" },
];

export function ServicesPanel({ localServices, peers, refreshLocalServices }: Props) {
  const [name, setName] = useState("");
  const [port, setPort] = useState<number | "">("");
  const [proto, setProto] = useState<"tcp" | "udp" | "both">("tcp");
  const [target, setTarget] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showPresets, setShowPresets] = useState(false);
  const [error, setError] = useState("");

  const applyPreset = (p: Preset) => {
    setName(p.defaultName);
    setPort(p.port);
    setProto(p.protocol);
    if (p.showAdvanced) {
      setShowAdvanced(true);
    } else {
      setTarget("");
    }
    setShowPresets(false);
  };
  const [dialing, setDialing] = useState<{ peerId: string; port: number; protocol: string } | null>(null);
  const [dialedAddrs, setDialedAddrs] = useState<Record<string, string>>({});
  const [detected, setDetected] = useState<LocalListener[]>([]);
  const [scanning, setScanning] = useState(false);
  const [lanDevices, setLanDevices] = useState<LANDiscovery[]>([]);
  const [lanScanning, setLanScanning] = useState(false);
  const [lanScanned, setLanScanned] = useState(false);
  const [lanProgress, setLanProgress] = useState<{ done: number; total: number; hits: number; current: string } | null>(null);
  // Guided "LAN qurilma" form. Lives next to the auto-scan list,
  // collects everything the host-side forwarder needs to point a
  // mesh port at a LAN-attached IP camera / NVR / printer / game
  // server / dev server.
  const [manualName, setManualName] = useState("");
  const [manualIP, setManualIP] = useState("");
  const [manualPort, setManualPort] = useState<number | "">("");
  const [manualProto, setManualProto] = useState<"tcp" | "udp" | "both">("tcp");
  // Optional mesh-side port. Empty → mirror the LAN port (most users
  // want 127.0.0.1:554 ↔ camera:554). Non-empty lets users remap, e.g.
  // expose a camera's :554 as :8554 so it doesn't collide with their
  // own local :554 listener.
  const [manualMeshPort, setManualMeshPort] = useState<number | "">("");
  const [showManual, setShowManual] = useState(false);

  useEffect(() => {
    return subscribe<{ done: number; total: number; hits: number; current: string }>(
      "lanscan:progress",
      (p) => setLanProgress(p),
    );
  }, []);

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

  const confirmRisk = async (target: string, protocol: "tcp" | "udp", port: number): Promise<boolean> => {
    let risk: RiskAssessment;
    try {
      risk = await app.AssessExposeRisk(target, protocol, port);
    } catch {
      return true; // assessment unavailable — don't block
    }
    if (risk.level === "safe") return true;
    const prefix = risk.level === "danger" ? "⚠️ XAVFLI" : "⚡ Diqqat";
    return window.confirm(
      `${prefix}: ${risk.reason}\n\n${risk.hint}\n\nHar holda davom etasizmi?`
    );
  };

  // parseSmartName detects when the user typed something like
  // "192.168.1.100:554" or "kamera 192.168.1.100:554" into the name
  // field and pulls out the IP, port, and (optional) human label.
  // Lets people share a LAN device with one input instead of three.
  const parseSmartName = (
    raw: string,
  ): { name: string; ip: string; port: number } | null => {
    const trimmed = raw.trim();
    if (!trimmed) return null;
    const match = trimmed.match(/^(?:([\w.\-]+)\s+)?(\d{1,3}(?:\.\d{1,3}){3}):(\d{1,5})$/);
    if (!match) return null;
    const [, label, ip, portStr] = match;
    const portNum = Number(portStr);
    if (portNum < 1 || portNum > 65535) return null;
    const cleanLabel =
      label || ip.split(".").pop()! ; // last octet as default label
    return { name: cleanLabel, ip, port: portNum };
  };

  const submitExpose = async () => {
    setError("");
    // Smart-input: user typed "192.168.1.100:554" (optionally with a
    // leading label) into the name field and didn't bother filling
    // anything else. Promote that to a full expose with target =
    // ip:port, port = port, name = label or last octet.
    const smart = parseSmartName(name);
    let effectivePort = port;
    let effectiveTarget = target.trim();
    let effectiveName = name;
    if (smart && (typeof port !== "number" || port < 1)) {
      effectivePort = smart.port;
      effectiveTarget = `${smart.ip}:${smart.port}`;
      effectiveName = smart.name;
    }
    if (typeof effectivePort !== "number" || effectivePort < 1 || effectivePort > 65535) {
      setError("Port 1–65535 oralig'ida bo'lishi kerak");
      return;
    }
    const trimmedTarget = effectiveTarget;
    if (trimmedTarget && !/^[\w.\-]+:\d{1,5}$/.test(trimmedTarget)) {
      setError("Target host:port shaklida bo'lishi kerak (masalan 192.168.1.100:554)");
      return;
    }
    // "both" expands to two exposes — same name/port/target on TCP
    // and UDP. Useful for Source / Steam game servers (CS2 27015 needs
    // both; the TCP side is RCON / query). Risk check runs once
    // against TCP since the target/port are identical.
    const protocols: ("tcp" | "udp")[] =
      proto === "both" ? ["tcp", "udp"] : [proto];
    if (!(await confirmRisk(trimmedTarget, protocols[0], effectivePort))) {
      return;
    }
    try {
      const baseName = (smart ? smart.name : effectiveName) ||
        `${proto === "both" ? "both" : proto}:${effectivePort}`;
      for (const p of protocols) {
        await app.ExposeService(baseName, p, effectivePort, trimmedTarget);
      }
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

  const togglePause = async (s: ServiceView) => {
    try {
      await app.SetExposeEnabled(s.port, s.protocol as "tcp" | "udp", !!s.paused);
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  // retargetService is the inline-edit path for an existing exposed
  // row. ExposeService is already idempotent on (port, protocol) —
  // calling it with the same key but a different target updates the
  // forwarder's target table and the persisted row in one shot. This
  // lets users fix a 127.0.0.1:80 mistarget to 192.168.1.100:80
  // without losing the row's name + history.
  const retargetService = async (s: ServiceView, newTarget: string) => {
    setError("");
    const trimmed = newTarget.trim();
    if (trimmed && !/^[\w.\-]+:\d{1,5}$/.test(trimmed)) {
      setError("Target host:port shaklida bo'lishi kerak (masalan 192.168.1.100:554)");
      return false;
    }
    if (trimmed) {
      const proto = s.protocol === "udp" ? "udp" : "tcp";
      if (!(await confirmRisk(trimmed, proto, s.port))) return false;
    }
    try {
      const proto = (s.protocol === "udp" ? "udp" : "tcp") as "tcp" | "udp";
      await app.ExposeService(s.name, proto, s.port, trimmed);
      await refreshLocalServices();
      return true;
    } catch (e: any) {
      setError(e?.message || String(e));
      return false;
    }
  };

  const scanLAN = async () => {
    setLanScanning(true);
    setLanProgress(null);
    try {
      const list = await app.ScanLAN();
      setLanDevices(list);
      setLanScanned(true);
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setLanScanning(false);
      setLanProgress(null);
    }
  };

  // submitManualLAN handles "I have a LAN device's IP+port and I want
  // to share it with the room". Supports a name, TCP / UDP / both, and
  // an optional mesh-side port distinct from the LAN port (the "Och
  // bilan ulanmoqchi bo'lgan port" mechanism — useful when the LAN
  // port collides with something the user has running locally).
  //
  // The pasted IP can be plain ("192.168.1.100") or "host:port" — we
  // split it so the user doesn't have to fight two fields when they
  // copy-paste from elsewhere.
  const submitManualLAN = async () => {
    setError("");
    let ipRaw = manualIP.trim();
    let pastedPort: number | null = null;
    // Smart-paste: if user typed "192.168.1.100:554" into the IP field,
    // split into IP + port automatically.
    const m = ipRaw.match(/^(\d{1,3}(?:\.\d{1,3}){3}):(\d{1,5})$/);
    if (m) {
      ipRaw = m[1];
      pastedPort = Number(m[2]);
    }
    const ip = ipRaw;
    if (!/^\d{1,3}(?:\.\d{1,3}){3}$/.test(ip)) {
      setError("IP 192.168.1.100 ko'rinishida bo'lishi kerak");
      return;
    }
    let lanPort: number;
    if (pastedPort !== null) {
      lanPort = pastedPort;
    } else if (typeof manualPort === "number") {
      lanPort = manualPort;
    } else {
      setError("Port 1–65535 oralig'ida bo'lishi kerak");
      return;
    }
    if (lanPort < 1 || lanPort > 65535) {
      setError("Port 1–65535 oralig'ida bo'lishi kerak");
      return;
    }
    // Mesh-side port: defaults to the LAN port. The Go forwarder
    // accepts any port for the mesh-side announcement; the target
    // string carries the LAN destination separately.
    const meshPort =
      typeof manualMeshPort === "number" && manualMeshPort >= 1 && manualMeshPort <= 65535
        ? manualMeshPort
        : lanPort;

    const target = `${ip}:${lanPort}`;
    const protocols: ("tcp" | "udp")[] =
      manualProto === "both" ? ["tcp", "udp"] : [manualProto];
    // Risk check on the first protocol — same target/port combo, the
    // assessor's verdict is identical for tcp/udp.
    if (!(await confirmRisk(target, protocols[0], lanPort))) return;
    try {
      const niceName =
        manualName.trim() || `host-${ip.split(".").pop()}`;
      for (const p of protocols) {
        await app.ExposeService(niceName, p, meshPort, target);
      }
      setManualName("");
      setManualIP("");
      setManualPort("");
      setManualMeshPort("");
      setManualProto("tcp");
      setShowManual(false);
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  const exposeLANDevice = async (d: LANDiscovery) => {
    setError("");
    const target = `${d.ip}:${d.port}`;
    const niceName =
      (d.hostname && d.hostname.split(".")[0]) ||
      `${d.service}-${d.ip.split(".").pop()}`;
    if (!(await confirmRisk(target, "tcp", d.port))) {
      return;
    }
    try {
      await app.ExposeService(niceName, "tcp", d.port, target);
      await refreshLocalServices();
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  // exposeAllSafe is the bulk version of exposeLANDevice. It walks the
  // whole scan result, skips anything already exposed, and silently
  // drops any target the risk assessor flagged as warn or danger —
  // routers, DB ports, RDP — so a single click on a public Wi-Fi
  // can't accidentally hand admin access to a friend group. Reports
  // a one-line summary at the end.
  const exposeAllSafe = async () => {
    setError("");
    if (lanDevices.length === 0) return;
    const candidates = lanDevices.filter(
      (d) => !localServices.some((s) => s.port === d.port && s.protocol === d.protocol),
    );
    if (candidates.length === 0) return;
    if (!window.confirm(
      `${candidates.length} ta qurilmani avtomatik ochishga harakat qilamiz.\n\n` +
      `XAVFLI portlar (router admin, DB, RDP) avtomatik o'tkazib yuboriladi — ` +
      `ularni qo'lda alohida tasdiqlashingiz mumkin.\n\nDavom etamizmi?`
    )) {
      return;
    }
    let opened = 0;
    let skipped = 0;
    for (const d of candidates) {
      const target = `${d.ip}:${d.port}`;
      let risk;
      try {
        risk = await app.AssessExposeRisk(target, "tcp", d.port);
      } catch {
        risk = { level: "safe" as const, reason: "", hint: "" };
      }
      if (risk.level !== "safe") {
        skipped++;
        continue;
      }
      const niceName =
        (d.hostname && d.hostname.split(".")[0]) ||
        `${d.service}-${d.ip.split(".").pop()}`;
      try {
        await app.ExposeService(niceName, "tcp", d.port, target);
        opened++;
      } catch {
        skipped++;
      }
    }
    await refreshLocalServices();
    window.alert(`${opened} ta servis ochildi.${skipped > 0 ? ` ${skipped} ta xavfli/xato bo'lgani uchun o'tkazib yuborildi (qo'lda Och bosib alohida ko'rib chiqing).` : ""}`);
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
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-violet-400" strokeWidth={2} />
            <h3 className="font-semibold text-sm">Mening servislarim</h3>
          </div>
          <div className="relative">
            <button
              onClick={() => setShowPresets((v) => !v)}
              className="text-[11px] text-violet-300 hover:text-violet-200 flex items-center gap-1"
            >
              <Sparkles className="w-3 h-3" strokeWidth={2.5} />
              Tezkor
            </button>
            {showPresets && (
              <>
                <div
                  className="fixed inset-0 z-30"
                  onClick={() => setShowPresets(false)}
                />
                <div className="absolute right-0 top-full mt-1 z-40 panel rounded-card p-1 min-w-[230px] max-h-[300px] overflow-y-auto shadow-xl">
                  {presets.map((p) => (
                    <button
                      key={p.id}
                      onClick={() => applyPreset(p)}
                      title={p.hint}
                      className="w-full text-left px-2.5 py-1.5 rounded hover:bg-white/[0.06] flex items-center gap-2 text-xs"
                    >
                      <span className="flex-1 truncate">{p.label}</span>
                      <span className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
                        p.protocol === "udp"
                          ? "bg-cyan-400/10 text-cyan-300"
                          : "bg-violet-400/10 text-violet-300"
                      }`}>
                        {p.protocol.toUpperCase()}
                      </span>
                      <span className="font-mono text-[10px] text-zinc-500 shrink-0">:{p.port}</span>
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>
        </div>
        <p className="text-xs text-zinc-500 mb-3">
          Lokal portni mesh ga oching — boshqa peerlar to'g'ridan-to'g'ri ulana oladi.
        </p>
        <div className="flex flex-wrap gap-2 items-stretch">
          <input
            type="text"
            placeholder="Nom yoki IP:port (192.168.1.100:554)"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input-base text-sm flex-1 min-w-[140px]"
            title="Oddiy nom ('kamera') yozsangiz pastdagi port bilan ekspoz qilinadi. To'g'ridan-to'g'ri 'IP:port' yoki 'nom IP:port' yozsangiz, port va target avtomatik to'ldiriladi."
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
              <button
                type="button"
                onClick={() => setProto("both")}
                className={`px-2 border-l border-white/10 ${proto === "both" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                title="Bir vaqtda TCP+UDP — Steam game serverlari (CS2 27015 game UDP + RCON TCP)"
              >
                Ikkalasi
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
                <ExposedRow
                  key={`${s.protocol}:${s.port}`}
                  s={s}
                  onTogglePause={() => togglePause(s)}
                  onRemove={() => removeExposed(s.port)}
                  onRetarget={(t) => retargetService(s, t)}
                />
              ))}
              {/* Inline diagnostic for any 'down' rows. Lives outside
                  the row itself so it can wrap at full width. The
                  'localhost-but-down' case gets a louder hint
                  pointing at the LAN-target affordance, because
                  this is by far the most common misconfiguration:
                  user pasted IP into the name, hit Och, and ended
                  up exposing 127.0.0.1:port instead of forwarding
                  to a LAN device. The pencil edit on the row fixes
                  it in one click. */}
              {localServices.some((s) => s.health === "down" && !s.paused) && (
                <div className="mt-1.5 panel rounded-input px-3 py-2 text-[10px] text-rose-300 bg-rose-500/[0.04] border-rose-500/20 leading-relaxed space-y-1">
                  {localServices
                    .filter((s) => s.health === "down" && !s.paused)
                    .map((s) => {
                      const isLocalhost = !s.target || s.target === `127.0.0.1:${s.port}`;
                      return (
                        <div key={`err-${s.protocol}:${s.port}`}>
                          <span className="font-mono text-rose-200">
                            {s.protocol.toUpperCase()}:{s.port}
                          </span>{" "}
                          <span className="text-rose-300/80">
                            {s.healthError ||
                              `target ${s.target || "—"} javob bermoqda emas`}
                          </span>
                          {isLocalhost && (
                            <div className="text-amber-300/90 mt-1 ml-1">
                              💡 Bu lokal kompyuterga (127.0.0.1) ishora qilyapti.
                              Tarmoqdagi qurilma kerak bo'lsa, ✏️ tugmasini bosib{" "}
                              target ni <code className="font-mono">192.168.x.x:{s.port}</code>{" "}
                              ga o'zgartiring.
                            </div>
                          )}
                        </div>
                      );
                    })}
                </div>
              )}
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
          <div className="flex items-center justify-between mb-2.5">
            <h3 className="font-semibold text-sm flex items-center gap-2">
              <Radar className="w-4 h-4 text-cyan-400" strokeWidth={2} />
              Tarmoqdagi qurilmalar
            </h3>
            <div className="flex items-center gap-3">
              {lanScanned && lanDevices.length > 0 && (
                <button
                  onClick={exposeAllSafe}
                  className="text-[11px] text-violet-300 hover:text-violet-200"
                  title="Hamma topilgan qurilmalarni avtomatik ochish (xavfli portlar — router admin, DB — o'tkazib yuboriladi)"
                >
                  Hammasini och
                </button>
              )}
              <button
                onClick={scanLAN}
                disabled={lanScanning}
                className="text-[11px] text-zinc-400 hover:text-white disabled:opacity-50"
              >
                {lanScanning ? "Skanerlanmoqda…" : lanScanned ? "Qayta skanerlash" : "Skanerlash"}
              </button>
            </div>
          </div>
          {!lanScanned && !lanScanning && (
            <div className="text-xs text-zinc-500 panel rounded-input px-3 py-3 text-center">
              "Skanerlash"ni bosing — RTSP kameralar, NVR, printerlar va boshqa LAN qurilmalari topiladi (~10s).
            </div>
          )}
          {lanScanning && lanProgress && (
            <div className="panel rounded-input px-3 py-2 mb-2 text-xs">
              <div className="flex items-center justify-between mb-1">
                <span className="text-zinc-300 truncate">
                  {lanProgress.current
                    ? `${lanProgress.current}…`
                    : "Boshlanmoqda…"}
                </span>
                <span className="text-zinc-500 font-mono shrink-0">
                  {lanProgress.done}/{lanProgress.total} · {lanProgress.hits} topildi
                </span>
              </div>
              <div className="h-1 bg-white/[0.04] rounded overflow-hidden">
                <div
                  className="h-full bg-gradient-to-r from-violet-500 to-cyan-400 transition-[width] duration-200"
                  style={{
                    width: `${Math.min(100, (lanProgress.done / Math.max(1, lanProgress.total)) * 100)}%`,
                  }}
                />
              </div>
            </div>
          )}
          {lanScanned && lanDevices.length === 0 && !lanScanning && (
            <div className="text-xs text-zinc-500 panel rounded-input px-3 py-3 text-center">
              Tarmoqda boshqa qurilma topilmadi.
            </div>
          )}
          <LANResults
            devices={lanDevices}
            localServices={localServices}
            onExposeOne={exposeLANDevice}
          />

          <div className="mt-2">
            <button
              type="button"
              onClick={() => setShowManual((v) => !v)}
              className="text-[11px] text-zinc-400 hover:text-zinc-200"
            >
              {showManual ? "− Qo'lda qo'shish" : "+ Qurilma topilmadimi? Qo'lda IP kiriting"}
            </button>
            {showManual && (
              <div className="mt-2 panel rounded-input p-3 space-y-2">
                {/* Row 1: name + protocol — set what kind of service
                    this is and how friends will find it in the list.
                    Name defaults to "host-<last-octet>" if left blank. */}
                <div className="flex gap-2 items-stretch">
                  <input
                    type="text"
                    placeholder="Nom (kamera, nvr, printer…)"
                    value={manualName}
                    onChange={(e) => setManualName(e.target.value)}
                    className="input-base text-sm flex-1"
                  />
                  <div className="flex rounded overflow-hidden border border-white/10 text-[11px] font-mono shrink-0">
                    <button
                      type="button"
                      onClick={() => setManualProto("tcp")}
                      className={`px-2 ${manualProto === "tcp" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                      title="HTTP, RTSP-TCP, SSH, Minecraft Java"
                    >
                      TCP
                    </button>
                    <button
                      type="button"
                      onClick={() => setManualProto("udp")}
                      className={`px-2 ${manualProto === "udp" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                      title="RTP video, CS2, Bedrock — UDP-based services"
                    >
                      UDP
                    </button>
                    <button
                      type="button"
                      onClick={() => setManualProto("both")}
                      className={`px-2 border-l border-white/10 ${manualProto === "both" ? "bg-violet-500/30 text-white" : "text-zinc-400 hover:bg-white/[0.04]"}`}
                      title="TCP + UDP (Steam game servers)"
                    >
                      Ikkalasi
                    </button>
                  </div>
                </div>
                {/* Row 2: LAN IP + port. The IP field smart-pastes
                    "host:port" so users dropping a copied address
                    don't need to split it manually. */}
                <div className="flex gap-2 items-stretch">
                  <input
                    type="text"
                    placeholder="LAN IP (192.168.1.100)"
                    value={manualIP}
                    onChange={(e) => setManualIP(e.target.value)}
                    className="input-base text-sm flex-1 min-w-[120px] font-mono"
                    title="Tarmoqdagi qurilmaning IP'si. 'IP:port' ko'rinishida yozsangiz, port avtomatik ajratiladi."
                  />
                  <input
                    type="number"
                    placeholder="LAN port"
                    value={manualPort}
                    onChange={(e) => setManualPort(e.target.value === "" ? "" : Number(e.target.value))}
                    className="input-base w-24 text-sm font-mono"
                    title="Qurilmaning portsi (masalan: 554 RTSP, 80 HTTP, 631 IPP printer)"
                  />
                  <button
                    onClick={submitManualLAN}
                    className="btn-primary rounded-btn px-3 flex items-center gap-1 text-sm shrink-0"
                  >
                    <Plus className="w-4 h-4" strokeWidth={2} />
                    Och
                  </button>
                </div>
                {/* Row 3 (optional): different mesh-side port. Default
                    is "same as LAN port" — the user only fills this
                    when they have a local listener on the same port
                    and need to remap. */}
                <input
                  type="number"
                  placeholder="Mesh-port (ixtiyoriy — bo'sh = LAN port)"
                  value={manualMeshPort}
                  onChange={(e) => setManualMeshPort(e.target.value === "" ? "" : Number(e.target.value))}
                  className="input-base w-full text-sm font-mono"
                  title="Mesh tomonida boshqa peerlar ulanadigan port. Standart — LAN portga teng. Lokalda shu port band bo'lsa boshqa raqam tanlang (masalan, kamera LAN'da 554, mesh'da 8554)."
                />
                <p className="text-[10px] text-zinc-500 leading-relaxed">
                  Tarmoqdagi boshqa qurilmaning portini xonadagi do'stlarga ulashing —
                  IP kamera, NVR, printer, dev server. Mesh-port bo'sh qoldirilsa LAN port bilan bir xil bo'ladi.
                </p>
              </div>
            )}
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
// LANResults groups scan hits by IP. With 30+ probed ports per host
// and several hosts, the flat-list rendering of v0.4.0 turned into a
// 100+-row scroll on busy networks; the grouped view stays readable.
// Default-collapsed when there are 4+ hosts so big networks don't
// force a paint of every row up-front.
function LANResults({
  devices,
  localServices,
  onExposeOne,
}: {
  devices: LANDiscovery[];
  localServices: ServiceView[];
  onExposeOne: (d: LANDiscovery) => void;
}) {
  const groups = React.useMemo(() => {
    const m = new Map<string, LANDiscovery[]>();
    for (const d of devices) {
      const arr = m.get(d.ip) || [];
      arr.push(d);
      m.set(d.ip, arr);
    }
    return [...m.entries()].map(([ip, items]) => ({
      ip,
      hostname: items.find((x) => x.hostname)?.hostname || "",
      items,
    }));
  }, [devices]);

  const startCollapsed = groups.length >= 4;
  const [open, setOpen] = useState<Record<string, boolean>>({});
  // First render after groups changes: seed open-state with the
  // collapse rule. We use a ref-equivalent trick — re-seeding is safe
  // because only first-time-true is meaningful.
  useEffect(() => {
    if (groups.length === 0) return;
    setOpen((prev) => {
      const next = { ...prev };
      for (const g of groups) {
        if (next[g.ip] === undefined) {
          next[g.ip] = !startCollapsed;
        }
      }
      return next;
    });
  }, [groups, startCollapsed]);

  if (groups.length === 0) return null;

  return (
    <div className="space-y-1.5">
      {groups.map((g) => {
        const isOpen = !!open[g.ip];
        const exposedCount = g.items.filter((d) =>
          localServices.some((s) => s.port === d.port && s.protocol === d.protocol),
        ).length;
        return (
          <div key={g.ip} className="panel rounded-input">
            <button
              onClick={() => setOpen((p) => ({ ...p, [g.ip]: !p[g.ip] }))}
              className="w-full px-3 py-2 flex items-center gap-2 text-left text-sm hover:bg-white/[0.03]"
            >
              <Globe className="w-3.5 h-3.5 text-cyan-400 shrink-0" strokeWidth={2} />
              <div className="min-w-0 flex-1 leading-tight">
                <div className="truncate text-xs font-medium">
                  {g.hostname || g.ip}
                </div>
                {g.hostname && (
                  <div className="font-mono text-[10px] text-zinc-500">{g.ip}</div>
                )}
              </div>
              <span className="text-[10px] text-zinc-500 shrink-0">
                {g.items.length} port{g.items.length === 1 ? "" : "lar"}
                {exposedCount > 0 && (
                  <span className="text-emerald-400/80 ml-1">· {exposedCount} ochilgan</span>
                )}
              </span>
              <span className="text-zinc-500 shrink-0 text-xs">
                {isOpen ? "▾" : "▸"}
              </span>
            </button>
            {isOpen && (
              <div className="border-t border-white/5 px-3 py-2 space-y-1.5">
                {g.items.map((d) => {
                  const exposed = localServices.some(
                    (s) => s.port === d.port && s.protocol === d.protocol,
                  );
                  return (
                    <div
                      key={`${d.protocol}:${d.port}`}
                      className="flex items-center gap-2 text-xs"
                    >
                      <span className="font-mono text-zinc-400 shrink-0">
                        :{d.port}
                      </span>
                      <span className="text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 bg-cyan-400/10 text-cyan-300 uppercase">
                        {d.service}
                      </span>
                      <span className="flex-1 min-w-0" />
                      {exposed ? (
                        <span className="text-[11px] text-emerald-400 font-medium shrink-0">ochilgan ✓</span>
                      ) : (
                        <button
                          onClick={() => onExposeOne(d)}
                          className="px-2 py-1 rounded text-[11px] font-medium bg-violet-500/15 text-violet-300 hover:bg-violet-500/25 shrink-0 whitespace-nowrap"
                        >
                          Och
                        </button>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// ExposedRow renders one row in 'Mening servislarim'. Encapsulates
// per-row state (the inline target editor) so the parent doesn't
// have to track which row is being edited and toggling one row
// doesn't blow away the input in another.
//
// The pencil icon swaps the row into an edit mode where the user can
// rewrite the LAN target — fixes the common "I forgot to fill in
// the LAN IP and now my row points at 127.0.0.1:80 by accident"
// problem without losing the row's name, history, or label.
function ExposedRow({
  s,
  onTogglePause,
  onRemove,
  onRetarget,
}: {
  s: ServiceView;
  onTogglePause: () => void;
  onRemove: () => void;
  onRetarget: (target: string) => Promise<boolean>;
}) {
  const paused = !!s.paused;
  const health = s.health || "unknown";
  const isLocalhost = !s.target || s.target === `127.0.0.1:${s.port}`;
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(isLocalhost ? "" : s.target || "");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!editing) {
      setDraft(isLocalhost ? "" : s.target || "");
    }
  }, [s.target, isLocalhost, editing]);

  const submit = async () => {
    setSaving(true);
    try {
      const ok = await onRetarget(draft);
      if (ok) setEditing(false);
    } finally {
      setSaving(false);
    }
  };

  const dotClass = paused
    ? "bg-amber-400 shadow-[0_0_4px_#fbbf24]"
    : health === "ok"
    ? "bg-emerald-400 shadow-[0_0_4px_#34d399]"
    : health === "down"
    ? "bg-rose-400 shadow-[0_0_4px_#fb7185]"
    : "bg-zinc-500";
  const dotLabel = paused
    ? "Pauza qilingan — peer'lar ulanolmaydi"
    : health === "ok"
    ? `Target ${s.target || "localhost"} javob bermoqda`
    : health === "down"
    ? `Ulanish muvaffaqiyatsiz: ${s.healthError || s.target || "?"}\n\nNimalarni tekshirish kerak:\n• Qurilma yoqilganmi?\n• IP manzili to'g'rimi?\n• Shu portda haqiqatan ham servis ishlayaptimi?`
    : s.protocol === "udp"
    ? "UDP holatini avtomatik tekshirib bo'lmaydi — peer ulanganda aniqlanadi"
    : "Holati hali tekshirilmagan";

  if (editing) {
    return (
      <motion.div
        layout
        className="panel rounded-input px-3 py-2 space-y-2 text-sm border-violet-500/30"
      >
        <div className="flex items-center gap-2">
          <Pencil className="w-3.5 h-3.5 text-violet-300 shrink-0" strokeWidth={2} />
          <span className="font-medium truncate flex-1">{s.name}</span>
          <span
            className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
              s.protocol === "udp"
                ? "bg-cyan-400/10 text-cyan-300"
                : "bg-violet-400/10 text-violet-300"
            }`}
          >
            {s.protocol.toUpperCase()}
          </span>
          <span className="font-mono text-xs text-zinc-500 shrink-0">:{s.port}</span>
        </div>
        <div className="flex gap-2 items-stretch">
          <input
            type="text"
            autoFocus
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") submit();
              if (e.key === "Escape") setEditing(false);
            }}
            placeholder={`LAN target (192.168.1.100:${s.port}) — bo'sh = localhost`}
            className="input-base text-sm flex-1 font-mono"
            disabled={saving}
          />
          <button
            onClick={submit}
            disabled={saving}
            title="Saqlash"
            className="px-2 rounded text-emerald-300 hover:bg-emerald-500/15 disabled:opacity-50"
          >
            <Check className="w-4 h-4" strokeWidth={2} />
          </button>
          <button
            onClick={() => setEditing(false)}
            disabled={saving}
            title="Bekor qilish"
            className="px-2 rounded text-zinc-400 hover:bg-white/[0.07] disabled:opacity-50"
          >
            <X className="w-4 h-4" strokeWidth={2} />
          </button>
        </div>
        <p className="text-[10px] text-zinc-500 leading-relaxed">
          Maqsad: tarmoqdagi qurilmaga forward qilish. Misol: kamera{" "}
          <code className="font-mono text-zinc-400">192.168.1.100:554</code>,
          NVR <code className="font-mono text-zinc-400">192.168.1.50:8000</code>.
          Bo'sh qoldirsangiz, lokal kompyuterdagi {s.port}-port ishlatiladi.
        </p>
      </motion.div>
    );
  }

  return (
    <motion.div
      layout
      initial={{ opacity: 0, height: 0 }}
      animate={{ opacity: 1, height: "auto" }}
      exit={{ opacity: 0, height: 0 }}
      className={`panel rounded-input px-3 py-2 flex items-center gap-2 text-sm ${
        paused ? "opacity-60" : ""
      }`}
      title={s.target ? `→ ${s.target}` : undefined}
    >
      <span
        title={dotLabel}
        className={`w-2 h-2 rounded-full shrink-0 ${dotClass}`}
      />
      <Globe
        className={`w-3.5 h-3.5 shrink-0 ${
          paused ? "text-amber-400" : "text-emerald-400"
        }`}
        strokeWidth={2}
      />
      <div className="min-w-0 flex-1 leading-tight">
        <div className="truncate font-medium flex items-center gap-1.5">
          {s.name}
          {!isLocalhost && (
            <span
              className="text-[9px] uppercase tracking-wider font-mono px-1 py-0.5 rounded bg-cyan-500/15 text-cyan-300 shrink-0"
              title={`LAN qurilma — ${s.target}`}
            >
              LAN
            </span>
          )}
        </div>
        {!isLocalhost && (
          <div
            className="font-mono text-[10px] text-cyan-300/70 truncate"
            title={s.target}
          >
            → {s.target}
          </div>
        )}
        {/* Show the target line for localhost rows too — the user
            looking at a 127.0.0.1:80 entry that's down should see
            the target right there so they know it's LOCAL, not LAN. */}
        {isLocalhost && (
          <div className="font-mono text-[10px] text-zinc-500 truncate">
            → 127.0.0.1:{s.port}{" "}
            <span className="text-zinc-600">(lokal)</span>
          </div>
        )}
      </div>
      <span
        className={`text-[10px] font-mono px-1.5 py-0.5 rounded shrink-0 ${
          s.protocol === "udp"
            ? "bg-cyan-400/10 text-cyan-300"
            : "bg-violet-400/10 text-violet-300"
        }`}
      >
        {s.protocol.toUpperCase()}
      </span>
      <span className="font-mono text-xs text-zinc-500 shrink-0">:{s.port}</span>
      <button
        onClick={() => setEditing(true)}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-violet-300 shrink-0"
        title="Target o'zgartirish (LAN qurilmaga forward qilish)"
      >
        <Pencil className="w-3.5 h-3.5" strokeWidth={2} />
      </button>
      <button
        onClick={onTogglePause}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-amber-300 shrink-0"
        title={paused ? "Davom ettirish" : "Vaqtincha to'xtatish"}
      >
        {paused ? (
          <Play className="w-3.5 h-3.5" strokeWidth={2} />
        ) : (
          <Pause className="w-3.5 h-3.5" strokeWidth={2} />
        )}
      </button>
      <button
        onClick={onRemove}
        className="p-1 rounded hover:bg-white/5 text-zinc-400 hover:text-rose-400 shrink-0"
        title="Olib tashlash"
      >
        <Trash2 className="w-3.5 h-3.5" strokeWidth={2} />
      </button>
    </motion.div>
  );
}

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
