import React, { useEffect, useState } from "react";
import { motion } from "framer-motion";
import { Sparkles, LogIn, Settings } from "lucide-react";
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

  const [mode, setMode] = useState<Mode>("idle");
  const [portalId, setPortalId] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [draftUrl, setDraftUrl] = useState("");

  useEffect(() => {
    app.SignalingURL().then((u) => {
      setSignalingUrl(u);
      setDraftUrl(u);
    });
  }, [setSignalingUrl]);

  const submit = async () => {
    setError("");
    if (!nickname.trim()) {
      setError("Avval taxallus yozing");
      return;
    }
    setBusy(true);
    try {
      let p;
      if (mode === "create") {
        p = await app.CreatePortal(nickname.trim(), false);
      } else {
        if (!portalId.trim() || !code.trim()) {
          setError("ID va kod ikkalasi kerak");
          return;
        }
        p = await app.JoinPortal(nickname.trim(), portalId.trim(), code.trim());
      }
      setPortal(p);
      setScreen("portal");
    } catch (e: any) {
      setError(e?.message || String(e));
    } finally {
      setBusy(false);
    }
  };

  const saveSettings = async () => {
    try {
      await app.SetSignalingURL(draftUrl);
      setSignalingUrl(draftUrl);
      setSettingsOpen(false);
    } catch (e: any) {
      setError(e?.message || String(e));
    }
  };

  return (
    <div className="h-full flex flex-col">
      <div className="draggable h-[40px] flex justify-end items-center px-3">
        <button
          className="no-drag p-1.5 rounded-md text-zinc-500 hover:text-white hover:bg-white/5"
          title="Sozlamalar"
          onClick={() => setSettingsOpen((v) => !v)}
        >
          <Settings className="w-4 h-4" strokeWidth={2} />
        </button>
      </div>

      <div className="flex-1 flex items-center justify-center px-6 -mt-6">
        <div className="w-full max-w-[420px] text-center">
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
                  onClick={() => setMode("idle")}
                  className="panel rounded-btn h-11 text-sm hover:bg-white/[0.07]"
                  disabled={busy}
                >
                  Orqaga
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

          <div className="text-xs text-zinc-600 mt-8 font-mono truncate">
            {signalingUrl}
          </div>
        </div>
      </div>

      {settingsOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-30"
          onClick={() => setSettingsOpen(false)}
        >
          <motion.div
            initial={{ scale: 0.95 }}
            animate={{ scale: 1 }}
            className="panel rounded-card p-6 w-[440px]"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="text-base font-semibold mb-3">Sozlamalar</div>
            <label className="text-xs uppercase tracking-widest text-zinc-500">
              Signal URL
            </label>
            <input
              type="text"
              value={draftUrl}
              onChange={(e) => setDraftUrl(e.target.value)}
              className="input-base w-full mt-1.5 font-mono text-sm"
            />
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={() => setSettingsOpen(false)}
                className="px-3 py-1.5 text-sm text-zinc-400 hover:text-white"
              >
                Bekor
              </button>
              <button
                onClick={saveSettings}
                className="btn-primary rounded-btn px-4 py-1.5 text-sm font-medium"
              >
                Saqlash
              </button>
            </div>
          </motion.div>
        </motion.div>
      )}
    </div>
  );
}
