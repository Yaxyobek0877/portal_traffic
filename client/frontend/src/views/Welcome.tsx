import React, { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { Sparkles, LogIn, Settings, History as HistoryIcon } from "lucide-react";
import { Logo } from "../components/Logo";
import { app } from "../lib/wails";
import { usePortalStore } from "../stores/portalStore";

type Mode = "idle" | "create" | "join";

export function Welcome() {
  const nickname = usePortalStore((s) => s.nickname);
  const setNickname = usePortalStore((s) => s.setNickname);
  const setPortal = usePortalStore((s) => s.setPortal);
  const setScreen = usePortalStore((s) => s.setScreen);
  const setSignalingUrl = usePortalStore((s) => s.setSignalingUrl);
  const signalingUrl = usePortalStore((s) => s.signalingUrl);
  const history = usePortalStore((s) => s.history);
  const setHistory = usePortalStore((s) => s.setHistory);

  const [mode, setMode] = useState<Mode>("idle");
  const [portalId, setPortalId] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  // generation counter so we can ignore a late-arriving result from a
  // connect attempt the user already cancelled. Without this, clicking
  // "Orqaga" while the request is in flight just lets the eventual
  // success drag the user back to the portal view.
  const genRef = useRef(0);

  useEffect(() => {
    app.SignalingURL().then(setSignalingUrl);
    app.RecentPortals(5).then(setHistory);
  }, [setSignalingUrl, setHistory]);

  const reuse = (id: string, c: string) => {
    setMode("join");
    setPortalId(id);
    setCode(c);
  };

  const cancel = async () => {
    genRef.current++;
    setBusy(false);
    setMode("idle");
    setError("");
    // Tear down whatever half-formed mesh the backend is currently
    // assembling. Without this, a stuck join can hold sockets open
    // until the next connect attempt collides.
    try {
      await app.Leave();
    } catch {}
  };

  const submit = async () => {
    setError("");
    if (!nickname.trim()) {
      setError("Avval taxallus yozing");
      return;
    }
    if (mode === "join" && (!portalId.trim() || !code.trim())) {
      setError("ID va kod ikkalasi kerak");
      return;
    }
    const myGen = ++genRef.current;
    setBusy(true);
    try {
      const p =
        mode === "create"
          ? await app.CreatePortal(nickname.trim(), false)
          : await app.JoinPortal(nickname.trim(), portalId.trim(), code.trim());
      if (myGen !== genRef.current) {
        // The user cancelled while we were waiting. Drop the result
        // silently — they're back on the welcome screen already.
        return;
      }
      setPortal(p);
      setScreen("portal");
    } catch (e: any) {
      if (myGen !== genRef.current) return;
      setError(e?.message || String(e));
    } finally {
      if (myGen === genRef.current) setBusy(false);
    }
  };

  return (
    <div className="h-full flex flex-col">
      <div className="draggable titlebar-pad flex justify-end items-center px-3" style={{ height: 68 }}>
        <button
          className="no-drag p-1.5 rounded-md text-zinc-500 hover:text-white hover:bg-white/5"
          title="Sozlamalar"
          onClick={() => setScreen("settings")}
        >
          <Settings className="w-4 h-4" strokeWidth={2} />
        </button>
      </div>

      <div className="flex-1 flex items-center justify-center px-6 -mt-6">
        <div className="w-full max-w-[440px] text-center">
          <motion.div
            initial={{ opacity: 0, scale: 0.85 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ type: "spring", stiffness: 200, damping: 22 }}
          >
            <Logo size={170} />
          </motion.div>
          <motion.h1
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 }}
            className="text-4xl font-extrabold tracking-tight mt-2"
          >
            Portal
          </motion.h1>
          <motion.p
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
            className="text-sm text-zinc-400 mt-2"
          >
            To'g'ridan-to'g'ri ulanish.<br /> Orada hech qanday server yo'q.
          </motion.p>

          <div className="mt-8 space-y-3 text-left">
            <div>
              <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                Taxallus
              </label>
              <input
                type="text"
                placeholder="alice"
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                className="input-base w-full mt-1.5"
                maxLength={24}
                autoFocus
              />
            </div>

            {mode === "join" && (
              <motion.div
                initial={{ opacity: 0, y: -6 }}
                animate={{ opacity: 1, y: 0 }}
                className="grid grid-cols-2 gap-2"
              >
                <div>
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    Portal ID
                  </label>
                  <input
                    type="text"
                    placeholder="123456"
                    value={portalId}
                    onChange={(e) => setPortalId(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))}
                    className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                    maxLength={6}
                  />
                </div>
                <div>
                  <label className="text-xs uppercase tracking-widest text-zinc-500 ml-1">
                    Kod
                  </label>
                  <input
                    type="text"
                    placeholder="654321"
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/[^0-9]/g, "").slice(0, 6))}
                    className="input-base w-full mt-1.5 font-mono text-center text-lg tracking-widest"
                    maxLength={6}
                  />
                </div>
              </motion.div>
            )}

            {error && <div className="text-xs text-rose-400">{error}</div>}

            {mode === "idle" ? (
              <div className="grid grid-cols-2 gap-3 pt-3">
                <button
                  onClick={() => setMode("create")}
                  className="btn-primary rounded-btn h-12 font-semibold flex items-center justify-center gap-2"
                >
                  <Sparkles className="w-4 h-4" />
                  Portal yaratish
                </button>
                <button
                  onClick={() => setMode("join")}
                  className="panel rounded-btn h-12 font-semibold flex items-center justify-center gap-2 hover:bg-white/[0.07]"
                >
                  <LogIn className="w-4 h-4" />
                  Qo'shilish
                </button>
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-3 pt-3">
                <button
                  onClick={busy ? cancel : () => setMode("idle")}
                  className="panel rounded-btn h-11 text-sm hover:bg-white/[0.07]"
                >
                  {busy ? "Bekor qilish" : "Orqaga"}
                </button>
                <button
                  onClick={submit}
                  disabled={busy}
                  className="btn-primary rounded-btn h-11 text-sm font-semibold disabled:opacity-50"
                >
                  {busy ? "Ulanmoqda..." : mode === "create" ? "Yaratish" : "Qo'shilish"}
                </button>
              </div>
            )}
          </div>

          {history.length > 0 && mode === "idle" && (
            <div className="mt-8 text-left">
              <div className="flex items-center gap-1.5 text-xs uppercase tracking-widest text-zinc-500 mb-2">
                <HistoryIcon className="w-3 h-3" />
                Yaqindagilar
              </div>
              <div className="space-y-1.5">
                {history.slice(0, 4).map((h) => (
                  <button
                    key={h.id}
                    onClick={() => h.code && reuse(h.portalId, h.code)}
                    disabled={!h.code}
                    className="w-full panel rounded-input px-3 py-2 flex items-center justify-between text-sm hover:bg-white/[0.07] disabled:opacity-50"
                  >
                    <div className="flex items-center gap-2 min-w-0">
                      <span className="font-mono text-violet-300">{h.portalId}</span>
                      <span className="text-zinc-500 truncate">{h.nickname}</span>
                    </div>
                    {h.isOwner && (
                      <span className="text-[10px] uppercase tracking-wider text-amber-400">
                        owner
                      </span>
                    )}
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="text-xs text-zinc-600 mt-8 font-mono truncate">{signalingUrl}</div>
        </div>
      </div>
    </div>
  );
}
